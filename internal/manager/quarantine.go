package manager

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"skyvern/internal/storage"
)

func ptr[T any](v T) *T {
	return &v
}

func (m *Manager) EnsureQuarantineRole(s *discordgo.Session, gid string) (string, error) {
	cfg, err := m.db.GetAntinukeCfg(gid)
	if err != nil {
		cfg = storage.AntinukeCfg{}
	}

	roles, err := s.GuildRoles(gid)
	if err != nil {
		return "", err
	}

	var exists bool
	var qRoleID string
	if cfg.QuarantineRoleID != "" {
		for _, r := range roles {
			if r.ID == cfg.QuarantineRoleID {
				exists = true
				qRoleID = r.ID
				break
			}
		}
	}

	if !exists {
		r, err := s.GuildRoleCreate(gid, &discordgo.RoleParams{
			Name:        "Quarantined",
			Color:       ptr(0x708090),
			Permissions: ptr(int64(0)),
		})
		if err != nil {
			return "", err
		}
		qRoleID = r.ID
		cfg.QuarantineRoleID = qRoleID
		_ = m.db.SaveAntinukeCfg(gid, cfg)
	}

	m.SyncQuarantineOverrides(s, gid, qRoleID)
	return qRoleID, nil
}

func (m *Manager) SyncQuarantineOverrides(s *discordgo.Session, gid, qRoleID string) {
	chans, err := s.GuildChannels(gid)
	if err != nil {
		return
	}
	denyPerms := int64(discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionVoiceConnect)
	for _, c := range chans {
		_ = s.ChannelPermissionSet(
			c.ID,
			qRoleID,
			discordgo.PermissionOverwriteTypeRole,
			0,
			denyPerms,
			discordgo.WithAuditLogReason("Quarantine channel override sync"),
		)
	}
}

func (m *Manager) QuarantineUser(s *discordgo.Session, gid, uid, reason, by string) error {
	qRoleID, err := m.EnsureQuarantineRole(s, gid)
	if err != nil {
		return fmt.Errorf("ensure quarantine role: %w", err)
	}

	if m.db.IsQuarantined(gid, uid) {
		_ = s.GuildMemberRoleAdd(gid, uid, qRoleID)
		return nil
	}

	mem, err := s.GuildMember(gid, uid)
	if err != nil {
		return fmt.Errorf("get member: %w", err)
	}

	originalRoles := make([]string, len(mem.Roles))
	copy(originalRoles, mem.Roles)

	err = m.db.SaveQuarantined(gid, uid, originalRoles, reason, time.Now(), by)
	if err != nil {
		return fmt.Errorf("save quarantine state: %w", err)
	}

	for _, r := range mem.Roles {
		_ = s.GuildMemberRoleRemove(gid, uid, r)
	}

	_ = s.GuildMemberRoleAdd(gid, uid, qRoleID)
	return nil
}

func (m *Manager) ReleaseUser(s *discordgo.Session, gid, uid string) error {
	cfg, err := m.db.GetAntinukeCfg(gid)
	var qRoleID string
	if err == nil {
		qRoleID = cfg.QuarantineRoleID
	}

	entry, err := m.db.GetQuarantined(gid, uid)
	if err != nil {
		if qRoleID != "" {
			_ = s.GuildMemberRoleRemove(gid, uid, qRoleID)
		}
		return err
	}

	if qRoleID != "" {
		_ = s.GuildMemberRoleRemove(gid, uid, qRoleID)
	}

	for _, rID := range entry.Roles {
		_ = s.GuildMemberRoleAdd(gid, uid, rID)
	}

	_ = m.db.DeleteQuarantined(gid, uid)
	return nil
}
