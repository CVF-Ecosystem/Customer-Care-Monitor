# R139 continuation source return — 2026-10-21

Status: SOURCE PATCH RETURNED; WORKER PYTHON CHECK FAILED ON A TEST FIXTURE; ROOT VERIFICATION PENDING
Risk: R2
Worker: Luna Max delegated source repair worker
Reviewer: Root

The consolidated RF1–RF4 changes are limited to the three authorized source paths. The exact source hashes, check reservations, result details, prelaunch Go wrapper transcript, and raw output hashes are recorded in [the compact source return](probes/r139_integrated_20261021/r2_worker_source_return.json), with the pre-check reservation at [r2_worker_check_reservation.json](probes/r139_integrated_20261021/r2_worker_check_reservation.json).

The supervisor now waits for final MySQL TCP readiness with `SELECT 1` against the admitted database, bounded by a monotonic 60-second window, at most 45 probes, one-second spacing, per-command limits, the global deadline, and the cleanup reserve. It rejects a success that arrives after the readiness deadline and does not retry a Docker timeout or journal/deadline error. Protected inventory comparison normalizes only ordering within container `Networks` memberships and retains raw before/after fields and identities. Socket result 10035 triggers a bounded query through the admitted, hash-pinned PowerShell binary; only exit zero, empty stderr, valid exact-port JSON, and zero listeners prove absence. The Go test and Python admission now bind the fresh `fixture_r2` evidence path; the Go change is only the path constant. Listener callsites pass their admission/runtime context, and PowerShell receipts are port-qualified so API, Vite, and database checks cannot overwrite each other.

The sole worker Python command was:

```powershell
python -B -m unittest discover -s scripts/tests -p "test_r139_execution_supervisor.py"
```

It exited 1 after 27 of 28 tests passed. The error was in the new cleanup-reserve test: its mocked `write_new` did not create `runtime/commands`, so the later stdout receipt write raised `FileNotFoundError`. I corrected only that test fixture by creating `runtime/commands`. The original failing log remains preserved at `.cvf/runtime/r139_integrated_execution_r2_20261021_source_checks/python_unittest.log` (SHA-256 `fff212d1209c74010852036c79582c6ffc033e50d878c7a8507eb08c75a2bc4d`). I did not rerun Python. Root owns the existing unspent Python verification slot; the fixture correction remains unverified by the worker.

The one actual offline Go compile-only command was:

```powershell
$env:GOPROXY='off'; $env:GOSUMDB='off'; $env:GOTOOLCHAIN='local'; go -C backend test -c -o ../.cvf/runtime/r139_integrated_execution_r2_20261021_source_checks/intervention_execution_browser.test.exe ./api/handlers
```

It exited 0 and did not execute the tests. The produced binary is `.cvf/runtime/r139_integrated_execution_r2_20261021_source_checks/intervention_execution_browser.test.exe`, SHA-256 `3aab6d901d5abfbd77d87102eb9c26b0f40cab94d87087f4cc293a3ec0c5a5d6`; the compiler was `C:\Program Files\Go\bin\go.exe`, SHA-256 `7d828191ba32519a9c9361789ab647486236ed45c660889196c7770a8ff1985c`. Its raw log is empty (SHA-256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`).

Before that compile, one PowerShell wrapper attempt failed while opening its redirected log file. It used `../.cvf/runtime/r139_integrated_execution_r2_20261021_source_checks` from the project root, which resolved under the workspace parent. `Out-File` failed before launching `go`; no compiler process or binary came from that attempt. The exact command and returned PowerShell error are retained at `.cvf/runtime/r139_integrated_execution_r2_20261021_source_checks/go_wrapper_prelaunch_failure.log` (SHA-256 `a3ae3aaddf6fb5f4c1e3724fb2610c3e5752eb5863a0abc1f26b0f43b9b29bc5`). The corrected wrapper wrote its log relative to the project root, then launched the single authorized compile.

The previous campaign remains `HOLD_SETUP`. Its sole supervisor exited before Go/backend/Vite/browser/provider; six MySQL probes ended at 05:59:14 UTC, initialization completed at 05:59:20, and final TCP readiness appeared at 05:59:21.477. One fixture start and final stop were consumed; native, backend, Vite, browser, auth, grant, execute, and provider calls remained zero. The original report and compact evidence are [R139 runtime setup failure and cost disposition](R139_RUNTIME_SETUP_FAILURE_AND_COST_DISPOSITION_2026-10-21.md) and [the retained runtime failure record](probes/r139_integrated_20261021/integrated_runtime_failure.json). The one-USD/one-call cumulative ceiling remains unchanged; no provider call or runtime continuation occurred in this source stage.

Return-time closeability was rechecked against the current work-order contract. Source scope, R2 ceiling, external-effect class, and commit owner are unchanged, and this return claims only source preparation. The focused worker result is not closeable until Root verifies the corrected Python fixture and completes the independent source review. G0 remains HOLD; G1/G2 and E01–E06 remain NOT RUN. No runtime acceptance, fixture freshness, or paid-call proof is claimed. The admitted future runtime root `.cvf/runtime/r139_integrated_execution_r2_20261021` remains absent; worker logs and the compile binary are isolated under the ignored sibling `r139_integrated_execution_r2_20261021_source_checks`.
