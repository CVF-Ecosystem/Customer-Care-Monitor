# CCMAI-GOV-001 BUILD evidence — downstream CVF machine gates

**Date:** 2026-10-01 · **Phase:** BUILD → REVIEW_PENDING · **Risk:** R2 · **Role:** IMPLEMENTATION_WORKER + COMMIT_STEWARD (Claude); independent REVIEWER: Codex
**Authority:** [SPEC](../specs/CVF_DOWNSTREAM_MACHINE_GATES_2026-09-30.md), [applicability decision](../decisions/CVF_DOWNSTREAM_GATE_APPLICABILITY_2026-09-30.md), [work order](../work_orders/CCMAI_GOV_001.md). Planning commit `8e776b5` (base `df0c4cc`); core `26c686c` read-only.
No product source, DB, provider/channel call, secret, push, merge or freeze. The pre-existing untracked `knowledge/_index.json` is not part of the commit. A pass of these checks proves the checked repository state only; it is not evidence that CVF controls runtime AI behavior.

## What was built

| Path | Purpose |
|---|---|
| `scripts/cvf_downstream_gate.py` | Python-stdlib gate (PyYAML only for the workflow-coverage gate). `preflight` runs seven fail-closed gates; `--base/--head`, `--files`, `--only`, `--skip-catalog`; exit 0 only if every selected gate passes; diagnostics name fields and paths, never credential values. |
| `scripts/tests/test_cvf_downstream_gate.py` | 28 tests (positive baseline, one mutation per negative case, real-git range tests). |
| `CVF_SESSION/tranches/CCMAI-GOV-001.json` | First structured tranche record (contract below). |
| `CVF_SESSION_MEMORY.md` | Machine-readable `<!-- cvf-front-marker {...} -->` block. |
| `.github/workflows/governance.yml`, `frontend.yml` | New PR checks (below). Backend and docs workflows are unchanged. |
| `AGENTS.md` | Added the "Downstream Machine Gates" section (invocation and contract); existing policy untouched. |
| `docs/catalog/ARTIFACT_REGISTRY.json`, `docs/INDEX.md` | Registered the tool, its tests and the tranche-record family; views regenerated with the catalog tool. |
| state, handoff, `IMPLEMENTATION_STATUS.json`, this evidence | Continuity and record. |

`.cvf/manifest.json` and `.cvf/policy.json` are unchanged (no new pointer field was needed; the marker lives in session memory).

## Applicability matrix (what actually runs)

| Control family | Downstream invocation | Status |
|---|---|---|
| Provenance / isolation / policy | gate `provenance`: manifest phase model, 40-hex core commit, live-evidence and mock-only-for-UI flags, required docs exist inside the project (paths outside the project, e.g. `..\WORKSPACE_RULES.md`, are skipped with a note: no sibling core on a runner), policy flags | Runs locally and in CI |
| Active continuity / next move | gate `continuity`: state vs handoff header (mode, phase token, role, tranche IDs in next move, parked checkpoint SHAs), memory front marker (mode, phase, handoff, parked flag, active tranche named in next move), `IMPLEMENTATION_STATUS.currentPhase`, handoff path inside the project, optional bootstrap read model must match if present (`BOOTSTRAP_MIGRATION_PENDING` note otherwise) | Runs locally and in CI |
| Phase / role / work-order dispatch; worker return, review, closure | gate `tranche` over `CVF_SESSION/tranches/<ID>.json`: status→phase consistency and record phase equal to session phase, allowed status transitions in `history`, required roles, **R2+ reviewer independent of implementation/repair worker**, R3 needs approval, `allowedPaths` + `prohibitedEffects` present, changed work orders must be bound to a record, changed files (worktree plus `baseCommit..HEAD`) must be inside `allowedPaths` or session/status/review records, `REVIEW_PASS` needs disposition, 40-hex `buildCommit` and existing review evidence, a pending/BUILD record cannot claim freeze | Runs locally and in CI; prospective (active tranche only) |
| Claim boundary | gate `claims` on the active tranche's changed Markdown: BUILD evidence cannot claim a frozen/closed state; a line claiming CVF governs AI/agent behavior needs `governanceReceipt` (real provider call, provider/model/request/response, not mock/synthetic); a line claiming a GitHub Actions result needs `ciRun` (run URL, 40-hex sha, conclusion success). Negated or pending wording ("not", "no", "unverified", "pending", "requires") is not a claim. The gate checks the `ciRun` shape only and says so; it never verifies a run | Runs locally and in CI |
| Secrets / public safety | gate `secrets` over the changed set: credential-type files (`.env*` except examples, keys, pems), private-key blocks, cloud/vendor key shapes, quoted credential literals; values never printed; `cvf-allow-secret-fixture` marks deliberate fixtures; ignored local secrets are never read | Runs locally and in CI |
| Workflow trigger coverage | gate `workflows` parses `.github/workflows/*.yml` and requires each path class (backend, CI-gate helper, frontend, docs, `.cvf`, `CVF_SESSION`, memory, status, `AGENTS.md`, the gate tool, workflows) to reach a relevant `pull_request` workflow | Runs in CI (needs PyYAML, pinned) and locally |
| Governed artifact catalog | gate `catalog` runs `manage_cvf_downstream_catalog.ps1 -Check` | Local and CI **windows** job: the catalog tool uses backslash paths and does not run on Linux |
| Backend DB execution | R019 five-sentinel workflow unchanged | Inherited; F08 public run still pending |
| Frontend / docs | new `frontend.yml` (`npm ci`, `npm test`, `npm run build`); existing `docs.yml` | CI (new) / unchanged |
| Core-only web, corpus, ADIF, agent-workspace, provider-packet gates; core `check_*` scripts | Not run: they depend on core state files, gate IDs and packet templates absent here (see the applicability decision). No claim that core gates run | Not applicable |

