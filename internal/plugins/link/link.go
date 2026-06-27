package link

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/storage"
)

type Peer struct {
	ID       string   `json:"id"`
	Host     string   `json:"host"`
	Bots     []string `json:"bots"`
	CPU      float64  `json:"cpu"`
	Mem      float64  `json:"mem"`
	Uptime   int64    `json:"uptime"`
	PingTime int64    `json:"ping_time"`
}

type Task struct {
	Action string `json:"action"` // "start", "stop", "reload"
	BotID  string `json:"bot_id"`
	Extra  string `json:"extra"` // config payloads
}

type LogLine struct {
	BotID   string `json:"bot_id"`
	Message string `json:"message"`
	Level   string `json:"level"`
}

var (
	peers = make(map[string]*Peer)
	tasks = make(map[string][]Task)
	mu    sync.RWMutex
	
	// Centralized Logs Queue
	peerLogs = []LogLine{}
	logMu    sync.Mutex

	// Database reference
	dbRef *storage.DB

	limits = make(map[string]time.Time)
	limMu  sync.Mutex

	safeRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
)

func Routes(db *storage.DB) {
	dbRef = db
	http.HandleFunc("/api/link/ping", handlePing)
	http.HandleFunc("/api/link/peers", handlePeers)
	http.HandleFunc("/api/link/task", handleTask)
	http.HandleFunc("/api/link/config", handleConfig)
	http.HandleFunc("/api/link/logs", handleLogs)
}

func checkAuth(r *http.Request) bool {
	key := os.Getenv("SKYVERN_NODE_TOKEN")
	if key == "" {
		return true
	}
	sent := r.Header.Get("X-Node-Token")
	return subtle.ConstantTimeCompare([]byte(sent), []byte(key)) == 1
}

func limit(r *http.Request) bool {
	limMu.Lock()
	defer limMu.Unlock()
	ip := r.RemoteAddr
	now := time.Now()
	if prev, ok := limits[ip]; ok {
		if now.Sub(prev) < 200*time.Millisecond {
			return false
		}
	}
	limits[ip] = now
	return true
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	if !limit(r) {
		w.WriteHeader(429)
		return
	}
	if !checkAuth(r) {
		w.WriteHeader(401)
		return
	}

	var p Peer
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		w.WriteHeader(400)
		return
	}

	if !safeRegex.MatchString(p.ID) || len(p.ID) > 64 {
		w.WriteHeader(400)
		return
	}

	p.PingTime = time.Now().Unix()

	mu.Lock()
	peers[p.ID] = &p
	pending := tasks[p.ID]
	tasks[p.ID] = nil
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if pending == nil {
		pending = []Task{}
	}
	_ = json.NewEncoder(w).Encode(pending)
}

