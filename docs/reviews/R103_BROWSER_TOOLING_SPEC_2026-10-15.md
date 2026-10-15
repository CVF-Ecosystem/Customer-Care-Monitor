# R103 browser tooling recovery — intended contract

Status: DRAFT
Risk: R2
Worker: /root/r103_tooling_worker
Reviewer: Codex /root

Owner authorizes repair after questioning the old-CDN404 conclusion. INTAKE: Node24.19 exists, driver157 directory empty, browsercache contains newer chromium1228/headless1228 plus ffmpeg1011/winldd1007. DESIGN: reconstruct exact driver1.57.0 layout from official npm playwright-core tarball and existing Node, verified by npm SHA512 integrity and exact package version, preserve old cache. Public upstream v0.5700.0/run.go proves package/cli.js + node.exe and --version1.57.0 are the bootstrap check. Version1.57 exists; old ZIP availability is distinct.

SPEC T01: archive official HTTPS provenance/integrity/version and safe paths verified; T02: driver157 installed with unchanged existing cache/node source, CLI --version exact; T03: required browser revision inspected, only Chromium-related absent artifacts installed with no browser garbage collection; T04: local about:blank/control+capture health observed and evidence attribution explicit, no external webpage; T05: source/global config/old evidence unchanged and actual requests/files/processes captured. A working standalone driver screenshot does not prove Antigravity integrated tool restored; distinguish package health, native pipe and IAB screenshot capabilities.

Browser UI branch is conditional on supported control+capture and independent Root admission, with existing pinned R101 artifact/frontend and sole preserved fixture only. B01..B10 remain NOT_RUN until actual actions/captures. No policy bypass, original app/DB/provider/secret or new product source is allowed. Stop and diagnose command failure, no blind retry/old-CDN loop; retain raw error. Existing screenshot/native-pipe failure may remain separately blocked even after driver fix.

WORK_ORDER will reference immutable seed committed before BUILD. Worker stages source repair script/plan first; Root reviews before network/cache effects. Root independently verifies version/files and local capability after worker return. R2 independent worker/reviewer roles, Root owns commits/closure. Existing R101/R102 status/budgets remain inherited unchanged; tooling is a new owner-authorized necessary prerequisite, not a product milestone pass.

Primary references: https://github.com/microsoft/playwright/releases/tag/v1.57.0 ; https://github.com/google-antigravity/antigravity-cli/issues/566 ; https://github.com/playwright-community/playwright-go/blob/v0.5700.0/run.go ; https://github.com/microsoft/playwright-python/blob/main/scripts/build_driver.py .
