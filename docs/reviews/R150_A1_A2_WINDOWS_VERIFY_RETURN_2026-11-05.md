# R150 Windows A1/A2 VERIFY worker return

Status: COMPLETE_PENDING_REVIEW
Tranche: CCMAI-RUNTIME-150
Risk: R2
Worker: Root/Local Windows VERIFY worker
Reviewer: independent R2 reviewer separate from the worker
Official order: `docs/work_orders/CCMAI_RUNTIME_150_A1_A2_WINDOWS_VERIFY.md` (LF SHA-256 `a7a2b74e1851cdcbd2dba010456e2b0d17a9ade1fdb83518e19d66343a79e398`)

## Result

**INCONCLUSIVE.** A3 full-tree recapture passed immediately before Chrome #1. A1-a exact-endpoint WebSocket opened and returned `Browser.getVersion`. The wrong-GUID A1-b WebSocket did not open; Node reported `Received network error or non-101 status code.` and the probe timed out without a recorded HTTP 4xx or close-before-open event. Under the work order this is an ambiguous refusal class, so the helper stopped without A1-c or A2. No retry or relaxed interpretation was used.

## Retained evidence

- `docs/reviews/probes/r150_a1_a2_windows_20261105/a3_prestart_full_tree.json` — raw SHA-256 `4725d732d884494e32700f88a744b95e955b78bb410dbd91a5911e1fdb65c230`; PASS, 347 installed files against 348 manifest files, only `README.md` absent, `package.json` SHA-256 `592ef654a308edcba5fb3c2b302173650096a9487e4120b96e7adbe830429b9f`, all 346 other file sizes and hashes equal, no extras.
- `docs/reviews/probes/r150_a1_a2_windows_20261105/windows_verify_raw.json` — raw SHA-256 `ec31273f3eb9629a7cfcd8b9e24c2f5ec02fb541fe76e6d1aca26f518b3529e0`; session `84c3904e-779f-4232-bab0-a04b9c0adf4f`, Chrome #1 PID 40468, fixed port 58837, endpoint from one localhost `/json/version` query. The valid control returned Chrome/156.0.8078.81. The wrong GUID changed one hex digit; it did not open and timed out at 10 seconds after the non-101 error event.
- Versions: Chrome `156.0.8078.81`, Node `v22.17.0`, Windows build `10.0.26200`, Antigravity `2.29.0`.
- Effect ledger: 1/3 Chrome launches; 0/1 daemon starts; 0/8 native calls; 2/6 raw WebSocket handshakes; 1/2 localhost endpoint queries; active elapsed 22.810 seconds. Provider, DB and product calls: 0.
- Original Chrome handle was closed and waited (`SIGTERM`). The terminal snapshot reported no owned-profile Chrome, MCP child, daemon or daemon PID file. Independent raw inspection found that **all** recorded Chrome root/descendant `ownedProfileArg` flags were false even though the launch argv specified the owned profile. Therefore that terminal owned-profile predicate does not prove exact Chrome descendant cleanup; this field is **UNKNOWN**. A later read-only query at `2026-11-05T07:45:27Z` found none of the twelve recorded root/descendant PIDs, as retained in `docs/reviews/probes/r150_a1_a2_windows_20261105/postrun_known_pid_readback.json`; it does not retroactively repair terminal evidence. The helper retained the original argv, environment key list, endpoint, raw WebSocket events, process snapshots and wait result.

## Boundaries and next move

A1 GUID rejection remains **UNPROVEN** under the packet's evidence class; A2 was **NOT RUN**. This result does not reopen R139 or unblock M3-05/M3-08. The authorized campaign is spent. A follow-up would require a newly reviewed finite work order and separate effect admission that specifies how to capture the actual HTTP handshake response without relaxing the A1 assertion. Integrated lifecycle remains unadmitted; Facebook/Zalo OA remains parked.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
