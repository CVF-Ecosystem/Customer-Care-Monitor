# R153 M2 entry inventory — worker return

Status: COMPLETE_PENDING_REVIEW
Risk: R2
Worker: Terra 6.1 read-only VERIFY worker
Reviewer: Astra Medium independent R2 reviewer

## Scope and role transition

Terra transitioned from held IMPLEMENTATION_WORKER to the dispatched read-only VERIFY responsibility after rehydrating the corrected canonical state and handoff at HEAD `2733ba12ffa14b8c3620f0bf280b208e9293dc46`. Dispatch continuity `50485cc7dc19b916d0b4e64ade0d8eaad222b55c`; seed `c96c479589f9d5d2b2a62dc02cbf14b27a23c5c2`. The official work order raw SHA-256 is `600e1b0dd13709d78f3979448d4bcf994c8fa0f80a063ab7be3fd3589aae479e`, matching the handoff marker. The previous attempt stopped before source reading due to continuity drift. Root corrected the canonical pointer before this pass.

This is a finite T0 source map with owned UNKNOWNs, not a runtime result. The approved roadmap `docs/reviews/planning/CCMA_PRODUCT_DELIVERY_ROADMAP_2026-10-13.md:347` asks for common admission across enabled entry paths; the historical M2 spec `docs/specs/M2_INTERVENTION_APPLICATION_SPEC_2026-10-14.md:16` and `:20` constrains its manual synthetic intervention route. Those documents are not implementation evidence.

## Observed registration and routing

| Entry class | Observed source | Admission and remaining uncertainty |
| --- | --- | --- |
| Manual/test-run | `backend/api/router.go:210-212` registers trigger, test-run, and intervention-run with jobs write permission; `backend/api/handlers/jobs.go:410-439` implements test-run and rejects intervention jobs at 419-420; `:448-458` rejects them in trigger. `backend/api/handlers/intervention.go:29-39` calls `InterventionRunner.RunManual`; `backend/engine/intervention.go:58-69,98` validates the intervention/manual/active path and calls `ReserveJobRun`. | Reservation is observed for the manual intervention route, but a common full M2 eligibility/disclosure/budget admission across all routes is **UNKNOWN**. Trigger/test-run worker and reservation call chain remains incomplete. Owner Root; destination a separately bounded source/evidence decision. |
| Scheduled | `backend/engine/scheduler.go:94-95,223-235` loads and registers cron jobs; `:250-264` rereads the job and calls `analyzer.RunJob`. | `backend/engine/analyzer.go:39-47` shows its `execute` rejects intervention jobs and calls `ReserveJobRun`, but the `RunJob` to `execute` link was not traced within this file budget. Shared M2 admission **UNKNOWN**. Owner Root; destination trace `RunJob` and its dependencies under a reviewed follow-up. |
| After-sync | `backend/engine/sync.go:243` invokes `TriggerAfterSyncJobs`; `backend/engine/scheduler.go:289-293,315-327` selects active after-sync jobs and calls `analyzer.RunJob`. | As for scheduled, the complete `RunJob` to eligibility/disclosure/budget chain is **UNKNOWN**. Sync reservation `backend/engine/sync.go:248-256` concerns channel sync; it is not evidence of analysis admission. Owner Root; destination include this chain in the same future comparison. |
| Agent / MCP | `backend/api/router.go:280-285` registers agent run; `backend/api/handlers/agents.go:42-48,124-175,265-287` checks operation permissions/configuration and calls `analyzer.RunJob` for matching active analysis jobs. `backend/mcp/tools.go:159-167` declares `cqa_trigger_job`; `backend/mcp/handlers.go:92-94,124,335-339` dispatches it and lists rights. | MCP handler's complete execution/reservation chain and agent's complete `RunJob` to M2 admission chain are **UNKNOWN**. Tool declaration and permissions do not establish shared M2 admission. Owner Root; destination trace MCP trigger plus analyzer execution in a separately bounded decision. |
| Rerun | `backend/api/router.go:225` registers execute with case read/write and jobs write permissions; `backend/api/handlers/intervention_waitdata.go:85,106` calls `engine.ClaimWaitDataExecution`; `backend/engine/intervention_waitdata_executor.go:181-189,226,272-293` shows a distinct claim, ownership and source/grant checks. | Whether this distinct claim and ordinary `ReserveJobRun` implement one complete M2 admission contract is **UNKNOWN**; their names and partial checks cannot establish equivalence. Owner Root; destination compare exact eligibility, disclosure, reservation and provider-call boundaries before designing consolidated evidence. |

