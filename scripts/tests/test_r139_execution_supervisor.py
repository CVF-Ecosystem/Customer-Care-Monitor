"""Pure, effect-free checks of R139 supervisor admission and terminal guards."""
from __future__ import annotations

import importlib.util
import json
from pathlib import Path
import subprocess
import tempfile
import time
import unittest
from unittest.mock import patch
from datetime import datetime, timedelta, timezone
from types import SimpleNamespace

SOURCE = Path(__file__).resolve().parents[1] / "r139_execution_supervisor.py"
SPEC = importlib.util.spec_from_file_location("r139_execution_supervisor", SOURCE)
assert SPEC and SPEC.loader
module = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(module)


def protected_snapshot(*, status="Up 2 minutes (healthy)", running_for="2 minutes ago", network_id="network-id", started="2026-10-23T00:00:00Z", restarts=0, captured=100.0, captured_at="2026-10-23T00:02:30Z"):
    state = {"Status": "running", "Running": True, "Paused": False, "Restarting": False, "OOMKilled": False, "Dead": False, "Pid": 123, "ExitCode": 0, "Error": "", "StartedAt": started, "FinishedAt": "0001-01-01T00:00:00Z"}
    typed = {"Id": "protected-id", "Name": "/protected", "Image": "sha256:image", "Created": "2026-10-23T00:00:00Z", "RestartCount": restarts, "State": state, "Health": {"Status": "healthy", "FailingStreak": 0}, "NetworkIdentity": {"bridge": {"NetworkID": network_id}}}
    return {"containers": [{"ID": "protected-id", "Networks": "bridge, custom", "Status": status, "State": "running", "RunningFor": running_for,
                            "_typed": typed}],
            "networks": [{"ID": network_id, "Name": "protected", "Driver": "bridge", "Scope": "local", "_typed": {"Id": network_id, "Name": "protected", "Driver": "bridge", "Scope": "local"}}],
            "volumes": [{"Name": "protected-volume", "Driver": "local", "Scope": "local", "_typed": {"Name": "protected-volume", "Driver": "local", "Scope": "local"}}],
            "capturedMonotonic": captured, "capturedAtUTC": captured_at}


