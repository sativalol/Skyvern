package manager
import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"github.com/bwmarrin/discordgo"
	bolt "go.etcd.io/bbolt"
	"skyvern/internal/config"
	"skyvern/internal/lavalink"
	"skyvern/internal/storage"
	"skyvern/internal/updater"
)
type instState struct {
	cfg      config.ResCfg
	session  *discordgo.Session
	clientID string
	running  bool
	lavalink *lavalink.Client
	lastErr  string
}
type tracker struct {
	mu          sync.RWMutex
	data        map[string]*storage.Analytics
	lastFlushed map[string]storage.Analytics
}
func (t *tracker) get(id string) storage.Analytics {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if a, ok := t.data[id]; ok {
		return *a
	}
	return storage.Analytics{}
}
func (t *tracker) incPrefix(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	a := t.ensure(id)
	a.PrefixCmds++
	a.TotalCmds++
}
func (t *tracker) incSlash(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	a := t.ensure(id)
	a.SlashCmds++
	a.TotalCmds++
}
func (t *tracker) setGuilds(id string, n int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ensure(id).GuildCount = n
}
func (t *tracker) totals() storage.Analytics {
	t.mu.RLock()
	defer t.mu.RUnlock()
	var out storage.Analytics
	for _, a := range t.data {
		out.TotalCmds += a.TotalCmds
		out.PrefixCmds += a.PrefixCmds
		out.SlashCmds += a.SlashCmds
		out.GuildCount += a.GuildCount
	}
	return out
}
func (t *tracker) ensure(id string) *storage.Analytics {
	if t.data == nil {
		t.data = make(map[string]*storage.Analytics)
	}
	if t.data[id] == nil {
		t.data[id] = &storage.Analytics{}
	}
	return t.data[id]
}
type Manager struct {
	db       *storage.DB
	commands []*Command
	mu        sync.RWMutex
	instances map[string]*instState
	stats     *tracker
	stopFlush chan struct{}
	compHandlers map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate)
	lastDailyQuestionDate map[string]string
	lastDailyQuoteDate    map[string]string
	dailyMu               sync.Mutex
	lastBoostLogged map[string]time.Time
	boostMu         sync.Mutex
	reminders   []storage.Reminder
	schedules   []storage.ScheduledMsg
	remindersMu sync.RWMutex
	schedulesMu sync.RWMutex
	antispamTracker map[string][]time.Time
	antispamMu      sync.Mutex
	palantirDB        *bolt.DB
	antispamCache     map[string]storage.AntispamCfg
	filterCache       map[string]storage.FilterCfg
	antilinkCache     map[string]storage.AntilinkCfg
	prefixCache       map[string]string
	palantirCache     storage.PalantirCfg
	palantirCacheInit bool
	regexCache        map[string][]*regexp.Regexp
	antinukeCache     map[string]storage.AntinukeCfg
	antiraidCache     map[string]storage.AntiraidCfg
	configMu          sync.RWMutex
	palantirChan chan *PalantirLog
	palantirWG   sync.WaitGroup
	whs  map[string]*discordgo.Webhook
	whMu sync.RWMutex
	xpCooldowns   map[string]time.Time
	xpCooldownsMu sync.Mutex
	joinHandlers []func(s *discordgo.Session, e *discordgo.GuildMemberAdd)
	BootTime     time.Time
}
func New(db *storage.DB, cmds []*Command) *Manager {
	rems, _ := db.ListAllReminders()
	schs, _ := db.ListAllSchedules()
	m := &Manager{
		db:                    db,
		commands:              cmds,
		instances:             make(map[string]*instState),
		stats:                 &tracker{},
		stopFlush:             make(chan struct{}),
		compHandlers:          make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate)),
		lastDailyQuestionDate: make(map[string]string),
		lastDailyQuoteDate:    make(map[string]string),
		lastBoostLogged:       make(map[string]time.Time),
		antispamTracker:       make(map[string][]time.Time),
		antispamCache:         make(map[string]storage.AntispamCfg),
		filterCache:           make(map[string]storage.FilterCfg),
		antilinkCache:         make(map[string]storage.AntilinkCfg),
		prefixCache:           make(map[string]string),
		regexCache:            make(map[string][]*regexp.Regexp),
		antinukeCache:         make(map[string]storage.AntinukeCfg),
		antiraidCache:         make(map[string]storage.AntiraidCfg),
		palantirChan:          make(chan *PalantirLog, 1000),
		reminders:             rems,
		schedules:             schs,
		whs:                   make(map[string]*discordgo.Webhook),
		xpCooldowns:           make(map[string]time.Time),
		BootTime:              time.Now(),
	}
	palDb, err := bolt.Open(config.ResolvePath("palantir.db"), 0600, nil)
	if err == nil {
		_ = palDb.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists([]byte("AuditLogs"))
			return err
		})
		m.palantirDB = palDb
	}
	go m.palantirWriterLoop()
	go m.flushLoop()
	go m.tempRoleLoop()
	go m.dailySchedulerLoop()
	go m.birthdayLoop()
	go m.remindLoop()
	go m.scheduleLoop()
	go m.giveawayLoop()
	go m.gcLoop()
	return m
}
func (m *Manager) DB() *storage.DB {
	return m.db
}
func (m *Manager) PalantirDB() *bolt.DB {
	return m.palantirDB
}
func (m *Manager) RegisterComponentHandler(customID string, fn func(s *discordgo.Session, i *discordgo.InteractionCreate)) {
	m.mu.Lock()
	if m.compHandlers == nil {
		m.compHandlers = make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate))
	}
	m.compHandlers[customID] = fn
	m.mu.Unlock()
}
func (m *Manager) RegisterJoinHandler(fn func(s *discordgo.Session, e *discordgo.GuildMemberAdd)) {
	m.mu.Lock()
	m.joinHandlers = append(m.joinHandlers, fn)
	m.mu.Unlock()
}
func (m *Manager) Close() {
	close(m.stopFlush)
	if m.palantirChan != nil {
		close(m.palantirChan)
	}
	m.palantirWG.Wait()
	if m.palantirDB != nil {
		_ = m.palantirDB.Close()
	}
}
func (m *Manager) AddCommands(cmds []*Command) {
	m.mu.Lock()
	m.commands = append(m.commands, cmds...)
	m.mu.Unlock()
}
func (m *Manager) Start(clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.instances[clientID]; ok && s.running {
		return nil
	}
	inst, err := m.db.GetBot(clientID)
	if err != nil {
		return fmt.Errorf("load bot: %w", err)
	}
	cfg := config.Resolve(config.GetGlobal(), inst)
	sess, err := discordgo.New("Bot " + inst.Token)
	if err != nil {
		return fmt.Errorf("discordgo init: %w", err)
	}
	sess.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuilds | discordgo.IntentMessageContent | discordgo.IntentsGuildMembers | discordgo.IntentsGuildMessageReactions | discordgo.IntentsGuildVoiceStates
	sess.State.MaxMessageCount = 1000
	host := config.GetGlobal().LavalinkHost
	if host == "" {
		host = os.Getenv("LAVALINK_HOST")
		if host == "" {
			host = "localhost:2333"
		}
	}
	pass := config.GetGlobal().LavalinkPass
	if pass == "" {
		pass = os.Getenv("LAVALINK_PASS")
		if pass == "" {
			pass = "youshallnotpass"
		}
	}
	state := &instState{
		cfg:      cfg,
		session:  sess,
		clientID: clientID,
		lavalink: lavalink.NewClient(host, pass, clientID, sess),
	}
	m.instances[clientID] = state
	m.attachHandlers(sess, state)
	if gateway, err := sess.GatewayBot(); err == nil && gateway.Shards > 1 {
		sess.ShardCount = gateway.Shards
	}
	if err := sess.Open(); err != nil {
		eStr := err.Error()
		eLow := strings.ToLower(eStr)
		if strings.Contains(eLow, "401") || strings.Contains(eLow, "unauthorized") || strings.Contains(eLow, "4004") || strings.Contains(eLow, "token") {
			state.lastErr = "Token invalid/expired. Update it in settings."
		} else {
			state.lastErr = eStr
		}
		state.running = false
		return fmt.Errorf("session open: %w", err)
	}
	state.running = true
	if cfg.AvatarURL != "" {
		go m.updateAvatar(sess, cfg.AvatarURL, clientID)
	}
	if len(inst.Presences) > 0 {
		go m.presenceLoop(sess, clientID, inst.Presences)
	}
	go m.backfillBotMeta(sess, clientID, inst)
	m.stats.setGuilds(clientID, len(sess.State.Guilds))
	go m.UploadVoiceMasterEmojis(sess)
	go m.UploadSystemEmojis(sess)
	go m.registerSlashCommands(sess)
	go m.notifyOwnerOfUpdate(sess)
	return nil
}
func (m *Manager) presenceLoop(sess *discordgo.Session, clientID string, presences []string) {
	idx := 0
	for {
		if !m.IsRunning(clientID) {
			break
		}
		text := presences[idx%len(presences)]
		guilds := len(sess.State.Guilds)
		users := 0
		for _, g := range sess.State.Guilds {
			users += g.MemberCount
		}
		ping := sess.HeartbeatLatency().String()
		text = strings.ReplaceAll(text, "{guilds}", fmt.Sprintf("%d", guilds))
		text = strings.ReplaceAll(text, "{users}", fmt.Sprintf("%d", users))
		text = strings.ReplaceAll(text, "{ping}", ping)

		st := discordgo.StatusOnline
		if inst, err := m.db.GetBot(clientID); err == nil && inst.Status != "" {
			switch strings.ToLower(strings.TrimSpace(inst.Status)) {
			case "dnd", "do_not_disturb", "donotdisturb":
				st = discordgo.StatusDoNotDisturb
			case "idle":
				st = discordgo.StatusIdle
			case "invisible", "offline":
				st = discordgo.StatusInvisible
			}
		}

		_ = sess.UpdateStatusComplex(discordgo.UpdateStatusData{
			Status: string(st),
			Activities: []*discordgo.Activity{
				{
					Name: text,
					Type: discordgo.ActivityTypeGame,
				},
			},
		})
		idx++
		time.Sleep(15 * time.Second)
	}
}
func (m *Manager) UploadVoiceMasterEmojis(s *discordgo.Session) {
	emojiServerID := config.GetGlobal().EmojiServerID
	if emojiServerID == "" {
		emojiServerID = "1411452931915645032"
	}
	_, err := s.Guild(emojiServerID)
	if err != nil {
		log.Printf("[EmojiLoader] Bot is not in the home emoji server %s: %v", emojiServerID, err)
		return
	}
	emojis, err := s.GuildEmojis(emojiServerID)
	if err != nil {
		log.Printf("[EmojiLoader] Failed to fetch guild emojis from %s: %v", emojiServerID, err)
		return
	}
	emojiMap := make(map[string]bool)
	for _, e := range emojis {
		emojiMap[e.Name] = true
	}
	dir := "internal/assets/vc-icons"
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("[EmojiLoader] Failed to read vc-icons directory: %v", err)
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".png") {
			continue
		}
		baseName := strings.TrimSuffix(entry.Name(), ".png")
		emojiName := "vm_" + strings.ReplaceAll(baseName, "-", "_")
		if emojiMap[emojiName] {
			continue
		}
		filePath := fmt.Sprintf("%s/%s", dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("[EmojiLoader] Failed to read emoji file %s: %v", filePath, err)
			continue
		}
		contentType := http.DetectContentType(data)
		b64 := base64.StdEncoding.EncodeToString(data)
		imageURI := fmt.Sprintf("data:%s;base64,%s", contentType, b64)
		_, err = s.GuildEmojiCreate(emojiServerID, &discordgo.EmojiParams{
			Name:  emojiName,
			Image: imageURI,
		})
		if err != nil {
			log.Printf("[EmojiLoader] Failed to upload emoji %s to guild %s: %v", emojiName, emojiServerID, err)
		} else {
			log.Printf("[EmojiLoader] Successfully uploaded emoji %s to guild %s", emojiName, emojiServerID)
		}
	}
}
func (m *Manager) UploadSystemEmojis(s *discordgo.Session) {
	emojiServerID := config.GetGlobal().EmojiServerID
	if emojiServerID == "" {
		emojiServerID = "1411452931915645032"
	}
	_, err := s.Guild(emojiServerID)
	if err != nil {
		log.Printf("[SystemEmojiLoader] Bot is not in the home emoji server %s: %v", emojiServerID, err)
		return
	}
	emojis, err := s.GuildEmojis(emojiServerID)
	if err != nil {
		log.Printf("[SystemEmojiLoader] Failed to fetch guild emojis from %s: %v", emojiServerID, err)
		return
	}
	emojiMap := make(map[string]bool)
	for _, e := range emojis {
		emojiMap[e.Name] = true
	}
	dir := "internal/assets/custom-icons"
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("[SystemEmojiLoader] Failed to read custom-icons directory: %v", err)
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".png") {
			continue
		}
		baseName := strings.TrimSuffix(entry.Name(), ".png")
		emojiName := "sys_" + strings.ReplaceAll(baseName, "-", "_")
		if emojiMap[emojiName] {
			continue
		}
		filePath := fmt.Sprintf("%s/%s", dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("[SystemEmojiLoader] Failed to read emoji file %s: %v", filePath, err)
			continue
		}
		contentType := http.DetectContentType(data)
		b64 := base64.StdEncoding.EncodeToString(data)
		imageURI := fmt.Sprintf("data:%s;base64,%s", contentType, b64)
		_, err = s.GuildEmojiCreate(emojiServerID, &discordgo.EmojiParams{
			Name:  emojiName,
			Image: imageURI,
		})
		if err != nil {
			log.Printf("[SystemEmojiLoader] Failed to upload emoji %s to guild %s: %v", emojiName, emojiServerID, err)
		} else {
			log.Printf("[SystemEmojiLoader] Successfully uploaded emoji %s to guild %s", emojiName, emojiServerID)
		}
	}
}
func (m *Manager) ResolveEmoji(s *discordgo.Session, gid string, name string) string {
	normalizedName := strings.ReplaceAll(name, "-", "_")
	emojiServerID := config.GetGlobal().EmojiServerID
	if emojiServerID == "" {
		emojiServerID = "1411452931915645032"
	}
	if ems, err := s.GuildEmojis(emojiServerID); err == nil {
		for _, e := range ems {
			if e.Name == normalizedName {
				return fmt.Sprintf("<:%s:%s>", e.Name, e.ID)
			}
		}
	}
	if gid != "" {
		if ems, err := s.GuildEmojis(gid); err == nil {
			for _, e := range ems {
				if e.Name == normalizedName {
					return fmt.Sprintf("<:%s:%s>", e.Name, e.ID)
				}
			}
		}
	}
	for _, gState := range s.State.Guilds {
		for _, e := range gState.Emojis {
			if e.Name == normalizedName {
				return fmt.Sprintf("<:%s:%s>", e.Name, e.ID)
			}
		}
	}
	switch name {
	case "sys_checkmark":
		return "[+]"
	case "sys_x":
		return "[!]"
	case "sys_warning":
		return "[!]"
	case "sys_lock":
		return "[🔒]"
	case "sys_unlock":
		return "[🔓]"
	default:
		return ""
	}
}
func (m *Manager) Stop(clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.instances[clientID]
	if !ok || !s.running {
		return nil
	}
	if s.lavalink != nil {
		s.lavalink.Close()
	}
	if err := s.session.Close(); err != nil {
		return fmt.Errorf("session close: %w", err)
	}
	s.running = false
	return nil
}
func (m *Manager) IsRunning(clientID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.instances[clientID]
	return ok && s.running
}
func (m *Manager) HasRunningBots() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, v := range m.instances {
		if v.running {
			return true
		}
	}
	return false
}
func (m *Manager) LastErr(clientID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.instances[clientID]; ok {
		return s.lastErr
	}
	return ""
}
func (m *Manager) backfillBotMeta(sess *discordgo.Session, clientID string, inst config.BotInst) {
	time.Sleep(2 * time.Second)
	if sess.State == nil || sess.State.User == nil {
		return
	}
	u := sess.State.User
	changed := false
	if inst.CustomName == "" && u.Username != "" {
		inst.CustomName = u.Username
		changed = true
	}
	if inst.AvatarURL == "" && u.Avatar != "" {
		inst.AvatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png?size=256", u.ID, u.Avatar)
		changed = true
	}
	if inst.ClientID == "" && u.ID != "" {
		inst.ClientID = u.ID
		changed = true
	}
	if !changed {
		return
	}
	_ = m.db.SaveBot(inst)
	if clientID != inst.ClientID {
		_ = m.db.DeleteBot(clientID)
	}
	m.mu.Lock()
	if state, ok := m.instances[clientID]; ok {
		state.cfg = config.Resolve(config.GetGlobal(), inst)
		if clientID != inst.ClientID {
			state.clientID = inst.ClientID
			m.instances[inst.ClientID] = state
			delete(m.instances, clientID)
		}
	}
	m.mu.Unlock()
}
func (m *Manager) Stats(clientID string) storage.Analytics { return m.stats.get(clientID) }
func (m *Manager) GlobalStats() storage.Analytics          { return m.stats.totals() }
func (m *Manager) UpdateInstance(cid string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	inst, err := m.db.GetBot(cid)
	if err != nil {
		return err
	}
	cfg := config.Resolve(config.GetGlobal(), inst)
	if s, ok := m.instances[cid]; ok {
		s.cfg = cfg
	}
	return nil
}
func (m *Manager) Snapshot() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]bool)
	for cid, inst := range m.instances {
		out[cid] = inst.running
	}
	return out
}
func (m *Manager) ResolvedCfgFor(cid string) (config.ResCfg, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if inst, ok := m.instances[cid]; ok {
		return inst.cfg, true
	}
	return config.ResCfg{}, false
}
func (m *Manager) Commands() []*Command {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Command, len(m.commands))
	copy(out, m.commands)
	return out
}
func (m *Manager) updateAvatar(sess *discordgo.Session, url, clientID string) {
	resp, err := http.Get(url) // #nosec G107
	if err != nil {
		fmt.Printf("[%s] avatar: %v\n", clientID, err)
		return
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[%s] avatar read: %v\n", clientID, err)
		return
	}
	ct := http.DetectContentType(data)
	if _, err := sess.UserUpdate("", "data:"+ct+";base64,"+base64.StdEncoding.EncodeToString(data), ""); err != nil {
		fmt.Printf("[%s] avatar set: %v\n", clientID, err)
	}
}
func (m *Manager) findByTrigger(trigger string) *Command {
	for _, c := range m.commands {
		if strings.EqualFold(c.Trigger, trigger) {
			return c
		}
		for _, a := range c.Aliases {
			if strings.EqualFold(a, trigger) {
				return c
			}
		}
	}
	return nil
}
func (m *Manager) FindCommand(trigger string) *Command {
	return m.findByTrigger(trigger)
}
func (m *Manager) findByName(name string) *Command {
	for _, c := range m.commands {
		if strings.EqualFold(c.Name, name) {
			return c
		}
	}
	return nil
}
func renderTemplate(tmpl string, msg *discordgo.Message, args []string) string {
	var pairs []string
	if msg.Author != nil {
		pairs = append(pairs, "{user.name}", msg.Author.Username, "{user.mention}", "<@"+msg.Author.ID+">")
	}
	pairs = append(pairs, "{channel.mention}", "<#"+msg.ChannelID+">")
	if strings.Contains(tmpl, "{args}") {
		pairs = append(pairs, "{args}", strings.Join(args, " "))
	}
	return strings.NewReplacer(pairs...).Replace(tmpl)
}
func (m *Manager) FirstSession() *discordgo.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, inst := range m.instances {
		if inst.running && inst.session != nil {
			return inst.session
		}
	}
	return nil
}
func (m *Manager) ListReminders(uid string) []storage.Reminder {
	m.remindersMu.RLock()
	defer m.remindersMu.RUnlock()
	var out []storage.Reminder
	for _, r := range m.reminders {
		if r.UserID == uid {
			out = append(out, r)
		}
	}
	return out
}
func (m *Manager) SaveReminder(r storage.Reminder) error {
	if err := m.db.SaveReminder(r); err != nil {
		return err
	}
	m.remindersMu.Lock()
	defer m.remindersMu.Unlock()
	for i, x := range m.reminders {
		if x.UserID == r.UserID && x.ID == r.ID {
			m.reminders[i] = r
			return nil
		}
	}
	m.reminders = append(m.reminders, r)
	return nil
}
func (m *Manager) DeleteReminder(uid, id string) error {
	if err := m.db.DeleteReminder(uid, id); err != nil {
		return err
	}
	m.remindersMu.Lock()
	defer m.remindersMu.Unlock()
	for i, x := range m.reminders {
		if x.UserID == uid && x.ID == id {
			m.reminders = append(m.reminders[:i], m.reminders[i+1:]...)
			break
		}
	}
	return nil
}
func (m *Manager) ListSchedules(gid string) []storage.ScheduledMsg {
	m.schedulesMu.RLock()
	defer m.schedulesMu.RUnlock()
	var out []storage.ScheduledMsg
	for _, s := range m.schedules {
		if s.GuildID == gid {
			out = append(out, s)
		}
	}
	return out
}
func (m *Manager) SaveSchedule(s storage.ScheduledMsg) error {
	if err := m.db.SaveSchedule(s); err != nil {
		return err
	}
	m.schedulesMu.Lock()
	defer m.schedulesMu.Unlock()
	for i, x := range m.schedules {
		if x.GuildID == s.GuildID && x.ID == s.ID {
			m.schedules[i] = s
			return nil
		}
	}
	m.schedules = append(m.schedules, s)
	return nil
}
func (m *Manager) DeleteSchedule(gid, id string) error {
	if err := m.db.DeleteSchedule(gid, id); err != nil {
		return err
	}
	m.schedulesMu.Lock()
	defer m.schedulesMu.Unlock()
	for i, x := range m.schedules {
		if x.GuildID == gid && x.ID == id {
			m.schedules = append(m.schedules[:i], m.schedules[i+1:]...)
			break
		}
	}
	return nil
}
func (m *Manager) GetLavalink(clientID string) *lavalink.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.instances[clientID]; ok {
		return s.lavalink
	}
	return nil
}
func (m *Manager) registerSlashCommands(s *discordgo.Session) {
	cmds := []*discordgo.ApplicationCommand{
		{
			Name:        "quote",
			Description: "Create a styled quote embed",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "text",
					Description: "The text to quote",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to quote",
					Required:    false,
				},
			},
		},
		{
			Name:        "impersonate",
			Description: "Send a message as another user via webhook",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to impersonate",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "message",
					Description: "The message to send",
					Required:    true,
				},
			},
		},
		{
			Name:        "fakemessage",
			Description: "Generates an image of a fake message from the specified user",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to fake a message from",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "message",
					Description: "The message content to render",
					Required:    true,
				},
			},
		},
	}
	for _, cmd := range cmds {
		_, _ = s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
	}
}

func (m *Manager) notifyOwnerOfUpdate(sess *discordgo.Session) {
	time.Sleep(10 * time.Second)

	latest, update, err := updater.CheckVersion(config.Version)
	if err != nil || !update {
		return
	}

	g := config.GetGlobal()
	repl := strings.NewReplacer(";", ",", " ", ",", "\n", ",", "\r", ",", "\t", ",")
	ownerIDs := g.OwnerIDs
	if strings.TrimSpace(ownerIDs) == "" {
		ownerIDs = config.DefGlobal().OwnerIDs
	}

	msg := fmt.Sprintf("⚠️ **Skyvern Update Available**\n\nVersion **%s** is now available (You are currently running **%s**).\nDownload the latest release here: https://esoteric.win/skyvern/releases", latest, config.Version)

	for _, part := range strings.Split(repl.Replace(ownerIDs), ",") {
		uid := strings.TrimSpace(part)
		if uid == "" {
			continue
		}
		ch, err := sess.UserChannelCreate(uid)
		if err != nil {
			continue
		}
		_, _ = sess.ChannelMessageSend(ch.ID, msg)
	}
}
