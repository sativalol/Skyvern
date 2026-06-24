package economy

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"skyvern/internal/config"
	"skyvern/internal/manager"
)

type AssetType int

const (
	AssetBusiness AssetType = iota
	AssetLand
	AssetHome
)

type Asset struct {
	ID    string
	Name  string
	Type  AssetType
	Cost  int64
	Yield int64 // hourly passive yield (businesses only)
	Desc  string
}

type HomeProperty struct {
	CapacityBonus int64
	TaxDiscount   float64
}

var HomeProperties = map[string]HomeProperty{
	"apartment":      {CapacityBonus: 5000, TaxDiscount: 0.10},
	"condo":          {CapacityBonus: 20000, TaxDiscount: 0.20},
	"duplex":         {CapacityBonus: 35000, TaxDiscount: 0.30},
	"mansion":        {CapacityBonus: 50000, TaxDiscount: 0.40},
	"suburban_house": {CapacityBonus: 100000, TaxDiscount: 0.50},
	"penthouse":      {CapacityBonus: 250000, TaxDiscount: 0.60},
	"skyscraper":     {CapacityBonus: 2000000, TaxDiscount: 0.80},
}

var assetsCatalog = map[string]Asset{
	"lemonade":        {ID: "lemonade", Name: "Lemonade Stand", Type: AssetBusiness, Cost: 1000, Yield: 10, Desc: "A sidewalk lemonade stand."},
	"retail":          {ID: "retail", Name: "Retail Store", Type: AssetBusiness, Cost: 5000, Yield: 60, Desc: "A local storefront."},
	"coffeeshop":      {ID: "coffeeshop", Name: "Coffee Shop", Type: AssetBusiness, Cost: 12000, Yield: 140, Desc: "A bustling neighborhood cafe."},
	"tech":            {ID: "tech", Name: "Tech Startup", Type: AssetBusiness, Cost: 25000, Yield: 350, Desc: "A high-growth tech company."},
	"crypto":          {ID: "crypto", Name: "Crypto Farm", Type: AssetBusiness, Cost: 100000, Yield: 1500, Desc: "A warehouse of mining rigs."},
	"hotel":           {ID: "hotel", Name: "Luxury Hotel", Type: AssetBusiness, Cost: 500000, Yield: 8500, Desc: "A 5-star hotel catering to elite guests."},
	"oil_well":        {ID: "oil_well", Name: "Oil Rig", Type: AssetBusiness, Cost: 5000000, Yield: 100000, Desc: "An offshore oil extraction rig pumping black gold."},
	"conglomerate":    {ID: "conglomerate", Name: "Mega Conglomerate", Type: AssetBusiness, Cost: 20000000, Yield: 500000, Desc: "A multinational corporate empire."},

	"plot":            {ID: "plot", Name: "Land Plot", Type: AssetLand, Cost: 2500, Desc: "Vacant land appreciating with inflation."},
	"beachfront_plot": {ID: "beachfront_plot", Name: "Beachfront Plot", Type: AssetLand, Cost: 15000, Desc: "Appreciates faster with inflation. Highly sought-after beachfront property."},
	"private_island":  {ID: "private_island", Name: "Private Island", Type: AssetLand, Cost: 1200000, Desc: "An exclusive tropical island. Appreciates heavily."},

	"apartment":      {ID: "apartment", Name: "Apartment", Type: AssetHome, Cost: 10000, Desc: "Adds +5,000 bank capacity, 10% tax discount."},
	"condo":          {ID: "condo", Name: "Condominium", Type: AssetHome, Cost: 40000, Desc: "A modern urban condominium. Adds +20,000 bank capacity, 20% tax discount."},
	"duplex":         {ID: "duplex", Name: "Duplex", Type: AssetHome, Cost: 60000, Desc: "A cozy suburban duplex. Adds +35,000 bank capacity, 30% tax discount."},
	"mansion":        {ID: "mansion", Name: "Mansion", Type: AssetHome, Cost: 80000, Desc: "Adds +50,000 bank capacity, 40% tax discount."},
	"suburban_house": {ID: "suburban_house", Name: "Suburban House", Type: AssetHome, Cost: 150000, Desc: "A spacious 4-bedroom house. Adds +100,000 bank capacity, 50% tax discount."},
	"penthouse":      {ID: "penthouse", Name: "Luxury Penthouse", Type: AssetHome, Cost: 350000, Desc: "A penthouse overlooking the city. Adds +250,000 bank capacity, 60% tax discount."},
	"skyscraper":     {ID: "skyscraper", Name: "Skyscraper", Type: AssetHome, Cost: 2500000, Desc: "A downtown commercial high-rise. Adds +2,000,000 bank capacity, 80% tax discount."},
}

