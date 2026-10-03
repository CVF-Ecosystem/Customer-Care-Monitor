# CCMAI-RUNTIME-036 — Frontend setup-status loading and recovery

Status: REVIEW_PASS

Date: 2026-10-03. Risk ceiling R2. [SPEC](../specs/SETUP_STATUS_RECOVERY_R036_2026-10-03.md). Immutable dispatcher seed `CVF_SESSION/authority/CCMAI-RUNTIME-036.json` committed 5a044ae82d11cdf59cdf28d0a8fe9f9410957bb3 before activation/BUILD. Standing local orchestration delegation and owner continuation authorize this bounded order; owner manually transfers to Claude, no automatic invocation.

## Assignment and scope

Claude IMPLEMENTATION_WORKER / BUILD COMMIT_STEWARD; Codex ORCHESTRATOR / SPEC_AUTHOR / WORK_ORDER_AUTHOR / planning COMMIT_STEWARD, then independent REVIEWER. Codex does not implement product source.

Exact scope: `frontend/src/router/index.ts` (setup status and unavailable route only), optional new `frontend/src/router/setupStatus.ts`, `frontend/src/App.vue` (loading/layout/profile sequencing), new `frontend/src/views/SetupStatusUnavailable.vue`, `frontend/src/i18n/vi.ts` and `en.ts` (status/retry keys only), `frontend/src/__tests__/setup-stale-token.spec.ts` (preserve AUTH-001 positives; replace obsolete failed-status expectation and compatible helpers), new `frontend/src/__tests__/setup-status-unavailable.spec.ts`. SPEC/order and continuity/status/reviews/catalog/index follow existing gates. Seed immutable.

Allowed: synthetic mocked UI/navigation tests, cached local frontend typecheck/build, docs/gates, bounded local commits. Excluded: credentials/config/.env, provider/channel/external network/downloads, DB/customer data, backend/auth store/Setup view/permission policy/global API interceptor/cookie changes, dependencies/lockfiles/workflows/gate/parent edits, actual MCP queue/Analyzer, push/merge/deployment/new FREEZE. Missing cached prerequisites -> BUILD_BLOCKED, no downloads or invented evidence.

## Execution and hand-back

1. Rehydrate canonical manifest/policy/bootstrap fallback/state/memory/active handoff/status/index, SPEC/order/record/seed and shared repair learning. Doctor and local knowledge ingest; exclude generated index/caches. Before product edits record worker declaration/acknowledgment, prevalidate and synchronize BUILD across continuity/order/tranche/status, then pass preflight. Failed gate stops editing; never backdate acknowledgment.
2. Consolidate SS-01..06 before implementation, including App initially mounted unresolved/unavailable and post-retry profile loading. Implement strict confirmation, finite status-only timeout, explicit single-flight Retry, localized loading/error and fixed local recovery. Preserve confirmed AUTH-001 behavior and existing permission rules.
3. Run focused new tests and AUTH-001, complete frontend suite without unexplained failures/skips, forced typecheck and frontend build from cached dependencies. Root cwd: `npm --prefix frontend test`; `node frontend/node_modules/vue-tsc/bin/vue-tsc.js -b frontend/tsconfig.json --force`; `npm --prefix frontend run build`. Deferred operations deterministic; use bounded microtask-loop detectors, not timer-only loop proof. Mocked UI evidence makes no governance assertion.
4. Demonstrate baseline controls and four SS-06 applied mutations in isolated copies/files. Record named assertion kills, survivors/inconclusive attempts; restore exact bytes and rerun final focused/full tests as warranted. Do not stage mutant source or claim universal absence of effects from bounded probes.
5. Write `docs/reviews/SETUP_STATUS_RECOVERY_R036_BUILD_2026-10-03.md`: exact source/seed identities and changed set, SS matrix, mounted vi/en observations, request/token/store counts, commands/exits/tests/skips, timeout/single-flight behavior, baseline/mutation failures, restoration hashes, all failed history and NOT RUN. Update SPEC implementation truth and status/catalog without rewriting predecessor dispositions or FREEZE.
6. Before commits run default/PR/full changed-set preflights, `python -B -m unittest discover -s scripts/tests -p "test_cvf_downstream_gate*.py"`, catalog generation/check, docs build, doctor and diff checks. Commit bounded BUILD locally; record exact 40-hex buildCommit through documentation hand-back if needed. Synchronize REVIEW_PENDING / REVIEW and return SHA/evidence/changed set to Codex. No self-approval or new FREEZE.

UI bypass, premature profile load, failure-based token clearing, request loop, payload coercion, failing required checks or unauthorized edit prevents acceptance. Same-scope repairs retain this seed; round three without independent new root cause requires REVIEW_COST_ESCALATION_REQUIRED. Backend/interceptor/auth/network expansion is a separate objective.

## Dispatch truth (historical, as written at dispatch)

*Historical: the paragraph below describes the state at dispatch commit dea891c. Current: Claude BUILD is REVIEW_PENDING; see the BUILD record.*

DISPATCH_READY / NOT_BUILT: no Claude worker started in this planning turn. R035/R034 local REVIEW_PASS / FREEZE_OPEN and R033 local-message FREEZE remain source-specific history. Actual MCP execution and live Pancake inputs/authority remain separate. No frontend test result, backend security, runtime AI/CVF governance, hosted readiness or new FREEZE claim.

## Independent acceptance (2026-10-03)

[Exact-BUILD independent review](../reviews/CCMAI_RUNTIME_036_INDEPENDENT_REVIEW_2026-10-03.md): REVIEW_PASS / FREEZE_OPEN for synthetic SS-01..06 only. Frontend299/299, typecheck/build, restored38 PASS; four mutations killed; original failed-status control fails then restored passes. Documentation-only current labels/abbreviated vi hash settled by reviewer-owned synchronization; no product/test/seed change. All failed history and happy-dom/fake-timer/cookie/multi-tab/live/backend limitations retained. No new worker BUILD or FREEZE.
