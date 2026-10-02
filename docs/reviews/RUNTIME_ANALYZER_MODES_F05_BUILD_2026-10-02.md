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

---

# R027-R1 repair evidence (appended 2026-10-02)

**Role:** REPAIR_WORKER + COMMIT_STEWARD (Claude), acknowledged in the handoff before repair. **Input:** [independent review](CCMAI_RUNTIME_027_F05_INDEPENDENT_REVIEW_2026-10-02.md) of BUILD `a1de36b` (CHANGES_REQUIRED F05-R1-01..04) and the updated work order. Same authority: seed unchanged, risk R2, no new path class; synthetic providers, disposable MySQL, zero real provider/channel/notification calls, no persistent DB, push or FREEZE. The reviewer's retained tests (`job_f05_review_test.go`, `analyzer_f05_review_test.go`, the two mounted one-sided tests) are unchanged and pass.

## Repairs

- **F05-R1-01 (admission):** `parseTriggerParams` in `jobs.go`. `limit` absent/empty/`0` is unlimited; otherwise an unsigned decimal (`^[0-9]+$` + `Atoi`: `+2`, spaces, hex, fractions, negatives, overflow → `invalid_run_parameters`; leading zeros accepted). `full` is absent/empty/`true`/`false`; `full=true` with an explicit non-conditional mode, or any other `full` value, is rejected; `full=true` alone still aliases conditional. Rejections happen before configuration loading, worker start, run, activity or cancel handle (asserted on each case).
- **F05-R1-02 (UI):** the paired-date guards are removed (only a reversed pair is invalid), so a single valid endpoint submits unchanged; the dialog now says since-last also omits conversations whose last message equals the anchor, conditional re-evaluates evaluated conversations, and dates (Vietnam time) only select conversations while each keeps its full context. Inline copy only.
- **F05-R1-03 (test-run bookkeeping):** the test run now finalizes through the existing checked `finalizeOrdinaryRun` with a `nil` checkpoint, so `last_run_status`/`updated_at` are written (tenant-scoped, locked, read-back verified) and `last_run_at` is preserved. `finalizeRunOnly` is deleted. `analyzer_incremental.go` is untouched.
- **F05-R1-04 (acceptance proof):** see the tests below.

## Worker tests changed to follow the SPEC (not the failing reviewer assertions)

- `job_trigger_modes_test.go`: `limit=0` and `limit=` moved from rejected to admitted (cap 0); added `+2`, whitespace, overflow, hex, `full` conflict/invalid/uppercase rejections and `007`, `full=false`, `full=true&limit=2`, empty `full` admissions.
- `analyzer_modes_test.go`: the matrix and finalize-failure tests no longer encode "a test run leaves the job row untouched"; they assert `last_run_status == success` and `updated_at` advanced for every explicit mode including the test run, and the finalize-fault test now covers test-run too.

## New acceptance tests

- **Route → Analyzer (real handler, real `startTriggerJob`, real `Analyzer`, DB, synthetic recording provider):** `TestTriggerJobRealRouteConditionalDateCapRepeatsEvaluation` — `unanalyzed` evaluates four conversations; then `mode=conditional&from=2026-10-02&to=2026-10-02&limit=2` re-evaluates exactly the two oldest conversations of that Vietnam day (the one at VN `00:00:00` with an earlier September message, and a mid-day one), excludes the one at `16:59:59Z` (previous VN day) and the later one (cap after date filter), creates a second evaluation of the repeated conversation, sends and saves its full 2-message snapshot, keeps the mode `conditional`, and preserves the checkpoint sentinel while recording `last_run_status`.
- **Evaluation/anchor isolation (single and batch):** PASS and SKIP legacy evaluations (no snapshot) count; a finding-only row, another job's evaluation, another tenant and an other-channel evaluated conversation do not; since-last anchor excludes the equal-time conversation and ignores the newer evaluated conversation in a non-input channel; NULL-only evaluated anchors fall back to unanalyzed; a 400-day-old conversation is selected by unanalyzed while the test run keeps its strict 7-day window (exactly 7 days excluded, +1 s included); equal `last_message_at` ties are cut by id for full/unanalyzed/test run; classification-job evaluations count; zero-message sources create no evaluation or provider call.
- **Full-snapshot repeat evaluation (single and batch):** a full rerun with cap 1 and a date+cap run each create another evaluation of an already evaluated conversation whose saved snapshot equals the sent transcript (2 messages, including the one outside the date bound), summary `found 1 / analyzed 1`, checkpoint preserved.
- **Named Vietnam date cases:** one shared list (same day, month rollover, year rollover, leap day, from-only, to-only, reversed, not-a-leap-year, impossible calendar date, time string, before the minimum, supplied maximum) is run (a) through the real `TriggerJob` admission (202 + exact params / 400 `invalid_date_range` with no config/worker/run), (b) in the Analyzer against an independent hand-computed oracle (`date − 7 h` arithmetic, not the parser) over conversations at ±1 s of each VN midnight and +3 h, under driver `loc=UTC` and `loc=Asia/Ho_Chi_Minh`, single and batch, with invalid cases creating no run and no provider call, and (c) in the mounted dialog for every case a date control can submit, under browser zones UTC and America/New_York (a date input cannot produce a malformed string, minimum or maximum; those are proved at layers (a)/(b)).
- **Terminal state:** the checkpoint sentinel is preserved for every explicit mode × cap class (full/unanalyzed/since-last cap 0 and 1; test run cap 1 and 50) × outcome (success, zero work, provider error, cancel at the last provider call, cancel with an empty selection), with the expected run status, `job.last_run_status == run.status`, and provider-reached assertions for the error/cancel outcomes. Job-write failure (trigger) and job-row-missing (job leaves the tenant during the run) fault tests cover all four explicit modes: the provider was reached, the run is `error` and stored non-success, no notification is sent, the checkpoint is kept; the control without a fault notifies exactly once. R025's own read-back/retry/missing-row tests cover the shared finalizer.

