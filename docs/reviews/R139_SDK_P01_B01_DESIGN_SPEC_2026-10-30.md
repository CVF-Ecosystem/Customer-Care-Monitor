# R139 P01/B01 native-interface and current-pin design

Status: CONDITIONAL_NATIVE_INTERFACE_DESIGN_PRODUCT_INTEGRATION_NOT_DESIGNED
Phase: DESIGN/SPEC only within global BUILD/BLOCKED
Risk: R2
Worker: Sol Medium SPEC_AUTHOR
Reviewer: Root independent design reviewer; separate Terra6.1High final reviewer
Author: Separate SPEC_AUTHOR, then REPAIR_WORKER for consolidated independent findings round1
Reviewers: Root independent; separate Terra final reviewer
Commit owner: Root only
Preparation: `ba55a586cbf94cb3ca529161cb5c617453a545e8`
Authority: `CVF_SESSION/authority/CCMAI-RUNTIME-139-SDK-P01-B01-DESIGN-ADMISSION.json`

## Receiving decision and explicit product hold

The selected **conditional native-interface capability candidate** reuses the installed native foreground `daemon/daemon.js` entry and exported native `sendCommand`. The daemon already owns one official Client/StdioClientTransport and shares that native server/context among callers. Two actual distinct actors can issue their own native operations in an exclusively owned scope while the existing lifecycle owner retains the original daemon handle. This is a source-supported internal native-interface proposal for Root technical adoption/source review, not a public interface stability guarantee, demonstrated runtime handoff, BUILD dispatch, or effect permission.

**HARD PRODUCT RELEASE HOLD:** native `browser.ensureBrowserLaunched` physically launches a replacement Chrome after connection loss; `index` reconstructs context (S14-S15). Pre/post process samples, an error or reconnect notice may detect this only **after** relaunch or an action. They cannot enforce zero physical replacement or prevent an action in that race. No proposed guard is proof that native relaunch was prevented. Product release requires source-backed enforcement preserving locked acceptance, or an explicit Owner decision changing that acceptance. Neither is supplied here.

The integrated path from provisional native evidence through original R139 G0/product admission/G1/G2/E06 close and terminal is **NOT_DESIGNED**. This design stops at a complete bounded native-interface capability lifecycle. That lifecycle must close its target and daemon and obtain original terminal/cleanup evidence; it is **not whole R139 G0 PASS** and cannot attest a later newly launched product context. Missing future proof alone does not mean the native interface is unsupported. Ordinary pinned internal-interface adoption belongs to Root; only a real locked-acceptance change belongs to Owner.

Two actual actor operations, original caller tool/process references and independent manual review establish actor responsibility. Native page/process receipts establish browser facts. No cryptographic native LLM-identity requirement is invented. Declared actor/scope/run labels alone and static review of producer logs cannot replace the independent native operation, and caller correlation is never claimed as native actor/browser-session attestation.

`design_contract.json` contains exact conditional typed methods, pin metadata, 17 future offline cases and unresolved decisions. `design_source_inventory.json` records **34/36 semantic sources**, including one direct VERSION source in repair1. The same cumulative30-active-minute/36-source envelope applies; no reset or new source investigation is requested. Historical sources/authorities and indirect/prior-role context count; exempt startup reads are listed separately. No tests, implementation, native processes, SDK imports, browser/network/provider/credential/binary queries occurred.

## Source-backed comparison

Inventory S01-S34 binds raw source hashes; repeated targeted reads do not add identities.

| Route | Native source fact | Disposition |
|---|---|---|
| Direct Client/Stdio | Native transport spawns a private child, exposes PID, rejects repeated start/disconnected send; native main owns one server transport. S04,S10-S13. | Retain prior single-owner capability. Two transports are different children; not selected actor API. Daemon internal Client/Stdio is implementation, not permission for callers to connect another client. |
| Official scoped CLI | Real `sessionId` multi-caller daemon; native CLI `start` stops an existing matching daemon and `startDaemon` detaches/unrefs. CLI tool commands auto-start if missing. S06-S09. | Rejected wrapper route for this proposal. No shared/default namespace, CLI restart, startup/ensure/version-repair/stopDaemon helper. |
| Foreground native daemon + native export | Native `DAEMON_SCRIPT_PATH` is the executable daemon entry invoked by vendor startup. Daemon owns one MCP Client/Stdio, relays native commands, cleans owned client/transport; exported `sendCommand` checks running state and never starts a daemon. S06-S08. | Selected conditional source candidate only. Original foreground owner/caller handles, complete bounded lifecycle and independent source review remain required. |
| Existing CUA, no change | Existing supervisor/Go page-target branches require genuine original CUA references, distinct producer/observer, exact target/final inventory/terminal. S20-S21. | Preserve unchanged. A supported future host with actual actors is an alternative; spent prior capability runs do not renew evidence or effect slots. |

