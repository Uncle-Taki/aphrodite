package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	cases := map[string]struct {
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		"missing command": {wantCode: 2, wantStdout: "Usage:"},
		"version":         {args: []string{"version"}, wantCode: 0, wantStdout: "aphrodite-cli v0.1.0"},
		"help":            {args: []string{"help"}, wantCode: 0, wantStdout: "Planned commands"},
		"short help":      {args: []string{"-h"}, wantCode: 0, wantStdout: "Usage:"},
		"long help":       {args: []string{"--help"}, wantCode: 0, wantStdout: "Commands:"},
		"unknown":         {args: []string{"wat"}, wantCode: 2, wantStderr: "unknown command: wat"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tc.args, &stdout, &stderr)
			if code != tc.wantCode {
				t.Fatalf("code = %d, want %d", code, tc.wantCode)
			}
			if tc.wantStdout != "" && !strings.Contains(stdout.String(), tc.wantStdout) {
				t.Fatalf("stdout missing %q: %s", tc.wantStdout, stdout.String())
			}
			if tc.wantStderr != "" && !strings.Contains(stderr.String(), tc.wantStderr) {
				t.Fatalf("stderr missing %q: %s", tc.wantStderr, stderr.String())
			}
		})
	}
}
