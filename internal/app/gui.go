package app

import (
	"flag"
	"fmt"
	"os"

	"github.com/tailstick/tailstick/internal/gui"
)

func RunGUI(args []string, rt Runtime) int {
	fs := flag.NewFlagSet("gui", flag.ContinueOnError)
	var (
		configPath  = fs.String("config", rt.ConfigPath, "config path")
		statePath   = fs.String("state", rt.StatePath, "state path")
		auditPath   = fs.String("audit", rt.AuditPath, "audit path")
		logPath     = fs.String("log", rt.LogPath, "log path")
		dryRun      = fs.Bool("dry-run", rt.DryRun, "dry-run mode")
		openBrowser = fs.Bool("open-browser", true, "open browser automatically")
		host        = fs.String("host", "127.0.0.1", "bind host")
		port        = fs.Int("port", 0, "bind port (0 picks an ephemeral port)")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	mgr, err := NewManager(Runtime{
		ConfigPath: *configPath,
		StatePath:  *statePath,
		AuditPath:  *auditPath,
		LogPath:    *logPath,
		DryRun:     *dryRun,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "init:", err)
		return 1
	}
	defer mgr.Close()

	if err := gui.Run(interruptContext(), &gui.Server{
		ConfigPath: mgr.Runtime.ConfigPath,
		Logf:       mgr.Logger.Info,
		EnrollFn:   mgr.Enroll,
	}, *openBrowser, *host, *port); err != nil {
		fmt.Fprintln(os.Stderr, "gui:", err)
		return 1
	}
	return 0
}
