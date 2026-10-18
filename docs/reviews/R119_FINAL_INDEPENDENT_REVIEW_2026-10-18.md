# R119 final independent review and bounded handoff

Status: CHANGES_REQUIRED
Risk: R2
Worker: /root/r119_runtime_design; delegated repair /root/r119_repair_luna
Reviewer: Codex /root
Scope: SOURCE_COMPILE_ACCEPTED_UI_RUNTIME_NOT_RUN

The product outcome was not achieved: no current proposal/history UI DOM or screenshot exists from R119. The sole setup exited1 after68.457s with MYSQL_PROTOCOL_INVALID before backend/Vite/product tab. Original setup terminal stderr and raw Docker commands are preserved; no conversion of NOT_RUN to PASS. UI source R1187f74a75f24tests/typecheck and real synthetic R1171of1 evidence remain inherited, not replayed.

## Accepted evidence and limits

- Immutable Root seed7d944e25 preceded BUILD; Root independently reviewed both authors and committed source96ba23cc. Worker2Python checks and Root1Python check each8unitPASS; Root offline compile1of2PASS36.706s, no downloads. Exact399source/binary/Node/Docker pins committed7ea25d35 before setup. No source edits during setup; independently final raw source/binary hashes unchanged.
- New exact owned container714f2f0c85ee5e01b7219fa77fae35d6da40b8d2c89c178cd86a882ee4681d52 start1/stop1. Root independently observed EXITED and exclusive owned network/volume identity. Protected baseline and final projections equal:7containers,8networks,470volumes; protected existing app/nginx/db remain running. Resources preserved, no deletion/restart.
- Setup cleanup recorded OWNED_FIXTURE_LISTENER_STILL_OPEN. Independent final port56024 connectEx10035 did not connect, but did not provide refusal proof. Listener closure remains NOT_PROVEN; do not claim settled cleanup or fabricate a serve terminal receipt.
- No setup readiness receipt/resource admission/serve intent; backend0/Vite0/producttabs0/auth0/caseActions0/provider0. No fixture seed data or durable DB accounting was observed. Effect/fingerprint DB proof NOT_OBSERVED, UI NOT_RUN, B09 NOT_RUN_UNSUPPORTED; B07 inherited bounded and B08 raster qualification retained. Full M2/M3/M1business/pilot/release OPEN; FREEZE OPEN. No provider/governance claim.

## Findings and responsibility

F1: readiness code wait_mysql calls recv(5) once and permanently rejects a short/nonprotocol read. A TCP read need not fill the requested length. This is a source robustness finding; received bytes were not captured, so exact failure cause is UNKNOWN. Do not label this Docker outage or owner shutdown. No production code defect is established.

F2: listener-close predicate accepted only its selected refusal code; actual Docker/Windows connectEx10035 does not meet it. Stopped Docker resource and non-success connection are evidence, not proof of the predicate. No retry or alternate probe stack used to force a PASS.

Root's admission review missed readiness assumptions despite8unitPASS. The unit suite did not discriminate fragmented/early MySQL greeting or this Windows port result. This is an independent-review/readiness miss in addition to helper implementation gaps; not attributed solely to Luna. Earlier source-only acceptance is historical, now insufficient for runtime readiness. One same-scope source repair round used; no new source edit after runtime failure.

## Finite next route and stop

Stop runtime now; sole R119 fixture start and final stop are consumed, old seeds/counters unchanged. Raw packet: docs/reviews/probes/r119_root/failed_setup_raw.zip, SHA/file list failed_setup_manifest.json; direct final observation final_observation/receipt.json; setup_001 actual exit/stdout/stderr. Reviewed source and exact binary admission are retained for transfer, not restart authority.

Next governed decision has three bounded steps: read-only failed-packet/root-cause review; only if a supported, finite readiness fix is justified, prepare explicit new start authority with source/test/admission discriminators before execution; then target the same missing current-proposal/history UI observation and independent terminal review. No automatic successor/tooling chain is dispatched here. No provider key/call is needed for this UI objective. Existing installed services remain outside scope; no push/merge/deploy.

Closeability recheck: A1/A2/A4 source and offline compile bounded accepted; A3 setup readiness failed; A5 listener/terminal completion incomplete; A6 unit evidence accepted with coverage limitations. Runtime/browser acceptance NOT_MET. CHANGES_REQUIRED, not closed.

Documentation synchronization correction: Root caught a local loop-variable path shadow that temporarily wrote registry JSON into readiness refresh stdout.bin instead of the registry. Restored exact original bytes from the already committed pre_runtime_raw.zip after verifying archive SHA; wrote the intended registry to its canonical path. Original raw restored SHA c007c7e1712540ee77c9ce8e454890d905ff0193ec24907cb08ffd9d00b25d98. No source/runtime change or lost original evidence; staged diff must exclude the restored raw path.
