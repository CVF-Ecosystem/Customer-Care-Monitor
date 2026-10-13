# R091 Outbound Delivery Worker Return

Status: COMPLETE_PENDING_REVIEW  
Self-declared worker-return artifact: yes  
Responds to work order: `docs/work_orders/CCMAI_RUNTIME_091_OUTBOUND_DELIVERY.md`  
dispatchWorkOrder: `docs/work_orders/CCMAI_RUNTIME_091_OUTBOUND_DELIVERY.md`  
executionBaseHead: `03e68e1ad91f74bb153e7fa5c8b646001b9f78cc`  
Risk: R2  
Worker: `/root/outbound_intake`, `INTERNAL_AGENT` implementation/repair worker  
Reviewer: Codex `/root`, Local independent reviewer and commit steward  
Decision owner: Local orchestrator / reviewer  
Seed: `CVF_SESSION/authority/CCMAI-RUNTIME-091.json`, immutable commit `5441a18116e4b897549750cd7317ffdb8ae9e74b`  
Model / effort: `gpt-6-luna`, max; session-reported, not provider-verified. Numeric token and bill usage were not exposed to this worker.  
Counts: one internal worker, one Local reviewer, zero external research agents; zero provider API or customer/account calls.

## Purpose

This return submits the bounded R091 C outbound-delivery, SMTP deadline, batch-bookkeeping, and three named R090 dependency changes for independent review. The intended source behavior is in place and the admitted synthetic Go/UI/type/build checks pass. Root owns final technical disposition, commit, and any publication. This worker has not committed, pushed, self-reviewed, or changed the private CVF Core.

Root Cause Cluster: `CCMAI-OUTBOUND-DELIVERY-BACKOFF`; rework generation `0`; one admitted implementation package; no successor tranche opened by this worker. `terminalReadinessVerdict: READY_FOR_REVIEW`. `adversarialRegressionDisposition: PASS_TARGETED_DEFECT_CLASS`. `consolidatedDefectClassSweep: COMPLETE_ALL_KNOWN_DEPENDENCIES`. `implementationAutonomyDisposition: CONTRACT_AUTHORITY_EVIDENCE_OUTCOME_ONLY`.

## Scope / Methodology

The worker used the immutable order and accepted R091 spec, kept production edits within the declared backend/frontend/helper path families, and used synthetic transports and owned disposable MySQL only. The original old-source campaign ran before production edits. A later, separately named old-source overlay restored the exact dispatch-HEAD blobs after edits for the mixed-delivery and provider-error detectors; it is supplementary evidence, not a pre-edit baseline. After Root identified a missing F07 parse/save common-tail assertion, the worker added only one bounded test to the already-authorized analyzer test file and ran only that new test on a second uniquely owned fixture.

The initial front-door mismatch and harness/UI capture problems remain preserved as failures or unknowns with no test credit. They were corrected only within their named scope. The original fixture and all lifecycle receipts were byte-preserved before the second fixture reused the helper's default manifest path. Evidence and raw/LF-normalized source hashes are bound in `docs/reviews/probes/r091_outbound_delivery/source_evidence_hashes.json`.

## Findings / Position

