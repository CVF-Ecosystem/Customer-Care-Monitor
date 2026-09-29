# CCMAI-RUNTIME-012 BUILD evidence — OAuth credential persistence truth

**Tranche:** `CCMAI-RUNTIME-012` · **Role:** IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD (Claude) · **Date:** 2026-09-29 · **Base commit:** `792fc1c` · **Risk:** R2 · **Authority:** [SPEC](../specs/RUNTIME_OAUTH_CREDENTIAL_PERSISTENCE_S1_2026-09-29.md), [work order](../work_orders/CCMAI_RUNTIME_012.md) · **Status:** `REVIEW_PENDING` for independent Codex review; no FREEZE.

## Rehydration and role transition

Before BUILD, Claude re-read the manifest, policy, active state, active handoff, SPEC and work order; core `26c686c` matches the manifest. `ACTIVE_SESSION_BOOTSTRAP_READ_MODEL.json` is absent (`BOOTSTRAP_MIGRATION_PENDING`, non-blocking). The role transition `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` was recorded in the active handoff before any source edit.

## Source trace and change

Only `backend/api/handlers/channels.go` changed in source. Both callbacks keep their order: missing code/state, R010 config admission, signed-state verification, tenant-scoped channel lookup, decrypt, token exchange(s), encrypt. The final write moved into one private helper, `persistOAuthCredentials(route, channelID, tenantID, updates)`:

- The predicate is now `id = ? AND tenant_id = ?` with the verified tenant (it was `id` only).
- It returns true only when `.Error == nil` and `RowsAffected == 1`. Otherwise it logs a bounded class (`<route> OAuth credentials for channel <id> not persisted: write failed` or `... N rows updated`) — no SQL error text, token, credential, code or state — and the callback returns the existing `redirectWithError(c, tenantID, "Authorization failed")`.
- Update field sets (Zalo: credentials, updated_at, external_id when an OA id is known; Facebook: credentials, external_id, name, updated_at) and the success 302 Locations are unchanged. No retry, compensation, refresh or new status field. The external code exchange is not reversed; a failed write is reported as a failure with no claim that tokens were stored.

## Tests

`channel_config_admission_test.go` gets one optional field on the shared recording transport, `stubOutbound.hook`, called after a request is recorded and before its synthetic reply (no behavior change when unset). New file `oauth_credential_persistence_test.go` reuses that fixture (disposable MySQL, synthetic tenants/state/credentials/replies; no real endpoint):

- `TestOAuthCallbackUpdateErrorIsNotSuccess` (zalo, facebook): a `BEFORE UPDATE` trigger scoped to the fixture channel forces a write error after the exchange. Asserts the exact tenant-scoped `Authorization failed` Location, the full expected outbound sequence before the failure, all channel rows byte-identical (credentials, external_id, name, updated_at), no `success`, SQL/trigger text, tokens, code or state in response or log, and a log naming the class and channel.
- `TestOAuthCallbackZeroRowUpdateIsNotSuccess` (zalo, facebook): the transport hook reassigns the channel to the other fixture tenant at the token step, after the authorized lookup. Asserts the hook fired once (non-vacuous), the same failure Location/outbound/leak checks, and that the reassigned foreign-tenant row changed only in `tenant_id` — no token, external id, name or timestamp written to another tenant's row.
- `TestOAuthCallbackSuccessPersistsExactlyOneRow` (detector): with the same hook observing, the accepted callback returns the unchanged success Location, exactly the expected outbound calls, exactly one row changed (the verified tenant's channel), and the decrypted credentials hold the synthetic tokens.
- The existing R010 OAuth tests still pass unchanged. Package globals (transport, client, loader) are restored in `t.Cleanup`; triggers are dropped; no `t.Parallel`.

## Commands and results

1. Focused: `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./api/handlers -Run 'TestOAuthCallback|TestChannelConfig|TestChannelOAuthCallbacks' -VerboseTests` → exit 0, all PASS (`ok .../api/handlers 6.124s`).
2. Full: `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1` → exit 0; all 13 packages with tests `ok` (`api/handlers` 18.5s, `engine` 6.2s).
3. From `backend/`: `go build ./...` and `go vet ./...` clean; `gofmt -l` on the new test file prints nothing.
4. Catalog `powershell -ExecutionPolicy Bypass -File scripts/manage_cvf_downstream_catalog.ps1 -Check` → PASS; doctor `powershell -ExecutionPolicy Bypass -File ../.Controlled-Vibe-Framework-CVF/scripts/check_cvf_workspace_agent_enforcement.ps1 -ProjectPath <project>` → `RESULT: PASS (25/25)` after synchronization; `git diff --check` clean (only LF/CRLF notices).
5. Cleanup: the script removed its disposable MySQL container and network after every run (`docker ps -a --filter name=ccma-test` shows none). The persistent Compose database was not used; `go.mod`/`go.sum` unchanged.

## Mutation checks — performed

Each mutation was applied temporarily to `channels.go`, the focused OAuth tests were run, and the file was restored byte-for-byte (`cmp` against a backup):

- Helper predicate reduced to `id = ?` only → `TestOAuthCallbackZeroRowUpdateIsNotSuccess` failed for zalo and facebook (callback returned the success Location and wrote to the reassigned row).
- `Error` and `RowsAffected` checks disabled → `TestOAuthCallbackUpdateErrorIsNotSuccess` and `TestOAuthCallbackZeroRowUpdateIsNotSuccess` failed for both routes.

## Untested behavior and limits

- No real Zalo or Facebook endpoint, real token or real channel was used; the exchange is a synthetic transport. Real provider error shapes, token expiry and repeated authorization are untested.
- A persistence failure leaves the provider-side authorization in place; the user must retry. No compensation is attempted, by design.
- A concurrent second callback for the same channel is not serialized; only the final write's truth is fixed. Sync single-flight, crash recovery, scheduler, `CreateChannel`, re-auth and UI behavior remain separate S1 work.
- Not live CVF governance proof: no provider API call was made and none was required, because no governance behavior is claimed.
