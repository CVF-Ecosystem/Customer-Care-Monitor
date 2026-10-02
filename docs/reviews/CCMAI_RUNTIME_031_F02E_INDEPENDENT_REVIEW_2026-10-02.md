# CCMAI-RUNTIME-031 — F02-E independent review

Date: 2026-10-02. Reviewer: Codex, independent from Claude implementation. Exact BUILD `8ed6d0b39195ade513906571daaaf537ddeb90a9`; `81849ccc472d25dba7b7aa0b5a686a7878466813` records SHA only. Disposition: REVIEW_PASS / FREEZE_OPEN for local F02E-01..09; final record synchronized 2026-10-03.

Authority: [SPEC](../specs/RUNTIME_FACEBOOK_MESSAGE_COVERAGE_F02E_2026-10-02.md), [work order](../work_orders/CCMAI_RUNTIME_031.md), [worker evidence](RUNTIME_FACEBOOK_MESSAGE_COVERAGE_F02E_BUILD_2026-10-02.md), active handoff and immutable dispatcher seed. Codex authored/committed seed before BUILD at tranche baseCommit `61eda32442b6049e0ed37a00f6664c7fe880da94`; seed exists there and current JSON equals committed authority. Worker before-edit acknowledgment is recorded; a final commit alone cannot independently prove intra-worktree timing. No broad procedural-compliance claim.

## Source and acceptance matrix

Product changes are confined to Facebook messages: bounded message request helper, redirect policy on a client copy, rebuilt authenticated URLs, safe next/cycle validation, strict message rows, first-seen deduplication and stable chronology. Shared doRequest, HealthCheck, conversation validator/traversal, engine source, Graph version, credentials and Pancake/Zalo source remain unchanged. Message mapping is moved from the old loop with preserved semantics; existing Pancake chronological helper is reused without edits. Reviewer made no product/test repair.

| ID | Independent evidence and result |
|---|---|
| F02E-01 | Initial fields/limit/GET/escaped-ID inventory PASS; configured token used once, provider token removed. Next URL parts validated before reconstruction; encoded-equivalent paths accepted then rebuilt into the initial escaped endpoint. Unsafe host/path/protocol/query probes directly assert one initial request and no subsequent transmission. |
| F02E-02 | Explicit array/row/identity/time validation PASS; invalid old/repeat rows beside eligible progress trigger validation-class errors before dedupe/filter. Message sentinel distinct from conversation sentinel. |
| F02E-03 | Empty/short pages with next continue; valid no-next terminal succeeds. Malformed paging/next on empty/old pages fails. Platform-specific contract preserved. |
| F02E-04 | Exact inclusive-window IDs, first mapping, stable ascending times/ties and sender/media/sticker/RawData mapping PASS. No new redaction or live ordering claim. |
| F02E-05 | Direct/alternating/token-rotated/query-order/encoded cycle probes PASS. Distinct empty/duplicate cursors continue. Page500 terminal succeeds; nonterminal500 errors without request501. |
| F02E-06 | Redirect probes measure zero destination requests; status/body/read/connection/Graph/decode/size failures are non-success and sanitized, bodies closed; deterministic cancellation probes PASS. Shared helper is not certified by this message-only repair. |
| F02E-07 | Actual adapter/engine/disposable MySQL: exact eligible stored IDs/counts, recorded cursor positions, configured tokens/no other hosts, tenant/channel isolation, checkpoint advance, after-sync/completion once, idempotent replay PASS. |
| F02E-08 | Six late-failure integration cases PASS: missing data, malformed row, unsafe next, actual connection error, separately labelled HTTP500, redirect. Failed diagnostic rows discarded, successful peer stored, partial status, checkpoint held, no after-sync/completion; successful retry/replay stores once. Connection case is an accepted regression, not independently credited as an old-source status detector. |
| F02E-09 | Original adapter causes 14 behavioral failures. Eight applied reviewer mutations fail named behavioral tests; source restored byte-exact, full channels baseline PASS. Worker first M1 INCONCLUSIVE/hanging-probe incident retained separately. |

