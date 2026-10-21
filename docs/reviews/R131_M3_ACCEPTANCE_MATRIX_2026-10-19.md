# R131 M3 Acceptance Matrix — 2026-10-19

Status: COMPLETE_PENDING_REVIEW
Risk: R1
Worker: Luna Max evidence analyst
Reviewer: Root independent reviewer
Scope: documentation-only reconciliation of M3-01 through M3-08; no product/runtime/provider operation or test execution in R131.

## Evidence basis and applicability

The acceptance contract is the M3 rows in `docs/reviews/planning/CCMA_PRODUCT_DELIVERY_ROADMAP_2026-10-13.md`, R101 H01–H08 in `docs/specs/M3_HUMAN_WORKFLOW_R101_SPEC_2026-10-15.md`, and the B01–B09 browser checklist in `docs/reviews/planning/R101_CLAUDE_BROWSER_EVIDENCE_HANDOFF_2026-10-15.md`. Test definitions below are identified separately from their executed receipts. R127 and R128 are inherited only within their accepted current-source synthetic workflow scopes.

R128's `docs/reviews/probes/reader_names_20261019/evidence_manifest.json` and `original_verification.zip` include the source identity and 414 source records. The in-memory comparison found 411 raw SHA matches, three frontend intervention files whose raw difference is CRLF/LF normalization only (CRLF-to-LF hashes match), and zero content mismatches. This establishes current content applicability after the R130 history rewrite; historical source commit IDs are not asserted to be ancestors. The product source tree remains unchanged in R131.

R128 `verification_summary.json` records `nativeExit: 0` for the seven named native test groups, `frontendRelevantTests: 21`, `frontendTestFiles: 2`, `frontendBuildExit: 0`, and `browserExit: 0`. Its browser record has seven action POSTs, four auth POSTs, and the exact outcomes used below. R127's first fixture run (`firstRunExit: 1`) and its accepted second run (`secondRunExit: 0`) remain preserved in `review_summary.json`; the first-run fixture-pointer mismatch is not erased or relabeled. Neither review is a full backend/frontend suite or full M3 acceptance.

## M3-01 — Queue priority, evidence state, and pagination

**Disposition: PARTIAL.** The current source contract is honest chronological order, not business priority. `backend/api/handlers/intervention.go::ListInterventionCases` filters tenant/state, computes `total` before cursor, orders `created_at ASC, id ASC`, returns `has_more` and `next_cursor`, and labels the order `created_at_asc_id_asc`. `frontend/src/views/InterventionCases.vue` shows total, the localized “Ordered by creation time” label, creation/due dates, an overdue chip, and a More control. R101 explicitly says deadline/urgency must be understandable and a UUID order must not be called priority; no M1 priority rule is inferred.

Executed evidence: R094's final native route evidence exercised a bounded limit/cursor response (worker return final Go command exit 0; `R094_M2_INTERVENTION_WORKER_RETURN_2026-10-14.md`, M2A07). That same return explicitly records foreign-tenant source/detail and multi-page count/order consistency as `NOT_RUN`. R127/R128 browser receipts do not add a multi-page queue run. The implementation shows the cursor contract, but test definitions or a single-page response do not prove cross-page count/order behavior.

**Gap:** the required current UI/API multi-page count/order proof is absent. A business urgency ranking also requires a separate M1 decision; the current oldest-first label and due/overdue display must remain described as such.

## M3-02 — Confirm, reject, correct, rights, and reason

**Disposition: PARTIAL.** `backend/engine/intervention_m3.go::M3ApplyAction` implements `CONFIRM`, `CORRECT`, and `REJECT` with role and case-state checks; the current view offers those actions for pending review and requires reasons. R128's executed `TestR101HumanWorkflowNative` and `TestR101SourceAndPermissionConflicts` are in `nativeTopLevelTests` with `nativeExit: 0`. R127/R128 browser receipts execute `CORRECT` (200), preserve the original proposal and history, and observe a stale `CONFIRM` (409). The R128 browser terminal reports `proposalUnchanged: true`, `staleActionConflict: true`, `sourceIdentityUnchanged: true`, and six new human events. R117 separately supplies accepted backend-only synthetic `CONFIRM` execution evidence; it is not browser proof.

