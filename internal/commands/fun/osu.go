package fun
import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/spoofer"
	"strings"
	"time"
	"github.com/bwmarrin/discordgo"
)
func init() {
	manager.RegisterHelp("osu", []manager.HelpPage{
		{
			Command:     "osu! Profile Lookup",
			Syntax:      ".osu <username> [mode]",
			Description: "Retrieves osu! profile statistics. Modes: osu, taiko, fruits, mania",
		},
	})
}
var Osu = &manager.Command{
	Trigger:     "osu",
	Name:        "osu",
	Description: "Retrieve simple OSU! profile information",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("osu")
		}
		user := ctx.Args[0]
		mode := "osu"
		if len(ctx.Args) > 1 {
			m := strings.ToLower(ctx.Args[1])
			if m == "taiko" || m == "fruits" || m == "catch" || m == "mania" {
				if m == "catch" {
					mode = "fruits"
				} else {
					mode = m
				}
			}
		}
		url := fmt.Sprintf("https://osu.ppy.sh/users/%s/%s", user, mode)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", spoofer.GetRandomUA())
		client := &http.Client{Timeout: 8 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return ctx.Reply("[!] osu! website is unreachable.")
		}
		defer resp.Body.Close()
		if resp.StatusCode == 404 {
			return ctx.Reply(fmt.Sprintf("[!] osu! user `%s` not found.", user))
		}
		body, _ := io.ReadAll(resp.Body)
		html := string(body)
		rxJSON := regexp.MustCompile(`(?s)<script\s+id="json-user"\s+type="application/json">\s*(.*?)\s*</script>`)
		match := rxJSON.FindStringSubmatch(html)
		if len(match) < 2 {
			return ctx.Reply("[!] Failed to parse profile data from page.")
		}
		var data struct {
			Username      string `json:"username"`
			ID            int    `json:"id"`
			AvatarURL     string `json:"avatar_url"`
			CountryCode   string `json:"country_code"`
			IsActive      bool   `json:"is_active"`
			Cover         struct {
				URL string `json:"url"`
			} `json:"cover"`
			Statistics struct {
				GlobalRank int     `json:"global_rank"`
				CountryRank int    `json:"country_rank"`
				PP          float64 `json:"pp"`
				HitAccuracy float64 `json:"hit_accuracy"`
				PlayCount   int     `json:"play_count"`
				Level       struct {
					Current int `json:"current"`
				} `json:"level"`
				GradeCounts struct {
					SS  int `json:"ss"`
					S   int `json:"s"`
					A   int `json:"a"`
				} `json:"grade_counts"`
			} `json:"statistics"`
		}
		if err := json.Unmarshal([]byte(match[1]), &data); err != nil {
			return ctx.Reply("[!] Error decoding profile payload.")
		}
		fields := []*discordgo.MessageEmbedField{
			config.Field("Global Rank", fmt.Sprintf("#%d", data.Statistics.GlobalRank), true),
			config.Field("Country Rank", fmt.Sprintf("#%d (%s)", data.Statistics.CountryRank, data.CountryCode), true),
			config.Field("PP", fmt.Sprintf("%.2f", data.Statistics.PP), true),
			config.Field("Accuracy", fmt.Sprintf("%.2f%%", data.Statistics.HitAccuracy), true),
			config.Field("Play Count", fmt.Sprintf("%d", data.Statistics.PlayCount), true),
			config.Field("Level", fmt.Sprintf("%d", data.Statistics.Level.Current), true),
			config.Field("Grades", fmt.Sprintf("SS: **%d** | S: **%d** | A: **%d**", data.Statistics.GradeCounts.SS, data.Statistics.GradeCounts.S, data.Statistics.GradeCounts.A), false),
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:        fmt.Sprintf("osu! Profile - %s", data.Username),
			ThumbnailURL: data.AvatarURL,
			ImageURL:     data.Cover.URL,
			Fields:       fields,
		})
		emb.URL = fmt.Sprintf("https://osu.ppy.sh/users/%d", data.ID)
		return ctx.Respond(emb)
	},
}