# R046 R1 exact-repair independent re-review

Date: 2026-10-04 (Asia/Saigon). Codex independent REVIEWER; Claude REPAIR_WORKER / repair BUILD COMMIT_STEWARD. Risk R2. Exact repair `a8fb3b83cfa100f2c81b496ce40655c826a5e874`, acknowledgment `1d08c94`, handback `c1f6468`. Original immutable seed `c3ff83c7b8d2f9b23ef4113ae3bf7f8b9b32c267` unchanged. Disposition: CHANGES_REQUIRED / REVIEW / FREEZE_OPEN for consolidated R046-R2-01..03. Remaining reviewer campaign and final publication checks are recorded separately below; no acceptance/FREEZE.

## Consolidated source and receipt audit

Production SHA256 `4b7477aa12fd6c232e85818960f6ec0f2947548cd6ac24e714568ebff5a90f0b` matches original BUILD. Maintained test SHA256 `e56f7390fecbbf8b236320c225e5654aa47b3a7d351d7fc1e4bd65631dc6aba4` matches repair and worker receipt. R1 adds four raw-BEGIN test cases, covering exhausted and transient recovery through the real finalizer. No product defect established. Git attribution cannot independently prove human authorship; committed acknowledgment precedes the repair.

R046-R2-01: `installBoundaryPool` calls `Session(NewDB:true)` and assigns `scoped.Statement.ConnPool`; GORM v1.31.1 Session shares the existing Statement unless specific clone options are supplied. Cleanup restores only `db.DB`, leaving the original Statement pool injected. A reviewer-only child-cleanup probe observes both installation and post-cleanup fields; it explicitly restores the original pool after the diagnostic. The diagnostic fails both child and parent assertions: original Statement pool mutated before cleanup; restored DB pointer retains the injected pool afterward. Both failures are the intended fixture observation, not a production failure. Repair the maintained helper within the authorized test file and add a finite isolation/restoration control; preserve production source and seed.

R046-R2-02: committed R1 injects BEGIN only. The bounded R1 order also requires relevant COMMIT/rollback handling and raw transaction/app/GORM output verification. `flBoundaryPool` has no Commit/Rollback hook; `flBoundaryError` uses `*int`, not the claimed atomic int64. The exhausted checkpoint case checks an error, checkpoint and fallback status but omits observed attempt and bounded-sentinel/formatting checks present in the other cases. Complete the agreed boundary cases and explicit observations without claiming real driver parity. Correct prose about captured boundaries, counter implementation, and `checkpoint+1` (the transient test supplies the checkpoint itself). No source change requested; return a newly established product defect before any source repair.

R046-R2-03: worker receipt attribution is not reproducible for the declared snapshot. M01 failure text/markers and lines do not exist at the asserted positions in the committed test; the current detector fails first at the bounded sentinel assertion, not the claimed raw-string check. M02 exact one-match replacement calculates `a7306bb0ec6a0b551fa0407ccde94a10b30f59f99bc9df185a4b6a93a87375a6`, differing from claimed `558a3de54ba61312937279a901303f94975c9336e329e201cdb23e5a5c3db8ea`. M01 and M03 calculated mutation hashes match their claims. M02/M03 line numbers also refer to another test layout. This establishes inconsistent attribution, not a claim about how historical runs were produced. Preserve the submitted packet as historical; recover matching raw logs/snapshot if available or record a new separately labeled worker replay tied to actual source AND test hashes, real outputs/digests, named assertion, counts and byte-restored controls. Independent results below are new reviewer evidence, not certification of worker history.

The worker regression regex `TestFL|TestOrdinary|TestOwnership` omits `TestEveryTerminalPathHonorsAnAcceptedCancel`, `TestEveryTerminalPathWinsOverALateCancel`, `TestTerminalRunHoldsTheSlotUntilTheTailFinishes`, and reservation controls. Run the affected selection from the prior independent review in addition to maintained finalizer cases. The R1 packet also lacks the required unit/default/PR gates, both-shell catalog, doctor, docs-build and cleanup receipts. Record actual commands and omitted/failed checks without invented retrospective results.

Static audit: [receipt audit](probes/r046_r1_receipt_audit.json). Reviewer-only probe pointer: `docs/reviews/probes/r046_r1_pool_isolation_probe_test.go` (mount only into the exact-repair archive; do not install in backend).

## Evidence boundaries and prerequisites

