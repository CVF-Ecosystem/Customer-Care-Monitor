# R140 selection-read worker return — 2026-11-05

Status: COMPLETE_PENDING_REVIEW
Tranche: `CCMAI-RUNTIME-140`
Risk: R2
Worker: Cloud IMPLEMENTATION_WORKER (Claude Code remote session, branch `claude/friendly-rubin-e2npmt`)
Reviewer: Root independent source REVIEWER (pending); advisory-only second opinion recorded below, not a review
Work order: `docs/work_orders/CCMAI_RUNTIME_140_SELECTION_READ_REPAIR.md`

The worker does not self-approve. Root owns review, acceptance, the material commit, continuity, and catalog. Only bounded frontend source/test correctness for M3-06 is claimed. No AI-governance, runtime, browser, product, E04/T3 or E06 claim is made. R139 remains BUILD_BLOCKED, M3-05/M3-08 remain BLOCKED, and P01 is not reopened.

## 0. Operator-granted cloud deviations (READ FIRST)

This return was produced in an ephemeral cloud container, not in the operator-local Windows workspace the release assumed. The first worker report was `BLOCKED_WITH_REASON` because of the following conditions, with zero edits and zero counters:

- `frontend/node_modules` was absent.
- `../WORKSPACE_RULES.md` was absent.
- `../.Controlled-Vibe-Framework-CVF` was absent.
- `../CVF_RULE_PACKS/operator-local` was absent.

The operator then evaluated the GitHub repository with the worker and **granted the deviations below as an operator decision**. The operator required that every deviation be written down clearly for Local/Root review on GitHub before any merge. The full record is `docs/reviews/probes/r140_selection_read_20261105/worker_operator_cloud_authorization.json`.

| ID | Deviation (operator-granted) | Work-order clause affected | Bound actually used |
|---|---|---|---|
| D1 | One `npm ci --ignore-scripts --no-audit --no-fund` in `frontend/` | "must not ... install or download dependencies" | 1 install, exit 0, 4.9 s, 209 packages from the committed lockfile. `package.json`/`package-lock.json` unchanged (LF SHA `e5cec0dc…` / `97729b73…`). Log: `worker_npm_ci.log` |
| D2 | Transport-only commit of the 4 allowed path classes, pushed to `claude/friendly-rubin-e2npmt` | "Worker must not commit, push" | No PR, no merge, no push to `main`. Not a material commit. Root re-commits or merges as COMMIT_STEWARD |
| D3 | `WORKSPACE_RULES.md` and the operator-local trusted bootstrap are `UNAVAILABLE_IN_CLOUD`. The in-repo installed profile 1.3 runner is used after lock-hash verification | AGENTS.md workspace-root stop rule; startup doctor route | No full-doctor or trusted-bootstrap PASS claim. Root must run the trusted operator-local route before the material commit |
| D4 | Node v22.23.4 / npm 10.9.9 instead of the CI's Node 20 | none (disclosure) | GitHub `frontend.yml` (Node 20) is the independent re-check if Root opens a PR |

If Root rejects any of D1–D4, it should discard the branch commit. Nothing on `main` changed.

## 1. Pre-BUILD acknowledgment (recorded before any edit)

- Role: IMPLEMENTATION_WORKER only. The worker does no orchestration, makes no authority, continuity or catalog change, and gives no self-approval.
- Checkout HEAD: `e54358ce71205964be27db0f88b9d7f9293cbca4`, equal to `origin/main`. The worktree was clean, with no parallel work to preserve.
- Release `root_release.json` was committed in `e54358c`, with status `RELEASED_SOURCE_ONLY`.
- Preparation base `723976c9…` is an ancestor of HEAD. Seed commit `4c3cc11e…` is an ancestor of HEAD.
- LF-normalized SHA-256 matched for 10 of 10 files:
  - work order `44fcad50…`
  - seed `82984c72…`
  - source_preimages `da51cde1…`
  - independent_admission `24e744a9…`
  - root_preflight_corrected `851c7cd9…`
  - root_profile_precommit_final `cccfab11…`
  - component preimage `756cba7b…`
  - test preimage `85c12c8d…`
  - protected `interventionExecutionHold.ts` `51b16dfe…`
  - protected `api/intervention.ts` `b7def524…`
