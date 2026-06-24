package storage
import (
	"encoding/json"
	bolt "go.etcd.io/bbolt"
)
type TicketProfile struct {
	GuildID      string   `json:"guild_id"`
	Blacklist    []string `json:"blacklist"`                    
	Trainees     []string `json:"trainees"`                                         
	SupportRoles []string `json:"support_roles"`
	Forms        []string `json:"forms"`                           
}
type TicketPanel struct {
	GuildID     string   `json:"guild_id"`
	Name        string   `json:"name"`
	ChannelID   string   `json:"channel_id"`
	MessageID   string   `json:"message_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Options     []string `json:"options"`                
}
type TicketOption struct {
	GuildID     string   `json:"guild_id"`
	Name        string   `json:"name"`
	Emoji       string   `json:"emoji"`
	Description string   `json:"description"`
	CategoryID  string   `json:"category_id"`
	Roles       []string `json:"roles"`
}
type TicketChannel struct {
	ChannelID string            `json:"channel_id"`
	GuildID   string            `json:"guild_id"`
	CreatorID string            `json:"creator_id"`
	ClaimerID string            `json:"claimer_id"`
	Open      bool              `json:"open"`
	Option    string            `json:"option"`
	Reason    string            `json:"reason"`
	Answers   map[string]string `json:"answers"`
	Allowed   []string          `json:"allowed"`                          
	Denied    []string          `json:"denied"`                          
}
type TicketStaffProfile struct {
	GuildID    string `json:"guild_id"`
	UserID     string `json:"user_id"`
	ClaimCount int    `json:"claim_count"`
	CloseCount int    `json:"close_count"`
	ReopenCount int   `json:"reopen_count"`
}
type TicketStats struct {
	GuildID      string `json:"guild_id"`
	TotalOpened  int    `json:"total_opened"`
	TotalClosed  int    `json:"total_closed"`
	TotalClaimed int    `json:"total_claimed"`
}
type TicketReason struct {
	GuildID   string `json:"guild_id"`
	Action    string `json:"action"`                                          
	TargetID  string `json:"target_id"`
	StaffID   string `json:"staff_id"`
	Reason    string `json:"reason"`
	Timestamp int64  `json:"timestamp"`
}
func (d *DB) GetTicketProfile(gid string) (TicketProfile, error) {
	var cfg TicketProfile
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktTicketProfiles).Get([]byte(gid))
		if v == nil {
			cfg = TicketProfile{GuildID: gid}
			return nil
		}
		return json.Unmarshal(v, &cfg)
	})
	return cfg, err
}
func (d *DB) SaveTicketProfile(cfg TicketProfile) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktTicketProfiles), []byte(cfg.GuildID), cfg)
	})
}
func (d *DB) GetTicketPanel(gid, name string) (TicketPanel, error) {
	var cfg TicketPanel
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktTicketPanels).Get([]byte(gid + ":" + name))
		if v == nil {
			cfg = TicketPanel{GuildID: gid, Name: name}
			return nil
		}
		return json.Unmarshal(v, &cfg)
	})
	return cfg, err
}
func (d *DB) SaveTicketPanel(cfg TicketPanel) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktTicketPanels), []byte(cfg.GuildID+":"+cfg.Name), cfg)
	})
}
func (d *DB) DeleteTicketPanel(gid, name string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bktTicketPanels).Delete([]byte(gid + ":" + name))
	})
}
func (d *DB) ListTicketPanels(gid string) ([]TicketPanel, error) {
	var list []TicketPanel
	err := d.b.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bktTicketPanels).Cursor()
		prefix := []byte(gid + ":")
		for k, v := c.Seek(prefix); k != nil && bytesHasPrefix(k, prefix); k, v = c.Next() {
			var p TicketPanel
			if err := json.Unmarshal(v, &p); err == nil {
				list = append(list, p)
			}
		}
		return nil
	})
	return list, err
}
func (d *DB) GetTicketOption(gid, name string) (TicketOption, error) {
	var cfg TicketOption
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktTicketOptions).Get([]byte(gid + ":" + name))
		if v == nil {
			cfg = TicketOption{GuildID: gid, Name: name}
			return nil
		}
		return json.Unmarshal(v, &cfg)
	})
	return cfg, err
}
func (d *DB) SaveTicketOption(cfg TicketOption) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktTicketOptions), []byte(cfg.GuildID+":"+cfg.Name), cfg)
	})
}
func (d *DB) DeleteTicketOption(gid, name string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bktTicketOptions).Delete([]byte(gid + ":" + name))
	})
}
func (d *DB) ListTicketOptions(gid string) ([]TicketOption, error) {
	var list []TicketOption
	err := d.b.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bktTicketOptions).Cursor()
		prefix := []byte(gid + ":")
		for k, v := c.Seek(prefix); k != nil && bytesHasPrefix(k, prefix); k, v = c.Next() {
			var o TicketOption
			if err := json.Unmarshal(v, &o); err == nil {
				list = append(list, o)
			}
		}
		return nil
	})
	return list, err
}
func (d *DB) GetTicketChannel(cid string) (TicketChannel, error) {
	var cfg TicketChannel
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktTicketChannels).Get([]byte(cid))
		if v == nil {
			cfg = TicketChannel{ChannelID: cid}
			return nil
		}
		return json.Unmarshal(v, &cfg)
	})
	return cfg, err
}
func (d *DB) SaveTicketChannel(cfg TicketChannel) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktTicketChannels), []byte(cfg.ChannelID), cfg)
	})
}
func (d *DB) DeleteTicketChannel(cid string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bktTicketChannels).Delete([]byte(cid))
	})
}
func (d *DB) ListTicketChannels(gid string) ([]TicketChannel, error) {
	var list []TicketChannel
	err := d.b.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bktTicketChannels).ForEach(func(k, v []byte) error {
			var tc TicketChannel
			if err := json.Unmarshal(v, &tc); err == nil && tc.GuildID == gid {
				list = append(list, tc)
			}
			return nil
		})
	})
	return list, err
}
func (d *DB) GetTicketStaffProfile(gid, uid string) (TicketStaffProfile, error) {
	var p TicketStaffProfile
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktTicketStaffProfiles).Get([]byte(gid + ":" + uid))
		if v == nil {
			p = TicketStaffProfile{GuildID: gid, UserID: uid}
			return nil
		}
		return json.Unmarshal(v, &p)
	})
	return p, err
}
func (d *DB) SaveTicketStaffProfile(p TicketStaffProfile) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktTicketStaffProfiles), []byte(p.GuildID+":"+p.UserID), p)
	})
}
func (d *DB) GetTicketStats(gid string) (TicketStats, error) {
	var s TicketStats
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktTicketStats).Get([]byte(gid))
		if v == nil {
			s = TicketStats{GuildID: gid}
			return nil
		}
		return json.Unmarshal(v, &s)
	})
	return s, err
}
func (d *DB) SaveTicketStats(s TicketStats) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktTicketStats), []byte(s.GuildID), s)
	})
}
func (d *DB) SaveTicketReason(r TicketReason) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktTicketReasons), []byte(r.GuildID+":"+r.TargetID+":"+r.Action), r)
	})
}
func (d *DB) GetTicketReason(gid, target, action string) (TicketReason, error) {
	var r TicketReason
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktTicketReasons).Get([]byte(gid + ":" + target + ":" + action))
		if v == nil {
			r = TicketReason{GuildID: gid, TargetID: target, Action: action}
			return nil
		}
		return json.Unmarshal(v, &r)
	})
	return r, err
}
func bytesHasPrefix(s, prefix []byte) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := range prefix {
		if s[i] != prefix[i] {
			return false
		}
	}
	return true
}