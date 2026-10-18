# R119 source implementation worker return

Status: COMPLETE_PENDING_REVIEW
Risk: R2
Worker: /root/r119_repair_luna (delegated REPAIR_WORKER)
Reviewer: Codex /root (independent)
Scope: FOUR_SOURCE_FILES_ONLY; no runtime admission

## Implemented

Finished the four source files authorized by `CCMAI_RUNTIME_119_CURRENT_UI_PROOF.md`:

- `scripts/r119_ui_supervisor.py`
- `scripts/r119_vite_config.mjs.template`
- `scripts/tests/test_r119_ui_supervisor.py`
- `backend/api/handlers/intervention_ui_r119_test.go`

The supervisor keeps the existing committed-admission contract and adds terminal source-pin and resource-admission rechecks, raw per-command Docker stdout/stderr, exact owned-container identity checks that tolerate an unassigned prestart network endpoint, independent Vite/backend route and two-auth-attempt accounting, safe current-detail response projection, final case/proposal/event fingerprint and provider-related count comparison, and child PID/creation-identity, wait, listener-close, fixture-stop, and protected-inventory receipts. The Go fixture validates its exact stop-file path before opening a listener and emits only the bounded safe detail projection; its UI badge remains driven by the existing `r101-direct-seed` value.

No product source, old runner, seed, authority, continuity, binary, provider, Docker, browser, or runtime was changed or invoked. The second check found no source admission, resource admission, setup receipt, or active runtime. This return therefore admits no fixture start or browser observation.

## Worker checks

Both authorized offline Python invocations were used; total 2/2. Check commands, raw byte files, byte counts, SHA-256 hashes, and exit codes are recorded in `docs/reviews/probes/r119_worker/checks.json`.

1. `python -B -m unittest scripts.tests.test_r119_ui_supervisor` — exit 0; 8 tests passed. This covers loopback port validation, method/auth caps, independently enforced Vite and Go caps, route/status matching, detail capture requirements, final effect-count equality, exact raw Docker output preservation with a subprocess double, and owned cleanup when the fixture has no published port.
2. `python -B scripts/r119_ui_supervisor.py inspect` — exit 0; read-only output recorded. All admission/runtime/setup/terminal presence fields were false.

Raw output files:

- `docs/reviews/probes/r119_worker/python_001.stdout.bin` — 0 bytes; SHA-256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.
- `docs/reviews/probes/r119_worker/python_001.stderr.bin` — 111 bytes; SHA-256 `a079559ead7a3e34ae5c67ded93f4f4a869c944303a5e4cbfcaa4f90a1d2d0ae`.
- `docs/reviews/probes/r119_worker/python_002.stdout.bin` — 180 bytes; SHA-256 `2a9e44724ee56cefb73095537f37d847623bdcf738d1a0c9d4da2a667a72cf28`.
- `docs/reviews/probes/r119_worker/python_002.stderr.bin` — 0 bytes; SHA-256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.

## Exact source hashes at worker return

These hashes cover the uncommitted source returned for independent review. Root must verify them again after the authorized source commit.

| Path | SHA-256 |
| --- | --- |
| `scripts/r119_ui_supervisor.py` | `f8f085a798abf5142a391103fbbaa3c12a49154674235699380e535a5b3695ad` |
| `scripts/r119_vite_config.mjs.template` | `7147fd7dea0ab53325459978ed980485b7787620e7af8d672f706a9965769237` |
| `scripts/tests/test_r119_ui_supervisor.py` | `8405dd12da0a03f6ea11f6994f2bade8d9e7a6730222555c25b3af9dc0355163` |
| `backend/api/handlers/intervention_ui_r119_test.go` | `8225b08530951ad7ba11978cf2a5ebdaa1764518d106b8d74c0d8fbad30db687` |

## Root source-binary admission recipe

After independent source review and the authorized source commit, Root owns the offline Go test-binary compilation and the committed `docs/reviews/probes/r119_root/source_binary_admission.json`. From the project root, compile without running tests using `go -C backend test -c -o ..\.cvf\runtime\r119_build\intervention-ui-r119.test.exe ./api/handlers`. Pin the source commit as an existing ancestor and SHA-256 every path required by the supervisor: every `backend/**/*.go` file; `backend/go.mod` and `backend/go.sum`; the four files above; all files under `frontend/src` and `frontend/public` when present; and `frontend/package.json`, `frontend/package-lock.json`, `frontend/vite.config.ts`, `frontend/index.html`, `frontend/tsconfig.json`, `frontend/tsconfig.app.json`, and `frontend/tsconfig.node.json`.

The admission object must include:

```json
{
  "trancheId": "CCMAI-RUNTIME-119",
  "image": "sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b",
  "sourceCommit": "<40-hex committed source revision that is an ancestor of HEAD>",
  "sourceSha256": { "<project-relative POSIX path>": "<64-hex sha256>", "...": "..." },
  "binaryPath": ".cvf/runtime/r119_build/<Root-selected-test-binary>",
  "binarySha256": "<64-hex sha256 of the compiled R119 test binary>",
  "nodePath": "<absolute Node executable path>",
  "nodeSha256": "<64-hex sha256>",
  "dockerPath": "<absolute Docker executable path>",
  "dockerSha256": "<64-hex sha256>"
}
```

Root must commit this exact admission before setup. Setup then writes the exact owned-resource receipt; Root commits `resource_admission.json` before `serve` can start. The supervisor rechecks source and resource admissions at terminal cleanup. Neither admission is present at this worker return.

## Disposition

Source return is complete pending Root's independent review and offline compile. Worker Python budget is exhausted at 2/2. No runtime, provider, secret, browser, or governance behavior was tested; no FREEZE or product acceptance claim is made.
