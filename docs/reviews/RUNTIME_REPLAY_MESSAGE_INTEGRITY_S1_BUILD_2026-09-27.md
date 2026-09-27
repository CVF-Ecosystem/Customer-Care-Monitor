# BUILD evidence: CCMAI-RUNTIME-003 — S1 replayed message integrity

**Work order:** `docs/work_orders/CCMAI_RUNTIME_003.md` · **Spec:**
`docs/specs/RUNTIME_REPLAY_MESSAGE_INTEGRITY_S1_2026-09-27.md` · **Risk:** R2 ·
**Ngày:** 2026-09-27 · **Role:** Claude (`IMPLEMENTATION_WORKER`) ·
**Kết quả:** BUILD complete, `REVIEW_PENDING` for Codex's independent review.
No FREEZE claimed; `CCMAI-RUNTIME-001/002` FREEZE remains separately open.

## Root cause

`backend/engine/sync.go`'s `upsertMessage`, on finding an existing row for the
same `(tenant_id, conversation_id, external_message_id)`, only updated the
stored `attachments` column, and only when at least one incoming attachment
carried a nonempty `LocalPath` (i.e. a file was freshly downloaded this run).
Every other field — content, sender name/type, content type, timestamp — was
silently ignored on replay. A source message edited within the one-hour
replay buffer (or simply re-fetched, as Zalo's adapter always does since it
does not filter by `since` — see capability table) left the stored row, and
any `ccma.snapshot.v1` digest built from it, stale with no error and no
signal.

## Changed set

- `backend/engine/sync.go`:
  - `upsertMessage`'s existing-row branch now delegates to a new
    `updateExistingMessage`, which builds a `map[string]interface{}` of only
    the fields that are both explicitly supplied (nonempty string / nonzero
    time) and different from what is stored, then issues one `Updates(...)`
    call only if that map is nonempty. An adapter that merely omits a field
    on a given reply can never erase a previously populated value, and an
    identical replay causes zero `UPDATE` statements.
  - New `mergeAttachments(existingJSON string, incoming []channels.Attachment)`
    implements the attachment side of the contract: identity is `(Type, URL,
    Name)` — the same triple `classifyAttachments` (in `snapshot.go`) already
    keys its digest fingerprint on, so this reuses an established concept
    rather than inventing a new one. An attachment whose identity matches a
    stored one keeps that stored `LocalPath` when the current reply doesn't
    supply one; an attachment with a different identity never inherits it. An
    empty incoming list is treated as "adapter didn't include attachment data
    this reply," not as a deletion signal, and leaves the stored list
    untouched (contract point 2).
  - The row's internal `ID` and the create path are untouched. No schema,
    model, DB-model, provider, analyzer, result, frontend or CVF core file
    was touched.
- New `backend/engine/sync_replay_test.go`: five DB-backed tests (below).

No change was made to `backend/channels/adapter.go` or any mapper
(`pancake.go`, `facebook.go`, `zalo_oa.go`): the "explicit nonempty/nonzero
value" rule the spec allows is sufficient to satisfy the contract without a
new typed presence signal, so that conditional allowance in the work order
was not exercised.

## Adapter capability table (source/test audit only — no adapter code changed)

