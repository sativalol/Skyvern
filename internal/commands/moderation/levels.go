package moderation

import (
	"fmt"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/moderation"
	"skyvern/internal/storage"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func init() {
	manager.RegisterHelp("levels", []manager.HelpPage{
		{
			Command:     "Levels View",
			Syntax:      ".levels [member]",
			Description: "View your level and experience or another member's rank.",
		},
		{
			Command:     "Levels Unlock",
			Syntax:      ".levels unlock",
			Description: "Enable the leveling system on this server.",
		},
		{
			Command:     "Levels Lock",
			Syntax:      ".levels lock",
			Description: "Disable the leveling system on this server.",
		},
		{
			Command:     "Levels Message Mode",
			Syntax:      ".levels messagemode <channel | dm | disabled>",
			Description: "Configure where level up notifications are sent.",
		},
		{
			Command:     "Levels Message Set",
			Syntax:      ".levels message <text>",
			Description: "Set a custom level up notification message.",
		},
		{
			Command:     "Levels Message View",
			Syntax:      ".levels message view",
			Description: "View the current server level up message.",
		},
		{
			Command:     "Levels Leaderboard",
			Syntax:      ".levels leaderboard",
			Description: "View top rank holders on the server.",
		},
		{
			Command:     "Levels Leaderboard Rename",
			Syntax:      ".levels leaderboard rename <text>",
			Description: "Change the default title of the leaderboard embed.",
		},
		{
			Command:     "Levels Set Rate",
			Syntax:      ".levels setrate <multiplier>",
			Description: "Adjust multiplier for XP gains (default: 1.0).",
		},
		{
			Command:     "Levels Stack Roles",
			Syntax:      ".levels stackroles <on | off>",
			Description: "Enable or disable stacking of previously earned level roles.",
		},
		{
			Command:     "Levels Ignore Target",
			Syntax:      ".levels ignore <#channel | @role>",
			Description: "Ignore or unignore a channel or role from gaining experience.",
		},
		{
			Command:     "Levels List Ignores",
			Syntax:      ".levels list",
			Description: "Show all ignored roles and channels.",
		},
		{
			Command:     "Levels Role Add",
			Syntax:      ".levels add <@role> <rank>",
			Description: "Map a rank requirement to a Discord role.",
		},
		{
			Command:     "Levels Role Remove",
			Syntax:      ".levels remove <rank>",
			Description: "Delete level role association from a rank.",
		},
		{
			Command:     "Levels Role Update",
			Syntax:      ".levels update <@role> <rank>",
			Description: "Update a level role's mapped rank.",
		},
		{
			Command:     "Levels Role List",
			Syntax:      ".levels roles",
			Description: "Show all level role milestones.",
		},
		{
			Command:     "Levels Sync Roles",
			Syntax:      ".levels sync",
			Description: "Update level roles for all members based on their current levels.",
		},
		{
			Command:     "Levels Reset Server",
			Syntax:      ".levels reset",
			Description: "Wipe all XP and level data for all members.",
		},
		{
			Command:     "Levels Personal Toggle",
			Syntax:      ".levels messages <on | off>",
			Description: "Toggle level up notification messages for yourself.",
		},
	})
	manager.RegisterHelp("setxp", []manager.HelpPage{
		{
			Command:     "Set XP",
			Syntax:      ".setxp <member> <amount>",
			Description: "Set a member's experience point total manually.",
		},
	})
	manager.RegisterHelp("removexp", []manager.HelpPage{
		{
			Command:     "Remove XP",
			Syntax:      ".removexp <member> <amount>",
			Description: "Deduct experience points from a member.",
		},
	})
	manager.RegisterHelp("setlevel", []manager.HelpPage{
		{
			Command:     "Set Level",
			Syntax:      ".setlevel <member> <level>",
			Description: "Force modify a member's rank level.",
		},
	})
}

var Levels = &manager.Command{
	Trigger:     "levels",
	Aliases:     []string{"level", "lvl", "rank"},
	Name:        "levels",
	Description: "Leveling system configuration and user levels",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()
		if len(ctx.Args) == 0 {
			return showUserLevel(ctx, ctx.AuthorID())
		}

		sub := strings.ToLower(ctx.Args[0])
		switch sub {
		case "unlock":
			cfg, _ := ctx.DB.GetLevelsCfg(gid)
			cfg.Enabled = true
			_ = ctx.DB.SaveLevelsCfg(gid, cfg)
			return ctx.Reply("[+] Leveling system has been unlocked (enabled).")

		case "lock":
			cfg, _ := ctx.DB.GetLevelsCfg(gid)
			cfg.Enabled = false
			_ = ctx.DB.SaveLevelsCfg(gid, cfg)
			return ctx.Reply("[+] Leveling system has been locked (disabled).")

		case "config":
			cfg, _ := ctx.DB.GetLevelsCfg(gid)
			var sb strings.Builder
			sb.WriteString("**Leveling Configuration:**\n")
			sb.WriteString(fmt.Sprintf("- Enabled: `%t`\n", cfg.Enabled))
			sb.WriteString(fmt.Sprintf("- Message Mode: `%s`\n", cfg.MessageMode))
			if cfg.MessageChan != "" {
				sb.WriteString(fmt.Sprintf("- Message Channel: <#%s>\n", cfg.MessageChan))
			}
			sb.WriteString(fmt.Sprintf("- XP Multiplier: `%.2f`\n", cfg.Rate))
			sb.WriteString(fmt.Sprintf("- Stack Roles: `%t`\n", cfg.StackRoles))
			sb.WriteString(fmt.Sprintf("- Leaderboard Title: `%s`\n", cfg.LeaderboardTitle))
			return ctx.Reply(sb.String())

		case "messagemode":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("levels")
			}
			cfg, _ := ctx.DB.GetLevelsCfg(gid)
			mode := strings.ToLower(ctx.Args[1])
			if mode == "dm" {
				cfg.MessageMode = "dm"
				cfg.MessageChan = ""
			} else if mode == "disabled" || mode == "off" {
				cfg.MessageMode = "disabled"
			} else {
				cid := strings.Trim(ctx.Args[1], "<#>")
				_, err := ctx.Session.Channel(cid)
				if err != nil {
					return ctx.SendHelp("levels")
				}
				cfg.MessageMode = "channel"
				cfg.MessageChan = cid
			}
			_ = ctx.DB.SaveLevelsCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("[+] Set level up message mode to `%s`.", cfg.MessageMode))

		case "message":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("levels")
			}
			cfg, _ := ctx.DB.GetLevelsCfg(gid)
			if strings.ToLower(ctx.Args[1]) == "view" {
				return ctx.Reply(fmt.Sprintf("Current level up message: `%s`", cfg.Message))
			}
			cfg.Message = strings.Join(ctx.Args[1:], " ")
			_ = ctx.DB.SaveLevelsCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("[+] Custom level up message set to: `%s`", cfg.Message))

		case "leaderboard":
			cfg, _ := ctx.DB.GetLevelsCfg(gid)
			if len(ctx.Args) > 1 && strings.ToLower(ctx.Args[1]) == "rename" {
				if len(ctx.Args) < 3 {
					return ctx.SendHelp("levels")
				}
				cfg.LeaderboardTitle = strings.Join(ctx.Args[2:], " ")
				_ = ctx.DB.SaveLevelsCfg(gid, cfg)
				return ctx.Reply(fmt.Sprintf("[+] Leaderboard title updated to `%s`.", cfg.LeaderboardTitle))
			}

			xpMap, err := ctx.DB.ListLevelsXP(gid)
			if err != nil || len(xpMap) == 0 {
				return ctx.Reply("[*] No leveling data exists for this server yet.")
			}

			type userRank struct {
				userID string
				xp     int64
				level  int
			}
			var ranks []userRank
			for uid, u := range xpMap {
				ranks = append(ranks, userRank{userID: uid, xp: u.XP, level: u.Level})
			}
			sort.Slice(ranks, func(i, j int) bool {
				return ranks[i].xp > ranks[j].xp
			})

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("**%s**\n\n", cfg.LeaderboardTitle))
			limit := 10
			if len(ranks) < limit {
				limit = len(ranks)
			}
			for i := 0; i < limit; i++ {
				sb.WriteString(fmt.Sprintf("%d. <@%s> - Level **%d** (%d total XP)\n", i+1, ranks[i].userID, ranks[i].level, ranks[i].xp))
			}
			return ctx.Reply(sb.String())

		case "setrate":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("levels")
			}
			rate, err := strconv.ParseFloat(ctx.Args[1], 64)
			if err != nil || rate <= 0 {
				return ctx.SendHelp("levels")
			}
			cfg, _ := ctx.DB.GetLevelsCfg(gid)
			cfg.Rate = rate
			_ = ctx.DB.SaveLevelsCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("[+] Set XP multiplier to `%.2f`.", rate))

		case "stackroles":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("levels")
			}
			opt := strings.ToLower(ctx.Args[1])
			cfg, _ := ctx.DB.GetLevelsCfg(gid)
			cfg.StackRoles = opt == "on" || opt == "true" || opt == "yes"
			_ = ctx.DB.SaveLevelsCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("[+] Set role stacking to `%t`.", cfg.StackRoles))

		case "ignore":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("levels")
			}
			target := ctx.Args[1]
			cfg, _ := ctx.DB.GetLevelsCfg(gid)

			cid := strings.Trim(target, "<#>")
			ch, err := ctx.Session.Channel(cid)
			if err == nil && ch.GuildID == gid {
				foundIdx := -1
				for i, id := range cfg.IgnoredChans {
					if id == cid {
						foundIdx = i
						break
					}
				}
				if foundIdx != -1 {
					cfg.IgnoredChans = append(cfg.IgnoredChans[:foundIdx], cfg.IgnoredChans[foundIdx+1:]...)
					_ = ctx.DB.SaveLevelsCfg(gid, cfg)
					return ctx.Reply(fmt.Sprintf("[+] Unignored channel <#%s> for XP.", cid))
				}
				cfg.IgnoredChans = append(cfg.IgnoredChans, cid)
				_ = ctx.DB.SaveLevelsCfg(gid, cfg)
				return ctx.Reply(fmt.Sprintf("[+] Ignored channel <#%s> for XP.", cid))
			}

			rid := strings.Trim(target, "<@&>")
			_, err = ctx.Session.State.Role(gid, rid)
			if err != nil {
				if roles, err := ctx.Session.GuildRoles(gid); err == nil {
					for _, r := range roles {
						if r.ID == rid {
							err = nil
							break
						}
					}
				}
			}
			if err == nil {
				foundIdx := -1
				for i, id := range cfg.IgnoredRoles {
					if id == rid {
						foundIdx = i
						break
					}
				}
				if foundIdx != -1 {
					cfg.IgnoredRoles = append(cfg.IgnoredRoles[:foundIdx], cfg.IgnoredRoles[foundIdx+1:]...)
					_ = ctx.DB.SaveLevelsCfg(gid, cfg)
					return ctx.Reply(fmt.Sprintf("[+] Unignored role <@&%s> for XP.", rid))
				}
				cfg.IgnoredRoles = append(cfg.IgnoredRoles, rid)
				_ = ctx.DB.SaveLevelsCfg(gid, cfg)
				return ctx.Reply(fmt.Sprintf("[+] Ignored role <@&%s> for XP.", rid))
			}
			return ctx.SendHelp("levels")

		case "list":
			cfg, _ := ctx.DB.GetLevelsCfg(gid)
			var sb strings.Builder
			sb.WriteString("**Ignored Channels & Roles:**\n\n")
			sb.WriteString("**Channels:**\n")
			if len(cfg.IgnoredChans) == 0 {
				sb.WriteString("None\n")
			} else {
				for _, id := range cfg.IgnoredChans {
					sb.WriteString(fmt.Sprintf("- <#%s>\n", id))
				}
			}
			sb.WriteString("\n**Roles:**\n")
			if len(cfg.IgnoredRoles) == 0 {
				sb.WriteString("None\n")
			} else {
				for _, id := range cfg.IgnoredRoles {
					sb.WriteString(fmt.Sprintf("- <@&%s>\n", id))
				}
			}
			return ctx.Reply(sb.String())

		case "add":
			if len(ctx.Args) < 3 {
				return ctx.SendHelp("levels")
			}
			rid, err := resolveRoleOrReply(ctx, ctx.Args[1])
			if err != nil {
				return nil
			}
			lvl, err := strconv.Atoi(ctx.Args[2])
			if err != nil || lvl <= 0 {
				return ctx.SendHelp("levels")
			}
			_ = ctx.DB.SaveLevelRole(gid, storage.LevelRole{RoleID: rid, Level: lvl})
			return ctx.Reply(fmt.Sprintf("[+] Added level role <@&%s> for level **%d**.", rid, lvl))

		case "remove":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("levels")
			}
			lvl, err := strconv.Atoi(ctx.Args[1])
			if err != nil {
				return ctx.SendHelp("levels")
			}
			roles, _ := ctx.DB.ListLevelRoles(gid)
			removed := false
			for _, lr := range roles {
				if lr.Level == lvl {
					_ = ctx.DB.DeleteLevelRole(gid, lr.RoleID)
					removed = true
				}
			}
			if removed {
				return ctx.Reply(fmt.Sprintf("[+] Removed level role associated with level **%d**.", lvl))
			}
			return ctx.Reply(fmt.Sprintf("[!] No level role found for level **%d**.", lvl))

		case "update":
			if len(ctx.Args) < 3 {
				return ctx.SendHelp("levels")
			}
			rid, err := resolveRoleOrReply(ctx, ctx.Args[1])
			if err != nil {
				return nil
			}
			lvl, err := strconv.Atoi(ctx.Args[2])
			if err != nil || lvl <= 0 {
				return ctx.SendHelp("levels")
			}
			_ = ctx.DB.SaveLevelRole(gid, storage.LevelRole{RoleID: rid, Level: lvl})
			return ctx.Reply(fmt.Sprintf("[+] Updated level role <@&%s> to level **%d**.", rid, lvl))

		case "roles":
			roles, err := ctx.DB.ListLevelRoles(gid)
			if err != nil || len(roles) == 0 {
				return ctx.Reply("[*] No custom level roles registered.")
			}
			sort.Slice(roles, func(i, j int) bool {
				return roles[i].Level < roles[j].Level
			})
			var sb strings.Builder
			sb.WriteString("**XP/Level Roles:**\n\n")
			for _, lr := range roles {
				sb.WriteString(fmt.Sprintf("- Level **%d**: <@&%s>\n", lr.Level, lr.RoleID))
			}
			return ctx.Reply(sb.String())

		case "sync":
			_ = ctx.Reply("[*] Syncing member level roles. This might take a moment...")
			go func() {
				mList, err := ctx.Session.GuildMembers(gid, "", 1000)
				if err != nil {
					return
				}
				synced := 0
				for _, mem := range mList {
					if mem.User.Bot {
						continue
					}
					u, _ := ctx.DB.GetUserXP(gid, mem.User.ID)
					ctx.Mgr.SyncLevelRoles(ctx.Session, gid, mem.User.ID, u.Level)
					synced++
					time.Sleep(100 * time.Millisecond)
				}
				_, _ = ctx.Session.ChannelMessageSend(ctx.ChanID(), fmt.Sprintf("[+] Sync completed. Synced level roles for %d members.", synced))
			}()
			return nil

		case "reset":
			_ = ctx.DB.ClearLevelsXP(gid)
			return ctx.Reply("[+] Reset all members level and XP data.")

		case "cleanup":
			roles, _ := ctx.DB.ListLevelRoles(gid)
			gRoles, err := ctx.Session.GuildRoles(gid)
			if err == nil {
				gRoleMap := make(map[string]bool)
				for _, r := range gRoles {
					gRoleMap[r.ID] = true
				}
				cleaned := 0
				for _, lr := range roles {
					if !gRoleMap[lr.RoleID] {
						_ = ctx.DB.DeleteLevelRole(gid, lr.RoleID)
						cleaned++
					}
				}
				return ctx.Reply(fmt.Sprintf("[+] Cleanup complete. Removed %d orphan level roles from database.", cleaned))
			}
			return ctx.Reply("[!] Failed to fetch guild roles for cleanup.")

		case "messages":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("levels")
			}
			opt := strings.ToLower(ctx.Args[1])
			u, _ := ctx.DB.GetUserXP(gid, ctx.AuthorID())
			u.MessagesToggle = opt == "on" || opt == "true" || opt == "yes"
			_ = ctx.DB.SaveUserXP(gid, ctx.AuthorID(), u)
			return ctx.Reply(fmt.Sprintf("[+] Set level up messages to `%t` for yourself.", u.MessagesToggle))

		default:
			m, err := moderation.ResolveMember(ctx.Session, gid, ctx.Args[0])
			if err != nil || m == nil {
				return ctx.SendHelp("levels")
			}
			return showUserLevel(ctx, m.User.ID)
		}
	},
}

