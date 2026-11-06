# R155 worker return — partial, uninstalled, blocked

Disposition: **PARTIAL / UNSUPPORTED**, independent Root review required. Fresh
R155 continuity and committed BUILD transition acknowledged before work. The
candidate is only an isolated provenance preflight; successful checks explicitly
return `UNSUPPORTED_HISTORICAL_COVERAGE_JOIN`. It does not fix the installed gate,
prove historical admission, or close R154/RC-01. Current R151, R139/M3, R150
inconclusive/A2 unrun/cleanup UNKNOWN, and account parking remain unchanged.

Artifacts and parent change contract: `docs/reviews/probes/r155_historical_binding_20261106/README.md`,
`candidate.py`, `test_candidate.py`, `baseline.json`, `candidate-tests.txt`, and
`candidate-tests-repair.txt`. The source lock uses LF-normalized SHA-256: installed
repair module `7c58133bc73af0b300cb2674768864c5c8ec6680cf7103905f705897faa8d03d`,
coverage module `14e1efd0acbd54cc9b86f0b33060a6dd57764c16812a07df8e18bca68bfa141f`,
runner `0447c57657ff0616f2b9c16b901fb96cb4381022acdfc7f198e1512e6c182a49`;
profile `41a99910a57a5d2131f16f83d72e38e07991bf0c4f833601e668fea548a7d2e6`,
bundle `b047fac687337aa83e8d17608c4a0712ea728eca24c02a66a8805690b3516ee3`.
Anchor project `e38d679548bdc80ba9f4bc10a2e0d09056a2189f`; checked HEAD
`b04f60c544617c494379cd8f92fe54aa1d3ed26a`. Source files inspected after
startup: installed repair, coverage and runner; R150 seed and tranche (5 total).

Checks, in order, exactly as authorized:

1. `python -B scripts/cvf_gates/cvf_downstream_gate_runner.py run --phase pr-ci --project-root . --base f7405c470207dc0d3be64e6eca79b7050d3f79d8 --head HEAD --expect-profile-sha256 41a99910a57a5d2131f16f83d72e38e07991bf0c4f833601e668fea548a7d2e6 --expect-runner-sha256 0447c57657ff0616f2b9c16b901fb96cb4381022acdfc7f198e1512e6c182a49 --expect-bundle-sha256 b047fac687337aa83e8d17608c4a0712ea728eca24c02a66a8805690b3516ee3 --json` — exit 2, original `COV_REPAIR_ACTION_MISMATCH` retained.
2. `python -B docs/reviews/probes/r155_historical_binding_20261106/test_candidate.py` — six methods passed; overall exit 1 due to Windows read-only Git object cleanup (`PermissionError`). Original result retained.
3. Same candidate test command, one justified rerun after changing test fixture teardown to make files writable — six methods, exit 0, 3.959 seconds. This is the single allowed repair rerun. No additional test invocations.

Not proved: historical audit/action join, correct c61 versus ec6 behavior, reviewed
freeze proof, old history/current cost and product guards, actual two-parent merge,
actual R151 behavior, complete adversarial cases and dependency virtual-tree support.
The README identifies their owner and necessary parent integration. Synthetic
fixture tests do not prove product behavior or live AI governance. Worker active
time and earlier advisory costs are UNKNOWN; no cap extension is claimed. No
provider, browser, network, database, product runtime or real repository commit
was used. The first temporary fixture remains because an attempted verified-path
PowerShell cleanup was rejected by automatic execution policy; the test evidence
and this limitation are retained. Root owns disposition and any safe cleanup.
