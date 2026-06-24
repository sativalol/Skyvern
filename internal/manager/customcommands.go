package manager

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"skyvern/internal/storage"
)

func ParseVariables(input string, ctx *CommandContext) string {
	res := input
	res = strings.ReplaceAll(res, "{user}", ctx.AuthorTag())
	res = strings.ReplaceAll(res, "{user.id}", ctx.AuthorID())
	res = strings.ReplaceAll(res, "{user.mention}", fmt.Sprintf("<@%s>", ctx.AuthorID()))

	gid := ctx.GuildID()
	gName := gid
	if g, err := ctx.Session.State.Guild(gid); err == nil {
		gName = g.Name
	}
	res = strings.ReplaceAll(res, "{guild}", gName)
	res = strings.ReplaceAll(res, "{guild.id}", gid)

	chName := ctx.ChanID()
	if ch, err := ctx.Session.State.Channel(ctx.ChanID()); err == nil {
		chName = ch.Name
	}
	res = strings.ReplaceAll(res, "{channel}", chName)
	res = strings.ReplaceAll(res, "{channel.mention}", fmt.Sprintf("<#%s>", ctx.ChanID()))

	allArgs := strings.Join(ctx.Args, " ")
	res = strings.ReplaceAll(res, "{args}", allArgs)
	res = strings.ReplaceAll(res, "{args.all}", allArgs)
	for i, arg := range ctx.Args {
		placeholder := fmt.Sprintf("{args.%d}", i)
		res = strings.ReplaceAll(res, placeholder, arg)
	}

	var targetID string
	if len(ctx.Args) > 0 {
		targetID = strings.Trim(ctx.Args[0], "<@!>")
	}
	if targetID == "" {
		targetID = ctx.AuthorID()
	}
	res = strings.ReplaceAll(res, "{target.id}", targetID)
	res = strings.ReplaceAll(res, "{target.mention}", fmt.Sprintf("<@%s>", targetID))

	return res
}

func (m *Manager) RunCustomCommand(s *discordgo.Session, cmd storage.CustomCommand, ctx *CommandContext) error {
	gid := ctx.GuildID()

	if !cmd.BypassExecPerm {
		if cmd.RequiredPerms > 0 {
			uPerms, err := ctx.UserChannelPermissions(ctx.AuthorID(), ctx.ChanID())
			if err != nil || (uPerms&cmd.RequiredPerms) != cmd.RequiredPerms {
				return fmt.Errorf("you do not have the required permissions to run this command")
			}
		}

		if len(cmd.AllowedRoles) > 0 {
			mem, err := s.GuildMember(gid, ctx.AuthorID())
			if err != nil {
				return fmt.Errorf("failed to verify roles")
			}
			hasRole := false
			for _, rID := range mem.Roles {
				for _, allowed := range cmd.AllowedRoles {
					if rID == allowed {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}
			if !hasRole {
				return fmt.Errorf("you do not have an allowed role to run this command")
			}
		}
	}

	for _, act := range cmd.Actions {
		switch act.Type {
		case "send_message":
			content := ParseVariables(act.Params["content"], ctx)
			chanID := act.Params["channel_id"]
			if chanID == "" {
				chanID = ctx.ChanID()
			} else {
				chanID = strings.Trim(chanID, "<#>")
			}
			if !cmd.BypassExecPerm {
				perms, err := ctx.UserChannelPermissions(ctx.AuthorID(), chanID)
				if err != nil || (perms&discordgo.PermissionSendMessages) == 0 {
					return fmt.Errorf("missing send messages perm in target chan")
				}
			}
			_, _ = s.ChannelMessageSend(chanID, content)

		case "add_role":
			rID := strings.Trim(act.Params["role_id"], "<@&>")
			tID := act.Params["user_id"]
			if tID == "" {
				if len(ctx.Args) > 0 {
					tID = strings.Trim(ctx.Args[0], "<@!>")
				} else {
					tID = ctx.AuthorID()
				}
			} else {
				tID = ParseVariables(tID, ctx)
				tID = strings.Trim(tID, "<@!>")
			}
			if !cmd.BypassExecPerm {
				perms, err := ctx.UserChannelPermissions(ctx.AuthorID(), ctx.ChanID())
				if err != nil || (perms&discordgo.PermissionManageRoles) == 0 {
					return fmt.Errorf("missing manage roles perm")
				}
				if !ctx.CanManageRole(rID) || !ctx.CheckHierarchy(tID) {
					return fmt.Errorf("hierarchy violation")
				}
			}
			_ = s.GuildMemberRoleAdd(gid, tID, rID)

		case "remove_role":
			rID := strings.Trim(act.Params["role_id"], "<@&>")
			tID := act.Params["user_id"]
			if tID == "" {
				if len(ctx.Args) > 0 {
					tID = strings.Trim(ctx.Args[0], "<@!>")
				} else {
					tID = ctx.AuthorID()
				}
			} else {
				tID = ParseVariables(tID, ctx)
				tID = strings.Trim(tID, "<@!>")
			}
			if !cmd.BypassExecPerm {
				perms, err := ctx.UserChannelPermissions(ctx.AuthorID(), ctx.ChanID())
				if err != nil || (perms&discordgo.PermissionManageRoles) == 0 {
					return fmt.Errorf("missing manage roles perm")
				}
				if !ctx.CanManageRole(rID) || !ctx.CheckHierarchy(tID) {
					return fmt.Errorf("hierarchy violation")
				}
			}
			_ = s.GuildMemberRoleRemove(gid, tID, rID)

		case "quarantine":
			tID := act.Params["user_id"]
			if tID == "" {
				if len(ctx.Args) > 0 {
					tID = strings.Trim(ctx.Args[0], "<@!>")
				} else {
					tID = ctx.AuthorID()
				}
			} else {
				tID = ParseVariables(tID, ctx)
				tID = strings.Trim(tID, "<@!>")
			}
			reason := ParseVariables(act.Params["reason"], ctx)
			if reason == "" {
				reason = "Custom command quarantine"
			}
			if !cmd.BypassExecPerm {
				perms, err := ctx.UserChannelPermissions(ctx.AuthorID(), ctx.ChanID())
				if err != nil || (perms&discordgo.PermissionManageGuild) == 0 {
					return fmt.Errorf("missing manage guild perm")
				}
				if !ctx.CheckHierarchy(tID) {
					return fmt.Errorf("hierarchy violation")
				}
			}
			_ = m.QuarantineUser(s, gid, tID, reason, "customcmd")

		case "dm":
			content := ParseVariables(act.Params["content"], ctx)
			tID := act.Params["user_id"]
			if tID == "" {
				tID = ctx.AuthorID()
			} else {
				tID = ParseVariables(tID, ctx)
				tID = strings.Trim(tID, "<@!>")
			}
			ch, err := s.UserChannelCreate(tID)
			if err == nil {
				_, _ = s.ChannelMessageSend(ch.ID, content)
			}
		}
	}

	return nil
}
