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

## Addendum: Repair round 1 (R004-R1/R004-R2/R004-R3)

**Entry:** Codex independent REVIEW of the BUILD commit above (`a9559e3`)
returned `CHANGES_REQUIRED` at
`docs/reviews/CCMAI_RUNTIME_004_INDEPENDENT_REVIEW_2026-09-27.md` for three
same-scope findings. Role transition `REVIEWER (Codex) -> REPAIR_WORKER
(Claude)` acknowledged in the active handoff before this round; scope is
exactly the four files the repair addendum in
`docs/work_orders/CCMAI_RUNTIME_004.md` allows.

### R004-R1 — linked-snapshot provenance validation

New exported `engine.VerifySnapshotProvenance(snap models.AnalysisSnapshot,
resultTenantID, resultConversationID, resultJobRunID string) bool`
(`backend/engine/snapshot.go`) checks, in order: the snapshot's own
`SchemaVersion` is `ccma.snapshot.v1`; `snap.TenantID`/`ConversationID`/
`JobRunID` match the result linking to it; SHA-256 of the stored manifest
bytes equals `snap.Digest`; the parsed manifest's own `schema_version`,
`tenant_id` and `conversation_id` fields also match; and `len(manifest.Messages)
== snap.MessageCount`. `attachSourceIntegrity`
(`backend/api/handlers/results.go`) now calls this before ever invoking
`CompareSnapshotToCurrentMessages`; any failure becomes
`verification_unavailable`, never a locally-matching result. The prior BUILD
round's `results_test.go` fixture hardcoded digest `"deadbeef"` and
`message_count: 1` regardless of the actual manifest — this repair replaces
that with a real SHA-256 digest and a manifest-derived message count
(`manifestMessageCount`), so the existing happy-path fixtures keep passing
under the new validation instead of falsely becoming
`verification_unavailable`.

New regressions: 7 engine unit tests
(`TestVerifySnapshotProvenanceAcceptsValidSnapshot` plus one rejection test
each for corrupt digest, wrong conversation link, wrong job-run link,
cross-tenant link, message-count mismatch, and unsupported schema) built
against the exact `(*conversationSnapshot).record` path production code
uses; 4 handler integration tests on disposable MySQL
(`TestSourceIntegrityCorruptDigestIsVerificationUnavailable`,
`TestSourceIntegrityWrongConversationLinkIsVerificationUnavailable`,
`TestSourceIntegrityWrongJobRunLinkIsVerificationUnavailable`,
`TestSourceIntegrityCrossTenantSnapshotLinkIsVerificationUnavailable`). The
cross-tenant case is covered at both levels: the engine unit test exercises
`VerifySnapshotProvenance`'s own tenant-ID comparison directly (the only way
to reach that branch, since the real tenant-scoped SQL lookup in
`attachSourceIntegrity` would never load a snapshot belonging to a different
tenant in the first place); the handler test proves that first line of
defense — the batched query itself — stays tenant-scoped after this repair.

### R004-R2 — added-earlier-message detection

`CompareSnapshotToCurrentMessages` (`backend/engine/snapshot.go`) now counts
current messages older than the manifest's bounded comparison window
(`earlierCount`) and compares that count against the manifest's own
`OmittedEarlierMessages`. A mismatch — including the central counterexample
from the review (zero omitted, one message inserted before the manifest's
earliest timestamp) — now returns `changed_since_analysis` instead of being
silently absorbed into `bound_currentness_unverified`. A matching count
still cannot identify which individual earlier messages are involved from
the stored manifest alone, so it is preserved as
`bound_currentness_unverified` — the no-positive-freshness claim is
unchanged.

This required updating one pre-existing test,
`TestCompareSnapshotIgnoresMessageOlderThanAnalyzedWindow`: it recorded
`omitted_earlier_messages: 3` but only ever added one older current message,
which the corrected count-comparison logic now (correctly) flags as a proven
count change. It is replaced by
`TestCompareSnapshotWithMatchingOmittedCountStaysBoundCurrentnessUnverified`,
which adds exactly three older messages to match the recorded count and
still asserts `bound_currentness_unverified` — preserving the original
test's documented intent (an unidentifiable-but-unchanged earlier window)
under semantics that no longer accept an unchecked count. Two new tests
cover the R004-R2 acceptance directly: zero-omitted-count
(`TestCompareSnapshotDetectsAddedEarlierMessageWithZeroOmittedCount`) and
nonzero-omitted-count-mismatch
(`TestCompareSnapshotDetectsEarlierMessageCountChangeWithNonzeroOmittedCount`).

### R004-R3 — endpoint/export/isolation/error-path evidence

Three new handler tests on disposable MySQL exercise the actual HTTP
handlers via `httptest`/`gin.CreateTestContext`, not just `fetchRows`:
- `TestListResultsResponseIncludesSourceIntegrityStatus` calls `ListResults`
  directly and asserts the JSON page response's
  `items[].source_integrity_status` for an edited-message row.
