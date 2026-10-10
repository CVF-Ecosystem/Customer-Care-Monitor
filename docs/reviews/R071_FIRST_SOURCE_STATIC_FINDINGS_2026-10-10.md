# R071 first-source static review before Go

Status: BUILD_STATIC_CHANGES_REQUIRED_NO_RUNTIME. Reviewer: Codex /root, independent of Luna implementation. Source: `86e6ff976c32cced25f10d7c8e17b470fdb5106a`, exactly five authorized paths. No Go invocation or runtime failure is asserted. First source remains immutable in Git. This is the pre-execution source gate within BUILD, not final REVIEW/FREEZE.

Root consolidated source, old-test preservation, APO01..12, scope, current seed/record/work order and execution dependencies before accepting one same-scope repair round. Two Analyzer additions are correctly adjacent to existing legacy hooks after successful returns; legacy arithmetic and effects are unchanged. Collector source correctly separates statuses, validates schema/source/value shape, preserves exact int64, snapshots values, derives side totals and handles independent overflow. No product semantic defect established by this static pass; runtime remains pending.

| Finding | Severity and ownership | Required repair |
|---|---|---|
| S1 | P1, Luna NEW test fixture defect | Overflow subtest seeds MaxInt64 then adds0 but expects overflow. Use real valid observations reaching the boundary then a positive addition, preserve healthy boundary+0 and opposite-side totals; exercise both overflow directions. |
| S2 | P2, Luna NEW test sensitivity gap | Incomplete-prefix tests first observe nil metadata, so totals are already absent independently of the response-coverage guard. Start with a known-complete prefix, then begin unresolved/error/unhooked successful calls and assert totals are withheld. Add nil-collector guard and valid-null/malformed-output sibling coverage. |
| S3 | P2, Luna bound-evidence gap | The maximum-counter test only maximizes one field. Measure the full fixed aggregate at maximum counters/numeric widths <=1200bytes; verify fixed response/status counter saturation without wrapping. |

One consolidated repair in the existing NEW test file; optional gofmt within the authorized five paths. Do not change legacy tests, adapter parser, pricing, effects, acceptance, seed, budgets or original packets. Parent supplies findings and capture infrastructure, never source/tests. Preserve first source and initial plan; repaired source/plan must be committed and independently accepted before Go. Current worker/root Go0, maxima8 each (2build+6tests), no automatic retries or reset.

Draft observations of an unused import and mistaken sibling-total expectation were corrected by Luna before the first source commit; they are not retained as committed-source findings or compiler failures. Root flagged contract-sensitive assertions generally before commit, without editing product code. Formatting cleanup is not a semantic defect. Prior missing session outputs remain INDETERMINATE, separate from source quality.

NOT RUN: compile, pure tests, race, DB/full backend runtime, application frontend/visual, provider/network/GitHub Actions. No governance, live Analyzer persistence, wire, billing or full roadmap closure claim. Facebook/Zalo OA parked; prior Git publication unverified and not retried.
