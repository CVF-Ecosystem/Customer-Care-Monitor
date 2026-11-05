# R151 same-tranche exact-source verification amendment diagnosis

Status: PROPOSED_PENDING_INDEPENDENT_R2_REVIEW
Risk: R2
Author: Root ORCHESTRATOR / WORK_ORDER_AUTHOR
Reviewer: Astra Medium independent R2 REVIEWER
Worker: Terra 6.1 IMPLEMENTATION_WORKER after renewed dispatch only

## Observed failure and bounded cause

After an initial no-source blocked return preserved at `2ed5025c`, Terra returned a source candidate at committed `d98623e0` but could not meet V04/V06/V07. Its historical red exit1 and successive 34/39 green fixture cases were run before the final runner wait change; the syntax batch also predates the final runner bytes. The source inspection found an original-handle timeout path that could be waited again during cleanup. The final change is plausible but untested. Astra accepted the blocked return, not the source. This is an incomplete implementation and an evidence sequencing defect; no completed repaired product outcome was observed to fail. R139 r2/r3 causal enrollment and all earlier UNKNOWNs remain.

Astra also identified test gaps in runner next-launch/terminal sequencing, negative transport write/EOF wiring and future source-manifest binding. The actual runtime behavior is UNKNOWN. Static strings in the runner are not proof of orchestration. No Chrome, daemon, provider, DB, product or external network effects occurred in R151.

## Finite discriminator and cost

Question: On the exact final R151 source bytes, does an injected offline orchestration/transport suite prove one original wait per handle, zero replacement launch and no terminal PASS after timeout or unknown inventory, plus bounded HTTP write/EOF classification and source admission binding?

Proposed amendment: total focused test allowance four (three already consumed; one new exact-source combined invocation), total syntax batches two (one already consumed; one final-byte batch). The original 45-minute ceiling and hard wall stop `2026-11-05T17:02:18+07:00` remain. No automatic retry: any unexpected corrected case failure, syntax failure or budget exhaustion returns BLOCKED_WITH_REASON with raw evidence. Required code changes stay in the existing R151 worker paths; the official work order and repair audit must be repinned and independently accepted before Terra resumes.

The combined test must inject handles, launch functions, snapshots, timers and transport events without opening a socket. It must bind final helper and runner hashes, exercise timeout before replacement, cleanup timeout, late exit, cleanup-record failure, descendant unknown, request URL/key/headers and remote EOF/reset/local teardown. The syntax batch follows all source edits. Positive controls/A3/A1-c/A2 remain source-bound; runtime admission remains separate. A green offline result can support only diagnostic helper acceptance, not A1/A2 or M3.

## Ownership and stop

Root owns amendment, audit, registry, continuity and commits. Terra owns bounded source/test/return edits after release. Astra independently reviews both admission and final result. Sol 6.1 Medium remains read-only technical advisor. Original source candidate, three test invocations, one syntax batch, 13-minute conservative wall debit and actual active minutes UNKNOWN remain in history. This amendment increases only offline verification allowance within the existing source scope, risk R2 and zero effect class.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
