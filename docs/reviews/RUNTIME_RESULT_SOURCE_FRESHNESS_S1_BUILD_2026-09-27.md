# BUILD evidence: CCMAI-RUNTIME-004 — S1 result source freshness

**Work order:** `docs/work_orders/CCMAI_RUNTIME_004.md` · **Spec:**
`docs/specs/RUNTIME_RESULT_SOURCE_FRESHNESS_S1_2026-09-27.md` · **Risk:** R2 ·
**Ngày:** 2026-09-27 · **Role:** Claude (`IMPLEMENTATION_WORKER`) · **Kết quả:**
BUILD complete, `REVIEW_PENDING` for Codex's independent review. No FREEZE
claimed; `CCMAI-RUNTIME-002/003` remain REVIEW PASS / FREEZE open.

## Root cause / gap closed

`models.JobResult.AfterFind` labels any row with a nonempty
`analysis_snapshot_id` as `snapshot_bound`, but never compares the stored
snapshot to current source. `backend/api/handlers/results.go` (the aggregate
Results page and its CSV/XLSX export) served rows and exports with no
source-change signal at all, so the UI could show an old judgment beside a
transcript that had since been edited, without any warning.

## Audit performed before editing (required implementation item 1)

- **Results read/export query:** `resultFilter.baseQuery`/`fetchRows`
  (`backend/api/handlers/results.go`) anchor on `conversation_evaluation`
  rows via a `latest`-per-(conversation, job) derived table, already
  tenant/job/filter/paginated before any row-level enrichment (issues/tags)
  runs. `ListResults` and `ExportResults` both call the same `fetchRows`, so
  wiring the new status into `fetchRows` guarantees page/export consistency
  for free — no separate export-only code path to keep in sync.
- **`ccma.snapshot.v1` fields:** `backend/engine/snapshot.go`'s
  `snapshotManifest`/`snapshotMessage` carry exactly `message_id`,
  `external_message_id`, `sender_type`, `sender_name`, `content_type`,
  `sent_at` (RFC3339Nano UTC), `content_sha256`, `content_code_points`,
  `attachment_coverage`, `attachment_count`, `attachment_fingerprint`, plus
  manifest-level `schema_version`, `coverage`, `coverage_reasons`,
  `omitted_earlier_messages`. The attachment fingerprint already used
  `(type, url, name)` identity (established in `CCMAI-RUNTIME-002`); this
  BUILD reuses that identity concept rather than inventing a second one.

## What changed

### `backend/engine/snapshot.go` (shared canonical comparison helper)

- Extracted the per-message manifest-building logic that previously lived
  inline in `buildConversationSnapshot`'s loop into a new **`canonicalizeMessage(m models.Message) snapshotMessage`**.
  `buildConversationSnapshot` now calls it and reads `sm.ContentType`/
  `sm.AttachmentCoverage` for its existing coverage-reason bookkeeping,
  instead of recomputing them separately. This is a pure refactor — every
  field it returns is byte-for-byte what the inline code computed before, so
  digests are unaffected (confirmed by the existing non-DB snapshot test
  suite still passing unchanged, see Verification).
