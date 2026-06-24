package moon

import (
	"strings"
	"time"

	"skyvern/internal/manager"
	"skyvern/internal/plugins"
	"skyvern/internal/storage"
)

type MoonPlugin struct{}

func init() {
	plugins.Register(&MoonPlugin{})
}

func (p *MoonPlugin) Name() string {
	return "moon"
}

func (p *MoonPlugin) Init(db *storage.DB, mgr *manager.Manager) error {
	return nil
}

func (p *MoonPlugin) Commands() []*manager.Command {
	return []*manager.Command{
		{
			Trigger:     "moon",
			Aliases:     []string{"mooncycle"},
			Name:        "moon",
			Description: "Show current moon phase in terminal style ASCII art",
			Category:    "fun",
			Execute: func(ctx *manager.CommandContext) error {
				// moon cycle calculation, output inside code block
				return ctx.Reply("```\n" + getMoonPhase(time.Now()) + "```")
			},
		},
	}
}

func getMoonPhase(t time.Time) string {
	b := 44
	a := 2551443

	age := (t.Unix() - 592531) % int64(a)
	if age < 0 {
		age += int64(a)
	}
	z := (age * 512) / int64(a)

	var sb strings.Builder
	aLimit := a

	for y := 2 - b; y <= b; y += 4 {
		x := -b
		for {
			x++
			if x >= aLimit {
				sb.WriteByte('\n')
				break
			}

			if x < 0 {
				if x*x+y*y < b*b {
					aLimit = 1 - x
					x = 1
				} else {
					sb.WriteByte(' ')
				}
			} else {
				var idx int
				if z < 256 {
					val := (x < aLimit * int(255 - z) / 256)
					if val {
						idx = 1
					} else {
						idx = 0
					}
				} else {
					val := (x < aLimit * int(511 - z) / 256)
					if val {
						idx = 0
					} else {
						idx = 1
					}
				}
				sb.WriteByte("#."[idx])
			}
		}
		aLimit = a
	}
	return sb.String()
}
