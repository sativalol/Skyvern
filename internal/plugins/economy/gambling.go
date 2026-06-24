package economy

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
	"github.com/bwmarrin/discordgo"
	"skyvern/internal/config"
	"skyvern/internal/manager"
)

type BJGame struct {
	GuildID    string
	UserID     string
	Bet        int64
	PlayerHand []Card
	DealerHand []Card
	Deck       []Card
	LastAction time.Time
}

type Card struct {
	Suit  string
	Value string
	Score int
}

type HLGame struct {
	GuildID    string
	UserID     string
	Bet        int64
	Number     int
	LastAction time.Time
}

var (
	activeBJs = make(map[string]*BJGame)
	bjMu      sync.Mutex

	activeHLs = make(map[string]*HLGame)
	hlMu      sync.Mutex
)

func init() {
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			bjMu.Lock()
			for k, g := range activeBJs {
				if time.Since(g.LastAction) > 5*time.Minute {
					delete(activeBJs, k)
				}
			}
			bjMu.Unlock()

			hlMu.Lock()
			for k, g := range activeHLs {
				if time.Since(g.LastAction) > 5*time.Minute {
					delete(activeHLs, k)
				}
			}
			hlMu.Unlock()
		}
	}()
}

func createDeck() []Card {
	suits := []string{"Hearts", "Diamonds", "Clubs", "Spades"}
	values := []string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}
	var deck []Card
	for _, s := range suits {
		for _, v := range values {
			score := 0
			switch v {
			case "J", "Q", "K":
				score = 10
			case "A":
				score = 11
			default:
				score, _ = strconv.Atoi(v)
			}
			deck = append(deck, Card{Suit: s, Value: v, Score: score})
		}
	}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}

func getHandScore(hand []Card) int {
	score := 0
	aces := 0
	for _, c := range hand {
		score += c.Score
		if c.Value == "A" {
			aces++
		}
	}
	for score > 21 && aces > 0 {
		score -= 10
		aces--
	}
	return score
}

func formatHand(hand []Card) string {
	var sb strings.Builder
	for _, c := range hand {
		sb.WriteString(fmt.Sprintf("[%s of %s] ", c.Value, c.Suit))
	}
	return sb.String()
}

func getResCfg(mgr *manager.Manager, s *discordgo.Session) config.ResCfg {
	if s.State != nil && s.State.User != nil {
		if rc, ok := mgr.ResolvedCfgFor(s.State.User.ID); ok {
			return rc
		}
	}
	return config.ResCfg{}
}

