# R046 R4 successor evidence corrections

Date: 2026-10-05 (Asia/Saigon). Author: Codex subagent r046_r4_worker, REPAIR_WORKER / BUILD COMMIT_STEWARD under separate R047 seed `a2cf5ded858c6ec5a6b0873e32bf984f65bc6968`. Parent Codex /root remains independent REVIEWER. Before-edit acknowledgment is committed at `b968383d6befa6d50527cffa5fb3b6368cacf78b`; no evidence edit preceded it. This packet supersedes inaccurate factual claims without rewriting historical packets. Worker disposition is **REVIEW_PENDING / REVIEW / FREEZE_OPEN**; original R046 remains CHANGES_REQUIRED. No acceptance or FREEZE is granted here.

The [R4 contract](CCMAI_RUNTIME_046_R4_COST_DISPOSITION_2026-10-05.md) and [R3 independent findings](CCMAI_RUNTIME_046_R3_INDEPENDENT_REREVIEW_2026-10-05.md) are inherited. This is one metadata correction pass with zero new runtime campaigns or mutations. Product and maintained tests remain at `91da0e88118b76a68031f432da50521fe6a341b7`. All old reports, receipts, runners, reviews and authority seeds are preserved.

## R046-R3-01: execution provenance

The committed Claude R3 runner mounts a filesystem livecopy, not the saved Git archive. The archive digest `1a1c07503902e5b3f8568ad19aba48765b9aadd5dfd107b5c4a557ee4abe33bd` and its commit comment identify the saved R2 archive only; they do not establish execution of that archive. [Machine-generated per-file manifest](probes/r046_r4_snapshot_manifest.json) records all 203 canonical export, saved archive, surviving livecopy and normalized hashes. The repository script `docs/reviews/probes/r046_r4_provenance.py` reads the explicit runner scratch paths without modifying them or running containers.

All 203 saved archive files equal a fresh R2 Git export. The surviving livecopy has three byte mismatches: `backend/api/handlers/agents.go`, `backend/api/handlers/job_dispatch_shared_test.go`, and `backend/engine/testdata/s0_vietnamese_intervention_corpus.json`. Each disappears after CRLF-to-LF normalization; there are no unexpected or missing files. **Executed-copy byte equality is false.** Normalized equivalence is a separate present-recovery observation, not exact byte equality or proof of past execution bytes. Raw Git blobs are a different comparison target because Git exports can apply line-ending conversion.

The original exact-archive execution condition remains **NOT MET / NOT VERIFIED**, with no implicit waiver. This packet improves disclosure; only a separate authorized contract disposition could accept an alternative. Protected finalizer production/test hashes still match the reference: `4b7477aa12fd6c232e85818960f6ec0f2947548cd6ac24e714568ebff5a90f0b` and `06ec2c8d15d6199a679f9daaf0835db1baa7a8a63a35b79df5e9465d90f7c449`. R2 independently accepted source/boundary observations remain inherited, not new R4 runs.

Eight recovered R3 logs match their saved receipt SHA-256 values again. Baseline is 22 top-level plus 6 subtest passes, 28 completed events; combined is 40 plus 47, 87 events. M01 names two failing tests; M02/M03 each name one. Restored selections pass. These are **Claude R3 observations**, independently recovered by Codex R3 and read-only recovered again by this subagent. Neither recovery certifies historical R2 execution, transient mutation bytes or worker environment checks.

## R046-R3-02: publication and commit attribution

