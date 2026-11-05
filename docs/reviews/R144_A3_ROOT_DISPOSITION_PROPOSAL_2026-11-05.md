# R144 A3: Root disposition proposal (cloud ORCHESTRATOR, pending Local)

Status: PROPOSAL_PENDING_LOCAL
Tranche: CCMAI-RUNTIME-144
Risk: R2
Worker: Claude Code cloud VERIFY worker (R144 comparison; separate from the reviewer)
Reviewer: Independent Opus review subagent (REVIEWER_ACCEPTED); Local remains the closing Root reviewer
Author: Cloud ORCHESTRATOR acting under the operator's full-roadmap grant ("toàn quyền xử lý theo roadmap…", 2026-11-05 cloud session). Not the reviewer.
Inputs:
- worker return `docs/reviews/R144_A3_PACKAGE_METADATA_VERIFY_RETURN_2026-11-05.md` (commit `a9df2da`)
- independent review `docs/reviews/R144_A3_PACKAGE_METADATA_INDEPENDENT_REVIEW_2026-11-05.md` (REVIEWER_ACCEPTED)

## 1. Why this is a proposal and not a closure

The active registry audit `repair_chain_assessment_a3_verify.json` pins the R144 work order bytes (`0f60ed08…`) as evidence and as `admittedAction.packet`.

- A cloud trial that changed only the work-order `Status:` line produced `COV_EVIDENCE_STALE`. The trial was reverted and never committed.
- So R144 cannot move to REVIEW_PASS / REVIEWER_ACCEPTED without a successor audit that moves the registry pointer.
- The work order reserves that to Root/Local ("Only Root can … update the R139 causal audit"), and the R144 seed forbids registry edits.

The cloud orchestrator therefore leaves the work order, tranche record, registry and audit unchanged and hands closure to Local.

## 2. Proposed disposition

1. **Accept the worker verdict METADATA_ONLY_CANDIDATE**, independently confirmed.
2. **Qualified A3 pin**, valid only for the captured Windows state:
   - installed `package.json` raw SHA-256 `592ef654a308edcba5fb3c2b302173650096a9487e4120b96e7adbe830429b9f`, defined as the npm `chrome-devtools-mcp@1.9.0` `package.json` (tarball `sha512-RnzXoJiU…ut9kSQ==`) minus `bugs` and `scripts`, re-serialized with indent 2 and no trailing newline;
   - plus the 19 Windows-hashed JS files (15 R141, 4 A3).
3. **Not A3 PASS yet.** Per the reviewer's I1, 59 of 78 `build/src` JS files, the non-JS files and `skills/` have no Windows hash. A **full-tree Windows hash capture** must match the npm manifest (§4) before any A1/A2 admission. The pin must be re-verified immediately before execution, because an Antigravity update can change the install.
4. **Findings for Local to rule on:**
   - **M1:** R144-D1 (one project-repo `git fetch`) and R144-D2 (worker branch commit and PR, no merge) conflict with the R144 seed and are disclosed. Grant source: the operator's messages in the cloud session ("Tôi cho phép, quy tắc cũ, bạn có quyền làm, nhưng không được merge", meaning "I allow it, old rules, you may do it, but no merge"). Local should confirm with the operator.
   - **L1:** the return's time line overstates the end time. Actual authorized work ran from 05:42:32Z to the worker commit at 05:45:12Z, about 3 minutes, inside the 20-minute ceiling. The reviewed return bytes are left unchanged; this note is the correction.
   - **L2:** the bundled yargs reads only `version` and named fields from `package.json`, never `bugs` or `scripts`. The verdict is unaffected.
5. R139 stays BUILD_BLOCKED and M3-05/M3-08 stay BLOCKED. No A1/A2, no-relaunch, M3 or runtime-governance claim.

## 3. What Local must do (cloud cannot)

1. Author a successor audit with independent review (as in R145–R147) that records the R144 outcome and moves the registry pointer, then close R144 formally.
2. Run the full-tree capture on Windows (§4) and compare it to the manifest.
3. Only then consider the A1/A2 Windows VERIFY admission with the seven R141 Local conditions.

## 4. Full-tree comparison aid (Root-preparation evidence)

`docs/reviews/probes/r144_a3_package_metadata_20261105/root_npm_full_tree_manifest.json` lists all **348** files in the verified npm tarball with byte size and SHA-256: 331 under `build/`, 78 of them `build/src/**/*.js`; 14 under `skills/`; plus `LICENSE`, `README.md` and `package.json`. It was read in memory from the retained tarball, with nothing executed and no network use.

**Read-only Windows capture** (PowerShell, from the package directory; no execution of package code):

```powershell
$root = "$env:LOCALAPPDATA\Programs\Antigravity\resources\app.asar.unpacked\node_modules\chrome-devtools-mcp"
Get-ChildItem -LiteralPath $root -Recurse -File | ForEach-Object {
  [pscustomobject]@{ path = $_.FullName.Substring($root.Length + 1).Replace('\','/'); bytes = $_.Length; sha256 = (Get-FileHash -Algorithm SHA256 -LiteralPath $_.FullName).Hash.ToLower() }
} | Sort-Object path | ConvertTo-Json -Depth 3 | Set-Content -Encoding utf8 windows_full_tree.json
```

**Expected classification:**
- `package.json` differs (explained by R144).
- Every other npm path is present with equal hash.
- List every extra or missing path separately; an Antigravity packaging step may add or remove files.
- Anything else means the A3 identity stays unresolved.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
