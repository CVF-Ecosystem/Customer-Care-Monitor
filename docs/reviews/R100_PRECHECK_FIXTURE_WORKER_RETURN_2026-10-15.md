# R100 precheck and owned fixture worker return

Status: FIXTURE_SEEDED_LIVE_NOT_RUN
Risk: R2
Worker: /root/r100_worker
Reviewer: Codex /root
Reviewed source: `a5eb32d8ae5ce8abe3413cafcba3fc620543689d`
Root source/binary admission: `b72762723f4c9779d67c02a2608ed0e48b0dc25e`

The Root-signed binary remained SHA-256 `a111c0b62394ea5497bae2e272151d5fc195d8a4dca3787cc152970b09817f7d`. The admitted `python -B scripts/r100_intervention_live_fixture.py precheck` exited 0; its retained stdout has only fixed schema/key-shape/endpoint-pin booleans, requested model `qwen3.7-flash`, and model availability `UNKNOWN`; stderr is empty. The provider credential was parsed in child memory only and is absent from these receipts. Exact safe command/cwd/exit/output: `docs/reviews/probes/r100_root/worker_precheck/`.

The admitted `python -B scripts/r100_intervention_live_fixture.py setup` exited 0 and printed only `R100_FIXTURE_SEEDED`; stderr is empty. It created one owned cached `mysql:8.0` loopback fixture, then launched the signed Go binary once in seed mode. The fixture has four distinct predeclared synthetic complaint conversations. Read-only post-setup ownership verification matched the fixture manifest, seed binary SHA, image/network/volume pin, and all three protected existing container IDs. Exact command/cwd/exit/output and the synthetic ownership/fixture summary: `docs/reviews/probes/r100_root/worker_setup/`.

Owned container ID `88c14ee97ffd41d8ab61da3679e2c6b97cc15033c16c35a1e246e0cbe298385f`; fixture manifest SHA-256 `1e16fb10672a5edd6893ad2580850340dcc465829e48004d6de674b910affdb8`; ownership SHA-256 `1e7f804765978e9e408c1f8aeb02a98f52863a1c9cc7ea7815701ff8d05a6c47`; pin SHA-256 `a050b3781fd7bcb7fa9afcecb70508c4419bdf86ab1b864b332bcdc4961b41ac`. The fixture binds only to loopback port 64944. `seed.go-start` exists; `worker.go-start` and all four physical slot markers do not. Aggregate R100 Go/binary launches used 2/4; provider calls used 0/4.

This is fixture admission evidence, not a current-source live proposal or governance claim. No live binary, real provider request, existing app/DB change, or old marker reset occurred. Root must independently inspect the ownership/pins and admit the bounded live process separately; the worker stops here.
