# R093 independent pre-effect source review

Status: SOURCE_APPROVED_FOR_BOUNDED_EFFECT
Risk: R2
Worker: /root/m1_worker
Reviewer: Codex /root

Root independently reviewed `scripts/ccma_alibaba_live_probe.py` at SHA-256
`41c6eb00b31a86294508954eeeb7e01ce61fb7f2c0535380580dac57ec5415fa`.
This decision admits only the existing two-slot synthetic Alibaba characterization,
not application integration, runtime CVF governance, billing or product acceptance.

The initial source at `be05bab1c5847f1e3f0453707d5eb7b09c2a09a4e9c78249e332e36e17004d6e`
was refused before credential/provider effects. Root reproduced a logically escaped
credential retained in a response and inconsistent usage reported as known. Source
review also found socket deadlines insufficient for stalled DNS/headers, missing
region pinning and raw CRLF/LF seed comparison. Initial source and findings remain in
`probes/r093_alibaba_live/pre_effect_source_be05.py.txt` and
`probes/r093_alibaba_live/pre_effect_initial_findings.json`.

One consolidated same-scope repair resolved these findings. Response inspection
decodes escaped and nested JSON strings; unsafe bodies are omitted. Usage totals
must reconcile. Region/model/endpoint are pinned; seed equality normalizes only
CRLF. A parent supervises exactly one child, drains its pipes, kills/reaps at the
absolute deadline and reports actual elapsed including cleanup. Key travels only
through in-memory stdin; child command/environment contain no key. Dispatch uses
exclusive actor/nonce markers before one HTTPS attempt, with no redirect or retry.
Child rechecks source approval, current actor/phase, committed seed and body pin.

Root ran 17 independent local checks, including encoded/nested/duplicate key cases,
usage knownness, region/model rejection, current committed seed, reviewer rejection
in BUILD, missing approval refusal, 350ms stalled-child kill/reap, 1.4MB pipe drain
and durable actor replay denial. Evidence:
`probes/r093_alibaba_live/independent_pre_effect_checks.json`.
Worker and Root each ran the 11-group zero-call self-check. These are local
safety/parser checks with constructed fixtures, not AI governance proof. Actual
provider requests and credential-file reads during these checks: zero.

Root may create the ignored approval binding this exact digest. Worker gets one
request in BUILD and reports immediately; Root then records REVIEW and independently
uses the sole reviewer slot within the five-minute campaign. Any mismatch, failure
or ambiguous dispatch consumes the slot; no reset, fallback or third call. Receipts
must retain actual request/response and known/unknown usage without secrets.

## Return-Time Closeability Recheck

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
outsideAuthorityBlockers: NONE
nextRepairRoute: SAME_SEED_WITHOUT_EXTRA_PROVIDER_REQUEST
workerRedispatchAllowed: YES_WITHIN_EXISTING_CAP

Source approval is conditional on static repository checks and exact-digest
reverification immediately before approval. No runtime outcome has been accepted.
