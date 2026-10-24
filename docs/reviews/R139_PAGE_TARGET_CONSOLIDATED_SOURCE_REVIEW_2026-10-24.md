# R139 combined source and activation review

Status: ACCEPTED_BOUNDED_SOURCE_AND_SOFTWARE_RUNTIME_OPEN
Risk: R2
Worker: Internal source implementation/repair worker and internal activation metadata repair worker
Reviewer: Root independent source reviewer; internal Luna Max read-only metadata reviewer
Authority: CCMAI-RUNTIME-139-SOURCE-AND-CONDITIONAL-REENTRY-ADMISSION.json, SHA256 0ef818bfa4a60b9bd1b01335996989ca4c00d63f11141d9074dd3c26befb5b4f
Dispatch: 79e02f6c67b242eea4540ad4d2fdce5b42fda106

One existing R139/M3-05/M3-08 order. Source authoring is finishing its interrupted initial candidate; this review consolidates predictable metadata execution blockers before any source check. One admitted consolidated repair is released for the two metadata designs and their own return/evidence only. No runtime, browser, service, key, provider, new clock or commit is released. All six source-check slots remain unused at this review.

| Finding | Required correction |
| --- | --- |
| F01 BLOCKER: product-root lifecycle conflicts. Activation accepts relative paths; recorder demands a relative existing root. The source supervisor requires the exact absolute authorized root absent before launch and creates it itself. | Both designs must validate the exact absolute source-authorized root, project confinement and no reparse. Recorder permits only the fresh absent root before launching the admitted supervisor; it must not create the product root. Product stop is written only after the existing exact root and original run-intent/admission identity establish ownership. Missing or mismatched binding is HOLD, no blind stop. G0-only abort creates no product root. |
| F02 BLOCKER: accepted-build reference schema mismatch. Code reads `acceptedBuildPinsRef.ownerEnvelopeSha256`, absent from declared `{path,sha256}` schema. | Use the already validated accepted build record's `ownerEnvelopeSha256` binding, or explicitly align schema and consumers. Do not require an undocumented duplicate field. |
| F03 HIGH: incomplete cap mapping and a 10-second PowerShell metadata query against the approved 5-second bound. | Bind the complete immutable cap set, keeping baseline/final enumeration separate. Use at most five seconds for the OS query. Explicitly require product-only limits in the later actual Runtime Effect Admission; do not claim activation alone enforces product effects. |

Root independently corroborated F01 against `fresh_runtime_admission` and `run`; Luna independently reported all three findings with code locations. Existing bounded metadata child waits, retained original product Popen/wait, original +1020/+1200 clock and post-commit signal controls are retained. No hypothetical replacement kill, scope widening, authority rewrite or Owner reconfirmation is needed.

Source authoring questions were delivered before candidate completion: Python/Go parity; baseline/creation/ledger chronology; exact observed capture PageId/URL; separate native-task handoff; metadata-only final inventory; native authenticity remains an independent raw-transcript review, never proved by envelope hash strings. Initial authoring must settle these together with payload consistency. Final check outcomes and disposition will be appended after actual evidence exists.

## Consolidated source repair release

Independent Luna source review and Root review agree on the following additions to the same consolidated repair, before any source-check slot is launched:

- F04: Go reads the page-target prefix under the project root; Python confines it to the readiness root. Confine both new Go prefix reads to readiness, reject overlapping readiness/product roots in the new branch, and exercise a valid-hash prefix outside readiness plus a preexisting product root in the existing pure tests. Align new-branch original task/tool/raw-reference text bounds (Python 512 versus Go 128); preserve legacy native guards. Keep legacy branches unchanged.
- F05: Align the source-owned activation recipe template's `newBudget` to the repaired activation validator's complete cap mapping, including separate baseline/final enumeration. Keep all actual native facts unfilled and all current executable flags false.
- F06: Within the already admitted focused unittest invocation only, validate the actual pure `validated_product_binding` helper with temporary roots and actual four-field run-intent shape: valid binding accepted; wrong admission/source/start-window/caps/review hash rejected. No copied validation or text-only assertion. Block every subprocess entry point during the test; load definitions under a non-main name and never call main/preclock/activate/monitor. No separate parser/import invocation or operational extraction is admitted.

Root resolves the reviewer's caution about unexecuted metadata as follows: the immutable Owner grant explicitly places source metadata and tests before the campaign clock and prohibits runtime/browser/service/key/provider effects. A narrowly isolated offline helper test inside one of the six approved checks is within that scope. It is not activation or a runtime release. Preserve the original unexecuted authoring receipts; after checks, qualify current metadata status as tested only for the named pure helper, never campaign-tested. No additional source path, helper, check slot or external effect is granted.

The audited pure Go selector is software-only and permits temporary filesystem evidence already used by the inherited nine pure cases; it forbids DB/network/provider/browser/Docker/product-process effects. Independent review confirms the new page-target case has only temporary file I/O and validator calls. Actual original native authenticity still requires later Root review. Go terminal helper coverage is software evidence; integrated runtime terminal enforcement belongs to the Python supervisor, not an invented integrated Go terminal result.

