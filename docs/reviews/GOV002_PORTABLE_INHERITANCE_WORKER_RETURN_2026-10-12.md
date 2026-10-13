# Worker Return - CCMAI-GOV-002-C1 Portable CVF Inheritance (round1 repair)

Memory class: EVIDENCE_RECORD
docType: review_artifact
Self-declared worker-return artifact: yes
Status: COMPLETE_PENDING_REVIEW
Packet: CCMAI-GOV-002-C1
Role: REPAIR_WORKER (round1, consolidated repair of R1-F1..R1-F4)
Actor identity: ccmai-gov002-worker
Model: claude-sonnet-5-5 (Claude Code, VS Code extension); this is the actual model the session exposes, not the Gemini self-declaration of the original return
Risk: R2
Worker: ccmai-gov002-worker
Reviewer: Codex /root (independent Local reviewer)
Commit mode: WORKER_MUST_NOT_COMMIT
executionHead: ca1c3aa29a6088916bbd5d31bef37e67277f8b9c
Authority seed: CVF_SESSION/authority/CCMAI-GOV-002-C1.json (seed commit f7bf57d0f4fe0e1920efdcc3609b6a8ea84cb68d, an earlier and different commit than executionHead)
Date: 2026-10-12

## Return Decision

COMPLETE_PENDING_REVIEW. All four binding findings (digest 4c062e7c74efefa2d2e9e6cd5811cbfb48b152ee7cd9c99905e6e8826f4c5bdb) are repaired inside the unchanged C1 seed. No commit, no push, no provider or application effect occurred; the working tree is left uncommitted for Local. The original round0 return and its five receipts are preserved verbatim under `docs/reviews/probes/gov002_review_round0/original-*`; this document replaces only the canonical return path, as the round1 admission directs.

This return proves static repository properties of the stated trees. It does not prove runtime AI governance, hosted CI, product readiness, docs55, R086 native or deployment status.

## Finding Repair Matrix

| Finding | Repair | Evidence (all under `docs/reviews/probes/gov002_review_round1/proof/` unless noted) |
| --- | --- | --- |
| R1-F1 strict parser | `scripts/cvf_downstream_gate.py`: the opt-in no longer re-implements the Status grammar with a regex. It loads the lock-pinned `check_gate_to_role_closeability.py` only after its LF-normalized SHA-256 equals the lock entry (the verified bytes are the executed bytes, no bytecode written), then judges the work order with the pinned `classify_status`. Backtick and tilde fences, duplicate, contradictory, malformed, blank and backtick-wrapped declarations now fail; plain, bold, annotated and CRLF forms pass. Legacy records without the two opt-in fields keep the original comparison untouched. | New `scripts/tests/test_cvf_downstream_gate_profile_compatibility.py` has 43 tests (was 20). `stage-discrimination.json`: against the preserved returned adapter (`gov002_review_round1/original-returned-cvf_downstream_gate.py.txt`, raw sha256 2f3be11a55c1bbc9039aaac79ff96c0d10708d25deacf11d5787e738fdf35865, equal to the round0 audit hash) in an isolated copy the new file FAILS (26 failures and 1 error across 15 tests, including the three captured false accepts); against the repaired gate in a second copy it passes 43/43. `stage-negatives.json` N15..N20 repeat fenced, tilde, backtick, malformed-second, duplicate and drifted-classifier cases through the real gate and real lock. |
| R1-F2 missing proof | Complete candidate-bound proof package; driver `gov002_review_round1/gov002_round1_proof.py` (stdlib only, disposable clones, no production mutation). | `stage-snapshot.json` (disposable candidate repo = HEAD plus every live change except declared evidence outputs, zero content mismatches, zero omitted paths); `stage-clone.json` (clone into another absolute container directory, initializer to `FRESH_CLONE_CONTINUITY_PASS`, doctor 27/27, sibling Core at the pinned public commit and clean, binding git-ignored, no original-machine path in instruction carriers, preflight 7/7, all 89 gate self-tests OK once, installer rerun `ALREADY_INSTALLED` with identical tree digest, Core `--trusted` run PASS, dirty and unrelated Core refused without being overwritten); `stage-negatives.json` (26 records: tampered copied module refused at stage 0 before any import in the pinned run, tampered runner, lock-hash tamper, workflow pin drift, consistent module-plus-lock edit refused by the CI pin and by the Core `--trusted` run, wrong pin argument, one-surface continuity drift, unknown standard field, missing `activeWorkOrders`, waiting-review order listed as executable, R2 self-review, ACTIVE-class order without its contract, plus controls N00, N13, N14); `stage-prci.json` (explicit `--base` ca1c3aa29a6088916bbd5d31bef37e67277f8b9c, `--head` HEAD and all three pins read from the workflow and cross-checked against the lock: 7 mandatory controls individually PASS; unknown base, missing base, unset base environment variable, head not checked out and wrong pin are each refused); `stage-buildtime.json` (explicit `--packet docs/work_orders/CCMAI_GOV_002.md` while Status was ACTIVE: CVF-DG-APPL-01 is a CHECK that names the packet and ROUTE-01 checked it, not a no-candidate N/A). |
| R1-F3 guide | `docs/reference/CVF_PROJECT_PORTABLE_CONTINUATION.md` rewritten: real repository and delivery branch, container layout, Python and PyYAML 6.0.2 prerequisites, `core.longpaths` and short-container advice, working relative links, progressive reads (bootstrap model, memory, active handoff named by the model, targeted full-state lookup only), explicit scoped-proof limits and the preserved docs55 and runtime gaps. `AGENTS.md` gains "Installed Public Gate Profile" with the phase command table, `--packet` use, the N/A-is-not-checked rule, the status mapping and the hook-template-only note. | Link check and gate reads in `stage-final.json`; `stage-clone.json` shows the guide's commands succeeding in a clean container. |
| R1-F4 continuity and return | BUILD acknowledged in the handoff before any edit; all surfaces synchronized to REVIEW and REVIEW_PENDING with `activeWorkOrders` empty, record opted in (`workOrderStatus` COMPLETE_PENDING_REVIEW), work order Status COMPLETE_PENDING_REVIEW, this return carries the closeability recheck and a candidate manifest. | Handoff sections "Round1 Repair BUILD Acknowledgment" and "Round1 Repair Return"; `docs/reviews/probes/gov002_round1_candidate_manifest_2026-10-12.json`; `docs/reviews/probes/gov002_round1_worker_receipt_2026-10-12.json`. |

