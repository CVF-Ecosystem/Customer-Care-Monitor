# R102 fixture JSON preservation comparison — repair round 2

Status: CHANGES_REQUIRED
Risk: R2
Worker: /root/r102_worker
Reviewer: Codex /root
Date: 2026-10-15

Root's late source read-ahead found that TestR102NativeLegacyEventSurvivesMigration compares the stored JSON After column to the pre-insert minified string. MySQL JSON storage may normalize serialization. The correct preservation baseline is the row read back before the schema simulation, not the caller's pre-storage string. This is a fixture correctness defect and Root source-review miss; no actual native failure or application data-loss defect is asserted.

One offline handlers compile PASS at source223c052, workerGo1/aggregateGo1 of8. Root stopped the worker before engine compile and before any MySQL/native/provider effect. Preserve that successful compile receipt and spent slot; the old binary is not evidence for repaired tests. Consolidated review of remaining direct JSON comparisons found original AnalysisSnapshot.Manifest uses mediumtext, while the legacy human-event comparison needs repair.

Second and final same-scope repair: in the authorized handler test file, read the inserted legacy row from the owned DB before dropping the two nullable columns, then compare preserved old fields against that stored baseline after migration twice. Check read errors. Refresh source manifest/return with actual counters1 and gofmt; no source mutation while any compile runs, no runtime, compile retry, seed/budget change or commit by worker. Root will re-pin/re-review hashes and commit before re-admitting handlers and engine compiles. Aggregate budget remains8, no slot reset. A third round without independent new root cause must stop with REVIEW_COST_ESCALATION_REQUIRED.
