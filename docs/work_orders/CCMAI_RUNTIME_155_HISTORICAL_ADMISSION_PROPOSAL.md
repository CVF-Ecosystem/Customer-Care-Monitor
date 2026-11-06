# R155 — historical admission fix proposal for parent review

Status: BLOCKED_WITH_REASON
workOrderStatusGrammar: cvf.workOrderStatusGrammar@1.0.0
Risk: R2
Authority: `CVF_SESSION/authority/CCMAI-RUNTIME-155.json` at `839e1fecc4f4fa23cb3896f1b5f4818458f01def`
Worker: Terra 6.1 High IMPLEMENTATION_WORKER, proposal only
Reviewer: Root independent R2 REVIEWER of Terra implementation
Design reviewer/advisor: Sol 6.1 Medium independent of Root seed/design authorship
Commit steward: Root

## INTAKE / DESIGN / SPEC and bounded outcome

Owner on 2026-11-06 directs handling the current blocker, permits Terra6.1 High and Sol6.1 Medium for hard work, and assigns Root orchestrator/reviewer. Actual main merged PR21 at e38d679548bdc80ba9f4bc10a2e0d09056a2189f, while former continuity still said unmerged. Root corrected that observed drift before this successor. Existing R154 immutable seed forbids gate/profile changes; R155 does not widen it. Public Core c4c8767 stays read-only and installed profile1.3 stays pinned.

Consumer: parent CVF maintainer and the next CCMA developer. Deliver one reviewable, executable but UNINSTALLED candidate and regression tests, with the original failure retained and exact parent change/adoption instructions. Candidate files live only under `docs/reviews/probes/r155_historical_binding_20261106/`; they are not loaded by installed gates or workflows. Parent delivery ee68e90d3 verifies56 hashed files plus one null-hash continuity directory; its repair module is unchanged normalized7c58133b, so installing profile1.4 would not fix the defect. No parent adoption has occurred.

Use the existing RC-01 criterion and registry pointer. This is bounded supporting work toward that same outcome, not a new maintenance criterion or product milestone. The complete coverage table below keeps RC-01 BLOCKED. Root reconciles canonical continuity, catalog and learning; worker cannot edit them. R139/M3, R151, R150 INCONCLUSIVE/A2 UNRUN/cleanup UNKNOWN, R152 deferral, R153 consumed cap and account parking persist. This work is independent because it neither executes nor changes any product/provider/browser/cleanup path or their acceptance.

## Locked source and design requirements

Anchor proposal to project e38d679548bdc80ba9f4bc10a2e0d09056a2189f, installed lock profile1.3 and source module hashes. Correct R150 admission anchor is c61b01fa89c40135093103a0852de70f82bd39ec (registry pins the A1/A2 audit). ec6eed18a5f1951d9dc54a5b49d14f804574ab67 still pins earlier R144 audit and MUST fail. Historical context cannot be inferred solely from a FROZEN label, unchanged file, or arbitrary old commit.

1. Propose strict separate historical-admission binding schema, exact keys and unique packet/tranche bindings. Bind milestone, tranche, exact packet path+LF-normalized digest, admission commit, historical registry/audit, seed, build and terminal commitments/evidence. Reject duplicate JSON keys, nonfinite values, unsafe/escaping paths, symlinks, oversize inputs, unknown or nonancestor commits and missing/ambiguous evidence. Pins are declared review evidence, not authenticated operator consent.
2. Historical action matching is available ONLY for retrospective range validation of independently frozen tranches. Keep it unavailable for bootstrap/pre-dispatch/worker-return, active work orders and explicit execution packets. The exact checked head must equal checkout HEAD; all anchors must be ancestors of that head, admission ancestor build ancestor terminal. Packet content at admission, terminal and checkout must match normalized digest; committed seed and terminal record must match bound bytes. Do not offer automatic permissive fallback on malformed/missing bindings.
3. Always perform CURRENT registry, current audit/evidence, repair-transition/history/cost and unresolved-product guards, including for retrospective candidates. Resolve historical registry at the admission commit and require it pins the selected historical audit. Validate exact historical action selected/kind/owner/output/packet/question/budget with existing semantics; old history must prefix current history and old affected criteria must be covered by current affected criteria. Substitute only the historical action join, not all present-day controls.
4. Preserve strict current/default behavior and current R151 action binding. Existing schema cannot select historical audit; show the minimal parent-owned schema/runner/coverage changes needed, not an invented recognized field added to the live registry. Retain existing inherited modules verbatim if copied for fixtures; candidate changes only proposal copies/diff. Do not hand-edit installed copies/lock/workflow.
5. Bound first implementation to retained historical dependency bytes that still match pins. If a historical dependency requires full virtual Git-tree support, fail explicit UNSUPPORTED with owned follow-up instead of broadening source resolver or silently using wrong bytes. Parent decides generalization and adoption. If full intended positive path cannot be implemented within finite scope, return honest partial proposal with exact unproved boundary; never claim RC-01 fixed.

