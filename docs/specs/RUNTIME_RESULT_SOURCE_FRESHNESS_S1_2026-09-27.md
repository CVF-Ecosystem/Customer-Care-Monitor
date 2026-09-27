# S1 result source freshness contract

**Tranche:** `CCMAI-RUNTIME-004` · **Phase:** SPEC · **Risk:** R2 · **Source baseline:** `CCMAI-RUNTIME-002` snapshot/evidence and `CCMAI-RUNTIME-003` same-ID replay passed independent REVIEW; both remain FREEZE open.

## Problem and outcome

`models.JobResult.AfterFind` currently calls a result `snapshot_bound` when `analysis_snapshot_id` is present; it does not compare the stored snapshot with today's source. `backend/api/handlers/results.go` serves the Results page and exports without a source-change signal. The UI can therefore show an old judgment beside a changed transcript without a warning. This tranche adds an explicit read-time source-integrity signal for this Results surface. It does not rerun AI, modify saved findings, establish upstream completeness, or authorize a human decision.

## Contract

1. Add a separate `source_integrity_status` to each Results-page row and export; keep the existing `evidence_status` meaning intact. Use bounded values: `changed_since_analysis`, `bound_currentness_unverified`, `legacy_unverified`, and `verification_unavailable`. Never label a result `current` or `safe` merely because a snapshot exists or a local comparison found no difference.
2. For a snapshot-bound result, load its tenant-scoped `AnalysisSnapshot` and parse/validate the stored manifest. Compare the snapshot's included message IDs and digest-relevant facts with current tenant-scoped message rows, using the same snapshot canonicalization/fingerprinting rules. Detect edited content, sender role/name, content type, timestamp, attachment identity, missing messages, and added messages that alter the conversation's observed set. An unchanged local comparison is only `bound_currentness_unverified`, because adapter history/removal coverage and timing may be incomplete. A changed raw payload alone need not alter this signal when it has no effect on the `ccma.snapshot.v1` manifest; document that limit.
3. Missing snapshot link is `legacy_unverified`. A broken snapshot link, malformed manifest, unsupported schema, or query/verification error must never be reported as unchanged. Return `verification_unavailable` for row-local corruption; a database-wide read failure may fail the request with an observable server error. Do not expose raw transcript or payload in the status/receipt.
4. Compute status only for the paged Results rows (or bounded export rows) after existing tenant, job, filter and pagination predicates. Avoid one query per row; batch snapshots and messages by tenant/conversation where possible. Preserve existing filtering/counts, result ordering and export row limits. CSV/XLSX must include the same status semantics as the page; the frontend table, card and detail dialog must display an understandable Vietnamese/English warning for changed, legacy and unavailable states, and explain that an unchanged local comparison does not prove full upstream currentness.
5. Status is read-only and must not alter stored result, snapshot, run, sync checkpoint or current messages. A later re-analysis may create a new result/snapshot through the existing workflow; this tranche never silently repairs one.

## Acceptance

- Disposable-MySQL tests seed snapshot/result/message rows without any provider call. They prove unchanged local source yields `bound_currentness_unverified`; same-ID content/role/attachment replay, a missing message and an added message yield `changed_since_analysis`; a legacy result yields `legacy_unverified`; corrupt/missing snapshot data cannot look unchanged; tenant boundaries hold.
- Results API pagination/counts and export remain consistent; both CSV and XLSX carry the status field. UI tests or a build verify the label appears in table, card and detail views and remains accessible without transcript permission.
- `go test ./... -count=1`, `go build ./...`, `go vet ./...`, frontend build, catalog check, workspace doctor and `git diff --check` pass. Record exact commands, fixture cleanup and the no-provider claim boundary.

## Deferred

This contract covers the aggregate Results surface and its export. A later separately scoped pass must assess job-specific result APIs, conversation evaluation badges, notification delivery, automatic re-analysis, upstream edit/delete discovery outside fetch windows, and S2/S3/S5 decisions. No CVF runtime-governance claim or production freshness guarantee follows from local snapshot comparison.
