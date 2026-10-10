# R063 bounded work-order draft — Rule input observation

Date: 2026-10-08. Status: WORK_ORDER_DRAFT_READY_FOR_SEED. Root ORCHESTRATOR / WORK_ORDER_AUTHOR / independent REVIEWER. Metadata risk R1, future source risk R2. [SPEC RO01..12](R063_RULE_AUTHORITY_OBSERVATION_DESIGN_SPEC_2026-10-08.md) at planning commit `ff9e985`; accepted source `d10e164249b5e7a87206d1d3bc5aec43a1a8ac2f`. R062 remains active/FROZEN. This draft is not an executable work order, immutable seed or source/runtime dispatch.

## Consolidated audit disposition

Accept exact effective-rule byte fingerprint in existing tenant/job-scoped Summary, with residual historical equality/dictionary inference recorded. `backend/api/handlers/jobs.go` ListJobs returns Job including current rule fields; ListJobRuns returns JobRun including Summary, both existing jobs:r routes. No new endpoint, permissions, public export, logging, raw rule retention or credential requirement. Existing readers may see historical digest even after rule update; do not describe it as anonymized or approved. This bounded R2 observation adds no authentication enforcement. A keyed digest would introduce secret lifecycle and a new scope; not selected.

Current Analyzer consumes its copied Job, but ordinary unchanged-selection checks local source snapshots rather than this proposed fingerprint. A rule update can coexist with a no-inference unchanged run under current behavior. Receipt records observed input even in no-work runs and never says the rule was submitted/evaluated. No selection/cache invalidation in this tranche; separate future semantic contract needed.

Exact scalar compatibility audit found only TestSPOrdinaryExplicitAndProgressPersistence and TestEXAllExplicitModesRetainScalars counting keys after deleting the two old receipts. Both retain every current assertion. Add exactly four lines after their existing `delete(scalars, "source_execution")`, once per file:

```go
if _, ok := scalars["rule_observation"]; !ok {
    t.Fatal("missing rule_observation in Analyzer summary")
}
delete(scalars, "rule_observation")
```

No edits to any other old-test line, fixture/helper, count/assertion, detector, tolerance or skip. All future RO fixtures require valid UUID parents/children, checked callback registration/removal and exact durable-reload assertions. Existing R060/R061 fault-test repairs stay byte-identical apart from that EX four-line block. No retroactive old packet certification.

## Proposed execution ownership and phase route

Owner-selected route inherited: Codex `/root/r059_worker`, model gpt-6.1-sol, reasoning medium, IMPLEMENTATION_WORKER / REPAIR_WORKER / BUILD COMMIT_STEWARD; Codex `/root` dispatcher/WORK_ORDER_AUTHOR and independent REVIEWER. Root canonical source/test edits zero. Reuse child only after dispatch; this draft does not call it. Source review must remain independent. Root directly fixes small authorized metadata.

Before BUILD: create formal `docs/work_orders/CCMAI_RUNTIME_063.md`, dispatcher-owned `CVF_SESSION/authority/CCMAI-RUNTIME-063.json` with matching R2 scope/roles/prohibitions and first-commit seed; validate schema/history, commit seed before activation. Active record baseCommit must contain original seed, not a moving branch placeholder. Activate WORK_ORDER with matching state/front marker/handoff/order/implementation/index; freshly rehydrate and declare each role. Worker records and commits before-edit BUILD acknowledgment; no edit of immutable seed. Record INTAKE/DESIGN/SPEC/WORK_ORDER trace, inherited accepted source and no source proof yet.

Allowed implementation paths, to be copied exactly into seed:

| Path | Limit |
|---|---|
| `backend/engine/analyzer.go` | Initialize immutable observation after valid consumed reservation, thread through existing single/batch/early/panic Summary writers; no execution-policy/lifecycle changes. |
| `backend/engine/source_preparation_receipt.go` | Composer only: optional variadic rule receipt preserves existing direct three-argument pure-test usage; production writers supply explicit captured receipt. No preparation collector change. |
| `backend/engine/rule_observation_receipt.go` | New fixed-enum/UUID/framed-hash bounded immutable receipt; no DB/provider/network/config access. |
| `backend/engine/rule_observation_receipt_test.go` | New pure tests, independent fixed hash vectors, framing/privacy/limits/freeze. |
| `backend/engine/rule_observation_receipt_db_test.go` | New integration observations for actual prompts, copied rules, no-work/setup/error/panic, Summary reload and write faults. |
| `backend/engine/source_preparation_receipt_db_test.go` | Only the four-line block above, one insertion. |
| `backend/engine/source_execution_receipt_db_test.go` | Only the four-line block above, one insertion. |

Formal order063 plus session/status/review/probe/catalog/roadmap metadata permitted under established gate artifact classes. Protect all other backend files, all old tests outside the exact two blocks, workflows/tooling/dependencies, old seeds/evidence and CVF core. No provider adapters/prompts/model schema/API/permission/scheduler edits. Product can change only observation/composition, no altered provider calls, usage writes, output validation/publication, scalar counters, selection/retry/checkpoint/cancel/ownership/notification behavior. No rule enforcement, WAIT_DATA state, semantic auto-skip, new provider/channel/secret/config/customer/persistent database use, external fetch, push/merge/deploy, full-S2/live-governance/hosted claim or FREEZE.

