# REPAIR evidence: CCMAI-RUNTIME-002 Gate B, round 1

**Work order:** `CCMAI-RUNTIME-002` Gate B · **Authority:** `docs/reviews/CCMAI_RUNTIME_002_GATE_B_INDEPENDENT_REVIEW_2026-09-27.md` (`CHANGES_REQUIRED`) · **Ngày:** 2026-09-27 · **Repairer:** Claude (`REPAIR_WORKER`, same objective/path/risk/commit boundary as Gate B, no new authorization) · **Kết quả:** repairs applied, REVIEW_PENDING for Codex re-review.

Same-scope repair per Governance Latency and Approval Continuity: objective, allowed path/artifact class, R2 risk, no external effect, no provider call, local commit only — unchanged from Gate B, so no new escalation was required.

## Findings repaired

### R2-B1 (HIGH) — dangling result / orphan snapshot on delete and prune

**`backend/api/handlers/channels.go` — `DeleteChannel`:** the cascade now runs inside `db.DB.Transaction`, deletes `JobResult` before `AnalysisSnapshot` (previously `JobResult` was never deleted here), and checks every step's error — a mid-cascade failure rolls back instead of leaving a dangling `analysis_snapshot_id` or a channel deleted with orphaned children.

**`backend/cli/prune_results.go` — `ApplyPrunePlan`:** after deleting a batch's stale `job_results`, it now also deletes `analysis_snapshots` matching `(tenant, conversation, stale run)` — but only rows whose `id` is `NOT IN` the surviving `job_results.analysis_snapshot_id` set, so a snapshot is never removed while a result still cites it. Both deletes stay in the same per-batch transaction as before.

Evidence:
- `backend/api/handlers/channels_test.go` (new) — `TestDeleteChannelRemovesResultsAndSnapshotsTogether`: seeds a channel/conversation/message/snapshot/result, calls `DeleteChannel` through a `gin.CreateTestContext`, asserts zero `job_results`, zero `analysis_snapshots`, and the channel itself gone.
- `backend/cli/prune_results_test.go` (new) — `TestApplyPrunePlanCleansOrphanSnapshotsKeepsReferenced`: three snapshots — one becomes orphaned by the prune and must be deleted, one belongs to the retained run and must survive untouched, one belongs to the stale run but is still cited by a surviving result and must survive (proves the `NOT IN` guard, not just conversation-scoping).

### R2-B2 (MEDIUM) — digest blind to attachment identity

**`backend/engine/snapshot.go`:** `classifyAttachments` now returns a third value, `fingerprint`. For valid attachment JSON it is a SHA-256 of the canonical typed fields (`type`, `url`, `name`, `local_path`) joined with unit/record separators — never the raw payload. For invalid JSON it is a SHA-256 of the raw bytes, so a source change still moves the digest even when coverage stays `ATTACHMENT_JSON_INVALID`. `snapshotMessage` gained `AttachmentFingerprint string \`json:"attachment_fingerprint,omitempty"\`` in the digest-bearing manifest.

Evidence: `backend/engine/snapshot_test.go` (new) — `TestSnapshotDigestChangesWithAttachmentIdentity`: same count/coverage, different `url`/`name`/`type`/`local_path` → digest changes in all four cases; identical valid JSON → identical digest; two different invalid-JSON payloads → different digests.

### R2-B3 (LOW) — DB test fallback DSN hardcoded to CQA

**`backend/engine/snapshot_db_test.go` — `connectTestDB`:** removed the `cqa:cqa_password@.../cqa` fallback. The function now calls `t.Skip` with a message when `TEST_DB_DSN` is unset, instead of silently defaulting to a schema/credential pair from the retired CQA identity. Scope: only the file the Gate B review named (`snapshot_db_test.go`); other pre-existing DB test files with the same historical pattern (`prune_results_test.go`, `reanalyze_test.go`) are outside this finding and this repair round.

## Verification

- `go build ./...`, `go vet ./engine ./api/... ./cli/...` — clean.
- Disposable `mysql:8.0` container (`ccma-runtime002-repair-test`, host port 33061, removed after the run; persistent Compose `ccma` stack untouched throughout):
  - `go test ./engine -run TestSingleAndBatchShareSnapshotContract\|TestInvalidEvidenceRejectsWholeConversation\|TestBatchRejectsCrossConversationRef\|TestSaveResultsRollsBackSnapshotOnFailure\|TestLegacyResultsAreMarkedUnverified` — all 5 pre-existing Gate B DB tests still PASS, unchanged behavior.
  - `go test ./api/handlers -run TestDeleteChannelRemovesResultsAndSnapshotsTogether` — PASS.
  - `go test ./cli -run TestApplyPrunePlanCleansOrphanSnapshotsKeepsReferenced\|TestDonBanDanhGiaTrungGiuLuotMoiNhat\|TestDonKhongDungToiTinNhanVaCuocChat` — PASS (new test plus both pre-existing prune tests unaffected).
  - `go test ./... -count=1` — all 13 packages `ok`.
- `go test ./engine -run TestSnapshot\|TestEvidenceRefs -v` (no DB) — 9 tests PASS including the new `TestSnapshotDigestChangesWithAttachmentIdentity`.
- `gofmt -l` on every touched/new file, CRLF-normalized before comparison: all clean except `channels.go`, whose only remaining diff is three pre-existing, unrelated alignment spots (`CreateChannel`'s `Metadata` literal, `zaloTokenResponse.Error` comment, `Data.OAID`/`Name` struct alignment) — none inside the lines this repair changed; confirmed via `git diff` that none of the added lines match those spots.
- Workspace doctor `check_cvf_workspace_agent_enforcement.ps1 -ProjectPath .` — PASS 25/25, including the governed downstream catalog `--check`.
- No frontend or docs files were touched this round, so `npm run build` / `docs:build` were not re-run.

## Claim boundary

No Claude/Gemini/OpenAI/xAI or other provider API was called and no API key was used. No channel sync, customer data, deploy or push. This round only closes the three cited findings — it does not reopen or re-certify any other part of Gate B, and it does not authorize S2/S3/S5, FREEZE, or any AI-runtime-governance claim. Local commit only; Codex re-reviews the changed set and this evidence next.
