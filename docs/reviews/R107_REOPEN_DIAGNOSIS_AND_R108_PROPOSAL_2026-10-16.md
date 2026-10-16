# REOPEN read-only diagnosis and proposed bounded R108 investigation

Status: DRAFT
Risk: R2
Worker: Codex /root (ORCHESTRATOR / SPEC_AUTHOR); Luna Max read-only diagnostic support
Reviewer: Codex /root for inherited UI evidence; new diagnostic implementation requires independent review
Disposition: DIAGNOSIS_PARTIAL_NEW_RUNTIME_AUTHORITY_REQUIRED

## Completed work and evidence

At HEAD `519e47aa3b3136a8736e48643e3d512c5c29c75b`, Root and Luna independently traced saved R107 observations and source without starting a process/server/fixture, compiling, running product tests, calling a provider or changing application source. Canonical R107 continuity is REVIEW / CHANGES_REQUIRED. Doctor passed 27/27 with the historical Core pin warning; Core remains read-only. Root source/blob and original-log receipt: `probes/r107_root/reopen_static_diagnosis.json`.

The engine, action handler and R101 fixture test source are byte-identical between historical target `200df3e4a7bc2a326425922a5230508a4fd9b0fe` and HEAD. Current event model adds two nullable R102 linkage fields; this does not prove the actual stopped fixture schema or explain the observed failure. No schema/index/trigger inventory was recorded at the failed request. Historical binary evidence cannot certify the current backend.

Original backend stdout has five actual case POST requests, distinguished from the route-registration debug line: CORRECT 200 (line160), stale CONFIRM409 (162), ASSIGN200 (173), RESOLVE200 (184), REOPEN503 (207), at 14:47:42 through 14:49:55. Saved before/after observations retain RESOLVED v4 and no REOPEN history. There is no saved raw response body or underlying SQL error for B06. AX shows a pending/disabled form then reloaded v4, without a saved settled error banner. The HTTP failure is observed; the exact JSON error body is source-inferred, not captured.

## What the code establishes

| Source location | Finding | Limit |
| --- | --- | --- |
| `backend/api/handlers/intervention_m3.go:16–35` | Invalid body/action maps400; conflict maps409; other engine errors map503. | The log records status, not response body or internal cause. |
| `backend/engine/intervention_m3.go:224–237` | REOPEN from RESOLVED requires existing valid assignee and future due time; target ASSIGNED, active outcome cleared. | Prior outcome survives in the event's before projection/history. Raw B06 metadata expecting OPEN was incorrect. |
| `backend/engine/intervention_m3.go:257–259` | Case-update error/zero affected rows maps conflict409. | An update failure alone does not explain this handler's503. |
| `backend/engine/intervention_m3.go:113–127,152–154,261–278,282–287` | Source reads, projection serialization, event insert or unclassified transaction/commit error can yield503. | No saved raw error selects one branch. Event collision, schema/charset and transient DB failures remain hypotheses. |
| `backend/api/handlers/intervention_m3_test.go:85` | Fixture forces GORM logger Silent before migration. | This explains missing SQL diagnostics, not the REOPEN failure itself. |
| `backend/api/handlers/intervention_m3_test.go:189–241` | Native lifecycle test includes successful REOPEN expectation. | Existing test code/prior acceptance does not prove this UI request succeeded. |

An expired/invalid deadline would ordinarily map400; stale state/version/source would ordinarily map409. These are not supported explanations for the observed503 along this mapped path. No product patch is justified from the current record. Root cause remains NOT_ESTABLISHED.

## Concrete proposed R108 scope — not dispatched

Purpose: identify the failed internal operation with one isolated synthetic native replay before deciding whether any application repair is needed. This proposal adds a diagnostic runtime campaign; it does not reset the spent R107 allowance or grant an automatic repair/UI retry.

