package general
import (
	"fmt"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"sort"
	"strings"
	"time"
	"github.com/bwmarrin/discordgo"
)
func init() {
	manager.RegisterHelp("boosters", []manager.HelpPage{
		{
			Command:     "Boosters List",
			Syntax:      ".boosters",
			Description: "Lists all current server boosters.",
		},
		{
			Command:     "Lost Boosters",
			Syntax:      ".boosters lost",
			Description: "Lists members who recently stopped boosting.",
		},
	})
}
var Boosters = &manager.Command{
	Trigger:     "boosters",
	Aliases:     []string{"boosts"},
	Name:        "boosters",
	Description: "View boosters or lost boosters",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()
		if len(ctx.Args) > 0 && strings.ToLower(ctx.Args[0]) == "lost" {
			g, err := ctx.Session.State.Guild(gid)
			if err != nil {
				g, err = ctx.Session.Guild(gid)
			}
			if err != nil {
				return ctx.Reply("[!] Failed to fetch guild details.")
			}
			var boostRole string
			for _, r := range g.Roles {
				if r.Managed && strings.EqualFold(r.Name, "Server Booster") {
					boostRole = r.ID
					break
				}
			}
			if boostRole == "" {
				members, err := ctx.Session.GuildMembers(gid, "", 1000)
				if err == nil {
					for _, m := range members {
						if m.PremiumSince != nil && len(m.Roles) > 0 {
							for _, rid := range m.Roles {
								for _, r := range g.Roles {
									if r.ID == rid && r.Managed {
										boostRole = r.ID
										break
									}
								}
								if boostRole != "" {
									break
								}
							}
						}
						if boostRole != "" {
							break
						}
					}
				}
			}
			if boostRole == "" {
				return ctx.Reply("[!] No boosting role found for this server.")
			}
			audit, err := ctx.Session.GuildAuditLog(gid, "", "", 25, 50)                      
			if err != nil {
				return ctx.Reply("[!] Failed to retrieve audit logs.")
			}
			var lost []string
			for _, entry := range audit.AuditLogEntries {
				for _, ch := range entry.Changes {
					if ch.Key != nil && string(*ch.Key) == "$remove" {
						if list, ok := ch.NewValue.([]any); ok {
							for _, r := range list {
								if rMap, ok := r.(map[string]any); ok {
									if id, ok := rMap["id"].(string); ok && id == boostRole {
										lost = append(lost, fmt.Sprintf("<@%s> (Moderator: <@%s>)", entry.TargetID, entry.UserID))
									}
								}
							}
						}
					}
				}
			}
			if len(lost) == 0 {
				return ctx.Reply("[*] No recently lost boosters found in audit logs.")
			}
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:       "Recently Lost Boosters",
				Description: strings.Join(lost, "\n"),
			})
			return ctx.Respond(emb)
		}
		members, err := ctx.Session.GuildMembers(gid, "", 1000)
		if err != nil {
			return ctx.Reply("[!] Failed to fetch guild members.")
		}
		var boosters []*discordgo.Member
		for _, m := range members {
			if m.PremiumSince != nil {
				boosters = append(boosters, m)
			}
		}
		if len(boosters) == 0 {
			return ctx.Reply("[*] This server has no boosters.")
		}
		sort.Slice(boosters, func(i, j int) bool {
			return boosters[i].PremiumSince.After(*boosters[j].PremiumSince)
		})
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("**Server Boosters (%d):**\n\n", len(boosters)))
		for idx, m := range boosters {
			dur := time.Since(*m.PremiumSince).Round(time.Hour * 24)
			sb.WriteString(fmt.Sprintf("%d. <@%s> - Boosted %d days ago\n", idx+1, m.User.ID, int(dur.Hours()/24)))
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "Server Boosters",
			Description: sb.String(),
		})
		return ctx.Respond(emb)
	},
}