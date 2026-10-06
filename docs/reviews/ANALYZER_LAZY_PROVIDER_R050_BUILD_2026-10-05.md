# R050 BUILD record — source-first application provider initialization

Date: 2026-10-05. Status: BUILT, REVIEW_PENDING (not accepted). Worker: Claude (`IMPLEMENTATION_WORKER` / `BUILD COMMIT_STEWARD`). Risk R2.
Work order: [CCMAI_RUNTIME_050](../work_orders/CCMAI_RUNTIME_050.md), SPEC: [ANALYZER_LAZY_PROVIDER_R050_2026-10-05](../specs/ANALYZER_LAZY_PROVIDER_R050_2026-10-05.md).

## 1. Commit identity, authority and source hashes

- **Authority seed**: `CVF_SESSION/authority/CCMAI-RUNTIME-050.json` (committed at `1008ab41f0693e2814cd06dfdb0fed98273a1167`, unchanged).
- **Tranche baseCommit**: `1008ab41f0693e2814cd06dfdb0fed98273a1167`.
- **Role acknowledgment commit**: `7606c857738f6b8b7d9036c0d01ee34316d3e3a1` (`docs: acknowledge R050 BUILD worker role and open BUILD phase`).
- **Exact BUILD commit**: `727d3229338e9b29c749612677a08b7fd1c65428` (`feat(engine): source-first application provider initialization (R050)`).
- **Changed file set** (3 files, 720 insertions(+), 19 deletions(-)):
  - `backend/engine/analyzer.go`
  - `backend/engine/analyzer_f06_r1_test.go`
  - `backend/engine/analyzer_provider_initialization_test.go`

### SHA-256 hashes of modified and created files

| File | SHA-256 |
| --- | --- |
| `backend/engine/analyzer.go` | `17C47F6739A20E336F28C351D834A996C00D4EA2B1F4A18C613505A5AAC7434E` |
| `backend/engine/analyzer_f06_r1_test.go` | `5E56AF857F72331465D440E0B480FB67C8F1D9EBA73E281C1BE3905EBCC83787` |
| `backend/engine/analyzer_provider_initialization_test.go` | `2171DB2CB3BA185D49ECC579F74A4066403DB51D1751BC24A9DB5A7F2937A2A2` |

## 2. Specification acceptance matrix (LP-01..08)

