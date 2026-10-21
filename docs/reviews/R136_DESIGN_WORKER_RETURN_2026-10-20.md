# R136 execution UI design worker return

Status: COMPLETE_PENDING_REVIEW

Risk: R1

Worker: Sol Medium delegated source/design IMPLEMENTATION_WORKER

Reviewer: Root independent source/design REVIEWER

Self-declared worker-return artifact: yes

Deliverables are `docs/specs/M3_EXECUTION_UI_R136_SPEC_2026-10-20.md`, the compact `docs/reviews/probes/r136_execution_ui_design_20261020/source_identity.json`, this return, and the pre-document role acknowledgment appended to the active R136 handoff. No production source or shared continuity/seed was edited by this worker. Source identity pins repository HEAD `b09b8033627b129c9d37fc3c28fea04f4c6294fa` and SHA-256 values for 13 read-only backend/frontend files cited in the design, including config defaults. Root must independently review current line claims before accepting the contract.

The intended contract separates the existing saved request from any later execution. Current source has a strict two-field execute POST and a server-owned synthetic startup binding when enabled; default config is disabled. Grant issuance remains separately server-side/operator-owned. The proposed writer-only read discovers an actor-bound grant handle only when current prerequisites appear satisfied; a second read-only status reports durable execution independently of process transport, including after publication moves the same case to OPEN. Neither GET would claim, issue, reserve, consume or call. The locked POST remains authoritative, and a readiness response never guarantees admission. No raw provider/config/credential/grant entry appears in user-facing flow.

The spec defines a deliberate one-click action with no automatic POST retry. A scoped `IN_FLIGHT` hold must persist **before** sending; persistence failure sends nothing, and a crash before response cannot silently re-enable execution. A temporary no-row status read cannot clear the hold. It scopes status polling and cancellation to tenant/case/request/actor/session/permission lifetime, keeps reader history available without controls, and maps errors to understandable VN/EN messages. Positive and nonready GET DTOs and read-failure 503 responses are explicit; a published case version derives from linked proposal/event lineage, never later mutable case state. The current asynchronous pre-call budget check means a 202 claim may still terminalize `DENIED` with zero debit/call. The finite next R2 candidate set now names each new/edit file and focused tests. A positive AI execution claim would require fresh owner authorization and an actual provider request/response; R117's physical 1/1 call is consumed and cannot be reused. Operator setup of synthetic mode, budget, key and grant remains a prerequisite under separate authority. Full M3, installed activation, business/pilot/release and governance claims stay open.

Root's consolidated round-one documentary review identified four same-scope gaps: pre-send persistence, exact GET shapes/read failures and published lineage, asynchronous budget timing, and explicit candidate filenames. These are corrected in the spec and return. No source/runtime check was invoked for this repair; Root independently verifies the final design.

Unresolved owner/operational decisions for any R2 dispatch are the fresh provider physical-call cap, exact synthetic tenant/job/actor/request and independently issued grant, secret/budget custodian, and disposable runtime/cleanup owner. The documentation can be reviewed without those effects. Worker counters: Go 0, Node 0, database 0, Docker 0, browser 0, provider 0, credential/network/download 0, grant/execute 0, commit/publication 0. This is static source analysis only; no runtime or AI governance test was run.

## Return-Time Closeability Recheck

closeabilityDisposition: CLOSEABLE

outsideAuthorityBlockers: NONE

nextRepairRoute: REVIEWER_LOCAL_REPAIR

workerRedispatchAllowed: YES

The documentary deliverables fit the R136 authority. Root's independent source/design review and any subsequent R2 authority remain pending; there is no self-approval or FREEZE claim.
