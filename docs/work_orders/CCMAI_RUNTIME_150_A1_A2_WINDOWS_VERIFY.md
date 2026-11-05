# R150: finite Windows A1/A2 VERIFY for OHC

Status: DISPATCH_READY
workOrderStatusGrammar: cvf.workOrderStatusGrammar@1.0.0
Tranche: CCMAI-RUNTIME-150
Authority: `CVF_SESSION/authority/CCMAI-RUNTIME-150.json`
Parent: R149 FROZEN design/spec; R139 BUILD_BLOCKED
Risk: R2

Risk note: when admitted, the risk is R2 or higher, because it starts Chrome and the native daemon on the operator's machine.
Worker: Root/Local Windows VERIFY worker, after independent audit review and effect admission
Reviewer: independent R2 reviewer separate from Root audit author and Windows worker

This is the official Local work order. Its DISPATCH_READY status does not itself authorize a process start. Execution requires a successor R139 repair audit whose discriminator is §1 and whose admittedAction pins these exact bytes, an independent admission review, a matching repair registry pointer, a passing pre-dispatch gate, and a separate effect admission. Immediately before any process start, the worker must recapture the complete installed package tree under §2. Any mismatch means NO_START.

R149 supplies design/spec only; it does not admit A1/A2. R139 remains BUILD_BLOCKED.

## 1. Question

On the operator's Windows machine, with the A3-identified `chrome-devtools-mcp@1.9.0` install:
- **A1:** does Chrome refuse a DevTools WebSocket connection whose `/devtools/browser/<guid>` path does not match the running browser, including after the port is reused by a new Chrome?
- **A2:** does the native daemon, started with the R149 closed argv, reject a page-scoped tool call that omits `pageId` before any handler action, while accepting the same call with the admitted `pageId`?

## 2. Preconditions (all, same session)

1. §1 of `docs/reviews/R149_OHC_SUCCESSOR_CONTRACT_2026-11-05.md`: full-tree recapture shows exactly the relaxed A3 identity (`package.json` `592ef654…`, `README.md` absent, the other 346 equal, no extras). Otherwise NO_START.
2. Record Chrome version (`chrome.exe --version` run from the shell is the only allowed "launch" before the campaign; it starts no browser session), Node version, OS build and Antigravity version.
3. Environment exactly as in R149 §2.3, with a fresh session UUID and fresh owned temp, user-data and evidence roots.
4. No other `chrome-devtools-mcp` daemon for that session ID, and no Chrome process using the owned user-data-dir.

## 3. Procedure and budget

The ceiling is **45 active minutes**.

Counts allowed:
- **Chrome launches: at most 3.** A1 needs a fixed-port relaunch; this is a VERIFY campaign, not the product single-launch rule.
- **Daemon starts: at most 1.**
- **Native tool calls: at most 8.**
- **Raw WebSocket handshakes: at most 6.**

Pages: `about:blank` only. Network: localhost only. Provider, DB and product calls: zero.

The owner-side helper may use only Node's built-in `WebSocket` (Node ≥ 22) or PowerShell. No packages are installed.
Count each localhost `/json/version` endpoint-discovery query explicitly in the effect ledger. It stays inside the 45-minute ceiling and is distinct from native calls and raw WebSocket handshakes.

**A1**

- **A1-a, control.** Launch Chrome #1 with the R149 §2.1 flags but `--remote-debugging-port=<fixed free port P>`. Read `DevToolsActivePort` if present. If absent, query only `http://127.0.0.1:P/json/version` for `webSocketDebuggerUrl`; require its port to be P and its path to be `/devtools/browser/<guid>`. If neither method yields that endpoint, stop INCONCLUSIVE. The localhost query is included in the existing network, time and call budgets. Open a raw WebSocket to the exact endpoint.
  - Expect: handshake succeeds.
  - Then send `Browser.getVersion` and record the reply.
- **A1-b.** Open a raw WebSocket to `ws://127.0.0.1:P/devtools/browser/<guid with one hex digit changed>`.
  - Expect: handshake refused (HTTP 4xx or connection close before open).
- **A1-c.** Close Chrome #1 through its handle and wait on it. Launch Chrome #2 on the same port P. Record its new GUID using the same bounded endpoint-discovery rule as A1-a; require a different GUID. Open a raw WebSocket to Chrome #1's old endpoint.
  - Expect: refused.
  - Then connect to Chrome #2's endpoint. Expect: success (control).
- **A1-d (optional, only if A1-c is inconclusive).** With no Chrome on P, connect to the old endpoint.
  - Expect: connection error.

