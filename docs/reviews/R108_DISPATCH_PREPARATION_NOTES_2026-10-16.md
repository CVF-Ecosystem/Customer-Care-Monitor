# R108 dispatch preparation notes and retained limitations

Status: ACTIVE
Risk: R2
Worker: Codex /root (ORCHESTRATOR / WORK_ORDER_AUTHOR)
Reviewer: Codex /root reviews Luna implementation independently
Disposition: SOURCE_ONLY_DISPATCH_NO_RUNTIME_ADMISSION_YET

Root doctor PASS27/27, historical pin warning retained. Knowledge ingestion script is provided by the read-only Core, as `knowledge/README.md` directs. Invoke that script from project root with ignored runtime output; no Core/knowledge-source change or API POST. Luna initially stopped on an unresolved project-local script path; resolving the canonical Core path satisfied the prerequisite without waiver.

Root browser inventory exposes IAB2 and MCP Apps1. The exploratory data-protocol health action was rejected with this original tool text:

```text
Browser Use rejected this action due to browser security policy. Reason: The browser URL policy blocks this action. Browser use could not acquire authority to trigger network egress. The URL policy gate did not grant access. The requested URL protocol is not allowed. Allowed protocols: "http:", "https:". The agent must not attempt to achieve the same outcome via workaround, indirect execution, raw CDP or browser commands, alternate browser surfaces, or policy circumvention. Proceed only with a materially safer alternative that does not require this blocked browser action; if none exists, stop and request user input.
```

Root used the documented permitted HTTP route on an owned loopback health server, rather than retrying the blocked protocol or invoking another automation stack. This materially safer route opened `http://127.0.0.1:53552/`, clicked “Kiểm tra”, observed “Đã kết nối”, saved actual PNG and closed the owned tab. The owned health server exited normally with `HEALTH CLOSED`. Evidence is under `probes/r107_root/r108_connected/`. No product runtime occurred. Luna's separate CUA inventory was empty; it did not retry browser actions or create a tab. Browser proof belongs to Root, not Luna. The native/source role has no browser dependency; future UI verification belongs to Root's connected browser after Luna-authored source is independently admitted.

Seed `86cee15ac68c315c2bb16f9ba6b25566831bfcc7` precedes BUILD dispatch `c23f539c5336aa8034ace5b23dd6b3941fb35724`. Exact worker is `/root/r108_browser_luna` (Luna Max); Root remains independent implementation reviewer/commit steward. R108 source-only order excludes production changes, installs/provider calls, Vite/listeners/browser product work and runtime before source admission. One native diagnostic campaign is separately bounded; no R107 counters reset.

Root preparation errors retained: initial draft record incorrectly put phase names INTAKE/DESIGN/SPEC into status history; legacy preflight returned FAIL6/7 with invalid transitions INTAKE→DESIGN, DESIGN→SPEC, SPEC→WORK_ORDER. Corrected the uncommitted record to status history WORK_ORDER→DISPATCH_READY→BUILD and a separate planningPhaseTrace. Initial order omitted dispatchKind, so ROUTE was NOT_APPLICABLE_WITH_REASON, not checked. Corrected uncommitted order to INITIAL. A shell quoting error during that metadata edit had no project effect and was corrected with apply_patch. No worker source/runtime effect occurred before these corrections.

Before dispatch commit, legacy preflight PASS7/7; explicit pre-dispatch with exact R108 order returned INST/CONT/RANGE/APPL/ROUTE PASS, mandatory5/failed0. APPL and ROUTE were actually checked. These static gates admit source preparation only; they do not prove runtime diagnosis, product UI, hosted CI, or AI governance behavior. Prior accepted static gate-unit suite remains unchanged; no gratuitous rerun was used as product proof.
