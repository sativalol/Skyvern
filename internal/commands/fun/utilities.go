package fun
import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"github.com/bwmarrin/discordgo"
	"skyvern/internal/config"
	"skyvern/internal/manager"
)
func init() {
	manager.RegisterHelp("randomip", []manager.HelpPage{
		{
			Command:     "Random IP Generator",
			Syntax:      ".randomip",
			Description: "Generates a random valid IPv4 address.",
		},
	})
	manager.RegisterHelp("duckduckgo", []manager.HelpPage{
		{
			Command:     "DuckDuckGo Search",
			Syntax:      ".duckduckgo <query>",
			Description: "Search DuckDuckGo with instant results.",
		},
	})
	manager.RegisterHelp("ocr", []manager.HelpPage{
		{
			Command:     "Optical Character Recognition",
			Syntax:      ".ocr [reply to image message]",
			Description: "Extracts text from an image attachment.",
		},
	})
	manager.RegisterHelp("ocrtr", []manager.HelpPage{
		{
			Command:     "OCR & Translate",
			Syntax:      ".ocrtr <target_lang> [reply to image message]",
			Description: "Extracts text from an image and translates it.",
		},
	})
	manager.RegisterHelp("palette", []manager.HelpPage{
		{
			Command:     "Color Palette Extractor",
			Syntax:      ".palette [reply to image message]",
			Description: "Extracts the dominant colors from an image.",
		},
	})
	manager.RegisterHelp("steal", []manager.HelpPage{
		{
			Command:     "Steal Emoji",
			Syntax:      ".steal <emoji> [name]",
			Description: "Adds a custom emoji from another server to this server.",
		},
	})
	manager.RegisterHelp("weather", []manager.HelpPage{
		{
			Command:     "Weather Search",
			Syntax:      ".weather <location>",
			Description: "Get weather information for any location.",
		},
	})
}
var RandomIP = &manager.Command{
	Trigger:     "randomip",
	Aliases:     []string{"rip", "genip"},
	Name:        "randomip",
	Description: "Generates a random IPv4 address",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		ip := fmt.Sprintf("%d.%d.%d.%d", rand.Intn(223)+1, rand.Intn(256), rand.Intn(256), rand.Intn(256))
		return ctx.Reply(fmt.Sprintf("[+] Generated IP: `%s`", ip))
	},
}
var DuckDuckGo = &manager.Command{
	Trigger:     "duckduckgo",
	Aliases:     []string{"ddg"},
	Name:        "duckduckgo",
	Description: "Search DuckDuckGo with instant results",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("duckduckgo")
		}
		query := strings.Join(ctx.Args, " ")
		apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1", url.QueryEscape(query))
		res, err := http.Get(apiURL)
		if err != nil {
			return ctx.Reply("[!] DDG service offline.")
		}
		defer res.Body.Close()
		var data struct {
			AbstractText string `json:"AbstractText"`
			AbstractURL  string `json:"AbstractURL"`
			Heading      string `json:"Heading"`
		}
		_ = json.NewDecoder(res.Body).Decode(&data)
		if data.AbstractText == "" {
			return ctx.Reply(fmt.Sprintf("[*] No instant answer found. Try direct search: https://duckduckgo.com/?q=%s", url.QueryEscape(query)))
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       data.Heading,
			Description: data.AbstractText,
		})
		emb.URL = data.AbstractURL
		return ctx.Respond(emb)
	},
}
func getImgURL(ctx *manager.CommandContext) string {
	if ctx.Message == nil {
		return ""
	}
	if len(ctx.Message.Attachments) > 0 {
		return ctx.Message.Attachments[0].URL
	}
	if ctx.Message.ReferencedMessage != nil && len(ctx.Message.ReferencedMessage.Attachments) > 0 {
		return ctx.Message.ReferencedMessage.Attachments[0].URL
	}
	return ""
}
func doOCR(imgURL string) (string, error) {
	apiURL := fmt.Sprintf("https://api.ocr.space/parse/imageurl?apikey=helloworld&url=%s", url.QueryEscape(imgURL))
	res, err := http.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	var data struct {
		ParsedResults []struct {
			ParsedText string `json:"ParsedText"`
		} `json:"ParsedResults"`
		ErrorMessage []string `json:"ErrorMessage"`
	}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return "", err
	}
	if len(data.ParsedResults) > 0 {
		return data.ParsedResults[0].ParsedText, nil
	}
	if len(data.ErrorMessage) > 0 {
		return "", fmt.Errorf("%s", data.ErrorMessage[0])
	}
	return "", fmt.Errorf("OCR parse failed")
}
var OCR = &manager.Command{
	Trigger:     "ocr",
	Name:        "ocr",
	Description: "Extracts text from an image attachment",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		imgURL := getImgURL(ctx)
		if imgURL == "" {
			return ctx.Reply("[!] Please attach or reply to a message containing an image.")
		}
		_ = ctx.Reply("[*] Parsing image, please wait...")
		txt, err := doOCR(imgURL)
		if err != nil || strings.TrimSpace(txt) == "" {
			return ctx.Reply("[!] No readable text found or OCR space limit hit.")
		}
		if len(txt) > 2000 {
			txt = txt[:1997] + "..."
		}
		return ctx.Reply(fmt.Sprintf("```\n%s\n```", txt))
	},
}
var OCRTR = &manager.Command{
	Trigger:     "ocrtr",
	Name:        "ocrtr",
	Description: "Extracts text from an image and translates it",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("ocrtr")
		}
		lang := ctx.Args[0]
		imgURL := getImgURL(ctx)
		if imgURL == "" {
			return ctx.Reply("[!] Please attach or reply to a message containing an image.")
		}
		_ = ctx.Reply("[*] Parsing and translating image...")
		txt, err := doOCR(imgURL)
		if err != nil || strings.TrimSpace(txt) == "" {
			return ctx.Reply("[!] No readable text found in the image.")
		}
		apiURL := fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=auto&tl=%s&dt=t&q=%s", lang, url.QueryEscape(txt))
		res, err := http.Get(apiURL)
		if err != nil {
			return ctx.Reply(fmt.Sprintf("[!] OCR succeeded, but translation failed.\nText:\n```\n%s\n```", txt))
		}
		defer res.Body.Close()
		var data []interface{}
		_ = json.NewDecoder(res.Body).Decode(&data)
		outer, ok := data[0].([]interface{})
		if !ok || len(outer) == 0 {
			return ctx.Reply(fmt.Sprintf("[!] OCR text:\n```\n%s\n```", txt))
		}
		var parts []string
		for _, item := range outer {
			inner, ok := item.([]interface{})
			if ok && len(inner) > 0 {
				if str, ok := inner[0].(string); ok {
					parts = append(parts, str)
				}
			}
		}
		translated := strings.Join(parts, "")
		if len(translated) > 1900 {
			translated = translated[:1897] + "..."
		}
		return ctx.Reply(fmt.Sprintf("**OCR Translation (%s):**\n```\n%s\n```", lang, translated))
	},
}
var Palette = &manager.Command{
	Trigger:     "palette",
	Aliases:     []string{"colors"},
	Name:        "palette",
	Description: "Extract dominant color palette from image",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		imgURL := getImgURL(ctx)
		if imgURL == "" {
			return ctx.Reply("[!] Please attach or reply to a message containing an image.")
		}
		resp, err := http.Get(imgURL)
		if err != nil {
			return ctx.Reply("[!] Failed to download image.")
		}
		defer resp.Body.Close()
		img, _, err := image.Decode(resp.Body)
		if err != nil {
			return ctx.Reply("[!] Invalid image format. Must be PNG or JPEG.")
		}
		bounds := img.Bounds()
		counts := make(map[string]int)
		for i := 0; i < 1000; i++ {
			x := rand.Intn(bounds.Max.X-bounds.Min.X) + bounds.Min.X
			y := rand.Intn(bounds.Max.Y-bounds.Min.Y) + bounds.Min.Y
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()
			rVal := r >> 8
			gVal := g >> 8
			bVal := b >> 8
			rRounded := (rVal / 32) * 32
			gRounded := (gVal / 32) * 32
			bRounded := (bVal / 32) * 32
			hex := fmt.Sprintf("#%02x%02x%02x", rRounded, gRounded, bRounded)
			counts[hex]++
		}
		type colorEntry struct {
			hex   string
			count int
		}
		var sorted []colorEntry
		for h, c := range counts {
			sorted = append(sorted, colorEntry{hex: h, count: c})
		}
		for i := 0; i < len(sorted); i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[i].count < sorted[j].count {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
		limit := 5
		if len(sorted) < limit {
			limit = len(sorted)
		}
		var hexes []string
		for i := 0; i < limit; i++ {
			hexes = append(hexes, sorted[i].hex)
		}
		colorBlocks := ""
		for _, h := range hexes {
			colorBlocks += fmt.Sprintf("`%s` \n", h)
		}
		chartCfg := fmt.Sprintf(`{
			type: 'bar',
			data: {
				labels: %s,
				datasets: [{
					data: [1, 1, 1, 1, 1],
					backgroundColor: %s
				}]
			},
			options: {
				legend: { display: false },
				scales: {
					yAxes: [{ display: false }],
					xAxes: [{ ticks: { fontColor: '#fff', fontSize: 16 } }]
				}
			}
		}`, func() string {
			b, _ := json.Marshal(hexes)
			return string(b)
		}(), func() string {
			b, _ := json.Marshal(hexes)
			return string(b)
		}())
		quickChartURL := fmt.Sprintf("https://quickchart.io/chart?width=500&height=100&c=%s", url.QueryEscape(chartCfg))
		imgResp, err := http.Get(quickChartURL)
		if err != nil {
			return ctx.Reply(fmt.Sprintf("[+] Dominant Colors:\n%s", colorBlocks))
		}
		defer imgResp.Body.Close()
		chartData, _ := io.ReadAll(imgResp.Body)
		_, err = ctx.Session.ChannelMessageSendComplex(ctx.ChanID(), &discordgo.MessageSend{
			Content: fmt.Sprintf("[+] Dominant Colors:\n%s", colorBlocks),
			Files: []*discordgo.File{
				{
					Name:        "palette.png",
					ContentType: "image/png",
					Reader:      bytes.NewReader(chartData),
				},
			},
		})
		return err
	},
}
var rxEmoji = regexp.MustCompile(`<a?:([a-zA-Z0-9_]+):(\d+)>`)
type Stealable struct {
	Type       string                        
	Name       string
	ID         string
	URL        string
	IsAnimated bool
}
type StealSession struct {
	AuthorID   string
	Stealables []Stealable
	Index      int
}
var (
	stealSessionsMu sync.Mutex
	stealSessions   = make(map[string]*StealSession)                        
)
var Steal = &manager.Command{
	Trigger:     "steal",
	Aliases:     []string{"enlarge", "jumbo"},
	Name:        "steal",
	Description: "Steals emoji or sticker from a message",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		p, err := ctx.Session.UserChannelPermissions(ctx.AuthorID(), ctx.ChanID())
		if err != nil || (p&discordgo.PermissionManageGuildExpressions) == 0 {
			return ctx.Reply(fmt.Sprintf("%s You need Manage Emojis/Stickers permission to steal emojis.", ctx.ErrorEmoji()))
		}
		var targetMsg *discordgo.Message
		if ctx.Message != nil && ctx.Message.ReferencedMessage != nil {
			targetMsg = ctx.Message.ReferencedMessage
		}
		var stealables []Stealable
		if len(ctx.Args) > 0 {
			match := rxEmoji.FindStringSubmatch(ctx.Args[0])
			if len(match) >= 3 {
				isAnimated := strings.Contains(match[0], "<a:")
				ext := "png"
				if isAnimated {
					ext = "gif"
				}
				stealables = append(stealables, Stealable{
					Type:       "emoji",
					Name:       match[1],
					ID:         match[2],
					URL:        fmt.Sprintf("https://cdn.discordapp.com/emojis/%s.%s", match[2], ext),
					IsAnimated: isAnimated,
				})
			} else {
				if targetMsg == nil {
					stealables = getRecentStealables(ctx.Session, ctx.ChanID())
				} else {
					stealables = extractStealables(targetMsg)
				}
			}
		} else {
			if targetMsg != nil {
				stealables = extractStealables(targetMsg)
			} else {
				stealables = getRecentStealables(ctx.Session, ctx.ChanID())
			}
		}
		if len(stealables) == 0 {
			return ctx.Reply(fmt.Sprintf("%s No custom emojis or stickers found.", ctx.ErrorEmoji()))
		}
		session := &StealSession{
			AuthorID:   ctx.AuthorID(),
			Stealables: stealables,
			Index:      0,
		}
		embed := buildStealEmbed(ctx.Cfg, session)
		components := buildStealComponents(session, "")
		respMsg, err := ctx.RespondAndGet(embed)
		if err != nil {
			return err
		}
		components = buildStealComponents(session, respMsg.ID)
		_, err = ctx.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
			ID:         respMsg.ID,
			Channel:    ctx.ChanID(),
			Components: &components,
		})
		stealSessionsMu.Lock()
		stealSessions[respMsg.ID] = session
		stealSessionsMu.Unlock()
		return err
	},
}
func buildStealEmbed(cfg config.ResCfg, session *StealSession) *discordgo.MessageEmbed {
	item := session.Stealables[session.Index]
	title := "Steal Emoji"
	if item.Type == "sticker" {
		title = "Steal Sticker"
	}
	desc := fmt.Sprintf("Page **%d** of **%d**\n\n**Name:** `%s`\n**Type:** `%s`\n**ID:** `%s`",
		session.Index+1, len(session.Stealables), item.Name, item.Type, item.ID)
	emb := config.Build(cfg, config.EmbedOpt{
		Title:       title,
		Description: desc,
	})
	emb.Image = &discordgo.MessageEmbedImage{
		URL: item.URL,
	}
	return emb
}
func buildStealComponents(session *StealSession, msgID string) []discordgo.MessageComponent {
	prevDisabled := session.Index == 0
	nextDisabled := session.Index == len(session.Stealables)-1
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "◀️",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("steal_prev_%s_%d", msgID, session.Index),
					Disabled: prevDisabled,
				},
				discordgo.Button{
					Label:    "Steal",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("steal_add_%s_%d", msgID, session.Index),
				},
				discordgo.Button{
					Label:    "▶️",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("steal_next_%s_%d", msgID, session.Index),
					Disabled: nextDisabled,
				},
			},
		},
	}
}
func extractStealables(msg *discordgo.Message) []Stealable {
	var out []Stealable
	if msg == nil {
		return out
	}
	for _, item := range msg.StickerItems {
		ext := "png"
		if item.FormatType == discordgo.StickerFormatTypeGIF {
			ext = "gif"
		}
		out = append(out, Stealable{
			Type: "sticker",
			Name: item.Name,
			ID:   item.ID,
			URL:  fmt.Sprintf("https://cdn.discordapp.com/stickers/%s.%s", item.ID, ext),
		})
	}
	matches := rxEmoji.FindAllStringSubmatch(msg.Content, -1)
	for _, match := range matches {
		name := match[1]
		id := match[2]
		isAnimated := strings.Contains(match[0], "<a:")
		ext := "png"
		if isAnimated {
			ext = "gif"
		}
		duplicate := false
		for _, ex := range out {
			if ex.Type == "emoji" && ex.ID == id {
				duplicate = true
				break
			}
		}
		if !duplicate {
			out = append(out, Stealable{
				Type:       "emoji",
				Name:       name,
				ID:         id,
				URL:        fmt.Sprintf("https://cdn.discordapp.com/emojis/%s.%s", id, ext),
				IsAnimated: isAnimated,
			})
		}
	}
	return out
}
func getRecentStealables(s *discordgo.Session, chanID string) []Stealable {
	var out []Stealable
	history, err := s.ChannelMessages(chanID, 50, "", "", "")
	if err != nil {
		return out
	}
	for _, m := range history {
		items := extractStealables(m)
		for _, item := range items {
			duplicate := false
			for _, ex := range out {
				if ex.Type == item.Type && ex.ID == item.ID {
					duplicate = true
					break
				}
			}
			if !duplicate {
				out = append(out, item)
			}
		}
	}
	return out
}
func resizeStaticImage(img image.Image, width, height int) image.Image {
	newImg := image.NewRGBA(image.Rect(0, 0, width, height))
	dx := img.Bounds().Dx()
	dy := img.Bounds().Dy()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := x * dx / width
			srcY := y * dy / height
			newImg.Set(x, y, img.At(srcX, srcY))
		}
	}
	return newImg
}
func downloadAndOptimizeAsset(url string, isSticker bool) ([]byte, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	contentType := http.DetectContentType(data)
	limit := 256 * 1024
	if isSticker {
		limit = 512 * 1024
	}
	if len(data) <= limit {
		return data, contentType, nil
	}
	if (strings.HasPrefix(contentType, "image/png") || strings.HasPrefix(contentType, "image/jpeg")) && !isSticker {
		img, _, err := image.Decode(bytes.NewReader(data))
		if err == nil {
			resized := resizeStaticImage(img, 128, 128)
			var buf bytes.Buffer
			err = png.Encode(&buf, resized)
			if err == nil && buf.Len() <= limit {
				return buf.Bytes(), "image/png", nil
			}
		}
	}
	return nil, "", fmt.Errorf("file size (%d KB) exceeds limit (%d KB)", len(data)/1024, limit/1024)
}
func createGuildSticker(s *discordgo.Session, guildID string, name string, tags string, fileBytes []byte, contentType string) (*discordgo.Sticker, error) {
	url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/stickers", guildID)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", name)
	_ = writer.WriteField("tags", tags)
	_ = writer.WriteField("description", "")
	ext := "png"
	if contentType == "image/gif" {
		ext = "gif"
	}
	part, err := writer.CreateFormFile("file", "sticker."+ext)
	if err != nil {
		return nil, err
	}
	_, _ = part.Write(fileBytes)
	_ = writer.Close()
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", s.Token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discord API error: status %d: %s", resp.StatusCode, string(respBytes))
	}
	var sticker discordgo.Sticker
	if err := json.NewDecoder(resp.Body).Decode(&sticker); err != nil {
		return nil, err
	}
	return &sticker, nil
}
func HandleStealComponent(s *discordgo.Session, i *discordgo.InteractionCreate, mgr *manager.Manager) {
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, "_")
	if len(parts) < 4 {
		return
	}
	action := parts[1]
	msgID := parts[2]
	idx, _ := strconv.Atoi(parts[3])
	stealSessionsMu.Lock()
	session, exists := stealSessions[msgID]
	stealSessionsMu.Unlock()
	errEmoji := mgr.ResolveEmoji(s, i.GuildID, "sys_x")
	successEmoji := mgr.ResolveEmoji(s, i.GuildID, "sys_checkmark")
	if !exists {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("%s Interactive session expired.", errEmoji),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	if i.Member == nil || i.Member.User == nil || i.Member.User.ID != session.AuthorID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("%s This is not your interactive session.", errEmoji),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	if idx < 0 || idx >= len(session.Stealables) {
		return
	}
	cfg, _ := mgr.DB().GetBot(s.State.User.ID)
	resolvedCfg := config.Resolve(config.GetGlobal(), cfg)
	switch action {
	case "prev":
		if session.Index > 0 {
			session.Index--
		}
	case "next":
		if session.Index < len(session.Stealables)-1 {
			session.Index++
		}
	case "add":
		item := session.Stealables[session.Index]
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		p, err := s.UserChannelPermissions(session.AuthorID, i.ChannelID)
		if err != nil || (p&discordgo.PermissionManageGuildExpressions) == 0 {
			followUp(s, i.Interaction, fmt.Sprintf("%s You do not have permission to manage emojis/stickers in this server.", errEmoji))
			return
		}
		isSticker := item.Type == "sticker"
		fileBytes, contentType, err := downloadAndOptimizeAsset(item.URL, isSticker)
		if err != nil {
			followUp(s, i.Interaction, fmt.Sprintf("%s Failed to download/compress asset: %v", errEmoji, err))
			return
		}
		if isSticker {
			sticker, err := createGuildSticker(s, i.GuildID, item.Name, "stolen", fileBytes, contentType)
			if err != nil {
				followUp(s, i.Interaction, fmt.Sprintf("%s Failed to create sticker: %v", errEmoji, err))
				return
			}
			followUp(s, i.Interaction, fmt.Sprintf("%s Successfully stole and added sticker **%s**!", successEmoji, sticker.Name))
		} else {
			b64 := "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(fileBytes)
			newEmoji, err := s.GuildEmojiCreate(i.GuildID, &discordgo.EmojiParams{
				Name:  item.Name,
				Image: b64,
			})
			if err != nil {
				followUp(s, i.Interaction, fmt.Sprintf("%s Failed to create emoji: %v", errEmoji, err))
				return
			}
			followUp(s, i.Interaction, fmt.Sprintf("%s Successfully stole and added emoji %s as `%s`!", successEmoji, newEmoji.MessageFormat(), newEmoji.Name))
		}
		return
	}
	embed := buildStealEmbed(resolvedCfg, session)
	components := buildStealComponents(session, msgID)
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}
func followUp(s *discordgo.Session, i *discordgo.Interaction, text string) {
	_, _ = s.FollowupMessageCreate(i, true, &discordgo.WebhookParams{
		Content: text,
	})
}
func getWeatherStyle(code string, desc string) (string, int) {
	d := strings.ToLower(desc)
	c, _ := strconv.Atoi(code)
	switch {
	case c == 113 || strings.Contains(d, "sunny") || strings.Contains(d, "clear"):
		return "☀️", 0xFFAA00
	case c == 116 || strings.Contains(d, "partly cloudy"):
		return "⛅", 0xAABBDD
	case c == 119 || strings.Contains(d, "cloudy"):
		return "☁️", 0x888888
	case c == 122 || strings.Contains(d, "overcast"):
		return "☁️", 0x555555
	case c == 143 || c == 248 || c == 260 || strings.Contains(d, "mist") || strings.Contains(d, "fog"):
		return "🌫️", 0xCCCCCC
	case strings.Contains(d, "thunder") || strings.Contains(d, "storm"):
		return "🌩️", 0x663399
	case strings.Contains(d, "snow") || strings.Contains(d, "sleet") || strings.Contains(d, "ice") || strings.Contains(d, "blizzard"):
		return "❄️", 0xCCFFFF
	case strings.Contains(d, "rain") || strings.Contains(d, "drizzle") || strings.Contains(d, "shower"):
		return "🌧️", 0x3366CC
	case strings.Contains(d, "wind") || strings.Contains(d, "blow"):
		return "💨", 0x77AADD
	default:
		return "🌡️", 0x2b2d31
	}
}
var Weather = &manager.Command{
	Trigger:     "weather",
	Name:        "weather",
	Description: "Provides detailed weather information for any location",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("weather")
		}
		loc := strings.Join(ctx.Args, " ")
		apiURL := fmt.Sprintf("https://wttr.in/%s?format=j1", url.QueryEscape(loc))
		res, err := http.Get(apiURL)
		if err != nil {
			return ctx.Reply("[!] Weather API offline.")
		}
		defer res.Body.Close()
		var data struct {
			Current []struct {
				TempC       string `json:"temp_C"`
				TempF       string `json:"temp_F"`
				FeelsLikeC  string `json:"FeelsLikeC"`
				FeelsLikeF  string `json:"FeelsLikeF"`
				Humidity    string `json:"humidity"`
				WindSpeed   string `json:"windspeedKmph"`
				UVIndex     string `json:"uvIndex"`
				PrecipMM    string `json:"precipMM"`
				WeatherCode string `json:"weatherCode"`
				Desc        []struct {
					Value string `json:"value"`
				} `json:"weatherDesc"`
			} `json:"current_condition"`
			Area []struct {
				Name []struct {
					Value string `json:"value"`
				} `json:"areaName"`
				Region []struct {
					Value string `json:"value"`
				} `json:"region"`
				Country []struct {
					Value string `json:"value"`
				} `json:"country"`
			} `json:"nearest_area"`
		}
		if err := json.NewDecoder(res.Body).Decode(&data); err != nil || len(data.Current) == 0 {
			return ctx.Reply(fmt.Sprintf("[!] Weather info for `%s` not found.", loc))
		}
		cur := data.Current[0]
		desc := "Unknown"
		if len(cur.Desc) > 0 {
			desc = cur.Desc[0].Value
		}
		place := loc
		if len(data.Area) > 0 {
			a := data.Area[0]
			var parts []string
			if len(a.Name) > 0 && a.Name[0].Value != "" {
				parts = append(parts, a.Name[0].Value)
			}
			if len(a.Region) > 0 && a.Region[0].Value != "" {
				parts = append(parts, a.Region[0].Value)
			}
			if len(a.Country) > 0 && a.Country[0].Value != "" {
				parts = append(parts, a.Country[0].Value)
			}
			if len(parts) > 0 {
				place = strings.Join(parts, ", ")
			}
		}
		emoji, color := getWeatherStyle(cur.WeatherCode, desc)
		descText := fmt.Sprintf(
			"**Condition:** %s %s\n"+
				"**Temperature:** %s°C / %s°F *(Feels like: %s°C / %s°F)*\n"+
				"**Humidity:** %s%%\n"+
				"**Wind Speed:** %s km/h\n"+
				"**UV Index:** %s\n"+
				"**Precipitation:** %s mm",
			desc, emoji,
			cur.TempC, cur.TempF, cur.FeelsLikeC, cur.FeelsLikeF,
			cur.Humidity,
			cur.WindSpeed,
			cur.UVIndex,
			cur.PrecipMM,
		)
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       fmt.Sprintf("Weather in %s", place),
			Description: descText,
		})
		emb.Color = color
		return ctx.Respond(emb)
	},
}