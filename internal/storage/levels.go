package storage
import (
	"encoding/json"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"strings"
)
type LevelsCfg struct {
	Enabled          bool     `json:"enabled"`
	MessageMode      string   `json:"message_mode"`                               
	MessageChan      string   `json:"message_chan"`
	Message          string   `json:"message"`
	Rate             float64  `json:"rate"`
	StackRoles       bool     `json:"stack_roles"`
	LeaderboardTitle string   `json:"leaderboard_title"`
	IgnoredChans     []string `json:"ignored_chans"`
	IgnoredRoles     []string `json:"ignored_roles"`
}
type UserXP struct {
	XP             int64 `json:"xp"`
	Level          int   `json:"level"`
	MessagesToggle bool  `json:"messages_toggle"`
}
type LevelRole struct {
	RoleID string `json:"role_id"`
	Level  int    `json:"level"`
}
func (d *DB) GetLevelsCfg(gid string) (LevelsCfg, error) {
	cfg := LevelsCfg{
		Enabled:          false,
		MessageMode:      "channel",
		Message:          "Congrats {user.mention}, you leveled up to level {level}!",
		Rate:             1.0,
		StackRoles:       true,
		LeaderboardTitle: "Server Leaderboard",
	}
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktLevelsCfg).Get([]byte(gid))
		if v == nil {
			return fmt.Errorf("not found")
		}
		return json.Unmarshal(v, &cfg)
	})
	return cfg, err
}
func (d *DB) SaveLevelsCfg(gid string, cfg LevelsCfg) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktLevelsCfg), []byte(gid), cfg)
	})
}
func (d *DB) GetUserXP(gid, uid string) (UserXP, error) {
	u := UserXP{
		MessagesToggle: true,
	}
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktLevelsXP).Get([]byte(gid + ":" + uid))
		if v == nil {
			return fmt.Errorf("not found")
		}
		return json.Unmarshal(v, &u)
	})
	return u, err
}
func (d *DB) SaveUserXP(gid, uid string, u UserXP) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktLevelsXP), []byte(gid+":"+uid), u)
	})
}
func (d *DB) ListLevelsXP(gid string) (map[string]UserXP, error) {
	out := make(map[string]UserXP)
	prefix := []byte(gid + ":")
	err := d.b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktLevelsXP)
		c := b.Cursor()
		for k, v := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, v = c.Next() {
			parts := strings.Split(string(k), ":")
			if len(parts) >= 2 {
				var u UserXP
				if err := json.Unmarshal(v, &u); err == nil {
					out[parts[1]] = u
				}
			}
		}
		return nil
	})
	return out, err
}
func (d *DB) ClearLevelsXP(gid string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktLevelsXP)
		c := b.Cursor()
		prefix := []byte(gid + ":")
		var keys [][]byte
		for k, _ := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, _ = c.Next() {
			buf := make([]byte, len(k))
			copy(buf, k)
			keys = append(keys, buf)
		}
		for _, k := range keys {
			if err := b.Delete(k); err != nil {
				return err
			}
		}
		return nil
	})
}
func (d *DB) SaveLevelRole(gid string, r LevelRole) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktLevelsRoles), []byte(gid+":"+r.RoleID), r)
	})
}
func (d *DB) DeleteLevelRole(gid, rid string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bktLevelsRoles).Delete([]byte(gid + ":" + rid))
	})
}
func (d *DB) ListLevelRoles(gid string) ([]LevelRole, error) {
	var out []LevelRole
	prefix := []byte(gid + ":")
	err := d.b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktLevelsRoles)
		c := b.Cursor()
		for k, v := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, v = c.Next() {
			var r LevelRole
			if err := json.Unmarshal(v, &r); err == nil {
				out = append(out, r)
			}
		}
		return nil
	})
	return out, err
}