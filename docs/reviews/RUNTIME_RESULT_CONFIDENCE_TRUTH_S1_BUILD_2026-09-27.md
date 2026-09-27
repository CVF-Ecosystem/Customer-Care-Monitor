# BUILD evidence: CCMAI-RUNTIME-005 — S1 result confidence truth

**Implementer:** Claude (`IMPLEMENTATION_WORKER`) · **Date:** 2026-09-27 · **Risk:** R2 · **Authority:** [SPEC](../specs/RUNTIME_RESULT_CONFIDENCE_TRUTH_S1_2026-09-27.md), [work order](../work_orders/CCMAI_RUNTIME_005.md), plus the owner-approved path addition recorded in the active handoff · **Status:** `REVIEW_PENDING` for independent Codex REVIEW. No FREEZE.

## Entry, block and scope change

Claude acknowledged `WORK_ORDER_AUTHOR (Codex) -> IMPLEMENTATION_WORKER (Claude)` at `5accbd1`. The pre-edit trace found that SPEC contract 1 (persisted `confidence = NULL`) forces `JobResult.Confidence` from `float64` to a nullable type. That breaks `backend/api/handlers/demo.go`, which is outside the allowed paths and which itself persisted invented confidence (0.92 / 0.88 / 0.90 / random 0.85–0.99). Claude recorded `BUILD_BLOCKED` with a bounded change request in local commit `c7ed39f`. It rejected the in-scope alternative, a dead `float64` field silently absorbing demo writes, because it would hide a behavior change.

The owner then approved the requested addition ("Bạn xử lý luôn đi"):
- `demo.go`, limited to its four `JobResult` literals;
- one compile-only fixture literal in `results_test.go`;
- a focused test in the existing `demo_test.go`.

Objective, R2 risk, external-effect ceiling, commit ownership and independent Codex REVIEW are unchanged.

## Trace (before editing)

