package economy

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/storage"
)

var defaultShopItems = []ShopItem{
	{ID: "fishing_rod", Name: "Fishing Rod", Description: "Reduces fishing cooldown by 5m and boosts luck.", Price: 500, Stock: -1},
	{ID: "hunting_rifle", Name: "Hunting Rifle", Description: "Reduces hunting cooldown by 5m and protects from attacks.", Price: 1000, Stock: -1},
	{ID: "pickaxe", Name: "Pickaxe", Description: "Reduces mining cooldown by 5m and yields extra ores.", Price: 750, Stock: -1},
	{ID: "tide_charm", Name: "Tide Charm", Description: "Increases legendary fish catch rate.", Price: 2000, Stock: -1},
	{ID: "lootbox", Name: "Lootbox", Description: "Yields random items/coins when opened.", Price: 300, Stock: -1},
	{ID: "banknote", Name: "Banknote", Description: "Increases bank max capacity permanently by 2000.", Price: 1200, Stock: -1},
	{ID: "shield", Name: "Shield", Description: "Consumable that auto-blocks a robbery attempt.", Price: 800, Stock: -1},
	{ID: "coffee", Name: "Coffee", Description: "Consumed to reset the work cooldown.", Price: 150, Stock: -1},
	{ID: "chocolate", Name: "Chocolate", Description: "Consumed for 200 XP instantly.", Price: 100, Stock: -1},
	{ID: "ring", Name: "Ring", Description: "A shiny diamond ring (collectible).", Price: 5000, Stock: -1},
}

