# R042 BUILD record — offline proof CLI usage guide and sample

Status: BUILT, REVIEW_PENDING (not accepted). Worker: Claude IMPLEMENTATION_WORKER (owner-transferred, R1 documentation-only). Reviewer: Codex, independent. [Order](../work_orders/CCMAI_RUNTIME_042.md), [SPEC](../specs/PANCAKE_OFFLINE_PROOF_USAGE_R042_2026-10-03.md). Claim scope: operator documentation and finite synthetic smoke of the accepted offline CLI only; not live inventory, universal path sandbox, symlink/race/mapped-drive closure, provider or CVF AI governance, hosted readiness or FREEZE.

## 1. Baseline, seed and identities

- Seed/baseCommit `9b10a8f6f67d314d6c4f5a3846396f28a5487948` (`CVF_SESSION/authority/CCMAI-RUNTIME-042.json`, unchanged, not in the changed set); HEAD at BUILD start `a55b477` (dispatch). Owner manual transfer: the owner message "Chuyển Claude: CCMAI_RUNTIME_042.md". Rehydration, declaration and WORK_ORDER -> BUILD acknowledgment were recorded in the active handoff and BUILD continuity passed the default preflight (7/7) before any test or edit.
- Accepted CLI source `2a44a8685adfdc3582697ce5094d06da3047207b` is unchanged: `git diff --name-only 9b10a8f -- backend frontend scripts .github/workflows CVF_SESSION/authority` is empty; against `2a44a86` the only difference in those paths is the R042 seed file itself, added by the dispatcher at `9b10a8f`. Working-tree hashes: `main.go` `fd7dcb9e0230056fb605fd0315a18cfa31b84292693a317aeba7a166c4c2084f`, `main_test.go` `bde3d475a21bbfe2a55ffbc96a83e71e34a6e7c3464dab1f7c1344326f316b2f`, `inventory.go` `6a403d31dd6f4e52d3aa14f78b989efb5b3226f4b772b54853fa5fd43ef8b8c6`, `inventory_test.go` `81d357f2596333fa15af549af34e6f0027b174b8def6169ab4d8289fc4cc208a`.
- Created documents: `docs/guide/pancake-offline-proof.md` (13692 bytes, SHA-256 `c9abc03ed797eb39b3a8cc90607e58f0a9e0954989ce4fff049ce5c23de1e7ad`), `docs/examples/pancake-proof/synthetic-inventory.json` (1075 bytes, SHA-256 `ef7e10bbb4ad96cf131a7b5a071c2e47fa2326fe2c1c37fdede34ba494b8ac27`, UTF-8 without BOM). Edited: only an appended and labeled section of `docs/reviews/F02_PANCAKE_LIVE_PROOF_PACKET_2026-10-03.md` (16 insertions, 0 deletions). Plus this record and the governed continuity/status/catalog/index files.
- Smoke executable (task-temp, never in the repository): `go -C backend build -o <task-temp>\pancake-proof.exe ./cmd/pancake-proof` from the repository root, exit 0, SHA-256 `3d52fc5e17394d9aecfc3a427e327e93e241d1776784b2b8adbd3ad168b8513c`. `GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local CGO_ENABLED=0`, go1.27.0 windows/amd64, no download, no compiler/privilege change. `git status` showed no executable in the repository.
- The receipt `source_sha` used in the smoke is `2a44a8685adfdc3582697ce5094d06da3047207b`, a value supplied on the command line as metadata (the CLI does not verify it); the actual compiled source identity is the hash set above.

## 2. DU matrix

