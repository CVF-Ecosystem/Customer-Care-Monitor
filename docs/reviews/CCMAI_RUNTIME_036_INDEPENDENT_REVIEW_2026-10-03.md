# R036 independent exact-BUILD review

Date: 2026-10-03. Reviewer: Codex, independent of Claude IMPLEMENTATION_WORKER. Exact BUILD `a805db2bbb3b30ea1841537351a538c37a1af28b`; dispatcher seed `5a044ae82d11cdf59cdf28d0a8fe9f9410957bb3`, authored/committed by Codex before dispatch/BUILD. **REVIEW_PASS / REVIEW / FREEZE_OPEN** for SS-01..06 synthetic frontend structure/navigation only. [SPEC](../specs/SETUP_STATUS_RECOVERY_R036_2026-10-03.md), [order](../work_orders/CCMAI_RUNTIME_036.md), [worker evidence](SETUP_STATUS_RECOVERY_R036_BUILD_2026-10-03.md). Sanitized replay summary: `docs/reviews/probes/r036_independent_summary.json`.

## Scope and result

Canonical continuity rehydrated before checks, doctor25/25 PASS; no header drift, compact bootstrap absent/nonblocking. Seed fields/risk/roles/effects exactly match the tranche, seed unchanged and present at baseCommit. Source at current hand-back equals exact BUILD; all eight frontend blob digests independently computed. Working-copy CRLF normalized to LF equals the blobs. This comparison does not claim historical byte restoration by the worker or cross-tab behavior. BUILD contains the eight authorized frontend paths and bounded governed documentation; backend/auth store/Setup view/interceptors/layouts/dependencies/workflows/seed unchanged. Worker acknowledgment is recorded as before-edit; a single BUILD commit cannot independently establish intra-worktree edit timing.

Source and executable tests cover strict boolean confirmation, one cached confirmed result, unavailable on errors/malformed payloads, no navigation-based retry, single-flight explicit Retry, status-only8000ms deadline, preserved credentials while uncertain, AUTH-001 confirmed-required clearing and fixed local configured recovery. App waits initial readiness before mounting default layout and loads an existing-token profile only on configured status, including recovery from unavailable. No blocking product defect found; no reviewer source/test repair.

| Contract | Independent evidence |
| --- | --- |
| SS-01..02 | Actual router/App tests: deferred loading, concurrent navigation, failure variants and no premature request/layout effects; HTTP401 probe runs real interceptors with mocked adapter. Fake-timer deadline test verifies abort and ignores late answer. |
| SS-03 | Real Vuetify unavailable view mounted in vi/en, rendering native button, busy/disabled state, single-flight double activation and repeated handled failures. Layout and RouterView are stubbed in App tests; no browser screenshot or actual app-shell E2E claim. |
| SS-04..05 | Retained AUTH-001 positives plus retry true/false, fixed local recovery/no return URL, confirmed cache and one post-recovery profile fetch. Full suite299/299 PASS,0 pending/todo;27 files,13.403s reporter duration. |
| SS-06 | Isolated exact-BUILD archive baseline38/38, four applied mutations killed by named assertions, bytes restored, final38/38 PASS. Worker ten-mutant campaign remains separately attributed; not all ten independently replayed. |

Independent forced typecheck and frontend production build exit0 (Vite bundle940ms). These are local checks using cached dependencies, with no downloads or live calls. Required documentation/gate validation is recorded in the active handoff before review commit.

## Applied regression controls and honest failures

| Replay | Result |
| --- | --- |
| MS1: failure -> configured | KILLED:26 semantic failures,12 pass,5.877s; named old AUTH-001 failure expectation detects entry into Setup/app instead of unavailable. |
| MS3: remove configured condition from profile gate | KILLED:1 failure,37 pass,5.900s; unavailable App fetches profile prematurely (expected0, actual1). This variant retains readiness gating. |
| MS4: remove retry in-flight check | KILLED:1 failure,37 pass,5.790s; double Retry requests3 instead of2. |
| MS9: remove watchdog resolution | KILLED:1 failure,37 pass,5.841s; fake-timer deadline leaves unresolved instead of unavailable. |
| Byte-restored exact BUILD |38/38 PASS,0 pending,5.852s; all three touched archive source files equal original bytes. Workspace source untouched. |

Initial MS4 preparation failed an exact LF-only match against CRLF archive bytes: **NOT_APPLIED**, no mutation/test performed in that attempt; finally restoration verified. Adapted the matching newline representation and reran MS4/MS9 successfully. This preparation failure is retained in the summary and not counted as a kill.

Broad old-source swap (pre-R036 router/App with current tests) produced31 failures/7 pass in10.819s. Two failures were an instrumentation TypeError for the missing old status-request config and an unclassified STACK_TRACE_ERROR: **INCONCLUSIVE**, not semantic detector proof. Other failures include named route assertions. To establish the original defect unambiguously, a separate named AUTH-001 failed-status control on old source fails1/1 with `expected tenants to be setup-unavailable`; byte-restored control passes1/1. Each named run intentionally filters six other AUTH-001 tests (6 pending), unlike the complete suite with zero pending. No unexplained error is relabeled PASS. Raw archive/logs are retained outside Git in the task temp directory.

## Documentation findings and disposition

R036-DOC-01: current handoff/catalog still said DISPATCH_READY/NOT_BUILT after hand-back; SPEC intake used current-tense pre-BUILD source wording. R036-DOC-02: abbreviated vi blob hash in worker table had an incorrect suffix (`4073b09` instead of actual `e073b09`). Full primary-source blob hashes otherwise match. These documentation-only findings are settled explicitly during reviewer-owned SESSION_SYNC_STEWARD publication: mark dispatch-time paragraphs historical, align current routing/catalog and add correct full vi/en blob hashes to worker evidence with a dated correction. Source/test/seed stays identical to exact BUILD. No Claude repair or product self-approval is introduced by this synchronization.

## Boundaries retained

Happy-dom, mocked API/adapter and in-memory storage only. No multiple tabs, real browser E2E, real server/network timing or cookie refresh. Eight-second deadline uses fake timers; only status requests are bounded, common120s client timeout and global interceptors unchanged. Backend/DB/race/live/provider/external network/GitHub NOT RUN. All worker failed history, previous R035 NOT RUN and GORM/checksum/default-client limits remain source-specific history. No runtime AI/CVF governance, backend-security, live availability, hosted-readiness or new FREEZE claim. Actual MCP job execution and live Pancake inputs/authority remain separate. R035/R034 acceptance and R033 local-message FREEZE unchanged.

Publication checks: gate46/46 PASS14.832s; default/PR/scoped preflight7/7 PASS; docs build final6.68s PASS, inherited env-highlighter warnings. Intermediate scoped catalog failure from stale generated index after a label edit is retained; regeneration and rerun PASS. All twelve changed files are governed documentation/evidence, product/test/seed unchanged.
