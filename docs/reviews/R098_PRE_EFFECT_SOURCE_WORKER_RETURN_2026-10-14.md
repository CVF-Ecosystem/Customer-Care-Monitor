# R098 pre-effect source worker return

Status: SOURCE_READY_PENDING_INDEPENDENT_ADMISSION
Risk: R2
Worker: /root/m2_worker
Reviewer: Codex /root
Role: /root/m2_worker IMPLEMENTATION_WORKER
Date: 2026-10-14

Only the new R098 Go harness, Python supervisor and Python offline tests were authored. The Go harness calls unchanged `r096IsolatedTransport()` directly; its compiled dependency remains SHA-256 `3d2340131829f7afcb54d81be7d903c29fc8b36f160d065a05cd338a5df50dad`. The twelve R094 product hashes match the recorded frozen identity. No old R095/R096/R097 source, seed, approval, receipt, marker or budget was edited.

The new Root-owned parent manifest is at `.cvf/runtime/owner_alibaba_r098_new2_20261014/owner_cap.json`, SHA-256 `875a7bb93a9ab4330806d1df6f0b3a62343d52fa20b34aa96a09913671766106`. Both Go and Python admission require `freshOwnerGrantId=CCMAI-RUNTIME-098-OWNER-NEW2-20261014`, `freshGrantCap=2`, `historicalR096Calls=2`, exact seed commit, model, region, endpoint and role slots. R095's known-zero receipt is checked only as historical integrity, not as fresh capacity. The Root approval must additionally pin the new three source hashes, frozen R094 hashes, unchanged R096 TLS dependency hash, compiled binary hash and parent manifest hash before any credential access or fixture operation.

The recorder consumes a fresh exclusive physical slot before its one HTTP/1.1 RoundTrip. On a transport error it records only a bounded diagnostic class (`TIMEOUT`, `CERTIFICATE`, `DNS`, `NETWORK_OPERATION`, `EOF`, `HTTP_PROTOCOL` or `UNKNOWN`) and timeout/URL-wrapper booleans; it never records the raw error, URL, authorization header or key. A worker attempt with no HTTP response hard-stops reviewer admission in both Go and Python. The existing 758 requested-token, 768 observed-output ceiling, 6000-byte request, 1 MiB response, 60-second request, 300-second campaign and no-retry/redirect/fallback bounds remain. The copied current `providers.alibaba_maas` parser uses owner-pinned model and treats local presets as observation only.

The single cached, loopback fixture retains one-time intent, pinned container/image/network/volume and protected-container checks. Cleanup can verify an exited owned container from exact `HostConfig.PortBindings` and network mode while requiring active bindings for a running fixture; it does not restart or recreate a fixture. The offline test covers this stopped-container metadata path. Root owns fixture creation and cleanup review after source admission.

Source SHA-256 pending independent review: Go `0fbd2590dc2425ec254196b04b839cf476020189d0869ef70e8cd54602aa0536`; Python supervisor `1944353ba5586b01e4e6a2c680f626e482b358e63675df7f7a800fe4c0d886e8`; Python tests `49466312c3fdd9a5f12e2fea90151fc9747a4f0607a9765977097d54e34b871f`. `python -B -m unittest scripts.tests.test_r098_intervention_live_fixture -v` passed **16/16**; `python -B scripts/r098_intervention_live_fixture.py offline` passed frozen R094 and TLS dependency identity; `git diff --check` passed. These are tooling/source checks, not live AI governance proof. R098 Go use is **0/3 worker, 0/1 reviewer**, physical calls **0/2**, fixture **0/1**. No credential read, Docker effect, binary compile/launch or provider request occurred.

Next: Root independently audits this exact source and parent schema, then grants or rejects the first compile. Only after exact binary/source admission may precheck, setup and paired live dispatch occur. App HTTP200 alone will not count as proposal acceptance; full M2/M3/pilot remain open.
