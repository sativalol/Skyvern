package general

import (
	"fmt"
	"regexp"
	"skyvern/internal/manager"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var rxReactRole = regexp.MustCompile(`^<@&(\d+)>$`)
var rxReactMsgLink = regexp.MustCompile(`channels/(?:\d+|@me)/(\d+)/(\d+)`)

func resolveReactRole(s *discordgo.Session, gid, query string) string {
	q := strings.TrimSpace(query)
	if q == "" {
		return ""
	}
	if m := rxReactRole.FindStringSubmatch(q); len(m) > 1 {
		return m[1]
	}
	roles, err := s.GuildRoles(gid)
	if err != nil {
		return ""
	}
	for _, r := range roles {
		if r.ID == q {
			return r.ID
		}
	}
	ql := strings.ToLower(q)
	for _, r := range roles {
		if strings.ToLower(r.Name) == ql {
			return r.ID
		}
	}
	return ""
}

func init() {
	manager.RegisterHelp("reactionrole", []manager.HelpPage{
		{
			Command:     "Reaction Role Help",
			Syntax:      ".reactionrole",
			Description: "Set up and manage reaction roles.",
		},
		{
			Command:     "Reaction Role Add",
			Syntax:      ".reactionrole add <message link> <reaction> <role>",
			Description: "Add a reaction role to a message.",
		},
		{
			Command:     "Reaction Role List",
			Syntax:      ".reactionrole list",
			Description: "View all reaction roles in the server.",
		},
		{
			Command:     "Reaction Role Reset",
			Syntax:      ".reactionrole reset",
			Description: "Clears all reaction roles from the server.",
		},
		{
			Command:     "Reaction Role Remove All",
			Syntax:      ".reactionrole removeall <message link>",
			Description: "Removes all reaction roles from a message.",
		},
		{
			Command:     "Reaction Role Remove Single",
			Syntax:      ".reactionrole remove <message link> <reaction>",
			Description: "Removes a specific reaction role from a message.",
		},
		{
			Command:     "Reaction Role Restore Settings",
			Syntax:      ".reactionrole restore <on | off>",
			Description: "Choose whether reaction roles restore when members rejoin.",
		},
	})
}

var ReactionRole = &manager.Command{
	Trigger:     "reactionrole",
	Aliases:     []string{"reactrole", "rr"},
	Name:        "reactionrole",
	Description: "Configure reaction roles on messages",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("reactionrole")
		}

		sub := strings.ToLower(ctx.Args[0])
		gid := ctx.GuildID()

		switch sub {
		case "add":
			if len(ctx.Args) < 4 {
				return ctx.Reply("Usage: `.reactionrole add <message link> <reaction> <role>`")
			}
			link := ctx.Args[1]
			emoji := ctx.Args[2]
			roleArg := ctx.Args[3]

			parts := rxReactMsgLink.FindStringSubmatch(link)
			if len(parts) < 3 {
				return ctx.Reply("[!] Invalid message link.")
			}
			chanID := parts[1]
			msgID := parts[2]

			rid := resolveReactRole(ctx.Session, gid, roleArg)
			if rid == "" {
				return ctx.Reply("[!] Could not resolve role.")
			}

			roles, err := ctx.Session.GuildRoles(gid)
			if err != nil {
				return ctx.Reply("[!] Failed to fetch guild roles.")
			}

			var targetRole *discordgo.Role
			for _, r := range roles {
				if r.ID == rid {
					targetRole = r
					break
				}
			}
			if targetRole == nil {
				return ctx.Reply("[!] Role not found in server.")
			}

			botMember, err := ctx.Session.GuildMember(gid, ctx.ClientID)
			if err != nil {
				return ctx.Reply("[!] Failed to verify bot status.")
			}

			botMaxPos := -1
			for _, r := range roles {
				for _, botRoleID := range botMember.Roles {
					if r.ID == botRoleID && r.Position > botMaxPos {
						botMaxPos = r.Position
					}
				}
			}

			if targetRole.Position >= botMaxPos {
				return ctx.Reply("[!] Security Alert: Target role is higher than or equal to the bot's own role. Action blocked.")
			}

			err = ctx.DB.SaveReactRole(gid, msgID, emoji, rid)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to save reaction role: %v", err))
			}

			err = ctx.Session.MessageReactionAdd(chanID, msgID, emoji)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[+] Set reaction role, but could not react to message: %v.", err))
			}

			return ctx.Reply(fmt.Sprintf("[+] Set reaction role on message `%s` with emoji %s assigning <@&%s>.", msgID, emoji, rid))

		case "list":
			if !checkPerm(ctx, discordgo.PermissionManageRoles) {
				return ctx.Reply("[!] You need Manage Roles permission.")
			}
			list, err := ctx.DB.ListReactRoles(gid)
			if err != nil || len(list) == 0 {
				return ctx.Reply("[*] No reaction roles configured for this server.")
			}
			var sb strings.Builder
			sb.WriteString("**Configured Reaction Roles:**\n\n")
			for msgID, emojis := range list {
				sb.WriteString(fmt.Sprintf("**Message ID: `%s`**\n", msgID))
				for emoji, roleID := range emojis {
					sb.WriteString(fmt.Sprintf("- %s -> <@&%s>\n", emoji, roleID))
				}
				sb.WriteString("\n")
			}
			return ctx.Reply(sb.String())

		case "remove":
			if !checkPerm(ctx, discordgo.PermissionManageRoles) {
				return ctx.Reply("[!] You need Manage Roles permission.")
			}
			if len(ctx.Args) < 3 {
				return ctx.Reply("Usage: `.reactionrole remove <message link> <reaction>`")
			}
			link := ctx.Args[1]
			emoji := ctx.Args[2]

			parts := rxReactMsgLink.FindStringSubmatch(link)
			if len(parts) < 3 {
				return ctx.Reply("[!] Invalid message link.")
			}
			msgID := parts[2]

			err := ctx.DB.DeleteReactRole(gid, msgID, emoji)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to delete reaction role: %v", err))
			}

			return ctx.Reply(fmt.Sprintf("[+] Removed reaction role on message `%s` with emoji %s.", msgID, emoji))

		case "removeall":
			if !checkPerm(ctx, discordgo.PermissionManageRoles) {
				return ctx.Reply("[!] You need Manage Roles permission.")
			}
			if len(ctx.Args) < 2 {
				return ctx.Reply("Usage: `.reactionrole removeall <message link>`")
			}
			link := ctx.Args[1]

			parts := rxReactMsgLink.FindStringSubmatch(link)
			if len(parts) < 3 {
				return ctx.Reply("[!] Invalid message link.")
			}
			msgID := parts[2]

			_ = ctx.DB.DeleteAllReactRolesForMsg(gid, msgID)
			return ctx.Reply(fmt.Sprintf("[+] Removed all reaction roles on message `%s`.", msgID))

		case "reset":
			if !checkPerm(ctx, discordgo.PermissionManageRoles) {
				return ctx.Reply("[!] You need Manage Roles permission.")
			}
			_ = ctx.DB.ClearReactRoles(gid)
			return ctx.Reply("[+] Cleared all reaction role registrations.")

		case "restore":
			if !checkPerm(ctx, discordgo.PermissionManageRoles) {
				return ctx.Reply("[!] You need Manage Roles permission.")
			}
			if len(ctx.Args) < 2 {
				return ctx.Reply("Usage: `.reactionrole restore <on | off>`")
			}
			opt := strings.ToLower(ctx.Args[1])
			cfg, _ := ctx.DB.GetGuildSettings(gid)

			if opt == "on" || opt == "true" || opt == "enable" {
				cfg.ReactionRolesRestore = true
			} else if opt == "off" || opt == "false" || opt == "disable" {
				cfg.ReactionRolesRestore = false
			} else {
				return ctx.Reply("[!] Invalid option. Use on/off or true/false.")
			}

			_ = ctx.DB.SaveGuildSettings(gid, cfg)
			return ctx.Reply(fmt.Sprintf("[+] Reaction roles restore set to `%t`.", cfg.ReactionRolesRestore))

		default:
			return ctx.SendHelp("reactionrole")
		}
	},
}
