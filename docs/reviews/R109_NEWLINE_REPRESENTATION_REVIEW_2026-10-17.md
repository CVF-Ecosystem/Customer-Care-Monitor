# Admission text-representation repair

Status: REVIEW_PASS
Risk: R2
Worker: /root/r108_browser_luna (Luna Max REPAIR_WORKER)
Reviewer: Codex /root
Disposition: SOURCE_REPRESENTATION_REPAIR_ACCEPTED_RUNTIME_NOT_RUN

Root's read-only committed admission check at b4bcc8a7 failed before creating
the campaign ledger: `committed bytes differ from workspace:
frontend/src/i18n/vi.ts`. The exact error and independently computed physical
and Git hashes are in first_admission_byte_failure.json. The locale is CRLF
in the workspace and LF in Git; normalizing only CRLF gives identical text.
This is an independent representation cause, with zero fixture starts/native
invocations; it is not a repeated runtime attempt or a budget reset.

The worker changed only two comparisons in the new supervisor. Physical
SHA256 pins remain exact. Both Git/workspace comparisons now require valid
UTF-8 and allow CRLF-to-LF only; other content differences are rejected. No
UI source, previous validation, old runner, binary or seed was rewritten.

Root reviewed the complete two-comparison diff and executed a targeted
read-only check: the actual CRLF locale agrees with its committed LF content,
an incorrect physical hash is rejected, and all ten bound evidence files
compare successfully. AST parse PASS. Evidence: newline_validation.json.
The three prior harness regressions and actual Node config import are
inherited for unchanged functions/template. UI14tests/typecheck remain exact
and unchanged. An initial inline checker invocation had shell quoting syntax
failure; the saved helper then completed the stated checks, with no runtime.

The initial admission bytes are preserved separately. Root must commit this
source and refresh the current admission pins before releasing the one
campaign. Broader product/native/browser acceptance is still pending.
