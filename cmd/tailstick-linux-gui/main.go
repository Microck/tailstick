// Package main is the Linux GUI entry point for tailstick.
package main

import (
	"os"

	"github.com/tailstick/tailstick/internal/app"
)

func main() {
	os.Exit(app.RunGUI(os.Args[1:], app.Runtime{}))
}
