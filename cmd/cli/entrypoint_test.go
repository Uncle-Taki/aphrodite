package main

import (
	"os"
	"testing"
)

func TestMain_ReturnsCommandExitCode(t *testing.T) {
	previousArgs, previousExit := os.Args, exitProcess
	t.Cleanup(func() { os.Args = previousArgs; exitProcess = previousExit })
	os.Args = []string{"aphrodite-cli", "version"}
	code := -1
	exitProcess = func(value int) { code = value }
	main()
	if code != 0 {
		t.Fatalf("exit code=%d", code)
	}
}
