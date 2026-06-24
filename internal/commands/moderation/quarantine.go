package moderation

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/moderation"
)

func init() {
	manager.RegisterHelp("quarantine", []manager.HelpPage{
		{
			Command:     "Quarantine User",
			Syntax:      ".quarantine <user> [reason]",
			Description: "Isolate a user by stripping roles and applying the quarantine role.",
		},
		{
			Command:     "Release User",
			Syntax:      ".release <user>",
			Description: "Restore a quarantined user's original roles.",
		},
		{
			Command:     "Quarantine List",
			Syntax:      ".quarantined",
			Description: "List all currently quarantined users in the server.",
		},
		{
			Command:     "Quarantine Setup",
			Syntax:      ".quarantine setup",
			Description: "Force setup or sync the quarantine role and channel overrides.",
		},
	})
	manager.RegisterHelp("release", []manager.HelpPage{
		{
			Command:     "Release User",
			Syntax:      ".release <user>",
			Description: "Restore a quarantined user's original roles.",
		},
	})
	manager.RegisterHelp("quarantined", []manager.HelpPage{
		{
			Command:     "Quarantine List",
			Syntax:      ".quarantined",
			Description: "List all currently quarantined users in the server.",
		},
	})
}

var Quarantine = &manager.Command{
	Trigger:     "quarantine",
	Aliases:     []string{"q"},
	Name:        "quarantine",
	Description: "Quarantine a member, stripping roles and applying quarantine role",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		if !checkPerm(ctx, discordgo.PermissionManageGuild) {
			return ctx.Reply("[!] You need Manage Guild permission to use this command.")
		}
		if len(ctx.Args) == 0 {
			return ctx.Reply("Usage: .quarantine <user> [reason] or .quarantine setup")
		}

		gid := ctx.GuildID()
		sub := strings.ToLower(ctx.Args[0])
		if sub == "setup" {
			qRoleID, err := ctx.Mgr.EnsureQuarantineRole(ctx.Session, gid)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to setup quarantine role: %v", err))
			}
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:       "Quarantine Setup Completed",
				Description: fmt.Sprintf("Created/verified quarantine role and synced permission overrides.\nRole ID: `%s`", qRoleID),
			})
			emb.Color = 0x708090
			return ctx.Respond(emb)
		}

		m, err := resolveMemberOrReply(ctx, ctx.Args[0])
		if err != nil {
			return nil
		}

		if !checkHierarchy(ctx, m.User.ID) {
			return ctx.Reply("[!] You cannot moderate this member due to role hierarchy.")
		}

		reason := "No reason provided."
		if len(ctx.Args) > 1 {
			reason = strings.Join(ctx.Args[1:], " ")
		}

		err = ctx.Mgr.QuarantineUser(ctx.Session, gid, m.User.ID, reason, ctx.AuthorID())
		if err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Failed to quarantine member: %v", err))
		}

		moderation.LogAction(ctx.Session, ctx.DB, gid, "Quarantine", ctx.AuthorID(), m.User.ID, reason)

		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "User Quarantined",
			Description: fmt.Sprintf("User **%s** has been quarantined.", m.User.Username),
			Fields: []*discordgo.MessageEmbedField{
				config.Field("User", fmt.Sprintf("<@%s> (`%s`)", m.User.ID, m.User.ID), true),
				config.Field("Moderator", fmt.Sprintf("<@%s>", ctx.AuthorID()), true),
				config.Field("Reason", reason, false),
			},
		})
		emb.Color = 0x708090
		return ctx.Respond(emb)
	},
}

var Release = &manager.Command{
	Trigger:     "release",
	Aliases:     []string{"unquarantine", "uq"},
	Name:        "release",
	Description: "Release a quarantined member and restore roles",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		if !checkPerm(ctx, discordgo.PermissionManageGuild) {
			return ctx.Reply("[!] You need Manage Guild permission to use this command.")
		}
		if len(ctx.Args) == 0 {
			return ctx.Reply("Usage: .release <user>")
		}

		gid := ctx.GuildID()
		m, err := resolveMemberOrReply(ctx, ctx.Args[0])
		if err != nil {
			return nil
		}

		if !checkHierarchy(ctx, m.User.ID) {
			return ctx.Reply("[!] You cannot moderate this member due to role hierarchy.")
		}

		err = ctx.Mgr.ReleaseUser(ctx.Session, gid, m.User.ID)
		if err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Failed to release member: %v", err))
		}

		moderation.LogAction(ctx.Session, ctx.DB, gid, "Release", ctx.AuthorID(), m.User.ID, "User released from quarantine")

		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "User Released",
			Description: fmt.Sprintf("User **%s** has been released from quarantine and roles restored.", m.User.Username),
			Fields: []*discordgo.MessageEmbedField{
				config.Field("User", fmt.Sprintf("<@%s> (`%s`)", m.User.ID, m.User.ID), true),
				config.Field("Moderator", fmt.Sprintf("<@%s>", ctx.AuthorID()), true),
			},
		})
		emb.Color = 0x708090
		return ctx.Respond(emb)
	},
}

var Quarantined = &manager.Command{
	Trigger:     "quarantined",
	Aliases:     []string{"ql", "quarantinelist"},
	Name:        "quarantined",
	Description: "List all currently quarantined members",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		if !checkPerm(ctx, discordgo.PermissionManageGuild) {
			return ctx.Reply("[!] You need Manage Guild permission to view quarantined list.")
		}

		gid := ctx.GuildID()
		list, err := ctx.DB.ListQuarantined(gid)
		if err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Failed to fetch quarantined users: %v", err))
		}

		if len(list) == 0 {
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:       "Quarantined Members",
				Description: "There are no quarantined members in this server.",
			})
			emb.Color = 0x708090
			return ctx.Respond(emb)
		}

		var sb strings.Builder
		for _, entry := range list {
			uName := entry.UserID
			if u, err := ctx.Session.User(entry.UserID); err == nil && u != nil {
				uName = u.Username
			}
			byName := entry.By
			if u, err := ctx.Session.User(entry.By); err == nil && u != nil {
				byName = u.Username
			}
			sb.WriteString(fmt.Sprintf("**%s** (`%s`) by **%s** on %s\nReason: %s\n\n",
				uName, entry.UserID, byName, entry.Timestamp.Format("2006-01-02 15:04:05"), entry.Reason))
		}

		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "Quarantined Members",
			Description: sb.String(),
		})
		emb.Color = 0x708090
		return ctx.Respond(emb)
	},
}
