package main

import (
	"os"

	"github.com/Microck/tailstick/internal/app"
)

func main() {
	os.Exit(app.RunGUI(os.Args[1:], app.Runtime{}))
}
