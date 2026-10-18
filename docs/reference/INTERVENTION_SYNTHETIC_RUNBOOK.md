# Synthetic intervention operator runbook

Use an isolated installation and invented data. The commands below require the ordinary backend database/JWT/encryption configuration, server CLI access, and an existing tenant owner/admin UUID. `intervention-issue-grant` also needs the UUID of the authenticated actor who created the rerun request. These commands do not call a provider.

Set `INTERVENTION_MODE=synthetic`, `INTERVENTION_PROVIDER=openai`, `INTERVENTION_MODEL=<approved model>`, `INTERVENTION_ENDPOINT=https://api.openai.com/v1`, `INTERVENTION_API_KEY_FILE=<absolute path>`, `INTERVENTION_CONFIG_REF=<operator revision>`, `INTERVENTION_REQUEST_BYTES=6000`, `INTERVENTION_OUTPUT_TOKENS=128`, and `INTERVENTION_TIMEOUT_SECONDS=15`. For Alibaba, use `INTERVENTION_PROVIDER=alibaba_maas`, `INTERVENTION_MODEL=qwen3.7-flash`, and the approved Alibaba compatible endpoint accepted by the existing single-call helper. Keep the credential outside the repository. The loader reads at most 8,193 bytes and rejects an empty, oversized, or multiline key.

Run the backend CLI with the same configuration as the server:

```text
backend intervention-prepare-demo --tenant <tenant-uuid> --issuer <owner-or-admin-uuid>
backend intervention-set-budget --tenant <tenant-uuid> --issuer <owner-or-admin-uuid> --job <job-uuid> --cap 2
```

The first command returns the manual job and two conversation IDs. The second sets a total call cap, which can only be raised and never resets `consumed`. Start the server with the same configuration. An authenticated actor with job write permission runs `POST /api/v1/tenants/:tenantId/jobs/:jobId/intervention-run` with `{"conversation_id":"<wait-conversation-uuid>"}`. The empty conversation produces a WAIT_DATA case without a provider call. Record its case version, original snapshot ID, and digest from the case detail.

Add the fixed invented source once:

```text
backend intervention-complete-demo-data --tenant <tenant-uuid> --issuer <owner-or-admin-uuid>
```

The command is idempotent and refuses an unexpected existing message. The authenticated actor posts a data revision to `/api/v1/tenants/:tenantId/intervention-cases/:caseId/data-revisions` using `expected_version`, `expected_snapshot_id`, `expected_snapshot_digest`, and a `reason`. Then the actor posts to `/api/v1/tenants/:tenantId/intervention-cases/:caseId/rerun-requests` with the updated case version, the original snapshot identity, returned `data_revision_id` and `data_revision_digest`, `reason`, and a unique `request_key`. Use the response's request UUID and current case version.

The owner/admin issues a narrowly bound grant:

```text
backend intervention-issue-grant --tenant <tenant-uuid> --issuer <owner-or-admin-uuid> --actor <request-actor-uuid> --case <case-uuid> --request <request-uuid> --evidence <approval-reference> --deadline <RFC3339-UTC-within-24-hours>
```

The actor executes `POST /api/v1/tenants/:tenantId/intervention-cases/:caseId/rerun-requests/:requestId/execute` with exactly `{"expected_case_version":<current-version>,"grant_id":"<grant-uuid>"}`. This is the first step that may consume budget and call the configured provider. Do not run it during software-only verification. A stale source, revoked membership, changed configuration, expired grant, or exhausted cap blocks a new call. An uncertain outcome retains its consumed unit. Review execution, attempt, receipt, and case state before any subsequent action.

The complaint conversation can be used for a separate manual initial run; that run may call the provider immediately and requires its own available cap. Keep the server disabled by default outside the intended synthetic exercise. Never use these commands on a real customer source.
