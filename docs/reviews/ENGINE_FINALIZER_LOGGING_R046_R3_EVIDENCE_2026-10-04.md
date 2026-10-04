# R046-R3 Evidence: Bounded Worker Evidence Campaign, Mutation Receipts & Publication Verification

Date: 2026-10-04 (Asia/Saigon). Role: Claude REPAIR_WORKER / repair BUILD COMMIT_STEWARD. Risk ceiling: R2.
Status: Repair BUILD complete, **REVIEW_PENDING** for independent Codex re-review. The worker does not set REVIEW_PASS or claim FREEZE.
Order: [R046](../work_orders/CCMAI_RUNTIME_046.md) (R3 Cost-Approved Evidence Repair).
Cost Disposition: [CONTINUE_ONE_EVIDENCE_ONLY_R3](CCMAI_RUNTIME_046_R3_COST_DISPOSITION_2026-10-04.md).
Independent R2 Evaluation: [CCMAI_RUNTIME_046_R2_INDEPENDENT_REREVIEW_2026-10-04.md](CCMAI_RUNTIME_046_R2_INDEPENDENT_REREVIEW_2026-10-04.md).
Machine Receipt: [r046_r3_worker_receipts.json](probes/r046_r3_worker_receipts.json).
Log Extracts: [r046_r3_log_extracts.json](probes/r046_r3_log_extracts.json).
Campaign Runner: `docs/reviews/probes/r046_r3_campaign_runner.ps1` (repository file; publication pointer corrected by independent reviewer on 2026-10-05 after reproducing the original dead-link build failure).

---

## 1. Identity, Scope, and Product Integrity

- **Immutable Dispatcher Seed**: `CVF_SESSION/authority/CCMAI-RUNTIME-046.json` first committed at `c3ff83c7b8d2f9b23ef4113ae3bf7f8b9b32c267` (= `baseCommit`); unchanged.
- **Reference Product Commit**: `91da0e88118b76a68031f432da50521fe6a341b7`.
- **Whole-Backend Byte Equality**: `git diff 91da0e88118b76a68031f432da50521fe6a341b7 -- backend` is empty (0 diff).
  - Production source `backend/engine/analyzer_incremental.go`: `4b7477aa12fd6c232e85818960f6ec0f2947548cd6ac24e714568ebff5a90f0b` (Unchanged).
  - Maintained test `backend/engine/analyzer_finalizer_logging_test.go`: `06ec2c8d15d6199a679f9daaf0835db1baa7a8a63a35b79df5e9465d90f7c449` (Unchanged).
- **Inherited Observations**:
  - Whole-backend `go build ./...` and `go vet ./engine` are inherited as `INHERITED` from independent R2 evaluation at `91da0e88118b76a68031f432da50521fe6a341b7`.
  - Settled findings R046-R2-01 (Statement pool isolation) and R046-R2-02 (complete transaction boundary hooks) are preserved without re-mutating source or tests.
- **R3 Evidence-Only Scope**:
  - `docs/reviews/ENGINE_FINALIZER_LOGGING_R046_R3_EVIDENCE_2026-10-04.md` (this report)
  - `docs/reviews/probes/r046_r3_worker_receipts.json` (machine receipt conforming to contract)
  - `docs/reviews/probes/r046_r3_log_extracts.json` (secret-free raw log extracts)
  - `docs/reviews/probes/r046_r3_campaign_runner.ps1` (disposable campaign runner source)
  - Bounded continuity/catalog metadata.

---

## 2. Execution Environment & Docker Isolation

The R3 campaign was executed in a strictly controlled, network-isolated Docker environment against a disposable copy of the backend source extracted from Git archive `r046-r3-source.tar` (`1a1c07503902e5b3f8568ad19aba48765b9aadd5dfd107b5c4a557ee4abe33bd`):

