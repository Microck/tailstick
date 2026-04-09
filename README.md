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

if you need to get a field machine onto your tailnet without building a backend service around the process, tailstick gives you a controlled local workflow. it keeps the operator flow simple, supports session, timed, and permanent access, and keeps cleanup tied to the same lease record instead of spreading state across ad hoc scripts.

## quickstart

1. go to [releases](https://github.com/Microck/tailstick/releases/latest) and download the binary you want for your machine. use the cli if you want a direct command flow, or use the gui if you want the local browser flow.
2. put that binary in a working folder, then place a `tailstick.config.json` file next to it.
3. write that config by hand from `configs/tailstick.config.example.json`, or use the optional [preset maker](https://tailstick.micr.dev/) if you want a faster way to build it.
4. make sure you have the secrets your presets depend on. in most setups that means a tailscale auth key, a tailscale ephemeral auth key, and a tailscale api key for cleanup. if you want password-gated operator use, you also need an operator password.
5. supply those values through environment variables such as `TAILSTICK_AUTH_KEY`, `TAILSTICK_EPHEMERAL_AUTH_KEY`, `TAILSTICK_API_KEY`, and `TAILSTICK_OPERATOR_PASSWORD`.
6. run the binary once the config and secrets are in place.

for normal operator use, use the release binaries. building from source is only for development work.

## common workflows

use the cli when you already know the preset and want a direct operator flow.

use the gui when you want a small local browser form instead of command flags.

use session mode when access should disappear with the machine session. use timed mode when access should expire after a fixed number of days. use permanent mode when you want to leave a normal tailscale install behind.

## common uses

common uses include short-lived field support, temporary onboarding for remote ops work, usb-based enrollment in restricted environments, and controlled permanent installs where you still want the setup to begin from a preset instead of a hand-written command every time.

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
- windows: `%ProgramData%\TailStick\tailstick-agent.exe`

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

## documentation

- [architecture](docs/architecture.md)
- [configuration](docs/configuration.md)
- [operations](docs/operations.md)
- [testing](docs/testing.md)
- [release runbook](docs/release-runbook.md)
- [plan](PLAN.md)

## license

[mit license](LICENSE)
