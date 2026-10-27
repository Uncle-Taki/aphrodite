package config

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRequiredSettings_RejectInvalidValues(t *testing.T) {
	for _, tc := range []struct {
		name, value string
		read        func()
	}{
		{"integer", "abc", func() { mustGetEnvInt("APHRODITE_REQUIRED_TEST") }},
		{"positive integer", "0", func() { mustGetPositiveEnvInt("APHRODITE_REQUIRED_TEST") }},
		{"negative integer", "-1", func() { mustGetPositiveEnvInt("APHRODITE_REQUIRED_TEST") }},
		{"duration", "tomorrow", func() { mustGetEnvDuration("APHRODITE_REQUIRED_TEST") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("APHRODITE_REQUIRED_TEST", tc.value)
			defer func() {
				recovered := recover()
				if recovered == nil || !strings.Contains(fmt.Sprint(recovered), "APHRODITE_REQUIRED_TEST") {
					t.Fatalf("expected configuration panic naming variable, got %v", recovered)
				}
			}()
			tc.read()
		})
	}
	t.Setenv("APHRODITE_REQUIRED_TEST", "3s")
	if got := mustGetEnvDuration("APHRODITE_REQUIRED_TEST"); got != 3*time.Second {
		t.Fatalf("duration = %v", got)
	}
}
