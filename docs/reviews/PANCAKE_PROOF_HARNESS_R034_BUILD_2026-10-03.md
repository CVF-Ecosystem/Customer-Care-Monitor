# R034 BUILD evidence — offline Pancake proof harness

Date: 2026-10-03 (Asia/Saigon). Tranche `CCMAI-RUNTIME-034`. Risk ceiling R2. Role: Claude IMPLEMENTATION_WORKER / BUILD COMMIT_STEWARD (owner-transferred; acknowledgment recorded in the active handoff `CVF_SESSION/handoffs/AGENT_HANDOFF_PANCAKE_PROOF_HARNESS_2026-10-03.md` **before the first source edit**). Independent REVIEWER: Codex. Status: BUILD complete, REVIEW_PENDING, FREEZE OPEN. Authority: [SPEC](../specs/PANCAKE_PROOF_HARNESS_R034_2026-10-03.md), [work order](../work_orders/CCMAI_RUNTIME_034.md), dispatcher seed `CVF_SESSION/authority/CCMAI-RUNTIME-034.json` (first committed at `d869624cc15f55b39516a36f8937454e617dc3b3`, not edited by the worker).

**Claim boundary.** This is offline harness behavior against synthetic in-memory fixtures only. No channel, provider, credential, network, database, media download, analyzer or notification was used. It is not live-channel inventory evidence, not provider-compatibility evidence, not CVF AI-governance proof and not hosted readiness. No FREEZE is claimed; R033/R030–R032 local FREEZE is inherited unchanged and R022–R024 FREEZE, global F02, governance and hosted readiness stay OPEN.

## 1. Source identity and changed set

- Baseline (seed) commit: `d869624cc15f55b39516a36f8937454e617dc3b3`; dispatch commit at start of BUILD: `9ba811b`.
- Exact BUILD commit SHA: recorded in `CVF_SESSION/tranches/CCMAI-RUNTIME-034.json` `buildCommit` and in the final section of this file by the follow-up documentation commit (a commit cannot contain its own SHA).
- New source files (all new, none pre-existing):

| File | SHA-256 |
| --- | --- |
| `backend/channels/pancake_proof.go` | `5a36147d5e13766d403a3fc1a6b8873010c8ca20ca03249c209b179b20f85e2b` |
| `backend/channels/pancake_proof_test.go` | `a16815a894a37cbc1ceb5f825d817cb2fbeb86f9e44a75ae50e6dabe3387da2e` |
| `backend/cmd/pancake-proof/main.go` | `b657be771d10c5891f2f6915d1a196376d2288ed51cad28c5249d087c974e929` |
| `backend/cmd/pancake-proof/main_test.go` | `5c5fe14b2cde9763fd71c3bb48a01ce6e7a1ea775ce8caa49206efaec87392e5` |

