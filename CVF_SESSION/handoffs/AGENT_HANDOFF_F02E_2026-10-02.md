# Agent Handoff — F02-E Facebook message coverage

Status: ACTIVE

## Current State

- Project: Customer-Care-Monitor-AI
- Current mode: REVIEW
- Active phase: REVIEW (CCMAI-RUNTIME-031 REVIEW_PENDING / FREEZE_OPEN; global F02 OPEN)
- Active role: REVIEWER (Codex) pending independent review of CCMAI-RUNTIME-031; IMPLEMENTATION_WORKER / COMMIT_STEWARD (Claude) BUILD complete, returned REVIEW_PENDING
- Next allowed move: CCMAI-RUNTIME-031 F02-E is REVIEW_PENDING / REVIEW / FREEZE_OPEN. Independent Codex REVIEWER verifies the exact local BUILD SHA recorded in this handoff, the changed set, F02E-01..09 evidence in the BUILD record, retained tests against the original adapter, mutations M1-M5 and the engine/disposable-MySQL observations; Claude must not self-review or repair during review. Meta endpoint documentation remains unverified; acceptance is the local safety contract only. R030/R029 and predecessors remain REVIEW_PASS / FREEZE_OPEN; global F02/live-channel/credential/provider/governance proof remains OPEN. No real provider/channel/credential action, persistent DB, engine/shared-request product edit, Graph version change, push, merge, deployment, parent edit or FREEZE.
- Parked operator checkpoint: none

## BUILD role acknowledgment (before first source edit)

2026-10-02: Claude rehydrated manifest/policy, state, memory, this handoff, status, order, SPEC, tranche, seed and the shared learnings (including the R030 additions); core `26c686cc99b8be965d2760f27fe875b03376c643` doctor 25/25; knowledge ingest complete; BOOTSTRAP_MIGRATION_PENDING nonblocking. Role transition `WORK_ORDER_AUTHOR (Codex) -> IMPLEMENTATION_WORKER / COMMIT_STEWARD (Claude)` recorded here before any product edit; state, front marker, header, status, order, SPEC, tranche and roadmap synchronized to BUILD, then scoped preflight run before the first source edit. Seed not edited. The BUILD-not-started wording in the planning section below is historical.

## BUILD return (REVIEW_PENDING)

See [BUILD record](../../docs/reviews/RUNTIME_FACEBOOK_MESSAGE_COVERAGE_F02E_BUILD_2026-10-02.md). Changed set: `backend/channels/facebook.go`, `backend/channels/facebook_messages_test.go`, `backend/engine/sync_facebook_messages_test.go`, `backend/engine/sync_facebook_coverage_test.go`, the BUILD record and continuity/status/SPEC/order/roadmap records. Worker-reported: channels 77 PASS lines/0 FAIL; original adapter fails 14 retained channel tests; mutations M1-M8 killed (M1 first run INCONCLUSIVE from a harness Access-denied after I killed a hung test process, rerun killed); disposable-MySQL engine tests F02E-07/08 pass; full backend run: every package ok except engine hit Go default 10 min timeout, engine rerun in two disjoint halves (209 tests) both ok, 0 FAIL/0 SKIP; build/vet PASS; race NOT RUN (CGO disabled/no C compiler); Docker residue none. Not proven: Messenger endpoint compatibility, live behaviour, global F02, provider/governance.

BUILD SHA: PENDING_LOCAL_COMMIT

## Planning acknowledgment and phase trace

Codex rehydrated manifest/policy/current state/memory/handoff/implementation/index and applicable learning. Core 26c686cc99b8be965d2760f27fe875b03376c643 doctor 25/25; knowledge ingest complete; BOOTSTRAP_MIGRATION_PENDING nonblocking. ORCHESTRATOR -> SPEC_AUTHOR -> WORK_ORDER_AUTHOR / SESSION_SYNC_STEWARD / COMMIT_STEWARD acknowledged first in [predecessor handoff](AGENT_HANDOFF_F02D_2026-10-02.md) before planning edits. Dispatcher Codex authored immutable seed committed at `61eda32442b6049e0ed37a00f6664c7fe880da94` before activation; baseCommit equals that commit. Claude future implementation/local BUILD commit; independent Codex REVIEW. No product BUILD/source or acceptance tests in this planning turn.

