# CCMAI-REPO-MIGRATE-001 — repository migration BUILD evidence

**Date:** 2026-09-29 · **Operator:** Codex · **Status:** `REVIEW_PENDING` after BUILD; independent R2 REVIEW and FREEZE remain open. **Authority:** [work order](../work_orders/CCMAI_REPO_MIGRATE_001.md) and owner direction to move to `CVF-Ecosystem/Customer-Care-Monitor`.

## Identity and attribution

The owner-supplied GitHub screenshot shows exactly Blackbird081, claude and codex in the new repository sidebar, resolving the rendered-display observation left by CREDIT-002. `LICENSE` retains `Copyright (c) 2025 SePay`, `Copyright (c) 2026 CVF-Ecosystem`, and the MIT permission notice. The first commit message documents the CQA source import; it is historical provenance, not a claim that the current refactored and redesigned product is unchanged CQA. Neither license nor commit history was rewritten; the CQA archive tag is outside this migration.

## BUILD and local verification

- Changed the Go module path and exact internal imports in 60 backend files to `github.com/CVF-Ecosystem/Customer-Care-Monitor/backend`; the application release-check URL also uses the new repository. `go test ./...` and `go build ./...` passed across all backend packages. These tests did not assert CVF governance behavior and did not call a provider.
- Changed active README, installation, introduction, VitePress base/social, Channels and Settings links to the new repository and Pages path. `npm run build` passed in `frontend`.
- The pre-existing docs build first failed on literal `<time>` in a review page; escaping the placeholder exposed seven dead links. The excluded design decision now links to its repository page, and six directory-only screenshot links point to the new repository tree while embedded screenshots remain local. `npm run docs:build` passed. No historical PR/Action-run URLs were relabeled as new-repository records.
- Catalog `-Check`, `git diff --check` and workspace doctor passed (25/25). The new remote `main` was verified at the pre-migration local tip `6cb88b2bfd36d79cf5df2329acfea6b320c654a4` before switching origin.

## Remote completion and limits

- Before switching `origin`, the new remote `main` matched local `6cb88b2`. `origin` now fetches and pushes `https://github.com/CVF-Ecosystem/Customer-Care-Monitor.git`; only `main` was pushed, without force or tags. Remote `main` matched local `5bd35e2e386680d07b58edb96f7ed618675b9bcf`; the worktree was clean.
- GitHub Pages was created with `build_type=workflow` through the [Pages API](https://docs.github.com/en/rest/pages/pages#create-a-github-pages-site). The first [docs run 36460265525](https://github.com/CVF-Ecosystem/Customer-Care-Monitor/actions/runs/36460265525) built and uploaded the artifact, but deploy returned 404 because it started before Pages was enabled. After Pages creation, rerunning failed jobs made attempt 2 `success`. [Backend Go run 36460265523](https://github.com/CVF-Ecosystem/Customer-Care-Monitor/actions/runs/36460265523) also succeeded for this SHA.
- The new public home, installation, introduction and channel guide URLs each returned HTTP 200 under `https://cvf-ecosystem.github.io/Customer-Care-Monitor/`. Authenticated Pages API GET returned HTTP 200 with `build_type=workflow` and that URL. Anonymous Pages API GET returned 404 despite the public site returning 200; the public URLs and successful workflow are the deployment evidence.
- Historical links to old PRs, runs and CQA provenance remain historical. Those repository metadata do not move by pushing `main`. The owner intends to delete the old repository separately; no deletion was performed here. No provider key, real channel call, customer data operation, archive-tag push or CVF runtime governance proof is part of this BUILD.
