# R139 Claude receiving admission proposal

Status: DRAFT_PENDING_OWNER_DECISION
Risk: R2
Worker: Claude owner-designated receiving worker
Reviewer: Root independent reviewer of Claude work; proposal author Root
Commit steward: Root
Admission effect: NONE — this document is a proposal, not an authority seed

## Decision and result sought

Same CCMAI-RUNTIME-139, M3-05/M3-08 G0 readiness. A accepted as bounded documentary intake with independent reviewer qualifications. Propose ONE bounded offline build/check plus explicit receiving-browser contract resolution, to avoid product fixture costs while binary and supported browser are missing. No helper tranche, old receipt rewrite or budget reset. B product execution is still blocked and is not granted by approval of this packet.

Owner decision requested on two separately identified boundaries: (1) one new offline compile and one selected pure-Go invocation in this receiving checkout; (2) amend S02's CUA-IAB-only topology to permit Claude's native browser_subagent ONLY if its own documented policy supports the proposed use, without retries of any denied route or evasion. A topology decision is not browser capability proof. This packet admits zero browser control/server/process probe, credential, Docker, DB, product server, grant, execute, external network or provider calls. No money requested or authorized.

## Exact offline build/check contract proposed

Review baseline:4265ce7deba049d84896ec0f6ddf32348e6aeaae. Before launch the committed admission must pin the exact final 40-hex dispatch HEAD, accepted product/source tree object IDs and this finalized proposal SHA256/Git blob; a symbolic branch, earlier baseline or phrase such as reviewed follow-up commits is not an executable binding. Relevant tracked paths must be clean at that exact admission commit; four accepted source files must match source material0ac2661825ff0c85bbdb74805d8086bca68991a7 and the sourceSha256 map in r7_root_source_review.json. Do not build if product/dependency/source drift or unrelated worktree changes are present.

Compiler: C:/Users/tiennm/AppData/Local/Programs/Go-1.26.7/go/bin/go.exe
Compiler SHA256:5463fe58fa999d74420f00ee1b36d31c3da90a57ff204159f159859567ad61fc
GOROOT: C:/Users/tiennm/AppData/Local/Programs/Go-1.26.7/go
backend/go.mod SHA256:15fe056859327cce63780c0390c82f93a7fa3c421adaffc9c289cd5961e24716
backend/go.sum SHA256:a7a5ba1e28e08702c104401b436cb139db1cea922fd07fdec27f99fa8308d67b

One compile, timeout120seconds, argv from project root:
`<exact compiler above> -C backend test -c -mod=readonly -o ../.cvf/runtime/r139_claude_receiving_20261023/handlers.test.exe ./api/handlers`

Minimal explicit noncredential environment: SYSTEMROOT, WINDIR, TEMP, TMP, USERPROFILE, LOCALAPPDATA and compiler/system PATH only; GOROOT above, GOPROXY=off, GOSUMDB=off, GOTOOLCHAIN=local, CGO_ENABLED=0. For BOTH compile and pure binary, TEMP and TMP must resolve to the fresh owned .cvf/runtime/r139_claude_receiving_20261023/temp/, GOCACHE to its gocache/ and GOTMPDIR to its gotmp/. Use absolute paths resolved inside this project and create only those proven fresh directories. Unsupported override or inability to verify confinement => HOLD before launch. Reuse the same scrubbed environment for the pure binary; do not inherit ambient test/provider/DSN credentials. Existing Go module cache is read-only as dependency source. Do not access credential files/ambient provider/DSN/admission variables or install/download dependencies. Cached dependencies missing => HOLD, no fallback compiler or retry.

One pure binary invocation only after compile exit0 and Root verifies the new binary receipt; timeout30seconds. Exact selector:
`^TestR139Pure(FinalEvidenceSuccess|FinalEvidenceFatalGoexit|FinalEvidenceIOFailure|JournalRequestIOFailureBlocksNext|JournalCapturedResponseSurvivesGoexit|JournalUnsafeResponseHasNoInventedReceipt|RouteAdmission)$`
Invocation: fresh binary above with `-test.v -test.count=1 -test.timeout=25s -test.run=<exact selector>`; no opt-in integrated test/environment. Seven expected PASS, zero SKIP; temporary/journal file I/O permitted in owned fresh temp directory. No DB/network/provider/credential effects. Failed/timed-out attempts consume their slot; no automatic rerun. Compiler subprocesses and pure-test process are the only proposed process launches. No broader Python/product/Go suites.