- **F04 — notification truth and bounded retry.** A transactional per-run manifest freezes the tenant/job/run binding, exact captured result IDs, output identities, and configuration. Output identity remains stable across output reordering and credential rotation; duplicate destinations at separate configured positions remain distinct outputs. Concurrent claimers have a unique per-output receipt claim. A confirmed `sent` output is skipped on reentry; only a `failed` outcome that proves nonacceptance is retryable. A `sending` receipt left by a crash and an `unknown` transport outcome are not automatically retried. New results or changed output configuration cannot extend or replace the frozen manifest. `NotifiedAt` is updated only for the captured result IDs and only after every frozen output is confirmed sent. Query, claim, log, outcome, and marker failures do not produce a false delivered/success result. An empty output set creates neither a completion manifest nor a notification marker.
- **F04 — provider outcomes.** Telegram returns typed outcomes: decoded API `ok=true` is confirmed acceptance, explicit API rejection is `failed`, and network/read/decode ambiguity is `unknown`. Error descriptions are secret-safe and omit request URL/token/body. Email records confirmed acceptance only after the SMTP final DATA reply is `250`; a later QUIT failure cannot reverse that acceptance. Explicit SMTP rejection is `failed`; ambiguous connection loss after DATA is `unknown`.
- **F05 — SMTP lifetime.** The SMTP client uses the earlier of caller cancellation/deadline and a 30-second default deadline. Cancellation closes the connection so a blocked greeting, command, or DATA read returns; the Analyzer owner remains held through notification completion and releases after this bounded operation returns. Adapter-specific retry behavior is unchanged.
- **F07 — common batch tail.** Provider-error early continuation was removed. Provider, parse, and persistence failures update the shared progress summary before using the same cancellable delay tail. The first two consecutive error batches use 10 seconds, the third and later use 30 seconds; a healthy batch resets the streak and uses the 2-second healthy delay. The sleep seam preserves the run context and allows deterministic delay assertions. `backend/ai/retry.go` is unchanged.
- **R090 admitted dependencies.** The notification-log route metadata is now aligned with the `settings` permission domain and an actual registered-route guard test covers allow/deny cases. The two pre-existing dashboard UI tests received only owner/Pinia identity setup needed to mount their existing assertions; assertion intent was retained. The exact R090 backend JSON selector now proves all named cases execute with zero skips.
- **UI status.** Notification logs render localized `sending` and `unknown` states with neutral status styling; unknown remains visibly unknown and does not fall through to a success style.

The reviewed source candidate remains `COMPLETE_PENDING_REVIEW`; no user retry endpoint, scheduler, recovery sweep, adapter retry change, or broad DB view/model redesign was added.

## Behavioral detector evidence

The first old-source Go JSON campaign, `old_source_negative.log`, invoked the actual dispatch-HEAD implementation before production edits. It failed the intended all-output delivery-truth detector, the Telegram secret-safe error detector, both bounded SMTP cancellation detectors after their short assertion window, and the provider-error batch progress/backoff detector. The mixed SMTP case in that first run used an incorrect fake-server AUTH reply and is not credited there. The server reply was corrected to `235`; the isolated actual-old-source rerun in `old_source_mixed_corrected.log` reached post-DATA acceptance and failed the mixed-delivery completion assertion as intended.

The separately named `old_source_supplemental.log` campaign ran after implementation edits. It temporarily restored the pinned old source blobs for `dispatcher.go`, `email.go`, `telegram.go`, and `engine/analyzer.go`, verified those bindings, and then verified restoration of current source. It reproduced the mixed-delivery and provider-error common-tail failures. The timing limitation is explicit in `source_negative_supplemental_bindings.json`: this is post-edit old-source evidence, not a complete before-edit baseline. The original failures and fake-server correction history remain intact.

The SMTP negative detector first recorded that old `smtp.SendMail` missed the short context-bounded return assertion. It then closed and joined only the exact test-owned blocked sockets/server goroutines. No outer campaign timeout was treated as SMTP behavioral evidence.

## Command Evidence

Admission evidence is `worker_admission_gates.json`: independent public Core workspace doctor PASS 27/27, legacy downstream preflight PASS 7/7, and trusted explicit pre-dispatch PASS 5/5 with `work_order_status_grammar` and `closeability_contract` actually CHECKED. The first stale-bootstrap attempt is retained in `worker_bootstrap_sync_diagnostic.json`: continuity failed 6/7 before any source edit, test, or effect; its correction time was not separately captured, and the misleading timestamp was withdrawn. The metadata-only projection correction was followed by the passing admission receipt.

The initial focused current-source compile invocation is preserved in `focused_go_r091.log`. It executed the unchanged receipt migration contract test successfully but stopped the notifications/engine packages on two unused imports. The subsequent command attempt whose Windows code-page reader failed has only an unknown/no-credit receipt, `focused_go_r091_after_import_fix_runner_failure.json`. After correcting the capture helper, the final focused Go JSON command was:

