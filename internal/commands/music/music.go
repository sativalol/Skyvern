package music

import (
	"encoding/json"
	"fmt"
	"net/http"
	"skyvern/internal/config"
	"skyvern/internal/lavalink"
	"skyvern/internal/manager"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func init() {
	manager.RegisterHelp("play", []manager.HelpPage{
		{
			Command:     "Play Track",
			Syntax:      ".play <url_or_search_query>",
			Description: "Play a song from YouTube, Spotify, or SoundCloud, or search for it.",
		},
		{
			Command:     "Play Next",
			Syntax:      ".play next <url_or_search_query>",
			Description: "Queue a track to play immediately after the current song finishes.",
		},
	})
	manager.RegisterHelp("fastforward", []manager.HelpPage{
		{
			Command:     "Fastforward",
			Syntax:      ".fastforward <seconds>",
			Description: "Fastforward the current song by a specified relative offset (in seconds or timestamp format).",
		},
	})
	manager.RegisterHelp("rewind", []manager.HelpPage{
		{
			Command:     "Rewind",
			Syntax:      ".rewind <seconds>",
			Description: "Rewind the current song by a specified relative offset (in seconds or timestamp format).",
		},
	})
	manager.RegisterHelp("preset", []manager.HelpPage{
		{
			Command:     "Preset Active",
			Syntax:      ".preset active",
			Description: "View the currently running equalizer or filter preset.",
		},
		{
			Command:     "Preset Configure",
			Syntax:      ".preset <name> [on/off]",
			Description: "Configure an audio equalizer filter (vaporwave, nightcore, boost, flat, chipmunk, piano, metal, vibrato, 8d, soft).",
		},
	})
	manager.RegisterHelp("seek", []manager.HelpPage{
		{
			Command:     "Seek",
			Syntax:      ".seek <timestamp/seconds>",
			Description: "Seek to a timestamp (e.g. 1:30 or 90) in the current song.",
		},
	})
}

var Play = &manager.Command{
	Trigger:     "play",
	Aliases:     []string{"p"},
	Name:        "play",
	Description: "Play a song from YouTube, Spotify, or SoundCloud",
	Category:    "music",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("play")
		}

		l := ctx.Mgr.GetLavalink(ctx.ClientID)
		if l == nil {
			return ctx.Reply("[!] Music player not initialized.")
		}

		vcID := findUserVC(ctx.Session, ctx.Message.GuildID, ctx.Message.Author.ID)
		if vcID == "" {
			return ctx.Reply("[!] You must be in a voice channel to play music.")
		}

		err := lavalink.SendVoiceStateUpdate(ctx.Session, ctx.Message.GuildID, vcID, false, false)
		if err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Failed to join voice channel: %v", err))
		}

		playNext := false
		args := ctx.Args
		if len(args) > 1 && strings.ToLower(args[0]) == "next" {
			playNext = true
			args = args[1:]
		} else if len(args) == 1 && strings.ToLower(args[0]) == "next" {
			return ctx.Reply("Usage: `.play next <url_or_search_query>`")
		}

		q := strings.Join(args, " ")
		tl, err := l.LoadTracks(q)
		if err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Failed to load tracks: %v", err))
		}

		tracks, playlistName, err := l.ParseLoadTracks(tl)
		if err != nil || len(tracks) == 0 {
			return ctx.Reply("[!] No tracks found.")
		}

		for i := range tracks {
			tracks[i].Requester = ctx.Message.Author.ID
		}
		p := l.GetPlayer(ctx.Message.GuildID)
		if playNext {
			p.AddNext(tracks, ctx.Message.ChannelID)
		} else {
			p.Add(tracks, ctx.Message.ChannelID)
		}

		if playlistName != "" {
			return ctx.Reply(fmt.Sprintf("[+] Loaded playlist **%s** with %d tracks.", playlistName, len(tracks)))
		}
		if len(tracks) > 1 {
			return ctx.Reply(fmt.Sprintf("[+] Queued %d tracks.", len(tracks)))
		}
		return ctx.Reply(fmt.Sprintf("[+] Queued **%s**.", tracks[0].Info.Title))
	},
}

