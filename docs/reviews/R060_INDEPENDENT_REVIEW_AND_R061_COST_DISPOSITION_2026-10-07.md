# R060 independent review and R061 bounded cost disposition

Date: 2026-10-07. Independent reviewer Codex `/root`; child `/root/r059_worker` owns source/repair. Result CHANGES_REQUIRED; finding R060-R2-01. Exact source `a7edba22589627aee2dd97ffe1921c340c6fbc93`, handback `d15739a9ec1313c7e7c2cc15ab734d6985bb79fb`, before-runtime reviewer acknowledgment `50fe70a`.

## Actual independent result

One campaign/one Go invocation,177.991s including isolation/compile/cleanup. All120 selected top-level tests completed:119 PASS/1 FAIL,308 total completed events305 PASS/3 FAIL/0 SKIP. Failing top-level `TestEXInitialProgressAndTerminalWriteFaults`, two failing subtests terminal/early_terminal: stored status required running. No build failure, timeout or panic. M01/M02/restored NOT_RUN; no automatic retry. Root directly verified raw stdout/stderr sizes and hashes. Exact210-file export restored, fresh second archive hash identical; all named container/network/cache/anonymous volumes absent after successful cleanup inventories. Application observation only, no provider or governance claim. Current46 repository gate units freshly PASS20.397s.

## Consolidated independent root cause

Existing `finalizeOrdinaryRun` retries the transactional terminal write, then best-effort marks status error, finished_at and bounded error_message without updating Summary or checkpoint. The new fixture callback faults only updates containing summary, so fallback succeeds. New EX terminal/early_terminal assertions incorrectly require running despite that existing fallback. This is a distinct root cause from R1 nonUUID binding. Product behavior must remain unchanged; stored fallback error alone does not certify a terminal execution receipt. Actual initial/progress prefixes must remain distinguished from full returned observation.

R060-R2-01 repair must affect only NEW EX DB fault tests: check reload errors; assert the observed fallback error plus unchanged durable prefix, no terminal execution-complete certification/checkpoint and full returned observation as appropriate. Include a separate fault case that also blocks fallback if asserting persisted running; do not loosen to arbitrary status or remove fault checks. Audit finalizer retries/fallback/early branch and callback teardown together. UUID/binding tests and all old source/tests/seeds/evidence remain intact. Root source/test edits0; child retains independent R2 repair ownership.

## Successor and unchanged total Go ceiling

Original R060 worker1campaign1Go and reviewer1campaign1Go consume both originally allowed campaigns; each stopped at positive failure. Preserve both failed packets. Root autonomously chooses separately seeded R061 before any further repair/runtime, because another campaign exceeds the old campaign cap. Same objective, NEW EX DB path only, same medium child/root independent reviewer and R2/effects; product edits prohibited. This explicitly records the campaign boundary instead of silently retrying under R060.

R061 worker repairs/commits with new workerGo0. R061 root gets exactly1 new campaign/max4Go: grouped EX and related SP/finalizer positives, actual M01, actual M02, restored EX controls. Cross-lineage Go ceiling8, already used2, planned6; max3 campaigns including the two retained failed campaigns. No reset, no additional provider/network/credential/publication authority. The third-repair escalation rule remains: this is repair round2 with an independently identified new root cause. No automatic retry if successor fails.

R060 stays CHANGES_REQUIRED, FREEZE_OPEN; independently passed104 older regression top-level tests are inherited local evidence on byte-identical product source,15 of16 new EX passed. They do not substitute for the failing fault contract or unrun mutations. R061 must settle those before acceptance. Full S2/global F02/live/governance/hosted remain open; Facebook/Zalo OA parked. No push/merge/deployment.

Raw evidence: `docs/reviews/probes/r060_independent_summary.json`, `r060_independent_positive.jsonl`, `r060_independent_positive_stderr.log`, `r060_independent_manifest.json`; source/plan/protected proof and runner already committed. Worker failure `56f5f85` and root disposition `9cfcf96` preserved. This report and new immutable seed are committed before R061 activation.
