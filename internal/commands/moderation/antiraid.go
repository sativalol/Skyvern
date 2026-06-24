package moderation

import (
	"fmt"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/storage"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var Antiraid = &manager.Command{
	Trigger:     "antiraid",
	Aliases:     []string{"ar"},
	Name:        "antiraid",
	Description: "Manage antiraid thresholds and actions",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()
		cfg, err := ctx.Mgr.GetAntiraidCfg(gid)
		if err != nil {
			cfg = storage.AntiraidCfg{
				Enabled:         false,
				JoinLimit:       10,
				Seconds:         10,
				Action:          "notify",
				AvatarEnabled:   false,
				AvatarAction:    "kick",
				NewAcctsEnabled: false,
				NewAcctsAgeMins: 1440,
				NewAcctsAction:  "kick",
				Whitelist:       []string{},
				RaidActive:      false,
			}
		}

		if len(ctx.Args) == 0 {
			return showAntiraidConfig(ctx, cfg)
		}

		sub := strings.ToLower(ctx.Args[0])

		if sub == "config" || sub == "settings" || sub == "status" {
			return showAntiraidConfig(ctx, cfg)
		}

		if !checkPerm(ctx, discordgo.PermissionManageGuild) {
			return ctx.Reply(fmt.Sprintf("%s Manage Guild permission required.", ctx.ErrorEmoji()))
		}

		switch sub {
		case "on", "enable":
			cfg.Enabled = true
			_ = ctx.Mgr.SaveAntiraidCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Antiraid engine enabled.", ctx.SuccessEmoji()))

		case "off", "disable":
			cfg.Enabled = false
			_ = ctx.Mgr.SaveAntiraidCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Antiraid engine disabled.", ctx.SuccessEmoji()))

		case "state":
			cfg.RaidActive = false
			_ = ctx.Mgr.SaveAntiraidCfg(gid, cfg)
			ctx.Mgr.UnlockGuild(ctx.Session, gid)
			return ctx.Reply(fmt.Sprintf("%s Server raid state deactivated and lockdown channel overrides removed.", ctx.SuccessEmoji()))

		case "avatar":
			if len(ctx.Args) < 2 {
				return ctx.Reply(fmt.Sprintf("%s Usage: `.antiraid avatar <on|off|kick|ban>`", ctx.WarningEmoji()))
			}
			val := strings.ToLower(ctx.Args[1])

			actionFlag := ""
			for i, arg := range ctx.Args {
				if (arg == "--action" || arg == "-action") && i+1 < len(ctx.Args) {
					actionFlag = strings.ToLower(ctx.Args[i+1])
				}
			}

			if val == "on" || val == "enable" {
				cfg.AvatarEnabled = true
				if actionFlag == "kick" || actionFlag == "ban" {
					cfg.AvatarAction = actionFlag
				}
			} else if val == "off" || val == "disable" {
				cfg.AvatarEnabled = false
			} else if val == "kick" || val == "ban" {
				cfg.AvatarEnabled = true
				cfg.AvatarAction = val
			} else {
				return ctx.Reply(fmt.Sprintf("%s Choice must be on, off, kick, or ban.", ctx.ErrorEmoji()))
			}

			_ = ctx.Mgr.SaveAntiraidCfg(gid, cfg)
			statusStr := "disabled"
			if cfg.AvatarEnabled {
				statusStr = fmt.Sprintf("enabled (action: %s)", cfg.AvatarAction)
			}
			return ctx.Reply(fmt.Sprintf("%s Avatar protection is now %s.", ctx.SuccessEmoji(), statusStr))

		case "newaccounts", "newaccount", "newaccts":
			if len(ctx.Args) < 2 {
				return ctx.Reply(fmt.Sprintf("%s Usage: `.antiraid newaccounts <on|off|kick|ban> [--age <duration>]`", ctx.WarningEmoji()))
			}
			val := strings.ToLower(ctx.Args[1])

			ageFlagVal := ""
			actionFlag := ""
			for i := 0; i < len(ctx.Args); i++ {
				arg := ctx.Args[i]
				if (arg == "--age" || arg == "-age") && i+1 < len(ctx.Args) {
					ageFlagVal = ctx.Args[i+1]
				}
				if (arg == "--action" || arg == "-action") && i+1 < len(ctx.Args) {
					actionFlag = strings.ToLower(ctx.Args[i+1])
				}
			}

			if val == "on" || val == "enable" {
				cfg.NewAcctsEnabled = true
			} else if val == "off" || val == "disable" {
				cfg.NewAcctsEnabled = false
			} else if val == "kick" || val == "ban" {
				cfg.NewAcctsEnabled = true
				cfg.NewAcctsAction = val
			} else {
				return ctx.Reply(fmt.Sprintf("%s Choice must be on, off, kick, or ban.", ctx.ErrorEmoji()))
			}

			if ageFlagVal != "" {
				mins := parseDurationMins(ageFlagVal)
				if mins > 0 {
					cfg.NewAcctsAgeMins = mins
				} else {
					return ctx.Reply(fmt.Sprintf("%s Invalid age duration. Use format like 24h, 1d, 60m.", ctx.ErrorEmoji()))
				}
			}

			if actionFlag == "kick" || actionFlag == "ban" {
				cfg.NewAcctsAction = actionFlag
			}

			_ = ctx.Mgr.SaveAntiraidCfg(gid, cfg)
			statusStr := "disabled"
			if cfg.NewAcctsEnabled {
				statusStr = fmt.Sprintf("enabled (age limit: %d mins, action: %s)", cfg.NewAcctsAgeMins, cfg.NewAcctsAction)
			}
			return ctx.Reply(fmt.Sprintf("%s New accounts protection is now %s.", ctx.SuccessEmoji(), statusStr))

		case "whitelist":
			if len(ctx.Args) < 2 {
				return ctx.Reply(fmt.Sprintf("%s Usage: `.antiraid whitelist <member>` or `.antiraid whitelist view`", ctx.WarningEmoji()))
			}
			val := ctx.Args[1]
			if strings.ToLower(val) == "view" {
				if len(cfg.Whitelist) == 0 {
					return ctx.Reply(fmt.Sprintf("%s There are no whitelisted users.", ctx.WarningEmoji()))
				}
				var list []string
				for _, uid := range cfg.Whitelist {
					list = append(list, fmt.Sprintf("<@%s> (`%s`)", uid, uid))
				}
				return ctx.Reply(fmt.Sprintf("One-time whitelisted users:\n%s", strings.Join(list, "\n")))
			}

			targetID := ""
			if m := rxMember.FindStringSubmatch(val); len(m) > 1 {
				targetID = m[1]
			} else {
				targetID = val
			}

			if _, err := strconv.ParseUint(targetID, 10, 64); err != nil {
				return ctx.Reply(fmt.Sprintf("%s Invalid user ID or mention.", ctx.ErrorEmoji()))
			}

			exists := false
			for _, wid := range cfg.Whitelist {
				if wid == targetID {
					exists = true
					break
				}
			}
			if !exists {
				cfg.Whitelist = append(cfg.Whitelist, targetID)
				_ = ctx.Mgr.SaveAntiraidCfg(gid, cfg)
			}
			return ctx.Reply(fmt.Sprintf("%s User <@%s> added to one-time whitelist.", ctx.SuccessEmoji(), targetID))

		case "massjoin":
			if len(ctx.Args) < 2 {
				return ctx.Reply(fmt.Sprintf("%s Usage: `.antiraid massjoin <on|off|kick|ban|lockdown|notify|limit> [flags]`", ctx.WarningEmoji()))
			}
			val := strings.ToLower(ctx.Args[1])

			limitFlagVal := 0
			secondsFlagVal := 0
			actionFlag := ""
			for i := 0; i < len(ctx.Args); i++ {
				arg := ctx.Args[i]
				if (arg == "--limit" || arg == "-limit") && i+1 < len(ctx.Args) {
					limitFlagVal, _ = strconv.Atoi(ctx.Args[i+1])
				}
				if (arg == "--seconds" || arg == "-seconds" || arg == "--window" || arg == "-window") && i+1 < len(ctx.Args) {
					secondsFlagVal, _ = strconv.Atoi(ctx.Args[i+1])
				}
				if (arg == "--action" || arg == "-action") && i+1 < len(ctx.Args) {
					actionFlag = strings.ToLower(ctx.Args[i+1])
				}
			}

			if val == "on" || val == "enable" {
				cfg.Enabled = true
			} else if val == "off" || val == "disable" {
				cfg.Enabled = false
			} else if val == "kick" || val == "ban" || val == "lockdown" || val == "notify" {
				cfg.Enabled = true
				cfg.Action = val
			} else {
				if strings.Contains(val, "/") {
					parts := strings.Split(val, "/")
					if len(parts) == 2 {
						lim, _ := strconv.Atoi(parts[0])
						secStr := strings.TrimSuffix(parts[1], "s")
						sec, _ := strconv.Atoi(secStr)
						if lim > 0 && sec > 0 {
							cfg.JoinLimit = lim
							cfg.Seconds = sec
							cfg.Enabled = true
						}
					}
				} else if lim, err := strconv.Atoi(val); err == nil && lim > 0 {
					cfg.JoinLimit = lim
					cfg.Enabled = true
				} else {
					return ctx.Reply(fmt.Sprintf("%s Invalid massjoin option. Use on/off, kick/ban/lockdown/notify, or threshold.", ctx.ErrorEmoji()))
				}
			}

			if limitFlagVal > 0 {
				cfg.JoinLimit = limitFlagVal
			}
			if secondsFlagVal > 0 {
				cfg.Seconds = secondsFlagVal
			}
			if actionFlag == "kick" || actionFlag == "ban" || actionFlag == "lockdown" || actionFlag == "notify" {
				cfg.Action = actionFlag
			}

			_ = ctx.Mgr.SaveAntiraidCfg(gid, cfg)
			statusStr := "disabled"
			if cfg.Enabled {
				statusStr = fmt.Sprintf("enabled (rate: %d joins in %d seconds, action: %s)", cfg.JoinLimit, cfg.Seconds, cfg.Action)
			}
			return ctx.Reply(fmt.Sprintf("%s Massjoin protection is now %s.", ctx.SuccessEmoji(), statusStr))

		default:
			return ctx.Reply(fmt.Sprintf("%s Unknown subcommand. Use config, state, avatar, newaccounts, whitelist, or massjoin.", ctx.ErrorEmoji()))
		}
	},
}

