"""Offline R117 supervisor control checks; no Docker or provider process starts."""
import importlib.util
import json
from pathlib import Path
from tempfile import TemporaryDirectory
from types import SimpleNamespace
import unittest
from unittest.mock import patch


SPEC = importlib.util.spec_from_file_location(
    "r117_waitdata_fixture", Path(__file__).resolve().parents[1] / "r117_waitdata_fixture.py"
)
fixture = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(fixture)


class R117FixtureTests(unittest.TestCase):
    def setUp(self):
        self.ident = "a" * 64
        self.network_id = "b" * 64
        self.pin = {
            "containerId": self.ident, "networkId": self.network_id,
            "ownerNonce": "owned-nonce", "protectedBaseline": [], "protectedResources": {},
        }

    def ports(self, port="49153"):
        return {"3306/tcp": [{"HostIp": "127.0.0.1", "HostPort": port}]}

    def value(self, state="running", ports=None):
        return {
            "Id": self.ident, "Image": fixture.IMAGE, "Name": "/" + fixture.NAME,
            "State": state,
            "Labels": {"ccmai.tranche": "CCMAI-RUNTIME-117", "ccmai.owner": "owned-nonce"},
            "Mounts": [{"Type": "volume", "Name": fixture.NAME, "Destination": "/var/lib/mysql"}],
            "Networks": {fixture.NAME: {"NetworkID": self.network_id}}, "Ports": ports,
        }

    def docker_response(self, state="running", ports=None, mutate=None):
        value = self.value(state, ports)
        if mutate:
            mutate(value)
        labels = {"ccmai.tranche": "CCMAI-RUNTIME-117", "ccmai.owner": "owned-nonce"}
        volume = {"Name": fixture.NAME, "Driver": "local", "Labels": labels, "Options": {}}
        network = {
            "Id": self.network_id, "Name": fixture.NAME, "Driver": "bridge",
            "Internal": False, "Labels": labels,
            "Options": {"com.docker.network.bridge.gateway_mode_ipv4": "nat",
                        "com.docker.network.bridge.host_binding_ipv4": "127.0.0.1"},
            "Containers": {self.ident: {}},
        }
        calls = []

        def fake_docker(*args, **kwargs):
            calls.append(args)
            if args[0] == "inspect":
                return SimpleNamespace(stdout=json.dumps(value).encode(), returncode=0)
            if args[:2] == ("volume", "inspect"):
                return SimpleNamespace(stdout=json.dumps(volume).encode(), returncode=0)
            if args[:2] == ("network", "inspect"):
                return SimpleNamespace(stdout=json.dumps(network).encode(), returncode=0)
            if args[:2] == ("ps", "-aq"):
                return SimpleNamespace(stdout=(self.ident + "\n").encode(), returncode=0)
            if args[0] == "stop":
                value["State"] = "exited"
                return SimpleNamespace(stdout=self.ident.encode(), returncode=0)
            raise AssertionError("unexpected Docker command: " + repr(args))

        return fake_docker, calls

    def test_full_published_port_shape(self):
        self.assertEqual(fixture.published_port(self.ports()), 49153)
        invalid = [None, {}, {"3306/tcp": None}, {"3306/tcp": []},
                   {"3306/tcp": [None]}, {"3306/tcp": [self.ports()["3306/tcp"][0]] * 2},
                   {"3306/tcp": [{"HostIp": "0.0.0.0", "HostPort": "49153"}]},
                   {"3306/tcp": [{"HostIp": "127.0.0.1", "HostPort": ""}]},
                   {"3306/tcp": [{"HostIp": "127.0.0.1", "HostPort": "3306"}], "80/tcp": None}]
        for candidate in invalid:
            with self.subTest(candidate=candidate), self.assertRaisesRegex(RuntimeError, "FIXTURE_PORT_INVALID"):
                fixture.published_port(candidate)

    def test_absent_port_still_verifies_exact_ownership(self):
        fake, _ = self.docker_response(ports=None)
        with patch.object(fixture, "docker", side_effect=fake):
            self.assertEqual(fixture.inspect_fixture(self.pin, check_protected=False, require_port=False)["State"], "running")
            with self.assertRaises(RuntimeError):
                fixture.inspect_fixture(self.pin, check_protected=False, require_port=True)

    def test_wrong_identity_mount_network_or_exclusivity_rejects(self):
        mutations = [
            lambda v: v.update(Id="c" * 64),
            lambda v: v["Labels"].update({"ccmai.owner": "stranger"}),
            lambda v: v["Mounts"][0].update(Name="other-volume"),
            lambda v: v["Networks"][fixture.NAME].update(NetworkID="c" * 64),
        ]
        for mutate in mutations:
            fake, _ = self.docker_response(ports=None, mutate=mutate)
            with self.subTest(mutate=mutate), patch.object(fixture, "docker", side_effect=fake):
                with self.assertRaises(RuntimeError):
                    fixture.inspect_fixture(self.pin, check_protected=False, require_port=False)

        fake, _ = self.docker_response(ports=None)
        def nonexclusive(*args, **kwargs):
            result = fake(*args, **kwargs)
            if args[:2] == ("ps", "-aq"):
                result.stdout += b"c\n"
            return result
        with patch.object(fixture, "docker", side_effect=nonexclusive), self.assertRaisesRegex(RuntimeError, "FIXTURE_NOT_EXCLUSIVE"):
            fixture.inspect_fixture(self.pin, check_protected=False, require_port=False)

    def test_cleanup_uses_container_receipt_without_port(self):
        fake, calls = self.docker_response(ports=None)
        with TemporaryDirectory() as temp:
            run, evidence = Path(temp) / "run", Path(temp) / "evidence"
            run.mkdir()
            evidence.mkdir()
            (run / "container-created.json").write_text(json.dumps(self.pin), encoding="utf-8")
            with patch.object(fixture, "RUN", run), patch.object(fixture, "EVIDENCE", evidence), \
                 patch.object(fixture, "docker", side_effect=fake), \
                 patch.object(fixture, "inventory", return_value=[]), \
                 patch.object(fixture, "resource_inventory", return_value={}):
                fixture.cleanup()
            receipt = json.loads((evidence / "cleanup_receipt.json").read_text(encoding="utf-8"))
            self.assertEqual(receipt["before"]["Ports"], None)
            self.assertEqual(receipt["after"]["State"], "exited")
            self.assertTrue(receipt["protectedBaselineEquals"])
            self.assertEqual(sum(args[0] == "stop" for args in calls), 1)
            self.assertFalse(any(args[0] == "rm" for args in calls))

    def test_cleanup_rejects_wrong_owner_before_stop(self):
        fake, calls = self.docker_response(ports=None, mutate=lambda v: v["Labels"].update({"ccmai.owner": "stranger"}))
        with TemporaryDirectory() as temp:
            run, evidence = Path(temp) / "run", Path(temp) / "evidence"
            run.mkdir()
            evidence.mkdir()
            (run / "container-created.json").write_text(json.dumps(self.pin), encoding="utf-8")
            with patch.object(fixture, "RUN", run), patch.object(fixture, "EVIDENCE", evidence), \
                 patch.object(fixture, "docker", side_effect=fake):
                with self.assertRaisesRegex(RuntimeError, "FIXTURE_IDENTITY_CHANGED"):
                    fixture.cleanup()
            self.assertFalse((run / "cleanup.effect").exists())
            self.assertFalse(any(args[0] == "stop" for args in calls))


if __name__ == "__main__":
    unittest.main()
