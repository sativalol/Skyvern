package economy

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"skyvern/internal/config"
	"skyvern/internal/manager"
)

type Waifu struct {
	Name   string
	Rarity string
}

var waifuPool = map[string][]string{
	"Common": {
		"Sakura Haruno", "Aqua", "Bulma", "Nami", "Ochaco Uraraka",
		"Winry Rockbell", "Kagome Higurashi", "Orihime Inoue",
	},
	"Uncommon": {
		"Asuka Langley", "Rin Tohsaka", "Mikasa Ackerman", "Lucy Heartfilia",
		"Hinata Hyuga", "Rukia Kuchiki", "Faye Valentine", "Tsunade", "Hatsune Miku",
	},
	"Rare": {
		"Rem", "Zero Two", "Nezuko Kamado", "Megumin", "Kurisu Makise",
		"Raphtalia", "Mai Sakurajima", "Emilia",
	},
	"Epic": {
		"Saber", "Esdeath", "Makima", "Yor Forger", "Mitsuri Kanroji",
		"Shinobu Kocho", "Violet Evergarden",
	},
	"Legendary": {
		"Kaguya Shinomiya", "Raiden Shogun", "Hu Tao", "Artoria Pendragon",
		"Senjougahara Hitagi",
	},
}

func getWaifuRarity(name string) string {
	for rarity, names := range waifuPool {
		for _, n := range names {
			if strings.EqualFold(n, name) {
				return rarity
			}
		}
	}
	return ""
}

func getWaifuSellBasePrice(rarity string) int64 {
	switch rarity {
	case "Common":
		return 200
	case "Uncommon":
		return 400
	case "Rare":
		return 800
	case "Epic":
		return 1500
	case "Legendary":
		return 4000
	default:
		return 0
	}
}

func waifuCmds(p *EconomyPlugin) []*manager.Command {
	return []*manager.Command{
		{
			Trigger:     "wroll",
			Name:        "wroll",
			Description: "Roll a random waifu for your collection",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				return executeWaifuRoll(ctx)
			},
		},
		{
			Trigger:     "wlist",
			Name:        "wlist",
			Description: "View your collected waifus",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				return executeWaifuList(ctx)
			},
		},
		{
			Trigger:     "waifu",
			Aliases:     []string{"waifus", "claim"},
			Name:        "waifu",
			Description: "Waifu gacha and collection menu",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				if len(ctx.Args) == 0 {
					return ctx.SendHelp("waifu")
				}
				sub := strings.ToLower(ctx.Args[0])
				switch sub {
				case "roll":
					return executeWaifuRoll(ctx)
				case "list":
					return executeWaifuList(ctx)
				case "sell":
					return executeWaifuSell(ctx)
				default:
					return ctx.SendHelp("waifu")
				}
			},
		},
	}
}

func executeWaifuRoll(ctx *manager.CommandContext) error {
	gid := ctx.GuildID()
	if gid == "" {
		return ctx.Reply("Must be used in a server.")
	}
	cfg := getCfg(ctx.DB, gid)
	if !cfg.Enabled {
		return ctx.Reply("Economy is disabled in this server.")
	}

	uid := ctx.AuthorID()
	a := getAcct(ctx.DB, gid, uid)

	inflation := getInflationIndex(ctx.DB, gid)
	cost := int64(float64(500) * inflation)

	if a.Wallet < cost {
		return ctx.Reply(fmt.Sprintf("Insufficient funds. A waifu roll costs %s | Wallet balance: %s",
			fmtCoins(cost, cfg), fmtCoins(a.Wallet, cfg)))
	}

	// Deduct roll cost
	a.Wallet -= cost

	// Draw rarity
	r := rand.Float32()
	var rarity string
	if r < 0.40 {
		rarity = "Common"
	} else if r < 0.70 {
		rarity = "Uncommon"
	} else if r < 0.88 {
		rarity = "Rare"
	} else if r < 0.97 {
		rarity = "Epic"
	} else {
		rarity = "Legendary"
	}

	// Pick random character
	list := waifuPool[rarity]
	name := list[rand.Intn(len(list))]

	if a.Waifus == nil {
		a.Waifus = make(map[string]int)
	}
	a.Waifus[name]++

	_ = saveAcct(ctx.DB, gid, uid, a)

	emb := config.Build(ctx.Cfg, config.EmbedOpt{
		Title:       "Waifu Gacha Roll Result",
		Description: fmt.Sprintf("You rolled: **%s** (%s rarity)!\n\nRoll Cost: %s\nRemaining Wallet: %s\nTotal Owned: %dx",
			name, rarity, fmtCoins(cost, cfg), fmtCoins(a.Wallet, cfg), a.Waifus[name]),
	})
	return ctx.Respond(emb)
}

