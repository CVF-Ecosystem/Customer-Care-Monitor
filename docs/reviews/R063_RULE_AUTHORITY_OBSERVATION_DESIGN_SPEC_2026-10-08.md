# R063 candidate — Rule and authority observation DESIGN/SPEC

Date: 2026-10-08 (Asia/Saigon). Author Codex `/root` ORCHESTRATOR / SPEC_AUTHOR. Status: SPEC_DRAFT_READY_FOR_BOUNDED_WORK_ORDER. Document risk R1; future implementation R2 with independent worker/reviewer. Candidate only: active R062 remains FREEZE; no R063 work order, authority seed, implementation dispatch or runtime budget. Baseline source `d10e164249b5e7a87206d1d3bc5aec43a1a8ac2f`, independent review `99052ab2d9bc76e09ca751fa40d4a50044b6098c`, local closure `ce2a00fa1884a82f7317e808ae9995d90d2ad299`.

## INTAKE: source truth and missing authority

| Inspected source | Verified boundary |
|---|---|
| `backend/db/models/job.go`, Job | QC RulesContent and SkipConditions, classification RulesConfig are mutable strings. UpdatedAt is an update timestamp; no immutable policy/rule version, approval record or WAIT_DATA ownership/deadline exists here. |
| `backend/ai/prompts.go`, BuildQCPrompt / BuildClassificationPrompt | QC consumes RulesContent plus SkipConditions; classification consumes RulesConfig. SKIP is a model instruction/result, not a deterministic pre-provider admission rule. |
| `backend/engine/analyzer.go`, executeReserved / runBatchMode | A by-value Job supplies prompts on single/batch paths. Reservation mismatch/reuse rejects before collectors; prepared-empty path can finish without constructing a prompt/provider. Analyzer has no authenticated actor/permission decision envelope. |
| `backend/engine/job_run_ownership.go` | Reservation/ownership binds tenant/job/run and serializes execution. Ownership success does not certify a user's right to execute or human override approval. |
| `backend/api/router.go`, job routes; `backend/api/middleware/tenant.go`, PermissionDenial | HTTP checks route-specific rights before handlers. Permission data is not passed as a verified execution receipt into Analyzer. |
| `backend/mcp/handlers.go`, toolPolicies / authorizeToolCall | MCP cqa_trigger_job requires jobs:w and messages:r; owner/admin/member handling belongs to that entry path. This differs from HTTP route requirements; do not merge the two into an invented shared approval. |
| `backend/engine/scheduler.go` | Cron/after-sync invoke Analyzer without HTTP/MCP user authorization context. Do not label these paths human approved. |
| Existing preparation/execution receipts | Describe local prepared source and logical invocation, not rule revision, permissions, eligibility, WAIT_DATA workflow or policy enforcement. Stored progress prefix can differ from returned terminal receipt. |

Read-only source assessment, no provider/config/credential/customer/database/network use. No runtime reproduction or CVF AI governance claim. Full S2/global F02/live/provider/billing/hosted remain OPEN; Facebook/Zalo OA parked.

## DESIGN: choose the smallest truthful dependency

First candidate adds `rule_observation` to existing Summary and preserves source_preparation, source_execution and scalars. It identifies the exact rule input in the Job value used by Analyzer. No rule engine, provider admission, permission recheck, rule interpretation, auto-skip, WAIT_DATA state transition, cache invalidation or schema migration. This is trace groundwork before separately designed admission policy.

Receipt version `ccmai.rule-observation.v1`, scope `analyzer_job_input_only`. Sanitized UUID tenant/job/run binding and fixed recognized job_type. `rule_authority` is always `JOB_INPUT_OBSERVED`, never APPROVED. `policy_version_status=NOT_AVAILABLE`, `permission_status=NOT_OBSERVED_AT_ANALYZER`, `wait_data_status=NOT_IMPLEMENTED`. These describe missing proof, not allowed/denied decisions. No role, user, permission JSON, human approval ID or scheduler origin guessed from context. Existing run success/error and model SKIP keep their existing meanings.

Fingerprint status distinguishes OBSERVED, UNSUPPORTED_JOB_TYPE and METADATA_INVALID. Unsupported types have no fabricated digest and continue existing execution behavior. Empty inputs are observed empty bytes, not absent authority or implicit default rules. Known job types use only effective rule fields: qc_analysis uses RulesContent and SkipConditions; classification uses RulesConfig. Inactive fields do not affect the digest. Do not normalize whitespace, reorder JSON, parse/repair invalid JSON or mutate the copied Job. Different effective bytes must produce different fingerprints; semantically equivalent JSON can deliberately differ.

Fingerprint contract: SHA-256 of ordered, length-delimited byte fields. For every string append unsigned 64-bit big-endian UTF-8 byte length then exact bytes. Ordered fields are domain `ccmai.rule-input.v1`, recognized job_type, then its effective fields in the order above. Output lowercase 64-hex. Length framing avoids separator ambiguity; JSON marshal/string concatenation are not substitutes. This identifies observed content, not a policy release, prompt-template version, authorization, model version or a cache key complete enough to reuse inference.