var Stop = &manager.Command{
	Trigger:     "stop",
	Aliases:     []string{"dc", "leave", "disconnect"},
	Name:        "stop",
	Description: "Stop music playback, clear queue, and leave voice channel",
	Category:    "music",
	Execute: func(ctx *manager.CommandContext) error {
		l := ctx.Mgr.GetLavalink(ctx.ClientID)
		if l == nil {
			return nil
		}
		p := l.GetPlayer(ctx.Message.GuildID)
		_ = p.Stop()
		l.RemovePlayer(ctx.Message.GuildID)
		return ctx.Reply("[+] Stopped playback and left voice channel.")
	},
}

var Pause = &manager.Command{
	Trigger:     "pause",
	Name:        "pause",
	Description: "Pause music playback",
	Category:    "music",
	Execute: func(ctx *manager.CommandContext) error {
		l := ctx.Mgr.GetLavalink(ctx.ClientID)
		if l == nil {
			return nil
		}
		p := l.GetPlayer(ctx.Message.GuildID)
		if err := p.Pause(true); err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Failed to pause: %v", err))
		}
		return ctx.Reply("[+] Paused.")
	},
}

var Resume = &manager.Command{
	Trigger:     "resume",
	Aliases:     []string{"unpause"},
	Name:        "resume",
	Description: "Resume music playback",
	Category:    "music",
	Execute: func(ctx *manager.CommandContext) error {
		l := ctx.Mgr.GetLavalink(ctx.ClientID)
		if l == nil {
			return nil
		}
		p := l.GetPlayer(ctx.Message.GuildID)
		if err := p.Pause(false); err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Failed to resume: %v", err))
		}
		return ctx.Reply("[+] Resumed.")
	},
}

var Skip = &manager.Command{
	Trigger:     "skip",
	Aliases:     []string{"s", "next"},
	Name:        "skip",
	Description: "Skip the current song",
	Category:    "music",
	Execute: func(ctx *manager.CommandContext) error {
		l := ctx.Mgr.GetLavalink(ctx.ClientID)
		if l == nil {
			return nil
		}
		p := l.GetPlayer(ctx.Message.GuildID)
		if err := p.Skip(); err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Failed to skip: %v", err))
		}
		return ctx.Reply("[+] Skipped.")
	},
}

var Queue = &manager.Command{
	Trigger:     "queue",
	Aliases:     []string{"q"},
	Name:        "queue",
	Description: "Display the current track queue",
	Category:    "music",
	Execute: func(ctx *manager.CommandContext) error {
		l := ctx.Mgr.GetLavalink(ctx.ClientID)
		if l == nil {
			return nil
		}
		p := l.GetPlayer(ctx.Message.GuildID)

		if len(ctx.Args) > 0 {
			sub := strings.ToLower(ctx.Args[0])
			switch sub {
			case "remove":
				if len(ctx.Args) < 2 {
					return ctx.Reply("Usage: `.queue remove <position>`")
				}
				pos, err := strconv.Atoi(ctx.Args[1])
				if err != nil || pos <= 0 {
					return ctx.Reply("[!] Invalid position. It must be a positive number.")
				}
				track, err := p.RemovePosition(pos)
				if err != nil {
					return ctx.Reply(fmt.Sprintf("[!] Failed to remove track: %v", err))
				}
				return ctx.Reply(fmt.Sprintf("[+] Removed **%s** from the queue.", track.Info.Title))

			case "shuffle":
				p.Shuffle()
				return ctx.Reply("[+] Shuffled the queue.")

			case "empty", "clear":
				p.Clear()
				return ctx.Reply("[+] Cleared the queue.")

			case "move":
				if len(ctx.Args) < 3 {
					return ctx.Reply("Usage: `.queue move <position> <new_position>`")
				}
				from, err1 := strconv.Atoi(ctx.Args[1])
				to, err2 := strconv.Atoi(ctx.Args[2])
				if err1 != nil || err2 != nil || from <= 0 || to <= 0 {
					return ctx.Reply("[!] Invalid positions. They must be positive numbers.")
				}
				err := p.MovePosition(from, to)
				if err != nil {
					return ctx.Reply(fmt.Sprintf("[!] Failed to move track: %v", err))
				}
				return ctx.Reply(fmt.Sprintf("[+] Moved track from position %d to %d.", from, to))
			}
		}

		q, cur := p.GetQueue()
		if len(q) == 0 {
			return ctx.Reply("[*] Queue is empty.")
		}

		var sb strings.Builder
		if cur >= 0 && cur < len(q) {
			current := q[cur]
			reqMention := "Unknown"
			if current.Requester != "" {
				reqMention = fmt.Sprintf("<@%s>", current.Requester)
			}
			sb.WriteString(fmt.Sprintf("**Now Playing:**\n**[%s](%s)**\n%s • %s • %s\n\n",
				current.Info.Title, current.Info.URI, current.Info.Author, formatDur(current.Info.Length), reqMention))
		}

		upNext := q[cur+1:]
		if len(upNext) > 0 {
			sb.WriteString("**Up Next:**\n")
			limit := len(upNext)
			if limit > 10 {
				limit = 10
			}
			for i := 0; i < limit; i++ {
				t := upNext[i]
				reqMention := "Unknown"
				if t.Requester != "" {
					reqMention = fmt.Sprintf("<@%s>", t.Requester)
				}
				sb.WriteString(fmt.Sprintf("`%d.` **[%s](%s)**\n%s • %s • %s\n",
					i+1, t.Info.Title, t.Info.URI, t.Info.Author, formatDur(t.Info.Length), reqMention))
			}
			if len(upNext) > 10 {
				sb.WriteString(fmt.Sprintf("\n*...and %d more track(s)*\n", len(upNext)-10))
			}
		} else if cur < 0 || cur >= len(q) {
			sb.WriteString("The queue is currently empty.")
		}

		remainingTracks := len(upNext)
		if cur >= 0 && cur < len(q) {
			remainingTracks++
		}
		sb.WriteString(fmt.Sprintf("\n**Loop:** %s | **Total:** %d tracks", p.LoopMode(), remainingTracks))

		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "Queue",
			Description: sb.String(),
		})
		return ctx.Respond(emb)
	},
}

