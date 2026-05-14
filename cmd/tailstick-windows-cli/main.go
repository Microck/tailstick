package main

import (
	"os"

	"github.com/Microck/tailstick/internal/app"
)

func main() {
	os.Exit(app.RunCLI(os.Args[1:], app.Runtime{}))
}
