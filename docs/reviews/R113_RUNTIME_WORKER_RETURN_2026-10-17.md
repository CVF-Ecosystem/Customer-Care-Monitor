# R113 Runtime Worker Return — 2026-10-17

Status: COMPLETE_PENDING_REVIEW
Risk: R2
Worker: /root/r113_readiness
Reviewer: Codex /root
Disposition: RETURNED_WITH_QUALIFICATIONS

## Outcome and claim boundary

The sole admitted R113 UI campaign reached `UI_READY_WAITING_ROOT` and supplied bounded synthetic observations. Root observed B07 reader controls without write actions, B08 at a 390×844 DOM viewport with queue/detail/history/owner forms mostly readable, and the current wording. B08 is partial: the shared onboarding banner clips at mobile width, including the developer-facing text `Cấu hình AI Provider`. B09 is unsupported by the advertised UI and remains `NOT_RUN`. An isolated `End` key press timed out; a fresh DOM remained responsive, the owner assignment choice was applied, and no retry or alternate automation stack was used. No AI-governance or provider behavior is claimed.

B07: `PASS_UI_CONTROLS` as observed by Root — fresh reader login after owner logout; queue/detail readable; save/action/reason controls absent; case revision stayed 1. B08: bounded evidence, not a global pass. The browser DOM CSS viewport was 390×844. Root captured the queue at 390×843 and detail/form/history at 375×811; original JPEG bytes were retained and PNG derivatives were not resized. An exact 390×844 raw PNG was not observed; no 844-pixel raster claim is made. Root-owned browser captures remain in `docs/reviews/probes/r113_root/browser/` and are not duplicated in this worker export. The existing baseline remains the accepted R110 input: UIa28cdd0a, runner9e7620b1, 26 pass / 3 known Dashboard failures, typecheck pass. No product source, tests, native binary, or continuity files were changed in this runtime tranche.

## Admission and binding

- Campaign: `r113-1792205107-25388`; project HEAD remained `0a26662386ed2d3c040433a8e81398a80b481048`.
- Reviewed source commit: `0622d185bfba8eae960138b57acd9a82f34842fd`.
- Runtime admission SHA-256: `c69adc06b1744222f5f671ac8d5745a2cca693e891a31fc74197393f00e5919e`.
- Seed SHA-256: `62a993c5f37f7765418847c9f953f4f983a35f12a0923d3e06b58dde4e025fba`.
- Work order SHA-256: `4fd7cc3e1a3a7efc4e9d8e73c4175319fce819c3107c4f19d5f6f97000d0e881`.
- R113 runner SHA-256: `79a5dfb98de2e49dc258653b727816638664bc5e19e715129c8b43768e0f59df`.
- Vite template SHA-256: `e5bdedb9f92065279219125cacd6b118d3072bed0d28ffc0d8f1b24749d1d65c`.
- Pinned helper SHA-256: `899afa92934dede2d99d28d591bc4232d29a52f764d10f8420588e8e940fb138`; Docker SHA-256: `6c699720bb9523dedb9eb5d22a49951ea9210dbdb1314a3554029eee9be76896`; Node SHA-256: `3602f2bb1a10f2cbab4c36886218a33c1ab3db87290e73b033c46c77147d0237`.
- Reused compiled binary (not included in export): 58,031,616 bytes, SHA-256 `cb73ee13e0369cca75ea276cc668f30c3ced42cead5f62c56d68f3c1eaa66f93`.

## Bounded runtime and cleanup

One fixture start, one backend launch, and one Vite launch were reserved and used. Startup completed in 2.725 seconds. The campaign allowed 8 UI/API POST submissions including auth; the ledgers record 6 submissions and 6 Gin observations (3 login, 3 logout), leaving 2 unused. Lifecycle/action submissions: 0. Provider requests: 0. Native invocations/Go compiles/retries: 0. The 1,200-second campaign limit and 75-second cleanup reserve were not expanded.

The worker issued the single cooperative `stop` command, which exited 0 and returned `STOP_REQUESTED`; it wrote `stop.request`. Polling the owned supervisor session `75592` then returned `Unknown process id`. A read-only process check found supervisor PID 25388, backend PID 23820, and Vite PID 24072 absent. The worker did not kill processes, issue a second start, or fabricate terminal success. The runner's `cleanup.json`, `campaign_result.json`, and `campaign_disposition.json` remain missing and were not rewritten.

Root independently validated the admitted fixture shape, issued one exact target stop, and then verified the R113 fixture exited, its network was empty, the target was the only volume user, and protected inventory matched both campaign baseline and pre-Root inventory. The independent receipt is `docs/reviews/probes/r113_root/independent_poststop/final_state_receipt.json` (SHA-256 `6c2045297badf9434ad3bbc0b017aacc005615e000c385ee764a0ef00efa5108`). Its result is `FINAL_FIXTURE_EXITED_PROTECTED_UNCHANGED`. This corroborates final fixture/protected state; it does not supply the missing runner-owned cleanup receipt or resolve why the supervisor session disappeared.

## Raw campaign export

The export at `docs/reviews/probes/r113_luna/runtime_export_2026-10-17/` contains all 196 raw campaign files, byte-for-byte verified against `.cvf/runtime/r113_workflow/campaign/`, totaling 80,089 bytes. Raw-tree SHA-256: `d01a867d49f2346ddc5695c099fb771899e8137c886e17574d07e1aec7fffb71`. The archive `campaign_raw.zip` is 59,016 bytes, SHA-256 `93b77944eb664a5c49f1fc3629729ae66ee5de795770051987cb018f29da60b3`. `manifest.json` lists each relative path, byte size, SHA-256, archive hash, stop/cleanup distinction, and exclusions.

No `.env` was opened, copied, or included. Campaign intent describes the environment as SystemRoot/WINDIR/TEMP/TMP plus explicitly declared synthetic fixture values, with no `.env` or provider settings; `ready.json` records `syntheticCredentialsIncluded=false`. Provider credentials and Root browser captures are excluded. The compiled binary resides outside the campaign directory and is excluded as identified above. The raw export includes process stdout/stderr `.bin` logs exactly as captured; no campaign file was transformed or redacted.

## Closeability recheck

This return stays within the authorized R113 runtime evidence/export and worker-return artifact paths. Objective, R2 ceiling, effect bounds, and Root commit/review ownership are unchanged. The campaign has stopped and Root's independent final-state receipt is available; the worker return is ready for independent review. Classification is `COMPLETE_PENDING_REVIEW`, not accepted or frozen. Runtime acceptance remains bounded by the B08 mobile clipping, B09 `NOT_RUN`, and missing runner-owned cleanup/result receipts. There are no pending worker operations. Independent review, disposition, commit, and any continuity updates remain Root-owned. No runtime admission or campaign authority is implied beyond R113.