func shopCmds(p *EconomyPlugin) []*manager.Command {
	return []*manager.Command{
		{
			Trigger:     "shop",
			Aliases:     []string{"store"},
			Name:        "shop",
			Description: "View the server shop",
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

				page := 1
				if len(ctx.Args) > 0 {
					if pVal, err := strconv.Atoi(ctx.Args[0]); err == nil && pVal > 0 {
						page = pVal
					}
				}

				emb, components := buildShopMessage(ctx.DB, ctx.Cfg, gid, ctx.AuthorID(), page)
				ms := &discordgo.MessageSend{
					Embeds:     []*discordgo.MessageEmbed{emb},
					Components: components,
				}

				if ctx.Interact != nil {
					return ctx.Session.InteractionRespond(ctx.Interact, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Embeds:     ms.Embeds,
							Components: ms.Components,
						},
					})
				}

				_, err := ctx.Session.ChannelMessageSendComplex(ctx.ChanID(), ms)
				return err
			},
		},
		{
			Trigger:     "buy",
			Name:        "buy",
			Description: "Purchase an item from the shop",
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
					return ctx.Reply("Usage: .buy <item_id> [quantity]")
				}

				itemID := strings.ToLower(ctx.Args[0])
				qty := 1
				if len(ctx.Args) > 1 {
					if q, err := strconv.Atoi(ctx.Args[1]); err == nil && q > 0 {
						qty = q
					}
				}

				// Check default items first
				var it ShopItem
				found := false
				for _, dit := range defaultShopItems {
					if dit.ID == itemID {
						it = dit
						found = true
						break
					}
				}

				if !found {
					it, found = getShopItem(ctx.DB, gid, itemID)
				}

				if !found {
					return ctx.Reply("Item not found in the shop.")
				}

				if it.Stock >= 0 && it.Stock < qty {
					return ctx.Reply(fmt.Sprintf("Not enough stock remaining. Current stock: %d", it.Stock))
				}

				uid := ctx.AuthorID()
				a := getAcct(ctx.DB, gid, uid)

				inflation := getInflationIndex(ctx.DB, gid)
				price := int64(float64(it.Price) * inflation)

				if qty <= 0 {
					return ctx.Reply("Quantity must be a positive integer.")
				}
				if price > 0 && int64(qty) > 9223372036854775807/price {
					return ctx.Reply("Quantity is too large.")
				}

				totalPrice := price * int64(qty)
				if a.Wallet < totalPrice {
					return ctx.Reply(fmt.Sprintf("You do not have enough coins in your wallet. Total cost: %s", fmtCoins(totalPrice, cfg)))
				}

				// handle role granting
				if it.RoleID != "" {
					err := ctx.Session.GuildMemberRoleAdd(gid, uid, it.RoleID)
					if err != nil {
						return ctx.Reply("Failed to assign the purchased role. Please contact an admin to check bot permissions.")
					}
				}

				a.Wallet -= totalPrice
				_ = saveAcct(ctx.DB, gid, uid, a)

				// add to inventory
				addItem(ctx.DB, gid, uid, itemID, qty)

				// adjust stock for custom items
				if it.Stock >= 0 && it.RoleID == "" { // default tools aren't stock-limited, only custom items
					it.Stock -= qty
					_ = saveShopItem(ctx.DB, gid, it)
				}

				return ctx.Reply(fmt.Sprintf("You bought %dx %s for %s.", qty, it.Name, fmtCoins(totalPrice, cfg)))
			},
		},
		{
			Trigger:     "inventory",
			Aliases:     []string{"inv"},
			Name:        "inventory",
			Description: "View your inventory items",
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
				if len(a.Inventory) == 0 {
					return ctx.Reply(fmt.Sprintf("%s's inventory is empty.", targetTag))
				}

				var lines []string
				for _, item := range a.Inventory {
					desc := ""
					if di, ok := DefaultItems[item.ID]; ok {
						desc = di.Description
					} else if si, ok := getShopItem(ctx.DB, gid, item.ID); ok {
						desc = si.Description
					}
					if desc == "" {
						desc = "No description available."
					}
					lines = append(lines, fmt.Sprintf("**%s** (ID: `%s`) x%d\n*%s*", item.Name, item.ID, item.Qty, desc))
				}

				emb := config.Build(ctx.Cfg, config.EmbedOpt{
					Title:       fmt.Sprintf("%s's Inventory", targetTag),
					Description: strings.Join(lines, "\n\n"),
				})
				return ctx.Respond(emb)
			},
		},
		{
			Trigger:     "sell",
			Name:        "sell",
			Description: "Sell back an item for coins",
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
					return ctx.Reply("Usage: .sell <item_id> [quantity]")
				}

				itemID := strings.ToLower(ctx.Args[0])
				qty := 1
				if len(ctx.Args) > 1 {
					if q, err := strconv.Atoi(ctx.Args[1]); err == nil && q > 0 {
						qty = q
					}
				}

				uid := ctx.AuthorID()
				a := getAcct(ctx.DB, gid, uid)

				itemIndex := -1
				for i, item := range a.Inventory {
					if item.ID == itemID {
						itemIndex = i
						break
					}
				}

				if itemIndex == -1 || a.Inventory[itemIndex].Qty < qty {
					return ctx.Reply("You do not have enough of that item to sell.")
				}

				var pricePerItem int64
				var name string
				isGatherable := false

				// Check gathering items first
				if di, ok := DefaultItems[itemID]; ok && di.BuyPrice == 0 {
					// Gatherables use dynamic supply-and-demand prices
					pricePerItem = getDynamicPrice(ctx.DB, gid, itemID, di.SellPrice)
					name = di.Name
					isGatherable = true
				} else {
					// Check default tools
					var foundTool bool
					for _, dit := range defaultShopItems {
						if dit.ID == itemID {
							inflation := getInflationIndex(ctx.DB, gid)
							pricePerItem = int64(float64(dit.Price) * inflation / 2.0)
							name = dit.Name
							foundTool = true
							break
						}
					}
					if !foundTool {
						if si, ok := getShopItem(ctx.DB, gid, itemID); ok {
							inflation := getInflationIndex(ctx.DB, gid)
							pricePerItem = int64(float64(si.Price) * inflation / 2.0)
							name = si.Name
						} else {
							return ctx.Reply("This item cannot be sold.")
						}
					}
				}

				totalSell := pricePerItem * int64(qty)
				a.Wallet += totalSell

				a.Inventory[itemIndex].Qty -= qty
				if a.Inventory[itemIndex].Qty <= 0 {
					a.Inventory = append(a.Inventory[:itemIndex], a.Inventory[itemIndex+1:]...)
				}

				_ = saveAcct(ctx.DB, gid, uid, a)

				// Update sold supply to dynamically drop prices for future trades
				if isGatherable {
					updateSoldSupply(ctx.DB, gid, itemID, qty)
				}

				return ctx.Reply(fmt.Sprintf("You sold %dx %s and received %s (Current Market Value: %s/ea).",
					qty, name, fmtCoins(totalSell, cfg), fmtCoins(pricePerItem, cfg)))
			},
		},
		{
			Trigger:     "use",
			Name:        "use",
			Description: "Use an item from your inventory",
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
					return ctx.Reply("Usage: .use <item_id>")
				}

				itemID := strings.ToLower(ctx.Args[0])
				uid := ctx.AuthorID()
				a := getAcct(ctx.DB, gid, uid)

				itemIndex := -1
				for i, item := range a.Inventory {
					if item.ID == itemID {
						itemIndex = i
						break
					}
				}

				if itemIndex == -1 || a.Inventory[itemIndex].Qty <= 0 {
					return ctx.Reply("You do not have this item in your inventory.")
				}

				useText := ""
				consumed := true

				switch itemID {
				case "blowfish":
					useText = "You ate the blowfish and got poisoned! You paid $100 for medical bills."
					a.Wallet -= 100
					if a.Wallet < 0 {
						a.Wallet = 0
					}
				case "seaweed":
					useText = "You ate the seaweed. It tasted salty but fresh."
				case "old_boot":
					useText = "You put the old boot on your head. You look absolutely ridiculous."
				case "rabbit":
					useText = "You opened your hands and let the rabbit go free. It hopped away happily into the bushes."
				case "duck":
					useText = "The duck quacked at you loudly and bit your hand. Ouch."
				case "banknote":
					a.BankMax += 2000
					useText = fmt.Sprintf("You used a Banknote! Your maximum bank capacity has been increased to %s.", fmtCoins(a.BankMax, cfg))
				case "coffee":
					a.LastWork = time.Time{}
					useText = "You drank a hot cup of Coffee! Your work cooldown has been reset. You can work again immediately."
				case "chocolate":
					consumed = false
					a.Inventory[itemIndex].Qty--
					if a.Inventory[itemIndex].Qty <= 0 {
						a.Inventory = append(a.Inventory[:itemIndex], a.Inventory[itemIndex+1:]...)
					}
					_ = saveAcct(ctx.DB, gid, uid, a)
					lvl, up := addXP(ctx.DB, gid, uid, 200)
					lvlStr := ""
					if up {
						lvlStr = fmt.Sprintf("\nLevel up! You reached Level %d! Your bank capacity is now %s.", lvl, fmtCoins(10000+int64(lvl)*5000, cfg))
					}
					useText = fmt.Sprintf("You ate a sweet piece of Chocolate and gained 200 XP!%s", lvlStr)
				case "lootbox":
					consumed = false
					inflation := getInflationIndex(ctx.DB, gid)
					coinsGained := int64(float64(rand.Intn(401)+100) * math.Sqrt(inflation))
					a.Wallet += coinsGained

					itemRoll := rand.Intn(100)
					var rewardItem string
					if itemRoll < 40 {
						rewardItem = "coal"
					} else if itemRoll < 70 {
						rewardItem = "cod"
					} else if itemRoll < 90 {
						rewardItem = "iron_ore"
					} else {
						rewardItem = "salmon"
					}

					a.Inventory[itemIndex].Qty--
					if a.Inventory[itemIndex].Qty <= 0 {
						a.Inventory = append(a.Inventory[:itemIndex], a.Inventory[itemIndex+1:]...)
					}
					_ = saveAcct(ctx.DB, gid, uid, a)

					a = addItem(ctx.DB, gid, uid, rewardItem, 1)
					di := DefaultItems[rewardItem]

					useText = fmt.Sprintf("You cracked open a Lootbox and found %s and 1x %s! (\"%s\")",
						fmtCoins(coinsGained, cfg), di.Name, di.Description)
				case "shield":
					consumed = false
					useText = "You cannot use a Shield directly. Keep it in your inventory to automatically block one robbery attempt!"
				default:
					consumed = false
					useText = "This item cannot be used, but you can sell it back to the shop."
				}

				if consumed {
					a.Inventory[itemIndex].Qty--
					if a.Inventory[itemIndex].Qty <= 0 {
						a.Inventory = append(a.Inventory[:itemIndex], a.Inventory[itemIndex+1:]...)
					}
					_ = saveAcct(ctx.DB, gid, uid, a)
				}

				return ctx.Reply(useText)
			},
		},
		{
			Trigger:     "shopadd",
			Name:        "shopadd",
			Description: "Add a custom item to the server shop",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				gid := ctx.GuildID()
				if gid == "" {
					return ctx.Reply("Must be used in a server.")
				}
				p, err := ctx.UserChannelPermissions(ctx.AuthorID(), ctx.ChanID())
				if err != nil || (p&discordgo.PermissionManageGuild) == 0 {
					return ctx.Reply("You need Manage Server permission to use this command.")
				}

				if len(ctx.Args) < 3 {
					return ctx.Reply("Usage: .shopadd <id> <price> <name> [description...] [--role <@role>]")
				}

				itemID := strings.ToLower(ctx.Args[0])
				// Ensure they don't overwrite default tools
				for _, dit := range defaultShopItems {
					if dit.ID == itemID {
						return ctx.Reply("This ID is reserved for default shop tools.")
					}
				}

				price, err := parseAmount(ctx.Args[1], 1000000000)
				if err != nil {
					return ctx.Reply("Invalid price amount.")
				}

				roleID := ""
				var remainingArgs []string
				for i := 2; i < len(ctx.Args); i++ {
					if ctx.Args[i] == "--role" && i+1 < len(ctx.Args) {
						roleID = resolveUser(ctx.Args[i+1])
						i++
					} else {
						remainingArgs = append(remainingArgs, ctx.Args[i])
					}
				}

				if len(remainingArgs) == 0 {
					return ctx.Reply("Please specify a name for the shop item.")
				}

				name := remainingArgs[0]
				desc := ""
				if len(remainingArgs) > 1 {
					desc = strings.Join(remainingArgs[1:], " ")
				}

				it := ShopItem{
					ID:          itemID,
					Name:        name,
					Description: desc,
					Price:       price,
					RoleID:      roleID,
					Stock:       -1,
				}

				_ = saveShopItem(ctx.DB, gid, it)

				roleMsg := ""
				if roleID != "" {
					roleMsg = fmt.Sprintf(" linked to role <@&%s>", roleID)
				}
				return ctx.Reply(fmt.Sprintf("Added item '%s' (ID: %s) to the shop for %s%s.", name, itemID, fmtCoins(price, getCfg(ctx.DB, gid)), roleMsg))
			},
		},
		{
			Trigger:     "shopremove",
			Name:        "shopremove",
			Description: "Remove an item from the server shop",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				gid := ctx.GuildID()
				if gid == "" {
					return ctx.Reply("Must be used in a server.")
				}
				p, err := ctx.UserChannelPermissions(ctx.AuthorID(), ctx.ChanID())
				if err != nil || (p&discordgo.PermissionManageGuild) == 0 {
					return ctx.Reply("You need Manage Server permission to use this command.")
				}

				if len(ctx.Args) == 0 {
					return ctx.Reply("Usage: .shopremove <item_id>")
				}

				itemID := strings.ToLower(ctx.Args[0])
				_, found := getShopItem(ctx.DB, gid, itemID)
				if !found {
					return ctx.Reply("Item not found in the shop.")
				}

				_ = deleteShopItem(ctx.DB, gid, itemID)
				return ctx.Reply(fmt.Sprintf("Removed item '%s' from the shop.", itemID))
			},
		},
		{
			Trigger:     "shopedit",
			Name:        "shopedit",
			Description: "Edit an existing shop item's price, description or stock",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				gid := ctx.GuildID()
				if gid == "" {
					return ctx.Reply("Must be used in a server.")
				}
				p, err := ctx.UserChannelPermissions(ctx.AuthorID(), ctx.ChanID())
				if err != nil || (p&discordgo.PermissionManageGuild) == 0 {
					return ctx.Reply("You need Manage Server permission to use this command.")
				}

				if len(ctx.Args) < 3 {
					return ctx.Reply("Usage: .shopedit <item_id> <price|desc|stock> <value...>")
				}

				itemID := strings.ToLower(ctx.Args[0])
				field := strings.ToLower(ctx.Args[1])
				val := strings.Join(ctx.Args[2:], " ")

				it, found := getShopItem(ctx.DB, gid, itemID)
				if !found {
					return ctx.Reply("Item not found in the shop.")
				}

				switch field {
				case "price":
					price, err := parseAmount(val, 1000000000)
					if err != nil {
						return ctx.Reply("Invalid price amount.")
					}
					it.Price = price
				case "desc", "description":
					it.Description = val
				case "stock":
					stock, err := strconv.Atoi(val)
					if err != nil {
						return ctx.Reply("Invalid stock number.")
					}
					it.Stock = stock
				default:
					return ctx.Reply("Invalid field to edit. Use price, desc, or stock.")
				}

				_ = saveShopItem(ctx.DB, gid, it)
				return ctx.Reply(fmt.Sprintf("Updated item '%s' successfully.", itemID))
			},
		},
	}
}

