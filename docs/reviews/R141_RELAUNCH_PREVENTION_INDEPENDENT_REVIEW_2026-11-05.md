# R141 independent design review: owner-held Chrome, connect-only native MCP (OHC)

Status: REVIEWER_ACCEPTED_DESIGN_PENDING_LOCAL
Tranche: CCMAI-RUNTIME-141
Risk: R2
Worker: Cloud ORCHESTRATOR acting as DESIGN_OWNER / SPEC_AUTHOR (separate agent from this reviewer)
Reviewer: Independent Opus design-review subagent appointed under operator grant (same cloud session; independence limitation disclosed)

Subject: `docs/reviews/R141_RELAUNCH_PREVENTION_DESIGN_SPEC_2026-11-05.md` and its packet, at build commit `bb5f976139853b01e5f857821f2574c2670effb6` (seed `19018e2d0c57f577f2cd309f7fa2611b0bc3a301`, branch `claude/friendly-rubin-e2npmt-r141`).
Inputs read: `AGENTS.md`, `R139_CAUSAL_CHAIN_ASSESSMENT_2026-11-05.md`, `R139_SDK_P01_B01_DESIGN_SPEC_2026-10-30.md`, `CVF_SESSION/authority/CCMAI-RUNTIME-141.json`, `CVF_SESSION/tranches/CCMAI-RUNTIME-141.json`, the R141 packet and spec, and `probes/r141_relaunch_prevention_20261105/source_verification.json`. I also read the extracted `chrome-devtools-mcp@1.9.0` package in the session scratchpad. I only read it: I did not execute, import or install it.

## 1. Independence limitation

- I run in the same cloud session as the spec author, and the orchestrator appointed me under the operator grant. I am a separate subagent with no part in authoring. Model family and git identity are shared. The gate cannot prove authorship, so Local must confirm the seed author and timing.
- I read the spec before the native source, which creates an anchoring risk. To offset it, I searched the whole `build/src` tree and the bundled `third_party` code for launch and spawn paths, and did not rely only on the spec's citations.
- This is static reading only. No Chrome, daemon, node, package, browser, DB or provider was started, and no network was used. I edited nothing except this file and made no commit.

## 2. Hash re-verification

I recomputed SHA-256 over the scratchpad package (`package.json` version `1.9.0`):

| Set | Result |
|---|---|
| 15 R139 inventory files (S06–S17, S28, S29, S34) | **15/15 match** both `npmTarballSha256` and `windowsRecordedSha256` in the probe |
| Windows-recorded hashes cross-check | Each of the 15 Windows hashes also appears in pre-existing R139 records (`sdk_p01_b01_design_20261030/design_source_inventory.json`, `design_contract.json` and `connection_recovery_20261029/native_cli/*`). The probe did not invent them. |
| Additional cited files | **5/5 match**: `config/browser-options.js`, `ToolHandler.js`, `tools/ToolDefinition.js`, `McpPage.js`, `package.json` |

The task description, spec §2 and A3 say "four" additional files, but the probe lists five, including `package.json`. A3 should cover all five (finding F6).

## 3. N1–N9 verification (paths under `build/src/`)

