# INTAKE — CCMAI-UX-011 Results screen redesign

**Date:** 2026-09-28 · **Owner direction:** proceed with independent UI redesign first; Claude must leave every backend/runtime intersection untouched · **Phase:** INTAKE complete, DESIGN next · **Risk ceiling:** R2 for UI implementation · **Design owner:** Claude under the existing owner delegation · **Independent reviewer:** Codex.

## Objective and inherited truth

Redesign `/results` and its result-detail dialog for desktop/mobile × light/dark, following the [UI roadmap](../roadmaps/UI_UX_REDESIGN_ROADMAP_2026-09-27.md), [design direction](../decisions/UI_DESIGN_DIRECTION_2026-09-27.md), [glossary](../reference/UI_GLOSSARY.md), and reviewed UX-000/UX-002/UX-010 components. Address UX-07/08/09 presentation: show the local-only source caveat above populated lists, make serious source warnings legible and prominent, and keep QC issues separate from classification tags. Preserve the already reviewed UX-002 note and visible source column.

`Results.vue` gets facets and paginated rows from the existing server endpoints; filters, sorting, pagination and exports are server-scoped. Treat the API response, existing limits and source-status meanings as fixed inputs. A new layout must not imply that one loaded page represents all results, that a filter count comes from a different scope, or that an export is limited to the visible page unless the endpoint actually does so.

## Independent lane: allowed for Claude

1. DESIGN: inspect current `frontend/src/views/Results.vue`, Job Detail, UX-000 components, UX-002 SPEC/evidence and baseline screenshots. Record an approved canvas version for Results desktop/mobile, QC/classification, populated/empty/loading/error, changed/unavailable/legacy source, detail dialog and dark mode. The existing mobile card behavior is a useful baseline.
2. SPEC: state each visible count's data source/scope, interactions and accessibility; separate intended behavior from current implementation. Keep all four reviewed `source_integrity_status` meanings and the local-only caveat. Never present null confidence as a number or a source status as fully verified. Preserve real navigation, filters, export and detail behavior.
3. WORK_ORDER: after SPEC, authorize only bounded frontend presentation changes in `frontend/src/views/Results.vue`, additive `frontend/src/components/ui/**` props if needed, additive i18n keys, focused frontend tests and UI evidence/continuity files. Record precise paths and acceptance before BUILD; acknowledge each role/phase transition in the active handoff. Claude may BUILD under that work order and return one local `REVIEW_PENDING` commit for independent Codex review.
4. Evidence: focused semantic/interaction tests with UI-only mocks, `vue-tsc`, frontend build/tests, desktop/mobile light/dark captures (including affected dialog/list states), diff/catalog/doctor checks. Synthetic disposable UI data only. No claim about live CVF governance behavior.

## Held intersections: Claude must not implement in UX-011

| Intersection | Boundary and owner |
|---|---|
| Dashboard QC violation count (`UX-02`, `qc_violation_count`) | Separate R2 API tranche under Codex orchestration. Do not change `issues` semantics or dashboard counters. |
| Demo-channel scheduler (`UX-06`) and sync status | Separate R2 runtime tranche under Codex orchestration. Do not edit scheduler, channel records/markers, credential handling or sync endpoints. |
| S1/S2+ runtime source truth, result contract, provider admission and human disposition | Backend roadmap. Do not edit Go, DB, migrations, API contracts, stores or runtime rules; do not claim more complete source verification or calibrated confidence. |
| Results pagination/filter/export semantics | Existing server contract. A desired new filter, count, scope or export mode that the API cannot supply becomes `BLOCKED_API_CONTRACT` in the SPEC; leave its implementation untouched and continue the independent presentation work. |
| Other screens | No Dashboard, Channels, Messages, Jobs or Job Detail source edits in UX-011. Shared component changes require additive, backward-compatible props and regression checks. |

If a proposed design requires a held intersection, draw or document it separately as a future proposal, clearly mark it unavailable in this BUILD, and keep UX-011 within the current contract. A boundary change needs a separate Codex-routed tranche; it is not implied by the owner's design delegation.

## Handoff condition

Claude records the DESIGN/`SPEC_AUTHOR` acknowledgment in the active handoff before design work, freezes a canvas version in the SPEC, writes the bounded work order before BUILD, then uses the role route `SPEC_AUTHOR → WORK_ORDER_AUTHOR → IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD`. Return `REVIEW_PENDING`, not self-PASS or FREEZE. No push, deployment, provider call, real channel sync or persistent `ccma` change.
