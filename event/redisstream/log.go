package redisstream

import (
	"log/slog"

	"github.com/lishimeng/x/event"
)

func Infof(format string, args ...any) {
	if event.Debug {
		slog.Info(format, args...)
	}

}
