# R149 independent design review: OHC successor contract and Windows A1/A2 VERIFY draft

Status: CHANGES_REQUIRED
Tranche: CCMAI-RUNTIME-149
Risk: R2
Worker: Cloud ORCHESTRATOR acting as SPEC_AUTHOR (separate agent from this reviewer)
Reviewer: Independent Opus design-review subagent appointed under operator grant (same cloud session; independence limitation disclosed)

Subject: build commit `d8e66c1184a888cbfeda66ba064332f6442724aa` on `claude/friendly-rubin-e2npmt-r149` (seed `abe76f66cb46978cc242bc1a3fdb6c0fb96a4577`, base main `aafcefbb4d074805b21d29594b8ef74f990d367c`).

Inputs read: `AGENTS.md`; `CVF_SESSION/authority/CCMAI-RUNTIME-149.json`; the R149 packet, contract and A1/A2 draft; R139 causal-chain assessment; R141 spec and its two-round independent review; R142 Local review; R148 Root disposition; `root_windows_full_tree_compare.json` and `root_npm_full_tree_manifest.json`; `R139_SDK_P01_B01_DESIGN_SPEC_2026-10-30.md:63`. Native source: the extracted, byte-verified `chrome-devtools-mcp@1.9.0` in the session scratchpad (`package/build/src`). I only read it. I did not execute, import or install it.

## 1. Independence limitation

- I run in the same cloud session as the spec author, and the orchestrator that wrote the contract appointed me under the operator grant. I am a separate subagent with no part in authoring. Model family and git identity are shared. The gate cannot prove authorship, so Local must confirm the seed's author and timing.
- I read the contract before the native source, which creates an anchoring risk. To offset it, I grepped all of `build/src` and the bundled `third_party` for environment reads, bare module specifiers and option conflicts, and did not rely only on the contract's citations.
- This review is static reading only. Nothing was started: no Chrome, daemon, node package, browser, DB or provider, and no network. I edited only this file and made no commit.

## 2. Citation verification (paths under `build/src/`)

| Contract citation | Verdict | Note |
|---|---|---|
| `daemon/daemon.js:16` | CONFIRMED | `sessionId = process.env.CHROME_DEVTOOLS_MCP_SESSION_ID \|\| ''` |
| `daemon/daemon.js:96-98` | CONFIRMED | `args: [INDEX_SCRIPT_PATH, ...mcpServerArgs]`, `env: process.env`. The bundled SDK merges the env over `getDefaultEnvironment()` (`third_party/index.js:40374-40404`, spawn with `shell:false`) |
| `daemon/daemon.js:136-145` | CONFIRMED | `status` returns `{pid, socketPath, startDate, version, args: mcpServerArgs}` as a **JSON string** in `result`. `mcpServerArgs = process.argv.slice(2)` (`:87`) |
| `daemon/utils.js:29-33` | CONFIRMED (off by one) | The Windows named-pipe block is `:30-35`, with `return` at `:34`. The pipe name uses session ID plus username and ignores XDG |
| `daemon/utils.js:37`, `:52`, `:45-62` | CONFIRMED | `getRuntimeHome` checks `XDG_RUNTIME_DIR` (`:52`) before the Windows fallback `path.join(os.tmpdir(), appName)` (`:61`) |
| `daemon/client.js:105-109` | CONFIRMED | `sendCommand` throws `Daemon is not running.` and has no start path |
| `config/browser-options.js:23,41,64` | CONFIRMED | `browserUrl`↔`wsEndpoint` conflict; `wsHeaders` implies `wsEndpoint` |
| `config/mcp-options.js:109,114` | CONFIRMED | `blockedUrlPattern`↔`allowedUrlPattern` (this is not a `wsEndpoint` conflict) |
| `utils/check-for-updates.js:23` | CONFIRMED | |
| `bin/chrome-devtools-mcp-main.js:20` | CONFIRMED | Only the value `'true'` has an effect |
| `CHROME_DEVTOOLS_MCP_NO_USAGE_STATISTICS` "(`config/mcp-options.js`)" | CONFIRMED, no line given | `:288` (same line as `CI`). `DISPLAY` is `browser.js:111,115`: Linux-only `detectDisplay`, which also **writes** `DISPLAY` |
| "15/15 (R141), 346/348 (R148)" | CONFIRMED | Matches the R141 review and `root_windows_full_tree_compare.json` |

## 3. Environment-variable audit

