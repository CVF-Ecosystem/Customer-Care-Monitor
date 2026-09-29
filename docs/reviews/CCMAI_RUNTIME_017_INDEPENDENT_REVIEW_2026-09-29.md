# CCMAI-RUNTIME-017 independent R2 review

**Date:** 2026-09-29 · **Reviewer:** Codex, independent of Claude BUILD · **BUILD commit:** `986d00d` (parent `b8750fb`) · **Disposition:** `REVIEW_PASS / FREEZE_OPEN` · **Risk:** R2.

## Review

Rehydrated the manifest, policy, active state, handoff, memory, implementation status and documentation index. The pinned CVF core `26c686c` matches `origin/main`; workspace doctor passed 25/25. The compact bootstrap file is absent (`BOOTSTRAP_MIGRATION_PENDING`, non-blocking). Recorded `COMMIT_STEWARD (Claude) → REVIEWER (Codex)` in the active handoff before independent checks.

Inspected the exact `b8750fb..986d00d` diff against the [SPEC](../specs/RUNTIME_DASHBOARD_QC_VIOLATION_COUNT_S1_2026-09-29.md) and [work order](../work_orders/CCMAI_RUNTIME_017.md). Production changes are limited to `backend/api/handlers/dashboard.go`. The new COUNT uses the existing tenant ID and effective `from`/`to` values, exact `result_type = 'qc_violation'`, and no join. It adds numeric `qc_violation_count` while leaving the `issues` query and all previous JSON keys intact. A new-count query failure returns only `{"error":"dashboard_unavailable"}` with HTTP 500 before any dashboard success body.

The four new tests call the real handler against synthetic rows on disposable MySQL. They cover two tenants, mixed result types, two findings on one conversation, both inclusive date edges and outside rows, an empty match, a qualifying-row positive detector, and a forced count-query error. The rejection test checks HTTP status and exact response body. BUILD evidence records three mutations (missing type predicate, missing tenant predicate, swallowed query error) caught by the tests.

Independent command: `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./api/handlers -Run 'TestDashboardQC' -VerboseTests` → exit 0, all four tests PASS (`handlers` 5.101s); the script removed its disposable MySQL container and network. Claude's recorded full backend 14-package test, build and vet checks passed. Catalog `-Check`, workspace doctor 25/25, JSON parsing and `git diff --check` passed after review synchronization.

## Disposition and limits

R017 meets its additive backend API contract; `REVIEW_PASS / FREEZE_OPEN`. The frontend still displays the existing `issues` metric under its accurate total-results label; switching a card to QC violations needs a separate frontend tranche. This count reports database finding rows, not channel completeness or AI correctness. No real provider/channel call, CVF governance proof, deployment, push or FREEZE occurred. S1 remains IN_PROGRESS; Zalo/legacy recovery and other API/UI intersections remain separate.
