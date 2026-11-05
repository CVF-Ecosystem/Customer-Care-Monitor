# R150 independent static review: Windows A1/A2 probe helpers

Status: REVIEW_PASS_STATIC_ONLY
Risk: R2
Reviewer: independent `/root/r148_independent_review` (no helper authorship or Windows effect execution)
Worker: Root/Local Windows VERIFY worker
Scope: `docs/reviews/probes/r150_a1_a2_windows_20261105/owner_runner.mjs`, `actor_call.mjs`, `prestart_full_tree.ps1`, `process_snapshot.ps1`

## Decision

**PASS** the four helpers for a bounded, conditional R150 A1/A2 Windows VERIFY. This is a source review only. The helpers must be committed with the admitted audit, registry and effect admission before invocation; their own committed-file guard currently rejects the uncommitted worktree. The actual process start still depends on the immediate A3 full-tree recapture returning PASS. No A1/A2 outcome or product admission follows from this review.

## Evidence and checks

1. **Pre-effect authority.** `owner_runner.mjs` constrains project, installed package, manifest, Chrome, output and temp to fixed canonical paths; pins the R144 manifest LF hash; checks the exact work-order, audit and independent audit-review hashes against the effect admission and M3 registry pointer; checks active R150 continuity and installed pre-dispatch RESULT PASS. Its Git status guard covers the admission, state, registry, three pinned documents and all four helpers, so uncommitted changes to those inputs stop before a process start. A clean status does not itself establish hosted CI or runtime behavior.
2. **A3 NO_START.** `prestart_full_tree.ps1` requires the fixed package/manifest/output paths and pinned manifest hash. It rejects package-root, file and directory reparse points. It enumerates the complete installed file tree, requires 347 actual against 348 expected, only `README.md` missing, no extra paths, 346 other files equal in byte size and SHA-256, and installed `package.json` SHA-256 `592ef654a308edcba5fb3c2b302173650096a9487e4120b96e7adbe830429b9f`. Only its exit zero permits the first Chrome launch; failure produces `NO_START`.
3. **Finite effects.** The owner uses Node's built-in WebSocket and an allowlisted environment with a new session ID, owned temporary profiles and evidence root. It debits at most 3 Chrome launches, 1 daemon start, 8 native calls, 6 raw WebSocket handshakes and 2 localhost endpoint queries. It checks the 45 active minute ceiling before effects and again before terminal PASS. Chrome receives only `about:blank`; endpoint fallback is restricted to `127.0.0.1` and the selected port. `actor_call.mjs` admits only the named native `sendCommand` methods, with `new_page` fixed to `about:blank` and `snapshot_valid` constrained to an integer page ID.
4. **A1/A2 classification.** A1-a requires a successful exact-endpoint WebSocket and `Browser.getVersion`. A1-b changes one GUID digit, requires close before open, and now repeats the exact-endpoint control. A1-c closes/waits for Chrome #1, launches Chrome #2 on the same port with a different GUID, requires close before open on the stale endpoint, then succeeds on Chrome #2's endpoint. Original Chrome handles and scoped process snapshots are checked around negative probes; unknown Chrome command lines stop classification. A2 checks native daemon PID/version/argv, full producer/observer page inventory, the raw failed `take_snapshot` response with the SDK input-validation prefix and `pageId`, unchanged following inventory, then a successful valid-page control. Missing or ambiguous evidence yields `INCONCLUSIVE`, not PASS.
5. **Cleanup and evidence.** The owner records versions, environment keys, effect counters, exact process arguments, endpoint URLs, WebSocket results, original handle waits, raw native envelopes and terminal process snapshots. It attempts native daemon stop, waits or terminates original handles, and downgrades PASS if owned Chrome, daemon handle/PID, MCP child or daemon PID file remains. `process_snapshot.ps1` reports process PID, parent PID, creation time, executable path and scoped profile match without copying profile contents.

The reviewed raw SHA-256 values are: `owner_runner.mjs` `a8c47af6a50f43e4532640afbbc359ea0cf0bd9d7ad3d6c14bdc2d50ce8f11f8`; `actor_call.mjs` `e8809991c2d26a0ff8dc83665cc4ac93aa199221164ae3d4665a26f30bb7a8c1`; `prestart_full_tree.ps1` `079494fff2685a60a1e7c8228971775b085e8f659548e660b0a22f50fc073c78`; `process_snapshot.ps1` `a7d682b6e75195f90014fed4f64540977ed83a7fa1ca846a011bbb2252627fba`.

## Limits

This was a read-only code and contract review. I did not invoke the helpers, Chrome, daemon, native tool, provider, database or product. Git status showed the R150 packet, admission and helpers were still uncommitted during review, so the owner guard would stop at this point. Runtime process ownership, Windows WebSocket close behavior, actual A3 recapture and cleanup must be judged from the retained worker evidence after an authorized run. R139 stays `BUILD_BLOCKED`; M3-05/M3-08 stay `BLOCKED`. An A1/A2 PASS would only remove a prerequisite for later consideration of a separately admitted integrated campaign.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
