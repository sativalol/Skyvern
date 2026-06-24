package fun

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/storage"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	bluntRot   = make(map[string][]string)
	bluntIndex = make(map[string]int)
	bluntHits  = make(map[string]int)
	bluntMu    sync.Mutex

	yartCharge = make(map[string]int)
	yartHits   = make(map[string]int)
	yartMu     sync.Mutex
)

var (
	juulCharge      = make(map[string]int)
	juulPods        = make(map[string]int)
	juulFlavor      = make(map[string]string)
	juulOwner       = make(map[string]string)
	juulToggled     = make(map[string]bool)
	juulHitsCount   = make(map[string]map[string]int)
	juulPassesCount = make(map[string]int)
	juulMu          sync.Mutex
)

var Blunt = &manager.Command{
	Trigger:     "blunt",
	Aliases:     []string{"smoke", "sesh"},
	Name:        "blunt",
	Description: "Smoke and pass the blunt",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()
		uid := ctx.AuthorID()
		bluntMu.Lock()
		defer bluntMu.Unlock()

		if len(ctx.Args) > 0 {
			sub := strings.ToLower(ctx.Args[0])
			if sub == "add" {
				if len(ctx.Args) < 2 {
					return ctx.Reply("[!] Specify a user to add.")
				}
				target := resolveVouchUser(ctx.Session, gid, ctx.Args[1])
				if target == "" {
					return ctx.Reply("[!] User not found.")
				}
				bluntRot[gid] = append(bluntRot[gid], target)
				return ctx.Reply(fmt.Sprintf("[+] Added <@%s> to the session rotation.", target))
			}
			if sub == "pass" {
				rot := bluntRot[gid]
				if len(rot) <= 1 {
					return ctx.Reply("[!] Not enough people in the rotation. Use `.blunt add <user>`.")
				}
				bluntIndex[gid] = (bluntIndex[gid] + 1) % len(rot)
				next := rot[bluntIndex[gid]]
				return ctx.Reply(fmt.Sprintf("[*] Blunt passed. It is now <@%s>'s turn to hit.", next))
			}
		}

		rot := bluntRot[gid]
		if len(rot) == 0 {
			bluntRot[gid] = []string{uid}
			rot = bluntRot[gid]
			bluntIndex[gid] = 0
		}

		current := rot[bluntIndex[gid]]
		if current != uid {
			return ctx.Reply(fmt.Sprintf("[!] It is not your turn in the rotation. Currently <@%s>'s turn.", current))
		}

		bluntHits[gid]++
		hits := bluntHits[gid]

		if hits >= 5 {
			bluntHits[gid] = 0
			bluntRot[gid] = nil
			bluntIndex[gid] = 0
			return ctx.Reply("💨 *Cough! Cough!* The blunt is finished. Start a new session with `.blunt`.")
		}

		return ctx.Reply(fmt.Sprintf("💨 <@%s> takes a hit of the blunt. (%d/5 hits remaining)", uid, 5-hits))
	},
}

