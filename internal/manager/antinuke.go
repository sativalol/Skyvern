package manager
import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"skyvern/internal/config"
	"skyvern/internal/storage"
)
var (
	anMu      sync.Mutex
	anActions = make(map[string]map[string]map[string][]time.Time)                                                  
)
func sfTimeConv(id string) (time.Time, error) {
	var sf int64
	var v uint64
	for _, r := range id {
		if r >= '0' && r <= '9' {
			v = v*10 + uint64(r-'0')
		}
	}
	sf = int64(v)
	t := (sf >> 22) + 1420070400000
	return time.Unix(0, t*int64(time.Millisecond)), nil
}
func (m *Manager) TrackAntinuke(s *discordgo.Session, gid, targetID string, act discordgo.AuditLogAction) {
	cfg, err := m.GetAntinukeCfg(gid)
	if err != nil || !cfg.Enabled {
		return
	}
	time.Sleep(800 * time.Millisecond)
	log, err := s.GuildAuditLog(gid, "", "", int(act), 5)
	if err != nil {
		return
	}
	bid := s.State.User.ID
	var actorID string
	now := time.Now()
	for _, ent := range log.AuditLogEntries {
		if targetID != "" && ent.TargetID != targetID {
			continue
		}
		if ent.UserID == bid {
			return                                          
		}
		t, err := sfTimeConv(ent.ID)
		if err == nil && now.Sub(t) < 8*time.Second {
			actorID = ent.UserID
			break
		}
	}
	if actorID == "" && len(log.AuditLogEntries) > 0 {
		first := log.AuditLogEntries[0]
		if first.UserID != bid {
			t, err := sfTimeConv(first.ID)
			if err == nil && now.Sub(t) < 8*time.Second {
				actorID = first.UserID
			}
		}
	}
	if actorID == "" {
		return
	}
	if m.db.IsAntinukeAdmin(gid, actorID) || m.db.IsAntinukeWhitelisted(gid, actorID) {
		return
	}
	if isGuildOwner(s, gid, actorID) {
		return
	}
	var actType string
	var limit, secs int
	switch act {
	case discordgo.AuditLogActionChannelCreate:
		if !cfg.ChanEnabled { return }
		actType = "chan_create"
		limit = cfg.ChanLimit
		secs = cfg.ChanSecs
	case discordgo.AuditLogActionChannelDelete:
		if !cfg.ChanEnabled { return }
		actType = "chan_delete"
		limit = cfg.ChanLimit
		secs = cfg.ChanSecs
	case discordgo.AuditLogActionRoleCreate:
		if !cfg.RoleEnabled { return }
		actType = "role_create"
		limit = cfg.RoleLimit
		secs = cfg.RoleSecs
	case discordgo.AuditLogActionRoleDelete:
		if !cfg.RoleEnabled { return }
		actType = "role_delete"
		limit = cfg.RoleLimit
		secs = cfg.RoleSecs
	case discordgo.AuditLogActionMemberBanAdd:
		if !cfg.BanEnabled { return }
		actType = "ban"
		limit = cfg.BanLimit
		secs = cfg.BanSecs
	case discordgo.AuditLogActionMemberKick:
		if !cfg.KickEnabled { return }
		actType = "kick"
		limit = cfg.KickLimit
		secs = cfg.KickSecs
	case discordgo.AuditLogActionBotAdd:
		if !cfg.BotaddEnabled { return }
		m.punishAdmin(s, gid, actorID, "botadd", 1, 0, resolveAction(cfg, "botadd"))
		return
	case discordgo.AuditLogActionWebhookCreate:
		if !cfg.WebhookEnabled { return }
		actType = "webhook"
		limit = cfg.WebhookLimit
		secs = cfg.WebhookSecs
	case discordgo.AuditLogActionWebhookDelete:
		if !cfg.WebhookEnabled { return }
		actType = "webhook"
		limit = cfg.WebhookLimit
		secs = cfg.WebhookSecs
	case discordgo.AuditLogActionEmojiCreate:
		if !cfg.EmojiEnabled { return }
		actType = "emoji"
		limit = cfg.EmojiLimit
		secs = cfg.EmojiSecs
	case discordgo.AuditLogActionEmojiDelete:
		if !cfg.EmojiEnabled { return }
		actType = "emoji"
		limit = cfg.EmojiLimit
		secs = cfg.EmojiSecs
	case discordgo.AuditLogActionChannelOverwriteCreate, discordgo.AuditLogActionChannelOverwriteUpdate, discordgo.AuditLogActionChannelOverwriteDelete:
		if !cfg.OverwriteEnabled { return }
		actType = "overwrite"
		limit = cfg.OverwriteLimit
		secs = cfg.OverwriteSecs
	case discordgo.AuditLogActionMemberPrune:
		if !cfg.PruneEnabled { return }
		m.punishAdmin(s, gid, actorID, "prune", 1, 0, resolveAction(cfg, "prune"))
		return
	case discordgo.AuditLogActionMessageBulkDelete:
		if !cfg.PurgeEnabled { return }
		actType = "purge"
		limit = cfg.PurgeLimit
		secs = cfg.PurgeSecs
	case discordgo.AuditLogActionStickerCreate, discordgo.AuditLogActionStickerDelete:
		if !cfg.StickerEnabled { return }
		actType = "sticker"
		limit = cfg.StickerLimit
		secs = cfg.StickerSecs
	case discordgo.AuditLogActionGuildUpdate:
		if !cfg.VanityEnabled { return }
		isVanityUpdate := false
		for _, ent := range log.AuditLogEntries {
			for _, chg := range ent.Changes {
				if chg.Key != nil && string(*chg.Key) == "vanity_url_code" {
					isVanityUpdate = true
					break
				}
			}
			if isVanityUpdate {
				break
			}
		}
		if !isVanityUpdate {
			return
		}
		actType = "vanity"
		limit = cfg.VanityLimit
		secs = cfg.VanitySecs
	default:
		return
	}
	anMu.Lock()
	if anActions[gid] == nil {
		anActions[gid] = make(map[string]map[string][]time.Time)
	}
	if anActions[gid][actorID] == nil {
		anActions[gid][actorID] = make(map[string][]time.Time)
	}
	var active []time.Time
	cutoff := now.Add(-time.Duration(secs) * time.Second)
	for _, ts := range anActions[gid][actorID][actType] {
		if ts.After(cutoff) {
			active = append(active, ts)
		}
	}
	active = append(active, now)
	anActions[gid][actorID][actType] = active
	anMu.Unlock()
	if len(active) > limit {
		m.punishAdmin(s, gid, actorID, actType, len(active), limit, resolveAction(cfg, actType))
	}
}
func resolvePermFlag(name string) int64 {
	switch strings.ToLower(name) {
	case "administrator", "admin":
		return discordgo.PermissionAdministrator
	case "manage_roles", "roles":
		return discordgo.PermissionManageRoles
	case "manage_guild", "guild":
		return discordgo.PermissionManageGuild
	case "ban_members", "ban":
		return discordgo.PermissionBanMembers
	case "kick_members", "kick":
		return discordgo.PermissionKickMembers
	case "manage_webhooks", "webhooks":
		return discordgo.PermissionManageWebhooks
	case "manage_channels", "channels":
		return discordgo.PermissionManageChannels
	default:
		return 0
	}
}
func parsePermsVal(val interface{}) int64 {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case string:
		num, _ := strconv.ParseInt(v, 10, 64)
		return num
	default:
		return 0
	}
}
func parseRoleIDsFromChange(val interface{}) []string {
	var ids []string
	if val == nil {
		return ids
	}
	slice, ok := val.([]interface{})
	if !ok {
		return ids
	}
	for _, item := range slice {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if idVal, ok := m["id"]; ok {
			if idStr, ok := idVal.(string); ok {
				ids = append(ids, idStr)
			}
		}
	}
	return ids
}
func (m *Manager) CheckRolePermissionUpdate(s *discordgo.Session, e *discordgo.GuildRoleUpdate) {
	cfg, err := m.db.GetAntinukeCfg(e.GuildID)
	if err != nil || !cfg.Enabled || !cfg.PermsEnabled || len(cfg.WatchRolePerms) == 0 {
		return
	}
	time.Sleep(800 * time.Millisecond)
	log, err := s.GuildAuditLog(e.GuildID, "", "", int(discordgo.AuditLogActionRoleUpdate), 5)
	if err != nil {
		return
	}
	bid := s.State.User.ID
	now := time.Now()
	var actorID string
	var targetEntry *discordgo.AuditLogEntry
	for _, ent := range log.AuditLogEntries {
		if ent.TargetID != e.Role.ID {
			continue
		}
		if ent.UserID == bid {
			return
		}
		t, err := sfTimeConv(ent.ID)
		if err == nil && now.Sub(t) < 8*time.Second {
			actorID = ent.UserID
			targetEntry = ent
			break
		}
	}
	if actorID == "" || targetEntry == nil {
		return
	}
	if m.db.IsAntinukeAdmin(e.GuildID, actorID) || m.db.IsAntinukeWhitelisted(e.GuildID, actorID) {
		return
	}
	if isGuildOwner(s, e.GuildID, actorID) {
		return
	}
	triggered := false
	for _, chg := range targetEntry.Changes {
		if chg.Key != nil && string(*chg.Key) == "permissions" {
			oldP := parsePermsVal(chg.OldValue)
			newP := parsePermsVal(chg.NewValue)
			diff := oldP ^ newP
			for _, pName := range cfg.WatchRolePerms {
				pFlag := resolvePermFlag(pName)
				if (diff & pFlag) != 0 {
					triggered = true
					break
				}
			}
		}
	}
	if triggered {
		m.punishAdmin(s, e.GuildID, actorID, "permissions (role)", 1, 0, cfg.Action)
	}
}
func (m *Manager) CheckMemberRolePermissionUpdate(s *discordgo.Session, e *discordgo.GuildMemberUpdate) {
	cfg, err := m.db.GetAntinukeCfg(e.GuildID)
	if err != nil || !cfg.Enabled || !cfg.PermsEnabled || len(cfg.WatchUserPerms) == 0 {
		return
	}
	time.Sleep(800 * time.Millisecond)
	log, err := s.GuildAuditLog(e.GuildID, "", "", int(discordgo.AuditLogActionMemberRoleUpdate), 5)
	if err != nil {
		return
	}
	bid := s.State.User.ID
	now := time.Now()
	var actorID string
	var targetEntry *discordgo.AuditLogEntry
	for _, ent := range log.AuditLogEntries {
		if ent.TargetID != e.Member.User.ID {
			continue
		}
		if ent.UserID == bid {
			return
		}
		t, err := sfTimeConv(ent.ID)
		if err == nil && now.Sub(t) < 8*time.Second {
			actorID = ent.UserID
			targetEntry = ent
			break
		}
	}
	if actorID == "" || targetEntry == nil {
		return
	}
	if m.db.IsAntinukeAdmin(e.GuildID, actorID) || m.db.IsAntinukeWhitelisted(e.GuildID, actorID) {
		return
	}
	if isGuildOwner(s, e.GuildID, actorID) {
		return
	}
	triggered := false
	roles, err := s.GuildRoles(e.GuildID)
	if err != nil {
		return
	}
	roleMap := make(map[string]*discordgo.Role)
	for _, r := range roles {
		roleMap[r.ID] = r
	}
	for _, chg := range targetEntry.Changes {
		if chg.Key != nil && (string(*chg.Key) == "$add" || string(*chg.Key) == "$remove") {
			rIDs := parseRoleIDsFromChange(chg.NewValue)
			for _, rid := range rIDs {
				if r, ok := roleMap[rid]; ok {
					for _, pName := range cfg.WatchUserPerms {
						pFlag := resolvePermFlag(pName)
						if (r.Permissions & pFlag) != 0 {
							triggered = true
							break
						}
					}
				}
				if triggered {
					break
				}
			}
		}
		if triggered {
			break
		}
	}
	if triggered {
		m.punishAdmin(s, e.GuildID, actorID, "permissions (member)", 1, 0, cfg.Action)
	}
}
func (m *Manager) punishAdmin(s *discordgo.Session, gid, actorID, actType string, count, limit int, punish string) {
	mem, err := s.GuildMember(gid, actorID)
	if err != nil {
		return
	}
	reason := fmt.Sprintf("[Skyvern Antinuke] Triggered %s limit (%d/%d actions)", actType, count, limit)
	m.logAntinukeAlert(s, gid, actorID, actType, count, limit, punish)
	switch strings.ToLower(punish) {
	case "ban":
		_ = s.GuildBanCreateWithReason(gid, actorID, reason, 7)
	case "kick":
		_ = s.GuildMemberDeleteWithReason(gid, actorID, reason)
	case "quarantine":
		_ = m.QuarantineUser(s, gid, actorID, reason, "antinuke")
	default:           
		for _, rid := range mem.Roles {
			_ = s.GuildMemberRoleRemove(gid, actorID, rid)
		}
	}
}

