# R151 narrow A1-b and cleanup instrument comparison

Status: DESIGN_PENDING_INDEPENDENT_REVIEW
Date: 2026-11-06
Risk: R2
Author: Root ORCHESTRATOR / SPEC_AUTHOR
Reviewer: Astra Medium independent R2 REVIEWER
Worker: Terra 6.1 IMPLEMENTATION_WORKER, held

## Decision in one screen

**NO-GO for a Windows run or source dispatch now.** Existing files do not form an independently accepted narrow executable instrument. A small, single-launch observer is a plausible next SPEC inside R151, but it needs its own exact source packet, deterministic offline wiring proof, independent R2 source review and separate Windows-effect admission. This comparison does not relax the original R151 A1/A2 acceptance contract or close R151. Stop this route if the thin instrument cannot meet the three observations below without rebuilding the blocked full runner or expanding into A1-c/A2.

This is one read-only comparison of committed R150/R151 source and reviews. No source edit, test, syntax check, Chrome/daemon/socket/provider/DB/product/network effect, gate invocation for execution, or new tranche is authorized by this document. The earlier five focused tests, two syntax batches, expired clocks and UNKNOWN historical active time remain spent.

## Exact reuse and gaps

| Existing source | Reusable bounded capability | Gap before any run |
| --- | --- | --- |
| `docs/reviews/probes/r151_evidence_capture_20261105/diagnostic_helpers.mjs` | `negativeProbe` emits a fresh HTTP WebSocket upgrade over `127.0.0.1` and retains parsed HTTP status/headers; `OwnershipLedger` requires a positive root and tracks observed descendants by PID/creation time. Offline helper fixtures were reviewed in R151 A2. | Helper acceptance is not acceptance of the calling observer. A transport reset, timeout, incomplete response or missing status remains INCONCLUSIVE. |
| `docs/reviews/probes/r151_evidence_capture_20261105/process_snapshot.ps1` | Corrected PowerShell 5.1 profile parsing and CIM row shape; worker fixtures cover zero/one/two and malformed profiles. | Needs an exact, reviewed original-handle owner and scheduled snapshots. A later PID absence cannot repair a missing terminal inventory. |
| `docs/reviews/probes/r151_evidence_capture_20261105/prestart_full_tree.ps1` | Existing fixed-path A3 recapture with the declared README/package.json exception. | Must be invoked immediately before a later admitted Chrome start and return PASS; current read-only comparison does not run it. |
| `docs/reviews/probes/r151_evidence_capture_20261105/owner_runner.mjs` | Source contains positive `Browser.getVersion` control, endpoint discovery, custody and cleanup code. | The full runner is BLOCKED at V04 and its persisted intermediate `PASS` is not authoritative. It must not be reused as the narrow instrument or relabeled as accepted. |
| R150 raw receipts | Establish the historical positive endpoint and generic A1-b timeout, one original-handle SIGTERM/wait, and later known-PID absence. | Actual wrong-GUID HTTP response and terminal descendant cleanup were not retained; they cannot be reconstructed. |

## Minimal prospective discriminator

One proposed thin observer would have **one owned Chrome root** and **one original handle**. It would record, within one original clock and one session identity:

1. Immediate A3 PASS before startup; exact endpoint positive WebSocket plus `Browser.getVersion` reply, with root PID/creation/executable/profile witness and a retained descendant ledger.
2. One wrong-GUID raw HTTP upgrade through the existing `negativeProbe`, including actual status/version/headers or the exact transport event. Only an observed HTTP 4xx supports the narrow refusal claim; a valid 101 is failure; every ambiguous/no-status result is INCONCLUSIVE. No generic WebSocket error is upgraded into an HTTP claim.
3. Signal and await the **same original handle** once, retain timeout/error, then take an in-campaign terminal snapshot and reduce every known root/descendant identity to ABSENT. Any timeout, unknown command/creation time, PID reuse, present descendant, missing snapshot or persistence error is INCONCLUSIVE. An independent reviewer reads raw observations, not a runner-written PASS label.

The thin source would need one entry point plus one focused offline wiring fixture, with one combined test invocation and one final-byte syntax batch only **if separately admitted**. These are proposed ceilings, not granted allowances. The test must call the actual proposed entry point with injected low-level process/socket/snapshot/persistence effects and show positive, negative and missing-evidence reductions; a fixture-authored conclusion does not qualify. It must retain raw events and effect counters. A first failed corrected invocation stops this route for causal review; no automatic repair loop, test reset or broader case matrix. If this cannot be done without extending the source surface beyond the narrow observer/helper or changing the acceptance contract, return NO-GO rather than growing the packet.

This could establish **fresh A1-b and scoped terminal cleanup observations only**. It cannot establish A1-c port reuse, A2 pageId behavior, physical no-relaunch or R139/M3 product acceptance. R150 remains historically INCONCLUSIVE/A2 UNRUN/cleanup UNKNOWN. The R151 parent remains BUILD_BLOCKED even if the narrow observation later succeeds.

## Authority and environment boundary

The immutable R151 seed requires deterministic offline source evidence, independent R2 source review, a separately reviewed effect admission, pre-dispatch PASS and immediate A3 full-tree recapture before Chrome/native startup. A new packet must pin exact source/commands/clock/effect caps and terminal cleanup reserve. The spent A2 REPAIR_ADMITTED registry pointer cannot fund it. No Windows effect is released here.

At this readback, the workspace `operator-local` source pack has changed from the Nov-5 admitted manifest (`ce825b8d1`, dirty) to `f8175616c` (clean). Its runner hash differs from the installed profile 1.3 runner, and its trusted bootstrap reports `LOCK_INVALID` plus missing `.cvf/workspace-activation.json`. The public-Core doctor reports the known public1.2/installed1.3 `LOCK_INVALID` only. The project-installed profile 1.3 files still match its lock 16/16 and the CI profile/runner/bundle pins. Root deliberately retains that installed pin and makes no pack upgrade. The new pack bootstrap failure is **not** a PASS; reconcile this provenance/activation boundary before any material source or effect packet. It does not authorize reading the new pack as an inherited rule for this R151 run.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
