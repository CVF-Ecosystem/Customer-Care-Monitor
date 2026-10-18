# Test DB pool lifetime and MCP pending attempt fixture

Status: REVIEWER_ACCEPTED
Risk: R2
Worker: Sol engine test DB pool lifecycle; Luna MCP pending notification assertion
Reviewer: Root independent source and terminal evidence review

Actual R125 complete full suite exited1: engine exhausted MySQL151 connections because the shared test connector overwrites pools without closing its predecessor, causing DB-unavailable skips and the new integration unable to connect; MCP retains a stale no-log-before-send expectation. Repair only the shared engine test connector lifetime and analogous MCP sending-attempt assertions. Keep production connection code and max_connections unchanged. Respect fixture teardown ordering.

Root independently reviews exact changes. Run complete engine and MCP packages with fresh synthetic MySQL, count1/serial/30m. Other packages passed in the complete R125 run and their source is unchanged; source hash delta must prove only two test files changed. Current whole-package coverage may combine complete package event streams with explicit provenance, never discard failures inside a selected package or relabel the old full invocation PASS. DB sentinel gate must reject any current DB-unavailable skip. Do not repeat already passed unchanged packages without a new concern.

R125-CI one-line workflow timeout alignment carries its prior authority and remains uncommitted at this seed boundary; no additional workflow expansion. No provider/real credential/customer source/installed app mutation, push or deployment. Review: docs/reviews/R126_FINAL_CURRENT_PACKAGE_REVIEW_2026-10-18.md.
