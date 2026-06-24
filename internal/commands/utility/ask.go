package utility

import (
	"fmt"
	"io"
	"net/http"
	"skyvern/internal/ai"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/search"
	"skyvern/internal/storage"
	"skyvern/internal/ai/tools"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func init() {
	manager.RegisterHelp("ask", []manager.HelpPage{
		{
			Command:     "Ask",
			Syntax:      ".ask <prompt>",
			Description: "Ask AI a question or give it a prompt.",
		},
	})
	manager.RegisterHelp("redeem", []manager.HelpPage{
		{
			Command:     "Redeem",
			Syntax:      ".redeem <key>",
			Description: "Redeem an AI token code to add to your balance.",
		},
	})
	manager.RegisterHelp("aihistory", []manager.HelpPage{
		{
			Command:     "AI History",
			Syntax:      ".aihistory [number]",
			Description: "View your past AI conversations list or detail a specific one.",
		},
	})
}



var AskCmd = &manager.Command{
	Trigger:     "ask",
	Aliases:     []string{"ai", "gpt"},
	Name:        "ask",
	Description: "Generate content using AI",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("ask")
		}

		g := config.GetGlobal()

		if !g.AIEnabled {
			msg := "[!] AI bots are currently disabled."
			if g.AIDisableReason != "" {
				msg = fmt.Sprintf("[!] AI bots are currently disabled: %s", g.AIDisableReason)
			}
			return ctx.Reply(msg)
		}

		provs, err := ctx.DB.ListAIProviders()
		if err != nil || len(provs) == 0 {
			return ctx.Reply("[!] No AI providers configured. Please set one up in the TUI Settings first.")
		}

		isOwner := isBotOwner(ctx.AuthorID())


		var chargeUser bool
		if !isOwner || !g.AIOwnerBypass {
			bal, err := ctx.DB.GetAIBalance(ctx.AuthorID())
			if err != nil || bal <= 0 {
				return ctx.Reply("[!] You do not have any AI tokens. You need tokens to use this command. Please buy tokens or hit up the owner(s) to get some.")
			}
			chargeUser = true
		}

		prompt := strings.Join(ctx.Args, " ")

		var fileContents []string
		processAttachments := func(attachments []*discordgo.MessageAttachment) {
			for _, att := range attachments {
				if att.Size > 2*1024*1024 {
					continue
				}
				resp, err := http.Get(att.URL)
				if err != nil {
					continue
				}
				defer resp.Body.Close()

				b, err := io.ReadAll(resp.Body)
				if err != nil {
					continue
				}

				ext := strings.ToLower(att.Filename)
				isText := false
				for _, tExt := range []string{".txt", ".log", ".json", ".yaml", ".yml", ".go", ".py", ".js", ".ts", ".html", ".css", ".md", ".sh", ".bat", ".toml", ".xml", ".ini", ".conf", ".cfg"} {
					if strings.HasSuffix(ext, tExt) {
						isText = true
						break
					}
				}
				if !isText && len(b) > 0 {
					isText = true
					for _, ch := range b {
						if ch == 0 {
							isText = false
							break
						}
					}
				}

				if isText {
					fileContents = append(fileContents, fmt.Sprintf("\n--- FILE: %s ---\n%s\n--- END OF FILE ---", att.Filename, string(b)))
				}
			}
		}

		if ctx.Message != nil {
			processAttachments(ctx.Message.Attachments)
			if ctx.Message.ReferencedMessage != nil {
				processAttachments(ctx.Message.ReferencedMessage.Attachments)
			}
		}

		if len(fileContents) > 0 {
			prompt += "\n" + strings.Join(fileContents, "\n")
		}

		var toolsList []ai.ToolDef

		toolsList = append(toolsList, ai.ToolDef{
			Name:        "web_search",
			Description: "Search DuckDuckGo Lite for the given query to get recent/relevant information.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The search query to look up.",
					},
				},
				"required": []string{"query"},
			},
			Handler: func(args map[string]interface{}) (string, error) {
				q, _ := args["query"].(string)
				if q == "" {
					return "Error: query parameter is empty", nil
				}
				html, err := search.FetchDDGLite(q)
				if err != nil {
					return fmt.Sprintf("Error fetching DDG search: %v", err), nil
				}
				var sb strings.Builder
				results := search.ParseDDGLite(html, 3)
				if len(results) == 0 {
					return "No results found.", nil
				}
				for i, r := range results {
					sb.WriteString(fmt.Sprintf("[%d] %s\nSnippet: %s\n\n", i+1, r.Title, r.Snippet))
				}
				return sb.String(), nil
			},
		})

		toolsList = append(toolsList, ai.ToolDef{
			Name:        "wikipedia_search",
			Description: "Search Wikipedia for the given query to get encyclopedic information.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The query to search on Wikipedia.",
					},
				},
				"required": []string{"query"},
			},
			Handler: func(args map[string]interface{}) (string, error) {
				q, _ := args["query"].(string)
				if q == "" {
					return "Error: query parameter is empty", nil
				}
				results, err := search.QueryWikipedia(q, 3)
				if err != nil {
					return fmt.Sprintf("Error querying Wikipedia: %v", err), nil
				}
				if len(results) == 0 {
					return "No results found.", nil
				}
				var sb strings.Builder
				for i, r := range results {
					sb.WriteString(fmt.Sprintf("[%d] Wikipedia: %s\nSnippet: %s\n\n", i+1, r.Title, r.Snippet))
				}
				return sb.String(), nil
			},
		})

		toolsList = append(toolsList, ai.ToolDef{
			Name:        "web_scrape",
			Description: "Scrapes the main text content, metadata, and links from the specified URL.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "The URL of the website to scrape.",
					},
				},
				"required": []string{"url"},
			},
			Handler: func(args map[string]interface{}) (string, error) {
				u, _ := args["url"].(string)
				if u == "" {
					return "Error: url parameter is empty", nil
				}
				res, err := tools.Scrape(u)
				if err != nil {
					return fmt.Sprintf("Error scraping URL: %v", err), nil
				}
				
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("URL: %s\nTitle: %s\nDescription: %s\n\nContent:\n%s\n\nDiscovered Links:\n", res.URL, res.Title, res.Description, res.TextContent))
				for _, link := range res.Links {
					sb.WriteString(fmt.Sprintf("- %s (%s)\n", link.Text, link.URL))
				}
				return sb.String(), nil
			},
		})

		toolsList = append(toolsList, ai.ToolDef{
			Name:        "web_crawl",
			Description: "Crawls a domain starting at the seed URL, returning a sitemap with page titles, descriptions, and content snippets.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "The seed URL to start crawling from.",
					},
					"max_depth": map[string]interface{}{
						"type":        "integer",
						"description": "Optional depth limit (default 1, max 2).",
					},
					"max_pages": map[string]interface{}{
						"type":        "integer",
						"description": "Optional page limit (default 10, max 20).",
					},
				},
				"required": []string{"url"},
			},
			Handler: func(args map[string]interface{}) (string, error) {
				u, _ := args["url"].(string)
				if u == "" {
					return "Error: url parameter is empty", nil
				}
				depth := 1
				maxPages := 10
				if dVal, ok := args["max_depth"].(float64); ok {
					depth = int(dVal)
				}
				if pVal, ok := args["max_pages"].(float64); ok {
					maxPages = int(pVal)
				}
				if depth > 2 {
					depth = 2
				}
				if maxPages > 20 {
					maxPages = 20
				}
				
				results, err := tools.Crawl(u, depth, maxPages)
				if err != nil {
					return fmt.Sprintf("Error crawling URL: %v", err), nil
				}
				if len(results) == 0 {
					return "No pages crawled.", nil
				}
				
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("=== CRAWLED PAGES FOR %s ===\n", u))
				for _, r := range results {
					sb.WriteString(fmt.Sprintf("- URL: %s\n  Title: %s\n  Desc: %s\n\n", r.URL, r.Title, r.Description))
				}
				return sb.String(), nil
			},
		})

		pCfg, err := ai.LoadPrompts()
		sysMsg := pCfg.SystemPrompt

		sysMsg = strings.ReplaceAll(sysMsg, "${currentDate}", time.Now().Format("Monday, January 2, 2006 3:04 PM MST"))
		sysMsg = strings.ReplaceAll(sysMsg, "${userRecognition}", fmt.Sprintf("User: %s (ID: %s)", ctx.AuthorTag(), ctx.AuthorID()))
		sysMsg = strings.ReplaceAll(sysMsg, "${channelContext}", fmt.Sprintf("Channel: <#%s>", ctx.ChanID()))
		sysMsg = strings.ReplaceAll(sysMsg, "${searchInstructions}", "You have tools available to search the web, search Wikipedia, scrape specific web pages, or crawl domains if you need information.")

		thinkingMsg, err := ctx.ReplyAndGet("[*] Thinking, please wait...")
		if err != nil {
			return err
		}

		var msgs []ai.Message
		if g.AIMemory {
			list, err := ctx.DB.ListAIConvos(ctx.AuthorID())
			if err == nil && len(list) > 0 {
				for i := 0; i < len(list); i++ {
					for j := i + 1; j < len(list); j++ {
						if list[i].Timestamp.After(list[j].Timestamp) {
							list[i], list[j] = list[j], list[i]
						}
					}
				}
				startIdx := 0
				if len(list) > 5 {
					startIdx = len(list) - 5
				}
				for _, c := range list[startIdx:] {
					msgs = append(msgs, ai.Message{Role: "user", Content: c.Prompt})
					msgs = append(msgs, ai.Message{Role: "assistant", Content: c.Response})
				}
			}
		}
		msgs = append(msgs, ai.Message{Role: "user", Content: prompt})

		res, err := ai.Generate(ctx.DB, provs[0].ID, ai.GenOpts{
			Messages:    msgs,
			SystemMsg:   sysMsg,
			Temperature: 0.7,
			MaxTokens:   1200,
			Tools:       toolsList,
		})

		if err != nil {
			return ctx.EditReply(thinkingMsg, fmt.Sprintf("[!] failed: %v", err))
		}

		_ = ctx.DB.SaveAIConvo(storage.AIConvo{
			ID:        thinkingMsg.ID,
			UID:       ctx.AuthorID(),
			Prompt:    prompt,
			Response:  res.Text,
			Timestamp: time.Now(),
		})

		bal, used, _ := ctx.DB.ChargeAndIncrementAIToken(ctx.AuthorID(), chargeUser)
		resText := res.Text
		if chargeUser {
			resText += fmt.Sprintf("\n\n*(Used 1 token. Remaining: %d. Total tokens used: %d)*", bal, used)
		} else {
			resText += fmt.Sprintf("\n\n*(Total tokens used: %d)*", used)
		}

		return ctx.EditOrReplyLarge(thinkingMsg, resText, "ai_response.txt")
	},
}