func buildShopMessage(db *storage.DB, rcfg config.ResCfg, gid, uid string, page int) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	cfg := getCfg(db, gid)
	inflation := getInflationIndex(db, gid)
	items := getShop(db, gid)

	var allItems []ShopItem
	for _, it := range defaultShopItems {
		allItems = append(allItems, it)
	}
	for _, it := range items {
		allItems = append(allItems, it)
	}

	pageSize := 5
	totalItems := len(allItems)
	totalPages := int(math.Ceil(float64(totalItems) / float64(pageSize)))
	if totalPages == 0 {
		totalPages = 1
	}

	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if end > totalItems {
		end = totalItems
	}

	pageItems := allItems[start:end]

	var fields []*discordgo.MessageEmbedField
	for _, it := range pageItems {
		price := int64(float64(it.Price) * inflation)
		stockStr := "Unlimited"
		if it.Stock >= 0 {
			stockStr = strconv.Itoa(it.Stock)
		}
		roleStr := ""
		if it.RoleID != "" {
			roleStr = fmt.Sprintf(" [Grants role <@&%s>]", it.RoleID)
		}
		desc := it.Description
		if desc == "" {
			desc = "No description provided."
		}
		val := fmt.Sprintf("Price: %s\nStock: %s\n%s%s", fmtCoins(price, cfg), stockStr, desc, roleStr)
		fields = append(fields, config.Field(fmt.Sprintf("%s (ID: %s)", it.Name, it.ID), val, false))
	}

	emb := config.Build(rcfg, config.EmbedOpt{
		Title:       "Server Shop",
		Description: "Use the dropdown to purchase an item, or use `.buy <item_id> [qty]`.",
		Fields:      fields,
	})
	emb.Footer = &discordgo.MessageEmbedFooter{
		Text: fmt.Sprintf("Page %d of %d | Currency: %s", page, totalPages, cfg.Symbol),
	}

	var components []discordgo.MessageComponent

	if len(pageItems) > 0 {
		var selectOptions []discordgo.SelectMenuOption
		for _, it := range pageItems {
			price := int64(float64(it.Price) * inflation)
			label := fmt.Sprintf("%s — %s", it.Name, fmtCoins(price, cfg))
			if len(label) > 100 {
				label = label[:97] + "..."
			}
			desc := it.Description
			if desc == "" {
				desc = "No description"
			}
			if len(desc) > 100 {
				desc = desc[:97] + "..."
			}
			selectOptions = append(selectOptions, discordgo.SelectMenuOption{
				Label:       label,
				Value:       it.ID,
				Description: desc,
			})
		}
		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    fmt.Sprintf("shop:buy_select:%s:%s", gid, uid),
					Placeholder: "Select an item to buy...",
					Options:     selectOptions,
				},
			},
		})
	}

	prevPage := page - 1
	nextPage := page + 1

	prevDisabled := prevPage < 1
	nextDisabled := nextPage > totalPages

	if prevPage < 1 {
		prevPage = 1
	}
	if nextPage > totalPages {
		nextPage = totalPages
	}

	buttonsRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "◀ Prev",
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("shop:page:%d:%s:%s", prevPage, gid, uid),
				Disabled: prevDisabled,
			},
			discordgo.Button{
				Label:    "Next ▶",
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("shop:page:%d:%s:%s", nextPage, gid, uid),
				Disabled: nextDisabled,
			},
		},
	}
	components = append(components, buttonsRow)

	return emb, components
}

