# CCMAI-RUNTIME-027 / F05 R1 — independent re-review

Date: 2026-10-02. Reviewer: Codex, independent of implementation/repair worker Claude. Risk R2.
Disposition: **REVIEW_PASS / FREEZE_OPEN** for repair `a9bff2275239f77d9af18995cae08e9f8933642f` on original BUILD `a1de36b6720cae4341f5937f6d9100b13bb48366`.
Authority: [SPEC](../specs/RUNTIME_ANALYZER_MODES_F05_2026-10-02.md), [work order](../work_orders/CCMAI_RUNTIME_027.md), unchanged Codex dispatcher seed/base `71c7ee761a1147d502d815ff5461e92f1c30bd4c`. [First review](CCMAI_RUNTIME_027_F05_INDEPENDENT_REVIEW_2026-10-02.md) remains historical CHANGES_REQUIRED evidence; [appended worker evidence](RUNTIME_ANALYZER_MODES_F05_BUILD_2026-10-02.md) describes R1. No reviewer product repair and no FREEZE/closure decision.

## Entry and source review

Rehydrated manifest/policy, bootstrap fallback state, memory/current prose, active handoff, implementation truth/index, SPEC/order/tranche/seed and appended evidence. Current surfaces agreed on R1 REVIEW_PENDING; no continuity drift. Compact bootstrap absent: BOOTSTRAP_MIGRATION_PENDING, nonblocking. Doctor 25/25 confirms public core remote/kit/clean worktree and manifest pin `26c686cc99b8be965d2760f27fe875b03376c643` matching origin/main. Knowledge ingest ran. Role REVIEWER (Codex), then SESSION_SYNC_STEWARD / COMMIT_STEWARD / ORCHESTRATOR for disposition synchronization. All 18 repair paths are within the existing authority; seed and analyzer_incremental.go unchanged. Retained reviewer HTTP/engine tests are byte-for-byte unchanged; mounted one-sided assertions remain intact.

## Disposition of F05-R1-01 through F05-R1-04

| Finding | Disposition and evidence |
| --- | --- |
| 01 — HTTP admission | PASS. parseTriggerParams validates full values/conflicts and unsigned decimal cap before config/dispatch. Empty/zero cap is unlimited. Original six-case review probe and expanded worker admission controls pass; invalid requests observe no dispatch/run/activity/cancel effects. Default since_last and legacy full=true remain. |
| 02 — UI one-sided dates/copy | PASS. Paired-date guards removed, reversed range and cap/at-least-one-condition guards retained. Original two mounted probes pass; copy names equal-timestamp omission and full local context. Named mounted request cases pass in browser zones UTC/New_York, with the original UTC/VN/New_York same-day cases retained. |
| 03 — test-run terminal bookkeeping | PASS. Removed run-only finalizer; every explicit mode uses existing checked transactional finalizer with nil checkpoint. Original single/batch review probe passes. Expanded checkpoint/outcome matrix and terminal write/missing-tenant faults pass; faults suppress notification and control notifies once. Shared R025 finalizer remains unchanged; ordinary cancellation/read-back/no-op and source-version regressions pass. |
| 04 — incomplete acceptance proof | PASS for bounded local contract. Added real TriggerJob -> real worker -> Analyzer -> MySQL conditional+VN-date+cap repeat-evaluation test passes and includes the earlier September message in sent/saved snapshot. Single/batch date oracle under UTC/VN driver locations, repeat evaluation/full snapshots, evaluation/anchor isolation, NULL anchor, equal-time cap and explicit terminal matrix pass. Worker supplies compiling old-source/pre-repair failures and restored mutations; these are worker evidence, independently assessed against source/tests rather than rerun as mutations here. |

The explicit cap no longer alters mode; full/date reruns include evaluated conversations and use full local snapshots. Dates select conversation last-message instants with Vietnam inclusive lower/exclusive next-midnight upper bounds. Only ordinary unlimited incremental execution advances last_run_at; test runs now also update terminal job status/updated_at. These intentional behavior changes remain within the accepted SPEC.