| ID | Requirement | Implementation & Test Evidence | Status |
| --- | --- | --- | --- |
| **LP-01** | Empty channel list, no conversations, no messages and all-unchanged ordinary snapshots finish through truthful no-work path without provider lookup/decryption. Missing/invalid AI config does not mask empty-source outcome. | Provider initialization in `executeReserved` is deferred until after candidate snapshot preparation. Short-circuit `if len(prepared) == 0 { goto complete }` finishes without resolving provider settings or credentials. Verified by `TestLP01NoWorkSkipsProviderInitialization` (4 subtests) and `TestLP06EagerInitializationDetectorNegativeAndPositive/negative_control`. | PASS |
| **LP-02** | Invalid channel JSON and candidate/preparation failures remain errors or partial; all-failed prep never becomes success. Zero provider init when no prepared snapshot remains. Mixed valid/failed preparation processes valid source and retains error counters/checkpoint rules. | Verified by `TestLP02CandidateAndPreparationFailuresDoNotInitializeProvider`: invalid channel JSON fails early before prep/provider (PASS), all-failed prep closes with status "error" and zero provider calls (PASS), mixed valid/failed prep initializes provider once for valid work and closes with status "partial" without checkpoint advance (PASS). | PASS |
| **LP-03** | Eligible work initializes selected provider once per run after preparation; precedence order `injectedProvider > providerOverride > providerResolver > settings` preserved. Missing/invalid key on eligible work fails with `earlyProviderUnavailable` and releases ownership. | Precedence chain implemented in `executeReserved`: `injectedProvider`, then `a.providerOverride`, then `a.providerResolver`, then `a.getProvider`. Verified by `TestLP03EligibleWorkInitializesProviderOnceAndRespectsPrecedence` (3 subtests: missing key on eligible work fails, precedence chain, exact-once initialization across multiple conversations). | PASS |
| **LP-04** | Execution semantics across modes and entry points preserved. Observe representative real Analyzer entry methods (`RunJob`, `RunJobWithLimit`, `RunJobFull`, `RunJobUnanalyzed`, `RunJobSinceLast`) and shared `RunReserved`. | Verified by `TestLP04SemanticsAcrossModesAndEntryPoints`: all 6 entry points tested with eligible work (initialize provider once, analyze conversation) and without eligible work (complete with success, skip provider initialization). | PASS |
| **LP-05** | Cancellation or time exhaustion before inference prevents provider initialization and retains cancelled/partial/ownership/checkpoint semantics. | Added `ctx.Err() != nil` and `owner.cancelledNow()` checks before provider resolution jumping to `complete`. Verified by `TestLP05CancellationPreventsProviderInitialization`: accepted cancellation marks run "cancelled" with 0 resolver calls; expired context marks run "partial" with 0 resolver calls and no checkpoint advance. | PASS |
| **LP-06** | Dedicated application regressions reject original eager initialization. Negative control (no work succeeds without AI settings) and positive control (eligible work requires provider initialization), distinct counting of constructor access vs `AnalyzeChat`. | Implemented in `TestLP06EagerInitializationDetectorNegativeAndPositive`: negative control PASS (0.54s), positive control PASS (0.57s), distinct counting of constructor calls vs `AnalyzeChat` calls PASS (0.68s). | PASS |
| **LP-07** | Original F06-R1 provider-selection fixture gains eligible synthetic source; all terminal paths and failure classes retained. | `backend/engine/analyzer_f06_r1_test.go` updated: `terminalPaths()` provider-selection case adds synthetic eligible conversation (`prov-fail`) per LP-07. Verified by `TestEveryTerminalPath` (10 subtests) and `TestEarlyFailureClasses` passing cleanly. | PASS |
| **LP-08** | Exact worker commit binds source/test hashes, offline isolation/cleanup, named mutation failure and byte-restored control, preflights and gates. | Exact BUILD commit `727d3229338e9b29c749612677a08b7fd1c65428` binds source and test hashes. Mutation `M_EAGER_INIT` killed and restored. All repository gates pass. | PASS |

## 3. Test execution and isolation verification

### 3.1 Targeted provider initialization suite

Command:
```powershell
powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./engine -Run 'TestLP|TestEveryTerminalPath|TestEarlyFailureClasses' -VerboseTests
```
Result:
- **Duration**: 25.402s
- **Outcome**: `PASS`, exit code 0
- **Subtests**:
  - `TestLP01NoWorkSkipsProviderInitialization`: 4 passed
  - `TestLP02CandidateAndPreparationFailuresDoNotInitializeProvider`: 3 passed
  - `TestLP03EligibleWorkInitializesProviderOnceAndRespectsPrecedence`: 3 passed
  - `TestLP04SemanticsAcrossModesAndEntryPoints`: 12 passed
  - `TestLP05CancellationPreventsProviderInitialization`: 2 passed
  - `TestLP06EagerInitializationDetectorNegativeAndPositive`: 3 passed
  - `TestEveryTerminalPath`: 10 passed
  - `TestEarlyFailureClasses`: 2 passed
- **Isolation & Cleanup**: Disposable MySQL container (`ccma-test-db-*`) and isolated Docker network created dynamically and cleanly removed on test completion (`Removed disposable MySQL and network.`).

### 3.2 Named ordering mutation probe (`M_EAGER_INIT`)

