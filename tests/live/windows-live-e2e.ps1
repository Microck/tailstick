Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Require-Env([string]$Name) {
  if ([string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($Name))) {
    throw "missing required env: $Name"
  }
}

function Get-BasicAuthHeader([string]$ApiKey) {
  $tokenBytes = [System.Text.Encoding]::ASCII.GetBytes("${ApiKey}:")
  return @{
    Authorization = "Basic $([Convert]::ToBase64String($tokenBytes))"
  }
}

function Wait-ForDeviceGone([string]$DeviceId, [hashtable]$Headers) {
  for ($i = 0; $i -lt 30; $i++) {
    try {
      Invoke-RestMethod -Method Get -Uri "https://api.tailscale.com/api/v2/device/$DeviceId" -Headers $Headers | Out-Null
      Start-Sleep -Seconds 2
    } catch {
      $statusCode = $_.Exception.Response.StatusCode.value__
      if ($statusCode -eq 404) {
        return
      }
      Start-Sleep -Seconds 2
    }
  }

  throw "device $DeviceId still exists after cleanup"
}

function Assert-TaskExists([string]$TaskName) {
  $queryOutput = (& schtasks /Query /TN $TaskName 2>&1 | Out-String).Trim()
  if ($LASTEXITCODE -ne 0) {
    throw "expected scheduled task $TaskName to exist: $queryOutput"
  }
}

function Assert-TaskMissing([string]$TaskName) {
  $queryOutput = (& schtasks /Query /TN $TaskName 2>&1 | Out-String).Trim()
  if ($LASTEXITCODE -eq 0) {
    throw "scheduled task $TaskName still exists: $queryOutput"
  }
}

Require-Env "TAILSTICK_EPHEMERAL_AUTH_KEY"
Require-Env "TAILSTICK_API_KEY"
Require-Env "TAILSTICK_OPERATOR_PASSWORD"

$workspace = if ($env:GITHUB_WORKSPACE) { $env:GITHUB_WORKSPACE } else { (Get-Location).Path }
$bin = Join-Path $workspace "dist\tailstick-windows-cli.exe"
$workDir = Join-Path $env:TEMP "tailstick-live-e2e"
$configPath = Join-Path $workDir "tailstick.config.json"
$statePath = Join-Path $workDir "state.json"
$logPath = Join-Path $workDir "tailstick.log"
$auditPath = Join-Path $workDir "audit.ndjson"
$programDataRoot = if ($env:ProgramData) { $env:ProgramData } else { "C:\ProgramData" }
$agentBinaryPath = Join-Path $programDataRoot "TailStick\tailstick-agent.exe"
$agentLauncherPath = Join-Path $programDataRoot "TailStick\agent.cmd"
$headers = Get-BasicAuthHeader $env:TAILSTICK_API_KEY
$deviceId = $null

New-Item -Path $workDir -ItemType Directory -Force | Out-Null

$config = @'
{
  "defaultPreset": "live-e2e-windows",
  "operatorPasswordEnv": "TAILSTICK_OPERATOR_PASSWORD",
  "presets": [
    {
      "id": "live-e2e-windows",
      "description": "live Windows E2E",
      "ephemeralAuthKeyEnv": "TAILSTICK_EPHEMERAL_AUTH_KEY",
      "cleanup": {
        "tailnet": "live-e2e",
        "apiKeyEnv": "TAILSTICK_API_KEY",
        "deviceDeleteEnabled": true
      }
    }
  ]
}
'@
$config | Set-Content -Path $configPath -Encoding UTF8

try {
  $runOutput = ((& $bin run `
    --config $configPath `
    --state $statePath `
    --log $logPath `
    --audit $auditPath `
    --preset live-e2e-windows `
    --mode session `
    --channel latest `
    --allow-existing `
    --password $env:TAILSTICK_OPERATOR_PASSWORD 2>&1) | Out-String).Trim()

  if ($LASTEXITCODE -ne 0) {
    throw "run command failed: $runOutput"
  }

  $state = Get-Content -Path $statePath -Raw | ConvertFrom-Json
  $record = $state.records | Select-Object -First 1
  if ($null -eq $record) {
    throw "state file did not contain a lease record"
  }
  if ($record.status -ne "active") {
    throw "expected active state after enrollment, got $($record.status)"
  }

  $deviceId = $record.deviceId
  $credentialRef = $record.credentialRef
  if ([string]::IsNullOrWhiteSpace($deviceId)) {
    throw "device id missing from state"
  }
  if ([string]::IsNullOrWhiteSpace($credentialRef) -or -not (Test-Path $credentialRef)) {
    throw "credential ref missing or unreadable"
  }

  Invoke-RestMethod -Method Get -Uri "https://api.tailscale.com/api/v2/device/$deviceId" -Headers $headers | Out-Null

  Assert-TaskExists "TailStickAgent-Startup"
  Assert-TaskExists "TailStickAgent-Periodic"

  if (-not (Test-Path $agentBinaryPath)) {
    throw "expected agent binary at $agentBinaryPath"
  }
  if (-not (Test-Path $agentLauncherPath)) {
    throw "expected agent launcher at $agentLauncherPath"
  }

  $cleanupOutput = ((& $bin cleanup `
    --config $configPath `
    --state $statePath `
    --log $logPath `
    --audit $auditPath `
    --lease-id $record.leaseId 2>&1) | Out-String).Trim()

  if ($LASTEXITCODE -ne 0) {
    throw "cleanup command failed: $cleanupOutput"
  }

  $stateAfterCleanup = Get-Content -Path $statePath -Raw | ConvertFrom-Json
  $recordAfterCleanup = $stateAfterCleanup.records | Select-Object -First 1
  if ($recordAfterCleanup.status -ne "cleaned") {
    throw "expected cleaned state after cleanup, got $($recordAfterCleanup.status)"
  }
  if (Test-Path $credentialRef) {
    throw "credential ref should be removed after cleanup"
  }

  Wait-ForDeviceGone -DeviceId $deviceId -Headers $headers
  $deviceId = $null

  $agentOutput = ((& $bin agent `
    --once `
    --config $configPath `
    --state $statePath `
    --log $logPath `
    --audit $auditPath 2>&1) | Out-String).Trim()

  if ($LASTEXITCODE -ne 0) {
    throw "agent --once command failed: $agentOutput"
  }

  Assert-TaskMissing "TailStickAgent-Startup"
  Assert-TaskMissing "TailStickAgent-Periodic"

  Start-Sleep -Seconds 5
  if (Test-Path $agentBinaryPath) {
    throw "agent binary still exists after self-removal"
  }
  if (Test-Path $agentLauncherPath) {
    throw "agent launcher still exists after self-removal"
  }

  Write-Host "windows-live-e2e: PASS"
} finally {
  if (-not [string]::IsNullOrWhiteSpace($deviceId)) {
    try {
      Invoke-RestMethod -Method Delete -Uri "https://api.tailscale.com/api/v2/device/$deviceId" -Headers $headers | Out-Null
    } catch {
    }
  }
}