func handlePeers(w http.ResponseWriter, r *http.Request) {
	if !checkAuth(r) {
		w.WriteHeader(401)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	
	mu.RLock()
	defer mu.RUnlock()
	
	list := make([]*Peer, 0, len(peers))
	for _, p := range peers {
		if time.Now().Unix()-p.PingTime < 15 {
			list = append(list, p)
		}
	}
	_ = json.NewEncoder(w).Encode(list)
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	if !checkAuth(r) {
		w.WriteHeader(401)
		return
	}
	
	botID := r.URL.Query().Get("bot")
	if botID == "" || !safeRegex.MatchString(botID) {
		w.WriteHeader(400)
		return
	}

	if dbRef == nil {
		w.WriteHeader(500)
		return
	}

	bot, err := dbRef.GetBot(botID)
	if err != nil {
		w.WriteHeader(404)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(bot)
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	if !checkAuth(r) {
		w.WriteHeader(401)
		return
	}

	var lines []LogLine
	if err := json.NewDecoder(r.Body).Decode(&lines); err != nil {
		w.WriteHeader(400)
		return
	}

	logMu.Lock()
	peerLogs = append(peerLogs, lines...)
	if len(peerLogs) > 1000 {
		peerLogs = peerLogs[len(peerLogs)-1000:] // cap logs buffer
	}
	logMu.Unlock()

	w.WriteHeader(200)
}

func handleTask(w http.ResponseWriter, r *http.Request) {
	if !checkAuth(r) {
		w.WriteHeader(401)
		return
	}

	peerID := r.URL.Query().Get("peer")
	action := r.URL.Query().Get("action")
	botID := r.URL.Query().Get("bot")
	extra := r.URL.Query().Get("extra")

	if peerID == "" || action == "" || botID == "" {
		w.WriteHeader(400)
		return
	}

	if !safeRegex.MatchString(peerID) || !safeRegex.MatchString(botID) {
		w.WriteHeader(400)
		return
	}
	
	if action != "start" && action != "stop" && action != "reload" {
		w.WriteHeader(400)
		return
	}

	mu.Lock()
	tasks[peerID] = append(tasks[peerID], Task{
		Action: action,
		BotID:  botID,
		Extra:  extra,
	})
	mu.Unlock()

	w.WriteHeader(200)
	_, _ = w.Write([]byte(`{"status":"queued"}`))
}

// ScheduleBot selects the connected peer node with the lowest resource load
func ScheduleBot(botID string) string {
	mu.Lock()
	defer mu.Unlock()

	var bestPeer string
	var minLoad float64 = 99999.0

	for id, p := range peers {
		// Ignore dead peers
		if time.Now().Unix()-p.PingTime >= 15 {
			continue
		}
		
		// Load index: CPU usage + (Mem usage / 10)
		load := p.CPU + (p.Mem / 10.0)
		if load < minLoad {
			minLoad = load
			bestPeer = id
		}
	}

	if bestPeer != "" {
		tasks[bestPeer] = append(tasks[bestPeer], Task{
			Action: "start",
			BotID:  botID,
		})
		return bestPeer
	}

	return ""
}

// PeerLogs returns the logs buffered from remote systems
func GetLogs() []LogLine {
	logMu.Lock()
	defer logMu.Unlock()
	res := make([]LogLine, len(peerLogs))
	copy(res, peerLogs)
	return res
}

func Connect(db *storage.DB, mgr *manager.Manager) {
	url := os.Getenv("SKYVERN_CONTROLLER")
	if url == "" {
		url = "http://localhost:8080"
	}
	host, err := os.Hostname()
	if err != nil {
		host = "unknown-peer"
	}
	id := fmt.Sprintf("peer-%d", time.Now().UnixNano()%100000)
	token := os.Getenv("SKYVERN_NODE_TOKEN")
	start := time.Now()

	fmt.Printf("[*] linking peer [%s] to controller: %s\n", host, url)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	client := &http.Client{Timeout: 5 * time.Second}

	for range ticker.C {
		// Collect running bots from local manager directly
		running := mgr.RunningBots()

		payload, _ := json.Marshal(Peer{
			ID:     id,
			Host:   host,
			Bots:   running,
			CPU:    0.5, // stub metrics
			Mem:    12.2,
			Uptime: int64(time.Since(start).Seconds()),
		})

		req, err := http.NewRequest(http.MethodPost, url+"/api/link/ping", bytes.NewBuffer(payload))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("X-Node-Token", token)
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("[!] ping failed: %v\n", err)
			continue
		}

		var pending []Task
		if err := json.NewDecoder(resp.Body).Decode(&pending); err == nil {
			for _, t := range pending {
				fmt.Printf("[*] task received: %s on bot %s\n", t.Action, t.BotID)
				if t.Action == "start" {
					// 1. Centralized config synchronization - Pull token from controller!
					cfgReq, err := http.NewRequest(http.MethodGet, url+"/api/link/config?bot="+t.BotID, nil)
					if err != nil {
						continue
					}
					if token != "" {
						cfgReq.Header.Set("X-Node-Token", token)
					}
					cfgResp, err := client.Do(cfgReq)
					if err != nil {
						fmt.Printf("[!] config sync failed for bot %s: %v\n", t.BotID, err)
						continue
					}
					
					var b config.BotInst
					if err := json.NewDecoder(cfgResp.Body).Decode(&b); err == nil {
						// save dynamically into in-memory/sqlite copy and spin up
						_ = db.SaveBot(b)
						_ = mgr.Start(t.BotID)
						
						// 2. Log Pipeline: Send confirmation log line back to controller
						logPayload, _ := json.Marshal([]LogLine{
							{BotID: t.BotID, Message: fmt.Sprintf("Bot instance started on peer node [%s]", host), Level: "info"},
						})
						logReq, _ := http.NewRequest(http.MethodPost, url+"/api/link/logs", bytes.NewBuffer(logPayload))
						logReq.Header.Set("Content-Type", "application/json")
						if token != "" {
							logReq.Header.Set("X-Node-Token", token)
						}
						logResp, errLog := client.Do(logReq)
						if errLog == nil {
							logResp.Body.Close()
						}
					}
					cfgResp.Body.Close()
				} else if t.Action == "stop" {
					_ = mgr.Stop(t.BotID)
				}
			}
		}
		resp.Body.Close()
	}
}
