package general

import (
	"fmt"
	"regexp"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/storage"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var rxBumpChan = regexp.MustCompile(`^<#(\d+)>$`)

var BumpReminder = &manager.Command{
	Trigger:     "bumpreminder",
	Aliases:     []string{"breminder", "bump"},
	Name:        "bumpreminder",
	Description: "Configure bump reminders",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		if !checkPerm(ctx, discordgo.PermissionManageServer) {
			return ctx.Reply(fmt.Sprintf("%s You need Manage Server permission to configure bump reminders.", ctx.ErrorEmoji()))
		}

		gid := ctx.GuildID()
		cfg, err := ctx.DB.GetBumpCfg(gid)
		if err != nil {
			cfg = storage.BumpCfg{
				Enabled:          false,
				Message:          "It is time to bump the server! Use /bump.",
				ThankYouMessage:  "Thank you for bumping the server!",
				AutoClean:        false,
				AutoLock:         false,
			}
		}

		if len(ctx.Args) == 0 {
			return showBumpConfig(ctx, cfg)
		}

		sub := strings.ToLower(ctx.Args[0])

		switch sub {
		case "on", "enable":
			if cfg.ChannelID == "" {
				return ctx.Reply(fmt.Sprintf("%s Please configure a channel first using `.bumpreminder channel <#channel>`.", ctx.WarningEmoji()))
			}
			cfg.Enabled = true
			_ = ctx.DB.SaveBumpCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Bump reminders enabled.", ctx.SuccessEmoji()))

		case "off", "disable":
			cfg.Enabled = false
			_ = ctx.DB.SaveBumpCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Bump reminders disabled.", ctx.SuccessEmoji()))

		case "channel", "chan":
			if len(ctx.Args) < 2 {
				return ctx.Reply(fmt.Sprintf("%s Usage: `.bumpreminder channel <#channel>`", ctx.WarningEmoji()))
			}
			chanArg := ctx.Args[1]
			cid := ""
			if m := rxBumpChan.FindStringSubmatch(chanArg); len(m) > 1 {
				cid = m[1]
			} else {
				cid = chanArg
			}

			ch, err := ctx.Session.Channel(cid)
			if err != nil || ch.GuildID != gid {
				return ctx.Reply(fmt.Sprintf("%s Could not resolve text channel.", ctx.ErrorEmoji()))
			}

			cfg.ChannelID = cid
			_ = ctx.DB.SaveBumpCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Bump reminders will be sent to <#%s>.", ctx.SuccessEmoji(), cid))

		case "message", "msg":
			if len(ctx.Args) < 2 {
				return ctx.Reply(fmt.Sprintf("%s Usage: `.bumpreminder message <text>` or `.bumpreminder message view`", ctx.WarningEmoji()))
			}
			if strings.ToLower(ctx.Args[1]) == "view" {
				return ctx.Reply(fmt.Sprintf("%s Current reminder message:\n```\n%s\n```", ctx.SuccessEmoji(), cfg.Message))
			}
			cfg.Message = strings.Join(ctx.Args[1:], " ")
			_ = ctx.DB.SaveBumpCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Set bump reminder message to:\n%s", ctx.SuccessEmoji(), cfg.Message))

		case "thankyou", "thanks", "ty":
			if len(ctx.Args) < 2 {
				return ctx.Reply(fmt.Sprintf("%s Usage: `.bumpreminder thankyou <text>` or `.bumpreminder thankyou view`", ctx.WarningEmoji()))
			}
			if strings.ToLower(ctx.Args[1]) == "view" {
				return ctx.Reply(fmt.Sprintf("%s Current thank you message:\n```\n%s\n```", ctx.SuccessEmoji(), cfg.ThankYouMessage))
			}
			cfg.ThankYouMessage = strings.Join(ctx.Args[1:], " ")
			_ = ctx.DB.SaveBumpCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Set thank you message to:\n%s", ctx.SuccessEmoji(), cfg.ThankYouMessage))

		case "autoclean", "clean":
			if len(ctx.Args) < 2 {
				return ctx.Reply(fmt.Sprintf("%s Usage: `.bumpreminder autoclean <on|off>`", ctx.WarningEmoji()))
			}
			choice := strings.ToLower(ctx.Args[1])
			if choice == "on" || choice == "true" {
				cfg.AutoClean = true
			} else if choice == "off" || choice == "false" {
				cfg.AutoClean = false
			} else {
				return ctx.Reply(fmt.Sprintf("%s Choice must be on or off.", ctx.ErrorEmoji()))
			}
			_ = ctx.DB.SaveBumpCfg(gid, cfg)
			status := "disabled"
			if cfg.AutoClean {
				status = "enabled"
			}
			return ctx.Reply(fmt.Sprintf("%s AutoClean is now %s.", ctx.SuccessEmoji(), status))

		case "autolock", "lock":
			if len(ctx.Args) < 2 {
				return ctx.Reply(fmt.Sprintf("%s Usage: `.bumpreminder autolock <on|off>`", ctx.WarningEmoji()))
			}
			choice := strings.ToLower(ctx.Args[1])
			if choice == "on" || choice == "true" {
				cfg.AutoLock = true
			} else if choice == "off" || choice == "false" {
				cfg.AutoLock = false
			} else {
				return ctx.Reply(fmt.Sprintf("%s Choice must be on or off.", ctx.ErrorEmoji()))
			}
			_ = ctx.DB.SaveBumpCfg(gid, cfg)
			status := "disabled"
			if cfg.AutoLock {
				status = "enabled"
			}
			return ctx.Reply(fmt.Sprintf("%s AutoLock is now %s.", ctx.SuccessEmoji(), status))

		case "config", "settings", "view":
			return showBumpConfig(ctx, cfg)

		default:
			return ctx.Reply(fmt.Sprintf("%s Unknown subcommand. Use channel, message, thankyou, autoclean, autolock, config, enable, or disable.", ctx.ErrorEmoji()))
		}
	},
}

func showBumpConfig(ctx *manager.CommandContext, cfg storage.BumpCfg) error {
	status := "Disabled"
	if cfg.Enabled {
		status = "Enabled"
	}
	chanStr := "Not set"
	if cfg.ChannelID != "" {
		chanStr = fmt.Sprintf("<#%s>", cfg.ChannelID)
	}

	cleanStr := "Disabled"
	if cfg.AutoClean {
		cleanStr = "Enabled"
	}
	lockStr := "Disabled"
	if cfg.AutoLock {
		lockStr = "Enabled"
	}

	desc := fmt.Sprintf(
		"**Status:** `%s`\n"+
			"**Channel:** %s\n"+
			"**AutoClean:** `%s`\n"+
			"**AutoLock:** `%s`\n\n"+
			"**Reminder Message:**\n%s\n\n"+
			"**Thank You Message:**\n%s",
		status, chanStr, cleanStr, lockStr, cfg.Message, cfg.ThankYouMessage,
	)

	emb := config.Build(ctx.Cfg, config.EmbedOpt{
		Title:       "Bump Reminder Configuration",
		Description: desc,
	})
	return ctx.Respond(emb)
}