| ID | Verdict | Reviewer citation and notes |
|---|---|---|
| N1 | CONFIRMED | `index.js:158-189`. `browserUrl \|\| wsEndpoint \|\| autoConnect` selects `ensureBrowserConnected` (159-172). Otherwise `ensureBrowserLaunched` (173-189). `serverArgs` is parsed once (`bin/chrome-devtools-mcp-main.js:18`) and nothing in `build/src` reassigns `wsEndpoint/browserUrl/autoConnect`. |
| N2 | CONFIRMED | `browser.js:24-105`. Returns the connected browser at 26-28. The `wsEndpoint` branch (38-43) takes precedence over `browserURL`, `channel` and `userDataDir`. Only `puppeteer.connect` runs (94). Failure throws `Could not connect to Chrome` (98-101). Bundled puppeteer `connect` (`third_party/index.js:67550-67562`, `67573-67583`, `67680-67688`) only opens a WebSocket transport and has no launch path. |
| N3 | CONFIRMED | `launch` is at `browser.js:122-197` and is called only by `ensureBrowserLaunched` (`198-207`). Its only caller is `index.js:173`. `puppeteer.launch` is called only at `browser.js:156`. In the bundle, `launch$1` (`third_party/index.js:68974`) is reached only through the launcher class (`:72705`, `:73368-73376`). |
| N4 | CONFIRMED (incomplete) | `config/browser-options.js:37-60`. Also: `wsEndpoint` conflicts with `executablePath` (89), `userDataDir` (99) and `channel` (105). The coerce function returns `undefined` for an empty value (42-45); see F3. |
| N5 | CONFIRMED | `McpContext.js:19-23` (comment and counter), `130-132` (test-only reset, with no caller in `build/src`), `394` (`nextPageId++`). After a reconnect, `#init` → `createPagesSnapshot` (`63`, `406-408`) gives every surviving tab a new, higher ID, so an old ID throws `No page found` (`323-328`). |
| N6 | CONFIRMED with qualification | The cited range is `index.js:190-203`, not 189-201. Also `McpResponse.js:505-507`. The notice is **one-shot per context** (`McpContext.js:318-322`) and is consumed by whichever tool call comes first (`ToolHandler.js:183-185`), from any actor. It is **lost** if `response.handle` throws after consumption (`ToolHandler.js:225`, catch at `238-252` returns only error text). `structuredContent` is emitted only when `experimentalStructuredContent` is set (`ToolHandler.js:233-235`); that is the default under `--viaCli` (`config/mcp-options.js:250-253`). See F1. |
| N7 | CONFIRMED with qualification | `config/mcp-options.js:39-43`, `ToolHandler.js:134-141`, `tools/ToolDefinition.js:16-34`. However, the handler itself **falls back to `getSelectedMcpPage()` when `pageId` is undefined** (`ToolHandler.js:190-196`). Required-ness comes only from SDK validation (see A2). `evaluate_script` is *not* page-scoped (`tools/script.js:61`, `defineTool`). Under `--viaCli` its `pageId` is optional in the schema (`script.js:71-79`, because `categoryExtensions` defaults to true at `mcp-options.js:246-249`). It is enforced only in the handler (`script.js:144-146`), and a `serviceWorkerId` path bypasses `pageId` (`script.js:121-141`). See F4. |
| N8 | CONFIRMED | `daemon/daemon.js:87,94-98`. argv is forwarded unchanged. The **full `process.env`** is also forwarded (97); see F3. The daemon starts the MCP child once (`setupMCPClient`, 91-107) and has no restart logic. |
| N9 | CONFIRMED | `browser.js:215-232`. In connected mode the browser is only disconnected (229-231). |

## 4. Counterexample hunt

