# R051 independent repair re-review

Date: 2026-10-05. Reviewer: Codex, independent of Claude REPAIR_WORKER. Disposition: **CHANGES_REQUIRED / REVIEW / FREEZE_OPEN** for R051 and residual R050. Exact test repair: `a4b378ae036ad767728fb2e0655559e9cf750b47`; handback: `c1072abbc46d6e7ce94cd54304a20a1c35837966`. No reviewer product/test repair.

Authority: separate R051 seed `a54cb73007f081fe4bbaa4baa11d28fed4a7d937`, [RP-01..04](../specs/ANALYZER_PROVIDER_REPAIR_R051_2026-10-05.md), [work order](../work_orders/CCMAI_RUNTIME_051.md), inherited [R050 findings](CCMAI_RUNTIME_050_INDEPENDENT_REVIEW_2026-10-05.md). Production `727d3229338e9b29c749612677a08b7fd1c65428` and both immutable seeds remain unchanged. The repair commit changes only the two authorized test files; ownership fixture adds one eligible conversation and preserves every assertion. Actual before-edit acknowledgment is `9f010bfefb24e1ade8fca12a9ca2dcaf968995a3`, committed before repair. The worker report's different full SHA does not resolve.

## Consolidated findings

### R051-R1-01 — P1: ordering mutation never executes a behavioral detector (RP-04)

`docs/reviews/probes/r050_r1_campaign_runner.ps1:205-216` inserts provider resolution immediately after `ctx := owner.ctx`, before `run` and `provider` declarations. Its unrestricted replacement also inserts into another function. The retained log, whose SHA matches the receipt, contains `undefined: provider`, `undefined: run`, `undefined: injectedProvider` and incompatible return signatures. Go emits `build-fail`; there are **zero named test completion events**. Exit 1 therefore means **INCONCLUSIVE / BUILD_ERROR**, not an ordering mutation killed through `earlyProviderUnavailable`. Restoration and the restored negative-control pass are valid separately.

Repair within existing authority: a successor runner must apply exactly one scoped ordering mutation in the isolated export, record match counts and the applied diff/hash, compile and execute the named LP06 no-work detector, require its intended behavioral assertion to fail, and restore original bytes in guaranteed cleanup. Assert mutant/restored outcomes rather than merely printing expected exits. Preserve this failed campaign and incorrect worker claim as history.

### R051-R1-02 — P2: machine handback still lacks verified identities and cleanup/publication observations (RP-04)

The runner copies the whole backend worktree (`:13`) and checks only analyzer.go against the production baseline. Receipt records three hashes, without the required full export manifest/test-overlay identity, acknowledgment/repair/handback mapping or changed/staged publication checks. Retained export has 204 tracked files and four byte differences from the repair commit: handlers/agents.go, handlers/job_dispatch_shared_test.go, go.mod and go.sum. Independent inspection finds these are CRLF differences only, **not a semantic production modification**; a truthful manifest must distinguish those bytes instead of claiming an exact Git export.

`anonymousVolumesAbsent` is assigned literal `true` (`:282`); no volume names or post-removal volume inventory were captured. `docker rm -f -v` is the correct teardown command but cannot support that observation by itself. Internal network/no host ports and retained raw command logs do support their narrower claims. Wrong acknowledgment SHA and per-suite counts recur: actual accepted-cancel suite has five subtests, and early-failure-class suite has five, rather than worker report's ten and three. Aggregate 42 top-level / 98 completed events / 98 PASS is correct.

Repair: successor evidence must bind its own exact committed archive plus declared two-test overlay/full manifest, actual acknowledgment SHA and exact repair/handback identities; derive names/counts from JSON; capture named volumes before teardown and verify each absence; record final staged paths, protected bytes, gate46/default/PR/staged preflights, catalog and final docs checks. Defer self-commit identity explicitly to committed handback. Keep old packets unchanged; a new campaign cannot retrospectively prove old volume cleanup.

### R051-R1-03 — P2: persistence observations remain capable of false passes (RP-03)

`backend/engine/analyzer_provider_initialization_test.go:201,208` ignores `.Count(...).Error`. A failed SQL count leaves zero and can pass the no-usage/no-results assertion. This was explicitly required in RP-03 and remains unfixed. LP01 reads the returned summary rather than reloading the persisted run/job, and the dedicated LP tests do not observe notifier effects. New missing/corrupt-key positive controls check returned failure/settings queries but lack the required persisted bounded error/finished state/ownership-release observations. Existing ownership/finalizer suites remain useful inherited evidence; they do not replace these scenario-specific observations.

Repair only the authorized initialization test file: fail on every observational query error, reload and check stored run/job/summary/checkpoint/results/usage and a synthetic notifier observer for the pertinent no-work/source-error/provider-failure cases, retaining all prior assertions. Distinguish counted settings queries and resolver seam invocations from actual decryption calls or factory construction, which are not separately counted here. No production seam expansion or live notification/provider use.

## Accepted portions and limits

| Contract / prior finding | Independent disposition |
| --- | --- |
| RP-01 / R050-R1-01 | Settled: eligible provider-selection fixture only, original assertions retained, required regression passes. |
| RP-02 / part of R050-R1-02 | Settled: single calls, batch calls and processed items separated; batch2 items is1 invocation and construction/resolver count may equal inference count. |
| RP-03 / residual R050-R1-02 | Partial: actual production settings query traps, corrupt-key no-work/source-error cases, candidate query failure and eligible missing/corrupt controls added; persistence/query-error proof remains open. |
| RP-04 / R050-R1-03 | Open: runnable ordering mutation and truthful complete successor receipt required. |

Independent exact-repair regression: **42 top-level / 98 completed events / 98 PASS / 0 FAIL / 0 SKIP**, command exit0, wall228.214s. Reviewer named database/network/anonymous volume captured and verified absent; this proves only the reviewer campaign cleanup.

Machine evidence: [independent summary](probes/r051_independent_summary.json). Rerunnable independent campaign: `docs/reviews/probes/r051_independent_campaign.ps1`. Reviewer exports the exact repair through `git archive`, uses cached images with `--pull=never`, readonly source/module cache, GOPROXY=off and disposable synthetic MySQL on an internal network without host ports. Full backend/race and new build/vet are NOT RUN in this review; worker build/vet raw logs and hashes are verified, and original independent production build/vet remain inherited. No new reviewer mutation was applied: the worker's retained build-fail already establishes the blocking defect.

Canonical rehydration, doctor PASS WITH NOTE25/1 (actual readonly core `8a4119e11db00e774ed8e7cf7d9a8caa309e81d1`, manifest pin mismatch warn-only), compact bootstrap absence/nonblocking migration note and local knowledge intake acknowledged. REVIEWER transitions to SESSION_SYNC_STEWARD / review-metadata COMMIT_STEWARD for this disposition only. Front surfaces agreed on pending review at intake; stale secondary catalog/roadmap/owner-routing and worker claims are corrected through successor current pointers, with original packets retained.

Next: Claude same-scope REPAIR_WORKER under existing R051 seed, committed rehydration/BUILD acknowledgment and preflight before edits, consolidate all three findings in repair round2; then exact committed REVIEW_PENDING handback for independent review. No new scope, production edit, push, merge, deployment or FREEZE. R050 stays CHANGES_REQUIRED; R049 and prior closures preserved. Facebook/Zalo OA remain OWNER_DEFERRED_FACEBOOK_ZALO_OA_ACCOUNTS. Evidence establishes synthetic application behavior only; real-provider CVF governance, full S2/global F02/live/hosted readiness remain unproved.
