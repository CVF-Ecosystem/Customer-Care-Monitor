# R151 third-repair cost and route decision

Status: ROOT_NO_GO_PENDING_INDEPENDENT_REVIEW
Date: 2026-11-06
Risk: R2
Decision owner: Root ORCHESTRATOR
Reviewer: Astra Medium independent R2 REVIEWER
Worker: Terra 6.1 IMPLEMENTATION_WORKER, held

## Decision and basis

Root resolves the recorded `REVIEW_COST_ESCALATION_REQUIRED` by deciding **NO-GO** for a third source correction of the current R151 full diagnostic runner. This decision stops that implementation route; it does not accept the A2 candidate or undo the independently accepted sealed design. The repeated V04 gap is observed: the fixture-supplied timeout callback could keep passing after the production close/wait/launch decision regressed. The accepted design fixes that seam but also requires a shared finalizer, independently checked candidate/seal publication, consumer validation and fault cases. Sol 6.1 Medium's read-only planning estimate is 250–450 changed/added lines, 20–30 focused cases, 45–75 active worker minutes plus independent review. This is a rough forecast, not measured cost or an allocation. One combined offline invocation cannot prove Windows atomic/exclusive publication semantics. A third short allowance would be mismatched to the work.

The immediate roadmap value of this full runner correction is diagnostic only. Even successful source acceptance would still need a separately admitted Windows A1/A2 measurement and would not close R139 physical no-relaunch, same-owner/context/original-clock integration or M3-05/M3-08. Five focused tests and two evidenced syntax batches are already spent; original Nov5 and A2 clocks/allowances are expired, prior active minutes UNKNOWN. The previous two CHANGES_REQUIRED design reviews and final design-only acceptance remain visible.

## Lower-cost prospective route, still held

Root chooses a **narrow fresh Windows observation** as the next design question inside R151, without opening another tranche. A separately reviewed packet would need to bind one owned Chrome launch; a valid-endpoint positive control and wrong-GUID raw HTTP upgrade through an accepted helper; original process identity and descendant ledger; original-handle termination/wait; and a terminal inventory retained by an independent observer. It must use source that has received narrow acceptance and must not read the blocked runner's persisted `PASS` field as authority. It must state exact stop/effect caps, original clock, cleanup and missing-observation handling before dispatch.

No current observation can reconstruct the missing R150 wrong-GUID HTTP status or its incomplete terminal descendant cleanup. A later PID absence is not retroactive proof. A fresh narrow run, if separately admitted and successful, could establish only fresh A1-b and scoped cleanup observations. R150 historical status remains INCONCLUSIVE, A2 UNRUN and cleanup UNKNOWN; A1-c/A2 and R139 product/physical holds remain open. The proposed route deliberately changes the diagnostic method from the accepted full-runner design, so the specific instrument and claim boundary need independent R2 review, source acceptance and a new Windows-effect admission before any execution. The existing R151 A2 REPAIR_ADMITTED registry pointer cannot fund it.

## Current governed move

R151 remains `BUILD_BLOCKED`, carrier `BLOCKED_WITH_REASON`, Terra held. Root may prepare one bounded read-only exact-instrument/observation comparison and seek independent R2 review within R151. No source edit, fixture/test/syntax invocation, Chrome/daemon/socket/provider/DB/product/network effect, replay, new tranche or product/governance claim is released by this decision. If the narrow route cannot provide reliable actual HTTP and terminal-ledger evidence without expanding beyond its bounded packet, stop and keep R151 blocked; do not revert silently to the costly third repair.

This is an orchestrator cost/route decision pending independent review, not an implementation order or Windows admission.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