| Probe | Result |
|---|---|
| Other Chrome launch or spawn sites in `build/src` (daemon, bin, tools, telemetry, devtools, extensions, lighthouse, screencast, performance) | `spawn` appears only in `daemon/client.js:90` (daemon), `telemetry/WatchdogClient.js:32` (node watchdog) and `utils/check-for-updates.js:64` (node update check). `execSync` appears in `browser.js:113` (DISPLAY probe) and `bin/check-latest-version.js:17` (npm registry). None of these launches Chrome. The Lighthouse bundle imports only `events/fs/path/process/stream/zlib`; `chrome-launcher` occurs only in an embedded package.json string. The screencast tool spawns ffmpeg (`third_party/index.js:73581`), not Chrome. PWA tools conflict with `wsEndpoint` (`config/category-options.js` PWA `conflicts`). **No in-process launch path exists while `wsEndpoint` is truthy.** |
| Bundled puppeteer `connect()` | It cannot launch. The `channel` route only reads `DevToolsActivePort` (`third_party/index.js:67597-67635`). |
| autoConnect implicitly enabled | No. The default is `false` and coerces to `undefined` (`browser-options.js:7-18`). yargs has no `.env()` or `.config()` (`mcp-options.js:263-302`), so environment variables cannot inject options. Even with `autoConnect` set, the `wsEndpoint` branch wins (`browser.js:38`). |
| **CLI/daemon route starting a second, launch-mode server** | **Counterexample found (F2).** A native CLI tool command auto-starts a daemon when none is running (`bin/chrome-devtools.js:190-194`). Its argv comes from the CLI invocation (`daemon/utils.js:103-135`) plus `--viaCli`. Without `--wsEndpoint`, that daemon's MCP child takes `index.js:173` and launches Chrome (`browser.js:156`). `startDaemon` (`client.js:79-100`) and a direct `bin/chrome-devtools-mcp.js` run behave the same way. `sendCommand` routes to whatever daemon holds the scope PID file (`utils.js:89-101`, `client.js:105-113`). The spec's closed argv set constrains only the owner's one daemon start. |
| argv injection or silent fallback | **Found (F3).** `--wsEndpoint=` (empty) coerces to `undefined` (`browser-options.js:42-45`). The middleware then defaults `channel='stable'` (`mcp-options.js:282-287`) and the server launches (`index.js:173`). yargs is not strict: unknown arguments are only logged (`mcp-options.js:298-301`). Aliases are `-w` and `--ws-endpoint`, and duplicate flags are accepted. |
| `pageId` required (A2) | It is statically supported. Bundled `@modelcontextprotocol/sdk` 1.30.0 (`third_party/bundled-packages.json:2`) validates arguments against the registered `zod.object(...).passthrough()` before the handler runs (`third_party/index.js:39070`, `39100-39111`). `pageId: zod.number()` has no `.optional()` (`ToolDefinition.js:33`). A missing `pageId` gets an `InvalidParams` tool error (`:39076-39083`) **before** `getContext`, so it does not even trigger a reconnect. Unknown keys are also rejected (`ToolHandler.js:143-169`). |
| Reconnect / same-context residual | **Counterexample found (F1).** The notice is one-shot, cross-actor and can be lost, and the spec's product window does not pin page IDs. An actor can therefore follow the native note "Call list_pages to see open pages" (`McpResponse.js:507`), adopt a new ID and continue page-mutating calls in the replaced context without ever seeing `reconnected`. |
| Non-page-scoped tools that can mutate state | `new_page` (`tools/pages.js:88-126`; creates and selects a tab, including the `isolatedContext` parameter), `close_page` and `select_page` (`pages.js:29-86`), `evaluate_script` with `serviceWorkerId`, and the extension tools (`tools/extensions.js:9-85`; category on by default under `--viaCli`). The spec's own rule (only `new_page` + `list_pages` are non-page-scoped) excludes these, but no closed tool and parameter allowlist is written down (F4). |
| Chrome self-restart through navigation | `chrome://` URLs are blocked except newtab/inspect (`utils/url.js`, `isAllowedUrl`). Even if Chrome restarted itself, the new browser has a new port and GUID under `--remote-debugging-port=0`, and the original handle exits, which leads to HOLD. This does not break the claim. |

## 5. Assumptions A1–A3

- **A1 (Chrome refuses a WebSocket upgrade to a foreign browser GUID; the GUID is new for every launch).** This is fair and honestly scoped: it is Chrome-side and not in this package, and the proposed two-step check discriminates correctly. Suggested refinement: also state the residual that a non-Chrome listener squatting the port after Chrome exits could accept any path. Owner exit detection is asynchronous, so a call could slip in during that race. This is consistent with R139's "not a hostile same-user isolation boundary", but it should be written down.
- **A2 (SDK enforces the required `pageId`).** Correctly marked as unverified at runtime, but scoped too conservatively. The source supports it statically (§4). The spec should cite `third_party/index.js:39070,39100-39111` and `ToolHandler.js:190-196`, and keep the offline check only as confirmation on the Windows install. A2 should also say that it does not cover `evaluate_script` (F4).
- **A3 (Windows copies of the extra files match).** Fair. It must list all **five** additional files, including `package.json`.

## 6. D1–D6 verdict