## Observed facts versus hypotheses

The lines above are source observations. A shared reservation name on some routes is not proof of common end-to-end M2 admission. A possible design question is whether the ordinary analysis and intervention/rerun paths need a common contract or explicitly accepted separate boundaries; that is a hypothesis for Root, not a finding of bypass. No runtime, governance, no-bypass, or product-acceptance conclusion follows.

## Source identities and cost

Exactly 12 distinct implementation files were opened, reaching the 12-file stop. Selection searches were used before reading; no further implementation files were opened after the cap. Source work ran approximately 15:16:37–15:17:03 on 2026-11-06 +07, under one conservative elapsed minute; actual active time is not separately instrumented. Raw SHA-256 identities:

| Path | SHA-256 |
| --- | --- |
| `backend/api/router.go` | `fbe99fee8c4def1e87373ac6068bd9e8406409939a3172e54a5717e86e934177` |
| `backend/api/handlers/jobs.go` | `8b759b8a2bd54250a1e050553f80afd9bc6679d94928e8acd2e01c40178453b9` |
| `backend/engine/analyzer.go` | `d1c9ba5153cbd79e891899f78d42fa0caacbaf7ef2ae570c1dc7084382971bc2` |
| `backend/engine/scheduler.go` | `335c5fccf1d64c0b8fdfece4c19b7f6d2e51f49494ca7fe203c587376724ffc5` |
| `backend/engine/sync.go` | `c841ea0865f1f4b83853409a0e8939c87f99d5009c42b6729cd01beefaaaced0` |
| `backend/api/handlers/intervention.go` | `af4cf9cc2a109e468cb9c4297e94581cc92c96934f9e2fafd0c3b52bc085ef58` |
| `backend/engine/intervention.go` | `cfc5f331385f81e112f7ecf039bfde8d2d5da11c6604a107256c8d3754e734f6` |
| `backend/api/handlers/intervention_waitdata.go` | `cd93360e52c6dd13fc62d3d3c1bc8f18aeaf55c7c359d2e6f026db01fe3e9a0f` |
| `backend/engine/intervention_waitdata_executor.go` | `14726fbc3ea31cddb1cf14742543e221c910ee9082308c733d4cbabf92ab3faf` |
| `backend/api/handlers/agents.go` | `ffbf8a94c8e0650b2f998531f43b425a3d07195c847e7c1639ffda17eff2b0e1` |
| `backend/mcp/handlers.go` | `502400023232a569c958070f9c91466f3a9480459a4c39b16e9296e0b7c268ef` |
| `backend/mcp/tools.go` | `a84c43502741769d61bb746872a97b4e532b534e7436dc2086d084b78e28f4e4` |

No tests, product runtime, provider, network, browser, daemon, DB, credentials, source edits or other effects were invoked. Only this return was written. R151/R139/M3 and account holds persist.

## Return-Time Closeability Recheck

The finite inventory is complete with explicit UNKNOWNs and awaits Astra's independent review; it is not product completion. One next decision for Root: determine whether to authorize a single consolidated source/evidence comparison of ordinary `RunJob`, MCP trigger and the distinct intervention/rerun claim against the M2-04 contract, preserving this 12-file cap and all prior budgets. Any new source work requires its own governed authority.
