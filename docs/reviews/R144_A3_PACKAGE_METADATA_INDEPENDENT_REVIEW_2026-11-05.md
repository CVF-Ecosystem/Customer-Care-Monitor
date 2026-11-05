# R144 A3 package metadata VERIFY: independent R2 review

Status: REVIEWER_ACCEPTED
Tranche: CCMAI-RUNTIME-144
Risk: R2
Worker: Claude Code cloud VERIFY worker (separate agent from this reviewer)
Reviewer: Independent Opus review subagent appointed under operator grant (same cloud session; independence limitation disclosed)

Reviewed range: `5c43dfdff2f6fd99634fd3485c098acea1bd2a43..a9df2da4b10dfff2c559f69b8e8fe76ef9325f00` on branch `claude/friendly-rubin-e2npmt-r144`.
Work order: `docs/work_orders/CCMAI_RUNTIME_144_A3_PACKAGE_METADATA_VERIFY.md`.
Worker return: `docs/reviews/R144_A3_PACKAGE_METADATA_VERIFY_RETURN_2026-11-05.md`.

**Verdict:** The worker's `METADATA_ONLY_CANDIDATE` is correct. I reproduced every source hash, the field-level diff and the byte-exact re-serialization on my own. I found no defect against the work order. The findings below are LOW or INFO and need Root rulings or record corrections; none requires a worker repair.

## 1. Independence limitation

- I did not do the comparison and did not write the return or `worker_compare.json`. I wrote my own scripts in a separate scratch directory (`scratchpad/r144_review_scripts/`) and re-derived every value from the raw bytes.
- I run in the same cloud session, on the same model family, as the worker. I share its scratchpad, including the retained R141 tarball, so I cannot independently confirm the tarball's provenance beyond its integrity hashes. I also cannot see the operator messages behind R144-D1 and R144-D2.
- The authority seed names Root/Local as reviewer. The operator appointed this subagent because Root/Local is not in this session. Root/Local should treat this as a separate R2 review, not a substitute for its own commit-steward check, as it did for R140.
- This file is the only file I wrote. I made no commit, push, merge, network call, install, import, execution, or Chrome/MCP/browser/DB/provider action.

## 2. Verifications (my own observed values)

### V1. Work order LF SHA-256

| Check | Observed |
|---|---|
| Worktree bytes / CRLF count | 17,359 bytes / 0 CRLF (already LF) |
| Raw SHA-256 = LF-normalized SHA-256 | `0f60ed08a21e77a5b9bdc3b11db57b420f640d5e2ced4b4668ff608a11887a2e` |
| Git blob at `5c43dfd` and at `a9df2da` | same hash, so the work order is unchanged in the range |

Result: **MATCH**.

### V2. Retained R141 tarball and npm `package.json`

Path: `/tmp/claude-0/-home-user-Customer-Care-Monitor/ec425cff-14b7-536e-b408-80d75d5738bd/scratchpad/cdm_tgz/cdm-1.9.0.tgz`. The file mtime is `2026-11-04T20:22:17Z`. That is 4 minutes before the R141 `source_verification.json` commit `19018e2` at 20:26:17Z.

| Check | Expected (`source_verification.json`) | Observed |
|---|---|---|
| Bytes | n/a | 2,458,275 |
| sha1 | `registryShasum` `86bf2bde0bc4ec07aa428b36ea612eaa7a37a77c` | `86bf2bde0bc4ec07aa428b36ea612eaa7a37a77c` (MATCH) |
| sha512 integrity | `registryIntegrity` `sha512-RnzXoJiUQ44hpOihWk90uOhLD/CnwDkDy0ldHMZONJ2nYQ+dWN1fq1luHHqyd+7FuYnyIlCY6uTThbN5ut9kSQ==` | identical (MATCH) |
| `package/package.json`, read in memory with Python `tarfile` | `6a14bfc51d9111957a5446ba861e462c3181c69c915f8133d8f2beac8c175021` | same hash (MATCH); 4,556 bytes, 117 LF, 0 CRLF, trailing newline, no BOM |

Extra cross-check: from the same tarball in memory, all 15 R141 inventory files and all 4 A3 files from `local_capture.json` hash to their recorded Windows values (19/19 MATCH).

### V3. Installed Windows capture

`installed_package.json` raw SHA-256 is `592ef654a308edcba5fb3c2b302173650096a9487e4120b96e7adbe830429b9f`, at 2,354 bytes, with 84 LF, 0 CRLF, no trailing newline and no BOM. The git blob at `a9df2da` has the same hash. This matches `local_capture.json` and the work order. Result: **MATCH**.

### V4. Exhaustive recursive diff (my own walker)

My walker compared dicts key by key in both directions, lists by index, and scalars by type and value. I also checked both files for duplicate keys.

