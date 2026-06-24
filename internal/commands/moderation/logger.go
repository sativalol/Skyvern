package moderation

import (
	"fmt"
	"regexp"
	"skyvern/internal/manager"
	"skyvern/internal/storage"
	"strings"
)

var rxHexColor = regexp.MustCompile(`^#?[0-9A-Fa-f]{6}$`)
var rxChannel = regexp.MustCompile(`^<#(\d+)>$`)
var rxMember = regexp.MustCompile(`^<@!?(\d+)>$`)

var categories = []string{"messages", "members", "roles", "channels", "invites", "emojis", "voice", "server"}

func isValidCategory(cat string) bool {
	if cat == "all" {
		return true
	}
	for _, c := range categories {
		if c == cat {
			return true
		}
	}
	return false
}

func init() {
	manager.RegisterHelp("log", []manager.HelpPage{
		{
			Command:     "Audit Logging",
			Syntax:      ".log add <#channel> <event> | .log remove <#channel> <event> | .log color <#channel> <event> <hex> | .log color list <#channel> | .log ignore <@member|#channel> | .log ignore list",
			Description: "Configure audit logging (messages, members, roles, channels, invites, emojis, voice, server, all).",
		},
	})
}

var Log = &manager.Command{
	Trigger:     "log",
	Name:        "log",
	Description: "Configure audit logging",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("log")
		}

		sub := strings.ToLower(ctx.Args[0])
		switch sub {
		case "add":
			return handleAdd(ctx)
		case "remove":
			return handleRemove(ctx)
		case "color":
			if len(ctx.Args) > 1 && strings.ToLower(ctx.Args[1]) == "list" {
				return handleColorList(ctx)
			}
			return handleColor(ctx)
		case "ignore":
			if len(ctx.Args) > 1 && strings.ToLower(ctx.Args[1]) == "list" {
				return handleIgnoreList(ctx)
			}
			return handleIgnore(ctx)
		default:
			return ctx.SendHelp("log")
		}
	},
}

func handleAdd(ctx *manager.CommandContext) error {
	if len(ctx.Args) < 3 {
		return ctx.Reply(fmt.Sprintf("%s Usage: .log add <#channel> <event>", ctx.WarningEmoji()))
	}

	cid := ""
	if m := rxChannel.FindStringSubmatch(ctx.Args[1]); len(m) > 1 {
		cid = m[1]
	} else {
		cid = ctx.Args[1]
	}

	ch, err := ctx.Session.Channel(cid)
	if err != nil || ch.GuildID != ctx.GuildID() {
		return ctx.Reply(fmt.Sprintf("%s Invalid channel.", ctx.ErrorEmoji()))
	}

	event := strings.ToLower(ctx.Args[2])
	if !isValidCategory(event) {
		return ctx.Reply(fmt.Sprintf("%s Invalid event. Valid choices: %s, all", ctx.ErrorEmoji(), strings.Join(categories, ", ")))
	}

	var cats []string
	if event == "all" {
		cats = categories
	} else {
		cats = []string{event}
	}

	for _, c := range cats {
		_ = ctx.DB.SaveLoggerSub(storage.LoggerSub{
			GuildID:   ctx.GuildID(),
			ChannelID: ch.ID,
			Category:  c,
		})
	}

	return ctx.Reply(fmt.Sprintf("%s Logging **%s** in %s.", ctx.SuccessEmoji(), event, ch.Mention()))
}

func handleRemove(ctx *manager.CommandContext) error {
	if len(ctx.Args) < 3 {
		return ctx.Reply(fmt.Sprintf("%s Usage: .log remove <#channel> <event>", ctx.WarningEmoji()))
	}

	cid := ""
	if m := rxChannel.FindStringSubmatch(ctx.Args[1]); len(m) > 1 {
		cid = m[1]
	} else {
		cid = ctx.Args[1]
	}

	ch, err := ctx.Session.Channel(cid)
	if err != nil || ch.GuildID != ctx.GuildID() {
		return ctx.Reply(fmt.Sprintf("%s Invalid channel.", ctx.ErrorEmoji()))
	}

	event := strings.ToLower(ctx.Args[2])
	if !isValidCategory(event) {
		return ctx.Reply(fmt.Sprintf("%s Invalid event. Valid choices: %s, all", ctx.ErrorEmoji(), strings.Join(categories, ", ")))
	}

	if event == "all" {
		_ = ctx.DB.DeleteAllLoggerSubs(ctx.GuildID(), ch.ID)
	} else {
		_ = ctx.DB.DeleteLoggerSub(ctx.GuildID(), ch.ID, event)
	}

	return ctx.Reply(fmt.Sprintf("%s Stopped logging **%s** in %s.", ctx.SuccessEmoji(), event, ch.Mention()))
}