**Gap:** source support and the UI option do not prove an accepted REJECT. No positive `"REJECT"` action appears in `backend/**/*_test.go`, and neither accepted browser action list includes it. Preserve the scope of the executed CORRECT/CONFIRM/denial evidence; do not infer REJECT acceptance from the switch branch.

## M3-03 — Snapshot/version currentness and competing reviewers

**Disposition: ACCEPTED_BOUNDED.** `backend/engine/intervention_m3.go::M3ApplyAction` checks the expected case version, snapshot identity/digest, and proposal revision under the serialized write path before atomically updating the projection and appending a human event. Its source comment limits currentness to the local source and explicitly excludes out-of-band SQL/upstream history. R128's executed `TestR101SourceAndPermissionConflicts` and `TestR101HumanWorkflowNative` passed (`nativeExit: 0`); the browser observed a stale `CONFIRM` as 409 and preserved the previous state. R127/R128 review receipts retain the same current-source content qualification recorded above.

Inherited concurrency proof: Root's independent R131 review streamed the retained local `.ccmai_local_evidence/review_completion_20261018/local_verification_originals.zip::go-test-r125.json` without extracting it into the repository. Its SHA-256 `a5f7abd409d6319cfe1b351c8d444f9641c6e11c5f1ecfa27138192c5cab2ff0` matches `docs/reviews/probes/review_completion_20261018/latest_package_coverage_manifest.json` at `packageProvenance["github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/handlers"].rawSHA256`; the stream records `TestR101CompetingReviewers` PASS, elapsed 0.64s. R126 accepted the complete handlers package from that original run. R126 also records the original full R125 command exited 1 for separate engine/MCP failures, so this does not promote that full invocation to green. The exact test function and `backend/engine/intervention_m3.go` blob are unchanged between R126 source `9ab54062976de181ffbb910be44393021d1bc3db` and current HEAD. This is applicable inherited package/test evidence, not a fresh R128 concurrency run.

The accepted bound remains local-source/version and tested concurrency semantics; upstream history and out-of-band SQL are outside the source contract.

## M3-04 — Assignment, deadline, outcome, and reopen history

**Disposition: ACCEPTED_BOUNDED.** `M3ApplyAction` and `InterventionCases.vue` implement assignment, deadline, resolve, and reopen; the executed `TestR101HumanWorkflowNative` checks an invalid assignee (400), valid assignment, resolution, reopening, preserved proposal, and human-event history. R127 and R128 actual browser receipts both record `CORRECT 200 → stale CONFIRM 409 → ASSIGN 200 → RESOLVE 200 → REOPEN 200`; reload shows the reopened ASSIGNED state with the former outcome in history. R128's `browserTerminal.case.history` records versions 2–5, with `priorOutcomePresent: true` on REOPEN. R127's WAIT_DATA assignment/escalation and R128's `waitCase` remain `WAITING_ON_DATA`, not resolved.

This is bounded synthetic workflow evidence. It does not decide business SLA/deadline policy, customer outcomes, or M1 workflow rules.

## M3-05 — WAIT_DATA supplement, rerun, and escalation

**Disposition: PARTIAL.** `backend/api/handlers/intervention_waitdata.go` implements `CreateInterventionDataRevision`, `CreateInterventionRerunRequest`, and `ExecuteInterventionRerun`; `backend/api/router.go` registers revision and request reads/writes plus explicit execution. R102's accepted local revision/request evidence only persisted an explicit `AWAITING_EXECUTION_AUTHORITY/NOT_RUN` intent. R117 later supplies a separate accepted backend-only synthetic executor receipt with one real provider request/response and a new linked run/snapshot/proposal; that evidence is not a current UI path, a live channel, or an M3 browser acceptance.

The current browser evidence proves only `ASSIGN_DATA 200` and `ESCALATE_DATA 200`, leaving the case in `WAITING_ON_DATA` without an outcome. In `frontend/src/api/intervention.ts`, the exported functions are only list/detail/assignees/apply-action; `frontend/src/views/InterventionCases.vue` displays “Assign a follow-up or escalate the request to continue” and offers only those WAIT_DATA actions. It has no data-revision, rerun-request, or execute API method/control.