**A2** (uses Chrome #2, or Chrome #3 if A1-c was skipped)

- **A2-a.** Start the daemon exactly as in R149 §2.2 with `--wsEndpoint` set to Chrome #2's endpoint. Call `status` through the native `sendCommand`.
  - Expect: `args` equals the admitted argv; `version` is `1.9.0`.
- **A2-b.** Producer `new_page` with `about:blank`. Observer `list_pages`. Record the full inventory (N-R1).
- **A2-c.** Call `take_snapshot` **without** `pageId`.
  - Expect: an SDK input-validation error before handler entry. Retain the raw error envelope; the following inventory alone cannot prove that this read-only handler was never entered.
  - Expect: no change to the page or the inventory, verified by a following observer `list_pages`.
- **A2-d.** Call `take_snapshot` with the admitted `pageId`.
  - Expect: success (control).
- **A2-e (optional).** Call `take_snapshot` with a non-existent `pageId` (max + 1000).
  - Expect: an unknown-page error.

**Cleanup**

1. Stop the daemon through its handle and wait on it.
2. Close the last Chrome through its handle and wait on it.
3. Confirm that no Chrome process uses the owned user-data-dir, no daemon PID file remains for the session, and the daemon-spawned MCP child has exited.

## 4. Evidence to retain (secret-free)

- The A3 recapture JSON and its comparison against the manifest.
- Versions.
- Exact argv and the environment key list (values are allowed only for the three admitted variables).
- Chrome GUIDs and ports.
- For each WebSocket handshake: the target URL and the outcome (status code or close reason).
- For each native call: the raw `sendCommand` envelope. For A2-c retain the SDK input-validation error envelope and following observer inventory.
- PIDs, creation times and wait results.
- Timings and counters.

Do not record browser profile contents.

## 5. Stop conditions

Stop on any of the following:
- the A3 precondition fails;
- an environment or argv mismatch;
- an unexpected Chrome or daemon process;
- any call outside §3;
- a result that cannot be classified;
- budget exhaustion;
- any need for network beyond localhost, an install or a product page.

Return the evidence with UNKNOWNs. Never retry with relaxed settings.

## 6. Outcome classes

| Class | Meaning |
|---|---|
| `A1_A2_PASS` | A1-b and A1-c refused, A1-a and A1-c controls succeeded, A2-c rejected with no side effect, A2-d succeeded |
| `A1_FAIL` | A non-matching GUID connected |
| `A2_FAIL` | A call without `pageId` executed or changed state |
| `INCONCLUSIVE` | Anything else, with exact UNKNOWNs |

A pass only removes A1/A2 as blockers for considering an integrated-lifecycle admission. It is not physical no-relaunch proof for the product campaign, and not an M3, runtime-governance or product acceptance.

## Gate-To-Role Closeability Contract

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
returnTimeRecheck: REQUIRED_BEFORE_REPAIR
implementationTopologyPolicy: BOUNDED_PATH_FAMILIES
foreseeableFileSplitDisposition: COVERED_BY_BOUNDED_PATH_FAMILY

| gateId | mustPassBy | repairOwner | repairPhase | mutationSurface | topology | commitOwner | commitPhase | dependsOn |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| authorization_review | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R150 seed, successor audit and exact work order | EXACT_PATHS | commit-steward | DISPATCH_COMMIT | NONE |
| pre_dispatch_gate | PRE_DISPATCH | orchestrator | PRE_DISPATCH | work order, coverage pointer and continuity | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | authorization_review |
| effect_admission | IMPLEMENTATION | orchestrator | IMPLEMENTATION | separate Chrome/daemon effect admission and counters | EXACT_PATHS | commit-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| dispatch_continuity | IMPLEMENTATION | session-sync-steward | IMPLEMENTATION | CVF_SESSION/handoffs/AGENT_HANDOFF_R150_A1_A2_WINDOWS_VERIFY_2026-11-05.md material-SHA marker | EXACT_PATHS | session-sync-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| a3_immediate_recapture | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | full-tree hash comparison and NO_START decision | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | effect_admission,dispatch_continuity |
| focused_checker_tests | WORKER_RETURN | implementation-worker | WORKER_RETURN | A1/A2 handshakes, native envelopes, process waits | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity,a3_immediate_recapture |
| adif_integrity | WORKER_RETURN | implementation-worker | WORKER_RETURN | R139 historical failure and current applicability limits | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | focused_checker_tests |
| pre_implementation_autorun | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | pre-BUILD rehydration and role acknowledgment | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| worker_return_fast | WORKER_RETURN | implementation-worker | WORKER_RETURN | exact outcome, counts, cleanup and UNKNOWNs | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | focused_checker_tests,adif_integrity,pre_implementation_autorun |
| reviewer_fast | REVIEW | reviewer | REVIEW | independent source and result review | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | worker_return_fast |
| pre_commit | PRE_MATERIAL_COMMIT | reviewer | PRE_MATERIAL_COMMIT | reviewed evidence, scope, continuity and catalog | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | reviewer_fast |
| terminal_completion_review | POST_MATERIAL | reviewer | POST_MATERIAL | A1/A2 outcome and retained R139 hold | EXACT_PATHS | commit-steward | CONTINUITY_COMMIT | pre_commit |
| continuity | CONTINUITY_COMMIT | session-sync-steward | CONTINUITY_COMMIT | state, handoff, status and catalog | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | terminal_completion_review |
| committed_range_closure | POST_MATERIAL_CLOSURE | reviewer | POST_MATERIAL_CLOSURE | committed R150 range and static gates | EXACT_PATHS | session-sync-steward | CONTINUITY_COMMIT | continuity |

## Complete milestone coverage

The machine-readable block is the full M3 projection. This is finite VERIFY of selected M3-05/M3-08 prerequisites, not IMPLEMENT or product acceptance.

```cvf-milestone-coverage-json
{"schemaVersion":"cvf.milestoneCoverage@1.0.0","authority":{"path":"docs/reviews/planning/CCMA_PRODUCT_DELIVERY_ROADMAP_2026-10-13.md","heading":"## 7. M3 — Người duyệt, giao việc và kết quả xử lý","sha256":"712817cd25936efd4fb13e0dd456ebe4ddceb5fccfd1cacb83042838724065a4"},"milestone":{"id":"M3","consumer":"Người duyệt và người xử lý ca trên một installation Pancake","outcome":"Hoàn tất review, giao việc, kết quả và lịch sử qua browser/backend theo tiêu chí M3 đã duyệt.","checkpoint":{"trigger":"R150 A1/A2 Windows VERIFY return or 45 active minute ceiling","owner":"Root/Local ORCHESTRATOR and independent R2 reviewer","decision":"Classify A1_A2_PASS, A1_FAIL, A2_FAIL or INCONCLUSIVE and independently review the exact evidence; integrated lifecycle still needs separate effect admission."}},"phase":"DISPATCH","action":{"kind":"VERIFY","selected":["M3-05","M3-08"],"owner":"Root/Local Windows VERIFY worker","output":"docs/reviews/R150_A1_A2_WINDOWS_VERIFY_RETURN_2026-11-05.md","budgetMinutes":45,"stop":"Stop on A3 drift, unexpected process, argv/environment violation, out-of-scope call, unclear result, budget or network boundary; no retry with relaxed settings.","next":"Return raw bounded evidence to a separate independent R2 reviewer; R139 remains BUILD_BLOCKED and integrated lifecycle needs its own effect admission.","question":"On the operator's Windows machine, with the A3-identified `chrome-devtools-mcp@1.9.0` install:\n- **A1:** does Chrome refuse a DevTools WebSocket connection whose `/devtools/browser/<guid>` path does not match the running browser, including after the port is reused by a new Chrome?\n- **A2:** does the native daemon, started with the R149 closed argv, reject a page-scoped tool call that omits `pageId` before any handler action, while accepting the same call with the admitted `pageId`?","sourceBoundary":"Only the immediately recaptured installed chrome-devtools-mcp@1.9.0 tree compared with the R144 npm manifest, the R149 frozen contract, owned Chrome about:blank and localhost native daemon. No provider, DB or product call."},"rows":[{"id":"M3-01","state":"IMPLEMENTED_UNPROVED","gap":"Quy tắc ưu tiên và applicability current-source còn mở.","owner":"Product Owner / Root","destination":"M1-01 quyết định trigger/quy tắc chờ và ngoại lệ; Owner xử lý quy tắc queue còn mở, Root đối chiếu source applicability trước G3.","evidence":[{"path":"docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md","sha256":"cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"}]},{"id":"M3-02","state":"IMPLEMENTED_UNPROVED","gap":"Quyền, reason và binding current-source chưa có acceptance đầy đủ.","owner":"Root / Product Owner","destination":"Root kiểm binding và trace; Owner chấp nhận process tại G3.","evidence":[{"path":"docs/reviews/R131_M3_ACCEPTANCE_REVIEW_2026-10-19.md","sha256":"ec78f8db455589e9bad8a35d6f757ebb0068de5e175853e841a3fd0188e4c31b"},{"path":"docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md","sha256":"cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"}]},{"id":"M3-03","state":"IMPLEMENTED_UNPROVED","gap":"Applicability lineage và negative hiện hành còn cần đối chiếu.","owner":"Root / Product Owner","destination":"Root kiểm source; G1 stale/competing negative nếu source thay đổi hoặc lộ gap; Owner chấp nhận effect.","evidence":[{"path":"docs/reviews/R131_M3_ACCEPTANCE_REVIEW_2026-10-19.md","sha256":"ec78f8db455589e9bad8a35d6f757ebb0068de5e175853e841a3fd0188e4c31b"},{"path":"docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md","sha256":"cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"}]},{"id":"M3-04","state":"IMPLEMENTED_UNPROVED","gap":"Role, SLA và demo current-source chưa hoàn tất.","owner":"Product Owner / Root","destination":"M1-02/04 quyết định role/SLA; G3 demo assignment, outcome và audit.","evidence":[{"path":"docs/reviews/R131_M3_ACCEPTANCE_REVIEW_2026-10-19.md","sha256":"ec78f8db455589e9bad8a35d6f757ebb0068de5e175853e841a3fd0188e4c31b"}]},{"id":"M3-05","state":"BLOCKED","gap":"R139 integration remains BUILD_BLOCKED. The operator-relaxed A3 identity is a same-session precondition; A1/A2 Windows behavior is unproved until this finite VERIFY returns and receives independent review.","owner":"Root / Product Owner","destination":"R150 returns A1/A2 evidence to Root; only an independent PASS may permit consideration of a separately admitted integrated lifecycle. M3 remains blocked.","evidence":[{"path":"docs/reviews/R137_INDEPENDENT_SOURCE_REVIEW_2026-10-21.md","sha256":"63cc0bc309a235c7d36384b7b78afa48acafea8267622d93d191780633071783"},{"path":"docs/reviews/R139_GEMINI_NATIVE_BROWSER_CHECK_ROOT_REVIEW_2026-10-28.md","sha256":"0f434528ae218886eca1c8c68efb68d3cab164cbad57b2a2b51af79f14135d47"},{"path":"docs/reviews/R142_LOCAL_STACK_MERGE_REVIEW_AND_SUCCESSOR_CONTRACT_2026-11-05.md","sha256":"570487e6720040efcee90e4d370f7be6ab04f0851b94e989ec3ecbd8a39f3435"}]},{"id":"M3-06","state":"IMPLEMENTED_UNPROVED","gap":"R140 selection-read source/component repair FROZEN; browser lifetime and channels-store applicability remain unproved.","owner":"Root / Product Owner","destination":"Root keeps browser lifetime for admitted R139 G2 and separately assesses channels-store applicability against M3 queue/filter.","evidence":[{"path":"docs/reviews/R137_INDEPENDENT_SOURCE_REVIEW_2026-10-21.md","sha256":"63cc0bc309a235c7d36384b7b78afa48acafea8267622d93d191780633071783"},{"path":"docs/reviews/R142_LOCAL_STACK_MERGE_REVIEW_AND_SUCCESSOR_CONTRACT_2026-11-05.md","sha256":"570487e6720040efcee90e4d370f7be6ab04f0851b94e989ec3ecbd8a39f3435"}]},{"id":"M3-07","state":"UNKNOWN","gap":"Phạm vi R101 hiện không gửi notification; applicability với source và scope hiện hành còn cần kiểm tra.","owner":"Root","destination":"G3 đối chiếu applicability của exclusion đã có; nếu scope đổi, áp dụng change control trước khi xét contract notification.","evidence":[{"path":"docs/reviews/R131_M3_ACCEPTANCE_MATRIX_2026-10-19.md","sha256":"7dabfc6a2a8bbf42a4447063e961a428849b0a020ae4c02589a7fb323510f9ac"}]},{"id":"M3-08","state":"BLOCKED","gap":"R139 integration remains BUILD_BLOCKED. The operator-relaxed A3 identity is a same-session precondition; A1/A2 Windows behavior is unproved until this finite VERIFY returns and receives independent review.","owner":"Root / Product Owner","destination":"R150 returns A1/A2 evidence to Root; only an independent PASS may permit consideration of a separately admitted integrated lifecycle. M3 remains blocked.","evidence":[{"path":"docs/reviews/R139_GEMINI_NATIVE_BROWSER_CHECK_ROOT_REVIEW_2026-10-28.md","sha256":"0f434528ae218886eca1c8c68efb68d3cab164cbad57b2a2b51af79f14135d47"},{"path":"docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md","sha256":"cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"},{"path":"docs/reviews/R142_LOCAL_STACK_MERGE_REVIEW_AND_SUCCESSOR_CONTRACT_2026-11-05.md","sha256":"570487e6720040efcee90e4d370f7be6ab04f0851b94e989ec3ecbd8a39f3435"}]}],"supportingWork":[],"conclusions":{"tranche":"PENDING","trancheSummary":"R150 A1/A2 finite Windows VERIFY is prepared; review, registry, pre-dispatch and effect admission precede execution.","trancheEvidence":[],"product":"PENDING","productSummary":"R139 BUILD_BLOCKED; M3-05/M3-08 BLOCKED; no product acceptance.","acceptance":[],"owner":"Root/Local technical closer; Product Owner retains M3 business acceptance"}}
```

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