CI wiring: `governance.yml` has **no path filter** (every pull request and pushes to main): checkout with full history at the PR head, PyYAML 6.0.2, gate self-tests, `preflight --skip-catalog --base origin/<base> --head HEAD` (ubuntu) plus a `windows-latest` job running the catalog gate. Permissions are `contents: read`; nothing deploys on a PR.

## Positive and negative evidence

- **Pre-migration failures** (gate first, before the marker/record/workflows existed): `continuity` failed (no front marker), `tranche` failed (no active tranche), `workflows` failed for 8 path classes (frontend, `.cvf`, session, memory, status, `AGENTS.md`, gate tool, workflows); provenance, claims, secrets and catalog passed.
- **The two observed drifts** are fixtures that pass as a baseline and then fail: a memory front marker saying `WORK_ORDER` against state `REVIEW`, and `IMPLEMENTATION_STATUS.currentPhase` `WORK_ORDER` against `REVIEW`. (Codex had already corrected the live files, so the fixtures reproduce them.)
- **Current repository, after migration:** `python scripts/cvf_downstream_gate.py preflight` → 7/7 PASS, including the catalog; a PR-range dry run `--base HEAD~60 --head HEAD --skip-catalog` also passes; a secret scan of every tracked file passes.
- **Tests:** `python -B -W error -m unittest discover -s scripts/tests` → 39 OK (28 new + 11 R019 gate tests). With PyYAML masked, the gate tests report 28 run / 4 skipped and pass, so the backend workflow's `unittest discover` (which also globs `scripts/tests`) stays green without PyYAML.
- **Mutations of the gate** (each applied to `cvf_downstream_gate.py`, tests run, file restored; the restored file passes): status phase ignored (2 failed), marker mode ignored (1), self-review allowed (1), `.env` allowed (2), path scope not enforced (2), `buildCommit` unchecked (1), governance-claim check off (1), workflow coverage no-op (2), freeze word in BUILD evidence allowed (1), history transitions unchecked (1). Every mutation was detected.
- Fixtures cover: continuity drifts (marker, status, handoff mode, tranche IDs, parked mismatch, missing/escaping handoff, malformed JSON, bootstrap model), provenance (phase model, policy flag, missing/escaping required doc, bad core commit), tranche (path scope, R2 self-review, repair-worker self-review, `REVIEW_PASS` evidence, freeze claims, invalid transition, phase mismatch, unbound work order, missing record, R3, empty required lists), claims (freeze word, governance receipt variants incl. mock/synthetic/false/empty, Actions claim with good and bad `ciRun`), secrets (files, key block, cloud key, literal, allow marker, binary, local secrets file not read, no value in output), workflows (bypass paths, missing/push-only workflow, real repository), CLI exit codes, and real-git range/worktree scope with a committed `.env` found in the PR range.

## Other gates

| Check | Result |
|---|---|
| Frontend `npx vitest run` | 18 files, 186 tests PASS |
| Frontend `npm run build` | built |
| Docs `npm run docs:build` | PASS after fixing one pre-existing dead link (below) |
| Catalog `-Check` (after `-Write` of the views) | PASS |
| Workspace doctor | 25/25 PASS |
| `git diff --check` | clean (CRLF warnings only) |
| Backend suite | not rerun: no backend or backend-workflow change; R019 evidence inherited |

**Incidental fix:** the docs build failed on a dead relative link in my earlier R020 BUILD evidence (`../../backend/api/handlers/agents.go` is outside the docs site). It was turned into a plain code path in `docs/reviews/RUNTIME_AGENT_HTTP_PERMISSION_ADMISSION_F01A_BUILD_2026-09-30.md`; without it the docs PR check would fail. No other content of that record changed.

## Claim limits

- Not proven: any GitHub-hosted run of the new workflows (PyYAML install, `fetch-depth: 0`, the PR head checkout, pwsh/Windows catalog job, frontend cache path) and the F08 backend run. Those need the authorized PR; Codex verifies actual runs, SHA and conclusions on GitHub.
- The path-scope check compares the active tranche's `baseCommit..HEAD` and the worktree. A later tranche changes the active record and base; if a base branch moves inside a PR merge checkout the range can include unrelated files (the workflow therefore checks out the PR head commit). Claim checks are prospective and read only the active tranche's changed files, so accepted historical records are not re-judged.
- The claim and secret detectors are bounded regular-expression heuristics with documented false negatives; negation wording can suppress a real claim, and the receipt check validates fields, not authenticity.
- No local git hook is installed; the documented local command and the PR workflow are the invocations.
- The core checkers were not executed or copied; this is a downstream adapter derived from their control families.

## Changed set

`scripts/cvf_downstream_gate.py`, `scripts/tests/test_cvf_downstream_gate.py`, `CVF_SESSION/tranches/CCMAI-GOV-001.json`, `.github/workflows/governance.yml`, `.github/workflows/frontend.yml`, `AGENTS.md`, `CVF_SESSION_MEMORY.md`, `CVF_SESSION/ACTIVE_SESSION_STATE.json`, the active handoff, `IMPLEMENTATION_STATUS.json`, `docs/catalog/ARTIFACT_REGISTRY.json`, `docs/INDEX.md`, `docs/reviews/RUNTIME_AGENT_HTTP_PERMISSION_ADMISSION_F01A_BUILD_2026-09-30.md` (link fix), and this evidence.

## Disposition

`REVIEW_PENDING` for Codex independent REVIEW; only after `REVIEW_PASS` may Codex open the authorized PR. Claude does not self-approve; no push, merge, deployment or freeze.
