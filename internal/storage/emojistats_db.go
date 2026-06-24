package storage
import (
	"encoding/json"
	bolt "go.etcd.io/bbolt"
)
func (d *DB) IncrementEmojiCount(gid string, emojiID string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktEmojiStats)
		stats := make(map[string]int)
		if v := b.Get([]byte(gid)); v != nil {
			_ = json.Unmarshal(v, &stats)
		}
		stats[emojiID]++
		val, err := json.Marshal(stats)
		if err != nil {
			return err
		}
		return b.Put([]byte(gid), val)
	})
}
func (d *DB) GetTopEmojis(gid string) (map[string]int, error) {
	stats := make(map[string]int)
	err := d.b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktEmojiStats)
		v := b.Get([]byte(gid))
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &stats)
	})
	return stats, err
}