`McpServer` owns context/mutex (S14); `McpContext.getPageById` resolves IDs in that context (S28). Native page tools return actual pages and refuse last-page close (S12). `McpResponse` provides source-defined text/optional structured shapes and reconnect notice with changed IDs (S29). Do not reuse v3's ExampleDomain-only parser for product pages, demand structuredContent always, invent CUA IDs or equate page IDs across restarts. Native VERSION source S34 exports **`1.9.0`**; exact raw SHA256 is in both JSON artifacts. This is source text, not an executed version command or availability check.

## Exact conditional native invocation and namespace

A future separately admitted technical lifecycle would use:

```text
command: C:/Program Files/nodejs/node.exe
entry: C:/Users/tiennm/AppData/Local/Programs/Antigravity/resources/app.asar.unpacked/node_modules/chrome-devtools-mcp/build/src/daemon/daemon.js
argv:
  --viaCli
  --isolated
  --headless
  --executablePath=C:/Program Files/Google/Chrome/Application/chrome.exe
  --no-usage-statistics
  --no-performance-crux
  --filesystemRoot=<fresh absolute owned evidence root>
env:
  CHROME_DEVTOOLS_MCP_SESSION_ID=<fresh exclusively claimed UUID namespace>
  CHROME_DEVTOOLS_MCP_NO_UPDATE_CHECKS=1
  CHROME_DEVTOOLS_MCP_NO_USAGE_STATISTICS=1
cwd: project root
```

This is not an executed command. Native scope regex is `/^[a-fA-F0-9-]+$/`; require fresh nonempty UUID plus exclusive claim and namespace/runtime/PID/pipe collision checks. Collision means STOP, never adopt/delete another scope. Native utils derive Windows runtime home `os.tmpdir()/chrome-devtools-mcp-<scope>` with `daemon.pid`, and named pipe from `chrome-devtools-mcp-<scope>-<native username>/server.sock`. Use the native derivation rather than a custom transport. Scope/PID file is not a native browser session, authenticated capability or hostile same-user isolation boundary.

Construct the same exact minimum admitted environment for original owner and each caller: omit XDG_RUNTIME_DIR/XDG_ROOT and ambient session/config/provider overrides; TEMP/TMP designate the same approved owned temp parent and native username/OS path resolution. No environment or credential payload was inspected here. No personal/default Chrome, unsupported browser/debugging-port launch, shared daemon or native CLI startup wrapper.

The existing original-child owner launches/holds foreground native daemon once. The selected actor-facing surface is **only** installed exported `sendCommand(command,scope,finiteTimeout)` through each real actor's original one-shot Node/tool invocation. At most one tiny future wrapper imports this export, invokes one fixed command, preserves original envelopes and exits. It cannot implement sockets/MCP/CDP, start/ensure/reconnect the daemon, impersonate actor IDs, dispatch other actors or become a framework.

Typed selected commands are:

- `status`: original `{method:status}`; native awaits `started`; outer `success:true` plus result JSON containing daemon PID/socket/startDate/version/args must match original held daemon and accepted VERSION1.9.0 **before any actor browser call**.
- `invoke_tool`: original `{method:invoke_tool,tool:<fixed native method>,args:<typed native args>}`; preserve outer `{success,result}` and inner unmodified MCP envelope before formatting. Outer success does not imply inner tool success. Reject malformed/types/inner isError and preserve original error/timeout/caller terminal; never retry uncertain calls.
- Browser method allowlist for capability only: `list_pages {}`, `new_page {url:admitted nonce URL,timeout:finite}`, `take_snapshot {pageId:actual}`, `take_screenshot {pageId:actual,format:png,filePath:fresh confined}`, `close_page {pageId:exact owned}`. No product execute under this design.
- `stop`: original `{method:stop}` only after original identity recheck; ack means stop requested, not physical exit/cleanup. On identity loss, no status/tool/stop to a possibly foreign replacement; only exact retained original handle/object cleanup, otherwise UNKNOWN.
- Existing owner `waitOriginalHandle`: original daemon and each actual caller handle, original finite deadline; observed original wait/exit/stdout-stderr or timeout/unavailable/UNKNOWN. Fresh lookups cannot replace missing original terminal outcomes.

