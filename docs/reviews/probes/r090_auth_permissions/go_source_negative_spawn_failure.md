# Go source-negative launch failure

Risk: R2
Reviewer: Codex /root

Worker: `/root/auth_intake`; tranche: `CCMAI-RUNTIME-090`.

The first fixture-runner invocation `python -B scripts/r089_auth_fixture.py run --proof-name go-source-negative --cwd backend -- ..\\.cvf\\runtime\\toolchain\\go\\bin\\go.exe test ./api -run '^TestR090' -count=1` exited 1 before Go started:

```text
r089_auth_fixture: [WinError 2] The system cannot find the file specified
```

The Go executable exists at the target root's `.cvf/runtime/toolchain/go/bin/go.exe`, but Windows `CreateProcess` did not resolve the relative executable against the child `cwd` supplied to `subprocess.run`. The fixture remained READY; no Go test executed, no DB request or source detector occurred, and no production source changed. The helper now resolves an in-workspace relative executable against the requested child working directory and records future spawn failures as `SPAWN_ERROR` with a persisted log.
