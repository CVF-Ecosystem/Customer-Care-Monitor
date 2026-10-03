# R039 — Retire historical Git-baseline assertion from offline CLI tests

Date: 2026-10-03. Status: SPEC_READY; BUILT, REVIEW_PENDING. Risk R1: test-only maintenance; runtime/admission/receipt/governance gates unchanged. Source planning anchor ae620ba; dispatcher seed/baseCommit `7a390dc08e7958015b107e3a3e3b890369b82cf1`. [Order](../work_orders/CCMAI_RUNTIME_039.md).

## Intake and design

`TestProductionAdapterFilesUnchangedInGit` in `backend/cmd/pancake-proof/main_test.go` compares a live checkout (including all backend/engine) against historical R034 seed d869624cc15f55b39516a36f8937454e617dc3b3. Its Git availability/baseline checks can SKIP in archives/shallow checkouts; on the current full checkout the actual protected-path comparison exits1 solely because accepted R038 deleted the uncalled helper in analyzer_incremental.go. This is direct Git/source evidence; the named Go test has NOT RUN during planning. R038 promised compile-only evidence and zero executed engine tests, not a passing CLI regression suite.

The original R034 requirement concerned unchanged production source **within that BUILD**, not a permanent ban on independently authorized later changes. Remove only the entire named test and immediately attached PH-07 comment. Preserve imports (all remain used), the six other top-level CLI tests, their helpers/fixtures/assertions and every product/harness source line. Do not change the baseline hash, prune its path list, add skips or weaken CLI safety tests to get green. Source-scope verification remains in immutable tranche allowlists, Git comparison and independent review; no repository gate or R034 authority change.

| ID | Required evidence |
| --- | --- |
| CS-01 | Before-edit cached named test in full Git checkout fails specifically at the historical protected-path assertion; capture exit/failing message. Historical Git diff names only backend/engine/analyzer_incremental.go. Missing Git/baseline or SKIP is BUILD_BLOCKED. After: exact file equality after deleting only named test/comment; six behavioral tests remain byte-identical. |
| CS-02 | Complete offline CLI suite uncached passes, no skipped tests; actual mounted main entrypoint remains exercised. Preserve PASS/non-PASS/invalid-input/sanitization/poisoned-environment/finite-fixture checks. Cached go build/vet all packages pass; no engine test execution. |
| CS-03 | Compare dispatcher baseCommit to BUILD/worktree for backend/channels, backend/engine, all non-target CLI source, go.mod/go.sum, frontend, scripts and workflows: empty diff. Only authorized test deletion plus bounded docs/continuity. Gate/docs/catalog/doctor/diff pass. Do not confuse the expected old baseline difference with a new product change. |

No new mirrored unit test or mutation campaign needed for deletion of a procedural Git assertion. The original failing test is a discriminating control; protect the remaining behavioral coverage by exact byte comparison and full suite execution. Require no skipped CLI tests; report actual top-level/subtest counts without calling compile-only engine evidence regression PASS.

## Evidence boundary

Allowed runtime here is only finite synthetic in-memory CLI tests; receipt remains SYNTHETIC_OFFLINE. No actual channel/provider request or runtime AI-governance claim. No adapter/harness/engine product edit, engine test/Analyzer execution, database/Docker, customer data, configuration/credential reads, external network, downloads, dependency/workflow/gate/parent change, push/merge/deployment or new FREEZE. Existing tests may create/read only their task-owned synthetic poison fixtures. Prior source acceptances and failures remain attributed. Live packet external inputs and real MCP execution remain separate.

*Historical (dispatch-time): DISPATCH_READY / NOT_BUILT, original test present, no worker started or Go commands executed in planning.* **Current:** Claude BUILD preserved the named original-test failure (exit 1 at the historical protected-path assertion, not a skip), removed only that test and its PH-07 comment (19 lines), and ran the full offline CLI suite uncached (6 tests PASS, 0 skips) plus cached build/vet; protected-path diff from the seed is empty (evidence: [BUILD record](../reviews/PANCAKE_CLI_SCOPE_ASSERTION_R039_BUILD_2026-10-03.md)). REVIEW_PENDING, not independently accepted. R038 REVIEW_PASS / FREEZE_OPEN and prior local acceptance/R033 FREEZE remain unchanged. Independent Codex review required after Claude BUILD.
