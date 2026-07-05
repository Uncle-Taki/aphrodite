package middleware

import (
	"errors"
	"testing"
)

type failingRequestIDReader struct{}

func (failingRequestIDReader) Read([]byte) (int, error) {
	return 0, errors.New("request id entropy unavailable")
}

func TestNewRequestIDEntropyError(t *testing.T) {
	old := requestIDReader
	requestIDReader = failingRequestIDReader{}
	t.Cleanup(func() {
		requestIDReader = old
	})

	if got := newRequestID(); got != "" {
		t.Fatalf("expected blank request id on entropy failure, got %q", got)
	}
}
