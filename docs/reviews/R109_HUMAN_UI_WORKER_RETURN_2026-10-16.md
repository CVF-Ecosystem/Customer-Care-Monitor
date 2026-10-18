# R109 human UI worker return

Status: BLOCKED_WITH_REASON
Risk: R2
Worker: /root/r108_browser_luna (Luna Max, IMPLEMENTATION_WORKER / admitted supervisor executor)
Reviewer: Codex /root
Disposition: BLOCKED_WITH_REASON

## CVF Agent Declaration and authority

Project: Customer-Care-Monitor-AI. CVF core: `../.Controlled-Vibe-Framework-CVF` at `e9ddcbcdcac2a9df520f5eb0761dbdb609b9deba`. Phase: BUILD. Risk ceiling: R2. Live evidence required: YES. Active role: Luna Max IMPLEMENTATION_WORKER and admitted supervisor executor; no browser role. Active handoff: `CVF_SESSION/handoffs/AGENT_HANDOFF_R109_HUMAN_UI_2026-10-16.md`. At campaign start, the next allowed move was the single admitted R109 campaign at HEAD `7fc2273b485b765543dd78fc60afc51ee68d0cb3`, reviewed source `5ab60830368482e072cb0150a7894235ca950c24`, followed by evidence return. Parked checkpoint: owner-deferred Facebook/Zalo OA account setup, credentials, connectivity and live tests.

Root's read-only check of the committed admission, recorded in `docs/reviews/probes/r109_root/committed_admission_check.json`, reports PASS, `campaignSlotUsed=false`, and the admitted reused binary SHA-256 `cb73ee13e0369cca75ea276cc668f30c3ced42cead5f62c56d68f3c1eaa66f93`. The check receipt itself was generated after the admitted HEAD commit. The worker acknowledged this authority in the active handoff before invoking the supervisor. Root's post-interruption read-only source check also passed for the admitted source, evidence, and binary pins; it makes no supervisor-finalization claim.

## Final disposition

BLOCKED_WITH_REASON. One admitted campaign produced useful partial current-source UI evidence, but the connected browser/execution context was interrupted and then denied by browser URL policy. B07 was not run; B08 and B09 remain not run. The original supervisor did not produce final cleanup or disposition receipts. Root independently recovered and stopped only the exact fixture; this does not replace the missing original supervisor receipt. The owner's stop/transfer condition now applies. Transfer the remaining product verification and open language repairs to a stable connected agent. No source, backend, UI, seed, work-order, or admission changes were made during the campaign. No commit, freeze, push, deployment, or full product acceptance is claimed.

## Actual campaign and resource disposition

The exact admitted command was invoked once:

```text
python -B docs/reviews/probes/r109_luna/r109_workflow_runner.py start --project-root .
```

It ran in owned exec session 6933. After the browser/executor interruption, polling that session returned `aborted`; the supervisor's final stdout/stderr, exit status, cleanup receipt, and terminal disposition are `NOT_OBSERVED`. The original campaign files remain untouched in `.cvf/runtime/r109_workflow/campaign/`, including its `stop.request` and `running.json`; no missing original receipt was synthesized or rewritten.

Actual admitted effects: one existing fixture start, one native invocation, one backend listener launch, one Vite launch, zero Go compiles, and zero retries. The native diagnostic exited 0, accepted the four-action lifecycle, and classified the historic failure as `NOT_REPRODUCED`; this does not identify the earlier R107 503 cause. There were 4 native action POSTs and 11 actual UI POSTs, including the initial automatic auth refresh, old-session logout, authentication, rejection, and owner actions. Root's reconciled evidence preserves the logout's missing pre-click reservation; no reservation was backdated. Two product tabs are confirmed; completion of an attempted third tab creation is `NOT_OBSERVED`.

Root's exact-resource recovery receipt at `docs/reviews/probes/r109_root/interrupted_cleanup/root_recovery_receipt.json` reports PASS. Root verified the exact fixture ID, image, label, network and exclusive volume, stopped that fixture once with `--timeout`, and observed final state `exited`. Protected Docker fields matched both the original campaign baseline and the state before Root cleanup. No process was killed by PID. PID 18728 was observed as `conhost`, and PID 20848 was absent. Root's separate recovery is recorded separately from the missing original supervisor cleanup. Owned browser-tab closure after the policy interruption is `NOT_OBSERVED`.

## Connected browser results

The authoritative matrix and captured limitations are in `docs/reviews/R109_ROOT_UI_OBSERVATIONS_2026-10-17.md`; exact browser policy error and interruption state are preserved in `docs/reviews/probes/r109_root/browser_interruption.json`.

