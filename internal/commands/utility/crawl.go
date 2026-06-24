package utility

import (
	"fmt"
	"strconv"
	"strings"
	"skyvern/internal/ai/tools"
	"skyvern/internal/manager"
)

func init() {
	manager.RegisterHelp("crawl", []manager.HelpPage{
		{
			Command:     "Crawl",
			Syntax:      ".crawl <url> [depth] [max_pages]",
			Description: "Crawls a domain starting at the seed URL and lists pages, titles, descriptions, and discovered links.",
		},
	})
}

var CrawlCmd = &manager.Command{
	Trigger:     "crawl",
	Name:        "crawl",
	Description: "Crawl a website starting at the seed URL",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("crawl")
		}

		u := ctx.Args[0]
		depth := 1
		maxPages := 10

		if len(ctx.Args) > 1 {
			if v, err := strconv.Atoi(ctx.Args[1]); err == nil && v > 0 {
				depth = v
			}
		}

		if len(ctx.Args) > 2 {
			if v, err := strconv.Atoi(ctx.Args[2]); err == nil && v > 0 {
				maxPages = v
			}
		}

		if depth > 2 {
			depth = 2
		}
		if maxPages > 20 {
			maxPages = 20
		}

		_ = ctx.Reply(fmt.Sprintf("[*] Crawling %s (depth limit: %d, max pages: %d), please wait...", u, depth, maxPages))

		results, err := tools.Crawl(u, depth, maxPages)
		if err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Crawl failed: %v", err))
		}

		if len(results) == 0 {
			return ctx.Reply("[!] No pages were successfully crawled.")
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("=== CRAWL REPORT FOR: %s ===\n", u))
		sb.WriteString(fmt.Sprintf("Total Pages Crawled: %d\n\n", len(results)))

		for i, res := range results {
			sb.WriteString(fmt.Sprintf("[%d] URL: %s\n", i+1, res.URL))
			if res.Title != "" {
				sb.WriteString(fmt.Sprintf("    Title: %s\n", res.Title))
			}
			if res.Description != "" {
				sb.WriteString(fmt.Sprintf("    Description: %s\n", res.Description))
			}
			
			snippet := res.TextContent
			if len(snippet) > 200 {
				snippet = snippet[:197] + "..."
			}
			snippet = strings.ReplaceAll(snippet, "\n", " ")
			sb.WriteString(fmt.Sprintf("    Content Snippet: %s\n", snippet))

			if len(res.Links) > 0 {
				sb.WriteString("    Links Discovered:\n")
				limit := len(res.Links)
				if limit > 5 {
					limit = 5
				}
				for _, link := range res.Links[:limit] {
					sb.WriteString(fmt.Sprintf("      - %s (%s)\n", link.Text, link.URL))
				}
				if len(res.Links) > 5 {
					sb.WriteString(fmt.Sprintf("      - ...and %d more link(s)\n", len(res.Links)-5))
				}
			}
			sb.WriteString("\n")
		}

		return ctx.ReplyLarge(sb.String(), "crawl.txt")
	},
}
