package moderation

import (
	"fmt"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/moderation"
	"time"

	"github.com/bwmarrin/discordgo"
)

func init() {
	manager.RegisterHelp("timeout", []manager.HelpPage{
		{
			Command:     "Timeout Member",
			Syntax:      ".timeout <user> <duration>",
			Description: "Timeout a member. Duration supports: m (minutes), h (hours), d (days). e.g. .timeout @user 10m",
		},
	})
	manager.RegisterHelp("tempban", []manager.HelpPage{
		{
			Command:     "Temporary Ban",
			Syntax:      ".tempban <user> <duration>",
			Description: "Bans a user temporarily. Duration supports: m, h, d. e.g. .tempban @user 1d",
		},
	})
}

var Ban = &manager.Command{
	Trigger:     "ban",
	Aliases:     []string{"b"},
	Name:        "ban",
	Description: "Ban a user from the server",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		return runModAction(ModAction{
			Ctx:       ctx,
			Perm:      discordgo.PermissionBanMembers,
			MinArgs:   1,
			Usage:     "ban <user> [reason]",
			CheckHier: true,
			DMAction:  "Ban",
			CaseType:  "ban",
			LogName:   "Ban",
			ActionFn: func(tid string, reason string) error {
				return ctx.Ban(tid, reason, 0)
			},
			SuccessMsg: func(uname string, cid int, reason string) string {
				return fmt.Sprintf("[+] Banned **%s** (Case #%d) | Reason: %s", uname, cid, reason)
			},
		})
	},
}

var Unban = &manager.Command{
	Trigger:     "unban",
	Aliases:     []string{"ub"},
	Name:        "unban",
	Description: "Unban a user by their ID",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		return runModAction(ModAction{
			Ctx:       ctx,
			Perm:      discordgo.PermissionBanMembers,
			MinArgs:   1,
			Usage:     "unban <user_id>",
			CheckHier: false,
			LogName:   "Unban",
			ActionFn: func(tid string, reason string) error {
				return ctx.Unban(tid, "Manual unban command")
			},
			SuccessMsg: func(uname string, cid int, reason string) string {
				return fmt.Sprintf("[+] Unbanned user ID **%s**.", uname)
			},
		})
	},
}

var Hardban = &manager.Command{
	Trigger:     "hardban",
	Aliases:     []string{"hb"},
	Name:        "hardban",
	Description: "Ban user and purge their messages",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		return runModAction(ModAction{
			Ctx:       ctx,
			Perm:      discordgo.PermissionBanMembers,
			MinArgs:   1,
			Usage:     "hardban <user> [reason]",
			CheckHier: true,
			DMAction:  "Hardban",
			CaseType:  "hardban",
			LogName:   "Hardban",
			ActionFn: func(tid string, reason string) error {
				return ctx.Ban(tid, reason+" (Purge 7d)", 7)
			},
			SuccessMsg: func(uname string, cid int, reason string) string {
				return fmt.Sprintf("[+] Hardbanned **%s** (Case #%d) and purged message history.", uname, cid)
			},
		})
	},
}

var Softban = &manager.Command{
	Trigger:     "softban",
	Aliases:     []string{"sb"},
	Name:        "softban",
	Description: "Kick user and purge their messages via quick ban/unban",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		return runModAction(ModAction{
			Ctx:       ctx,
			Perm:      discordgo.PermissionBanMembers,
			MinArgs:   1,
			Usage:     "softban <user>",
			CheckHier: true,
			DMAction:  "Softban",
			CaseType:  "softban",
			LogName:   "Softban",
			ActionFn: func(tid string, reason string) error {
				if err := ctx.Ban(tid, "Softban (Purge)", 7); err != nil {
					return err
				}
				return ctx.Unban(tid, "Softban completion")
			},
			SuccessMsg: func(uname string, cid int, reason string) string {
				return fmt.Sprintf("[+] Softbanned and kicked **%s** (Case #%d) (purged messages).", uname, cid)
			},
		})
	},
}

var Tempban = &manager.Command{
	Trigger:     "tempban",
	Aliases:     []string{"tb"},
	Name:        "tempban",
	Description: "Temporarily ban a user",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) < 2 {
			return ctx.SendHelp("tempban")
		}
		durStr := ctx.Args[1]
		lastChar := durStr[len(durStr)-1]
		if lastChar >= '0' && lastChar <= '9' {
			durStr += "m"
		}
		dur, err := time.ParseDuration(durStr)
		if err != nil {
			return ctx.Reply("[!] Invalid duration. Use e.g. 60m, 2h, 1d.")
		}

		return runModAction(ModAction{
			Ctx:       ctx,
			Perm:      discordgo.PermissionBanMembers,
			MinArgs:   2,
			Usage:     "tempban <user> <duration> [reason]",
			CheckHier: true,
			DMAction:  "Tempban",
			CaseType:  "tempban",
			LogName:   "Tempban",
			ExtraFields: []*discordgo.MessageEmbedField{
				config.Field("Duration", dur.String(), true),
			},
			ActionFn: func(tid string, reason string) error {
				if err := ctx.Ban(tid, fmt.Sprintf("Tempban: %s | Reason: %s", dur.String(), reason), 0); err != nil {
					return err
				}
				go func() {
					time.Sleep(dur)
					_ = ctx.Unban(tid, "Temporary ban expired")
					moderation.LogAction(ctx.Session, ctx.DB, ctx.GuildID(), "Tempban Expired (Auto-Unban)", ctx.Session.State.User.ID, tid, "Automatic temporary ban expiration")
				}()
				return nil
			},
			SuccessMsg: func(uname string, cid int, reason string) string {
				return fmt.Sprintf("[+] Tempbanned **%s** (Case #%d) for %s | Reason: %s", uname, cid, dur.String(), reason)
			},
		})
	},
}

