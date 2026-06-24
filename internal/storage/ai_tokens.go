package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	bolt "go.etcd.io/bbolt"
)

type AIBalance struct {
	Balance    int `json:"balance"`
	TokensUsed int `json:"tokens_used"`
}

type AIKey struct {
	Key    string `json:"key"`
	Tokens int    `json:"tokens"`
}

func (d *DB) SaveAIBalance(uid string, bal int) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktAITokens)
		k := []byte("balance:" + uid)
		var rec AIBalance
		v := bkt.Get(k)
		if v != nil {
			_ = json.Unmarshal(v, &rec)
		}
		rec.Balance = bal
		vBytes, err := json.Marshal(rec)
		if err != nil {
			return err
		}
		return bkt.Put(k, vBytes)
	})
}

func (d *DB) GetAIBalance(uid string) (int, error) {
	var bal AIBalance
	err := d.b.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktAITokens)
		v := bkt.Get([]byte("balance:" + uid))
		if v == nil {
			return fmt.Errorf("no balance record")
		}
		return json.Unmarshal(v, &bal)
	})
	if err != nil {
		return 0, err
	}
	return bal.Balance, nil
}

func (d *DB) SaveAIKey(key string, tokens int) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktAITokens)
		k := []byte("key:" + key)
		v, err := json.Marshal(AIKey{Key: key, Tokens: tokens})
		if err != nil {
			return err
		}
		return bkt.Put(k, v)
	})
}

func (d *DB) GetAIKey(key string) (int, error) {
	var ak AIKey
	err := d.b.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktAITokens)
		v := bkt.Get([]byte("key:" + key))
		if v == nil {
			return fmt.Errorf("invalid key")
		}
		return json.Unmarshal(v, &ak)
	})
	if err != nil {
		return 0, err
	}
	return ak.Tokens, nil
}

func (d *DB) DeleteAIKey(key string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktAITokens)
		return bkt.Delete([]byte("key:" + key))
	})
}

func (d *DB) ListAIKeys() (map[string]int, error) {
	out := make(map[string]int)
	err := d.b.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktAITokens)
		c := bkt.Cursor()
		prefix := []byte("key:")
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			var ak AIKey
			if err := json.Unmarshal(v, &ak); err == nil {
				out[ak.Key] = ak.Tokens
			}
		}
		return nil
	})
	return out, err
}

func (d *DB) ChargeAndIncrementAIToken(uid string, charge bool) (int, int, error) {
	var bal int
	var used int
	err := d.b.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktAITokens)
		k := []byte("balance:" + uid)
		var rec AIBalance
		v := bkt.Get(k)
		if v != nil {
			_ = json.Unmarshal(v, &rec)
		}
		if charge && rec.Balance > 0 {
			rec.Balance--
		}
		rec.TokensUsed++
		bal = rec.Balance
		used = rec.TokensUsed
		vBytes, err := json.Marshal(rec)
		if err != nil {
			return err
		}
		return bkt.Put(k, vBytes)
	})
	return bal, used, err
}
