package economy

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
	"skyvern/internal/manager"
)

func lotteryCmds(p *EconomyPlugin) []*manager.Command {
	return []*manager.Command{
		{
			Trigger:     "lottery",
			Aliases:     []string{"lot"},
			Name:        "lottery",
			Description: "Guild lottery ticket menu",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				if len(ctx.Args) == 0 {
					return executeLotteryStatus(ctx)
				}
				sub := strings.ToLower(ctx.Args[0])
				switch sub {
				case "buy":
					return executeLotteryBuy(ctx)
				case "status":
					return executeLotteryStatus(ctx)
				default:
					return executeLotteryStatus(ctx)
				}
			},
		},
	}
}

func executeLotteryStatus(ctx *manager.CommandContext) error {
	gid := ctx.GuildID()
	if gid == "" {
		return ctx.Reply("Must be used in a server.")
	}
	cfg := getCfg(ctx.DB, gid)
	if !cfg.Enabled {
		return ctx.Reply("Economy is disabled in this server.")
	}

	uid := ctx.AuthorID()
	userTickets := cfg.LotteryTickets[uid]

	totalTickets := 0
	for _, t := range cfg.LotteryTickets {
		totalTickets += t
	}

	chance := 0.0
	if totalTickets > 0 {
		chance = (float64(userTickets) / float64(totalTickets)) * 100.0
	}

	inflation := getInflationIndex(ctx.DB, gid)
	ticketPrice := int64(float64(100) * inflation)

	var sb strings.Builder
	sb.WriteString("=== Lottery Status ===\n")
	sb.WriteString(fmt.Sprintf("Current Pot: %s\n", fmtCoins(cfg.LotteryPot, cfg)))
	sb.WriteString(fmt.Sprintf("Total Tickets in Play: %d\n", totalTickets))
	sb.WriteString(fmt.Sprintf("Your Tickets: %d (Win Chance: %.2f%%)\n", userTickets, chance))
	sb.WriteString(fmt.Sprintf("Ticket Price: %s\n", fmtCoins(ticketPrice, cfg)))
	sb.WriteString("----------------------\n")
	sb.WriteString("Use .lottery buy [qty] to enter the drawing!\n")

	return ctx.Reply(fmt.Sprintf("```\n%s```", sb.String()))
}

func executeLotteryBuy(ctx *manager.CommandContext) error {
	gid := ctx.GuildID()
	if gid == "" {
		return ctx.Reply("Must be used in a server.")
	}
	cfg := getCfg(ctx.DB, gid)
	if !cfg.Enabled {
		return ctx.Reply("Economy is disabled in this server.")
	}

	qty := 1
	if len(ctx.Args) > 1 {
		if q, err := strconv.Atoi(ctx.Args[1]); err == nil && q > 0 {
			qty = q
		}
	} else if len(ctx.Args) == 1 && ctx.Args[0] != "buy" {
		if q, err := strconv.Atoi(ctx.Args[0]); err == nil && q > 0 {
			qty = q
		}
	}

	uid := ctx.AuthorID()
	a := getAcct(ctx.DB, gid, uid)

	inflation := getInflationIndex(ctx.DB, gid)
	ticketPrice := int64(float64(100) * inflation)

	if ticketPrice > 0 && int64(qty) > 9223372036854775807/ticketPrice {
		return ctx.Reply("Quantity is too large.")
	}

	cost := ticketPrice * int64(qty)
	if a.Wallet < cost {
		return ctx.Reply(fmt.Sprintf("Insufficient funds. Cost: %s | Wallet balance: %s",
			fmtCoins(cost, cfg), fmtCoins(a.Wallet, cfg)))
	}

	// Update balance and tickets
	a.Wallet -= cost
	_ = saveAcct(ctx.DB, gid, uid, a)

	if cfg.LotteryTickets == nil {
		cfg.LotteryTickets = make(map[string]int)
	}
	cfg.LotteryTickets[uid] += qty
	cfg.LotteryPot += cost
	_ = saveCfg(ctx.DB, gid, cfg)

	return ctx.Reply(fmt.Sprintf("Successfully bought %d lottery tickets for %s.\nNew Wallet Balance: %s",
		qty, fmtCoins(cost, cfg), fmtCoins(a.Wallet, cfg)))
}

