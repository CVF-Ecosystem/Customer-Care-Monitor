# Work order CCMAI-RUNTIME-003 — S1 replayed message integrity

**State:** `READY_FOR_ASSIGNEE_ACK` · **Risk:** R2 · **Assignee:** Claude (`IMPLEMENTATION_WORKER`) · **Independent reviewer:** Codex (`REVIEWER`) · **Authority:** user “tiếp tục” after Gate B REVIEW PASS and `docs/specs/RUNTIME_REPLAY_MESSAGE_INTEGRITY_S1_2026-09-27.md`.

## Entry and role route

This tranche inherits S0 corpus and the accepted S1 sync truth/snapshot contracts from `CCMAI-RUNTIME-001/002`; it enters at BUILD after this SPEC/WORK_ORDER. `CCMAI-RUNTIME-002` passed REVIEW but remains FREEZE open. Before BUILD, Claude must rehydrate the current manifest/policy/state/memory/handoff/status/index, declare the CVF context and append the `WORK_ORDER_AUTHOR (Codex) -> IMPLEMENTATION_WORKER (Claude)` acknowledgment to the active handoff. Claude then returns a local BUILD evidence commit for Codex's independent R2 REVIEW. No self-approval or FREEZE.

## Objective and allowed paths

Make a same-ID replay within the fetched window update stored message facts that the adapter actually supplies, while preserving the internal message identity, valid local attachment paths and honest sync status. The result must feed the existing shared snapshot builder so its digest reflects changed stored facts.

Allowed implementation paths: `backend/engine/sync.go`, `backend/engine/*sync*_test.go`, `backend/engine/snapshot_test.go`, and, only if a typed presence/completeness signal is necessary, `backend/channels/adapter.go` plus the three existing adapter mapper files/tests (`pancake.go`, `facebook.go`, `zalo_oa.go` and matching tests). Do not alter provider, analyzer, result, DB model/migration, frontend or CVF core code. Allowed evidence/continuity paths: a new `docs/reviews/RUNTIME_REPLAY_MESSAGE_INTEGRITY_S1_BUILD_2026-09-27.md`, the active handoff, active state, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json`, and the roadmap/spec status lines if source truth changes. Keep unrelated `.gitignore` and `docs/references/` worktree entries out of the commit.

If the implementation needs a path outside this class, changes schema, or requires a real channel/provider call, stop before that effect and return a bounded change request to Codex.

## Required implementation

1. Audit current adapter mapping and pagination from local source/test fixtures. Record a capability table for Pancake, Facebook and Zalo: stable external ID, fields supplied on replay, removal/edit signals and fetch-window limits. Treat missing upstream fields as unknown, not proof of deletion.
2. For a found row, serialize/validate incoming data before mutation. Update changed, supplied nonempty content, valid role/name/content type and nonzero timestamp; update attachment identity without transferring a saved `LocalPath` to a different attachment. Preserve the existing ID and avoid unnecessary writes on identical replay. Do not erase populated fields from omitted/zero adapter data. Keep raw data error handling explicit.
3. Propagate every update/serialization error into the existing channel `partial` path. `last_sync_at` must remain the last successful checkpoint and after-sync jobs must not start for a partial run. Do not convert missing or removed upstream rows into deletes in this tranche.
4. Add focused tests that execute the SPEC acceptance on disposable MySQL `CCMA`: changed same-ID Vietnamese text/emoji and sender/timestamp produce one updated row and changed snapshot digest; unchanged replay is idempotent; attachment identity controls local-path retention; a write error is observable and cannot produce false success. Reuse existing sync/snapshot helpers where possible. Tests must fail on the pre-change behavior for the central stale-message case.

## Evidence and exit

Record changed set, capability table, before/after regression, DB fixture/cleanup, exact commands and results, unresolved adapter limits and the no-provider claim boundary in the BUILD evidence file. Run focused tests, `go test ./... -count=1`, `go build ./...`, `go vet ./...`, AutoMigrate twice on disposable MySQL `CCMA`, catalog `-Check`, workspace doctor and `git diff --check`. Do not use mock output as CVF governance proof; this work order makes no CVF runtime-governance claim.

On success, Claude transitions `IMPLEMENTATION_WORKER -> SESSION_SYNC_STEWARD -> COMMIT_STEWARD`, updates continuity/status, creates one local commit without push, and returns `REVIEW_PENDING` to Codex. If tests fail or an acceptance condition remains open, report `BUILD_BLOCKED` with exact evidence and do not claim completion.

**External-effect ceiling:** local source/docs/tests and disposable MySQL only. No provider API, credential, real channel sync, customer data, persistent Compose database reset, deployment, push, S2/S3/S5 implementation, cross-project edit or FREEZE.
