# R151 corrected production seam design independent review

Status: CHANGES_REQUIRED
Date: 2026-11-06
Risk: R2
Reviewer: Astra Medium independent R2 DESIGN REVIEWER, /root/r151_expired_review
Author: Root SPEC_AUTHOR / ORCHESTRATOR
Worker: Terra 6.1 IMPLEMENTATION_WORKER, held
Design: `docs/reviews/R151_V04_PRODUCTION_SEAM_DESIGN_2026-11-06.md`
Design LF SHA-256: `2b70a4b43d6172f1c86cc6ac8a3280c1cabf2dba84d5436df46f7d542d092379`

## Decision

CHANGES_REQUIRED on one remaining terminal-publication failure sequence. The replacement transaction and production mutation contract remain directionally accepted at design scope; the shared finalizer, explicit nonterminal intermediate records and fallback failure qualification substantially address the earlier F01. They do not yet establish the claimed readback-failure behavior. No source work or third repair is admitted.

## F01 continuation — replacement success followed by readback failure

The corrected design writes a receipt containing terminalComplete:true and a PASS outcome through temporary-file replacement, then verifies it by readback. It also asserts that if persistence or readback fails, the previous file remains nonterminal and no file is admissible as terminal PASS. Those statements are not simultaneously guaranteed by that order.

Concrete sequence: temporary write succeeds; replacement succeeds; readback throws, returns mismatched bytes or becomes unavailable. The destination can now contain the complete PASS candidate, even though finalization returns INCONCLUSIVE or fails to emit its fallback. A persistence operation can also complete its effect before reporting an error. Atomic replacement prevents a partial replacement under its supported semantics; it does not roll back a successful replacement when a later readback fails. This is a design analysis, not an executed filesystem test.

Required design correction: distinguish candidate receipt publication from authoritative acceptance. Define what independently observable evidence tells the authorized consumer that the required verification succeeded, and what the consumer does when that evidence is absent, mismatched, ambiguous or unavailable. A file's own terminalComplete:true field cannot by itself prove the writer subsequently read it successfully. For example, an explicitly linked verification/publication acknowledgment or a consumer-owned verification step may supply that boundary; Root must choose the precise contract and its failure semantics. Do not rely on cleanup deletion/rollback succeeding after an I/O failure, or claim that the old nonterminal file necessarily survived a successful replacement.

Correct the proposed injected cases accordingly. Through the actual production finalizer, separately cover failure before replacement, replacement taking effect followed by an error, successful replacement followed by readback failure/mismatch, and those states with fallback failure. Assert that any candidate left on disk is not authoritative under the actual consumer rule. Test a successful finalization as well, so rejecting every candidate cannot satisfy the discriminator. Bound the session/source identity and content linkage; stale acknowledgment or prior-session success must not validate a new candidate. These remain future proposed fixtures, not execution allowance.

This finding continues the prior terminal-evidence concern. It does not introduce a requirement to prove power-loss durability or all Windows filesystem behavior. The existing requirement to choose a supported atomic primitive and qualify unproved platform behavior remains appropriate.

## Elements accepted in the corrected design

- Intermediate writes are explicitly nonterminal; historical R150/R151 raw files cannot be reinterpreted as the new terminal schema.
- The same exported production finalizeSession is called by CLI and injected fixtures and owns the outcome decision. Injecting persistence/fallback operations instead of a fixture-authored outcome addresses the test-seam failure pattern.
- Missing cleanup/custody, timeout, exceeded clock and cleanup/persistence errors reject PASS. Fallback failure does not create a claim of durable retention. No automatic runtime replay follows.
- The production replacement transaction owns close/timeout/launch eligibility and rejects caller-authored close decisions. Its timeout mutation must change the production decision while keeping the fixture inputs fixed; a fake callback failure cannot substitute. Pinning and isolated mutant provenance remain necessary in a future exact test plan.
- Existing import safety, caps, profile/port, guarded cleanup and positive/A3/A1-c/A2 source contracts remain protected. This does not prove actual Chrome physical no-relaunch or native A1/A2 behavior.

## Authority, cost and review context

The declared 15-minute read-only design ceiling is retained; no new preparation interval or source budget is inferred. Actual compliance with that preparation ceiling was not independently timed in this review. Historical five focused invocations, two evidenced syntax batches, expired Nov5/A2 allowances and prior active minutes UNKNOWN remain. REVIEW_COST_ESCALATION_REQUIRED is unresolved before any third correction. Even future design acceptance would not grant a repair, test or syntax allocation.

CVF Agent Declaration: Customer-Care-Monitor-AI; Core `../.Controlled-Vibe-Framework-CVF` at `c4c8767ac8861854f40d466ba338ffd11eb9558e`; independent R2 DESIGN REVIEWER, live governance evidence required YES. Canonical bootstrap/state/memory/handoff/implementation truth/index were re-read before review. Current BUILD phase has BUILD_BLOCKED next move for bounded read-only design only. R151 handoff remains active; Facebook/Zalo OA remains parked.

Only this review file was authored. No source edits, tests, syntax checks, runtime effects, gates or commits were performed. R139/M3 remain blocked; R150 A1-b INCONCLUSIVE/A2 UNRUN/cleanup UNKNOWN remain. No source/product/runtime-governance acceptance follows.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
