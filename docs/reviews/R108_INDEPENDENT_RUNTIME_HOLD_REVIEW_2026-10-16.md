# R108 independent runtime HOLD review

Status: CHANGES_REQUIRED
Risk: R2
Worker: /root/r108_browser_luna (Luna Max)
Reviewer: Codex /root (independent)
Disposition: BLOCKED_READINESS_BEFORE_NATIVE_NO_PRODUCT_RESULT

R108 compiled the exact admitted current source successfully, started its exact fixture once, then stopped before native invocation with `MySQL handshake ended early`. Usage is compile1/fixture1/native0/actionPOST0/retry0. The error is in readiness handling, not an observed REOPEN result. The old R107 REOPEN503 remains unexplained; neither reproduction nor repair is established.

Execution HEAD was `b4a53d4b9fb63c3073b880eb9c7e3e8e6510eafa`, reviewed source `a4b1d490676b5ef9f165446a2aca33f10adb39df`, seed/base `86cee15ac68c315c2bb16f9ba6b25566831bfcc7`. The preserved current-source test binary is 58,031,616 bytes, SHA256 `cb73ee13e0369cca75ea276cc668f30c3ced42cead5f62c56d68f3c1eaa66f93`; it is an eligible source-bound input for a separately admitted successor, not permission to retry R108. No old historical binary was substituted.

## Root findings and accountability

The readiness loop catches OS/timeout errors, but its `_recv_exact` raises Hold on a socket closing before a complete packet. That Hold escapes the loop at the first early EOF. This is an unhandled not-ready probe condition; the original bytes do not establish why MySQL/forwarding closed. Root's source review missed this branch. A successor must treat bounded startup EOF as not-ready, preserve probe outcomes, and stop on real invalid/error packets or deadline; it must not restart the service or silently reset reservations.

The exact stop command exited0. Original stdout was:

```text
Flag --time has been deprecated, use --timeout instead
e1555e03d69360e1cd7729afd9e52856443f5172d76454653beec981a7d6477a
```

The strict complete-stdout equality check reported `fixture stop identity mismatch`. No actual identity mismatch was observed. Root independently inspected the exact fixture as exited and protected Docker safe fields equal to the captured baseline; receipt `probes/r108_root/post_runtime_independent_verification.json`, new starts0/stops0. Original cleanup still says HOLD, reports stopAttempted false despite the actual stop command, and campaign_result is PENDING_FINALIZATION/proposedHOLD while the terminal disposition is HOLD. Do not rewrite those inconsistencies. A successor uses the supported --timeout flag, preserves all stdout, and verifies actual final identity/state rather than conflating a banner with identity drift.

All 19 directly owned Go/Docker commands exited0 without timeout or start failure; the supervisor returned1. Native reservation is absent. Root accepts the worker's truthful blocked disposition and the existence of a compiled binary, not runtime diagnosis/product acceptance. R108 has no remaining fixture-start allowance and will not be replayed.

## Evidence and gates

Worker return and acknowledgment are `R108_REOPEN_DIAGNOSTIC_WORKER_RETURN_2026-10-16.md` and `R108_WORKER_ACKNOWLEDGMENT_2026-10-16.md`. Export `probes/r108_luna/runtime_export_r108-1792155371-21092/` contains 73 copied non-binary runtime files, including terminal disposition, plus its export manifest; the empty docker-cli-config directory is retained. The 58MB binary remains ignored with size/hash recorded. Root verifies source/export hashes before material commit and preserves captured byte representations separately from Git normalization.

Original worker-return gate failed ROLE-01 because the acknowledgment used “Reviewer and commit steward:” rather than the required Reviewer line. The corrected docs-only recheck passed six controls with APPL actually checked while the order was ACTIVE. Both full transcripts remain in r108_luna. That static PASS does not change the runtime HOLD. Independent final gates use exact order/return/review packets; after blocked status APPL is NOT_APPLICABLE_WITH_REASON, not checked closeability.

## Next bounded product route

Under the owner's existing audit/self-handle and connected-environment grant, Root prepares one product-focused successor: reuse the compiled current backend (zero new Go compiles), repair readiness/cleanup in a new supervisor while preserving R108, repair the consolidated intervention UI-language findings, and admit one fixture campaign with one native diagnostic followed by one backend/Vite UI phase only if native passes. Root owns the connected browser; Luna has source/runtime roles only. The finite successor must state action/time/resource budgets and source admission before effects. No repeated owner approval is needed inside that granted objective/risk/effect boundary, and no old seed/budget is widened.

No production backend source repair is justified yet. Dashboard3/full M2/M3/pilot/release remain OPEN; R102 local frozen acceptance and parked accounts remain unchanged. UI/UX end-user language is mandatory and not yet repaired or rendered-verified.
