# CCMAI-RUNTIME-012 independent review

**Reviewer:** Codex, independent of Claude's BUILD · **Build commit:** `0edfe80` · **Date:** 2026-09-29 · **Disposition:** `REVIEW_PASS`; FREEZE open.

## Source and evidence

I compared the exact BUILD diff with the [SPEC](../specs/RUNTIME_OAUTH_CREDENTIAL_PERSISTENCE_S1_2026-09-29.md) and [work order](../work_orders/CCMAI_RUNTIME_012.md). Product changes are confined to `channels.go`; the optional transport hook and new test file are under the allowed handler-test path. Both callbacks retain their prior admission, state, lookup, exchange, encryption and success Location. Their final update now filters by `id AND tenant_id`, using the tenant from verified state, and returns success only when `.Error == nil` and `RowsAffected == 1`. Error or zero rows produce the existing generic tenant-scoped `Authorization failed` redirect. The new log contains a bounded failure class, route and channel ID, without the SQL error or token material.

The focused tests force a MySQL update error with a fixture-scoped trigger and a zero-row update by reassigning the channel to another fixture tenant after the authorized lookup. Both routes assert the expected synthetic exchanges occurred, report failure, and do not persist new credentials on failure or into a foreign tenant's row. Accepted-path tests detect exactly one changed row and decrypt the expected synthetic tokens. BUILD evidence records two direct mutations: ID-only predicate and disabled result checks; the focused tests failed for both routes, and the source was restored. `git rev-parse 0edfe80^` confirms the recorded base `792fc1c`.

I independently ran `powershell -ExecutionPolicy Bypass -File ../scripts/test-backend.ps1 -Packages ./api/handlers -Run 'TestOAuthCallback|TestChannelConfig|TestChannelOAuthCallbacks'` from `backend/`: exit 0; `api/handlers` passed in 4.802s; the disposable MySQL container and network were removed. Claude's BUILD evidence records a separate full backend pass (13 packages), build and vet. Catalog and workspace doctor passed, including doctor 25/25.

Before review, I reported `BLOCKED_CONTINUITY_DRIFT`: the handoff Current State header still said WORK_ORDER although active state, memory, its R012 BUILD section and the owner handoff all said REVIEW_PENDING. I corrected only the stale header, recorded that correction and re-read continuity. It was not an acceptance decision.

No blocking finding remains for the bounded persistence contract. Real OAuth endpoints, provider error shapes and concurrent callbacks for the same channel were not tested; external code exchange cannot be undone after a DB failure. These synthetic tests do not prove live CVF governance. R012 passes REVIEW only; S1 remains IN_PROGRESS, with no push, deployment or FREEZE.
