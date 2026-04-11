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

type Context struct {
	OS      string
	Arch    string
	Host    string
	BootID  string
	ExePath string
}

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

func IsLinux() bool   { return runtime.GOOS == "linux" }
func IsWindows() bool { return runtime.GOOS == "windows" }

func windowsProgramData() string {
	root := os.Getenv("ProgramData")
	if root == "" {
		root = `C:\ProgramData`
	}
	return root
}

func StatePath() string {
	if IsWindows() {
		return filepath.Join(windowsProgramData(), "TailStick", "state.json")
	}
	return "/var/lib/tailstick/state.json"
}

func LogPath() string {
	if IsWindows() {
		return filepath.Join(windowsProgramData(), "TailStick", "tailstick.log")
	}
	return "/var/log/tailstick.log"
}

func LocalSecretPath() string {
	if IsWindows() {
		return filepath.Join(windowsProgramData(), "TailStick", "secrets")
	}
	return "/var/lib/tailstick/secrets"
}

func AgentBinaryPath() string {
	if IsWindows() {
		return filepath.Join(windowsProgramData(), "TailStick", "tailstick-agent.exe")
	}
	return "/var/lib/tailstick/tailstick-agent"
}

func EnsureParent(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
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

func ElevationHint(exePath string, args []string) string {
	if IsLinux() {
		return "rerun with sudo"
	}
	if IsWindows() {
		return "rerun from an elevated PowerShell or Command Prompt (Run as Administrator)"
	}
	return "rerun with elevated privileges"
}
