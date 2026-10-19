# Proposed R105 — bounded historical R101 UI evidence

Status: DRAFT
Risk: R2
Worker: Claude / Antigravity-IDE as REVIEW_EVIDENCE_OPERATOR
Reviewer: Codex /root
Commit mode: WORKER_MUST_NOT_COMMIT
providerExecutionAuthority: NONE
dispatchKind: INITIAL
Dispatch readiness: NOT_DISPATCHABLE
Author: Codex /root, WORK_ORDER_AUTHOR
Date: 2026-10-16

## Purpose and current authority boundary

Collect real B01-B10 browser evidence for the historical R101 human-workflow implementation at200df3e4a7bc2a326425922a5230508a4fd9b0fe. This is historical synthetic software evidence; it cannot establish current R102 backend acceptance, full M3 completion, AI governance, pilot or release. Product-source repair is outside scope. Three Dashboard baseline failures remain OPEN.

This is the corrected work-order draft, replacing the 2026-10-15 Claude evidence handoff and the subsequent chat prompt. Proposed R105 is not an opened tranche. R104 names diagnostic artifacts only; no R104 authority/record exists. Do not create authority by renaming a worker report. Canonical active tranche remains CLOSED R103; its unused slots stay unused. No runtime action is granted by this draft or its candidate seed. Owner's no-commit/no-FREEZE instruction remains effective. Root may prepare/review documents, not dispatch BUILD under an uncommitted seed.

Candidate authority: R105_R101_UI_AUTHORITY_CANDIDATE_2026-10-16.json in this directory. Before execution Root must publish a formal order under docs/work_orders/, create the matching authority/tranche records, commit the immutable seed before reliance, set baseCommit to a commit containing it, synchronize canonical continuity and pass active pre-dispatch checks. No bootstrapException is proposed. Activation is a Root-owned remaining step, not a worker task.

## Dispatcher evidence and remaining holds

Read docs/reviews/learnings/feedback_dispatch_readiness_and_browser_evidence.md first. Root's actual inspection receipt is docs/reviews/probes/r105_dispatch_preparation/readiness_inventory.json.

| Condition | Actual result | Owner / required action before dispatch |
| --- | --- | --- |
| Target selected | HISTORICAL_R101_ONLY | Root; never label this current backend proof |
| Existing binary | PASS exact SHA256 below | Worker rechecks immediately before launch |
| Historical source mapping | PASS17/17 against Git200df3e with explicit CRLF mapping | Preserve manifest and original byte hashes |
| Current source guard |14/17 match; router/model/mysql drift at current64238c4 | Expected historical/current distinction; old review_campaign.py is not an admissible runner for this campaign |
| Fixture identity | PASS safe inspect; stopped, label/image/mount/network match; network has no attached containers while stopped | Root verified exclusive volume use in volume_exclusivity.json; recheck identity/current safe inventory and post-start mapping/readiness through the reviewed runner |
| Protection baseline | Old baseline differs: added /ccmai-r102-review-20261015 | Preserve old baseline; new campaign takes a separately named current baseline and proves its protected IDs/images/mounts/state unchanged across its own run |
| Cached frontend tools and inputs | Vite entry exists; all137 tracked frontend files pinned in frontend_input_manifest.json | Root rechecks these captured hashes; temporary no-env/API-only runner config still requires source review |
| Historical integrated bridge provenance | Root compared original invocation/response, source-copy recording and before/after frames; accepted data-URL diagnostic only | SATISFIED for historical provenance; see docs/reviews/R104_BRIDGE_SUPPLEMENT_INDEPENDENT_REVIEW_2026-10-16.md. Actual HTTP access and live driver/cache remain unobserved; no diagnostic replay |
| Runner/config source | Root independently accepted final source f27c5bfc with12 offline tests, syntax and actual input checks | SATISFIED_SOURCE_ONLY: docs/reviews/R105_RUNNER_INDEPENDENT_SOURCE_REVIEW_2026-10-16.md; real lifecycle remains NOT_RUN |
| Authority/continuity | No effective successor seed/order/record | HOLD: Root activation only, respecting current no-commit instruction |
| End-to-end HTTP access | NOT_RUN in new campaign | Check against the actual owned Vite origin after admitted launch; do not infer it from about:blank |

