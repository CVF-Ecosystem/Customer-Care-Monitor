# R152 workspace activation maintenance

Status: DRAFT
workOrderStatusGrammar: cvf.workOrderStatusGrammar@1.0.0
Tranche: CCMAI-RUNTIME-152
Authority: `CVF_SESSION/authority/CCMAI-RUNTIME-152.json`
Risk: R2
Worker: Terra 6.1 IMPLEMENTATION_WORKER
Reviewer: Astra Medium independent R2 REVIEWER
Commit steward: Root

## Outcome and source

Adopt the applicable shared CVF workspace update into this project in one tooling maintenance packet. Use `docs/reviews/R152_PARENT_UPDATE_APPLICABILITY_2026-11-06.md` and the clean `f8175616c` delivered pack; keep public Core pinned separately at `c4c8767ac8861854f40d466ba338ffd11eb9558e`. Preserve all R151/R139/M3 holds, costs and project-specific decisions. This packet does not authorize product source, provider, Windows process or network activity.

## Worker scope and sequence

1. Rehydrate the current state/handoff, exact seed, applicability review, pack manifest, profile migration guide and project-development standard. Stop on disagreement or source hash mismatch. Record pre-change identity and the old 1.3/public-doctor version-mismatch result.
2. Prepare `docs/reviews/probes/r152_workspace_activation_20261106/upgrade_authority.json` with `cvf.workspaceActivationUpgrade@1.0.0`, exact Core/delivery/profile/runner/bundle pins from verified delivered source, and hashed `docs/reviews/R152_UPGRADE_INDEPENDENT_REVIEW_2026-11-06.md`. Root/Astra review the exact record and pins before policy references it or pack code executes. The worker must not invent a full source commit from the abbreviated pack anchor.
3. After Root releases the reviewed upgrade record, set explicit `CORE_AND_SUPPLEMENTAL` and `OPERATOR_LOCAL_PACK` policy, pin its raw record hash, then run the verified delivered installer once with `--upgrade` and `--core-commit UNKNOWN`. Keep any partial failure and reconcile preserved project-owned CI pins deliberately.
4. Generate a project-owned `.cvf/workspace-activation.json` assessing the exact existing R151 work-order action, deriving identity from actual post-upgrade comparison. Give every mandatory pre-dispatch control one action/criterion row and real hashed project evidence. A reviewed, action-specific supplemental assessment is required. APPLIED controls require actual configured invocation and reviewed invocation evidence; genuinely held R151 controls remain HELD, with `REVIEW_REQUIRED` retained rather than forced green. Do not claim a command was run or independently reviewed before it occurred. If review timing requires Root to finish the receipt after worker return, return a candidate and its open fields without fabricating acceptance. R151 remains held; binding its genuine M3 action does not dispatch it.
5. Update only the authorized inherited instruction section in `AGENTS.md` to describe profile 1.4 and current doctor split. Preserve all project-specific R151/R139 holds and historical profile 1.3 records. Return exact changed files, hashes, commands/results, failures and unknowns.

Worker may edit only `workerAllowedPaths` from the immutable seed. Root owns authority, review, catalog, continuity and commits. No retry loop: one install attempt plus at most one same-scope repair of a concrete migration mismatch; at repair round three without an independent new root cause stop `REVIEW_COST_ESCALATION_REQUIRED`.

R152 is the bootstrap migration action. Its installed-profile 1.3 pre-dispatch and independent R2 packet review govern release before upgrade, at the three current lock pins. Before any future dispatch, the tranche record must map `DISPATCH_READY` exactly to this packet's status and the required contracts must pass. A genuine maintenance criterion and its project authority are not yet available. No M3 criterion is selected to stand in for tooling maintenance. After migration, use the new trusted bootstrap and pre-commit checks for this maintenance change. R152's independence from held M3-05/08 is assessed in `docs/reviews/R152_PARENT_UPDATE_APPLICABILITY_2026-11-06.md` and requires independent Astra acceptance before BUILD.

## Review and acceptance

- All delivered executable source hashes and installed 17-file bundle match the manifest and profile 1.4 lock; public Core remains clean and pinned.
- Reviewed upgrade record predates pack execution route; policy and receipt bind exact raw bytes and R151 action/criteria. CI carries actual three new pins. The activation launcher and trusted bootstrap give their bounded, explicit results, including an honest `REVIEW_REQUIRED`/HELD assessment if R151 controls cannot be admitted.
- Installed profile pre-commit and portable preflight pass for the changed set; required downstream unittest suite passes if gate/continuity/workflow files changed. The new trusted bootstrap may remain non-green solely for explicitly held R151 action controls, which Astra must review without interpreting as a maintenance failure. R2 source and receipt semantics get independent Astra review before Root closes.
- Maintenance conclusion is separate from product conclusion: R151 remains held, R139/M3 blocked, Facebook/Zalo OA parked; no live governance, hosted CI, deployment or product readiness claim.

Stop and return `BLOCKED_WITH_REASON` if any exact source, policy, authority, receipt, CI or gate invariant fails without a bounded same-cause correction. Preserve failing output and partial writes; do not widen scope or reset costs.

## Parked disposition — 2026-11-06

The initial installed1.3 pre-dispatch on a prospective DISPATCH_READY packet failed `contract_missing` and `COV_BLOCK_REQUIRED`. A closeability graph can be supplied, but the current milestone registry has no genuine maintenance criterion for installation. The proposed M3-06 shortcut was rejected during independent review as unrelated to the tooling change. No worker BUILD, installer, policy route, lock, CI or activation receipt change is released. Keep this packet DRAFT and held before dispatch until a project-authorized maintenance coverage route is designed and independently reviewed; do not use null-authority VERIFY to disguise installation.
