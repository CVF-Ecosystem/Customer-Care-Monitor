# R046 exact-BUILD independent review

Date: 2026-10-04 (Asia/Saigon). Reviewer: Codex, independent of Claude IMPLEMENTATION_WORKER / BUILD COMMIT_STEWARD. Risk ceiling R2. Disposition: CHANGES_REQUIRED / REVIEW / FREEZE_OPEN for R046-R1-01..02; no production defect established.

Exact BUILD985fa60b3766444a7b71a5303da6c6bc14e63bf0; hand-backf20cbd6d8ca367cdd01c8115e6423d48f08e51be. Dispatcher seedc3ff83c7b8d2f9b23ef4113ae3bf7f8b9b32c267 unchanged and present at baseCommit. Current backend equals exact BUILD, only authorized analyzer_incremental.go and analyzer_finalizer_logging_test.go changed from pre-BUILD. Both worker source SHA256 values match committed blobs. All prior tranche records and immutable seeds unchanged. Git identity alone cannot prove individual authorship or intra-worktree acknowledgment timing; worker's same-BUILD acknowledgment is attributed, not independently timed.

## Consolidated findings

R046-R1-01: maintained tests do not reach the new transaction-boundary normalization. TestFL01BoundedAppRetryLog and TestFL05RetryLogDetector inject BEFORE UPDATE statement faults, which the old transaction closure already maps to errFinalizeWrite. TestFL04AdversarialErrorNotFormatted explicitly states its adversarialError never enters finalizeOrdinaryRun. Removing the complete normalization switch and using lastErr=txErr survives all14 FL tests/19 pass events, zero FAIL/SKIP. This is a detector gap, not proof of a production leak in submitted source. Add maintained finite raw BEGIN/COMMIT boundary injections through the actual finalizer, with observable formatting counter, bounded returned/app/GORM output, sentinel preservation, transient retry/recovery and exhausted-failure/checkpoint assertions. Preserve existing positive/fallback/sink controls. Independent reviewer probe and restoration evidence will be recorded below; worker must maintain equivalent coverage in the authorized test file.

R046-R1-02: submitted FL-05 receipts label D01..03 ACTIVE_PASS but provide no applied mutant, exact replacement/diff, source hashes, named behavioral failure or byte-restored baseline. Passing detector tests alone does not prove sensitivity. FL-06 grouped regex/counts also lacks the complete affected terminal/ownership selection and precise executed counts. Provide separate worker campaign receipts under the unchanged source/test paths, including all baseline/mutant/restored commands/exits/top-level/subtest/SKIP counts, full SHA256, named assertions, failures/survivors/inconclusive attempts and cleanup. Recover actual old logs if available; otherwise label new replay instead of inventing historical evidence. Inherit original counts as worker-attributed, not reviewer re-execution. No source change requested unless a new defect is independently established and returned.

## Prerequisite repair and boundaries

Original doctor23/25 FAIL (stale public core/catalog), stale nested ownerRouting/no-BUILD narrative and cross-shell catalog ordering conflict remain historical. Owner continued after blockers; standard reconciler preserved backup and cloned public tip8a4119e11db00e774ed8e7cf7d9a8caa309e81d1. Ignored local binding refreshed; no sibling project manifests or core product code edited. Manifest historical pin26c686cc99b8be965d2760f27fe875b03376c643 remains public-reachable. Doctor PASS WITH NOTE: pin mismatch warning and new downstream gate profile NOT_INSTALLED/migration required; no inherited profile coverage claim. Catalog receipt ID renamed to remove PS5.1/7 sort ambiguity; artifact path/receipt bytes unchanged, both-shell catalog and portable preflight7/7 PASS. Bootstrap read model absent, BOOTSTRAP_MIGRATION_PENDING nonblocking. No silent gate/policy weakening or pin migration.

Review execution uses exact Git archive, cached golang:1.26-alpine/mysql:8.0 and read-only module cache, task cache, disposable synthetic DB on internal network without host ports. Repository product/tests untouched. Cold baseline compile/vet266.561s; baseline14 top-level/19 total PASS0 FAIL/SKIP; normalization-bypass mutant14/19 PASS0 FAIL/SKIP82.042s. Remaining mutation/restoration/regression/probe results pending at draft time. Receipts supplied after execution; no final disposition from this draft.

