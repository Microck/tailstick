# Testing

## Automated Matrix

`ci.yml` now runs three lanes:

1. Unit and package validation (`gofmt`, `go vet`, `go test`, multi-entrypoint builds)
2. Linux sandbox E2E in Docker (`ubuntu:24.04`)
3. Windows VM smoke on `windows-latest`

## Live E2E

File: `.github/workflows/live-e2e.yml`

This workflow is manual by design (`workflow_dispatch`) and uses real Tailscale credentials stored as GitHub Actions secrets:

- `TAILSTICK_LIVE_E2E_API_KEY`
- `TAILSTICK_LIVE_E2E_OPERATOR_PASSWORD`

It is intentionally separate from the default CI matrix because it creates real tailnet devices and performs real cleanup/delete operations.

Each live lane mints a short-lived auth key from the Tailscale API at runtime so the workflow does not depend on long-lived or previously-consumed auth keys.

### Linux Live E2E

- Runs inside a privileged Ubuntu 24.04 systemd container built from `tests/live/linux-live-e2e.dockerfile`
- Installs the real Tailscale package in the container
- Starts an isolated `tailscaled` instance with its own socket and state file
- Executes the real Linux CLI binary against the live tailnet
- Verifies real device creation, cleanup, API deletion, and agent self-removal

### Windows Live E2E

- Runs on GitHub-hosted `windows-latest`
- Installs the real Tailscale Windows client on the ephemeral runner
- Executes the real Windows CLI binary against the live tailnet
- Verifies live enrollment, scheduled task creation, cleanup, API deletion, and agent self-removal

These live lanes are isolated from the developer workstation running Codex. The Linux lane is additionally isolated from the host runner via a dedicated container and `tailscaled` socket/state.

## Linux Sandbox E2E

File: `tests/sandbox/linux-sandbox-e2e.sh`

This test executes the real Linux CLI binary in an isolated container and validates:

- Enrollment path for a timed lease
- Agent binary copy to `/var/lib/tailstick/tailstick-agent`
- Agent scheduling artifacts (`tailstick-agent.service` and `tailstick-agent.timer`)
- Cleanup to `cleaned` state
- Cleanup secret file removal
- Agent self-removal when no managed leases remain

It uses local fake command fixtures for `tailscale` and `systemctl` so behavior is deterministic and does not require real tailnet credentials.

## Windows VM Smoke

File: `tests/sandbox/windows-vm-smoke.ps1`

This test executes the real Windows CLI binary on a GitHub-hosted Windows VM and validates:

- Enrollment command flow
- State persistence
- Cleanup flow and status transition to `cleaned`
- Agent reconciliation command path

It runs with `--dry-run` because full non-dry-run enrollment requires elevated privileges and host-side Windows service/task effects that are not guaranteed in the shared CI runner context.
