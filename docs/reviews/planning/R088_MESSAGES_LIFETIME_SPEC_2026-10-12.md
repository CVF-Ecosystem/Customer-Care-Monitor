# R088 SPEC — Messages request lifetime and selection correctness

Memory class: SPECIFICATION_RECORD
docType: specification
Status: ACTIVE_REFERENCE
Risk: R2
Worker: ccmai-r088-worker (implementation not started)
Reviewer: Codex /root
Author: Codex /root SPEC_AUTHOR
Date: 2026-10-12

## Outcome, source and selection decision

Fix inheritance roadmap F06 only: the Messages screen must display messages, conversation identity and evaluation belonging to its current tenant/selection lifetime. A late response must not replace the content the user has selected. This supports S1 source correctness and S5 reliable reading, not completion of the human review/assignment workflow or a whole product stage.

Source baseline: db4eb5cf20de43320ac435e7ec4d9389cb41f303; hashes and read scope are in ../probes/r088_dispatch_source_audit_2026-10-12.json. Runtime source is unchanged from the earlier bounded review. GOV002 portable package was independently accepted and published at d0ebc890d043b5f94b647c1b228e8f1660f83292, continuity db4eb5cf20de43320ac435e7ec4d9389cb41f303.

Select package B before A/C. F06 is independently reproducible with synthetic deferred API responses and existing Vue/Pinia test tooling. Auth and outbound packages require separate policy/persistence/transport contracts and fixture budgets; the R086 native prerequisite remains NOT_RUN. This is a deliberate reorder of the proposed sequence, not closure or reprioritization of the two P1 security findings. After R088, reassess A/C from the existing seven-finding ledger; no automatic successor.

The inherited value is the lifecycle/identity contract, implemented independently in Vue/Pinia. No private CVF ResultViewer code, tests or evidence is copied. Public ResultViewer at the pinned Core differs from the latest private fix and is not an implementation source for this task. Core remains read-only.

## Observable contract

| ID | Required behavior | Discriminating evidence |
| --- | --- | --- |
| UI01 | Selecting B invalidates A immediately. While B is pending, its header/count/messages/evaluation/download source must not expose A's settled data. Current loading/empty presentation may be reused; do not restyle the screen. | Start with settled A, select B without settling it; assert visible identity/content and store state, not only response completion. |
| UI02 | Message/conversation and evaluation publication is bound to captured tenant, selected conversation and a fresh request generation. A→B→A is a new A lifetime, so an old A response cannot win because ids match. | Controlled deferred responses with distinct old-A/new-A messages and evaluation, resolved in multiple orders. |
| UI03 | A superseded success, rejection, catch or finally must not alter current loading/error/evaluation/selection/list status or invoke stale scrolling/downstream evaluation work. Message and evaluation loading/errors are independently correct for the current selection. | Settle/reject old request while current request is still pending; assert current spinner/error remains correct, then settle the healthy request. |
| UI04 | Generation protection exists at the shared Pinia mutation boundary as well as the component. Direct overlapping store calls cannot overwrite current messages/currentConversation or current list/total. Cleanup by an old owner cannot invalidate a newer owner's operation. | Store-level deferred calls independent of component plus mounted component using the real store. |
| UI05 | Deselect/mobile-back, tenant-route change and unmount invalidate pending work and clear old detail identity/data consistently. Reuse of the same Pinia instance by a new mount is safe. | Reset before completion; tenant t1→t2 with the same conversation id; unmount/remount with one shared Pinia, then old completion/error. |
| UI06 | List pagination/filter/search loads capture query and tenant at issue time. Latest applicable request wins list/total/loading/error; late older list requests cannot restore prior query results. Evaluation-map status is tenant/lifetime-bound and stays honest on current failure. | Two distinct pages/queries, reversed successes and late rejection/finally; tenant map response arriving after reset/unmount. |
| UI07 | Mounted initialization and deep-link awaits stop continuation when their original lifetime is gone. Late channel/page/list/map completion must not select an old conversation or start a stale detail chain. Active deep links and messages/QC/classification tabs still work. | Unmount/tenant-switch at an awaited initialization boundary; current healthy deep-link control. |
| UI08 | Scoped cleanup handles the component's pending authenticated image fetch and existing debounce. A late image result after invalidation/unmount cannot retain a newly allocated object URL or write stale cache state; cleanup must not revoke a resource owned by a later lifetime. | Stub fetch/Blob/URL only with synthetic bytes; settle after unmount/reset, assert revoke/no retained allocation and current lifetime unaffected. |

