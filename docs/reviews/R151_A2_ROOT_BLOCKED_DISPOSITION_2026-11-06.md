# R151 A2 source-only blocked disposition

Status: BUILD_BLOCKED_SOURCE_NOT_ACCEPTED

Date: 2026-11-06

Risk: R2

Worker: Terra 6.1 IMPLEMENTATION_WORKER

Reviewer: Astra Medium independent R2 REVIEWER

Role: Root ORCHESTRATOR / SESSION_SYNC_STEWARD / COMMIT_STEWARD. This records the result; it does not self-approve R2 source.

## Decision

Accept the worker's `BLOCKED_WITH_REASON` return and Astra's independent `BLOCKED` source review in `docs/reviews/R151_A2_SOURCE_INDEPENDENT_REVIEW_2026-11-06.md`. Keep the A2 runner/test candidate and raw logs as a bounded, unaccepted source attempt. Do not admit a Windows A1/A2 effect, reopen R139 integrated BUILD, claim M3-05/M3-08 acceptance, or claim runtime CVF governance behavior.

The independently identified cleanup bypass is improved: actual `runCleanup` is executed under a custody persistence throw, and the fixture sees later Chrome close and terminal snapshot attempts with UNKNOWN outcome. The test suite reports 52 PASS, zero skips and worker-reported exit 0; final-byte Node/PowerShell syntax log reports success. The seven-file candidate manifest matches current raw bytes. These results do not close V04.

## Observed remaining cause and limits

The replacement fixture calls exported `runReplacement`, which forwards to the shared helper, but injects a test-written close callback that throws on timeout. Production `closeHandle` and its custody/wait/kill path, plus the production launch/debit closure, are not executed by this fixture. The fixture could remain green if production stopped throwing on timeout. The private main path and final persistence reduction are also unexecuted. This is an observed evidence-design gap, not an observed Chrome runtime failure.

Astra independently reviewed final-hash V06 static paths for A3, positive WebSocket, A1-c and A2; that source result does not substitute for V04 injected integration or native behavior. Root retains the exact worker return's historical A1 details and the Nov-6 reviewer F01/F02 qualifications. No provider call or Windows product effect occurred in this source-only A2 work as worker-reported; there was no independent OS-wide effect inventory.

## Cumulative accounting and next move

- Original Nov-5 hard stop `17:02:18+07:00` remains expired. Historical focused tests 4/4; historical syntax evidence 1/2, with unused old syntax slot expired; historical active minutes UNKNOWN.
- A2 release `2026-11-06T07:42:44+07:00`; worker source/check stop by `07:46:40+07:00`, conservatively under four elapsed minutes. A2 spent its single focused test and single syntax batch. Cumulative focused test invocations **5**, cumulative evidenced syntax batches **2**. Remaining A2 test/syntax/source-pass allowance **zero**; time remaining does not restore it.
- R151 returns to BUILD_BLOCKED. Root will audit the repeated V04 integration-proof gap and repair cost before any dependent IMPLEMENT. A third repair pass without an independent new root cause requires `REVIEW_COST_ESCALATION_REQUIRED`; a new tranche/name/model cannot reset history. Any prospective work needs an exact finite authority and independent R2 admission before worker action. No automatic retry or extra process effect is released by this disposition.

R150 remains A1-b INCONCLUSIVE, A2 UNRUN and terminal Chrome descendant cleanup UNKNOWN; its spent effects stay spent. R139 physical no-relaunch and same-owner/context/original-clock integrated proof remain hard-held, M3-05/M3-08 BLOCKED, Facebook/Zalo OA parked. Source-only progress is a prerequisite improvement, not product acceptance.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