var RedeemCmd = &manager.Command{
	Trigger:     "redeem",
	Name:        "redeem",
	Description: "Redeem an AI token code",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.Reply("Usage: `.redeem <key>`")
		}
		key := ctx.Args[0]
		tokens, err := ctx.DB.GetAIKey(key)
		if err != nil {
			return ctx.Reply("❌ Invalid or expired token code.")
		}

		currBal, _ := ctx.DB.GetAIBalance(ctx.AuthorID())
		newBal := currBal + tokens
		_ = ctx.DB.SaveAIBalance(ctx.AuthorID(), newBal)
		_ = ctx.DB.DeleteAIKey(key)

		return ctx.Reply(fmt.Sprintf("🎉 **Success!** You redeemed %d tokens. Your new balance is %d tokens.", tokens, newBal))
	},
}

var TokensCmd = &manager.Command{
	Trigger:     "aibalance",
	Aliases:     []string{"aibal", "aitokens", "tokens"},
	Name:        "aibalance",
	Description: "Check your current AI token balance",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		isOwner := isBotOwner(ctx.AuthorID())
		targetID := ctx.AuthorID()
		if len(ctx.Args) > 0 {
			targetID = parseUserMention(ctx.Args[0])
			if targetID == "" {
				return ctx.Reply("[!] Invalid user mention.")
			}
			if targetID != ctx.AuthorID() && !isOwner {
				return ctx.Reply("[!] Only bot owners can view other users' balances.")
			}
		}

		bal, _ := ctx.DB.GetAIBalance(targetID)
		if targetID == ctx.AuthorID() {
			return ctx.Reply(fmt.Sprintf("🪙 Your AI Token Balance: **%d** tokens.", bal))
		}
		return ctx.Reply(fmt.Sprintf("🪙 AI Token Balance for <@%s>: **%d** tokens.", targetID, bal))
	},
}

