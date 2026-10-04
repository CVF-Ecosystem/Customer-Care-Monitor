# R046-R1 Repair Evidence: Maintained Transaction-Boundary Tests & Mutation Receipts

Date: 2026-10-04 (Asia/Saigon). Role: Claude REPAIR_WORKER / repair BUILD COMMIT_STEWARD. Risk ceiling R2.
Status: Repair BUILD complete, **REVIEW_PENDING** for independent Codex re-review. The worker does not set REVIEW_PASS or FREEZE.
Order: [R046](../work_orders/CCMAI_RUNTIME_046.md) (Repair Order R1). Codex independent review: [CCMAI_RUNTIME_046_INDEPENDENT_REVIEW_2026-10-04.md](CCMAI_RUNTIME_046_INDEPENDENT_REVIEW_2026-10-04.md).

## 1. Identity, Scope, and Production Source Preservation

- **Seed**: `CVF_SESSION/authority/CCMAI-RUNTIME-046.json` first committed at `c3ff83c7b8d2f9b23ef4113ae3bf7f8b9b32c267` (= baseCommit); unchanged.
- **Repair BUILD Acknowledgment**: committed at `1d08c94` (`docs: acknowledge R046 repair round 1 worker role and open BUILD phase`).
- **Production Source Status**: Byte-for-byte identical to original BUILD commit `985fa60b3766444a7b71a5303da6c6bc14e63bf0`. No production defect was established by Codex review or worker repair.
  - `backend/engine/analyzer_incremental.go`: `4b7477aa12fd6c232e85818960f6ec0f2947548cd6ac24e714568ebff5a90f0b` (Unchanged, 0 git diff)
- **Authorized Test & Evidence Changes**:
  - `backend/engine/analyzer_finalizer_logging_test.go`: `e56f7390fecbbf8b236320c225e5654aa47b3a7d351d7fc1e4bd65631dc6aba4`
  - `docs/reviews/ENGINE_FINALIZER_LOGGING_R046_R1_REPAIR_2026-10-04.md` (this file)
  - `docs/reviews/probes/r046_r1_worker_receipts.json` (machine receipts)

---

## 2. R046-R1-01: Maintained Transaction-Boundary Test Suite

In response to Codex finding R046-R1-01 (detector gap where transaction boundary raw errors bypassed existing tests), 5 new maintained test components were implemented in `backend/engine/analyzer_finalizer_logging_test.go`:

1. **Transaction Boundary Test Infrastructure**:
   - `flBoundaryPool`: custom `gorm.ConnPool` / `gorm.TxBeginner` capturing `BeginTx`, `Commit`, `Rollback` calls with thread-safe counters and simulated raw driver errors.
   - `flBoundaryError`: adversarial error wrapper tracking calls to `.Error()` via an atomic formatting counter (`formats int64`), ensuring raw driver error formatting can be observed.
   - `installBoundaryPool(t, pool)`: cleanly installs mock transaction boundary pool onto newly created `Session(&gorm.Session{NewDB: true})` without violating `gorm.Statement` copy constraints.

2. **Maintained Test Specifications & Invariants**:

| Test Name | Boundary Condition | Verified Invariant | Result |
| --- | --- | --- | --- |
| `TestFL01TransactionBoundaryUnknownErrorIsContained` | Synthetic raw driver error on `BeginTx` with observable `.Error()` counter | Error `.Error()` is called 0 times (`formats == 0`); raw error string is not leaked into log buffer or returned error wrapper; returned error matches `errFinalizeWrite`; fallback error mark succeeds | PASS (0.64s) |
| `TestFL03TransactionBoundaryTransientRecovery` | Transient failure (attempt 1 fails `BeginTx`, attempt 2 succeeds) | Finalizer retries transparently on attempt 2; transaction commits; proposed checkpoint successfully advances to `checkpoint+1` | PASS (0.68s) |
| `TestFL03TransactionBoundaryExhaustedDoesNotAdvanceCheckpoint` | Persistent failure on all `BeginTx` attempts | Retries exhaust up to limit (3 attempts); terminal error recorded; non-nil proposed checkpoint is **NOT** advanced | PASS (0.76s) |
| `TestFL05TransactionBoundaryNormalizationDetector` | Normalization switch sensitivity detector | Injects raw boundary error marker; asserts that bypassing normalization switch fails the test immediately | PASS (0.61s) |

