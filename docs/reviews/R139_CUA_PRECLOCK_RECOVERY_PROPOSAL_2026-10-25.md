# R139 CUA preclock boot-identity recovery proposal

Status: **DRAFT — static design only; no implementation, check, relaunch, or effect authority**  
Tranche/order: existing `CCMAI-RUNTIME-139`, roadmap `M3-05` / `M3-08`  
Risk: R2
Worker: /root/r139_cua_recipe_worker (static SPEC_AUTHOR)
Reviewer: Root independent recovery-design REVIEWER  
Author role: prospective internal G0 actor, temporarily SPEC_AUTHOR for this proposal  
Decision owner: Root ORCHESTRATOR and independent R2 reviewer before any revised dispatch

## Observed failure and preserved boundary

The first and only original preclock launch used clean dispatch `fd730262bd55d73d7bdc681d891f0429b912997a`. Its original controller process returned exit 1 before reservation. The committed boot string was `|2026-10-23T05:48:27.5010520Z`; the same scrubbed PowerShell query returned `|2026-10-23T05:48:27.5037950Z`, a 2.743 ms difference. The host component is empty because the controller environment did not include `COMPUTERNAME`. The earlier two read-only observations happened to match exactly; that does not establish exact-string stability. The physical cause of the later difference remains unknown.

Source: [independent HOLD review](R139_CUA_PRECLOCK_HOLD_REVIEW_2026-10-25.md) and [original hold receipt](probes/r139_integrated_20261021/native_operational_20261024/cua_preclock_attempt_20261025/root_hold_receipt.json). Preserve the original child logs and outer original wait/exit. No `campaign_reservation.json`, G0 signal, or activation receipt was written. Campaign/service/CUA/fixture/key/provider effects remain zero. The first controller attempt is spent; it is not eligible for retry under the sealed contract. All eight original source checks are spent, including the retained failed worker attempt.

## Proposed identity contract, conditional on capability evidence

The revised preclock check must bind **host identity**, **boot session**, and **clock sanity** as separate facts. Merely raising the tolerance on `Win32_OperatingSystem.LastBootUpTime` or replacing the pinned string with the new observation is insufficient. The raw timestamp remains evidence, not the sole identity token.

1. **Host identity.** A fresh, read-only, locally available OS identity must be observed in both a sealed baseline and the later preclock query using the same pinned binary, scrubbed environment, and exact query. Candidate: `Win32_ComputerSystemProduct.UUID` through the already used CIM facility, paired with the OS-reported computer name from CIM rather than ambient `COMPUTERNAME`. This is a candidate, not a proven stable or unique primitive on this host. Missing, empty, placeholder, malformed, or changed identity is HOLD. A cloned VM can duplicate this metadata; the contract must describe this limit and keep checkout/tool/boot proof and original process custody as additional bindings.
2. **Boot session.** Prefer a stable, OS-recorded boot-event identity that does not derive from the mutable high-resolution `LastBootUpTime` rendering, such as the latest eligible OS-start event record identity if the event source is present and its selection is unambiguous. The event source, exact selection rule, retention, access, and stability must be established by a bounded read-only capability observation before naming it in an executable contract. A missing/ambiguous/truncated event history is HOLD, not a fallback to timestamp tolerance. If no suitable stable boot-session discriminator is available, return HOLD for an independent design decision rather than silently weaken the gate.
3. **Clock sanity.** Keep the existing UTC-versus-monotonic campaign drift checks. In preclock, parse the two `LastBootUpTime` values as strict UTC timestamps and record their signed difference. A small difference such as the observed 2.743 ms may be classified as *timestamp-rendering variation* only when the sealed host and boot-session identities match and the independent clock sanity check passes. The allowed difference, sampling horizon, and clock sanity rule must be set from the capability evidence before implementation; this proposal does not invent a safe threshold. Any large difference, unparseable value, reversal inconsistent with the boot identity, or evidence of a clock step is HOLD. Small wall-clock adjustments below a future threshold may be indistinguishable from rendering variation; this limit must be explicit, not claimed away.

This contract detects an ordinary reboot by a changed boot-session record and host substitution by a changed host identity. It cannot cryptographically exclude a cloned machine with copied identifiers or a compromised OS. No runtime governance claim follows from these metadata checks.

## One bounded discovery and review decision

Before source repair, request **one** newly authorized read-only local OS metadata capability observation, with exact PowerShell argv, output schema, five-second timeout, original `Popen`/wait/exit/raw stdout/stderr and no retry. It may inspect the already used CIM OS value, candidate CIM host identity and one explicitly selected boot-event source. It must not query Docker, ports, browser, provider, key, network, product process, or fixture. Use the same scrubbed environment planned for the controller. Independently review whether the candidate identifiers exist, are non-placeholder, parse consistently, and can be pinned without leaking raw host identifiers into public artifacts; store raw locally with a digest or redacted derived form in the repository. One observation alone cannot prove future stability: compare it with existing retained boot observations where fields overlap, then define the conservative failure rule. If the stable boot event cannot be established, stop at DESIGN/HOLD.