| Case | Result | Observed result and limit |
| --- | --- | --- |
| B01 | PASS | Fresh owner login, queue, and pending case observed. |
| B02 | PASS | CORRECT submitted once; update 2 and the human explanation/next step persisted. |
| B03 | PASS | Stale CONFIRM reached the server as HTTP 409; the UI showed the conflict and refreshed without replaying a write. |
| B04 | PASS | ASSIGN persisted as update 3 with the selected member and deadline after reload. |
| B05 | PASS | RESOLVE persisted as update 4 and appeared in the terminal filter after reload. |
| B06 | PASS | REOPEN returned HTTP 200; update 5 is assigned with the new deadline, active outcome cleared, and the exact old outcome retained in readable history. |
| B07 | NOT_RUN | Owner logout returned HTTP 200. The interruption occurred before fresh reader login; reader write-control absence was not tested. |
| B08 | NOT_RUN | A 390x844 viewport request returned without error, but the actual DOM and screenshot remained 1280x720 after reload. Reset was invoked; no mobile acceptance is claimed. |
| B09 | NOT_RUN | No documented offline capability was exposed. No network change or interception was used. |
| B10 | PASS (functional) | WAIT_DATA assignment and escalation persisted as updates 2 and 3, and the case remained waiting/incomplete after reload. Completion controls stayed unavailable. The source-language finding below remains open. |

The initial HTTP health bridge passed before the product campaign; it is not product UI evidence. Original screenshots, DOM captures, metadata, the policy interruption record, and cleanup command outputs are exported with the runtime manifest below. The exact CUA calls/results remain in the task transcript; the saved metadata is an operator record, not a raw serialized transcript. The browser denied the attempted page acquisition with the exact URL-policy message recorded in `browser_interruption.json`. No workaround, alternate browser surface, or new runtime was attempted.

## Open product-language findings

- The WAIT_DATA alert correctly says information is incomplete, but a generic second sentence claims a local match. Replace it with a neutral limitation for every source status; do not imply a match for INCOMPLETE.
- The shared shell still shows “Cấu hình AI Provider”, “Kết nối MCP”, and service terms. The intervention-specific raw codes/JSON and machine identifiers were removed, but application-wide plain language remains open.
- The observation named `14_b08_narrow_detail_history` is desktop-size evidence and must not be used as mobile proof.

The UI source passed Root's focused 14-test run and forced typecheck before runtime admission. Root's source review accepted those changes and the supervisor. No product code was edited after admission or during this campaign.

## Byte-verified evidence export

The full export is `docs/reviews/probes/r109_luna/runtime_export_r109-interrupted-20261017/`. Its `r109_worker_evidence_manifest.json` lists 306 byte-verified files: the original R109 runtime tree, all Root browser screenshots/DOM/metadata and POST reservations, the exact browser interruption record, Root's complete interruption-cleanup command receipts, the authoritative Root B matrix, and the post-interruption source-pin check. Every included source/copy pair has matching size and SHA-256; bytes were copied without line-ending or content transformation. The campaign's empty `docker-cli-config` directory is recorded. The manifest is 127,576 bytes with SHA-256 `ed804155327eeb0cd511185e15d8647cffeae24ecc1e7eb0cd51c2edcb56815f`; a final pass verified all 306 included byte pairs.

The reused 58,031,616-byte binary was not copied; the manifest records its original ignored-runtime path and SHA-256 `cb73ee13e0369cca75ea276cc668f30c3ced42cead5f62c56d68f3c1eaa66f93`. Root's source/binary pin verification remains available at `docs/reviews/probes/r109_root/binary_reuse_verification.json` and `docs/reviews/probes/r109_root/post_runtime_source_verification.json`.

The export explicitly records two gaps: the original supervisor cleanup/disposition was not observed, and owned tab closure after browser-policy denial was not observed. Root's separate fixture recovery PASS is preserved and does not fill either gap.

## Gate transcript and review boundary

The first worker-return gate ran before Root's factual wording corrections; its complete stdout/stderr, exit code and duration are in `docs/reviews/probes/r109_luna/worker_return_gate/`. The second gate ran after correcting the admission-check receipt wording; its transcript is in `docs/reviews/probes/r109_luna/worker_return_gate_recheck/`. The final gate ran after correcting the browser-call evidence description; its transcript is in `docs/reviews/probes/r109_luna/worker_return_gate_final/`. All three runs reported six PASS controls, zero failures, and no NOT_APPLICABLE lines. The gate is a static repository check only; it does not convert NOT_RUN browser cases or missing supervisor finalization into passes. Independent review and disposition remain with Root.