Native `status` identifies daemon state, not MCP/Chrome/actor/browser session. Exact PID+creation/executable/argv/parent/profile evidence independently binds daemon→MCP Node→Chrome. Browser creation is lazy: first native list_pages may launch Chrome; capture ownership before dependent parsing/actions. Sampling is qualified and never continuous no-relaunch enforcement.

## Admissions, provisional review and completion

1. **Current:** DESIGN/SPEC only. Independent adoption/source acceptance and a separately bounded WORK_ORDER precede any source BUILD. No live IDs, release, claim, accepted integrated build, new campaign or effect admission is created.
2. **Future technical native-interface effect admission:** separately commits exact native/source/tool/version pins, original owner and two real actors, fresh scope/resources, finite counters/deadline and cleanup for a capability lifecycle only. It precedes foreground launch/status/browser calls. It is distinct from product/provider authority.
3. **Provisional capability evidence:** producer A creates/controls/captures actual target; distinct observer B independently lists and observes that same target through its own original native caller tool/process return. Root may review this as a prerequisite, but it is not complete capability acceptance or R139 G0 PASS before required close/terminal evidence.
4. **Complete native-interface capability:** A closes exact target; B independently obtains final native inventory proving target absence/baseline preserved; actual one-shot callers retain original terminal receipts. Original owner stops exact native scope, waits original daemon handle, records terminal outcome and known daemon/MCP-child/Chrome PID+creation chronology/absence or UNKNOWN. Only then may independent review accept this bounded capability slice. Native last support blank may remain until daemon closes its owned browser. Stop ack, removed PID file, SDK close return, caller exit or daemon exit0 alone do not prove physical cleanup.
5. **Integrated product continuation:** NOT_DESIGNED/HARD_HOLD. A distinct post-prerequisite product effect admission would still be required but cannot cure the relaunch-enforcement and original-clock/held-context/terminal gaps. Do not insert a fake `G0_REVIEW_SEAL` before mandatory terminals or treat caller exits as final held-daemon/E06 evidence. A completed capability closes its context; later product creation is a separate newly admitted context, never inherited same-context proof.

Any later integrated design must preserve the existing normative R139 requirements (S18-S19,S23-S27,S30): one original admitted1200s campaign, cleanup+1020s/180s reserve, query-local UTC/monotonic agreement2s and query5s/same boot/pinned clock, through review, product admission, held-context use and cleanup. Native sealing cannot restart campaign time. This pass does not source-justify that integrated ordering, waive complete G0 terminals or open a new clock. All current/previous campaign caps and expired/spent receipts remain immutable.

Proposed counter classes remain separate: one foreground start/one native SDK child-connect; at most four status controls and one stop for a future bounded capability; browser-tool calls/captures/actual actor one-shot invocations; explicit CIM passes; native internal liveness checks; product/provider effects (zero under this capability scope). Each sendCommand performs native liveness; count it separately from external CIM observations. Future technical admission derives exact positive finite caps from the fixed whole sequence before launch; missing/unbounded/overrun cap HOLDs. Unknown/unsuccessful attempts consume, never refund/reset. No polling/ensure/version helper outside that sequence.

Cleanup finally must continue despite parser/evidence/close failures, using only exact original objects/handles/admitted ownership; known absence or UNKNOWN stays qualified, never global absence or arbitrary/user-browser kill. Durable source checkpoints reuse the existing ledger (S26): exclusive writer, exact source/contract/pins/inputs/receipt bindings, intent/cost persisted before effect. Orphan external IN_FLIGHT/unknown never resends, failed IN_FLIGHT persistence sends zero. Offline checkpoint reuse and product no-resend remain inherited requirements; no G1/G2 implementation is implied by this capability design.

## P01 immutable proposed current execution identity

Old S30 authority/S27 preflight retain Coree9ddcbcd/profile1.1. Current manifest/lock declare Core `15e39450853184eb65dbde73398c69c3171a6241` / profile1.2. The JSON proposes a **new immutable Root-owned successor**, not actual repin: current raw lock hash; LF-normalized profile `faaf9273ada3fd826e6372898d134fe7ad0157528f25b7720211c31cfc209bd5`, runner `716544aa7a38251009867a4e0150be51b79b4304a8c5f6d7afddab4e7c5d9c07`, bundle `80edcd2e4b605f440b94df963ed304ffed0f803bfa057a951360e39f014efc2b`; public remote; latest accepted componentfc4bbe01 and exact source/tool/tree metadata from S05, current component reconciliation S03. Native1.9.0 source/raw hash is separate. Inherited executable/package/tool pins were not independently queried here.

