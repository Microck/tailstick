// Package tailscale provides a client for interacting with the Tailscale CLI and API.
// It handles installation checks, auth key-based enrollment, device deletion via the
// Tailscale API, and status queries.
package tailscale

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/tailstick/tailstick/internal/model"
	"github.com/tailstick/tailstick/internal/platform"
)

// Client wraps Tailscale CLI operations through a platform.Runner for testability.
type Client struct {
	Runner platform.Runner
}

var defaultDeleteDeviceHTTPClient = &http.Client{Timeout: 15 * time.Second}

// IsInstalled checks whether the Tailscale CLI is available on the system.
func (c Client) IsInstalled(ctx context.Context) bool {
	_, err := c.Runner.Run(ctx, []string{"tailscale", "version"})
	return err == nil
}

// EnsureInstalled installs Tailscale if not present, then pins the version for the stable channel.
func (c Client) EnsureInstalled(ctx context.Context, preset model.Preset, channel model.Channel, stableVersion string) error {
	if !c.IsInstalled(ctx) {
		cmd := installCommand(preset, channel)
		if len(cmd) == 0 {
			return fmt.Errorf("no install command configured for platform %s", runtime.GOOS)
		}
		if _, err := c.Runner.Run(ctx, cmd); err != nil {
			return err
		}
	}
	if channel == model.ChannelStable {
		version := strings.TrimSpace(stableVersion)
		if version == "" {
			return fmt.Errorf("stable channel requires configured stable version")
		}
		// Enforce stable pinning explicitly so stable never means "latest".
		if _, err := c.Runner.Run(ctx, []string{"tailscale", "update", "--yes", "--version", version}); err != nil {
			return fmt.Errorf("pin stable version %s: %w", version, err)
		}
	}
	return nil
}

// Up runs "tailscale up" with auth key, hostname, tags, routes, and optional exit node.
func (c Client) Up(ctx context.Context, preset model.Preset, deviceName string, mode model.LeaseMode, exitNode string) error {
	auth := preset.AuthKey
	if mode == model.LeaseModeSession {
		if strings.TrimSpace(preset.EphemeralAuthKey) == "" {
			return fmt.Errorf("session mode requires ephemeral auth key")
		}
		auth = preset.EphemeralAuthKey
	}
	if strings.TrimSpace(auth) == "" {
		return fmt.Errorf("missing auth key")
	}

	authArg, cleanupAuthKeyFile, err := authKeyArg(auth)
	if err != nil {
		return err
	}
	defer cleanupAuthKeyFile()

	args := []string{
		"tailscale", "up",
		authArg,
		"--hostname=" + deviceName,
		"--reset",
	}
	if len(preset.Tags) > 0 {
		args = append(args, "--advertise-tags="+strings.Join(preset.Tags, ","))
	}
	if preset.AcceptRoutes {
		args = append(args, "--accept-routes")
	}
	if exitNode != "" {
		args = append(args, "--exit-node="+exitNode)
	}
	_, err = c.Runner.Run(ctx, args)
	return err
}

// Down runs "tailscale down" to disconnect from the tailnet.
func (c Client) Down(ctx context.Context) error {
	_, err := c.Runner.Run(ctx, []string{"tailscale", "down"})
	return err
}

// Logout runs "tailscale logout" to remove local node credentials.
func (c Client) Logout(ctx context.Context) error {
	_, err := c.Runner.Run(ctx, []string{"tailscale", "logout"})
	return err
}

// Status queries the current Tailscale status, with a fallback for schema drift.
func (c Client) Status(ctx context.Context) (model.TailscaleStatus, error) {
	out, err := c.Runner.Run(ctx, []string{"tailscale", "status", "--json"})
	if err != nil {
		return model.TailscaleStatus{}, err
	}
	var parsed model.TailscaleStatus
	if err := json.Unmarshal([]byte(out), &parsed); err == nil {
		return parsed, nil
	}

	// Fallback for schema drift: decode the minimal self data dynamically.
	var root map[string]any
	if err := json.Unmarshal([]byte(out), &root); err != nil {
		return model.TailscaleStatus{}, err
	}
	selfRaw, ok := root["Self"].(map[string]any)
	if !ok {
		return model.TailscaleStatus{}, fmt.Errorf("status json missing Self object")
	}
	var st model.TailscaleStatus
	if v, ok := selfRaw["ID"].(string); ok {
		st.Self.ID = v
	}
	if v, ok := selfRaw["DNSName"].(string); ok {
		st.Self.DNSName = v
	}
	if v, ok := selfRaw["HostName"].(string); ok {
		st.Self.HostName = v
	}
	return st, nil
}