The worker receives an active order only after every remaining pre-dispatch HOLD is resolved by Root. Post-start readiness is then a bounded first stage within already granted authority. Missing prerequisite discovered by Root must not become another speculative worker assignment.

## Exact inputs and resource identity

- Project CWD: D:/UNG DUNG AI/TOOL AI 2026/CVF-Workspace/Customer-Care-Monitor-AI. No Core or other application working directory.
- Historical source:200df3e4a7bc2a326425922a5230508a4fd9b0fe. Source manifest: docs/reviews/probes/r101_root/formatted_source_manifest.json. Preserve its hashes and known byte representation; verify all17 against historical Git blobs and all9 manifest frontend paths against the current files. Full tracked frontend source/config/lock pins are in docs/reviews/probes/r105_dispatch_preparation/frontend_input_manifest.json; verify captured bytes before dispatch and launch.
- Binary: .cvf/runtime/r101_worker/r101_handlers.test.exe; SHA2564c7575e180dd198d995bfe09bdc45061740a987be8ec0765f02b203e1c251f31. No compile or replacement.
- Container: e1555e03d69360e1cd7729afd9e52856443f5172d76454653beec981a7d6477a; name ccmai-r101-review-final-20261015.
- Image: sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b; label ccmai.tranche=CCMAI-RUNTIME-101.
- Volume: ccmai-r101-review-final-20261015 at /var/lib/mysql, sole admitted container user.
- Network:35c3959ac41517050defad05cbad7a08717b24c8879683e25baf3e2702625e98; same name as container; exclusive to that fixture when running.
- Database: r101_fixture. Fixture-only credentials come from the pinned synthetic fixture source; never read .env or Docker Config.Env.
- Old recorded port56039, API URL52962 and PID9536 are historical only. Actual DB port must be127.0.0.1 after start; actual API URL/tenant/case/reader/owner must come from the new process stdout.
- Old counters remain workerGo1/reviewerGo2/aggregateGo3, fixtures2/2, nodeChecks3, frontendBuilds1, provider0/mock0. Do not edit or reset them for this proposed new campaign.

## New runner contract — prerequisite owned by Root

This document does not authorize execution by the evidence operator. Root staged the runner/config under docs/reviews/probes/r105_root/; independent source review is PASS_RUNTIME_NOT_RUN in docs/reviews/R105_RUNNER_INDEPENDENT_SOURCE_REVIEW_2026-10-16.md. Exact captured pins and formal field values are prepared in R105_R101_UI_AUTHORITY_CANDIDATE_2026-10-16.json; no unknown command is delegated to Claude. The formal seed must cover both Root preparation artifacts and worker outputs, with separate worker path restrictions; or the reviewed runner must already be committed before the tranche baseCommit and bound read-only by hash. Settle this before committing the immutable seed, not after BUILD.

The runner must keep new receipts under docs/reviews/probes/r105_claude/ and runtime logs under .cvf/runtime/r105_claude/. Never overwrite R101/R103/R104 manifests, baselines, counters, ready files or process receipts.

Required stages: inspect -> reserve -> start existing fixture -> bounded readiness -> launch pinned binary -> publish new readiness -> launch owned Vite -> observe UI -> cleanup. Reserve each effect before executing it, keep actual exit/stdout/stderr/timeout and monotonic attempts. Record failed/abandoned attempts; a crash must not reset budget. Reject an existing reservation or output directory rather than replaying it.

Inspect uses safe Docker fields only and validates IDs, image, labels, sole-volume ownership, exclusive network and protected current inventory. A separately named new baseline may account for the known R102 addition but cannot replace the old baseline or waive unexplained drift. Start exactly the existing container once; no setup/native branches, recreate, compose, prune or pull. After start poll port/SQL readiness at most90s; accept only one loopback mapping and SELECT1/schema r101_fixture, no access to another DB. Record actual port only in new evidence.

Binary environment: R101_MYSQL_DSN points solely to that validated fixture and R101_BROWSER_FIXTURE=1. Command is the pinned executable with -test.run ^TestR101BrowserFixture$ -test.v -test.timeout=35m. Reserve one binary launch first. Historical test seeds a NEW synthetic tenant/case in this existing disposable DB; this mutation is explicitly part of the proposed authority. It is not reuse of old tenant readiness. Parse R101_BROWSER_READY and R101_BROWSER_OWNER from this launch only; fail on early exit or no readiness within60s. The fixture server ends after30min; do not restart on expiry.

