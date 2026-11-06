# R151 candidate/seal design independent review

Status: ACCEPTED_DESIGN_ONLY
Date: 2026-11-06
Risk: R2
Reviewer: Astra Medium independent R2 DESIGN REVIEWER, /root/r151_expired_review
Author: Root SPEC_AUTHOR / ORCHESTRATOR
Worker: Terra 6.1 IMPLEMENTATION_WORKER, held
Design: `docs/reviews/R151_V04_PRODUCTION_SEAM_DESIGN_2026-11-06.md`
Design LF SHA-256: `61791ff0aa88f7a24a10a17c5d20b484cfa7eb7f1caf69a1d2833b0ef4a4159a`

## Decision

ACCEPTED_DESIGN_ONLY. The second correction resolves the prior F01 continuation at design scope by separating persisted candidate bytes from an acceptance seal and making later consumer verification authoritative. No remaining blocking finding is identified in this narrow protocol review. Both prior CHANGES_REQUIRED reviews remain valid history. This is not source acceptance, a filesystem proof, a cost-escalation resolution or a third-repair release.

## Failure and acceptance semantics

Candidate write/replacement success followed by error or readback failure no longer assumes rollback. The candidate may remain, but the finalizer must not publish a seal. Its terminalComplete:true field is necessary but insufficient for acceptance. The writer returns INCONCLUSIVE, and fallback failure creates no claim of durable evidence. This directly addresses the previous contradiction that the old nonterminal file necessarily survives replacement.

The seal is a distinct immutable publication linked to candidate byte hash, session nonce, exact source and terminal-evidence hash, issued only after successful candidate verification. Writer PASS requires successful seal/candidate readback. If seal publication takes effect but reports failure, or seal readback fails, the writer remains INCONCLUSIVE. A later authorized consumer may accept only after independently verifying the already published artifacts and all required identity/evidence/clock linkages. This is explicitly a different observation from writer success; it does not require deleting or rolling back a seal after I/O failure.

The consumer rejects missing, mismatched, malformed, conflicting, duplicate, stale, wrong-session/source or unavailable artifacts. A temporary file, stdout-only claim or unsealed candidate cannot authorize PASS. Non-reused session identity prevents an earlier seal from validating a later candidate. The future exact contract must implement the stated linkages and consumer checks, including required final evidence/clock validity, rather than treating the presence of two files as success.

The design appropriately limits provenance claims. Confirmation that the seal was issued after candidate verification depends on the pinned, reviewed producer path with seal creation private to its finalizer, plus consumer validation. File hashes alone do not authenticate a malicious writer or prove execution order. Those stronger properties are explicitly excluded. Supported atomic replacement and exclusive publication on the target filesystem remain implementation/admission dependencies, not established facts in this review.

## Discriminating fixtures and production seam

The proposed successful finalization case prevents an always-rejecting consumer from satisfying the tests. Separate injected cases distinguish failure before replacement, replacement taking effect before an error, candidate readback throw/mismatch, fallback failure, and seal publication/readback ambiguity. The last case allows qualified later acceptance only when a valid seal was actually published after candidate verification and the independent consumer validates the pair. Thus the fixture contract no longer demands the impossible absence of bytes already published successfully.

The same production finalizeSession must own these decisions in CLI and fixtures; low-level persistence/readback/publication/fallback operations may be injected. The future test plan must use the actual consumer validation path as well, not a fixture-authored substitute for acceptance. This follows the design's independent-consumer requirement and remains an implementation-review obligation.

The production replacement transaction and mutation discriminator retain their previous design merit: no caller-supplied close decision, one original wait, timeout/unknown/error blocking debit and launch, complete ABSENT ledger required, one debit/start with no retry. The proposed mutation must alter the pinned production timeout decision with the same fixture inputs so that the regression is detected. An isolated mutant must not change the final tested candidate. This is still an unexecuted discriminator, not a test result.

Existing import safety, guarded cleanup, caps/profile/port and positive/A3/A1-c/A2 source contracts remain preserved by the design. Physical no-relaunch and native behavior are not proved by these source plans.

## Cost and authority remain held

REVIEW_COST_ESCALATION_REQUIRED remains unresolved before any third correction. Design acceptance permits Root to assess that cost/admission decision and prepare a separately governed finite work order; it grants no source edit, test, syntax invocation, Windows effect or replay. The spent A2 registry admission cannot fund new work.

Five focused invocations, two evidenced syntax batches, expired Nov5/A2 allowances and prior active minutes UNKNOWN remain cumulative. The declared 15-minute read-only design preparation ceiling is retained; this reviewer did not independently measure compliance or create a new preparation interval. The future one-test/one-syntax proposal is not an allocation.

## Review context

CVF Agent Declaration: Customer-Care-Monitor-AI; Core `../.Controlled-Vibe-Framework-CVF` at `c4c8767ac8861854f40d466ba338ffd11eb9558e`; independent R2 DESIGN REVIEWER, live governance evidence required YES. Canonical bootstrap/state/memory/handoff/implementation truth/index were re-read. Current BUILD phase retains BUILD_BLOCKED next move for read-only design preparation and review. R151 handoff remains active; Facebook/Zalo OA stays parked.

Only this review file was authored. No source edits, tests, syntax checks, runtime effects, gates or commits were performed. R139/M3 remain blocked, R150 A1-b INCONCLUSIVE/A2 UNRUN/cleanup UNKNOWN remain. No source/product/runtime-governance/hosted-CI acceptance follows.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