**Claim checked:** the package outside `third_party` reads exactly the seven listed variables. This is **incomplete**:

- `telemetry/persistence.js:22,25` reads `LOCALAPPDATA` and `XDG_DATA_HOME` (via `const { env } = process`). This path is reached only when `usageStatistics` is on (`index.js:43-45`). Under the admitted `--no-usage-statistics` plus `..._NO_USAGE_STATISTICS=1`, it is inert.
- Node built-ins read the environment implicitly:
  - `os.tmpdir()` reads `TEMP`/`TMP` (then `SystemRoot`/`windir`) on Windows. It decides the **PID-file location** (`daemon/utils.js:61`), the default filesystem root (`config/mcp-options.js:8`) and temp files (`utils/files.js:10`).
  - `os.homedir()` reads `USERPROFILE`. It is used only by the update-check cache (gated off) and the launch-only path (`browser.js:129`).
- On Windows, `TEMP`/`TMP` therefore plays the same load-bearing role that the contract gives `XDG_RUNTIME_DIR` on POSIX. If the owner and an actor had different `TEMP` values, the actor's `sendCommand` would read a different PID file. That fails closed (`Daemon is not running` leads to HOLD), but it is a real coupling.
- `R139_SDK_P01_B01_DESIGN_SPEC_2026-10-30.md:63`, which the contract cites, already requires the same owned `TEMP`/`TMP` parent for the owner and every caller. The contract should restate this in §2.3 rather than imply that only XDG matters.

**Bundled `third_party` reads.** None of them weakens connect-only mode:

| Read | Location | Effect under the closed argv |
|---|---|---|
| `PUPPETEER_WEBDRIVER_BIDI_ONLY`, `PUPPETEER_TEST_EXPERIMENTAL_CHROME_FEATURES`, `PUPPETEER_DANGEROUS_NO_SANDBOX` | `index.js:72848,73062,73128` | Launcher only; unreachable while `wsEndpoint` is set. Unsetting them is harmless hardening |
| `LOCALAPPDATA`, `CHROME_CONFIG_HOME`, `XDG_CONFIG_HOME`, Windows program-file prefixes | `index.js:67980-68072` | Puppeteer browser-path and channel resolution only (launch/channel) |
| `DEBUG` | `lighthouse-devtools-mcp-bundle.js:571` | `debug` logging only |
| `NODE_DEBUG` | `index.js:10176` | semver logging; `NODE_*` is unset |
| `YARGS_MIN_NODE_VERSION` | `index.js:6173` | Can only make yargs refuse to run, which fails closed |
| yargs `getEnv` | `index.js:6587` | No `.env()` prefix is configured (R141 review); used for locale and help only |
| `WS_NO_BUFFER_UTIL`, `WS_NO_UTF_8_VALIDATE` | `index.js:69438,69911` | **Checked for out-of-tree code loading.** No such loading occurs: the bare `require('bufferutil')` runs in ESM scope, where no `require` binding exists (the module defines only `require$1`/`require$2`, `:6184,6578`). It therefore throws and is caught. `utf-8-validate` is reached only when `buffer.isUtf8` is missing. No native addon from Antigravity's parent `node_modules` is loaded |
| `PATH`, `PATHEXT`, `comspec`, `OSTYPE` | `index.js:39841-40263` (cross-spawn) | The MCP child is spawned from the absolute `process.execPath` with `shell:false` |

Other bare specifiers that resolve outside the package tree:
- `proxy-agent` is used only by `httpRequest` (`index.js:67708-67712`), which is the browser-download path.
- `@toon-format/toon` and `@blackwell-systems/gcf` load only with `--experimentalDataFormat`, which the closed argv forbids.

So the A3 package-tree identity is not bypassed by module resolution in this configuration.

**Proxy.** No `HTTP(S)_PROXY`/`NO_PROXY` read exists in the bundle. Puppeteer's `NodeWebSocketTransport` uses bundled `ws` with no agent (`index.js`, `class NodeWebSocketTransport`). Node's own `fetch`/`WebSocket` honour proxy variables only with `NODE_USE_ENV_PROXY`/`--use-env-proxy`, and `NODE_*` is unset. Proxy variables therefore do not affect the localhost CDP WebSocket. They may stay in the OS minimum or be removed.

