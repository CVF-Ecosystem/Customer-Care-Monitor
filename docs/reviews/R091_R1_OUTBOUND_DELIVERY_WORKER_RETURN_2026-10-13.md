# CCMAI-RUNTIME-091 R1 outbound delivery worker return

Memory class: FULL_RECORD  
Record type: worker return  
Status: COMPLETE_PENDING_REVIEW  
Risk: R2  
Worker: /root/outbound_intake, INTERNAL_AGENT, gpt-6-luna, effort max  
Reviewer: Codex /root, independent Local reviewer and commit steward  
Phase: WORKER_RETURN  
Immutable seed: `5441a18116e4b897549750cd7317ffdb8ae9e74b`  
Execution HEAD: `03e68e1ad91f74bb153e7fa5c8b646001b9f78cc`

## Returned result

The four consolidated R091-R1 finding clusters are implemented and the named focused controls pass. This is a worker return for independent review; it does not accept the source, close the broader inheritance roadmap, or authorize a commit by this worker. Root remains responsible for source review, final disposition, commit and any publication on the already-authorized project branch.

`telegram.go` now treats only a decoded object with an explicit JSON boolean `ok:true` as confirmed sent and explicit `ok:false` as proven rejection. Missing, null, non-object, malformed and wrong-type acceptance fields remain unknown. Errors stay secret-safe. The tests cover missing/null/non-object/wrong-type envelopes and the pre-fix retry-classification defect.

`smtp_session.go` tracks the final EHLO response so `email.go` can distinguish the existing `net/smtp` HELO fallback from a successful ESMTP greeting that omitted AUTH. HELO-only delivery remains supported as before; successful ESMTP without AUTH still fails before DATA. Existing AUTH/STARTTLS handling, the bounded operation context, and post-DATA acceptance semantics remain covered by synthetic loopback tests.

`updateRunNotifiedAt` now locks and reads the exact captured result rows in the transaction, fails if any captured row is absent, updates only previously unmarked captured rows, and checks the affected count. An idempotent reentry over already-marked captured rows keeps their original timestamps and succeeds without a redundant write.

The R1 adversarial controls cover claim-persistence failure with no outbound call and pending receipt, receipt/log outcome rollback to conservative `sending` after accepted SMTP with reentry making no duplicate call, stalled EHLO and final-DATA cancellation with prompt return and server-observed close, and actual Analyzer three-error backoff/reset plus cancellation during the common wait preventing the next provider call. The current focused run also exercises the existing frozen-set, mixed/all-failed delivery, concurrent claim, SMTP acceptance boundary, cancellation, marker and parse/save common-tail controls.

## Candidate-negative evidence and correction

The old-production campaign ran before the three production edits, against the pinned current R091 candidate and with synthetic providers/transports/database only. `r1_old_source_negative.log` records 21 started tests, 13 test pass events, 8 fail events, and 0 skips. The behavioral failures were the missing/null Telegram acceptance cases, HELO fallback, and result deletion after the old unlocked marker snapshot. Parent/subtest failure events are not counted as additional defect families.

That first campaign also exposed an over-specific Analyzer test assertion: the cancellation path correctly made no next provider call, but the run's terminal status was `error`, not the test's assumed `cancelled`. The assertion was narrowed to the contract under test (wait completed, one provider call, nonnil run). The named unchanged-production repeat in `r1_analyzer_cancel_assertion_correction.log` passed all three top-level/subtest terminal events with no skip at 2026-10-12 23:31:02 UTC. The initial failure remains preserved and is not detector credit. The original baseline binding and correction are both retained; the final current bindings are in the R1 hash manifest.

## Focused final evidence

`go test -json ./notifications ./engine -run '^TestR091' -count=1` passed 53 selected test and subtest executions: 45 notification and 8 engine terminal test events, 0 failures, and 0 skips. Both packages passed. The detailed JSON event stream is `docs/reviews/probes/r091_outbound_delivery_r1/final_r1_notifications_engine.log`.

The mandatory owner-release integration `TestRunOwnerRemainsHeldUntilContextBoundSMTPReturns` was rerun separately after the SMTP fix. It passed with the synthetic MySQL fixture and loopback SMTP relay; the Analyzer kept the run owner while notification SMTP was blocked, then released it after cancellation/deadline. Its log is `docs/reviews/probes/r091_outbound_delivery_r1/final_r1_analyzer_owner_release.log`.

Return synchronization now passes the legacy seven-gate preflight. The first return-preflight attempt was 6/7 because the handoff's current Active role literal differed from session state; that exact line was aligned and the named repeat passed 7/7. Both receipts are retained as `legacy_return_preflight.json` and `legacy_return_preflight_repeat.json`. The original trusted pre-dispatch receipt remains 5/5 PASS with both APPL controls actually CHECKED before effects; the independent Root review gate is still pending.

