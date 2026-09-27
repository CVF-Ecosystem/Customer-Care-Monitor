# BUILD evidence: S1 snapshot và evidence contract

**Work order:** `CCMAI-RUNTIME-002` Gate B · **Spec:** `docs/specs/RUNTIME_SNAPSHOT_EVIDENCE_S1_2026-09-27.md` · **Ngày:** 2026-09-27 · **Implementer:** Claude (IMPLEMENTATION_WORKER) · **Kết quả:** BUILD_PASS / REVIEW_PENDING (Codex independent REVIEWER).

Gate A precondition: `docs/reviews/CCMAI_RUNTIME_001_INDEPENDENT_REVIEW_2026-09-27.md` — `PASS_WITH_REPAIRS`, repair round 1 re-verified.

## Changed set

| Path | Change |
|---|---|
| `backend/engine/snapshot.go` (new) | `loadConversationSnapshot` + `buildConversationSnapshot`: one builder for single and batch; ordered `sent_at` UTC then `message.id`; query errors returned; manifest `ccma.snapshot.v1` with ID/metadata/content SHA-256 and code-point length; SHA-256 lowercase hex digest; typed coverage; transcript with `msg:<id>` and RFC3339 +07:00 per line; `validateEvidenceRefs`. |
| `backend/engine/analyzer.go` | Single and batch paths call `loadConversationSnapshot`; load/build error counts as conversation error. `saveResults(runID, snap, …)` validates every finding's refs before writing, then inserts snapshot + results in one transaction; results carry `analysis_snapshot_id`; `detail.evidence_refs` stores refs. |
| `backend/db/models/analysis_snapshot.go` (new), `backend/db/mysql.go` | `analysis_snapshots` table, unique `(job_run_id, conversation_id)`, index `(tenant_id, conversation_id)` and `digest`; manifest `mediumtext` (see finding below). AutoMigrate registration. |
| `backend/db/models/job.go` | Nullable `job_results.analysis_snapshot_id`; derived `evidence_status` (`snapshot_bound` / `legacy_unverified`) via `AfterFind`. File re-formatted by `gofmt` (alignment only). |
| `backend/ai/prompts.go` (+test) | QC and classification JSON contract require `evidence_refs` (`message_id`, exact `quote`, optional code-point `start`/`end`); `FormatChatTranscript` emits source identity when `MessageID` is set. |
| `backend/api/handlers/channels.go`, `jobs.go`, `demo.go` | Declared dependency: delete `analysis_snapshots` alongside results/conversations (channel delete/purge, job delete/clear results/clear runs, demo reset). |
| `backend/engine/snapshot_test.go`, `snapshot_db_test.go` (new), `reanalyze_test.go` | Unit and MySQL-backed tests; existing reanalyze test double now cites real message refs; fixture cleanup includes snapshots. |

Gate A repair files (`scheduler.go`, `sync_review_repair_test.go`, `frontend/src/views/Channels.vue`) are recorded in the Gate A review.

## Schema and migration outcome

- New table `analysis_snapshots` (11 columns), `job_results.analysis_snapshot_id char(36) NULL` with index. No backfill: existing rows stay `NULL` and read as `legacy_unverified`.
- Disposable `mysql:8.0` test schema: dropped/recreated, AutoMigrate run twice clean; `SHOW CREATE TABLE` shows the unique/secondary indexes above.
- Persistent Compose `ccma` / schema `CCMA`: `docker compose -p ccma up -d --build app`, then a second `restart`; both logs show `Database migration completed`, pricing sync disabled (`dùng bảng tĩnh`), `loaded 0 cron jobs`, no error/duplicate-index line. Schema now has 17 tables; `manifest mediumtext NOT NULL`, `analysis_snapshot_id` nullable. Business rows remain 0 (tenants, conversations, messages, job_results, analysis_snapshots, ai_usage_logs).

Finding during BUILD: storing the manifest in a MySQL `JSON` column made the stored text differ from the hashed canonical bytes (MySQL re-serializes JSON), so `SHA-256(stored manifest)` would not equal `digest`. The column is `mediumtext`; `TestSingleAndBatchShareSnapshotContract` now asserts the stored manifest hashes to the stored digest.