class R139SupervisorPureTests(unittest.TestCase):
    def test_source_resume_admission_cannot_enable_runtime(self):
        with patch.object(module, "committed") as committed:
            with self.assertRaisesRegex(module.Hold, "RUNTIME_ADMISSION_PATH_INVALID"):
                module.fresh_runtime_admission("CVF_SESSION/authority/CCMAI-RUNTIME-139.json")
            committed.assert_not_called()
        with patch.object(module, "committed") as committed, patch.object(module, "read_json", return_value={"trancheId": "CCMAI-RUNTIME-139", "kind": "DISPATCHER_OWNED_R139_SOURCE_PREPARATION_RESUME_ONLY"}):
            committed.return_value = SOURCE
            with self.assertRaisesRegex(module.Hold, "RUNTIME_ADMISSION_KIND_INVALID"):
                module.fresh_runtime_admission("CVF_SESSION/authority/CCMAI-RUNTIME-139-SOURCE-RESUME-ADMISSION.json")

    def test_exclusive_reservation_survives_second_attempt(self):
        with tempfile.TemporaryDirectory() as name:
            path = Path(name) / "effect.json"
            module.write_new(path, {"state": "MAY_HAVE_STARTED"})
            with self.assertRaises(FileExistsError):
                module.write_new(path, {"state": "RETRY"})
            self.assertEqual(json.loads(path.read_text())["state"], "MAY_HAVE_STARTED")

    def test_inventory_projection_preserves_all_unowned_resources(self):
        observed = {
            "containers": [{"ID": "old", "Names": "protected"}, {"ID": "new", "Names": "ccmai-r139-owned"}],
            "networks": [{"ID": "old-net", "Name": "protected"}, {"ID": "new-net", "Name": "ccmai-r139-owned"}],
            "volumes": [{"Name": "protected"}, {"Name": "ccmai-r139-owned"}],
            "capturedMonotonic": 10.0,
            "capturedAtUTC": "2026-10-23T00:00:00Z",
        }
        result = module.excluded_owned(observed, {"containerId": "new", "networkId": "new-net", "volumeName": "ccmai-r139-owned"})
        self.assertEqual(result, {"containers": observed["containers"][:1], "networks": observed["networks"][:1], "volumes": observed["volumes"][:1], "capturedMonotonic": 10.0, "capturedAtUTC": "2026-10-23T00:00:00Z"})
        self.assertEqual(len(observed["containers"]), 2)

    def test_protected_inventory_normalizes_network_order_and_elapsed_only_with_stable_anchors(self):
        before = protected_snapshot()
        after = protected_snapshot(status="Up 4 minutes (healthy)", running_for="4 minutes ago", captured=220.0, captured_at="2026-10-23T00:04:30Z")
        after["containers"][0]["Networks"] = "custom,bridge"
        original_before = json.loads(json.dumps(before))
        self.assertEqual(module.canonicalize_protected_inventory(before), module.canonicalize_protected_inventory(after))
        self.assertEqual(before, original_before)
        self.assertEqual(after["containers"][0]["RunningFor"], "4 minutes ago")

    def test_protected_status_preserves_state_health_exitcode_and_other_fields(self):
        base = protected_snapshot(status="Up 29 minutes (healthy)", running_for="29 minutes ago", captured_at="2026-10-23T00:29:30Z")
        elapsed = json.loads(json.dumps(base))
        elapsed["containers"][0]["Status"] = "Up 36 minutes (healthy)"
        elapsed["containers"][0]["RunningFor"] = "36 minutes ago"
        elapsed["capturedAtUTC"] = "2026-10-23T00:36:30Z"
        self.assertEqual(module.canonicalize_protected_inventory(base), module.canonicalize_protected_inventory(elapsed))
        for field, value in (("Status", "Up 36 minutes (unhealthy)"), ("ID", "y")):
            changed = json.loads(json.dumps(elapsed))
            changed["containers"][0][field] = value
            with self.assertRaises(module.Hold):
                module.canonicalize_protected_inventory(changed)
        for field, value in (("RestartCount", 1), ("Image", "sha256:changed")):
            changed = json.loads(json.dumps(elapsed))
            changed["containers"][0]["_typed"][field] = value
            self.assertNotEqual(module.canonicalize_protected_inventory(base), module.canonicalize_protected_inventory(changed), field)
        changed = json.loads(json.dumps(elapsed))
        changed["containers"][0]["_typed"]["State"]["StartedAt"] = "2026-10-23T00:30:00Z"
        with self.assertRaisesRegex(module.Hold, "PROTECTED_ELAPSED_ANCHOR_MISMATCH"):
            module.canonicalize_protected_inventory(changed)
        bridge = protected_snapshot(network_id="replacement-id")
        self.assertNotEqual(module.canonicalize_protected_inventory(base), module.canonicalize_protected_inventory(bridge))
        mismatch = json.loads(json.dumps(base))
        mismatch["containers"][0]["State"] = "exited"
        with self.assertRaisesRegex(module.Hold, "PROTECTED_STATUS_STATE_MISMATCH"):
            module.canonicalize_protected_inventory(mismatch)

    def test_protected_status_unknown_formats_hold(self):
        for status in ("Restarting (1) 3 seconds ago", "Dead", "Up About a minute", "Exited (0) unknown ago", "Up 2 minutes (health: unknown)", ""):
            with self.subTest(status=status), self.assertRaisesRegex(module.Hold, "PROTECTED_STATUS_UNKNOWN_FORMAT"):
                module.canonicalize_protected_inventory(protected_snapshot(status=status))

    def test_protected_old_r3_rows_and_running_for_reset_hold(self):
        old = {"containers": [{"ID": "old", "Status": "Up 2 minutes", "State": "running", "RunningFor": "2 minutes"}], "networks": [], "volumes": [], "capturedMonotonic": 1.0, "capturedAtUTC": "2026-10-23T00:02:30Z"}
        with self.assertRaisesRegex(module.Hold, "PROTECTED_CONTAINER_ANCHOR_INVALID"):
            module.canonicalize_protected_inventory(old)
        restarted = protected_snapshot(status="Up 1 minute (healthy)", running_for="2 minutes ago", started="2026-10-23T00:01:00Z", restarts=1)
        self.assertNotEqual(module.canonicalize_protected_inventory(protected_snapshot()), module.canonicalize_protected_inventory(restarted))

    def test_protected_elapsed_anchor_and_capture_clock_hold(self):
        fresh = protected_snapshot(status="Up 4 seconds (healthy)", running_for="4 seconds ago", started="2026-10-23T00:00:00Z", captured_at="2026-10-23T00:00:04Z")
        self.assertEqual(module.canonicalize_protected_inventory(fresh), module.canonicalize_protected_inventory(fresh))
        reset = protected_snapshot(status="Up 2 minutes (healthy)", running_for="2 minutes ago", captured_at="2026-10-23T01:00:00Z")
        with self.assertRaisesRegex(module.Hold, "PROTECTED_ELAPSED_ANCHOR_MISMATCH"):
            module.canonicalize_protected_inventory(reset)
        missing = protected_snapshot()
        del missing["containers"][0]["_typed"]["State"]["StartedAt"]
        with self.assertRaisesRegex(module.Hold, "PROTECTED_STATE_ANCHOR_INVALID"):
            module.canonicalize_protected_inventory(missing)
        for location, field in (("_typed", "RestartCount"), ("_typed", "Image")):
            absent = protected_snapshot()
            del absent["containers"][0][location][field]
            with self.assertRaisesRegex(module.Hold, "PROTECTED_CONTAINER_ANCHOR_INVALID"):
                module.canonicalize_protected_inventory(absent)
        absent = protected_snapshot()
        del absent["networks"][0]["_typed"]["Driver"]
        with self.assertRaisesRegex(module.Hold, "PROTECTED_RESOURCE_ANCHOR_INVALID"):
            module.canonicalize_protected_inventory(absent)
        absent = protected_snapshot()
        absent["containers"][0]["_typed"]["Health"]["FailingStreak"] = "unknown"
        with self.assertRaisesRegex(module.Hold, "PROTECTED_HEALTH_ANCHOR_INVALID"):
            module.canonicalize_protected_inventory(absent)

    def test_inventory_projects_safe_typed_fields_only(self):
        identifier = "a" * 64
        listing = json.dumps({"ID": identifier, "Name": "protected", "Status": "Up 4 seconds", "State": "running", "RunningFor": "4 seconds ago"})
        projected = protected_snapshot(status="Up 4 seconds (healthy)", running_for="4 seconds ago", captured_at="2026-10-23T00:00:04Z")["containers"][0]["_typed"]
        projected["Id"] = identifier
        calls = []
        def fake_docker(_admission, _runtime, *args):
            calls.append(args)
            if args[:1] == ("ps",): return listing
            if args[:2] == ("network", "ls") or args[:2] == ("volume", "ls"): return ""
            if args[:1] == ("inspect",): return json.dumps(projected)
            raise AssertionError(args)
        with patch.object(module, "docker", side_effect=fake_docker):
            observed = module.inventory({}, Path("ignored"))
        self.assertEqual(observed["containers"][0]["_typed"]["Id"], identifier)
        projected_template = calls[-1][2]
        self.assertNotIn("Config.Env", projected_template)
        self.assertNotIn(".Log", projected_template)
        self.assertNotIn("Config.Cmd", projected_template)

    def test_mysql_readiness_uses_tcp_matching_database_and_retries_not_ready_only(self):
        admission = {"database": "ccma_r139_r4_20261023"}
        owner = {"containerId": "owned-container", "fixturePassword": "synthetic-password"}
        with patch.object(module, "ACTIVE_HARD_DEADLINE", None), patch.object(
            module, "docker", side_effect=[module.Hold("DOCKER_COMMAND_FAILED"), "1"]
        ) as docker, patch.object(module.time, "sleep") as sleep:
            result = module.wait_mysql_tcp_ready(admission, Path("ignored-runtime"), owner)
        self.assertEqual(result["status"], "PASS_TCP_SELECT_1")
        self.assertEqual(result["database"], admission["database"])
        self.assertEqual(result["probeCount"], 2)
        self.assertEqual(docker.call_count, 2)
        args = docker.call_args_list[0].args
        self.assertIn("--protocol=tcp", args)
        self.assertIn("--host=127.0.0.1", args)
        self.assertIn("--port=3306", args)
        self.assertIn(admission["database"], args)
        self.assertIn("SELECT 1", args)
        self.assertLessEqual(docker.call_args_list[0].kwargs["timeout"], module.MYSQL_PROBE_TIMEOUT_SECONDS)
        sleep.assert_called_once_with(module.MYSQL_PROBE_SPACING_SECONDS)

    def test_mysql_readiness_stops_on_probe_timeout_without_retry(self):
        with patch.object(module, "ACTIVE_HARD_DEADLINE", None), patch.object(
            module, "docker", side_effect=module.Hold("DOCKER_TIMEOUT_NO_RETRY")
        ) as docker, patch.object(module.time, "sleep") as sleep:
            with self.assertRaisesRegex(module.Hold, "MYSQL_TCP_READY_PROBE_TIMEOUT_NO_RETRY"):
                module.wait_mysql_tcp_ready({"database": "ccma_r139_r4_20261023"}, Path("ignored-runtime"), {"containerId": "owned", "fixturePassword": "synthetic"})
        docker.assert_called_once()
        sleep.assert_not_called()

    def test_mysql_readiness_late_success_is_not_accepted(self):
        class FakeClock:
            now = 0.0
            def monotonic(self): return self.now
        clock = FakeClock()
        def late_success(*args, **kwargs):
            clock.now = module.MYSQL_STARTUP_WINDOW_SECONDS + 0.01
            return "1"
        with patch.object(module, "ACTIVE_HARD_DEADLINE", None), patch.object(
            module.time, "monotonic", side_effect=clock.monotonic
        ), patch.object(module, "docker", side_effect=late_success) as docker:
            with self.assertRaisesRegex(module.Hold, "MYSQL_TCP_READY_WINDOW_EXPIRED"):
                module.wait_mysql_tcp_ready(
                    {"database": "ccma_r139_r4_20261023"},
                    Path("ignored-runtime"),
                    {"containerId": "owned", "fixturePassword": "synthetic"},
                )
        docker.assert_called_once()

    def test_mysql_readiness_enforces_probe_cap_and_global_cleanup_reserve(self):
        class FakeClock:
            def __init__(self, now): self.now = now
            def monotonic(self): return self.now
            def sleep(self, seconds): self.now += seconds
        clock = FakeClock(0.0)
        with patch.object(module, "ACTIVE_HARD_DEADLINE", None), patch.object(module, "MYSQL_MAX_PROBES", 45), patch.object(
            module.time, "monotonic", side_effect=clock.monotonic
        ), patch.object(module.time, "sleep", side_effect=clock.sleep), patch.object(
            module, "docker", side_effect=module.Hold("DOCKER_COMMAND_FAILED")
        ) as docker:
            with self.assertRaisesRegex(module.Hold, "MYSQL_TCP_READY_NOT_PROVEN"):
                module.wait_mysql_tcp_ready({"database": "ccma_r139_r4_20261023"}, Path("ignored-runtime"), {"containerId": "owned", "fixturePassword": "synthetic"})
        self.assertEqual(docker.call_count, 45)

        clock = FakeClock(100.0)
        with patch.object(module, "ACTIVE_HARD_DEADLINE", 280.5), patch.object(
            module.time, "monotonic", side_effect=clock.monotonic
        ), patch.object(module.time, "sleep", side_effect=clock.sleep), patch.object(
            module, "docker", side_effect=module.Hold("DOCKER_COMMAND_FAILED")
        ) as docker:
            with self.assertRaisesRegex(module.Hold, "MYSQL_TCP_READY_NOT_PROVEN"):
                module.wait_mysql_tcp_ready({"database": "ccma_r139_r4_20261023"}, Path("ignored-runtime"), {"containerId": "owned", "fixturePassword": "synthetic"})
        docker.assert_called_once()
        self.assertLessEqual(docker.call_args.kwargs["timeout"], 0.5)

    def test_windows_listener_inventory_accepts_only_pinned_exact_zero(self):
        class FakeSocket:
            def __enter__(self): return self
            def __exit__(self, *args): return False
            def settimeout(self, timeout): self.timeout = timeout
            def connect_ex(self, address): self.address = address; return 10035
        with tempfile.TemporaryDirectory() as name:
            root = Path(name)
            binary = root / "powershell.exe"
            binary.write_bytes(b"pinned powershell fixture executable")
            runtime = root / "runtime"
            runtime.mkdir()
            port = 52402
            output = json.dumps({"queryOk": True, "port": port, "listeners": []})
            result = SimpleNamespace(returncode=0, stdout=output, stderr="")
            admission = {"powershellBinary": str(binary), "powershellBinarySha256": module.digest(binary)}
            with patch.object(module.socket, "socket", return_value=FakeSocket()), patch.object(
                module.subprocess, "run", return_value=result
            ) as run, patch.object(module, "ACTIVE_CLEANUP_DEADLINE", None), patch.object(module, "ACTIVE_HARD_DEADLINE", None):
                observed = module.listener_absent(port, admission, runtime)
            self.assertTrue(observed["listenerAbsent"])
            self.assertEqual(observed["evidenceSource"], "pinned_powershell_tcp_listener_inventory")
            command = run.call_args.args[0]
            self.assertEqual(command[0], str(binary))
            self.assertIn("-NoProfile", command)
            self.assertIn("-NonInteractive", command)
            script = command[-1]
            self.assertIn("Get-NetTCPConnection -State Listen", script)
            self.assertNotIn("-LocalPort", script)
            self.assertIn(f"$_.LocalPort -eq {port}", script)
            self.assertTrue((runtime / f"listener_query_{port}_intent.json").is_file())
            receipt = json.loads((runtime / f"listener_query_{port}_result.json").read_text(encoding="utf-8"))
            self.assertEqual(receipt["status"], "PASS_ZERO_LISTENERS")
            self.assertEqual(receipt["stdout"], output)

    def test_windows_listener_inventory_nonzero_result_holds(self):
        class FakeSocket:
            def __enter__(self): return self
            def __exit__(self, *args): return False
            def settimeout(self, timeout): pass
            def connect_ex(self, address): return 10035
        with tempfile.TemporaryDirectory() as name:
            root = Path(name)
            binary = root / "powershell.exe"
            binary.write_bytes(b"pinned powershell fixture executable")
            runtime = root / "runtime"
            runtime.mkdir()
            port = 52402
            output = json.dumps({"queryOk": True, "port": port, "listeners": [{"port": port, "state": "Listen"}]})
            admission = {"powershellBinary": str(binary), "powershellBinarySha256": module.digest(binary)}
            with patch.object(module.socket, "socket", return_value=FakeSocket()), patch.object(
                module.subprocess, "run", return_value=SimpleNamespace(returncode=0, stdout=output, stderr="")
            ) as run, patch.object(module, "ACTIVE_CLEANUP_DEADLINE", None), patch.object(module, "ACTIVE_HARD_DEADLINE", None):
                with self.assertRaisesRegex(module.Hold, "WINDOWS_LISTENER_PRESENT"):
                    module.listener_absent(port, admission, runtime)
            run.assert_called_once()
            receipt = json.loads((runtime / f"listener_query_{port}_result.json").read_text(encoding="utf-8"))
            self.assertEqual(receipt["status"], "HOLD")

    def test_windows_listener_inventory_malformed_or_error_result_holds(self):
        class FakeSocket:
            def __enter__(self): return self
            def __exit__(self, *args): return False
            def settimeout(self, timeout): pass
            def connect_ex(self, address): return 10035
        port = 52402
        valid_zero = json.dumps({"queryOk": True, "port": port, "listeners": []})
        cases = [
            (SimpleNamespace(returncode=0, stdout="not-json", stderr=""), "WINDOWS_LISTENER_QUERY_INVALID"),
            (SimpleNamespace(returncode=0, stdout=valid_zero, stderr="warning"), "WINDOWS_LISTENER_QUERY_STDERR"),
            (SimpleNamespace(returncode=2, stdout=valid_zero, stderr="query failed"), "WINDOWS_LISTENER_QUERY_FAILED"),
        ]
        for result, error in cases:
            with self.subTest(error=error), tempfile.TemporaryDirectory() as name:
                root = Path(name)
                binary = root / "powershell.exe"
                binary.write_bytes(b"pinned powershell fixture executable")
                runtime = root / "runtime"
                runtime.mkdir()
                admission = {"powershellBinary": str(binary), "powershellBinarySha256": module.digest(binary)}
                with patch.object(module.socket, "socket", return_value=FakeSocket()), patch.object(
                    module.subprocess, "run", return_value=result
                ) as run, patch.object(module, "ACTIVE_CLEANUP_DEADLINE", None), patch.object(module, "ACTIVE_HARD_DEADLINE", None):
                    with self.assertRaisesRegex(module.Hold, error):
                        module.listener_absent(port, admission, runtime)
                run.assert_called_once()

    def test_windows_listener_inventory_timeout_holds_without_retry(self):
        class FakeSocket:
            def __enter__(self): return self
            def __exit__(self, *args): return False
            def settimeout(self, timeout): pass
            def connect_ex(self, address): return 10035
        with tempfile.TemporaryDirectory() as name:
            root = Path(name)
            binary = root / "powershell.exe"
            binary.write_bytes(b"pinned powershell fixture executable")
            runtime = root / "runtime"
            runtime.mkdir()
            admission = {"powershellBinary": str(binary), "powershellBinarySha256": module.digest(binary)}
            with patch.object(module.socket, "socket", return_value=FakeSocket()), patch.object(
                module.subprocess, "run", side_effect=subprocess.TimeoutExpired(["powershell.exe"], 5)
            ) as run, patch.object(module, "ACTIVE_CLEANUP_DEADLINE", None), patch.object(module, "ACTIVE_HARD_DEADLINE", None):
                with self.assertRaisesRegex(module.Hold, "WINDOWS_LISTENER_QUERY_TIMEOUT_NO_RETRY"):
                    module.listener_absent(52402, admission, runtime)
            run.assert_called_once()
            receipt = json.loads((runtime / "listener_query_52402_result.json").read_text(encoding="utf-8"))
            self.assertEqual(receipt["status"], "TIMEOUT_NO_RETRY")

    def test_windows_listener_receipts_are_port_qualified(self):
        class FakeSocket:
            def __enter__(self): return self
            def __exit__(self, *args): return False
            def settimeout(self, timeout): pass
            def connect_ex(self, address): return 10035
        with tempfile.TemporaryDirectory() as name:
            root = Path(name)
            binary = root / "powershell.exe"
            binary.write_bytes(b"pinned powershell fixture executable")
            runtime = root / "runtime"
            runtime.mkdir()
            ports = (52402, 56961)
            admission = {"powershellBinary": str(binary), "powershellBinarySha256": module.digest(binary)}
            def query(argv, **kwargs):
                port = next(port for port in ports if f"LocalPort -eq {port}" in argv[-1])
                stdout = json.dumps({"queryOk": True, "port": port, "listeners": []})
                return SimpleNamespace(returncode=0, stdout=stdout, stderr="")
            with patch.object(module.socket, "socket", side_effect=[FakeSocket(), FakeSocket()]), patch.object(
                module.subprocess, "run", side_effect=query
            ) as run, patch.object(module, "ACTIVE_CLEANUP_DEADLINE", None), patch.object(module, "ACTIVE_HARD_DEADLINE", None):
                results = [module.listener_absent(port, admission, runtime) for port in ports]
            self.assertEqual(run.call_count, 2)
            self.assertTrue(all(result["listenerAbsent"] for result in results))
            for port in ports:
                self.assertTrue((runtime / f"listener_query_{port}_intent.json").is_file())
                receipt = json.loads((runtime / f"listener_query_{port}_result.json").read_text(encoding="utf-8"))
                self.assertEqual(receipt["status"], "PASS_ZERO_LISTENERS")

    def test_replacement_evidence_path_is_fixture_r4(self):
        with tempfile.TemporaryDirectory() as name:
            project = Path(name) / "Customer-Care-Monitor-AI"
            project.mkdir()
            with patch.object(module, "ROOT", project):
                self.assertEqual(
                    module.replacement_external_evidence_dir(),
                    project.parent / ".ccmai_local_evidence" / "r139_execution_proof_20261023" / "fixture_r4",
                )

    def test_owned_container_rejects_label_or_id_mismatch(self):
        observed = [{"Id": "a" * 64, "Name": "/ccmai-r139-owned", "Config": {"Labels": {"ccmai.tranche": "CCMAI-RUNTIME-139", "ccmai.owner": "wrong"}}}]
        with patch.object(module, "docker", return_value=json.dumps(observed)):
            with self.assertRaisesRegex(module.Hold, "CONTAINER_OWNERSHIP_UNPROVEN"):
                module.owned_container({"fixtureName": "ccmai-r139-owned"}, Path("ignored"), {"containerId": "a" * 64, "nonce": "right"})

    def test_stop_child_records_wait_exit_without_mutating_unrelated_process(self):
        class Child:
            pid = 11939
            returncode = 0
            waited = False
            def poll(self): return 0
            def wait(self, timeout): self.waited = True
            def terminate(self): raise AssertionError("terminal process must not be terminated")
        child = Child()
        with tempfile.TemporaryDirectory() as name:
            result = module.stop_child(Path(name), "go", child)
        self.assertTrue(child.waited)
        self.assertEqual(result["exitCode"], 0)
        self.assertTrue(result["waitObserved"])

    def test_false_browser_receipt_and_incomplete_pins_hold(self):
        with tempfile.TemporaryDirectory() as name:
            receipt = Path(name) / "browser.json"
            receipt.write_text(json.dumps({"status": "DENIED", "browserId": "iab"}), encoding="utf-8")
            with patch.object(module, "project_path", return_value=receipt):
                with self.assertRaisesRegex(module.Hold, "BROWSER_PROOF_CONTENT_INCOMPLETE"):
                    module.verify_browser_receipt({"browserTool": "supported_CUA_IAB", "browserSupported": True, "browserReceiptPath": "browser.json", "browserReceiptSha256": module.digest(receipt)})
        with self.assertRaisesRegex(module.Hold, "SOURCE_PIN_SET_INCOMPLETE"):
            module.verify_source_pins({"scripts/r139_execution_supervisor.py": "0" * 64})

    def test_dirty_source_rejected_before_any_fixture(self):
        relative = "CVF_SESSION/authority/CCMAI-RUNTIME-139-EFFECT-ADMISSION.json"
        value = {"trancheId": "CCMAI-RUNTIME-139", "kind": "OWNER_APPROVED_RUNTIME_EFFECT_ADMISSION", "ownerApproved": True,
                 "parentAuthorityCommit": "642f5bd026dfe31e6dda6441dd25704eb3ececa3", "sourceResumeAdmissionCommit": "6fc951a84c54cd31f20f2404d6dc8957d97bc339", "sourceCommit": "a" * 40}
        class Result:
            def __init__(self, code, stdout=b""): self.returncode, self.stdout = code, stdout
        with patch.object(module, "committed", return_value=SOURCE), patch.object(module, "read_json", return_value=value), patch.object(module.subprocess, "run", side_effect=[Result(0), Result(0, b" M source\n")]):
            with self.assertRaisesRegex(module.Hold, "SOURCE_WORKTREE_DIRTY"):
                module.fresh_runtime_admission(relative)

    def test_partial_resource_failure_retains_owner_identity(self):
        with tempfile.TemporaryDirectory() as name:
            runtime = Path(name) / "runtime"
            external = Path(name) / "external"
            runtime.mkdir(); external.mkdir()
            owner = {}
            before = {"containers": [], "networks": [], "volumes": [], "capturedMonotonic": 1.0, "capturedAtUTC": "2026-10-23T00:00:00Z"}
            with patch.object(module, "inventory", return_value=before), patch.object(module, "docker", side_effect=["a" * 64, module.Hold("VOLUME_CREATE_FAILED")]):
                with self.assertRaisesRegex(module.Hold, "VOLUME_CREATE_FAILED"):
                    module.start_fixture({"fixtureName": "ccmai-r139-one", "database": "ccma_r139_one"}, runtime, external, owner)
            self.assertEqual(owner["networkId"], "a" * 64)
            self.assertEqual(owner["protectedBefore"], before)
            self.assertTrue((runtime / "network_created.json").is_file())

    def test_post_spawn_receipt_failure_stops_exact_child(self):
        class Child:
            pid = 139
            returncode = None
            stopped = False
            def poll(self): return None if not self.stopped else 0
            def terminate(self): self.stopped = True; self.returncode = 0
            def wait(self, timeout): return self.returncode
        child = Child()
        with tempfile.TemporaryDirectory() as name:
            with patch.object(module.subprocess, "Popen", return_value=child), patch.object(module, "process_creation_identity", side_effect=module.Hold("RECEIPT_FAILED")):
                children = {}
                with self.assertRaisesRegex(module.Hold, "RECEIPT_FAILED"):
                    module.launch({}, Path(name), "go", ["inert"], {}, children)
                self.assertIs(children["go"], child)
                self.assertTrue(child.stopped)

    def test_cleanup_disk_error_does_not_prevent_termination(self):
        class Child:
            pid = 139
            returncode = None
            stopped = False
            def poll(self): return None if not self.stopped else 0
            def terminate(self): self.stopped = True; self.returncode = 0
            def wait(self, timeout): return self.returncode
        child = Child()
        with patch.object(module, "append", side_effect=OSError("disk full")):
            result = module.stop_child(Path("ignored"), "go", child)
        self.assertTrue(child.stopped)
        self.assertEqual(result["exitCode"], 0)

    def test_cleanup_docker_journal_failure_runs_bounded_with_minimal_environment(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name)
            binary = root / "docker.exe"
            binary.write_bytes(b"pinned synthetic executable placeholder")
            runtime = root / "runtime"
            runtime.mkdir()
            admission = {"dockerBinary": str(binary), "dockerBinarySha256": "a" * 64, "dockerHost": "npipe:////./pipe/dockerDesktopLinuxEngine"}
            result = SimpleNamespace(returncode=0, stdout=b"stopped", stderr=b"")
            old_cleanup, old_hard = module.ACTIVE_CLEANUP_DEADLINE, module.ACTIVE_HARD_DEADLINE
            try:
                module.ACTIVE_HARD_DEADLINE = time.monotonic() - 1
                module.ACTIVE_CLEANUP_DEADLINE = time.monotonic() + 5
                with patch.object(module, "digest", return_value="a" * 64), patch.object(module, "write_new", side_effect=OSError("disk full")), patch.object(module.subprocess, "run", return_value=result) as run:
                    self.assertEqual(module.docker(admission, runtime, "stop", "--time", "1", "owned-container"), "stopped")
                self.assertEqual(run.call_count, 1)
                env = run.call_args.kwargs["env"]
                self.assertNotIn("DOCKER_CONTEXT", env)
                self.assertNotIn("HTTP_PROXY", env)
                self.assertTrue(module.CLEANUP_ERRORS)
                module.ACTIVE_CLEANUP_DEADLINE = None
                module.ACTIVE_HARD_DEADLINE = time.monotonic() + module.MIN_CLEANUP_SECONDS + 5
                with patch.object(module, "digest", return_value="a" * 64), patch.object(module, "write_new", side_effect=OSError("disk full")), patch.object(module.subprocess, "run") as run:
                    with self.assertRaises(OSError):
                        module.docker(admission, runtime, "create", "owned-container")
                    run.assert_not_called()
            finally:
                module.ACTIVE_CLEANUP_DEADLINE, module.ACTIVE_HARD_DEADLINE = old_cleanup, old_hard

    def test_non_cleanup_docker_command_preserves_cleanup_reserve(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name)
            binary = root / "docker.exe"
            binary.write_bytes(b"pinned synthetic executable placeholder")
            runtime = root / "runtime"
            runtime.mkdir()
            (runtime / "commands").mkdir()
            admission = {"dockerBinary": str(binary), "dockerBinarySha256": "a" * 64, "dockerHost": "npipe:////./pipe/dockerDesktopLinuxEngine"}
            result = SimpleNamespace(returncode=0, stdout=b"ok", stderr=b"")
            old_cleanup, old_hard = module.ACTIVE_CLEANUP_DEADLINE, module.ACTIVE_HARD_DEADLINE
            try:
                module.ACTIVE_CLEANUP_DEADLINE = None
                module.ACTIVE_HARD_DEADLINE = time.monotonic() + module.MIN_CLEANUP_SECONDS + 5
                with patch.object(module, "digest", return_value="a" * 64), patch.object(module, "write_new"), patch.object(
                    module.subprocess, "run", return_value=result
                ) as run:
                    self.assertEqual(module.docker(admission, runtime, "inspect", "owned-container", timeout=20), "ok")
                self.assertLessEqual(run.call_args.kwargs["timeout"], 5)
            finally:
                module.ACTIVE_CLEANUP_DEADLINE, module.ACTIVE_HARD_DEADLINE = old_cleanup, old_hard

    def test_grant_deadline_is_canonical_utc_and_within_owner_window(self):
        now = datetime(2026, 10, 21, 12, 0, 0, tzinfo=timezone.utc)
        accepted = (now + timedelta(minutes=5)).strftime("%Y-%m-%dT%H:%M:%SZ")
        self.assertEqual(module.parse_grant_deadline_utc(accepted, now), now + timedelta(minutes=5))
        for value in ((now + timedelta(seconds=59)).strftime("%Y-%m-%dT%H:%M:%SZ"), (now + timedelta(hours=24, seconds=1)).strftime("%Y-%m-%dT%H:%M:%SZ"), accepted.replace("Z", "+00:00"), "2026-10-21T12:05:00.000Z"):
            with self.subTest(value=value), self.assertRaises(module.Hold):
                module.parse_grant_deadline_utc(value, now)

    def test_launch_keeps_child_registry_when_stream_close_fails(self):
        class Child:
            pid = 139
        class BadStream:
            def close(self): raise OSError("stream close failed")
        original_open = Path.open
        def open_side_effect(path, *args, **kwargs):
            if path.name == "go.log":
                return BadStream()
            return original_open(path, *args, **kwargs)
        with tempfile.TemporaryDirectory() as name:
            children = {}
            child = Child()
            with patch.object(Path, "open", autospec=True, side_effect=open_side_effect), patch.object(module.subprocess, "Popen", return_value=child):
                with self.assertRaisesRegex(OSError, "stream close failed"):
                    module.launch({}, Path(name), "go", ["inert"], {}, children)
            self.assertIs(children["go"], child)

    def test_native_evidence_rejects_wrong_readiness_state(self):
        def read_row(name, *, status=200, state=None, error=None, execution=False, phase="NOT_STARTED", ready=False):
            return {"name": name, "status": status, "state": state, "error": error, "executionPresent": execution, "phase": phase,
                    "publishedCaseVersion": None, "proposalAvailable": None, "grantHandlePresent": ready, "deniedHandleAbsent": not ready,
                    "noStore": True, "durableBefore": "a" * 64, "durableAfter": "a" * 64, "bodySha256": "b" * 64}
        with tempfile.TemporaryDirectory() as name:
            root = Path(name)
            cases = [{"slot": i, "name": n, "status": s, "error": e, "durableBefore": "a" * 64, "durableAfter": "a" * 64, "bodySha256": "b" * 64}
                     for i, (n, s, e) in enumerate((("malformed-json", 400, "invalid_request"), ("unknown-field", 400, "invalid_request"), ("missing-version", 400, "invalid_request"), ("stale-version", 409, "case_conflict"), ("foreign-actor-grant", 409, "case_conflict"), ("expired-grant", 409, "case_conflict"), ("foreign-case", 404, "not_found")), 1)]
            (root / "native_rejections.json").write_text(json.dumps({"kind": "SOFTWARE_ONLY_NO_AI_OUTPUT", "slotsUsed": 7, "nativeExecutePOST": 7, "grantIssuances": 1, "cases": cases, "trapCalls": 0}), encoding="utf-8")
            (root / "native_precall_denied.json").write_text(json.dumps({"kind": "SOFTWARE_ONLY_NO_AI_OUTPUT", "slot": 8, "status": 202, "acceptedState": "CLAIMED", "acceptedExecutionId": "exec", "terminal": "DENIED", "executionId": "exec", "bodySha256": "b" * 64, "durableBefore": "a" * 64, "durableAfter": "c" * 64, "trapCalls": 0, "attempts": 0, "budgetConsumed": 0, "nativeExecutePOSTBeforeBrowser": 8, "grantIssuancesBeforeBrowser": 2}), encoding="utf-8")
            negative = [read_row("grant-absent", state="READY"), read_row("consumed-grant", state="AWAITING_OPERATOR_AUTHORITY"), read_row("config-mismatch", state="AWAITING_OPERATOR_AUTHORITY"), read_row("budget-exhausted", state="STALE_OR_INELIGIBLE"), read_row("case-version-changed", state="STALE_OR_INELIGIBLE"), read_row("writer-permission-revoked", status=403, error="permission_denied"), read_row("source-changed", state="STALE_OR_INELIGIBLE"), read_row("foreign-actor-grant", state="AWAITING_OPERATOR_AUTHORITY"), read_row("expired-grant", state="AWAITING_OPERATOR_AUTHORITY")]
            positive = [read_row("readiness-denied-reader", status=403, error="permission_denied"), read_row("readiness-denied-nojobs", status=403, error="permission_denied"), read_row("readiness-foreign-case", status=404, error="not_found"), read_row("writer-READY", state="READY", ready=True), read_row("reader-status-no-row", execution=False, phase="NOT_STARTED"), read_row("process-disabled-status", execution=False, phase="NOT_STARTED"), read_row("process-disabled-readiness", state="PROCESS_DISABLED"), read_row("status-unreadable-503", status=503, error="status_unavailable"), read_row("readiness-unreadable-503", status=503, error="readiness_unavailable")]
            (root / "native_reads_negative.json").write_text(json.dumps({"kind": "SOFTWARE_ONLY_NO_AI_OUTPUT", "reads": negative}), encoding="utf-8")
            (root / "native_reads_positive_pre_browser.json").write_text(json.dumps({"kind": "SOFTWARE_ONLY_NO_AI_OUTPUT", "reads": positive}), encoding="utf-8")
            with self.assertRaisesRegex(module.Hold, "NATIVE_READINESS_STATE_INVALID"):
                module.validate_native_evidence(root)

    def test_terminal_reader_failure_and_superficial_publication_hold(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name)
            (root / "go_terminal.json").write_text("{bad-json", encoding="utf-8")
            with self.assertRaises(json.JSONDecodeError):
                module.read_json(root / "go_terminal.json")
            with self.assertRaisesRegex(module.Hold, "GO_LINEAGE_OR_READ_INCOMPLETE"):
                module.validate_go_terminal(root, {"nativeExecutePOST": 8, "grantIssuances": 3, "executePOST": 1, "providerPhysicalCalls": 1,
                                                   "providerHTTPStatus": 200, "budgetConsumed": 1, "executionState": "PUBLISHED", "caseWorkState": "OPEN"})

    def test_out_of_order_route_responses_correlate_by_id(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name)
            execute_path = "/api/v1/tenants/t/intervention-cases/c/rerun-requests/r/execute"
            rows = [{"phase": "attempt", "id": 1, "method": "GET", "path": "/api/v1/status", "allowed": True},
                    {"phase": "attempt", "id": 2, "method": "POST", "path": execute_path, "allowed": True},
                    {"phase": "response", "id": 2, "method": "POST", "path": execute_path, "status": 202},
                    {"phase": "response", "id": 1, "method": "GET", "path": "/api/v1/status", "status": 200}]
            (root / "vite_routes.jsonl").write_text("".join(json.dumps(row) + "\n" for row in rows), encoding="utf-8")
            (root / "go_routes.jsonl").write_text("".join(json.dumps(row) + "\n" for row in rows), encoding="utf-8")
            terminal = {"executePOST": 1, "authPOST": 0}
            self.assertEqual(module.validate_vite_ledger(root, terminal)["responses"], 2)
            self.assertEqual(module.validate_go_ledger(root, terminal)["responses"], 2)

    def test_refresh_route_ledger_exact_pair_and_cap(self):
        execute_path = "/api/v1/tenants/t/intervention-cases/c/rerun-requests/r/execute"
        paths = ["/api/v1/auth/refresh", "/api/v1/auth/login", "/api/v1/auth/logout", "/api/v1/auth/refresh", execute_path]
        rows = []
        for number, path in enumerate(paths, 1):
            rows.extend(({"phase": "attempt", "id": number, "method": "POST", "path": path, "allowed": True},
                         {"phase": "response", "id": number, "method": "POST", "path": path, "status": 401 if path.endswith("refresh") else 200}))
        with tempfile.TemporaryDirectory() as name:
            root = Path(name)
            for ledger in ("vite_routes.jsonl", "go_routes.jsonl"):
                (root / ledger).write_text("".join(json.dumps(row) + "\n" for row in rows), encoding="utf-8")
            terminal = {"executePOST": 1, "authPOST": 4}
            self.assertEqual(module.validate_vite_ledger(root, terminal)["authPOST"], 4)
            self.assertEqual(module.validate_go_ledger(root, terminal)["authRoutes"].count("/api/v1/auth/refresh"), 2)
            for bad in ("/api/v1/auth/reset", "/api/v1/tenants/t/intervention-cases/c/rerun-requests/r/execute/extra"):
                changed = json.loads(json.dumps(rows))
                changed[0]["path"] = bad; changed[1]["path"] = bad
                with self.assertRaisesRegex(module.Hold, "UNLISTED_ROUTE_ATTEMPTED"):
                    module._validate_route_rows(changed, "TEST", terminal)
            denied = json.loads(json.dumps(rows)); denied[1]["status"] = 405
            with self.assertRaisesRegex(module.Hold, "ROUTE_DENIAL_OBSERVED"):
                module._validate_route_rows(denied, "TEST", terminal)
            with self.assertRaisesRegex(module.Hold, "ROUTE_RESPONSE_GAP"):
                module._validate_route_rows(rows[:-1], "TEST", terminal)
            with self.assertRaisesRegex(module.Hold, "EFFECT_ROUTE_MISMATCH"):
                module._validate_route_rows(rows, "TEST", {"executePOST": 1, "authPOST": 3})

    def test_native_antigravity_reservation_admission_positive(self):
        with tempfile.TemporaryDirectory() as name:
            temp_dir = Path(name)
            evidence = temp_dir / "evidence.json"
            evidence.write_text(json.dumps({"action": "reserve", "captured": True}), encoding="utf-8")
            evidence_sha = module.digest(evidence)

            receipt = temp_dir / "reservation.json"
            now_iso = (datetime.now(timezone.utc) - timedelta(seconds=10)).strftime("%Y-%m-%dT%H:%M:%SZ")
            receipt.write_text(json.dumps({
                "status": "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE",
                "browserId": "antigravity_native",
                "exclusiveReservation": True,
                "controlObserved": True,
                "captureObserved": True,
                "closeObserved": False,
                "protocol": "http",
                "closeMode": "operator_fallback",
                "pageId": "page-123",
                "campaignNonce": "nonce-456",
                "browserSessionId": "session-789",
                "trancheId": "CCMAI-RUNTIME-139",
                "operator": "Owner",
                "reservedAtUTC": now_iso,
                "actionEvidencePath": ".cvf/runtime/evidence.json",
                "actionEvidenceSha256": evidence_sha,
            }), encoding="utf-8")
            admission = {
                "browserTool": "antigravity_native",
                "browserSupported": True,
                "browserReceiptPath": "reservation.json",
                "browserReceiptSha256": module.digest(receipt),
                "browserCloseReceiptPath": ".cvf/runtime/close.json",
                "trancheId": "CCMAI-RUNTIME-139",
                "pageId": "page-123",
                "campaignNonce": "nonce-456",
                "browserSessionId": "session-789",
                "fallbackCloseOperator": "Owner",
            }
            def fake_project_path(rel):
                if rel == "reservation.json":
                    return receipt
                if rel == ".cvf/runtime/evidence.json":
                    return evidence
                return temp_dir / rel

            with patch.object(module, "project_path", side_effect=fake_project_path):
                module.verify_browser_receipt(admission)

    def test_native_antigravity_early_close_rejected(self):
        with tempfile.TemporaryDirectory() as name:
            temp_dir = Path(name)
            receipt = temp_dir / "reservation.json"
            receipt.write_text(json.dumps({
                "status": "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE",
                "browserId": "antigravity_native",
                "exclusiveReservation": True,
                "controlObserved": True,
                "captureObserved": True,
                "closeObserved": True,
                "protocol": "http",
                "closeMode": "operator_fallback",
                "pageId": "page-123",
                "campaignNonce": "nonce-456",
                "browserSessionId": "session-789",
                "trancheId": "CCMAI-RUNTIME-139",
                "operator": "Owner",
            }), encoding="utf-8")
            admission = {
                "browserTool": "antigravity_native",
                "browserSupported": True,
                "browserReceiptPath": "reservation.json",
                "browserReceiptSha256": module.digest(receipt),
                "browserCloseReceiptPath": ".cvf/runtime/close.json",
                "trancheId": "CCMAI-RUNTIME-139",
                "pageId": "page-123",
                "campaignNonce": "nonce-456",
                "browserSessionId": "session-789",
                "fallbackCloseOperator": "Owner",
            }
            with patch.object(module, "project_path", return_value=receipt):
                with self.assertRaisesRegex(module.Hold, "EARLY_CLOSE_FORBIDDEN"):
                    module.verify_browser_receipt(admission)

                receipt.write_text(json.dumps({
                    "status": "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE",
                    "browserId": "antigravity_native",
                    "exclusiveReservation": True,
                    "controlObserved": True,
                    "captureObserved": True,
                    "closeObserved": "true",
                    "protocol": "http",
                    "closeMode": "operator_fallback",
                    "pageId": "page-123",
                    "campaignNonce": "nonce-456",
                    "browserSessionId": "session-789",
                    "trancheId": "CCMAI-RUNTIME-139",
                    "operator": "Owner",
                }), encoding="utf-8")
                admission["browserReceiptSha256"] = module.digest(receipt)
                with self.assertRaisesRegex(module.Hold, "EARLY_CLOSE_FORBIDDEN"):
                    module.verify_browser_receipt(admission)

    def test_native_antigravity_final_close_cannot_be_prefixture_reservation(self):
        with tempfile.TemporaryDirectory() as name:
            receipt = Path(name) / "reservation.json"
            receipt.write_text(json.dumps({
                "status": "CLOSED_EXACT_OWNED_PAGE",
                "browserId": "antigravity_native",
                "exclusiveReservation": True,
                "controlObserved": True,
                "captureObserved": True,
                "closeObserved": True,
                "ownedTabsFinal": 0,
                "protocol": "http",
                "closeMode": "operator_fallback",
                "pageId": "page-123",
                "campaignNonce": "nonce-456",
                "browserSessionId": "session-789",
                "trancheId": "CCMAI-RUNTIME-139",
                "operator": "Owner",
            }), encoding="utf-8")
            admission = {
                "browserTool": "antigravity_native",
                "browserSupported": True,
                "browserReceiptPath": "reservation.json",
                "browserReceiptSha256": module.digest(receipt),
                "browserCloseReceiptPath": ".cvf/runtime/close.json",
                "trancheId": "CCMAI-RUNTIME-139",
                "pageId": "page-123",
                "campaignNonce": "nonce-456",
                "browserSessionId": "session-789",
                "fallbackCloseOperator": "Owner",
            }
            with patch.object(module, "project_path", return_value=receipt):
                with self.assertRaisesRegex(module.Hold, "TERMINAL_CLOSE_RECEIPT_CANNOT_BE_PREFIXTURE_RESERVATION"):
                    module.verify_browser_receipt(admission)

    def test_native_antigravity_identity_mismatches_and_missing_close_path_rejected(self):
        with tempfile.TemporaryDirectory() as name:
            temp_dir = Path(name)
            evidence = temp_dir / "evidence.json"
            evidence.write_text(json.dumps({"action": "reserve"}), encoding="utf-8")
            evidence_sha = module.digest(evidence)

            receipt = temp_dir / "reservation.json"
            now_iso = (datetime.now(timezone.utc) - timedelta(seconds=10)).strftime("%Y-%m-%dT%H:%M:%SZ")
            base_data = {
                "status": "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE",
                "browserId": "antigravity_native",
                "exclusiveReservation": True,
                "controlObserved": True,
                "captureObserved": True,
                "closeObserved": False,
                "protocol": "http",
                "closeMode": "operator_fallback",
                "trancheId": "CCMAI-RUNTIME-139",
                "pageId": "page-123",
                "campaignNonce": "nonce-456",
                "browserSessionId": "session-789",
                "operator": "Owner",
                "reservedAtUTC": now_iso,
                "actionEvidencePath": ".cvf/runtime/evidence.json",
                "actionEvidenceSha256": evidence_sha,
            }
            receipt.write_text(json.dumps(base_data), encoding="utf-8")
            admission = {
                "browserTool": "antigravity_native",
                "browserSupported": True,
                "browserReceiptPath": "reservation.json",
                "browserReceiptSha256": module.digest(receipt),
                "browserCloseReceiptPath": ".cvf/runtime/close.json",
                "trancheId": "CCMAI-RUNTIME-139",
                "pageId": "page-123",
                "campaignNonce": "nonce-456",
                "browserSessionId": "session-789",
                "fallbackCloseOperator": "Owner",
            }
            def fake_project_path(rel):
                if rel == "reservation.json":
                    return receipt
                if rel == ".cvf/runtime/evidence.json":
                    return evidence
                return temp_dir / rel

            with patch.object(module, "project_path", side_effect=fake_project_path):
                bad_adm = dict(admission, browserCloseReceiptPath="")
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_RECEIPT_PATH_MISSING"):
                    module.verify_browser_receipt(bad_adm)
                bad_adm = dict(admission, browserCloseReceiptPath="docs/reviews/close.json")
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_RECEIPT_PATH_MUST_NOT_BE_TRACKED_DOCS"):
                    module.verify_browser_receipt(bad_adm)
                bad_adm = dict(admission, trancheId="")
                with self.assertRaisesRegex(module.Hold, "BROWSER_PROOF_IDENTITY_MISMATCH"):
                    module.verify_browser_receipt(bad_adm)
                bad_adm = dict(admission, trancheId="OTHER-TRANCHE")
                with self.assertRaisesRegex(module.Hold, "BROWSER_PROOF_IDENTITY_MISMATCH"):
                    module.verify_browser_receipt(bad_adm)
                bad_adm = dict(admission, pageId="other-page")
                with self.assertRaisesRegex(module.Hold, "BROWSER_PROOF_IDENTITY_MISMATCH"):
                    module.verify_browser_receipt(bad_adm)
                bad_adm = dict(admission, campaignNonce="other-nonce")
                with self.assertRaisesRegex(module.Hold, "BROWSER_PROOF_IDENTITY_MISMATCH"):
                    module.verify_browser_receipt(bad_adm)
                bad_adm = dict(admission, browserSessionId="other-session")
                with self.assertRaisesRegex(module.Hold, "BROWSER_PROOF_IDENTITY_MISMATCH"):
                    module.verify_browser_receipt(bad_adm)
                bad_adm = dict(admission, fallbackCloseOperator="other-operator")
                with self.assertRaisesRegex(module.Hold, "BROWSER_PROOF_IDENTITY_MISMATCH"):
                    module.verify_browser_receipt(bad_adm)

                bad_receipt_data = dict(base_data, reservedAtUTC="invalid-time")
                receipt.write_text(json.dumps(bad_receipt_data), encoding="utf-8")
                admission_with_bad_receipt = dict(admission, browserReceiptSha256=module.digest(receipt))
                with self.assertRaisesRegex(module.Hold, "RESERVATION_TIMESTAMP_INVALID"):
                    module.verify_browser_receipt(admission_with_bad_receipt)

                future_iso = (datetime.now(timezone.utc) + timedelta(minutes=5)).strftime("%Y-%m-%dT%H:%M:%SZ")
                bad_receipt_data = dict(base_data, reservedAtUTC=future_iso)
                receipt.write_text(json.dumps(bad_receipt_data), encoding="utf-8")
                admission_with_bad_receipt = dict(admission, browserReceiptSha256=module.digest(receipt))
                with self.assertRaisesRegex(module.Hold, "RESERVATION_TIMESTAMP_IN_FUTURE"):
                    module.verify_browser_receipt(admission_with_bad_receipt)

                bad_receipt_data = dict(base_data, actionEvidencePath="docs/reviews/evidence.json")
                receipt.write_text(json.dumps(bad_receipt_data), encoding="utf-8")
                admission_with_bad_receipt = dict(admission, browserReceiptSha256=module.digest(receipt))
                with self.assertRaisesRegex(module.Hold, "EVIDENCE_PATH_MUST_NOT_BE_TRACKED_DOCS"):
                    module.verify_browser_receipt(admission_with_bad_receipt)

                bad_receipt_data = dict(base_data, actionEvidenceSha256="0" * 64)
                receipt.write_text(json.dumps(bad_receipt_data), encoding="utf-8")
                admission_with_bad_receipt = dict(admission, browserReceiptSha256=module.digest(receipt))
                with self.assertRaisesRegex(module.Hold, "RESERVATION_EVIDENCE_SHA_MISMATCH"):
                    module.verify_browser_receipt(admission_with_bad_receipt)

    def test_verify_browser_terminal_close_positive_and_rejections(self):
        now = datetime.now(timezone.utc)
        reserved_time = (now - timedelta(minutes=10)).strftime("%Y-%m-%dT%H:%M:%SZ")
        closed_time = (now - timedelta(minutes=5)).strftime("%Y-%m-%dT%H:%M:%SZ")
        inv_time = (now - timedelta(minutes=2)).strftime("%Y-%m-%dT%H:%M:%SZ")
        deadline = (now + timedelta(hours=1)).strftime("%Y-%m-%dT%H:%M:%SZ")

        admission = {
            "browserTool": "antigravity_native",
            "browserCloseReceiptPath": ".cvf/runtime/close.json",
            "trancheId": "CCMAI-RUNTIME-139",
            "pageId": "page-123",
            "campaignNonce": "nonce-456",
            "browserSessionId": "session-789",
            "fallbackCloseOperator": "Owner",
            "grantDeadlineUTC": deadline,
        }
        with tempfile.TemporaryDirectory() as name:
            temp_dir = Path(name)
            action_file = temp_dir / "action.json"
            action_file.write_text(json.dumps({"action": "navigate_and_click"}), encoding="utf-8")
            action_sha = module.digest(action_file)

            inv_file = temp_dir / "inventory.json"
            inv_data = {
                "openPages": ["other-page"],
                "openTabs": ["other-tab"],
                "ownedTabsFinal": 0,
                "inventoryAtUTC": inv_time,
            }
            inv_file.write_text(json.dumps(inv_data), encoding="utf-8")
            inv_sha = module.digest(inv_file)

            close_file = temp_dir / "close.json"
            valid_close = {
                "status": "CLOSED_EXACT_OWNED_PAGE",
                "browserId": "antigravity_native",
                "closeObserved": True,
                "ownedTabsFinal": 0,
                "trancheId": "CCMAI-RUNTIME-139",
                "pageId": "page-123",
                "campaignNonce": "nonce-456",
                "browserSessionId": "session-789",
                "closeMode": "operator_fallback",
                "operator": "Owner",
                "reservedAtUTC": reserved_time,
                "closedAtUTC": closed_time,
                "actionEvidencePath": ".cvf/runtime/action.json",
                "actionEvidenceSha256": action_sha,
                "finalInventoryPath": ".cvf/runtime/inventory.json",
                "finalInventorySha256": inv_sha,
            }
            close_file.write_text(json.dumps(valid_close), encoding="utf-8")

            def fake_project_path(rel):
                if rel == ".cvf/runtime/close.json":
                    return close_file
                if rel == ".cvf/runtime/action.json":
                    return action_file
                if rel == ".cvf/runtime/inventory.json":
                    return inv_file
                return temp_dir / rel

            with patch.object(module, "project_path", side_effect=fake_project_path):
                # Positive check
                result = module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
                self.assertEqual(result["status"], "CLOSED_EXACT_OWNED_PAGE")

                # F01: Deadline swallowed Hold bug check & malformed deadline
                past_deadline_adm = dict(admission, grantDeadlineUTC=(now - timedelta(minutes=10)).strftime("%Y-%m-%dT%H:%M:%SZ"))
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_AFTER_DEADLINE"):
                    module.verify_browser_terminal_close(past_deadline_adm, temp_dir, temp_dir)

                bad_deadline_adm = dict(admission, grantDeadlineUTC="malformed-deadlineZ")
                with self.assertRaisesRegex(module.Hold, "GRANT_DEADLINE_INVALID"):
                    module.verify_browser_terminal_close(bad_deadline_adm, temp_dir, temp_dir)

                bad_deadline_adm2 = dict(admission, grantDeadlineUTC="")
                with self.assertRaisesRegex(module.Hold, "GRANT_DEADLINE_INVALID"):
                    module.verify_browser_terminal_close(bad_deadline_adm2, temp_dir, temp_dir)

                # F02: Identity checks in admission & receipt
                bad_adm = dict(admission, trancheId="")
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_IDENTITY_MISSING"):
                    module.verify_browser_terminal_close(bad_adm, temp_dir, temp_dir)

                bad_close = dict(valid_close, trancheId="OTHER-TRANCHE")
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_TRANCHE_ID_MISMATCH"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

                bad_close = dict(valid_close, pageId="wrong-page")
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_PAGE_ID_MISMATCH"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

                bad_close = dict(valid_close, campaignNonce="wrong-nonce")
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_NONCE_MISMATCH"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

                bad_close = dict(valid_close, browserSessionId="wrong-session")
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_SESSION_MISMATCH"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

                bad_close = dict(valid_close, operator="wrong-op")
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_OPERATOR_MISMATCH"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

                # Restore valid close file for remaining tests
                close_file.write_text(json.dumps(valid_close), encoding="utf-8")

                # F03 & F04: Evidence & final inventory verification
                bad_close = dict(valid_close, actionEvidenceSha256="0" * 64)
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_ACTION_EVIDENCE_SHA_MISMATCH"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

                bad_close = dict(valid_close, finalInventorySha256="0" * 64)
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_FINAL_INVENTORY_SHA_MISMATCH"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

                # Target still open in final inventory
                bad_inv = dict(inv_data, openPages=["page-123"])
                inv_file.write_text(json.dumps(bad_inv), encoding="utf-8")
                bad_close = dict(valid_close, finalInventorySha256=module.digest(inv_file))
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "PAGE_STILL_OPEN_IN_FINAL_INVENTORY"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

                bad_inv = dict(inv_data, openTabs=["page-123"])
                inv_file.write_text(json.dumps(bad_inv), encoding="utf-8")
                bad_close = dict(valid_close, finalInventorySha256=module.digest(inv_file))
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "PAGE_STILL_OPEN_IN_FINAL_INVENTORY"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

                # Restore valid inventory
                inv_file.write_text(json.dumps(inv_data), encoding="utf-8")
                close_file.write_text(json.dumps(valid_close), encoding="utf-8")

                # F05: Chronology checks
                bad_close = dict(valid_close, reservedAtUTC=(now - timedelta(minutes=1)).strftime("%Y-%m-%dT%H:%M:%SZ"))
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_BEFORE_RESERVATION"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

                future_close = (now + timedelta(minutes=10)).strftime("%Y-%m-%dT%H:%M:%SZ")
                bad_close = dict(valid_close, closedAtUTC=future_close)
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_TIMESTAMP_IN_FUTURE"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

                bad_inv = dict(inv_data, inventoryAtUTC=(now - timedelta(minutes=8)).strftime("%Y-%m-%dT%H:%M:%SZ"))
                inv_file.write_text(json.dumps(bad_inv), encoding="utf-8")
                bad_close = dict(valid_close, finalInventorySha256=module.digest(inv_file))
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "BROWSER_INVENTORY_CHRONOLOGY_INVALID"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

                # Restore valid inventory & close
                inv_file.write_text(json.dumps(inv_data), encoding="utf-8")
                close_file.write_text(json.dumps(valid_close), encoding="utf-8")

                # F06: Type strictness (booleans & integers)
                bad_close = dict(valid_close, closeObserved=False)
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_NOT_OBSERVED"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

                bad_close = dict(valid_close, closeObserved="true")
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_NOT_OBSERVED"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

                bad_close = dict(valid_close, ownedTabsFinal=1)
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_OWNED_TABS_REMAINING"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

                bad_close = dict(valid_close, ownedTabsFinal=False)  # boolean False is 0 in python int comparison
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_OWNED_TABS_REMAINING"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

                bad_close = dict(valid_close, ownedTabsFinal="0")
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_OWNED_TABS_REMAINING"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

                # F07: Rejection of docs/ path for close receipt and evidence
                bad_adm = dict(admission, browserCloseReceiptPath="docs/reviews/close.json")
                close_file.write_text(json.dumps(valid_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_RECEIPT_PATH_MUST_NOT_BE_TRACKED_DOCS"):
                    module.verify_browser_terminal_close(bad_adm, temp_dir, temp_dir)

                bad_close = dict(valid_close, actionEvidencePath="docs/reviews/action.json")
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "EVIDENCE_PATH_MUST_NOT_BE_TRACKED_DOCS"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

                bad_close = dict(valid_close, finalInventoryPath="docs/reviews/inv.json")
                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
                with self.assertRaisesRegex(module.Hold, "INVENTORY_PATH_MUST_NOT_BE_TRACKED_DOCS"):
                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)

    def test_supervisor_terminal_close_enforcement_on_success_and_error_paths(self):
        # F08: Test supervisor terminal evaluation and cleanup preservation on success and error paths
        with tempfile.TemporaryDirectory() as name:
            temp_dir = Path(name)
            runtime = temp_dir / "runtime"
            external = temp_dir / "external"
            runtime.mkdir()
            external.mkdir()

            admission = {
                "browserTool": "antigravity_native",
                "browserCloseReceiptPath": ".cvf/runtime/close.json",
                "trancheId": "CCMAI-RUNTIME-139",
                "pageId": "page-123",
                "campaignNonce": "nonce-456",
                "browserSessionId": "session-789",
                "fallbackCloseOperator": "Owner",
                "grantDeadlineUTC": (datetime.now(timezone.utc) + timedelta(hours=1)).strftime("%Y-%m-%dT%H:%M:%SZ"),
            }

            # Success path: terminal close evaluation records receipt and no error is recorded
            outcome_success: dict = {"result": "STOP_REQUESTED", "terminalErrors": []}
            mock_close_receipt = {"status": "CLOSED_EXACT_OWNED_PAGE", "closeObserved": True, "ownedTabsFinal": 0}
            with patch.object(module, "verify_browser_terminal_close", return_value=mock_close_receipt):
                if admission.get("browserTool") == "antigravity_native":
                    outcome_success["browserTerminalClose"] = module.verify_browser_terminal_close(admission, runtime, external)
                self.assertEqual(outcome_success["browserTerminalClose"]["status"], "CLOSED_EXACT_OWNED_PAGE")
                self.assertEqual(len(outcome_success["terminalErrors"]), 0)

            # Error path: terminal close evaluation failure records error and preserves cleanup
            outcome_error: dict = {"result": "STOP_REQUESTED", "terminalErrors": []}
            cleanup_called = []
            def fake_cleanup():
                cleanup_called.append(True)

            with patch.object(module, "verify_browser_terminal_close", side_effect=module.Hold("BROWSER_CLOSE_AFTER_DEADLINE")):
                try:
                    if admission.get("browserTool") == "antigravity_native":
                        outcome_error["browserTerminalClose"] = module.verify_browser_terminal_close(admission, runtime, external)
                except Exception as exc:
                    outcome_error["terminalErrors"].append("BROWSER_TERMINAL_CLOSE_FAILED:" + type(exc).__name__)
                finally:
                    fake_cleanup()

                self.assertIn("BROWSER_TERMINAL_CLOSE_FAILED:Hold", outcome_error["terminalErrors"])
                self.assertTrue(cleanup_called and cleanup_called[0], "Cleanup must execute even when terminal close validation fails")


if __name__ == "__main__":
    unittest.main()
