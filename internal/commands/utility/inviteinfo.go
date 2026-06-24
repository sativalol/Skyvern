package utility

import (
	"fmt"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func init() {
	manager.RegisterHelp("inviteinfo", []manager.HelpPage{
		{
			Command:     "Invite Info",
			Syntax:      ".inviteinfo <code>",
			Description: "Gets details about a Discord invite link or code.",
		},
	})
}

var InviteInfo = &manager.Command{
	Trigger:     "inviteinfo",
	Aliases:     []string{"invite"},
	Name:        "inviteinfo",
	Description: "View basic invite code information",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("inviteinfo")
		}

		code := ctx.Args[0]
		if strings.Contains(code, "discord.gg/") {
			parts := strings.Split(code, "discord.gg/")
			code = parts[len(parts)-1]
		} else if strings.Contains(code, "discord.com/invite/") {
			parts := strings.Split(code, "discord.com/invite/")
			code = parts[len(parts)-1]
		}
		code = strings.Split(code, "?")[0]

		inv, err := ctx.Session.Invite(code)
		if err != nil {
			return ctx.Reply("[!] Invite code not found or expired.")
		}

		gName := "Unknown Guild"
		gID := ""
		if inv.Guild != nil {
			gName = inv.Guild.Name
			gID = inv.Guild.ID
		}

		var fields []*discordgo.MessageEmbedField
		fields = append(fields, config.Field("Server Name", gName, true))
		if gID != "" {
			fields = append(fields, config.Field("Server ID", fmt.Sprintf("`%s`", gID), true))
		}
		if inv.Channel != nil {
			fields = append(fields, config.Field("Channel", fmt.Sprintf("#%s (ID: %s)", inv.Channel.Name, inv.Channel.ID), false))
		}
		if inv.Inviter != nil {
			fields = append(fields, config.Field("Inviter", fmt.Sprintf("%s (%s)", inv.Inviter.Username, inv.Inviter.ID), false))
		}
		fields = append(fields, config.Field("Approximate Members", fmt.Sprintf("Online: **%d** | Total: **%d**", inv.ApproximatePresenceCount, inv.ApproximateMemberCount), false))

		if !inv.CreatedAt.IsZero() {
			fields = append(fields, config.Field("Created At", fmt.Sprintf("<t:%d:F> (<t:%d:R>)", inv.CreatedAt.Unix(), inv.CreatedAt.Unix()), false))
		}

		if inv.ExpiresAt != nil {
			fields = append(fields, config.Field("Expires At", fmt.Sprintf("<t:%d:F> (<t:%d:R>)", inv.ExpiresAt.Unix(), inv.ExpiresAt.Unix()), false))
		}

		tempStr := "False"
		if inv.Temporary {
			tempStr = "True"
		}
		fields = append(fields, config.Field("Temporary Membership", tempStr, true))

		maxUsesStr := "Unlimited"
		if inv.MaxUses > 0 {
			maxUsesStr = fmt.Sprintf("%d uses", inv.MaxUses)
		}
		fields = append(fields, config.Field("Uses", fmt.Sprintf("%d / %s", inv.Uses, maxUsesStr), true))

		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:  "Invite Code Information: " + inv.Code,
			Fields: fields,
		})
		return ctx.Respond(emb)
	},
}