The current reviewer should decide this capability contract first. Its effect class is local OS read-only metadata, outside the original exhausted source-check budget and outside the unreserved campaign; it needs a distinct explicit admission. This is not permission to run the query now.

## Acceptance cases for a future pure parser/check

| Case | Required disposition |
| --- | --- |
| Exact raw value and both identities match; clock sane | PASS identity gate; continue existing preclock checks. |
| Raw boot timestamp differs by 2.743 ms; independently established host and boot-session identities match; difference and clock sanity satisfy reviewed bound | PASS with both raw values and qualification recorded. |
| Raw mismatch exceeds the reviewed bound, even if identities match | HOLD; no reservation. |
| Boot-session identity changes, including after a real reboot | HOLD; no reservation, regardless of timestamp proximity. |
| Host identity changes or becomes placeholder | HOLD; no reservation, regardless of boot timestamp. |
| UTC moves backward, monotonic/UTC comparison exceeds the reviewed bound, or boot-time parsing contradicts observed clock | HOLD with raw timing evidence. |
| Missing, malformed, duplicate, or ambiguous host/boot/event data; command nonzero/timeout/original wait unknown | HOLD; no fallback or automatic retry. |

Future focused Python fixtures must exercise all cases and exact source/receipt boundaries. Those fixtures are design examples, not current test results.

## Bounded implementation and launch request after design acceptance

Do not edit the committed `reentry_activation_design.py.txt`, its accepted build pin record, old boot proof, grants, or original failure receipts in place. If capability review succeeds, authorize one successor *version* of the Root-only activation design under `docs/reviews/probes/r139_integrated_20261021/native_operational_20261024/`, focused pure-Python identity fixtures in the already allowed `scripts/tests/test_r139_execution_supervisor.py`, a new boot-identity proof and exact recipe/technical authority references, and the eight canonical continuity templates within the same R139 order. Root owns grants, state, catalog, commits, and original launch custody. No Go handler, supervisor, binary, Vite, product source, or provider change is proposed.

Proposed new source/check ceiling after independent approval: one design/source authoring pass, one consolidated repair pass, **one worker focused Python check and one independent Root focused Python check**, zero Go rebuild/Go test and zero product/runtime test. A check attempt counts when invoked, including failure. Independent R2 review of the revised contract, exact changed paths, raw capability receipt, source diff, check receipts and current pins must precede any launch authority. If the reviewed design fails at repair round three without a new root cause, record `REVIEW_COST_ESCALATION_REQUIRED` before continuing.

One consolidated Owner decision must cover the complete conditional recovery package: the single read-only identity observation, one authoring/one consolidated repair pass, two focused Python attempts, and **at most one fresh original preclock invocation** after independent source/packet acceptance and normal clean commit. Root performs the conditional technical release within that one approved package; do not ask for a second Owner approval between these same-scope steps. The later invocation requires its own exclusive log path and original wait/exit. The failed invocation remains attempt 1 and is not rewritten or retried. A subsequent clean dispatch must pin the revised design/controller bytes, host/boot proof, actor ACKs, recipe, eight canonical templates, hooks, tools, and expected HEAD. The revised controller must still reserve the same one unspent campaign only after all preclock predicates pass, then preserve the original +1020 s cleanup and +1200 s expiry. Existing unspent caps remain one campaign, five internal queries, 64 CUA methods/four stages, and at most one physical provider attempt within USD 1, all conditional; old spent and unknown outcomes remain spent/unknown. Product fixture/key/provider effects still require actual G0 proof, Root review, and a separate Runtime Effect Admission.

No new tranche, helper order, automatic alternate browser stack, public release, deployment, or claim of G0/product success is part of this proposal. If any identity prerequisite is unproved, the disposition stays HOLD and the next move is a consolidated design decision within the existing R139 order.

## Root consolidated design review

Disposition: ACCEPTED_AS_CONSOLIDATED_DECISION_PACKET_ONLY. The observed exact-string mismatch is established; a replacement primitive and physical cause remain unproved. One decision must cover discovery, bounded repair/checks and conditional launch, with no helper tranche or additional per-stage Owner relay. Keep the original test path, all eight spent checks, original invocation1, and campaign0 unchanged. Proposed two new focused attempts would make the source-check lineage ceiling10; no Go compile or pure-Go attempt is requested. No part of this draft admits implementation, metadata discovery, retry, fixture, browser, key or provider effects. Capability discovery must HOLD if the proposed identifiers cannot be established; source review and later actual G0/Runtime Effect Admission remain required.
