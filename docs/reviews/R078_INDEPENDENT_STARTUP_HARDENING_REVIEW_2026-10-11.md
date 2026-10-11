# R078 independent startup hardening review

Status: REVIEW_PASS; exact-source synthetic local startup contract accepted.

Independent root raw proof is [summary](probes/r078_independent_summary.json).
Build PASS62.083s; positive PASS12 top-level/16 events18.842s; M01 expected
exit1 with SH02 fail/SH04 PASS8.859s; restored PASS12/16 in14.010s. No skip,
timeout, build fault or unrelated failure. Each role used exactly4 Go calls,
aggregate8/8 and retry0. Each packet has20 original raw references verified
by length/SHA256 and TEMP backup. All8 full source manifest checkpoints per
role match; every owned5 containers/2 volumes/internal network is absent and
the guarded source TEMP snapshot was removed. Protected369 source and114
old authority/tranche artifacts remain byte-identical.

Root is independent of Luna max implementation/repair authorship. Scope is the
new scheduler binary on an isolated synthetic MySQL fixture. The first source
`6219da376c63d806ce412bfbae2a738a0af529bc` and first runner packet `52837bf`
remain in Git history. Repair source is
`b7c826aa9e412b9e463716d6449b99a78819bfa9`, bound backend archive SHA256
`3f135def15d46a5a3729a5718558af882715b69e103ffc6e4db5f536136abd84`.

Product review: `Scheduler.Start` removes the unconditional JobRun cleanup
call and its now-unused helper. Sync lease recovery, cron loading, scheduler
start/stop and normal worker finalization remain. No schema, ownership fence,
status/checkpoint, provider/channel, dependency or other product changes.
Unknown running rows continue blocking admission; this tranche supplies no
recovery bypass or evidence that an owner died.

The [first static findings](R078_FIRST_STATIC_FINDINGS_2026-10-11.md) were
CHANGES_REQUIRED before any Go invocation. F1 required unrelated-job and
different-tenant row snapshots and unmasked cancellation-context assertions.
F2 required an anchored old-call/helper mutation, cumulative exact source/tree
binding, guarded owned TEMP cleanup and fail-closed typed/hash-checked summary
publication. One worker repair addressed both. Root independently replayed
the pure mutation and receipt rejection contracts before runtime release.
No product semantic repair was required; original source/runner runtime was
NOT_RUN and must not be retrospectively called PASS.

SH01 exercises actual Start while coordinator A holds a real reserved worker
behind a synthetic provider barrier; fresh coordinator B stays persisted-busy
and row bytes remain stable. SH02 checks four full row snapshots under an
attempted-update trap. SH03 checks free-job admission, rejected cancellation
without context signaling, exact owner signaling and ordinary finalization
after release. SH04 preserves eligible expired lease recovery and the
ineligible channel/checkpoint/credential/no-work controls. SH05 verifies
unknown persisted ownership and a reservation query fault fail closed without
leaking rows/owners. SH06 injects cron-load query failure and observes the
underlying scheduler still start. SH07 loads a controlled future cron without
provider/notification work and bounds shutdown. Fixture tenant rows and global
callbacks/factories are restored; tests do not run unsafe globals in parallel.

Worker raw proof is [summary](probes/r078_worker_summary.json). Root audited
12 top-level/16 PASS events in each positive/restored command, zero skips,
M01 named SH02 attempted-update failure with SH04 healthy PASS, exact full
source restoration and exact-name cleanup. Worker used one build plus three
tests, zero retries. The root campaign reuses the reviewed worker harness with
separate reviewer attribution, internal network, containers, volumes, TEMP
snapshot and raw evidence; independence is execution/source evaluation, not
independent harness authorship.

The read-only raw auditor `docs/reviews/probes/r078_root_evidence_audit.py`
checks events against the actual JSONL, every published raw length/hash and
original TEMP backup, archive identity, cache image IDs, no host ports,
read-only source/modules, restoration and owned cleanup. It executes no Go
or Docker command. Raw hashes describe original capture bytes; Git's line
ending conversion is a separate representation and must not be confused with
capture bytes.

Authority seed `0579a792c473e53eb11c6efa007ab9d539370dec` was committed before
activation/BUILD, authored by root dispatcher; worker did not edit it. Root
owns commits and mechanically bound repaired source/archive pointers. Scoped
doctor is PASS WITH NOTE25/1: actual public core `e9ddcbcdcac2a9df520f5eb0761dbdb609b9deba`
matches cached origin/main and is clean; old manifest pin/local binding and
missing gate-profile/compact bootstrap migration notes remain explicit.

This is synthetic application-contract evidence. No real provider call,
runtime CVF governance, hosted CI, deployment, mixed-binary/fleet safety,
durable recovery or full-roadmap completion is established. Facebook/Zalo OA
account/live setup remains parked; no push/merge/deploy authority is used.

Repository publication checks: first final preflight/cached PR range6/7 failed
on root-authored descriptive disposition, corrected to exact REVIEW_PASS enum
with scope in a separate field before commit. This is root metadata overhead,
not a worker source defect or runtime failure. Gate units46PASS11.867s from
preparation are inherited unchanged (scripts/tests unchanged); PS5 regeneration
and PS7 catalog check PASS, four local report links verified. No docs-site build
or hosted CI claim. Final preflight/PR-range must pass before review commit.
