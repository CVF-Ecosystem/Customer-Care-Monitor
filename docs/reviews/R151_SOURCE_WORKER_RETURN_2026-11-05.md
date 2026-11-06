# R151 source worker return

## A2 source-only return — 2026-11-06

Status: BLOCKED_WITH_REASON
Risk: R2
Worker: Terra 6.1 IMPLEMENTATION_WORKER
Reviewer: Astra Medium independent REVIEWER

The A2 source pass is complete and its focused checks passed; V06 still requires independent static review of the exact final A3, positive WebSocket, A1-c and A2 paths. String-presence checks do not discharge that obligation. Independent review of the full diff and other acceptance obligations also remains. No source acceptance or runtime admission is claimed.

The runner now puts executable setup inside its CLI entry and exposes the production replacement and cleanup functions for injected fixtures. Guarded daemon stop, wait, kill, Chrome close, terminal inventory and persistence record failures; cleanup errors and missing terminal evidence prevent terminal ABSENT. The single offline test imported the runner and exercised the real cleanup and replacement seams with custody persistence throwing, timeout and zero replacement launches, and an unknown descendant. It also executed the existing PowerShell, ownership, HTTP transport and reviewed-manifest fixtures. This is offline diagnostic evidence only.

Candidate identities, raw SHA-256, are recorded in `docs/reviews/probes/r151_evidence_capture_20261105/worker_a2_source_manifest.json` (manifest raw-byte SHA-256 `80c51918248391b333b5aff9ee023eb17b940eb430b832ad4a2689b897fee523`; LF-normalized SHA-256 differs). Runner `0714964711391772de75f151ed97e7d27ba3f61c02f4056b3e535604b88ffcca`; tests `57370533b859c86cce93142b26bf4ca897d74a2c60bb7827a0e91ab7a266b740`. The manifest also pins the unchanged helper, actor, and PowerShell sources.

- Combined invocation: `node docs/reviews/probes/r151_evidence_capture_20261105/worker_offline_tests.mjs` redirected to `worker_a2_offline_test.log`, exit 0, 52 PASS, 0 skips. Raw log SHA-256 `11204b544f49a5721834edd0fb58251dc15a2aec6fc967779a057671c95a3c50`. Root's hold arrived after this invocation had already completed; Root was informed immediately.
- Final-byte syntax batch: `node --check` on owner_runner.mjs, diagnostic_helpers.mjs and worker_offline_tests.mjs; Windows PowerShell 5.1 `System.Management.Automation.Language.Parser.ParseFile` on process_snapshot.ps1, prestart_full_tree.ps1 and worker_historical_red.ps1. Each Node exit was 0, each parser reported 0 errors, batch exit 0. Raw `worker_a2_syntax.log` SHA-256 `775cd3f75f05a80820b4b13693fb84cf7900fc15393d4c1abaf200020cfe229f`. No source or test bytes were edited after the manifest or checks.
- This A2 allocation spent one of one focused test and one of one syntax batch. Historical four focused tests and one evidenced syntax remain spent; old unused syntax expired; earlier active time remains UNKNOWN. Release `2026-11-06T07:42:44+07:00`, source/check work stopped by `2026-11-06T07:46:40+07:00`, conservative elapsed under four minutes against the 20-minute active cap, hard wall deadline 08:12:44.
- Changed paths: `owner_runner.mjs`, `worker_offline_tests.mjs`, new `worker_a2_source_manifest.json`, `worker_a2_offline_test.log`, `worker_a2_syntax.log` under `docs/reviews/probes/r151_evidence_capture_20261105/`, plus this return. R150 and protected sources were not edited. No Chrome, daemon, socket, provider, DB, product, network or package effects were invoked.

### Return-Time Closeability Recheck

BLOCKED pending Astra's independent R2 source review, including final-hash V06 control flow and remaining obligations, followed by Root's gates and disposition. The worker has no further test or source correction allowance in A2, does not commit or alter authority, and did not dispatch effects. R150 remains INCONCLUSIVE, A2 UNRUN and terminal cleanup UNKNOWN; R139 and M3 remain blocked.

Historical original return: BLOCKED_WITH_REASON, risk R2, worker Terra 6.1 IMPLEMENTATION_WORKER, reviewer Astra Medium independent REVIEWER.

## Disposition

