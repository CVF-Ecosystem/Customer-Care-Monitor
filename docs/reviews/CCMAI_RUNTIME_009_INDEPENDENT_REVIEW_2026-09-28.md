# CCMAI-RUNTIME-009 independent REVIEW

**Reviewer:** Codex (independent of Claude's BUILD) · **Build commit:** `e042d71` · **Result:** `CHANGES_REQUIRED` · **Risk:** R2 · **Date:** 2026-09-28.

## Scope and accepted evidence

I compared the changed set with [SPEC](../specs/RUNTIME_JOB_CONFIG_ADMISSION_S1_2026-09-28.md) and [work order](../work_orders/CCMAI_RUNTIME_009.md). Product edits stay within `jobs.go` and a focused handler test file. Both handlers load configuration after tenant-scoped lookup and before launch/202; error or nil config returns generic `500 job_start_failed`; the successful path passes the same config pointer to a worker without reloading. Source preserves the 202 response bodies and existing trigger parameter resolution. No real provider/governance behavior is claimed.

Independent checks: `go build ./...` and `go vet ./...` passed from `backend/`. I reran the 10 focused admission tests using `scripts/test-backend.ps1 -Packages ./api/handlers -Run 'TestTestRunJob|TestTriggerJob|TestJobDispatchUsesRealConfigValidation' -VerboseTests` on a separate disposable MySQL `CCMA`; the command exited 0, the handler package passed, and the script removed its test database container/network. Workspace doctor passed 25/25. The BUILD record reports the full backend suite passing; I did not repeat it because the remaining finding is missing acceptance coverage, not an observed backend failure.

## Finding R009-R1 — required side-effect and limit proof is incomplete

The SPEC's acceptance requires focused tests to prove rejected requests create no job run or cancel function and that test-run's limit remains 3 at launch. The new rejection tests assert only that stubbed launcher counters stay at zero. The fixture does not count `job_runs` for its job or inspect `jobCancelFuncs` after rejection. Source inspection supports the intended early return, but the required direct regression assertions are absent. The accepted test-run test stubs `startTestRunJob(job, cfg)`; the limit `3` remains inside the real launcher and is never observed by that test. Thus the test cannot detect a future change to that limit.

**Repair acceptance, same scope:** add focused assertions for both endpoints' error and nil-config rejection paths that no `job_runs` row and no `jobCancelFuncs` entry exist for the fixture job. Make the accepted test-run path expose and assert the launch limit `3` through a small private seam or equivalent testable helper without invoking `engine.NewAnalyzer` or a provider. Preserve the existing successful 202 bodies, config identity, timeout/cancel semantics and test isolation. Re-run focused and full backend tests on disposable MySQL, build, vet, catalog/doctor and diff check; append exact results to BUILD evidence. Keep edits to `backend/api/handlers/jobs.go`, focused tests and the existing R009 evidence/continuity files. Return one local repair commit as `REVIEW_PENDING` for Codex re-review.

## Disposition

R009 remains `REVIEW_PENDING / CHANGES_REQUIRED`; no FREEZE. R001–R008 REVIEW PASS and S1 IN_PROGRESS are unchanged. No provider call, real channel sync, customer data, persistent Compose DB change, deployment or push was performed by this review.
