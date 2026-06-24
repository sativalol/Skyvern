package general

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"skyvern/internal/config"
	"skyvern/internal/manager"
)

var Ping = &manager.Command{
	Trigger:     "ping",
	Name:        "ping",
	Description: "Check bot responsiveness and connection latency",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		targetID := "1478564651536093409"
		targetUser, err := ctx.Session.User(targetID)
		name := "Unknown"
		if err == nil && targetUser != nil {
			name = targetUser.Username
		}

		site := "esoteric.win"
		titles := []string{
			fmt.Sprintf("%s's 2g yart", name),
			fmt.Sprintf("%s's penjamin", name),
			fmt.Sprintf("%s's private jet", name),
			fmt.Sprintf("%s's bugatti", name),
			fmt.Sprintf("%s's koenigsegg", name),
			fmt.Sprintf("%s's 11.2 Million Dollar Yacht", name),
			fmt.Sprintf("%s's satellite", name),
			fmt.Sprintf("%s's mc server", name),
			fmt.Sprintf("%s's jbl speaker", name),
			site,
		}

		idx := int(time.Now().UnixNano() % int64(len(titles)))
		if idx < 0 {
			idx = -idx
		}
		randomTitle := titles[idx]

		t := time.Now()
		msg, err := ctx.ReplyAndGet(fmt.Sprintf("Pinging %s...", randomTitle))
		if err != nil {
			return err
		}
		restPing := time.Since(t)
		wsPing := ctx.Session.HeartbeatLatency()

		uptime := time.Since(ctx.Mgr.BootTime)
		d := int(uptime.Hours() / 24)
		h := int(uptime.Hours()) % 24
		m := int(uptime.Minutes()) % 60
		s := int(uptime.Seconds()) % 60
		
		var uptimeParts []string
		if d > 0 { uptimeParts = append(uptimeParts, fmt.Sprintf("%dd", d)) }
		if h > 0 { uptimeParts = append(uptimeParts, fmt.Sprintf("%dh", h)) }
		if m > 0 { uptimeParts = append(uptimeParts, fmt.Sprintf("%dm", m)) }
		if s > 0 || len(uptimeParts) == 0 { uptimeParts = append(uptimeParts, fmt.Sprintf("%ds", s)) }
		uptimeStr := strings.Join(uptimeParts, " ")

		var fields []*discordgo.MessageEmbedField
		fields = append(fields, config.Field("Bot Latency", fmt.Sprintf("`%dms`", restPing.Milliseconds()), true))
		fields = append(fields, config.Field("API Latency", fmt.Sprintf("`%dms`", wsPing.Milliseconds()), true))
		fields = append(fields, config.Field("Network", fmt.Sprintf("`Shard %d/%d`", ctx.Session.ShardID, ctx.Session.ShardCount), true))
		fields = append(fields, config.Field("Uptime", fmt.Sprintf("`%s`", uptimeStr), true))

		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:  fmt.Sprintf("it took %dms to ping %s", restPing.Milliseconds(), randomTitle),
			Fields: fields,
		})
		emb.Color = 0x2b2d31

		if ctx.Interact != nil {
			_, err = ctx.Session.InteractionResponseEdit(ctx.Interact, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{emb},
			})
			return err
		}
		if msg != nil {
			_, err = ctx.Session.ChannelMessageEditEmbed(ctx.ChanID(), msg.ID, emb)
			return err
		}
		return nil
	},
}
