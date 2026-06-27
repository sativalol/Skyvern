package utility

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"skyvern/internal/ai"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/storage"

	bolt "go.etcd.io/bbolt"
)

func init() {
	manager.RegisterHelp("asklogs", []manager.HelpPage{
		{
			Command:     "AskLogs",
			Syntax:      ".asklogs <query>",
			Description: "Ask AI questions about the server history retrieved from Palantir logs.",
		},
	})
}

var AskLogsCmd = &manager.Command{
	Trigger:     "asklogs",
	Aliases:     []string{"loggpt", "historyask", "raglogs"},
	Name:        "asklogs",
	Description: "Ask AI a question about server history using Palantir logs.",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("asklogs")
		}

		g := config.GetGlobal()
		if !g.AIEnabled {
			return ctx.Reply("[!] AI features are currently disabled.")
		}

		provs, err := ctx.DB.ListAIProviders()
		if err != nil || len(provs) == 0 {
			return ctx.Reply("[!] No AI providers configured. Set one up in the TUI first.")
		}

		isOwner := isBotOwner(ctx.AuthorID())
		var chargeUser bool
		if !isOwner || !g.AIOwnerBypass {
			bal, err := ctx.DB.GetAIBalance(ctx.AuthorID())
			if err != nil || bal <= 0 {
				return ctx.Reply("[!] You do not have any AI tokens. Charge your balance first.")
			}
			chargeUser = true
		}

		prompt := strings.Join(ctx.Args, " ")

		thinkingMsg, err := ctx.ReplyAndGet("[*] Searching Palantir logs and thinking, my sweet girl...")
		if err != nil {
			return err
		}

		// Retrieve logs from Palantir
		palDb := ctx.Mgr.PalantirDB()
		if palDb == nil {
			return ctx.EditReply(thinkingMsg, "[!] Palantir logging database is not open or configured.")
		}

		var logs []manager.PalantirLog
		_ = palDb.View(func(tx *bolt.Tx) error {
			bkt := tx.Bucket([]byte("AuditLogs"))
			if bkt == nil {
				return nil
			}
			c := bkt.Cursor()
			count := 0
			for k, v := c.Last(); k != nil; k, v = c.Prev() {
				var l manager.PalantirLog
				if err := json.Unmarshal(v, &l); err == nil {
					if l.GuildID == ctx.GuildID() {
						logs = append(logs, l)
						count++
						if count >= 1000 { // Scan last 1000 logs max
							break
						}
					}
				}
			}
			return nil
		})

		// Rank logs
		matched := rankLogs(prompt, logs, 15)

		var logContext strings.Builder
		if len(matched) == 0 {
			logContext.WriteString("No matching server events found in logs.\n")
		} else {
			for idx, l := range matched {
				timeStr := l.Timestamp.Format("2006-01-02 15:04:05")
				logContext.WriteString(fmt.Sprintf("[%d] Time: %s | Event: %s | Details: %s %s (User ID: %s, Chan ID: %s)\n",
					idx+1, timeStr, l.Category, l.Title, l.Desc, l.UserID, l.ChannelID))
			}
		}

		// Prepare LLM request
		systemPrompt := "You are a helpful server administrator. You are asked questions about the server history.\n" +
			"Answer the user's question using only the following matching events retrieved from the server logs:\n\n" +
			"<logs>\n" + logContext.String() + "</logs>\n\n" +
			"Format your response cleanly. If the logs do not contain the answer, tell the user that the logs do not show that history."

		res, err := ai.Generate(ctx.DB, provs[0].ID, ai.GenOpts{
			SystemMsg:   systemPrompt,
			UserMsg:     prompt,
			Temperature: 0.5,
			MaxTokens:   800,
		})

		if err != nil {
			return ctx.EditReply(thinkingMsg, fmt.Sprintf("[!] Failed to generate AI response: %v", err))
		}

		_ = ctx.DB.SaveAIConvo(storage.AIConvo{
			ID:        thinkingMsg.ID,
			UID:       ctx.AuthorID(),
			Prompt:    "History RAG: " + prompt,
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

		return ctx.EditOrReplyLarge(thinkingMsg, resText, "rag_response.txt")
	},
}

func rankLogs(query string, logs []manager.PalantirLog, limit int) []manager.PalantirLog {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		if len(logs) > limit {
			return logs[:limit]
		}
		return logs
	}

	type scoredLog struct {
		log   manager.PalantirLog
		score float64
	}

	var scored []scoredLog
	for _, l := range logs {
		content := strings.ToLower(l.Title + " " + l.Desc + " " + l.Category)
		score := 0.0
		for _, w := range words {
			if len(w) > 2 { // ignore short words
				if strings.Contains(content, w) {
					score += 1.0
				}
			}
		}
		if score > 0 {
			scored = append(scored, scoredLog{log: l, score: score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	var out []manager.PalantirLog
	for i := 0; i < len(scored) && i < limit; i++ {
		out = append(out, scored[i].log)
	}
	return out
}