## Old-source and pre-repair failures (compiling)

- **Pre-F05 base `ebfa05b`** (git worktree, one file copied in): `analyzer_f05_probe_test.go` uses only public entry points and R025 fixture helpers. Command: `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./engine -Run TestF05Probe -VerboseTests`. Result at base: **4/4 FAIL** — full+cap `got []` (excluded evaluated conversation), Vietnam day `got []` (UTC parsing / strict lower bound), invalid date silently accepted, unlimited unanalyzed run moved `last_run_at` to the scan time. On the repaired source: 4/4 PASS.
- **Pre-repair BUILD `a1de36b`** (git worktree; repaired test files copied in; engine `-Run 'TestExplicit|TestEntry|TestF05'`, handlers `-Run 'TestTriggerJob|TestF05Review'`): engine — 10 test-run checkpoint/bookkeeping subtests, 4 matrix subtests, 2 finalize-failure subtests FAIL; handlers — 5 strict-admission rejection cases and 2 admission-acceptance cases FAIL. Everything else, including the new route/isolation/date tests, passed there (they close missing evidence rather than expose new defects).

## Mutations (compiling, each restored; commands: `scripts/test-backend.ps1 -Packages <pkg> -Run <focused set>`)

| Mutation | Result |
|---|---|
| test run skips the job-row terminal write | killed |
| `full=true` accepted with an explicit non-conditional mode | killed |
| signed-plus cap accepted | killed |
| zero cap rejected | killed |
| since-last anchor ignores the current input channels | killed |
| finding-only rows count as evaluated | killed |
| explicit snapshot cut by a time bound (date bound cuts context) | killed |
| frontend: restore the to-requires-from guard | killed (3 mounted tests) |

The F05 BUILD mutations were rerun against the repaired source with the same focused sets (`-Run 'TestExplicit|TestSinceLast|TestUnanalyzed|TestFullAnd|TestFullMode|TestOnlyOrdinary|TestPlanValidation|TestEntryPoints|TestVietnam'`):

| Mutation | Result |
|---|---|
| since-last anchor error falls back to "no anchor" | killed |
| another job's evaluation counts as evaluated | killed |
| a cap switches `conditional` to `unanalyzed` | killed |
| explicit runs write the checkpoint | killed |
| candidate order reversed before the cap | killed |
| inclusive upper date bound | killed (rerun alone after the batch run was interrupted; the file was restored and the leftover disposable container/network removed) |

The earlier "test run touches the job row" mutation is superseded by "test run skips the job-row terminal write" above. The UTC-parsing mutation (pkg parser) was killed at BUILD (`TestParseBusinessRangeBoundaries`); the parser is unchanged in R1.

## Gates

- Backend, whole module, uncached, disposable MySQL via the scratch wrapper around `scripts/test-backend.ps1` (`go test -json`): **824 pass, 0 fail, 2 optional skips** (`ai/pricing` live fetch, `storage` S3); `python scripts/ci_db_test_gate.py` **PASSED** (five sentinels pass, **0 DB-unavailable skips**). Disposable MySQL container and network were removed by the script; afterwards `docker ps -a`/`docker network ls` showed no `ccma-test` resources. Worktree probes created and then removed `git worktree` copies (`git worktree list` shows only the main checkout). Go build and vet clean.
- Frontend: vitest 24 files / **239 tests pass** (was 222); `vue-tsc -b --force` clean; production build OK; docs build OK.
- Docs build PASS (11.9 s), workspace doctor 25/25 PASS, gate unit tests 46/46 PASS, explicit-`--files` preflight over the 18 changed paths 7/7 PASS including catalog and tranche (the first preflight correctly rejected my history token `REPAIR`; the gate state machine is `CHANGES_REQUIRED -> BUILD -> REVIEW_PENDING`, so the record uses `BUILD`). Not a whole-worktree pass: the untracked `knowledge/_index.json` and Python bytecode directories are excluded.

## Residuals and limits

- Review item "the only route test used since_last without an anchor": replaced by the conditional date+cap route test above; the earlier since-last route test is kept and described as unanalyzed-fallback evidence only.
- `analyzer_incremental.go` (outside the allowlist) is unmodified; `isOrdinaryIncremental` remains an unreferenced helper (separate cleanup).
- Since-last stays an event-time cursor (older, equal-timestamp or backdated source omitted); no source-version, F02 completeness, F06 ownership/cancellation or hosted-CI claim. Synthetic providers prove selection, persistence and request contracts only; no governance claim.