## Independently executed checks

- Full channels JSON run: 77 top-level tests, 88 PASS including subtests, 0 FAIL/SKIP, package 3.601 s. Restored post-detector full baseline repeats 77 top-level/88 total PASS, 0 FAIL/SKIP.
- `scripts/test-backend.ps1 -Packages ./engine -Run '^TestFacebookSync(StoresExactEligibleMessageIDsAcrossPages|MessageFailureIsPartialAndRetryCompletes)$' -VerboseTests`: actual adapter plus isolated MySQL, 2 top-level/8 including subtests PASS, 0 FAIL/SKIP; package 8.275 s. Disposable resources removed by script.
- `go -C backend build ./...`, `go -C backend vet ./...`: PASS. Mandatory gate tests 46/46 PASS (33.890 s).
- Core doctor25/25; current pin/public remote/origin-main/kit checked. Knowledge ingest complete; generated untracked index excluded. BOOTSTRAP_MIGRATION_PENDING nonblocking.

## Applied original-source and mutation evidence

Baseline bytes: 18861, SHA256 `8341b19347fda089e643a55246b8562d1cc816eba4906bb953c347d463c1aa23`. Reviewer harness asserts each mutation's one matched replacement, actual byte change and finally byte restoration. Tests use explicit `-timeout=30s` plus a 50 s process bound; no detector timeout occurred. Original comes from planning revision `c0f3db1`, adding declaration-only new error/budget/size symbols so current tests compile; no original behavior replacement. Its budget transport is the committed repaired finite fixture.

| Case | Observed behavioral failure; each exit1, no build error |
|---|---|
| Original adapter + symbol-only shim | 14 top-level tests: request inventory, blank ID, unsafe next, malformed page, invalid row, empty/short continuation, terminal forms, since filtering, mapping/order, cycles, distinct cursors, budget, redirects, HTTP/late errors. |
| M1 old row returns nil early | SinceFiltersOutputWithoutEarlyStop. |
| M2 empty page returns nil early | EmptyAndShortPagesWithNextContinue. |
| M3 force validation outcome true after validMessageNext | UnsafeNextLinksAreRejectedBeforeAnyRequest. Outbound URL reconstruction still exists; this detector specifically rejects accepting an invalid next, not a claim that the mutation sent a real external request. |
| M4 return nil after page budget | PageBudgetBoundary. |
| M5 skip seen ID before timestamp validation | InvalidRowsAreNeverFilteredAway. Reviewer one-insertion bypass differs from worker two-replacement harness. |
| M6 remove message redirect policy | RedirectsAreNeverFollowed. |
| M7 include link token in canonical identity | CyclesFailClosed. |
| M8 skip HTTP status guard | HTTPFailuresAndLateErrorsAreNeverSuccess. |

All failures are named assertions, not compile/harness errors. No mutation/shim remains; exact BUILD product source identity verified afterward. Worker campaign's first M1 Access-denied result remains INCONCLUSIVE; its rerun kill and other worker results are attributed. Independent successful reproduction does not erase earlier evidence. Worker initial runaway probe/test repair and old-engine connection subtest failure mechanism were inspected, not independently replayed as historical incidents.

## Timeout, partition and evidence limits

Worker's full-backend first run did not pass: engine hit Go's default10m timeout. Other packages reportedly completed and no preceding engine assertion failed. Worker then ran two disjoint engine subsets, 94+115=209 top-level tests, with both packages ok and zero skips. Reviewer independently enumerated current engine tests with go -C backend test ./engine -list ^Test: exactly209 top-level names, regex partitions94/115 disjoint and exhaustive; sorted-name inventory SHA256 cd234e4afa0d49c4458044e13f3b7e52dfe9592fb4ab165fdbc8ff95799591d5. This verifies partition membership, not worker executions. These are partitioned regression results, not a successful monolithic run. Host slowdown is a hypothesis; the cause of the first timeout is not established from this review.

