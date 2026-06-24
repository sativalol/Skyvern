package lavalink
import (
	"fmt"
	"math/rand"
	"sync"
	"time"
	"github.com/bwmarrin/discordgo"
)
type Player struct {
	client  *Client
	guildID string
	chanID  string
	queue   []Track
	cur     int
	vol     int
	paused  bool
	loop    string
	preset  string
	mu      sync.Mutex
}
func (p *Player) Add(tracks []Track, textChan string) {
	p.mu.Lock()
	p.chanID = textChan
	start := len(p.queue) == 0
	p.queue = append(p.queue, tracks...)
	p.mu.Unlock()
	if start {
		p.PlayIndex(0)
	}
}
func (p *Player) PlayIndex(idx int) {
	p.mu.Lock()
	if idx < 0 || idx >= len(p.queue) {
		p.mu.Unlock()
		return
	}
	p.cur = idx
	p.paused = false
	t := p.queue[p.cur]
	p.mu.Unlock()
	payload := map[string]any{
		"encodedTrack": t.Encoded,
		"paused":       false,
	}
	_ = p.client.UpdatePlayer(p.guildID, payload)
}
func (p *Player) PlayNext() {
	p.mu.Lock()
	if len(p.queue) == 0 {
		p.mu.Unlock()
		return
	}
	next := p.cur
	switch p.loop {
	case "track":
	case "queue":
		next = (p.cur + 1) % len(p.queue)
	default:
		next = p.cur + 1
	}
	if next < 0 || next >= len(p.queue) {
		p.queue = nil
		p.cur = 0
		cid := p.chanID
		p.mu.Unlock()
		_ = p.client.DestroyPlayer(p.guildID)
		if cid != "" && p.client.Session != nil {
			_, _ = p.client.Session.ChannelMessageSend(cid, "Queue finished.")
		}
		return
	}
	p.cur = next
	p.mu.Unlock()
	p.PlayIndex(next)
}
func (p *Player) Skip() error {
	p.mu.Lock()
	if p.cur+1 >= len(p.queue) && p.loop != "queue" {
		p.mu.Unlock()
		return p.Stop()
	}
	p.mu.Unlock()
	p.PlayNext()
	return nil
}
func (p *Player) Stop() error {
	p.mu.Lock()
	p.queue = nil
	p.cur = 0
	p.mu.Unlock()
	_ = p.client.DestroyPlayer(p.guildID)
	if p.client.Session != nil {
		_ = SendVoiceStateUpdate(p.client.Session, p.guildID, "", false, false)
	}
	return nil
}
func (p *Player) Pause(paused bool) error {
	p.mu.Lock()
	p.paused = paused
	p.mu.Unlock()
	payload := map[string]any{
		"paused": paused,
	}
	return p.client.UpdatePlayer(p.guildID, payload)
}
func (p *Player) Volume(v int) error {
	if v < 0 {
		v = 0
	}
	if v > 150 {
		v = 150
	}
	p.mu.Lock()
	p.vol = v
	p.mu.Unlock()
	payload := map[string]any{
		"volume": v,
	}
	return p.client.UpdatePlayer(p.guildID, payload)
}
func (p *Player) Seek(ms int64) error {
	payload := map[string]any{
		"position": ms,
	}
	return p.client.UpdatePlayer(p.guildID, payload)
}
func (p *Player) Shuffle() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.queue) <= p.cur+1 {
		return
	}
	sub := p.queue[p.cur+1:]
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(sub), func(i, j int) {
		sub[i], sub[j] = sub[j], sub[i]
	})
}
func (p *Player) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.queue) == 0 {
		return
	}
	p.queue = []Track{p.queue[p.cur]}
	p.cur = 0
}
func (p *Player) SetLoop(mode string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.loop = mode
}
func (p *Player) NowPlaying() (Track, int64, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cur < 0 || p.cur >= len(p.queue) {
		return Track{}, 0, false
	}
	return p.queue[p.cur], int64(p.cur), true
}
func (p *Player) GetQueue() ([]Track, int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.queue, p.cur
}
func (p *Player) LoopMode() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.loop == "" {
		return "off"
	}
	return p.loop
}
func (p *Player) Paused() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.paused
}
func (p *Player) AnnounceStart() {
	p.mu.Lock()
	cid := p.chanID
	if cid == "" || p.client.Session == nil || p.cur < 0 || p.cur >= len(p.queue) {
		p.mu.Unlock()
		return
	}
	t := p.queue[p.cur]
	isPaused := p.paused
	loopMode := p.loop
	if loopMode == "" {
		loopMode = "off"
	}
	p.mu.Unlock()
	isPausedStr := "Playing"
	if isPaused {
		isPausedStr = "Paused"
	}
	reqMention := "Unknown"
	if t.Requester != "" {
		reqMention = fmt.Sprintf("<@%s>", t.Requester)
	}
	emb := &discordgo.MessageEmbed{
		Title: "Now Playing",
		Description: fmt.Sprintf("**[%s](%s)**\n\n**Status:** %s\n**Duration:** 0:00 / %s\n**Requested By:** %s\n**Loop:** %s",
			t.Info.Title, t.Info.URI, isPausedStr, formatDur(t.Info.Length), reqMention, loopMode),
		Color: 0x00ff00,
	}
	_, _ = p.client.Session.ChannelMessageSendEmbed(cid, emb)
}
func (p *Player) Vol() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.vol
}
func formatDur(ms int64) string {
	s := (ms / 1000) % 60
	m := (ms / (1000 * 60)) % 60
	h := (ms / (1000 * 60 * 60)) % 24
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
func (p *Player) RemovePosition(pos int) (Track, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	idx := p.cur + pos
	if idx <= p.cur || idx >= len(p.queue) {
		return Track{}, fmt.Errorf("invalid position")
	}
	t := p.queue[idx]
	p.queue = append(p.queue[:idx], p.queue[idx+1:]...)
	return t, nil
}
func (p *Player) MovePosition(from, to int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	fromIdx := p.cur + from
	toIdx := p.cur + to
	if fromIdx <= p.cur || fromIdx >= len(p.queue) || toIdx <= p.cur || toIdx >= len(p.queue) {
		return fmt.Errorf("invalid positions")
	}
	t := p.queue[fromIdx]
	p.queue = append(p.queue[:fromIdx], p.queue[fromIdx+1:]...)
	p.queue = append(p.queue[:toIdx], append([]Track{t}, p.queue[toIdx:]...)...)
	return nil
}
func (p *Player) AddNext(tracks []Track, textChan string) {
	p.mu.Lock()
	p.chanID = textChan
	start := len(p.queue) == 0
	if start {
		p.queue = append(p.queue, tracks...)
		p.mu.Unlock()
		p.PlayIndex(0)
		return
	}
	idx := p.cur + 1
	p.queue = append(p.queue[:idx], append(tracks, p.queue[idx:]...)...)
	p.mu.Unlock()
}
func (p *Player) SetPreset(name string, setting bool) error {
	p.mu.Lock()
	if !setting {
		p.preset = ""
	} else {
		p.preset = name
	}
	p.mu.Unlock()
	var filters map[string]any
	if setting {
		switch name {
		case "vaporwave":
			filters = map[string]any{
				"timescale": map[string]any{
					"speed": 0.85,
					"pitch": 0.80,
				},
			}
		case "nightcore":
			filters = map[string]any{
				"timescale": map[string]any{
					"speed": 1.25,
					"pitch": 1.25,
				},
			}
		case "chipmunk":
			filters = map[string]any{
				"timescale": map[string]any{
					"speed": 1.10,
					"pitch": 1.50,
				},
			}
		case "boost":
			filters = map[string]any{
				"equalizer": []map[string]any{
					{"band": 0, "gain": 0.25},
					{"band": 1, "gain": 0.15},
					{"band": 2, "gain": 0.10},
					{"band": 3, "gain": 0.05},
				},
			}
		case "piano":
			filters = map[string]any{
				"equalizer": []map[string]any{
					{"band": 0, "gain": -0.10},
					{"band": 1, "gain": -0.10},
					{"band": 2, "gain": -0.05},
					{"band": 3, "gain": 0.05},
					{"band": 4, "gain": 0.10},
					{"band": 5, "gain": 0.15},
					{"band": 6, "gain": 0.15},
					{"band": 7, "gain": 0.15},
				},
			}
		case "metal":
			filters = map[string]any{
				"equalizer": []map[string]any{
					{"band": 0, "gain": 0.15},
					{"band": 1, "gain": 0.10},
					{"band": 2, "gain": 0.05},
					{"band": 3, "gain": 0.0},
					{"band": 4, "gain": -0.05},
					{"band": 5, "gain": -0.10},
					{"band": 6, "gain": -0.05},
					{"band": 7, "gain": 0.0},
					{"band": 8, "gain": 0.05},
					{"band": 9, "gain": 0.10},
				},
			}
		case "soft":
			filters = map[string]any{
				"equalizer": []map[string]any{
					{"band": 0, "gain": 0.05},
					{"band": 1, "gain": 0.05},
					{"band": 2, "gain": 0.0},
					{"band": 3, "gain": 0.0},
					{"band": 4, "gain": 0.0},
					{"band": 5, "gain": -0.05},
					{"band": 6, "gain": -0.05},
					{"band": 7, "gain": -0.10},
					{"band": 8, "gain": -0.10},
					{"band": 9, "gain": -0.10},
				},
			}
		case "vibrato":
			filters = map[string]any{
				"tremolo": map[string]any{
					"frequency": 10,
					"depth":     0.5,
				},
			}
		case "8d":
			filters = map[string]any{
				"rotation": map[string]any{
					"rotationHz": 0.2,
				},
			}
		case "karaoke":
			filters = map[string]any{
				"karaoke": map[string]any{
					"level":       1.0,
					"monoLevel":   1.0,
					"filterBand":  220.0,
					"filterWidth": 100.0,
				},
			}
		}
	}
	payload := map[string]any{
		"filters": filters,
	}
	return p.client.UpdatePlayer(p.guildID, payload)
}
func (p *Player) Preset() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.preset == "" {
		return "none"
	}
	return p.preset
}