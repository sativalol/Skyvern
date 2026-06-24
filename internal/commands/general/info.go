package general
import (
	"fmt"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/moderation"
	"strings"
	"time"
	"github.com/bwmarrin/discordgo"
)
var permNames = map[int64]string{
	discordgo.PermissionCreateInstantInvite: "Create Invite",
	discordgo.PermissionKickMembers:         "Kick Members",
	discordgo.PermissionBanMembers:          "Ban Members",
	discordgo.PermissionAdministrator:       "Administrator",
	discordgo.PermissionManageChannels:      "Manage Channels",
	discordgo.PermissionManageGuild:         "Manage Server",
	discordgo.PermissionAddReactions:        "Add Reactions",
	discordgo.PermissionViewAuditLogs:       "View Audit Logs",
	discordgo.PermissionViewChannel:         "View Channel",
	discordgo.PermissionSendMessages:        "Send Messages",
	discordgo.PermissionSendTTSMessages:     "Send TTS",
	discordgo.PermissionManageMessages:      "Manage Messages",
	discordgo.PermissionEmbedLinks:          "Embed Links",
	discordgo.PermissionAttachFiles:         "Attach Files",
	discordgo.PermissionReadMessageHistory:  "Read History",
	discordgo.PermissionMentionEveryone:     "Mention Everyone",
	discordgo.PermissionUseExternalEmojis:   "Use External Emojis",
	discordgo.PermissionVoiceConnect:        "Connect Voice",
	discordgo.PermissionVoiceSpeak:          "Speak Voice",
	discordgo.PermissionVoiceMuteMembers:    "Mute Members",
	discordgo.PermissionVoiceDeafenMembers:  "Deafen Members",
	discordgo.PermissionVoiceMoveMembers:    "Move Members",
	discordgo.PermissionVoiceUseVAD:         "Use VAD",
	discordgo.PermissionChangeNickname:      "Change Nickname",
	discordgo.PermissionManageNicknames:     "Manage Nicknames",
	discordgo.PermissionManageRoles:         "Manage Roles",
	discordgo.PermissionManageWebhooks:      "Manage Webhooks",
	discordgo.PermissionManageEmojis:        "Manage Emojis",
	discordgo.PermissionManageThreads:       "Manage Threads",
}
func decodePerms(p int64) string {
	if (p & discordgo.PermissionAdministrator) != 0 {
		return "Administrator (All Permissions)"
	}
	var list []string
	for bit, name := range permNames {
		if (p & bit) != 0 {
			list = append(list, name)
		}
	}
	if len(list) == 0 {
		return "None"
	}
	return strings.Join(list, ", ")
}
var userBadges = map[int]string{
	1 << 0:  "Discord Employee",
	1 << 1:  "Partnered Server Owner",
	1 << 2:  "HypeSquad Events Member",
	1 << 3:  "Bug Hunter Level 1",
	1 << 6:  "House Bravery",
	1 << 7:  "House Brilliance",
	1 << 8:  "House Balance",
	1 << 9:  "Early Supporter",
	1 << 10: "Team User",
	1 << 14: "Bug Hunter Level 2",
	1 << 16: "Verified Bot",
	1 << 17: "Early Verified Bot Developer",
	1 << 18: "Moderator Programs Alumni",
	1 << 22: "Active Developer",
}
func decodeBadges(flags int) string {
	var list []string
	for bit, name := range userBadges {
		if (flags & bit) != 0 {
			list = append(list, name)
		}
	}
	if len(list) == 0 {
		return "None"
	}
	return strings.Join(list, ", ")
}
func resolveUser(s *discordgo.Session, gid, query string) (*discordgo.User, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}
	m, err := moderation.ResolveMember(s, gid, q)
	if err == nil && m != nil && m.User != nil {
		return m.User, nil
	}
	raw := q
	if strings.HasPrefix(raw, "<@") && strings.HasSuffix(raw, ">") {
		raw = strings.Trim(raw, "<@!>")
	}
	return s.User(raw)
}
func creationTime(id string) time.Time {
	var v uint64
	for _, r := range id {
		if r >= '0' && r <= '9' {
			v = v*10 + uint64(r-'0')
		}
	}
	t := (v >> 22) + 1420070400000
	return time.Unix(0, int64(t)*int64(time.Millisecond))
}
var ServerInfo = &manager.Command{
	Trigger:     "serverinfo",
	Aliases:     []string{"si", "server", "guildinfo"},
	Name:        "serverinfo",
	Description: "Display detailed information about the current server",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()
		if len(ctx.Args) > 0 {
			gid = ctx.Args[0]
		}
		g, err := ctx.Session.State.Guild(gid)
		if err != nil {
			g, err = ctx.Session.Guild(gid)
			if err != nil {
				return ctx.Reply("[!] Failed to fetch server info.")
			}
		}
		created := creationTime(g.ID)
		owner, _ := ctx.Session.User(g.OwnerID)
		ownerTag := g.OwnerID
		if owner != nil {
			ownerTag = fmt.Sprintf("%s (%s)", owner.Username, owner.ID)
		}
		textChans := 0
		voiceChans := 0
		categoryChans := 0
		stageChans := 0
		newsChans := 0
		for _, ch := range g.Channels {
			switch ch.Type {
			case discordgo.ChannelTypeGuildText:
				textChans++
			case discordgo.ChannelTypeGuildVoice:
				voiceChans++
			case discordgo.ChannelTypeGuildCategory:
				categoryChans++
			case discordgo.ChannelTypeGuildStageVoice:
				stageChans++
			case discordgo.ChannelTypeGuildNews:
				newsChans++
			}
		}
		staticEmojis := 0
		animatedEmojis := 0
		for _, em := range g.Emojis {
			if em.Animated {
				animatedEmojis++
			} else {
				staticEmojis++
			}
		}
		online := 0
		idle := 0
		dnd := 0
		offline := 0
		for _, p := range g.Presences {
			switch p.Status {
			case discordgo.StatusOnline:
				online++
			case discordgo.StatusIdle:
				idle++
			case discordgo.StatusDoNotDisturb:
				dnd++
			default:
				offline++
			}
		}
		if offline == 0 && len(g.Presences) > 0 {
			offline = g.MemberCount - len(g.Presences)
		} else if len(g.Presences) == 0 {
			offline = g.MemberCount
		}
		var features []string
		for _, f := range g.Features {
			features = append(features, string(f))
		}
		featuresStr := strings.Join(features, ", ")
		if featuresStr == "" {
			featuresStr = "None"
		}
		fields := []*discordgo.MessageEmbedField{
			config.Field("Owner", ownerTag, true),
			config.Field("Server ID", fmt.Sprintf("`%s`", g.ID), true),
			config.Field("Created At", fmt.Sprintf("<t:%d:F> (<t:%d:R>)", created.Unix(), created.Unix()), false),
			config.Field("Boost Status", fmt.Sprintf("Level **%d** (%d boosts)", g.PremiumTier, g.PremiumSubscriptionCount), true),
			config.Field("Verification Level", fmt.Sprintf("%d", g.VerificationLevel), true),
			config.Field("NSFW Level", fmt.Sprintf("%d", g.NSFWLevel), true),
			config.Field("Channels Breakdown", fmt.Sprintf("Text: **%d** | Voice: **%d** | Category: **%d** | Stage: **%d** | News: **%d**", textChans, voiceChans, categoryChans, stageChans, newsChans), false),
			config.Field("Members Presence", fmt.Sprintf("Online: **%d** | Idle: **%d** | DND: **%d** | Offline: **%d**\nTotal Count: **%d**", online, idle, dnd, offline, g.MemberCount), false),
			config.Field("Emojis & Stickers", fmt.Sprintf("Static: **%d** | Animated: **%d** | Stickers: **%d**", staticEmojis, animatedEmojis, len(g.Stickers)), false),
			config.Field("Vanity Code", fmt.Sprintf("%s", g.VanityURLCode), true),
			config.Field("Roles Count", fmt.Sprintf("%d roles", len(g.Roles)), true),
			config.Field("Features List", featuresStr, false),
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:        g.Name,
			Description:  fmt.Sprintf("Detailed Server Information for **%s**", g.Name),
			Fields:       fields,
			ThumbnailURL: g.IconURL("256"),
		})
		return ctx.Respond(emb)
	},
}
var RoleInfo = &manager.Command{
	Trigger:     "roleinfo",
	Aliases:     []string{"ri"},
	Name:        "roleinfo",
	Description: "Display detailed information about a role",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.Reply("Usage: .roleinfo <role>")
		}
		gid := ctx.GuildID()
		roleArg := strings.Join(ctx.Args, " ")
		rid := resolveRole(ctx.Session, gid, roleArg)
		if rid == "" {
			return ctx.Reply("[!] Could not resolve role.")
		}
		roles, err := ctx.Session.GuildRoles(gid)
		if err != nil {
			return ctx.Reply("[!] Failed to fetch server roles.")
		}
		var targetRole *discordgo.Role
		for _, r := range roles {
			if r.ID == rid {
				targetRole = r
				break
			}
		}
		if targetRole == nil {
			return ctx.Reply("[!] Role not found.")
		}
		members, err := ctx.Session.GuildMembers(gid, "", 1000)
		memberCount := 0
		if err == nil {
			for _, m := range members {
				for _, r := range m.Roles {
					if r == rid {
						memberCount++
						break
					}
				}
			}
		}
		created := creationTime(targetRole.ID)
		colorHex := fmt.Sprintf("#%06X", targetRole.Color)
		fields := []*discordgo.MessageEmbedField{
			config.Field("Role ID", fmt.Sprintf("`%s`", targetRole.ID), true),
			config.Field("Color", fmt.Sprintf("`%s` (%d)", colorHex, targetRole.Color), true),
			config.Field("Position / Level", fmt.Sprintf("%d", targetRole.Position), true),
			config.Field("Mentionable", fmt.Sprintf("`%t`", targetRole.Mentionable), true),
			config.Field("Managed (Bot/Integration)", fmt.Sprintf("`%t`", targetRole.Managed), true),
			config.Field("Hoisted (Sidebar Split)", fmt.Sprintf("`%t`", targetRole.Hoist), true),
			config.Field("Members with Role", fmt.Sprintf("**%d** members", memberCount), true),
			config.Field("Created At", fmt.Sprintf("<t:%d:F> (<t:%d:R>)", created.Unix(), created.Unix()), false),
			config.Field("Permissions Decoded", fmt.Sprintf("```\n%s\n```", decodePerms(targetRole.Permissions)), false),
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:  "Role Info: " + targetRole.Name,
			Fields: fields,
		})
		return ctx.Respond(emb)
	},
}
var Whois = &manager.Command{
	Trigger:     "whois",
	Aliases:     []string{"userinfo", "ui", "user"},
	Name:        "whois",
	Description: "Fetch technical data on a specific user",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()
		query := ""
		if len(ctx.Args) > 0 {
			query = ctx.Args[0]
		} else {
			query = ctx.AuthorID()
		}
		usr, err := resolveUser(ctx.Session, gid, query)
		if err != nil || usr == nil {
			return ctx.Reply("[!] Could not resolve user.")
		}
		fullUser, _ := ctx.Session.User(usr.ID)
		if fullUser == nil {
			fullUser = usr
		}
		created := creationTime(usr.ID)
		joined := "Not in server"
		rolesStr := "None"
		nickname := "None"
		boostingSince := "Not Boosting"
		mem, err := ctx.Session.State.Member(gid, usr.ID)
		if err != nil {
			mem, _ = ctx.Session.GuildMember(gid, usr.ID)
		}
		if mem != nil {
			if !mem.JoinedAt.IsZero() {
				joined = fmt.Sprintf("<t:%d:F> (<t:%d:R>)", mem.JoinedAt.Unix(), mem.JoinedAt.Unix())
			}
			if mem.Nick != "" {
				nickname = mem.Nick
			}
			if mem.PremiumSince != nil {
				boostingSince = fmt.Sprintf("Boosting since <t:%d:F> (<t:%d:R>)", mem.PremiumSince.Unix(), mem.PremiumSince.Unix())
			}
			if len(mem.Roles) > 0 {
				var mentionList []string
				for _, r := range mem.Roles {
					mentionList = append(mentionList, "<@&"+r+">")
				}
				rolesStr = strings.Join(mentionList, ", ")
			}
		}
		badges := decodeBadges(int(fullUser.PublicFlags))
		chanPerms := "None"
		p, err := ctx.UserChannelPermissions(usr.ID, ctx.ChanID())
		if err == nil {
			chanPerms = decodePerms(p)
		}
		fields := []*discordgo.MessageEmbedField{
			config.Field("Username", usr.Username, true),
			config.Field("Nickname", nickname, true),
			config.Field("User ID", fmt.Sprintf("`%s`", usr.ID), true),
			config.Field("Bot User", fmt.Sprintf("`%t`", usr.Bot), true),
			config.Field("Registered At", fmt.Sprintf("<t:%d:F> (<t:%d:R>)", created.Unix(), created.Unix()), false),
			config.Field("Joined Server At", joined, false),
			config.Field("Premium Boosting Status", boostingSince, false),
			config.Field("Public Badges", badges, false),
			config.Field("Roles List", rolesStr, false),
			config.Field("Key Permissions in Channel", fmt.Sprintf("```\n%s\n```", chanPerms), false),
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:        fmt.Sprintf("User Info - %s", usr.Username),
			Fields:       fields,
			ThumbnailURL: usr.AvatarURL("256"),
		})
		return ctx.Respond(emb)
	},
}
var Pfp = &manager.Command{
	Trigger:     "pfp",
	Aliases:     []string{"avatar", "av"},
	Name:        "pfp",
	Description: "Shows a user's profile picture",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()
		query := ""
		if len(ctx.Args) > 0 {
			query = ctx.Args[0]
		} else {
			query = ctx.AuthorID()
		}
		usr, err := resolveUser(ctx.Session, gid, query)
		if err != nil || usr == nil {
			return ctx.Reply("[!] Could not resolve user.")
		}
		url := usr.AvatarURL("2048")
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:    fmt.Sprintf("%s's Profile Picture", usr.Username),
			ImageURL: url,
		})
		return ctx.Respond(emb)
	},
}
var Banner = &manager.Command{
	Trigger:     "banner",
	Name:        "banner",
	Description: "Displays the server's banner or a user's banner",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()
		if len(ctx.Args) == 0 {
			g, err := ctx.Session.State.Guild(gid)
			if err != nil {
				g, err = ctx.Session.Guild(gid)
			}
			if err != nil || g.Banner == "" {
				return ctx.Reply("[!] Server does not have a banner set.")
			}
			url := g.BannerURL("2048")
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:    fmt.Sprintf("%s's Server Banner", g.Name),
				ImageURL: url,
			})
			return ctx.Respond(emb)
		}
		usr, err := resolveUser(ctx.Session, gid, ctx.Args[0])
		if err != nil || usr == nil {
			return ctx.Reply("[!] Could not resolve user.")
		}
		fullUser, err := ctx.Session.User(usr.ID)
		if err != nil || fullUser.Banner == "" {
			return ctx.Reply(fmt.Sprintf("[!] **%s** does not have a banner set.", usr.Username))
		}
		url := fullUser.BannerURL("2048")
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:    fmt.Sprintf("%s's User Banner", usr.Username),
			ImageURL: url,
		})
		return ctx.Respond(emb)
	},
}