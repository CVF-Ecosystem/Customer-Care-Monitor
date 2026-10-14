# R097 independent local TLS review

Status: REVIEWER_ACCEPTED
Risk: R1
Worker: /root/m2_worker
Reviewer: Codex /root
Date: 2026-10-14
Material commit: 510856d05108f47dad3eca944ccd52bb277d0f27

Root transitions to independent REVIEWER after worker return; canonical manifest/policy/bootstrap/state/memory/handoff/implementation/index re-read, doctor PASS27/27 with historical manifest-pin warning retained. Immutable seed83c29ce precedes activation092048c and the only worker Go invocation. Root confirms seed author/timing and same pinned scope; worker made no commits or source edits.

Accepted only pure local TLS tooling: stdout contains exactly one RUN/PASS for TestR096TransportTLSOffline, no SKIP, exit0; compiler WORK directory retained. Root independently hashes the copied binary f8feff858ce71ddc5c97b75ae7b15b4db8c25320543627ae8047ab646fefd99a (40,401,408 bytes), without launching it, and verifies actual Go source SHA3d2340131829f7afcb54d81be7d903c29fc8b36f160d065a05cd338a5df50dad. Source checks use the production-scoped transport constructor against an HTTP2-offering loopback TLS server, trust only its test certificate, preserve certificate verification, require HTTP1.1/status200 and reject h2. Independent Root before/after local probe inherited from R096 reproduces ALPN defect before and completes HTTP1.1/200 after. No mock AI decision or live governance claim.

Frozen R09412 source hashes exact; historical R0953 harness files have no delta from6b7c9b1; original R096 unknown real outcomes and failed overstrict assertion remain retained. R097 Go1/1, Root0Go, provider0/secret0/DB0/fixture0. Existing R096 owned fixture verified-clean and three protected container IDs unchanged. Old R0932/2, R095 consumed pre-provider claim and R096 parent2/2/Go7/7 never reset. No installed activation, push, merge or deployment.

Review finds no open defect in the bounded local regression. R097 may FREEZE LOCAL_TOOLING_ONLY after gates/continuity commit. R096 live remains CHANGES_REQUIRED/FREEZE OPEN: providerHTTP0, no response, model/usage/billing UNKNOWN, proposals0. Matching remote ALPN cause is inference, not proved. Root owns prior transport admission miss and activation continuity draft misses. Passing local test does not establish actual Alibaba response, AI output validation, full M2/M3, pilot, release, hosted CI or universal CVF behavior.

Return-time closeability recheck: bounded R097 proof CLOSEABLE; all required logs/source/binary identities exist, named test PASS and scope has no external effects. No worker redispatch/repair/replay allowed after Go1/1. Full live M2/owner budget and M3 are outside this order, not hidden completion dependencies. Product next move is concrete R098 corrected-source fresh max2 live approval packet; no automatic new seed/effect until owner grants fresh budget.

Static checks observed for proof material: legacy local and origin/main range7/7 PASS;89 gate unit tests PASS; explicit worker-return/precommit mandatory6/6 PASS. Profile closeability NOT_APPLICABLE_WITH_REASON for inactive COMPLETE_PENDING_REVIEW is explicitly unchecked; pre-dispatch ACTIVE contract was checked5/5. Final reviewer/precommit/range checks follow against this review. These checks prove repository state only, not hosted CI or runtime AI governance.

References: docs/reviews/probes/r097_local_tls/worker_proof.json; root_dependency_audit.json in that folder; docs/reviews/probes/r096_root/tls_repro_stdout.json; cleanup_receipt.json in that folder; docs/reviews/R097_LOCAL_TLS_WORKER_RETURN_2026-10-14.md.
