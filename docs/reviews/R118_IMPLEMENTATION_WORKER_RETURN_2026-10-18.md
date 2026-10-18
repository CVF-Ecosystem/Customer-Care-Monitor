# R118 implementation worker return

Status: COMPLETE_PENDING_REVIEW
Tranche: CCMAI-RUNTIME-118
Risk: R2
Worker: /root/r118_ui_worker
Reviewer: Codex /root
Role: /root/r118_ui_worker IMPLEMENTATION_WORKER
Authority: committed dispatcher seed `40a301fa33e0b9cb364b283008d08b449da39d18`; dispatch material `de3eba55`; source only, R2.

## Delivered source behavior

- The detail view selects the active proposal through the case/top-level pointer and validates its current run, snapshot, digest and positive integer revision. Contradictory pointers, missing target, duplicate target or lineage mismatch leave proposal actions disabled with a readable notice. Legacy null/omitted pointer selects only one full-lineage candidate while no active execution exists, matching the backend action contract in `backend/engine/intervention_m3.go` lines 146–175.
- The displayed heading identifies the current proposal. Proposal-bound writes use that proposal's ID and revision. `ASSIGN_DATA` and `ESCALATE_DATA` send an empty proposal ID and revision zero even if the detail contains a valid proposal. Existing one-shot, permission, route lifetime, conflict and refresh behavior remains exercised by the focused tests.
- Vietnamese and English history labels cover `OBSERVE_DATA`, `REQUEST_RERUN` and `EXECUTE_RERUN`. Only the exact system reason `approved execution` on `EXECUTE_RERUN` is translated; other reason text stays verbatim. Lowercase `bound_currentness_unverified` maps to the existing readable local-match source label for history presentation.

## Evidence and limits

Focused frontend invocation 1/3: `node node_modules/vitest/vitest.mjs run src/__tests__/intervention-ui.test.ts src/__tests__/intervention-presentation.test.ts src/__tests__/intervention-api.test.ts`, from `frontend/`; exit 0, 3 files and 24 tests passed. Raw output and exact command/exit receipt: `docs/reviews/probes/r118_worker/focused_frontend.*`.

Focused frontend invocation 2/3: `node node_modules/vue-tsc/bin/vue-tsc.js --noEmit -p tsconfig.app.json`, from `frontend/`; exit 0. Raw output and exact command/exit receipt: `docs/reviews/probes/r118_worker/vue_tsc.*`. No third invocation used. Exact final SHA-256 file bindings: `docs/reviews/probes/r118_worker/source_hashes.json`. `git diff --check` had no whitespace finding (Git line-ending warnings only).

The new tests discriminate pointer choice from array order, unknown/mismatched/duplicate/contradictory pointers, unique and blocked legacy cases, proposal-less data actions, and Vietnamese/English system-history wording with human text preserved. Existing reader, route and conflict tests passed. These are source/component checks only. No product browser, backend/DB fixture, provider call, secret, runtime start or hosted CI check occurred. Changed-screen proof, B08 raster, B09 offline, full M2/M3, pilot, release and M1 business decisions remain open; R117 real call evidence is inherited without replay. Historical Dashboard failures are not rerun or claimed fixed.

## Closeability recheck and handback

Rechecked the active order's gate-to-role contract at worker return: authorized frontend/test paths and this return/probe family were used; worker did not edit the seed, tranche record, continuity or commit. `focused_checker_tests` and source identity evidence are present. `worker_return_fast`, independent `reviewer_fast`, material pre-commit and continuity/range closure remain Root-owned. No scope or external-effect boundary changed and no same-scope repair round was consumed.

Profile `worker-return` gate: explicit order and return packets, mandatory controls 6/6 PASS, exit 0; raw receipt `docs/reviews/probes/r118_worker/worker_return_gate.*`. The first pass classified the return as non-role-bearing before the Risk/Worker/Reviewer header was added; the final recorded gate pass is the role-bearing recheck.

Disposition requested: independent Root REVIEW of the exact six source files and evidence; Root owns any accepted repair, commit and continuity synchronization. This return does not assert FREEZE or rendered UI acceptance.
