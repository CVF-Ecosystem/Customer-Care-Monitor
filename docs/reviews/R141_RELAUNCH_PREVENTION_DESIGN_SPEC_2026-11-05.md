# R141 physical-relaunch prevention: owner-held Chrome with connect-only native MCP

Status: DESIGN_SPEC_REPAIR1_PENDING_INDEPENDENT_REREVIEW
Tranche: CCMAI-RUNTIME-141
Risk: R2
Worker: Cloud ORCHESTRATOR acting as DESIGN_OWNER / SPEC_AUTHOR (documentary only)
Reviewer: Independent Opus design-review subagent (pending)
Packet: `docs/reviews/R141_RELAUNCH_PREVENTION_DESIGN_PACKET_2026-11-05.md`
Seed: `CVF_SESSION/authority/CCMAI-RUNTIME-141.json` (commit `19018e2d0c57f577f2cd309f7fa2611b0bc3a301`)

This is a design and specification only. It proposes the reopen prerequisite named by `docs/reviews/R139_CAUSAL_CHAIN_ASSESSMENT_2026-11-05.md`: "a named supported native primitive or bounded technical alternative [that] proves how it prevents forbidden relaunch before action". It does **not** reopen the R139 repair audit, release implementation, start any process or accept any M3 criterion. Local decides reopen after its own review.

## 1. Problem restated from the controlling records

- `R139_SDK_P01_B01_DESIGN_SPEC_2026-10-30.md` set a hard hold. Native `browser.ensureBrowserLaunched` physically launches a replacement Chrome after connection loss, and `index` rebuilds the context. Samples and reconnect notices only detect this after relaunch or after an action.
- The causal assessment marks the integrated lifecycle NOT_EXECUTABLE until pre-action prevention is source-backed. Any alternative "may not silently weaken the physical/no-relaunch requirement".
- The locked requirement this design must preserve: no hidden physical Chrome replacement may occur before or during a product action, and the same held context must persist from provisional readiness through product work to terminal evidence.

## 2. Source identity

`docs/reviews/probes/r141_relaunch_prevention_20261105/source_verification.json` records the identity check.

- The public `chrome-devtools-mcp@1.9.0` npm tarball matches the registry integrity (sha512) and shasum.
- All **15/15** native files that R139 recorded from the operator's Windows Antigravity install (S06–S17, S28, S29, S34) are byte-identical to the tarball.
- Five further files come from the same verified tarball but were not in the R139 inventory: `config/browser-options.js`, `ToolHandler.js`, `tools/ToolDefinition.js`, `McpPage.js` and `package.json`. Their hashes are listed in the probe. **Local must confirm them on Windows.** Repair round 1 (F6) corrected this count from four to five.
- The package was never installed, imported or executed, and no package bytes are committed.

## 3. Native source facts (static; paths under `build/src/`)

| ID | Fact | Citation |
|---|---|---|
| N1 | The server chooses the browser path per call. If `browserUrl`, `wsEndpoint` or `autoConnect` is set, it calls `ensureBrowserConnected`. Otherwise it calls `ensureBrowserLaunched`. | `index.js:159-173` |
| N2 | `ensureBrowserConnected` returns the existing browser if still connected. Otherwise it only calls `puppeteer.connect` to the configured endpoint and throws `Could not connect to Chrome` on failure. There is no launch fallback. | `browser.js:24-105` |
| N3 | The only `launch` call site is `ensureBrowserLaunched` (`browser.js:198-207`, calling `launch` at `:122-197`). A grep of the package, excluding bundled puppeteer, finds its only caller at `index.js:173`. | `browser.js`, `index.js` |
| N4 | `--wsEndpoint` (alias `-w`) is a supported option. It must use the `ws:`/`wss:` protocol and conflicts with `--browserUrl`. Its example form is `ws://127.0.0.1:9222/devtools/browser/<id>`. | `config/browser-options.js:37-60` |
| N5 | Page IDs come from a process-wide counter that is never reset outside a test helper. The source comment says an ID issued before a reconnect "fails to resolve instead of hitting an unrelated page". | `McpContext.js:19-23,130-132,394` |
| N6 | When the browser object changes, the context is disposed and rebuilt with `reconnected: true`. The **next** response carries `structuredContent.reconnected = true` and a "browser was restarted or reconnected" note. This is detection after the fact. The notice is emitted **once per context, to whichever caller comes first**, and can be lost if response formatting throws (`ToolHandler.js:183,225,238-252`). Another actor may therefore never see it. | `index.js:190-203`, `McpResponse.js:505-507` |
| N7 | `pageIdRouting` defaults to `true`. With it on and not `--slim`, page-scoped tools add `pageId: zod.number()` to their schema. The handler then resolves the page through `getPageById`. If `pageId` is absent, the handler falls back to the selected page (`ToolHandler.js:190-196`). The reviewer cites the bundled MCP SDK as rejecting a missing required `pageId` before the handler runs (`third_party/index.js:39070,39100-39111`). Exception: `evaluate_script` declares `pageId` optional and offers a `serviceWorkerId` bypass (`tools/script.js:60-108`). | `config/mcp-options.js:39-43`, `ToolHandler.js:134-140,190-196`, `tools/ToolDefinition.js:16-34` |
| N8 | The foreground daemon forwards its argv unchanged to the MCP server child (`args: [INDEX_SCRIPT_PATH, ...mcpServerArgs]`), so N1–N7 apply on the R139-selected daemon route. | `daemon/daemon.js:87,96` |
| N9 | In connected mode `closeBrowser` only disconnects; it does not close Chrome. Chrome lifetime therefore stays with whoever launched it. | `browser.js:215-232` |

