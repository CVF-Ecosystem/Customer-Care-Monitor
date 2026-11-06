# R154 postmerge provenance and range-gate hold

Status: MERGE_PROVENANCE_ACCEPTED_REQUIRED_RANGE_GATE_HOLD
Date: 2026-11-06
Risk: R2
Worker: Terra 6.1 candidate implementation; Root COMMIT_STEWARD merge commit
Reviewer: /root/r152_roadmap_form_review, independent R2 REVIEWER
Merge: 229d98b3b33a17e4d23b3ded34cdd4455c5ca5d3

## Verdict

Accept bounded merge/provenance preservation. Required installed-profile remote-base range gate remains FAIL; R154/RC-01 must not FREEZE or claim completed clean reconciliation. Portable checks do not supersede that failure. No green hosted-CI, runtime governance, product acceptance or publication-completion claim is established.

Re-read project AGENTS and current R154 state, candidate review and worker return; existing canonical continuity still describes BUILD and must be synchronized by Root to the observed gate hold. Core c4c8767ac8861854f40d466ba338ffd11eb9558e and installed1.3 remain unchanged, live governance evidence required YES, R2 independent reviewer. R139/M3, R151, R150 limitations, R153 consumed pass, R152 deferral and account parking remain.

## Provenance checked

Merge parents are exactly 70f3e3256e9e512501b4dd3433216bc53770e191 and f7405c470207dc0d3be64e6eca79b7050d3f79d8, establishing both parent histories as ancestors. Actual merged-HEAD first-add query for canonical R150 returns ec6eed18a5f1951d9dc54a5b49d14f804574ab67. Canonical seed blob b5fa38150e4970aa82a04c219301dc245fc8041a matches that first commit and R150 base c61b01fa89c40135093103a0852de70f82bd39ec. All three archived cloud originals at merge HEAD again match their pinned remote blobs. Candidate review's exact six errata/probe preservation and F1-F6 qualifications remain the bounded acceptance basis; no cloud erratum is thereby retroactively applied to Windows work.

## Exact range result and cause

Reviewer independently invoked installed1.3 pr-ci with base f7405c470207dc0d3be64e6eca79b7050d3f79d8, head 229d98b3b33a17e4d23b3ded34cdd4455c5ca5d3, and all three expected pins from the installed lock. The command resolved 106 candidates and reported RESULT FAIL, mandatoryControls=8, failed=1. INST/CONT/RANGE/APPL/ROLE/CLAIM passed; ROUTE explicitly NOT_APPLICABLE_WITH_REASON (no dispatchKind, not checked). COV alone failed:

COV_REPAIR_ACTION_MISMATCH @ docs/work_orders/CCMAI_RUNTIME_150_A1_A2_WINDOWS_VERIFY.md

The historical Windows work order is present in the remote-base range but unchanged against the local merge parent. Its embedded M3 action is VERIFY with the Root Windows owner and R150 output. Current docs/milestone-coverage-authority.json points repairControls.M3.audit to repair_chain_assessment_r151_evidence_capture.json, whose admittedAction is IMPLEMENT with Terra and the R151 source return. scripts/cvf_gates/cvf_repair_convergence.py:163-166 compares selected criteria and kind/owner/output against that current admitted action. The mismatch follows from historical dispatch evidence evaluated against successor authority. Frozen R150 tranche status does not exempt that embedded block from this range validation.

This explains the observed failure; it does not itself prove a sanctioned gate exception or certify a complete fix. It is not evidence of new runtime regression or seed corruption. No actual hosted run was inspected: this is a locally reproduced PR-range failure. Root reported portable range 7/7 PASS; that separate result cannot cancel the required installed-profile failure.

## Safe continuation boundary

Retain merge/provenance acceptance and explicit required-range HOLD. Do not switch the repair registry back, rewrite frozen R150 evidence, disable/skip coverage or upgrade the profile solely to make this result green. A correction distinguishing historical admission from current dispatch needs separately bounded authority and independent review; R154 seed prohibits gate/profile edits. Preserve the failure and cumulative costs. Current RC-01 acceptance includes scoped gates, so no R154 FREEZE or COMPLETE under this review.

A review branch/draft PR may disclose the reconciled result and failing local range check within the operator's repository-sync authority, without claiming hosted success. No main fast-forward/merge, release or deployment is approved by this review. Root owns the next explicit disposition; failure does not grant history rewrite or new product effects.

## Reviewed input hashes

Raw SHA-256:

| Input | SHA-256 |
| --- | --- |
| docs/reviews/R154_WORKER_RETURN_2026-11-06.md | 833bcb53a45cd748010dfd51dd256873972431cc4f5e2c1c8949b1b73063402d |
| docs/reviews/R154_REPOSITORY_RECONCILIATION_INDEPENDENT_REVIEW_2026-11-06.md | 34375b77fece12d257d87a55d5cbca23a72a2f66b9ea782543f6e2aa6ec905ab |
| docs/work_orders/CCMAI_RUNTIME_154_REPOSITORY_RECONCILIATION.md | f0225f2aa3f95cc525e463ccebcb9d0296985aa3c47867d323e61c02786ea47f |
| .cvf/gate-profile.lock.json | 2e536dcf2cee7d44f6715a5875431490e42acbc5ec8b7e5e05280478771439bb |
| docs/milestone-coverage-authority.json | 92e4d219c36c32842223b07d92345d3b4bea85a4dba2273de4d5a1b8df647ca1 |
| scripts/cvf_gates/cvf_repair_convergence.py | 227c56680648623eed294b49795e113ac520658276c8503eb6268b925a36fd3c |

Reviewer wrote only this artifact and ran one exact static pr-ci check plus bounded source/provenance reads. No staging, commit, continuity/packet edits, broad tests, provider/network/runtime effects were performed.
