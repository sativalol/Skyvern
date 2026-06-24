package general

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"skyvern/internal/config"
	"skyvern/internal/manager"
)

var Yo = &manager.Command{
	Trigger:     "yo",
	Name:        "yo",
	Description: "Check bot responsiveness, connection latency and uptime",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		targetID := "1281996800340791452"
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
			
			fmt.Sprintf("%s's jbl speaker", name),
			site,
		}

		idx := int(time.Now().UnixNano() % int64(len(titles)))
		if idx < 0 {
			idx = -idx
		}
		randomTitle := titles[idx]

		t := time.Now()
		msg, err := ctx.ReplyAndGet(fmt.Sprintf("uhh, pinging %s...", randomTitle))
		if err != nil {
			return err
		}
		restPing := time.Since(t)
		wsPing := ctx.Session.HeartbeatLatency()
		uptime := time.Since(ctx.Mgr.BootTime).Round(time.Second)

		var fields []*discordgo.MessageEmbedField
		fields = append(fields, config.Field("latency", fmt.Sprintf("`%dms`", wsPing.Milliseconds()), true))
		fields = append(fields, config.Field("api", fmt.Sprintf("`%dms`", restPing.Milliseconds()), true))
		fields = append(fields, config.Field("uptime", fmt.Sprintf("`%s`", uptime), true))

		if ctx.Session.ShardCount > 1 {
			fields = append(fields, config.Field("shard", fmt.Sprintf("`%d/%d`", ctx.Session.ShardID, ctx.Session.ShardCount), true))
		}

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