## 4. Selected design: owner-held Chrome, connect-only MCP (OHC)

**Principle.** Take the native launch capability away from the MCP server entirely, using its own supported connect-only mode (N1–N3). Give the one original campaign owner the single, non-renewable Chrome launch and the original process handle.

### 4.1 Owner launch (once per campaign)

1. The campaign owner (the existing supervisor role) launches `chrome.exe` directly, exactly once, with:
   - `--remote-debugging-port=0`
   - `--user-data-dir=<fresh owned directory>`
   - headless and no-first-run flags fixed by the later execution admission
2. The owner keeps the original process handle, PID, creation time, executable and argv.
3. The owner reads `<user-data-dir>/DevToolsActivePort` (port and `/devtools/browser/<guid>` path) and records the browser GUID and the resulting `ws://127.0.0.1:<port>/devtools/browser/<guid>`.
4. The owner never launches Chrome again in the campaign. If Chrome exits for any reason, the campaign goes to CLEANUP and then TERMINAL with HOLD. There is no relaunch, retry or deadline renewal.

### 4.2 MCP configuration (closed set)

- **Allowed:** start the R139-selected foreground daemon with `--wsEndpoint=<recorded endpoint>` plus the already-reviewed non-launch options (`--viaCli`, `--no-usage-statistics`, `--no-performance-crux`, `--filesystemRoot=<owned root>`, the session-ID env vars).
- **Forbidden:**
  - `--browserUrl` (it resolves whatever Chrome answers on a port and does not pin the GUID)
  - `--autoConnect`, `--channel`, `--userDataDir`
  - `--isolated`, `--executablePath`, `--headless` (launch-only)
  - `--no-page-id-routing`, `--slim`
- A preflight must reject any argv outside this closed set before the daemon starts.

### 4.3 Why this prevents physical relaunch before action

| Threat | Prevention | Basis |
|---|---|---|
| MCP relaunches Chrome after connection loss | The launch branch is unreachable while `wsEndpoint` is set. A lost connection makes the next tool call try `puppeteer.connect` to the same endpoint; if Chrome is gone, the call fails before any page action. | N1, N2, N3 (source-proven) |
| A different Chrome answers on the same port (port reuse) | The endpoint path carries the original browser GUID. A new Chrome instance has a different GUID, so the connect to the old path fails. | **A1 (assumption, Chrome-side, not in this package)** |
| The owner relaunches | The owner contract allows exactly one launch and treats a Chrome exit as terminal HOLD. | Contract (to be enforced in a later implementation and checked by review) |
| A stale page ID targets a page in a replaced context | Page-scoped tools require `pageId` and resolve it with `getPageById`. IDs are process-wide monotonic, so a pre-reconnect ID fails to resolve. | N5, N7 (source); **A2** for schema enforcement |
| A transient WebSocket drop reconnects to the **same** Chrome | Not a physical relaunch, but the context is replaced (N6). Page-scoped actions with old IDs fail closed (N5, N7). Any response carrying `reconnected: true` ends the product window: no further product action, then CLEANUP and TERMINAL HOLD. | N5, N6, N7 |

