package economy

import (
	bolt "go.etcd.io/bbolt"
	"skyvern/internal/manager"
	"skyvern/internal/plugins"
	"skyvern/internal/storage"
)

func init() {
	plugins.Register(&EconomyPlugin{})

	manager.RegisterHelp("balance", []manager.HelpPage{
		{
			Command:     "View Balance",
			Syntax:      ".balance [@user]",
			Description: "View wallet, bank, net worth, and level/XP.",
		},
	})
	manager.RegisterHelp("deposit", []manager.HelpPage{
		{
			Command:     "Deposit Coins",
			Syntax:      ".deposit <amount|all>",
			Description: "Move coins from your wallet to your bank.",
		},
	})
	manager.RegisterHelp("withdraw", []manager.HelpPage{
		{
			Command:     "Withdraw Coins",
			Syntax:      ".withdraw <amount|all>",
			Description: "Move coins from your bank to your wallet.",
		},
	})
	manager.RegisterHelp("pay", []manager.HelpPage{
		{
			Command:     "Pay User",
			Syntax:      ".pay <@user> <amount>",
			Description: "Transfer coins from your wallet to another user.",
		},
	})
	manager.RegisterHelp("daily", []manager.HelpPage{
		{
			Command:     "Daily Reward",
			Syntax:      ".daily",
			Description: "Claim your daily coin reward (24h cooldown) with streak bonuses.",
		},
	})
	manager.RegisterHelp("work", []manager.HelpPage{
		{
			Command:     "Work Job",
			Syntax:      ".work",
			Description: "Work a random job for coins (30m cooldown).",
		},
	})
	manager.RegisterHelp("beg", []manager.HelpPage{
		{
			Command:     "Beg Spare Change",
			Syntax:      ".beg",
			Description: "Beg for coins on the street (30s cooldown).",
		},
	})
	manager.RegisterHelp("crime", []manager.HelpPage{
		{
			Command:     "Commit Crime",
			Syntax:      ".crime",
			Description: "Attempt a high-risk crime to steal coins (1h cooldown).",
		},
	})
	manager.RegisterHelp("rob", []manager.HelpPage{
		{
			Command:     "Rob User",
			Syntax:      ".rob <@user>",
			Description: "Attempt to rob coins from another user's wallet (2h cooldown).",
		},
	})
	manager.RegisterHelp("fish", []manager.HelpPage{
		{
			Command:     "Go Fishing",
			Syntax:      ".fish",
			Description: "Go fishing to catch items or coins (30m cooldown).",
		},
	})
	manager.RegisterHelp("hunt", []manager.HelpPage{
		{
			Command:     "Go Hunting",
			Syntax:      ".hunt",
			Description: "Go hunting to catch wild animals or coins (30m cooldown).",
		},
	})
	manager.RegisterHelp("mine", []manager.HelpPage{
		{
			Command:     "Go Mining",
			Syntax:      ".mine",
			Description: "Go mining for ores, gems, or coins (30m cooldown).",
		},
	})
	manager.RegisterHelp("slots", []manager.HelpPage{
		{
			Command:     "Slot Machine",
			Syntax:      ".slots <bet>",
			Description: "Spin the slot machine to win payouts.",
		},
	})
	manager.RegisterHelp("coinflipbet", []manager.HelpPage{
		{
			Command:     "Coinflip Bet",
			Syntax:      ".cf <heads|tails> <bet>",
			Description: "Bet on heads or tails for a 2x payout.",
		},
	})
	manager.RegisterHelp("blackjack", []manager.HelpPage{
		{
			Command:     "Play Blackjack",
			Syntax:      ".blackjack <bet>",
			Description: "Play blackjack against the dealer with interactive buttons.",
		},
	})
	manager.RegisterHelp("dice", []manager.HelpPage{
		{
			Command:     "Dice Bet",
			Syntax:      ".dice <guess: 1-6> <bet>",
			Description: "Bet on a 6-sided die roll for a 5x payout.",
		},
	})
	manager.RegisterHelp("roulette", []manager.HelpPage{
		{
			Command:     "Roulette Bet",
			Syntax:      ".roulette <bet> <red|black|green|0-36>",
			Description: "Bet on roulette outcomes for up to 35x payout.",
		},
	})
	manager.RegisterHelp("highlow", []manager.HelpPage{
		{
			Command:     "HighLow Bet",
			Syntax:      ".highlow <bet>",
			Description: "Guess if the next number will be higher or lower.",
		},
	})
	manager.RegisterHelp("shop", []manager.HelpPage{
		{
			Command:     "View Shop",
			Syntax:      ".shop",
			Description: "View custom server items listed in the shop.",
		},
	})
	manager.RegisterHelp("buy", []manager.HelpPage{
		{
			Command:     "Buy Item",
			Syntax:      ".buy <item_id> [quantity]",
			Description: "Purchase custom items from the server shop.",
		},
	})
	manager.RegisterHelp("inventory", []manager.HelpPage{
		{
			Command:     "View Inventory",
			Syntax:      ".inventory [@user]",
			Description: "View items in your or another user's inventory.",
		},
	})
	manager.RegisterHelp("sell", []manager.HelpPage{
		{
			Command:     "Sell Item",
			Syntax:      ".sell <item_id> [quantity]",
			Description: "Sell back items for 50% value (custom) or fixed value (default).",
		},
	})
	manager.RegisterHelp("use", []manager.HelpPage{
		{
			Command:     "Use Item",
			Syntax:      ".use <item_id>",
			Description: "Use a consumable item from your inventory.",
		},
	})
	manager.RegisterHelp("richest", []manager.HelpPage{
		{
			Command:     "Rich Leaderboard",
			Syntax:      ".richest [page]",
			Description: "View the server leaderboard sorted by net worth.",
		},
	})
	manager.RegisterHelp("networth", []manager.HelpPage{
		{
			Command:     "Net Worth Breakdown",
			Syntax:      ".networth [@user]",
			Description: "Show asset details (wallet, bank, inventory, stocks).",
		},
	})
	manager.RegisterHelp("stock", []manager.HelpPage{
		{
			Command:     "Stock Watchlist",
			Syntax:      ".stock [list]",
			Description: "View live market prices for major stocks.",
		},
		{
			Command:     "View Stock Chart",
			Syntax:      ".stock view <symbol>",
			Description: "View 1d chart and position details for a stock.",
		},
		{
			Command:     "Buy Shares",
			Syntax:      ".stock buy <symbol> <shares>",
			Description: "Purchase shares using your wallet balance.",
		},
		{
			Command:     "Sell Shares",
			Syntax:      ".stock sell <symbol> <shares>",
			Description: "Sell owned shares back to the market.",
		},
		{
			Command:     "Stock Portfolio",
			Syntax:      ".stock portfolio",
			Description: "Show your total shares and valuation.",
		},
	})
	manager.RegisterHelp("eco", []manager.HelpPage{
		{
			Command:     "Economy Admin",
			Syntax:      ".eco <set|add|remove|reset|resetall|symbol|name|toggle> [args]",
			Description: "Administrative configurations for the economy plugin.",
		},
	})
	manager.RegisterHelp("shopadd", []manager.HelpPage{
		{
			Command:     "Shop Add",
			Syntax:      ".shopadd <id> <price> <name> [desc] [--role <@role>]",
			Description: "Add a custom item (optionally linked to a role) to the shop.",
		},
	})
	manager.RegisterHelp("shopremove", []manager.HelpPage{
		{
			Command:     "Shop Remove",
			Syntax:      ".shopremove <id>",
			Description: "Remove a custom item from the server shop.",
		},
	})
	manager.RegisterHelp("shopedit", []manager.HelpPage{
		{
			Command:     "Shop Edit",
			Syntax:      ".shopedit <id> <price|desc|stock> <value>",
			Description: "Edit fields of an existing custom shop item.",
		},
	})
	manager.RegisterHelp("waifu", []manager.HelpPage{
		{
			Command:     "Waifu Roll",
			Syntax:      ".waifu roll (or .wroll)",
			Description: "Roll a random waifu character.",
		},
		{
			Command:     "Waifu Collection",
			Syntax:      ".waifu list (or .wlist) [@user]",
			Description: "View your or another user's collected waifus.",
		},
		{
			Command:     "Sell Waifu",
			Syntax:      ".waifu sell <character_name>",
			Description: "Sell a duplicate waifu for coins.",
		},
	})
	manager.RegisterHelp("lottery", []manager.HelpPage{
		{
			Command:     "Buy Tickets",
			Syntax:      ".lottery buy [quantity]",
			Description: "Buy lottery tickets (100 coins each).",
		},
		{
			Command:     "Lottery Status",
			Syntax:      ".lottery status",
			Description: "Show the current pot size and your ticket count.",
		},
	})
	manager.RegisterHelp("assets", []manager.HelpPage{
		{
			Command:     "Buy Real Estate/Business",
			Syntax:      ".buyasset <id> [qty]",
			Description: "Invest in businesses, plots of land, or homes.",
		},
		{
			Command:     "Sell Real Estate/Business",
			Syntax:      ".sellasset <id> [qty]",
			Description: "Sell back your owned investments.",
		},
		{
			Command:     "Business Collection",
			Syntax:      ".business [collect] (or .assets)",
			Description: "Collect passive income accumulated from businesses (capped at 24h).",
		},
	})
}