Vite uses cached node_modules only, VITE_API_TARGET set to the newly parsed API origin, host127.0.0.1, port3011, --strictPort. Before start verify that port is free; if occupied stop without killing its owner or silently choosing another origin. No .env read: the active runner must specify a reviewed Vite config with envDir disabled and only /api proxy to the new fixture; do not use existing default /mcp forwarding to localhost8080. Keep source/config/dependencies unchanged; any temporary config lives in ignored R105 runtime and is hashed from reviewed source. This config template/renderer and exact Vite invocation are now source-reviewed; the generated runtime config hash and fresh API origin are captured by the admitted runner, not improvised by the worker.

Cleanup must verify PID/start-time/command ownership before stopping only this campaign's Vite and backend, close only owned tabs and stop only this container if the campaign started it. Preserve container/network/volume/data. Compare safe protected inventory before/after; retain anomalies and residual owned PIDs. No recursive delete, cache changes or user-app restart.

## Proposed finite budgets — ineffective until seed activation

| Effect | Maximum |
| --- | --- |
| Provider/mock/secret/customer/channel calls |0 |
| Compile/native/frontend test/build/install/download/pull |0 |
| New/recreated Docker fixtures/networks/volumes |0 |
| Start existing named fixture |1 |
| Pinned historical backend binary launch |1 |
| Vite server launch |1 |
| Product browser tabs |3 simultaneously; close only owned tabs |
| Application action submissions |10 total, including rejected/conflict submissions |
| Backend lifetime / whole product campaign |30min /35min including cleanup |
| Automatic runtime retry / source repair |0 |

Use a new campaign ledger; do not consume or claim unused historical slots. Evidence/log export corrections can occur without runtime replay. On application defect record repro and stop dependent actions; Root decides any separately bounded source repair. A failed readiness stage does not permit another start or a new fixture.

## Worker procedure after a future active dispatch

Rehydrate canonical state and emit declaration; verify effective order/seed/build binding and the Root pre-dispatch receipt. Revalidate ephemeral identities/ports and cached hashes without reopening settled design. Use Antigravity browser_subagent only through its documented permitted operations. No file://, CDP/standalone substitute, network interception, cross-surface workaround or policy bypass. Reuse exported R104 capability evidence only with its limitations; navigating the real127.0.0.1 origin remains a separate observation. Stop on a policy denial; preserve exact tool response.

Do not run the old review_campaign.py ready_existing/serve commands: they require all17 current-tree hashes and the old protection baseline and overwrite shared historical artifacts. Do not change those guards. Only the new reviewed runner admitted by Root is allowed.

Login uses new ownerEmail/reader values from the new readiness receipt; literal synthetic password is defined in the pinned test, never a real account. Stop if setup/origin/tenant is unexpected. No MCP/OAuth/channel sync/jobs/reset-demo routes, provider construction or notifications.

## B01-B10 acceptance and evidence

Requirements inherit H01-H08 of docs/specs/M3_HUMAN_WORKFLOW_R101_SPEC_2026-10-15.md. Native proof is inherited, not rerun. Capture original proposal/history before any action. For each case preserve actor, time, tenant/case/version, expected/actual, raw tool output and screenshot reference. Desktop target1440x900 and narrow390x844; report actual dimensions. No PASS based solely on narration.

| ID | Ordered action | Required observation |
| --- | --- | --- |
| B01 | Owner login, queue and new pending case detail | Correct tenant/case; original proposal/history captured |
| B02 | Open same pending case in tabs A/B; A CORRECT with reason/explanation/next action | Human correction separate from original; actor/time/reason persisted |
| B03 | Tab B submits stale action if UI permits | Conflict409 or explicit local stale protection; no false success/replay. Auto-refresh preventing stale submit means server-conflict branch NOT_RUN |
| B04 | Latest state ASSIGN tenant user/deadline, reload | Assignment/deadline/status persist, no external notification |
| B05 | RESOLVE with outcome, reload and terminal filter | Persisted outcome/status and retained prior history |
| B06 | REOPEN with reason/future deadline, reload | Reopened state; prior outcome/history/proposal retained |
| B07 | Logout/login new reader, queue/detail | Read access, no write controls/actions; never reuse owner token |
| B08 | Narrow viewport queue/detail/history; owner form only after owner login | Readable content and usable controls; no hidden important action |
| B09 | Tool-supported offline reload then online reload, if supported | Real error/recovery without fake success or action replay; unsupported offline is NOT_RUN, no firewall/network change |
| B10 | Existing WAIT_DATA synthetic case, if present in this fixture | Owner/deadline/escalation and confirmation/resolve guard observed; no AI rerun or new fixture. Missing case is NOT_RUN |

