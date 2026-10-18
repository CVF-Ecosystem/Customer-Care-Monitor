# R102 candidate — Backend WAIT_DATA: source revision and explicit rerun request

Status: ACCEPTED_INTENDED_CONTRACT
Date: 2026-10-15
Roadmap: M3-05, supporting M2-02; full milestones remain OPEN
Author: Codex /root, ORCHESTRATOR and SPEC_AUTHOR
Design adviser: /root/waitdata_design, Sol Medium, read-only
Execution state: NOT_DISPATCHED; no R102 BUILD authority installed

## Product outcome and scope

After missing conversation data has arrived through an already authorized source writer, an authorized user can record what local source is now available and submit a durable, traceable request for reanalysis. The request remains `AWAITING_EXECUTION_AUTHORITY`. This slice does not execute reanalysis, produce an AI proposal, certify upstream completeness, or complete the case.

This is the finite backend part that can progress while R101 browser tooling is blocked. It does not replace R101 H08 or repair the three inherited Dashboard tests. Source supplementation itself uses existing sanctioned sync/import paths under their own authority; these endpoints accept no transcript, fabricated message, attachment upload, or caller assertion that coverage is complete. No real channel sync is authorized by this packet.

## Source facts and design decision

- `backend/engine/intervention.go:142-173`: RunManual reuses the stable case and updates run/snapshot/projection after checking ownership. The case has a tenant/job/conversation/use-case unique identity (`backend/db/models/intervention.go`). Creating a second case with the same identity is not a safe successor design.
- `backend/engine/intervention_m3.go:70-76`: pending proposal, human assessment, data owner or any human event causes the existing ownership guard to block ordinary rerun. Do not remove or bypass this guard.
- `backend/engine/intervention_m3.go:97-146`: current membership and channel/conversation/message/case/snapshot locks support local transaction checks. `ASSIGN_DATA`/`ESCALATE_DATA` already exist; they do not refresh source or request reanalysis.
- `backend/engine/snapshot.go:74-94`: the loader uses global db.DB. A new transaction-bound revision builder must read through its supplied transaction; calling the global loader while pretending to hold a consistent source view is prohibited.
- `backend/engine/snapshot.go:373-390`: AnalysisSnapshot is linked to a JobRun. Do not invent a JobRun or attach newly observed source to the old AI run. Use a distinct append-only data-revision record.

Decision: add local data revision and rerun-intent records linked to the existing case. Retain original AnalysisSnapshot, case run/snapshot references, proposal, receipt, outcome and all human events unchanged. No new case lineage or new executor is introduced. Completion of an actual rerun later requires a separately specified, authorized execution-and-publication contract preserving this history.

## Data contract

`InterventionDataRevision` is append-only: server UUID, tenant/case/job/conversation, creation actor/time, originating case version, original snapshot id/digest, canonical current-source manifest/digest, coverage/reasons/message count, immutable observation status and bounded reason. The schema distinguishes original analysis identity from current local observation. Manifests contain only the canonical metadata/hashes needed by the existing snapshot contract; do not duplicate transcript text, provider output or secrets.

`InterventionRerunRequest` is append-only for this slice: server UUID, tenant/case/job/conversation, exact data revision/digest, expected original analysis identity, actor/time/reason, client request key and canonical payload hash; fixed status `AWAITING_EXECUTION_AUTHORITY`, execution `NOT_RUN`. No consumer, scheduled worker, status mutation or dispatch endpoint is added. A durable unique constraint plus case serialization prevents two pending intents for the same case/data revision. The request key is tenant/case scoped; maximum128 ASCII safe characters, never a credential.

Successful POSTs increment the case version through CAS and append exactly one human event in the same transaction as their new record. Add only nullable `data_revision_id` and `rerun_request_id` references to InterventionHumanEvent and its API projection if required for discoverability. This permits existing events to remain byte-preserved. Original case snapshot/run/proposal identities and DataOwnerID/DueAt remain unchanged; work stays WAITING_ON_DATA. New actions must not accidentally certify the original incomplete snapshot as current.

