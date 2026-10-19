# Git artifact hygiene

Status: SUPERSEDED

## Current State

- Current mode: REVIEW
- Active phase: REVIEW
- Active role: Root ORCHESTRATOR / COMMIT_STEWARD; owner independent review pending
- Active tranche: CCMAI-RUNTIME-129
- Next allowed move: CCMAI-RUNTIME-129 REVIEW_PENDING: exact R126 raw ZIP is ignored and absent from the current feature branch tree; intact local backup retained at ../.ccmai_local_evidence/review_completion_20261018/local_verification_originals.zip. Owner independent review pending. No merge/main/history rewrite/deploy; prior Git history still contains the blob.
- Parked operator checkpoint: OWNER_DEFERRED_FACEBOOK_ZALO_OA_ACCOUNTS

Owner requested that the oversized local verification archive be ignored and excluded from the pushed feature branch. The unchanged 75,961,107-byte file is preserved outside the repository at `../.ccmai_local_evidence/review_completion_20261018/local_verification_originals.zip`. The branch has not been merged to main. Existing commits still contain the archive blob; no history rewrite was authorized.

Work order: `docs/work_orders/CCMAI_RUNTIME_129_GIT_ARTIFACT_HYGIENE.md`. Owner independent review is pending.


R130 acknowledgment: owner requested a feature-branch history rewrite to remove the ZIP from reachable commits; main remains excluded. R129 authority remains immutable and is not widened. R130 authority seed and bounded work order are recorded before BUILD.
