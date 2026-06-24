package tools

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

type CrawlTask struct {
	URL   string
	Depth int
}

func Crawl(seedURL string, maxDepth, maxPages int) ([]ScrapeResult, error) {
	if maxDepth <= 0 {
		maxDepth = 2
	}
	if maxPages <= 0 {
		maxPages = 10
	}

	seedParsed, err := url.Parse(seedURL)
	if err != nil {
		return nil, fmt.Errorf("invalid seed URL: %w", err)
	}
	if seedParsed.Host == "" {
		seedURL = "https://" + seedURL
		seedParsed, err = url.Parse(seedURL)
		if err != nil {
			return nil, fmt.Errorf("invalid seed URL after prefix fix: %w", err)
		}
	}

	targetHost := seedParsed.Host

	var results []ScrapeResult
	visited := make(map[string]bool)
	queue := []CrawlTask{{URL: seedURL, Depth: 0}}

	normalize := func(raw string) string {
		p, err := url.Parse(raw)
		if err != nil {
			return raw
		}
		p.Fragment = ""
		rawPath := strings.TrimSuffix(p.Path, "/")
		p.Path = rawPath
		return p.String()
	}

	visited[normalize(seedURL)] = true

	for len(queue) > 0 {
		if len(results) >= maxPages {
			break
		}

		cur := queue[0]
		queue = queue[1:]

		if len(results) > 0 {
			time.Sleep(250 * time.Millisecond)
		}

		res, err := ScrapeWithOptions(cur.URL, ScrapeOpts{
			Timeout: 8 * time.Second,
		})
		if err != nil {
			continue
		}

		results = append(results, *res)

		if cur.Depth >= maxDepth {
			continue
		}

		for _, l := range res.Links {
			lParsed, err := url.Parse(l.URL)
			if err != nil {
				continue
			}

			if lParsed.Host != targetHost {
				continue
			}

			norm := normalize(l.URL)
			if visited[norm] {
				continue
			}

			visited[norm] = true
			queue = append(queue, CrawlTask{
				URL:   l.URL,
				Depth: cur.Depth + 1,
			})
		}
	}

	return results, nil
}
