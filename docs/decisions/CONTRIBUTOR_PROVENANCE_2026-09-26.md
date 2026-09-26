# Contributor provenance for the new product repository

Status: ACCEPTED
Tranche: CCMAI-CREDIT-001
Phase: DESIGN
Risk: R2 (rewrite of the public default branch)

## Decision

The owner requested that this repository present Blackbird081 as its human project owner, with Claude and Codex credited as supporting agents. SePay remains credited as the source of the inherited CQA code and MIT license. The GitHub Contributors panel is derived from default-branch commits, so README text alone cannot remove inherited CQA commit authors. The owner explicitly chose a history rewrite in this session.

Preserve the current complete history in the remote branch `archive/cqa-import-history-2026-09-26`. Rebuild `main` from source-tree milestones as an independent product history, with Blackbird081 as project importer and Claude/Codex co-author trailers where their assistance actually occurred. Use `--force-with-lease` against the inspected remote tip. Keep the source reference and SePay copyright in README/LICENSE. Do not change application source or pretend the imported CQA code was originally written by the project owner.

The archive branch retains the original authorship and commit chronology. GitHub may refresh the Contributors display asynchronously after the default branch changes. The repo's CVF runtime status is outside this tranche.

Source: [GitHub contributor graph rules](https://docs.github.com/en/repositories/viewing-activity-and-data-for-your-repository/viewing-a-projects-contributors), [GitHub co-author trailers](https://docs.github.com/en/pull-requests/how-tos/commit-changes/creating-a-commit-with-multiple-authors).