var NP = &manager.Command{
	Trigger:     "np",
	Aliases:     []string{"nowplaying"},
	Name:        "np",
	Description: "Show details about the currently playing song",
	Category:    "music",
	Execute: func(ctx *manager.CommandContext) error {
		l := ctx.Mgr.GetLavalink(ctx.ClientID)
		if l == nil {
			return nil
		}
		p := l.GetPlayer(ctx.Message.GuildID)
		t, _, ok := p.NowPlaying()
		if !ok {
			return ctx.Reply("[*] Nothing is currently playing.")
		}

		pos := int64(0)
		if playerInfo, err := getLavalinkPlayer(l, ctx.Message.GuildID); err == nil {
			pos = playerInfo.Position
		}

		isPausedStr := "Playing"
		if p.Paused() {
			isPausedStr = "Paused"
		}
		reqMention := "Unknown"
		if t.Requester != "" {
			reqMention = fmt.Sprintf("<@%s>", t.Requester)
		}
		desc := fmt.Sprintf("**[%s](%s)**\n\n**Status:** %s\n**Duration:** %s / %s\n**Requested By:** %s\n**Loop:** %s",
			t.Info.Title, t.Info.URI, isPausedStr, formatDur(pos), formatDur(t.Info.Length), reqMention, p.LoopMode())

		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "Now Playing",
			Description: desc,
		})
		return ctx.Respond(emb)
	},
}

var Volume = &manager.Command{
	Trigger:     "volume",
	Aliases:     []string{"vol"},
	Name:        "volume",
	Description: "Set player volume level",
	Category:    "music",
	Execute: func(ctx *manager.CommandContext) error {
		l := ctx.Mgr.GetLavalink(ctx.ClientID)
		if l == nil {
			return nil
		}
		p := l.GetPlayer(ctx.Message.GuildID)
		if len(ctx.Args) == 0 {
			return ctx.Reply(fmt.Sprintf("[*] Current volume is **%d%%**.", p.Vol()))
		}
		v, err := strconv.Atoi(ctx.Args[0])
		if err != nil || v < 0 || v > 150 {
			return ctx.Reply("[!] Volume must be a number between 0 and 150.")
		}
		if err := p.Volume(v); err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Failed to set volume: %v", err))
		}
		return ctx.Reply(fmt.Sprintf("[+] Set volume to **%d%%**.", v))
	},
}

