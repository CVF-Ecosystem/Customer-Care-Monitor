# R041 BUILD record — explicit synthetic inventory input for the offline Pancake CLI

Status: BUILT, REVIEW_PENDING (not accepted). Worker: Claude IMPLEMENTATION_WORKER (owner-transferred R2). Reviewer: Codex, independent. [Order](../work_orders/CCMAI_RUNTIME_041.md), [SPEC](../specs/PANCAKE_OFFLINE_INVENTORY_INPUT_R041_2026-10-03.md). Claim scope: offline synthetic input parsing and reconciliation only; not live inventory, provider/channel behavior, CVF AI governance, hosted readiness or FREEZE.

## 1. Baseline, seed and identities

- Seed/baseCommit `b19602ea66a47310b503a03ec2954092560231cc` (`CVF_SESSION/authority/CCMAI-RUNTIME-041.json`, unchanged, not in the changed set). HEAD at BUILD start `e620b73` (dispatch).
- Owner manual transfer: the owner message "Chuyển cho Claude: CCMAI_RUNTIME_041.md" in this session. Rehydration, declaration and WORK_ORDER -> BUILD acknowledgment were recorded in the active handoff and BUILD continuity passed the default preflight (7/7) before any test or edit.
- Source SHA-256 (working tree at BUILD): `main.go` `fd7dcb9e0230056fb605fd0315a18cfa31b84292693a317aeba7a166c4c2084f` (169 lines; baseCommit blob `b657be771d10c5891f2f6915d1a196376d2288ed51cad28c5249d087c974e929`), `main_test.go` `bde3d475a21bbfe2a55ffbc96a83e71e34a6e7c3464dab1f7c1344326f316b2f` (208 lines, **unchanged from baseCommit**), `inventory.go` `328007c292525d05bd803a9f0c8494c293449a76e426c8fe7817f9255e180f69` (366 lines, new), `inventory_test.go` `dad0a4c5908e655693b1518c8eb7d4bf5105c63cc60cbeb12f4b8733ee1d24ed` (477 lines, new).

## 2. Changed set and design

Product source: `backend/cmd/pancake-proof/main.go` (+33/−2), new `inventory.go`, new `inventory_test.go`. `main_test.go` was not touched, so the six existing behavioral tests are byte-identical and unmodified. Governed documentation: this record, SPEC/order status, tranche, state, memory, handoff, status, registry, generated index.

- New option `-inventory PATH` (counted by a small `flag.Value`; a second occurrence is a usage error). Allowed only with the default or an explicit `-scenario pass`; any other scenario is rejected with a fixed message **before** the file is touched. Without the option nothing changes. The loaded inventory replaces only the expectation; the transcript, page, token, since and transport remain the existing synthetic constants.
- `loadInventoryFile` admits one explicit local regular file: rejects empty path, `-`, NUL, `://`, and `\\` / `//` prefixes; `os.Lstat` must report a regular file (symlinks, junctions, directories, devices and pipes are not regular); size over the ceiling is rejected up front; after open, `f.Stat` must still be regular and `os.SameFile` with the Lstat result. No stdin, URL, include, glob, environment default or discovery.
- `parseInventory` reads through `io.LimitReader(r, 1 MiB + 1)` and rejects (never truncates) when more than 1 MiB arrives; checks `utf8.Valid`; tokenizes with `encoding/json` `Decoder.Token` into a generic tree that rejects duplicate keys (even equal values), nesting over 64 and any trailing data; then converts through explicit field-by-field checks (not `Unmarshal` into the proof library types). Each object must contain exactly its required keys (missing -> `missing_field`, extra -> `unknown_field`; matching is case-sensitive, so aliases are missing/unknown), correct JSON types (null/number rejected), schema `pancake-proof-synthetic-inventory/1`, `evidence_type` `SYNTHETIC_OFFLINE`, `provenance` matching the proof label pattern with a `synthetic-` prefix, `page_id` exactly `synthetic-page-001`, `syn-` prefixed safe identifiers of at most 128 bytes (no U+FFFD), unique conversation IDs and per-conversation message IDs, RFC3339/RFC3339Nano instants with explicit zone, nonzero (normalized to UTC), `sender_type` customer|agent, `content_type` text|attachment, attachment `type` image and a 1..256 byte name without control characters or U+FFFD. Limits: 100 conversations, 1000 total messages, 16 attachments per message.
- Errors: the loader returns one `*inventoryError` whose `Error()` is the fixed string `inventory rejected`; a private code exists only for tests. The CLI prints the fixed `pancake-proof: inventory rejected` (exit 2, empty stdout) and never the path, parser/OS error, IDs, names or bytes.
- Not changed: `backend/channels`, `backend/engine`, proof library/options/receipt schema, dependencies, frontend, scripts, workflows, historical seeds.

