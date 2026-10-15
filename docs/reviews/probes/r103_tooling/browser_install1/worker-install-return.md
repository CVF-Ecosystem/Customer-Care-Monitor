# R103 browser installation return — single admitted campaign

Risk: R2
Worker: /root/r103_tooling_worker
Reviewer: Codex /root
Date: 2026-10-16
Tranche: CCMAI-RUNTIME-103
Role: IMPLEMENTATION_WORKER, under Root's `ADMIT_REQUIRED_CHROMIUM_INSTALL_ONLY` decision
Reviewer and commit steward: Codex `/root`

## Result

The single admitted command passed in 39.98 seconds with process exit code `0`:

```text
python -B docs/reviews/probes/r103_tooling/repair_driver.py install-browser
PASS; receipts: .cvf/runtime/r103_tooling/20261015T221339Z_install-browser_e1edf681
```

The committed script hash remained `c0fb1ee114db689e8cd71eb310534569de1514378c8f6fbf272a78ec40b2e15a`. The command revalidated the installed Playwright package and Node manifest, reported Node `v24.19.0`, CLI `Version 1.57.0`, set `PLAYWRIGHT_BROWSERS_PATH=C:\Users\tiennm\AppData\Local\ms-playwright` and `PLAYWRIGHT_SKIP_BROWSER_GC=1`, and cleared the per-process browser download-host overrides. It installed Chromium 143.0.7499.4 / revision 1200 and Chromium Headless Shell 143.0.7499.4 / revision 1200. CLI host-requirements validation passed for Chromium, ffmpeg, headless shell, and winldd.

## Download observations

The captured Playwright debug log records one attempt (`#1`) for each artifact, both HTTP response status 200, and no retry. It reports these exact transfers:

| Artifact | Observed URL | CLI-reported bytes |
| --- | --- | ---: |
| Chromium 1200 | `https://cdn.playwright.dev/dbazure/download/playwright/builds/chromium/1200/chromium-win64.zip` | 178,067,817 |
| Chromium Headless Shell 1200 | `https://cdn.playwright.dev/dbazure/download/playwright/builds/chromium/1200/chromium-headless-shell-win64.zip` | 112,400,590 |

Combined reported archive bytes: 290,468,407. The CLI said ffmpeg 1011 and winldd 1007 were already downloaded, so it did not download them. The CLI owns its HTTP stack: actual HTTP request count and redirect count are **unobserved**; the output shows the final response codes and bytes only. The script validates CLI-emitted URLs after the fact and does not enforce browser redirect hosts or an aggregate browser-transfer byte cap. No network-guard enforcement claim is made. The raw stdout/stderr retain the progress lines, debug attempts, URLs, status codes, sizes, and the non-fatal Node `[DEP0169]` deprecation warnings.

## Preservation evidence

Root's saved baseline at `docs/reviews/probes/r103_root/browser_cache_before_install.json` contained 610 files. A full post-install path/size/SHA-256 comparison found all 610 still present and unchanged: missing `0`, changed `0`. The install added 603 files totaling 669,761,836 expanded bytes; the only new top-level directories are `chromium-1200` and `chromium_headless_shell-1200`. Existing `.links`, `chromium-1228`, `chromium_headless_shell-1228`, `ffmpeg-1011`, and `winldd-1007` remain. No old cache entries were removed, no global settings changed, and no retry occurred.

## Portable evidence and limits

The JSON/TXT command receipts, Node/CLI version outputs, before/after cache inventories, pinned dry-run, install result, and raw installer stdout/stderr are copied under this directory. No executable, browser archive, npm metadata, or extracted package was copied. The campaign ledger records driver campaign `1`, browser campaign `1`, command exit `0`, two observed download attempts, and the unobserved HTTP request/redirect measures.

No page launch, screenshot/capture, Antigravity/Codex application interaction, app process action, UI test, fixture, server, provider, or commit occurred. Root's independent browser/cache verification and local blank-page health probe remain the next governed actions. This install alone does not demonstrate supported Antigravity integration, R101 UI acceptance, or AI-governance behavior.

Root normalized review-role header metadata and Markdown whitespace for repository checks; raw command outputs/receipts unchanged.
