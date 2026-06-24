package storage
import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	bolt "go.etcd.io/bbolt"
)
type SavedEmbed struct {
	Name      string `json:"name"`
	JSONCode  string `json:"json_code"`
	CreatorID string `json:"creator_id"`
	CreatedAt int64  `json:"created_at"`
}
func (d *DB) SaveEmbed(gid, name, jsonCode, creatorID string) error {
	name = strings.ToLower(name)
	return d.b.Update(func(tx *bolt.Tx) error {
		emb := SavedEmbed{
			Name:      name,
			JSONCode:  jsonCode,
			CreatorID: creatorID,
			CreatedAt: time.Now().Unix(),
		}
		return putJSON(tx.Bucket(bktSavedEmbeds), []byte(gid+":"+name), emb)
	})
}
func (d *DB) GetEmbed(gid, name string) (SavedEmbed, error) {
	name = strings.ToLower(name)
	var emb SavedEmbed
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktSavedEmbeds).Get([]byte(gid + ":" + name))
		if v == nil {
			return fmt.Errorf("embed not found")
		}
		return json.Unmarshal(v, &emb)
	})
	return emb, err
}
func (d *DB) DeleteEmbed(gid, name string) error {
	name = strings.ToLower(name)
	return d.b.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bktSavedEmbeds).Delete([]byte(gid + ":" + name))
	})
}
func (d *DB) ListEmbeds(gid string) ([]SavedEmbed, error) {
	var list []SavedEmbed
	err := d.b.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bktSavedEmbeds).Cursor()
		prefix := []byte(gid + ":")
		for k, v := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, v = c.Next() {
			var emb SavedEmbed
			if err := json.Unmarshal(v, &emb); err == nil {
				list = append(list, emb)
			}
		}
		return nil
	})
	return list, err
}
type NameRecord struct {
	Old       string `json:"old"`
	New       string `json:"new"`
	Timestamp int64  `json:"timestamp"`
}
func (d *DB) AppendMemberNameHistory(gid, uid, oldNick, newNick string) error {
	if oldNick == newNick {
		return nil
	}
	return d.b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktNameHistory)
		key := []byte(gid + ":member:" + uid)
		var history []NameRecord
		if v := b.Get(key); v != nil {
			_ = json.Unmarshal(v, &history)
		}
		if len(history) >= 30 {
			history = history[1:]
		}
		history = append(history, NameRecord{
			Old:       oldNick,
			New:       newNick,
			Timestamp: time.Now().Unix(),
		})
		return putJSON(b, key, history)
	})
}
func (d *DB) GetMemberNameHistory(gid, uid string) ([]NameRecord, error) {
	var history []NameRecord
	err := d.b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktNameHistory)
		v := b.Get([]byte(gid + ":member:" + uid))
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &history)
	})
	return history, err
}
func (d *DB) ClearMemberNameHistory(gid, uid string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bktNameHistory).Delete([]byte(gid + ":member:" + uid))
	})
}
func (d *DB) AppendGuildNameHistory(gid, oldName, newName string) error {
	if oldName == newName {
		return nil
	}
	return d.b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktNameHistory)
		key := []byte(gid + ":guild")
		var history []NameRecord
		if v := b.Get(key); v != nil {
			_ = json.Unmarshal(v, &history)
		}
		if len(history) >= 30 {
			history = history[1:]
		}
		history = append(history, NameRecord{
			Old:       oldName,
			New:       newName,
			Timestamp: time.Now().Unix(),
		})
		return putJSON(b, key, history)
	})
}
func (d *DB) GetGuildNameHistory(gid string) ([]NameRecord, error) {
	var history []NameRecord
	err := d.b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktNameHistory)
		v := b.Get([]byte(gid + ":guild"))
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &history)
	})
	return history, err
}
type AFKMention struct {
	AuthorID  string `json:"author_id"`
	ChannelID string `json:"channel_id"`
	MsgID     string `json:"msg_id"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}
func (d *DB) AddAFKMention(gid, uid string, m AFKMention) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktAFKMentions)
		key := []byte(gid + ":" + uid)
		var list []AFKMention
		if v := b.Get(key); v != nil {
			_ = json.Unmarshal(v, &list)
		}
		if len(list) >= 20 {
			list = list[1:]
		}
		list = append(list, m)
		return putJSON(b, key, list)
	})
}
func (d *DB) GetAFKMentions(gid, uid string) ([]AFKMention, error) {
	var list []AFKMention
	err := d.b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktAFKMentions)
		v := b.Get([]byte(gid + ":" + uid))
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &list)
	})
	return list, err
}
func (d *DB) ClearAFKMentions(gid, uid string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bktAFKMentions).Delete([]byte(gid + ":" + uid))
	})
}
type CmdStat struct {
	Trigger string `json:"trigger"`
	Count   int    `json:"count"`
}
func (d *DB) IncrementCommandUse(trigger string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktGlobal)
		key := []byte("cmdstat:" + trigger)
		count := 0
		if v := b.Get(key); v != nil {
			_ = json.Unmarshal(v, &count)
		}
		count++
		return putJSON(b, key, count)
	})
}
func (d *DB) GetTopCommands() ([]CmdStat, error) {
	var list []CmdStat
	err := d.b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktGlobal)
		c := b.Cursor()
		prefix := []byte("cmdstat:")
		for k, v := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, v = c.Next() {
			trigger := strings.TrimPrefix(string(k), "cmdstat:")
			var count int
			if err := json.Unmarshal(v, &count); err == nil {
				list = append(list, CmdStat{
					Trigger: trigger,
					Count:   count,
				})
			}
		}
		return nil
	})
	return list, err
}