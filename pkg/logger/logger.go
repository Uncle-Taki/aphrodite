package logger

import (
	"log/slog"
	"os"
)

var L *slog.Logger = slog.Default()

func Init(debug bool) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	L = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(L)
}
