# R147: separate R139 repair audit and finite A3 VERIFY admission

Status: CLOSED
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

## Review return before material commit

Independent subagent review `docs/reviews/R147_INDEPENDENT_A3_AUDIT_REVIEW_2026-11-05.md` accepted the exact diagnostic-only audit. Root pinned its normalized SHA-256 `c338137496722f72a9c53025867761d442c0ce2f9038029ceb5968ef80ce8ef2` in the successor audit, and pinned the successor audit hash `9f0a083d6dfab2e4bcd0182d6d0aabf66c4231bf35fe5c47f023f627dd53e7e9` in the existing M3 repair registry. The installed profile R144 pre-dispatch returned PASS 6/6; route was `NOT_APPLICABLE_WITH_REASON` because this packet declares no dispatchKind. This is a static admission, not an A3 result or R139 product proof.

## Final bounded disposition

Independent review and the committed R147 material range (`07ff3183` to `aa226a88`) passed portable 7/7 and installed profile 8/8 static gates. An initial local pr-ci invocation used incorrect manually supplied profile pins and was refused at stage 0; the corrected invocation used the exact three pins in `.github/workflows/cvf-downstream-gates.yml` and passed 8/8. This failure is retained; it did not change the installed pack. R147 closes as documentary `DIAGNOSTIC_ONLY` admission. R144 still requires its own committed dispatch transition before Claude begins.
