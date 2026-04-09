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

1. Open the latest release on GitHub and download the two files for your machine: the CLI binary and the GUI binary.
2. Put both binaries in a working folder on the USB or target machine.
3. Create a `tailstick.config.json` file in that same folder.
4. Add the secrets your preset needs before you run anything.
5. Launch the CLI for scripted use, or launch the GUI if you want the browser-based operator flow.

Get the binaries from [GitHub Releases](https://github.com/Microck/tailstick/releases/latest).

### config file

TailStick expects a `tailstick.config.json` next to the binaries.

You can:

- copy `configs/tailstick.config.example.json` and edit it manually, or
- use the optional [preset maker](https://tailstick.micr.dev/) to generate a starting config file

The preset maker is optional. It just helps you build `tailstick.config.json` faster and explains the fields.

### what you need before first run

Most setups need these values available at runtime:

- a Tailscale auth key for normal installs
- a Tailscale ephemeral auth key for session-only installs
- a Tailscale API key if you want device cleanup through the API
- an operator password only if you want password-gated use

By default these are usually supplied through environment variables such as:

- `TAILSTICK_AUTH_KEY`
- `TAILSTICK_EPHEMERAL_AUTH_KEY`
- `TAILSTICK_API_KEY`
- `TAILSTICK_OPERATOR_PASSWORD`

### using the preset maker

The preset maker helps generate a config file. Use it if you want a faster way to define presets, install commands, cleanup behavior, and allowed options for operators.

Use the generated JSON as your `tailstick.config.json` file.

### running tailstick

Use the CLI when you want a direct command-based flow.

Use the GUI when you want a local browser form for the operator.

Typical examples:

- `tailstick-cli-linux-amd64 run ...`
- `tailstick-gui-windows-amd64.exe`

If you want to build from source for development, the repo still supports that, but normal operators should use release binaries instead.

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
