package vouch
import (
	"encoding/json"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"skyvern/internal/storage"
)
var bktVouchCfg = []byte("VouchPluginCfg")
type VouchCfg struct {
	AltCheckEnabled bool `json:"alt_check"`
	AltCheckRequire bool `json:"alt_require"`                                                   
	AccountAgeDays  int  `json:"acct_age_days"`           
}
func getVouchCfg(db *storage.DB, gid string) VouchCfg {
	var cfg VouchCfg
	_ = db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktVouchCfg)
		if bkt == nil {
			return nil
		}
		v := bkt.Get([]byte(gid))
		if v == nil {
			return fmt.Errorf("not found")
		}
		return json.Unmarshal(v, &cfg)
	})
	return cfg
}
func saveVouchCfg(db *storage.DB, gid string, cfg VouchCfg) error {
	return db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktVouchCfg)
		if bkt == nil {
			return fmt.Errorf("bucket missing")
		}
		b, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		return bkt.Put([]byte(gid), b)
	})
}