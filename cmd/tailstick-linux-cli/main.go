// Package main is the Linux CLI entry point for tailstick.
package main

import (
	"os"

	"github.com/tailstick/tailstick/internal/app"
)

func main() {
	os.Exit(app.RunCLI(os.Args[1:], app.Runtime{}))
}
