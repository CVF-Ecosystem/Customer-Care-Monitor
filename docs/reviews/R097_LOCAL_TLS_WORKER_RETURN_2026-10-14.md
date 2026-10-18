# R097 local TLS proof worker return

Status: COMPLETE_PENDING_REVIEW
Risk: R1
Worker: /root/m2_worker
Reviewer: Codex /root
Role: /root/m2_worker IMPLEMENTATION_WORKER
Date: 2026-10-14

The one authorized offline Go invocation passed. `TestR096TransportTLSOffline` executed, was not skipped, and returned PASS with process exit 0. The corrected R096 source SHA-256 `3d2340131829f7afcb54d81be7d903c29fc8b36f160d065a05cd338a5df50dad` matched the immutable R097 seed before and after the invocation. The test uses a trusted loopback TLS server that offers HTTP/2 and checks an HTTP/1.1 response; this is local protocol behavior only.

Exact command from the project root, with `GOPROXY=off` and `GOSUMDB=off`: `go -C backend test -work -run '^TestR096TransportTLSOffline$' -count=1 -v ./api/handlers`. Worker Go use is **1/1**. Raw stdout and stderr are [retained in the local proof folder](probes/r097_local_tls/worker_proof.json), alongside the structured receipt. Go reported `WORK=C:\Users\tiennm\AppData\Local\Temp\go-build940954240`; the built `b001\handlers.test.exe` was copied without execution to ignored `.cvf/runtime/r097_local_tls/handlers.test.exe`. Original and copy both hash to `f8feff858ce71ddc5c97b75ae7b15b4db8c25320543627ae8047ab646fefd99a` (40,401,408 bytes).

Root's independent static dependency audit is [recorded separately](probes/r097_local_tls/root_dependency_audit.json): frozen R094 product source and R095 harness identities remained exact before this proof. I made no source edit, provider call, credential read, database or fixture operation, download, binary launch, commit, push or deployment. R096's two consumed provider slots and 7/7 Go invocations remain exhausted. The actual Alibaba transport failure cause and live M2 outcome remain unknown; this local test does not prove AI governance or product acceptance. Independent Root review and bounded disposition are next.