| Channel | Stable external ID | Fields actually supplied on replay (source) | Removal/edit signal | Fetch-window / pagination limit |
|---|---|---|---|---|
| **Pancake** (`backend/channels/pancake.go`) | `pancakeMessage.ID` (`toSyncedMessage`, line 351) | Content (`pancakeContent`, prefers `original_message` over HTML-stripped `message`, lines 448-454), sender type/name (lines 336-348, resolved from `from.id`/`admin_name`), content type (text/attachment/sticker, lines 355-375), timestamp (`parsePancakeTime`, lines 356, 458+), attachments (`toAttachment`, lines 410+) | `pancakeMessage.IsRemoved` (line 200) is parsed and stashed into `RawData["is_removed"]` (lines 397-399) **but is not acted on** — a removed message is not currently distinguished from an edited one by this tranche; deletion/tombstone semantics are explicitly deferred by the spec. | `FetchMessages` (line 271) paginates up to `pancakeMaxPages = 200` batches of ~30 (line 36, comment at 275-276) and stops once a message's `SentAt` is before `since` (line 309) — an edit made to a message older than the fetch window will not be seen. `FetchRecentConversations` similarly bounds by `since`/`updated_at` (lines 213-238). |
| **Facebook** (`backend/channels/facebook.go`) | Graph message `id` (line 189) | Content (`message` field, line 175, 192), sender type/name (from `from.id`/`from.name`, lines 178-186), content type (text/attachment/sticker, lines 193, 233-234, 240-242), timestamp (`created_time`, parsed line 167), attachments (image/video/file/media fallbacks, lines 199-232) | **None found.** No deleted/edited field is parsed from the Graph response in this adapter; Messenger's Graph API does not expose an edit/delete flag through the fields requested here (`id,message,from,to,created_time,attachments,shares,sticker`, line 145). This is an open coverage gap, not a verified "Facebook never signals deletion." | Standard Graph cursor pagination (`paging.next`, lines 248-253, unbounded page count) but the loop returns as soon as it sees a message older than `since` (line 170-172) — same fetch-window bound as Pancake. Message ordering (newest page first) is assumed, not independently re-verified by this audit. |
| **Zalo OA** (`backend/channels/zalo_oa.go`) | `msg["message_id"]` (line 260, formatted via `%v`) | Content (`message` field, line 261), sender type/name (`src`/`from_display_name`, lines 262-271), content type (`type` field, line 284), timestamp (`time` epoch-millis, line 254-255), attachment URL/name for non-text types (lines 283-316) | **None found**, and the adapter explicitly does **not** filter by `since` at all — the comment at line 258 says "Don't filter by since — let DB dedup handle duplicates," so Zalo re-fetches its full paged window (offset/count, lines 229-236) on every sync. This makes Zalo the channel most directly exercised by this tranche's fix: every sync is effectively a full replay of whatever the API still returns for that conversation. | Offset-based pagination (`offset`/`pageSize=10`, lines 229-230, 324) with no hard page cap — it stops only when a page returns fewer than `pageSize` rows (line 321-323). Very long conversations could mean many round trips; not measured by this audit. |

Empty/removed-message and missing-history behavior beyond the above is
**documented as open**, per the spec — this audit read source and existing
adapter tests (`channels/pancake_test.go`, `channels/registry_test.go`; no
Facebook/Zalo-specific fixture test files exist) and did not add or run a
real adapter/provider call.

## Acceptance mapping

| Acceptance | Test | Result |
|---|---|---|
| Same-ID replay with changed nonempty Vietnamese/emoji text, sender name and valid timestamp updates exactly one row and moves the snapshot digest | `TestUpsertMessageReplayUpdatesStaleContentAndDigest` | PASS. Also verified to **fail against the pre-change `upsertMessage`** (see "Before/after regression" below) — this is the spec's required central stale-message case. |
| Unchanged replay keeps row identity/values, causes no meaningful DB mutation | `TestUpsertMessageIdenticalReplayIsNoOpAndPreservesAttachmentLocalPath` | PASS. Proven with a `BEFORE UPDATE` trigger that turns any `UPDATE` on the row into an error — the call still returns `nil`, so no `UPDATE` was ever issued. |
| Attachment with matching identity retains its local path | Same test as above | PASS. |
| Attachment identity change does not carry the old local path | `TestUpsertMessageAttachmentIdentityChangeDoesNotInheritLocalPath` | PASS. |
| Empty incoming attachment list must not erase a stored list (contract point 2) | `TestUpsertMessageEmptyAttachmentReplayRetainsStoredList` | PASS (not a named acceptance bullet, but explicit spec contract text; added for completeness). |
| Injected DB write failure returns an error; enclosing sync path still records `partial` without advancing the checkpoint or running after-sync work | `TestUpsertMessageWriteFailureIsReturnedAndLeavesRowUnchanged` plus the **existing, unmodified** `TestSyncProgressFinalStatus` and `TestBuildSyncStatusUpdatesOnlyAdvancesSuccessfulCheckpoint` (`backend/engine/sync_status_test.go`) | PASS. **Boundary stated per the spec's own allowance**: a direct `SyncChannel`-level test would need a real channel adapter, which this work order forbids. `SyncChannel`'s handling of an `upsertMessage` error is unchanged by this BUILD (`sync.go` lines ~189-196: any error goes through `progress.fail(...)` and sets `conversationFailed = true`), and that handling is exactly what the two cited existing tests already prove drives `partial` status and a frozen `last_sync_at`. This BUILD's own new test proves the half those existing tests can't: that `upsertMessage` itself returns the write error instead of swallowing it, and that the row is not left half-mutated by the failed statement. |
| Adapter capability table cites concrete source/test paths, distinguishes observed from assumed | Capability table above | Done — every cell cites a file and line range from this repo; unverified points are explicitly marked "not measured"/"open," not claimed. |

