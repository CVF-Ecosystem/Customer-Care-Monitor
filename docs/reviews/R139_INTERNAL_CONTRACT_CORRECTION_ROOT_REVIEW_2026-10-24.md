# R139 internal contract correction — independent source review

Status: ACCEPTED_BOUNDED_SOURCE_AND_SOFTWARE_CHECKS
Risk: R2
Worker: Internal Sol Medium `/root/r139_internal_contract_repair`
Reviewer: Root independent source REVIEWER, assisted by Luna Max read-only audit
Commit steward: Root
Product B: BLOCKED_PENDING_OPERATIONAL_ADMISSION

C1–C4 source correction is accepted at the exact hashes below. Python37/37 PASS, offline Go compile PASS, exact pure Go9/9 PASS with0FAIL/0SKIP; all original process handles were retained and wait/exit observed. This resolves the source predicate/evidence/production-path test gaps from the prior repair review. It does not establish native browser capability, runtime cleanup, real provider/AI governance, G0 aggregate, G1–G3/E05/E06/full M3, hosted CI, release or deployment.

| Check | Existing cap used | Result | Seconds |
| --- | --- | --- | --- |
| Root focused Python | 1/1 | 37PASS,0FAIL/0SKIP, exit0 | 3.869 |
| Offline Go compile | 1/1 | exit0, original wait observed, stdout/stderr empty | 179.418 |
| Root exact pure Go | 1/1 | 9PASS,0FAIL/0SKIP, exit0 | 5.456 |

Compile completed within the admitted180seconds at179.418seconds; no timeout or retry. New binary SHA256 `a1027ac676eb68527c636781432a1cd22129d70f1e6ac968f0843e87bc1df825`, size59310592bytes; binary/cache stay ignored. The exact pure selector and input/init audit are in `root_internal_binary_audit.json`. All three checks used the pinned tools, exact admitted argv, scrubbed allowlist environment and exclusive reservations; source hashes were unchanged after every check. Backend ignored/untracked input sets were empty and reparse inputs absent before compile. Slots are consumed, not renewed. Worker executed zero source checks and made no commit; this is based on submitted tool/evidence history, not system-wide absence telemetry.

## Source and acceptance

| Path | Actual reviewed SHA256 |
| --- | --- |
| `scripts/r139_execution_supervisor.py` | `160ffd51ac6d82ae906b9070f16b06f0f703eed31d2360aab14577bbf4c79de1` |
| `scripts/tests/test_r139_execution_supervisor.py` | `e28546a7cc8ebde7c4d9482097e015662b1b87eb83d0dedf775914360764a9f9` |
| `backend/api/handlers/intervention_execution_browser_test.go` | `25085f86bcce25e3bbf1cf8841c8c5e72284aec0f5adf0675038f81d435852b6` |

C1 now requires bounded native contract/tranche/campaign/page/session/operator identities and the admission-selected close mode across records, exact booleans/counts, and a named human fallback without application-role substitution. Go `UseNumber`/`Int64` rejects fractional/scientific zero tokens, matching Python exact integer checks; legacy IAB retained. C2 requires readable hashed JSON evidence, project-confined admission-pinned reservation receipt, raw creation/control/capture under distinct ignored readiness root, future action/close/inventory paths under the exact product root, prelaunch absence and terminal presence with actual postcreation digest collection. C3 checks complete canonical inventory/raw equivalence, observed page absence/zero owned count, chronology through deadline/current time and captured sequence through close. Hashes alone never authenticate native/Owner provenance or all future nonreuse.

C4 runs the actual Python `module.run` and real terminal verifier in success/ordinary-error/missing/malformed close scenarios with synthetic temporary files, fake original children and owned cleanup. External Docker/launch/listener/HTTP/key/source-admission/route boundaries are stubbed. The readiness stub emits an artificial `apiOrigin: h` because its mock side_effect iterates a string; these tests assert finalization/close behavior, not valid URL readiness or actual browser service. This limitation is retained; no additional repair/check or URL/browser claim is inferred. The two added Go cases use temp files; the seven inherited pure cases/init were audited. Go terminal helper remains software-test-only, while Python run is the operational close enforcement point.

