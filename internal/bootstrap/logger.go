package bootstrap

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"skyvern/internal/config"
	"skyvern/pkg/tui"
)

func SetupLogger() *os.File {
	f, err := os.OpenFile(config.ResolvePath("skyvern.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return os.Stderr
	}
	_, _ = fmt.Fprintf(f, "started %s\n\n", time.Now().Format(time.RFC3339))
	mw := io.MultiWriter(f, &tui.LogInterceptor{})
	log.SetOutput(mw)
	os.Stderr = f
	return f
}
