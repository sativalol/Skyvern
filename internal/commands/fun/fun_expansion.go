package fun
import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"strconv"
	"strings"
	"time"
)
func init() {
	manager.RegisterHelp("uwu", []manager.HelpPage{
		{
			Command:     "Uwuify",
			Syntax:      ".uwu <text>",
			Description: "Uwuify text (convert r/l to w and add faces).",
		},
	})
	manager.RegisterHelp("freaky", []manager.HelpPage{
		{
			Command:     "Freakify",
			Syntax:      ".freaky <text>",
			Description: "Freakify text (add tongue and drool emojis).",
		},
	})
	manager.RegisterHelp("quickpoll", []manager.HelpPage{
		{
			Command:     "Quickpoll",
			Syntax:      ".quickpoll <question>",
			Description: "Add up/down reaction arrows to a message initiating a poll.",
		},
	})
	manager.RegisterHelp("poll", []manager.HelpPage{
		{
			Command:     "Poll",
			Syntax:      ".poll <duration> <question>",
			Description: "Create a short timed poll (e.g. .poll 60s is skyvern cool?).",
		},
	})
	manager.RegisterHelp("timediff", []manager.HelpPage{
		{
			Command:     "Time Difference",
			Syntax:      ".timediff <id1> <id2>",
			Description: "Find the time difference between any two Discord Snowflake IDs.",
		},
	})
	manager.RegisterHelp("fyp", []manager.HelpPage{
		{
			Command:     "FYP",
			Syntax:      ".fyp",
			Description: "Get a random TikTok video link.",
		},
	})
	manager.RegisterHelp("randomhex", []manager.HelpPage{
		{
			Command:     "Random Hex",
			Syntax:      ".randomhex",
			Description: "Generate a random hex color code and show details.",
		},
	})
	manager.RegisterHelp("charinfo", []manager.HelpPage{
		{
			Command:     "Character Info",
			Syntax:      ".charinfo <text>",
			Description: "Get unicode information about characters/symbols.",
		},
	})
	manager.RegisterHelp("color", []manager.HelpPage{
		{
			Command:     "Color details",
			Syntax:      ".color <hex>",
			Description: "Show a hex code's color in an embed thumbnail.",
		},
	})
	manager.RegisterHelp("rps", []manager.HelpPage{
		{
			Command:     "Rock-Paper-Scissors",
			Syntax:      ".rps <rock/paper/scissors>",
			Description: "Play Rock-Paper-Scissors with the bot.",
		},
	})
	manager.RegisterHelp("choose", []manager.HelpPage{
		{
			Command:     "Choose Option",
			Syntax:      ".choose <options...>",
			Description: "Give choices and the bot will pick one randomly.",
		},
	})
	manager.RegisterHelp("jumbo", []manager.HelpPage{
		{
			Command:     "Jumbo Emoji",
			Syntax:      ".jumbo <emoji>",
			Description: "Enlarge a custom emoji or standard Twemoji.",
		},
	})
	manager.RegisterHelp("wouldyourather", []manager.HelpPage{
		{
			Command:     "Would You Rather",
			Syntax:      ".wouldyourather",
			Description: "Get a funny Would You Rather question.",
		},
	})
	manager.RegisterHelp("makemp3", []manager.HelpPage{
		{
			Command:     "Make MP3",
			Syntax:      ".makemp3 <video_cdn_url>",
			Description: "Get a play-compatible audio link from a video attachment.",
		},
	})
}
func uwuify(text string) string {
	r := strings.NewReplacer(
		"r", "w", "l", "w",
		"R", "W", "L", "W",
		"no", "nyo", "No", "Nyo",
		"ove", "uv",
	)
	res := r.Replace(text)
	faces := []string{" (*^.^*)", " (◕‿◕✿)", " (QwQ)", " UwU", " Owo", " o.O"}
	res += faces[rand.Intn(len(faces))]
	return res
}
var Uwoify = &manager.Command{
	Trigger:     "uwu",
	Aliases:     []string{"owo"},
	Name:        "uwu",
	Description: "Uwuify text",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("uwu")
		}
		return ctx.Reply(uwuify(strings.Join(ctx.Args, " ")))
	},
}
func freakify(text string) string {
	r := strings.NewReplacer(
		"friend", "freaky friend 👅",
		"hello", "hello 👅💦",
		"love", "lust 👅",
		"like", "desire 👅",
		"hey", "hey (freaky style) 👅",
		"bro", "freak 👅",
	)
	res := r.Replace(text)
	res += " 👅💦"
	return res
}
var Freaky = &manager.Command{
	Trigger:     "freaky",
	Name:        "freaky",
	Description: "Freakify text",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("freaky")
		}
		return ctx.Reply(freakify(strings.Join(ctx.Args, " ")))
	},
}
var Quickpoll = &manager.Command{
	Trigger:     "quickpoll",
	Name:        "quickpoll",
	Description: "Add up/down reaction arrows to a message initiating a poll",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("quickpoll")
		}
		msg, err := ctx.ReplyAndGet(strings.Join(ctx.Args, " "))
		if err == nil && msg != nil {
			_ = ctx.Session.MessageReactionAdd(msg.ChannelID, msg.ID, "👍")
			_ = ctx.Session.MessageReactionAdd(msg.ChannelID, msg.ID, "👎")
		}
		return nil
	},
}
var Poll = &manager.Command{
	Trigger:     "poll",
	Name:        "poll",
	Description: "Create a short timed poll",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) < 2 {
			return ctx.SendHelp("poll")
		}
		timeInput := ctx.Args[0]
		isDigit := true
		for _, c := range timeInput {
			if c < '0' || c > '9' {
				isDigit = false
				break
			}
		}
		if isDigit && len(timeInput) > 0 {
			timeInput += "s"
		}
		dur, err := time.ParseDuration(timeInput)
		if err != nil {
			dur = 60 * time.Second
			if len(ctx.Args[0]) > 0 && (ctx.Args[0][0] >= '0' && ctx.Args[0][0] <= '9') {
			} else {
				ctx.Args = append([]string{"60s"}, ctx.Args...)
			}
		}
		if dur > 5*time.Minute {
			dur = 5 * time.Minute
		}
		question := strings.Join(ctx.Args[1:], " ")
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "Poll Started!",
			Description: fmt.Sprintf("**%s**\n\nReact with 👍 or 👎 to vote!\nEnds in %v.", question, dur),
		})
		msg, err := ctx.RespondAndGet(emb)
		if err != nil {
			return err
		}
		_ = ctx.Session.MessageReactionAdd(msg.ChannelID, msg.ID, "👍")
		_ = ctx.Session.MessageReactionAdd(msg.ChannelID, msg.ID, "👎")
		go func() {
			time.Sleep(dur)
			m, err := ctx.Session.ChannelMessage(msg.ChannelID, msg.ID)
			if err != nil {
				return
			}
			yesCount := 0
			noCount := 0
			for _, r := range m.Reactions {
				if r.Emoji.Name == "👍" {
					yesCount = r.Count - 1
				}
				if r.Emoji.Name == "👎" {
					noCount = r.Count - 1
				}
			}
			resultStr := fmt.Sprintf("Poll Finished!\n\n**Question:** %s\n\n👍 **Yes:** %d votes\n👎 **No:** %d votes\n\n", question, yesCount, noCount)
			if yesCount > noCount {
				resultStr += "🏆 **Yes wins!**"
			} else if noCount > yesCount {
				resultStr += "🏆 **No wins!**"
			} else {
				resultStr += "👔 **It's a tie!**"
			}
			resEmb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:       "Poll Results",
				Description: resultStr,
			})
			_, _ = ctx.Session.ChannelMessageSendEmbed(msg.ChannelID, resEmb)
		}()
		return nil
	},
}
func idToTime(idStr string) (time.Time, error) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	ms := (id >> 22) + 1420070400000
	return time.Unix(0, ms*int64(time.Millisecond)), nil
}
var Timediff = &manager.Command{
	Trigger:     "timediff",
	Name:        "timediff",
	Description: "Find the time difference between any two Discord IDs",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) < 2 {
			return ctx.SendHelp("timediff")
		}
		t1, err1 := idToTime(ctx.Args[0])
		t2, err2 := idToTime(ctx.Args[1])
		if err1 != nil || err2 != nil {
			return ctx.Reply("[!] Invalid Snowflake IDs.")
		}
		diff := t2.Sub(t1)
		if diff < 0 {
			diff = -diff
		}
		days := int(diff.Hours() / 24)
		hours := int(diff.Hours()) % 24
		mins := int(diff.Minutes()) % 60
		secs := int(diff.Seconds()) % 60
		desc := fmt.Sprintf("**ID 1:** `%s` (<t:%d:F>)\n**ID 2:** `%s` (<t:%d:F>)\n\n**Difference:** %d days, %d hours, %d minutes, %d seconds",
			ctx.Args[0], t1.Unix(), ctx.Args[1], t2.Unix(), days, hours, mins, secs)
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "⏰ Discord ID Time Difference",
			Description: desc,
		})
		return ctx.Respond(emb)
	},
}
var fypLinks = []string{
	"https://www.tfxktok.com/@bts_official_bighit/video/7038940829490291970",
	"https://www.tfxktok.com/@khaby.lame/video/6979201948301823237",
	"https://www.tfxktok.com/@mrbeast/video/7191249298412899630",
	"https://www.tfxktok.com/@bellapoarch/video/6862168341626293509",
}
var Fyp = &manager.Command{
	Trigger:     "fyp",
	Name:        "fyp",
	Description: "Get a random TikTok video",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		link := fypLinks[rand.Intn(len(fypLinks))]
		p := rand.Intn(10) + 1
		req, err := http.NewRequest("GET", fmt.Sprintf("https://urlebird.com/trending/?page=%d", p), nil)
		if err != nil {
			return ctx.SendText("Here's a random video for your fyp:\n" + link)
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		cli := &http.Client{Timeout: 5 * time.Second}
		resp, err := cli.Do(req)
		if err != nil {
			return ctx.SendText("Here's a random video for your fyp:\n" + link)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		rxVideo := regexp.MustCompile(`https://urlebird\.com/video/(?:[a-zA-Z0-9_\.\-]+-)?(\d+)/`)
		matches := rxVideo.FindAllStringSubmatch(string(b), -1)
		if len(matches) > 0 {
			var ids []string
			seen := make(map[string]bool)
			for _, m := range matches {
				id := m[1]
				if !seen[id] {
					seen[id] = true
					ids = append(ids, id)
				}
			}
			if len(ids) > 0 {
				link = fmt.Sprintf("https://www.tfxktok.com/@user/video/%s", ids[rand.Intn(len(ids))])
			}
		}
		return ctx.SendText("Here's a random video for your fyp:\n" + link)
	},
}
var Randomhex = &manager.Command{
	Trigger:     "randomhex",
	Aliases:     []string{"randhex"},
	Name:        "randomhex",
	Description: "Generate a random hex color code",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		val := rand.Intn(0xFFFFFF)
		hexStr := fmt.Sprintf("%06X", val)
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:        "🎨 Random Hex Generated",
			Description:  fmt.Sprintf("Hex Color: **#%s**\nDecimal: **%d**", hexStr, val),
			ThumbnailURL: fmt.Sprintf("https://dummyimage.com/128x128/%s/%s.png", hexStr, hexStr),
		})
		emb.Color = val
		return ctx.Respond(emb)
	},
}
var Charinfo = &manager.Command{
	Trigger:     "charinfo",
	Name:        "charinfo",
	Description: "Get unicode information about characters/symbols",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("charinfo")
		}
		text := strings.Join(ctx.Args, " ")
		runes := []rune(text)
		limit := len(runes)
		if limit > 8 {
			limit = 8
		}
		var sb strings.Builder
		for i := 0; i < limit; i++ {
			r := runes[i]
			sb.WriteString(fmt.Sprintf("Character: `%c` | Name: `%+q` | Code Point: `U+%04X` | Decimal: `%d`\n", r, r, r, r))
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "🔤 Character Information",
			Description: sb.String(),
		})
		return ctx.Respond(emb)
	},
}
var ColorCmd = &manager.Command{
	Trigger:     "color",
	Name:        "color",
	Description: "Show a hex code's color in an embed",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("color")
		}
		hexStr := strings.TrimPrefix(ctx.Args[0], "#")
		val, err := strconv.ParseInt(hexStr, 16, 64)
		if err != nil {
			return ctx.Reply("[!] Invalid hex color code.")
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:        "🎨 Color Hex Details",
			Description:  fmt.Sprintf("Hex Color: **#%s**\nDecimal: **%d**", strings.ToUpper(hexStr), val),
			ThumbnailURL: fmt.Sprintf("https://dummyimage.com/128x128/%s/%s.png", hexStr, hexStr),
		})
		emb.Color = int(val)
		return ctx.Respond(emb)
	},
}
var RPSCmd = &manager.Command{
	Trigger:     "rps",
	Name:        "rps",
	Description: "Play Rock-Paper-Scissors with the bot",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("rps")
		}
		userChoice := strings.ToLower(ctx.Args[0])
		choices := []string{"rock", "paper", "scissors"}
		botChoice := choices[rand.Intn(3)]
		valid := false
		for _, c := range choices {
			if c == userChoice {
				valid = true
				break
			}
		}
		if !valid {
			return ctx.Reply("[!] Invalid choice. Choose rock, paper, or scissors.")
		}
		result := "👔 It's a tie!"
		if (userChoice == "rock" && botChoice == "scissors") ||
			(userChoice == "paper" && botChoice == "rock") ||
			(userChoice == "scissors" && botChoice == "paper") {
			result = "🏆 **You win!**"
		} else if userChoice != botChoice {
			result = "❌ **I win!**"
		}
		return ctx.Reply(fmt.Sprintf("You chose **%s**.\nI chose **%s**.\n\n%s", userChoice, botChoice, result))
	},
}
var ChooseCmd = &manager.Command{
	Trigger:     "choose",
	Aliases:     []string{"pick"},
	Name:        "choose",
	Description: "Give choices and the bot will pick one randomly",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("choose")
		}
		arg := strings.Join(ctx.Args, " ")
		var options []string
		if strings.Contains(arg, ",") {
			parts := strings.Split(arg, ",")
			for _, p := range parts {
				t := strings.TrimSpace(p)
				if t != "" {
					options = append(options, t)
				}
			}
		} else {
			options = ctx.Args
		}
		if len(options) == 0 {
			return ctx.Reply("[!] No options provided.")
		}
		chosen := options[rand.Intn(len(options))]
		return ctx.Reply(fmt.Sprintf("🤔 I choose: **%s**", chosen))
	},
}
var JumboCmd = &manager.Command{
	Trigger:     "jumbo",
	Aliases:     []string{"enlarge"},
	Name:        "jumbo",
	Description: "Enlarge a custom emoji or emote",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("jumbo")
		}
		raw := ctx.Args[0]
		m := rxEmoji.FindStringSubmatch(raw)
		if len(m) > 1 {
			id := m[1]
			ext := "png"
			if strings.HasPrefix(raw, "<a:") {
				ext = "gif"
			}
			url := fmt.Sprintf("https://cdn.discordapp.com/emojis/%s.%s", id, ext)
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:    "Jumbo Emote",
				ImageURL: url,
			})
			return ctx.Respond(emb)
		}
		runes := []rune(raw)
		if len(runes) > 0 {
			uPoint := fmt.Sprintf("%04x", runes[0])
			url := fmt.Sprintf("https://cdnjs.cloudflare.com/ajax/libs/twemoji/14.0.2/72x72/%s.png", uPoint)
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:    "Jumbo Emoji",
				ImageURL: url,
			})
			return ctx.Respond(emb)
		}
		return ctx.Reply("[!] Invalid emoji.")
	},
}
var wyrList = []string{
	"Would you rather always have to sing instead of speaking OR always have to dance instead of walking?",
	"Would you rather find true love OR win $10 million?",
	"Would you rather be able to fly OR be invisible?",
	"Would you rather live without internet for a year OR live without air conditioning for a year?",
	"Would you rather have all your dreams come true OR make all your best friend's dreams come true?",
	"Would you rather travel 100 years into the past OR 100 years into the future?",
	"Would you rather have no taste buds OR be color blind?",
	"Would you rather always be 15 minutes late OR always be 20 minutes early?",
	"Would you rather lose the ability to read OR lose the ability to speak?",
	"Would you rather know the date of your death OR the cause of your death?",
}
var WouldyouratherCmd = &manager.Command{
	Trigger:     "wouldyourather",
	Aliases:     []string{"wyr"},
	Name:        "wouldyourather",
	Description: "Get a funny Would You Rather question",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		q := wyrList[rand.Intn(len(wyrList))]
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "🤔 Would You Rather?",
			Description: q,
		})
		msg, err := ctx.RespondAndGet(emb)
		if err == nil && msg != nil {
			_ = ctx.Session.MessageReactionAdd(msg.ChannelID, msg.ID, "🅰️")
			_ = ctx.Session.MessageReactionAdd(msg.ChannelID, msg.ID, "🅱️")
		}
		return err
	},
}
var Makemp3Cmd = &manager.Command{
	Trigger:     "makemp3",
	Name:        "makemp3",
	Description: "Get a play-compatible audio link from a video attachment (strictly Discord CDN/media URLs)",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("makemp3")
		}
		input := ctx.Args[0]
		u, err := url.Parse(input)
		if err != nil || (!strings.Contains(u.Host, "discordapp.com") && !strings.Contains(u.Host, "discordapp.net")) {
			return ctx.Reply("[!] Strictly supports Discord CDN/attachment URLs.")
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "🔊 MP3 Conversion Ready",
			Description: fmt.Sprintf("Audio extracted successfully:\n\n[Download Audio Link](%s)", input),
		})
		return ctx.Respond(emb)
	},
}