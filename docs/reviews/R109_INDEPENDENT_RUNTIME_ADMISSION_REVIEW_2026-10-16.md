# Independent admission of the shared R109 campaign

Status: REVIEW_PASS
Risk: R2
Worker: /root/r108_browser_luna (Luna Max IMPLEMENTATION_WORKER)
Reviewer: Codex /root
Commit steward: Codex /root
Disposition: RUNTIME_ADMITTED_AFTER_COMMITTED_SOURCE_AND_PINNED_RECEIPT

Root independently reviewed the entire new supervisor and Vite template,
the reused immutable R108 helper, R109 seed/order, binary reuse receipt and
the UI source/validation. Observation date: 2026-10-17 Asia/Saigon. The 16th
in the packet names remains their established identifier, not a new budget.

## Findings resolved before runtime

- Pipe capture uses read1 and complete-line redaction. Root's actual pipe
  regression observes readiness while the writer is still open and confirms
  redaction of a synthetic password split across writes.
- MySQL readiness has a 90-second startup bound. Empty/short startup reads
  remain NOT_READY; a complete protocol-10 greeting is required. A complete
  server error stops once, retaining only a safe numeric diagnostic.
- Post-run integrity compares stable helper hashes rather than newly imported
  module identity. Git verification receives the remaining campaign deadline;
  reader joins are bounded within cleanup. Entire backend and frontend source
  trees are checked, including untracked source.
- Exact fixture identity/network/volume/protection are checked before start
  and after cleanup. Stop uses --timeout and structured final state rather
  than treating all Docker stdout as an ID. Owned Popen handles are registered
  before result receipts and are closed in finally.
- POST observations distinguish requests to the real API from a local harness
  limit denial. No request body or auth header is captured. A local 429 is not
  product behavior. All attempts and Gin-observed POSTs remain counted; Root
  reserves before UI submissions and leaves headroom below fourteen.
- Port conflict is checked before fixture effects, setup failures retain a
  receipt, and status prefers the terminal disposition.

Root executed AST/import inspection, three focused offline harness regressions
and an actual cached Node config import with assertions. All PASS. Evidence is
under `docs/reviews/probes/r109_root/harness_validation/`. This is tooling
validation, with simulated MySQL packet fragments; no database/service/provider
or product browser was invoked by those regressions. The independent current
browser HTTP click/change/screenshot probe separately PASSed and was cleaned,
under `bridge_renewed/`. Driver/cache versions are NOT_OBSERVED.

## Admission and limits

Execution requires a committed source revision and the separate committed
`runtime_admission.json`, pinning source, seed, active order and evidence.
The admitted campaign reuses binary cb73ee13 (58,031,616 bytes), with zero Go
compiles; one existing fixture start, one native invocation/four action POSTs,
then only on NOT_REPRODUCED the same fixture plus one backend and one Vite for
Root's connected UI. Maximum 2100 seconds including cleanup, no retry, three
owned product tabs and fourteen UI POST attempts including auth/rejections.

No production backend change, provider call, governance claim, dependency
installation or real data access. Native failure stops before product UI;
actual browser failure stops owned runtime and returns to the owner for
transfer. B09 stays NOT_RUN if no documented offline control exists. Passing
current REOPEN does not retroactively establish the cause of R107's 503.
Dashboard baseline and broader product acceptance remain open.

This admits the bounded execution only. Native and product UI results are
NOT_RUN at this review; no FREEZE, release or worker self-approval.

## Pre-start text representation correction — 2026-10-17

The first read-only committed check stopped on the locale's physical CRLF vs
Git LF representation, before the campaign slot was created. Root accepted
the independently reviewed two-comparison repair in
R109_NEWLINE_REPRESENTATION_REVIEW_2026-10-17.md and targeted checks in
newline_validation.json. Exact physical hashes stay mandatory. The current
admission must point to the repaired source; the initial admission is retained
as runtime_admission_initial.json. No fixture or native budget was consumed.