## Procedural Deviations And Corrections (original facts preserved)

- The original worker pass recorded no BUILD acknowledgment. That deviation stays recorded in state, handoff and tranche; the round1 BUILD acknowledgment (2026-10-12T19:16:07+07:00, before the first repair edit) does not backdate or replace it.
- The original report named the authority seed f7bf57d as its execution base. The actual execution HEAD for this repair is ca1c3aa29a6088916bbd5d31bef37e67277f8b9c; both values are now labelled separately.
- The original report quoted 66 tests in 22.544s while its stored receipt said 18.324s. That mismatch is preserved in the round0 snapshots and not re-litigated; the round1 counts and timings come from `stage-clone.json` (89 tests in 10.837s) and `stage-discrimination.json`.
- Model and telemetry: the original return self-declared Gemini and is not attributed to Sonnet. This repair ran as claude-sonnet-5-5; reasoning effort is a session parameter of 10 whose scale is not documented to the worker; token usage and bill are NOT_AVAILABLE_WITH_REASON (not exposed to the worker). `x-actualWorkerModel` and `x-ownerSubagentPreference` in the state file are historical R086 values and do not describe this worker; Local owns that field.

## Newly Found Defects Fixed Or Disclosed Inside This Repair

1. Fixed (found by the fresh-clone proof): two registry ids carried a double hyphen (`gov002--cvf-gate-profile-lock-json`, `gov002--github-workflows-cvf-downstream-gates-yml`). Windows PowerShell 5.1 sorts ids culturally and pwsh ordinally, so the generated `docs/INDEX.md` differed by shell and doctor check 25 failed in the clone. The two ids were renamed to single-hyphen forms; the views were regenerated with the gate-selected pwsh and `-Check` passes under both pwsh and Windows PowerShell 5.1. Only registry metadata changed.
2. Fixed in the guide: a clone below a long absolute path fails with `Filename too long` on Windows; the documented clone command now sets `core.longpaths` and recommends a short container directory.
3. Disclosed, not fixable inside this seed: a plain local `run` without `--expect-*-sha256` imports project modules before `verify_install` reports a tampered module, so tampered code can run once before the failure (`stage-negatives.json` N01b-unpinned-disclosure). The pinned CI invocation and the Core `--trusted` doctor refuse before import (N01, N05a, N05b). Mitigation is the pinned workflow and the doctor; the Core owns the design.
4. Observed, not changed: the original pass rewrote `.cvf/manifest.json` and `AGENTS.md` formatting (BOM removed, indentation normalized). Content values were preserved and the doctor passes; Local may prefer a smaller diff.

## Actual Invocations (project root, `python -B` or `PYTHONDONTWRITEBYTECODE=1`)

| Check | Where | Outcome |
| --- | --- | --- |
| Rehydration reads and acknowledgment; legacy `scripts/cvf_downstream_gate.py preflight` before the first edit | live | PASS 7/7 |
| New compatibility tests during development | live | 43 tests OK |
| BUILD-time runner phases pre-dispatch, worker-return, reviewer-fast, pre-commit with explicit packets while Status was ACTIVE | live | PASS; APPL-01 CHECK on the work order |
| Installer rerun with `--core-commit e9ddcbcdcac2a9df520f5eb0761dbdb609b9deba` | live and clone | `ALREADY_INSTALLED`, tree digest unchanged |
| Proof stages snapshot, clone, negatives, prci, discrimination | disposable repos under a short work directory | `unexpectedCount` 0 in every stage file |
| Final-state phases pre-dispatch, worker-return, reviewer-fast, pre-commit, preflight, pinned closeability checker with `--enforce`, Core doctor, `git diff --check`, protected-path check | live, after this document existed | recorded in `stage-final.json`; the round1 receipt indexes every stage file hash |
| Not run on purpose | n/a | docs build (docs55 stays a separate P0), application tests, Docker, provider calls, hosted CI |

