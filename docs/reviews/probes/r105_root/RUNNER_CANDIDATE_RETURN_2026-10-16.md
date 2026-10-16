# R105 Root source candidate return — 2026-10-16

Status: COMPLETE_PENDING_REVIEW
Risk: R2
Worker: /root/r105_runner_candidate (Sol Medium)
Reviewer: Codex /root
Role: SOURCE_CANDIDATE_AUTHOR for Root; Root owns independent R2 review, effective authority, and any future dispatch. Runtime NOT_RUN; proposed R105 remains DRAFT/NOT_DISPATCHABLE. No commit or FREEZE.

## Source repair round 1 — current result

Root's initial review correctly found that the first draft was only a scaffold with unconditional runtime HOLD. The first-draft hashes and checks below are retained as history, not current source identities. The repaired candidate now implements strict prospective admission and a directly supervised `start` campaign, read-only `status`, and cooperative `stop` request. Current `start` still fails before effects because the effective R105 formal order/seed/record/review receipt do not exist.

Current source SHA256:

| Path | SHA256 |
| --- | --- |
| `r105_ui_runner.py` | `6bf35415bfa564da19abde61dcd1a187d40ced710d94c2398875c1df86409bde` |
| `test_r105_ui_runner.py` | `1cebf39ddfbe75c20c72753441bd75ad61697257751977b21fcf5d9429b9ca6d` |
| `r105_vite_config.mjs.template` | `86aa4af32624314323e5e6b3b42046f60e564ac3878f54a17cf214fbdfc8050b` |

`check_admission` expects the following prospective formal schema, for Root to create only after independent source review: exact order `docs/work_orders/CCMAI_RUNTIME_105_R101_UI_EVIDENCE.md` with one `Status: ACTIVE`; authority `CVF_SESSION/authority/CCMAI-RUNTIME-105.json` with `authorityKind: DISPATCHER_OWNED_HISTORICAL_R101_UI_EVIDENCE`, `trancheId`, `riskCeiling: R2`, roles, allowed paths, prohibited effects, `historicalTargetCommit`, `binarySha256`, `executionBudget`, `nodeExe`, `nodeSha256`, `dockerExe`, `dockerSha256`, and `runtimePins`; matching record `CVF_SESSION/tranches/CCMAI-RUNTIME-105.json` in BUILD with baseCommit; current state BUILD/active105. `runtimePins` is an exact map of project-relative paths to captured workspace byte SHA256 for runner, template, frontend manifest, historical manifest and `docs/reviews/probes/r105_root/source_review_receipt.json`. The receipt must be Root-authored, committed at the record base, with `disposition: SOURCE_REVIEW_PASS_RUNTIME_NOT_RUN`, `reviewer: Codex /root`, `sourceCandidateAuthor: /root/r105_runner_candidate`, and `runtimePins` for the four source/input paths excluding itself. The first committed seed bytes and base version must equal current seed content after explicit UTF-8 CRLF-to-LF normalization; `admission.json` records the resulting UTF8_LF SHA256. Byte pins for the other files remain exact captured workspace bytes and must be rechecked after any checkout.

The exact finite `executionBudget` fields required here are `existingFixtureStarts=1`, `historicalBackendLaunches=1`, `viteLaunches=1`, `runtimeRetries=0`, `backendSeconds=1800`, `wholeCampaignSeconds=2100`, and zero `providerCalls`, `mockCalls`, `goCompiles`, `testBuildCommands`, `newFixtures`. Formal allowed paths must contain the R105 Root, worker-evidence, and runtime families. Node is fixed to `C:/Program Files/nodejs/node.exe` with SHA256 `3602f2bb1a10f2cbab4c36886218a33c1ab3db87290e73b033c46c77147d0237`; Docker executable must be an absolute file pinned by `dockerSha256`. Root must reconcile these proposed field names with the formal gate schema before activation. A draft candidate or a bare approval flag cannot pass.

