package economy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	bolt "go.etcd.io/bbolt"
	"skyvern/internal/config"
	"skyvern/internal/manager"
)

type userBal struct {
	UserID string
	Wallet int64
	Bank   int64
}

func getRichest(mgr *manager.Manager, gid string) ([]userBal, error) {
	var list []userBal
	err := mgr.DB().View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktEcoAccts)
		if bkt == nil {
			return nil
		}
		c := bkt.Cursor()
		prefix := []byte(gid + ":")
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			parts := strings.Split(string(k), ":")
			if len(parts) < 2 {
				continue
			}
			uid := parts[1]
			var a EcoAccount
			if err := json.Unmarshal(v, &a); err == nil {
				list = append(list, userBal{
					UserID: uid,
					Wallet: a.Wallet,
					Bank:   a.Bank,
				})
			}
		}
		return nil
	})
	return list, err
}

func leaderboardCmds(p *EconomyPlugin) []*manager.Command {
	return []*manager.Command{
		{
			Trigger:     "richest",
			Aliases:     []string{"leaderboard", "lb"},
			Name:        "richest",
			Description: "View the server economy leaderboard",
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

				list, err := getRichest(ctx.Mgr, gid)
				if err != nil {
					return ctx.Reply("Failed to fetch leaderboard data.")
				}

				if len(list) == 0 {
					return ctx.Reply("No economy accounts found on this server.")
				}

				sort.Slice(list, func(i, j int) bool {
					return (list[i].Wallet + list[i].Bank) > (list[j].Wallet + list[j].Bank)
				})

				page := 1
				if len(ctx.Args) > 0 {
					if p, err := strconv.Atoi(ctx.Args[0]); err == nil && p > 0 {
						page = p
					}
				}

				perPage := 10
				totalPages := (len(list) + perPage - 1) / perPage
				if page > totalPages {
					page = totalPages
				}

				start := (page - 1) * perPage
				end := start + perPage
				if end > len(list) {
					end = len(list)
				}

				var lines []string
				for i := start; i < end; i++ {
					item := list[i]
					net := item.Wallet + item.Bank
					lines = append(lines, fmt.Sprintf("%d. <@%s> - Net Worth: %s (Wallet: %s | Bank: %s)",
						i+1, item.UserID, fmtCoins(net, cfg), fmtCoins(item.Wallet, cfg), fmtCoins(item.Bank, cfg)))
				}

				emb := config.Build(ctx.Cfg, config.EmbedOpt{
					Title:       "Server Richest Leaderboard",
					Description: strings.Join(lines, "\n"),
					ThumbnailURL: fmt.Sprintf("Page %d of %d", page, totalPages), // repurposed thumbnail or footer can show this
				})
				emb.Footer.Text = fmt.Sprintf("Page %d/%d | Total Users: %d", page, totalPages, len(list))

				return ctx.Respond(emb)
			},
		},
		{
			Trigger:     "networth",
			Aliases:     []string{"nw"},
			Name:        "networth",
			Description: "Detailed asset net worth breakdown",
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

				var invVal int64
				for _, item := range a.Inventory {
					if di, ok := DefaultItems[item.ID]; ok {
						invVal += di.SellPrice * int64(item.Qty)
					} else if si, ok := getShopItem(ctx.DB, gid, item.ID); ok {
						invVal += (si.Price / 2) * int64(item.Qty)
					}
				}

				var stocksVal int64
				var symbols []string
				for sym, sh := range a.Stocks {
					if sh > 0 {
						symbols = append(symbols, sym)
					}
				}
				if len(symbols) > 0 {
					if res, err := getQuoteWithRetry(symbols); err == nil {
						for _, item := range res.QuoteResponse.Result {
							shares := a.Stocks[item.Symbol]
							stocksVal += int64(shares * item.RegularMarketPrice)
						}
					}
				}

				net := a.Wallet + a.Bank + invVal + stocksVal

				description := fmt.Sprintf(
					"**Wallet:** %s\n**Bank:** %s\n**Inventory Value:** %s\n**Stocks Value:** %s\n\n**Total Net Worth:** %s",
					fmtCoins(a.Wallet, cfg),
					fmtCoins(a.Bank, cfg),
					fmtCoins(invVal, cfg),
					fmtCoins(stocksVal, cfg),
					fmtCoins(net, cfg),
				)

				emb := config.Build(ctx.Cfg, config.EmbedOpt{
					Title:       fmt.Sprintf("%s's Net Worth Breakdown", targetTag),
					Description: description,
				})
				return ctx.Respond(emb)
			},
		},
	}
}
