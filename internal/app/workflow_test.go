package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	intcrypto "github.com/tailstick/tailstick/internal/crypto"
	"github.com/tailstick/tailstick/internal/logging"
	"github.com/tailstick/tailstick/internal/model"
	"github.com/tailstick/tailstick/internal/platform"
	"github.com/tailstick/tailstick/internal/state"
	"github.com/tailstick/tailstick/internal/tailscale"
)

func TestBuildDeviceName(t *testing.T) {
	got := buildDeviceName(model.LeaseModeTimed, "ops-read", "finance-laptop", "abc123", "night")
	want := "tsusb-timed-ops-read-finance-laptop-abc123-night"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestShouldCleanupSessionOnBootChange(t *testing.T) {
	rec := model.LeaseRecord{
		Mode:          model.LeaseModeSession,
		CreatedBootID: "boot-a",
		Status:        model.LeaseStatusActive,
	}
	if !shouldCleanup(rec, "boot-b", time.Now().UTC()) {
		t.Fatalf("expected session lease to cleanup after boot id change")
	}
}

func TestShouldCleanupTimedOnExpiry(t *testing.T) {
	exp := time.Now().UTC().Add(-1 * time.Minute)
	rec := model.LeaseRecord{
		Mode:      model.LeaseModeTimed,
		ExpiresAt: &exp,
		Status:    model.LeaseStatusActive,
	}
	if !shouldCleanup(rec, "same-boot", time.Now().UTC()) {
		t.Fatalf("expected timed lease to cleanup on expiry")
	}
}

func TestCleanupRecordConsumesCredentialFileAndCleansLease(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake tailscale helper is only implemented for unix-like platforms")
	}

	mgr, cmdLogPath, _, cleanup := newWorkflowTestManager(t, false)
	rec := model.LeaseRecord{
		LeaseID:         "lease-1",
		PresetID:        "ops-read",
		Mode:            model.LeaseModeTimed,
		Channel:         model.ChannelStable,
		Hostname:        mgr.HostCtx.Host,
		DeviceName:      "tsusb-timed-ops-read-dev-box-lease-1",
		CredentialRef:   writeEncryptedCleanupSecret(t, cleanup, mgr.HostCtx),
		InstallSnapshot: model.Install{LinuxUninstall: []string{"tailscale", "uninstall"}},
	}

	updated := mgr.cleanupRecord(context.Background(), rec)

	if updated.Status != model.LeaseStatusCleaned {
		t.Fatalf("got status %q want %q", updated.Status, model.LeaseStatusCleaned)
	}
	if updated.LastError != "" {
		t.Fatalf("expected no cleanup error, got %q", updated.LastError)
	}
	if updated.LastReconcileResult != "cleanup_ok" {
		t.Fatalf("got reconcile result %q want cleanup_ok", updated.LastReconcileResult)
	}
	if updated.LastReconciledAt == nil {
		t.Fatal("expected cleanup to stamp last reconciled time")
	}
	if _, err := os.Stat(rec.CredentialRef); !os.IsNotExist(err) {
		t.Fatalf("expected credential file to be removed, stat err=%v", err)
	}

	gotCommands := readCommandLog(t, cmdLogPath)
	wantCommands := []string{
		"down",
		"logout",
		"uninstall",
	}
	if !equalStringSlices(gotCommands, wantCommands) {
		t.Fatalf("got commands %v want %v", gotCommands, wantCommands)
	}
}

func TestForceCleanupRequiresLeaseID(t *testing.T) {
	mgr, _, _, _ := newWorkflowTestManager(t, true)

	err := mgr.ForceCleanup(context.Background(), "   ")
	if err == nil || !strings.Contains(err.Error(), "lease id is required") {
		t.Fatalf("got err %v want lease id required", err)
	}
}

