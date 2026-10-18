# M3 local synthetic human workflow

Status: ACCEPTED_INTENDED_CONTRACT
Date: 2026-10-15
Tranche: CCMAI-RUNTIME-101

## Outcome and boundary

An authorized reviewer can open a source-bound case in the existing application, confirm/reject/correct its assessment, assign a valid tenant member and deadline, record an outcome, and reopen without erasing history. Root ORCHESTRATOR/SPEC_AUTHOR accepts this intended contract after Luna Max inventory and Sol Medium transactional design. Implementation and acceptance remain unproved until independent REVIEW.

Roadmap M3/S5, following R100 bounded real M2 proof. R100 source/receipt/counters are immutable; unused calls are unavailable. R101 tests the human software workflow using explicitly synthetic, directly seeded snapshots/proposals, without any provider or mock provider. It must not claim live AI governance or a continuous live M2-to-M3 demonstration. Full M2 and M1 people/SLA/corpus/threshold decisions remain open; real business roles and SLA are not inferred from fixture users/deadlines. Facebook/Zalo OA remain parked. No installed activation, outbound delivery, pilot, publication or deployment.

## State and durable action contract

Keep machine eligibility/execution/proposal identity separate from human assessment and work state. Add an append-only human event table and additive case projection columns. Preserve original proposal revisions/evidence and machine receipts. Correction stores human explanation/next action separately, labelled human; no model confidence or invented evidence. Use separate execution assignee and WAIT_DATA owner. Every action requires a bounded nonempty reason, positive expected case version and exact snapshot identity/digest; assessment actions also bind current proposal identity/revision. Server supplies actor/time/tenant.

CONFIRM/CORRECT require PENDING_REVIEW and current complete source; resulting human assessment is confirmed/corrected with work still OPEN. REJECT records rejected assessment and terminal REJECTED work state. ASSIGN after confirmed/corrected sets a real same-tenant member, future RFC3339 deadline and ASSIGNED state; reassignment/deadline changes create events. RESOLVE requires accepted assessment, assigned work and a bounded nonempty outcome; accepted assessment alone never resolves work. REOPEN from RESOLVED returns ASSIGNED and requires a future deadline, retaining prior outcome history. No notification is sent and notification state cannot decide business state.

WAITING_ON_DATA supports ASSIGN_DATA with same-tenant owner, future deadline, reason and next action; ESCALATE_DATA records reason/next action while remaining waiting. It cannot be confirmed/resolved or silently marked complete. Display the source supplement/escalation route and explicit manual M2 rerun dependency; no automatic rerun or AI call. Full WAIT_DATA refresh/rerun acceptance remains open if not delivered. Rerun cannot overwrite any human-owned case: extend the existing pending-review guard using human history/projection, including assigned WAIT_DATA and terminal cases.

POST actions must freshly verify known role plus intervention_cases:r and :w inside the transaction; owner/admin semantics retained, members need explicit grants, revoked/unknown/malformed rights fail. Resolve/reopen restricted to authorized reviewers under this resource, not merely assignee identity. Assignee selection returns minimal member id/name and validates current same-tenant membership; invalid/cross-tenant assignment is rejected without writes.

Use one MySQL transaction for current authority, source validation, case version CAS and event insert. Lock the channel using supported sync-writer lock order, conversation/source and case; ensure concurrent insert/update/delete via supported application writers cannot invalidate source during accepted action. Verify snapshot provenance and case/proposal/run/snapshot identities plus full local-message comparison. Unknown/corrupt/stale/partial source blocks assessment actions with a safe conflict/unavailable response. Assignment/outcome on accepted cases also recheck source; data ownership/escalation may proceed with explicit stale/unknown status and must never certify current source. Define the actual supported-writer limitation honestly; arbitrary out-of-band SQL and upstream channel completeness are not proven.

Every accepted action atomically increments case version and appends actor/time/action/reason, before/after projection and immutable proposal/snapshot references. Failed event insertion, CAS loss, source read error, permissions or validation produces no partial case/event write. Duplicate submit/replay and two concurrent reviewers cannot produce two accepted events for the same expected version. Conflict never automatically replays a user action. Human history has no update/delete API.

## API and UI

Preserve existing tenant paths; add POST /intervention-cases/:caseId/actions and a bounded minimal assignee lookup. GET detail adds ordered human history, original proposal/evidence and explicit local source integrity status. GET queue has documented filters, deterministic order, cursor, page size and honest counts/has-more; include pending, assigned and WAIT_DATA, with resolved/rejected discoverable. Queue urgency/deadline must be understandable; do not call a UUID order a priority ranking.

Add a route/menu and Vietnamese/English labels in the existing UI style, plus member permission grant controls. Display immutable AI proposal separately from human corrections, exact evidence and source status; show current human state, assignee/data owner, deadline, overdue and complete history. No raw HTML rendering. Require reason before writes, prevent double-click, clear forms after success and require reload/review after conflict. Async requests are scoped to route tenant/case, session and permission lifetime; old detail/action/list/assignee responses cannot update a new context. Clear data/actions at navigation/logout/revocation; failure is visible and never counted as success. Do not depend on channels store for this workflow.

## Acceptance and finite route

| ID | Required local software evidence |
| --- | --- |
| H01 | Real authenticated backend + MySQL confirm/reject/correct, original proposal unchanged, actor/time/reason/before-after persisted |
| H02 | Assign/reassign/deadline/resolve/reopen lifecycle; outcome and prior history survive reload |
| H03 | Missing/read-only/unknown/revoked roles, cross-tenant and invalid assignee rejected with no writes |
| H04 | Stale version/proposal/source changed/deleted/appended/corrupt, concurrent reviewer and duplicate submit negative cases |
| H05 | Transaction/event insert failure rolls back projection/version; rerun guard protects human-owned case |
| H06 | WAIT_DATA owner/deadline/escalation and overdue display without completion or provider effects |
| H07 | UI route/session/tenant lifetime, denied/error/conflict, double-click and unchanged existing UI checks |
| H08 | Actual browser with real backend/MySQL: happy workflow, correction/history, persistence reload, denied/stale/failure; Vietnamese desktop and one narrow viewport; captures tied to source |

At most 16 Go/build/binary invocations (worker8/root6/reserve2), two new cached MySQL fixtures, finite node checks per seed. Source audit first, then one integrated local campaign rather than helper successors. Root owns independent REVIEW and commits. Two consolidated same-scope repairs permitted within counters; third without independent new cause stops for REVIEW_COST_ESCALATION_REQUIRED. Protect original app/DB, record counters and failures and clean only owned labelled resources.

Acceptance may close only implemented and evidenced rows. Source/unit, actual MySQL, browser, historical real-provider and deployed evidence are distinct. Missing browser or negative proof keeps bounded M3 acceptance open. M3 full exit additionally requires roadmap M3-01..08, including any explicitly deferred WAIT_DATA rerun integration; no blanket stage closure.
