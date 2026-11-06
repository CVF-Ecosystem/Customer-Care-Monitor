# R150 independent review: Windows A1/A2 VERIFY return

Status: REVIEW_PASS_INCONCLUSIVE_DIAGNOSTIC_ONLY
Risk: R2
Worker: Root/Local Windows VERIFY worker
Reviewer: independent `/root/r148_independent_review`, separate from the worker and audit author
Official work order: `docs/work_orders/CCMAI_RUNTIME_150_A1_A2_WINDOWS_VERIFY.md` (LF SHA-256 `a7a2b74e1851cdcbd2dba010456e2b0d17a9ade1fdb83518e19d66343a79e398`)
Return reviewed: `docs/reviews/R150_A1_A2_WINDOWS_VERIFY_RETURN_2026-11-05.md` (LF SHA-256 `820c0035793c3e5b1eaa5c155f108b5403798c34060d148d5992d26fddfb7d10`)
Worker evidence commit: `7052df5c6da31d6867172d26a298a7bb4b72bb9c`

## Decision

**PASS the return as an honest, bounded `INCONCLUSIVE` diagnostic**, with Chrome descendant cleanup at the terminal observation explicitly `UNKNOWN`. This is not an A1 or A2 PASS. The single admitted campaign is spent. There is no basis to rerun it under R150, reopen R139, admit an integrated lifecycle, or unblock M3-05/M3-08. A later bounded diagnostic would need a separately reviewed packet and effect admission, including a method to retain the actual HTTP upgrade response and a repaired process ownership predicate.

## Independent evidence checks

1. **A3 prestart.** The raw `a3_prestart_full_tree.json` SHA-256 is `4725d732d884494e32700f88a744b95e955b78bb410dbd91a5911e1fdb65c230`, equal to the return's pin. I compared its 347 file entries against the pinned 348-entry R144 npm manifest independently: only `README.md` is missing, there are no extra entries, the other 346 files match both byte size and SHA-256, and installed `package.json` has the operator-admitted SHA-256 `592ef654a308edcba5fb3c2b302173650096a9487e4120b96e7adbe830429b9f`. The receipt verdict and exit status are PASS/0. Its event at `07:42:18.316Z` precedes the only Chrome start at `07:42:18.331Z`.
2. **Authority and helper identity.** The effect admission pins the official work order, R139 successor audit and independent audit review, and limits the worker to this finite Windows VERIFY. The raw runner recorded installed pre-dispatch RESULT PASS with six mandatory controls and ROUTE not applicable. Current helper raw SHA-256 values equal those in the prior independent static review: owner `a8c47af6a50f43e4532640afbbc359ea0cf0bd9d7ad3d6c14bdc2d50ce8f11f8`, actor `e8809991c2d26a0ff8dc83665cc4ac93aa199221164ae3d4665a26f30bb7a8c1`, A3 script `079494fff2685a60a1e7c8228971775b085e8f659548e660b0a22f50fc073c78`, process snapshot script `a7d682b6e75195f90014fed4f64540977ed83a7fa1ca846a011bbb2252627fba`.
3. **Effects and scope.** The raw `windows_verify_raw.json` SHA-256 is `ec31273f3eb9629a7cfcd8b9e24c2f5ec02fb541fe76e6d1aca26f518b3529e0`, equal to the return's pin. Its event list has one Chrome start, two WebSocket handshakes and one localhost `/json/version` query; no daemon start or native call. Counters are 1/3 Chrome, 0/1 daemon, 0/8 native, 2/6 WebSocket and 1/2 endpoint queries; active time is 22.810 seconds against 45 minutes. The recorded Chrome argv uses the fixed port, owned temporary profile, headless mode and `about:blank`; the observed helper targets are `127.0.0.1`. No provider, database or product call appears in the helper ledger.
4. **A1 classification and A2 boundary.** A1-a opened the exact endpoint and received `Browser.getVersion` for Chrome/156.0.8078.81. A1-b changed exactly one GUID hex digit. The negative WebSocket did not open, but it timed out after Node's generic `Received network error or non-101 status code.` without a recorded HTTP 4xx or close-before-open event. The helper correctly returned `INCONCLUSIVE` at A1-b. There is no A1-c, daemon or A2 event, so A1 GUID rejection is unproved and A2 was not run. The raw event sequence shows no retry or relaxed classification.
5. **Cleanup qualification.** The original Chrome #1 handle, PID 40468, received `SIGTERM` and returned a wait result with `signalCode: SIGTERM`; no daemon or MCP child had been started, and no daemon PID file remained. However, every in-campaign Chrome root/descendant process row reports `ownedProfileArg: false`, including PID 40468 whose launch argv specified the owned user-data-dir. The terminal `leftoverOwnedChrome: []` therefore cannot prove the descendants were absent at that instant. The corrected worker return states this as `UNKNOWN`. `postrun_known_pid_readback.json` (raw SHA-256 `33b7e6078144e6cfbfb721ba424b1a147e2c714790d15136cd4de2b161ebadac`) found none of the twelve recorded PIDs at `07:45:27Z`; this later read-only observation does not repair the terminal predicate.

## Limits and disposition

I reviewed retained files and made no Chrome, daemon, native, provider, database or product call. I had previously reviewed the helpers statically, but did not author them or execute the campaign; the observed false ownership flags narrow that earlier prospective cleanup assessment and are retained here rather than hidden. The receipt does not independently measure Chrome background network traffic or prove physical no-relaunch behavior. Root committed the exact return, raw receipts, later PID readback and synchronized continuity at `7052df5c6da31d6867172d26a298a7bb4b72bb9c` before this verdict. R139 remains `BUILD_BLOCKED`, M3-05/M3-08 remain `BLOCKED`, and Facebook/Zalo OA remains parked.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
