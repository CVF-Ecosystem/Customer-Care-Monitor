# CCMAI-RUNTIME-032 — F02-F Zalo local message traversal

Status: REVIEW_PENDING. Dispatched DISPATCH_READY; issued 2026-10-03 by Codex ORCHESTRATOR -> SPEC_AUTHOR -> WORK_ORDER_AUTHOR. Risk ceiling R2. Claude local BUILD complete 2026-10-03.

Authority: [SPEC](../specs/RUNTIME_ZALO_MESSAGE_COVERAGE_F02F_2026-10-03.md), owner resume and new immutable dispatcher seed `CVF_SESSION/authority/CCMAI-RUNTIME-032.json`. Seed must be committed at record baseCommit before BUILD. R031/R030/R024 and predecessors remain REVIEW_PASS / FREEZE_OPEN; global F02 stays OPEN.

## Roles and authorized scope

Claude IMPLEMENTATION_WORKER / COMMIT_STEWARD owns one local BUILD, returns REVIEW_PENDING with exact SHA. Codex owns planning/session/planning commits, then independent REVIEWER; no self-approval or reviewer product/test repair. This prepared order does not start an agent or authorize provider use.

- `backend/channels/zalo_oa.go`: FetchMessages and message-only validators/bounded request/redirect helper. Preserve FetchRecentConversations, shared doRequest/doRequestRaw/refreshToken, HealthCheck, credentials, OAuth and endpoint protocol. Helpers may reuse existing constants/safe error utilities unchanged.
- `backend/channels/zalo*_test.go`: new message probes and retained regressions; original assertions not weakened.
- `backend/engine/sync_zalo*_test.go`: actual adapter/engine/isolated MySQL message coverage and R024 fixture extensions. No engine product source or other platform tests.
- This SPEC/order/roadmap plus session/tranche/status/review and required catalog/index records under repository gate conventions. Immutable seed never worker-edited.

Allowed local effects: blocked synthetic HTTP/loopback media fixtures, disposable isolated MySQL/test tenant, backend tests/build/vet, docs/catalog/doctor/gates and bounded local commits. No unexpected network fallback, real channel/provider/credential use, persistent/customer DB, parent edit, shared/conversation/OAuth protocol behavior change, other-platform/UI/schema/permissions/workflow/tooling/dependency/timestamp migration, push/merge/deploy/FREEZE. Reference browsing is not live-channel authority.

## Required execution and return

1. Rehydrate manifest/policy/bootstrap fallback/current state/memory/active handoff/status/index, SPEC/order/seed/record and relevant shared learnings. Run doctor and knowledge ingest (generated index may stay in OS temp). Record role acknowledgment in active handoff before first source edit; synchronize BUILD across state/front marker/header/current prose/status/order/tranche/roadmap, prevalidate all writes and run scoped preflight. Project-root cwd; use go -C backend. Check every prerequisite exit. Missing compact bootstrap is nonblocking BOOTSTRAP_MIGRATION_PENDING; contradictory authority blocks at INTAKE.
2. Consolidate F02F-01..09/dependency/negative-observer plan. Preserve full-history/no-since-filter Zalo behavior. Protocol reference does not certify terminal/snapshot/string-link compatibility. If shared request/refresh/conversation/engine behavior must change, stop BUILD_BLOCKED with concrete authority amendment; no silent widening.
3. Implement message-only contract with exact validation, physical offsets to explicit empty, deterministic mapping/dedup/order, finite500 pages, context/error sanitation and message-only blocked redirects. Keep inherited refresh persistence/one retry; test each request inventory and attempted destination directly.
4. Run adapter positive/negative/mapping/control matrix, original-source detectors and finite applied mutations from SPEC. Restore byte identity in finally after each mutation; record competing guards, first survivors and actual assertion. A timeout/no-op/build error does not count as killed.
5. Run actual adapter+engine on disposable MySQL: exact >130 stored IDs and old history, success checkpoint/dispatch, late malformed/actual connection/HTTP500 errors separately, partial diagnostic discard/peer progress/checkpoint hold/zero dispatch, retry/replay, ownership/token and lease-exclusion regressions. Zero relevant DB skips; record optional unrelated skips separately. Never use synthetic provider output as AI-governance proof.
6. Full channels, focused R012-R016/R024/replay tests, full uncached backend with R019 five DB sentinels, build/vet; docs build/catalog/doctor/diff/default and PR-range preflight plus gate unit tests. Use explicit generous full-run timeout informed by R031; partitions remain separate evidence. Any disposable capacity override must be recorded. Inspect skip identities; retain all failed/inconclusive runs. Race unavailable: NOT RUN with reason.
7. Write `docs/reviews/RUNTIME_ZALO_MESSAGE_COVERAGE_F02F_BUILD_2026-10-03.md` with requirements/mapped-field/request/DB/status/checkpoint/dispatch matrix, old-source and mutation assertions/restoration, exact commands/exits/counts/skips, fixture isolation/cleanup inventory and limits. Preserve no live/global-F02/governance/hosted-CI claim. Cleanup via verified PowerShell literal paths; resource teardown/file cleanup/verification separate.
8. Synchronize REVIEW_PENDING everywhere; mark planning-era no-BUILD prose historical and preserve intended SPEC contract. No self-review/FREEZE. Return exact local BUILD SHA and changed set; buildCommit may be supplied in following documentation commit to avoid self-reference. Independent Codex checks seed author/timing, integrated source, all nine requirements, mutation sensitivity, regressions and claim limits.
9. Before each authorized commit run default preflight, `preflight --base origin/main --head HEAD`, explicit complete changed-set preflight and `python -B -m unittest discover -s scripts/tests -p "test_cvf_downstream_gate*.py"`. Do not suppress/rename failed whole-tree results. No source/tooling cleanup outside scope.

## Failure and disposition

Missing eligible full-history ID, unsafe request, short/malformed nil success, invalid duplicate discarded, wrong mapping/offset/order, unbounded traversal, diagnostic message storage, advanced checkpoint/after-sync on partial, weakened probe or in-scope failed check means CHANGES_REQUIRED/BUILD_BLOCKED. Same-scope repairs continue under existing authority; third same-root round requires REVIEW_COST_ESCALATION_REQUIRED. Boundary changes go to dispatcher; seed unchanged.

Review may accept local Zalo contract only. Actual offset snapshot stability, terminal/provider retention/permissions/string-link mapping, global F02, live provider governance and FREEZE remain separate. This planning checkpoint contains no product edit or BUILD/test result.
