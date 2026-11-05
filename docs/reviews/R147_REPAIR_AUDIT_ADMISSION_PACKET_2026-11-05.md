# R147: separate R139 repair audit and finite A3 VERIFY admission

Status: DRAFT
workOrderStatusGrammar: cvf.workOrderStatusGrammar@1.0.0
Risk: R2
Worker: Root/Local audit author
Reviewer: Independent subagent reviewer, separate from Root audit author
Authority: `CVF_SESSION/authority/CCMAI-RUNTIME-147.json` (seed commit `07ff318312861fcd9807e874dce0ffd669bda980`)

## Outcome

Root authors a successor to the retained R139 causal assessment, preserves r2/r3 UNMET history, UNKNOWN causes and cumulative cost, and admits only a 20 active minute A3 package-metadata comparison under R144. An independent reviewer evaluates the new audit and bounded admission before the registry pointer changes. R139 stays BUILD_BLOCKED and M3-05/M3-08 stay BLOCKED.

## Source and decision boundary

The observed failure is the unmet integrated R139 product outcome. The A3 discriminator addresses a narrower unresolved prerequisite: installed Windows `chrome-devtools-mcp@1.9.0` `package.json` hash differs from the R141 verified npm tarball while four JS files match. The difference may be serialization or semantic metadata; its cause is UNKNOWN until exact bytes are compared. The R144 packet and local capture bound the question. A missing retained R141 tarball stops the worker and requires separate source-acquisition admission.

The successor audit may set `DIAGNOSTIC_ONLY` with exactly one `VERIFY` action, selected M3-05/M3-08, no IMPLEMENT and no runtime effects. Root pins the exact R144 packet hash, independent review evidence and new audit hash in the existing registry. Reviewer checks history/cost preservation, observed versus hypothesized cause, finite budget, stop/reopen conditions and precise R144 join. No new network fetch, Chrome/MCP/browser/provider/DB action, product acceptance or physical-relaunch proof is admitted.

## Retained failed admissions

R145 seed `d3b04cd4` failed authorityKind grammar; R146 seed `44907dc2` had an incorrect R145 review-path allowlist. Both gate failures occurred before audit or worker action. Their commits remain immutable, confer no scope, and do not reset historical R139 outcomes or budget. R147 seed is the corrected authority.

## Close condition

R147 closes after independent review, static gate acceptance of successor audit and registry, committed-range validation, and a separate R144 pre-dispatch PASS. If any check fails, R144 remains DRAFT and failure is retained. Root is sole commit steward. Claude receives final R144 packet through operator.
