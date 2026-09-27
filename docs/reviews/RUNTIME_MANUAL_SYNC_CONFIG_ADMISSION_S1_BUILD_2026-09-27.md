# BUILD evidence: CCMAI-RUNTIME-008 — manual sync configuration admission

**Implementer:** Claude (`IMPLEMENTATION_WORKER`) · **Date:** 2026-09-28 · **Risk:** R2 · **Authority:** [SPEC](../specs/RUNTIME_MANUAL_SYNC_CONFIG_ADMISSION_S1_2026-09-27.md), [work order](../work_orders/CCMAI_RUNTIME_008.md) · **Status:** `REVIEW_PENDING` for independent Codex REVIEW. No FREEZE.

## Entry

Claude acknowledged `WORK_ORDER_AUTHOR (Codex) -> IMPLEMENTATION_WORKER (Claude)` in the active handoff at `12acef6`, after rehydrating manifest, policy, state, handoff, memory, implementation status, docs index, SPEC and work order.

Changed source:
- `backend/api/handlers/channels.go`;
- `backend/api/handlers/sync_start_test.go` (seam signature and fixture only);
- new focused test `backend/api/handlers/sync_config_admission_test.go`.

No path addition was needed. Codex's parallel uncommitted `CCMAI-DEMO-BRAND-001` files were not touched or staged.

## Trace (before editing)

1. `SyncChannelNow` does the tenant-scoped lookup (404 on miss), then the R007 checked `syncing` write, then `startManualSync(tenantID, channel)`, then 202 `sync_started`.
2. The worker `runManualSync` called `cfg, _ := config.Load()`. The error was ignored, so an invalid configuration gave `cfg == nil`, which went into `engine.NewSyncEngine(nil)` **after** `syncing` had been recorded and 202 sent.
3. `config.Load` fails when `JWT_SECRET` is missing or shorter than 32 characters, `ENCRYPTION_KEY` is missing or not 32 bytes, or `DB_PASSWORD` is missing. Its messages name the variables; they do not contain values.

## Implementation (`channels.go`)

- **New private seam** `var loadManualSyncConfig = config.Load`.
- **`SyncChannelNow`:** after the unchanged tenant-scoped lookup, and **before** any status write, it calls the loader. If it returns an error or nil, it:
  - logs only `manual sync for channel <id> not admitted: configuration invalid` (no error text, since validation messages describe secret settings);
  - returns the existing generic `500 {"error":"sync_start_failed"}`;
  - writes no status, checkpoint or error, and launches no worker.

  Otherwise the R007 checked start write runs unchanged.
- **`startManualSync(tenantID, channel, cfg)`** now carries the validated `*config.Config`. **`runManualSync`** receives it and **no longer calls `config.Load`**.
- **Unchanged:** the 10-minute timeout, `SyncEngine.SyncChannel` call, R007 panic recovery (`handleManualSyncPanic`), 202 body, 404 path, and R007 write-error/zero-row behavior.

## Tests

The fixture now also stubs `loadManualSyncConfig`: it returns a synthetic `&config.Config{Env: "test"}` or a forced error, and counts calls. Both package-global seams are restored in `t.Cleanup`, and no test uses `t.Parallel`. The launcher is still stubbed, so no adapter, credential or provider is called.

| Test | Proves |
|---|---|
| `TestSyncChannelNowConfigFailureStartsNoWorker` | A forced load error gives generic `500 sync_start_failed`, zero launches, the previous status `success` still visible, and one load. The injected error text (`ENCRYPTION_KEY=SECRET-DO-NOT-LEAK`) appears in neither body nor log; the log names the channel and "configuration invalid". |
| `TestSyncChannelNowNilConfigIsNotAdmitted` | A nil config without an error is also refused: non-2xx, no launch, status unchanged. |
| `TestSyncChannelNowPassesValidatedConfigToWorker` | The unchanged 202 body; exactly one load and one launch; the launched worker receives **the same pointer** the loader returned; `syncing` is persisted before launch and before any response byte. |
| `TestSyncChannelNowWrongTenantSkipsConfigLoad` | Another tenant gets 404, with **zero** config loads, no launch and status unchanged (tenant check stays ahead of config). |
| `TestSyncChannelNowUsesRealConfigValidation/invalid` | The **real `config.Load`** with synthetic env (`JWT_SECRET` too short) gives non-2xx, no launch, and no "JWT" detail in body or log. |
| `TestSyncChannelNowUsesRealConfigValidation/valid` | The real `config.Load` with synthetic env (32+ character JWT secret, 32-byte encryption key, placeholder DB password) gives 202, one launch, the worker's config carrying exactly those synthetic values, and status `syncing`. |
| R007 regressions (`TestSyncChannelNowRecordsStart…`, `…WriteFailure…`, `…ZeroRow…`, `…OtherTenantCannotStart`, three `TestHandleManualSyncPanic…`) | Pass after the seam signature change. |

**Non-vacuity (mutation).** With the admission check disabled (`if false && (err != nil || cfg == nil)`), `TestSyncChannelNowConfigFailureStartsNoWorker`, `TestSyncChannelNowNilConfigIsNotAdmitted` and `TestSyncChannelNowUsesRealConfigValidation/invalid` each failed with `status 202, want non-2xx; body {"message":"sync_started"}`. The source was restored from a backup, with no marker left.

## Verification

Run with the new `scripts/test-backend.ps1`. It starts disposable `mysql:8.0` with `--log-bin-trust-function-creators=1` at startup, runs `golang:1.26-alpine` with the module cache read-only, `GOPROXY=off` and `GOFLAGS=-mod=readonly`, and always cleans up. The host's Windows Application Control was not bypassed. All trigger-based tests passed first time, with no `Error 1419`.

```text
scripts/test-backend.ps1 -Packages ./api/handlers -Run 'TestSyncChannelNow|TestHandleManualSyncPanic|TestDeleteChannel|TestPurgeChannel' -VerboseTests
                                                                  15/15 PASS (5 new + R007 + channel regressions), exit 0
mutation (admission check disabled)                               3 regressions FAIL as expected; restored
scripts/test-backend.ps1                                          all 13 packages ok, exit 0
host: go build ./... ; go vet ./...                               clean
backend/go.mod, backend/go.sum                                    unchanged
gofmt -l                                                          new test files clean; channels.go only has its 3 pre-existing hunks
                                                                  (~167, ~520, ~599), disjoint from this change (688–729)
git diff --check                                                  clean
scripts/manage_cvf_downstream_catalog.ps1 -Check                  PASS
check_cvf_workspace_agent_enforcement.ps1 -ProjectPath .          PASS 25/25
```

Cleanup: every disposable MySQL container and network was removed by the script; none remain. The persistent Compose `ccma` stack was not touched.

## Limits and residuals

- **Untested branch:** `runManualSync` itself (a real `SyncEngine` run with the passed config) is not executed by tests, because that would require a real adapter. The call site is covered by source inspection and the passed-pointer test.
- **Load timing:** configuration is loaded per manual request (as before, just earlier); a later configuration change is picked up by the next request.
- **Out of scope, observed during trace:** other handlers still ignore `config.Load` errors — `channels.go` OAuth/credential paths (lines ~97, ~398, ~446, ~634, ~784), `jobs.go` trigger/test-run (~358, ~408) and `agents.go` (~87). Candidates for a separate tranche.
- **Still S1 residuals:** scheduler/agent paths, cross-path single-flight, and stale `syncing` after a crash.
- No real channel sync, credential, customer data, provider API, deployment or push was used. This is not live CVF governance proof.