The supervisor reserves each of the three effect slots using exclusive file creation plus flush/fsync, then retains direct `Popen` handles. It starts only the exact existing fixture, polls the fresh loopback DB port and synthetic schema, launches the pinned historical binary with its new synthetic fixture, parses fresh UUID readiness, renders the no-env config with JSON-escaped literals and a random health nonce, launches cached Vite on strict port 3011, and checks the nonce through direct loopback socket HTTP while the owned Vite handle remains live. It stops by 1740 seconds of backend lifetime or 2040 seconds of whole campaign time, leaving up to 60 seconds for cleanup within each respective authority ceiling. `finally` terminates/waits only direct child handles and stops only the exact owned fixture; protected safe-field inventory is compared before/after. A separate `stop` writes a one-time control file for the living supervisor and never targets a PID. Current frontend/binary input checks apply to `inspect/start`, not `status/stop` after an admitted campaign. Docker `--format` emits only selected safe fields; `Config.Env` is never returned. Command and process stdout/stderr are separate; retained logs are bounded to 1 MiB per stream with full redacted byte counts, digest, truncation metadata and exit receipts. This is source design, not observed runtime behavior.

Actual round-1 offline checks: `python -B -m unittest docs/reviews/probes/r105_root/test_r105_ui_runner.py -v` **7 PASS** (including injected deadline/failure cleanup and CRLF seed representation); `Get-Content -Raw docs/reviews/probes/r105_root/r105_vite_config.mjs.template | node --check --input-type=module` exit 0; `python -B docs/reviews/probes/r105_root/r105_ui_runner.py inspect --project-root .` exit 0 with 137 pinned frontend files and the exact historical binary. No `start`, `status`, or `stop` CLI call, Docker, SQL, backend, Vite, browser, provider, mock, network, compile, install, download, or commit occurred. Root must independently source-review the candidate and test the real lifecycle only after formal authority; runtime safety and product evidence are not accepted by these offline checks.

Remaining source/runtime review limits: Docker CLI `--format` compatibility, live Vite plugin/module resolution, exact process/log receipt behavior, DB readiness and container stop, and integrated product browser HTTP access have not been exercised. A hard process termination can leave the fixture or child process behind; the new attempt ledger prevents replay, and cleanup then needs Root's independent ownership assessment rather than broad PID/tree kill. The candidate does not claim R101 B01-B10 or Dashboard acceptance, current R102 backend acceptance, AI governance, or full M3 completion.

## Initial scaffold history (superseded by round 1)

## Changed set

Only this directory received four new files: `r105_ui_runner.py`, `test_r105_ui_runner.py`, `r105_vite_config.mjs.template`, and this return. The source SHA256 values before this return were:

| Source | SHA256 |
| --- | --- |
| `r105_ui_runner.py` | `6dc951deee22bcb253bda3231e068d17b94623e4c0fc7fba7a4cbe82ac967099` |
| `test_r105_ui_runner.py` | `ca03da788d892bbd2175cf43ebd69fa45e2341f561a4e7d78d6996eca2de9ebe` |
| `r105_vite_config.mjs.template` | `c9c6f6f0e2917b2a3792eec955857911d003413803d2b1731eb44a77f15d4356` |

The Python candidate imports without runtime work. `inspect` verifies all 137 pinned current frontend files and the exact historical binary bytes; historical 17-row Git blob/CRLF, Docker safe-field identity/exclusivity, protected inventory, fresh Go readiness, and irreversible per-effect reservation are isolated validators with injected offline tests. They do not yet form an admitted execution path. The template retains Vue and Vuetify plugins, fixes frontend root, disables env file loading and ambient `VITE_` exposure, and limits proxying to `/api` on a fresh loopback origin. It is a reviewed-source proposal only; no generated config or Vite launch occurred.

## Actual offline checks

