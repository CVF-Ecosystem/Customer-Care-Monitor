# R120 read-only lifetime review and transfer decision

Status: CHANGES_REQUIRED
Phase: REVIEW
Risk: R2
Worker: /root/r120_lifetime_audit_luna
Reviewer: Codex /root
Scope: Existing source/evidence diagnosis only; worker Luna Max read-only, Root independent review and record synchronization.

## Decision

Preserve the accepted bounded synthetic current-proposal/history observations. R120 remains CHANGES_REQUIRED / FREEZE OPEN. Existing files cannot supply the missing original terminal, held-child wait/exit, or final durable accounting. Actual process disappearance cause remains UNKNOWN. No product defect or source repair is established by this review; no new implementation tranche or runtime is dispatched.

Owner renewed roadmap adherence, Root ORCHESTRATOR / independent REVIEWER, Luna Max worker and Sol Medium for difficult work. This is not a renewal of exhausted runtime/provider budgets. The permitted failed-terminal/lifetime review is performed here without reopening BUILD.

## Evidence and inference boundaries

| Finding | Source | Conclusion |
| --- | --- | --- |
| Stop command writes an intent and cooperative stop file, then returns. | `scripts/r120_ui_supervisor.py:887`–`892`; existing `stop_001/receipt.json` | Its exit 0 proves the request command finished, not completion of the original supervisor or children. |
| Child shutdown, final accounting checks and terminal publication run in the original supervisor's `finally`; terminal is written after those operations. | `scripts/r120_ui_supervisor.py:830`–`884` | This evidence path depends on the original process continuing through cleanup. That dependency is a source observation, not proof of what terminated this run. |
| Backend emits final fingerprint/effect records after observing the stop file and closing its server. | `backend/api/handlers/intervention_ui_r120_test.go:221`–`237` | The archived log contains neither final marker. Final values and original child completion cannot be reconstructed from initial values or later listener absence. |
| All 303 archived member sizes and SHA256 values match the final manifest; archive SHA256 is `37193f0a8d6bc88e935be0aec99fbdc539d6cb5e7d4c4f33cff7169c3eb85151`. | `probes/r120_root/lifetime_review_20261018.json` | Existing archive integrity verified in this continuation. Missing terminal and serve receipt are absent both in the archive and at their current local paths. |
| Archived effect ledger has no `supervisor-finally` stop reservation or child terminate/kill reservation; backend has no final, PASS, FAIL or timeout marker. | Same verification receipt and original archive | No recorded finalization is available. Absence does not distinguish external process termination, interruption, tooling lifetime loss or an unrecorded failure. |
| Serve intent to explicit stop request is approximately 507.988 seconds. | Archived `serve_intent.json` / `stop_intent.json` | Request occurred before the declared 1200-second limit. Original total lifetime and cleanup deadline compliance remain NOT_PROVEN. |

The original final review remains authoritative for actual UI and separate exact orphan cleanup. Separate cleanup proves its own bounded container/listener/protected-inventory observations; it cannot retroactively prove original process exit or database accounting. No cause is attributed to Docker, owner shutdown, agent tooling, or Go timeout.

## Finite transfer contract

1. Transfer this decision with the original final review, original archive/manifest, current R120 work order, immutable authority seed, admissions and budget record. Do not manufacture missing files or substitute a later database query for the original fingerprint.
2. Before requesting any new runtime, identify a supported execution arrangement that retains the original supervisor handle and records its actual exit through cooperative shutdown. Capability and source/resource admission must be concrete before dispatch. A chat continuation, stopped listener, or new screenshot is insufficient.
3. Any proposed runtime is a separate governed decision with fresh explicit finite authority; old R120 caps stay consumed. Keep its purpose tied to the remaining M3 evidence, inherit accepted UI observations, and require original terminal/child exits/durable accounting together. No automatic successor, general supervisor redesign, dependency installation, provider call or workflow replay is authorized here.

Until that decision is ready, the next move remains read-only transfer review. There is no justified source repair to send to Sol Medium. B07 remains inherited bounded, B08 CSS390x844/native JPEG375x811 qualified, B09 unsupported NOT_RUN; full M2/M3, M1 business acceptance, pilot and release remain open. Facebook/Zalo OA remain parked.

## Continuity correction and verification

The top-level `IMPLEMENTATION_STATUS.json` roadmap projection still described R119 setup failure and current UI as unverified. It is corrected to R120 bounded UI observed / original terminal and durable accounting missing. Historical R119 records and evidence remain unchanged. This correction changes no acceptance criterion or execution authority.

Root rehydrated canonical continuity, manifest/policy, public Core instructions, bootstrap log and targeted documentation index. Doctor passed 27/27, public remote/main and clean Core verified; manifest pin `e9ddcbcdcac2a9df520f5eb0761dbdb609b9deba` differs from current public Core `fa34ed49e56648acafe8b0388e48e0aa0523f007` (doctor warning only). Knowledge ingest produced one local chunk; no external ingestion was requested.

This continuation performs static file inspection and record synchronization only: zero product tests, runtime starts, browser actions or provider calls. Required repository metadata checks are recorded separately; they do not prove runtime governance, hosted CI or M3 completion.

Repository checks: governance unit tests 89 PASS (37.966 seconds); legacy preflight 7/7 PASS; profile reviewer-fast 7 mandatory / 0 failed, with ROUTE and ROLE checked. The first profile invocation did not recognize the descriptive role headings; standard Risk/Worker/Reviewer declarations were added and ROLE was actually checked on the next invocation. APPL remains NOT_APPLICABLE_WITH_REASON because the work order is BLOCKED_WITH_REASON, so the dispatch closeability contract was not checked or reopened. Pre-commit remains required before committing this records-only packet.