```text
go test -json ./notifications ./engine -run '^TestR091|^TestRunOwnerRemainsHeldUntilContextBoundSMTPReturns$' -count=1 -p=1
```

It exited 0: 30 terminal test/subtest PASS events, 0 FAIL, 0 SKIP; both packages PASS. It covers frozen manifests, per-output sent/failed/unknown behavior, concurrent claims, persistence/query/marker errors, credential and order stability, zero outputs, Telegram outcomes and redaction, SMTP acceptance/rejection/ambiguity/cancellation/default deadline, provider-error progress/backoff, and the actual Analyzer notification-tail owner-release integration. The separate unchanged migration proof test passed in the earlier invocation before the package compile failure: `TestR091NotificationDeliveryMigrationIsIdempotentAndReceiptKeyFitsMySQL`.

The exact R090 selector passed with JSON evidence in `r090_dashboard_exact_selector.log`: 15 top-level tests (7 R090 and 8 dashboard regressions), 30 terminal test/subtest PASS events, 0 FAIL, 0 SKIP, and both packages PASS. Root independently parsed the captured JSON and confirmed digest `89128c490a10359d96ba6acbb87a0752066ed25918ab31bb09a15f6698b06c8c`.

The named F07 supplemental command was run alone after the new test change:

```text
go test -json ./engine -run '^TestR091ParseAndSaveBatchFailuresPersistProgressAndUseCommonBackoffTail$' -count=1 -p=1
```

It exited 0: one parent test and two subtests (`parse`, `save`) passed; 0 FAIL, 0 SKIP; package PASS. For both failures the test reads the stored error summary at the common wait and before subsequent provider calls, records `[10s, 2s]` via the injected sleep seam, and confirms final accounting of one error and two successful analyses. The isolated run is `r091_f07_parse_save_common_tail.log`.

The first frontend command is preserved in `frontend_focused_tests.json` and `.log`: 18 tests across five suites passed, while the route suite failed at import with `window is not defined`, before executing assertions. The admitted route-test correction installed the existing happy-dom environment in that test only. Its exact repeat passed 5/5 cases in `frontend_route_guard_repeat.log` (settings allow, jobs deny, missing settings deny, owner allow, admin allow). Thus 23 UI assertions passed across the two retained invocations, with the failed first route suite counted only in the correction history. Forced `vue-tsc -b --force` and `vite build` each exited 0; receipts are `frontend_vue_tsc_forced.json` and `frontend_vite_build.json`.

The one early runner display failure after the successful focused Go capture is retained separately; the captured JSON/log and exit code were saved before the console echo raised a Windows encoding error. No broad duplicate Go, UI, type, or build campaign followed unchanged successful bindings.

Final return-gate evidence was collected after the metadata and test-fixture corrections. The named legacy downstream preflight passes all 7/7 gates in worker_return_preflight_final.json. The first trusted worker-return attempt is preserved as a failure because the current handoff still said REVIEW_PENDING; the current handoff must remain ACTIVE, while tranche/order remain REVIEW_PENDING. After that correction, worker-return.json reports exit 0: INST/CONT/RANGE/ROLE/CLAIM pass, and APPL is NOT_APPLICABLE_WITH_REASON because the work order is already COMPLETE_PENDING_REVIEW. The correction did not change application behavior. The only application test rerun for the preflight secret-scan correction was TestR091Telegram*, passing in the focused 	elegram_secret_fixture_repeat.json receipt. The first failed toolchain path resolution invoked no tests and is recorded there.

## Synthetic database and side effects

Fixture one used pinned `mysql:8.0` digest `sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b`, container `ccma-r091-db-fe67adefe1fd`, network `ccma-r091-net-fe67adefe1fd`, database/user `ccma_r091`, and loopback `127.0.0.1:64596`. Its anonymous `/var/lib/mysql` volume was `e9f8ec6d5ac8d3b3a9c7cf9819261511ac3f619b572146028b0b86abc2cc7a99`. The initial readiness record proves only a host TCP handshake plus an in-container app-user `SELECT 1`; authenticated host-loopback `PingContext` and `SELECT 1` were separately proven later while fixture one remained live, before the final focused Go and R090 JSON campaigns. That sequencing limitation is recorded and not retroactively relabeled as pre-baseline SQL readiness.

