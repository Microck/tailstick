// Package platform provides host detection, path resolution, and privilege checks
// for supported operating systems (Linux and Windows).
package platform

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Context holds detected host properties including OS, architecture, hostname,
// boot ID, and the executable path of the running binary.
type Context struct {
	OS      string
	Arch    string
	Host    string
	BootID  string
	ExePath string
}

// Detect collects the current host's OS, architecture, hostname, boot ID, and executable path.
func Detect() (Context, error) {
	host, err := os.Hostname()
	if err != nil {
		return Context{}, fmt.Errorf("hostname: %w", err)
	}
	exe, err := os.Executable()
	if err != nil {
		return Context{}, fmt.Errorf("executable: %w", err)
	}
	ctx := Context{
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
		Host:    sanitizeHost(host),
		BootID:  "",
		ExePath: exe,
	}
	ctx.BootID = detectBootID(ctx.OS)
	return ctx, nil
}

// IsLinux returns true when running on a Linux host.
func IsLinux() bool   { return runtime.GOOS == "linux" }

// IsWindows returns true when running on a Windows host.
func IsWindows() bool { return runtime.GOOS == "windows" }

// StatePath returns the platform-specific path for the state.json file.
func StatePath() string {
	if IsWindows() {
		root := os.Getenv("ProgramData")
		if root == "" {
			root = `C:\ProgramData`
		}
		return filepath.Join(root, "TailStick", "state.json")
	}
	return "/var/lib/tailstick/state.json"
}

// LogPath returns the platform-specific path for the tailstick log file.
func LogPath() string {
	if IsWindows() {
		root := os.Getenv("ProgramData")
		if root == "" {
			root = `C:\ProgramData`
		}
		return filepath.Join(root, "TailStick", "tailstick.log")
	}
	return "/var/log/tailstick.log"
}

// LocalSecretPath returns the platform-specific directory for locally encrypted secrets.
func LocalSecretPath() string {
	if IsWindows() {
		root := os.Getenv("ProgramData")
		if root == "" {
			root = `C:\ProgramData`
		}
		return filepath.Join(root, "TailStick", "secrets")
	}
	return "/var/lib/tailstick/secrets"
}

// AgentBinaryPath returns the platform-specific path where the reconciliation agent binary is installed.
func AgentBinaryPath() string {
	if IsWindows() {
		root := os.Getenv("ProgramData")
		if root == "" {
			root = `C:\ProgramData`
		}
		return filepath.Join(root, "TailStick", "tailstick-agent.exe")
	}
	return "/var/lib/tailstick/tailstick-agent"
}

// EnsureParent creates all directories in the path of the given file's parent directory.
func EnsureParent(path string) error {
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	return nil
}

func detectBootID(osName string) string {
	switch osName {
	case "linux":
		b, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
		if err == nil {
			return strings.TrimSpace(string(b))
		}
	case "windows":
		out, err := exec.Command("wmic", "os", "get", "lastbootuptime", "/value").CombinedOutput()
		if err == nil {
			sc := bufio.NewScanner(strings.NewReader(string(out)))
			for sc.Scan() {
				line := strings.TrimSpace(sc.Text())
				if strings.HasPrefix(strings.ToLower(line), "lastbootuptime=") {
					return strings.TrimPrefix(line, "LastBootUpTime=")
				}
			}
		}
	}
	return "unknown"
}

func sanitizeHost(in string) string {
	trimmed := strings.TrimSpace(strings.ToLower(in))
	if trimmed == "" {
		return "host"
	}
	var out []rune
	for _, r := range trimmed {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			out = append(out, r)
			continue
		}
		if r == '_' || r == '.' || r == ' ' {
			out = append(out, '-')
		}
	}
	if len(out) == 0 {
		return "host"
	}
	return strings.Trim(string(out), "-")
}

// RequireSupportedLinux returns an error if the current Linux distribution is not Debian or Ubuntu.
func RequireSupportedLinux() error {
	if !IsLinux() {
		return nil
	}
	b, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return errors.New("linux target unsupported: cannot read /etc/os-release")
	}
	txt := strings.ToLower(string(b))
	if strings.Contains(txt, "id=ubuntu") || strings.Contains(txt, "id=debian") {
		return nil
	}
	return errors.New("linux target unsupported: only debian/ubuntu are supported in v1")
}

// IsElevated reports whether the current process is running with administrator/root privileges.
func IsElevated() bool {
	if IsLinux() {
		return os.Geteuid() == 0
	}
	if IsWindows() {
		// "net session" exits with non-zero when not elevated.
		err := exec.Command("cmd", "/c", "net", "session").Run()
		return err == nil
	}
	return true
}

// ElevationHint returns a human-readable message describing how to re-run with elevated privileges.
func ElevationHint(exePath string, args []string) string {
	if IsLinux() {
		return "rerun with sudo"
	}
	if IsWindows() {
		return "rerun from an elevated PowerShell or Command Prompt (Run as Administrator)"
	}
	return "rerun with elevated privileges"
}
