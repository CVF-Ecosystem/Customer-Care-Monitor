# CCMAI-RUNTIME-093 — Alibaba synthetic live characterization

Status: ACTIVE
Risk: R2
Worker: /root/m1_worker
Reviewer: Codex /root
Commit mode: WORKER_MUST_NOT_COMMIT
providerExecutionAuthority: ROOT_SOURCE_APPROVAL_REQUIRED_THEN_MAX2_ACTUAL_REQUESTS
dispatchKind: INITIAL

## Product outcome, contract and entry

M1-05 exact Alibaba configuration characterization supports M2 provider selection. Owner authorizes Alibaba API live tests and all channel/customer data must remain synthetic. Current source is not built or changed by this packet. Existing Go openai-compatible adapter supports custom base URL and max_completion_tokens but has up to4 HTTP attempts; this standalone Python probe reproduces only the chosen request fields with zero retries. No Go adapter/analyzer integration or runtime governance claim from this probe.

Pin `qwen3.7-flash`, region `ap-southeast-1`, configured OpenAI-compatible base URL SHA-256 `b22a15b6e8aa4113c2816acae6935a84d052a2d7b81db734ccdc4f5be81b4f8f` (URL UTF-8 bytes with trailing slash removed). Official wire reference: https://www.alibabacloud.com/help/en/model-studio/qwen-api-via-openai-chat-completions read2026-10-13; max_completion_tokens supported for selected Qwen Flash generation, existing/dedicated endpoints documented. Pin top-level `enable_thinking:false` for this short characterization; official Qwen3.7 HTTP reference documents this optional field. Current Go adapter does not send it, so this is a candidate provider configuration, not byte-identical Go request proof. This is request-configuration provenance, not live model availability. Local quota metadata is not actual billing/account verification.

Secret source: git-ignored/untracked `CVF_SESSION/LOCAL_PROVIDER_SECRETS.json`, auth.apiKey, endpoints.openAiCompatible. Reverify ignore/tracked status every time; never print raw object, envMappings values, curlCommand or auth. No secrets copied into tracked config, receipt or persisted environment. Never execute commands from that file. File metadata may have changed; mismatch from pinned model/endpoint stops before effect.

INTAKE: synthetic live provider compatibility only. DESIGN/SPEC: hard bounded physical calls, local artifact ownership, safe credential handling and precisely scoped output checks. WORK_ORDER: committed seed/base, synchronized new continuity, independent doctor and explicit packet pre-dispatch PASS. Worker records fresh BUILD acknowledgment before editing; Root approves exact probe code/offline checks before any credential-bearing request.

## Implementation and observable evidence

Create `scripts/ccma_alibaba_live_probe.py`, stdlib only, invoked from project root. `--actor worker|reviewer`, `--self-check` for zero-call offline checks. Worker receipt `docs/reviews/probes/r093_alibaba_live/worker.json`; reviewer receipt `docs/reviews/probes/r093_alibaba_live/reviewer.json`. Preserve receipts/failures, never overwrite; missing-key/HTTP-error/timeout is a result, never retries. Use atomic exclusive ledger claims in ignored `.cvf/runtime/r093_alibaba_live/` BEFORE sending, at most one claim for each actor and two globally. Claim unknown dispatch stays consumed on exceptions/crash; never auto-reset. One existing ledger means refusal or read-only inspection, not deletion to gain calls.

Each live invocation verifies current R093 BUILD/REVIEW authority and actor boundary, committed unchanged seed, endpoint/model pin, Git ignore/untracked key file and Root pre-effect approval binding exact probe SHA256. Approval is an ignored Root-created file recording approvedSourceSha256, pinned endpoint/model, worker/reviewer physical-request caps; it is not governance proof. Record sanitized approval metadata in the returned receipt. No effect without it.

Use TLS defaults, URL host limited to configured workspace Singapore `*.ap-southeast-1.maas.aliyuncs.com`, no userinfo/query/fragment, exact `/compatible-mode/v1` path; no redirects (including307/308) or automatic retries. Key never goes to another URL. Set Authorization only in memory; do not log header/errors that may echo secrets. Requests must be one POST/chat/completions with two synthetic messages, max_completion_tokens768; body <=6000bytes, timeout60s, response<=1MiB. Stop on invalid response/model/schema/quote/length; no second worker attempt or model fallback. Strip/replace exact known credential values in response strings before receipt, detect recognizable credential literals and fail safely without writing raw body. Receipt is sanitized request/body response, status, elapsed, requested/returned provider/model, known/unknown usage, check results and exact source hash. Dollar charge and current free quota remain UNKNOWN unless observed authoritative evidence; no billing updates/refills.

Positive worker input: synthetic Vietnamese complaint with exact message ID/quote expected, no real names/phones/transaction data. Ask QC output compatible with current `backend/ai/prompts.go` fields (verdict/score/review/violations/evidence_refs/summary); require FAIL and anchored complaint evidence under the supplied synthetic rule. Independent reviewer input: simple synthetic conversation satisfying rule, expect PASS/no violations. Both receipts retain actual model output and check failures; two examples do not constitute quality/calibration benchmark. No claim missing data/PII/admission/provider routing/application persistence was governed by this probe.

