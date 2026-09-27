# Independent REVIEW: CCMAI-RUNTIME-006 job-result source integrity

**Reviewer:** Codex (`REVIEWER`) · **Date:** 2026-09-27 · **Target:** local BUILD commit `61a775a` · **Disposition:** `CHANGES_REQUIRED` for one UI claim-boundary repair (`R006-R1`). R006 remains `REVIEW_PENDING`; no FREEZE.

## Accepted checks

- The changed paths match the R006 work order. The shared `computeSourceIntegrity` uses tenant-scoped chunked snapshot/message queries and R004's provenance/comparison functions. Job-result JSON handlers attach one of the four statuses per row; QC/classification CSV and XLSX exports retain every distinct status in a grouped conversation and add per-result traceability. Query errors are checked before file headers/bytes. R005 confidence serialization is carried through the embedded result model.
- The classification export labels chat content as read at export time. Frontend group logic lists distinct statuses most concerning first; unknown/missing status maps to unavailable. The table, card and dialog bind status labels/icons. The existing test covers label presence and grouping, and the frontend build is recorded as passing in BUILD evidence.
- Codex independently ran the focused R006 handler tests plus the selected R004 aggregate failure regressions with `-count=1` on a fresh disposable `mysql:8.0` `CCMA` database, using `golang:1.26-alpine`, a read-only source/module-cache mount, `GOPROXY=off` and `GOFLAGS=-mod=readonly`. Result: `ok` for `./api/handlers`; no test skip was used for these cases. MySQL SQL readiness was confirmed first, and the disposable container/network were removed. Claude's separate BUILD evidence records all 13 backend packages, vitest 8/8, frontend build, build/vet, catalog and doctor passing. Codex also ran workspace doctor 25/25.

## Finding R006-R1 — local-only caveat is hidden in the dialog

The SPEC requires an understandable warning on job table/card/dialog views **with a visible explanation** that `bound_currentness_unverified` is only a local comparison. `JobDetail.vue` adds the `results_source_note` alert only inside the detail dialog (`v-card-text`, around line 635). The results-tab table and card views show icons/chips and short labels but no local-only explanation until a user opens a dialog. The short label “Chưa xác minh đầy đủ” does not explain the upstream history limit. The existing i18n/grouping test verifies keys and ordering, not placement of that explanation. A user can therefore read a bound result in the primary job view without seeing the qualification required by the SPEC.

**Repair acceptance:** place the existing bilingual `results_source_note` visibly in the results tab above the table/card switch, so it remains visible with either view when results are shown. Keep status badges and the dialog note; do not change status computation or other product behavior. Verify the template location by source inspection and run frontend `npm run build` and the existing focused i18n/grouping test. Append commands/results to BUILD evidence. This is a same-scope R2 repair in `frontend/src/views/Jobs/JobDetail.vue` plus evidence/continuity only; no new path, provider call or database effect is needed. Return one local repair commit as `REVIEW_PENDING` for Codex re-review.

## Boundary

The local status does not prove upstream completeness. S1 remains IN_PROGRESS; R001–R005 remain REVIEW PASS / FREEZE open. No provider API, real channel sync, customer data, persistent Compose database change, deployment, push, S2/S3/S5 implementation or FREEZE is authorized by this review.
