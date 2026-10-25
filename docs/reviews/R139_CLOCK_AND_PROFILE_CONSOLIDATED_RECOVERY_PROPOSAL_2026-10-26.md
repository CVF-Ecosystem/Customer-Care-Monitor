# R139 consolidated clock and installed-profile recovery decision

Status: PROPOSED_NOT_ADMITTED
Risk: R2
Worker: Root ORCHESTRATOR / SPEC_AUTHOR, with read-only Sol Medium clock design and Luna Max compatibility diagnosis
Reviewer: Root proposal review; implementation requires independent R2 review
Date: 2026-10-26

## Outcome and existing authority

One existing CCMAI-RUNTIME-139 order serves roadmap M3-05/M3-08 and the integrated G0–G3 proof. This proposal creates no tranche, campaign, provider slot or product acceptance. Owner's latest instruction is `xử lý đi`: Root proceeds with prerequisite reconciliation and prepares this concrete decision. It is not recorded as approval of an unstated new acceptance threshold, source-check cap or gate migration.

The original source commit `72371595cf1e650a5df074fb5b84cf5472576cec` and accepted-build pin `f818a976fc347340d659c13be27ff5dcd66ebe44e6a694224b0f26cba72fc5bb` retain their bounded offline acceptance. Worker and Root each passed 45 focused tests. Historical eight plus recovery two source-check invocations are spent: 10/10. The original controller PID24484 exit1 remains historical; the additional controller invocation is unused 0/1, as are the conditional campaign 0/1, services 0/5, CUA 0/64 and provider 0/1. Product B remains BLOCKED_PENDING_OPERATIONAL_ADMISSION; Facebook/Zalo OA stay parked.

## Prerequisite reconciliation completed

The official `update_cvf_workspace_public_core.ps1` ran successfully from the project working directory with the resolved workspace root and no manifest-update, overlay or pending-change override. It preserved the clean former Core at `../_cvf-core-backups/.Controlled-Vibe-Framework-CVF-20261026-004532`, cloned the public remote and refreshed workspace wrappers. Core HEAD is now `b8ead4f0d1bf9224338a206b2abcd893ec1801b1`, matching origin/main with a clean worktree. No downstream source was executed inside Core and no Core source was edited by an agent.

The next doctor reports 27/28: freshness now PASS; CVF-DG-INST-01 fails LOCK_INVALID. The current Core verifier requires `cvf.downstreamGateProfile@1.2.0`; the project intentionally still carries its previously installed and pinned 1.1.0 profile. This is a version mismatch, not evidence that someone altered the installed files. Manifest e9ddcb versus latest Core remains a disclosed warning. Do not suppress the doctor, edit its expected ID, hand-edit the lock or treat the old portable preflight PASS as doctor PASS.

Proposed migration, conditional on explicit admission, uses only the public Core installer:

```powershell
python -B ../.Controlled-Vibe-Framework-CVF/scripts/lib/downstream_governance/cvf_downstream_gate_runner.py install --project-root . --core-root ../.Controlled-Vibe-Framework-CVF --core-commit b8ead4f0d1bf9224338a206b2abcd893ec1801b1 --upgrade
```

Allow one install and one consolidated reconciliation of generated pins/workflow and the existing milestone coverage carriers. Preserve any project-owned workflow; reconcile its three pins explicitly rather than replacing custom content. Installer/doctor/profile receipts are repository prerequisite checks, never additional R139 Python source-check slots. A failed or newly incompatible control remains HOLD, with no profile downgrade, override, gate patch, silent bypass or open-ended migration. Do not rewrite an immutable seed to widen paths; the new admission must explicitly cover migration paths and the existing gate's authority mechanism must accept that scope before a commit can proceed. If that mechanism cannot represent the admission without modifying a gate, return the precise scope failure instead of making the gate pass.

Root's targeted source review also found a concrete admission-representation limit: current `scripts/cvf_downstream_gate.py:604-617,645-688` judges scope from the original immutable seed and does not load supplemental admissions to extend its allowedPaths. Merely approving or writing a supplementary admission cannot make installer-managed gate/lock/workflow paths fit that seed. Therefore the migration portion is **not dispatch-ready** under the current adapter, even after a general approval. This decision must settle a supported authority representation with the governing CVF owner before installation; no second seed, helper tranche, seed rewrite, `--files` scope omission or local gate patch is inferred. This is a known dependency, not an unknown to discover by spending another install attempt. Clock source work also stays dependent on that resolution. The narrow fallback is to retain all current HOLD evidence and the still-valid historical 1.1 bundle; it is not a claim that the latest doctor passed.

## Clock defect and successor contract

Independent HOLD review `R139_BOOT_IDENTITY_PRECLOCK_CLOCK_HOLD_REVIEW_2026-10-25.md` establishes -3.8507008s and -3.9758600s differences between UTC elapsed and monotonic elapsed since the frozen 08:40 baseline. The current contract requires absolute difference <=2s, so launching it would predictably consume the single invocation before reservation. Physical cause is unknown.

Propose a separate v3 contract/controller; preserve the v2 contract, proof, controller and receipts byte-for-byte as predecessors. Retain exact host digest, full boot-event digest, derived host/boot ID, strict timestamp grammar, CIM/event bounds and exact identity equality. Replace only the multi-hour UTC/monotonic baseline acceptance comparison with a current same-process bracket around the original single OS child query:

