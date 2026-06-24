package captcha
import (
	"encoding/json"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"skyvern/internal/storage"
)
var bktCaptchaCfg = []byte("CaptchaPluginCfg")
type CaptchaConfig struct {
	Enabled          bool   `json:"enabled"`
	VerifiedRoleID   string `json:"verified_role_id"`
	UnverifiedRoleID string `json:"unverified_role_id"`
	MaxAttempts      int    `json:"max_attempts"`                   
	FailureAction    string `json:"failure_action"`                                              
	TimeoutMinutes   int    `json:"timeout_minutes"`                
}
func getCaptchaCfg(db *storage.DB, gid string) CaptchaConfig {
	cfg := CaptchaConfig{
		MaxAttempts:    3,
		FailureAction:  "kick",
		TimeoutMinutes: 5,
	}
	_ = db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktCaptchaCfg)
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
func saveCaptchaCfg(db *storage.DB, gid string, cfg CaptchaConfig) error {
	return db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktCaptchaCfg)
		if bkt == nil {
			var err error
			bkt, err = tx.CreateBucketIfNotExists(bktCaptchaCfg)
			if err != nil {
				return err
			}
		}
		b, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		return bkt.Put([]byte(gid), b)
	})
}