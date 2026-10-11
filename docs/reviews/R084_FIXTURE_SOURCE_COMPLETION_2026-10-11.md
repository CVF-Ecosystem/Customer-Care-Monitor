# R084 Fixture Source Completion Readiness — 2026-10-11

Status: worker source and portable unit acceptance pass; independent review and commit are pending. This is a readiness handback, not a tranche closure or runtime acceptance.

## Scope completed

The source now calls Campaign._validate_started_ownership() immediately after starting the exact container and before any MySQL call, including SELECT 1. It validates the inspected image, exact container and network identities, task labels, internal-network properties, and exact network membership. The same validation runs again after readiness and before the schema probe.

The tests preserve all 18 prior safety tests and add five controls. An instrumented event sequence proves that ownership validation precedes the first MySQL invocation and that mismatched actual image, container ID, NetworkID, network ID, labels, internal status, or membership results in zero MySQL calls. In-memory subprocess.run controls raise subprocess.TimeoutExpired on Campaign network/container creation and exercise Campaign.run() through DockerCli, exact-name reconciliation, cleanup, and unresolved receipts. A thrown container-removal failure also preserves identifiers and does not claim absence. These are local resource-harness controls; no Docker executable or daemon was invoked.

## SC01–SC05 evidence

- **SC01–SC02, ownership ordering:** test_started_ownership_validation_precedes_every_mysql_call checks start → container/network inspect → first SQL for the positive path, repeats post-readiness validation, and verifies zero SQL for each mismatch case.
- **SC03, raised create timeouts:** test_native_network_timeout_is_reconciled_or_retained_without_false_absence, test_native_container_timeout_reconciles_owned_resource_and_cleans_both_ids, and test_unresolved_container_timeout_preserves_receipt_and_cleans_known_network cover reconciled and unresolved outcomes through the campaign boundary.
- **SC04, cleanup failure:** test_thrown_container_remove_failure_keeps_ids_and_never_claims_absence covers a thrown removal error; the preserved tests continue to cover still-present resources, uncertain inspection, and known-resource cleanup behavior.
- **SC05, worker unit capture:** one invocation of python -B -m unittest scripts.tests.test_jobrun_fence_fixture -v ran using Python 3.13. Exit code **0**, **23 tests**, duration **3.606 seconds**, not timed out. Raw child stdout is empty (0 bytes); raw child stderr is 4,719 bytes and contains the verbose test names and Ran 23 tests ... OK summary. See [suite-execution.json](probes/r084_fixture_source/worker/source_unit/suite-execution.json), [stdout.txt](probes/r084_fixture_source/worker/source_unit/stdout.txt), and [stderr.txt](probes/r084_fixture_source/worker/source_unit/stderr.txt).
- **SC06, claim boundary:** this document maps source controls to portable test results. It makes no CVF governance or runtime claim.
- **SC07, independent review:** root review, source commit, and the separate reviewer suite remain pending.

The first launcher attempt failed before Python process creation because PowerShell returned multiple Python executable paths to Start-Process. It is recorded separately in [launcher-error.json](probes/r084_fixture_source/worker/source_unit/launcher-error.json); it is not a suite result or invocation. The corrected run above is the sole worker suite invocation.

Physical SHA-256 and byte length were captured before and after that invocation, separately from Git-filtered blob identity and HEAD. Both source and test files match across the captures. scripts/jobrun_fence_fixture.py is 72,611 bytes, SHA-256 be520b58742dc7b30f603d4bb5ff86f1986a2c30b48be3a3201256c82664b9c5, filtered blob a50401c790c13e15feba27bd1bcb1c7bfac4622a. scripts/tests/test_jobrun_fence_fixture.py is 41,950 bytes, SHA-256 2e566d49f218be2e0e288a2a36a5eb5d6805e089e404ff25fc789e7781f3d01c, filtered blob c913102d26a8a76d668cb9a9f02e86bd1f9800ea. Both remained dirty relative to HEAD e0d50430baea35b61e01b77f104dbaee7131decc; no source or test edits followed the unit capture. See the [pre-suite](probes/r084_fixture_source/worker/source_unit/pre-suite-identities-final.json) and [post-suite](probes/r084_fixture_source/worker/source_unit/post-suite-identities.json) identity records.

## Retained limits

The R083 independent review returned the original F1/F3 findings; its records remain unchanged: [R083 independent fixture safety review](R083_INDEPENDENT_FIXTURE_SAFETY_REVIEW_2026-10-11.md) and [R083 fixture prerequisite readiness](R083_FIXTURE_PREREQUISITE_READINESS_2026-10-11.md).

No fixture campaign, Docker/context/image/resource inspection, SQL, Go, provider call, external network action, credential read, image pull/install, or deployment occurred. Fixture availability remains **NOT_RUN**; actual image/settings/DDL remain **UNKNOWN**; fence integration remains **OPEN**; deployed status remains **UNKNOWN**. The passing portable suite does not establish fixture availability, CVF runtime governance, or deployment adoption. Root owns the independent review and commit decision.