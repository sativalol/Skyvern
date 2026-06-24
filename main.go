package main
import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime/debug"
	"skyvern/internal/bootstrap"
	"skyvern/internal/commands"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/lavalink"
	"skyvern/internal/plugins"
	_ "skyvern/internal/plugins/fun"
	_ "skyvern/internal/plugins/vouch"
	_ "skyvern/internal/plugins/captcha"
	_ "skyvern/internal/plugins/moon"
	_ "skyvern/internal/plugins/economy"
	_ "skyvern/internal/plugins/customcommands"
	"skyvern/internal/storage"
	"strings"
	"skyvern/pkg/tui"
	"skyvern/internal/updater"
	"syscall"
	"time"
)
func main() {
	bootstrap.HandleDumpCmds()

	f := bootstrap.SetupLogger()
	defer f.Close()
	defer func() {
		if r := recover(); r != nil {
			out := fmt.Sprintf("\n[PANIC] %v\n\n%s\n", r, debug.Stack())
			_, _ = fmt.Fprint(f, out)
			f.Close()
			_ = os.Rename(config.ResolvePath("skyvern.log"), config.ResolvePath(fmt.Sprintf("crash_%s.log", time.Now().Format("2006-01-02_15-04-05"))))
			panic(r)
		}
	}()
	if b, err := os.ReadFile(config.ResolvePath("ascii")); err == nil {
		fmt.Print(tui.Shrink(string(b), 2))
	} else {
		fmt.Println(tui.Logo)
	}
	fmt.Printf("  Skyvern | Version %s\n", config.Version)
	go func() {
		if latest, update, err := updater.CheckVersion(config.Version); err == nil && update {
			fmt.Printf("\n  [!] UPDATE AVAILABLE: Version %s is out. Current version: %s\n", latest, config.Version)
			fmt.Println("      Download it from: https://esoteric.win/skyvern/releases\n")
		}
	}()
	fmt.Println("  Loading cfgs...")
	db, err := storage.Open(config.ResolvePath("bots.db"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "db init: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	if savedG, err := db.GetGlobal(); err == nil {
		config.SetGlobal(savedG)
	}
	g := config.GetGlobal()
	if g.LavalinkPass == "" || g.LavalinkPass == "youshallnotpass" {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err == nil {
			g.LavalinkPass = hex.EncodeToString(b)
		} else {
			g.LavalinkPass = "skyvern_rand_pass_fallback_123"
		}
		if err := db.SaveGlobal(g); err != nil {
			fmt.Fprintf(os.Stderr, "[!] Failed to save global config: %v\n", err)
		}
		config.SetGlobal(g)
	}
	if !g.SetupDone {
		fmt.Println("\n  [~] Running setup checks...")
		if _, err := exec.LookPath("java"); err == nil {
			fmt.Println("      Java: Found")
		} else {
			fmt.Println("      Java: Not Found (Lavalink requires Java!)")
		}
		if _, err := os.Stat(config.ResolvePath("lavalink/Lavalink.jar")); err == nil {
			fmt.Println("      Lavalink: Found")
		} else {
			fmt.Println("      Lavalink: Not Found (lavalink/Lavalink.jar missing)")
		}
		rd := bufio.NewReader(os.Stdin)
		fmt.Printf("\n  [?] Enable Lavalink auto-start? (y/n) [current: %t]: ", g.AutoStartLavalink)
		if ans, err := rd.ReadString('\n'); err == nil {
			ans = strings.ToLower(strings.TrimSpace(ans))
			if ans == "y" || ans == "yes" {
				g.AutoStartLavalink = true
			} else if ans == "n" || ans == "no" {
				g.AutoStartLavalink = false
			}
		}
		g.SetupDone = true
		if err := db.SaveGlobal(g); err != nil {
			fmt.Fprintf(os.Stderr, "[!] Failed to save global config: %v\n", err)
		}
		config.SetGlobal(g)
	}
	isLocal := g.LavalinkHost == "" || strings.Contains(g.LavalinkHost, "localhost") || strings.Contains(g.LavalinkHost, "127.0.0.1")
	if g.AutoStartLavalink && isLocal {
		lavalink.StartServer(config.ResolvePath)
		fmt.Print("  Waiting for local Lavalink server to start...")
		start := time.Now()
		client := &http.Client{Timeout: 500 * time.Millisecond}
		for time.Since(start) < 25*time.Second {
			req, err := http.NewRequest("GET", "http://localhost:2333/version", nil)
			if err == nil {
				req.Header.Set("Authorization", g.LavalinkPass)
				resp, err := client.Do(req)
				if err == nil {
					resp.Body.Close()
					if resp.StatusCode == http.StatusOK {
						break
					}
				}
			}
			fmt.Print(".")
			time.Sleep(250 * time.Millisecond)
		}
		fmt.Println(" Done!")
	}
	defer lavalink.StopServer()
	mgr := manager.New(db, commands.Registry)
	defer mgr.Close()
	commands.Init(mgr)
	for _, p := range plugins.Loaded() {
		if err := p.Init(db, mgr); err != nil {
			fmt.Fprintf(os.Stderr, "plugin %s init failed: %v\n", p.Name(), err)
			continue
		}
		mgr.AddCommands(p.Commands())
	}
	headless := false
	for _, arg := range os.Args {
		if arg == "--headless" || arg == "-d" || arg == "--daemon" {
			headless = true
			break
		}
	}

	if list, err := db.ListBots(); err == nil {
		for _, b := range list {
			if b.IsEnabled {
				if err := mgr.Start(b.ClientID); err != nil {
					fmt.Fprintf(os.Stderr, "[!] Failed to start bot %s: %v\n", b.ClientID, err)
				}
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "[!] Failed to list bots: %v\n", err)
	}

	if !headless {
		if err := tui.Run(db, mgr); err != nil {
			fmt.Fprintf(os.Stderr, "[!] TUI exited: %v\n", err)
			fmt.Println("    To run without TUI (headless daemon mode), use: --headless")
		}
		if !mgr.HasRunningBots() {
			return
		}
		fmt.Println("\ntui closed but bots are running in background. ctrl+c to exit")
	} else {
		fmt.Println("\n[*] Running in headless daemon mode. Press Ctrl+C to exit.")
		if !mgr.HasRunningBots() {
			fmt.Println("[!] Warning: No bots are currently enabled. Enable them via TUI first or check bots.db configuration.")
		}
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
