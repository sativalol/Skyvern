package customcommands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/plugins"
	"skyvern/internal/storage"
)

func init() {
	plugins.Register(&CustomCommandsPlugin{})

	manager.RegisterHelp("customcmd", []manager.HelpPage{
		{
			Command:     "Create Custom Command",
			Syntax:      ".customcmd create <trigger> [description]",
			Description: "Create a new custom command trigger.",
		},
		{
			Command:     "Delete Custom Command",
			Syntax:      ".customcmd delete <trigger>",
			Description: "Delete a custom command trigger.",
		},
		{
			Command:     "Edit Add Action",
			Syntax:      ".customcmd edit <trigger> action add <type> <params...>",
			Description: "Add a sequential action. Supported types: send_message, add_role, remove_role, quarantine, dm. Params are specified as key=value.",
		},
		{
			Command:     "Edit Clear Actions",
			Syntax:      ".customcmd edit <trigger> action clear",
			Description: "Clear all actions from a custom command.",
		},
		{
			Command:     "Edit Required Permissions",
			Syntax:      ".customcmd edit <trigger> reqs perms <permission_name>",
			Description: "Set a required permission for executing the command (e.g., manage_messages, administrator, none).",
		},
		{
			Command:     "Edit Allowed Roles",
			Syntax:      ".customcmd edit <trigger> reqs role <role_id>",
			Description: "Add a role that is allowed to run the command.",
		},
		{
			Command:     "Edit Clear Allowed Roles",
			Syntax:      ".customcmd edit <trigger> reqs role clear",
			Description: "Clear all role requirements.",
		},
		{
			Command:     "Edit Bypass Execution Permissions",
			Syntax:      ".customcmd edit <trigger> bypass <on|off>",
			Description: "Enable or disable bypassing role/permission checks when running the action (owner only).",
		},
		{
			Command:     "List Custom Commands",
			Syntax:      ".customcmd list",
			Description: "List all custom commands configured in the server.",
		},
		{
			Command:     "Custom Command Details",
			Syntax:      ".customcmd info <trigger>",
			Description: "Show details and actions of a custom command.",
		},
	})
}

type CustomCommandsPlugin struct {
	db  *storage.DB
	mgr *manager.Manager
}

func (p *CustomCommandsPlugin) Name() string {
	return "customcommands"
}

func (p *CustomCommandsPlugin) Init(db *storage.DB, mgr *manager.Manager) error {
	p.db = db
	p.mgr = mgr
	return nil
}

