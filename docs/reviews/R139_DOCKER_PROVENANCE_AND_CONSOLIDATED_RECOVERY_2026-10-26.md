# R139 Docker provenance and one consolidated recovery decision

Status: DRAFT
Risk: R2
Worker: Root forensic metadata analyst and specification author
Reviewer: Independent Sol Medium `/root/r139_toolchain_adoption_review` (independent decision-only review recorded; operational authority pending)
Date: 2026-10-26

## Findings established without running Docker

The local installer log records Docker Desktop4.97.0/build244584, previous installation4.96.3/build244472 at2026-10-26T12:30:01Z, the exact Docker.exe staging/patch target at12:31:19Z and application start12:33:04Z. Current Docker.exe mtime12:31:20.801173Z fits that transaction and predates original controller2560 at12:34:49Z. Installed file metadata reports Desktop4.97.0.244584 and CLI29.9.2. This strongly links the proximate file replacement to the installation transaction. It does not identify who initiated it, prove automatic update, or establish the payload download source. Current and prior exact staging/temp paths named in the log are absent; no old approved copy was found in those locations or the prior narrow bin listing. No whole-machine search was made.

Current full-file SHA256 is17cbf2f5f5f7693bbeda46e35dc930050b40aff06f41f9f5a69e4e2dd7fc12aa, size44,922,800. Original approved pin remainsb8f907c73a89c9bd0a88bddb0cb5008728af4dea0d773b3f3614e9f203737e04. The old accepted-build record is unchanged. Its 13 source pins, accepted binary, Vite and the other4 tools remain the inherited19 matching files; no source build/test is rerun and no new pin is adopted by this report.

Root's static PE/PKCS7/X509 analysis computed the image digest excluding the PE checksum, security directory entry and certificate table, and matched the embedded Spc image digest232ee700d1f179f467acaa05df25790dfe139e9c4068184500c6e67cdb695b53. Signed-attribute content digest and signer RSA signature verify mathematically. The signer certificate is Docker Inc, has digital-signature/code-signing usage and current validity; its issuer signature matches embedded DigiCert intermediate, which verifies against DigiCert Trusted Root G4 present in the local Windows ROOT store with code-signing trust purpose. These facts are preserved in offline_signature_analysis.json. The Python parser used its documented BER fallback for the PKCS7 content and emitted a warning, retained in the record.

This is qualified offline cryptographic evidence. No online revocation, Microsoft WinVerifyTrust publisher policy, payload acquisition provenance, actual CLI/daemon compatibility or equality to the old full-file pin was verified. The old file is unavailable for a byte diff, so the precise changed byte region and old-version-to-old-hash mapping remain unknown. Current signature validity does not authorize a different accepted pin. No unknown Docker executable, installer, daemon, browser or provider was run; no binary was copied, repaired or substituted. In-memory zero-field/checksum candidates did not match the original hash and were never written.

## Concrete same-order proposal for Owner decision

Adopt the exact current17cb hash only under explicit Owner authority and independent review, with the above offline verification limits accepted. After approval, copy the CLI bytes to the fresh project-owned `.cvf/runtime/r139_docker_tool_snapshot_20261026/docker.exe`; verify source and copy size/hash before use, preserve the installed file, and issue a distinct reviewed accepted-build record. The snapshot directory is absent now. This is a campaign-owned pinned copy to separate the campaign target from the Docker Desktop installation target; it is not a claim of tamper-proof storage. No old accepted pin or failure receipt is edited, and no software installation/upgrade/daemon restart is proposed.

The proposed accepted-build JSON inherits source7ef30823746d72e935607c0bebf6ab306099eb48 and all source/check/Go/Vite evidence fromad348196. Only Docker tool path/hash is proposed to change; rootReviewApproved:false and proposalStatus prevent treating this as acceptance. The full static launch guard source is stored as specification data and is not executed or integrated. Its contract checks all5 tools, binary, Vite and13 source pins, plus existing owner/ref/canonical/exclusive-path checks BEFORE any extraction, counted Popen, OS query or clock. A mismatch must HOLD without consuming a new original-controller invocation; the already spent two attempts are never refunded. Caller path/schema/reparse validation and original source-controller checks remain required.

One worker focused pure metadata guard-check process and one independent Root focused guard-check process are proposed. Retain all16 prior counted checks; do not rerun old suites, build Go or test Go. These are new planned checks for a new guard, not spent/accepted evidence. Actual integration, current actual file readback, independent consolidated review, repository gates and normal clean commit are required before any newly admitted runtime.

Request one additional original preclock invocation: cumulative controller cap2→3, retain used2, no retries of old paths or attempts. The one campaign remains used0/cap1,1200seconds, cleanup+1020/setup180; exact service queries5, CUA64 stages8/4/48/4, inventories3, one owned tab, provider1≤USD1 remain unchanged. The exact five admitted queries provide the future bounded tool-service compatibility observation after actual reservation/signal; no extra uncounted CLI/version/daemon probe is proposed. Actual G0 needs Root independent evidence review and a separate Runtime Effect Admission before fixture/key/provider. No Antigravity, helper tranche/order, Core/profile update, push or deployment is proposed; M3-05/M3-08 remain the target of the same R139.

## Current disposition

No adoption or third invocation is released. Current Root original-controller allowance stays2/2 spent, campaign0/1, service/CUA/fixture/key/provider0; Product B remains BLOCKED_PENDING_OPERATIONAL_ADMISSION, Facebook/Zalo OA parked. Quota-validated review9baa and16 checks remain reusable at matching bindings. Owner is reviewing one concrete tool-adoption/snapshot/guard/verification/preclock package, not a chain of per-step prompts.

Evidence and proposal: `docs/reviews/probes/r139_integrated_20261021/native_operational_20261024/docker_provenance_20261026/`. This metadata/provenance decision is not live AI-governance evidence or a runtime PASS.