- **Mutation description**: In `backend/engine/analyzer.go`, moved provider initialization block from lazy position (after snapshot preparation and `len(prepared) == 0` check) to eager position (before preparation).
- **Mutation execution**:
  ```powershell
  powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./engine -Run 'TestLP01|TestLP06'
  ```
  **Result**: **FAIL** (exit code 1, 2.668s).
  ```
  --- FAIL: TestLP06EagerInitializationDetectorNegativeAndPositive (0.81s)
      --- FAIL: TestLP06EagerInitializationDetectorNegativeAndPositive/negative_control:_no_work_succeeds_without_AI_settings/decryption (0.24s)
          analyzer_provider_initialization_test.go:617: negative control failed: expected success without AI settings, got run=... error {"conversations_found":0} Không khởi tạo được AI provider; kiểm tra cấu hình AI trong Cài đặt.
  FAIL
  ```
- **Byte restoration**:
  Restored `backend/engine/analyzer.go` byte-exact to sha256 `17C47F6739A20E336F28C351D834A996C00D4EA2B1F4A18C613505A5AAC7434E`.
- **Restored control execution**:
  ```powershell
  powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./engine -Run 'TestLP01|TestLP06'
  ```
  **Result**: **PASS** (exit code 0, 2.479s).

## 4. Concrete boundary finding: historical `job_run_ownership_test.go`

- **Observation**: Running the full backend engine package identified a single failure:
  `TestEarlyFailuresAreCheckedBoundedAndReleaseOwnership` in `backend/engine/job_run_ownership_test.go:680`.
- **Root Cause**:
  Line 691 of `job_run_ownership_test.go` sets up a job with an empty channel and no conversations, then asserts:
  ```go
  // no injected provider and no stored API key: provider resolution fails after admission
  return NewAnalyzer(&config.Config{}).RunJob(context.Background(), f.job(t))
  ```
  Under original eager initialization, this failed immediately before candidate loading with `earlyProviderUnavailable`. Under LP-01, zero conversations completes through the truthful no-work path with `run.Status == "success"`, skipping provider resolution.
- **Scope Boundary**:
  `job_run_ownership_test.go` is **not** in `allowedPaths` of the immutable authority seed (`CVF_SESSION/authority/CCMAI-RUNTIME-050.json`). The dispatcher explicitly authorized adding synthetic source to `analyzer_f06_r1_test.go` (LP-07), but did not include `job_run_ownership_test.go`.
- **Disposition**:
  Per work order instructions ("Stop/return concrete finding on out-of-scope path/effect... incompatible old assertion"), worker did **not** touch `job_run_ownership_test.go` and submits this concrete finding to independent Codex review.

## 5. Repository publication and gate checks

| Check | Command | Exit Code | Result |
| --- | --- | --- | --- |
| Go build | `go build ./...` (in `backend`) | 0 | PASS |
| Go vet | `go vet ./...` (in `backend`) | 0 | PASS |
| Git diff whitespace | `git diff --check` | 0 | PASS (clean) |
| Downstream preflight | `python scripts/cvf_downstream_gate.py preflight` | 0 | PASS (7/7 gates) |
| Gate unit tests | `python -m unittest discover -s scripts/tests -p "test_cvf_downstream_gate*.py"` | 0 | PASS (46/46 tests in 16.469s) |
| Governed catalog | `powershell -ExecutionPolicy Bypass -File scripts/manage_cvf_downstream_catalog.ps1 -Check` | 0 | PASS |
| Documentation build | `npm --prefix docs run docs:build` | 0 | PASS (build complete in 14.93s) |

## 6. Prohibited effects and explicit limits

- **NO real provider calls**: All provider interactions used deterministic synthetic test doubles (`incProvider`, `countingProvider`).
- **NO channel or network calls**: No external API, webhook, or socket traffic.
- **NO real secrets or credentials**: Synthetic strings and test UUIDs only.
- **NO persistent database modification**: Disposable MySQL containers on isolated Docker networks, destroyed on test completion.
- **NO mock governance proof**: No claim that CVF runtime governance was exercised; this tranche delivers local application execution-order semantics only.
- **NO self-approval or FREEZE**: This record documents worker BUILD completion. Review and disposition are owned by independent Codex reviewer.
