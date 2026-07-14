# scripts/

Local dev-only helper scripts for running bot-service against the real
Telegram API without deploying anywhere. Not used in CI/CD or Kubernetes —
those use the Dockerfile/Helm chart instead.

## Usage

```powershell
# from the bot-service directory (or anywhere; paths are self-relative)
powershell -ExecutionPolicy Bypass -File .\scripts\start.ps1
```

This will, in order:
1. Stop any previous run started by this script (safe to re-run any time).
2. Download `cloudflared.exe` into `scripts/bin/` on first run only (kept
   locally, never committed — see root `.gitignore`).
3. `go build` bot-service into `bin/bot-service.exe`.
4. Start bot-service, then start a Cloudflare quick tunnel
   (`https://<random>.trycloudflare.com`) pointing at it.
5. Call Telegram's `setWebhook` with that tunnel URL and the secrets from
   your `.env`.

When it finishes it prints the tunnel URL and confirms Telegram accepted
the webhook. Message your bot on Telegram right after.

To stop everything cleanly:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\stop.ps1
```

## Logs

Everything goes to `../logs/` (i.e. `services/bot-service/logs/`, created
automatically, gitignored):

| File | Contents |
|---|---|
| `bot-service.err.log` | **this is the one you want** — Zap (the structured logger) writes to stderr by default, so every startup line, HTTP request, and command dispatch result lands here, not in `bot-service.log` |
| `bot-service.log` | bot-service's stdout — just Gin's route-registration banner at startup; normally nothing else |
| `cloudflared.log` | tunnel connection log (cloudflared writes here, to stderr), including the line with the public `https://*.trycloudflare.com` URL for this run |
| `cloudflared.out.log` | cloudflared's stdout - normally empty |
| `run.pid.json` | process IDs + the current tunnel/webhook URL, used by `stop.ps1` - safe to delete, it's regenerated on the next `start.ps1` |

**View logs live** (PowerShell):
```powershell
Get-Content .\logs\bot-service.err.log -Wait -Tail 50
```

**View logs live** (Git Bash / the Bash tool):
```bash
tail -f logs/bot-service.err.log
```

Every request bot-service handles logs a `correlation_id` - grep for it to
follow one Telegram message through parsing, downstream calls, and the
reply:
```bash
grep "<correlation_id>" logs/bot-service.err.log
```

## Known limitation

The tunnel URL changes every time `start.ps1` runs (free Cloudflare quick
tunnels are ephemeral, not tied to an account) — that's expected. The
script re-registers the webhook automatically each run, so you don't need
to do anything manually; just re-run `start.ps1` at the start of each dev
session.
