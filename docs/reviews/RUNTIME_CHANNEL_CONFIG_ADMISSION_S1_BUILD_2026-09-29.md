# CCMAI-RUNTIME-010 BUILD evidence — channel credential/OAuth configuration admission

**Tranche:** `CCMAI-RUNTIME-010` · **Role:** IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD (Claude) · **Date:** 2026-09-29 · **Base commit:** `7660aea` · **Risk:** R2 · **Authority:** [SPEC](../specs/RUNTIME_CHANNEL_CONFIG_ADMISSION_S1_2026-09-29.md), [work order](../work_orders/CCMAI_RUNTIME_010.md) · **Status:** `REVIEW_PENDING` for independent Codex review; no FREEZE.

## Rehydration and role transition

Before BUILD, Claude re-read the manifest, policy, active state, active handoff, session memory, implementation status, docs index, SPEC and work order. Workspace doctor passed 25/25 at `7660aea` with core `26c686c`. `ACTIVE_SESSION_BOOTSTRAP_READ_MODEL.json` is absent, so this is recorded as `BOOTSTRAP_MIGRATION_PENDING` (non-blocking). The handoff's `Current State` header was stale (it still showed CREDIT-002). It was reconciled, not treated as drift, because state, memory and the R010 tranche section agreed on the next move. The role transition `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` was recorded in the active handoff before any source edit.

## Source trace and change

Only `backend/api/handlers/channels.go` changed. It adds one private seam, `var loadChannelSecurityConfig = config.Load`, used by all five sites. There is one load per request and no cache. Each site now checks `err != nil || cfg == nil` before its first use of the config, and logs only a failure-class line (`... not admitted: configuration invalid`), never the validation error:

| Handler | Checks kept before the load | Rejection (existing generic response) | Side effects the check now precedes |
|---|---|---|---|
| `CreateChannel` | JSON binding / 400 | 500 `{"error":"create_channel_failed"}` | Encryption, Facebook page-token exchange, Pancake health check, channel insert |
| `TestChannelConnection` | tenant-scoped lookup / 404 | 500 `{"error":"decrypt_failed"}` | Decryption, success response |
| `ReauthChannel` | tenant-scoped lookup / 404 | 500 `{"error":"Decrypt failed"}` | Decryption, OAuth state signing, `redirect_url` |
| `ZaloOAuthCallback` | missing code/state redirect | 302 `/login?zalo_auth=error&message=Authorization+failed` | State verification, channel lookup, token exchange, OA fetch, channel update |
| `FacebookOAuthCallback` | missing code/state redirect | same redirect as Zalo (existing `redirectWithError`) | State verification, lookup, three Graph exchanges, channel update |

The callbacks cannot run a tenant check before the config load because the tenant comes from the HMAC-signed state, and verifying that state needs `JWTSecret`. The existing missing-parameter redirect still runs first. The rejection redirect carries no tenant, state or code. Valid-path code after each check is unchanged. It uses the same `cfg` pointer as before: status, body and redirect shapes, state format, credential schema and token persistence are all unchanged. Existing gofmt deviations elsewhere in `channels.go` were not touched.

## Tests

New file `backend/api/handlers/channel_config_admission_test.go`, run on disposable MySQL. The fixture uses two synthetic tenants and synthetic Zalo and Facebook channels whose credentials are encrypted with a synthetic 32-byte key. It also uses:

- a stubbed loader that counts calls;
- `httpClientWithTimeout` and `http.DefaultTransport` replaced by a recording `RoundTripper` that answers the known Zalo/Graph endpoints with synthetic tokens.

No real OAuth endpoint, channel or provider is contacted. Package globals are restored in `t.Cleanup`, and no test uses `t.Parallel`.

- `TestChannelConfigFailureIsNotAdmitted`: covers 5 routes × {loader error, nil config}, 10 subtests in total. Each subtest checks:
  - the exact status and body or Location;
  - exactly one config load and no panic;
  - zero outbound requests;
  - all channel rows (id, tenant, name, external_id, credential bytes, updated_at) byte-identical before and after;
  - none of `SECRET`, `ENCRYPTION_KEY`, `state=`, `redirect_url`, `success` or the OAuth code in the body, Location or log;
  - the log names the failure class.