- Documentation/continuity touched in the BUILD commit: this record, the SPEC (current-implementation section), the work order (status), the live packet (harness pointer; still NOT DISPATCHED), `CVF_SESSION/` state/handoff/tranche, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json`, `docs/INDEX.md` and the artifact registry/catalog.
- Protected files byte-identical to the baseline: `backend/channels/{pancake,adapter,registry,facebook,zalo_oa}.go`, `backend/engine`, `backend/go.mod`, `backend/go.sum` (git diff against `d869624` is empty; also asserted by `TestProductionAdapterFilesUnchangedInGit`). The authority seed is unmodified. No dependency, workflow, gate-tooling, UI or schema change.

## 2. Design as built

- `PancakeProofOptions` + `ProofInventory` → `RunPancakeProof(ctx, opts, inv) (*ProofReceipt, error)`; `MarshalProofReceipt(receipt, sensitive)`; `ProofTranscript` (finite in-memory `http.RoundTripper`).
- The **unchanged** `PancakeAdapter` performs all traversal and mapping. The harness builds a dedicated in-package instance (struct literal; no edit to `pancake.go`) whose `http.Client` uses only the injected transport wrapped by `proofTransport`, with `CheckRedirect` returning `http.ErrUseLastResponse`. A nil `Transport` is rejected before any work (`transport_required`); there is no `http.DefaultTransport` fallback and no credential/config/env discovery in library or CLI.
- `proofTransport` order per request: admission → shared budget/deadline check → underlying call → 3xx rejection → 8 MiB read bound → independent row observation → sanitized record. Denied and over-budget requests never reach the underlying transport.
- The adapter actually uses `/api/public_api/v2/.../conversations` for the conversation list and `/api/public_api/v1/.../conversations/{id}/messages` for messages. The live packet text mentions v1 for both; the harness admits exactly what the unchanged adapter sends (documented deviation, no adapter change).
- Messages are fetched only for conversations that are both observed by the adapter and approved by the independent inventory; an unexpected observed conversation is reported as `extra` and its message path is never requested.
- Zero `MinRequestInterval`/`RetryBackoff` mean no pacing/backoff (offline fixtures). A later authorized live caller must set them explicitly; that is a documented limitation, not demonstrated live behavior.

### Input schema (library)

`PancakeProofOptions{SourceSHA (40 lowercase hex), PageID, Token (synthetic in R034), PseudonymKey (nonempty), Since (non-zero), Transport (required), MaxAttempts 1..50, MaxDuration >0..10m, MinRequestInterval, RetryBackoff ≥0, Now}`; `ProofInventory{Provenance ([A-Za-z0-9._:-]{1,64}), PageID (must equal option), Conversations[]{ID, UpdatedAt, Messages[]{ID, SentAt, SenderType, ContentType, Attachments[]{Type, Name}}}}`. Inventory lists only rows that must be observed; old/duplicate/non-INBOX rows are expected to be filtered by the adapter.

### Receipt schema `pancake-proof-receipt/1`

Top level: `schema_version, evidence_type ("SYNTHETIC_OFFLINE"), live (false), governance_claim (false), source_sha, started_at, finished_at, inventory{digest, provenance, page_ref, since, conversation_count, message_count}, options{max_attempts, max_duration_ms, min_request_interval_ms, retry_backoff_ms, response_ceiling_bytes, runs}, budget{attempts_used, denied, elapsed_ms}, denials[{run, reason}], runs[], disposition (PASS|FAIL|INCOMPLETE), reasons[]`. Each run: `index, started_at, finished_at, conversation_until, until_stable, requests[{seq, run, kind, endpoint_template, conversation_ref, cursor_ref, current_count, status, outcome, rows, empty_terminal, response_bytes}], conversation_rows / message_rows{physical_rows, duplicate_rows, old_rows, non_inbox_rows, raw_eligible, mapped}, conversation_terminal, message_terminals, message_fetches, conversations / messages{expected, observed, missing[], extra[], mismapped[{ref, fields[]}], adapter_omitted[]}, disposition (adds NOT_RUN), reasons[]`. Identifiers are `HMAC-SHA256(key, "cvf-pancake-proof/v1\0" + domain + "\0" + value)` truncated to 16 hex with a domain prefix (`pag_`, `con_`, `mes_`, `cur_`); the message domain value is `conversationID\0messageID`. `MarshalProofReceipt` fails closed (`ErrPancakeProofUnsafe`) for a non-`SYNTHETIC_OFFLINE` type, `live`/`governance_claim` true, or any raw sensitive value ≥6 bytes in the output.

Disposition rules: demonstrable reconciliation mismatch on a complete run → FAIL; adapter/HTTP/redirect/size/budget/deadline/cancel/missing-terminal/empty-inventory → INCOMPLETE and later runs NOT_RUN; overall = FAIL if any run FAIL, else INCOMPLETE unless every run PASS.

### CLI

`go -C backend run ./cmd/pancake-proof -source-sha <40-hex> -key <nonempty> [-scenario pass|missing-conversation|redirect|empty-inventory] [-max-attempts N] [-max-duration D]` from the project root. It uses a built-in finite transcript (hard ceiling 200) and an independently written expectation. No flag selects a transport, endpoint, credential, config, database, media or output file; any other option is rejected (exit 2, option text not echoed). Stdout: receipt JSON. Stderr: `pancake-proof: SYNTHETIC OFFLINE receipt from an in-memory transcript; not live-channel, provider or AI-governance evidence`. Exit codes: 0 offline PASS, 1 FAIL/INCOMPLETE, 2 invalid input/usage/sanitation failure. The `-key` value appears in the caller's process arguments; it is synthetic by contract in R034.

Receipt excerpt from an actual run (`-source-sha d869624c… -key doc-example-key`, exit 0; timestamps vary):

```json
{ "schema_version": "pancake-proof-receipt/1", "evidence_type": "SYNTHETIC_OFFLINE", "live": false, "governance_claim": false,
  "inventory": { "provenance": "synthetic-cli-fixture-v1", "page_ref": "pag_f6b2b1fa678abac4", "conversation_count": 2, "message_count": 3 },
  "budget": { "attempts_used": 12, "denied": 0 }, "disposition": "PASS",
  "runs[0].requests[1]": { "kind": "conversations", "endpoint_template": "/api/public_api/v2/pages/{page}/conversations", "cursor_ref": "cur_c09df627c239f230", "status": 200, "outcome": "ok", "rows": 0, "empty_terminal": true } }
