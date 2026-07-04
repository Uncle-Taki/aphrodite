package logger_test

import (
	"log/slog"
	"testing"

	"aphrodite/pkg/logger"
)

func TestL_DefaultIsNonNil(t *testing.T) {
	if logger.L == nil {
		t.Fatal("logger.L must be initialised at package load")
	}
}

func TestInit_ReplacesGlobalLoggerAndSlogDefault(t *testing.T) {
	prev := logger.L
	defer func() {
		logger.L = prev
		slog.SetDefault(prev)
	}()

	logger.Init(true)
	if logger.L == prev {
		t.Fatal("Init should replace logger.L")
	}
	if slog.Default() != logger.L {
		t.Fatal("Init should make logger.L the slog default")
	}
}

func TestInit_DebugAndNonDebugBothInitialise(t *testing.T) {
	prev := logger.L
	defer func() {
		logger.L = prev
		slog.SetDefault(prev)
	}()
	logger.Init(false)
	debug := logger.L
	logger.Init(true)
	if logger.L == debug {
		t.Fatal("calling Init again should yield a fresh logger")
	}
}
