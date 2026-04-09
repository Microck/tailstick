# Release Runbook

This runbook is the canonical release path for `tailstick`.

## Scope

- Cut a semver tag (`vX.Y.Z`)
- Push tag to trigger GitHub Actions release workflow
- Verify release artifacts and checks

## Prerequisites

- `gh auth status` returns a logged-in maintainer
- local branch is up to date with `origin/main`
- no uncommitted changes

## Preflight Checklist

1. Validate code quality:

```bash
go test ./...
go vet ./...
make icons
go build ./cmd/tailstick-linux-cli ./cmd/tailstick-linux-gui ./cmd/tailstick-windows-cli ./cmd/tailstick-windows-gui
make sandbox-linux
```

2. Confirm version intent:
- `internal/app/cli.go` `Version` constant matches release target.

3. Confirm release workflow exists:
- `.github/workflows/release.yml` triggers on `v*` tags.

## Release Procedure

Assume target version is `v0.1.0`:

```bash
git checkout main
git pull --ff-only origin main
git tag v0.1.0
git push origin main
git push origin v0.1.0
```

Tag push triggers the release workflow which builds and uploads:

- `tailstick-linux-amd64.tar.gz`
- `tailstick-linux-arm64.tar.gz`
- `tailstick-windows-amd64.tar.gz`
- `tailstick-windows-arm64.tar.gz`

## Verify Release

1. Monitor workflow:

```bash
gh run list -R Microck/tailstick --workflow release.yml --limit 5
gh run watch -R Microck/tailstick <run-id> --exit-status
```

2. Verify release entry and assets:

```bash
gh release view v0.1.0 -R Microck/tailstick
gh release view v0.1.0 -R Microck/tailstick --json assets,url
```

3. Optional post-release smoke:
- Download one Linux and one Windows archive from the release and verify expected binaries are present.

## Rollback

If release workflow fails before artifacts are correct:

1. Fix issue on `main`.
2. Delete remote/local tag.
3. Recreate and push corrected tag.

```bash
git tag -d v0.1.0
git push origin :refs/tags/v0.1.0
git tag v0.1.0
git push origin v0.1.0
```
