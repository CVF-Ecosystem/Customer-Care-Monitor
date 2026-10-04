# R041 independent exact-BUILD review

Date: 2026-10-03. Codex independent REVIEWER of Claude IMPLEMENTATION_WORKER. **CHANGES_REQUIRED / REVIEW / FREEZE_OPEN**. BUILD `e9043813ea5b9ffbbf5ee6218d4a9cdb343f515b`, parent dispatch `e620b73`, handback `0ed861e`; seed/base `b19602ea66a47310b503a03ec2954092560231cc`. [Order and R1 scope](../work_orders/CCMAI_RUNTIME_041.md), [SPEC](../specs/PANCAKE_OFFLINE_INVENTORY_INPUT_R041_2026-10-03.md), [worker record](PANCAKE_OFFLINE_INVENTORY_INPUT_R041_BUILD_2026-10-03.md).

## Findings

| ID | Severity / independent evidence | Required disposition |
| --- | --- | --- |
| R041-R1-01 | High, OI-03/no-network boundary: inventory.go68..73 blocks only two identical leading separators. Windows filepath.VolumeName classifies both mixed leading separator forms as UNC; current filter returns false, so control reaches os.Lstat before rejection. No share/server was contacted in review. | Host-correct syntactic UNC rejection before any filesystem I/O; non-I/O regression and applied detector. |
| R041-R1-02 | High, OI-03: inventory.go73 only Lstats the final file. A synthetic inventory reached through a task-owned ancestor directory junction is accepted; mounted CLI exits0 and emits PASS. Before/after SameFile compares the same target and cannot detect ancestor traversal. | Reject detectable symlink/reparse ancestors before opening; direct and ancestor regression, mounted junction fixture exit2/empty stdout/fixed stderr. |
| R041-R1-03 | Medium, evidence honesty: inventory_test.go320 logs SKIP but parent test PASS and no machine skip event occurs. Worker discloses this accurately, but platform coverage remains absent from machine counts. | Isolated real t.Skip subtest, preserve other admission checks; explicitly retain actual symlink rejection UNVERIFIED until a capable host executes it. |

Current-state prose also contradicted implemented/REVIEW_PENDING headers with no-loader/no-worker/NOT_BUILT statements. Reviewer retires dispatch prose in memory/status/order/handoff and preserves historical labels. Copied acknowledgment R038 SPEC/R1 seed labels are wrong for R041/R2; historical text is retained and explained, not retroactively certified. Source repairs remain Claude's responsibility under unchanged seed.

## Source and contract checks

First authority commit is the seed/base, ancestor of BUILD, identical parsed first/BUILD/current content. Dispatcher attribution derives from routing and prior planning, shared Git author Blackbird081 cannot prove which agent wrote a seed or pre-edit timing. Seed-to-BUILD protected channels/engine/dependencies/frontend/scripts/workflows/authority/main_test.go diff empty. Only main.go(+33/-2), inventory.go and inventory_test.go change backend. Current LF-normalized source equals exact BUILD. SHA256 main.go `fd7dcb9e0230056fb605fd0315a18cfa31b84292693a317aeba7a166c4c2084f`; main_test.go `bde3d475a21bbfe2a55ffbc96a83e71e34a6e7c3464dab1f7c1344326f316b2f`; inventory.go `328007c292525d05bd803a9f0c8494c293449a76e426c8fe7817f9255e180f69`; inventory_test.go `dad0a4c5908e655693b1518c8eb7d4bf5105c63cc60cbeb12f4b8733ee1d24ed`. All match worker; original six tests byte-preserved.

OI-01/02/05 source and mounted regressions support expectation-only override, strict nested/duplicate/null/type/UTF8 validation, bounded parse, fixed sanitized output and offline receipt. OI-03 is blocked by findings above. OI-04 intended fail-closed invariant accepted with explicit SPEC-author correction: library pancake_proof.go1165..1180 prioritizes reconciliation_mismatch FAIL over empty_inventory INCOMPLETE. Nonempty observed with empty expected yields FAIL/exit1/4 attempts; both empty yields legacy INCOMPLETE. Equivalent file retains12 attempts; expectation-based message scheduling means unequal inventories do not require identical attempts. No library change or new scenario coupling. OI-06 protected scope and current tests verified, worker mutation campaign attribution remains separate. Overall acceptance withheld.

## Independent execution and reproduction