**Windows OS minimum.** An over-minimised environment can break sockets: Winsock needs `SystemRoot`, and Node's temp handling needs `TEMP`/`TMP`. Both failures are fail-closed, but they would waste the single campaign launch. Local should name the exact OS-minimum keys instead of "the OS minimum", for example: `SystemRoot`, `windir`, `TEMP`, `TMP`, `USERPROFILE`, `LOCALAPPDATA`, `APPDATA`, `USERNAME`, `PATH`, `PATHEXT`, `COMSPEC`, `SystemDrive`, `PROCESSOR_ARCHITECTURE`. The values must be the same for the owner and every actor.

**argv blind spot.** `status.args` is `process.argv.slice(2)`, so it cannot show Node runtime flags placed before the script, such as `node --import=x daemon.js`. `NODE_OPTIONS` is unset, but the contract does not pin the Node executable path or an empty `execArgv`.

## 4. Per-condition verdicts

| Condition | Verdict | Basis / counterexample |
|---|---|---|
| **F3** `status.args` and env allowlist | BOUND, with gaps (non-blocking) | §2.4 requires exact, ordered equality plus `version` and `pid`, which matches `daemon.js:87,138-144`. Comparison must be element-wise on `JSON.parse(envelope.result).args`. All three R139 variables are set, and XDG is resolved correctly from `utils.js:45-62`. Gaps: the env-read list is incomplete, and the TEMP/TMP coupling and the Node-path/`execArgv` blind spot are not stated (§3). Owner argv must be passed as an array with no shell. Avoid a trailing `\` in `--filesystemRoot` (Windows quoting). Pin `node.exe` by absolute path, not a version-manager shim: a shim's PID would not equal `status.pid` (fail-closed). |
| **F5** per-command liveness, race stated | BOUND | §3 F5 checks the original Chrome handle and the daemon child before every release and records the hostile same-user race as an accepted limit. A benign exit inside the window fails closed: N2 connect fails with no launch. |
| **F6** errata plus all N4 conflicts | **NOT FULLY BOUND (blocking, C1)** | The errata (§6, five files) is correct. The N4 list (§2.2 lines 68-71) omits the `wsEndpoint` conflicts `executablePath` (`browser-options.js:89`), `userDataDir` (`:99`) and `channel` (`:105`). It also omits the PWA category conflict (`config/category-options.js:37`) and `autoConnect`'s conflicts (`:10`). It lists `allowed/blockedUrlPattern` instead, which is not a `wsEndpoint` conflict. The R141 round-2 review named exactly this residual ("N4 still omits the executablePath/userDataDir/channel conflicts"), and R142 condition 3 requires citing **all** of them. |
| **N-R1** full inventory | BOUND (non-blocking notes) | §3 pins the full `list_pages` inventory, including the initial tab. Mutation is limited to the admitted ID. The only allowed change is URL/title on the admitted page, and §4 makes any other change a HOLD. Native listings are `id: label [selected]` (`McpResponse.js:601,620`), so the `[selected]` marker is part of the inventory and must not change either. Popups and `target=_blank` correctly HOLD (fail-safe). |
| **N-R2** owner gate vs R-F1 | BOUND (non-blocking gap) | The sequencing-gate wording matches R142 condition 5. Superseding R-F1's relay does **not** reduce safety: the R141 reviewer already found that F1 safety rests on pinning plus N5 under either reading. The owner's "sees every response" property now depends on the actor contract. §4 does not make the following a HOLD: (a) a missing or late envelope; (b) a `sendCommand` timeout or `Socket closed`, which is an unknown tool outcome (60 s default, `client.js:101,115-118,130-134`); (c) any actor call without a release (bullet 4 of the R141 review's N-R2 binding). Today these only stall until the clock cutoff. |
| **N-R3** descendants allowed | BOUND (non-blocking) | §4 is correct. The detection basis and timing are unstated. Use a parent chain with a creation-time check (Windows PID reuse) and check at preflight, at each release (with F5) and at TERMINAL. |
| **A1/A2 routing** | BOUND | Contract §7 and the draft header route admission only through Local's repair-audit successor (discriminator plus `admittedAction`), independent admission review, pre-dispatch gate and separate effect admission. Neither uses admission language. |

## 5. A3 identity (C2)

§1 states the operator's relaxation exactly:
- `package.json` must equal `592ef654…` (explained by R144);
- `README.md` must be absent;
- the other 346 of the 348 manifest paths must have equal bytes and SHA-256;
- there must be no extras;
- the capture is a same-session recapture immediately before execution.

This matches the seed's `ownerAuthorization` and the R148 compare (347 installed, 346 matched, missing `README.md`, only `package.json` mismatched). It meets C2.

Non-blocking precision items:
- **Capture method.** Specify `Get-ChildItem -Recurse -File -Force`. The R148 capture omitted `-Force`, so hidden or system files are invisible to "no extras". The effect is inert here (§3 found no out-of-tree or extra-file loading), but exactness needs it.
- **Comparison rule.** Paths are relative, forward-slash and compared case-sensitively.
- **TERMINAL recapture.** Add one. "Immediate recapture" does not detect an Antigravity update *during* the run.

## 6. Draft packet assessment (C4)

Finite: yes.
- Budgets: 45 minutes, ≤3 Chrome launches, ≤1 daemon start, ≤8 native calls, ≤6 handshakes. Counting the procedure gives 7 native calls (including `status`) and 5 handshakes, both inside the limits.
- Stop conditions and the four outcome classes are closed, with UNKNOWNs retained and no retry.
- There is no admission language, and the pass is correctly limited ("not physical no-relaunch proof … not M3").

Node's built-in global `WebSocket` is enabled by default from Node 22 and fits the package's `engines` (`^20.19 || ^22.12 || >=23`). It is appropriate for a control handshake, but see the A1 discriminator finding below.

Defects and design notes:

1. **`chrome.exe --version` (precondition 2, draft line 29). Blocking; operator-machine safety; R139:63.** Chrome on Windows does not, to my knowledge, implement `--version` as print-and-exit; that switch is handled on POSIX builds. On Windows the command starts a normal browser with the **default user profile**, or signals an already-running default-profile instance to open a window. That makes it an unbudgeted launch of a personal/default Chrome, which R139:63 forbids ("No personal/default Chrome"). It contradicts the draft's own "starts no browser session".
   - Fix: read `(Get-Item '<chrome.exe>').VersionInfo.ProductVersion` without executing it, and record the executable path and SHA-256.
2. **A1 discriminator (MEDIUM, non-blocking).**
   - Problem: the WHATWG `WebSocket` API exposes no HTTP status. A refused upgrade, `ECONNREFUSED` and a reset all surface as `error` followed by close `1006`, so "connection close before open" (A1-b, A1-c) cannot tell GUID rejection apart from "nothing listening".
   - Status capture: require the non-101 HTTP status. Either the PowerShell `ClientWebSocket` exception text carries it, or a Node built-in `http` upgrade request does (the draft currently excludes `node:http`).
   - Listener check: require proof that the listener on P is the expected Chrome browser PID (`Get-NetTCPConnection -LocalPort P -State Listen` → `OwningProcess`).
   - Classification: a TCP-level refusal is INCONCLUSIVE.
   - Ordering: in A1-c, run the Chrome #2 control both before and after the old-endpoint attempt.
3. **Fixed port P (MEDIUM, non-blocking).**
   - Choose P outside Windows' excluded port ranges (`netsh int ipv4 show excludedportrange`) and outside the dynamic range.
   - Between Chrome #1 exit and Chrome #2 start, another process can bind P. Chrome #2 then has no DevTools server, and the old-endpoint test would hit a foreign listener.
   - Require Chrome #2's `DevToolsActivePort` to report P and the listener PID to match; otherwise INCONCLUSIVE, and do not try the old endpoint.
   - Exposure on the operator's machine is limited: localhost only, fresh profile, `about:blank`, for minutes.
4. **Daemon stop and cleanup (MEDIUM, non-blocking; also contract §5).**
   - On Windows, "stop the daemon through its handle" is `TerminateProcess`. `daemon.js` cleanup runs only on signals or `stop` (`:124-135,209-247`), so the PID file remains. Cleanup step 3 ("no daemon PID file remains") then fails, or invites manual deletion.
   - The MCP child (`node chrome-devtools-mcp.js`, a grandchild) is never waited on. It exits on stdin EOF (`bin/chrome-devtools-mcp-main.js:25-50`), but that is not an original wait.
   - Fix: the owner sends its own `sendCommand({method:'stop'})` (not an actor call), then does a bounded wait on the daemon handle, with forced termination as the recorded fallback. Record the MCP-child PID from the process tree and observe its exit.
5. **A2 details (LOW).**
   - A2-a omits the contract's `pid` equality check.
   - A2-c's side-effect check is non-discriminating, because `take_snapshot` is read-only. The discriminator is the response class. State the accepted rejection shapes: envelope `success:false`, or `result.isError` with an MCP input-validation (-32602) message naming `pageId`.
   - Optionally use `navigate_page` to `about:blank#a` without `pageId`. A fall-through to the selected page would then be visible in the following `list_pages`.
   - Close the raw A1 WebSocket clients before the daemon connects.
   - "Chrome #3 if A1-c was skipped" has no skip condition.
   - Give each step a finite timeout (Chrome start, handshake, `sendCommand`).

