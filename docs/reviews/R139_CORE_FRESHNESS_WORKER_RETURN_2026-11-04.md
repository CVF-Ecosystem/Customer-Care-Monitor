# R139 core freshness worker return — 2026-11-04

Status: COMPLETE_PENDING_REVIEW

Scope: finite static checker repair under `CCMAI-RUNTIME-139-CORE-FRESHNESS-20261104-ADMISSION.json`; preparation `38952df8bcf9e01b18a05a1da1095b1e7eda6fd8`. Worker IMPLEMENTATION_WORKER; Root and separate Terra remain independent reviewers and Root owns commits/adoption. Worker source and regression work is complete; independent review and all actual Core/project transaction conclusions remain with Root.

The checker adds one fixed second successor for Core `15e39450853184eb65dbde73398c69c3171a6241` to `c4c8767ac8861854f40d466ba338ffd11eb9558e`. Original first-repin constants, admission and route remain intact. The second route reads immutable seed, maintenance and first-repin evidence from their committed first blobs and payload snapshots, validates their identities/ancestry, and compares the new exact eighteen-file map with the real active range and worktree. Only the five admitted existing payloads may change. Other historical payloads, all installed sources, workflow and lock metadata remain frozen. Staged regular-file modes, digests, authority/payload atomicity, aliases, duplicates and successor deletion are checked.

`worker_transaction.ps1` is an uninvoked SOURCE CANDIDATE only. Default invocation returns inertly. Root's independently reviewed exact source hash and explicit Execute invocation are required; there is no release JSON. Its fixed preparation/authority/preimage/Core pins, local mirror, child-only Git environment, permanent attempt latch and updater handling remain subject to Root/Terra source review. No preflight, mirror, updater, installer or project adoption was performed by the worker.

Root accepted consolidated candidate findings before any invocation: remove the earlier proposed release JSON entirely; reuse the validated-parent collision scan rather than repeatedly walking all ancestors; and retain input-lock custody when updater/descendant quiescence is UNKNOWN. The revised source samples at most 64 scoped PID/creation/executable identities during the original updater wait. A 600-second timeout ends acceptance and writes UNKNOWN; the same script process and original handles remain alive until original-updater/known-identity quiescence is observed. Unavailable CIM retains custody. Late quiescence never becomes acceptance PASS; sampled absence is qualified, and no separate guardian, kill, retry or restore is introduced. Root owns the conditional final source/parse check; no worker wrapper invocation occurred.

Frozen checker raw SHA256: `89090d14d8eebbc90f65305013a5555b158846e5c6ed07208c6e003d8c5fb5ca`; frozen test raw SHA256: `d109dafcbbc006fbca1f21d8bf6b19c8b8582779c7a7cce3d56770bd1f7edad3`; final transaction candidate raw SHA256: `81a20833a9dc187c72b920730e0af6501a909551030e1f433621b476237a19e2`.

Peer's remaining predicate finding was included in the same consolidated candidate repair: queried same PID plus matching known nonempty creation means alive regardless of executable path; multiple/missing/ambiguous identities hold custody; only no PID or different known nonempty creation establishes original-identity absence. Executable is diagnostic only. Null original process after launch assignment is explicitly UNKNOWN. Root's earlier parse evidence is for earlier source bytes; the worker makes no final-candidate parse PASS claim. Final small predicate change requires independent manual source review.

Root separately reported two failed operational preflights with no updater/mirror/latch: inherited `GIT_PAGER` was correctly rejected; then native command discovery returned two executable paths and made an invalid ProcessStartInfo filename. Root accepted the latter as a new independent host root cause. Candidate repair 2 changes exactly the Git and pwsh command discovery expressions to `Select-Object -First 1`, preserving normal command-resolution order. Custody/checker/tests are unchanged; no worker check batch or wrapper invocation was added. Root retains the original failed preflights and owns any bounded changed-source preflight.

## Verification and retained failure

Batch 1 focused: **20 tests, 4 failures, 0 errors, 0 skips**, 271.722 seconds. Original outputs, raw source hashes and child subprocess records are retained under `core_freshness_20261104/worker_focused*`. The original fixture exposed 27 effective files because it omitted inherited seed exemptions and included an automatically allowed historical path. The production checker correctly retained the eighteen-file constraint. Root accepted one consolidated fixture-only repair: preserve original test defaults, commit the second fixture's inherited seed exemptions from the beginning, and model seventeen historical out-of-seed payloads plus the existing new test file. Three diagnostic failures cascaded from the fixture-count rejection; final full-suite results determine their resolution.

While batch 1 ran, final static readback added explicit staged regular-file mode validation and its matching-blob/symlink-index regression. Batch 1 therefore supports only its recorded initial source hashes; it is not evidence for final source bytes. Checker/test edits were frozen before batch 2.

Batch 2 full gate units: **124 tests PASS, 0 failures, 0 errors, 0 skips**, process duration 729.542 seconds (unittest reports 728.593 seconds). The prior baseline was 113; eleven new fixed-second-successor tests increase the expected suite count to 124, subject to future separately authorized test changes. Original outputs and child subprocess records are retained under `worker_full*`; the receipt pins exactly the frozen checker/test hashes above. All original focused failures remain preserved and resolved by the full batch. No failed batch was overwritten or silently repeated. No separate repair test batch was invoked.

## Source and cost accounting

Original clock receipt: `worker_started.json`, UTC `2026-11-04T12:33:49.771698+00:00`, monotonic `1061177.2530539`, before source edits. Ceiling: 40 wall minutes, eighteen semantic input sources and two initial test batches; one consolidated repair only for accepted findings. At full-batch launch, elapsed 703.682 seconds; unmeasured billing and prior branch wall time remain UNKNOWN.

Completion receipt sampled UTC `2026-11-04T12:58:13.873387+00:00`, monotonic `1062641.3547155`, elapsed 1464.102 seconds (24 minutes 24 seconds), before final return wording. Two test batches total, zero separate repair batches; child subprocess originals captured: focused 2045, full 4866. Negative fixture commands can intentionally return nonzero; suite outcomes above govern test conclusions. `worker_completion.json` records exact batch commands, counts and final source hashes. Candidate command discovery repair 2 has its distinct Root-accepted host cause; no additional worker check was run for it.

Ten semantic sources total, including all eight carried feasibility sources:

1. `scripts/cvf_downstream_gate.py`.
2. `scripts/tests/test_cvf_downstream_gate_core_repin.py`.
3. First `CORE-REPIN-MAINTENANCE-ADMISSION.json`.
4. Cached public Core `docs/EXPORT_MANIFEST.md` diff.
5. Cached public Core product milestone coverage contract diff.
6. Public Core `scripts/update_cvf_workspace_public_core.ps1`.
7. Public Core `scripts/lib/downstream_governance/cvf_dg_install.py`.
8. Public Core workspace doctor source.
9. Prior `core_reconciliation_20261029/run_core_update.ps1`.
10. Current `core_freshness_20261104/root_preimages.json`.

Current canonical startup/active authority reads are exempt under the admission. No new broad domain audit, provider/browser/product effect, native adapter code, Core update, network access, authority/continuity/catalog edit or commit occurred. Historical costs, doctor FAIL27/28, pending native source admission, physical relaunch HARD_HOLD, integrated lifetime NOT_DESIGNED, five Core product applicability UNKNOWNs, full R139/M3/FREEZE OPEN and parked Facebook/Zalo OA remain unchanged. The next move is independent source/transaction-candidate review, not runtime or product acceptance.
