# R097 pure local TLS tooling proof

Status: ACCEPTED_INTENDED_CONTRACT
Risk: R1

One prerequisite for M2 live proposal evidence: verify the corrected TLS harness regression against a loopback TLS server offering HTTP/2. Run exactly one Go invocation from project root, GOPROXY=off/GOSUMDB=off, selector ^TestR096TransportTLSOffline$, -work -count=1 -v ./api/handlers. No source edits, provider, secret, DB/fixture, app, download or publication effects. Source SHA is immutable authority-pinned; old R096 counters are unchanged.

Acceptance: selector really executes (no skip); process exit0 and named test PASS; actual source hash matches seed; retain stdout/stderr and WORK binary hash. Root independently checks source/caps, exact executed membership, original failed assertion and previous independent before/after TLS protocol probe. This is local software protocol proof only, no AI governance, actual Alibaba response, M2 product or pilot acceptance. Failure stops without replay or automatic successor.
