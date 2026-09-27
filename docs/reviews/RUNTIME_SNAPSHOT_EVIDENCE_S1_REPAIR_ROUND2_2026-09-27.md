# REPAIR evidence: CCMAI-RUNTIME-002 Gate B, round 2

**Work order:** `CCMAI-RUNTIME-002` Gate B · **Authority:** `docs/reviews/CCMAI_RUNTIME_002_GATE_B_REREVIEW_2026-09-27.md` (`CHANGES_REQUIRED_ROUND_2`, target commit `f924a6b`) · **Ngày:** 2026-09-27 · **Repairer:** Claude (`REPAIR_WORKER`, same objective/path/risk/commit boundary, no new authorization) · **Kết quả:** repairs applied, REVIEW_PENDING for Codex re-review.

Round 1 closed R2-B1 (prune orphan cleanup), R2-B2 (attachment fingerprint present) and R2-B3 (DSN fallback removed). This round closes the two findings Codex's re-review raised against round 1's own fix.

## Findings repaired

### R2-RR1 (HIGH) — channel cascade not fully inside the transaction

Round 1 still read `convIDs` with a plain `Pluck` **before** `db.DB.Transaction` and ignored its error, and ran `os.RemoveAll` for attachment files **inside** the transaction before commit. A conversation created between the `Pluck` and the transaction's `DELETE conversations WHERE channel_id=...` would have its conversation row deleted by the channel-scoped delete, but its messages/results/snapshot would not be in the stale `convIDs` list and would be orphaned. Deleting files before commit also meant a DB rollback could not undo the already-deleted files.

**`backend/api/handlers/channels.go` — `DeleteChannel`:**
1. The conversation read moved inside `db.DB.Transaction`, using `tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("channel_id = ? AND tenant_id = ?", ...).Find(&conversations)` — a locking (`FOR UPDATE`) read on the same predicate the final `DELETE conversations` uses. Under InnoDB REPEATABLE READ this takes gap/next-key locks on the scanned range, so a conversation cannot be inserted for this channel mid-cascade and slip past the ID list used to delete its children. The read's error is now checked and aborts the transaction.
2. `os.RemoveAll` for attachment directories now runs **after** the transaction commits, using the `convIDs` captured from the locked read; a file-deletion failure is logged as a warning, not treated as a DB rollback (a DB rollback cannot un-delete a file, so deleting files first would have meant real data loss surviving a rolled-back cascade).

Evidence: `backend/api/handlers/channels_test.go` (new) — `TestDeleteChannelFailureRollsBackWholeCascade`: seeds a channel with two `job_results` sharing one snapshot, installs a MySQL `BEFORE DELETE` trigger on `job_results` that raises a SQL error for one sentinel row, calls `DeleteChannel`, asserts it returns 500, and that the channel, conversation, message, **both** `job_results` and the snapshot are all still present — nothing was partially applied. `TestDeleteChannelRemovesResultsAndSnapshotsTogether` (round 1's happy-path test) continues to pass unchanged.

Test-environment note: the disposable MySQL container needed `SET GLOBAL log_bin_trust_function_creators = 1` (run once, as root, outside any test) before the `ccma` test user could `CREATE TRIGGER` with binary logging on. This is a container/test-fixture prerequisite for the failure-injection technique, not a product or migration requirement.

### R2-RR2 (MEDIUM) — attachment fingerprint delimiter collision

Round 1's fingerprint joined typed attachment fields with raw `\x1f`/`\x1e` bytes without escaping. The re-review reproduced a real collision: `{type: "a\u001fb", url: "c"}` and `{type: "a", url: "b\u001fc"}` serialize to the identical joined byte string and therefore the identical SHA-256.

**`backend/engine/snapshot.go` — `classifyAttachments`:** now re-encodes the typed `[]channels.Attachment` slice with `json.Marshal` and hashes those bytes instead of hand-joining fields. Fixed struct field order plus JSON's own escaping of quotes/colons/commas/control characters inside string values makes the encoding unambiguous — the same delimiter-collision pair now produces different digests, because the encoder itself escapes any embedded `\u001f`/`\u001e` characters rather than leaving them as raw joinable bytes. The invalid-JSON path now hashes the **untrimmed** `raw` string instead of `trimmed`, so a whitespace-only source change also moves the digest, per the re-review's literal reading of the "raw bytes" claim.

Evidence: `backend/engine/snapshot_test.go` — `TestSnapshotDigestChangesWithAttachmentIdentity` gained three cases:
- reproduces the exact `type/url` delimiter-collision pair from the re-review and asserts the digests now differ;
- same typed attachment values under different JSON whitespace/key order still produce the *same* digest (per repair acceptance #3);
- two invalid-JSON payloads differing only in leading whitespace now produce different digests (per repair acceptance #2, "no trim before hash").

## Verification

- `go build ./...`, `go vet ./engine ./api/...` — clean.
- Disposable `mysql:8.0` container (`ccma-runtime002-repair2-test`, host port 33062, removed after the run; persistent Compose `ccma` stack untouched throughout):
  - `go test ./api/handlers -run TestDeleteChannel` — both `TestDeleteChannelRemovesResultsAndSnapshotsTogether` (round 1) and the new `TestDeleteChannelFailureRollsBackWholeCascade` PASS.
  - `go test ./... -count=1` — all 13 packages `ok`.
- `go test ./engine -run TestSnapshot\|TestEvidenceRefs -v` (no DB) — all 8 tests PASS, including the three new `TestSnapshotDigestChangesWithAttachmentIdentity` cases.
- `gofmt -l` on every file touched this round, CRLF-normalized before comparison: `channels_test.go` and `snapshot.go`/`snapshot_test.go` clean; `channels.go`'s only remaining diff is the same three pre-existing, unrelated spots identified in round 1 (`CreateChannel`'s `Metadata` literal, `zaloTokenResponse.Error` comment alignment, `Data.OAID`/`Name` struct alignment) — confirmed via `git diff -U0` that none of this round's added lines touch those spots.
- Workspace doctor `check_cvf_workspace_agent_enforcement.ps1 -ProjectPath .` — PASS 25/25.
- No frontend or docs files were touched this round.

## Claim boundary

No Claude/Gemini/OpenAI/xAI or other provider API was called and no API key was used. No channel sync, customer data, deploy or push. This round only closes R2-RR1 and R2-RR2 — it does not reopen, re-certify or re-scope any other part of Gate B, and it does not authorize S2/S3/S5, FREEZE, or any AI-runtime-governance claim. Local commit only; Codex re-reviews the changed set and this evidence next.
