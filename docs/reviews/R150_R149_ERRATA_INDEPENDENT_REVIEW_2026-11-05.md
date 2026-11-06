# R150 independent review: errata binding inputs for frozen R149

Status: REVIEWER_ACCEPTED_PENDING_LOCAL
Tranche: CCMAI-RUNTIME-150
Risk: R2
Worker: Cloud ORCHESTRATOR acting as SPEC_AUTHOR (separate agent from this reviewer)
Reviewer: Independent Opus review subagent appointed under operator grant (same cloud session; independence limitation disclosed)

Subject: build `821940c4b20cecbb6ff7184e5befbd5a80e3d9d5` on `claude/friendly-rubin-e2npmt-r150`. Seed `4164a28d174da1625f4a4b2fa03414c95962cf25`. Base `main` `6ded46cedd9cd9a9cf166e46c46ec0c407d10768`. PR #18 head `ec86f91051cae45644f3e9e1d22ac0ee8ed2ff04`.

## 1. Independence limitation

I did not write the errata, the packet or the evidence copies. I am a subagent in the same cloud session as the errata author, and that orchestrator appointed me. I am therefore separate from the author, but not organizationally independent of it. This review was static: I read files and `git show` blobs, and I read the native package sources in place under the scratchpad `cdm_src/package`. I did not import or execute them. I did not start Chrome, the daemon, a browser, a DB or a provider, and I made no network call, commit or push. The only file I wrote is this one. Root/Local must still review independently before any merge.

## 2. Line-reference verification (against `6ded46c`)

| Erratum | Cited location | Content found at `6ded46c` | Result |
|---|---|---|---|
| E-B3 | draft `:29` | "Record Chrome version (`chrome.exe --version` run from the shell …)" | Correct |
| E-B3 | `R139_SDK_P01_B01_DESIGN_SPEC_2026-10-30.md:63` | "No personal/default Chrome, unsupported browser/debugging-port launch, shared daemon or native CLI startup wrapper"; also the TEMP/TMP rule | Correct |
| E-B1a | contract `:19` | "Where two sources conflict, the later item wins." Items 1–6 at `:21-26` put the contract last, so it wins | Correct |
| E-B1b/c/d | contract §5 `:141-145` | The five lifecycle rows (ADMITTED … CLEANUP → TERMINAL) | Correct |
| E-B2 | contract §2.2 `:68-73` | The F6 N4 list (3 conflicts) and the forbidden-flag sentence | Correct |
| E-NF1 | draft `:41` | "Raw WebSocket handshakes: at most 6." | Correct |
| E-M1 | draft `:52-58` | A1-b `:52-53`, A1-c `:54-56`, A1-d `:57-58` | Correct. The range also takes in A1-d |
| E-M2 | draft A1-c | `:54-56` | Correct |
| E-M3 | draft `:75`, contract §5 CLEANUP | "Stop the daemon through its handle and wait on it." / `:145` | Correct |
| E-M4 | contract §2.3 `:98` | The "Allowed: the OS minimum …" line | Correct as an anchor |
| E-M5, E-NF4 | contract §0/§3 | `:17-26`, `:114-126` | Correct |
| E-NF2 | draft `:45` | "may use only Node's built-in `WebSocket` … or PowerShell" | Correct |
| E-A2 | draft `:62-66` | A2-a `:62-63`, A2-c `:65-67` | Correct anchor, but it stops one line short of A2-c's second expectation (`:67`) |
| E-L | contract `:101` | "`daemon/utils.js:29-33`" | Correct |
| E-L | contract `:48` (`--headless=new`) | Present | Correct |

Native citations, checked in the 1.9.0 package (`package.json` version `1.9.0`):

