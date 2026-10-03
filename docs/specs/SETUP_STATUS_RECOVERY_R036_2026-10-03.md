# R036 — Setup-status loading and recovery

Status: SPEC_READY / NOT_BUILT

Date: 2026-10-03. Risk R2. Source baseline b4b119679ad6f37bbac85ed009201a42f9cc8944. Immutable seed 5a044ae82d11cdf59cdf28d0a8fe9f9410957bb3. [Order](../work_orders/CCMAI_RUNTIME_036.md).

## Intake and design

The AUTH-001 review and implementation status explicitly leave failed `/setup/status` fallback open. Current router sets `needsSetup=false` and `setupChecked=true` on request failure, caching an unverified configured state. App waits router readiness then may fetch a profile using a retained token. This is source assessment; frontend runtime tests have not run in this planning turn.

Use unresolved/loading, unavailable, confirmed setup-required and confirmed configured states. Accept only a non-null response object with boolean `needs_setup`; missing, null, string and numeric fields are unavailable. Cache confirmed results for the page lifetime; failures never become configured. Initial check and explicit retry use the existing public endpoint with a status-specific timeout at most 10 seconds. Keep global API interceptors unchanged: `/setup/status` already matches their `/setup` authentication exclusion. Authorization headers and HttpOnly cookies are outside this change.

Register `/setup-unavailable`, name `setup-unavailable`, before the tenant route, using auth layout without guest-token redirection. Render a localized neutral message and accessible Retry button. While the first check is pending, App renders localized loading under auth layout; DefaultLayout and protected content must not mount. Unavailable shows no login/setup form or tenant shell. Hide raw errors/payloads/credentials. Disable Retry visibly during its request. No polling, full-page reload or automatic retry from redirected/repeated navigation.

Only user Retry after unavailable requests status again. Concurrent callers share one in-flight request. Failure stays unavailable with local tokens and auth store intact, re-enabling Retry. Confirmed true preserves AUTH-001: clear stale local access/legacy refresh tokens and store session, settle on Setup without loops. Confirmed false recovers through fixed local `/` and existing auth/tenant/permission guards: token -> normal entry, no token -> Login. No query-supplied return URL. App mounted initially on unavailable must resume an existing-token profile fetch only after configured confirmation, once despite concurrent notifications. Existing login/Setup profile loading and successful `markSetupComplete()` transition stay intact. No new backend request after successful Setup.

Optional `frontend/src/router/setupStatus.ts` may hold reactive status/single-flight state. No auth-store, Setup-view, global interceptor, layout redesign or authorization-policy change. vi/en additions cover status loading, unavailable and retry wording only.

## Acceptance matrix

| ID | Required synthetic local evidence |
| --- | --- |
| SS-01 | Deferred initial check: loading rendered; no DefaultLayout/profile/tenant validation or permissions/login or setup submission before confirmation. Concurrent navigation shares one check and settles without microtask redirect loop. |
| SS-02 | Network rejection, timeout, HTTP401/500, missing/null/string/numeric `needs_setup` and null body settle unavailable. `/setup`, `/login`, `/`, tenant and unavailable entry cannot bypass it. No profile/tenant/refresh/login/setup calls, token deletion/rotation or store clearing caused by uncertainty; no raw errors. |
| SS-03 | Mount real unavailable component in vi/en: clear message, accessible Retry, visibly disabled while deferred. Double Retry makes one new request; repeated failure re-enables it, preserving credentials/store without unhandled rejection. Repeated navigation never auto-retries. |
| SS-04 | Retry true with stale storage/store tokens clears existing AUTH-001 local session fields and reaches Setup; no premature profile/refresh. Preserve three entry routes, no-loop and successful Setup regressions. |
| SS-05 | Retry false with/without token follows existing guards to normal entry/Login. App already mounted on unavailable resumes profile once. Direct initial true/false, confirmed caching and successful Setup remain compatible. |
| SS-06 | Old-source controls and four applied mutations detect false configured fallback, permissive payload coercion, premature profile fetch and broken retry/single-flight behavior by named semantic assertions. Preserve survivors/inconclusive attempts; byte-restore and verify final tests. |

Test actual router and App lifecycle plus mounted real unavailable view; helper-only checks or replacing one old assertion are insufficient. Synthetic tokens/mock API responses prove UI structure/navigation/request sequencing only, never AI/CVF governance decisions. A real common-client interceptor with mocked adapter is preferable for HTTP401 refresh observation to a mocked client that cannot execute interceptors; state precisely which observation is possible. Preserve AUTH-001 positive tests, replacing only obsolete failed-status expectation and compatible helpers. No broad accepted-error workaround. Complete frontend suite must have no unexplained failures/skips; no backend DB test requested.

## Current implementation and boundaries

DISPATCH_READY / NOT_BUILT. Intended frontend contract only, no product fix or independent acceptance yet. Backend status correctness, server installation races, cross-tab state, refresh-cookie reinstall invalidation, global interceptor defects, MCP job execution and live Pancake proof are separate. Previous R035 race/full-backend/frontend/live/provider/GitHub NOT RUN and GORM/checksum/default-client limits remain source-specific history; future R036 UI tests cannot cover them retroactively. No backend security, runtime governance, live availability, hosted readiness or new FREEZE claim.
