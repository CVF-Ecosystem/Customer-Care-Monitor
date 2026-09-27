# SPEC — Display data fixes (`CCMAI-UX-001a`)

**Date:** 2026-09-28 · **Author:** Claude (`SPEC_AUTHOR`) · **Risk:** R2 (numbers and times users read as facts; one backend demo-data change) · **Work order:** [`CCMAI_UX_001A`](../work_orders/CCMAI_UX_001A.md) · **Findings:** UX-01, UX-02 (label only), UX-03, UX-05 in the [baseline](../reviews/UI_UX_REVIEW_BASELINE_2026-09-27.md) · **Roadmap:** Phase 1.

**Dependency (stacked, declared):** uses `formatRelative` from `CCMAI-UX-000` (`frontend/src/utils/format.ts`, REVIEW_PENDING) and adds one helper to that file. Review UX-000 first; if UX-000 changes `formatRelative`, this tranche follows.

## Source truth at `96d2472`

| Finding | Cause |
|---|---|
| UX-01 negative "phút trước" | `Dashboard.vue` `timeAgo()` returns negative minutes for future timestamps. `demo.go` builds `baseTime` with `Truncate(24h)` (UTC midnight) + 8–20 h, so same-day demo conversations can lie in the future, and evaluation time = last message + 30 min. |
| UX-02 "Vấn đề: 385" | `dashboard.go` counts **all** `job_results` in the period (evaluations + tags + violations) under the label "Vấn đề". |
| UX-03 "1 vấn đề" on classification | `JobDetail.vue` card header shows `violations.length + ' vấn đề'` for both job types; for classification those entries are tags. |
| UX-05 duplicate/missing days | `JobDetail.vue` trend keys by `toISOString().slice(0,10)` (UTC date) but labels with local `getDate()`; the two disagree around midnight. |

## Required behavior

1. **UX-01 frontend.** Dashboard recent activity uses `formatRelative` (never negative, minutes < 60, hours < 24, then days).
2. **UX-01 demo data.** Demo conversations are placed on Vietnam business hours (08:00–20:00 at UTC+07:00) and shifted back one day while their last message would fall later than the demo runs' start (`now − 2h`). Result: no demo message, conversation or result timestamp is in the future, and every demo result is created before the demo runs finish. Content, counts, IDs, statuses and the R005 confidence rule are unchanged. Existing databases are not rewritten.
3. **UX-02 label (frontend only).** The dashboard card keeps its API value but is labelled for what it counts: "Kết quả đánh giá" with the hint "Mọi kết quả trong khoảng thời gian: đánh giá, vấn đề và nhãn". Counting only `qc_violation` would change the meaning of the existing `issues` API field and is **BLOCKED** (`BLOCKED_API_CONTRACT`) pending an owner/Codex decision (option: add a new read-only field in a backend tranche).
4. **UX-03.** On classification jobs the card header shows "N nhãn"; QC keeps "N vấn đề".
5. **UX-05.** The trend groups and labels by the same Vietnam calendar day (`Asia/Ho_Chi_Minh`, via a new `vnDateKey` helper in `format.ts`), sorted chronologically; one point per day.

## Acceptance

- Go test on disposable MySQL (`scripts/test-backend.ps1`): with the demo clock fixed at an early-morning instant, import produces 0 messages/conversations/results later than the runs' start/finish, and every conversation starts between 08:00 and 20:00 UTC+07:00. The test fails on the pre-change `demo.go`.
- Vitest: `vnDateKey` around the UTC/VN midnight boundary; the trend helper returns one point per VN day; classification label; dashboard label keys exist in `vi`/`en`.
- `vue-tsc -b --force`, build, `npm test`, full backend suite, `go vet`, `gofmt`, screenshots of dashboard and job detail (desktop/mobile, light/dark) with 0 JS errors.

## Out of scope

UX-04 (two sources for Job Detail counts — needs a data-source decision), UX-06 (sync status origin, investigated separately), the backend recount for UX-02, any API change.