## Independently executed verification

Commands ran from project root; synthetic providers and disposable MySQL only. Actual child test exits were checked explicitly. Application source stayed at exact repair SHA throughout tests.

- `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./api/handlers -Run 'TestF05|TestTriggerJob|Test.*Business' -VerboseTests`: **18 top-level PASS**, zero failures/skips, package 17.333 s, actual exit 0. Includes original admission probe, conditional date+cap repeat route, named date admission and inherited R026 extrema/shared-parser regressions.
- `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./engine -Run 'TestF05|TestPlan|TestEntry|TestExplicit|TestSinceLast|TestUnanalyzed|TestFull|TestVietnam|TestOnlyOrdinary|TestOrdinary|TestNonOrdinary|Test.*Snapshot' -VerboseTests`: **62 top-level PASS**, zero failures/skips, package 159.977 s, actual exit 0. Includes four old-source-compatible probes on repaired source, original terminal probe, single/batch repeat snapshot, UTC/VN named oracle, evaluation/anchor controls, checkpoint/terminal faults and R025/snapshot regressions. This is focused engine validation, not every engine test or the full backend/R019 gate.
- `npm --prefix frontend test -- --reporter=dot`: **239/239 PASS**, 24 files, duration 34.30 s, actual exit 0. `node frontend/node_modules/vue-tsc/bin/vue-tsc.js -b frontend/tsconfig.json --force`: PASS. `npm --prefix frontend run build`: PASS, Vite 2.05 s.
- `PYTHONDONTWRITEBYTECODE=1`, `python -m unittest discover -s scripts/tests -p 'test_cvf_downstream_gate*.py'`: **46/46 PASS**, 24.181 s. `npm --prefix docs run docs:build`: PASS, 12.65 s before this disposition artifact; final docs build/preflight recorded in handoff. Doctor 25/25, diff check and 25-path pre-disposition full F05 explicit preflight 7/7 PASS, including catalog.
- Both test wrappers removed their disposable MySQL containers/networks; final docker ps has only the existing development services. git worktree list has only the main checkout. Existing untracked knowledge index and Python bytecode are excluded from review commit.

Worker full backend **824 PASS / 2 optional skips**, R019 five-sentinel/zero-DB-unavailable-skip gate, Go build/vet, old-source 4/4 failures and mutation kills remain specifically **Claude BUILD/repair evidence**. Reviewed test/source correspondence and reported cleanup; not independently re-executed full backend, old worktrees or mutations. No newer hosted CI success claimed; PR #1 remains draft at older remote head 3e0b37e.

## Limits and next governed move

- isOrdinaryIncremental is dead code in a file outside R027 allowlist; separate cleanup, nonblocking.
- since_last remains an event-time cursor with older/equal/backdated omissions, distinct from ordinary source-version selection.
- Full local context and explicit repeated analysis can increase work/transcript size; no provider quality or cost-saving proof follows.
- Mounted UI checks cover supported named cases, not extrema. Contrary to the worker's shorthand that a date input cannot produce min/max values, an HTML date input can represent such calendar values; rejection/representability is authoritative at shared parser/API/engine, with inherited extrema tests retained. No blanket browser validation or real browser-to-server proof is claimed.
- Synthetic evidence proves local selection, persistence and request contracts; **no CVF AI governance claim or receipt**, real provider/channel/notification delivery, persistent DB change, timestamp/DSN migration, tenant timezone activation, F06 ownership/cancellation or upstream completeness proof.

R027/F05 source remediation has passed independent REVIEW; FREEZE stays OPEN. R025/R026 remain REVIEW_PASS / FREEZE_OPEN. Next: owner may dispatch a separate bounded F06 INTAKE/DESIGN/SPEC/WORK_ORDER; no F06 BUILD is granted by this review. F02 live/message limitations and F07 remain separate. No push, merge, deployment or parent-CVF work.