Run B02/B03 before B04. Read-only fixture API observations may corroborate history but cannot replace UI observation. Do not create cases through an unrelated API or mutate the fixture just to manufacture coverage. Missing required observations keep acceptance open.

## Return and review

New report: docs/reviews/R105_R101_UI_WORKER_RETURN_2026-10-16.md. One valid Status line COMPLETE_PENDING_REVIEW or BLOCKED_WITH_REASON, Risk R2, named Worker and Reviewer. Separate tool diagnostic from product results and historical target from current source.

Include exact admitted runner/config/inputs, start/stop/port/PID/ledger receipts, B01-B10 table, before/after DOM observations and raw browser tool responses or exported accessible recording, unedited PNGs, errors and limitations, cleanup/protection comparison, SHA256/size manifest with explicit Git line-ending handling. Save gate command/stdout/stderr/exit with the effective order and return passed as explicit packets. A no-packet/N/A gate is not acceptance. Do not create Root's independent review or label a placeholder accepted.

Root independently evaluates evidence. No worker commit/push/approval/FREEZE; no full M2/M3/pilot/release or runtime AI governance claim.

## Gate-To-Role Closeability Contract

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: NOT_CLOSEABLE
returnTimeRecheck: REQUIRED_BEFORE_REPAIR
implementationTopologyPolicy: BOUNDED_PATH_FAMILIES
foreseeableFileSplitDisposition: COVERED_BY_BOUNDED_PATH_FAMILY

Root must replace NOT_CLOSEABLE only after the activation, runner/config review and current frontend/resource revalidation holds are resolved; historical session evidence is accepted with qualifications. A static draft check may return NOT_APPLICABLE; it cannot establish active closeability. The later active order must bind authorization/seed history, source/config review, pre-dispatch, reservation/readiness, worker-return, independent review and cleanup to their actual owners and evidence paths. No active dispatch is claimed in this draft.

## Historical bridge supplement review — 2026-10-16

Root accepted existing data-URL diagnostic provenance in docs/reviews/R104_BRIDGE_SUPPLEMENT_INDEPENDENT_REVIEW_2026-10-16.md. No need to request another evidence-export/browser retry. Manifest6/6 verified and copies match original local sources; recording shows WAITING then CLICKED. The worker withdrew its earlier tab-closed claim: trace has no close call and metadata reported ACTIVE at capture. Residual-tab disposition must verify exact original page ownership on the original supported surface; no process kill or new browser action is granted by this draft. Manifest self-hash/size in worker prose is outdated; independent review records the correction. No HTTP/Vite/R101 proof or exact integrated driver/cache observation follows. Remaining runner/config and authority holds stay with Root.

## Source-prepared execution entry points — not yet authorized to run

Root source review and exact pins: docs/reviews/R105_RUNNER_INDEPENDENT_SOURCE_REVIEW_2026-10-16.md; docs/reviews/probes/r105_root/source_review_receipt.json. Final runner SHA256f27c5bfc729c67bf717473a7dbc894352446c9f74813be08243350e46a4d478a; template8c8a40d463247443044578a16cdd2452f44c090d6f5c7fdf5a6068e67c5efbda. Root actually verified12 offline tests,137 frontend hashes,17 historical Git rows, pinned binary and safe read-only Docker format/engine binding. This does not prove server/browser lifecycle.

After Root's effective formal dispatch only, from project root:

```powershell
python -B docs/reviews/probes/r105_root/r105_ui_runner.py inspect --project-root .
python -B docs/reviews/probes/r105_root/r105_ui_runner.py start --project-root .
```

