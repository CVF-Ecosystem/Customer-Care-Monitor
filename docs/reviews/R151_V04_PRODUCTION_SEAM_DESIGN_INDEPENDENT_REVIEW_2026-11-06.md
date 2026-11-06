# R151 production seam design independent review

Status: CHANGES_REQUIRED
Date: 2026-11-06
Risk: R2
Reviewer: Astra Medium independent R2 DESIGN REVIEWER, /root/r151_expired_review
Author: Root SPEC_AUTHOR / ORCHESTRATOR
Worker: Terra 6.1 IMPLEMENTATION_WORKER, held
Design: `docs/reviews/R151_V04_PRODUCTION_SEAM_DESIGN_2026-11-06.md`
Design LF SHA-256: `c7e4bbc25d9b73bda6fe75ea6a511beffa5cd7627e15ee00f76ef3171aeaaa6b`

## Decision

CHANGES_REQUIRED for a concrete terminal-persistence contract and discriminator. The proposed production replacement transaction addresses the repeated V04 seam gap, but the design has not yet specified how to prevent a stale intermediate persisted PASS from being treated as a terminal receipt when final persistence fails. This is a design-only finding. No third repair, test, syntax check or runtime effect is granted.

## F01 — complete the final persistence design before implementation

The design requires a downgrade and bounded stdout/stderr receipt after final-save failure, but leaves the authoritative persisted terminal state and the executable finalization seam unspecified. This matters in the current exact runner: line423 sets outcome A1_A2_PASS before cleanup; record at169 calls save at155-157; cleanup records at37/42 can therefore persist that outcome before final cleanup reduction. If the final save at440 then fails, stdout can say INCONCLUSIVE while the prior file still contains A1_A2_PASS. This is a static possible sequence, not an observed runtime failure.

Revise the design to define which receipt/field constitutes terminal completion and how a consumer distinguishes intermediate state from a final accepted outcome. For example, keep intermediate persistence nonterminal and commit the final accepted terminal receipt only after cleanup and its required persistence succeed; the exact mechanism is Root's design choice. Specify the failure behavior if both file persistence and the fallback sink fail: acceptance must remain absent/UNKNOWN, without a claim that evidence was durably retained.

Add a direct injected finalization discriminator using the same production finalization path that CLI calls: after otherwise successful controls and cleanup, force the final persistence operation to fail and verify that no available artifact is admissible as terminal PASS, the returned/fallback outcome is non-PASS when its sink works, and no replay occurs. Also distinguish failure while recording cleanup evidence from failure of the final save. The present bullet about cleanup persistence and later Chrome observation does not execute this last path; repeating it would leave the A2 F02 qualification unresolved. An explicitly shared finalization function or another testable production boundary is needed, rather than a fixture that independently computes the downgrade.

## Replacement design assessment

The proposed transaction moves the actual timeout rejection and launch eligibility into the same production function called by CLI and fixtures. Prohibiting caller-supplied close decisions directly addresses the A2 substitution problem. Required order is sound at design level: reject consumed/timed-out custody, signal/wait once, reject timeout/throw/persistence/unknown results, require complete ABSENT terminal identities, then debit and start once. A failed debit cannot lead to launch, and a start failure cannot authorize retry. Profile/port binding and observed identity retention remain explicit.

The stated mutation discriminator is appropriate if it changes the production decision under test, with the same timeout fixture and downstream conditions that would otherwise allow launch. Changing only a fake wait/callback outcome is not that mutation. Any isolated mutant must be derived from the pinned production candidate; the original candidate must remain unchanged. The future exact test plan must identify the mutation target, show a normal positive case reaches one launch, and require the timeout assertion to reject the mutant. This review approves that discriminating intent, not an unimplemented mutation mechanism or a test invocation.

Current custody exposes hasTimedOut and one-wait enforcement but does not expose a general consumed-handle predicate or retained result/identity record. The design's stronger consumed/result requirement is therefore prospective implementation work, not an already available capability. A future implementation must own that state in the shared production transaction/custody, not duplicate it in the fixture. This fits the intended runner/helper boundary subject to a later exact packet.

## Preserved controls, budget and authority

The design explicitly retains the effect-free import boundary, guarded runCleanup, original caps/profile/port and independently reviewed positive WebSocket/A3/A1-c/A2 sequencing. This preserves the source contract; it does not establish physical Chrome no-relaunch or any native A1/A2 result. Moving those controls or changing protected/product paths requires review rather than being absorbed into the extraction.

The 15-minute read-only preparation ceiling, one design artifact and zero test/source/effect allowance are bounded and consistent with current authority. The document does not provide start/stop measurements, so this review confirms the declared ceiling, not independently measured compliance with it. Historical five focused invocations, two evidenced syntax batches, spent Nov5/A2 allowances and prior active minutes UNKNOWN remain intact.

REVIEW_COST_ESCALATION_REQUIRED remains unresolved before a third correction. Even corrected design acceptance would allow Root only to weigh and explicitly resolve the cost/admission decision under a separate finite work order; it would not release Terra. The suggested one combined future test and one syntax batch are proposals only. Root must not reuse the spent REPAIR_ADMITTED A2 registry pointer for new execution.

## Review context and limitations

CVF Agent Declaration: Customer-Care-Monitor-AI; Core `../.Controlled-Vibe-Framework-CVF` at `c4c8767ac8861854f40d466ba338ffd11eb9558e`; independent DESIGN REVIEWER, R2 ceiling, live governance evidence required YES. Current bootstrap/state/memory/handoff/implementation truth/index and accepted causal audit were re-read. Current mode remains BUILD with BUILD_BLOCKED next move and read-only design preparation permitted. R151 handoff remains active; Facebook/Zalo OA stays parked.

Only this review artifact was authored. Current source was read to assess the design; no source edit, mutation test, syntax check, gate, runtime effect or commit occurred. R139/M3-05/M3-08 remain blocked, R150 A1-b INCONCLUSIVE/A2 UNRUN/cleanup UNKNOWN remain, and no product or runtime governance claim follows.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
