# CCMAI-CREDIT-003 — remove obsolete source-import messages from main history

**Date:** 2026-09-29 · **Authority:** owner explicitly requested removal of the root commit's CQA import title, source URL, upstream SHA and archive-history claim, and similar commit messages · **Risk ceiling:** R2 under the accepted contributor-history rewrite precedent · **Status:** BUILD complete / REVIEW_PENDING.

## INTAKE, DESIGN and SPEC

`main` is a clean 118-commit linear history at `bb1f24cc5ef69451f25af0042ca251bceb810cfb`; `origin/main` matches it. The new repository has only `main` and no archive tag. The old import/archive prose remains in the root commit and three later commit subjects. Remove that GitHub-visible prose without claiming original authorship of imported code. `LICENSE` must retain SePay's copyright and MIT permission notice.

Reword only these four commit messages: root `3036d5e`, archive-decision `71ecbff`, inherited-guide audit `306ea3f`, and archive-cleanup `bb1f24c`. The replacement root subject is `Establish Customer Care Monitor AI source baseline`; retain its Claude co-author trailer. Later subjects should describe the product work without the obsolete archive claim. Leave unrelated commit messages, authors, author dates, co-author trailers, every commit tree, current source and governance evidence contents unchanged.

## Bounded BUILD and checks

1. Record the current local/remote tip, root and final tree hashes, commit count, author/co-author set, and ref set. Create a temporary **local** candidate branch; rewrite messages there without touching `main` or `origin` while validating.
2. Check candidate commit count, final tree hash, selected message changes, author/co-author set and no remaining import URL/archive claim in commit messages. Validate catalog, doctor and build gates; a tree-identical rewrite may inherit source-test evidence, but note the exact limitation.
3. Move the local `main` to the validated candidate and push only `main` using `--force-with-lease` against the exact inspected remote SHA. Do not push backup refs or tags. Verify remote SHA, CI and Pages; remove local temporary refs after successful verification.
4. Record old/new root and tip SHA, tree identity, user-visible results, and any historical SHA-link limitation in evidence and continuity. Return `REVIEW_PENDING` for independent R2 review; no self-approval or FREEZE.

The root import text is removed from commit messages. Historical governance documents and inherited user guides are separate records; this order does not erase them or change application behavior. The old repository is left for the owner to delete. No provider, credential, database, deployment code or CVF core work is authorized.

Role route: `ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR → IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD` (Codex). The owner explicitly authorized removal of the public historical prose; the only public history mutation allowed is the exact-lease `main` update after candidate validation.
