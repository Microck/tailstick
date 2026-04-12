package model

import "time"

type LeaseMode string

const (
	LeaseModeSession   LeaseMode = "session"
	LeaseModeTimed     LeaseMode = "timed"
	LeaseModePermanent LeaseMode = "permanent"
)

type Channel string

const (
	ChannelStable Channel = "stable"
	ChannelLatest Channel = "latest"
)

type LeaseStatus string

const (
	LeaseStatusActive        LeaseStatus = "active"
	LeaseStatusCleanupQueued LeaseStatus = "cleanup_queued"
	LeaseStatusCleanupFailed LeaseStatus = "cleanup_failed"
	LeaseStatusCleaned       LeaseStatus = "cleaned"
)

type Config struct {
	StableVersion       string   `json:"stableVersion"`
	DefaultPreset       string   `json:"defaultPreset"`
	OperatorPassword    string   `json:"operatorPassword"`
	OperatorPasswordEnv string   `json:"operatorPasswordEnv"`
	Presets             []Preset `json:"presets"`
}

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

type Install struct {
	LinuxStable      []string `json:"linuxStable"`
	LinuxLatest      []string `json:"linuxLatest"`
	WindowsStable    []string `json:"windowsStable"`
	WindowsLatest    []string `json:"windowsLatest"`
	LinuxUninstall   []string `json:"linuxUninstall"`
	WindowsUninstall []string `json:"windowsUninstall"`
}

type Cleanup struct {
	Tailnet             string `json:"tailnet"`
	APIKey              string `json:"apiKey"`
	APIKeyEnv           string `json:"apiKeyEnv"`
	DeviceDeleteEnabled bool   `json:"deviceDeleteEnabled"`
}

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

type LocalState struct {
	SchemaVersion int           `json:"schemaVersion"`
	UpdatedAt     time.Time     `json:"updatedAt"`
	Records       []LeaseRecord `json:"records"`
}

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

type TailscaleSelf struct {
	ID      string `json:"ID"`
	DNSName string `json:"DNSName"`
}

type TailscaleStatus struct {
	Self TailscaleSelf `json:"Self"`
}
