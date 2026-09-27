# UI screenshots

**Tranche:** `CCMAI-UX-000`. Repeatable desktop/mobile × light/dark captures for UI review evidence.

## Requirements

- Node 24 (uses the built-in `fetch` and `WebSocket`; no npm packages).
- Chrome or Edge installed locally (auto-detected; override with `CHROME_PATH` or `--chrome`).
- Docker Desktop for app mode.

## Static preview (no backend)

```powershell
powershell -ExecutionPolicy Bypass -File scripts/ui-screenshots.ps1 -Mode preview -OutDir <folder>
```

Builds the frontend, serves `dist` with `vite preview` on port 4173, and captures `/wireframes/design-system` with its review and confirm dialogs.

## Full app with synthetic data

```powershell
powershell -ExecutionPolicy Bypass -File scripts/ui-screenshots.ps1 -Mode app -OutDir <folder>
```

What it does, in order:

1. Starts a **new** Compose project `ccma-uishot-<timestamp>` from `scripts/ui-screenshots/compose.yml`. MySQL runs on `tmpfs`; secrets are random and written to a temp file outside the repo. The repo `.env` and the persistent project `ccma` are never read or touched.
2. Creates a throwaway admin through `/api/v1/setup` and imports the built-in synthetic demo data through the demo endpoint.
3. Binds two synthetic snapshots in that disposable database only: one valid manifest listing a message that does not exist (renders `changed_since_analysis`) and one whose digest does not match its bytes (renders `verification_unavailable`). Everything else stays `legacy_unverified`.
4. Captures dashboard, channels, channel detail, messages, jobs, QC and classification job detail, results and settings.
5. Runs `docker compose down -v --rmi local` and deletes the temp env and token files (`-KeepEnvironment` skips this for debugging).

No AI provider is called; demo results are pre-written synthetic data.

## Output

`<route>--<desktop|mobile>--<light|dark>.png` plus `report.json` with, per page, uncaught exceptions, `console.error` calls, browser log errors, requests to other hosts and whether the page scrolls horizontally. The script exits non-zero when any page has a JS error.

Direct use against any running app:

```powershell
node scripts/ui-screenshots.mjs --base http://127.0.0.1:3000 --out <folder> --routes "name=/path,/other" [--token-file <file>] [--themes light,dark] [--viewports desktop,mobile]
```

Desktop is 1440×900 at 1×; mobile is 390×844 at 2×. Pages taller than 8000px are cut at 8000px.
