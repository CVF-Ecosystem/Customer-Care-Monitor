# R149 successor contract: owner-held Chrome (OHC) for the R139 integrated lifecycle

Status: FROZEN_DESIGN_ONLY
Tranche: CCMAI-RUNTIME-149
Risk: R2
Worker: Cloud ORCHESTRATOR acting as SPEC_AUTHOR (documentary only)
Reviewer: independent `/root/r148_independent_review` after Root/Local two-file documentary repair
Seed: `CVF_SESSION/authority/CCMAI-RUNTIME-149.json` (commit `abe76f66cb46978cc242bc1a3fdb6c0fb96a4577`)

This contract consolidates every binding requirement that an executable OHC campaign must satisfy. It does **not** admit or run anything:
- A1/A2 stay unrun.
- R139 stays BUILD_BLOCKED.
- Local alone records any repair-audit successor and effect admission.

The draft Windows VERIFY packet is `docs/reviews/R149_A1_A2_WINDOWS_VERIFY_WORK_ORDER_DRAFT_2026-11-05.md`.

## 0. Controlling inputs and precedence

Where two sources conflict, the later item wins.

1. `docs/reviews/R139_CAUSAL_CHAIN_ASSESSMENT_2026-11-05.md`: Root-selected lifecycle and reopen prerequisite.
2. `docs/reviews/R141_RELAUNCH_PREVENTION_DESIGN_SPEC_2026-11-05.md`: OHC design, N1–N9, and §4.5 amendments R-F1 to R-F4.
3. `docs/reviews/R141_RELAUNCH_PREVENTION_INDEPENDENT_REVIEW_2026-11-05.md`: rounds 1 and 2, including N-R1 to N-R3.
4. `docs/reviews/R142_LOCAL_STACK_MERGE_REVIEW_AND_SUCCESSOR_CONTRACT_2026-11-05.md` and `docs/reviews/R148_ROOT_R144_A3_DISPOSITION_2026-11-05.md`: the seven Local conditions.
5. Operator decision of 2026-11-05 (cloud session): relaxed A3 full-tree condition (§1).
6. This contract.

Native citations below refer to `build/src/` in the byte-verified `chrome-devtools-mcp@1.9.0`. That is the R141 tarball: 15/15 files matched in R141, and 346/348 matched the Windows install in R148.

## 1. A3 package identity (relaxed condition)

The installation is acceptable **only** if a same-session capture, taken immediately before any execution, shows exactly the following against `docs/reviews/probes/r144_a3_package_metadata_20261105/root_npm_full_tree_manifest.json` (348 files):

- `package.json` raw SHA-256 equals `592ef654a308edcba5fb3c2b302173650096a9487e4120b96e7adbe830429b9f`, as explained by R144: npm minus `bugs`/`scripts`, re-serialized.
- `README.md` is absent. This is the operator-relaxed allowance.
- Every other one of the 346 paths is present, with equal byte length and SHA-256.
- There are no extra paths.

Any other difference means NO_START. An Antigravity update between capture and execution is covered by the immediate recapture.

## 2. Closed process configuration

### 2.1 Chrome (owner, once per campaign)

The owner launches `chrome.exe` directly, exactly once, with these flags:
- `--remote-debugging-port=0`
- `--user-data-dir=<fresh owned dir>`
- `--headless=new`
- `--no-first-run`
- `--no-default-browser-check`
- initial URL `about:blank`

The owner:
- keeps the original handle, PID, creation time, executable path and argv;
- reads `<user-data-dir>/DevToolsActivePort` to get the port and the `/devtools/browser/<guid>` path;
- records the endpoint `ws://127.0.0.1:<port>/devtools/browser/<guid>`.

### 2.2 Daemon (owner, once per campaign; R-F2)

The owner starts `node <pkg>/build/src/daemon/daemon.js` as a foreground child and holds its handle. The argv is **exactly**:

- `--viaCli`
- `--wsEndpoint=<recorded endpoint>`
- `--no-usage-statistics`
- `--no-performance-crux`
- `--filesystemRoot=<fresh owned evidence root>`