**Residual, stated honestly:**
- After a silent same-Chrome reconnect, a non-page-scoped tool such as `new_page` or `list_pages` can still execute once before the notice is seen.
- That call runs in the same physical Chrome, so the no-relaunch requirement holds.
- It does breach the same-held-context requirement for that one call, and it is detected in that same response.
- The design therefore requires the product action sequence to carry a `pageId` on every page-mutating call. The only non-page-scoped calls are `new_page` (once, at the start of the window) and `list_pages`.
- Any `reconnected: true` makes the whole window HOLD and its evidence non-acceptable. The window is not "repaired".

### 4.5 Repair round 1: binding amendments (from independent review F1–F5)

These amendments are part of the design. Where they conflict with §4.1–§4.3 or §5, they take precedence.

**R-F2. Single daemon start; no auto-start route (fixes no-relaunch).**
- The native CLI auto-starts a daemon with its own serialized options when none is running for the session (`bin/chrome-devtools.js:187-194`). That daemon would not carry `--wsEndpoint` and would reach the launch branch (`index.js:173`).
- Therefore the **owner alone** starts exactly one foreground daemon process: `node <daemon.js>` with the closed-set argv. The owner holds its original handle.
- Actors issue commands **only** through the exported native `sendCommand`. It throws `Daemon is not running.` and never starts a daemon (`daemon/client.js:105-109`).
- The native CLI binary (`bin/chrome-devtools.js`), its `start` command and any `startDaemon`/ensure helper are forbidden for the whole campaign.
- The session-ID namespace is exclusively claimed by the owner.
- A daemon or MCP-child exit, or a `Daemon is not running` error, is HOLD. There is no restart.
- Preflight before OWNER_HELD and an observation at TERMINAL must both show:
  - no other daemon socket for the claimed session ID;
  - no Chrome process using the owned user-data-dir other than the owner's PID.

**R-F1. Pinned page-ID set; owner sees every response (fixes same-held-context).**
- At PRELIMINARY_OBSERVED the owner records the pinned set: the page ID returned by the admitted `new_page` and confirmed by the independent observer's own native call.
- During the product window every page-scoped call must use an ID in the pinned set.
- All actor commands are relayed through the owner's `sendCommand` boundary, so the owner receives **every** response from every actor and checks it before releasing the next command.
- The owner stops new work and goes to HOLD on any of the following:
  - `reconnected: true` in any actor's response;
  - any `No page found` or other unknown-page error;
  - any page ID outside the pinned set appearing in any response;
  - any `list_pages` result that differs from the pinned set.
- Actors may not re-acquire IDs. `list_pages` is observation only, and its result must equal the pinned set.

**Residual (restated).** A silent same-Chrome reconnect can still allow one call to execute in the rebuilt context. With R-F1, that call can only be:
- a pinned-ID call, which fails because the old ID does not resolve (N5); or
- `list_pages`, which is read-only.

Its response, or the next one, triggers HOLD at the owner. No product mutation can succeed in a replaced context without the owner seeing a HOLD condition first. The physical no-relaunch requirement is unaffected.

**R-F3. Exact argv and environment match.**
- The preflight parses argv exactly and rejects anything not in the closed set. The package itself only logs unknown flags.
- `--wsEndpoint` must be present and non-empty, use the `ws:` scheme, have host `127.0.0.1`, use the recorded port, and have path `/devtools/browser/<recorded guid>`. An empty `--wsEndpoint=` would make N1's condition false and fall through to launch.
- `--browserUrl`, `--autoConnect` and every launch-only or routing-disabling flag listed in §4.2 are rejected.
- The environment is an allowlist: the two session/no-update/no-stats variables reviewed in R139 plus the OS minimum. `NODE_OPTIONS` and any other `NODE_*`/`npm_*` injection variables must be unset.

**R-F4. Explicit tool allowlist for the product window.**
- **Allowed:**
  - `new_page` (once, owner-relayed, before PRELIMINARY_OBSERVED);
  - `list_pages` (observation only);
  - with explicit pinned `pageId`: `navigate_page`, `take_snapshot`, `wait_for`, `take_screenshot`, `click`, `fill`, `fill_form`, `type_text`, `press_key`, `hover`, `handle_dialog`.
- **Forbidden:**
  - `evaluate_script` (optional `pageId`, `serviceWorkerId` bypass);
  - `select_page`, `close_page`, `resize_page`, `click_at`, `drag`, `upload_file`, `get_tab_id`;
  - all extension, devtools, lighthouse, performance, memory, network-mutation, emulation, screencast, pwa and webmcp tools.