| Property | Value | Verification |
| --- | --- | --- |
| Go Image | `golang:1.26-alpine` | `sha256:8ac98ca534ac3f51e1f420a1dd2c15e74c75cfa0f23f3ad27eb5d7236c349a0c` |
| MySQL Image | `mysql:8.0` | `sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b` |
| Image Pull Policy | `--pull=never` | Verified: cached local images only, no network fetch |
| Docker Network | `r046r3net` (`--internal`) | Verified: internal loopback only; no external routing |
| Host Ports | None | No `-p` or `-P` bindings; containers communicate via internal DNS |
| Module Cache | Read-only mount | `-v C:\Users\tiennm\go\pkg\mod:/go/pkg/mod:ro` |
| Go Toolchain Env | `GOPROXY=off`, `GOTOOLCHAIN=local`, `CGO_ENABLED=0`, `GOFLAGS=-mod=readonly` | Local toolchain only |
| Anonymous Volumes | Removed on teardown | `docker rm -f -v r046r3mysql` explicitly removes anonymous volumes |

---

## 3. Actual Machine Execution & Counts

All counts and assertion names below are derived directly from the machine-generated `go test -json` event logs in `r046-r3-logs/`. Completed events carrying `Test` were parsed, separating top-level and subtests and excluding package-level records.

### Summary Table

| Run | Selection Pattern | Exit | Seconds | Top Pass | Sub Pass | Top Fail | Sub Fail | Skip | Total Events | Raw Log SHA256 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| **Baseline** | `^(TestFL)` | 0 | 158.857 | 22 | 6 | 0 | 0 | 0 | 28 | `f7ecf840d2f720a28bfbb5e05e5ac43d6d0eb7d95262f5a4b429d63dead79595` |
| **M01 Mutated** | `^(TestFL01TransactionBoundaryUnknownErrorIsContained\|TestFL05TransactionBoundaryNormalizationDetector)` | 1 | 2.536 | 0 | 0 | 2 | 0 | 0 | 2 | `e1b3fcb15d7e048e0fbc01230daba7fa747c861fb08523ff86edcdb97a19cbea` |
| **M01 Restored** | Same as above | 0 | 2.450 | 2 | 0 | 0 | 0 | 0 | 2 | `0ca6873ba8658a300e3dae9a1062305995f46fd806c90c993b4aba56c1ab767b` |
| **M02 Mutated** | `^(TestFL05FallbackLogDetector)` | 1 | 1.936 | 0 | 0 | 1 | 0 | 0 | 1 | `fce6a974470e72d7de0cc75a1d9b81b02195904236ae43314cc9d679d75f22ba` |
| **M02 Restored** | Same as above | 0 | 1.880 | 1 | 0 | 0 | 0 | 0 | 1 | `3e08f4462b7316c9a147ab58af51d9e96807aeb67e63dd65468bcc51185ebf3f` |
| **M03 Mutated** | `^(TestFL05GormSinkDetector)` | 1 | 1.910 | 0 | 0 | 1 | 0 | 0 | 1 | `7cb9b5e59f127dd29b4ea5876d3577a24265bce3ccc234774803f037a6d0ba2e` |
| **M03 Restored** | Same as above | 0 | 1.850 | 1 | 0 | 0 | 0 | 0 | 1 | `4309e598a71f14449f655680190753123875ba933e637ce642c6c9c7ec5a0fea` |
| **Combined** | `^(TestFL\|TestOrdinary\|TestEveryTerminal\|TestTerminalRunHolds\|TestAcceptedCancel\|TestCancelled\|TestReservation\|TestOwnership)` | 0 | 207.358 | 40 | 47 | 0 | 0 | 0 | 87 | `e4280dbcb861a985a9260b2aa0aad9b82eaba0c31684d14527f698a7dd545c9c` |

---

## 4. Mutation Attribution & Detector Kills

Each mutation was applied as an exact single-match text replacement against `backend/engine/analyzer_incremental.go`. Full SHA256 hashes were calculated directly from the file bytes before, during, and after replacement.

