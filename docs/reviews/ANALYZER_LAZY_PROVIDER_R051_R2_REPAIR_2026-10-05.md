# CCMAI-RUNTIME-051 Round 2 Test Repair and Evidence Report

Date: 2026-10-05.  
Author: Claude (REPAIR_WORKER / BUILD COMMIT_STEWARD).  
Independent Reviewer: Codex (independent REVIEWER).  
Tranche: `CCMAI-RUNTIME-051` (Round 2 repair).  
Exact test repair commit: `145bd41109c1e3ac3fb1a85f261c61f668b2fe4d`.  
Before-edit acknowledgment commit: `c228931df25026211cf4e33d4586db7020bc8b5c`.  
Authority seed: separate `a54cb73007f081fe4bbaa4baa11d28fed4a7d937` (committed before activation).  
Predecessor reviews: [R050 review](CCMAI_RUNTIME_050_INDEPENDENT_REVIEW_2026-10-05.md), [R051 round 1 re-review](CCMAI_RUNTIME_051_INDEPENDENT_REREVIEW_2026-10-05.md).

---

## 1. Resolution of Round 1 Re-Review Findings

### R051-R1-01 — P1: Ordering mutation killed through behavioral detector (RP-04)
- **Root cause in Round 1:** The previous runner inserted provider resolution before variable declarations, causing Go compile failure (`BUILD_ERROR`) rather than a behavioral assertion failure.
- **Round 2 Resolution:**
    - Successor runner `docs/reviews/probes/r051_r2_campaign_runner.ps1` applies a single scoped ordering mutation to `executeReserved` in the isolated copy of `backend/engine/analyzer.go`: eager provider resolution is placed right after `var provider ai.AIProvider` and before conversation selection / snapshot preparation.
  - Compiles cleanly with zero compilation errors (`hasBuildFail: false`).
  - Executed against named detector `TestLP06EagerInitializationDetectorNegativeAndPositive/negative_control`: failed behaviorally with exit code 1 and exactly 1 named failure event (`failEventCount: 1`), confirming that eager resolution improperly triggers provider decryption on empty-work runs.
  - Restored original bytes byte-for-byte (`17C47F6739A20E336F28C351D834A996C00D4EA2B1F4A18C613505A5AAC7434E`), re-executed negative control on restored source, and confirmed clean pass (`exitCode: 0`, `restoredPassed: true`).
  - Result: `killed: true`.

### R051-R1-02 — P2: Truthful machine receipt, exact git archive, and volume inventory (RP-04)
- **Root cause in Round 1:** Runner copied the backend tree (introducing CRLF differences on 4 files) and literal boolean for volume cleanup without capturing volume names.
- **Round 2 Resolution:**
  - Exported exact source via `git archive --format=tar --output=$archive $BuildCommit backend` and extracted cleanly, ensuring zero CRLF or working tree drift (archive SHA256: `B4D4DAFF222388D1F721294B0D2AB47899BB70943966BFF1742B7C045054BE38`).
  - Recorded exact acknowledgment commit `c228931df25026211cf4e33d4586db7020bc8b5c` and exact test repair commit `145bd41109c1e3ac3fb1a85f261c61f668b2fe4d`.
  - Derived all test counts dynamically from JSONL event stream: 42 top-level test suites passed, 98 completed test events, 98 PASS, 0 FAIL, 0 SKIP.
  - Inspected and recorded disposable database anonymous volume before teardown (`d2598abb0df0d942d83d17d3a58a42535d2c39917894054b6e85ceef36b3dd0f`), and verified its complete removal post-teardown (`remainingNamedVolumes: []`, `anonymousVolumesAbsent: true`).
  - Machine receipt committed at `docs/reviews/probes/r051_r2_worker_receipt.json`.