No transcript mutation endpoint and no human-history update/delete route. New metadata appears in dedicated bounded read APIs; existing R101 detail response and UI behavior remain compatible.

## API contract

All paths are under `/api/v1/tenants/:tenantId/intervention-cases/:caseId` with authentication and tenant-scoped reads. Actor/tenant/time are server-owned. Write bodies have a16KiB limit, reject malformed/unknown fields, require positive expected_version, bounded nonempty reason<=500 Unicode characters and exact original expected_snapshot_id/digest. No provider/model/budget/job target/tenant override fields are accepted.

| Method/path | Contract |
| --- | --- |
| POST `/data-revisions` | Capture current local source for this WAITING_ON_DATA case under locks. No source input in body. A genuinely new digest is required; unchanged source returns409 `source_unchanged` without writes. A changed but still incomplete source may be recorded honestly. Return201, revision and resulting case version. |
| GET `/data-revisions` | Minimal metadata, observed time, local coverage/missing reasons, no transcript; stable created_at/id cursor, limit1..100/default20, has_more. Bounded detail GET `/data-revisions/:revisionId` returns one same-case record with canonical manifest when authorized. |
| POST `/rerun-requests` | Bind exact data_revision_id/digest, current expected case version, original snapshot identity, reason and request_key. Recheck current source equals that revision, local complete coverage and existing complaint/wait anchor criterion. Require no previous AI/human assessment or execution outcome. Return201 with explicit `AWAITING_EXECUTION_AUTHORITY` / `NOT_RUN`, never202 accepted-for-dispatch. |
| GET `/rerun-requests` | Same bounded tenant/case metadata pagination; exposes request status, revision/actor/time and execution NOT_RUN, no AI proposal/usage invention. |

Both POSTs require current known tenant role with intervention_cases r+w. Request additionally requires jobs w, checked on the same freshly locked membership row; being DataOwnerID alone grants no execution-related privilege. Owner/admin follow existing semantics. Read APIs require intervention_cases r. Unknown/malformed/revoked/cross-tenant access fails closed with safe errors. A new intent must use the latest committed data revision; a newer observation makes an older revision ineligible even if the caller supplies its valid identifier.

Exact request-key replay is checked after fresh authorization and case scoping, before rejecting the old version: same canonical payload returns200 existing intent with its original observation time and explicit NOT_RUN, without writes/version/event duplication or a fresh-source claim. Same key with changed payload returns409. A new key for a revision with an existing pending intent returns409 with a safe conflict code. Never automatically replay a write after a conflict.

Return400 invalid body;403 insufficient current permission;404 tenant/case/revision mismatch without disclosing another tenant;409 version/source/state/key conflict;503 storage/unavailable. Do not put SQL, credentials or payloads in errors/logs. Positive incomplete revision is a source observation, not a ready-to-run request. Missing/corrupt original analysis provenance yields an explicit conflict/unavailable response, not reconstructed history.

## Transaction and concurrency contract

Locate parent IDs outside the transaction only as hints. Recheck all identities under locks. Use one consistent order: actor membership → job → channel → conversation → ordered source messages → case → original snapshot → revision/request records. Validate current job type/manual/activity/input channel, selected synthetic Pancake fixture marker and original snapshot provenance. Parent/channel locking must match supported sync writers; inspect existing writer order before BUILD, and stop for an actual incompatible cycle instead of changing unrelated writers.

Build the observation with transaction-bound canonical helpers and messages only; reuse same-package buildConversationSnapshot on locked messages, not the global DB loader. Read at most1001 rows to detect a maximum1000-message observation; reject larger input or canonical manifest exceeding1MiB as source_unavailable without truncation or writes. Do not expose raw manifest bytes in logs. Validate case WAITING_ON_DATA, expected version/snapshot/digest and no accepted/rejected/resolved assessment. Request creation additionally rechecks latest revision lineage, current digest/coverage/anchor and jobs permission. No transaction may resolve a provider, call RunManual, reserve a JobRun or alter attempt/budget ledgers. Do not clear DataOwnerID, history or ownership guard to make an action succeed.

