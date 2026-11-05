# R140 selection-read independent source review — 2026-11-05

Status: REVIEWER_ACCEPTED_SOURCE_PENDING_ROOT_CLOSURE
Tranche: CCMAI-RUNTIME-140
Risk: R2
Worker: Cloud IMPLEMENTATION_WORKER (Claude Code remote session; separate agent from this reviewer)
Reviewer: Independent Opus subagent appointed by operator (separate from implementation worker; same cloud session — independence limitation disclosed)
Work order: `docs/work_orders/CCMAI_RUNTIME_140_SELECTION_READ_REPAIR.md`
Worker return under review: `docs/reviews/R140_SELECTION_READ_WORKER_RETURN_2026-11-05.md`
Reviewed range: `e54358ce71205964be27db0f88b9d7f9293cbca4..fa5022e3013b792e9ecc39bd153819216b7976d4` (branch `claude/friendly-rubin-e2npmt`, commits `59788f0`, `fa5022e`)

This review covers bounded frontend source and component-test correctness for M3-06 only. It makes no AI-governance, provider, runtime, browser, backend, E04/T3, E06 or product acceptance claim. R139 stays BUILD_BLOCKED. M3-05 and M3-08 stay BLOCKED. This review does not close the tranche.

## 1. Appointment and independence limitations

- **Appointment.** The operator (repository owner) appointed this reviewer because Root/Codex is not available in this cloud session. The work order names "Root independent source reviewer" as reviewer. This review is therefore an operator-appointed substitute. Root must decide whether to adopt it or to re-review.
- **Author separation.** The reviewer did not write or edit the reviewed source, tests, return or worker probes. The only file this reviewer wrote is this review.
- **Independence limitation.** The reviewer runs on the same model family, in the same cloud container and session, as the implementation worker and the advisory second opinion. That is weaker than the provider-separated independence normally expected for R2. The source behavior oracle below was derived from the work-order contract and the source, not from the worker narrative, but the shared-session limitation stays.
- **Rehydration limitation.** The reviewer read the work order, the return, the operator cloud authorization, `root_release.json`, `source_preimages.json`, the probes and the current `ACTIVE_SESSION_STATE.json` fields. State at review: mode BUILD, active tranche CCMAI-RUNTIME-140, tranche record status BUILD, active handoff `CVF_SESSION/handoffs/AGENT_HANDOFF_R140_SELECTION_READ_REPAIR_2026-11-05.md`. The reviewer did not do a full First-Request rehydration: no sibling Core, no `WORKSPACE_RULES.md`, no operator-local pack (all absent in cloud, see D3).
- **Effects.** 0 commits, 0 pushes, 0 installs, 0 app/browser/DB starts, 0 provider calls, 0 secrets read. 1 static gate run. 0 vitest runs. 0 typecheck runs.

## 2. Scope check

`git diff --name-only e54358c HEAD` returns exactly 11 paths:

- `frontend/src/components/InterventionExecutionStatus.vue`
- `frontend/src/__tests__/intervention-execution-status.test.ts`
- `docs/reviews/R140_SELECTION_READ_WORKER_RETURN_2026-11-05.md`
- `docs/reviews/probes/r140_selection_read_20261105/worker_*` (8 files: `npm_ci.log`, `operator_cloud_authorization.json`, `test_green.{log,meta}`, `test_red_original.{log,meta}`, `typecheck.{log,meta}`)

All 11 paths are inside the four allowed path classes. No session, authority, continuity, catalog, gate, CI or registry path changed. The gate's `RANGE_RESOLVED` note independently counts the same 11 candidates.

Protected surfaces: `git diff --stat e54358c HEAD` is empty for:

- `frontend/src/components/interventionExecutionHold.ts`
- `frontend/src/api/intervention.ts`
- `frontend/package.json`
- `frontend/package-lock.json`
- root `package.json` and `package-lock.json`
- `backend/`

`git status --short` is clean. `frontend/node_modules` is git-ignored.

## 3. Final-byte hashes (LF-normalized SHA-256, recomputed by reviewer)

