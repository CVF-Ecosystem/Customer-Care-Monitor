# R046 R2 exact-repair independent re-review

Date: 2026-10-04 (Asia/Saigon). Codex independent REVIEWER, separate from Claude REPAIR_WORKER / repair BUILD COMMIT_STEWARD. Exact repair `91da0e88118b76a68031f432da50521fe6a341b7`, preceding worker acknowledgment `4130c75`, handback `71edf3a`. Immutable seed `c3ff83c7b8d2f9b23ef4113ae3bf7f8b9b32c267`; R2 ceiling and original scope/effect/commit ownership unchanged. Production SHA256 `4b7477aa12fd6c232e85818960f6ec0f2947548cd6ac24e714568ebff5a90f0b`; test SHA256 `06ec2c8d15d6199a679f9daaf0835db1baa7a8a63a35b79df5e9465d90f7c449`, both match submitted receipt and committed blobs. No reviewer source/maintained-test/seed repair. No production defect established.

## Finding evaluation

R046-R2-01 SETTLED: non-nil Context triggers GORM v1.31.1 Statement clone. The maintained helper stores the original pool, injects into the clone, and restores pool/pointer during cleanup. Maintained baseline and the previously failing independent pool probe now pass. Removing Context cloning in the exact-repair archive causes the maintained child test to fail the original-Statement mutation assertion; parent fails because its child failed, not because post-cleanup retention was observed. New explicit cleanup still restores the pool in this mutant. Byte-restored maintained control passes. This distinguishes pre-install isolation from post-cleanup restoration and avoids overclaiming two distinct assertions from two FAIL events.

R046-R2-02 evaluation: added ConnPoolBeginner wrapper returns a real sql.Tx wrapped by flBoundaryTx, which intercepts Commit/Rollback. Synthetic commit faults return before underlying commit; GORM's deferred rollback resolves the real transaction before retry. Rollback hook delegates to real rollback before returning the injected error. Atomic formatting/attempt counters, maintained BEGIN/COMMIT exhausted and transient cases and rollback/missing-sentinel case pass. The strengthened exhausted BEGIN case observes attempts, bounded sentinel/formatting, checkpoint and fallback. The rollback error may be ignored by GORM; no invented public identity contract. Additional reviewer-only COMMIT/rollback checkpoint and GORM-sink observations are recorded separately below; no claim of real driver parity or concurrent/race proof.

R046-R2-03 OPEN: R2 mutation replacement/hash reconciliation is improved: all three one-match replacement hashes now calculate to the stated full hashes, including M02 `a7306bb0ec6a0b551fa0407ccde94a10b30f59f99bc9df185a4b6a93a87375a6`. Named failure text and source lines also match the current test. Execution attribution remains incomplete and contradictory:

- Baseline receipt counts claim20 top-level/26 PASS but its own assertion list contains22 top-level/28 PASS; committed source has22 TestFL functions and independent exact-repair execution reports22/28, zero FAIL/SKIP.
- M01 command selects both UnknownErrorIsContained and NormalizationDetector, but records only one failure. Both test bodies require errFinalizeWrite; the exact normalization-bypass replay fails both, plus the newly covered COMMIT case in the broader independent selection. A compiling named kill is observed; the worker's complete command/count receipt is not certified.
- Combined regression receipt reports34 top-level/64 PASS. Independent selection/actual counts are recorded below and kept separate; no retrospective count correction is attributed to worker execution without matching raw logs.
- No per-run raw log/digest or declared scratch path, exact per-run test/overlay snapshot digest, or required publication/unit/catalog/doctor/docs-build/task-cleanup receipts are supplied in the R2 packet. The declared test runner creates a default Docker network (no --internal), has no --pull=never and removes MySQL without -v. Its invocation alone does not establish the claimed bounded internal/cached-only execution and anonymous-volume teardown. No actual worker external call or leaked volume is asserted from script inspection; historical resource provenance remains unverified.

Targeted project and task-temp discovery found prior reviewer campaigns and the submitted worker packets, not a declared matching worker raw campaign. This is bounded discovery, not a claim that no logs exist anywhere. Preserve R1/R2 worker packets as submitted; recover matching original logs/snapshot if available or execute a new properly labeled worker campaign after the cost decision. The reviewer's new source-sensitivity/restoration campaign proves current synthetic application observations and cannot certify historical worker execution. [Receipt audit](probes/r046_r2_receipt_audit.json) records the exact internal count contradiction and all calculated hashes.

## Repair cost checkpoint

CHANGES_REQUIRED remains for R046-R2-03. R2-01 is settled; final R2-02 observations follow below. The evidence-attribution finding repeats from the original review and R1. No independent new product root cause is established. Before a third repair round, record **REVIEW_COST_ESCALATION_REQUIRED** per the R046 order and AGENTS repair latency rule. This review does not dispatch repair round3 or grant BUILD/FREEZE. ORCHESTRATOR must record the cost/continuation disposition before another worker round; no repeated same-scope approval is inferred from this record.

## Bootstrap, limits and preservation

