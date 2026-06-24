package tui
import (
	"encoding/json"
	"fmt"
	"net/http"
	"skyvern/internal/ai"
	"skyvern/internal/config"
	"skyvern/internal/lavalink"
	"skyvern/internal/storage"
	"strconv"
	"strings"
	"time"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)
type formField struct {
	placeholder string
	value       string
	mask        bool
}
func createFormInputs(fields []formField, accent lipgloss.Color, lockFirst bool, lockedPlaceholder string) ([]textinput.Model, int) {
	inputs := make([]textinput.Model, len(fields))
	inputStyle := lipgloss.NewStyle().Foreground(accent)
	for i, f := range fields {
		ti := textinput.New()
		ti.Placeholder = f.placeholder
		ti.SetValue(f.value)
		ti.Prompt = " > "
		ti.TextStyle = inputStyle
		ti.Width = 45
		if f.mask {
			ti.EchoMode = textinput.EchoPassword
			ti.EchoCharacter = '•'
		}
		if i == 0 && lockFirst {
			if lockedPlaceholder != "" {
				ti.Placeholder = lockedPlaceholder
			} else {
				ti.Placeholder = f.placeholder + " (Locked)"
			}
			ti.Blur()
		}
		inputs[i] = ti
	}
	return inputs, 0
}
func (m *Model) initGlobalInputs() {
	fields := []formField{
		{"Global Name", "", false},
		{"Global Prefix", "", false},
		{"Global Footer", "", false},
		{"Global Embed Color (Hex)", "", false},
		{"Matrix Color (rgb/preset/hex)", "", false},
		{"Storage Location (local/portable/appdata)", "", false},
		{"Spotify Enabled (yes/no)", "", false},
		{"Logo URL", "", false},
		{"Always on Top (yes/no)", "", false},
		{"Show Logo (yes/no)", "", false},
		{"Auto Start Lavalink (yes/no)", "", false},
		{"Lavalink Host (e.g. localhost:2333)", "", false},
		{"Lavalink Password", "", true},
		{"Home Emoji Server ID", "", false},
		{"Bot Owner IDs (comma-separated)", "", false},
		{"AI Bots Enabled (yes/no)", "", false},
		{"AI Disabled Reason", "", false},
		{"AI Memory Enabled (yes/no)", "", false},
		{"AI Tokens Required (yes/no)", "", false},
		{"AI Owners Bypass Tokens (yes/no)", "", false},
		{"Commands Enabled (yes/no)", "", false},
	}
	g := config.GetGlobal()
	fields[0].value = g.Name
	fields[1].value = g.Prefix
	fields[2].value = g.Footer
	fields[3].value = fmt.Sprintf("%06x", g.EmbedColor)
	fields[4].value = g.MatrixColor
	fields[5].value = string(config.GetTuiCfg().Loc)
	fields[6].value = g.Spotify
	fields[7].value = g.FooterIcon
	if g.AlwaysOnTop {
		fields[8].value = "yes"
	} else {
		fields[8].value = "no"
	}
	if g.ShowLogo {
		fields[9].value = "yes"
	} else {
		fields[9].value = "no"
	}
	if g.AutoStartLavalink {
		fields[10].value = "yes"
	} else {
		fields[10].value = "no"
	}
	fields[11].value = g.LavalinkHost
	fields[12].value = g.LavalinkPass
	fields[13].value = g.EmojiServerID
	if g.OwnerIDs != "" {
		fields[14].value = g.OwnerIDs
	} else {
		fields[14].value = config.DefGlobal().OwnerIDs
	}
	if g.AIEnabled {
		fields[15].value = "yes"
	} else {
		fields[15].value = "no"
	}
	fields[16].value = g.AIDisableReason
	if g.AIMemory {
		fields[17].value = "yes"
	} else {
		fields[17].value = "no"
	}
	if g.AITokenReq {
		fields[18].value = "yes"
	} else {
		fields[18].value = "no"
	}
	if g.AIOwnerBypass {
		fields[19].value = "yes"
	} else {
		fields[19].value = "no"
	}
	if g.CommandsOn {
		fields[20].value = "yes"
	} else {
		fields[20].value = "no"
	}
	th := Themes[curTheme]
	m.inputs, m.focus = createFormInputs(fields, th.Accent, false, "")
	m.inputs[0].Focus()
}
func (m *Model) initPalantirInputs() {
	fields := []formField{
		{"Palantir Enabled (yes/no)", "", false},
		{"Blocked Servers", "", false},
		{"Blocked Channels", "", false},
		{"Blocked Users", "", false},
		{"Blocked Events", "", false},
	}
	pCfg, _ := m.mgr.GetPalantirCfg()
	if pCfg.Enabled {
		fields[0].value = "yes"
	} else {
		fields[0].value = "no"
	}
	fields[1].value = strings.Join(pCfg.BlockedGuilds, ", ")
	fields[2].value = strings.Join(pCfg.BlockedChannels, ", ")
	fields[3].value = strings.Join(pCfg.BlockedUsers, ", ")
	fields[4].value = strings.Join(pCfg.BlockedEvents, ", ")
	th := Themes[curTheme]
	m.inputs, m.focus = createFormInputs(fields, th.Accent, false, "")
	m.inputs[0].Focus()
}
func (m *Model) initInputs(b *config.BotInst) {
	fields := []formField{
		{"Client ID", "", false},
		{"Bot Token", "", true},
		{"Prefix (e.g. .)", "", false},
		{"Custom Name Override", "", false},
		{"Custom Footer Override", "", false},
		{"Custom Color (Hex e.g. 1a1a1a)", "", false},
		{"Avatar URL", "", false},
		{"Status (online/dnd/idle/invisible)", "", false},
		{"Presences (comma-separated)", "", false},
	}
	if b != nil {
		fields[0].value = b.ClientID
		fields[1].value = b.Token
		fields[2].value = b.Prefix
		fields[3].value = b.CustomName
		fields[4].value = b.CustomFooter
		if b.CustomColor != 0 {
			fields[5].value = fmt.Sprintf("%06x", b.CustomColor)
		}
		fields[6].value = b.AvatarURL
		fields[7].value = b.Status
		fields[8].value = strings.Join(b.Presences, ",")
	}
	th := Themes[curTheme]
	m.inputs, m.focus = createFormInputs(fields, th.Accent, b != nil, "Client ID (Locked)")
	if b != nil {
		m.focus = 1
		m.inputs[1].Focus()
	} else {
		m.inputs[0].Focus()
	}
}
func (m *Model) initAIInputs(p *storage.AIProvider) {
	fields := []formField{
		{"Provider ID", "", false},
		{"Provider Type", "", false},
		{"Friendly Name", "", false},
		{"API Key", "", true},
		{"Base URL Override", "", false},
		{"Default Model", "", false},
		{"Fallback ID (ID or 'random')", "", false},
		{"Max Tokens Limit (0 = unlimited)", "", false},
		{"Max Requests Limit (0 = unlimited)", "", false},
	}
	if p != nil {
		fields[0].value = p.ID
		fields[1].value = p.Type
		fields[2].value = p.Name
		fields[3].value = p.APIKey
		fields[4].value = p.BaseURL
		fields[5].value = p.DefaultModel
		fields[6].value = p.FallbackID
		fields[7].value = strconv.FormatInt(p.MaxTokens, 10)
		fields[8].value = strconv.FormatInt(p.MaxRequests, 10)
	} else {
		fields[1].value = "openai"
		fields[2].value = "New OpenAI"
		fields[5].value = "gpt-4o-mini"
		fields[7].value = "0"
		fields[8].value = "0"
	}
	th := Themes[curTheme]
	m.inputs, m.focus = createFormInputs(fields, th.Accent, p != nil, "Provider ID (Locked)")
	if p != nil {
		m.focus = 1
		m.inputs[1].Focus()
	} else {
		m.inputs[0].Focus()
	}
}
func (m *Model) focusInput(idx int) {
	if idx < 0 {
		idx = len(m.inputs) - 1
	}
	if idx >= len(m.inputs) {
		idx = 0
	}
	if idx == 0 && len(m.inputs) > 0 && strings.Contains(m.inputs[0].Placeholder, "(Locked)") {
		if m.focus == 1 {
			idx = len(m.inputs) - 1
		} else {
			idx = 1
		}
	}
	m.inputs[m.focus].Blur()
	m.focus = idx
	m.inputs[m.focus].Focus()
}
func (m *Model) saveGlobalSettings() {
	name := strings.TrimSpace(m.inputs[0].Value())
	if name == "" {
		name = config.DefaultName
	}
	prefix := strings.TrimSpace(m.inputs[1].Value())
	if prefix == "" {
		prefix = config.DefaultPrefix
	}
	footer := strings.TrimSpace(m.inputs[2].Value())
	if footer == "" {
		footer = config.DefaultFooter
	}
	colStr := strings.TrimPrefix(m.inputs[3].Value(), "#")
	colVal, err := strconv.ParseInt(colStr, 16, 32)
	if err != nil {
		colVal = config.ColorDefault
	}
	matrixCol := strings.TrimSpace(m.inputs[4].Value())
	if matrixCol == "" {
		matrixCol = "rgb"
	}
	spotifyVal := strings.TrimSpace(m.inputs[6].Value())
	if spotifyVal == "" {
		spotifyVal = "no"
	}
	logoVal := strings.TrimSpace(m.inputs[7].Value())
	topVal := strings.TrimSpace(strings.ToLower(m.inputs[8].Value()))
	alwaysTop := topVal == "yes" || topVal == "true" || topVal == "1"
	showLogoVal := strings.TrimSpace(strings.ToLower(m.inputs[9].Value()))
	showLogo := showLogoVal == "yes" || showLogoVal == "true" || showLogoVal == "1" || showLogoVal == ""
	lavVal := strings.TrimSpace(strings.ToLower(m.inputs[10].Value()))
	autoLavalink := lavVal == "yes" || lavVal == "true" || lavVal == "1" || lavVal == ""
	lavHost := strings.TrimSpace(m.inputs[11].Value())
	if lavHost == "" {
		lavHost = "localhost:2333"
	}
	lavPass := strings.TrimSpace(m.inputs[12].Value())
	if lavPass == "" {
		lavPass = "youshallnotpass"
	}
	emojiServerID := strings.TrimSpace(m.inputs[13].Value())
	if emojiServerID == "" {
		emojiServerID = "1411452931915645032"
	}
	ownerIDs := strings.TrimSpace(m.inputs[14].Value())
	aiVal := strings.TrimSpace(strings.ToLower(m.inputs[15].Value()))
	aiEnabled := aiVal == "yes" || aiVal == "true" || aiVal == "1" || aiVal == ""
	aiDisableReason := strings.TrimSpace(m.inputs[16].Value())
	aiMemVal := strings.TrimSpace(strings.ToLower(m.inputs[17].Value()))
	aiMemory := aiMemVal == "yes" || aiMemVal == "true" || aiMemVal == "1" || aiMemVal == ""
	aiTokVal := strings.TrimSpace(strings.ToLower(m.inputs[18].Value()))
	aiTokenReq := aiTokVal == "yes" || aiTokVal == "true" || aiTokVal == "1"
	aiBypassVal := strings.TrimSpace(strings.ToLower(m.inputs[19].Value()))
	aiOwnerBypass := aiBypassVal == "yes" || aiBypassVal == "true" || aiBypassVal == "1" || aiBypassVal == ""
	cmdVal := strings.TrimSpace(strings.ToLower(m.inputs[20].Value()))
	commandsOn := cmdVal == "yes" || cmdVal == "true" || cmdVal == "1" || cmdVal == ""
	storeLoc := config.StorageLoc(strings.TrimSpace(strings.ToLower(m.inputs[5].Value())))
	if storeLoc != config.LocPortable && storeLoc != config.LocAppData {
		storeLoc = config.LocLocal
	}
	_ = config.SaveTuiCfg(config.TuiCfg{Loc: storeLoc})
	g := config.GlobalCfg{
		Name:              name,
		Prefix:            prefix,
		Footer:            footer,
		EmbedColor:        int(colVal),
		MatrixColor:       matrixCol,
		TuiTheme:          curTheme,
		Spotify:           spotifyVal,
		FooterIcon:        logoVal,
		AlwaysOnTop:       alwaysTop,
		ShowLogo:          showLogo,
		AutoStartLavalink: autoLavalink,
		LavalinkHost:      lavHost,
		LavalinkPass:      lavPass,
		EmojiServerID:     emojiServerID,
		OwnerIDs:          ownerIDs,
		AIEnabled:         aiEnabled,
		AIDisableReason:   aiDisableReason,
		AIMemory:          aiMemory,
		AITokenReq:        aiTokenReq,
		AIOwnerBypass:     aiOwnerBypass,
		CommandsOn:        commandsOn,
	}
	_ = m.db.SaveGlobal(g)
	config.SetGlobal(g)
	SetAlwaysOnTop(alwaysTop)
	isLocal := strings.Contains(lavHost, "localhost") || strings.Contains(lavHost, "127.0.0.1")
	if autoLavalink && isLocal {
		lavalink.StartServer(config.ResolvePath)
	} else {
		lavalink.StopServer()
	}
	for _, b := range m.bots {
		_ = m.mgr.UpdateInstance(b.ClientID)
	}
	m.reload()
}
func (m *Model) savePalantirSettings() {
	pEnabled := strings.TrimSpace(strings.ToLower(m.inputs[0].Value())) == "yes" || strings.TrimSpace(strings.ToLower(m.inputs[0].Value())) == "true"
	splitTrim := func(s string) []string {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		parts := strings.Split(s, ",")
		var out []string
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	}
	pCfg := storage.PalantirCfg{
		Enabled:         pEnabled,
		BlockedGuilds:   splitTrim(m.inputs[1].Value()),
		BlockedChannels: splitTrim(m.inputs[2].Value()),
		BlockedUsers:    splitTrim(m.inputs[3].Value()),
		BlockedEvents:   splitTrim(m.inputs[4].Value()),
	}
	_ = m.mgr.SavePalantirCfg(pCfg)
}
func (m *Model) saveEdit() {
	cid := strings.TrimSpace(m.inputs[0].Value())
	if cid == "" && len(m.bots) > 0 && m.inputs[0].Placeholder == "Client ID (Locked)" {
		cid = m.bots[m.selIdx].ClientID
	}
	token := strings.TrimSpace(m.inputs[1].Value())
	customName := strings.TrimSpace(m.inputs[3].Value())
	avatarURL := strings.TrimSpace(m.inputs[6].Value())
	if cid == "" && token != "" {
		req, err := http.NewRequest("GET", "https://discord.com/api/v10/users/@me", nil)
		if err == nil {
			req.Header.Set("Authorization", "Bot "+token)
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == 200 {
					var u struct {
						ID       string `json:"id"`
						Username string `json:"username"`
						Avatar   string `json:"avatar"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&u); err == nil {
						cid = u.ID
						if customName == "" {
							customName = u.Username
						}
						if avatarURL == "" && u.Avatar != "" {
							avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", u.ID, u.Avatar)
						}
					}
				}
			}
		}
	}
	if cid == "" {
		return
	}
	colStr := strings.TrimPrefix(m.inputs[5].Value(), "#")
	colVal, _ := strconv.ParseInt(colStr, 16, 32)
	statusVal := strings.TrimSpace(m.inputs[7].Value())
	pres := strings.TrimSpace(m.inputs[8].Value())
	var presences []string
	if pres != "" {
		for _, p := range strings.Split(pres, ",") {
			pt := strings.TrimSpace(p)
			if pt != "" {
				presences = append(presences, pt)
			}
		}
	}
	b := config.BotInst{
		ClientID:     cid,
		Token:        token,
		Prefix:       strings.TrimSpace(m.inputs[2].Value()),
		CustomName:   customName,
		CustomFooter: strings.TrimSpace(m.inputs[4].Value()),
		CustomColor:  int(colVal),
		AvatarURL:    avatarURL,
		Status:       statusVal,
		Presences:    presences,
	}
	_ = m.db.SaveBot(b)
	_ = m.mgr.UpdateInstance(b.ClientID)
	m.reload()
}
func (m *Model) initPromptInputs() {
	pCfg, _ := ai.LoadPrompts()
	fields := []formField{
		{"Default System Prompt", pCfg.SystemPrompt, false},
	}
	th := Themes[curTheme]
	m.inputs, m.focus = createFormInputs(fields, th.Accent, false, "")
	m.inputs[0].Focus()
}
func (m *Model) savePromptSettings() {
	val := m.inputs[0].Value()
	_ = ai.SavePrompts(ai.PromptsConfig{SystemPrompt: val})
}