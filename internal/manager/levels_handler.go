package manager

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func XPForLevel(lvl int) int64 {
	if lvl <= 0 {
		return 0
	}
	return int64(100 * lvl * lvl)
}

func LevelForXP(xp int64) int {
	if xp <= 0 {
		return 0
	}
	lvl := 0
	for XPForLevel(lvl+1) <= xp {
		lvl++
	}
	return lvl
}

func (m *Manager) HandleMessageXP(s *discordgo.Session, msg *discordgo.MessageCreate) {
	if msg.GuildID == "" || msg.Author == nil || msg.Author.Bot {
		return
	}

	cfg, err := m.db.GetLevelsCfg(msg.GuildID)
	if err != nil || !cfg.Enabled {
		return
	}

	for _, chID := range cfg.IgnoredChans {
		if chID == msg.ChannelID {
			return
		}
	}

	if msg.Member != nil {
		for _, roleID := range msg.Member.Roles {
			for _, ignoredRoleID := range cfg.IgnoredRoles {
				if roleID == ignoredRoleID {
					return
				}
			}
		}
	}

	cooldownKey := msg.GuildID + ":" + msg.Author.ID
	m.xpCooldownsMu.Lock()
	lastXPTime, onCooldown := m.xpCooldowns[cooldownKey]
	if onCooldown && time.Since(lastXPTime) < 1*time.Minute {
		m.xpCooldownsMu.Unlock()
		return
	}
	m.xpCooldowns[cooldownKey] = time.Now()
	m.xpCooldownsMu.Unlock()

	baseXP := 15 + rand.Int63n(11)
	gained := int64(float64(baseXP) * cfg.Rate)
	if gained <= 0 {
		gained = 1
	}

	u, _ := m.db.GetUserXP(msg.GuildID, msg.Author.ID)
	u.XP += gained

	newLevel := LevelForXP(u.XP)
	oldLevel := u.Level
	u.Level = newLevel

	_ = m.db.SaveUserXP(msg.GuildID, msg.Author.ID, u)

	if newLevel > oldLevel {
		m.SyncLevelRoles(s, msg.GuildID, msg.Author.ID, newLevel)

		if u.MessagesToggle && cfg.MessageMode != "disabled" {
			msgText := cfg.Message
			if msgText == "" {
				msgText = "Congrats {user.mention}, you leveled up to level {level}!"
			}
			msgText = strings.ReplaceAll(msgText, "{user.mention}", "<@"+msg.Author.ID+">")
			msgText = strings.ReplaceAll(msgText, "{user.name}", msg.Author.Username)
			msgText = strings.ReplaceAll(msgText, "{level}", fmt.Sprintf("%d", newLevel))

			if cfg.MessageMode == "dm" {
				ch, err := s.UserChannelCreate(msg.Author.ID)
				if err == nil {
					_, _ = s.ChannelMessageSend(ch.ID, msgText)
				}
			} else {
				chanID := cfg.MessageChan
				if chanID == "" {
					chanID = msg.ChannelID
				}
				_, _ = s.ChannelMessageSend(chanID, msgText)
			}
		}
	}
}

func (m *Manager) SyncLevelRoles(s *discordgo.Session, guildID, userID string, currentLevel int) {
	roles, err := m.db.ListLevelRoles(guildID)
	if err != nil || len(roles) == 0 {
		return
	}

	cfg, _ := m.db.GetLevelsCfg(guildID)

	member, err := s.GuildMember(guildID, userID)
	if err != nil {
		return
	}

	var rolesToAdd []string
	var rolesToRemove []string

	highestRoleLevel := -1
	var highestRoleID string

	for _, lr := range roles {
		if currentLevel >= lr.Level {
			if lr.Level > highestRoleLevel {
				highestRoleLevel = lr.Level
				highestRoleID = lr.RoleID
			}
			rolesToAdd = append(rolesToAdd, lr.RoleID)
		} else {
			rolesToRemove = append(rolesToRemove, lr.RoleID)
		}
	}

	if !cfg.StackRoles && highestRoleID != "" {
		for _, lr := range roles {
			if lr.RoleID != highestRoleID {
				rolesToRemove = append(rolesToRemove, lr.RoleID)
			}
		}
		rolesToAdd = []string{highestRoleID}
	}

	memberRoleMap := make(map[string]bool)
	for _, rID := range member.Roles {
		memberRoleMap[rID] = true
	}

	for _, rID := range rolesToRemove {
		if memberRoleMap[rID] {
			_ = s.GuildMemberRoleRemove(guildID, userID, rID)
		}
	}
	for _, rID := range rolesToAdd {
		if !memberRoleMap[rID] {
			_ = s.GuildMemberRoleAdd(guildID, userID, rID)
		}
	}
}