func resolveAction(cfg storage.AntinukeCfg, mod string) string {
	var act string
	switch mod {
	case "chan_create", "chan_delete":
		act = cfg.ChanAction
	case "role_create", "role_delete":
		act = cfg.RoleAction
	case "ban":
		act = cfg.BanAction
	case "kick":
		act = cfg.KickAction
	case "webhook":
		act = cfg.WebhookAction
	case "emoji":
		act = cfg.EmojiAction
	case "vanity":
		act = cfg.VanityAction
	case "botadd":
		act = cfg.BotaddAction
	case "overwrite":
		act = cfg.OverwriteAction
	case "prune":
		act = cfg.PruneAction
	case "purge":
		act = cfg.PurgeAction
	case "sticker":
		act = cfg.StickerAction
	}
	if act == "" {
		return cfg.Action
	}
	return act
}
func (m *Manager) logAntinukeAlert(s *discordgo.Session, gid, actorID, actType string, count, limit int, punish string) {
	anCfg, err := m.db.GetAntinukeCfg(gid)
	if err == nil && anCfg.NukeWebhookURL != "" {
		payload := map[string]interface{}{
			"content": "⚠️ **[Skyvern Antinuke Alert] Emergency Protection Triggered**",
			"embeds": []map[string]interface{}{
				{
					"title": "Antinuke Alert - Protection Triggered",
					"color": 16711680,
					"fields": []map[string]interface{}{
						{"name": "Guild ID", "value": gid, "inline": true},
						{"name": "Administrator", "value": fmt.Sprintf("<@%s> (`%s`)", actorID, actorID), "inline": true},
						{"name": "Action Type", "value": actType, "inline": true},
						{"name": "Actions Count", "value": fmt.Sprintf("%d/%d", count, limit), "inline": true},
						{"name": "Punishment Applied", "value": punish, "inline": true},
					},
					"timestamp": time.Now().Format(time.RFC3339),
				},
			},
		}
		if data, err := json.Marshal(payload); err == nil {
			go func() {
				req, _ := http.NewRequest("POST", anCfg.NukeWebhookURL, bytes.NewBuffer(data))
				req.Header.Set("Content-Type", "application/json")
				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Do(req)
				if err == nil {
					resp.Body.Close()
				}
			}()
		}
	}

	mCfg, err := m.db.GetModlog(gid)
	if err != nil || mCfg.ChannelID == "" {
		return
	}
	inst, err := m.db.GetBot(s.State.User.ID)
	var resolved config.ResCfg
	if err == nil {
		resolved = config.Resolve(config.GetGlobal(), inst)
	} else {
		resolved = config.Resolve(config.GetGlobal(), config.BotInst{})
	}
	ef := []*discordgo.MessageEmbedField{
		config.Field("Administrator", fmt.Sprintf("<@%s> (`%s`)", actorID, actorID), true),
		config.Field("Action Type", actType, true),
		config.Field("Actions count", fmt.Sprintf("%d/%d", count, limit), true),
		config.Field("Punishment Applied", punish, true),
	}
	emb := config.Build(resolved, config.EmbedOpt{
		Title:  "Antinuke Alert - Protection Triggered",
		Fields: ef,
	})
	emb.Color = 0xff0000                                 
	_, _ = s.ChannelMessageSendEmbed(mCfg.ChannelID, emb)
}
func isGuildOwner(s *discordgo.Session, gid, uid string) bool {
	g, err := s.State.Guild(gid)
	if err != nil {
		g, err = s.Guild(gid)
	}
	return err == nil && g.OwnerID == uid
}