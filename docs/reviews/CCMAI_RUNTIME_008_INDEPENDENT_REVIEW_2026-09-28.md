# Independent review: CCMAI-RUNTIME-008

**Reviewer:** Codex (`REVIEWER`) · **Date:** 2026-09-28 · **Target:** local BUILD commit `edc6323` · **Disposition:** `PASS` for R2 REVIEW. S1 remains IN_PROGRESS; FREEZE remains open.

## Decision

The changed product source is confined to `backend/api/handlers/channels.go` and focused handler tests, as authorized by the [work order](../work_orders/CCMAI_RUNTIME_008.md). After the tenant-scoped lookup, `SyncChannelNow` calls `loadManualSyncConfig` before the R007 `syncing` write. An error or nil config returns generic `500 sync_start_failed`, with no status mutation or worker launch and no raw validation detail in response/log. A valid `*config.Config` is passed through `startManualSync` to `runManualSync`; the worker no longer reloads configuration. The R007 checked write, 202 body, timeout and panic recovery remain in place.

Codex inspected the new tests and the seam restoration. A forced load error and nil result are refused; valid config pointer identity, persisted start state, wrong-tenant 404 before config load, and the real `config.Load` with synthetic valid/invalid environment values are covered. Existing R007 start-write and panic tests still run. No test invokes a real adapter or provider; the actual worker `SyncEngine` call remains a source-inspected branch.

## Independent verification

- Compared `edc6323^..edc6323` with the SPEC/work order and checked the changed-set diff; `git diff --check` is clean.
- Reran `scripts/test-backend.ps1 -Packages ./api/handlers -Run 'TestSyncChannelNow|TestHandleManualSyncPanic|TestDeleteChannel|TestPurgeChannel'` against a separate disposable `mysql:8.0` database via `golang:1.26-alpine`: PASS for the focused R008, R007 and channel regression group. The script removed its MySQL container and network; no `ccma-test-*` container remains.
- Claude's BUILD evidence separately records all 13 backend packages, `go build ./...`, `go vet ./...`, mutation check, catalog check and doctor 25/25 passing. Codex reran doctor 25/25 before review.

The finding in `channels.go` OAuth/credential handlers, `jobs.go` and `agents.go` that other `config.Load` errors are ignored is valid source-trace input for a separately scoped tranche. It does not block this manual-sync work order. Cross-path single-flight and stale `syncing` after a crash also remain S1 residuals. R001–R008 FREEZE decisions and S1 closure remain open; no real channel sync, credential/provider call, customer data, persistent Compose DB change, deployment, push or live CVF governance claim follows from this review.
