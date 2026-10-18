# R121 capability admission hold and minimal correction proposal

Status: CHANGES_REQUIRED
Risk: R2
Worker: Codex /root (capability author/operator only; product source not started)
Reviewer: /root/r121_isolation_luna (advisory independent static capability assessment; tranche disposition remains Root HOLD)

## Actual result

The original exec session 66544 remained available across a separate cooperative stop request. The parent retained its child process object, observed child exit0 through wait, and wrote a receipt after the child's end marker. The original write_stdin return, however, reports outer exit_code1 instead of the predeclared17. A2 is NOT_MET. Do not infer a successful admission from the child's exit0 or the receipt's declaredParentExit17.

Root used `python -B .cvf/runtime/r121_capability/lifecycle.py` through the default PowerShell shell without explicit native exit propagation. The source's successful branch calls sys.exit(17), and the returned stdout contains its valid receipt. This makes outer-shell exit mapping a concrete suspicion, but its exact behavior has not been independently reproduced. This observation does not establish the cause of the old R120 disappearance or a failure of product source, Docker or browser capability.

Raw initial/final tool returns are saved as `probes/r121_root/capability/exec_initial.json` and `exec_final.json`. The original probe source, reservation, parent-ready/receipt, child-start/end and stop sentinel are preserved byte-for-byte in `original_bytes.zip`, with hashes/sizes in `manifest.json`. Archive SHA256: `2e2a4c6115c59024d371854f403a7562c4caad23f1bd772b0d2674dd071628b7`. The actual child was joined and the original command returned; no dummy process is left awaiting a stop.

## Scope and accountability

Capability1of1 consumed. No retry, source copies, focused software checks, Go compile, fixture/backend/Vite/tab start, business action or provider call occurred. Luna acknowledged the hold before Root granted BUILD, so the implementation/source budgets remain unused. Root authored the probe and owns the missing shell exit propagation in its invocation. Independent static review may evaluate the proposed correction, but cannot turn this invocation into the required exit17 proof.

This is a failed admission, not an accepted product tranche. R121 remains CHANGES_REQUIRED / FREEZE OPEN. R120 bounded UI evidence, original missing terminal/accounting, unknown disappearance cause and consumed caps remain unchanged. Full M3 and other milestone exits remain open; Facebook/ZaloOA remain parked.

## Concrete prospective correction — not executed

Use the same finite probe bytes in a fresh isolated scratch directory and explicitly propagate the native exit through PowerShell:

```powershell
python -B .cvf/runtime/r121_capability_confirmation/lifecycle.py
exit $LASTEXITCODE
```

Requested additional scope, if separately granted: exactly one harmless confirmation, max90seconds, one short parent/child, one cooperative file stop, no retry, listener, Docker, DB, browser, secret or provider use. Require actual original exec exit17 matching the observed child wait/exit0 and receipt. Do not overwrite or execute the existing consumed probe directory. This proposal changes only invocation/isolated output location, not supervisor or product logic.

No new permission is assumed here. The immutable seed and A2 stop at the first capability mismatch; the existing R121 record does not grant a second invocation. Product runtime remains unadmitted. Any subsequent decision must preserve this failure and existing caps rather than silently altering the seed or acceptance criteria.

## Record checks

Doctor27/27PASS and governance unit89PASS (36.057seconds) preceded the dispatch. The initial dispatch checks found stale continuity/catalog pointers; Root corrected activeTranche, active handoff and duplicate catalog entry before dispatch commit d3b13107. Pre-dispatch APPL/ROUTE checkedPASS and legacy7/7PASS, with seed33186d01 committed before the unused source BUILD grant. No source BUILD grant was issued. These static checks are not product/runtime AI governance evidence.

Independent Luna assessment verified archive and every member hash, original child wait/exit0/stop marker, and actual outerexec1 vs required17. Luna agrees the proposed explicit native exit propagation is the smallest procedural correction; shell mapping remains inferred, not independently reproduced. The invocation text is separately Root-transcribed in invocation_record.json because the raw exec result contains no argv. No acceptance, rerun or cap renewal follows from this advisory review. Luna had observed the pre-result pointers during rehydration; Root has since synchronized REVIEW/CHANGES_REQUIRED across current canonical surfaces.