| ID | Verdict | Reason |
|---|---|---|
| D1 | PASS (minor) | Every N-fact cites a verified file:line. The N6 range is off by about one line, and the extra-file count differs (F6). |
| D2 | **FAIL** | The threat of a second, launch-mode daemon/server (native CLI auto-start, `startDaemon`, direct MCP main, or a restart after daemon death) is not argued (F2). The in-process threats are argued correctly. |
| D3 | **FAIL** | The stated same-context residual ("detected in that same response", limited to one call) is inaccurate. As written, the product window allows silent continuation in a replaced context (F1). The no-relaunch requirement is not weakened in-process. |
| D4 | PASS (with note) | §5 refines the Root lifecycle without contradicting it: A1–A3 is the source-boundary enforcement precondition, with the same clock, owner, CLEANUP/TERMINAL and HOLD rules. It should state that the Root rows it does not repeat still control: Root verification at PRELIMINARY_OBSERVED→PRODUCT_WINDOW, G1/E01–E03 before G2/E04–E06, and TERMINAL→INDEPENDENT_ACCEPTANCE. It should also add daemon/MCP-child exit as a HOLD trigger (F2). |
| D5 | PASS | It makes no reopen, implementation, runtime, acceptance or AI-governance claim. Next moves are owned by Local. The milestone-coverage block is phase `PLAN` with `selected: []`, M3-05/M3-08 stay `BLOCKED` and conclusions are `PENDING`. That is appropriate for a design-only packet. "Local decides reopen" matches the causal assessment's "Root alone may reopen" because the workspace uses Local/Root for the same agent; writing "Local (Root)" would help traceability. One caution: the phrase "prevention is pre-action, subject to A1–A3" is overstated until F1 and F2 are repaired. |
| D6 | Satisfied by this review | Findings are returned with counterexamples. |

## 7. Findings (ranked)

**F1: HIGH, blocking (D3; locked same-held-context requirement).** The reconnect notice is not a reliable per-actor detector, and the product window does not pin page IDs.

- Source:
  - one-shot per context: `McpContext.js:318-322`
  - consumed by the first call from any actor: `ToolHandler.js:183-185`
  - lost if formatting throws: `ToolHandler.js:225,238-252`
  - the native text tells the caller to re-list: `McpResponse.js:507`
  - spec §4.3 allows `list_pages` during the window
  - spec §5 requires only "an explicit `pageId`", not the admitted one
- Counterexample:
  1. Producer `new_page` returns id 1, and the observer `take_snapshot {pageId:1}` succeeds.
  2. The WebSocket drops while the same Chrome stays alive.
  3. The observer's next `list_pages` reconnects and consumes the notice in *its* response. Alternatively, that response's formatting throws and the notice is lost.
  4. Producer `click {pageId:1}` fails with `No page found`.
  5. The producer, or its harness, calls `list_pages`, sees the same product tab as id 2, and runs `click {pageId:2}` successfully in the rebuilt context. No producer response ever carried `reconnected: true`.
- Required change:
  - (a) Fix the admitted page-ID set at OWNER_HELD→PRELIMINARY_OBSERVED: the ID returned by the original `new_page`, which the observer independently confirms. Any later call naming an ID outside that set is forbidden and leads to HOLD.
  - (b) Treat `No page found` or a target error on an admitted ID as context loss leading to HOLD. Never re-discover the page through `list_pages` or adopt new IDs.
  - (c) The owner must inspect every original response from every actor. A missing notice is never proof that no reconnect happened.
  - (d) Restate the residual: with (a) and (b), page-mutating actions fail closed through N5 whether or not the notice arrives. The remaining exposure is non-page-scoped calls only (`new_page` at the start, read-only `list_pages`).

**F2: HIGH, blocking (D2; locked no-relaunch requirement).** A second server process in launch mode is not prevented by the design text.

- Source:
  - CLI auto-start: `bin/chrome-devtools.js:190-194`
  - argv serialization: `daemon/utils.js:103-135`
  - `startDaemon`: `daemon/client.js:79-100`
  - launch branch: `index.js:173`, `browser.js:156`
  - PID-file liveness: `daemon/utils.js:89-101`
