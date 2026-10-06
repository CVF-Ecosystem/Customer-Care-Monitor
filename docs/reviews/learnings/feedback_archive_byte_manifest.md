# Shared learning — compare extraction with archive member bytes

Date: 2026-10-06. LOCAL_GUIDANCE_RECORDED / UPSTREAM_ASSESSMENT_DEFERRED. Source: [R052 independent review](../CCMAI_RUNTIME_052_INDEPENDENT_REVIEW_2026-10-06.md), exact failed evidence17d60560 and handbackd9e8e8f. Independent archive and protected-file audits confirm this local observation.

R3 runner compared an exact Git archive extraction with raw Git blobs. On this Windows Git configuration, three non-Go files exported with CRLF while their blobs had LF; archive SHA and all204 extracted members were correct. The strict comparator failed before any runtime/resource/Go command. Syntax validation alone had not checked the manifest representation contract.

Use sorted path/hash/size records derived from exact archive members to verify extraction membership and bytes, retaining raw blob differences as separate diagnostics. Bind archive SHA to actual Git-resolved source and configuration-observed output. Never silently normalize runtime source, equate normalized equality with byte equality, weaken membership checks, or change Git configuration to manufacture an old archive identity. Exercise pure finite manifest checks before expensive runtime commands. Preserve the failed attempt and require separately recorded cost disposition before retry when the work order says stop.

Applied in successor R053 planning; worker implementation still pending. Existing R051 RP-04 acceptance remains open; no application/runtime/governance acceptance or parent adoption follows from this lesson. Proposed reusable parent guidance is deferred, with no parent edit or external transmission.
