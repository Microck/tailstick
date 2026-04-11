// Package model defines the core domain types for tailstick, including lease
// configurations, presets, state records, and Tailscale API response shapes.
package model

import "time"

// LeaseMode represents the duration policy for a tailscale enrollment lease.
type LeaseMode string

const (
	LeaseModeSession   LeaseMode = "session"
	LeaseModeTimed     LeaseMode = "timed"
	LeaseModePermanent LeaseMode = "permanent"
)

// Channel specifies the tailscale installation release channel.
type Channel string

const (
	ChannelStable Channel = "stable"
	ChannelLatest Channel = "latest"
)

// LeaseStatus tracks the lifecycle phase of a lease record.
type LeaseStatus string

const (
	LeaseStatusActive        LeaseStatus = "active"
	LeaseStatusCleanupQueued LeaseStatus = "cleanup_queued"
	LeaseStatusCleanupFailed LeaseStatus = "cleanup_failed"
	LeaseStatusCleaned       LeaseStatus = "cleaned"
)

// Config is the top-level tailstick configuration loaded from tailstick.config.json.
type Config struct {
	StableVersion       string   `json:"stableVersion"`
	DefaultPreset       string   `json:"defaultPreset"`
	OperatorPassword    string   `json:"operatorPassword"`
	OperatorPasswordEnv string   `json:"operatorPasswordEnv"`
	Presets             []Preset `json:"presets"`
}

// Preset defines a named enrollment profile with auth keys, tags, and install/cleanup commands.
type Preset struct {
	ID                     string   `json:"id"`
	Description            string   `json:"description"`
	AuthKey                string   `json:"authKey"`
	AuthKeyEnv             string   `json:"authKeyEnv"`
	EphemeralAuthKey       string   `json:"ephemeralAuthKey"`
	EphemeralAuthKeyEnv    string   `json:"ephemeralAuthKeyEnv"`
	Tags                   []string `json:"tags"`
	AcceptRoutes           bool     `json:"acceptRoutes"`
	AllowExitNodeSelection bool     `json:"allowExitNodeSelection"`
	ApprovedExitNodes      []string `json:"approvedExitNodes"`
	StableVersionOverride  string   `json:"stableVersionOverride"`
	Install                Install  `json:"install"`
	Cleanup                Cleanup  `json:"cleanup"`
}

// Install holds platform-specific install and uninstall command templates for tailscale.
type Install struct {
	LinuxStable      []string `json:"linuxStable"`
	LinuxLatest      []string `json:"linuxLatest"`
	WindowsStable    []string `json:"windowsStable"`
	WindowsLatest    []string `json:"windowsLatest"`
	LinuxUninstall   []string `json:"linuxUninstall"`
	WindowsUninstall []string `json:"windowsUninstall"`
}

// Cleanup configures post-lease device deletion via the Tailscale API.
type Cleanup struct {
	Tailnet             string `json:"tailnet"`
	APIKey              string `json:"apiKey"`
	APIKeyEnv           string `json:"apiKeyEnv"`
	DeviceDeleteEnabled bool   `json:"deviceDeleteEnabled"`
}

// RuntimeOptions holds the resolved operator inputs for a single enrollment invocation.
type RuntimeOptions struct {
	PresetID       string
	Mode           LeaseMode
	Channel        Channel
	Days           int
	CustomDays     int
	DeviceSuffix   string
	ExitNode       string
	AllowExisting  bool
	NonInteractive bool
	Password       string
}

// LeaseRecord is the persistent state of a single enrollment lease, stored in state.json.
type LeaseRecord struct {
	LeaseID             string      `json:"leaseId"`
	PresetID            string      `json:"presetId"`
	Mode                LeaseMode   `json:"mode"`
	Channel             Channel     `json:"channel"`
	DurationDays        int         `json:"durationDays"`
	Hostname            string      `json:"hostname"`
	DeviceName          string      `json:"deviceName"`
	CreatedAt           time.Time   `json:"createdAt"`
	ExpiresAt           *time.Time  `json:"expiresAt,omitempty"`
	CreatedBootID       string      `json:"createdBootId"`
	Status              LeaseStatus `json:"status"`
	LastError           string      `json:"lastError,omitempty"`
	LastReconcileResult string      `json:"lastReconcileResult,omitempty"`
	LastReconciledAt    *time.Time  `json:"lastReconciledAt,omitempty"`
	DeviceID            string      `json:"deviceId,omitempty"`
	NodeName            string      `json:"nodeName,omitempty"`
	InstallSnapshot     Install     `json:"installSnapshot"`
	PresetCleanup       Cleanup     `json:"presetCleanup"`
	CredentialRef       string      `json:"credentialRef,omitempty"`
	EncryptedSecret     string      `json:"encryptedSecret,omitempty"`
}

// LocalState is the on-disk state file containing all lease records.
type LocalState struct {
	SchemaVersion int           `json:"schemaVersion"`
	UpdatedAt     time.Time     `json:"updatedAt"`
	Records       []LeaseRecord `json:"records"`
}

// AuditEntry is a single line in the audit log, recording a lease lifecycle event.
type AuditEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	LeaseID    string    `json:"leaseId"`
	Action     string    `json:"action"`
	PresetID   string    `json:"presetId"`
	Mode       LeaseMode `json:"mode"`
	Channel    Channel   `json:"channel"`
	DeviceName string    `json:"deviceName"`
	Host       string    `json:"host"`
	Message    string    `json:"message"`
}

// TailscaleSelf contains the identity fields from the tailscale status API.
type TailscaleSelf struct {
	ID       string `json:"ID"`
	DNSName  string `json:"DNSName"`
	HostName string `json:"HostName"`
}

// TailscaleStatus is the parsed output of "tailscale status --json".
type TailscaleStatus struct {
	Self TailscaleSelf `json:"Self"`
}
