package general
import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"sort"
	"strings"
	"github.com/bwmarrin/discordgo"
)
func init() {
	manager.RegisterHelp("emoji", []manager.HelpPage{
		{
			Command:     "Emoji View",
			Syntax:      ".emoji <emoji>",
			Description: "Displays a large version of an emoji.",
		},
		{
			Command:     "Emoji Add",
			Syntax:      ".emoji add <emoji> [name]",
			Description: "Adds a custom emoji to the server.",
		},
		{
			Command:     "Emoji Remove",
			Syntax:      ".emoji remove <emoji>",
			Description: "Removes a custom emoji from the server.",
		},
		{
			Command:     "Emoji AddMany",
			Syntax:      ".emoji addmany <emotes...>",
			Description: "Bulk adds custom emojis to the server.",
		},
		{
			Command:     "Emoji RemoveMany",
			Syntax:      ".emoji removemany <emotes...>",
			Description: "Bulk removes custom emojis from the server.",
		},
		{
			Command:     "Emoji RemoveDuplicates",
			Syntax:      ".emoji removeduplicates",
			Description: "Removes duplicate custom emojis with the same name.",
		},
		{
			Command:     "Emoji Rename",
			Syntax:      ".emoji rename <emoji> <new_name>",
			Description: "Renames a custom emoji.",
		},
		{
			Command:     "Emoji Stats",
			Syntax:      ".emoji stats",
			Description: "View the top 10 most used emotes in this server.",
		},
		{
			Command:     "Emoji Info",
			Syntax:      ".emoji information <message_link>",
			Description: "View information about the most recent emoji used in a message.",
		},
	})
}
var Emoji = &manager.Command{
	Trigger:     "emoji",
	Aliases:     []string{"emote"},
	Name:        "emoji",
	Description: "Returns large emoji, adds, removes, or manages server emotes",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("emoji")
		}
		sub := strings.ToLower(ctx.Args[0])
		rxEmoji := regexp.MustCompile(`<(a)?:([a-zA-Z0-9_]+):([0-9]+)>`)
		switch sub {
		case "add":
			if len(ctx.Args) < 2 {
				return ctx.Reply("Usage: .emoji add <emoji> [name]")
			}
			raw := ctx.Args[1]
			matches := rxEmoji.FindStringSubmatch(raw)
			if len(matches) < 4 {
				return ctx.Reply("[!] Invalid custom emoji format.")
			}
			isAnimated := matches[1] == "a"
			origName := matches[2]
			id := matches[3]
			name := origName
			if len(ctx.Args) >= 3 {
				name = ctx.Args[2]
			}
			ext := "png"
			if isAnimated {
				ext = "gif"
			}
			url := fmt.Sprintf("https://cdn.discordapp.com/emojis/%s.%s", id, ext)
			resp, err := http.Get(url)
			if err != nil {
				return ctx.Reply("[!] Failed to download emoji image.")
			}
			defer resp.Body.Close()
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return ctx.Reply("[!] Failed to read emoji data.")
			}
			contentType := http.DetectContentType(b)
			b64 := base64.StdEncoding.EncodeToString(b)
			dataURI := fmt.Sprintf("data:%s;base64,%s", contentType, b64)
			newEmoji, err := ctx.Session.GuildEmojiCreate(ctx.GuildID(), &discordgo.EmojiParams{
				Name:  name,
				Image: dataURI,
			})
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to upload emoji: %v", err))
			}
			return ctx.Reply(fmt.Sprintf("[*] Added custom emoji %s as **:%s:**", newEmoji.MessageFormat(), newEmoji.Name))
		case "remove":
			if !checkPerm(ctx, discordgo.PermissionManageEmojis) {
				return ctx.Reply("[!] Missing Manage Expressions permission.")
			}
			if len(ctx.Args) < 2 {
				return ctx.Reply("Usage: .emoji remove <emoji>")
			}
			raw := ctx.Args[1]
			matches := rxEmoji.FindStringSubmatch(raw)
			var id string
			if len(matches) >= 4 {
				id = matches[3]
			} else {
				id = raw
			}
			err := ctx.Session.GuildEmojiDelete(ctx.GuildID(), id)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to remove emoji: %v", err))
			}
			return ctx.Reply("[*] Emote successfully removed.")
		case "addmany":
			if len(ctx.Args) < 2 {
				return ctx.Reply("Usage: .emoji addmany <emotes...>")
			}
			content := strings.Join(ctx.Args[1:], " ")
			matches := rxEmoji.FindAllStringSubmatch(content, -1)
			if len(matches) == 0 {
				return ctx.Reply("[!] No custom emojis found to add.")
			}
			successCount := 0
			for _, m := range matches {
				isAnimated := m[1] == "a"
				name := m[2]
				id := m[3]
				ext := "png"
				if isAnimated {
					ext = "gif"
				}
				url := fmt.Sprintf("https://cdn.discordapp.com/emojis/%s.%s", id, ext)
				resp, err := http.Get(url)
				if err != nil {
					continue
				}
				b, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					continue
				}
				contentType := http.DetectContentType(b)
				b64 := base64.StdEncoding.EncodeToString(b)
				dataURI := fmt.Sprintf("data:%s;base64,%s", contentType, b64)
				_, err = ctx.Session.GuildEmojiCreate(ctx.GuildID(), &discordgo.EmojiParams{
					Name:  name,
					Image: dataURI,
				})
				if err == nil {
					successCount++
				}
			}
			return ctx.Reply(fmt.Sprintf("[*] Bulk added %d emojis.", successCount))
		case "removemany":
			if !checkPerm(ctx, discordgo.PermissionManageEmojis) {
				return ctx.Reply("[!] Missing Manage Expressions permission.")
			}
			if len(ctx.Args) < 2 {
				return ctx.Reply("Usage: .emoji removemany <emotes...>")
			}
			content := strings.Join(ctx.Args[1:], " ")
			matches := rxEmoji.FindAllStringSubmatch(content, -1)
			if len(matches) == 0 {
				return ctx.Reply("[!] No custom emojis found to remove.")
			}
			successCount := 0
			for _, m := range matches {
				id := m[3]
				err := ctx.Session.GuildEmojiDelete(ctx.GuildID(), id)
				if err == nil {
					successCount++
				}
			}
			return ctx.Reply(fmt.Sprintf("[*] Bulk removed %d emojis.", successCount))
		case "removeduplicates":
			if !checkPerm(ctx, discordgo.PermissionManageEmojis) {
				return ctx.Reply("[!] Missing Manage Expressions permission.")
			}
			emojis, err := ctx.Session.GuildEmojis(ctx.GuildID())
			if err != nil {
				return ctx.Reply("[!] Failed to fetch guild emojis.")
			}
			seen := make(map[string]bool)
			deletedCount := 0
			for _, e := range emojis {
				lowerName := strings.ToLower(e.Name)
				if seen[lowerName] {
					err = ctx.Session.GuildEmojiDelete(ctx.GuildID(), e.ID)
					if err == nil {
						deletedCount++
					}
				} else {
					seen[lowerName] = true
				}
			}
			return ctx.Reply(fmt.Sprintf("[*] Pruned %d duplicate emojis from the server.", deletedCount))
		case "rename":
			if !checkPerm(ctx, discordgo.PermissionManageEmojis) {
				return ctx.Reply("[!] Missing Manage Expressions permission.")
			}
			if len(ctx.Args) < 3 {
				return ctx.Reply("Usage: .emoji rename <emoji> <new_name>")
			}
			raw := ctx.Args[1]
			newName := ctx.Args[2]
			matches := rxEmoji.FindStringSubmatch(raw)
			var id string
			if len(matches) >= 4 {
				id = matches[3]
			} else {
				id = raw
			}
			em, err := ctx.Session.GuildEmojiEdit(ctx.GuildID(), id, &discordgo.EmojiParams{
				Name: newName,
			})
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to rename emoji: %v", err))
			}
			return ctx.Reply(fmt.Sprintf("[*] Successfully renamed emoji to **:%s:**", em.Name))
		case "stats":
			if !checkPerm(ctx, discordgo.PermissionManageEmojis) {
				return ctx.Reply("[!] Missing Manage Expressions permission.")
			}
			stats, err := ctx.DB.GetTopEmojis(ctx.GuildID())
			if err != nil || len(stats) == 0 {
				return ctx.Reply("[*] No custom emoji statistics found.")
			}
			emojis, err := ctx.Session.GuildEmojis(ctx.GuildID())
			if err != nil {
				return ctx.Reply("[!] Failed to fetch guild emojis.")
			}
			type emStat struct {
				Emoji *discordgo.Emoji
				Count int
			}
			var list []emStat
			for _, em := range emojis {
				if count, ok := stats[em.ID]; ok {
					list = append(list, emStat{Emoji: em, Count: count})
				}
			}
			sort.Slice(list, func(i, j int) bool {
				return list[i].Count > list[j].Count
			})
			if len(list) > 10 {
				list = list[:10]
			}
			var sb strings.Builder
			sb.WriteString("**Top 10 Most Used Emotes:**\n\n")
			for idx, item := range list {
				sb.WriteString(fmt.Sprintf("%d. %s (`:%s:`): **%d** times\n", idx+1, item.Emoji.MessageFormat(), item.Emoji.Name, item.Count))
			}
			return ctx.Reply(sb.String())
		case "information":
			if len(ctx.Args) < 2 {
				return ctx.Reply("Usage: .emoji information <message_link>")
			}
			link := ctx.Args[1]
			parts := strings.Split(link, "/")
			if len(parts) < 7 {
				return ctx.Reply("[!] Invalid message link.")
			}
			chanID := parts[len(parts)-2]
			msgID := parts[len(parts)-1]
			msg, err := ctx.Session.ChannelMessage(chanID, msgID)
			if err != nil {
				return ctx.Reply("[!] Failed to fetch message from link.")
			}
			allMatches := rxEmoji.FindAllStringSubmatch(msg.Content, -1)
			if len(allMatches) == 0 {
				return ctx.Reply("[!] No custom emojis found in the specified message.")
			}
			lastMatch := allMatches[len(allMatches)-1]
			isAnimated := lastMatch[1] == "a"
			name := lastMatch[2]
			id := lastMatch[3]
			ext := "png"
			if isAnimated {
				ext = "gif"
			}
			url := fmt.Sprintf("https://cdn.discordapp.com/emojis/%s.%s", id, ext)
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:       "Emoji Information",
				Description: fmt.Sprintf("**Name:** `%s`\n**ID:** `%s`\n**Animated:** `%t`\n\n[Download link](%s)", name, id, isAnimated, url),
				ThumbnailURL: url,
			})
			return ctx.Respond(emb)
		default:
			raw := ctx.Args[0]
			matches := rxEmoji.FindStringSubmatch(raw)
			if len(matches) < 4 {
				return ctx.Reply("[!] Input must be a custom guild emoji.")
			}
			isAnimated := matches[1] == "a"
			name := matches[2]
			id := matches[3]
			ext := "png"
			if isAnimated {
				ext = "gif"
			}
			url := fmt.Sprintf("https://cdn.discordapp.com/emojis/%s.%s", id, ext)
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:    fmt.Sprintf("Emoji: :%s:", name),
				ImageURL: url,
			})
			return ctx.Respond(emb)
		}
	},
}