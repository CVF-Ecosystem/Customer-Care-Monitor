# R139 boot identity recovery — independent review record

Status: REVIEW_PASS
Risk: R2
Worker: Root admission author; Sol implementation worker conditional on capability review
Reviewer: independent `/root/r139_cua_preclock_reviewer`
Date: 2026-10-25
Scope: one local read-only OS metadata capability observation only; source and relaunch remain unreleased.

## Initial admission review before observation

The Owner approved the single R139 recovery package (`Duyệt gói phục hồi R139`) under the existing M3-05/M3-08 order. I reviewed supplemental admission `CVF_SESSION/authority/CCMAI-RUNTIME-139-BOOT-IDENTITY-RECOVERY-ADMISSION.json` at SHA-256 `ab4b4d148b52992d562b665baae6f96e9c812d987c4d197de0024272e317ffdb` against approved proposal commit `faac97f5d1ba40cc03a6b7c5ee4c3c4ab0d74634`, the original independent preclock HOLD, accepted build pins, and current WORK_ORDER continuity. The authority's exact PowerShell script UTF-8 SHA-256 is `2324b4439551b833d9d1209f187ac921f44a13552646c1a67f720bc5485b9913`, matching its declared hash. Its PowerShell path/hash/size match the accepted build pin record.

The command is one `-NoProfile -NonInteractive` invocation with a five-second cap and zero retry. It reads `Win32_OperatingSystem`, `Win32_ComputerSystemProduct`, and the latest System `Microsoft-Windows-Kernel-General` event ID 12; it emits one compressed JSON object with OS computer name, product UUID, last-boot UTC string, and event record/time/XML `StartTime`. A missing or duplicate `StartTime` throws. This is a bounded local metadata discovery query; it does not call Docker, ports, browser, product service, network, credentials, or provider. The record requires original `Popen`/wait/exit and raw stdout/stderr in ignored local custody, with only redacted shape/digests committed. The query may fail or return unusable identifiers. It does not yet establish that the event is an eligible stable boot discriminator or that UUID/name are reliable on this host.

The admission preserves original controller attempt 1 and its `CURRENT_HOST_BOOT_DRIFT` failure, campaign reservation 0, old source checks 8/8 spent with one failed attempt, and old unknown provider history. It adds at most one metadata observation; later source budget is one authoring pass, one consolidated repair, worker focused Python 1 and Root focused Python 1 (total lineage ceiling 10), zero Go compile/tests. One additional original preclock invocation is conditional on a usable capability receipt, explicit reviewed identity contract, independent R2 source/packet review, both bounded checks, normal clean commit, fresh exclusive paths and original process custody. The existing one campaign, five service queries, 64 CUA methods/four stages, 1200-second clock, and one physical provider call within USD 1 are unchanged and remain conditional. No product effect is released by this admission.

**Verdict: PASS for the exact single read-only capability observation after Root commits this supplemental admission and its aligned continuity through normal gates.** The observation must use the exact approved argv/sha/pinned binary/scrubbed environment, reserve its one attempt before launch, retain original process receipts and raw private output, and stop on timeout, nonzero, ambiguous, malformed or missing data without retry or fallback. An event ID 12 record or UUID value alone is not a capability PASS; I will review the actual receipt and candidate identity semantics before source BUILD or either Python check. The previously committed controller/design, boot proof, authority seeds, and original failure receipts remain immutable.

## Later review checkpoints

- Actual one-query receipt: PENDING.
- Explicit identity contract and source candidate: PENDING.
- Worker focused Python attempt (new cap 1): NOT_RUN.
- Root focused Python attempt (new cap 1): NOT_RUN.
- Revised clean technical packet and original preclock attempt: NOT_RUN.

## Root continuity preparation note

Initial repository preflight failed because Root put DESIGN/SPEC phase labels into the tranche status history. Those uncommitted invalid entries were removed, retaining the committed CHANGES_REQUIRED history and REVIEW phase pending capability evidence. No source BUILD was performed. Owner-approved read-only observation is a bounded review prerequisite; conditional source dispatch remains blocked. No gate bypass or allowance change.
