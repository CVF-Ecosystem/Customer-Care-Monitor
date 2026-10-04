# R046 engine finalizer driver-error containment

Date: 2026-10-04 (Asia/Saigon). Status: SPEC_ACCEPTED_FOR_WORK_ORDER; implementation NOT STARTED. Risk ceiling R2. Codex SPEC_AUTHOR. [Order](../work_orders/CCMAI_RUNTIME_046.md).

## INTAKE and DESIGN

Owner selected finalizer driver-error logging after bounded local MCP execution closure R045. Baseline backend equals accepted repair b4ec91ea03a9f247e209389c3792c86494eac8b3. In backend/engine/analyzer_incremental.go, finalizeOrdinaryRun maps statement failures to sentinels, but Transaction can return driver errors from transaction boundaries; retry logs print lastErr with %v, fallback logs print err with %v and the returned wrapper carries lastErr. The GORM logger can independently emit SQL/error detail. Static risk identified; no new leak reproduction or execution claimed in planning.

Design: use a scoped finalizer database session/error boundary that contains both application and GORM sink output without modifying global db/logger state. Expose fixed bounded error classes and trusted job/run correlation; never format an unknown error object through Error, String, formatting, or unwrap into a public string. Normalize unknown transaction errors to the existing terminal-write failure class; preserve errors.Is(errFinalizeWrite/errFinalizeMissing), and preserve missing-row no-retry behavior. Keep necessary internal diagnostics bounded; do not promise observability of raw causes.

Only finalizeOrdinaryRun and directly necessary private helpers in analyzer_incremental.go may change. Retry attempts/delay, transaction/parent-run lock order, tenant/job/run scoping, read-back verification, rollback, fallback predicates/message, cancellation/ownership/checkpoint/notification behavior and success semantics remain unchanged. Other analyzer logs/functions and callers are protected. No dependency, schema, public API or global logger change.

## Acceptance matrix

| ID | Required observation |
| --- | --- |
| FL-01 bounded app errors | Exercise a transaction-boundary raw synthetic error and fallback write error through the real finalizer. Captured application logs and returned error exclude recognizable synthetic secret, DSN, SQL and driver-detail fragments. Retry/fallback classes and trusted correlation remain useful. Ordinary statement errors alone cannot establish transaction-boundary containment. |
| FL-02 GORM sink | Install a capturing GORM logger at the caller DB boundary. Induce actual finalizer transaction and fallback statement faults carrying synthetic detail. No fault detail, statement SQL or bound payload reaches that sink. A silent application logger alone is insufficient; do not globally silence the caller DB. |
| FL-03 lifecycle invariants | Successful terminal job/run update and checkpoint; transient failure retry/recovery; exhausted retries return bounded failure without advancing checkpoint or notifying; missing row breaks retries; fallback success/failure remains bounded and scoped. Preserve cancellation/rejection/terminal-tail slot release via existing regression selection. |
| FL-04 error contract | Known terminal-write/missing sentinels remain discoverable by errors.Is, unknown begin/commit/rollback driver text cannot escape through returned wrapper; no new broad error identity promise. Include an adversarial error whose formatting is observable if needed to detect accidental formatting. |
| FL-05 meaningful detectors | Finite old-source or applied semantic controls discriminate raw retry/returned error, raw fallback log, and GORM sink suppression independently. Each needs a named relevant behavioral failure, full source hashes/diff, byte restoration and restored-baseline PASS; timeout/compile error/unrelated failure is INCONCLUSIVE. |
| FL-06 isolation | Cached-only build/vet, uncached focused finalizer and affected engine ownership/terminal regressions on disposable internal synthetic MySQL. No real config/provider/channel/notification/persistent data. New required cases must run, not silently SKIP. Record race/full-suite NOT RUN reasons and any fixture limitations. |

The contract concerns local application error containment on these paths only. Synthetic errors/providers are application fixtures, not AI governance evidence. General engine log safety, real driver/server/provider parity, live governance and hosted readiness remain unverified. Previously recorded errors/mutation survivors/incidents and R044/R045 local disposition remain historical evidence.
