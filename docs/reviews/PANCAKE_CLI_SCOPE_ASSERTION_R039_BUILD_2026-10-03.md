# R039 BUILD record — retire historical Git-baseline assertion from the offline CLI tests

Status: BUILT, REVIEW_PENDING (not accepted). Worker: Claude IMPLEMENTATION_WORKER. Risk R1, test-only. [Order](../work_orders/CCMAI_RUNTIME_039.md), [SPEC](../specs/PANCAKE_CLI_SCOPE_ASSERTION_R039_2026-10-03.md).

## 1. Baseline, seed and hashes

- Seed/baseCommit `7a390dc08e7958015b107e3a3e3b890369b82cf1` (`CVF_SESSION/authority/CCMAI-RUNTIME-039.json`, unchanged, not in the changed set); HEAD at BUILD start `6e5db68` (dispatch).
- Role acknowledgment/declaration and the WORK_ORDER -> BUILD transition were recorded in the active handoff, BUILD continuity was synchronized and passed the default preflight (7/7) **before** the control run and before the edit.
- `backend/cmd/pancake-proof/main_test.go` SHA-256 (LF file): before `5c5fe14b2cde9763fd71c3bb48a01ce6e7a1ea775ce8caa49206efaec87392e5`, after `bde3d475a21bbfe2a55ffbc96a83e71e34a6e7c3464dab1f7c1344326f316b2f`.

## 2. CS-01 — original control and exact deletion

**Control (before the edit)**, project-root cwd, `GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local CGO_ENABLED=0`:

`go -C backend test ./cmd/pancake-proof -run '^TestProductionAdapterFilesUnchangedInGit$' -count=1 -v` -> **exit 1 (preserved, expected)**:

```
=== RUN   TestProductionAdapterFilesUnchangedInGit
    main_test.go:225: a protected production file differs from the tranche baseline: exit status 1
--- FAIL: TestProductionAdapterFilesUnchangedInGit (0.14s)
FAIL
```

Line 225 is the historical `git diff --exit-code --quiet d869624c… -- <protected paths>` assertion; it is neither a compile/prerequisite failure nor a SKIP (Git and the baseline commit were present). `git diff --name-only d869624cc15f55b39516a36f8937454e617dc3b3 -- backend/channels/{pancake,adapter,registry,facebook,zalo_oa}.go backend/engine backend/go.mod backend/go.sum` names exactly one path: `backend/engine/analyzer_incremental.go` (the accepted R038 helper removal). Baseline therefore matched the expected state.

**Deletion**: exactly the PH-07 comment line, the whole `TestProductionAdapterFilesUnchangedInGit` function and the one blank line before the comment: 19 lines removed, 0 added (`git diff --stat`: 1 file, 19 deletions). Performed by a script asserting exactly one `// PH-07:` marker, one function and an `}\n` file ending; file is LF and stayed so. No baseline retarget, skip, path pruning or weakening.

Preserved test bodies (SHA-256 prefix of each `func … }` body, before = after):

| Test | Body digest | Result |
| --- | --- | --- |
| TestMain | c6450d2da6f1b135 | identical |
| TestCLIPassReceiptIsLabelledSynthetic | 5a5cf5c4720195b1 | identical |
| TestCLINonPassExitCodes | b0d6fabecdf6c6dc | identical |
| TestCLIInvalidInputRejectedWithoutReceipt | 9b42c92a9b899163 | identical |
| TestCLIPoisonedCredentialAndConfigEnvironmentIsIgnored | 48d885cb5c8e4751 | identical |
| TestCLIEntrypointAndRunAgree | e6d3753056ad16a5 | identical |
| TestCLIFixtureIsFiniteAndIndependent | baf95551bc6cc6fd | identical |
| TestProductionAdapterFilesUnchangedInGit | a233eaecf12e0786 | removed |

Six top-level behavior tests plus `TestMain` are byte-identical; imports unchanged and all still compile (`os/exec` and `strings` remain used by other code).

## 3. CS-02 — full offline suite and builds

| Command | Result |
| --- | --- |
| `go -C backend test ./cmd/pancake-proof -count=1 -v` (uncached) | exit 0, `ok … 5.108s`; **6 top-level tests PASS, 0 subtests, 0 FAIL, 0 SKIP** (mounted `main` entrypoint exercised by TestCLIEntrypointAndRunAgree/TestMain) |
| `go -C backend build ./...` | exit 0 |
| `go -C backend vet ./...` | exit 0 |

Only the CLI package's own finite synthetic in-memory/temp-file tests executed (receipts remain SYNTHETIC_OFFLINE). The engine package and its tests were **not** run.

## 4. CS-03 — protected-path comparison

`git diff --name-only 7a390dc -- backend/channels backend/engine backend/cmd/pancake-proof/main.go backend/go.mod backend/go.sum frontend scripts .github/workflows` -> **empty (0 paths)**. `git diff --name-only 7a390dc -- backend` -> only `backend/cmd/pancake-proof/main_test.go`. Outside governed documentation/continuity nothing else changed. The expected difference from the old R034 baseline (`analyzer_incremental.go`) is the accepted R038 change and is not a new product change.

## 5. Other checks

Workspace doctor 25/25 PASS before BUILD; default preflight 7/7 PASS after BUILD sync (before control and edit). Docs build, catalog `-Write`/`-Check`, diff check, PR-range/changed-set preflights and gate unit tests are recorded in the active handoff.

## 6. Failures and NOT RUN

- The control test failure above is the only failing command and is the expected, preserved original behavior.
- No other failed attempt in this BUILD.
- NOT RUN: engine package tests, Analyzer, race detector, DB/Docker, sockets beyond the CLI tests' own, real channel/provider requests, credential/config discovery, network, downloads, frontend, GitHub Actions, push, merge, deployment. No runtime AI-governance, live, hosted-readiness or FREEZE claim.

## 7. BUILD identity and hand-back

The exact BUILD commit SHA is recorded in the tranche record `buildCommit` and the active handoff by a follow-up documentation commit that changes no source (a commit cannot contain its own SHA). Independent Codex REVIEW is next; no self-approval, push, merge, deployment or FREEZE.
