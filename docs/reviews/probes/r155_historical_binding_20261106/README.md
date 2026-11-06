# R155 historical admission — uninstalled partial candidate

Disposition: **UNSUPPORTED / NOT READY FOR INTEGRATION**. `candidate.py` is a
standalone, project-local provenance preflight. It deliberately returns
`UNSUPPORTED_HISTORICAL_COVERAGE_JOIN` even when its checks succeed. No installed
or parent module imports it and it cannot produce a historical admission PASS.
The `candidate1` schema is a design sketch, not a recognized live schema.

The candidate checks strict JSON/bounds, exact binding shape, unique packet and
tranche, relative non-symlink paths and LF-normalized file digests, retrospective
`pr-ci` only, no active/explicit packet, exact checkout head, full SHA anchors,
ancestry ordering, and selected Git tree bytes. It pins admission registry, audit
and seed, terminal record, and packet at admission and terminal, and checks
retained local review evidence. It does not authenticate review or consent.

## Required parent work before any use

1. Define and independently review the separate binding schema and delivery,
   including immutable binding/review provenance, tranche identity and independent
   freeze evidence. Review exact seed/terminal semantics: the R150 terminal record
   intentionally names a CLOSED status carrier, while the admitted historical
   packet is joined through the seed and admission audit. A `FROZEN` string alone
   is insufficient. Verify admission `c61b01fa89c40135093103a0852de70f82bd39ec`;
   the earlier `ec6eed18a5f1951d9dc54a5b49d14f804574ab67` is not that admission.
2. Extend the parent runner/coverage API with an explicit, trusted range-candidate
   context carrying phase, checked head, active orders and explicit execution
   packets. Do not expose historical selection to execution paths. Reject ambiguous
   selection and unsafe inputs before invoking any join. Parent must also cover
   symlink race resistance, Git object read bounds, and all unselected entries.
3. Keep current registry/audit validation, transition and cost checks, unresolved
   product guards, and all unrelated coverage validation running unconditionally.
   Validate historical registry selection of the exact audit, retained dependency
   pins and historical audit against the existing semantics; compare history prefix
   and affected criteria against current state. Only then substitute the historical
   action join (selected/kind/owner/output/packet/question/budget), leaving current
   action strict for active and default validation. This join is **not implemented**.
4. Add complete positive/negative regression tests, including the actual R150/c61
   and wrong ec6 case, both-parent collision, frozen proof, mutable evidence and
   seed/terminal, current R151, duplicate/nonfinite/unsafe/symlink/oversize,
   malformed/unavailable commits, costs/history resets, product COMPLETE and
   execution rejection. Where retained dependencies differ from historical pins,
   explicitly refuse as UNSUPPORTED pending a separately reviewed virtual Git-tree
   resolver. The six local tests exercise only the preflight subset, not this list.
5. Parent review, source delivery with verified hashes, a separately authorized
   project upgrade and original range recheck are distinct future steps. This
   proposal changes none of those surfaces and does not close RC-01.

## Evidence

`baseline.json`: installed profile1.3 original `f7405c47..HEAD` pr-ci result,
exit 2, `COV_REPAIR_ACTION_MISMATCH` retained. `candidate-tests.txt`: six methods
passed but suite exit 1 because Windows denied removal of read-only Git objects.
`candidate-tests-repair.txt`: one authorized targeted rerun after fixing cleanup,
six methods pass, exit 0. These are static candidate fixtures, not live governance.
The first disposable fixture directory remains: an attempted verified-path
PowerShell removal was rejected by the execution policy. Do not treat it as a
project source artifact; Root may decide how to handle it within policy.
