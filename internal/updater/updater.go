package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type DownloadItem struct {
	Name string `json:"name"`
}

func CheckVersion(current string) (string, bool, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://esoteric.win/api/downloads")
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("bad response: %d", resp.StatusCode)
	}

	var items []DownloadItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return "", false, err
	}

	re := regexp.MustCompile(`skyvern-([0-9]+\.[0-9]+\.[0-9]+)`)

	latest := ""
	for _, item := range items {
		matches := re.FindStringSubmatch(item.Name)
		if len(matches) < 2 {
			continue
		}
		ver := matches[1]
		if latest == "" || compareSemVer(ver, latest) > 0 {
			latest = ver
		}
	}

	if latest == "" {
		return "", false, fmt.Errorf("no versions found on server")
	}

	return latest, compareSemVer(latest, current) > 0, nil
}

func compareSemVer(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")
	pA := strings.Split(a, ".")
	pB := strings.Split(b, ".")

	for i := 0; i < 3; i++ {
		var vA, vB int
		if i < len(pA) {
			vA, _ = strconv.Atoi(pA[i])
		}
		if i < len(pB) {
			vB, _ = strconv.Atoi(pB[i])
		}
		if vA > vB {
			return 1
		}
		if vA < vB {
			return -1
		}
	}
	return 0
}
