package captcha

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"skyvern/internal/manager"
)

func captchaCommands(p *CaptchaPlugin) []*manager.Command {
	return []*manager.Command{
		{
			Trigger:     "verify",
			Name:        "verify",
			Description: "Verification system configuration and panel deployment",
			Category:    "general",
			Execute: func(ctx *manager.CommandContext) error {
				if len(ctx.Args) == 0 {
					return ctx.Reply("Unknown subcommand. Options: `panel`, `status`, `config`, `autosetup`, `remove`")
				}

				sub := strings.ToLower(ctx.Args[0])
				gid := ctx.GuildID()
				if gid == "" {
					return ctx.Reply("Must be used in a server.")
				}

				switch sub {
				case "panel":
					title := "Server Verification Required"
					desc := "Click the button below to start the verification process."
					if len(ctx.Args) > 1 {
						title = ctx.Args[1]
					}
					if len(ctx.Args) > 2 {
						desc = strings.Join(ctx.Args[2:], " ")
					}

					emb := &discordgo.MessageEmbed{
						Title:       title,
						Description: desc,
						Color:       0x2b2d31,
						Footer: &discordgo.MessageEmbedFooter{
							Text:    ctx.Cfg.Footer,
							IconURL: ctx.Cfg.FooterIcon,
						},
					}

					comp := []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.Button{
									Label:    "Start Verification",
									Style:    discordgo.PrimaryButton,
									CustomID: "captcha_start:" + gid,
									Emoji: &discordgo.ComponentEmoji{
										Name: "🔐",
									},
								},
							},
						},
					}

					_, err := ctx.Session.ChannelMessageSendComplex(ctx.ChanID(), &discordgo.MessageSend{
						Embeds:     []*discordgo.MessageEmbed{emb},
						Components: comp,
					})
					return err

				case "status":
					cfg := getCaptchaCfg(p.db, gid)
					enabledStr := "No"
					if cfg.Enabled {
						enabledStr = "Yes"
					}
					vRole := "None"
					if cfg.VerifiedRoleID != "" {
						vRole = "<@&" + cfg.VerifiedRoleID + ">"
					}
					uRole := "None"
					if cfg.UnverifiedRoleID != "" {
						uRole = "<@&" + cfg.UnverifiedRoleID + ">"
					}

					emb := &discordgo.MessageEmbed{
						Title: "Esoterica Security Status",
						Color: 0xB2CCD5,
						Fields: []*discordgo.MessageEmbedField{
							{Name: "Captcha Enabled", Value: enabledStr, Inline: true},
							{Name: "Verified Role", Value: vRole, Inline: true},
							{Name: "Unverified Role", Value: uRole, Inline: true},
							{Name: "Max Attempts", Value: strconv.Itoa(cfg.MaxAttempts), Inline: true},
							{Name: "Failure Action", Value: strings.Title(cfg.FailureAction), Inline: true},
							{Name: "Timeout (Minutes)", Value: strconv.Itoa(cfg.TimeoutMinutes), Inline: true},
						},
						Footer: &discordgo.MessageEmbedFooter{
							Text:    ctx.Cfg.Footer,
							IconURL: ctx.Cfg.FooterIcon,
						},
					}

					_, err := ctx.Session.ChannelMessageSendEmbed(ctx.ChanID(), emb)
					return err

				case "config":
					if len(ctx.Args) < 3 {
						return ctx.Reply("Usage: `.verify config <on/off> <@verified_role> [unverified_role/@role/none] [max_attempts] [kick/ban/none]`")
					}

					enabled := ctx.Args[1] == "on" || ctx.Args[1] == "yes" || ctx.Args[1] == "true"
					vRole := parseRoleMention(ctx.Args[2])
					if vRole == "" {
						return ctx.Reply("[!] Invalid verified role.")
					}

					uRole := ""
					if len(ctx.Args) > 3 && ctx.Args[3] != "none" {
						uRole = parseRoleMention(ctx.Args[3])
					}

					maxAttempts := 3
					if len(ctx.Args) > 4 {
						if val, err := strconv.Atoi(ctx.Args[4]); err == nil && val > 0 {
							maxAttempts = val
						}
					}

					failAction := "kick"
					if len(ctx.Args) > 5 {
						act := strings.ToLower(ctx.Args[5])
						if act == "ban" || act == "none" || act == "kick" {
							failAction = act
						}
					}

					cfg := CaptchaConfig{
						Enabled:          enabled,
						VerifiedRoleID:   vRole,
						UnverifiedRoleID: uRole,
						MaxAttempts:      maxAttempts,
						FailureAction:    failAction,
						TimeoutMinutes:   5,
					}

					_ = saveCaptchaCfg(p.db, gid, cfg)
					return ctx.Reply(fmt.Sprintf("[+] Captcha config saved successfully! Status: %t, Verified Role: <@&%s>", enabled, vRole))

				case "autosetup":
					perms, err := ctx.Session.UserChannelPermissions(ctx.AuthorID(), ctx.ChanID())
					if err != nil || (perms&discordgo.PermissionManageGuild) == 0 {
						return ctx.Reply("[!] You need Manage Guild permission to run autosetup.")
					}

					_ = ctx.Reply("⚙️ **Starting verification autosetup...**")

					var vRole *discordgo.Role
					roles, err := ctx.Session.GuildRoles(gid)
					if err == nil {
						for _, r := range roles {
							if r.Name == "Verified" {
								vRole = r
								break
							}
						}
					}
					if vRole == nil {
						vRole, err = ctx.Session.GuildRoleCreate(gid, &discordgo.RoleParams{
							Name: "Verified",
						})
						if err != nil {
							return ctx.Reply(fmt.Sprintf("❌ Failed to create Verified role: %v", err))
						}
					}

					var uRole *discordgo.Role
					if err == nil {
						for _, r := range roles {
							if r.Name == "Unverified" {
								uRole = r
								break
							}
						}
					}
					if uRole == nil {
						uRole, err = ctx.Session.GuildRoleCreate(gid, &discordgo.RoleParams{
							Name: "Unverified",
						})
						if err != nil {
							return ctx.Reply(fmt.Sprintf("❌ Failed to create Unverified role: %v", err))
						}
					}

					var vChan *discordgo.Channel
					chans, err := ctx.Session.GuildChannels(gid)
					if err == nil {
						for _, c := range chans {
							if c.Name == "verify" || c.Name == "verification" {
								vChan = c
								break
							}
						}
					}

					if vChan == nil {
						vChan, err = ctx.Session.GuildChannelCreateComplex(gid, discordgo.GuildChannelCreateData{
							Name: "verify",
							Type: discordgo.ChannelTypeGuildText,
							PermissionOverwrites: []*discordgo.PermissionOverwrite{
								{
									ID:   gid,
									Type: discordgo.PermissionOverwriteTypeRole,
									Deny: discordgo.PermissionViewChannel,
								},
								{
									ID:    uRole.ID,
									Type:  discordgo.PermissionOverwriteTypeRole,
									Allow: discordgo.PermissionViewChannel | discordgo.PermissionReadMessageHistory,
									Deny:  discordgo.PermissionSendMessages,
								},
								{
									ID:   vRole.ID,
									Type: discordgo.PermissionOverwriteTypeRole,
									Deny: discordgo.PermissionViewChannel,
								},
							},
						})
						if err != nil {
							return ctx.Reply(fmt.Sprintf("❌ Failed to create verification channel: %v", err))
						}
					}

					cfg := CaptchaConfig{
						Enabled:          true,
						VerifiedRoleID:   vRole.ID,
						UnverifiedRoleID: uRole.ID,
						MaxAttempts:      3,
						FailureAction:    "kick",
						TimeoutMinutes:   5,
					}
					_ = saveCaptchaCfg(p.db, gid, cfg)

					emb := &discordgo.MessageEmbed{
						Title:       "Server Verification Required",
						Description: "Click the button below to start the verification process and gain access to the server.",
						Color:       0x2b2d31,
						Footer: &discordgo.MessageEmbedFooter{
							Text:    ctx.Cfg.Footer,
							IconURL: ctx.Cfg.FooterIcon,
						},
					}

					comp := []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.Button{
									Label:    "Start Verification",
									Style:    discordgo.PrimaryButton,
									CustomID: "captcha_start:" + gid,
									Emoji: &discordgo.ComponentEmoji{
										Name: "🔐",
									},
								},
							},
						},
					}

					_, err = ctx.Session.ChannelMessageSendComplex(vChan.ID, &discordgo.MessageSend{
						Embeds:     []*discordgo.MessageEmbed{emb},
						Components: comp,
					})
					if err != nil {
						return ctx.Reply(fmt.Sprintf("❌ Setup complete but failed to deploy panel: %v", err))
					}

					return ctx.Reply(fmt.Sprintf("✅ **Autosetup completed successfully!**\n\n- Created/Resolved Role: <@&%s> (Verified)\n- Created/Resolved Role: <@&%s> (Unverified)\n- Created/Resolved Channel: <#%s>\n- Deployed verification panel in <#%s>.\n\n*Note: Remember to configure other channels so that only the 'Verified' role can access them.*", vRole.ID, uRole.ID, vChan.ID, vChan.ID))

				case "remove":
					perms, err := ctx.Session.UserChannelPermissions(ctx.AuthorID(), ctx.ChanID())
					if err != nil || (perms&discordgo.PermissionManageGuild) == 0 {
						return ctx.Reply("[!] You need Manage Guild permission to run verify remove.")
					}

					if len(ctx.Args) < 2 {
						return ctx.Reply("Usage: `.verify remove <@user>`")
					}

					targetID := parseUserMention(ctx.Args[1])
					if targetID == "" {
						return ctx.Reply("[!] Invalid user mention.")
					}

					cfg := getCaptchaCfg(p.db, gid)

					if cfg.VerifiedRoleID != "" {
						_ = ctx.Session.GuildMemberRoleRemove(gid, targetID, cfg.VerifiedRoleID)
					}
					if cfg.UnverifiedRoleID != "" {
						_ = ctx.Session.GuildMemberRoleAdd(gid, targetID, cfg.UnverifiedRoleID)
					}

					return ctx.Reply(fmt.Sprintf("✅ Removed verification for <@%s>.", targetID))

				default:
					return ctx.Reply("Unknown subcommand. Options: `panel`, `status`, `config`, `autosetup`, `remove`")
				}
			},
		},
	}
}

func parseRoleMention(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "<@&") && strings.HasSuffix(s, ">") {
		return s[3 : len(s)-1]
	}
	return s
}

func parseUserMention(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "<@") && strings.HasSuffix(s, ">") {
		s = strings.TrimPrefix(s, "<@")
		s = strings.TrimSuffix(s, ">")
		s = strings.TrimPrefix(s, "!")
	}
	return s
}
