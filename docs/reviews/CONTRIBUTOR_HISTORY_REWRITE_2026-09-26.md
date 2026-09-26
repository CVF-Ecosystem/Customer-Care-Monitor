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

## Follow-up: GitHub sidebar cache

The owner observed a five-avatar Contributors sidebar after the rewrite: the
two inherited CQA accounts remained visible alongside Blackbird081, Claude,
and Codex. On recheck, the REST contributors endpoint returned Blackbird081
with six contributions; the repository statistics endpoint returned HTTP 202
while GitHub calculated the data. The six commits currently on `main` have
Blackbird081 as primary author, and the old history remains on the archive
branch only. Thus the sidebar currently disagrees with the default-branch
history and API. This is an observed display discrepancy, not evidence that
the rewrite failed.

GitHub documents that contributor displays and statistics may take about
24 hours to refresh after a force-push or history rewrite. If the display is
still incorrect after that period, the repository owner should contact GitHub
Support. See [GitHub's contributor documentation](https://docs.github.com/en/repositories/viewing-activity-and-data-for-your-repository/viewing-a-projects-contributors#contributor-data-is-stale-after-history-changes).

Do not merge `archive/cqa-import-history-2026-09-26` into `main`: doing so would
make the old commits reachable from the default branch again. The archive is
preserved solely for attribution and audit. A repository content PR cannot
directly invalidate GitHub's Contributors sidebar cache.

## Open review

An independent human reviewer should compare the archive and new main refs,
confirm source/license attribution, and check GitHub's refreshed Contributors
panel. Do not close this tranche or claim a settled three-person GitHub panel
until that review is recorded.

No CVF runtime-governance behavior was asserted or tested by this work.
