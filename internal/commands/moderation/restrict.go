package moderation

import (
	"fmt"
	"skyvern/internal/manager"
	"skyvern/internal/storage"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func init() {
	manager.RegisterHelp("restrict", []manager.HelpPage{
		{
			Command:     "Restrict Toggle",
			Syntax:      ".restrict <command>",
			Description: "Toggle disabling a command server-wide.",
		},
		{
			Command:     "Restrict Whitelist Channel",
			Syntax:      ".restrict <command> whitelist <channel>",
			Description: "Restrict a command to only work in a specific channel.",
		},
		{
			Command:     "Restrict Blacklist Channel",
			Syntax:      ".restrict <command> blacklist <channel>",
			Description: "Prevent a command from working in a specific channel.",
		},
		{
			Command:     "Restrict Role",
			Syntax:      ".restrict <command> role <@role>",
			Description: "Restrict command execution to users with a specific role.",
		},
		{
			Command:     "Restrict Clear",
			Syntax:      ".restrict <command> clear",
			Description: "Clear all restrictions for a command.",
		},
		{
			Command:     "Restrict List",
			Syntax:      ".restrict list",
			Description: "List all restricted commands in this server.",
		},
	})
}

var Restrict = &manager.Command{
	Trigger:     "restrict",
	Aliases:     []string{"restrictcommand", "rc"},
	Name:        "restrict",
	Description: "Manage command restrictions",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		if !checkPerm(ctx, discordgo.PermissionManageServer) {
			return ctx.Reply("[!] You need Manage Server permission.")
		}
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("restrict")
		}

		gid := ctx.GuildID()
		sub := strings.ToLower(ctx.Args[0])

		if sub == "list" {
			m, err := ctx.DB.ListCmdRestrictions(gid)
			if err != nil || len(m) == 0 {
				return ctx.Reply("[*] No command restrictions configured.")
			}
			var sb strings.Builder
			sb.WriteString("Command Restrictions:\n\n")
			for cmdName, rest := range m {
				var parts []string
				if rest.ServerDisabled {
					parts = append(parts, "Disabled server-wide")
				}
				if rest.RoleID != "" {
					parts = append(parts, fmt.Sprintf("Role: <@&%s>", rest.RoleID))
				}
				if len(rest.WhitelistChans) > 0 {
					var mentions []string
					for _, c := range rest.WhitelistChans {
						mentions = append(mentions, fmt.Sprintf("<#%s>", c))
					}
					parts = append(parts, "Only in: "+strings.Join(mentions, ", "))
				}
				if len(rest.BlacklistChans) > 0 {
					var mentions []string
					for _, c := range rest.BlacklistChans {
						mentions = append(mentions, fmt.Sprintf("<#%s>", c))
					}
					parts = append(parts, "Disabled in: "+strings.Join(mentions, ", "))
				}
				if len(parts) > 0 {
					sb.WriteString(fmt.Sprintf("- `%s`: %s\n", cmdName, strings.Join(parts, " | ")))
				}
			}
			if sb.Len() == 22 {
				return ctx.Reply("[*] No active command restrictions configured.")
			}
			return ctx.Reply(sb.String())
		}

		cmd := ctx.Mgr.FindCommand(sub)
		if cmd == nil {
			return ctx.Reply(fmt.Sprintf("[!] Unknown command `%s`.", sub))
		}
		if cmd.Trigger == "restrict" || cmd.Trigger == "owner" {
			return ctx.Reply("[!] You cannot restrict this command.")
		}

		rest, _ := ctx.DB.GetCmdRestriction(gid, cmd.Trigger)

		if len(ctx.Args) == 1 {
			rest.ServerDisabled = !rest.ServerDisabled
			_ = ctx.DB.SaveCmdRestriction(gid, cmd.Trigger, rest)
			if rest.ServerDisabled {
				return ctx.Reply(fmt.Sprintf("[+] Command `%s` is now disabled server-wide.", cmd.Trigger))
			}
			return ctx.Reply(fmt.Sprintf("[+] Command `%s` is no longer disabled server-wide.", cmd.Trigger))
		}

		action := strings.ToLower(ctx.Args[1])
		switch action {
		case "clear", "reset":
			_ = ctx.DB.SaveCmdRestriction(gid, cmd.Trigger, storage.CmdRestriction{})
			return ctx.Reply(fmt.Sprintf("[+] Cleared all restrictions for command `%s`.", cmd.Trigger))

		case "whitelist", "only":
			if len(ctx.Args) < 3 {
				return ctx.Reply(fmt.Sprintf("Usage: `.restrict %s whitelist <channel>`", cmd.Trigger))
			}
			cid, err := resolveChannelOrReply(ctx, ctx.Args[2])
			if err != nil {
				return nil
			}
			found := -1
			for idx, c := range rest.WhitelistChans {
				if c == cid {
					found = idx
					break
				}
			}
			if found >= 0 {
				rest.WhitelistChans = append(rest.WhitelistChans[:found], rest.WhitelistChans[found+1:]...)
				_ = ctx.DB.SaveCmdRestriction(gid, cmd.Trigger, rest)
				return ctx.Reply(fmt.Sprintf("[+] Command `%s` is no longer whitelisted for <#%s>.", cmd.Trigger, cid))
			} else {
				for idx, c := range rest.BlacklistChans {
					if c == cid {
						rest.BlacklistChans = append(rest.BlacklistChans[:idx], rest.BlacklistChans[idx+1:]...)
						break
					}
				}
				rest.WhitelistChans = append(rest.WhitelistChans, cid)
				_ = ctx.DB.SaveCmdRestriction(gid, cmd.Trigger, rest)
				return ctx.Reply(fmt.Sprintf("[+] Command `%s` is now whitelisted for <#%s>.", cmd.Trigger, cid))
			}

		case "blacklist", "disable":
			if len(ctx.Args) < 3 {
				return ctx.Reply(fmt.Sprintf("Usage: `.restrict %s blacklist <channel>`", cmd.Trigger))
			}
			cid, err := resolveChannelOrReply(ctx, ctx.Args[2])
			if err != nil {
				return nil
			}
			found := -1
			for idx, c := range rest.BlacklistChans {
				if c == cid {
					found = idx
					break
				}
			}
			if found >= 0 {
				rest.BlacklistChans = append(rest.BlacklistChans[:found], rest.BlacklistChans[found+1:]...)
				_ = ctx.DB.SaveCmdRestriction(gid, cmd.Trigger, rest)
				return ctx.Reply(fmt.Sprintf("[+] Command `%s` is no longer blacklisted for <#%s>.", cmd.Trigger, cid))
			} else {
				for idx, c := range rest.WhitelistChans {
					if c == cid {
						rest.WhitelistChans = append(rest.WhitelistChans[:idx], rest.WhitelistChans[idx+1:]...)
						break
					}
				}
				rest.BlacklistChans = append(rest.BlacklistChans, cid)
				_ = ctx.DB.SaveCmdRestriction(gid, cmd.Trigger, rest)
				return ctx.Reply(fmt.Sprintf("[+] Command `%s` is now blacklisted in <#%s>.", cmd.Trigger, cid))
			}

		case "role":
			if len(ctx.Args) < 3 {
				return ctx.Reply(fmt.Sprintf("Usage: `.restrict %s role <@role>`", cmd.Trigger))
			}
			val := ctx.Args[2]
			if val == "clear" || val == "none" || val == "reset" {
				rest.RoleID = ""
				_ = ctx.DB.SaveCmdRestriction(gid, cmd.Trigger, rest)
				return ctx.Reply(fmt.Sprintf("[+] Removed role restriction from command `%s`.", cmd.Trigger))
			}
			rid, err := resolveRoleOrReply(ctx, val)
			if err != nil {
				return nil
			}
			rest.RoleID = rid
			_ = ctx.DB.SaveCmdRestriction(gid, cmd.Trigger, rest)
			return ctx.Reply(fmt.Sprintf("[+] Command `%s` restricted to <@&%s>.", cmd.Trigger, rid))

		default:
			if cid, err := resolveChannelOrReply(ctx, ctx.Args[1]); err == nil && cid != "" {
				found := -1
				for idx, c := range rest.BlacklistChans {
					if c == cid {
						found = idx
						break
					}
				}
				if found >= 0 {
					rest.BlacklistChans = append(rest.BlacklistChans[:found], rest.BlacklistChans[found+1:]...)
					_ = ctx.DB.SaveCmdRestriction(gid, cmd.Trigger, rest)
					return ctx.Reply(fmt.Sprintf("[+] Command `%s` is no longer blacklisted for <#%s>.", cmd.Trigger, cid))
				} else {
					for idx, c := range rest.WhitelistChans {
						if c == cid {
							rest.WhitelistChans = append(rest.WhitelistChans[:idx], rest.WhitelistChans[idx+1:]...)
							break
						}
					}
					rest.BlacklistChans = append(rest.BlacklistChans, cid)
					_ = ctx.DB.SaveCmdRestriction(gid, cmd.Trigger, rest)
					return ctx.Reply(fmt.Sprintf("[+] Command `%s` is now blacklisted in <#%s>.", cmd.Trigger, cid))
				}
			}
			return ctx.SendHelp("restrict")
		}
	},
}
