# Worker Return - CCMAI-RUNTIME-088 Messages selection lifetime (F06)

Memory class: EVIDENCE_RECORD
docType: review_artifact
Self-declared worker-return artifact: yes
Status: COMPLETE_PENDING_REVIEW
Packet: CCMAI-RUNTIME-088
Role: IMPLEMENTATION_WORKER
Actor identity: ccmai-r088-worker
Model: claude-sonnet-5-5 (Claude Code, VS Code extension), the model the session exposes
Risk: R2
Worker: ccmai-r088-worker
Reviewer: Codex /root (independent Local reviewer)
Commit mode: WORKER_MUST_NOT_COMMIT
executionHead: 086330e1860df291ea58a56726914bb0b4d53087
Dispatch material: 5d11fd4ffd5a912dad633213defd8c1e0d3333b5; authority seed commit 45ac7a91a60f9f64239492e0db838c809ca51fca (three distinct commits)
Date: 2026-10-12

## Return Decision And What Changed For The User

COMPLETE_PENDING_REVIEW for F06 UI01-UI08 only. Nothing was committed or pushed and no other finding was touched. In the Messages screen, the header, message count, messages, evaluation tabs and download now always belong to the conversation the user currently selected. A reply that arrives late - after the user chose another conversation, chose the same one again, went back, switched tenant, left the page or came back - can no longer replace what is on screen, end a current spinner, show a stale error, start follow-up work, or leave an allocated image behind. This is synthetic UI evidence only: no real endpoint, backend, provider, browser session or customer data was used, so it proves nothing about live behavior, hosted CI, deployment, the other six findings (F01/F02 P1 security findings remain OPEN) or any product stage.

## Worker Self-Proof Fields

rootCauseClusterId: CCMAI-MESSAGES-REQUEST-LIFETIME
reworkGeneration: 0
consolidatedDefectClassSweep: COMPLETE_ALL_KNOWN_DEPENDENCIES
productionBindingEvidence: source LF sha256 Messages.vue aa874e48b44c437f..., conversations.ts a972c98506c124b9..., tests conversations-lifetime caf6cef6ef25927a..., messages-lifetime db5f1a74f23f9980..., startup 498820178a6183be..., resources bc979f3905c96214..., harness 0d1cf061bfdd0dcb... (full digests in `docs/reviews/probes/r088_messages_lifetime/final-hashes.json`); commands and results in `final-tests.json`, `final-typecheck.json`, `final-build.json`
adversarialRegressionDisposition: PASS_TARGETED_DEFECT_CLASS
successorTrancheOpened: NO
implementationAutonomyDisposition: CONTRACT_AUTHORITY_EVIDENCE_OUTCOME_ONLY
internalAgentInvocations: 0
externalAgentInvocations: 0
providerCalls: 0
backendRuns: 0
tokenAndBillUsage: NOT_AVAILABLE_WITH_REASON: not exposed to the worker
modelEffort: session parameter 10, scale not documented to the worker
terminalReadinessVerdict: READY_FOR_REVIEW

These fields describe the worker's own work. They do not prove provider identity and do not replace independent review.

## Design In One Paragraph

The shared store (`frontend/src/stores/conversations.ts`) now gives every list and detail call a generation and publishes only if that generation is still the latest; a superseded success or failure returns `false` instead of publishing or throwing, and starting a detail call hides the previous conversation at once. Each call may carry an `owner` (one Messages instance): `resetDetail(owner)` and `reset(owner)` are no-ops when a newer owner has since started work, so an old screen's cleanup cannot cancel a newer screen's request. `Messages.vue` adds a small lifetime scheme: a lifetime stamp (tenant plus generation) re-checked after every awaited step of initialization, deep link, list, status map and detail; a selection generation advanced by every explicit selection (so A -> B -> A is a new A even though the ids match); list and status-map ordering; an image epoch for authenticated image fetches; a tenant watcher that ends the old tenant's lifetime and restarts the screen (the route component is reused on a tenant change and previously kept the old tenant's selection and data); and header/count/download bound to a detail whose id equals the selection. No new dependency, file outside the seed, backend, API or style change; no optional helper file was needed.

## Acceptance Matrix

