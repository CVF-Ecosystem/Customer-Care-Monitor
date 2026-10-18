# R120 source review round 1

Status: CHANGES_REQUIRED
Risk: R2
Worker: /root/r120_ui_readiness_luna
Reviewer: Codex /root
Scope: SOURCE_ONLY_NO_RUNTIME

Root independently reviewed the complete identity-normalized delta: Go harness and Vite template match accepted R119 structure exactly after identity rename; no product source change. Greeting accumulation/retry, partial timeout/read errors, pin rechecks/canonical PowerShell, raw listener query, exact Docker Pid0/Ports{}, held child cleanup and reserve180 reviewed. Worker18unitPASS and inspectPASS2of2; Root18unitPASS1of2/exit0, four hashes unchanged/runtime absent. Runtime not admitted.

Consolidated same-scope repair1of2 before compile: (F1) read_mysql_greeting remaining<=0 between reads raises plain TimeoutError, losing an already received byte count to outer zero. Raise typed MySQLGreetingTimeout(len(header)) on that branch and add a direct offline discriminator for partial prefix followed by expired deadline, preserving count2. Do not modify other behavior. (F2) worker return must explicitly include Risk:R2, Worker and independent Reviewer fields so role-bearing packet classification is checked; preserve original2check receipts and honest Root budget update.

Worker checks2/2 exhausted: no new worker check/inspect. Root has one remaining Python check for repaired19case suite. Root remains independent author of review/authority/commit only. No Go/runtime/provider/start spent, no source change outside four authorized files and worker return. R119 immutable and all historical PASS/failure receipts retained; no FREEZE or product claim.