## 7. Contradiction check (C3)

N1–N9 are not contradicted:
- the closed argv keeps `wsEndpoint` truthy (N1–N3);
- `--no-page-id-routing` and `--slim` stay forbidden (N7);
- the owner closes Chrome (N9);
- `reconnected` is a HOLD (N6).

The Root-selected lifecycle is not respected as an authority, however (blocking finding B1 below).

## 8. C1–C6

| ID | Verdict | Reason |
|---|---|---|
| C1 | **FAIL** | F6 does not cite all N4 conflicts (B2). The other six conditions are bound, with non-blocking gaps (§4). |
| C2 | PASS | §5 above |
| C3 | **FAIL** | Precedence inversion and omitted Root rows (B1) |
| C4 | **FAIL** | Finite and free of admission language, but precondition 2 directs a default-profile Chrome start (B3) |
| C5 | PASS | Diff `aafcefbb..d8e66c1`: 13 files, all inside the seed's `allowedPaths`. No probe bytes, no `docs/milestone-coverage-authority.json` or repair-audit edit, R139 stays BUILD_BLOCKED, no execution. `docs/catalog/ARTIFACT_REGISTRY.json` changes only the handoff pointer (allowed). Wording note L1 below. |
| C6 | Satisfied by this review | Findings are returned with counterexamples. |