func gamblingCmds(p *EconomyPlugin) []*manager.Command {
	return []*manager.Command{
		{
			Trigger:     "slots",
			Aliases:     []string{"slot"},
			Name:        "slots",
			Description: "Spin the slot machine",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				gid := ctx.GuildID()
				if gid == "" {
					return ctx.Reply("Must be used in a server.")
				}
				cfg := getCfg(ctx.DB, gid)
				if !cfg.Enabled {
					return ctx.Reply("Economy is disabled in this server.")
				}
				if len(ctx.Args) == 0 {
					return ctx.Reply("Usage: .slots <bet>")
				}

				uid := ctx.AuthorID()
				a := getAcct(ctx.DB, gid, uid)

				bet, err := parseAmount(ctx.Args[0], a.Wallet)
				if err != nil {
					return ctx.Reply("Invalid bet amount.")
				}
				if bet > a.Wallet {
					return ctx.Reply("You don't have enough coins in your wallet.")
				}

				a.Wallet -= bet
				_ = saveAcct(ctx.DB, gid, uid, a)

				emojis := []string{"Cherry", "Lemon", "Grape", "Bell", "Diamond"}
				r1 := emojis[rand.Intn(len(emojis))]
				r2 := emojis[rand.Intn(len(emojis))]
				r3 := emojis[rand.Intn(len(emojis))]

				payout := int64(0)
				msg := ""

				if r1 == r2 && r2 == r3 {
					multiplier := int64(4)
					if r1 == "Diamond" {
						multiplier = 10
					} else if r1 == "Bell" {
						multiplier = 6
					}
					payout = bet * multiplier
					msg = fmt.Sprintf("JACKPOT! You got three %s! You won %s!", r1, fmtCoins(payout, cfg))
				} else if r1 == r2 || r2 == r3 || r1 == r3 {
					payout = int64(float64(bet) * 1.5)
					matched := r2
					if r1 == r3 {
						matched = r1
					}
					msg = fmt.Sprintf("You got two %s! You won %s!", matched, fmtCoins(payout, cfg))
				} else {
					msg = fmt.Sprintf("You spun and got nothing. You lost %s.", fmtCoins(bet, cfg))
				}

				a = getAcct(ctx.DB, gid, uid)
				a.Wallet += payout
				_ = saveAcct(ctx.DB, gid, uid, a)

				emb := config.Build(ctx.Cfg, config.EmbedOpt{
					Title:       "Slot Machine",
					Description: fmt.Sprintf("```\n[ %s | %s | %s ]\n```\n%s", r1, r2, r3, msg),
				})
				return ctx.Respond(emb)
			},
		},
		{
			Trigger:     "coinflipbet",
			Aliases:     []string{"cf"},
			Name:        "coinflipbet",
			Description: "Bet on heads or tails",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				gid := ctx.GuildID()
				if gid == "" {
					return ctx.Reply("Must be used in a server.")
				}
				cfg := getCfg(ctx.DB, gid)
				if !cfg.Enabled {
					return ctx.Reply("Economy is disabled in this server.")
				}
				if len(ctx.Args) < 2 {
					return ctx.Reply("Usage: .cf <heads/tails> <bet>")
				}

				guess := strings.ToLower(ctx.Args[0])
				betStr := ctx.Args[1]
				if guess != "heads" && guess != "tails" && guess != "h" && guess != "t" {
					guess = strings.ToLower(ctx.Args[1])
					betStr = ctx.Args[0]
				}

				if guess != "heads" && guess != "tails" && guess != "h" && guess != "t" {
					return ctx.Reply("Guess must be heads or tails.")
				}

				uid := ctx.AuthorID()
				a := getAcct(ctx.DB, gid, uid)

				bet, err := parseAmount(betStr, a.Wallet)
				if err != nil {
					return ctx.Reply("Invalid bet amount.")
				}
				if bet > a.Wallet {
					return ctx.Reply("You don't have enough coins in your wallet.")
				}

				a.Wallet -= bet
				_ = saveAcct(ctx.DB, gid, uid, a)

				result := "heads"
				if rand.Float32() < 0.5 {
					result = "tails"
				}

				isWin := false
				if (guess == "heads" || guess == "h") && result == "heads" {
					isWin = true
				} else if (guess == "tails" || guess == "t") && result == "tails" {
					isWin = true
				}

				payout := int64(0)
				responseMsg := ""
				if isWin {
					payout = bet * 2
					a = getAcct(ctx.DB, gid, uid)
					a.Wallet += payout
					_ = saveAcct(ctx.DB, gid, uid, a)
					responseMsg = fmt.Sprintf("The coin landed on %s! You won %s!", result, fmtCoins(bet, cfg))
				} else {
					responseMsg = fmt.Sprintf("The coin landed on %s! You lost %s.", result, fmtCoins(bet, cfg))
				}

				return ctx.Reply(responseMsg)
			},
		},
		{
			Trigger:     "blackjack",
			Aliases:     []string{"bj"},
			Name:        "blackjack",
			Description: "Play a game of blackjack against the dealer",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				gid := ctx.GuildID()
				if gid == "" {
					return ctx.Reply("Must be used in a server.")
				}
				cfg := getCfg(ctx.DB, gid)
				if !cfg.Enabled {
					return ctx.Reply("Economy is disabled in this server.")
				}
				if len(ctx.Args) == 0 {
					return ctx.Reply("Usage: .blackjack <bet>")
				}

				uid := ctx.AuthorID()
				key := gid + ":" + uid

				bjMu.Lock()
				if _, ok := activeBJs[key]; ok {
					bjMu.Unlock()
					return ctx.Reply("You already have an active blackjack game.")
				}
				bjMu.Unlock()

				a := getAcct(ctx.DB, gid, uid)
				bet, err := parseAmount(ctx.Args[0], a.Wallet)
				if err != nil {
					return ctx.Reply("Invalid bet amount.")
				}
				if bet > a.Wallet {
					return ctx.Reply("You don't have enough coins in your wallet.")
				}

				a.Wallet -= bet
				_ = saveAcct(ctx.DB, gid, uid, a)

				deck := createDeck()
				pHand := []Card{deck[0], deck[2]}
				dHand := []Card{deck[1], deck[3]}
				deck = deck[4:]

				pScore := getHandScore(pHand)
				dScore := getHandScore(dHand)

				if pScore == 21 {
					winnings := int64(float64(bet) * 2.5)
					a = getAcct(ctx.DB, gid, uid)
					a.Wallet += winnings
					_ = saveAcct(ctx.DB, gid, uid, a)

					emb := config.Build(ctx.Cfg, config.EmbedOpt{
						Title: "Blackjack — Game Over (Natural Blackjack!)",
						Description: fmt.Sprintf("**Your Hand:** %s (Score: %d)\n**Dealer's Hand:** %s (Score: %d)\n\nYou got a Natural Blackjack and won %s!",
							formatHand(pHand), pScore, formatHand(dHand), dScore, fmtCoins(winnings-bet, cfg)),
					})
					return ctx.Respond(emb)
				}

				g := &BJGame{
					GuildID:    gid,
					UserID:     uid,
					Bet:        bet,
					PlayerHand: pHand,
					DealerHand: dHand,
					Deck:       deck,
					LastAction: time.Now(),
				}

				bjMu.Lock()
				activeBJs[key] = g
				bjMu.Unlock()

				walletAmt := getAcct(ctx.DB, gid, uid).Wallet

				emb := config.Build(ctx.Cfg, config.EmbedOpt{
					Title: "Blackjack",
					Description: fmt.Sprintf("**Your Hand:** %s (Score: %d)\n**Dealer's Hand:** [%s of %s] [Hidden]\n\nDo you want to Hit, Stand, or Double Down?",
						formatHand(pHand), pScore, dHand[0].Value, dHand[0].Suit),
				})

				components := []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.Button{Label: "Hit", Style: discordgo.PrimaryButton, CustomID: fmt.Sprintf("bj:hit:%s:%s", gid, uid)},
							discordgo.Button{Label: "Stand", Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("bj:stand:%s:%s", gid, uid)},
							discordgo.Button{Label: "Double Down", Style: discordgo.SuccessButton, CustomID: fmt.Sprintf("bj:double:%s:%s", gid, uid), Disabled: walletAmt < bet},
						},
					},
				}

				if ctx.Interact != nil {
					return ctx.Session.InteractionRespond(ctx.Interact, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{emb}, Components: components},
					})
				}
				_, err = ctx.Session.ChannelMessageSendComplex(ctx.ChanID(), &discordgo.MessageSend{
					Embeds:     []*discordgo.MessageEmbed{emb},
					Components: components,
				})
				return err
			},
		},
		{
			Trigger:     "dice",
			Aliases:     []string{"roll"},
			Name:        "dice",
			Description: "Bet on a 6-sided dice roll",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				gid := ctx.GuildID()
				if gid == "" {
					return ctx.Reply("Must be used in a server.")
				}
				cfg := getCfg(ctx.DB, gid)
				if !cfg.Enabled {
					return ctx.Reply("Economy is disabled in this server.")
				}
				if len(ctx.Args) < 2 {
					return ctx.Reply("Usage: .dice <guess: 1-6> <bet>")
				}

				guess, err1 := strconv.Atoi(ctx.Args[0])
				betStr := ctx.Args[1]
				if err1 != nil || guess < 1 || guess > 6 {
					guess, err1 = strconv.Atoi(ctx.Args[1])
					betStr = ctx.Args[0]
				}

				if err1 != nil || guess < 1 || guess > 6 {
					return ctx.Reply("Dice guess must be a number between 1 and 6.")
				}

				uid := ctx.AuthorID()
				a := getAcct(ctx.DB, gid, uid)

				bet, err := parseAmount(betStr, a.Wallet)
				if err != nil {
					return ctx.Reply("Invalid bet amount.")
				}
				if bet > a.Wallet {
					return ctx.Reply("You don't have enough coins in your wallet.")
				}

				a.Wallet -= bet
				_ = saveAcct(ctx.DB, gid, uid, a)

				roll := rand.Intn(6) + 1
				if roll == guess {
					payout := bet * 5
					a = getAcct(ctx.DB, gid, uid)
					a.Wallet += payout
					_ = saveAcct(ctx.DB, gid, uid, a)
					return ctx.Reply(fmt.Sprintf("The die landed on %d! You guessed correctly and won %s!", roll, fmtCoins(payout-bet, cfg)))
				}

				return ctx.Reply(fmt.Sprintf("The die landed on %d! You lost your bet of %s.", roll, fmtCoins(bet, cfg)))
			},
		},
		{
			Trigger:     "roulette",
			Aliases:     []string{"rl"},
			Name:        "roulette",
			Description: "Bet on red, black, green, or a specific number in roulette",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				gid := ctx.GuildID()
				if gid == "" {
					return ctx.Reply("Must be used in a server.")
				}
				cfg := getCfg(ctx.DB, gid)
				if !cfg.Enabled {
					return ctx.Reply("Economy is disabled in this server.")
				}
				if len(ctx.Args) < 2 {
					return ctx.Reply("Usage: .roulette <bet> <red/black/green/0-36>")
				}

				uid := ctx.AuthorID()
				a := getAcct(ctx.DB, gid, uid)

				bet, err := parseAmount(ctx.Args[0], a.Wallet)
				if err != nil {
					return ctx.Reply("Invalid bet amount.")
				}
				if bet > a.Wallet {
					return ctx.Reply("You don't have enough coins in your wallet.")
				}

				choice := strings.ToLower(ctx.Args[1])

				a.Wallet -= bet
				_ = saveAcct(ctx.DB, gid, uid, a)

				roll := rand.Intn(37)
				color := "black"
				reds := map[int]bool{
					1: true, 3: true, 5: true, 7: true, 9: true, 12: true, 14: true, 16: true, 18: true,
					19: true, 21: true, 23: true, 25: true, 27: true, 30: true, 32: true, 34: true, 36: true,
				}

				if roll == 0 {
					color = "green"
				} else if reds[roll] {
					color = "red"
				}

				isWin := false
				payoutMultiplier := int64(0)

				if choice == "red" && color == "red" {
					isWin = true
					payoutMultiplier = 2
				} else if choice == "black" && color == "black" {
					isWin = true
					payoutMultiplier = 2
				} else if choice == "green" && color == "green" {
					isWin = true
					payoutMultiplier = 35
				} else if num, err := strconv.Atoi(choice); err == nil && num == roll {
					isWin = true
					payoutMultiplier = 35
				}

				payout := int64(0)
				resMsg := ""
				if isWin {
					payout = bet * payoutMultiplier
					a = getAcct(ctx.DB, gid, uid)
					a.Wallet += payout
					_ = saveAcct(ctx.DB, gid, uid, a)
					resMsg = fmt.Sprintf("The roulette landed on %d (%s)! You won %s!", roll, strings.ToUpper(color), fmtCoins(payout-bet, cfg))
				} else {
					resMsg = fmt.Sprintf("The roulette landed on %d (%s)! You lost %s.", roll, strings.ToUpper(color), fmtCoins(bet, cfg))
				}

				return ctx.Reply(resMsg)
			},
		},
		{
			Trigger:     "highlow",
			Aliases:     []string{"hl"},
			Name:        "highlow",
			Description: "Guess if the next number will be higher or lower",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				gid := ctx.GuildID()
				if gid == "" {
					return ctx.Reply("Must be used in a server.")
				}
				cfg := getCfg(ctx.DB, gid)
				if !cfg.Enabled {
					return ctx.Reply("Economy is disabled in this server.")
				}
				if len(ctx.Args) == 0 {
					return ctx.Reply("Usage: .highlow <bet>")
				}

				uid := ctx.AuthorID()
				key := gid + ":" + uid

				hlMu.Lock()
				if _, ok := activeHLs[key]; ok {
					hlMu.Unlock()
					return ctx.Reply("You already have an active highlow game.")
				}
				hlMu.Unlock()

				a := getAcct(ctx.DB, gid, uid)
				bet, err := parseAmount(ctx.Args[0], a.Wallet)
				if err != nil {
					return ctx.Reply("Invalid bet amount.")
				}
				if bet > a.Wallet {
					return ctx.Reply("You don't have enough coins in your wallet.")
				}

				a.Wallet -= bet
				_ = saveAcct(ctx.DB, gid, uid, a)

				num := rand.Intn(100) + 1

				g := &HLGame{
					GuildID:    gid,
					UserID:     uid,
					Bet:        bet,
					Number:     num,
					LastAction: time.Now(),
				}

				hlMu.Lock()
				activeHLs[key] = g
				hlMu.Unlock()

				emb := config.Build(ctx.Cfg, config.EmbedOpt{
					Title:       "HighLow",
					Description: fmt.Sprintf("The current number is **%d**.\n\nGuess if the next number (1-100) will be **Higher**, **Lower**, or the **Jackpot** (exactly equal)?", num),
				})

				components := []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.Button{Label: "Higher", Style: discordgo.PrimaryButton, CustomID: fmt.Sprintf("hl:higher:%s:%s", gid, uid)},
							discordgo.Button{Label: "Lower", Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("hl:lower:%s:%s", gid, uid)},
							discordgo.Button{Label: "Jackpot (Same)", Style: discordgo.SuccessButton, CustomID: fmt.Sprintf("hl:jackpot:%s:%s", gid, uid)},
						},
					},
				}

				if ctx.Interact != nil {
					return ctx.Session.InteractionRespond(ctx.Interact, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{emb}, Components: components},
					})
				}
				_, err = ctx.Session.ChannelMessageSendComplex(ctx.ChanID(), &discordgo.MessageSend{
					Embeds:     []*discordgo.MessageEmbed{emb},
					Components: components,
				})
				return err
			},
		},
		{
			Trigger:     "scratch",
			Name:        "scratch",
			Description: "Bet on a scratch card",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				gid := ctx.GuildID()
				if gid == "" {
					return ctx.Reply("Must be used in a server.")
				}
				cfg := getCfg(ctx.DB, gid)
				if !cfg.Enabled {
					return ctx.Reply("Economy is disabled in this server.")
				}
				if len(ctx.Args) < 1 {
					return ctx.Reply("Usage: .scratch <bet>")
				}

				uid := ctx.AuthorID()
				a := getAcct(ctx.DB, gid, uid)

				bet, err := parseAmount(ctx.Args[0], a.Wallet)
				if err != nil {
					return ctx.Reply("Invalid bet amount.")
				}
				if bet > a.Wallet {
					return ctx.Reply("You don't have enough coins in your wallet.")
				}

				a.Wallet -= bet
				_ = saveAcct(ctx.DB, gid, uid, a)

				symbols := []string{"COIN", "GEM", "BAR", "BOOT"}
				grid := make([]string, 9)
				for i := range grid {
					grid[i] = symbols[rand.Intn(len(symbols))]
				}

				var matches []string
				checkLine := func(i, j, k int) {
					if grid[i] == grid[j] && grid[j] == grid[k] {
						matches = append(matches, grid[i])
					}
				}

				checkLine(0, 1, 2)
				checkLine(3, 4, 5)
				checkLine(6, 7, 8)
				checkLine(0, 3, 6)
				checkLine(1, 4, 7)
				checkLine(2, 5, 8)
				checkLine(0, 4, 8)
				checkLine(2, 4, 6)

				payout := int64(0)
				for _, sym := range matches {
					switch sym {
					case "COIN":
						payout += bet * 2
					case "GEM":
						payout += bet * 4
					case "BAR":
						payout += bet * 6
					case "BOOT":
						payout += int64(float64(bet) * 0.5)
					}
				}

				winMsg := ""
				if len(matches) > 0 {
					a = getAcct(ctx.DB, gid, uid)
					a.Wallet += payout
					_ = saveAcct(ctx.DB, gid, uid, a)
					winMsg = fmt.Sprintf("Congratulations! You matched %d line(s) and won %s!", len(matches), fmtCoins(payout, cfg))
				} else {
					winMsg = fmt.Sprintf("No matches this time! You lost your bet of %s.", fmtCoins(bet, cfg))
				}

				var sb strings.Builder
				sb.WriteString("=== Scratch Card ===\n")
				sb.WriteString(fmt.Sprintf("[%s] [%s] [%s]\n", grid[0], grid[1], grid[2]))
				sb.WriteString(fmt.Sprintf("[%s] [%s] [%s]\n", grid[3], grid[4], grid[5]))
				sb.WriteString(fmt.Sprintf("[%s] [%s] [%s]\n", grid[6], grid[7], grid[8]))
				sb.WriteString("====================\n")
				sb.WriteString(winMsg)

				return ctx.Reply(fmt.Sprintf("```\n%s```", sb.String()))
			},
		},
		{
			Trigger:     "race",
			Aliases:     []string{"horserace", "hr"},
			Name:        "race",
			Description: "Bet on a horse race",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				gid := ctx.GuildID()
				if gid == "" {
					return ctx.Reply("Must be used in a server.")
				}
				cfg := getCfg(ctx.DB, gid)
				if !cfg.Enabled {
					return ctx.Reply("Economy is disabled in this server.")
				}
				if len(ctx.Args) < 2 {
					return ctx.Reply("Usage: .race <horse: 1-5> <bet>")
				}

				horse, err := strconv.Atoi(ctx.Args[0])
				if err != nil || horse < 1 || horse > 5 {
					return ctx.Reply("Please choose a horse number between 1 and 5.")
				}

				uid := ctx.AuthorID()
				a := getAcct(ctx.DB, gid, uid)

				bet, err := parseAmount(ctx.Args[1], a.Wallet)
				if err != nil {
					return ctx.Reply("Invalid bet amount.")
				}
				if bet > a.Wallet {
					return ctx.Reply("You don't have enough coins in your wallet.")
				}

				a.Wallet -= bet
				_ = saveAcct(ctx.DB, gid, uid, a)

				positions := make([]int, 5)
				trackLen := 15

				renderTrack := func(pos []int, status string) string {
					var s strings.Builder
					s.WriteString("=== Horse Race ===\n")
					for i := range pos {
						lane := make([]byte, trackLen)
						for j := range lane {
							lane[j] = '-'
						}
						hp := pos[i]
						if hp >= trackLen {
							hp = trackLen - 1
						}
						lane[hp] = byte('1' + i)
						s.WriteString(fmt.Sprintf("Horse %d: [%s]\n", i+1, string(lane)))
					}
					s.WriteString("==================\n")
					s.WriteString(status)
					return fmt.Sprintf("```\n%s```", s.String())
				}

				msg, err := ctx.Session.ChannelMessageSend(ctx.ChanID(), renderTrack(positions, "And they're off!"))
				if err != nil {
					return err
				}

				for step := 0; step < 4; step++ {
					time.Sleep(1 * time.Second)
					for i := range positions {
						positions[i] += rand.Intn(3)
					}
					_, _ = ctx.Session.ChannelMessageEdit(ctx.ChanID(), msg.ID, renderTrack(positions, "Racing..."))
				}

				maxPos := -1
				var winners []int
				for i, pos := range positions {
					if pos > maxPos {
						maxPos = pos
						winners = []int{i + 1}
					} else if pos == maxPos {
						winners = append(winners, i+1)
					}
				}

				winningHorse := winners[rand.Intn(len(winners))]
				winStatus := ""

				if horse == winningHorse {
					payout := bet * 4
					a = getAcct(ctx.DB, gid, uid)
					a.Wallet += payout
					_ = saveAcct(ctx.DB, gid, uid, a)
					winStatus = fmt.Sprintf("Horse %d won! You won %s!", winningHorse, fmtCoins(payout, cfg))
				} else {
					winStatus = fmt.Sprintf("Horse %d won! You lost your bet of %s.", winningHorse, fmtCoins(bet, cfg))
				}

				time.Sleep(1 * time.Second)
				_, err = ctx.Session.ChannelMessageEdit(ctx.ChanID(), msg.ID, renderTrack(positions, winStatus))
				return err
			},
		},
	}
}

