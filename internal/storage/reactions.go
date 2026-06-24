package storage
import (
	"encoding/json"
	"strings"
	bolt "go.etcd.io/bbolt"
)
type NoSelfReactCfg struct {
	Enabled      bool            `json:"enabled"`
	BypassAdmins bool            `json:"bypass_admins"`
	Punishment   string          `json:"punishment"`
	Exempts      map[string]bool `json:"exempts"`                                 
	Emojis       map[string]bool `json:"emojis"`                                   
}
func (d *DB) SaveReactionTrigger(gid, trigger, emoji, authorID string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		k := []byte(gid + ":" + strings.ToLower(trigger) + ":" + emoji)
		return tx.Bucket(bktReactionTriggers).Put(k, []byte(authorID))
	})
}
func (d *DB) DeleteReactionTrigger(gid, trigger, emoji string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		k := []byte(gid + ":" + strings.ToLower(trigger) + ":" + emoji)
		return tx.Bucket(bktReactionTriggers).Delete(k)
	})
}
func (d *DB) DeleteAllReactionTriggers(gid, trigger string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktReactionTriggers)
		prefix := []byte(gid + ":" + strings.ToLower(trigger) + ":")
		c := bkt.Cursor()
		var keys [][]byte
		for k, _ := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, _ = c.Next() {
			buf := make([]byte, len(k))
			copy(buf, k)
			keys = append(keys, buf)
		}
		for _, k := range keys {
			if err := bkt.Delete(k); err != nil {
				return err
			}
		}
		return nil
	})
}
func (d *DB) ClearReactionTriggers(gid string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktReactionTriggers)
		prefix := []byte(gid + ":")
		c := bkt.Cursor()
		var keys [][]byte
		for k, _ := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, _ = c.Next() {
			buf := make([]byte, len(k))
			copy(buf, k)
			keys = append(keys, buf)
		}
		for _, k := range keys {
			if err := bkt.Delete(k); err != nil {
				return err
			}
		}
		return nil
	})
}
func (d *DB) ListReactionTriggers(gid string) (map[string][]string, error) {
	out := make(map[string][]string)
	err := d.b.View(func(tx *bolt.Tx) error {
		prefix := []byte(gid + ":")
		c := tx.Bucket(bktReactionTriggers).Cursor()
		for k, _ := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, _ = c.Next() {
			parts := strings.Split(string(k), ":")
			if len(parts) >= 3 {
				trigger := parts[1]
				emoji := parts[2]
				out[trigger] = append(out[trigger], emoji)
			}
		}
		return nil
	})
	return out, err
}
func (d *DB) GetReactionTriggerOwner(gid, trigger string) (string, error) {
	owner := ""
	err := d.b.View(func(tx *bolt.Tx) error {
		prefix := []byte(gid + ":" + strings.ToLower(trigger) + ":")
		c := tx.Bucket(bktReactionTriggers).Cursor()
		k, v := c.Seek(prefix)
		if k != nil && strings.HasPrefix(string(k), string(prefix)) {
			owner = string(v)
		}
		return nil
	})
	return owner, err
}
func (d *DB) SavePrevReactTrigger(gid, trigger, emoji, authorID string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		k := []byte(gid + ":" + strings.ToLower(trigger) + ":" + emoji)
		return tx.Bucket(bktPrevReactTriggers).Put(k, []byte(authorID))
	})
}
func (d *DB) DeletePrevReactTrigger(gid, trigger, emoji string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		k := []byte(gid + ":" + strings.ToLower(trigger) + ":" + emoji)
		return tx.Bucket(bktPrevReactTriggers).Delete(k)
	})
}
func (d *DB) DeleteAllPrevReactTriggers(gid, trigger string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktPrevReactTriggers)
		prefix := []byte(gid + ":" + strings.ToLower(trigger) + ":")
		c := bkt.Cursor()
		var keys [][]byte
		for k, _ := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, _ = c.Next() {
			buf := make([]byte, len(k))
			copy(buf, k)
			keys = append(keys, buf)
		}
		for _, k := range keys {
			if err := bkt.Delete(k); err != nil {
				return err
			}
		}
		return nil
	})
}
func (d *DB) ClearPrevReactTriggers(gid string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktPrevReactTriggers)
		prefix := []byte(gid + ":")
		c := bkt.Cursor()
		var keys [][]byte
		for k, _ := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, _ = c.Next() {
			buf := make([]byte, len(k))
			copy(buf, k)
			keys = append(keys, buf)
		}
		for _, k := range keys {
			if err := bkt.Delete(k); err != nil {
				return err
			}
		}
		return nil
	})
}
func (d *DB) ListPrevReactTriggers(gid string) (map[string][]string, error) {
	out := make(map[string][]string)
	err := d.b.View(func(tx *bolt.Tx) error {
		prefix := []byte(gid + ":")
		c := tx.Bucket(bktPrevReactTriggers).Cursor()
		for k, _ := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, _ = c.Next() {
			parts := strings.Split(string(k), ":")
			if len(parts) >= 3 {
				trigger := parts[1]
				emoji := parts[2]
				out[trigger] = append(out[trigger], emoji)
			}
		}
		return nil
	})
	return out, err
}
func (d *DB) GetPrevReactTriggerOwner(gid, trigger string) (string, error) {
	owner := ""
	err := d.b.View(func(tx *bolt.Tx) error {
		prefix := []byte(gid + ":" + strings.ToLower(trigger) + ":")
		c := tx.Bucket(bktPrevReactTriggers).Cursor()
		k, v := c.Seek(prefix)
		if k != nil && strings.HasPrefix(string(k), string(prefix)) {
			owner = string(v)
		}
		return nil
	})
	return owner, err
}
func (d *DB) SaveChannelAutoReacts(gid, cid string, emojis []string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		b, err := json.Marshal(emojis)
		if err != nil {
			return err
		}
		return tx.Bucket(bktMsgAutoReacts).Put([]byte(gid+":"+cid), b)
	})
}
func (d *DB) DeleteChannelAutoReacts(gid, cid string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bktMsgAutoReacts).Delete([]byte(gid + ":" + cid))
	})
}
func (d *DB) GetChannelAutoReacts(gid, cid string) ([]string, error) {
	var emojis []string
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktMsgAutoReacts).Get([]byte(gid + ":" + cid))
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &emojis)
	})
	return emojis, err
}
func (d *DB) ListChannelAutoReacts(gid string) (map[string][]string, error) {
	out := make(map[string][]string)
	err := d.b.View(func(tx *bolt.Tx) error {
		prefix := []byte(gid + ":")
		c := tx.Bucket(bktMsgAutoReacts).Cursor()
		for k, v := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, v = c.Next() {
			parts := strings.Split(string(k), ":")
			if len(parts) >= 2 {
				var emojis []string
				if err := json.Unmarshal(v, &emojis); err == nil {
					out[parts[1]] = emojis
				}
			}
		}
		return nil
	})
	return out, err
}
func (d *DB) GetNoSelfReactCfg(gid string) (NoSelfReactCfg, error) {
	cfg := NoSelfReactCfg{
		Enabled:    false,
		Punishment: "remove",
		Exempts:    make(map[string]bool),
		Emojis:     make(map[string]bool),
	}
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktNoSelfReact).Get([]byte(gid))
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &cfg)
	})
	if cfg.Exempts == nil {
		cfg.Exempts = make(map[string]bool)
	}
	if cfg.Emojis == nil {
		cfg.Emojis = make(map[string]bool)
	}
	return cfg, err
}
func (d *DB) SaveNoSelfReactCfg(gid string, cfg NoSelfReactCfg) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktNoSelfReact), []byte(gid), cfg)
	})
}