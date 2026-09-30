# CCMAI-RUNTIME-019 / F08 — independent R019-R1 re-review

Date: 2026-09-30. Reviewer: Codex, independent of Claude REPAIR_WORKER. Reviewed repair commit: `4f3a5c35760bfd900a1bea748d68912e515270db` (parent `eb30bfc0ea34134a52b4f7c0c4ac4de468edb16e`). Disposition: **REVIEW_PASS / FREEZE_OPEN** for the local CI gate and R019-R1 repair. F08's public CI outcome remains open until an authorized GitHub Actions run; no push or FREEZE here.

## Scope and source review

The exact repair diff changes `scripts/ci_db_test_gate.py`, `scripts/tests/test_ci_db_test_gate.py`, BUILD evidence and session continuity only. It does not change the workflow, product source, DB assertions or Compose. The parser now adds `bad_lines` to gate problems and recognizes `DB not available` in skip output. That phrase matches `backend/engine/integration_test.go:93`. The regression tests cover a stray non-JSON line, a damaged trailing JSON record, and that English DB skip after all five sentinels pass. The existing optional S3 skip remains allowed. The file-handle ResourceWarnings noted in the first review were removed. No raw skip output or DSN is printed in the gate report.

## Independent checks

- `python -B -W error -m unittest discover -s scripts/tests -v`: 11/11 PASS, no warning.
- Direct `gate.evaluate` probes against the committed parser: complete synthetic five-sentinel log PASS; same log plus stray non-JSON line FAIL; plus truncated JSON tail FAIL; plus `TestIntegrationFullJobFlow` skip output containing the source phrase `DB not available` FAIL; plus optional S3 skip PASS. All five outcomes matched the acceptance contract.
- Source inspection confirms all five package-qualified sentinels still exist; the workflow still runs the full backend suite against disposable MySQL, checks application-user readiness, evaluates the parser and then builds. `git diff eb30bfc 4f3a5c3 --name-only` confirms no workflow or product change in the repair.
- The BUILD record says Claude replayed the original full-suite disposable-MySQL Go JSON log through the repaired parser: 234,247 events, zero invalid records, 454 test passes, five sentinel passes, two optional skips, no DB skip, gate exit 0. The original log is not retained in the workspace, so Codex did not independently replay that positive log or rerun MySQL in this re-review. The parser repair only adds two rejection conditions; the reported zero-invalid/no-DB-skip positive log would remain accepted. The BUILD record also reports both no-DB logs still exit 1.
- Workspace doctor passed 25/25 before source review. Catalog and diff checks are recorded at review commit closure.

## Claim boundary and next move

The two blocking findings in the first independent review are resolved in local source and tests. This review accepts the R019-R1 repair and local F08 gate behavior. It does not establish a successful GitHub-hosted workflow, never-ready MySQL behavior on that runner, secret masking, port binding or `GITHUB_ENV` handling there. Those require an actual authorized workflow run. Keep F08 **OPEN for public CI verification / FREEZE_OPEN**; F01–F07 remain separate open findings. No provider API call was needed or made because this review makes no CVF runtime governance claim. No push, deployment, persistent DB change or FREEZE.