| Item | Behavior now | Principal tests (all real Messages.vue + real store, synthetic deferred API) |
| --- | --- | --- |
| UI01 | Selecting B clears A's header, count, messages, evaluation and download source immediately; B shows its list-row name while pending | messages-lifetime: "settled A, then pending B ..."; conversations-lifetime: "a new detail call hides the previous conversation immediately" |
| UI02 | Every selection has a fresh generation; old-A and new-A replies resolved in both orders, evaluation replies reversed | messages-lifetime UI02 (both orders, reversed evaluation); conversations-lifetime A -> B -> A |
| UI03 | Superseded success/rejection/finally cannot change spinner, error, evaluation, selection or start follow-up work; current failure and recovery still work | messages-lifetime UI03 group (messages and evaluation variants, id-mismatch, healthy controls) |
| UI04 | Generation protection at the store boundary plus the component; old-owner cleanup cannot invalidate newer work | conversations-lifetime (direct store, overlapping detail/list, owner-scoped cleanup) and mounted real-store tests |
| UI05 | Back/deselect, tenant t1 -> t2 with the same id, unmount/remount on one Pinia, including late success and error | messages-lifetime UI05 group |
| UI06 | Latest list request wins list/total/busy/error; status map is tenant/lifetime bound and honest on current failure | messages-lifetime-startup list and status-map groups |
| UI07 | Initialization and deep link stop when their lifetime ended; healthy deep link, other-page deep link and tabs still work | messages-lifetime-startup UI07 group, including t1 -> t2 -> t1 |
| UI08 | A finished lifetime allocates no object URL, later lifetimes load shared URLs themselves, active resources survive earlier cleanup, tenant change and unmount release | messages-lifetime-resources |

## Source Sensitivity Before The Production Edit

Isolated overlay of the unchanged source (runner `r088_overlay_runner.py`, sources restored from `git show HEAD`, hashes equal to the dispatch audit, `node_modules` junctioned to the locked ignored install, live tree never written). The final test set run on that unchanged source: 76 tests, 34 pass (healthy controls plus the 14 existing Messages tests) and 42 fail. Detector credit is claimed only for failures that are named behavioral assertions: 34 of the 42 (`baseline-classification.json`). Earned no credit and reported separately: 3 tests that call functions the old store does not have (`resetDetail`/`reset` missing), 1 return-value contract test, 2 failures that are a TypeError on a request the old code never issued, and 2 that stop at a harness lookup because the old screen never reloads the new tenant's list. Examples of credited assertions: "header must not show A while B is pending", "old A must not publish while new A is pending", "late older list must not publish", "stale catch must not become the list error", "an allocation after teardown would leak", "a dead lifetime must not continue its initialization". Four earlier baseline runs of earlier test versions are preserved (`baseline-unchanged-source-run1/2/3.json`; run1 35 fail / 34 pass of 69).

## Mutation Proof (optional, disposable overlays only)

Final run `mutation-results.json`: 24 behavior-changing one-match mutations (exact match counts, before/after digests, restoration by discarding the overlay), 21 KILLED by a named failing assertion, 3 SURVIVED. The survivors are classified, not hidden: M16 (stale list catch sets the list error), M21 (stale messages catch sets the error) and M22 (deselect does not advance the selection generation) change a component guard that the store boundary already makes unreachable (the store swallows superseded failures and `resetDetail` supersedes the same requests), so they are defense in depth and observationally equivalent to the baseline of the store, not killed. Two earlier mutation runs are preserved: run 1 (20 mutations) had 5 survivors that revealed missing tests (stale-success evaluation spinner, other-page deep link after unmount, response body still downloading, id-mismatched detail, store-id header guard), run 2 (24) had 5 survivors that led to the round-trip tenant and stale-failed-fetch tests. No mutation was applied in the live worktree.

## Final Candidate Verification (live worktree, once)

| Check | Command | Result |
| --- | --- | --- |
| Locked dependency setup | `npm --prefix frontend ci --ignore-scripts` (once, 13:38:35Z to 13:38:44Z) | exit 0, 209 packages from the existing lock; package.json and lock byte-identical; `npm audit` reported 17 vulnerabilities (3 moderate, 14 high) and nothing was changed in response |
| Focused UI campaign | `npm --prefix frontend test -- <4 new lifetime specs> src/__tests__/messages-screen.spec.ts src/__tests__/messages-business-day.spec.ts` | exit 0, 76 passed, 0 failed, 0 skipped or todo (conversations-lifetime 14, messages-lifetime 21, startup 16, resources 11, messages-screen 10, messages-business-day 4), 25.2 s |
| Forced type-check | frontend cwd, `npx --no-install vue-tsc -b --force` | exit 0, 15.9 s |
| Build | `npm --prefix frontend run build` | exit 0 (vue-tsc -b then vite build, 19.1 s); `dist` is git-ignored |
| Protected paths | status of package/lock/config/backend/.cvf/scripts/.github/authority seed | clean; seed equals its single first commit; frontend changes only inside the seed; catalog changes are only registrations of this return's own evidence plus three stale descriptions |