Preserve raw versus normalized hash conventions; original seeds/authorities/frozen artifacts/evidence and installed gate/profile/Core remain unchanged. Future versioned preflight retains workspace/public-origin/clean pinned Git-object/kit checks, legacy seven gates, current profile controls and exact packet/coverage/role applicability. N/A/shape agreement is not semantic PASS; doctor/component acceptance is not integrated product governance. No automatic fetch/upgrade/revert. Source compatibility, exact future integrated source/test/binary/tool bindings and all five findings remain open admission conditions.

S22 reports63PASS/3FAIL/2ERROR reproduced old baseline. Exact product applicability is **UNKNOWN** for all five: deep JSON interpreter-token expectation (both outcomes reject); missing private real-authority fixtures; absent public historical1.1 install path blocking genuine upgrade/generated-CI assertions; same absent path blocking edited-CI preservation assertions; missing parent intake-registry fixture blocking owner/digest/links. The JSON maps individual named cases and product relevance. No validator/tests beyond those33 sources were read, no private fixtures reconstructed, no broad v3 waiver or full-suite pass. Root/Core maintainers own exact path mapping before product readmission through the existing learning destination.

## Minimal source candidate and owned remainder

Immediate prospective v1 source targets are only `native_owned_command_v1.mjs` (one-shot native export/original envelope/caller terminal evidence) and `r139_current_pinned_preflight_v1.py.txt` (P01 immutable current-pin preflight), under the existing design probe directory. A separate Root WORK_ORDER must derive/fix exact minimal source diff, fixtures and existing owner invocation. Any demonstrated missing original-child/evidence owner connection must be named there; it is not a general manager grant.

Supervisor/Python tests/Go fixture and versioned activation/recorder/payload successors are **contingent future integration candidates**, not required v1 edits or admitted paths. G1/G2/IN_FLIGHT changes are not derived or authorized by P01/interface capability. Keep product router/engine/frontend, original seeds/evidence, Core/profile/gates and frozen artifacts unchanged. Root alone fixes future authority/accepted-build records.

Future offline cases T01-T17 cover exact/mixed pins, separate transport/context confusion, fake actor labels/log review, conditional real native caller shapes, forged CUA/session IDs, PID/profile/creation mismatch, shared/colliding scopes/autostart, original envelope/error handling, uncertainty/no resend, original clock expiry, writer/close failure, target/final inventory drift, provider/product claim inflation, Python/Go route parity, physical-relaunch race and premature completion/caller-versus-daemon terminals. They are proposed source-derived checks, not tests run or governance proof.

Full eight-row M3 coverage/owned remainder is inherited from current Root admission review: M3-01 priority/applicability, M3-02 permission/reason, M3-03 stale/competing, M3-04 role/SLA, M3-06 visible hold/lifetime implemented-unproved under Root/Product Owner current-source/G3 destinations; M3-07 notification applicability UNKNOWN owned by Root; M3-05/E01-E06 and M3-08 product browser/backend/viewport/offline acceptance BLOCKED. No waiver or missing criterion disappears. Future E01-E06 actual product/provider request-response-usage proof needs its distinct owner-agreed environment/synthetic resources/credentials/provider-model-endpoint/budget/caps/deadline/cleanup admission after a valid integrated design; no choices/probes/grants here.

Consolidated repair1 corrects selected API, separates technical/product admissions, removes premature G0 seal and unproved integrated continuation, makes physical-relaunch enforcement a hard product hold, narrows immediate paths and binds source VERSION1.9.0. Source count34 includes all prior33; no budget reset. Tranche conclusion: conditional native-interface/P01 source design is reviewable, full integrated design remains NOT_DESIGNED. Product conclusion: fullR139/M3/FREEZE OPEN; nativev3SPENT/browser23/v2cleanupUNKNOWN/old27source/4controllers/1campaign/5services/prioraudit17of16+Root2/allUNKNOWNs retained; Facebook/ZaloOA PARKED. Root owns ordinary technical adoption and remaining integrated design/enforcement; actual locked-acceptance changes go to Owner, not to a worker-invented waiver.
