# R072 legacy UI optional-extension compatibility

Status: ACCEPTED_FOR_BOUNDED_BUILD_BY_DELEGATED_ORCHESTRATOR.

Root ORCHESTRATOR / SPEC_AUTHOR / WORK_ORDER_AUTHOR. INTAKE: accepted R071 emits optional usage_observation.adapter_usage_presence; existing frontend exactKeys(value, required) hides otherwise valid legacy usage. DESIGN: allow this named extension without interpreting or exposing it. R1 pure presentation structure compatibility, no governance or security decision. Existing authority/binding and unsafe-number rejection remain unchanged. New presence display and int64 browser representation require a separate future contract.

Acceptance:
- UC01: absent extension produces the unchanged old projection; a representative source-shaped R071 aggregate produces exactly the same full RunObservation output.
- UC02: optional extension contents are opaque/ignored, including null, malformed objects, future-version fields, primitives, arrays and private marker strings; none appears in projected output. This is no validation/acceptance claim about presence metadata.
- UC03: only this named usage-level key becomes optional. Other unknown usage keys, missing/invalid legacy fields, wrong legacy version/basis, malformed totals and unsafe integers still make usage unavailable.
- UC04: context and receipt tenant/job/run binding remain enforced. Known local zero remains zero; incomplete null totals remain null, never inferred from the ignored extension. Independent sections retain existing behavior.
- UC05: root never edits product/tests. Fresh Luna first source commit before tests; root reviews diff and new fixtures before runtime. Existing tests/backend/authority/old packets protected. Only one product parser change and one NEW test file.
- UC06: each role at most two Vitest invocations (expected one focused positive, reserve for accepted same-scope repair only) and one forced vue-tsc check; aggregate four Vitest/two typechecks, Go0. Capture exact commands/source/environment/native stdout/stderr/exit and actual test counts. No automatic retry. No mutation campaign required for this one-line low-impact compatibility repair; regressions and rejection tests must directly exercise the public projector.
- UC07: independent REVIEW before local compatibility-only FREEZE. Retain failed attempts, per-tranche Luna source/test findings and root assistance separately. No token/time/cost inference or matched Sol comparison.

Allowed source paths are the parser and NEW run-observation-adapter-compat.spec.ts only. No UI component, i18n, API, permission, backend, dependency, gate/workflow, network, provider, DB, credentials or publication changes. Full S2/S3/S5/globalF02, live governance, queue/real Analyzer, UI presence display/billing remain OPEN; Facebook/Zalo OA parked and prior R068 publication UNVERIFIED/not retried.

Bootstrap: doctor25/1 retains core pin/migration note; compact bootstrap absent BOOTSTRAP_MIGRATION_PENDING. Core read-only ab714ed matches cached origin/main. Knowledge README TEMP-only ingest, no POST. Root fixes stale R071 index BUILD description directly as known-value metadata residue. Parent read-only guessed ROADMAP/bootstrap/script paths failed and were corrected; zero source/runtime effects, tracked as root overhead.
