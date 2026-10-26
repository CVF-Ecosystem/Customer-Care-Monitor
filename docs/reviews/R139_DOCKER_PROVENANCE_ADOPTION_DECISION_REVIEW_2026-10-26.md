# R139 Docker provenance and adoption decision — independent review

Status: REVIEW_PASS
Risk: R2
Worker: Root forensic metadata analyst and specification author
Reviewer: Independent Sol Medium `/root/r139_toolchain_adoption_review`
Date: 2026-10-26
Scope: DECISION_PACKET_ONLY_QUALIFIED; no tool adoption, source acceptance, controller release, or runtime acceptance

## Disposition

I accept `docs/reviews/R139_DOCKER_PROVENANCE_AND_CONSOLIDATED_RECOVERY_2026-10-26.md` as one reviewable, bounded Owner decision for the existing CCMAI-RUNTIME-139 / M3-05 / M3-08 order. The packet asks for exact-current-Docker-byte adoption, a fresh campaign-owned CLI snapshot, a distinct accepted-build record, a full static prelaunch guard with two focused offline check processes, and one additional original controller allowance. This review passes the **proposal and claim boundaries only**. The proposal is not itself Owner approval or an operational admission.

## Independent checks

- Rehydrated manifest, policy, compact and canonical state, session memory, active handoff, implementation status, and index. Current mode/phase is REVIEW; compact and canonical next move and parked checkpoint agree. The inherited Core execution pin remains `e9ddcbcdcac2a9df520f5eb0761dbdb609b9deba` with profile1.1. The separately recorded Owner stable-pin direction forbids treating sibling Core freshness as automatic campaign adoption.
- Read the original root-observed receipts: controller PID23500 returned exit1 `COORD_OWNER`; controller PID2560 returned exit1 `TOOL_PIN` before source child, OS identity query, reservation, or clock. Used original controllers remain 2/2, campaign 0/1, service/CUA/fixture/key/provider zero. The second receipt's correction preserves the original invocation; the old private readback's contradictory `newControllerInvoked:false` is not used for accounting.
- Independently recomputed the installed Docker CLI whole-file SHA256 as `17cbf2f5f5f7693bbeda46e35dc930050b40aff06f41f9f5a69e4e2dd7fc12aa` and the installer-log SHA256 as `de3b602caeccb4fabd956169e1054343f32bd94f7c307d1879212b43cfa2cfe4`. The recorded installer target/timing and file mtime support an installation transaction as the proximate replacement event. The actor, automatic-update status, and download source remain unknown.
- Recomputed the four SHA256 links in `consolidated_adoption_decision.json`; all match their referenced JSON files. Reviewed the offline PE/PKCS7/X509 record: image/content digests, RSA signature, Docker Inc code-signing leaf, DigiCert intermediate and locally trusted G4 root are reported as verified, with the parser BER fallback warning retained. I did not rerun the cryptographic parser or claim Microsoft publisher-policy approval. Revocation, WinVerifyTrust, acquisition provenance, Docker execution, and daemon compatibility remain unverified.
- Compared proposed accepted-build pins with the existing `CVF_SESSION/authority/CCMAI-RUNTIME-139-V3-ACTIVATION-ENV-ACCEPTED-BUILD-PINS-ADMISSION.json`: source commit, all 13 source pins, backend/frontend trees, binary, Vite, and the four other tool records are unchanged. Only Docker path/hash changes in the proposal; `rootReviewApproved:false` and `proposalStatus:DRAFT_REQUIRES_EXPLICIT_TOOL_ADOPTION_AND_INDEPENDENT_REVIEW` prevent silent adoption. The proposed snapshot path does not yet exist.
- Inspected the guard as **source code stored in JSON data**. Its proposed audit counts 5 tools + binary + Vite + 13 source files = 20 and fails closed on missing, invalid, size or digest mismatches. Caller schema, alias/reparse validation, actual integration before Popen, and independent focused verification are explicit future conditions. No guard source was imported, executed, tested, or integrated in this review.

## Boundary and required next move

The decision is suitable for one explicit Owner response on the consolidated package. Until that response and a subsequent separately reviewed implementation/admission, the old approved Docker pin remains authoritative; no file copy, new pin, focused guard-check process, third controller, Docker/daemon query, campaign reservation, browser, fixture, key, or provider effect is released. The proposed cumulative controller ceiling is 3 with 2 already used; the currently admitted ceiling stays 2. The 16 spent source checks and prior review `9baaed702e485362dae018954f2307e9d69edf30891d903f4edeb52c5a573110` remain historical evidence, not fresh guard proof. Product B remains `BLOCKED_PENDING_OPERATIONAL_ADMISSION`; Facebook and Zalo OA remain parked.

The proposed guard must be integrated and tested against the actual adopted build record, and the complete 20-file readback must occur before any future counted controller Popen. A later successful guard or controller cannot by itself prove live CVF governance, G0–G3, product acceptance, or provider behavior. No real provider API call occurred in this decision review, so it makes no runtime AI-governance claim.
