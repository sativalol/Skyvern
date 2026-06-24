package storage

import (
	"encoding/json"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

type CommandAction struct {
	Type   string            `json:"type"`   // send_message, add_role, remove_role, quarantine, dm
	Params map[string]string `json:"params"`
}

type CustomCommand struct {
	Trigger        string          `json:"trigger"`
	Description    string          `json:"desc"`
	RequiredPerms  int64           `json:"required_perms"`
	AllowedRoles   []string        `json:"allowed_roles"`
	BypassExecPerm bool            `json:"bypass_exec_perm"`
	Actions        []CommandAction `json:"actions"`
}

func (d *DB) SaveCustomCommand(gid, trigger string, cmd CustomCommand) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		gbkt, err := tx.Bucket(bktCustomCmds).CreateBucketIfNotExists([]byte(gid))
		if err != nil {
			return err
		}
		return putJSON(gbkt, []byte(trigger), cmd)
	})
}

func (d *DB) GetCustomCommand(gid, trigger string) (CustomCommand, error) {
	var cmd CustomCommand
	err := d.b.View(func(tx *bolt.Tx) error {
		gbkt := tx.Bucket(bktCustomCmds).Bucket([]byte(gid))
		if gbkt == nil {
			return fmt.Errorf("no custom commands")
		}
		v := gbkt.Get([]byte(trigger))
		if v == nil {
			return fmt.Errorf("command not found")
		}
		return json.Unmarshal(v, &cmd)
	})
	return cmd, err
}

func (d *DB) DeleteCustomCommand(gid, trigger string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		gbkt := tx.Bucket(bktCustomCmds).Bucket([]byte(gid))
		if gbkt == nil {
			return nil
		}
		return gbkt.Delete([]byte(trigger))
	})
}

func (d *DB) ListCustomCommands(gid string) ([]CustomCommand, error) {
	var out []CustomCommand
	err := d.b.View(func(tx *bolt.Tx) error {
		gbkt := tx.Bucket(bktCustomCmds).Bucket([]byte(gid))
		if gbkt == nil {
			return nil
		}
		return gbkt.ForEach(func(k, v []byte) error {
			var cmd CustomCommand
			if err := json.Unmarshal(v, &cmd); err == nil {
				out = append(out, cmd)
			}
			return nil
		})
	})
	return out, err
}
