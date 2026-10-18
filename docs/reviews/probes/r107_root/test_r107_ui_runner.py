"""Offline-only tests for the inert R107 source candidate."""

import hashlib
import importlib.util
import io
import json
import queue
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

SOURCE = Path(__file__).with_name("r107_ui_runner.py")
spec = importlib.util.spec_from_file_location("r107_ui_runner", SOURCE)
runner = importlib.util.module_from_spec(spec)
spec.loader.exec_module(runner)


def place_history(root, place):
    for effect in runner.EFFECTS:
        place(runner.PARENT_RUNTIME + "/" + effect + ".json", json.dumps({"effect": effect, "state": "RESERVED", "replayAllowed": False}).encode())
    place(runner.PARENT_RUNTIME + "/cleanup.json", json.dumps({"errors": [], "fixtureStopAttempted": True, "ownedProcessStopAttempted": True, "backendExitCode": 1, "viteExitCode": 1}).encode())


class CandidateTests(unittest.TestCase):
    def test_formal_admission_packet_and_history_binding(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            def place(relative, content):
                path = root / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(content)
                return path
            for name in (runner.RUNNER, runner.TEMPLATE, runner.FRONTEND_MANIFEST, runner.HISTORICAL_MANIFEST):
                place(name, b"pinned input\n")
            place(".cvf/runtime/r101_worker/r101_handlers.test.exe", b"injected binary")
            docker = place("docker.exe", b"injected docker")
            place(runner.ORDER, b"Status: ACTIVE\n")
            capability = {
                "tool": "mcp__cua_repl.js", "browser": "iab",
                "http": {"capture": "PASS", "closedCall": "SUCCESS", "after": "button Control OK"},
                "productEffects": {name: 0 for name in ("fixtureStarts", "backendLaunches", "viteLaunches", "applicationActions", "providerCalls", "mockCalls")},
            }
            capability_bytes = (json.dumps(capability) + "\n").encode()
            place(runner.CAPABILITY_RECEIPT, capability_bytes)
            roles = {"implementationWorker": runner.WORKER, "reviewer": "Codex /root"}
            review = {"disposition": "SOURCE_REVIEW_PASS_RUNTIME_NOT_RUN", "reviewer": "Codex /root", "sourceCandidateAuthor": runner.WORKER, "runtimePins": {name: runner.sha256(root / name) for name in (runner.RUNNER, runner.TEMPLATE, runner.FRONTEND_MANIFEST, runner.HISTORICAL_MANIFEST)}}
            review_bytes = (json.dumps(review) + "\n").encode()
            place(runner.REVIEW, review_bytes)
            allowed = ["docs/reviews/probes/r107_root/**", "docs/reviews/probes/r107_cua/**", ".cvf/runtime/r107_cua/**"]
            budget = {"existingFixtureStarts": 1, "historicalBackendLaunches": 1, "viteLaunches": 1, "runtimeRetries": 0, "wholeCampaignSeconds": 2100, "backendSeconds": 1800, "providerCalls": 0, "mockCalls": 0, "goCompiles": 0, "testBuildCommands": 0, "newFixtures": 0}
            seed = {"authorityKind": "DISPATCHER_OWNED_HISTORICAL_R101_UI_EVIDENCE", "riskCeiling": "R2", "trancheId": runner.TRANCHE, "worker": runner.WORKER, "browserTool": runner.BROWSER_TOOL, "browserSurface": runner.BROWSER_SURFACE, "capabilityReceipt": {"path": runner.CAPABILITY_RECEIPT, "sha256": runner.sha256(root / runner.CAPABILITY_RECEIPT)}, "roles": roles, "allowedPaths": allowed, "prohibitedEffects": [], "executionBudget": budget, "historicalTargetCommit": runner.TARGET, "binarySha256": runner.BINARY_SHA256, "nodeExe": "C:/Program Files/nodejs/node.exe", "nodeSha256": "3602f2bb1a10f2cbab4c36886218a33c1ab3db87290e73b033c46c77147d0237", "dockerExe": str(docker), "dockerSha256": runner.sha256(docker), "dockerHost": "npipe:////./pipe/dockerDesktopLinuxEngine", "runtimePins": {**review["runtimePins"], runner.REVIEW: runner.sha256(root / runner.REVIEW)}}
            seed_bytes = (json.dumps(seed) + "\n").encode()
            place(runner.SEED, seed_bytes.replace(b"\n", b"\r\n"))
            current_seed = {"bytes": seed_bytes.replace(b"\n", b"\r\n")}
            record = {"trancheId": runner.TRANCHE, "riskCeiling": "R2", "worker": runner.WORKER, "browserTool": runner.BROWSER_TOOL, "browserSurface": runner.BROWSER_SURFACE, "roles": roles, "allowedPaths": allowed, "prohibitedEffects": [], "baseCommit": "b" * 40, "status": "BUILD", "phase": "BUILD", "workOrder": runner.ORDER, "workOrderStatusGrammar": "cvf.workOrderStatusGrammar@1.0.0", "workOrderStatus": "ACTIVE"}
            place(runner.RECORD, json.dumps(record).encode())
            place(runner.STATE, json.dumps({"activeTranche": runner.TRANCHE, "activePhase": "BUILD", "currentMode": "BUILD"}).encode())
            parent_seed_bytes = b'{"trancheId":"CCMAI-RUNTIME-106"}\n'
            place(runner.PARENT_SEED, parent_seed_bytes)
            place(runner.PARENT_ORDER, b"Status: BLOCKED_WITH_REASON\n")
            parent_record = {"status": "BUILD_BLOCKED", "x-actualExecution": {"fixtureStarts": 1, "backendLaunches": 1, "viteLaunches": 1, "runtimeRetries": 0}}
            place(runner.PARENT_RECORD, json.dumps(parent_record).encode())
            place_history(root, place)
            seed["campaignLineage"] = {"parent": "CCMAI-RUNTIME-106", "parentSpent": {"fixtureStarts": 1, "backendLaunches": 1, "viteLaunches": 1}, "newDistinctCampaigns": 1, "predecessorPins": {name: runner.sha256(root / name) for name in [runner.PARENT_SEED, *[runner.PARENT_RUNTIME + "/" + effect + ".json" for effect in runner.EFFECTS], runner.PARENT_RUNTIME + "/cleanup.json"]}}
            module_path = "docs/reviews/probes/r107_root/module_validation.json"
            place(module_path, json.dumps({"disposition": "PASS", "exitCode": 0, "templateSha256": runner.sha256(root / runner.TEMPLATE)}).encode())
            seed["moduleValidation"] = {"path": module_path, "sha256": runner.sha256(root / module_path)}
            seed_bytes = (json.dumps(seed) + "\n").encode()
            place(runner.SEED, seed_bytes)
            current_seed["bytes"] = seed_bytes
            def fake_git(_root, *args):
                if args[0] == "log":
                    return ("c" * 40 + "\n").encode()
                if args[0] == "show":
                    path = args[1].split(":", 1)[1]
                    if path == runner.REVIEW:
                        return review_bytes
                    if path == runner.PARENT_SEED:
                        return parent_seed_bytes
                    return current_seed["bytes"]
                return b""
            original_sha = runner.sha256
            def pinned_sha(path):
                if str(path).replace("\\", "/") == seed["nodeExe"]:
                    return seed["nodeSha256"]
                if str(path).endswith("r101_handlers.test.exe"):
                    return runner.BINARY_SHA256
                return original_sha(path)
            with patch.object(runner, "sha256", side_effect=pinned_sha):
                self.assertEqual("b" * 40, runner.check_admission(root, git=fake_git)["baseCommit"])
                place(runner.ORDER, b"```\nStatus: ACTIVE\n```\n")
                with self.assertRaises(runner.Hold):
                    runner.check_admission(root, git=fake_git)
                place(runner.ORDER, b"Status: ACTIVE\n")
                record["workOrder"] = "docs/work_orders/wrong.md"
                place(runner.RECORD, json.dumps(record).encode())
                with self.assertRaises(runner.Hold):
                    runner.check_admission(root, git=fake_git)

                record["workOrder"] = runner.ORDER
                def persist_packet():
                    current_seed["bytes"] = (json.dumps(seed) + "\n").encode()
                    place(runner.SEED, current_seed["bytes"])
                    place(runner.RECORD, json.dumps(record).encode())
                for field, bad in (("worker", "someone-else"), ("browserTool", "other-tool"), ("browserSurface", "other-surface")):
                    with self.subTest(binding=field):
                        seed[field] = bad
                        record[field] = bad
                        persist_packet()
                        with self.assertRaises(runner.Hold):
                            runner.check_admission(root, git=fake_git)
                        seed[field] = getattr(runner, {"worker": "WORKER", "browserTool": "BROWSER_TOOL", "browserSurface": "BROWSER_SURFACE"}[field])
                        record[field] = seed[field]
                wrong_paths = ["docs/reviews/probes/r105_root/**", "docs/reviews/probes/r105_claude/**", ".cvf/runtime/r105_claude/**"]
                seed["allowedPaths"] = wrong_paths
                record["allowedPaths"] = wrong_paths
                persist_packet()
                with self.assertRaises(runner.Hold):
                    runner.check_admission(root, git=fake_git)
                seed["allowedPaths"] = allowed
                record["allowedPaths"] = allowed
                seed["trancheId"] = "CCMAI-RUNTIME-106"
                record["trancheId"] = "CCMAI-RUNTIME-106"
                persist_packet()
                with self.assertRaises(runner.Hold):
                    runner.check_admission(root, git=fake_git)

    def test_predecessor_guard_rejects_missing_or_consumed_history(self):
        def check(mutator=None, must_hold=False):
            with tempfile.TemporaryDirectory() as temp:
                root = Path(temp)
                def place(relative, content):
                    path = root / relative
                    path.parent.mkdir(parents=True, exist_ok=True)
                    path.write_bytes(content)
                    return path
                parent_seed_bytes = b'{"trancheId":"CCMAI-RUNTIME-106"}\n'
                place(runner.PARENT_SEED, parent_seed_bytes)
                place(runner.PARENT_ORDER, b"Status: BLOCKED_WITH_REASON\n")
                record = {"status": "BUILD_BLOCKED", "x-actualExecution": {"fixtureStarts": 1, "backendLaunches": 1, "viteLaunches": 1, "runtimeRetries": 0}}
                place(runner.PARENT_RECORD, json.dumps(record).encode())
                place_history(root, place)
                if mutator:
                    mutator(root, record, place)
                def fake_git(_root, *args):
                    if args[0] == "log":
                        return ("c" * 40 + "\n").encode()
                    if args[0] == "show":
                        return parent_seed_bytes
                    return b""
                if must_hold:
                    with self.assertRaises(runner.Hold):
                        runner.verify_predecessor(root, git=fake_git)
                else:
                    runner.verify_predecessor(root, git=fake_git)

        check()
        check(lambda _root, record, place: (record["x-actualExecution"].pop("viteLaunches"), place(runner.PARENT_RECORD, json.dumps(record).encode())), True)
        check(lambda _root, record, place: (record["x-actualExecution"].update(backendLaunches=0), place(runner.PARENT_RECORD, json.dumps(record).encode())), True)
        check(lambda _root, record, place: (record.update(status="BUILD"), place(runner.PARENT_RECORD, json.dumps(record).encode())), True)
        check(lambda _root, _record, place: place(runner.PARENT_ORDER, b"Status: ACTIVE\n"), True)
        check(lambda root, _record, _place: (root / runner.PARENT_RUNTIME / "cleanup.json").unlink(), True)
        check(lambda _root, _record, place: place(runner.PARENT_SEED, b'{"trancheId":"edited"}\n'), True)

    def test_timeout_and_launch_failure_have_receipts(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            exe = root / "tool.exe"
            exe.write_bytes(b"synthetic executable")
            adapter = runner.ShellAdapter(root, {"dockerExe": str(exe), "nodeExe": str(exe), "dockerHost": "npipe:////./pipe/dockerDesktopLinuxEngine"}, root)
            self.assertEqual([str(exe), "--host", "npipe:////./pipe/dockerDesktopLinuxEngine", "--config", str(root / "docker-cli-config")], adapter.docker_prefix)
            timeout = subprocess.TimeoutExpired([str(exe), "inspect"], 3, output=b"partial", stderr=b"late")
            with patch.object(runner.subprocess, "run", side_effect=timeout):
                with self.assertRaises(runner.Hold):
                    adapter.command([str(exe), "inspect"], timeout=3)
            receipt = json.loads((root / "command_001.json").read_text())
            self.assertTrue((root / "docker-cli-config").is_dir())
            self.assertEqual("TimeoutExpired", receipt["failureType"])
            self.assertEqual("partial", receipt["stdout"])
            self.assertIsNone(receipt["exitCode"])
            with patch.object(runner.subprocess, "Popen", side_effect=OSError("injected")):
                with self.assertRaises(runner.Hold):
                    adapter.launch([str(exe), "--test"], {})
            self.assertTrue((root / "launch_002.intent.json").is_file())
            self.assertEqual("OSError", json.loads((root / "launch_002.result.json").read_text())["failureType"])
            with self.assertRaises(runner.Hold):
                runner.ShellAdapter(root, {"dockerExe": str(exe), "nodeExe": str(exe), "dockerHost": "tcp://remote:2375"}, root)

    def test_cooperative_stop_request_is_idempotent(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            output = root / runner.RUNTIME
            output.mkdir(parents=True)
            admission = {"seedSha256": "a" * 64, "pins": {runner.RUNNER: "b" * 64}}
            (output / "admission.json").write_text(json.dumps({"seedSha256": admission["seedSha256"], "runnerSha256": admission["pins"][runner.RUNNER]}))
            (output / "running.json").write_text("{}")
            runner.request_stop(root, admission)
            first = (output / "stop.request").read_bytes()
            runner.request_stop(root, admission)
            self.assertEqual(first, (output / "stop.request").read_bytes())

    def test_post_spawn_receipt_failure_stops_owned_handle(self):
        class Child:
            pid = 41357
            stopped = False
            def poll(self):
                return 0 if self.stopped else None
            def terminate(self):
                self.stopped = True
            def wait(self, timeout):
                return 0

        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            exe = root / "injected.exe"
            exe.write_bytes(b"synthetic executable")
            adapter = runner.ShellAdapter(root, {"dockerExe": str(exe), "nodeExe": str(exe), "dockerHost": "npipe:////./pipe/dockerDesktopLinuxEngine"}, root)
            child = Child()
            real_write = runner.write_once
            def fail_result(path, payload):
                if path.name.endswith(".result.json"):
                    raise OSError("injected receipt disk failure")
                return real_write(path, payload)
            with patch.object(runner.subprocess, "Popen", return_value=child), patch.object(runner, "write_once", side_effect=fail_result):
                with self.assertRaisesRegex(runner.Hold, "owned PID 41357; direct-handle stop completed"):
                    adapter.launch([str(exe)], {})
            self.assertTrue(child.stopped)
            self.assertTrue((root / "launch_001.intent.json").is_file())

    def test_reader_failure_and_multibyte_truncation(self):
        class Broken:
            def __iter__(self):
                yield "first\n"
                raise OSError("injected read failure")

        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            failed = runner.collect_lines(Broken(), root / "failed.log", queue.Queue(maxsize=1))
            failed.join(timeout=2)
            self.assertEqual("OSError", failed.error)
            self.assertEqual("OSError", json.loads((root / "failed.log.meta.json").read_text())["readerError"])
            long_line = "€" * 400000
            retained = runner.collect_lines(iter([long_line]), root / "utf8.log", queue.Queue(maxsize=1))
            retained.join(timeout=2)
            self.assertIsNone(retained.error)
            data = (root / "utf8.log").read_bytes()
            data.decode("utf-8", errors="strict")
            meta = json.loads((root / "utf8.log.meta.json").read_text())
            self.assertEqual(len(data), meta["retainedBytes"])
            self.assertEqual(runner.sha256(root / "utf8.log"), meta["retainedSha256"])
            self.assertTrue(meta["truncated"])

    def test_seed_git_lf_representation(self):
        lf = b'{"trancheId":"CCMAI-RUNTIME-106"}\n'
        crlf = lf.replace(b"\n", b"\r\n")
        self.assertEqual(runner.canonical_git_text(lf), runner.canonical_git_text(crlf))
        self.assertNotEqual(runner.canonical_git_text(lf), runner.canonical_git_text(crlf.replace(b"106", b"105")))
        with self.assertRaises(runner.Hold):
            runner.canonical_git_text(b"bad\rtext")

    def test_frontend_pins_and_path_escape(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            files = []
            for i in range(137):
                name = f"frontend/src/file{i}.txt"
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(str(i).encode())
                files.append({"path": name, "bytes": path.stat().st_size, "sha256": runner.sha256(path)})
            manifest = {"files": files}
            self.assertEqual(137, runner.validate_frontend(root, manifest))
            (root / files[0]["path"]).write_bytes(b"changed")
            with self.assertRaises(runner.Hold):
                runner.validate_frontend(root, manifest)
            with self.assertRaises(runner.Hold):
                runner.inside(root, "../outside")

    def test_historical_crlf_mapping_and_duplicate_rejection(self):
        rows = []
        for i in range(17):
            raw = f"row {i}\n".encode()
            represented = raw.replace(b"\n", b"\r\n") if i == 0 else raw
            rows.append({"path": f"src/{i}.txt", "sha256": hashlib.sha256(represented).hexdigest()})
        self.assertEqual(17, runner.validate_historical_rows(rows, lambda commit, path: f"row {Path(path).stem}\n".encode()))
        rows[1]["path"] = rows[0]["path"]
        with self.assertRaises(runner.Hold):
            runner.validate_historical_rows(rows, lambda commit, path: b"anything")

    def test_fixture_identity_port_and_fresh_ready(self):
        c = {
            "Id": runner.CONTAINER, "Name": "/" + runner.NAME, "Image": runner.IMAGE,
            "Config": {"Labels": runner.LABEL}, "State": {"Status": "exited"},
            "Mounts": [{"Type": "volume", "Name": runner.NAME, "Destination": "/var/lib/mysql", "RW": True}],
            "HostConfig": {"PortBindings": {"3306/tcp": [{"HostIp": "127.0.0.1", "HostPort": ""}]}},
            "NetworkSettings": {"Networks": {runner.NAME: {"NetworkID": runner.NETWORK}}, "Ports": {"3306/tcp": [{"HostIp": "127.0.0.1", "HostPort": "56039"}]}},
        }
        n = {"Id": runner.NETWORK, "Name": runner.NAME, "Containers": {}}
        runner.validate_container(c, n, [runner.CONTAINER])
        self.assertEqual(56039, runner.parse_db_port(c))
        c["NetworkSettings"]["Ports"]["3306/tcp"][0]["HostIp"] = "0.0.0.0"
        with self.assertRaises(runner.Hold):
            runner.parse_db_port(c)
        c["NetworkSettings"]["Ports"]["3306/tcp"][0]["HostIp"] = "127.0.0.1"
        uid = "11111111-1111-1111-1111-111111111111"
        lines = [f"R101_BROWSER_READY http://127.0.0.1:52962 tenant={uid} case={uid} reader={uid}", f"R101_BROWSER_OWNER {uid}@example.invalid"]
        self.assertEqual("http://127.0.0.1:52962", runner.parse_ready(lines)["apiOrigin"])
        with self.assertRaises(runner.Hold):
            runner.parse_ready(lines + lines[:1])

    def test_reservation_is_irreversible_and_runtime_holds(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            ledger = root / "ledger"
            ledger.mkdir()
            runner.reserve(ledger, "fixture_start")
            with self.assertRaises(runner.Hold):
                runner.reserve(ledger, "fixture_start")
            self.assertEqual("RESERVED", json.loads((ledger / "fixture_start.json").read_text())["state"])
            with self.assertRaises(runner.Hold):
                runner.check_admission(root)

    def test_config_literal_render_and_environment_filter(self):
        template = "const root = '__R105_FRONTEND_ROOT__'; const api = '__R105_API_ORIGIN__'; const nonce = '__R105_NONCE__';"
        with tempfile.TemporaryDirectory() as temp:
            result = runner.render_config(template, Path(temp), "http://127.0.0.1:52962", "a" * 32)
            self.assertIn('"http://127.0.0.1:52962"', result)
            self.assertNotIn("__R105_", result)
            with self.assertRaises(runner.Hold):
                runner.render_config(template, Path(temp), "http://localhost:8080", "a" * 32)
        with patch.dict(runner.os.environ, {"VITE_API_TARGET": "http://evil", "OPENAI_API_KEY": "secret"}, clear=True):
            self.assertEqual({}, runner.minimal_env())

    def test_supervised_deadline_and_exception_cleanup(self):
        uid = "11111111-1111-1111-1111-111111111111"

        class Process:
            def __init__(self, lines, pid):
                self.stdout = io.StringIO(lines)
                self.stderr = io.StringIO("")
                self.pid = pid
                self.terminated = False

            def poll(self):
                return 0 if self.terminated else None

            def terminate(self):
                self.terminated = True

            def wait(self, timeout):
                return 0

        class Adapter:
            def __init__(self, fail_vite=False):
                self.fail_vite = fail_vite
                self.processes = []
                self.fixture_stopped = False

            def protected(self):
                return [{"id": "protected"}]

            def fixture(self):
                return None, None, None

            def start_fixture(self):
                return None

            def db_ready(self, deadline, clock, pause):
                return 56039

            def launch(self, args, env):
                self.processes.append(Process(
                    f"R101_BROWSER_READY http://127.0.0.1:52962 tenant={uid} case={uid} reader={uid}\nR101_BROWSER_OWNER {uid}@example.invalid\n" if not self.processes else "Vite ready\n",
                    len(self.processes) + 100,
                ))
                return self.processes[-1]

            def vite_ready(self, process, nonce, deadline, clock, pause):
                if self.fail_vite:
                    raise runner.Hold("injected readiness failure")

            def stop_fixture(self):
                self.fixture_stopped = True

        class Clock:
            value = 0.0
            def now(self):
                return self.value
            def pause(self, amount):
                self.value += 2101

        for fail in (False, True):
            with self.subTest(fail=fail), tempfile.TemporaryDirectory() as temp:
                root = Path(temp)
                template = root / runner.TEMPLATE
                template.parent.mkdir(parents=True)
                template.write_text("const root = '__R105_FRONTEND_ROOT__'; const api = '__R105_API_ORIGIN__'; const nonce = '__R105_NONCE__';", encoding="utf-8")
                vite = root / "frontend/node_modules/vite/bin/vite.js"
                vite.parent.mkdir(parents=True)
                vite.write_text("// cached", encoding="utf-8")
                node = root / "node.exe"
                node.write_bytes(b"injected node fixture")
                adapter, clock = Adapter(fail), Clock()
                admission = {"seedSha256": "a" * 64, "baseCommit": "b" * 40, "pins": {runner.RUNNER: "c" * 64}, "nodeExe": str(node)}
                with patch.object(runner, "validate_container"), patch.object(runner, "port_free", return_value=True), patch.object(runner.Path, "cwd", return_value=root):
                    if fail:
                        with self.assertRaises(runner.Hold):
                            runner.run_campaign(root, admission, adapter, clock=clock.now, pause=clock.pause)
                    else:
                        runner.run_campaign(root, admission, adapter, clock=clock.now, pause=clock.pause)
                self.assertTrue(adapter.fixture_stopped)
                self.assertTrue(all(p.terminated for p in adapter.processes))
                self.assertTrue((root / runner.RUNTIME / "cleanup.json").is_file())
                with self.assertRaises(runner.Hold):
                    runner.run_campaign(root, admission, adapter, clock=clock.now, pause=clock.pause)


if __name__ == "__main__":
    unittest.main()