- Capture parent UTC and monotonic immediately before its original Popen and immediately after the original wait returns with exit0. Require nonnegative UTC and monotonic intervals, monotonic duration <=5s, and exact arithmetic `abs(UTC interval - monotonic interval) <=2s`.
- Keep child observedUTC within the parent interval with the existing +/-2s slack, event timestamps no later than observedUTC, and current parent start/child observation no earlier than the frozen proof observation.
- Preserve all original command/executable pins, original handle/wait/exit custody, timeout/nonzero/unknown HOLD, no retry/query fallback, and the separate campaign +1020/+1200 deadlines.

This changes the acceptance claim from continuous agreement since the old observation to agreement during the current query. It is not a wider numeric tolerance and is not an inference that the old failed evidence passes. Timestamps alone cannot establish fresh child output; original live process custody is mandatory. A jump and reversal wholly between captures, copied host identifiers, tampered event logs and a compromised OS remain outside the evidence.

## Consolidated proposed execution bounds

After one Owner decision, perform one authoring pass and at most one consolidated repair pass, then independent review before spending checks. Proposed source paths are a new `reentry_activation_boot_identity_v3_design.py.txt` under the existing R139 probe directory and the existing `scripts/tests/test_r139_execution_supervisor.py`. Root owns new v3 contract, predecessor binding, accepted-build, technical recipe and decision receipts. V2 source/proof/contract/reviews and old grants remain immutable. No Go/product/supervisor/Vite/dependency edit or Go build/test is admitted.

Propose exactly two additional focused Python invocations: Worker1 and Root1, increasing total lineage cap from10 to12 with the original10 still spent. Reserve each before execution and retain original raw stdout/stderr, Popen/wait/exit and unchanged source hashes. No incidental parser/import/checker/prototype or extra source test outside these two slots.

The consolidated focused cases cover a valid query hours later despite the historical offset; exact +/-2s local drift and both signs just beyond2; exact5s and >5s; UTC/monotonic reversal; observedUTC at and outside each slack boundary; rollback before the original proof observation; NaN/Inf/bool/string/missing/extra fields; duplicate JSON keys and malformed dates; host/UUID/full event/boot digest mismatch. Existing supervisor, authority, cap and terminal guards remain in the same focused suite. Synthetic cases assert component logic only, not live governance behavior.

After accepted source and packet review, exact clean commit and current prerequisite PASS, preserve the existing one unspent controller invocation and campaign rather than adding another. A successor admission must explicitly supersede and disable v2 release: v2 plus v3 together may consume at most ONE future invocation, using one shared remaining counter and one shared campaign cap. V3 references the immutable v2 proof and original contract/proof hashes; do not alter the old proof's contractRef or claim a new OS baseline. Refresh actor/recipe bindings; Root may release the original v3 preclock operation once. Actual G0 independent review and a separate Root Runtime Effect Admission still precede fixture/key/provider. Keep campaign20min, services5, CUA64 in four stages, inventories3, owned tab1, G2 status16, provider1 and <=USD1, with no automatic retry or stacking. One decision covers the bounded repair through conditional release; no per-step Owner relay.

## Paths and decision boundary

The proposed admission must add only the installer-managed `scripts/cvf_gates/**`, `.cvf/gate-profile.lock.json`, generated/reconciled `.github/workflows/cvf-downstream-gates.yml`, `docs/milestone-coverage-authority.json`, and directly necessary existing coverage carrier sections in the R139 work order and its referenced roadmap, plus the two source paths and existing R139 metadata/continuity/catalog classes. Profile1.2 adds mandatory CVF-DG-COV-01; explicitly admit its coverage authority/carriers, not only new pin bytes. The installer pins Core b8ead4f; the manifest's old public pin need not be silently rewritten. Historical seed, receipt and accepted-build records remain unchanged. The independent reviewer must assess authority representation, exact changed set and coverage before commit; a scope mismatch is HOLD, never permission to hand-edit gate code.

Approval requested is this entire profile migration + successor temporal contract + two new source checks, with existing runtime caps unchanged. The exhausted check budget, changed temporal acceptance and additional gate/workflow artifact class are real boundaries under AGENTS.md Governance Latency and Approval Continuity. No implementation or launch follows merely from writing this proposal.

## Independent design and compatibility findings

Sol Medium `/root/r139_clock_recovery_design` reviewed the design read-only and required the explicit shared v2/v3 invocation counter and immutable predecessor binding now included above. Luna Max `/root/r139_core_profile_compatibility` confirmed the mismatch through current `cvf_dg_common.py:16`, `cvf_dg_install.py:208-210`, trusted doctor routing, and `CVF_DOWNSTREAM_GATE_PROFILE_MIGRATION.md:48-71`: reinstalling1.1 cannot satisfy the latest verifier, and upgrading is deliberately operator-initiated. The missing coverage authority and new mandatory COV-01 are included in this proposed boundary. No subagent ran tests, imported source, queried OS/browser/services, installed a profile or changed implementation. These are DESIGN findings, not source/runtime acceptance.

## Product position and next move

Selected work is BLOCKED: offline source is accepted within its old contract, while live G0–G3 and E01–E06 fulfillment are not established. Root owns the consolidated admission and independent release route in this same R139 order. Product B cannot yet demonstrate the integrated operational outcome. Continue only after the concrete consolidated decision and prerequisite compatibility are settled; otherwise preserve HOLD and all unspent effects.
