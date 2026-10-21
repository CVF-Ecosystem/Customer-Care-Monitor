# R139 source-evidence independent review

Status: HOLD / CHANGES_REQUIRED
Tranche: CCMAI-RUNTIME-139
Worker: Sol Medium /root/r139_source_evidence_sol
Reviewer: Root independent source REVIEWER (no source authorship)
Dispatch: `a1c1b22b262660690d997079d26c4141dad54f68`
Risk: R2

Root rehydrated canonical current files and transitioned ORCHESTRATOR to independent REVIEWER before disposition. Core doctor27/27 passed; public Core b69e1088 matches origin/main, inherited manifest/profile e9ddcb pin warning retained. The Owner-approved one source-only round is consumed; original runtime/manual/cleanup HOLDs are immutable.

## Findings

The candidate changes exactly the three admitted sources. Root independently inspected the early R139 cleanup finalizer, actual safe request/response journaling, observed execution/case fields before publication assertions, and exclusive file IO. Historical shared transport helpers remain untouched. The completed terminal is reserved for validated publication; partial/fatal paths write a distinct HOLD receipt and do not fabricate the missing historical response. This is source analysis, not runtime proof.

The Docker comparison strictly parses elapsed text for Up/Exited and validates matching State. Health/exit code/IDs/RunningFor and every other field remain exact. Unknown status formats fail closed, and raw inventory copies remain unchanged. Source intent matches the approved scope; Python behavior has not been executed in this round.

The independent package-init and exact six-test safety audit is in `probes/r139_integrated_20261021/r3_root_go_init_audit.json`. It permits only filesystem/in-memory IO checks. The worker's offline Go compile exited0. Root independently verified all three source hashes, binary hash and both raw-log hashes against `r3_source_checks.json` without launching the binary.

**R139-SRC-CHECK-01: required focused checks are incomplete.** The sole selected-binary invocation exited2 with `flag provided but not defined: -test`, before any test started. Zero tests started/passed/skipped; no assertion failure or successful skip is claimed. PowerShell/native flag handling is a plausible cause, not proven by argv tracing. Worker and Root Python invocations remain0/1 under the first-failure stop. No retry/source edit followed. Compile alone cannot establish source acceptance.

## Disposition and next move

HOLD/CHANGES_REQUIRED; no REVIEW_PASS, FREEZE, source acceptance, E05/E06/M3 or hosted CI claim. Candidate patch is retained for review, not discarded or silently accepted. Source round1/1, compile1/1 and selected binary invocation1/1 consumed; workerPython0/1 and RootPython0/1 remain stopped. New source-stage key/DB/Docker/fixture/browser/network/provider effects0. The old paid slot1/1 MAY_HAVE_STARTED remains consumed; response/usage/billing and manual-hold root cause UNKNOWN.

The concrete check-only recovery proposal is `probes/r139_integrated_20261021/r3_check_only_recovery_proposal.json`: no source edits/recompile; one separately counted literal-argv binary invocation, then the two unused Python checks only if Go passes. Exact source/binary identities, timeouts, six-test/zero-skip acceptance and first-failure stop are pinned. It is not authority and cannot execute without explicit Owner approval of this new check boundary. The failure log remains preserved locally with its public hash; no runtime or API grant is requested.

Closeability recheck: source receipt and independent review exist, but focused checks are incomplete. Contract cannot close. Keep same R139, preserve roadmap gates and parked Facebook/Zalo OA.
