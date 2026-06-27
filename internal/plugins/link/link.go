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

var (
	peers = make(map[string]*Peer)
	tasks = make(map[string][]Task)
	mu    sync.RWMutex
	
	// ratelimit trace (remoteIP -> lastRequestTime)
	limits = make(map[string]time.Time)
	limMu  sync.Mutex

	safeRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
)

func Routes() {
	http.HandleFunc("/api/link/ping", handlePing)
	http.HandleFunc("/api/link/peers", handlePeers)
	http.HandleFunc("/api/link/task", handleTask)
}

func checkAuth(r *http.Request) bool {
	key := os.Getenv("SKYVERN_NODE_TOKEN")
	if key == "" {
		return true // skip if not configured
	}
	
	// timing attack prevention
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
			return false // too fast
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
		// drop dead peers after 15s
		if time.Now().Unix()-p.PingTime < 15 {
			list = append(list, p)
		}
	}
	_ = json.NewEncoder(w).Encode(list)
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
		bots, err := db.ListBots()
		if err != nil {
			continue
		}
		
		running := []string{}
		for _, b := range bots {
			if mgr.IsRunning(b.ClientID) {
				running = append(running, b.ClientID)
			}
		}

		payload, _ := json.Marshal(Peer{
			ID:     id,
			Host:   host,
			Bots:   running,
			CPU:    0.8,
			Mem:    18.6,
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
				fmt.Printf("[*] executing task: %s on bot %s\n", t.Action, t.BotID)
				if t.Action == "start" {
					_ = mgr.Start(t.BotID)
				} else if t.Action == "stop" {
					_ = mgr.Stop(t.BotID)
				}
			}
		}
		resp.Body.Close()
	}
}
