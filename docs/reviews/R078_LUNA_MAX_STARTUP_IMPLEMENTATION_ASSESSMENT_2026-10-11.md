# R078 Luna max startup implementation assessment

Fresh source/test implementation by `gpt-6-luna` at `max`, root orchestrator
and independent reviewer. Final disposition is REVIEW_PASS on the synthetic
startup contract in the [review](R078_INDEPENDENT_STARTUP_HARDENING_REVIEW_2026-10-11.md).

First product source `6219da3` correctly removed the startup sweep with a small
bounded diff. The first static handback was CHANGES_REQUIRED: F1 missed two
row-isolation controls and direct owner-context checks; F2 had a mutation
count bug and incomplete source/publication/cleanup guards. These were found
before runtime; original first source/runner compile and positives are
NOT_RUN. One consolidated worker repair changed the NEW test and NEW runner,
not product semantics. Repaired source is `b7c826a`.

First actual worker build PASS; first actual positive PASS12 top-level/16
events after the static repair. M01 was killed by the named attempted-write
assertion with a healthy lease control; restored positives PASS12/16. Worker
used exactly4 Go invocations, no retry. Source restoration, raw receipt
hashes and owned fixture cleanup were independently read back. These are
synthetic application checks, not governance/provider proof.

Root supplied the bounded order, acceptance/failure conditions, existing
fixture/source pointers, consolidated findings and offline execution contract.
Luna authored the product/test/worker runner; root supplied no implementation
or test draft. Root mechanically bound source/archive pointers, reused the
reviewed runner for separate execution and authored a read-only raw auditor.
Root errors/overhead are recorded separately: initial continuity projection
fix, guessed file/doctor searches, one failed shell-quoted Python metadata
edit and correction of the copied reviewer actor label before runtime.
Those are not worker product findings. No runtime retry came from them.

Independent root build/positive/M01/restored also PASS on the same source,
with12 top-level/16 events in each positive/restored and the same named
detector/healthy control. Aggregate8/8 Go, retry0; both cleanup packets PASS.

Quality conclusion: bounded
product edit was correct on first static review, while test discrimination
and evidence tooling needed one repair. This is one fresh implementation
sample, separate from R076 design and R077 topology proposal. Historical five
implementation entries and the three documentary/audit arrays stay intact.
There is no matched max/xhigh/Sol comparison, model token/cost telemetry or
inference-time measurement; Go wall time cannot establish model speed.

Durable recovery/fencing and deployed topology remain unresolved. The accepted
contract cannot clear stale rows or establish old-binary/mixed-fleet safety.
