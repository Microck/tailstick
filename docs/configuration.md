# Configuration Reference

TailStick config is JSON.

Example: [configs/tailstick.config.example.json](/home/ubuntu/workspace/tailscale-usb/configs/tailstick.config.example.json)

## Top-Level

- `stableVersion`: default pinned stable version used when channel is `stable`
- `defaultPreset`: preset selected when none is passed
- `operatorPassword`: optional enroll-time password gate value
- `operatorPasswordEnv`: optional env var name for enroll-time password gate
- `presets`: array of enrollment profiles

## Preset Fields

- `id`: unique preset id
- `description`: human-readable summary
- `authKey`: reusable enrollment key (plaintext optional)
- `authKeyEnv`: environment variable that provides auth key
- `ephemeralAuthKey`: optional key for session mode (plaintext optional)
- `ephemeralAuthKeyEnv`: environment variable for session-mode ephemeral key
- `tags`: tags applied on `tailscale up --advertise-tags`
- `acceptRoutes`: if true, includes `--accept-routes`
- `allowExitNodeSelection`: enables runtime `--exit-node`
- `approvedExitNodes`: allowed runtime exit-node values
- `stableVersionOverride`: optional profile-specific stable version label
- `install`: platform command overrides
- `cleanup`: delete-device control-plane settings

## Install Commands

Each command is an argv array, for example:

```json
["bash", "-lc", "curl -fsSL https://tailscale.com/install.sh | sh"]
```

Fields:

- `linuxStable`
- `linuxLatest`
- `windowsStable`
- `windowsLatest`
- `linuxUninstall`
- `windowsUninstall`

If omitted, TailStick applies internal defaults.

When channel is `stable`, TailStick enforces pinning via:

```bash
tailscale update --yes --version <stableVersion>
```

## Cleanup Block

- `tailnet`: informational tailnet label for operator context
- `apiKey`: Tailscale API key used for `DELETE /api/v2/device/{device_id}` (plaintext optional)
- `apiKeyEnv`: environment variable that provides API key
- `deviceDeleteEnabled`: enables API deletion during cleanup

If `deviceDeleteEnabled=false` or no API key is set, TailStick still performs local cleanup (`tailscale down`, `tailscale logout`, uninstall where applicable) but cannot force control-plane deletion.

Cleanup credentials are stored machine-locally in encrypted form. Encryption is machine-bound so unattended cleanup can still decrypt without operator return.

## Environment Expansion

TailStick expands `${ENV_VAR}` placeholders in config values before parsing JSON.

Example:

```json
{
  "authKey": "${TAILSTICK_AUTH_KEY}"
}
```