| Citation | Content | Result |
|---|---|---|
| `config/browser-options.js:10` | `autoConnect.conflicts: ['isolated','executablePath']` | Correct |
| `:89` | `executablePath.conflicts: ['browserUrl','wsEndpoint']` | Correct |
| `:99` | `userDataDir.conflicts: ['browserUrl','wsEndpoint','isolated']` | Correct |
| `:105` | `channel.conflicts: ['browserUrl','wsEndpoint','executablePath']` | Correct |
| `config/category-options.js:37` | PWA `conflicts: ['autoConnect','browserUrl','wsEndpoint']`, `offByDefault: true` | Correct |
| `config/mcp-options.js:90,95` | `experimentalFfmpegPath`/`experimentalScreencastFps` `implies: 'experimentalScreencast'` | Correct |
| `daemon/client.js:145-146` | `sendCommand({method:'stop'})`, then `waitForFile(pidFilePath, true)` | Correct |
| `daemon/utils.js:30-35` | `if (IS_WINDOWS) { … named pipe … }` block | Correct. The main citation `:29-33` is off; line 29 is `appName` |
| `daemon/utils.js:60-61` | Windows fallback `path.join(os.tmpdir(), appName)` | Correct |
| E-M4 claim "`status.args` cannot show Node flags" | `daemon/daemon.js` `mcpServerArgs = process.argv.slice(2)`, returned as `args` | Correct. `execArgv` is excluded |
| E-M4 `LOCALAPPDATA`/`XDG_DATA_HOME` | `telemetry/persistence.js:22,25` | Correct |

**All file:line references in the errata tables are correct at `6ded46c`.** The only imprecision is in range ends (E-A2, E-M1), and both still point at the correct steps.

## 3. Defect existence on `main`

| Erratum | Claimed defect | Exists at `6ded46c`? |
|---|---|---|
| E-B3 | Draft directs `chrome.exe --version` | **Yes**, `:29`. On Windows, `chrome.exe` does not print a version to the console; it opens a browser session with the default profile, which R139 SDK `:63` forbids. The proposed `(Get-Item …).VersionInfo.ProductVersion` reads file metadata and executes nothing |
| E-B1a | "Later item wins" ranks the contract above the Root lifecycle and R141 | **Yes**, `:19`, `:21-26`. Local's independent review `:38` limits it so it "cannot authorize a waiver of the R139 hard hold or the seven Local conditions", but does not make the Root rows controlling. The bytes are unchanged |
| E-B1b | §5 lacks Root verification and G1/E01–E03 before G2/E04–E06 | **Yes**. Row `:143` has no Root verification and no G1/G2 ordering. The source row is `R139_CAUSAL_CHAIN_ASSESSMENT:40` |
| E-B1c | §5 lacks TERMINAL → INDEPENDENT_ACCEPTANCE | **Yes**. There is no such row. Source: R139 causal `:43` |
| E-B1d | §5 lacks R-F2's CLI/`start`/`startDaemon` ban and its TERMINAL check | **Yes**. The ban is in R141 spec R-F2 (`:91`) and the TERMINAL observation at `:95-97`. The contract has only the ADMITTED preflight (`:141`). `:117` describes `sendCommand` behavior, which is not a ban |
| E-B2 | N4 list incomplete | **Yes**, `:68-71` lists 3 conflicts. Note: `:73` forbids every one of those flags by name, and "rejects any other flag" covers `--categoryPwa`. The gap is in citation completeness (R142 condition 3), not in safety, as the cloud B2 itself says |
| E-NF1 | Handshake budget 6 | **Partly.** Under the `main` procedure alone, the A1 WebSocket opens number 4, or 5 with A1-d. That fits 6. But `:49` charges the `/json/version` queries to unspecified "existing … call budgets", and Local's own review (`:36`) asks for them to be counted explicitly. The shortfall to 8–10 arises only once E-M1 (raw HTTP upgrade per probe) is applied. See F3 |
| E-M1…E-M5, E-NF2, E-NF3, E-NF4, E-A2 | Missing evidence/HOLD/cleanup/wording items | **Yes**, all absent from the `main` bytes |
| E-L citation | `utils.js:29-33` | **Yes** |
| E-L coverage wording | "A3 identity accepted" should read "accepted subject to same-session recapture" | **No.** The `main` text (`:161`, rows M3-05/M3-08) already reads "A3 identity accepted under the operator-relaxed condition … subject to same-session recapture", and has since `d8e66c1`. The erratum is a no-op and misstates cloud L1. See F1 |

## 4. Evidence-copy verification (E4)