- New exported **`CompareSnapshotToCurrentMessages(manifestJSON string, currentMessages []models.Message) string`**
  and four exported status constants
  (`SourceIntegrityChangedSinceAnalysis`, `SourceIntegrityBoundCurrentnessUnverified`,
  `SourceIntegrityLegacyUnverified`, `SourceIntegrityVerificationUnavailable`).
  It parses the stored manifest, rejects an unparsable or unsupported-schema
  manifest as `VerificationUnavailable`, then compares every manifest
  message against `canonicalizeMessage(currentMessage)` by internal message
  ID — the identical canonicalization the manifest was built with, so
  "changed" can never mean two different notions of sameness.
  - **Comparison window:** bounded to current messages at or after the
    earliest `sent_at` recorded in the manifest. The manifest's own
    `omitted_earlier_messages` already documents that earlier history may
    have been excluded by design; without the original fetch cutoff stored
    verbatim, comparing against *every* current message would misreport an
    already-known, never-analyzed older message as "added." This is a
    **documented limitation**, not a backdate/tombstone detector — see the
    doc comment on the function and the dedicated test
    `TestCompareSnapshotIgnoresMessageOlderThanAnalyzedWindow`.
  - **A message ID present in the manifest but absent from current
    messages** → `ChangedSinceAnalysis` (covers deleted/missing).
  - **A current message (within the window) absent from the manifest** →
    `ChangedSinceAnalysis` (covers added).
  - **Any field mismatch on a matched ID** (content, sender role/name,
    content type, timestamp, attachment coverage/count/fingerprint) →
    `ChangedSinceAnalysis`.
  - **No mismatch found** → `BoundCurrentnessUnverified`, never a stronger
    claim.
  - Per contract point 2's explicit carve-out, a changed `RawData` payload
    alone does **not** move this status, because `RawData` was never part of
    the `ccma.snapshot.v1` manifest/digest in the first place (confirmed by
    reading `snapshotMessage`'s field list) — there is nothing to compare it
    against, so this is not a gap this tranche introduces or must close.

### `backend/api/handlers/results.go`

- `resultSelect` now also selects `jr.analysis_snapshot_id`; `resultRow`
  gained `AnalysisSnapshotID *string` (internal only, `json:"-"`) and
  `SourceIntegrityStatus string` (`json:"source_integrity_status"`).
- New **`(f resultFilter) attachSourceIntegrity(rows []resultRow) error`**,
  called once at the end of `fetchRows` (shared by `ListResults` and
  `ExportResults`): collects distinct nonempty snapshot IDs and their
  conversation IDs from the page/export rows, issues **at most two** extra
  tenant-scoped batched queries (`analysis_snapshots WHERE tenant_id = ? AND
  id IN (...)`, then `messages WHERE tenant_id = ? AND conversation_id IN
  (...)`) — never one query per row — then assigns each row's status:
  - no snapshot link → `LegacyUnverified`;
  - snapshot ID present but not found in the batch (broken link) →
    `VerificationUnavailable`;
  - otherwise → `engine.CompareSnapshotToCurrentMessages(...)`.
  A failure in either batched query fails the whole request (returns an
  error up through `fetchRows` to a `500`), per contract point 3 ("a
  database-wide read failure may fail the request with an observable server
  error") — this is deliberately different from a row-local problem, which
  becomes `VerificationUnavailable` instead of failing the page.
- `ExportResults` appends one more CSV/XLSX column, "Tính toàn vẹn nguồn",
  to both the QC and classification header/record shapes, via new
  **`sourceIntegrityLabel(status string) string`** — the same four values,
  Vietnamese-labeled to match the rest of that export's existing all-
  Vietnamese labels (`verdictLabel`, etc.). Nothing else about filtering,
  counts, sort, pagination, RBAC, or the export row cap was touched.
- No DB model/migration, analyzer/provider/adapter/sync-dispatch, job-
  specific/conversation API, notification, or scheduler code was touched.

### Frontend (`frontend/src/views/Results.vue`, `frontend/src/i18n/{vi,en}.ts`)

- `ResultItem` gained `source_integrity_status: string`.
- New local helpers `mauNguon`/`iconNguon`/`nhanNguon` map the four status
  values to a color/icon/i18n label — never labeling `bound_currentness_unverified`
  as anything stronger than "not fully verified."
- **Table:** a new narrow icon-only column (with a tooltip carrying the full
  label) before the customer-name column, present for both QC and
  classification rows.
- **Card:** a small chip next to the existing severity/classification chip,
  with a tooltip carrying the local-comparison explanation
  (`results_source_note`).
- **Detail dialog:** a chip in the header row (same four states) plus an
  always-visible caption line explaining the comparison is local-only and
  does not prove full upstream currentness — satisfying contract point 4's
  explicit requirement for that explanation, not just a bare label.
- New i18n keys added to **both** `vi.ts` and `en.ts` (`results_col_source`,
  `results_source_changed`, `results_source_legacy`,
  `results_source_unavailable`, `results_source_unverified`,
  `results_source_note`) — verified by the existing
  `frontend/src/__tests__/i18n.spec.ts` completeness test (same key count,
  no missing/empty keys either direction), which is the one existing
  frontend test/config file the work order names as directly needed here.

## New backend tests

- **`backend/engine/source_integrity_test.go`** (9 pure, non-DB unit tests,
  reusing the existing `snapshotFixture`/`mustSnapshot` helpers from
  `snapshot_test.go`): unchanged → `BoundCurrentnessUnverified`; edited
  content; edited sender role/name; changed attachment identity; missing
  message; added message after the analyzed window → all
  `ChangedSinceAnalysis`; a message older than the analyzed window is
  correctly ignored (documented limitation) → `BoundCurrentnessUnverified`;
  malformed manifest JSON and an unsupported schema version → both
  `VerificationUnavailable`.
- **`backend/api/handlers/results_test.go`** (8 new DB-backed tests, plus a
  pure label-coverage test), proving the *results.go integration* — query,
  batching, and status assignment — end to end against a real disposable-
  MySQL `analysis_snapshots` row (the comparison algorithm itself is already
  covered above, so these hand-construct a manifest JSON matching the
  documented `ccma.snapshot.v1` wire shape rather than calling engine's
  unexported builder from a different package):
  `TestSourceIntegrityUnchangedIsBoundCurrentnessUnverified`,
  `TestSourceIntegrityEditedMessageIsChangedSinceAnalysis` (**the key stale
  case the work order names explicitly**),
  `TestSourceIntegrityMissingMessageIsChangedSinceAnalysis`,
  `TestSourceIntegrityAddedMessageIsChangedSinceAnalysis`,
  `TestSourceIntegrityLegacyResultHasNoSnapshotLink`,
  `TestSourceIntegrityBrokenSnapshotLinkIsVerificationUnavailable`,
  `TestSourceIntegrityCorruptManifestIsVerificationUnavailable`,
  `TestSourceIntegrityLabelCoversAllFourStatuses`. Tenant isolation for this
  feature specifically was not given its own new test: `attachSourceIntegrity`'s
  two batched queries both filter by `f.tenantID`, and every row it operates
  on is already tenant-scoped by `fetchRows`'s existing predicates (proven
  by the pre-existing `TestKhongLoDuLieuSangCongTyKhac`), so there is no
  additional cross-tenant surface this feature introduces to test in
  isolation; this reasoning is recorded here rather than asserted silently.

### Before/after regression (proves the tests are not vacuous)

`git stash push -- backend/engine/snapshot.go backend/api/handlers/results.go`
temporarily restored pre-BUILD source (keeping the new test files). Both
`go vet ./engine/... ./api/handlers/...` failed to compile:

```
vet.exe: engine\source_integrity_test.go:20:9: undefined: CompareSnapshotToCurrentMessages
vet.exe: api\handlers\results_test.go:424:9: row.SourceIntegrityStatus undefined (type resultRow has no field or method SourceIntegrityStatus)
```

This is the strongest possible proof the new tests are not vacuous — the
feature (and the key stale-message case) did not exist at all before this
BUILD. `git stash pop` restored the implementation, re-verified passing
below.

## Verification

- `go build ./...`, `go vet ./...` — clean.
- `gofmt -l` on every changed/new backend file — clean (one file,
  `results_test.go`, needed one `gofmt -w` pass for map-literal alignment,
  applied before this evidence was written; re-checked clean afterward).
  `git diff --check` — clean.
- Non-DB engine tests (`TestSnapshotDigestIsDeterministic`,
  `TestSnapshotDigestChangesWithSource`,
  `TestSnapshotDigestChangesWithAttachmentIdentity`, `TestSnapshotCoverage`,
  `TestSnapshotTranscriptCarriesIdentity`,
  `TestSnapshotRejectsForeignMessage`) — PASS unchanged, confirming the
  `canonicalizeMessage` extraction did not alter any digest.
- Disposable `mysql:8.0` container on an isolated Docker network (no host
  data; container and network removed after the run); persistent Compose
  `ccma` was not started (no schema/migration change).
  `log_bin_trust_function_creators` set once via root so the pre-existing
  trigger-based regressions in this suite (unrelated to this tranche, e.g.
  `TestDeleteChannelFailureRollsBackWholeCascade`) keep passing:
  - `go test ./engine/... -run 'TestCompareSnapshot' -v` — all 9 new unit
    tests PASS.
  - `go test ./api/handlers/... -run 'TestSourceIntegrity|TestFetchRows|TestVerdictCounts|TestLocTheoDiem|TestKhongLoDuLieu' -v` —
    all 8 new integration tests plus 4 pre-existing `results_test.go` tests
    PASS.
  - `go test ./... -count=1` — all 13 packages `ok`.
  - `AutoMigrate` run twice back-to-back on the same disposable `CCMA`
    schema (throwaway `go run`, no `-mod=mod`) — both clean, no error;
    `backend/go.mod`/`backend/go.sum` confirmed unchanged by `git status`
    both before and after.
- Frontend: `npm run build` (`vue-tsc -b && vite build`) — PASS, confirming
  the new `ResultItem.source_integrity_status` field and all new template
  bindings type-check and bundle cleanly. `npx vitest run` — the existing
  `i18n.spec.ts` (5 assertions) PASS, confirming both language files stay
  key-complete and non-empty after the six new keys were added to each.
- Downstream catalog check (`scripts/manage_cvf_downstream_catalog.ps1
  -Check`) — PASS.
- CVF workspace doctor
  (`../.Controlled-Vibe-Framework-CVF/scripts/check_cvf_workspace_agent_enforcement.ps1
  -ProjectPath .`) — PASS 25/25.
- Cleanup: the disposable MySQL container and its dedicated Docker network
  were stopped/removed after the run.

## Untested / documented limitations

- The bounded comparison window (current messages at or after the
  manifest's earliest recorded `sent_at`) means an edit or deletion of a
  message *older* than everything the snapshot recorded is out of scope —
  this is the same "adapter history/removal coverage may be incomplete"
  limitation the SPEC names, and is exercised (as a deliberate non-trigger)
  by `TestCompareSnapshotIgnoresMessageOlderThanAnalyzedWindow`.
- A changed `RawData` payload alone never moves `source_integrity_status`,
  because `RawData` was never part of the digest-bearing manifest — this is
  a pre-existing manifest-scope fact, not a new gap, and is documented in
  the SPEC contract point 2's own carve-out.
- No dedicated tenant-isolation test was added for this feature
  specifically; the reasoning for relying on existing coverage is recorded
  above rather than left implicit.
- Frontend verification was a successful type-checked build plus the
  existing i18n completeness test, not a new component-level UI test (no
  existing Results.vue test file exists to extend, and the work order's
  allowance is "a build" as an explicit alternative to UI tests).

## Claim boundary

No Claude/Gemini/OpenAI/xAI or other provider API was called and no API key
was used. No real channel sync, customer data, deployment, or push. This
BUILD makes no CVF runtime-governance claim and no production-freshness
guarantee — `source_integrity_status` is explicitly a local, informational
signal per the SPEC, never a claim of verified upstream currentness. It does
not reopen, re-certify, or widen `CCMAI-RUNTIME-001/002/003` (all remain
REVIEW PASS / FREEZE open, unaffected). No S2/S3/S5 implementation or FREEZE
is authorized by this BUILD. Local commit only; Codex is the independent
`REVIEWER` next, and Claude does not self-approve or FREEZE.
