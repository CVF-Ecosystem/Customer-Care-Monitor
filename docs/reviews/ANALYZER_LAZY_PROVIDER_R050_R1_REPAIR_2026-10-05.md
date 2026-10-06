# R051 R050 R1 test and evidence repair record

Date: 2026-10-05. Status: REPAIRED, REVIEW_PENDING (not accepted). Worker: Claude (`REPAIR_WORKER` / `BUILD COMMIT_STEWARD`). Risk R2.
Work order: [CCMAI_RUNTIME_051](../work_orders/CCMAI_RUNTIME_051.md), Repair SPEC: [ANALYZER_PROVIDER_REPAIR_R051_2026-10-05](../specs/ANALYZER_PROVIDER_REPAIR_R051_2026-10-05.md), Independent Review: [CCMAI_RUNTIME_050_INDEPENDENT_REVIEW_2026-10-05](CCMAI_RUNTIME_050_INDEPENDENT_REVIEW_2026-10-05.md).

## 1. Authority, baseline, and file hashes

- **Separate repair seed**: `CVF_SESSION/authority/CCMAI-RUNTIME-051.json` (committed at `a54cb73007f081fe4bbaa4baa11d28fed4a7d937`).
- **Tranche baseCommit**: `a54cb73007f081fe4bbaa4baa11d28fed4a7d937`.
- **Inherited production baseline**: `727d3229338e9b29c749612677a08b7fd1c65428` (`feat(engine): source-first application provider initialization (R050)`).
- **Original R050 authority seed**: `CVF_SESSION/authority/CCMAI-RUNTIME-050.json` (committed at `1008ab41f0693e2814cd06dfdb0fed98273a1167`, unchanged).
- **Role acknowledgment commit**: `9f010bf9e34e5a95ae86c47cbffba711200388d0` (`docs: acknowledge R051 repair worker role and open BUILD phase`).
- **Exact test repair commit**: `a4b378ae036ad767728fb2e0655559e9cf750b47` (`test(engine): repair ownership fixture and initialization test regressions (R051)`).
- **Working tree source status**: Canonical production source `backend/engine/analyzer.go` is byte-identical to `727d3229338e9b29c749612677a08b7fd1c65428`. Only authorized test paths were modified.

### SHA-256 hashes of modified and verified files

| File | SHA-256 | Status |
| --- | --- | --- |
| `backend/engine/analyzer.go` | `17C47F6739A20E336F28C351D834A996C00D4EA2B1F4A18C613505A5AAC7434E` | Canonical production source (byte-identical to R050 BUILD) |
| `backend/engine/job_run_ownership_test.go` | `F8456B02D831B285626BE1C8469075B2BBDCBF1D97AB9BB3395047278E168B8E` | Repaired provider-selection fixture (RP-01) |
| `backend/engine/analyzer_provider_initialization_test.go` | `495B505D035D9BC57CD6DBF1C54BCC95F1507558C94D3ABA89A4E414860E4C0C` | Repaired counters, traps, and candidate error (RP-02, RP-03) |

## 2. Findings resolution matrix (R050-R1-01..03 / RP-01..04)

