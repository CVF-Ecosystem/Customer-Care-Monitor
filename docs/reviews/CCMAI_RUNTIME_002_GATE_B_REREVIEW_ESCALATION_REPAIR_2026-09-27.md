# Independent re-review: CCMAI-RUNTIME-002 Gate B escalation repair

**Reviewer:** Codex (`REVIEWER`) · **Date:** 2026-09-27 · **Target:** local commit `103650a` · **Disposition:** `CHANGES_REQUIRED` (test/evidence completion only). Gate B remains `REVIEW_PENDING`.

## Accepted source behavior

- `ResetDemoData` now uses one `db.DB.Transaction` for both parent locks, all eleven deletes and the tenant-settings update. Each statement returns its `.Error`; the transaction result, including begin/commit failure, controls the HTTP response. A returned error rolls back and yields 500. This resolves the observed unchecked-delete defect R3-E1 in source.
- No other deletion path, schema or provider code changed in the repair commit. The nullable legacy result path remains intact.
- Independent run on disposable MySQL 8 `CCMA`: `go test ./api/handlers -run '^TestResetDemoData' -count=1` PASS; `go test ./... -count=1` PASS across all 13 tested packages. The test container and its Go cache volumes were removed. The persistent `ccma` Compose database was not used. Workspace doctor PASS 25/25. A direct Windows Go test was blocked by Application Control before tests ran; it is not counted as test evidence.

## Blocking acceptance gap: R3-E1-T1

The owner-authorized work order requires the permanent forced-delete regression to prove that **result, snapshot, run, job, conversation, channel and demo flag** remain unchanged. `TestResetDemoDataFailureRollsBackEverything` seeds a `job_result` without `analysis_snapshot_id` and creates no `analysis_snapshot`. It then checks only channel, conversation, job, run, result and demo flag. The success test likewise has no snapshot. Thus the tests exercise the legacy result path but never the snapshot-bound evidence path; the explicit snapshot assertion remains unfulfilled. The repair evidence also says the regression checks every required row, which overstates what the test checks.

This is a coverage/evidence gap, not a demonstrated production-code failure. Source transaction semantics strongly support snapshot rollback, but Gate B acceptance requires an executable permanent assertion.

## Required disposition

The owner has authorized Codex as orchestrator/reviewer to package the remaining work for Claude. Complete the same narrow R3-E1 repair by adding a snapshot-bound fixture and checking the snapshot on both forced-failure and success paths. Keep application source unchanged unless that expanded regression reveals a defect; report any such defect before widening scope. Update the repair evidence, rerun the specified MySQL checks, then return one local commit without push for independent Codex re-review. Exact contract is appended to `docs/work_orders/CCMAI_RUNTIME_002.md`.

No real provider API was invoked; this review makes no claim that CVF governs AI/agent runtime behavior. No channel sync, customer data, deployment, S2/S3/S5 or FREEZE is authorized. Existing unrelated `.gitignore`, `docs/references/` and `knowledge/_index.json` worktree entries were excluded from the review and commit scope.
