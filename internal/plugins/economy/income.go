package economy

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"skyvern/internal/manager"
)

var jobs = []string{
	"Discord Moderator", "Software Engineer", "Dog Walker", "Fast Food Worker",
	"TikTok Influencer", "Janitor", "Bartender", "Uber Driver", "Crypto Trader",
	"Professional Gamer", "Chef", "Astronaut", "Private Investigator",
}

var begFails = []string{
	"Some rich person spat on you and walked away.",
	"A cop saw you begging and told you to move along.",
	"You got ignored by everyone passing by.",
	"A stranger threw a half-eaten sandwich at your face.",
	"You found a coin on the floor but it was just a plastic decoy.",
}

var crimeFails = []string{
	"You tried to pickpocket a police officer. You got arrested and fined.",
	"Your grand heist failed because you forgot to open the safe before carrying it.",
	"You tried to hack the server but ended up deleting your own browser history.",
	"You got caught shoplifting a single energy drink.",
}

var crimeWins = []string{
	"You successfully pickpocketed a tourist for",
	"You robbed a small convenience store and got away with",
	"You sold bootleg DVDs in an alleyway and made",
	"You successfully scammed a crypto enthusiast out of",
}

func incomeCmds(p *EconomyPlugin) []*manager.Command {
	return []*manager.Command{
		{
			Trigger:     "daily",
			Name:        "daily",
			Description: "Claim your daily coin reward",
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

				uid := ctx.AuthorID()
				a := getAcct(ctx.DB, gid, uid)

				if cd, active := getCooldown(a.LastDaily, 24*time.Hour); active {
					return ctx.Reply(fmt.Sprintf("You can claim your next daily reward in %d hours, %d minutes.", int(cd.Hours()), int(cd.Minutes())%60))
				}

				streak := a.Streak
				if !a.LastDaily.IsZero() && time.Since(a.LastDaily) < 48*time.Hour {
					streak++
				} else {
					streak = 1
				}

				inflation := getInflationIndex(ctx.DB, gid)
				incomeScale := math.Sqrt(inflation)

				base := rand.Int63n(cfg.DailyMax-cfg.DailyMin+1) + cfg.DailyMin
				streakBonus := int64(streak * 50)
				if streakBonus > 1500 {
					streakBonus = 1500
				}
				total := int64(float64(base)*incomeScale + float64(streakBonus)*incomeScale)

				// Progressive Wealth Tax
				netWorth := a.Wallet + a.Bank
				var tax int64
				var taxRate float64
				if netWorth > 1000000 {
					taxRate = 0.03
				} else if netWorth > 200000 {
					taxRate = 0.015
				} else if netWorth > 50000 {
					taxRate = 0.005
				}

				if taxRate > 0 {
					tax = int64(float64(netWorth) * taxRate)
					var discountSum float64
					for hType, qty := range a.Homes {
						if prop, ok := HomeProperties[hType]; ok {
							discountSum += float64(qty) * prop.TaxDiscount
						}
					}
					discount := 1.0 - discountSum
					if discount < 0.20 {
						discount = 0.20
					}
					tax = int64(float64(tax) * discount)

					if tax > a.Wallet {
						a.Bank -= (tax - a.Wallet)
						a.Wallet = 0
						if a.Bank < 0 {
							a.Bank = 0
						}
					} else {
						a.Wallet -= tax
					}
				}

				a.Wallet += total
				a.Streak = streak
				a.LastDaily = time.Now()

				_ = saveAcct(ctx.DB, gid, uid, a)

				lvl, up := addXP(ctx.DB, gid, uid, 50)
				lvlStr := ""
				if up {
					lvlStr = fmt.Sprintf("\nLevel up! You reached Level %d! Your bank capacity is now %s.", lvl, fmtCoins(10000+int64(lvl)*5000, cfg))
				}

				taxStr := ""
				if tax > 0 {
					taxStr = fmt.Sprintf("\nWealth Tax Paid: %s to the Guild Treasury (progressive rate: %.1f%%).", fmtCoins(tax, cfg), taxRate*100.0)
				}

				_, _, lotStr := drawLotteryIfNeeded(ctx)

				return ctx.Reply(fmt.Sprintf("You claimed your daily reward of %s! (Streak: %d days, Bonus: +%s)%s%s%s",
					fmtCoins(total, cfg), streak, fmtCoins(int64(float64(streakBonus)*incomeScale), cfg), lvlStr, taxStr, lotStr))
			},
		},
		{
			Trigger:     "work",
			Name:        "work",
			Description: "Work a shift at your job to earn coins",
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

				uid := ctx.AuthorID()
				a := getAcct(ctx.DB, gid, uid)

				if cd, active := getCooldown(a.LastWork, 30*time.Minute); active {
					return ctx.Reply(fmt.Sprintf("You are too tired to work! Try again in %d minutes, %d seconds.", int(cd.Minutes()), int(cd.Seconds())%60))
				}

				job := jobs[rand.Intn(len(jobs))]

				inflation := getInflationIndex(ctx.DB, gid)
				incomeScale := math.Sqrt(inflation)

				baseEarnings := rand.Int63n(cfg.WorkMax-cfg.WorkMin+1) + cfg.WorkMin
				earnings := int64(float64(baseEarnings) * incomeScale)

				a.Wallet += earnings
				a.LastWork = time.Now()
				_ = saveAcct(ctx.DB, gid, uid, a)

				lvl, up := addXP(ctx.DB, gid, uid, 15)
				lvlStr := ""
				if up {
					lvlStr = fmt.Sprintf("\nLevel up! You reached Level %d! Your bank capacity is now %s.", lvl, fmtCoins(10000+int64(lvl)*5000, cfg))
				}

				return ctx.Reply(fmt.Sprintf("You worked as a %s and earned %s!%s", job, fmtCoins(earnings, cfg), lvlStr))
			},
		},
		{
			Trigger:     "beg",
			Name:        "beg",
			Description: "Beg people on the street for coins",
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

				uid := ctx.AuthorID()
				a := getAcct(ctx.DB, gid, uid)

				if cd, active := getCooldown(a.LastBeg, 30*time.Second); active {
					return ctx.Reply(fmt.Sprintf("People are annoyed with you. Beg again in %d seconds.", int(cd.Seconds())))
				}

				a.LastBeg = time.Now()

				if rand.Float32() < 0.3 {
					_ = saveAcct(ctx.DB, gid, uid, a)
					return ctx.Reply(begFails[rand.Intn(len(begFails))])
				}

				inflation := getInflationIndex(ctx.DB, gid)
				incomeScale := math.Sqrt(inflation)

				baseEarnings := rand.Int63n(91) + 10
				earnings := int64(float64(baseEarnings) * incomeScale)

				a.Wallet += earnings
				_ = saveAcct(ctx.DB, gid, uid, a)

				lvl, up := addXP(ctx.DB, gid, uid, 2)
				lvlStr := ""
				if up {
					lvlStr = fmt.Sprintf("\nLevel up! You reached Level %d! Your bank capacity is now %s.", lvl, fmtCoins(10000+int64(lvl)*5000, cfg))
				}

				return ctx.Reply(fmt.Sprintf("A kind stranger handed you %s.%s", fmtCoins(earnings, cfg), lvlStr))
			},
		},
		{
			Trigger:     "crime",
			Name:        "crime",
			Description: "Attempt a high-risk crime to steal coins",
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

				uid := ctx.AuthorID()
				a := getAcct(ctx.DB, gid, uid)

				if cd, active := getCooldown(a.LastCrime, 1*time.Hour); active {
					return ctx.Reply(fmt.Sprintf("The heat is too high. Try again in %d minutes, %d seconds.", int(cd.Minutes()), int(cd.Seconds())%60))
				}

				a.LastCrime = time.Now()

				inflation := getInflationIndex(ctx.DB, gid)
				incomeScale := math.Sqrt(inflation)

				if rand.Intn(100) < cfg.CrimeFailPct {
					baseFine := rand.Int63n(cfg.CrimeMin/2 + 1) + (cfg.CrimeMin / 2)
					fine := int64(float64(baseFine) * incomeScale)
					if fine > a.Wallet {
						fine = a.Wallet
					}
					if fine < 0 {
						fine = 0
					}
					a.Wallet -= fine
					_ = saveAcct(ctx.DB, gid, uid, a)
					return ctx.Reply(fmt.Sprintf("%s\nYou were fined %s.", crimeFails[rand.Intn(len(crimeFails))], fmtCoins(fine, cfg)))
				}

				baseWinnings := rand.Int63n(cfg.CrimeMax-cfg.CrimeMin+1) + cfg.CrimeMin
				winnings := int64(float64(baseWinnings) * incomeScale)

				a.Wallet += winnings
				_ = saveAcct(ctx.DB, gid, uid, a)

				lvl, up := addXP(ctx.DB, gid, uid, 30)
				lvlStr := ""
				if up {
					lvlStr = fmt.Sprintf("\nLevel up! You reached Level %d! Your bank capacity is now %s.", lvl, fmtCoins(10000+int64(lvl)*5000, cfg))
				}

				return ctx.Reply(fmt.Sprintf("%s %s!%s", crimeWins[rand.Intn(len(crimeWins))], fmtCoins(winnings, cfg), lvlStr))
			},
		},
		{
			Trigger:     "rob",
			Aliases:     []string{"steal"},
			Name:        "rob",
			Description: "Rob coins from another user's wallet",
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
					return ctx.Reply("Usage: .rob <@user>")
				}

				targetID := resolveUser(ctx.Args[0])
				if targetID == "" {
					return ctx.Reply("Invalid user mention or ID.")
				}

				uid := ctx.AuthorID()
				if targetID == uid {
					return ctx.Reply("You cannot rob yourself.")
				}

				a := getAcct(ctx.DB, gid, uid)
				if cd, active := getCooldown(a.LastRob, 2*time.Hour); active {
					return ctx.Reply(fmt.Sprintf("You need to lie low. Rob again in %d minutes, %d seconds.", int(cd.Minutes()), int(cd.Seconds())%60))
				}

				targetAcct := getAcct(ctx.DB, gid, targetID)

				hasShield := false
				shieldIndex := -1
				for idx, item := range targetAcct.Inventory {
					if item.ID == "shield" && item.Qty > 0 {
						hasShield = true
						shieldIndex = idx
						break
					}
				}

				if hasShield {
					targetAcct.Inventory[shieldIndex].Qty--
					if targetAcct.Inventory[shieldIndex].Qty <= 0 {
						targetAcct.Inventory = append(targetAcct.Inventory[:shieldIndex], targetAcct.Inventory[shieldIndex+1:]...)
					}
					_ = saveAcct(ctx.DB, gid, targetID, targetAcct)

					a.LastRob = time.Now()
					_ = saveAcct(ctx.DB, gid, uid, a)

					return ctx.Reply(fmt.Sprintf("<@%s>'s shield activated and blocked the robbery attempt! The shield was destroyed.", targetID))
				}

				if targetAcct.Wallet < 500 {
					return ctx.Reply("It's not even worth robbing them, they have less than 500 coins in their wallet.")
				}

				a.LastRob = time.Now()

				inflation := getInflationIndex(ctx.DB, gid)
				incomeScale := math.Sqrt(inflation)

				if rand.Intn(100) < cfg.RobFailPct {
					baseFine := rand.Int63n(500) + 200
					fine := int64(float64(baseFine) * incomeScale)
					if fine > a.Wallet {
						fine = a.Wallet
					}
					if fine < 0 {
						fine = 0
					}
					a.Wallet -= fine
					targetAcct.Wallet += fine
					_ = saveAcct(ctx.DB, gid, uid, a)
					_ = saveAcct(ctx.DB, gid, targetID, targetAcct)
					return ctx.Reply(fmt.Sprintf("You got caught trying to rob <@%s> and paid them a fine of %s.", targetID, fmtCoins(fine, cfg)))
				}

				pct := float64(rand.Intn(41)+10) / 100.0
				robbedAmt := int64(float64(targetAcct.Wallet) * pct)
				if robbedAmt <= 0 {
					robbedAmt = 1
				}

				a.Wallet += robbedAmt
				targetAcct.Wallet -= robbedAmt

				_ = saveAcct(ctx.DB, gid, uid, a)
				_ = saveAcct(ctx.DB, gid, targetID, targetAcct)

				lvl, up := addXP(ctx.DB, gid, uid, 40)
				lvlStr := ""
				if up {
					lvlStr = fmt.Sprintf("\nLevel up! You reached Level %d! Your bank capacity is now %s.", lvl, fmtCoins(10000+int64(lvl)*5000, cfg))
				}

				return ctx.Reply(fmt.Sprintf("You successfully robbed %s from <@%s>'s wallet!%s", fmtCoins(robbedAmt, cfg), targetID, lvlStr))
			},
		},
	}
}
