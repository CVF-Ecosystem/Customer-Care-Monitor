# DRAFT work order: finite Windows A1/A2 VERIFY for OHC (for Local admission)

Status: DRAFT
Drafted under: CCMAI-RUNTIME-149 (cloud ORCHESTRATOR, documentary only)
Risk: R2

Risk note: when admitted, the risk is R2 or higher, because it starts Chrome and the native daemon on the operator's machine.
Worker: Cloud ORCHESTRATOR as SPEC_AUTHOR of this draft (not the executor)
Reviewer: Independent Opus review subagent for the draft (pending); Local independent reviewer at admission

This is a **draft, not a dispatch.** It becomes executable only after Local completes all of the following:
- copies it into its own tranche and work order;
- records a successor R139 repair audit whose `discriminator` is the question in §1 and whose `admittedAction` pins that work order;
- obtains independent admission review;
- passes the pre-dispatch gate;
- grants a separate effect admission for the Chrome and daemon starts below.

Nothing here admits A1/A2.

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

**A1**

- **A1-a, control.** Launch Chrome #1 with the R149 §2.1 flags but `--remote-debugging-port=<fixed free port P>`. Read `DevToolsActivePort`. Open a raw WebSocket to the exact endpoint.
  - Expect: handshake succeeds.
  - Then send `Browser.getVersion` and record the reply.
- **A1-b.** Open a raw WebSocket to `ws://127.0.0.1:P/devtools/browser/<guid with one hex digit changed>`.
  - Expect: handshake refused (HTTP 4xx or connection close before open).
- **A1-c.** Close Chrome #1 through its handle and wait on it. Launch Chrome #2 on the same port P. Record its new GUID. Open a raw WebSocket to Chrome #1's old endpoint.
  - Expect: refused.
  - Then connect to Chrome #2's endpoint. Expect: success (control).
- **A1-d (optional, only if A1-c is inconclusive).** With no Chrome on P, connect to the old endpoint.
  - Expect: connection error.

**A2** (uses Chrome #2, or Chrome #3 if A1-c was skipped)

- **A2-a.** Start the daemon exactly as in R149 §2.2 with `--wsEndpoint` set to Chrome #2's endpoint. Call `status` through the native `sendCommand`.
  - Expect: `args` equals the admitted argv; `version` is `1.9.0`.
- **A2-b.** Producer `new_page` with `about:blank`. Observer `list_pages`. Record the full inventory (N-R1).
- **A2-c.** Call `take_snapshot` **without** `pageId`.
  - Expect: an input-validation error.
  - Expect: no change to the page or the inventory, verified by a following observer `list_pages`.
- **A2-d.** Call `take_snapshot` with the admitted `pageId`.
  - Expect: success (control).
- **A2-e (optional).** Call `take_snapshot` with a non-existent `pageId` (max + 1000).
  - Expect: an unknown-page error.

**Cleanup**

1. Stop the daemon through its handle and wait on it.
2. Close the last Chrome through its handle and wait on it.
3. Confirm that no Chrome process uses the owned user-data-dir and no daemon PID file remains for the session.

## 4. Evidence to retain (secret-free)

- The A3 recapture JSON and its comparison against the manifest.
- Versions.
- Exact argv and the environment key list (values are allowed only for the three admitted variables).
- Chrome GUIDs and ports.
- For each WebSocket handshake: the target URL and the outcome (status code or close reason).
- For each native call: the raw `sendCommand` envelope.
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

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