Same-scope continuation produced candidate source and offline fixture evidence. V01–V03 and V05 received bounded fixture coverage; V06 has source-bound positive/A2 controls and V07 retains commands and hashes. V04 is not complete: after the final green run, source inspection found that the runner could repeat an original wait after timeout. The candidate was changed to retain timed-out handle identity and prevent a second wait. This final runner edit has not been tested or syntax-checked, and the offline tests do not exercise the runner's complete next-launch and terminal sequencing. The worker has used all three permitted focused test invocations and the single syntax batch. Independent review and any further test authority remain with Root. No source or product acceptance is claimed.

## Work and evidence

Historical preimages were read from R150. New actor is byte-identical to the R150 preimage; the A3 helper is a copy with R151 output path and schema changed. New snapshot uses Windows argument parsing and exact normalized profile comparison. Candidate ledger retains observed PID/creation identities and terminal states; candidate negative capture uses bounded raw TCP parsing, a fresh WebSocket key, and one settlement/teardown. The runner retains A3, positive control, A1-c and A2 paths and requires a separate R151 effect admission. These descriptions are implementation intent, not independent acceptance.

Focused invocation 1: Windows PowerShell 5.1 `-NoProfile -ExecutionPolicy Bypass -File docs/reviews/probes/r151_evidence_capture_20261105/worker_historical_red.ps1`; expected exit 1, wrapped count 1 / match count 0. Invocation 2: `node docs/reviews/probes/r151_evidence_capture_20261105/worker_offline_tests.mjs`; 34 PASS, 0 skips, exit 0. Invocation 3: the same command after additions and changes; 39 PASS, 0 skips, exit 0. The syntax batch ran `node --check` on four MJS files and PowerShell Parser.ParseFile on three PS1 files; all exit 0 / zero parser errors at that time. The runner was edited afterward to integrate tested waitOriginal and then prevent repeated timeout waits; therefore syntax and green results do not attest the final runner bytes. Logs preserve actual results.

Raw SHA-256 of candidate artifacts at return:

- actor_call.mjs `e8809991c2d26a0ff8dc83665cc4ac93aa199221164ae3d4665a26f30bb7a8c1`
- diagnostic_helpers.mjs `6983d9b62966edd2b2a5fc5ec8892d7d76e5a24d7ec1c8a1ba7aa5d62933a607`
- owner_runner.mjs `d11012f6bb3bbdc681fd239eaccea068eb0dd68786682ebd145afea44f784ed9`
- prestart_full_tree.ps1 `8b1dec9c5f945f988b2a32a4804559c1ce44cf20225c902bd19985ed2bbd4db2`
- process_snapshot.ps1 `50b99cdbb0f7fd621edc5fe493bacd65e0dd0b56d8f5722ba4cf42a54f81d939`
- worker_offline_tests.mjs `fb01fb786e7ff85e0a7e45e12f24093ca00ab1de55f5d762eba8844a9e496c5a`
- worker_historical_red.ps1 `8a4de27d8b19445216cb6a3ac71ef01fc6e04192509cd5f7b8e83ca209b7fa98`
- worker_historical_red.log `3165702eb5823453c9e4bc2492753e1f4ce5476fce66d86c972530e15760d686`
- worker_green.log `4dc64b45f180ecd5fdb14de38517cc914f629fe305505ec9bf259a67a84c8d7a`
- worker_green_final.log `74f53b7f9ae53acfbe223b671a4a34be71e2c1d298aceefb413b48d1f6aa56c6`
- worker_syntax.log `4eb790d2fe29232f8927dec05dd9ae373c8bb6d5f373743605c1abe3a0efed4b`

All files above are under `docs/reviews/probes/r151_evidence_capture_20261105/`; this return is the only other worker-changed path. Root-owned `root_*` files were not edited. R150 files remain unchanged.

## Cost and boundary

Original release 16:17:18; resumed 16:23:14; source work stopped 16:30:16 on 2026-11-05 Asia/Saigon. Original hard stop 17:02:18; cumulative elapsed wall interval approximately 13 minutes. Actual active minutes remain UNKNOWN. Focused tests 3/3, syntax batch 1/1. Chrome, daemon, provider, database, product, listening socket and network effects: 0. R150 remains INCONCLUSIVE with A2 unrun and terminal cleanup UNKNOWN; R139 and M3-05/M3-08 remain blocked.

## Return-Time Closeability Recheck

BLOCKED. Independent R2 review, final-runner syntax and behavioral verification, including original timeout preventing next launch and terminal PASS, remain outstanding. Root owns continuity, scope, gates and commits. The worker did not self-review, commit or dispatch effects.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
