package search

import "testing"

func TestSearch(t *testing.T) {
	html, err := FetchDDGLite("golang")
	if err != nil {
		t.Fatalf("FetchDDGLite failed: %v", err)
	}
	if len(html) == 0 {
		t.Fatal("Empty html returned from DDG Lite")
	}

	results := ParseDDGLite(html, 3)
	if len(results) == 0 {
		t.Log("Warning: No results parsed")
	} else {
		t.Logf("Parsed %d results", len(results))
		for _, r := range results {
			if r.Title == "" {
				t.Error("Parsed result has empty title")
			}
		}
	}

	wikiResults, err := QueryWikipedia("Go (programming language)", 1)
	if err != nil {
		t.Fatalf("QueryWikipedia failed: %v", err)
	}
	if len(wikiResults) == 0 {
		t.Fatal("No results returned from Wikipedia search")
	}
	if wikiResults[0].Title == "" {
		t.Error("Wikipedia result has empty title")
	}
}
