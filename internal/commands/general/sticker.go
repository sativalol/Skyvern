package general
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"skyvern/internal/manager"
	"strings"
	"github.com/bwmarrin/discordgo"
)
func init() {
	manager.RegisterHelp("sticker", []manager.HelpPage{
		{
			Command:     "Sticker Help",
			Syntax:      ".sticker",
			Description: "View sticker help options.",
		},
		{
			Command:     "Sticker Add",
			Syntax:      ".sticker add <url> <name>",
			Description: "Downloads a sticker from URL and adds it to the server.",
		},
		{
			Command:     "Sticker Remove",
			Syntax:      ".sticker remove <name/id>",
			Description: "Removes a sticker from the server.",
		},
		{
			Command:     "Sticker Rename",
			Syntax:      ".sticker rename <new_name>",
			Description: "Rename the attached/referenced sticker to new name.",
		},
		{
			Command:     "Sticker Tag",
			Syntax:      ".sticker tag",
			Description: "Add server vanity to stickers tag.",
		},
		{
			Command:     "Sticker Cleanup",
			Syntax:      ".sticker cleanup",
			Description: "Clean server sticker names.",
		},
	})
}
func listStickers(s *discordgo.Session, gid string) ([]*discordgo.Sticker, error) {
	u := "https://discord.com/api/v10/guilds/" + gid + "/stickers"
	resp, err := s.Request("GET", u, nil)
	if err != nil {
		return nil, err
	}
	var res []*discordgo.Sticker
	err = json.Unmarshal(resp, &res)
	return res, err
}
func deleteSticker(s *discordgo.Session, gid, sid string) error {
	u := "https://discord.com/api/v10/guilds/" + gid + "/stickers/" + sid
	_, err := s.Request("DELETE", u, nil)
	return err
}
func editSticker(s *discordgo.Session, gid, sid string, name string) (*discordgo.Sticker, error) {
	u := "https://discord.com/api/v10/guilds/" + gid + "/stickers/" + sid
	payload := map[string]string{
		"name": name,
	}
	resp, err := s.Request("PATCH", u, payload)
	if err != nil {
		return nil, err
	}
	var res discordgo.Sticker
	err = json.Unmarshal(resp, &res)
	return &res, err
}
func createSticker(s *discordgo.Session, gid string, name string, fileBytes []byte, ext string, contentType string) (*discordgo.Sticker, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("name", name)
	_ = w.WriteField("description", "Added via Skyvern")
	_ = w.WriteField("tags", "skyvern")
	fw, err := w.CreateFormFile("file", name+"."+ext)
	if err != nil {
		return nil, err
	}
	_, _ = fw.Write(fileBytes)
	w.Close()
	req, err := http.NewRequest("POST", "https://discord.com/api/v10/guilds/"+gid+"/stickers", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", s.Token)
	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discord status %d: %s", resp.StatusCode, string(b))
	}
	b, _ := io.ReadAll(resp.Body)
	var res discordgo.Sticker
	err = json.Unmarshal(b, &res)
	return &res, err
}
var Sticker = &manager.Command{
	Trigger:     "sticker",
	Name:        "sticker",
	Description: "Modify or add stickers to your server",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("sticker")
		}
		sub := strings.ToLower(ctx.Args[0])
		switch sub {
		case "add":
			if !checkPerm(ctx, discordgo.PermissionManageEmojis) {
				return ctx.Reply("[!] Missing Manage Expressions permission.")
			}
			if len(ctx.Args) < 3 {
				return ctx.Reply("Usage: .sticker add <url> <name>")
			}
			url := ctx.Args[1]
			name := strings.Join(ctx.Args[2:], " ")
			resp, err := http.Get(url)
			if err != nil {
				return ctx.Reply("[!] Failed to download sticker from URL.")
			}
			defer resp.Body.Close()
			var buf bytes.Buffer
			_, _ = io.Copy(&buf, resp.Body)
			contentType := resp.Header.Get("Content-Type")
			ext := "png"
			if strings.Contains(contentType, "gif") {
				ext = "gif"
			} else if strings.Contains(contentType, "json") {
				ext = "json"
			}
			st, err := createSticker(ctx.Session, ctx.GuildID(), name, buf.Bytes(), ext, contentType)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to upload sticker: %v", err))
			}
			return ctx.Reply(fmt.Sprintf("[*] Successfully added sticker **%s** (ID: `%s`).", st.Name, st.ID))
		case "remove":
			if !checkPerm(ctx, discordgo.PermissionManageEmojis) {
				return ctx.Reply("[!] Missing Manage Expressions permission.")
			}
			if len(ctx.Args) < 2 {
				return ctx.Reply("Usage: .sticker remove <name/id>")
			}
			query := strings.Join(ctx.Args[1:], " ")
			stickers, err := listStickers(ctx.Session, ctx.GuildID())
			if err != nil {
				return ctx.Reply("[!] Failed to fetch guild stickers.")
			}
			var target *discordgo.Sticker
			for _, st := range stickers {
				if st.ID == query || strings.EqualFold(st.Name, query) {
					target = st
					break
				}
			}
			if target == nil {
				return ctx.Reply("[!] Sticker not found.")
			}
			err = deleteSticker(ctx.Session, ctx.GuildID(), target.ID)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to delete sticker: %v", err))
			}
			return ctx.Reply(fmt.Sprintf("[*] Successfully removed sticker **%s**.", target.Name))
		case "rename":
			if !checkPerm(ctx, discordgo.PermissionManageEmojis) {
				return ctx.Reply("[!] Missing Manage Expressions permission.")
			}
			if len(ctx.Args) < 2 {
				return ctx.Reply("Usage: .sticker rename <new_name>")
			}
			newName := strings.Join(ctx.Args[1:], " ")
			var stickerID string
			if ctx.Message != nil && ctx.Message.ReferencedMessage != nil && len(ctx.Message.ReferencedMessage.StickerItems) > 0 {
				stickerID = ctx.Message.ReferencedMessage.StickerItems[0].ID
			}
			if stickerID == "" {
				return ctx.Reply("[!] You must reply to a message containing a sticker to rename it.")
			}
			st, err := editSticker(ctx.Session, ctx.GuildID(), stickerID, newName)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to rename sticker: %v", err))
			}
			return ctx.Reply(fmt.Sprintf("[*] Successfully renamed sticker to **%s**.", st.Name))
		case "tag":
			if !checkPerm(ctx, discordgo.PermissionManageEmojis) || !checkPerm(ctx, discordgo.PermissionManageGuild) {
				return ctx.Reply("[!] Missing Manage Expressions or Manage Guild permissions.")
			}
			g, err := ctx.Session.State.Guild(ctx.GuildID())
			if err != nil {
				g, err = ctx.Session.Guild(ctx.GuildID())
			}
			if err != nil {
				return ctx.Reply("[!] Failed to fetch guild details.")
			}
			vanity := g.VanityURLCode
			if vanity == "" {
				vanity = "vanity"
			}
			stickers, err := listStickers(ctx.Session, ctx.GuildID())
			if err != nil {
				return ctx.Reply("[!] Failed to fetch guild stickers.")
			}
			count := 0
			for _, st := range stickers {
				u := "https://discord.com/api/v10/guilds/" + ctx.GuildID() + "/stickers/" + st.ID
				payload := map[string]string{
					"tags": vanity,
				}
				_, err = ctx.Session.Request("PATCH", u, payload)
				if err == nil {
					count++
				}
			}
			return ctx.Reply(fmt.Sprintf("[*] Tagged %d stickers with server vanity: **%s**.", count, vanity))
		case "cleanup":
			if !checkPerm(ctx, discordgo.PermissionManageEmojis) || !checkPerm(ctx, discordgo.PermissionManageGuild) {
				return ctx.Reply("[!] Missing Manage Expressions or Manage Guild permissions.")
			}
			stickers, err := listStickers(ctx.Session, ctx.GuildID())
			if err != nil {
				return ctx.Reply("[!] Failed to fetch guild stickers.")
			}
			count := 0
			for _, st := range stickers {
				cleanName := strings.ReplaceAll(st.Name, "-", "")
				cleanName = strings.ReplaceAll(cleanName, "_", "")
				cleanName = strings.ReplaceAll(cleanName, " ", "")
				if cleanName != st.Name && cleanName != "" {
					_, err = editSticker(ctx.Session, ctx.GuildID(), st.ID, cleanName)
					if err == nil {
						count++
					}
				}
			}
			return ctx.Reply(fmt.Sprintf("[*] Cleaned up names for %d stickers.", count))
		default:
			return ctx.SendHelp("sticker")
		}
	},
}