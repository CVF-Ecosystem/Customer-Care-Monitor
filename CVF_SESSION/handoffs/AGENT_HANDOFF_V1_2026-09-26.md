# Agent Handoff V1

Status: ACTIVE

## Current State

- Project: Customer-Care-Monitor-AI
- Current mode: BUILD
- Active phase: BUILD
- Active role: IMPLEMENTATION_WORKER
- Next allowed move: Execute CCMAI-CREDIT-001 within README/provenance scope; verify archive and rewritten main before REVIEW.
- Parked operator checkpoint: none

## Seven-Step Control Chain

`INTAKE -> DESIGN -> SPEC -> WORK_ORDER -> BUILD -> REVIEW -> FREEZE`

This chain preserves seven governed decisions. A roadmap tranche may inherit
accepted evidence and enter at the earliest still-open stage; it does not need
to recreate all seven artifacts from zero. Record the inheritance and require
evidence for each applicable transition. A transition gate is not necessarily
a standalone independent-review artifact. `REVIEW` is the formal result
evaluation before `FREEZE`.

## Role Assignment

Roles are responsibilities, not provider names. One agent may hold several
roles only when each transition is recorded. Available roles are ORCHESTRATOR,
SPEC_AUTHOR, WORK_ORDER_AUTHOR, IMPLEMENTATION_WORKER, REVIEWER, REPAIR_WORKER,
CLOSER, COMMIT_STEWARD, and SESSION_SYNC_STEWARD. High-risk work requires an
independent reviewer.

## Completed

- INTAKE accepted the owner's request to apply CVF to this repository, bounded
  to project governance files and local workspace binding (R2).
- DESIGN chose the public downstream bootstrap and governed catalog; the prior
  CQA documentation homepage was preserved in `docs/legacy/CQA_INDEX.md`.
- Project doctor passed 25/25 after the catalog migration. The adoption record
  and static check are under `docs/decisions/` and `docs/reviews/`.

## Open Work

- Specify a bounded runtime control before changing application behavior.
- Workspace-wide gate failures in unrelated sibling repositories remain open;
  see `docs/reviews/CVF_ONBOARDING_CHECK_2026-09-26.md`.

## Active Tranche: CCMAI-CREDIT-001

- INTAKE: owner requested README credits for Blackbird081, Claude, and Codex;
  SePay remains source attribution. The owner selected the Git history rewrite
  needed to change GitHub's Contributors panel.
- DESIGN: preserve old history in an archive branch, reconstruct only the new
  product's milestones on `main`, and retain LICENSE/source links.
- SPEC: `docs/specs/CONTRIBUTOR_CREDIT_2026-09-26.md`.
- WORK_ORDER: `docs/work_orders/CCMAI_CREDIT_001.md` (R2, authorized in session).
- Role route recorded before BUILD: ORCHESTRATOR -> SPEC_AUTHOR ->
  WORK_ORDER_AUTHOR -> IMPLEMENTATION_WORKER. The current role is
  IMPLEMENTATION_WORKER; next transition is COMMIT_STEWARD after checks.
- Claim boundary: README and Git provenance only; no CVF runtime claim.

## Claim Boundary

This handoff records repository governance onboarding. It does not claim that
CVF controls the application runtime or that a provider-backed test passed.
