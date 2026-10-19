# R130 feature branch history rewrite — owner accepted and published

Status: REVIEW_PASS
Risk: R2
Worker: Codex /root Git history maintenance
Reviewer: Project owner (independent)

## Result

The owner reviewed and approved the rewritten graph at `74bbd4ce1e4b58607d6aa071b039a73856b4deef` before publication. The ZIP was introduced by `90bc02ed14c5e3997ec97fe400a64cf06cae6f70`, whose parent `9ab54062976de181ffbb910be44393021d1bc3db` and all earlier history were retained. The rewritten tip tree matches the pre-rewrite local safety tip `727f65825af72c017497412769cff15b37ed7308`; no commit reachable from the replacement branch contains `docs/reviews/probes/review_completion_20261018/local_verification_originals.zip`.

The original 75,961,107-byte archive remains outside the repository at `../.ccmai_local_evidence/review_completion_20261018/local_verification_originals.zip`. Its SHA-256 is `11173e8cb3c2927e424d3dbe469ef53006ddb5739693208b14142fa5384c39fe`, matching the preserved original blob.

## Publication receipt

`--force-with-lease` replaced `codex/ccma-m2-intervention` from `eb94f622c31fa8717677ba6090074c30a7381ef4` with `74bbd4ce1e4b58607d6aa071b039a73856b4deef` after the owner approval. GitHub ref readback confirmed the feature branch at `74bbd4ce1e4b58607d6aa071b039a73856b4deef` and `main` unchanged at `d7a6f7635608d1ad44169372a6b5e0fbcbcd09c4`. No merge was performed. The local safety ref and archive backup remain available.

## Verification

- Feature-branch reachable-history scan: ZIP path absent.
- Pre-rewrite and rewritten tip tree comparison: identical (exit 0).
- Local archive SHA-256 and size: match the retained original.
- `scripts/cvf_downstream_gate.py preflight`: PASS (7/7).
- CVF `pre-commit` gate before publication: PASS (6 mandatory controls).
- Remote feature and main refs: read back at the values above.

R130 covers Git history hygiene only; it makes no product behavior or runtime governance claim.
