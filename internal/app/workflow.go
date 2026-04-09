package app

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/tailstick/tailstick/internal/config"
	intcrypto "github.com/tailstick/tailstick/internal/crypto"
	"github.com/tailstick/tailstick/internal/logging"
	"github.com/tailstick/tailstick/internal/model"
	"github.com/tailstick/tailstick/internal/platform"
	"github.com/tailstick/tailstick/internal/state"
	"github.com/tailstick/tailstick/internal/tailscale"
)

type Runtime struct {
	ConfigPath string
	StatePath  string
	AuditPath  string
	LogPath    string
	DryRun     bool
}

type Manager struct {
	Runtime Runtime
	Logger  *logging.Logger
	Runner  platform.Runner
	TS      tailscale.Client
	HostCtx platform.Context
}

func NewManager(rt Runtime) (*Manager, error) {
	host, err := platform.Detect()
	if err != nil {
		return nil, err
	}
	if rt.ConfigPath == "" {
		rt.ConfigPath = defaultConfigPath(host.ExePath)
	}
	if rt.StatePath == "" {
		rt.StatePath = platform.StatePath()
	}
	if rt.AuditPath == "" {
		rt.AuditPath = filepath.Join(filepath.Dir(rt.ConfigPath), "logs", "tailstick-audit.ndjson")
	}
	if rt.LogPath == "" {
		rt.LogPath = platform.LogPath()
	}
	if err := platform.EnsureParent(rt.StatePath); err != nil {
		return nil, err
	}
	if err := platform.EnsureParent(rt.LogPath); err != nil {
		return nil, err
	}
	log, err := logging.New(rt.LogPath)
	if err != nil {
		return nil, err
	}
	runner := platform.Runner{DryRun: rt.DryRun}
	return &Manager{
		Runtime: rt,
		Logger:  log,
		Runner:  runner,
		TS:      tailscale.Client{Runner: runner},
		HostCtx: host,
	}, nil
}

func (m *Manager) Close() {
	_ = m.Logger.Close()
}

