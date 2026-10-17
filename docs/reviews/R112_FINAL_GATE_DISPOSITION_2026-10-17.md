# Final static checks and runtime disposition

Status: RECORDED
Risk: R2
Worker: /root/r108_browser_luna
Reviewer: Codex /root
Disposition: STATIC_CHECKS_ACCEPTED_RUNTIME_HOLD_TRANSFER_REQUIRED

Runtime outcome stays FAILED before fixture startup. B07/B08/B09 and current
rendered wording are NOT_RUN; final fixture/protection state NOT_OBSERVED.
Passing static checks do not fill those gaps or close the three Dashboard
baseline failures. No FREEZE/product/governance/hostedCI acceptance.

| Check | Actual result and qualification |
| --- | --- |
| R110 source validation |26PASS/3knownDashboardFAIL, actualexit1; typecheckPASS. Exact baseline qualification retained. |
| R110 worker return |First static gate captured6PASS while110ACTIVE, before later blocked status. |
| R111 dispatch preparation |Profile5PASS but legacy tranche scope FAIL; never dispatched. Immutable seed retained. |
| R112 initial preparation |Legacy phase/transition FAIL for provisional111PARKED; corrected to REVIEW_PENDING→CHANGES_REQUIRED/BLOCKED_WITH_REASON, recheck7PASS. |
| R112 source binding |32functions/classes and template identity parity,11source pinsPASS; no frontend tests/runtime. |
| R112 worker-return first capture |Wrapper TextEncoder error; exit/output/control lines NOT_CAPTURED. No assumed pass. |
| R112 worker-return authorized static recheck |exit0,6PASS including active APPL; actual earlier return SHA pinned. Final result paragraph appended afterward and independently reviewed. |
| Final reviewer-fast |exit0,7mandatory control lines; APPL NOT_APPLICABLE_WITH_REASON because order blocked, six other controlsPASS. Final worker/report bytes supplied. |
| Final legacy preflight |exit0,7/7PASS. |
| Final workspace doctor |exit0,27/27PASS; historical manifest/Core pin warning retained. |
| Final pre-commit |exit0,6mandatory control lines; blocked APPL NOT_APPLICABLE, five other controlsPASS. Full exact-packet receipt retained. |

Full Root final receipts are under probes/r112_root/final_checks. Earlier
captured checks, including FAIL, are under predecessor_checks; initial110
continuity/catalog preparation output was not separately exported and remains
in the task transcript, with the defects described in R110_DISPATCH_REVIEW.
No claim is made that every conversation tool call has a raw file export.

Evidence representations are explicit: raw ZIP preserves physical captured
bytes, SHA manifests bind physical files, staged-byte contract records Git
normalization differences. Binary58,031,616bytes/cb73ee13 remains excluded
with path/hash, not rebuilt. No overwritten old reports or seed/counter reset.
Root synchronizes blocked REVIEW continuity and returns the preserved packet
for another stable connected agent under fresh bounded authority.