| Layer | Resolved identity and limits |
| --- | --- |
| Claude R3 submitted handback | `c09c8f62585945d3624e5a0f3ddc305bb35c4552`; historical receipt's `PENDING_COMMIT` field remains unchanged. Its catalog/index generation is not a VitePress build; its publication assertions remain attributed worker claims. |
| Independently recovered R3 logs | Original logs and digests validated in R3 independent audit and this manifest; this proves recovered R3 results only. |
| Codex corrected R3 review publication | `fefa846f82c99b1cdf67913f206b8712eaca42f5`, with [review publication receipt](probes/r046_r3_review_publication.json). Initial worker Markdown docs build failed on the runner `.ps1` dead link; the pointer correction and later PASS belong to Codex's corrected snapshot. |
| New delegated R4 worker publication | Actual commands, exits, outputs, parent and artifact hashes are in [R4 worker receipt](probes/r046_r4_worker_receipts.json). R4 uses no containers or runtime campaign. Its eventual artifact commit is resolved in the committed handback or parent review, not fabricated within its own content. |

Historical R3 exact-staged execution without snapshot-bound output remains **NOT VERIFIED**. Historical internal-network inspections and teardown exit codes remain worker-declared rather than independently witnessed. R3 recovered teardown output names the task resources; Codex R3 current absence checks are observations at that review time. Cleanup is **INHERITED**, not newly executed R4 cleanup. No new container-absence observation is claimed here.

The receipt distinguishes actual R4 checks from inherited validation. Snapshot hashes exclude the publication receipt itself to avoid circular self-hashing; the staged path set includes it. Later exact handback SHA binds the complete packet. All gates prove checked repository metadata state only, not AI governance, hosted CI or runtime behavior.

## R046-R3-03: historical errata

| Original R3 categorical claim | Correction | Source and confidence |
| --- | --- | --- |
| R1 exercised only statement-level injection | R1 already added BEGIN-boundary injection. R2 added COMMIT/rollback hooks and Statement restoration coverage. | Maintained R1/R2 test history and R1/R2 independent reviews; high confidence. |
| R2 count errors were caused by ad hoc event grouping | The R2 discrepancy's cause remains unverified. Correct recovered counts here belong to R3. | R2 independent receipt audit versus R3 raw-log audit; high confidence in attribution limit, cause NOT VERIFIED. |
| Machine logs prove both historical R2 M01 tests failed | R3 M01 logs prove R3's two named failures: `TestFL01TransactionBoundaryUnknownErrorIsContained` and `TestFL05TransactionBoundaryNormalizationDetector`. Matching original R2 proof has not been recovered in this pass. | R3 log hashes and named output; high confidence for R3 only, R2 two-test result NOT VERIFIED. |
| Categorical Windows-host R2 execution environment | The recovered R3 packet does not establish R2's past environment. No R2 execution environment is certified by R4. | R2/R3 independent reviews; historical R2 environment NOT VERIFIED. |

Historical [R1 review](CCMAI_RUNTIME_046_R1_INDEPENDENT_REREVIEW_2026-10-04.md), [R2 review](CCMAI_RUNTIME_046_R2_INDEPENDENT_REREVIEW_2026-10-04.md) and [R3 review](CCMAI_RUNTIME_046_R3_INDEPENDENT_REREVIEW_2026-10-05.md) retain all failed/unsupported claims. No R3 evidence is relabeled as R2 evidence.

## Validation and handback

Before-edit acknowledgment checks passed: doctor PASS WITH NOTE25/1 (known manifest-pin warning and uninstalled migration profile), gate suite46 in17.276s, docs build15.51s, default/PR/exact-staged preflights7/7, both-shell catalog and diff check. One initial PowerShell5.1 catalog invocation lacked ExecutionPolicy Bypass and was policy-blocked; no evidence edit followed that failure until the correctly scoped rerun passed. A read-only guessed probe filename was absent during preparation; the actual committed probe was then read. Both incidents are retained in the receipt.

Final R4 publication checks follow this final Markdown and catalog generation; actual commands/results are recorded in the machine receipt. Child commits REVIEW_PENDING for parent independent review. No automatic fifth correction, source/test repair, contract waiver, push/merge/deployment/FREEZE, credential/provider/channel/external-network/persistent-data/live-governance or hosted-readiness authority. Facebook/Zalo OA accounts remain parked. Parent independently evaluates correction completeness and the unchanged exact-archive residual; the worker never self-approves.
