# R135 queue/reject fixture worker return

Status: COMPLETE_PENDING_REVIEW

Risk: R2

Worker: Sol Medium IMPLEMENTATION_WORKER

Reviewer: Root independent source/browser REVIEWER

Self-declared worker-return artifact: yes

The new opt-in `backend/api/handlers/intervention_queue_browser_test.go` admits only `R135_QUEUE_BROWSER=1`, matching `CCMAI-RUNTIME-135`, `ccma_r135_*` database and preexisting nonce owner table, loopback DSN, and the exact external `../.ccmai_local_evidence/r135_queue_reject_20261020/fixture` directory. It requires a fresh one-file identity directory, nil provider/executor, the inherited built frontend, and an unset compatibility DSN before connecting. It then uses `m3FixtureStart` and binds the actual seeded target proposal pointer before baselining. The application login password in `ready.json` is the helper's `R101SyntheticOnly!123`; the separate fresh MySQL password is part of Root-owned external provisioning.

The fixture seeds 24 OPEN cases including the target, one ASSIGNED, one WAITING_ON_DATA, terminal REJECTED and RESOLVED distractors, and one foreign-tenant case. The 24 OPEN cases share a timestamp, so both seven-row native pages and the 20-row browser page split an equal-time group by ID. It records exact ordered IDs and totals in `expected_queue.json` and `ready.json`. Real authenticated-router GET checks page through default, OPEN, WAITING_ON_DATA, REJECTED and RESOLVED views at limit 7, check cursor termination, total, order, no duplicate/omission, and deny foreign access. No native positive action runs. The browser server permits only finite read routes, at most four auth POSTs and one target action POST. Its external route journal captures each actual queue GET query, state, input/output cursor, response IDs, total, order and `has_more` for Root's DOM comparison.

The held server uses an exact nonce-owned stop, 20-minute deadline, Serve join and fail-safe terminal receipt. Terminal assertions require one REJECT action with nonempty reason, version 2 REJECTED/HUMAN_REJECT, exact before/after event projections, the active proposal lineage, unchanged target source and all snapshot/proposal/message rows, unchanged non-target cases, and unchanged provider/run/grant/budget/receipt/execution effect counters. Activity-log counts are reported separately because user login/logout audit activity is legitimate and is not an AI counter. The terminal records action/auth counts, observed status, effect deltas and actual Go test failure state. Actual browser, process exit and protected-resource acceptance remain Root-owned.

The single authorized cached offline compile from `backend` used `GOPROXY=off`, `GOSUMDB=off`, `GOTOOLCHAIN=local` and `go test -c -mod=readonly -o ../../.ccmai_local_evidence/r135_queue_reject_20261020/worker_compile/handlers_r135.test.exe ./api/handlers`. It exited 0. The external empty compile log SHA-256 is `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`; compiled binary SHA-256 is `d77dc76c905a9fb4ec13bb9b60fb812f2572ab6b2637acf396b5b693758ac7e3`. That compile covered pre-review source SHA-256 `dddad9d3646db09d8f4ae2f57d916a19359df50a86971631a7d5a5d7ad911b93`. Root's consolidated source review then identified two same-scope evidence gaps: the native GET count omitted two foreign probes and only the target snapshot/proposal/message rows were compared. I corrected both without a second compile, then ran `gofmt` on the sole new fixture as a formatting-only step. Final source SHA-256 is `1545939b3a0dc4e421522a8297da8f204599716f9c25614ba0cdac9cddda9e9b`; Root's reserved compile must validate this final version before runtime. `git diff --check` exits 0.

Worker counters: Go compile 1/1 permitted, Go runtime 0, Node 0, MySQL fixture/start/stop 0, Docker 0, browser 0, auth/action POSTs 0, provider 0, network downloads 0, commits 0. No production source, shared metadata or dispatcher seed was edited. The role acknowledgment was appended to the active R135 handoff before source editing. R134 accepted evidence and all earlier budgets remain historical and unchanged. This return makes no AI/CVF governance, full M3, hosted CI, pilot or release claim.

## Return-Time Closeability Recheck

closeabilityDisposition: CLOSEABLE

outsideAuthorityBlockers: NONE

nextRepairRoute: REVIEWER_LOCAL_REPAIR

workerRedispatchAllowed: YES

Root's independent final-source compile, source admission, one bounded browser campaign and terminal/protected-resource review remain pending. No self-approval or FREEZE claim is made.