**F6: N4 conflicts.** The pre-start check rejects any other flag. The relevant native conflicts are:
- `browserUrl` conflicts with `wsEndpoint` (`config/browser-options.js:23,41`);
- `wsHeaders` implies `wsEndpoint` (`config/browser-options.js:64`);
- `allowedUrlPattern` conflicts with `blockedUrlPattern` (`config/mcp-options.js:109,114`).

`--browserUrl`, `--autoConnect`, `--channel`, `--userDataDir`, `--isolated`, `--executablePath`, `--headless`, `--slim`, `--no-page-id-routing` and `--wsHeaders` are forbidden. An empty `--wsEndpoint=` is forbidden, because it would fall through to launch (N1).

### 2.3 Environment allowlist (F3)

The package itself, outside its bundled `third_party`, reads only these environment variables:
- `CHROME_DEVTOOLS_MCP_SESSION_ID` (`daemon/daemon.js:16`)
- `CHROME_DEVTOOLS_MCP_NO_UPDATE_CHECKS` (`utils/check-for-updates.js:23`)
- `CHROME_DEVTOOLS_MCP_NO_USAGE_STATISTICS` (`config/mcp-options.js`)
- `CHROME_DEVTOOLS_MCP_CRASH_ON_UNCAUGHT` (`bin/chrome-devtools-mcp-main.js:20`)
- `XDG_RUNTIME_DIR` (`daemon/utils.js:37,52`)
- `CI`
- `DISPLAY`

The daemon forwards its full environment to the MCP child (`daemon/daemon.js:96-98`).

Admitted environment for the owner and every actor:
- **Set:** `CHROME_DEVTOOLS_MCP_SESSION_ID=<fresh exclusively claimed UUID>`, `CHROME_DEVTOOLS_MCP_NO_UPDATE_CHECKS=1`, `CHROME_DEVTOOLS_MCP_NO_USAGE_STATISTICS=1`. These are the three R139 variables.
- **Must be unset:**
  - `CHROME_DEVTOOLS_MCP_CRASH_ON_UNCAUGHT`
  - `XDG_RUNTIME_DIR`, `XDG_ROOT`
  - `CI`, `DISPLAY`
  - `NODE_OPTIONS` and every other `NODE_*`
  - every `npm_*`
  - every `PUPPETEER_*`
  - `DEBUG`
- **Allowed:** the OS minimum, which is the R139 minimum-environment rule (`R139_SDK_P01_B01_DESIGN_SPEC_2026-10-30.md:63`).

**XDG resolution.**
- On Windows the daemon socket is a named pipe that includes the username and ignores `XDG_RUNTIME_DIR` (`daemon/utils.js:29-33`).
- However, `getRuntimeHome`, which gives the PID-file location, checks `XDG_RUNTIME_DIR` **before** the Windows fallback (`daemon/utils.js:45-62`). A set `XDG_RUNTIME_DIR` would therefore move the PID file and break single-daemon detection. That is why it must be unset.
- `XDG_ROOT` is not read by the package; it stays unset only to satisfy R139.

### 2.4 Post-start check (F3)

After the daemon is ready, the owner calls `status` through the native `sendCommand`. The result carries `args: mcpServerArgs`, along with `pid`, `version` and `socketPath` (`daemon/daemon.js:136-145`). The campaign requires:
- `args` equals the admitted argv exactly, in order;
- `version` equals `1.9.0`;
- `pid` equals the owner's daemon child PID.

Any difference means HOLD before PRELIMINARY_OBSERVED.

## 3. Actor protocol

- **N-R2 (supersedes the "owner relays" wording of R-F1):**
  - Each actor, producer or observer, makes its **own** original `sendCommand` call (`daemon/client.js:105-109`; it never starts a daemon).
  - The actor returns the unmodified response envelope and its terminal outcome to the owner.
  - The owner is a **sequencing gate**. It releases the next command only after checking the prior response against §4. The owner never originates, alters or replays an actor call.