| Path | Kind |
|---|---|
| `/scripts` | REMOVED in installed |
| `/bugs` | REMOVED in installed |

- **Added:** none. **Changed values or types:** none, at any depth. **Duplicate keys:** none in either file.
- **Key order:** the npm top-level order minus `scripts` and `bugs` equals the installed order exactly (18 keys).
- **Serialization:** this is **CONFIRMED**. `json.dumps({npm minus bugs, scripts}, indent=2, ensure_ascii=False)` encoded as UTF-8 with no trailing newline equals the installed bytes exactly. Its SHA-256 is `592ef654…429b9f`.
- **npm file:** it equals `json.dumps(npm, indent=2, ensure_ascii=False) + "\n"`. It has no non-ASCII bytes, so `ensure_ascii` does not matter.

So the installed file is a deterministic re-serialization of the npm 1.9.0 manifest with two keys removed. It is not a different upstream manifest. This accounts for the whole raw-hash discrepancy, which the work order's claim boundary requires before any pin.

### V5. CRLF explanation for `sourceVerificationRawSha256`

- The committed `source_verification.json` raw SHA-256 is `0d94408e86b64667d67a37e233353633f171be1486e1b5b0890fca0b7c565ac9` (7,571 bytes, 0 CRLF).
- Converting every LF to CRLF gives 7,728 bytes and SHA-256 `754215058fec4f73dc10e4fd745a62d41f6e83641d7a8ed9091c9bd1691e4d44`. That exactly equals the value in `local_capture.json`.
- The file has one git revision (`19018e2`). `.gitattributes` forces LF only for `*.sh`, `*.go`, `Dockerfile` and `docker/**`, not `*.json`, so a Windows `core.autocrlf=true` checkout would produce CRLF.
- **Observed:** the CRLF rendering reproduces Local's hash byte for byte. **Inference, not observed:** the cause was Local's autocrlf checkout.

The explanation is **CONFIRMED**. There is no content conflict. Separately, the installed `package.json` has no CR bytes, which agrees with R142's statement that "LF normalization does not change the installed hash".

### V6. Impact analysis and verdict

I compared each field directly. All of the following are **equal** in both files:

- `bin`: `chrome-devtools-mcp` maps to `./build/src/bin/chrome-devtools-mcp.js`, and `chrome-devtools` maps to `./build/src/bin/chrome-devtools.js`.
- `main`: `./build/src/index.js`.
- `type`: `module`.
- `engines`: `node ^20.19.0 || ^22.12.0 || >=23`.
- `peerDependencies`: `@blackwell-systems/gcf ^2.2.2`, `@toon-format/toon ^4.1.0`. `peerDependenciesMeta` marks both optional.
- `files`, `overrides`, `allowScripts`, and `devDependencies` (32 entries).

These fields are **absent in both**: `exports`, `imports`, `dependencies`, `optionalDependencies`, `bundle(d)Dependencies`, `os`, `cpu`, `module`, `browser`, `workspaces`, `gypfile`, `binary`.