var Juul = &manager.Command{
	Trigger:     "juul",
	Name:        "juul",
	Description: "Share a juul with your friends!",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()
		uid := ctx.AuthorID()
		juulMu.Lock()
		defer juulMu.Unlock()

		if _, ok := juulCharge[gid]; !ok {
			juulCharge[gid] = 100
			juulPods[gid] = 100
			juulFlavor[gid] = "mint"
			juulToggled[gid] = true
			juulHitsCount[gid] = make(map[string]int)
		}

		sub := "hit"
		if len(ctx.Args) > 0 {
			sub = strings.ToLower(ctx.Args[0])
		}

		if sub == "toggle" {
			p, err := ctx.Session.UserChannelPermissions(uid, ctx.ChanID())
			if err != nil || (p&discordgo.PermissionManageGuild) == 0 {
				return ctx.Reply("[!] You need Manage Guild permission to use this.")
			}
			juulToggled[gid] = !juulToggled[gid]
			state := "enabled"
			if !juulToggled[gid] {
				state = "disabled"
			}
			return ctx.Reply(fmt.Sprintf("[+] Juul has been %s in this server.", state))
		}

		if !juulToggled[gid] {
			return ctx.Reply("[!] The Juul is currently disabled in this server.")
		}

		switch sub {
		case "flavor":
			p, err := ctx.Session.UserChannelPermissions(uid, ctx.ChanID())
			if err != nil || (p&discordgo.PermissionManageGuild) == 0 {
				return ctx.Reply("[!] You need Manage Guild permission to change flavor.")
			}
			if len(ctx.Args) < 2 {
				return ctx.Reply("[!] Specify a flavor.")
			}
			flv := strings.Join(ctx.Args[1:], " ")
			juulFlavor[gid] = flv
			return ctx.Reply(fmt.Sprintf("[+] Changed Juul flavor to **%s** 🧪", flv))

		case "charge":
			juulCharge[gid] = 100
			return ctx.Reply("[+] Juul charged to 100% 🔋")

		case "pod":
			juulPods[gid] = 100
			return ctx.Reply("[+] Slapped in a fresh pod 🧪")

		case "pass":
			if len(ctx.Args) < 2 {
				return ctx.Reply("[!] Specify a member to pass to.")
			}
			target := resolveVouchUser(ctx.Session, gid, ctx.Args[1])
			if target == "" {
				return ctx.Reply("[!] Member not found.")
			}
			owner := juulOwner[gid]
			if owner != "" && owner != uid {
				return ctx.Reply(fmt.Sprintf("[!] You don't have the Juul. Currently held by <@%s>.", owner))
			}
			juulOwner[gid] = target
			juulPassesCount[gid]++
			return ctx.Reply(fmt.Sprintf("[*] Passed the Juul to <@%s> 🤝", target))

		case "steal":
			owner := juulOwner[gid]
			if owner == uid {
				return ctx.Reply("[!] You already have the Juul.")
			}
			juulOwner[gid] = uid
			if owner == "" {
				return ctx.Reply("[*] You grabbed the Juul 📱")
			}
			return ctx.Reply(fmt.Sprintf("[*] <@%s> stole the Juul from <@%s>! 😈", uid, owner))

		case "stats":
			owner := "Nobody"
			if juulOwner[gid] != "" {
				owner = fmt.Sprintf("<@%s>", juulOwner[gid])
			}
			var topUsers []string
			var topCounts []int
			for k, v := range juulHitsCount[gid] {
				topUsers = append(topUsers, k)
				topCounts = append(topCounts, v)
			}
			for i := 0; i < len(topUsers); i++ {
				for j := i + 1; j < len(topUsers); j++ {
					if topCounts[i] < topCounts[j] {
						topCounts[i], topCounts[j] = topCounts[j], topCounts[i]
						topUsers[i], topUsers[j] = topUsers[j], topUsers[i]
					}
				}
			}
			topStr := "No hits yet."
			if len(topUsers) > 0 {
				var lines []string
				limit := len(topUsers)
				if limit > 3 {
					limit = 3
				}
				for i := 0; i < limit; i++ {
					lines = append(lines, fmt.Sprintf("%d. <@%s> — **%d** hits", i+1, topUsers[i], topCounts[i]))
				}
				topStr = strings.Join(lines, "\n")
			}

			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title: "🔌 Server Juul Stats",
				Description: fmt.Sprintf(
					"🔋 **Battery:** %d%%\n"+
						"🧪 **Pod Juice:** %d%%\n"+
						"🍒 **Flavor:** %s\n"+
						"👑 **Held by:** %s\n"+
						"🤝 **Total Passes:** %d\n\n"+
						"🏆 **Top Hitters:**\n%s",
					juulCharge[gid], juulPods[gid], juulFlavor[gid], owner, juulPassesCount[gid], topStr,
				),
			})
			return ctx.Respond(emb)

		case "hit":
			owner := juulOwner[gid]
			if owner != "" && owner != uid {
				return ctx.Reply(fmt.Sprintf("[!] You don't have the Juul. Currently held by <@%s>. Steal it with `.juul steal`.", owner))
			}
			if juulCharge[gid] <= 0 {
				return ctx.Reply("[!] The Juul is dead. Charge it with `.juul charge`.")
			}
			if juulPods[gid] <= 0 {
				return ctx.Reply("[!] The pod is empty. Replace it with `.juul pod`.")
			}

			hitCharge := rand.Intn(10) + 5
			hitPod := rand.Intn(8) + 4
			juulCharge[gid] -= hitCharge
			juulPods[gid] -= hitPod
			if juulCharge[gid] < 0 {
				juulCharge[gid] = 0
			}
			if juulPods[gid] < 0 {
				juulPods[gid] = 0
			}

			juulHitsCount[gid][uid]++
			juulOwner[gid] = uid

			return ctx.Reply(fmt.Sprintf("💨 <@%s> takes a hit of the Juul.\n🔋 **Battery:** %d%% | 🧪 **Pod:** %d%% | 🍒 **Flavor:** %s", uid, juulCharge[gid], juulPods[gid], juulFlavor[gid]))
		}
		return nil
	},
}

