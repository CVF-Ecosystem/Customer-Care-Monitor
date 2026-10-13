# R093 independent live outcome review

Status: CHANGES_REQUIRED_RETURNED_NO_RETRY
Risk: R2
Worker: /root/m1_worker
Reviewer: Codex /root

The pinned Alibaba-compatible endpoint returned HTTP 200 and `qwen3.7-flash`
for both real requests over synthetic inputs. This establishes exact HTTP wire
compatibility for these observations. The positive output contract failed;
R093 is returned with that finding and its request budget exhausted. No full
acceptance, application integration, CVF runtime governance, quality benchmark,
pilot or release claim follows.

| Actual observation | Worker complaint | Independent reviewer neutral case |
| --- | --- | --- |
| HTTP / returned model | 200 / qwen3.7-flash | 200 / qwen3.7-flash |
| Measured wall clock including process startup/cleanup | 5.141 seconds | 2.422 seconds |
| Reported prompt / completion / total tokens | 320 / 311 / 631 | 334 / 96 / 430 |
| Model/shape/schema/synthetic verdict checks | PASS | PASS |
| Required evidence references | FAIL | Not applicable: no violations |
| Overall output contract | FAIL | PASS |
| Returned provider field / authoritative monetary charge | UNKNOWN / UNKNOWN | UNKNOWN / UNKNOWN |

The model referenced `msg:r093-syn-agent-002`; the supplied ID was
`r093-syn-agent-002`. It also omitted the required complaint-message anchor.
Root checked the retained actual response against the exact request and agrees
with rejection. The model's FAIL verdict cannot substitute for valid evidence.
Do not weaken ID/quote validation or normalize an unknown ID silently to turn this
observation into a pass. A later application packet must present canonical IDs
unambiguously, require the needed anchors and reject invalid output before proposal
publication. It needs its own bounded application-path authority and evidence.

Receipts: `probes/r093_alibaba_live/worker.json` and
`probes/r093_alibaba_live/reviewer.json`. Independent inspection:
`probes/r093_alibaba_live/independent_live_inspection.json`. Both requests bind
source SHA-256 `41c6eb00b31a86294508954eeeb7e01ce61fb7f2c0535380580dac57ec5415fa`.
Root verified exact request bodies/hashes, output checks, reconciled token reports,
two distinct responses and the two exclusive claim/dispatch markers. Campaign
completed in 53.228 seconds, inside its original 300-second window. Total reported
tokens: 1061; numeric charge and actual remaining account quota remain UNKNOWN.
No retry, redirect, fallback, third request or ledger reset occurred. Secret value
is absent from receipts, and logical response strings passed the known-key guard.

The initial source failure and one consolidated repair remain disclosed in the
pre-effect review. Root separately corrected a Windows/PowerShell7 catalog-ID sort
drift; it also corrected an UTF-8 omission and an overbroad use of the response
scanner in its local inspection harness. These did not change the probe, receipts,
provider budget or observed failed result. Inspection corrections are retained in
the inspection artifact; no additional provider request occurred.

## Roadmap route reassessment

R092 intended contract and R093 provider characterization are two supporting
scopes. The next delivery outcome is M2: one synthetic manual intervention route
produces a snapshot-bound validated proposal, with WAIT_DATA/DENY ending before
provider disclosure. Its remaining acceptance includes durable physical-call
admission/unknown settlement and an actual application-path receipt. Then M3 adds
human review, assignment and resolved/reopened outcome in the browser.

No further generic R093 tooling/repair campaign is authorized. The failed positive
response is an application acceptance input, not grounds to reset its budget.
No Go, Docker or MySQL was found in PATH or checked conventional installation
paths. A coherent M2 packet must include one bounded verified local toolchain/DB
setup attempt; current R093 does not authorize installs, DB or application edits.
Absent setup evidence is an explicit runtime acceptance gap, not a source pass.
Business personnel/SLA/quality/financial limits are required before measured pilot;
they do not block the specified synthetic engineering route. Accounts stay parked.

## Return-Time Closeability Recheck

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
outsideAuthorityBlockers: NONE
nextRepairRoute: NO_R093_PROVIDER_REDISPATCH; NEXT_BOUNDED_M2_APPLICATION_PACKET
workerRedispatchAllowed: NO_PROVIDER_BUDGET_EXHAUSTED

Root returns R093 with the positive-output finding, preserves FREEZE OPEN and
records actual material SHA only after the steward commit. Repository gate results
are static checks only; hosted CI and deployment were not run.
