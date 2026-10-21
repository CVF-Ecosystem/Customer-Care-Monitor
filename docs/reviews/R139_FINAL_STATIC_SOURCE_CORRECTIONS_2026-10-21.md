# R139 — final bounded same-scope source corrections

Status: CHANGES_REQUIRED_SOURCE_ONLY
Risk: R2
Worker: Sol Medium initial author; Luna Max proposed repair worker
Reviewer: Root independent source/evidence REVIEWER

Root review of the post-initial-revision snapshot, no runtime/check invocation. Original F01–F08 remain the acceptance contract. Sol stopped on a usage limit before worker Go3 and final source return. Shared worker Go2/3 both failed; Python2/3 latest12PASS reported; original logs were not persisted, so these are worker-reported facts, not reconstructed raw receipts. Source is not accepted.

1. Python Docker wrapper still inherits ambient environment despite minimal env at child launch. Pass the same explicit minimal env to Docker; preserve explicit local pipe/empty config and no proxy/context credentials.
2. Cleanup Docker operations still depend on successful journal writes before the command. Disk-full/journal error can prevent exact-owner inspect/stop and leave a running container. Add a bounded cleanup-only, best-effort-journal path: verify exact owner/image/mount/network, stop only the already-admitted owned container, retain errors and HOLD. Normal setup must still fail before effects if reservation/journal fails. Expired campaign deadline must stop campaign, not prohibit finite owned cleanup; bound and record cleanup separately, and mark any lifecycle overrun HOLD.
3. Child registry assignment follows fallible stream.close. Register returned Popen immediately before closing/logging; preserve cleanup if stream.close or PID receipt fails.
4. grantDeadlineUTC is only checked for presence in Python and ignored by Go issuer (uses now+5min). Parse UTC/future/allowed maximum before fixture and bind the exact admitted deadline to Go admission/env/server issuer. No newly minted expiry beyond owner authority. Ensure actual configured provider/model/prompt/config match admission; future runtime allowance remains0 now.
5. Cross-check final Go native read matrix, standalone admission, and terminal lineage/counters against final supervisor schema. Reject G1 failure before G2. Keep exact external evidence path and preserved graph; ensure success-state expectations are real assertions and all remaining gaps named. Do not fix product files or fabricate test output.

This is the second and final same-scope source revision under original authority. No worker Go/Python allowance resets. Remaining shared workerGo1/Python1, independent RootGo1/Python2. Record pre-effect reservations and exact command/stdout/stderr/exit/source hashes for remaining checks; binaries/raw logs only ignored runtime/private evidence, compact metadata only in project. If the same root cause needs a third source revision, stop and record REVIEW_COST_ESCALATION_REQUIRED. No blind retry/helper successor.
