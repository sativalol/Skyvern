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
	"log"
	"math"
	"net/http"
	"os"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	_ "golang.org/x/image/webp"
)

func init() {
	manager.RegisterHelp("quote", []manager.HelpPage{
		{
			Command:     "Quote",
			Syntax:      ".quote [text] or reply to a message",
			Description: "Create a styled quote card. Reply to a message to quote it, or provide text to quote yourself.",
		},
	})
}

var Quote = &manager.Command{
	Trigger:     "quote",
	Aliases:     []string{"q"},
	Name:        "quote",
	Description: "Create a styled quote embed",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		var content string
		var targetUser *discordgo.User
		var ts time.Time

		if ctx.Interact != nil {
			var textOpt string
			var userOpt *discordgo.User
			for _, opt := range ctx.Interact.ApplicationCommandData().Options {
				switch opt.Name {
				case "text":
					textOpt = opt.StringValue()
				case "user":
					userOpt = opt.UserValue(ctx.Session)
				}
			}

			if textOpt != "" {
				content = textOpt
				if userOpt != nil {
					targetUser = userOpt
				} else {
					targetUser = ctx.Interact.Member.User
				}
				ts = time.Now()
			} else {
				return ctx.Reply("[!] Please provide text to quote.")
			}
		} else {
			custom := strings.Join(ctx.Args, " ")
			if custom != "" {
				content = custom
				if ctx.Message != nil && ctx.Message.Author != nil {
					targetUser = ctx.Message.Author
				}
				ts = time.Now()
			} else if ctx.Message != nil && ctx.Message.ReferencedMessage != nil {
				ref := ctx.Message.ReferencedMessage
				if ref.Content == "" {
					return ctx.Reply("[!] That message has no text content to quote.")
				}
				content = ref.Content
				targetUser = ref.Author
				t, err := discordgo.SnowflakeTimestamp(ref.ID)
				if err == nil {
					ts = t
				} else {
					ts = time.Now()
				}
			} else {
				return ctx.Reply("[!] Reply to a message or provide text to quote.")
			}
		}

		if targetUser == nil {
			return ctx.Reply("[!] Could not resolve the quoted user.")
		}

		if len(content) > 1800 {
			content = content[:1797] + "..."
		}

		avatarURL := targetUser.AvatarURL("512")

		var imgBytes []byte
		var imgErr error

		avatarResp, err := http.Get(avatarURL)
		if err != nil {
			log.Printf("Quote avatar download err: %v", err)
		} else {
			defer avatarResp.Body.Close()
			dec, _, err := image.Decode(avatarResp.Body)
			if err != nil {
				log.Printf("Quote avatar decode err: %v", err)
			} else {
				var fontPath string
				for _, p := range []string{
					"C:/Windows/Fonts/arial.ttf",
					"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
					"/usr/share/fonts/TTF/DejaVuSans.ttf",
					"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
					"/usr/share/fonts/truetype/freefont/FreeSans.ttf",
				} {
					if _, err := os.Stat(p); err == nil {
						fontPath = p
						break
					}
				}
				if fontPath == "" {
					log.Printf("Quote no system font found")
				} else {
					fBytes, err := os.ReadFile(fontPath)
					if err != nil {
						log.Printf("Quote read font err: %v", err)
					} else {
						f, err := freetype.ParseFont(fBytes)
						if err != nil {
							log.Printf("Quote parse font err: %v", err)
						} else {
							canvas := image.NewRGBA(image.Rect(0, 0, 1200, 600))
							draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.Black), image.Point{}, draw.Src)
							drawImageProp(canvas, dec, 0, 0, 700, 600)
							for y := 0; y < 600; y++ {
								for x := 0; x < 1200; x++ {
									t := float64(x) / 1200.0
									alpha := getAlpha(t)
									c := canvas.At(x, y)
									r, g, b, _ := c.RGBA()
									factor := 1.0 - alpha
									newR := uint8(float64(r>>8) * factor)
									newG := uint8(float64(g>>8) * factor)
									newB := uint8(float64(b>>8) * factor)
									canvas.Set(x, y, color.RGBA{newR, newG, newB, 255})
								}
							}
							d := &font.Drawer{
								Dst: canvas,
								Src: image.NewUniform(color.White),
								Face: truetypeFace(f, 38),
							}
							lines := getLines(d, `"`+content+`"`, 500)
							lineHeight := 48
							totalHeight := len(lines) * lineHeight
							startY := 300 - (totalHeight / 2) - 30
							for i, line := range lines {
								w := d.MeasureString(line).Round()
								d.Dot = fixed.P(900-w/2, startY+i*lineHeight+38)
								d.DrawString(line)
							}
							d.Face = truetypeFace(f, 28)
							d.Src = image.NewUniform(color.RGBA{221, 221, 221, 255})
							authorText := "- " + targetUser.Username
							wAuthor := d.MeasureString(authorText).Round()
							authorY := startY + totalHeight + 20
							d.Dot = fixed.P(900-wAuthor/2, authorY+28)
							d.DrawString(authorText)
							d.Face = truetypeFace(f, 18)
							d.Src = image.NewUniform(color.RGBA{119, 119, 119, 255})
							handleText := "@" + strings.ToLower(targetUser.Username)
							wHandle := d.MeasureString(handleText).Round()
							d.Dot = fixed.P(900-wHandle/2, authorY+35+18)
							d.DrawString(handleText)
							var buf bytes.Buffer
							if err := png.Encode(&buf, canvas); err != nil {
								log.Printf("Quote png encode err: %v", err)
							} else {
								imgBytes = buf.Bytes()
							}
						}
					}
				}
			}
		}

		if imgErr == nil && len(imgBytes) > 0 {
			if ctx.Interact != nil {
				return ctx.Session.InteractionRespond(ctx.Interact, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Files: []*discordgo.File{
							{
								Name:        "quote.png",
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
						Name:        "quote.png",
						ContentType: "image/png",
						Reader:      bytes.NewReader(imgBytes),
					},
				},
			})
			return err
		}

		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Description: fmt.Sprintf("*\"%s\"*", content),
		})
		emb.Author = &discordgo.MessageEmbedAuthor{
			Name:    targetUser.Username,
			IconURL: avatarURL,
		}
		emb.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: avatarURL,
		}
		emb.Timestamp = ts.Format(time.RFC3339)
		emb.Color = 0x2b2d31

		return ctx.Respond(emb)
	},
}

