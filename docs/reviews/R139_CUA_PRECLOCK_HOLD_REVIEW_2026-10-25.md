# R139 CUA original preclock launch — independent HOLD review

Status: HOLD_CURRENT_HOST_BOOT_DRIFT_BEFORE_RESERVATION
Risk: R2
Worker: Root original-control custodian
Reviewer: independent `/root/r139_cua_preclock_reviewer`
Date: 2026-10-25
Scope: one actual original controller launch at committed technical packet; no G0 campaign or product proof.

## Evidence and finding

The normal clean technical commit was `fd730262bd55d73d7bdc681d891f0429b912997a`. At review, `git status --porcelain=v1` was empty. Actual `HEAD` blobs exactly equal working bytes for sealed recipe SHA-256 `514373d7c4d9dd0bc946e7d7088d5fb8fa541fd5cec351096f0e6d14abbee0d0`, technical authority `eab4bbcdcabb00b5dd006ece8404bf2ec15583d9db5021b4858bbcab4453b1e8`, and the three corrected authority blobs `0ee21ae6…`, `fa7f2f34…`, `bd21d9e9…`. This settles the prior staged byte-custody condition; it does not settle runtime G0.

Root's ignored outer custody wrapper recorded one exact command under the project root, recipe SHA and dispatch HEAD. Its original `Popen` returned PID 24484 at `2026-10-25T07:49:26.115379+00:00`; its original `p.wait()` returned exit 1 at `07:49:34.046163+00:00`. The exclusive custody receipts are `.cvf/runtime/r139_cua_reentry_20261025_readiness/control_outer/{intent.json,started.json,original_wait_exit.json,stdout.bin,stderr.bin}`. Their SHA-256 values respectively are `830d8830d6cbb2de707d845ad04423593f96b03a0f1cc8430321458f28220848`, `875f0a8e0f6ba3894fe46fc9fc89fb9056ae464d8689ca89e01046a9e391bd8e`, `35ffc826049d3c82929c8c63c1a1b40e97933a91d424e5d2cf12c9e845955356`, `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`, and `1a53afae13441a09e3d3adf183f4cf5dee7023c8196bad293d30120422569bd7`. The controller's traceback names `CURRENT_HOST_BOOT_DRIFT` at frozen preclock line 452.

The controller's existing child 014 PowerShell OS metadata query returned exit 0 and `WAIT_RETURNED` within its five-second cap, with no stderr. Its recorded stdout is exactly `|2026-10-23T05:48:27.5037950Z` (SHA-256 `e3e1bbd2bfb39f843a2ce94aa45b56d756340f4b72bdb31e05a7a1ef81fd7d26`); the child receipt SHA-256 is `2d97c860dddd940877ca2842a54d026ba49e677dc686b29b480e42d452bec6df`. The committed boot proof requires `|2026-10-23T05:48:27.5010520Z`. The boot-time strings differ by 0.002743 seconds. The same command and scrubbed environment produced the mismatch at the controller's exact equality check. The evidence establishes an unstable exact string for this check; it does **not** prove a host reboot, an operating-system defect, or a safe replacement comparison.

The failure occurred before the reservation write and signal. `campaign_reservation.json`, `root_g0_release.json`, and `reentry_activation_receipt.json` were absent at review. No service query, CUA method/page, fixture, key content read, provider attempt, or product process was released by this launch; no G0 capability or product acceptance is claimed. The originally approved source checks remain eight of eight consumed, including the retained failed worker attempt. The fresh campaign was not reserved; this fact is not authority to relaunch the failed original preclock transaction.

## Disposition and next boundary

**HOLD.** The earlier `R139_CUA_G0_TECHNICAL_RELEASE_REVIEW_2026-10-25.md` remains a static preclock packet REVIEW_PASS. Actual original launch failed its independent host-boot assertion, so G0 is NOT_RUN, runtime/key/provider admission is absent, product B remains `BLOCKED_PENDING_OPERATIONAL_ADMISSION`, and R139 remains one open order with Facebook/Zalo OA parked. Preserve the original process/child receipts, clean committed packet, original launch count one, reservation count zero, and all old spent/unknown histories. Do not retry this controller, alter the committed boot proof or frozen controller in place, run another OS query, or infer a refund/reset.

A continuation requires one consolidated contract decision within the existing R139 scope: specify a defensible host/boot identity check with failure rules and evidence, bound any necessary frozen-controller/pin change, independent R2 review, a new exact launch/effect authority, and truthful continuity/commit before any later attempt. Determine whether the earlier exact boot string was derived from an intrinsically variable source or another cause from the preserved evidence; the physical cause is unresolved. No new tranche, helper order, automatic browser fallback, or product effect follows from this review.