func (p *EconomyPlugin) handleBJInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, ":")
	if len(parts) < 4 {
		return
	}
	action := parts[1]
	gid := parts[2]
	uid := parts[3]

	clickerID := i.User.ID
	if clickerID == "" && i.Member != nil && i.Member.User != nil {
		clickerID = i.Member.User.ID
	}

	if clickerID != uid {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "This is not your game.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	key := gid + ":" + uid
	bjMu.Lock()
	g, ok := activeBJs[key]
	if !ok {
		bjMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Content: "Game not found or expired.", Embeds: nil, Components: nil},
		})
		return
	}
	g.LastAction = time.Now()
	bjMu.Unlock()

	cfg := getCfg(p.db, gid)
	rcfg := getResCfg(p.mgr, s)

	switch action {
	case "hit":
		g.PlayerHand = append(g.PlayerHand, g.Deck[0])
		g.Deck = g.Deck[1:]
		pScore := getHandScore(g.PlayerHand)

		if pScore > 21 {
			bjMu.Lock()
			delete(activeBJs, key)
			bjMu.Unlock()

			emb := config.Build(rcfg, config.EmbedOpt{
				Title: "Blackjack — Game Over (Bust!)",
				Description: fmt.Sprintf("**Your Hand:** %s (Score: %d)\n**Dealer's Hand:** %s (Score: %d)\n\nYou busted and lost %s.",
					formatHand(g.PlayerHand), pScore, formatHand(g.DealerHand), getHandScore(g.DealerHand), fmtCoins(g.Bet, cfg)),
			})
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{emb}, Components: nil},
			})
			return
		}

		emb := config.Build(rcfg, config.EmbedOpt{
			Title: "Blackjack",
			Description: fmt.Sprintf("**Your Hand:** %s (Score: %d)\n**Dealer's Hand:** [%s of %s] [Hidden]\n\nDo you want to Hit or Stand?",
				formatHand(g.PlayerHand), pScore, g.DealerHand[0].Value, g.DealerHand[0].Suit),
		})
		walletAmt := getAcct(p.db, gid, uid).Wallet
		components := []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{Label: "Hit", Style: discordgo.PrimaryButton, CustomID: fmt.Sprintf("bj:hit:%s:%s", gid, uid)},
					discordgo.Button{Label: "Stand", Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("bj:stand:%s:%s", gid, uid)},
					discordgo.Button{Label: "Double Down", Style: discordgo.SuccessButton, CustomID: fmt.Sprintf("bj:double:%s:%s", gid, uid), Disabled: walletAmt < g.Bet},
				},
			},
		}
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{emb}, Components: components},
		})

	case "stand":
		pScore := getHandScore(g.PlayerHand)
		dScore := getHandScore(g.DealerHand)

		for dScore < 17 {
			g.DealerHand = append(g.DealerHand, g.Deck[0])
			g.Deck = g.Deck[1:]
			dScore = getHandScore(g.DealerHand)
		}

		bjMu.Lock()
		delete(activeBJs, key)
		bjMu.Unlock()

		payout := int64(0)
		resText := ""

		if dScore > 21 {
			payout = g.Bet * 2
			resText = fmt.Sprintf("Dealer busted! You won %s!", fmtCoins(g.Bet, cfg))
		} else if pScore > dScore {
			payout = g.Bet * 2
			resText = fmt.Sprintf("You beat the dealer! You won %s!", fmtCoins(g.Bet, cfg))
		} else if pScore < dScore {
			resText = fmt.Sprintf("Dealer won. You lost %s.", fmtCoins(g.Bet, cfg))
		} else {
			payout = g.Bet
			resText = "Push! Your bet was refunded."
		}

		if payout > 0 {
			a := getAcct(p.db, gid, uid)
			a.Wallet += payout
			_ = saveAcct(p.db, gid, uid, a)
		}

		emb := config.Build(rcfg, config.EmbedOpt{
			Title: "Blackjack — Game Over",
			Description: fmt.Sprintf("**Your Hand:** %s (Score: %d)\n**Dealer's Hand:** %s (Score: %d)\n\n%s",
				formatHand(g.PlayerHand), pScore, formatHand(g.DealerHand), dScore, resText),
		})
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{emb}, Components: nil},
		})

	case "double":
		a := getAcct(p.db, gid, uid)
		if a.Wallet < g.Bet {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "You do not have enough coins to double down.", Flags: discordgo.MessageFlagsEphemeral},
			})
			return
		}

		a.Wallet -= g.Bet
		_ = saveAcct(p.db, gid, uid, a)

		g.Bet *= 2
		g.PlayerHand = append(g.PlayerHand, g.Deck[0])
		g.Deck = g.Deck[1:]
		pScore := getHandScore(g.PlayerHand)

		bjMu.Lock()
		delete(activeBJs, key)
		bjMu.Unlock()

		if pScore > 21 {
			emb := config.Build(rcfg, config.EmbedOpt{
				Title: "Blackjack — Game Over (Bust!)",
				Description: fmt.Sprintf("**Your Hand:** %s (Score: %d)\n**Dealer's Hand:** %s (Score: %d)\n\nYou busted on double down and lost %s.",
					formatHand(g.PlayerHand), pScore, formatHand(g.DealerHand), getHandScore(g.DealerHand), fmtCoins(g.Bet, cfg)),
			})
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{emb}, Components: nil},
			})
			return
		}

		dScore := getHandScore(g.DealerHand)
		for dScore < 17 {
			g.DealerHand = append(g.DealerHand, g.Deck[0])
			g.Deck = g.Deck[1:]
			dScore = getHandScore(g.DealerHand)
		}

		payout := int64(0)
		resText := ""

		if dScore > 21 {
			payout = g.Bet * 2
			resText = fmt.Sprintf("Dealer busted! You won %s!", fmtCoins(g.Bet/2, cfg))
		} else if pScore > dScore {
			payout = g.Bet * 2
			resText = fmt.Sprintf("You beat the dealer! You won %s!", fmtCoins(g.Bet/2, cfg))
		} else if pScore < dScore {
			resText = fmt.Sprintf("Dealer won. You lost %s.", fmtCoins(g.Bet, cfg))
		} else {
			payout = g.Bet
			resText = "Push! Your bet was refunded."
		}

		if payout > 0 {
			a = getAcct(p.db, gid, uid)
			a.Wallet += payout
			_ = saveAcct(p.db, gid, uid, a)
		}

		emb := config.Build(rcfg, config.EmbedOpt{
			Title: "Blackjack — Game Over (Double Down)",
			Description: fmt.Sprintf("**Your Hand:** %s (Score: %d)\n**Dealer's Hand:** %s (Score: %d)\n\n%s",
				formatHand(g.PlayerHand), pScore, formatHand(g.DealerHand), dScore, resText),
		})
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{emb}, Components: nil},
		})
	}
}

