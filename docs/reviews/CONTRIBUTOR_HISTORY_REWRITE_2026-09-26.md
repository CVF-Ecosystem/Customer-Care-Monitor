# CCMAI-CREDIT-001: history rewrite verification

Status: REVIEW OPEN
Risk: R2; independent human review remains required.

## Observed result

- Remote `archive/cqa-import-history-2026-09-26` points to old main
  `3a9f5e4bb2c6a7186f11ac734c8b82812e503276`, preserving the imported
  CQA history and its original authors.
- Rewritten remote `main` reached
  `4d95f9a587507448176775a4a7d718388874061d` via `--force-with-lease`.
- Five reconstructed main commits have Blackbird081 as primary author. The
  import commit credits Claude through a co-author trailer; subsequent product
  commits credit Codex through co-author trailers. The import message identifies
  the upstream CQA commit and explicitly disclaims original authorship.
- README names Blackbird081, Claude, and Codex as this product's development
  team. It links upstream CQA, the preserved archive, and LICENSE; LICENSE
  retains the SePay copyright notice.
- The candidate tree matched the inspected staged tree before the rewrite.
  `git diff --cached --check`, project doctor (25/25), and catalog check passed.
  No application source behavior was changed in this tranche.
- Immediately after the rewrite, GitHub's REST contributors endpoint returned
  only Blackbird081 as primary author. The web Contributors panel had not yet
  populated. GitHub's computed display must be checked again after refresh;
  this record does not claim that all three avatars already appear.

## Open review

An independent human reviewer should compare the archive and new main refs,
confirm source/license attribution, and check GitHub's refreshed Contributors
panel. Do not close this tranche or claim a settled three-person GitHub panel
until that review is recorded.

No CVF runtime-governance behavior was asserted or tested by this work.
