# R107 partial historical R101 UI independent review

Status: CHANGES_REQUIRED
Risk: R2
Worker: /root/r107_vite_worker (Luna Max, REVIEW_EVIDENCE_OPERATOR)
Reviewer: Codex /root (independent of UI execution and source candidate author)
Disposition: BLOCKED_WITH_REASON

The integrated browser and sequential-import Vite configuration worked in the admitted campaign. Product acceptance remains blocked: B01–B05 and B07 have positive observations, B06 REOPEN failed with HTTP 503, and B08–B10 were not completed. This is partial historical R101 synthetic UI evidence, not current R102 backend acceptance. No FREEZE or product repair is granted by this review.

## Authority, source and separation

Execution HEAD was `a489e4954d954ac45ab5e5228ab5fecd89627f17`; immutable dispatcher seed/base `5b019f563009b1fc82205d3f1dcc9cf13efc075a` preceded BUILD. Historical application source was `200df3e4a7bc2a326425922a5230508a4fd9b0fe`, binary SHA256 `4c7575e180dd198d995bfe09bdc45061740a987be8ec0765f02b203e1c251f31`. The current checkout includes R102 backend changes; the old binary does not establish their UI behavior. Accepted R107 actual Node module loading and 13 offline checks at `59127d0a37ed78bca74c6e76f347ecf5a1cc4b0c` are inherited, not replayed. The runner and template remain pinned by the immutable seed. Root reviews evidence and synchronizes metadata; the UI worker neither commits nor approves.

Fresh continuity rehydration and doctor passed 27/27, with the historical manifest pin warning: actual public Core `fa34ed49e56648acafe8b0388e48e0aa0523f007` matches origin/main. Core remains read-only. Risk ceiling R2, live evidence required YES for governance claims; no provider call, mock governance, customer data or governance behavior claim was made here.

## Browser result and product result

Root's pre-campaign HTTP/control/PNG capability receipt is `probes/r107_root/browser_capability_receipt.json`. Worker integrated IAB screenshots and accessibility observations, plus ready/running receipts, prove this campaign reached the application. The failed visibility-override calls are preserved; subsequent calls without the override succeeded. This does not prove a Claude/Antigravity bridge or continued browser availability: the resumed session returned empty browser/app inventories.

| Case | Review result | Expected and observed |
| --- | --- | --- |
| B01 | PASS | Owner login, queue, fresh case and original proposal/history visible before mutation. |
| B02 | PASS | CORRECT preserves proposal and records human fields/reason, v2; action HTTP 200. |
| B03 | PASS | Stale v1 CONFIRM conflicts; original backend log records HTTP 409 at 14:48:05, UI refreshes v2. |
| B04 | PASS | ASSIGN persists member/deadline after reload at v3; HTTP 200. Root visually inspected `23_b04_after_reload.png`. |
| B05 | PASS | RESOLVE persists outcome at v4 and appears in resolved filter; HTTP 200. |
| B06 | FAIL | One REOPEN request at 14:49:55 returns HTTP 503; GET succeeds but case stays RESOLVED v4 with no REOPEN event. Expected historical target is ASSIGNED. |
| B07 | PASS | Reader login sees queue and WAIT_DATA detail without action form. Root inspected `45_b07_reader_detail.png`. This proves visible controls only, not server-side tamper resistance. |
| B08 | NOT_RUN | Interrupted before responsive/viewport acceptance; control support was not established by a test. |
| B09 | NOT_RUN | Interrupted before offline/error acceptance; no network-control test was performed. |
| B10 | NOT_RUN | Reader WAIT_DATA observation is insufficient for owner guard/escalation acceptance. |

Root inspected saved screenshots, AX and action metadata against the unchanged exported backend log. The case route ID differs from the queue's displayed conversation ID by design. B06 raw metadata expects OPEN; source REOPEN targets ASSIGNED. Preserve that expectation error separately: HTTP 503 and absence of transition remain failures under either expectation. The generic action-unavailable handler response does not reveal the underlying database/engine error; root cause is NOT_ESTABLISHED. No source change or retry was made.

## Impact and cleanup limits

Actual starts were fixture/backend/Vite 1/1/1, new fixtures 0, retries 0, product tabs 2 (IDs 1 and 2), five case submissions plus owner login/logout and reader login = eight POST submissions. A stale rejected action counts. Interim B06 metadata overcounts one pending/actual action; the original values remain intact and the worker explains the corrected count. Provider/mock/compile/new product test/build counts are zero in this campaign.

