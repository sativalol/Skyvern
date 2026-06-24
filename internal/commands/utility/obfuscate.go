package utility

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"skyvern/internal/config"
	"skyvern/internal/downloader"
	"skyvern/internal/manager"
	"skyvern/internal/obfuscator"
	"strconv"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type obfSession struct {
	uid    string
	code   string
	file   string
	preset string
	pid    int
	at     bool
	dvm    bool
	song   string
	lyrics string
}

var (
	obfSessionsMu sync.Mutex
	obfSessions   = make(map[string]*obfSession)
)

func genObfSessionID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func init() {
	manager.RegisterHelp("obfuscate", []manager.HelpPage{
		{
			Command:     "Obfuscate",
			Syntax:      ".obfuscate [options] or .obf [options]",
			Description: "Obfuscate a Lua or LuaU script with interactive parameters. Attach the file or reply to a message containing it.",
		},
	})
}

var Obfuscate = &manager.Command{
	Trigger:     "obfuscate",
	Aliases:     []string{"obf"},
	Name:        "obfuscate",
	Description: "Obfuscate Lua/LuaU files",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		var scriptContent string
		var filename = "script.lua"
		var hasSource bool

		if ctx.Message != nil && len(ctx.Message.Attachments) > 0 {
			att := ctx.Message.Attachments[0]
			if att.Size > 5*1024*1024 {
				return ctx.Reply("[!] Attached file is too large (max 5MB).")
			}
			resp, err := http.Get(att.URL)
			if err == nil {
				defer resp.Body.Close()
				b, _ := io.ReadAll(resp.Body)
				scriptContent = string(b)
				filename = att.Filename
				hasSource = true
			}
		}

		if !hasSource && ctx.Message != nil && ctx.Message.ReferencedMessage != nil {
			if len(ctx.Message.ReferencedMessage.Attachments) > 0 {
				att := ctx.Message.ReferencedMessage.Attachments[0]
				if att.Size <= 5*1024*1024 {
					resp, err := http.Get(att.URL)
					if err == nil {
						defer resp.Body.Close()
						b, _ := io.ReadAll(resp.Body)
						scriptContent = string(b)
						filename = att.Filename
						hasSource = true
					}
				}
			} else if ctx.Message.ReferencedMessage.Content != "" {
				referencedURL := strings.TrimSpace(ctx.Message.ReferencedMessage.Content)
				if !strings.HasPrefix(referencedURL, "http://") && !strings.HasPrefix(referencedURL, "https://") {
					scriptContent = ctx.Message.ReferencedMessage.Content
					hasSource = true
				}
			}
		}

		var scriptURL string
		if !hasSource {
			for _, arg := range ctx.Args {
				if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
					scriptURL = arg
					break
				}
			}
			if scriptURL != "" {
				if strings.Contains(scriptURL, "github.com") && strings.Contains(scriptURL, "/blob/") {
					scriptURL = strings.Replace(scriptURL, "github.com", "raw.githubusercontent.com", 1)
					scriptURL = strings.Replace(scriptURL, "/blob/", "/", 1)
				} else if strings.Contains(scriptURL, "pastebin.com") && !strings.Contains(scriptURL, "/raw/") {
					scriptURL = strings.Replace(scriptURL, "pastebin.com/", "pastebin.com/raw/", 1)
				}

				resp, err := http.Get(scriptURL)
				if err == nil {
					defer resp.Body.Close()
					b, err := io.ReadAll(resp.Body)
					if err == nil {
						scriptContent = string(b)
						hasSource = true
						parts := strings.Split(scriptURL, "/")
						baseName := parts[len(parts)-1]
						if strings.Contains(baseName, "?") {
							baseName = strings.Split(baseName, "?")[0]
						}
						if baseName == "" || !strings.Contains(baseName, ".") {
							filename = "script.lua"
						} else {
							filename = baseName
						}
					}
				}
			}
		}

		if !hasSource && len(ctx.Args) > 0 {
			scriptContent = strings.Join(ctx.Args, " ")
			hasSource = true
		}

		if !hasSource || strings.TrimSpace(scriptContent) == "" {
			return ctx.Reply("[!] No script provided. Attach a Lua/LuaU file, reply to one, paste a script link, or provide the code text.")
		}

		preset := "Vmify"
		pid := 0
		at := true
		dvm := false
		song := "random"
		var lyrics string

		if len(ctx.Args) > 0 && (len(ctx.Message.Attachments) > 0 || (ctx.Message.ReferencedMessage != nil) || scriptURL != "") {
			for i := 0; i < len(ctx.Args); i++ {
				arg := ctx.Args[i]
				clean := strings.ToLower(arg)
				for strings.HasPrefix(clean, "-") {
					clean = clean[1:]
				}

				if clean == "noat" || clean == "no-at" || clean == "noantitamper" || clean == "no-antitamper" {
					at = false
				} else if clean == "dvm" || clean == "doublevm" || clean == "double-vm" {
					dvm = true
				} else if clean == "lyrics" || clean == "song" {
					if i+1 < len(ctx.Args) {
						next := ctx.Args[i+1]
						if !strings.HasPrefix(next, "-") {
							if strings.Contains(next, "|") {
								parts := strings.Split(next, "|")
								var lines []string
								for _, p := range parts {
									pClean := strings.TrimSpace(p)
									if pClean != "" {
										lines = append(lines, pClean)
									}
								}
								lyrics = strings.Join(lines, "\n")
								i++
							} else {
								song = strings.ToLower(next)
								i++
							}
						}
					}
				} else {
					val, err := strconv.Atoi(clean)
					if err == nil && !strings.Contains(clean, ".") && !strings.Contains(clean, "/") {
						pid = val
					} else {
						knownPresets := []string{"minify", "compress", "weak", "vmify", "medium", "strong"}
						for _, kp := range knownPresets {
							if kp == clean {
								preset = strings.Title(clean)
								break
							}
						}
					}
				}
			}
		}

		cleanedFile := filepath.Base(filename)
		cleanedFile = strings.ReplaceAll(cleanedFile, "..", "")
		cleanedFile = strings.ReplaceAll(cleanedFile, "/", "")
		cleanedFile = strings.ReplaceAll(cleanedFile, "\\", "")
		if cleanedFile == "" || cleanedFile == "." {
			cleanedFile = "script.lua"
		}

		sid := genObfSessionID()
		sess := &obfSession{
			uid:    ctx.AuthorID(),
			code:   scriptContent,
			file:   cleanedFile,
			preset: preset,
			pid:    pid,
			at:     at,
			dvm:    dvm,
			song:   song,
			lyrics: lyrics,
		}

		obfSessionsMu.Lock()
		obfSessions[sid] = sess
		obfSessionsMu.Unlock()

		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "Lyrics Obfuscation",
			Description: "Would you like to enable lyrics obfuscation for this script?",
		})

		components := []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Yes",
						Style:    discordgo.SecondaryButton,
						CustomID: "obf_lyrics:yes:" + sid,
					},
					discordgo.Button{
						Label:    "No",
						Style:    discordgo.SecondaryButton,
						CustomID: "obf_lyrics:no:" + sid,
					},
				},
			},
		}

		if ctx.Interact != nil {
			return ctx.Session.InteractionRespond(ctx.Interact, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds:     []*discordgo.MessageEmbed{emb},
					Components: components,
				},
			})
		}
		_, err := ctx.Session.ChannelMessageSendComplex(ctx.ChanID(), &discordgo.MessageSend{
			Embeds:     []*discordgo.MessageEmbed{emb},
			Components: components,
		})
		return err
	},
}

