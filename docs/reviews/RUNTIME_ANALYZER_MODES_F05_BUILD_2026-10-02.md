# CCMAI-RUNTIME-027 / F05 BUILD evidence — analyzer modes, caps, Vietnam dates, snapshots and checkpoint

**Date:** 2026-10-02 · **Phase:** BUILD → REVIEW_PENDING · **Risk:** R2 · **Role:** IMPLEMENTATION_WORKER + COMMIT_STEWARD (Claude); independent REVIEWER: Codex
**Authority:** [SPEC](../specs/RUNTIME_ANALYZER_MODES_F05_2026-10-02.md), [work order](../work_orders/CCMAI_RUNTIME_027.md), dispatcher seed `CVF_SESSION/authority/CCMAI-RUNTIME-027.json` (committed at base `71c7ee7`, not edited).
Synthetic providers, mocked frontend API and disposable MySQL only. Real provider/channel/notification calls: **zero**; no API key, persistent DB, schema/DSN/timestamp migration, push or FREEZE. No governance claim is made (no `governanceReceipt`).

## What changed

**Engine** (`backend/engine`)
- `analyzer_modes.go` (new): an explicit `runPlan{mode, limit, dates}`. Modes: `ordinary` (R025 source-version selection, no cap), `test_run`, `conditional`, `unanalyzed`, `since_last`. `newPlan` rejects before any side effect: negative cap, test run without a cap, ordinary with a cap, dates on any mode but `conditional` (`ErrInvalidRunParameters`). All public entry points keep their names/signatures and delegate to `Analyzer.execute`; the mode is chosen by the entry point, never inferred from the cap.
- Candidate selection per explicit mode, tenant- and input-channel-scoped, ordered `last_message_at ASC, id ASC` (NULL first), **cap applied after eligibility**:
  - `conditional`: every conversation, optional Vietnam date bounds (`>= From`, `< ToExclusive`, typed instants, so NULL never matches a bound); a cap no longer turns it into "unanalyzed only".
  - `unanalyzed`: no `conversation_evaluation` of *this tenant and job* (EXISTS over `job_results` ⨝ `job_runs` with tenant and job identity; PASS/SKIP/classification evaluations count; finding-only rows, other jobs/tenants do not).
  - `since_last`: `last_message_at >` the newest evaluated conversation's last message in the job's current channels; no evaluated conversation → unanalyzed policy. **A failed anchor query fails the run** (no fallback to "no anchor").
  - `test_run`: window `last_message_at > run start − 7d` on the clock captured once, not yet evaluated, capped.
- Snapshots: every explicit mode analyzes the **full local snapshot** (same builder/format as ordinary, no time cutoff), prepared before the provider call and saved unchanged. A date bound selects the conversation; it never cuts its context.
- Terminal state: only the ordinary unlimited run writes `last_run_at` (R025 checked finalizer, unchanged). Other explicit modes write `last_run_status`/`updated_at` through the same checked finalizer **with no checkpoint**; a test run keeps its previous behavior of not touching the job row, using the new `finalizeRunOnly` (same lock/retry/read-back rules, run row only). Cancellation (context error or stored `cancelled`) marks the run interrupted for every mode and never moves the checkpoint. `conversations_found = prepared + errors` for all modes.
- Seams (tests only): exported `NewAnalyzerWithProvider` (narrow, used by the route test); handler var `newTriggerAnalyzer`.

**Shared parser** (`backend/pkg/business_dates.go`, new): the R026 parser moved to `pkg` (`ParseBusinessRange`, `BusinessRange`, `ErrInvalidDateRange`, exported min/max) so handler and engine share one implementation; `api/handlers/business_dates.go` is now a thin adapter (aliases + `parseBusinessRange` wrapper); R026 behavior and its tests are unchanged.

**HTTP** (`jobs.go` `TriggerJob`): strict admission after the tenant-scoped job lookup and before config loading/worker start: unknown mode, `limit` present but not a positive integer, or dates on a non-conditional mode → `400 invalid_run_parameters`; malformed/reversed dates → `400 invalid_date_range`. The legacy `full=true` and absent mode still mean `conditional` / `since_last`. `TestRunJob` is unchanged.

**Frontend** (`JobDetail.vue`): the cap field is shown for every mode and labeled as a count cap; date fields are labeled Vietnam time and only conditional sends them; integer ≥ 1 validation blocks the confirm button; the conditional "at least one condition" rule stays. Copy is inline in the view (i18n files are outside the allowed paths).

## Tests (all in allowed paths)

