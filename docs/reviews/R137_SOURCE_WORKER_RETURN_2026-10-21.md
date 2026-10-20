# R137 source worker return — 2026-10-21

Status: COMPLETE_PENDING_REVIEW
Risk: R2
Worker: Sol Medium delegated implementation worker / round-1 repair worker
Reviewer: Root independent REVIEWER
Role: Sol Medium IMPLEMENTATION_WORKER, then round-1 REPAIR_WORKER. Root retains independent REVIEWER and sole COMMIT_STEWARD.
Authority: committed R137 seed `7d468de3e9c238616c42fba6e0a4a3ebff81756c`; source dispatch HEAD `aca012104d04db2847b2d1c04efd34bfc3e87537`.
Source hashes and worker counters: `docs/reviews/probes/r137_execution_read_ui_20261020/worker_source_identity.json`.

## Delivered source scope

- Added two GET routes with earliest router `Cache-Control: no-store`: writer readiness requires intervention case read/write and jobs write; reader status requires intervention case read. New handlers map malformed, forbidden, missing and unavailable reads to bounded responses.
- Readiness uses read-only SQL transactions and plain reads for membership, case/request, existing execution, synthetic process configuration, job/channel/source limits, original snapshot, latest revision, proposal/running job, actor-bound grant and budget. READY exposes one opaque handle and observed deadline/version only for the matching actor. Known missing prerequisites remain non-ready; SQL uncertainty returns 503. This is advisory; locked POST admission remains authoritative.
- Status reads the immutable request intent and unique execution row even if transport is disabled. Publication requires the linked successful JobRun, proposal, `EXECUTE_RERUN` event and event projection, bound to request/execution actor, job, conversation, revision and original snapshot. No-row remains an observation, never a no-call claim.
- Added typed frontend GETs and exact two-field one-shot execute POST. Case detail mounts request status/history after `WAITING_ON_DATA` transitions to `OPEN`. The component validates its session/actor/permissions/case-version lifetime, refreshes detail/readiness before confirmation, persists a non-secret origin-local marker before POST, and blocks resend after uncertain completion or reload. Status observation is capped at 12 reads, at least five seconds apart, with a separate one-minute abort deadline for a stalled read; server grant deadline uses a conservative monotonic client interval. UI copy is user-facing in Vietnamese and English.
- Pure Go and frontend component/transport tests cover DTO omissions, identity/link mutations, known missing versus failed read, reader control absence, failed marker persistence, no-row uncertainty across remount, duplicate click, deadline expiry during confirmation, late actor/permission response, stale READY clearing, and exact one-shot request shape. Mocks exercise UI structure/lifetime only, not governance or AI behavior.

Worker source/test paths: `backend/api/router.go`, `backend/api/handlers/intervention_execution_read.go`, `backend/engine/intervention_execution_read.go`, `backend/engine/intervention_execution_read_test.go`, `frontend/src/api/intervention.ts`, `frontend/src/views/InterventionCases.vue`, `frontend/src/components/InterventionExecutionStatus.vue`, `frontend/src/components/interventionExecutionHold.ts`, `frontend/src/i18n/vi.ts`, `frontend/src/i18n/en.ts`, `frontend/src/__tests__/intervention-api.test.ts`, `frontend/src/__tests__/intervention-execution-status.test.ts`, `frontend/src/__tests__/intervention-ui.test.ts`. Existing executor, grant issuer, startup, POST handler and model/schema files have no worker edit.

## Exact local checks and preserved failures

All commands ran from project root with installed caches, no downloads or DB/server/browser start. Both Go commands used `GOPROXY=off`, `GOSUMDB=off`, `GOTOOLCHAIN=local`.

