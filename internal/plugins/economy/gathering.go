package economy

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"skyvern/internal/manager"
	"skyvern/internal/storage"
)

type DefaultItem struct {
	ID          string
	Name        string
	Description string
	SellPrice   int64
	BuyPrice    int64
}

var DefaultItems = map[string]DefaultItem{
	// Fish
	"cod":             {ID: "cod", Name: "Cod", Description: "A common cod fish.", SellPrice: 40},
	"salmon":          {ID: "salmon", Name: "Salmon", Description: "A tasty salmon.", SellPrice: 60},
	"seaweed":         {ID: "seaweed", Name: "Seaweed", Description: "Salty green seaweed.", SellPrice: 15},
	"jellyfish":       {ID: "jellyfish", Name: "Jellyfish", Description: "A squishy, stinging jellyfish.", SellPrice: 150},
	"blowfish":        {ID: "blowfish", Name: "Blowfish", Description: "Spiky and dangerous blowfish.", SellPrice: 200},
	"old_boot":        {ID: "old_boot", Name: "Old Boot", Description: "A smelly boot fished from the depths.", SellPrice: 5},
	"golden_fish":     {ID: "golden_fish", Name: "Golden Fish", Description: "A legendary fish made of solid gold!", SellPrice: 1500},
	"sunken_treasure": {ID: "sunken_treasure", Name: "Sunken Treasure", Description: "A rusty chest filled with ancient valuables.", SellPrice: 2500},
	"tuna":            {ID: "tuna", Name: "Tuna", Description: "A large tuna fish.", SellPrice: 100},
	"squid":           {ID: "squid", Name: "Squid", Description: "A slippery nocturnal squid.", SellPrice: 75},

	// Hunting
	"rabbit":         {ID: "rabbit", Name: "Rabbit", Description: "A small fluffy rabbit.", SellPrice: 50},
	"duck":           {ID: "duck", Name: "Duck", Description: "Quack.", SellPrice: 50},
	"deer":           {ID: "deer", Name: "Deer", Description: "A majestic deer.", SellPrice: 200},
	"boar":           {ID: "boar", Name: "Boar", Description: "A wild boar with sharp tusks.", SellPrice: 250},
	"fox":            {ID: "fox", Name: "Fox", Description: "A cunning red fox.", SellPrice: 180},
	"dragon_egg":     {ID: "dragon_egg", Name: "Dragon Egg", Description: "A legendary warm dragon egg.", SellPrice: 3000},
	"mythic_griffin": {ID: "mythic_griffin", Name: "Mythic Griffin", Description: "A captured griffin of legend.", SellPrice: 5000},
	"bear":            {ID: "bear", Name: "Bear", Description: "A large grizzly bear.", SellPrice: 350},

	// Mining
	"coal":            {ID: "coal", Name: "Coal", Description: "A lump of coal.", SellPrice: 30},
	"iron_ore":        {ID: "iron_ore", Name: "Iron Ore", Description: "Raw iron ore, needs smelting.", SellPrice: 80},
	"gold_ore":        {ID: "gold_ore", Name: "Gold Ore", Description: "Raw gold ore, shiny.", SellPrice: 250},
	"emerald":         {ID: "emerald", Name: "Emerald", Description: "A cut green emerald.", SellPrice: 600},
	"diamond":         {ID: "diamond", Name: "Diamond", Description: "A rare sparkling diamond.", SellPrice: 2000},
	"ruby":            {ID: "ruby", Name: "Ruby", Description: "A blood-red ruby.", SellPrice: 1200},
	"copper_ore":      {ID: "copper_ore", Name: "Copper Ore", Description: "Raw copper ore.", SellPrice: 50},
	"sapphire":        {ID: "sapphire", Name: "Sapphire", Description: "A beautiful blue gem.", SellPrice: 800},
	"meteorite_shard": {ID: "meteorite_shard", Name: "Meteorite Shard", Description: "A cosmic fragment of rock.", SellPrice: 4000},

	// Tools
	"fishing_rod":   {ID: "fishing_rod", Name: "Fishing Rod", Description: "Reduces fishing cooldown by 5m and boosts luck.", SellPrice: 250, BuyPrice: 500},
	"hunting_rifle": {ID: "hunting_rifle", Name: "Hunting Rifle", Description: "Reduces hunting cooldown by 5m and protects from attacks.", SellPrice: 500, BuyPrice: 1000},
	"pickaxe":       {ID: "pickaxe", Name: "Pickaxe", Description: "Reduces mining cooldown by 5m and yields extra ores.", SellPrice: 375, BuyPrice: 750},
	"tide_charm":    {ID: "tide_charm", Name: "Tide Charm", Description: "Increases legendary fish catch rate.", SellPrice: 1000, BuyPrice: 2000},
}