func showUserLevel(ctx *manager.CommandContext, userID string) error {
	usr, err := ctx.Session.User(userID)
	if err != nil {
		return ctx.Reply("[!] Could not resolve user.")
	}
	u, _ := ctx.DB.GetUserXP(ctx.GuildID(), userID)

	currLvlXP := manager.XPForLevel(u.Level)
	nextLvlXP := manager.XPForLevel(u.Level + 1)
	xpInLvl := u.XP - currLvlXP
	neededInLvl := nextLvlXP - currLvlXP

	pBar := progressBar(xpInLvl, neededInLvl)

	emb := config.Build(ctx.Cfg, config.EmbedOpt{
		Title: fmt.Sprintf("%s's Rank Info", usr.Username),
		Fields: []*discordgo.MessageEmbedField{
			config.Field("Level", fmt.Sprintf("**%d**", u.Level), true),
			config.Field("Total Experience", fmt.Sprintf("**%d** XP", u.XP), true),
			config.Field("Progress to Level "+fmt.Sprintf("%d", u.Level+1), pBar, false),
		},
		ThumbnailURL: usr.AvatarURL("128"),
	})
	return ctx.Respond(emb)
}

func progressBar(current, max int64) string {
	if max <= 0 {
		max = 1
	}
	percent := float64(current) / float64(max)
	if percent > 1.0 {
		percent = 1.0
	}
	if percent < 0.0 {
		percent = 0.0
	}
	bars := int(percent * 10)
	var sb strings.Builder
	sb.WriteString("`")
	for i := 0; i < 10; i++ {
		if i < bars {
			sb.WriteString("■")
		} else {
			sb.WriteString("□")
		}
	}
	sb.WriteString("`")
	return fmt.Sprintf("%s  %d / %d XP (%.1f%%)", sb.String(), current, max, percent*100)
}

