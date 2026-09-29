# S1 OAuth credential persistence truth

**Tranche:** `CCMAI-RUNTIME-012` · **Phase:** SPEC · **Risk:** R2 · **Entry:** R001–R011 independent REVIEW PASS / FREEZE open; S1 IN_PROGRESS.

## Source finding and decision

`ZaloOAuthCallback` and `FacebookOAuthCallback` exchange the authorization code, encrypt new credentials, then call `db.DB...Updates(...)` without checking `.Error` or `.RowsAffected`. Both redirect with `*_auth=success` even if token persistence fails or the channel disappeared. The channel was originally resolved by `(id, tenant_id)`, but the final write currently filters by ID alone. A success redirect must mean exactly one channel belonging to the verified tenant accepted the new credentials.

## Contract

1. Keep the existing missing-code/state, config admission, signed-state verification, tenant-scoped channel lookup, decryption, synthetic token exchange and encryption order. This tranche changes only the final channel write and its disposition; do not change OAuth state format, provider requests, credential schema, token contents or success redirect.
2. For both callbacks, write the existing update fields with a predicate on both `id` and verified `tenant_id`. Treat a DB error or `RowsAffected != 1` as persistence failure. Do not emit any success redirect in these cases; return the existing generic tenant-scoped error redirect shape using `Authorization failed`. Log only a bounded failure class and route/channel identifier, never SQL detail, credentials, tokens, code or state.
3. On exactly one successful update, retain the existing 302 success Location (`zalo_auth=success` or `fb_auth=success`). The already-completed external code exchange cannot be rolled back; a persistence error must be reported honestly, with no claim that tokens were stored.
4. `CreateChannel`, connection test, re-auth, R010 config admission, scheduler, agent/job dispatch, OAuth code exchange reliability, sync single-flight/crash recovery and frontend behavior are separate work. No real OAuth/channel/provider call or CVF governance claim is authorized.

## Acceptance

- Focused tests for both callbacks use a valid synthetic signed state and stubbed HTTP transport. A forced MySQL update error must yield the generic failure redirect, no success, no stored credential/external ID/name change, no SQL/secret leakage. A zero-row final update (for example, tenant reassignment or deletion after lookup through a controlled synthetic exchange hook) must also fail without writing tokens to another tenant's row. Prove the expected outbound exchange occurred before the persistence failure.
- Existing successful callback tests still prove 302 success and persisted tokens on exactly one tenant-scoped row. Add a direct predicate/row-count assertion or a race fixture so a regression to ID-only update or ignored `RowsAffected` is caught. Restore any global test hooks and clean disposable fixtures.
- Run focused and full backend tests on disposable MySQL, `go build ./...`, `go vet ./...`, catalog check, workspace doctor and `git diff --check`. Evidence records commands, cleanup, synthetic limits and absence of live governance proof.
