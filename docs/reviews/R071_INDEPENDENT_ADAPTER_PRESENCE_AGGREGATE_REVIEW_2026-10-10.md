# R071 independent adapter-presence aggregate review

Status: REVIEW_PASS for the bounded local APO01..12 contract. Reviewer: Codex /root, independent from Luna `/root/r071_worker`. Accepted source `ee6b1c0b465c025c5d536c5cd4261336e26a5071`. Initial source86e6ff9/plan0402ade and consolidated pre-Go S1/S2/S3 findings remain preserved. One NEW-test repair, no product-semantic repair. Static approval1116141 preceded all actual Go commands.

Each role ran one offline wholebackend compile and four selected pure tests: positive20top/65outcomes PASS, M01 intended detector FAIL with known-zero healthy PASS, M02 intended malformed-count detector FAIL with valid-int healthy PASS, restored10top/38outcomes PASS. The expected mutation exits1 are semantic kills, not unexpected campaign failures. No skips, panic, timeout or retries; worker5/8 and reviewer5/8 attempts, ten actual Go processes total. Native streams, reservations, full manifests and mutation diffs are retained in four `r071_runtime_*_first` packets. [Independent raw audit](probes/r071_independent_raw_evidence_audit_2026-10-10.json) recomputes actual results and all physical hashes/backups.

| Contract | Independent finding/evidence |
|---|---|
| APO01 | Legacy UO fields/arithmetic unchanged; six old UO/four EX tests byte-identical and PASS, omission/600byte legacy envelope checked. |
| APO02 | Active-success once-only guard independent of old scalar hook; nil/no-active/inflight/error/duplicate covered; nil metadata unobserved despite old scalar values. |
| APO03 | Fixed schema/source allowlist and Gemini-unavailable contradiction handling; raw source/diagnostic not retained, normalization/privacy tests PASS. |
| APO04 | Known requires nonnil nonnegative int64; other statuses disallow values; siblings independent, input Complete ignored; null and malformed-output controls PASS. |
| APO05 | Known0 and integers above2^53 through MaxInt64 exact in Go; healthy controls PASS; no scalar fallback for absent metadata. Browser representation untested. |
| APO06 | Fixed saturation and independent side sum overflow; known-prefix totals withheld across unresolved/error/unhooked success; MaxInt64+0 remains valid, +1 overflows. |
| APO07 | Input numeric snapshot and independent freeze pointer proven; full maximum numeric-width layout <=1200bytes; no raw/provider/model/content fields. Artificial maximum layout fixture proves size only. |
| APO08 | Aggregate outside Calls survives205calls and actual3000byte trimming; legacy1500 tests unchanged/PASS, normal128KiB source cap preserved. No promise below irreducible fixed envelope. |
| APO09 | Exactly two additive Analyzer successful-return hooks next to old observeUsage; no pricing/admission/provider/logging/writes/results semantic change. Wiring checked statically, no Analyzer persistence end-to-end test. |
| APO10 | Each offline wholebackend compile PASS; exact20 positives, both intended mutations with healthy controls, restoredNEW10; no DB/broad-prefix tests selected. |
| APO11 | Immutable seed precedes BUILD, original and resumed/repair acknowledgments retained, first source precedes compiler, own root plan independently selected. Actual220-member archive, source-only diffs/restoration/second archive, four separately verified raw backups. Parent never edits product/tests. |
| APO12 | Five reserved attempts/five processes per role; caps8(2build+6tests) unchanged, reserve unused; zero automatic retries. Ledger preserves first source/findings/admin defects and assistance. |

Raw audit verifies58 packet files and eight physical before/after tar copies across four backup inventories (66 entries), exact approved archive SHA a9c134179ffc8a96f98107474e155d7d28430c73a8320f2561d4e1e76e5dfd26, all220 source files restored in each owned scratch. All407 protected physical files outside three allowed existing paths unchanged, old tests/authority seeds intact; R071 seed matches first committed bytes at da8d34a. Ten owned Go parent PIDs absent in a later read-only snapshot. TEMP scratch/backups retained, not deleted. These are bounded file/process observations, not complete IO/network/write history.

Worker authored product/tests and source commits; root supplies SPEC, consolidated static findings, shared capture infrastructure, independent plan/audit and metadata. One Git identity cannot machine-prove authorship; reviewer confirms observed dispatch and timing. Gate passes concern repository state, not universal agent control.

NOT RUN: race (CGO0), full backend runtime/DB tests, app frontend/visual, real Analyzer persistence/provider/wire/billing, live/network/GitHub Actions. No real provider call, no CVF runtime governance or full S2/S3/S5/globalF02 closure claim. Queue/real Analyzer, consumer UI/storage, pricing history/settlement/permissions/policy-version remain separate OPEN. Facebook/Zalo OA parked; prior R068 Git publication remains unverified and not retried; R071 publication unauthorized.