1. Prepare a separate opt-in diagnostic fixture/test under `backend/api/handlers/intervention_reopen_r108_test.go`, reusing existing real router/MySQL and synthetic fixture semantics. Do not edit production engine/handler/model, R101 tests, old runner/seed or prior evidence. Capture GORM error diagnostics within this fixture only; never log ambient config, DSN/password, auth headers or tokens. Allowlist operation stage, MySQL number/SQLState and affected index/constraint; exclude SQL binds and unsafe error text. Record table column/index/trigger metadata before and after any current-source fixture migration. Do not inject a failure and then call it the R107 cause.
2. Root independently reviews the diagnostic source, exact invocation and fixture binding before launch. Build one new test binary from the explicitly recorded current source including diagnostic fixture; never relabel the old binary as current. Record the old/current schema representation difference separately.
3. One start of the existing disposable R101 MySQL container only after exact identity/ownership and protected-inventory checks. Container `e1555e03d69360e1cd7729afd9e52856443f5172d76454653beec981a7d6477a`, image `sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b`, volume/name `ccmai-r101-review-final-20261015`, network `35c3959ac41517050defad05cbad7a08717b24c8879683e25baf3e2702625e98`, ownership label `ccmai.tranche=CCMAI-RUNTIME-101` must match; any drift is HOLD. Zero Docker create/recreate/pull/prune, zero changes to R102 or user app/DB. New synthetic tenant/case IDs only; retain R107 rows. Run one native diagnostic invocation against this fixture, with at most four action POSTs: CORRECT → ASSIGN → RESOLVE → REOPEN, including the saved B06 Unicode reason and a relative future deadline. Fixture seed writes are separate; each successful POST includes projection/event writes. Record exact synthetic action fields, HTTP status, sanitized response/error fields, and before/after case/event/proposal counts; no credential-bearing request dump. Maximum campaign600s from before the first effect, including compile, fixture start, invocation, capture and cleanup; no retries, provider/mock calls0, browser/backend-listener/Vite launches0, installs0. Source compile ceiling1. Always clean only owned processes and stop the fixture only if this campaign started it; record cleanup receipt and protected safe-field comparison.
4. If replay reproduces503, return the original diagnostic error and rollback state for independent review. If it succeeds, return NOT_REPRODUCED and the schema/source/fixture differences; do not call the R107 defect fixed. Either result stops this diagnostic scope. No speculative application patch, second replay or automatic browser campaign.

The packet is reviewable but remains DRAFT: no dispatcher seed, committed base, active work order or runtime admission has been issued for R108. A fresh finite allowance is needed because R107 explicitly has runtimeRetries0 and its one campaign is spent. Original authority/evidence/counters remain immutable.

## Remaining UI acceptance contract

After the diagnostic disposition and any separately admitted repair, a source-bound UI campaign must state whether it targets historical R101 or current R102, with matching new build/hash. Reuse the six positive R107 observations only within their historical claim boundary; do not rerun them merely for volume.

| Item | Required observation |
| --- | --- |
| B06 | REOPEN once from v4, success only if ASSIGNED v5 with new due time and cleared current outcome; prior outcome remains in event before/history and proposal survives reload. Failure includes sanitized response/log and no retry. |
| B08 | Actual supported narrow viewport (proposed390×844) queue/detail/history and owner controls; capture viewport and screenshots. An untried control is NOT_RUN, never unsupported. |
| B09 | Check documented tool support first; if offline control can be exercised, real offline reload error then online recovery with no mutation replay. Otherwise NOT_RUN with the exact unexercised-control reason; no unsupported-tool claim, firewall or interception workaround. |
| B10 | Existing synthetic WAIT_DATA case as owner: assign data owner/future deadline/next action, escalate, show still WAITING_ON_DATA, retain source limitation and completion guard. Absence of ordinary CONFIRM/RESOLVE UI controls establishes UI guard only; a server rejection needs separately authorized real request proof. Reader observation from B07 does not satisfy this. |
| Cleanup | Final supervisor/child exit/fixture state/protected resources and actual owned-tab close results, with byte manifest. Missing receipt remains a gap. |

This document does not expand the roadmap, close M3, repair the three Dashboard baseline failures, or assert AI governance behavior. R102 local frozen acceptance and parked Facebook/Zalo OA are unchanged.

## Static proposal checks

`probes/r107_root/diagnosis_preflight.*`: exit0, provenance/continuity/tranche/claims/secrets/workflows/catalog PASS7/7. `diagnosis_reviewer_fast.*`: exit0, INST/CONT/RANGE/ROUTE/ROLE/CLAIM PASS; APPL NOT_APPLICABLE_WITH_REASON because R107 is BLOCKED_WITH_REASON, closeability not checked. No failed control was emitted. These checks do not admit the draft for execution, establish the REOPEN root cause, or replace independent review of the future diagnostic source. The prior 89 gate units and 13 source/module checks are inherited unchanged, not replayed for this document-only work.

Luna Max independently cross-checked the draft against source and B06 AX. Root incorporated its clarifications on action-POST vs underlying-write counts, the full campaign clock, exact fixture identity, allowlisted diagnostics, B06 active vs historical outcome, and B09 unexercised-control wording. This is read-only diagnostic support, not acceptance of an unimplemented diagnostic test or a runtime dispatch.
