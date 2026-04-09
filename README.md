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
</p>

---

`tailstick` is a usb-delivered tailscale enrollment tool for authorized sysadmin workflows on windows and linux. it gives operators one command surface for controlled onboarding, bounded lease modes, and automatic cleanup that can continue after the usb is unplugged.

[documentation](docs/) | [github](https://github.com/Microck/tailstick)

## why

if you need temporary or permanent tailscale access on field machines without standing up a backend service, `tailstick` gives you a practical path.

- explicit operator launch with cli and gui entrypoints
- deterministic lease records and machine-local audit surfaces
- session, timed, and permanent modes with shared lifecycle logic
- post-usb cleanup continuity through a machine-local agent binary
- stable vs latest install channel control with explicit operator choice

## quickstart

tailstick currently targets:

- windows (administrator shell required)
- debian/ubuntu linux with `systemd` (root required)

### 1. download the latest release

Download the archive for the machine you are using from [GitHub Releases](https://github.com/Microck/tailstick/releases/latest).

linux amd64:

```bash
curl -L -o tailstick-linux-amd64.tar.gz https://github.com/Microck/tailstick/releases/latest/download/tailstick-linux-amd64.tar.gz
tar -xzf tailstick-linux-amd64.tar.gz
```

linux arm64:

```bash
curl -L -o tailstick-linux-arm64.tar.gz https://github.com/Microck/tailstick/releases/latest/download/tailstick-linux-arm64.tar.gz
tar -xzf tailstick-linux-arm64.tar.gz
```

windows amd64 (powershell):

```powershell
Invoke-WebRequest -Uri https://github.com/Microck/tailstick/releases/latest/download/tailstick-windows-amd64.tar.gz -OutFile tailstick-windows-amd64.tar.gz
tar -xzf tailstick-windows-amd64.tar.gz
```

windows arm64 (powershell):

```powershell
Invoke-WebRequest -Uri https://github.com/Microck/tailstick/releases/latest/download/tailstick-windows-arm64.tar.gz -OutFile tailstick-windows-arm64.tar.gz
tar -xzf tailstick-windows-arm64.tar.gz
```

### 2. place your config next to the binaries

Create `tailstick.config.json` next to the extracted CLI and GUI binaries.

If you are preparing it from this repo:

```bash
cp configs/tailstick.config.example.json tailstick.config.json
```

### 3. provide runtime secrets

```bash
export TAILSTICK_AUTH_KEY='tskey-auth-...'
export TAILSTICK_EPHEMERAL_AUTH_KEY='tskey-auth-...'
export TAILSTICK_API_KEY='tskey-api-...'
# optional, only when operator password gating is enabled
export TAILSTICK_OPERATOR_PASSWORD='choose-a-strong-password'
```

### 4. create a lease

Run a timed lease:

```bash
./tailstick-linux-cli run   --preset ops-readonly   --mode timed   --days 3   --channel stable
```

Start the GUI launcher:

```bash
./tailstick-linux-gui
```

The GUI opens a local browser tab and also prints the localhost URL.

For remote preview or lab access on a specific interface:

```bash
./tailstick-linux-gui --host 0.0.0.0 --port 18080 --open-browser=false
```

### building from source

Most operators should use the release archives above. Building from source is only needed for local development:

```bash
go build -o tailstick-linux-cli ./cmd/tailstick-linux-cli
go build -o tailstick-linux-gui ./cmd/tailstick-linux-gui
go build -o tailstick-windows-cli.exe ./cmd/tailstick-windows-cli
go build -o tailstick-windows-gui.exe ./cmd/tailstick-windows-gui
```

## platform support

| platform | cli | gui | full lease runtime |
| --- | --- | --- | --- |
| linux (debian/ubuntu + `systemd`) | supported | supported | supported |
| windows | supported | supported | supported |

## command surface

| command | purpose |
| --- | --- |
| `tailstick run` | enroll machine and create lease |
| `tailstick agent` | run reconciliation loop (`--once` for a single pass) |
| `tailstick cleanup` | force cleanup for one lease id |
| `tailstick version` | print version |

for lease modes, tailstick installs local scheduling:

- linux: `tailstick-agent.service` + `tailstick-agent.timer`
- windows: `TailStickAgent-Startup` + `TailStickAgent-Periodic`

the scheduled command runs from a machine-local binary copy:

- linux: `/var/lib/tailstick/tailstick-agent`
- windows: `%ProgramData%\\TailStick\\tailstick-agent.exe`

## lease modes

| mode | behavior |
| --- | --- |
| `session` | cleaned after reboot/shutdown detection |
| `timed` | cleaned when expiry is reached (`1`, `3`, `7`, or bounded custom days) |
| `permanent` | leaves normal tailscale install and no automatic cleanup |

## examples

timed lease with advanced custom duration:

```bash
./tailstick-linux-cli run \
  --preset ops-readonly \
  --mode timed \
  --custom-days 14 \
  --channel stable \
  --allow-existing
```

session lease with approved exit node:

```bash
./tailstick-linux-cli run \
  --preset ops-readonly \
  --mode session \
  --exit-node 100.64.0.5 \
  --channel latest
```

manual cleanup:

```bash
./tailstick-linux-cli cleanup --lease-id <lease-id>
```

## testing

core checks:

```bash
go test ./...
go vet ./...
go build ./cmd/tailstick-linux-cli ./cmd/tailstick-linux-gui ./cmd/tailstick-windows-cli ./cmd/tailstick-windows-gui
```

isolated linux sandbox e2e:

```bash
make sandbox-linux
```

ci also runs:

- linux docker sandbox e2e: `tests/sandbox/linux-sandbox-e2e.sh`
- windows vm smoke: `tests/sandbox/windows-vm-smoke.ps1`

## branding assets

- canonical logo source: `assets/icon/tailstick-logo.png`
- windows executable icon resources: `cmd/tailstick-windows-*/resource_windows_amd64.syso`
- gui favicon asset: `internal/gui/tailstick-favicon.png`

regenerate windows icon resources after logo updates:

```bash
make icons
```

## documentation

- [architecture](docs/architecture.md)
- [configuration](docs/configuration.md)
- [operations](docs/operations.md)
- [testing](docs/testing.md)
- [release runbook](docs/release-runbook.md)
- [plan](PLAN.md)

## disclaimer

this project is for legitimate administrative use on machines you own or are explicitly authorized to manage. it does not attempt stealth installation or stealth removal.

## license

[mit license](LICENSE)
