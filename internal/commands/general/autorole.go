package general

import (
	"fmt"
	"regexp"
	"skyvern/internal/manager"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var rxRole = regexp.MustCompile(`^<@&(\d+)>$`)

func checkPerm(ctx *manager.CommandContext, perm int64) bool {
	uid := ctx.AuthorID()
	if uid == "" {
		return false
	}
	gid := ctx.GuildID()
	g, err := ctx.Session.State.Guild(gid)
	if err != nil {
		g, err = ctx.Session.Guild(gid)
	}
	if err == nil && g.OwnerID == uid {
		return true
	}
	if ctx.DB.HasBypass(gid, uid) {
		return true
	}
	p, err := ctx.Session.UserChannelPermissions(uid, "")
	if err != nil {
		return false
	}
	if (p & discordgo.PermissionAdministrator) != 0 {
		return true
	}
	return (p & perm) == perm
}

func resolveRole(s *discordgo.Session, gid, query string) string {
	q := strings.TrimSpace(query)
	if q == "" {
		return ""
	}

	if m := rxRole.FindStringSubmatch(q); len(m) > 1 {
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

	for _, r := range roles {
		if strings.Contains(strings.ToLower(r.Name), ql) {
			return r.ID
		}
	}

	return ""
}

func init() {
	manager.RegisterHelp("autorole", []manager.HelpPage{
		{
			Command:     "Autorole Help",
			Syntax:      ".autorole",
			Description: "Set up automatic role assign on member join.",
		},
		{
			Command:     "Autorole Add",
			Syntax:      ".autorole add <role>",
			Description: "Adds an autorole and assigns it on join to members.",
		},
		{
			Command:     "Autorole List",
			Syntax:      ".autorole list",
			Description: "View a list of every auto role.",
		},
		{
			Command:     "Autorole Reset",
			Syntax:      ".autorole reset",
			Description: "Clears every autorole for the guild.",
		},
		{
			Command:     "Autorole Remove",
			Syntax:      ".autorole remove <role>",
			Description: "Removes an autorole and stops assigning it on join.",
		},
	})
}

var Autorole = &manager.Command{
	Trigger:     "autorole",
	Name:        "autorole",
	Description: "Configure roles given to users automatically upon joining",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("autorole")
		}

		gid := ctx.GuildID()
		sub := strings.ToLower(ctx.Args[0])

		switch sub {
		case "add":
			if len(ctx.Args) < 2 {
				return ctx.Reply("[!] Please specify a role to add.")
			}
			roleArg := ctx.Args[1]
			rid := resolveRole(ctx.Session, gid, roleArg)
			if rid == "" {
				return ctx.Reply("[!] Could not resolve role.")
			}

			roles, _ := ctx.DB.GetAutoroles(gid)
			for _, r := range roles {
				if r == rid {
					return ctx.Reply("[!] Role is already in the autorole list.")
				}
			}
			roles = append(roles, rid)
			_ = ctx.DB.SaveAutoroles(gid, roles)
			return ctx.Reply(fmt.Sprintf("[+] Added role ID `%s` to autoroles.", rid))

		case "remove":
			if !checkPerm(ctx, discordgo.PermissionManageRoles) {
				return ctx.Reply("[!] You need Manage Roles permission.")
			}
			if len(ctx.Args) < 2 {
				return ctx.Reply("[!] Please specify a role to remove.")
			}
			roleArg := ctx.Args[1]
			rid := resolveRole(ctx.Session, gid, roleArg)
			if rid == "" {
				return ctx.Reply("[!] Could not resolve role.")
			}

			roles, _ := ctx.DB.GetAutoroles(gid)
			found := false
			var next []string
			for _, r := range roles {
				if r == rid {
					found = true
				} else {
					next = append(next, r)
				}
			}
			if !found {
				return ctx.Reply("[!] Role is not in the autorole list.")
			}
			_ = ctx.DB.SaveAutoroles(gid, next)
			return ctx.Reply(fmt.Sprintf("[+] Removed role ID `%s` from autoroles.", rid))

		case "list":
			if !checkPerm(ctx, discordgo.PermissionManageRoles) {
				return ctx.Reply("[!] You need Manage Roles permission.")
			}
			roles, err := ctx.DB.GetAutoroles(gid)
			if err != nil || len(roles) == 0 {
				return ctx.Reply("[*] No autoroles configured for this server.")
			}
			var sb strings.Builder
			sb.WriteString("Configured Autoroles:\n\n")
			for _, rid := range roles {
				sb.WriteString(fmt.Sprintf("- <@&%s> (`%s`)\n", rid, rid))
			}
			return ctx.Reply(sb.String())

		case "reset":
			if !checkPerm(ctx, discordgo.PermissionManageRoles) {
				return ctx.Reply("[!] You need Manage Roles permission.")
			}
			_ = ctx.DB.SaveAutoroles(gid, nil)
			return ctx.Reply("[+] All autoroles have been reset.")

		default:
			return ctx.SendHelp("autorole")
		}
	},
}
