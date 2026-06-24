package general

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"skyvern/internal/config"
	"skyvern/internal/manager"
)

func isSlashCommand(name string) bool {
	return name == "quote" || name == "impersonate"
}

func getCatEmoji(s *discordgo.Session, mgr *manager.Manager, gid string, cat string) string {
	emoji := mgr.ResolveEmoji(s, gid, "sys_"+strings.ToLower(cat))
	if emoji != "" {
		return emoji
	}
	switch strings.ToLower(cat) {
	case "general":
		return "💬"
	case "moderation":
		return "🛡️"
	case "fun":
		return "🎉"
	case "music":
		return "🎵"
	case "utility":
		return "🔧"
	case "tickets":
		return "🎫"
	default:
		return "📁"
	}
}

func getComponentEmoji(s *discordgo.Session, mgr *manager.Manager, gid string, cat string) discordgo.ComponentEmoji {
	emojiStr := mgr.ResolveEmoji(s, gid, "sys_"+strings.ToLower(cat))
	if emojiStr != "" && strings.HasPrefix(emojiStr, "<:") && strings.HasSuffix(emojiStr, ">") {
		inner := strings.TrimSuffix(strings.TrimPrefix(emojiStr, "<:"), ">")
		parts := strings.Split(inner, ":")
		if len(parts) == 2 {
			return discordgo.ComponentEmoji{
				Name: parts[0],
				ID:   parts[1],
			}
		}
	}

	unicode := "📁"
	switch strings.ToLower(cat) {
	case "general":
		unicode = "💬"
	case "moderation":
		unicode = "🛡️"
	case "fun":
		unicode = "🎉"
	case "music":
		unicode = "🎵"
	case "utility":
		unicode = "🔧"
	case "tickets":
		unicode = "🎫"
	}
	return discordgo.ComponentEmoji{
		Name: unicode,
	}
}

var Help = &manager.Command{
	Trigger:     "help",
	Name:        "help",
	Description: "List all available commands grouped by category",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) > 0 {
			cmdName := strings.ToLower(ctx.Args[0])
			if _, ok := manager.GetHelp(cmdName); ok {
				return ctx.SendHelp(cmdName)
			}
			if cmd := ctx.Mgr.FindCommand(cmdName); cmd != nil {
				pages := []manager.HelpPage{
					{
						Command:     strings.Title(cmd.Name),
						Syntax:      fmt.Sprintf("%s%s", ctx.Cfg.Prefix, cmd.Trigger),
						Description: cmd.Description,
					},
				}
				e := manager.BuildHelpEmbed(ctx.Cfg, cmd.Name, pages, 0)
				return ctx.Respond(e)
			}
			return ctx.Reply(fmt.Sprintf("[!] Command `%s` not found.", cmdName))
		}

		cMap := groupCmds(ctx.Mgr.Commands())
		totalPrefix := 0
		totalSlash := 0
		for _, cmd := range ctx.Mgr.Commands() {
			if isSlashCommand(cmd.Trigger) {
				totalSlash++
			} else {
				totalPrefix++
			}
		}

		e := buildHomeHelp(ctx.Cfg, totalPrefix, totalSlash, ctx.Cfg.Prefix)
		comps := buildComps(cMap, "", 0, "prefix", ctx.Session, ctx.Mgr, ctx.GuildID())

		if ctx.Interact != nil {
			return ctx.Session.InteractionRespond(ctx.Interact, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds:     []*discordgo.MessageEmbed{e},
					Components: comps,
				},
			})
		}
		_, err := ctx.Session.ChannelMessageSendComplex(ctx.ChanID(), &discordgo.MessageSend{
			Embeds:     []*discordgo.MessageEmbed{e},
			Components: comps,
		})
		return err
	},
}

func groupCmds(cmds []*manager.Command) map[string][]*manager.Command {
	cMap := make(map[string][]*manager.Command)
	for _, cmd := range cmds {
		c := strings.ToLower(cmd.Category)
		if c == "" {
			c = "other"
		}
		if c == "owner" || c == "Owner" {
			continue
		}
		cMap[c] = append(cMap[c], cmd)
	}
	return cMap
}

