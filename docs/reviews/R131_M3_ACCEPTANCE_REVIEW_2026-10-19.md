# R131 — Independent M3 acceptance review

Status: REVIEWER_ACCEPTED
Risk: R1
Worker: Luna Max evidence analyst
Reviewer: Root independent reviewer

## Independent checks before the worker return

Root read the current M3-01..08 roadmap and locked R101 H01..H08 contract, current queue/detail/action/UI source, current native/UI test definitions, and the accepted R126/R127/R128 dispositions. No product test campaign or provider call was run in R131.

The R128 archive SHA-256 matches its manifest; all 36 listed entries match their hashes. Original native, frontend-test, frontend-build and browser exit receipts are all 0. The original native log has the seven named top-level PASS results and no SKIP; reader assertions show named assignee/actor, no unnamed fallback, zero action selectors and zero save buttons.

The 414 archived source identities remain applicable to the current checkout: 411 raw SHA-256 matches and three frontend intervention files matching after CRLF-to-LF normalization, with zero content differences. A raw-byte equality claim for all 414 files would be incorrect. Historical source commit IDs remain receipt identities after R130 history rewriting; the hash comparison establishes present content applicability.

Source read-ahead finds these boundaries to check in the returned matrix:

- Queue uses `created_at ASC, id ASC`, total before cursor, state filters and bounded look-ahead pagination. UI says oldest first and shows due/overdue. Basic list proof does not establish multi-page count/filter coverage or a business urgency ranking.
- REJECT exists in the action implementation, but the current backend test search found no positive REJECT case. CORRECT/ASSIGN/RESOLVE/REOPEN proof does not prove that unexecuted branch.
- Backend data-revision and rerun-request routes exist; the current intervention frontend API/view exposes neither. WAIT_DATA ownership/escalation and backend rerun evidence do not establish a complete user-facing supplement/request flow.
- The locked R101 scope sends no notification. No notification retry integration should be invented solely to manufacture an acceptance requirement.
- Offline B09 remains NOT_RUN on the unsupported browser control. Its checklist permits that label; it is neither evidence of a product defect nor, by itself, a universal blocker for every bounded M3 result.

## Consolidated pre-acceptance findings — round 1

1. M3-03's claimed absence of executed two-reviewer proof is incorrect. Root streamed the retained local `go-test-r125.json` from the R126 archive without extraction, verified its SHA-256 against `latest_package_coverage_manifest.json#/packageProvenance/.../rawSHA256` (`a5f7abd409d6319cfe1b351c8d444f9641c6e11c5f1ecfa27138192c5cab2ff0`), and found `TestR101CompetingReviewers` run then PASS, elapsed 0.64s. The full original run still exited 1 for the separately documented engine/MCP failures; the R126 accepted derived whole-package provenance retains the complete passing handlers package. The exact competing-reviewer test function and `backend/engine/intervention_m3.go` blob are unchanged from R126 source `9ab54062976de181ffbb910be44393021d1bc3db`. This is applicable inherited native evidence, not a fresh R128 concurrency run. Correct the false gap without rerunning the campaign or promoting the original full run to green.
2. The B01..B09 browser checklist is in `docs/reviews/planning/R101_CLAUDE_BROWSER_EVIDENCE_HANDOFF_2026-10-15.md`; the locked R101 SPEC contains H01..H08. Correct the combined source citation.

All eight rows were compared with the current source and accepted evidence. The finite request-only WAIT_DATA UI continuation is supported. The worker owns these two documentation corrections under unchanged R131 authority; no new product/runtime action is needed.

## Disposition

REVIEW_PASS for the corrected worker matrix at bae7a8182fefabbf76f7882b212ce2cb114b877d (the full Git build SHA in the tranche record is authoritative). Both round-1 findings were corrected by the worker under the same authority; Root removed trailing Markdown whitespace only as commit steward. M3 remains PARTIAL: M3-03/04/06 are accepted within their evidence bounds, M3-07 is not applicable with the locked no-notification reason, and M3-01/02/05/08 retain their listed gaps. R131 makes no new product/runtime governance, provider quality, hosted CI, pilot or release claim. The next finite product move is the request-only WAIT_DATA UI slice below.

## Proposed finite product continuation — intended only

M3-05 is the concrete user-facing integration gap. A bounded successor can expose existing data-revision and rerun-request APIs for WAIT_DATA cases: a permitted reviewer records newly synchronized source with a reason, sees the resulting local observation honestly, and requests reassessment of the latest complete revision. Readers can see the saved follow-up history; creation of a reassessment request also requires the backend's `jobs:w` permission. The UI must explain that saving a request does not start an AI call. It must not expose grant/configuration identifiers, invent an approve/execute API, modify channel data, or claim channel completeness.

Source scope should be frontend transport/types, one follow-up component, its integration in the existing case view, VN/EN text and focused UI structure/RBAC/lifetime checks. Preserve parent route/session/permission lifetime and single-submit/no-replay behavior. A conflict or uncertain write requires an explicit reload; source revision and request state must come from server responses rather than local success assumptions. Keep all original proposal and human history intact. Backend behavior, provider/grant activation and real execution remain outside that frontend slice.

Any successor requires its own immutable R2 dispatcher authority, current SPEC/work order and independent Root review. A paid/provider campaign needs new bounded applicable authority; consumed historical calls are unavailable. This paragraph grants no implementation or runtime effect in R131.