Root resolves the read-only adviser's no-case-mutation alternative as follows: new records are append-only, old events remain unchanged, while each accepted new mutation advances the existing case CAS version and appends one new event. This keeps stale action checks shared with existing ASSIGN_DATA/ESCALATE_DATA rather than maintaining an uncoordinated parallel version. The only old case projection change allowed is version/updated_at. This is a design choice, not a runtime result. The extra jobs-w gate is deliberate defense for execution-related intent, even though this slice cannot execute it. Source bounds above are local safety limits, not owner-approved pilot thresholds.

CAS version increment, new revision/request row and exactly one event either all commit or all roll back. Current data ownership/deadline/escalation survive. Two writers at one expected version produce at most one mutation; identical request-key replay may read the winning intent. Concurrent supported source changes and membership revocation serialize to either an action-before-change or rejected stale/forbidden result. Never claim protection against arbitrary out-of-band SQL or upstream history omitted by the channel.

Coverage `complete` and a matching digest mean only the local canonical source was suitable at the recorded instant. A stored intent is not a capability token for future dispatch: any later executor must independently reauthorize/revalidate source and budget; it may not consume this record blindly.

## Acceptance matrix — intended, not results

| ID | Actual local software proof required before bounded acceptance |
| --- | --- |
| W01 | Real router/auth + MySQL: existing WAIT_DATA with assigned data owner, sanctioned-writer-shaped synthetic source addition, revision persisted and read back through new read APIs; old snapshot/proposal/history unchanged |
| W02 | Changed incomplete source records explicit incomplete observation; request refuses it. Unchanged source records nothing. Complete+anchor local revision allows durable intent with NOT_RUN, original case remains WAITING_ON_DATA |
| W03 | Read-only, missing/unknown/malformed/revoked member, missing jobs w, other-tenant case/revision all rejected with no case/event/revision/request writes |
| W04 | Stale version, wrong original snapshot, corrupt/missing provenance, changed/appended/deleted source after capture, wrong job/state/fixture marker and missing anchor reject safely |
| W05 | Competing revision/request writers, request vs source update, membership revocation and concurrent existing human data action have deterministic barrier tests; one accepted mutation at most per case version |
| W06 | Same key/body replay returns same intent; key/body mismatch and second pending intent conflict; no duplicate event/version increment |
| W07 | Forced revision/request/event insert or CAS failure rolls back all new/projection writes; existing ownership guard unchanged; nil runner and job-run/budget/attempt row counts unchanged in direct-seeded human-software fixture |
| W08 | Fresh and legacy synthetic schema migration twice, nullable event additions preserve old rows; cursor pagination and minimal read authorization; errors/body bounds and no payload secrets in logs |

W01..08 use actual disposable MySQL software tests after future runtime admission, not mock AI governance proof. No mock provider is permitted. A source-only compile is not W01..08 acceptance. R101 H08 and full WAIT_DATA reanalysis/integration remain OPEN even if these rows later pass.

## Finite remaining delivery route

1. Dispatcher freezes the packet's scope/authority and commits a new seed before source BUILD. This planning packet grants no effects.
2. One coherent source change for revision+intent+API+native tests, consolidated independent source review and bounded same-scope repairs.
3. One admitted synthetic MySQL campaign and independent review of exact source/evidence; accept only the local revision/intent capability. Actual AI rerun and browser acceptance require their respective outstanding authority and evidence.

No browser runtime repair, generic helper campaign, Dashboard test repair, release work or real-customer data is a prerequisite to this backend slice. Do not close full M3 from request persistence alone.
