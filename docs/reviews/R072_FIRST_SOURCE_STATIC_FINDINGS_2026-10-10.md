# R072 first-source independent static review

Disposition: CHANGES_REQUIRED (NEW test only; no runtime has run).

Source 5285d9ff995e52aa227b5870616ce60e4c024033, worker fresh Luna xhigh; root independent REVIEWER, product/test edits0. Exactly two authorized source files; one parser line adds only the opaque optional key. Product semantics conform to UC01..04 by source inspection; no product repair requested. Baseline confirms every protected file except authorized parser unchanged; first source and immutable seed preserved.

Consolidated findings before first Vitest/typecheck:
- T1 P1: expectedLegacyProjection.execution.values includes ACCEPTED, REJECTED and NOT_SEPARATELY_OBSERVABLE. Existing execution() returns totals + usage_writes + fixed flags; it never spreads parsing counts. Thus the positive exact-output oracle contradicts actual accepted legacy behavior. Repair the NEW expected fixture to the existing return contract; do not widen product output to satisfy it. This is static evidence, not an executed failing test or compiler failure.
- T2 P2: the legacy-rejection group tests missing/version/unsafe/contradictory/unrelated keys without the optional extension, so it cannot demonstrate UC03 when that field is present. Exercise those rejection cases with the extension present as well as absent, while retaining independent execution availability where appropriate. This is coverage/sensitivity debt, no runtime mutation result.

Same-scope repair round1 authorized under existing seed: NEW test file only; preserve parser byte-for-byte, existing tests/backend/seed/first-source commit. Worker rehydrates and records REPAIR_WORKER / BUILD acknowledgment before edits, commits repaired source and returns for static approval. No runtime yet; original budget unchanged and unused, no new owner checkpoint. Root supplies findings, never edits tests. Approval of a test plan waits for corrected exact oracle. UC05..07 continue; no third-round authorization.

Root overhead separate: guessed read-only paths corrected with rg; stale R071 index description repaired. Duplicate gate-unit suite (root46/46 14.606s, worker46/46 31.875s) arose because inheritance clarification arrived after launch; preserve both, classify coordination overhead rather than product defect/model speed. Core doctor25/1 notes remain, BOOTSTRAP_MIGRATION_PENDING. No governance/live/provider/network/DB/GitHub/push/Go activity.
