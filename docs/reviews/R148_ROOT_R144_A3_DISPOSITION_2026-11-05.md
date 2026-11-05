# Root disposition: R144 A3 metadata and Windows package tree

Date: 2026-11-05
Risk: R2
Worker: Claude Code cloud documentary VERIFY worker
Reviewer: separate cloud Opus reviewer for the R144 comparison; separate R148 reviewer for the successor audit
Root/Local: ORCHESTRATOR, documentary audit author and commit steward; final closure follows independent R148 review

## Operator deviations

| ID | Root ruling | Basis and scope |
| --- | --- | --- |
| R144-D1 | ACCEPT for the worker's one project-repository fetch | The operator's current request expressly identifies D1 as granted. The worker disclosed one `git fetch origin main` to obtain the pinned project base; no npm/package fetch was used. Root's separate fetch of PR #17 for this review is a repository read under the current request. The R144 seed text remains immutable; this ruling is confined to the granted effects. |
| R144-D2 | ACCEPT for branch transport only | The operator's current request expressly identifies D2 as granted. Worker committed its return, opened PR #17 and did not merge. Root/Local retains merge and final closure. This does not generalize to future workers. |

The seed's zero-network and no-worker-commit clauses remain in its first committed bytes. These exceptional effects are visible rather than silently treated as seed compliance.

## Metadata result and review findings

The R144 worker's `METADATA_ONLY_CANDIDATE` and the separate review are accepted for the exact two JSON byte streams. Installed `package.json` SHA-256 is `592ef654a308edcba5fb3c2b302173650096a9487e4120b96e7adbe830429b9f`, 2,354 bytes. The verified npm 1.9.0 tarball's manifest SHA-256 is `6a14bfc51d9111957a5446ba861e462c3181c69c915f8133d8f2beac8c175021`, 4,556 bytes. Removing only top-level `bugs` and `scripts` and re-serializing with two-space indentation and no trailing newline reproduces the installed bytes. Runtime-relevant metadata fields are unchanged. Bundled yargs can read `package.json` at runtime, but its cited usage does not read either removed key (L2). The worker's approximate eight-minute return line conflicts with the 05:45:12Z worker commit: observed 05:42:32Z start to commit is about 2m40s; retain the overestimate as an inexact worker record (L1). Neither qualification changes the metadata verdict.

## Read-only Windows full-tree check

The PowerShell `Get-ChildItem -Recurse -File` / `Get-FileHash -Algorithm SHA256` capture from the current Antigravity `chrome-devtools-mcp` directory produced `windows_full_tree.json`. `root_windows_full_tree_compare.json` compares paths, byte lengths and hashes against PR #17's 348-file npm manifest. There are **347 installed paths**, **346 exact matches**, **one missing path `README.md`**, **no extras**, and **one common-path mismatch: `package.json`**, fully explained above. No file other than `package.json` has a hash mismatch. The operator's stricter condition that only `package.json` may differ fails because `README.md` is absent. Its absence is not claimed to affect the native runtime; its packaging cause is unknown.

The current installation therefore has a qualified metadata identity only. It is not a full-tree A3 PASS. A1/A2 Windows VERIFY is **not admitted**. A same-session complete tree and package pin would be required before any future execution, with independent review and separate effect authority.

## Seven R141 Local conditions retained

1. F3: compare post-start `status.args` with admitted argv; allowlist all three R139 session environment variables and resolve `XDG_RUNTIME_DIR`/`XDG_ROOT` concern.
2. F5: confirm the original Chrome handle remains alive before every command; retain the small hostile same-user race limit.
3. F6: correct four-versus-five wording in R141 spec section 8 and cite all N4 option conflicts in any executable successor.
4. N-R1: pin full page inventory at PRELIMINARY_OBSERVED, including Chrome's first tab; mutate only the admitted page ID; HOLD on unexpected inventory change.
5. N-R2: owner gates sequence; actors make their own original `sendCommand`, return unmodified envelope and terminal, and receive another release only after prior response review. Owner does not originate or replay actor calls.
6. N-R3: allow Chrome children descended from the original owner handle, reject unrelated foreign Chrome processes.
7. A1/A2 need separate finite Windows VERIFY; A3 five-file metadata discrepancy must be resolved and full-tree identity must satisfy the operator's current only-package.json condition before consideration.

## Claim and product boundary

R144 can close only as a finite documentary comparison with a qualified candidate. The successor R139 repair audit records `BLOCKED`, preserves both failed repairs and cumulative UNKNOWN costs, and admits no new action. R139 remains BUILD_BLOCKED; M3-05/M3-08 remain BLOCKED. No physical no-relaunch, same-context/original-clock, live governance, product, Chrome/MCP, browser, DB or provider result is asserted. Facebook/Zalo OA remains parked.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
