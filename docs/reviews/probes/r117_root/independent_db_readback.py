"""Root read-only verification of retained R117 synthetic rows using inherited R116 test schema; no provider call.

Uses the admitted fixture supervisor's exact container ownership checks. It reads
only an explicit intervention table allowlist in r116_fixture, never user keys,
password hashes, channel credentials or the application DSN.
"""
import hashlib
import importlib.util
import json
from pathlib import Path
import re


def main():
    root = Path(__file__).resolve().parents[4]
    source = root / "scripts/r117_waitdata_fixture.py"
    spec = importlib.util.spec_from_file_location("r117_root_admitted_supervisor", source)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    module.verify_source()
    module.committed(module.LIVE_ADMISSION)
    pin = module.load(module.RUN / "fixture.json")
    before = module.inspect_fixture(pin)
    if before["State"] != "running":
        raise RuntimeError("FIXTURE_NOT_RUNNING")
    wire_path = module.EVIDENCE / "live/r116_live_wire.json"
    wire = module.load(wire_path)
    execution_id = wire["execution_id"]
    if not re.fullmatch(r"[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}", execution_id):
        raise RuntimeError("EXECUTION_ID_INVALID")

    def query(sql):
        result = module.docker("exec", "-e", "MYSQL_PWD=" + module.DB_PASSWORD,
                               pin["containerId"], "mysql", "--default-character-set=utf8mb4",
                               "-h", "127.0.0.1", "-u", "root", "-N", "-B", "--raw",
                               "r116_fixture", "-e", sql)
        if len(result.stdout) > 4 * 1024 * 1024:
            raise RuntimeError("DB_READBACK_TOO_LARGE")
        return result.stdout.decode("utf-8").splitlines()

    tenant_rows = query("SELECT tenant_id FROM intervention_executions WHERE id='" + execution_id + "'")
    if len(tenant_rows) != 1 or not re.fullmatch(r"[0-9a-f-]{36}", tenant_rows[0]):
        raise RuntimeError("EXECUTION_TENANT_UNRESOLVED")
    tenant = tenant_rows[0]
    tables = ["intervention_executions", "intervention_execution_grants", "intervention_cases",
              "intervention_rerun_requests", "intervention_data_revisions", "analysis_snapshots",
              "intervention_attempts", "intervention_budgets", "intervention_proposals",
              "intervention_receipts", "intervention_human_events", "job_runs", "messages"]
    rows = {}
    for table in tables:
        columns = query("SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA='r116_fixture' AND TABLE_NAME='" + table + "' ORDER BY ORDINAL_POSITION")
        if not columns or any(not re.fullmatch(r"[a-z0-9_]+", column) for column in columns):
            raise RuntimeError("SCHEMA_UNEXPECTED")
        fields = ",".join("'" + column + "',`" + column + "`" for column in columns)
        raw = query("SELECT JSON_OBJECT(" + fields + ") FROM `" + table + "` WHERE tenant_id='" + tenant + "' ORDER BY id LIMIT 1000")
        rows[table] = [json.loads(row) for row in raw]
        if len(raw) == 1000:
            raise RuntimeError("READBACK_TRUNCATED")

    executions = [row for row in rows["intervention_executions"] if row["id"] == execution_id]
    if len(executions) != 1:
        raise RuntimeError("EXECUTION_ROW_MISMATCH")
    execution = executions[0]
    cases = [row for row in rows["intervention_cases"] if row["id"] == execution["case_id"]]
    if len(cases) != 1:
        raise RuntimeError("CASE_ROW_MISMATCH")
    case = cases[0]
    proposals = [row for row in rows["intervention_proposals"] if row["run_id"] == execution["run_id"]]
    attempts = [row for row in rows["intervention_attempts"] if row["execution_id"] == execution_id]
    grants = [row for row in rows["intervention_execution_grants"] if row["id"] == execution["grant_id"]]
    budgets = [row for row in rows["intervention_budgets"] if row["job_id"] == execution["job_id"]]
    old_snapshots = [row for row in rows["analysis_snapshots"] if row["id"] == execution["original_snapshot_id"]]
    new_snapshots = [row for row in rows["analysis_snapshots"] if row["id"] == execution["snapshot_id"]]
    receipts = [row for row in rows["intervention_receipts"] if row["run_id"] == execution["run_id"]]
    checks = {
        "sameCasePublished": execution["state"] == "PUBLISHED" and execution["terminal_code"] == "PROPOSAL_VALID" and case["active_execution_id"] == execution_id and case["run_id"] == execution["run_id"] and case["snapshot_id"] == execution["snapshot_id"],
        "oneActiveProposal": len(proposals) == 1 and case["active_proposal_id"] == proposals[0]["id"] and proposals[0]["snapshot_digest"] == execution["snapshot_digest"],
        "oneAttemptAndDebit": len(attempts) == 1 and len(budgets) == 1 and budgets[0]["consumed"] == 1,
        "grantConsumedBySameExecution": len(grants) == 1 and grants[0]["state"] == "CONSUMED" and grants[0]["consumed_execution_id"] == execution_id,
        "originalSnapshotRetained": len(old_snapshots) == 1 and old_snapshots[0]["digest"] == execution["original_snapshot_digest"],
        "newSnapshotBound": len(new_snapshots) == 1 and new_snapshots[0]["digest"] == execution["snapshot_digest"],
        "oneTerminalReceipt": len(receipts) == 1 and receipts[0]["code"] == "PROPOSAL_VALID",
        "observedUsageMatchesWire": len(attempts) == 1 and attempts[0]["response_model"] == wire["response_model"] and attempts[0]["input_tokens"] == wire["input_tokens"] and attempts[0]["output_tokens"] == wire["output_tokens"],
    }
    response = wire.get("response", {})
    usage = response.get("usage", {}) if isinstance(response, dict) else {}
    usage = usage if isinstance(usage, dict) else {}
    prompt_tokens, completion_tokens = usage.get("prompt_tokens"), usage.get("completion_tokens")
    usage_known = type(prompt_tokens) is int and type(completion_tokens) is int and prompt_tokens >= 0 and completion_tokens >= 0
    old_runs = [row for row in rows["job_runs"] if row["id"] == wire["original_run_id"]]
    old_receipts = [row for row in rows["intervention_receipts"] if row["id"] == wire["original_receipt_id"]]
    revisions = [row for row in rows["intervention_data_revisions"] if row["id"] == wire["original_revision_id"]]
    requests = [row for row in rows["intervention_rerun_requests"] if row["id"] == wire["request_id"]]
    human = [row for row in rows["intervention_human_events"] if row["case_id"] == wire["case_id"] and row["action"] == "CONFIRM" and row["proposal_id"] == wire["active_proposal_id"] and row["actor_id"] == wire["actor_id"] and row["snapshot_id"] == wire["snapshot_id"]]
    new_runs = [row for row in rows["job_runs"] if row["id"] == execution["run_id"] and row["job_id"] == execution["job_id"]]
    snapshots_for_run = [row for row in rows["analysis_snapshots"] if row["job_run_id"] == execution["run_id"]]
    events_for_case = [row for row in rows["intervention_human_events"] if row["case_id"] == wire["case_id"]]
    checks.update({
        "rawWireUsageKnownAndFaithful": usage_known and len(attempts) == 1 and attempts[0]["input_tokens"] == prompt_tokens and attempts[0]["output_tokens"] == completion_tokens and attempts[0]["response_model"] == response.get("model") and wire["http_status"] == 200,
        "oneNewRunAndSnapshot": len(new_runs) == 1 and len(snapshots_for_run) == 1 and snapshots_for_run[0]["id"] == execution["snapshot_id"],
        "confirmedActionAndEventCount": wire["human_confirm_status"] == 200 and len(events_for_case) == wire["event_count_before_action"] + 1,
        "freshGrantEvidence": wire["owner_grant"] == "CCMAI-RUNTIME-117-OWNER-NEW1-20261017" and len(grants) == 1 and grants[0]["evidence_ref"] == wire["owner_grant"] + ":" + module.sha(module.LIVE_ADMISSION),
        "oldRunAndReceiptRetained": len(old_runs) == 1 and old_runs[0]["status"] == wire["original_run_status"] and len(old_receipts) == 1 and old_receipts[0]["code"] == wire["original_receipt_code"],
        "originalRevisionAndRequestRetained": len(revisions) == 1 and revisions[0]["digest"] == wire["original_revision_digest"] and len(requests) == 1 and requests[0]["status"] == wire["original_request_status"],
        "humanConfirmPersisted": len(human) == 1 and case["human_assessment"] == "CONFIRM" and case["version"] == wire["case_version_before_action"] + 1,
    })
    after = module.inspect_fixture(pin)
    module.exclusive(module.EVIDENCE / "independent_db_readback.json", {
        "reviewer": "Codex /root", "at": module.utc(), "readOnly": True,
        "readbackSourceSha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
        "supervisorSha256": module.sha(source), "wireReceiptSha256": module.sha(wire_path),
        "fixturePinSha256": module.sha(module.RUN / "fixture.json"),
        "checks": checks, "rows": rows, "before": before, "after": after,
        "claimBoundary": "Independent synthetic DB readback only. Root separately verifies provider wire, original-object equality, human action and any unproved matrix cases. No OS network sandbox or production claim.",
    })
    print("R117_ROOT_DB_READBACK " + ("PASS_BOUNDED_CHECKS" if all(checks.values()) else "CHANGES_REQUIRED"))
    if not all(checks.values()):
        raise RuntimeError("DB_READBACK_CHECK_FAILED")


if __name__ == "__main__":
    main()