| Copy | Compared with | Result |
|---|---|---|
| `cloud_subagent_r149_review_ec86f91.md` | `ec86f91:docs/reviews/R149_OHC_SUCCESSOR_INDEPENDENT_REVIEW_2026-11-05.md` | `cmp` byte-identical. SHA-256 `f781396053e898e0c8fdb7e060d29003aff4980a396f2c60425a326249a0bb9a` matches the disclosed `f7813960…0bb9a`. LF only |
| `cloud_repaired_r149_draft_ec86f91.md` | `ec86f91:docs/reviews/R149_A1_A2_WINDOWS_VERIFY_WORK_ORDER_DRAFT_2026-11-05.md` | `cmp` byte-identical (SHA-256 `c61d0bd0…e688`) |
| `cloud_repaired_r149_contract_ec86f91.md` | `ec86f91:docs/reviews/R149_OHC_SUCCESSOR_CONTRACT_2026-11-05.md` | `diff` shows exactly two changes: one added HTML header comment (line 1), and the fence tag `cvf-milestone-coverage-json` renamed to `json-historical-coverage-copy-not-live`. Both are disclosed in errata `:26` and in the comment |

The copies keep the cloud review under a new name. That avoids the path collision with Local's same-named review on `main`.

## 5. E1–E6 verdicts

| ID | Verdict | Basis |
|---|---|---|
| E1 | **PASS** | All seven blocking errata cite a `main` location and give a concrete correction. E-NF1 defers the exact number to Local but states the binding rule: cover every A1 step and count discovery explicitly (F3) |
| E2 | **PASS** | §2: every reference is correct. Two range ends are imprecise but still point at the correct steps |
| E3 | **PASS** | Local's A1 `/json/version` discovery and its source-boundary citations (F5, N-R1, N-R3, A1/A2) are stated as kept (`:20`). No erratum reverts them; E-M1/E-M2 add evidence alongside the discovery rule. Local's `main` bytes fix no item the errata calls missing. Local's review notes overlap four items without fixing them (F4) |
| E4 | **PASS** | §4 |
| E5 | **PASS** | `git diff --name-only 6ded46c 821940c`: 15 files, all inside the seed's `allowedPaths`. No frozen R149 contract/draft/review bytes, R149 tranche record, repair audit, `docs/milestone-coverage-authority.json`, product source, gate or CI changed. The build did not touch the seed (`git diff 4164a28 821940c` excludes `CVF_SESSION/authority/`). The R149 handoff gains only a supersession banner and `Status: SUPERSEDED` (continuity, `CVF_SESSION/**`). `ARTIFACT_REGISTRY.json` changes only the active-handoff pointer. Errata `§4` makes no admission, execution or acceptance claim |
| E6 | **PASS** (this review) | Findings below; none is a defect against E1–E6 |

## 6. Findings, ranked

None of these blocks acceptance against E1–E6. F1 is a real accuracy defect in a non-blocking erratum. It should be corrected in the tranche's one consolidated repair round if the orchestrator uses it; otherwise Local must work from cloud L1 in the evidence copy, not from the E-L text.

**F1: MEDIUM-LOW (accuracy of a non-blocking erratum).** E-L says the coverage wording "should read 'accepted subject to same-session recapture'". `main` `:161` already says exactly that (as did `d8e66c1`), so applying E-L changes nothing. The actual cloud finding L1 (evidence copy `:188-190`) is different:
- the word "accepted" implies a Local acceptance that has not been recorded; L1 suggests "R148 capture satisfies the operator-relaxed condition; Local acceptance and same-session recapture pending";
- the destination "R144 finite A3 metadata VERIFY returns to Root" is stale, because R144 is closed.

The errata drops both points. Correction: replace E-L's coverage item with L1's two points.

**F2: LOW (internal consistency, fails closed).** E-B1a says the stricter requirement applies, and that "the operator's A3 relaxation is the only deliberate relaxation". E-NF4 names only N-R2 as a sanctioned replacement. Two R141-review amendments are not named:
- N-R3 (non-descendant Chrome only) relaxes the literal text of R-F2 (R141 spec `:95-97`, "no Chrome process … other than the owner's PID");
- N-R1 replaces R-F1's pinned set with a full inventory.

E-B1d itself relies on N-R3 ("no non-descendant Chrome"). Read literally, E-B1a would restore R-F2's stricter text and HOLD on Chrome's own child processes. That is not unsafe, but it would waste launches. Cloud B1(a) originally named "N-R1..N-R3 supersession of R-F1 wording" alongside A3; the cloud repair narrowed it, and the errata inherited the narrowing. Recommendation: Local names N-R1–N-R3 (R141 independent review round 2) as accepted amendments next to A3.

