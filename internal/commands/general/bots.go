package general

import (
	"fmt"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"strings"
)

func init() {
	manager.RegisterHelp("bots", []manager.HelpPage{
		{
			Command:     "Bots List",
			Syntax:      ".bots",
			Description: "Lists all bots present in the current server.",
		},
	})
}

var Bots = &manager.Command{
	Trigger:     "bots",
	Name:        "bots",
	Description: "View all bots in the server",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		members, err := ctx.Session.GuildMembers(ctx.GuildID(), "", 1000)
		if err != nil {
			return ctx.Reply("[!] Failed to fetch guild members.")
		}

		var bots []string
		for _, m := range members {
			if m.User != nil && m.User.Bot {
				bots = append(bots, fmt.Sprintf("<@%s> (`%s`)", m.User.ID, m.User.ID))
			}
		}

		if len(bots) == 0 {
			return ctx.Reply("[*] No bots found in this server.")
		}

		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       fmt.Sprintf("Bots in Server (%d)", len(bots)),
			Description: strings.Join(bots, "\n"),
		})
		return ctx.Respond(emb)
	},
}
