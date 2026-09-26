# CVF adoption for Customer Care Monitor AI

Status: ACCEPTED FOR PROJECT WORKFLOW
Date: 2026-09-26
Risk: R2 (governance configuration and future external AI use)

## INTAKE

The owner requested that future changes to this repository be controlled by the CVF mechanism used in the sibling workspace projects. The authorized scope is this repository's governance files and its local workspace binding. The upstream CQA repository and the CVF core are read-only references.

Acceptance boundary: a portable core pin, project policy, agent instructions, continuity state, governed catalog, and a passing project doctor. This adoption does not add runtime CVF approval, provider routing, DLP, or evidence verification to the application.

## DESIGN

Use the public CVF downstream bootstrap at core commit `19386f64e6bc36d1dcdbadca6ff97253feefb1bf`. Keep the core as a sibling and the local binding ignored by Git. Preserve the old CQA documentation homepage under `docs/legacy/CQA_INDEX.md`; let the catalog manager own `docs/INDEX.md`.

Keep the product's single-company boundary from `docs/PRODUCT_DIRECTION.md`. New AI governance controls require separate specifications, bounded work orders, and real provider evidence before any runtime claim.

## Decision and next move

The project enters DESIGN for its next application change. Before BUILD, define the intended control, evidence, risk, owner, acceptance criteria, and work order. The catalog starts with no source-verified runtime modules; its empty state must not be interpreted as missing application code.
