# Changelog

All notable changes to [Tailstick](https://github.com/Microck/tailstick) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## April 2026

### Documentation

- docs: improve README with requirements, CLI flags, env vars, build instructions (81720b9)
- docs: tighten readme for operators (807792b)
- docs: fix quickstart download guidance (daf3831)
- docs: simplify quickstart copy (ecb686b)
- docs: add release runbook and harden gitignore (f73f935)

### Changed

- [nightshift] docs-backfill: add package doc comments and CONTRIBUTING.md (6ab3f18)
- Update README to remove common workflows and uses (3d512e0)

### Fixed

- fix(security): harden gui preset and auth handling (32cff66)
- fix(preset-maker): validate presets before export (2eb9b47)
- fix(e2e): use supported agent install commands (cfbc6a1)
- fix(e2e): clear windows live script exit state (400e9f4)
- fix(e2e): stop windows agent after self-removal (2b1b349)
- fix(e2e): detach windows agent artifact cleanup (169089b)
- fix(e2e): handle live cleanup races (f41de17)
- fix(e2e): correct live task assertions (a313bb1)
- fix(ci): mint fresh tailscale keys per live run (169a9e2)
- fix(e2e): stabilize live windows and linux lanes (a07d745)
- fix: allow pages workflow to bootstrap site (3884f6e)
- fix: normalize windows smoke script output matching (650a64e)

### Removed

- Delete PLAN.md (bc42e0d)

### Testing

- test(e2e): poll for windows agent artifact cleanup (6234bb4)
- test(e2e): wait for windows scheduled task state (1ea4d9a)
- test(e2e): timeout windows cli phases (a99f8a9)
- test(e2e): bound windows live api waits (8f595e7)
- test(ci): add live Tailscale E2E workflows (254c930)
- test(workflow): cover cleanup and agent reconciliation (e300801)

### Added

- feat: simplify operator onboarding (c1b0263)
- feat: cut 1.0.0 release and pages deploy (a647084)
- feat: restyle web ui as tailscale-style clone (ae812fc)
- feat: allow gui host/port and add release-asset quickstart (2abc3b9)
- feat: add project logo resources for gui and windows binaries (a9abd1f)
- feat: bootstrap tailstick with lease engine and publish-ready docs (c45de01)
