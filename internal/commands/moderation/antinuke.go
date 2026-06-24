package moderation

import (
	"fmt"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/moderation"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func resolvePermFlag(name string) int64 {
	switch strings.ToLower(name) {
	case "administrator", "admin":
		return discordgo.PermissionAdministrator
	case "manage_roles", "roles":
		return discordgo.PermissionManageRoles
	case "manage_guild", "guild":
		return discordgo.PermissionManageGuild
	case "ban_members", "ban":
		return discordgo.PermissionBanMembers
	case "kick_members", "kick":
		return discordgo.PermissionKickMembers
	case "manage_webhooks", "webhooks":
		return discordgo.PermissionManageWebhooks
	case "manage_channels", "channels":
		return discordgo.PermissionManageChannels
	default:
		return 0
	}
}

func init() {
	manager.RegisterHelp("antinuke", []manager.HelpPage{
		{
			Command:     "Antinuke Toggle",
			Syntax:      ".antinuke <on|off>",
			Description: "Turn the antinuke protection engine on or off (owner only).",
		},
		{
			Command:     "Antinuke Admin",
			Syntax:      ".antinuke admin <member>",
			Description: "Give or remove a user's permission to edit antinuke settings (owner only).",
		},
		{
			Command:     "Antinuke Admins List",
			Syntax:      ".antinuke admins",
			Description: "View all antinuke admins.",
		},
		{
			Command:     "Antinuke Whitelist",
			Syntax:      ".antinuke whitelist <member>",
			Description: "Whitelist or unwhitelist a member from triggering antinuke limits or new bots from joining (owner only).",
		},
		{
			Command:     "Antinuke List",
			Syntax:      ".antinuke list",
			Description: "View all enabled modules along with whitelisted members & bots and admins.",
		},
		{
			Command:     "Antinuke Config",
			Syntax:      ".antinuke config",
			Description: "View server configuration for Antinuke.",
		},
		{
			Command:     "Antinuke BotAdd Module",
			Syntax:      ".antinuke botadd <on|off>",
			Description: "Prevent new bot additions.",
		},
		{
			Command:     "Antinuke Channel Module",
			Syntax:      ".antinuke channel <on|off> [limit] [seconds]",
			Description: "Prevent mass channel create and delete.",
		},
		{
			Command:     "Antinuke Webhook Module",
			Syntax:      ".antinuke webhook <on|off> [limit] [seconds]",
			Description: "Prevent mass webhook creation.",
		},
		{
			Command:     "Antinuke Kick Module",
			Syntax:      ".antinuke kick <on|off> [limit] [seconds]",
			Description: "Prevent mass member kick.",
		},
		{
			Command:     "Antinuke Ban Module",
			Syntax:      ".antinuke ban <on|off> [limit] [seconds]",
			Description: "Prevent mass member ban.",
		},
		{
			Command:     "Antinuke Role Module",
			Syntax:      ".antinuke role <on|off> [limit] [seconds]",
			Description: "Prevent mass role delete.",
		},
		{
			Command:     "Antinuke Emoji Module",
			Syntax:      ".antinuke emoji <on|off> [limit] [seconds]",
			Description: "Prevent mass emoji delete.",
		},
		{
			Command:     "Antinuke Vanity Module",
			Syntax:      ".antinuke vanity <on|off> [limit] [seconds]",
			Description: "Punish users that change the server vanity.",
		},
		{
			Command:     "Antinuke Permissions Module",
			Syntax:      ".antinuke permissions <role|user> <permission> <on|off>",
			Description: "Watch dangerous permissions being granted or removed.",
		},
	})
}

func isAntinukeAdminOrOwner(ctx *manager.CommandContext) bool {
	if isOwner(ctx) {
		return true
	}
	return ctx.DB.IsAntinukeAdmin(ctx.GuildID(), ctx.AuthorID())
}

func fmtEnabled(b bool) string {
	if b {
		return "`Enabled`"
	}
	return "`Disabled`"
}