- Public Core remote HEAD is `c4c8767a…`, which matches the manifest pin. It was not cloned.
- Every installed profile file matches `.cvf/gate-profile.lock.json`. The in-repo runner `pre-dispatch --packet <work order>` returned PASS, 6 mandatory controls, exit 0. ROUTE-01 is `NOT_APPLICABLE_WITH_REASON`, which means not checked.
- Allowed files: the 4 path classes in the work order. All counters were at 0 before edits.

## 2. Changed source (exact scope)

| File | Preimage LF SHA-256 | Final LF SHA-256 |
|---|---|---|
| `frontend/src/components/InterventionExecutionStatus.vue` | `756cba7b19b347e72315ef1490255c9413e46c933844a6f9d69f0400df36c81a` | `b174b3b610ae0b64eb08aa15c81d7dbad023058edcd86e77bc9e1642c5b8932a` |
| `frontend/src/__tests__/intervention-execution-status.test.ts` | `85c12c8d1023c19cbf120232c52af46b255dfcc0321537ae7875bcd157fd61e4` | `4bb59b72f5214c202926139f0f021df54845e71f15520ac2f17da3b347cecd9f` |

The protected files `interventionExecutionHold.ts` and `api/intervention.ts` are unchanged. `package.json`, `package-lock.json`, `backend/`, `src/api`, `src/i18n`, gates/CI, authority, continuity and registry are unchanged. No new module was added.

### Design implemented (contract items 1–6)

- **Separate read identity.** `readSeq` and an `activeRead: ReadOwner` hold the selection/read identity. They are independent of the execution/lifetime `generation` and the execution `controller`, and neither of those is incremented or aborted for a selection.
- **Captured owner per read.** Every `readSelected` (initial load, selection including A→B→A, refresh, and the post-send observation reuse) calls `beginRead`. This captures an immutable `{seq, requestId, mine, key}` and a per-read `AbortController`.
- **Superseding a read.** A newer read invalidates the older one even when the request id is the same: it increments `readSeq`, aborts the old per-read controller, and detaches the old parent listener immediately.
- **Parent signal link.** The per-read controller is linked to the parent signal: the execution `controller.signal` for UI reads, or `poll.signal` for observation. Aborting a superseded read therefore never aborts the execution controller or the poll loop. A poll deadline abort still reaches the in-flight read.
- **Ownership check.** `owns(read)` requires all of the following: mounted, current generation and lifetime, execution controller not aborted, external signal not aborted, `read.seq===readSeq`, `read.requestId===selectedId`, and the per-read signal not aborted. The check runs before every read-origin mutation:
  - status write
  - OBSERVED hold update and `loadHold`
  - error and null-status writes on reject
  - readiness follow-up request
  - `installReadiness` (and therefore the expiry timer)
  - readiness error
  - the return value used by `refreshSelected`'s finally-equivalent `loading=false`
- **Settle-path coverage.** These checks hold on resolve, on reject, and when the promise ignores abort, because the decision is the identity, not the signal.
- **Expiry timer callback.** The callback only acts if it is still the installed `expiryTimer`.
- **Lifetime and unmount.** `load()` (lifetime, permission, actor, token, tenant, case, external-signal change) and `onBeforeUnmount` call `invalidateReads()`.
- **`selectRequest` guard.** `selectRequest` now also returns early while `loading`. This mirrors the already-disabled buttons, so a superseded refresh can never leave its spinner stuck.
- **Unchanged.** POST, the fresh detail/readiness revalidation, the pre-send IN_FLIGHT marker, no-resend, the uncertain hold, the poll bounds (12 reads, ≥5 s, ≤60 s), DTO validation, localized labels and the expiry calculation are all byte-unchanged in logic.

## 3. Commands, exit codes, timing and counters

All commands ran from `frontend/` using installed dependencies only. No npx. `PYTHONDONTWRITEBYTECODE=1` was set for Python.

