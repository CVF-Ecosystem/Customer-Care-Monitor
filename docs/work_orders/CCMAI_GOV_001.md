# CCMAI-GOV-001 — materialize applicable downstream CVF machine gates

Status: DISPATCH_READY. Issued 2026-09-30 by Codex (ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR). Planning base: `df0c4ccf875fb0e96258fe4f85106a102908f2cb`. Risk ceiling R2. Authority: [SPEC](../specs/CVF_DOWNSTREAM_MACHINE_GATES_2026-09-30.md) and [applicability decision](../decisions/CVF_DOWNSTREAM_GATE_APPLICABILITY_2026-09-30.md). This work order implements the owner's downstream machine-gate request before the authorized PR/F08 public-run step.

## Assignment and return

Claude is IMPLEMENTATION_WORKER and COMMIT_STEWARD for a local BUILD commit; Codex is independent REVIEWER. Rehydrate canonical continuity, run the workspace doctor, and record `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` in the active handoff before editing. Implement the SPEC with a truthful applicability matrix; return exact commit SHA, changed paths, command results, negative fixture results and claim limits as `REVIEW_PENDING`. No self-approval or FREEZE. If a core gate cannot be materialized truthfully, document the incompatibility and implement its bounded downstream equivalent; do not silently omit a required control.

## Allowed scope

- New `scripts/cvf_downstream_gate*` and `scripts/tests/test_cvf_downstream_gate*` files plus small structured contract/fixture under `.cvf/` or `CVF_SESSION/` needed by the SPEC. Add only necessary dependencies; prefer Python standard library and existing PowerShell catalog tool.
- `.github/workflows/` for governance and frontend PR checks and exact trigger adjustments; do not weaken backend MySQL or docs gates. `frontend/package.json` only if necessary to expose existing test/build commands, not application source.
- `CVF_SESSION_MEMORY.md`, active state/handoff, `IMPLEMENTATION_STATUS.json`, `AGENTS.md` only to add the machine-readable pointer/contract and tool invocation instructions while preserving existing governance policy. `.cvf/manifest.json` and `.cvf/policy.json` are read-only authority except a narrowly required new pointer field, which must be explained in BUILD evidence.
- This order/SPEC/decision, one BUILD evidence record under `docs/reviews/`, catalog registry/generated index and module catalog if registered artifacts change, and roadmap status.
- Read-only: product source (`backend/`, `frontend/src/`), DB migrations/data, Compose, local `.env`, provider code and CVF core. No API key, provider/channel call or production data. If a required gate needs product changes or external authority, stop with a bounded blocker.

## Execution and failure conditions

1. Inventory each applicable gate and its hook/CI invocation. Use the SPEC's contract and actual project schemas. Make the two observed continuity drifts fail before fixing/migrating front markers. Establish a passing baseline after migration.
2. Add focused positive and negative tests for continuity, prospective packet/role/claim checks, secret hygiene and workflow trigger coverage. Assert the command's exit status and safe diagnostics. Avoid test logic that merely repeats checker implementation.
3. Run local governance preflight, test suite for the gate, catalog `-Check`, frontend test/build and docs build where available, `git diff --check`, and workspace doctor. Keep R019 backend suite evidence inherited; do not rerun full MySQL solely for governance file changes unless backend workflow changed. If backend workflow changes, run its parser tests and local disposable-MySQL suite, or report a blocker.
4. Record exact files, gate statuses, negative mutations and limits in BUILD evidence. Ensure CI can run from a fresh checkout; no sibling core assumption. One local commit only. If any required gate is unimplemented or tests fail, return `BUILD_BLOCKED`, not `REVIEW_PENDING`.

## Review and PR boundary

Codex reviews the exact diff and reruns selected negative/positive checks. Only after `REVIEW_PASS` may Codex create the already authorized GitHub PR. The PR is the use case for F08; local results cannot close F08. Codex separately inspects actual Actions run SHA, five DB sentinels, job conclusion and credential/log behavior before updating F08 evidence. R020 stays parked, F01-B and F02–F07 remain open. No merge, deployment or FREEZE under this order.
