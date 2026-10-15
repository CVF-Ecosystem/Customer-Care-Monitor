# R103 pinned Chromium install admission

Status: COMPLETE_PENDING_REVIEW
Risk: R2
Worker: /root/r103_tooling_worker
Reviewer: Codex /root
Disposition: ADMIT_REQUIRED_CHROMIUM_INSTALL_ONLY

Root independently recomputed npm archive SHA512 against saved official metadata and all335 installed package-file hashes/sizes plus installed Node hash; CLI independently returns Version1.57.0. Driver1.57 cache repair PASS, old browser top-level inventory unchanged; actual npm2requests/1989716bytes/0redirects. Pinned Chromium plan inspected: chromium and headless shell1200 (143.0.7499.4), ffmpeg1011/winldd1007 already cached. Only cdn.playwright.dev and playwright.download.prss.microsoft.com dry-run hosts, no unexpected URL. Root also captured per-file old browsercache hashes before browser effect, beyond worker's top-level inventory.

Admit ONE install-browser command using unchanged source c0fb1ee114db689e8cd71eb310534569de1514378c8f6fbf272a78ec40b2e15a: fresh installed-manifest/version revalidation, explicit canonical cache path, PLAYWRIGHT_SKIP_BROWSER_GC=1, no global settings. Download/install only required Chromium-related artifacts from the pinned official plan; no install-deps/OS changes. Preserve every old file, raw command/output/exit, emitted URLs/request/size observations and actual attempt count. Stop on failure; do not rerun/install alternatives. Browser CLI handles its own official redirects; post-hoc emitted-URL validation is observational, not proof of enforced redirect/transfer bounds. Record any downloader retry or unobserved metric honestly, not as network0. No UI/page/app fixture/server/provider effects are admitted to worker.

After success Root independently validates browser versions, old file hashes and installed manifest, then tests local blank-page driver launch/control/capture only. Managed Antigravity/Codex integration and R101 UI are still NOT_RUN until actual supported browser observations. No user app restart/kill or policy bypass.
