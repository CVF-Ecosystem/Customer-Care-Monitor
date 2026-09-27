# S1 job-dispatch configuration admission

**Tranche:** `CCMAI-RUNTIME-009` · **Phase:** SPEC · **Risk:** R2 · **Entry:** R001–R008 independent REVIEW PASS / FREEZE open; S1 IN_PROGRESS.

## Source finding and decision

`TestRunJob` and `TriggerJob` first resolve a job by ID and tenant, then launch a goroutine and return HTTP 202. Each goroutine calls `cfg, _ := config.Load()` and passes the possibly nil configuration to `engine.NewAnalyzer`. An invalid runtime configuration can therefore be acknowledged as a started job without a usable worker. Apply the reviewed R008 admission pattern to these two job endpoints, bounded to dispatch truth.

## Contract

1. Preserve the existing tenant-scoped job lookup and 404 response. After a job is found, load and validate configuration before launching any goroutine or returning 202. On load error or nil configuration, return a generic non-2xx `job_start_failed` response. Do not create a job run, store a cancel function, start analysis, or expose configuration values/errors in the response or logs.
2. On valid configuration, pass that exact `*config.Config` to the launched worker. The worker must not reload configuration. Keep existing successful 202 bodies (`test_run_started` and `job_triggered`), asynchronous execution, mode/parameter semantics, cancellation registration and normal analyzer behavior.
3. A missing or other-tenant job must remain 404 and must not attempt configuration load. This contract does not add single-flight or idempotency, change run lifecycle semantics, or claim successful analysis from HTTP 202.
4. Agent-run, scheduler, channel OAuth/credential handlers, real provider calls, customer data and CVF runtime governance are outside R009. They remain separate S1 or later work.

## Acceptance

- Focused handler tests for both endpoints prove: forced config error and nil config return generic non-2xx without a worker/cancel function/job run; the wrong-tenant path returns 404 without loading config; valid config is passed by identity to one launch and retains the endpoint's 202 body. Verify trigger mode/parameters and test-run limit remain unchanged at launch.
- Exercise real `config.Load` validation with synthetic temporary environment values. A deterministic loader/launcher seam may be private to `jobs.go`; restore package globals after each test and do not use parallel tests with those seams. No real analyzer, adapter or provider is launched by endpoint tests.
- Run focused and full backend tests with a disposable MySQL `CCMA` database, build, vet, catalog check, workspace doctor and diff check. Record exact commands, cleanup and limitations. Synthetic checks prove handler admission only and are not live governance evidence.
