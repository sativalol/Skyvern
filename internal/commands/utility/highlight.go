package utility
import (
	"fmt"
	"skyvern/internal/manager"
	"strings"
)
func init() {
	manager.RegisterHelp("highlight", []manager.HelpPage{
		{
			Command:     "Highlight Add",
			Syntax:      ".highlight add <keyword>",
			Description: "Receive notifications when a keyword is mentioned in chat.",
		},
		{
			Command:     "Highlight Remove",
			Syntax:      ".highlight remove <keyword>",
			Description: "Remove a keyword from highlight alerts.",
		},
		{
			Command:     "Highlight List",
			Syntax:      ".highlight list",
			Description: "List all your registered highlight keywords.",
		},
		{
			Command:     "Highlight Reset",
			Syntax:      ".highlight reset",
			Description: "Reset/clear all your highlights.",
		},
		{
			Command:     "Highlight Ignore",
			Syntax:      ".highlight ignore <member/channel/role>",
			Description: "Ignore highlight triggers from a user, role, or channel.",
		},
		{
			Command:     "Highlight Ignore List",
			Syntax:      ".highlight ignore list",
			Description: "View list of ignored targets.",
		},
	})
}
var Highlight = &manager.Command{
	Trigger:     "highlight",
	Aliases:     []string{"hl"},
	Name:        "highlight",
	Description: "Set notifications for when a keyword is said",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return listHighlights(ctx)
		}
		sub := strings.ToLower(ctx.Args[0])
		switch sub {
		case "add":
			if len(ctx.Args) < 2 {
				return ctx.Reply("Usage: .highlight add <keyword>")
			}
			kw := strings.Join(ctx.Args[1:], " ")
			err := ctx.DB.AddHighlight(ctx.AuthorID(), kw)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Error saving highlight: %v", err))
			}
			return ctx.Reply(fmt.Sprintf("[*] Added keyword highlight for **%s**.", kw))
		case "remove":
			if len(ctx.Args) < 2 {
				return ctx.Reply("Usage: .highlight remove <keyword>")
			}
			kw := strings.Join(ctx.Args[1:], " ")
			err := ctx.DB.RemoveHighlight(ctx.AuthorID(), kw)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Error removing highlight: %v", err))
			}
			return ctx.Reply(fmt.Sprintf("[*] Removed keyword highlight for **%s**.", kw))
		case "list":
			return listHighlights(ctx)
		case "reset":
			err := ctx.DB.ResetHighlights(ctx.AuthorID())
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Error resetting highlights: %v", err))
			}
			return ctx.Reply("[*] Cleared all your highlight keywords.")
		case "ignore":
			if len(ctx.Args) < 2 {
				return ctx.Reply("Usage: .highlight ignore <member/channel/role>")
			}
			arg := ctx.Args[1]
			if strings.EqualFold(arg, "list") {
				return listHighlightIgnores(ctx)
			}
			targetID := resolveIgnoreID(arg)
			list, _ := ctx.DB.GetHighlightIgnores(ctx.AuthorID())
			found := false
			for _, x := range list {
				if x == targetID {
					found = true
					break
				}
			}
			if found {
				_ = ctx.DB.RemoveHighlightIgnore(ctx.AuthorID(), targetID)
				return ctx.Reply(fmt.Sprintf("[*] Removed `<@%s>` / `<#%s>` from ignore list.", targetID, targetID))
			}
			_ = ctx.DB.AddHighlightIgnore(ctx.AuthorID(), targetID)
			return ctx.Reply(fmt.Sprintf("[*] Added `<@%s>` / `<#%s>` to highlight ignore list.", targetID, targetID))
		default:
			kw := strings.Join(ctx.Args, " ")
			err := ctx.DB.AddHighlight(ctx.AuthorID(), kw)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Error saving highlight: %v", err))
			}
			return ctx.Reply(fmt.Sprintf("[*] Added keyword highlight for **%s**.", kw))
		}
	},
}
func listHighlights(ctx *manager.CommandContext) error {
	list, err := ctx.DB.GetHighlights(ctx.AuthorID())
	if err != nil || len(list) == 0 {
		return ctx.Reply("[*] You do not have any highlight keywords configured.")
	}
	var sb strings.Builder
	sb.WriteString("**Your Highlights:**\n")
	for idx, kw := range list {
		sb.WriteString(fmt.Sprintf("%d. **%s**\n", idx+1, kw))
	}
	return ctx.Reply(sb.String())
}
func listHighlightIgnores(ctx *manager.CommandContext) error {
	list, err := ctx.DB.GetHighlightIgnores(ctx.AuthorID())
	if err != nil || len(list) == 0 {
		return ctx.Reply("[*] Your ignore list is empty.")
	}
	var sb strings.Builder
	sb.WriteString("**Ignored Targets:**\n")
	for idx, id := range list {
		sb.WriteString(fmt.Sprintf("%d. `<@%s>` / `<#%s>` (ID: `%s`)\n", idx+1, id, id, id))
	}
	return ctx.Reply(sb.String())
}
func resolveIgnoreID(q string) string {
	q = strings.TrimSpace(q)
	if strings.HasPrefix(q, "<#") && strings.HasSuffix(q, ">") {
		return strings.Trim(q, "<#>")
	}
	if strings.HasPrefix(q, "<@&") && strings.HasSuffix(q, ">") {
		return strings.Trim(q, "<@&>")
	}
	if strings.HasPrefix(q, "<@") && strings.HasSuffix(q, ">") {
		return strings.Trim(q, "<@!>")
	}
	return q
}