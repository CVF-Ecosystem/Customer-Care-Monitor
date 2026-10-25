# R139 stable execution pin, query-local clock and durable resume review

Status: REVIEW_PASS
Risk: R2
Worker: Sol Medium `/root/r139_clock_v3_worker` and `/root/r139_pinned_resume_worker`
Reviewer: Root, with independent read-only Luna Max `/root/r139_resume_independent_review`
Date: 2026-10-26
Scope: Owner-directed offline repair under committed admission78ae94c6, dispatch594d2a66; one R139 order, no runtime effects or product acceptance.

## Initial consolidated static findings

No source check, source import, pin export, OS query, browser/service/provider operation or campaign was used to discover these findings. Root and Luna independently read the candidate and original public Git paths. All findings were dispatched as one combined repair pass before spending focused checks.

| ID | Finding | Required repair |
| --- | --- | --- |
| C01 | V3 still validates historical baseline elapsed intervals, although the new contract uses historical identity only. | Retain exact predecessor/proof/identity checks; apply duration/order/clock agreement only to the current query. |
| C02 | Parent clock samples surround child helper validation/logging as well as the query. | Capture immediately around the original OS child Popen and successful original wait/communicate, before receipt logging/fsync; preserve other child callers. |
| C03 | A test incorrectly expects UTC5.000001/mono5 to exceed2s drift. | Accept that case; reject7.000001/5 while accepting7/5, using exact arithmetic. |
| C04 | New verification copied fictional used1/cap1/attempt1 limits from the exhausted old grant. | Bind the accepted latest successful attempt to positive cumulative usage and the complete sequence of retained attempt refs; no invented Owner cap, reset or discarded failure. |
| P01 | Pinned kit names initialize_cvf_clone.ps1, absent in public e9ddcbcd. | Use the actual pinned project/repository clone initializer names. |
| P02 | HEAD equality with cached origin/main repeats the moving-latest dependency. | Require exact admitted public pin availability/ancestry and clean public checkout; report later cached revisions as information. |
| P03 | Policy truthiness, lock-controlled paths and incomplete export containment weaken pin verification. | Require literal booleans/exact Core ref; BOM-safe JSON; immutable profile paths; exact inventory and containment/reparse checks before execution. |
| P04 | Pin tests cover only constants/hash convention. | Exercise mocked Git inspection/export/run success and negative origin/pin/byte/path/update cases, retaining all required checks. |
| R01 | Arbitrary input strings and return bytes can be labeled a reusable result. | Bind input and receipt envelopes to the frozen stage/source/contract/pins and original external intent/return; retain caller's original custody requirement. |
| R02 | A caller-selected new ledger path can initialize counters again. | Bind a fixed ledger path and campaign/repair identity to the frozen authority; preserve an initialization anchor and reject missing prior ledger/new-path reset. |
| R03 | Helper-only tests do not prove actual campaign resume. | Wire and demonstrate Root's offline checkpoint use; keep G0–G3/runtime resume explicitly unproved until its actual callsites and receipts are reviewed. |

Cumulative history remains10 prior source checks, zero new focused batches at this review. V2 proof/contract/design and prior accepted reviews remain immutable; shared future v2/v3 preclock slot and all campaign effects remain unused. The current profile1.2 doctor incompatibility is preserved historically; the Owner-selected route verifies pinned1.1 controls independently and must not claim the latest doctor passed.

## Product position

Source repair is not yet accepted. The intended result is to continue offline work after quota interruption from validated durable results, without adopting a new Core revision, losing cumulative accounting, or replaying uncertain effects. Product B remains BLOCKED_PENDING_OPERATIONAL_ADMISSION; actual G0–G3/E01–E06 and M3 acceptance remain open under the existing R139 order.


## Final independent source and offline outcome

Root and independent Luna read back the repaired source. The initial C01–R03 findings are settled for the admitted offline scope. Exact source: v3 `acfec28024a249ac9ee20f49d37a3f53597d9fc1338164060fb2a9c3625b9ef4`; pinned preflight `81784375b6948dee973ce1dd4a3b66851786273079f4becfbdd7c7378fbca233`; resume `7ab2bf4c5b97009d36b932fd7aff5cf6d5b42584844f59fc02f5605f06156eda`; pin/resume test `9dd100bae4dcfa18b5c1fd098de56fecc534ad59fd1a4361bda0cbb256b1f227`; supervisor test `3c6aafa1573880be5d3e23eed26ad5edee1a208adda23511509daa7df81262ac`.

Original focused evidence: clock worker PID14932 exit0/46 tests; pin worker first PID6296 exit1/16 tests with one stale expected-message assertion; repaired pin worker PID25072 exit0/16 tests; Root combined PID29180 exit0/62 tests. All original waits returned and source hashes stayed unchanged during each run. The one-line expectation repair changed only the pin test, not the clock worker's relevant design/test dependencies; no second clock check was needed. Cumulative lineage is10 historical plus4 new focused invocations, including the failed pin attempt. The Root combined invocation supplies both clock and pin/resume reviewer evidence and is counted once.

Root's actual offline custody coordinator persists intent before Popen, original wait/exit and raw output hashes, then seals the reviewed ledger. A controlled orchestration stop after the failed pin original return and before checkpoint seal was recovered from the exact original result: `reused=true`, `recoveredUnsealedOriginalReturn=true`, `launchedNewChild=false`, original FAIL/exit1 preserved. A later completed Root checkpoint reuse returned PASS/exit0 without a new test child; focused count remained4 and original intent/result bytes were unchanged. Public receipts are `pinned_resume_interruption_recovery_20261026.json` and `pinned_resume_completed_reuse_20261026.json`. This is an actual local offline interruption/reuse exercise, not a real provider quota event. Missing original outcomes remain UNKNOWN with no replay. A successor ledger after the one-line test repair pins the old ledger and all old intent/return/envelope bytes and explicitly carries the prior two new invocations; it does not reset counters or discard the failed result.

Pinned provenance inspection/export from public e9ddcbcd succeeded without network or installed-module/Core edits. The full pinned verifier attempts are retained: first two failed legacy catalog rendering; Windows PowerShell's generated index passed its own check but differed from the pwsh engine selected by the legacy gate. Regenerating/checking sequentially with `pwsh -NoProfile` settled that metadata issue. The third full attempt passed legacy7/7 and trusted1.1 INST/CONT/RANGE/APPL/ROUTE/CLAIM, while ROLE rejected missing canonical Worker/Reviewer lines in worker returns. Root corrected only those header fields and canonical return status; no source/test rerun follows from these metadata fixes. Final normal pre-commit controls remain required before material commit. Do not claim that latest Core1.2 doctor passed; it is superseded for this campaign by the explicit Owner pin decision and reviewed pinned-source verification.

Independent Luna's final read found no remaining behavior blocker within this offline scope, and explicitly qualified that G0–G3/provider/runtime integration remains unproved. Source REVIEW_PASS covers the hash-bound implementation and local offline recovery. It does not grant an extra campaign/provider slot or FREEZE/M3/product closure. Root must bind the clean source material commit, then prepare the distinct v3 technical packet and actual G0 review before runtime effects.