func updateStatus(s *discordgo.Session, i *discordgo.Interaction, cfg config.ResCfg, text string) {
	emb := config.Wrap(cfg, text)
	_, _ = s.InteractionResponseEdit(i, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{emb},
	})
}

func HandleObfuscateLyricsComponent(s *discordgo.Session, i *discordgo.InteractionCreate, mgr *manager.Manager) {
	cid := i.MessageComponentData().CustomID
	parts := strings.Split(cid, ":")
	if len(parts) < 3 {
		return
	}
	choice := parts[1]
	sid := parts[2]

	obfSessionsMu.Lock()
	sess, exists := obfSessions[sid]
	if exists {
		delete(obfSessions, sid)
	}
	obfSessionsMu.Unlock()

	bCfg, _ := mgr.DB().GetBot(s.State.User.ID)
	cfg := config.Resolve(config.GetGlobal(), bCfg)
	emoji := mgr.ResolveEmoji(s, i.GuildID, "sys_x")

	if !exists {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("%s Interactive session expired.", emoji),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if i.Member == nil || i.Member.User == nil || i.Member.User.ID != sess.uid {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("%s This is not your interactive session.", emoji),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		obfSessionsMu.Lock()
		obfSessions[sid] = sess
		obfSessionsMu.Unlock()
		return
	}

	emb := config.Wrap(cfg, "[*] Running obfuscator, please wait...")
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{emb},
			Components: []discordgo.MessageComponent{},
		},
	})

	go func() {
		tmp := filepath.Join(".", "internal", "obfuscator", "temp")
		_ = os.MkdirAll(tmp, 0755)

		inPath := filepath.Join(tmp, fmt.Sprintf("in_%s", sess.file))
		minPath := filepath.Join(tmp, fmt.Sprintf("min_%s", sess.file))
		outPath := filepath.Join(tmp, fmt.Sprintf("obf_%s", sess.file))

		err := os.WriteFile(inPath, []byte(sess.code), 0644)
		if err != nil {
			updateStatus(s, i.Interaction, cfg, fmt.Sprintf("[!] Failed to write temp input: %v", err))
			return
		}
		defer os.Remove(inPath)
		defer os.Remove(minPath)
		defer os.Remove(outPath)

		dir := filepath.Join(".", "internal", "obfuscator")
		var luaBin string

		dlURL, filename, archiveType := getLuaURL()
		var candidates []string
		if filename != "" {
			candidates = append(candidates, filepath.Join(dir, "bin", filename))
		}
		candidates = append(candidates,
			filepath.Join(dir, "bin", "lua5.1.exe"),
			filepath.Join(dir, "bin", "lua5.1"),
			"lua5.1",
			"lua51",
			"luajit",
			"lua",
		)

		for _, c := range candidates {
			if strings.Contains(c, string(filepath.Separator)) {
				if _, err := os.Stat(c); err == nil {
					luaBin = c
					break
				}
			} else {
				if p, err := exec.LookPath(c); err == nil {
					luaBin = p
					break
				}
			}
		}

		if luaBin == "" && dlURL != "" {
			destDir := filepath.Join(dir, "bin")
			_ = os.MkdirAll(destDir, 0755)
			dlPath := filepath.Join(destDir, filename)
			if _, err := os.Stat(dlPath); os.IsNotExist(err) {
				updateStatus(s, i.Interaction, cfg, fmt.Sprintf("[*] Lua not found. Downloading static Lua 5.1 (%s)...", filename))
				var dlErr error
				if archiveType == "tgz" {
					dlErr = downloader.ExtractTgzFile(dlURL, filename, dlPath)
				} else if archiveType == "zip" {
					dlErr = downloader.ExtractZipToDir(dlURL, destDir)
				} else {
					dlErr = downloader.Download(dlURL, dlPath)
				}
				if dlErr != nil {
					updateStatus(s, i.Interaction, cfg, fmt.Sprintf("[!] Failed to download/extract Lua: %v", dlErr))
					return
				}
				_ = os.Chmod(dlPath, 0755)
			}
			luaBin = dlPath
		}

		if luaBin == "" {
			luaBin = "lua"
		}


		absDir, err := filepath.Abs(dir)
		if err != nil {
			updateStatus(s, i.Interaction, cfg, fmt.Sprintf("[!] Failed to get absolute path of base directory: %v", err))
			return
		}

		if err := obfuscator.RestoreTools(absDir); err != nil {
			updateStatus(s, i.Interaction, cfg, fmt.Sprintf("[!] Failed to restore obfuscator tools: %v", err))
			return
		}

		absLua := luaBin
		if _, err := os.Stat(luaBin); err == nil {
			absLua, _ = filepath.Abs(luaBin)
		}

		args := []string{
			filepath.Join("tools", "obf", "prometheus", "cli.lua"),
			filepath.Join("temp", fmt.Sprintf("in_%s", sess.file)),
			"--preset", sess.preset,
			"--LuaU",
			"--out", filepath.Join("temp", fmt.Sprintf("min_%s", sess.file)),
		}

		c := exec.Command(absLua, args...)
		c.Dir = absDir
		if out, err := c.CombinedOutput(); err != nil {
			updateStatus(s, i.Interaction, cfg, fmt.Sprintf("[!] Compilation phase failed: %s (error: %v)", string(out), err))
			return
		}

		minCode, err := os.ReadFile(minPath)
		if err != nil {
			updateStatus(s, i.Interaction, cfg, fmt.Sprintf("[!] Failed to read minified code: %v", err))
			return
		}

		var at string
		if sess.at {
			atFile := filepath.Join(dir, "tools", "obf", "esoteric", "antitamper")
			atData, err := os.ReadFile(atFile)
			if err != nil {
				updateStatus(s, i.Interaction, cfg, fmt.Sprintf("[!] Failed to read antitamper template: %v", err))
				return
			}
			at = string(atData)
		}

		code, err := obfuscator.Obfuscate(string(minCode), at, obfuscator.Opts{
			PlaceID:      sess.pid,
			UseDoubleVM:  sess.dvm,
			UseLyrics:    (choice == "yes"),
			CustomLyrics: sess.lyrics,
			SelectedSong: sess.song,
		})
		if err != nil {
			updateStatus(s, i.Interaction, cfg, fmt.Sprintf("[!] Obfuscation VM phase failed: %v", err))
			return
		}

		err = os.WriteFile(outPath, []byte(code), 0644)
		if err != nil {
			updateStatus(s, i.Interaction, cfg, fmt.Sprintf("[!] Failed to write output file: %v", err))
			return
		}

		updateStatus(s, i.Interaction, cfg, "[+] Obfuscation complete. Sending file...")

		f, err := os.Open(outPath)
		if err != nil {
			_, _ = s.ChannelMessageSend(i.ChannelID, fmt.Sprintf("[!] Failed to open obfuscated file for upload: %v", err))
			return
		}
		defer f.Close()

		_, err = s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
			Content: "[+] Obfuscated script file:",
			Files: []*discordgo.File{
				{
					Name:   fmt.Sprintf("obfuscated_%s", sess.file),
					Reader: f,
				},
			},
		})
	}()
}

