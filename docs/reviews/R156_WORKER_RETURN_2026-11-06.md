# R156 worker return — actual joined candidate, independent verification pending

Disposition: **BLOCKED_PENDING_INDEPENDENT_REVIEW / FINAL_SOURCE_UNVERIFIED_BY_WORKER**.
Sol transitions from advisory to IMPLEMENTATION_WORKER under committed R156 seed
`0ea490f02f993d396def1cda32215dd1212cbad2` and BUILD marker `f1e2a9af707ff8d18127b035e5ddbe1c45a96e41`.
Canonical startup and exact packet raw SHA-256
`db8577f948788a811b3512e91d5c94fe0c2b2c91ea66f3d546f41caebc109f7f`
were rehydrated before implementation. Public Core c4c8767 remains read-only.

## Delivered behavior and source identity

The proposal under `docs/reviews/probes/r156_historical_join_20261106/` implements
the actual copied runner → coverage → repair historical audit/action join, rather
than an unconditional UNSUPPORTED scaffold. Exactly three installed module copies
are changed, plus one strict helper; other copied bundle dependencies match their
original hashes in `source_manifest.json`. `parent_candidate.patch` is the minimal
three-module diff; helper source, deterministic fixtures, README, construction
script, clock, raw failed logs and receipts are retained. Installed modules,
policy, lock, workflows, public Core, parent and R155 artifacts were not edited.

Current registry/audit, retained evidence, repair history/cost transition and
unresolved-product guards remain active. Historical registry bytes are read from
the admission Git blob with a pre-read size bound, separately from the current
R151 registry. Historical audit/action validation reuses existing parser and
matching semantics. Explicit retrospective context, immutable proposed binding
and review provenance, full ancestry, exact packet/seed, independently frozen
terminal record, separate CLOSED carrier and review evidence constrain selection.
Post-freeze mutations followed by reverts are rejected. Execution/active/explicit
contexts are excluded; final alias correction rejects backslash and `./` aliases.
Declared review pins are not authenticated operator or parent acceptance.

Final LF/raw SHA-256 (these candidate text files use LF):

- runner `923af9caf155b842ad5c14ae307966517274d1b28b70df303011de12e734cabc`
- coverage `f992b5bf646b76d70753b099bcab89ec91f04eae507d6f14688cb9ad8863d54b`
- repair `87bf3f18a37138e554a004ed5bb0b1a27c74a98fd300669f1a1eaffcd268883d`
- helper `3af939ccd7e5b6953498fa363aee14fa39eb003143ad696686f70d05d33dc68b`
- final tests `e564ea96b2375e6596da392c3c6b230823d5cb77af8d9b95ce4fc632f29e76c8`
- three-module diff `63ae8823f05f644fcb0e5c9a475cafc05cd03a94e4cf02cbf5fdace33b21c1bc`

## Verification truth — no worker green claim

Both authorized invocations used `python -B docs/reviews/probes/r156_historical_join_20261106/run_suite.py`.

1. Run 1: exit 1, zero test methods. Windows MAX_PATH prevented deep evidence
   materialization under the nested proposal fixture. Original traceback retained
   in `candidate_run_1.txt` and timing in `candidate_run_1.json`.
2. One justified correction moved fixtures to the shorter ignored
   `PROJECT/.cvf/runtime/r156/`, without changing the default coverage file reader.
   Root's consolidated static observations also added literal Git paths and bounded
   post-freeze mutation/revert protection before the final invocation.
3. Run 2: exit 1; 10 methods, 9 pass and 1 assertion failure, 110.452 seconds
   unittest time / 110.590 seconds process time. Actual retained c61 positive with
   current R151 registry unchanged, strict default R150 denial, current R151,
   actual copied runner control, genuine two-parent conflict, history/cost/product,
   ancestry, mutation/revert and other executed adversarial checks passed.
   Wrong ec6 was correctly denied with HIST_BLOB_ENTRY because the admitted packet
   does not exist there; my test incorrectly expected HIST_AUDIT_SELECTION.
   Duplicate/unselected subcases after that assertion did not execute.

After run 2, Root identified a real raw-path alias exclusion defect: discovery
normalizes backslashes while context carried raw explicit paths. Under Root's
explicit correction instruction, I rejected noncanonical active/explicit paths,
added backslash/`./` negatives, corrected the ec6 expectation with an explicit
wrong-registry audit assertion, split duplicate/unselected cases into independent
methods, and added optional `R156_REVIEW_REPORT_PATH` so review output cannot
overwrite worker fixture evidence. **No worker rerun followed these corrections.**
Final tests have twelve methods with parameterized cases. Root's one remaining
independent suite must establish their final result; it is not reported here as
worker evidence or an installed gate pass.

## Scope, cost and next action

R155 three spent invocations + R156 two = **five worker invocations, exhausted**.
Earlier/cumulative actual active time and monetary cost remain UNKNOWN. Prospective
monotonic measurements and a conservative declaration wall bound are retained in
`clock.json`, run receipts and `worker_return_receipt.json`; no nominal 120-minute
compliance claim or allowance reset is made. Manual semantic source inspection
covered the three installed modules, common result primitives, R150 seed/tranche,
CLOSED carrier and registry (eight files); lock hashes were provenance metadata.
Pinned evidence materialization/validation is separately listed in
`fixture_sources.json`, not concealed as additional source discovery.

Only proposal files and this return were written as tracked-artifact candidates;
temporary isolated fixture IO is under the authorized ignored runtime. Fixture Git
commits are confined to those disposable repositories. Actual history is read via
Git object alternates; only the finite pinned evidence closure is materialized.
No real project stage/commit, application, provider, network, browser, daemon, DB,
credentials, customer/channel, deploy or release effect was used. Fixtures were
retained without recursive deletion. Root's separately created learning intake
was observed but not edited by this worker.

Root should inspect final source/diff and run the independent suite directly:
`python -B docs/reviews/probes/r156_historical_join_20261106/test_join.py`, setting
`R156_REVIEW_REPORT_PATH` to Root's own ignored report destination. Do not invoke
`run_suite.py` again. Worker stops now; no further automatic successor or test.

Parent source/schema acceptance, verified delivery, deliberate authorized project
upgrade and original required range recheck remain separate future decisions.
Original installed COV_REPAIR_ACTION_MISMATCH is retained from R155, not rerun.
R154/RC-01, R155, R151/R139/M3 remain blocked; R150 spent effects, INCONCLUSIVE/A2
UNRUN/terminal cleanup UNKNOWN and Facebook/Zalo OA parking remain unchanged.