func (m *Manager) Enroll(ctx context.Context, opts model.RuntimeOptions) (model.LeaseRecord, error) {
	if !m.Runtime.DryRun && !platform.IsElevated() {
		return model.LeaseRecord{}, fmt.Errorf("elevated privileges are required for enrollment; %s", platform.ElevationHint(m.HostCtx.ExePath, nil))
	}
	if runtime.GOOS == "linux" {
		if err := platform.RequireSupportedLinux(); err != nil {
			return model.LeaseRecord{}, err
		}
	}

	cfg, err := config.Load(m.Runtime.ConfigPath)
	if err != nil {
		return model.LeaseRecord{}, err
	}
	preset, err := config.FindPreset(cfg, opts.PresetID)
	if err != nil {
		return model.LeaseRecord{}, err
	}
	if expected := strings.TrimSpace(cfg.OperatorPassword); expected != "" {
		if subtle.ConstantTimeCompare([]byte(opts.Password), []byte(expected)) != 1 {
			return model.LeaseRecord{}, fmt.Errorf("operator password is invalid")
		}
	}
	preset = config.ResolvePresetSecrets(preset)
	if err := validateExitNode(preset, opts.ExitNode); err != nil {
		return model.LeaseRecord{}, err
	}
	stableVersion := strings.TrimSpace(cfg.StableVersion)
	if strings.TrimSpace(preset.StableVersionOverride) != "" {
		stableVersion = strings.TrimSpace(preset.StableVersionOverride)
	}
	days, err := tailscale.ParseDurationDays(opts.Mode, opts.Days, opts.CustomDays)
	if err != nil {
		return model.LeaseRecord{}, err
	}

	alreadyInstalled := m.TS.IsInstalled(ctx)
	if alreadyInstalled && !opts.AllowExisting {
		return model.LeaseRecord{}, fmt.Errorf("tailscale is already installed; rerun with --allow-existing to override")
	}

	if err := m.TS.EnsureInstalled(ctx, preset, opts.Channel, stableVersion); err != nil {
		return model.LeaseRecord{}, fmt.Errorf("ensure tailscale installed: %w", err)
	}

	leaseID, err := newLeaseID()
	if err != nil {
		return model.LeaseRecord{}, fmt.Errorf("generate lease id: %w", err)
	}
	deviceName := buildDeviceName(opts.Mode, preset.ID, m.HostCtx.Host, leaseID, opts.DeviceSuffix)
	now := time.Now().UTC()
	expiresAt := tailscale.Future(now, days)
	if err := m.TS.Up(ctx, preset, deviceName, opts.Mode, opts.ExitNode); err != nil {
		return model.LeaseRecord{}, fmt.Errorf("tailscale up: %w", err)
	}
	status, statusErr := m.TS.Status(ctx)
	if statusErr != nil {
		m.Logger.Error("tailscale status failed: %v", statusErr)
	}

	secretPayload, err := json.Marshal(preset.Cleanup)
	if err != nil {
		return model.LeaseRecord{}, err
	}
	machineCtx := tailscale.BuildMachineContext(m.HostCtx.Host, m.HostCtx.ExePath)
	encSecret, err := intcrypto.Encrypt(string(secretPayload), "", machineCtx)
	if err != nil {
		return model.LeaseRecord{}, err
	}
	secretRef, err := m.writeLeaseSecret(leaseID, encSecret)
	if err != nil {
		return model.LeaseRecord{}, err
	}

	cleanupState := preset.Cleanup
	cleanupState.APIKey = ""

	rec := model.LeaseRecord{
		LeaseID:             leaseID,
		PresetID:            preset.ID,
		Mode:                opts.Mode,
		Channel:             opts.Channel,
		DurationDays:        days,
		Hostname:            m.HostCtx.Host,
		DeviceName:          deviceName,
		CreatedAt:           now,
		ExpiresAt:           expiresAt,
		CreatedBootID:       m.HostCtx.BootID,
		Status:              model.LeaseStatusActive,
		DeviceID:            status.Self.ID,
		NodeName:            status.Self.DNSName,
		InstallSnapshot:     preset.Install,
		PresetCleanup:       cleanupState,
		CredentialRef:       secretRef,
		EncryptedSecret:     encSecret,
		LastReconcileResult: "enrolled",
	}

	st, err := state.Load(m.Runtime.StatePath)
	if err != nil {
		return model.LeaseRecord{}, err
	}
	st = state.UpsertRecord(st, rec)
	if err := state.Save(m.Runtime.StatePath, st); err != nil {
		return model.LeaseRecord{}, err
	}
	if err := state.AppendAudit(m.Runtime.AuditPath, model.AuditEntry{
		LeaseID:    rec.LeaseID,
		Action:     "enrolled",
		PresetID:   rec.PresetID,
		Mode:       rec.Mode,
		Channel:    rec.Channel,
		DeviceName: rec.DeviceName,
		Host:       rec.Hostname,
		Message:    "lease created",
	}); err != nil {
		m.Logger.Error("append audit failed: %v", err)
	}

	if opts.Mode != model.LeaseModePermanent {
		if err := m.installAgent(ctx); err != nil {
			return rec, err
		}
	}
	m.Logger.Info("lease %s created mode=%s preset=%s device=%s", rec.LeaseID, rec.Mode, rec.PresetID, rec.DeviceName)
	return rec, nil
}

func (m *Manager) AgentOnce(ctx context.Context) error {
	st, err := state.Load(m.Runtime.StatePath)
	if err != nil {
		return err
	}
	changed := false
	now := time.Now().UTC()
	for i := range st.Records {
		rec := st.Records[i]
		if rec.Status == model.LeaseStatusCleaned {
			continue
		}
		if shouldCleanup(rec, m.HostCtx.BootID, now) {
			updated := m.cleanupRecord(ctx, rec)
			if updated.Status != rec.Status || updated.LastError != rec.LastError {
				changed = true
				st.Records[i] = updated
			}
		} else {
			ts := now
			rec.LastReconciledAt = &ts
			rec.LastReconcileResult = "no_action"
			st.Records[i] = rec
			changed = true
		}
	}
	if changed {
		if err := state.Save(m.Runtime.StatePath, st); err != nil {
			return err
		}
	}
	if !hasActiveManagedLeases(st) {
		if err := m.uninstallAgent(ctx); err != nil {
			m.Logger.Error("agent self-removal failed: %v", err)
		} else {
			m.Logger.Info("agent self-removal completed: no active managed leases")
		}
	}
	return nil
}