- **`scripts` (removed, 27 entries):** there is no `preinstall`, `install`, `postinstall`, `prepack` or `prepublish`. The only lifecycle hook is `prepare` (`node scripts/prepare.ts`), which npm runs on local or git installs, never when Node loads a module. `scripts/` is not in `files`, and the tarball does not ship it.
- **`bugs` (removed):** an informational URL only.
- **Invocation route:** R141 SPEC lines 89 and 91 and R141 review line 116 confirm the admitted route is `node <daemon.js>` by path. `bin`, the native CLI, `start`/`startDaemon` and `npm run` are forbidden, and `bin` is unchanged anyway.
- **Runtime read of `package.json` (a reviewer check beyond the worker's analysis):** a bounded in-memory text search of the tarball JS shows the bundled yargs in `build/src/third_party/index.js` (`kPkgUp`/`kGuessVersion`) reads the nearest `package.json` at runtime. It uses `version` (unchanged) and caller-named `pkgConf` keys. It does not read `bugs` or `scripts`. `build/src/version.js` hardcodes `VERSION = '1.9.0'`. The worker's "no field read by Node at module resolution differs" is correct but incomplete. This runtime read does not change the verdict.
- **Executable code:** `package.json` cannot change JS bytes. On Windows, only 19 JS files are pinned.

**Choice among the three allowed verdicts:**

- `BLOCKED_SOURCE_UNAVAILABLE` does not apply. The tarball was retained and its integrity matches.
- `PACKAGE_IDENTITY_UNRESOLVED` would be correct only if some field or value difference could change the admitted invocation. None does.
- So **`METADATA_ONLY_CANDIDATE` is correct**, scoped to `package.json`.

**UNKNOWNs:** stated honestly but not quantified. I quantify UNKNOWN 1 from the tarball, which has 348 regular files:

- `build/src` holds 331 files: 78 `.js`, 251 `.md`, 1 `.json` and 1 `THIRD_PARTY_NOTICES`. There are also 14 files under `skills/`, plus `LICENSE`, `README.md` and `package.json`.
- Windows hashes exist for 19 of the 78 JS files. **59 JS files, the non-JS `build/src` files and `skills/` are not pinned on Windows.** `daemon/daemon.js`, `index.js`, `browser.js`, `bin/*`, `third_party/index.js` and `tools/pages.js` are among the 19 pinned files. Their transitive imports are not all pinned.

UNKNOWNs 2 (which tool removed the keys), 3 (Antigravity update drift) and 4 (A1/A2 unrun) are stated correctly.

### V7. Scope, deviations, counters

- **Changed files:** `git diff --name-only 5c43dfd a9df2da` lists exactly two files: `docs/reviews/R144_A3_PACKAGE_METADATA_VERIFY_RETURN_2026-11-05.md` and `docs/reviews/probes/r144_a3_package_metadata_20261105/worker_compare.json`. Both are inside the seed's `allowedPaths` (`docs/reviews/R144_*.md`, `docs/reviews/probes/r144_a3_package_metadata_20261105/**`).
- **Files left untouched:** no source, test, gate, CI, continuity, state, handoff, tranche, authority, registry or catalog file changed. The work order still reads `Status: DISPATCH_READY`.
- **Deviations:** R144-D1 (one `git fetch origin main` of the project repository) and R144-D2 (branch commit and PR, no merge) are disclosed in a separate section with the work-order clauses they depart from. See finding M1 for the conflict with the authority seed.
- **No new package fetch:** the tarball mtime (2026-11-04 20:22Z) predates R144, and the worker's extraction left only `scratchpad/r144/package/package.json` (4,556 bytes) plus `compare.txt` and `recheck.txt`. That is consistent with no new package fetch and a single-file extraction.
- **Counters:** 1 comparison pass and 1 recheck are consistent with the scratch artifacts: `compare.txt` at 05:43:35Z and `recheck.txt` at 05:43:50Z. The time record is inconsistent; see L1.

### V8. Gate (run once)

The reviewer-fast gate was run once (command and full output in section 5). It exited with code 0: `RESULT: PASS phase=reviewer-fast mandatoryControls=8 failed=0`. `CVF-DG-ROUTE-01` was NOT_APPLICABLE_WITH_REASON, which means it was **not checked**, not that it passed.

## 3. Findings, ranked by severity

No HIGH findings. No finding is a defect against the work order's comparison contract.

**M1 (MEDIUM, procedural; Root ruling required).** R144-D1 and R144-D2 contradict the dispatcher-owned authority seed, not just the work-order text.

- The seed's `prohibitedEffects` say "No worker commit, push or merge", and `executionBudget.networkFetches` is 0.
- The worker made one project-repository fetch, committed `a9df2da`, and says it opened a PR.
- Both are disclosed and attributed to an operator grant. The fetch was of the project repository, not a package source, and does not affect the comparison evidence.
- I cannot see the operator messages. Root must confirm the grant and record a ruling, as it did for the R140 D2 exception, before treating the branch as admissible transport. The gate's RANGE and ROLE controls check paths and roles only. They cannot detect this.

**L1 (LOW, record accuracy).** The return's time record cannot be reconciled with git.

- The return says the authorized work ran "05:42:32Z – about 05:50Z, about 8 active minutes".
- Commit `a9df2da`, which contains that return, has author and committer timestamps of `2026-11-05T05:45:12Z`. The worker's scratch files run from `r144_start.txt` (05:42:32Z) to `recheck.txt` (05:43:50Z).
- The return cannot record an end time that comes after its own commit. Observed elapsed time to commit is about 2 min 40 s. The 8-minute figure is an estimate that overstates the time, conservatively, and stays inside the 20-minute ceiling.
- The 05:37:45Z "first blocked check" cannot be verified from repository evidence. Root should record the actual time as "≤ ~3 minutes to commit (git timestamp); worker estimate 8 minutes".

**L2 (LOW, completeness).** The impact analysis leaves out that bundled yargs reads `package.json` at runtime (V6). This does not change the result: it reads `version` and caller-named keys only, never `bugs` or `scripts`. Root's pin text should not claim "package.json is never read at runtime".

**I1 (INFO).** UNKNOWN 1 is honest but not quantified: 59 of 78 `build/src` JS files, plus the non-JS files and `skills/`, have no Windows hash (V6).

**I2 (INFO).** The 4 A3 JS hashes the worker relies on come from Local's `local_capture.json`. The cloud worker did not capture them. I confirmed they match the tarball.

**I3 (INFO).** The CRLF cause (autocrlf checkout) is an inference. The CRLF rendering reproducing `754215058f…` exactly is observed.

## 4. Recommended Root disposition for the A3 pin

1. **Rule on M1 (D1 and D2) first.** If the operator grant is not confirmed, keep this review and the worker evidence as retained content, and transport them under Root's own commit.
2. **Accept a qualified, exact `package.json` pin.** Suggested wording:

   > A3 `package.json`: installed raw SHA-256 `592ef654a308edcba5fb3c2b302173650096a9487e4120b96e7adbe830429b9f` (2,354 bytes) is byte-identical to `json.dumps(indent=2, ensure_ascii=False)`, with no trailing newline, of the `package/package.json` of npm `chrome-devtools-mcp@1.9.0`. That tarball has integrity `sha512-RnzXoJiUQ44hpOihWk90uOhLD/CnwDkDy0ldHMZONJ2nYQ+dWN1fq1luHHqyd+7FuYnyIlCY6uTThbN5ut9kSQ==` and sha1 `86bf2bde0bc4ec07aa428b36ea612eaa7a37a77c`, and its `package.json` hashes to `6a14bfc5…5021`. The installed file has the top-level keys `bugs` and `scripts` removed. No other key or value differs. `bin`, `main`, `type`, `engines`, `peerDependencies`, `files` and `exports`/`imports`/`dependencies` (absent) are unchanged. Executable identity is pinned only for the 19 Windows-hashed JS files: 15 from R141 and 4 from A3 (`local_capture.json`). It is valid only for the captured installation state.

   On this basis A3 becomes **5/5 for the five named files: A3 PASS (qualified)**. It is not a full-package identity.
3. **Full `build/src` Windows capture: recommended as required before any A1/A2 admission.** Admitted A1/A2 behavior depends on the whole import closure of `daemon/daemon.js` and `index.js`, and 59 of the 78 JS files are unpinned. Hashing all 348 files in the installed package directory and comparing them with the tarball manifest is a cheap, effect-free Windows step. It needs no install, execution or Chrome. It removes UNKNOWN 1 and is less error-prone than deriving an import closure by hand. Root could make it the first bounded step of the A1/A2 admission rather than a separate tranche. Whichever route Root takes:
   - the pin should be re-verified with a same-session hash recheck right before any A1/A2 execution, because of UNKNOWN 3 (Antigravity update drift);
   - any mismatch should return A3 to HOLD.
4. Correct the time record per L1 when synchronizing continuity.
5. R139 stays BUILD_BLOCKED, M3-05/M3-08 stay BLOCKED, and the R139 repair audit and registry are unchanged until Root authors a separate admission.

## 5. Gate result

Command, run once from the project root:

```text
PYTHONDONTWRITEBYTECODE=1 python3 -B scripts/cvf_gates/cvf_downstream_gate_runner.py run --phase reviewer-fast --project-root . --base 5c43dfdff2f6fd99634fd3485c098acea1bd2a43 --head HEAD --packet docs/work_orders/CCMAI_RUNTIME_144_A3_PACKAGE_METADATA_VERIFY.md --packet docs/reviews/R144_A3_PACKAGE_METADATA_VERIFY_RETURN_2026-11-05.md
```

```text
[PASS] CVF-DG-INST-01 PASS
[PASS] CVF-DG-CONT-01 PASS
[PASS] CVF-DG-RANGE-01 PASS  (RANGE base=5c43dfd… head=a9df2da… candidates=2 deleted=0)
[PASS] CVF-DG-APPL-01 PASS   (checked: work order)
[PASS] CVF-DG-ROUTE-01 NOT_APPLICABLE_WITH_REASON - no dispatch packet declares dispatchKind in scope  (NOT checked)
[PASS] CVF-DG-ROLE-01 PASS   (checked: worker return)
[PASS] CVF-DG-CLAIM-01 PASS  (0 explicit claim lines, 2 documents)
[PASS] CVF-DG-COV-01 PASS    (mechanical joins only; Local owns semantic acceptance)
RESULT: PASS phase=reviewer-fast mandatoryControls=8 failed=0
EXIT=0
```

The run was against HEAD `a9df2da`, before this review file existed. A pass proves only the checked static repository state. It does not prove that the operator grant exists, that the PR exists, hosted CI or runtime behavior.

## 6. Claim boundary

This review supports at most a qualified A3 `package.json` identity decision by Root. It does **not** support:

- any A1 or A2 result or admission;
- physical no-relaunch or relaunch prevention;
- reopening R139, changing its causal audit, or M3-05/M3-08 or other M3 acceptance;
- Chrome, MCP, daemon, browser, DB or provider behavior;
- live CVF governance of AI or agent behavior;
- hosted CI, deployment or product readiness.

The Facebook/Zalo OA parked checkpoint and the R139 HARD_HOLD are unchanged.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