- Workspace doctor: PASS 27/27; manifest core pin `e9ddcb...` differs from clean public core `fa34ed...`, reported by doctor as warning.
- `python -B -m unittest docs/reviews/probes/r105_root/test_r105_ui_runner.py -v`: 4 PASS.
- `Get-Content -Raw r105_vite_config.mjs.template | node --check --input-type=module`: exit 0.
- `python -B r105_ui_runner.py inspect --project-root .`: exit 0; 137 current frontend file hashes and binary SHA256 `4c7575e180dd198d995bfe09bdc45061740a987be8ec0765f02b203e1c251f31` checked. This is offline input inspection, not historical binary execution or current backend acceptance.
- No Docker, SQL, backend, Vite, product browser, provider, mock provider, install, download, network effect, or commit was performed. Existing source, historical evidence/counters, continuity, and seed were not edited.

## Consolidated HOLDs for independent review and future implementation

1. **Admission:** R105 is DRAFT and no effective formal seed/order/record exists. The candidate has no `--approved` or force path. All `start`, `status`, and `stop` CLI stages reject even if the named files later appear: committed first seed content at base, active record/continuity/packet, independent review and exact runner/config hashes need an implemented, independently reviewed verifier. A DRAFT file or worker flag cannot admit effects.
2. **Runtime supervisor:** An owned Windows process supervisor with creation-time/executable/command checks, strict 35 minute campaign deadline, 30 minute fixture expiry handling, fresh stdout readiness within 60 seconds, Vite port 3011 exclusivity, exact command and minimal child environment, evidence logging/redaction, and `finally` cleanup is not implemented. A detached `start` would leave Vite unbounded, so no launch is permitted. Cleanup after a state transition or deadline must rely on the original admission and exact ownership receipts, not current BUILD status; this remains to implement.
3. **Fixture lifecycle:** Existing container one-start, post-start one-loopback DB port, bounded SELECT1/schema check, synthetic fixture-only DSN, network/volume protected before/after comparison and safe owned stop are not connected to runtime stages. The source validator never reads Docker `Config.Env`. The fixture test seeds new synthetic tenant/case and may migrate schema; future authority and evidence must count that DB mutation and one backend launch. No third fixture/retry.
4. **Historical binding:** The 17-row validator requires an injected Git blob reader and is covered offline; `inspect` does not call Git or bind the historical row manifest. Root must verify all 17 against commit `200df3e4a7bc2a326425922a5230508a4fd9b0fe` with explicit CRLF mapping immediately before future admission. The old historical source/protection runner remains unsuitable and untouched.
5. **Config isolation:** The template's two placeholders require strict JSON-string substitution, exact generated config hash, reviewed absolute package resolution, cached dependencies and a minimal environment. Node syntax alone does not prove Vite/plugin resolution or absence of transitive ambient reads. No Vite execution or HTTP `/api` observation occurred.
6. **Browser and residual diagnostic tab:** Actual integrated HTTP access, owned product-tab capture/actions, and exact original residual-tab ownership/closure remain Root/worker evidence tasks under a future effective order. R104 data-URL diagnostic is historical provenance only. R101 B01-B10 and three Dashboard failures remain OPEN.

Root should treat the source as a reviewable safety scaffold, not an execution-ready runner. A future bounded implementation/review must close these HOLDs before activation; failures must be retained rather than worked around by altering old baselines, counters, or guards.

## Source repair round 2 — final candidate for independent review

This section supersedes round-1 source hashes and the initial scaffold description while preserving both historical review rounds. Final source SHA256:

| Path | SHA256 |
| --- | --- |
| `r105_ui_runner.py` | `b83dfacc3b4849c6d54f5b824d5360732d706ff1cd2fe26d7c9cdc7e7f93415d` |
| `test_r105_ui_runner.py` | `6c9c3bd9ba3f5dddc35a7c9fcd5675bfbf88b2137620e5ce35e23ff0b19960d0` |
| `r105_vite_config.mjs.template` | `8c8a40d463247443044578a16cdd2452f44c090d6f5c7fdf5a6068e67c5efbda` |

The final candidate records command timeouts and launch failures before raising: bounded command receipts include redacted argv, timeout limit, partial stdout/stderr, null exit code and failure type; each process launch has an exclusive intent receipt before `Popen` and a success/failure result receipt. A reader thread records its own failure and partial-log metadata. The supervisor joins every reader, verifies the retained log size/hash and reports any reader, metadata, stop or protected-inventory failure in cleanup. Multibyte truncation retains only valid UTF-8 bytes, with explicit retained/full redacted byte counts and truncation state. Cooperative `stop` repeats idempotently.

