# CCMAI-RUNTIME-076 recovery DESIGN/SPEC

Status: BUILD

Root ORCHESTRATOR / WORK_ORDER_AUTHOR; independent root REVIEWER; Luna max /root/r076_worker documentary IMPLEMENTATION_WORKER. Risk R1 source-only documents. Owner 2026-10-11 continues roadmap and explicitly changes worker effort to max. Prior xhigh evidence remains unchanged.

INTAKE: R074 source audit identified unknown crash/restart/topology/stranded-row behavior; R075 closure remote SHA9181f3826f6a90db0042b25b9c955b253ea0828c verified 2026-10-11 by ordinary push exit0 (already up-to-date) and ls-remote exit0. DESIGN/SPEC deliverable follows; documentary BUILD is document authoring, not application BUILD.

Immutable R076 seed must be committed before active baseCommit/worker release. Worker rehydrates manifest/policy/state/memory/handoff/status/index and relevant learning; emits CVF declaration and messages ACK BEFORE edits, then waits for root release. Core remains read-only. Missing compact bootstrap nonblocking. Knowledge ingest root canonical-core fallback generated TEMP index, no POST.

Worker edits exactly three NEW files in seed: decision, intended spec, source identity JSON. Root owns work order, record, continuity, catalog, reviews, assessment and every commit. Worker does not run git add/commit/push or application Go/Vitest/build/DB/browser/provider/config reads. Static source/git reads only. Source baseline recorded in identity at release. No root draft of worker deliverables.

DR01: Source-grounded current admission, context, worker, abort, terminal/release and cancellation behavior; cite paths/line ranges and hashes, distinguish facts/inference/UNKNOWN.
DR02: Compare explicit single-process fail-closed/manual reconciliation, automatic reclaim, durable queue/lease choices; recommend a bounded next slice, explain alternatives and prerequisites without inventing deployed topology.
DR03: Never infer owner death from missing local owner or age. Model live owner, remote owner, stopped/fenced owner, partial effects, terminal uncertainty and crash between reservation/start, execution/terminal, terminal/notification. Preserve tenant isolation and Job->JobRun lock order.
DR04: Proposed safe recovery must require demonstrable old-writer exclusion/quiescence, exact run identity, checked conditional writes, uncertain outcomes fail closed; partial external effects and unresolved terminal writes cannot become success or automatic retries/checkpoint advancement. Preserve old run/receipt/results/history.
DR05: Specify identity/idempotency/retry/duplicate/cancellation semantics and explicit compatibility with HTTP/MCP since_last/default cap/date behavior, cron/after-sync/agent and current contracts. No promise of exactly-once provider or delivery.
DR06: Separate intended vs implemented; unresolved owner/topology decisions remain OPEN and later work cannot silently assume answers. Avoid designing an operator bypass that clears running rows while an old process may write.
DR07: Testable acceptance matrix with positive/fault/concurrency/tenant/conditional-write/partial-effect cases, named negative controls and restoration/cleanup. Label ALL future runtime evidence NOT_RUN. Mock output cannot prove AI governance.
DR08: Concrete next implementation scope prerequisites, path/artifact class, risk minimumR2 for runtime/persisted data/security; proposed budget/roles and failure boundaries only, no seed or runtime dispatch granted by this tranche. Provider/secret/external/release boundary stays separately authorized; accounts parked.
DR09: Decision and spec cross references resolve; material source facts tie to hashes/line ranges; no wholesale historical evidence replay/current-HEAD certification. Existing R074 limits and broader S2/S3/S5/live/governance OPEN retained.
DR10: Exact three-file artifact ownership, independent consolidated review, first-pass findings/repair rounds/root assistance recorded separately. At third repair without independent new root cause record REVIEW_COST_ESCALATION_REQUIRED and stop.

Root runs repository preflight/PR-range/gate units/catalog for metadata commits; no duplicate worker campaigns. REVIEW_PASS only after committed worker document snapshot and independent review; local FREEZE closes this design/spec packet only, never recovery implementation. Failure: source drift, unsafe recovery assumptions, missing prerequisites, uncited current behavior, scope/effect change.
