# CCMAI-GOV-001 GOV1-R1 repair evidence — downstream machine gates

**Date:** 2026-10-01 · **Phase:** BUILD (repair) → REVIEW_PENDING · **Risk:** R2 · **Role:** REPAIR_WORKER + COMMIT_STEWARD (Claude); independent REVIEWER: Codex
**Authority:** [independent review](CCMAI_GOV_001_INDEPENDENT_REVIEW_2026-10-01.md), the GOV1-R1 addendum in the [work order](../work_orders/CCMAI_GOV_001.md) and the reviewer-owned authority seed `CVF_SESSION/authority/CCMAI-GOV-001.json` (read-only for the worker; unchanged). Parent BUILD `54663eb`; repair base `67d743e`; core `26c686c`.
No push, merge, deployment, provider/channel call, parent-CVF work or freeze. Product source, backend/docs workflows and the R020 link repair are untouched. The earlier [BUILD evidence](CVF_DOWNSTREAM_MACHINE_GATES_GOV001_BUILD_2026-10-01.md) stays as recorded.

## Findings and repair

| Finding | Repair in `scripts/cvf_downstream_gate.py` |
|---|---|
| F1 whitespace-split Git paths dropped hostile names | All Git path lists use `-z` and split on NUL. Scope lists include deletions (a deleted out-of-scope file is still a change); content scans (secrets, claims) read only files that exist. Committed `backend/out of scope.go` and `secrets/my key.pem` are now reported by name (POSIX also covers a newline inside a name). |
| F2 git/range failure became an empty change set | New `GateError`: every Git call raises on failure with a bounded message. `--base`/`--head` are validated as commits; the tranche `baseCommit` must be a 40-hex, existing commit and an ancestor of `HEAD`; `git diff base...head` failures (for example no merge base), a non-git tree without `--files`, and an unreadable changed-file set make `tranche`, `claims` and `secrets` fail with `changed-file set unavailable: …`. Explicit `--files` stays the only non-git mode. Secrets in a PR run scan the range plus uncommitted work and do not depend on the tranche pin. |
| F3 next move bound only by tranche ID; stale work-order status | The handoff header's next move must equal `state.nextAllowedMove` after normalization (whitespace, backticks, `**`, curly quotes); same-ID text that permits self-approval now fails. The active work order's (and any changed bound order's) `Status:` line, read from its first 20 lines, must be a known state equal to the tranche record's status; stale, unknown and missing lines fail. The work order's history text is otherwise untouched. |
| F4 worker-editable record was its own authority | `check_authority` compares the record to `CVF_SESSION/authority/<ID>.json`: risk and roles equal, `allowedPaths` a subset (no widening), seed `prohibitedEffects` all retained. **Scope is judged by the seed's `allowedPaths`**, not the record. In a git checkout the seed must be committed, equal its first committed content (rewrite, or delete and re-add with other content, fails), and, unless it declares `bootstrapException`, exist at the record's `baseCommit` (seeded before BUILD). Missing, malformed or wrong-kind seeds fail. This tranche's seed declares the reviewer bootstrap exception, reported as a note. |
| F5 `--skip-catalog` printed PASS | The catalog gate reports `[SKIP] catalog: not executed in this process; the Windows CI job … must run it`, is not counted, and the summary reads `6/6 executed gates passed; 1 skipped`. A real failure in the other gates still fails. The Windows `catalog` job stays in `governance.yml` unchanged. |

**Trust boundary (stated, not claimed away):** the gate proves that the worker's record matches a seed that has not changed since it was first committed in this repository's history and, for later tranches, that the seed predates the pin. It cannot prove who authored the seed (one git identity is used), cannot survive history rewriting or force-push, and does not authenticate the dispatcher. The reviewer must confirm the seed's author and timing; each later tranche needs its seed recorded before BUILD.

Also changed: the tranche record (status `REVIEW_PENDING`, history extended after `CHANGES_REQUIRED`), the work order `Status:` line, state, handoff header (now the state's exact next-move text), `AGENTS.md` wording, and continuity notes. The authority seed and `.cvf` authority files were not edited.

## Tests (57 total: 46 gate tests + 11 R019 tests; 4 gate tests skip without PyYAML)

New fixtures, each with a passing baseline first: same-ID contradictory next move; cosmetic markdown normalization passes; stale, unknown and missing work-order status; worker widening `allowedPaths`, dropping a prohibited effect, changing a role or the risk; narrowing allowed and scope following the seed even when the record is narrower; missing, malformed and wrong-kind seed; real-git fixtures for committed paths with spaces/semicolons/parentheses (and newline on POSIX), deleted out-of-scope files, unknown/invalid `--base`, `--head`, an option-like ref, unknown/non-hex/null/non-ancestor pins, an unrelated-history range where `git diff` fails, a non-git tree, seed rewrite after introduction, seed deletion, a seed added after the pin with and without `bootstrapException`, an uncommitted seed; SKIP semantics (shown as SKIP, not counted, never hides a failure, not applied unless requested).

Before the repairs the reviewer's probes passed the old code; the old fixtures now also exercise the new rules (for example the R2 self-review test changes record and seed together so the independence rule is still tested in isolation).

**Mutations of the repaired gate** (applied, tests run, restored; restored file passes): whitespace path splitting (1 failed), PR base not validated (1), pin not validated (1), ancestor check off (1), Git errors swallowed (1), next-move text unbound (1), work-order status unchecked (1), seed widening unchecked (1), seed rewrite unchecked (1), scope taken from the record instead of the seed (1), skip printed as pass (2), seed timing unchecked (1). Two mutations initially survived (Git errors swallowed; scope from record) and drove two additional tests; all 12 are now detected.

## Gates

| Check | Result |
|---|---|
| `python -B -W error -m unittest discover -s scripts/tests` | 57 tests OK |
| Same with PyYAML masked | gate tests 46 run, 4 skipped, OK (keeps the backend workflow's discover green) |
| `python scripts/cvf_downstream_gate.py preflight` (local worktree) | 7/7 executed gates PASS |
| `preflight --skip-catalog --base HEAD~60 --head HEAD` | 6/6 executed gates PASS; catalog SKIP |
| Before the handoff migration the new next-move rule failed on the live files (`handoff Next allowed move differs from state.nextAllowedMove`) and passes after the header was aligned | observed |
| Catalog `-Check`, workspace doctor 25/25, `git diff --check`, `npm run docs:build` | PASS |
| Backend suite | not rerun; no backend or backend-workflow change (R019 inherited) |

## Claim limits

Unchanged from the BUILD evidence: no GitHub-hosted run of the new workflows, regex-bound claim/secret detectors, no local git hook, and core checkers not executed. The Windows `catalog` job result and the F08 run remain unknown until the authorized PR. Passing these gates proves repository state only, not runtime AI governance.

## Changed set

`scripts/cvf_downstream_gate.py`, `scripts/tests/test_cvf_downstream_gate.py`, `CVF_SESSION/tranches/CCMAI-GOV-001.json`, `docs/work_orders/CCMAI_GOV_001.md` (Status line only), `AGENTS.md`, `CVF_SESSION/ACTIVE_SESSION_STATE.json`, the active handoff, `CVF_SESSION_MEMORY.md`, and this evidence.

## Disposition

`REVIEW_PENDING` for Codex re-review; the PR and F08 Actions evidence wait for `REVIEW_PASS`. Claude does not self-approve; no push, merge, deployment or freeze.