- Closing the target is done by the owner at CLEANUP through the original Chrome handle, not by an actor tool.
- The later execution admission may narrow this list but must not widen it.

### 4.4 Assumptions that need a separately admitted check (not run here)

- **A1.** (Repair F5 scope note: if the owner's Chrome exits and a **non-Chrome** process binds the port, the connect also has to fail or produce no usable CDP session. The owner's Chrome-exit HOLD already covers this, because the owner observes the original-handle exit before any further product action.) Chrome refuses a WebSocket upgrade to `/devtools/browser/<guid>` when `<guid>` is not its own browser target ID. The connect then fails with no fallback.
  - Check: owner launches Chrome, records the GUID, then connects with a mutated GUID; expect failure.
  - Then: kill that Chrome, start a new one on the same port, connect with the old endpoint; expect failure.
- **A2.** The MCP SDK request validation enforces the registered `pageId: zod.number()` as required. A page-scoped call without `pageId` is then rejected, not routed to the selected page.
  - The reviewer found static support in the bundled SDK 1.30.0 (`third_party/index.js:39070,39100-39111`), so A2 is now low risk. It stays a check because it governs behaviour on the actual Windows runtime.
  - Check: one offline call without `pageId`; expect a validation error.
  - Under §4.5, `evaluate_script` is forbidden, so its optional `pageId` does not depend on A2.
- **A3.** The Windows install's copies of the **five** additional cited files equal the tarball hashes in the probe.
  - Check: a hash comparison on Windows, with no execution.

All three checks need Chrome or Windows. They belong to a future execution admission run by Local, not to this tranche. Until they pass, OHC is **DESIGNED_SOURCE_BACKED_PENDING_A1_A3**, not executable.

## 5. Successor integrated-lifecycle binding (proposal)

This binds OHC into the Root-selected lifecycle in the causal assessment. It refines that lifecycle and does not replace it.

| State | OHC-specific requirement |
|---|---|
| ADMITTED → OWNER_HELD | Preconditions: exact pins including the tarball identity, an exact argv/env preflight PASS (R-F3), a single-daemon/no-foreign-Chrome preflight (R-F2), and A1–A3 evidence accepted. The owner then performs its single Chrome launch, records the GUID endpoint and starts the one daemon with `--wsEndpoint`. A missing precondition means NO_START. |
| OWNER_HELD → PRELIMINARY_OBSERVED | Through the owner relay, the producer creates the admitted target with `new_page`. The independent observer issues its own native call against that page ID. The owner records the pinned page-ID set (R-F1). Both responses must lack `reconnected`. |
| PRELIMINARY_OBSERVED → PRODUCT_WINDOW | Only allowlisted tools (R-F4) are used, each with a pinned `pageId`. The owner checks every response for the R-F1 HOLD conditions before releasing the next command. The original clock stays as-is (+1020 s work cutoff). |
| Any state, on any R-F1 HOLD condition, a connect failure, a daemon/child exit or a Chrome exit | Stop new work and go to CLEANUP, then TERMINAL HOLD. Unknown provider outcomes keep their debit or hold. No retry, relaunch or renewal. |
| CLEANUP → TERMINAL | Owner closes Chrome through its original handle and waits on it within the +1200 s reserve. Daemon cleanup is exact-owned. The closing evidence is the original-handle wait plus the recorded GUID/PID. |

## 5a. Repair history

Independent review round 1 (`docs/reviews/R141_RELAUNCH_PREVENTION_INDEPENDENT_REVIEW_2026-11-05.md`) returned CHANGES_REQUIRED.

- **F1 and F2 (blocking):** addressed by R-F1 and R-F2.
- **F3–F5 (non-blocking):** addressed by R-F3, R-F4 and the A1 scope note.
- **F6 (citations and file count):** corrected in §2, N6 and A3.
- **A2:** narrowed with the reviewer's SDK citation.

This is the single consolidated repair round the seed allows.

## 6. Options considered

| Option | Disposition |
|---|---|
| Keep launch mode and detect relaunch by PID or reconnect samples | Rejected. The R139 design already showed this is detection after action, not prevention. |
| `--browserUrl` connect mode | Rejected. It resolves the current debugger on a port with no GUID pinning. |
| `--autoConnect` with `--userDataDir` (reads `DevToolsActivePort`) | Rejected. It re-reads the port file at every reconnect, so a replacement Chrome in that profile would be accepted. |
| Patch or wrap the native package to remove `ensureBrowserLaunched` | Rejected. It edits a vendor install and is not a supported primitive. |
| **OHC with `--wsEndpoint` (selected)** | Supported native option. Source-proven no-launch path. Prevention is pre-action, subject to A1–A3. |