func assetsCmds(p *EconomyPlugin) []*manager.Command {
	return []*manager.Command{
		{
			Trigger:     "buyasset",
			Aliases:     []string{"buyproperty", "buybusiness"},
			Name:        "buyasset",
			Description: "Purchase real estate or businesses",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				gid := ctx.GuildID()
				if gid == "" {
					return ctx.Reply("Must be used in a server.")
				}
				cfg := getCfg(ctx.DB, gid)
				if !cfg.Enabled {
					return ctx.Reply("Economy is disabled.")
				}
				if len(ctx.Args) == 0 {
					return ctx.Reply("Usage: .buyasset <id> [qty]")
				}

				id := strings.ToLower(ctx.Args[0])
				asset, ok := assetsCatalog[id]
				if !ok {
					var available []string
					for k := range assetsCatalog {
						available = append(available, k)
					}
					return ctx.Reply("Asset not found. Available assets: " + strings.Join(available, ", "))
				}

				qty := 1
				if len(ctx.Args) > 1 {
					q, err := strconv.Atoi(ctx.Args[1])
					if err != nil || q <= 0 {
						return ctx.Reply("Quantity must be a positive integer.")
					}
					qty = q
				}

				uid := ctx.AuthorID()
				a := getAcct(ctx.DB, gid, uid)

				inflation := getInflationIndex(ctx.DB, gid)
				price := int64(math.Ceil(float64(asset.Cost) * inflation))

				if price > 0 && int64(qty) > 9223372036854775807/price {
					return ctx.Reply("Quantity is too large.")
				}

				total := price * int64(qty)
				if a.Wallet < total {
					return ctx.Reply(fmt.Sprintf("Insufficient wallet balance. Total cost: %s", fmtCoins(total, cfg)))
				}

				a.Wallet -= total
				switch asset.Type {
				case AssetBusiness:
					if a.Businesses == nil {
						a.Businesses = make(map[string]int)
					}
					a.Businesses[id] += qty
				case AssetLand:
					if a.Land == nil {
						a.Land = make(map[string]int)
					}
					a.Land[id] += qty
				case AssetHome:
					if a.Homes == nil {
						a.Homes = make(map[string]int)
					}
					a.Homes[id] += qty
				}

				// If businesses owned and LastCollect is zero, align to now so they don't get 24h instant passive income.
				if asset.Type == AssetBusiness && a.LastCollect.IsZero() {
					a.LastCollect = time.Now()
				}

				_ = saveAcct(ctx.DB, gid, uid, a)

				return ctx.Reply(fmt.Sprintf("Successfully bought %dx %s for %s.\nNew Wallet Balance: %s",
					qty, asset.Name, fmtCoins(total, cfg), fmtCoins(a.Wallet, cfg)))
			},
		},
		{
			Trigger:     "sellasset",
			Aliases:     []string{"sellproperty", "sellbusiness"},
			Name:        "sellasset",
			Description: "Sell back real estate or businesses",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				gid := ctx.GuildID()
				if gid == "" {
					return ctx.Reply("Must be used in a server.")
				}
				cfg := getCfg(ctx.DB, gid)
				if !cfg.Enabled {
					return ctx.Reply("Economy is disabled.")
				}
				if len(ctx.Args) == 0 {
					return ctx.Reply("Usage: .sellasset <id> [qty]")
				}

				id := strings.ToLower(ctx.Args[0])
				asset, ok := assetsCatalog[id]
				if !ok {
					return ctx.Reply("Asset not found.")
				}

				qty := 1
				if len(ctx.Args) > 1 {
					q, err := strconv.Atoi(ctx.Args[1])
					if err != nil || q <= 0 {
						return ctx.Reply("Quantity must be a positive integer.")
					}
					qty = q
				}

				uid := ctx.AuthorID()
				a := getAcct(ctx.DB, gid, uid)

				owned := 0
				switch asset.Type {
				case AssetBusiness:
					owned = a.Businesses[id]
				case AssetLand:
					owned = a.Land[id]
				case AssetHome:
					owned = a.Homes[id]
				}

				if owned < qty {
					return ctx.Reply(fmt.Sprintf("You only own %dx %s.", owned, asset.Name))
				}

				inflation := getInflationIndex(ctx.DB, gid)
				mult := 0.50
				if asset.Type == AssetLand {
					mult = 0.70
				}

				price := int64(math.Floor(float64(asset.Cost) * inflation * mult))
				payout := price * int64(qty)

				a.Wallet += payout
				switch asset.Type {
				case AssetBusiness:
					a.Businesses[id] -= qty
					if a.Businesses[id] <= 0 {
						delete(a.Businesses, id)
					}
				case AssetLand:
					a.Land[id] -= qty
					if a.Land[id] <= 0 {
						delete(a.Land, id)
					}
				case AssetHome:
					a.Homes[id] -= qty
					if a.Homes[id] <= 0 {
						delete(a.Homes, id)
					}
				}

				_ = saveAcct(ctx.DB, gid, uid, a)

				return ctx.Reply(fmt.Sprintf("Sold %dx %s for %s.\nNew Wallet Balance: %s",
					qty, asset.Name, fmtCoins(payout, cfg), fmtCoins(a.Wallet, cfg)))
			},
		},
		{
			Trigger:     "business",
			Aliases:     []string{"assets", "businesses", "properties"},
			Name:        "business",
			Description: "View or collect passive business income",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				gid := ctx.GuildID()
				if gid == "" {
					return ctx.Reply("Must be used in a server.")
				}
				cfg := getCfg(ctx.DB, gid)
				if !cfg.Enabled {
					return ctx.Reply("Economy is disabled.")
				}

				uid := ctx.AuthorID()
				a := getAcct(ctx.DB, gid, uid)

				collect := false
				if len(ctx.Args) > 0 && strings.EqualFold(ctx.Args[0], "collect") {
					collect = true
				}

				// Calculate business stats
				totalHourlyYield := int64(0)
				for bid, qty := range a.Businesses {
					if asset, ok := assetsCatalog[bid]; ok {
						totalHourlyYield += int64(qty) * asset.Yield
					}
				}

				hours := 0.0
				if !a.LastCollect.IsZero() {
					hours = time.Since(a.LastCollect).Hours()
				}
				if hours > 24.0 {
					hours = 24.0
				}
				if hours < 0 {
					hours = 0
				}

				inflation := getInflationIndex(ctx.DB, gid)

				if collect {
					if totalHourlyYield == 0 {
						return ctx.Reply("You do not own any income generating businesses.")
					}
					accrued := float64(totalHourlyYield) * hours
					payout := int64(math.Floor(accrued * math.Sqrt(inflation)))

					if payout <= 0 {
						return ctx.Reply("No income has accumulated yet. Wait a bit longer.")
					}

					a.Wallet += payout
					a.LastCollect = time.Now()
					_ = saveAcct(ctx.DB, gid, uid, a)

					emb := config.Build(ctx.Cfg, config.EmbedOpt{
						Title: "Business Collection",
						Description: fmt.Sprintf("Accumulated: %.2f hours (24h cap)\nHourly Yield: %s/hour (Base)\nInflation Factor: x%.2f\n\nTotal Collected: %s\nNew Wallet Balance: %s",
							hours, fmtCoins(totalHourlyYield, cfg), math.Sqrt(inflation), fmtCoins(payout, cfg), fmtCoins(a.Wallet, cfg)),
					})
					return ctx.Respond(emb)
				}

				// Otherwise, view asset status
				var fields []*discordgo.MessageEmbedField

				// Businesses
				var bizLines []string
				for bid, qty := range a.Businesses {
					if asset, ok := assetsCatalog[bid]; ok {
						bizLines = append(bizLines, fmt.Sprintf("- %s: %dx (Yield: %s/hour total)", asset.Name, qty, fmtCoins(int64(qty)*asset.Yield, cfg)))
					}
				}
				if len(bizLines) > 0 {
					fields = append(fields, config.Field("Businesses", strings.Join(bizLines, "\n"), false))
				}

				// Real Estate & Land
				var propLines []string
				for pid, qty := range a.Homes {
					if asset, ok := assetsCatalog[pid]; ok {
						propLines = append(propLines, fmt.Sprintf("- %s: %dx (%s)", asset.Name, qty, asset.Desc))
					}
				}
				for lid, qty := range a.Land {
					if asset, ok := assetsCatalog[lid]; ok {
						propLines = append(propLines, fmt.Sprintf("- %s: %dx (%s)", asset.Name, qty, asset.Desc))
					}
				}
				if len(propLines) > 0 {
					fields = append(fields, config.Field("Properties & Land", strings.Join(propLines, "\n"), false))
				}

				// Pending Passive Income Info
				pending := int64(0)
				if totalHourlyYield > 0 {
					pending = int64(math.Floor(float64(totalHourlyYield) * hours * math.Sqrt(inflation)))
				}
				statusDesc := fmt.Sprintf("Pending Income: %s\nAccumulated: %.2f / 24.00 hours\nHourly Income: %s/hour",
					fmtCoins(pending, cfg), hours, fmtCoins(int64(math.Floor(float64(totalHourlyYield)*math.Sqrt(inflation))), cfg))

				fields = append(fields, config.Field("Passive Income Status", statusDesc, false))

				emb := config.Build(ctx.Cfg, config.EmbedOpt{
					Title:       fmt.Sprintf("%s's Investment Portfolio", ctx.AuthorTag()),
					Description: "Invest in businesses or properties with .buyasset <id>.\nCollect passive business yield with .business collect.",
					Fields:      fields,
				})
				return ctx.Respond(emb)
			},
		},
	}
}