| ID | Result | Evidence |
| --- | --- | --- |
| DU-01 | PASS | Guide is Vietnamese; PowerShell commands run from the repository root with quoted paths; documents the cached-Go prerequisites, the `-o` task-temp build (and warns that a bare `go build` leaves `backend\pancake-proof.exe`), every flag with default and range (`-source-sha`, `-key`, `-scenario`, `-max-attempts` 1..50 default 20, `-max-duration` default 1m up to 10m, `-inventory`), no live/config/token option, `-inventory` only with the pass scenario, duplicate `-inventory` rejected. The six guide PowerShell blocks were extracted from the file and executed as written (section 3.2) with the temp root redirected to a task-owned directory. Links to the R041 SPEC, BUILD and review and to the live packet. |
| DU-02 | PASS | Sample is hand-authored, exact schema/evidence/page, `synthetic-docs-v1`, two conversations and three messages with instants, senders, content types and the one image attachment; no real ID, text, token, capture, config or path. The guide documents required nesting, types, enums, safe/unique IDs, offset rule, unknown/duplicate/null/trailing rejection, and limits (1 MiB, 100 conversations, 1000 messages, 16 attachments, depth 64, IDs 128 bytes, names 256 bytes). The JSON block embedded in the guide equals the checked-in sample (parsed and text equality asserted by script). |
| DU-03 | PASS | Section 3.1. |
| DU-04 | PASS | Guide explains exits 0/1/2, synthetic-only meaning, no echo of path/IDs/names/key, `source-sha` as caller-supplied metadata with `git rev-parse HEAD` for recording the real revision, the key as a demonstration pseudonym key, both runs sharing the budget. The no-leak property was checked on three smoke outputs (section 3.1). |
| DU-05 | PASS | Guide and live packet keep symlink UNVERIFIED (two actual SKIP identities named in the R041 review/BUILD: `TestInventoryLinkAdmission/symlink-final-file` and `.../symlink-ancestor-directory`), race NOT RUN (no C compiler), mapped/subst drives not classified, TOCTOU narrowed not eliminated, linked/reparse profile ancestors refused, "not an arbitrary-path sandbox". Inputs used are task-owned local files in an unlinked location. Future live requirements (independent inventory, credentials/network authority, quiescence, capture, provenance, transport) retained. Nothing was attempted against real shares, tools, privileges or guards. |
| DU-06 | PASS | The packet's post-R034 section ("Post-R1 live integration assessment") is labeled a historical snapshot directly under its heading with a link to the new section; its table (including the no-loader row) and every other line are byte-preserved (diff: 16 insertions, 0 deletions). The appended "Current post-R041 assessment" separates the completed synthetic loader from the missing live loader, provenance, credential, transport, quiescence and capture requirements and repeats the platform limits. No live work dispatched, no acceptance altered. |
| DU-07 | PASS | Section 4. |

## 3. Smoke evidence

### 3.1 DU-03 runs (finite, one process each, 60 s ceiling per process, never reached; stdout, stderr and exit recorded independently)

Task-temp root is an unlinked local directory under the session scratchpad. Inputs: `inventory.json` (copy of the sample), `inventory-empty.json` (sample with `conversations` emptied, UTF-8 without BOM), `inventory-duplicate.json` (sample with `provenance` repeated), `does-not-exist.json`. Key `demo-pseudonym-key-0001`.

| Run | Arguments (besides `-source-sha`/`-key`) | Exit | Observation |
| --- | --- | --- | --- |
| a | none (no-inventory control) | 0 | PASS, SYNTHETIC_OFFLINE, live=false, governance_claim=false, provenance `synthetic-cli-fixture-v1`, 12 attempts, 2 conversations, 3 messages; stdout 7606 bytes, stderr 124 bytes (banner) |
| b | `-inventory` sample copy | 0 | PASS, SYNTHETIC_OFFLINE, live=false, governance_claim=false, provenance `synthetic-docs-v1`, 12 attempts, 2/3; digest prefix `6735e3cc6d2b` |
| c | same as b (second run) | 0 | identical to b after normalizing only `started_at`, `finished_at`, `conversation_until`, `elapsed_ms`; stderr identical |
| a vs b | | | the only differences in the normalized receipts are the `digest` and `provenance` lines |
| d | `-inventory` expected-empty copy | 1 | FAIL, reasons `run1:reconciliation_mismatch` and `run2:reconciliation_mismatch`, 4 attempts, 0 conversations/messages expected |
| e | `-scenario empty-inventory` (legacy, no `-inventory`) | 1 | INCOMPLETE, reasons `run1:empty_inventory` and `run2:previous_run_incomplete`, 1 attempt |
| f | `-scenario redirect -inventory <nonexistent>` | 2 | stdout empty; stderr `pancake-proof: -inventory is supported only with the pass scenario` (the scenario is rejected before the file is opened: the path does not exist yet the message is the scenario conflict) |
| g | `-inventory` duplicate-key copy | 2 | stdout empty; stderr `pancake-proof: inventory rejected` |
| h | `-inventory` given twice | 2 | stdout empty; fixed usage message |
| i | `-inventory` nonexistent file | 2 | stdout empty; stderr `pancake-proof: inventory rejected` |

