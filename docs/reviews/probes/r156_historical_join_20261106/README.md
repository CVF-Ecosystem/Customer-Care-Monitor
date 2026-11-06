# R156 uninstalled historical audit/action join candidate

Status: **UNVERIFIED_BY_WORKER_AFTER_REVIEW_CORRECTIONS**. Static candidate only;
not installed, parent accepted, a governance receipt, or RC-01/product acceptance.

This proposal modifies copies of exactly the installed runner, coverage and repair
modules, adds one strict helper, and copies all other bundle dependencies verbatim.
`source_manifest.json` pins originals and candidates; `parent_candidate.patch`
contains the three-module unified diff. The helper is delivered separately.
`prepare.py` reconstructs those three copies from the still-unchanged installed
source. No production/default file-reader behavior was changed.

The copied runner passes an explicit retrospective context into copied coverage
and repair validation. Existing current-registry transition, audit, evidence,
history/cost and product guards execute first. An eligible frozen packet may then
match its independently reviewed historical audit/action. Current/default R151
matching stays strict. The helper reads only the historical registry from its Git
blob, never requiring today's registry to equal historical bytes. Old audit and
other dependencies must remain pinned in the current worktree; actual drift
refuses admission. There is no general historical filesystem or bypass.

The proposed `cvf.historicalRepairAdmissions@candidate2` manifest has exactly
`schemaVersion`, `bindings`, `review`. Bindings have the exact keys in helper
`FIELDS`; references are path plus LF-normalized SHA-256. The context separately
pins manifest path/hash/introduction commit and checked head/base. An immutable
review artifact binds the canonical JSON of all bindings and declares distinct
worker/reviewer identities. These are declared review/provenance checks, not
authenticated consent or real parent acceptance. Synthetic review declarations
exist only inside static fixtures. All bindings are validated, including those
not selected. Execution phases, active/explicit packets and noncanonical aliases
are rejected. Admission/build/terminal ancestry, seed, frozen terminal record,
CLOSED status carrier and review evidence are joined. Bounded post-terminal Git
history also rejects a later mutation followed by a revert.

Git reads use literal paths, disabled replacement objects, regular blob modes,
size checks before object capture, one-MiB object and sixteen-MiB cumulative
limits, finite metadata histories and timeouts. Retained current/audit dependencies
also use the existing validator's own finite read budget. R150's historical
packet remains the admitted work order; the frozen record's `workOrder` may be
the separately validated CLOSED carrier.

## Evidence and exact limits

- Run 1: exit 1, zero methods executed. Windows MAX_PATH prevented setup under
  the nested proposal directory. Raw output and receipt retained.
- Run 2: exit 1; ten methods executed, nine passed and one failed. The real c61
  positive, strict current/default R151, copied runner integration, actual
  two-parent conflict fixture, current history/cost/product guards and
  post-freeze mutation/revert rejection passed. The ec6 case was denied with
  `HIST_BLOB_ENTRY` because its admitted packet is absent; the test incorrectly
  demanded the later `HIST_AUDIT_SELECTION` denial. Its following duplicate and
  unselected subcases were consequently not executed in that run.
- After run 2, Root found raw backslash/`./` execution aliases could avoid exact
  exclusion. The helper now rejects noncanonical active/explicit paths. Tests
  include those aliases, verify ec6's different audit plus actual missing-blob
  denial, and place duplicate/unselected cases in independent methods. These
  final source/tests were **not rerun by the worker**. Root owns the remaining
  independent suite and its result; no worker green total is claimed.

R155's three invocations plus these two total five worker invocations, exhausted.
`clock.json` and run receipts retain prospective monotonic timing. A conservative
wall bound starts at the startup timestamp 2026-11-06T11:14:07.6922275Z, preceding
the declaration and clock-recording origin 11:14:22.145283Z. Earlier/cumulative
active time and monetary cost remain UNKNOWN. No additional retry is authorized.

Fixtures use isolated repositories in ignored `PROJECT/.cvf/runtime/r156/` after
the first setup failure. They read actual retained history through Git object
alternates, materialize only the pinned evidence closure, and commit only inside
their disposable repositories. They never stage/commit the real project. No
full source tree checkout, provider, application, browser, DB or network run is
used. Fixtures are retained without recursive cleanup. `fixture_sources.json`
records run 2's fixture/head/evidence closure. Reviewer runs should set
`R156_REVIEW_REPORT_PATH` to their own report so that worker evidence remains
unchanged.

## Parent integration and independent verification

Review all three diffs plus helper and final test corrections. A reviewer can run
`python -B docs/reviews/probes/r156_historical_join_20261106/test_join.py` directly,
with `R156_REVIEW_REPORT_PATH` pointing to their own ignored evidence location.
This runs twelve methods with parameterized adversarial cases; a method count is
not a subcase count. Do not use `run_suite.py`: the worker allowance is exhausted.

The actual positive uses c61b01fa89c40135093103a0852de70f82bd39ec, build7052df5c,
terminaled9d55b2, the real current R151 registry and the real copied validators.
The final suite also constructs a local two-parent conflict without replacing
current authority with historical authority. Proposed binding/review artifacts
are created solely inside those fixtures, never admitted to this project.

Parent must independently accept/version the schema and source, include the new
helper in its delivery/bundle manifest and tests, and deliver verified hashes.
Only a separately authorized project upgrade may update installed copies,
lock/workflow pins or live binding authority. Candidate CLI flags
`--historical-admissions`, `--historical-admissions-sha256` and
`--historical-admissions-commit` do not make the old installed bundle valid.
Stage-0 installation/pin controls remain intact. This suite exercises the actual
copied coverage control, not a claim that the complete installed PR gate passed.

The installed original failure, R154/RC-01 BLOCKED, R155 partial result, all
R139/R151/M3 holds, spent R150 effects/cleanup UNKNOWN and account parking remain.