func parsePageFromFooter(footerText string) int {
	if !strings.HasPrefix(footerText, "Page ") {
		return 1
	}
	parts := strings.Split(footerText, " ")
	if len(parts) < 2 {
		return 1
	}
	p, err := strconv.Atoi(parts[1])
	if err != nil {
		return 1
	}
	return p
}

func (p *EconomyPlugin) handleShopInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
			Data: &discordgo.InteractionResponseData{
				Content: "This is not your shop menu. Run `.shop` to open your own.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	cfg := getCfg(p.db, gid)
	rcfg := getResCfg(p.mgr, s)

	switch action {
	case "page":
		if len(parts) < 5 {
			return
		}
		pageNum, err := strconv.Atoi(parts[2])
		if err != nil {
			pageNum = 1
		}
		gid = parts[3]
		uid = parts[4]

		emb, components := buildShopMessage(p.db, rcfg, gid, uid, pageNum)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{emb},
				Components: components,
			},
		})

	case "buy_select":
		values := i.MessageComponentData().Values
		if len(values) == 0 {
			return
		}
		itemID := values[0]
		qty := 1

		var it ShopItem
		found := false
		for _, dit := range defaultShopItems {
			if dit.ID == itemID {
				it = dit
				found = true
				break
			}
		}
		if !found {
			it, found = getShopItem(p.db, gid, itemID)
		}

		if !found {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Item not found in the shop.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		if it.Stock >= 0 && it.Stock < qty {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Not enough stock remaining. Current stock: %d", it.Stock),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		a := getAcct(p.db, gid, uid)
		inflation := getInflationIndex(p.db, gid)
		price := int64(float64(it.Price) * inflation)
		totalPrice := price * int64(qty)

		if a.Wallet < totalPrice {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("You do not have enough coins in your wallet. Total cost: %s", fmtCoins(totalPrice, cfg)),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		if it.RoleID != "" {
			err := s.GuildMemberRoleAdd(gid, uid, it.RoleID)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Failed to assign the purchased role. Please contact an admin to check bot permissions.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
		}

		a.Wallet -= totalPrice
		_ = saveAcct(p.db, gid, uid, a)

		addItem(p.db, gid, uid, itemID, qty)

		if it.Stock >= 0 && it.RoleID == "" {
			it.Stock -= qty
			_ = saveShopItem(p.db, gid, it)
		}

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("🎉 You successfully bought %dx **%s** for %s!", qty, it.Name, fmtCoins(totalPrice, cfg)),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

		page := 1
		if len(i.Message.Embeds) > 0 && i.Message.Embeds[0].Footer != nil {
			page = parsePageFromFooter(i.Message.Embeds[0].Footer.Text)
		}
		emb, components := buildShopMessage(p.db, rcfg, gid, uid, page)
		_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			ID:         i.Message.ID,
			Channel:    i.Message.ChannelID,
			Embeds:     &[]*discordgo.MessageEmbed{emb},
			Components: &components,
		})
	}
}
