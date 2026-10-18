# M2 manual synthetic intervention application contract

Status: ACCEPTED_INTENDED_CONTRACT
Date: 2026-10-14
Tranche: CCMAI-RUNTIME-094
Source base: 30d4de76dd62c092038ec7092ec99a0239076113

## Product outcome and boundary

One selected manual synthetic conversation can produce a persistent, snapshot-bound intervention proposal and a tenant-scoped readable queue/detail. A missing/forbidden input stops before provider construction; an invalid output never becomes a proposal. This is the next coherent M2 step after accepted R092 intended contract and returned R093 wire observations. M3 human review, correction, assignment and outcome remain separate.

Root accepts the Sol Medium design after Luna source intake. Existing snapshot construction/validation and job reservation/finalization are reused without changing their contracts. This is intended behavior, not implementation acceptance. R093 positive invalid-ID/missing-anchor output and its exhausted 2/2 cap remain immutable.

## Enabled route and tenant authority

Use `JobType=intervention`, enabled, `ScheduleType=manual`, one selected conversation and synthetic/demo channel marker verified from database rows. Preserve ordinary job creation whitelist; R094 fixture provisions the intervention job, so user setup/UI is not delivered here. No real-channel/customer data, outbound notification or scheduled/agent intervention effects.

`POST /api/v1/tenants/:tenantId/jobs/:jobId/intervention-run` requires a known tenant role and jobs:w, and validates server-owned job/channel/conversation scope. `GET /api/v1/tenants/:tenantId/intervention-cases` and `GET /api/v1/tenants/:tenantId/intervention-cases/:caseId` require intervention_cases:r. Owner/admin retain existing permission semantics; member missing/malformed permission is denied. Cross-tenant detail is 404. Bound queue pagination/filter/order consistently. Do not enable review/assignment/outcome mutations.

Legacy Analyzer execute and reserved execution reject intervention jobs before provider initialization. Legacy HTTP trigger/test-run and job mutation must not bypass manual/synthetic/profile restrictions. Unknown profile/policy/configuration fails closed. Default running application receives no new executable provider configuration: the intervention service is disabled/unconfigured unless server-owned explicit configuration is supplied; clients cannot choose endpoint/key/cap/policy or reset budget.

## State and source contract

Eligibility: ELIGIBLE, WAIT_DATA, DENY. Execution: NOT_RUN, RULES_ONLY, LLM_PENDING, LLM_SUCCEEDED, ERROR, CANCELLED, USAGE_UNKNOWN. Proposal: NOT_CREATED or PENDING_REVIEW. Case: OPEN or WAITING_ON_DATA. Keep these axes separate; successful HTTP is not accepted proposal or resolved case.

Missing/empty/incomplete snapshot or required business evidence yields WAIT_DATA with reason and requested next action. Missing owner/SLA remains null/unknown, never an invented deadline. Denied permission, source/profile or unsafe disclosure yields DENY. Unknown input cannot become ELIGIBLE. Synthetic-only policy is explicit; no general real-data DLP/rights guarantee is claimed.

Bind every execution to tenant, actor, job/run, conversation, use-case/policy/prompt/schema version, AnalysisSnapshot ID/digest and exact canonical message IDs. Freeze transcript contents in memory corresponding to manifest hashes; validate every quote against that frozen snapshot, with Unicode offsets when supplied. Never silently repair prefixed/unknown IDs. Recheck current source plus job ownership/cancellation before publication; source change/deletion/appended message or canceled run cannot publish stale proposal. Preserve source/history; do not rewrite old snapshots/results.

## Physical call admission and output

A server-owned persistent budget has an immutable cap for its identity. In a short MySQL transaction, atomically consume a slot and record run/request identity before the HTTP attempt. No DB lock spans the HTTP call. Contending requests cannot exceed cap. Crash-left/timeout/ambiguous transport or failed settlement remains consumed and unknown; no automatic refund/reclaim/reset. One bounded attempt only, no redirect/retry/fallback. A call-count cap is not price, billing or token reservation proof; numeric unknowns stay unknown. Zero real-provider requests are authorized in R094.

Provide a scoped no-retry OpenAI-compatible adapter option without changing existing provider defaults. Timeout, input/output size and completion-token caps are explicit, errors sanitized, raw authorization/header/key excluded. Local transport uses loopback only. A future live source-bound packet must independently validate endpoint/model/options and global request cap; R093 permission/counters do not authorize it.

Proposal output carries schema, conversation/snapshot digest, kind, explanation, recommended next action and evidence. Require complaint/customer-wait anchor appropriate to proposal kind, canonical customer message ID and exact quote; fabricated/missing anchor, foreign digest/ID, malformed output or stale source produces a sanitized rejection receipt and no proposal. Provider success with invalid output remains invalid. Persist case/snapshot/proposal/receipt transitions with checked affected-row counts and atomic publication. Usage/model/vendor/price presence remain KNOWN/UNKNOWN accurately. Do not expose raw model response or transcript in public receipts; expose validated evidence quotes and safe reason codes only.

## Acceptance and evidence

M2A01 manual synthetic enabled path persists exactly bound snapshot/proposal and readable case. M2A02 permission/foreign source/DENY and WAIT_DATA produce zero calls. M2A03 cap contention, repeated/recreated caller and ambiguous outcome never reset consumed budget. M2A04 invalid IDs/quotes/offsets/anchor/schema/digest produce no proposal. M2A05 stale source/cancellation blocks publication. M2A06 legacy paths reject intervention job; normal job behavior/default adapter remains compatible. M2A07 tenant queue/detail permission, pagination and safe response shape. M2A08 failure persistence keeps truthful states and sanitization. M2A09 owned disposable MySQL cleanup and protected existing resources unchanged.

Use actual MySQL fixture plus loopback HTTP for software behavior, with positive, negative, concurrent and failure cases. Preserve raw failing test receipts. Reviewer independently evaluates returned evidence, runs focused checks with information gain and a named regression detector where useful, and confirms restoration. No mock result is governance proof. Real application provider/governance evidence, browser M3, business/pilot thresholds, billing, hosted CI and deploy remain OPEN.

## Finite route and stop

One source/application packet, up to two consolidated repairs, then independent review. Native tools already present: Go1.27, Node24.19/npm11.17, Docker29.8 and cached mysql:8.0; use offline module cache, no install/pull. Worker and reviewer each own at most one labelled loopback-only disposable MySQL fixture, never existing ccma-app/nginx/db. Stop unsupported dependency, fixture ownership/cleanup failure, uncertain budget, scope widening, or third repair without independent root cause. Next: separately bounded live M2 approval/evidence, then M3 human workflow; no endless support successor.