## 9. Findings, ranked

**B1: MEDIUM, blocking (C3; R139 reopen condition; R141 §5 "refines, does not replace").**
- Location: contract line 19 ("Where two sources conflict, the later item wins"), with the R139 causal assessment ranked lowest (line 21). Line 10 claims the contract "consolidates every binding requirement".
- What the §5 table (lines 136-145) omits:
  - from the Root lifecycle: "Root verifies required provisional evidence … within the original clock; G1/E01-E03 precedes G2/E04-E06"; the TERMINAL → INDEPENDENT_ACCEPTANCE row; the rule that cleanup is independently scheduled with failure evidence preserved and that stop acknowledgment or PID absence is not an original wait;
  - from R-F2: the explicit prohibition of the native CLI, `start` and `startDaemon` and direct `bin/chrome-devtools-mcp*.js` runs, the exclusive session namespace, and the TERMINAL single-daemon/no-foreign-Chrome observation.
- Counterexample: an executor treats R149 as the consolidated, highest-precedence contract. It enters PRODUCT_WINDOW right after the owner gate and F5 checks, with no Root verification, and runs E04-E06 actions before G1/E01-E03. It closes TERMINAL without the R-F2 terminal observation. Every step complies with R149 §5 as written. The R141 reviewer (D4) already asked that unrepeated Root rows be stated as still controlling.
- Required change:
  - (a) Precedence: the R139 Root lifecycle, R141 N1–N9 and R141 §4.5 remain controlling. R149 only adds requirements. "Later wins" applies only to the N-R1..N-R3 supersession of R-F1 wording and to the operator's A3 relaxation.
  - (b) Add the omitted Root rows and R-F2 items, or state that they apply unchanged.

**B2: LOW severity, blocking for C1 (R142 condition 3).**
- Location: contract lines 68-71.
- Problem: the N4 conflict list is incomplete. The flags are forbidden anyway, so safety is unaffected, but the condition says "cite all N4 option conflicts", and the R141 round-2 review flagged exactly these omissions.
- Required change: add `executablePath`↔`wsEndpoint` (`browser-options.js:89`), `userDataDir`↔`wsEndpoint` (`:99`), `channel`↔`wsEndpoint` (`:105`), PWA category↔`wsEndpoint` (`category-options.js:37`) and `autoConnect`↔`isolated/executablePath` (`:10`). Mark `allowed/blockedUrlPattern` as unrelated to N4.

**B3: MEDIUM, blocking (C4 honesty; R139:63; operator-machine safety).**
- Location: draft line 29. `chrome.exe --version` is not a non-launching version query on Windows (§6 item 1).
- Required change: use a non-executing file-version read and record the executable path and SHA-256.

