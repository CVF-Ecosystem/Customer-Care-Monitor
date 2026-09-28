# CCMAI-CREDIT-003 — public main message cleanup BUILD evidence

**Date:** 2026-09-29 · **Operator:** Codex · **Status:** `REVIEW_PENDING` after BUILD, independent R2 REVIEW and FREEZE open · **Authority:** [work order](../work_orders/CCMAI_CREDIT_003.md) and the owner's explicit request to remove the root import/source/archive lines.

## Pre-rewrite authority and scope

- The new `origin` was `https://github.com/CVF-Ecosystem/Customer-Care-Monitor.git`, with exactly one remote ref, `main` at `bb1f24cc5ef69451f25af0042ca251bceb810cfb`. The local planning tip before the candidate rewrite was `b336a68c5c5c3a49a27659c6adb0f75fcf467b80`, 119 linear commits, no merge commits or tags.
- The original root was `3036d5e53948e500469d26b0cc3421cd6f7569c5`; its body included the upstream URL, upstream SHA and obsolete archive-history claim. `LICENSE` contains the SePay and CVF-Ecosystem MIT notices and was outside scope.

## Local candidate checks

- Rewrote only four selected messages on `rewrite_candidate`: the root is now `Establish Customer Care Monitor AI source baseline`, retaining its Claude co-author trailer; three later archive/inherited-guide subjects are neutral. New root: `4a92494b2609ff28e465f050a64fc211dca1ae76`.
- The candidate before evidence synchronization was `4e0099de827c51b669235f1e958be8f409c8464c`. Old and candidate commit counts were both 119. Every paired commit had identical tree hash, author name, author email and author timestamp. The final tree hash was identical at `9927b4c3ef8f4d1fbe35b4b663ec79469aa33d84`; `git diff main rewrite_candidate` was empty. The distinct co-author trailer set was unchanged.
- A case-insensitive scan of candidate commit messages found no root import title, upstream URL, archive branch or archive tag claim. The historical review documents still contain old SHA references; the message rewrite does not relabel them as new SHA records.

## Remote completion and limits

- The inspected remote `main` was still `bb1f24cc5ef69451f25af0042ca251bceb810cfb`. An exact-lease push (`--force-with-lease=refs/heads/main:bb1f24c...`) updated only `main` to the evidence-bearing candidate `73ee01fbf01af149e337a3f011054bb1b3a7218a`. Remote `ls-remote` matched and showed no tags.
- Local `main` now tracks that remote SHA. The temporary `rewrite_candidate` branch and `refs/original/refs/heads/rewrite_candidate` backup ref were removed; local branch list contains only `main`, and there are no local tags. The worktree is clean.
- The final candidate adds this evidence and continuity to the tree-identical message rewrite. Historical review documents still contain old SHA references; those are old observations rather than resolvable commit identifiers on the new `main`. GitHub may retain unreachable old commit objects/cache temporarily after the force update. The old repository remains the owner's separate deletion task.
- Follow-up commit `d9152b3a997307738368d97a180d79f897529e6a` synchronized evidence/state and was pushed normally. [Docs run 36462883704](https://github.com/CVF-Ecosystem/Customer-Care-Monitor/actions/runs/36462883704) completed successfully; the public Pages home returned HTTP 200. Local docs build, catalog check and workspace doctor 25/25 passed. The GitHub contributor-statistics endpoint returned HTTP 202, still recalculating after the rewrite.
- GitHub's commit API returned HTTP 200 for both the new root `4a92494` (neutral title) **and the old unreachable root `3036d5e`** (old import title). This proves the old commit remains accessible by its exact SHA at the time of this check, although it is no longer on `main`. [GitHub's history-cleanup documentation](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository) explains that cached views and unreachable objects can persist after a force push, and that Support's purge assistance is limited to qualifying sensitive data. No claim of server-side erasure is made here.
- This is a message rewrite, not a source change or CVF governance proof.