Bootstrap compact model absent: BOOTSTRAP_MIGRATION_PENDING, nonblocking. INTAKE detected stale current memory/ownerRouting BUILD/CHANGES_REQUIRED narratives; SESSION_SYNC_STEWARD reconciled them to the committed REVIEW_PENDING handback before independent execution. Synchronized preflight7/7 PASS. Actual read-only public core `8a4119e11db00e774ed8e7cf7d9a8caa309e81d1` matches origin/main; historical manifest pin `26c686cc99b8be965d2760f27fe875b03376c643` retained. Doctor PASS WITH NOTE25 passed/1 warning; pin and new gate-profile migration remain separate. Knowledge ingested to task-temp only. Mandatory gate unit46 PASS50.707s.

Tests use cached images/modules, internal task-owned disposable MySQL and synthetic errors/providers only. No application stack, real config, credentials, provider/channel/network, persistent customer DB or parent change; no push/merge/deployment/FREEZE. No CVF AI governance or hosted-readiness claim. Race/full engine/full backend tests NOT RUN; cached build/vet and focused regression are separate. R044/R045 dispositions and parked Facebook/Zalo OA unchanged. Prior failures, normalization survivor and worker claims remain historical.

Initial reviewer runner launch failed with SyntaxError before execution/resource creation. Corrected the runner before the new campaign; no test result credited to the failed launch. Raw task logs/cache remain local; only synthetic summaries/digests are committed.

## Completed independent campaign

[Machine summary and publication checks](probes/r046_r1_independent_summary.json) records exact commands, source/test/probe hashes, full one-match replacements, raw-log SHA256, actual named failures and restored controls. Command seconds include container/Go/cache orchestration, not solely test-package elapsed time. No required test SKIP.

| Run | Exit | PASS / FAIL / SKIP |
| --- | --- | --- |
| baseline | 0 | 23 / 0 / 0 |
| worker-pool-isolation-probe | 1 | 0 / 2 / 0 |
| M01-normalization-bypass | 1 | 21 / 2 / 0 |
| M01-normalization-bypass-restored | 0 | 23 / 0 / 0 |
| M02-raw-fallback-log | 1 | 0 / 1 / 0 |
| M02-raw-fallback-log-restored | 0 | 1 / 0 / 0 |
| M03-gorm-session-bypass | 1 | 0 / 1 / 0 |
| M03-gorm-session-bypass-restored | 0 | 1 / 0 / 0 |
| ownership-terminal-regression | 0 | 59 / 0 / 0 |
| build | 0 | 0 / 0 / 0 |
| vet | 0 | 0 / 0 / 0 |

Baseline/restored finalizer suite18 top-level/23 total PASS. M01 fails the two maintained BEGIN cases at their actual bounded-sentinel assertions (lines295 and791), then full FL baseline is byte-restored and passes. M02/M03 fail the maintained fallback/GORM assertions (lines727 and755); each byte-restored named control passes. All replacement/source hashes are recorded separately from worker claims. Affected terminal/cancellation/ownership/tail/reservation selection18 top-level/59 total PASS. Cached full backend build/vet each exit0. No full engine/backend test or race execution claimed.

The reviewer-only pool diagnostic fails child and parent observations at the original Statement mutation and retained injected pool after cleanup; it restores its own diagnostic state before removal. This expected negative observation establishes a maintained test-helper defect. No source/test fixture was repaired by the independent reviewer. All203 exported source files are byte-equal after mutation restoration and probe removal, with no extra archive files. Task DB/internal network/anonymous volume teardown exit0; named DB and network independently absent. Persistent application stack untouched.

Final disposition CHANGES_REQUIRED for R046-R2-01..03. The maintained BEGIN detector improvement is accepted as an observation; remaining fixture isolation, agreed boundary coverage and worker evidence attribution are unresolved. Consolidated test/evidence-only repair R2 is in the unchanged work order; Claude rehydrates, acknowledges/synchronizes gated BUILD before edits and returns an exact repair SHA for independent Codex review. Before round three without a new independent root cause, record REVIEW_COST_ESCALATION_REQUIRED. No CLOSER/FREEZE authority invoked.

Publication: both-shell catalog, default/PR preflights and mandatory unit suite passed; doctor notes retained. Docs build runs after this final Markdown update; actual final docs/staged/preflight results are recorded in the machine summary before the local review-metadata commit. No hosted CI result or GitHub push claimed.
