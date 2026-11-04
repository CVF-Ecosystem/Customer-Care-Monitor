# R142 packet: docs site build repair (config only)

Status: COMPLETE_PENDING_REVIEW
workOrderStatusGrammar: cvf.workOrderStatusGrammar@1.0.0
Risk: R1
Worker: Cloud ORCHESTRATOR acting as IMPLEMENTATION_WORKER (docs config only)
Reviewer: Cloud ORCHESTRATOR acting as R1 REVIEWER (recorded role transition) plus the GitHub Actions docs build
Authority: `CVF_SESSION/authority/CCMAI-RUNTIME-142.json` (seed commit `38b76fd1214308376ebf64986f0803a9439f7da8`)

## Problem

`Build and Deploy Docs` has been red on `main` since at least run #37 (2026-10-21), and on every PR since.

- VitePress compiles each published markdown page as a Vue template.
- Historical review text such as `Promise<void>` (for example `reviews/R139_AUTONOMOUS_COMPLETION_ROUTE_2026-10-27.md`) is parsed as an unclosed HTML element: "Element is missing end tag".
- A static scan of the 732 published markdown files found 10 review files with raw tag-like text outside code spans.
- The same scan found no raw HTML in guide or home pages, and no `{{` interpolation anywhere.

## Change

`docs/.vitepress/config.ts` now sets `markdown: { html: false }`, so markdown-it escapes raw HTML instead of passing it to the Vue compiler.

- No evidence, review or receipt file is edited, which keeps historical bytes and hash pins intact.
- No dependency or CI workflow change.

## Verification

- **Objective check:** the GitHub Actions `Build and Deploy Docs` build job on the PR head must pass.
- No `npm ci` in the cloud (operator choice).
- **If the build reveals a further, different error:** the cloud agent may make at most one more config-only attempt within the seed. Otherwise it stops with BLOCKED_WITH_REASON and the exact log.

## Boundaries

- No product, runtime, AI-governance or M3 claim.
- No merge to `main`.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
