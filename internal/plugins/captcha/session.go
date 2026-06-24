package captcha
import (
	"math/rand"
	"sync"
	"time"
)
type CaptchaSession struct {
	GuildID   string
	Answer    string
	Attempts  int
	ExpiresAt time.Time
}
var (
	sessionsMu sync.Mutex
	sessions   = make(map[string]CaptchaSession)                               
)
func setSession(userID, guildID, answer string, timeoutMinutes int) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	key := userID + ":" + guildID
	sessions[key] = CaptchaSession{
		GuildID:   guildID,
		Answer:    answer,
		Attempts:  0,
		ExpiresAt: time.Now().Add(time.Duration(timeoutMinutes) * time.Minute),
	}
}
func getSession(userID, guildID string) (CaptchaSession, bool) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	key := userID + ":" + guildID
	s, ok := sessions[key]
	if !ok {
		return CaptchaSession{}, false
	}
	if time.Now().After(s.ExpiresAt) {
		delete(sessions, key)
		return CaptchaSession{}, false
	}
	return s, true
}
func updateAttempts(userID, guildID string, attempts int) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	key := userID + ":" + guildID
	if s, ok := sessions[key]; ok {
		s.Attempts = attempts
		sessions[key] = s
	}
}
func updateAnswer(userID, guildID, answer string) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	key := userID + ":" + guildID
	if s, ok := sessions[key]; ok {
		s.Answer = answer
		sessions[key] = s
	}
}
func deleteSession(userID, guildID string) {
	sessionsMu.Lock()
	sessionsMu.Unlock()
	delete(sessions, userID+":"+guildID)
}
func genDecoy(correct string) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, len(correct))
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}