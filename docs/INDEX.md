# Project Documentation Index

Machine-readable source: `docs/catalog/ARTIFACT_REGISTRY.json`

## Start Here

- Active session/phase/role state.: `CVF_SESSION/ACTIVE_SESSION_STATE.json`
- F02 remaining evidence and scoped FREEZE assessment; next bounded planning move.: `docs/reviews/F02_REMAINING_EVIDENCE_AND_FREEZE_ASSESSMENT_2026-10-02.md`
- Initial agent handoff.: `CVF_SESSION/handoffs/AGENT_HANDOFF_V1_2026-09-26.md`
- Downstream gate learning intake for CVF parent assessment and transfer; source evidence and deferred disposition.: `docs/reviews/learnings/CCMAI_TO_CVF_DOWNSTREAM_GATE_LEARNING_INTAKE_2026-10-01.md`
- Owner-agreed shared finding/learning folder convention: immediate project learning and CVF parent intake.: `docs/reviews/learnings/README.md`
- Shared learning: repair acknowledgment, continuity synchronization, mutation validity and evidence honesty; read before BUILD/REPAIR.: `docs/reviews/learnings/feedback_cvf_repair_workflow.md`
- Shared learning: shell cleanup and MSYS paths; read before disposable-resource cleanup.: `docs/reviews/learnings/feedback_shell_cleanup_and_paths.md`
- Shared sync coverage learning: conversation, message, storage, live-channel and governance evidence boundaries.: `docs/reviews/learnings/feedback_sync_coverage_evidence_layers.md`
- Project continuity front door.: `CVF_SESSION_MEMORY.md`
- CVF enforcement manifest.: `.cvf/manifest.json`
- CVF governance policy.: `.cvf/policy.json`
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