```

## 3. PH acceptance matrix (tests that carry each requirement)

| ID | Evidence (committed tests) | Observed assertions |
| --- | --- | --- |
| PH-01 | `TestProofPH01TwoRunsExactSequence` | Two runs against one inventory: exactly 20 physical requests (10/run). Per run conversation cursor transitions `∅ → cur(c-old3) → cur(c-ddd5)` ending in an explicit-empty page, then messages per conversation with `current_count` 0,3,5 / 0,1 / 0,1. Physical rows 6 (dup 1, old 1, non-INBOX 1, eligible 3, mapped 3) and message rows 7 (dup 1, old 1, eligible 5, mapped 5). Equal-since conversation and message included; old rows excluded; short pages; zone-less and offset timestamps. Inventory is hand-written, not derived from observed output. |
| PH-02 | `TestProofPH02AdmissionDeniesBeforeTransport`, `TestProofPH02RedirectsNeverFollowed` | 23 denied requests (POST, http, explicit port, alternate host, host suffix, userinfo, fragment, health endpoint, other API version, other page, traversal, encoded separator, unapproved conversation, media host, extra/duplicate query key, wrong/missing token, unobserved cursor, wrong `since`, non-numeric values, bad `current_count`) each return a denial with the underlying transport call count unchanged and no attempt consumed; 2 positive controls are admitted and reach the transport exactly twice. Same-host, other-host and relative 302s: exactly 1 transport call, reason `redirect_rejected`, run 2 NOT_RUN. |
| PH-03 | `TestProofPH03SharedBudget`, `TestProofPH03DeadlineAndCancellation` | N=20 passes with 20 calls; N=19 → INCOMPLETE with exactly 19 calls (request 20 never reaches the transport); N=9 stops before run 1's last explicit-empty page → never PASS; N=10 → run 1 PASS, run 2 first request denied by the shared counter; a 429 retry consumes budget (N=20 INCOMPLETE, N=21 PASS, first record outcome `http_429`); option ceilings 0/51 attempts and 0/10m+1ns rejected with zero calls. In-flight 60 ms deadline ends < 1 s with `deadline_or_cancelled`; fake-clock elapsed stops requests; pre-cancelled context makes 0 calls; mid-run cancel is INCOMPLETE; a >8 MiB body is `response_oversize`, never PASS. |
| PH-04 | `TestProofPH04ReconciliationFailures` (9 cases), `…EmptyInventoryCannotPass`, `…InventoryWithoutMessagesCannotPass`, `…ExpectedButEmptyResultIsFail` | missing/extra conversation and message, wrong sender, content type, attachment type, zone-less timestamp (UTC-parsed vs +07 expectation) and conversation instant each → FAIL with the pseudonymous reference and field name; extra conversation's messages never requested; empty inventory and no-message inventory → INCOMPLETE; expected-but-empty result → FAIL with 3 missing. Raw physical/eligible accounting is observed in the transport independently of adapter output; the actual `until` is recorded per run (`conversation_until`, `until_stable`). |
| PH-05 | `TestProofPH05ReceiptSchemaAndDigest`, `…DeterministicSerialization`, `…InvalidSetupRejectedBeforeWork` | Header/options/inventory summary asserted; UTC `Z` times; refs equal across runs and invocations and equal to the independently computed HMAC; changing one expected instant changes the digest; changing the key changes refs; two serializations (with the wall-clock `conversation_until` zeroed) are byte-identical. 14 invalid setups (missing transport/SHA/key/token/since/provenance, short or uppercase SHA, free-text provenance, page mismatch, duplicate IDs, zero time, negative backoff) rejected with `ErrPancakeProofInput`, zero transport calls and no secret in the error. CLI exit 0 only for offline PASS. |
| PH-06 | `TestProofPH06PseudonymIsDomainSeparatedHMAC`, `TestProofPH06CanariesNeverAppear` | Unique canaries (token, key, page, conversation/message IDs, message text, attachment name/URL, query/cursor, malformed body, transport error text, 500 body, redirect target, panic text) never appear in receipt or errors across 6 scenarios + panic + input-error; transport panic is recovered as INCOMPLETE; `MarshalProofReceipt` refuses a receipt containing a raw secret, `live:true` or a non-synthetic type. Serialized templates are `{page}`/`{conversation}` only. |
| PH-07 | `TestProofPH07NoDefaultTransportAndNoWork`; CLI `TestCLIPoisonedCredentialAndConfigEnvironmentIsIgnored`, `…InvalidInputRejectedWithoutReceipt`, `…EntrypointAndRunAgree`, `…FixtureIsFiniteAndIndependent`, `TestProductionAdapterFilesUnchangedInGit` | Real `main()` is executed through a re-exec of the test binary. With poisoned token/page/DB/proxy/cert/API-key environment and poisoned `.env`/`config.json` in the working directory the exit code, normalized stdout and stderr are identical to a clean environment; no poisoned value surfaces and the directory is unchanged. 22 cases (unsupported live/credential/config/db/base-url/transport/download/output flags, bad or missing input, unknown scenario, positional argument) exit 2 with empty stdout, no echo of the supplied value and no created file. |
| PH-08 | `TestProofPH08FailureSemantics`, `TestProofTranscriptIsFinite` | Late malformed page, non-2xx, repeated cursor, message no-progress and missing array → INCOMPLETE with the specific class and run 2 NOT_RUN; a second-run 500 leaves run 1 PASS attributed as `run2:` reasons and overall INCOMPLETE; a second-run mismatch is FAIL; fixture ceiling and unscripted-request failure keep mutated guards finite. |

## 4. Commands and results

All Go commands were run from the project root with `GOPROXY=off GOTOOLCHAIN=local` and `go -C backend`.

| Check | Result |
| --- | --- |
| Workspace doctor (`check_cvf_workspace_agent_enforcement.ps1`) | 25/25 PASS before BUILD |
| `go test -count=1 -v ./channels/ -run TestProof` | PASS: 17 top-level tests, 58 PASS lines including subtests, 0 FAIL |
| `go test -count=1 -v ./cmd/pancake-proof/` | PASS: 7 tests (6 CLI + git-baseline), 0 FAIL |
| `go test -count=1 ./channels/ ./cmd/pancake-proof/` (complete channels package incl. existing Pancake/Facebook/Zalo tests) | PASS (`channels` 2.6 s, `cmd/pancake-proof` 3.6 s) |
| `go -C backend build ./...` | exit 0 |
| `go -C backend vet ./...` | exit 0 |
| `gofmt -l` on the four new files | no output |
| `go test -race ./channels/ ./cmd/pancake-proof/` | **NOT RUN**: `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1` (CGO disabled, no C compiler on this host); exit 2. Not enabled or worked around. |
| DB-dependent engine/backend suites | **NOT RUN** (not required; no DB fixture used). Inherited local product evidence is not reused as fresh evidence here. |
| Provider/channel/network/GitHub Actions/CVF Web bridge | **NOT RUN** (no authority; documentation-and-offline-harness tranche). |
| Docs build (`npm run docs:build`, inherited `env`-highlighter warnings) | first attempt **FAILED** on a dead link (this record linked a `CVF_SESSION` path outside `docs/`); link changed to plain text; rerun PASS (7.85 s) |
| Catalog `-Write` then `-Check` | PASS (new artifact `runtime034-proof-harness-build` registered; work-order description updated) |
| `git diff --check` | first run flagged one trailing blank line in the handoff; fixed; rerun clean (CRLF notices only) |
| Gate preflight: default, `--base origin/main --head HEAD`, explicit 15-file `--files` set | 7/7 PASS each (before the BUILD commit; re-run before the follow-up commit) |
| `python -B -m unittest discover -s scripts/tests -p "test_cvf_downstream_gate*.py"` | 46 tests OK (8.6 s) |

## 5. Request, redirect, time and side-effect observations

Every test asserts the underlying call count through `ProofTranscript.Calls()` (the transport observer) in addition to receipt fields. Totals: clean two-run fixture = 20 calls; budget N=19 = 19 calls; shared-counter N=10 = 10 calls; redirect cases = 1 call each; admission table = 0 calls for 23 denials; cancelled context = 0 calls. Time: deadline tests complete in well under 1 s (in-flight case bounded by a 2 s safety in the fixture so a broken guard fails an assertion instead of hanging). Side effects: library tests touch only memory; CLI tests use `t.TempDir()` working directories and assert them unchanged; no network, DB or file output exists in the code path.

## 6. Mutation evidence (applied, restored, rerun)

Runner: an out-of-repository script reading the exact bytes of `pancake_proof.go` (SHA-256 above), requiring exactly the stated match count, writing the mutant, running `go -C backend test -count=1 -timeout 90s -run TestProof ./channels/`, restoring bytes in `finally`, verifying restored hash equality and re-running the baseline. Baseline before and after: PASS; final restored hash equals the source hash in §1. A kill requires a failing behavioral assertion; build errors, panics and timeouts are INCONCLUSIVE.

First attempt (earlier, on the pre-strengthening test file): 16 mutants all failed tests, but three kills rested on weaker evidence — M9 (empty-inventory guard) was masked by the separate `empty_message_inventory` guard, M8 (oversize) by the unchanged adapter's own 8 MiB refusal, M6a by the sanitation scan only; and the redirect test checked the reason before the call count. I therefore added `TestProofPH04InventoryWithoutMessagesCannotPass`, asserted call counts first in the redirect test, and added variants removing the competing guard (M9b/M9c) and the scan (M6c). The full campaign below was re-run against the final test file. A first M6c attempt was **INCONCLUSIVE (build: unused range variable)**; the mutant was repaired and M6c re-run alone.

| ID | Mutation (matches) | Disposition | Tests failing / first observed assertion |
| --- | --- | --- | --- |
| M1a | host admission → `case false` (1) | KILLED | `PH02AdmissionDeniesBeforeTransport`: `alternate-host: want denial, got resp…`, `denied request reached the underlying transport` |
| M1b | drop `!t.approved[id]` path approval (1) | KILLED | same test: `unapproved-conv`/`encoded-separator` reached transport |
| M2 | remove `CheckRedirect` + 3xx rejection (2×1) | KILLED | `PH02RedirectsNeverFollowed` (3 subtests): `a redirect hop reached the transport`; `PH07…`: `proof adapter must use the injected transport and a redirect refusal` |
| M3a | reset attempts at each run (1) | KILLED | `PH01…`: `want exactly 20 physical requests, transport=20 receipt=10`; `PH03SharedBudget` N+1 / second-run / 429 cases |
| M3b | budget `>=` → `>` (1) | KILLED | `PH03SharedBudget/request-N-plus-1-never-reaches-transport`, `budget-ends-mid-traversal…`, `second-run…` |
| M4 | remove `WithTimeout` and transport deadline check (2×1) | KILLED | `PH03Deadline/in-flight-deadline` (`transport_error` instead of deadline after the 2 s fixture safety), `elapsed-before-request` (`PASS calls=20`) |
| M5a | missing conversation no longer recorded (1) | KILLED | `PH04…/missing-conversation`: `want FAIL, got PASS []`; `ExpectedButEmptyResultIsFail` |
| M5b | missing message no longer recorded (1) | KILLED | `PH04…/missing-message`: `want FAIL, got PASS []` |
| M6a | raw transport error text into request outcome (1) | KILLED | `PH06…/transport-error-with-canary`: `receipt failed its own sanitation scan` (the fail-closed scan fired) |
| M6c | M6a with the sanitation scan also disabled (2×1) | KILLED (after INCONCLUSIVE first attempt) | `PH06…/transport-error-with-canary`: `receipt leaks "tok-CANARY-7f3a91c2"`, `receipt leaks "CANARY-ERR-5521"`; `marshal-fails-closed`: raw-secret receipt not refused |
| M6b | pseudonym returns the raw value (1) | KILLED | `PH01…` request mismatch (refs no longer HMAC), `PH04…`, `PH06PseudonymIsDomainSeparatedHMAC` |
| M7 | page token comparison disabled (1) | KILLED | `PH02Admission…`: `wrong-token: want denial` |
| M8 | harness oversize bound ×100 (1) | KILLED on reason class | `PH03…/oversize-response-never-passes`: expected `response_oversize`, got `adapter_error`. The unchanged adapter still refuses the body, so PASS is prevented by two layers; this mutant demonstrates only the harness layer's classification. |
| M9a | `empty_inventory` guard removed (1) | KILLED on reason class only | `PH04EmptyInventoryCannotPass`: reason `empty_message_inventory` instead of `empty_inventory` (disposition stayed INCOMPLETE through the competing guard) |
| M9b | both empty guards removed (2×1) | KILLED | `PH04EmptyInventoryCannotPass`: `got PASS []`; `InventoryWithoutMessagesCannotPass` |
| M9c | no-message guard removed (1) | KILLED | `PH04InventoryWithoutMessagesCannotPass`: `got PASS []` |
| M10 | run 2 executes after incomplete run 1 (1) | KILLED | `PH02Redirects…`: `a redirect hop reached the transport: 2 calls`; `PH08…` NOT_RUN assertions |
| M11 | message `sent_at` comparison removed (1) | KILLED | `PH04…/zoneless-timestamp-disagreement`: `want FAIL, got PASS []`; `PH08/second-run-mismatch-is-fail` |
| M12 | attachment comparison removed (1) | KILLED | `PH04…/wrong-attachment`: `want FAIL, got PASS []` |

Required six semantic mutations are covered by M1, M2, M3, M4, M5 and M6 (every one has an applied-and-killed row with behavioral assertions). Mutated-source hashes and full logs are held outside the repository (scratch run directories `attempt1`, `attempt2`, `attempt3`) and are summarized here; they are not committed.

## 7. Limitations and open items

- `adapter_omitted` (rows the transport observed as eligible but the adapter failed to map) cannot be triggered with the unchanged adapter and these fixtures because both apply the same filters; tests show it is empty on clean runs and the field exists to expose a future adapter regression. No mutation targets it.
- `until_stable=false` → FAIL is implemented but not exercisable with the real adapter, which pins `until`; no mutation or test claims it.
- Request ordering of message fetches is by sorted conversation ID (deterministic), not provider order.
- Race detector NOT RUN (CGO unavailable). The code uses one mutex-guarded transport state and sequential adapter calls.
- Admission also requires `since` to equal the configured instant and cursors to be IDs seen earlier in the same run; real-provider cursor semantics are unverified.
- Real provider timestamp/protocol compatibility, tenant visibility, retention and quiescence remain unknown. A live run still needs a controlled page/tenant, independent inventory, capture handling and explicit credential/network authority under a new work order. The live packet remains PREPARED_NOT_DISPATCHED / EXTERNAL_INPUT_REQUIRED.
- A reviewer should verify: seed timing/identity at `d869624`, byte-identity of protected files, the PH matrix, an independent applied mutation sample (suggest M2, M3a, M4, M5a, M6c), and that the CLI has no path to a network or credential.

## 8. BUILD identity and hand-back

Exact BUILD commit: `69cf3a0981f8e1322040bc3b2427a4c47245e9f9` (parent `9ba811b`; seed `d869624cc15f55b39516a36f8937454e617dc3b3` unchanged). The four Go blobs in that commit hash to the SHA-256 values in section 1. This follow-up documentation commit records the SHA in the tranche record and moves the tranche to REVIEW_PENDING; it modifies no source. Gate preflights, catalog check, docs build, `git diff --check` and gate unit tests are re-run for the follow-up commit and recorded in the active handoff. Independent Codex review is next; no self-approval, push, merge, deployment or FREEZE.
