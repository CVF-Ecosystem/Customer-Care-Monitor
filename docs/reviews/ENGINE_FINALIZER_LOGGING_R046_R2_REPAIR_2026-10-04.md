# R046-R2 Repair Evidence: Statement Isolation, Complete Transaction Boundary Hooks & Mutation Receipts

Date: 2026-10-04 (Asia/Saigon). Role: Claude REPAIR_WORKER / repair BUILD COMMIT_STEWARD. Risk ceiling R2.
Status: Repair BUILD complete, **REVIEW_PENDING** for independent Codex re-review. The worker does not set REVIEW_PASS or FREEZE.
Order: [R046](../work_orders/CCMAI_RUNTIME_046.md) (Repair Order R2). Codex independent re-review: [CCMAI_RUNTIME_046_R1_INDEPENDENT_REREVIEW_2026-10-04.md](CCMAI_RUNTIME_046_R1_INDEPENDENT_REREVIEW_2026-10-04.md).

---

## 1. Identity, Scope, and Production Source Preservation

- **Seed**: `CVF_SESSION/authority/CCMAI-RUNTIME-046.json` first committed at `c3ff83c7b8d2f9b23ef4113ae3bf7f8b9b32c267` (= baseCommit); unchanged.
- **Repair BUILD Acknowledgment (Round 2)**: committed at `4130c75` (`docs: acknowledge R046 repair round 2 worker role and open BUILD phase`).
- **Production Source Status**: Byte-for-byte identical to baseline SHA `4b7477aa12fd6c232e85818960f6ec0f2947548cd6ac24e714568ebff5a90f0b`. No production defect was established by Codex re-review or worker repair.
  - `backend/engine/analyzer_incremental.go`: `4b7477aa12fd6c232e85818960f6ec0f2947548cd6ac24e714568ebff5a90f0b` (Unchanged, 0 git diff).
- **Authorized Test & Evidence Changes**:
  - `backend/engine/analyzer_finalizer_logging_test.go`: `06ec2c8d15d6199a679f9daaf0835db1baa7a8a63a35b79df5e9465d90f7c449`
  - `docs/reviews/ENGINE_FINALIZER_LOGGING_R046_R2_REPAIR_2026-10-04.md` (this file)
  - `docs/reviews/probes/r046_r2_worker_receipts.json` (machine receipts)

---

## 2. R046-R2-01: Statement Pool Isolation & Cleanup Restoration

In response to Codex finding R046-R2-01 (where `installBoundaryPool` mutated `originalDB.Statement.ConnPool` due to GORM Statement sharing in `Session(&gorm.Session{NewDB: true})`):

1. **Statement Cloning Mechanism**:
   - In GORM `v1.31.1`, `db.Session(&gorm.Session{NewDB: true})` reuses the parent `Statement` pointer (`tx.Statement = db.Statement`).
   - By specifying a non-nil `Context` (`Session(&gorm.Session{Context: context.Background()})`), GORM triggers `tx.Statement = tx.Statement.clone()`, ensuring the scoped session operates on its own Statement copy.
   - `t.Cleanup` restores both the global DB pointer `db.DB = originalDB` and explicitly re-pins `originalDB.Statement.ConnPool = originalPool`.

2. **Maintained Isolation Verification**:
   - Adapted reviewer probe into the maintained test suite: `TestFLBoundaryPoolRestoresOriginalStatement`.
   - Asserts that installing the pool does not mutate `originalStatement.ConnPool` before cleanup, and that post-cleanup both `db.DB` and `originalDB.Statement.ConnPool` are restored.
   - Result: PASS (0.66s, subtest `install_and_cleanup` PASS 0.00s).

---

## 3. R046-R2-02: Complete Transaction Boundary Hooks (Commit & Rollback)

In response to Codex finding R046-R2-02 (the mock pool only captured `BeginTx` and used plain `int` counter without Commit/Rollback error handling):

1. **Atomic Formatting Counter**:
   - `flBoundaryError` was updated with `formatted *int64` and `atomic.AddInt64(e.formatted, 1)` to eliminate data race potential and provide precise thread-safe formatting detection.

2. **Transaction Hook Struct (`flBoundaryTx`)**:
   - Implements `gorm.Tx` (`*sql.Tx`, `Commit() error`, `Rollback() error`).
   - Delegates execution to the underlying database transaction while intercepting `Commit` and `Rollback` calls up to configured limits (`failCommitRemaining`, `failRollbackRemaining`).
   - Handles standard database/sql semantics: ignores expected `sql.ErrTxDone` on post-error rollback calls so GORM's deferred cleanup does not format or chain spurious errors.

3. **Complete Boundary Test Coverage**:

| Test Name | Boundary Condition | Verified Invariant | Result |
| --- | --- | --- | --- |
| `TestFLBoundaryPoolRestoresOriginalStatement` | Statement isolation & cleanup | Original `Statement.ConnPool` never mutated; DB pointer & pool fully restored | PASS (0.66s) |
| `TestFL01TransactionBoundaryUnknownErrorIsContained` | `BeginTx` fault (all attempts fail) | Formatted 0 times; bounded `errFinalizeWrite` returned; fallback status="error" recorded | PASS (0.67s) |
| `TestFL01TransactionBoundaryCommitFaultIsContained` | `Commit` fault (all attempts fail) | Formatted 0 times; no leak to returned error or app log; fallback status="error" recorded | PASS (0.70s) |
| `TestFL01TransactionBoundaryRollbackFaultIsContained` | Internal failure triggers `Rollback` fault | Formatted 0 times; `errFinalizeMissing` preserved; no leak into returned error or app log | PASS (0.63s) |
| `TestFL03TransactionBoundaryTransientRecovery` | Transient `BeginTx` fault (attempt 1 fails, attempt 2 succeeds) | Transparent retry; transaction commits; proposed checkpoint advances | PASS (0.71s) |
| `TestFL03TransactionBoundaryCommitTransientRecovery` | Transient `Commit` fault (attempt 1 fails, attempt 2 succeeds) | Transparent retry; attempt 2 commits successfully; checkpoint advances | PASS (0.70s) |
| `TestFL03TransactionBoundaryExhaustedDoesNotAdvanceCheckpoint` | Persistent `BeginTx` fault across all attempts | 3 failed attempts observed; non-nil proposed checkpoint demonstrably does NOT advance | PASS (0.73s) |
| `TestFL05TransactionBoundaryNormalizationDetector` | Normalization switch sensitivity detector | Bypassing normalization fails test immediately with explicit message | PASS (0.68s) |

---

## 4. R046-R2-03: Separately Labeled Worker Mutation Campaign R2

A fresh mutation campaign R2 was executed on the live repository. Machine-readable receipts are stored in [r046_r2_worker_receipts.json](probes/r046_r2_worker_receipts.json).

### Mutation Campaign Summary

| Mutant | Target & Applied Replacement | Mutated SHA256 | Test Outcome | Killing Test Assertions | Restored Baseline |
| --- | --- | --- | --- | --- | --- |
| **M01: Normalization Bypass** | Target lines 225-234 replaced with `lastErr = txErr` | `b212ec5594d86ca7958c499b40cd7269277ccd82e9f8dd152640bdd11274cbcd` | **KILLED** (Exit 1) | `analyzer_finalizer_logging_test.go:1044: FL-05 BOUNDARY DETECTOR FAILED: returned error does not match errFinalizeWrite sentinel (normalization switch missing or bypassed): finalize job run: fl05-detector-boundary-driver-leak-marker` | Byte-restored to `4b7477aa...` (Exit 0) |
| **M02: Raw Fallback Log** | Target line 256 replaced with `log.Printf("... write error: %v", run.ID, err)` | `a7306bb0ec6a0b551fa0407ccde94a10b30f59f99bc9df185a4b6a93a87375a6` | **KILLED** (Exit 1) | `analyzer_finalizer_logging_test.go:980: FL-05 FALLBACK DETECTOR FAILED: raw driver marker "fl-r046-detector-fallback-marker-05" escaped into fallback log (fix missing or reverted): ... write error: Error 1644 (45000): fl-r046-detector-fallback-marker-05` | Byte-restored to `4b7477aa...` (Exit 0) |
| **M03: GORM Session Bypass** | Target line 139 replaced with `Logger.LogMode(logger.Info)` | `426a427bd5dd7d378fcf489cc8128e098b52194cf34bc242206eb7b2fc7bf9d3` | **KILLED** (Exit 1) | `analyzer_finalizer_logging_test.go:1008: FL-05 GORM DETECTOR FAILED: raw driver marker "fl-r046-detector-gorm-marker-05" appeared in GORM sink (finalizerDB fix missing or reverted): ... Error 1644 (45000): fl-r046-detector-gorm-marker-05` | Byte-restored to `4b7477aa...` (Exit 0) |

*Note on M02 SHA256*: In R1 review, Codex noted that replacing line 256 yields `a7306bb0ec6a0b551fa0407ccde94a10b30f59f99bc9df185a4b6a93a87375a6`. The R2 campaign verifies and documents this exact hash on disk.

---

## 5. Verification Matrix & Regression Suite

- **Targeted Finalizer Suite** (`powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./engine -Run 'TestFL' -VerboseTests`):
  - 20 top-level tests, 26 total test/subtest runs: **ALL PASS** (0 failures, 0 skips, 17.033s).
- **Comprehensive Regression Selection** (`-Run 'TestFL|TestOrdinary|TestEveryTerminal|TestTerminalRunHolds|TestAcceptedCancel|TestCancelled|TestReservation|TestOwnership'`):
  - 34 top-level tests, 64 total test/subtest runs: **ALL PASS** (0 failures, 0 skips, 68.733s).
- **Code Quality & Static Analysis**:
  - `go vet ./engine`: PASS (0 errors).
  - `go build ./...`: PASS (0 errors).

---

## 6. Limits, Boundary, and Claim Protections

- All tests executed against internal synthetic MySQL container fixtures on isolated loopback Docker network.
- NOT RUN: race detector (CGO disabled on this Windows host).
- No production database, live channel, or real AI provider network calls were made.
- Facebook and Zalo OA accounts remain strictly `PARKED`.
- Claude acts exclusively as `REPAIR_WORKER / repair BUILD COMMIT_STEWARD`. This evidence packet hands work back to Codex as independent `REVIEWER` with status `REVIEW_PENDING`. No claim of `REVIEW_PASS` or `FREEZE` is made.
