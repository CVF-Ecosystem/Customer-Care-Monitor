# CCMAI-RUNTIME-032 / F02-F — independent review

Disposition: CHANGES_REQUIRED / REVIEW / FREEZE_OPEN. Date: 2026-10-03 (Asia/Saigon). Reviewer: Codex, independent of Claude implementation/repair worker. Exact BUILD `b31b9749664485e686838b47cc37d7433eb0179d`; `26126b1` supplies the SHA and changes no product source. Authority: [SPEC](../specs/RUNTIME_ZALO_MESSAGE_COVERAGE_F02F_2026-10-03.md), [order](../work_orders/CCMAI_RUNTIME_032.md), seed first committed at `6983a891b58272eb74e6ae0b793a71a55d1d8f8e`. Review intake declaration/role acknowledgment is in the active workspace handoff `CVF_SESSION/handoffs/AGENT_HANDOFF_F02F_2026-10-03.md`.

## Required findings (consolidated R032-R1 repair scope)

| ID | Priority / target | Independently observed finding | Required repair and acceptance |
| --- | --- | --- | --- |
| R032-R1-01 | P2 detector/evidence; `backend/channels/zalo_messages_test.go:453`, BUILD M13 evidence | Applied one-match mutation disabling `if seenPages[fp]` in an isolated exact-BUILD archive. Running `go test ./channels -count=1 -timeout 120s -json -run '^TestZaloMessagesRepeatedPageFails$'` exits1 with `panic: runtime error: index out of range` in `cycle_A-B-A`. The page generator indexes a three-element slice with `n`; its caller's ceiling `n > 5` is too late. This is a fixture failure, not the named semantic assertion claimed for M13. Original worker mutation source/log was not available for exact replay; the reviewer mutation is explicit and its outcome is independently observed. | Make the committed cycle fixture finite and valid after disabling the guard: valid recurring A/B pages until an outer ceiling, or an explicit terminal/control after the initial A-B-A. Assert the intended repeat class and exact request count so guard removal produces a named assertion with no panic/timeout/build error. Apply the mutation, capture the assertion, restore identical source bytes and rerun positive controls. Correct worker evidence so the earlier reported M13 kill is not independently certified; preserve original worker M7/M9 build INCONCLUSIVE and all first results. |
| R032-R1-02 | P2 continuity prose; SPEC implementation-truth paragraph, memory current tranche, status current R032 record | SPEC ends with **Current status DISPATCH_READY / WORK_ORDER** and future BUILD authority after its header says BUILD complete / REVIEW_PENDING. Current memory still says Claude owns **future bounded BUILD**; IMPLEMENTATION_STATUS says the implemented traversal/mapping are **to be implemented**. Machine front pointers agree REVIEW, but current implementation prose contradicts them. A blanket historical note does not identify these current statements. | Synchronize each current statement to the actual R1 BUILD/return/re-review state; mark historical planning paragraphs locally as historical. Update SPEC implementation pointer, memory/current status, order/handoff/tranche/roadmap and index/catalog as applicable. Preserve intended contract and earlier evidence. Run continuity/default/PR-range/catalog checks; no false current DISPATCH_READY or future initial BUILD statement. |

These are test/evidence/continuity repairs under the existing immutable R032 scope. No new product behavior or wider authority is requested. Reviewer makes no source/test repair. Existing order failure conditions require CHANGES_REQUIRED for weak detector/incomplete current prose. An independently finite reviewer probe confirms the production repeat guard, but does not repair the committed fixture or retrospectively certify the worker M13 assertion.

## Source, scope and acceptance assessment

Source inspection and independent adapter reproduction support F02F-01..06's local contract: exact rows/numbers, physical offsets to explicit empty, full-history/no-since-filter, inherited mapping, stable order/dedup/conflict handling, finite budget/context checks, message-only blocked redirects, bounded bodies and safe errors. This is a provisional assessment, not REVIEW_PASS while the two findings remain open. F02F-07..08 local paths passed the independent focused MySQL verification below; worker full-backend/mutations remain attributed separately. F02F-09 detector/continuity acceptance is held for R1.

