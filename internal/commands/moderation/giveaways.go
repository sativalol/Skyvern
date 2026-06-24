package moderation

import (
	"fmt"
	"skyvern/internal/manager"
	"skyvern/internal/storage"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func HandleGiveawayJoin(s *discordgo.Session, i *discordgo.InteractionCreate, mgr *manager.Manager) {
	customID := i.MessageComponentData().CustomID
	mid := strings.TrimPrefix(customID, "giveaway_join_")
	gid := i.GuildID
	uid := i.Member.User.ID
	db := mgr.DB()

	g, err := db.GetGiveaway(gid, mid)
	if err != nil {
		respondEphemeral(s, i, "[!] Giveaway data not found.")
		return
	}
	if g.Ended {
		respondEphemeral(s, i, "[!] This giveaway has already ended.")
		return
	}

	member := i.Member
	if member == nil {
		respondEphemeral(s, i, "[!] Failed to fetch member info.")
		return
	}

	if g.AgeDays > 0 {
		t, err := manager.SnowflakeToTime(uid)
		if err == nil {
			age := time.Since(t)
			if age < time.Duration(g.AgeDays)*24*time.Hour {
				respondEphemeral(s, i, fmt.Sprintf("[!] Your account must be at least %d days old to enter (current: %d days).", g.AgeDays, int(age.Hours()/24)))
				return
			}
		}
	}

	if g.StayDays > 0 {
		stay := time.Since(member.JoinedAt)
		if stay < time.Duration(g.StayDays)*24*time.Hour {
			respondEphemeral(s, i, fmt.Sprintf("[!] You must be in this server for at least %d days to enter (current: %d days).", g.StayDays, int(stay.Hours()/24)))
			return
		}
	}

	if g.MinLevel > 0 || g.MaxLevel > 0 {
		xp, _ := db.GetUserXP(gid, uid)
		if g.MinLevel > 0 && xp.Level < g.MinLevel {
			respondEphemeral(s, i, fmt.Sprintf("[!] You must be at least level %d to enter (your level: %d).", g.MinLevel, xp.Level))
			return
		}
		if g.MaxLevel > 0 && xp.Level > g.MaxLevel {
			respondEphemeral(s, i, fmt.Sprintf("[!] You cannot enter if you are above level %d (your level: %d).", g.MaxLevel, xp.Level))
			return
		}
	}

	if len(g.RequiredRoles) > 0 {
		hasAll := true
		memberRoles := make(map[string]bool)
		for _, r := range member.Roles {
			memberRoles[r] = true
		}
		var missing []string
		for _, rr := range g.RequiredRoles {
			if !memberRoles[rr] {
				hasAll = false
				missing = append(missing, fmt.Sprintf("<@&%s>", rr))
			}
		}
		if !hasAll {
			respondEphemeral(s, i, fmt.Sprintf("[!] You are missing required roles to enter: %s", strings.Join(missing, ", ")))
			return
		}
	}

	for _, entry := range g.Entries {
		if entry == uid {
			respondEphemeral(s, i, "[!] You have already entered this giveaway!")
			return
		}
	}

	g.Entries = append(g.Entries, uid)
	_ = db.SaveGiveaway(g)

	resCfg, _ := mgr.ResolvedCfgFor(s.State.User.ID)
	emb := manager.BuildGiveawayEmbed(g, resCfg)
	comp := manager.BuildGiveawayButton(g.MessageID, g.Ended)
	_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    g.ChannelID,
		ID:         g.MessageID,
		Embeds:     &[]*discordgo.MessageEmbed{emb},
		Components: &[]discordgo.MessageComponent{comp},
	})

	respondEphemeral(s, i, "🎉 You have successfully entered the giveaway!")
}

func respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, text string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: text,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func init() {
	manager.RegisterHelp("giveaways", []manager.HelpPage{
		{
			Command:     "Giveaways Help",
			Syntax:      ".giveaways",
			Description: "Show help for the giveaways command system.",
		},
		{
			Command:     "Giveaways Start",
			Syntax:      ".giveaways start <channel> <duration> <winners> <prize>",
			Description: "Start a giveaway in the specified channel.",
		},
		{
			Command:     "Giveaways Reroll",
			Syntax:      ".giveaways reroll <message link> [winners]",
			Description: "Redraw winners for an ended giveaway.",
		},
		{
			Command:     "Giveaways End",
			Syntax:      ".giveaways end <message link>",
			Description: "End an active giveaway early and draw winners.",
		},
		{
			Command:     "Giveaways Cancel",
			Syntax:      ".giveaways cancel <message link>",
			Description: "Cancel an active giveaway.",
		},
		{
			Command:     "Giveaways List",
			Syntax:      ".giveaways list",
			Description: "List all active giveaways on the server.",
		},
		{
			Command:     "Giveaways Edit Roles",
			Syntax:      ".giveaways edit roles <message link> <roles>",
			Description: "Edit the roles awarded to winners upon winning.",
		},
		{
			Command:     "Giveaways Edit Image",
			Syntax:      ".giveaways edit image <message link> <url or attachment>",
			Description: "Edit the embed image.",
		},
		{
			Command:     "Giveaways Edit Age",
			Syntax:      ".giveaways edit age <message link> <days>",
			Description: "Edit minimum account age in days to enter.",
		},
		{
			Command:     "Giveaways Edit Color",
			Syntax:      ".giveaways edit color <message link> <color>",
			Description: "Edit embed color.",
		},
		{
			Command:     "Giveaways Edit Thumbnail",
			Syntax:      ".giveaways edit thumbnail <message link> <url or attachment>",
			Description: "Edit embed thumbnail.",
		},
		{
			Command:     "Giveaways Edit Max Level",
			Syntax:      ".giveaways edit maxlevel <message link> <level>",
			Description: "Edit maximum level requirement to enter.",
		},
		{
			Command:     "Giveaways Edit Required Roles",
			Syntax:      ".giveaways edit requiredroles <message link> <roles>",
			Description: "Edit roles required to enter.",
		},
		{
			Command:     "Giveaways Edit Stay",
			Syntax:      ".giveaways edit stay <message link> <days>",
			Description: "Edit minimum server stay in days to enter.",
		},
		{
			Command:     "Giveaways Edit Description",
			Syntax:      ".giveaways edit description <message link> <text>",
			Description: "Edit custom description in the embed.",
		},
		{
			Command:     "Giveaways Edit Prize",
			Syntax:      ".giveaways edit prize <message link> <prize>",
			Description: "Edit giveaway prize name.",
		},
		{
			Command:     "Giveaways Edit Winners",
			Syntax:      ".giveaways edit winners <message link> <count>",
			Description: "Edit giveaway winners count.",
		},
		{
			Command:     "Giveaways Edit Host",
			Syntax:      ".giveaways edit host <message link> <members>",
			Description: "Edit host member.",
		},
		{
			Command:     "Giveaways Edit Duration",
			Syntax:      ".giveaways edit duration <message link> <date>",
			Description: "Edit new duration (e.g. 1h, 2d).",
		},
		{
			Command:     "Giveaways Edit Min Level",
			Syntax:      ".giveaways edit minlevel <message link> <level>",
			Description: "Edit minimum level requirement to enter.",
		},
	})
}

