# R156 — incremental historical audit/action join proposal

Status: CLOSED
workOrderStatusGrammar: cvf.workOrderStatusGrammar@1.0.0
Risk: R2
Authority: `CVF_SESSION/authority/CCMAI-RUNTIME-156.json` committed at `0ea490f02f993d396def1cda32215dd1212cbad2`
Worker: Sol 6.1 Medium IMPLEMENTATION_WORKER, proposal only
Reviewer: Root independent R2 REVIEWER of Sol implementation
Design reviewer: Terra 6.1 High, independent of Root seed/packet and future Sol implementation
Commit steward: Root

## INTAKE / DESIGN / SPEC — explicit incremental decision

User authorizes resolution, Sol6.1 Medium fallback when difficult, Root orchestrator/reviewer. R155 partial initial implementation is independently NOT_ACCEPTED_AS_FIX; no historical join, current/historical registry conflation, weak provenance-only fixtures. Original3workerinvocations and active/cost UNKNOWN remain spent. Root chooses this single necessary incremental implementation after source-backed review, not new discovery or automatic allowance renewal. R155 artifacts remain byte-identical, R155 BUILD_BLOCKED. Record: docs/reviews/R155_INDEPENDENT_IMPLEMENTATION_REVIEW_2026-11-06.md and CVF_SESSION/history/r155-parent-delivery-20261106/r156_seed_review.md.

Consumer/outcome: parent CVF maintainer receives an executable, reviewed but UNINSTALLED candidate implementing the historical audit/action join while preserving all current controls. Same existing RC-01, still BLOCKED until parent acceptance, verified delivery, deliberate authorized adoption and original required range PASS. No new criterion or product milestone. Root owns decision/authority/review/commit; Sol implements; Terra independently reviews this packet. SameR2/project-only effects/commit owner, user fallback applied expressly.

Pinned public Core c4c8767 remains read-only; installed profile1.3 lock/pins unchanged; parent ee68e90d3 has nofix. Reuse Root verified56hashedfiles/manifest receipt and project-development/form mapping from R155. Do not install1.4. Public doctor latest FAIL26/28 exactly LOCK_INVALID/COV_OBJECT_SHAPE version mismatch; no fullPASS. All R139/R151/R150 budgets/UNKNOWNs and FB/ZaloOA parking persist. This static candidate work is independent of those runtime paths because it neither executes nor changes them.

## Exact implementation scope / locked requirements

Write only `docs/reviews/probes/r156_historical_join_20261106/**` and `docs/reviews/R156_WORKER_RETURN_2026-11-06.md`. Copy dependencies from installed bundle verbatim with hashes; change copies of EXACTLY the three pinned runner, coverage and repair-convergence modules, plus at most one strict candidate helper if needed. No installed or parent mutation, no redefinition of default controls, no copying secrets or full Git trees. Include minimal unified diff, README integration steps and tests. Candidate format is proposed, not recognized live authority.

Implement the missing join directly. Reuse strict current validator primitives and admission semantics. Historical context must be explicit per retrospective candidate and unavailable to pre-dispatch/worker-return/bootstrap, activeWorkOrders, or explicit execution packets. Exact checked head equals checkout; unknown/nonancestor commits and ambiguous bindings fail closed. Do not use a FROZEN label as sole admission.

Historical binding exact schema ties milestone/tranche, packet LFdigest, admission/build/terminal commits, historical registry/audit, immutable seed and terminal record, independent review evidence and immutable binding/review provenance. Bind current checked head as context, not mutable authority. Validate every binding, not only selected one; reject duplicate/nonfinite/unsafe/symlink/oversize data. Use bounded Git object size check BEFORE reading stdout (1MiB/file, finite total), regular-blob modes, exact paths and timeouts. Historical registry comes from admission Git object, not today's registry file. Retained old audit/evidence still use existing pinned file checks; if those actually drift, explicit UNSUPPORTED, no broader virtual resolver.

Always validate current registry/current audit/evidence and repair history transition, oldhistory-prefix/currentaffected-superset, current cost UNKNOWN and unresolved-product guards. Then load historical admission registry and require it pins the exact historical audit. Parse/validate seed, terminal tranche/FROZEN/CLOSED/build/review and their provenance; account for R150 CLOSED status carrier rather than incorrectly requiring terminal record.workOrder equal historical admitted packet. Existing seed scope and old audit admittedAction must authorize exact packet. Validate all old selected/kind/owner/output/packet/question/budget joins under existing semantics. Substitute ONLY historical action matching for the eligible retrospectively frozen packet; default/current R151 and unrelated malformed current audits stay strict. No broad exemption or fallback.

