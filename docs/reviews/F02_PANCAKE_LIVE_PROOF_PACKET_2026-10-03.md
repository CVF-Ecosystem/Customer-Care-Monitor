# Pancake live-channel proof packet

Date: 2026-10-03 (Asia/Saigon). Author: Codex ORCHESTRATOR. Status: PREPARED_NOT_DISPATCHED / EXTERNAL_INPUT_REQUIRED. Source baseline: `ec95dbac10963618ae3e4bc74517503860f7c396`. Risk ceiling: R2. Authority: current R033 handoff and standing local orchestration delegation permit this documentation assessment and local commit. This packet grants no BUILD, credential inspection or network execution. R030-R032 local FREEZE remains intact; R022-R024 FREEZE, global F02, provider governance and hosted readiness remain OPEN.

## Intake and design decision

Select Pancake first, on one controlled test page and tenant with an independently prepared expected inventory. Limit the claim to observed live conversation/message retrieval and mapping against that inventory at one exact source revision. A channel GET cannot establish CVF controlling AI; any such governance claim requires its own authorized real AI-provider request/response and independent review.

Use adapter-only execution with in-memory results. Do not run the application sync engine: that route has database/checkpoint/attachment/after-sync effects outside this packet. No media downloads, analyzer, notifications, token rotation, production records or persistent database writes. Storage/checkpoint/dispatch evidence stays inherited from local reviews, not newly demonstrated live.

## Source-derived constraints

Reviewed source: `backend/channels/pancake.go`, [R023 conversation review](CCMAI_RUNTIME_023_F02B_INDEPENDENT_REVIEW_2026-10-01.md), [R030 message review](CCMAI_RUNTIME_030_F02D_INDEPENDENT_REVIEW_2026-10-02.md), and [shared evidence-layer learning](learnings/feedback_sync_coverage_evidence_layers.md). No public provider documentation or live endpoint was queried in this assessment; protocol compatibility remains to verify under the future network authority.

| Observed source behavior | Consequence for the future execution contract |
| --- | --- |
| Conversation paging fixes `until` at fetch start and follows the last physical row ID; `since` filters eligible INBOX conversations. | Record actual fixed `until`, requested `since`, cursor transitions and physical versus eligible rows. Retry fetches may select a different `until`; compare each run against its own controlled inventory/window. |
| Message paging traverses full history using physical `current_count`; `since` filters locally. | A narrow time window does not bound fetched history. The entire selected conversation history must fit the request budget, including explicit-empty terminal pages. |
| Requests insert page access token into the query; 429 permits three retries. | Count every HTTP attempt, including retries. Never record complete query URLs, authorization material or unsanitized transport errors. |
| Response ceiling is 8 MiB; adapter permits 200 pages per fetch and uses a 30-second HTTP timeout. | Apply a stricter aggregate execution budget across every fetch/retry; adapter page limits alone do not implement the proposed 50-request/10-minute ceiling. |
| Constructor uses the default HTTP redirect policy. | Future harness must reject redirects and unexpected hosts before sending a request. Verify this locally before credential use; do not assume the current constructor enforces an endpoint allowlist. |
| Zone-less timestamps parse as UTC. | Expected inventory must include independently known instants/offsets to assess that assumption; mapped timestamps alone cannot validate their own interpretation. |

These are source observations and harness requirements, not reproduced live defects or a dispatched implementation order.

## Required external inputs before a work order

| Input | Required content | Current state |
| --- | --- | --- |
| Controlled tenant/page | One nonproduction tenant/page reference; owner confirms permission to read its test-only history. No customer data. | NOT PROVIDED |
| Credential authority | Explicit permission to use an existing least-privilege test-page credential and its local reference/source. Supply no token in chat or committed files. Token creation/rotation is excluded. | NOT PROVIDED |
| Expected inventory | Independently supplied conversation IDs/types/updated instants and message IDs/sent instants/attachment metadata, with page identity, visibility, retention conditions and provenance. Include excluded old rows and boundary-equal rows where available. | NOT PROVIDED |
| Quiescence | Freeze test-page writes during both retrieval runs, or declare any changes and classify snapshot reconciliation as incomplete. | NOT CONFIRMED |
| Capture handling | Approved local raw-capture location, allowed sanitized artifacts, cleanup/retention owner and deadline; deterministic pseudonymous identifiers. | NOT PROVIDED |

The owner may provide references and conditions together. Missing inputs block live dispatch, not this local planning assessment. Empty results or inventory derived only from the same adapter are insufficient for completeness acceptance.