func getLuaURL() (string, string, string) {
	osVal := runtime.GOOS
	archVal := runtime.GOARCH
	switch osVal {
	case "windows":
		if archVal == "386" {
			return "https://downloads.sourceforge.net/project/luabinaries/5.1.5/Tools%20Executables/lua-5.1.5_Win32_bin.zip", "lua5.1.exe", "zip"
		}
		return "https://downloads.sourceforge.net/project/luabinaries/5.1.5/Tools%20Executables/lua-5.1.5_Win64_bin.zip", "lua5.1.exe", "zip"
	case "linux":
		if archVal == "arm64" {
			return "https://github.com/pocomane/lua_static_battery/releases/download/v0.1-rc/lua_static_battery_arm_linux.tar.gz", "lua_static_battery.exe", "tgz"
		}
		return "https://github.com/pocomane/lua_static_battery/releases/download/v0.1-rc/lua_static_battery_linux.tar.gz", "lua_static_battery.exe", "tgz"
	case "android":
		return "https://github.com/pocomane/lua_static_battery/releases/download/v0.1-rc/lua_static_battery_arm_linux.tar.gz", "lua_static_battery.exe", "tgz"
	case "darwin":
		if archVal == "arm64" {
			return "https://github.com/dyne/luabinaries/releases/download/54f813a/lua51-macos-arm64", "lua51-macos-arm64", "raw"
		}
		return "https://github.com/dyne/luabinaries/releases/download/54f813a/lua51-macos-x64", "lua51-macos-x64", "raw"
	}
	return "", "", ""
}



