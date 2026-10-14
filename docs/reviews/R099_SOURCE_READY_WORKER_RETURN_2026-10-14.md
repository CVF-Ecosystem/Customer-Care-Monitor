# R099 source-only worker return

Status: COMPLETE_PENDING_REVIEW
Risk: R2
Worker: /root/m2_worker (Sol Medium, IMPLEMENTATION_WORKER)
Reviewer: Codex /root (independent REVIEWER)
Tranche: CCMAI-RUNTIME-099

The candidate changes only `backend/ai/openai_compatible.go`, new `backend/ai/chat_once_response.go`, and `backend/engine/intervention.go`. Following Root's independent source admission and commit `0640431`, the one authorized offline Go build of `./ai ./engine ./api/handlers` passed with exit 0. It ran from the project root with `go -C backend`, `GOPROXY=off`, and `GOSUMDB=off`; stdout and stderr were empty. The raw command, cwd, streams, exit, and exact source hashes are in `docs/reviews/probes/r099_source/`. Worker Go budget is 1/1 exhausted. No test, provider call, credential read, database, fixture, runtime launch, or old R098 evidence change occurred. The R098 two actual calls and four Go invocations remain exhausted. Existing installed application and database were untouched. `git diff --check` passed; changed-source runtime behavior remains **NOT_RUN**.

## Source identity

| Path | SHA-256 |
| --- | --- |
| `backend/ai/openai_compatible.go` | `a1ac47fcba0e813d139b2ab4dcd47d872ac5614b9d5528c805cd007df34e99fd` |
| `backend/ai/chat_once_response.go` | `7a809aea477017ee6205a47803f4caac68064fa610e7b7dc7679cf236734246a` |
| `backend/engine/intervention.go` | `59bf11d0e74e6f0cb39ab52a006a32c5dd9de26be320680b9a2d87b577b5ecf3` |

## Consolidated source audit

For a one-call bounded HTTP 200 body, top-level JSON and a unique nonempty model string are checked before inspecting choices. A missing, duplicate, or mismatched model remains untrusted. An exact matching model creates response metadata with provider and wire usage presence before the choice decode and semantic checks. Missing/malformed choices, refusal, provider error field, empty content, and any finish reason other than `stop` yield a private, constant-text observed-response error and no content. No refusal, provider body, transcript, URL, key, or header enters that error. Missing/invalid usage retains absent/invalid presence and therefore UNKNOWN rather than zero. Transport, non-200, read failure, oversized/unparsed body, and unknown/mismatched model keep the existing unknown branch.

Known wire token counts fill the legacy flat integer fields before semantic choice decoding only when safely representable as `int`. UsagePresence remains authoritative; absent, invalid, or unrepresentable counts never become a claim of known zero. The engine recognizes the private observed-response error only with nonempty model/provider and usage-presence metadata. It persists the physical response and its known/partial/unknown usage before terminal `OUTPUT_INVALID`, with no proposal. Settlement failure follows existing `SETTLEMENT_FAILED` handling; other errors remain `PROVIDER_UNKNOWN`. Existing cap admission, source snapshot, cancellation, and atomic publication code was not changed.

The explicit `WithAlibabaQwen37OnceThinkingDisabled` option returns a cloned provider only for model `qwen3.7-flash`, provider kind `openai`, bounded completion tokens 1–768, and an exact HTTPS Singapore Alibaba `compatible-mode/v1` base URL with valid nonempty DNS workspace labels and without userinfo, query, fragment, unsafe port, or encoded path. `AnalyzeChatOnce` rechecks it before request construction and adds only top-level `enable_thinking:false` when opted in. Constructors, default request body, `AnalyzeChat`, and batch request behavior remain unchanged. No caller has been modified to opt in; future use requires separate source-bound live authority.

Root's first consolidated precompile source-admission feedback identified incoherent flat counters on malformed choices and an overly permissive endpoint prefix. Both were corrected in this same BUILD draft; no Go invocation or effect followed that feedback.

These are source and compile claims only. R098's HTTP 200, empty-answer receipts explain the repair but do not prove the changed source. No current runtime AI governance, valid proposal, or full M2 outcome is claimed. Root retains independent final review and commit ownership; no further worker Go invocation or external effect is authorized.
