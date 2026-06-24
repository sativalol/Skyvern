package storage
import (
	"encoding/json"
	bolt "go.etcd.io/bbolt"
)
func (d *DB) GetHighlights(uid string) ([]string, error) {
	var res []string
	err := d.b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktHighlights)
		v := b.Get([]byte(uid))
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &res)
	})
	return res, err
}
func (d *DB) AddHighlight(uid string, kw string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktHighlights)
		var list []string
		if v := b.Get([]byte(uid)); v != nil {
			_ = json.Unmarshal(v, &list)
		}
		for _, x := range list {
			if x == kw {
				return nil
			}
		}
		list = append(list, kw)
		val, err := json.Marshal(list)
		if err != nil {
			return err
		}
		return b.Put([]byte(uid), val)
	})
}
func (d *DB) RemoveHighlight(uid string, kw string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktHighlights)
		v := b.Get([]byte(uid))
		if v == nil {
			return nil
		}
		var list []string
		_ = json.Unmarshal(v, &list)
		idx := -1
		for i, x := range list {
			if x == kw {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil
		}
		list = append(list[:idx], list[idx+1:]...)
		if len(list) == 0 {
			return b.Delete([]byte(uid))
		}
		val, err := json.Marshal(list)
		if err != nil {
			return err
		}
		return b.Put([]byte(uid), val)
	})
}
func (d *DB) ResetHighlights(uid string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bktHighlights).Delete([]byte(uid))
	})
}
func (d *DB) GetHighlightIgnores(uid string) ([]string, error) {
	var res []string
	err := d.b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktHighlightIgnores)
		v := b.Get([]byte(uid))
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &res)
	})
	return res, err
}
func (d *DB) AddHighlightIgnore(uid string, targetID string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktHighlightIgnores)
		var list []string
		if v := b.Get([]byte(uid)); v != nil {
			_ = json.Unmarshal(v, &list)
		}
		for _, x := range list {
			if x == targetID {
				return nil
			}
		}
		list = append(list, targetID)
		val, err := json.Marshal(list)
		if err != nil {
			return err
		}
		return b.Put([]byte(uid), val)
	})
}
func (d *DB) RemoveHighlightIgnore(uid string, targetID string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktHighlightIgnores)
		v := b.Get([]byte(uid))
		if v == nil {
			return nil
		}
		var list []string
		_ = json.Unmarshal(v, &list)
		idx := -1
		for i, x := range list {
			if x == targetID {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil
		}
		list = append(list[:idx], list[idx+1:]...)
		if len(list) == 0 {
			return b.Delete([]byte(uid))
		}
		val, err := json.Marshal(list)
		if err != nil {
			return err
		}
		return b.Put([]byte(uid), val)
	})
}
func (d *DB) GetAllHighlights() (map[string][]string, error) {
	res := make(map[string][]string)
	err := d.b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktHighlights)
		return b.ForEach(func(k, v []byte) error {
			var list []string
			if err := json.Unmarshal(v, &list); err == nil {
				res[string(k)] = list
			}
			return nil
		})
	})
	return res, err
}
func (d *DB) GetAllHighlightIgnores() (map[string][]string, error) {
	res := make(map[string][]string)
	err := d.b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktHighlightIgnores)
		return b.ForEach(func(k, v []byte) error {
			var list []string
			if err := json.Unmarshal(v, &list); err == nil {
				res[string(k)] = list
			}
			return nil
		})
	})
	return res, err
}