Actual retained anchors: admissionc61b01fa89c40135093103a0852de70f82bd39ec, build7052df5c6da31d6867172d26a298a7bb4b72bb9c, terminaled9d55b25290fd8b640f16e0635fbe895e9b9b5f. ec6eed18a5f1951d9dc54a5b49d14f804574ab67 is WRONG registry/audit and mustdeny. R150 packet normalizeda7a2b74e1851cdcbd2dba010456e2b0d17a9ade1fdb83518e19d66343a79e398; old/current audit files retained. Current M3 registry remainsR151. Base original rangef7405c470207dc0d3be64e6eca79b7050d3f79d8 and original failure remains in R155baseline; do not rerun baseline as extra discovery.

## Finite cumulative allowance and required evidence

Prospective Sol active ceiling45minutes, measured from fresh declaration; stop if not able to meter it. At most12 additional semantic files after mandatory startup. Two additional candidate test invocations maximum: one consolidated suite, one justified targeted rerun if correction needed. Compile checks count. R1553used invocations + R156atmost2 = cumulative5workerinvocations maximum, not reset. R155 actualactive UNKNOWN, cumulativeactualactive UNKNOWN; nominal75+45 is not verified total compliance or remaining75. Root independent reviewer has at mostone final candidate suite, distinct from worker budget. No automatic further successor after exhaustion. Return BLOCKED with artifacts if unresolved; do not water down requirements.

One consolidated suite must include actual retained c61 historical positive with currentR151 registry unchanged, ec6wrong anchornegative, strict currentR151/default behavior, execution/active/explicit denial, forged freeze/unknowntranche, wrong head/nonancestor/order, changed packet/audit/seed/terminal/review, action/question/budget/criteria mismatch, duplicate/unsafe/symlink/oversize/nonfinite, unrelated malformed current audit, erased/reorderedhistory/UNKNOWNcostreset and unresolved productCOMPLETE. Include a synthetic both-parent merge/collision fixture that retains separate current authority and historical admission. The positive cannot merely return UNSUPPORTED or test constants. Test real copied-module integration, not a replacement callback pretending to be existing validator. Label STATIC_CANDIDATE_ONLY and distinguish methods/subcases. Raw commands/output/exits/counts/timing/source hashes and all failures retained. Artifact completion is not installed/gate/product acceptance.

Python -B and isolated static Python/Git fixture subprocesses only. Fixture Git init/add/commit allowed only in worker-owned disposable repositories under the proposal directory or ignoredruntime child. Worker never stages/commits project. Prefer deterministic cleanup avoiding Windows read-only objects; use ignoredruntime fixture directory if cleanup uncertainty, retain it without destructive workaround. No application/provider/browser/DB/network/credentials/channel/customer/push/merge/deploy/release effect or changing outside project.

## Return/review/parent boundary

Fresh worker rehydrates canonical summary/state/memory/active handoff/status/index and declares before code. Root commits BUILD marker first. Worker returns honest functional outcome and UNSUPPORTED boundaries; Root independently reviews exact code/diff and actual negative controls. Parent owns final schema/source acceptance and verified delivery; project upgrade/lock/workflow is separate future authority. No parent files/messages changed here. R154/RC-01 and product remain blocked even if candidate tests pass. Root produces one existing-folder parent learning intake linking R155 failures and R156 result, with no public claim or dispatch to others.

## Gate-To-Role Closeability Contract

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
returnTimeRecheck: REQUIRED_BEFORE_REPAIR
implementationTopologyPolicy: BOUNDED_PATH_FAMILIES
foreseeableFileSplitDisposition: COVERED_BY_BOUNDED_PATH_FAMILY

| gateId | mustPassBy | repairOwner | repairPhase | mutationSurface | topology | commitOwner | commitPhase | dependsOn |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| authorization_review | PRE_DISPATCH | orchestrator | PRE_DISPATCH | committed R156 seed and RC-01 scope | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | NONE |
| pre_dispatch_gate | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R156 packet and RC coverage | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | authorization_review |
| dispatch_continuity | IMPLEMENTATION | session-sync-steward | IMPLEMENTATION | CVF_SESSION/handoffs/AGENT_HANDOFF_R156_HISTORICAL_ADMISSION_2026-11-06.md material-SHA marker | EXACT_PATHS | session-sync-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| focused_checker_tests | WORKER_RETURN | implementation-worker | WORKER_RETURN | baseline failure and uninstalled candidate tests | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| adif_integrity | WORKER_RETURN | implementation-worker | WORKER_RETURN | retained failures and unsupported boundaries | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests |
| pre_implementation_autorun | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | fresh R156 worker rehydration/declaration | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| worker_return_fast | WORKER_RETURN | implementation-worker | WORKER_RETURN | proposal-only return and costs | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests,adif_integrity,pre_implementation_autorun |
| reviewer_fast | REVIEW | reviewer | REVIEW | Root independent candidate review | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | worker_return_fast |
| pre_commit | PRE_MATERIAL_COMMIT | reviewer | PRE_MATERIAL_COMMIT | exact scope and continuity | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | reviewer_fast |
| terminal_completion_review | POST_MATERIAL | reviewer | POST_MATERIAL | proposal only, RC/product blocked | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | pre_commit |
| continuity | CONTINUITY_COMMIT | session-sync-steward | CONTINUITY_COMMIT | state, handoff, memory, status and catalog | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | terminal_completion_review |
| committed_range_closure | POST_MATERIAL_CLOSURE | reviewer | POST_MATERIAL_CLOSURE | R156 committed range only; original failure retained | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | continuity |

