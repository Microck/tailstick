package app

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tailstick/tailstick/internal/model"
)

const Version = "0.1.0"

func RunCLI(args []string, rt Runtime) int {
	if len(args) == 0 {
		return runEnroll(args, rt)
	}
	switch args[0] {
	case "run":
		return runEnroll(args[1:], rt)
	case "agent":
		return runAgent(args[1:], rt)
	case "cleanup":
		return runCleanup(args[1:], rt)
	case "version":
		fmt.Println(Version)
		return 0
	case "help", "-h", "--help":
		printHelp()
		return 0
	default:
		// Default to enrollment to keep operator UX simple.
		return runEnroll(args, rt)
	}
}

func runEnroll(args []string, rt Runtime) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	var (
		preset         = fs.String("preset", "", "preset id")
		mode           = fs.String("mode", "session", "lease mode: session|timed|permanent")
		channel        = fs.String("channel", "stable", "install channel: stable|latest")
		days           = fs.Int("days", 3, "timed lease days: 1,3,7")
		customDays     = fs.Int("custom-days", 0, "advanced custom lease days (1-30)")
		suffix         = fs.String("suffix", "", "optional device name suffix")
		exitNode       = fs.String("exit-node", "", "optional approved exit node")
		allowExisting  = fs.Bool("allow-existing", false, "allow existing tailscale install")
		nonInteractive = fs.Bool("non-interactive", false, "non-interactive mode")
		password       = fs.String("password", "", "optional operator password for gated presets/config")
		configPath     = fs.String("config", rt.ConfigPath, "config path")
		statePath      = fs.String("state", rt.StatePath, "state path")
		auditPath      = fs.String("audit", rt.AuditPath, "audit path")
		logPath        = fs.String("log", rt.LogPath, "log path")
		dryRun         = fs.Bool("dry-run", rt.DryRun, "print commands without executing")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if pwEnv := strings.TrimSpace(os.Getenv("TAILSTICK_OPERATOR_PASSWORD")); pwEnv != "" && *password == "" {
		*password = pwEnv
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

	ctx := interruptContext()
	rec, err := mgr.Enroll(ctx, model.RuntimeOptions{
		PresetID:       *preset,
		Mode:           model.LeaseMode(*mode),
		Channel:        model.Channel(*channel),
		Days:           *days,
		CustomDays:     *customDays,
		DeviceSuffix:   *suffix,
		ExitNode:       *exitNode,
		AllowExisting:  *allowExisting,
		NonInteractive: *nonInteractive,
		Password:       *password,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "enroll:", err)
		return 1
	}
	fmt.Printf("Lease created: %s\nDevice name: %s\nMode: %s\n", rec.LeaseID, rec.DeviceName, rec.Mode)
	if rec.ExpiresAt != nil {
		fmt.Printf("Expires at: %s\n", rec.ExpiresAt.UTC().Format(time.RFC3339))
	}
	return 0
}

func runAgent(args []string, rt Runtime) int {
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	var (
		once       = fs.Bool("once", false, "run one reconciliation pass")
		interval   = fs.Duration("interval", time.Minute, "reconciliation interval")
		configPath = fs.String("config", rt.ConfigPath, "config path")
		statePath  = fs.String("state", rt.StatePath, "state path")
		auditPath  = fs.String("audit", rt.AuditPath, "audit path")
		logPath    = fs.String("log", rt.LogPath, "log path")
		dryRun     = fs.Bool("dry-run", rt.DryRun, "dry-run mode")
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

	ctx := interruptContext()
	if *once {
		if err := mgr.AgentOnce(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "agent:", err)
			return 1
		}
		return 0
	}
	if err := mgr.AgentRun(ctx, *interval); err != nil && err != context.Canceled {
		fmt.Fprintln(os.Stderr, "agent:", err)
		return 1
	}
	return 0
}

func runCleanup(args []string, rt Runtime) int {
	fs := flag.NewFlagSet("cleanup", flag.ContinueOnError)
	var (
		leaseID    = fs.String("lease-id", "", "lease id to clean")
		configPath = fs.String("config", rt.ConfigPath, "config path")
		statePath  = fs.String("state", rt.StatePath, "state path")
		auditPath  = fs.String("audit", rt.AuditPath, "audit path")
		logPath    = fs.String("log", rt.LogPath, "log path")
		dryRun     = fs.Bool("dry-run", rt.DryRun, "dry-run mode")
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
	if err := mgr.ForceCleanup(interruptContext(), *leaseID); err != nil {
		fmt.Fprintln(os.Stderr, "cleanup:", err)
		return 1
	}
	fmt.Println("cleanup complete")
	return 0
}

func interruptContext() context.Context {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		cancel()
	}()
	return ctx
}

func printHelp() {
	fmt.Print(`tailstick commands:
  run       create lease and enroll
  agent     run lease reconciliation agent
  cleanup   force cleanup by lease id
  version   print version
`)
}