### R051-R1-03 — P2: Persistence, notifier, and observational query-error checks (RP-03)
- **Root cause in Round 1:** Ignored `.Count(...).Error` on usage/result queries; LP01 read returned summary instead of DB; positive controls lacked stored run/notifier assertions.
- **Round 2 Resolution:**
  - Updated `backend/engine/analyzer_provider_initialization_test.go`:
    - Replaced every `.Count(...)` call with explicit `if err := ...Count(...).Error; err != nil { t.Fatalf(...) }`, ensuring immediate test failure if a count query fails.
    - Added `trapJobNotifications(t *testing.T) *notifier` helper to safely observe `sendJobNotifications` dispatch and arguments, restoring default behavior via `t.Cleanup`.
    - **LP-01:** Reloaded stored `models.JobRun` and `models.Job` from DB; verified `storedRun.Status == "success"`, `storedRun.FinishedAt != nil`, `!JobRunActive(f.tenantID, f.jobID)`; parsed summary directly from `storedRun.Summary`; asserted `nt.count() == 0` on no-work runs.
    - **LP-02:** Checked `storedRun` reload, `!JobRunActive`, zero notifications on invalid channel JSON, candidate query failure, and all-failed prep; verified notification dispatch on mixed valid/failed preparation with `output_schedule = 'instant'`.
    - **LP-03:** Added `trapJobNotifications` (asserting 0 notifications), reloaded `storedRun` (`Status == "error"`, `FinishedAt != nil`, error message contains `"Không khởi tạo được AI provider"`), and verified `!JobRunActive` on missing key and undecryptable key positive controls.
    - **LP-06:** Added notification traps (0 calls), `storedRun` reloads, ownership release assertions, and error-checked count queries to negative control and both missing/corrupt key positive controls.

---

## 2. Test Campaign Summary

| Step | Command | Result | Duration |
| --- | --- | --- | --- |
| 1. Compile test binary | `go test -c -o /dev/null ./engine` | **PASS (exit 0)** | 72.83s |
| 2. Static vet | `go vet ./engine` | **PASS (exit 0)** | 70.65s |
| 3. Exact repair test suite | `go test ./engine -json -count=1 -p 1 -timeout 300s -run "TestLP\|..."` | **PASS (exit 0)** | 197.94s |
| 4. Ordering mutation test | `go test ./engine -json -count=1 -run '.../negative_control'` (mutated) | **KILLED (exit 1)** | 72.48s |
| 5. Baseline restoration check | `go test ./engine -json -count=1 -run '.../negative_control'` (restored) | **PASS (exit 0)** | 74.72s |
| 6. Resource cleanup | `docker rm -f -v`, `docker network rm` | **CLEAN (0 remaining)** | <1s |

- **Aggregated test events:** 42 top-level test suites, 98 completed events: **98 PASS, 0 FAIL, 0 SKIP**.
- **Mutation result:** `M_EAGER_INIT_ORDERING` killed behaviorally without build errors; baseline restored byte-for-byte.
- **Resource verification:** Container, network, and anonymous volume confirmed absent.

---

## 3. Preserved Boundaries & Invariant Truth

- **Production code invariance:** `backend/engine/analyzer.go` remains untouched and byte-identical to `727d3229338e9b29c749612677a08b7fd1c65428` (`17C47F6739A20E336F28C351D834A996C00D4EA2B1F4A18C613505A5AAC7434E`).
- **Authority seed invariance:** Original R050 authority seed `1008ab41f0693e2814cd06dfdb0fed98273a1167` and R051 authority seed `a54cb73007f081fe4bbaa4baa11d28fed4a7d937` remain untouched.
- **Historical retention:** Predecessor review returns and worker submissions remain preserved as historical records.
- **Parked checkpoints:** `OWNER_DEFERRED_FACEBOOK_ZALO_OA_ACCOUNTS` remains parked.
- **Scope limitation:** Purely synthetic disposable local test repair; no real provider calls, live channel calls, or CVF runtime governance claims.

Handed over to **Codex (independent REVIEWER)** under `REVIEW_PENDING` for re-review of commit `145bd41109c1e3ac3fb1a85f261c61f668b2fe4d`.
