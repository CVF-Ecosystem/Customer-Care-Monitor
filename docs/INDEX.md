# Project Documentation Index

Machine-readable source: `docs/catalog/ARTIFACT_REGISTRY.json`

## Start Here

- Active session/phase/role state.: `CVF_SESSION/ACTIVE_SESSION_STATE.json`
- Historical pre-message F02 evidence/FREEZE assessment; next-step recommendation superseded by the post-R032 assessment.: `docs/reviews/F02_REMAINING_EVIDENCE_AND_FREEZE_ASSESSMENT_2026-10-02.md`
- Historical post-R032 assessment/proposal; owner delegates local closure decisions, now executed under R033; live remains OPEN.: `docs/reviews/F02_POST_R032_EVIDENCE_AND_LOCAL_FREEZE_PROPOSAL_2026-10-03.md`
- Current active handoff: R033 bounded local message closure and remaining live boundaries.: `CVF_SESSION/handoffs/AGENT_HANDOFF_LOCAL_MESSAGE_FREEZE_2026-10-03.md`
- Downstream gate learning intake for CVF parent assessment and transfer; source evidence and deferred disposition.: `docs/reviews/learnings/CCMAI_TO_CVF_DOWNSTREAM_GATE_LEARNING_INTAKE_2026-10-01.md`
- Owner-agreed shared finding/learning folder convention: immediate project learning and CVF parent intake.: `docs/reviews/learnings/README.md`
- Shared learning: repair acknowledgment, continuity synchronization, mutation validity and evidence honesty; read before BUILD/REPAIR.: `docs/reviews/learnings/feedback_cvf_repair_workflow.md`
- Shared learning: shell cleanup and MSYS paths; read before disposable-resource cleanup.: `docs/reviews/learnings/feedback_shell_cleanup_and_paths.md`
- Shared sync coverage learning: conversation, message, storage, live-channel and governance evidence boundaries.: `docs/reviews/learnings/feedback_sync_coverage_evidence_layers.md`
- Prepared Pancake live-proof packet; required external inputs and network/credential authority outstanding; no dispatch.: `docs/reviews/F02_PANCAKE_LIVE_PROOF_PACKET_2026-10-03.md`
- Historical handoff through F07 review and F02-D intake acknowledgment; targeted lookup only.: `CVF_SESSION/handoffs/AGENT_HANDOFF_V1_2026-09-26.md`
- Historical F02-D BUILD/review and F02-E planning intake; targeted lookup only.: `CVF_SESSION/handoffs/AGENT_HANDOFF_F02D_2026-10-02.md`
- Historical F02-E review, owner pause/resume and F02-F intake acknowledgment; targeted lookup only.: `CVF_SESSION/handoffs/AGENT_HANDOFF_F02E_2026-10-02.md`
- Project continuity front door.: `CVF_SESSION_MEMORY.md`
- Reviewed F02-D Pancake local message contract; source and evidence boundaries.: `docs/specs/RUNTIME_PANCAKE_MESSAGE_COVERAGE_F02D_2026-10-02.md`
- Local message order FROZEN under separate R033 closure authority; original independent review and live/global F02 limits retained.: `docs/work_orders/CCMAI_RUNTIME_030.md`
- Reviewed F02-E Facebook message local safety contract and acceptance matrix.: `docs/specs/RUNTIME_FACEBOOK_MESSAGE_COVERAGE_F02E_2026-10-02.md`
- Local message order FROZEN under separate R033 closure authority; original independent review and live/global F02 limits retained.: `docs/work_orders/CCMAI_RUNTIME_031.md`
- Independent R032 exact-BUILD review: CHANGES_REQUIRED for finite cycle detector/evidence and current implementation prose.: `docs/reviews/CCMAI_RUNTIME_032_F02F_INDEPENDENT_REVIEW_2026-10-03.md`
- Active F02-F Zalo local full-history message contract and acceptance matrix.: `docs/specs/RUNTIME_ZALO_MESSAGE_COVERAGE_F02F_2026-10-03.md`
- Local message order FROZEN under separate R033 closure authority; original independent review and live/global F02 limits retained.: `docs/work_orders/CCMAI_RUNTIME_032.md`
- Independent R032 exact-R1 re-review: REVIEW_PASS for local contract; finite semantic M13 detector and current prose settled; FREEZE_OPEN.: `docs/reviews/CCMAI_RUNTIME_032_R1_INDEPENDENT_REREVIEW_2026-10-03.md`
- Historical R032 review/assessment and owner local-closure delegation acknowledgment.: `CVF_SESSION/handoffs/AGENT_HANDOFF_F02F_2026-10-03.md`
- CVF enforcement manifest.: `.cvf/manifest.json`
- CVF governance policy.: `.cvf/policy.json`
- R033 closure review and local FREEZE decision; source identity and inherited evidence limits.: `docs/reviews/CCMAI_RUNTIME_033_LOCAL_MESSAGE_CLOSURE_2026-10-03.md`
- R033 separate local message closure authority; inherits R030-R032 independent product review.: `docs/work_orders/CCMAI_RUNTIME_033.md`
- Closed schema reference for the Artifact Registry.: `docs/catalog/schemas/ARTIFACT_REGISTRY.schema.json`
- Closed schema reference for the Module Registry.: `docs/catalog/schemas/MODULE_REGISTRY.schema.json`
- Standard-library catalog validation and rendering functions.: `scripts/lib/downstream_catalog/CvfDownstreamCatalogLib.ps1`
- Executable catalog manager (--check / --write).: `scripts/manage_cvf_downstream_catalog.ps1`
- Portable downstream machine gates: provenance, continuity, tranche/role contract, claim boundary, secret hygiene, workflow coverage and catalog (CCMAI-GOV-001).: `scripts/cvf_downstream_gate.py`
- Positive and negative fixtures for the downstream machine gates.: `scripts/tests/test_cvf_downstream_gate.py`
- Machine implementation-truth surface.: `IMPLEMENTATION_STATUS.json`
- Generated documentation index.: `docs/INDEX.md`
- Generated human module catalog.: `docs/catalog/MODULE_CATALOG.md`

## Governed Artifact Families

- Decisions.: `docs/decisions/`
- Reviews and evidence.: `docs/reviews/`
- Roadmaps.: `docs/roadmaps/`
- Specifications.: `docs/specs/`
- Dispatcher-owned tranche authority seeds; GOV-001 is an explicit reviewer-seeded bootstrap exception.: `CVF_SESSION/authority/`
- Structured per-tranche contract records (phase, roles, allowed paths, review disposition) read by the downstream gate.: `CVF_SESSION/tranches/`
- Work orders.: `docs/work_orders/`

Plans describe intended work. `IMPLEMENTATION_STATUS.json`, source, tests, and
review evidence determine what is actually implemented.