- **R-F4 tool allowlist:** unchanged from R141 §4.5. `evaluate_script` stays forbidden.
- **N-R1:**
  - At PRELIMINARY_OBSERVED the owner records the **full** page inventory returned by the observer's own `list_pages`, including Chrome's initial tab.
  - Mutating tools may target only the single admitted page ID created by the producer's `new_page`.
  - A later `list_pages` result must equal the pinned inventory, except for URL/title changes on the admitted page.
  - Source boundary: `build/src/tools/pages.js:13-27,90-116` defines `list_pages` and `new_page`; the full inventory pin and single admitted mutation target are owner-side checks, not native MCP guarantees.
- **F5:** before releasing **each** command, the owner confirms that its original Chrome handle and daemon child are still alive. `build/src/daemon/client.js:105-109` checks daemon liveness before `sendCommand`, but does not check the original Chrome handle; that extra check belongs to the owner. Stated limit: a hostile process of the same user can act between the check and the command; this residual race is accepted and recorded, not prevented.

## 4. HOLD conditions (any one ends new work, then CLEANUP)

- `reconnected: true` in any response (N6), or any unknown-page or `No page found` error.
- A page ID outside the pinned inventory, an unexpected inventory change (N-R1), or a mutation targeting a page other than the admitted one.
- A connect failure, a `Daemon is not running` error, or the exit of Chrome, the daemon or the MCP child.
- `status` mismatch (§2.4), A3 mismatch (§1), or an argv/environment violation (§2).
- **N-R3:** a Chrome process using the owned user-data-dir that is **not** a descendant of the owner's original Chrome handle. Children descended from that handle (renderer, GPU, utility) are allowed. `build/src/browser.js:38-40,89-96` shows the `wsEndpoint` path connects through Puppeteer; it does not establish process ancestry. The owner must measure ancestry from Windows process creation data.
- Clock: the original +1020 s work cutoff is reached. There is no renewal.

## 5. Lifecycle (refines R141 §5)

| State | Requirement |
|---|---|
| ADMITTED → OWNER_HELD | §1 recapture PASS; A1 and A2 evidence accepted by Local; single-daemon/no-foreign-Chrome preflight. Then the owner launches Chrome once, records the GUID endpoint, starts the daemon once, and §2.4 `status` must PASS. |
| OWNER_HELD → PRELIMINARY_OBSERVED | The producer's own `new_page` (released by the owner); the observer's own call on that page ID; the owner pins the full inventory (N-R1). |
| PRELIMINARY_OBSERVED → PRODUCT_WINDOW | Allowlisted tools only, each with the admitted `pageId`; owner gate and F5 check before every release. |
| any → CLEANUP | On any §4 condition. Unknown provider outcomes keep their debit or hold; there is no retry or relaunch. |
| CLEANUP → TERMINAL | The owner closes Chrome through its original handle and waits on it and on the daemon child within the +1200 s reserve. Evidence: original-handle waits, GUID, PIDs, argv/env receipts, all actor envelopes. |

## 6. Errata to R141 (F6)

The R141 spec §8 step 1 says "the four files". The correct count is **five**: `browser-options.js`, `ToolHandler.js`, `ToolDefinition.js`, `McpPage.js` and `package.json`. R148 has since compared the full tree. This contract corrects the wording here instead of editing the frozen R141 bytes.

## 7. Still not executable

- A1 (Chrome refuses a non-matching browser GUID, including after port reuse) and A2 (a missing `pageId` is rejected before the handler) must pass the separately admitted Windows VERIFY in the draft packet.
- Source boundary for A1: `build/src/browser.js:38-40,89-96` passes the supplied endpoint to Puppeteer; it does not prove Chrome's GUID rejection on a raw WebSocket. That behavior is the Windows A1 discriminator. Source boundary for A2: `build/src/tools/ToolDefinition.js:16-24,32-34` defines the required `pageId`; `build/src/ToolHandler.js:137-140,183-199` attaches that schema to page tools and routes validated calls. The Windows A2 discriminator must observe rejection and absence of handler effects, not infer them from static source alone.
- Only then may Local consider a successor repair audit that names an integrated-lifecycle action. Execution of the integrated product campaign still needs its own effect admission.
- No physical no-relaunch, M3, runtime-governance or product claim follows from this contract.

## Complete milestone coverage