The accepted unchanged C evidence remains available in `docs/reviews/probes/r091_outbound_delivery/`, including the original 82-entry source/evidence manifest, prior database migration proof, F07 parse/save receipt, R090 per-case JSON selector, 23 valid UI cases, forced typecheck and Vite build. No A/UI/build repeat was warranted by the R1 source changes. The Root-owned R090 final review accepts A as `REVIEW_PASS`; its 15 mandatory top-level cases and 30 total terminal events had no skip or invalid records.

## Fixture lifecycle and external effects

The same already-owned fixture run `5b005e32bafc491287e1633995591743` was reused. When inspected before the final focused runs its exact container was exited with code 255, OOM false, and no recorded error; the cause is unknown. Before restart, the worker reverified exact container/network ownership labels, the single recorded anonymous MySQL volume, and the loopback-only dynamic port binding. The same container, network and volume were restarted; no new fixture or Docker resource was created.

After restart the actual host-side app-user `PingContext` and `SELECT 1` probe passed on `127.0.0.1:63747` at 2026-10-13 00:18:46–00:18:47 UTC, before any final test command. The original pre-restart readiness is preserved as historical evidence and is not substituted for this check. The R1 fixture manifest contains no password or DSN. Cleanup removed only the exact owned container with its anonymous volume and exact network; read-only inspection verified all three resources absent. Receipts: `host_sql_readiness_after_resume.json`, `fixture_manifest.json`, and `fixture_cleanup_verification.json` in the R1 proof directory.

Real provider/API calls, real SMTP, customer runs, account access, application-server traffic, off-host network calls, deployment, commit and push were all zero. HTTP/SMTP interactions were synthetic and loopback-only; SQL ran only against this owned disposable fixture.

## Reused acceptance and limits

The first return, original 82-entry manifest, earlier proof and fixture lifecycle records remain unchanged. The R1 evidence is additive under `docs/reviews/probes/r091_outbound_delivery_r1/`; the hash manifest binds the current candidate sources/tests and new proof bytes without rewriting those earlier receipts.

The retained uncertainty boundary is intentional: a crash-left or externally ambiguous `sending`/`unknown` output is not automatically retried. These synthetic controls establish application behavior but make no claim about a real Telegram provider, real SMTP relay, customer data, production database, hosted environment or deployed product.

Public Export Disposition: DEFERRED_PRIVATE_ONLY.

## Handoff

Worker completion is `COMPLETE_PENDING_REVIEW`; the next allowed move is Root's independent R091-R1 evidence and source review. No successor round is opened by this return. The worker made no commit or push. Model/effort are session-reported, not provider-verified; numeric token and billing totals are unavailable to the worker.

## Root return-envelope reconciliation (2026-10-13)

Authored by Codex /root after independent evidence intake, not by the worker and not backdated. The original worker return is preserved byte-for-byte at docs/reviews/probes/r091_local_review/r1_worker_return_original.txt. The R1 69-entry manifest binds that original return; only this metadata addendum changes its current return path binding. All other 68 entries remain current. The worker omitted the required scalar self-proof envelope despite passing the imported gate; Root resolves those fields from the preserved order/receipts and completed source/evidence review, without production edits, new effects, or redispatch.

rootCauseClusterId: CCMAI-OUTBOUND-DELIVERY-BACKOFF
reworkGeneration: 1
consolidatedDefectClassSweep: COMPLETE_ALL_KNOWN_DEPENDENCIES
productionBindingEvidence: docs/reviews/probes/r091_outbound_delivery_r1/r1_hash_manifest.json; 69 original raw/LF bindings verified by Root, current return mapped to preserved original bytes; final_json_case_intake.json independently parses 53 focused PASS plus owner-release1PASS, zero FAIL/SKIP.
adversarialRegressionDisposition: PASS_TARGETED_DEFECT_CLASS
successorTrancheOpened: NO
implementationAutonomyDisposition: CONTRACT_AUTHORITY_EVIDENCE_OUTCOME_ONLY
internalAgentInvocationCount: 0
externalAgentInvocationCount: 0
providerInvocationCount: 0
quotaUsage: NOT_AVAILABLE_WITH_REASON: numeric worker token/billing counters not exposed
terminalReadinessVerdict: READY_FOR_REVIEW

Invocation counts refer to the worker's own child/external/provider invocations, not Root's creation/resumption of this internal worker. Model gpt-6-luna and effort max remain session-reported, not provider-verified. Required envelope absence is attributed to the worker; imported gate PASS does not imply this unimplemented control was enforced.