## Before/after regression (proves the tests are not vacuous)

`git stash push -- backend/engine/sync.go` was used to temporarily restore the
pre-change `upsertMessage`, and the five new tests were re-run unmodified
against it:

```
=== RUN   TestUpsertMessageReplayUpdatesStaleContentAndDigest
--- FAIL: TestUpsertMessageReplayUpdatesStaleContentAndDigest (0.25s)
=== RUN   TestUpsertMessageIdenticalReplayIsNoOpAndPreservesAttachmentLocalPath
--- PASS: TestUpsertMessageIdenticalReplayIsNoOpAndPreservesAttachmentLocalPath (0.23s)
=== RUN   TestUpsertMessageAttachmentIdentityChangeDoesNotInheritLocalPath
--- FAIL: TestUpsertMessageAttachmentIdentityChangeDoesNotInheritLocalPath (0.22s)
=== RUN   TestUpsertMessageEmptyAttachmentReplayRetainsStoredList
--- FAIL: TestUpsertMessageEmptyAttachmentReplayRetainsStoredList (0.22s)
=== RUN   TestUpsertMessageWriteFailureIsReturnedAndLeavesRowUnchanged
--- FAIL: TestUpsertMessageWriteFailureIsReturnedAndLeavesRowUnchanged (0.23s)
```

Four of five fail on the pre-change code, including the central stale-message
case the spec explicitly requires to fail. (The idempotency test happens to
pass on old code too — unsurprising, since the old code was already a
silent no-op whenever no attachment carried a fresh local path; that is not a
contradiction, it is the same bug from a different angle.) `git stash pop`
restored the fix before continuing; `git status` confirmed only the intended
files changed afterward.

## Verification

- `go build ./...`, `go vet ./...` — clean.
- `gofmt -l backend/engine/sync.go backend/engine/sync_replay_test.go` — clean
  (both files are LF already, no CRLF-normalization caveat needed). `git diff
  --check` on the changed files — clean.
- Disposable `mysql:8.0` container on an isolated Docker network (no host
  data; container and network removed after the run); persistent Compose
  `ccma` was **not** started or touched by this BUILD (no reason to — no
  schema/config change). `log_bin_trust_function_creators` set once via root
  for the two trigger-based tests, matching the established pattern from the
  Gate B demo/channel regressions:
  - `go test ./engine/... -run 'TestUpsertMessage' -v` — all 5 new tests PASS.
  - `go test ./... -count=1` — all 13 packages `ok` (no existing test was
    modified or broken; `upsertMessage`'s create path and every other engine
    test are unaffected).
  - `AutoMigrate` run twice back-to-back on the same disposable `CCMA` schema
    via a throwaway `go run` (no `-mod=mod`) — both clean, no error;
    `backend/go.mod`/`backend/go.sum` confirmed unchanged by `git status`
    both before and after (no schema/dependency change was made or expected).
- Downstream catalog check (`scripts/manage_cvf_downstream_catalog.ps1
  -Check`) — PASS.
- CVF workspace doctor
  (`../.Controlled-Vibe-Framework-CVF/scripts/check_cvf_workspace_agent_enforcement.ps1
  -ProjectPath .`) — PASS 25/25.
- Cleanup: the disposable MySQL container and its dedicated Docker network
  were stopped/removed after the run.

## Unresolved adapter limits (carried forward, not fixed by this tranche)

- Facebook and Zalo expose no observed removal/edit signal in the fields this
  audit read; Pancake's `is_removed` flag is parsed into raw data but not
  acted on. None of the three is asserted to *never* signal deletion —
  absence of evidence in the fields/tests read is not evidence of absence.
