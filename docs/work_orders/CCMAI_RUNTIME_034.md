# CCMAI-RUNTIME-034 — Offline Pancake proof harness

Status: REVIEW_PENDING

Date: 2026-10-03 (Asia/Saigon). Risk ceiling R2. [SPEC](../specs/PANCAKE_PROOF_HARNESS_R034_2026-10-03.md), [live packet](../reviews/F02_PANCAKE_LIVE_PROOF_PACKET_2026-10-03.md), immutable dispatcher seed `CVF_SESSION/authority/CCMAI-RUNTIME-034.json` committed at `d869624cc15f55b39516a36f8937454e617dc3b3`. Owner authorizes Codex coordination/review and manually transfers this work order to Claude. No live/provider/API/credential authority is granted.

## Roles and scope

Claude IMPLEMENTATION_WORKER / local BUILD COMMIT_STEWARD. Codex ORCHESTRATOR / SPEC_AUTHOR / WORK_ORDER_AUTHOR / planning COMMIT_STEWARD, then independent REVIEWER. User transfers order; no agent was automatically invoked. Codex does not implement product/harness source under this order. R033 and R030-R032 local FREEZE remain intact; all live/global F02/governance/hosted boundaries stay OPEN.

Allowed implementation paths are **new files only**: `backend/channels/pancake_proof.go`, `backend/channels/pancake_proof_test.go`, `backend/cmd/pancake-proof/main.go`, `backend/cmd/pancake-proof/main_test.go`. Document intended contract/current implementation in this SPEC/order/live packet and session/tranche/status/review/catalog/index records under gate conventions. Worker never edits immutable authority seeds. No changes to existing `pancake.go`, other adapters, engine, application wiring, schemas, dependencies, workflows, gate scripts or UI. New harness includes protective request admission; existing application behavior remains unchanged.

Permitted effects: synthetic in-memory transport fixtures, local harness tests/build/vet/docs/catalog/doctor/gates and bounded local commits. No actual channel/provider/external network, real credentials/config/.env inspection, customer data, persistent DB, media fetch, analyzer/notification/token rotation, parent edits, push/merge/deploy/FREEZE. Set `GOPROXY=off` and `GOTOOLCHAIN=local` for Go commands; missing cached prerequisites mean BUILD_BLOCKED, not implicit network authority. Do not use blocked mocks as CVF runtime-governance proof.

## Required execution

1. Rehydrate manifest/policy/bootstrap fallback/state/memory/active handoff/status/index, SPEC/order/seed/record and relevant shared learning. Run workspace doctor and local knowledge ingest; keep generated index outside the tracked changed set. Record worker role acknowledgment in active handoff **before source edits**, prevalidate/synchronize BUILD across state/front marker/handoff/status/order/tranche, run scoped preflight and stop if it fails. Use project-root cwd and `go -C backend`.
2. Consolidate PH-01..08 implementation/test/receipt/negative-observer plan. Create additive harness and offline-only CLI with explicit injected transport and no fallback network/config discovery. Actual unchanged adapter must perform traversal/mapping. User-facing CLI clearly labels synthetic output and has no live/credential mode.
3. Run discriminating acceptance tests, exact request observers and six finite applied semantic mutations from SPEC; restore bytes after each and rerun baseline. Preserve failed/inconclusive/surviving first attempts with cause and actual assertion. Do not weaken original tests or quietly implement broader adapter repairs.
4. Run required scoped Go/tests/build/vet/race and docs/catalog/doctor/diff/gates. Retain any failing command or unavailable check. No expensive DB-dependent backend suite is required here; label it NOT RUN rather than reusing prior results as fresh evidence.
5. Write `docs/reviews/PANCAKE_PROOF_HARNESS_R034_BUILD_2026-10-03.md`: exact source/build SHA, complete changed set, PH matrix, CLI invocation and receipt example, input/receipt schema, per-test request/redirect/time/side-effect observations, synthetic-canary sanitation results, mutations/restoration hashes, commands/exits/counts and limitations. No real secrets/data/requests or runtime governance claim. Update implementation truth only for what is built; prepared live work remains NOT DISPATCHED.
6. Synchronize REVIEW_PENDING everywhere, preserving R033 closure and historical failures. Record exact BUILD SHA in tranche `buildCommit` (following documentation commit if needed to avoid self-reference), commit locally and return SHA/changed set/evidence. Codex independently reviews. No self-approval, push or FREEZE.
7. Before each commit run default preflight, `python scripts/cvf_downstream_gate.py preflight --base origin/main --head HEAD`, explicit full changed-set preflight and `python -B -m unittest discover -s scripts/tests -p "test_cvf_downstream_gate*.py"`. Review the exact staged set; prevent generated caches/build outputs from entering it. No automatic cleanup outside verified generated paths.

## Return, failure and later authority

Return one exact local REVIEW_PENDING BUILD for Codex with all PH-01..08 evidence. Unsafe destination, leaked canary, exceeded aggregate cap, hidden default network, inventory false PASS, unaudited terminal, source-scope drift or an in-scope failed check prevents acceptance. Same-scope repairs follow accepted findings; third same-root repair needs REVIEW_COST_ESCALATION_REQUIRED. Existing adapter defects or missing input/dependency authority return BUILD_BLOCKED and a concrete proposal to dispatcher; seed unchanged.

Local harness work does not depend on a live tenant/token. Once this is independently reviewed, a separate live work order requires controlled tenant/page, independently prepared inventory, quiescence/retention and capture handling plus explicit credential/network authorization. Harness REVIEW_PASS never waives those requirements or proves provider compatibility/global completeness.