Seed Git blobs are identical at first commit/baseCommit and exact BUILD; roles remain Claude implementation/commit, Codex independent review. Git diff confirms engine source, shared request/refresh/conversation/OAuth functions, other-platform source and tracked tooling unchanged. `extractZaloDataArray` removal has no remaining caller. No live provider/channel/credential action or persistent DB was performed.

Two changed R024 assertions are contract replacements, not hidden removals: short-page request1 -> page+empty request2, and105 ->210 message requests for105 conversations. Original mapping/storage assertions remain. Their fixture extensions are scoped to accepted new paging behavior. Optional-string null and missing error-key rejection match the specified conservative local contract; no live compatibility claim follows. Empty `type` -> default text is specified; object-link precedence remains inherited, live string-link mapping unverified.

## Independent commands and results

All commands run from project-root cwd; isolated probes use `go -C <OS-temp exact-BUILD archive>/backend`. No mutation touched the shared project source. Original-source comparison replaces only the isolated adapter with `git show 21e5795:backend/channels/zalo_oa.go` plus a temporary message sentinel declaration so current tests compile. Raw logs stay outside Git, never pasted as credential/customer evidence.

| Check | Independent result |
| --- | --- |
| `go -C backend test ./channels -count=1 -json` | PASS94 top-level /192 including subtests, zero FAIL/SKIP. |
| `go -C backend build ./...`; `go -C backend vet ./...` | Both exit0. |
| Isolated baseline / restored baseline, selection `^(TestZaloMessages\|TestZaloFetchMessagesMappingUnchanged\|TestR032IndependentFiniteCycle)` | Each PASS19 top-level /106 total, zero FAIL/SKIP. Reviewer finite-cycle probe is additional ephemeral evidence, never committed product test. |
| Original adapter against selected retained/new probes |6 top-level /78 total named behavioral failures: mapping paging, full-history traversal, short/duplicate pages, malformed-success, redirect and blank-conversation-ID. No source acceptance attributed to original adapter. |
| Eleven applied one-match reviewer mutations | All semantic failures: short terminal, malformed data empty success, fixed-size offset, validation-after-duplicate, budget disabled, terminal context disabled, redirects followed, link override disabled, conflicting row accepted, sort disabled, repeat guard disabled. Each restored byte identity; no compilation error, panic, timeout or no-op earns kill credit in this campaign. |
| Worker-style cycle fixture with repeat guard disabled | Exit1 **fixture panic**, separate from the finite reviewer semantic detector. R032-R1-01 open. |
| Full uncached backend: isolated internal Docker network/no host port, source/module cache read-only, existing image/no dependency downloads, `-p 1 -timeout 40m -json`, MySQL `max_connections=1000` | **INTERRUPTED_BY_REVIEWER**, wrapper exit1 after stopping only its exact named Go runner. Stopped once consolidated findings established; incomplete run is not PASS or a reproduced product failure. Partial package/test outcomes retained separately. Disposal completed before focused run. Worker full1074 PASS/2 external SKIP/R019 PASS remains worker-reported, not relabeled as reviewer proof. |
| Focused actual Zalo adapter+engine/disposable MySQL: same isolated recipe, `go test ./engine -run Zalo -count=1 -p 1 -timeout 40m -json` | Exit0, PASS8 top-level /13 total, zero FAIL/SKIP,25.699s. Includes all3 new message tests (6 including subtests), R024 coverage/checkpoint/refresh and bounded credential-write failure. Container/network teardown exit0. Full F08 sentinel gate NOT RUN on this focused selection. |
| Default and `--base origin/main --head HEAD` preflights |7/7 PASS at REVIEW intake; NOT runtime AI governance proof. |
| Gate unit tests `python -B -m unittest discover -s scripts/tests -p 'test_cvf_downstream_gate*.py'` |46/46 PASS39.893s. |
| Docs build `cmd /c npm --prefix docs run docs:build` |PASS21.54s; inherited env-highlighter warnings only. |
| Workspace doctor / knowledge ingest |25/25 PASS; generated knowledge output in OS temp. BOOTSTRAP_MIGRATION_PENDING nonblocking. |
| Race detector |NOT RUN: CGO disabled/no C compiler; no race-safety claim. |

