# R151 causal diagnosis before diagnostic helper repair

Status: DESIGN_SPEC_PENDING_INDEPENDENT_REVIEW
Risk: R2
Author: Root ORCHESTRATOR / WORK_ORDER_AUTHOR
Reviewer: Astra Medium independent R2 reviewer
Worker: Terra 6.1, source-only after dispatch
Parent: R150 FROZEN INCONCLUSIVE; R139 BUILD_BLOCKED

## Observed cause, hypothesis and remaining unknown

1. **Observed ownership measurement cause.** R150 invoked Windows PowerShell 5.1 through `System32/WindowsPowerShell/v1.0/powershell.exe`. In `process_snapshot.ps1`, `@(ConvertFrom-Json -InputObject $OwnedProfilesJson)` makes one outer element of type `System.Object[]` for the two profile strings. Its `IndexOf` comparison therefore matches neither profile. Root reproduced `COUNT=1`, `MATCH_COUNT=0` with a synthetic Chrome argv; direct assignment produced `COUNT=2`, `MATCH_COUNT=1`. Astra independently reproduced this on Windows PowerShell 5.1.26100.9711. This explains R150's all-false `ownedProfileArg` fields, but not whether any descendant survived at terminal observation.
2. **Observed HTTP evidence loss.** R150 raw A1-b reports `opened:false`, `timeout:true` and generic `Received network error or non-101 status code.` The helper's `error` listener stores only the message and waits for `close` or its timer. It does not retain an HTTP status. This source path explains the `INCONCLUSIVE` measurement. The actual Chrome HTTP response versus transport failure remains **UNKNOWN**.
3. **Source-proven false-pass candidates, historical occurrence unproven.** `checkOwnedChrome` does not require a positive exact-profile root witness. Substring matching can match a prefix or unrelated argument. Terminal cleanup filters only profile-bearing Chrome rows, excluding descendants that omit the argument. `closeHandle` discards a timed-out wait result, and the Chrome #1 handle can then be dropped before replacement. These code paths can mask failure; R150's later PID absence cannot reconstruct its terminal cleanup.
4. **Separate product cause.** R139's accepted causal assessment shows a lifecycle contract conflict: isolated capability closes its context, while integrated E06 requires the original context, owner and clock. Pre-action physical no-relaunch enforcement remains unsupported. Repairing R151 evidence capture does not solve that product prerequisite. Historical provider/manual-hold and r3 interruption causes remain UNKNOWN.

## Finite discriminating check and expected result

Question: Can the R150 HTTP upgrade and Chrome ownership evidence defects be repaired and verified offline on the current Windows PowerShell 5.1/Node toolchain without weakening A1/A2 or changing product effects?

Terra creates only new R151 helpers and deterministic fixtures. The source-only check must prove exact ownership with positive root and descendant identity, bounded original-handle waits, and distinct HTTP 4xx/101/ambiguous transport classes. Red reproduction of the historical defect and green successor tests are required. Expected outcome is either `OFFLINE_CAPTURE_ACCEPTED` after independent review, or `BLOCKED_WITH_REASON` naming a failed predicate. No offline result establishes Chrome's actual A1-b response.

After an accepted source review, Root may consider one separately admitted, finite Windows A1/A2 VERIFY within R151. It must retain the real response and cleanup on the actual Windows toolchain. `A1_A2_PASS` only removes an admission prerequisite; `A1_FAIL`, `A2_FAIL`, `INCONCLUSIVE` or `NO_START` preserves R139/M3 holds. No automatic retry follows any outcome.

## Ownership, costs and stop

Root owns the causal assessment, work order, registry and dispatch. Terra owns only admitted source/test/return paths. Astra independently reviews the audit and source outcome. Sol 6.1 Medium provided read-only advice on using Node built-in HTTP upgrade events; it authored no source or authority. R150 spent one Chrome launch and two WebSocket handshakes; its terminal cleanup is UNKNOWN. Earlier r2/r3 attempt costs remain cumulative UNKNOWN. R151 source-only ceiling: 45 active worker minutes, three focused test invocations, one syntax check; any later process-effect budget requires its own admission and is never borrowed from R150.

Stop before BUILD if the packet, audit, registry or independent review disagree. Stop worker work on protected edits, ambiguous tests, budget exhaustion or a need for runtime effects. Stop before any Windows process start unless source review, effect admission, A3 recapture and current gates pass. Missing a product no-relaunch primitive still blocks integrated execution, even if A1/A2 later pass.

## Review and claim boundary

This is a source-grounded causal assessment and intended diagnostic design, not product acceptance, runtime CVF governance evidence or permission to start Chrome. The independent review and work-order gate decide source-only dispatch. R139 BUILD_BLOCKED, M3-05/M3-08 BLOCKED and Facebook/Zalo OA parked remain unchanged.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
