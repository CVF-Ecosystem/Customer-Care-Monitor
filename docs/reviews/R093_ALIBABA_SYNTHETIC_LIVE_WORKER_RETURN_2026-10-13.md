# CCMAI-RUNTIME-093 Alibaba synthetic live worker return

Record type: worker return  
Status: COMPLETE_PENDING_REVIEW  
Risk: R2  
Worker: /root/m1_worker, Luna max, IMPLEMENTATION_WORKER  
Reviewer: Codex /root, independent reviewer and commit steward  
Phase: WORKER_RETURN  
Immutable seed: 7639e463bb18cb27ff561e01ccf3768fad33da89  
Dispatch material: 03e95ce3a09abd98d8605f40a5e0c4ad0eecf4d6  
Source SHA-256: 41c6eb00b31a86294508954eeeb7e01ce61fb7f2c0535380580dac57ec5415fa

## Returned result

The exact-source approved probe made one worker request and the independent reviewer made the remaining one request. Both used synthetic inputs and returned HTTP 200 from the pinned Alibaba-compatible endpoint. The campaign cap is exhausted: two physical requests total, one per actor, zero retries, no fallback, and no further provider request is authorized under R093.

The worker receipt is docs/reviews/probes/r093_alibaba_live/worker.json (SHA-256 f01029155a1bd61e638d9516ae3ce9c0dc5c74cd2b7148794ecd14a9d4af549b). It records OUTPUT_CHECKS_FAIL after 5.141 seconds. Returned model was qwen3.7-flash; returned provider provenance is UNKNOWN. Reported usage is KNOWN_REPORTED: 320 prompt tokens + 311 completion tokens = 631 total. Response shape, model, schema, and synthetic rule checks passed; message-anchored evidence failed with complaint_evidence_reference_invalid_or_missing. The sanitized response body was retained, and the credential-content detector found none.

The independent reviewer receipt is docs/reviews/probes/r093_alibaba_live/reviewer.json (SHA-256 d5b48649295af098fa99bbd05f46057c9e682b774e9c40b326a01555cf35738f). It records OUTPUT_CHECKS_PASS after 2.422 seconds for the simple synthetic conversation. Returned model was qwen3.7-flash; returned provider provenance is UNKNOWN. Reported usage is KNOWN_REPORTED: 334 prompt tokens + 96 completion tokens = 430 total. These two examples are not a quality or calibration benchmark and the reviewer example does not replace the worker evidence failure.

Billing amount and quota remain UNKNOWN; token usage is provider-reported receipt data, not billing evidence. Before source approval, one offline self-check attempt failed because its synthetic endpoint fixture used the live endpoint hash; the fixture was corrected to inject its own hash, and the final 11-group zero-call self-check passed. No secret value was printed, committed, copied, or persisted by the worker. No source edit or commit followed Root's approval.

## Scope and limits

This return records exact HTTP wire observations and the probe's synthetic output checks only. It does not establish Go adapter or Analyzer integration, CVF governance behavior, customer/channel-data handling, provider-routing governance, audit behavior, production safety, model quality, billing, or completion of M1, M2, or M3. No application source, database, frontend, real channel, account, customer data, outbound notification, deployment, commit, or push effect occurred.

The worker did not self-review. Root owns independent source/receipt disposition and commit. Real-channel and pilot choices, business SLAs, assignees, quality thresholds, and financial ceiling remain open outside this bounded R093 return.

## Return-Time Closeability Recheck

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0  
closeabilityDisposition: CLOSEABLE  
outsideAuthorityBlockers: NONE  
nextRepairRoute: ROOT_INDEPENDENT_REVIEW_THEN_SEPARATE_M2_ANCHOR_VALIDATION_SCOPE  
workerRedispatchAllowed: NO_ALL_R093_ACTOR_SLOTS_CONSUMED

This return is ready for Root's independent review. Both actor slots are consumed; any further provider test requires new authority. No worker commit or global continuity synchronization was performed.