func buildHomeHelp(cfg config.ResCfg, totalPrefix, totalSlash int, prefix string) *discordgo.MessageEmbed {
	e := config.Build(cfg, config.EmbedOpt{
		Title:       cfg.Name + " Help Navigator",
		Description: fmt.Sprintf("Welcome to %s. You can find more information about us [here](https://%s/).", cfg.Name, cfg.Footer),
	})
	e.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: cfg.AvatarURL}

	e.Fields = append(e.Fields, config.Field(
		"Search & Info",
		fmt.Sprintf("Prefix: `%s`\nSpecific Command: `%shelp <command>`\nWebsite: [%s](https://%s/commands)", prefix, prefix, cfg.Footer, cfg.Footer),
		false,
	))
	e.Fields = append(e.Fields, config.Field(
		"Statistics",
		fmt.Sprintf("Total: %d commands (%d Prefix | %d Slash)", totalPrefix+totalSlash, totalPrefix, totalSlash),
		false,
	))
	return e
}

func buildCategoryHelp(cfg config.ResCfg, cmds []*manager.Command, cat string, page int, prefix string, viewType string, s *discordgo.Session, mgr *manager.Manager, gid string) *discordgo.MessageEmbed {
	commandsPerPage := 6
	var filtered []*manager.Command
	for _, cmd := range cmds {
		isSlash := isSlashCommand(cmd.Trigger)
		if viewType == "slash" && isSlash {
			filtered = append(filtered, cmd)
		} else if viewType == "prefix" && !isSlash {
			filtered = append(filtered, cmd)
		}
	}

	totalPages := (len(filtered) + commandsPerPage - 1) / commandsPerPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	displayCat := strings.ToUpper(cat)
	emoji := getCatEmoji(s, mgr, gid, cat)
	if emoji != "" {
		displayCat = emoji + " " + displayCat
	}

	e := config.Build(cfg, config.EmbedOpt{
		Title: displayCat,
	})

	start := page * commandsPerPage
	end := start + commandsPerPage
	if end > len(filtered) {
		end = len(filtered)
	}

	var lines []string
	if len(filtered) > 0 {
		for _, cmd := range filtered[start:end] {
			prefixChar := prefix
			if viewType == "slash" {
				prefixChar = "/"
			}
			lines = append(lines, fmt.Sprintf("**%s%s**\n%s", prefixChar, cmd.Trigger, cmd.Description))
		}
		e.Description = strings.Join(lines, "\n\n")
	} else {
		e.Description = fmt.Sprintf("No %s commands found in this category.", viewType)
	}

	e.Footer = &discordgo.MessageEmbedFooter{
		Text:    fmt.Sprintf("Page %d of %d • %s", page+1, totalPages, cfg.Footer),
		IconURL: cfg.FooterIcon,
	}
	return e
}