func (p *CustomCommandsPlugin) Commands() []*manager.Command {
	return []*manager.Command{
		{
			Trigger:     "customcmd",
			Aliases:     []string{"cc", "customcmds"},
			Name:        "customcmd",
			Description: "Manage server custom commands",
			Category:    "moderation",
			Execute: func(ctx *manager.CommandContext) error {
				perms, err := ctx.UserChannelPermissions(ctx.AuthorID(), ctx.ChanID())
				if err != nil || (perms&discordgo.PermissionManageGuild) == 0 {
					return ctx.Reply("[!] You need Manage Guild permission to use this command.")
				}

				if len(ctx.Args) == 0 {
					return ctx.SendHelp("customcmd")
				}

				gid := ctx.GuildID()
				sub := strings.ToLower(ctx.Args[0])

				switch sub {
				case "create":
					if len(ctx.Args) < 2 {
						return ctx.Reply("[!] Usage: .customcmd create <trigger> [description]")
					}
					trigger := strings.ToLower(ctx.Args[1])
					if ctx.Mgr.FindCommand(trigger) != nil {
						return ctx.Reply("[!] A native command with this name already exists.")
					}

					desc := "Custom command"
					if len(ctx.Args) > 2 {
						desc = strings.Join(ctx.Args[2:], " ")
					}

					cc := storage.CustomCommand{
						Trigger:     trigger,
						Description: desc,
						Actions:     []storage.CommandAction{},
					}
					_ = p.db.SaveCustomCommand(gid, trigger, cc)

					emb := config.Build(ctx.Cfg, config.EmbedOpt{
						Title:       "Custom Command Created",
						Description: fmt.Sprintf("[+] Created custom command `. %s`.\nDescription: %s", trigger, desc),
					})
					emb.Color = 0x708090
					return ctx.Respond(emb)

				case "delete":
					if len(ctx.Args) < 2 {
						return ctx.Reply("[!] Usage: .customcmd delete <trigger>")
					}
					trigger := strings.ToLower(ctx.Args[1])
					cc, err := p.db.GetCustomCommand(gid, trigger)
					if err != nil {
						return ctx.Reply("[!] Custom command not found.")
					}

					g, _ := ctx.Session.State.Guild(gid)
					isOwner := g != nil && g.OwnerID == ctx.AuthorID()
					if cc.BypassExecPerm && !isOwner {
						return ctx.Reply("[!] Only the server owner can delete a command with execution bypass enabled.")
					}

					_ = p.db.DeleteCustomCommand(gid, trigger)

					emb := config.Build(ctx.Cfg, config.EmbedOpt{
						Title:       "Custom Command Deleted",
						Description: fmt.Sprintf("[+] Deleted custom command `. %s`.", trigger),
					})
					emb.Color = 0x708090
					return ctx.Respond(emb)

				case "edit":
					if len(ctx.Args) < 4 {
						return ctx.Reply("[!] Usage: .customcmd edit <trigger> <action|reqs|bypass> <sub_action> [params]")
					}
					trigger := strings.ToLower(ctx.Args[1])
					cc, err := p.db.GetCustomCommand(gid, trigger)
					if err != nil {
						return ctx.Reply("[!] Custom command not found.")
					}

					g, _ := ctx.Session.State.Guild(gid)
					isOwner := g != nil && g.OwnerID == ctx.AuthorID()
					if cc.BypassExecPerm && !isOwner {
						return ctx.Reply("[!] Only the server owner can edit a command with execution bypass enabled.")
					}

					field := strings.ToLower(ctx.Args[2])
					subfield := strings.ToLower(ctx.Args[3])

					switch field {
					case "action":
						if subfield == "clear" {
							cc.Actions = []storage.CommandAction{}
							_ = p.db.SaveCustomCommand(gid, trigger, cc)
							return ctx.Reply("[+] Cleared all actions from command.")
						} else if subfield == "add" {
							if len(ctx.Args) < 5 {
								return ctx.Reply("[!] Usage: .customcmd edit <trigger> action add <type> <params...>")
							}
							actType := strings.ToLower(ctx.Args[4])
							supportedTypes := map[string]bool{
								"send_message": true,
								"add_role":     true,
								"remove_role":  true,
								"quarantine":   true,
								"dm":           true,
							}
							if !supportedTypes[actType] {
								return ctx.Reply("[!] Unsupported action type. Choose from: send_message, add_role, remove_role, quarantine, dm.")
							}

							params := parseParams(ctx.Args[5:])
							cc.Actions = append(cc.Actions, storage.CommandAction{
								Type:   actType,
								Params: params,
							})
							_ = p.db.SaveCustomCommand(gid, trigger, cc)
							return ctx.Reply(fmt.Sprintf("[+] Added action `%s` to command.", actType))
						} else {
							return ctx.Reply("[!] Invalid sub-action. Use `add` or `clear`.")
						}

					case "reqs":
						if subfield == "perms" {
							if len(ctx.Args) < 5 {
								return ctx.Reply("[!] Usage: .customcmd edit <trigger> reqs perms <permission_name>")
							}
							permName := strings.ToLower(ctx.Args[4])
							var perm int64
							switch permName {
							case "administrator", "admin":
								perm = discordgo.PermissionAdministrator
							case "manage_guild", "guild":
								perm = discordgo.PermissionManageGuild
							case "manage_roles", "roles":
								perm = discordgo.PermissionManageRoles
							case "manage_channels", "channels":
								perm = discordgo.PermissionManageChannels
							case "ban_members", "ban":
								perm = discordgo.PermissionBanMembers
							case "kick_members", "kick":
								perm = discordgo.PermissionKickMembers
							case "manage_messages", "messages":
								perm = discordgo.PermissionManageMessages
							case "none":
								perm = 0
							default:
								return ctx.Reply("[!] Unknown permission name. Use e.g. manage_messages, administrator, or none.")
							}
							cc.RequiredPerms = perm
							_ = p.db.SaveCustomCommand(gid, trigger, cc)
							return ctx.Reply(fmt.Sprintf("[+] Required permission set to `%s`.", permName))
						} else if subfield == "role" {
							if len(ctx.Args) < 5 {
								return ctx.Reply("[!] Usage: .customcmd edit <trigger> reqs role <role_id|clear>")
							}
							val := ctx.Args[4]
							if strings.ToLower(val) == "clear" {
								cc.AllowedRoles = []string{}
								_ = p.db.SaveCustomCommand(gid, trigger, cc)
								return ctx.Reply("[+] Cleared allowed role requirements.")
							}
							roleID := strings.Trim(val, "<@&>")
							cc.AllowedRoles = append(cc.AllowedRoles, roleID)
							_ = p.db.SaveCustomCommand(gid, trigger, cc)
							return ctx.Reply(fmt.Sprintf("[+] Added role <@&%s> to allowed execution roles.", roleID))
						} else {
							return ctx.Reply("[!] Invalid sub-action. Use `perms` or `role`.")
						}

					case "bypass":
						g, _ := ctx.Session.State.Guild(gid)
						if g != nil && g.OwnerID != ctx.AuthorID() {
							return ctx.Reply("[!] Only the server owner can toggle execution bypass.")
						}
						val := strings.ToLower(ctx.Args[3])
						if val == "on" || val == "enable" || val == "true" {
							cc.BypassExecPerm = true
							_ = p.db.SaveCustomCommand(gid, trigger, cc)
							return ctx.Reply("[+] Execution bypass enabled (Actions run using bot privileges).")
						} else {
							cc.BypassExecPerm = false
							_ = p.db.SaveCustomCommand(gid, trigger, cc)
							return ctx.Reply("[+] Execution bypass disabled (Actions run using user privileges).")
						}
					default:
						return ctx.Reply("[!] Invalid edit field. Use `action`, `reqs`, or `bypass`.")
					}

				case "list":
					list, err := p.db.ListCustomCommands(gid)
					if err != nil || len(list) == 0 {
						return ctx.Reply("[!] No custom commands configured in this server.")
					}
					var sb strings.Builder
					for _, cmd := range list {
						sb.WriteString(fmt.Sprintf("• `. %s` - %s (%d actions)\n", cmd.Trigger, cmd.Description, len(cmd.Actions)))
					}
					emb := config.Build(ctx.Cfg, config.EmbedOpt{
						Title:       "Server Custom Commands",
						Description: sb.String(),
					})
					emb.Color = 0x708090
					return ctx.Respond(emb)

				case "info":
					if len(ctx.Args) < 2 {
						return ctx.Reply("[!] Usage: .customcmd info <trigger>")
					}
					trigger := strings.ToLower(ctx.Args[1])
					cc, err := p.db.GetCustomCommand(gid, trigger)
					if err != nil {
						return ctx.Reply("[!] Custom command not found.")
					}

					var sb strings.Builder
					sb.WriteString(fmt.Sprintf("• **Trigger:** `%s`\n", cc.Trigger))
					sb.WriteString(fmt.Sprintf("• **Description:** %s\n", cc.Description))
					sb.WriteString(fmt.Sprintf("• **Required Permission Value:** `%d`\n", cc.RequiredPerms))
					sb.WriteString(fmt.Sprintf("• **Bypass Executive Permissions:** `%t`\n\n", cc.BypassExecPerm))

					if len(cc.AllowedRoles) > 0 {
						sb.WriteString("**Allowed Roles:**\n")
						for _, rID := range cc.AllowedRoles {
							sb.WriteString(fmt.Sprintf("- <@&%s>\n", rID))
						}
						sb.WriteString("\n")
					}

					sb.WriteString("**Action Sequence:**\n")
					if len(cc.Actions) == 0 {
						sb.WriteString("No actions defined.\n")
					} else {
						for idx, act := range cc.Actions {
							sb.WriteString(fmt.Sprintf("%d. Type: `%s`\n", idx+1, act.Type))
							for k, v := range act.Params {
								sb.WriteString(fmt.Sprintf("   └ `%s`: %s\n", k, v))
							}
						}
					}

					emb := config.Build(ctx.Cfg, config.EmbedOpt{
						Title:       "Custom Command Info",
						Description: sb.String(),
					})
					emb.Color = 0x708090
					return ctx.Respond(emb)

				default:
					return ctx.Reply("Unknown subcommand. Options: `create`, `delete`, `edit`, `list`, `info`")
				}
			},
		},
	}
}

func parseParams(args []string) map[string]string {
	params := make(map[string]string)
	full := strings.Join(args, " ")
	keys := []string{"content=", "channel_id=", "role_id=", "user_id=", "reason="}
	for _, key := range keys {
		idx := strings.Index(strings.ToLower(full), key)
		if idx != -1 {
			valStart := idx + len(key)
			nextKeyIdx := -1
			for _, otherKey := range keys {
				if otherKey == key {
					continue
				}
				oIdx := strings.Index(strings.ToLower(full[valStart:]), otherKey)
				if oIdx != -1 {
					oIdx += valStart
					if nextKeyIdx == -1 || oIdx < nextKeyIdx {
						nextKeyIdx = oIdx
					}
				}
			}
			var val string
			if nextKeyIdx != -1 {
				val = full[valStart:nextKeyIdx]
			} else {
				val = full[valStart:]
			}
			val = strings.TrimSpace(val)
			val = strings.Trim(val, "\"`")
			params[strings.TrimSuffix(key, "=")] = val
		}
	}
	return params
}
