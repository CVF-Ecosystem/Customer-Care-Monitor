# R139 CUA staged authority byte custody — independent review

Status: PASS_BOUNDED_METADATA_CORRECTION_WITH_COMMIT_CHECK_QUALIFICATION
Risk: R2
Worker: Root COMMIT_STEWARD (exact-byte staging author)

Reviewer: independent R139 REVIEWER `/root/r139_cua_preclock_reviewer`  
Date: 2026-10-25  
Scope: staged storage bytes of three existing authority JSON records only; no source, software check, runtime, browser, service, credential, or provider effect.

## Authority and method

Current compact bootstrap, active state/handoff, memory front marker, implementation phase, and docs index agree on BUILD/preclock only, R2, one R139 order, campaign/service/browser/key/provider counts zero. Public Core `b69e1088978e8c90ac58a2243fcf8f90d0f857ed` matches `origin/main`; the manifest's older `e9ddcbcd...` pin warning is inherited. Root reported current workspace doctor 27/27 PASS. The operative probe is `docs/reviews/probes/r139_integrated_20261021/native_operational_20261024/cua_git_byte_custody_correction_20261025.json`.

I read HEAD and index blobs with `git show`, the working files as raw bytes, and compared SHA-256 and parsed JSON values. This is metadata inspection, not a source check. `git diff --cached --name-status` listed precisely these three staged authority paths at inspection time. `git diff --name-status` listed other Root preclock drafts, but none of these three authority files had an unstaged diff. No Git configuration, attribute, hook, product source, or binary change was staged with the correction.

| Existing authority record | HEAD SHA-256 | Index and working SHA-256 | Parsed HEAD = index | Index = working bytes |
| --- | --- | --- | --- | --- |
| `CCMAI-RUNTIME-139-INTERNAL-CUA-SOURCE-ADMISSION.json` | `6a952adb494854da29d5b182cad3ae82511fb573ace82dc79a0b2559fa75ebf9` | `0ee21ae60ea9f3248b36af58ed003892f3e4c1f088c687f8683f9ae4f37ddd9e` | YES | YES |
| `CCMAI-RUNTIME-139-CUA-OBSERVATION-CAPS-ADMISSION.json` | `a33adf4e374175641821d3b76572f044c88afdc0f63279eb7566ce65bc4aa9dd` | `fa7f2f344c75b6900f1fb65d74c9327e9e53d1957714226d880fa5a30eab9491` | YES | YES |
| `CCMAI-RUNTIME-139-CUA-ACCEPTED-BUILD-PINS-ADMISSION.json` | `2f73ee83d61d0eb415dfe71574a12f123c3b0419b0ed53a0da0180a2052874dd` | `bd21d9e97cca7a32f350912ec23c0d53b39ed0b61ced90a435aad7e6b8e9e32b` | YES | YES |

HEAD stores LF; the approved working and staged blobs contain CRLF on all 156, 74, and 100 lines respectively. The staged blob object IDs are `4fc06f97291ec08962624a5ef3da568ddf0d8226`, `309d9358fd3cf2a89672a68f3ca40cf48099b9d5`, and `51b5db711df9ec72d27e0bf44e12e315855726a3`. Parsed equality establishes no changed authorization fields, caps, actor assignments, or effect permission. The active tranche's dispatcher seed remains `CVF_SESSION/authority/CCMAI-RUNTIME-139.json`; these three are supplemental Root-owned admission/pin records, and the worker did not edit them.

The frozen activation design's `committed()` function compares the SHA-256 of working bytes with a pin and requires `git show HEAD:<path>` to equal those bytes exactly (`reentry_activation_design.py.txt`, lines 151–154). The existing LF blobs would fail that exact-byte condition against the approved CRLF pins. The staged raw blobs resolve that narrow mismatch when committed normally. This review does not execute or validate the activation design.

## Commit qualification and disposition

`git diff --cached --check` currently reports CRLF lines as trailing whitespace because local Git whitespace handling does not exempt carriage returns. This is a diff diagnostic on the deliberate byte correction, not an authorization-field difference. Root must use the normal commit path with configured identity and hooks intact and retain the actual commit/gate result. If a mandatory gate or hook rejects the staged bytes, HOLD; do not bypass or weaken the gate. The post-commit `HEAD` blobs and working files must still match the three approved SHA-256 values before any original controller reservation.

The first reviewer-fast profile invocation after this review was written reported `ROLE` / `RISK_MISSING` for this review record. I added the required `Risk: R2` metadata line without changing the byte evidence, verdict, authority fields, caps, or formal G0 review. The failed gate receipt remains part of the review history; Root must rerun the affected governance gate and retain its actual result.

The next pre-commit profile invocation reported `ROLE_FIELD_INVALID` because an R2 review requires exactly one nonempty `Worker:` line as well as the existing independent `Reviewer:` line. I read the installed `cvf_dg_review.py` role declaration rule and added the single Root worker line above. Both failed profile results remain recorded; Root must retain the final actual gate result. This documentary header correction does not change any reviewed authority byte or formal G0 review hash.

**Verdict:** PASS only for staged metadata byte custody and semantic equivalence. No G0 technical release, campaign reservation, service query, browser action, fixture, key read, provider attempt, product acceptance, or FREEZE is approved here. Final G0 recipe/technical authority require a separate independent review of exact committed references and renewed actual actor acknowledgments. Historical readiness acknowledgments do not establish current actor presence.

## Additional unexecuted Root process custody inspection

I inspected `.cvf/runtime/r139_cua_reentry_20261025_readiness/root_original_control_custody.py` as static text at SHA-256 `ebe1e4a76a9c9a538ded30e9076d2d935967ddd7c3615f99111d67938a126823`. It checks a supplied recipe SHA and `COMMITTED_PRECLOCK` status, hashes the frozen controller design against `57ae20a2bb0054e38d5a3d919f5e2997780b56981a82f2f31924eddf623e5854`, writes the controller with exclusive creation, creates an exclusive custody directory, records an intent, invokes one `subprocess.Popen` without a shell, records the returned PID, calls the original `p.wait()`, and records its return code. No retry, process kill, or controller mutation appears in this wrapper. I did not execute or import it.

The wrapper is ignored local material, so its reviewed bytes are not protected by a repository commit. Root must recheck that exact SHA-256 immediately before the one original launch and retain the actual bytes/receipt with the campaign evidence. The wrapper relies on Root supplying the reviewed repository path and on the launched controller's own preclock checks for exact committed inputs; it is not a standalone admission gate. At this inspection, the recipe remains `DRAFT_PENDING_COMPLETE_CANONICAL_SEAL_AND_INDEPENDENT_REVIEW` and the current actor acknowledgments are pending renewal. This inspection does not widen the byte-correction verdict or start the campaign.
