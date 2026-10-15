# R101 — Independent review of Claude browser return

Status: BLOCKED_WITH_REASON
Risk: R2
Worker: Claude / Evidence Operator (application implementation remains /root/m3_design)
Reviewer: Codex /root
Date: 2026-10-15
Disposition: BLOCKED_RETURN_RECORDED_WITH_QUALIFICATIONS
Build commit: 200df3e4a7bc2a326425922a5230508a4fd9b0fe
Freeze: OPEN

## Decision

The return is recorded as a blocked browser attempt, not accepted browser evidence or a product pass. Claude reports `BLOCKED_BROWSER_TOOLING_PLAYWRIGHT_DRIVER_404_AZURECDN`. B01–B10 remain NOT_RUN at the browser layer. No image capture, actual browser observation, or action/persistence result was delivered. Existing native software evidence remains inherited, not rerun by Claude and not a substitute for H08.

Root rehydrated the current manifest/policy/bootstrap/state/memory/active handoff/implementation/index. Doctor PASS27/27 with the retained historical manifest-pin warning; public Core fa34ed49e56648acafe8b0388e48e0aa0523f007 matches origin/main. Root acts as independent REVIEWER, ORCHESTRATOR and SESSION_SYNC_STEWARD. No application repair, server restart, browser call, test binary launch, provider call or download was performed in this review.

## Independently checked facts

Read-only verification receipt: `docs/reviews/probes/r101_root/claude_return_independent_verification.json`. It records observation time, report/manifest hashes, source comparisons, Docker safe-field inspections and host listener/process probes.

- All three listed Claude evidence files match their manifest SHA256 and byte size. The manifest excludes the report and itself; Root's new receipt records hashes for both. Matching a hash establishes file integrity, not that its narrative is independently authenticated.
- All 17 source manifest entries match. The existing binary SHA256 is `4c7575e180dd198d995bfe09bdc45061740a987be8ec0765f02b203e1c251f31`. Git HEAD remains 200df3e4a7bc2a326425922a5230508a4fd9b0fe; tracked diff and untracked-file queries under backend/frontend are empty. This supports no application source change and no new commit during the handback.
- Shared launch counters and browser process/ready records show workerGo1, reviewerGo2, aggregateGo3/16; fixtures2/2, Node checks3/6, frontend build1/1, recorded provider/mock0. The second reviewer launch is the real-router browser fixture, not an additional native test campaign. Vite was reported launched once on this final fixture; the per-fixture development-server allowance is consumed.
- The same labelled final container, image, named volume and network identities are preserved. The container is currently exited. No listener is present on reported ports3011/52962/56039, and PID9536 / the named test process is absent at review time. No cleanup mutation was needed.
- All three protected-container safe-field snapshots (id/name/image/status/mounts) equal Root's recorded baseline. This confirms current equality for those fields; it does not prove every historical action or all possible resource invariants.

## Qualifications and consolidated findings

1. **Browser proof is absent.** The failure JSON, cleanup JSON and port reconciliation JSON are operator-authored records. No exported raw browser-tool transcript, Vite process receipt with OS PID, HTTP200 command output, or before/after protection snapshot from Claude accompanies them. Root corroborates current cleanup and existing fixture launch records, but does not independently authenticate the exact CDN HTTP404 chain, past Vite proxy behavior, past UI rendering or historical noninterference. The report's local readiness assertion is not application acceptance.
2. **Strict no-download compliance cannot be accepted.** The recorded error explicitly says the browser manager attempted to install/fetch a Playwright driver from an external CDN. The packet and seed forbid install/download effects. An automatic tool bootstrap may have caused this; no intent or successful installation is inferred. Nevertheless a failed fetch is an attempted external network/download effect, so the report's blanket boundary-compliance wording is too broad. Retain the original return unchanged as evidence and record this exception here. Do not repeat the attempt, install a driver, repair the browser runtime, fetch an alternate URL or change automation stacks under R101. Provider/API budget0 is distinct from external HTTP activity; provider0 must not be described as network0.
3. **Continuity needs actual counters.** Canonical continuity previously still said await return / aggregate2. This review synchronizes the actual aggregate3 and blocked disposition without altering the immutable seed, phase or status grammar.

These are review/evidence and infrastructure findings, not new application defects or a third source-repair round. No source repair is authorized by this disposition. Root does not silently rewrite Claude's original report to remove the failed attempt or stronger historical claims.

## Finite next move and claim boundary

Keep R101 REVIEW_PENDING / COMPLETE_PENDING_REVIEW and FREEZE OPEN. Keep the three reproduced full-frontend Dashboard baseline failures OPEN. Full M2/M3, WAIT_DATA rerun, M1 business inputs, pilot/release and Facebook/Zalo OA parked checkpoint are unchanged.

Do not automatically relaunch the fixture or Vite for another browser attempt: the one development-server launch per final fixture has been used, even though Go slots remain, and fixture2/2 does not permit replacement. First establish a browser/manual operator that can actually run locally without downloads or a policy bypass. Any new server-launch allowance requires a bounded dispatcher/owner authority decision; do not edit the existing seed or infer the allowance from unused Go capacity. A request to repair the three old Dashboard tests needs its own exact-path authority because those paths are outside R101.

The next roadmap step is to record the remaining browser execution route and its narrow authority before any effects, then collect H08 evidence and independently review it. There is no browser setup campaign, source expansion, provider execution, installed activation, hosted-CI assertion, push or deployment in this disposition. All current review material remains a local uncommitted checkpoint until commit-steward checks; no tranche closure is declared.
