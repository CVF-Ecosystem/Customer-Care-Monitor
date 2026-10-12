# Worker Return - CCMAI-RUNTIME-088 round1 repair of R088-R1-F01

Memory class: EVIDENCE_RECORD
docType: review_artifact
Self-declared worker-return artifact: yes
Status: COMPLETE_PENDING_REVIEW
Packet: CCMAI-RUNTIME-088
Role: IMPLEMENTATION_WORKER / REPAIR_WORKER
Actor identity: ccmai-r088-worker
Model: claude-sonnet-5-5 (Claude Code, VS Code extension), the model the session exposes
Risk: R2
Worker: ccmai-r088-worker
Reviewer: Codex /root (independent Local reviewer)
Commit mode: WORKER_MUST_NOT_COMMIT
executionHead: 086330e1860df291ea58a56726914bb0b4d53087
Dispatch material: 5d11fd4ffd5a912dad633213defd8c1e0d3333b5; authority seed commit 45ac7a91a60f9f64239492e0db838c809ca51fca (three distinct commits)
Date: 2026-10-13

## Return Decision And What Changed For The User

COMPLETE_PENDING_REVIEW for the single consolidated finding R088-R1-F01 only. Nothing was committed or pushed and no other finding was touched. In the Messages screen, a conversation link that is still being looked up in the background (for example `?conv=...` from another page) is no longer allowed to override what the user did in the meantime. If the user opens another conversation, opens the linked conversation themselves, or goes back to the list while the link lookup is pending, the late lookup (success, failure, or the extra page reload for a link on another page) now stops: it no longer changes the page, no longer reopens the old conversation, and no longer sends a detail or evaluation request for it. A link nobody interrupted still opens its conversation on the requested tab, on the current page and on another page. This is synthetic UI evidence only: no real endpoint, backend, provider, browser session or customer data was used, so it proves nothing about live behavior, hosted CI, deployment, the global channel store, the other six findings (F01/F02 P1 security findings remain OPEN) or any product stage.

## Worker Self-Proof Fields

rootCauseClusterId: CCMAI-MESSAGES-REQUEST-LIFETIME
reworkGeneration: 1
priorFindingSetDigest: 3129726740a0c7a9d0569207a4646e90812ee02c8d7d24fb7588213231e2dbe1
consolidatedDefectClassSweep: COMPLETE_ALL_KNOWN_DEPENDENCIES
productionBindingEvidence: source LF sha256 Messages.vue ec4222235730892e001f238b4883407b6e3e46ee8fbe7440c78e03eba5316b6c (round0 candidate aa874e48b44c437f50e73f19872a15c2121aed1b2e32dff03785b881676c3888), startup spec 8921c792fcd56d08b817b50aa094e071f481c688d2ce8e95edde434f59829c72 (round0 498820178a6183be22e82d6b37ebf62d93470eeaba19e5eb742f823ade7c83a7); conversations.ts, the other three lifetime specs and the harness are byte-identical to round0 (full digests in the round1 hashes receipt); commands and results in the round1 oldsource, tests, typecheck and build receipts
adversarialRegressionDisposition: PASS_TARGETED_DEFECT_CLASS
successorTrancheOpened: NO
implementationAutonomyDisposition: CONTRACT_AUTHORITY_EVIDENCE_OUTCOME_ONLY
internalAgentInvocations: 0
externalAgentInvocations: 0
providerCalls: 0
backendRuns: 0
tokenAndBillUsage: NOT_AVAILABLE_WITH_REASON: not exposed to the worker
modelEffort: session parameter not exposed numerically to this session, scale unknown
terminalReadinessVerdict: READY_FOR_REVIEW

These fields describe the worker's own work. They do not prove provider identity and do not replace independent review.

## Review-Dispatch Convergence Echo

returnedDispatchKind: REWORK
dispatchSurface: INTERNAL_AGENT
parentAssignmentId: CCMAI-RUNTIME-088
reviewRoundCount: 1
dependencyAuditDisposition: COMPLETE_BEFORE_FIRST_REPAIR
reworkFindingDisposition: CONSOLIDATED_ALL_DEPENDENT_FINDINGS
newIndependentCriticalEvidence: NONE
regressionGuardDisposition: REQUIRED_AND_PLANNED_FOR_EACH_TARGETED_DEFECT
nextDispatchDisposition: ONE_CONSOLIDATED_REWORK
workerRepairTurnCount: 1
successorTrancheOpened: NO