var Seek = &manager.Command{
	Trigger:     "seek",
	Name:        "seek",
	Description: "Seek to a timestamp (e.g. 1:30 or 90)",
	Category:    "music",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("seek")
		}
		l := ctx.Mgr.GetLavalink(ctx.ClientID)
		if l == nil {
			return nil
		}
		p := l.GetPlayer(ctx.Message.GuildID)

		input := ctx.Args[0]
		ms := parseDurationToMs(input)

		if err := p.Seek(ms); err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Seek failed: %v", err))
		}
		return ctx.Reply(fmt.Sprintf("[+] Seeked to **%s**.", formatDur(ms)))
	},
}

var Fastforward = &manager.Command{
	Trigger:     "fastforward",
	Aliases:     []string{"ff"},
	Name:        "fastforward",
	Description: "Fastforward the current song by a specified duration",
	Category:    "music",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("fastforward")
		}
		l := ctx.Mgr.GetLavalink(ctx.ClientID)
		if l == nil {
			return nil
		}
		p := l.GetPlayer(ctx.Message.GuildID)

		playerInfo, err := getLavalinkPlayer(l, ctx.Message.GuildID)
		if err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Could not get current playback position: %v", err))
		}

		offset := parseDurationToMs(ctx.Args[0])
		if offset <= 0 {
			return ctx.Reply("[!] Invalid duration.")
		}

		newPos := playerInfo.Position + offset
		if err := p.Seek(newPos); err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Fastforward failed: %v", err))
		}
		return ctx.Reply(fmt.Sprintf("[+] Fastforwarded to **%s**.", formatDur(newPos)))
	},
}

var Rewind = &manager.Command{
	Trigger:     "rewind",
	Aliases:     []string{"rw"},
	Name:        "rewind",
	Description: "Rewind the current song by a specified duration",
	Category:    "music",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("rewind")
		}
		l := ctx.Mgr.GetLavalink(ctx.ClientID)
		if l == nil {
			return nil
		}
		p := l.GetPlayer(ctx.Message.GuildID)

		playerInfo, err := getLavalinkPlayer(l, ctx.Message.GuildID)
		if err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Could not get current playback position: %v", err))
		}

		offset := parseDurationToMs(ctx.Args[0])
		if offset <= 0 {
			return ctx.Reply("[!] Invalid duration.")
		}

		newPos := playerInfo.Position - offset
		if newPos < 0 {
			newPos = 0
		}

		if err := p.Seek(newPos); err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Rewind failed: %v", err))
		}
		return ctx.Reply(fmt.Sprintf("[+] Rewound to **%s**.", formatDur(newPos)))
	},
}

var Preset = &manager.Command{
	Trigger:     "preset",
	Name:        "preset",
	Description: "Manage audio equalizers and presets",
	Category:    "music",
	Execute: func(ctx *manager.CommandContext) error {
		l := ctx.Mgr.GetLavalink(ctx.ClientID)
		if l == nil {
			return nil
		}
		p := l.GetPlayer(ctx.Message.GuildID)

		validPresets := map[string]bool{
			"vaporwave": true,
			"karaoke":   true,
			"nightcore": true,
			"boost":     true,
			"flat":      true,
			"chipmunk":  true,
			"piano":     true,
			"metal":     true,
			"vibrato":   true,
			"8d":        true,
			"soft":      true,
		}

		if len(ctx.Args) == 0 {
			var sb strings.Builder
			sb.WriteString("Available presets:\n")
			for preset := range validPresets {
				status := "off"
				if p.Preset() == preset {
					status = "on"
				}
				sb.WriteString(fmt.Sprintf("- **%s** (currently %s)\n", preset, status))
			}
			sb.WriteString("\nUse `.preset <name> <on/off>` to configure.")
			return ctx.Reply(sb.String())
		}

		sub := strings.ToLower(ctx.Args[0])
		if sub == "active" {
			return ctx.Reply(fmt.Sprintf("[*] Current active preset: **%s**", p.Preset()))
		}

		if !validPresets[sub] {
			return ctx.Reply("[!] Invalid preset name. Use `.preset` to view all available presets.")
		}

		setting := true
		if len(ctx.Args) > 1 {
			val := strings.ToLower(ctx.Args[1])
			if val == "off" || val == "disable" || val == "false" {
				setting = false
			}
		}

		if err := p.SetPreset(sub, setting); err != nil {
			return ctx.Reply(fmt.Sprintf("[!] Failed to set preset: %v", err))
		}

		if setting {
			return ctx.Reply(fmt.Sprintf("[+] Preset **%s** turned ON.", sub))
		}
		return ctx.Reply(fmt.Sprintf("[+] Preset **%s** turned OFF.", sub))
	},
}