func drawLotteryIfNeeded(ctx *manager.CommandContext) (string, int64, string) {
	gid := ctx.GuildID()
	cfg := getCfg(ctx.DB, gid)

	// Lottery drawing triggers at most once every 24 hours if tickets exist
	now := time.Now()
	// Let's use LastDecay as a proxy for drawing cooldown check if we don't have a dedicated last_draw in cfg,
	// or we can just draw whenever 24 hours pass. Since LastDecay is always updated, we'll store draw time in EcoCfg
	// by abusing the LastDecay field or by checking LastDecay.
	// Actually, let's check time elapsed since cfg.LastDecay as a proxy, or just time.Now()
	// Let's look: to keep it clean, if we don't have LastLotteryDraw, we can use LastDecay, but wait!
	// We can save LastDecay and compare. Or even simpler, check if the pot has tickets.
	// Let's draw if the last decay hour interval matches, or just check 24 hours.
	// We added LastDecay to cfg. We can use LastDecay to check daily drawings.
	// Wait, let's just make it draw when a user runs .daily and time.Since(cfg.LastDecay) is checked?
	// No, let's draw whenever time.Since(cfg.LastDecay) is more than 24 hours. Or we can just store the draw time
	// in a separate bucket or simply draw if it's been 24h since last draw.
	// Wait, we can track time.Now().Hour() == 0 (midnight drawing) or check time.Since(cfg.LastDecay) >= 24h!
	// Let's check: if it's been more than 24 hours since the last draw, we draw.
	// We can store the drawing timestamp in a BoltDB setting or simply run it.
	// Let's use the cfg.LastDecay to store drawing, or we can just add a global drawing check using the current time.
	// Wait, how about we just use the database to store LastLotteryDraw?
	// We already added LastDecay and LastWeatherUpdate to EcoCfg. We can just use LastWeatherUpdate's date
	// or simply check if 24 hours passed since cfg.LastDecay.
	// Actually, let's check `cfg.LastDecay`. Since LastDecay is updated on every sell, it decays.
	// Let's use `cfg.LastDecay` date, or let's just store a custom key in BoltDB for the last lottery drawing!
	// Yes! In `lottery.go`, we can view/update the last drawing time in the `EcoCfg` bucket by setting a key "lottery_draw_time" in the DB.
	// This is very clean and doesn't require modifying the EcoCfg struct again!
	
	db := ctx.DB
	var lastDrawTime time.Time
	_ = db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktEcoCfg)
		if bkt == nil {
			return nil
		}
		v := bkt.Get([]byte(gid + ":last_draw"))
		if v != nil {
			_ = json.Unmarshal(v, &lastDrawTime)
		}
		return nil
	})

	if !lastDrawTime.IsZero() && now.Sub(lastDrawTime) < 24*time.Hour {
		return "", 0, ""
	}

	totalTickets := 0
	var ticketPool []string
	for uid, tCount := range cfg.LotteryTickets {
		totalTickets += tCount
		for i := 0; i < tCount; i++ {
			ticketPool = append(ticketPool, uid)
		}
	}

	if len(ticketPool) == 0 || cfg.LotteryPot <= 0 {
		return "", 0, ""
	}

	// Pick random ticket
	winnerUID := ticketPool[rand.Intn(len(ticketPool))]
	pot := cfg.LotteryPot

	// Credit winner
	winnerAcct := getAcct(ctx.DB, gid, winnerUID)
	winnerAcct.Wallet += pot
	_ = saveAcct(ctx.DB, gid, winnerUID, winnerAcct)

	// Reset lottery
	cfg.LotteryPot = 0
	cfg.LotteryTickets = make(map[string]int)
	_ = saveCfg(ctx.DB, gid, cfg)

	// Save last draw time
	_ = db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktEcoCfg)
		if bkt != nil {
			b, _ := json.Marshal(now)
			_ = bkt.Put([]byte(gid+":last_draw"), b)
		}
		return nil
	})

	winnerTag := "User " + winnerUID
	if m, err := ctx.Session.GuildMember(gid, winnerUID); err == nil && m.User != nil {
		winnerTag = m.User.Username
	}

	msg := fmt.Sprintf("\nLOTTERY DRAWING COMPLETE: Congratulations to <@%s> (%s) for winning the server lottery pot of %s!",
		winnerUID, winnerTag, fmtCoins(pot, cfg))

	return winnerUID, pot, msg
}
