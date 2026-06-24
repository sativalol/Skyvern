package storage
import (
	"encoding/json"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"strings"
	"time"
)
type Giveaway struct {
	GuildID       string    `json:"guild_id"`
	ChannelID     string    `json:"channel_id"`
	MessageID     string    `json:"message_id"`
	HostID        string    `json:"host_id"`
	Prize         string    `json:"prize"`
	WinnersCount  int       `json:"winners_count"`
	EndTime       time.Time `json:"end_time"`
	Ended         bool      `json:"ended"`
	Entries       []string  `json:"entries"`
	Winners       []string  `json:"winners"`
	AgeDays       int      `json:"age_days"`
	StayDays      int      `json:"stay_days"`
	MinLevel      int      `json:"min_level"`
	MaxLevel      int      `json:"max_level"`
	RequiredRoles []string `json:"required_roles"`
	AwardRoles    []string `json:"award_roles"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Image       string `json:"image"`
	Thumbnail   string `json:"thumbnail"`
}
func (d *DB) SaveGiveaway(g Giveaway) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktGiveaways), []byte(g.GuildID+":"+g.MessageID), g)
	})
}
func (d *DB) GetGiveaway(gid, mid string) (Giveaway, error) {
	var g Giveaway
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktGiveaways).Get([]byte(gid + ":" + mid))
		if v == nil {
			return fmt.Errorf("giveaway not found")
		}
		return json.Unmarshal(v, &g)
	})
	return g, err
}
func (d *DB) DeleteGiveaway(gid, mid string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bktGiveaways).Delete([]byte(gid + ":" + mid))
	})
}
func (d *DB) ListActiveGiveaways(gid string) ([]Giveaway, error) {
	var out []Giveaway
	prefix := []byte(gid + ":")
	err := d.b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktGiveaways)
		c := b.Cursor()
		for k, v := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, v = c.Next() {
			var g Giveaway
			if err := json.Unmarshal(v, &g); err == nil {
				if !g.Ended {
					out = append(out, g)
				}
			}
		}
		return nil
	})
	return out, err
}
func (d *DB) ListAllActiveGiveaways() ([]Giveaway, error) {
	var out []Giveaway
	err := d.b.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bktGiveaways).ForEach(func(_, v []byte) error {
			var g Giveaway
			if err := json.Unmarshal(v, &g); err == nil {
				if !g.Ended {
					out = append(out, g)
				}
			}
			return nil
		})
	})
	return out, err
}