var Loop = &manager.Command{
	Trigger:     "loop",
	Aliases:     []string{"repeat"},
	Name:        "loop",
	Description: "Set queue or track loop mode",
	Category:    "music",
	Execute: func(ctx *manager.CommandContext) error {
		l := ctx.Mgr.GetLavalink(ctx.ClientID)
		if l == nil {
			return nil
		}
		p := l.GetPlayer(ctx.Message.GuildID)

		mode := "off"
		if len(ctx.Args) > 0 {
			mode = strings.ToLower(ctx.Args[0])
		} else {
			switch p.LoopMode() {
			case "off":
				mode = "track"
			case "track":
				mode = "queue"
			default:
				mode = "off"
			}
		}

		if mode != "off" && mode != "track" && mode != "queue" {
			return ctx.Reply("[!] Invalid mode. Choose from: off, track, queue.")
		}

		p.SetLoop(mode)
		return ctx.Reply(fmt.Sprintf("[+] Loop mode set to **%s**.", mode))
	},
}

var Shuffle = &manager.Command{
	Trigger:     "shuffle",
	Aliases:     []string{"shuf"},
	Name:        "shuffle",
	Description: "Shuffle the music queue",
	Category:    "music",
	Execute: func(ctx *manager.CommandContext) error {
		l := ctx.Mgr.GetLavalink(ctx.ClientID)
		if l == nil {
			return nil
		}
		p := l.GetPlayer(ctx.Message.GuildID)
		p.Shuffle()
		return ctx.Reply("[+] Shuffled the queue.")
	},
}

var Clear = &manager.Command{
	Trigger:     "clear",
	Aliases:     []string{"clearqueue"},
	Name:        "clear",
	Description: "Clear the music queue",
	Category:    "music",
	Execute: func(ctx *manager.CommandContext) error {
		l := ctx.Mgr.GetLavalink(ctx.ClientID)
		if l == nil {
			return nil
		}
		p := l.GetPlayer(ctx.Message.GuildID)
		p.Clear()
		return ctx.Reply("[+] Cleared the queue.")
	},
}

func findUserVC(s *discordgo.Session, gid, uid string) string {
	vs, err := s.State.VoiceState(gid, uid)
	if err == nil && vs != nil {
		return vs.ChannelID
	}
	g, err := s.State.Guild(gid)
	if err == nil {
		for _, v := range g.VoiceStates {
			if v.UserID == uid {
				return v.ChannelID
			}
		}
	}
	return ""
}

func formatDur(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func parseDurationToMs(input string) int64 {
	if strings.Contains(input, ":") {
		parts := strings.Split(input, ":")
		if len(parts) == 2 {
			m, _ := strconv.Atoi(parts[0])
			s, _ := strconv.Atoi(parts[1])
			return int64(m*60+s) * 1000
		} else if len(parts) == 3 {
			h, _ := strconv.Atoi(parts[0])
			m, _ := strconv.Atoi(parts[1])
			s, _ := strconv.Atoi(parts[2])
			return int64(h*3600+m*60+s) * 1000
		}
	}
	s, _ := strconv.Atoi(input)
	return int64(s) * 1000
}

func makepeniscum(current, total int64, width int) string {
	if total <= 0 {
		return "[" + strings.Repeat("-", width) + "]"
	}
	progress := float64(current) / float64(total)
	pos := int(progress * float64(width))
	if pos < 0 {
		pos = 0
	}
	if pos > width {
		pos = width
	}
	bar := strings.Repeat("▬", pos) + "🔘" + strings.Repeat("▬", width-pos)
	return bar
}

type jerkmeoff struct {
	Position int64 `json:"position"`
}

func getLavalinkPlayer(l *lavalink.Client, gid string) (*lavalinkPlayer, error) {
	sess := l.SessID()
	if sess == "" {
		return nil, fmt.Errorf("no session")
	}
	u := fmt.Sprintf("http://%s/v4/sessions/%s/players/%s", l.Host(), sess, gid)
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", l.Pwd())

	resp, err := lavalink.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status: %d", resp.StatusCode)
	}

	var res lavalinkPlayer
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}
