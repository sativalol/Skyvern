package manager

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"skyvern/internal/config"
	"skyvern/internal/storage"
)

var rxMsgLink = regexp.MustCompile(`channels/(?:\d+|@me)/(\d+)/(\d+)`)

func ParseDuration(s string) (time.Duration, error) {
	s = strings.ToLower(s)
	if strings.HasSuffix(s, "d") {
		valStr := s[:len(s)-1]
		val, err := strconv.Atoi(valStr)
		if err != nil {
			return 0, err
		}
		return time.Duration(val) * 24 * time.Hour, nil
	}
	if strings.HasSuffix(s, "w") {
		valStr := s[:len(s)-1]
		val, err := strconv.Atoi(valStr)
		if err != nil {
			return 0, err
		}
		return time.Duration(val) * 7 * 24 * time.Hour, nil
	}
	if len(s) > 0 && s[len(s)-1] >= '0' && s[len(s)-1] <= '9' {
		s += "m"
	}
	return time.ParseDuration(s)
}

func SnowflakeToTime(idStr string) (time.Time, error) {
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	ms := (id >> 22) + 1420070400000
	return time.UnixMilli(int64(ms)), nil
}

func ParseColor(colorStr string) int {
	colorStr = strings.TrimPrefix(colorStr, "#")
	if val, err := strconv.ParseInt(colorStr, 16, 32); err == nil {
		return int(val)
	}
	switch strings.ToLower(colorStr) {
	case "red":
		return 0xFF0000
	case "green":
		return 0x00FF00
	case "blue":
		return 0x0000FF
	case "yellow":
		return 0xFFFF00
	case "purple":
		return 0x800080
	case "orange":
		return 0xFFA500
	case "pink":
		return 0xFFC0CB
	case "cyan":
		return 0x00FFFF
	default:
		return 0x2b2d31
	}
}

func DrawWinners(entries []string, count int) []string {
	if len(entries) == 0 {
		return nil
	}
	pool := make([]string, len(entries))
	copy(pool, entries)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(pool), func(i, j int) {
		pool[i], pool[j] = pool[j], pool[i]
	})

	if count > len(pool) {
		count = len(pool)
	}
	return pool[:count]
}

func BuildGiveawayButton(mid string, ended bool) discordgo.MessageComponent {
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Join Giveaway 🎉",
				Style:    discordgo.SuccessButton,
				CustomID: "giveaway_join_" + mid,
				Disabled: ended,
			},
		},
	}
}

func BuildGiveawayEmbed(g storage.Giveaway, cfg config.ResCfg) *discordgo.MessageEmbed {
	color := 0x2b2d31
	if g.Color != "" {
		color = ParseColor(g.Color)
	}

	var reqs []string
	if g.MinLevel > 0 {
		reqs = append(reqs, fmt.Sprintf("Minimum Level: **%d**", g.MinLevel))
	}
	if g.MaxLevel > 0 {
		reqs = append(reqs, fmt.Sprintf("Maximum Level: **%d**", g.MaxLevel))
	}
	if g.AgeDays > 0 {
		reqs = append(reqs, fmt.Sprintf("Minimum Account Age: **%d days**", g.AgeDays))
	}
	if g.StayDays > 0 {
		reqs = append(reqs, fmt.Sprintf("Minimum Server Stay: **%d days**", g.StayDays))
	}
	if len(g.RequiredRoles) > 0 {
		var ms []string
		for _, r := range g.RequiredRoles {
			ms = append(ms, fmt.Sprintf("<@&%s>", r))
		}
		reqs = append(reqs, fmt.Sprintf("Required Roles: %s", strings.Join(ms, ", ")))
	}

	desc := g.Description
	if desc == "" {
		desc = "React with the button below to enter!"
	}

	var sb strings.Builder
	sb.WriteString(desc)
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("**Prize:** %s\n", g.Prize))
	sb.WriteString(fmt.Sprintf("**Hosted by:** <@%s>\n", g.HostID))

	if len(reqs) > 0 {
		sb.WriteString("\n**Requirements:**\n- ")
		sb.WriteString(strings.Join(reqs, "\n- "))
		sb.WriteString("\n")
	}

	if g.Ended {
		sb.WriteString("\n**Status:** **Ended**\n")
		if len(g.Winners) > 0 {
			var wm []string
			for _, w := range g.Winners {
				wm = append(wm, fmt.Sprintf("<@%s>", w))
			}
			sb.WriteString(fmt.Sprintf("**Winners:** %s\n", strings.Join(wm, ", ")))
		} else {
			sb.WriteString("**Winners:** None drawn\n")
		}
	} else {
		sb.WriteString(fmt.Sprintf("**Entries:** `%d`\n", len(g.Entries)))
		sb.WriteString(fmt.Sprintf("**Ends:** <t:%d:R>\n", g.EndTime.Unix()))
	}

	emb := &discordgo.MessageEmbed{
		Title:       "🎉 **GIVEAWAY** 🎉",
		Description: sb.String(),
		Color:       color,
	}
	if g.Image != "" {
		emb.Image = &discordgo.MessageEmbedImage{URL: g.Image}
	}
	if g.Thumbnail != "" {
		emb.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: g.Thumbnail}
	}
	if !g.Ended {
		emb.Footer = &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Winners: %d | Ends", g.WinnersCount),
		}
		emb.Timestamp = g.EndTime.Format(time.RFC3339)
	}
	return emb
}

func (m *Manager) EndGiveaway(s *discordgo.Session, g storage.Giveaway, resCfg config.ResCfg) {
	winners := DrawWinners(g.Entries, g.WinnersCount)
	g.Winners = winners
	g.Ended = true
	_ = m.db.SaveGiveaway(g)

	emb := BuildGiveawayEmbed(g, resCfg)
	comp := BuildGiveawayButton(g.MessageID, true)
	_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    g.ChannelID,
		ID:         g.MessageID,
		Embeds:     &[]*discordgo.MessageEmbed{emb},
		Components: &[]discordgo.MessageComponent{comp},
	})

	for _, winID := range winners {
		for _, roleID := range g.AwardRoles {
			_ = s.GuildMemberRoleAdd(g.GuildID, winID, roleID)
		}
	}

	if len(winners) > 0 {
		var winMentions []string
		for _, w := range winners {
			winMentions = append(winMentions, fmt.Sprintf("<@%s>", w))
		}
		_, _ = s.ChannelMessageSend(g.ChannelID, fmt.Sprintf("🎉 Congratulations %s, you won **%s**!", strings.Join(winMentions, ", "), g.Prize))
	} else {
		_, _ = s.ChannelMessageSend(g.ChannelID, fmt.Sprintf("🎉 No eligible winners could be drawn for **%s**.", g.Prize))
	}
}

func (m *Manager) giveawayLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.mu.RLock()
			var activeSess *discordgo.Session
			for _, inst := range m.instances {
				if inst.running {
					activeSess = inst.session
					break
				}
			}
			m.mu.RUnlock()

			if activeSess == nil {
				continue
			}

			giveaways, err := m.db.ListAllActiveGiveaways()
			if err != nil || len(giveaways) == 0 {
				continue
			}

			now := time.Now()
			for _, g := range giveaways {
				if g.EndTime.Before(now) {
					resCfg, _ := m.ResolvedCfgFor(activeSess.State.User.ID)
					m.EndGiveaway(activeSess, g, resCfg)
				}
			}

		case <-m.stopFlush:
			return
		}
	}
}

func GetRxMsgLink() *regexp.Regexp {
	return rxMsgLink
}
