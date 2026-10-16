# Root connected browser verification

Status: PRE_EXECUTION_PLAN
Risk: R2
Worker: /root/r108_browser_luna (implementation and supervisor)
Reviewer: Codex /root (independent source review and connected browser observations)
Disposition: NOT_RUN_PENDING_SOURCE_VALIDATION_AND_RUNTIME_ADMISSION

Root uses the existing connected in-app browser through mcp__cua_repl only.
HTTP click/capture capability was observed in the retained R108 receipt; current
inventory and visibility/viewport capabilities remain available. Product access
is still unproven for this new campaign. Browser failure stops owned runtime and
is returned to the owner for transfer. No standalone automation or network policy
changes. The unrelated about:blank tab seen in inventory is not assumed owned.

## Sequence and finite effects

Native four-action diagnostic precedes product access and uses a separate case.
If native fails, every UI case remains NOT_RUN. If native passes, Root uses the
new ready tenant/case and the literal synthetic fixture login from the admitted
test source. The owner and reader are separate sessions through normal UI.

| Case | Observation |
| --- | --- |
| B01 | Owner login, queue, pending case, original proposal and initial history. |
| B02 | Open the same pending case in a second owned tab; first tab records a correction and readable reason/explanation/next action. |
| B03 | Second tab submits its stale pending action once; observe real conflict or record precisely the local guard branch exercised. |
| B04 | Refresh first tab, assign an offered tenant user with a future deadline, reload and verify persistence. |
| B05 | Record a distinctive synthetic outcome, reload, verify terminal state/filter and retained proposal/history. |
| B06 | Reopen with a future deadline, reload, verify assigned state, active outcome cleared and exact previous outcome retained in readable history. |
| B08 | At 390 by 844, inspect queue/detail/history and the owner form; reset viewport afterward. |
| B10 | Existing waiting-data case only: assign offered data owner/deadline/next step then escalate once; observe still-waiting state, source limitation and absent completion actions. |
| B07 | Sign out owner and sign in the ready reader; queue/detail readable, write controls absent. |
| B09 | NOT_RUN if no documented offline capability can be exercised; currently only visibility and viewport are exposed. |

Three owned product tabs maximum, fourteen UI POST submissions including auth and
rejected writes. Expected lifecycle writes: correction, stale confirmation,
assignment, outcome, reopen, data assignment, data escalation = seven. Sign-in,
sign-out and tenant-selection effects count toward the same fourteen ceiling;
reserve before each submission and stop before exceeding it. Record actuals,
never infer that a click sent a request without supporting evidence. Native
action POSTs are counted separately (four maximum). No direct API mutations.

Capture original DOM/screenshot bytes before and after meaningful actions,
actual tool commands/results, current source/binary/config pins, versions,
actor/time/reason, displayed wording and any limitations. Root does not edit
source during the admitted runtime or commit while its HEAD pin is active.
An observed passing current REOPEN does not retroactively resolve the historical
R107 503 root cause. Dashboard baseline, full product and live AI governance
acceptance remain outside this packet.
