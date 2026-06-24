package fun

import (
	"fmt"
	"skyvern/internal/manager"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

func init() {
	manager.RegisterHelp("impersonate", []manager.HelpPage{
		{
			Command:     "Impersonate",
			Syntax:      ".impersonate <@user> <message>",
			Description: "Send a message as another user via webhook. Requires Administrator.",
		},
	})
}

var (
	impCooldowns   = make(map[string]time.Time)
	impCooldownsMu sync.Mutex
	impCooldownDur = 10 * time.Second
)

var Impersonate = &manager.Command{
	Trigger:     "impersonate",
	Aliases:     []string{"imp"},
	Name:        "impersonate",
	Description: "Send a message as another user via webhook",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()
		uid := ctx.AuthorID()
		isOwner := isBotOwner(uid)

		var isAdmin bool
		if isOwner {
			isAdmin = true
		} else {
			p, err := ctx.Session.UserChannelPermissions(uid, ctx.ChanID())
			isAdmin = err == nil && (p&discordgo.PermissionAdministrator) != 0
			if !isAdmin {
				g, err := ctx.Session.State.Guild(gid)
				if err == nil && g.OwnerID == uid {
					isAdmin = true
				}
			}
		}

		if !isAdmin {
			return ctx.Reply(fmt.Sprintf("%s You need Administrator permission to use this.", ctx.ErrorEmoji()))
		}

		if !isOwner {
			impCooldownsMu.Lock()
			if exp, ok := impCooldowns[uid]; ok && time.Now().Before(exp) {
				left := time.Until(exp).Seconds()
				impCooldownsMu.Unlock()
				return ctx.Reply(fmt.Sprintf("%s Cooldown active. Wait %.1fs.", ctx.WarningEmoji(), left))
			}
			impCooldowns[uid] = time.Now().Add(impCooldownDur)
			impCooldownsMu.Unlock()
		}

		var target *discordgo.User
		var msg string

		if ctx.Interact != nil {
			for _, opt := range ctx.Interact.ApplicationCommandData().Options {
				switch opt.Name {
				case "user":
					target = opt.UserValue(ctx.Session)
				case "message":
					msg = opt.StringValue()
				}
			}
		} else {
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("impersonate")
			}
			targetRaw := ctx.Args[0]
			targetID := strings.TrimPrefix(strings.TrimSuffix(strings.TrimPrefix(targetRaw, "<@"), ">"), "!")
			if targetID == "" {
				return ctx.Reply(fmt.Sprintf("%s Invalid user mention.", ctx.ErrorEmoji()))
			}

			var err error
			target, err = ctx.Session.User(targetID)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("%s Could not find that user.", ctx.ErrorEmoji()))
			}
			msg = strings.Join(ctx.Args[1:], " ")
		}

		if target == nil {
			return ctx.Reply(fmt.Sprintf("%s Could not resolve target user.", ctx.ErrorEmoji()))
		}

		if isBotOwner(target.ID) && !isOwner && uid != target.ID {
			return ctx.Reply(fmt.Sprintf("%s You cannot impersonate a bot owner.", ctx.ErrorEmoji()))
		}

		if target.ID == ctx.Session.State.User.ID && !isOwner {
			return ctx.Reply(fmt.Sprintf("%s Can't impersonate me.", ctx.ErrorEmoji()))
		}

		if strings.TrimSpace(msg) == "" {
			return ctx.Reply(fmt.Sprintf("%s Message cannot be empty.", ctx.ErrorEmoji()))
		}

		if len(msg) > 2000 {
			msg = msg[:2000]
		}

		avatarURL := target.AvatarURL("256")

		name := target.Username
		mem, err := ctx.Session.GuildMember(gid, target.ID)
		if err == nil && mem.Nick != "" {
			name = mem.Nick
		}

		wh, err := ctx.Session.WebhookCreate(ctx.ChanID(), name, avatarURL)
		if err != nil {
			return ctx.Reply(fmt.Sprintf("%s Failed to create webhook. Bot needs Manage Webhooks permission.", ctx.ErrorEmoji()))
		}

		_, err = ctx.Session.WebhookExecute(wh.ID, wh.Token, false, &discordgo.WebhookParams{
			Content:   msg,
			Username:  name,
			AvatarURL: avatarURL,
		})

		_ = ctx.Session.WebhookDelete(wh.ID)

		if err != nil {
			return ctx.Reply(fmt.Sprintf("%s Failed to send webhook message.", ctx.ErrorEmoji()))
		}

		if ctx.Message != nil {
			_ = ctx.Session.ChannelMessageDelete(ctx.ChanID(), ctx.Message.ID)
		}

		if ctx.Interact != nil {
			_ = ctx.Session.InteractionRespond(ctx.Interact, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Successfully sent message.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}

		return nil
	},
}
