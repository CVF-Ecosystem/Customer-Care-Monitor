# Independent R108 diagnostic source admission

Status: SOURCE_REVIEW_PASS
Risk: R2
Worker: /root/r108_browser_luna (Luna Max, IMPLEMENTATION_WORKER)
Reviewer: Codex /root (independent source review)
Disposition: SOURCE_REVIEW_PASS_RUNTIME_NOT_RUN

Root reviewed the entire opt-in Go diagnostic and one-shot Python supervisor. Production engine/handler/model/frontend source is unchanged. This admits source only, not a product repair or native result; runtime still requires the separate committed source_admission.json and an explicit Root instruction. Exact final source SHA256 values are in probes/r108_root/reviewed_source_pins.json.

The diagnostic reuses the real synthetic R101 router/MySQL fixture; four action POSTs are bounded and captured before/after. The saved B06 Unicode reason is reused with future due time. Safe query/create/update callback error classes, MySQL number/SQLState/constraint and pre/post schema/charset/collation/index/trigger metadata are recorded without unsafe SQL/binds/credentials. Full persisted case/event SHA256 and unchanged proposal comparison support failed-action rollback checks. Native success must reach ASSIGNED v5, clear active outcome and retain a nonempty previous outcome in history. Exact historical outcome text still needs the later rendered UI acceptance; do not broaden this into full M3 proof.

Root consolidated review found missing/unused imports and incomplete failed-action observations; worker corrected them before compile. A refinement added full persisted-state hashes rather than claiming equality from partial summaries. Root runner review then found incorrect seed base location, leading-comma map JSON, effect-vs-cleanup deadline threading, missing native timeout check, optimistic finalization and incomplete whole-backend pin protection. All were corrected. A newly introduced nullable termination-metadata access was corrected before runtime. No failed compile/native attempt was spent in these source repairs.

Runner checks committed independent admission, exact two source pins at HEAD/worktree, reviewed source ancestor, clean entire backend/runner, immutable seed at record base/first-add, active record/order/continuity, pinned Docker/Go and local pipe. One exclusive append-only campaign reserves compile/fixture/native before effects; offline cached Go only, no listener/Vite/browser/provider/install. Owned process handles and tree termination are scoped; fixture start intent precedes command; finally uses the reserved cleanup deadline and stops only the exact disposable fixture it attempted to start. Full sanitized stdout/stderr bytes and hashes are retained. Finalization failure becomes HOLD, not success.

Validation: Python AST parse PASS (no module execution); actual safe Docker target/network/protected format outputs parsed as JSON PASS, fixture exited and starts0/compile0/native0. Final format strings equal inspected ones; later source delta was metadata/guard repair only. Go version was actually observed as go1.27.0 windows/amd64, not inferred from go.mod. The source is gofmt formatted; compilation has not run. Original R107 logs/source/seed/counters and Dashboard failures remain unchanged.

Limitations: GORM query/create/update callbacks cannot guarantee capture of transaction begin/commit or failures outside those callbacks; NOT_CAPTURED is explicit. Earlier lifecycle failure is a HOLD with its original diagnostic in full stdout, not REOPEN reproduction. Internal files_manifest precedes the final disposition marker; worker export must separately hash the entire final runtime including marker/empty directory. Resource checks compare safe Docker fields, not user-database content hashes. No root cause, repair, full product, hosted CI or AI governance acceptance is claimed.
