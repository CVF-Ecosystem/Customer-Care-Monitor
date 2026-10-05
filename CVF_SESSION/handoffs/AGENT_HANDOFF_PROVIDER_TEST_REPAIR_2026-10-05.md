# Provider test/evidence repair handoff

Status: ACTIVE

## Current State

- Project: Customer-Care-Monitor-AI
- Current mode: REVIEW
- Active phase: REVIEW
- Active role: Codex independent REVIEWER after Claude REPAIR_WORKER handback; no self-approval
- Next allowed move: CCMAI-RUNTIME-051 REVIEW_PENDING under separate committed seed a54cb73007f081fe4bbaa4baa11d28fed4a7d937 and exact test repair commit a4b378ae036ad767728fb2e0655559e9cf750b47: Codex independent REVIEWER next verifies findings resolution R050-R1-01..03 (RP-01..04), canonical production source 727d3229338e9b29c749612677a08b7fd1c65428 byte-identity, and repair evidence in docs/reviews/ANALYZER_LAZY_PROVIDER_R050_R1_REPAIR_2026-10-05.md. Original R050 authority seed 1008ab41f0693e2814cd06dfdb0fed98273a1167 untouched; R050 remains CHANGES_REQUIRED pending review. No self-approval/FREEZE; accounts parked under OWNER_DEFERRED_FACEBOOK_ZALO_OA_ACCOUNTS.
- Parked operator checkpoint: OWNER_DEFERRED_FACEBOOK_ZALO_OA_ACCOUNTS: Facebook account and Zalo OA account setup/credentials/connectivity/live tests parked until owner resumes; prior local acceptance preserved.

## Separate authority and activation acknowledgment

After committed seeda54cb73007f081fe4bbaa4baa11d28fed4a7d937, canonical manifest/policy/state/memory/predecessor handoff/implementation/index rehydrated. Codex ORCHESTRATOR -> SPEC_AUTHOR / WORK_ORDER_AUTHOR -> SESSION_SYNC_STEWARD / planning COMMIT_STEWARD declared; current WORK_ORDER/R2, live evidence required YES, actual readonly core8a4119e11db00e774ed8e7cf7d9a8caa309e81d1, manifest26c686cc warn-only mismatch, doctor PASS WITH NOTE25/1; bootstrap absent nonblocking, local knowledge TEMP/no POST. Owner next authorizes the committed review's separate fixture repair path under standing work-order delegation. INTAKE/DESIGN/SPEC/WORK_ORDER complete, RP-01..04 inherits unchanged LP-01..08. Separate R051 allows only two test paths and successor evidence; production/source contracts and original R050 seed/path ceiling untouched. Claude remains REPAIR_WORKER/BUILD commit steward, independent Codex reviewer; no subagent invoked. Gate46 PASS11.967s, seed docs7.28s/default/PR/staged7/7/catalog/diff PASS. No BUILD yet; worker committed before-edit acknowledgment/gate is next. R050 CHANGES_REQUIRED/failures retained, R049/prior closures preserved; accounts parked.

## Worker rehydration and BUILD acknowledgment (Claude, 2026-10-05)

Role transition WORK_ORDER_AUTHOR (Codex) -> REPAIR_WORKER (Claude), with BUILD COMMIT_STEWARD held by Claude, acknowledged before any test or evidence edit. Rehydrated: manifest, policy, canonical state, memory, this handoff, IMPLEMENTATION_STATUS, docs index, R051 order/SPEC/seed/tranche record, and shared repair-workflow learning. Workspace doctor 25/1 PASS WITH NOTE; gate tests 46/46 PASS (15.077s); downstream gate preflight 7/7 PASS. Compact bootstrap read model absent: BOOTSTRAP_MIGRATION_PENDING, nonblocking. Separate seed a54cb73007f081fe4bbaa4baa11d28fed4a7d937 verified present at baseCommit and unchanged. Original R050 authority seed 1008ab41f0693e2814cd06dfdb0fed98273a1167 and canonical production source 727d3229338e9b29c749612677a08b7fd1c65428 remain byte-identical.

CVF Agent Declaration: Customer-Care-Monitor-AI; CVF core ../.Controlled-Vibe-Framework-CVF at 26c686cc99b8be965d2760f27fe875b03376c643; phase BUILD; risk ceiling R2; live evidence required YES (no governance claim made; synthetic provider evidence is application proof only); active role REPAIR_WORKER / BUILD COMMIT_STEWARD; active handoff this file; next allowed move as header; parked checkpoint OWNER_DEFERRED_FACEBOOK_ZALO_OA_ACCOUNTS.

## Worker repair execution and handback (Claude, 2026-10-05)

Exact test repair commit `a4b378ae036ad767728fb2e0655559e9cf750b47` (`test(engine): repair ownership fixture and initialization test regressions (R051)`).
- **RP-01 / R050-R1-01**: Provider-selection fixture in `backend/engine/job_run_ownership_test.go` gained eligible synthetic source `prov-fail`; all bounded-error, finished-state, job status, ownership release, and readmission assertions preserved. Relevant ownership regressions pass cleanly.
- **RP-02 / R050-R1-02**: `countingProvider` refactored to track `singleCalls`, `batchCalls`, and `itemsCount` distinctly. Accurate semantically valid assertions for single and batch modes implemented.
- **RP-03 / R050-R1-02**: Production `getProvider` branch tested with corrupt encrypted key `X'DEADBEEF'` and zero settings queries trapped via `trapAISettingsQueries` on no-work paths. Candidate query error fixture added to `TestLP02` via `failTable(t, "conversations", ...)`. All-failed prep with corrupt key verified. Mixed prep with corrupt key on production path verified. Positive controls for missing and corrupt key verified.
- **RP-04 / R050-R1-03**: Truthful isolated campaign executed via `docs/reviews/probes/r050_r1_campaign_runner.ps1` with `--internal` network, disposable MySQL, no host ports, `docker rm -f -v` teardown. Full machine receipt saved at `docs/reviews/probes/r050_r1_worker_receipt.json`. 42 top-level test suites, 98 completed events, 98 PASS, 0 FAIL, 0 SKIP. Build and vet pass. Mutation `M_EAGER_INIT` killed (exit 1) and restored byte-for-byte (exit 0).
- Handed back to Codex as independent REVIEWER under `REVIEW_PENDING`. No self-approval or FREEZE. Canonical production source and original R050 authority seed remain untouched; accounts parked under OWNER_DEFERRED_FACEBOOK_ZALO_OA_ACCOUNTS.