var Yart = &manager.Command{
	Trigger:     "yart",
	Name:        "yart",
	Description: "Virtual weed pen",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()
		uid := ctx.AuthorID()
		yartMu.Lock()
		defer yartMu.Unlock()

		if _, ok := yartCharge[gid]; !ok {
			yartCharge[gid] = 100
			yartHits[gid] = 200
		}

		if len(ctx.Args) > 0 {
			sub := strings.ToLower(ctx.Args[0])
			if sub == "charge" {
				yartCharge[gid] = 100
				return ctx.Reply("[+] Weed pen fully charged. 🔋")
			}
		}

		if yartCharge[gid] <= 0 {
			return ctx.Reply("[!] Weed pen is dead. Charge it with `.yart charge`.")
		}
		if yartHits[gid] <= 0 {
			return ctx.Reply("[!] Cartridge is completely empty.")
		}

		if len(ctx.Args) > 0 && strings.ToLower(ctx.Args[0]) == "blinker" {
			yartCharge[gid] -= 25
			yartHits[gid] -= 20
			if yartCharge[gid] < 0 {
				yartCharge[gid] = 0
			}
			if yartHits[gid] < 0 {
				yartHits[gid] = 0
			}
			return ctx.Reply(fmt.Sprintf("💨 🔟💨 <@%s> hits a 10s blinker!\n🔋 **Battery:** %d%% | 🍯 **Oil:** %d%%", uid, yartCharge[gid], (yartHits[gid]*100)/200))
		}

		yartCharge[gid] -= 5
		yartHits[gid] -= 3
		if yartCharge[gid] < 0 {
			yartCharge[gid] = 0
		}
		if yartHits[gid] < 0 {
			yartHits[gid] = 0
		}

		return ctx.Reply(fmt.Sprintf("💨 <@%s> takes a pull from the weed pen.\n🔋 **Battery:** %d%% | 🍯 **Oil:** %d%%", uid, yartCharge[gid], (yartHits[gid]*100)/200))
	},
}

func weedProgressBar(val float64, max float64, length int) string {
	if val < 0 {
		val = 0
	}
	if val > max {
		val = max
	}
	filled := int((val / max) * float64(length))
	if filled > length {
		filled = length
	}
	empty := length - filled
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", empty) + "]"
}

