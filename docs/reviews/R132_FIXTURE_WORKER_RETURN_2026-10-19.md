# R132 fixture worker return — 2026-10-19

Status: COMPLETE_PENDING_REVIEW

Self-declared worker-return artifact: yes

Risk: R2

Worker: Sol Medium IMPLEMENTATION_WORKER (fixture only)

Reviewer: Root independent source/browser REVIEWER

Source: `backend/api/handlers/intervention_followup_browser_test.go` (current SHA-256 `D1B8D14D5826C9D8032BEEA131955F7D59AAEABFD9D74B5F944462C9EB46AA22`). No production backend source, schema, provider, installed app, dispatcher seed, or frontend file was edited by this worker. Pre-BUILD acknowledgment was appended to the active R132 handoff. No commit was made.

## What is implemented

The default test skips unless `R132_WAITDATA_BROWSER=1`. Admission validates a fresh external, absolute, non-symlink evidence directory, a tranche/host/port/database/nonce identity file, a matching loopback MySQL DSN, and exactly one preexisting nonce-table row **before** setting the compatibility DSN or calling `m3FixtureStart` (which connects and migrates). The existing helper supplies the real router and a synthetic WAIT_DATA case. The fixture adds one local customer message in that case's conversation before its baseline, then checks native reader capture 403 and no-jobs request 403 with unchanged case and effect counts.

It serves the actual built frontend and real API on an ephemeral `127.0.0.1` listener. Its route guard allows read paths and at most four auth, two capture, and one request browser POSTs. All other POSTs, including execute/grant routes, are denied. The journal contains only method/path/status. A one-shot Gorm query callback requires a private context marker present only on the first actual browser capture POST's message-source read; concurrent GETs cannot consume the fault. That POST must yield the real handler's sanitized 503 with no write. The second explicit browser capture must return 201. The callback is removed at test cleanup. No provider or executor is installed.

After a nonce-matched stop or finite 20-minute browser deadline (Root's Go timeout must be at least 30 minutes), the fixture closes and joins its owned Serve goroutine. It writes `backend_terminal.json` even on failure after server start, with `testFailed` and observation errors. Final checks require one complete revision with original snapshot lineage and one `AWAITING_EXECUTION_AUTHORITY` / `NOT_RUN` request, exactly `OBSERVE_DATA` then `REQUEST_RERUN` human events at versions 2 and 3, case version 3 still `WAITING_ON_DATA` / `WAIT_DATA` / `NOT_RUN`, original source message/snapshot/proposals and the helper's other case/proposal unchanged, plus unchanged protected attempts/budgets/grants/executions/receipts/runs/AIUsageLog counts. Native denials are recorded separately from browser route counts. The Go test process must exit normally after this terminal; Root observes that exit independently.

## Exact Root provisioning / runtime contract

Root creates one **new**, identity-owned cached MySQL fixture; no historical R123 database or container is reused. In the new database, whose name matches `^ccma_r132_[a-z0-9_]{1,32}$`, create this table and insert exactly one cryptographically fresh 32-character lowercase hex nonce before the test starts:

```sql
CREATE TABLE ccma_r132_fixture_owner (nonce CHAR(32) NOT NULL PRIMARY KEY);
INSERT INTO ccma_r132_fixture_owner (nonce) VALUES ('<fresh-lowercase-hex-32>');
```

The exact external evidence directory is `../.ccmai_local_evidence/r132_waitdata_request_20261019/fixture` relative to project root. Immediately before test start it must contain **only** regular `fixture_identity.json` (no symlinks or prior output), with:

```json
{"trancheId":"CCMAI-RUNTIME-132","host":"127.0.0.1","port":33060,"database":"ccma_r132_<fresh_name>","nonce":"<same-fresh-lowercase-hex-32>"}
```

The port above is an example; use the actual new bound loopback port (1024–65535). Set `R132_WAITDATA_BROWSER=1`, `R132_EVIDENCE_DIR=<absolute exact directory>`, `R132_FRONTEND_DIST=<absolute frontend/dist>`, and `R132_MYSQL_DSN=<synthetic-user>:<synthetic-password>@tcp(127.0.0.1:<same-port>)/<same-database>?parseTime=true`. `R101_MYSQL_DSN` must be unset on admission; the fixture sets it only after nonce proof for the existing helper, then clears it. No real credentials are needed. Root removes the worker's offline compile log and binary from this external directory **before** creating the fresh identity-only admission directory.

`ready.json` supplies `origin`, tenant/case IDs and the helper's synthetic owner/reader login emails and password. The browser performs owner login, capture (first 503), explicit reload/retry capture (201), request (201), reload/history, logout, reader login/history with no write controls, logout. It writes `stop.json` with exact `{"trancheId":"CCMAI-RUNTIME-132","nonce":"<same nonce>"}` once observations are complete. It must not call execute/grant/provider endpoints. Root checks terminal, journal, held Go test exit and owned server/MySQL final stop, with protected-resource comparison.

## Checks and limits

One offline compile-only Go invocation was used: from `backend`, with `GOPROXY=off`, `GOSUMDB=off`, `GOTOOLCHAIN=local`, command `go test -c -mod=readonly -o (Join-Path $compileDir 'handlers.test.exe') ./api/handlers` after `$compileDir='..\.ccmai_local_evidence\r132_waitdata_request_20261019\fixture'`; exit **0**, count **1/1**. That initial command resolved to a project-local ignored temporary directory. I moved its raw log and binary directly to the required sibling external evidence directory and removed the empty temporary directory. The raw `compile.log` is empty (SHA-256 `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`). The compile preceded final source-only invariants and Root review-round-one repairs, so it is not an exact-final-source compile receipt. Root has the remaining seed Go invocation budget after source audit. `gofmt` and `git diff --check` passed; no DB/test/server/browser/provider run occurred by this worker.

Actual native/browser persistence, friendly UI error behavior, process exit, MySQL stop and protected-resource equality are **NOT_RUN** pending Root admission. This synthetic product fixture cannot prove CVF AI governance. No self-approval or FREEZE claim is made.

## Return-Time Closeability Recheck

closeabilityDisposition: CLOSEABLE

outsideAuthorityBlockers: NONE

nextRepairRoute: REVIEWER_LOCAL_REPAIR

workerRedispatchAllowed: YES

All source repairs remained within the existing R132 fixture authority and resolved Root's first source-review findings. Root's independently admitted runtime evidence and final review remain pending by design.
