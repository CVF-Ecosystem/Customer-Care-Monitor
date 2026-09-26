# CCMAI-CREDIT-001: README credit and repository history

Status: AUTHORIZED
Risk ceiling: R2
Owner authorization: active request and explicit choice to rewrite `main` history

## Allowed changes

- README contributor/source wording, this tranche's decision/spec/work order/review, and CVF continuity pointers.
- Remote ref `archive/cqa-import-history-2026-09-26` as a preserved copy of the old `main`.
- Remote `main` via a verified `--force-with-lease` after the archive ref is confirmed.

## Required sequence

1. Edit and review README, keeping upstream CQA/SePay attribution.
2. Run project doctor and catalog check; inspect the exact staged tree.
3. Create the archive ref from the inspected old `main` and verify it remotely.
4. Construct the new history locally from existing source-tree milestones, without deleting the archive or editing application source.
5. Verify its tree and author/co-author set; force-push with lease only if remote `main` still has the inspected tip.
6. Record observed results and open limits in the review/handoff.

Stop if source attribution is lost, the archive ref is missing, the remote tip changed unexpectedly, or the CVF checks fail.

## Owner-directed follow-up, 2026-09-26

The owner asked to remove GitHub's Compare & pull request prompt for the
recently pushed archive branch. Preserve its exact old-main commit in an
annotated tag, verify the tag target remotely, update current README and
continuity references, then delete the archive branch. Keep `main` free of
the old CQA commit ancestry. This changes the archive ref type only; the
original source attribution requirement remains in force.