NOT RUN: race detector (CGO=0), full engine/backend suite, real provider/channel/notification/config/credential/external network/persistent or customer DB/application runtime. Synthetic application error-containment tests are not CVF AI governance proof, general engine secret-safety, live parity or hosted readiness. Accounts parked; R044/R045 local closure and prior dispositions unchanged. No push/merge/deployment/FREEZE.

Reviewer incidents: guessed .tmp path absent; docker top with comm-only ps output refused because PID field missing, corrected to pid,comm; no source/resource mutation from those read-only lookups. First PS5.1 catalog regeneration passed its check but PowerShell7 gate failed6/7; cross-shell ID correction and successful both-shell checks are separate. Original worker failures/omitted validation remain attributed; no historical doctor/receipt is fabricated.

## Completed independent execution

Sanitized [receipts](probes/r046_independent_summary.json) contain exact commands/counts/named assertions/log SHA256, one-match replacements and full baseline/mutated/restored hashes. Reviewer boundary probe `docs/reviews/probes/r046_transaction_boundary_probe_test.go` mounted only in an exact-BUILD archive.

| Run | Exit | PASS / FAIL / SKIP events |
| --- | --- | --- |
| baseline | 0 | 19 / 0 / 0 |
| M01-normalization-bypass | 0 | 19 / 0 / 0 |
| M01-normalization-bypass-restored | 0 | 19 / 0 / 0 |
| M02-raw-fallback-log | 1 | 0 / 1 / 0 |
| M02-raw-fallback-log-restored | 0 | 1 / 0 / 0 |
| M03-gorm-session-bypass | 1 | 0 / 1 / 0 |
| M03-gorm-session-bypass-restored | 0 | 1 / 0 / 0 |
| ownership-terminal-regression | 0 | 59 / 0 / 0 |
| build | 0 | 0 / 0 / 0 |
| vet | 0 | 0 / 0 / 0 |
| boundary-probe-baseline | 0 | 2 / 0 / 0 |
| M01-with-boundary-probe | 1 | 1 / 1 / 0 |
| M01-with-boundary-probe-restored | 0 | 2 / 0 / 0 |

Baseline/restored FL suite14 top-level19 total events; affected ordinary/terminal ownership regression18 top-level59 total events, all zero FAIL/SKIP. Removing the normalization switch survives the entire FL suite; byte-restored full FL suite passes. Raw fallback log mutant fails the named FL05 fallback detector; enabling GORM Info in scoped finalizer session fails the named GORM detector; each restored test passes. Both compiling behavioral controls are distinct from the normalization survivor. No mutation applied to repository source.

Reviewer probe injects a synthetic raw BEGIN error at the real GORM ConnPool transaction boundary, counts failed BEGIN attempts and observable Error formatting, and tests transient failure followed by genuine transaction/checkpoint success. Both baseline cases pass, no driver Error formatting observed. Same normalization-bypass mutant now fails TestR046ReviewerUnknownBeginErrorIsContained at its bounded-sentinel assertion; transient case passes. Removing the mutant restores2 PASS0 FAIL/SKIP. This supports the submitted normalization branch for this synthetic BEGIN path, but the maintained worker suite still lacks the required detector. Real BEGIN/COMMIT/ROLLBACK provider/driver parity is unverified, not inferred from this overlay.

Cached full backend build/vet exit0. Full engine/backend tests and race NOT RUN. Archive audit verifies all exported files byte-equal after overlay removal; task containers/internal network/anonymous DB volume removal exit0 and named DB/network absence independently verified. Persistent application stack untouched; task archive/raw logs/cache retained locally without credentials in committed receipts.

Final REVIEW decision: CHANGES_REQUIRED for R046-R1-01..02 only. Submitted production scope is preserved and these runs establish no production defect requiring source repair. Concrete test/evidence-only Claude R1 return in the unchanged work order; Codex remains independent reviewer. No CLOSER/FREEZE or live/governance readiness disposition.

Publication incident retained: initial docs build FAIL on a link to the repository-only Go probe (not a VitePress page/static asset). Corrected to a code pointer without changing ignoreDeadLinks or probe bytes; corrected-state docs build follows. Default/PR preflights each7/7 PASS, both-shell catalog PASS, gate unit46 PASS53.037s and fresh doctor PASS WITH NOTE (pin/profile migration limits above). These repository checks do not replace product or live governance evidence.