**Gap:** users lack the current UI flow to submit and inspect a source revision and create/read an explicit rerun request. The executed R117 backend path does not fill this user-visible gap. A future execution control needs its own explicit user and authority contract; R131 authorizes none.

## M3-06 — Async route, tenant, case, session, and permission lifetime

**Disposition: ACCEPTED_BOUNDED.** `frontend/src/views/InterventionCases.vue` binds `currentLifetime` to tenant, case, access token, role, and permissions; `stillCurrent`, generation tracking, `AbortController`, route/session/permission `watch`, and `onBeforeUnmount` block late responses from updating a changed lifetime. R128's two focused frontend files/21 executed tests passed (`frontendBuildExit: 0`); `intervention-ui.test.ts` includes stale case-A-after-navigation, session/permission change clearing, duplicate-submit prevention, and a late action after navigation. R127/R128 actual browser receipts additionally observed owner/reader logout, and the reader view had no action/save controls. These are focused UI tests and bounded browser observations, not proof of every conceivable timing interleaving.

## M3-07 — Notification state separate from business state

**Disposition: NOT_APPLICABLE_WITH_REASON.** R101's M3 contract explicitly says no notification is sent and notification state cannot decide business state (`M3_HUMAN_WORKFLOW_R101_SPEC_2026-10-15.md`). The current M3 handlers, engine, and view expose no notification workflow. Therefore there is no M3 notification side effect to accept or reject here. This does not accept a general notification system or any M1 notification policy.

## M3-08 — Actual browser/backend workflow, failure cases, persistence, language, viewport

**Disposition: PARTIAL.** R127's accepted browser review records `secondRunExit: 0`, actual browser action results for CORRECT/ASSIGN/RESOLVE/REOPEN and WAIT_DATA ASSIGN_DATA/ESCALATE_DATA, stale CONFIRM 409, reader write 403 through the authenticated native route, reload-persistent history, and unchanged source/proposal fingerprints. R128's actual browser fixture records `browserExit: 0`, those same seven action outcomes, four auth POSTs, zero provider/accounting deltas, `readerWriteDeniedNative: true`, and `serverStopped: true`. The accepted R128 reader view showed names after fresh login/reload with zero action controls. Both receipts are attached to synthetic fixture data only.

**Gap:** no inherited receipt records a distinct backend/write failure shown end-to-end in the actual browser; passing mocked UI failure tests and a source test definition do not fill that evidence gap. R127 records a CSS viewport of 390×844, but its narrow JPEG content raster is 375px wide, so no exact 390px raster claim is made. B09 offline remains `NOT_RUN_UNSUPPORTED_BROWSER_CONTROL`; that unsupported offline check is preserved and is not treated as the sole M3 blocker. The M3 row still has the separate browser-failure evidence gap above.

## Bounded disposition and next move

**Overall M3 status: PARTIAL; not a milestone acceptance.** M3-03, M3-04, M3-06, and M3-07 are accepted only within their stated bounds; M3-01, M3-02, M3-05, and M3-08 retain the listed gaps. R104 is diagnostic/browser-bridge provenance only and is not M3 product-browser acceptance. R117's real-provider backend executor evidence does not establish current UI integration or CVF governance behavior. M1 business decisions, M2 live-channel operation, M4 pilot, hosted CI, release, and deployment remain separate and unaccepted.

One finite next product move: prepare a separate R2 SPEC and work order for a **request-only WAIT_DATA UI path** over the already registered data-revision and rerun-request APIs, including revision/request status and history, with no automatic rerun and no provider/live execution in that scoped move. Review the execution path and its authorization separately before exposing any execute control. This move addresses the concrete current user-visible M3-05 gap; it does not close the remaining M3-01/02/08 proof rows and does not grant implementation authority through R131.

R131 worker return: COMPLETE_PENDING_REVIEW. Root remains the independent reviewer and decides the final M3 disposition.
