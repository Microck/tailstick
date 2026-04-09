Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$workspace = if ($env:GITHUB_WORKSPACE) { $env:GITHUB_WORKSPACE } else { (Get-Location).Path }
$bin = Join-Path $workspace "dist\\tailstick-windows-cli.exe"
$workDir = Join-Path $env:TEMP "tailstick-sandbox"
$configPath = Join-Path $workDir "tailstick.config.json"
$statePath = Join-Path $workDir "state.json"
$logPath = Join-Path $workDir "tailstick.log"
$auditPath = Join-Path $workDir "audit.ndjson"

New-Item -Path $workDir -ItemType Directory -Force | Out-Null

$config = @'
{
  "stableVersion": "1.76.6",
  "defaultPreset": "ops-readonly",
  "operatorPassword": "op-pass",
  "presets": [
    {
      "id": "ops-readonly",
      "description": "windows vm smoke preset",
      "authKey": "tskey-auth-fake",
      "ephemeralAuthKey": "tskey-auth-ephemeral",
      "tags": ["tag:ops-usb"],
      "acceptRoutes": true,
      "allowExitNodeSelection": true,
      "approvedExitNodes": ["100.64.0.5"],
      "cleanup": {
        "tailnet": "example.com",
        "deviceDeleteEnabled": false
      }
    }
  ]
}
'@
$config | Set-Content -Path $configPath -Encoding UTF8

$runArgs = @(
  "run",
  "--config", $configPath,
  "--state", $statePath,
  "--log", $logPath,
  "--audit", $auditPath,
  "--preset", "ops-readonly",
  "--mode", "timed",
  "--days", "3",
  "--channel", "stable",
  "--allow-existing",
  "--password", "op-pass",
  "--dry-run"
)

$runOutput = ((& $bin @runArgs 2>&1) | Out-String).Trim()
if ($LASTEXITCODE -ne 0) {
  throw "run command failed: $runOutput"
}
if ($runOutput -notmatch "Lease created:") {
  throw "run output did not contain lease creation marker: $runOutput"
}

$stateRaw = Get-Content -Path $statePath -Raw
if ($stateRaw -notmatch '"status":\s*"active"') {
  throw "state did not contain active lease status"
}
$leaseMatch = [regex]::Match($stateRaw, '"leaseId"\s*:\s*"([^"]+)"')
if (-not $leaseMatch.Success) {
  throw "could not parse lease id from state"
}
$leaseId = $leaseMatch.Groups[1].Value

$cleanupArgs = @(
  "cleanup",
  "--config", $configPath,
  "--state", $statePath,
  "--log", $logPath,
  "--audit", $auditPath,
  "--lease-id", $leaseId,
  "--dry-run"
)
$cleanupOutput = ((& $bin @cleanupArgs 2>&1) | Out-String).Trim()
if ($LASTEXITCODE -ne 0) {
  throw "cleanup command failed: $cleanupOutput"
}

$agentArgs = @(
  "agent",
  "--once",
  "--config", $configPath,
  "--state", $statePath,
  "--log", $logPath,
  "--audit", $auditPath,
  "--dry-run"
)
$agentOutput = ((& $bin @agentArgs 2>&1) | Out-String).Trim()
if ($LASTEXITCODE -ne 0) {
  throw "agent --once command failed: $agentOutput"
}

$stateAfter = Get-Content -Path $statePath -Raw
if ($stateAfter -notmatch '"status":\s*"cleaned"') {
  throw "state did not contain cleaned lease status after cleanup"
}

Write-Host "windows-vm-smoke: PASS"
