package storage
import (
	"encoding/json"
	"fmt"
	"strings"
	bolt "go.etcd.io/bbolt"
)
type ModlogCfg struct {
	ChannelID  string `json:"channel_id"`
	LogDiscord bool   `json:"log_discord"`
}
func (d *DB) SaveModlog(gid string, cfg ModlogCfg) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktModlogs), []byte(gid), cfg)
	})
}
func (d *DB) GetModlog(gid string) (ModlogCfg, error) {
	var cfg ModlogCfg
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktModlogs).Get([]byte(gid))
		if v == nil {
			return fmt.Errorf("not configured")
		}
		return json.Unmarshal(v, &cfg)
	})
	return cfg, err
}
type AntispamCfg struct {
	Enabled     bool     `json:"enabled"`
	Limit       int      `json:"limit"`
	Seconds     int      `json:"seconds"`
	Action      string   `json:"action"`
	TimeoutSecs int      `json:"timeout_secs"`
	Whitelist   []string `json:"whitelist"`
	BypassPerms bool     `json:"bypass_perms"`
}
func (d *DB) SaveAntispamCfg(gid string, cfg AntispamCfg) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		v, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		return tx.Bucket(bktAntispam).Put([]byte(gid), v)
	})
}
func (d *DB) GetAntispamCfg(gid string) (AntispamCfg, error) {
	cfg := AntispamCfg{
		BypassPerms: false,
	}
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktAntispam).Get([]byte(gid))
		if v == nil {
			return fmt.Errorf("not found")
		}
		return json.Unmarshal(v, &cfg)
	})
	if err != nil {
		cfg = AntispamCfg{
			Enabled:     false,
			Limit:       5,
			Seconds:     3,
			Action:      "timeout",
			TimeoutSecs: 600,
			BypassPerms: false,
		}
	}
	if cfg.TimeoutSecs == 0 {
		cfg.TimeoutSecs = 600
	}
	return cfg, nil
}
type FilterCfg struct {
	Enabled      bool     `json:"enabled"`
	BlockedWords []string `json:"blocked_words"`
	AllowedWords []string `json:"allowed_words"`
	Regexes      []string `json:"regexes"`
	BypassPerms  bool     `json:"bypass_perms"`
	Whitelist    []string `json:"whitelist"`
}
func (d *DB) SaveFilterCfg(gid string, cfg FilterCfg) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		v, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		return tx.Bucket(bktFilters).Put([]byte(gid), v)
	})
}
func (d *DB) GetFilterCfg(gid string) (FilterCfg, error) {
	cfg := FilterCfg{
		BypassPerms: false,
	}
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktFilters).Get([]byte(gid))
		if v == nil {
			return fmt.Errorf("not found")
		}
		return json.Unmarshal(v, &cfg)
	})
	if err != nil {
		cfg = FilterCfg{
			Enabled:     false,
			BypassPerms: false,
		}
	}
	return cfg, nil
}
type PalantirCfg struct {
	Enabled         bool     `json:"enabled"`
	BlockedGuilds   []string `json:"blocked_guilds"`
	BlockedChannels []string `json:"blocked_channels"`
	BlockedUsers    []string `json:"blocked_users"`
	BlockedEvents   []string `json:"blocked_events"`
}
func (d *DB) SavePalantirCfg(cfg PalantirCfg) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		v, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		return tx.Bucket(bktPalantirCfg).Put([]byte("global"), v)
	})
}
func (d *DB) GetPalantirCfg() (PalantirCfg, error) {
	var cfg PalantirCfg
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktPalantirCfg).Get([]byte("global"))
		if v == nil {
			return fmt.Errorf("not found")
		}
		return json.Unmarshal(v, &cfg)
	})
	if err != nil {
		cfg = PalantirCfg{
			Enabled: true,
		}
	}
	return cfg, nil
}
type AntilinkCfg struct {
	Enabled          bool     `json:"enabled"`
	Action           string   `json:"action"`
	TimeoutSecs      int      `json:"timeout_secs"`
	BypassPerms      bool     `json:"bypass_perms"`
	Whitelist        []string `json:"whitelist"`
	AllowedDomains   []string `json:"allowed_domains"`
	BlockedDomains   []string `json:"blocked_domains"`
	BlockInvitesOnly bool     `json:"block_invites_only"`
}
func (d *DB) SaveAntilinkCfg(gid string, cfg AntilinkCfg) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		v, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		return tx.Bucket(bktAntilink).Put([]byte(gid), v)
	})
}
func (d *DB) GetAntilinkCfg(gid string) (AntilinkCfg, error) {
	cfg := AntilinkCfg{
		BypassPerms: false,
	}
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktAntilink).Get([]byte(gid))
		if v == nil {
			return fmt.Errorf("not found")
		}
		return json.Unmarshal(v, &cfg)
	})
	if err != nil {
		cfg = AntilinkCfg{
			Enabled:          false,
			Action:           "delete",
			TimeoutSecs:      600,
			BypassPerms:      false,
			BlockInvitesOnly: false,
		}
	}
	if cfg.TimeoutSecs == 0 {
		cfg.TimeoutSecs = 600
	}
	return cfg, nil
}
type StarboardCfg struct {
	ChannelID string `json:"channel_id"`
	Threshold int    `json:"threshold"`
	Enabled   bool   `json:"enabled"`
}
func (d *DB) SaveStarboardCfg(gid string, cfg StarboardCfg) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktStarboardCfg), []byte(gid), cfg)
	})
}
func (d *DB) GetStarboardCfg(gid string) (StarboardCfg, error) {
	var cfg StarboardCfg
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktStarboardCfg).Get([]byte(gid))
		if v == nil {
			return fmt.Errorf("not configured")
		}
		return json.Unmarshal(v, &cfg)
	})
	return cfg, err
}
type TempVoiceCfg struct {
	ParentChannelID   string `json:"parent_channel_id"`
	CategoryID        string `json:"category_id"`
	InterfaceChanID   string `json:"interface_chan_id,omitempty"`
	InterfaceMsgID    string `json:"interface_msg_id,omitempty"`
	Enabled           bool   `json:"enabled"`
	PrivateCategoryID string `json:"private_category_id,omitempty"`
	MusicOnly         bool   `json:"music_only,omitempty"`
	DefaultInterface  bool   `json:"default_interface,omitempty"`
	DefaultRoleID     string `json:"default_role_id,omitempty"`
	DefaultBitrate    int    `json:"default_bitrate,omitempty"`
	DefaultName       string `json:"default_name,omitempty"`
	DefaultRegion     string `json:"default_region,omitempty"`
	JoinRoleID        string `json:"join_role_id,omitempty"`
}
func (d *DB) SaveTempVoiceCfg(gid string, cfg TempVoiceCfg) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktTempVoiceCfg), []byte(gid), cfg)
	})
}
func (d *DB) GetTempVoiceCfg(gid string) (TempVoiceCfg, error) {
	var cfg TempVoiceCfg
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktTempVoiceCfg).Get([]byte(gid))
		if v == nil {
			return fmt.Errorf("not configured")
		}
		return json.Unmarshal(v, &cfg)
	})
	return cfg, err
}
type VanityCfg struct {
	Text    string `json:"text"`
	RoleID  string `json:"role_id"`
	Enabled bool   `json:"enabled"`
}
func (d *DB) SaveVanityCfg(gid string, cfg VanityCfg) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktVanityCfg), []byte(gid), cfg)
	})
}
func (d *DB) GetVanityCfg(gid string) (VanityCfg, error) {
	var cfg VanityCfg
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktVanityCfg).Get([]byte(gid))
		if v == nil {
			return fmt.Errorf("not configured")
		}
		return json.Unmarshal(v, &cfg)
	})
	return cfg, err
}
type BoostCfg struct {
	ChannelID string `json:"channel_id"`
	Message   string `json:"message"`
}
func (d *DB) SaveBoostCfg(gid string, cfg BoostCfg) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktBoostCfg), []byte(gid), cfg)
	})
}
func (d *DB) GetBoostCfg(gid string) (BoostCfg, error) {
	var cfg BoostCfg
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktBoostCfg).Get([]byte(gid))
		if v == nil {
			return fmt.Errorf("not configured")
		}
		return json.Unmarshal(v, &cfg)
	})
	return cfg, err
}
type HallCfg struct {
	FameChannelID  string `json:"fame_channel_id"`
	FameThreshold  int    `json:"fame_threshold"`
	ShameChannelID string `json:"shame_shannel_id"`
	ShameThreshold int    `json:"shame_threshold"`
}
func (d *DB) SaveHallCfg(gid string, cfg HallCfg) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktHallCfg), []byte(gid), cfg)
	})
}
func (d *DB) GetHallCfg(gid string) (HallCfg, error) {
	var cfg HallCfg
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktHallCfg).Get([]byte(gid))
		if v == nil {
			return fmt.Errorf("not configured")
		}
		return json.Unmarshal(v, &cfg)
	})
	return cfg, err
}
type BumpCfg struct {
	ChannelID        string `json:"channel_id"`
	Message          string `json:"message"`
	Enabled          bool   `json:"enabled"`
	ThankYouMessage  string `json:"thank_you_message"`
	AutoClean        bool   `json:"auto_clean"`
	AutoLock         bool   `json:"auto_lock"`
}
func (d *DB) SaveBumpCfg(gid string, cfg BumpCfg) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktBumpReminder), []byte(gid), cfg)
	})
}
func (d *DB) GetBumpCfg(gid string) (BumpCfg, error) {
	var cfg BumpCfg
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktBumpReminder).Get([]byte(gid))
		if v == nil {
			return fmt.Errorf("not configured")
		}
		return json.Unmarshal(v, &cfg)
	})
	if err != nil {
		cfg = BumpCfg{
			Enabled:          false,
			Message:          "It is time to bump the server! Use /bump.",
			ThankYouMessage:  "Thank you for bumping the server!",
			AutoClean:        false,
			AutoLock:         false,
		}
	}
	if cfg.Message == "" {
		cfg.Message = "It is time to bump the server! Use /bump."
	}
	if cfg.ThankYouMessage == "" {
		cfg.ThankYouMessage = "Thank you for bumping the server!"
	}
	return cfg, nil
}
type LoggerSub struct {
	GuildID    string `json:"guild_id"`
	ChannelID  string `json:"channel_id"`
	Category   string `json:"category"`
	EmbedColor string `json:"embed_color,omitempty"`
}
type LoggerIgnore struct {
	GuildID    string `json:"guild_id"`
	TargetID   string `json:"target_id"`
	TargetType string `json:"target_type"`
}
func (d *DB) SaveLoggerSub(sub LoggerSub) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		k := []byte(sub.GuildID + ":" + sub.ChannelID + ":" + sub.Category)
		v, err := json.Marshal(sub)
		if err != nil {
			return err
		}
		return tx.Bucket(bktLoggerSubs).Put(k, v)
	})
}
func (d *DB) DeleteLoggerSub(gid, cid, cat string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		k := []byte(gid + ":" + cid + ":" + cat)
		return tx.Bucket(bktLoggerSubs).Delete(k)
	})
}
func (d *DB) DeleteAllLoggerSubs(gid, cid string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktLoggerSubs)
		c := b.Cursor()
		prefix := []byte(gid + ":" + cid + ":")
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
func (d *DB) GetLoggerSubs(gid, cat string) ([]LoggerSub, error) {
	var out []LoggerSub
	err := d.b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktLoggerSubs)
		c := b.Cursor()
		prefix := []byte(gid + ":")
		for k, v := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, v = c.Next() {
			var sub LoggerSub
			if err := json.Unmarshal(v, &sub); err == nil {
				if sub.Category == cat {
					out = append(out, sub)
				}
			}
		}
		return nil
	})
	return out, err
}
func (d *DB) GetChannelLoggerSubs(gid, cid string) ([]LoggerSub, error) {
	var out []LoggerSub
	prefix := []byte(gid + ":" + cid + ":")
	err := d.b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktLoggerSubs)
		c := b.Cursor()
		for k, v := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, v = c.Next() {
			var sub LoggerSub
			if err := json.Unmarshal(v, &sub); err == nil {
				out = append(out, sub)
			}
		}
		return nil
	})
	return out, err
}
func (d *DB) SaveLoggerIgnore(ig LoggerIgnore) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		k := []byte(ig.GuildID + ":" + ig.TargetID)
		v, err := json.Marshal(ig)
		if err != nil {
			return err
		}
		return tx.Bucket(bktLoggerIgnores).Put(k, v)
	})
}
func (d *DB) DeleteLoggerIgnore(gid, targetID string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		k := []byte(gid + ":" + targetID)
		return tx.Bucket(bktLoggerIgnores).Delete(k)
	})
}
func (d *DB) IsLoggerIgnored(gid, targetID string) bool {
	ignored := false
	_ = d.b.View(func(tx *bolt.Tx) error {
		k := []byte(gid + ":" + targetID)
		v := tx.Bucket(bktLoggerIgnores).Get(k)
		if v != nil {
			ignored = true
		}
		return nil
	})
	return ignored
}
func (d *DB) GetLoggerIgnores(gid string) ([]LoggerIgnore, error) {
	var out []LoggerIgnore
	prefix := []byte(gid + ":")
	err := d.b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktLoggerIgnores)
		c := b.Cursor()
		for k, v := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, v = c.Next() {
			var ig LoggerIgnore
			if err := json.Unmarshal(v, &ig); err == nil {
				out = append(out, ig)
			}
		}
		return nil
	})
	return out, err
}
type AntinukeCfg struct {
	Enabled        bool     `json:"enabled"`
	ChanEnabled    bool     `json:"chan_enabled"`
	ChanLimit      int      `json:"chan_limit"`
	ChanSecs       int      `json:"chan_secs"`
	RoleEnabled    bool     `json:"role_enabled"`
	RoleLimit      int      `json:"role_limit"`
	RoleSecs       int      `json:"role_secs"`
	BanEnabled     bool     `json:"ban_enabled"`
	BanLimit       int      `json:"ban_limit"`
	BanSecs        int      `json:"ban_secs"`
	KickEnabled    bool     `json:"kick_enabled"`
	KickLimit      int      `json:"kick_limit"`
	KickSecs       int      `json:"kick_secs"`
	BotaddEnabled  bool     `json:"botadd_enabled"`
	BotLimit       int      `json:"bot_limit"`
	BotSecs        int      `json:"bot_secs"`
	WebhookEnabled bool     `json:"webhook_enabled"`
	WebhookLimit   int      `json:"webhook_limit"`
	WebhookSecs    int      `json:"webhook_secs"`
	EmojiEnabled   bool     `json:"emoji_enabled"`
	EmojiLimit     int      `json:"emoji_limit"`
	EmojiSecs      int      `json:"emoji_secs"`
	VanityEnabled  bool     `json:"vanity_enabled"`
	VanityLimit    int      `json:"vanity_limit"`
	VanitySecs     int      `json:"vanity_secs"`
	PermsEnabled   bool     `json:"perms_enabled"`
	WatchRolePerms []string `json:"watch_role_perms"`
	WatchUserPerms []string `json:"watch_user_perms"`
	Action         string   `json:"action"`

	// New fields for antinuke expansion
	QuarantineEnabled bool   `json:"quarantine_enabled"`
	QuarantineRoleID  string `json:"quarantine_role_id"`
	OverwriteEnabled  bool   `json:"overwrite_enabled"`
	OverwriteLimit    int    `json:"overwrite_limit"`
	OverwriteSecs     int    `json:"overwrite_secs"`
	PruneEnabled      bool   `json:"prune_enabled"`
	PurgeEnabled      bool   `json:"purge_enabled"`
	PurgeLimit        int    `json:"purge_limit"`
	PurgeSecs         int    `json:"purge_secs"`
	StickerEnabled    bool   `json:"sticker_enabled"`
	StickerLimit      int    `json:"sticker_limit"`
	StickerSecs       int    `json:"sticker_secs"`
	AutoReverse       bool   `json:"auto_reverse"`
	AutoRestore       bool   `json:"auto_restore"`
	StaffLogChannelID string `json:"staff_log_channel_id,omitempty"`
	StaffLogEnabled   bool   `json:"staff_log_enabled"`

	// Per-module action overrides
	ChanAction      string `json:"chan_action,omitempty"`
	RoleAction      string `json:"role_action,omitempty"`
	BanAction       string `json:"ban_action,omitempty"`
	KickAction      string `json:"kick_action,omitempty"`
	WebhookAction   string `json:"webhook_action,omitempty"`
	EmojiAction     string `json:"emoji_action,omitempty"`
	VanityAction    string `json:"vanity_action,omitempty"`
	BotaddAction    string `json:"botadd_action,omitempty"`
	OverwriteAction string `json:"overwrite_action,omitempty"`
	PruneAction     string `json:"prune_action,omitempty"`
	PurgeAction     string `json:"purge_action,omitempty"`
	StickerAction   string `json:"sticker_action,omitempty"`
}
func (d *DB) SaveAntinukeCfg(gid string, cfg AntinukeCfg) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktAntinukeCfg), []byte(gid), cfg)
	})
}
func (d *DB) GetAntinukeCfg(gid string) (AntinukeCfg, error) {
	var cfg AntinukeCfg
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktAntinukeCfg).Get([]byte(gid))
		if v == nil {
			return fmt.Errorf("not configured")
		}
		return json.Unmarshal(v, &cfg)
	})
	if err != nil {
		cfg = AntinukeCfg{
			Enabled:        false,
			ChanEnabled:    true,
			ChanLimit:      4,
			ChanSecs:       10,
			RoleEnabled:    true,
			RoleLimit:      4,
			RoleSecs:       10,
			BanEnabled:     true,
			BanLimit:       4,
			BanSecs:        10,
			KickEnabled:    true,
			KickLimit:      4,
			KickSecs:       10,
			BotaddEnabled:  true,
			BotLimit:       2,
			BotSecs:        10,
			WebhookEnabled: true,
			WebhookLimit:   4,
			WebhookSecs:    10,
			EmojiEnabled:   true,
			EmojiLimit:     4,
			EmojiSecs:      10,
			VanityEnabled:  true,
			VanityLimit:    2,
			VanitySecs:     10,
			PermsEnabled:   true,
			Action:         "strip",
			OverwriteEnabled: true,
			OverwriteLimit:   4,
			OverwriteSecs:    10,
			PruneEnabled:     true,
			PurgeEnabled:     true,
			PurgeLimit:       10,
			PurgeSecs:        10,
			StickerEnabled:   true,
			StickerLimit:     4,
			StickerSecs:      10,
			AutoReverse:      true,
			AutoRestore:      false,
		}
	} else {
		if cfg.ChanLimit == 0 { cfg.ChanLimit = 4 }
		if cfg.ChanSecs == 0 { cfg.ChanSecs = 10 }
		if cfg.RoleLimit == 0 { cfg.RoleLimit = 4 }
		if cfg.RoleSecs == 0 { cfg.RoleSecs = 10 }
		if cfg.BanLimit == 0 { cfg.BanLimit = 4 }
		if cfg.BanSecs == 0 { cfg.BanSecs = 10 }
		if cfg.KickLimit == 0 { cfg.KickLimit = 4 }
		if cfg.KickSecs == 0 { cfg.KickSecs = 10 }
		if cfg.BotLimit == 0 { cfg.BotLimit = 2 }
		if cfg.BotSecs == 0 { cfg.BotSecs = 10 }
		if cfg.WebhookLimit == 0 { cfg.WebhookLimit = 4 }
		if cfg.WebhookSecs == 0 { cfg.WebhookSecs = 10 }
		if cfg.EmojiLimit == 0 { cfg.EmojiLimit = 4 }
		if cfg.EmojiSecs == 0 { cfg.EmojiSecs = 10 }
		if cfg.VanityLimit == 0 { cfg.VanityLimit = 2 }
		if cfg.VanitySecs == 0 { cfg.VanitySecs = 10 }
		if cfg.Action == "" { cfg.Action = "strip" }
		if cfg.OverwriteLimit == 0 { cfg.OverwriteLimit = 4 }
		if cfg.OverwriteSecs == 0 { cfg.OverwriteSecs = 10 }
		if cfg.PurgeLimit == 0 { cfg.PurgeLimit = 10 }
		if cfg.PurgeSecs == 0 { cfg.PurgeSecs = 10 }
		if cfg.StickerLimit == 0 { cfg.StickerLimit = 4 }
		if cfg.StickerSecs == 0 { cfg.StickerSecs = 10 }
	}
	return cfg, nil
}
type AntiraidCfg struct {
	Enabled         bool     `json:"enabled"`
	JoinLimit       int      `json:"join_limit"`
	Seconds         int      `json:"seconds"`
	Action          string   `json:"action"`                                       
	AvatarEnabled   bool     `json:"avatar_enabled"`
	AvatarAction    string   `json:"avatar_action"`                 
	NewAcctsEnabled bool     `json:"new_accts_enabled"`
	NewAcctsAgeMins int      `json:"new_accts_age_mins"`
	NewAcctsAction  string   `json:"new_accts_action"`                 
	Whitelist       []string `json:"whitelist"`
	RaidActive      bool     `json:"raid_active"`

	// New fields
	QuarantineOnRaid       bool   `json:"quarantine_on_raid"`
	MentionEnabled         bool   `json:"mention_enabled"`
	MentionLimit           int    `json:"mention_limit"`
	MentionSecs            int    `json:"mention_secs"`
	MentionAction          string `json:"mention_action"`
	ScoreThreshold         int    `json:"score_threshold"`
	BlockInvitesDuringRaid bool   `json:"block_invites_raid"`
	ScanMaliciousLinks     bool   `json:"scan_malicious"`
}
func (d *DB) SaveAntiraidCfg(gid string, cfg AntiraidCfg) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktAntiraidCfg), []byte(gid), cfg)
	})
}
func (d *DB) GetAntiraidCfg(gid string) (AntiraidCfg, error) {
	var cfg AntiraidCfg
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktAntiraidCfg).Get([]byte(gid))
		if v == nil {
			return fmt.Errorf("not configured")
		}
		return json.Unmarshal(v, &cfg)
	})
	if err != nil {
		cfg = AntiraidCfg{
			Enabled:         false,
			JoinLimit:       10,
			Seconds:         10,
			Action:          "notify",
			AvatarEnabled:   false,
			AvatarAction:    "kick",
			NewAcctsEnabled: false,
			NewAcctsAgeMins: 1440,
			NewAcctsAction:  "kick",
			Whitelist:       []string{},
			RaidActive:      false,
			MentionEnabled:         false,
			MentionLimit:           8,
			MentionSecs:            5,
			MentionAction:          "kick",
			ScoreThreshold:         15,
			BlockInvitesDuringRaid: true,
			ScanMaliciousLinks:     true,
		}
	}
	if cfg.AvatarAction == "" {
		cfg.AvatarAction = "kick"
	}
	if cfg.NewAcctsAgeMins == 0 {
		cfg.NewAcctsAgeMins = 1440
	}
	if cfg.NewAcctsAction == "" {
		cfg.NewAcctsAction = "kick"
	}
	if cfg.Whitelist == nil {
		cfg.Whitelist = []string{}
	}
	if cfg.ScoreThreshold == 0 {
		cfg.ScoreThreshold = 15
	}
	if cfg.MentionLimit == 0 {
		cfg.MentionLimit = 8
	}
	if cfg.MentionSecs == 0 {
		cfg.MentionSecs = 5
	}
	if cfg.MentionAction == "" {
		cfg.MentionAction = "kick"
	}
	return cfg, nil
}
type ClownboardCfg struct {
	Enabled         bool     `json:"enabled"`
	ChannelID       string   `json:"channel_id"`
	Threshold       int      `json:"threshold"`
	Emoji           string   `json:"emoji"`
	SelfStar        bool     `json:"self_star"`
	Timestamp       bool     `json:"timestamp"`
	Attachments     bool     `json:"attachments"`
	JumpURL         bool     `json:"jump_url"`
	Color           int      `json:"color"`
	IgnoredChannels []string `json:"ignored_channels"`
	IgnoredMembers  []string `json:"ignored_members"`
	IgnoredRoles    []string `json:"ignored_roles"`
}
func (d *DB) SaveClownboardCfg(gid string, cfg ClownboardCfg) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bktClownboardCfg), []byte(gid), cfg)
	})
}
func (d *DB) GetClownboardCfg(gid string) (ClownboardCfg, error) {
	var cfg ClownboardCfg
	err := d.b.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bktClownboardCfg).Get([]byte(gid))
		if v == nil {
			return fmt.Errorf("not configured")
		}
		return json.Unmarshal(v, &cfg)
	})
	if err != nil {
		cfg = ClownboardCfg{
			Enabled:     false,
			Threshold:   3,
			Emoji:       "🤡",
			SelfStar:    false,
			Timestamp:   true,
			Attachments: true,
			JumpURL:     true,
			Color:       0xffa500,
		}
	}
	return cfg, nil
}
type ClownPost struct {
	OrigID   string `json:"orig_id"`
	CbMsgID  string `json:"cb_msg_id"`
	ChanID   string `json:"chan_id"`
	AuthorID string `json:"author_id"`
	Count    int    `json:"count"`
	Text     string `json:"text"`
}
func (d *DB) SaveClownPost(gid string, post ClownPost) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		k := []byte(gid + ":" + post.OrigID)
		return putJSON(tx.Bucket(bktClownboardMsg), k, post)
	})
}
func (d *DB) GetClownPost(gid string, origID string) (ClownPost, error) {
	var post ClownPost
	err := d.b.View(func(tx *bolt.Tx) error {
		k := []byte(gid + ":" + origID)
		v := tx.Bucket(bktClownboardMsg).Get(k)
		if v == nil {
			return fmt.Errorf("not found")
		}
		return json.Unmarshal(v, &post)
	})
	return post, err
}
func (d *DB) DeleteClownPost(gid string, origID string) error {
	return d.b.Update(func(tx *bolt.Tx) error {
		k := []byte(gid + ":" + origID)
		return tx.Bucket(bktClownboardMsg).Delete(k)
	})
}
func (d *DB) ListClownPosts(gid string) ([]ClownPost, error) {
	var out []ClownPost
	prefix := []byte(gid + ":")
	err := d.b.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bktClownboardMsg).Cursor()
		for k, v := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, v = c.Next() {
			var post ClownPost
			if json.Unmarshal(v, &post) == nil {
				out = append(out, post)
			}
		}
		return nil
	})
	return out, err
}