Rehydrated canonical manifest/policy/bootstrap fallback/state/memory/handoff/status/index; BOOTSTRAP_MIGRATION_PENDING nonblocking. INTAKE detected stale ownerRouting and catalog current pointers from R1; SESSION_SYNC_STEWARD reconciled them to the committed R2 REVIEW_PENDING handback before execution. Synchronized preflight7/7 PASS. Public core read-only `8a4119e11db00e774ed8e7cf7d9a8caa309e81d1` matches origin/main; historical manifest pin `26c686cc99b8be965d2760f27fe875b03376c643` retained. Doctor PASS WITH NOTE25 passed/1 warning, separate pin/profile migration; task-temp knowledge ingest only, no POST. Mandatory unit46 PASS14.708s.

Independent runs use cached images/modules, GOPROXY off, GOTOOLCHAIN local, readonly module inputs, task-owned internal synthetic MySQL and no host ports. No real config/credentials/provider/channel/external network/persistent customer DB, product/parent/tooling/compiler changes, push/merge/deployment/FREEZE or hosted/live/CVF AI governance claim. Synthetic provider dependencies are application fixtures, not governance proof. Full engine/backend tests and race NOT RUN; affected tests plus cached build/vet are separately scoped. R044/R045/prior dispositions and parked Facebook/Zalo OA preserved. Git identity does not independently prove individual authorship; committed role acknowledgment precedes repair. Historical failed reviews/mutant survivor/worker receipt limits remain.

## Completed independent evidence

[Machine campaign/publication receipt](probes/r046_r2_independent_summary.json) includes exact source/test/overlay hashes, mutation replacement/match/hash/restoration, raw-log digests, actual names/counts/assertions, archive equality and resource teardown. Durations include Docker/cache/Go orchestration. No required SKIP.

| Campaign / run | Exit | PASS / FAIL / SKIP |
| --- | --- | --- |
| R2 / baseline | 0 | 28 / 0 / 0 |
| R2 / worker-pool-isolation-probe | 0 | 2 / 0 / 0 |
| R2 / T01-statement-isolation-bypass | 1 | 0 / 2 / 0 |
| R2 / T01-statement-isolation-bypass-restored | 0 | 2 / 0 / 0 |
| R2 / M01-normalization-bypass | 1 | 1 / 3 / 0 |
| R2 / M01-normalization-bypass-restored | 0 | 4 / 0 / 0 |
| R2 / M02-raw-fallback-log | 1 | 0 / 1 / 0 |
| R2 / M02-raw-fallback-log-restored | 0 | 1 / 0 / 0 |
| R2 / M03-gorm-session-bypass | 1 | 0 / 1 / 0 |
| R2 / M03-gorm-session-bypass-restored | 0 | 1 / 0 / 0 |
| R2 / ownership-terminal-regression | 0 | 59 / 0 / 0 |
| R2 / build | 0 | 0 / 0 / 0 |
| R2 / vet | 0 | 0 / 0 / 0 |
| R2-boundary-supplement / boundary-sink-checkpoint-baseline | 0 | 4 / 0 / 0 |
| R2-boundary-supplement / M03-boundary-sink-checkpoint | 1 | 0 / 4 / 0 |
| R2-boundary-supplement / boundary-sink-checkpoint-restored | 0 | 4 / 0 / 0 |

R046-R2-02 SETTLED for this local synthetic contract: independent reviewer-only COMMIT exhausted/transient and rollback/missing cases verify exact retry/rollback counters, raw formatting0, bounded sentinels/application logs, empty capturing GORM Info sink, non-advancing checkpoint and fallback terminal error on exhausted COMMIT, and genuine committed success/checkpoint on transient recovery. Enabling scoped GORM Info fails each subtest at its named SQL/detail sink assertion, then byte-restored controls pass. Probe pointer `docs/reviews/probes/r046_r2_boundary_sink_probe_test.go`; mount only into exact-repair archive. Maintained tests and independent observations remain distinct; no broad real-driver/race claim.

Finalizer22 top-level/28 total and affected terminal/ownership18 top-level/59 total pass. Selections are disjoint; union40 top-level/87 events, derived from two independent runs, not a claimed single combined invocation. This differs from worker34/64 for the stated combined regex; historical worker counts remain unverified. Three production mutants and separate maintained isolation sensitivity mutant fail relevant assertions, with byte-restored controls passing. Cached full backend build/vet exit0. All203 exported files in EACH archive restored byte-equal; reviewer overlays removed, no extra files. Task DB/internal network/anonymous volume removal exit0; both named resources independently absent. Persistent application stack untouched; raw logs/cache retained only locally.

Disposition **CHANGES_REQUIRED / REVIEW / FREEZE_OPEN**, solely R046-R2-03; R2-01..02 settled without reviewer production/test repair. **REVIEW_COST_ESCALATION_REQUIRED**, no round3 BUILD dispatch. Next ORCHESTRATOR assesses and records cost/continuation before further worker evidence work. The R046 order carries the concrete remaining evidence requirements under unchanged authority; source/test changes are not requested. User handback authorizes this independent review, not a FREEZE or new external effect. Final publication checks after this last Markdown edit are recorded in the JSON summary before local metadata commit; no hosted CI or GitHub push claim.
