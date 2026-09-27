# BUILD evidence: CCMAI-RUNTIME-006 — S1 job-result source integrity

**Implementer:** Claude (`IMPLEMENTATION_WORKER`) · **Date:** 2026-09-27 · **Risk:** R2 · **Authority:** [SPEC](../specs/RUNTIME_JOB_RESULT_SOURCE_INTEGRITY_S1_2026-09-27.md), [work order](../work_orders/CCMAI_RUNTIME_006.md) · **Status:** `REVIEW_PENDING` for independent Codex REVIEW. No FREEZE.

## Entry

Claude acknowledged `WORK_ORDER_AUTHOR (Codex) -> IMPLEMENTATION_WORKER (Claude)` in the active handoff at `c24ec4f`, after rehydrating manifest, policy, state, handoff, memory, implementation status, docs index, SPEC and work order. All changes stay inside the allowed paths; no path addition was needed. Engine source was reused, not edited.

## Trace (before editing)

| Path | Grouping / behavior found |
|---|---|
| `ListJobResults` (run) | Flat `[]models.JobResult`, newest first. The query error was ignored. No status. |
| `ListAllJobResults` (job) | Flat `JobResultWithConvDate` (embeds `JobResult`, plus conversation date/customer) across all of the job's runs. The run-ID and result query errors were ignored. |
| `ExportJobResults` QC | Groups **all runs'** results per conversation, in result order (`created_at DESC`); the last evaluation processed wins verdict/review/score. Every query error was ignored. |
| `exportClassification` | Same all-runs grouping. Chat text came from **one query per conversation**, read at export time rather than from the snapshot, with errors ignored. |
| `JobDetail.vue` | Groups by conversation but keeps only each conversation's **latest run**. Presents the groups in a QC table, a classification table, QC/classification cards, and a detail dialog. There was no source warning anywhere. |
| R004 `attachSourceIntegrity` | Two tenant-scoped batched queries (snapshots, then messages). `VerifySnapshotProvenance` runs before `CompareSnapshotToCurrentMessages`. |

The two grouping paths therefore differ. The page summarizes the latest run's results; the export summarizes every run's results for a conversation. Each surface shows the distinct statuses of exactly the results it groups.

## Implementation

- **Shared evaluator (`backend/api/handlers/source_integrity.go`).** `computeSourceIntegrity(tenantID, refs)` is R004's logic moved unchanged, except that the two IN lists are now chunked at 500 IDs (`chunkStrings`). Query cost is `ceil(snapshots/500) + ceil(conversations/500)` reads, never one per result. Status rules and error strings are identical:
  - no link → `legacy_unverified`;
  - missing, cross-tenant or provenance-invalid snapshot → `verification_unavailable`;
  - otherwise R004's comparison.

  Any batch error is returned. `distinctSourceIntegrityLabels` renders all distinct statuses, most concerning first (changed → unavailable → legacy → bound).
- **Aggregate Results (`results.go`).** `attachSourceIntegrity` is now a thin wrapper over the shared evaluator. Its behavior and tests are unchanged.
- **Job JSON (`jobs.go`).**
  - `ListJobResults` returns `JobResultWithIntegrity` (embedded `JobResult` plus `source_integrity_status`).
  - `ListAllJobResults` returns `JobResultWithConvDate` with a new `source_integrity_status`.
  - Both check every query error and return `500 {"error":"query_failed"}` before writing any success body.
  - R005 `confidence`/`confidence_basis`, `evidence_status` and every other field pass through the embedded model unchanged.
- **Job exports (`jobs.go`).**
  - One shared loader (`loadJobResultsWithIntegrity`) with checked errors. The job-type lookup and the classification chat read are also checked, and all of them run before any header or byte is written.
  - The chat read is now batched: chunked IN, ordered `conversation_id, sent_at, id`, with the same `[name] content` format as before.
  - QC and classification exports each add two columns: `Tính toàn vẹn nguồn` (all distinct statuses in the group, most concerning first, never collapsed) and `Chi tiết toàn vẹn nguồn (mã kết quả)` (`<result id>: <label>` for every grouped result, for traceability).
  - The classification chat header now reads `Nội dung chat (đọc lúc xuất file, có thể khác bản đã phân tích)`.
  - Existing row selection, grouping, ordering and other columns are unchanged.
  - Writing now reuses `writeResultsCSV` / `writeResultsXLSX`. **Observable difference:** CSV header cells are now quoted, as the aggregate export already does. Values and column order are the same.