func parseDurationMins(s string) int {
	s = strings.ToLower(s)
	if strings.HasSuffix(s, "d") {
		d, _ := strconv.Atoi(strings.TrimSuffix(s, "d"))
		return d * 1440
	}
	if strings.HasSuffix(s, "h") {
		h, _ := strconv.Atoi(strings.TrimSuffix(s, "h"))
		return h * 60
	}
	if strings.HasSuffix(s, "m") {
		m, _ := strconv.Atoi(strings.TrimSuffix(s, "m"))
		return m
	}
	if strings.HasSuffix(s, "s") {
		sec, _ := strconv.Atoi(strings.TrimSuffix(s, "s"))
		return sec / 60
	}
	v, _ := strconv.Atoi(s)
	return v
}

func showAntiraidConfig(ctx *manager.CommandContext, cfg storage.AntiraidCfg) error {
	status := "Disabled"
	if cfg.Enabled {
		status = "Enabled"
	}
	raidState := "Normal"
	if cfg.RaidActive {
		raidState = "⚠️ Raid Active / Lockdown"
	}

	avatarStatus := "Disabled"
	if cfg.AvatarEnabled {
		avatarStatus = fmt.Sprintf("Enabled (Action: %s)", cfg.AvatarAction)
	}

	newAcctsStatus := "Disabled"
	if cfg.NewAcctsEnabled {
		newAcctsStatus = fmt.Sprintf("Enabled (Age: %d mins, Action: %s)", cfg.NewAcctsAgeMins, cfg.NewAcctsAction)
	}

	desc := fmt.Sprintf(
		"**Server Raid State:** `%s`\n"+
			"**Engine Enabled:** `%s`\n\n"+
			"**Massjoin Protection (Rate):**\n"+
			"- Limit: `%d` joins\n"+
			"- Window: `%d` seconds\n"+
			"- Mitigation: `%s`\n\n"+
			"**Avatar Protection:** `%s`\n"+
			"**New Accounts Protection:** `%s`\n"+
			"**Whitelist Count:** `%d` users",
		raidState, status, cfg.JoinLimit, cfg.Seconds, cfg.Action,
		avatarStatus, newAcctsStatus, len(cfg.Whitelist),
	)

	emb := config.Build(ctx.Cfg, config.EmbedOpt{
		Title:       "Antiraid Configuration",
		Description: desc,
	})
	return ctx.Respond(emb)
}