func BuildWeedMessage(cfg config.ResCfg, wp storage.WeedPlant, page int, gid string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	now := time.Now()
	stage := "Seed 🌱"
	if wp.Health <= 0 {
		stage = "Dead 💀"
	} else if wp.Growth > 80 {
		stage = "Flowering 🪷"
	} else if wp.Growth > 50 {
		stage = "Vegging 🍃"
	} else if wp.Growth > 20 {
		stage = "Sprout 🌿"
	}
	if wp.Growth >= 100 {
		stage = "Ready to Cut! 🍁"
	}

	var statuses []string
	if now.Before(wp.LightUntil) {
		left := wp.LightUntil.Sub(now)
		statuses = append(statuses, fmt.Sprintf("💡 **LED boost on** (active for **%d minutes**)", int(left.Minutes())))
	}
	if wp.Infested {
		statuses = append(statuses, "🐛 **infested with bugs** (growing slow, losing health)")
	}
	if wp.Health > 0 {
		if wp.Water < 15.0 {
			statuses = append(statuses, "🥀 **dying of thirst** (needs water)")
		} else if wp.Water > 90.0 {
			statuses = append(statuses, "🥀 **drowning** (too much water)")
		}
		if wp.Fertilizer < 15.0 {
			statuses = append(statuses, "⚠️ **starving** (needs fertilizer)")
		} else if wp.Fertilizer > 90.0 {
			statuses = append(statuses, "⚠️ **nutrient burn** (too much fertilizer)")
		}
	}
	statusStr := strings.Join(statuses, "\n")
	if statusStr == "" {
		statusStr = "chillin"
	}

	waterCd := "ready 💧"
	if now.Before(wp.LastWater.Add(15 * time.Minute)) {
		waterCd = fmt.Sprintf("<t:%d:R>", wp.LastWater.Add(15*time.Minute).Unix())
	}
	fertCd := "ready 🧪"
	if now.Before(wp.LastFert.Add(30 * time.Minute)) {
		fertCd = fmt.Sprintf("<t:%d:R>", wp.LastFert.Add(30*time.Minute).Unix())
	}
	lightCd := "ready 💡"
	if now.Before(wp.LastLight.Add(2 * time.Hour)) {
		lightCd = fmt.Sprintf("<t:%d:R>", wp.LastLight.Add(2*time.Hour).Unix())
	}

	var embed *discordgo.MessageEmbed
	if page == 0 {
		desc := fmt.Sprintf(
			"keep it alive. or don't. whatever.\n\n"+
				"Plant Stage\n%s\n\n"+
				"Server Stash\n%d grams total\n\n"+
				"Growth\n%s %.1f%%\n\n"+
				"Health\n%s %.1f%%\n\n"+
				"Moisture (Water)\n%s %.1f%%\n\n"+
				"Nutrients (Fertilizer)\n%s %.1f%%\n\n"+
				"Statuses / Issues\n%s\n\n"+
				"Cooldowns\n"+
				"water: %s\n"+
				"fertilize: %s\n"+
				"lights: %s",
			stage, wp.Yields,
			weedProgressBar(wp.Growth, 100, 10), wp.Growth,
			weedProgressBar(wp.Health, 100, 10), wp.Health,
			weedProgressBar(wp.Water, 100, 10), wp.Water,
			weedProgressBar(wp.Fertilizer, 100, 10), wp.Fertilizer,
			statusStr,
			waterCd, fertCd, lightCd,
		)
		embed = config.Build(cfg, config.EmbedOpt{
			Title:       "🌿 server weed plant",
			Description: desc,
		})
	} else if page == 1 {
		embed = config.Build(cfg, config.EmbedOpt{
			Title: "💧 Weed Command - Water",
			Description: "**Usage:** `.weed water`\n\n" +
				"Splashes some water on the plant to keep it hydrated.\n" +
				"Moisture is set to **80.0%**.\n\n" +
				"**Cooldown:** 15 minutes",
		})
	} else if page == 2 {
		embed = config.Build(cfg, config.EmbedOpt{
			Title: "🧪 Weed Command - Fertilize",
			Description: "**Usage:** `.weed fertilize`\n\n" +
				"Tosses some fertilizer in the soil to supply nutrients.\n" +
				"Nutrients are set to **80.0%**.\n\n" +
				"**Cooldown:** 30 minutes",
		})
	} else if page == 3 {
		embed = config.Build(cfg, config.EmbedOpt{
			Title: "💡 Weed Command - Light",
			Description: "**Usage:** `.weed light`\n\n" +
				"Flips the LED grow lights on.\n" +
				"Doubles the plant's growth speed for 1 hour.\n\n" +
				"**Cooldown:** 2 hours",
		})
	} else if page == 4 {
		embed = config.Build(cfg, config.EmbedOpt{
			Title: "✨ Weed Command - Spray",
			Description: "**Usage:** `.weed spray`\n\n" +
				"Sprays pesticides on the plant to kill any infesting bugs.\n" +
				"Only usable when the plant is infested.\n\n" +
				"**Cooldown:** None",
		})
	} else if page == 5 {
		embed = config.Build(cfg, config.EmbedOpt{
			Title: "🍁 Weed Command - Harvest",
			Description: "**Usage:** `.weed harvest`\n\n" +
				"Cuts down the fully grown plant.\n" +
				"Yields **15 - 40 grams** of top-shelf bud.\n" +
				"Only usable when Growth reaches **100%**.\n\n" +
				"**Cooldown:** None",
		})
	}

	prevPage := page - 1
	if prevPage < 0 {
		prevPage = 5
	}
	nextPage := page + 1
	if nextPage > 5 {
		nextPage = 0
	}

	comps := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "◀",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("weed_page:%s:%d", gid, prevPage),
				},
				discordgo.Button{
					Label:    "🌿 Plant",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("weed_page:%s:0", gid),
					Disabled: page == 0,
				},
				discordgo.Button{
					Label:    "▶",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("weed_page:%s:%d", gid, nextPage),
				},
			},
		},
	}

	return embed, comps
}

