# CCMAI-RUNTIME-026 / F04 — independent review

Date: 2026-10-02 (Asia/Saigon). Reviewer: Codex; implementation worker: Claude. Risk R2. Disposition: REVIEW_PASS with bounded reviewer repair / FREEZE_OPEN.

## Exact source, continuity and authority

- Reviewed BUILD: `41b6c5006e474e282875af712ac5283cce35d6e8`; planning parent `b160cecf7138d9abdc86b1e8b41dcb9f2fa4e77d`. Dispatcher seed/base `8f366ebd3ed2df3a1302a0635844d9a5f5d0d1a9` is unchanged, committed before BUILD. All 32 BUILD paths fit the seed plus standard continuity/review artifacts.
- At entry, only pre-existing untracked `knowledge/_index.json`; excluded from commits. Reviewer changes below are separate from Claude's BUILD and committed with this review, not attributed to the worker.
- Rehydrated manifest/policy/state/memory/handoff/status/index, [SPEC](../specs/RUNTIME_BUSINESS_DAY_FILTERS_F04_2026-10-02.md), [work order](../work_orders/CCMAI_RUNTIME_026.md) and seed/record. Missing compact bootstrap: BOOTSTRAP_MIGRATION_PENDING, nonblocking. Core `26c686cc99b8be965d2760f27fe875b03376c643`; doctor 25/25 PASS; knowledge ingest ran.
- During source/probe review, stale implementation prose still said F04-F07 untouched and Claude BUILD pending. Reported BLOCKED_CONTINUITY_DRIFT; correction `a1bee750cfbe0799dd1ffc9f98d5d71c4c638d41` narrowed to F05-F07 and independent REVIEW pending before disposition. Machine pointers/current memory prose were already REVIEW_PENDING. Rehydrated after correction. [Parent learning intake](learnings/CCMAI_TO_CVF_DOWNSTREAM_GATE_LEARNING_INTAKE_2026-10-01.md) extends the same duplicate-current-prose gap; no parent implementation/tests.

## Contract decisions

| Area | Independent conclusion |
| --- | --- |
| Intentional API changes | Exact-seven/exact-28 presets, no-date Dashboard = VN today, one-sided supplied interval open on the opposite side, invalid dates = bounded 400 all match SPEC. Public to remains inclusive **date**, internal upper bound is exclusive next-midnight. |
| Storage and grouping | Typed range/CASE parameters keep the driver's location authoritative. Both real UTC/VN connections write/read distinct wall times and return matching results. Two bounded grouped scans over the same 31-date horizon, 30 CASE comparisons per row; no timezone tables, session/DSN change or full-message loading. This proves fixture semantics, not identification of arbitrary historical mixed encodings. |
| Dashboard | Filtered counts/channel counts/recent rows/cost_period share one interval; today/month costs and series have upper bounds independent of selection. Date buckets and role/distinct-chat/count meanings preserved. Modified queries check errors; static/onboarding errors remain outside scope. |
| Results / Cost Logs / exports | List/count/export share parsed bounds; conv vs eval identity, pagination/provider/tenant/channel filters and source-integrity/confidence contracts preserved. CSV/XLSX print VN times; TXT/CSV message export still selects conversations and exports all their messages. |
| Frontend | Calendar arithmetic uses VN keys and UTC calendar getters, independent of the browser zone. Labels describe VN time, not the inactive Settings selector. Mounted views capture actual list/export parameters against matching named backend cases. No actual browser-to-server execution is claimed. |
| Changed R017 tests | VN first/last-millisecond edges replace obsolete UTC/BETWEEN edges. Added 01:00 VN row prevents lost/gained rows cancelling in a count; expected QC 5 / issues 7 and increment checks remain scoped. No DB assertion weakened. |
| Claim boundaries | F04 acceptance is the covered local read/report contract. Mixed historical storage, tenant timezone activation, general writer migration, F05 analyzer dates/modes, F06 ownership and F02 real/message coverage stay separate. No provider-quality, upstream-completeness or CVF AI-governance claim. |

## Independent finding: validate actual supplied bounds

The BUILD checked calendar labels against a numeric year range rather than the **actual bounds used by the query**:

1. A retained probe against BUILD failed in three cases (from=1000-01-01 with omitted/same/next to). The parser returned nil error and a VN midnight whose UTC instant is in year 0999, outside DATETIME for the supported UTC connection. Existing positive test wrongly admitted that lower bound.
2. A second retained probe after the lower fix (upper validation still unchanged from BUILD) failed: from=9999-12-31 with omitted to was rejected, though the supplied lower bound is representable on UTC/VN connections and no next-midnight upper bound is requested. A supplied to=9999-12-31 must remain rejected because the VN next-midnight wall time reaches year 10000.

These are one local validation root cause. Reviewer-local repair route: same objective, acceptance contract, authorized handler/test paths, R2/effects; no schema, DSN, migration, real provider/channel or credential use, new interface or F05/F06 edit. Under the explicit small-repair rule, Codex acts as REVIEWER performing bounded repair, then SESSION_SYNC_STEWARD/COMMIT_STEWARD/ORCHESTRATOR; no extra repair tranche. Claude's independent BUILD attribution is retained.

Repair in `business_dates.go`: reject an actual supplied lower instant below the common UTC minimum; allow the last calendar label to parse; check year overflow only when generating a supplied to's exclusive boundary. In `business_dates_test.go`, three retained reviewer tests cover the failures, adjacent/to-only positives and all five handlers returning exact 400 JSON before querying (host test has no DB configured). Two original representable-limit expectations were corrected to this actual-bound contract; ordinary dates/assertions remain.

## Independently executed evidence

- Original BUILD focused disposable MySQL: `scripts/test-backend.ps1 -Packages ./api/handlers -Run 'Business|StorageMatrix|DashboardQCViolation|Results' -VerboseTests`; 18 top-level PASS, 6.131 s, exit 0, no skips. This initial selection did not include all message/Cost Logs tests; subsequent full handlers run did.
- Host negative probes: `go test ./api/handlers -run '^TestBusinessReview' -count=1 -v` failed the three lower-bound cases (0.364 s). Later upper-only probe failed (0.349 s). Negative behavior is recorded above; compilation succeeded.
- Host final parser/API probes: `go test ./api/handlers -run 'BusinessReview|ParseBusinessRange' -count=1 -v`: six top-level PASS, 0.339 s, including five handler subcases without DB configuration.
- After lower-bound repair, all handlers on disposable MySQL: `scripts/test-backend.ps1 -Packages ./api/handlers -VerboseTests`: **122 top-level PASS, 0 skips**, 44.277 s, exit 0. Upper-only refinement followed this run; final source focused run (`Business|StorageMatrix|Vietnam|MessageExport|CostLogs|DashboardQCViolation|Results`) passed 26 top-level tests with 0 skips, 9.153 s, exit 0. Disposable DB/network removed; persistent Compose services untouched.
- Frontend full `npx vitest run`: 23 files / **214 tests PASS**, 12.74 s. Forced `npx vue-tsc -b --force` and `npx vite build` PASS. Backend `go build ./...` PASS. No frontend source repair was needed.

Local logs: `%TEMP%/ccmai-r026-review-backend.log`, `ccmai-r026-review-final-handlers.log`, `ccmai-r026-review-final-extrema.log`. No credentials/log dump copied into repository.

Worker [BUILD evidence](RUNTIME_BUSINESS_DAY_FILTERS_F04_BUILD_2026-10-02.md) reports full backend 635 pass / 0 fail / 2 optional skips, R019 five-sentinel gate, old-source handler/view failures and 9/9 mutations. Those were inspected as worker evidence, not independently rerun in their entirety. Its statement about no mixed encoding applies to source/fixtures examined, not an audit of historical persistent data; that remains explicitly unproved/outside scope.

## Disposition and repository checks

REVIEW_PASS for the covered read/report contract on BUILD plus reviewer fixes. Original extrema validation failed independent probes; corrected-source tests pass. Gate unit tests 46/46 PASS (18.135 s); docs build PASS (8.24 s); doctor 25/25 and diff check PASS. Final explicit ten-file repository preflight passed 7/7 including catalog; final docs build passed (5.63 s) and final Go build passed; unrelated knowledge index is excluded, so this is not whole-worktree PASS. Next governed move after REVIEW_PASS: F05 DESIGN/SPEC/WORK_ORDER. FREEZE remains OPEN; no push, merge, deployment, provider/channel calls or parent-CVF work. Historical F08/GOV hosted evidence at draft PR head `3e0b37ea729c75fe1a319dda61ee384b71706e7c` does not prove these later local changes.
