# R151 V04 production seam design and verification contract

Status: DESIGN_SECOND_REPAIR_PENDING_INDEPENDENT_REVIEW

Date: 2026-11-06

Risk: R2

Author: Root SPEC_AUTHOR / ORCHESTRATOR

Worker: Terra 6.1 IMPLEMENTATION_WORKER, held

Reviewer: Astra Medium independent R2 REVIEWER

## Bound and reason

This is a same-R151, read-only design output following the accepted V04 causal/cost audit. Root's preparation reads only the committed R151 runner, helper, test, worker return, independent reviews and current authority. It has a 15-minute preparation ceiling, one design artifact, zero source edits, zero test/syntax invocations and zero Chrome/daemon/socket/provider/DB/product/network effects. It does not resolve `REVIEW_COST_ESCALATION_REQUIRED`, allocate a third repair, or alter the spent A2 audit/registry admission.

The exact failure to distinguish is: A2's fixture supplies its own `close` function that throws on timeout, while production `closeHandle` and the launch/debit closure remain private in `main`. The fixture could pass after production stopped rejecting timeout. An accepted next test must fail under that production regression. A1 helper-green and A2 wrapper-green histories remain retained; no success is inferred from this design.

## Intended production boundary

The current runner already has an effect-free import guard and exports `runCleanup`. Preserve those. Replace the one-line `runReplacement(options)` forwarding seam with one exported **production replacement transaction** that owns the decisions currently split between `closeHandle` at runner lines238-247 and the call/launch closure at line372. Its inputs are low-level injected effects and current runtime state, such as `{handle, custody, waitOriginalHandle, signalOriginal, record, snapshot, ledger, debitLaunch, startChrome, expectedProfile}`. It must not accept a caller-supplied `close` decision callback. Production `main` and the offline fixture call this same transaction. The fixture may inject only low-level wait result, signal result, process factory, snapshot, record sink and clock; it must not implement timeout rejection or launch eligibility itself.

The transaction has this order:

1. Reject a previously timed-out or already-consumed original handle without a second wait or replacement launch. Preserve its PID/creation/profile identity and prior result in custody.
2. Signal the original handle if still alive and retain signal success/failure. Await its original handle result **once** through custody; a timeout, throw, persistence failure or unknown result is terminal UNKNOWN for this transaction. Retain the error and never debit/launch Chrome #2 after it.
3. Snapshot the full original root/descendant ledger, record the terminal identity evidence, and require every known identity ABSENT. Missing, reused, present or unknown descendant state blocks launch. A late exit cannot undo a previously timed-out result.
4. Only then call the production launch debit and process factory once. Bind the intended profile/port and retain the returned handle identity. A debit/save failure cannot be followed by a launch. An unexpected launch/start failure remains recorded and cannot be retried automatically.

This transaction must preserve the original R151 caps and exact profile/port, positive WebSocket, A3, A1-c and A2 sequencing reviewed by Astra. Any extraction that moves these controls, changes the effect class, or requires protected/product paths is outside this design and stops at review. `runCleanup` remains independently guarded so a custody error cannot skip Chrome close or terminal snapshot.

## Authoritative terminal receipt and finalization seam

The current `record()` may persist `A1_A2_PASS` before cleanup finishes. That stored field is a **candidate**, not a terminal result. A future implementation must make all intermediate writes explicitly nonterminal: `terminalComplete:false` and no authoritative `PASS` field. It may store `candidateOutcome` and events for diagnosis. A consumer must reject every file lacking `terminalComplete:true` and a final receipt produced by the shared finalization function, regardless of any intermediate candidate text. Old R150/R151 raw evidence remains historical and cannot be reinterpreted as this new receipt.

Extract one exported production `finalizeSession` function called by `main` and the injected fixture. Its low-level injected operations are candidate persistence, readback, seal publication and a bounded fallback sink, **not** an injected outcome decision. It computes the final outcome only after `runCleanup`, complete terminal inventory/custody, cleanup-error accounting and the original clock/caps. Any unknown identity, timeout, cleanup/persistence error or exceeded clock yields non-PASS.

For an otherwise valid PASS, it writes a **candidate** receipt under a unique session path with `terminalComplete:true`, final outcome, source hash, session nonce, original-clock data and complete terminal evidence. It atomically replaces the candidate path, then reads back and hashes the committed candidate bytes. The candidate field alone never grants acceptance. Only after successful candidate readback may the finalizer publish a separate, immutable **acceptance seal** at a unique path linked to that session nonce, source hash, candidate byte hash and terminal-evidence hash. The seal is atomically published without overwriting an existing seal and is never published on any candidate write, replacement or readback failure. The finalizer reads back the seal and candidate before it emits/returns PASS. No cleanup deletion or rollback is assumed after an I/O failure.

