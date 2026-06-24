package economy

import (
	"fmt"
	"strings"
	"time"
	"skyvern/internal/storage"
)

func fmtInt(n int64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return sign + s
	}
	var out []byte
	for i := len(s) - 1; i >= 0; i-- {
		out = append([]byte{s[i]}, out...)
		if (len(s)-i)%3 == 0 && i != 0 {
			out = append([]byte{','}, out...)
		}
	}
	return sign + string(out)
}

func fmtCoins(amount int64, cfg EcoCfg) string {
	return fmt.Sprintf("%s%s %s", cfg.Symbol, fmtInt(amount), cfg.CurrencyName)
}

func resolveUser(arg string) string {
	s := strings.TrimSpace(arg)
	if strings.HasPrefix(s, "<@") && strings.HasSuffix(s, ">") {
		s = strings.TrimPrefix(s, "<@")
		s = strings.TrimSuffix(s, ">")
		s = strings.TrimPrefix(s, "!")
		s = strings.TrimPrefix(s, "&") // handle role just in case, though we want user
		return s
	}
	// check if purely numerical ID
	if s == "" {
		return ""
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return ""
		}
	}
	return s
}

func getCooldown(last time.Time, dur time.Duration) (time.Duration, bool) {
	el := time.Since(last)
	if el >= dur {
		return 0, false
	}
	return dur - el, true
}

func addXP(db *storage.DB, gid, uid string, amount int64) (int, bool) {
	acct := getAcct(db, gid, uid)
	acct.XP += amount
	newLvl := acct.Level
	for {
		nextXP := int64((newLvl + 1) * (newLvl + 1) * 100)
		if acct.XP >= nextXP {
			newLvl++
		} else {
			break
		}
	}
	up := newLvl > acct.Level
	acct.Level = newLvl
	acct.BankMax = 10000 + int64(newLvl)*5000
	_ = saveAcct(db, gid, uid, acct)
	return newLvl, up
}
