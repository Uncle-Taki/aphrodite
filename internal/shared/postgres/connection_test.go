package postgres

import (
	"testing"

	"aphrodite/pkg/config"
)

func TestConnectReturnsErrorForUnreachableDatabase(t *testing.T) {
	_, err := Connect(config.DatabaseConfig{
		Host:     "127.0.0.1",
		Port:     "1",
		User:     "aphrodite",
		Password: "aphrodite",
		Name:     "aphrodite",
		SSLMode:  "disable",
	})
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestConnectDebugLoggerBranchReturnsErrorForUnreachableDatabase(t *testing.T) {
	oldDebug := config.C.Debug
	config.C.Debug = true
	defer func() { config.C.Debug = oldDebug }()

	_, err := Connect(config.DatabaseConfig{
		Host:     "127.0.0.1",
		Port:     "1",
		User:     "aphrodite",
		Password: "aphrodite",
		Name:     "aphrodite",
		SSLMode:  "disable",
	})
	if err == nil {
		t.Fatal("expected connection error")
	}
}
