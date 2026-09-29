# CCMAI-CREDIT-003 — independent R2 review of the public main message cleanup

**Date:** 2026-09-29 · **Reviewer:** Claude (`REVIEWER`); the implementer was Codex · **Authority:** [work order](../work_orders/CCMAI_CREDIT_003.md), [BUILD evidence](CCMAI_CREDIT_003_MESSAGE_REWRITE_2026-09-29.md) · **Disposition:** `REVIEW_PASS` with two non-blocking observations; FREEZE is a CLOSER decision and remains open.

This is a separate record from `CCMAI-RUNTIME-010`. It does not accept or review R010, and R010 acceptance does not depend on it. Role transition: `COMMIT_STEWARD (Claude, R010 local commit a9dcbed) → REVIEWER (Claude, CREDIT-003)`. Claude did not implement, push or commit any part of CREDIT-003.

## What was checked independently

All checks were read-only, local git plus public unauthenticated GitHub reads. No history, ref, remote or file content was changed.

1. **Paired history.** The old pre-rewrite planning tip `b336a68` and the new pre-evidence candidate `4e0099d` still exist locally. The old tip is not on `main`; `4e0099d` is an ancestor of `main`. A script compared both chains commit by commit in topological order:
   - 119 = 119 commits, with no merges;
   - every pair has an identical tree hash, author name, author email and raw author date;
   - every pair has an identical sorted co-author trailer list;
   - the final tree is `9927b4c3ef8f4d1fbe35b4b663ec79469aa33d84` on both.
2. **Exactly four messages changed**, matching the work order's list:
   - root `3036d5e → 4a92494`: `Import CQA source snapshot …` with upstream URL/SHA/archive prose becomes `Establish Customer Care Monitor AI source baseline`, and its Claude co-author trailer is kept;
   - `71ecbff → 93d90e4`: `preserve CQA history as tag` becomes `record contributor provenance decision (#2)`;
   - `306ea3f → 413cfaa`: `clarify CQA fork and audit user guides` becomes `audit inherited user guides (#3)`;
   - `bb1f24c → 7b41a13`: `drop separate CQA archive requirement` becomes `close repository archive cleanup`.
3. **No residual import prose on current main.** A case-insensitive scan of every commit message on `HEAD` found no upstream repository name/owner, upstream SHA, archive branch/tag name, `import CQA`, or original-history claim, and no subject containing `CQA`. The only pattern hit was a false positive: `source snapshots` in an unrelated analysis feature subject.
4. **License preserved.** `LICENSE` keeps `MIT License`, `Copyright (c) 2025 SePay` and `Copyright (c) 2026 CVF-Ecosystem`. Its only history on main is the root and `3baadb7 Preserve SePay and CVF-Ecosystem MIT attributions`, both unchanged in tree by the rewrite.
5. **Remote state.** `git ls-remote origin` shows only `HEAD` and `refs/heads/main`, at `75ba054` (the tip before R010), and no tags. `73ee01f`, the exact-lease target named in the evidence, is an ancestor of `origin/main`. Local branches are only `main`, with no tags.
6. **Public observations match the evidence.**
   - The GitHub Actions API reports Docs run `36462883704` as `completed / success` on `d9152b3`.
   - The Pages home returned HTTP 200.
   - The commit API returns HTTP 200 for new root `4a92494` (neutral title) and still returns HTTP 200 for old root `3036d5e`, with the old import title, by exact SHA.

   The BUILD evidence already discloses the last point and claims no server-side erasure.

## Findings

No blocking finding. The rewrite meets every work-order acceptance point: selected messages only, identical trees and authorship, preserved license and co-authors, `main`-only exact-lease update, and honest disclosure of limits.

- **O1 (non-blocking, disclosed):** GitHub still serves the old root `3036d5e` by exact SHA. Removal from the public cache is outside what a force push can do. If the owner needs it gone, that is a separate owner decision (for example, contacting GitHub Support), and it may not qualify for a purge.
- **O2 (non-blocking, local only, undocumented):** two local refs, `refs/rewrites/customer-care-pre-credit` (`049e44c`) and `refs/rewrites/customer-care-candidate-final` (`4d95f9a`), dated 2026-09-26 (CREDIT-001 era), still keep the old root `3036d5e` and its import message reachable in this clone. They are not on the remote and predate CREDIT-003, so the evidence's "only `main`, no tags" statement is accurate for branches and tags. They are not mentioned in any record, however. Deleting them is a destructive local ref action, so the reviewer did not do it; it is left to the owner or a CLOSER decision. Local `refs/codex/turn-diffs/*` checkpoints were also observed; they are tool-local and not reviewed here.

## Limits

The exact-lease push itself was not re-observed; only its outcome was verified (remote ref set and ancestry). The GitHub contributor sidebar was not re-inspected. This is a history/provenance review, not a source change or CVF governance proof, and no provider or credential was used.

## CLOSER follow-up: local rewrite refs

After this independent review, Codex took the separate CLOSER responsibility for observation O2 under the owner's earlier instruction to remove unneeded CQA history. Both exact local targets were verified, each contained old root `3036d5e`, `main` excluded that root, and `git ls-remote --heads --tags origin` showed only `main`. Codex deleted only `refs/rewrites/customer-care-pre-credit` (`049e44c`) and `refs/rewrites/customer-care-candidate-final` (`4d95f9a`) with exact-old-value `git update-ref -d`; `git for-each-ref refs/rewrites` is now empty. No branch, tag, remote ref, commit object, license or worktree content was changed by this action. O2 is resolved for these two refs. O1 remains: GitHub may still serve the old SHA directly. This follow-up does not change Claude's original REVIEW_PASS or close FREEZE.
