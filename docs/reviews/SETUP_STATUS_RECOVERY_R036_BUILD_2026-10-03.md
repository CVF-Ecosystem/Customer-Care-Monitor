# R036 BUILD evidence — frontend setup-status loading and recovery

Date: 2026-10-03 (Asia/Saigon). Tranche `CCMAI-RUNTIME-036`. Risk ceiling R2. Role: Claude IMPLEMENTATION_WORKER / BUILD COMMIT_STEWARD (owner-transferred); acknowledgment recorded in the active handoff `CVF_SESSION/handoffs/AGENT_HANDOFF_SETUP_STATUS_RECOVERY_2026-10-03.md` **before the first source edit**, followed by BUILD synchronization and a passing 7/7 preflight. Independent REVIEWER: Codex. Status: BUILD complete, REVIEW_PENDING, FREEZE OPEN. Authority: [SPEC](../specs/SETUP_STATUS_RECOVERY_R036_2026-10-03.md), [work order](../work_orders/CCMAI_RUNTIME_036.md), dispatcher seed `CVF_SESSION/authority/CCMAI-RUNTIME-036.json` first committed at `5a044ae82d11cdf59cdf28d0a8fe9f9410957bb3` (not edited by the worker).

**Claim boundary.** Synthetic mocked frontend tests of routing, rendering and request sequencing only. No backend, credential, network (beyond in-process mocks), provider or database was used. This is not backend-security evidence, not CVF runtime-governance proof, not live availability, hosted readiness or a FREEZE. R035/R034 REVIEW_PASS / FREEZE_OPEN and R033/local-message FREEZE are inherited unchanged.

## 1. Source identity and changed set

Baseline: dispatch commit `dea891c` (seed `5a044ae`). Exact BUILD SHA: recorded in `CVF_SESSION/tranches/CCMAI-RUNTIME-036.json` `buildCommit` by the follow-up documentation commit. SHA-256 below is of the committed blob (several checkouts use CRLF working copies, so a working-tree hash can differ by line endings only; the mutation runner below worked on working-copy bytes and verified restoration against them).

| File | Change | Committed-blob SHA-256 |
| --- | --- | --- |
| `frontend/src/router/setupStatus.ts` (new) | reactive state, strict parser, single-flight request, finite deadline, explicit retry | `7da9aadd1c424773133de7665087ea21ced45b18b114bd4f64ded892f0723e19` |
| `frontend/src/router/index.ts` | setup-status block and `/setup-unavailable` route only | `66023274e9eb85eec31e2d6d8d1d1b24aaf251d20f43b79c77840b647471d90d` |
| `frontend/src/App.vue` | loading until first navigation settles; profile load only after confirmed configured, once | `d5357684eed24b76a4b6c6582a6fa42de170f2b59570eaee83ee55bede6f65ee` |
| `frontend/src/views/SetupStatusUnavailable.vue` (new) | localized message and accessible Retry | `71386477eca15fa8d420c1323b9679c5d3dbbd9bb2e951bffafc8bcad68a4f94` |
| `frontend/src/i18n/vi.ts`, `en.ts` | five `setup_status_*` keys each | `c4fa500a74c516bf707db39f14edad2deabf80731100a61d7c69014bfe073b09`, `128df13e7e6de67f4a7626557612237dda190ac722bbf6149c218b934e24260d` |
| `frontend/src/__tests__/setup-stale-token.spec.ts` | obsolete failed-status expectation replaced; App mount gets the i18n plugin and a progress stub; AUTH-001 positives untouched | `ac69d4655e0b65d6c9d1b5d6658ec280eb68bac442d3284219e36efd5bf1b8d5` |
| `frontend/src/__tests__/setup-status-unavailable.spec.ts` (new) | SS-01..05 tests (31) | `973e8d2b9a0a7edcab713ced78fc0421d8e4010927646322fdf16d3e2adacae2` |

Not touched: auth store, Setup view, global API interceptor, layouts, permission policy, backend, dependencies, lockfiles, workflows, gate tooling, the authority seed.

## 2. Behavior as built