## Acceptance mapping

| # | Acceptance | Evidence | Result |
|---|---|---|---|
| 1 | Same data → same digest; content/role/timestamp/content type/attachment change → new digest | `TestSnapshotDigestIsDeterministic` (reordered load, other timezone), `TestSnapshotDigestChangesWithSource` (5 mutations, 1 ns timestamp shift) | PASS |
| 2 | Vietnamese with diacritics + emoji, code-point offsets | `TestEvidenceRefsUnicodeCodePointOffsets` (valid `[9,19)` over `😡`; byte offsets rejected) | PASS |
| 3 | Reject unknown message, wrong quote, wrong offset, empty refs, cross-conversation | `TestEvidenceRefsRejectInvalid` (10 cases), `TestBatchRejectsCrossConversationRef` (real batch run: conversation citing its batch neighbour rejected, the other saved) | PASS |
| 4 | Single and batch share one builder and persist snapshot relation for every valid result | Source: both paths call only `loadConversationSnapshot`. `TestSingleAndBatchShareSnapshotContract` runs single (`ai_batch_mode=false`) and batch: 2 snapshots, 4 results each, every result linked to its conversation snapshot; stored manifest/digest equal a rebuild through the shared loader; SHA-256(stored manifest) = digest | PASS |
| 5 | Load/build/persist failure = conversation error, no success, no orphan result | `TestInvalidEvidenceRejectsWholeConversation` (single + batch: errors=2, 0 results, 0 snapshots); `TestSaveResultsRollsBackSnapshotOnFailure` (insert failure after snapshot insert rolls both back; duplicate snapshot persist fails without extra results); load errors increment `errorCount` in both paths | PASS |
| 6 | Backend test/build, frontend build, Compose restart on `CCMA`, docs build, catalog, doctor, diff check | See Validation | PASS |
| 7 | No real provider, channel sync, customer data; no gate/cost claim | See Claim boundary | PASS |

Compatibility: `TestLegacyResultsAreMarkedUnverified` — legacy row serializes `"analysis_snapshot_id":null` and `"evidence_status":"legacy_unverified"`. API, export, notification and MCP read `models.JobResult` via GORM and only gain fields; `go test ./...` including `api/handlers` and `notifications` passes.

## Validation

- Host Go 1.27.0: `go build ./...`, `go vet ./engine ./ai ./db/... ./api/...` clean; `TEST_DB_DSN=<disposable mysql> go test ./engine -count=1 -v` → 34 tests PASS, none skipped.
- Container `golang:1.26-alpine` (go1.26.8, host module cache, `GOPROXY=off`, `-mod=readonly`, disposable MySQL): `go build ./...`, `go vet`, `go test ./... -count=1` → all 13 packages `ok`. (Host run of `notifications` is blocked by Windows Application Control from executing the test binary; the container run passes.)
- `frontend: npm run build` → built; `docs: npm run docs:build` (VitePress) → build complete; `git diff --check` → clean.
- After continuity sync: catalog `scripts/manage_cvf_downstream_catalog.ps1 -Check` → PASS; workspace doctor `check_cvf_workspace_agent_enforcement.ps1 -ProjectPath .` → PASS 25/25.
- Disposable test container `ccma-s1-snapshot-test-20260927` and its anonymous volume were removed; no test rows were written to `CCMA`.

Pre-existing, not changed: `engine/integration_test.go` (build tag `integration`) does not compile because `SmartMockProvider` lacks `AnalyzeChatBatch`; `gofmt -l` flags CRLF-checkout files.

## Claim boundary

No Claude/Gemini/OpenAI/xAI or other provider API was called and no API key was used; all analysis in tests used in-process test doubles, which prove parsing and persistence only — not AI quality, Vietnamese understanding, governance or cost savings. No channel sync, customer data, deploy or push. Snapshot coverage is recorded but does not decide eligibility; candidate selection, typed eligibility/execution/disposition, provider admission, budget, cache reuse and human review remain S2/S3/S5. Pre-existing channel-delete handler still leaves `job_results` behind (only snapshots were added to its cascade).