Root's named evidence-gap direction required fixture two, again with the same pinned image and synthetic database/user, container `ccma-r091-db-8f4167ee0ea8`, network `ccma-r091-net-8f4167ee0ea8`, loopback `127.0.0.1:64328`, and anonymous `/var/lib/mysql` volume `02d96b18a065239003e0e40e6698e3f505aab3f2a7426d3690f437fd07d2b798`. Host app-user `PingContext` plus `SELECT 1` passed before the one supplemental test. Each fixture used a generated synthetic credential held only in the ignored helper state and omitted from persisted manifests/receipts.

Both containers, both networks, and both recorded anonymous volumes were removed and independently checked absent by exact name and run label. Fixture one cleanup is `fixture_cleanup_verification_first.json`; fixture two cleanup is `fixture_cleanup_verification_second.json`. `fixture_one_archival_receipt.json` binds exact byte copies of fixture one's manifest, SQL readiness, and cleanup receipts before the default helper path was reused. The helper state file is absent. No existing application database/container, customer data, account, provider API, real SMTP server, application server, deployment, or off-host network was used; no image pull, dependency install, or broad Docker prune occurred.

## Changed Files

Worker-owned application changes are limited to:

- Backend delivery persistence and migrations: `backend/db/models/notification_delivery.go`, `backend/db/models/setting.go`, `backend/db/mysql.go`, and `backend/db/mysql_notification_receipt_contract_test.go`.
- Backend notification delivery and SMTP: `backend/notifications/dispatcher.go`, `backend/notifications/delivery_attempt.go`, `backend/notifications/delivery_contract.go`, `backend/notifications/delivery_manifest.go`, `backend/notifications/email.go`, `backend/notifications/smtp_session.go`, `backend/notifications/telegram.go`, `backend/notifications/dispatcher_delivery_baseline_test.go`, `backend/notifications/dispatcher_delivery_test.go`, `backend/notifications/email_baseline_test.go`, `backend/notifications/email_delivery_test.go`, and `backend/notifications/telegram_delivery_test.go`.
- Backend batch behavior and ownership integration: `backend/engine/analyzer.go`, `backend/engine/analyzer_batch_backoff.go`, `backend/engine/analyzer_batch_bookkeeping_contract_test.go`, `backend/engine/analyzer_batch_bookkeeping_test.go`, and `backend/engine/job_run_ownership_test.go`.
- Frontend notification state/permissions: `frontend/src/views/NotificationLogs.vue`, `frontend/src/i18n/en.ts`, `frontend/src/i18n/vi.ts`, `frontend/src/router/index.ts`, `frontend/src/__tests__/notification-delivery-status.spec.ts`, and `frontend/src/__tests__/notification-route-permissions.spec.ts`.
- The two admitted mounted-test setup adaptations: `frontend/src/__tests__/dashboard-business-day.spec.ts` and `frontend/src/__tests__/dashboard-qc-card.spec.ts`; assertion behavior was retained.
- The admitted synthetic fixture runner: `scripts/r091_outbound_fixture.py`.

R091 probe artifacts are under `docs/reviews/probes/r091_outbound_delivery/`; this return is the sole worker-return document. Root's concurrent metadata changes to `CVF_SESSION/*`, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json`, `docs/INDEX.md`, `docs/catalog/ARTIFACT_REGISTRY.json`, and the inheritance roadmap are identified separately in the operation trace and were not overwritten. The work order's status transition is the worker's authorized status-only synchronization. No file deletion or rename occurred.

## Checker Source Read-Ahead Block

