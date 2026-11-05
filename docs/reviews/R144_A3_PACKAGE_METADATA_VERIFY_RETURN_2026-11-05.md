# R144 A3 package metadata VERIFY: worker return

Status: COMPLETE_PENDING_REVIEW
Tranche: CCMAI-RUNTIME-144
Risk: R2
Worker: Claude Code cloud VERIFY worker (documentary comparison only)
Reviewer: Root/Local independent REVIEWER (pending)
Work order: `docs/work_orders/CCMAI_RUNTIME_144_A3_PACKAGE_METADATA_VERIFY.md` (LF SHA-256 `0f60ed08a21e77a5b9bdc3b11db57b420f640d5e2ced4b4668ff608a11887a2e`, verified)

**Verdict: METADATA_ONLY_CANDIDATE**

The installed Windows `package.json` is byte-for-byte the R141 npm `package.json` with two top-level keys removed (`bugs`, `scripts`), re-serialized with 2-space indentation and no trailing newline. No field read by Node at module resolution, and no CLI or executable field, differs. Per the work order, this is a candidate, not A3 PASS: Root decides whether to set an exact installed-package pin.

## 1. Startup and role (before work)

- **CVF Agent Declaration:** project Customer-Care-Monitor-AI; public Core `c4c8767` (manifest pin, not cloned in the cloud); phase WORK_ORDER, tranche CCMAI-RUNTIME-144 DISPATCH_READY; risk ceiling R2; live evidence required for any AI-governance claim (none made); active role IMPLEMENTATION_WORKER (VERIFY); active handoff `CVF_SESSION/handoffs/AGENT_HANDOFF_R144_A3_PACKAGE_METADATA_VERIFY_2026-11-05.md`; parked checkpoint OWNER_DEFERRED_FACEBOOK_ZALO_OA_ACCOUNTS.
- **Checkout:** new work branch `claude/friendly-rubin-e2npmt-r144` at exactly `5c43dfdff2f6fd99634fd3485c098acea1bd2a43`. No other work was reset; the earlier R142 branch is untouched.
- **Admission boundary checked:**
  - registry audit pointer is `repair_chain_assessment_a3_verify.json`, disposition `DIAGNOSTIC_ONLY`, ceiling 20 minutes, `admittedAction` packet SHA equal to this work order;
  - pre-dispatch gate (runner 1.3) PASS, 6 controls, 0 failed. ROUTE-01 was NOT_APPLICABLE, meaning not checked.

## 2. Operator-granted deviations (read first)

| ID | Deviation | Work-order clause | Bound used |
|---|---|---|---|
| R144-D1 | One `git fetch origin main` of the **project repository** to obtain `5c43dfd`, which was not in the cloud workspace (first reply was BLOCKED_WITH_REASON) | Operator prompt: "không tự git fetch/pull"; seed: "No new network fetch" (written for package sources) | 1 fetch of `main` only. No package or npm fetch. |
| R144-D2 | Worker commits this return and its probe to a branch and opens a PR; no merge | "It does not commit, push or merge"; "D2 exception applied only to R140" | The operator said "quy tắc cũ, bạn có quyền làm, nhưng không được merge" (old rules apply, you may do it, but no merge). Branch-only commit, PR to `main`, Root/Local merges or rejects. |

No continuity, state, handoff, tranche record, authority, registry, catalog or source file was edited. Root owns those.

## 3. Source identity checks

| Source | Expected | Observed | Result |
|---|---|---|---|
| R141 tarball `cdm-1.9.0.tgz` (retained in the cloud scratchpad since 2026-11-04 20:22Z; 2,458,275 bytes) | sha512 `RnzXoJiU…ut9kSQ==`, shasum `86bf2bde…a77c` (from `source_verification.json`) | identical | MATCH; not re-downloaded |
| npm `package/package.json` from that tarball | `6a14bfc5…5021` | `6a14bfc5…5021`, 4,556 bytes | MATCH |
| Installed Windows `installed_package.json` (committed) | `592ef654…429b9f`, 2,354 bytes | identical | MATCH |
| `source_verification.json` (committed) | `local_capture.json` records `75421505…4d44` | committed raw = `0d94408e…5ac9`; the CRLF rendering of the same content = `75421505…4d44` | Same content. The capture hashed a Windows CRLF checkout; this is not a source conflict. |

## 4. Comparison (one pass)

**Raw bytes and serialization**

| | npm | installed |
|---|---|---|
| Bytes | 4,556 | 2,354 |
| LF lines | 117 | 84 |
| CRLF / BOM | 0 / no | 0 / no |
| Trailing newline | yes | **no** |
| Indent | 2 spaces | 2 spaces |
| Key order | — | npm order preserved, minus 2 keys |

**Parsed JSON (exhaustive recursive diff)**