```text
[PASS] CVF-DG-INST-01 PASS
[PASS] CVF-DG-CONT-01 PASS
[PASS] CVF-DG-RANGE-01 PASS
       note: WORKTREE_SCOPE mode=WORKTREE_SCOPE base=None head=7fc2273b485b765543dd78fc60afc51ee68d0cb3 candidates=425 deleted=0
[PASS] CVF-DG-APPL-01 PASS
       note: checked: docs/work_orders/CCMAI_RUNTIME_109_HUMAN_UI_COMPLETION.md
[PASS] CVF-DG-ROLE-01 PASS
       note: checked: docs/reviews/R109_HUMAN_UI_WORKER_RETURN_2026-10-16.md, docs/reviews/R109_ROOT_UI_OBSERVATIONS_2026-10-17.md, docs/reviews/probes/r109_luna/runtime_export_r109-interrupted-20261017/root_ui_observations.md
[PASS] CVF-DG-CLAIM-01 PASS
       note: 0 explicit claim line(s) and 4 document(s) scanned
RESULT: PASS phase=worker-return mandatoryControls=6 failed=0
```

### Corrected-packet worker-return recheck

Command: `python -B scripts/cvf_gates/cvf_downstream_gate_runner.py run --phase worker-return --project-root . --packet docs/work_orders/CCMAI_RUNTIME_109_HUMAN_UI_COMPLETION.md --packet docs/reviews/R109_HUMAN_UI_WORKER_RETURN_2026-10-16.md`. Exit code 0; duration 1,116 ms; stderr 0 bytes. The exact stdout, stderr and exit receipt are preserved under `docs/reviews/probes/r109_luna/worker_return_gate_recheck/`.

```text
[PASS] CVF-DG-INST-01 PASS
[PASS] CVF-DG-CONT-01 PASS
[PASS] CVF-DG-RANGE-01 PASS
       note: WORKTREE_SCOPE mode=WORKTREE_SCOPE base=None head=7fc2273b485b765543dd78fc60afc51ee68d0cb3 candidates=430 deleted=0
[PASS] CVF-DG-APPL-01 PASS
       note: checked: docs/work_orders/CCMAI_RUNTIME_109_HUMAN_UI_COMPLETION.md
[PASS] CVF-DG-ROLE-01 PASS
       note: checked: docs/reviews/R109_HUMAN_UI_WORKER_RETURN_2026-10-16.md, docs/reviews/R109_INDEPENDENT_INTERRUPTED_REVIEW_2026-10-17.md, docs/reviews/R109_ROOT_UI_OBSERVATIONS_2026-10-17.md, docs/reviews/probes/r109_luna/runtime_export_r109-interrupted-20261017/root_ui_observations.md
[PASS] CVF-DG-CLAIM-01 PASS
       note: 0 explicit claim line(s) and 5 document(s) scanned
RESULT: PASS phase=worker-return mandatoryControls=6 failed=0
```

### Final-packet worker-return recheck

Command: `python -B scripts/cvf_gates/cvf_downstream_gate_runner.py run --phase worker-return --project-root . --packet docs/work_orders/CCMAI_RUNTIME_109_HUMAN_UI_COMPLETION.md --packet docs/reviews/R109_HUMAN_UI_WORKER_RETURN_2026-10-16.md`. Exit code 0; duration 1,167 ms; stderr 0 bytes. The exact stdout, stderr and exit receipt are preserved under `docs/reviews/probes/r109_luna/worker_return_gate_final/`.

```text
[PASS] CVF-DG-INST-01 PASS
[PASS] CVF-DG-CONT-01 PASS
[PASS] CVF-DG-RANGE-01 PASS
       note: WORKTREE_SCOPE mode=WORKTREE_SCOPE base=None head=7fc2273b485b765543dd78fc60afc51ee68d0cb3 candidates=435 deleted=0
[PASS] CVF-DG-APPL-01 PASS
       note: checked: docs/work_orders/CCMAI_RUNTIME_109_HUMAN_UI_COMPLETION.md
[PASS] CVF-DG-ROLE-01 PASS
       note: checked: docs/reviews/R109_HUMAN_UI_WORKER_RETURN_2026-10-16.md, docs/reviews/R109_INDEPENDENT_INTERRUPTED_REVIEW_2026-10-17.md, docs/reviews/R109_ROOT_UI_OBSERVATIONS_2026-10-17.md, docs/reviews/probes/r109_luna/runtime_export_r109-interrupted-20261017/root_ui_observations.md
[PASS] CVF-DG-CLAIM-01 PASS
       note: 0 explicit claim line(s) and 5 document(s) scanned
RESULT: PASS phase=worker-return mandatoryControls=6 failed=0
```