## Root Cause And Repair

Root cause (as Local found, confirmed here): `initialize()` re-checked only the tenant/mount lifetime after each await. An explicit selection or mobile back advances the selection generation, but that is a different counter, so the older automatic deep-link continuation still passed its check and called `selectConversation` for the old link (or reloaded the link's page first).

Repair, one place in `frontend/src/views/Messages.vue`: `initialize()` records the selection generation as `intent` when it starts and defines `linkCurrent()` as "tenant/mount lifetime alive and no explicit selection or deselect since". The deep-link block is entered only while `linkCurrent()`, and re-checks it after the page-lookup response, after the other-page list reload, and after the lookup `catch` fallback, each before it changes the page, selects or fetches. Tenant change and unmount are unchanged: they still end the lifetime, and the new tenant's `initialize()` starts a new `intent`, so its own deep link is a fresh automatic selection. List/status-map initialization is unchanged. No store, channels, dependency, config, Core, gate or backend edit; the two existing Messages suites were not edited.

Defect-class sweep done before the edit (all awaited boundaries of automatic work that can select): page lookup success, page lookup rejection, other-page reload (explicit selection or back during the reload), selection made before the lookup is even issued, selecting the deep-linked conversation itself, tenant change. The remaining awaited steps in `initialize()` (channels, list, status map) never select and are already tenant/mount guarded. Left unchanged and not treated as selection intent: a manual page, filter or search change made while the lookup is pending (it does not select or deselect; the link still reloads its own page as before).

## Acceptance Matrix (R088-R1-F01)

| Boundary | Behavior now | Test (real Messages.vue + real Pinia store, synthetic deferred API) in messages-lifetime-startup |
| --- | --- | --- |
| Healthy untouched link | Selects the linked conversation on its tab | healthy control: an untouched pending deep link still selects A on the requested tab |
| Healthy lookup failure | Falls back to the current page and still selects | healthy control: a page lookup failure with no user action falls back ... |
| Page success + select B | B header, messages kept; no A messages or evaluation request | page success: selecting B while the lookup is pending ... |
| Page success + select A itself | One A detail only, no second A lifetime | page success: explicitly selecting the deep-linked conversation itself ... |
| Page success + mobile back | Stays on the list; no A request | page success: mobile back while the lookup is pending ... |
| Page rejection + select B / back | B kept / stays deselected; fallback does not select A | page rejection: selecting B ...; page rejection: mobile back ... |
| Other page + select B first | No page-2 list reload, B kept, no A request | other-page lookup: selecting B first ... |
| Other-page reload pending + select B / back | After the reload settles B kept / deselected, no A request | other-page reload pending: selecting B ...; ... mobile back ... |
| Selection before lookup issued | No stale A detail or evaluation | selection made before the lookup is even issued ... |
| Tenant change after a selection | New tenant's own link still runs and selects | a tenant change still starts the new tenant's deep link ... |
| Existing healthy links | Current-page and other-page links unchanged | existing "healthy control" tests in the same spec, unchanged and passing |

## Source Sensitivity Before The Production Edit

The 12 new tests were written first and run on the unchanged round0 candidate (Messages.vue LF sha256 aa874e48..., recorded in each receipt) in the live tree before the edit, with the production file untouched. Two invocations are preserved: the first oldsource receipt (28 tests, 19 pass, 9 fail) contained a harness defect of mine in the reload test (it expected exactly two list requests, but the page watcher also issues a request, so that test failed at its setup line; not credited), and was corrected; the second oldsource receipt (28 tests, 19 pass, 9 fail) fails every one of the 9 behavioral tests at its named behavioral assertion: header stays on the explicit B ("expected 'Khách A' to contain 'Khách B'"), no second A lifetime ("expected 2 to be 1"), back must end the pending selection ("expected true to be false"), the stale link must not reload the list ("expected 3 to be 1"), and the B/back variants after the reload. Healthy controls pass on the old source: untouched link, lookup failure fallback, tenant change. Detector credit: 9 of 9 new behavioral tests in the second invocation; none from the first invocation's setup failure. Local's own mounted probe (round1 Local review) matches the same two behaviors. No mutation campaign was run (not required for this round); no source was mutated.

## Final Candidate Verification (live worktree)

| Check | Command | Result |
| --- | --- | --- |
| Focused UI campaign | `npm --prefix frontend test -- <4 lifetime specs> src/__tests__/messages-screen.spec.ts src/__tests__/messages-business-day.spec.ts` | exit 0, 88 passed, 0 failed, 0 skipped or todo (conversations-lifetime 14, messages-lifetime 21, startup 28, resources 11, messages-screen 10, messages-business-day 4), about 18 s |
| Forced type-check | frontend cwd, `npx --no-install vue-tsc -b --force` | exit 0, about 21 s |
| Build | `npm --prefix frontend run build` | exit 0, about 21 s; `dist` is git-ignored |
| Dependency setup | none; the installed locked modules from round0 were reused | package.json and lock byte-identical (hashes in the round1 hashes receipt) |
| Protected paths and round0 | package.json, lock, channels store, vite config, the two existing Messages suites, authority seed; 21 round0 evidence and return files | git status clean for the protected paths; seed equals its single first committed content; all 21 round0 files byte-identical to the round0 candidate manifest (round1 protected receipt) |

Originally 76 tests passed in round0; the 76 original tests are all still passing with 12 added. Preserved failed or partial invocations: the first oldsource invocation described above. A PowerShell attempt to pass the JSON reporter flags was dropped by the shell and produced no receipt; the runner script was used instead and no source or test changed between them.

## Decision Points And Residual Risk

- A selection made before the lookup is issued now skips the lookup entirely, so no `/page` request is sent for a link the user already superseded.
- A manual page, filter or search change during a pending lookup is not newer selection intent (see sweep).
- Unchanged and still OPEN, as Local disclosed: the global channel store has no generation protection (a slow channel list of the old tenant can still fill the filter options); the 17 reported dependency vulnerabilities are not repaired; docs55, R086 native/fence and the other six findings stay OPEN. Not proven: any real browser, scroll behavior, mobile layout beyond the synthetic width, real network ordering.

## Governance And Gates

- BUILD acknowledgment for round1 recorded in the active handoff (section Round1 Worker BUILD Acknowledgment, 2026-10-13T03:38:20+07:00, HEAD 086330e) before any source or test edit; state, memory marker, handoff, bootstrap model, implementation status, tranche record (`workOrderStatus` ACTIVE) and work order moved to BUILD/ACTIVE together by one prevalidated sync; legacy preflight 7/7 and the explicit-packet pre-dispatch gate (APPL and ROUTE actually CHECKED the ACTIVE packet) passed before the first edit.
- Return state: every surface is REVIEW / REVIEW_PENDING, the work order displays COMPLETE_PENDING_REVIEW and `activeWorkOrders` is empty. Gate results at return are in the round1 gates receipt; N/A rows there mean the contract was not checked for that packet.
- Core ADIF is not applicable (no Core mutation and no downstream ADIF registry); this disclosure stands in its place.
- Changed in round1: `frontend/src/views/Messages.vue`, `frontend/src/__tests__/messages-lifetime-startup.spec.ts`, continuity surfaces, this return and the round1 evidence directory. Round0 return and evidence are untouched.
- No shared learning record was written. Candidate lesson for Local: an explicit user action and an automatic continuation need the same generation to be compared, not only the lifetime that started them; one counter for tenant/mount and one for selection is easy to get half right.

## Return-Time Closeability Recheck

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
outsideAuthorityBlockers: NONE
nextRepairRoute: INDEPENDENT_LOCAL_REVIEW_OF_RETURNED_EVIDENCE
workerRedispatchAllowed: NO

## Public Export Disposition

DEFERRED_PRIVATE_ONLY. The worker exported nothing and pushed nothing. Local owns review, commits and publication.

## Agent Operation Trace Block

| Field | Value |
| --- | --- |
| Actor | ccmai-r088-worker |
| Phase transitions | CHANGES_REQUIRED to BUILD (acknowledged before edits), BUILD to REVIEW_PENDING |
| Manifest delta | Messages.vue and the startup spec plus round1 evidence; enumerated in the round1 hashes receipt |
| Commit | NONE |
