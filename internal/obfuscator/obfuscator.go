package obfuscator
import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
)
type Song struct {
	ID    string
	Title string
	Lines []string
}
type Opts struct {
	PlaceID      int
	AntiTamper   string
	UseDoubleVM  bool
	UseLyrics    bool
	CustomLyrics string
	SelectedSong string
}
var lyricBank = []Song{
	{
		ID:    "astley",
		Title: "Never Gonna Give You Up - Rick Astley",
		Lines: []string{
			"We're no strangers to love",
			"You know the rules and so do I (do I)",
			"A full commitment's what I'm thinking of",
			"You wouldn't get this from any other guy",
			"I just wanna tell you how I'm feeling",
			"Gotta make you understand",
			"Never gonna give you up",
			"Never gonna let you down",
			"Never gonna run around and desert you",
			"Never gonna make you cry",
			"Never gonna say goodbye",
			"Never gonna tell a lie and hurt you",
		},
	},
	{
		ID:    "soad",
		Title: "Chop Suey! - System of a Down",
		Lines: []string{
			"Wake up (wake up)",
			"Grab a brush and put a little makeup",
			"Hide the scars to fade away the shakeup",
			"Why'd you leave the keys upon the table?",
			"Here you go create another fable",
			"You wanted to!",
			"I don't think you trust in my self-righteous suicide",
			"I cry when angels deserve to die",
		},
	},
	{
		ID:    "linkin",
		Title: "In The End - Linkin Park",
		Lines: []string{
			"It starts with one thing, I don't know why",
			"It doesn't even matter how hard you try",
			"Keep that in mind, I designed this rhyme",
			"To explain in due time",
			"All I know is time is a valuable thing",
			"Watch it fly by as the pendulum swings",
			"I wasted it all just to watch you go",
			"I kept everything inside and even though I tried, it all fell apart",
			"What it meant to me will eventually be a memory of a time when I tried so hard",
		},
	},
	{
		ID:    "daft",
		Title: "Harder, Better, Faster, Stronger - Daft Punk",
		Lines: []string{
			"Work it, make it, do it, makes us",
			"Harder, better, faster, stronger",
			"More than, hour, our, never",
			"Ever, after, work is, over",
			"Work it harder, make it better",
			"Do it faster, makes us stronger",
			"More than ever, hour after",
			"Our work is never over",
		},
	},
	{
		ID:    "runaway",
		Title: "Runaway - Kanye West",
		Lines: []string{
			"And I always find, yeah, I always find something wrong",
			"You been putting up with my shit for just too long",
			"I'm so gifted at finding what I don't like the most",
			"So a toast to the scumbags, toast to the jerks",
			"Toast to the classless, toast to the fools",
			"Let's have a toast for the douchebags",
			"Let's have a toast for the assholes",
			"Let's have a toast for the scumbags",
			"Every one of them that I know",
			"Baby, write back, write back, write back",
			"Run away, run away, run away, run away",
			"I think it's time for us to go",
		},
	},
	{
		ID:    "heartless",
		Title: "Heartless - Kanye West",
		Lines: []string{
			"In the night, I hear them talk",
			"The coldest story ever told",
			"Somewhere far along this road",
			"He lost his soul to a woman so heartless",
			"How could you be so heartless?",
			"Oh, how could you be so heartless?",
		},
	},
	{
		ID:    "power",
		Title: "Power - Kanye West",
		Lines: []string{
			"I guess every superhero need his theme music",
			"No one man should have all that power",
			"The clock's ticking, I just count the hours",
			"Stop tripping, I'm tripping off the power",
			"21st-century schizoid man",
		},
	},
	{
		ID:    "iwonder",
		Title: "I Wonder - Kanye West",
		Lines: []string{
			"Find your dreams come true",
			"And I wonder if you know what it means",
			"To find your dreams come true",
			"I've been waiting on this my whole life",
			"These dreams be waking me up at night",
			"You say he get on your nerves",
		},
	},
	{
		ID:    "devil",
		Title: "Devil In A New Dress - Kanye West",
		Lines: []string{
			"I love it when you're angry, angry",
			"We love Jesus, but she done learned a lot from Satan",
			"Hard to be humble when you stuntin' on a Jumbotron",
			"I'm double-parked, outside of the steakhouse",
			"You love me for me, could you love me for the money?",
			"Put your hands up in the sky",
		},
	},
	{
		ID:    "touchsky",
		Title: "Touch The Sky - Kanye West",
		Lines: []string{
			"I gotta testify, come up in the spot looking extra fly",
			"For the day I die, I'm gonna touch the sky",
			"Back when Gucci was the brand to buy",
			"I'm trying to catch the beat, trying to touch the sky",
			"I guess we at the top, baby",
		},
	},
	{
		ID:    "wire",
		Title: "Through The Wire - Kanye West",
		Lines: []string{
			"Through the fire, to the limit, to the wall",
			"For a chance to be with you, I'd gladly risk it all",
			"What if somebody told you that I was almost gone?",
			"They can't stop me from writing, through the wire",
			"I drink a Boost for breakfast, an Ensure for dessert",
			"Thank God I ain't too cool for the safe belt",
		},
	},
	{
		ID:    "homecoming",
		Title: "Homecoming - Kanye West",
		Lines: []string{
			"I met this girl when I was three years old",
			"And what I loved most she had so much soul",
			"She said she was from the streets, she was Windy",
			"Do you think about me now and then?",
			"Now I'm back, homecoming Chicago",
		},
	},
	{
		ID:    "stronger",
		Title: "Stronger - Kanye West",
		Lines: []string{
			"Work it, make it, do it, makes us",
			"Harder, better, faster, stronger",
			"N-now th-that that don't kill me",
			"Can only make me stronger",
			"I need you to hurry up now",
			"Cause I can't wait much longer",
		},
	},
	{
		ID:    "nochurch",
		Title: "No Church In The Wild - Kanye West",
		Lines: []string{
			"Human beings in a mob",
			"What's a mob to a king?",
			"What's a king to a god?",
			"What's a god to a non-believer",
			"Who don't believe in anything?",
			"We make it out alive, church in the wild",
		},
	},
	{
		ID:    "bound2",
		Title: "Bound 2 - Kanye West",
		Lines: []string{
			"Bound to fall in love",
			"Bound to fall in love with you",
			"Uh-huh, honey",
			"I wanna fuck you for the second time in the morning",
			"Leave you for the morning, bound to fall in love",
			"Close your eyes and let the word paint a thousand pictures",
		},
	},
	{
		ID:    "golddigger",
		Title: "Gold Digger - Kanye West",
		Lines: []string{
			"She take my money when I'm in need",
			"Yeah, she's a trifling friend indeed",
			"Oh, she's a gold digger, way over town",
			"That digs on me, gold digger",
			"Now I ain't saying she a gold digger",
			"But she ain't messing with no broke niggas",
		},
	},
	{
		ID:    "paris",
		Title: "Niggas in Paris - Kanye West & Jay-Z",
		Lines: []string{
			"Ball so hard motherfuckers wanna fine me",
			"But first niggas gotta find me",
			"What's fifty grand to a motherfucker like me?",
			"Can you please remind me?",
			"Niggas in Paris, that shit cray",
			"Got my niggas in Paris and they going gorillas",
		},
	},
	{
		ID:    "lights",
		Title: "All Of The Lights - Kanye West",
		Lines: []string{
			"All of the lights, all of the lights",
			"Cop lights, flashlights, spotlights",
			"Strobe lights, streetlights, fast life",
			"Turn up the lights in here, baby",
			"Extra bright, I want y'all to see this",
			"Fast life, keep it moving",
		},
	},
	{
		ID:    "wecare",
		Title: "We Don't Care - Kanye West",
		Lines: []string{
			"We don't care what people say",
			"If this is your first time hearing this",
			"You are about to experience something so cold",
			"Drug dealer buy back the neighborhood",
			"And we don't care what people say",
			"We gonna make it anyway",
		},
	},
	{
		ID:    "wolves",
		Title: "Wolves - Kanye West",
		Lines: []string{
			"You gotta let me know",
			"I'm just a bad boy, I need a good girl",
			"We surrounded by the wolves",
			"You left your fridge open, somebody just took a sandwich",
			"Wrap your arms around me, wolves",
		},
	},
	{
		ID:    "onsight",
		Title: "On Sight - Kanye West",
		Lines: []string{
			"Kanye West head of the table, on sight",
			"Yeezy season approaching, fuck whatever y'all been hearing",
			"Fuck whatever y'all been wearing",
			"A monster about to come alive again",
			"He'll give us what we need, not what we want",
		},
	},
	{
		ID:    "fatherstretch",
		Title: "Father Stretch My Hands (Pt. 1 & Pt. 2) - Kanye West",
		Lines: []string{
			"You're the only power, you're the only power",
			"Father stretch my hands, stretch my hands to you",
			"If I don't trust you, I'd rather be blind",
			"Perfect afternoon, bleach on my T-shirt",
			"I just wanna feel liberated, I, I",
			"Up in the morning, miss you bad",
			"Now I'm in the streets, stretch my hands",
		},
	},
}
func init() {
	rand.Seed(time.Now().UnixNano())
}
func ri(lo, hi int) int {
	if lo >= hi {
		return lo
	}
	return rand.Intn(hi-lo+1) + lo
}
func cleanWord(w string) string {
	var sb strings.Builder
	for _, r := range w {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
func wrapWithIR(luaCode, antiTamperTemplate string, cfg Opts) string {
	var song Song
	if cfg.CustomLyrics != "" {
		lines := strings.Split(cfg.CustomLyrics, "\n")
		var cleanLines []string
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if l != "" {
				cleanLines = append(cleanLines, l)
			}
		}
		if len(cleanLines) == 0 {
			cleanLines = []string{"default custom lyric placeholder"}
		}
		song = Song{
			Title: "Custom User Soundtrack",
			Lines: cleanLines,
		}
	} else if cfg.SelectedSong != "" && cfg.SelectedSong != "random" {
		var found bool
		for _, s := range lyricBank {
			if s.ID == cfg.SelectedSong || strings.Contains(strings.ToLower(s.Title), strings.ToLower(cfg.SelectedSong)) {
				song = s
				found = true
				break
			}
		}
		if !found {
			song = lyricBank[rand.Intn(len(lyricBank))]
		}
	} else {
		song = lyricBank[rand.Intn(len(lyricBank))]
	}
	used := make(map[string]bool)
	_n := func() string {
		if cfg.UseLyrics {
			for attempt := 0; attempt < 2000; attempt++ {
				line := song.Lines[rand.Intn(len(song.Lines))]
				parts := strings.Fields(line)
				var words []string
				for _, w := range parts {
					cw := cleanWord(w)
					if len(cw) > 0 {
						words = append(words, cw)
					}
				}
				if len(words) > 0 {
					wLen := ri(2, 4)
					if wLen > len(words) {
						wLen = len(words)
					}
					start := ri(0, len(words)-wLen)
					slice := words[start : start+wLen]
					var name string
					for _, sw := range slice {
						name += strings.Title(strings.ToLower(sw))
					}
					if len(name) > 2 {
						if used[name] {
							name += strconv.Itoa(ri(10, 999))
						}
						if !used[name] {
							used[name] = true
							return name
						}
					}
				}
			}
			var allWords []string
			for _, line := range song.Lines {
				for _, w := range strings.Fields(line) {
					cw := cleanWord(w)
					if len(cw) > 2 {
						allWords = append(allWords, cw)
					}
				}
			}
			if len(allWords) > 0 {
				for attempt := 0; attempt < 1000; attempt++ {
					var name string
					count := ri(2, 4)
					for j := 0; j < count; j++ {
						w := allWords[rand.Intn(len(allWords))]
						name += strings.Title(strings.ToLower(w))
					}
					if used[name] {
						name += strconv.Itoa(ri(10, 999))
					}
					if !used[name] {
						used[name] = true
						return name
					}
				}
			}
		}
		themes := []func() string{
			func() string {
				n := "_0x"
				for j := 0; j < 6; j++ {
					n += string("0123456789ABCDEF"[rand.Intn(16)])
				}
				return n
			},
		}
		for a := 0; a < 50000; a++ {
			theme := themes[rand.Intn(len(themes))]
			n := theme()
			if !used[n] {
				used[n] = true
				return n
			}
		}
		panic("name collision")
	}
	const pool = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz"
	alphaRunes := []rune(pool)
	rand.Shuffle(len(alphaRunes), func(i, j int) {
		alphaRunes[i], alphaRunes[j] = alphaRunes[j], alphaRunes[i]
	})
	alpha := string(alphaRunes[:16])
	rawSeeds := make([]int, 6)
	for i := 0; i < 6; i++ {
		rawSeeds[i] = ri(300, 9000)
	}
	seedExprs := make([]string, 6)
	for i, v := range rawSeeds {
		a := ri(1, v-1)
		seedExprs[i] = fmt.Sprintf("%d+%d", a, v-a)
	}
	mix := func(seeds []int, initVal int, pId int) int {
		h := initVal
		if pId > 0 {
			h = (h ^ (pId % 256)) & 0xFF
		}
		for _, v := range seeds {
			h = (h ^ (v & 0xFF)) & 0xFF
			h = (((h << 5) | (h >> 3)) & 0xFF)
		}
		return h
	}
	targetPlaceId := cfg.PlaceID
	k1 := mix(rawSeeds, 0xA5, targetPlaceId)
	var reversedSeeds []int
	for i := len(rawSeeds) - 1; i >= 0; i-- {
		reversedSeeds = append(reversedSeeds, rawSeeds[i])
	}
	k2 := mix(reversedSeeds, 0x5C, targetPlaceId)
	s0 := k1
	s1 := k2
	s2 := (k1 ^ k2) & 0xFF
	if s2 == 0 {
		s2 = 3
	}
	s3 := (k1 + k2) & 0xFF
	if s3 == 0 {
		s3 = 4
	}
	xs := func() int {
		t := int(uint32(s3) ^ (uint32(s3) << 11))
		s3 = s2
		s2 = s1
		s1 = s0
		s0 = int((uint32(s0) ^ (uint32(s0) >> 19)) ^ (uint32(t) ^ (uint32(t) >> 8)))
		return (s0 ^ s1 ^ s2 ^ s3) & 0xFF
	}
	enc := func(s string) string {
		var sb strings.Builder
		for i := 0; i < len(s); i++ {
			o := (int(s[i]) ^ xs()) & 0xFF
			sb.WriteByte(alpha[o>>4])
			sb.WriteByte(alpha[o&0xF])
		}
		return sb.String()
	}
	finalTargetCode := luaCode
	if cfg.UseDoubleVM {
		vNestedCode := _n()
		vNestedDec := _n()
		vNestedXn := _n()
		vDoubleLs := _n()
		vDoubleChar := _n()
		vDoubleBxor := _n()
		vDoubleConcat := _n()
		vDoubleUnpack := _n()
		vDoubleEnv := _n()
		vSInner := _n()
		vSlInner := _n()
		vTempInner := _n()
		vTlInner := _n()
		vKInner := _n()
		vIInner := _n()
		innerKey := ri(30, 200)
		innerStep := ri(2, 8)
		encBytes := make([]string, len(luaCode))
		for i := 0; i < len(luaCode); i++ {
			encBytes[i] = strconv.Itoa(int(luaCode[i]) ^ ((innerKey + i*innerStep) % 256))
		}
		var doubleVMLyrics string
		if cfg.UseLyrics {
			var doubleVMLyricVars []string
			numLines := 3
			if len(song.Lines) < numLines {
				numLines = len(song.Lines)
			}
			for j := 0; j < numLines; j++ {
				line := song.Lines[rand.Intn(len(song.Lines))]
				varName := _n()
				escaped := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(line, "\\", "\\\\"), "\"", "\\\""), "\n", "\\n")
				doubleVMLyricVars = append(doubleVMLyricVars, fmt.Sprintf("local %s = \"%s\"", varName, escaped))
			}
			doubleVMLyrics = strings.Join(doubleVMLyricVars, " ") + " "
		}
		finalTargetCode = fmt.Sprintf(
			"return function(%s, %s, %s, %s, %s) %slocal %s = {%s} local function %s(t) local %s = {} local %s = 0 local %s = {} local %s = 0 local %s = %d for %s = 1, #t do %s = %s + 1 %s[%s] = %s(t[%s], %s) %s = (%s + %d) %% 256 if %s >= 2000 or %s == #t then %s = %s + 1 %s[%s] = %s(%s(%s, 1, %s)) %s = {} %s = 0 end end return %s(%s) end local %s = (getfenv and getfenv()) or _G if %s and type(%s) == \"table\" then %s.loadstring = %s.loadstring or %s %s.load = %s.load or %s end if type(_G) == \"table\" then _G.loadstring = _G.loadstring or %s _G.load = _G.load or %s end local %s = %s(%s(%s)) if %s then %s() end end",
			vDoubleLs, vDoubleChar, vDoubleBxor, vDoubleConcat, vDoubleUnpack, doubleVMLyrics, vNestedCode, strings.Join(encBytes, ","), vNestedDec, vSInner, vSlInner, vTempInner, vTlInner, vKInner, innerKey, vIInner, vTlInner, vTlInner, vTempInner, vTlInner, vDoubleBxor, vIInner, vKInner, vKInner, vKInner, innerStep, vTlInner, vIInner, vSlInner, vSlInner, vSInner, vSlInner, vDoubleChar, vDoubleUnpack, vTempInner, vTlInner, vTempInner, vTlInner, vDoubleConcat, vSInner, vDoubleEnv, vDoubleEnv, vDoubleEnv, vDoubleEnv, vDoubleEnv, vDoubleLs, vDoubleEnv, vDoubleEnv, vDoubleLs, vDoubleLs, vDoubleLs, vNestedXn, vDoubleLs, vNestedDec, vNestedCode, vNestedXn, vNestedXn,
		)
	}
	nChunks := ri(18, 25)
	plaintexts := make([]string, nChunks)
	pos := 0
	for i := 0; i < nChunks; i++ {
		rem := len(finalTargetCode) - pos
		if i == nChunks-1 {
			plaintexts[i] = finalTargetCode[pos:]
			break
		}
		avg := rem / (nChunks - i)
		if avg <= 1 {
			plaintexts[i] = finalTargetCode[pos : pos+1]
			pos += 1
			continue
		}
		minTake := avg / 2
		if minTake < 1 {
			minTake = 1
		}
		maxTake := avg * 2
		if maxTake >= rem {
			maxTake = rem - 1
		}
		take := ri(minTake, maxTake)
		plaintexts[i] = finalTargetCode[pos : pos+take]
		pos += take
	}

	execOrder := make([]int, nChunks)
	for i := 0; i < nChunks; i++ {
		execOrder[i] = i
	}
	rand.Shuffle(len(execOrder), func(i, j int) {
		execOrder[i], execOrder[j] = execOrder[j], execOrder[i]
	})

	mkSt := func() func() int {
		u := make(map[int]bool)
		return func() int {
			for {
				s := ri(1000, 9999)
				if !u[s] {
					u[s] = true
					return s
				}
			}
		}
	}()
	stateKey := ri(50, 255)
	stateStep := ri(1, 10)
	stateIds := make([]int, nChunks+3)
	for i := 0; i < len(stateIds); i++ {
		stateIds[i] = mkSt()
	}
	deadSid := mkSt()

	chunks := make([]string, nChunks)
	destIdxKeys := make([]int, nChunks)
	transKeys := make([]int, nChunks)

	firstStateXorVal := xs()
	firstStateTransKey := (stateIds[2]*stateStep + stateKey) ^ firstStateXorVal

	for step := 0; step < nChunks; step++ {
		chunkIdx := execOrder[step]
		chunks[chunkIdx] = enc(plaintexts[chunkIdx])
		val := xs()
		destIdxKeys[chunkIdx] = (chunkIdx + 1) ^ val
		cs := 0
		for _, b := range plaintexts[chunkIdx] {
			cs = (cs*31 + int(b)) % 256
		}
		s0 = (s0 ^ cs) & 0xFF

		valNext := xs()
		nextStateIndex := step + 1
		var nextStateId int
		if nextStateIndex == nChunks {
			nextStateId = stateIds[len(stateIds)-1]
		} else {
			nextStateId = stateIds[2+nextStateIndex]
		}
		transKeys[chunkIdx] = (nextStateId*stateStep + stateKey) ^ valNext
	}
	vKillFlag := _n()
	vDec := _n()
	vJunkTable := _n()
	var junkVals []string
	for i := 0; i < 16; i++ {
		junkVals = append(junkVals, strconv.Itoa(ri(1, 255)))
	}
	junkData := strings.Join(junkVals, ",")

	vSpec := _n()
	vLibTable := _n()
	names := []string{"bit32", "string", "table", "math", "task", "game", "setmetatable", "rawset", "rawget", "getfenv", "pcall", "pairs", "coroutine", "loadstring", "load", "unpack", "debug", "next"}

	rand.Shuffle(len(names), func(i, j int) {
		names[i], names[j] = names[j], names[i]
	})

	libMap := make(map[string]int)
	for i, name := range names {
		libMap[name] = i + 1
	}

	S := func(lib string) string {
		return fmt.Sprintf("%s[%d]", vSpec, libMap[lib])
	}
	getJunk := func() string {
		v1 := _n()
		v2 := _n()
		v3 := _n()
		if cfg.UseLyrics && rand.Float32() < 0.6 {
			line := song.Lines[rand.Intn(len(song.Lines))]
			escaped := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(line, "\\", "\\\\"), "\"", "\\\""), "\n", "\\n")
			return fmt.Sprintf("local %s = \"%s\" if not %s then local %s = %d return %s end ", v1, escaped, v1, v2, ri(1, 100), v2)
		}
		ops := []string{"+", "-", "*", "%"}
		op := ops[rand.Intn(len(ops))]
		val1 := ri(1, 1000)
		val2 := ri(1001, 2000)
		var cond string
		if rand.Intn(2) == 0 {
			cond = fmt.Sprintf("%d < %d", val1, val2)
		} else {
			cond = fmt.Sprintf("%d > %d", val2, val1)
		}
		return fmt.Sprintf("local %s=%d if %s then local function %s(%s) return %s%s(%s or %d) end end ", v1, ri(1, 1000), cond, v2, v3, v1, op, v3, ri(1, 100))
	}
	processTemplate := func(template string) string {
		if template == "" {
			return ""
		}
		reBC := regexp.MustCompile(`--\[\[[\s\S]*?\]\]`)
		t := reBC.ReplaceAllString(template, "")
		reLC := regexp.MustCompile(`(?m)--[^\r\n]*`)
		t = reLC.ReplaceAllString(t, "")
		junkValuesStr := strings.Split(junkData, ",")
		var junkValues []int
		for _, valStr := range junkValuesStr {
			v, _ := strconv.Atoi(valStr)
			junkValues = append(junkValues, v)
		}
		reStr := regexp.MustCompile(`"([^"]*)"`)
		t = reStr.ReplaceAllStringFunc(t, func(match string) string {
			p1 := match[1 : len(match)-1]
			if len(p1) == 0 {
				return `""`
			}
			choice := rand.Intn(3)
			vInnerS := _n()
			vInnerK := _n()
			vInnerI := _n()
			vInnerV := _n()

			switch choice {
			case 0:
				// Type A: Feedback XOR cipher
				initialKey := ri(50, 200)
				step := ri(1, 10)
				currentKey := initialKey
				var bytes []string
				for i := 0; i < len(p1); i++ {
					b := int(p1[i]) ^ currentKey
					bytes = append(bytes, strconv.Itoa(b))
					currentKey = (currentKey + b + step) % 256
				}
				vDecoded := _n()
				vInnerArgs := _n()
				return fmt.Sprintf("(function() local %s={} local %s=%d local %s={%s} for %s=1,#%s do local %s=%s[%s] local %s=%s.bxor(%s,%s) %s[%s]=%s %s=(%s+%s+%d)%%256 end return %s.char(%s(%s,1,#%s)) end)()",
					vInnerS, vInnerK, initialKey, vInnerArgs, strings.Join(bytes, ","), vInnerI, vInnerArgs, vInnerV, vInnerArgs, vInnerI, vDecoded, S("bit32"), vInnerV, vInnerK, vInnerS, vInnerI, vDecoded, vInnerK, vInnerK, vInnerV, step, S("string"), S("unpack"), vInnerS, vInnerS)

			case 1:
				// Type B: Additive/subtractive cipher with dynamic step LCG
				initialKey := ri(50, 200)
				lcgMult := ri(3, 15)*2 + 1
				lcgInc := ri(3, 29)*2 + 1
				currentKey := initialKey
				var bytes []string
				for i := 0; i < len(p1); i++ {
					b := (int(p1[i]) + currentKey) % 256
					bytes = append(bytes, strconv.Itoa(b))
					currentKey = (currentKey*lcgMult + lcgInc + b) % 256
				}
				vInnerArgs := _n()
				return fmt.Sprintf("(function() local %s={} local %s=%d local %s={%s} for %s=1,#%s do local %s=%s[%s] %s[%s]=(%s-%s+256)%%256 %s=(%s*%d+%d+%s)%%256 end return %s.char(%s(%s,1,#%s)) end)()",
					vInnerS, vInnerK, initialKey, vInnerArgs, strings.Join(bytes, ","), vInnerI, vInnerArgs, vInnerV, vInnerArgs, vInnerI, vInnerS, vInnerI, vInnerV, vInnerK, vInnerK, vInnerK, lcgMult, lcgInc, vInnerV, S("string"), S("unpack"), vInnerS, vInnerS)

			default:
				// Type C: Multiplicative rolling XOR using Junk Table
				initialKey := ri(50, 200)
				step := ri(1, 10)
				junkIdx := ri(0, 15)
				currentKey := initialKey
				var bytes []string
				for i := 0; i < len(p1); i++ {
					junkVal := 0
					if len(junkValues) > 0 {
						junkVal = junkValues[(i+1+junkIdx)%len(junkValues)]
					}
					key := (currentKey*3 + junkVal) % 256
					b := int(p1[i]) ^ key
					bytes = append(bytes, strconv.Itoa(b))
					currentKey = (currentKey + step) % 256
				}
				vInnerKey := _n()
				vInnerArgs := _n()
				return fmt.Sprintf("(function() local %s={} local %s=%d local %s={%s} for %s=1,#%s do local %s=%s[%s] local %s=(%s*3+(%s[(%s+%d)%%#%s+1] or 0))%%256 %s[%s]=%s.bxor(%s,%s) %s=(%s+%d)%%256 end return %s.char(%s(%s,1,#%s)) end)()",
					vInnerS, vInnerK, initialKey, vInnerArgs, strings.Join(bytes, ","), vInnerI, vInnerArgs, vInnerV, vInnerArgs, vInnerI, vInnerKey, vInnerK, vJunkTable, vInnerI, junkIdx, vJunkTable, vInnerS, vInnerI, S("bit32"), vInnerV, vInnerKey, vInnerK, vInnerK, step, S("string"), S("unpack"), vInnerS, vInnerS)
			}
		})
		reId := regexp.MustCompile(`\b_[a-zA-Z0-9_]+\b`)
		ids := reId.FindAllString(t, -1)
		idMap := make(map[string]string)
		protectedNames := map[string]bool{vSpec: true, vJunkTable: true, vKillFlag: true, "_G": true}
		for _, id := range ids {
			if protectedNames[id] {
				continue
			}
			if _, exists := idMap[id]; !exists {
				idMap[id] = _n()
			}
		}
		t = strings.ReplaceAll(t, "ESOTERIC_KILLED", vKillFlag)
		reG := regexp.MustCompile(`_G[ \t]*\.[ \t]*([a-zA-Z0-9_]+)`)
		t = reG.ReplaceAllStringFunc(t, func(m string) string {
			p1 := reG.FindStringSubmatch(m)[1]
			if p1 == vKillFlag {
				return fmt.Sprintf("_G[%q]", vKillFlag)
			}
			return m
		})
		return reId.ReplaceAllStringFunc(t, func(m string) string {
			if mapped, ok := idMap[m]; ok {
				return mapped
			}
			return m
		})
	}
	randomizedAntiTamper := processTemplate(antiTamperTemplate)
	var cmEntriesList []string
	for i := 0; i < len(alpha); i++ {
		cmEntriesList = append(cmEntriesList, fmt.Sprintf("[%d]=%d", alpha[i], i))
	}
	cmEntries := strings.Join(cmEntriesList, ",")
	vSm := _n()
	vSmD := _n()
	vSt := _n()
	vCm := _n()
	vUb := _n()
	vBuf := _n()
	vXn := _n()
	var vSv []string
	for i := 0; i < 6; i++ {
		vSv = append(vSv, _n())
	}
	vFn := _n()
	vChArr := _n()
	vXs0 := _n()
	vXs1 := _n()
	vXs2 := _n()
	vXs3 := _n()
	stDone := 0
	var states [][]string
	nJunk := ri(3, 6)
	nFake := ri(3, 6)
	if targetPlaceId > 0 {
		states = append(states, []string{strconv.Itoa(stateIds[0]), fmt.Sprintf("if (%s and %s.PlaceId or 0)==%d then %s=%d else %s=%d end", S("game"), S("game"), targetPlaceId, vSt, stateIds[1]*stateStep+stateKey, vSt, deadSid*stateStep+stateKey)})
	} else {
		states = append(states, []string{strconv.Itoa(stateIds[0]), fmt.Sprintf("%s=%d", vSt, stateIds[1]*stateStep+stateKey)})
	}
	states = append(states, []string{strconv.Itoa(deadSid), fmt.Sprintf("while true do %s.wait(9e9) end", S("task"))})
	vPairsCount := _n()
	vK := _n()
	states = append(states, []string{strconv.Itoa(stateIds[1]), fmt.Sprintf("local %s=0 local %s=nil while true do %s=%s(%s,%s) if not %s then break end %s=%s+1 end if %s~=%d then while true do %s.wait(9e9) end end local xOpt=math.random(10,100) if (xOpt*xOpt+xOpt)%%2==0 then %s=%s.bxor(%d,%s()) else %s=%d end",
		vPairsCount, vK, vK, S("next"), vSmD, vK, vK, vPairsCount, vPairsCount, vPairsCount, len(stateIds)+1+nJunk+nFake, S("task"), vSt, S("bit32"), firstStateTransKey, vXn, vSt, deadSid*stateStep+stateKey)})
	for i := 0; i < nChunks; i++ {
		chunkIdx := execOrder[i]
		sid := stateIds[2+i]

		vChunkH := _n()
		vChunkHb := _n()
		vChunkB := _n()
		vChunkBl := _n()
		vChunkTemp := _n()
		vChunkTl := _n()
		vChunkJ := _n()
		vCs := _n()
		vCi := _n()
		vDestIdx := _n()
		// decode loop
		decodePart := fmt.Sprintf(
			"local %s=%q local %s=%s(%s) local %s={} local %s=0 local %s={} local %s=0 for %s=1,#%s,2 do %s=%s+1 %s[%s]=%s.bxor(%s[%s[%s]]*16+%s[%s[%s+1]],%s()) if %s>=2000 or %s>=#%s-1 then %s=%s+1 %s[%s]=%s.char(%s(%s,1,%s)) %s={} %s=0 end end local %s=%s.bxor(%d,%s()) %s[%s]=%s.concat(%s)",
			vChunkH, chunks[chunkIdx], vChunkHb, vUb, vChunkH,
			vChunkB, vChunkBl, vChunkTemp, vChunkTl,
			vChunkJ, vChunkH,
			vChunkTl, vChunkTl, vChunkTemp, vChunkTl,
			S("bit32"), vCm, vChunkHb, vChunkJ, vCm, vChunkHb, vChunkJ, vXn,
			vChunkTl, vChunkJ, vChunkH,
			vChunkBl, vChunkBl, vChunkB, vChunkBl,
			S("string"), S("unpack"), vChunkTemp, vChunkTl,
			vChunkTemp, vChunkTl,
			vDestIdx, S("bit32"), destIdxKeys[chunkIdx], vXn, vBuf, vDestIdx, S("table"), vChunkB,
		)
		// trace hash update
		traceHashPart := fmt.Sprintf(
			" do _G[\"%s\"] = (((_G[\"%s\"] or 0) * 31) + %d) %% 4294967296 end",
			vKillFlag+"_trace", vKillFlag+"_trace", sid,
		)

		// Varying opaque predicates
		var predicateStr string
		trapVal := (ri(1000, 9999)*stateStep + stateKey)
		vXopt := _n()
		switch ri(0, 2) {
		case 0:
			// Predicate 1: x * (x + 1) % 2 == 0
			predicateStr = fmt.Sprintf("local %s=math.random(10,100) if (%s*(%s+1))%%2==0 then %s=%s.bxor(%d,%s()) else %s=%d end",
				vXopt, vXopt, vXopt, vSt, S("bit32"), transKeys[chunkIdx], vXn, vSt, trapVal)
		case 1:
			// Predicate 2: (x^2 | x) & 1 == 0
			// (x^2 | x) & 1 is always odd for odd x and even for even x, wait, x^2 has same parity as x, so x^2 | x has same parity.
			// Let's use (x * 2) % 2 == 0 which is trivially always true.
			predicateStr = fmt.Sprintf("local %s=math.random(10,100) if (%s*2)%%2==0 then %s=%s.bxor(%d,%s()) else %s=%d end",
				vXopt, vXopt, vSt, S("bit32"), transKeys[chunkIdx], vXn, vSt, trapVal)
		default:
			// Predicate 3: constant hash condition
			hVal := 0
			for j := 1; j <= 5; j++ {
				hVal = (hVal*31 + j) % 10000
			}
			vJ := _n()
			predicateStr = fmt.Sprintf("local %s=0 for %s=1,5 do %s=(%s*31+%s)%%10000 end if %s==%d then %s=%s.bxor(%d,%s()) else %s=%d end",
				vXopt, vJ, vXopt, vXopt, vJ, vXopt, hVal, vSt, S("bit32"), transKeys[chunkIdx], vXn, vSt, trapVal)
		}

		// integrity checksum — fold last chunk's plaintext hash into vXs0 to chain layers
		chainPart := fmt.Sprintf(
			" do local %s=0 for %s=1,#%s[%s] do %s=(%s*31+%s[%s]:byte(%s))%%256 end %s=%s.bxor(%s,%s)%%256 end %s %s",
			vCs, vCi, vBuf, vDestIdx,
			vCs, vCs, vBuf, vDestIdx, vCi,
			vXs0, S("bit32"), vXs0, vCs,
			traceHashPart, predicateStr,
		)
		body := decodePart + chainPart
		states = append(states, []string{strconv.Itoa(sid), body})

	}
	// Calculate expected trace hash
	expectedTraceHash := 0
	for i := 0; i < nChunks; i++ {
		sid := stateIds[2+i]
		expectedTraceHash = ((expectedTraceHash * 31) + sid) % 4294967296
	}

	for i := 0; i < nJunk; i++ {
		junkSid := mkSt()
		junkBody := fmt.Sprintf(
			"local xOpt=math.random(10,100) if (xOpt*(xOpt+1))%%2==0 then %s=%d else %s=%d end",
			vSt, deadSid*stateStep+stateKey, vSt, deadSid*stateStep+stateKey,
		)
		states = append(states, []string{strconv.Itoa(junkSid), junkBody})
	}
	
	for i := 0; i < nFake; i++ {
		fakeSid := mkSt()
		garbageLen := ri(80, 200)
		garbageBytes := make([]byte, garbageLen)
		for j := range garbageBytes {
			garbageBytes[j] = byte(ri(32, 126))
		}
		vFkH := _n()
		vFkHb := _n()
		vFkB := _n()
		vFkBl := _n()
		vFkTemp := _n()
		vFkTl := _n()
		vFkJ := _n()
		fakeBody := fmt.Sprintf(
			"local %s=%q local %s=%s(%s) local %s={} local %s=0 local %s={} local %s=0 for %s=1,#%s,2 do %s=%s+1 %s[%s]=%s.bxor(%s[%s[%s]]*16+%s[%s[%s+1]],%s()) if %s>=2000 or %s>=#%s-1 then %s=%s+1 %s[%s]=%s.char(%s(%s,1,%s)) %s={} %s=0 end end",
			vFkH, string(garbageBytes), vFkHb, vUb, vFkH,
			vFkB, vFkBl, vFkTemp, vFkTl,
			vFkJ, vFkH,
			vFkTl, vFkTl, vFkTemp, vFkTl,
			S("bit32"), vCm, vFkHb, vFkJ, vCm, vFkHb, vFkJ, vXn,
			vFkTl, vFkJ, vFkH,
			vFkBl, vFkBl, vFkB, vFkBl,
			S("string"), S("unpack"), vFkTemp, vFkTl,
			vFkTemp, vFkTl,
		)
		states = append(states, []string{strconv.Itoa(fakeSid), fakeBody})
	}
	vFinalLs := _n()
	vReadIdx := _n()
	vReaderFn := _n()
	states = append(states, []string{strconv.Itoa(stateIds[len(stateIds)-1]), fmt.Sprintf("local %s = 1 local function %s() if %s > #%s then return nil end local data = %s[%s] %s = %s + 1 return data end local %s = %s or %s local %s local success, err = pcall(function() %s = %s(%s) end) if not success or not %s then %s = %s(%s.concat(%s)) end if not %s then return end %s = %d %s = %s",
		vReadIdx, vReaderFn, vReadIdx, vBuf, vBuf, vReadIdx, vReadIdx, vReadIdx, vFinalLs, S("load"), S("loadstring"), vFn, vFn, vFinalLs, vReaderFn, vFn, vFn, vFinalLs, S("table"), vBuf, vFn, vSt, stDone*stateStep+stateKey, vChArr, vFn)})
	vDecKey := _n()
	vDecStep := _n()
	var lyricVars []string
	if cfg.UseLyrics {
		for _, line := range song.Lines {
			words := strings.Fields(line)
			var cleanWords []string
			for _, w := range words {
				cw := cleanWord(w)
				if len(cw) > 2 {
					cleanWords = append(cleanWords, cw)
				}
			}
			var varName string
			if len(cleanWords) > 0 {
				count := ri(2, 3)
				if count > len(cleanWords) {
					count = len(cleanWords)
				}
				for j := 0; j < count; j++ {
					w := cleanWords[rand.Intn(len(cleanWords))]
					if j == 0 {
						varName += strings.Title(strings.ToLower(w))
					} else {
						varName += strings.ToLower(w)
					}
				}
			} else {
				varName = "Lyric"
			}
			varName += strconv.Itoa(ri(10, 999))
			if used[varName] {
				varName += strconv.Itoa(ri(1, 9))
			}
			used[varName] = true
			escaped := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(line, "\\", "\\\\"), "\"", "\\\""), "\n", "\\n")
			lyricVars = append(lyricVars, fmt.Sprintf("local %s = \"%s\"", varName, escaped))
		}
	}
	vDecrypt := _n()
	vDecryptedNames := _n()
	vIdx := _n()
	vS := _n()
	vT := _n()
	vW := _n()
	vI := _n()
	vCounter := _n()
	vDecVal := _n()
	vByteVal := _n()

	var encryptedNamesList []string
	var initialKeys []string
	var keysA []string
	var keysB []string
	var keysC []string

	for _, name := range names {
		k := ri(50, 200)
		a := ri(3, 30)
		b := ri(3, 30)
		c := ri(3, 30)
		initialKeys = append(initialKeys, strconv.Itoa(k))
		keysA = append(keysA, strconv.Itoa(a))
		keysB = append(keysB, strconv.Itoa(b))
		keysC = append(keysC, strconv.Itoa(c))

		var sb strings.Builder
		for idx := 0; idx < len(name); idx++ {
			dec := int(name[idx])
			val := (dec + k) % 256
			sb.WriteString(fmt.Sprintf("\\%d", val))
			k = (a*k*k + b*k + c + (dec % 16)) % 256
			a = (a + dec) % 256
		}
		encryptedNamesList = append(encryptedNamesList, fmt.Sprintf("\"%s\"", sb.String()))
	}
	var lines []string
	vParamEnc := _n()
	vParamK := _n()
	vParamA := _n()
	vParamB := _n()
	vParamC := _n()
	vOutTable := _n()
	vKVar := _n()
	vAVar := _n()
	vBVar := _n()
	vCVar := _n()

	lines = append(lines, fmt.Sprintf("local function %s(%s, %s, %s, %s, %s) local %s = {} local %s = %s local %s, %s, %s = %s, %s, %s for %s = 1, #%s do local %s = string.byte(%s, %s) local %s = (%s - %s + 256) %% 256 %s[%s] = string.char(%s) %s = (%s * %s * %s + %s * %s + %s + (%s %% 16)) %% 256 %s = (%s + %s) %% 256 end return table.concat(%s) end",
		vDecrypt, vParamEnc, vParamK, vParamA, vParamB, vParamC, vOutTable, vKVar, vParamK, vAVar, vBVar, vCVar, vParamA, vParamB, vParamC, vCounter, vParamEnc, vByteVal, vParamEnc, vCounter, vDecVal, vByteVal, vKVar, vOutTable, vCounter, vDecVal, vKVar, vAVar, vKVar, vKVar, vBVar, vKVar, vCVar, vDecVal, vAVar, vAVar, vDecVal, vOutTable))

	lines = append(lines, fmt.Sprintf("local %s = {%s} local %s = {%s} local %s = {%s} local %s = {%s} local %s = {%s} local %s = {} for %s=1,#%s do local name = %s(%s[%s], %s[%s], %s[%s], %s[%s], %s[%s]) local globalVal = _G[name] or getfenv()[name] if name == \"unpack\" then globalVal = table.unpack or unpack elseif name == \"bit32\" and not globalVal then local b = _G.bit or {} globalVal = { bxor = b.bxor or function(x,y) local r,m=0,1 while x>0 or y>0 do if (x%%2)~=(y%%2) then r=r+m end x,y,m=math.floor(x/2),math.floor(y/2),m*2 end return r end, bor = b.bor or function(x,y) local r,m=0,1 while x>0 or y>0 do if (x%%2)==1 or (y%%2)==1 then r=r+m end x,y,m=math.floor(x/2),math.floor(y/2),m*2 end return r end, lshift = b.lshift or function(x,s) return (x*(2^s))%%4294967296 end, rshift = b.rshift or function(x,s) return math.floor(x/(2^s)) end } end %s[%s] = globalVal end",
		vDecryptedNames, strings.Join(encryptedNamesList, ","), vIdx, strings.Join(initialKeys, ","), vS, strings.Join(keysA, ","), vT, strings.Join(keysB, ","), vW, strings.Join(keysC, ","), vLibTable, vI, vDecryptedNames, vDecrypt, vDecryptedNames, vI, vIdx, vI, vS, vI, vT, vI, vW, vI, vLibTable, vI))
	lines = append(lines, fmt.Sprintf("local %s = %s", vSpec, vLibTable))
	vKillFn := _n()
	vDbgCheck := _n()
	vLsCheck := _n()
	vInfoCheck := _n()
	vStartClock := _n()

	lines = append(lines, fmt.Sprintf("local %s = os.clock() local function %s() while true do if %s and %s.wait then %s.wait(9e9) end end end",
		vStartClock, vKillFn, S("task"), S("task"), S("task")))
	lines = append(lines, fmt.Sprintf("local %s = %s local %s = %s or %s if %s then if %s then local success, %s = pcall(%s.getinfo, %s) if success and %s and %s.what ~= \"C\" then %s() end local success4, info4 = pcall(%s.getinfo, 4) if success4 and info4 ~= nil then %s() end end end",
		vDbgCheck, S("debug"), vLsCheck, S("loadstring"), S("load"), vLsCheck, vDbgCheck, vInfoCheck, vDbgCheck, vLsCheck, vInfoCheck, vInfoCheck, vKillFn, vDbgCheck, vKillFn))
	
	// Environment Poison Check (Check that core types of functions have not been tampered with)
	lines = append(lines, fmt.Sprintf("if type(string.byte) ~= \"function\" or type(string.char) ~= \"function\" or type(table.concat) ~= \"function\" or type(type) ~= \"function\" or type(tostring) ~= \"function\" or type(pcall) ~= \"function\" then %s() end", vKillFn))
	
	// Timing checks (wall-clock detection of stepping through debugger)
	lines = append(lines, fmt.Sprintf("if os.clock() - %s > 1.0 then %s() end", vStartClock, vKillFn))

	// hook detection
	lines = append(lines, fmt.Sprintf("if %s and %s.gethook and %s.gethook() then %s() end", vDbgCheck, vDbgCheck, vDbgCheck, vKillFn))

	// getfenv context validation
	lines = append(lines, fmt.Sprintf("if type(getfenv) == \"function\" and getfenv(0) ~= getfenv(1) then %s() end", vKillFn))

	// Anti-Analysis Tool Detection (Dark Dex, SimpleSpy, etc.)
	lines = append(lines, fmt.Sprintf("pcall(function() if game and game.GetService then local cg = game:GetService(\"CoreGui\") if cg then local c = cg:GetChildren() for i = 1, #c do if c[i] then local name = c[i].Name if name == \"Dex\" or name == \"Dark Dex\" or name == \"SimpleSpy\" or name == \"Spy\" then _G[%q] = true %s() end end end end end if gethui then local hui = gethui() if hui then local c = hui:GetChildren() for i = 1, #c do if c[i] then local name = c[i].Name if name == \"Dex\" or name == \"Dark Dex\" or name == \"SimpleSpy\" or name == \"Spy\" then _G[%q] = true %s() end end end end end end)",
		vKillFlag, vKillFn, vKillFlag, vKillFn))

	// loop execution timing check
	vT1 := _n()
	vT2 := _n()
	vLoopI := _n()
	lines = append(lines, fmt.Sprintf("local %s = os.clock() for %s=1,50000 do local _ = %s*%s end local %s = os.clock() if (%s - %s) > 1.0 then %s() end",
		vT1, vLoopI, vLoopI, vLoopI, vT2, vT2, vT1, vKillFn))

	vFuncs := _n()
	vIt := _n()
	vF := _n()
	lines = append(lines, fmt.Sprintf("if %s then local %s = {%s, %s, %s} local %s = nil while true do %s, %s = %s(%s, %s) if not %s then break end if %s then local success, info = pcall(%s.getinfo, %s) if success and info and info.what ~= \"C\" then %s() end end end end",
		vDbgCheck, vFuncs, S("getfenv"), S("pcall"), S("setmetatable"), vIt, vIt, vF, S("next"), vFuncs, vIt, vIt, vF, vDbgCheck, vF, vKillFn))
	if cfg.UseLyrics {
		lines = append(lines, strings.Join(lyricVars, " "))
	}
	lines = append(lines, getJunk())
	lines = append(lines, fmt.Sprintf("local %s={%s}", vJunkTable, junkData))
	lines = append(lines, fmt.Sprintf("local %s=%d local %s=%d", vDecKey, ri(50, 200), vDecStep, ri(1, 10)))
	vDecS := _n()
	vDecK := _n()
	vDecI := _n()
	lines = append(lines, fmt.Sprintf("local function %s(t) local %s={} local %s=%s for %s=1,#t do %s[%s]=%s.bxor(t[%s],(%s+(%s[%s%%#%s+1] or 0))%%256) %s=(%s+t[%s]+%s)%%256 end return %s.char(%s(%s,1,#%s)) end",
		vDec, vDecS, vDecK, vDecKey, vDecI, vDecS, vDecI, S("bit32"), vDecI, vDecK, vJunkTable, vDecI, vJunkTable, vDecK, vDecK, vDecI, vDecStep, S("string"), S("unpack"), vDecS, vDecS))
	lines = append(lines, getJunk())
	lines = append(lines, getJunk())
	lines = append(lines, randomizedAntiTamper)
	lines = append(lines, fmt.Sprintf("local %s={%s}", vCm, cmEntries))
	vUbS := _n()
	vUbT := _n()
	vUbC := _n()
	vUbI := _n()
	vUbR := _n()
	vUbJ := _n()
	lines = append(lines, fmt.Sprintf("local function %s(%s) local %s={} local %s=0 for %s=1,#%s,4096 do local %s={%s.byte(%s,%s,%s.min(%s+4095,#%s))} for %s=1,#%s do %s=%s+1 %s[%s]=%s[%s] end end return %s end",
		vUb, vUbS, vUbT, vUbC, vUbI, vUbS, vUbR, S("string"), vUbS, vUbI, S("math"), vUbI, vUbS, vUbJ, vUbR, vUbC, vUbC, vUbT, vUbC, vUbR, vUbJ, vUbT))
	
	expectedEnvHash := 0
	if targetPlaceId > 0 {
		expectedEnvHash = expectedEnvHash ^ (targetPlaceId % 256)
		expectedEnvHash = ((expectedEnvHash * 31) + 17) % 256
	}

	vEnvHash := _n()
	if targetPlaceId > 0 {
		lines = append(lines, fmt.Sprintf("local %s = 0 if %s and (type(%s) == \"userdata\" or type(%s) == \"table\") then local pId = %s.PlaceId if type(pId) == \"number\" then %s = %s.bxor(%s, pId %% 256) %s = ((%s * 31) + 17) %% 256 end end",
			vEnvHash, S("game"), S("game"), S("game"), S("game"), vEnvHash, S("bit32"), vEnvHash, vEnvHash, vEnvHash))
	} else {
		lines = append(lines, fmt.Sprintf("local %s = 0", vEnvHash))
	}

	lines = append(lines, fmt.Sprintf("local %s,%s,%s,%s,%s,%s=%s",
		vSv[0], vSv[1], vSv[2], vSv[3], vSv[4], vSv[5], strings.Join(seedExprs, ",")))
	vMxName := _n()
	vMxH := _n()
	vMxP := _n()
	vArgs := _n()
	vMxI := _n()
	vV := _n()
	if targetPlaceId > 0 {
		lines = append(lines, fmt.Sprintf("local function %s(%s,...) local %s=(%s and (type(%s) == \"userdata\" or type(%s) == \"table\") and %s.PlaceId or 0) if %s~=0 then %s=%s.bxor(%s,%s%%256) end local %s={...} for %s=1,#%s do local %s=%s[%s] %s=%s.bxor(%s,%s%%256) %s=%s.bor(%s.lshift(%s,5),%s.rshift(%s,3))%%256 end return %s end",
			vMxName, vMxH, vMxP, S("game"), S("game"), S("game"), S("game"), vMxP, vMxH, S("bit32"), vMxH, vMxP, vArgs, vMxI, vArgs, vV, vArgs, vMxI, vMxH, S("bit32"), vMxH, vV, vMxH, S("bit32"), S("bit32"), vMxH, S("bit32"), vMxH, vMxH))
	} else {
		lines = append(lines, fmt.Sprintf("local function %s(%s,...) local %s={...} for %s=1,#%s do local %s=%s[%s] %s=%s.bxor(%s,%s%%256) %s=%s.bor(%s.lshift(%s,5),%s.rshift(%s,3))%%256 end return %s end",
			vMxName, vMxH, vArgs, vMxI, vArgs, vV, vArgs, vMxI, vMxH, S("bit32"), vMxH, vV, vMxH, S("bit32"), S("bit32"), vMxH, S("bit32"), vMxH, vMxH))
	}
	
	// Seed PRNG initial state using environment-derived runtime keys
	lines = append(lines, fmt.Sprintf("local %s,%s=%s(%s.bxor(%s, %d),%s),%s(%s.bxor(%s, %d),%s)",
		vXs0, vXs1, vMxName, S("bit32"), vEnvHash, expectedEnvHash ^ 0xA5, strings.Join(vSv, ","),
		vMxName, S("bit32"), vEnvHash, expectedEnvHash ^ 0x5C, strings.Join(reversedSeedsString(vSv), ",")))
	lines = append(lines, fmt.Sprintf("local %s,%s=%s.bxor(%s,%s)%%256,%s.max(1,(%s+%s)%%256)",
		vXs2, vXs3, S("bit32"), vXs0, vXs1, S("math"), vXs0, vXs1))
	vXnT := _n()
	vXnS := _n()
	// mix all 4 state words so sequential output isn't trivially invertible
	lines = append(lines, fmt.Sprintf("local function %s() local %s=%s.bxor(%s,%s.lshift(%s,11)) local %s=%s %s=%s %s=%s %s=%s %s=%s.bxor(%s.bxor(%s,%s.rshift(%s,19)),%s.bxor(%s,%s.rshift(%s,8))) %s=%s return %s.bxor(%s,%s.bxor(%s,%s.bxor(%s,%s)))%%256 end",
		vXn, vXnT, S("bit32"), vXs3, S("bit32"), vXs3, vXnS, vXs0, vXs3, vXs2, vXs2, vXs1, vXs1, vXs0, vXnS, S("bit32"), S("bit32"), vXnS, S("bit32"), vXnS, S("bit32"), vXnT, S("bit32"), vXnT, vXs0, vXnS,
		S("bit32"), vXnS, S("bit32"), vXs1, S("bit32"), vXs2, vXs3))
	lines = append(lines, fmt.Sprintf("local %s={} local %s=nil", vBuf, vChArr))
	lines = append(lines, fmt.Sprintf("local %s=%d", vSt, stateIds[0]*stateStep+stateKey))
	lines = append(lines, fmt.Sprintf("local %s={}", vSmD))
	lines = append(lines, fmt.Sprintf("local %s=%s({}, {__call=function() if _G[%q] then return end %s(%s,%s)() end})",
		vSm, S("setmetatable"), vKillFlag, S("rawget"), vSmD, vSt))

	rand.Shuffle(len(states), func(i, j int) {
		states[i], states[j] = states[j], states[i]
	})
	for _, s := range states {
		num, _ := strconv.Atoi(s[0])
		obfStateId := num*stateStep + stateKey
		lines = append(lines, fmt.Sprintf("%s(%s,%d,function() %s end)",
			S("rawset"), vSmD, obfStateId, s[1]))
	}
	lines = append(lines, fmt.Sprintf("repeat %s() until (((%s-%d)/%d==%d and (_G[%q] or 0)==%d) or _G[%q])",
		vSm, vSt, stateKey, stateStep, stDone, vKillFlag+"_trace", expectedTraceHash, vKillFlag))
	if cfg.UseDoubleVM {
		lines = append(lines, fmt.Sprintf("if %s and not _G[%q] then return %s()(%s or %s, %s.char, %s.bxor, %s.concat, %s) end",
			vChArr, vKillFlag, vChArr, S("loadstring"), S("load"), S("string"), S("bit32"), S("table"), S("unpack")))
	} else {
		lines = append(lines, fmt.Sprintf("if %s and not _G[%q] then return %s() end",
			vChArr, vKillFlag, vChArr))
	}
	var outputCode string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			outputCode += l + " "
		}
	}
	outputCode = regexp.MustCompile(`\s+`).ReplaceAllString(outputCode, " ")
	return outputCode
}
func reversedSeedsString(s []string) []string {
	var r []string
	for i := len(s) - 1; i >= 0; i-- {
		r = append(r, s[i])
	}
	return r
}
type mainConfigStruct struct {
	site string
}
var mainConfig = mainConfigStruct{
	site: "esoteric.win",
}
func Obfuscate(code string, antiTamper string, cfg Opts) (string, error) {
	cfg.AntiTamper = antiTamper
	return wrapWithIR(code, antiTamper, cfg), nil
}