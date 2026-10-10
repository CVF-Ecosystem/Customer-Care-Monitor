# R059 pre-BUILD contract finding and concrete successor proposal

Date: 2026-10-07. Root independent reviewer/orchestrator source audit, corroborated by worker `/root/r059_worker`. Source edits0, BUILD acknowledgment commits0, campaigns0, Go0. Seed `f162e635d757b2e436d88e0b8fda9ab5840e478f` and activation `2c5fcc9fda6f935353db62fdd8c3f7d1cc9173e4` preserved. Worker remains gpt-6.1-sol / medium; root independent reviewer. No role/budget/provider/network expansion proposed.

## Consolidated finding

Existing `TestSPOrdinaryExplicitAndProgressPersistence` in `backend/engine/source_preparation_receipt_db_test.go:108` removes only `source_preparation` before enforcing exact legacy scalar key counts1(initial)/5(progress/final). Required top-level `source_execution` necessarily makes those counts2/6. The current R059 seed protects all existing tests with prohibited effect `existing_test_edit`; the DESIGN/SPEC/draft requires byte-identical old tests. These requirements cannot all hold for the additive observation change.

Root inspected the single/batch Analyzer call sites, Summary composer, receipt-aware early helper and all existing engine tests matching Summary equality/map-count patterns. Only this old integration test has the incompatible top-level scalar count assertion. The pure preparationSummary scalar test calls the unchanged legacy composer and needs no edit; legacy close/Abort sentinel and stored `{}` assertions remain protected. Worker independently confirmed the same cause. No Go reproduction was run or claimed; the contradiction follows directly from the existing test and required serialized key.

Do not hide execution receipt, move it inside preparation, detect test callbacks, change scalar keys/counts, weaken existing preparation assertions or silently widen R059's seed. The compatibility failure is a pre-build contract/path boundary, not a worker source defect or evidence retry.

## Proposed minimal old-test maintenance

Exactly one existing test file would be additionally allowed: `backend/engine/source_preparation_receipt_db_test.go`. Exactly one location in the scalar-verification loop would change:

```diff
             delete(scalars, "source_preparation")
+            if _, ok := scalars["source_execution"]; !ok {
+                t.Fatal("missing source_execution in Analyzer summary")
+            }
+            delete(scalars, "source_execution")
             if scalars["conversations_found"] != float64(1) {
```

All original scalar key counts/values, frozen preparation-receipt equality, snapshot/provenance/provider-call/old-fixture assertions and old test name remain unchanged. The added check requires the execution receipt in every observed Analyzer initial/progress/final summary. Dedicated new EX tests still verify its complete structure and semantic facts. This patch separates two authorized receipt members from scalar fields without permitting arbitrary extra keys. No generalized ignored-key list or scalar count relaxation.

## Proposed authority disposition

If owner accepts this exact path/assertion-maintenance boundary: preserve original R059 seed/activation as historical undispatched source packet; commit a new bounded successor seed/work order/tranche under R060 before BUILD. R060 inherits EX01..12, accepted baseline, same worker gpt-6.1-sol medium and independent root reviewer; additionally permits only the patch above in the one old test file. Every other old test remains byte-identical. The original R059 pending source record is parked/superseded, not certified or closed. No source/test change under the old seed.

Retain campaigns0/Go0; aggregate worker max1/4Go plus reviewer max1/4Go remains8 total across R059/R060, no budget reset. Require fresh canonical rehydration/declaration and committed before-edit BUILD acknowledgment under the new seed. Worker campaign, two applied mutations and restored controls unchanged. Source compatibility and exact patch are independently reviewed. No provider/channel/config/credential/customer/persistent DB/core/push/merge/deploy/FREEZE or full S2/governance/hosted claim.

Owner directs root to audit and choose rather than request routine confirmation. Root consolidated independent audit chooses the exact4-line patch above as test-maintenance for the additive contract; no legacy scalar/preparation assertion is removed. R059 is parked with source/build not started; a new R060 seed authorizes only this narrow old-test amendment before BUILD. This document records the decision; it does not apply the source/test patch. Existing local R058 closure and Facebook/Zalo OA parked checkpoint remain unchanged.

Publication setup note retained: the first claims preflight treated PREBUILD in this filename as BUILD evidence and rejected wording about a historical/nonexistent frozen state. Root changed that wording to closed without changing any disposition or gate; first docs build7.70s PASS. No runtime executed. Avoidable operator wait recorded: root asked before applying its consolidated test-maintenance judgment; owner instructs autonomous audit/disposition. Root continues via immutable successor authority and preserves source independence.
