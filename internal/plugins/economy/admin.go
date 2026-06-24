package economy

import (
	"bytes"
	"fmt"
	"strings"
	"github.com/bwmarrin/discordgo"
	bolt "go.etcd.io/bbolt"
	"skyvern/internal/manager"
)

func adminCmds(p *EconomyPlugin) []*manager.Command {
	return []*manager.Command{
		{
			Trigger:     "eco",
			Name:        "eco",
			Description: "Admin commands for managing server economy settings and balances",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				gid := ctx.GuildID()
				if gid == "" {
					return ctx.Reply("Must be used in a server.")
				}

				perms, err := ctx.UserChannelPermissions(ctx.AuthorID(), ctx.ChanID())
				if err != nil || (perms&discordgo.PermissionManageGuild) == 0 {
					return ctx.Reply("You need Manage Server permission to use economy admin commands.")
				}

				if len(ctx.Args) == 0 {
					return ctx.Reply("Subcommands: set, add, remove, reset, resetall, symbol, name, toggle")
				}

				sub := strings.ToLower(ctx.Args[0])
				cfg := getCfg(ctx.DB, gid)

				switch sub {
				case "toggle":
					cfg.Enabled = !cfg.Enabled
					_ = saveCfg(ctx.DB, gid, cfg)
					status := "disabled"
					if cfg.Enabled {
						status = "enabled"
					}
					return ctx.Reply(fmt.Sprintf("Economy system has been %s on this server.", status))

				case "symbol":
					if len(ctx.Args) < 2 {
						return ctx.Reply("Usage: .eco symbol <new_symbol>")
					}
					cfg.Symbol = ctx.Args[1]
					_ = saveCfg(ctx.DB, gid, cfg)
					return ctx.Reply(fmt.Sprintf("Currency symbol updated to: %s", cfg.Symbol))

				case "name":
					if len(ctx.Args) < 2 {
						return ctx.Reply("Usage: .eco name <new_name>")
					}
					cfg.CurrencyName = strings.Join(ctx.Args[1:], " ")
					_ = saveCfg(ctx.DB, gid, cfg)
					return ctx.Reply(fmt.Sprintf("Currency name updated to: %s", cfg.CurrencyName))

				case "set", "add", "remove":
					if len(ctx.Args) < 4 {
						return ctx.Reply(fmt.Sprintf("Usage: .eco %s <@user> <wallet|bank> <amount>", sub))
					}
					targetID := resolveUser(ctx.Args[1])
					if targetID == "" {
						return ctx.Reply("Invalid target user.")
					}

					field := strings.ToLower(ctx.Args[2])
					if field != "wallet" && field != "bank" {
						return ctx.Reply("Must specify wallet or bank.")
					}

					amount, err := parseAmount(ctx.Args[3], 1000000000)
					if err != nil {
						return ctx.Reply("Invalid amount.")
					}

					acct := getAcct(ctx.DB, gid, targetID)

					if sub == "set" {
						if field == "wallet" {
							acct.Wallet = amount
						} else {
							acct.Bank = amount
						}
					} else if sub == "add" {
						if field == "wallet" {
							acct.Wallet += amount
						} else {
							acct.Bank += amount
						}
					} else {
						// remove
						if field == "wallet" {
							acct.Wallet -= amount
							if acct.Wallet < 0 {
								acct.Wallet = 0
							}
						} else {
							acct.Bank -= amount
							if acct.Bank < 0 {
								acct.Bank = 0
							}
						}
					}

					_ = saveAcct(ctx.DB, gid, targetID, acct)
					return ctx.Reply(fmt.Sprintf("Successfully updated <@%s>'s %s. New %s balance: %s",
						targetID, field, field, fmtCoins(getAcct(ctx.DB, gid, targetID).Wallet, cfg)))

				case "reset":
					if len(ctx.Args) < 2 {
						return ctx.Reply("Usage: .eco reset <@user>")
					}
					targetID := resolveUser(ctx.Args[1])
					if targetID == "" {
						return ctx.Reply("Invalid target user.")
					}

					_ = ctx.DB.Update(func(tx *bolt.Tx) error {
						bkt := tx.Bucket(bktEcoAccts)
						if bkt == nil {
							return nil
						}
						return bkt.Delete(ecoKey(gid, targetID))
					})
					return ctx.Reply(fmt.Sprintf("Successfully reset economy data for <@%s>.", targetID))

				case "resetall":
					err := ctx.DB.Update(func(tx *bolt.Tx) error {
						bkt := tx.Bucket(bktEcoAccts)
						if bkt == nil {
							return nil
						}
						c := bkt.Cursor()
						prefix := []byte(gid + ":")
						var keysToDelete [][]byte
						for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
							tmpKey := make([]byte, len(k))
							copy(tmpKey, k)
							keysToDelete = append(keysToDelete, tmpKey)
						}
						for _, key := range keysToDelete {
							if err := bkt.Delete(key); err != nil {
								return err
							}
						}
						return nil
					})
					if err != nil {
						return ctx.Reply("Failed to reset server economy data.")
					}
					return ctx.Reply("Successfully reset ALL economy data for this server.")

				default:
					return ctx.Reply("Unknown subcommand. Subcommands: set, add, remove, reset, resetall, symbol, name, toggle")
				}
			},
		},
	}
}
