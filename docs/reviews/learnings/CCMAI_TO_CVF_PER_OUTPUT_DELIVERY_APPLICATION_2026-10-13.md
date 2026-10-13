# CCMA project application: per-output delivery evidence

Memory class: governed-project-learning
Status: ACTIVE_REFERENCE
docType: project_learning_application
Date: 2026-10-13
Parent finding: CCMA-OUTPUT-DELIVERY-TRUTH-20261013

## Source and decision

Parent source is private Core commit b4d44307c22f68340746eccd2089d121430905d3; companion docs/reference/downstream_gate_profile/CVF_PER_OUTPUT_DELIVERY_EVIDENCE_CHECKLIST.md; SHA-256 13126809f82d4e05384fb3c293db84c8ce6925540ef55766a815a772d445bda3. The copied clauses below make this guidance usable from the project without changing its pinned public Core (e9ddcbcdcac2a9df520f5eb0761dbdb609b9deba) or gate profile lock.

R091's final independent review at docs/reviews/R091_FINAL_INDEPENDENT_LOCAL_REVIEW_2026-10-13.md accepted F04 at build d121c9ecf24a869aea92264db1e01cd853dec0ec: captured per-run outputs, durable claim before send, sent outputs skipped, proven failures retried, and unknown/crash-left/persistence-failure outcomes held without automatic resend. The review records synthetic mixed/all-failed/reentry/concurrent-claim cases. The project branch was codex/cvf-portable-inheritance at cdf5537e10c7a285f0f08d51753a749eac75abb8 when this application began. Those fixes predate this Core guidance. This record adopts the review criteria for future outbound work; it does not retroactively claim R091 used this checklist or change its accepted status.

## Adopted checklist (copied from the Core companion)

Review each stable output identity separately. Record the frozen intended set, whether a durable claim preceded the effect, explicit remote acceptance, rejection or uncertainty, the committed outcome row and any checked result marker write. A function return, aggregate "sent" count or successful notification log cannot establish those facts for every output.

1. Classify explicit remote rejection as proven failure only if the transport contract supports it. Timeout, lost response, ambiguous SMTP DATA write/reply or failed outcome persistence are unknown. Do not silently classify them as unsent or retry-safe.
2. On reentry, skip outputs with a durable accepted result. Retry only outputs proven failed under the declared policy; hold unknown or crash-left sending outputs for reconciliation. A local idempotent claim does not prove an exactly-once external effect.
3. Exercise mixed output, all failed, concurrent claim, failure before effect, failure while persisting outcome and reentry. Check outbound call count, committed receipt/log rows, exact marker targets and affected-row counts.
4. Keep an accepted output's timestamp on reentry. Failed and unknown outputs must not inherit another output's marker. Bound caller lifetime through the outbound tail so a stalled transport cannot indefinitely retain ownership.

Positive review example: output A has explicit acceptance and a committed receipt/marker; output B has a distinct identity and proven failure. Reentry skips A and may retry B under policy. Negative review example: ambiguous SMTP DATA or failed outcome persistence cannot be resolved by an aggregate "sent" status; keep the output unresolved until reconciliation.

## Project use and verification

Adoption verification: the companion bytes read from the exact Core commit match the SHA-256 above; this record carries the four per-output evidence/retry clauses and the positive/negative examples. The R091 review was inspected to confirm the future-review trigger, not rerun or relabeled. Project preflight passed 7/7, catalog validation passed, and the workspace doctor passed 27/27 after the documentation update. No source or project tests were rerun.

Trigger this checklist for a future CCMA work order or review that sends to multiple outputs or claims retry after uncertain delivery. In INTAKE/SPEC, name output identities, the frozen intended set, transport acceptance/rejection/unknown conditions, and durable outcome/marker evidence. In REVIEW, assess each output and the listed failure/reentry cases against the work order and retained evidence; classify missing or ambiguous evidence as unresolved. Use R091's final review as a historical project example, not as a new checklist execution. No project code or tests were rerun for this adoption.

R091 remains REVIEWER_ACCEPTED / CLOSED_PASS_BOUNDED within its synthetic/local scope. Its review excludes real SMTP/provider, customer/account, hosted CI, browser and deployment proof; project-wide S0-S7 and FREEZE remain open. This written project guidance does not assert that CVF governed CCMA's AI runtime, guarantee exactly once delivery, or create a runtime gate.

| Boundary | State | Evidence / next move |
|---|---|---|
| Project finding/action | RESOLVED_BOUNDED | R091 final review and accepted build d121c9ecf24a869aea92264db1e01cd853dec0ec. |
| Parent value/implementation | MERGED_BOUNDED / IMPLEMENTED_BOUNDED | Parent learning record; written checklist only. |
| Return to project | RECEIVED_LOCAL_PENDING_REMOTE_READBACK | This project record embeds the checklist; parent Core receipt remains NOT_RETURNED until published project evidence is read back. |
| Project application | APPLIED | Documentation-only prospective review criteria; no retroactive R091 claim. |
| Public Core export | DEFERRED_PRIVATE_ONLY | Separate public-sync disposition; no public edit or repin in this project change. |

Claim boundary: adoption is verified as a portable project review reference. It supplies no new runtime, live-provider, production or product-release proof.