## Finalized receipt contract

Use SPEC framing exactly: domain, recognized type, effective byte fields, each length uint64 big-endian; SHA-256 lowercase hex. No source/prompt/schema version implied beyond receipt/domain algorithm versions. Binding metadata invalid has priority: suppress all identifiers and fingerprint, status METADATA_INVALID. For valid bindings unsupported type emits no raw type/fingerprint and UNSUPPORTED_JOB_TYPE; recognized types produce OBSERVED, including empty bytes. Fixed policy/permission/WAIT_DATA unavailable statuses and JOB_INPUT_OBSERVED label are observations of missing provenance, never authority grants. Job type is fixed allowlist only; invalid IDs cannot leak arbitrary input. Total receipt serialized at most 2 KiB, deep immutable copy. Existing source receipts untouched in shape/meaning.

RO01..12 remain requirements. Integration fixtures observe both exact supplied system prompt and expected digest; test DB rule edits after Job capture without rerunning selection. Tests distinguish initial/progress/final/returned and durable prefix, status-only terminal fallback success and failure. Do not assert running merely because Summary failed. Every callback error checked and removed before fixture cleanup. Reservation rejection/reuse remains no effects/no newly fabricated receipt. Unknown job_type existing behavior retained.

## Bounded evidence campaign and cost

Fresh R063 budget is proposed, not borrowed from R059–R061: worker max1 campaign/4 Go, independent root max1 campaign/4 Go, total max2 campaigns/8 Go. No exploratory Go build/vet/test outside these invocations. Root/static Python/catalog/docs checks outside Go budget. A `go test` command's default compile/vet is not a separately executed vet claim.

| Invocation per role | Required evidence |
|---|---|
| 1 | Positive group: all new TestRO, all existing TestSP/TestEX and source-derived preparation/provider-lazy/ownership/finalizer controls. Before invocation record exact inventory/regexp, expected names and no DB skip. No unbounded full-backend rerun. |
| 2 | M01 source-only isolated mutation changes QC effective fingerprint input, omitting SkipConditions. Named independent known-vector/effective-field detector fails semantically; include a healthy contrasting classification/control case. |
| 3 | Restore baseline then M02 erases rule_observation only from stored terminal Summary while returned receipt remains intact. Named DB reload detector must fail semantically. |
| 4 | Restore exact complete archive bytes and run all new TestRO plus mutation targets; PASS, zero unexpected skip. |

Exact mutation edits/diffs/hashes recorded after source exists and before control; tests/detectors unchanged. Mutations only in disposable exported source, not canonical worktree. Compiler/panic/setup/timeout failure is INCONCLUSIVE, not killed. Root selects and verifies its own controls at exact committed worker source. Test names above prefixes are planned; final names/inventory fixed before first invocation, not invented successful counts.

Cached offline images/modules only; private task-internal disposable MySQL, no host ports, readonly baseline archive/module cache, fresh task compile cache. Record original/sandbox 210-plus-new-file manifests in their own byte domain, physical/hash protection and source/plan identifiers; compare full archive members, not Git LF against physical CRLF. Disable module/proxy/checksum downloads. Positive timeout480s/attach600s; focused controls180s/attach300s; DB readiness90s. Capture partial JSON Go completion logs/stdout/stderr/commands/states/exits/raw hashes on failure, then verified exact-resource cleanup, preserving live ccma resources. No implicit restart, retry or budget reset after failure.

If any stage fails, retain completed counts and NOT_RUN controls, record root cause and same-scope repair/cost disposition before continuation. At repair round three without independent new root cause, record REVIEW_COST_ESCALATION_REQUIRED. Scope/risk/roles/runtime budget boundary changes need explicit recorded successor authority. Routine same-scope metadata fixes continue without operator wait.

## Handback and review gate

Worker source commit then REVIEW_PENDING handback includes exact source and base, source/test diff, protected old files and exact eight-line compatibility proof, RO mapping, positive/mutant/restored receipts, restoration/second archive/cleanup and failures. Root independently audits contract/source/raw outcomes/old behavior/claim boundaries; REVIEW_PASS only if all scoped requirements met. Neither worker self-review nor this draft grants FREEZE. Separate bounded closure authority follows accepted independent review.

Before commits default preflight, resolved origin/main PR range, exact staged paths, catalog PS5.1/7, diff/protection, docs build after final Markdown; required gate-unit tests. Planning checks are repository evidence, not runtime governance proof. All external provider/API/channel/live credentials/customer DB/core/push/merge/deployment tests NOT_RUN. Facebook/Zalo OA parked. Full S2/global F02/provider/billing/permission-policy enforcement/WAIT_DATA/live/CVF-governance/hosted readiness remain OPEN.

Disposition: consolidated audit settled fingerprint exposure, exact compatibility path class, independent worker/reviewer route and finite budget in a reviewable draft. No owner clarification needed within standing autonomous planning authority. Next governed move is seed/schema/prebuild audit and formal WORK_ORDER activation; source changes remain zero until that sequence completes.
