package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"skyvern/internal/manager"
	"skyvern/internal/plugins/link"
	"skyvern/internal/storage"
	"time"
)

//go:embed static/*
var staticFS embed.FS

var (
	dbRef     *storage.DB
	mgrRef    *manager.Manager
	startTime time.Time
)

func StartServer(db *storage.DB, mgr *manager.Manager) {
	dbRef = db
	mgrRef = mgr
	startTime = time.Now()

	portStr := os.Getenv("SKYVERN_WEB_PORT")
	if portStr == "" {
		portStr = "8080"
	}

	// Serve static files (style.css, app.js)
	http.Handle("/static/", http.FileServer(http.FS(staticFS)))

	http.HandleFunc("/", handleHome)
	http.HandleFunc("/api/bots", handleGetBots)
	http.HandleFunc("/api/bot/start", handleStartBot)
	http.HandleFunc("/api/bot/stop", handleStopBot)
	http.HandleFunc("/api/stats", handleStats)
	
	// Register peer synchronization routes
	link.Routes(db)

	go func() {
		fmt.Printf("  [+] Starting Web Dashboard on http://localhost:%s\n", portStr)
		if err := http.ListenAndServe(":"+portStr, nil); err != nil {
			fmt.Printf("  [!] Web Dashboard failed to start: %v\n", err)
		}
	}()
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "Failed to load index.html", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data)
}

func handleGetBots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	bots, err := dbRef.ListBots()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type webBot struct {
		ClientID   string `json:"client_id"`
		CustomName string `json:"custom_name"`
		IsEnabled  bool   `json:"is_enabled"`
		Running    bool   `json:"running"`
		LastErr    string `json:"last_err"`
	}

	var list []webBot
	for _, b := range bots {
		list = append(list, webBot{
			ClientID:   b.ClientID,
			CustomName: b.CustomName,
			IsEnabled:  b.IsEnabled,
			Running:    mgrRef.IsRunning(b.ClientID),
			LastErr:    mgrRef.LastErr(b.ClientID),
		})
	}

	_ = json.NewEncoder(w).Encode(list)
}

func handleStartBot(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	// 3. Load Balancing - Try scheduling remotely first if peer is connected
	if peerID := link.ScheduleBot(id); peerID != "" {
		if bot, errGet := dbRef.GetBot(id); errGet == nil {
			bot.IsEnabled = true
			_ = dbRef.SaveBot(bot)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","peer":"` + peerID + `"}`))
		return
	}

	err := mgrRef.Start(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if bot, errGet := dbRef.GetBot(id); errGet == nil {
		bot.IsEnabled = true
		_ = dbRef.SaveBot(bot)
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func handleStopBot(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	err := mgrRef.Stop(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if bot, errGet := dbRef.GetBot(id); errGet == nil {
		bot.IsEnabled = false
		_ = dbRef.SaveBot(bot)
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	bots, _ := dbRef.ListBots()
	activeCount := 0
	for _, b := range bots {
		if mgrRef.IsRunning(b.ClientID) {
			activeCount++
		}
	}

	var totalCmds int64 = 0
	for _, b := range bots {
		stats := mgrRef.Stats(b.ClientID)
		totalCmds += stats.TotalCmds
	}

	stats := map[string]interface{}{
		"uptime_seconds": int(time.Since(startTime).Seconds()),
		"active_bots":    activeCount,
		"total_bots":     len(bots),
		"total_commands": totalCmds,
	}

	_ = json.NewEncoder(w).Encode(stats)
}