- Counterexample:
  1. The owner's daemon or its MCP child exits mid-campaign. The spec's HOLD triggers name only `reconnected`, connect failure and Chrome exit.
  2. An actor or harness then runs `chrome-devtools take_snapshot --sessionId=<scope> ...`.
  3. A new daemon starts without `--wsEndpoint`, and its first tool call physically launches a new Chrome.
  4. Later `sendCommand` calls route to that daemon.

  The spec's "launch branch unreachable while `wsEndpoint` is set" holds per process only. R139 rejected the CLI route, but R141's prevention argument and closed set do not restate or rely on that.
- Required change, in §4.2/§4.3/§5:
  - The owner's single foreground daemon start is the **only** server start in the campaign.
  - The actor surface is only the exported `sendCommand` (`client.js:105-109`, which throws if no daemon is running and never starts one).
  - The native `chrome-devtools` CLI, `startDaemon`/`start`, and direct `bin/chrome-devtools-mcp.js` or `chrome-devtools-mcp-main.js` invocations are forbidden.
  - Daemon or MCP-child exit, or a `status` identity mismatch (PID, startDate, args from `daemon.js:138-147`), leads to CLEANUP→TERMINAL HOLD with no restart.
  - Add this as an explicit threat row.

**F3: MEDIUM, should fix in the same round.** The closed-set preflight must be exact.

- Required properties:
  - exact token equality `--wsEndpoint=ws://127.0.0.1:<port>/devtools/browser/<guid>`, equal to the recorded endpoint and non-empty
  - no `-w` or `--ws-endpoint` alias
  - no duplicate flags
  - every token from the closed list, with no `--no-*` variants beyond those listed
  - after start, `status.args` equals the admitted argv (`daemon.js:143`)
- Why: an empty value falls through to launch (`browser-options.js:42-45` → `mcp-options.js:282-287` → `index.js:173`), and yargs only logs unknown arguments (`mcp-options.js:298-301`).
- Environment: the daemon forwards its full environment to the MCP child (`daemon.js:97`). The closed set should therefore name the admitted environment, inheriting R139's minimum-environment rule and explicitly excluding `NODE_OPTIONS` and `XDG_RUNTIME_DIR`.

**F4: MEDIUM, non-blocking.** Write a closed tool and parameter allowlist for the product window, as R139 did for capability, instead of the indirect "page-scoped" rule.

- Name `evaluate_script` explicitly: it is non-page-scoped and its `pageId` is optional in the schema under `--viaCli` (`script.js:61,71-79,144-149`). Require its `pageId`, and forbid `serviceWorkerId` (`script.js:121-141`), `select_page`, `close_page` outside cleanup, the extension tools, and `new_page.isolatedContext` unless admitted.
- Consider adding `--no-category-extensions` to the allowed closed set as hardening.
- Update the A2 scope accordingly.

**F5: LOW, non-blocking.** Extend A1 with the residual from a non-Chrome port squatter (see §5).

**F6: LOW, non-blocking.**
- Correct the N6 citation to `index.js:190-203`.
- Make §2 and A3 say **five** additional files, including `package.json`.
- Add to N4 that `wsEndpoint` also conflicts with `executablePath`, `userDataDir` and `channel`.

All blocking findings can be fixed in documentation within the one consolidated repair round the seed allows. They do not require runtime evidence, source edits or a change of design direction. OHC with `--wsEndpoint` remains the right primitive: the in-process no-launch property (N1–N3) is confirmed.

## 8. Gate result

The command was run once from the project root:
`PYTHONDONTWRITEBYTECODE=1 python3 -B scripts/cvf_gates/cvf_downstream_gate_runner.py run --phase reviewer-fast --project-root . --base 70b0c2200871bd30977d281327e4ad8fe0071e7c --head HEAD --packet <R141 packet> --packet <R141 spec>`. The gate accepted `--base`. Exit code **0**.

```
[PASS] CVF-DG-INST-01 PASS
[PASS] CVF-DG-CONT-01 PASS
[PASS] CVF-DG-RANGE-01 PASS  (RANGE_RESOLVED mode=RANGE base=70b0c22… head=bb5f9761… candidates=11 deleted=0)
[PASS] CVF-DG-APPL-01 NOT_APPLICABLE_WITH_REASON - no candidate work-order packet in scope
[PASS] CVF-DG-ROUTE-01 NOT_APPLICABLE_WITH_REASON - no dispatch packet declares dispatchKind in scope
[PASS] CVF-DG-ROLE-01 PASS  (checked: R141 packet, R141 spec)
[PASS] CVF-DG-CLAIM-01 PASS  (0 explicit claim lines, 2 documents)
[PASS] CVF-DG-COV-01 PASS  (R141 spec; mechanical joins only; Local owns semantic acceptance)
RESULT: PASS phase=reviewer-fast mandatoryControls=8 failed=0
```

