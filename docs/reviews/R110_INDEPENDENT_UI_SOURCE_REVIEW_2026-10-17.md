# Independent R110 UI source review

Status: REVIEW_PASS
Risk: R2
Worker: /root/r108_browser_luna
Reviewer: Codex /root
Disposition: BOUNDED_SOURCE_ACCEPTED_WITH_THREE_INHERITED_TEST_FAILURES_OPEN

Root independently reviewed the six authorized source/test diffs against
b313727765da25fcf29599b2005472f8209c90b7. Four product files change presentation
only: vi/en translations, Dashboard label lookup and JobDetail alert text.
No wire key, route, permission, provider selection, request or source-state
behavior changes. Unknown service state remains unknown. The universal source
limitation no longer claims a data match for an incomplete source.

Ordinary navigation now says Kết nối trợ lý AI / AI assistant connections,
AI service and access key labels replace Provider/API-key terminology, and
the Dashboard uses user-facing application/storage/schedule labels. Root's
single consolidated refinement restored unrelated storage-option text and
made assistant navigation and Vietnamese model/key labels more precise.
No broad application-language or settings-runtime acceptance is claimed.

The Dashboard test still asserts three neutral unknown chips, visible
explanation, ordering, no false healthy state and no extra health requests;
its translated row-name assertions were updated and old technical labels
are prohibited. Two presentation assertions cover source-limit truthfulness
and ordinary AI/assistant wording. Existing lifecycle tests remain unchanged.

## Actual validation and remaining failures

One cached pinned Node invocation ran29 tests:26 PASS and3 FAIL, actual exit1
retained. All16 intervention tests PASS, Dashboard10 PASS/3 FAIL. Forced
vue-tsc PASS exit0. No rerun, weakened assertion or baseline repair.

Root compared the full three failing test names and their error signatures
against docs/reviews/probes/r101_worker/frontend_full.json: unresolved resolve
callback, missing4242 metric and unexpected /tenants/t1/me request match the
existing baseline. They remain OPEN. baseline_qualification.json pins that
historical evidence, actual current stdout/stderr/receipts and all11 source
inputs. Actual test exit1 is not rewritten to0. The qualification helper's
first read-only comparison rejected trailing CR in historical names; stripping
that line terminator resolved only the comparison, without another test run.

Receipts and source SHA256 pins are under r110_root/validation_1. Environment
contained only SystemRoot/WINDIR/TEMP/TMP, envDir false, cached modules and
sequential config imports. No provider/network/install/.env or product runtime.
Entire backend diff/worktree is empty, binary cb73ee13/58031616 bytes verified.

This accepts the bounded presentation change and authorizes further tooling
review. Runtime still needs independent exact committed admission; no full
suite, B07-B09, product readiness, AI governance or FREEZE acceptance.
