# Human workflow UI language audit

Status: FINDINGS_OPEN
Risk: R1
Worker: Codex /root (ORCHESTRATOR / source audit)
Reviewer: Independent implementation review required after repair
Disposition: SOURCE_FINDINGS_ONLY_RENDERED_REPAIR_NOT_YET_RUN

Owner explicitly requires ordinary UI/UX language for end users, without Dev vocabulary. This targeted source audit covers the human intervention workflow; it is not an application-wide language acceptance or a production-source repair. Root's connected HTTP click/capture probe passes; Luna's isolated subagent browser inventory is empty, so Root owns later UI verification while Luna may implement source/native work.

## Consolidated findings and intended repair

| Surface in frontend/src/views/InterventionCases.vue | Source finding | Intended user-facing behavior |
| --- | --- | --- |
| filters computed | OPEN/ASSIGNED/WAITING_ON_DATA/RESOLVED/REJECTED appear as filter titles. | Use the existing localized state labels; wire values stay unchanged. |
| WAIT_DATA alert | Raw `detail.case.reason_code` and copy mentioning a manual M2 path. | Explain the missing-data situation and next user step; no internal reason codes or roadmap phase names. |
| message()/knownLabel() | Raw API error code and unknown raw enum are appended to displayed text. | Map known failures to useful localized instructions; safe generic fallback for unknown values. Preserve transport/status semantics and one-request write behavior. |
| original proposal | Raw kind/provider/model/snapshot digest and message IDs appear in ordinary cards. | Show a localized proposal type and quoted source content. Keep synthetic-data disclosure in plain language; keep machine diagnostics out of ordinary flow. Do not misrepresent synthetic results as live AI. |
| history details | Raw before/after JSON projections are directly printed. | Render a labeled before/after comparison of meaningful business fields, localized states/assessment, people/deadlines/outcome, with safe malformed-history fallback. Preserve original audit data unchanged. |
| queue/detail identifiers | Primary title is raw conversation UUID, version shown as `vN`. | Use a readable case reference and localized revision label where needed. Do not change routing IDs or lifetime guards; keep stable visible reference for distinguishing cases. |
| loading progress | Hardcoded English ARIA label. | Localize accessible loading text in Vietnamese/English. |

Scope for a subsequent source order should remain the intervention view, a small presentation helper only if needed, vi/en strings and focused existing/new intervention presentation tests. No backend permission/state/currentness/API/retry changes; no raw data or event history mutation. Unknown codes must not become HTML or leak through fallback text. Diagnostics stay in evidence/logs. Show source limitations honestly in ordinary terms.

## Required verification after bounded implementation

Run the focused intervention UI/API tests once and necessary cached build/type check under the next order's finite budget. Meaningful tests cover unknown API/enum fallback, localized filters/reason text, readable audit history including malformed JSON, preservation of proposal/history and case/route/session lifetime, and no action replay. Existing Dashboard baseline failures remain OPEN.

Root then uses the actual connected runtime browser, with a source-matched backend/binary and synthetic fixture, to verify the rendered human workflow and remaining B06/B08-B10 cases. An unavailable browser stops execution and is reported to the owner. Offline case B09 is NOT_RUN if no documented offline control can be exercised; never invent support or use firewall/interception workarounds. Capture exact before/after UI and narrow viewport. Static string checks alone do not establish rendered language acceptance.