- `analyzer_modes_test.go` (engine, single **and** batch, disposable MySQL): plan validation matrix; entry points reject invalid parameters with no run row and no provider call; **mode matrix** with named conversations (evaluated by the job, evaluated only by another job, never evaluated new/old/NULL, other channel, other tenant) for unanalyzed (cap 0/2), full (cap 0/2 — cap does not switch to unanalyzed), since-last (anchor), test run (window, cap 1/5): exact evaluated sets, provider call counts, checkpoint equals a pre-set sentinel, test run leaves the job row untouched and other modes record `last_run_status`; since-last without anchor; **anchor error** (the table is renamed so the query really fails: error names the anchor, run `error`, zero provider calls); **Vietnam dates under driver `loc=UTC` and `loc=Asia/Ho_Chi_Minh`** with the four named instants (`16:59:59Z`/`17:00:00Z` on two days), from-only, to-only, cap after the date filter, and a **full-snapshot** case (a September message outside the date window is sent and saved: 2 messages); cancellation (context and stored) keeps the checkpoint; finalize-failure via a `BEFORE UPDATE ON jobs` trigger (reached: run `error`, stored not success, checkpoint kept) with a test-run positive control (trigger not reached, success); only ordinary advances the checkpoint after a full run.
- `job_trigger_modes_test.go` (handlers): 14 rejection cases (each: 400 + code, **no config load, no worker, no run row, no cancel handle**), 8 accepted combinations with exact params reaching the worker, and **one real route → real `startTriggerJob` → `Analyzer` → DB** execution (synthetic provider through `newTriggerAnalyzer`): `mode=since_last&limit=2` over three conversations evaluates exactly the two oldest, status `success`, checkpoint preserved.
- `pkg/business_dates_test.go`: inclusive/exclusive boundaries, open ends, 8 invalid inputs.
- `job-run-dialog-modes.spec.ts` (frontend, mounted `JobDetail`, mocked API): exact request URLs for since-last+cap, unanalyzed, conditional dates under browser zones UTC / Asia/Ho_Chi_Minh / America/New_York, dates+cap, stale dates not sent after switching mode, and invalid input (1.5, 0, −2, reversed dates, no condition) sending nothing.
- Existing tests changed: the R025 `TestOrdinaryIncrementalModeBoundary` (a flag-combination predicate that no longer exists) is replaced by `TestOrdinaryPlanIsOnlyTheUncappedOrdinaryMode`; `setupIncFixture` connects through `connectIncDB` (location-aware, default unchanged). No other existing test changed; all R017/R025/R026 tests pass unmodified.

## Non-vacuity (compiling mutations, each restored byte-for-byte)

| Mutation | Result |
|---|---|
| anchor error returns "no anchor" | killed |
| other job's evaluation counts as evaluated | killed |
| cap switches `conditional` to `unanalyzed` | killed |
| explicit runs write the checkpoint | killed |
| candidate order reversed before the cap | killed |
| test run takes the job-row finalizer | killed |
| upper bound inclusive | killed |
| date parsed in UTC instead of Vietnam (pkg tests) | killed |

## Gates run

- Backend, whole module on disposable MySQL via `scripts/test-backend.ps1`: **706 pass, 0 fail, 2 optional skips**; R019 gate `ci_db_test_gate.py` PASSED (five sentinels, **0 DB-unavailable skips**). My first full run showed 6 DB-unavailable skips (new fixtures exhausted connections); fixed by closing the previous pool in the new fixtures.
- Frontend: vitest 24 files / 222 tests pass; `vue-tsc -b --force` clean; production build OK.
- Docs build PASS (6.71 s); workspace doctor 25/25 PASS; gate unit tests 46/46 PASS; explicit-`--files` preflight over the 19 changed files 7/7 PASS including catalog (not a whole-worktree pass; the pre-existing untracked `knowledge/_index.json` is excluded).

## Deviations, limits and deliberate behavior changes

- **Allowed-path discipline:** `backend/engine/analyzer_incremental.go` is *not* in the allowed paths and is therefore unmodified. Consequence: `isOrdinaryIncremental` remains as an unreferenced helper (its old test was replaced); explicit non-test runs reuse `finalizeOrdinaryRun` with a `nil` checkpoint, and `finalizeRunOnly` in `analyzer_modes.go` covers the test run. A follow-up may delete the dead helper.
- Behavior changes accepted by the SPEC: explicit runs never move `last_run_at`; transcript context for explicit modes is the full local snapshot (grows relative to date-cut transcripts); `conditional`+cap re-analyzes evaluated conversations; since-last is an event-time cursor (it does not cover older, equal-timestamp or backdated unevaluated conversations).
- Not done / out of scope: F06 run ownership/cancellation registry, scheduler/adapters, timestamp/DSN storage, tenant-timezone Settings, i18n keys for the dialog copy. Batch-mode partial AI failures keep the pre-existing accounting.
- Evidence limits: synthetic providers prove selection, persistence and request contracts, not model quality or CVF runtime governance.
