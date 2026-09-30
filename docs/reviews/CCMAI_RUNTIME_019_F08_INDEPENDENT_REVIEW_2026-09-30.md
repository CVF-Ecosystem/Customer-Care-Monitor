# CCMAI-RUNTIME-019 / F08 — independent BUILD review

Date: 2026-09-30. Reviewer: Codex (independent of Claude IMPLEMENTATION_WORKER). BUILD commit: `bc7d0676d42da2f21196d25f9ade86ffaf73461f`, parent `265083238022a487975c58beb663d7d27ad8daf6`. Disposition: **CHANGES_REQUIRED / REVIEW_OPEN**. F08 remains OPEN; no FREEZE.

## Authority and changed set

Reviewed [SPEC](../specs/RUNTIME_CI_DB_TEST_GATE_F08_2026-09-30.md), [work order](../work_orders/CCMAI_RUNTIME_019.md), exact commit and [BUILD evidence](RUNTIME_CI_DB_TEST_GATE_F08_BUILD_2026-09-30.md). The commit changes only the permitted CI workflow, new standard-library gate/helper tests and evidence/continuity. `git diff --exit-code 2650832 bc7d067 -- backend frontend docker-compose.yml` returned 0. No provider call, product-source repair, push or deployment is claimed.

Continuity drift was reported at INTAKE: active state, handoff, BUILD evidence and owner message said REVIEW_PENDING while the session-memory front pointer and `IMPLEMENTATION_STATUS.currentPhase` still said WORK_ORDER. Only those two stale summary fields were aligned, then continuity was reread before source review. Doctor: PASS 25/25.

## Accepted observations

- `.github/workflows/backend.yml` provisions a disposable MySQL 8 container, checks the application user's connection and `log_bin_trust_function_creators`, passes a generated CI-only DSN through `GITHUB_ENV`, runs `go test ./... -json -count=1 -p 1` and the gate, preserves `go build ./...`, and removes the container with `if: always()`. Path filters include the new helper/tests. The source contains no Alibaba key or persistent Compose DB call.
- Five required package-qualified sentinel names match DB-dependent tests in `db`, `api/handlers`, `engine`, `cli` and `storagecfg`. A skipped or missing sentinel and a failed sentinel/package are rejected by the helper. Codex independently reran the nine parser unit tests (9/9 PASS); Claude's positive disposable-MySQL full run (454 tests passed, two unrelated optional skips, all sentinels PASS) is recorded in BUILD evidence but was not independently repeated in this review.
- The worker correctly disclosed that the actual GitHub-hosted readiness, port binding, mask and `GITHUB_ENV` behavior are unverified. Local source inspection and shell syntax alone are not a public CI run. This limitation does not authorize a push or an F08 closure claim.

## Blocking finding R019-R1 — gate can pass an incomplete or DB-unavailable log

1. `scripts/ci_db_test_gate.py:48-51` counts malformed/non-JSON lines but ignores them in `problems`; `:113` returns success when all sentinels passed. Independent negative probe: `evaluate(good_log() + ['not-json'])` returned `True` and reported `non-JSON lines ignored: 1`. The SPEC requires absent/truncated test output to fail. A log with valid sentinel events plus a damaged trailing line cannot be called complete.
2. `scripts/ci_db_test_gate.py:31,83` recognizes only `TEST_DB_DSN` and `khong ket noi duoc DB test` as DB skips. Existing `backend/engine/integration_test.go:93` uses `Skipping integration test - DB not available: ...`. Independent probe appended that package-qualified skipped test to a passing sentinel log: `evaluate(...)` returned `True`, classified it as `other skip`, and printed `GATE PASSED`. This violates the SPEC requirement to reject **any** DB-unavailable skip, including a transient outage after sentinels have passed.

Both probes use synthetic Go JSON events as permitted by the negative-gate acceptance contract. They test the committed helper directly, not a modified source file. The existing nine tests do not cover these cases. Repair must reject malformed records, including a damaged trailing line, and recognize the English DB-unavailable skip in current source. Add regression tests that fail before the repair and pass afterward; keep optional S3/live-fetch skips allowed. Do not echo raw output or DSNs.

## Non-blocking observations and limits

- `scripts/tests/test_ci_db_test_gate.py:99-100` writes two files without context managers. `python -W error -m unittest discover -s scripts/tests -v` exited 0 but emitted ignored `ResourceWarning` messages for both unclosed files. The repair may close these handles within the already allowed test file; this is not the F08 blocker.
- The worker has not run the workflow's never-ready MySQL branch on a GitHub runner. Source shows readiness failure exits before the test step; treat runner behavior as pending an actual authorized workflow run. The BUILD's local real no-DB tests demonstrate `go test` can exit 0 while the gate exits 1 for five sentinel skips, but they do not exercise the hosted readiness step.

## Review checks and next move

Commands: `git diff-tree --no-commit-id --name-status -r bc7d067`; `git diff --exit-code 2650832 bc7d067 -- backend frontend docker-compose.yml`; `python -W error -m unittest discover -s scripts/tests -v` (9/9 PASS with ResourceWarnings); two direct `python -B` parser probes above (both unexpected PASS); catalog/doctor source checks. No disposable DB or provider call was run independently after the blocking parser defect was found.

Claude is next REPAIR_WORKER under the same R2 objective and local-commit boundary. Repair only `scripts/ci_db_test_gate.py`, `scripts/tests/test_ci_db_test_gate.py`, BUILD evidence and continuity, plus workflow only if a demonstrated parser integration requirement needs it. Return one local repair commit as REVIEW_PENDING. Codex re-reviews. No push, deployment, product source, persistent DB, provider call or FREEZE.