var AIHistoryCmd = &manager.Command{
	Trigger:     "aihistory",
	Aliases:     []string{"aih", "convos", "aiquestions"},
	Name:        "aihistory",
	Description: "View your past AI conversations and questions",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		list, err := ctx.DB.ListAIConvos(ctx.AuthorID())
		if err != nil || len(list) == 0 {
			return ctx.Reply("[!] No past AI conversations found.")
		}

		for i := 0; i < len(list); i++ {
			for j := i + 1; j < len(list); j++ {
				if list[i].Timestamp.Before(list[j].Timestamp) {
					list[i], list[j] = list[j], list[i]
				}
			}
		}

		if len(ctx.Args) > 0 {
			var num int
			_, _ = fmt.Sscanf(ctx.Args[0], "%d", &num)
			if num < 1 || num > len(list) {
				return ctx.Reply(fmt.Sprintf("[!] Invalid number. Choose between 1 and %d.", len(list)))
			}
			c := list[num-1]
			tStr := c.Timestamp.Format("01/02/2006 3:04 PM")
			fullText := fmt.Sprintf("📜 **AI Conversation #%d** (%s)\n\n**Question:**\n%s\n\n**Answer:**\n%s", num, tStr, c.Prompt, c.Response)
			if len(fullText) > 1900 {
				return ctx.EditOrReplyLarge(nil, fullText, fmt.Sprintf("convo_%d.txt", num))
			}
			return ctx.Reply(fullText)
		}

		var sb strings.Builder
		sb.WriteString("📜 **Your Past AI Conversations:**\n\n")
		limit := 5
		if len(list) < limit {
			limit = len(list)
		}

		for i := 0; i < limit; i++ {
			c := list[i]
			tStr := c.Timestamp.Format("01/02 3:04 PM")
			q := c.Prompt
			if len(q) > 60 {
				q = q[:57] + "..."
			}
			sb.WriteString(fmt.Sprintf("**%d.** [%s] Q: `%s`\n", i+1, tStr, q))
		}
		sb.WriteString("\n💡 *Use `.aihistory <number>` to view the full details of a conversation.*")

		return ctx.Reply(sb.String())
	},
}