Preserved failed or partial invocations: the first forced `vue-tsc` after the tests were written failed with TS6133 (an unused `beforeEach` import in the startup spec), fixed and rerun; baseline runs 1-3 and candidate iterations 1-3 are earlier test versions (`candidate-iter1/2/3.json`: 70, 74, 76 passed); mutation runs 1-2 are earlier. Governance commands and results are listed under "Governance And Gates" below.

## Decision Points, Behavior Changes And Residual Risk

- Tenant change now restarts the screen: the old code had no tenant watcher, so the selection and list of the old tenant stayed. Page returns to 1 and the tenant-bound channel filter is cleared; other filters and the search text are kept.
- While a conversation loads, the QC and classification tabs show the loading spinner (previously the empty text, which was misleading) and the header uses the list row's name. The header and count use the store detail only when its `id` equals the selected id; the backend handler returns `conv.ID` as `id`, the same value the list returns, so legitimate data is unaffected.
- A repeated explicit selection of the same conversation issues a fresh generation; tab and locale switches keep valid data; list/search changes do not clear a same-tenant selection (tested).
- Not covered by UI01-UI08 and left unchanged: the channel store has no generation protection (a slow channel list of the old tenant could still fill the filter options), `doExport` and `shareConversation` use their own flows, and two Messages instances sharing one store are only supported through the owner rules above. Image blobs for a selection are kept until reset, tenant change or unmount (the earlier behavior), not revoked per selection.
- Image revocation of a still-displayed `<img>` after a tenant change relies on the selection being cleared first; the lightbox source is cleared with the cache.
- Not proven: any real browser, scroll behavior, mobile layout beyond the synthetic width used, real network ordering, or the other six findings.

## Governance And Gates

- BUILD acknowledgment recorded in the active handoff before any source edit or dependency setup (20:38:22+07:00 on HEAD 086330e); state, memory marker, handoff, bootstrap model, implementation status, tranche record (opted in, `workOrderStatus` ACTIVE) and work order moved to BUILD/ACTIVE together; legacy preflight 7/7 and the explicit-packet pre-dispatch gate (APPL actually CHECKED the packet) passed before the first edit.
- Return state: every surface is REVIEW / REVIEW_PENDING, the work order displays COMPLETE_PENDING_REVIEW and `activeWorkOrders` is empty. Gate results at return are in `final-gates.json`; N/A rows there mean the contract was not checked for that packet.
- Core ADIF is not applicable (no Core mutation and no downstream ADIF registry); this disclosure stands in its place.
- Candidate paths and hashes: `docs/reviews/probes/r088_messages_lifetime/candidate-manifest.json`. Source and test files changed: `frontend/src/views/Messages.vue`, `frontend/src/stores/conversations.ts`, four new specs and `frontend/src/__tests__/fixtures/conversation-lifetime/harness.ts`. The two existing Messages suites were not edited.
- No shared learning record was written; candidate lessons for Local: (1) an id comparison cannot protect A -> B -> A, use a per-request generation; (2) put the generation at the shared store boundary and give cleanup an owner, otherwise an old owner's cleanup cancels a newer owner's request; (3) component guards made redundant by the store boundary survive mutation as equivalent - say so instead of forcing a brittle test; (4) a stub that cannot return a pending response body hides the second half of an async resource race.

## Return-Time Closeability Recheck

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
outsideAuthorityBlockers: NONE
nextRepairRoute: INDEPENDENT_LOCAL_REVIEW_OF_RETURNED_EVIDENCE
workerRedispatchAllowed: NO

## Public Export Disposition

DEFERRED_PRIVATE_ONLY. The worker exported nothing and pushed nothing; the project branch holds only the dispatch material. Local owns review, commits and publication.

## Agent Operation Trace Block

| Field | Value |
| --- | --- |
| Actor | ccmai-r088-worker |
| Phase transitions | WORK_ORDER to BUILD (acknowledged before edits), BUILD to REVIEW_PENDING |
| Manifest delta | NONE: every path is enumerated in the candidate manifest |
| Commit | NONE |
