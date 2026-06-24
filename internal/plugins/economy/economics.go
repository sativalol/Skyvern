package economy

import (
	"bytes"
	"encoding/json"
	"math"
	"math/rand"
	"time"

	bolt "go.etcd.io/bbolt"
	"skyvern/internal/storage"
)

func countUsers(db *storage.DB, gid string) int {
	count := 0
	_ = db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktEcoAccts)
		if bkt == nil {
			return nil
		}
		c := bkt.Cursor()
		prefix := []byte(gid + ":")
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			count++
		}
		return nil
	})
	if count == 0 {
		return 1
	}
	return count
}

func getTotalSupply(db *storage.DB, gid string) (int64, int) {
	var total int64
	numUsers := 0
	stockSyms := make(map[string]bool)
	sharesMap := make(map[string]float64)

	_ = db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktEcoAccts)
		if bkt == nil {
			return nil
		}
		c := bkt.Cursor()
		prefix := []byte(gid + ":")
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			var a EcoAccount
			if err := json.Unmarshal(v, &a); err == nil {
				total += a.Wallet + a.Bank
				numUsers++
				for sym, sh := range a.Stocks {
					if sh > 0 {
						stockSyms[sym] = true
						sharesMap[sym] += sh
					}
				}
			}
		}
		return nil
	})

	if numUsers == 0 {
		return 0, 1
	}

	// Fetch stock prices to add to total supply
	if len(stockSyms) > 0 {
		var syms []string
		for sym := range stockSyms {
			syms = append(syms, sym)
		}
		if q, err := getQuoteWithRetry(syms); err == nil {
			for _, item := range q.QuoteResponse.Result {
				sh := sharesMap[item.Symbol]
				total += int64(sh * item.RegularMarketPrice)
			}
		}
	}

	return total, numUsers
}

func getInflationIndex(db *storage.DB, gid string) float64 {
	supply, numUsers := getTotalSupply(db, gid)
	avg := float64(supply) / float64(numUsers)
	// Base is 10k coins average balance. Bounded between 0.5 and 100.
	idx := avg / 10000.0
	if idx < 0.5 {
		return 0.5
	}
	if idx > 100.0 {
		return 100.0
	}
	return idx
}

func getDynamicPrice(db *storage.DB, gid string, itemID string, baseSell int64) int64 {
	cfg := getCfg(db, gid)

	// Time-based decay of recently sold supply (5% decay per hour)
	now := time.Now()
	elapsed := now.Sub(cfg.LastDecay)
	if elapsed > 0 {
		decay := math.Exp(-elapsed.Hours() * 0.05)
		for k, v := range cfg.ItemSoldSupply {
			nv := v * decay
			if nv < 0.01 {
				delete(cfg.ItemSoldSupply, k)
			} else {
				cfg.ItemSoldSupply[k] = nv
			}
		}
		cfg.LastDecay = now
		_ = saveCfg(db, gid, cfg)
	}

	supply := cfg.ItemSoldSupply[itemID]
	numUsers := countUsers(db, gid)

	// Price drops with supply: Bounded between 0.3 (oversupply) and 1.5 (scarcity)
	priceMult := 1.5 - (supply / (float64(numUsers)*10.0 + 5.0))
	if priceMult < 0.3 {
		priceMult = 0.3
	}
	if priceMult > 1.5 {
		priceMult = 1.5
	}

	inflation := getInflationIndex(db, gid)
	finalPrice := float64(baseSell) * priceMult * inflation

	if finalPrice < 1 {
		return 1
	}
	if finalPrice > 1e9 {
		return 1e9
	}
	return int64(finalPrice)
}

func updateSoldSupply(db *storage.DB, gid string, itemID string, qty int) {
	cfg := getCfg(db, gid)
	if cfg.ItemSoldSupply == nil {
		cfg.ItemSoldSupply = make(map[string]float64)
	}
	cfg.ItemSoldSupply[itemID] += float64(qty)
	_ = saveCfg(db, gid, cfg)
}

func updateWeather(db *storage.DB, gid string) string {
	cfg := getCfg(db, gid)
	now := time.Now()
	if now.Sub(cfg.LastWeatherUpdate) >= 2*time.Hour {
		r := rand.Float32()
		var w string
		if r < 0.35 {
			w = "Sunny"
		} else if r < 0.60 {
			w = "Rainy"
		} else if r < 0.80 {
			w = "Cloudy"
		} else if r < 0.95 {
			w = "Foggy"
		} else {
			w = "Stormy"
		}
		cfg.CurrentWeather = w
		cfg.LastWeatherUpdate = now
		_ = saveCfg(db, gid, cfg)
	}
	return cfg.CurrentWeather
}