func HandleWeedComponent(s *discordgo.Session, i *discordgo.InteractionCreate, mgr *manager.Manager) {
	parts := strings.Split(i.MessageComponentData().CustomID, ":")
	if len(parts) < 3 {
		return
	}
	gid := parts[1]
	pageIdx, _ := strconv.Atoi(parts[2])

	wp, err := mgr.DB().GetWeedPlant(gid)
	if err != nil {
		wp = storage.WeedPlant{
			Growth:     0,
			Water:      50,
			Fertilizer: 50,
			Health:     100,
			LastAction: time.Now(),
		}
	}

	now := time.Now()
	elapsed := now.Sub(wp.LastAction).Hours()
	if elapsed > 0.005 {
		wp.Water -= elapsed * 6.0
		wp.Fertilizer -= elapsed * 4.0

		if wp.Water < 0 {
			wp.Water = 0
		}
		if wp.Fertilizer < 0 {
			wp.Fertilizer = 0
		}

		healthChange := 0.0
		if wp.Water < 15.0 || wp.Water > 90.0 {
			healthChange -= elapsed * 2.5
		} else {
			healthChange += elapsed * 1.5
		}

		if wp.Fertilizer < 15.0 || wp.Fertilizer > 90.0 {
			healthChange -= elapsed * 2.5
		} else {
			healthChange += elapsed * 1.5
		}

		if wp.Infested {
			healthChange -= elapsed * 5.0
		}

		wp.Health += healthChange
		if wp.Health > 100 {
			wp.Health = 100
		}
		if wp.Health < 0 {
			wp.Health = 0
		}

		if wp.Health <= 0 {
			wp.Growth = 0
			wp.Water = 0
			wp.Fertilizer = 0
			wp.Health = 0
			wp.Infested = false
		} else {
			if wp.Health > 30.0 {
				growthRate := 4.0
				if wp.Infested {
					growthRate *= 0.5
				}
				if now.Before(wp.LightUntil) {
					growthRate *= 2.0
				}
				wp.Growth += elapsed * growthRate
				if wp.Growth > 100 {
					wp.Growth = 100
				}
			}
		}

		if !wp.Infested && wp.Growth > 0 && wp.Growth < 100 {
			prob := 1.0 - math.Pow(0.92, elapsed)
			if rand.Float64() < prob {
				wp.Infested = true
				wp.PestUntil = now
			}
		}

		wp.LastAction = now
		_ = mgr.DB().SaveWeedPlant(gid, wp)
	}

	botCfg, _ := mgr.DB().GetBot(s.State.User.ID)
	resCfg := config.Resolve(config.GetGlobal(), botCfg)

	embed, comps := BuildWeedMessage(resCfg, wp, pageIdx, gid)

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: comps,
		},
	})
}

