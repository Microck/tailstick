// Package tailscale provides a client wrapper around the tailscale CLI and API
// for device enrollment, status checks, and device deletion.
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

// Client wraps tailscale CLI operations and the Tailscale API for device management.
type Client struct {
	Runner platform.Runner
}

var deleteDeviceHTTPClient = http.DefaultClient

func (c Client) IsInstalled(ctx context.Context) bool {
	_, err := c.Runner.Run(ctx, []string{"tailscale", "version"})
	return err == nil
}

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

	args := []string{
		"tailscale", "up",
		"--auth-key=" + auth,
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
	_, err := c.Runner.Run(ctx, args)
	return err
}

func (c Client) Down(ctx context.Context) error {
	_, err := c.Runner.Run(ctx, []string{"tailscale", "down"})
	return err
}

func (c Client) Logout(ctx context.Context) error {
	_, err := c.Runner.Run(ctx, []string{"tailscale", "logout"})
	return err
}

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

func (c Client) Uninstall(ctx context.Context, preset model.Preset) error {
	cmd := uninstallCommand(preset)
	if len(cmd) == 0 {
		return nil
	}
	_, err := c.Runner.Run(ctx, cmd)
	return err
}

// DeleteDevice removes a device from the tailnet using the Tailscale API.
// It is a no-op if either apiKey or deviceID is empty.
func DeleteDevice(ctx context.Context, apiKey, deviceID string) error {
	if strings.TrimSpace(apiKey) == "" || strings.TrimSpace(deviceID) == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, "https://api.tailscale.com/api/v2/device/"+deviceID, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(apiKey, "")
	resp, err := deleteDeviceHTTPClient.Do(req)
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

// BuildMachineContext constructs a machine-bound context string from the OS, architecture,
// hostname, and (on Linux) the machine-id. Used to bind encrypted secrets to a host.
func BuildMachineContext(host, _ string) string {
	info := []string{runtime.GOOS, runtime.GOARCH, strings.ToLower(strings.TrimSpace(host))}
	if runtime.GOOS == "linux" {
		if b, err := os.ReadFile("/etc/machine-id"); err == nil {
			info = append(info, strings.TrimSpace(string(b)))
		}
	}
	return strings.Join(info, "|")
}

// ParseDurationDays resolves the effective lease duration in days based on mode and operator inputs.
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

// Future returns a time pointer set to ts plus the given number of days, or nil if days <= 0.
func Future(ts time.Time, days int) *time.Time {
	if days <= 0 {
		return nil
	}
	t := ts.Add(time.Duration(days) * 24 * time.Hour)
	return &t
}