func hasItem(a EcoAccount, itemID string) bool {
	for _, item := range a.Inventory {
		if item.ID == itemID && item.Qty > 0 {
			return true
		}
	}
	return false
}

func addItem(db *storage.DB, gid, uid, itemID string, qty int) EcoAccount {
	a := getAcct(db, gid, uid)
	found := false
	for i, item := range a.Inventory {
		if item.ID == itemID {
			a.Inventory[i].Qty += qty
			found = true
			break
		}
	}
	if !found {
		name := itemID
		if di, ok := DefaultItems[itemID]; ok {
			name = di.Name
		} else if si, ok := getShopItem(db, gid, itemID); ok {
			name = si.Name
		}
		a.Inventory = append(a.Inventory, InvItem{ID: itemID, Name: name, Qty: qty})
	}
	_ = saveAcct(db, gid, uid, a)
	return a
}

func gatheringCmds(p *EconomyPlugin) []*manager.Command {
	return []*manager.Command{
		{
			Trigger:     "fish",
			Name:        "fish",
			Description: "Go fishing to catch fish or find coins",
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

				hasRod := hasItem(a, "fishing_rod")
				hasCharm := hasItem(a, "tide_charm")

				cooldown := 30 * time.Minute
				if hasRod {
					cooldown = 25 * time.Minute
				}

				if cd, active := getCooldown(a.LastFish, cooldown); active {
					return ctx.Reply(fmt.Sprintf("You already fished recently. Wait %d minutes, %d seconds.", int(cd.Minutes()), int(cd.Seconds())%60))
				}

				a.LastFish = time.Now()
				_ = saveAcct(ctx.DB, gid, uid, a)

				lvl, up := addXP(ctx.DB, gid, uid, 20)
				lvlStr := ""
				if up {
					lvlStr = fmt.Sprintf("\nLevel up! You reached Level %d! Your bank capacity is now %s.", lvl, fmtCoins(10000+int64(lvl)*5000, cfg))
				}

				weather := updateWeather(ctx.DB, gid)
				hour := time.Now().Hour()
				isNight := hour >= 20 || hour < 6

				inflation := getInflationIndex(ctx.DB, gid)
				incomeScale := math.Sqrt(inflation)

				// Weather / Tool bonuses
				successRate := float32(0.4)
				if weather == "Rainy" || weather == "Stormy" {
					successRate = 0.55
				}
				if hasRod {
					successRate += 0.1
				}

				if rand.Float32() < successRate {
					baseCoins := rand.Int63n(151) + 50
					coins := int64(float64(baseCoins) * incomeScale)
					a = getAcct(ctx.DB, gid, uid)
					a.Wallet += coins
					_ = saveAcct(ctx.DB, gid, uid, a)
					return ctx.Reply(fmt.Sprintf("[Weather: %s] You went fishing and reeled in a wet wallet containing %s!%s", weather, fmtCoins(coins, cfg), lvlStr))
				}

				// Rarity calculations
				roll := rand.Intn(100)
				if hasRod {
					roll += 10
				}
				if hasCharm {
					roll += 15
				}
				if weather == "Rainy" {
					roll += 10
				} else if weather == "Stormy" {
					roll += 15
				}

				var itemID string
				if isNight && rand.Float32() < 0.3 {
					itemID = "squid"
				} else if roll < 50 {
					commons := []string{"cod", "salmon", "seaweed", "old_boot"}
					itemID = commons[rand.Intn(len(commons))]
				} else if roll < 85 {
					rares := []string{"jellyfish", "blowfish", "tuna"}
					itemID = rares[rand.Intn(len(rares))]
				} else {
					legendaries := []string{"golden_fish", "sunken_treasure"}
					itemID = legendaries[rand.Intn(len(legendaries))]
				}

				addItem(ctx.DB, gid, uid, itemID, 1)
				di := DefaultItems[itemID]

				return ctx.Reply(fmt.Sprintf("[Weather: %s] You went fishing and caught a %s! (\"%s\")%s", weather, di.Name, di.Description, lvlStr))
			},
		},
		{
			Trigger:     "hunt",
			Name:        "hunt",
			Description: "Go hunting in the forest for wildlife or coins",
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

				hasRifle := hasItem(a, "hunting_rifle")

				cooldown := 30 * time.Minute
				if hasRifle {
					cooldown = 25 * time.Minute
				}

				if cd, active := getCooldown(a.LastHunt, cooldown); active {
					return ctx.Reply(fmt.Sprintf("You already hunted recently. Wait %d minutes, %d seconds.", int(cd.Minutes()), int(cd.Seconds())%60))
				}

				a.LastHunt = time.Now()
				_ = saveAcct(ctx.DB, gid, uid, a)

				lvl, up := addXP(ctx.DB, gid, uid, 20)
				lvlStr := ""
				if up {
					lvlStr = fmt.Sprintf("\nLevel up! You reached Level %d! Your bank capacity is now %s.", lvl, fmtCoins(10000+int64(lvl)*5000, cfg))
				}

				weather := updateWeather(ctx.DB, gid)
				inflation := getInflationIndex(ctx.DB, gid)
				incomeScale := math.Sqrt(inflation)

				// Weather risks (Stormy lightning strike)
				if weather == "Stormy" && !hasRifle && rand.Float32() < 0.2 {
					loss := int64(float64(rand.Intn(201)+100) * incomeScale)
					a = getAcct(ctx.DB, gid, uid)
					a.Wallet -= loss
					if a.Wallet < 0 {
						a.Wallet = 0
					}
					_ = saveAcct(ctx.DB, gid, uid, a)
					return ctx.Reply(fmt.Sprintf("[Weather: Stormy] You got caught in a heavy thunderstorm. Lightning struck a nearby tree and you dropped %s out of panic!%s", fmtCoins(loss, cfg), lvlStr))
				}

				successRate := float32(0.4)
				if weather == "Sunny" {
					successRate = 0.5
				}
				if hasRifle {
					successRate += 0.15
				}

				if rand.Float32() < successRate {
					baseCoins := rand.Int63n(191) + 60
					coins := int64(float64(baseCoins) * incomeScale)
					a = getAcct(ctx.DB, gid, uid)
					a.Wallet += coins
					_ = saveAcct(ctx.DB, gid, uid, a)
					return ctx.Reply(fmt.Sprintf("[Weather: %s] You went hunting and tracked down a lost bag containing %s!%s", weather, fmtCoins(coins, cfg), lvlStr))
				}

				roll := rand.Intn(100)
				if hasRifle {
					roll += 15
				}
				if weather == "Sunny" {
					roll += 10
				}

				var itemID string
				if roll < 50 {
					commons := []string{"rabbit", "duck"}
					itemID = commons[rand.Intn(len(commons))]
				} else if roll < 85 {
					rares := []string{"deer", "boar", "fox"}
					itemID = rares[rand.Intn(len(rares))]
				} else {
					legendaries := []string{"dragon_egg", "mythic_griffin", "bear"}
					itemID = legendaries[rand.Intn(len(legendaries))]
				}

				// Bear attack hazard
				if itemID == "bear" && !hasRifle && rand.Float32() < 0.5 {
					loss := int64(float64(rand.Intn(301)+200) * incomeScale)
					a = getAcct(ctx.DB, gid, uid)
					a.Wallet -= loss
					if a.Wallet < 0 {
						a.Wallet = 0
					}
					_ = saveAcct(ctx.DB, gid, uid, a)
					return ctx.Reply(fmt.Sprintf("[Weather: %s] You encountered a wild Grizzly Bear! You didn't have a rifle to defend yourself and had to drop %s to distract it and run away!%s", weather, fmtCoins(loss, cfg), lvlStr))
				}

				addItem(ctx.DB, gid, uid, itemID, 1)
				di := DefaultItems[itemID]

				return ctx.Reply(fmt.Sprintf("[Weather: %s] You went hunting and brought back a %s! (\"%s\")%s", weather, di.Name, di.Description, lvlStr))
			},
		},
		{
			Trigger:     "mine",
			Aliases:     []string{"dig"},
			Name:        "mine",
			Description: "Mine in the caves for precious ores and gems",
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

				hasPick := hasItem(a, "pickaxe")

				cooldown := 30 * time.Minute
				if hasPick {
					cooldown = 25 * time.Minute
				}

				if cd, active := getCooldown(a.LastMine, cooldown); active {
					return ctx.Reply(fmt.Sprintf("You already mined recently. Wait %d minutes, %d seconds.", int(cd.Minutes()), int(cd.Seconds())%60))
				}

				a.LastMine = time.Now()
				_ = saveAcct(ctx.DB, gid, uid, a)

				lvl, up := addXP(ctx.DB, gid, uid, 20)
				lvlStr := ""
				if up {
					lvlStr = fmt.Sprintf("\nLevel up! You reached Level %d! Your bank capacity is now %s.", lvl, fmtCoins(10000+int64(lvl)*5000, cfg))
				}

				weather := updateWeather(ctx.DB, gid)
				inflation := getInflationIndex(ctx.DB, gid)
				incomeScale := math.Sqrt(inflation)

				// Cave-in hazard
				if weather == "Stormy" && rand.Float32() < 0.15 {
					a = getAcct(ctx.DB, gid, uid)
					loss := int64(100 * incomeScale)
					a.Wallet -= loss
					if a.Wallet < 0 {
						a.Wallet = 0
					}
					_ = saveAcct(ctx.DB, gid, uid, a)
					return ctx.Reply(fmt.Sprintf("[Weather: Stormy] A sudden cave-in occurred due to rain-soaked ground! You had to pay %s for rescue operations and found no ores.%s", fmtCoins(loss, cfg), lvlStr))
				}

				successRate := float32(0.4)
				if hasPick {
					successRate += 0.1
				}

				if rand.Float32() < successRate {
					baseCoins := rand.Int63n(141) + 40
					coins := int64(float64(baseCoins) * incomeScale)
					a = getAcct(ctx.DB, gid, uid)
					a.Wallet += coins
					_ = saveAcct(ctx.DB, gid, uid, a)
					return ctx.Reply(fmt.Sprintf("[Weather: %s] You swung your pickaxe and cracked open a geode containing %s!%s", weather, fmtCoins(coins, cfg), lvlStr))
				}

				roll := rand.Intn(100)
				if hasPick {
					roll += 10
				}

				var itemID string
				if roll < 50 {
					commons := []string{"coal", "iron_ore", "copper_ore"}
					itemID = commons[rand.Intn(len(commons))]
				} else if roll < 85 {
					rares := []string{"gold_ore", "emerald", "sapphire"}
					itemID = rares[rand.Intn(len(rares))]
				} else {
					legendaries := []string{"diamond", "ruby", "meteorite_shard"}
					itemID = legendaries[rand.Intn(len(legendaries))]
				}

				qty := 1
				if hasPick && rand.Float32() < 0.5 {
					qty = rand.Intn(2) + 2 // 2 or 3 ores
				}

				addItem(ctx.DB, gid, uid, itemID, qty)
				di := DefaultItems[itemID]

				return ctx.Reply(fmt.Sprintf("[Weather: %s] You went mining and extracted %dx %s! (\"%s\")%s", weather, qty, di.Name, di.Description, lvlStr))
			},
		},
	}
}