- **States:** `unresolved`, `unavailable`, `required`, `configured` (`setupStatus.state`). Only a non-null object with a boolean `needs_setup` is an answer (`parseSetupStatus`); null, missing, string, number, array and null-field payloads, rejections, HTTP 401/500 and timeouts are `unavailable`.
- **Request:** one shared in-flight request (`requestSetupStatus` / `retrySetupStatus`), `api.get('/setup/status', { timeout: 8000, signal })` with a local watchdog that aborts at 8 s (≤ 10 s) and ignores a late answer. Confirmed results are cached for the page lifetime. An unavailable result is not retried by navigation; only `retrySetupStatus()` (used by the Retry button) starts a new request, and only while unavailable.
- **Router:** the guard first awaits the status. Unavailable sends every entry (including `/setup`, `/login`, `/`, tenant routes) to the new `setup-unavailable` route (auth layout, no guest redirect, registered before `/:tenantId`) with no token, store, profile, tenant, permission, refresh, login or setup call. Confirmed required keeps the AUTH-001 path (clear stale local session, settle on Setup). Confirmed configured leaves `/setup-unavailable` through the fixed `/` path (no return URL) and then the unchanged guards.
- **App:** auth layout with a localized `role=status` loading text until `router.isReady()`; the default layout and protected views do not mount before that. The profile loads once, only when the state is configured, a token exists and no user is loaded; a later retry that confirms configured resumes it through a state watcher, guarded by a single flag.
- **View:** `role=alert` card with title, description and a native Retry `<button>`; while retrying it is `disabled`, `aria-busy=true` and reads "Đang thử lại..." / "Trying again..."; no form, no raw error text.
- Mounted observations (vi / en): title `Chưa kiểm tra được trạng thái hệ thống` / `System status could not be checked`; button `Thử lại` / `Try again`.

## 3. SS acceptance matrix

| ID | Tests | Observed |
| --- | --- | --- |
| SS-01 | `SS-01 …loading…`, `…follows the locale` | with the status deferred and two concurrent navigations: loading text present (vi and en), `auth-layout` mounted, `default-layout` absent, the only API call is one `/setup/status`, no POST; resolving once settles both navigations with 1 status request |
| SS-02 | 14 failure variants + deadline + real-client 401 | for each variant, entering `/setup`, `/login`, `/`, `/tenant-x/jobs` and `/setup-unavailable` lands on `setup-unavailable`; the complete API call list equals `['/setup/status']`, no POST, token and legacy refresh token in storage and the store unchanged. A never-settling request is aborted at 8001 ms and ends unavailable (late answer ignored; the assertion precedes awaiting the navigation so a missing deadline fails rather than hangs). **Real client + mocked adapter:** the real `api` interceptors run; a 401 on `/setup/status` produces exactly one adapter request, no `/auth/refresh`, no `window.location` change and no token change (observable with the mocked adapter; a mocked client could not exercise the interceptor) |
| SS-03 | view tests (vi, en), double Retry, no auto-retry, repeated failure | message, button, `role=alert`, no input, no raw text; click → status calls 1→2, `retrying` true, `disabled` + `aria-busy`; a second click and a concurrent `retrySetupStatus()` keep it at 2; failure re-enables the button with credentials intact and no non-status API call; five navigations never trigger a retry; three failing retries run without unhandled rejections (count 4 total) |
| SS-04 | `SS-04` ×2 | Retry true with stale storage and store tokens: local session cleared (both tokens null, `accessToken ''`, `user null`), route `setup`, 0 profile calls, 0 POST, under the 50-read loop bound; same via the Retry button |
| SS-05 | `SS-05` ×6 | Retry false with a token → `tenants`, without → `login`; leaving `/setup-unavailable?next=https://evil.invalid` lands on `/login` without the query; App mounted on the unavailable page then three concurrent retries plus an extra notification → exactly 1 profile call and the default layout appears; initially configured → 0 profile calls before the first navigation settles, then 1; confirmed answers cached (1 status request over four navigations for both outcomes); successful Setup path marks configured with no further status request |
| SS-06 | section 5 | old-source control and 10 applied mutations, all detected by named assertions |

AUTH-001 positives (three entry routes settle on Setup without a loop, App does not load a profile before the status is known, successful Setup keeps the new token, configured guest/protected behavior) are unchanged and pass.

## 4. Commands and results

All commands from the project root with cached dependencies only.

| Check | Result |
| --- | --- |
| Workspace doctor | 25/25 PASS before BUILD |
| `npm --prefix frontend test -- setup-status-unavailable setup-stale` | 2 files, 38 tests PASS (31 new + 7 AUTH-001) |
| `npm --prefix frontend test` (complete frontend suite, JSON reporter) | exit 0: 27 files, 299 tests PASS, 0 failed, 0 pending/skipped/todo |
| `node frontend/node_modules/vue-tsc/bin/vue-tsc.js -b frontend/tsconfig.json --force` | exit 0 |
| `npm --prefix frontend run build` | exit 0 (`vue-tsc -b && vite build`, built in ~1.2 s) |
| Backend / DB / live / provider / network / GitHub Actions / browser E2E | **NOT RUN** (out of scope; no authority) |
| Docs build, catalog, diff check, preflights, gate unit tests | recorded in section 8 and the active handoff |

