# CCMAI-RUNTIME-015 repair R1/R2 BUILD evidence

**Date:** 2026-09-29

**Disposition:** REVIEW_PENDING for independent Claude re-review; no self-PASS or FREEZE.

**Authority:** [R015 work order repair addendum](../work_orders/CCMAI_RUNTIME_015.md) and [independent findings](CCMAI_RUNTIME_015_INDEPENDENT_REVIEW_2026-09-29.md).
**Risk:** R2. No live provider/channel call or CVF governance claim.

## Route and change

At INTAKE, the active handoff header still said `REVIEW_PENDING` while the state, memory, status and appended handoff review said `CHANGES_REQUIRED`. Codex aligned that stale header, rehydrated, and declared `REVIEWER (Claude) → WORK_ORDER_AUTHOR → REPAIR_WORKER (Codex)` before source work. The owner-authorized repair keeps the original R015 paths and external-effect limits.

- R015-R1: a failed `store.Exists` probe now means that old-object reuse is unconfirmed. The sync downloads to a new attempt key through the existing primary-to-local `Put` fallback. A storage probe is no longer returned as `ErrSyncWriteFailed`, which is reserved for failed owned DB writes. If the transfer itself fails, existing per-attachment failure handling may yield partial status.
- R015-R2: a deterministic per-engine seam runs after a completed conversation. The full `SyncReservedChannel` test takes ownership from A after its first conversation, admits B and writes B's conversation/message/count, then proves A returns `sync_ownership_lost` before writing its second conversation. B's row remains `syncing` with its run ID and count unchanged.

The store seam is scoped to the engine instance used by the test. The synthetic store fails both `Exists` and primary `Put`; a local fallback persists the new object. The test checks the new `local_path`, local object existence, later message and successful final status. The tests use a disposable MySQL and synthetic adapter/HTTP media; they do not contact OAuth, a channel, S3, or an AI provider.

## Gates

| Check | Result |
| --- | --- |
| Focused R015-R1/R2 `./engine` tests on disposable MySQL | PASS (2 tests) |
| Full backend suite on disposable MySQL | PASS (all 13 tested packages; exit 0) |
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| Catalog `-Check` | PASS |
| Workspace doctor | PASS 25/25 |
| `git diff --check` | PASS |

Both test runner invocations removed their temporary MySQL containers and networks. The running `ccma` Compose stack was not touched. No push or FREEZE. O1 (same-millisecond count zero-row), O2 (`JSON_SEARCH` wildcard orphan), and O3 (single-use Zalo token rotation) remain independent-review observations; this repair does not claim to resolve them.
