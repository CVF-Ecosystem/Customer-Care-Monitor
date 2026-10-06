# R055 SPEC: observational source-preparation receipts

Date: 2026-10-06. Intended behavior; design-time implementation NOT_STARTED; current child source05c59e9 REVIEW_PASS/FREEZE_OPEN under R057 independent evidencec4f78f2. R2, independent Codex `/root` review after child implementation. [Decision](../decisions/SOURCE_PREPARATION_RECEIPT_R055_2026-10-06.md), [work order](../work_orders/CCMAI_RUNTIME_055.md), seed8d9137d8fd4c23d08744abc30b0b9975c68da626. This is local application observation, not CVF governance/full S2/provider proof.

## Closed v1 observation contract

Add source_preparation to Summary only for consumed, correctly bound Analyzer execution. No receipt/side effect for mismatched/already-consumed reservations or legacy Abort/setup paths. Envelope version `ccmai.source-preparation.v1`; scope `preparation_only`; existing analysis mode and sanitized tenant/job/run identity. Null/unavailable counts and binding metadata are explicit where facts were not observed.

Selection status NOT_ATTEMPTED / FAILED / COMPLETE. Selected is nullable until successful candidate query and means its returned ordered list, including explicit caps, not the tenant universe or upstream completeness. Visited increments once before work on a candidate. Each visited candidate gets exactly one observed outcome; in-flight panic is PREPARATION_INTERRUPTED. Unvisited=selected−visited when selected is known, else null. Selection failure is run-level, never a fabricated per-conversation outcome. Cancellation before/within preparation retains observed prefix and an explicit stop reason; unvisited candidates are not assigned snapshot errors or skip decisions.

Outcomes: EMPTY_SOURCE, UNCHANGED_VERIFIED (ordinary only after existing provenance check), PREPARED_FOR_INFERENCE, SNAPSHOT_ERROR, SOURCE_VERSION_ERROR, PREPARATION_INTERRUPTED. Reference states distinguish NONE, LEGACY_NO_SNAPSHOT, VERIFIED, LOOKUP_FAILED and PROVENANCE_UNVERIFIABLE from the existing query/check result; optional bound evaluation/run/snapshot ids reflect the SAME observed query, never a lookup added for receipt. Preserve errors/wrapping/order and scalar error counts. Empty source means no prepared local source, not no intervention. PREPARED_FOR_INFERENCE does not mean provider initialized/called or data complete; partial coverage remains explicitly partial.

Snapshot metadata only when available and correctly bound: current schema version, digest as normalized valid SHA256hex, coverage complete/partial/empty and allowlisted NO_MESSAGES/HISTORY_WINDOWED/UNSUPPORTED_CONTENT_TYPE/ATTACHMENT_NOT_REPRESENTED/ATTACHMENT_JSON_INVALID. No manifest/message/external-message ids/sender/name/body/content/prompt/URL/key/raw error. Unsafe or foreign ids/hash/enums are omitted and metadata invalid/incomplete flagged. Sanitization only affects observation, never selection/source validation/inference or counters.

Counts contain all visited observations by outcome. Cap entries to first200 in original candidate order AND256KiB encoded envelope; count omitted entries independently, counts never derived from retained list. Describe entry completeness separately from preparation scan completion. Large inputs retain exact observed counts/unvisited/omission and flags; never claim a sampled list is exhaustive. Snapshot/reference metadata and frozen slices are immutable for downstream summary writers. Pure tests may exercise a lower internal serializer byte budget without adding public configuration or changing production limits.

## Persistence and compatibility

Preserve exact original scalar summary keys and values at initial/single-error/single-progress/batch-progress/final stages. One shared builder adds receipt, no Summary DB read/extra query. Batch receives frozen preparation receipt or the already constructed run envelope, not a new preparation run. Final persistence uses existing finalizer transaction; successful row reload verifies stored receipt. Receipt-aware early input/selection/provider failure and panic retain available observations with correct unknown/interrupted status through a new summary-aware close helper. Original closeOwnedRun wrapper remains behavior-compatible: {} persisted, original returned-object semantics, cancellation/owner decision/checkpoint/retry/admission unchanged. Only new receipt-aware calls set returned Summary. Existing notifications/AI usage/results/snapshot writes unchanged.