```cvf-milestone-coverage-json
{"schemaVersion": "cvf.milestoneCoverage@1.0.0", "authority": {"path": "docs/reviews/planning/CCMA_PRODUCT_DELIVERY_ROADMAP_2026-10-13.md", "heading": "## 7. M3 — Người duyệt, giao việc và kết quả xử lý", "sha256": "712817cd25936efd4fb13e0dd456ebe4ddceb5fccfd1cacb83042838724065a4"}, "milestone": {"id": "M3", "consumer": "Người duyệt và người xử lý ca trên một installation Pancake", "outcome": "Hoàn tất review, giao việc, kết quả và lịch sử qua browser/backend theo tiêu chí M3 đã duyệt.", "checkpoint": {"trigger": "R149 successor contract and A1/A2 packet draft complete with independent review", "owner": "Cloud ORCHESTRATOR (operator full-roadmap grant); Root/Local decides audit successor, admission and merge", "decision": "Accept or reject the OHC successor contract and the A1/A2 Windows VERIFY draft; never admit, run or reopen R139 from this PLAN."}}, "phase": "PLAN", "action": {"kind": "PLAN", "selected": [], "owner": "Cloud ORCHESTRATOR as SPEC_AUTHOR; independent Opus reviewer", "output": "docs/reviews/R149_OHC_SUCCESSOR_CONTRACT_2026-11-05.md and docs/reviews/R149_A1_A2_WINDOWS_VERIFY_WORK_ORDER_DRAFT_2026-11-05.md", "budgetMinutes": 120, "stop": "Stop on any need for execution, network, repair-audit/registry edit or product-source change.", "next": "Local reviews the PR; if accepted, Local authors the repair-audit successor naming the A1/A2 VERIFY discriminator and admits the Windows packet."}, "rows": [{"id": "M3-01", "state": "IMPLEMENTED_UNPROVED", "gap": "Quy tắc ưu tiên và applicability current-source còn mở.", "owner": "Product Owner / Root", "destination": "M1-01 quyết định trigger/quy tắc chờ và ngoại lệ; Owner xử lý quy tắc queue còn mở, Root đối chiếu source applicability trước G3.", "evidence": [{"path": "docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md", "sha256": "cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"}]}, {"id": "M3-02", "state": "IMPLEMENTED_UNPROVED", "gap": "Quyền, reason và binding current-source chưa có acceptance đầy đủ.", "owner": "Root / Product Owner", "destination": "Root kiểm binding và trace; Owner chấp nhận process tại G3.", "evidence": [{"path": "docs/reviews/R131_M3_ACCEPTANCE_REVIEW_2026-10-19.md", "sha256": "ec78f8db455589e9bad8a35d6f757ebb0068de5e175853e841a3fd0188e4c31b"}, {"path": "docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md", "sha256": "cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"}]}, {"id": "M3-03", "state": "IMPLEMENTED_UNPROVED", "gap": "Applicability lineage và negative hiện hành còn cần đối chiếu.", "owner": "Root / Product Owner", "destination": "Root kiểm source; G1 stale/competing negative nếu source thay đổi hoặc lộ gap; Owner chấp nhận effect.", "evidence": [{"path": "docs/reviews/R131_M3_ACCEPTANCE_REVIEW_2026-10-19.md", "sha256": "ec78f8db455589e9bad8a35d6f757ebb0068de5e175853e841a3fd0188e4c31b"}, {"path": "docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md", "sha256": "cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"}]}, {"id": "M3-04", "state": "IMPLEMENTED_UNPROVED", "gap": "Role, SLA và demo current-source chưa hoàn tất.", "owner": "Product Owner / Root", "destination": "M1-02/04 quyết định role/SLA; G3 demo assignment, outcome và audit.", "evidence": [{"path": "docs/reviews/R131_M3_ACCEPTANCE_REVIEW_2026-10-19.md", "sha256": "ec78f8db455589e9bad8a35d6f757ebb0068de5e175853e841a3fd0188e4c31b"}]}, {"id": "M3-05", "state": "BLOCKED", "gap": "R139 integration BLOCKED. OHC successor contract (R149) drafted; A3 identity accepted under the operator-relaxed condition (package.json explained, README.md absent) subject to same-session recapture; A1/A2 Windows VERIFY unrun and unadmitted.", "owner": "Root / Product Owner", "destination": "R144 finite A3 metadata VERIFY returns to Root; separate A1/A2 admission and R139 causal audit remain required before runtime.", "evidence": [{"path": "docs/reviews/R137_INDEPENDENT_SOURCE_REVIEW_2026-10-21.md", "sha256": "63cc0bc309a235c7d36384b7b78afa48acafea8267622d93d191780633071783"}, {"path": "docs/reviews/R139_GEMINI_NATIVE_BROWSER_CHECK_ROOT_REVIEW_2026-10-28.md", "sha256": "0f434528ae218886eca1c8c68efb68d3cab164cbad57b2a2b51af79f14135d47"}, {"path": "docs/reviews/R142_LOCAL_STACK_MERGE_REVIEW_AND_SUCCESSOR_CONTRACT_2026-11-05.md", "sha256": "570487e6720040efcee90e4d370f7be6ab04f0851b94e989ec3ecbd8a39f3435"}]}, {"id": "M3-06", "state": "IMPLEMENTED_UNPROVED", "gap": "R140 selection-read source/component repair FROZEN; browser lifetime and channels-store applicability remain unproved.", "owner": "Root / Product Owner", "destination": "Root keeps browser lifetime for admitted R139 G2 and separately assesses channels-store applicability against M3 queue/filter.", "evidence": [{"path": "docs/reviews/R137_INDEPENDENT_SOURCE_REVIEW_2026-10-21.md", "sha256": "63cc0bc309a235c7d36384b7b78afa48acafea8267622d93d191780633071783"}, {"path": "docs/reviews/R142_LOCAL_STACK_MERGE_REVIEW_AND_SUCCESSOR_CONTRACT_2026-11-05.md", "sha256": "570487e6720040efcee90e4d370f7be6ab04f0851b94e989ec3ecbd8a39f3435"}]}, {"id": "M3-07", "state": "UNKNOWN", "gap": "Phạm vi R101 hiện không gửi notification; applicability với source và scope hiện hành còn cần kiểm tra.", "owner": "Root", "destination": "G3 đối chiếu applicability của exclusion đã có; nếu scope đổi, áp dụng change control trước khi xét contract notification.", "evidence": [{"path": "docs/reviews/R131_M3_ACCEPTANCE_MATRIX_2026-10-19.md", "sha256": "7dabfc6a2a8bbf42a4447063e961a428849b0a020ae4c02589a7fb323510f9ac"}]}, {"id": "M3-08", "state": "BLOCKED", "gap": "R139 integration BLOCKED. OHC successor contract (R149) drafted; A3 identity accepted under the operator-relaxed condition (package.json explained, README.md absent) subject to same-session recapture; A1/A2 Windows VERIFY unrun and unadmitted.", "owner": "Root / Product Owner", "destination": "R144 finite A3 metadata VERIFY returns to Root; separate A1/A2 admission and R139 causal audit remain required before runtime.", "evidence": [{"path": "docs/reviews/R139_GEMINI_NATIVE_BROWSER_CHECK_ROOT_REVIEW_2026-10-28.md", "sha256": "0f434528ae218886eca1c8c68efb68d3cab164cbad57b2a2b51af79f14135d47"}, {"path": "docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md", "sha256": "cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"}, {"path": "docs/reviews/R142_LOCAL_STACK_MERGE_REVIEW_AND_SUCCESSOR_CONTRACT_2026-11-05.md", "sha256": "570487e6720040efcee90e4d370f7be6ab04f0851b94e989ec3ecbd8a39f3435"}]}], "supportingWork": [], "conclusions": {"tranche": "PENDING", "trancheSummary": "R149 successor contract and A1/A2 Windows draft authored; independent review pending", "trancheEvidence": [], "product": "PENDING", "productSummary": "M3 not accepted; M3-05/M3-08 BLOCKED; R139 not reopened", "acceptance": [], "owner": "Root/Local (audit, admission, merge); operator product decisions"}}
```

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