func handleThresholdModule(ctx *manager.CommandContext, name string, enabled *bool, limit *int, secs *int) error {
	if !isAntinukeAdminOrOwner(ctx) {
		return ctx.Reply(fmt.Sprintf("%s You do not have permission to modify antinuke settings.", ctx.ErrorEmoji()))
	}
	if len(ctx.Args) < 2 {
		return ctx.Reply(fmt.Sprintf("%s Missing status. Usage: .antinuke %s <on|off> [limit] [seconds]", ctx.WarningEmoji(), name))
	}

	status := strings.ToLower(ctx.Args[1])
	if status == "on" || status == "enable" {
		*enabled = true
		if len(ctx.Args) >= 4 {
			lim, err1 := strconv.Atoi(ctx.Args[2])
			scs, err2 := strconv.Atoi(ctx.Args[3])
			if err1 != nil || err2 != nil || lim <= 0 || scs <= 0 {
				return ctx.Reply(fmt.Sprintf("%s Threshold limit and seconds must be positive integers.", ctx.ErrorEmoji()))
			}
			*limit = lim
			*secs = scs
		}
		return ctx.Reply(fmt.Sprintf("%s Antinuke %s module enabled with limit `%d` per `%d`s.", ctx.SuccessEmoji(), name, *limit, *secs))
	} else if status == "off" || status == "disable" {
		*enabled = false
		return ctx.Reply(fmt.Sprintf("%s Antinuke %s module disabled.", ctx.SuccessEmoji(), name))
	} else {
		return ctx.Reply(fmt.Sprintf("%s Invalid status. Use `on` or `off`.", ctx.ErrorEmoji()))
	}
}