**M1: MEDIUM, non-blocking.** The A1 refusal discriminator (§6 item 2): require HTTP status plus listener-PID evidence, and classify a TCP refusal as INCONCLUSIVE.

**M2: MEDIUM, non-blocking.** Fixed-port collision and Windows excluded ranges (§6 item 3).

**M3: MEDIUM, non-blocking.** Daemon stop through the native `stop`, plus MCP-child observation. This affects draft cleanup lines 75-77 and contract §5 line 144 (§6 item 4).

**M4: LOW-MEDIUM, non-blocking.** Env-claim completeness (contract line 77): `LOCALAPPDATA`/`XDG_DATA_HOME` (gated), the implicit `TEMP`/`TMP`/`USERPROFILE` reads, an explicit `TEMP`/`TMP` equality rule for owner and actors, named Windows OS-minimum keys including `SystemRoot`, and pinning the Node executable with an empty `execArgv` (§3).

**M5: LOW, non-blocking.** N-R2 HOLD completeness: a missing or late envelope, a `sendCommand` timeout or `Socket closed` (unknown outcome), and an unreleased actor call (§4).

**L1: LOW.** Coverage rows M3-05 and M3-08 (contract line 159):
- "A3 identity accepted under the operator-relaxed condition" reads as an acceptance that Local has not recorded. Suggested wording: "R148 capture satisfies the operator-relaxed condition; Local acceptance and same-session recapture pending".
- The destination text "R144 finite A3 metadata VERIFY returns to Root" is stale, because R144 is closed.

**L2: LOW.** Contract line 48 fixes `--headless=new`. R141 §4.1 left headless to the execution admission, and it touches the M3-08 viewport/raster qualification. Local should confirm it.

**L3: LOW.** A3 capture precision: `-Force`, the path-normalisation rule and a TERMINAL recapture (§5). N-R3 detection basis (§4). The `[selected]` marker in the N-R1 inventory (§4). The `utils.js:29-33` citation should read `:30-35`, and `CI`/`..._NO_USAGE_STATISTICS` should cite `mcp-options.js:288`.

All three blocking items are documentary fixes. They fit the seed's single consolidated repair round, and none needs execution, network or a change of design direction.

## 10. Gate result

Run once from the project root (the gate accepted `--base aafcefbb`). Exit code **0**.

```
[PASS] CVF-DG-INST-01 PASS
[PASS] CVF-DG-CONT-01 PASS
[PASS] CVF-DG-RANGE-01 PASS  (RANGE_RESOLVED mode=RANGE base=aafcefbb4d07… head=d8e66c1184a8… candidates=13 deleted=0)
[PASS] CVF-DG-APPL-01 NOT_APPLICABLE_WITH_REASON - no candidate work-order packet in scope
[PASS] CVF-DG-ROUTE-01 NOT_APPLICABLE_WITH_REASON - no dispatch packet declares dispatchKind in scope
[PASS] CVF-DG-ROLE-01 PASS  (checked: R149 draft, contract, packet)
[PASS] CVF-DG-CLAIM-01 PASS  (0 explicit claim lines, 3 documents)
[PASS] CVF-DG-COV-01 PASS  (R149 contract; mechanical joins only; Local owns semantic acceptance)
RESULT: PASS phase=reviewer-fast mandatoryControls=8 failed=0
```

APPL-01 and ROUTE-01 were **not checked**; they did not pass. The gate passing proves only the checked static repository state. It does not prove the semantic findings above, which the gate cannot judge.

## 11. What remains for Local

1. Confirm this reviewer's independence and the seed's authorship and timing.
2. Rule on the single consolidated repair round for B1–B3 (M1–M5 recommended in the same round), and on whether this reviewer or another re-reviews it.
3. After acceptance: author any R139 repair-audit successor that names the A1/A2 discriminator, copy and adapt the draft into a Local work order (with M1–M3 applied), and obtain independent admission review, the pre-dispatch gate and a separate Windows effect admission.
4. Merge, and run the trusted bootstrap, the public doctor (retaining the two known version mismatches) and the Windows catalog gate.

This review makes no runtime, browser, physical no-relaunch, AI-governance, R139, M3-05/M3-08 or product claim. A1/A2 remain unrun and unadmitted. R139 remains BUILD_BLOCKED. Facebook/Zalo OA remain parked.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
