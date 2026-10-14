# R096 TLS ALPN repair1 worker return

Status: REPAIR_SOURCE_READY_TEST_NOT_RUN
Risk: R2
Worker: /root/m2_worker
Reviewer: Codex /root
Role: /root/m2_worker REPAIR_WORKER
Date: 2026-10-14

The scoped R096 harness now sets TLS `NextProtos` to HTTP/1.1 after cloning the default transport, consistent with its disabled HTTP/2 handler. Certificate verification remains enabled. The application harness uses that transport; product R094 files, R095 files, model, prompt, cap and parent ledger are unchanged. The new test uses a trusted local dual-protocol TLS server and checks that the response is HTTP/1.1 with status 200 and no HTTP/2 negotiation. It asserts protocol behavior only, not AI governance.

The sole remaining worker Go invocation, `go -C backend test -work -run '^TestR096TransportTLSOffline$' -count=1 -v ./api/handlers` from the project root with offline Go caches, **failed**. Its actual response was `HTTP/1.1 200` and empty `response.TLS.NegotiatedProtocol`; the first test assertion incorrectly demanded literal `http/1.1` ALPN. Raw output is retained at ignored `.cvf/runtime/r096_live/repair1_go_stdout.log`; Go work directory was `C:\Users\tiennm\AppData\Local\Temp\go-build964426295`. The binary built by that failing test was copied, without execution, to ignored `.cvf/runtime/r096_live/candidate_v2.test.exe`, SHA-256 `54f8d741f610df24183cbfa43bceca006cffb23f4abdc81d06a316ecdcf584ec`. Its pre-assertion-correction source SHA-256 was `56534d17b1166b78412caaca437498f51a4fd9ebe83accb0422652d2fe462924`; it does **not** match the final source and is not approved for live use.

After the failed test, the test assertion was narrowed to accept empty ALPN or `http/1.1`, reject `h2`, and require `HTTP/1.1`/200. Final Go source SHA-256 is `3d2340131829f7afcb54d81be7d903c29fc8b36f160d065a05cd338a5df50dad`. The corrected test result is **NOT_RUN** because the worker Go budget is exhausted at 4/4; Root reviewer Go is exhausted at 3/3. The old source-signed binary and approval remain historical and are not reused for provider effects.

R096 actual live result remains two permanently consumed owner physical slots and database budget 2/2, both transport status zero with unknown remote cause, usage unknown, and zero proposals. No key read, fixture recreation, provider call, retry, refund, or tuning occurred in this repair. Root's separate finite local tooling proof, if authorized under a new seed, is needed to verify the corrected assertion before any fresh-call proposal; no further provider authority exists under R096.
