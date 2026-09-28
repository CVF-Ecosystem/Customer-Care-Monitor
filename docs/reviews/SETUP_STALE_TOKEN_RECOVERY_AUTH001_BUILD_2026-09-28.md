# BUILD evidence — `CCMAI-AUTH-001` Setup with a stale browser token

**Date:** 2026-09-28/29 · **Worker:** Claude (`IMPLEMENTATION_WORKER`) · **Status:** `REVIEW_PENDING` for independent Codex review · **Authority:** [work order](../work_orders/CCMAI_AUTH_001.md), [SPEC](../specs/SETUP_STALE_TOKEN_RECOVERY_AUTH001_2026-09-28.md) · **Risk:** R2 · **Claim boundary:** frontend routing and auth-state behavior only. The tests use the real router guard and auth store with a mocked API and synthetic tokens. The browser check used a disposable Compose project. No provider call, no saved credential, and no governance claim.

## Source cause (proven)

The bug has two parts.

1. **Redirect loop (hang).** The `router/index.ts` guard sent every route except Setup to Setup when `needs_setup=true`. It then applied the guest rule to Setup ("guest route + any token → `/`"). A stale token therefore produced `/setup → / → /setup → …`.
   - The guard awaits only on its first pass, when it fetches the status, so every later pass resolves in microtasks. The loop never yields to the event loop, which is why the browser tab stops responding and why a timer-based test could not end it.
   - Before the fix, the regression test reproduced this with the real guard. After 50 token reads the test reports "redirect loop detected" for entry at `/setup`, `/login` and `/`. The first version of the test, which relied on a timer, hung the test worker the same way.
2. **Profile/refresh path.** `App.vue` called `fetchProfile()` in `onMounted`. `app.mount` runs before the first navigation's async setup-status check finishes, so a stale store token could request `/profile` first. A 401 then triggers the interceptor refresh. When the refresh fails, the interceptor sets `window.location.href='/login'` and reloads the whole page. The regression test's App case shows this ordering: mutation M3 below reintroduces the early profile request and is caught.

## Fix and why each changed file is needed

| File | Change | Why |
|---|---|---|
| `frontend/src/router/index.ts` | When the server answers `needs_setup=true`, the guard clears the local session if one exists, then resolves: Setup is allowed and every other route redirects to Setup. The guest and requires-auth rules no longer run in that state. A failed status request still sets `needsSetup=false`, so it clears nothing and does not admit Setup. | Makes a positive status take precedence over browser credentials and removes the loop. |
| `frontend/src/stores/auth.ts` | New `clearLocalSession()` clears `user`, `accessToken` and `tenantPerms`, and removes `cqa_access_token` and the legacy `cqa_refresh_token` without calling the server. `logout()` is unchanged. | The store keeps its own token ref, separate from localStorage, so clearing only storage would leave `App.vue` with a stale in-memory token. |
| `frontend/src/App.vue` | Awaits `router.isReady()` before deciding whether to load the profile. | Ensures the guard has checked the setup status and cleared stale credentials before any profile/refresh request can run. |
| `frontend/src/__tests__/setup-stale-token.spec.ts` (new) | 7 tests. | Regression and acceptance coverage. |

`api/index.ts` is not changed: once stale credentials are cleared before the profile load, no 401 or refresh occurs on an unconfigured instance. The HttpOnly refresh cookie from an earlier installation is outside this tranche (SPEC §3).

## Before/after regression

| Run | Result |
|---|---|
| Before the fix (source at `546b81f`, new test file) | **4 failed, 3 passed.** The three stale-token entries (`/setup`, `/login`, `/`) and the App case fail with `{kind:'error', error:'redirect loop detected'}`. The configured-installation, status-error and successful-Setup cases pass, which confirms they describe existing behavior. |
| After the fix | **7 passed.** |

## Acceptance matrix (SPEC §2)

