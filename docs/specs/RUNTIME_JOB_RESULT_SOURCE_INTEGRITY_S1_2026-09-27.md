# S1 job-result source-integrity contract

**Tranche:** `CCMAI-RUNTIME-006` · **Phase:** SPEC · **Risk:** R2 · **Entry:** R001–R005 independent REVIEW PASS; each FREEZE and S1 closure remain open.

## Problem and decision

R004 checks source integrity on aggregate Results rows and its export. The job-specific `ListJobResults`, `ListAllJobResults`, and `ExportJobResults` paths in `backend/api/handlers/jobs.go` still return snapshot-bound findings and grouped exports without that check. `JobDetail.vue` presents those findings without a source-change warning. The classification export also reads current chat content, which may differ from the chat at analysis time. This is an open S1 stale-source presentation gap, not evidence that the underlying stored finding changed.

Apply R004's **same local comparison and four status meanings** to each job-result row: `changed_since_analysis`, `bound_currentness_unverified`, `legacy_unverified`, `verification_unavailable`. A successful local comparison never proves full upstream currentness. Keep the stored verdict, evidence, snapshot and R005 confidence semantics unchanged.

## Contract

1. Both job-result JSON endpoints include `source_integrity_status` for every returned result. Reuse R004's provenance verification and canonical comparison through one shared tenant-scoped batch implementation, so the aggregate Results path retains its current behavior. No one-query-per-result path. A broken/cross-tenant snapshot or malformed manifest is `verification_unavailable`; a shared snapshot/message query failure fails the request observably before a success body is sent.
2. Job-specific CSV and XLSX exports include a source-integrity column. Because each export row groups possibly multiple results/runs for a conversation, represent **all distinct per-result statuses** in a stable order; do not collapse a mixed group into a single reassuring value. The export may add a result-ID/status detail column if needed for traceability. The classification chat-text header or adjacent explanation must say the text is read at export time, not the analyzed snapshot. Keep existing row selection, tenant isolation, ordering and other export columns. Verify query errors before sending file headers or bytes.
3. `JobDetail.vue` shows an understandable warning for changed, legacy and unavailable states in table/card/dialog views, with a visible explanation that `bound_currentness_unverified` is only a local comparison. For groups spanning multiple results, show every distinct status or a clear mixed-state summary that cannot hide `changed_since_analysis`. Vietnamese and English labels must be consistent with R004. No permission bypass or raw transcript disclosure follows from a status badge.
4. Computation is read-only. Do not rerun AI, mutate results/messages/snapshots, mark work accepted, suppress or send notifications, or infer upstream deletion outside adapter fetch windows. R005 `confidence` and `confidence_basis` remain intact.

## Acceptance

- Disposable-MySQL tests execute both actual job-result handlers and both export formats with unchanged, edited, legacy, broken/cross-tenant and mixed-run/mixed-status fixtures. Assert exact status values, stable grouped export representation, preserved existing fields and tenant isolation.
- Force the shared snapshot/message query to fail and prove both JSON handlers and both export formats return an observable server error with no partial downloadable file. Existing R004 aggregate Results status and failure tests still pass after helper extraction.
- Frontend build/type check and a focused label/grouping test or direct component assertion show the warning in job table/card/dialog. Backend focused/full tests, build/vet, catalog, doctor and diff check pass. Record fixture cleanup and the no-provider/no-upstream-completeness boundary.

## Deferred

Conversation-specific APIs, notification delivery, automatic re-analysis, real-channel edit/delete discovery outside fetch windows, S2/S3/S5 decisions and S1 FREEZE require separate disposition. This tranche makes no CVF runtime-governance or production freshness claim.
