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


## why

if you need to get a field machine onto your tailnet without building a backend service around the process, tailstick gives you a controlled local workflow. it keeps the operator flow simple, supports session, timed, and permanent access, and keeps cleanup tied to the same lease record instead of spreading state across ad hoc scripts.

## requirements

- **elevated privileges** are required for enrollment and cleanup (`sudo` on linux, administrator on windows).
- **linux**: debian or ubuntu with `systemd`. other distributions are not supported in the current release.
- **windows**: standard windows with scheduled tasks available.
- **go 1.22+** is required for building from source.

## quickstart

1. go to [releases](https://github.com/Microck/tailstick/releases/latest) and download the binary you want for your machine. use the cli if you want a direct command flow, or use the gui if you want the local browser flow. binaries are available for `linux/amd64`, `linux/arm64`, `windows/amd64`, and `windows/arm64`.
2. put that binary in a working folder, then place a `tailstick.config.json` file next to it.
3. write that config by hand from `configs/tailstick.config.example.json`, or use the optional [preset maker](https://tailstick.micr.dev/) if you want a faster way to build it.
4. make sure you have the secrets your presets depend on. in most setups that means a tailscale auth key, a tailscale ephemeral auth key, and a tailscale api key for cleanup. if you want password-gated operator use, you also need an operator password.
5. supply those values through environment variables such as `TAILSTICK_AUTH_KEY`, `TAILSTICK_EPHEMERAL_AUTH_KEY`, `TAILSTICK_API_KEY`, and `TAILSTICK_OPERATOR_PASSWORD`.
6. run the binary once the config and secrets are in place.

for normal operator use, use the release binaries. building from source is only for development work.

## platform support

| platform | cli | gui | full lease runtime |
| --- | --- | --- | --- |
| linux (debian/ubuntu + `systemd`) | supported | supported | supported |
| windows | supported | supported | supported |

release binaries are built for `linux/amd64`, `linux/arm64`, `windows/amd64`, and `windows/arm64`.

## command surface

| command | purpose |
| --- | --- |
| `tailstick run` | enroll machine and create lease |
| `tailstick agent` | run reconciliation loop (`--once` for a single pass) |
| `tailstick cleanup` | force cleanup for one lease id |
| `tailstick version` | print version |

### `run` flags

| flag | default | description |
| --- | --- | --- |
| `--preset` | `""` | preset id from config |
| `--mode` | `session` | lease mode: `session`, `timed`, or `permanent` |
| `--channel` | `stable` | install channel: `stable` or `latest` |
| `--days` | `3` | timed lease duration in days |
| `--custom-days` | `0` | custom lease days (1–30, overrides `--days`) |
| `--suffix` | `""` | optional device name suffix |
| `--exit-node` | `""` | optional approved exit node |
| `--allow-existing` | `false` | allow existing tailscale install |
| `--non-interactive` | `false` | non-interactive mode |
| `--password` | `""` | operator password (or set `TAILSTICK_OPERATOR_PASSWORD`) |
| `--config` | `tailstick.config.json` | config file path |
| `--state` | platform default | state file path |
| `--audit` | `logs/tailstick-audit.ndjson` | audit log path |
| `--log` | platform default | log file path |
| `--dry-run` | `false` | print commands without executing |

### `agent` flags

| flag | default | description |
| --- | --- | --- |
| `--once` | `false` | run one reconciliation pass and exit |
| `--interval` | `1m` | reconciliation interval |

the agent also accepts `--config`, `--state`, `--audit`, `--log`, and `--dry-run`.

### `cleanup` flags

| flag | default | description |
| --- | --- | --- |
| `--lease-id` | `""` | lease id to clean (required) |

the cleanup command also accepts `--config`, `--state`, `--audit`, `--log`, and `--dry-run`.

### gui flags

the gui binary (`tailstick-linux-gui` / `tailstick-windows-gui`) starts a local browser-based setup wizard. in addition to the shared flags (`--config`, `--state`, `--audit`, `--log`, `--dry-run`):

| flag | default | description |
| --- | --- | --- |
| `--host` | `127.0.0.1` | bind host |
| `--port` | `0` | bind port (`0` picks an ephemeral port) |
| `--open-browser` | `true` | open browser automatically |

## lease modes

| mode | behavior |
| --- | --- |
| `session` | cleaned after reboot/shutdown detection |
| `timed` | cleaned when expiry is reached (set duration with `--days` or `--custom-days`) |
| `permanent` | leaves normal tailscale install and no automatic cleanup |

for lease modes, tailstick installs local scheduling:

- linux: `tailstick-agent.service` + `tailstick-agent.timer`
- windows: `TailStickAgent-Startup` + `TailStickAgent-Periodic`

the scheduled command runs from a machine-local binary copy:

- linux: `/var/lib/tailstick/tailstick-agent`
- windows: `%ProgramData%\TailStick\tailstick-agent.exe`

## environment variables

secrets are resolved from environment variables. the config file supports `authKeyEnv`, `ephemeralAuthKeyEnv`, and `apiKeyEnv` fields that name the env var to read. common variables:

| variable | purpose |
| --- | --- |
| `TAILSTICK_AUTH_KEY` | tailscale auth key |
| `TAILSTICK_EPHEMERAL_AUTH_KEY` | tailscale ephemeral auth key |
| `TAILSTICK_API_KEY` | tailscale api key for cleanup |
| `TAILSTICK_OPERATOR_PASSWORD` | operator password for gated presets |

see `.env.example` for a template.

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

single agent pass:

```bash
./tailstick-linux-cli agent --once
```

## building from source

```bash
go build ./cmd/tailstick-linux-cli
go build ./cmd/tailstick-linux-gui
```

cross-compile all targets:

```bash
make build-all
```

run tests:

```bash
go test ./...
```

see the `Makefile` for additional targets (`fmt`, `vet`, `sandbox-linux`, `icons`).

## documentation

- [architecture](docs/architecture.md)
- [configuration](docs/configuration.md)
- [operations](docs/operations.md)
- [testing](docs/testing.md)
- [release runbook](docs/release-runbook.md)
- [contributing](CONTRIBUTING.md)

## license

[mit license](LICENSE)