| # | Requirement | Evidence |
|---|---|---|
| 1 | Real guard, stale token + `needs_setup=true`: `/setup`, `/login` and `/` settle on Setup with bounded navigation, and there is no profile or refresh request | Tests "entering {path} settles on Setup without a loop" (3 cases). Each asserts `router.push` resolves without failure, the route is `setup`, `/setup/status` is called once and `/profile` is never called. Navigation is bounded by a token-read counter. Refresh cannot run because the API module is mocked, and no request that could 401 is made. |
| 2 | Stale local and in-memory auth state is cleared before an App profile load | The same tests assert both localStorage keys are `null`, `accessToken` is `''` and `user` is `null`. Test "App does not load a profile with the stale token…" mounts the real `App.vue` and asserts `/profile` is not requested. Browser check: see below. |
| 3 | `needs_setup=false` keeps configured behavior; a status error neither clears nor admits | With a token: `/login` goes to `tenants` and `/setup` goes to `tenants`, and the token is kept. Without a token: `/` goes to `login` and `/setup` goes to `login`. With a status error: the token is kept and `/setup` is not admitted. |
| 4 | Successful Setup keeps the new token, loads the profile and reaches the authenticated route | The test starts from a stale token and runs `/setup` → `authStore.setup()` → `markSetupComplete()` → `/`. The route becomes `tenants`, localStorage and the store hold the **new** token, and the profile email is loaded. |
| 5 | Gates | Below. |

**Mutation check (scratch script, each restored afterwards):**
- M2, the guard without clearing: caught, 4 failed.
- M3, `App.vue` without `router.isReady()`: caught, 1 failed.
- M4, a status error treated as `needs_setup`: caught, 1 failed.
- M5, clearing that skips the store's `user`/`accessToken`: caught, 4 failed.
- M1, which keeps clearing but falls through to the old guest rule, passes 7/7. It is an equivalent mutant: once the token is cleared in the same pass, the guest rule cannot fire. A full revert to the original guard is the "before" run above and is caught.

## Gates

| Command | Result |
|---|---|
| `npx vitest run src/__tests__/setup-stale-token.spec.ts` | 7/7 |
| `npm test` | 16 files, **167 passed** |
| `npx vue-tsc -b --force` | exit 0 |
| `npm run build` | exit 0 |
| `git diff --check`, catalog `-Check`, workspace doctor | see the commit |

## Browser corroboration (disposable)

- **Environment.** Compose project `ccma-uishot-auth-20260928235841` on `scripts/ui-screenshots/compose.yml`, with throwaway secrets and tmpfs MySQL. It was left not set up (`needs_setup=true` before and after). Headless Chrome stored `cqa_access_token=none` before each page load, the same placeholder that hung the tab in the UX-014 R1 observation.
- **Captures.** `/setup`, `/` and `/login` at desktop/mobile × light/dark, **12 page loads, all completed**: 0 JS errors, 0 overflow, 0 external requests. The inspected `/` capture shows the Setup form.
- **App log.** Only `GET /api/v1/setup/status` appears, 14 times. There is no `/api/v1/profile` and no `/api/v1/auth/refresh`.
- **Cleanup.** The project was removed with `down -v --remove-orphans --rmi local`, the scratch env file was deleted, and no container or volume remains. The persistent `ccma` stack was not touched.
- **Screenshots.** Not committed, because the work order allows only this evidence file.
- **Build timing.** The image was built before a final revert that restored `logout()` exactly. That revert does not touch the Setup path, and the unit tests were re-run on the final source.

## Remaining limitations

- A failed `/setup/status` request still falls back to "setup not needed", as before and as the SPEC requires. If an unconfigured server is briefly unreachable, the user sees Login rather than Setup until a reload.
- Setup status is still cached once per page session.
- HttpOnly refresh-cookie invalidation after a reinstall and general token validation are out of scope (SPEC §3).
- `docs/guide/s3-storage.md` (the stale `cqa-app` command) remains a separate documentation task.