APPL-01 and ROUTE-01 are **not checked**; they did not pass. The 11 changed files all fall within the seed's `allowedPaths`. No package bytes are committed. The gate passing proves only the checked static repository state.

## 9. What remains for Local

1. Confirm my independence and the seed authorship and timing. Rule on whether the spec author's one repair round for F1–F3 (and optionally F4–F6) is re-reviewed by this reviewer or another.
2. **A3 on Windows:** hash the five additional files (`config/browser-options.js`, `ToolHandler.js`, `tools/ToolDefinition.js`, `McpPage.js`, `package.json`) in the Antigravity install. No execution is needed.
3. **A1/A2:** check them only through a separately admitted finite VERIFY packet with a fresh effect admission. A2 may be narrowed to confirming the static SDK evidence above.
4. Decide on reopen after the repaired spec is accepted, by recording the updated repair-chain audit. Nothing in this review reopens R139, releases implementation or admits runtime.
5. Merge the PR to `main`, and run the trusted bootstrap, doctor and Windows catalog gates.

This review makes no claim of runtime, browser, AI-governance, M3-05/M3-08 or product acceptance. Facebook/Zalo OA remain parked.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY

## Re-review after repair round 1

Subject: the spec at commit `5fa904438d94bea40d2a884da2c5389a0a2c91be`, with the new §4.5 R-F1..R-F4, the amended §5 table, §5a, and the N6/N7/A1/A2/A3/§2 fixes. The same constraints applied as in round 1: static reading only, no execution, no network, no commit. I edited only this file: I changed the top `Status:` line and appended this section. The round-1 content above is otherwise unchanged; it matches the copy the orchestrator committed in `5fa9044`.

### Cited-line verification (package dir, `build/src/`)

| Spec citation | Verdict |
|---|---|
| `bin/chrome-devtools.js:187-194` (CLI auto-start) | CONFIRMED. The tool-command handler starts at 187. At 193-194, `if (!isDaemonRunning(sessionId)) await start(serializeArgs(mcpOptions, argv), sessionId)`. |
| `daemon/client.js:105-109` (`sendCommand` never starts a daemon) | CONFIRMED. At 107-108 it throws `Daemon is not running.`, and the function contains no start path. |
| `tools/script.js:60-108` (`evaluate_script` optional `pageId` and `serviceWorkerId`) | CONFIRMED. The optional `pageId` is at 71-79. The `serviceWorkerId` schema runs 103-110, ending just past the cited 108. The bypass is in the handler at 121-141. |
| R-F4 allowlist page-scoping | CONFIRMED. Every allowed `pageId` tool is `definePageTool`: `navigate_page` (`pages.js:128`), `handle_dialog` (`pages.js:293`), `take_snapshot` and `wait_for` (`snapshot.js`), `take_screenshot` (`screenshot.js:142`), and `click`, `hover`, `fill`, `type_text`, `fill_form`, `press_key` (`input.js:119-536`). All of them get the required `pageId` schema (N7/A2). |

### Disposition of round-1 findings