var Kick = &manager.Command{
	Trigger:     "kick",
	Aliases:     []string{"k"},
	Name:        "kick",
	Description: "Kick a user from the server",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		return runModAction(ModAction{
			Ctx:       ctx,
			Perm:      discordgo.PermissionKickMembers,
			MinArgs:   1,
			Usage:     "kick <user>",
			CheckHier: true,
			DMAction:  "Kick",
			CaseType:  "kick",
			LogName:   "Kick",
			ActionFn: func(tid string, reason string) error {
				return ctx.Kick(tid, reason)
			},
			SuccessMsg: func(uname string, cid int, reason string) string {
				return fmt.Sprintf("[+] Kicked **%s** (Case #%d).", uname, cid)
			},
		})
	},
}

var Timeout = &manager.Command{
	Trigger:     "timeout",
	Aliases:     []string{"to", "time"},
	Name:        "timeout",
	Description: "Timeout a user",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) < 2 {
			return ctx.SendHelp("timeout")
		}
		durStr := ctx.Args[1]
		lastChar := durStr[len(durStr)-1]
		if lastChar >= '0' && lastChar <= '9' {
			durStr += "m"
		}
		dur, err := time.ParseDuration(durStr)
		if err != nil {
			return ctx.Reply("[!] Invalid duration. Use e.g. 15m, 2h, 1d.")
		}
		until := time.Now().Add(dur)

		return runModAction(ModAction{
			Ctx:       ctx,
			Perm:      discordgo.PermissionModerateMembers,
			MinArgs:   2,
			Usage:     "timeout <user> <duration> [reason]",
			CheckHier: true,
			DMAction:  "Timeout",
			CaseType:  "timeout",
			LogName:   "Timeout",
			ExtraFields: []*discordgo.MessageEmbedField{
				config.Field("Duration", dur.String(), true),
			},
			ActionFn: func(tid string, reason string) error {
				return ctx.Timeout(tid, &until, reason)
			},
			SuccessMsg: func(uname string, cid int, reason string) string {
				return fmt.Sprintf("[+] Timed out **%s** (Case #%d) until %s | Reason: %s", uname, cid, until.Format("15:04:05"), reason)
			},
		})
	},
}

var Untimeout = &manager.Command{
	Trigger:     "untimeout",
	Aliases:     []string{"uto", "untime"},
	Name:        "untimeout",
	Description: "Remove timeout from a user",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		return runModAction(ModAction{
			Ctx:       ctx,
			Perm:      discordgo.PermissionModerateMembers,
			MinArgs:   1,
			Usage:     "untimeout <user>",
			CheckHier: true,
			DMAction:  "Untimeout",
			LogName:   "Untimeout",
			ActionFn: func(tid string, reason string) error {
				return ctx.Timeout(tid, nil, reason)
			},
			SuccessMsg: func(uname string, cid int, reason string) string {
				return fmt.Sprintf("[+] Removed timeout from **%s**.", uname)
			},
		})
	},
}

var Nickname = &manager.Command{
	Trigger:     "nickname",
	Aliases:     []string{"nick"},
	Name:        "nickname",
	Description: "Change a user's nickname",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		return runModAction(ModAction{
			Ctx:       ctx,
			Perm:      discordgo.PermissionManageNicknames,
			MinArgs:   1,
			Usage:     "nickname <user> <new_nickname>",
			CheckHier: true,
			LogName:   "Nickname Change",
			ActionFn: func(tid string, reason string) error {
				return ctx.Nick(tid, reason, fmt.Sprintf("New nickname: %s", reason))
			},
			SuccessMsg: func(uname string, cid int, reason string) string {
				return fmt.Sprintf("[+] Changed **%s** nickname to **%s**.", uname, reason)
			},
		})
	},
}

var ForceNick = &manager.Command{
	Trigger:     "forcenick",
	Aliases:     []string{"fnick", "forcename", "fn"},
	Name:        "forcenick",
	Description: "Locks a user's nickname so they cannot change it",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		return runModAction(ModAction{
			Ctx:       ctx,
			Perm:      discordgo.PermissionManageNicknames,
			MinArgs:   1,
			Usage:     "forcenick <user> <nickname>",
			CheckHier: true,
			LogName:   "Nickname Force Lock",
			ActionFn: func(tid string, reason string) error {
				_ = ctx.DB.SaveNicklock(ctx.GuildID(), tid, reason)
				return ctx.Nick(tid, reason, fmt.Sprintf("Locked nickname: %s", reason))
			},
			SuccessMsg: func(uname string, cid int, reason string) string {
				return fmt.Sprintf("[Locked] Nickname lock active for **%s** -> **%s**.", uname, reason)
			},
		})
	},
}

var UnforceNick = &manager.Command{
	Trigger:     "unforcenick",
	Aliases:     []string{"ufnick", "unforcename", "unfn"},
	Name:        "unforcenick",
	Description: "Unlock a user's nickname",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		return runModAction(ModAction{
			Ctx:       ctx,
			Perm:      discordgo.PermissionManageNicknames,
			MinArgs:   1,
			Usage:     "unforcenick <user>",
			CheckHier: true,
			LogName:   "Nickname Force Unlock",
			ActionFn: func(tid string, reason string) error {
				return ctx.DB.DeleteNicklock(ctx.GuildID(), tid)
			},
			SuccessMsg: func(uname string, cid int, reason string) string {
				return fmt.Sprintf("[Unlocked] Nickname lock removed for **%s**.", uname)
			},
		})
	},
}
