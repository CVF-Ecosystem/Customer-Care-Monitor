# CCMAI-RUNTIME-017 — Dashboard QC violation count API

**Date:** 2026-09-29 · **Phase:** SPEC · **Risk:** R2 · **Authority:** [S1 roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), [UX-001a contract note](DISPLAY_DATA_FIXES_UX001A_2026-09-28.md), [UX-011 INTAKE](RESULTS_SCREEN_UX011_INTAKE_2026-09-28.md).

## Decision

Add `qc_violation_count` to `GET /dashboard` as a read-only, tenant-scoped count of `job_results` rows whose `result_type = 'qc_violation'` and `created_at` falls in the same effective `from`/`to` interval as the existing `issues` field. Count finding rows, not distinct conversations, jobs or runs. An empty match returns numeric `0`.

Keep `issues` byte-for-byte compatible in meaning and JSON type: it counts **all** job-result rows in the selected interval, including evaluations and classification tags. Keep existing date parsing/defaults and other response keys unchanged. No frontend card switch in R017; a later frontend tranche may choose which metric to show and how to label it.

## Required behavior

1. A tenant sees only its own QC violation rows. Multiple violations on one conversation count separately; `conversation_evaluation` and `classification_tag` do not count. Rows outside the selected interval do not count. `issues` still includes all tenant result types inside the interval.
2. For a valid request and healthy DB, return HTTP 200 with both integer fields. For a failure of the new count query, return a generic HTTP 500 `dashboard_unavailable` response; do not serialize a false zero or SQL details. Do not alter error handling of unrelated existing queries in this bounded tranche.
3. Use the existing tenant context and effective date bounds. No join or relation that can multiply a result row. No schema migration or write.
4. Tests exercise the real dashboard handler with disposable MySQL: mixed result types, two tenants, multiple QC violations for one conversation, inside/outside date bounds, empty period, query failure, and presence/meaning of both fields. A positive detector must prove the new field changes when a qualifying row is added. Direct rejection/error tests must inspect HTTP status and body.

## Boundary and claim limits

Only the additive backend API contract is in scope. Frontend, API client/store types, other endpoints, result write logic, job-run selection, adapter/provider behavior and CVF governance are out of scope. The number is a database row count, not a claim that the upstream channel is complete or that an AI finding is correct. Synthetic MySQL tests do not establish live governance proof.

**Exit:** independent R2 review confirms the contract and failure behavior; FREEZE remains a separate decision. S1 remains IN_PROGRESS.