| Finding | Disposition | Basis |
|---|---|---|
| F1 (blocking): reconnect notice and unpinned page IDs | **RESOLVED** | R-F1 does five things: it pins the target ID at PRELIMINARY_OBSERVED, forbids re-acquiring IDs, makes `No page found` or any unknown ID a HOLD, requires the owner to inspect every actor response before releasing the next command, and treats any `reconnected` as HOLD. With these, every page-mutating call in a rebuilt context fails closed through N5 whether or not the notice arrives. My round-1 counterexample, where the producer adopts id 2 from `list_pages`, is now forbidden and becomes a HOLD. The §5 rows carry the same rules. |
| F2 (blocking): second launch-mode server | **RESOLVED** | R-F2 makes the owner's start the only daemon start, limits actors to `sendCommand`, forbids the CLI, `start`, `startDaemon` and ensure helpers, makes the session namespace exclusive to the owner, and makes daemon or MCP-child exit (or `Daemon is not running`) a HOLD with no restart. It adds a single-daemon preflight and a terminal observation, and §5 adds daemon/child exit to the any-state HOLD row. |
| F3: exact argv and environment | **PARTIAL (non-blocking)** | Exact parsing and the full `wsEndpoint` shape (scheme, host, port, GUID path, non-empty) are bound, and `NODE_OPTIONS`/`NODE_*`/`npm_*` are excluded. Gaps: no post-start check that `status.args` equals the admitted argv (`daemon.js:143`); it says "the two ... variables reviewed in R139" where R139 lists three (`CHROME_DEVTOOLS_MCP_SESSION_ID`, `_NO_UPDATE_CHECKS`, `_NO_USAGE_STATISTICS`); and R139's `XDG_RUNTIME_DIR`/`XDG_ROOT` omission is not restated. |
| F4: tool allowlist | **RESOLVED** | The explicit allowlist and denylist are complete for the product window. `evaluate_script`, `select_page`, `close_page`, extension and PWA tools are forbidden, and the list can be narrowed but not widened. Note: `navigate_page` accepts an `initScript` (arbitrary JavaScript on the pinned page). It is page-scoped, so it does not affect relaunch or context, but the execution admission may want to forbid it. |
| F5: A1 port-squatter residual | **PARTIAL (non-blocking)** | The scope note is present, but it claims the owner "observes the original-handle exit before any further product action". Exit observation is asynchronous, so that holds only if the owner checks original-handle liveness before releasing **each** command. Even then, a tiny race remains with a hostile same-user listener, which is out of scope under R139. The WORK_ORDER should state the per-release liveness check. |
| F6: citations and counts | **PARTIAL (non-blocking)** | §2, N6 and A3 are corrected. Remaining: §8 step 1 still says "hashing the **four** files" (it should be five); N4 still omits the `executablePath/userDataDir/channel` conflicts; the stale §4.2 closed-set wording and the §4.3 residual ("detected in that same response") remain, though §4.5 explicitly takes precedence. |

### New-counterexample hunt on the amendments

**N-R1: MEDIUM, non-blocking (fail-closed, but cannot run as written).** The pinned-set equality rule conflicts with native page listings.
- The owner-launched Chrome normally has an initial tab, and that tab gets a page ID on first context creation (`McpContext.js:63,406-408`).
- `new_page`, `navigate_page` and `list_pages` responses all list every page (`setIncludePages(true)` at `tools/pages.js:123,249,23`, rendered through `McpResponse.js:276-278`). Product popups or `target=_blank` would also add IDs (`McpContext.js:83-96`).
- R-F1 says HOLD on "any page ID outside the pinned set appearing in any response" and "any `list_pages` result that differs from the pinned set". If the pinned set is only the `new_page` ID, the first `navigate_page` or `list_pages` therefore HOLDs.
- This never weakens no-relaunch or same-context, because it fails safe. It does make the lifecycle unrunnable as literally written.
- Required in the source WORK_ORDER or admission: define the pinned **inventory** as the full page-ID set observed at PRELIMINARY_OBSERVED. Mutating calls go only to the target ID. Any change in inventory leads to HOLD, unless a popup rule is explicitly admitted.

**N-R2: MEDIUM, non-blocking condition on D4 (owner-relay boundary).**
- "All actor commands are relayed through the owner's `sendCommand` boundary" can be read two ways:
  - The owner **originates** the native calls. That would conflict with the Root lifecycle's "distinct actors with original native call receipts" and with R139 P01 ("each real actor's original one-shot ... invocation"; the wrapper "cannot ... dispatch other actors").
  - The owner acts as a **sequencing gate**. That is consistent.