Result evidence: exact argv, cwd, environment allowlist, start/end, actual exit/timedOut, original stdout/stderr, compiler/source/new binary hashes, test names/pass/fail/skip counts, changed paths and budget accounting. Root must review compile identity before release of the sole pure invocation; this dependency is within the single packet and does not renew counters. A compile success alone is not test success or native/browser readiness. Each attempted or uncertain compile/test invocation consumes its own slot even on failure; an unattempted dependent test stays unused and does not become authority for later replay. Only compile exit0, exact seven PASS/zero SKIP and reviewed source/tool/new binary/changed-path pins yield ACCEPTED_BOUNDED_LOCAL_CHECKS. Every other outcome is HOLD, no retry; B runtime stays blocked.

Allowed new evidence only: docs/reviews/probes/r139_integrated_20261021/claude_receiving_20261023/build_* and docs/reviews/R139_CLAUDE_RECEIVING_BUILD_RETURN_2026-10-23.md. Ephemeral binary/cache/temp only in .cvf/runtime/r139_claude_receiving_20261023/. Existing path or ambiguous ownership => HOLD before launch; do not delete/reuse old paths. No binary/cache/archive committed. Worker may not edit source, authority, core, gates, workflows, dependency files, old receipts, continuity/catalog or commit. Root owns evidence registration/continuity/commits. Required static repository gates separate from build/test caps.

## Browser contract resolution proposed; no probe granted

If owner approves the topology amendment, Claude may document the already-exposed native tool's policy/capability declaration in its return; this is metadata only and must not invoke the browser. Root reviews whether the declaration actually permits loopback navigation, visual capture, own-tab closure and stable execution ownership. No assumption that browser_subagent is equivalent to CUA, no browser-launch grant, no bypass of an origin/tool policy denial. Unavailable policy/capability => HOLD_TRANSFER; do not fetch/install an automation stack.

Before an inert probe, a separately committed same-R139 operational admission must bind the exact executable command/harness, owned loopback target/port/nonce, parent and child handles, read/capture/close counts, finite wall timeout and exact-owned cleanup/wait/exit/listener evidence, plus independent review. No product fixture before successful actual capability proof. This proposal does not label that future contract complete or closeable.

## Product B remains blocked

A current-source integrated runtime admission still needs actual capability receipt, reviewed new binary identity, protected inventory/drift disposition, exact frontend/config/tool pins, exclusive fixture identities, synthetic cases/actors/grants, provider custodian/model/endpoint and a fresh owner cost/call grant, finite original-handle lifecycle/cleanup and independent review. These values cannot be invented from planning or old expired authority. Old uncertain physical slot stays consumed; old r3 spent/expired/newAPI0 unchanged. Protected drift HOLD and old-tab UNKNOWN retained. Facebook/Zalo OA parked. No runtime AI governance, full M3, CI, release or deployment claim.

## Activation and role contract

This proposal has no executable authority. Root must obtain the explicit boundary decision, resolve independent review findings, then commit a fresh same-R139 dispatcher-owned admission pinning this proposal digest, exact baseline/tool/source paths, role mapping, limits and owner reply before any build. Worker never edits that seed. Source record remains CHANGES_REQUIRED / order BLOCKED_WITH_REASON until actual admission/continuity transition; do not relabel this packet DISPATCH_READY. Root is reviewer of Claude implementation, not independent reviewer of this Root-authored proposal. Independent read-only proposal review is required before requesting the concrete owner decision.


## Independent proposal review

Luna Max /root/r139_admission_review, no proposal authorship, read-only, accepted this bounded draft for owner decision with three pre-dispatch clarifications: owned TEMP/TMP/cache for both commands; exact final source/dispatch/proposal binding; explicit acceptance/consumed-slot accounting. Root incorporated all three above. Reviewer performed no source writes/compiler/test/browser/network/DB/provider execution. This review accepts a non-effective decision packet, not an executable seed, actual capability, runtime governance or product readiness.
