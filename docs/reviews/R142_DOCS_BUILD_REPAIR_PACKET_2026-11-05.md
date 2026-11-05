# R142 packet: docs site build repair (config only)

Status: CLOSED
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
- The same scan found no raw HTML in guide or home pages.

## Change

`docs/.vitepress/config.ts` now sets `markdown: { html: false }`, so markdown-it escapes raw HTML instead of passing it to the Vue compiler.

- No evidence, review or receipt file is edited, which keeps historical bytes and hash pins intact.
- No dependency or CI workflow change.

## CI attempt 1 result and attempt 2 change

- **Attempt 1** (head `ad146cd`, [run](https://github.com/CVF-Ecosystem/Customer-Care-Monitor/actions/runs/41928227654)): the original `Promise<void>` error was gone. The build then failed on this packet's own text: an inline code span containing a double opening brace raised "Interpolation end sign was not found".
- **Cause:** VitePress adds `v-pre` to fenced code blocks but not to inline code spans. The first scan wrongly treated inline code as safe.
- A corrected scan found one historical file (`reviews/R139_CLOCK_V3_G0_TECHNICAL_REVIEW_2026-10-26.md`) with a double-brace placeholder in inline code. It does not break the build, but the placeholder text was being dropped from the rendered page.
- **Attempt 2 change, still config-only:** a markdown-it hook adds `v-pre` to every inline `<code>`, and this packet line was reworded. This is the last CI attempt the seed allows. If it fails, the tranche stops at BLOCKED_WITH_REASON with the log.

## CI attempt 2 result: stopped

- **Attempt 2** (head `edfe97b`, [run](https://github.com/CVF-Ecosystem/Customer-Care-Monitor/actions/runs/41928475690)): every Vue compile error is gone, including `Promise<void>` and the inline-code double brace.
- The build now fails only on 4 dead links to `roadmaps/CCMA_DELIVERY_ACCEPTANCE_PLAN_2026-10-21`, from `reviews/learnings/feedback_orchestrator_roadmap_control.md` and `reviews/planning/CCMA_PRODUCT_DELIVERY_ROADMAP_2026-10-13.md` (3 links).
- `roadmaps/**` is in `srcExclude` but missing from `ignoreDeadLinks`, unlike `specs/` and `work_orders/`.
- The seed's two CI attempts are spent, so the tranche stops at BLOCKED_WITH_REASON.
- **Proposed next fix (needs operator approval of one more CI attempt):** change `ignoreDeadLinks` to `[/(^|\/)(specs|work_orders|roadmaps)\//]`.

## CI attempt 3 (operator-approved)

The operator approved one more config-only attempt. `ignoreDeadLinks` now also covers `roadmaps/`. If this fails, the tranche stops at BLOCKED_WITH_REASON again and asks before any further attempt.

## Verification

- **Objective check:** the GitHub Actions `Build and Deploy Docs` build job on the PR head must pass.
- No `npm ci` in the cloud (operator choice).
- **If the build reveals a further, different error:** the cloud agent may make at most one more config-only attempt within the seed. Otherwise it stops with BLOCKED_WITH_REASON and the exact log.

## Boundaries

- No product, runtime, AI-governance or M3 claim.
- No merge to `main`.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
