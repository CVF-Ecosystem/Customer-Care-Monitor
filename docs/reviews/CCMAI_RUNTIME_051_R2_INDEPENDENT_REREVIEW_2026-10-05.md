# R051 round 2 independent re-review

Date: 2026-10-05. Codex independent REVIEWER, separate from Claude REPAIR_WORKER. **CHANGES_REQUIRED / REVIEW / FREEZE_OPEN** for R051 and residual R050. Target repair `145bd41109c1e3ac3fb1a85f261c61f668b2fe4d`, submitted handback `0bfe3654cb1a0ae75105093cf076f754be169d08`. Reviewer changes only review/continuity metadata, not worker source, tests, runner, receipt or historical packets.

Authority remains separate seed `a54cb73007f081fe4bbaa4baa11d28fed4a7d937`, [RP-01..04](../specs/ANALYZER_PROVIDER_REPAIR_R051_2026-10-05.md) and [work order](../work_orders/CCMAI_RUNTIME_051.md), with [previous findings](CCMAI_RUNTIME_051_INDEPENDENT_REREVIEW_2026-10-05.md). Production baseline `727d3229338e9b29c749612677a08b7fd1c65428`, original R050 seed and R051 seed are unchanged. Round2 repair changes only the authorized initialization test file; the settled ownership fixture and other tests remain unchanged.

## Findings disposition

| Previous finding | Independent result |
| --- | --- |
| R051-R1-01 | Behavioral detector repaired: raw mutant log hash matches, no build-fail, named LP06 negative control fails the intended no-work-success assertion. Restored control passes and retained export equals archive bytes. Remaining harness/receipt requirements are consolidated below. |
| R051-R1-02 | Partial: exact archive, dynamic aggregate counts and captured named anonymous-volume removal added. Acknowledgment SHA, complete manifest/publication receipt and isolation requirements still incomplete. |
| R051-R1-03 | Settled: Count queries check Error, stored runs/summaries and relevant job/checkpoint values are reloaded, terminal/ownership checks and zero notifier effects added; mixed partial work has a one-dispatch positive control. Original LP assertions preserved. |

## R051-R2-01 — P2: RP-04 machine handback remains incomplete

This is a residual of the previously identified receipt/harness root cause, not a new production defect or expanded acceptance requirement.

1. **Wrong acknowledgment identity recurs.** Report, receipt and active handoff claim `c228931df25026211cf4e33d4586db7020bc8b5c`, which does not resolve as a commit. Actual before-edit acknowledgment is `c228931c35339f56cac0a934394e586082a3f7c1`, the parent of the exact test repair. Its existence and ordering are confirmed; the false full SHA cannot be accepted as a verified identity.
2. **Complete handback/publication receipt is absent.** The committed receipt has archive hash, aggregate test counts, raw logs and resource observations, but lacks the required full file/hash manifest, changed/staged set, exact publication checks and explicit final artifact-commit/handback mapping. Old planningPublication fields are historical, not final round2 gate evidence. Reviewer independently verifies the B4D4DAFF archive equals a newly generated archive of145bd411, all204 retained files equal that archive, and command/mutant/restored hashes match. These are valid reviewer observations; they do not retroactively establish the worker's missing final checks.
3. **Compile/vet use default Docker networking.** `docs/reviews/probes/r051_r2_campaign_runner.ps1:92,112` omit `--network`; tests/mutations use the named internal network. `GOPROXY=off` and `--pull=never` restrict dependency access but do not disable default bridge connectivity. Receipt's blanket externalNetwork=false is not established for every command. No actual external API call was observed or asserted by this review. All containers must use the authorized internal network or `--network none` for commands needing no database.
4. **Mutation restoration is not guaranteed on failure.** Runner writes the isolated mutant at216, checks the kill and may throw, then restores bytes only on the success path at269. Its outer finally at299 removes Docker resources without restoring source. The successful retained campaign did restore byte-for-byte; a failed mutant/harness path would leave the isolated source mutated. RP-04 and the prior finding explicitly require guaranteed byte restoration, match counts and an applied diff/digest. These safeguards remain absent from the successor runner/receipt.

Concrete remaining correction: retain old submissions; derive actual Git identities, bind a successor exact committed archive/full manifest and final staged/gate/docs/catalog results, use internal/none networking for every command, record scoped replacement count and applied diff/hash, and restore original bytes in a nested finally. No new application test or production change is indicated by this review. Do not rerun the full backend or broaden acceptance merely to repair metadata.

## Independent evidence and claim boundaries

[Machine summary](probes/r051_r2_independent_summary.json) records exact export manifest, completed test names, raw command identities, worker audit and independent named-resource teardown. Reviewer campaign is rerunnable at `docs/reviews/probes/r051_r2_independent_campaign.ps1`, using git archive145bd411, cached images, readonly source/module cache, GOPROXY=off and disposable synthetic MySQL on an internal network/no host ports.

Independent campaign: **42 top-level / 98 completed events / 98 PASS / 0 FAIL / 0 SKIP**, command exit0, wall229.364s; named database/network/anonymous volume verified absent.

Worker aggregate42 top-level/98 completed98 PASS0 FAIL/SKIP independently parsed and verified. Worker mutation is accepted from retained raw named behavioral failure plus restored-pass logs and archive equality; no new reviewer mutation is necessary. New reviewer compile/vet/full backend/race NOT RUN: the focused test compiles the changed test file, worker engine compile/vet logs/hashes are checked, and prior production build/vet remain inherited. Full S2, global F02, real provider/channel, CVF runtime governance and hosted readiness remain unproved. No real config/credentials/customer/persistent DB, push, merge, deploy or FREEZE.

Fresh rehydration: manifest/policy/state/memory/handoff/implementation/index, relevant order/SPEC/seed, public core instructions/workspace rules/bootstrap/knowledge and shared repair learning read. Doctor PASS WITH NOTE25/1, actual readonly core `8a4119e11db00e774ed8e7cf7d9a8caa309e81d1`, manifest pin26c686cc mismatch warn-only; compact bootstrap absent/nonblocking migration note. Front surfaces agree REVIEW_PENDING at intake; stale secondary index/spec/owner routing are synchronized to this result. REVIEWER transitions to SESSION_SYNC_STEWARD / review-metadata COMMIT_STEWARD solely to publish the disposition locally.

## Repair cost escalation and next move

**REVIEW_COST_ESCALATION_REQUIRED** is recorded before any third repair round. RP-01/02 and R050-R1-01/02 are settled; residual R050-R1-03 / RP-04 remains open. R051 has now submitted two repair rounds, and these residual requirements repeat the existing root cause. No third BUILD is dispatched.

Project `AGENTS.md`, Governance Latency and Approval Continuity, requires: “At repair round three without an independent new root cause, stop and record `REVIEW_COST_ESCALATION_REQUIRED` before continuing.” Applicable shared learning also states that it does not approve repair round three. ORCHESTRATOR must evaluate this concrete residual evidence/harness correction and record a governed cost disposition before further BUILD. Reviewer cannot waive RP-04 or self-repair the worker packet to manufacture acceptance.

R050/R051 remain CHANGES_REQUIRED with freezeOPEN; historical failures and earlier closures preserved. Facebook/Zalo OA remain OWNER_DEFERRED_FACEBOOK_ZALO_OA_ACCOUNTS. This return is a completed independent review, not a new repair dispatch.