func TestForceCleanupReturnsNotFoundForUnknownLease(t *testing.T) {
	mgr, _, statePath, _ := newWorkflowTestManager(t, true)
	if err := state.Save(statePath, model.LocalState{Records: []model.LeaseRecord{}}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	err := mgr.ForceCleanup(context.Background(), "missing-lease")
	if err == nil || !strings.Contains(err.Error(), "lease missing-lease not found") {
		t.Fatalf("got err %v want not found", err)
	}
}

func TestForceCleanupCleansLeaseAndPersistsState(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake tailscale helper is only implemented for unix-like platforms")
	}

	mgr, cmdLogPath, statePath, cleanup := newWorkflowTestManager(t, false)
	rec := model.LeaseRecord{
		LeaseID:         "lease-1",
		PresetID:        "ops-read",
		Mode:            model.LeaseModeTimed,
		Channel:         model.ChannelStable,
		Hostname:        mgr.HostCtx.Host,
		DeviceName:      "tsusb-timed-ops-read-dev-box-lease-1",
		CredentialRef:   writeEncryptedCleanupSecret(t, cleanup, mgr.HostCtx),
		InstallSnapshot: model.Install{LinuxUninstall: []string{"tailscale", "uninstall"}},
		Status:          model.LeaseStatusActive,
	}
	if err := state.Save(statePath, model.LocalState{Records: []model.LeaseRecord{rec}}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	if err := mgr.ForceCleanup(context.Background(), rec.LeaseID); err != nil {
		t.Fatalf("force cleanup: %v", err)
	}

	st, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if len(st.Records) != 1 {
		t.Fatalf("got %d records want 1", len(st.Records))
	}
	if st.Records[0].Status != model.LeaseStatusCleaned {
		t.Fatalf("got status %q want %q", st.Records[0].Status, model.LeaseStatusCleaned)
	}

	gotCommands := readCommandLog(t, cmdLogPath)
	wantCommands := []string{
		"down",
		"logout",
		"uninstall",
	}
	if !equalStringSlices(gotCommands, wantCommands) {
		t.Fatalf("got commands %v want %v", gotCommands, wantCommands)
	}
}

func TestAgentOnceSkipsCleanedRecordsWithoutSavingState(t *testing.T) {
	mgr, _, statePath, _ := newWorkflowTestManager(t, true)
	rec := model.LeaseRecord{
		LeaseID: "lease-cleaned",
		Mode:    model.LeaseModeTimed,
		Status:  model.LeaseStatusCleaned,
	}
	if err := state.Save(statePath, model.LocalState{Records: []model.LeaseRecord{rec}}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	before, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read state before agent run: %v", err)
	}

	if err := mgr.AgentOnce(context.Background()); err != nil {
		t.Fatalf("agent once: %v", err)
	}

	after, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read state after agent run: %v", err)
	}
	if !bytes.Equal(after, before) {
		t.Fatal("expected cleaned-only state to remain unchanged")
	}

	logBody := readFile(t, mgr.Runtime.LogPath)
	if !strings.Contains(logBody, "agent self-removal completed: no active managed leases") {
		t.Fatalf("expected self-removal log, got %q", logBody)
	}
}