- Neither reading creates a technical boundary: the daemon socket accepts any same-user caller (`daemon/daemon.js:176-188`, owner-only permissions at 190-191). F1 safety therefore rests on the pinning/HOLD rules (N5) plus the actor contract, equally under both readings.
- Required binding for Local and the WORK_ORDER:
  - Each actor performs its own original one-shot `sendCommand` invocation, only when the owner releases it.
  - Each actor returns its unmodified envelope and caller terminal to the owner before the next release.
  - The owner never originates, alters or replays an actor command.
  - Any actor call without a release is a contract breach and leads to HOLD.

**N-R3: LOW, non-blocking.** The R-F2 preflight wording "no Chrome process using the owned user-data-dir other than the owner's PID" must allow the owner Chrome's own descendant processes (renderer, GPU, utility, crashpad). Otherwise it trips falsely, though safely. Bind it to the descendant tree of the original handle.

**N-R4: no counterexample found.** These properties still hold after the amendments:
- **Single-daemon preflight.** It forbids the CLI route. A stray daemon under a different or default session would launch a Chrome with a different profile (`browser.js:128-133`). That Chrome cannot be reached through the GUID-pinned endpoint, so it cannot replace the held context. It is still a forbidden effect, and the terminal observation should cover it.
- **Exact argv rule.** It closes the empty-`wsEndpoint` fallthrough.
- **`autoConnect`.** It stays rejected and defaults off.
- **Launch path.** No path to `index.js:173` remains for the owner's daemon.

### D1–D6 after repair

| ID | Verdict |
|---|---|
| D1 | PASS. Citations verified. The "four files" wording in §8 is a residual typo. |
| D2 | **PASS**. The second-server threat is now argued (R-F2), and the in-process threats are unchanged. |
| D3 | **PASS**. Same-context is enforced fail-closed through R-F1 plus N5, the residual is restated correctly in §4.5, and no-relaunch is not weakened. |
| D4 | PASS with condition N-R2. §5 refines the Root lifecycle and adds daemon/child exit HOLD. The relay must be read as a sequencing gate that preserves original actor receipts. |
| D5 | PASS. It makes no reopen, implementation, runtime or acceptance claim, and the PLAN coverage block is unchanged and appropriate. |
| D6 | PASS. Independent review is completed for this design-review scope, subject to the independence limitation in §1. |

### Gate result (re-run once)

The same command was run at head `5fa904438d94…`. `--base` was accepted. Exit code **0**.

```
[PASS] CVF-DG-INST-01 PASS
[PASS] CVF-DG-CONT-01 PASS
[PASS] CVF-DG-RANGE-01 PASS  (RANGE_RESOLVED mode=RANGE base=70b0c22… head=5fa90443… candidates=12 deleted=0)
[PASS] CVF-DG-APPL-01 NOT_APPLICABLE_WITH_REASON - no candidate work-order packet in scope
[PASS] CVF-DG-ROUTE-01 NOT_APPLICABLE_WITH_REASON - no dispatch packet declares dispatchKind in scope
[PASS] CVF-DG-ROLE-01 PASS  (checked: R141 packet, spec, independent review)
[PASS] CVF-DG-CLAIM-01 PASS  (0 explicit claim lines, 3 documents)
[PASS] CVF-DG-COV-01 PASS  (R141 spec; mechanical joins only; Local owns semantic acceptance)
RESULT: PASS phase=reviewer-fast mandatoryControls=8 failed=0
```

APPL-01 and ROUTE-01 are **not checked**; they did not pass. The gate passing proves only the checked static repository state.

### Re-review verdict

**REVIEWER_ACCEPTED_DESIGN_PENDING_LOCAL.**
- F1 and F2 are resolved, and no new blocking defect was found.
- The seed's single consolidated repair round has been used. The open non-blocking items (F3, F5, F6 residuals and N-R1..N-R3) go to Local as binding conditions for the updated repair-chain audit and any future source WORK_ORDER or execution admission. They are not to be fixed by another spec repair round in this tranche.
- Local still owns:
  - confirming reviewer independence and the seed's authorship;
  - A3 on Windows (five files);
  - the admitted A1/A2 VERIFY;
  - the R139 reopen decision;
  - the merge.
- This acceptance covers the design document only. It is not runtime, browser, AI-governance, M3-05/M3-08 or product acceptance.