func executeWaifuList(ctx *manager.CommandContext) error {
	gid := ctx.GuildID()
	if gid == "" {
		return ctx.Reply("Must be used in a server.")
	}
	cfg := getCfg(ctx.DB, gid)
	if !cfg.Enabled {
		return ctx.Reply("Economy is disabled in this server.")
	}

	targetID := ctx.AuthorID()
	targetTag := ctx.AuthorTag()
	if len(ctx.Args) > 1 && strings.EqualFold(ctx.Args[0], "list") {
		if resolved := resolveUser(ctx.Args[1]); resolved != "" {
			targetID = resolved
			if m, err := ctx.Session.GuildMember(gid, targetID); err == nil && m.User != nil {
				targetTag = m.User.Username
			} else if u, err := ctx.Session.User(targetID); err == nil {
				targetTag = u.Username
			}
		}
	} else if len(ctx.Args) > 0 && !strings.EqualFold(ctx.Args[0], "list") && !strings.EqualFold(ctx.Args[0], "roll") && !strings.EqualFold(ctx.Args[0], "sell") {
		if resolved := resolveUser(ctx.Args[0]); resolved != "" {
			targetID = resolved
			if m, err := ctx.Session.GuildMember(gid, targetID); err == nil && m.User != nil {
				targetTag = m.User.Username
			} else if u, err := ctx.Session.User(targetID); err == nil {
				targetTag = u.Username
			}
		}
	}

	a := getAcct(ctx.DB, gid, targetID)
	if len(a.Waifus) == 0 {
		return ctx.Reply(fmt.Sprintf("%s's waifu collection is empty.", targetTag))
	}

	rarities := []string{"Legendary", "Epic", "Rare", "Uncommon", "Common"}
	var lines []string

	for _, rarity := range rarities {
		var list []string
		for name, count := range a.Waifus {
			if getWaifuRarity(name) == rarity {
				list = append(list, fmt.Sprintf("%s (x%d)", name, count))
			}
		}
		if len(list) > 0 {
			lines = append(lines, fmt.Sprintf("__**%s Rarity**__\n%s", rarity, strings.Join(list, ", ")))
		}
	}

	emb := config.Build(ctx.Cfg, config.EmbedOpt{
		Title:       fmt.Sprintf("%s's Waifu Collection", targetTag),
		Description: strings.Join(lines, "\n\n"),
	})
	return ctx.Respond(emb)
}

func executeWaifuSell(ctx *manager.CommandContext) error {
	gid := ctx.GuildID()
	if gid == "" {
		return ctx.Reply("Must be used in a server.")
	}
	cfg := getCfg(ctx.DB, gid)
	if !cfg.Enabled {
		return ctx.Reply("Economy is disabled in this server.")
	}

	if len(ctx.Args) < 2 {
		return ctx.Reply("Usage: .waifu sell <character name>")
	}

	name := strings.Join(ctx.Args[1:], " ")
	uid := ctx.AuthorID()
	a := getAcct(ctx.DB, gid, uid)

	actualName := ""
	for wName := range a.Waifus {
		if strings.EqualFold(wName, name) {
			actualName = wName
			break
		}
	}

	if actualName == "" || a.Waifus[actualName] <= 0 {
		return ctx.Reply(fmt.Sprintf("You do not own any copies of %q.", name))
	}

	rarity := getWaifuRarity(actualName)
	basePrice := getWaifuSellBasePrice(rarity)
	inflation := getInflationIndex(ctx.DB, gid)
	payout := int64(float64(basePrice) * math.Sqrt(inflation))

	a.Waifus[actualName]--
	if a.Waifus[actualName] <= 0 {
		delete(a.Waifus, actualName)
	}

	a.Wallet += payout
	_ = saveAcct(ctx.DB, gid, uid, a)

	return ctx.Reply(fmt.Sprintf("Successfully sold 1x copy of %s (%s) for %s.\nNew Wallet Balance: %s",
		actualName, rarity, fmtCoins(payout, cfg), fmtCoins(a.Wallet, cfg)))
}