Do not hash the generated prompt/transcript or include raw rules, skip conditions, names, timestamps, arbitrary job type, input/config, output, SQL errors, credentials or permission data. A content digest is not anonymization: low-entropy rule inputs may be guessed. Keep the digest in the existing tenant-scoped Summary, which ListJobRuns already serializes under jobs:r. ListJobs exposes current rule strings to the same jobs:r audience; the new historical digest can reveal equality or allow guesses about previous rule inputs. The bounded draft accepts this residual historical correlation at R2; it is not anonymization. No new endpoint, permission broadening, logs/public export or raw input retention. No secret/key management is introduced.

Initialize after reservation validation/consumption and alongside existing collectors, using the same copied Job as prompt construction; no DB reread. Once initialized it is immutable for the run; later DB edits cannot replace it. Invalid UUID or inconsistent job/run binding suppresses identifiers and marks metadata_incomplete, using existing sanitization contract. Observation cannot read unrelated tenants. One run-level record, no per-conversation growth; serialized rule_observation budget 2 KiB, fixed enums/UUID/digest only. No new queries, schema/provider changes or source-selection decision.

Thread through all existing initial/progress/provider-error/final/early/panic Summary writes. Rejected reservation has no run receipt; no-work/provider-construction failure may still describe observed job input without claiming rules were submitted. Stored-prefix/returned-terminal limitations remain explicit: presence in a returned object never certifies durable terminal persistence. Old receipts and scalar tests require a consolidated contract audit before dispatch; do not silently break or broadly rewrite existing assertions.

## SPEC: acceptance and falsification plan

| ID | Required future proof |
|---|---|
| RO01 | Known single QC binding matches exact framed RulesContent/SkipConditions digest; all identifiers sanitized. |
| RO02 | Classification uses only RulesConfig, both single and batch; inactive-field edits do not alter digest. |
| RO03 | Effective input change, whitespace, Unicode and raw JSON order change alter fingerprint; framing collision controls differ; independent fixed expected vectors verify algorithm. |
| RO04 | Empty effective fields yield exact digest; unsupported type yields fixed unavailable status/no raw type or digest, preserving existing behavior. |
| RO05 | Rules edited in DB after admitted Job capture do not change receipt or prompt inputs actually supplied in that run. No reload query added. |
| RO06 | HTTP/MCP/direct/scheduler cannot acquire an invented APPROVED/ALLOWED actor or policy version; fixed unavailable statuses remain honest. |
| RO07 | Empty source, unchanged source, setup failure, provider error and panic retain honest rule observation; zero logical invocation stays zero where already guaranteed. |
| RO08 | All durable Summary paths carry additive receipt; failed writes preserve actual prior prefix, distinguish returned completion and fallback terminal status, remove callbacks safely. |
| RO09 | Preparation/execution values, every prior scalar assertion, provider calls/prompts/retries, selection/checkpoint/ownership/notifications unchanged. Exact existing-test amendment, if necessary, audited before seed. |
| RO10 | Malformed/foreign/mismatched IDs suppress bindings; fixed metadata statuses, no raw rules/prompt/errors/secrets; 2 KiB cap and frozen independent copies. |
| RO11 | Mutation substituting a DB-latest rule fingerprint or wrong effective field fails a named semantic assertion; source bytes restored and baseline rerun. Choose controls only after final WO scope. |
| RO12 | No WAIT_DATA run status, owner/deadline/retry, admission permission decision, semantic auto-skip or policy enforcement is introduced or claimed. |

Planned evidence is synthetic local application-observation proof only. It cannot establish CVF controlling AI behavior; any future such claim requires a separately authorized real provider request/response. No runtime executed by this design. Budgets must be newly bounded in a future work order; unused R059–R061 two Go are historical capacity, not new authorization.

## Deferred contracts and next governed move

Permission trace requires actual authenticated decision context from each HTTP/MCP entry path plus an explicit service principal contract for cron/after-sync/direct execution. Admission/ownership is insufficient. Human override needs actor/time/reason and verification separate from successful reservation. Do not pass untrusted request booleans as authority.

Immutable policy/rule releases require version ownership, update/revocation semantics and binding to the run's actual effective input; content fingerprint alone does not solve that. WAIT_DATA requires a genuine eligibility/execution/disposition contract, persisted owner/deadline/reason/retry, cancellation and idempotency, distinction from no source/unchanged/error, and review of all entry paths. These behavior/security/schema boundaries need separate specifications and independent R2 review before enforcement. PII gate, budget reservation, usage/pricing provenance and S2 exit sampling remain separate.

Next ORCHESTRATOR audits this candidate into a concrete bounded work-order draft: settle digest exposure, exact existing-test compatibility and provider-free evidence scope; assign independent implementation/reviewer and finite campaign/Go cost before immutable seed/activation. Existing owner medium-child/root-review route is available when dispatched; no child dispatched for this document. Root handles routine metadata directly. No push/merge/deployment or source authority from this planning artifact.

2026-10-08 consolidated pre-dispatch audit: [bounded work-order draft](R063_BOUNDED_WORK_ORDER_DRAFT_2026-10-08.md) resolves existing Summary exposure and exact two old-test compatibility blocks. This supersedes the earlier unresolved digest-exposure sentence; BUILD is still not dispatched. Metadata-invalid binding suppresses digest as well as identifiers; unsupported-type applies only to valid bindings. Observation deliberately does not change incremental unchanged-selection or cache semantics when rules change.