- `TestExportResultsCSVAndXLSXIncludeSourceIntegrityColumn` calls
  `ExportResults` for both `format=csv` and `format=xlsx` against the same
  unchanged row and asserts both carry the identical
  `sourceIntegrityLabel(bound_currentness_unverified)` value in their last
  column (CSV as a substring match on the rendered file; XLSX opened with
  `excelize.OpenReader` and read back via `GetRows`) — proving export-format
  equivalence, not just that each format compiles.
- `TestListResultsSnapshotBatchQueryFailureIsObservable` forces
  `attachSourceIntegrity`'s first batched query (`analysis_snapshots`) to
  fail by renaming that table for the duration of the test
  (`RENAME TABLE analysis_snapshots TO analysis_snapshots_forced_failure`,
  restored in `t.Cleanup` before the fixture's own row-deletion cleanup
  runs, via Go's LIFO cleanup order) and asserts `ListResults` returns a
  non-2xx status rather than a page that silently omits or
  mis-states the affected rows. MySQL triggers (the technique this
  codebase's other failure-injection tests use, e.g.
  `TestDeleteChannelFailureRollsBackWholeCascade` and
  `TestResetDemoDataFailureRollsBackEverything`) cannot intercept a `SELECT`
  the way they intercept `DELETE`/`UPDATE`, so this uses a disposable
  -MySQL-only table rename instead; it is scoped to one test, undone
  unconditionally, and this package's tests run sequentially (no
  `t.Parallel`), so no other test observes the renamed table. The
  `messages` query in the same function follows an identical
  fail-immediately pattern (`if err != nil { return fmt.Errorf(...) }`), so
  this one representative case is treated as covering both; a second,
  near-identical test renaming `messages` instead was judged to add
  duplicate coverage rather than new proof and was not added.
- The `TestSourceIntegrityCrossTenantSnapshotLinkIsVerificationUnavailable`
  test listed under R004-R1 above also satisfies the work order's
  tenant-isolation requirement for this feature's newly joined
  snapshot/message reads.

### Repair round 1 validation

- `TEST_DB_DSN=<disposable mysql:8.0, isolated Docker network, no host
  data, log_bin_trust_function_creators=1 set once via root> go test
  ./engine/... -run 'TestCompareSnapshot|TestVerifySnapshotProvenance' -v`
  — 18 tests PASS (9 pre-existing digest/coverage-neutral tests untouched by
  this round were not in this filter; the 9 `TestCompareSnapshot*`/7
  `TestVerifySnapshotProvenance*` tests targeted by this round all PASS,
  2 of which are new-name replacements of the one test this round revised).
- `go test ./api/handlers/... -run
  'TestSourceIntegrity|TestFetchRows|TestVerdictCounts|TestLocTheoDiem|TestKhongLoDuLieu|TestListResults|TestExportResults'
  -v` — 19 tests PASS (11 pre-existing plus 8 new).
- `go test ./... -count=1 -p 1` — all 13 packages `ok`, no regressions in
  any package this round did not touch.
- Before/after proof: `git stash push -- backend/engine/snapshot.go
  backend/api/handlers/results.go` (keeping both test files) then `go vet
  ./engine/... ./api/handlers/...` failed to compile:
  `vet.exe: engine\source_integrity_test.go:212:6: undefined:
  VerifySnapshotProvenance` — the R004-R1 tests cannot even compile against
  pre-repair source. `git stash pop` restored the fix, re-verified passing
  above. (The R004-R2 count-comparison logic change could not be isolated
  the same way without also removing the R004-R1 symbol the same file now
  needs to compile; its behavior change was instead verified by tracing the
  pre-repair function, which never counted or compared earlier-window
  messages at all, so both new R004-R2 tests would have returned
  `bound_currentness_unverified` — the compile-failure proof above already
  demonstrates neither new test file is vacuous.)
- `go build ./...`, `go vet ./...` — clean.
- `gofmt -l` on all four changed files flags `engine/snapshot.go` and
  `api/handlers/results.go` only because of the same `core.autocrlf`-driven
  CRLF conversion documented in prior rounds; confirmed clean with `gofmt
  -l` after stripping `\r` into a scratch copy. `git diff --check` — clean
  (only the expected "LF will be replaced by CRLF" advisory notices, no
  actual whitespace errors).
- `AutoMigrate` run twice back-to-back against the same disposable schema
  (throwaway `go run`, no `-mod=mod`) — both clean, no error;
  `backend/go.mod`/`backend/go.sum` confirmed unchanged by `git status`
  before and after.
- Frontend: not touched this round (no allowed frontend path was needed for
  these three findings), so no frontend build/test was re-run.
- Downstream catalog check (`scripts/manage_cvf_downstream_catalog.ps1
  -Check`) — PASS. CVF workspace doctor — PASS 25/25.
- Cleanup: the disposable MySQL container and its dedicated Docker network
  were stopped/removed after the run; the persistent Compose `ccma` stack
  (`ccma-app-1`, `ccma-db-1`) was never started, reset or touched.

### Repair round 1 remaining limitations

