# Learning — Root must settle execution readiness before dispatch

Status: RECORDED_AND_APPLIED_TO_DRAFT_EFFECTIVENESS_OPEN
Source project: Customer-Care-Monitor-AI
Source revision: 64238c411b2fd9e966ef90ec73a243c8b355305e
Updated: 2026-10-16
Owner: Codex /root ORCHESTRATOR

## Trigger / Observation

Read before preparing or assigning any browser/UI evidence job, after a prerequisite failure, and before promoting a draft to DISPATCH_READY. Owner reported two independent agents failed to deliver the requested UI rerun. Root independently inspected both returns: Codex stopped on denied file:// navigation; Claude reported blank-page bridge success but correctly stopped before R101 because executable UI authority was missing. B01-B10 were NOT_RUN, not product test failures.

Root sent a chat prompt before preparing the current authority, supported tool route and source/build/fixture binding. Asking the worker to discover predictable missing authority transferred dispatcher work and operator wait to the worker. A successful blocked return is honest evidence, but does not make the original dispatch effective. This is failure to apply existing pre-dispatch/role controls, not evidence that more governance stages or another blind retry are needed.

## Source / Evidence

- Owner-reported repeated dispatch failure in the current session.
- Source-inspected returns: ../R101_BRIDGE_RECHECK_WORKER_RETURN_2026-10-16.md and ../R104_BRIDGE_RECHECK_WORKER_RETURN_2026-10-16.md.
- Independent review: ../R104_BRIDGE_RECHECK_INDEPENDENT_REVIEW_2026-10-16.md, D01-D06.
- Independent artifact check: ../probes/r104_root_review/artifact_verification.json;13/13 hashes match, final CLICKED PNG visually inspected. Final image alone does not authenticate a click transition/tool provenance; R104 raw tool/recording export remains missing.
- Root read-only preparation: ../probes/r105_dispatch_preparation/readiness_inventory.json. Binary hash matches; all17 historical source entries verified against commit200df3e including line-ending representations. Current tree matches14/17; router/model/mysql differ. Old protection baseline lacks the subsequently added R102 container. Fixture remains stopped. No runtime was launched.

## Project Action / Verification

Root must complete the following evidence-backed checklist before dispatch; each item needs an actual path/hash/result, accountable owner and explicit PASS or HOLD. An unknown is HOLD. A DRAFT/static gate PASS is never dispatch admission.

1. Product scope and claim: historical or current-source acceptance is selected explicitly. Do not mix old binary with a current-backend claim.
2. Authority: effective packet, dispatcher seed in committed history, baseCommit containing that seed, record/worker/tool/risk/allowed effects/paths/budget all agree; Root owns activation. A report label or nextAllowedMove does not create a tranche.
3. Tool route: tool-specific supported scheme/control/capture is established with session evidence; do not infer Codex CUA capability from Antigravity, use denied file:// routes, or install/retry another stack to evade a policy denial.
4. Inputs: actual binary, historical source mapping, current frontend/config/dependency hashes, runner source and output paths are verified. Read the runner's guards; do not tell a worker to invoke a runner already known to reject the chosen source.
5. Resources: exact fixture/container/image/volume/network identities, ownership/exclusivity and protection inventory are checked before launch. Old ports, PIDs and DB data are not readiness proof. Compare historical baseline drift explicitly; never overwrite historical evidence to force PASS.
6. Budget and lifecycle: finite campaign/start/action/time budgets, append-only attempt reservation before effects, no inherited counter reset, cleanup only for proven owned processes/resources. Distinguish static preparation from checks necessarily performed after start.
7. Evidence: raw tool calls/responses or accessible recording, before/after observations, original screenshots, command/stdout/stderr/exit receipts and byte hashes. Separate reported, source-inspected and independently observed facts.
8. Gates: explicit packets, read individual control lines, document NOT_APPLICABLE. A draft or absent packet cannot satisfy active closeability. Root issues one consolidated repair list before another handoff.

Local checks: legacy preflight7/7 PASS and reviewer-fast7 mandatory controls PASS; APPL explicitly NOT_APPLICABLE because this is a planning draft, so active closeability remains unchecked. Initial continuity validation failed for missing full active tranche ID in nextAllowedMove, then all pointers were corrected; validation_history.json preserves that finding. No runtime retry.