- INTAKE: isolate remaining Facebook message layer from accepted R022 conversations and R030 Pancake messages.
- DESIGN: preserve existing Graph v21.0/fields; official Messenger/Graph endpoint docs unavailable, official generic SDK limited reference. Local contract: valid data/rows/paging, next-link terminal, safe endpoint before token transmission, blocked redirects, HTTP/size/budget/cycle controls, inclusive filtering and stable chronology. No endpoint/live compatibility claim.
- SPEC: [F02-E](../../docs/specs/RUNTIME_FACEBOOK_MESSAGE_COVERAGE_F02E_2026-10-02.md) defines F02E-01..09 and evidence boundaries. Distinct safe empty/duplicate pages with next continue; no Pancake terminal-rule transplant.
- WORK_ORDER: [R031](../../docs/work_orders/CCMAI_RUNTIME_031.md) limits Facebook source/tests while preserving accepted Pancake scope, real adapter+engine on isolated synthetic HTTP/MySQL, independent reviewer. DISPATCH_READY; BUILD not started at this planning checkpoint.

## Shared learning triggers

Read [folder convention](../../docs/reviews/learnings/README.md), [coverage layers](../../docs/reviews/learnings/feedback_sync_coverage_evidence_layers.md), [repair/mutation learning](../../docs/reviews/learnings/feedback_cvf_repair_workflow.md), [cleanup paths](../../docs/reviews/learnings/feedback_shell_cleanup_and_paths.md) as relevant. Keep project-root cwd, check prerequisite exits, record role/BUILD before source edits, prevalidate rerunnable sync, test invalid duplicates alongside valid progress, count actual mutation changes/restore bytes, retain survivors/unexplained failures and race NOT RUN. Label actual connection error distinctly from HTTP500. Reusable findings belong in shared learning during handling, with local application and parent disposition separate; no parent edit authorized.

## Preserved acceptance and limits

R030 exact BUILD `31daee1d2f736166c4514ec2487d9b94b94727cd` remains REVIEW_PASS / FREEZE_OPEN: [review](../../docs/reviews/CCMAI_RUNTIME_030_F02D_INDEPENDENT_REVIEW_2026-10-02.md). R029/R022 and predecessors unchanged. Global F02, Zalo messages, real visibility/retention/cursor/offset stability, credentials/provider/governance proof and FREEZE remain OPEN. Generic SDK/public documentation retrieval is not channel proof. No real provider/channel/credential action, persistent DB, parent edit, push/deploy or product source action in planning.

## Planning verification

Seed scoped preflight7/7; mandatory gate tests46/46 PASS (31.288 s); diff PASS. Default and origin/main..HEAD preflights FAIL only on pre-existing excluded knowledge index/two Python bytecode files; no whole-worktree/PR PASS. Final 12-path seed/planning scoped preflight7/7 PASS; catalog regeneration/check PASS; docs build PASS (15.09 s), new-document local links/diff PASS. Seed Git-normalized blob identity PASS (Windows newline conversion distinguished from byte restoration of mutations). Product source/tests untouched and R030 tranche JSON/disposition unchanged. SESSION_SYNC_STEWARD / COMMIT_STEWARD (Codex) records final WORK_ORDER context and commits planning locally; no closure/FREEZE. Product/adapter/engine/DB/race/live checks NOT RUN in documentation-only planning.

## BUILD verification (Claude, worker-reported)

Build-start scoped preflight 7/7 PASS before the first source edit; workspace doctor 25/25 and knowledge ingest before BUILD. At return: catalog regenerate and check PASS, docs build PASS (15.08 s; run through cmd because the PowerShell npm.ps1 shim is blocked by execution policy), mandatory gate unit tests 46/46 PASS (24.4 s), `git diff --check` clean, Docker inventory shows no `ccma-test-*` residue. Explicit changed-set preflight below; default, whole-worktree and `origin/main..HEAD` preflights are expected to fail only on the three excluded pre-existing untracked files and are not claimed as PASS. Race NOT RUN (CGO disabled/no C compiler).
