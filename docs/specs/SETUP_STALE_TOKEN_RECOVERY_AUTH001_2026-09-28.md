# SPEC — Recover Setup when the browser retains an old token (`CCMAI-AUTH-001`)

**Date:** 2026-09-28 · **Author:** Codex (`ORCHESTRATOR → SPEC_AUTHOR`) · **Risk:** R2 (authentication flow) · **Status:** BUILD done (Claude), `REVIEW_PENDING` for independent Codex review ([evidence](../reviews/SETUP_STALE_TOKEN_RECOVERY_AUTH001_BUILD_2026-09-28.md)).

## 0. INTAKE and source facts

UX-014 R1 observed a browser tab hanging on `/setup` in a disposable, unconfigured instance when `cqa_access_token=none` remained in localStorage. `/auth/refresh` returned 401; removing the token let Setup render ([observation](../reviews/SETTINGS_USERS_AUTH_UX014_BUILD_2026-09-28.md)). This is an observed reproduction, not a completed root-cause proof.

Source inspection identifies two compatible paths to address:

1. `router/index.ts` checks `GET /setup/status` once. If `needs_setup=true`, every route except Setup redirects to Setup. It then treats Setup as a guest route and, if any token string exists, redirects to `/`. That produces `/setup → / → /setup` for a stale token.
2. `App.vue` calls `authStore.fetchProfile()` on mount if the store initialized with a token. A 401 may cause `api/index.ts` to refresh and, on failure, set `window.location.href='/login'`. The API interceptor uses localStorage for authorization, while the store keeps a separate token ref.

`GET /setup/status` is public; `POST /setup` creates the first owner and returns an access token. Setup already records that token and loads the profile through the auth store. No server contract change is needed.

## 1. DESIGN decisions

- A **successful** server response with `needs_setup=true` takes precedence over any browser credential. Setup must be reachable from `/setup`, `/login` and protected routes without a redirect loop.
- On that positive response, discard the prior installation's local access token and legacy refresh-token key, clear corresponding in-memory authentication state, and prevent a stale-token profile/refresh attempt from forcing the page off Setup. Do not use an old token to authorize first-time setup.
- When `needs_setup=false`, preserve the configured-installation login, guest and protected-route behavior. A failed `/setup/status` request is not proof that setup is required; do not clear credentials or silently grant setup access because of that failure. Existing error handling may remain, but record its limitation.
- After successful `POST /setup`, keep the new access token and profile, mark setup complete and navigate normally. Do not delete the new token on the next route.
- Resolve this in the frontend only. Do not change backend setup policy, provider calls, database state, tenant permissions or the persistent `ccma` stack.

## 2. Acceptance

1. A focused automated test uses the **real application router guard** with mocked `GET /setup/status`; a stale nonempty token plus `needs_setup=true` resolves `/setup` in bounded navigation. It also covers entry at `/login` and `/`, and proves no loop or unexpected profile/refresh request. Tests should catch a reversion to the current guard order.
2. The same positive setup status clears local and in-memory stale auth state before an App profile load can run. Mock or spy on profile/refresh to show no stale request; a disposable browser capture may corroborate it. Do not call a real auth/provider API solely for this test.
3. `needs_setup=false` retains configured-installation guest/protected behavior with and without a token; a setup-status error does not clear a credential or admit Setup.
4. The successful Setup flow retains the **new** token, loads profile and reaches the authenticated route.
5. Focused tests, full frontend Vitest, forced typecheck, production build, diff check, catalog check and workspace doctor pass. Record commands/results and any browser evidence in the BUILD artifact. Synthetic tokens only; no secrets in logs or screenshots.

## 3. Claim boundary and open items

Mocked auth/setup tests prove frontend routing behavior only. They are not CVF governance evidence and need no provider call. HttpOnly refresh-cookie invalidation across a reinstall and general auth-token validation are outside this tranche. Separately correct the old `cqa-app` command in `docs/guide/s3-storage.md`; do not fold that documentation change into this auth BUILD.