Synthetic sample: `validInventoryJSON` in `inventory_test.go`, written by hand rather than marshalled from `buildFixture`: two conversations `syn-conv-a` (two messages, one image attachment `photo-001.jpg`) and `syn-conv-b` (one message), provenance `synthetic-file-v1`.

## 3. OI matrix

| ID | Named tests (all in `backend/cmd/pancake-proof`) | Result |
| --- | --- | --- |
| OI-01 | `TestInventoryValidFilePassesThroughMountedMain` (mounted `main` via the existing `execMain` helper: exit 0, PASS, 12 attempts, caller provenance retained, SYNTHETIC_OFFLINE / live false / governance false, two runs identical, an equivalent file reproduces the built-in receipt, explicit `-scenario pass` accepted), `TestInventoryValidFileParsesToCallerExpectations` | PASS; the original-main control fails the mounted acceptance (section 5) |
| OI-02 | `TestInventoryStrictSchemaRejectsEveryMalformedInput`: 49 subtests, each asserting the exact internal rejection code **and** the mounted CLI result (exit 2, empty stdout, fixed stderr). Covers wrong/missing schema, evidence, provenance (3 forms) and page; missing field at root/conversation/message/attachment; unknown field at all four levels incl. a token-like and a raw-capture-like key; case-alias keys; duplicate keys at root/message/attachment even with equal values; null/number/string/object type errors; trailing garbage and a second document; truncation; empty file; non-object root; invalid UTF-8 and U+FFFD; unsafe/duplicate/too-long/unprefixed IDs; time without zone, date-only, zero instant; enum violations; attachment name empty/control/257 bytes | PASS |
| OI-03 | `TestInventoryReadAndResourceLimits` (counting reader: an endless source stops at exactly 1 MiB+1 bytes; an exactly-1 MiB file parses and passes through mounted main; 1 MiB+1 is rejected by the parser and by mounted main; 100/101 conversations, 1000/1001 messages, 16/17 attachments; depth 64 accepted by the decoder and 65 rejected), `TestInventoryFileAdmissionAndOptionConflicts` (missing, directory, empty, `-`, UNC, `//`, `file://`, `http://`, NUL; scenario conflicts rejected before the file opens, proven by a nonexistent path yielding the conflict message; empty path, duplicate option, missing value) | PASS, except the symlink probe (section 6) |
| OI-04 | `TestInventoryExpectationsStayIndependentOfObservations`: nine divergent expectations (extra and removed conversation, removed and added message, sender, content type, timestamp, attachment name, conversation `updated_at`) each exit 1 and never PASS, each with a digest distinct from the base and from each other; empty expectation (section 6); equivalent offsets (`+07:00`, `-05:00`) and reordered conversations/messages give the same digest and PASS | PASS |
| OI-05 | `TestInventoryInputNeverAppearsInOutput`: canary conversation/message/attachment/key/value/path/filename absent from stdout/stderr for a FAIL receipt and for four rejected inputs; nonexistent and directory paths give only the fixed message; input file unmodified; no files created beside it; poisoned environment gives identical behavior; `-live/-token/-config/-base-url/-db` stay rejected next to `-inventory` | PASS |
| OI-06 | section 4 | PASS |