var SetXP = &manager.Command{
	Trigger:     "setxp",
	Name:        "setxp",
	Description: "Set a user's experience",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) < 2 {
			return ctx.SendHelp("setxp")
		}
		m, err := resolveMemberOrReply(ctx, ctx.Args[0])
		if err != nil {
			return nil
		}
		amount, err := strconv.ParseInt(ctx.Args[1], 10, 64)
		if err != nil || amount < 0 {
			return ctx.SendHelp("setxp")
		}

		u, _ := ctx.DB.GetUserXP(ctx.GuildID(), m.User.ID)
		u.XP = amount
		u.Level = manager.LevelForXP(amount)
		_ = ctx.DB.SaveUserXP(ctx.GuildID(), m.User.ID, u)

		ctx.Mgr.SyncLevelRoles(ctx.Session, ctx.GuildID(), m.User.ID, u.Level)
		return ctx.Reply(fmt.Sprintf("[+] Set **%s**'s XP to `%d` (Level **%d**).", m.User.Username, u.XP, u.Level))
	},
}

var RemoveXP = &manager.Command{
	Trigger:     "removexp",
	Name:        "removexp",
	Description: "Remove experience from a user",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		if !checkPerm(ctx, discordgo.PermissionManageGuild) {
			return ctx.Reply("[!] You need Manage Guild permission to use this command.")
		}
		if len(ctx.Args) < 2 {
			return ctx.SendHelp("removexp")
		}
		m, err := resolveMemberOrReply(ctx, ctx.Args[0])
		if err != nil {
			return nil
		}
		amount, err := strconv.ParseInt(ctx.Args[1], 10, 64)
		if err != nil || amount < 0 {
			return ctx.SendHelp("removexp")
		}

		u, _ := ctx.DB.GetUserXP(ctx.GuildID(), m.User.ID)
		u.XP -= amount
		if u.XP < 0 {
			u.XP = 0
		}
		u.Level = manager.LevelForXP(u.XP)
		_ = ctx.DB.SaveUserXP(ctx.GuildID(), m.User.ID, u)

		ctx.Mgr.SyncLevelRoles(ctx.Session, ctx.GuildID(), m.User.ID, u.Level)
		return ctx.Reply(fmt.Sprintf("[+] Removed `%d` XP from **%s** (new XP: `%d`, Level **%d**).", amount, m.User.Username, u.XP, u.Level))
	},
}

var SetLevel = &manager.Command{
	Trigger:     "setlevel",
	Name:        "setlevel",
	Description: "Set a user's level",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		if !checkPerm(ctx, discordgo.PermissionManageGuild) {
			return ctx.Reply("[!] You need Manage Guild permission to use this command.")
		}
		if len(ctx.Args) < 2 {
			return ctx.SendHelp("setlevel")
		}
		m, err := resolveMemberOrReply(ctx, ctx.Args[0])
		if err != nil {
			return nil
		}
		lvl, err := strconv.Atoi(ctx.Args[1])
		if err != nil || lvl < 0 {
			return ctx.SendHelp("setlevel")
		}

		u, _ := ctx.DB.GetUserXP(ctx.GuildID(), m.User.ID)
		u.Level = lvl
		u.XP = manager.XPForLevel(lvl)
		_ = ctx.DB.SaveUserXP(ctx.GuildID(), m.User.ID, u)

		ctx.Mgr.SyncLevelRoles(ctx.Session, ctx.GuildID(), m.User.ID, u.Level)
		return ctx.Reply(fmt.Sprintf("[+] Set **%s**'s level to **%d** (XP set to `%d`).", m.User.Username, u.Level, u.XP))
	},
}
