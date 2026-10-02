# Shared learning — shell cleanup and path conversion

Date: 2026-10-02. Status: LOCAL_GUIDANCE_RECORDED / UPSTREAM_ASSESSMENT_DEFERRED.
Source: owner-transferred summary of Claude-local `feedback_shell_cleanup_and_paths.md`. The private memory file was not read or copied; no shell incident was reproduced in this intake. Maintainer: project ORCHESTRATOR / SESSION_SYNC_STEWARD; applies to any provider or agent using the relevant shell.

## Finding and scope

Owner reports that `rm` with a shell-variable argument was blocked, causing the combined command to perform none of its intended steps, and that MSYS converted an argument of the form `name=/path`. These are environment-specific execution failures, not proof that every variable-based removal is invalid or every shell behaves like MSYS. The exact rejected command/log is not present in the shared evidence; classify the incident as OWNER_REPORTED, not independently verified.

## Reusable practice

1. Keep resource teardown, filesystem deletion and post-cleanup verification as separate commands. Check each result explicitly; a rejected cleanup command must not suppress teardown or masquerade as completed cleanup. Do not group everything behind a shell chain whose first rejection hides later work.
2. On Windows, use one shell end-to-end, preferably native PowerShell `Remove-Item -LiteralPath` for file cleanup. Before recursive deletion or moving, resolve the absolute target and verify it stays inside the intended disposable workspace or explicitly named directory. Use explicit, verified paths; a fixed absolute path still needs validation. Do not pass enumerated PowerShell paths into `cmd /c` or another shell for deletion.
3. When Git Bash/MSYS must pass a literal path-like argument to a native executable, apply `MSYS_NO_PATHCONV=1` to that individual invocation only and check its effect on the intended argument. Keep ordinary path translation enabled elsewhere. This setting addresses argument conversion, not approvals or filesystem permissions; never use it to bypass a rejected action.
4. Capture exit codes and a secret-free resource inventory after teardown. Verify named containers, networks, volumes, images and temporary files independently. If cleanup fails, name the residual resources and required follow-up; do not claim cleanup from an attempted command.

## Acceptance for future tooling

A future authorized cleanup helper should test a project path with spaces, an MSYS literal filter/path argument, missing resources and failure in each cleanup stage. A teardown failure must remain visible and must not silently become success after later file cleanup. A path outside the declared disposable boundary must be rejected before mutation. These are proposed tests, NOT RUN here; no helper/tooling change is implemented by this learning.

## Evidence and disposition

[F07 BUILD evidence](../RUNTIME_DASHBOARD_SERVICE_STATUS_F07_BUILD_2026-10-02.md) records the worker's disposable capture environment and cleanup inventory. It provides context for the resource lifecycle; it does not independently establish the reported shell root cause. R029 BUILD is `4c6653021878827cba678adb1ae87e9a196d5e85`, REVIEW_PENDING; this learning grants no F07 review acceptance.

Store this shared record in Git, index it through `docs/catalog/ARTIFACT_REGISTRY.json` and generated `docs/INDEX.md`, and link it from `CVF_SESSION_MEMORY.md`. Read when preparing shell commands or resource cleanup. New incidents should append exact sanitized command shape, shell/version, exit code, observed effect and disposition here, not only to provider-local memory.

Parent CVF assessment is [deferred through the existing learning intake](CCMAI_TO_CVF_DOWNSTREAM_GATE_LEARNING_INTAKE_2026-10-01.md). Proposed earliest control: shared Windows/MSYS execution guidance and cleanup-helper acceptance. No parent edit, automated prevention or runtime AI governance proof is claimed.