Root releases source worker repair writes for F04–F06 in its existing three source paths/payload/own return only. Metadata F01–F03 repair hashes are activation `6df45acd58a657a19ffaf2e70a064d217c3de8ebda7c67b1fcb89a36b78bcb89` and recorder `af51faf9944ec7dce7d4240edfc24fe311887d4caa535f47e7afbffa133fb00d`; Root static review corroborates those corrections. Checks remain held until the combined repaired candidate is returned. Repository gate unit suite passed 89 tests in 58.959 seconds, separately from the six source-check slots.

## Repaired candidate / bounded worker check release

Root statically checked the combined repaired candidate and released worker Python attempt 1, followed only on success by offline Go compile attempt 1. Before each launch the worker must exclusively reserve the attempt and preserve actual pins, argv/environment, raw stdout/stderr, original handle wait/exit and timeout. Python timeout 120 seconds; offline compile timeout 300 seconds, no network/download/dependency installation. First failure stops dependent checks. Root's independent Python/pure-Go slots remain held. Source writes are frozen during checks; own return/evidence writes remain allowed.

F04 integrated reachability clarification: Python requires the product root absent before supervisor launch. Go is launched after that same supervisor creates the root. The new Go guard rejects an ordinary preexisting root, but accepts the supervisor-created exact root only with its actual existing RUN/ROOT/SOURCE_COMMIT/ADMISSION_SHA environment bindings and matching four-field run-intent, including bounded recent start. Both negative and legitimate owned-root cases are in the pure Go test. A blanket absent-root check in Go would incorrectly block G1; Root caught and resolved that within this same consolidated repair before using any check slot. Legacy branch guards remain unchanged.

Final candidate hashes at release: supervisor `3907b6fdad76a50d804245622296891496f9e54b566d2ed24adcfc8c67ec5ebf`; Python tests `a4d4bf8ba461eecf195683eb6f5ab2c13cea8914da3208d540d3f467c57fe6a7`; Go test `df940dbc14dd350cbd8c2d183efcf5db23ffe0efed935231a9405c0d339f13d4`; payload `f0996b831f0ab53989af8ce079306059fc1cc30eb962194b7485ea6f5e7b4b05`. Legacy preflight PASS 7/7 on the repaired working set. No runtime/campaign, browser, service, key or provider effect is released by this section.

## Independent acceptance of actual offline evidence

F01–F06 are resolved in the exact candidate above. Independent Luna read-only re-review matched all six source/metadata hashes and found no unresolved material blocker; Root corroborated code, actual receipts and current binary bytes.

| Actual check | Result | Use / cap |
| --- | --- | --- |
| Worker focused Python | 40 PASS; original wait exit 0; unittest 6.310s | 1 / 2 |
| Worker offline Go compile | Exit 0; original wait 105.366s; raw stdout/stderr empty | 1 / 2 |
| Root independent focused Python | 40 PASS; original wait exit 0; unittest 6.472s | 1 / 1 |
| Root audited pure Go | Exact 10 selected tests PASS, zero skips; original wait exit 0 in 3.329s | 1 / 1 |

Four of six maximum attempts were used; the unused worker attempts are not needed and do not become a runtime/retry allowance. Source, metadata and binary hashes stayed unchanged. Compiled binary is 59,411,456 bytes, SHA256 `74072a5f34b94ed090520bba3b21095cc2caac18be6eb08e6b10fe9a8d1fdb61`. Root verified each worker reservation/result/raw stdout/raw stderr against its compact receipt hashes. Root check reservations/results and `root_combined_source_acceptance.json` preserve the actual evidence bindings. Worker `binaryExecuted=false` describes its handback; Root subsequently ran only the audited pure selector, never the integrated fixture test.

The metadata authoring receipts remain historical unexecuted reports. The two focused Python invocations subsequently loaded both design definitions under non-main names with subprocess entry points blocked and exercised only the actual pure product-binding helper. This supports syntax/definition loading and named binding cases only. Recorder process lifecycle, activation, deadline coordinator, browser capability, original native provenance and G0–G3 product effects remain NOT_RUN for this campaign. No mock or pure software result proves runtime AI governance.

Root accepts the adapted source and bounded offline software evidence, pending material commit and separate accepted-build pins. Product B remains BLOCKED_PENDING_OPERATIONAL_ADMISSION; no campaign clock, browser/service/key/provider release, product closure, hosted CI or public release is claimed. Next authorized work is complete preclock operational preparation under the same Owner envelope, followed by actual G0 and independent Runtime Effect Admission if its prerequisites hold. Current metadata-only Antigravity availability request has no reply at this review; do not spend the campaign to discover coordination availability.

Repository gate qualification: the first worker-return profile invocation failed ROLE-01 because Root omitted this review's explicit Worker line. Root added that single required declaration; source and check results were unchanged. The next worker-return invocation passed all six mandatory controls, including active-order APPL and independent-review ROLE. This documentary gate failure is retained and consumes no source-check attempt.