var Giveaways = &manager.Command{
	Trigger:     "giveaways",
	Aliases:     []string{"giveaway", "gwy", "gw"},
	Name:        "giveaways",
	Description: "Manage and configure giveaways",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("giveaways")
		}

		rxMsgLink := manager.GetRxMsgLink()
		sub := strings.ToLower(ctx.Args[0])
		switch sub {
		case "start":
			if len(ctx.Args) < 5 {
				return ctx.SendHelp("giveaways")
			}
			chanID := strings.Trim(ctx.Args[1], "<#>")
			_, err := ctx.Session.Channel(chanID)
			if err != nil {
				return ctx.Reply("[!] Channel not found.")
			}
			dur, err := manager.ParseDuration(ctx.Args[2])
			if err != nil {
				return ctx.Reply("[!] Invalid duration. Use e.g. 1h, 2d, 1w.")
			}
			winnersCount, err := strconv.Atoi(ctx.Args[3])
			if err != nil || winnersCount <= 0 {
				return ctx.Reply("[!] Winners count must be a positive number.")
			}
			prize := strings.Join(ctx.Args[4:], " ")
			if prize == "" {
				return ctx.Reply("[!] Please specify a prize.")
			}

			endTime := time.Now().Add(dur)
			g := storage.Giveaway{
				GuildID:      ctx.GuildID(),
				ChannelID:    chanID,
				HostID:       ctx.AuthorID(),
				Prize:        prize,
				WinnersCount: winnersCount,
				EndTime:      endTime,
				Ended:        false,
			}

			emb := manager.BuildGiveawayEmbed(g, ctx.Cfg)
			msg, err := ctx.Session.ChannelMessageSendComplex(chanID, &discordgo.MessageSend{
				Embeds:     []*discordgo.MessageEmbed{emb},
				Components: []discordgo.MessageComponent{manager.BuildGiveawayButton("temp", false)},
			})
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to send giveaway message: %v", err))
			}

			g.MessageID = msg.ID
			comp := manager.BuildGiveawayButton(g.MessageID, false)
			_, _ = ctx.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
				Channel:    chanID,
				ID:         g.MessageID,
				Components: &[]discordgo.MessageComponent{comp},
			})

			_ = ctx.DB.SaveGiveaway(g)
			return ctx.Reply(fmt.Sprintf("[+] Giveaway started in <#%s>!", chanID))

		case "reroll":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("giveaways")
			}
			link := ctx.Args[1]
			parts := rxMsgLink.FindStringSubmatch(link)
			if len(parts) < 3 {
				return ctx.Reply("[!] Invalid message link.")
			}
			mid := parts[2]

			g, err := ctx.DB.GetGiveaway(ctx.GuildID(), mid)
			if err != nil {
				return ctx.Reply("[!] Giveaway not found in database.")
			}
			if !g.Ended {
				return ctx.Reply("[!] That giveaway is still active. End it first using `.giveaways end <link>`.")
			}

			winnersCount := g.WinnersCount
			if len(ctx.Args) > 2 {
				count, err := strconv.Atoi(ctx.Args[2])
				if err == nil && count > 0 {
					winnersCount = count
				}
			}

			winners := manager.DrawWinners(g.Entries, winnersCount)
			g.Winners = winners
			_ = ctx.DB.SaveGiveaway(g)

			emb := manager.BuildGiveawayEmbed(g, ctx.Cfg)
			comp := manager.BuildGiveawayButton(g.MessageID, true)
			_, _ = ctx.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
				Channel:    g.ChannelID,
				ID:         g.MessageID,
				Embeds:     &[]*discordgo.MessageEmbed{emb},
				Components: &[]discordgo.MessageComponent{comp},
			})

			if len(winners) > 0 {
				var winMentions []string
				for _, w := range winners {
					winMentions = append(winMentions, fmt.Sprintf("<@%s>", w))
				}
				_, _ = ctx.Session.ChannelMessageSend(g.ChannelID, fmt.Sprintf("🎉 **Reroll:** Congratulations %s, you won **%s**!", strings.Join(winMentions, ", "), g.Prize))
				return ctx.Reply("[+] Giveaway rerolled successfully.")
			} else {
				_, _ = ctx.Session.ChannelMessageSend(g.ChannelID, fmt.Sprintf("🎉 **Reroll:** No eligible winners could be drawn for **%s**.", g.Prize))
				return ctx.Reply("[*] Giveaway rerolled but no entries were found.")
			}

		case "end":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("giveaways")
			}
			link := ctx.Args[1]
			parts := rxMsgLink.FindStringSubmatch(link)
			if len(parts) < 3 {
				return ctx.Reply("[!] Invalid message link.")
			}
			mid := parts[2]

			g, err := ctx.DB.GetGiveaway(ctx.GuildID(), mid)
			if err != nil {
				return ctx.Reply("[!] Giveaway not found in database.")
			}
			if g.Ended {
				return ctx.Reply("[!] That giveaway has already ended.")
			}

			ctx.Mgr.EndGiveaway(ctx.Session, g, ctx.Cfg)
			return ctx.Reply("[+] Giveaway ended successfully.")

		case "cancel":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("giveaways")
			}
			link := ctx.Args[1]
			parts := rxMsgLink.FindStringSubmatch(link)
			if len(parts) < 3 {
				return ctx.Reply("[!] Invalid message link.")
			}
			mid := parts[2]

			g, err := ctx.DB.GetGiveaway(ctx.GuildID(), mid)
			if err != nil {
				return ctx.Reply("[!] Giveaway not found.")
			}
			if g.Ended {
				return ctx.Reply("[!] That giveaway has already ended.")
			}

			g.Ended = true
			_ = ctx.DB.SaveGiveaway(g)

			emb := manager.BuildGiveawayEmbed(g, ctx.Cfg)
			emb.Title = "❌ **GIVEAWAY CANCELLED** ❌"
			emb.Description = fmt.Sprintf("This giveaway for **%s** has been cancelled.", g.Prize)
			comp := manager.BuildGiveawayButton(g.MessageID, true)
			_, _ = ctx.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
				Channel:    g.ChannelID,
				ID:         g.MessageID,
				Embeds:     &[]*discordgo.MessageEmbed{emb},
				Components: &[]discordgo.MessageComponent{comp},
			})

			return ctx.Reply("[+] Giveaway cancelled successfully.")

		case "list":
			list, err := ctx.DB.ListActiveGiveaways(ctx.GuildID())
			if err != nil || len(list) == 0 {
				return ctx.Reply("[*] There are no active giveaways on this server.")
			}

			var sb strings.Builder
			sb.WriteString("**Active Giveaways:**\n\n")
			for i, g := range list {
				sb.WriteString(fmt.Sprintf("%d. **%s** in <#%s> (Ends: <t:%d:R>) - [Link](https://discord.com/channels/%s/%s/%s)\n",
					i+1, g.Prize, g.ChannelID, g.EndTime.Unix(), g.GuildID, g.ChannelID, g.MessageID))
			}
			return ctx.Reply(sb.String())

		case "edit":
			if len(ctx.Args) < 3 {
				return ctx.SendHelp("giveaways")
			}

			param := strings.ToLower(ctx.Args[1])
			link := ctx.Args[2]

			parts := rxMsgLink.FindStringSubmatch(link)
			if len(parts) < 3 {
				return ctx.Reply("[!] Invalid message link.")
			}
			mid := parts[2]

			g, err := ctx.DB.GetGiveaway(ctx.GuildID(), mid)
			if err != nil {
				return ctx.Reply("[!] Giveaway not found.")
			}
			if g.Ended {
				return ctx.Reply("[!] That giveaway has already ended.")
			}

			switch param {
			case "roles":
				if len(ctx.Args) < 4 {
					return ctx.Reply("[!] Please specify roles.")
				}
				var roles []string
				for _, arg := range ctx.Args[3:] {
					roles = append(roles, strings.Trim(arg, "<@&>"))
				}
				g.AwardRoles = roles

			case "image":
				url := ""
				if len(ctx.Args) > 3 {
					url = ctx.Args[3]
				} else if ctx.Message != nil && len(ctx.Message.Attachments) > 0 {
					url = ctx.Message.Attachments[0].URL
				}
				if url == "" {
					return ctx.Reply("[!] Please specify an image URL or upload an attachment.")
				}
				g.Image = url

			case "age":
				if len(ctx.Args) < 4 {
					return ctx.Reply("[!] Please specify min account age in days.")
				}
				days, err := strconv.Atoi(ctx.Args[3])
				if err != nil || days < 0 {
					return ctx.Reply("[!] Age must be a positive integer.")
				}
				g.AgeDays = days

			case "color":
				if len(ctx.Args) < 4 {
					return ctx.Reply("[!] Please specify a hex color or name.")
				}
				g.Color = ctx.Args[3]

			case "thumbnail":
				url := ""
				if len(ctx.Args) > 3 {
					url = ctx.Args[3]
				} else if ctx.Message != nil && len(ctx.Message.Attachments) > 0 {
					url = ctx.Message.Attachments[0].URL
				}
				if url == "" {
					return ctx.Reply("[!] Please specify a thumbnail URL or upload an attachment.")
				}
				g.Thumbnail = url

			case "maxlevel":
				if len(ctx.Args) < 4 {
					return ctx.Reply("[!] Please specify max level.")
				}
				lvl, err := strconv.Atoi(ctx.Args[3])
				if err != nil || lvl < 0 {
					return ctx.Reply("[!] Max level must be a positive integer.")
				}
				g.MaxLevel = lvl

			case "requiredroles":
				if len(ctx.Args) < 4 {
					return ctx.Reply("[!] Please specify required roles.")
				}
				var roles []string
				for _, arg := range ctx.Args[3:] {
					roles = append(roles, strings.Trim(arg, "<@&>"))
				}
				g.RequiredRoles = roles

			case "stay":
				if len(ctx.Args) < 4 {
					return ctx.Reply("[!] Please specify stay duration in days.")
				}
				days, err := strconv.Atoi(ctx.Args[3])
				if err != nil || days < 0 {
					return ctx.Reply("[!] Stay days must be a positive integer.")
				}
				g.StayDays = days

			case "description":
				if len(ctx.Args) < 4 {
					return ctx.Reply("[!] Please specify description text.")
				}
				g.Description = strings.Join(ctx.Args[3:], " ")

			case "prize":
				if len(ctx.Args) < 4 {
					return ctx.Reply("[!] Please specify a prize.")
				}
				g.Prize = strings.Join(ctx.Args[3:], " ")

			case "winners":
				if len(ctx.Args) < 4 {
					return ctx.Reply("[!] Please specify winners count.")
				}
				count, err := strconv.Atoi(ctx.Args[3])
				if err != nil || count <= 0 {
					return ctx.Reply("[!] Winners count must be a positive integer.")
				}
				g.WinnersCount = count

			case "host":
				if len(ctx.Args) < 4 {
					return ctx.Reply("[!] Please specify a host member.")
				}
				hostID := strings.Trim(ctx.Args[3], "<@!>")
				g.HostID = hostID

			case "duration":
				if len(ctx.Args) < 4 {
					return ctx.Reply("[!] Please specify a new duration.")
				}
				dur, err := manager.ParseDuration(ctx.Args[3])
				if err != nil {
					return ctx.Reply("[!] Invalid duration format. Use e.g. 1h, 2d, 1w.")
				}
				g.EndTime = time.Now().Add(dur)

			case "minlevel":
				if len(ctx.Args) < 4 {
					return ctx.Reply("[!] Please specify min level.")
				}
				lvl, err := strconv.Atoi(ctx.Args[3])
				if err != nil || lvl < 0 {
					return ctx.Reply("[!] Min level must be a positive integer.")
				}
				g.MinLevel = lvl

			default:
				return ctx.Reply(fmt.Sprintf("[!] Unknown parameter %q to edit.", param))
			}

			_ = ctx.DB.SaveGiveaway(g)

			emb := manager.BuildGiveawayEmbed(g, ctx.Cfg)
			comp := manager.BuildGiveawayButton(g.MessageID, g.Ended)
			_, _ = ctx.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
				Channel:    g.ChannelID,
				ID:         g.MessageID,
				Embeds:     &[]*discordgo.MessageEmbed{emb},
				Components: &[]discordgo.MessageComponent{comp},
			})

			return ctx.Reply(fmt.Sprintf("[+] Giveaway parameter `%s` updated.", param))

		default:
			return ctx.SendHelp("giveaways")
		}
	},
}
