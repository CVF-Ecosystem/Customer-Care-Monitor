# R059 bounded work order — draft awaiting independent reviewer identity

Status: DRAFT_NOT_DISPATCHED

Date: 2026-10-07 (Asia/Saigon). Prepared by Codex `/root` WORK_ORDER_AUTHOR. Document risk R1; proposed source BUILD risk R2. [DESIGN/SPEC EX01..12](R059_EXECUTION_RECEIPT_DESIGN_SPEC_2026-10-07.md) committed at `9f01473e327bd0b25fda14d85d13dea55bbe802e`. This packet is reviewable planning, not an active work order, dispatcher seed, tranche record, BUILD acknowledgment or implementation acceptance. R058 local FREEZE and all original evidence remain unchanged.

## Ownership and actual dispatch dependency

Proposed IMPLEMENTATION_WORKER / REPAIR_WORKER / BUILD COMMIT_STEWARD: Codex `/root`, consistent with owner instruction to continue without subagents. Independent R2 REVIEWER: UNASSIGNED. No historic child role is reused. CLOSER is not dispatched by this packet. Root may check its tests and source but cannot give its own R2 implementation REVIEW_PASS.

Project `AGENTS.md`, Provider-Neutral Role Contract: “For high-risk or governance-significant work, REVIEWER must be independent from IMPLEMENTATION_WORKER”; downstream tranche rule also requires independent R2 reviewer identity. The committed R059 SPEC requires that identity before source BUILD. Remaining dispatch fact is who will independently review the exact source/evidence checkpoint. No agent is called to fill this slot, and owner is not silently assigned technical source review. All other bounded execution choices are specified below.

## Objective and scope

Implement an observational `source_execution` receipt alongside unchanged `source_preparation` in existing JobRun.Summary. It measures logical Analyzer-to-provider invocation, response/error/interruption, usage-row write outcome, parser visibility and per-member result-publication facts. It changes no admission, inference choice, prompt, retry, pricing or business result. EX01..12 are the acceptance contract; this work order narrows implementation paths without expanding it.

Source baseline: `05c59e9978e5cf108b0d9118f64c7454805ec64e`. Project planning baseline: `9f01473e327bd0b25fda14d85d13dea55bbe802e`. Before dispatch, verify HEAD backend matches the source baseline and canonical continuity matches the current handoff. Commit a real immutable authority seed with assigned roles before its activation record's baseCommit. Never store this draft as a valid seed with UNASSIGNED reviewer or alter an old seed to authorize new behavior.

## Exact candidate implementation paths

| Path | Authorized purpose after actual seed/activation |
|---|---|
| `backend/engine/analyzer.go` | Own execution collector; observe existing single/batch begin/return, usage Create error, parser and publication branches; compose summaries including early failure and panic. Preserve all existing operation order and control decisions. |
| `backend/engine/source_preparation_receipt.go` | Additive shared Summary composition only. Keep existing preparationSummary signature and behavior for callers without execution receipt, preserve preparation collector and all its metadata/count/cap semantics. |
| `backend/engine/source_execution_receipt.go` (new) | Typed run-owned collector, fixed enums, sanitized UUID binding, reconciliation counters, bounded entry/member retention, frozen deep copies and additive Summary builder. |
| `backend/engine/source_execution_receipt_test.go` (new) | Pure reconciliation, freeze isolation, bounds, privacy and interrupted-state tests. |
| `backend/engine/source_execution_receipt_db_test.go` (new) | Full Analyzer single/batch/mode/cancellation/write-fault/panic/terminal-row and unchanged legacy behavior controls, using existing helpers and synthetic provider stubs. |

Inspection confirmed `closeOwnedRunSummary` already accepts a composed string and preserves legacy close/Abort behavior. `backend/engine/job_run_ownership.go` is therefore excluded. Existing prep/ownership tests are read-only; do not broaden paths merely to simplify a test helper. No other old test edits, provider/AI interface, model, migration, dependency, frontend, tooling, workflow or configuration changes. Standard continuity/status/catalog/new review evidence may be updated as separately enumerated in the actual seed.

## Implementation sequence and guardrails

1. Rehydrate/declaration; validate assigned implementation/review identities, authority first-blob/baseCommit ancestry and protected-path inventory. Commit BUILD acknowledgment before source edits. That acknowledgment records the full required tuple, exact roles and new execution budget, distinct from consumed R055/R057 budgets.
2. Add pure collector and shared composer. Implement begin once immediately before the existing single/batch method call, after its cancellation gate. Single = one member; batch = one call/N members. Record returns and independent usage/publication dimensions without observing nested HTTP retry attempts.
3. Wrap the existing summary writers and early/panic paths with frozen execution observation. Do not restart preparation or read prior Summary. Observe usage insertion errors from the existing write while preserving execution flow. All legacy scalar counts and existing preparation receipt remain unchanged for identical inputs.
4. Add the new pure/DB tests mapped to EX01..12. Expose no production test knobs/public config. Use existing cancellation barriers/fault hooks and private new pure helpers only within the allowed paths. No new permission or result-processing mechanism.
5. Inspect source and changed set before running the single worker evidence campaign. Run bounded positive, two applied mutation controls and restored positive groups as below. Preserve source manifests, logs, original failure and cleanup truth.
6. Commit source checkpoint, then evidence and REVIEW_PENDING handback with already committed exact source/evidence SHAs. Root does not transition to REVIEWER for this R2 source; named independent reviewer evaluates the handback. Finding repair stays same-scope under unchanged authority; small pointer fixes are handled directly by the responsible reviewer with checks. Scope/role/network/budget boundaries require a new disposition first.

