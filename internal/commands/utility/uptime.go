package utility

import (
	"fmt"
	"time"

	"skyvern/internal/config"
	"skyvern/internal/manager"
)

var Uptime = &manager.Command{
	Trigger:     "uptime",
	Name:        "uptime",
	Description: "Shows how long the bot engine has been running",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		diff := time.Since(ctx.Mgr.BootTime).Round(time.Second)
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "Bot Uptime",
			Description: fmt.Sprintf("Online for: **%s**", diff),
		})
		emb.Color = 0x2b2d31
		return ctx.Respond(emb)
	},
}
