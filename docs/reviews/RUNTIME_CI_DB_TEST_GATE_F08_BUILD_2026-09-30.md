# CCMAI-RUNTIME-019 / F08 BUILD evidence — backend CI DB-test gate

**Date:** 2026-09-30 · **Phase:** BUILD → REVIEW_PENDING · **Risk:** R2 · **Role:** IMPLEMENTATION_WORKER + COMMIT_STEWARD (Claude); independent REVIEWER: Codex
**Authority:** [SPEC](../specs/RUNTIME_CI_DB_TEST_GATE_F08_2026-09-30.md), [work order](../work_orders/CCMAI_RUNTIME_019.md), [F08 finding](CCMAI_F01_F08_LOCAL_SOURCE_REVIEW_2026-09-30.md).
Planning-commit parent: `d194ed0` (HEAD at BUILD start `2650832`, worktree clean). CI configuration and a stdlib gate helper only; no product Go/TypeScript source, DB test assertion, Compose resource, persistent DB, secret, provider/channel call, push or FREEZE.

## Changed set

| Path | Purpose |
|---|---|
| `.github/workflows/backend.yml` | Disposable MySQL 8 with readiness check, full DB-backed suite with `-json -count=1 -p 1`, gate helper, then `go build ./...`; path filters extended to the helper and its tests. |
| `scripts/ci_db_test_gate.py` | Standard-library parser of `go test -json`; fails unless the required DB sentinels PASS and no DB-unavailable skip occurs. |
| `scripts/tests/test_ci_db_test_gate.py` | 9 unit tests with synthetic logs (positive and negative). |
| this evidence file, active handoff, active state, session memory | Records and continuity. |

## Workflow design

1. `python3 -m unittest discover -s scripts/tests` checks the gate itself first.
2. `docker run mysql:8.0 --log-bin-trust-function-creators=1` (needed by the trigger-based failure tests; GitHub service containers cannot pass server arguments) with schema `CCMA`, user `ccma` and a per-run `openssl rand -hex 16` password, masked with `::add-mask::`, bound to `127.0.0.1:3306`. Readiness loop (60 × 2 s) requires the **application user** to log in and read `log_bin_trust_function_creators = 1`; otherwise the step fails. `TEST_DB_DSN` is written to `GITHUB_ENV` only in this job; nothing echoes it and no artifact is uploaded.
3. `go test ./... -json -count=1 -p 1 > $RUNNER_TEMP/gotest.json`; the exit code is captured (no `-e`), the gate always runs, and the step fails if either the test or the gate is nonzero. `test -n "$TEST_DB_DSN"` fails fast when unset.
4. `go build ./...` (preserved). `docker rm -f` runs with `if: always()`.

## Required sentinels (one per DB-owning package; each helper `t.Skip`s without a reachable DB)

| Package | Test | Source |
|---|---|---|
| `backend/db` | `TestChannelSyncRunIDColumnOnFreshSchema` | `db/sync_run_migration_test.go:55` |
| `backend/api/handlers` | `TestDeleteChannelRemovesResultsAndSnapshotsTogether` | `api/handlers/channels_test.go:35` |
| `backend/engine` | `TestSingleAndBatchShareSnapshotContract` | `engine/snapshot_db_test.go:191` |
| `backend/cli` | `TestApplyPrunePlanCleansOrphanSnapshotsKeepsReferenced` | `cli/prune_results_test.go:204` |
| `backend/storagecfg` | `TestTatS3VanDocDuocThongTinDaLuu` | `storagecfg/loader_test.go:51` |

Identities are package-qualified (`github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/<pkg>`). The gate also requires each sentinel package to reach a terminal package `pass` (truncated logs fail) and fails on **any** test whose skip output names `TEST_DB_DSN` or "khong ket noi duoc DB test", not only sentinels. Other skips are counted and listed but do not fail. The report prints only names and counts, never raw output.

## Positive run (disposable MySQL, isolated Docker network, no host port)

Equivalent of `scripts/test-backend.ps1` with `-json` (that script cannot emit JSON): `mysql:8.0` + `golang:1.26-alpine`, read-only host module cache, `GOPROXY=off`, same flags.

- `go test ./... -json -count=1 -p 1`: exit 0; `go build ./...`: exit 0.
- Gate: 234,247 events; tests pass=454 fail=0 skip=2; all five sentinels `PASS`; db-unavailable skips 0; other skips 2 (`ai/pricing.TestFetchThatTuNguonNgoai` live fetch and `storage.TestNoiCatS3` optional S3) — both are opt-in and correctly not counted as DB coverage. `GATE PASSED`, exit 0.
- Cleanup: container and network removed by trap; `docker ps -a` / `docker network ls` show no `ccma-f08*` resources. The persistent Compose database was not used.

## Negative checks

Unit tests (`python -W error -m unittest discover -s scripts/tests`, 9/9 OK): positive; required test SKIP with package PASS → fail; required test missing → fail; required test failed → fail; truncated package result → fail; empty/non-JSON log → fail; an unrelated DB-unavailable skip → fail while an optional S3 skip → pass; raw connection text is never echoed; CLI exit codes 0 / 1 / 1 (good / skipped / unreadable log).

Real reproduction of F08 (five sentinel tests only, container with `--network none`):

| TEST_DB_DSN | `go test` exit | sentinels | Gate |
|---|---|---|---|
| unset | **0** (the false green the finding describes) | 5 × SKIP | **exit 1**, 5 DB-unavailable skips |
| unreachable (`127.0.0.1:1`) | **0** | 5 × SKIP | **exit 1**, 5 DB-unavailable skips |

Workflow shell: all five `run` blocks pass `bash -n`; YAML parses; triggers cover `backend/**`, the workflow, the helper and its tests. Unreachable/never-ready MySQL makes the readiness step `exit 1` before any test step; that branch was reviewed but not executed (Actions-only).

## Other gates

`git diff --check` clean (CRLF warnings only); catalog `-Check` PASS; workspace doctor 25/25. No temporary mutation of tracked files was made; `__pycache__` directories were removed, so the tree holds only the paths above.

## Claim limits

- Proven: local disposable-MySQL positive run, real no-DB false-green reproduction turned into a gate failure, and parser negatives.
- **Not proven:** an actual GitHub Actions run (needs an authorized push); the `docker run` port binding, masking and `GITHUB_ENV` behavior on GitHub-hosted runners. F08 is not claimed closed for public CI until a workflow run passes.
- The sentinel list is a coverage floor, not proof every DB test ran; the gate additionally rejects any DB-unavailable skip. Adding a new DB-owning package requires adding its sentinel (documented in the helper).
- No F01–F07 fix, real-channel completeness, provider call or CVF-governance claim.

## Disposition

`REVIEW_PENDING` for Codex independent REVIEW. Claude does not self-approve; no push, deployment or FREEZE.