Applied now: repaired draft packet and candidate authority are linked below; obsolete handoff is marked superseded, canonical next move requires Root preparation rather than another blind browser retry. The draft remains NOT_DISPATCHABLE while activation/runner/session-evidence holds remain. This is a document-level preventive control, not a new machine-enforced rule. Verify effectiveness on the next genuinely ready dispatch; do not claim recurrence is impossible.

## Parent Value / Dedup

Related local record: feedback_orchestrator_roadmap_control.md. This record adds the concrete dispatch-readiness failure mechanism, tool-surface distinction and historical/current-source binding. Proposal for CVF parent: a dispatcher-owned evidence checklist that cannot treat DRAFT/N/A as executable readiness. Prepared locally only; no message sent and no Core change, parent acceptance or new control identifier claimed.

## Feedback State And Links

| Boundary | State | Evidence / owner / next move |
| --- | --- | --- |
| Project finding/action | DOCUMENT_REPAIR_APPLIED_EXECUTION_OPEN | Root; corrected draft and readiness audit |
| Parent value decision | PROPOSED_NOT_SENT | Root; parent may assess under separate scope |
| Parent implementation | NOT_IMPLEMENTED | No Core/tooling change |
| Return to source project | NOT_RETURNED | No parent receipt |
| Source project adoption | APPLIED_TO_DRAFT_EFFECTIVENESS_OPEN | Next dispatch must satisfy all eight checks |

## Claim Boundary

No browser replay, DB start/query, provider call, governance proof, product test, commit or FREEZE in this repair. R101 B01-B10 and three Dashboard failures remain open. Repaired work-order draft: ../planning/R105_R101_UI_EVIDENCE_WORK_ORDER_DRAFT_2026-10-16.md.

## Supplement application update — 2026-10-16

Root independently accepted recorded data-URL diagnostic provenance in ../R104_BRIDGE_SUPPLEMENT_INDEPENDENT_REVIEW_2026-10-16.md; existing export/recording HOLD is satisfied without replay. Supplement exposed an inaccurate prior tab-closed claim: no close call exists and worker metadata showed ACTIVE. Always distinguish task completion from resource cleanup; require actual close result/post-close observation before claiming closure. Raw gate history and manifest self-hash prose also need source checks; reviewer recorded factual qualifications without rewriting evidence. Runner/config/authority holds remain Root-owned; next-dispatch effectiveness remains OPEN.

## Source-preparation application — 2026-10-16

Root prepared/reviewed actual runner/config before another UI dispatch: docs/reviews/R105_RUNNER_INDEPENDENT_SOURCE_REVIEW_2026-10-16.md. Independent12 offline tests plus137frontend/17historical source hashes and read-only Docker binding/format PASS. New concrete lessons: compare seed history with explicit LF handling; bind Docker local pipe and empty config rather than ambient credential context; preserve child handle ownership even when post-spawn receipt writing fails; surface log-reader failures instead of reporting cleanup success. Old tests/counters/evidence remain unchanged. Source readiness is settled, but runtime and effective immutable authority remain absent. This applies the checklist in preparation; effectiveness of an actual ready UI dispatch remains OPEN.

## Worker-transfer capability check — 2026-10-16

Owner authorized Codex/subagent execution after external-Claude dispatch preparation. Root checked the new execution surface before changing immutable authority or consuming runtime: actual CUA inventory had no browsers; documented iab creation returned Browser is not available: iab. Sol Medium assessed the blocked transfer and named the runner bindings that a future successor would need. Root suspended the order and preserved all campaign counters. A model change is not evidence of browser availability; prior external Antigravity data-URL evidence cannot establish current Codex HTTP capability. This check prevented a blind runtime dispatch; actual UI-delivery effectiveness remains OPEN. Receipts: ../probes/r105_root/CODEX_BROWSER_CAPABILITY_2026-10-16.json and ../probes/r105_root/CODEX_WORKER_CAPABILITY_RETURN_2026-10-16.md.
