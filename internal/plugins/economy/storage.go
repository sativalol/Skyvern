package economy

import (
	"bytes"
	"encoding/json"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"skyvern/internal/storage"
	"time"
)

var (
	bktEcoAccts = []byte("EcoAccounts")
	bktEcoCfg   = []byte("EcoCfg")
	bktEcoShop  = []byte("EcoShop")
)

type EcoAccount struct {
	Wallet    int64              `json:"wallet"`
	Bank      int64              `json:"bank"`
	BankMax   int64              `json:"bank_max"` // starts at 10000, grows with prestige/level
	LastDaily time.Time          `json:"last_daily"`
	LastWork  time.Time          `json:"last_work"`
	LastCrime time.Time          `json:"last_crime"`
	LastRob   time.Time          `json:"last_rob"`
	LastBeg   time.Time          `json:"last_beg"`
	LastFish  time.Time          `json:"last_fish"`
	LastHunt  time.Time          `json:"last_hunt"`
	LastMine  time.Time          `json:"last_mine"`
	Streak    int                `json:"streak"`
	Inventory []InvItem          `json:"inv"`
	XP        int64              `json:"xp"`
	Level     int                `json:"level"`
	Stocks      map[string]float64 `json:"stocks,omitempty"` // symbol -> shares
	Waifus      map[string]int     `json:"waifus,omitempty"` // waifu name -> count
	Land        map[string]int     `json:"land,omitempty"`        // plot -> count
	Businesses  map[string]int     `json:"businesses,omitempty"`  // business -> count
	Homes       map[string]int     `json:"homes,omitempty"`       // home -> count
	LastCollect time.Time          `json:"last_collect,omitempty"`
}

type InvItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Qty  int    `json:"qty"`
}

type EcoCfg struct {
	Enabled           bool               `json:"enabled"`
	Symbol            string             `json:"symbol"`        // default: "$"
	CurrencyName      string             `json:"currency_name"` // default: "coins"
	DailyMin          int64              `json:"daily_min"`
	DailyMax          int64              `json:"daily_max"`
	WorkMin           int64              `json:"work_min"`
	WorkMax           int64              `json:"work_max"`
	CrimeMin          int64              `json:"crime_min"`
	CrimeMax          int64              `json:"crime_max"`
	CrimeFailPct      int                `json:"crime_fail_pct"`
	RobFailPct        int                `json:"rob_fail_pct"`
	StartBal          int64              `json:"start_bal"`
	ItemSoldSupply    map[string]float64 `json:"item_sold_supply,omitempty"`
	LastDecay         time.Time          `json:"last_decay,omitempty"`
	CurrentWeather    string             `json:"current_weather,omitempty"`
	LastWeatherUpdate time.Time          `json:"last_weather_update,omitempty"`
	LotteryTickets    map[string]int     `json:"lottery_tickets,omitempty"` // user_id -> ticket_count
	LotteryPot        int64              `json:"lottery_pot,omitempty"`
}

type ShopItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"desc"`
	Price       int64  `json:"price"`
	RoleID      string `json:"role_id,omitempty"` // if it grants a role
	Stock       int    `json:"stock"`             // -1 = unlimited
}

func ecoKey(gid, uid string) []byte {
	return []byte(gid + ":" + uid)
}

func getCfg(db *storage.DB, gid string) EcoCfg {
	var c EcoCfg
	found := false
	_ = db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktEcoCfg)
		if bkt == nil {
			return nil
		}
		v := bkt.Get([]byte(gid))
		if v == nil {
			return nil
		}
		found = true
		return json.Unmarshal(v, &c)
	})
	if !found {
		c.Enabled = true
	}
	// defaults if uninitialized
	if c.Symbol == "" {
		c.Symbol = "$"
	}
	if c.CurrencyName == "" {
		c.CurrencyName = "coins"
	}
	if c.ItemSoldSupply == nil {
		c.ItemSoldSupply = make(map[string]float64)
	}
	if c.CurrentWeather == "" {
		c.CurrentWeather = "Sunny"
	}
	if c.LastWeatherUpdate.IsZero() {
		c.LastWeatherUpdate = time.Now()
	}
	if c.LastDecay.IsZero() {
		c.LastDecay = time.Now()
	}
	if c.LotteryTickets == nil {
		c.LotteryTickets = make(map[string]int)
	}
	if c.DailyMin == 0 {
		c.DailyMin = 100
		c.DailyMax = 500
	}
	if c.WorkMin == 0 {
		c.WorkMin = 50
		c.WorkMax = 200
	}
	if c.CrimeMin == 0 {
		c.CrimeMin = 100
		c.CrimeMax = 1000
		c.CrimeFailPct = 40
	}
	if c.RobFailPct == 0 {
		c.RobFailPct = 50
	}
	return c
}

func saveCfg(db *storage.DB, gid string, cfg EcoCfg) error {
	return db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktEcoCfg)
		if bkt == nil {
			return fmt.Errorf("cfg bucket missing")
		}
		b, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		return bkt.Put([]byte(gid), b)
	})
}

func getAcct(db *storage.DB, gid, uid string) EcoAccount {
	var a EcoAccount
	_ = db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktEcoAccts)
		if bkt == nil {
			return nil
		}
		v := bkt.Get(ecoKey(gid, uid))
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &a)
	})
	if a.Stocks == nil {
		a.Stocks = make(map[string]float64)
	}
	if a.Waifus == nil {
		a.Waifus = make(map[string]int)
	}
	if a.Land == nil {
		a.Land = make(map[string]int)
	}
	if a.Businesses == nil {
		a.Businesses = make(map[string]int)
	}
	if a.Homes == nil {
		a.Homes = make(map[string]int)
	}
	if a.LastCollect.IsZero() {
		a.LastCollect = time.Now()
	}
	var bonus int64
	for hType, qty := range a.Homes {
		if prop, ok := HomeProperties[hType]; ok {
			bonus += int64(qty) * prop.CapacityBonus
		}
	}
	a.BankMax = 10000 + int64(a.Level)*5000 + bonus
	return a
}

func saveAcct(db *storage.DB, gid, uid string, a EcoAccount) error {
	return db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktEcoAccts)
		if bkt == nil {
			return fmt.Errorf("acct bucket missing")
		}
		b, err := json.Marshal(a)
		if err != nil {
			return err
		}
		return bkt.Put(ecoKey(gid, uid), b)
	})
}

func getShop(db *storage.DB, gid string) []ShopItem {
	var items []ShopItem
	_ = db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktEcoShop)
		if bkt == nil {
			return nil
		}
		// iterate through all items for this guild
		c := bkt.Cursor()
		prefix := []byte(gid + ":")
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			var it ShopItem
			if err := json.Unmarshal(v, &it); err == nil {
				items = append(items, it)
			}
		}
		return nil
	})
	return items
}

func getShopItem(db *storage.DB, gid, itemID string) (ShopItem, bool) {
	var it ShopItem
	found := false
	_ = db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktEcoShop)
		if bkt == nil {
			return nil
		}
		v := bkt.Get([]byte(gid + ":" + itemID))
		if v == nil {
			return nil
		}
		if err := json.Unmarshal(v, &it); err == nil {
			found = true
		}
		return nil
	})
	return it, found
}

func saveShopItem(db *storage.DB, gid string, it ShopItem) error {
	return db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktEcoShop)
		if bkt == nil {
			return fmt.Errorf("shop bucket missing")
		}
		b, err := json.Marshal(it)
		if err != nil {
			return err
		}
		return bkt.Put([]byte(gid+":"+it.ID), b)
	})
}

func deleteShopItem(db *storage.DB, gid, itemID string) error {
	return db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktEcoShop)
		if bkt == nil {
			return fmt.Errorf("shop bucket missing")
		}
		return bkt.Delete([]byte(gid + ":" + itemID))
	})
}
