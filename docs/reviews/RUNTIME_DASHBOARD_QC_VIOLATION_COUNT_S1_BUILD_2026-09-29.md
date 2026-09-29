# CCMAI-RUNTIME-017 — BUILD evidence: Dashboard `qc_violation_count`

**Date:** 2026-09-29 · **Status:** `REVIEW_PENDING` (Claude, IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD) · **Risk:** R2 · **Authority:** [SPEC](../specs/RUNTIME_DASHBOARD_QC_VIOLATION_COUNT_S1_2026-09-29.md), [work order](../work_orders/CCMAI_RUNTIME_017.md). Base commit `b8750fb`; role transition `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` recorded in the active handoff before any source edit.

## Change

`backend/api/handlers/dashboard.go` (`GetDashboard`): one additive query and one additive JSON key.

- `qc_violation_count` = `COUNT(*)` of `job_results` where `tenant_id = ? AND result_type = 'qc_violation' AND created_at BETWEEN from AND to`, using the same `from`/`to` variables as `issues`. Single table, no join. Empty match serializes as numeric `0`.
- If that count query fails the handler returns HTTP 500 `{"error":"dashboard_unavailable"}` immediately, with no SQL or driver text and no false zero. Error handling of every other existing query is unchanged.
- `issues` and all other response keys are untouched (still every result type in the interval).

No frontend, API client/store, router, model, schema or write change. The `gofmt -l` report on `dashboard.go` is the pre-existing import ordering (`gin` import placed before module imports); it is present at `HEAD` and was not touched.

## Tests

New `backend/api/handlers/dashboard_qc_violation_count_test.go` calls the real `GetDashboard` with gin test contexts against disposable MySQL (synthetic rows, three tenants, window `2026-03-10`):

| Test | Proves |
|---|---|
| `…IsScopedByTypeTenantAndInterval` | tenant A: 4 violations (two on one conversation + rows exactly on both interval edges); `issues` = 6 (4 + evaluation + tag) with rows 1 s outside each edge excluded from both; tenant B sees 3/3; all 12 pre-existing keys still present; both fields are JSON numbers |
| `…IsNumericZeroWhenNothingMatches` | tenant with only an evaluation: `qc_violation_count` is the number `0` (`"qc_violation_count":0` in the body), `issues` = 1 |
| `…ChangesWhenAQualifyingRowIsAdded` | adding a qualifying row moves 4 → 5 (`issues` 6 → 7); adding an evaluation to A or a violation to B leaves A at 5 |
| `…QueryFailureReturnsGeneric500` | a GORM query callback fails only the COUNT whose vars contain `qc_violation`; asserts the hook fired, HTTP 500, body exactly `{"error":"dashboard_unavailable"}`, no `select`/`forced`/table/field name leak; after removing the hook the same request returns 4 |

## Commands and results

- `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./api/handlers -Run 'TestDashboardQC' -VerboseTests` → 4/4 PASS.
- `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./...` → all 14 packages with tests `ok` (handlers 49.3 s, engine 41.1 s); the script printed "Removed disposable MySQL and network", and `docker ps -a` showed no `ccma-test*` container afterwards. The persistent Compose `ccma` database was never touched.
- `go build ./...` OK; `go vet ./...` OK; `git diff --check` OK (CRLF conversion warning only).
- Catalog `-Check` and workspace doctor: see the final line of this file.

## Mutation checks (production file only, each restored byte-for-byte from a saved copy)

| Mutant | Result |
|---|---|
| M1 drop the `result_type` predicate | `IsScopedByTypeTenantAndInterval` FAILS |
| M2 drop the `tenant_id` predicate | `IsScopedByTypeTenantAndInterval` FAILS |
| M3 ignore the count error (false zero / 200) | `QueryFailureReturnsGeneric500` FAILS (`status 200, want 500`) |

An interval-edge mutation was not run; the boundary rows are asserted by the first test.

## Failure modes and claim limits

- Failure mode: a DB error on the new count returns the whole dashboard as 500 (the SPEC's required behavior); the frontend does not read the field yet, so it is unaffected until a later tranche reads it.
- The value is a database row count of finding rows, not distinct conversations, and not a statement that the channel data is complete or that an AI finding is correct.
- Synthetic disposable MySQL only; no provider or channel call, no customer data. This is not live CVF governance proof and does not need one (database API metric).
- Not verified: behavior against the persistent Compose database and rendering in the UI (frontend unchanged).

## Changed set

`backend/api/handlers/dashboard.go`, `backend/api/handlers/dashboard_qc_violation_count_test.go`, this file, active state/handoff, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json`, S1 roadmap note. Not self-approved; FREEZE open; push not performed.

**Final gates (after continuity sync):** catalog `-Check` PASS; workspace doctor 25/25 PASS; `git diff --check` clean.