| # | Command | Source SHA (vue / test) | Start–end (UTC) | Exit | Result | Evidence |
|---|---|---|---|---|---|---|
| install | `npm ci --ignore-scripts --no-audit --no-fund` (D1) | – | 2026-11-04 ~19:49, 4.9 s | 0 | 209 packages | `worker_npm_ci.log` |
| T1 red | `node node_modules/vitest/vitest.mjs run src/__tests__/intervention-execution-status.test.ts` | `756cba7b…` (original) / `4bb59b72…` | 19:52:17.262–19:52:22.195 | **1** | 28 total: 18 pass, **10 fail**, 0 skip | `worker_test_red_original.{log,meta}` |
| T2 green | same command | `b174b3b6…` / `4bb59b72…` | 19:53:15.449–19:53:18.935 | **0** | 28 total: 28 pass, 0 fail, 0 skip | `worker_test_green.{log,meta}` |
| TC | `node node_modules/vue-tsc/bin/vue-tsc.js -b --pretty false` | `b174b3b6…` / `4bb59b72…` | 19:53:18.939–19:53:30.899 | **0** | no diagnostics | `worker_typecheck.{log,meta}` |

Counters against the budget:

| Item | Used |
|---|---|
| Focused test processes | 2 of 3 (third not used) |
| Typecheck | 1 of 1 |
| Install (D1) | 1 |
| Provider calls / browser / DB / app start | 0 / 0 / 0 / 0 |
| Secrets read | 0 |
| Repair rounds | 0 |
| Active time | from operator grant (19:48:35Z) to green/typecheck (19:53:31Z) about 5 min, plus return and gate writing. Total under the 45-minute cap |

Mandatory static gate runs are listed separately in §6. The 28 cases are one suite run twice, not 56 unique cases.

## 4. Acceptance matrix S01–S10

"Red on original" means T1 failed that test on preimage `756cba7b…`. All new selection-only tests assert `calls.execute` was not called (0 POST).

| ID | Test(s) | Red on original (T1) | Green (T2) | Notes |
|---|---|---|---|---|
| S01 | `S01: a stale rejection for A cannot clear B…` | **RED, behavioral**: stale A reject rendered "Progress cannot be checked right now" and nulled B's status | PASS | Also asserts no new readiness or execute call |
| S02 | `S02: A-new wins over A-old although the request ids match` | **RED, behavioral**: A-old overwrote A-new with "Request accepted; processing has not been confirmed" | PASS | Asserts no extra readiness, no hold written, no Start |
| S03 | 5 tests: A→B resolve, A→B reject, A→B→A resolve, A→B→A reject, obsolete short-lived READY timer | A→B reject **RED, behavioral** (error shown). Timer **RED, behavioral** (the old 3 s READY replaced the current 60 s expiry and disabled Start after 2.5 s). A→B→A ×2 **RED on the first assertion** (old readiness signal not aborted), so later behavioral asserts did not execute in T1. A→B resolve **passed** on the original (already guarded by request id; kept as a regression) | 5/5 PASS | Controlled fake timers used for the timer case |
| S04 | refresh overlapping a selection read (spinner kept); stale selection failure after refresh | **RED, behavioral** ×2: stale selection overwrote status during the refresh, and a stale reject showed an error after the refresh | 2/2 PASS | The overlap goes through enabled UI buttons (asserted not disabled). An initial-load overlap is not user-reachable (buttons are disabled while loading) and is guarded by `generation` |
| S05 | `S05: an obsolete status promise that ignores abort…` | **RED on the first assertion** (superseded read signal not aborted). Behavioral asserts did not execute in T1 | PASS | Promise resolves despite abort. Asserts no follow-up status/readiness and an unchanged UI |
| S06 | 5× prop change (actor, permission, token, tenant, case); unmount with pending read; unmount with READY timer | passed on the original (preservation) | 7/7 PASS | Asserts the pending signal is aborted, there is no late mutation and no follow-up request, and `vi.getTimerCount()===0` after unmount |
| S07 | current 403 → localized "You no longer have permission…" plus unknown status; current success; current readiness failure → unavailable | passed on the original (preservation) | PASS | No blanket suppression of current errors |
| S08 | obsolete CLAIMED read for A while current A shows an UNCERTAIN hold | **RED, behavioral**: durable hold rewritten UNCERTAIN→OBSERVED | PASS | Hold stays UNCERTAIN, B has no hold, no readiness, no Start, 0 POST |
| S09 | 9 pre-existing tests (storage failure zero POST, uncertain hold no resend, two-field POST, actor switch, deadline, double click, unknown refresh, 60 s poll abort) | passed in T1 | 9/9 PASS | The existing test bodies are unchanged. The poll test still observes abort at 60 s through the per-read signal linked to `poll.signal` |
| S10 | scope and typecheck | – | TC exit 0 | Exactly the 2 source/test files changed. No dependency, API, backend or execution-contract change |

