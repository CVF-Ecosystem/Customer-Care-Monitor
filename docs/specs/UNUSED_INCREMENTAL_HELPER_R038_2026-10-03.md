# R038 — Remove uncalled incremental-mode helper

Status: SPEC_READY; BUILT, REVIEW_PENDING.

Date: 2026-10-03. Risk ceiling R1, no runtime change. Baseline9bfecedaeeeff4a586ac2685871ad807976deabf; immutable seed `1d6c1e1ededc8f38afe149c6ba334ba8edf9f301`. [Order](../work_orders/CCMAI_RUNTIME_038.md).

## Intake and design

R027 independent reviews recorded `isOrdinaryIncremental` as nonblocking dead code outside their allowlist. At this baseline repository Go-source search finds only its definition and comment in `backend/engine/analyzer_incremental.go`, no caller/test use. Current selection uses explicit runPlan/mode behavior, preserved by this cleanup.

Delete only the private function declaration/body and its directly attached three-line comment (and separating whitespace as needed). Do not replace it, add callers, infer modes from flags, change any executable statement/import or rewrite surrounding comments. The time import remains used elsewhere. If a reference appears in the current source, stop BUILD_BLOCKED and return the discrepancy instead of changing callers. Search current tracked Go/assembly/source inputs, excluding historical documentation from executable-use counts.

| ID | Required local evidence |
| --- | --- |
| UH-01 | Before: one declaration, zero executable/test references. After: symbol absent from source. Exact diff removes only helper/comment, surrounding file unchanged. |
| UH-02 | Existing backend and engine test sources still compile: cached Go build/vet all packages and compile-only engine test binary. No test binary run or new tests/mutations required. Zero executed tests must not be described as a passing regression suite. |
| UH-03 | Mode/snapshot/checkpoint/provider/admission/terminal/cancellation behavior and all other source/test files unchanged. Preserve R025/R027 historical acceptance and limitations; docs/catalog/gates/diff pass. |

Compilation/static evidence only; no functional DB/provider regression claim. No credentials/.env/config, runtime/Analyzer/DB/Docker/provider/channel/network action. Missing cached Go/docs prerequisites -> BUILD_BLOCKED, no downloads. Existing mandatory gate unit tests run, no new low-value test mirroring a deletion.

## Current truth

*Historical (dispatch-time): DISPATCH_READY / NOT_BUILT, helper present, no worker started.* **Current:** Claude BUILD removed only `isOrdinaryIncremental` and its comment from `backend/engine/analyzer_incremental.go` (evidence: [BUILD record](../reviews/UNUSED_INCREMENTAL_HELPER_R038_BUILD_2026-10-03.md)); cached Go build, vet and compile-only engine test build exit 0, zero tests executed; REVIEW_PENDING, not independently accepted. Source search only in planning; Go build/vet/test compilation NOT RUN here. Existing local acceptances/NOT RUN/effect limits and R033 FREEZE unchanged. Live Pancake and actual MCP execution remain separate. No runtime CVF/governance/hosted/new FREEZE claim.