Project-root cwd, Go1.27.0 windows/amd64, GOPROXY=off/GOSUMDB=off/GOTOOLCHAIN=local/CGO_ENABLED=0. Task-owned synthetic files and overlays only. `go -C backend test ./cmd/pancake-proof -count=1 -json`:13 top-level/49 subtests PASS,0 FAIL/formal SKIP,6.161s,1 logged symlink omission (required privilege unavailable). `go -C backend test ./channels -count=1 -json`:116 top-level/156 subtests PASS,0 FAIL/SKIP,3.341s. `go -C backend build -o NUL ./...` and vet all packages exit0. gcc absent; race NOT RUN. These pass counts do not settle missing admission coverage.

Original-main control: task-temp Go overlay substitutes only main.go with seed blob, new tests/loader retained. `go -C backend test -overlay TASK_TEMP/original-overlay.json ./cmd/pancake-proof -run '^TestInventoryValidFilePassesThroughMountedMain$' -count=1 -v` exits1/package3.069s at named valid-file acceptance: original CLI rejects inventory flag, exit2. No compile failure or tracked source edit.

Pure UNC regression (no file/network operation):

```go
for _, p := range []string{`\/server/share/inv.json`, `/\server\share\inv.json`} {
    filtered := p == "" || p == "-" || strings.ContainsRune(p, 0) ||
        strings.Contains(p, "://") || strings.HasPrefix(p, `\\`) || strings.HasPrefix(p, "//")
    volume := filepath.VolumeName(p)
    if strings.HasPrefix(volume, `\\`) && !filtered {
        t.Errorf("Windows UNC path escapes loader pre-I/O rejection")
    }
}
```

Both volumes normalize to `\\server\share`, both filtered=false, both fail the intended assertion. First reviewer classifier mistakenly tested mixed separators in normalized VolumeName and passed; corrected assertion fails as above, no historical rewrite. This verifies path classification plus static control flow, not an actual remote I/O trace.

Junction reproduction: extract hand-authored validInventoryJSON from committed inventory_test.go into TASK_TEMP/target/inventory.json. `New-Item -ItemType Junction -Path TASK_TEMP/junction -Target TASK_TEMP/target` succeeds without symlink privilege. Append task-temp overlay test to inventory_test.go and run mounted `execMain(..., base("-inventory", TASK_TEMP/junction/inventory.json)...)`. Assertion requires exit2/empty stdout/inventoryRejected; observed exit0, PASS=true and normal synthetic banner. Named `TestReviewerR041JunctionAncestor` fails semantically (0.99s); combined corrected two-probe run exits1/package3.061s, no panic/timeout/compile failure. Initial junction probe also fails (0.91s). Fixtures never reference customer input, real shares, config or credentials; no product file modified. Symlink rejection itself remains UNVERIFIED; junction evidence is a separate mechanism.

Worker eight applied mutations and restoration are inspected/attributed, not independently replayed. Review stops expansion at reproduced blockers; no claim of independent nine-control campaign. Raw JSONL/probe overlays are task-temp, reproducible recipe above; source unchanged. Failed reviewer convenience rg wildcard/read of a nonexistent historical repair filename and first classifier assertion corrected locally; neither changes contract evidence. Shell compiler lookup ends nonzero because gcc absent; build/vet separately exit0.

## Boundaries and return

Reviewer ORCHESTRATOR returns first bounded R1 repair in existing order under unchanged R2 seed/path/effect/commit-owner contract. No automated delegation or product repair. Worker go-build exe deletion, rejected heredoc and three repaired development test failures remain attributed in original BUILD evidence; no claim they were independently recreated. Exact BUILD has no executable artifact in changed set. Shared workflow learning records platform SKIP and full path interpretation. Parent intake/adoption DEFERRED.

Doctor25/25 PASS; compact bootstrap absent/nonblocking, knowledge ingest task-temp only. No CVF Web bridge needed: this tranche makes no AI-governance claim. Engine/full backend DB/frontend/Analyzer/race, real inventory/capture/credential/config read, provider/channel/network/download/GitHub Actions/push/merge/deploy/FREEZE NOT RUN. Existing R034-R036 acceptance, R040/R037-R039/R033 closures unchanged. Offline code/test/repository-state evidence does not prove runtime CVF AI governance or live channel completeness.

Final publication checks: default and origin/main..HEAD PR-range preflight7/7 PASS; mandatory gate tests46/46 OK13.406s; docs build exit0 in13.94s with inherited env-highlighter warnings; catalog -Write/-Check and diff checks PASS. Reviewer changes eleven governed documentation/continuity/learning files only, source/tests/authority/protected closures unchanged. Current state/memory/handoff/order/tranche/status/index agree CHANGES_REQUIRED / REVIEW, R041, parked none. Exact staged-set preflight/diff checked before local review commit; no push/merge/deploy/FREEZE.