| Path | Change | npm value |
|---|---|---|
| `/bugs` | REMOVED in installed | `{"url":"https://github.com/ChromeDevTools/chrome-devtools-mcp/issues"}` |
| `/scripts` | REMOVED in installed | 27 npm-run scripts (`cli:generate`, `clean`, `bundle`, `build`, `typecheck`, `format`, `check-format`, `gen`, `docs:generate`, `start`, `start-debug`, `test`, `test:no-build`, `test:notices`, `test:notices:no-build`, `test:only`, `test:update-snapshots`, `prepare`, `sync`, `verify-server-json-version`, `update-lighthouse`, `update-metrics`, `verify-npm-package`, `eval`, `count-tokens`, `profile`, `test:memory`) |

There are no added keys and no changed values anywhere, at any depth.

## 5. Recheck (one, disputed field: the serialization)

Question: can the raw hash difference be fully explained by the two removals?

Result: **yes, exactly.**
- `json.dumps(npm minus {bugs, scripts}, indent=2)` with no trailing newline reproduces `592ef654a308edcba5fb3c2b302173650096a9487e4120b96e7adbe830429b9f` byte-for-byte.
- The npm file itself equals `json.dumps(indent=2)` plus a newline.

This means the installed file is a deterministic re-serialization, not a different upstream version.

## 6. Impact analysis

| Area | Observation | Impact |
|---|---|---|
| `bin` | equal: `chrome-devtools-mcp` → `./build/src/bin/chrome-devtools-mcp.js`, `chrome-devtools` → `./build/src/bin/chrome-devtools.js` | None |
| CLI entrypoint | R141 and R139 invoke `node <daemon/daemon.js>` directly by path, not via `bin` or `npm run`; `bin` is unchanged anyway | None |
| `main` | equal: `./build/src/index.js` | None |
| `type` | equal: `module` | None (ESM resolution unchanged) |
| `exports` / `imports` | absent in both | None |
| `engines` | equal: `node ^20.19.0 \|\| ^22.12.0 \|\| >=23` | None |
| `dependencies` | absent in both (bundled package) | None |
| `peerDependencies` / `peerDependenciesMeta` | equal (`@blackwell-systems/gcf`, `@toon-format/toon`, both optional) | None |
| `files`, `overrides`, `allowScripts`, `devDependencies` (32) | equal | None |
| `scripts` (removed) | Used only by `npm run` and npm lifecycle. The only lifecycle entry was `prepare`, which npm runs for local or git installs, not for runtime module loading | No effect on `node <path>` execution. The R141 contract already forbids `npm run`/CLI routes. |
| `bugs` (removed) | Informational URL | None |
| Executable code | `package.json` cannot change JS bytes. Only the Windows-hashed JS files are pinned: 15 R141 files plus 4 A3 files | See UNKNOWN 1 |

## 7. Observed vs UNKNOWN

**Observed:** the installed `package.json` is exactly the npm 1.9.0 `package.json` minus `bugs` and `scripts`, re-serialized. All runtime-, CLI- and resolution-relevant fields are identical.

**UNKNOWN:**
1. Identity of the other files in the installed `build/src` tree. Only 19 JS files were hashed on Windows (15 in R141, 4 in A3); `daemon/daemon.js` is among the 15. A full-tree pin would need a separate Windows hash capture.
2. Which tool removed the keys (presumably Antigravity's packaging step). This is not determinable from these two files.
3. Whether the Windows install can change on an Antigravity update. The pin applies only to the captured state.
4. A1 and A2 remain unrun. Nothing here proves physical no-relaunch, R139/M3 acceptance or CVF runtime governance.

## 8. Counters and time

| Item | Value |
|---|---|
| First (blocked) check | 2026-11-05T05:37:45Z, about 1 minute |
| Authorized work | 05:42:32Z – about 05:50Z, about 8 active minutes of the 20 allowed |
| Comparison passes | 1 |
| Recheck passes | 1 |
| Project-repo git fetch | 1 (R144-D1) |
| Package or network fetch | 0 |
| Install / import / execute / Chrome / MCP / provider / DB | 0 |

Tarball handling: only `package/package.json` was extracted, with `tar`, into an isolated scratch directory and read with `python3 -I`. No package bytes are committed.

Evidence: `docs/reviews/probes/r144_a3_package_metadata_20261105/worker_compare.json`.

Static gates:
- Runner 1.3 `worker-return` with the work order and this return: PASS, 7 controls, 0 failed.
- Preflight `--skip-catalog`: PASS 6/6. The catalog gate is Windows CI only.

These prove repository state only.

## 9. For Root

- Decide whether to accept a **qualified exact installed-package pin**. A suggested form: the installed `package.json` raw SHA `592ef654…`, defined as npm 1.9.0 tarball integrity `sha512-RnzX…` minus `bugs`/`scripts`, plus the 19 Windows-hashed JS files.
- Decide whether UNKNOWN 1 needs a full-tree Windows capture before an A1/A2 admission.
- Rule on R144-D1 and R144-D2.
- R139 stays BUILD_BLOCKED; M3-05/M3-08 stay BLOCKED.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