- **Frontend.**
  - `stores/jobs.ts`: the `SourceIntegrityStatus` type, the `source_integrity_status` field, and a pure `distinctSourceIntegrity` helper (a missing or unknown value counts as unavailable, never dropped). The same label keys as R004 (`results_source_*`) plus icons and colors.
  - `JobDetail.vue`: a new "Source" column with icon and tooltip in both tables; label chips in both card headers; label chips plus the always-visible `results_source_note` local-only caveat in the detail dialog. Every distinct status in a group is shown, so a changed result cannot be hidden by others. Nothing is persisted, triggered or re-analyzed, and there is no RBAC change.
  - New i18n key `job_source_col` (`Nguồn` / `Source`).

## Tests

| Test (disposable MySQL unless noted) | Proves |
|---|---|
| `TestJobResultEndpointsCarrySourceIntegrityStatus` | Real `ListJobResults` (5 rows) and `ListAllJobResults` (6 rows) return the exact status per result: changed (old run of an edited chat), bound (new run), legacy (no link), unavailable (corrupt digest), unavailable (well-formed snapshot owned by another tenant). R005 `confidence: null` / `unavailable`, `evidence` and `evidence_status` are preserved. Another tenant sees no rows from either endpoint. |
| `TestExportJobResultsShowsDistinctGroupStatuses` (QC and classification × CSV and XLSX) | Exact headers. The mixed conversation (changed + bound across runs) shows `changed; bound` in that order, and its detail attributes the change to the old result. Legacy, corrupt and cross-tenant groups show their own label. Existing QC issue/score and classification tag/chat columns are unchanged. The chat header carries the export-time caveat. Another tenant's export has only the header. |
| `TestJobResultQueryFailuresAreObservable` (`analysis_snapshots` and `messages` made unreadable) | Both JSON endpoints, the QC export and the classification export (CSV and XLSX) each return 500 JSON `query_failed`, with no `Content-Disposition`, no BOM and no `PK` prefix. |
| `TestChunkStringsCoversEveryItemOnce`, `TestDistinctSourceIntegrityLabelsLeadsWithMostConcerning` (unit) | Chunk boundaries (0, 1, 500, 501, 1003) keep every ID once and in order. The label order leads with changed. |
| All R004 aggregate tests (`TestSourceIntegrity*`, `TestListResults*`, `TestExportResults*`, `TestFetchRows*`, etc.) | Still pass after the helper extraction. |
| `frontend/src/__tests__/i18n.spec.ts` (vitest) | Every status label key, `job_source_col` and the note exist non-empty in `vi` and `en`. `distinctSourceIntegrity` lists every status of a mixed group, changed first, and maps missing or unknown values to unavailable. |

No component-mount test for `JobDetail.vue` exists or was added. The visible grouping logic lives in the tested pure helper, and template bindings are type-checked by `vue-tsc`. `i18n.spec.ts` is the only existing frontend test file and covers these labels, so it was extended rather than creating a new test path.

**Non-vacuity.**
- Mutation: `distinctSourceIntegrityLabels` was changed to return only the least concerning label. The mixed-group export assertions (QC CSV and XLSX) and the order unit test then failed with `mixed group status = "Chưa xác minh đầy đủ (so sánh cục bộ)"`. The source was restored from a backup, with no marker left.
- Before/after: with only `jobs.go` stashed to pre-change source, the new test file fails to compile (`undefined: exportIntegrityHeader`). The file was restored with `git stash pop`.

## Verification

This host's Windows Application Control still blocks newly built Go test binaries, as recorded in R005, and it was not bypassed. Go tests ran in `golang:1.26-alpine` (go1.26.8) on disposable `mysql:8.0` `ccma-r006-db` / network `ccma-r006-net` (no host data, `log_bin_trust_function_creators=1`), with the module cache read-only, `GOPROXY=off` and `GOFLAGS=-mod=readonly`.