## 4. Commands and results

Project root, `GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local CGO_ENABLED=0`, go1.27.0 windows/amd64, no downloads.

| Check | Result |
| --- | --- |
| Workspace doctor | 25/25 PASS before BUILD |
| Default preflight after BUILD sync, before any test or edit | 7/7 PASS |
| `gofmt -l backend/cmd/pancake-proof` | no files listed |
| `go -C backend test ./cmd/pancake-proof -count=1 -v` (uncached) | exit 0, `ok 12.733s`: 13 top-level tests PASS (6 pre-existing + 7 new), 49 subtests PASS, 0 FAIL, 0 `--- SKIP` lines; one attributed logged skip (section 6) |
| `go -C backend test ./channels -count=1 -v` (uncached, finite in-memory fixtures) | exit 0, `ok 5.022s`: 116 top-level tests PASS, 0 FAIL, 0 SKIP |
| `go -C backend build -o NUL ./...` | exit 0 |
| `go -C backend vet ./...` | exit 0 |
| Race | NOT RUN: `go env CC` is `gcc` and no `gcc` is on PATH (the race detector needs cgo); CGO stayed disabled |
| Protected paths vs baseCommit (`backend/channels`, `backend/engine`, `backend/go.mod`, `backend/go.sum`, `frontend`, `scripts`, `.github/workflows`, `CVF_SESSION/authority`) | `git diff --name-only` empty (0 paths); `git diff --name-only <base> -- backend` lists only `backend/cmd/pancake-proof/main.go` plus the two new files |
| docs build, catalog, diff check, PR-range/changed-set preflights, gate unit tests | recorded in the active handoff |

## 5. Control and mutations

Runner outside the repository (`r041_mut.py`, session scratchpad): asserts exactly one match, applies the one-match change, runs `go -C backend test ./cmd/pancake-proof -count=1`, restores the original bytes in `finally`, verifies the SHA-256 equals the pre-mutation hash, then reruns the baseline. Baseline before the campaign exit 0; final baseline exit 0.