In-memory receipt after a failed write is not durability evidence. Keep existing DB-write/terminal failure return/status/checkpoint behavior; recording gaps emit fixed bounded receipt diagnostic class/stage without driver text or unvalidated identifiers. No new extra DB activity write, repair/retry or authority from a receipt. Missing/invalid stored key cannot be treated as complete/zero-work by a future consumer. Retention follows existing JobRun row lifecycle; no new purge policy.

## Acceptance

| ID | Required observable proof |
| --- | --- |
| SP-01 | Version/binding/enum serialization, selected unknown versus zero, invalid/misbound metadata omitted/flagged, no prohibited strings. |
| SP-02 | Exact outcome/count reconciliation, first200 order, byte bound and honest omission/unvisited/in-flight interruption, frozen copies unchanged. |
| SP-03 | Ordinary empty/unchanged/changed/partial source outcomes and SAME reference query metadata; no extra query, existing returned lists/errors unchanged. |
| SP-04 | Explicit modes preserve cap/date/full-context ordering, no fabricated version skip, empty/partial/changed/error observations correct. |
| SP-05 | Missing/corrupt/misbound snapshot and candidate/query failures remain errors, not successful skip; unknown selection counts truthful. |
| SP-06 | Context/owner cancellation before/mid-preparation, provider-init boundary and panic preserve terminal precedence, source counts/unvisited/interrupted truthful. |
| SP-07 | Initial/single-error/single-progress/batch-progress summaries retain same scalar keys/values and frozen receipt; provider call counts/outcomes unchanged. |
| SP-08 | Stored and returned final/early provider failure/panic summaries retain valid receipts; legacy close/Abort behavior unchanged. |
| SP-09 | Write/terminal failure retains original admission/checkpoint/status/error/retry behavior; recording-gap diagnostics bounded, no false durability claim. |
| SP-10 | Existing LP/ordinary/explicit/ownership/F06 regressions PASS with old test bytes unchanged; new tests assert direct DB reload and no extra query where required. |
| SP-11 | Exact committed archive/full manifest and command-by-command isolation, raw JSON counts/names, source/test/authority/ack/build/evidence/handback identities, named cleanup. |
| SP-12 | Applied one-match receipt-erasure mutation fails a named stored-terminal-receipt assertion (no build/timeout), controlled throw/finally byte restoration, restored baseline PASS; final publication gates/docs/diff/secret/protected checks. |

New tests in dedicated files only, maintained top-level names prefixed `TestSP` for integration and `TestPreparationReceipt` for pure contract tests. Derive expected top-level names from exact committed test bytes and command selection; JSON must contain every selected detector with0 FAIL/SKIP on positive controls. Helper errors/build failures/timeouts never count as a behavioral mutation kill. Expected mutation assertion should clearly identify missing source_preparation in a stored row.

## Authority and claims

Allowed source changes only in new receipt helper/tests and analyzer/incremental/modes/narrow ownership terminal-summary wrapper. No model/migration/snapshot semantics/provider/prompts/scheduler/permissions/frontend/tooling/dependency/old tests/old packets or seeds. Preserve R050/R051/R053/R054 frozen snapshot history, global F02/live/complete S2 limitations and parked Facebook/Zalo OA.

Cached offline Go/Docker/disposable synthetic MySQL only; no actual provider/channel/config/.env/credentials/external network/customer/persistent DB. Application test doubles are fault fixtures, not mock governance evidence. A CVF runtime governance claim would require separate real-provider authority/evidence; none is granted or asserted by this SPEC. No push/merge/deploy/FREEZE/self-approval. Exact source/evidence independently reviewed; repeated repairs follow existing same-scope/cost rules without unnecessary owner waits.
