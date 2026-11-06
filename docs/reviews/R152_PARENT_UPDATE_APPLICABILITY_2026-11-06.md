# R152 parent workspace update applicability

Status: APPLICABILITY_DECISION_PENDING_INDEPENDENT_REVIEW
Date: 2026-11-06
Risk: R2
Effect boundary: no product/runtime effect
Author: Root ORCHESTRATOR / SPEC_AUTHOR / WORK_ORDER_AUTHOR
Worker: Root documentary assessment; Terra not released
Reviewer: Astra Medium independent R2 REVIEWER

## Source and decision

The public project Core remains `c4c8767ac8861854f40d466ba338ffd11eb9558e`; its clean HEAD and `origin/main` match the manifest. The active operator-local workspace pack is a separate delivery with clean abbreviated source anchor `f8175616c`. Its raw manifest SHA-256 is `2544c83270e18b1f61e48d9631d9df3c28e0159a14807835c4e4e0000f416ab5`; all 53 hashed delivered files matched. The full source commit is UNKNOWN from this delivery. The installed project profile is 1.3; the delivered profile is 1.4. Public doctor retains its historical 1.2/installed-profile version mismatch; it is not a full PASS.

Verified prospective source pins, using LF normalization for the inherited lock: profile `cef123ab02f3c4610e14c8b610ef9f2c00d8312d28ff8b72bd3968cb70d09998`, runner `6b608a8a138f54b5dfc0462f5845fcf6a5bc916270ada5b0bf6d6909a153b3bd`, 17-file bundle `234e1cccd3a5f69ce2809a515c802c67825cacefe92435290068b343f14901c3`. Review these pins before policy selects the pack; verify the actual installer output again after installation. Upgrade record: `docs/reviews/probes/r152_workspace_activation_20261106/upgrade_authority.json`. Independent pin decision: `docs/reviews/R152_UPGRADE_INDEPENDENT_REVIEW_2026-11-06.md`.

Apply the project-development application standard now through this existing review and the current project handoff. It connects common baseline, project controls and task evidence without changing product criteria. Profile 1.4 adds source-route, action/criterion and per-control evidence bindings, but its installation is parked pending a genuine maintenance coverage admission. Delivery is verified; installation and activation are not claimed. Neither the standard nor the uninstalled checker proves runtime AI governance.

| Layer | Applied project decision | Evidence / owner |
| --- | --- | --- |
| Common baseline | Preserve public Core pin; separately identify the operator-local profile 1.4 delivery and apply its development guidance now. Profile installation remains parked. | Manifest and file hashes above; Root owns mapping, Astra reviews. |
| Project-specific controls | Keep R151 `BUILD_BLOCKED`, R150 A1-b INCONCLUSIVE/A2 UNRUN/cleanup UNKNOWN, R139/M3 blocked, original spent budgets and Facebook/Zalo OA parking. R151 seed prohibits gate/profile/workflow edits. | R151 handoff, seed and milestone registry; Root owns continuation. |
| Task evidence | R152 records exact source comparison and the failed pre-dispatch route. A future reviewed maintenance authority must precede policy/lock/CI/receipt changes. | R152 work order and failed gate evidence; Astra judges, Root synchronizes. |

Native/shared use when admitted: invoke the delivered installer and portable activation helper; do not build a project-specific replacement. The current R151 action packet can be assessed later but its source/effect work stays held. R152 maintenance is independent of M3 repair implementation; no product use case or broad test matrix is admitted.

## Causal and boundary assessment

`python ../Check-CVF-Project-Activation.py --project .` currently returns `BLOCKED_SOURCE_POLICY` because project policy has no explicit activation source policy. The new pack's trusted bootstrap reports installed-identity mismatch and missing activation receipt against installed 1.3. This is expected version drift, not proof that R151 repair failed. The public Core lacks the new helper. An explicit `CORE_AND_SUPPLEMENTAL` plus `OPERATOR_LOCAL_PACK` route may be appropriate after a separately reviewed exact upgrade record and maintenance admission; it has not been selected in policy.

R151's immutable dispatcher seed cannot be widened to cover `.cvf/policy.json`, inherited gate files or CI. R152's separate maintenance seed is necessary but not sufficient: installed1.3 pre-dispatch on the proposed `DISPATCH_READY` packet failed `contract_missing` and `COV_BLOCK_REQUIRED`. A closeability table is routine; the substantive issue is that the existing milestone registry has no genuine maintenance criterion for installing shared tooling. Selecting M3-06 just to pass the gate would misstate a product contribution, and null-authority finite VERIFY cannot authorize implementation. The failed output is retained under `docs/reviews/probes/r152_workspace_activation_20261106/pre_dispatch_failed_20261106.txt`. R152 remains DRAFT/WORK_ORDER before worker BUILD, and the profile remains 1.3.

R152 independence assessment: The proposed change would touch policy, installed gate modules, CI, project instruction and continuity metadata only. It would not change the R151 diagnostic helper, its V04 orchestration, R139 integrated execution or any M3 product criterion. R151's exhausted source budget and R139 failed-repair audit remain held in their existing records. This explains why a future maintenance action may proceed independently after a valid admission; it does not supply the missing machine-contract route. Astra independently reviews this separation. Unknown future applicability is not claimed as solved.

## Finite exit

Current exit: the shared development application mapping is recorded and independently reviewed; R152 installation remains unadmitted. One future maintenance route must either admit a genuine maintenance criterion within the project authority or provide a reviewed profile contract that explicitly supports tooling migration. Then revise this same R152 packet and review before dispatch, preserving the failed attempt and source pins. Do not expand to product investigation or create per-control tranches.