| Finding / Spec ID | Description | Repair Implementation & Discrimination Evidence | Status |
| --- | --- | --- | --- |
| **R050-R1-01** / **RP-01** | `TestEarlyFailuresAreCheckedBoundedAndReleaseOwnership` provider-selection case failed under lazy init due to empty channel completing as no-work success. Required separate authority for ownership fixture. | Added eligible synthetic source `prov-fail` to setup in `backend/engine/job_run_ownership_test.go:693`. All bounded-error, stored-error, finished-state, job status, ownership release, and readmission assertions preserved. Verified: `TestEarlyFailuresAreCheckedBoundedAndReleaseOwnership` passes cleanly (0 FAIL, 0 SKIP). | RESOLVED |
| **R050-R1-02** / **RP-02** | `countingProvider` conflated item counts with batch inference calls (incremented by `len(items)`). Assertions required distinct constructor access, single calls, batch calls, and item counts. | Refactored `countingProvider` to track `singleCalls`, `batchCalls`, and `itemsCount` distinctly. In batch mode (2 items): `constructed=1`, `singleCalls=0`, `batchCalls=1`, `totalCalls=1`, `itemsCount=2`. In non-batch mode: `constructed=1`, `singleCalls=2`, `batchCalls=0`, `totalCalls=2`, `itemsCount=2`. Semantically precise assertions replaced flawed inequality checks. | RESOLVED |
| **R050-R1-02** / **RP-03** | Corrupt encrypted key tests previously bypassed `getProvider` via `providerResolver`. Production negative control only tested missing settings with empty channel. Missing candidate selection query error case in LP-02. | 1) Added `trapAISettingsQueries` GORM callback to trap/observe queries to `app_settings` for AI settings.<br>2) Tested LP-01 production path with corrupt key `X'DEADBEEF'` and no resolver/override: `atomic.LoadInt64(queries) == 0`, run succeeds cleanly.<br>3) Added candidate selection query failure to LP-02 via `failTable(t, "conversations", ...)`: fails with `earlyCandidateSelection` before provider, `queries == 0`.<br>4) Added all-failed preparation with corrupt key: `queries == 0`.<br>5) Added mixed prep with corrupt key on production path: fails at provider init with `earlyProviderUnavailable`, `queries > 0`.<br>6) Tested LP-06 positive controls for both missing settings and corrupt key: both fail with `earlyProviderUnavailable`, `queries > 0`. | RESOLVED |
| **R050-R1-03** / **RP-04** | Worker receipt missing / incomplete; incorrect acknowledgment SHA; conflicting test counts; default network and lack of `-v` volume cleanup in helper script. | 1) Executed truthful isolated campaign via `docs/reviews/probes/r050_r1_campaign_runner.ps1`.<br>2) Used `--internal` Docker network without host ports.<br>3) Teardown uses `docker rm -f -v` and network deletion; verified container, network, and anonymous volumes absent.<br>4) Machine receipt saved at [docs/reviews/probes/r050_r1_worker_receipt.json](probes/r050_r1_worker_receipt.json) recording commands, exits, log hashes, duration, mutation results, and resource absence.<br>5) Acknowledgment commit identity `9f010bfdocs: acknowledge R051 repair worker role and open BUILD phase` accurately recorded.<br>6) Python caches cleaned and verified absent before gate execution. | RESOLVED |

## 3. Machine campaign verification

### 3.1 Execution environment and resource isolation

- **Runner**: `docs/reviews/probes/r050_r1_campaign_runner.ps1`
- **Machine receipt**: `docs/reviews/probes/r050_r1_worker_receipt.json`
- **Network**: `ccmai-r050-r1-net-bcf12703` (`internal=true`, no external network, no host port bindings)
- **Database**: `ccmai-r050-r1-db-bcf12703` (`mysql:8.0`, `--log-bin-trust-function-creators=1`, `--max-connections=1000`)
- **Go Container**: `golang:1.26-alpine` (`--pull=never`, `GOFLAGS=-mod=readonly`, `GOPROXY=off`, `GOTOOLCHAIN=local`, `CGO_ENABLED=0`)
- **Mounts**: Readonly bind mounts for task `/src` and offline `/go/pkg/mod`.

### 3.2 Command results

| Command | Args | Exit | Duration | Log SHA-256 |
| --- | --- | --- | --- | --- |
| `exact-repair` | `test ./engine -json -count=1 -p 1 -timeout 300s -run TestLP\|TestEveryTerminalPath\|TestEarlyFailureClasses\|TestEarlyFailuresAreCheckedBoundedAndReleaseOwnership\|TestFL\|TestF05\|TestF06\|TestTerminalRunHoldsTheSlot` | 0 | 194.94s | `312AEF07846D3DC7E5205E70138B8DC261DA5ACAA1C0CD7233EF8145161E2AC4` |
| `build` | `build ./...` | 0 | 117.95s | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `vet` | `vet ./...` | 0 | 127.10s | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |

### 3.3 Test event breakdown (`exact-repair`)

- **Top-level test suites**: 42 passed
- **Total completed test events**: 98
- **PASS**: 98
- **FAIL**: 0
- **SKIP**: 0

Key suites verified:
- `TestEarlyFailuresAreCheckedBoundedAndReleaseOwnership`: PASS (provider selection and invalid input both checked, bounded, and released ownership)
- `TestLP01NoWorkSkipsProviderInitialization`: 5/5 PASS (empty channel, 0 convs, 0 msgs, all unchanged, unanalyzed mode; all verify 0 settings queries on production path)
- `TestLP02CandidateAndPreparationFailuresDoNotInitializeProvider`: 5/5 PASS (invalid JSON, candidate query error, all-failed prep, mixed prep with resolver, mixed prep with corrupt key)
- `TestLP03EligibleWorkInitializesProviderOnceAndRespectsPrecedence`: 4/4 PASS
- `TestLP04SemanticsAcrossModesAndEntryPoints`: 12/12 PASS
- `TestLP05CancellationPreventsProviderInitialization`: 2/2 PASS
- `TestLP06EagerInitializationDetectorNegativeAndPositive`: 5/5 PASS (negative control with corrupt key, positive missing key, positive corrupt key, batch distinct count, non-batch distinct count)
- `TestEveryTerminalPathHonorsAnAcceptedCancel`: 10/10 PASS
- `TestEarlyFailureClassesNeverLogOrStoreTheUnderlyingCause`: 3/3 PASS
- All FL, F05, F06 regressions: PASS

### 3.4 Ordering mutation sensitivity probe (`M_EAGER_INIT`)

- **Mutated file**: `backend/engine/analyzer.go` (applied only in isolated task export copy, canonical repo untouched)
- **Mutant SHA-256**: `C545E182779D75BA1DEC75A291DF8141EEA9ECBA65D8F17B04F22B62C140BBB9`
- **Mutant execution**: `go test ./engine -json -count=1 -run TestLP06EagerInitializationDetectorNegativeAndPositive/negative_control`
  - **Outcome**: Exited with code `1` (FAIL as expected; eager resolution failed with `earlyProviderUnavailable` on corrupt key)
  - **Log SHA-256**: `D528F15F2129C2292404062C7F285B62DCE5610B1E3D292BDEE844A23469D09E`
- **Restoration verification**:
  - Restored file verified byte-for-byte: `17C47F6739A20E336F28C351D834A996C00D4EA2B1F4A18C613505A5AAC7434E`
  - Restored test rerun: Exited with code `0` (PASS)
  - Log SHA-256: `991D04DFF481724CD560201B4F22ABA43C0A0F838C456E75123F55027CF31E0B`

### 3.5 Resource teardown and volume absence

- `databaseRemoveExit`: 0 (`docker rm -f -v`)
- `networkRemoveExit`: 0 (`docker network rm`)
- `databaseAbsent`: `true`
- `networkAbsent`: `true`
- `anonymousVolumesAbsent`: `true`

## 4. Repository boundaries and governance

- **No live provider or credential use**: Purely synthetic loopback stubs and mock database fixtures.
- **No external network**: Docker network `--internal` with zero host port bindings.
- **No production code drift**: `backend/engine/analyzer.go` verified byte-identical to `727d3229338e9b29c749612677a08b7fd1c65428`.
- **No seed widening**: Original R050 seed `CVF_SESSION/authority/CCMAI-RUNTIME-050.json` untouched.
- **No self-approval**: Handback to independent Codex review as `REVIEW_PENDING`.
