package fun

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"net/http"
	"skyvern/internal/manager"
	"skyvern/internal/moderation"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/golang/freetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"
	_ "golang.org/x/image/webp"
)

func init() {
	manager.RegisterHelp("fakemessage", []manager.HelpPage{
		{
			Command:     "Fake Message",
			Syntax:      ".fakemessage <user> <message>",
			Description: "Generates a fake Discord message image from the specified user.",
		},
	})
}

var FakeMessage = &manager.Command{
	Trigger:     "fakemessage",
	Aliases:     []string{"fakemsg", "fakedm"},
	Name:        "fakemessage",
	Description: "Generates an image of a fake message from the specified user",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		var targetUser *discordgo.User
		var messageText string

		if ctx.Interact != nil {
			for _, opt := range ctx.Interact.ApplicationCommandData().Options {
				if opt.Name == "user" {
					targetUser = opt.UserValue(ctx.Session)
				} else if opt.Name == "message" {
					messageText = opt.StringValue()
				}
			}
			if ctx.GuildID() == "" {
				userID := strings.Trim(ctx.Args[0], "<@!>")
				if u, err := ctx.Session.User(userID); err == nil {
					targetUser = u
				} else {
					return ctx.Reply("[!] Could not resolve user in DMs. Use a mention or user ID.")
				}
			} else {
				member, err := moderation.ResolveMember(ctx.Session, ctx.GuildID(), ctx.Args[0])
				if err != nil || member == nil {
					return ctx.Reply("[!] Could not resolve user.")
				}
				targetUser = member.User
			}
			messageText = strings.Join(ctx.Args[1:], " ")
		}

		if targetUser == nil || messageText == "" {
			return ctx.Reply("[!] Invalid user or message content.")
		}

		if isBotOwner(targetUser.ID) {
			return ctx.Reply("[!] Security Restriction: Protected user.")
		}

		f, err := freetype.ParseFont(goregular.TTF)
		if err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Failed to parse regular font: %v", err))
		}
		fBold, err := freetype.ParseFont(gobold.TTF)
		if err != nil {
			fBold = f
		}

		const canvasWidth = 700
		const padding = 20
		const avatarSize = 40
		const textX = padding + avatarSize + 12
		const contentWidth = canvasWidth - textX - padding

		d := &font.Drawer{Face: truetypeFace(f, 16)}
		lines := wrapText(d, messageText, contentWidth)
		textHeight := len(lines) * 20
		canvasHeight := padding + 16 + 5 + textHeight + padding
		if canvasHeight < 80 {
			canvasHeight = 80
		}

		canvas := image.NewRGBA(image.Rect(0, 0, canvasWidth, canvasHeight))
		bgColor := color.RGBA{0x31, 0x33, 0x38, 0xff}
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(bgColor), image.Point{}, draw.Src)

		if resp, err := http.Get(targetUser.AvatarURL("128")); err == nil {
			defer resp.Body.Close()
			if dec, _, err := image.Decode(resp.Body); err == nil {
				drawCircleAvatar(canvas, dec, padding, padding, avatarSize)
			}
		}

		displayName := targetUser.Username
		if member, err := ctx.Session.GuildMember(ctx.GuildID(), targetUser.ID); err == nil && member != nil && member.Nick != "" {
			displayName = member.Nick
		}

		du := &font.Drawer{
			Dst:  canvas,
			Src:  image.NewUniform(color.RGBA{0xF2, 0xF3, 0xF5, 0xff}),
			Face: truetypeFace(fBold, 16),
		}
		du.Dot = fixed.P(textX, padding+16)
		du.DrawString(displayName)
		userW := du.MeasureString(displayName).Round()

		dt := &font.Drawer{
			Dst:  canvas,
			Src:  image.NewUniform(color.RGBA{0x94, 0x9B, 0xA4, 0xff}),
			Face: truetypeFace(f, 12),
		}
		dt.Dot = fixed.P(textX+userW+8, padding+14)
		dt.DrawString("Today at " + time.Now().Format("3:04 PM"))

		dx := &font.Drawer{
			Dst:  canvas,
			Src:  image.NewUniform(color.RGBA{0xDB, 0xDE, 0xE1, 0xff}),
			Face: truetypeFace(f, 16),
		}

		currY := padding + 16 + 5
		for _, line := range lines {
			dx.Dot = fixed.P(textX, currY+16)
			dx.DrawString(line)
			currY += 20
		}

		var buf bytes.Buffer
		if err := png.Encode(&buf, canvas); err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Failed to encode PNG: %v", err))
		}
		imgBytes := buf.Bytes()

		if ctx.Interact != nil {
			return ctx.Session.InteractionRespond(ctx.Interact, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Files: []*discordgo.File{
						{
							Name:        "fakemessage.png",
							ContentType: "image/png",
							Reader:      bytes.NewReader(imgBytes),
						},
					},
				},
			})
		}

		_, err = ctx.Session.ChannelMessageSendComplex(ctx.ChanID(), &discordgo.MessageSend{
			Files: []*discordgo.File{
				{
					Name:        "fakemessage.png",
					ContentType: "image/png",
					Reader:      bytes.NewReader(imgBytes),
				},
			},
		})
		return err
	},
}

func drawCircleAvatar(dst draw.Image, src image.Image, x, y, size int) {
	r := size / 2
	cx, cy := x+r, y+r
	w, h := src.Bounds().Dx(), src.Bounds().Dy()

	for dy := 0; dy < size; dy++ {
		for dx := 0; dx < size; dx++ {
			px, py := x+dx, y+dy
			if (px-cx)*(px-cx)+(py-cy)*(py-cy) <= r*r {
				sx := dx * w / size
				sy := dy * h / size
				dst.Set(px, py, src.At(sx, sy))
			}
		}
	}
}

func wrapText(d *font.Drawer, text string, maxWidth int) []string {
	var lines []string
	for _, para := range strings.Split(text, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		cur := words[0]
		for _, w := range words[1:] {
			if d.MeasureString(cur + " " + w).Round() < maxWidth {
				cur += " " + w
			} else {
				lines = append(lines, cur)
				cur = w
			}
		}
		lines = append(lines, cur)
	}
	return lines
}
