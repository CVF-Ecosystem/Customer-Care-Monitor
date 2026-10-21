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
        }
        result = module.excluded_owned(observed, {"containerId": "new", "networkId": "new-net", "volumeName": "ccmai-r139-owned"})
        self.assertEqual(result, {"containers": observed["containers"][:1], "networks": observed["networks"][:1], "volumes": observed["volumes"][:1]})
        self.assertEqual(len(observed["containers"]), 2)

    def test_protected_inventory_normalizes_network_order_and_status_elapsed_only(self):
        before = {
            "containers": [{"ID": "protected-id", "Networks": "bridge, custom", "Status": "Up 2 minutes", "State": "running", "RunningFor": "3 minutes"}],
            "networks": [{"ID": "network-id", "Name": "protected"}],
            "volumes": [{"Name": "protected-volume"}],
        }
        after = {
            "containers": [{"ID": "protected-id", "Networks": "custom,bridge", "Status": "Up 2 minutes", "State": "running", "RunningFor": "3 minutes"}],
            "networks": [{"ID": "network-id", "Name": "protected"}],
            "volumes": [{"Name": "protected-volume"}],
        }
        original_before = json.loads(json.dumps(before))
        self.assertEqual(module.canonicalize_protected_inventory(before), module.canonicalize_protected_inventory(after))
        self.assertEqual(before, original_before)
        changed_status = json.loads(json.dumps(after))
        changed_status["containers"][0]["Status"] = "Up 4 minutes"
        self.assertEqual(module.canonicalize_protected_inventory(before), module.canonicalize_protected_inventory(changed_status))
        self.assertEqual(changed_status["containers"][0]["Status"], "Up 4 minutes")

    def test_protected_status_preserves_state_health_exitcode_and_other_fields(self):
        base = {"containers": [{"ID": "x", "Status": "Up 29 minutes (healthy)", "RunningFor": "fixed", "State": "running", "ExitCode": 0}], "networks": [], "volumes": []}
        elapsed = json.loads(json.dumps(base))
        elapsed["containers"][0]["Status"] = "Up 36 minutes (healthy)"
        self.assertEqual(module.canonicalize_protected_inventory(base), module.canonicalize_protected_inventory(elapsed))
        for field, value in (("Status", "Up 36 minutes (unhealthy)"), ("ExitCode", 1), ("RunningFor", "changed"), ("ID", "y")):
            changed = json.loads(json.dumps(elapsed))
            changed["containers"][0][field] = value
            self.assertNotEqual(module.canonicalize_protected_inventory(base), module.canonicalize_protected_inventory(changed), field)
        exited = json.loads(json.dumps(base))
        exited["containers"][0]["Status"] = "Exited (0) 22 hours ago"
        exited["containers"][0]["State"] = "exited"
        exited_later = json.loads(json.dumps(exited))
        exited_later["containers"][0]["Status"] = "Exited (0) 23 hours ago"
        self.assertEqual(module.canonicalize_protected_inventory(exited), module.canonicalize_protected_inventory(exited_later))
        exited_later["containers"][0]["Status"] = "Exited (137) 23 hours ago"
        self.assertNotEqual(module.canonicalize_protected_inventory(exited), module.canonicalize_protected_inventory(exited_later))
        mismatch = json.loads(json.dumps(base))
        mismatch["containers"][0]["State"] = "exited"
        with self.assertRaisesRegex(module.Hold, "PROTECTED_STATUS_STATE_MISMATCH"):
            module.canonicalize_protected_inventory(mismatch)

    def test_protected_status_unknown_formats_hold(self):
        for status in ("Restarting (1) 3 seconds ago", "Dead", "Up About a minute", "Exited (0) unknown ago", "Up 2 minutes (health: unknown)", ""):
            with self.subTest(status=status), self.assertRaisesRegex(module.Hold, "PROTECTED_STATUS_UNKNOWN_FORMAT"):
                module.canonicalize_protected_inventory({"containers": [{"ID": "x", "Status": status, "State": "running"}], "networks": [], "volumes": []})

    def test_mysql_readiness_uses_tcp_matching_database_and_retries_not_ready_only(self):
        admission = {"database": "ccma_r139_r2_20261021"}
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
                module.wait_mysql_tcp_ready({"database": "ccma_r139_r2_20261021"}, Path("ignored-runtime"), {"containerId": "owned", "fixturePassword": "synthetic"})
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
                    {"database": "ccma_r139_r2_20261021"},
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
                module.wait_mysql_tcp_ready({"database": "ccma_r139_r2_20261021"}, Path("ignored-runtime"), {"containerId": "owned", "fixturePassword": "synthetic"})
        self.assertEqual(docker.call_count, 45)

        clock = FakeClock(100.0)
        with patch.object(module, "ACTIVE_HARD_DEADLINE", 280.5), patch.object(
            module.time, "monotonic", side_effect=clock.monotonic
        ), patch.object(module.time, "sleep", side_effect=clock.sleep), patch.object(
            module, "docker", side_effect=module.Hold("DOCKER_COMMAND_FAILED")
        ) as docker:
            with self.assertRaisesRegex(module.Hold, "MYSQL_TCP_READY_NOT_PROVEN"):
                module.wait_mysql_tcp_ready({"database": "ccma_r139_r2_20261021"}, Path("ignored-runtime"), {"containerId": "owned", "fixturePassword": "synthetic"})
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

    def test_replacement_evidence_path_is_fixture_r2(self):
        with tempfile.TemporaryDirectory() as name:
            project = Path(name) / "Customer-Care-Monitor-AI"
            project.mkdir()
            with patch.object(module, "ROOT", project):
                self.assertEqual(
                    module.replacement_external_evidence_dir(),
                    project.parent / ".ccmai_local_evidence" / "r139_execution_proof_20261021" / "fixture_r2",
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
            before = {"containers": [], "networks": [], "volumes": []}
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
            rows = [{"phase": "attempt", "id": 1, "method": "GET", "path": "/api/v1/status", "allowed": True},
                    {"phase": "attempt", "id": 2, "method": "POST", "path": "/api/v1/x/execute", "allowed": True},
                    {"phase": "response", "id": 2, "method": "POST", "path": "/api/v1/x/execute", "status": 202},
                    {"phase": "response", "id": 1, "method": "GET", "path": "/api/v1/status", "status": 200}]
            (root / "vite_routes.jsonl").write_text("".join(json.dumps(row) + "\n" for row in rows), encoding="utf-8")
            (root / "go_routes.jsonl").write_text("".join(json.dumps(row) + "\n" for row in rows), encoding="utf-8")
            terminal = {"executePOST": 1, "authPOST": 0}
            self.assertEqual(module.validate_vite_ledger(root, terminal)["responses"], 2)
            self.assertEqual(module.validate_go_ledger(root, terminal)["responses"], 2)


if __name__ == "__main__":
    unittest.main()
