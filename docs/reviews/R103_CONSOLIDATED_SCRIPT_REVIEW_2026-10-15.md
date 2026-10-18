# R103 consolidated pre-runtime script review

Status: BLOCKED_WITH_REASON
Risk: R2
Worker: /root/r103_tooling_worker
Reviewer: Codex /root
Disposition: CHANGES_REQUIRED_SOURCE_ONLY

Root reviewed frozen draft c65550a38042472c34a46346b104906c48019f77f68b17290542df2cf5da2edd. No download/cache effect admitted yet. Sol read-only adviser corroborated concerns; Root owns disposition. Initial [NODE] REPL bug was caught and fixed before return, no failed runtime. Required consolidated repair1:

1. Actual CLI --version output is `Version 1.57.0`, not `1.57.0`; require that exact tagged output and keep Node --version. Use stdin DEVNULL in subprocess probes.
2. Reject Windows colon/ADS/device/reserved names and trailing dots/spaces in all tar components, alongside existing traversal/links/collision checks. Verify local targets resolve to exact authorized paths and are not junctions/symlinks.
3. Verify installed node hash after copy and record per-file installed package/Node manifest + preexisting browser top-level inventory (do not claim full historical file-byte preservation from names alone). Record each npm request intent/result/redirect with bounded bytes before later publish/version failure could hide network evidence.
4. Pin per-command browser cache path and skip-browser-GC; clear only per-process download-host overrides that could redirect tooling to unauthorized sources. Browser mode must freshly verify exact package name/version + CLI/node against the installed manifest. Root admits install-driver ONLY after repaired source commit; install-browser is a distinct later admission after Root sees pinned package browser plan. Current post-hoc DEBUG URL parsing does not enforce redirect/transfer/time totals, and cannot be described as doing so. Keep browser mode inert/unadmitted for now; do not build a general proxy/framework.

Preserve scope/seed/old caches/evidence, no source/dependency/global-setting changes. Minimal consolidated source repair and return/hash, no runtime until Root re-admission.