## Complete existing RC coverage

RC-01 is the sole existing RC criterion; it remains BLOCKED. Proposal work never substitutes for original gate or M1–M5 acceptance.

```cvf-milestone-coverage-json
{
  "schemaVersion": "cvf.milestoneCoverage@1.0.0",
  "authority": {
    "path": "docs/reviews/planning/CCMA_REPOSITORY_CONTINUITY_2026-11-06.md",
    "heading": "## Repository continuity criterion",
    "sha256": "9bddf80ac48026e8409a487f2e7882ca3e01c1d93ff9b8e6292331a1d464a86f"
  },
  "milestone": {
    "id": "RC",
    "consumer": "CCMA Owner, next developer and parent CVF maintainer",
    "outcome": "One coherent development history with preserved R150 provenance and checked required gates",
    "checkpoint": {
      "trigger": "R156 actual joined candidate return or added cap exhaustion",
      "owner": "Root ORCHESTRATOR",
      "decision": "Review uninstalled proposal, prepare precise parent intake, keep RC-01 blocked until reviewed delivery/adoption and original range pass"
    }
  },
  "phase": "DISPATCH",
  "action": {
    "kind": "IMPLEMENT",
    "selected": [
      "RC-01"
    ],
    "owner": "Sol 6.1 Medium IMPLEMENTATION_WORKER, proposal only",
    "output": "docs/reviews/R156_WORKER_RETURN_2026-11-06.md",
    "budgetMinutes": 45,
    "stop": "Stop at45 prospective active minutes,12 source files or2candidateinvocations; preserve R155spent3 andUNKNOWN costs; no further automatic successor",
    "next": "Root independent review then parent intake; no installed adoption"
  },
  "rows": [
    {
      "id": "RC-01",
      "state": "BLOCKED",
      "gap": "R155 initial scaffold lacks joins and conflates historical/current registry; original installed failure remains",
      "owner": "Root ORCHESTRATOR; parent CVF maintainer owns acceptance/delivery",
      "destination": "R156 executable candidate and parent intake; reviewed parent delivery/adoption before original range recheck",
      "evidence": [
        {
          "path": "docs/reviews/R154_POSTMERGE_PROVENANCE_AND_GATE_HOLD_2026-11-06.md",
          "sha256": "ed5e43bd6992412e3def99de8e8bf88a56ca4f434205a2689ba8f9632843b3f2"
        },
        {
          "path": "docs/reviews/R155_INDEPENDENT_IMPLEMENTATION_REVIEW_2026-11-06.md",
          "sha256": "88bad9653338691c43d968e7af0bc61710d4ceb442134ba73d15f91a67241c12"
        }
      ]
    }
  ],
  "supportingWork": [
    {
      "criterionIds": [
        "RC-01"
      ],
      "reason": "Current delivered parent module lacks temporal admission selection; bounded candidate makes parent repair reviewable without altering installed controls",
      "budgetMinutes": 45,
      "stop": "Only additional R156allowance; R1553used andUNKNOWNactive remain, no reset",
      "returnToProduct": "Return to RC-01 original range and then M2/M3 roadmap only after parent acceptance, delivery and authorized adoption"
    }
  ],
  "conclusions": {
    "tranche": "PENDING",
    "trancheSummary": "Incremental uninstalled join implementation awaiting reviewed dispatch",
    "trancheEvidence": [],
    "product": "PENDING",
    "productSummary": "RC-01 original required gate and M2/M3 product holds remain; no runtime or hosted CI acceptance",
    "acceptance": [],
    "owner": "Root technical proposal disposition; parent owns source acceptance; Product Owner retains product acceptance"
  }
}
```

## Public Export Disposition

DEFERRED_PRIVATE_ONLY

## Final bounded disposition

Material build705ef67c772020959d04066f14ae0b7dd77c81a8 retains final code and independent12/12 exit0 evidence. Root independent implementation review accepted STATIC_CANDIDATE_ONLY; installed1.3 scoped R156 range0ea490f0..705ef67c and portable7/7 passed. APPL and ROUTE explicitly not applicable at terminal checks; dispatch APPL was checked earlier. This block preserves original DISPATCH coverage snapshot and RC-01 BLOCKED. R156 closes only this tested uninstalled proposal. Parent acceptance/version/delivery, deliberate project adoption and ORIGINAL R154 required range still outstanding. R155 partial, all old failures/spend/UNKNOWN and product/account holds retained. No further worker/reviewer test allowance.
