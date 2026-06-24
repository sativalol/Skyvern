package tui

import (
	"strings"
	"sync"
)

var (
	logLines []string
	logMu    sync.RWMutex
	maxLines = 100
)

type LogInterceptor struct{}

func (l *LogInterceptor) Write(p []byte) (n int, err error) {
	logMu.Lock()
	defer logMu.Unlock()
	
	text := strings.TrimSpace(string(p))
	if text == "" {
		return len(p), nil
	}

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		logLines = append(logLines, line)
		if len(logLines) > maxLines {
			logLines = logLines[len(logLines)-maxLines:]
		}
	}
	return len(p), nil
}

func GetLogs() []string {
	logMu.RLock()
	defer logMu.RUnlock()
	cpy := make([]string, len(logLines))
	copy(cpy, logLines)
	return cpy
}