No monetary/token/provider/model/rule-version/approval values in this slice. Begin/end counts are interface observations, not HTTP requests or certified live inference. Successful usage write is operation success, not later reload or billing proof; result parse/write error is not necessarily AI failure. Pending outcomes after panic remain unknown. Returned Summary after failed terminal storage is not durability evidence.

Retain first200 calls and at most200 member UUIDs across the envelope, encoded execution ceiling128KiB. Preserve existing preparation256KiB ceiling independently. Serialize only fixed enums, normalized bound UUIDs and nonnegative counters; all full observation counts survive entry truncation. Private smaller serializer bound may support a deterministic pure test. Fixed prefix/error classes contain no prompt, transcript, response, rules, raw error, URL or arbitrary provider metadata. Existing raw application logs receive no new privacy certification.

## One consolidated evidence campaign per role

Each role: max1 campaign / max4 Go invocations. Worker plus independent reviewer: max2 campaigns / max8 Go invocations total. Pure repository/static/catalog/docs checks are outside Go budget. No uncounted exploratory Go test/build/vet. Normal `go test` compile/default vet is observed through the positive command; do not claim a separate explicit `go vet ./...` run.

| Go invocation | Required proof |
|---|---|
| 1 | All new execution pure/DB tests plus grouped existing SP, LP, preparation/snapshot, single/batch/modes, ownership/finalizer controls. Source-derived inventory and exact regexp/files decided before invocation; evidence names/counts must show no silent omissions or DB skips. |
| 2 | Applied M01: corrupt logical-call observation so prepared candidates are counted as invocations before a provider call. A named zero-invocation cancellation/no-call detector must fail semantically; compile/timeout/DB/setup failure is inconclusive. |
| 3 | Restore baseline, apply M02: erase execution receipt only from stored terminal Summary while returned receipt remains intact. Named DB-reload assertion must fail semantically; no receipt-only returned-object detector accepted. |
| 4 | Restore exact baseline bytes and repeat the healthy new execution tests plus the mutation target controls; all pass without skip. |

Do not fabricate a mutation diff before source exists. After implementation, record exact single-match edits, baseline/mutant hashes, reversible lossless diff and named detectors before each control. Both mutations occur only in the isolated exported tree, never canonical source/tests; no detector weakening. The independent reviewer performs its own campaign on the exact committed source, without relying solely on worker outcomes. The reviewer may choose an independent equivalent mutation site after documenting it within unchanged contract/budget.

Disposable infrastructure only: already available images/cache, readonly exported source and readonly module cache, Go proxy/checksum lookup disabled, private task-internal MySQL with no public ports, task-specific compile cache if needed. Do not fetch missing images/dependencies or use real config/secrets without a separately authorized boundary. Configure positive group timeout480s/attach600s including cold compile, focused mutation/control runtime180s/attach300s, MySQL readiness90s. The campaign controller must save partial stdout/stderr/container logs/state on timeout before scoped cleanup. No automatic retry or budget reset after a bind-mount/setup/timeout failure; retain failure and obtain cost disposition before continuing.

At every stage verify full exported-source file/byte manifest, not Git blob hashes compared directly to physical CRLF bytes. Confirm all old canonical tests/seeds/evidence remain unchanged and new source changes remain inside the seed. Capture JSON Go completion events, subtest/top-level counts, exit, elapsed, skipped names, command, sanitized temporary DSN and raw log hashes. No copied `.env`, provider key, real account/endpoint or customer data. Remove only exactly named task containers/network/task cache/anonymous DB volumes after verifying targets; prove absence, leave live `ccma-*` resources untouched.

## Failure, handback and acceptance contract

Any behavioral regression, privacy leak, counter mismatch, missing durable-reload proof, unexplained skip, incomplete raw output/restoration/cleanup, scope drift or independent-review role missing blocks acceptance. Infra-only mutant failure is not a kill. A partial campaign is reported incomplete, with each unexecuted control NOT_RUN. Never rewrite earlier failed packets or claim full suite success from partial output.

Before source/evidence/handback commits: default preflight, PR-range preflight against resolved origin/main, exact staged path preflight, catalog PS5.1/7, diff/secret/protected checks and docs build after final Markdown. Gate units may inherit 46PASS52.797s at R058 closure9042f91 while scripts/tests/workflows remain unchanged; rerun if a new tooling issue arises. Metadata changes that follow check recording require restaging and staged preflight again. All local evidence claims remain application proof, not CVF AI-governance proof or hosted CI success.

Independent reviewer checks EX01..12 against actual source, negative controls, old behavior and claim limits, then records REVIEW_PASS or CHANGES_REQUIRED with exact buildCommit/evidence. FREEZE is not granted by this BUILD work order. A separately bounded closure decision follows settled independent review; no push/merge/deployment/release permission here. Full S2/global F02/live/rules/PII/budget/usage-price provenance remain open; Facebook/Zalo OA accounts parked.

## Disposition

Work-order scope/path/test/budget packet completed; reviewer identity is the remaining dispatch input. Source BUILD not started, no new Go/runtime/subagents, no authority seed or active R059 record. The project current active tranche remains CCMAI-RUNTIME-058/FREEZE. Once reviewer is named, root can validate matching seed schema, commit seed first, activate bounded WORK_ORDER, acknowledge BUILD and implement without asking again for routine same-scope steps. This draft does not claim approval on behalf of an unassigned person or agent.
