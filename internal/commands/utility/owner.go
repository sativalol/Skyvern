package utility
import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"skyvern/internal/ai"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/storage"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)
func init() {
	manager.RegisterHelp("owner", []manager.HelpPage{
		{
			Command:     "Owner",
			Syntax:      ".owner [subcommand]",
			Description: "Bot management dashboard. Run `.owner` to see all commands.",
		},
	})
}
func cleanID(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
func isBotOwner(uid string) bool {
	g := config.GetGlobal()
	cleanUID := cleanID(uid)
	if cleanUID == "" {
		return false
	}
	repl := strings.NewReplacer(";", ",", " ", ",", "\n", ",", "\r", ",", "\t", ",")
	if g.OwnerIDs != "" {
		for _, part := range strings.Split(repl.Replace(g.OwnerIDs), ",") {
			if cleanID(part) == cleanUID {
				return true
			}
		}
	}
	if g.OwnerIDs == "" {
		for _, part := range strings.Split(repl.Replace(config.DefGlobal().OwnerIDs), ",") {
			if cleanID(part) == cleanUID {
				return true
			}
		}
	}
	return false
}
func parseUserMention(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "<@") && strings.HasSuffix(s, ">") {
		s = strings.TrimPrefix(s, "<@")
		s = strings.TrimSuffix(s, ">")
		s = strings.TrimPrefix(s, "!")
	}
	return s
}
func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
var OwnerCmd = &manager.Command{
	Trigger:     "owner",
	Aliases:     []string{"owners"},
	Name:        "owner",
	Description: "Manage bot owners, AI providers, prompts, and configuration",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		g := config.GetGlobal()
		isOwner := isBotOwner(ctx.AuthorID())
		if !isOwner {
			var pings []string
			repl := strings.NewReplacer(";", ",", " ", ",", "\n", ",", "\r", ",", "\t", ",")
			ownerIDs := g.OwnerIDs
			if strings.TrimSpace(ownerIDs) == "" {
				ownerIDs = config.DefGlobal().OwnerIDs
			}
			for _, part := range strings.Split(repl.Replace(ownerIDs), ",") {
				id := cleanID(part)
				if id != "" {
					pings = append(pings, fmt.Sprintf("<@%s>", id))
				}
			}
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:       "Bot Owners",
				Description: strings.Join(pings, "\n"),
			})
			emb.Color = 0x808080
			return ctx.Respond(emb)
		}
		if len(ctx.Args) == 0 {
			return ownerDashboard(ctx, g)
		}
		subcmd := strings.ToLower(ctx.Args[0])
		if subcmd == "generate" || subcmd == "gen" {
			if len(ctx.Args) < 2 {
				return ctx.Reply("Usage: `.owner generate <amount>`")
			}
			tokens, err := strconv.Atoi(ctx.Args[1])
			if err != nil || tokens <= 0 {
				return ctx.Reply("[!] Invalid amount. Must be a positive integer.")
			}
			key := fmt.Sprintf("ai-%s-%s-%s", randHex(4), randHex(4), randHex(4))
			_ = ctx.DB.SaveAIKey(key, tokens)
			return ctx.Reply(fmt.Sprintf("**AI Token Code Generated:**\n`%s`\nValue: **%d** tokens", key, tokens))
		}
		if subcmd == "keys" || subcmd == "list" {
			keys, err := ctx.DB.ListAIKeys()
			if err != nil || len(keys) == 0 {
				return ctx.Reply("No active AI token codes.")
			}
			var sb strings.Builder
			sb.WriteString("**Active AI Token Codes:**\n")
			for k, v := range keys {
				sb.WriteString(fmt.Sprintf("• `%s` — %d tokens\n", k, v))
			}
			return ctx.Reply(sb.String())
		}
		if subcmd == "remove" || subcmd == "delete" {
			if len(ctx.Args) < 2 {
				return ctx.Reply("Usage: `.owner remove <key>`")
			}
			key := ctx.Args[1]
			if _, err := ctx.DB.GetAIKey(key); err != nil {
				return ctx.Reply("[!] Code not found.")
			}
			_ = ctx.DB.DeleteAIKey(key)
			return ctx.Reply("[+] Token code invalidated.")
		}
		if subcmd == "add" {
			if len(ctx.Args) < 3 {
				return ctx.Reply("Usage: `.owner add <@user> <amount>`")
			}
			targetID := parseUserMention(ctx.Args[1])
			amount, err := strconv.Atoi(ctx.Args[2])
			if targetID == "" || err != nil || amount <= 0 {
				return ctx.Reply("[!] Invalid args. Usage: `.owner add <@user> <amount>`")
			}
			curr, _ := ctx.DB.GetAIBalance(targetID)
			_ = ctx.DB.SaveAIBalance(targetID, curr+amount)
			return ctx.Reply(fmt.Sprintf("Added **%d** tokens to <@%s>. New balance: **%d**.", amount, targetID, curr+amount))
		}
		if subcmd == "balance" || subcmd == "bal" {
			if len(ctx.Args) < 2 {
				return ctx.Reply("Usage: `.owner balance <@user>`")
			}
			targetID := parseUserMention(ctx.Args[1])
			if targetID == "" {
				return ctx.Reply("[!] Invalid user mention.")
			}
			bal, _ := ctx.DB.GetAIBalance(targetID)
			return ctx.Reply(fmt.Sprintf("AI balance for <@%s>: **%d** tokens.", targetID, bal))
		}
		if subcmd == "ai" {
			if len(ctx.Args) < 2 {
				return ctx.Reply("[!] Use `.owner ai on` or `.owner ai off [reason]`.")
			}
			action := strings.ToLower(ctx.Args[1])
			if action == "off" {
				reason := ""
				if len(ctx.Args) > 2 {
					reason = strings.Join(ctx.Args[2:], " ")
				}
				g.AIEnabled = false
				g.AIDisableReason = reason
				_ = ctx.DB.SaveGlobal(g)
				config.SetGlobal(g)
				if reason != "" {
					return ctx.Reply(fmt.Sprintf("[+] AI disabled: %s", reason))
				}
				return ctx.Reply("[+] AI disabled.")
			} else if action == "on" {
				g.AIEnabled = true
				g.AIDisableReason = ""
				_ = ctx.DB.SaveGlobal(g)
				config.SetGlobal(g)
				return ctx.Reply("[+] AI enabled.")
			}
			return ctx.Reply("[!] Use `on` or `off`.")
		}
		if subcmd == "provider" || subcmd == "prov" {
			return handleProvider(ctx)
		}
		if subcmd == "prompt" {
			return handlePrompt(ctx)
		}
		if subcmd == "set" {
			if len(ctx.Args) < 3 {
				return ctx.Reply("[!] Use `.owner set <setting> <value>`.")
			}
			targetID := parseUserMention(ctx.Args[1])
			if targetID != "" {
				amount, err := strconv.Atoi(ctx.Args[2])
				if err == nil && amount >= 0 {
					_ = ctx.DB.SaveAIBalance(targetID, amount)
					return ctx.Reply(fmt.Sprintf("Set <@%s>'s AI token balance to **%d**.", targetID, amount))
				}
			}
			setting := strings.ToLower(ctx.Args[1])
			value := strings.Join(ctx.Args[2:], " ")
			lowerVal := strings.ToLower(value)
			isTrue := lowerVal == "yes" || lowerVal == "true" || lowerVal == "1" || lowerVal == "on" || lowerVal == "enable" || lowerVal == "enabled"
			switch setting {
			case "name":
				g.Name = value
			case "prefix":
				g.Prefix = value
			case "footer":
				g.Footer = value
			case "embedcolor", "color":
				colStr := strings.TrimPrefix(value, "#")
				colVal, err := strconv.ParseInt(colStr, 16, 32)
				if err != nil {
					return ctx.Reply("[!] Invalid hex color.")
				}
				g.EmbedColor = int(colVal)
			case "matrixcolor":
				g.MatrixColor = value
			case "spotify":
				g.Spotify = value
			case "alwaysontop":
				g.AlwaysOnTop = isTrue
			case "showlogo":
				g.ShowLogo = isTrue
			case "autostartlavalink", "autolavalink":
				g.AutoStartLavalink = isTrue
			case "lavalinkhost":
				g.LavalinkHost = value
			case "lavalinkpass":
				g.LavalinkPass = value
			case "emojiserverid":
				g.EmojiServerID = value
			case "ownerids":
				g.OwnerIDs = value
			case "commandson", "commandsenabled":
				g.CommandsOn = isTrue
			case "aimemory", "memory":
				g.AIMemory = isTrue
			default:
				return ctx.Reply(fmt.Sprintf("[!] Unknown setting `%s`.", setting))
			}
			_ = ctx.DB.SaveGlobal(g)
			config.SetGlobal(g)
			return ctx.Reply(fmt.Sprintf("[+] `%s` updated.", setting))
		}
		return ctx.Reply("[!] Unknown subcommand. Run `.owner` for the full list.")
	},
}
func ownerDashboard(ctx *manager.CommandContext, g config.GlobalCfg) error {
	var pings []string
	repl := strings.NewReplacer(";", ",", " ", ",", "\n", ",", "\r", ",", "\t", ",")
	ownerIDs := g.OwnerIDs
	if strings.TrimSpace(ownerIDs) == "" {
		ownerIDs = config.DefGlobal().OwnerIDs
	}
	for _, part := range strings.Split(repl.Replace(ownerIDs), ",") {
		id := cleanID(part)
		if id != "" {
			pings = append(pings, fmt.Sprintf("<@%s>", id))
		}
	}
	aiStatus := "Enabled"
	if !g.AIEnabled {
		aiStatus = "Disabled"
		if g.AIDisableReason != "" {
			aiStatus = fmt.Sprintf("Disabled (%s)", g.AIDisableReason)
		}
	}
	provs, _ := ctx.DB.ListAIProviders()
	provSummary := "none"
	if len(provs) > 0 {
		var names []string
		for _, p := range provs {
			names = append(names, fmt.Sprintf("`%s` (%s)", p.ID, p.Type))
		}
		provSummary = strings.Join(names, ", ")
	}
	promptCfg, _ := ai.LoadPrompts()
	promptPreview := promptCfg.SystemPrompt
	if len(promptPreview) > 80 {
		promptPreview = promptPreview[:80] + "..."
	}
	desc := fmt.Sprintf(
		"**Owners:** %s\n\n"+
			"**[Global Settings]**\n"+
			"• Name: `%s`  Prefix: `%s`\n"+
			"• Footer: `%s`\n"+
			"• Embed Color: `#%06x`\n"+
			"• AI Status: `%s`  AI Memory: `%t`\n"+
			"• Spotify: `%s`  AutoLavalink: `%v`\n"+
			"• Commands Enabled: `%t`  Lavalink Host: `%s`\n\n"+
			"**[AI Providers]** %s\n\n"+
			"**[System Prompt]**\n```%s```\n\n"+
			"**[Commands]**\n"+
			"• `.owner set <setting> <value>` — edit global settings\n"+
			"• `.owner ai on|off [reason]` — toggle AI\n"+
			"• `.owner provider list|add|set|delete|show` — manage AI providers\n"+
			"• `.owner prompt show|set <text>|reset` — manage system prompt\n"+
			"• `.owner generate <n>` / `.owner keys` / `.owner remove <key>`\n"+
			"• `.owner add|balance|set <@user> <n>` — token balances",
		strings.Join(pings, ", "),
		g.Name, g.Prefix, g.Footer, g.EmbedColor,
		aiStatus, g.AIMemory, g.Spotify, g.AutoStartLavalink, g.CommandsOn, g.LavalinkHost,
		provSummary, promptPreview,
	)
	emb := config.Build(ctx.Cfg, config.EmbedOpt{
		Title:       "Bot Management",
		Description: desc,
	})
	emb.Color = 0x2b2d31
	return ctx.Respond(emb)
}
func handleProvider(ctx *manager.CommandContext) error {
	args := ctx.Args[1:]                    
	if len(args) == 0 {
		return ctx.Reply("Subcommands: `list`, `add <id> <type> <apikey> [model]`, `set <id> <field> <val>`, `delete <id>`, `show [id]`")
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "list":
		provs, _ := ctx.DB.ListAIProviders()
		if len(provs) == 0 {
			return ctx.Reply("No AI providers configured.")
		}
		var sb strings.Builder
		sb.WriteString("**AI Providers:**\n")
		for _, p := range provs {
			keyHint := "none"
			if p.APIKey != "" {
				n := len(p.APIKey)
				if n > 4 {
					keyHint = "…" + p.APIKey[n-4:]
				} else {
					keyHint = "set"
				}
			}
			sb.WriteString(fmt.Sprintf("• `%s` — type:`%s` model:`%s` key:`%s`\n",
				p.ID, p.Type, p.DefaultModel, keyHint))
		}
		return ctx.Reply(sb.String())
	case "show":
		id := "default"
		if len(args) > 1 {
			id = args[1]
		}
		p, err := ctx.DB.GetAIProvider(id)
		if err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Provider `%s` not found.", id))
		}
		keyHint := "(none)"
		if p.APIKey != "" {
			n := len(p.APIKey)
			if n > 4 {
				keyHint = "…" + p.APIKey[n-4:]
			} else {
				keyHint = "(set)"
			}
		}
		return ctx.Reply(fmt.Sprintf(
			"**Provider: `%s`**\nType: `%s`\nName: `%s`\nModel: `%s`\nBase URL: `%s`\nAPI Key: `%s`\nFallback: `%s`\nMax Tokens: `%d`  Max Requests: `%d`",
			p.ID, p.Type, p.Name, p.DefaultModel, p.BaseURL, keyHint, p.FallbackID, p.MaxTokens, p.MaxRequests))
	case "add":
		if len(args) < 4 {
			return ctx.Reply("Usage: `.owner provider add <id> <type> <apikey> [model]`\nTypes: `openai`, `anthropic`, `ollama`, `gemini`")
		}
		p := storage.AIProvider{
			ID:           args[1],
			Type:         strings.ToLower(args[2]),
			APIKey:       args[3],
			Name:         args[1],
			DefaultModel: "gpt-4o-mini",
		}
		if len(args) > 4 {
			p.DefaultModel = args[4]
		}
		if err := ctx.DB.SaveAIProvider(p); err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Save failed: %v", err))
		}
		return ctx.Reply(fmt.Sprintf("[+] Provider `%s` (%s) added.", p.ID, p.Type))
	case "set":
		if len(args) < 4 {
			return ctx.Reply("Usage: `.owner provider set <id> <field> <value>`\nFields: `apikey`, `model`, `type`, `baseurl`, `fallback`, `maxtokens`, `maxrequests`, `name`")
		}
		id := args[1]
		p, err := ctx.DB.GetAIProvider(id)
		if err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Provider `%s` not found.", id))
		}
		field := strings.ToLower(args[2])
		val := strings.Join(args[3:], " ")
		switch field {
		case "apikey", "key":
			p.APIKey = val
		case "model", "defaultmodel":
			p.DefaultModel = val
		case "type":
			p.Type = val
		case "baseurl", "url":
			p.BaseURL = val
		case "fallback", "fallbackid":
			p.FallbackID = val
		case "name":
			p.Name = val
		case "maxtokens":
			n, _ := strconv.ParseInt(val, 10, 64)
			p.MaxTokens = n
		case "maxrequests":
			n, _ := strconv.ParseInt(val, 10, 64)
			p.MaxRequests = n
		default:
			return ctx.Reply(fmt.Sprintf("[!] Unknown field `%s`.", field))
		}
		if err := ctx.DB.SaveAIProvider(p); err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Save failed: %v", err))
		}
		return ctx.Reply(fmt.Sprintf("[+] Provider `%s`.`%s` updated.", id, field))
	case "delete", "del", "rm":
		if len(args) < 2 {
			return ctx.Reply("Usage: `.owner provider delete <id>`")
		}
		if err := ctx.DB.DeleteAIProvider(args[1]); err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Delete failed: %v", err))
		}
		return ctx.Reply(fmt.Sprintf("[+] Provider `%s` deleted.", args[1]))
	}
	return ctx.Reply("[!] Unknown provider subcommand. Try `list`, `add`, `set`, `delete`, `show`.")
}
func handlePrompt(ctx *manager.CommandContext) error {
	args := ctx.Args[1:]
	if len(args) == 0 {
		return ctx.Reply("Subcommands: `show`, `set <text>`, `reset`")
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "show":
		cfg, _ := ai.LoadPrompts()
		prompt := cfg.SystemPrompt
		if len(prompt) > 3900 {
			for len(prompt) > 0 {
				chunk := prompt
				if len(chunk) > 1900 {
					chunk = prompt[:1900]
				}
				prompt = prompt[len(chunk):]
				if err := ctx.Reply("```\n" + chunk + "\n```"); err != nil {
					return err
				}
			}
			return nil
		}
		return ctx.Reply("**System Prompt:**\n```\n" + prompt + "\n```")
	case "set":
		newPrompt := strings.Join(args[1:], " ")

		var fileContent string
		if ctx.Message != nil {
			var atts []*discordgo.MessageAttachment
			if len(ctx.Message.Attachments) > 0 {
				atts = ctx.Message.Attachments
			} else if ctx.Message.ReferencedMessage != nil && len(ctx.Message.ReferencedMessage.Attachments) > 0 {
				atts = ctx.Message.ReferencedMessage.Attachments
			}
			if len(atts) > 0 {
				att := atts[0]
				// stay safe on memory
				if att.Size <= 5*1024*1024 {
					resp, err := http.Get(att.URL)
					if err == nil {
						defer resp.Body.Close()
						b, _ := io.ReadAll(resp.Body)
						fileContent = string(b)
					}
				}
			}
		}



		if fileContent != "" {
			newPrompt = fileContent
		}

		if strings.TrimSpace(newPrompt) == "" {
			return ctx.Reply("[!] Usage: `.owner prompt set <text>` or attach/reply with a file.")
		}

		cfg := ai.PromptsConfig{SystemPrompt: newPrompt}
		if err := ai.SavePrompts(cfg); err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Save failed: %v", err))
		}
		return ctx.Reply(fmt.Sprintf("[+] System prompt updated (%d chars).", len(newPrompt)))
	case "reset":
		cfg := ai.PromptsConfig{SystemPrompt: ai.DefaultSystemPrompt}
		_ = ai.SavePrompts(cfg)
		return ctx.Reply("[+] System prompt reset to default.")
	}
	return ctx.Reply("[!] Unknown prompt subcommand. Try `show`, `set <text>`, `reset`.")
}