If candidate persistence takes effect and then reports an error, or candidate readback fails/mismatches, the candidate can remain on disk but has no matching seal; the finalizer returns INCONCLUSIVE and attempts a bounded fallback non-PASS receipt. A failed fallback is reported as UNKNOWN with **no claim of durable retention**. No automatic rerun or second Chrome launch follows. If seal publication itself takes effect but reports an error, or seal readback fails, the finalizer also returns INCONCLUSIVE and makes no writer-side PASS claim. A later authorized consumer may nevertheless accept that already published seal only after independently validating both artifacts and every linkage below; this is a qualified later acceptance, not a claimed writer success. For a non-PASS outcome, finalization may attempt a terminal non-PASS receipt, but a failed write still leaves no accepted terminal receipt.

The authorized consumer accepts PASS only when it independently reads a complete candidate **and** its unique acceptance seal, verifies both byte hashes, session nonce, exact source identity, final evidence/clock hash and absence of conflicting or duplicate seals, and confirms the seal was issued only after the candidate verification step by the same production finalizer. A missing, mismatched, ambiguous or unavailable seal/candidate yields UNKNOWN, never PASS. A stale intermediate file, stdout alone, temporary file, unsealed candidate, or seal from another session/source cannot authorize PASS. The exact session identity must be generated before any candidate write and never reused; old R150/R151 raw files have no seal and remain historical. The implementation must choose supported atomic replacement and exclusive seal-publication primitives on the target filesystem and fail closed if their semantics cannot be established. This design does not itself prove those Windows filesystem properties or cryptographic provenance against a malicious writer; the future packet must bind the producer/consumer paths and keep seal creation private to the finalizer.

## Finite discriminating checks for a future separately admitted source pass

The future packet, if cost escalation is resolved, may budget **one** combined exact-source offline invocation and **one** final-byte syntax batch after one source pass; this is a proposal, not an allocation. A single combined invocation must call the exported production transaction with injected low-level effects and cover:

- normal original exit plus complete descendant ABSENT: exactly one wait, one debit and one Chrome #2 launch;
- original timeout: exactly one wait, zero debit and zero launch, explicit timed-out custody retained;
- second call after timeout or late exit: no second wait/debit/launch;
- custody `record`/save throw, signal throw, ledger UNKNOWN/PID reuse and snapshot throw: zero debit/launch, retained error/UNKNOWN;
- debit/save throw or start failure: no unaccounted replacement and no retry;
- cleanup persistence throw: later Chrome close/terminal observation still attempted, final result not PASS.
- successful finalization through the **actual exported `finalizeSession`**: the consumer independently accepts exactly one correctly linked candidate/seal and the writer returns PASS;
- separately force candidate write failure before replacement, replacement taking effect then reporting an error, and successful replacement followed by candidate readback throw/mismatch: in each case any candidate left on disk is unsealed and inadmissible, writer returns INCONCLUSIVE and fallback emits non-PASS if available, without replay;
- repeat those failure cases with fallback also failing: no candidate becomes accepted, evidence availability remains UNKNOWN, and no durable-error claim is made;
- force seal publication/readback failure separately: the writer cannot report PASS; if an atomic seal was actually published despite the reported error, the consumer must independently verify its complete candidate/session/source/evidence linkage before any qualified later acceptance. Missing, malformed or conflicting seals reject. Distinguish this from an earlier cleanup-event persistence throw, which must still allow later cleanup attempts.

The **mutation discriminator** is defined before implementation: if the production transaction's timeout rejection is removed or altered to allow progression, its timeout and second-call cases must fail. It may be demonstrated inside the same admitted combined invocation using an isolated copy or dependency-level mutation mechanism, without modifying the final tested production file after candidate pinning. A test that only makes a fixture-authored callback throw does not meet this discriminator. The same combined invocation must directly force final-save failure through the shared production finalizer; a fixture-computed downgrade does not count. Root must review the exact test plan for this sensitivity before dispatch; Astra independently verifies the final source/test identity and V01-V07 evidence.

All implementation/test edits, if separately authorized later, precede a new candidate manifest and its one combined test/syntax batch. The cumulative cost remains five focused invocations and two evidenced syntax batches now; old Nov-5 and A2 allowances are spent. A failed corrected test, changed final implementation byte, timeout, ambiguity or third-round cost boundary returns BLOCKED with raw evidence, not a new tranche or automatic replay. No runtime effect follows from source acceptance; a later Windows A1/A2 VERIFY still needs separate admission and immediate A3 recapture.

## Decision requested from independent review

Astra should decide whether this boundary actually forces production close/timeout/launch decisions into the same injected function and whether the mutation discriminator would detect the repeated V04 gap. It should identify any unacceptable dependency or missing persistence/terminal guard. Acceptance of this design would only permit Root to weigh the cost of a **third** same-R151 source correction and resolve `REVIEW_COST_ESCALATION_REQUIRED` under a new finite work order. No worker action is released by a design review.

R139/M3-05/M3-08 stay BLOCKED; R150 A1-b INCONCLUSIVE, A2 UNRUN and terminal cleanup UNKNOWN; Facebook/Zalo OA remains parked. No runtime CVF governance proof is claimed.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
