<#
.SYNOPSIS
  Builds and starts bot-service locally, exposes it via a Cloudflare quick
  tunnel, and registers the tunnel URL as the Telegram webhook.

.DESCRIPTION
  Run repeatedly for each dev session - it stops any previously started
  instance (tracked in logs/run.pid.json), rebuilds, restarts, opens a new
  tunnel (the URL changes every run, that's normal for a free quick tunnel),
  and re-registers the webhook so Telegram points at the new URL.

  All paths are resolved relative to this script's own location, so it can
  be run from any working directory:

      powershell -ExecutionPolicy Bypass -File .\scripts\start.ps1

.NOTES
  Requires: .env in the service root (copy .env.example -> .env and fill in
  BOT_TELEGRAM_BOT_TOKEN, BOT_TELEGRAM_WEBHOOK_SECRET,
  BOT_TELEGRAM_WEBHOOK_PATH_SECRET). Requires `go` on PATH.
#>

$ErrorActionPreference = "Stop"

$ScriptDir  = Split-Path -Parent $MyInvocation.MyCommand.Path
$ServiceDir = Split-Path -Parent $ScriptDir
$LogDir     = Join-Path $ServiceDir "logs"
$BinDir     = Join-Path $ScriptDir "bin"
$ServiceBin = Join-Path $ServiceDir "bin"
$EnvFile    = Join-Path $ServiceDir ".env"
$PidFile    = Join-Path $LogDir "run.pid.json"

New-Item -ItemType Directory -Force -Path $LogDir     | Out-Null
New-Item -ItemType Directory -Force -Path $BinDir     | Out-Null
New-Item -ItemType Directory -Force -Path $ServiceBin | Out-Null

# Writes/overwrites run.pid.json immediately as each process starts (rather
# than only once at the very end) so that if a later step fails (tunnel
# never comes up, Telegram rejects the webhook, ...), the next run's
# stop.ps1 call still finds and kills whatever was already spawned instead
# of leaving an orphaned process holding the port.
function Save-RunState {
    param($BotServicePid, $CloudflaredPid, $TunnelUrl, $WebhookUrl)
    [ordered]@{
        BotServicePid  = $BotServicePid
        CloudflaredPid = $CloudflaredPid
        TunnelUrl      = $TunnelUrl
        WebhookUrl     = $WebhookUrl
        StartedAt      = (Get-Date).ToString("o")
    } | ConvertTo-Json | Set-Content -Path $PidFile
}

if (-not (Test-Path $EnvFile)) {
    Write-Error "$EnvFile not found. Copy .env.example to .env and fill in your Telegram bot token/secrets first."
    exit 1
}

# --- parse the .env file for the values this script needs directly ---
$envValues = @{}
Get-Content $EnvFile | ForEach-Object {
    if ($_ -match '^\s*#' -or $_ -match '^\s*$') { return }
    if ($_ -match '^\s*([A-Z0-9_]+)\s*=\s*(.*)$') {
        $envValues[$matches[1]] = $matches[2].Trim().Trim('"''')
    }
}
$BotToken      = $envValues["BOT_TELEGRAM_BOT_TOKEN"]
$WebhookSecret = $envValues["BOT_TELEGRAM_WEBHOOK_SECRET"]
$PathSecret    = $envValues["BOT_TELEGRAM_WEBHOOK_PATH_SECRET"]
$Port          = $envValues["BOT_HTTP_PORT"]
if (-not $Port) { $Port = 8080 }

if (-not $BotToken -or $BotToken -eq "123456:ABC-your-telegram-bot-token") {
    Write-Error "BOT_TELEGRAM_BOT_TOKEN is missing/placeholder in $EnvFile"
    exit 1
}
if (-not $WebhookSecret -or -not $PathSecret) {
    Write-Error "BOT_TELEGRAM_WEBHOOK_SECRET / BOT_TELEGRAM_WEBHOOK_PATH_SECRET missing in $EnvFile"
    exit 1
}

# --- stop whatever a previous run of this script started ---
& (Join-Path $ScriptDir "stop.ps1") | Out-Null

# --- ensure cloudflared.exe is present (downloaded once, then reused) ---
$Cloudflared = Join-Path $BinDir "cloudflared.exe"
if (-not (Test-Path $Cloudflared)) {
    Write-Output "Downloading cloudflared (first run only)..."
    Invoke-WebRequest -Uri "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-windows-amd64.exe" -OutFile $Cloudflared
}

# --- build bot-service ---
Write-Output "Building bot-service..."
Push-Location $ServiceDir
try {
    go build -o (Join-Path $ServiceBin "bot-service.exe") ./cmd
} finally {
    Pop-Location
}

# --- start bot-service ---
$BotLog    = Join-Path $LogDir "bot-service.log"
$BotErrLog = Join-Path $LogDir "bot-service.err.log"
$BotProc = Start-Process -FilePath (Join-Path $ServiceBin "bot-service.exe") `
    -WorkingDirectory $ServiceDir `
    -RedirectStandardOutput $BotLog `
    -RedirectStandardError $BotErrLog `
    -PassThru -WindowStyle Hidden

Write-Output "bot-service started (PID $($BotProc.Id))"
Save-RunState -BotServicePid $BotProc.Id -CloudflaredPid $null -TunnelUrl $null -WebhookUrl $null
Start-Sleep -Seconds 2

# --- start the cloudflared quick tunnel ---
# cloudflared logs to stderr; stdout is normally empty, but Start-Process
# requires two distinct files even so.
$TunnelLog    = Join-Path $LogDir "cloudflared.log"
$TunnelOutLog = Join-Path $LogDir "cloudflared.out.log"
$CfProc = Start-Process -FilePath $Cloudflared `
    -ArgumentList "tunnel", "--url", "http://localhost:$Port" `
    -RedirectStandardOutput $TunnelOutLog `
    -RedirectStandardError $TunnelLog `
    -PassThru -WindowStyle Hidden

Write-Output "cloudflared started (PID $($CfProc.Id))"
Save-RunState -BotServicePid $BotProc.Id -CloudflaredPid $CfProc.Id -TunnelUrl $null -WebhookUrl $null

# --- wait for the tunnel URL to show up in its log ---
$TunnelUrl = $null
for ($i = 0; $i -lt 30; $i++) {
    Start-Sleep -Seconds 1
    foreach ($candidateLog in @($TunnelLog, $TunnelOutLog)) {
        if (Test-Path $candidateLog) {
            $match = Select-String -Path $candidateLog -Pattern "https://[a-z0-9-]+\.trycloudflare\.com" | Select-Object -First 1
            if ($match) {
                $TunnelUrl = $match.Matches[0].Value
                break
            }
        }
    }
    if ($TunnelUrl) { break }
}

if (-not $TunnelUrl) {
    Write-Error "Could not detect the tunnel URL from $TunnelLog after 30s. Open the log and check manually."
    exit 1
}

Write-Output "Tunnel URL: $TunnelUrl"

# --- wait for the tunnel's DNS to actually resolve/forward before telling
#     Telegram about it (trycloudflare.com subdomains take a few seconds
#     to propagate right after creation) ---
Write-Output "Waiting for tunnel to become reachable..."
$tunnelReady = $false
for ($i = 0; $i -lt 20; $i++) {
    try {
        $health = Invoke-WebRequest -Uri "$TunnelUrl/health" -UseBasicParsing -TimeoutSec 5
        if ($health.StatusCode -eq 200) {
            $tunnelReady = $true
            break
        }
    } catch {
        Start-Sleep -Seconds 3
    }
}
if (-not $tunnelReady) {
    Write-Error "Tunnel never became reachable at $TunnelUrl/health. Check $TunnelLog."
    exit 1
}

# --- register the webhook with Telegram (retry a few times: Telegram's own
#     resolvers can lag a couple seconds behind what's reachable from here) ---
$WebhookUrl = "$TunnelUrl/webhook/telegram/$PathSecret"
$resp = $null
for ($i = 0; $i -lt 5; $i++) {
    try {
        $resp = Invoke-RestMethod -Method Post -Uri "https://api.telegram.org/bot$BotToken/setWebhook" -Body @{
            url          = $WebhookUrl
            secret_token = $WebhookSecret
        }
        if ($resp.ok) { break }
    } catch {
        $resp = $null
    }
    Start-Sleep -Seconds 3
}

if (-not $resp) {
    Write-Error "Could not reach Telegram's setWebhook API after retries."
    exit 1
}
Write-Output ("setWebhook response: {0}" -f ($resp | ConvertTo-Json -Compress))

if (-not $resp.ok) {
    Write-Error "Telegram rejected the webhook registration - check the bot token and the response above."
    exit 1
}

# --- persist final state for stop.ps1 and for your own reference ---
Save-RunState -BotServicePid $BotProc.Id -CloudflaredPid $CfProc.Id -TunnelUrl $TunnelUrl -WebhookUrl $WebhookUrl

Write-Output ""
Write-Output "Ready. Message your bot on Telegram now."
Write-Output "Logs directory: $LogDir"
Write-Output "  - bot-service.log / bot-service.err.log : bot-service stdout/stderr"
Write-Output "  - cloudflared.log                        : tunnel connection + the public URL"
Write-Output "  - cloudflared.out.log                    : cloudflared stdout (normally empty)"
Write-Output "Stop everything with: powershell -ExecutionPolicy Bypass -File .\scripts\stop.ps1"
