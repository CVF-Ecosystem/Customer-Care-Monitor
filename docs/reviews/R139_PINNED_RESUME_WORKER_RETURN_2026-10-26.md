# R139 pinned preflight and durable resume — worker return

Status: COMPLETE_PENDING_REVIEW
Risk: R2
Worker: Sol Medium /root/r139_pinned_resume_worker
Reviewer: Root independent source/evidence REVIEWER
Role: `/root/r139_pinned_resume_worker` IMPLEMENTATION_WORKER; Root is independent REVIEWER and COMMIT_STEWARD.
Order: `CCMAI-RUNTIME-139` for M3-05/M3-08. Authority: `CVF_SESSION/authority/CCMAI-RUNTIME-139-PINNED-RESUME-REPAIR-ADMISSION.json` at SHA-256 `78ae94c663cb2de1484c79da0efed6382cbf85c6bc1d791d208f87358b02cc0f`; committed dispatch `594d2a66ca14bc30f1f0272216bd39a4a4a62d48`.

## Changed source

- `docs/reviews/probes/r139_integrated_20261021/native_operational_20261024/r139_pinned_preflight.py.txt` SHA-256 `81784375b6948dee973ce1dd4a3b66851786273079f4becfbdd7c7378fbca233`: verifies public origin, cached pin ancestry, clean Core, workspace boundary, pinned public kit, exact installed 1.1 profile and lock identity. Exports immutable pinned Git object bytes to ignored project runtime and can invoke the existing seven legacy gates plus the exact trusted pinned profile. Cached `origin/main` and current checkout are informational; remote freshness is explicitly unchecked, and the current Core 1.2 doctor is not claimed PASS.
- `docs/reviews/probes/r139_integrated_20261021/native_operational_20261024/r139_durable_resume.py.txt` SHA-256 `7ab2bf4c5b97009d36b932fd7aff5cf6d5b42584844f59fc02f5605f06156eda`: exact frozen binding, source/input/checkpoint hashes, one-time anchor, exclusive OS lock, fsync/replace ledger, cumulative reservations, original external intent before effect, UNKNOWN on interruption, and exact receipt/result-file verification on reopen/reuse. Expiry preserves validated offline receipts but blocks new external effects pending cleanup and separate re-admission. It performs no external callback.
- `docs/reviews/probes/r139_integrated_20261021/native_operational_20261024/test_r139_pinned_resume.py` SHA-256 `9dd100bae4dcfa18b5c1fd098de56fecc534ad59fd1a4361bda0cbb256b1f227`: pure offline tests of mocked pin/provenance/export/check invocation and checkpoint failure/reuse paths.

## Original checks and interruption evidence

The first `worker_pin_resume` original child was invoked once through Root's ignored offline coordinator. PID 6296 returned exit 1 after 16 tests: 15 PASS and 1 assertion failure. The assertion expected the older version-drift wording while the strengthened anchor correctly rejected changed authority earlier. Its original stdout/stderr hashes, PID/wait/exit, intent, source hashes and raw files remain under `.cvf/runtime/r139_pinned_resume_repair_20261026/worker_pin_resume/`. The coordinator simulated a quota-like stop after the original wait and before checkpoint seal. Root recovered that exact original FAIL without a new child; no test result or prior counter was rewritten.

One authorized one-line test expectation repair changed only the regex. The successor `worker_pin_resume_repair` ran once in generation 2, retaining the prior complete ledger and receipts. PID 25072 returned exit 0; 16 tests PASS, 0 failures, 0 skips, original wait returned, source unchanged. Original result and hashes are in `.cvf/runtime/r139_pinned_resume_repair_20261026/worker_pin_resume_repair/`. Historical 10 source checks remain spent. This worker used two new focused attempts including the failed original; there was no automatic retry, provider call, key read, browser or runtime campaign effect.

## Claim boundary and next governed move

The helper is a reusable offline custody component. It is not yet wired into live G0–G3 execution and does not prove live governance or product acceptance. The receipt envelope checks structural binding to stage, input, frozen authority, saved external intent and exact original result file; Root must still custody the actual process/tool return. Root independently reviews source and the retained failure, then runs its separately reserved combined check and pinned preflight before any technical release. Product B remains blocked pending operational admission; Facebook and Zalo OA remain parked.
