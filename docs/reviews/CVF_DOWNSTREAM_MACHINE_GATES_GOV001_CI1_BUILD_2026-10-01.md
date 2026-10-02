# CCMAI-GOV-001 GOV1-CI1 — hosted governance-job repair (REPAIR_WORKER evidence)

**Status:** REVIEW_PENDING for Codex. **Role:** REPAIR_WORKER and COMMIT_STEWARD (Claude); reviewer Codex stays independent. **Risk:** R2, unchanged. **Authority:** [hosted CI review](CCMAI_PR_001_HOSTED_CI_REVIEW_2026-10-01.md) and the GOV1-CI1 addendum in the [work order](../work_orders/CCMAI_GOV_001.md). Repair base `0cd5f52`; gate source, tests and the authority seed are unchanged.

## Root cause and change

`governance.yml` runs the gate self-tests and then the preflight in one checkout. Python wrote `scripts/__pycache__/*.pyc` and `scripts/tests/__pycache__/*.pyc`; the gate correctly counts untracked paths and rejected them. The only implementation change is `PYTHONDONTWRITEBYTECODE: '1'` as job-level `env` on both jobs of `.github/workflows/governance.yml`, so it covers every step and every Python subprocess started by the tests. The gate's untracked-file detection and the allowed paths are not changed.

## Clean-checkout-style proof

Fresh `git clone` of the repair base at `0cd5f52`, the edited workflow copied in, `origin/main` pinned to the PR base `7481196`, the exact workflow test command first, then the exact PR-range preflight (`--skip-catalog --base origin/main --head HEAD`). PyYAML 6.0.3 was present locally (the runner pins 6.0.2).

| Sequence | Tests | `.pyc` files in repo | Preflight |
|---|---|---|---|
| A. without the variable (reproduces the hosted failure) | 46 OK | 2 | **FAIL**, exit 1: `tranche` rejects both `__pycache__/*.pyc` paths |
| B. `PYTHONDONTWRITEBYTECODE=1` | 46 OK | **0** | **PASS**, exit 0: 6/6 executed gates passed, catalog printed `SKIP` |
| C. B plus untracked `backend/rogue.go` | n/a | n/a | **FAIL**, exit 1: `changed path outside the authorized allowedPaths: backend/rogue.go` |

## Other checks

Workspace doctor 25/25 PASS; catalog `-Check` PASS (inside the repo preflight and the doctor); full gate suite PASS; `git diff --check` clean. Local Python is 3.13, the runner uses 3.12.

## Claim limits

Only the hosted rerun at the new PR head proves the Ubuntu job; the variable is set at job level and the clean-checkout run above is local. The Windows catalog job also carries the variable but already passed at `19fdc4e`. Python could still write `.pyc` if a future step runs a tool that ignores the variable; the gate would then fail loudly, as intended. No push, merge, deployment, provider/channel call, parent-CVF work or FREEZE was performed. R020 stays parked; F08 closure still waits for the repaired PR head.
