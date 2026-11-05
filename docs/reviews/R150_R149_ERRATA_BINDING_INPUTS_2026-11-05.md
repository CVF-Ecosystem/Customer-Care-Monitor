# R150 errata: binding inputs for any A1/A2 successor of frozen R149

Status: ERRATA_PENDING_INDEPENDENT_REVIEW
Tranche: CCMAI-RUNTIME-150
Risk: R2
Worker: Cloud ORCHESTRATOR acting as SPEC_AUTHOR (documentary errata only)
Reviewer: Independent Opus review subagent (pending)
Seed: `CVF_SESSION/authority/CCMAI-RUNTIME-150.json` (commit `4164a28d174da1625f4a4b2fa03414c95962cf25`)
Base: `main` at `6ded46c` (R149 FROZEN via PR #19)

## 1. Why this errata exists

Two independent reviews of R149 ran in parallel on different bytes.

| Path | Reviewed bytes | Result |
|---|---|---|
| Local (PR #19, merged) | build `d8e66c1` plus Local's two-file repair (A1 endpoint discovery and source-boundary citations) | Root first pass CHANGES_REQUIRED (C1, C4), then independent `/root/r148_independent_review` PASS. R149 FROZEN on `main` |
| Cloud (PR #18, not merged) | build `d8e66c1` | Opus subagent round 1 **CHANGES_REQUIRED** with B1–B3 blocking. Cloud repair `e448d5a`. Round 2 REVIEWER_ACCEPTED_PENDING_LOCAL with NF1–NF4 and partial items open |

Local's review states that the cloud review artifact was not on the remote when Local reviewed. That was true; it was pushed later. As a result, **none of B1–B3 and none of the cloud repair is on `main`.** Local's two corrections (A1 `/json/version` discovery, source-boundary citations) are on `main` and are kept.

Frozen bytes are not edited. Each item below is a **binding input**: any successor A1/A2 work order, and any later executable OHC contract, must incorporate it or record an independently reviewed rejection.

Evidence, copied byte-faithfully from PR #18:
- `docs/reviews/probes/r150_r149_errata_20261105/cloud_subagent_r149_review_ec86f91.md`: the full round-1 and round-2 subagent review. LF SHA-256 `f7813960…0bb9a`.
- `cloud_repaired_r149_contract_ec86f91.md`: a header comment was added and the coverage fence tag renamed so the copy is not read as a live packet. Original blob at commit `ec86f91`.
- `cloud_repaired_r149_draft_ec86f91.md`: verbatim.

## 2. Blocking errata (must be applied before any admission)

| ID | Location on `main` | Defect | Required correction |
|---|---|---|---|
| E-B3 (safety) | `R149_A1_A2_WINDOWS_VERIFY_WORK_ORDER_DRAFT_2026-11-05.md:29` | Tells the operator to run `chrome.exe --version`. On Windows this can start Chrome with the operator's default profile, which `R139_SDK_P01_B01_DESIGN_SPEC_2026-10-30.md:63` forbids | Read the version from file metadata only, for example `(Get-Item '<chrome.exe>').VersionInfo.ProductVersion`. Never execute `chrome.exe` outside the admitted A1 launches |
| E-B1a | `R149_OHC_SUCCESSOR_CONTRACT_2026-11-05.md:19` | "the later item wins" ranks the contract above the Root-selected lifecycle and R141 | The contract only adds requirements. The Root lifecycle and R141 N1–N9/§4.5 stay controlling, and the stricter requirement applies. The operator's A3 relaxation is the only deliberate relaxation, limited to §1 |
| E-B1b | contract §5, `:141-145` | Drops Root verification before PRODUCT_WINDOW, including **G1/E01–E03 before G2/E04–E06** | Restore that row text from the Root lifecycle in `R139_CAUSAL_CHAIN_ASSESSMENT_2026-11-05.md` |
| E-B1c | contract §5 | Drops TERMINAL → INDEPENDENT_ACCEPTANCE | Restore: only after TERMINAL is the integrated lifecycle and E01–E06 assessed independently; partial or unknown fields never become PASS |
| E-B1d | contract §5 | Drops R-F2's native CLI and `startDaemon` ban and its TERMINAL check | Restore: native CLI (`bin/chrome-devtools.js`), `start` and `startDaemon`/ensure are forbidden for the whole campaign; at TERMINAL, no other daemon socket or PID file for the session and no non-descendant Chrome on the owned user-data-dir |
| E-B2 | contract §2.2, `:68-73` | N4 conflict list incomplete | Add `autoConnect` ↔ `isolated`, `executablePath` (`config/browser-options.js:10`); `executablePath` ↔ `browserUrl`, `wsEndpoint` (`:89`); `userDataDir` ↔ `browserUrl`, `wsEndpoint`, `isolated` (`:99`); `channel` ↔ `browserUrl`, `wsEndpoint`, `executablePath` (`:105`); PWA category ↔ `autoConnect`, `browserUrl`, `wsEndpoint` (`config/category-options.js:37`), so the PWA category stays off; `mcp-options.js:90,95` experimental implications stay unset |
| E-NF1 | draft `:41` | "Raw WebSocket handshakes: at most 6" is below what the A1 procedure needs once every probe also captures the HTTP status and listener PID (E-M1 below), plus Local's `/json/version` discovery | Set a budget that covers every A1 step. The cloud review estimated 8–10 handshakes or upgrade requests plus discovery queries; Local fixes the exact number. Otherwise the run can only end INCONCLUSIVE after spending Chrome launches |

## 3. Non-blocking errata (apply, or record a reviewed reason not to)

| ID | Location on `main` | Correction |
|---|---|---|
| E-M1 | draft A1-b/A1-c (`:52-58`) | Node's built-in `WebSocket` cannot tell a refused upgrade from no listener. Each probe also sends a raw HTTP upgrade over TCP (Node `net`, no packages) and records the status line, plus the port's listening PID (`Get-NetTCPConnection -LocalPort P -State Listen`), which must be the current Chrome. A refusal counts only with a Chrome listener and a non-101 status |
| E-M2 | draft A1-c | Before launching Chrome #2, confirm P has no listener. After launch, confirm the listener on P is Chrome #2; otherwise INCONCLUSIVE with no retry. State how P is chosen |
| E-M3 | draft `:75`, contract §5 CLEANUP | Stop the daemon with the native `stop` command (`daemon/client.js:145-146` waits for PID-file removal). Wait on the daemon handle and confirm the MCP child has exited. **E-NF3:** add a bounded fallback (terminate the owned daemon tree by handle) if `stop` fails, and record it |
| E-M4 | contract §2.3 `:98` | `TEMP`/`TMP` must be the same fresh owned directory for the owner and every actor, because on Windows `os.tmpdir()` sets the PID-file directory (`daemon/utils.js:60-61`). Start the daemon with no Node flags before the script, and compare the OS process command line, because `status.args` cannot show Node flags. Record `LOCALAPPDATA`/`XDG_DATA_HOME` (usage statistics are disabled) |
| E-M5 | contract §3 | Under N-R2, each of the following is a HOLD: a missing or truncated envelope, an actor call timeout, an actor call made without an owner release, and a response not returned to the owner |
| E-NF2 | draft `:45` | Allow Node built-in `net` in addition to `WebSocket`/PowerShell |
| E-NF4 | contract §0/§3 | Name N-R2 explicitly as the approved replacement of R-F1's "owner relays" wording |
| E-A2 | draft A2-a/A2-c (`:62-66`) | A2-a also checks `status.pid` equals the owner's daemon child PID. A2-c states the accepted rejection shapes (an MCP input-validation error for the missing required `pageId`) and treats any other outcome as INCONCLUSIVE |
| E-L | contract `:101` and coverage block | Citation `daemon/utils.js:29-33` should be `:30-35`. The coverage wording "A3 identity accepted" should read "accepted subject to same-session recapture". Confirm `--headless=new` against the M3-08 viewport/raster qualification. Remaining L3 precision items are listed in the subagent review §Re-review |

## 4. What this does not do

- No edit to frozen R149 bytes, the repair audit, the registry or any work order.
- No admission, execution or R139 reopen. R139 stays BUILD_BLOCKED; M3-05/M3-08 stay BLOCKED.
- PR #18 is superseded by PR #19 and will be closed as superseded. Its content survives as the evidence above.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
