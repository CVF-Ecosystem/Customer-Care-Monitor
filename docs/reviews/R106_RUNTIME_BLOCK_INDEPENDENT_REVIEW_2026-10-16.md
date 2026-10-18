# R106 independent runtime blocker review

Status: BLOCKED_WITH_REASON
Risk: R2
Worker: /root/r105_successor_route (Luna Max REVIEW_EVIDENCE_OPERATOR)
Reviewer: Codex /root
Disposition: BLOCKED_RUNTIME_VITE_ESM_IMPORT_RACE
Source/seed base: 88f0c5e64185e538cfc47381a0820cacd73bab51
Dispatch commit: f9e87e576833551e513c7958f160918a6be090af
Freeze: OPEN

## Decision

Record the single failed runtime campaign and keep product acceptance open. The real historical R101 binary reached fixture readiness, but the owned Vite process exited before readiness. No running.json was produced; no product browser navigation, UI screenshot or application action followed. B01–B10 remain NOT_RUN. This is a runtime configuration/dependency loading failure, not an observed product UI defect or browser-policy denial. Root does not accept UI/product/governance readiness from source tests, successful backend seeding or static gates.

## Independently observed failure

Root read the original runtime child streams, readiness, three reservation files, process receipts, cleanup and protected inventories. Vite stderr reports ERR_REQUIRE_ESM_RACE_CONDITION: its ESM index was not fully loaded when vite-plugin-vuetify/dist/index.cjs required vite. The pinned template concurrently imports vite, @vitejs/plugin-vue and vite-plugin-vuetify via Promise.all. Root also read the installed plugin's explicit require('vite') statement. This establishes the observed import-order failure mechanism; a future sequential-import repair has not been executed or proved effective.

The unchanged template came from the accepted R105 candidate. Root's source review and injected offline tests did not expose this real Node/module-load race. That is a readiness/review gap owned by Root, separate from the worker's compliant one-attempt execution. Source13PASS and gate89PASS remain bounded static facts, not evidence that the actual Vite configuration starts. Preserve the failed template, exact Node hash, raw stderr and previous review rather than retroactively changing their hashes or acceptance wording.

Runner start was invoked once and returned HOLD/exit1. Backend ready origin was http://127.0.0.1:60641; Vite target was127.0.0.1:3011. Fresh synthetic tenant44e423df-ea44-4e06-b4da-adab32cc83fd and case68d1a6b3-b6fe-4ffd-804f-65ec15c465fe are historical failed-campaign identifiers only. They cannot be reused as readiness for another attempt. The three append-only reservations consumed fixture/backend/Vite1 each. R105 runtime stays absent and its counters remain zero; historical R101/R102 counters and immutable seeds are unchanged. Shared105/106 campaign remaining0, runtime retries0.

## Cleanup, protection and artifact checks

Cleanup.json exists with errors[] and actual backendExitCode1/viteExitCode1; retain these nonzero exits. The supervisor used its own finally cleanup on readiness failure, rather than reaching the later manual cooperative-stop branch. Do not represent an unrun cooperative stop as runtime-tested. The exact fixture stop command returned exit0 and the original container ID; the deprecation warning for --time is retained.

Root's separate read-only safe Docker/process verification is docs/reviews/probes/r106_root/post_runtime_independent_verification.json. Exact R101 fixture is exited, protected stable rows including R102 and original application/database match the captured before baseline, and recorded PIDs19068/7924 were absent at observation. No resource mutation or cleanup replay was used. TCP connect_ex returned10035 for3011/60641; this establishes no successful connection in that probe, not a no-listener/refusal assertion. All container/network/volume identities remain preserved; current safe fields do not prove every historical effect.

Root recomputed all38 entries of the runtime export manifest: original and exported bytes/SHA256/length match, zero mismatches, with the empty docker-cli-config directory present. Evidence is docs/reviews/probes/r106_cua/runtime/ and runtime_export_manifest.json. Runtime logs are original captured streams; the worker's supervisor/preflight receipt narrative is separately attributed and does not independently authenticate all tool history. There are no product screenshots or DOM observations because product UI was never opened. The Root HTTP health PNG remains prerequisite evidence only.

Root also recomputed all45 entries of the final worker evidence manifest: zero hash/length mismatches. Manifest SHA256903a4cb5e971ed8a01586c3d7941ac46da4a7897eaddafff22dcd114e12658d1 excludes its own self-hash. The separate captured_git_byte_contract.json records captured versus staged Git bytes: five text representations differ through normalization. captured_worker_evidence.zip preserves the exact45 captured files plus the manifest itself and the empty directory; Root verified all46 archive members against captured hashes/lengths. Restore original bytes from the named archive members rather than normalized checkout text. No raw worker file, manifest, Git attributes or Git configuration was changed.

Worker-return at the effective ACTIVE packet passed six controls, including APPL checked/PASS. Root reviewer-fast after blocking passed seven mandatory controls, with APPL explicitly NOT_APPLICABLE_WITH_REASON because the order is now BLOCKED; closeability is not re-proven by that N/A. Initial Root reviewer-fast caught missing exact Worker/Reviewer headers in the proposed R107 decision packet; Root corrected those documentary fields once, retaining both failure and corrected receipts. No source/runtime retry or repair occurred. Static gate results do not turn the failed campaign into UI acceptance.

No provider/mock, download/install, product source repair, new fixture, customer/channel operation or publication/deployment is admitted or claimed. The three Dashboard baseline failures, R101/UI FREEZE and full M2/M3/pilot/release remain OPEN; R102 local acceptance and Facebook/ZaloOA parked checkpoint remain unchanged.

## Finite next boundary

The campaign is spent; do not restart, delete reservations, change output suffix or mutate either immutable seed. Root saved an unapplied proposal at docs/reviews/probes/r106_root/proposed_sequential_imports.patch: await Vite import completely, then import Vue and Vuetify sequentially. It changes no actual source or generated config. A separately reviewed next-scope source/config and real module-load readiness check plus a new explicit finite runtime allowance are required before another fixture/backend/Vite campaign. The current order contains retries0 and admits no worker source repair. No automatic successor runtime or FREEZE follows from this blocked review.