## Worker artifact / verification contract

Write only `docs/reviews/R155_WORKER_RETURN_2026-11-06.md` and the proposal directory. Provide `README.md`, candidate source/diff, deterministic stdlib unittest fixture(s), and secret-free JSON/text run evidence. Record exact source hashes, commands, exits, test counts and actual time or UNKNOWN. Explicitly label all fixtures STATIC_CANDIDATE_ONLY and original installed failure separately. No provider-backed claim is being made.

Maximum75 active worker minutes for cumulative R155 implementation; earlier advisory effort and cost remain UNKNOWN and are not reset. Two INITIAL test invocations only: (A) one exact installed1.3 original R150 range reproduction using pinned f7405c470207dc0d3be64e6eca79b7050d3f79d8 to current checked HEAD with all lock pins; retain expected failure, do not repair installed code; (B) one consolidated candidate static fixture suite. One justified repair round permits at most ONE targeted rerun of affected candidate tests; preserve failing result and reason. Pure compile checks count as invocations; do not add them separately. Use Python -B; disposable fixtures must be inside proposal directory or its ignored runtime child, never third-party existing resources. Git/Python subprocesses for isolated static fixtures and read-only source/provenance are permitted. No app, browser, daemon, DB, network, credentials or provider runtime.

Required fixture comparisons: exact historical positive with current registry still R151; default/active R151 requires current audit; wrong admission c61/ec6 distinction; forged freeze/unknown tranche; unknown/nonancestor/checkout-head mismatch; packet/audit/evidence/seed/terminal mutations; criteria/action/question/budget mismatch; duplicate/unsafe/nonfinite/oversize bindings; erased/reordered history and UNKNOWN-cost reset; unresolved product COMPLETE; historical binding rejected for execution admission; both-parent merge/collision scenario without losing current authority. Group subcases in a single suite. Report UNSUPPORTED explicitly if a requirement cannot be proved; no synthetic green total when expected behavior is not implemented.

Stop at budget, exhausted invocations, unexplained source/authority/ancestry mismatch or need for installed/source-pack mutation. Return bounded partial/BLOCKED with concrete unresolved dependency; do not create automatic successor, reset counters or weaken tests. Source reading should be targeted, at most20 additional implementation/reference files after startup; count and stop. Root retains review, authority, continuity, staging/commit and parent handoff. Worker never stages or commits the project repository. Synthetic git init/add/commit is permitted only inside worker-owned disposable fixture repositories for ancestry tests; never modify real project history.

## Root review and parent transfer boundary

Root independently inspects candidate code, source identities, actual run evidence and negative controls, and may run one focused independent candidate suite (separate reviewer process). Sol may review a hard finding without taking over implementation. Parent acceptance, parent implementation, refreshed manifest/file hashes and deliberate project upgrade require a future authorized gate delivery; this packet does NOT install or edit parent files. Put one precise learning/intake record in the existing shared folder linking proposal and evidence. No message to others or remote publication is authorized here. R154/RC-01 and product remain blocked even if candidate fixtures pass. Root retains all known doctor/gate failures and avoids any live AI enforcement claim.

## Gate-To-Role Closeability Contract

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
returnTimeRecheck: REQUIRED_BEFORE_REPAIR
implementationTopologyPolicy: BOUNDED_PATH_FAMILIES
foreseeableFileSplitDisposition: COVERED_BY_BOUNDED_PATH_FAMILY