- The bounded comparison window's older-message handling is now
  count-based, not identity-based: a nonzero omitted count that stays the
  same but has one earlier message silently swapped for a different one
  (same count) remains undetectable from the stored manifest alone. This is
  the same class of limitation the SPEC already names for adapter
  history/removal coverage, narrowed rather than newly introduced by this
  repair.
- No production/runtime governance claim is made; `source_integrity_status`
  remains a local, informational signal only. No S2/S3/S5, provider call,
  real channel sync, customer data, deployment or FREEZE is authorized by
  this repair round.

## Addendum: Repair round 2 (R004-R3-T1 test/evidence completion)

**Entry:** Codex re-review of repair commit `a074870`
(`docs/reviews/CCMAI_RUNTIME_004_REPAIR_R1_REREVIEW_2026-09-27.md`)
accepted R004-R1/R004-R2 and returned `CHANGES_REQUIRED_ROUND_2` for
R004-R3-T1 only. Role transition `REVIEWER (Codex) -> REPAIR_WORKER
(Claude)` was acknowledged in the active handoff at `c7e4145`. The only
changed source file is `backend/api/handlers/results_test.go`. No production
source, model/migration, frontend or CVF core file changed, and no
product-source defect was found.

### Changes

- `TestExportResultsCSVAndXLSXIncludeSourceIntegrityColumn` now has two
  subtests. `unchanged` expects `bound_currentness_unverified`. `changed`
  edits the source message after analysis and expects
  `changed_since_analysis`. Each subtest calls the real `ExportResults`
  handler for `format=csv` and `format=xlsx`. For CSV, the test parses the
  output with `encoding/csv` after removing the BOM. For XLSX, it reads the
  file back with `excelize`. In both formats it asserts that the header's
  last column is `Tính toàn vẹn nguồn` and that the one data row's last cell
  is exactly `sourceIntegrityLabel(<expected status>)`. Round 1 used a
  substring match; these are exact per-column checks.
- New `TestExportResultsSnapshotBatchQueryFailureIsObservable` forces the
  batched `analysis_snapshots` SELECT to fail and calls `ExportResults` for
  both `csv` and `xlsx`. For each format it asserts:
  - the status is 4xx/5xx;
  - there is no `Content-Disposition` download header;
  - `Content-Type` is `application/json`;
  - the body has no CSV BOM and no XLSX `PK` prefix;
  - the body decodes to `{"error":"query_failed"}`.
- The table-rename failure injection is now the shared
  `forceSnapshotQueryFailure(t)` helper. The existing
  `TestListResultsSnapshotBatchQueryFailureIsObservable` uses it with the
  same behavior. The helper's cleanup restores the table before the
  fixture's own DELETEs run, because Go runs cleanups in reverse order. A
  restore failure is now reported with `t.Errorf` so the remaining cleanups
  still run.

### Validation (disposable MySQL)

`mysql:8.0` container `ccma-r004-r2-db` on its own Docker network
`ccma-r004-r2-net` (host port 33072, no host data), schema `CCMA`,
`SET GLOBAL log_bin_trust_function_creators=1` via root.
`TEST_DB_DSN=ccma:***@tcp(127.0.0.1:33072)/CCMA?...`, `GOFLAGS=-mod=readonly`.

```text
go vet ./api/handlers/                                                     clean
go test ./engine ./api/handlers -run 'TestCompareSnapshot|TestVerifySnapshot|TestSourceIntegrity|TestListResults|TestExportResults|TestFetchRows|TestVerdictCounts|TestLocTheoDiem|TestKhongLoDuLieu' -count=1 -v
    engine: 18 PASS; handlers: 21 top-level PASS, including
    TestExportResultsCSVAndXLSXIncludeSourceIntegrityColumn/{unchanged,changed},
    TestListResultsSnapshotBatchQueryFailureIsObservable,
    TestExportResultsSnapshotBatchQueryFailureIsObservable/{csv,xlsx}
go test ./... -count=1 -p 1                                                all 13 packages ok
SHOW TABLES LIKE 'analysis_snapshots%' (after suite)                      analysis_snapshots only (rename restored)
gofmt -l results_test.go (LF-normalized copy)                             clean
git diff --check                                                           clean (autocrlf advisory only)
backend/go.mod, backend/go.sum                                             unchanged
scripts/manage_cvf_downstream_catalog.ps1 -Check                           PASS
check_cvf_workspace_agent_enforcement.ps1 -ProjectPath .                   PASS 25/25
```

Existing filter, count, order and tenant tests (`TestFetchRows*`,
`TestVerdictCounts*`, `TestLocTheoDiem*`, `TestKhongLoDuLieu*`) still pass.
The export-cap branch was not changed. No frontend build was run because
frontend source is outside this round's scope.

Cleanup: the disposable container and network were removed. The persistent
Compose `ccma` stack (`ccma-app-1`, `ccma-db-1`) was not started, reset or
touched.

### Boundary

This round adds test evidence only. No provider call, API key, real channel
sync, customer data, deployment or push was used. This is not live CVF
governance proof. It does not claim upstream source completeness. It
authorizes no S2/S3/S5 work and no FREEZE. Status is `REVIEW_PENDING` for
Codex's independent re-review.
