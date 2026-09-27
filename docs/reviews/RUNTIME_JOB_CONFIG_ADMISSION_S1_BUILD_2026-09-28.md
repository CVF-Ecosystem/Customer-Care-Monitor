# CCMAI-RUNTIME-009 BUILD evidence — job-dispatch configuration admission

**Tranche:** `CCMAI-RUNTIME-009` · **Role:** IMPLEMENTATION_WORKER (Claude) · **Date:** 2026-09-28 · **Base commit:** `720f5b6` · **Authority:** [SPEC](../specs/RUNTIME_JOB_CONFIG_ADMISSION_S1_2026-09-28.md), [work order](../work_orders/CCMAI_RUNTIME_009.md).

## Source change

`backend/api/handlers/jobs.go` — `TestRunJob` and `TriggerJob`:

1. After the existing tenant-scoped job lookup (unchanged 404 on miss), each
   handler now calls `loadJobDispatchConfig()` (a package variable defaulting
   to `config.Load`) before launching any goroutine or returning 202. On
   `err != nil` or a nil config, the handler returns generic
   `500 {"error":"job_start_failed"}`, logs only `"<endpoint> for job %s not
   admitted: configuration invalid"` (job name, not the raw validation error),
   and does not launch a goroutine, store a cancel function, or create a job
   run.
2. On valid configuration, the handlers call `startTestRunJob(job, cfg)` /
   `startTriggerJob(job, cfg, triggerJobParams{...})` — package-variable
   launchers (also allowed test seams) that carry the exact validated
   `*config.Config` pointer into `engine.NewAnalyzer`. Neither launcher
   reloads configuration. `TriggerJob`'s existing mode resolution
   (`unanalyzed`/`conditional`/`since_last`, `full=true` back-compat), date
   range, and `limit` parsing are unchanged and are passed through unchanged
   in `triggerJobParams`. `TestRunJob`'s hardcoded limit of 3 is unchanged.
   Cancellation registration (`jobCancelFuncs.Store`/`Delete`), the 30-minute
   context timeout, and panic recovery/logging are unchanged, only moved
   inside the launcher variables.
3. Existing 202 response bodies (`test_run_started`, `job_triggered`) are
   byte-for-byte unchanged. No change to `CancelJob`, `ListJobRuns`, or any
   other handler in the file.

No edits to `backend/config`, `backend/engine`, models/migrations, channels,
agents, adapters, scheduler, provider clients, permissions, frontend,
notifications, or CVF core, matching the work order's allowed scope.

## Test change

New `backend/api/handlers/job_config_admission_test.go` (all tests stub
`loadJobDispatchConfig`, `startTestRunJob`, `startTriggerJob`; no real
analyzer, adapter, or provider runs):

- `TestTestRunJobConfigFailureStartsNoWorker` / `TestTriggerJobConfigFailureStartsNoWorker`:
  forced `loadJobDispatchConfig` error containing a synthetic secret string
  (`ENCRYPTION_KEY=SECRET-DO-NOT-LEAK`) returns generic
  `{"error":"job_start_failed"}` with no worker launch; the secret and the
  `ENCRYPTION_KEY` name are absent from both the response body and the log;
  the log names only the failure class (`"configuration invalid"`).
- `TestTestRunJobNilConfigIsNotAdmitted` / `TestTriggerJobNilConfigIsNotAdmitted`:
  a `nil` config from a successful load is rejected the same way as an error.
- `TestTestRunJobPassesValidatedConfigToWorker` / `TestTriggerJobPassesValidatedConfigAndParamsToWorker`:
  valid config yields the unchanged 202 body, exactly one config load, one
  launch, the launcher receiving the *same pointer* (`!=` identity check) as
  the stub's `cfg`, and (for trigger) the resolved `mode`/`from`/`to`/`limit`
  reaching `startTriggerJob` unchanged via a query string
  (`mode=conditional&from=2026-01-01&to=2026-01-31&limit=7`).
- `TestTriggerJobDefaultModeAndLimitReachWorkerUnchanged`: no query string
  still resolves to `since_last`/empty dates/`maxConv=0`, proving admission
  does not alter the pre-existing default-resolution behavior.
- `TestTestRunJobWrongTenantSkipsConfigLoad` / `TestTriggerJobWrongTenantSkipsConfigLoad`:
  a job ID that exists only under another tenant returns 404 with
  `cfgLoads == 0`, proving the tenant-scoped lookup still runs first and a
  missing/foreign job never reaches configuration validation.
- `TestJobDispatchUsesRealConfigValidation`: restores `loadJobDispatchConfig`
  to the real `config.Load` and uses `t.Setenv` (not `t.Parallel`, restored by
  cleanup) with synthetic `JWT_SECRET`/`ENCRYPTION_KEY`/`DB_PASSWORD` values —
  `invalid` (too-short JWT secret) is rejected with no `"JWT"` substring
  leaked in body or log; `valid` reaches the launcher with the exact synthetic
  values on the validated config. No adapter, channel, or provider is
  touched — `config.Load` only reads environment variables and returns a
  struct.

## Mutation check (tests are meaningful)

