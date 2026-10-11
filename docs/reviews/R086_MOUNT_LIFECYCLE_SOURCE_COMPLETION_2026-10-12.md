# R086 mount lifecycle source completion

Date: 2026-10-12  
Tranche: `CCMAI-RUNTIME-086`  
Disposition: worker BUILD handback; independent review pending

## Result and scope

The authorized source-only pass is complete in `scripts/jobrun_fence_fixture.py` and `scripts/tests/test_jobrun_fence_fixture.py`. The one worker harness invocation exited `0`: 27 tests ran and passed. The 23 original test method bodies are unchanged; four focused tests were added. No source or test files changed after the suite. This result covers local Python source and its in-memory harness only.

The pending validation phase permits an empty resolved `Mounts` list only after the existing ownership, no-port, no-bind, internal-network, and exact configured tmpfs checks pass. Missing, null, non-list, malformed, or unauthorized nonempty mount observations fail closed. Started validation requires the exact owned network ID and exactly one writable tmpfs mount at `/var/lib/mysql`. Campaign flow validates the pending observation after create, then validates started container/network ownership before readiness SQL and again at the existing schema-probe boundary.

The runtime receipt now keeps bounded observations for each pending/started check: phase, role/task ID, valid expected container/network/image IDs, mount shape/count and selected type/destination/RW facts, configured tmpfs target/count and options-match facts, and accepted/rejected status with a safe failure code. It omits mount sources and raw host paths, full inspect objects, environment values, and unrelated labels.

## Acceptance map

| Requirement | Implementation and worker evidence |
|---|---|
| ML01 — pending and started validation are distinct and fail closed | `validate_container_inspect(..., mount_phase=...)` enforces pending empty-list allowance only after configuration checks, while started requires the exact network ID and one authorized writable tmpfs mount. Direct cases are in `test_container_inspect_distinguishes_pending_and_started_mounts`. |
| ML02 — start and validate isolation before any SQL | `Campaign._create_container` records pending validation; `_validate_started_ownership` records started validation before readiness and at the existing later schema-probe boundary. Campaign tests assert order and that invalid started inspection causes no readiness call. |
| ML03 — portable tests exercise Campaign order and rejection cases | Four added tests cover valid empty-pending then started mounts, exact started network ID, invalid started mount shapes/values, invalid pending mounts/configuration, and no readiness SQL for rejected started state. Existing 23 methods and their AST bodies match the pre-edit baseline. |
| ML04 — bounded sanitized phase observations survive validation failures | `_mount_lifecycle_observation` selects only typed facts and omits source/path values. Campaign tests inspect persisted success/failure receipts and assert host-like source/destination strings are absent. |
| ML05 — one captured worker suite with physical identities | One `unittest discover` process ran; raw output and capture receipts are listed below. Source/test physical SHA-256 values were identical before and after the invocation. No Docker daemon, SQL server, fixture campaign, Go, provider, or external network was invoked. |
| ML06 — claim boundary and readiness | This record reports source/harness evidence only. R085 remains a failed `MOUNT_INVALID` before start/SQL with cleanup verified; its exact raw `Mounts` value was unavailable. Actual database state/settings/grants/DDL are not observed here; fence/integration readiness remains OPEN and deployed facts UNKNOWN. |
| ML07 — independent disposition | Root's independent source/protection review and one reviewer suite remain pending. Worker has not committed, approved runtime effects, or closed the tranche. The known 55 R078/R079 dead links continue to block FREEZE. |

## Captured evidence

The worker runner launched exactly one command: `python -m unittest discover -s scripts/tests -p test_jobrun_fence_fixture.py -v`. The process returned `0` without timeout after 2.077573 seconds. The retained raw stderr reports `Ran 27 tests in 1.531s` followed by `OK`; all 27 test event lines passed. Raw stdout is empty. Source SHA-256 was `952cbfa16b60da1ac2f0189ed3ec130adcb03f9e5a168751d3d7134085cd6e0e` before and after; test SHA-256 was `8d2dfd982fcf1b8c14f1387ee12508c2a51e0ecd77f85a7a1ac568661ec90f62` before and after.

The capture wrapper's initial summary parser did not account for carriage returns before line endings and left its parsed count null. The retained raw stderr was then parsed read-only into the separate raw-summary receipt; the suite was not rerun. Before the suite, a static helper also stopped on an incorrect expected count of 26 while observing 27 methods. This was a preflight counter typo, not a test result or source defect; no tests had run at that point, and no source/test correction was made for it. Raw output from that failed helper was not retained and is not reconstructed.

Pre-suite static review recorded all 23 baseline method-body hashes unchanged, exactly four new methods, the protected R083 namespace/order constants unchanged, and `git diff --check` exit `0`. Root's pre-suite static review assistance was incorporated: event ordering uses actual enumerated positions; task/container/network/image IDs are asserted at their exact fields; started validation requires exact `NetworkID`; configured option facts distinguish string presence from an authorized options match.

Worker probe receipts and raw captures are under `docs/reviews/probes/r086_mount_source/worker/`:

- `pre_edit_baseline.json`
- `pre_suite_static_review.json`
- `pre_suite_helper_failure_summary.json`
- `worker_harness_receipt.json`
- `worker_harness_raw_summary.json`
- `worker_harness_stdout.raw.bin`
- `worker_harness_stderr.raw.bin`
- `worker_launcher_errors.json`
- `capture_worker_suite.py` and `pre_suite_static_review.py`

The sole worker suite budget is used. Root owns the commit and independent review; no source repair or additional worker suite is authorized by this handback.