The usage-limit interruption outlasted the 2100-second allowance. No further runtime or UI action is authorized. Root's `probes/r107_root/post_runtime_independent_verification.json` records the exact fixture already exited, recorded backend/Vite PIDs absent, and protected Docker safe fields equal to the captured baseline. Original container/volume/network remain preserved. Cooperative stop wrote a request, but polling session 37563 returned `write_stdin failed: Unknown process id 37563`. No `cleanup.json` exists: supervisor exit code, finally completion and deadline enforcement are NOT_OBSERVED. TCP result 10035 is inconclusive. Empty current browser inventory prevented tab close calls; closure of tabs 1/2 is NOT_OBSERVED. These are explicit acceptance gaps, not a cleanup PASS.

## Evidence integrity and checks

Worker return: `R107_R101_UI_WORKER_RETURN_2026-10-16.md`; evidence directory `probes/r107_cua/`. The runtime manifest preserves 30 files / 61,727 bytes and the empty docker-cli-config directory. Later observations contain actual PNG/AX bytes and timestamped metadata, but a complete raw CUA invocation transcript and final stream EOF receipts are absent. Do not reconstruct them or claim complete invocation provenance. Root independently verifies the final all-artifact manifest before accepting the archive; the separate byte contract preserves captured bytes when Git text normalization differs.

The original worker-return packet ran once while the order was ACTIVE: six controls PASS, including APPL checked. Its saved command/stdout/stderr are in `worker_return_gate_tool_call.json`. Final report clarifications came later; that invocation does not cover those edits. Root's final explicit reviewer-fast/pre-commit packet covers the final report and this review. Once order status is BLOCKED_WITH_REASON, APPL's NOT_APPLICABLE_WITH_REASON is expected and is not closeability approval. All actual gate lines, including failures and applicability limits, remain in the Root receipts. The resumed gate unit suite passed 89 tests; it is static governance tooling proof only.

## Disposition and next bounded move

Final Root verification: `r107_cua_worker_return_evidence_manifest.json` has 176 matching SHA256/size entries totaling 3,955,004 bytes. The separate `probes/r107_root/captured_worker_evidence.zip` verifies 177 members (including the manifest), preserving captured text and the empty directory; `captured_git_byte_contract.json` records seven Git representation differences. Archive SHA256: `244de5893a022de73ae02aebcfae9885a860151cff9f069e32ac90e5c4d179fe`.

Saved final check results:

| Receipt under probes/r107_root | Result | Actual applicability |
| --- | --- | --- |
| resume_gate_units | 89 PASS | Static tooling unit suite, no product/runtime claim. |
| final_legacy_preflight | PASS 7/7 | provenance, continuity, tranche, claims, secrets, workflows, catalog. |
| final_reviewer_fast | PASS, 7 mandatory controls, 0 failed | INST/CONT/RANGE/ROUTE/ROLE/CLAIM PASS; APPL NOT_APPLICABLE_WITH_REASON: BLOCKED_WITH_REASON, closeability NOT checked. |

Each receipt has original command, exit code, stdout and stderr. No FAIL was emitted by those final checks; the product B06 FAIL remains. Pre-commit receipt is recorded separately after final metadata/catalog staging, without converting the disposition to acceptance.

`final_precommit` returned PASS with six mandatory controls / zero failed; APPL explicitly remained NOT_APPLICABLE_WITH_REASON. Independent comparison also found all 30 runtime originals still match their exported hashes. `raw_whitespace_check` intentionally retains exit 2: 30 original AX lines contain trailing spaces. Those captured bytes were preserved. The authored-files-only check passes. Root's first attempt at that exclusion failed in PowerShell because the pathspec was unquoted; the corrected quoted invocation is saved and no project action occurred in the failed attempt. No global Git setting or raw evidence was changed to hide the warning.

Return R107 as CHANGES_REQUIRED / BLOCKED_WITH_REASON, freeze OPEN. Retain six positive observations without accepting B01–B10 as a complete suite. Root must bound investigation of the REOPEN 503 and the source/fixture contract before any product repair or renewed UI campaign; remaining B08–B10 and cleanup evidence need a finite successor authority. No automatic restart or dispatcher-seed edit follows this review. Three Dashboard baseline failures, full M2/M3, pilot and release stay OPEN; R102 local frozen acceptance is unchanged; Facebook/Zalo OA remain parked.