| ID | Applied change (1 match each) | Result | First failing assertion |
| --- | --- | --- | --- |
| M1 | expected-input override bypass: `inv = loaded` -> `_ = loaded` (`main.go`) | KILLED (3 tests) | `TestInventoryValidFilePassesThroughMountedMain`: receipt missing `"provenance": "synthetic-file-v1"`; also the expectation and canary tests |
| M2 | size-cap bypass: read-check ceiling raised by 1 GiB (`inventory.go`) | KILLED (1) | `TestInventoryReadAndResourceLimits`: endless input code `syntax` after reading 1048577 bytes, want `too_large` |
| M3 | nested unknown-field acceptance: `len(obj) != len(keys)` -> `len(obj) < 0` | KILLED (8) | unknown-root/conversation/message/attachment subtests (also depth, token-like key and canary tests) |
| M4 | duplicate-key acceptance: `dup` -> `dup && false` | KILLED (4) | `.../duplicate-root-key-same-value`, `.../duplicate-message-key`, `.../duplicate-attachment-key`: want `duplicate_key`, got none |
| M5 | raw error/path leakage: CLI prints path and error beside the fixed message | KILLED (52) | every fixed-stderr assertion, including `TestInventoryInputNeverAppearsInOutput` |
| M6 | scenario conflict checked after file open: condition -> `false` | KILLED (1) | `TestInventoryFileAdmissionAndOptionConflicts`: `missing-conversation: want the scenario conflict before the file is opened, got 2 "pancake-proof: inventory rejected"` (captured by rerunning that one mutant restricted to the test, because the campaign's first captured line was the logged symlink skip; mutant restored byte-equal again) |
| M7 | trailing-data acceptance: `err != io.EOF` -> `&& false` | KILLED (3) | `.../trailing-garbage`, `.../trailing-second-document`: want `trailing_data`, got none |
| M8 | null accepted as string: `!ok` -> `!ok && obj[key] != nil` | KILLED (2) | `.../null-required-string`: want `type`, got `provenance` |
| C0 | original `main.go` (baseCommit blob `b657be77...e929`) overlaid with the new loader and tests | KILLED (55 failed entries incl. six top-level), **not a build failure** | `TestInventoryValidFilePassesThroughMountedMain`: `a valid synthetic inventory must PASS offline, got exit 2 stderr="pancake-proof: invalid usage; ..."` (the original rejects `-inventory`) |

**Eight mutations plus the original-main control: 9 KILLED, 0 SURVIVED, 0 INCONCLUSIVE, 0 NOT_APPLIED; every file restored byte-equal; final baseline PASS.** Duplicate and unknown-field cases are covered by separate mutants (M3 and M4).

## 6. Failures, deviations and limits

- **Symlink admission NOT verified on this host.** `os.Symlink` fails ("A required privilege is not held by the client"), so `TestInventoryFileAdmissionAndOptionConflicts` logs `SKIP symlink admission` and continues (it does not call `t.Skip`, so the test passes without exercising that branch). The directory case and other non-regular files are covered through the `os.Lstat` mode check; a reviewer on a host that can create symlinks should confirm `not_regular_file` for a link.
- **SPEC wording vs library behavior (empty expectation).** SPEC OI-04 says an empty expected inventory "remains INCOMPLETE, exit 1". Against the unchanged transcript, which yields two conversations, the library reports **FAIL** (`reconciliation_mismatch`, observed rows extra) with exit 1; `INCOMPLETE/empty_inventory` needs an empty observation, which is the existing `empty-inventory` scenario and cannot be combined with `-inventory`. The test asserts the actual FAIL and "never PASS" (4 attempts, because message fetches follow the expected conversations). Reviewer to confirm this satisfies the intent.
- Request counts: for the expectation equal to the transcript, `attempts_used` is 12 (same as the built-in fixture); divergent expectations are asserted as non-PASS exit 1 with distinct digests, not for identical attempt counts.
- File admission reports both unavailable and non-regular files with the internal code `not_regular_file`; a missing path and a directory are not distinguished (both give the fixed message).
- A TOCTOU window between `Lstat` and `Open` is narrowed by the post-open `Stat` and `SameFile` check, not eliminated; inputs are caller-provided local synthetic files.
- Failed attempts: `go build ./cmd/pancake-proof` without `-o` wrote `backend/pancake-proof.exe`; it was detected by `git status` before staging, deleted, and never committed (later builds used `-o NUL`). During development three test edits broke the build or an assertion (a null/object case that really exercised `unknown_field`, a literal newline inside a format string from a scripting slip, and the empty-expectation assertion, corrected to the real FAIL above); a long shell heredoc for this record was rejected by the shell before anything ran and the file was written with the editor tool instead. All were resolved before the recorded runs.
- `TestInventoryStrictSchemaRejectsEveryMalformedInput` asserts the specific internal code for each case so a case cannot pass for an unrelated rejection.
- NOT RUN: race detector (no C compiler), engine and full DB suites, Analyzer, real inventory/capture/credential/config read, provider/channel/network, downloads, frontend, GitHub Actions, push, merge, deployment. No `.env` or config was read; the only files read are task-owned synthetic temporary files created by the tests. No live, provider-governance, hosted-readiness or FREEZE claim; the caller-supplied provenance label does not establish independently verified inventory or quiescence.

## 7. BUILD identity and hand-back

Exact BUILD commit: `e9043813ea5b9ffbbf5ee6218d4a9cdb343f515b` (parent `e620b73`; seed `b19602ea66a47310b503a03ec2954092560231cc` unchanged). It is recorded in the tranche record `buildCommit` and the active handoff by a follow-up documentation commit that changes no source (a commit cannot contain its own SHA). Pre-commit validation (docs build, catalog, doctor, diff, preflights, gate tests) is in the active handoff. Independent Codex REVIEW is next; no self-approval, push, merge, deployment or FREEZE.
