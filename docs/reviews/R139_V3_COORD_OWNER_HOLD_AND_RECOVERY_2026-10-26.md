# R139 v3 preclock HOLD and consolidated recovery decision

Status: CHANGES_REQUIRED
Risk: R2
Worker: Root technical metadata author; Sol Medium packet worker
Reviewer: Sol Medium `/root/r139_final_technical_review` independently audited original HOLD and remaining static preclock conditions
Date: 2026-10-26

Original controller PID23500 returned exit1 at COORD_OWNER before reservation. Root wrapper PID20572 observed the original wait/exit. Shared future controller use is 1/1; campaign remains 0/1. Service queries, CUA, fixture starts, key reads and provider attempts are all zero. No retry is released. Product B remains BLOCKED_PENDING_OPERATIONAL_ADMISSION; M3-05/M3-08, G0–G3 and runtime governance acceptance remain open.

## Actual cause and review limitation

Dispatch632b7ef6931a2c772e5c104009dc43fafc1bdd92 correctly pinned the attempted coordination owner hash85915def7220357e2c0da904c7c223ee705a3ea00a95ac32b547ab85b388429a. The actual ignored JSON lacks required freshGrantSha256 and rootExclusive:true, and retains superseded source b330/build3366. Root and independent static review checked its hash but missed its actual required fields. This is a packet preparation/review defect, not a browser or provider failure. The earlier technical REVIEW_PASS remains historical at its exact bytes and is superseded for execution by this actual failure.

All30 child waits returned; optional git config core.hooksPath absence returned1 as accepted optional metadata. The boot-query child026 returned0 and the controller reached the owner guard after validating boot identity. This is control-path evidence, not a public disclosure of raw host metadata or a provider governance receipt. Original raw query output stays private; public evidence stores hashes only.

Independent failure audit found the existing extracted controller, custody and child-log root are occupied and cannot be reused. Before HOLD-state synchronization the eight canonical old hashes/templates,42 protected refs,13 source pins,binary/Vite pins and historical section matched. Catalog/gate/render operations inside the controller were NOT_RUN; the earlier repository precommit7/profile6 PASS does not imply campaign setup PASS.

## One consolidated recovery proposal — no execution authority

Keep the same source7ef30823746d72e935607c0bebf6ab306099eb48/buildad348196 and16 counted source checks. Do not build/test Go or repeat settled Python tests. Preserve all failed receipts, previous immutable admissions and original controller23500/exit1. Owner decision requested: ONE additional original preclock invocation, cumulative shared controller cap1→2 with used1 retained; campaign cap1 and every external cap remain unchanged. The new controller contains at most its one original current boot query; no separate OS discovery or live precheck is proposed.

The corrected owner proposal is concrete: required freshGrantSha256/rootExclusive and exact current source/build/nativeContractId/nonce are supplied in the referenced JSON. New nonce: 4f37f37b37fa5295611c84b63d1d9910. Four fresh proposed roots in the decision JSON were read-only verified absent. Root must validate the actual generated owner schema/full identity and hash BEFORE extraction, original controller invocation or boot query. The existing source owner guard only checks required grant/exclusivity/hash; source/build/nonce equality must therefore be explicitly verified against actual bytes by Root and independent reviewer, without asserting a machine guarantee that is absent.

After a new Owner decision, bind a separate additional-slot supplement in a new Root wrapper, preserve the old process receipt, use exclusive controller/log/coordination/recorder paths, refresh actual worker ACKs and every dependent reference, regenerate all eight current canonical baseline hashes/templates after this HOLD, independently review the complete actual packet, run the fixed e9/profile1.1 repository gates and make one clean normal exact-byte commit. Only then invoke once, retain original Popen/wait/exit, and condition five exact queries and native G0 on the actual reservation and Root signal. Actual G0 review plus separate Runtime Effect Admission still precede fixture/key/provider.

This proposal adds no helper tranche, Antigravity, service allowance, page, API attempt, USD, CUA method, inventory, retry, campaign duration or cleanup time. Original whole1200/+1020cleanup/setup180, queries5, CUA64 stages8/4/48/4, one owned tab and provider1≤USD1 remain. No Core fetch/update/adoption, push, deployment or production effect is authorized. Facebook/Zalo OA remains parked.

Evidence: `docs/reviews/probes/r139_integrated_20261021/native_operational_20261024/v3_coord_owner_hold_20261026/root_hold_receipt.json`, original `started.json`, `original_wait_exit.json`, `intent.json`, `coordination_owner_as_attempted.json`; concrete fix `corrected_owner_proposal.json`; decision `consolidated_recovery_decision.json`. These are original process metadata and Root-authored proposals, not native browser/provider transcripts.
