# R108 REOPEN diagnostic worker return

Status: BLOCKED_WITH_REASON
Risk: R2
Worker: /root/r108_browser_luna (Luna Max, IMPLEMENTATION_WORKER)
Reviewer: Codex /root (independent)
Commit steward: Codex /root
Disposition: ONE_ADMITTED_CAMPAIGN_SPENT_HOLD_BEFORE_NATIVE_TEST

## Authority and pre-effect acknowledgment

The worker rehydrated current project continuity, the R108 handoff and work
order, tranche record, immutable seed, source review and committed runtime
admission. The project doctor passed 27/27 checks with the existing warning
that the hidden Core is at `fa34ed49e56648acafe8b0388e48e0aa0523f007` while the
manifest pins `e9ddcbcdcac2a9df520f5eb0761dbdb609b9deba`. The Core still tracks
the public remote and is clean. Knowledge ingest wrote only the ignored worker
index at `.cvf/runtime/r108_ingested_knowledge_worker.json`.

Before effects, the worker acknowledgment was recorded at
`docs/reviews/R108_WORKER_ACKNOWLEDGMENT_2026-10-16.md`. HEAD was
`b4a53d4b9fb63c3073b880eb9c7e3e8e6510eafa`; the worktree was clean before the
acknowledgment. Both admitted source hashes matched the committed
`docs/reviews/probes/r108_root/source_admission.json`. No compile, fixture
start or native invocation had occurred before this campaign.

## Exact admitted call and original result

The worker issued the one admitted supervisor command from the project root:

```text
python -B docs/reviews/probes/r108_luna/r108_native_runner.py start --project-root .
```

The first 30-second tool capture had empty stdout/stderr and the process was
still active. The same process then completed; no second supervisor call was
made. Its final tool output and exit were:

```text
stdout: {"trancheId": "CCMAI-RUNTIME-108", "status": "HOLD", "reason": "MySQL handshake ended early", "evidenceDirectory": "D:\\UNG DUNG AI\\TOOL AI 2026\\CVF-Workspace\\Customer-Care-Monitor-AI\\.cvf\\runtime\\r108_native\\campaign", "elapsedSeconds": 62.428}
stderr: (empty)
exit code: 1
```

The campaign result records 60.721 seconds internally; the command's final
summary reports 62.428 seconds. Both values are retained as emitted. The
campaign exceeded no 600-second limit.

## Actual usage and diagnosis

| Effect | Allowed | Used | Observed result |
| --- | ---: | ---: | --- |
| Offline current-source Go compile | 1 | 1 | Exit 0; binary SHA-256 `cb73ee13e0369cca75ea276cc668f30c3ced42cead5f62c56d68f3c1eaa66f93`, 58,031,616 bytes. |
| Start exact existing R101 fixture | 1 | 1 | Docker returned exit 0; returned ID matched the admitted container ID. |
| Native diagnostic invocation | 1 | 0 | Not reached. The readiness handshake failed before the native invocation reservation. |
| Case action POSTs | 4 | 0 | No action requests were sent. |
| Runtime retries | 0 | 0 | None. No replay is allowed. |

The supervisor stopped with the exact reason `MySQL handshake ended early`.
The native test did not execute, so this campaign establishes neither the
original REOPEN cause nor a reproduced or fixed product result.

## Cleanup and protected-resource evidence

The supervisor attempted to stop only the exact fixture this campaign had
started. The recorded Docker stop command exited 0 and returned the exact
container ID, preceded by Docker's warning `Flag --time has been deprecated,
use --timeout instead`. The runner compared the complete stdout to the ID and
recorded `fixture stop identity mismatch`; the campaign remains HOLD. The
worker did not independently inspect Docker after the supervisor returned, so
the final fixture status is NOT_OBSERVED by this worker. Root will perform the
independent post-campaign inspection. Do not interpret the runner's string
comparison failure as observed container identity drift.

The protected-container before/after safe-field comparison is `true`. All 19
owned Go/Docker command processes recorded exit code 0, no start failure and no
timeout; no process-tree termination was needed. The runner's cleanup record
and command transcript are preserved without alteration.

One evidence consistency gap remains visible: `campaign_disposition.json`
records HOLD, while `campaign_result.json` still says
`PENDING_FINALIZATION` with `proposedStatus: HOLD`; cleanup also reports
`stopAttempted: false` despite the recorded stop command. These original files
are retained unchanged for Root review.

## Preserved evidence

The original ignored campaign remains at `.cvf/runtime/r108_native/campaign/`,
including the compiled binary. A source copy of all 73 non-binary campaign
files is hash-verified at
`docs/reviews/probes/r108_luna/runtime_export_r108-1792155371-21092/`; the
empty `docker-cli-config` directory is retained there. The excluded binary is
preserved in the ignored runtime directory and recorded in the export manifest
at 58,031,616 bytes with SHA-256
`cb73ee13e0369cca75ea276cc668f30c3ced42cead5f62c56d68f3c1eaa66f93`. The
export manifest lists per-file sizes and hashes. The runner's original
`files_manifest.json` is copied as captured; it predates the final disposition
marker, which is separately included and hashed by the export manifest.

No source file, seed, budget or existing receipt was changed by the campaign.
The worker did not use a provider, mock provider, secret/config file, product
browser, backend listener, Vite, installation, download, or a second fixture.
No production repair, commit, self-review, push, deployment or FREEZE is
claimed. Root retains independent review, final fixture-state inspection and
disposition ownership.

## Next governed move

This R108 one-shot campaign is spent and blocked before the native diagnostic.
Preserve the failure and cleanup mismatch. Any later attempt or source repair
requires Root's separately reviewed successor authority; this worker will not
retry or modify the admitted source or runner under this return.
