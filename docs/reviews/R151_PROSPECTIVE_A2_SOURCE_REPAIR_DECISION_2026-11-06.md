# R151 prospective Amendment A2: runner cleanup and exact source proof

Status: PROPOSED_NOT_ADMITTED

Date: 2026-11-06

Risk: R2

Author: Root ORCHESTRATOR / WORK_ORDER_AUTHOR

Worker: Terra 6.1 IMPLEMENTATION_WORKER, conditional on fresh dispatch

Reviewer: Astra Medium independent R2 REVIEWER

Parent: the same `CCMAI-RUNTIME-151`; no successor tranche or product work

## Decision needed

The original R151 source clock stopped at `2026-11-05T17:02:18+07:00` and cannot fund another edit or check. The amended candidate is retained at commit `c942ad29`; its seven raw hashes match the worker manifest and the fourth focused fixture log reports 49 PASS with zero skips. The last runner edit followed the one evidenced syntax batch. Astra found a static path where a custody write failure during `waitExit(daemon)` exits `finally` before Chrome cleanup and terminal inventory. The tests do not execute the runner orchestration. R151 is `BUILD_BLOCKED`, source acceptance INCONCLUSIVE.

This document proposes a **new, prospective allocation** within R151. It does not amend the official work order or release a worker. Root will update the current R151 audit, packet, registry and continuity, obtain independent R2 admission review, pin exact bytes, pass pre-dispatch, commit a material marker and record Terra's fresh role acknowledgment before BUILD. The old stop and all old attempts remain immutable.

## Exact proposed allocation

- One source correction pass in the existing R151 diagnostic directory. Worker may edit only `owner_runner.mjs`, `diagnostic_helpers.mjs`, `worker_offline_tests.mjs`, a new `worker_*` raw evidence/manifest file, and `R151_SOURCE_WORKER_RETURN_2026-11-05.md`. No R150 or product file changes.
- New source-only budget: **20 active worker minutes**, with a conservative **30 wall-minute hard stop measured from the future committed BUILD release marker**. Root records the exact release timestamp before dispatch. Unknown prior active minutes are not subtracted from or represented as known; this is a separate prospective allocation.
- One additional combined focused offline test invocation and one additional static syntax batch, both after all source edits and an exact candidate manifest. The history stays 4/4 focused invocations and 1/2 evidenced syntax batches under the expired allowance. New allowance would make cumulative focused ceiling 5, with 4 already spent, and one separately allocated syntax batch; the expired unused syntax slot is not revived. No rerun after a failed corrected case or changed final byte.
- No Chrome, daemon, listening socket, provider, external network, package fetch, DB, credential, customer or product effect. No Windows A1/A2 VERIFY, R139 integrated campaign, deployment, push or release.

## Required discriminator and stop

1. Guard every awaited cleanup operation, including daemon wait and custody persistence, so one failure is retained and subsequent Chrome cleanup and terminal inventory are still attempted within bounded waits. If the runner cannot preserve the terminal evidence, it reports BLOCKED/INCONCLUSIVE; it never emits PASS.
2. The one combined offline test must execute the actual runner orchestration through injected handles, custody failure, timed-out original wait, replacement-launch counter and terminal snapshots. It must prove that a custody save throw cannot skip later Chrome cleanup/inventory, that timeout stops another Chrome launch, and that unknown descendant state cannot pass. It also retains the previously passing HTTP helper and positive/A3/A1-c/A2 checks. Importing the runner must not launch any process or socket.
3. The final syntax batch checks the exact final runner/helper/test bytes and the existing PowerShell probe parsers. The worker return records commands, exits, skips, candidate hashes, cumulative costs and unproved paths. Any source edit after the check invalidates acceptance.

Stop at the first unexpected corrected test or syntax failure, any missing direct-orchestration seam, protected path need, budget limit, ambiguous evidence or prospective hard wall. Preserve the raw failure and return `BLOCKED_WITH_REASON`; audit cause before another dependent action. Astra alone decides source review. A source REVIEW_PASS would allow Root to consider a **separate** R151 Windows effect admission with immediate A3 recapture; it would not accept A1/A2, R139, M3 or runtime CVF governance behavior.

## Product checkpoint

M3-05/M3-08 and R139 remain BLOCKED. R150 A1-b INCONCLUSIVE, A2 UNRUN and terminal Chrome cleanup UNKNOWN remain. Physical no-relaunch, one owner/context/original clock and E01-E06 integrated proof remain open. Facebook/Zalo OA account work remains parked. This supporting repair is justified only because the current instrument cannot yet produce trustworthy A1/A2 diagnostic evidence; no new product milestone is claimed.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
