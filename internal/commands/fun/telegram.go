package fun
import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/spoofer"
	"strings"
	"time"
)
func init() {
	manager.RegisterHelp("telegram", []manager.HelpPage{
		{
			Command:     "Telegram Lookup",
			Syntax:      ".telegram <username>",
			Description: "Fetch public profile details for a Telegram user, channel, or group.",
		},
	})
}
var Telegram = &manager.Command{
	Trigger:     "telegram",
	Aliases:     []string{"tg"},
	Name:        "telegram",
	Description: "Gets profile information on the given Telegram user or group",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("telegram")
		}
		username := ctx.Args[0]
		username = strings.TrimPrefix(username, "@")
		url := fmt.Sprintf("https://t.me/%s", username)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", spoofer.GetRandomUA())
		client := &http.Client{Timeout: 6 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return ctx.Reply("[!] Telegram server is unreachable.")
		}
		defer resp.Body.Close()
		if resp.StatusCode == 404 {
			return ctx.Reply(fmt.Sprintf("[!] Telegram user/group `%s` not found.", username))
		}
		body, _ := io.ReadAll(resp.Body)
		html := string(body)
		rxTitle := regexp.MustCompile(`<meta property="og:title" content="([^"]+)">`)
		rxDesc := regexp.MustCompile(`<meta property="og:description" content="([^"]+)">`)
		rxImage := regexp.MustCompile(`<meta property="og:image" content="([^"]+)">`)
		titleMatch := rxTitle.FindStringSubmatch(html)
		descMatch := rxDesc.FindStringSubmatch(html)
		imgMatch := rxImage.FindStringSubmatch(html)
		title := username
		if len(titleMatch) > 1 {
			title = titleMatch[1]
		}
		desc := "No description available."
		if len(descMatch) > 1 {
			desc = descMatch[1]
			desc = strings.ReplaceAll(desc, "&amp;", "&")
			desc = strings.ReplaceAll(desc, "&quot;", "\"")
			desc = strings.ReplaceAll(desc, "&lt;", "<")
			desc = strings.ReplaceAll(desc, "&gt;", ">")
		}
		var avatar string
		if len(imgMatch) > 1 {
			avatar = imgMatch[1]
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:        title,
			Description:  desc,
			ThumbnailURL: avatar,
		})
		emb.URL = url
		return ctx.Respond(emb)
	},
}