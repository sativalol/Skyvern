package general
import (
	"fmt"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"strings"
	"github.com/bwmarrin/discordgo"
)
func init() {
	manager.RegisterHelp("membercount", []manager.HelpPage{
		{
			Command:     "Member Count",
			Syntax:      ".membercount",
			Description: "Displays details on server size including humans/bots split.",
		},
	})
	manager.RegisterHelp("channelinfo", []manager.HelpPage{
		{
			Command:     "Channel Info",
			Syntax:      ".channelinfo <channel>",
			Description: "View metadata for a channel.",
		},
	})
	manager.RegisterHelp("serveravatar", []manager.HelpPage{
		{
			Command:     "Server Avatar",
			Syntax:      ".serveravatar [user]",
			Description: "Get the server-specific avatar of a member.",
		},
	})
	manager.RegisterHelp("serverbanner", []manager.HelpPage{
		{
			Command:     "Server Banner",
			Syntax:      ".serverbanner [user]",
			Description: "Get the server-specific banner of a member.",
		},
	})
	manager.RegisterHelp("guildicon", []manager.HelpPage{
		{
			Command:     "Guild Icon",
			Syntax:      ".guildicon [guild_id]",
			Description: "Returns the icon of the server.",
		},
	})
	manager.RegisterHelp("guildbanner", []manager.HelpPage{
		{
			Command:     "Guild Banner",
			Syntax:      ".guildbanner [guild_id]",
			Description: "Returns the banner of the server.",
		},
	})
	manager.RegisterHelp("splash", []manager.HelpPage{
		{
			Command:     "Guild Splash",
			Syntax:      ".splash [guild_id]",
			Description: "Returns the invite splash background of the server.",
		},
	})
}
var MemberCount = &manager.Command{
	Trigger:     "membercount",
	Aliases:     []string{"mc"},
	Name:        "membercount",
	Description: "View server member count details",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()
		var err error
		if _, err = ctx.Session.State.Guild(gid); err != nil {
			if _, err = ctx.Session.Guild(gid); err != nil {
				return ctx.Reply("[!] Failed to fetch guild details.")
			}
		}
		members, err := ctx.Session.GuildMembers(gid, "", 1000)
		if err != nil {
			return ctx.Reply("[!] Failed to fetch guild members list.")
		}
		humans := 0
		bots := 0
		for _, m := range members {
			if m.User != nil && m.User.Bot {
				bots++
			} else {
				humans++
			}
		}
		fields := []*discordgo.MessageEmbedField{
			config.Field("Total Members", fmt.Sprintf("%d", len(members)), true),
			config.Field("Humans", fmt.Sprintf("%d", humans), true),
			config.Field("Bots", fmt.Sprintf("%d", bots), true),
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:  "Server Member Count",
			Fields: fields,
		})
		return ctx.Respond(emb)
	},
}
var ChannelInfo = &manager.Command{
	Trigger:     "channelinfo",
	Aliases:     []string{"ci"},
	Name:        "channelinfo",
	Description: "View information about a channel",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		cid := ctx.ChanID()
		if len(ctx.Args) > 0 {
			raw := ctx.Args[0]
			if strings.HasPrefix(raw, "<#") && strings.HasSuffix(raw, ">") {
				cid = strings.Trim(raw, "<#>")
			} else {
				cid = raw
			}
		}
		ch, err := ctx.Session.Channel(cid)
		if err != nil {
			return ctx.Reply("[!] Channel not found.")
		}
		created := creationTime(ch.ID)
		chType := "Text"
		switch ch.Type {
		case discordgo.ChannelTypeGuildVoice:
			chType = "Voice"
		case discordgo.ChannelTypeGuildCategory:
			chType = "Category"
		case discordgo.ChannelTypeGuildNews:
			chType = "Announcement"
		case discordgo.ChannelTypeGuildStageVoice:
			chType = "Stage"
		}
		var fields []*discordgo.MessageEmbedField
		fields = append(fields, config.Field("Name", ch.Name, true))
		fields = append(fields, config.Field("ID", fmt.Sprintf("`%s`", ch.ID), true))
		fields = append(fields, config.Field("Type", chType, true))
		fields = append(fields, config.Field("Position", fmt.Sprintf("%d", ch.Position), true))
		if ch.ParentID != "" {
			parentName := ch.ParentID
			parentChan, err := ctx.Session.Channel(ch.ParentID)
			if err == nil {
				parentName = fmt.Sprintf("%s (`%s`)", parentChan.Name, parentChan.ID)
			}
			fields = append(fields, config.Field("Category", parentName, false))
		}
		if ch.Type == discordgo.ChannelTypeGuildVoice || ch.Type == discordgo.ChannelTypeGuildStageVoice {
			fields = append(fields, config.Field("Bitrate", fmt.Sprintf("%d kbps", ch.Bitrate/1000), true))
			limitStr := "Unlimited"
			if ch.UserLimit > 0 {
				limitStr = fmt.Sprintf("%d users", ch.UserLimit)
			}
			fields = append(fields, config.Field("User Limit", limitStr, true))
		} else {
			nsfwStr := "False"
			if ch.NSFW {
				nsfwStr = "True"
			}
			fields = append(fields, config.Field("NSFW Filter", nsfwStr, true))
			slowmodeStr := "Disabled"
			if ch.RateLimitPerUser > 0 {
				slowmodeStr = fmt.Sprintf("%d seconds", ch.RateLimitPerUser)
			}
			fields = append(fields, config.Field("Slowmode", slowmodeStr, true))
		}
		roleOverwrites := 0
		memberOverwrites := 0
		for _, ow := range ch.PermissionOverwrites {
			if ow.Type == discordgo.PermissionOverwriteTypeRole {
				roleOverwrites++
			} else if ow.Type == discordgo.PermissionOverwriteTypeMember {
				memberOverwrites++
			}
		}
		fields = append(fields, config.Field("Permission Overwrites", fmt.Sprintf("Roles: **%d** | Members: **%d**", roleOverwrites, memberOverwrites), false))
		fields = append(fields, config.Field("Created At", fmt.Sprintf("<t:%d:F> (<t:%d:R>)", created.Unix(), created.Unix()), false))
		topic := ch.Topic
		if topic == "" {
			topic = "No topic set."
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "Channel Information: " + ch.Name,
			Description: fmt.Sprintf("**Topic:** %s", topic),
			Fields:      fields,
		})
		return ctx.Respond(emb)
	},
}
var ServerAvatar = &manager.Command{
	Trigger:     "serveravatar",
	Aliases:     []string{"sav"},
	Name:        "serveravatar",
	Description: "Get the server avatar of a member or yourself",
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
		mem, err := ctx.Session.State.Member(gid, usr.ID)
		if err != nil {
			mem, _ = ctx.Session.GuildMember(gid, usr.ID)
		}
		if mem == nil || mem.Avatar == "" {
			return ctx.Reply("[!] User does not have a server-specific avatar set.")
		}
		url := mem.AvatarURL("2048")
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:    fmt.Sprintf("%s's Server Avatar", usr.Username),
			ImageURL: url,
		})
		return ctx.Respond(emb)
	},
}
var ServerBanner = &manager.Command{
	Trigger:     "serverbanner",
	Aliases:     []string{"sbanner"},
	Name:        "serverbanner",
	Description: "Get the server banner of a member or yourself",
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
		fullUser, err := ctx.Session.User(usr.ID)
		if err != nil || fullUser.Banner == "" {
			return ctx.Reply("[!] User does not have a banner set.")
		}
		url := fullUser.BannerURL("2048")
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:    fmt.Sprintf("%s's Banner", usr.Username),
			ImageURL: url,
		})
		return ctx.Respond(emb)
	},
}
var GuildIcon = &manager.Command{
	Trigger:     "guildicon",
	Aliases:     []string{"icon", "gicon"},
	Name:        "guildicon",
	Description: "Returns guild icon",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		targetGID := ctx.GuildID()
		if len(ctx.Args) > 0 {
			targetGID = ctx.Args[0]
		}
		g, err := ctx.Session.State.Guild(targetGID)
		if err != nil {
			g, err = ctx.Session.Guild(targetGID)
		}
		if err != nil {
			return ctx.Reply("[!] Server not found or bot lacks access.")
		}
		if g.Icon == "" {
			return ctx.Reply("[!] Server does not have an icon.")
		}
		url := g.IconURL("2048")
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:    fmt.Sprintf("%s's Icon", g.Name),
			ImageURL: url,
		})
		return ctx.Respond(emb)
	},
}
var GuildBanner = &manager.Command{
	Trigger:     "guildbanner",
	Aliases:     []string{"gbanner"},
	Name:        "guildbanner",
	Description: "Returns banner icon",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		targetGID := ctx.GuildID()
		if len(ctx.Args) > 0 {
			targetGID = ctx.Args[0]
		}
		g, err := ctx.Session.State.Guild(targetGID)
		if err != nil {
			g, err = ctx.Session.Guild(targetGID)
		}
		if err != nil {
			return ctx.Reply("[!] Server not found or bot lacks access.")
		}
		if g.Banner == "" {
			return ctx.Reply("[!] Server does not have a banner.")
		}
		url := g.BannerURL("2048")
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:    fmt.Sprintf("%s's Banner", g.Name),
			ImageURL: url,
		})
		return ctx.Respond(emb)
	},
}
var Splash = &manager.Command{
	Trigger:     "splash",
	Name:        "splash",
	Description: "Returns splash background",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		targetGID := ctx.GuildID()
		if len(ctx.Args) > 0 {
			targetGID = ctx.Args[0]
		}
		g, err := ctx.Session.State.Guild(targetGID)
		if err != nil {
			g, err = ctx.Session.Guild(targetGID)
		}
		if err != nil {
			return ctx.Reply("[!] Server not found or bot lacks access.")
		}
		if g.Splash == "" {
			return ctx.Reply("[!] Server does not have a splash image.")
		}
		url := fmt.Sprintf("https://cdn.discordapp.com/splashes/%s/%s.png?size=2048", g.ID, g.Splash)
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:    fmt.Sprintf("%s's Splash Image", g.Name),
			ImageURL: url,
		})
		return ctx.Respond(emb)
	},
}