# Git artifact hygiene — R129

Status: COMPLETE_PENDING_REVIEW
Risk: R2
Worker: Codex /root — repository artifact hygiene
Reviewer: Owner / independent review pending

## Objective

Keep the 72 MB R126 raw verification ZIP out of the current branch tree while preserving the local copy. Ignore that exact path for future runs, remove its broken current-tree references from the catalog and index, and clarify the R126 review record.

## Scope

- `.gitignore`
- `docs/reviews/probes/review_completion_20261018/local_verification_originals.zip`
- `docs/reviews/R126_FINAL_CURRENT_PACKAGE_REVIEW_2026-10-18.md`
- `docs/INDEX.md`
- `docs/catalog/ARTIFACT_REGISTRY.json`
- R129 work order, review, authority, tranche, and continuity records

No application source, provider call, main-branch update, merge, deployment, or history rewrite. Keep the local ZIP at `../.ccmai_local_evidence/review_completion_20261018/local_verification_originals.zip`.

## Acceptance

The exact archive path is ignored and absent from the committed branch tree; its local backup remains byte-identical; catalog checks pass; the owner-requested feature branch push succeeds and remote HEAD matches. Existing branch history still contains the already-published blob.

## Evidence and limits

The archive contains 70 raw verification logs/results and expands to 1,600,128,989 bytes. The prior R126 summary, manifests, cleanup receipt, and screenshot remain tracked. This cleanup does not rewrite Git history or claim removal of the object from older commits.