An AbortController alone is not the correctness contract: synthetic transports may resolve despite cancellation. Identity/lifetime guards must prevent publication in that case. Internal decomposition, generation counters/tokens and cancellation strategy are the worker's choice within the allowed paths.

List/filter changes do not arbitrarily clear an otherwise valid same-tenant selected conversation unless current product behavior requires it. Explicit deselection/tenant change/unmount do clear it. Locale and tab switching within an unchanged selection preserve its valid data and do not manufacture a new selection. A repeated explicit selection issues a fresh generation. No persistent storage/schema/API/authorization change.

## Required evidence and finite budget

- Use actual Messages.vue and actual conversations store in mounted tests with the API boundary mocked; do not stub away the functions under judgment or rely only on extracted functions/regex/source-string assertions. Use separate direct-store tests for its publication boundary.
- Before editing production code, add the focused regressions and run them against the source baseline in an isolated copy/overlay. Capture named behavioral failures for stale detail publication, same-id ABA and stale list/finally where present. Missing helper APIs, setup errors, timeouts and diagnostics-only differences are not detector credit.
- Capture healthy controls. After repair run all new lifetime tests plus existing messages-screen.spec.ts and messages-business-day.spec.ts once for the final candidate; run forced vue-tsc and build once. Record exact test identities, commands, exit codes, skips and sanitized relevant failures. No mandatory case may be skipped silently.
- node_modules is absent at dispatch; Node v22.17.0 and npm10.9.2 are available. One bounded npm --prefix frontend ci --ignore-scripts using the existing lockfile and public registry.npmjs.org is admitted (maximum 10 minutes). No dependency/lock/config changes or credential inspection. If this cannot make required tests runnable, return one consolidated setup blocker; do not open a fixture/installation project.
- Tests use synthetic tenant/customer ids and mock HTTP/fetch/image/clipboard/download boundaries. Do not start the application/backend/dev server, connect to actual endpoints, read local tokens/.env/customer data, or use provider/DB/Docker/Go/SMTP.
- Repeat focused checks only for a named changed defect or test failure. No broad backend/docs/native/live/browser/full-stack campaign. Optional mutations occur only in disposable copies with match-count/diff/behavioral assertion/restoration proof; no production mutation in the live worktree.

## Source ownership and topology

Worker owns Messages.vue, conversations.ts, optional conversationRequestScope*.ts helpers in frontend/src/utils, new messages-lifetime*.spec.ts / conversations-lifetime*.spec.ts and bounded conversation-lifetime fixtures. Existing two Messages suites may gain tests or fixture corrections that are documented; do not remove/weaken existing assertions. Public package/config/lock, other UI views/stores, backend, Core, installed gates and all historical authorities/evidence are protected.

Messages.vue already has 1209 lines. Prefer small local changes; if shared lifetime logic needs extraction, the scoped helper family permits ordinary decomposition without another approval. Avoid a layout/style refactor or broad component split. Document any helper as optional implementation topology, not an extra framework deliverable.

## Stage and claim boundaries

Planning/spec acceptance is not source acceptance. All seven runtime findings remain OPEN until separately reviewed. R088 may close F06 only from the returned contract evidence. docs55, R086 native/fence, global channel coverage, accounts, admission/human workflow/pilot/release and deployment stay open/parked/unknown. No ETA/percentage or hosted CI claim from local gates.

## Checker Source Read-Ahead Block

Read actual selection/list/map/image/initialization functions, template identity/loading/download consumers and both existing Messages test harnesses; only Messages.vue consumes these store actions. Read public pinned active-status grammar, exact closeability graph columns/required gates, continuity activeWorkOrders/status rules, legacy seed first-commit/base/path checks and catalog behavior. These are bounded reads, not complete corpus evidence or proof that agents automatically inherit every rule.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY. Draft preparation before delivery; no new public export claim. A later dispatch/publication receipt must bind actual remote/SHA/artifact path. No private Core code or evidence is exported by this specification.
