package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"skyvern/internal/manager"
	"skyvern/internal/storage"
	"time"
)

var (
	dbRef  *storage.DB
	mgrRef *manager.Manager
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

	http.HandleFunc("/", handleHome)
	http.HandleFunc("/api/bots", handleGetBots)
	http.HandleFunc("/api/bot/start", handleStartBot)
	http.HandleFunc("/api/bot/stop", handleStopBot)
	http.HandleFunc("/api/stats", handleStats)

	go func() {
		fmt.Printf("  [+] Starting Web Dashboard on http://localhost:%s\n", portStr)
		if err := http.ListenAndServe(":"+portStr, nil); err != nil {
			fmt.Printf("  [!] Web Dashboard failed to start: %v\n", err)
		}
	}()
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(htmlContent))
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

const htmlContent = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Skyvern | Web Dashboard</title>
    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;600;800&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg: #0d0f14;
            --card-bg: rgba(22, 28, 38, 0.6);
            --accent: #bd93f9;
            --accent-glow: rgba(189, 147, 249, 0.3);
            --text: #f8f8f2;
            --subtle: #9ba4b4;
            --green: #50fa7b;
            --red: #ff5555;
            --border: rgba(255, 255, 255, 0.05);
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
            user-select: none;
        }

        body {
            font-family: 'Outfit', sans-serif;
            background-color: var(--bg);
            color: var(--text);
            overflow-x: hidden;
            background-image: radial-gradient(circle at 10% 20%, rgba(189, 147, 249, 0.08) 0%, transparent 40%),
                              radial-gradient(circle at 90% 80%, rgba(80, 250, 123, 0.05) 0%, transparent 40%);
            min-height: 100vh;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 2rem 1.5rem;
        }

        header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 3rem;
            border-bottom: 1px solid var(--border);
            padding-bottom: 1.5rem;
        }

        h1 {
            font-size: 2.2rem;
            font-weight: 800;
            background: linear-gradient(135deg, #ffffff 0%, var(--accent) 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            letter-spacing: -1px;
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
            gap: 1.5rem;
            margin-bottom: 3rem;
        }

        .stat-card {
            background: var(--card-bg);
            backdrop-filter: blur(12px);
            border: 1px solid var(--border);
            border-radius: 16px;
            padding: 1.5rem;
            box-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.3);
            transition: transform 0.3s ease, border-color 0.3s ease;
        }

        .stat-card:hover {
            transform: translateY(-4px);
            border-color: rgba(189, 147, 249, 0.2);
        }

        .stat-label {
            color: var(--subtle);
            font-size: 0.9rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 1px;
            margin-bottom: 0.5rem;
        }

        .stat-value {
            font-size: 2rem;
            font-weight: 800;
            color: #ffffff;
        }

        .bots-section {
            background: var(--card-bg);
            backdrop-filter: blur(12px);
            border: 1px solid var(--border);
            border-radius: 20px;
            padding: 2rem;
            box-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.3);
        }

        .section-title {
            font-size: 1.5rem;
            font-weight: 600;
            margin-bottom: 1.5rem;
            display: flex;
            align-items: center;
            justify-content: space-between;
        }

        .bot-list {
            display: flex;
            flex-direction: column;
            gap: 1rem;
        }

        .bot-row {
            display: flex;
            justify-content: space-between;
            align-items: center;
            background: rgba(255, 255, 255, 0.02);
            border: 1px solid var(--border);
            padding: 1.2rem 1.5rem;
            border-radius: 12px;
            transition: background 0.2s ease, border-color 0.2s ease;
        }

        .bot-row:hover {
            background: rgba(255, 255, 255, 0.04);
            border-color: rgba(255, 255, 255, 0.1);
        }

        .bot-info {
            display: flex;
            align-items: center;
            gap: 1rem;
        }

        .status-dot {
            width: 12px;
            height: 12px;
            border-radius: 50%;
            display: inline-block;
            box-shadow: 0 0 10px rgba(0,0,0,0.5);
        }

        .status-dot.running {
            background-color: var(--green);
            box-shadow: 0 0 12px var(--green);
            animation: pulse 2s infinite;
        }

        .status-dot.stopped {
            background-color: var(--subtle);
        }

        .status-dot.error {
            background-color: var(--red);
            box-shadow: 0 0 12px var(--red);
        }

        .bot-details {
            display: flex;
            flex-direction: column;
        }

        .bot-name {
            font-weight: 600;
            font-size: 1.1rem;
            color: #ffffff;
        }

        .bot-id {
            font-size: 0.8rem;
            color: var(--subtle);
            font-family: monospace;
            margin-top: 0.2rem;
        }

        .bot-error {
            font-size: 0.8rem;
            color: var(--red);
            margin-top: 0.2rem;
        }

        .btn {
            background: rgba(189, 147, 249, 0.1);
            border: 1px solid var(--accent);
            color: var(--text);
            padding: 0.6rem 1.2rem;
            border-radius: 8px;
            cursor: pointer;
            font-weight: 600;
            font-size: 0.9rem;
            transition: all 0.2s ease;
            outline: none;
        }

        .btn:hover {
            background: var(--accent);
            color: var(--bg);
            box-shadow: 0 0 15px var(--accent-glow);
        }

        .btn.stop {
            border-color: var(--red);
            background: rgba(255, 85, 85, 0.1);
        }

        .btn.stop:hover {
            background: var(--red);
            color: #ffffff;
            box-shadow: 0 0 15px rgba(255, 85, 85, 0.3);
        }

        @keyframes pulse {
            0% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(80, 250, 123, 0.7); }
            70% { transform: scale(1); box-shadow: 0 0 0 8px rgba(80, 250, 123, 0); }
            100% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(80, 250, 123, 0); }
        }

        @media (max-width: 600px) {
            .bot-row {
                flex-direction: column;
                align-items: flex-start;
                gap: 1rem;
            }
            .btn {
                width: 100%;
                text-align: center;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>⚡ SKYVERN <span style="font-size: 1rem; color: var(--accent); font-weight: 400; border: 1px solid var(--border); padding: 2px 8px; border-radius: 12px; vertical-align: middle;">Web UI</span></h1>
            <div id="connection-status" style="font-size: 0.9rem; color: var(--green);">● Connected</div>
        </header>

        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-label">Active Instances</div>
                <div id="stat-active" class="stat-value">0</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Total Commands</div>
                <div id="stat-cmds" class="stat-value">0</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Uptime</div>
                <div id="stat-uptime" class="stat-value">0s</div>
            </div>
        </div>

        <div class="bots-section">
            <div class="section-title">
                <span>Bot Instances</span>
                <span id="bot-count" style="font-size: 0.9rem; color: var(--subtle); font-weight: 400;">0 loaded</span>
            </div>
            <div id="bot-list" class="bot-list">
                <!-- Loaded dynamically -->
            </div>
        </div>
    </div>

    <script>
        async function fetchStats() {
            try {
                const res = await fetch('/api/stats');
                const data = await res.json();
                document.getElementById('stat-active').innerText = data.active_bots + ' / ' + data.total_bots;
                document.getElementById('stat-cmds').innerText = data.total_commands;
                
                const sec = data.uptime_seconds;
                const h = Math.floor(sec / 3600);
                const m = Math.floor((sec % 3600) / 60);
                const s = sec % 60;
                document.getElementById('stat-uptime').innerText = 
                    (h > 0 ? h + 'h ' : '') + (m > 0 ? m + 'm ' : '') + s + 's';
            } catch (err) {
                console.error("stats fetch failed", err);
            }
        }

        async function fetchBots() {
            try {
                const res = await fetch('/api/bots');
                const bots = await res.json();
                document.getElementById('bot-count').innerText = bots.length + ' loaded';
                
                const container = document.getElementById('bot-list');
                container.innerHTML = '';
                
                bots.forEach(bot => {
                    const row = document.createElement('div');
                    row.className = 'bot-row';
                    
                    let statusClass = 'stopped';
                    if (bot.running) {
                        statusClass = 'running';
                    } else if (bot.last_err) {
                        statusClass = 'error';
                    }
                    
                    const name = bot.custom_name ? bot.custom_name : 'Unnamed Bot';
                    const btnClass = bot.running ? 'stop' : 'start';
                    const btnText = bot.running ? 'Stop Bot' : 'Start Bot';
                    
                    const errHtml = bot.last_err ? '<span class="bot-error">Error: ' + bot.last_err + '</span>' : '';
                    
                    row.innerHTML = 
                        '<div class="bot-info">' +
                            '<span class="status-dot ' + statusClass + '"></span>' +
                            '<div class="bot-details">' +
                                '<span class="bot-name">' + name + '</span>' +
                                '<span class="bot-id">Client ID: ' + bot.client_id + '</span>' +
                                errHtml +
                            '</div>' +
                        '</div>' +
                        '<div>' +
                            '<button class="btn ' + btnClass + '" onclick="toggleBot(\'' + bot.client_id + '\', ' + bot.running + ')">' +
                                btnText +
                            '</button>' +
                        '</div>';
                    container.appendChild(row);
                });
            } catch (err) {
                console.error("bots fetch failed", err);
            }
        }

        async function toggleBot(id, running) {
            const endpoint = running ? '/api/bot/stop' : '/api/bot/start';
            try {
                const res = await fetch(endpoint + '?id=' + id);
                if (res.ok) {
                    fetchBots();
                    fetchStats();
                } else {
                    alert("Action failed!");
                }
            } catch (err) {
                alert("Request error: " + err);
            }
        }

        setInterval(() => {
            fetchStats();
            fetchBots();
        }, 3000);

        fetchStats();
        fetchBots();
    </script>
</body>
</html>
`