CVF-DG-APPL-01 and CVF-DG-ROUTE-01 print `NOT_APPLICABLE_WITH_REASON` in several rows of the proof where the work order is no longer ACTIVE or no dispatch packet is in scope. Those rows mean the contract was not checked in that tree; contract coverage is claimed only for the BUILD-time rows listed above.

## Changed Files And Candidate Manifest

The exact path list with raw and LF-normalized SHA-256 for every changed or untracked non-ignored path is `docs/reviews/probes/gov002_round1_candidate_manifest_2026-10-12.json`; it also lists what changed since the round0 audit intake and the faithful-snapshot identity. Round1 touched: `scripts/cvf_downstream_gate.py`, `scripts/tests/test_cvf_downstream_gate_profile_compatibility.py`, `docs/reference/CVF_PROJECT_PORTABLE_CONTINUATION.md`, `AGENTS.md`, `docs/work_orders/CCMAI_GOV_002.md` (Status line only), the state, tranche, implementation-status, bootstrap-model, memory and handoff surfaces, the catalog registry and its two generated views, this return and the new evidence under `docs/reviews/probes/`. The 49 historical handoff status migrations, the installed profile files, the lock, the generated workflow, the learning template and all round0 snapshots are byte-identical to the round0 audit intake (see `unchangedSinceRound0Count`). No backend, frontend, policy, seed, existing workflow or existing legacy test file changed.

One path is not worker work: `docs/reviews/planning/CCMA_RUNTIME_CVF_INHERITANCE_ROADMAP_2026-10-12.md` (Codex /root planning draft, Status DRAFT, HOLD) appeared in the shared worktree at about 19:43+07:00, after the candidate snapshot. At the same moment Local added a "queued runtime inheritance planning" pointer to the state, handoff and memory surfaces, which made my first snapshot drift from the live tree. The worker did not author, edit, stage or register any of that. To keep the proof bound to the tree Local will actually review, the snapshot, clone, negative and pr-ci stages were re-run after it appeared, so the final proof covers the live tree including Local's additions; the manifest marks the planning draft `notWorkerAuthored`, and the worker takes no responsibility for its content. The final-state gates also pass with it present.

The disposable candidate snapshot used for the clone, negative and pr-ci proof is commit 933de8465e8c5995637ff53c805ae47eedf0418f, tree 4772043ee0c50e5ea254ab617ab279207df66894, parent ca1c3aa29a6088916bbd5d31bef37e67277f8b9c (a throwaway local repository, not a project commit). The manifest's `liveDriftSinceSnapshot` must be empty; it is the machine check that no live candidate file changed after that snapshot.

The faithful snapshot excludes only this return, the round1 receipt, the manifest, the proof driver and the stage files, because they record results about the snapshot; its catalog views omit the registry entries of those paths.

## Return-Time Closeability Recheck

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
outsideAuthorityBlockers: NONE
nextRepairRoute: INDEPENDENT_LOCAL_REVIEW_OF_RETURNED_EVIDENCE
workerRedispatchAllowed: NO

This return does not request a redispatch. If Local finds a new independent root cause, one more same-authority repair is possible under the stop rule (round two would be the last without a new root cause).

## Review-Cost And Convergence Record

reviewRoundCount: 1
workerRepairTurnCount: 1
newRootCauseCountThisRepair: 0 independent of the four admitted findings; two dependent defects (registry ids, long path) found by the proof and fixed inside F2/F3
providerCallCount: 0
externalInvocationCount: 0
tokenOrQuotaUsage: NOT_AVAILABLE_WITH_REASON: not exposed to the worker
elapsedMinutes: BUILD acknowledged 19:16:07+07:00; return-state synchronization 19:34:22+07:00; final stage timestamps are in the stage files
commitCount: 0

## Claim Boundary, Open Items And Export

- Open and unchanged: the 55 inherited docs deadlinks (P0), R086 source-only acceptance with native readiness NOT_RUN, durable fence/recovery integration OPEN, Facebook and Zalo OA accounts PARKED, deployment UNKNOWN. No hosted CI result exists for this package; the generated workflow is configuration only.
- Public Export Disposition: DEFERRED_PRIVATE_ONLY for the private CVF parent; the project branch `codex/cvf-portable-inheritance` currently holds only the dispatch preparation commit. Nothing was pushed by this worker.
- No shared learning record was written in this repair; the reusable lessons are: do not re-implement a pinned grammar, delegate to the pinned classifier after a hash check; registry ids must sort identically under ordinal and culture comparison; a fresh-clone proof catches what same-tree gates cannot. Local may promote them under `docs/reviews/learnings/`.

## Agent Operation Trace Block

| Field | Value |
| --- | --- |
| Actor | ccmai-gov002-worker |
| Phase transitions | REVIEW to BUILD (acknowledged before edits), BUILD to REVIEW_PENDING |
| Manifest delta | NONE: every path is enumerated in the candidate manifest |
| Commit | NONE |
