# CCMAI-RUNTIME-072 bounded work order

Status: DISPATCH_READY

Risk ceiling: R1. Implementation/repair/source commit steward: Codex /root/r072_worker (fresh gpt-6-luna xhigh). Independent reviewer/closer and metadata commit steward: Codex /root. Owner next/continue and explicit orchestrator/reviewer delegation authorize this local compatibility repair; root reviewed the concrete contract under existing direction. Immutable seed committed at 1fb7f6cd11a14d5c8ddd05942ac8bac75d904663.

SPEC: [UC01..07](../reviews/R072_LEGACY_UI_EXTENSION_COMPATIBILITY_SPEC_2026-10-10.md). Allow only the optional adapter_usage_presence usage key; ignore all contents and expose none. Preserve every legacy check and section output. Source paths: frontend/src/views/Jobs/job-detail/run-observation.ts and NEW frontend/src/__tests__/run-observation-adapter-compat.spec.ts. Existing tests and all backend files immutable; source baseline recorded before BUILD.

Worker first rehydrates state/memory/handoff/status/index and acknowledges BUILD in handoff/record before source edits. Commit product/new test before first runtime and return exact commit/test plan to root static review. Root never writes source/tests. Then each role independently gets at most2 Vitest calls (one expected positive focused three existing observation specs + NEW spec; reserve only accepted repair) and one forced vue-tsc check. Max aggregate4 Vitest/2 typecheck, Go0; no automatic retries. Capture raw stdout/stderr/command/exit/test counts and source hash. Failure stops pending consolidated root review. Review code/fixtures and run own tests before scoped local closure. No provider/network/DB/credentials/backend/UI display/dependency/gate/core/push changes; no governance claim.

Track first source quality, test reasoning and accepted repairs separately from root planning/metadata/harness overhead; Sol comparison remains unmatched and no model time/cost is available. Full roadmap, live, realAnalyzer and presence display/billing remain OPEN. Facebook/Zalo OA parked; prior R068 publication remains UNVERIFIED/not retried.