- Fetch-window limits mean an edit made outside the `since` buffer (Pancake,
  Facebook) is simply never seen by this tranche; it is not detected,
  reported, or claimed to be handled.
- Zalo's unbounded offset pagination for very long conversations was not
  load-tested.
- Deletion/tombstone semantics, edits outside fetch windows, a source-version
  protocol, full adapter coverage, provider admission, and S2/S3/S5 runtime
  gates all remain deferred per the spec — this BUILD does not touch any of
  them.

## Claim boundary (initial BUILD, commit `2a4e530`)

No Claude/Gemini/OpenAI/xAI or other provider API was called and no API key
was used. No real channel sync, customer data, deployment, or push. This
BUILD makes no CVF runtime-governance claim and does not claim full adapter
edit/delete coverage — only what the capability table above cites is
asserted, and everything else is marked open. `CCMAI-RUNTIME-001/002` FREEZE
remains separately open and is not affected by this tranche. Local commit
only; Codex is the independent `REVIEWER` for this BUILD next, and Claude
does not self-approve or FREEZE.

## Addendum: Repair round 1 — R003-R1/R003-R2 (2026-09-27)

**Authority:** Codex independent review of commit `2a4e530`,
`docs/reviews/CCMAI_RUNTIME_003_INDEPENDENT_REVIEW_2026-09-27.md`
(`CHANGES_REQUIRED`), narrowed contract appended to
`docs/work_orders/CCMAI_RUNTIME_003.md` ("Repair round 1 — R003-R1/R003-R2") ·
**Repairer:** Claude (`REPAIR_WORKER`, allowed paths `backend/engine/sync.go`
and `backend/engine/sync_replay_test.go` only, plus this evidence and
continuity).

### R003-R1 — replay ignored supplied raw-data changes and serialization errors

