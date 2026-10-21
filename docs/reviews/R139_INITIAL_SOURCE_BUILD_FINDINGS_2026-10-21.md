# R139 — consolidated initial source BUILD findings

Status: BUILD_FEEDBACK_SOURCE_NOT_ACCEPTED
Risk: R2
Worker: Sol Medium delegated fixture worker
Reviewer: Root independent source/evidence REVIEWER, assisted by read-only Luna Max

Static review of the first draft only. No DB/native/browser/provider/credential/grant/execute effects. Root collected predictable issues before the worker's final checks. Worker checks so far: Go compile2/3 both failed on source name/local issues; purePython1/3 PASS5. No complete worker handback yet; this is one coordinated initial BUILD revision, not accepted product evidence or a new helper tranche. Preserve all failed command receipts. Original source budget unchanged.

## F01 — ownership and held-child cleanup, P1

`run` assigns owner only after `start_fixture` fully returns; a failure after creating a network/volume/container can lose cleanup authority/handle. `launch` can spawn then fail writing PID receipt and lose its process handle. `stop_child` writes an effect log before terminate; a disk write failure can prevent termination. Publish mutable owned resource/child handles immediately to caller-owned registries before fallible evidence writes; independently protect all cleanup stages and log errors without preventing cleanup. Handle partial setup with exact names+nonce labels/IDs/mount/network verification; no broad deletion, retry or ambiguous ownership. Preserve owned volumes. Terminal JSON read/validation/write errors must not skip remaining cleanup.

## F02 — prerequisites and source/receipt identity before effects, P1

Verify all local inputs (source tree including dirty changes, test binary, frontend/Vite CLI/dependency pins, provider key path/limits/approval metadata), supported control/capture/close receipt content and actual pinned image cache BEFORE Docker mutations. Require bounded concrete provider/model/endpoint, approved synthetic namespace/issuer/actor mapping and amount/currency/deadline, not only booleans. Bind Go fixture to same admission and env values before its nonce connection; standalone Go invocation must not bypass admission. Runtime admission must be committed and exact-source bound; old DENIED receipt or source-resume admission cannot enable runtime. Sanitize Docker/subprocess env; avoid ambient HTTP proxies on loopback readiness. No extra secret read/download/probe now.

## F03 — G1 must stop G2 and include actual read matrix, P1

Ignore-free `t.Run` results and `t.Failed()` gate BEFORE real transport/key/browser readiness. Use exact native8 slots with sacrificial request/PromptVersion callback (ConfigRef alone does not deny markWaitDataCall). Add required read cases: writer READY; reader/no-jobs/foreign denied; disabled; no row vs unreadable503; expired/consumed/foreign grant; source/version/permission/budget/config mismatch; every relevant GET has durable before/after fingerprint and no handle except current READY. The current draft only covers a fraction. Fixture preparation writes are distinct from application no-write observations. This is native software-only; no synthetic AI output or actual runtime acceptance.

## F04 — immutable lineage and durable terminal evidence, P1

`InterventionAttempt` has no case_id; use execution_id/run_id and validate full request→execution→jobrun→snapshot→proposal→EXECUTE_RERUN event lineage. Compare complete original request identity/content, not only status/execution/version. Verify post-publication real status route/reader/history and frozen published version. Record final durable fingerprint, per-route counters and validated sanitized real request/response/usage (only future authority). Existing R102 helper cleans tenant rows after test return: do not imply the preserved volume holds the recorded graph. Prefer R139-owned seed/preservation-controlled fixture using existing model/helper source, without editing historical tests; otherwise explicitly account post-cleanup DB state. Emit terminal evidence on failures when possible, without hiding HOLD.

## F05 — external evidence versus runtime paths, P1

S01 requires exact fresh external `.ccmai_local_evidence/r139_execution_proof_20261021/fixture`, no symlink/junction and only ownership identity before connection. Current source uses ignored `.cvf/runtime` for all Go/provider evidence. Keep process/command state in ignored runtime; bind the specified external evidence directory for original private wire/fingerprint/terminal receipts and publish only safe compact project references/hashes. Validate before first SQL connection, with identity path equal the admitted location. No secret copied into project/Git.

## F06 — honest readiness and outcome validation, P1

Validate native receipts and correlated complete route ledger; health-check unique Vite nonce with child liveness before BROWSER_READY. Require original Go wait/exit0 and no skipped/failed required native stage; a populated JSON/count alone must not turn a failing test into supervisor success. Vite's explicit owned termination may have a platform termination exit code; classify it explicitly rather than pretending it is natural exit0. Correlate responses by unique request ID rather than completion order (concurrent GET responses may finish out of order). Stream failures/missing responses stay HOLD. Block unauthorized mutations at both genuine Go HTTP entrypoint and Vite proxy, and count native versus browser attempts separately.

## F07 — complete finite lifetime/ownership evidence, P1

Start one absolute monotonic lifecycle deadline before setup, retain cleanup reserve and clamp every setup/ready/wait/cleanup timeout. Do not restart a20minute budget after setup. Verify exact container/image/network/volume labels and mount/attachment IDs; inventory comparison must not exclude arbitrary resources solely by matching names. Verify exact API/UI/DB listener absence using reliable owned-port observations and original child joins, plus stopped container PID0/no published ports. Require readiness inputs before fixture; source guard must reject all existing browser denial artifacts.

## F08 — meaningful pure checks and bounded return

Add effect-free tests for partial resource setup failure, post-spawn receipt-write failure, cleanup log/terminal-read failure, false/denied browser receipt, dirty source/incomplete pin rejection, failed child/Go exit with superficially populated terminal, and response correlation with out-of-order completion. These test runner software only, not runtime AI governance. Use remaining worker Python2 and Go1 budget only after coordinated revision; Root retains independent Python2/Go1. All runtime/browser/key/provider0, no old cap reset, no product path widening. Return complete changed hashes, commands and retained failures, exact remaining gaps and source-only disposition.

Luna's initial parked-checkpoint drift report was a false positive: canonical state uses parkedOperatorCheckpoint, compact bootstrap uses parkedCheckpoint. Root verified actual nonnull exact values and gate source; Luna acknowledged correction. No redundant alias/authority mutation was needed. Some ongoing draft findings (Vite health and ledger validators) were being addressed before this consolidated snapshot; assess final source once rather than treating intermediate missing code as persistent defects.