| Surface | Finding |
|---|---|
| Writers | `engine/analyzer.go` `saveResults`: 4× fabricated `Confidence: 1.0` (QC evaluation, QC violation, classification evaluation, classification SKIP evaluation) plus 1× tag model value. `api/handlers/demo.go` `ImportDemoData`: 4 invented values. |
| Schema | Legacy `job_results.confidence` is already `double NULL` (checked by running HEAD's `AutoMigrate` on a fresh disposable schema). No ALTER is needed on that column. |
| JSON readers | `jobs.go` `ListJobResults` / `ListAllJobResults` (the two job-specific result paths), `conversations.go`, `agents.go`, `dashboard.go`, `mcp/handlers.go` all serialize `models.JobResult` directly and never name `Confidence`. A model-level change therefore covers them without editing them. `ListAllJobResults` embeds `JobResult` in `JobResultWithConvDate`, and GORM still runs the promoted `AfterFind` (proven by test). |
| Formatters | `notifications/dispatcher.go:158` printed a bare `(%.0f%%)`. The frontend type `stores/jobs.ts` declared `confidence: number`. `JobDetail.vue` does not present confidence, so it is untouched. |
| Validation | `validateAIResult` already rejects a missing or out-of-`[0,1]` tag confidence before any write. It is unchanged. |

## Implementation

- **Model (`db/models/job.go`).** The stored `Confidence *float64` (same `confidence` column) and a new `ConfidenceBasis *string` (`varchar(40)`, nullable) are `json:"-"`. JSON output comes from derived, non-persisted `ReportedConfidence *float64 json:"confidence"` and `ReportedConfidenceBasis string json:"confidence_basis"`, set in `AfterFind`. A number is exposed only for `classification_tag` + basis `model_reported_uncalibrated` + value in `[0,1]`; everything else yields `null` / `unavailable`. Because the exposed fields are separate from the stored ones, a later `Save` of a loaded row writes the stored values back unchanged and cannot rewrite history (tested). There is no `calibrated` basis value. `UnavailableConfidence()` and `ModelReportedConfidence(v)` build the stored pairs.
- **Analyzer.** The four fabricated `1.0` writes now store `NULL` / `unavailable`. Tags store the validated model number with `model_reported_uncalibrated`. Verdict, score, evidence refs, snapshot binding and the single transaction are unchanged.
- **Demo writer (owner-approved).** All four demo literals store `NULL` / `unavailable`, tags included, because a random demo number is not model-reported. The invented `{"confidence": …}` tag `detail` JSON is now `{}`.
- **Notifications.** A tag line shows a number only when it is exposed, worded as `— mô hình tự ước lượng NN%, chưa hiệu chuẩn`. Legacy, invalid and unknown values show no percentage.
- **Frontend (`stores/jobs.ts`).** The type is now `confidence: number | null` plus `confidence_basis: 'unavailable' | 'model_reported_uncalibrated'`. No component defaults it to 0 or 100%.
- **Compile-only fixture edits.** `Confidence: 1` was removed from `api/handlers/results_test.go` (`themKetQua`, owner-approved) and `engine/snapshot_db_test.go` (in scope). Neither asserted on confidence.

## Tests added

| Test | Proves |
|---|---|
| `engine/confidence_db_test.go` `TestQCResultsStoreNoFabricatedConfidence` | New QC evaluation and violation rows store `NULL` / `unavailable` in the raw columns, expose `nil` / `unavailable` on read, and stay snapshot-bound. This is the central regression. |
| `TestClassificationTagKeepsOnlyModelReportedUncalibratedValue` | A tag stores and exposes `0.7` / `model_reported_uncalibrated`; the classification evaluation is `NULL` / `unavailable`. |
| `TestClassificationSkipEvaluationHasNoConfidence` | The no-tag SKIP evaluation is `NULL` / `unavailable`. |
| `TestInvalidModelTagConfidenceIsRejected` (above one / negative / missing) | Existing validation rejects each case, and no result or snapshot row is written. |
| `TestLegacyAndMislabeledConfidenceIsNotExposed` | Legacy rows (QC 1.0, tag 0.73, NULL basis), a `model_reported` basis on a non-tag row, an out-of-range model value and an unknown `calibrated` basis all expose `nil` / `unavailable`. After `Save`, the stored value and basis are unchanged. |
| `api/handlers/job_results_confidence_test.go` `TestJobResultEndpointsSerializeConfidenceTruth` | The real `ListJobResults` and `ListAllJobResults` handlers return an explicit `confidence` key: `null` for legacy QC, legacy tag and new unavailable rows; `0.64` only for the model-reported tag; the matching `confidence_basis` for each. |
| `api/handlers/demo_test.go` `TestImportDemoDataStoresNoInventedConfidence` | Every row from the real `ImportDemoData` stores `NULL` / `unavailable` and serializes `null`. The test asserts evaluation, violation and tag rows exist, so it is not vacuous. |
| `notifications/dispatcher_test.go` `TestNotificationBodyNeverShowsBarePercentage` | Exactly one percentage, labeled as the model's own uncalibrated estimate. Legacy, invalid and unknown tags show none. The old `(NN%)` form is absent. |

**Regression proof (mutation).** The analyzer's unavailable pair was temporarily replaced with the old fabricated `1.0` (basis NULL). `TestQCResultsStoreNoFabricatedConfidence` and `TestClassificationSkipEvaluationHasNoConfidence` then failed with `stored confidence={1 true} basis={ false}, want NULL/unavailable`. The file was restored from a backup, and a check confirmed no mutation marker remained.

## Migration and rollback

Disposable `mysql:8.0` container `ccma-r005-db` on isolated network `ccma-r005-net` (no host data), with two schemas: `CCMA` (pre-existing table) and `CCMA_FRESH`. `log_bin_trust_function_creators=1` was set via root.

1. HEAD's (`c7ed39f`) `AutoMigrate`, run from a temporary git worktree, built the legacy schema on `CCMA`: `confidence double NULL` and no basis column. Four legacy rows were seeded (QC evaluation 1.0, violation 1.0, tag 0.73, classification evaluation 1.0).
2. The new `AutoMigrate` ran **twice** on `CCMA` and **twice** on `CCMA_FRESH`, with `automigrate ok` all four times. Both schemas end with `confidence double NULL` and `confidence_basis varchar(40) NULL`. All four legacy rows kept their exact values, with a NULL basis. The fresh schema had 0 rows.
3. **Rollback:** the old binary's `AutoMigrate` against the new schema succeeds and keeps `confidence_basis`, since GORM does not drop columns. The old binary reads both legacy rows and a new-style `NULL` / `unavailable` row without error. **Limit:** old code scans NULL confidence into its `float64` as `0`, so after a rollback, new rows would show `confidence: 0` through the old API until rolled forward again. Rolling forward is safe: the basis column already exists and no stored value was rewritten.
4. `go.mod` / `go.sum` were unchanged. The temporary migration command was built outside the repository and removed; the worktree was removed.

## Verification

This host's Windows Application Control policy began blocking freshly built Go test binaries mid-session, both in `%TEMP%\go-build*` and in the scratchpad (`An Application Control policy has blocked this file.`). The policy was not bypassed. Tests ran in `golang:1.26-alpine` (go1.26.8) on the disposable MySQL network, with the host module cache mounted read-only, `GOPROXY=off` and `GOFLAGS=-mod=readonly`. This is the same containerized approach used by earlier tranches.

```text
host: go build ./... ; go vet ./...                                   clean
container: go test ./engine ./api/handlers ./notifications -run '<R005 + related snapshot/demo/validation tests>' -count=1 -v
    engine 10 top-level PASS (incl. 3 invalid-confidence subtests); handlers 4 PASS (incl. both endpoint subtests); notifications 1 PASS
container: go build ./... && go vet ./... && go test ./... -count=1 -p 1   all 13 packages ok
mutation (fabricated 1.0 reintroduced)                                   2 regressions FAIL as expected; source restored
frontend: npm run build (vue-tsc -b && vite build)                       PASS
gofmt -l (LF-normalized) on all changed Go files                         clean for every changed line; demo.go's 6 pre-existing hunks do not overlap this change (verified against HEAD)
scripts/manage_cvf_downstream_catalog.ps1 -Check                         PASS
check_cvf_workspace_agent_enforcement.ps1 -ProjectPath .                 PASS 25/25
git diff --check                                                         clean
```

Cleanup: the disposable container, network, temporary worktree and scratch binaries were removed. The persistent Compose `ccma` stack (`ccma-app-1`, `ccma-db-1`) was not started, reset or touched.

## Remaining limits

- The rollback shows NULL confidence as `0` under the old binary (see above).
- The aggregate Results page never displayed confidence and was not changed. Other direct `JobResult` JSON consumers (`conversations.go`, `agents.go`, `dashboard.go`, `mcp/handlers.go`) get the new semantics through the model but have no dedicated endpoint tests; only the two job-specific paths named by the SPEC are tested end to end.
- Demo tags lose their invented confidence entirely instead of carrying a labeled number. This is deliberate.
- No calibration, human disposition, provider admission or probability claim is made. This is not live CVF governance proof: no provider API, credential, channel sync, customer data, deployment or push was used.
