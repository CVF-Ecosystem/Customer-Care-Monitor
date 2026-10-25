# R139 v3 activation environment independent source review

Status: REVIEW_PASS
Risk: R2
Worker: Sol Medium /root/r139_v3_activation_source_audit
Reviewer: Root independent source/evidence REVIEWER
Date: 2026-10-26
Scope: same-order activation child environment and internal setup guard; no controller or campaign execution.

Root dispatched supplement177e3512 under the existing Owner repair admission78ae94c6 at commit0eda76156fa075eaa148a36b968eb0f2dbb37622. The new prerelease root cause was concrete: the scrubbed PATH omitted the pinned PowerShell directory, so the legacy catalog gate could not discover its interpreter. The internal60s activation transaction also had to fit large catalog Write/Check, gates and a normal commit. These are operational dependency defects; the prior b330f70b source/checks/accepted-build3366eb41 and all original receipts remain immutable evidence within their prior scope.

Reviewed design SHA-256: `36f5f30777938224a6d3daf0d50b452a466ffcd4dc77f2a9e5b8e0a4ef9e241d`. Reviewed test SHA-256: `4bbd3abb54646a9bcaa691c8b29ccf49f470539475a2a87732b77e2790be2283`. The diff adds a pure deterministic child environment factory used by preclock, including exactly pinned PS5/Git/Python directories and System32 while excluding ambient PATH and Git process overrides. This makes explicit catalog generation and legacy discovery use the same pinned PS5, without importing ambient pwsh or modifying installed gates. The internal setup guard is180s; its boundary and labels match. Original +1020 cleanup/+1200 campaign deadlines, current-query5s, drift2s, browser/service/provider limits and normal Git identity/hooks remain unchanged.

The initial candidate omitted explicit os/shutil imports for new tests; Root caught this statically before any new check and the worker corrected it. The source parser/digest/query-local clock implementation was not changed. The hardcoded source review reference now selects this new review, preserving the old fixed-source review rather than overwriting it.

Worker's single dependency batch PID24424 returned original wait/exit0 with exactly2 tests PASS,0skip. Root's independent full supervisor batch PID15472 returned original wait/exit0 with48 tests PASS,0skip. Both retained raw output and intent/result/source hashes under `.cvf/runtime/r139_pinned_resume_repair_20261026/`; public cumulative v3 worker/root attempt2 receipts link them. No unchanged pin/resume16 suite was repeated. The third ledger generation pins the preceding ledger and all four prior focused results, not a new quota or allowance. Counted source lineage is14 prior plus2 new =16; the six new repair invocations include the retained failed pin attempt.

Root independently inspected the source diff and exact frozen bytes. No Go build/test, OS identity query, browser/API/service/credential operation or activation controller was used. These pure/component checks do not prove runtime AI governance, hosted CI, actual G0–G3/E01–E06, quota-provider behavior or Product B readiness. Root must bind the new source material commit and reseal its wrapper, recipe and actor metadata before the one shared future controller slot can be used. Product B remains BLOCKED_PENDING_OPERATIONAL_ADMISSION; Facebook/Zalo OA remain parked.