### M01: Transaction Boundary Normalization Switch Bypass
- **Target**: Lines 225–234 in `backend/engine/analyzer_incremental.go`
- **Replacement**: `lastErr = txErr` (bypasses containment switch)
- **Match Count**: Exactly 1
- **Baseline SHA256**: `4b7477aa12fd6c232e85818960f6ec0f2947548cd6ac24e714568ebff5a90f0b`
- **Mutated SHA256**: `b212ec5594d86ca7958c499b40cd7269277ccd82e9f8dd152640bdd11274cbcd` (Exact match to expected)
- **Test Selection**: `^(TestFL01TransactionBoundaryUnknownErrorIsContained|TestFL05TransactionBoundaryNormalizationDetector)`
- **Behavioral Outcome**: **`KILLED_NAMED_BEHAVIORAL_ASSERTION`** (Exit code 1).
  - Failed assertions:
    1. `TestFL05TransactionBoundaryNormalizationDetector`:
       ```
       analyzer_finalizer_logging_test.go:1044: FL-05 BOUNDARY DETECTOR FAILED: returned error does not match errFinalizeWrite sentinel (normalization switch missing or bypassed): finalize job run: fl05-detector-boundary-driver-leak-marker
       ```
    2. `TestFL01TransactionBoundaryUnknownErrorIsContained`: Fails because raw boundary driver error escaped containment into the returned error.
- **Restoration**: Restored byte-for-byte to `4b7477aa12fd6c232e85818960f6ec0f2947548cd6ac24e714568ebff5a90f0b`.
- **Restored Test**: Both tests PASS (Exit code 0).

### M02: Raw Fallback Error Log Format Bypass
- **Target**: Line 256 in `backend/engine/analyzer_incremental.go`
- **Replacement**: `log.Printf("[analyzer] fallback error mark failed for run %s: write error: %v", run.ID, err)`
- **Match Count**: Exactly 1
- **Baseline SHA256**: `4b7477aa12fd6c232e85818960f6ec0f2947548cd6ac24e714568ebff5a90f0b`
- **Mutated SHA256**: `a7306bb0ec6a0b551fa0407ccde94a10b30f59f99bc9df185a4b6a93a87375a6` (Exact match to expected)
- **Test Selection**: `^(TestFL05FallbackLogDetector)`
- **Behavioral Outcome**: **`KILLED_NAMED_BEHAVIORAL_ASSERTION`** (Exit code 1).
  - Failed assertion:
    ```
    analyzer_finalizer_logging_test.go:980: FL-05 FALLBACK DETECTOR FAILED: raw driver marker "fl-r046-detector-fallback-marker-05" escaped into fallback log (fix missing or reverted): ... write error: Error 1644 (45000): fl-r046-detector-fallback-marker-05
    ```
- **Restoration**: Restored byte-for-byte to `4b7477aa12fd6c232e85818960f6ec0f2947548cd6ac24e714568ebff5a90f0b`.
- **Restored Test**: PASS (Exit code 0).

### M03: Scoped GORM Session Suppression Bypass
- **Target**: Line 139 in `backend/engine/analyzer_incremental.go`
- **Replacement**: `return db.DB.Session(&gorm.Session{Logger: db.DB.Logger.LogMode(logger.Info)})`
- **Match Count**: Exactly 1
- **Baseline SHA256**: `4b7477aa12fd6c232e85818960f6ec0f2947548cd6ac24e714568ebff5a90f0b`
- **Mutated SHA256**: `426a427bd5dd7d378fcf489cc8128e098b52194cf34bc242206eb7b2fc7bf9d3` (Exact match to expected)
- **Test Selection**: `^(TestFL05GormSinkDetector)`
- **Behavioral Outcome**: **`KILLED_NAMED_BEHAVIORAL_ASSERTION`** (Exit code 1).
  - Failed assertion:
    ```
    analyzer_finalizer_logging_test.go:1008: FL-05 GORM DETECTOR FAILED: raw driver marker "fl-r046-detector-gorm-marker-05" appeared in GORM sink (finalizerDB fix missing or reverted): ... Error 1644 (45000): fl-r046-detector-gorm-marker-05
    ```
- **Restoration**: Restored byte-for-byte to `4b7477aa12fd6c232e85818960f6ec0f2947548cd6ac24e714568ebff5a90f0b`.
- **Restored Test**: PASS (Exit code 0).

---

## 5. Teardown Verification

Following execution of the combined test suite, deterministic resource cleanup and absence verification were completed:

