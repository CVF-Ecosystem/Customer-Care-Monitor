# R114 Runtime Worker Return

Status: BLOCKED_WITH_REASON
Risk: R2
Worker: /root/r114_onboarding_luna
Reviewer: Codex /root
Tranche: CCMAI-RUNTIME-114
Phase: BUILD

## Result

The sole admitted R114 supervisor start was issued once with:

`python -B docs/reviews/probes/r114_luna/r114_workflow_runner.py start --project-root .`

The durable exec session was `42009`. It initially returned no stdout or stderr and remained active at the worker's first 30-second poll. The campaign produced `running.json` and `ready.json` under `.cvf/runtime/r114_workflow/campaign/`; Root observed the local browser URL and synthetic owner, then owned all browser actions. Root later polled the same exec session and recorded `Unknown process id 42009` in `docs/reviews/probes/r114_root/final_inspect/process_and_terminal_observation.json`. No actual worker process exit code, final stdout/stderr, campaign result receipt, original cleanup receipt, or original source-after receipt is available. Campaign duration is not established from a terminal receipt.

Root issued one cooperative stop request using the runner's `stop --project-root .` command and recorded exit 0 / `STOP_REQUESTED`. That request was made after the campaign deadline on the resumed turn; there is no evidence the original supervisor received it. No duplicate stop, restart, or fixture stop was performed by the worker. Root independently observed the fixture exited and exclusive, all four protected containers matched the campaign baseline, and the backend/Vite processes and listeners were absent at final inspection. These observations do not establish original supervisor cleanup or exit.

## Independently observed bounded UI outcome

Root's `docs/reviews/probes/r114_root/independent_outcome.json` records the onboarding label as `Cai dat AI` with no `AI Provider` developer label, and all five chip labels fully visible at CSS viewport 390×844 in queue/detail views. Root separately qualified screenshot raster dimensions and MIME: queue JPEG 390×843, detail JPEG 375×811, with a derived PNG recorded without resizing. These are Root observations; this worker performed no browser actions.

The bounded result records B07 as inherited R113 UI, B08 as full onboarding labels observed at 390×844 with raster/history qualifications, and B09 as NOT_RUN because an offline control is unsupported. Runtime counts recorded by Root: one fixture start, one backend launch, one Vite launch, one product tab, one synthetic login and logout, two UI POSTs total, zero case-action POSTs, zero native or Go invocations, zero runtime frontend validations, zero provider or mock-governance calls, zero retries, and zero new fixtures.

Root's independent outcome disposition is `CHANGES_REQUIRED_NO_FREEZE` because original supervisor terminal, cleanup, and source-after receipts are missing. The bounded onboarding UI observation is accepted as qualified evidence; the runtime tranche is not frozen. No full-suite, full M2/M3, release, hosted-CI, or AI-governance claim is made.

## Preserved evidence and limits

Root preserved the surviving campaign files as an exact physical copy in `docs/reviews/probes/r114_root/campaign_snapshot/`, with manifest `docs/reviews/probes/r114_root/campaign_snapshot_manifest.json` and archive SHA-256 `990612332c0bb3e3e606a1de9ce303d6d589ac48aeb39836e1785d056b888029`. The snapshot contains 185 files / 73,584 bytes. It preserves existing receipts without fabricating missing supervisor records.

Relevant Root records are `docs/reviews/probes/r114_root/independent_outcome.json`, `docs/reviews/probes/r114_root/independent_source_after.json`, `docs/reviews/probes/r114_root/final_inspect/process_and_terminal_observation.json`, `docs/reviews/probes/r114_root/final_inspect/inventory.json`, and `docs/reviews/probes/r114_root/campaign_snapshot_manifest.json`. The later independent source readback reports a clean frontend/backend worktree, matching binary hash, and admission pins, while explicitly qualifying itself as distinct from the missing original supervisor source-after receipt.

No retry or further runtime operation is authorized by this return. Root owns independent final disposition and any continuity or follow-on work under separately recorded authority.