**Red evidence limitation:** three tests (S03 A→B→A resolve and reject, S05) failed in T1 on an abort-signal assertion that precedes their behavioral assertions. The behavioral red of those three paths on the original is therefore **not proven by T1**. The worker did not spend the reserved third process to re-prove it. S02 and S08 are the behavioral same-id (A→B→A) red proofs on the original, for status and for the durable hold.

## 5. Advisory second opinion (not a review)

The operator allowed an independent advisory model. It ran a read-only review by hand-tracing the diff, without running tests or typecheck, and is not a Root review. Its result: **no blocking defects.**

What the advisor confirmed by tracing:
- No stale success, error, finally or timer path remains, including when a promise ignores abort, for the same id A→B→A, after lifetime change and after unmount.
- The execution, POST, no-resend, hold and poll bounds are unchanged.
- The poll deadline abort still reaches the in-flight read, and supersession cannot end the poll loop.
- No reachable stuck `loading` state was introduced.
- Listeners are detached on supersede and in finally.

Non-blocking advisor findings, retained for Root. The worker did not repair these, because of scope and the no-self-repair rule:

1. **Low, S02 `expect(holdOf(reqA)).toBeNull()`.** This assertion is vacuous on the original source, because no marker is seeded. S08 is the real hold-ownership proof.
2. **Low.** S03 A→B resolve and the S06 lifetime cases pass on the original source. They are regression guards, not race proofs, consistent with §4.
3. **Low.** The `signal.aborted` assertions indexed by mock call depend on call ordering, which makes them brittle. They are acceptable because the contract asks for abort where supported.
4. **Nit.** In S03-timer and S06, fake timers are installed after mount, so the first real 60 s timer is not fake-cleared. This is harmless because of the timer self-guard.
5. **Nit.** `readSelected` returns `true` on early exits that mean "not current". This is safe only because callers re-check `current()`. A comment or a tri-state return would be clearer.
6. **Pre-existing, not new.** If `props.signal` aborts without its identity changing, `loading` can stay true.

**Worker note on the advisor's claim.** The advisor says the S03 A→B→A cases "fail against the original source". That rests on tracing. In T1 those two tests, and S05, failed first on the abort assertion. Their behavioral red therefore remains **unproven by execution**, as stated in §4.

## 6. Static gate (worker-return)

- `python -B scripts/cvf_gates/cvf_downstream_gate_runner.py run --phase worker-return --project-root . --packet docs/work_orders/CCMAI_RUNTIME_140_SELECTION_READ_REPAIR.md --packet docs/reviews/R140_SELECTION_READ_WORKER_RETURN_2026-11-05.md` returned `RESULT: PASS phase=worker-return mandatoryControls=7 failed=0`, exit 0. It ran on the in-repo installed profile 1.3 with lock hashes verified, not the operator-local trusted route (D3).
- `python -B scripts/cvf_downstream_gate.py preflight --skip-catalog` returned PASS (6/6 executed gates passed), exit 0. `catalog` was SKIPPED: it is not counted as a pass, and the Windows CI job must run it. Provenance noted that `..\WORKSPACE_RULES.md` was not checked (portable mode).
- These static passes prove the checked repository state only.

## 7. Unknowns and not done

- No browser or runtime proof. M3-06 full browser lifetime remains R139 G2. There is no live provider or governance claim.
- The trusted operator-local bootstrap and the public doctor were not run in the cloud (D3).
- Node 22 vs CI Node 20 (D4).
- The operator allowed an advisory model and subagents. The advisor is read-only and does not replace the Root review.
- Root must decide on D1–D4 and perform the review, the material commit, the continuity update and the committed-range closure.
