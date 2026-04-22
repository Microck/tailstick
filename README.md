<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset=".github/assets/tailstick-logo-darkmode.svg">
    <source media="(prefers-color-scheme: light)" srcset=".github/assets/tailstick-logo-lightmode.svg">
    <img src=".github/assets/tailstick-logo-lightmode.svg" alt="tailstick" width="720">
  </picture>
</p>

<p align="center">
  <a href="https://github.com/Microck/tailstick/releases"><img src="https://img.shields.io/github/v/release/Microck/tailstick?display_name=tag&style=flat-square&label=release&color=000000" alt="release badge"></a>
  <a href="https://github.com/Microck/tailstick/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/Microck/tailstick/ci.yml?branch=main&style=flat-square&label=ci&color=000000" alt="ci badge"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-mit-000000?style=flat-square" alt="license badge"></a>
  <a href="https://goreportcard.com/report/github.com/Microck/tailstick"><img src="https://goreportcard.com/badge/github.com/Microck/tailstick?style=flat-square&color=000000" alt="go report card"></a>
</p>

<p align="center">
  USB-delivered Tailscale enrollment for authorized sysadmin workflows.
  One command surface for controlled onboarding, bounded lease modes, and automatic cleanup.
</p>

---

## Table of Contents

- [Why Tailstick](#why-tailstick)
- [Requirements](#requirements)
- [Quickstart](#quickstart)
- [Platform Support](#platform-support)
- [Command Surface](#command-surface)
- [Lease Modes](#lease-modes)
- [Environment Variables](#environment-variables)
- [Examples](#examples)
- [Building from Source](#building-from-source)
- [Documentation](#documentation)
- [License](#license)

---

## Why Tailstick

If you need to get a field machine onto your tailnet without building a backend service around the process, Tailstick gives you a controlled local workflow. It keeps the operator flow simple, supports **session**, **timed**, and **permanent** access modes, and ties cleanup to the same lease record instead of spreading state across ad hoc scripts.

**Key features:**

- **USB-delivered** — carry a single binary on a USB drive, no network infrastructure needed
- **Three lease modes** — session (clean on reboot), timed (auto-expiry), permanent (no auto-cleanup)
- **Automatic cleanup** — scheduled reconciliation agent removes leases when they expire
- **CLI and GUI** — direct command flow or local browser-based wizard
- **Password-gated presets** — restrict sensitive enrollment operations with operator passwords
- **Audit logging** — NDJSON audit trail for every enrollment and cleanup action
- **Cross-platform** — Linux (Debian/Ubuntu + systemd) and Windows support

## Requirements

- **Elevated privileges** are required for enrollment and cleanup (`sudo` on Linux, Administrator on Windows).
- **Linux**: Debian or Ubuntu with `systemd`. Other distributions are not supported in the current release.
- **Windows**: Standard Windows with Scheduled Tasks available.
- **Go 1.22+** is required for building from source.
- **Tailscale auth keys** — an auth key, ephemeral auth key, and API key (see [Environment Variables](#environment-variables)).

## Quickstart

1. Go to [releases](https://github.com/Microck/tailstick/releases/latest) and download the binary for your platform. Use the CLI for a direct command flow, or the GUI for the local browser wizard. Binaries are available for `linux/amd64`, `linux/arm64`, `windows/amd64`, and `windows/arm64`.
2. Place the binary in a working folder alongside a `tailstick.config.json` file.
3. Write the config by hand from [`configs/tailstick.config.example.json`](configs/tailstick.config.example.json), or use the optional [preset maker](https://tailstick.micr.dev/) for a faster setup.
4. Ensure the secrets your presets require are available. Typically this means a Tailscale auth key, ephemeral auth key, and API key for cleanup. For password-gated operator use, also set an operator password.
5. Supply secrets via environment variables: `TAILSTICK_AUTH_KEY`, `TAILSTICK_EPHEMERAL_AUTH_KEY`, `TAILSTICK_API_KEY`, and `TAILSTICK_OPERATOR_PASSWORD`.
6. Run the binary once the config and secrets are in place.

For normal operator use, use the release binaries. Building from source is only for development work.

## Platform Support

| Platform | CLI | GUI | Full Lease Runtime |
| --- | --- | --- | --- |
| Linux (Debian/Ubuntu + `systemd`) | ✅ Supported | ✅ Supported | ✅ Supported |
| Windows | ✅ Supported | ✅ Supported | ✅ Supported |

Release binaries are built for `linux/amd64`, `linux/arm64`, `windows/amd64`, and `windows/arm64`.

## Command Surface

| Command | Purpose |
| --- | --- |
| `tailstick run` | Enroll machine and create lease |
| `tailstick agent` | Run reconciliation loop (`--once` for a single pass) |
| `tailstick cleanup` | Force cleanup for one lease ID |
| `tailstick version` | Print version |

### `run` Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--preset` | `""` | Preset ID from config |
| `--mode` | `session` | Lease mode: `session`, `timed`, or `permanent` |
| `--channel` | `stable` | Install channel: `stable` or `latest` |
| `--days` | `3` | Timed lease duration in days |
| `--custom-days` | `0` | Custom lease days (1–30, overrides `--days`) |
| `--suffix` | `""` | Optional device name suffix |
| `--exit-node` | `""` | Optional approved exit node |
| `--allow-existing` | `false` | Allow existing Tailscale install |
| `--non-interactive` | `false` | Non-interactive mode |
| `--password` | `""` | Operator password (or set `TAILSTICK_OPERATOR_PASSWORD`) |
| `--config` | `tailstick.config.json` | Config file path |
| `--state` | Platform default | State file path |
| `--audit` | `logs/tailstick-audit.ndjson` | Audit log path |
| `--log` | Platform default | Log file path |
| `--dry-run` | `false` | Print commands without executing |

### `agent` Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--once` | `false` | Run one reconciliation pass and exit |
| `--interval` | `1m` | Reconciliation interval |

The agent also accepts `--config`, `--state`, `--audit`, `--log`, and `--dry-run`.

### `cleanup` Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--lease-id` | `""` | Lease ID to clean (required) |

The cleanup command also accepts `--config`, `--state`, `--audit`, `--log`, and `--dry-run`.

### GUI Flags

The GUI binary (`tailstick-linux-gui` / `tailstick-windows-gui`) starts a local browser-based setup wizard. In addition to the shared flags (`--config`, `--state`, `--audit`, `--log`, `--dry-run`):

| Flag | Default | Description |
| --- | --- | --- |
| `--host` | `127.0.0.1` | Bind host |
| `--port` | `0` | Bind port (`0` picks an ephemeral port) |
| `--open-browser` | `true` | Open browser automatically |

## Lease Modes

| Mode | Behavior |
| --- | --- |
| `session` | Cleaned after reboot/shutdown detection |
| `timed` | Cleaned when expiry is reached (set duration with `--days` or `--custom-days`) |
| `permanent` | Leaves normal Tailscale install with no automatic cleanup |

For lease modes, Tailstick installs local scheduling:

- **Linux**: `tailstick-agent.service` + `tailstick-agent.timer`
- **Windows**: `TailStickAgent-Startup` + `TailStickAgent-Periodic`

The scheduled command runs from a machine-local binary copy:

- **Linux**: `/var/lib/tailstick/tailstick-agent`
- **Windows**: `%ProgramData%\TailStick\tailstick-agent.exe`

## Environment Variables

Secrets are resolved from environment variables. The config file supports `authKeyEnv`, `ephemeralAuthKeyEnv`, and `apiKeyEnv` fields that name the env var to read.

| Variable | Purpose |
| --- | --- |
| `TAILSTICK_AUTH_KEY` | Tailscale auth key |
| `TAILSTICK_EPHEMERAL_AUTH_KEY` | Tailscale ephemeral auth key |
| `TAILSTICK_API_KEY` | Tailscale API key for cleanup |
| `TAILSTICK_OPERATOR_PASSWORD` | Operator password for gated presets |

See [`.env.example`](.env.example) for a template.

## Examples

Timed lease with custom duration:

```bash
./tailstick-linux-cli run \
  --preset ops-readonly \
  --mode timed \
  --custom-days 14 \
  --channel stable \
  --allow-existing
```

Session lease with approved exit node:

```bash
./tailstick-linux-cli run \
  --preset ops-readonly \
  --mode session \
  --exit-node 100.64.0.5 \
  --channel latest
```

Manual cleanup:

```bash
./tailstick-linux-cli cleanup --lease-id <lease-id>
```

Single agent pass:

```bash
./tailstick-linux-cli agent --once
```

## Building from Source

```bash
go build ./cmd/tailstick-linux-cli
go build ./cmd/tailstick-linux-gui
```

Cross-compile all targets:

```bash
make build-all
```

Run tests:

```bash
go test ./...
```

See the [`Makefile`](Makefile) for additional targets (`fmt`, `vet`, `sandbox-linux`, `icons`).

## Documentation

| Document | Description |
| --- | --- |
| [Architecture](docs/architecture.md) | System design and component overview |
| [Configuration](docs/configuration.md) | Config file format and preset options |
| [Operations](docs/operations.md) | Deployment, monitoring, and troubleshooting |
| [Testing](docs/testing.md) | Test strategy and running the test suite |
| [Release Runbook](docs/release-runbook.md) | Steps for cutting a new release |
| [Contributing](CONTRIBUTING.md) | How to contribute to Tailstick |

## License

[MIT License](LICENSE)