## Proposed bounded SPEC for the future work order

1. Bind exact 40-hex source SHA, tenant/page reference and immutable expected inventory digest before execution. Independently verify snapshot identity against accepted local source and relevant dependencies.
2. Permit only HTTPS GET to `pages.fm`, path templates `/api/public_api/v1/pages/{approvedPage}/conversations` and `/api/public_api/v1/pages/{approvedPage}/conversations/{approvedConversation}/messages`. Encode approved IDs as path segments; reject other methods/hosts/paths and all redirects. No unrelated health/probe endpoint is authorized.
3. Cap all HTTP attempts at 50 and aggregate duration at 10 minutes, including retries and both traversals; response ceiling 8 MiB per response. Cancel before any request would exceed the cap. Reaching a cap before explicit exhaustion is INCOMPLETE, never PASS. A future inventory that cannot fit requires a newly bounded order rather than silently increasing caps.
4. Run conversation retrieval with exhaustive limit 0, then message retrieval for each expected/observed eligible test conversation. Account for unexpected IDs as discrepancies; do not silently drop them. Repeat against the same quiescent inventory within the same aggregate ceiling. Reconcile raw eligible rows and mapped rows separately so parser/mapping omissions are visible.
5. Record explicit-empty terminal evidence, page number, physical row counts, cursor/offset progress, duplicates, eligibility/filter decisions and per-run expected/observed set differences. Check timestamp instants and attachment metadata without downloading attachments. Missing, extra, invalid or mismapped rows prevent PASS.
6. Record UTC start/end, request sequence, method, sanitized endpoint template, safe parameters, status, response shape/count, pseudonymous IDs, timestamp/mapping outcomes and inventory provenance. Preserve sanitized request/response evidence sufficient to reconcile sets. Raw bodies, text, real IDs, full URLs, signed media URLs and credentials stay out of Git and terminal output. Review sanitized artifacts before commit; retain raw captures only at the specifically approved location until the agreed deadline.
7. Stop on auth/permission errors; no rotation, fallback tenant, unrelated API or retry outside the cap. Do not interpret provider failure or malformed/empty payload as successful exhaustion.
8. Independent REVIEW verifies actual authorized live requests, evidence provenance, set/time/attachment reconciliation, cap/allowlist behavior and disclosure of failures. A worker cannot self-approve R2 live evidence. Choose and record reviewer/worker ownership before dispatch.

## Execution sequence and disposition contract

Current work is INTAKE assessment/design requirements only. Once required inputs and explicit network/credential authority are supplied: rehydrate continuity; record bounded DESIGN/SPEC; issue a new WORK_ORDER and immutable dispatcher seed committed before BUILD. Separate local harness implementation from credential/network execution if needed; authorize exact paths, roles, outputs, failure conditions and commit ownership. First validate cap, redirects, host/path rejection and sanitization locally. Synthetic checks validate the harness only and must never be relabeled live or AI-governance proof. No worker is dispatched by this document.

Future outcomes: PASS only for the exact tested Pancake page/inventory/window/source under recorded conditions; FAIL for a demonstrated reconciliation or compatibility defect; INCOMPLETE for caps, missing provenance/terminal evidence or changing inventory; BLOCKED before requests for missing authority/input/harness preconditions. Any product fix returns to a separately bounded repair and independent review. No outcome here closes Facebook/Zalo, global F02, conversation FREEZE, runtime AI governance, hosted CI, pilot or deployment readiness.

## Assessment result

Concrete packet prepared; external inputs and authorization are still required. New evidence is source/document inspection and local validation only. Live channel/provider requests, credential reads, backend/frontend/runtime/DB tests and GitHub checks NOT RUN. R033 and all other tranche dispositions unchanged. No CVF Web governance bridge is needed for this documentation-only assessment.

## Offline harness update (R034 BUILD, 2026-10-03)

The offline harness required by the aggregate cap/redirect/allowlist/sanitization requirements above now exists as new files under R034 ([BUILD record](PANCAKE_PROOF_HARNESS_R034_BUILD_2026-10-03.md)), pending independent review. This packet remains PREPARED_NOT_DISPATCHED / EXTERNAL_INPUT_REQUIRED: no tenant/page, credential authority, inventory, quiescence, capture handling or network authority was supplied, and no live request, credential read or provider call occurred. Correction recorded from the unchanged adapter: conversation listing uses the `/api/public_api/v2/` path and messages use `/api/public_api/v1/`; a future live work order must allow exactly those.
