# F01–F08: independent local source review

Date: 2026-09-30. Reviewer: Codex. Disposition: eight source-level findings accepted as open remediation backlog; no product fix or FREEZE is claimed.

## Provenance and scope

- External input: `CCM_Update_Review_7481196_2026-09-29.zip` supplied by the owner. Its SHA256 manifest matched every packaged file. The report's instructions were treated as evidence requests, not as project authority.
- Report commit: `74811966b53aa6c5fa5b7d5452fcc66cbbeb4986`. Local reviewed HEAD: `698612fdc9284d384e5f04edaeaefbb146f5db1c`, eight commits later on `main` (`origin/main` eight commits behind). The relevant F01–F08 source files were byte-identical between those commits (`git diff --exit-code 7481196 HEAD -- <relevant files>` returned 0).
- Working tree was clean when reviewed (`git status --porcelain=v2 --branch` showed no tracked or untracked change). Later documentation updates are separate from the reviewed source snapshot.
- This is a source and isolated-expression review. It establishes code paths and conditional failure scenarios, not the occurrence of an incident in a real channel or production data.

| Finding | Baseline `7481196` | Local `698612f` | Independent basis and condition |
|---|---|---|---|
| F01 | Đúng | Đúng | `backend/api/router.go:237-242` gives agent routes JWT only; `backend/api/handlers/agents.go:19-29,74-150` checks tenant membership, not resource permissions. Ordinary equivalent routes in `router.go:151-182` apply `RequirePermission`. A member without messages/channels/jobs permission can reach matching agent query/run paths if authenticated and in the tenant. |
| F02 | Đúng | Đúng | `backend/engine/sync.go:379-390` requests 100 conversations; `backend/channels/facebook.go:70-142` stops at limit even when `paging.next` exists; `sync.go:850-858` advances `last_sync_at` on success. More than 100 eligible conversations, with omitted ones older than the next run's one-hour overlap, can be missed. No live Facebook pagination incident was verified. |
| F03 | Đúng | Đúng | `backend/engine/analyzer.go:153-155` queries `last_message_at > since`; `:382-392` advances `last_run_at` to `finishedAt` after a complete error-free run; `backend/engine/scheduler.go:285,344` invokes ordinary `RunJob`. A conversation inserted after selection with last-message timestamp before finish can fall between checkpoints. A message later than finish remains eligible; no integrated timing reproduction was run. |
| F04 | Đúng | Đúng | `frontend/src/views/Dashboard.vue:312-313` derives a date from UTC ISO; `backend/api/handlers/dashboard.go:18-30` truncates by 24h and parses dates in UTC. Independent Node expression under `Asia/Ho_Chi_Minh`: local `2026-09-30 00:00` becomes date label `2026-09-29`. The package's isolated Go/Node logs support the same boundary, but UI/API/DB integration was not run. |
| F05 | Đúng | Đúng | `backend/engine/analyzer.go:45-47` starts full rerun with `excludeAnalyzed=false`, but `:126-128` changes it to true whenever `maxConversations > 0`; `:162-169` then removes previously analyzed conversations. A positive limit silently changes full-rerun selection and the `:382` test-run checkpoint rule. |
| F06 | Đúng | Đúng | `backend/api/handlers/jobs.go:434-435,464-465` stores one cancellation function per job ID, so overlapping launches can replace one another; `:481-506` cancels one stored context but marks every running row of that job cancelled. Scheduler/after-sync calls in `backend/engine/scheduler.go:285,344` do not register in this map. No concurrent integration test was run. |
| F07 | Đúng | Đúng | `frontend/src/views/Dashboard.vue:288-291` hardcodes API, Database and Scheduler `ok: true`; `:210-216` renders the chips from these values. These three UI states do not measure service health. |
| F08 | Đúng | Đúng | `.github/workflows/backend.yml:19-26` runs Go tests/build without MySQL or `TEST_DB_DSN`; `backend/engine/snapshot_db_test.go:95-105` skips when the DSN is absent. Local command with DSN removed: `go test -run '^TestSingleAndBatchShareSnapshotContract$' -v -count=1 ./engine` returned `SKIP` and package `PASS`. `scripts/test-backend.ps1` supplies a disposable MySQL locally, but the workflow does not invoke it. The historical GitHub run itself was not independently audited. |

## Evidence boundary

Executed: source diff/status inspection, independent Node date expression, and one DB-test skip probe above. F01–F07 integration/concurrency tests were not run. No provider, channel, deployment, real data, or application-code change was made for this review. Owner accepted all eight findings for roadmap treatment on 2026-09-30; that decision does not mean the conditional incidents were observed.