| File | Reviewer hash | Return | Meta | Match |
|---|---|---|---|---|
| `InterventionExecutionStatus.vue` (HEAD) | `b174b3b610ae0b64eb08aa15c81d7dbad023058edcd86e77bc9e1642c5b8932a` | same | green `vue_sha` same | yes |
| `intervention-execution-status.test.ts` (HEAD) | `4bb59b72f5214c202926139f0f021df54845e71f15520ac2f17da3b347cecd9f` | same | green and red `test_sha` same | yes |
| `InterventionExecutionStatus.vue` at `e54358c` (preimage) | `756cba7b19b347e72315ef1490255c9413e46c933844a6f9d69f0400df36c81a` | same | red `vue_sha` same | equals `source_preimages.json` |
| test at `e54358c` (preimage) | `85c12c8d1023c19cbf120232c52af46b255dfcc0321537ae7875bcd157fd61e4` | same | n/a | equals `source_preimages.json` |
| `interventionExecutionHold.ts` | `51b16dfef44d6fd0d293499c6f11068d00b17b18022df637a51c57372f8e3c7c` | same | n/a | equals protected pin |
| `api/intervention.ts` | `b7def52447f1312991836556cfe89594a05371fd1b32f8ffa66aa86e75c62bbc` | same | n/a | equals protected pin |
| `frontend/package.json` | `e5cec0dc55ecfd1855d17e109b738351ee638897bb7080e47a4b9dfd9e073894` | same | n/a | unchanged in range |
| `frontend/package-lock.json` | `97729b7331a38c99dd21450c0c85ce498875dc28e5b872200f3edf23bdd4a16d` | same | n/a | unchanged in range |

Logged results, as read by the reviewer:

- **Red (T1).** The log summary is `Tests 10 failed | 18 passed (28)`, and the meta records `exit=1`. That matches the claim.
  - The 10 failing test names are S01, S02, S03 ×4 (A→B reject, A→B→A resolve, A→B→A reject, timer), S04 ×2, S05 and S08. That matches return §4.
  - Every failure location in the red log (lines 178, 195, 214, 225, 248, 265, 282, 293 and 380) points to the matching assertion in the **final** test file.
  - This supports the meta claim that T1 used the final test bytes (`4bb59b72…`) against the original component.
- **Green (T2).** The log summary is `Tests 28 passed (28)`, and the meta records `exit=0`.
- **Typecheck.** The log is empty, with no diagnostics, and the meta records `exit=0`.
- **Timing.** The start/end times are consistent with the return.

## 4. Source review (independent oracle)

The reviewer read the full diff (`git diff e54358c HEAD -- frontend/src/components/InterventionExecutionStatus.vue`) and the whole file, and traced each contract item.

**Contract items 1–2: separate identity.** `readSeq`, `activeRead` and `ReadOwner` (lines 60–75) are separate from `generation` and `controller`. `beginRead` (line 63) increments the identity through `invalidateReads`. It neither increments `generation` nor aborts `controller`. Each `readSelected` call (line 131) captures an immutable `{seq, requestId, mine, key}`. That covers the initial load (line 179), selection (line 188), refresh (line 194) and observation (line 217). A newer read of the same id gets a new `seq`. `owns` (line 75) then fails for the older read because `read.seq!==readSeq`.

**Item 3: abort, detach and guard.**
- **Abort.** A superseded read's own controller is aborted (line 62).
- **Listener detach.** The listener on the parent signal is removed on supersede (line 62) and in `finally` (lines 154 and 74). The `attached` flag makes the removal idempotent.
- **Guard decides.** The decision comes from `owns(read)`, not from the promise, so an abort-ignoring resolve or reject cannot commit.
- **Parent linkage.** The parent signal is linked through a forwarding listener. Superseding a read never aborts `controller` or `poll`.

**Item 4: ownership before every read-origin mutation.**

| Read-origin effect | Guard | Line |
|---|---|---|
| status write | `owns` after await | 139 → 141 |
| OBSERVED hold write and `loadHold` | same branch, after `owns` | 142 |
| reject: status null and error | `if (owns(read))` | 143 |
| readiness follow-up request | `owns` recheck | 144 → 148 |
| readiness install and expiry timer | `owns` after await | 149 → 151 |
| readiness error | `if (owns(read))` | 152 |
| refresh `loading=false` | `current && stillNewest` | 195 |
| timer callback | `expiryTimer!==timer` self-guard | 86 |

The reviewer traced the following paths:

- **Stale success, error, finally and timer paths, including same-id A→B→A.** No unguarded path remains.
- **Initial selection or refresh overlap.** No user-reachable overlap remains:
  - Selection is now blocked while `loading` (line 184), matching the existing `:disabled="loading"` on the template buttons (line 10).
  - Refresh is blocked while `loading` or `sending` (line 191).
  - The only remaining overlap is a selection read (which does not set `loading`) superseded by a refresh. That is handled because the selection caller ignores the return value and the refresh read owns `loading`.
- **Stuck spinner.** The reviewer looked for a reachable case where `stillNewest===false` while `current` is true and found none. Such a case needs a newer read or `invalidateReads` during a refresh. A newer read is UI-blocked, and `load` and unmount both change `generation`.

**Item 5: lifetime and unmount.** `load()` (line 158) and `onBeforeUnmount` (line 260) call `invalidateReads()`. Lifetime changes still flow through `generation` and `lifetime` in `current()`. DTO validation, `safeError` localization and `deadlineFromServer` are byte-unchanged.

**Item 6: execution semantics preserved.** `executeOnce` (lines 230–257) is byte-unchanged. That covers:
- the generation and controller ownership
- `refreshCase` and the fresh readiness revalidation
- the pre-send `beginExecutionHold` IN_FLIGHT marker
- the single `executeInterventionRerun` POST
- the UNCERTAIN hold on failure, with no resend

`observeAfterSend` (lines 205–229) is byte-unchanged. That covers the 12 iterations, the 5 s `delayObservation` spacing, the 60 s `deadlineTimer` and `poll` linked to `controller.signal`.

The observation read now passes `poll.signal` as the parent. The deadline abort therefore reaches the in-flight read synchronously through the forwarding listener, which is equivalent to passing `poll.signal` directly. During `sending`, selection and refresh are blocked (lines 184 and 191), so a manual read can never supersede or cancel an observation read. The OBSERVED write after a real send still happens, because the observation read owns the identity; the existing two-field POST test asserts `OBSERVED`.

### Findings, ranked by severity (none blocking)

1. **Low (latent, not reachable through the UI): `InterventionExecutionStatus.vue:195` together with `:183-188`.**
   - **What happens.** If a non-UI caller ever started a selection read while a refresh is pending, the refresh would return `stillNewest=false` and leave `loading=true`. The selection read never clears `loading`, so the spinner would stick.
   - **Counterexample.** A programmatic `selectRequest` during a refresh.
   - **Why it is not reachable now.** The line-184 guard and the template's `:disabled="loading"` close this path. The finding needs no repair. It is recorded so that a future change that removes the line-184 guard re-examines it.
2. **Nit: `InterventionExecutionStatus.vue:133`.** `readSelected` returns `true` on early exits that mean "not current". This is safe only because every caller re-checks `current()` (line 195) or ignores the value. The same point was raised by the advisor; it is not a defect.
3. **Pre-existing, not introduced: `InterventionExecutionStatus.vue:89, 259`.** If `props.signal` aborts without the prop identity changing, `current()` turns false and an in-progress refresh leaves `loading=true`. The original source behaves the same way. This is outside the R140 contract.

No finding contradicts the contract. The reviewer found no stale success, error, finally, timer, hold-write or follow-up-request path that survives supersession.

## 5. Test review

- **Pure append.** The test diff removes 0 lines and appends one hunk after the existing `describe` (from line 142). The nine pre-existing tests (lines 40–141), including the `beforeEach`/`afterEach` setup, are byte-unchanged.
- **UI-driven, zero POST.** Every new S01–S08 case drives the component through rendered buttons. The `click` helper asserts the button exists and is not disabled before it triggers. Every case asserts `calls.execute` was not called. The obsolete promises are deferreds that ignore the abort signal.

Test findings, none blocking:

1. **Disclosed limitation: behavioral red not proven by execution (lines 225 and 293).**
   - **What failed in T1.** The two S03 A→B→A cases and S05 failed in T1 on the `signalOf(...).aborted===true` assertion. That assertion comes before their behavioral assertions.
   - **Trace on the original source.**
     - For S03 A→B→A resolve, the old readiness (call index 1) for `reqA` passes the original guard: `current`, `controller.signal` not aborted, and `requestId===selectedId` (A). The original then calls `installReadiness(READY)` and renders Start, so line 228 (`btn(start)` undefined) would fail.
     - For the reject variant, the original sets `error` to "Progress cannot be checked right now", so line 230 would fail.
     - For S05, the original commits `status` and then issues the follow-up readiness request, so line 298 (readiness count unchanged) would fail.
   - **Verdict.** The reviewer accepts the behavioral red by trace, not by execution. Same-id behavioral red is proven by execution in S02 (line 195) and S08 (line 380).
   - **Why it was not re-run.** Proving the trace by execution would need a modified test or source, which the reviewer may not create. This limitation does not block bounded source acceptance.
2. **Low: line 200.** `expect(holdOf(reqA)).toBeNull()` in S02 is vacuous on both the original and the fixed source. `updateExecutionHold` writes only when a prior marker exists (`interventionExecutionHold.ts`), and S02 seeds none. S08 is the effective hold-ownership proof.
3. **Low: S04 stale-finally clause.** The overlap reachable through the UI (selection read then refresh) has no stale `finally` that could clear `loading`, on either the original or the fixed source. The red in both S04 cases comes from the stale status and error overwrite. "Stale finally cannot clear new loading" is satisfied by construction (lines 184, 191 and 195), not by a red-then-green test. The work order requires the user-accessible path without bypassing UI disabling, and the tests meet that requirement.
4. **Low: S06 (lines 305–350).** The tests assert that the pending signal is aborted, that no late mutation or follow-up request happens, and that `vi.getTimerCount()===0`. They do not assert listener detachment directly. The reviewer verified it in source instead (`invalidateReads` and `endRead`). S06 passes on the original, so it is a preservation guard, as disclosed.
5. **Nit: line 363.** S07 checks only that the vi key resolves; the component is never rendered in vi. Localized rendering stays covered by the unchanged `safeError` and i18n usage.
6. **Nit: lines 241, 327 and 345.** Fake timers are installed after mount, so the first real 60 s timer is cleared through the faked `clearTimeout`. This is harmless because of the line-86 self-guard and unmount.
7. **Brittleness: lines 225, 293, 294, 313 and 333.** The `signalOf(..., i)` indexing depends on mock call order. That is acceptable for deterministic deferred tests.

**Rerun decision.** The reviewer did **not** run vitest or the typecheck. The logged evidence is internally consistent: hashes match, counts and exit codes match, and the failure lines map onto the final file. The review ceiling forbids routine duplicate runs.

## 6. S01–S10 verdicts

| ID | Reviewer verdict | Basis |
|---|---|---|
| S01 | PASS | Executed red on original (line 178, behavioral) and green. Guard at line 143. Zero readiness and zero execute asserted |
| S02 | PASS | Executed behavioral red (line 195) and green. `seq` guard proves same-id supersession. The hold assertion at line 200 is vacuous (test finding 2) |
| S03 | PASS (with limitation) | A→B reject and timer: executed behavioral red. A→B→A ×2: red on the abort assertion, behavioral red by trace (test finding 1). A→B resolve passes on original (regression guard). Timer self-guard and `owns` before `installReadiness` |
| S04 | PASS | Both cases: executed behavioral red and green through enabled UI buttons. Stale-finally clause satisfied by construction (test finding 3) |
| S05 | PASS (with limitation) | Abort-ignoring deferred. Red on the abort assertion, behavioral red by trace. Green asserts no follow-up status or readiness and an unchanged UI |
| S06 | PASS | Five prop changes plus two unmount cases. Preservation: passes on original and fixed source. Listener disposal verified in source (test finding 4) |
| S07 | PASS | Current 403 is localized and current success renders. Current readiness failure shows the unavailable message. No blanket suppression |
| S08 | PASS | Executed behavioral red (UNCERTAIN→OBSERVED rewrite, line 380) and green. Hold stays UNCERTAIN, B has no hold, no readiness, zero POST |
| S09 | PASS | All nine pre-existing tests unchanged and green (T1 and T2). `executeOnce` and `observeAfterSend` byte-unchanged. 60 s poll abort test green |
| S10 | PASS | Exactly two source/test files. Typecheck exit 0 (logged). No dependency, API, backend, i18n or execution-contract change |

## 7. Static gate (one run)

Command, run from the project root:

`PYTHONDONTWRITEBYTECODE=1 python3 -B scripts/cvf_gates/cvf_downstream_gate_runner.py run --phase reviewer-fast --project-root . --base e54358ce71205964be27db0f88b9d7f9293cbca4 --head HEAD --packet docs/work_orders/CCMAI_RUNTIME_140_SELECTION_READ_REPAIR.md --packet docs/reviews/R140_SELECTION_READ_WORKER_RETURN_2026-11-05.md`

The `--base` and `--head` arguments were accepted. Output:

```
[PASS] CVF-DG-INST-01 PASS
[PASS] CVF-DG-CONT-01 PASS
[PASS] CVF-DG-RANGE-01 PASS
       note: RANGE_RESOLVED mode=RANGE base=e54358ce71205964be27db0f88b9d7f9293cbca4 head=fa5022e3013b792e9ecc39bd153819216b7976d4 candidates=11 deleted=0
[PASS] CVF-DG-APPL-01 PASS
       note: checked: docs/work_orders/CCMAI_RUNTIME_140_SELECTION_READ_REPAIR.md
[PASS] CVF-DG-ROUTE-01 NOT_APPLICABLE_WITH_REASON - no dispatch packet declares dispatchKind in scope
[PASS] CVF-DG-ROLE-01 PASS
       note: checked: docs/reviews/R140_SELECTION_READ_WORKER_RETURN_2026-11-05.md
[PASS] CVF-DG-CLAIM-01 PASS
       note: 0 explicit claim line(s) and 2 document(s) scanned
[PASS] CVF-DG-COV-01 PASS
       note: checked: docs/work_orders/CCMAI_RUNTIME_140_SELECTION_READ_REPAIR.md
       note: mechanical joins only; Local owns semantic acceptance
RESULT: PASS phase=reviewer-fast mandatoryControls=8 failed=0
```

The exit code was 0.

- `CVF-DG-ROUTE-01` is `NOT_APPLICABLE_WITH_REASON`. That means it was **not checked**; it does not mean the route passed.
- The run used the in-repo installed profile 1.3 runner, not the operator-local trusted route (D3).
- This gate run happened before this review file existed. It therefore did not scan this file.
- A gate pass proves the checked static repository state only.

## 8. Operator-granted deviations D1–D4: review impact only

The reviewer neither grants nor rejects these deviations.

| ID | Impact on evidence correctness |
|---|---|
| D1 (`npm ci`) | **None on source bytes.** The lockfile and `package.json` are unchanged in the range, with hashes as in §3. The installed dependencies come from the committed lockfile with `--ignore-scripts` (vitest v4.1.0 per the log). `node_modules` is ignored and not committed. The evidence is valid for the lockfile-pinned toolchain. |
| D2 (transport commits on a branch) | **None on correctness.** The reviewed bytes are exactly those at `fa5022e`. Root's material commit must carry the same LF hashes (`b174b3b6…`, `4bb59b72…`), or this review does not apply. |
| D3 (no trusted bootstrap or WORKSPACE_RULES in the cloud) | **None on source correctness. Governance-route gap remains.** Root must still run the trusted operator-local route and the public doctor before the material commit. |
| D4 (Node 22 instead of CI Node 20) | **Low.** No Node-version-sensitive API is used in the diff. GitHub `frontend.yml` on Node 20 remains the independent re-check. |

## 9. What remains for Root

1. Decide whether to adopt this operator-appointed review, given the same-session and same-model independence limitation in §1, or to perform its own R2 independent review.
2. Accept or reject D1–D4. Rejection means discarding the branch commits.
3. Make the material commit to `main` as COMMIT_STEWARD, with byte-identical source (hashes in §3). Then run the committed-range `pre-commit`, `pr-ci` and `committed_range_closure` gates.
4. Run the trusted operator-local bootstrap (version-specific doctor route) and the public doctor locally. Keep the documented LOCK_INVALID and COV_OBJECT_SHAPE results.
5. Run the Windows catalog gate; the cloud preflight skipped `catalog`.
6. Update continuity: tranche record, state, memory, handoff, `IMPLEMENTATION_STATUS`, roadmap and catalog.
7. Keep R139 BUILD_BLOCKED and M3-05 and M3-08 BLOCKED. M3-06 full browser lifetime remains unproved (R139 G2).
8. Make no product, runtime, browser, provider or AI-governance acceptance on the basis of this review.