## 7. What this does not do

- No implementation of the owner launcher, the argv preflight or a GUID recorder.
- No change to the R139 supervisor or harness.
- No runtime check of A1–A3.
- No edit to `docs/milestone-coverage-authority.json` or to the R139 repair-chain audit.
- No M3-05/M3-08 progress claim: they remain BLOCKED until Local, after review, records a reopen with an admitted next action.
- No AI-governance claim.
- Facebook/Zalo OA remain parked.

## 8. Proposed next moves (for Local, after review)

1. Local reviews this spec and the independent review, then confirms A3 by hashing the four files on Windows.
2. If accepted, Local records an updated repair-chain audit. Its disposition stays not-yet-implement, and it names A1/A2 as the discriminator for a finite VERIFY packet run on Windows with a fresh effect admission.
3. Only after A1/A2 pass may a source WORK_ORDER for the owner launcher and argv preflight be dispatched. Execution of the integrated lifecycle then needs its own admission.

## Complete milestone coverage

```cvf-milestone-coverage-json
{"schemaVersion": "cvf.milestoneCoverage@1.0.0", "authority": {"path": "docs/reviews/planning/CCMA_PRODUCT_DELIVERY_ROADMAP_2026-10-13.md", "heading": "## 7. M3 — Người duyệt, giao việc và kết quả xử lý", "sha256": "712817cd25936efd4fb13e0dd456ebe4ddceb5fccfd1cacb83042838724065a4"}, "milestone": {"id": "M3", "consumer": "Người duyệt và người xử lý ca trên một installation Pancake", "outcome": "Hoàn tất review, giao việc, kết quả và lịch sử qua browser/backend theo tiêu chí M3 đã duyệt.", "checkpoint": {"trigger": "R141 design spec and independent design review complete", "owner": "Cloud ORCHESTRATOR (operator-granted); Local agent decides reopen and merge", "decision": "Accept or reject OHC as the reopen prerequisite design; never reopen the R139 audit, release implementation or runtime from this packet."}}, "phase": "PLAN", "action": {"kind": "PLAN", "selected": [], "owner": "Cloud ORCHESTRATOR as DESIGN_OWNER/SPEC_AUTHOR; independent Opus design reviewer", "output": "docs/reviews/R141_RELAUNCH_PREVENTION_DESIGN_SPEC_2026-11-05.md and its independent review", "budgetMinutes": 120, "stop": "Stop on any need for runtime, Chrome/daemon start, product-source edit, repair-audit edit or network beyond the single approved fetch.", "next": "Local reviews the PR, confirms A3 on Windows, and decides whether to record an updated repair audit with an A1/A2 VERIFY discriminator."}, "rows": [{"id": "M3-01", "state": "IMPLEMENTED_UNPROVED", "gap": "Quy tắc ưu tiên và applicability current-source còn mở.", "owner": "Product Owner / Root", "destination": "M1-01 quyết định trigger/quy tắc chờ và ngoại lệ; Owner xử lý quy tắc queue còn mở, Root đối chiếu source applicability trước G3.", "evidence": [{"path": "docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md", "sha256": "cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"}]}, {"id": "M3-02", "state": "IMPLEMENTED_UNPROVED", "gap": "Quyền, reason và binding current-source chưa có acceptance đầy đủ.", "owner": "Root / Product Owner", "destination": "Root kiểm binding và trace; Owner chấp nhận process tại G3.", "evidence": [{"path": "docs/reviews/R131_M3_ACCEPTANCE_REVIEW_2026-10-19.md", "sha256": "ec78f8db455589e9bad8a35d6f757ebb0068de5e175853e841a3fd0188e4c31b"}, {"path": "docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md", "sha256": "cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"}]}, {"id": "M3-03", "state": "IMPLEMENTED_UNPROVED", "gap": "Applicability lineage và negative hiện hành còn cần đối chiếu.", "owner": "Root / Product Owner", "destination": "Root kiểm source; G1 stale/competing negative nếu source thay đổi hoặc lộ gap; Owner chấp nhận effect.", "evidence": [{"path": "docs/reviews/R131_M3_ACCEPTANCE_REVIEW_2026-10-19.md", "sha256": "ec78f8db455589e9bad8a35d6f757ebb0068de5e175853e841a3fd0188e4c31b"}, {"path": "docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md", "sha256": "cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"}]}, {"id": "M3-04", "state": "IMPLEMENTED_UNPROVED", "gap": "Role, SLA và demo current-source chưa hoàn tất.", "owner": "Product Owner / Root", "destination": "M1-02/04 quyết định role/SLA; G3 demo assignment, outcome và audit.", "evidence": [{"path": "docs/reviews/R131_M3_ACCEPTANCE_REVIEW_2026-10-19.md", "sha256": "ec78f8db455589e9bad8a35d6f757ebb0068de5e175853e841a3fd0188e4c31b"}]}, {"id": "M3-05", "state": "BLOCKED", "gap": "E01–E06 chưa có runtime proof; physical-relaunch và integrated same-context/original-clock đang HOLD. R141 proposes owner-held Chrome with connect-only --wsEndpoint as source-backed pre-action relaunch prevention, pending independent design review, Local reopen decision and Windows checks A1-A3.", "owner": "Root / Product Owner", "destination": "Cùng R139 G0–G3 sau causal-chain audit, thiết kế tích hợp và admission phù hợp; Owner chấp nhận user consequence.", "evidence": [{"path": "docs/reviews/R137_INDEPENDENT_SOURCE_REVIEW_2026-10-21.md", "sha256": "63cc0bc309a235c7d36384b7b78afa48acafea8267622d93d191780633071783"}, {"path": "docs/reviews/R139_GEMINI_NATIVE_BROWSER_CHECK_ROOT_REVIEW_2026-10-28.md", "sha256": "0f434528ae218886eca1c8c68efb68d3cab164cbad57b2a2b51af79f14135d47"}]}, {"id": "M3-06", "state": "IMPLEMENTED_UNPROVED", "gap": "R140 source/test REVIEW_PASS on branch, pending Local merge; full browser lifetime remains unproved.", "owner": "Root / Product Owner", "destination": "R140 source/test correction then Root review; actual browser proof remains R139 G2 after integration admission.", "evidence": [{"path": "docs/reviews/R137_INDEPENDENT_SOURCE_REVIEW_2026-10-21.md", "sha256": "63cc0bc309a235c7d36384b7b78afa48acafea8267622d93d191780633071783"}]}, {"id": "M3-07", "state": "UNKNOWN", "gap": "Phạm vi R101 hiện không gửi notification; applicability với source và scope hiện hành còn cần kiểm tra.", "owner": "Root", "destination": "G3 đối chiếu applicability của exclusion đã có; nếu scope đổi, áp dụng change control trước khi xét contract notification.", "evidence": [{"path": "docs/reviews/R131_M3_ACCEPTANCE_MATRIX_2026-10-19.md", "sha256": "7dabfc6a2a8bbf42a4447063e961a428849b0a020ae4c02589a7fb323510f9ac"}]}, {"id": "M3-08", "state": "BLOCKED", "gap": "Current-source browser/backend demo và disposition viewport/offline còn mở; physical-relaunch HARD_HOLD, integrated lifetime DESIGNED_NOT_EXECUTABLE; successor admission pending; relaunch prevention unresolved. R141 proposes owner-held Chrome with connect-only --wsEndpoint as source-backed pre-action relaunch prevention, pending independent design review, Local reopen decision and Windows checks A1-A3.", "owner": "Root / Product Owner", "destination": "Cùng R139 causal-chain audit trước thiết kế G2 proof và G3; Owner chấp nhận qualification viewport/raster/offline.", "evidence": [{"path": "docs/reviews/R139_GEMINI_NATIVE_BROWSER_CHECK_ROOT_REVIEW_2026-10-28.md", "sha256": "0f434528ae218886eca1c8c68efb68d3cab164cbad57b2a2b51af79f14135d47"}, {"path": "docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md", "sha256": "cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"}]}], "supportingWork": [], "conclusions": {"tranche": "PENDING", "trancheSummary": "R141 design/spec authored; independent review pending", "trancheEvidence": [], "product": "PENDING", "productSummary": "M3 not accepted; M3-05/M3-08 BLOCKED; R139 integration not reopened", "acceptance": [], "owner": "Local agent (merge/reopen); operator product decisions"}}
```

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
