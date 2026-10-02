# CCMAI-RUNTIME-027 / F05 — independent review

Date: 2026-10-02. Reviewer: Codex, independent of implementation worker Claude. Risk R2.
Disposition: **CHANGES_REQUIRED / FREEZE_OPEN** for exact BUILD `a1de36b6720cae4341f5937f6d9100b13bb48366`. No product repair or acceptance by this review. Retained independent tests deliberately fail pending Claude repair; no tranche closure.
Authority: [SPEC](../specs/RUNTIME_ANALYZER_MODES_F05_2026-10-02.md), [work order](../work_orders/CCMAI_RUNTIME_027.md), unchanged dispatcher seed/base `71c7ee761a1147d502d815ff5461e92f1c30bd4c`. [Worker evidence](RUNTIME_ANALYZER_MODES_F05_BUILD_2026-10-02.md) is BUILD evidence, not independent validation.

## Review entry and scope

Rehydrated manifest/policy, bootstrap fallback state, memory current prose, active handoff, implementation truth/index, SPEC/order/tranche/seed; core public remote/pin/origin-main checks passed doctor 25/25. Compact bootstrap absent: BOOTSTRAP_MIGRATION_PENDING, nonblocking. Knowledge ingest ran. Reported BLOCKED_CONTINUITY_DRIFT because implementation limitations still said R027 DISPATCH_READY and F05-F07 product source untouched, while current pointers correctly said REVIEW_PENDING. SESSION_SYNC_STEWARD corrected only those stale phrases and recorded the intake before source review; this does not accept BUILD. Role transition to REVIEWER (Codex), then ORCHESTRATOR / SESSION_SYNC_STEWARD / COMMIT_STEWARD for this review/repair dispatch. Seed authored/committed by dispatcher Codex before Claude's BUILD, unmodified; all 19 BUILD paths are inside authorized scope plus allowed continuity/review records. Core remains read-only.

## Blocking findings and repair acceptance

### F05-R1-01 — HTTP admission differs from the published contract (high)

`TriggerJob` in `backend/api/handlers/jobs.go` accepts only a present positive Atoi limit, never validates `full` or conflicts, and Atoi accepts a signed-plus integer. SPEC date/request points 3–4 require absent/empty/zero cap to mean unlimited, decimal unsigned syntax, validated full flag, and rejection of full=true with explicit nonconditional mode before configuration/dispatch.

Independent retained `backend/api/handlers/job_f05_review_test.go::TestF05ReviewAdmissionMatchesSpec` proves five failures:

| Request | SPEC | BUILD observed |
| --- | --- | --- |
| mode=unanalyzed&limit=0 | 202, cap 0 | 400 invalid_run_parameters |
| mode=unanalyzed&limit= | 202, cap 0 | 400 invalid_run_parameters |
| mode=unanalyzed&full=true | 400, no dispatch | 202, config loaded, worker/run created |
| mode=conditional&full=maybe | 400, no dispatch | 202, config loaded, worker/run created |
| mode=conditional&limit=%2B2 | 400, no dispatch | 202, config loaded, worker/run created |

`full=false` with since_last is a passing control. Repair strict parsing against SPEC, preserve default/legacy alias and positive caps, and update the worker's contradictory zero/empty rejection expectations. Assert no config/worker/run/activity/cancel handle on rejected cases, not merely a response code. No auth/cancellation-registry changes.

### F05-R1-02 — valid one-sided dates cannot be submitted from the UI (medium)

`JobDetail.vue` retains the old paired-date guards in runDateFromError/runDateToError. The API and shared engine parser permit an open side, but the dialog disables confirm. Two retained mounted tests in `frontend/src/__tests__/job-run-dialog-modes.spec.ts` prove from-only and to-only both fail `SPEC permits an open endpoint: expected true to be false`.

Repair the guards so a single valid endpoint can submit unchanged; retain reversed-range/invalid cap and at-least-one-condition rules. Also complete the SPEC's narrow dialog explanation: since-last omits equal timestamps as well as older source, and conditional dates select conversations while analysis retains full local context. Inline copy is within authority; i18n files remain read-only. Visible caps in all modes prevent a hidden conditional cap; that design is acceptable if request tests preserve the chosen mode.

### F05-R1-03 — test-run terminal job bookkeeping is an unapproved SPEC deviation (medium)

`Analyzer.execute` routes modeTestRun to finalizeRunOnly, deliberately preserving the whole job row. SPEC Terminal bookkeeping says every explicit mode must preserve last_run_at **and update last_run_status/updated_at through checked tenant-scoped persistence**; it gives no test-run exemption. Keeping the old behavior cannot silently replace this accepted contract. Worker mode-matrix/finalizer tests encode that divergence.

Independent retained `backend/engine/analyzer_f05_review_test.go::TestF05ReviewTestRunRecordsTerminalJobStatus` fails in both single and batch: returned/stored run succeeds but job last_run_status remains `prior` and updated_at remains the old sentinel. last_run_at correctly remains unchanged. Repair test-run terminal bookkeeping to the existing SPEC with no checkpoint argument; update contradictory R027 expectations and prove run/job write failures do not produce successful completion/notification. Do not change R025 finalizer source in analyzer_incremental.go or widen the seed. If preserving the whole job row is required instead, return an explicit acceptance-contract decision to the owner before changing SPEC; this review grants no such change.

### F05-R1-04 — required acceptance proof is incomplete (medium)

