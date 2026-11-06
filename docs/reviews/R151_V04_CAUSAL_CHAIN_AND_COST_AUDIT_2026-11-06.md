# R151 V04 repeated gap: causal chain and repair cost audit

Status: REVIEW_COST_ESCALATION_REQUIRED_BEFORE_THIRD_REPAIR

Date: 2026-11-06

Risk: R2

Author: Root ORCHESTRATOR

Worker: Terra 6.1 IMPLEMENTATION_WORKER, held

Reviewer: Astra Medium independent R2 REVIEWER, pending review of this audit

## Outcome boundary

The product target remains R139/M3-05/M3-08 integrated acceptance for one owner/context/original clock. R151 is only a diagnostic prerequisite intended to establish A1/A2 HTTP and Chrome ownership evidence. R150 A1-b was INCONCLUSIVE, A2 was UNRUN and terminal Chrome cleanup UNKNOWN. No A1/A2 Windows rerun, integrated product run or live provider call occurred under R151.

The current R151 source candidate is unaccepted. Its A2 fixture reports 52 PASS and the final syntax batch passes. Astra independently accepted the source-level preservation of positive WebSocket, A3, A1-c and A2 control flow (V06), but blocked V04 direct production timeout/next-launch integration proof. The changed source and review are retained at `85666e26`. The static and injected helper evidence cannot establish Chrome runtime behavior or M3 product acceptance.

## Attempts, observed cause and hypotheses

| Attempt | Actual evidence | Result and cost |
| --- | --- | --- |
| R151 initial candidate | Historical red, then 34 and 39 PASS; one syntax batch before a final runner edit | Source return BLOCKED; three focused invocations, one syntax batch spent. Exact final runner untested. |
| Amendment A1 | Exact seven-file candidate manifest and 49 PASS/zero skips; no final-byte syntax; runner cleanup await could throw before Chrome cleanup | Source BLOCKED after original Nov-5 wall stop; one added focused invocation spent, old unused syntax slot expired. |
| Amendment A2 | Actual `runCleanup` injected custody-failure fixture; 52 PASS/zero skips and final-byte syntax; final manifest matches | Astra BLOCKED V04. One new focused invocation and one syntax batch spent. Five focused invocations and two evidenced syntax batches cumulatively. |

The **observed cause of repeated V04 non-acceptance** is test seam placement. In A1, fixtures exercised shared helper functions but not the runner. In A2, `runCleanup` is tested through the actual runner export, but `runReplacement` is a one-line wrapper around the helper. The timeout test supplies its own `close` callback that throws. Production `closeHandle` (including custody wait, timed-out identity guard, kill and record) and the production replacement `launch`/debit closure remain private in `main` and unexecuted by the fixture. The test could stay green if production stopped rejecting a timed-out original handle. This directly explains the independent V04 rejection; it is not an observed runtime failure.

The monolithic, previously effectful `main` made a safe injected production seam harder to expose. A2 moved setup behind a guarded CLI entry, improving import safety, but did not extract the production close/launch closure. That source structure is an observed dependency. It is a **hypothesis**, not evidence, that the one-test cap or time pressure caused the worker to choose a forwarding wrapper. Actual historical active minutes are UNKNOWN; A2 source/check interval was under four elapsed minutes. Do not assign an unmeasured human or model cause.

The independent cleanup-bypass cause found after A1 is materially improved by A2: a custody persistence exception inside actual `runCleanup` no longer skips later Chrome-close and terminal snapshot attempts in the injected fixture. Final `main` terminal reduction/persistence failure is still unexecuted. A green test and static guard are narrower than a persisted runtime cleanup receipt.

## Remaining dependencies and finite discriminator

Before another source IMPLEMENT, Root must produce a **read-only design/verification packet** identifying one production function that owns original-handle close, timeout rejection, replacement launch debit and terminal guard, with injected low-level process/custody/snapshot operations. The same function must be called by CLI and fixture. The packet must state a mutation check: deleting the production timeout rejection must make the proposed test fail. It must also specify how a final save failure is retained without a false terminal PASS. Astra independently reviews the exact design and whether it distinguishes this repeated gap. This preparation authorizes no source edit, test, syntax batch or runtime effect.

Only if that design is independently accepted may Root consider a separate, finite same-R151 source allocation with cumulative cost retained and a new exact work order, budget, stop and acknowledgment. No new tranche is justified by this repeated supporting gap. Before a third repair pass without an independent new root cause, record and resolve `REVIEW_COST_ESCALATION_REQUIRED`; the same V04 gap is **not** a new root cause. A reviewer request for another helper-green run is insufficient. If the production seam cannot be isolated within the allowed R151 paths and zero-effect class, stop and reassess the route to the product outcome.

Root owns scope/cost and the next admission; Terra remains held; Astra is independent reviewer. Reopen for source work only after exact design review, explicit new finite authority, valid pins/gates, committed transition and role acknowledgment. A later Windows effect admission requires accepted source, its own review and immediate A3 recapture. R139 physical no-relaunch, one owner/context/original clock and E01-E06 proof remain open.

## Claim and cost boundary

All five focused test invocations, two evidenced syntax batches, failed/blocked candidate histories, original Nov-5 hard stop, R150 process attempts and UNKNOWNs remain cumulative. The A2 worker reported zero Chrome, daemon, socket, provider, DB, product and network effects; this is not an OS-wide independent inventory. No CVF governance behavior claim is made from mocks or static checks. Facebook/Zalo OA account work stays parked.

This audit classifies A2 as **incomplete source verification**, matching Astra's review. It does not invent a completed repaired product failure or rewrite R139 r2/r3 enrollment in `docs/milestone-coverage-authority.json`. That existing repairControls history remains authoritative for the completed failed R139 outcomes. The present cost escalation is a prospective stop before a third R151 correction without a new independent cause.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