// Uninstall removes Tailscale using the preset's configured uninstall command.
func (c Client) Uninstall(ctx context.Context, preset model.Preset) error {
	cmd := uninstallCommand(preset)
	if len(cmd) == 0 {
		return nil
	}
	_, err := c.Runner.Run(ctx, cmd)
	return err
}

// DeleteDevice removes a device from the tailnet via the Tailscale API.
func DeleteDevice(ctx context.Context, apiKey, deviceID string) error {
	return deleteDevice(ctx, defaultDeleteDeviceHTTPClient, apiKey, deviceID)
}

func deleteDevice(ctx context.Context, client *http.Client, apiKey, deviceID string) error {
	if strings.TrimSpace(apiKey) == "" || strings.TrimSpace(deviceID) == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, "https://api.tailscale.com/api/v2/device/"+deviceID, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(apiKey, "")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	bodyText := strings.TrimSpace(string(body))
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	return fmt.Errorf("delete device failed: status=%d body=%s", resp.StatusCode, bodyText)
}

func authKeyArg(auth string) (string, func(), error) {
	f, err := os.CreateTemp("", "tailstick-auth-key-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("create auth key temp file: %w", err)
	}
	path := f.Name()
	cleanup := func() {
		_ = os.Remove(path)
	}
	if _, err := f.WriteString(auth); err != nil {
		cleanup()
		_ = f.Close()
		return "", func() {}, fmt.Errorf("write auth key temp file: %w", err)
	}
	if err := f.Chmod(0o600); err != nil && runtime.GOOS != "windows" {
		cleanup()
		_ = f.Close()
		return "", func() {}, fmt.Errorf("chmod auth key temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("close auth key temp file: %w", err)
	}
	return "--auth-key=file:" + path, cleanup, nil
}

func installCommand(preset model.Preset, channel model.Channel) []string {
	if runtime.GOOS == "windows" {
		if channel == model.ChannelLatest && len(preset.Install.WindowsLatest) > 0 {
			return preset.Install.WindowsLatest
		}
		if channel == model.ChannelStable && len(preset.Install.WindowsStable) > 0 {
			return preset.Install.WindowsStable
		}
		return []string{"powershell", "-NoProfile", "-Command", "winget install -e --id Tailscale.Tailscale --accept-package-agreements --accept-source-agreements"}
	}
	if channel == model.ChannelLatest && len(preset.Install.LinuxLatest) > 0 {
		return preset.Install.LinuxLatest
	}
	if channel == model.ChannelStable && len(preset.Install.LinuxStable) > 0 {
		return preset.Install.LinuxStable
	}
	return []string{"bash", "-lc", "curl -fsSL https://tailscale.com/install.sh | sh"}
}

func uninstallCommand(preset model.Preset) []string {
	if runtime.GOOS == "windows" {
		if len(preset.Install.WindowsUninstall) > 0 {
			return preset.Install.WindowsUninstall
		}
		return []string{"powershell", "-NoProfile", "-Command", "winget uninstall -e --id Tailscale.Tailscale"}
	}
	if len(preset.Install.LinuxUninstall) > 0 {
		return preset.Install.LinuxUninstall
	}
	return []string{"bash", "-lc", "apt-get remove -y tailscale"}
}

// BuildMachineContext assembles a machine-specific context string used for secret key derivation.
func BuildMachineContext(host, _ string) string {
	info := []string{runtime.GOOS, runtime.GOARCH, strings.ToLower(strings.TrimSpace(host))}
	if runtime.GOOS == "linux" {
		if b, err := os.ReadFile("/etc/machine-id"); err == nil {
			info = append(info, strings.TrimSpace(string(b)))
		}
	}
	return strings.Join(info, "|")
}

// ParseDurationDays validates and returns the lease duration in days for the given mode.
func ParseDurationDays(mode model.LeaseMode, defaultDays, customDays int) (int, error) {
	switch mode {
	case model.LeaseModeSession:
		return 0, nil
	case model.LeaseModePermanent:
		return 0, nil
	case model.LeaseModeTimed:
		if customDays > 0 {
			if customDays < 1 || customDays > 30 {
				return 0, fmt.Errorf("custom days must be between 1 and 30")
			}
			return customDays, nil
		}
		for _, allowed := range []int{1, 3, 7} {
			if defaultDays == allowed {
				return defaultDays, nil
			}
		}
		return 0, fmt.Errorf("timed lease requires days in {1,3,7} or custom-days in [1,30]")
	default:
		return 0, fmt.Errorf("invalid lease mode %s", mode)
	}
}

// Future returns a time pointer set to the given number of days from now, or nil if days <= 0.
func Future(ts time.Time, days int) *time.Time {
	if days <= 0 {
		return nil
	}
	t := ts.Add(time.Duration(days) * 24 * time.Hour)
	return &t
}
