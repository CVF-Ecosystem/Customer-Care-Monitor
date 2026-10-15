# R102 WAIT_DATA backend — implementation worker source return

Status: BLOCKED_WITH_REASON
Blocking reason: Source implementation is ready for Root's independent admission; W01–W08 native evidence cannot run under the current source-only boundary.
Date: 2026-10-15
Risk: R2
Worker: /root/r102_worker, IMPLEMENTATION_WORKER
Reviewer: Codex /root
Seed: c199f13d01bdae7c4ad2b12e8528e0f4bce412e3
Dispatch: 11a60c007e202c9c44f484bcce3c4c6c5426e9c4
Phase: BUILD; R102 scoped runtime acceptance NOT_RUN

Root added the explicit Risk/Reviewer header fields for the pinned role checker after source return; these fields name ownership and do not assert runtime approval by the worker.

## Exact changed implementation set

The eight Go paths in [source manifest](probes/r102_worker/source_manifest.json) are the only implementation paths changed. I also appended my BUILD role acknowledgment to the active handoff and wrote this source-only return/manifest. Root concurrently owns canonical continuity and its own review probes. I did not edit the seed, shared snapshot/sync/M3/RunManual source, or R101 source.

The implementation adds append-only `InterventionDataRevision` and nonexecuting `InterventionRerunRequest`, nullable event references, additive migration, two POST and three GET routes. Source capture and request creation recheck membership and original analysis provenance, serialize through channel/source/case locks, advance case version with CAS, and append one event atomically. They leave old run/snapshot/proposal/event rows and ownership projection untouched. Request requires fresh `jobs:w`; exact key replay reauthorizes and returns the stored NOT_RUN intent even after source changes, with no new mutation. Current source observation is bounded to 1000 messages, 4 MiB message content and 1 MiB manifest. The persisted canonical manifest uses mediumtext and is checked against its SHA256 digest on reuse/detail. No path invokes an AI runner or creates a JobRun, attempt or budget entry.

## Designed W01–W08 coverage, not executed evidence

| ID | New test design |
| --- | --- |
| W01 | `TestR102NativeRevisionIntent`: real router/auth + explicit R102 MySQL DSN, source addition, persisted revision/detail/list, assigned owner and original snapshot/event preservation. |
| W02 | `TestR102NativeRevisionIntent`, `TestR102NativeDenialsAndRollback`: canonical unchanged empty source 409, changed incomplete revision and request refusal, complete anchor and stored AWAITING_EXECUTION_AUTHORITY/NOT_RUN intent. |
| W03 | `TestR102NativeDenialsAndRollback`, `TestR102NativeForeignRevisionAnchorAndPermissions`, `TestR102NativeSourceChangesReplayAndRevocation`, `TestR102NativeMembershipRevocationBarrier`: read-only/unknown/malformed/revoked member, missing jobs-w and scoped 404 paths. |
| W04 | `TestR102NativeStateAndIdentityRejections`, `TestR102NativeSourceMutationRejection`, `TestR102NativeForeignRevisionAnchorAndPermissions`, `TestR102NativeCorruptRevisionUnavailable`: stale identity/version, corrupted provenance/revision, changed/removed/appended source, job/state/fixture and anchor denial. |
| W05 | `TestR102NativeChannelBarrier`, `TestR102NativeCompetingWriters`, `TestR102NativeCompetingRequestKeys`, `TestR102NativeHumanActionCompetition`, `TestR102NativeMembershipRevocationBarrier`, `TestR102NativeSyncWriterBeforeRequest`, `TestR102NativeActionBeforeSourceAndRevocation`: controlled channel/member locks in both orders, different actors, version winner, existing ASSIGN_DATA race, action-first and writer/revocation-first outcomes. The action-first test uses a test-only GORM event callback after the action has taken locks; all barrier goroutines have finite release/completion paths. |
| W06 | `TestR102NativeRevisionIntent`, `TestR102NativeCompetingRequestKeys`, `TestR102NativeSourceChangesReplayAndRevocation`, `TestR102PayloadHashAndBounds`: exact replay, changed-key payload, second pending intent, distinct-key race, full semantic hash. |
| W07 | `TestR102NativeDenialsAndRollback`, `TestR102NativeLatestVersionAndRequestRollback`, `TestR102NativeInsertAndCASRollbackBranches`, `TestR102NativeRevisionIntent`: test-scoped GORM callbacks independently force revision insert, request insert and case-CAS zero rows; each denied transaction asserts unchanged case/event/revision/request state. Existing event-insert collisions, nil runner and unchanged original snapshot/proposal/JobRun/Attempt/Budget counts are also covered. Existing ownership guard source was not modified. |
| W08 | `TestR102NativeLegacyEventSurvivesMigration`, `TestR102NativePaginationAndMessageCap`, `TestR102NativeBodyAndCursorBounds`, `TestR102ContentCapPrecedesTranscriptBuild`, `TestR102NativeSilentSQLSuccessAndFailure`: repeat migration of an exclusively owned synthetic legacy event after dropping only two new nullable columns, cursor and bounds, body limit, 1001-message SQL cap, pre-builder content cap, and captured Info-level SQL logger sentinel exclusion on success and induced failure. |

Native tests require `R102_MYSQL_DSN` with parsed `tcp` host `127.0.0.1` and exact database `r102_fixture`. They use unique generated tenant/user/case IDs, and cleanup is restricted to those generated rows. No application `.env`, original DB or preserved R101 fixture is read. The selected group names all start `TestR102`; an absent DSN explicitly SKIPs native tests, which must not be reported as acceptance. Root owns the one new labelled fixture, launch budget and independent source/runtime review.

## Commands and limits

`gofmt -w` on the eight authorized Go paths: exit 0. `git diff --check`: exit 0. Source hashes and sizes are in the linked manifest. Go build/test/binary launches 0/8 (worker 0/4); MySQL fixtures 0/1; provider 0; mock 0; Node/browser/server 0. No raw Go stdout/stderr or no-SKIP result exists because runtime remains prohibited until Root admits exact source. Source-only static checks do not establish W01–W08 acceptance, runtime AI governance, actual rerun or full M3. The R101 browser and Dashboard baseline failures remain open.

Root's next move is consolidated independent source review of this exact changed set and test soundness. Any accepted same-scope finding returns to me as REPAIR_WORKER within the existing R102 seed. Root then decides whether to admit the finite offline compile and owned synthetic MySQL campaign.

## Consolidated repair round 1 source return

Root's [four-group independent source review](R102_CONSOLIDATED_SOURCE_REVIEW_2026-10-15.md) returned CHANGES_REQUIRED. Within the same seed and eight-path source scope, I added per-call silent service sessions, an Info-level captured logger test with reason/manifest sentinels, a held-lock action-before-writer-and-revocation test, test-scoped insert/CAS callbacks, and checked assertion/cleanup DB errors. No global logger, shared source, provider or production hook changed. The tests have not run; their expected HTTP codes and rollback conditions remain unverified until Root's source admission and finite MySQL campaign. The earlier W05/W07 coverage limits above are superseded by these test designs, not by runtime evidence. Go0/8, fixture0/1, provider0/mock0 remain unchanged.
