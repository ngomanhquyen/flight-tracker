<#
.SYNOPSIS
  Stops the bot-service + cloudflared tunnel processes started by
  start.ps1 (reads their PIDs from logs/run.pid.json).

.DESCRIPTION
      powershell -ExecutionPolicy Bypass -File .\scripts\stop.ps1

  Safe to run even if nothing is running - it just reports that there was
  nothing to stop. Also called automatically at the top of start.ps1 so
  each dev session starts from a clean state.
#>

$ErrorActionPreference = "Stop"

$ScriptDir  = Split-Path -Parent $MyInvocation.MyCommand.Path
$ServiceDir = Split-Path -Parent $ScriptDir
$LogDir     = Join-Path $ServiceDir "logs"
$PidFile    = Join-Path $LogDir "run.pid.json"

if (-not (Test-Path $PidFile)) {
    Write-Output "Nothing tracked as running (no $PidFile)."
    exit 0
}

$state = Get-Content $PidFile | ConvertFrom-Json
foreach ($p in @($state.BotServicePid, $state.CloudflaredPid)) {
    if (-not $p) { continue }
    try {
        Stop-Process -Id $p -Force -ErrorAction Stop
        Write-Output "Stopped PID $p"
    } catch {
        Write-Output "PID $p was not running (already stopped)"
    }
}

Remove-Item $PidFile -Force -ErrorAction SilentlyContinue
Write-Output "Done."
