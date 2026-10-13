# R091 independent Local review — round 1

Memory class: EVIDENCE_RECORD
docType: completion_review
Status: CHANGES_REQUIRED
Risk: R2
Worker: /root/outbound_intake
Reviewer: Codex /root
Review-Cost Telemetry: REQUIRED

## Decision And Candidate

One consolidated same-seed repair for four source/evidence clusters in probes/r091_local_review/round1_findings.json. Root independently read all changed production clusters, new route/status tests, old fixture diffs and targeted DB/SMTP/batch tests. Candidate source/evidence manifest82/82 raw and LF bindings match; no Root production edits. Actual dispatch/closure base03e68e1ad91f74bb153e7fa5c8b646001b9f78cc. This is the first formal C review, not a new tranche or a return to Claude.

## Findings And Dependency Audit

- R091-R1-F01: Telegram absent/null ok is treated as explicit rejection and therefore retryable failed; reserve failed for explicit Boolean false and retain missing/null/malformed envelopes as unknown.
- R091-R1-F02: SMTP HELO fallback now fails AUTH-unavailable although the pinned original SendMail skips auth when EHLO fails/HELO succeeds. Preserve that existing healthy path and retain ESMTP missing-AUTH rejection.
- R091-R1-F03: marker Count is not a locking read and Update ignores affected rows. A concurrent captured-row removal can become nil success; verify the marked captured set safely.
- R091-R1-F04: mandatory claim/outcome persistence error, command/DATA cancellation closure, actual batch streak/reset and cancel-before-next-call branches have no valid case evidence. Add narrowly focused controls; valid existing campaigns are reused.

Exact source bases, named before/after assertions, proof/fixture budget and repair bounds are in the finding JSON and existing order's appended repair envelope. Initial Go migration1PASS, notification/engine30 terminalPASS, A15 mandatory top-level/30terminalPASS/0SKIP, parse/save3terminalPASS, UI18+5PASS/type/build0 remain valid at their recorded hashes. A production and adapter/dependency files are unchanged from44bb345. Original worker return/logs/hashes remain immutable; repair returns a new artifact/proof folder. Historical setup/encoding/bootstrap/secret-scanner faults and first fixture readiness timing limits are candidly retained. Both original synthetic fixture resource sets have absence receipts. No browser/live/hosted CI or product closure claim.

## Checker Source Read-Ahead Block

Read current seed/order/SPEC, legacy transition grammar, trusted phase applicability, closeability and review-cost rework envelope. Same paths/risk/effect/commit owner; seed unchanged. Worker must rehydrate BUILD, acknowledge same repair and pass legacy plus explicit active APPL CHECKED before tests/fixture/source. Root preserves independent review; no automatic successor. Review third round requires escalation under the standard.

## Return-Time Closeability Recheck

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
outsideAuthorityBlockers: NONE
nextRepairRoute: SAME_SEED_R091_CONSOLIDATED_LUNA_REPAIR
workerRedispatchAllowed: YES

## Review Cost Telemetry And Stop Disposition

reviewRoundCount: 1
workerRepairTurnCount: 0
newRootCauseCountThisRound: 3
dependentFindingCountThisRound: 1
elapsedReviewMinutes: 8.459
tokenOrQuotaUsage: NOT_AVAILABLE_WITH_REASON: numeric usage/bill not exposed
providerCallCount: 0
valueDelta: Independent source and evidence audit identifies unsafe retry classification, a preserved SMTP path regression, marker concurrency and mandatory case gaps in one finite repair.
stopDisposition: CONSOLIDATE_SINGLE_REPAIR
preRepairAuditDisposition: COMPLETE_BEFORE_FIRST_REPAIR
materialCommitCount: 0
continuityCommitCount: 0
commitPlanDisposition: DEFAULT_ONE_MATERIAL_ONE_CONTINUITY
latencyDisposition: NOT_MEASURED_WITH_REASON: no end-to-end latency timer
avoidableDelayClass: GATE_DISCOVERY_LOOP

## Public Export Disposition

DEFERRED_PRIVATE_ONLY. C source is uncommitted and not accepted; Root alone publishes the accepted existing project branch. FREEZE/full S0-S7, channels/dependency17/docs55/R086 native-fence/accounts/browser/live/hosted/deploy remain open or parked.

## Root preparation accountability

Named metadata/catalog defects during repair-envelope preparation are preserved in probes/r091_local_review/root_catalog_family_correction.json: unsupported catalog family, implicit cp1252 read failure, directory-family versus leaf mismatch and missing seven rework-routing booleans. Root consulted schema/checker, corrected only metadata, then legacy7/7 and reviewer-fast7/7 passed (inactive APPL not checked). These are Root faults; no production/proof-result edit or waiver, and they are not worker defects. Review timer covers source/evidence evaluation plus this preparatory gate work at the timestamp in round1_intake.json.