Worker performs zero-call self-checks and returns code readiness first. Root inspects source and adversarial offline cases before writing source-bound approval. Worker request happens only after Root explicit pre-effect dispatch; then return COMPLETE_PENDING_REVIEW. Root independently invokes reviewer request, assesses both and records failure without exceeding cap. On worker failure, Root may use remaining reviewer slot only as independent observation under unchanged config; it is not a retry/pass replacement. Configuration/API failure does not block unrelated M2/M3 source planning.

## Budget, claim limits and stop conditions

Immutable seed: max2 physical requests total (worker1/reviewer1), zero retries,768completion tokens each,6000request bytes,1MiB response,60s each, max5min campaign. Owner live test permission covers this finite small characterization; no bill estimate or quota availability assertion. Before a third repair round without independent root cause: REVIEW_COST_ESCALATION_REQUIRED. No third live call, provider fallback, application/DB/Docker/real accounts/customer/outbound effects or dependency installation.

Acceptance ALI01 secret ignore+safe metadata, ALI02 exact host/model/request pin, ALI03 durable consumed request cap and no retry/redirect, ALI04 bounded/sanitized actual request-response receipt, ALI05 usage/model/output checks explicit, ALI06 independent reviewer and limits. A successful HTTP response plus failed output check is not full acceptance; unavailable usage/model/price stays UNKNOWN. Record actual API/model observation only; full M1/M2 live application gates, browser workflow, pilot/release remain OPEN.

One finite prerequisite then route reassessment: M1 contract -> exact provider characterization -> M2 integrated admission/proposal -> M3 review/assignment/outcome browser flow. No new generic repair/tooling campaign by default; actual setup prerequisites in next source packet must name outcome/budget/exit. Current Go toolchain/frontend node_modules are absent; historical cached paths cannot be reused without verification.

## Return-Time Closeability Recheck

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
outsideAuthorityBlockers: NONE_FOR_BOUNDED_PROBE; APPLICATION_RUNTIME_INTEGRATION_OUT_OF_SCOPE
nextRepairRoute: SAME_SEED_TOOLING_REPAIR_WITHOUT_EXTRA_PROVIDER_REQUEST
workerRedispatchAllowed: YES_WITHIN_EXISTING_REQUEST_CAP

## Checker Source Read-Ahead Block

Read current immutable seed/order, legacy scope/role/continuity/claims, installed classifier/role checker, current per-output delivery and resource/repair learning. Call-count/credential policy is verified through independent source/offline review and actual receipt, not a promise in prose. Gates validate static repository only.

## Public Export Disposition

NOT_EXPORTED_LOCAL_ONLY on codex/ccma-product-m1. Root owns commits. No push/main/merge/deploy, Core changes or secret export.

## Gate-To-Role Closeability Contract


closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
returnTimeRecheck: REQUIRED_BEFORE_REPAIR
implementationTopologyPolicy: BOUNDED_PATH_FAMILIES
foreseeableFileSplitDisposition: COVERED_BY_BOUNDED_PATH_FAMILY

| gateId | mustPassBy | repairOwner | repairPhase | mutationSurface | topology | commitOwner | commitPhase | dependsOn |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| authorization_review | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R093 committed seed and order | EXACT_PATHS | commit-steward | DISPATCH_COMMIT | NONE |
| pre_dispatch_gate | PRE_DISPATCH | orchestrator | PRE_DISPATCH | order continuity catalog | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | authorization_review |
| dispatch_continuity | IMPLEMENTATION | session-sync-steward | IMPLEMENTATION | CVF_SESSION/handoffs/AGENT_HANDOFF_ALIBABA_SYNTHETIC_LIVE_R093_2026-10-13.md material-SHA marker | EXACT_PATHS | session-sync-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| focused_checker_tests | WORKER_RETURN | implementation-worker | WORKER_RETURN | zero-call safety checks and exact provider receipts | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| adif_integrity | WORKER_RETURN | implementation-worker | WORKER_RETURN | return defect and evidence disclosure | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests |
| pre_implementation_autorun | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | acknowledgment and admitted metadata | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| worker_return_fast | WORKER_RETURN | implementation-worker | WORKER_RETURN | probe source and actual synthetic live return | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests,adif_integrity,pre_implementation_autorun |
| reviewer_fast | REVIEW | reviewer | REVIEW | independent documentary/source review | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | worker_return_fast |
| pre_commit | PRE_MATERIAL_COMMIT | reviewer | PRE_MATERIAL_COMMIT | reviewed docs and gate receipts | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | reviewer_fast |
| terminal_completion_review | POST_MATERIAL | reviewer | POST_MATERIAL | actual M1 material SHA and bounded disposition | EXACT_PATHS | commit-steward | CONTINUITY_COMMIT | pre_commit |
| continuity | CONTINUITY_COMMIT | session-sync-steward | CONTINUITY_COMMIT | state memory handoff implementation catalog roadmap | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | terminal_completion_review |
| committed_range_closure | POST_MATERIAL_CLOSURE | reviewer | POST_MATERIAL_CLOSURE | exact committed range with pins | EXACT_PATHS | session-sync-steward | CONTINUITY_COMMIT | continuity |