| Worker cap | Actual invocation and result |
| --- | --- |
| Go 1/6 | `go -C backend test ./api ./api/handlers ./engine -run '^TestExecutionReadPure' -count=1` — exit 0. API/handlers compiled; only selected engine pure tests ran. |
| Go 2/6 | Same command after source/link changes — exit 0; four selected pure engine tests pass. No DB test ran. |
| Node 1/6 | `npm --offline --prefix frontend test -- intervention-execution-status.test.ts` — exit 1; four tests failed because happy-dom has no `window.confirm` function for `vi.spyOn`. Fixed the test fixture. |
| Node 2/6 | `npm --offline --prefix frontend test -- intervention-execution-status.test.ts intervention-api.test.ts` — exit 1; API file passed, three component tests failed after a `Storage.prototype.setItem` spy leaked across tests. Explicitly restored the spy. |
| Node 3/6 | `npm --offline --prefix frontend test -- intervention-execution-status.test.ts -t 'uses exactly'` — exit 0; one selected pass, seven skips (diagnostic only). |
| Node 4/6 | `npm --offline --prefix frontend test -- intervention-execution-status.test.ts intervention-api.test.ts` — exit 0; 12/12 pass across two files. |
| Root independent Node 1/4 | Expanded three-file run — exit 1, 33/34 pass; existing `intervention-ui.test.ts` saved-notice fixture expected one request POST but got zero after the new sibling consumed a count-sequenced mock response. Recorded in Root review; this is the round-1 F1 trigger. |
| Node 5/6 | `npm --offline --prefix frontend test -- intervention-execution-status.test.ts intervention-api.test.ts intervention-ui.test.ts` — exit 0; 36/36 pass across three files after round-1 repair. |
| Frontend build 1/2 | `npm --offline --prefix frontend run build` — exit 0; `vue-tsc -b` and Vite build passed. |
| Frontend build 2/2 | Same command after round-1 repair — exit 0; `vue-tsc -b` and Vite build passed. |
| Diff check | `git diff --check` — exit 0 after removing an acknowledgment EOF blank line; line-ending warnings only. |

The original two worker Node failures and Root's expanded test failure remain recorded above. No broad Go suite, existing MySQL integration suite, real-router DB run, product browser, provider call or actual execute route was run. Source-only tests and build do not establish E06, AI execution, CVF runtime governance, full M3, hosted CI or release readiness. Zero DB fixtures, server/Vite starts, browser campaigns, provider calls, credential use, grant effects, execute effects, push/merge/deploy. Historical R117 1/1 call remains consumed.

## Consolidated round-1 repair response

F1: The mounted-parent follow-up fixture now returns revisions/requests from explicit persisted state that changes only when the mocked capture/request action completes. The new sibling can read at any time without prematurely exposing a request. The test module mocks the new status/readiness/execute API exports and asserts request history survives on an `OPEN` parent for a reader with zero execute controls. This repairs the Root 33/34 failure without weakening the saved-notice assertions.

F2: The poll uses a monotonic 60-second deadline and a poll-local `AbortController` linked to the component controller. Its timer aborts even a stalled status GET at the deadline; every read checks the poll signal before applying a result. The timer and parent abort listener are removed in `finally`, and the component controller remains usable for later manual refresh. A focused component test holds a status read until the deadline, verifies abort and zero resend, then performs manual refresh. No `Date.now` rollback can lengthen the polling loop.

## Return-time closeability recheck

`pre_implementation_autorun`: PASS, delegated BUILD acknowledgment appended to active handoff before source; round-1 REPAIR acknowledgment appended before repair. `focused_checker_tests`: PASS for selected source/component scope, with original failed Node invocations and Root's 33/34 finding retained. `adif_integrity`: PASS for retaining failures and E06 boundaries. `worker_return_fast`, independent reviewer gate, pre-commit and terminal closure remain Root-owned PENDING. One consolidated same-scope repair round used; no scope/effect widening or round-three condition arose. Disposition requested: independent source/component REVIEW; actual read-API/runtime/effect acceptance remains OPEN for a separate authorized campaign.