- **Accepted-path detector** `TestChannelConfigAcceptedPathsReachSuccessBoundary`: with a valid config, the same fixture observes each route's side effects:
  - Facebook create: 201, one Graph `me/accounts` request, and a stored page token that decrypts with the validated key.
  - Zalo create: 201 and a stored row, with no outbound request.
  - Connection test: the unchanged 200 body.
  - Re-auth (Zalo and Facebook): a `redirect_url` whose `state` equals `signOAuthState` with the validated secret, and no outbound request.
  - Zalo callback: `?zalo_auth=success`, two requests, and stored tokens plus `external_id`.
  - Facebook callback: `?fb_auth=success`, three requests, and a stored page token.

  This shows that the zero-outbound and unchanged-row assertions in the failure tests can detect side effects when they occur.
- `TestChannelConfigRoutesUseTheLoadedConfig`: a loaded config with a different key and secret makes decryption fail and callback state be rejected, with no outbound request or write. The routes therefore use the loaded pointer.
- `TestChannelConfigRequestChecksRunBeforeConfigLoad`: wrong-tenant connection-test and re-auth requests return 404, and a malformed create returns 400. Missing code, missing state or both give the existing `Missing+code+or+state` redirect for both callbacks. In every case the config load count is 0 and nothing changes.
- `TestChannelOAuthCallbacksKeepStateAndTenantOrder`: a forged signature yields `Authorization+failed`. A state validly signed for another tenant yields that tenant's `Channel+not+found` redirect. Each case loads the config exactly once, with no outbound request or write.
- `TestChannelConfigUsesRealConfigValidation`: the real `config.Load` runs with synthetic `t.Setenv` values. A JWT secret that is too short rejects re-auth and the Zalo callback, with no `JWT` text in body or log and no change. A valid environment completes the Zalo callback with stubbed outbound.

## Commands and results

1. Focused: from `backend/`, `powershell -ExecutionPolicy Bypass -File ../scripts/test-backend.ps1 -Packages ./api/handlers -Run 'TestChannelConfig|TestChannelOAuthCallbacks' -VerboseTests`. All 6 tests and their 26 leaf cases PASS; `ok .../api/handlers 12.739s`.
2. Full: `powershell -ExecutionPolicy Bypass -File ../scripts/test-backend.ps1` gave exit 0. All 13 packages with tests are `ok`, including `api/handlers` (30.2s) and `engine` (13.5s). The earlier R007/R008/R009 channel, sync and job tests are part of that run.
3. `go build ./...` and `go vet ./...` from `backend/` are clean. `gofmt -l` on the new test file prints nothing.
4. `git diff --check` is clean. The only output is the existing LF→CRLF notice for the handoff file.
5. Doctor, which also runs catalog `-Check`: `powershell -ExecutionPolicy Bypass -File ../.Controlled-Vibe-Framework-CVF/scripts/check_cvf_workspace_agent_enforcement.ps1 -ProjectPath <project>` gave `RESULT: PASS (25/25)` before BUILD. After the status, roadmap, handoff, memory and state were synchronized, it again gave `RESULT: PASS (25/25)`, with the catalog check passing.
6. Cleanup: the test script removed its MySQL container and network after each run. `docker ps -a` and `docker network ls` filtered on `ccma-test` show nothing. The persistent Compose `ccma` database was not used. `go.mod` and `go.sum` are unchanged.

## Mutation check — not performed

A temporary source mutation was attempted to show that the failure tests would fail without the guards. The local permission classifier refused it because it temporarily disabled the five security guards. It was not retried in any other form, and the source was confirmed unchanged afterward. Non-vacuity therefore rests on the accepted-path detector above, which runs the same fixture, stubs and row/outbound observers, and on the fact that a nil config on the pre-BUILD source would dereference `cfg.EncryptionKey`/`cfg.JWTSecret` and panic. A reviewer who wants direct mutation evidence can revert only `channels.go` in an isolated worktree and rerun the focused tests.

## Untested behavior and limits

- No real Zalo, Facebook or Pancake endpoint, real token, or real channel was used. The outbound paths are proven only against a synthetic transport. The accepted Pancake create path, which runs a live health check, is not exercised; its admission is covered by the shared pre-encryption check.
- `TestChannelConnection` still returns success without an adapter health check (existing TODO, out of scope).
- The callback success paths still ignore the result of the final `db.DB...Updates` call. That write-error handling is outside this tranche's config-admission scope and is recorded for a later tranche.
- `agents.go`, scheduler and sync single-flight, crash recovery and UI/API intersections remain separate S1 work.
- This is not live CVF governance proof: no provider API call was made, and none was required, because no governance behavior is claimed.