func TestAgentOnceCleansExpiredLeaseAndPersistsState(t *testing.T) {
	mgr, _, statePath, _ := newWorkflowTestManager(t, true)
	expired := time.Now().UTC().Add(-1 * time.Minute)
	rec := model.LeaseRecord{
		LeaseID:   "lease-expired",
		Mode:      model.LeaseModeTimed,
		Status:    model.LeaseStatusActive,
		ExpiresAt: &expired,
	}
	if err := state.Save(statePath, model.LocalState{Records: []model.LeaseRecord{rec}}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	if err := mgr.AgentOnce(context.Background()); err != nil {
		t.Fatalf("agent once: %v", err)
	}

	st, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if len(st.Records) != 1 {
		t.Fatalf("got %d records want 1", len(st.Records))
	}
	got := st.Records[0]
	if got.Status != model.LeaseStatusCleaned {
		t.Fatalf("got status %q want %q", got.Status, model.LeaseStatusCleaned)
	}
	if got.LastReconcileResult != "cleanup_ok" {
		t.Fatalf("got reconcile result %q want cleanup_ok", got.LastReconcileResult)
	}
	if got.LastReconciledAt == nil {
		t.Fatal("expected cleanup to stamp last reconciled time")
	}
}

func TestAgentRunStopsAfterSelfRemovalWhenNoActiveLeasesRemain(t *testing.T) {
	mgr, _, statePath, _ := newWorkflowTestManager(t, true)
	rec := model.LeaseRecord{
		LeaseID: "lease-cleaned",
		Mode:    model.LeaseModeTimed,
		Status:  model.LeaseStatusCleaned,
	}
	if err := state.Save(statePath, model.LocalState{Records: []model.LeaseRecord{rec}}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- mgr.AgentRun(ctx, 10*time.Millisecond)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("agent run returned err: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("agent run did not stop after self-removal")
	}

	logBody := readFile(t, mgr.Runtime.LogPath)
	if !strings.Contains(logBody, "tailstick agent stopping: no active managed leases remain") {
		t.Fatalf("expected stop log, got %q", logBody)
	}
}

func TestAgentOnceMarksActiveLeaseAsNoAction(t *testing.T) {
	mgr, _, statePath, _ := newWorkflowTestManager(t, true)
	expiresAt := time.Now().UTC().Add(2 * time.Hour)
	rec := model.LeaseRecord{
		LeaseID:   "lease-active",
		Mode:      model.LeaseModeTimed,
		Status:    model.LeaseStatusActive,
		ExpiresAt: &expiresAt,
	}
	if err := state.Save(statePath, model.LocalState{Records: []model.LeaseRecord{rec}}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	if err := mgr.AgentOnce(context.Background()); err != nil {
		t.Fatalf("agent once: %v", err)
	}

	st, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	got := st.Records[0]
	if got.Status != model.LeaseStatusActive {
		t.Fatalf("got status %q want %q", got.Status, model.LeaseStatusActive)
	}
	if got.LastReconcileResult != "no_action" {
		t.Fatalf("got reconcile result %q want no_action", got.LastReconcileResult)
	}
	if got.LastReconciledAt == nil {
		t.Fatal("expected reconcile timestamp for no-action path")
	}
}

func TestWindowsScheduledTaskCommandUsesShortLauncher(t *testing.T) {
	root := t.TempDir()
	agentPath := filepath.Join(root, "TailStick", "tailstick-agent.exe")
	rt := Runtime{
		ConfigPath: filepath.Join(root, strings.Repeat("config-segment-", 8), "tailstick.config.json"),
		StatePath:  filepath.Join(root, strings.Repeat("state-segment-", 8), "state.json"),
		AuditPath:  filepath.Join(root, strings.Repeat("audit-segment-", 8), "audit.ndjson"),
		LogPath:    filepath.Join(root, strings.Repeat("log-segment-", 8), "tailstick.log"),
	}

	launcherPath := windowsAgentLauncherPath(agentPath)
	taskCmd := windowsScheduledTaskCommand(launcherPath)
	if len(taskCmd) > 261 {
		t.Fatalf("task command length = %d, want <= 261", len(taskCmd))
	}
	if !strings.HasSuffix(launcherPath, filepath.Join("TailStick", "agent.cmd")) {
		t.Fatalf("launcher path %q should live beside the agent binary", launcherPath)
	}

	launcherBody := windowsAgentLauncherContent(agentPath, rt)
	for _, want := range []string{agentPath, rt.ConfigPath, rt.StatePath, rt.AuditPath, rt.LogPath} {
		if !strings.Contains(launcherBody, want) {
			t.Fatalf("launcher body missing %q", want)
		}
	}
}

func TestLinuxAgentInstallCommandsStartOnlyTheTimer(t *testing.T) {
	got := linuxAgentInstallCommands()
	want := [][]string{
		{"systemctl", "daemon-reload"},
		{"systemctl", "enable", "tailstick-agent.timer"},
		{"systemctl", "start", "tailstick-agent.timer"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d commands want %d", len(got), len(want))
	}
	for i := range want {
		if !equalStringSlices(got[i], want[i]) {
			t.Fatalf("command %d = %v want %v", i, got[i], want[i])
		}
	}
	for _, cmd := range got {
		if len(cmd) >= 3 && cmd[0] == "systemctl" && cmd[1] == "start" && cmd[2] == "tailstick-agent.service" {
			t.Fatalf("install sequence must not start the oneshot service directly: %v", cmd)
		}
	}
}

func TestWindowsDelayedDeleteCommandUsesDetachedProcess(t *testing.T) {
	targets := []string{
		`C:\ProgramData\TailStick\tailstick-agent.exe`,
		`C:\ProgramData\TailStick\agent.cmd`,
	}

	got := windowsDelayedDeleteCommand(targets)
	if len(got) != 4 {
		t.Fatalf("got %d command parts want 4", len(got))
	}
	if got[0] != "powershell" || got[1] != "-NoProfile" || got[2] != "-Command" {
		t.Fatalf("unexpected command prefix: %v", got[:3])
	}
	if !strings.Contains(got[3], "Start-Process -FilePath cmd.exe") {
		t.Fatalf("expected detached Start-Process launcher, got %q", got[3])
	}
	for _, target := range targets {
		if !strings.Contains(got[3], target) {
			t.Fatalf("cleanup command missing target %q", target)
		}
	}
	if strings.Contains(got[3], "/B") {
		t.Fatalf("cleanup command should not use cmd start /B: %q", got[3])
	}
}

func newWorkflowTestManager(t *testing.T, dryRun bool) (*Manager, string, string, model.Cleanup) {
	t.Helper()

	root := t.TempDir()
	statePath := filepath.Join(root, "state.json")
	auditPath := filepath.Join(root, "audit.ndjson")
	logPath := filepath.Join(root, "tailstick.log")
	cmdLogPath := filepath.Join(root, "tailscale-commands.log")
	if !dryRun {
		installFakeTailscale(t, root, cmdLogPath)
	}

	logger, err := logging.New(logPath)
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	t.Cleanup(func() {
		_ = logger.Close()
	})

	runner := platform.Runner{DryRun: dryRun}
	mgr := &Manager{
		Runtime: Runtime{
			ConfigPath: filepath.Join(root, "tailstick.config.json"),
			StatePath:  statePath,
			AuditPath:  auditPath,
			LogPath:    logPath,
			DryRun:     dryRun,
		},
		Logger: logger,
		Runner: runner,
		TS:     tailscale.Client{Runner: runner},
		HostCtx: platform.Context{
			Host:    "dev-box",
			BootID:  "boot-b",
			ExePath: filepath.Join(root, "tailstick"),
		},
	}
	cleanup := model.Cleanup{
		Tailnet:             "example.ts.net",
		DeviceDeleteEnabled: false,
	}
	return mgr, cmdLogPath, statePath, cleanup
}

func installFakeTailscale(t *testing.T, dir, logPath string) {
	t.Helper()

	scriptPath := filepath.Join(dir, "tailscale")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
printf '%%s\n' "$*" >> %q
case "${1:-}" in
  version)
    echo "1.0.0"
    ;;
  status)
    echo '{"Self":{"ID":"device-123","DNSName":"node.example.ts.net","HostName":"dev-box"}}'
    ;;
esac
`, logPath)
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tailscale: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func writeEncryptedCleanupSecret(t *testing.T, cleanup model.Cleanup, hostCtx platform.Context) string {
	t.Helper()

	payload, err := json.Marshal(cleanup)
	if err != nil {
		t.Fatalf("marshal cleanup secret: %v", err)
	}
	enc, err := intcrypto.Encrypt(string(payload), "", tailscale.BuildMachineContext(hostCtx.Host, hostCtx.ExePath))
	if err != nil {
		t.Fatalf("encrypt cleanup secret: %v", err)
	}
	path := filepath.Join(t.TempDir(), "cleanup.enc")
	if err := os.WriteFile(path, []byte(enc), 0o600); err != nil {
		t.Fatalf("write cleanup secret: %v", err)
	}
	return path
}

func readCommandLog(t *testing.T, path string) []string {
	t.Helper()

	body := strings.TrimSpace(readFile(t, path))
	if body == "" {
		return nil
	}
	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, strings.TrimSpace(line))
	}
	return out
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	return string(b)
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
