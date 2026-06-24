package search

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"skyvern/internal/spoofer"
	"strings"
	"time"
)

func FetchDDGLite(query string) (string, error) {
	val := url.Values{}
	val.Set("q", query)
	req, err := http.NewRequest("POST", "https://lite.duckduckgo.com/lite/", strings.NewReader(val.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	spoofer.SetHeaders(req, "")

	cli := &http.Client{Timeout: 6 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type Result struct {
	Title   string
	Snippet string
}

func CleanHTML(s string) string {
	re := regexp.MustCompile("<[^>]*>")
	s = re.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "class=\"searchmatch\"", "")
	s = strings.ReplaceAll(s, "span", "")
	return strings.TrimSpace(s)
}

func ParseDDGLite(html string, limit int) []Result {
	reSnippet := regexp.MustCompile(`(?s)<td class=['"]result-snippet['"][^>]*>(.*?)<\/td>`)
	reLink := regexp.MustCompile(`(?s)<a[^>]*class=['"]result-link['"][^>]*>(.*?)<\/a>`)

	snippets := reSnippet.FindAllStringSubmatch(html, limit)
	links := reLink.FindAllStringSubmatch(html, limit)

	var res []Result
	for i := 0; i < len(snippets); i++ {
		title := "No Title"
		if i < len(links) {
			title = CleanHTML(links[i][1])
		}
		snippet := CleanHTML(snippets[i][1])
		res = append(res, Result{Title: title, Snippet: snippet})
	}
	return res
}

func QueryWikipedia(query string, limit int) ([]Result, error) {
	apiURL := fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=query&list=search&srsearch=%s&utf8=&format=json", url.QueryEscape(query))
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", spoofer.GetRandomUA())
	cli := &http.Client{Timeout: 5 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Query struct {
			Search []struct {
				Title   string `json:"title"`
				Snippet string `json:"snippet"`
			} `json:"search"`
		} `json:"query"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var res []Result
	for i, item := range data.Query.Search {
		if i >= limit {
			break
		}
		res = append(res, Result{
			Title:   item.Title,
			Snippet: CleanHTML(item.Snippet),
		})
	}
	return res, nil
}

