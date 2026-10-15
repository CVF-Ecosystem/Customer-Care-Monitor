# R103 driver installation return — single admitted campaign

Risk: R2
Worker: /root/r103_tooling_worker
Reviewer: Codex /root
Date: 2026-10-15
Tranche: CCMAI-RUNTIME-103
Role: IMPLEMENTATION_WORKER, after Root's `SOURCE_REVIEW_PASS_INSTALL_DRIVER_ONLY` admission
Reviewer and commit steward: Codex `/root`

## Result

The one admitted command completed successfully with process exit code `0`:

```text
python -B docs/reviews/probes/r103_tooling/repair_driver.py install-driver
PASS; receipts: .cvf/runtime/r103_tooling/20261015T140402Z_install-driver_83073f48
```

The runtime result is PASS in 40.95 seconds. The official package installed at `C:\Users\tiennm\AppData\Local\ms-playwright-go\1.57.0`; its CLI reports exactly `Version 1.57.0`. Existing Node was `v24.19.0`; source and installed `node.exe` hashes match at `3602f2bb1a10f2cbab4c36886218a33c1ab3db87290e73b033c46c77147d0237`. The package manifest records 335 files totaling 8,290,562 bytes.

## Official npm provenance and request evidence

The script made two HTTPS requests to `registry.npmjs.org`, both returned HTTP 200, and no redirect occurred. Metadata response size was 2,684 bytes; the exact tarball `https://registry.npmjs.org/playwright-core/-/playwright-core-1.57.0.tgz` was 1,987,032 bytes. Its SHA-256 is `8d2942f575e1b772bae8bd2c4d71f9055bf6581bc79d7001b2b277335b89ef2a`; npm's SHA-512 SRI check passed with `sha512-agTcKlMw/mjBWOnD6kFZttAAGHgi/Nw0CZ2o6JqWSbMlI219lAFLZZCyqByTsvVAJq5XA5H8cA6PrvBRpBWEuQ==`. Combined response bytes: 1,989,716. The portable `network.json` and `npm-requests.jsonl` contain the bounded public request summaries and per-request intent/result ledger.

## Pinned Chromium dry-run

The pinned `browsers.json` and CLI dry-run identify Chromium `143.0.7499.4`, revision `1200`, and headless shell revision `1200`; associated artifacts are ffmpeg revision `1011` and Windows `winldd` revision `1007`. The dry-run reported only the two admitted official hostnames, `cdn.playwright.dev` and `playwright.download.prss.microsoft.com`, with no unexpected host. The exact declared URLs were:

- `https://cdn.playwright.dev/builds/chromium/1200/chromium-headless-shell-win64.zip`
- `https://cdn.playwright.dev/builds/chromium/1200/chromium-win64.zip`
- `https://cdn.playwright.dev/builds/ffmpeg/1011/ffmpeg-win64.zip`
- `https://cdn.playwright.dev/builds/winldd/1007/winldd-win64.zip`
- `https://cdn.playwright.dev/dbazure/download/playwright/builds/chromium/1200/chromium-headless-shell-win64.zip`
- `https://cdn.playwright.dev/dbazure/download/playwright/builds/chromium/1200/chromium-win64.zip`
- `https://cdn.playwright.dev/dbazure/download/playwright/builds/ffmpeg/1011/ffmpeg-win64.zip`
- `https://cdn.playwright.dev/dbazure/download/playwright/builds/winldd/1007/winldd-win64.zip`
- `https://playwright.download.prss.microsoft.com/dbazure/download/playwright/builds/chromium/1200/chromium-headless-shell-win64.zip`
- `https://playwright.download.prss.microsoft.com/dbazure/download/playwright/builds/chromium/1200/chromium-win64.zip`
- `https://playwright.download.prss.microsoft.com/dbazure/download/playwright/builds/ffmpeg/1011/ffmpeg-win64.zip`
- `https://playwright.download.prss.microsoft.com/dbazure/download/playwright/builds/winldd/1007/winldd-win64.zip`

These are dry-run declarations only. No browser-download request was made, no browser was installed, and no UI/page/capture/app action occurred. The script's limitation remains: actual browser CLI redirects and aggregate browser-transfer bytes are not intercepted or enforced. The preexisting top-level browser cache inventory was unchanged; its prior Chromium revision 1228 and other entries were preserved.

## Portable evidence

Copied under this directory: CLI/Node stdout, stderr and command receipts; the browser-cache before/after inventory; dry-run output and plan; public npm request ledger and integrity summary; successful result; and full per-file installed package manifest. The raw npm metadata, `.tgz`, extracted staging package and `node.exe` were excluded. The pre-run campaign reservation records one driver campaign, zero browser campaigns and the exact source hash/commit. No retry occurred.

Installed tooling health is not proof of Antigravity integration, supported browser control/capture, R101 UI acceptance, or AI-governance behavior. Browser installation remains separately unadmitted pending Root's review of this plan.

Root normalized review-role header metadata and Markdown whitespace for repository checks; raw command outputs/receipts unchanged.
