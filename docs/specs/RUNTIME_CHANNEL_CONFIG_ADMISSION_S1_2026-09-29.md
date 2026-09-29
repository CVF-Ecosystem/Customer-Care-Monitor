# S1 channel credential and OAuth configuration admission

**Tranche:** `CCMAI-RUNTIME-010` · **Phase:** SPEC · **Risk:** R2 · **Entry:** R001–R009 independent REVIEW PASS / FREEZE open; S1 IN_PROGRESS.

## Problem and decision

Five call sites in `backend/api/handlers/channels.go` discard `config.Load()` errors: `CreateChannel`, `TestChannelConnection`, `ZaloOAuthCallback`, `ReauthChannel` and `FacebookOAuthCallback`. A failed load returns nil and can panic before encryption, decryption or OAuth state verification. Admit each credential/OAuth action only after a non-nil validated configuration. This is a fail-closed request-boundary fix; it does not change adapters or OAuth success behavior.

## Contract

1. Preserve existing request validation and tenant-scoped channel lookup order. For valid input and a found authorized channel, load configuration before encryption/decryption, state signing/verification, DB mutation or outbound token exchange. Missing OAuth code/state retains its existing redirect without attempting config load.
2. On load error or nil config, return the existing route-appropriate generic 500/error redirect. Do not expose validation text, key material, credentials, OAuth code/state or tokens in response/log. No channel write, success response, redirect URL containing a signed state, or outbound call may occur.
3. On valid config, use that exact config for the route's existing encryption/decryption and OAuth path. Keep existing status/body/redirect shape, tenant isolation, callback state verification, and successful downstream behavior. Do not change credential schema, provider endpoints, callback state format or token persistence semantics.
4. `agents.go`, `jobs.go`, scheduler, manual-sync single-flight and crash recovery are separate tranches. No real OAuth/provider/channel call or CVF governance claim is part of this work.

## Acceptance

- Focused tests cover forced loader error and nil config at each of the five call sites, with nonvacuous no-write/no-outbound checks where applicable; invalid tenant still returns 404 without loading config on tenant-scoped routes. Missing callback code/state remains its existing redirect and skips config load.
- Synthetic valid-config checks prove each route reaches its prior success boundary and uses the validated pointer. Existing OAuth state and credential tests continue to pass. A real `config.Load` validation test uses synthetic environment values only. No actual channel token or external service is used.
- Focused and full backend tests on disposable MySQL, `go build ./...`, `go vet ./...`, catalog check, doctor 25/25 and diff check pass. Evidence names untested external behavior and the absence of live governance proof.