Reviewer repeat detector: alternate valid one-row A/B pages for at most six requests, then return a fixed synthetic ceiling error. Require `errors.Is(..., ErrZaloMessageCoverageIncomplete)`, displayed repeat class and exactly three requests. Baseline passes; disabling the repeat guard reaches the ceiling and fails the explicit assertion. This separates production guard behavior from fixture array indexing. R1 should apply this principle to the committed detector without weakening original coverage.

## Limits and next governed move

Claude REPAIR_WORKER / COMMIT_STEWARD may fix R032-R1-01..02 within unchanged R032 seed/path/risk/effect/ownership boundaries, acknowledge BUILD before edits, produce repair evidence, and return exact local repair SHA with REVIEW_PENDING to independent Codex re-review. No independent new root cause is asserted for a third repair round; existing escalation rule remains. Reviewer accepts no silent self-repair. R031/R030/R024 and predecessors remain REVIEW_PASS / FREEZE_OPEN.

Live Zalo offset stability, actual empty-terminal behavior, retention/visibility/permissions, string links, global F02, provider governance, hosted CI and FREEZE remain OPEN. Worker max_connections override and earlier M7/M9 build failures remain explicit; reviewer local results do not erase them. No push, merge, deployment, parent edit or FREEZE.

## Final captures and artifact integrity

Interrupted full-run partial terminal inventory:91 top-level/199 total test PASS, zero reported FAIL, one optional pricing test SKIP; handlers package had not reached terminal pass.124622 captured JSON events. This intentionally interrupted receipt has no suite completion or R019 credit. Focused Zalo receipt is complete, independent and separate. Both recipes use max_connections1000 only in their own disposable DB; tracked test tooling unchanged. Raw Windows PowerShell redirection produced UTF-16; receipts were decoded then losslessly serialized to UTF-8 JSON before counting/hashing. No raw outputs or transient DB connection details committed.

Secret-free SHA256 receipts (OS-temp evidence, not permanent raw log availability):

- `channels.jsonl`: `48abb22e8fdbbb9242500ff337dbaf50accc7074d21651ac1a0888c8af3a0b5e`.
- `focused-zalo-engine-utf8.jsonl`: `2e03f812ead11bea5a4d8703be2f53dc26920df11804fba70bf357b8a1826dc3`.
- `full-backend-utf8.jsonl`: `838a92d88c46241b0e4a0c51582c1b66f3958f402357bd9b7e5f1f8e0140e2e5`.
- `detector-summary.json`: `6d5c945e5f0bbc465f2b26b05156eb7db202bf5e7ee3eec5ea424e551f4b946b`.
- `worker-cycle-fixture-check.jsonl`: `c70d77dd52765538c039fade3845c09773144b3d9712430191f0f06bedfcdc74`.

Isolated adapter original/restored byte SHA256: `213a3f7b3a962b8fed3fb643f4a2212b2bfb363f27323089ec493811d880dc8e`. Shared project source compared to exact BUILD throughout; no source/test/seed/tooling mutation. Final disposition synchronization/checks recorded in active handoff after execution. This commit records open findings and a repair route, not tranche closure.

Final return checks: default/PR-range preflights7/7, catalog regeneration/check, gate unit tests46/46 PASS50.999s, local links/source-preservation/diff PASS. The first docs build containing this new review failed13.90s on its workspace handoff hyperlink (the file exists locally but is outside published pages); reviewer corrected its own artifact to a plain workspace path without editing configuration. Rerun docs build PASS28.09s. Eleven documentation/continuity files only; source/tests/seed/tooling unchanged and named temporary containers/networks absent.