1. **Database Container Removal**:
   - Command: `docker rm -f -v r046r3mysql`
   - Exit code: 0
   - Absence verification: `docker ps -aq --filter name=^r046r3mysql$` returned empty string.
2. **Network Removal**:
   - Command: `docker network rm r046r3net`
   - Exit code: 0
   - Absence verification: `docker network ls -q --filter name=^r046r3net$` returned empty string.
3. **Source Integrity**:
   - File SHA256 of `backend/engine/analyzer_incremental.go`: `4b7477aa12fd6c232e85818960f6ec0f2947548cd6ac24e714568ebff5a90f0b` (Identical to reference commit `91da0e88`).
   - File SHA256 of `backend/engine/analyzer_finalizer_logging_test.go`: `06ec2c8d15d6199a679f9daaf0835db1baa7a8a63a35b79df5e9465d90f7c449` (Identical to reference commit `91da0e88`).

---

## 6. Publication Gates & Verification Checks

| Check | Command | Result / Notes |
| --- | --- | --- |
| **Workspace Doctor** | `powershell -ExecutionPolicy Bypass -File ../.Controlled-Vibe-Framework-CVF/scripts/check_cvf_workspace_agent_enforcement.ps1 -ProjectPath .` | **PASS WITH NOTE** (25 passed, 1 warning on downstream gate profile migration) |
| **Gate Unit Suite** | `python -m unittest discover -s scripts/tests -p "test_cvf_downstream_gate*.py"` | **PASS** (46 tests passed in 32.7s, OK) |
| **Catalog Check (PS 5.1)** | `powershell -ExecutionPolicy Bypass -File scripts/manage_cvf_downstream_catalog.ps1 -Check` | **PASS** (Governed downstream catalog is valid and generated views match source truth) |
| **Catalog Check (PS 7)** | `pwsh -ExecutionPolicy Bypass -File scripts/manage_cvf_downstream_catalog.ps1 -Check` | **PASS** (Governed downstream catalog is valid and generated views match source truth) |
| **Downstream Gate (Preflight)** | `python scripts/cvf_downstream_gate.py preflight` | **PASS** (7/7 gates passed) |
| **Downstream Gate (PR Range)** | `python scripts/cvf_downstream_gate.py preflight --base origin/main --head HEAD` | **PASS** (7/7 gates passed) |
| **Diff Check** | `git diff 91da0e88118b76a68031f432da50521fe6a341b7 -- backend` | **PASS** (Clean diff, 0 backend changes relative to reference commit) |

---

## 7. Historical Attribution Disclosures & Limits

1. **R1 and R2 Historical Limitations**:
   - In R1, transaction-boundary testing only exercised statement-level injection; COMMIT and Rollback hook coverage was not yet complete.
   - In R2, the worker report recorded 20 top-level / 26 total test passes due to ad-hoc event grouping; the exact machine truth derived from Go test JSON events is **22 top-level / 28 total events**.
   - In R2, M01 selected two tests (`TestFL01TransactionBoundaryUnknownErrorIsContained` and `TestFL05TransactionBoundaryNormalizationDetector`), but only the detector failure was listed in the summary table; the machine log proves that both tests failed when normalization was bypassed.
   - The R2 campaign was run using the local test script on Windows host rather than within an explicit `--pull=never` and `--internal` Docker container.
2. **R3 Machine Truth**:
   - The R3 campaign was executed against a disposable source copy in network-isolated Docker containers with `--pull=never`, anonymous volume teardown (`-v`), and full event parsing.
   - All counts (22/28 baseline, 40/87 combined regression) match the exact machine event counts.
   - All three mutations (M01, M02, M03) produced exact hash matches and killed the designated behavioral detectors with explicit error messages.
3. **Boundary and Prohibited Effects**:
   - All tests used synthetic loopback MySQL containers with disposable schemas.
   - No production database, real provider credentials, external networks, or live channel integrations were accessed.
   - Facebook and Zalo OA accounts remain strictly `PARKED`.
   - Claude acts as `REPAIR_WORKER / repair BUILD COMMIT_STEWARD`. Status is **REVIEW_PENDING** for independent Codex review. No self-approval or FREEZE claim is made.