**F3: LOW (justification of a blocking item).** E-NF1 is marked blocking, but most of its shortfall depends on E-M1, which is non-blocking. Under the `main` procedure alone, at most 5 WebSocket opens are needed. The defect that stands on `main` regardless is that `:49` charges discovery queries to unnamed budgets. Recommendation: state both conditions. If E-M1 is applied, the budget must be at least 10 handshakes/upgrades (or separate counters). In all cases, the `/json/version` queries (at most 2) get their own explicit counter.

**F4: LOW (narrative completeness and fairness to Local).** Errata §1 is accurate: the bytes from both review paths, Root's first-pass C1/C4, the separate PASS, and that none of B1–B3 is on `main`. Two gaps:
- It does not credit Local's independent review `:36,38`, which already raised admission notes overlapping E-A2 (raw SDK validation-error envelope), E-NF1 (count `/json/version` explicitly), E-M3 (verify the MCP child) and E-B1a ("later wins" cannot waive the R139 hold or the seven conditions; no present conflict found). These are notes, not byte fixes, so the "missing on `main`" claims stand. Crediting them would help Local de-duplicate.
- The statement that "the cloud review artifact was not on the remote" comes from Root's first-pass review (`R149_ROOT_INDEPENDENT_REVIEW:9`), not from the `/root/r148_independent_review` PASS.

"It was pushed later" cannot be verified offline. Commit times are consistent with it: `d8e66c1` 06:32 UTC, and the round-1 review was first committed in `e448d5a` at 06:45 UTC.

**F5: LOW (misuse risk).** The cloud-repaired contract and draft copies predate Local's corrections. The draft copy has no `/json/version` rule, and the contract copy has no "Source boundary" citations. The errata makes only its items binding (`:22`), which mitigates this, but it never warns that adopting a copy wholesale would revert Local's corrections. Recommendation: add that warning in Local's successor.

**F6: LOW (precision).**
- E-M3's "native `stop` command" could be read as the CLI `stop` (`bin/chrome-devtools.js:128,133`), which E-B1d bans. Better wording: "`sendCommand({method:'stop'})` as in `stopDaemon` (`daemon/client.js:139-147`), never the CLI".
- E-B3 omits cloud B3's "record the executable path and SHA-256".
- E-B2 omits cloud B2's "mark `allowed/blockedUrlPattern` as unrelated to N4".
- E-A2 range ends at `:66`, not `:67`.

## 7. Gate result (run once)

`python3 -B scripts/cvf_gates/cvf_downstream_gate_runner.py run --phase reviewer-fast --project-root . --base 6ded46cedd9c… --head HEAD --packet <R150 packet> --packet <R150 binding inputs>`, run before this file existed:

- PASS: INST-01, CONT-01, RANGE-01 (base `6ded46ce…`, head `821940c4…`, candidates=15, deleted=0), ROLE-01 (5 documents), CLAIM-01 (0 claim lines).
- APPL-01, ROUTE-01, COV-01: `NOT_APPLICABLE_WITH_REASON`. These controls were **not checked**; they did not pass a contract.
- `RESULT: PASS phase=reviewer-fast mandatoryControls=8 failed=0`, exit 0.

This proves checked static repository state only.

## 8. What remains for Local

1. Root/Local independent review of this tranche and of this review. The independence limit is in §1.
2. Decide F1–F6: fix F1 in the single repair round, or carry F1–F6 into the successor.
3. Any successor A1/A2 work order and executable OHC contract must start from the `main` bytes and apply E-B1a–E-B3 and E-NF1, with F2/F3 resolved. Apply or reject the non-blocking items with recorded reasons, using cloud L1/L2/L3 and M2/M4 residuals from the evidence copy. Then follow the existing route: repair-audit successor naming the A1/A2 discriminator, independent admission review, pre-dispatch gate, and a separate Windows effect admission.
4. Merge, and closing PR #18 as superseded, remain Root/Local actions.

This review makes no runtime, browser, physical no-relaunch, AI-governance, R139, M3-05/M3-08 or product claim. A1/A2 remain unrun and unadmitted. R139 remains BUILD_BLOCKED.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