func (m *Manager) AgentRun(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = 1 * time.Minute
	}
	m.Logger.Info("tailstick agent started interval=%s", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := m.AgentOnce(ctx); err != nil {
			m.Logger.Error("agent iteration failed: %v", err)
		}
		st, err := state.Load(m.Runtime.StatePath)
		if err != nil {
			return err
		}
		if !hasActiveManagedLeases(st) {
			m.Logger.Info("tailstick agent stopping: no active managed leases remain")
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (m *Manager) ForceCleanup(ctx context.Context, leaseID string) error {
	if strings.TrimSpace(leaseID) == "" {
		return fmt.Errorf("lease id is required")
	}
	st, err := state.Load(m.Runtime.StatePath)
	if err != nil {
		return err
	}
	found := false
	for i := range st.Records {
		if st.Records[i].LeaseID == leaseID {
			found = true
			st.Records[i] = m.cleanupRecord(ctx, st.Records[i])
			break
		}
	}
	if !found {
		return fmt.Errorf("lease %s not found", leaseID)
	}
	return state.Save(m.Runtime.StatePath, st)
}

func (m *Manager) cleanupRecord(ctx context.Context, rec model.LeaseRecord) model.LeaseRecord {
	ts := time.Now().UTC()
	rec.LastReconciledAt = &ts
	rec.LastReconcileResult = "cleanup_started"
	rec.Status = model.LeaseStatusCleanupQueued
	var errs []string
	preset := m.resolvePresetFromConfig(rec.PresetID)
	if !installSnapshotEmpty(rec.InstallSnapshot) {
		preset.Install = rec.InstallSnapshot
	}

	if err := m.TS.Down(ctx); err != nil {
		errs = append(errs, "tailscale down: "+err.Error())
	}
	if err := m.TS.Logout(ctx); err != nil {
		errs = append(errs, "tailscale logout: "+err.Error())
	}

	cleanupCfg := rec.PresetCleanup
	encodedSecret := rec.EncryptedSecret
	if strings.TrimSpace(rec.CredentialRef) != "" {
		if b, err := os.ReadFile(rec.CredentialRef); err == nil {
			encodedSecret = strings.TrimSpace(string(b))
		}
	}
	if encodedSecret != "" {
		machineCtx := tailscale.BuildMachineContext(m.HostCtx.Host, m.HostCtx.ExePath)
		raw, err := intcrypto.Decrypt(encodedSecret, "", machineCtx)
		if err == nil {
			_ = json.Unmarshal([]byte(raw), &cleanupCfg)
		}
	}
	if strings.TrimSpace(cleanupCfg.APIKey) == "" {
		if cfgCleanup := m.resolveCleanupFromConfig(rec.PresetID); strings.TrimSpace(cfgCleanup.APIKey) != "" {
			cleanupCfg = cfgCleanup
		}
	}

	if cleanupCfg.DeviceDeleteEnabled && cleanupCfg.APIKey != "" && rec.DeviceID != "" {
		delCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		if err := tailscale.DeleteDevice(delCtx, cleanupCfg.APIKey, rec.DeviceID); err != nil {
			errs = append(errs, "delete device API: "+err.Error())
		}
		cancel()
	}

	if rec.Mode != model.LeaseModePermanent {
		if err := m.TS.Uninstall(ctx, preset); err != nil {
			errs = append(errs, "uninstall tailscale: "+err.Error())
		}
	}

	if len(errs) > 0 {
		rec.Status = model.LeaseStatusCleanupFailed
		rec.LastError = strings.Join(errs, "; ")
		rec.LastReconcileResult = "cleanup_failed"
		m.Logger.Error("lease %s cleanup failed: %s", rec.LeaseID, rec.LastError)
		return rec
	}
	rec.Status = model.LeaseStatusCleaned
	rec.LastError = ""
	rec.LastReconcileResult = "cleanup_ok"
	if strings.TrimSpace(rec.CredentialRef) != "" {
		_ = os.Remove(rec.CredentialRef)
	}
	_ = state.AppendAudit(m.Runtime.AuditPath, model.AuditEntry{
		LeaseID:    rec.LeaseID,
		Action:     "cleaned",
		PresetID:   rec.PresetID,
		Mode:       rec.Mode,
		Channel:    rec.Channel,
		DeviceName: rec.DeviceName,
		Host:       rec.Hostname,
		Message:    "lease cleanup completed",
	})
	m.Logger.Info("lease %s cleanup completed", rec.LeaseID)
	return rec
}

func shouldCleanup(rec model.LeaseRecord, currentBootID string, now time.Time) bool {
	switch rec.Mode {
	case model.LeaseModeSession:
		return rec.CreatedBootID != "" && rec.CreatedBootID != "unknown" && currentBootID != rec.CreatedBootID
	case model.LeaseModeTimed:
		return rec.ExpiresAt != nil && !now.Before(*rec.ExpiresAt)
	default:
		return false
	}
}

func (m *Manager) installAgent(ctx context.Context) error {
	agentPath, err := m.ensureLocalAgentBinary()
	if err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		return m.installWindowsAgent(ctx, agentPath)
	}
	return m.installLinuxAgent(ctx, agentPath)
}

func (m *Manager) installLinuxAgent(ctx context.Context, agentPath string) error {
	servicePath := "/etc/systemd/system/tailstick-agent.service"
	timerPath := "/etc/systemd/system/tailstick-agent.timer"

	service := linuxAgentServiceContent(agentPath, m.Runtime)
	timer := `[Unit]
Description=TailStick lease agent timer

[Timer]
OnBootSec=30s
OnUnitActiveSec=60s
Persistent=true
Unit=tailstick-agent.service

[Install]
WantedBy=timers.target
`
	if m.Runtime.DryRun {
		m.Logger.Info("[dry-run] would install linux systemd service at %s and timer at %s", servicePath, timerPath)
		return nil
	}
	if err := os.WriteFile(servicePath, []byte(service), 0o644); err != nil {
		return fmt.Errorf("write systemd service: %w", err)
	}
	if err := os.WriteFile(timerPath, []byte(timer), 0o644); err != nil {
		return fmt.Errorf("write systemd timer: %w", err)
	}
	for _, cmd := range linuxAgentInstallCommands() {
		if _, err := m.Runner.Run(ctx, cmd); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) installWindowsAgent(ctx context.Context, agentPath string) error {
	launcherPath := windowsAgentLauncherPath(agentPath)
	launcherBody := windowsAgentLauncherContent(agentPath, m.Runtime)
	if m.Runtime.DryRun {
		m.Logger.Info("[dry-run] would install windows agent launcher at %s", launcherPath)
	} else {
		if err := platform.EnsureParent(launcherPath); err != nil {
			return err
		}
		if err := os.WriteFile(launcherPath, []byte(launcherBody), 0o644); err != nil {
			return fmt.Errorf("write windows agent launcher: %w", err)
		}
	}

	taskCmd := windowsScheduledTaskCommand(launcherPath)
	if len(taskCmd) > 261 {
		return fmt.Errorf("windows scheduled task target exceeds schtasks /TR limit: %d", len(taskCmd))
	}
	commands := [][]string{
		{"schtasks", "/Create", "/TN", "TailStickAgent-Startup", "/SC", "ONSTART", "/TR", taskCmd, "/RL", "HIGHEST", "/F"},
		{"schtasks", "/Create", "/TN", "TailStickAgent-Periodic", "/SC", "MINUTE", "/MO", "1", "/TR", taskCmd, "/RL", "HIGHEST", "/F"},
		{"schtasks", "/Run", "/TN", "TailStickAgent-Startup"},
	}
	for _, cmd := range commands {
		if _, err := m.Runner.Run(ctx, cmd); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) uninstallAgent(ctx context.Context) error {
	var err error
	if runtime.GOOS == "windows" {
		err = m.uninstallWindowsAgent(ctx)
	} else {
		err = m.uninstallLinuxAgent(ctx)
	}
	removeErr := m.removeLocalAgentArtifacts(ctx)
	if err != nil {
		return err
	}
	if removeErr != nil {
		return removeErr
	}
	return nil
}

func (m *Manager) uninstallLinuxAgent(ctx context.Context) error {
	if m.Runtime.DryRun {
		m.Logger.Info("[dry-run] would uninstall linux tailstick agent service/timer")
		return nil
	}
	_, _ = m.Runner.Run(ctx, []string{"systemctl", "disable", "--now", "tailstick-agent.timer"})
	_, _ = m.Runner.Run(ctx, []string{"systemctl", "disable", "--now", "tailstick-agent.service"})
	_ = os.Remove("/etc/systemd/system/tailstick-agent.timer")
	_ = os.Remove("/etc/systemd/system/tailstick-agent.service")
	_, _ = m.Runner.Run(ctx, []string{"systemctl", "daemon-reload"})
	return nil
}

func (m *Manager) uninstallWindowsAgent(ctx context.Context) error {
	for _, task := range []string{"TailStickAgent", "TailStickAgent-Startup", "TailStickAgent-Periodic"} {
		_, _ = m.Runner.Run(ctx, []string{"schtasks", "/Delete", "/TN", task, "/F"})
	}
	return nil
}

func (m *Manager) ensureLocalAgentBinary() (string, error) {
	source := strings.TrimSpace(m.HostCtx.ExePath)
	target := platform.AgentBinaryPath()
	if source == "" {
		return "", fmt.Errorf("cannot install local agent binary: executable path is empty")
	}
	if m.Runtime.DryRun {
		m.Logger.Info("[dry-run] would copy local agent binary from %s to %s", source, target)
		return target, nil
	}
	if err := platform.EnsureParent(target); err != nil {
		return "", err
	}
	if filepath.Clean(source) == filepath.Clean(target) {
		if runtime.GOOS != "windows" {
			_ = os.Chmod(target, 0o755)
		}
		return target, nil
	}
	in, err := os.Open(source)
	if err != nil {
		return "", fmt.Errorf("open source binary: %w", err)
	}
	defer in.Close()

	mode := os.FileMode(0o755)
	if info, err := in.Stat(); err == nil {
		mode = info.Mode().Perm()
		if runtime.GOOS != "windows" && mode&0o111 == 0 {
			mode |= 0o755
		}
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return "", fmt.Errorf("open target binary: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return "", fmt.Errorf("copy agent binary: %w", err)
	}
	if err := out.Close(); err != nil {
		return "", fmt.Errorf("close target binary: %w", err)
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(target, mode|0o111)
	}
	return target, nil
}

func (m *Manager) removeLocalAgentArtifacts(ctx context.Context) error {
	targets := []string{platform.AgentBinaryPath()}
	if runtime.GOOS == "windows" {
		targets = append(targets, windowsAgentLauncherPath(platform.AgentBinaryPath()))
	}
	if m.Runtime.DryRun {
		m.Logger.Info("[dry-run] would remove local agent artifacts %s", strings.Join(targets, ", "))
		return nil
	}
	if runtime.GOOS != "windows" {
		err := os.Remove(targets[0])
		if err == nil || os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("remove local agent binary: %w", err)
	}

	if _, delayedErr := m.Runner.Run(ctx, windowsDelayedDeleteCommand(targets)); delayedErr != nil {
		return fmt.Errorf("schedule delayed local agent artifact delete: %w", delayedErr)
	}
	return nil
}

func windowsAgentLauncherPath(agentPath string) string {
	return filepath.Join(filepath.Dir(agentPath), "agent.cmd")
}

func windowsScheduledTaskCommand(launcherPath string) string {
	return fmt.Sprintf(`"%s"`, launcherPath)
}

func windowsAgentLauncherContent(agentPath string, rt Runtime) string {
	return strings.Join([]string{
		"@echo off",
		fmt.Sprintf(`"%s" agent --once --config "%s" --state "%s" --audit "%s" --log "%s"`, agentPath, rt.ConfigPath, rt.StatePath, rt.AuditPath, rt.LogPath),
		"",
	}, "\r\n")
}

func linuxAgentServiceContent(agentPath string, rt Runtime) string {
	return fmt.Sprintf(`[Unit]
Description=TailStick lease agent
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=%q agent --once --config %q --state %q --audit %q --log %q

[Install]
WantedBy=multi-user.target
`, agentPath, rt.ConfigPath, rt.StatePath, rt.AuditPath, rt.LogPath)
}

func linuxAgentInstallCommands() [][]string {
	return [][]string{
		{"systemctl", "daemon-reload"},
		{"systemctl", "enable", "tailstick-agent.timer"},
		{"systemctl", "start", "tailstick-agent.timer"},
	}
}

func windowsDelayedDeleteCommand(targets []string) []string {
	quotedTargets := make([]string, 0, len(targets))
	for _, target := range targets {
		quotedTargets = append(quotedTargets, fmt.Sprintf(`"%s"`, strings.ReplaceAll(target, `"`, `""`)))
	}
	cmdLine := "/c ping 127.0.0.1 -n 3 >NUL & del /f /q " + strings.Join(quotedTargets, " ")
	ps := fmt.Sprintf(`Start-Process -FilePath cmd.exe -ArgumentList '%s' -WindowStyle Hidden`, strings.ReplaceAll(cmdLine, `'`, `''`))
	return []string{"powershell", "-NoProfile", "-Command", ps}
}

func validateExitNode(preset model.Preset, value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	if !preset.AllowExitNodeSelection {
		return fmt.Errorf("preset %s does not allow exit node selection", preset.ID)
	}
	for _, approved := range preset.ApprovedExitNodes {
		if strings.EqualFold(strings.TrimSpace(approved), strings.TrimSpace(value)) {
			return nil
		}
	}
	return fmt.Errorf("exit node %q is not in preset approved list", value)
}

func buildDeviceName(mode model.LeaseMode, preset, host, leaseID, suffix string) string {
	modeToken := map[model.LeaseMode]string{
		model.LeaseModeSession:   "session",
		model.LeaseModeTimed:     "timed",
		model.LeaseModePermanent: "perm",
	}[mode]
	base := fmt.Sprintf("tsusb-%s-%s-%s-%s", modeToken, sanitizeNamePart(preset), sanitizeNamePart(host), sanitizeNamePart(leaseID))
	if s := sanitizeNamePart(suffix); s != "" {
		base += "-" + s
	}
	if len(base) > 63 {
		base = base[:63]
	}
	return strings.Trim(base, "-")
}

func sanitizeNamePart(in string) string {
	in = strings.TrimSpace(strings.ToLower(in))
	if in == "" {
		return ""
	}
	var out []rune
	for _, r := range in {
		switch {
		case r >= 'a' && r <= 'z':
			out = append(out, r)
		case r >= '0' && r <= '9':
			out = append(out, r)
		default:
			out = append(out, '-')
		}
	}
	return strings.Trim(strings.Join(strings.Fields(strings.ReplaceAll(string(out), "--", "-")), "-"), "-")
}

func newLeaseID() (string, error) {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return strings.ToLower(strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), "=")), nil
}

func hasActiveManagedLeases(st model.LocalState) bool {
	for _, rec := range st.Records {
		if rec.Mode == model.LeaseModePermanent {
			continue
		}
		if rec.Status != model.LeaseStatusCleaned {
			return true
		}
	}
	return false
}

func installSnapshotEmpty(inst model.Install) bool {
	return len(inst.LinuxStable) == 0 &&
		len(inst.LinuxLatest) == 0 &&
		len(inst.WindowsStable) == 0 &&
		len(inst.WindowsLatest) == 0 &&
		len(inst.LinuxUninstall) == 0 &&
		len(inst.WindowsUninstall) == 0
}

func defaultConfigPath(exePath string) string {
	exeDir := filepath.Dir(exePath)
	if exeDir == "" || exeDir == "." {
		return config.DefaultConfigFile
	}
	return filepath.Join(exeDir, config.DefaultConfigFile)
}

func (m *Manager) resolveCleanupFromConfig(presetID string) model.Cleanup {
	cfg, err := config.Load(m.Runtime.ConfigPath)
	if err != nil {
		return model.Cleanup{}
	}
	p, err := config.FindPreset(cfg, presetID)
	if err != nil {
		return model.Cleanup{}
	}
	p = config.ResolvePresetSecrets(p)
	return p.Cleanup
}

func (m *Manager) resolvePresetFromConfig(presetID string) model.Preset {
	cfg, err := config.Load(m.Runtime.ConfigPath)
	if err != nil {
		return model.Preset{}
	}
	p, err := config.FindPreset(cfg, presetID)
	if err != nil {
		return model.Preset{}
	}
	return config.ResolvePresetSecrets(p)
}

func (m *Manager) writeLeaseSecret(leaseID, encrypted string) (string, error) {
	if m.Runtime.DryRun {
		return "dry-run://secret/" + leaseID, nil
	}
	secretDir := platform.LocalSecretPath()
	if err := os.MkdirAll(secretDir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(secretDir, leaseID+".enc")
	if err := os.WriteFile(path, []byte(encrypted), 0o600); err != nil {
		return "", err
	}
	return path, nil
}
