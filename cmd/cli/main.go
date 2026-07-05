package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		usage(stdout)
		return 2
	}

	switch args[0] {
	case "version":
		fmt.Fprintln(stdout, "aphrodite-cli v0.1.0")
	case "help", "-h", "--help":
		usage(stdout)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		usage(stderr)
		return 2
	}
	return 0
}

func usage(w io.Writer) {
	fmt.Fprint(w, `aphrodite CLI

Usage:
  aphrodite-cli <command> [args...]

Commands:
  version    Print the CLI version
  help       Show this help

Planned commands (wired to the same use cases the API uses):
  users create       Create a user (role=admin requires super admin key)
  users promote      Promote an existing user to admin
  posts purge        Delete a post and its comments
`)
}