`git stash push -- backend/api/handlers/jobs.go` reverted only the production
file (test file kept); running the new tests against that pre-BUILD source
failed to *compile* — `undefined: triggerJobParams`, `startTestRunJob`,
`startTriggerJob`, `loadJobDispatchConfig` (10 errors, `[build failed]`) —
which is a stronger signal than a runtime failure: the new tests cannot even
run without the new admission seams existing. `git stash pop` restored the
BUILD source; `go build ./...` / `go vet ./...` passed clean afterward, and
the focused suite (see below) passed again on the restored source.

## Validation commands and results

All commands run from the repository root or `backend/` as noted; disposable
MySQL used throughout via `scripts/test-backend.ps1` (own Docker network, no
host port, no host data, `--log-bin-trust-function-creators=1`, container and
network always removed in a `finally` block). The persistent Compose stack
(`ccma-db-1`, `ccma-app-1`, `ccma-nginx-1`) was running throughout and was
never touched, stopped, or connected to.

1. Focused admission tests (`backend/`):
   `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./api/handlers -Run 'TestTestRunJob|TestTriggerJob|TestJobDispatchUsesRealConfigValidation' -VerboseTests`
   → all 10 new tests `PASS` (`TestTestRunJobConfigFailureStartsNoWorker`,
   `TestTestRunJobNilConfigIsNotAdmitted`, `TestTestRunJobPassesValidatedConfigToWorker`,
   `TestTestRunJobWrongTenantSkipsConfigLoad`, `TestTriggerJobConfigFailureStartsNoWorker`,
   `TestTriggerJobNilConfigIsNotAdmitted`, `TestTriggerJobPassesValidatedConfigAndParamsToWorker`,
   `TestTriggerJobDefaultModeAndLimitReachWorkerUnchanged`,
   `TestTriggerJobWrongTenantSkipsConfigLoad`, `TestJobDispatchUsesRealConfigValidation`
   with both `invalid`/`valid` subtests); `ok ... 7.074s`.
2. Full backend suite: `powershell -ExecutionPolicy Bypass -File ../scripts/test-backend.ps1`
   (run from `backend/`) → all 13 packages `ok` (`ai`, `ai/catalog`,
   `ai/pricing`, `api/handlers` 23.4s, `channels`, `cli`, `config`, `engine`
   13.9s, `notifications`, `pkg`, `pkg/password`, `storage`, `storagecfg`).
   AutoMigrate ran clean once at container start (single migration pass; no
   duplicate-index or migration error observed in the captured log).
3. `go build ./...` and `go vet ./...` (from `backend/`, host Go 1.26): clean,
   no output, both before and after the mutation check restored the source.
4. `gofmt -l backend/api/handlers/jobs.go backend/api/handlers/job_config_admission_test.go`:
   no output (both files already formatted).
5. `git diff --check`: clean except the pre-existing, already-recorded
   `core.autocrlf` LF→CRLF notice on the handoff file (not a new whitespace
   defect in the two Go files).
6. CVF workspace doctor (also runs the governed downstream catalog
   `-Check`): `powershell -ExecutionPolicy Bypass -File
   "../.Controlled-Vibe-Framework-CVF/scripts/check_cvf_workspace_agent_enforcement.ps1"
   -ProjectPath "<project root>"` → `RESULT: PASS (25/25 checks passed)`.

## Untested branch and limitations

- The real (non-stubbed) `startTestRunJob`/`startTriggerJob` bodies — the
  actual goroutine launch, `engine.NewAnalyzer` construction, cancel
  registration, and panic recovery — are exercised only by manual code
  reading and by the fact that this is a refactor preserving prior,
  previously-tested behavior (cancellation semantics were not reintroduced
  incorrectly: the same `jobCancelFuncs.Store`/`Delete`/context-timeout/panic
  pattern that existed before this tranche is now inside the launcher
  variables, unchanged line-for-line other than the parameter it receives).
  No new test runs the real launcher end-to-end against a real analyzer,
  because doing so would require a real AI provider or a mocked analyzer,
  neither of which is authorized or exists as a seam in this tranche's scope.
- No test asserts on `CancelJob` interaction with a dispatch rejected by
  admission (there is nothing to cancel since no run/cancel-func is created);
  this matches the SPEC's "do not create a job run, store a cancel function"
  requirement, verified by the rejection tests' launch-count assertions.
- Per the SPEC, this evidence proves handler admission logic only via
  synthetic fixtures and disposable MySQL; it is not live AI-provider
  governance evidence and makes no such claim.

## Boundary confirmation

No real AI provider call, real channel/credential use, customer data,
persistent Compose database change, deployment, or push occurred. The
persistent `ccma` Compose stack was not restarted, migrated, or otherwise
touched. Agent-run, scheduler, channel OAuth/credential handlers, and
cross-path sync concurrency remain out of scope, per the SPEC.

## Status

`REVIEW_PENDING` for independent Codex R2 REVIEW. No self-approval, FREEZE,
or S1-closure claim is made by this BUILD.

## Addendum: Repair round 1 (R009-R1)