func handleColor(ctx *manager.CommandContext) error {
	if len(ctx.Args) < 4 {
		return ctx.Reply(fmt.Sprintf("%s Usage: .log color <#channel> <event> <hex>", ctx.WarningEmoji()))
	}

	cid := ""
	if m := rxChannel.FindStringSubmatch(ctx.Args[1]); len(m) > 1 {
		cid = m[1]
	} else {
		cid = ctx.Args[1]
	}

	ch, err := ctx.Session.Channel(cid)
	if err != nil || ch.GuildID != ctx.GuildID() {
		return ctx.Reply(fmt.Sprintf("%s Invalid channel.", ctx.ErrorEmoji()))
	}

	event := strings.ToLower(ctx.Args[2])
	if !isValidCategory(event) || event == "all" {
		return ctx.Reply(fmt.Sprintf("%s Invalid event. Valid choices: %s", ctx.ErrorEmoji(), strings.Join(categories, ", ")))
	}

	color := ctx.Args[3]
	if !rxHexColor.MatchString(color) {
		return ctx.Reply(fmt.Sprintf("%s Invalid hex color code.", ctx.ErrorEmoji()))
	}
	if !strings.HasPrefix(color, "#") {
		color = "#" + color
	}

	subs, err := ctx.DB.GetChannelLoggerSubs(ctx.GuildID(), ch.ID)
	if err != nil {
		return ctx.Reply(fmt.Sprintf("%s Database error.", ctx.ErrorEmoji()))
	}

	var targetSub *storage.LoggerSub
	for i := range subs {
		if subs[i].Category == event {
			targetSub = &subs[i]
			break
		}
	}

	if targetSub == nil {
		return ctx.Reply(fmt.Sprintf("%s Event **%s** is not active in %s.", ctx.ErrorEmoji(), event, ch.Mention()))
	}

	targetSub.EmbedColor = color
	_ = ctx.DB.SaveLoggerSub(*targetSub)

	return ctx.Reply(fmt.Sprintf("%s Color override set for **%s** to `%s` in %s.", ctx.SuccessEmoji(), event, color, ch.Mention()))
}

func handleColorList(ctx *manager.CommandContext) error {
	if len(ctx.Args) < 3 {
		return ctx.Reply(fmt.Sprintf("%s Usage: .log color list <#channel>", ctx.WarningEmoji()))
	}

	cid := ""
	if m := rxChannel.FindStringSubmatch(ctx.Args[2]); len(m) > 1 {
		cid = m[1]
	} else {
		cid = ctx.Args[2]
	}

	ch, err := ctx.Session.Channel(cid)
	if err != nil || ch.GuildID != ctx.GuildID() {
		return ctx.Reply(fmt.Sprintf("%s Invalid channel.", ctx.ErrorEmoji()))
	}

	subs, err := ctx.DB.GetChannelLoggerSubs(ctx.GuildID(), ch.ID)
	if err != nil {
		return ctx.Reply(fmt.Sprintf("%s Database error.", ctx.ErrorEmoji()))
	}

	var lines []string
	for _, s := range subs {
		if s.EmbedColor != "" {
			lines = append(lines, fmt.Sprintf("`%s`: `%s`", s.Category, s.EmbedColor))
		}
	}

	if len(lines) == 0 {
		return ctx.Reply(fmt.Sprintf("%s No custom display colors in %s.", ctx.WarningEmoji(), ch.Mention()))
	}

	return ctx.Reply(fmt.Sprintf("%s Custom colors in %s:\n\n%s", ctx.SuccessEmoji(), ch.Mention(), strings.Join(lines, "\n")))
}

func handleIgnore(ctx *manager.CommandContext) error {
	if len(ctx.Args) < 2 {
		return ctx.Reply(fmt.Sprintf("%s Usage: .log ignore <@member|#channel>", ctx.WarningEmoji()))
	}

	target := ctx.Args[1]
	targetID := ""
	targetType := ""

	if m := rxMember.FindStringSubmatch(target); len(m) > 1 {
		targetID = m[1]
		targetType = "member"
	} else if m := rxChannel.FindStringSubmatch(target); len(m) > 1 {
		targetID = m[1]
		targetType = "channel"
	} else {
		targetID = target
		if _, err := ctx.Session.Channel(targetID); err == nil {
			targetType = "channel"
		} else if _, err := ctx.Session.GuildMember(ctx.GuildID(), targetID); err == nil {
			targetType = "member"
		}
	}

	if targetID == "" || targetType == "" {
		return ctx.Reply(fmt.Sprintf("%s Could not resolve member or channel.", ctx.ErrorEmoji()))
	}

	if ctx.DB.IsLoggerIgnored(ctx.GuildID(), targetID) {
		_ = ctx.DB.DeleteLoggerIgnore(ctx.GuildID(), targetID)
		return ctx.Reply(fmt.Sprintf("%s Removed exclusion for target `%s`.", ctx.SuccessEmoji(), targetID))
	}

	_ = ctx.DB.SaveLoggerIgnore(storage.LoggerIgnore{
		GuildID:    ctx.GuildID(),
		TargetID:   targetID,
		TargetType: targetType,
	})

	return ctx.Reply(fmt.Sprintf("%s Excluded target `%s` (%s) from audit logging.", ctx.SuccessEmoji(), targetID, targetType))
}

func handleIgnoreList(ctx *manager.CommandContext) error {
	ignores, err := ctx.DB.GetLoggerIgnores(ctx.GuildID())
	if err != nil {
		return ctx.Reply(fmt.Sprintf("%s Database error.", ctx.ErrorEmoji()))
	}

	if len(ignores) == 0 {
		return ctx.Reply(fmt.Sprintf("%s No entities are excluded from logging.", ctx.WarningEmoji()))
	}

	var lines []string
	for _, ig := range ignores {
		mention := ""
		if ig.TargetType == "member" {
			mention = fmt.Sprintf("<@%s>", ig.TargetID)
		} else {
			mention = fmt.Sprintf("<#%s>", ig.TargetID)
		}
		lines = append(lines, fmt.Sprintf("%s (%s)", mention, ig.TargetType))
	}

	return ctx.Reply(fmt.Sprintf("%s Excluded from logs:\n\n%s", ctx.SuccessEmoji(), strings.Join(lines, "\n")))
}
