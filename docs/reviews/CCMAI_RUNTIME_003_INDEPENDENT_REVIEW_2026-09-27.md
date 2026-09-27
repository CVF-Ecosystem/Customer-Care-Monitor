# Independent REVIEW: CCMAI-RUNTIME-003 BUILD

**Reviewer:** Codex (`REVIEWER`) · **Date:** 2026-09-27 · **Target:** local commit `2a4e530` · **Disposition:** `CHANGES_REQUIRED`. This is the first R2 review round for this tranche; same-scope repair may proceed under the existing work order. No FREEZE.

## Accepted portions

- Same-external-ID replay now preserves internal row identity, updates changed nonempty content/name/time, and moves the shared snapshot digest. The central regression fails on pre-change source as documented.
- Attachment merge retains a local path only for the same `(type, URL, name)` identity, does not transfer one across a changed identity, and does not erase a stored list from an empty incoming list. Identical replay issues no UPDATE in the tested case.
- Adapter capability audit names concrete source locations and leaves removal, missing history and fetch-window limits open. No real channel/provider call or governance-runtime claim was made.
- Independent disposable MySQL 8 `CCMA` verification: `go test ./engine -run '^TestUpsertMessage' -count=1` PASS; `go test ./... -count=1` PASS across all 13 tested packages. `GOFLAGS=-mod=readonly`; database container and dedicated Go-cache volumes removed afterward. Workspace doctor PASS 25/25. These passing tests do not resolve the findings below.

## Blocking findings

### R003-R1 — replay ignores supplied raw-data changes and serialization errors

`docs/specs/RUNTIME_REPLAY_MESSAGE_INTEGRITY_S1_2026-09-27.md` contract 2 requires incoming raw data to be serialized with error checks before a write; contract 3 requires any JSON error to propagate to the honest `partial` path. The create branch of `upsertMessage` marshals `msg.RawData`, but `updateExistingMessage` never reads `msg.RawData` or `existing.RawData` (`backend/engine/sync.go`, current lines 375-415). Therefore a changed source raw payload remains stale in storage, including a newly observed `is_removed` marker carried by Pancake's mapper, and an unmarshalable replay raw payload can return success without a detectable error. This is a source-level counterexample to the SPEC, even though the existing five tests pass.

**Repair acceptance:** Treat a nonempty incoming raw-data map as supplied. Marshal it before any mutation, return marshal errors, compare canonical JSON with the stored value to avoid an UPDATE on unchanged replay, and persist a changed valid payload with the same row update. Preserve existing raw data when the adapter omits it; do not infer message deletion or act on `is_removed` in this slice. Add MySQL tests for changed raw payload, unchanged no-op, and unmarshalable payload returning an error with the stored row unchanged. Record the exact absence/empty-map rule in evidence. A source value such as `math.NaN()` may be used only in a synthetic test to force marshal failure.

### R003-R2 — sender-role acceptance lacks an executable assertion

The SPEC acceptance requires a replay with changed sender **role/name**; `TestUpsertMessageReplayUpdatesStaleContentAndDigest` changes `SenderName` but supplies `SenderType: "customer"` both times (`backend/engine/sync_replay_test.go`, current lines 117-145). The production branch appears to update a changed nonempty `sender_type`, but the mandatory test does not exercise or assert it.

**Repair acceptance:** Change the replay test or add a focused case that supplies a valid role transition, asserts the stored `sender_type`, preserves the internal row ID/count, and checks the snapshot digest changes. Keep the Vietnamese/emoji, timestamp and sender-name assertions. Exercise changed content type separately if it can be done within the same small fixture, since the SPEC also names that source field; do not broaden adapter behavior.

## Repair boundary and next move

Both findings are within the current R2 objective, allowed source/test paths, risk and external-effect class; this is the first repair round, so no review-cost escalation is needed. Claude returns as `REPAIR_WORKER` after rehydration and handoff acknowledgment, edits only `backend/engine/sync.go` and `backend/engine/sync_replay_test.go` plus evidence/continuity, reruns focused and full disposable-MySQL checks, and makes one local commit without push. Codex then re-reviews independently. Exact addendum is in `docs/work_orders/CCMAI_RUNTIME_003.md`.

No provider call, real channel sync, customer data, persistent database reset, deployment, S2/S3/S5 implementation or FREEZE is authorized. Existing unrelated `.gitignore` and `docs/references/` worktree entries are outside this review.