```text
host: go build ./... ; go vet ./...                                          clean
container: go test ./api/handlers -run 'TestJobResult|TestExportJobResults|TestSourceIntegrity|TestListResults|TestExportResults|TestFetchRows|TestVerdictCounts|TestLocTheoDiem|TestKhongLoDuLieu' -count=1 -v
                                                                            all PASS (incl. 5 export, 2 failure-table and 3 endpoint subtests)
container: go build ./... && go vet ./... && go test ./... -count=1 -p 1     all 13 packages ok
mutation (collapse mixed group)                                             3 assertions FAIL as expected; restored
pre-change jobs.go                                                          new tests fail to compile; restored
frontend: npx vitest run                                                    8/8 PASS (5 existing + 3 new)
frontend: npm run build (vue-tsc -b && vite build)                          PASS
gofmt -l (LF-normalized) on changed Go files                                clean
git diff --check                                                            clean
scripts/manage_cvf_downstream_catalog.ps1 -Check                            PASS
check_cvf_workspace_agent_enforcement.ps1 -ProjectPath .                    PASS 25/25
```

Cleanup: the disposable container and network were removed. The persistent Compose `ccma` stack was not started, reset or touched. No DB model or migration changed.

## Limits and untested branches

- A status is a local comparison against current rows only. It does not prove upstream completeness, and nothing was read from a real channel.
- The export loads every message of the grouped conversations for the comparison, in chunks. Memory grows with conversation size. The job export has no row cap, as before; that cap is outside this scope.
- The error path of the classification chat-text query is covered only indirectly: when `messages` is unreadable, the earlier results subquery fails first. Its own check is present in the source but is not isolated by a test.
- The detail column uses full result IDs, which is long but traceable.
- Conversation-specific APIs, notifications, re-analysis and S2/S3/S5 remain out of scope. No provider API, credential, customer data, deployment or push was used, and this is not live CVF governance proof.

## Addendum: Repair round 1 (R006-R1 local-only caveat visibility)

**Entry:** Codex REVIEW of `61a775a` (`docs/reviews/CCMAI_RUNTIME_006_INDEPENDENT_REVIEW_2026-09-27.md`, committed at `3b31aeb`) accepted the backend, API and export work. It returned `CHANGES_REQUIRED` for R006-R1 only: `results_source_note` was visible only inside the Job Detail dialog, not on the primary table and card views. Claude recorded `REVIEWER (Codex) -> REPAIR_WORKER (Claude)` in the active handoff before editing.

**Change:** `frontend/src/views/Jobs/JobDetail.vue` only. A `v-alert` (info, tonal) showing the existing bilingual `results_source_note` is placed inside the results tab, directly after the filter/toolbar row and before the table/card switch. It renders whenever result groups are shown (`v-if="filteredGroupedResults.length"`), so it is visible in both views, including classification and QC tables and cards. Status badges and the dialog note are unchanged. There is no change to status computation, backend, i18n keys, store or any other product behavior.

**Placement by source inspection** (`grep -n` line numbers after the edit):

```text
284  <div v-if="activeTab === 'results'">                 results tab
341  {{ $t('results_source_note') }}                      new always-visible note (alert opens at line 340)
344  <div v-if="!filteredGroupedResults.length" ...>      empty state
348  <div v-else-if="viewMode === 'table' && isClassification">
378  <div v-else-if="viewMode === 'table'">               QC table
423  <v-card v-for="group in paginatedResults" ...>       card view
640  dialog note (unchanged)
```

**Verification:**

```text
frontend: npx vitest run src/__tests__/i18n.spec.ts                        8/8 PASS (label keys incl. results_source_note, grouping)
frontend: npm run build (vue-tsc -b && vite build)                         PASS
git diff --check                                                           clean
scripts/manage_cvf_downstream_catalog.ps1 -Check                           PASS
check_cvf_workspace_agent_enforcement.ps1 -ProjectPath .                   PASS 25/25
```

No backend source changed, so the backend suite was not re-run this round; the prior run and Codex's independent run stand. The placement is checked by source inspection and the type-checked build, not by a rendered-component test; no component-mount test exists for this view. No provider API, credential, channel sync, customer data, database, deployment or push was used, and no upstream-completeness or live-governance claim is made. Status: `REVIEW_PENDING` for Codex re-review.