Reviewer first complete unpartitioned `go test ./... -count=1 -p 1 -timeout 20m -json` used disposable MySQL/Docker, same images/readonly module cache/GOPROXY=off/CGO_ENABLED=0/UTC DSN. Exit0, 16 tested packages PASS (3 packages have no tests), 963 PASS including subtests /509 top-level, 0 FAIL, **3 test SKIP**: external pricing live-fetch and S3 probes, plus TestZaloCheckpointIsBoundedByFetchStart due to MySQL Error1040 Too many connections. Engine package completed623.107 s, 443 PASS including subtests /208 top-level and1 DB SKIP. This run is NOT credited as zero-DB-skip acceptance. JSON log180214662 bytes, SHA256 d5eb29515a75461620e10469ee201b194e9eab488c809123fdb13c9d1c694fd4, `%TEMP%/ccmai-r031-review-full-backend.jsonl`; no raw logs committed.

Separate unpartitioned engine rerun on a new isolated MySQL with `--max-connections=1000`, verified @@max_connections1000 before tests, same20m timeout and immutable source/tooling: exit0, **209 top-level /444 PASS including subtests,0 FAIL/SKIP**, package628.731 s. Command `go test ./engine -count=1 -p 1 -timeout 20m -json`; golang1.26-alpine/mysql8.0, readonly module cache/GOPROXY=off/CGO_ENABLED=0/UTC DSN. Log `%TEMP%/ccmai-r031-review-engine-monolithic-1000.jsonl`,101472270 bytes, SHA256 `ddc20fe5c6199d78000b49dcb2bf42f135cfb961c1dea13d59265f7e69e6ac38`. DB/network teardown exit0. Capacity override belongs only to this disposable fixture; no repository tooling/production setting changed. This independently closes engine zero-DB-skip regression coverage, while other packages retain the first full-run evidence; there is no claimed single full-backend invocation with zero DB skips. Error1040 establishes connection exhaustion at that skip, not a root cause for the worker's earlier timeout or proof of a specific pool leak. A later pass cannot retroactively turn the worker's10m invocation into PASS or explain its timeout.

Race NOT RUN: CGO_ENABLED=0 and no gcc executable found. Deterministic hooks establish only the tested ordering. No Messenger endpoint compatibility (docs unavailable), live permissions/retention/ordering/cursor stability, Zalo message completeness, global F02, real credentials, provider readiness, runtime CVF governance or hosted CI verification. Synthetic transport/DB acceptance remains local application semantics. No real provider/channel action, persistent database, parent edit, push/merge/deploy or FREEZE.

## Final verification and disposition

REVIEW_PASS / FREEZE_OPEN: all nine local acceptance requirements are supported; no blocking source/test defect found. Independent reviewer made no product/test repair. Source and immutable seed are unchanged. Final synchronized continuity, scoped/default/PR gates, catalog/docs/links/diff and resource inventory verification follow below; review commit contains documentation only.

Final verification (2026-10-03): full tranche explicit19-path preflight7/7 PASS, including synchronized REVIEW_PASS; catalog regeneration/check PASS, changed-set local links and git diff --check PASS; docs build PASS27.23 s. Mandatory gate unit tests46/46 PASS33.890 s earlier in this independent review. Default worktree and origin/main..HEAD preflight both FAIL6/7 solely on pre-existing untracked knowledge/_index.json and the two Python bytecode files; none staged, no full-tree/PR PASS. Docker container/network ccma-test-* inventory empty after teardown. Exact BUILD backend diff empty and seed JSON equals committed base authority. Review/session/local commit steward Codex acknowledged separately; worker BUILD steward remains Claude in immutable authority. This disposition is independent local source acceptance, not tranche closure/FREEZE.
