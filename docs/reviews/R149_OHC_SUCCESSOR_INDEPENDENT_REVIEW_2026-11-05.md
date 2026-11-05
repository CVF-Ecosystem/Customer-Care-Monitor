# R149 OHC successor: independent R2 review

Status: REVIEW_PASS_DOCUMENTARY_ONLY
Risk: R2
Worker: Root/Local repair worker for the two post-build document corrections; original SPEC_AUTHOR was the cloud ORCHESTRATOR
Reviewer: independent `/root/r148_independent_review`, separate from both document authors
Reviewed base: `d8e66c1184a888cbfeda66ba064332f6442724aa` plus the two-file worktree repair shown by `git diff`; R149 seed: `abe76f66cb46978cc242bc1a3fdb6c0fb96a4577`.

## Decision

**PASS C1–C6 for a documentary successor contract and DRAFT Windows A1/A2 VERIFY packet.** The Root first-pass C1 and C4 findings are corrected in the reviewed worktree. The contract and packet remain non-executable drafts; this review does not admit or run A1/A2. Root/Local still owns any successor R139 repair audit, finite Windows effect admission, pre-dispatch review and merge. R139 remains `BUILD_BLOCKED`; M3-05/M3-08 remain `BLOCKED`.

## Evidence and acceptance

| Criterion | Independent result |
| --- | --- |
| C1, seven Local conditions | PASS. F3 binds exact `status.args` and all three R139 session variables; F5 checks original Chrome and daemon handles before each release; F6 corrects five-file wording and cites N4 conflicts; N-R1 pins full inventory including the initial tab; N-R2 keeps each actor's original `sendCommand` and unmodified envelope while the owner only sequences; N-R3 permits descendants of the original Chrome and rejects unrelated processes; A1/A2 remain a separate finite Windows discriminator. The new F5, N-R1, N-R3 and A1/A2 citations state where native source ends and owner-side enforcement begins. |
| C2, A3 identity | PASS at design scope. The operator-relaxed predicate is exact: installed `package.json` equals the R144 explained hash, `README.md` is absent, 346 other paths match in size and SHA-256, and no extras appear. A same-session recapture immediately before any execution is required; any drift means NO_START. This is a declared operator decision, not a fresh Windows runtime result. |
| C3, lifecycle/source consistency | PASS for the documentary contract. Connect-only `wsEndpoint`, one owner-held Chrome and daemon for the eventual product campaign, owner sequencing, full page inventory and original clock remain aligned with R141/R139. The A1 test draft may launch Chrome more than once because it is a separate VERIFY campaign; it is not the product campaign. |
| C4, finite draft | PASS. The draft has a 45-active-minute ceiling, at most 3 Chrome launches, 1 daemon start, 8 native calls and 6 raw WebSocket handshakes; localhost-only network, `about:blank` pages and zero provider/DB/product calls. It states preconditions, stops, cleanup, evidence and `A1_A2_PASS`, `A1_FAIL`, `A2_FAIL`, `INCONCLUSIVE` outcomes. Fixed-port Chrome discovery now first reads `DevToolsActivePort` if present, then permits only `http://127.0.0.1:P/json/version`, validates port and browser GUID path, and stops inconclusive if neither works. A1-c uses the same bounded rule and requires a different GUID. |
| C5, effects/claims | PASS in the reviewed repository diff: only the contract and draft were repaired, `git diff --check` is clean, and no repair audit, registry, product source or executable work order changed. No Chrome, daemon, browser, provider or product action was performed by this reviewer. Static file inspection cannot prove the absence of effects outside this worktree. |
| C6, independent review | PASS by this separate R2 review. Root's first-pass `CHANGES_REQUIRED` record and the exact two-file repair remain visible; this is a new judgment on the repaired bytes, not a rewrite of that finding. |

## Direct installed-source verification

I read the current Windows `chrome-devtools-mcp@1.9.0` files in place and compared SHA-256 for ten cited JS files to the retained npm full-tree manifest; **10/10 matched**. I did not import or execute the package. Relevant checks:

- `daemon/client.js:105-109` checks daemon liveness and throws `Daemon is not running.`; `sendCommand` itself does not start a daemon. It does **not** check the original Chrome handle, which the contract assigns to the owner (F5). `daemon/daemon.js:87,94-98,136-147` forwards exact argv and environment to the MCP child and returns `status` with `pid`, `version`, `socketPath` and `args`.
- `daemon/utils.js:29-34` selects a Windows named pipe before considering `XDG_RUNTIME_DIR` for the socket, but `:45-62` checks `XDG_RUNTIME_DIR` before the Windows PID-home fallback. Requiring that variable unset is justified. `XDG_ROOT` was not found in the package's direct environment reads; it remains excluded by the R139 rule.
- `browser.js:38-40,89-101` passes the supplied `wsEndpoint` to `puppeteer.connect` and throws on connect failure; it does not prove Chrome's GUID rejection. A1 must observe that on Windows. `tools/pages.js:13-27,90-117` defines `list_pages` and `new_page`; full inventory and mutation limits are owner-side conditions (N-R1).
- `tools/ToolDefinition.js:16-24,32-34` marks page-scoped tools and defines required numeric `pageId`; `ToolHandler.js:134-141,183-203` attaches that schema and handles a page call. `tools/snapshot.js:8-10` confirms `take_snapshot` is page-scoped. The bundled SDK validates tool input before executing the handler (`third_party/index.js:39070-39071,39100-39112`). This supports the A2 hypothesis but does not replace the Windows test.
- `config/browser-options.js:23,41,64` confirms `browserUrl`/`wsEndpoint` conflict and `wsHeaders` implication; `config/mcp-options.js:109,114` confirms the allowed/blocked URL-pattern conflict.

## Admission notes and limits

The A2 draft's following `list_pages` can detect an inventory change, but by itself cannot prove that the read-only `take_snapshot` handler was never invoked. At admission, Local should require the raw SDK validation-error envelope matching the source ordering above, plus the retained observer call, before accepting A2. The fixed-port `/json/version` request is bounded to endpoint discovery for Chrome #1 and #2 and should be counted explicitly in the future effect ledger. Cleanup must verify the MCP child as well as the daemon and Chrome; `daemon/daemon.js:209-250` has a cleanup path, but a forced Windows termination is not itself proof that it ran. These are execution-evidence requirements for Local's later packet, not an A1/A2 result or a grant to start processes.

The contract's general “later item wins” sentence cannot authorize a waiver of the R139 hard hold or the seven Local conditions. I found no present conflict in the reviewed operative clauses. Any actual conflict during Local admission requires explicit reconciliation under the existing authority.

Installed-profile `reviewer-fast` passed on the repaired worktree; APPL and ROUTE were `NOT_APPLICABLE_WITH_REASON`, while COV checked the contract's PLAN block. The public Core doctor retains only the known public1.2/installed1.3 INST/COV version mismatches; trusted installed1.3 bootstrap passed. These are static artifact checks, not live AI governance, A1/A2 behavior, physical no-relaunch, M3 acceptance or hosted CI proof.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
