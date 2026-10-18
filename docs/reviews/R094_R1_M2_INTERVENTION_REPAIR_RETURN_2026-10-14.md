# R094 consolidated M2 repair1 return

Status: COMPLETE_PENDING_INDEPENDENT_REVIEW
Risk: R2
Worker: `/root/m2_worker` REPAIR_WORKER
Reviewer: Codex `/root`
Authority: unchanged R094 seed `0e0663559c38279531f070afe231756f79b79422`
Commit: NONE — worker did not commit

This return addresses Root's [F1–F4 consolidated review](R094_INITIAL_INDEPENDENT_M2_REVIEW_2026-10-14.md) within the accepted source/test families. The revised source identity is [repair1 SHA-256 manifest](probes/r094_worker/source_hashes_repair1.json). The worker's Go invocations remain **6/6** and its sole disposable MySQL fixture has been removed. The revised source and new tests are **NOT_RUN**; Root owns the independent compile/native campaign with its remaining reviewer budget. The original worker logs and source hashes remain attached to the [initial return](R094_M2_INTERVENTION_WORKER_RETURN_2026-10-14.md) and are not claimed for this revision.

F1: publication errors are mapped through a fixed safe-code whitelist; arbitrary driver/commit text becomes `PERSIST_FAILED`. A physical response is settled on the stable consumed attempt before JSON/evidence and stale-source validation. Attempt state, actual response model, independently classified input/output usage presence, nullable numeric counts and KNOWN/PARTIAL/UNKNOWN status persist without raw body, headers or key. Invalid/stale proposal branches retain response truth, while failure of the checked settlement write prevents proposal publication and never refunds the already consumed cap. The root test fixture now includes a disposable-schema attempt-table failure as the last assertion and tests the safe case code/cap preservation; its result is pending.

F2: the scoped `AnalyzeChatOnce` path rejects completion-token counts outside 1–768, has a 60-second independent HTTP client timeout, refuses redirects/retries, enforces existing request/response byte caps, and verifies actual response `model` equals the configured model. The old `AnalyzeChat` path was not changed. Boundary, mismatch and oversize tests were added but not run after repair.

F3: unknown policy or unmarked/foreign profile returns structured safe 403 `DENY`/`NOT_RUN`/`SOURCE_DENIED` before run reservation or transport; cross-tenant conversation remains 404. Middleware authorization denial remains a 403 without case creation. The ordinary app's nil service returns 503 before case/call. Revised route tests exercise policy/profile deny, disabled state and actual cross-tenant case/source 404; results pending.

F4: added targeted route cases for fabricated quote, Unicode offset mismatch, kind/anchor mismatch, wrong digest/schema, malformed JSON, concurrent same-job admission, and checked response settlement failure. Original positive, WAIT_DATA, stale, cancellation, permissions, cap recreation, legacy bypass and no-retry/default-adapter selectors remain. Added source-only usage presence and publication-error whitelist unit tests. Native acceptance remains pending Root execution, especially the new destructive last-case failure injection in Root's own disposable schema. This does not add a live provider, credential, real data or installed application configuration.

No new Docker/container/DB/provider effect occurred in repair1. `gofmt` and `git diff --check` completed, and source files were hashed; these are syntax/format/source-identity checks, not Go compilation or native behavior proof. Full M2/live/M3/pilot/hosted/deployed outcomes remain OPEN.