---

## 3. R046-R1-02: Worker Mutation Campaign & Machine Receipts

As required by R046-R1-02, a full mutation campaign was executed against the repository under unchanged source/test paths. Complete machine-readable receipts are stored in [r046_r1_worker_receipts.json](probes/r046_r1_worker_receipts.json).

### Mutation Campaign Summary

| Mutant | Target & Applied Replacement | Mutated SHA256 | Test Outcome | Killing Test Assertions | Restored Baseline |
| --- | --- | --- | --- | --- | --- |
| **M01: Normalization Bypass** | `switch { ... default: lastErr = errFinalizeWrite }` replaced with `lastErr = txErr` | `b212ec5594d86ca7958c499b40cd7269277ccd82e9f8dd152640bdd11274cbcd` | **KILLED** (Exit 1; 21 PASS, 2 FAIL) | `TestFL01TransactionBoundaryUnknownErrorIsContained` ("raw boundary error leaked into returned error wrapper") & `TestFL05TransactionBoundaryNormalizationDetector` ("raw driver marker escaped") | PASS (Exit 0; 23 PASS / 18 top-level, 6.104s) |
| **M02: Raw Fallback Log** | `log.Printf("[analyzer] fallback error mark failed for run %s: write error", run.ID)` replaced with `...: write error: %v, err` | `558a3de54ba61312937279a901303f94975c9336e329e201cdb23e5a5c3db8ea` | **KILLED** (Exit 1; 0 PASS, 1 FAIL) | `TestFL05FallbackLogDetector` ("raw driver marker escaped into fallback log") | PASS (Exit 0; 1 PASS, 0.755s) |
| **M03: GORM Session Bypass** | `LogMode(logger.Silent)` replaced with `LogMode(logger.Info)` | `426a427bd5dd7d378fcf489cc8128e098b52194cf34bc242206eb7b2fc7bf9d3` | **KILLED** (Exit 1; 0 PASS, 1 FAIL) | `TestFL05GormSinkDetector` ("raw driver marker appeared in GORM sink") | PASS (Exit 0; 1 PASS, 0.777s) |

All mutants were byte-restored back to baseline SHA `4b7477aa12fd6c232e85818960f6ec0f2947548cd6ac24e714568ebff5a90f0b`.

---

## 4. Test & Verification Matrix

- `go test ./engine -count=1 -p 1 -json -timeout=180s -run ^TestFL`: 18 top-level tests (23 total events) PASS, 0 fail, 0 skip (6.379s).
- `go test ./engine -count=1 -p 1 -json -timeout=180s -run 'TestFL|TestOrdinary|TestOwnership'`: 31 top-level tests (60 total events) PASS, 0 fail, 0 skip (26.470s).
- `go vet ./engine`: PASS (0 errors, clean statement and pool installation).
- `go build ./...`: PASS (0 errors).

---

## 5. Limits, Boundary, and Claim Protections

- All tests executed against internal synthetic MySQL container fixtures on isolated loopback network.
- NOT RUN: race detector (CGO disabled on this Windows host).
- No production database, live channel, or real AI provider network calls were made.
- Facebook and Zalo OA accounts remain strictly `PARKED`.
- Claude acts exclusively as `REPAIR_WORKER / repair BUILD COMMIT_STEWARD`. This evidence packet hands work back to Codex as independent `REVIEWER` with status `REVIEW_PENDING`. No claim of `REVIEW_PASS` or `FREEZE` is made.