Start is a foreground supervisor, not a detached command. Keep its original tool/session alive while the browser operator works. Wait for BOTH .cvf/runtime/r105_claude/ready.json AND running.json, with the supervisor still alive. ready.json alone establishes backend readiness, not Vite readiness. Open only http://127.0.0.1:3011 through Antigravity. Read new ready fields apiOrigin,tenant,case,reader,owner; never reuse old browser_ready.json. New owner is an email; reader email is `<new reader UUID>@example.invalid`. Use the pinned synthetic test password only. Avoid preexisting browser sessions/tokens; verify the exact new tenant/case before mutations. Do not clear or reuse another app's stored session.

To request cleanup in a separate project-root terminal:

```powershell
python -B docs/reviews/probes/r105_root/r105_ui_runner.py stop --project-root .
```

This writes an idempotent cooperative stop request to the original supervisor. It does not kill by PID. Wait for original supervisor exit and cleanup.json; verify empty errors, actual child exits, same fixture stopped, protected comparison and complete stream metadata. status reports receipt presence only, not independent proof a process is live. If supervisor is hard-killed, keep the burnt ledger/resources and return BLOCKED for Root ownership review; never restart or broad-kill. Preserve/export the entire new runtime receipt/log directory with hashes to docs/reviews/probes/r105_claude/; all browser captures/tool transcripts also go there. Synthetic DSN/password is redacted in runner command logs; no real credentials/env are admitted.

The source stops normal UI work by1740s of backend life or2040s of campaign time, then performs owned cleanup;30min/35min remain ceilings, not a fresh retry allowance. Deadline/cleanup behavior is source/injected-test evidence until a real campaign runs. Cooperative cleanup cannot guarantee recovery from catastrophic supervisor termination.

Root disposition for old R104 diagnostic tab: preserve untouched, no reuse, no closure claim. Its last recorded ACTIVE state remains historical/unverified live. It is not one of the new R105 product tabs; never close it or a user tab as incidental cleanup. This is a no-effect disposition, not another prerequisite browser replay.

## Prepared gate-to-role graph for future activation

NOT_CLOSEABLE above remains until actual immutable seed/base/continuity/active gates are settled. This table preassigns every mutation/receipt so Root does not hand predictable dispatcher gaps to the worker. The future formal packet changes closeability only after the missing activation authority is real.

| gateId | mustPassBy | repairOwner | repairPhase | mutationSurface | topology | commitOwner | commitPhase | dependsOn |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| authorization_review | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R105 immutable seed source-review receipt formal order and record | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | NONE |
| pre_dispatch_gate | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R105 order continuity catalog and exact source pins | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | authorization_review |
| dispatch_continuity | IMPLEMENTATION | session-sync-steward | IMPLEMENTATION | R105 active state handoff and front marker | BOUNDED_PATH_FAMILIES | session-sync-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| focused_checker_tests | WORKER_RETURN | implementation-worker | WORKER_RETURN | R105 reservations readiness B01-B10 actual browser receipts and owned cleanup | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| adif_integrity | WORKER_RETURN | implementation-worker | WORKER_RETURN | R105 raw failures ledger manifests limits and all NOT_RUN branches | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests |
| pre_implementation_autorun | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | R105 effective admission and operator acknowledgment | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| worker_return_fast | WORKER_RETURN | implementation-worker | WORKER_RETURN | R105 formal packet worker return and evidence | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests,adif_integrity,pre_implementation_autorun |
| reviewer_fast | REVIEW | reviewer | REVIEW | R105 source-bound historical UI evidence and limits | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | worker_return_fast |
| pre_commit | PRE_MATERIAL_COMMIT | reviewer | PRE_MATERIAL_COMMIT | reviewed evidence continuity catalog and changed set | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | reviewer_fast |
| terminal_completion_review | POST_MATERIAL | reviewer | POST_MATERIAL | exact material commit and bounded disposition | EXACT_PATHS | commit-steward | CONTINUITY_COMMIT | pre_commit |
| continuity | CONTINUITY_COMMIT | session-sync-steward | CONTINUITY_COMMIT | memory state handoff implementation catalog | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | terminal_completion_review |
| committed_range_closure | POST_MATERIAL_CLOSURE | reviewer | POST_MATERIAL_CLOSURE | exact local range with pinned profile | EXACT_PATHS | session-sync-steward | CONTINUITY_COMMIT | continuity |