type EconomyPlugin struct {
	db  *storage.DB
	mgr *manager.Manager
}

func (p *EconomyPlugin) Name() string {
	return "economy"
}

func (p *EconomyPlugin) Init(db *storage.DB, mgr *manager.Manager) error {
	p.db = db
	p.mgr = mgr

	if err := db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(bktEcoAccts); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(bktEcoCfg); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(bktEcoShop); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	// Register interactive component handlers
	mgr.RegisterComponentHandler("bj:*", p.handleBJInteraction)
	mgr.RegisterComponentHandler("hl:*", p.handleHLInteraction)
	mgr.RegisterComponentHandler("shop:*", p.handleShopInteraction)

	return nil
}

func (p *EconomyPlugin) Commands() []*manager.Command {
	var cmds []*manager.Command
	cmds = append(cmds, currencyCmds(p)...)
	cmds = append(cmds, incomeCmds(p)...)
	cmds = append(cmds, gatheringCmds(p)...)
	cmds = append(cmds, gamblingCmds(p)...)
	cmds = append(cmds, shopCmds(p)...)
	cmds = append(cmds, leaderboardCmds(p)...)
	cmds = append(cmds, adminCmds(p)...)
	cmds = append(cmds, stockCmds(p)...)
	cmds = append(cmds, waifuCmds(p)...)
	cmds = append(cmds, lotteryCmds(p)...)
	cmds = append(cmds, assetsCmds(p)...)
	return cmds
}