No-leak check over the stdout and stderr of runs b, d and g: none of `syn-conv-a`, `syn-msg-a1`, `photo-001`, the key, `synthetic-offline-token`, `synthetic-page-001`, `inventory.json` or the task-temp directory name appears. The safe provenance label appears by design. Every exit code was captured immediately per process (a PowerShell script using separate process objects), so no expected nonzero exit was masked by a later command.

### 3.2 Guide blocks executed as written

The six PowerShell blocks of the guide (setup/build, sample copy, `$sha`/`$key`, default run, inventory run, `empty-inventory` run) were extracted from `docs/guide/pancake-offline-proof.md` and executed in order with `TEMP`/`TMP` pointed at a task-owned directory so the guide's `GetTempPath()` landed there: build succeeded, default run exit 0 (PASS, 12 attempts, provenance `synthetic-cli-fixture-v1`), inventory run exit 0 (PASS, 12 attempts, provenance `synthetic-docs-v1`), `empty-inventory` run exit 1 (INCOMPLETE, 1 attempt). The `$sha` there is the current `git rev-parse HEAD`, again supplied metadata. No executable appeared in the repository.

## 4. Checks (DU-07)

| Check | Result |
| --- | --- |
| Workspace doctor | 25/25 PASS before BUILD |
| Default preflight after BUILD sync, before any test or edit | 7/7 PASS |
| `git diff --name-only 9b10a8f -- backend frontend scripts .github/workflows CVF_SESSION/authority` | empty |
| Guide embedded JSON vs checked-in sample | equal |
| `npm --prefix docs run docs:build` | **FAIL (pre-existing, out of scope)**: `build error: [vitepress] 1 dead link(s) found`; the single dead link is `./../../../CVF_SESSION/handoffs/AGENT_HANDOFF_OFFLINE_PROOF_USAGE_2026-10-03` in `docs/reviews/learnings/feedback_cvf_repair_workflow.md` line 85, a file introduced by the dispatch commit `a55b477` and outside the R042 allowed paths (the site excludes `specs/` and `work_orders/` but not `reviews/learnings/`, and a link leaving `docs/` cannot resolve). No dead link is reported for the new guide, the live-packet section or this record, and no R042 file introduces one. The file was not edited (scope); the fix (turn that markdown link into a code span or add it to `ignoreDeadLinks`) is returned to the ORCHESTRATOR/reviewer. Earlier builds in this session passed before that dispatch commit. |
| catalog -Write/-Check, doctor, `git diff --check` | PASS (an EOF blank line introduced in the packet append was corrected before the final check) |
| default and PR-range preflights, gate unit tests | recorded in the active handoff (PASS) |

## 5. Inherited versus new evidence, limits and NOT RUN

- Inherited and not re-run: R041-R1 independent acceptance of the CLI source (suite counts, mutations, original-main control), including the two real SKIP symlink subtests and race NOT RUN. New here: the documentation, the sample and the finite smoke above. No Go test suite, mutation, new test or helper source was run or added.
- Preserved limitations: symlink rejection UNVERIFIED, race NOT RUN (no C compiler), mapped/subst drives unclassified, TOCTOU not eliminated, linked/reparse profile ancestors refused, no arbitrary-path sandbox claim.
- The smoke is synthetic and offline: it is neither live completeness nor CVF governance proof; `source_sha` in a receipt is not an independent identity proof.
- Incidents: the docs-build dead link above (pre-existing, unrepaired, scope-bounded); a trailing blank line added by my packet append, caught by `git diff --check` and fixed. The first guide-run script printed receipts in full, so a filter was added before the recorded run (no effect on results). `git status` listed only intended files at every check.
- NOT RUN: real inventory/capture/customer data, `.env`/config/credential reads, provider/channel/network, shares, downloads, DB/Docker, engine or full backend suites, Analyzer, race, GitHub Actions, push, merge, deployment. No FREEZE claim.

## 6. BUILD identity and hand-back

Exact BUILD commit: `6b401195edcc99e9bc9568c45c7ff8962c7d7480` (seed `9b10a8f6f67d314d6c4f5a3846396f28a5487948` unchanged). It is recorded in the tranche record `buildCommit` and the active handoff by a follow-up documentation commit that changes no guide, sample or source (a commit cannot contain its own SHA). Independent Codex REVIEW is next; no self-approval, push, merge, deployment or FREEZE.
