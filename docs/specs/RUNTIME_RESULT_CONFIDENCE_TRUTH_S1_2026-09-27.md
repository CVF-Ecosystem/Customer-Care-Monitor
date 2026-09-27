# S1 result confidence truth contract

**Tranche:** `CCMAI-RUNTIME-005` · **Phase:** SPEC · **Risk:** R2 · **Entry:** S1 remains IN_PROGRESS after R001–R004 independent REVIEW PASS; each FREEZE remains open.

## Problem and decision

`backend/engine/analyzer.go` persists `Confidence: 1.0` for QC evaluations, QC violations, and classification evaluations, although no calibrated confidence was measured. Classification tags contain a model-supplied number; `backend/notifications/dispatcher.go` presents it as a percentage. `JobResult` exposes the numeric field to job-specific APIs, and the frontend store types it as a required number. This can imply certainty or calibrated probability that the evidence does not support.

Keep confidence separate from verdict, severity, score, source integrity and evidence status. A numeric classification value may be retained only as **model-reported, uncalibrated** information. QC and conversation evaluations have **no numeric confidence**. No result in this tranche is labeled calibrated.

## Contract

1. Add nullable persistence for `confidence` and an explicit nullable `confidence_basis` discriminator. New QC evaluations/violations and classification conversation evaluations store `confidence = NULL`, `confidence_basis = unavailable`. New classification tags may store their validated model-reported number in `[0,1]` with `confidence_basis = model_reported_uncalibrated`. No `calibrated` value exists in this tranche. Preserve verdict, score, evidence refs, snapshot binding and transactional write behavior.
2. Job-specific result JSON uses `confidence: number | null` and `confidence_basis: unavailable | model_reported_uncalibrated`. The numeric value is exposed only when its persisted basis is `model_reported_uncalibrated` and the result is a classification tag; otherwise JSON returns `null` with `unavailable`. Existing rows have a NULL basis after migration and therefore expose no numeric confidence, including old QC placeholders of `1.0` and old classification tags whose provenance cannot be established. Do not rewrite their stored values. A model-reported number must never be labeled calibrated or shown as an objective probability.
3. Notification text must not show a bare percentage. If it includes the model-reported tag value, label its origin and lack of calibration in Vietnamese; a missing/invalid/legacy-unknown value yields no percentage. Frontend types and any visible job-specific result presentation must accept missing confidence and avoid defaulting it to 0 or 100%.
4. Schema transition on an existing `job_results` table must preserve rows and work on a fresh database. Any nullable-column/metadata change must be idempotent and documented; do not silently rewrite historical values into a different meaning. The local persistent Compose database is not a validation target for this tranche.
5. This is result semantics, not provider admission, confidence calibration, human approval, or a claim that CVF governs runtime AI. No provider call or customer data is needed for acceptance.

## Acceptance

- Focused disposable-MySQL tests prove new QC and evaluation rows have no fabricated confidence; classification tag values retain only the explicit uncalibrated basis; legacy QC and classification rows containing old numeric values serialize with `confidence: null`, `confidence_basis: unavailable`. Test both actual job-result API paths and notification formatter, not only a helper.
- Fresh and pre-existing-table migration paths are exercised twice without losing rows; rollback/compatibility behavior is documented. A test proves invalid model tag confidence is rejected by existing validation.
- Existing snapshot/evidence/result tests remain passing. Backend tests/build/vet, relevant frontend type check/build, catalog check, workspace doctor and diff check pass. Evidence records exact commands and the no-provider boundary.

## Deferred

Measured calibration, human disposition, provider routing/admission, automatic action, and probability claims remain later work. R001–R004 FREEZE decisions and S1 closure remain separate.
