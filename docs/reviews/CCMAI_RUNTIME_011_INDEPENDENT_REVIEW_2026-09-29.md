# CCMAI-RUNTIME-011 independent review

**Reviewer:** Codex, independent of Claude's BUILD · **Build commit:** `44eead4` · **Date:** 2026-09-29 · **Disposition:** `CHANGES_REQUIRED` at REVIEW; FREEZE open.

## Scope and checks

I compared the change with the [SPEC](../specs/RUNTIME_AGENT_CONFIG_ADMISSION_S1_2026-09-29.md) and [work order](../work_orders/CCMAI_RUNTIME_011.md). Product edits are limited to `backend/api/handlers/agents.go` and a focused handler test. `AgentRun` retains request binding and tenant authorization before config admission; an authorized unknown name returns 404 before loading. Known names now reject load error or nil config with generic 500 before either dispatcher, and the accepted path passes the same config pointer to the existing synchronous handler. The log omits loader error and request params. Source behavior meets the bounded admission design.

I independently ran `powershell -ExecutionPolicy Bypass -File ../scripts/test-backend.ps1 -Packages ./api/handlers -Run 'TestAgentRun'` from `backend/`. Exit 0; `api/handlers` passed in 4.134s, and the script removed its disposable MySQL container and network. Claude's BUILD evidence records the full backend suite (13 packages), build, vet, catalog and doctor passing. The six error/nil tests, accepted-path detector and reported 6/6 guard mutation give useful evidence for the guard.

Before review, I reported `BLOCKED_CONTINUITY_DRIFT`: the handoff Current State header still said WORK_ORDER although active state, memory, the handoff's R011 BUILD section and the owner's handoff said REVIEW_PENDING. I corrected only that stale header, recorded the correction and re-read continuity. This was not a review acceptance decision.

## Finding R011-R1 — two required rejection observers are missing

The work order requires direct assertions of no channel/**job-run** write and no outbound request on each of the three agents' error and nil-config paths. The new test checks zero dispatcher calls and unchanged channel `last_sync_status`, but does not query `job_runs` before/after rejection or install an outbound recorder. The accepted-path detector proves its channel-write observer is live, but cannot validate the unimplemented job-run and outbound assertions. Source inspection makes these effects unlikely before dispatch; the direct regression proof explicitly required by this R2 work order remains incomplete.

**Repair acceptance:** in the existing focused test file, add tenant-scoped job-run before/after assertions for all six rejection cases and a recording outbound transport or equivalent direct request observer that asserts zero calls. Show the observer can detect a synthetic call on an accepted stub path without contacting a real endpoint. Keep the accepted config identity, response and no-dispatch checks. Use only synthetic fixtures and disposable MySQL; restore any package-global seams/transport and avoid parallel tests.

## Finding R011-R2 — BUILD base commit is incorrect

The BUILD evidence names `b0e39b1` as its base, but `git rev-parse 44eead4^` is `10af2c67cfa5551b002d9bc7d7f2ba11cbce263e`, the committed R011 plan. Correct the evidence metadata. This is a provenance correction, with no source behavior change.

## Disposition

Return a single same-scope repair commit as `REVIEW_PENDING` for Codex re-review. Limit edits to `backend/api/handlers/agents.go` only if a small test seam is needed, the focused handler test, BUILD evidence, work-order repair addendum and continuity/status records. Re-run focused and full backend tests on disposable MySQL, build, vet, catalog, doctor and diff check. No push, real provider/channel call, governance claim or FREEZE. S1 remains IN_PROGRESS.
