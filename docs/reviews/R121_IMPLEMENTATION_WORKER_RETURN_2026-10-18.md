# R121 implementation worker return

Status: COMPLETE_PENDING_REVIEW  
Phase: BUILD  
Risk: R2  
Worker: /root/r121_isolation_luna  
Reviewer: Codex /root  
Commit steward: Codex /root  
Build grant: `a3f39e5db88dbaa08e0781f9b45ff808af99f98f`  
Base: `33186d0133fdf7272fe61f20ee1c9ac17bb5b11f`

## Delivered source scope

Created only the four authorized source copies. Each target is byte-for-byte equal to its R120 source after the literal byte replacements `R120` → `R121` and `r120` → `r121`. Source line endings and all other bytes are preserved. The original source hashes were unchanged after copying, and neither old identity remains in a target. Hashes and replacement counts are recorded in `docs/reviews/probes/r121_worker/source_copy_manifest.json`.

| Source | New copy | Source SHA256 | New SHA256 | Replacements (upper/lower) |
| --- | --- | --- | --- | --- |
| `backend/api/handlers/intervention_ui_r120_test.go` | `backend/api/handlers/intervention_ui_r121_test.go` | `4c06353d372155e09b7106d140f84061f72022c2dd33c444075b2eeb9bf61cd3` | `9189b27519afb033a5740d5fdc477dcf9c33aca1d9618448d52009de37e294a9` | 28 / 29 |
| `scripts/r120_ui_supervisor.py` | `scripts/r121_ui_supervisor.py` | `6a70d1f38428ed70d7382bc69c8e5e8bfd0f49bad97ceadbc4be46d77b8a0217` | `686a75b1f766960a1e8d223bc8cc80b1776d09a95d4518eff0cfca9b97c9d989` | 27 / 16 |
| `scripts/r120_vite_config.mjs.template` | `scripts/r121_vite_config.mjs.template` | `1881ec5c99f7d9e35c67e30583b5237de00ddd860fe5a010e2ed09c61a115bc1` | `9f2b970e414a1947c9ce78b06a35cf1af5778283d3c50e2ba4edc643a03e13c3` | 7 / 2 |
| `scripts/tests/test_r120_ui_supervisor.py` | `scripts/tests/test_r121_ui_supervisor.py` | `7b6185ebae41be93698d77cb04b97bc2c9a7f4addef0194c5daa07dac1f003d6` | `39a67b509d4e653f02d551d3a0d8824ecaa446c114374d3babfe9959a42bfd4b` | 9 / 3 |

No product implementation logic, shared files, old R120 artifacts, seed, or global continuity records were changed. No source repairs were needed. The capability evidence remains bounded to a short uninterrupted local process lifecycle; it does not establish campaign execution, 1200-second retention, restart survival, or resolve the old R120 disappearance cause.

## Focused worker check

One Python invocation ran the copied offline supervisor unit file:

```text
python -B scripts/tests/test_r121_ui_supervisor.py
```

Result: exit 0; 19 tests passed in 1.347 seconds. Raw output and command/exit receipt are in `docs/reviews/probes/r121_worker/python_check_01/`. Stdout was 0 bytes (SHA256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`). Stderr was 123 bytes (SHA256 `6931c70652245de253330856bd3efbbbe782d0b285a05d78ad2ac5054fbd851b`) and contains the unittest summary. Worker Python checks used: 1/2. Source repair rounds used: 0/2.

No Go compile, product runtime, Docker, database, browser, provider, case action, or commit was performed by the worker. Root retains its separately authorized compile, runtime, review, and commit responsibilities.

## Root-owned recorder static review

`docs/reviews/probes/r121_root/foreground_wrapper.py` was inspected read-only and not executed. Its `parents[4]` resolves to the project root for its current path. It retains the `Popen` object, records the Windows process creation identity, captures stdout and stderr as raw files, waits on the child, writes a wait/exit receipt, and returns the child exit through `sys.exit(main())`. Root must launch the Python wrapper through PowerShell with explicit native exit propagation, such as `python -B <wrapper.py> serve; exit $LASTEXITCODE`.

Potential runtime hold for Root to resolve: after `Popen` succeeds, process-identity capture and `child_started.json` writing occur before `child.wait()` and are not protected by a `finally` or exact-owned fallback. If either operation raises, the wrapper could exit while its newly started supervisor remains running without a wait/terminal receipt. The wrapper is Root-authored and outside this worker’s edit scope.

## Return-time closeability recheck

The active handoff, work order, and tranche identify BUILD and this worker’s four-copy scope. The committed Root BUILD grant and acknowledgment precede these edits. The focused checker passed, and the source-copy manifest records byte-exact transformations and unchanged preimages. This packet supplies the worker return and its evidence. The independent reviewer-fast and later pre-commit/terminal/continuity/committed-range checks remain Root-owned and pending; this return does not self-approve or close R121.

Disposition: COMPLETE_PENDING_REVIEW. R121 source review remains pending; runtime and full M3 acceptance remain open.
