# Case participant names — independent review

Status: REVIEWER_ACCEPTED
Risk: R2
Worker: Sol Medium backend participant scope; Luna Max frontend reader labels
Reviewer: Root independent source/native/browser evidence review

REVIEW_PASS for the observed reader-name defect at `bc5333946ddefcfa137db4f3c253ea507bdf06e5`. Root reviewed all six source/test files independently; no schema, action transaction, source guard or provider behavior changed. Detail adds optional participants containing only user_id/name for same-tenant IDs referenced by the current case, actor events and direct historical assignee/data-owner projection fields. One batch query; malformed IDs ignored; lookup failure returns503 with only participants_unavailable. Full assignee directory requires both read and write. UI prefers detail participants and retains its localized missing-user fallback; it does not load the full directory for readers.

## Verification

Actual native MySQL7 top-level tests PASS (7.510s), including historical names, exact participant field/privacy boundaries, unrelated/foreign omission, reader directory403/editor200, injected participant-query error503 and restored healthy200, and existing lifecycle/source/permission regressions. Frontend2 relevant files/21tests PASS (20.98s), including reader actor/current/historical names and stale-response navigation, no action controls/directory call. Typecheck/Vite build PASS. Current backend/frontend snapshot hashes match all271/143 files at this source. Full Go/full frontend suites were not repeated for this bounded change; no fresh full-suite claim.

Root operated the actual current frontend/backend browser: CORRECT200/staleCONFIRM409/ASSIGN200/RESOLVE200/REOPEN200, WAIT_DATA assignment/escalation200, reload persistence and original outcome history. Fresh reader login/reload displayed both current assignee and actor names; WAIT_DATA displayed its data owner and actor names. Both reader views had zero action/save controls and no unnamed fallback. Journal after reader login contains no full-assignee directory request. Browser native fixture PASS exit0 (240.663s), Docker exited0, actual server termination observed; proposal/source identity unchanged, exactly6new human events and zero provider/accounting deltas. No real keys/customer data or installed app mutation.

Current R128 HTTP tabs closed after logout; owned synthetic MySQL stopped, listeners absent, containers/volumes/networks preserved. Three original app containers remain running by exact-ID observation; no all-content comparison claimed. Two older expired fixture error tabs were left unchanged after browser URL policy blocked access; no policy workaround attempted.

Full M3, B09 offline, business/pilot/hosted/release acceptance remain OPEN. No push/deploy. R127 historical failures and original receipts remain unchanged. Next consolidate the remaining M3 criteria using existing proof, rather than opening a generic support scope.

Evidence: [summary](probes/reader_names_20261019/verification_summary.json), [original receipts](probes/reader_names_20261019/original_verification.zip), [manifest](probes/reader_names_20261019/evidence_manifest.json), [reader screenshot](probes/reader_names_20261019/reader_names.jpg), [history screenshot](probes/reader_names_20261019/reader_history.jpg), [cleanup](probes/reader_names_20261019/cleanup_receipt.json).
