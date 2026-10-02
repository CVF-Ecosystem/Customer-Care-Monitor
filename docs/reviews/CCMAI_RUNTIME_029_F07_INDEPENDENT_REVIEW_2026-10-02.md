# R029 / F07 independent review

Date: 2026-10-02. Reviewer: Codex, independent of Claude IMPLEMENTATION_WORKER. Exact BUILD: `4c6653021878827cba678adb1ae87e9a196d5e85`. Disposition: REVIEW_PASS / FREEZE_OPEN. Risk ceiling: R2.

Authority: [SPEC](../specs/RUNTIME_DASHBOARD_SERVICE_STATUS_F07_2026-10-02.md), [work order](../work_orders/CCMAI_RUNTIME_029.md), immutable seed `CVF_SESSION/authority/CCMAI-RUNTIME-029.json`, base `0f442f238cbe68a89b47dc43d99f2ffe3fa8b997`. [Worker evidence](RUNTIME_DASHBOARD_SERVICE_STATUS_F07_BUILD_2026-10-02.md). This accepts the bounded frontend unknown-status contract; it does not measure service health or prove runtime CVF governance.

## Source and provenance

All four changed frontend files and four committed capture images are identical to the exact BUILD. Subsequent learning commits change documentation only. The BUILD diff changes only service-card names/rendering, two scoped locale keys and a new mounted test file, plus authorized evidence/continuity/roadmap/order artifacts. Existing QC/date/i18n tests, Dashboard requests, metrics, date filters, latest-request guard, demo flows and navigation remain unchanged.

Seed bytes equal their first committed content and exist at baseCommit before BUILD. Codex dispatcher authorship is recorded in planning/handoff; Claude worker ownership is supported by BUILD/handoff attribution. A shared Git identity alone does not prove role identity or intra-worktree timing. Worker records a before-edit acknowledgment; no universal procedural-compliance claim is issued.

## Requirements evaluation

| Requirement / group | Independent assessment |
| --- | --- |
| F07-01 / A | Service names contain no health state; explicit unknown chips exist on first render, pending and resolved response. Mounted deferred case passes. |
| F07-02/03 / E | Exact scoped VI/EN text, visible explanation, neutral heading/chips and help icons. Locale/class tests pass. Four committed crops visually inspected: readable desktop/mobile, no clipped service-card text or fabricated green/down state. |
| F07-04 / B/C | Positive/zero/empty data, network/timeout/500/503 and repeated success/failure cases pass. Fixed service-card presentation is independent of request outcomes. Synthetic failures are UI fixtures, not measured outages. |
| F07-05 / D | Incidental health and scheduler hints are ignored. Timer-advance/request inventory test passes. Source contains only existing Dashboard/demo GET and user-triggered demo POST/DELETE; no health call, new polling, timer or timestamp. |
| F07-06 / F | Source diff preserves unrelated behavior and existing regression files. Full frontend suite passes, including QC, Vietnam business-day and i18n regressions. |
| F07-07 / F | Matrix, capture links, request inventory and limits are present. Old source and one required success-color mutation independently fail; byte-restored focused baseline passes. Worker results and independent checks remain distinguished. |

## Independent execution

| Check | Result |
| --- | --- |
| Full frontend: `npm --prefix frontend test` | 26 files / 268 tests PASS, no failed/skipped tests, 13.20 s. |
| Forced typecheck: `npx --prefix frontend vue-tsc -b frontend/tsconfig.json --force` | PASS. |
| Production build: `npm --prefix frontend run build` | PASS, bundling 906 ms. |
| Gate tests: `python -m unittest discover -s scripts/tests -p 'test_cvf_downstream_gate*.py'` | 46/46 PASS, 20.600 s. |
| Doctor | 25/25 PASS, core matches pinned public origin/main. |
| Docs build before disposition | PASS, 5.78 s; final artifact checks recorded in handoff. |

Focused command for each detector: `node node_modules/vitest/vitest.mjs run src/__tests__/dashboard-service-status.spec.ts`, executed in frontend. Old `Dashboard.vue` from `4c66530^` with current translations/test: 13/13 FAIL, 5.05 s; representative behavioral assertion `expected +0 to be 3`. Narrow mutation: add `:color="name === 'Database' ? 'success' : undefined"` to the sole unknown chip template. Expected/actual marker matches: 1/1; changed bytes verified, SHA256 `cd286f2373557f06a1a6cf450f0c9dd9d1dec4015812b24ab693124f2ee9c212`. Mutation: 13/13 FAIL, 5.13 s; intended neutral-class assertion fails against `(text|bg)-(success|error|warning)`. KILLED, not a build/harness failure. Original file restored in finally and checked byte-for-byte; focused baseline 13/13 PASS, 5.34 s. No test/source change is committed from these temporary probes.

Sanitized logs are reviewer-local Temp files in `ccmai-f07-review-rdht54t6`: `old-source.log`, `database-success-mutation.log`, `restored-baseline.log`. Commands, mutation and failing assertions are preserved here for reproduction. The worker's other three mutations, full-page capture reports, environment lifecycle and cleanup remain worker evidence; they were not independently rerun.

## Documentation findings and learning

The current roadmap row retained “chưa sửa source” alongside REVIEW_PENDING after BUILD. Reviewer reported BLOCKED_CONTINUITY_DRIFT, corrected this current pointer and rehydrated before formal disposition. No product fix; this extends the existing [downstream current-prose learning](learnings/CCMAI_TO_CVF_DOWNSTREAM_GATE_LEARNING_INTAKE_2026-10-01.md). Parent assessment remains deferred.

Worker evidence's DB-fixture NOT RUN phrase was clarified by a disclosed reviewer addendum: backend integration fixtures were not run, but the screenshot environment used disposable MySQL in tmpfs and generated throwaway local secrets. That already-disclosed capture environment is not a persistent DB or real provider/channel call. Reviewer did not recreate it or inspect credentials. Preserve this distinction rather than claiming that no DB was used anywhere.

## Limits and next move

Review asserts the UI explicitly lacks measured health. Backend probes/telemetry, real API/DB/Scheduler availability, governance proof, provider/channel calls, backend suite/R019/race and new screenshots were NOT RUN in this review. No persistent DB, parent edit, push, merge, deployment or FREEZE. R028/F06, R027/F05 and R025/R026 retain their separate review dispositions and limits.

REVIEWER -> ORCHESTRATOR / SESSION_SYNC_STEWARD / COMMIT_STEWARD for disposition synchronization and local review commit. No F07 repair remains dispatched. Next governed move is assess remaining F02 live/message evidence and FREEZE readiness under separate bounded authority; this review grants no new API/provider/credential/deployment action. F07 presentation remediation is accepted; FREEZE remains OPEN.

## Final disposition verification

Complete 30-path seed/planning/BUILD/learning/review/disposition explicit preflight: 7/7 PASS including catalog; final docs build PASS (6.07 s), diff check PASS. Existing reviews family covers this record, so index/catalog remain unchanged. Default and origin/main..HEAD preflights FAIL (6/7) solely on pre-existing `knowledge/_index.json` and two Python bytecode files outside authority, excluded from the local review commit. No whole-worktree/PR gate or hosted-CI PASS claim. Product source and seed remain unchanged.
