# R094 initial independent source and evidence review

Status: CHANGES_REQUIRED
Risk: R2
Worker: /root/m2_worker
Reviewer: Codex /root
Date: 2026-10-14

## Scope and source

Root reread the current R094 canonical tuple, accepted SPEC/order/seed, implementation current projections and index before independent REVIEWER evaluation. Worker source is uncommitted and identified by probes/r094_worker/source_hashes.json; worker six Go invocations are exhausted. Existing authenticated native route and adapter results are consumed, not relabeled as complete M2 or live governance. Two fixture-input/rate-limit failures remain in original raw logs. Root has four Go invocations and one disposable fixture remaining under the unchanged seed.

## Consolidated findings before first repair

F1 SANITIZED_OUTCOMES: `RunManual` passes `pubErr.Error()` to `finish`; a transaction/commit driver error can therefore become a stored/returned reason. Whitelist only known safe codes and map everything else to PERSIST_FAILED. A consumed attempt is currently settled only with a valid proposal, so received-but-invalid or stale output is left CONSUMED with UNKNOWN even when response/usage were received. Settle the physical response truth independently of proposal acceptance, preserve known/partial/unknown usage accurately, and keep every attempted slot consumed on settlement failure. Add atomic publication/settlement failure evidence; no false successful case or refund.

F2 BOUNDED_ADAPTER: `AnalyzeChatOnce` limits bytes but does not validate its constructor's completion-token value; it can inherit the old 16384-token default. Its client has no independent timeout for a direct admitted invocation. Add explicit scoped token/timeout bounds without changing AnalyzeChat defaults. Model currently reports configured p.model as if response metadata; expose/validate actual response model or keep unavailable metadata explicitly unknown. Preserve no retry/redirect and add exact boundary/error tests. No real request is admitted by these changes.

F3 EXPLICIT_DENIAL: Non-synthetic/foreign-profile source returns generic unavailable503 rather than the intended DENY decision. Add a safe explicit source/profile denial outcome, with zero reservation/provider disclosure and no cross-tenant existence leak. Authorization denial may remain middleware403 without a persisted case; define this accurately. Test default disabled and cross-tenant source/detail as well as marker/policy denial.

F4 ACCEPTANCE_GAPS: Worker correctly marks concurrency, broader malformed evidence, cross-tenant and failure-persistence cases NOT_RUN. Add meaningful targeted cases for cap contention/reentry, fabricated quote, invalid Unicode offsets, kind/anchor mismatch, wrong digest/schema/malformed output, and failed settlement/atomic publication. Preserve the existing positive, stale, cancellation, member permissions and legacy-bypass detectors. Root will run one consolidated revised suite using its own owned fixture; worker must not exceed6 Go. This is one same-scope repair, not a new supporting tranche.

## Route and disposition

Return source/tests to Sol Medium REPAIR_WORKER inside immutable R094 paths, effects, roles and counters. Root retains independent reviewer and commit owner; no seed edit, new provider budget, secret read, service activation on existing app, push or deployment. Reviewer-local metadata corrections remain Root-owned. After repair, Root source recheck and native tests assess only bounded local software acceptance; real application provider/governance, M2 whole exit, M3 browser and pilot remain OPEN. Third repair without independent root cause stops as required by AGENTS.md.
