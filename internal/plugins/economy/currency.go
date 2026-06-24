package economy

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"skyvern/internal/config"
	"skyvern/internal/manager"
)

func currencyCmds(p *EconomyPlugin) []*manager.Command {
	return []*manager.Command{
		{
			Trigger:     "balance",
			Aliases:     []string{"bal"},
			Name:        "balance",
			Description: "View wallet, bank, and net worth",
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

				targetID := ctx.AuthorID()
				targetTag := ctx.AuthorTag()
				if len(ctx.Args) > 0 {
					if resolved := resolveUser(ctx.Args[0]); resolved != "" {
						targetID = resolved
						if m, err := ctx.Session.GuildMember(gid, targetID); err == nil && m.User != nil {
							targetTag = m.User.Username
						} else if u, err := ctx.Session.User(targetID); err == nil {
							targetTag = u.Username
						} else {
							targetTag = "User " + targetID
						}
					}
				}

				a := getAcct(ctx.DB, gid, targetID)
				netWorth := a.Wallet + a.Bank

				emb := config.Build(ctx.Cfg, config.EmbedOpt{
					Title: fmt.Sprintf("%s's Balance", targetTag),
					Fields: []*discordgo.MessageEmbedField{
						config.Field("Wallet", fmtCoins(a.Wallet, cfg), true),
						config.Field("Bank", fmt.Sprintf("%s / %s", fmtCoins(a.Bank, cfg), fmtCoins(a.BankMax, cfg)), true),
						config.Field("Net Worth", fmtCoins(netWorth, cfg), true),
						config.Field("Level / XP", fmt.Sprintf("Level %d (XP: %s)", a.Level, fmtInt(a.XP)), true),
					},
				})
				return ctx.Respond(emb)
			},
		},
		{
			Trigger:     "deposit",
			Aliases:     []string{"dep"},
			Name:        "deposit",
			Description: "Move coins from wallet to bank",
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
					return ctx.Reply("Usage: .deposit <amount|all>")
				}

				uid := ctx.AuthorID()
				a := getAcct(ctx.DB, gid, uid)
				if a.Wallet <= 0 {
					return ctx.Reply("You don't have any coins in your wallet to deposit.")
				}

				space := a.BankMax - a.Bank
				if space <= 0 {
					return ctx.Reply("Your bank is already full.")
				}

				amt, err := parseAmount(ctx.Args[0], a.Wallet)
				if err != nil {
					return ctx.Reply("Invalid deposit amount.")
				}

				fee := int64(math.Ceil(float64(amt) * 0.01))
				if a.Wallet < amt+fee {
					argLower := strings.ToLower(ctx.Args[0])
					if argLower == "all" || argLower == "max" {
						amt = int64(float64(a.Wallet) / 1.01)
						if amt > space {
							amt = space
						}
						fee = int64(math.Ceil(float64(amt) * 0.01))
						if amt <= 0 {
							return ctx.Reply("You do not have enough coins in your wallet to cover the 1% deposit fee.")
						}
					} else {
						return ctx.Reply(fmt.Sprintf("Insufficient funds to cover the 1%% bank fee. Needed: %s (Amount: %s + Fee: %s) | Wallet: %s",
							fmtCoins(amt+fee, cfg), fmtCoins(amt, cfg), fmtCoins(fee, cfg), fmtCoins(a.Wallet, cfg)))
					}
				}

				if amt > space {
					return ctx.Reply(fmt.Sprintf("Your bank does not have enough space. Remaining space: %s", fmtCoins(space, cfg)))
				}

				a.Wallet -= (amt + fee)
				a.Bank += amt
				_ = saveAcct(ctx.DB, gid, uid, a)

				return ctx.Reply(fmt.Sprintf("Deposited %s into your bank. A 1%% transaction fee of %s was paid.", fmtCoins(amt, cfg), fmtCoins(fee, cfg)))
			},
		},
		{
			Trigger:     "withdraw",
			Aliases:     []string{"with"},
			Name:        "withdraw",
			Description: "Move coins from bank to wallet",
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
					return ctx.Reply("Usage: .withdraw <amount|all>")
				}

				uid := ctx.AuthorID()
				a := getAcct(ctx.DB, gid, uid)
				if a.Bank <= 0 {
					return ctx.Reply("You don't have any coins in your bank to withdraw.")
				}

				amt, err := parseAmount(ctx.Args[0], a.Bank)
				if err != nil {
					return ctx.Reply("Invalid withdrawal amount.")
				}

				if amt > a.Bank {
					return ctx.Reply("You don't have that many coins in your bank.")
				}

				a.Wallet += amt
				a.Bank -= amt
				_ = saveAcct(ctx.DB, gid, uid, a)

				return ctx.Reply(fmt.Sprintf("Withdrew %s from your bank.", fmtCoins(amt, cfg)))
			},
		},
		{
			Trigger:     "pay",
			Aliases:     []string{"give"},
			Name:        "pay",
			Description: "Send coins to another user",
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
					return ctx.Reply("Usage: .pay <@user> <amount>")
				}

				targetID := resolveUser(ctx.Args[0])
				if targetID == "" {
					return ctx.Reply("Invalid user mention or ID.")
				}

				uid := ctx.AuthorID()
				if targetID == uid {
					return ctx.Reply("You cannot pay yourself.")
				}

				a := getAcct(ctx.DB, gid, uid)
				amt, err := parseAmount(ctx.Args[1], a.Wallet)
				if err != nil {
					return ctx.Reply("Invalid transfer amount.")
				}

				if amt > a.Wallet {
					return ctx.Reply("You don't have that many coins in your wallet.")
				}

				tax := int64(math.Ceil(float64(amt) * 0.05))
				netAmt := amt - tax
				if netAmt <= 0 {
					return ctx.Reply("Transfer amount is too small; net transfer must be at least 1 coin.")
				}

				targetAcct := getAcct(ctx.DB, gid, targetID)

				a.Wallet -= amt
				targetAcct.Wallet += netAmt

				_ = saveAcct(ctx.DB, gid, uid, a)
				_ = saveAcct(ctx.DB, gid, targetID, targetAcct)

				return ctx.Reply(fmt.Sprintf("Transferred %s to <@%s>. A 5%% transfer tax of %s was deducted (recipient received %s).",
					fmtCoins(amt, cfg), targetID, fmtCoins(tax, cfg), fmtCoins(netAmt, cfg)))
			},
		},
	}
}

func parseAmount(arg string, maxVal int64) (int64, error) {
	arg = strings.ToLower(strings.TrimSpace(arg))
	if arg == "all" || arg == "max" {
		return maxVal, nil
	}
	arg = strings.ReplaceAll(arg, ",", "")
	val, err := strconv.ParseInt(arg, 10, 64)
	if err != nil || val <= 0 {
		return 0, fmt.Errorf("invalid amount")
	}
	return val, nil
}