Read-only source owners consulted before writing this return: target `scripts/cvf_downstream_gate.py`; public Core `governance/compat/check_worker_return_quality_gate.py`, `check_agent_operation_trace.py`, `check_delta_execution_claim_boundary.py`, `check_public_export_disposition.py`, `run_worker_return_fast_gate.py`, and `build_worker_return_skeleton_scaffold.py`; plus the accepted R091 work order and spec. The return uses the required worker-return identity and status literals and records source/claim boundaries, command evidence, changed paths, no-commit status, and public disposition. The public Core checker/standard files are read-ahead authorities only; no Core checker, gate, policy, dependency, or historical seed was edited. The work order does not install a target-specific worker-return-fast command; Root owns independent review and any corresponding gate choice.

## Agent Operation Trace Block

| Field | Observed value |
|---|---|
| Actor / agent type | `/root/outbound_intake`, `INTERNAL_AGENT`, gpt-6-luna max; one internal implementation worker |
| Decision / review owner | Codex `/root`, Local independent reviewer and commit steward |
| Session / invocation | `CCMAI-RUNTIME-091`, task `/root/outbound_intake`; exact tool commands and times are in the fixture manifests and JSON receipts |
| Working directory | `D:\UNG DUNG AI\TOOL AI 2026\CVF-Workspace\Customer-Care-Monitor` |
| Command/tool surface | PowerShell; pinned workspace doctor and public Core phase runner; bundled Go 1.26 toolchain; Node/npm frontend checks; Python fixture helper; Docker exact-owned local synthetic resources |
| Expected manifest | Immutable seed `5441a18116e4b897549750cd7317ffdb8ae9e74b`; bounded path families in the R091 work order; two named dashboard fixture-only setup files and three admitted R090 dependencies |
| Actual changed set | Listed above and bound in `source_evidence_hashes.json`; application changes remain inside admitted paths; current untracked probe/return artifacts are under the admitted review probe/return surface |
| Manifest delta | One narrow F07 parse/save integration test was added to the admitted analyzer bookkeeping test file after Root identified that the provider-only live-tail assertion did not cover those error branches. No production behavior, dependency, adapter retry, or scheduler expansion accompanied it. |
| Before / after | BUILD/ACTIVE after actual admissions; now REVIEW / REVIEW_PENDING / COMPLETE_PENDING_REVIEW after required focused proof and fixture cleanup |
| Diff evidence | `git diff --name-status`; `git status --short --untracked-files=all`; dispatch HEAD remains `03e68e1ad91f74bb153e7fa5c8b646001b9f78cc` |
| Approval boundary | Operator-authorized R091 C build; Root retained source review, final disposition, commit, and publication; provider/customer/account/deployment effects remained forbidden |
| External invocation counts | Zero external research agents, zero provider API calls, zero customer/account calls, zero real outbound notifications |
| Fixture effect counts | Two fresh labeled MySQL fixtures; host-loopback app-user SQL readiness and synthetic SQL tests only; both exact resource sets removed and verified absent |
| Deletion / rename disposition | None |
| Claim boundary | Targeted source behavior under fake providers/transports, owned loopback SMTP/HTTP, local mounted UI assertions, and disposable MySQL only; no real-provider, live-governance, hosted-CI, customer, deployed, or whole-product claim |

Tracked `git diff --name-status` at the return boundary consists of the seven current-session/work-order/implementation status surfaces; backend `setting.go`, `mysql.go`, `analyzer.go`, `job_run_ownership_test.go`, `dispatcher.go`, `email.go`, and `telegram.go`; frontend's two fixture files, two i18n files, router, and notification log view; and Root-owned `docs/INDEX.md`, `docs/catalog/ARTIFACT_REGISTRY.json`, and the inheritance roadmap. New application files are the bounded models/helpers/tests listed above. The exact untracked evidence set is recorded in the worktree output and hash manifest. No unexpected application path, deleted file, or renamed file was observed.

## Delta Execution Claim Boundary Control Block

The worker's application delta is source/test only in the R091 seed-authorized bounded path families. The supplemental host-SQL probe is retained under the R091 evidence directory and contains no credentials; the actual synthetic DSN is read from ignored helper state and is never printed or persisted. Initial and corrected harness failures remain separate receipts. Current source and proof bytes are bound in `source_evidence_hashes.json`, including raw-byte and LF-normalized SHA-256 for application source/tests and retained campaign evidence; `backend/ai/retry.go` is recorded as unchanged. The build branch has no worker commit; `git rev-parse HEAD` still equals the immutable execution base.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY. This worker did not publish or sync artifacts to the public CVF workspace. Root owns reviewed project-branch commit/publication. No private Core export, merge, deployment, or main-branch action is claimed.