**Role:** REPAIR_WORKER (Claude) · **Base:** `c67904b` (Codex review of `e042d71`) · **Authority:** [independent review](CCMAI_RUNTIME_009_INDEPENDENT_REVIEW_2026-09-28.md) and the R009-R1 addendum in the [work order](../work_orders/CCMAI_RUNTIME_009.md). Paths touched: `backend/api/handlers/jobs.go`, `backend/api/handlers/job_config_admission_test.go`, this evidence file and R009 continuity files only.

### Source change (behavior-preserving)

- New `const testRunConversationLimit = 3`. `TestRunJob` passes it to
  `startTestRunJob(job, cfg, testRunConversationLimit)`; the launcher
  variable takes `limit int` and calls `analyzer.RunJobWithLimit(ctx, job, limit)`.
  The value reaching the analyzer is still 3. The real launcher's timeout,
  `jobCancelFuncs` registration/removal, panic recovery and log lines are
  unchanged. Admission order, the `job_start_failed` rejection, 202 bodies,
  config identity and trigger parameters are unchanged.

### Test change

- The stub launchers now reproduce the real worker's observable side effects
  (store a cancel handle under the job ID in `jobCancelFuncs` and insert one
  `job_runs` row for the job). Without this, "no cancel handle / no job run"
  would hold trivially because stubs create nothing. Fixture setup asserts
  the job starts with zero `job_runs` and no cancel handle. Cleanup removes
  the handle and the job's runs.
- `assertNoRunOrCancel` counts `job_runs WHERE job_id = <fixture job>`
  (error-checked) and looks up `jobCancelFuncs`. Both `assertNoTestRunStart`
  and `assertNoTriggerStart` call it, so it runs on all seven rejection
  checks: error and nil config for each endpoint, wrong tenant for each
  endpoint, and the real-`config.Load` invalid subtest (trigger).
- `assertWorkerStarted` (1 run row, cancel handle present) runs in the
  accepted test-run and trigger tests. This proves the detectors fire when
  a launch occurs, so their silence on rejection is meaningful.
- `TestTestRunJobPassesValidatedConfigToWorker` asserts the launch limit
  equals the literal `3`, not the constant, so changing the constant fails.

### Mutation check

Repaired `jobs.go` was saved to the session scratchpad. Two temporary
mutations were applied: `testRunConversationLimit = 4`, and a
`startTestRunJob(...)` call inside `TestRunJob`'s config-rejection branch
before the 500 response. Then
`test-backend.ps1 -Packages ./api/handlers -Run 'TestTestRunJob' -VerboseTests`
failed as expected:

- `TestTestRunJobConfigFailureStartsNoWorker` and
  `TestTestRunJobNilConfigIsNotAdmitted` failed with "test-run worker
  launched 1 times despite rejection".
- `TestTestRunJobPassesValidatedConfigToWorker` failed with "test-run
  launched with limit 4, want 3".
- `TestTestRunJobWrongTenantSkipsConfigLoad` passed, as expected, because
  the 404 path is before both mutations.

The backup was then restored, and `grep` confirmed limit `3` and no
launcher call in either rejection branch. Build and vet were clean.

Limitations of the mutation check:

- The launch-on-rejection mutation was stopped first by the existing
  launch-count `Fatalf`. The new side-effect assertions were not the first
  to fail. Their sensitivity is shown instead by `assertWorkerStarted`
  passing on accepted launches.
- The matching trigger-side mutation (a launch in `TriggerJob`'s rejection
  branch) was not run. The session's tool permission policy blocked that
  temporary edit. It was not attempted another way.
- The trigger rejection tests use the same `assertNoTriggerStart` →
  `assertNoRunOrCancel` path, and it passes on the unmutated source.

### Validation (repaired source)

1. `go build ./...`, `go vet ./...`, and
   `gofmt -l api/handlers/jobs.go api/handlers/job_config_admission_test.go`
   (from `backend/`) were clean.
2. The focused tests on disposable MySQL all passed:
   `scripts/test-backend.ps1 -Packages ./api/handlers -Run 'TestTestRunJob|TestTriggerJob|TestJobDispatchUsesRealConfigValidation' -VerboseTests`.
   That is 10 tests plus the `invalid`/`valid` subtests
   (`ok ... 6.890s`). The script removed its container and network.
3. The full backend suite on disposable MySQL (`scripts/test-backend.ps1`)
   passed on the restored source. All 13 packages were `ok`, including
   `api/handlers` (27.8s) and `engine` (13.0s). The script removed its
   container and network.
4. After the continuity sync, `IMPLEMENTATION_STATUS.json` and
   `ACTIVE_SESSION_STATE.json` parse as valid JSON. The CVF workspace doctor
   (`check_cvf_workspace_agent_enforcement.ps1 -ProjectPath <project root>`,
   including the governed catalog `-Check`) returned
   `RESULT: PASS (25/25 checks passed)`. `git diff --check` reported only
   the known `core.autocrlf` LF→CRLF notices and no whitespace errors.

The persistent Compose `ccma` stack was not touched. There was no provider
call, real channel sync, customer data, deployment or push. Status is
`REVIEW_PENDING` for Codex re-review. No FREEZE.
