# Connected browser observations and interrupted return

Status: BLOCKED_WITH_REASON
Risk: R2
Worker: /root/r108_browser_luna (Luna Max implementation and supervisor)
Reviewer: Codex /root (independent implementation review)
Browser operator: Codex /root
Disposition: PARTIAL_CURRENT_SOURCE_EVIDENCE_TRANSFER_REQUIRED

Source: 5ab60830368482e072cb0150a7894235ca950c24; admitted execution HEAD:
7fc2273b485b765543dd78fc60afc51ee68d0cb3. Backend is the unchanged current
source represented by the retained 58,031,616-byte binary cb73ee13; exact
SHA256 is in runtime_admission.json. UI source first passed fourteen focused
tests and forced typecheck. Source/binary pins were independently rechecked
after interruption and remain stable. No current claim uses the old R107 binary.

The fresh HTTP browser bridge opened/clicked/captured PASS before product
startup. The native diagnostic then ran exactly once: four actions, exit zero,
NOT_REPRODUCED and lifecycle accepted. This does not establish the historical
503 cause or retroactively change R107's failed evidence.

## Product observations

All named capture stems refer to original DOM/PNG/metadata files under
`docs/reviews/probes/r109_root/browser/`. Metadata action descriptions are
operator records, not a raw serialized export of the entire tool conversation.
Actual tool calls/results remain in the task transcript. The captures are
actual rendered data and screenshot bytes, not reconstructed images.

| Case | Result | Expected and actual observation | Capture stems |
| --- | --- | --- | --- |
| B01 | PASS | Fresh owner login and fresh tenant queue; new pending case update1, initial proposal and empty history. An inherited synthetic reader session was signed out first. | 01, 02 |
| B02 | PASS | CORRECT once -> update2, distinct human explanation/next step, persisted actor/time/reason and original proposal. | 03, 04, 05 |
| B03 | PASS | Second tab kept update1; stale CONFIRM produced real HTTP409, readable conflict and update2 reload. No automatic write replay. | 06, 07; backend access log |
| B04 | PASS | Offered tenant member assigned; update3 and 20/10/2026 15:30 deadline survive reload. | 08, 09 |
| B05 | PASS | Outcome saved at update4, survives reload; terminal filter contains case. | 10, 11 |
| B06 | PASS | Exact prior Vietnamese reopen reason succeeds HTTP200; reload update5 assigned, new 21/10/2026 16:45 deadline, active outcome cleared. Expanded readable history retains the exact prior outcome, original proposal unchanged. | 12, 13 |
| B07 | NOT_RUN | Owner logout HTTP200 observed. Interruption occurred before fresh reader login; current reader write-control absence was not tested. | browser_interruption.json; access log |
| B08 | NOT_RUN | Documented viewport.set(390x844) returned without error; actual DOM and PNG remained1280x720, including after one reload. Reset invoked. No mobile acceptance. | 14_b08_dimensions; 14_b08_viewport_outcome |
| B09 | NOT_RUN | No documented offline control exposed; no network changes or interception used to manufacture coverage. | capability inventory |
| B10 | PASS (functional) | Existing waiting case: assign data owner/deadline/next step -> update2; escalate -> update3; reload still waiting/incomplete, owner/deadline/history retained. Action list offers data assignment/escalation only, not confirmation or completion. A wording defect below remains OPEN. | 15 through 18 |

Exact outcome retained in B06 history: “Đã xác nhận lịch giao mới với khách và
hẹn theo dõi ngày mai.” Its presence in the history's Before field and absence
as active outcome are visible together in capture13.

## Open findings and evidence qualifications

- The waiting-data source alert says information is incomplete, but its generic
  second sentence claims a local match. Replace that sentence with a neutral
  limitation for every source status; do not claim a match for INCOMPLETE.
- The shared shell still exposes “Cấu hình AI Provider”, “Kết nối MCP”, and
  service terms on the dashboard. The intervention-specific raw codes/JSON/
  machine identifiers were removed, but application-wide plain language is
  not accepted. These visible findings are preserved, not hidden by dismissing
  the banner or editing source during the campaign.
- The artifact named `14_b08_narrow_detail_history` was captured after the
  resize request but is actually desktop size. Its original provisional
  action label is corrected explicitly by dimensions/outcome receipts. It
  must never be used as mobile proof.
- Initial automatic auth refresh and the unlabeled old-session logout were
  reconciled from real server counts. The logout lacked a before-click
  reservation; this procedural gap is preserved in the ledger. No backdated
  reservation is claimed. All eleven actual UI POSTs remain counted.

## Interruption and cleanup

The last multi-action browser call was aborted. On continuation, inventory
showed unreachable product tabs; the recorded Vite PID was absent and the old
backend PID was now conhost. Acquiring the error page was explicitly denied by
browser URL policy; exact error is retained in browser_interruption.json.
No workaround, alternate automation, runtime restart or old-PID kill followed.
The cause of the execution-context interruption is NOT_OBSERVED.

The original supervisor did not leave cleanup.json or a terminal disposition.
Root requested cooperative stop and independently recovered only the exact
owned fixture, which was still running. Structured ID/image/label/network/
exclusive-volume checks passed; one stop returned it to exited. Protected
Docker safe fields match both the original campaign baseline and the state
before Root cleanup. Full commands/outputs and root_recovery_receipt.json are
under interrupted_cleanup/. This separate recovery does not replace the
missing original supervisor receipt. Final tab closure is NOT_OBSERVED after
the denied acquisition; before interruption, closing tabB had returned success.

Actual effects: Go compiles0, fixture starts1, native invocations1/actionPOSTs4,
backend listeners1, Vite1, UI POSTs11 including refresh/logout/login/rejection,
two confirmed product tabs created; completion of the attempted third creation
is NOT_OBSERVED. No provider/customer/channel calls, install, push or deployment.
The two visible pages after reset cannot establish that third-tab creation.

Owner's explicit stop/transfer condition now applies. Return this partial
packet to an agent with a stable connected browser/execution context; preserve
spent counters, original gaps and OPEN language findings. No FREEZE or full
R101/product acceptance.
