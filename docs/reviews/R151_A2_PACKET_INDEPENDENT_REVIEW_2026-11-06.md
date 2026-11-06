# R151 Amendment A2 exact-packet independent review

Status: CHANGES_REQUIRED
Date: 2026-11-06
Risk: R2
Reviewer: Astra Medium independent R2 REVIEWER, /root/r151_expired_review
Worker: Terra 6.1 IMPLEMENTATION_WORKER, held
Dispatcher and commit steward: Root

## Exact inputs and decision

Reviewed official DRAFT packet `docs/work_orders/CCMAI_RUNTIME_151_EVIDENCE_CAPTURE_REPAIR.md`, LF SHA-256 `18e46208832172cfa0e0a284b84428faf8aad8f64f67ba3757811c6d7b16acd8`, and audit `docs/reviews/probes/r139_integrated_20261021/causal_assessment_20261105/repair_chain_assessment_r151_evidence_capture.json`, LF SHA-256 `a84be7be8225709686c969bf0d2b29254255e398257cc10d49d4ca5f44d20a16`.

CHANGES_REQUIRED before exact-packet acceptance. No dispatch is granted. The packet substantially incorporates the A2 proposal conditions, but its current coverage contract still contains the expired supporting-work budget. Hash consistency does not resolve that semantic contradiction.

## Consolidated required fixes

1. **F01 — blocking coverage conflict:** the embedded coverage `action.budgetMinutes` and audit allocate 20 new active minutes, while `supportingWork[0].budgetMinutes` remains 45 and its stop still says original 45-minute/hard-wall limit, four cumulative focused tests and two syntax batches. Replace this current supporting-work contract with the prospective A2 20-active/30-wall allocation, one additional focused invocation and one newly allocated syntax batch, preserving old counts/deadline explicitly as history. The selected supporting work cannot simultaneously be governed by an expired allocation and a new one.
2. **F02 — historical instruction ambiguity:** the section `Same-tranche amendment A1: exact final source only` still instructs Terra, “Before the remaining invocation,” and states that the original ceiling/stop “remain.” Mark this section explicitly historical/expired and non-executable, with A2 the sole prospective source execution contract. Preserve the previous obligation/history text; do not silently delete the failed or incomplete attempt. The packet elsewhere correctly states expiry, but the retained imperative should not be left to worker interpretation.

After these corrections, recalculate the packet pin in the audit and the audit pin in the existing milestone registry. Return the exact corrected bytes for independent readback. The audit's pending decision is appropriate during this review; after acceptance, Root must pin the actual accepted exact-packet review rather than treating the proposal review as final admission. Status transition, pre-dispatch gates, material marker, fresh deadline and worker acknowledgment remain separate prerequisites.

## Conditions already satisfied in the reviewed draft

- The new A2 section narrows worker edits to the named runner/helper/test, uniquely named worker evidence and existing return. Historical logs/manifests cannot be overwritten; R150/product paths and the immutable seed stay protected. Root retains audit, authority, continuity, gates and commit ownership; Terra and Astra remain separate R2 worker/reviewer.
- It retains old four focused tests, one evidenced syntax batch, expired unused syntax allowance, original Nov5 17:02:18 stop and prior active minutes UNKNOWN. The prose allocates 20 measured active/30 wall minutes prospectively, records a release timestamp before edits, uses conservative elapsed time if active measurement is unavailable, and forbids retries after failed checks or changed final implementation bytes.
- V04 requires actual injected runner orchestration: custody persistence exception, later Chrome cleanup/terminal observation, timeout preventing replacement and unknown descendants preventing PASS. It rejects import effects and parallel helper-only proof. Guarded cleanup is the named correction, not an open-ended redesign.
- V06 now requires direct injected sequencing or explicit independent static path review for positive/A3/A1-c/A2 controls; string presence alone is insufficient. Actual HTTP helper and source-manifest regressions remain required. Unproved worker obligations keep the return BLOCKED_WITH_REASON; this review does not pre-accept future static paths.
- V07 retains exact hashes, commands, exits, skips, cumulative cost, unproved paths and final-byte syntax after source edits. Complete return remains conditional on all V01-V07 obligations, without critical skips.
- Zero Chrome/daemon/provider/DB/product/network/listening-socket effects remain explicit. Later Windows verification requires separate authority and immediate A3 recapture. The seed's later effect ceilings are not repurposed as source execution permission.

## Integrity observations and limitations

The audit's packet hash matches the reviewed DRAFT. The milestone registry's M3 audit pointer/hash matches the reviewed audit. All 24 audit history/observed/review file-reference occurrences match their LF hashes. The embedded coverage retains all eight M3 rows; all 16 row/independence evidence-reference occurrences match their LF hashes. These are read-only identity checks, not execution of a machine gate or a complete claim that every coverage contract condition passed.

The audit separates the static custody exception path from unknown actual occurrence and unknown aggregate R139 product cause. It preserves r2/r3 history, cumulative minutes UNKNOWN, a finite discriminator, owner, stop/reopen conditions and pending independent decision. The current authority/continuity remains BUILD_BLOCKED while the official packet is DRAFT; no worker action is implied. The separate status carrier remains the active tranche's blocked work-order pointer.

CVF Agent Declaration: Customer-Care-Monitor-AI; public Core `../.Controlled-Vibe-Framework-CVF` at `c4c8767ac8861854f40d466ba338ffd11eb9558e`; independent REVIEWER, R2 ceiling, live governance evidence required YES. Current bootstrap/state/memory/handoff/implementation truth/index were re-read and coherently retain BUILD_BLOCKED. This action only records exact-packet review. Prior doctor/provenance results are inherited from the preceding review, not rerun here.

Only this review file was authored. No source edits, tests, syntax checks, gates, commits or runtime effects were performed. R139/M3-05/M3-08 remain blocked; R150 INCONCLUSIVE, A2 UNRUN and cleanup UNKNOWN remain; Facebook/Zalo OA remains parked. No source, product, hosted CI or runtime governance acceptance follows.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