Static review caught and corrected Go fractional zero acceptance, unsafe untyped timestamp equality, broad project-path mock and missing inert test admission fields BEFORE the single check chain. No failed source check was erased or retried. Original earlier worker Python38PASS/1FAIL and original G0 HOLD remain historical.

## Handback timing and evidence authority

Owner explicitly reassigned source work to internal subagent; Antigravity is browser-only. Actor supplement committed at `8f6063b12cf36c6391e29d43b0fd684c3a9c61e4`; source and protected pins were verified before BUILD and worker acknowledged before edits. Root and Luna remain independent from source implementation.

Root sent a final C2 compatibility note before the worker's frozen return arrived; worker processed it just after sealing its return and before Root's stop arrived. Root owns this crossed-message timing. It did not authorize an automatic new round or new check/effect cap. The final project-confined receipt variant and positive cases were independently reviewed by Root/Luna and included in ALL actual check hashes above. Original `R139_INTERNAL_CONTRACT_CORRECTION_RETURN_2026-10-23.md` SHA256 `b93b83fed4205b533717c73415d8b4456e3e49a58aa5dbfb66be7dadb55d23d7` is preserved byte-for-byte in `root_internal_original_worker_return_bytes.json`: its earlier hashes/embedded diff are stale, not authoritative for the checked source. The working report has only two Root-added exact role header lines to satisfy the repository gate; original content is retained in that snapshot. Its word tracked describes an allowed receipt location, not proof that any actual future receipt is tracked. Actual source authority is this review plus Root original Git diff bytes/base64/digest in `root_internal_source_diff.json`; no worker report rewrite requested. This qualification prevents treating handback freeze timing as compliant evidence.

One existing C1–C4 correction completed; no helper tranche/order or reset. Root retains source acceptance; aggregate R139 stays REVIEW/CHANGES_REQUIRED and FREEZE open because operational proof remains open. Standing `OWNER-ROADMAP-SINGLE-SCOPE-20261023` remains mandatory: follow same M3-05/M3-08 G0→G1→G2→G3 order, internal agents handle non-browser work, Antigravity only the required browser segment. Next Root consolidates actual operational pins/ownership/current browser evidence and a committed same-order runtime supplement under the existing conditional campaign/API1/USD1 grant. No new provider/browser/process/DB/Docker/key effect was activated by these software checks. Old spent/uncertain slots, old-tab/protected/provider UNKNOWN and Facebook/ZaloOA parked remain.

Receipts: `docs/reviews/probes/r139_integrated_20261021/antigravity_integrated_source_20261023/root_internal_*` contain reservations, original stdout/stderr, result timestamps/waits/exits/tool/source/binary/environment pins, static and binary audits and original actual source diff. These are bounded local software evidence only.

Repository gate finding retained: the first pinned worker-return and reviewer-fast phases failed ROLE_FIELD_INVALID because the worker report lacked exact Worker/Reviewer header lines. Root preserved original bytes/hash/base64 before normalizing those two metadata lines. No source/test/budget change or worker redispatch. Fresh gates are run after this fix; no claim that the initial gates passed.

Final repository verification: legacy preflight7/7 PASS; governance89unitPASS44.680s; pinned worker-return6/reviewer-fast7/pre-commit6 controls PASS after the documented role-header metadata correction. APPL remains NOT_APPLICABLE_WITH_REASON because aggregate work order is BLOCKED_WITH_REASON: active operational closeability was NOT checked. Root/Luna independent source reviews and the actual software receipts support only the bounded source acceptance. No hosted CI/runtime-governance claim. Source and pinned correction contract section remain byte/hash unchanged after checks.

Exact original stdout/stderr bytes for all three checks are additionally preserved with SHA256/base64 in `root_internal_original_check_output_bytes.json`, so Git text newline normalization cannot erase raw output provenance. Full staged diff-check flags literal unified-diff context whitespace only in the preserved worker return; scoped staged diff-check excluding that evidence artifact passes all source and other files. The artifact remains intact with original byte snapshot; no global formatting-pass claim.
