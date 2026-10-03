# CCMAI-RUNTIME-041 — Offline synthetic inventory input

Status: REVIEW_PENDING

Date: 2026-10-03. R2 input parsing/sanitization; worker Claude, independent reviewer Codex. [SPEC](../specs/PANCAKE_OFFLINE_INVENTORY_INPUT_R041_2026-10-03.md). Immutable seed `CVF_SESSION/authority/CCMAI-RUNTIME-041.json`, committed `b19602ea66a47310b503a03ec2954092560231cc` before activation/BUILD. Standing local orchestration delegation; owner manually transfers order to Claude. R2 execution requires that owner transfer/review; Codex does not automatically invoke a worker or implement source.

## Bounded source scope

Only backend/cmd/pancake-proof/main.go, main_test.go, new inventory.go/inventory_test.go; this SPEC/order and governed continuity/status/evidence/catalog/index. No adapter/proof-library/receipt/engine/dependency/UI/workflow/tooling/parent or existing seed change. Existing six behavioral tests must remain semantically intact; helper/entrypoint changes only to integrate optional explicit synthetic file input. No real inventory/capture/credential/config reads, default discovery, network/socket/DB/engine/Analyzer execution, downloads, media, push/merge/deployment/FREEZE. Only synthetic task-owned fixture parsing and in-memory CLI/channel tests permitted.

## Worker sequence

1. Rehydrate canonical manifest/policy/state/memory/active handoff/status/index, SPEC/order/tranche/seed/shared learning. Doctor/local knowledge ingest; exclude generated index/cache. Record fresh declaration/WORK_ORDER -> BUILD acknowledgment before tests or edits; synchronize BUILD surfaces and pass preflight. Verify R2 owner manual transfer before implementation; independent Codex review afterward.
2. Consolidate OI-01..06/schema/resource/error boundaries; implement explicit loader/CLI integration. Missing cached prerequisite or need to edit protected product/harness paths -> BUILD_BLOCKED with concrete reason, no scope expansion. No externally sourced data or secret read.
3. Set GOPROXY=off, GOSUMDB=off, GOTOOLCHAIN=local, CGO_ENABLED=0; run uncached `go -C backend test ./cmd/pancake-proof -count=1 -v`, `go -C backend test ./channels -count=1 -v` (finite in-memory fixtures), `go -C backend build ./...`, `go -C backend vet ./...`. Do not run engine/full DB suites. Race only if cached supported compiler is available without enabling external effects; otherwise retain NOT RUN reason.
4. Execute mounted original-main control and at least four finite semantic mutations defined in SPEC. Assert actual applied changes, named failures, byte restoration and restored baseline; no-op/compile/panic/timeout INCONCLUSIVE. Synthetic test files only, no actual input/network/config.
5. Write docs/reviews/PANCAKE_OFFLINE_INVENTORY_INPUT_R041_BUILD_2026-10-03.md with exact base/seed/diff/hashes, wire schema/sample (synthetic only), mapped OI named tests/CLI commands, limits/attempt counters, original-source/control/mutation/restoration evidence, all failures/skips and NOT RUN. Verify protected paths/six retained tests and existing closures unchanged. Claim offline input/reconciliation only.
6. Catalog -Write/-Check, docs build/doctor/diff, default/PR/full changed-set preflights and mandatory gate tests before bounded local BUILD commit. Return exact40-hex BUILD SHA via documentation hand-back; synchronize REVIEW_PENDING / REVIEW and give Codex evidence. No self-approval/push/merge/deployment/FREEZE.

Same-scope repairs retain seed; third repair without independent new root cause stops REVIEW_COST_ESCALATION_REQUIRED. Missing evidence, leaked input/path, widened reads/transcript coupling, unchecked/truncated JSON, protected-file edit or false live/governance claim prevents acceptance.

## Current boundary

DISPATCH_READY / NOT_BUILT, no worker started. New file option/schema is intended behavior only. R034 offline acceptance, R035 unavailable MCP, R036 synthetic UI acceptance unchanged; R040/R037-R039 and R033 closures unchanged. Live inventory/execution/MCP dispatch are separate future scopes. No real provider-governance claim or receipt.