## External Knowledge Intake Routing

No external research or remote repository absorption was used. The worker is a shared-workspace `INTERNAL_AGENT`; Local verifies private project state and owns final technical disposition. Public absence was not treated as evidence of private absence.

## Rescan Intelligence Hardening

No repository-wide corpus scan is claimed. Evidence gathering stayed within the exact R091 source, test, and named probe paths; independent hashes bind the resulting scope. No broad source rescan or historical-seed scan was performed.

## Corpus Completeness And Report Integrity

Not applicable: this return reports bounded implementation and focused test evidence, not a corpus scan, inventory, or “all files read” claim. Each cited detector and result is linked to its retained raw log/receipt.

## Finding-To-Governance Learning Disposition

The supplemental F07 evidence question was recorded before the second fixture and addressed through the existing allowed analyzer-test path and deterministic sleep seam. The reusable learning is bounded: verify each distinct error class that is required to converge on a common tail, not only the first observed provider-error branch. No new governance rule, checker, or CVF Core change was proposed by this worker.

## Epistemic Process Block

Observed evidence is separated from inference. Actual package/test events, exact selector counts, UI counts, fixture readiness, cleanup checks, retained setup failures, and source hashes are cited above. The initial fixture's authenticated host SQL readiness occurred after its first source-negative campaign, and the supplemental old-source overlay occurred after production edits; both limits are explicit. Synthetic tests do not establish real provider/API acceptance, deployed operation, live AI governance, or full product-stage readiness.

## Claim Boundary

This return supports the R091 targeted application behavior and the named synthetic dependencies only. It does not claim successful delivery to a real provider or customer, real SMTP behavior, deployed/runtime readiness, live governance compliance, hosted CI, complete seven-finding acceptance, stage exit, release readiness, or public export. R090 remains subject to Root's final combined disposition until this evidence is independently reviewed.

## Risk / Corrective Action

Risk remains R2 within the authorized boundary. There is no worker implementation blocker. The next allowed action is Root's independent local review of the returned source, tests, hashes, and receipts. Root may accept, require a named correction, or record a bounded residual; the worker does not pre-judge that decision.

## Machine Closure Package

Not applicable at worker return: the source candidate has not been independently accepted or committed. Root owns reviewer gates, final acceptance, material commit, publication, and terminal closure.

## No-Commit Statement

Worker made no commit or push. `git rev-parse HEAD` remains `03e68e1ad91f74bb153e7fa5c8b646001b9f78cc`; Root retains commit stewardship.

## Worker Experience Retrospective

The most useful correction was recording the first bootstrap mismatch and every capture/setup defect rather than converting them into silent retries. The initial dashboard route test failed before assertions because it imported browser history without a DOM; adding the existing happy-dom environment exposed and passed the intended permission controls. The first focused Go compile stopped on two unused imports, and two Windows console readers had encoding faults after or during capture; their original receipts remain separate from successful logs. Fixture one also lacked pre-baseline host-authenticated readiness, so that limit is stated and fixture two verified host SQL before its only test. These faults did not require wider source changes or broad test reruns.

## Current Worktree Record

```text
git status --short --untracked-files=all
```

The captured worktree contains the admitted source/test changes, R091 evidence under `docs/reviews/probes/r091_outbound_delivery/`, the worker-return document, and the separate Root-owned continuity/catalog/roadmap edits. There are no untracked `__pycache__` files, generated dependencies, or worker commits.

## Final lifecycle position

`Status: COMPLETE_PENDING_REVIEW`; current project phase `REVIEW`; tranche `REVIEW_PENDING`; order status `COMPLETE_PENDING_REVIEW`. Independent Root review is pending. The parked Facebook/Zalo account checkpoint remains parked. Numeric token/billing usage remains unavailable to this worker.

