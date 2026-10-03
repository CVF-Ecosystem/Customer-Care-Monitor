# CCMAI-RUNTIME-035 — Truthful unavailable MCP trigger response

Status: DISPATCH_READY

Date: 2026-10-03 (Asia/Saigon). Risk ceiling R2. [SPEC](../specs/MCP_TRIGGER_JOB_TRUTH_R035_2026-10-03.md). Dispatcher seed `CVF_SESSION/authority/CCMAI-RUNTIME-035.json` committed at `4b714a35bc6f49abe508d3ec4f32504e500054ba` before activation/BUILD. Standing local orchestration delegation and owner's reaffirmed ORCHESTRATOR/REVIEWER assignment authorize this bounded order. Owner manually transfers it to Claude; no automatic agent invocation.

## Assignment and allowed scope

Claude IMPLEMENTATION_WORKER / BUILD COMMIT_STEWARD; Codex ORCHESTRATOR / SPEC_AUTHOR / WORK_ORDER_AUTHOR / planning COMMIT_STEWARD, then independent REVIEWER. Codex authors/reviews and does not implement product source. Local response correction only; no queue/Analyzer implementation or live prerequisite.

Authorized source paths: `backend/mcp/handlers.go` (toolTriggerJob only), `backend/mcp/tools.go` (trigger description only), `backend/mcp/permission_admission_test.go` (trigger-specific expectations/helpers while preserving other contracts), new `backend/mcp/trigger_contract_test.go`. SPEC/order may document necessary clarifications/current implementation; continuity/status/review/catalog/index records follow existing gate conventions. Original authority seed is immutable.

Allowed effects: local build/tests/vet/docs/gates, synthetic disposable loopback MySQL tests, bounded local commits. Prohibited: real credential/config/.env inspection; provider/channel/external network; persistent/customer DB; actual queue/analyzer/sync/notification effects; permission policy/OAuth/server/engine/HTTP/frontend/schema/dependency/workflow/gate/parent changes; push/merge/deployment/FREEZE. No downloading absent prerequisites. Application permission tests are not CVF governance proof.

## Execution sequence

1. Rehydrate canonical manifest/policy/bootstrap fallback/state/memory/active handoff/status/index and R035 SPEC/order/record/seed, relevant shared repair learning. Run doctor and local knowledge ingest; keep generated index/cache files out of commits. Record worker acknowledgment before source edits. Prevalidate and synchronize state/front marker/handoff/status/order/tranche to BUILD, then run preflight; failed preflight stops editing.
2. Consolidate MT-01..06 dependencies before implementation. Return the fixed unavailable tool error after the existing authorized tenant lookup, preserve generic lookup failure and no-dispatch behavior, correct the published description. Never weaken R021 admission or make arbitrary tool errors count as accepted positive cases.
3. Add direct/mounted contract tests, forced read error, other-tenant job within an admitted tenant, rights/member/owner/admin regressions, no-write/no-run observations and tools/list assertion. Use actual disposable MySQL; no DB skips for required evidence. Demonstrate original-source detectors and four applied mutations with byte restoration, retaining failures/inconclusive attempts.
4. Run complete MCP/new focused tests, build/vet, race if locally available; required default, PR-range and explicit full changed-set preflights, `python -B -m unittest discover -s scripts/tests -p "test_cvf_downstream_gate*.py"`, catalog, doctor and diff checks before commits. Use GOPROXY=off/GOTOOLCHAIN=local and root cwd. Do not run the application or use persistent DB.
5. Create `docs/reviews/MCP_TRIGGER_JOB_TRUTH_R035_BUILD_2026-10-03.md` with source/seed identity, exact changed set, MT matrix, direct/mounted request/response examples (synthetic), observed lookup/write/state/outbound effects and limits, old-source/mutation failures, restoration hashes, commands/exits/test counts/skip identities and unavailable checks. Update SPEC current implementation and IMPLEMENTATION_STATUS only from actual evidence. Preserve prior tranche dispositions and failed history.
6. Commit bounded BUILD locally, then record its exact 40-hex SHA in tranche `buildCommit` and evidence using a documentation hand-back commit if needed. Synchronize REVIEW_PENDING / REVIEW across continuity/order/tranche. Return exact SHA, changed set and evidence to Codex. No self-approval, new FREEZE or push.

Scope mismatch, changed authorization policy, generic accepted-error test weakening, disclosure, actual dispatch, missing required DB evidence or failing checks prevents REVIEW_PASS. Missing local prerequisites returns BUILD_BLOCKED with concrete dependency. Same-scope repair follows reviewer findings under this seed; round three without independent new root cause requires REVIEW_COST_ESCALATION_REQUIRED. New actual execution is separately scoped, never improvised here.

## Dispatch boundary

This is ready for owner transfer to Claude, with no additional routine approval request. Worker BUILD has not started. R034 remains offline REVIEW_PASS / FREEZE_OPEN; Pancake live packet remains PREPARED_NOT_DISPATCHED / EXTERNAL_INPUT_REQUIRED. R033/local-message FREEZE stays unchanged. No public/provider governance, hosted readiness or deployment claim.
