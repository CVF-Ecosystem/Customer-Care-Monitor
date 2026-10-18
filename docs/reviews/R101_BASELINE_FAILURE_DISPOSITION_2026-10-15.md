# R101 legacy frontend failure disposition

Status: REVIEWER_ACCEPTED
Risk: R2
Reviewer: Codex /root
Worker: /root/m3_design (Sol Medium)

Actual candidate offline compile and focused M3 frontend checks passed. Full frontend suite failed3/441, all in the historical dashboard-service-status test. Root performed exactly one isolated prior-source replay using git archive4424e9d (before M3 source), cached modules only, no installs/downloads/source edits. The same three cases failed: missing deferred Dashboard request, absent4242 metrics, and unrecognized /tenants/t1/me. Dashboard, auth-store and that test bytes match the baseline exactly. Raw/structured proof: probes/r101_root/baseline_setup.json, baseline_result.json and baseline_dashboard.log. This establishes the failure predates R101; it does not turn the default full suite green.

The historical UI test supplies a demo-shaped answer to the existing permission-bootstrap request, so Dashboard never receives read authority and correctly does not request its data. Root disposition: continue the already-authorized M3 build/native/browser acceptance route with source200df3e frozen, recording this inherited test-contract blocker. No Dashboard source or test edits fit the R101 seed; no scope expansion or automatic supporting successor occurs here. The failure is not spent as source repair round2 because no R101 source repair is indicated.

Node verification counters: worker focused1 + full1, Root isolated baseline1 =3 of6; build0 of1. Go compile1 of16 (worker1/8); fixture0 of2; native/browser NOT_RUN; API/mock0. Continue one cached build, ownership-safe fixture, six exact native cases, then real-router/Vite browser proof. Stop on new actual contract/security/ownership/source/protected-state failure. Preserve the failed full-suite and baseline logs; no replay to conceal them.

R101 cannot be declared CLOSED/FROZEN while required tests fail. Final independent review may accept only evidenced local human capability with FREEZE_OPEN and a finite explicit disposition for the3 historical UI fixture failures. Full M2/M3/WAIT_DATA rerun, M1 business/pilot/release remain OPEN; accounts parked. No provider, installed activation, push, merge or deploy.