func buildComps(cMap map[string][]*manager.Command, activeCat string, page int, viewType string, s *discordgo.Session, mgr *manager.Manager, gid string) []discordgo.MessageComponent {
	var opts []discordgo.SelectMenuOption
	var cats []string
	for c := range cMap {
		cats = append(cats, c)
	}

	for _, c := range cats {
		compEmoji := getComponentEmoji(s, mgr, gid, c)
		opts = append(opts, discordgo.SelectMenuOption{
			Label:       strings.Title(c),
			Value:       c,
			Description: fmt.Sprintf("Show %s commands", c),
			Emoji:       &compEmoji,
			Default:     c == activeCat,
		})
	}

	selectMenu := discordgo.SelectMenu{
		CustomID:    fmt.Sprintf("help_select:%s", viewType),
		Placeholder: "Choose a category to navigate",
		Options:     opts,
	}

	prevDisabled := true
	nextDisabled := true

	if activeCat != "" {
		cmds := cMap[activeCat]
		var filtered []*manager.Command
		for _, cmd := range cmds {
			isSlash := isSlashCommand(cmd.Trigger)
			if viewType == "slash" && isSlash {
				filtered = append(filtered, cmd)
			} else if viewType == "prefix" && !isSlash {
				filtered = append(filtered, cmd)
			}
		}
		totalPages := (len(filtered) + 5) / 6
		if totalPages < 1 {
			totalPages = 1
		}
		prevDisabled = page <= 0
		nextDisabled = page >= totalPages-1
	}

	prevButton := discordgo.Button{
		Label:    "Previous",
		Style:    discordgo.SecondaryButton,
		CustomID: fmt.Sprintf("help_prev:%s:%d:%s", activeCat, page, viewType),
		Disabled: prevDisabled,
	}

	toggleLabel := "Switch to Slash"
	if viewType == "slash" {
		toggleLabel = "Switch to Prefix"
	}
	toggleButton := discordgo.Button{
		Label:    toggleLabel,
		Style:    discordgo.SecondaryButton,
		CustomID: fmt.Sprintf("help_toggle:%s:%d:%s", activeCat, page, viewType),
		Disabled: activeCat == "",
	}

	nextButton := discordgo.Button{
		Label:    "Next",
		Style:    discordgo.SecondaryButton,
		CustomID: fmt.Sprintf("help_next:%s:%d:%s", activeCat, page, viewType),
		Disabled: nextDisabled,
	}

	dmButton := discordgo.Button{
		Label:    "Send Full List",
		Style:    discordgo.SecondaryButton,
		CustomID: fmt.Sprintf("help_dm:%s:%d:%s", activeCat, page, viewType),
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{selectMenu},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{prevButton, toggleButton, nextButton},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{dmButton},
		},
	}
}

func HandleHelpComponent(s *discordgo.Session, i *discordgo.InteractionCreate, mgr *manager.Manager) {
	data := i.MessageComponentData()
	parts := strings.Split(data.CustomID, ":")
	action := parts[0]

	viewType := "prefix"
	activeCat := ""
	page := 0

	if action == "help_select" {
		if len(parts) > 1 {
			viewType = parts[1]
		}
		if len(data.Values) > 0 {
			activeCat = data.Values[0]
		}
	} else {
		if len(parts) >= 4 {
			activeCat = parts[1]
			_, _ = fmt.Sscanf(parts[2], "%d", &page)
			viewType = parts[3]
		}
	}

	if action == "help_prev" {
		page--
	} else if action == "help_next" {
		page++
	} else if action == "help_toggle" {
		if viewType == "prefix" {
			viewType = "slash"
		} else {
			viewType = "prefix"
		}
		page = 0
	}

	inst, ok := mgr.ResolvedCfgFor(s.State.User.ID)
	if !ok {
		inst = config.Resolve(config.GetGlobal(), config.BotInst{})
	}
	cMap := groupCmds(mgr.Commands())

	if action == "help_dm" {
		dmChan, err := s.UserChannelCreate(i.Member.User.ID)
		if err != nil {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "[!] I couldn't DM you. Please enable your direct messages.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		emb := config.Build(inst, config.EmbedOpt{
			Title:       inst.Name + " - Full Command List",
			Description: "Here is a list of all commands available in the bot:",
		})
		for cat, cmds := range cMap {
			var lines []string
			for _, cmd := range cmds {
				lines = append(lines, fmt.Sprintf("`%s`", cmd.Trigger))
			}
			emb.Fields = append(emb.Fields, config.Field(
				strings.Title(cat),
				strings.Join(lines, ", "),
				false,
			))
		}
		_, _ = s.ChannelMessageSendEmbed(dmChan.ID, emb)

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "[+] Sent the full command list to your DMs!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var e *discordgo.MessageEmbed
	if activeCat == "" {
		totalPrefix := 0
		totalSlash := 0
		for _, cmd := range mgr.Commands() {
			if isSlashCommand(cmd.Trigger) {
				totalSlash++
			} else {
				totalPrefix++
			}
		}
		e = buildHomeHelp(inst, totalPrefix, totalSlash, inst.Prefix)
	} else {
		e = buildCategoryHelp(inst, cMap[activeCat], activeCat, page, inst.Prefix, viewType, s, mgr, i.GuildID)
	}

	comps := buildComps(cMap, activeCat, page, viewType, s, mgr, i.GuildID)

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{e},
			Components: comps,
		},
	})
}