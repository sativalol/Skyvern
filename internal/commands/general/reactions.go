package general
import (
	"fmt"
	"regexp"
	"skyvern/internal/manager"
	"strings"
	"github.com/bwmarrin/discordgo"
)
var rxMsgLink = regexp.MustCompile(`channels/(?:\d+|@me)/(\d+)/(\d+)`)
var rxEmoji = regexp.MustCompile(`<a?:([a-zA-Z0-9_]+):(\d+)>`)
func resolveReactionEmoji(s string) string {
	m := rxEmoji.FindStringSubmatch(s)
	if len(m) > 2 {
		return m[1] + ":" + m[2]
	}
	return s
}
func init() {
	manager.RegisterHelp("react", []manager.HelpPage{
		{
			Command:     "React",
			Syntax:      ".react <message link> <emoji>",
			Description: "React to a message with one or more emojis.",
		},
	})
	manager.RegisterHelp("reaction", []manager.HelpPage{
		{
			Command:     "Reaction AutoReact",
			Syntax:      ".reaction <subcommand>",
			Description: "Configure word-based reactions.",
		},
	})
	manager.RegisterHelp("previousreact", []manager.HelpPage{
		{
			Command:     "Previous Message AutoReact",
			Syntax:      ".previousreact <subcommand>",
			Description: "Configure reactions that target the previous message in the channel.",
		},
	})
	manager.RegisterHelp("noselfreact", []manager.HelpPage{
		{
			Command:     "No Self React",
			Syntax:      ".noselfreact <subcommand>",
			Description: "Prevent self reactions on messages.",
		},
	})
}
var React = &manager.Command{
	Trigger:     "react",
	Name:        "react",
	Description: "Add a reaction(s) to a message",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) < 2 {
			return ctx.SendHelp("react")
		}
		parts := rxMsgLink.FindStringSubmatch(ctx.Args[0])
		if len(parts) < 3 {
			return ctx.SendText("invalid message link format")
		}
		cid := parts[1]
		mid := parts[2]
		for _, raw := range ctx.Args[1:] {
			emoji := resolveReactionEmoji(raw)
			_ = ctx.Session.MessageReactionAdd(cid, mid, emoji)
		}
		return ctx.SendText("added reactions")
	},
}
func handleGenericReact(
	ctx *manager.CommandContext,
	cmdName string,
	saveFn func(string, string, string, string) error,
	deleteFn func(string, string, string) error,
	deleteAllFn func(string, string) error,
	clearFn func(string) error,
	getOwnerFn func(string, string) (string, error),
	listFn func(string) (map[string][]string, error),
) error {
	if len(ctx.Args) == 0 {
		return ctx.SendHelp(cmdName)
	}
	sub := strings.ToLower(ctx.Args[0])
	gid := ctx.GuildID()
	switch sub {
	case "add":
		if len(ctx.Args) < 3 {
			return ctx.SendHelp(cmdName)
		}
		emoji := resolveReactionEmoji(ctx.Args[1])
		trigger := strings.ToLower(strings.Join(ctx.Args[2:], " "))
		_ = saveFn(gid, trigger, emoji, ctx.AuthorID())
		return ctx.SendText(fmt.Sprintf("added %s trigger: `%s` -> %s", cmdName, trigger, emoji))
	case "delete", "remove":
		if len(ctx.Args) < 3 {
			return ctx.SendHelp(cmdName)
		}
		emoji := resolveReactionEmoji(ctx.Args[1])
		trigger := strings.ToLower(strings.Join(ctx.Args[2:], " "))
		_ = deleteFn(gid, trigger, emoji)
		return ctx.SendText(fmt.Sprintf("removed %s trigger: `%s` -> %s", cmdName, trigger, emoji))
	case "deleteall":
		if len(ctx.Args) < 2 {
			return ctx.SendHelp(cmdName)
		}
		trigger := strings.ToLower(strings.Join(ctx.Args[1:], " "))
		_ = deleteAllFn(gid, trigger)
		return ctx.SendText(fmt.Sprintf("removed all %s triggers for `%s`", cmdName, trigger))
	case "clear":
		_ = clearFn(gid)
		return ctx.SendText(fmt.Sprintf("cleared all %s triggers", cmdName))
	case "owner":
		if len(ctx.Args) < 2 {
			return ctx.SendHelp(cmdName)
		}
		trigger := strings.ToLower(strings.Join(ctx.Args[1:], " "))
		owner, err := getOwnerFn(gid, trigger)
		if err != nil || owner == "" {
			return ctx.SendText(fmt.Sprintf("no %s trigger found", cmdName))
		}
		return ctx.SendText(fmt.Sprintf("owner of %s trigger `%s` is <@%s>", cmdName, trigger, owner))
	case "list":
		m, err := listFn(gid)
		if err != nil || len(m) == 0 {
			return ctx.SendText(fmt.Sprintf("no %s triggers configured", cmdName))
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%s triggers:\n", cmdName))
		for trigger, emojis := range m {
			sb.WriteString(fmt.Sprintf("- `%s`: %s\n", trigger, strings.Join(emojis, ", ")))
		}
		return ctx.SendText(sb.String())
	}
	return nil
}
var Reaction = &manager.Command{
	Trigger:     "reaction",
	Aliases:     []string{"reacts"},
	Name:        "reaction",
	Description: "Word reaction trigger configuration",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) > 0 && strings.ToLower(ctx.Args[0]) == "messages" {
			gid := ctx.GuildID()
			if len(ctx.Args) > 1 && strings.ToLower(ctx.Args[1]) == "list" {
				m, err := ctx.DB.ListChannelAutoReacts(gid)
				if err != nil || len(m) == 0 {
					return ctx.SendText("no channel auto reactions configured")
				}
				var sb strings.Builder
				sb.WriteString("channel auto reactions:\n")
				for cid, emojis := range m {
					sb.WriteString(fmt.Sprintf("<#%s>: %s\n", cid, strings.Join(emojis, ", ")))
				}
				return ctx.SendText(sb.String())
			}
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("reaction")
			}
			cid := strings.Trim(ctx.Args[1], "<#>")
			if len(ctx.Args) == 2 {
				_ = ctx.DB.DeleteChannelAutoReacts(gid, cid)
				return ctx.SendText(fmt.Sprintf("removed auto reactions for <#%s>", cid))
			}
			var emojis []string
			for _, raw := range ctx.Args[2:] {
				if len(emojis) >= 3 {
					break
				}
				emojis = append(emojis, resolveReactionEmoji(raw))
			}
			_ = ctx.DB.SaveChannelAutoReacts(gid, cid, emojis)
			return ctx.SendText(fmt.Sprintf("configured auto reactions for <#%s>: %s", cid, strings.Join(emojis, ", ")))
		}
		return handleGenericReact(
			ctx,
			"reaction",
			ctx.DB.SaveReactionTrigger,
			ctx.DB.DeleteReactionTrigger,
			ctx.DB.DeleteAllReactionTriggers,
			ctx.DB.ClearReactionTriggers,
			ctx.DB.GetReactionTriggerOwner,
			ctx.DB.ListReactionTriggers,
		)
	},
}
var PreviousReact = &manager.Command{
	Trigger:     "previousreact",
	Aliases:     []string{"prevreact"},
	Name:        "previousreact",
	Description: "Word reaction trigger targeting the previous message",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		return handleGenericReact(
			ctx,
			"previousreact",
			ctx.DB.SavePrevReactTrigger,
			ctx.DB.DeletePrevReactTrigger,
			ctx.DB.DeleteAllPrevReactTriggers,
			ctx.DB.ClearPrevReactTriggers,
			ctx.DB.GetPrevReactTriggerOwner,
			ctx.DB.ListPrevReactTriggers,
		)
	},
}
var NoSelfReact = &manager.Command{
	Trigger:     "noselfreact",
	Name:        "noselfreact",
	Description: "Prevent self reactions on messages",
	Category:    "moderation",
	Execute: func(ctx *manager.CommandContext) error {
		p, err := ctx.Session.UserChannelPermissions(ctx.AuthorID(), ctx.ChanID())
		if err != nil || (p&discordgo.PermissionAdministrator) == 0 {
			return ctx.SendText("you need Administrator permissions to use this command")
		}
		gid := ctx.GuildID()
		cfg, _ := ctx.DB.GetNoSelfReactCfg(gid)
		if len(ctx.Args) == 0 {
			var sb strings.Builder
			sb.WriteString("no self react settings:\n")
			sb.WriteString(fmt.Sprintf("enabled: `%t`\n", cfg.Enabled))
			sb.WriteString(fmt.Sprintf("bypass admins: `%t`\n", cfg.BypassAdmins))
			sb.WriteString(fmt.Sprintf("punishment: `%s`\n", cfg.Punishment))
			sb.WriteString(fmt.Sprintf("exempts count: `%d`\n", len(cfg.Exempts)))
			sb.WriteString(fmt.Sprintf("emojis count: `%d`\n", len(cfg.Emojis)))
			return ctx.SendText(sb.String())
		}
		sub := strings.ToLower(ctx.Args[0])
		switch sub {
		case "toggle":
			if len(ctx.Args) > 1 {
				val := strings.ToLower(ctx.Args[1])
				cfg.Enabled = val == "on" || val == "true" || val == "yes" || val == "1"
			} else {
				cfg.Enabled = !cfg.Enabled
			}
			_ = ctx.DB.SaveNoSelfReactCfg(gid, cfg)
			return ctx.SendText(fmt.Sprintf("no self react enabled state: `%t`", cfg.Enabled))
		case "bypass":
			g, err := ctx.Session.Guild(gid)
			if err != nil {
				return ctx.SendText("failed to fetch guild metadata")
			}
			if ctx.AuthorID() != g.OwnerID {
				return ctx.SendText("only the Server Owner can use the bypass subcommand")
			}
			if len(ctx.Args) > 1 {
				val := strings.ToLower(ctx.Args[1])
				cfg.BypassAdmins = val == "on" || val == "true" || val == "yes" || val == "1"
			} else {
				cfg.BypassAdmins = !cfg.BypassAdmins
			}
			_ = ctx.DB.SaveNoSelfReactCfg(gid, cfg)
			return ctx.SendText(fmt.Sprintf("bypass admins state: `%t`", cfg.BypassAdmins))
		case "exempt":
			if len(ctx.Args) > 1 && strings.ToLower(ctx.Args[1]) == "list" {
				if len(cfg.Exempts) == 0 {
					return ctx.SendText("no exempts configured")
				}
				var sb strings.Builder
				sb.WriteString("exempted IDs:\n")
				for id := range cfg.Exempts {
					sb.WriteString(fmt.Sprintf("- `%s`\n", id))
				}
				return ctx.SendText(sb.String())
			}
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("noselfreact")
			}
			target := ctx.Args[1]
			target = strings.Trim(target, "<@!&#>")
			if cfg.Exempts == nil {
				cfg.Exempts = make(map[string]bool)
			}
			if cfg.Exempts[target] {
				delete(cfg.Exempts, target)
				_ = ctx.DB.SaveNoSelfReactCfg(gid, cfg)
				return ctx.SendText(fmt.Sprintf("removed `%s` from exemptions", target))
			}
			cfg.Exempts[target] = true
			_ = ctx.DB.SaveNoSelfReactCfg(gid, cfg)
			return ctx.SendText(fmt.Sprintf("added `%s` to exemptions", target))
		case "punishment":
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("noselfreact")
			}
			pType := strings.ToLower(ctx.Args[1])
			cfg.Punishment = pType
			_ = ctx.DB.SaveNoSelfReactCfg(gid, cfg)
			return ctx.SendText(fmt.Sprintf("set punishment to `%s`", pType))
		case "emoji":
			if len(ctx.Args) > 1 && strings.ToLower(ctx.Args[1]) == "list" {
				if len(cfg.Emojis) == 0 {
					return ctx.SendText("no emojis configured (all self reactions blocked)")
				}
				var sb strings.Builder
				sb.WriteString("monitored emojis:\n")
				for emoji := range cfg.Emojis {
					sb.WriteString(fmt.Sprintf("- %s\n", emoji))
				}
				return ctx.SendText(sb.String())
			}
			if len(ctx.Args) < 2 {
				return ctx.SendHelp("noselfreact")
			}
			emoji := resolveReactionEmoji(ctx.Args[1])
			if cfg.Emojis == nil {
				cfg.Emojis = make(map[string]bool)
			}
			if cfg.Emojis[emoji] {
				delete(cfg.Emojis, emoji)
				_ = ctx.DB.SaveNoSelfReactCfg(gid, cfg)
				return ctx.SendText(fmt.Sprintf("removed %s from monitored list", emoji))
			}
			cfg.Emojis[emoji] = true
			_ = ctx.DB.SaveNoSelfReactCfg(gid, cfg)
			return ctx.SendText(fmt.Sprintf("added %s to monitored list", emoji))
		}
		return nil
	},
}