The packet parser now requires exactly one unfenced full-line `Status: ACTIVE` and a matching record `workOrder`, `workOrderStatusGrammar: cvf.workOrderStatusGrammar@1.0.0`, and `workOrderStatus: ACTIVE` for start. The proposed formal seed additionally requires `dockerHost: npipe:////./pipe/dockerDesktopLinuxEngine`; remote Docker hosts fail admission. The Docker CLI receives the explicit pinned local pipe and a new empty config directory inside the R105 runtime output, with the minimal child environment. It does not use ambient Docker contexts, user Docker config, credentials, or remote endpoints. Root separately recorded a read-only Docker CLI compatibility check at `docs/reviews/probes/r105_dispatch_preparation/docker_readonly_binding_check.json`; this worker did not invoke Docker. These added fields need to be included in the future formal seed/record; R105 remains DRAFT.

Actual round-2 offline checks: `python -B -m unittest docs/reviews/probes/r105_root/test_r105_ui_runner.py -v` **11 PASS**, including injected subprocess timeout, process-launch failure, reader failure, UTF-8 cap, repeated stop request, and full prospective admission with fake Git and code-fence/path-drift rejection. `Get-Content -Raw docs/reviews/probes/r105_root/r105_vite_config.mjs.template | node --check --input-type=module` exit 0. `python -B docs/reviews/probes/r105_root/r105_ui_runner.py inspect --project-root .` exit 0, checking 137 captured frontend bytes and the historical binary SHA256. Root's round-1 independent read-only check separately verified 17 historical Git blobs; no historical Git claim is inferred from this worker's `inspect` command.

No runtime `start/status/stop`, Docker CLI, SQL, backend, Vite, browser, provider, mock, network, build, installation, credential read, commit, or FREEZE was performed by this worker. Real lifecycle safety and R101 product/browser evidence still require effective formal authority, Root source review and a separately admitted runtime observation. This candidate and its offline tests do not claim CVF AI governance behavior.

## Source repair round 3 — independent new root cause

Root found `LIVE_HANDLE_LOST_AFTER_SPAWN_ON_RECEIPT_FAILURE`: `Popen` could succeed, then writing its result receipt could fail before the caller received the direct child handle. This was distinct from round-2 timeout and log-reader findings. `ShellAdapter.launch` now owns that failure window. On a post-spawn result-receipt exception it calls `stop_owned` on the local `Popen` handle before raising; the HOLD identifies the owned PID and reports whether the direct-handle stop completed or failed. It does not enumerate PIDs, kill a process tree, or retry the launch.

Final round-3 source SHA256:

| Path | SHA256 |
| --- | --- |
| `r105_ui_runner.py` | `f27c5bfc729c67bf717473a7dbc894352446c9f74813be08243350e46a4d478a` |
| `test_r105_ui_runner.py` | `92924900b0c39b068babdb769d5e57048ea7dd948486215e2401e8289d5d3c40` |
| `r105_vite_config.mjs.template` | `8c8a40d463247443044578a16cdd2452f44c090d6f5c7fdf5a6068e67c5efbda` |

Actual round-3 checks: `python -B -m unittest docs/reviews/probes/r105_root/test_r105_ui_runner.py -v` **12 PASS**, including injected result-receipt failure after a fake successful spawn and assertion that its exact child handle was stopped. `Get-Content -Raw docs/reviews/probes/r105_root/r105_vite_config.mjs.template | node --check --input-type=module` exit 0. `python -B docs/reviews/probes/r105_root/r105_ui_runner.py inspect --project-root .` exit 0 with 137 pinned frontend files and exact historical binary. Runtime effects, Docker/SQL/browser/provider calls and commits remain zero. R105 remains DRAFT/NOT_DISPATCHABLE pending Root review and formal authority.
