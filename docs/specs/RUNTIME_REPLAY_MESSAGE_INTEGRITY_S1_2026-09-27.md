# S1 message replay integrity contract

**Tranche:** `CCMAI-RUNTIME-003` · **Phase:** SPEC · **Risk:** R2 · **Source baseline:** `CCMAI-RUNTIME-001` sync truth and `CCMAI-RUNTIME-002` snapshot/evidence, both accepted in REVIEW; neither is FREEZE.

## Problem and bounded outcome

`backend/engine/sync.go/upsertMessage` currently treats an existing `(tenant_id, conversation_id, external_message_id)` as unchanged unless an incoming attachment has `LocalPath`. If a source message is replayed with changed nonempty text, role/name, timestamp, content type or attachment metadata but no new local path, the stored message stays stale. A later `ccma.snapshot.v1` digest then describes that stale row. The one-hour replay buffer and unique index already exist; this tranche makes a replayed, explicitly supplied source value update the existing row without creating a duplicate.

This is a data-reliability slice. It does not decide AI eligibility, execute a provider, resolve upstream deletion semantics or claim every edit can be discovered outside the adapter's fetch window.

## Contract

1. For a matching external message ID, preserve the internal message ID and row count. An identical replay causes no meaningful DB mutation; a changed, nonempty content value updates stored content and changes the next snapshot digest. Valid changed sender role/name, content type and nonzero timestamp are persisted when explicitly supplied by the adapter. Do not overwrite a populated value with an absent/zero value merely because an adapter omitted a field.
2. Serialize incoming attachment and raw-data values with error checks before writing. Preserve a previously stored `LocalPath` only for an attachment with the same stable source identity (type, URL and name, or a documented stronger adapter identity). A different attachment must not inherit the old local path. If an empty incoming list cannot be distinguished from an omitted field, retain the old list and mark the limitation in adapter coverage; do not infer deletion.
3. On any DB or JSON error, return an error to `SyncChannel`. Existing `partial` status, unchanged `last_sync_at` and suppressed after-sync jobs must still hold. No write may claim success while the persisted row is stale due to a detectable error.
4. The snapshot builder remains the one shared single/batch path. A changed stored content/role/time/attachment identity must move the digest; an unchanged replay must not. Existing evidence references to an internal message ID remain addressable, but prior analysis must be understood as bound to its old snapshot digest. This tranche does not silently rewrite prior results.
5. Audit Pancake, Facebook and Zalo adapters from local source/test fixtures: identify which fields are actually supplied on replay, whether removed/edited records are signaled, and how the `since`/pagination windows limit detection. Record unsupported or ambiguous cases as coverage limits. Do not turn absence from a partial page into deletion, and do not claim full upstream edit/delete support from synthetic tests.

## Acceptance

- DB-backed replay test: same external ID twice; second call changes nonempty Vietnamese text (including emoji), sender role/name and valid timestamp; exactly one internal row remains with updated fields, and `ccma.snapshot.v1` digest changes.
- DB-backed idempotency test: unchanged replay keeps row identity and values; attachment with matching identity retains its local path; attachment identity change does not carry the old local path.
- Failure test: injected DB write failure returns an error and the enclosing sync path records `partial` without advancing the successful checkpoint or triggering after-sync work. If a direct `SyncChannel` test is impractical without an actual adapter call, prove the returned error plus existing `syncProgress`/status tests and state the boundary.
- Adapter capability table cites concrete source/test paths for each channel and distinguishes observed fields from assumptions. Empty/removed messages and missing-history behavior are documented as open unless verified.
- `go test ./... -count=1`, `go build ./...`, `go vet ./...`, AutoMigrate twice on disposable MySQL `CCMA`, catalog check and workspace doctor pass. No provider call, real channel sync, customer data, persistent database reset, deployment, push or FREEZE.

## Deferred

Deletion/tombstone semantics, edits outside fetch windows, source-version protocol, full adapter coverage, provider admission and S2/S3/S5 runtime gates require separate specs. Confidence calibration is likewise outside this slice. The existing snapshot transcript already includes message ID, sender type and RFC3339 timezone; this tranche does not reimplement those accepted S1 behaviors.