var Weed = &manager.Command{
	Trigger:     "weed",
	Name:        "weed",
	Description: "Grow and nurture a server-wide weed plant with boosts, pests, and custom yields",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()
		uid := ctx.AuthorID()
		now := time.Now()

		wp, err := ctx.DB.GetWeedPlant(gid)
		if err != nil {
			wp = storage.WeedPlant{
				Growth:     0,
				Water:      50,
				Fertilizer: 50,
				Health:     100,
				LastAction: now,
			}
		}

		if wp.Health == 0 && wp.Growth == 0 && wp.Water == 0 && wp.Fertilizer == 0 {
			wp.Health = 100
			wp.Water = 50
			wp.Fertilizer = 50
		}

		elapsed := now.Sub(wp.LastAction).Hours()
		if elapsed > 0.005 {
			wp.Water -= elapsed * 6.0
			wp.Fertilizer -= elapsed * 4.0

			if wp.Water < 0 {
				wp.Water = 0
			}
			if wp.Fertilizer < 0 {
				wp.Fertilizer = 0
			}

			healthChange := 0.0
			if wp.Water < 15.0 || wp.Water > 90.0 {
				healthChange -= elapsed * 2.5
			} else {
				healthChange += elapsed * 1.5
			}

			if wp.Fertilizer < 15.0 || wp.Fertilizer > 90.0 {
				healthChange -= elapsed * 2.5
			} else {
				healthChange += elapsed * 1.5
			}

			if wp.Infested {
				healthChange -= elapsed * 5.0
			}

			wp.Health += healthChange
			if wp.Health > 100 {
				wp.Health = 100
			}
			if wp.Health < 0 {
				wp.Health = 0
			}

			if wp.Health <= 0 {
				wp.Growth = 0
				wp.Water = 0
				wp.Fertilizer = 0
				wp.Health = 0
				wp.Infested = false
			} else {
				if wp.Health > 30.0 {
					growthRate := 4.0
					if wp.Infested {
						growthRate *= 0.5
					}
					if now.Before(wp.LightUntil) {
						growthRate *= 2.0
					}
					wp.Growth += elapsed * growthRate
					if wp.Growth > 100 {
						wp.Growth = 100
					}
				}
			}

			if !wp.Infested && wp.Growth > 0 && wp.Growth < 100 {
				prob := 1.0 - math.Pow(0.92, elapsed)
				if rand.Float64() < prob {
					wp.Infested = true
					wp.PestUntil = now
				}
			}

			wp.LastAction = now
			_ = ctx.DB.SaveWeedPlant(gid, wp)
		}

		if len(ctx.Args) > 0 {
			sub := strings.ToLower(ctx.Args[0])
			switch sub {
			case "water":
				if now.Before(wp.LastWater.Add(15 * time.Minute)) {
					left := wp.LastWater.Add(15 * time.Minute).Sub(now)
					return ctx.Reply(fmt.Sprintf("chill, it was just watered. wait **%d min %d sec**.", int(left.Minutes()), int(left.Seconds())%60))
				}
				wp.Water = 80.0
				wp.LastWater = now
				wp.LastAction = now
				if wp.Health <= 0 {
					wp.Health = 20.0
				}
				_ = ctx.DB.SaveWeedPlant(gid, wp)
				return ctx.Reply("💧 splashed some water on it. moisture's at **80.0%** now.")

			case "fertilize":
				if now.Before(wp.LastFert.Add(30 * time.Minute)) {
					left := wp.LastFert.Add(30 * time.Minute).Sub(now)
					return ctx.Reply(fmt.Sprintf("too much nutrients will burn it. wait **%d min %d sec**.", int(left.Minutes()), int(left.Seconds())%60))
				}
				wp.Fertilizer = 80.0
				wp.LastFert = now
				wp.LastAction = now
				_ = ctx.DB.SaveWeedPlant(gid, wp)
				return ctx.Reply("🧪 tossed some fertilizer in the soil. nutrients at **80.0%**.")

			case "light":
				if now.Before(wp.LastLight.Add(2 * time.Hour)) {
					left := wp.LastLight.Add(2 * time.Hour).Sub(now)
					return ctx.Reply(fmt.Sprintf("lights are cooling down. wait **%d hours, %d min**.", int(left.Hours()), int(left.Minutes())%60))
				}
				wp.LightUntil = now.Add(1 * time.Hour)
				wp.LastLight = now
				wp.LastAction = now
				_ = ctx.DB.SaveWeedPlant(gid, wp)
				return ctx.Reply("💡 flipped the LED grow lights on. double growth speed for the next hour.")

			case "spray":
				if !wp.Infested {
					return ctx.Reply("nah, no bugs on it right now. don't drown it in chemicals.")
				}
				wp.Infested = false
				wp.LastAction = now
				_ = ctx.DB.SaveWeedPlant(gid, wp)
				return ctx.Reply("✨ sprayed the bugs. they're dead now.")

			case "harvest":
				if wp.Growth < 100 {
					return ctx.Reply(fmt.Sprintf("nah, it isn't ready to cut yet. growth is only at **%.1f%%**.", wp.Growth))
				}
				yield := rand.Intn(26) + 15
				wp.Growth = 0
				wp.Water = 40
				wp.Fertilizer = 40
				wp.Health = 100
				wp.Infested = false
				wp.Yields += yield
				wp.LastAction = now
				_ = ctx.DB.SaveWeedPlant(gid, wp)
				return ctx.Reply(fmt.Sprintf("🍁 **HARVESTED!** <@%s> cut the plant down and got **%d grams** of top-shelf bud. Total stash is at **%d grams**.", uid, yield, wp.Yields))
			}
		}

		botCfg, _ := ctx.DB.GetBot(ctx.Session.State.User.ID)
		resCfg := config.Resolve(config.GetGlobal(), botCfg)
		emb, comps := BuildWeedMessage(resCfg, wp, 0, gid)

		if ctx.Interact != nil {
			return ctx.Session.InteractionRespond(ctx.Interact, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds:     []*discordgo.MessageEmbed{emb},
					Components: comps,
				},
			})
		}

		_, err = ctx.Session.ChannelMessageSendComplex(ctx.ChanID(), &discordgo.MessageSend{
			Embeds:     []*discordgo.MessageEmbed{emb},
			Components: comps,
		})
		return err
	},
}
