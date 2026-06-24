package utility
import (
	"encoding/json"
	"fmt"
	"regexp"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/moderation"
	"sort"
	"strings"
	"time"
	"github.com/bwmarrin/discordgo"
)

type EmbedJson struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       int    `json:"color"`
	Thumbnail   string `json:"thumbnail"`
	Image       string `json:"image"`
	Footer      struct {
		Text string `json:"text"`
		Icon string `json:"icon"`
	} `json:"footer"`
	Fields []struct {
		Name   string `json:"name"`
		Value  string `json:"value"`
		Inline bool   `json:"inline"`
	} `json:"fields"`
}

func init() {
	manager.RegisterHelp("embed", []manager.HelpPage{
		{
			Command:     "Embed Create",
			Syntax:      ".embed create <name> <json>",
			Description: "Create a new custom named embed from a JSON template.",
		},
		{
			Command:     "Embed Edit",
			Syntax:      ".embed edit <name> <json>",
			Description: "Edit an existing saved named embed.",
		},
		{
			Command:     "Embed Preview",
			Syntax:      ".embed preview <name>",
			Description: "Preview a saved embed template in the channel without writing it.",
		},
		{
			Command:     "Embed List",
			Syntax:      ".embed list",
			Description: "List all saved embed templates on the server.",
		},
		{
			Command:     "Embed Delete",
			Syntax:      ".embed delete <name>",
			Description: "Delete a saved named embed template.",
		},
		{
			Command:     "Embed Copy",
			Syntax:      ".embed copy <msg_link> [name]",
			Description: "Extract an embed structure from a Discord message link.",
		},
	})
	manager.RegisterHelp("names", []manager.HelpPage{
		{
			Command:     "Names History",
			Syntax:      ".names [member]",
			Description: "View historical usernames and nicknames logged for a server member.",
		},
	})
	manager.RegisterHelp("gnames", []manager.HelpPage{
		{
			Command:     "Guild Names",
			Syntax:      ".gnames",
			Description: "View historical name changes for this server.",
		},
	})
	manager.RegisterHelp("invites", []manager.HelpPage{
		{
			Command:     "Invites",
			Syntax:      ".invites",
			Description: "List all active invite codes, creators, uses, and expiration dates.",
		},
	})
	manager.RegisterHelp("topcommands", []manager.HelpPage{
		{
			Command:     "Top Commands",
			Syntax:      ".topcommands",
			Description: "Display a leaderboard of the most executed commands on the server.",
		},
	})
}
var rxMsgLink = regexp.MustCompile(`channels/(\d+)/(\d+)/(\d+)`)
var rxEmbedChan = regexp.MustCompile(`^<#(\d+)>$`)
func parseMessageLink(link string) (string, string, string) {
	m := rxMsgLink.FindStringSubmatch(link)
	if len(m) >= 4 {
		return m[1], m[2], m[3]
	}
	return "", "", ""
}
var EmbedExpansion = &manager.Command{
	Trigger:     "embed",
	Aliases:     []string{"createembed", "editembed", "embedcode"},
	Name:        "embed",
	Description: "Manage named embeds or preview and copy existing messages",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		p, err := ctx.Session.UserChannelPermissions(ctx.AuthorID(), ctx.ChanID())
		if err != nil || (p&discordgo.PermissionManageMessages) == 0 {
			return ctx.Reply("[!] You need Manage Messages permission to use this command.")
		}
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("embed")
		}
		sub := strings.ToLower(ctx.Args[0])
		gid := ctx.GuildID()
		switch sub {
		case "create":
			if len(ctx.Args) < 3 {
				return ctx.Reply("[!] Usage: `.embed create <name> <json>`")
			}
			name := ctx.Args[1]
			jsonPayload := strings.Join(ctx.Args[2:], " ")
			var payload EmbedJson
			if err := json.Unmarshal([]byte(jsonPayload), &payload); err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Invalid JSON payload: %v", err))
			}
			if err := ctx.DB.SaveEmbed(gid, name, jsonPayload, ctx.AuthorID()); err != nil {
				return ctx.Reply("[!] Failed to save embed.")
			}
			return ctx.Reply(fmt.Sprintf("[+] Saved embed as **%s**.", name))
		case "edit":
			if len(ctx.Args) < 3 {
				return ctx.Reply("[!] Usage: `.embed edit <name> <json>`")
			}
			name := ctx.Args[1]
			_, err := ctx.DB.GetEmbed(gid, name)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Saved embed **%s** not found.", name))
			}
			jsonPayload := strings.Join(ctx.Args[2:], " ")
			var payload EmbedJson
			if err := json.Unmarshal([]byte(jsonPayload), &payload); err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Invalid JSON payload: %v", err))
			}
			if err := ctx.DB.SaveEmbed(gid, name, jsonPayload, ctx.AuthorID()); err != nil {
				return ctx.Reply("[!] Failed to edit embed.")
			}
			return ctx.Reply(fmt.Sprintf("[+] Saved embed **%s** has been updated.", name))
		case "preview":
			if len(ctx.Args) < 2 {
				return ctx.Reply("[!] Usage: `.embed preview <name>`")
			}
			name := ctx.Args[1]
			saved, err := ctx.DB.GetEmbed(gid, name)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Saved embed **%s** not found.", name))
			}
			var payload EmbedJson
			_ = json.Unmarshal([]byte(saved.JSONCode), &payload)
			embed := &discordgo.MessageEmbed{
				Title:       payload.Title,
				Description: payload.Description,
				Color:       payload.Color,
			}
			if payload.Thumbnail != "" {
				embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: payload.Thumbnail}
			}
			if payload.Image != "" {
				embed.Image = &discordgo.MessageEmbedImage{URL: payload.Image}
			}
			if payload.Footer.Text != "" {
				embed.Footer = &discordgo.MessageEmbedFooter{
					Text:    payload.Footer.Text,
					IconURL: payload.Footer.Icon,
				}
			}
			for _, f := range payload.Fields {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:   f.Name,
					Value:  f.Value,
					Inline: f.Inline,
				})
			}
			return ctx.Respond(embed)
		case "list":
			list, err := ctx.DB.ListEmbeds(gid)
			if err != nil || len(list) == 0 {
				return ctx.Reply("[*] No saved embeds in this server.")
			}
			var names []string
			for _, emb := range list {
				names = append(names, fmt.Sprintf("- **%s** (created by <@%s>)", emb.Name, emb.CreatorID))
			}
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:       "Saved Embeds list",
				Description: strings.Join(names, "\n"),
			})
			return ctx.Respond(emb)
		case "delete":
			if len(ctx.Args) < 2 {
				return ctx.Reply("[!] Usage: `.embed delete <name>`")
			}
			name := ctx.Args[1]
			if err := ctx.DB.DeleteEmbed(gid, name); err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Saved embed **%s** not found.", name))
			}
			return ctx.Reply(fmt.Sprintf("[+] Deleted embed **%s**.", name))
		case "copy":
			if len(ctx.Args) < 2 {
				return ctx.Reply("[!] Usage: `.embed copy <msg_link> [name]`")
			}
			_, cid, mid := parseMessageLink(ctx.Args[1])
			if cid == "" || mid == "" {
				return ctx.Reply("[!] Invalid message link.")
			}
			msg, err := ctx.Session.ChannelMessage(cid, mid)
			if err != nil || len(msg.Embeds) == 0 {
				return ctx.Reply("[!] Message not found or contains no embeds.")
			}
			first := msg.Embeds[0]
			var payload EmbedJson
			payload.Title = first.Title
			payload.Description = first.Description
			payload.Color = first.Color
			if first.Thumbnail != nil {
				payload.Thumbnail = first.Thumbnail.URL
			}
			if first.Image != nil {
				payload.Image = first.Image.URL
			}
			if first.Footer != nil {
				payload.Footer.Text = first.Footer.Text
				payload.Footer.Icon = first.Footer.IconURL
			}
			for _, f := range first.Fields {
				payload.Fields = append(payload.Fields, struct {
					Name   string `json:"name"`
					Value  string `json:"value"`
					Inline bool   `json:"inline"`
				}{
					Name:   f.Name,
					Value:  f.Value,
					Inline: f.Inline,
				})
			}
			b, _ := json.Marshal(payload)
			jsonStr := string(b)
			if len(ctx.Args) >= 3 {
				name := ctx.Args[2]
				if err := ctx.DB.SaveEmbed(gid, name, jsonStr, ctx.AuthorID()); err != nil {
					return ctx.Reply("[!] Failed to save embed.")
				}
				return ctx.Reply(fmt.Sprintf("[+] Copied and saved embed as **%s**.", name))
			}
			return ctx.Reply(fmt.Sprintf("```json\n%s\n```", jsonStr))
		default:
			// Fallback to ad-hoc JSON posting
			targetChanID := ctx.ChanID()
			jsonArgIdx := 0

			if m := rxEmbedChan.FindStringSubmatch(ctx.Args[0]); len(m) > 1 {
				targetChanID = m[1]
				jsonArgIdx = 1
			}

			if len(ctx.Args) <= jsonArgIdx {
				return ctx.Reply("[!] Missing JSON payload.")
			}

			jsonPayload := strings.Join(ctx.Args[jsonArgIdx:], " ")
			var payload EmbedJson
			if err := json.Unmarshal([]byte(jsonPayload), &payload); err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Invalid JSON payload: %v", err))
			}

			embed := &discordgo.MessageEmbed{
				Title:       payload.Title,
				Description: payload.Description,
				Color:       payload.Color,
			}

			if payload.Thumbnail != "" {
				embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: payload.Thumbnail}
			}

			if payload.Image != "" {
				embed.Image = &discordgo.MessageEmbedImage{URL: payload.Image}
			}

			if payload.Footer.Text != "" {
				embed.Footer = &discordgo.MessageEmbedFooter{
					Text:    payload.Footer.Text,
					IconURL: payload.Footer.Icon,
				}
			}

			for _, f := range payload.Fields {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:   f.Name,
					Value:  f.Value,
					Inline: f.Inline,
				})
			}

			_, err = ctx.Session.ChannelMessageSendEmbed(targetChanID, embed)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to send embed: %v", err))
			}

			if targetChanID != ctx.ChanID() {
				return ctx.Reply("[+] Embed posted successfully.")
			}
			return nil
		}
	},
}
var CreateEmbedCmd = &manager.Command{
	Trigger:     "createembed",
	Name:        "createembed",
	Description: "Alias to `.embed create`",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) < 2 {
			return ctx.Reply("Usage: `.createembed <name> <json>`")
		}
		args := append([]string{"create"}, ctx.Args...)
		ctx.Args = args
		return EmbedExpansion.Execute(ctx)
	},
}
var EditEmbedCmd = &manager.Command{
	Trigger:     "editembed",
	Name:        "editembed",
	Description: "Edit an embed message sent by the bot",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		p, err := ctx.Session.UserChannelPermissions(ctx.AuthorID(), ctx.ChanID())
		if err != nil || (p&discordgo.PermissionManageMessages) == 0 {
			return ctx.Reply("[!] You need Manage Messages permission to use this command.")
		}
		if len(ctx.Args) < 2 {
			return ctx.Reply("Usage: `.editembed <msg_link> <json>`")
		}
		_, cid, mid := parseMessageLink(ctx.Args[0])
		if cid == "" || mid == "" {
			return ctx.Reply("[!] Invalid message link.")
		}
		msg, err := ctx.Session.ChannelMessage(cid, mid)
		if err != nil {
			return ctx.Reply("[!] Message not found.")
		}
		if msg.Author.ID != ctx.Session.State.User.ID {
			return ctx.Reply("[!] I can only edit embeds in messages that I sent.")
		}
		jsonPayload := strings.Join(ctx.Args[1:], " ")
		var payload EmbedJson
		if err := json.Unmarshal([]byte(jsonPayload), &payload); err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Invalid JSON payload: %v", err))
		}
		embed := &discordgo.MessageEmbed{
			Title:       payload.Title,
			Description: payload.Description,
			Color:       payload.Color,
		}
		if payload.Thumbnail != "" {
			embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: payload.Thumbnail}
		}
		if payload.Image != "" {
			embed.Image = &discordgo.MessageEmbedImage{URL: payload.Image}
		}
		if payload.Footer.Text != "" {
			embed.Footer = &discordgo.MessageEmbedFooter{
				Text:    payload.Footer.Text,
				IconURL: payload.Footer.Icon,
			}
		}
		for _, f := range payload.Fields {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   f.Name,
				Value:  f.Value,
				Inline: f.Inline,
			})
		}
		_, err = ctx.Session.ChannelMessageEditEmbed(cid, mid, embed)
		if err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Failed to edit embed: %v", err))
		}
		return ctx.Reply("[+] Embed edited successfully.")
	},
}
var EmbedCodeCmd = &manager.Command{
	Trigger:     "embedcode",
	Name:        "embedcode",
	Description: "Copy an existing embed's JSON code from a message link",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.Reply("Usage: `.embedcode <msg_link>`")
		}
		args := []string{"copy", ctx.Args[0]}
		ctx.Args = args
		return EmbedExpansion.Execute(ctx)
	},
}
var NamesCmd = &manager.Command{
	Trigger:     "names",
	Aliases:     []string{"namehistory"},
	Name:        "names",
	Description: "View username and nickname history of a member or yourself",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()
		target := ctx.AuthorID()
		if len(ctx.Args) > 0 {
			m, err := moderation.ResolveMember(ctx.Session, gid, ctx.Args[0])
			if err != nil || m == nil {
				return ctx.Reply("[!] Could not resolve member.")
			}
			target = m.User.ID
		}
		history, err := ctx.DB.GetMemberNameHistory(gid, target)
		if err != nil || len(history) == 0 {
			return ctx.Reply("[*] No name history found for this user.")
		}
		var sb strings.Builder
		for i, rec := range history {
			tStr := time.Unix(rec.Timestamp, 0).Format("2006-01-02 15:04")
			sb.WriteString(fmt.Sprintf("%d. `%s` -> `%s` (at %s)\n", i+1, rec.Old, rec.New, tStr))
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "Member Name History",
			Description: sb.String(),
		})
		return ctx.Respond(emb)
	},
}
var ClearNamesCmd = &manager.Command{
	Trigger:     "clearnames",
	Name:        "clearnames",
	Description: "Reset your name history",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()
		uid := ctx.AuthorID()
		if err := ctx.DB.ClearMemberNameHistory(gid, uid); err != nil {
			return ctx.Reply("[!] Failed to clear name history.")
		}
		return ctx.Reply("[+] Cleared your name history successfully.")
	},
}
var GNamesCmd = &manager.Command{
	Trigger:     "gnames",
	Aliases:     []string{"guildnames"},
	Name:        "gnames",
	Description: "View guild name changes history",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()
		history, err := ctx.DB.GetGuildNameHistory(gid)
		if err != nil || len(history) == 0 {
			return ctx.Reply("[*] No server name history found.")
		}
		var sb strings.Builder
		for i, rec := range history {
			tStr := time.Unix(rec.Timestamp, 0).Format("2006-01-02 15:04")
			sb.WriteString(fmt.Sprintf("%d. **%s** -> **%s** (at %s)\n", i+1, rec.Old, rec.New, tStr))
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "Server Name History",
			Description: sb.String(),
		})
		return ctx.Respond(emb)
	},
}
var InvitesCmd = &manager.Command{
	Trigger:     "invites",
	Name:        "invites",
	Description: "View all active server invites",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		p, err := ctx.Session.UserChannelPermissions(ctx.AuthorID(), ctx.ChanID())
		if err != nil || (p&discordgo.PermissionManageGuild) == 0 {
			return ctx.Reply("[!] You need Manage Guild permission to use this command.")
		}
		invites, err := ctx.Session.GuildInvites(ctx.GuildID())
		if err != nil || len(invites) == 0 {
			return ctx.Reply("[*] No active invites found in this server.")
		}
		var sb strings.Builder
		for i, inv := range invites {
			usesStr := fmt.Sprintf("%d uses", inv.Uses)
			if inv.MaxUses > 0 {
				usesStr = fmt.Sprintf("%d/%d uses", inv.Uses, inv.MaxUses)
			}
			expStr := "Never"
			if inv.MaxAge > 0 {
				expTime := inv.CreatedAt.Add(time.Duration(inv.MaxAge) * time.Second)
				expStr = fmt.Sprintf("<t:%d:R>", expTime.Unix())
			}
			creator := "Unknown"
			if inv.Inviter != nil {
				creator = inv.Inviter.Username
			}
			sb.WriteString(fmt.Sprintf("%d. **code:** `%s` | **by:** `%s` | **uses:** `%s` | **expires:** %s\n",
				i+1, inv.Code, creator, usesStr, expStr))
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "Active Invites List",
			Description: sb.String(),
		})
		return ctx.Respond(emb)
	},
}
var TopCommandsCmd = &manager.Command{
	Trigger:     "topcommands",
	Aliases:     []string{"topcmds"},
	Name:        "topcommands",
	Description: "View the most used commands",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		stats, err := ctx.DB.GetTopCommands()
		if err != nil || len(stats) == 0 {
			return ctx.Reply("[*] No command usage stats logged yet.")
		}
		sort.Slice(stats, func(i, j int) bool {
			return stats[i].Count > stats[j].Count
		})
		var sb strings.Builder
		limit := len(stats)
		if limit > 10 {
			limit = 10
		}
		for i := 0; i < limit; i++ {
			sb.WriteString(fmt.Sprintf("%d. `.%s` — **%d** executions\n", i+1, stats[i].Trigger, stats[i].Count))
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "Top Commands Leaderboard",
			Description: sb.String(),
		})
		return ctx.Respond(emb)
	},
}