Worker evidence and tests do not supply the complete acceptance set. The only real route-to-Analyzer test uses since_last with **no evaluation anchor**, so it falls back to unanalyzed and cannot prove the required conditional date+cap rerun of an already evaluated conversation. Date DB tests exercise only single mode; mounted dates cover one October case and omit the required shared rollover/leap/extrema/open-end cases. Explicit-mode matrix is QC-only and lacks finding-only/SKIP/classification/legacy/null-anchor/current-channel anchor controls and deterministic equal-time cap ordering. Explicit terminal fault coverage is a jobs trigger for one non-test mode; the new run-only finalizer lacks direct run-write/read-back/missing-scope/retry-failure coverage. The evidence lists mutation kills but gives no reproducible compiling old-source failures or exact full-suite/R019 commands/cleanup details.

Complete the SPEC acceptance groups, prioritizing the exact route-to-Analyzer conditional+date+cap repeat-evaluation proof, single/batch full-context date checks, evaluation/anchor isolation controls and terminal failure/notification assertions. Supply compiling pre-repair failures and mutation results with exact commands/outcomes, full backend/R019/build results and disposable-resource cleanup. Reuse accepted evidence where genuinely applicable and cite it precisely; do not claim a helper-only/missing-anchor test proves a different entry/mode. This is the original acceptance scope, not new product scope.

## Independent verification actually run

All application tests use local synthetic providers/dispatch and isolated disposable MySQL; no real provider/channel/notification calls and no CVF AI-governance claim. Commands ran from project root. The shell wrappers printed the underlying failing exit explicitly (because printing a result can leave the wrapper process exit zero); the results below are the actual tool/test exits.

- `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./api/handlers -Run 'TestF05Review|TestTriggerJob' -VerboseTests`: actual exit 1; 8 existing top-level tests PASS, 1 new review group FAIL (5/6 subcases fail); no skips; package 18.981 s. Invalid cases demonstrably reached config/worker/run. DB/network removed.
- `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./engine -Run 'TestPlan|TestEntryPoints|TestExplicitMode|TestSinceLast|TestFullMode|TestOnlyOrdinary|TestOrdinary|TestNonOrdinary|Test.*Snapshot' -VerboseTests`: actual exit 0; 49 top-level PASS, zero failures/skips; package 71.102 s. Includes R025 source-version/cancellation/read-back regressions, explicit mode matrix, UTC/VN date/full-context test, anchor query fault and snapshot/provenance checks. This is focused engine verification, not the full backend/R019 gate. DB/network removed.
- `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./engine -Run '^TestF05ReviewTestRunRecordsTerminalJobStatus$' -VerboseTests`: actual exit 1; 1 review group FAIL, both single/batch subcases fail; package 1.131 s, zero skips. DB/network removed.
- `npm --prefix frontend test -- --reporter=dot`: actual exit 1; original 222 tests PASS plus 2 new review tests FAIL, 24 files (23 pass/1 fail), duration 14.58 s. No production component edits by reviewer.
- `PYTHONDONTWRITEBYTECODE=1`, `python -m unittest discover -s scripts/tests -p 'test_cvf_downstream_gate*.py'`: 46/46 PASS, 24.799 s. `npm --prefix docs run docs:build`: PASS, 11.77 s before this review artifact was added; final docs build recorded in handoff. Doctor 25/25 and diff check PASS.

Worker-reported full backend 706 PASS / 2 optional skips, R019, frontend typecheck/build and eight mutations were inspected as worker claims, not independently rerun/reproduced here. No new hosted CI proof; PR #1 remains draft at older remote head 3e0b37e. Default worktree gate includes unrelated untracked knowledge index and Python bytecode directories; explicit preflight excludes them and must include every BUILD/review change. No whole-worktree PASS is claimed.

## Disposition of the five worker notes

- Keeping `analyzer_incremental.go` unchanged is correct scope discipline; the unreferenced isOrdinaryIncremental helper is nonblocking cleanup for separate authorized work.
- Replacing the flag-combination mode test with an explicit-plan test is structurally justified; reviewed public entry methods choose fixed modes, and independently rerun R025 behavior regressions pass. Existing test fixture connector change enables storage-location cases and was also inspected. This does not resolve the contract failures above.
- `NewAnalyzerWithProvider` and `newTriggerAnalyzer` are narrow authorized test seams; production factory stays NewAnalyzer, reference inspection shows synthetic injection in tests. Their current route test still misses required conditional rerun proof.
- Event-time since_last residual (older/equal/backdated source omitted) is accepted by SPEC; no source-version parity or F02 channel completeness claim follows.
- Inline dialog copy respects the allowed paths; i18n cleanup is not a blocker. Complete the actual mandated wording/validation within JobDetail.vue.

## Next governed move

Claude REPAIR_WORKER / COMMIT_STEWARD executes one consolidated R027-R1 repair under the unchanged objective, risk R2, allowed paths, prohibited effects and commit ownership, acknowledges that role after rehydration, then returns one local REVIEW_PENDING commit and appended repair evidence to independent Codex re-review. Keep the retained failing probes; repair source/contradictory worker tests rather than weaken them. No new authority seed, real API call, persistent DB, F06 change, push/merge/deployment, parent-CVF work or FREEZE. R025/R026 remain REVIEW_PASS / FREEZE_OPEN; F05 acceptance stays OPEN.
