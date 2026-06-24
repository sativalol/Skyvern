package utility
import (
	"fmt"
	"regexp"
	"sort"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/storage"
	"strconv"
	"strings"
	"github.com/bwmarrin/discordgo"
)
func init() {
	manager.RegisterHelp("clownboard", []manager.HelpPage{
		{
			Command:     "Clownboard Showcase",
			Syntax:      ".clownboard",
			Description: "Showcase the worst/funniest messages in the server.",
		},
		{
			Command:     "Clownboard Threshold",
			Syntax:      ".clownboard threshold <threshold>",
			Description: "Sets the reaction count needed for messages to reach the clownboard.",
		},
		{
			Command:     "Clownboard Set Log Channel",
			Syntax:      ".clownboard set <channel>",
			Description: "Sets the channel where clownboard posts will be logged.",
		},
		{
			Command:     "Clownboard Unlock",
			Syntax:      ".clownboard unlock",
			Description: "Unlock/enable the clownboard system.",
		},
		{
			Command:     "Clownboard Lock",
			Syntax:      ".clownboard lock",
			Description: "Lock/disable the clownboard system.",
		},
		{
			Command:     "Clownboard Ignore Toggle",
			Syntax:      ".clownboard ignore <channel|member|role>",
			Description: "Toggle ignore status on a channel, member, or role.",
		},
		{
			Command:     "Clownboard Ignore List",
			Syntax:      ".clownboard ignore list",
			Description: "View list of all ignored channels, members, and roles.",
		},
		{
			Command:     "Clownboard SelfStar",
			Syntax:      ".clownboard selfstar <on|off>",
			Description: "Configure whether message authors can clown/star their own posts.",
		},
		{
			Command:     "Clownboard Timestamp",
			Syntax:      ".clownboard timestamp <on|off>",
			Description: "Toggle showing timestamps on clownboard posts.",
		},
		{
			Command:     "Clownboard Attachments",
			Syntax:      ".clownboard attachments <on|off>",
			Description: "Toggle showing attachments on clownboard posts.",
		},
		{
			Command:     "Clownboard JumpURL",
			Syntax:      ".clownboard jumpurl <on|off>",
			Description: "Toggle showing jump links on clownboard posts.",
		},
		{
			Command:     "Clownboard Emoji",
			Syntax:      ".clownboard emoji <emoji>",
			Description: "Configure the reaction emoji trigger (default is clown emoji).",
		},
		{
			Command:     "Clownboard Color",
			Syntax:      ".clownboard color <color>",
			Description: "Configure the embed color.",
		},
		{
			Command:     "Clownboard Configuration",
			Syntax:      ".clownboard config",
			Description: "View the current clownboard configurations.",
		},
		{
			Command:     "Clownboard Reset",
			Syntax:      ".clownboard reset",
			Description: "Reset all clownboard configurations to defaults.",
		},
	})
}
var rxClownChan = regexp.MustCompile(`^<#(\d+)>$`)
func parseBoolSetting(val string) (bool, error) {
	val = strings.ToLower(val)
	if val == "on" || val == "true" || val == "yes" || val == "enable" {
		return true, nil
	}
	if val == "off" || val == "false" || val == "no" || val == "disable" {
		return false, nil
	}
	return false, fmt.Errorf("invalid setting")
}
func cleanEmoji(input string) string {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "<") && strings.HasSuffix(input, ">") {
		input = strings.TrimPrefix(input, "<")
		input = strings.TrimSuffix(input, ">")
		if strings.HasPrefix(input, "a:") {
			input = strings.TrimPrefix(input, "a:")
		} else if strings.HasPrefix(input, ":") {
			input = strings.TrimPrefix(input, ":")
		}
	}
	return input
}
func toggleSlice(slice []string, val string) []string {
	for i, item := range slice {
		if item == val {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return append(slice, val)
}
func fmtState(b bool) string {
	if b {
		return "`On`"
	}
	return "`Off`"
}
var Clownboard = &manager.Command{
	Trigger:     "clownboard",
	Aliases:     []string{"cb"},
	Name:        "clownboard",
	Description: "Configure server clownboard showcase",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()
		if len(ctx.Args) == 0 {
			posts, err := ctx.DB.ListClownPosts(gid)
			if err != nil || len(posts) == 0 {
				return ctx.Reply(fmt.Sprintf("%s No messages have reached the clownboard yet.", ctx.SuccessEmoji()))
			}
			sort.Slice(posts, func(i, j int) bool {
				return posts[i].Count > posts[j].Count
			})
			var sb strings.Builder
			limit := len(posts)
			if limit > 10 {
				limit = 10
			}
			for i := 0; i < limit; i++ {
				p := posts[i]
				text := p.Text
				if len(text) > 50 {
					text = text[:47] + "..."
				}
				sb.WriteString(fmt.Sprintf("%d. [Jump to Post](https://discord.com/channels/%s/%s/%s) - **%d** clowns | <@%s>: %s\n",
					i+1, gid, p.ChanID, p.OrigID, p.Count, p.AuthorID, text))
			}
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:       "Server Clownboard Showcase",
				Description: sb.String(),
			})
			return ctx.Respond(emb)
		}
		sub := strings.ToLower(ctx.Args[0])
		cfg, err := ctx.DB.GetClownboardCfg(gid)
		if err != nil {
			cfg = storage.ClownboardCfg{
				Enabled:     false,
				Threshold:   3,
				Emoji:       "🤡",
				SelfStar:    false,
				Timestamp:   true,
				Attachments: true,
				JumpURL:     true,
				Color:       0xffa500,
			}
		}
		if sub == "config" {
			emojiDisp := cfg.Emoji
			if strings.Contains(emojiDisp, ":") {
				emojiDisp = "<:" + emojiDisp + ">"
			}
			desc := fmt.Sprintf(
				"**Status:** %s\n"+
					"**Channel:** <#%s>\n"+
					"**Threshold:** `%d` reactions\n"+
					"**Emoji:** %s\n"+
					"**Self Star:** %s\n"+
					"**Timestamp:** %s\n"+
					"**Attachments:** %s\n"+
					"**Jump URL:** %s\n"+
					"**Color:** `0x%X`",
				fmtState(cfg.Enabled), cfg.ChannelID, cfg.Threshold, emojiDisp,
				fmtState(cfg.SelfStar), fmtState(cfg.Timestamp), fmtState(cfg.Attachments), fmtState(cfg.JumpURL),
				cfg.Color,
			)
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:       "Clownboard Configurations",
				Description: desc,
			})
			return ctx.Respond(emb)
		}
		if sub == "ignore" && len(ctx.Args) >= 2 && strings.ToLower(ctx.Args[1]) == "list" {
			var sb strings.Builder
			sb.WriteString("**Ignored Channels:**\n")
			if len(cfg.IgnoredChannels) == 0 {
				sb.WriteString("- None\n")
			} else {
				for _, cid := range cfg.IgnoredChannels {
					sb.WriteString(fmt.Sprintf("- <#%s> (`%s`)\n", cid, cid))
				}
			}
			sb.WriteString("\n**Ignored Members:**\n")
			if len(cfg.IgnoredMembers) == 0 {
				sb.WriteString("- None\n")
			} else {
				for _, uid := range cfg.IgnoredMembers {
					sb.WriteString(fmt.Sprintf("- <@%s> (`%s`)\n", uid, uid))
				}
			}
			sb.WriteString("\n**Ignored Roles:**\n")
			if len(cfg.IgnoredRoles) == 0 {
				sb.WriteString("- None\n")
			} else {
				for _, rid := range cfg.IgnoredRoles {
					sb.WriteString(fmt.Sprintf("- <@&%s> (`%s`)\n", rid, rid))
				}
			}
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:       "Clownboard Ignored Targets",
				Description: sb.String(),
			})
			return ctx.Respond(emb)
		}
		p, err := ctx.Session.UserChannelPermissions(ctx.AuthorID(), ctx.ChanID())
		if err != nil || (p&discordgo.PermissionManageGuild) == 0 {
			return ctx.Reply(fmt.Sprintf("%s You need Manage Guild permission to configure the clownboard.", ctx.ErrorEmoji()))
		}
		switch sub {
		case "threshold", "limit":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("clownboard")
			}
			count, err := strconv.Atoi(ctx.Args[1])
			if err != nil || count <= 0 {
				return ctx.Reply(fmt.Sprintf("%s Invalid threshold count. Must be a positive number.", ctx.ErrorEmoji()))
			}
			cfg.Threshold = count
			_ = ctx.DB.SaveClownboardCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Clownboard reaction threshold set to %d.", ctx.SuccessEmoji(), count))
		case "unlock", "enable":
			if cfg.ChannelID == "" {
				return ctx.Reply(fmt.Sprintf("%s Please configure a log channel first using `.clownboard set <#channel>`.", ctx.ErrorEmoji()))
			}
			cfg.Enabled = true
			_ = ctx.DB.SaveClownboardCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Clownboard unlocked and enabled.", ctx.SuccessEmoji()))
		case "lock", "disable":
			cfg.Enabled = false
			_ = ctx.DB.SaveClownboardCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Clownboard locked and disabled.", ctx.SuccessEmoji()))
		case "set", "channel":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("clownboard")
			}
			chanArg := ctx.Args[1]
			cid := ""
			if m := rxClownChan.FindStringSubmatch(chanArg); len(m) > 1 {
				cid = m[1]
			} else {
				cid = chanArg
			}
			ch, err := ctx.Session.Channel(cid)
			if err != nil || ch.GuildID != gid {
				return ctx.Reply(fmt.Sprintf("%s Invalid text channel.", ctx.ErrorEmoji()))
			}
			cfg.ChannelID = cid
			_ = ctx.DB.SaveClownboardCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Clownboard channel set to <#%s>.", ctx.SuccessEmoji(), cid))
			cleaned := cleanEmoji(ctx.Args[1])
			cfg.Emoji = cleaned
			_ = ctx.DB.SaveClownboardCfg(gid, cfg)
			emojiDisp := cleaned
			if strings.Contains(emojiDisp, ":") {
				emojiDisp = "<:" + emojiDisp + ">"
			}
			return ctx.Reply(fmt.Sprintf("%s Clownboard emoji set to %s.", ctx.SuccessEmoji(), emojiDisp))
		case "selfstar", "selfclown":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("clownboard")
			}
			val, err := parseBoolSetting(ctx.Args[1])
			if err != nil {
				return ctx.Reply(fmt.Sprintf("%s Invalid setting. Use `on` or `off`.", ctx.ErrorEmoji()))
			}
			cfg.SelfStar = val
			_ = ctx.DB.SaveClownboardCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Clownboard selfstar option set to %s.", ctx.SuccessEmoji(), fmtState(val)))
		case "timestamp":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("clownboard")
			}
			val, err := parseBoolSetting(ctx.Args[1])
			if err != nil {
				return ctx.Reply(fmt.Sprintf("%s Invalid setting. Use `on` or `off`.", ctx.ErrorEmoji()))
			}
			cfg.Timestamp = val
			_ = ctx.DB.SaveClownboardCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Clownboard timestamp option set to %s.", ctx.SuccessEmoji(), fmtState(val)))
		case "attachments":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("clownboard")
			}
			val, err := parseBoolSetting(ctx.Args[1])
			if err != nil {
				return ctx.Reply(fmt.Sprintf("%s Invalid setting. Use `on` or `off`.", ctx.ErrorEmoji()))
			}
			cfg.Attachments = val
			_ = ctx.DB.SaveClownboardCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Clownboard attachments option set to %s.", ctx.SuccessEmoji(), fmtState(val)))
		case "jumpurl", "jumplink":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("clownboard")
			}
			val, err := parseBoolSetting(ctx.Args[1])
			if err != nil {
				return ctx.Reply(fmt.Sprintf("%s Invalid setting. Use `on` or `off`.", ctx.ErrorEmoji()))
			}
			cfg.JumpURL = val
			_ = ctx.DB.SaveClownboardCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Clownboard jumpurl option set to %s.", ctx.SuccessEmoji(), fmtState(val)))
		case "color":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("clownboard")
			}
			colInt := manager.ParseColor(ctx.Args[1])
			cfg.Color = colInt
			_ = ctx.DB.SaveClownboardCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Clownboard embed color set to `0x%X`.", ctx.SuccessEmoji(), colInt))
		case "ignore":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("clownboard")
			}
			arg := ctx.Args[1]
			if strings.HasPrefix(arg, "<#") && strings.HasSuffix(arg, ">") {
				cid := strings.Trim(arg, "<#>")
				ch, err := ctx.Session.Channel(cid)
				if err != nil || ch.GuildID != gid {
					return ctx.Reply(fmt.Sprintf("%s Invalid channel.", ctx.ErrorEmoji()))
				}
				cfg.IgnoredChannels = toggleSlice(cfg.IgnoredChannels, cid)
				_ = ctx.DB.SaveClownboardCfg(gid, cfg)
				return ctx.Reply(fmt.Sprintf("%s Channel <#%s> ignore state toggled.", ctx.SuccessEmoji(), cid))
			} else if strings.HasPrefix(arg, "<@&") && strings.HasSuffix(arg, ">") {
				rid := strings.Trim(arg, "<@&>")
				roles, err := ctx.Session.GuildRoles(gid)
				found := false
				if err == nil {
					for _, r := range roles {
						if r.ID == rid {
							found = true
							break
						}
					}
				}
				if !found {
					return ctx.Reply(fmt.Sprintf("%s Invalid role.", ctx.ErrorEmoji()))
				}
				cfg.IgnoredRoles = toggleSlice(cfg.IgnoredRoles, rid)
				_ = ctx.DB.SaveClownboardCfg(gid, cfg)
				return ctx.Reply(fmt.Sprintf("%s Role <@&%s> ignore state toggled.", ctx.SuccessEmoji(), rid))
			} else if (strings.HasPrefix(arg, "<@") || strings.HasPrefix(arg, "<@!")) && strings.HasSuffix(arg, ">") {
				uid := strings.Trim(arg, "<@!>")
				_, err := ctx.Session.GuildMember(gid, uid)
				if err != nil {
					return ctx.Reply(fmt.Sprintf("%s Invalid member.", ctx.ErrorEmoji()))
				}
				cfg.IgnoredMembers = toggleSlice(cfg.IgnoredMembers, uid)
				_ = ctx.DB.SaveClownboardCfg(gid, cfg)
				return ctx.Reply(fmt.Sprintf("%s Member <@%s> ignore state toggled.", ctx.SuccessEmoji(), uid))
			} else {
				if ch, err := ctx.Session.Channel(arg); err == nil && ch.GuildID == gid {
					cfg.IgnoredChannels = toggleSlice(cfg.IgnoredChannels, arg)
					_ = ctx.DB.SaveClownboardCfg(gid, cfg)
					return ctx.Reply(fmt.Sprintf("%s Channel <#%s> ignore state toggled.", ctx.SuccessEmoji(), arg))
				}
				if _, err := ctx.Session.GuildMember(gid, arg); err == nil {
					cfg.IgnoredMembers = toggleSlice(cfg.IgnoredMembers, arg)
					_ = ctx.DB.SaveClownboardCfg(gid, cfg)
					return ctx.Reply(fmt.Sprintf("%s Member <@%s> ignore state toggled.", ctx.SuccessEmoji(), arg))
				}
				roles, err := ctx.Session.GuildRoles(gid)
				if err == nil {
					for _, r := range roles {
						if r.ID == arg {
							cfg.IgnoredRoles = toggleSlice(cfg.IgnoredRoles, arg)
							_ = ctx.DB.SaveClownboardCfg(gid, cfg)
							return ctx.Reply(fmt.Sprintf("%s Role <@&%s> ignore state toggled.", ctx.SuccessEmoji(), arg))
						}
					}
				}
				return ctx.Reply(fmt.Sprintf("%s Could not resolve target as a valid channel, member, or role.", ctx.ErrorEmoji()))
			}
		case "reset":
			cfg = storage.ClownboardCfg{
				Enabled:     false,
				Threshold:   3,
				Emoji:       "🤡",
				SelfStar:    false,
				Timestamp:   true,
				Attachments: true,
				JumpURL:     true,
				Color:       0xffa500,
			}
			_ = ctx.DB.SaveClownboardCfg(gid, cfg)
			return ctx.Reply(fmt.Sprintf("%s Clownboard configuration reset to defaults.", ctx.SuccessEmoji()))
		default:
			return ctx.SendHelp("clownboard")
		}
	},
}