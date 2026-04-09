# Testing

## Automated Matrix

`ci.yml` now runs three lanes:

1. Unit and package validation (`gofmt`, `go vet`, `go test`, multi-entrypoint builds)
2. Linux sandbox E2E in Docker (`ubuntu:24.04`)
3. Windows VM smoke on `windows-latest`

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