var Antinuke = &manager.Command{
	Trigger:     "antinuke",
	Aliases:     []string{"an"},
	Name:        "antinuke",
	Description: "Manage antinuke thresholds, admins, and whitelists (owner/admin)",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()

		if len(ctx.Args) == 0 {
			return ctx.SendHelp("antinuke")
		}

		switch sub := strings.ToLower(ctx.Args[0]); sub {
		case "enable", "on":
			if !isOwner(ctx) {
				return ctx.Reply(fmt.Sprintf("%s Only the server owner can modify global antinuke settings.", ctx.ErrorEmoji()))
			}
			cfg, _ := ctx.Mgr.GetAntinukeCfg(gid)
			cfg.Enabled = true
			_ = ctx.Mgr.SaveAntinukeCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Antinuke protection enabled globally.", ctx.SuccessEmoji()))

		case "disable", "off":
			if !isOwner(ctx) {
				return ctx.Reply(fmt.Sprintf("%s Only the server owner can modify global antinuke settings.", ctx.ErrorEmoji()))
			}
			cfg, _ := ctx.Mgr.GetAntinukeCfg(gid)
			cfg.Enabled = false
			_ = ctx.Mgr.SaveAntinukeCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Antinuke protection disabled globally.", ctx.SuccessEmoji()))

		case "admin":
			if !isOwner(ctx) {
				return ctx.Reply(fmt.Sprintf("%s Only the server owner can manage antinuke admins.", ctx.ErrorEmoji()))
			}
			if len(ctx.Args) < 2 {
				return ctx.Reply(fmt.Sprintf("%s Usage: .antinuke admin <member>", ctx.WarningEmoji()))
			}
			m, err := moderation.ResolveMember(ctx.Session, gid, ctx.Args[1])
			if err != nil || m == nil {
				return ctx.Reply(fmt.Sprintf("%s Could not resolve member.", ctx.ErrorEmoji()))
			}

			if ctx.DB.IsAntinukeAdmin(gid, m.User.ID) {
				_ = ctx.DB.DeleteAntinukeAdmin(gid, m.User.ID)
				return ctx.Reply(fmt.Sprintf("%s Removed **%s** from antinuke admins.", ctx.SuccessEmoji(), m.User.Username))
			} else {
				_ = ctx.DB.AddAntinukeAdmin(gid, m.User.ID)
				return ctx.Reply(fmt.Sprintf("%s Added **%s** to antinuke admins.", ctx.SuccessEmoji(), m.User.Username))
			}

		case "admins":
			if !isAntinukeAdminOrOwner(ctx) {
				return ctx.Reply(fmt.Sprintf("%s You do not have permission to view antinuke admins.", ctx.ErrorEmoji()))
			}
			admins, err := ctx.DB.ListAntinukeAdmins(gid)
			if err != nil || len(admins) == 0 {
				return ctx.Reply(fmt.Sprintf("%s No users in antinuke admins list.", ctx.WarningEmoji()))
			}
			var sb strings.Builder
			sb.WriteString("Antinuke Admins:\n\n")
			for _, uid := range admins {
				sb.WriteString(fmt.Sprintf("- <@%s> (`%s`)\n", uid, uid))
			}
			return ctx.Reply(sb.String())

		case "whitelist", "wl":
			if !isOwner(ctx) {
				return ctx.Reply(fmt.Sprintf("%s Only the server owner can modify whitelists.", ctx.ErrorEmoji()))
			}
			if len(ctx.Args) < 2 {
				return ctx.Reply(fmt.Sprintf("%s Usage: .antinuke whitelist <member>", ctx.WarningEmoji()))
			}
			m, err := moderation.ResolveMember(ctx.Session, gid, ctx.Args[1])
			if err != nil || m == nil {
				return ctx.Reply(fmt.Sprintf("%s Could not resolve member.", ctx.ErrorEmoji()))
			}

			if ctx.DB.IsAntinukeWhitelisted(gid, m.User.ID) {
				_ = ctx.DB.DeleteAntinukeWhitelist(gid, m.User.ID)
				return ctx.Reply(fmt.Sprintf("%s Removed **%s** from antinuke whitelist.", ctx.SuccessEmoji(), m.User.Username))
			} else {
				_ = ctx.DB.AddAntinukeWhitelist(gid, m.User.ID)
				return ctx.Reply(fmt.Sprintf("%s Added **%s** to antinuke whitelist.", ctx.SuccessEmoji(), m.User.Username))
			}

		case "botadd":
			if !isAntinukeAdminOrOwner(ctx) {
				return ctx.Reply(fmt.Sprintf("%s You do not have permission to modify antinuke settings.", ctx.ErrorEmoji()))
			}
			if len(ctx.Args) < 2 {
				return ctx.Reply(fmt.Sprintf("%s Missing status. Usage: .antinuke botadd <on|off>", ctx.WarningEmoji()))
			}
			status := strings.ToLower(ctx.Args[1])
			cfg, _ := ctx.Mgr.GetAntinukeCfg(gid)
			if status == "on" || status == "enable" {
				cfg.BotaddEnabled = true
				_ = ctx.Mgr.SaveAntinukeCfg(gid, cfg)
				return ctx.Reply(fmt.Sprintf("%s Antinuke botadd module enabled.", ctx.SuccessEmoji()))
			} else if status == "off" || status == "disable" {
				cfg.BotaddEnabled = false
				_ = ctx.Mgr.SaveAntinukeCfg(gid, cfg)
				return ctx.Reply(fmt.Sprintf("%s Antinuke botadd module disabled.", ctx.SuccessEmoji()))
			} else {
				return ctx.Reply(fmt.Sprintf("%s Invalid status. Use `on` or `off`.", ctx.ErrorEmoji()))
			}

		case "channel":
			cfg, _ := ctx.Mgr.GetAntinukeCfg(gid)
			err := handleThresholdModule(ctx, "channel", &cfg.ChanEnabled, &cfg.ChanLimit, &cfg.ChanSecs)
			if err == nil {
				_ = ctx.Mgr.SaveAntinukeCfg(gid, cfg)
			}
			return err

		case "webhook":
			cfg, _ := ctx.Mgr.GetAntinukeCfg(gid)
			err := handleThresholdModule(ctx, "webhook", &cfg.WebhookEnabled, &cfg.WebhookLimit, &cfg.WebhookSecs)
			if err == nil {
				_ = ctx.Mgr.SaveAntinukeCfg(gid, cfg)
			}
			return err

		case "kick":
			cfg, _ := ctx.Mgr.GetAntinukeCfg(gid)
			err := handleThresholdModule(ctx, "kick", &cfg.KickEnabled, &cfg.KickLimit, &cfg.KickSecs)
			if err == nil {
				_ = ctx.Mgr.SaveAntinukeCfg(gid, cfg)
			}
			return err

		case "ban":
			cfg, _ := ctx.Mgr.GetAntinukeCfg(gid)
			err := handleThresholdModule(ctx, "ban", &cfg.BanEnabled, &cfg.BanLimit, &cfg.BanSecs)
			if err == nil {
				_ = ctx.Mgr.SaveAntinukeCfg(gid, cfg)
			}
			return err

		case "role":
			cfg, _ := ctx.Mgr.GetAntinukeCfg(gid)
			err := handleThresholdModule(ctx, "role", &cfg.RoleEnabled, &cfg.RoleLimit, &cfg.RoleSecs)
			if err == nil {
				_ = ctx.Mgr.SaveAntinukeCfg(gid, cfg)
			}
			return err

		case "emoji":
			cfg, _ := ctx.Mgr.GetAntinukeCfg(gid)
			err := handleThresholdModule(ctx, "emoji", &cfg.EmojiEnabled, &cfg.EmojiLimit, &cfg.EmojiSecs)
			if err == nil {
				_ = ctx.Mgr.SaveAntinukeCfg(gid, cfg)
			}
			return err

		case "vanity":
			cfg, _ := ctx.Mgr.GetAntinukeCfg(gid)
			err := handleThresholdModule(ctx, "vanity", &cfg.VanityEnabled, &cfg.VanityLimit, &cfg.VanitySecs)
			if err == nil {
				_ = ctx.Mgr.SaveAntinukeCfg(gid, cfg)
			}
			return err

		case "permissions", "perms":
			if !isAntinukeAdminOrOwner(ctx) {
				return ctx.Reply("[!] You do not have permission to modify antinuke settings.")
			}
			if len(ctx.Args) < 4 {
				return ctx.Reply("[!] Usage: .antinuke permissions <role|user> <permission> <on|off>")
			}
			typee := strings.ToLower(ctx.Args[1])
			permName := strings.ToLower(ctx.Args[2])
			flags := strings.ToLower(ctx.Args[3])

			if typee != "role" && typee != "user" && typee != "member" {
				return ctx.Reply("[!] Type must be `role` or `user`.")
			}

			pFlag := resolvePermFlag(permName)
			if pFlag == 0 {
				return ctx.Reply("[!] Unknown or invalid permission. Supported: administrator (admin), manage_roles (roles), manage_guild (guild), ban_members (ban), kick_members (kick), manage_webhooks (webhooks), manage_channels (channels).")
			}

			cfg, _ := ctx.Mgr.GetAntinukeCfg(gid)
			enabled := false
			if flags == "on" || flags == "enable" {
				enabled = true
			} else if flags == "off" || flags == "disable" {
				enabled = false
			} else {
				return ctx.Reply("[!] Status must be `on` or `off`.")
			}

			if typee == "role" {
				if enabled {
					found := false
					for _, p := range cfg.WatchRolePerms {
						if strings.EqualFold(p, permName) {
							found = true
							break
						}
					}
					if !found {
						cfg.WatchRolePerms = append(cfg.WatchRolePerms, permName)
					}
				} else {
					var newPerms []string
					for _, p := range cfg.WatchRolePerms {
						if !strings.EqualFold(p, permName) {
							newPerms = append(newPerms, p)
						}
					}
					cfg.WatchRolePerms = newPerms
				}
			} else {
				if enabled {
					found := false
					for _, p := range cfg.WatchUserPerms {
						if strings.EqualFold(p, permName) {
							found = true
							break
						}
					}
					if !found {
						cfg.WatchUserPerms = append(cfg.WatchUserPerms, permName)
					}
				} else {
					var newPerms []string
					for _, p := range cfg.WatchUserPerms {
						if !strings.EqualFold(p, permName) {
							newPerms = append(newPerms, p)
						}
					}
					cfg.WatchUserPerms = newPerms
				}
			}

			cfg.PermsEnabled = true
			_ = ctx.Mgr.SaveAntinukeCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("[+] Watch dangerous permission `%s` for `%s` set to `%t`.", permName, typee, enabled))

		case "config":
			if !isAntinukeAdminOrOwner(ctx) {
				return ctx.Reply("[!] You do not have permission to view antinuke configuration.")
			}
			cfg, _ := ctx.Mgr.GetAntinukeCfg(gid)
			status := "Disabled"
			if cfg.Enabled {
				status = "Enabled"
			}
			desc := fmt.Sprintf(
				"**Global Status:** `%s`\n"+
					"**Action:** `%s`\n\n"+
					"**Modules Configuration:**\n"+
					"- Channel (Mass Create/Delete): %s (Limit: `%d` / `%d`s)\n"+
					"- Webhook (Mass Create): %s (Limit: `%d` / `%d`s)\n"+
					"- Kick (Mass Member Kick): %s (Limit: `%d` / `%d`s)\n"+
					"- Ban (Mass Member Ban): %s (Limit: `%d` / `%d`s)\n"+
					"- Role (Mass Role Delete): %s (Limit: `%d` / `%d`s)\n"+
					"- Emoji (Mass Emoji Delete): %s (Limit: `%d` / `%d`s)\n"+
					"- Vanity (Mass Vanity Change): %s (Limit: `%d` / `%d`s)\n"+
					"- Bot Additions: %s\n"+
					"- Permissions Watch: %s\n\n"+
					"**Watched Role Permissions:** `%v`\n"+
					"**Watched User Permissions:** `%v`",
				status, cfg.Action,
				fmtEnabled(cfg.ChanEnabled), cfg.ChanLimit, cfg.ChanSecs,
				fmtEnabled(cfg.WebhookEnabled), cfg.WebhookLimit, cfg.WebhookSecs,
				fmtEnabled(cfg.KickEnabled), cfg.KickLimit, cfg.KickSecs,
				fmtEnabled(cfg.BanEnabled), cfg.BanLimit, cfg.BanSecs,
				fmtEnabled(cfg.RoleEnabled), cfg.RoleLimit, cfg.RoleSecs,
				fmtEnabled(cfg.EmojiEnabled), cfg.EmojiLimit, cfg.EmojiSecs,
				fmtEnabled(cfg.VanityEnabled), cfg.VanityLimit, cfg.VanitySecs,
				fmtEnabled(cfg.BotaddEnabled),
				fmtEnabled(cfg.PermsEnabled),
				cfg.WatchRolePerms,
				cfg.WatchUserPerms,
			)
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:       "Antinuke Configuration",
				Description: desc,
			})
			return ctx.Respond(emb)

		case "list":
			if !isAntinukeAdminOrOwner(ctx) {
				return ctx.Reply("[!] You do not have permission to view antinuke lists.")
			}
			cfg, _ := ctx.Mgr.GetAntinukeCfg(gid)
			var enabledModules []string
			if cfg.Enabled {
				if cfg.ChanEnabled {
					enabledModules = append(enabledModules, "Channel")
				}
				if cfg.WebhookEnabled {
					enabledModules = append(enabledModules, "Webhook")
				}
				if cfg.KickEnabled {
					enabledModules = append(enabledModules, "Kick")
				}
				if cfg.BanEnabled {
					enabledModules = append(enabledModules, "Ban")
				}
				if cfg.RoleEnabled {
					enabledModules = append(enabledModules, "Role")
				}
				if cfg.EmojiEnabled {
					enabledModules = append(enabledModules, "Emoji")
				}
				if cfg.VanityEnabled {
					enabledModules = append(enabledModules, "Vanity")
				}
				if cfg.BotaddEnabled {
					enabledModules = append(enabledModules, "BotAdd")
				}
				if cfg.PermsEnabled {
					enabledModules = append(enabledModules, "Permissions")
				}
			}

			admins, _ := ctx.DB.ListAntinukeAdmins(gid)
			whitelists, _ := ctx.DB.ListAntinukeWhitelists(gid)

			var sb strings.Builder
			sb.WriteString("**Antinuke Active Status & Whitelists**\n\n")
			sb.WriteString("**Enabled Modules:**\n")
			if len(enabledModules) == 0 {
				sb.WriteString("`None` (Antinuke is globally disabled or no modules enabled)\n")
			} else {
				sb.WriteString(fmt.Sprintf("`%s`\n", strings.Join(enabledModules, ", ")))
			}

			sb.WriteString("\n**Antinuke Admins (Manage Config):**\n")
			if len(admins) == 0 {
				sb.WriteString("- `None`\n")
			} else {
				for _, uid := range admins {
					sb.WriteString(fmt.Sprintf("- <@%s> (`%s`)\n", uid, uid))
				}
			}

			sb.WriteString("\n**Whitelisted Members / Bots (Bypass Action Limits):**\n")
			if len(whitelists) == 0 {
				sb.WriteString("- `None`\n")
			} else {
				for _, uid := range whitelists {
					sb.WriteString(fmt.Sprintf("- <@%s> (`%s`)\n", uid, uid))
				}
			}

			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:       "Antinuke Status List",
				Description: sb.String(),
			})
			return ctx.Respond(emb)

		default:
			return ctx.SendHelp("antinuke")
		}
	},
}