func isBotOwner(uid string) bool {
	g := config.GetGlobal()
	cleanUID := cleanID(uid)
	if cleanUID == "" {
		return false
	}
	repl := strings.NewReplacer(";", ",", " ", ",", "\n", ",", "\r", ",", "\t", ",")
	if g.OwnerIDs != "" {
		for _, part := range strings.Split(repl.Replace(g.OwnerIDs), ",") {
			if cleanID(part) == cleanUID {
				return true
			}
		}
	}
	if g.OwnerIDs == "" {
		for _, part := range strings.Split(repl.Replace(config.DefGlobal().OwnerIDs), ",") {
			if cleanID(part) == cleanUID {
				return true
			}
		}
	}
	return false
}

func cleanID(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func drawImageProp(dst draw.Image, src image.Image, x, y, w, h int) {
	srcW := src.Bounds().Dx()
	srcH := src.Bounds().Dy()
	scale := math.Max(float64(w)/float64(srcW), float64(h)/float64(srcH))
	cx := (srcW - int(float64(w)/scale)) / 2
	cy := (srcH - int(float64(h)/scale)) / 2
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			sx := cx + int(float64(dx)/scale)
			sy := cy + int(float64(dy)/scale)
			if sx >= 0 && sx < srcW && sy >= 0 && sy < srcH {
				dst.Set(x+dx, y+dy, src.At(sx, sy))
			}
		}
	}
}

func getAlpha(t float64) float64 {
	if t <= 0.35 {
		return 0.2 + (0.6-0.2)*(t/0.35)
	} else if t <= 0.55 {
		return 0.6 + (1.0-0.6)*((t-0.35)/0.20)
	}
	return 1.0
}

func truetypeFace(f *truetype.Font, size float64) font.Face {
	return truetype.NewFace(f, &truetype.Options{
		Size: size,
		DPI:  72,
	})
}

func getLines(d *font.Drawer, text string, maxWidth int) []string {
	words := strings.Split(text, " ")
	var lines []string
	currentLine := words[0]
	for i := 1; i < len(words); i++ {
		word := words[i]
		width := d.MeasureString(currentLine + " " + word).Round()
		if width < maxWidth {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	lines = append(lines, currentLine)
	return lines
}