Failed history during BUILD: the first run of the preserved AUTH-001 spec failed once because App now renders i18n text and a Vuetify progress component (`$t is not a function`, then `Could not find defaults instance`); the spec's App mount helper received the real i18n plugin and a progress stub (assertions unchanged). The very first run of the file also hit a 5 s timeout on a cold start while the module cache warmed; it did not recur. The pre-commit preflight `secrets` gate FAILED on the new spec (a quoted synthetic token literal); the two synthetic constants now carry the gate's `cvf-allow-secret-fixture` marker (comment-only change; the focused and full suites were re-run afterwards). The new spec's first run had one test bug (navigating to the page already shown is a no-op); the test now leaves the page first.

## 5. Old-source control and mutations (SS-06)

Runner (outside the repository): exact one-match byte edits to `setupStatus.ts`, `App.vue` or `router/index.ts`, `npm … test -- setup-status-unavailable setup-stale`, bytes restored in `finally`, hash equality checked, baseline before and after (38 passed each). Build errors, timeouts and panics would be INCONCLUSIVE; none occurred.

**Old-source control:** the committed pre-R036 `router/index.ts` and `App.vue` were swapped in with the new spec: **12 tests FAIL semantically**, e.g. `expected 'tenants' to be 'setup-unavailable'` (a failed status opened the app), the loading and no-profile-before-confirmation tests, and the replaced failed-status test.

| ID | Mutation | Result and first assertion |
| --- | --- | --- |
| MS1 | failed request treated as configured (false configured fallback) | KILLED (12): `expected 'setup' to be 'setup-unavailable'` |
| MS2 | payload coercion `Boolean(value)` | KILLED (4): string/number variants, `expected 'setup' to be 'setup-unavailable'` |
| MS3 | profile fetch not gated by ready/configured, started before `isReady()` | KILLED (4): `expected 1 to be +0` (profile loaded prematurely) |
| MS4 | retry single-flight check removed | KILLED (1): double Retry — `expected 3 to be 2` |
| MS5 | navigation retries an unavailable status automatically | KILLED (12): `expected 4 to be 1` |
| MS6 | uncertainty clears the local session | KILLED (12): `expected null to be 'valid-synthetic-token'` |
| MS7 | unavailable page cannot be left after confirmation | KILLED (1): return-URL/leave test |
| MS8 | default layout rendered before ready | KILLED (2): loading state assertions |
| MS9 | status deadline removed | KILLED (1): `expected 'unresolved' to be 'unavailable'` |
| MS10 | retrying flag never set | KILLED (1): `expected false to be true` (button not disabled) |

**10 KILLED, 0 SURVIVED, 0 INCONCLUSIVE, 0 NOT_APPLIED.** The four SS-06 families map to MS1 (false configured fallback), MS2 (payload coercion), MS3 (premature profile fetch) and MS4/MS5/MS10 (retry/single-flight). Mutant sources are not staged or committed; logs and hashes are held outside the repository.

## 6. Limitations

- Mocked API and in-memory storage: no real browser, no real server timing, no cross-tab behavior, no real cookie/refresh flow. The 8 s timeout is exercised with fake timers and an abort signal on a mock, not with a real network stack.
- Only the status request is time-bounded here; the shared client's 120 s default and the global interceptor are unchanged. The HTTP 401 interceptor observation uses a mocked axios adapter (and a mocked global adapter for the refresh call).
- Backend status correctness, installation races, refresh-cookie invalidation after reinstall, MCP execution and live Pancake proof remain separate objectives. No FREEZE, hosted readiness or governance claim.
- Reviewer suggestions: replay MS1, MS3, MS4 and MS9 independently; confirm the replaced obsolete expectation preserves AUTH-001; check `git diff` shows no change outside the listed files.

## 7. Open follow-ups that are not claims

Loading text is shared by the first-navigation wait; if a later tranche adds slow lazy chunks, the same message would show for them. Not addressed here.

## 8. BUILD identity and hand-back

Exact BUILD commit: `a805db2bbb3b30ea1841537351a538c37a1af28b` (parent `dea891c`; seed `5a044ae82d11cdf59cdf28d0a8fe9f9410957bb3` unchanged). The follow-up documentation commit records it in the tranche record `buildCommit` and changes no source. Pre-commit validation (docs build, catalog, diff check, preflights, gate unit tests) is recorded in the active handoff. Independent Codex REVIEW is next; no self-approval, push, merge, deployment or FREEZE.

## Reviewer correction and source-specific disposition (Codex, 2026-10-03)

Independent exact-BUILD [review](CCMAI_RUNTIME_036_INDEPENDENT_REVIEW_2026-10-03.md) is REVIEW_PASS / FREEZE_OPEN for synthetic UI contract only. Original worker REVIEW_PENDING/next-review wording above is historical hand-back. The vi table abbreviation originally ended `4073b09`, which was a transcription error: correct full committed-blob vi/en hashes now replace abbreviations. Reviewer independently computed all eight blob digests; CRLF-normalized working copies equal blobs. No source/test/seed repair; previous worker failures and all mocked-timing/cookie/multi-tab/live limits retained.
