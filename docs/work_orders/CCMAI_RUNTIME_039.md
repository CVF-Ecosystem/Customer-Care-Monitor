# CCMAI-RUNTIME-039 — Offline CLI historical scope-test maintenance

Status: REVIEW_PENDING

Date: 2026-10-03. Risk R1, test-only. [SPEC](../specs/PANCAKE_CLI_SCOPE_ASSERTION_R039_2026-10-03.md). Immutable dispatcher seed `CVF_SESSION/authority/CCMAI-RUNTIME-039.json` committed `7a390dc08e7958015b107e3a3e3b890369b82cf1` before activation/BUILD. Standing local orchestration authority; owner manually transfers to Claude. Codex authors/reviews, no automatic worker invocation.

## Bounded scope and sequence

Claude IMPLEMENTATION_WORKER / BUILD COMMIT_STEWARD; Codex independent REVIEWER. Only `backend/cmd/pancake-proof/main_test.go`: remove TestProductionAdapterFilesUnchangedInGit and its PH-07 comment. SPEC/order and governed session/status/evidence/catalog/index may change. Seed read-only. No other source/test/import/fixture changes. No baseline retarget, skip insertion or path-pruning repair.

1. Rehydrate manifest/policy/current state/memory/active handoff/status/index, SPEC/order/tranche/seed and shared repair learning. Doctor/local knowledge ingest; exclude generated index/cache. Record declaration and role transition WORK_ORDER -> BUILD **before** execution/edit; synchronize BUILD pointers/status/order/tranche and pass preflight.
2. Consolidate CS-01..03. In project-root cwd with GOPROXY=off, GOSUMDB=off, GOTOOLCHAIN=local, CGO_ENABLED=0, run `go -C backend test ./cmd/pancake-proof -run '^TestProductionAdapterFilesUnchangedInGit$' -count=1 -v`. Expected original failure must identify the historical protected-file comparison, not a compile/prerequisite failure or SKIP. Preserve its failed exit/output. Verify Git/baseline and named changed-path source. If baseline differs unexpectedly, return BUILD_BLOCKED.
3. Delete only named test/comment, preserve complete remainder byte-for-byte. Run `go -C backend test ./cmd/pancake-proof -count=1 -v`, `go -C backend build ./...`, `go -C backend vet ./...` from caches. No engine tests, runtime Analyzer, sockets/DB/Docker, credential/config discovery or downloads. CLI tests' in-memory adapter calls and synthetic temp files are permitted. No new tests or mutations needed.
4. Write `docs/reviews/PANCAKE_CLI_SCOPE_ASSERTION_R039_BUILD_2026-10-03.md`: seed/base/source hashes, exact deletion and six preserved test bodies, named original failure, full CLI counts/zero skips, command exits, protected path comparison from this seed to BUILD, all failures and NOT RUN. Protected paths: backend/channels, backend/engine, CLI main.go, backend/go.mod/go.sum, frontend, scripts, .github/workflows. Missing cached tools -> BUILD_BLOCKED; do not broaden source scope.
5. Catalog -Write/-Check, docs build, doctor, diff, default/PR/full-changed-set preflights and required gate unit tests before local commit. Return exact40-hex BUILD SHA/evidence as REVIEW_PENDING / REVIEW, all continuity synchronized. Independent Codex reviews CS-01..03; worker cannot self-approve. No push/merge/deployment/FREEZE.

Same-scope repairs use unchanged seed; third repair without independent new root cause stops REVIEW_COST_ESCALATION_REQUIRED. Unexplained original assertion failure, modified behavior coverage, new production diff, sanitizer/network effect or inaccurate test claims prevents acceptance.

## Current disposition

DISPATCH_READY / NOT_BUILT; original test present, no worker started. R034 offline and R038 static/compile-only acceptance unchanged. Expected named-test failure is source-derived/direct-Git evidence until Claude executes the authorized control. Global F02/live/MCP execution/governance/hosted/FREEZE remain separate.
