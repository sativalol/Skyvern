package storage
import (
	"encoding/json"
	"fmt"
	"time"
	bolt "go.etcd.io/bbolt"
)
type RoleBackup struct {
	Name        string `json:"name"`
	Color       int    `json:"color"`
	Hoist       bool   `json:"hoist"`
	Mentionable bool   `json:"mentionable"`
	Permissions int64  `json:"permissions"`
	Position    int    `json:"position"`
}
type OverwriteBackup struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  int    `json:"type"`                        
	Allow int64  `json:"allow"`
	Deny  int64  `json:"deny"`
}
type ChannelBackup struct {
	ID                   string            `json:"id"`
	Name                 string            `json:"name"`
	Type                 int               `json:"type"`
	Topic                string            `json:"topic,omitempty"`
	Bitrate              int               `json:"bitrate,omitempty"`
	UserLimit            int               `json:"user_limit,omitempty"`
	ParentID             string            `json:"parent_id,omitempty"`
	Position             int               `json:"position"`
	NSFW                 bool              `json:"nsfw,omitempty"`
	RateLimitPerUser     int               `json:"rate_limit_per_user,omitempty"`
	PermissionOverwrites []OverwriteBackup `json:"permission_overwrites,omitempty"`
}
type ServerBackup struct {
	GuildID   string          `json:"guild_id"`
	Name      string          `json:"name"`
	CreatedAt time.Time       `json:"created_at"`
	Roles     []RoleBackup    `json:"roles"`
	Channels  []ChannelBackup `json:"channels"`
}
func (d *DB) SaveBackup(bid string, bk ServerBackup) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktBackups)
		v, err := json.Marshal(bk)
		if err != nil {
			return err
		}
		return bkt.Put([]byte(bid), v)
	})
}
func (d *DB) GetBackup(bid string) (ServerBackup, error) {
	var bk ServerBackup
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktBackups).Get([]byte(bid))
		if v == nil {
			return fmt.Errorf("backup %q not found", bid)
		}
		return json.Unmarshal(v, &bk)
	})
	return bk, err
}
func (d *DB) DeleteBackup(bid string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bktBackups).Delete([]byte(bid))
	})
}
func (d *DB) ListGuildBackups(gid string) (map[string]ServerBackup, error) {
	res := make(map[string]ServerBackup)
	err := d.b.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktBackups)
		c := bkt.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var bk ServerBackup
			if err := json.Unmarshal(v, &bk); err == nil {
				if bk.GuildID == gid {
					res[string(k)] = bk
				}
			}
		}
		return nil
	})
	return res, err
}