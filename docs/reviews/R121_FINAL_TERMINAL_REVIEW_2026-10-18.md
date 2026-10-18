# R121 original terminal independent review

Status: CHANGES_REQUIRED
Risk: R2
Worker: /root/r121_isolation_luna; repair /root/r121_namespace_audit_sol
Reviewer: Codex /root

The original foreground serve exec30261 returned1. Wrapper receipt holds supervisor25580, waitObserved true, exit1, metadataError null. This establishes actual original termination; R120 missing original proof remains unchanged.

At scripts/r121_ui_supervisor.py:821 the readiness print serializes the Vietnamese manifest with ensure_ascii=False to a redirected stdout pipe. UnicodeEncodeError for U+1ECD occurred immediately after campaign_ready.json publication, before the campaign loop/browser. Root previously observed the ready file without checking whether the following print succeeded; readiness alone was insufficient. No browser tab/login/logout/case action/provider call occurred. Source and binary unchanged. This is a harness encoding defect, not a product behavior failure.

Finally cleanup completed: Vite25940 wait/exit1 after termination, backend27804 wait/exit0; owned fixture e8386c20 exited/Pid0/Ports{}. Actual PowerShell listener observations for3019/62861/56961 each exited0 with empty tables. Protected9containers/10networks/472volumes equal; source405/binary pins equal after cleanup. Fixture network/volume/container preserved. One final stop consumed in supervisor finally; no duplicate stop.

terminal.json is HOLD with FINAL_EVIDENCE_MISSING: vite_routes.jsonl does not exist and the controller requires it before running either accounting validator. Backend raw final fingerprint matches initial and effect counts match (attempts/budgets/executions/grants/receipts/runs0, humanEvents3,snapshots2). These raw markers are retained; combined route/accounting acceptance and UI presentation remain NOT_RUN/NOT_ACCEPTED.

Sol independently assessed runtime/recorder evidence read-only and confirmed the encoding root cause, missing-validator condition, waits/listeners/fixture/protected/source cleanup. Root independently compared source and raw artifacts. This does not self-approve Sol's earlier namespace implementation; its prior source review remains separate.

Evidence: docs/reviews/probes/r121_root/campaign_terminal_raw.zip preserves221 runtime files, excluding docker-config; per-file sizes/hashes and archive hash in campaign_terminal_manifest.json. campaign_terminal_summary.json is a derived review summary, serve_001/receipt.json and tool_observations.json preserve original execution observations. Synthetic fixture values only (cvf-allow-secret-fixture); no provider secrets or governance claim.

Disposition: CHANGES_REQUIRED, FREEZE OPEN. Original sole campaign/start/check caps remain consumed. No automatic retry/successor permitted by immutable seed stopCondition. Only read-only bounded follow-up decision remains; no new runtime authorized by this review. Minimal prospective correction is encoding-safe readiness output, subject to a separately bounded execution decision; no repair or replay performed here. FullM2/M3/M1business/pilot/release OPEN; Facebook/Zalo OA parked; no push/merge/deploy.

Validation inherited before this final record update:19 focused Python tests pass,89 downstream gate tests pass; no further tests consumed. Final repository gates recorded by actual outputs after synchronization. These prove static checks only, never runtime AI governance.

Actual final static validation: legacy preflight7/7 PASS; profile reviewer-fast and pre-commit PASS. APPL closeability explicitly NOT_APPLICABLE_WITH_REASON because BLOCKED_WITH_REASON; it was not checked and is not a completion pass. git diff --check exited0 for this final record update. No runtime governance/hosted CI claim.