`updateExistingMessage` never read `msg.RawData` or `existing.RawData` at
all, so a changed source raw payload (including a newly observed removal/edit
marker such as Pancake's `is_removed`) stayed stale in storage with no
error, and an unmarshalable incoming raw value could return success.

**Fix:** new `mergeRawData(existingJSON string, incoming map[string]interface{})`
in `backend/engine/sync.go`, wired into `updateExistingMessage`'s existing
`updates` map alongside content/sender/content-type/timestamp/attachments:

- A nonempty incoming map is treated as explicitly supplied. It is marshaled
  first, with its error returned immediately (`fmt.Errorf("marshal message
  raw data: %w", err)`) — this is the write-error path the repair acceptance
  requires; a value the standard library cannot encode (a `NaN` float, used
  only in the new test to force this) now surfaces as an error instead of
  silently keeping stale data.
- The incoming canonical JSON is compared against a re-marshaled canonical
  form of the *stored* value (parsed back into a `map[string]interface{}`
  and re-encoded), not a raw string compare — this avoids a false "changed"
  purely from MySQL's own JSON-column reformatting of the stored text, the
  same technique already used for `mergeAttachments`.
- A nil or empty incoming map is indistinguishable from an adapter that
  simply didn't attach raw data to this reply, so it is never treated as a
  change and never erases a previously stored value. This tranche still does
  not act on any removal/edit marker inside the payload (`is_removed` or
  otherwise) — it is persisted as supplied, nothing more; that is explicitly
  out of scope per the SPEC and the repair acceptance.

### R003-R2 — sender-role acceptance lacked an executable assertion

`TestUpsertMessageReplayUpdatesStaleContentAndDigest` changed `SenderName`
but supplied the same `SenderType: "customer"` on both calls, so the SPEC's
"changed sender role/name" acceptance bullet was only half-exercised.

**Fix (test-only):** the same central test now also changes `SenderType`
(`"customer"` → `"agent"`, a valid role transition) and `ContentType`
(`"text"` → `"sticker"`) on the replay call, and asserts both the reloaded
`SenderType` and `ContentType` alongside the existing content/sender-name/
timestamp/one-row/internal-ID/digest assertions. No production behavior
changed for this finding — the pre-existing nonempty/changed check for
`sender_type` already covered it; only the missing assertion was added,
confirmed by the before/after regression below (this specific test still
passes on pre-repair source, exactly as the review's own analysis predicted).

### New regression tests

`backend/engine/sync_replay_test.go` gains three DB-backed tests:

- **`TestUpsertMessageReplayUpdatesChangedRawData`** — a replay with a
  changed, nonempty raw-data map (toggling a `is_removed`-style boolean)
  persists the new value; exactly one row remains.
- **`TestUpsertMessageIdenticalRawDataReplayIsNoOp`** — a structurally
  identical (but distinct Go map value, as a real JSON-decoded replay would
  be) raw-data replay causes no `UPDATE`, proven with the same
  `BEFORE UPDATE`-trigger technique the other idempotency test already uses.
- **`TestUpsertMessageUnmarshalableRawDataReturnsErrorAndLeavesRowUnchanged`**
  — a raw-data map containing `math.NaN()` (unencodable by `encoding/json`)
  returns a non-nil error, and the stored row is confirmed unchanged
  (content and raw_data both compared before/after).

### Verification

- `go build ./...`, `go vet ./...` — clean.
- Fresh disposable `mysql:8.0` container on an isolated Docker network (no
  host data; container and network removed after the run); persistent
  Compose `ccma` was not started (no schema/config change).
  `log_bin_trust_function_creators` set once via root for the trigger-based
  idempotency tests:
  - `go test ./engine/... -run 'TestUpsertMessage' -v` — all 8 replay tests
    PASS (5 existing + 3 new).
  - `go test ./... -count=1` — all 13 packages `ok`.
  - `AutoMigrate` run twice back-to-back on the same disposable `CCMA` schema
    (throwaway `go run`, no `-mod=mod`) — both clean, no error;
    `backend/go.mod`/`backend/go.sum` confirmed unchanged by `git status`
    both before and after.
- **Before/after regression:** `git stash push -- backend/engine/sync.go`
  restored the pre-repair source (keeping the new tests); re-running
  `go test ./engine/... -run 'TestUpsertMessage' -v` showed exactly
  `TestUpsertMessageReplayUpdatesChangedRawData` and
  `TestUpsertMessageUnmarshalableRawDataReturnsErrorAndLeavesRowUnchanged`
  FAIL — precisely the two R003-R1 defects — while the R003-R2 assertion
  addition and everything else still passed (matching the review's own
  characterization that the source's `sender_type` handling was not
  actually broken, only untested). `git stash pop` restored the fix, which
  was then re-verified passing (8/8) before continuing.
- `gofmt -l backend/engine/sync.go backend/engine/sync_replay_test.go` flags
  both files, but only because `core.autocrlf=true` converted their
  working-tree line endings to CRLF during an earlier `git stash pop` in
  this session (confirmed via `file`); stripping `\r` and re-running `gofmt
  -l` on both shows zero diagnostics, and Git's `autocrlf` normalizes back
  to LF on commit regardless of the working-tree encoding, so the committed
  blob is unaffected. `git diff --check` on both files — clean.
- Downstream catalog check (`scripts/manage_cvf_downstream_catalog.ps1
  -Check`) — PASS.
- CVF workspace doctor
  (`../.Controlled-Vibe-Framework-CVF/scripts/check_cvf_workspace_agent_enforcement.ps1
  -ProjectPath .`) — PASS 25/25.

### Raw-data presence rule (recorded per repair acceptance)

An incoming `RawData` is considered "explicitly supplied" if and only if
`len(msg.RawData) > 0`. A `nil` map and an empty (zero-length) map are
indistinguishable in Go and are both treated as "the adapter did not attach
raw data to this reply" — never as a signal to erase or ignore a previously
stored value. This mirrors the identical rule already used for the
attachment list (`mergeAttachments`) and for every other field in
`updateExistingMessage` (nonempty string / nonzero time as the presence
signal).

### Claim boundary (repair round 1)

No Claude/Gemini/OpenAI/xAI or other provider API was called and no API key
was used. No real channel sync, customer data, deployment, or push. This
repair closes only R003-R1 and R003-R2 as named in Codex's independent
review — it does not touch any adapter, provider, analyzer, DB model/
migration, frontend or CVF core file, and does not reopen, re-certify, or
widen any other part of this tranche or of `CCMAI-RUNTIME-001/002` (FREEZE
remains separately open for both). No S2/S3/S5 or governance-runtime claim is
made. Local commit only; Codex re-reviews the changed set and this evidence
next.