func (p *EconomyPlugin) handleHLInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, ":")
	if len(parts) < 4 {
		return
	}
	action := parts[1]
	gid := parts[2]
	uid := parts[3]

	clickerID := i.User.ID
	if clickerID == "" && i.Member != nil && i.Member.User != nil {
		clickerID = i.Member.User.ID
	}

	if clickerID != uid {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "This is not your game.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	key := gid + ":" + uid
	hlMu.Lock()
	g, ok := activeHLs[key]
	if !ok {
		hlMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Content: "Game not found or expired.", Embeds: nil, Components: nil},
		})
		return
	}
	hlMu.Unlock()

	hlMu.Lock()
	delete(activeHLs, key)
	hlMu.Unlock()

	cfg := getCfg(p.db, gid)
	rcfg := getResCfg(p.mgr, s)
	nextNum := rand.Intn(100) + 1

	isWin := false
	isJackpot := false

	if action == "higher" && nextNum > g.Number {
		isWin = true
	} else if action == "lower" && nextNum < g.Number {
		isWin = true
	} else if action == "jackpot" && nextNum == g.Number {
		isWin = true
		isJackpot = true
	}

	payout := int64(0)
	resMsg := ""

	if isWin {
		if isJackpot {
			payout = g.Bet * 6
			resMsg = fmt.Sprintf("JACKPOT! The next number was indeed %d (exactly equal)! You won %s!", nextNum, fmtCoins(payout-g.Bet, cfg))
		} else {
			payout = g.Bet * 2
			resMsg = fmt.Sprintf("Correct! The next number was %d. You won %s!", nextNum, fmtCoins(g.Bet, cfg))
		}
		a := getAcct(p.db, gid, uid)
		a.Wallet += payout
		_ = saveAcct(p.db, gid, uid, a)
	} else {
		resMsg = fmt.Sprintf("Incorrect. The next number was %d. You lost %s.", nextNum, fmtCoins(g.Bet, cfg))
	}

	emb := config.Build(rcfg, config.EmbedOpt{
		Title:       "HighLow — Result",
		Description: fmt.Sprintf("The starting number was %d.\nThe next number is %d.\n\n%s", g.Number, nextNum, resMsg),
	})
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{emb}, Components: nil},
	})
}