| gateId | mustPassBy | repairOwner | repairPhase | mutationSurface | topology | commitOwner | commitPhase | dependsOn |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| authorization_review | PRE_DISPATCH | orchestrator | PRE_DISPATCH | committed R155 seed and RC-01 scope | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | NONE |
| pre_dispatch_gate | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R155 packet and RC coverage | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | authorization_review |
| dispatch_continuity | IMPLEMENTATION | session-sync-steward | IMPLEMENTATION | CVF_SESSION/handoffs/AGENT_HANDOFF_R155_HISTORICAL_ADMISSION_2026-11-06.md material-SHA marker | EXACT_PATHS | session-sync-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| focused_checker_tests | WORKER_RETURN | implementation-worker | WORKER_RETURN | baseline failure and uninstalled candidate tests | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| adif_integrity | WORKER_RETURN | implementation-worker | WORKER_RETURN | retained failures and unsupported boundaries | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests |
| pre_implementation_autorun | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | fresh R155 worker rehydration/declaration | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| worker_return_fast | WORKER_RETURN | implementation-worker | WORKER_RETURN | proposal-only return and costs | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests,adif_integrity,pre_implementation_autorun |
| reviewer_fast | REVIEW | reviewer | REVIEW | Root independent candidate review | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | worker_return_fast |
| pre_commit | PRE_MATERIAL_COMMIT | reviewer | PRE_MATERIAL_COMMIT | exact scope and continuity | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | reviewer_fast |
| terminal_completion_review | POST_MATERIAL | reviewer | POST_MATERIAL | proposal only, RC/product blocked | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | pre_commit |
| continuity | CONTINUITY_COMMIT | session-sync-steward | CONTINUITY_COMMIT | state, handoff, memory, status and catalog | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | terminal_completion_review |
| committed_range_closure | POST_MATERIAL_CLOSURE | reviewer | POST_MATERIAL_CLOSURE | R155 committed range only; original failure retained | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | continuity |

## Complete RC milestone coverage

The existing authoritative RC table remains unchanged. All its criteria (RC-01 only) are accounted for below; proposal support does not satisfy the required original merged-range gate. M1–M5 and their owners/destinations remain in the existing master roadmap. Do not add a maintenance milestone or switch current M3 audit.

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
      "trigger": "R155 candidate return or bounded budget exhaustion",
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
    "owner": "Terra 6.1 High IMPLEMENTATION_WORKER, proposal only",
    "output": "docs/reviews/R155_WORKER_RETURN_2026-11-06.md",
    "budgetMinutes": 75,
    "stop": "Stop at75 active worker minutes,20 source files,2 initial invocations plus1 justified targeted rerun, source/authority uncertainty or need for installed/parent mutation",
    "next": "Root independently reviews candidate and prepares parent intake without installed activation"
  },
  "rows": [
    {
      "id": "RC-01",
      "state": "BLOCKED",
      "gap": "Original merged-range gate compares frozen R150 with current R151 audit; delivered parent pack has no fix",
      "owner": "Root ORCHESTRATOR; parent CVF maintainer owns acceptance/delivery",
      "destination": "R155 uninstalled candidate and parent intake; then independently reviewed delivery and deliberate adoption before original R154 gate recheck",
      "evidence": [
        {
          "path": "docs/reviews/R154_POSTMERGE_PROVENANCE_AND_GATE_HOLD_2026-11-06.md",
          "sha256": "ed5e43bd6992412e3def99de8e8bf88a56ca4f434205a2689ba8f9632843b3f2"
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
      "budgetMinutes": 75,
      "stop": "Same cumulative R155 cap; no renewal or automatic successor",
      "returnToProduct": "Return to RC-01 original range and then M2/M3 roadmap only after parent acceptance, delivery and authorized adoption"
    }
  ],
  "conclusions": {
    "tranche": "PENDING",
    "trancheSummary": "Uninstalled proposal implementation awaiting independently reviewed dispatch",
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

## Actual initial implementation disposition

Terra returned PARTIAL/UNSUPPORTED; Root independent review requires changes. Provenance-only preflight cannot admit and incorrectly equates current/historical registry bytes. Six subset tests are not outcome acceptance. Both initial invocations and one cleanup repair rerun are spent; no R155 second pass. Retain all original artifacts. See R155_INDEPENDENT_IMPLEMENTATION_REVIEW_2026-11-06.md for the exact incremental Root decision and parent boundary. RC-01 and product remain BLOCKED.
