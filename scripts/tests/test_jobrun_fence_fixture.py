"""Portable safety tests for the R083 disposable fixture harness."""

from __future__ import annotations

import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parent))
import jobrun_fence_fixture as fixture  # noqa: E402


TASK_ID = "0123456789abcdef0123456789abcdef"
CONTAINER_ID = "a" * 64
NETWORK_ID = "b" * 64


class FakeResourceOps(fixture.FixtureResourceOps):
    def __init__(self, container=None, network=None):
        self.container = container
        self.network = network
        self.removed = []

    def inspect_container(self, identifier):
        return self.container

    def remove_container(self, identifier):
        self.removed.append(("container", identifier))
        self.container = None

    def inspect_network(self, identifier):
        return self.network

    def remove_network(self, identifier):
        self.removed.append(("network", identifier))
        self.network = None


def valid_container(role="worker", task_id=TASK_ID):
    names = fixture.names_for(role, task_id)
    return {
        "id": CONTAINER_ID,
        "name": "/" + names["container"],
        "imageId": "sha256:" + "c" * 64,
        "imageReference": "mysql:8.0",
        "labels": fixture.expected_labels(role, task_id),
        "networkMode": names["network"],
        "networks": {names["network"]: {"NetworkID": ""}},
        "ports": None,
        "binds": None,
        "tmpfs": {"/var/lib/mysql": "rw,noexec,nosuid,size=512m"},
        "mounts": [{"Type": "tmpfs", "Destination": "/var/lib/mysql", "RW": True}],
    }


class FixtureSafetyTests(unittest.TestCase):
    def test_task_names_and_labels_are_role_scoped_and_grant_safe(self):
        worker = fixture.names_for("worker", TASK_ID)
        reviewer = fixture.names_for("reviewer", TASK_ID)
        self.assertNotEqual(worker["network"], reviewer["network"])
        self.assertNotEqual(worker["schema"], reviewer["schema"])
        for key in ("schema", "principal", "denySchema", "probeTable"):
            self.assertRegex(worker[key], r"^[a-z0-9]+$")
        self.assertTrue(worker["network"].startswith("ccmai-r083-"))
        self.assertTrue(worker["container"].startswith("ccmai-r083-"))
        self.assertEqual(len(worker["principal"]), 32)
        self.assertEqual(fixture.expected_labels("worker", TASK_ID)["ccmai.cvf.role"], "worker")
        with self.assertRaises(fixture.FixtureError):
            fixture.names_for("other", TASK_ID)

    def test_remote_docker_endpoints_fail_closed(self):
        self.assertEqual(fixture.validate_local_endpoint("unix:///var/run/docker.sock"), "unix-local")
        self.assertEqual(fixture.validate_local_endpoint("npipe:////./pipe/docker_engine"), "npipe-local")
        for endpoint in (
            "tcp://127.0.0.1:2375",
            "ssh://docker.example",
            "https://docker.example",
            "npipe:////server/pipe/docker_engine",
            "npipe://server/pipe/docker_engine",
            "unix://docker.example/var/run/docker.sock",
            "unix://relative/docker.sock",
            "unix:///var/run/docker.sock?remote=1",
        ):
            with self.subTest(endpoint=endpoint), self.assertRaises(fixture.FixtureError):
                fixture.validate_local_endpoint(endpoint)

    def test_cached_image_metadata_requires_a_full_canonical_digest(self):
        digest = "d" * 64
        parsed = fixture.parse_image_metadata(
            f"sha256:{'e' * 64}|[\"docker.io/library/mysql@sha256:{digest}\"]|linux|amd64"
        )
        self.assertEqual(parsed["repositoryDigest"], f"mysql@sha256:{digest}")
        for value in (
            "short|[]|linux|amd64",
            f"sha256:{'e' * 64}|[]|linux|amd64",
            f"sha256:{'e' * 64}|[\"mysql:8.0\"]|linux|amd64",
        ):
            with self.subTest(value=value), self.assertRaises(fixture.FixtureError):
                fixture.parse_image_metadata(value)

    def test_grants_accept_only_global_usage_and_literal_task_schema(self):
        names = fixture.names_for("worker", TASK_ID)
        principal = names["principal"]
        schema = names["schema"]
        accepted = [
            f"GRANT USAGE ON *.* TO '{principal}'@'127.0.0.1'",
            f"GRANT CREATE, DROP, INSERT, SELECT, UPDATE, DELETE ON `{schema}`.* TO '{principal}'@'127.0.0.1'",
        ]
        parsed = fixture.parse_grant_scope_lines(accepted, schema=schema, principal=principal)
        self.assertTrue(parsed["noGlobalOrUnownedSchemaPrivileges"])
        self.assertEqual(len(parsed["grants"]), 2)
        unsafe = (
            accepted + [f"GRANT CREATE ON *.* TO '{principal}'@'127.0.0.1'"],
            [accepted[0], accepted[1] + " WITH GRANT OPTION"],
            [accepted[0], f"GRANT CREATE, DROP, INSERT, SELECT, UPDATE, DELETE ON `{schema}%`.* TO '{principal}'@'127.0.0.1'"],
            [accepted[0], f"GRANT CREATE, DROP, INSERT, SELECT, UPDATE, DELETE ON `{schema}`.* TO '{principal}'@'%'"],
        )
        for lines in unsafe:
            with self.subTest(lines=lines), self.assertRaises(fixture.FixtureError):
                fixture.parse_grant_scope_lines(lines, schema=schema, principal=principal)

    def test_expected_denial_accepts_only_exact_task_database_privilege_error(self):
        names = fixture.names_for("worker", TASK_ID)
        expected = f"ERROR 1044 (42000): Access denied for user '{names['principal']}'@'127.0.0.1' to database '{names['denySchema']}'"
        self.assertTrue(fixture.expected_privilege_denial(expected, principal=names["principal"], schema=names["denySchema"]))
        self.assertTrue(fixture.expected_privilege_denial(expected + " at line 1", principal=names["principal"], schema=names["denySchema"]))
        self.assertTrue(
            fixture.expected_privilege_denial(
                expected.replace("ERROR 1044 (42000):", "ERROR 1044 (42000) at line 1:"),
                principal=names["principal"],
                schema=names["denySchema"],
            )
        )
        self.assertFalse(fixture.expected_privilege_denial("ERROR 1045 (28000): access denied", principal=names["principal"], schema=names["denySchema"]))
        self.assertFalse(fixture.expected_privilege_denial(expected.replace(names["denySchema"], "other"), principal=names["principal"], schema=names["denySchema"]))

    def test_container_inspect_accepts_only_authorized_tmpfs_network_and_no_host_exposure(self):
        names = fixture.names_for("worker", TASK_ID)
        labels = fixture.expected_labels("worker", TASK_ID)
        observed = valid_container()
        self.assertEqual(
            fixture.validate_container_inspect(
                observed,
                name=names["container"],
                image_id=observed["imageId"],
                network_name=names["network"],
                labels=labels,
                expected_id=CONTAINER_ID,
                expected_network_id=NETWORK_ID,
            ),
            CONTAINER_ID,
        )
        mutations = (
            ("ports", {"3306/tcp": [{"HostPort": "3306"}]}),
            ("binds", ["C:\\host:/var/lib/mysql"]),
            ("networkMode", "bridge"),
            ("tmpfs", {"/var/lib/mysql": "rw,size=512m"}),
            ("mounts", [{"Type": "bind", "Destination": "/var/lib/mysql", "RW": True}]),
        )
        for key, value in mutations:
            with self.subTest(key=key):
                unsafe = dict(observed)
                unsafe[key] = value
                with self.assertRaises(fixture.FixtureError):
                    fixture.validate_container_inspect(
                        unsafe,
                        name=names["container"],
                        image_id=observed["imageId"],
                        network_name=names["network"],
                        labels=labels,
                        expected_id=CONTAINER_ID,
                        expected_network_id=NETWORK_ID,
                    )
        inherited = valid_container()
        inherited["labels"] = {**labels, "image.vendor": "mysql"}
        self.assertEqual(
            fixture.validate_container_inspect(
                inherited,
                name=names["container"],
                image_id=inherited["imageId"],
                network_name=names["network"],
                labels=labels,
                expected_id=CONTAINER_ID,
                expected_network_id=NETWORK_ID,
            ),
            CONTAINER_ID,
        )
        for key, value in (("imageId", "sha256:" + "d" * 64), ("id", "e" * 64), ("labels", {**labels, "ccmai.cvf.role": "reviewer"})):
            unsafe = valid_container()
            unsafe[key] = value
            with self.subTest(identity_key=key), self.assertRaises(fixture.FixtureError):
                fixture.validate_container_inspect(
                    unsafe,
                    name=names["container"],
                    image_id="sha256:" + "c" * 64,
                    network_name=names["network"],
                    labels=labels,
                    expected_id=CONTAINER_ID,
                    expected_network_id=NETWORK_ID,
                )
        started = valid_container()
        started["networks"] = {names["network"]: {"NetworkID": NETWORK_ID}}
        self.assertEqual(
            fixture.validate_container_inspect(
                started,
                name=names["container"],
                image_id=started["imageId"],
                network_name=names["network"],
                labels=labels,
                expected_id=CONTAINER_ID,
                expected_network_id=NETWORK_ID,
                require_network_id=True,
            ),
            CONTAINER_ID,
        )
        with self.assertRaises(fixture.FixtureError):
            fixture.validate_container_inspect(
                started,
                name=names["container"],
                image_id=started["imageId"],
                network_name=names["network"],
                labels=labels,
                expected_id=CONTAINER_ID,
                expected_network_id="c" * 64,
                require_network_id=True,
            )

    def test_network_inspect_requires_internal_and_exact_owner_labels(self):
        names = fixture.names_for("reviewer", TASK_ID)
        labels = fixture.expected_labels("reviewer", TASK_ID)
        observed = {
            "id": NETWORK_ID,
            "name": names["network"],
            "internal": True,
            "labels": labels,
            "containers": {},
        }
        self.assertEqual(
            fixture.validate_network_inspect(observed, name=names["network"], labels=labels, expected_id=NETWORK_ID),
            NETWORK_ID,
        )
        inherited = dict(observed)
        inherited["labels"] = {**labels, "image.vendor": "mysql"}
        self.assertEqual(fixture.validate_network_inspect(inherited, name=names["network"], labels=labels), NETWORK_ID)
        attached = dict(observed)
        attached["containers"] = {CONTAINER_ID: {"Name": names["container"]}}
        self.assertEqual(
            fixture.validate_network_inspect(
                attached,
                name=names["network"],
                labels=labels,
                expected_id=NETWORK_ID,
                expected_container_id=CONTAINER_ID,
                expected_container_name=names["container"],
            ),
            NETWORK_ID,
        )
        with self.assertRaises(fixture.FixtureError):
            fixture.validate_network_inspect(attached, name=names["network"], labels=labels, expected_id="c" * 64)
        for key, value in (("internal", False), ("labels", {}), ("name", "other")):
            with self.subTest(key=key):
                unsafe = dict(observed)
                unsafe[key] = value
                with self.assertRaises(fixture.FixtureError):
                    fixture.validate_network_inspect(unsafe, name=names["network"], labels=labels)

    def test_uncertain_create_is_reconciled_only_to_exact_labelled_resource(self):
        labels = fixture.expected_labels("worker", TASK_ID)
        matching = {"id": NETWORK_ID, "labels": labels}
        result = fixture.CommandResult(1, "", "response lost", 100)
        resource_id, disposition, observed_id = fixture.reconcile_unknown_creation(
            result=result,
            intended_name="random-task-name",
            expected_labels_value=labels,
            inspect_exact_name=lambda name: matching if name == "random-task-name" else None,
        )
        self.assertEqual(resource_id, NETWORK_ID)
        self.assertEqual(disposition, "RECONCILED_OWN_CREATE_RESPONSE_UNKNOWN")
        self.assertEqual(observed_id, NETWORK_ID)
        resource_id, disposition, observed_id = fixture.reconcile_unknown_creation(
            result=result,
            intended_name="random-task-name",
            expected_labels_value=labels,
            inspect_exact_name=lambda _name: {"id": NETWORK_ID, "labels": {}},
        )
        self.assertIsNone(resource_id)
        self.assertEqual(disposition, "UNKNOWN_OWNERSHIP_OR_CREATE_OUTCOME")
        self.assertEqual(observed_id, NETWORK_ID)
        resource_id, disposition, observed_id = fixture.reconcile_unknown_creation(
            result=fixture.CommandResult(None, "", "", 100, True, "DOCKER_COMMAND_TIMEOUT"),
            intended_name="random-task-name",
            expected_labels_value=labels,
            inspect_exact_name=lambda _name: None,
        )
        self.assertIsNone(resource_id)
        self.assertEqual(disposition, "UNKNOWN_OWNERSHIP_OR_CREATE_OUTCOME")
        self.assertIsNone(observed_id)
        for resource, name in (
            ("network", fixture.names_for("worker", TASK_ID)["network"]),
            ("container", fixture.names_for("worker", TASK_ID)["container"]),
        ):
            with self.subTest(resource=resource):
                _resource_id, uncertain, _observed_id = fixture.reconcile_unknown_creation(
                    result=fixture.CommandResult(None, "", "", 100, True, "DOCKER_COMMAND_TIMEOUT"),
                    intended_name=name,
                    expected_labels_value=labels,
                    inspect_exact_name=lambda _name: None,
                )
                self.assertEqual(uncertain, "UNKNOWN_OWNERSHIP_OR_CREATE_OUTCOME")

    def test_not_found_requires_exact_daemon_resource_response(self):
        network_name = fixture.names_for("worker", TASK_ID)["network"]
        self.assertTrue(
            fixture._known_not_found(
                f"Error response from daemon: network {network_name} not found",
                resource="network",
                identifier=network_name,
            )
        )
        for message in (
            f"invalid context: network {network_name} not found",
            f"Error response from daemon: container {network_name} not found",
            "not found",
        ):
            with self.subTest(message=message):
                self.assertFalse(fixture._known_not_found(message, resource="network", identifier=network_name))

    def test_cleanup_removes_only_exact_owned_ids_and_verifies_absence(self):
        names = fixture.names_for("worker", TASK_ID)
        labels = fixture.expected_labels("worker", TASK_ID)
        container = valid_container()
        network = {
            "id": NETWORK_ID,
            "name": names["network"],
            "labels": labels,
            "containers": {},
        }
        ops = FakeResourceOps(container=container, network=network)
        result = fixture.cleanup_owned_resources(
            ops,
            container_id=CONTAINER_ID,
            network_id=NETWORK_ID,
            container_expected={"name": names["container"], "labels": labels},
            network_expected={"name": names["network"], "labels": labels},
        )
        self.assertEqual(result["status"], "VERIFIED_CLEAN")
        self.assertEqual(ops.removed, [("container", CONTAINER_ID), ("network", NETWORK_ID)])

        unowned = valid_container()
        unowned["labels"] = {}
        ops = FakeResourceOps(container=unowned)
        result = fixture.cleanup_owned_resources(
            ops,
            container_id=CONTAINER_ID,
            network_id=None,
            container_expected={"name": names["container"], "labels": labels},
            network_expected={"name": names["network"], "labels": labels},
        )
        self.assertEqual(result["status"], "FAILED_CLEANUP")
        self.assertEqual(ops.removed, [])

    def test_cleanup_continues_to_known_network_after_uncertain_container_inspection(self):
        names = fixture.names_for("worker", TASK_ID)
        labels = fixture.expected_labels("worker", TASK_ID)

        class UncertainContainerOps(FakeResourceOps):
            def inspect_container(self, identifier):
                raise RuntimeError("uncertain inspect")

        network = {"id": NETWORK_ID, "name": names["network"], "labels": labels, "containers": {}}
        ops = UncertainContainerOps(network=network)
        result = fixture.cleanup_owned_resources(
            ops,
            container_id=CONTAINER_ID,
            network_id=NETWORK_ID,
            container_expected={"name": names["container"], "labels": labels},
            network_expected={"name": names["network"], "labels": labels},
        )
        self.assertEqual(result["status"], "FAILED_CLEANUP")
        self.assertEqual(ops.removed, [("network", NETWORK_ID)])
        self.assertFalse(result.get("absenceVerified", False))

    def test_cleanup_refuses_to_claim_absence_when_container_remains(self):
        names = fixture.names_for("worker", TASK_ID)
        labels = fixture.expected_labels("worker", TASK_ID)

        class StillPresentOps(FakeResourceOps):
            def remove_container(self, identifier):
                self.removed.append(("container", identifier))

        ops = StillPresentOps(container=valid_container())
        result = fixture.cleanup_owned_resources(
            ops,
            container_id=CONTAINER_ID,
            network_id=None,
            container_expected={"name": names["container"], "labels": labels, "imageId": "sha256:" + "c" * 64},
            network_expected={"name": names["network"], "labels": labels},
        )
        self.assertEqual(result["status"], "FAILED_CLEANUP")
        self.assertEqual(result["failureCode"], "CLEANUP_CONTAINER_REMAINS")

    def test_uncertain_campaign_keeps_failure_even_after_cleaning_known_network(self):
        names = fixture.names_for("worker", TASK_ID)
        labels = fixture.expected_labels("worker", TASK_ID)
        network = {"id": NETWORK_ID, "name": names["network"], "labels": labels, "containers": {}}
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            task_dir = root / "task"
            task_dir.mkdir()
            campaign = fixture.Campaign(root, "worker", TASK_ID, task_dir)
            ops = FakeResourceOps(network=network)
            campaign.ops = ops
            campaign.network_id = NETWORK_ID
            campaign.uncertain_create_outcome = True
            campaign.unresolved_create_resources = [{"resource": "container", "name": names["container"], "id": None}]
            campaign._cleanup()
        self.assertEqual(campaign.receipt["cleanup"]["status"], "FAILED_CLEANUP")
        self.assertFalse(campaign.receipt["cleanup"]["absenceVerified"])
        self.assertEqual(ops.removed, [("network", NETWORK_ID)])
        self.assertEqual(campaign.receipt["cleanup"]["unresolvedResources"][0]["name"], names["container"])

    def test_unknown_network_only_attempt_is_not_reported_as_not_required(self):
        names = fixture.names_for("worker", TASK_ID)
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            task_dir = root / "task"
            task_dir.mkdir()
            campaign = fixture.Campaign(root, "worker", TASK_ID, task_dir)
            campaign.unresolved_create_resources = [{"resource": "network", "name": names["network"], "id": None}]
            campaign.uncertain_create_outcome = True
            campaign._cleanup()
        self.assertEqual(campaign.receipt["cleanup"]["status"], "FAILED_CLEANUP")
        self.assertFalse(campaign.receipt["cleanup"]["absenceVerified"])
        self.assertEqual(campaign.receipt["cleanup"]["failureCode"], "RESOURCE_CREATE_OUTCOME_UNCERTAIN")

    def test_task_output_is_confined_and_never_reuses_a_task_directory(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            task_dir = fixture.create_task_output_directory(root, "worker", TASK_ID)
            self.assertEqual(task_dir.relative_to(root).as_posix(), f"{fixture.OUTPUT_ROOT}/worker/{TASK_ID}")
            with self.assertRaises(fixture.FixtureError):
                fixture.create_task_output_directory(root, "worker", TASK_ID)

    def test_filtered_git_blob_hash_accepts_crlf_worktree_for_lf_blob(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            subprocess.run(["git", "init", "--quiet"], cwd=root, check=True)
            (root / ".gitattributes").write_text("fixture.txt text eol=lf\n", encoding="ascii")
            physical = b"first line\r\nsecond line\r\n"
            (root / "fixture.txt").write_bytes(physical)
            filtered = fixture._filtered_worktree_blob_sha1(root, "fixture.txt", physical)
            self.assertEqual(filtered, fixture._blob_sha1(b"first line\nsecond line\n"))
            self.assertNotEqual(fixture._blob_sha1(physical), filtered)

    def test_command_receipt_omits_argv_and_redacts_secret_output(self):
        secret = "fixture-only-password"
        completed = __import__("subprocess").CompletedProcess(
            args=[], returncode=0, stdout=f"password={secret}", stderr=""
        )
        with mock.patch.object(fixture.subprocess, "run", return_value=completed) as run:
            docker = fixture.DockerCli(Path.cwd(), env={"PATH": os.environ.get("PATH", "")})
            docker.call(
                "test command",
                ["exec", "container-id", "--env", "MYSQL_PWD"],
                input_text="SELECT 1;",
                redactions=(secret,),
                extra_env={"MYSQL_PWD": secret},
                record_output=True,
            )
        self.assertNotIn(secret, json.dumps(docker.commands))
        self.assertIn("[REDACTED]", docker.commands[0]["safeStdout"])
        self.assertNotIn(secret, json.dumps(run.call_args.args[0]))
        self.assertEqual(run.call_args.kwargs["env"]["MYSQL_PWD"], secret)

    def test_docker_timeout_is_returned_for_exact_create_reconciliation(self):
        with mock.patch.object(fixture.subprocess, "run", side_effect=subprocess.TimeoutExpired(["docker", "network", "create"], 1)):
            docker = fixture.DockerCli(Path.cwd(), env={"PATH": os.environ.get("PATH", "")})
            result = docker.call("create owned internal network", ["network", "create"], check=False, timeout=1)
        self.assertTrue(result.timed_out)
        self.assertEqual(result.failure_code, "DOCKER_COMMAND_TIMEOUT")
        self.assertIsNone(result.exit_code)
        self.assertEqual(docker.commands[0]["outputStatus"], "OMITTED_TIMEOUT_PARTIAL")


class MemoryDockerSubprocess:
    """In-memory daemon boundary for Campaign create/reconcile/cleanup tests."""

    def __init__(
        self,
        *,
        timeout_network_create=False,
        network_create_effect=True,
        timeout_container_create=False,
        container_create_effect=True,
    ):
        self.timeout_network_create = timeout_network_create
        self.network_create_effect = network_create_effect
        self.timeout_container_create = timeout_container_create
        self.container_create_effect = container_create_effect
        self.networks = {}
        self.containers = {}
        self.calls = []

    @staticmethod
    def _labels(args):
        labels = {}
        for index, item in enumerate(args[:-1]):
            if item == "--label":
                key, value = args[index + 1].split("=", 1)
                labels[key] = value
        return labels

    @staticmethod
    def _completed(command, returncode=0, stdout="", stderr=""):
        return subprocess.CompletedProcess(command, returncode, stdout, stderr)

    def _network(self, identifier):
        return next(
            (item for item in self.networks.values() if identifier in (item["id"], item["name"])),
            None,
        )

    def _container(self, identifier):
        return next(
            (item for item in self.containers.values() if identifier in (item["id"], item["name"], item["name"].lstrip("/"))),
            None,
        )

    def __call__(self, command, **kwargs):
        self.calls.append(list(command))
        args = command[1:]
        if args[:2] == ["network", "create"]:
            name = args[-1]
            labels = self._labels(args)
            if self.network_create_effect:
                self.networks[name] = {
                    "id": NETWORK_ID,
                    "name": name,
                    "internal": True,
                    "labels": labels,
                    "containers": {},
                }
            if self.timeout_network_create:
                raise subprocess.TimeoutExpired(command, kwargs.get("timeout", 1))
            return self._completed(command, stdout=f"{NETWORK_ID}\n")
        if args[:2] == ["container", "create"] or args[:1] == ["create"]:
            name = args[args.index("--name") + 1]
            network_name = args[args.index("--network") + 1]
            labels = self._labels(args)
            image_id = args[-1]
            if self.container_create_effect:
                role = labels["ccmai.cvf.role"]
                task_id = labels["ccmai.cvf.task-id"]
                container = valid_container(role, task_id)
                container.update(
                    {
                        "id": CONTAINER_ID,
                        "name": f"/{name}",
                        "imageId": image_id,
                        "networkMode": network_name,
                        "networks": {network_name: {"NetworkID": NETWORK_ID}},
                        "labels": labels,
                    }
                )
                self.containers[name] = container
                network = self._network(network_name)
                if network is not None:
                    network["containers"][CONTAINER_ID] = {"Name": name}
            if self.timeout_container_create:
                raise subprocess.TimeoutExpired(command, kwargs.get("timeout", 1))
            return self._completed(command, stdout=f"{CONTAINER_ID}\n")
        if args[:2] == ["network", "inspect"]:
            identifier = args[-1]
            observed = self._network(identifier)
            if observed is None:
                return self._completed(command, 1, stderr=f"Error response from daemon: network {identifier} not found")
            return self._completed(command, stdout=json.dumps(observed))
        if args[:2] == ["container", "inspect"]:
            identifier = args[-1]
            observed = self._container(identifier)
            if observed is None:
                return self._completed(command, 1, stderr=f"Error response from daemon: container {identifier} not found")
            return self._completed(command, stdout=json.dumps(observed))
        if args[:2] == ["container", "rm"]:
            identifier = args[-1]
            observed = self._container(identifier)
            if observed is not None:
                self.containers.pop(observed["name"].lstrip("/"), None)
                network = self._network(observed["networkMode"])
                if network is not None:
                    network["containers"].pop(observed["id"], None)
            return self._completed(command)
        if args[:2] == ["network", "rm"]:
            observed = self._network(args[-1])
            if observed is not None:
                self.networks.pop(observed["name"], None)
            return self._completed(command)
        raise AssertionError(f"unexpected simulated Docker command: {args!r}")


class FixtureCompletionTests(unittest.TestCase):
    def _campaign(self, root):
        task_dir = root / "task"
        task_dir.mkdir()
        campaign = fixture.Campaign(root, "worker", TASK_ID, task_dir)
        image = {"imageId": "sha256:" + "c" * 64, "repoDigest": "mysql@sha256:" + "d" * 64}

        def prepare():
            campaign.docker = fixture.DockerCli(root, env={})
            campaign.receipt["plan"]["image"] = image
            return image, "test-engine"

        campaign.prepare = prepare
        return campaign

    def test_started_ownership_validation_precedes_every_mysql_call(self):
        names = fixture.names_for("worker", TASK_ID)
        labels = fixture.expected_labels("worker", TASK_ID)

        class OrderedDocker:
            def __init__(self, events):
                self.events = events
                self.commands = []

            def call(self, label, args, **kwargs):
                self.events.append(("docker", label, tuple(args)))
                return fixture.CommandResult(0, "", "", 1)

        class OrderedOps(fixture.FixtureResourceOps):
            def __init__(self, events, container, network):
                self.events = events
                self.container = container
                self.network = network

            def inspect_container(self, identifier):
                self.events.append(("container-inspect", identifier))
                return self.container

            def remove_container(self, identifier):
                raise AssertionError("cleanup is outside this start-order check")

            def inspect_network(self, identifier):
                self.events.append(("network-inspect", identifier))
                return self.network

            def remove_network(self, identifier):
                raise AssertionError("cleanup is outside this start-order check")

        def run_case(container_change=None, network_change=None, expect_sql=True):
            with tempfile.TemporaryDirectory() as temp:
                root = Path(temp)
                events = []
                container = valid_container()
                container["networks"] = {names["network"]: {"NetworkID": NETWORK_ID}}
                network = {
                    "id": NETWORK_ID,
                    "name": names["network"],
                    "internal": True,
                    "labels": dict(labels),
                    "containers": {CONTAINER_ID: {"Name": names["container"]}},
                }
                if container_change:
                    container_change(container)
                if network_change:
                    network_change(network)
                task_dir = root / "task"
                task_dir.mkdir()
                campaign = fixture.Campaign(root, "worker", TASK_ID, task_dir)
                campaign.docker = OrderedDocker(events)
                campaign.ops = OrderedOps(events, container, network)
                campaign.container_id = CONTAINER_ID
                campaign.network_id = NETWORK_ID
                campaign.receipt["plan"]["image"] = {"imageId": "sha256:" + "c" * 64}

                def mysql(label, sql, **kwargs):
                    events.append(("mysql", label, sql))
                    return fixture.CommandResult(0, "1\n", "", 1)

                campaign._mysql = mysql
                if expect_sql:
                    campaign._start_and_wait()
                    first_sql = next(index for index, event in enumerate(events) if event[0] == "mysql")
                    self.assertEqual(events[0][0:2], ("docker", "start exact fixture container"))
                    self.assertEqual(events[1], ("container-inspect", CONTAINER_ID))
                    self.assertEqual(events[2], ("network-inspect", NETWORK_ID))
                    self.assertGreater(first_sql, 2)
                    self.assertEqual(sum(event[0] == "container-inspect" for event in events), 2)
                    self.assertEqual(sum(event[0] == "network-inspect" for event in events), 2)
                else:
                    with self.assertRaises(fixture.FixtureError):
                        campaign._start_and_wait()
                    self.assertFalse(any(event[0] == "mysql" for event in events))

        run_case()
        run_case(container_change=lambda item: item.update(imageId="sha256:" + "e" * 64), expect_sql=False)
        run_case(container_change=lambda item: item.update(id="f" * 64), expect_sql=False)
        run_case(
            container_change=lambda item: item["networks"][names["network"]].update(NetworkID="e" * 64),
            expect_sql=False,
        )
        run_case(network_change=lambda item: item.update(id="e" * 64), expect_sql=False)
        run_case(network_change=lambda item: item.update(containers={}), expect_sql=False)
        run_case(network_change=lambda item: item.update(internal=False), expect_sql=False)
        run_case(container_change=lambda item: item["labels"].update({"ccmai.cvf.role": "reviewer"}), expect_sql=False)
        run_case(network_change=lambda item: item["labels"].update({"ccmai.cvf.role": "reviewer"}), expect_sql=False)

    def test_native_network_timeout_is_reconciled_or_retained_without_false_absence(self):
        names = fixture.names_for("worker", TASK_ID)
        for effect, expected_cleanup in ((True, "VERIFIED_CLEAN"), (False, "FAILED_CLEANUP")):
            with self.subTest(network_create_effect=effect), tempfile.TemporaryDirectory() as temp:
                root = Path(temp)
                daemon = MemoryDockerSubprocess(
                    timeout_network_create=True,
                    network_create_effect=effect,
                )
                campaign = self._campaign(root)

                def stop_before_container(_image):
                    raise fixture.FixtureError("TEST_STOP", "network-only create control")

                campaign._create_container = stop_before_container
                with mock.patch.object(fixture.subprocess, "run", side_effect=daemon):
                    result = campaign.run()
                self.assertNotEqual(result, 0)
                self.assertEqual(
                    campaign.receipt["runtime"]["networkCreate"],
                    "RECONCILED_OWN_CREATE_RESPONSE_UNKNOWN" if effect else "UNKNOWN_OWNERSHIP_OR_CREATE_OUTCOME",
                )
                self.assertEqual(campaign.receipt["cleanup"]["status"], expected_cleanup)
                self.assertFalse(any(call[1:2] == ["create"] for call in daemon.calls))
                self.assertTrue(
                    any(
                        record["label"] == "create owned internal network" and record["timedOut"]
                        for record in campaign.receipt["commands"]
                    )
                )
                create_index = next(
                    index for index, call in enumerate(daemon.calls) if call[1:3] == ["network", "create"]
                )
                reconcile_index = next(
                    index
                    for index, call in enumerate(daemon.calls)
                    if call[1:3] == ["network", "inspect"] and call[-1] == names["network"]
                )
                self.assertLess(create_index, reconcile_index)
                if effect:
                    self.assertTrue(campaign.receipt["cleanup"]["absenceVerified"])
                    remove_index = next(
                        index for index, call in enumerate(daemon.calls) if call[1:3] == ["network", "rm"]
                    )
                    self.assertGreater(remove_index, reconcile_index)
                else:
                    self.assertFalse(campaign.receipt["cleanup"]["absenceVerified"])
                    self.assertEqual(
                        campaign.receipt["cleanup"]["unresolvedResources"],
                        [{"resource": "network", "name": names["network"], "id": None}],
                    )

    def test_native_container_timeout_reconciles_owned_resource_and_cleans_both_ids(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            daemon = MemoryDockerSubprocess(timeout_container_create=True, container_create_effect=True)
            campaign = self._campaign(root)

            def stop_before_start():
                raise fixture.FixtureError("TEST_STOP", "create reconciliation control")

            campaign._start_and_wait = stop_before_start
            with mock.patch.object(fixture.subprocess, "run", side_effect=daemon):
                result = campaign.run()
            self.assertNotEqual(result, 0)
            self.assertEqual(campaign.receipt["runtime"]["containerCreate"], "RECONCILED_OWN_CREATE_RESPONSE_UNKNOWN")
            self.assertEqual(campaign.receipt["runtime"]["status"], "FAILED")
            self.assertEqual(campaign.receipt["cleanup"]["status"], "VERIFIED_CLEAN")
            self.assertTrue(campaign.receipt["cleanup"]["containerRemoved"])
            self.assertTrue(campaign.receipt["cleanup"]["networkRemoved"])
            self.assertTrue(campaign.receipt["cleanup"]["absenceVerified"])
            self.assertTrue(
                any(
                    record["label"] == "create owned tmpfs MySQL container" and record["timedOut"]
                    for record in campaign.receipt["commands"]
                )
            )
            container_create_index = next(index for index, call in enumerate(daemon.calls) if call[1:2] == ["create"])
            container_name = fixture.names_for("worker", TASK_ID)["container"]
            container_reconcile_index = next(
                index
                for index, call in enumerate(daemon.calls)
                if index > container_create_index
                and call[1:3] == ["container", "inspect"]
                and call[-1] == container_name
            )
            container_remove_index = next(index for index, call in enumerate(daemon.calls) if call[1:3] == ["container", "rm"])
            network_remove_index = next(index for index, call in enumerate(daemon.calls) if call[1:3] == ["network", "rm"])
            self.assertLess(container_create_index, container_reconcile_index)
            self.assertLess(container_reconcile_index, container_remove_index)
            self.assertLess(container_remove_index, network_remove_index)

    def test_unresolved_container_timeout_preserves_receipt_and_cleans_known_network(self):
        names = fixture.names_for("worker", TASK_ID)
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            daemon = MemoryDockerSubprocess(timeout_container_create=True, container_create_effect=False)
            campaign = self._campaign(root)
            with mock.patch.object(fixture.subprocess, "run", side_effect=daemon):
                result = campaign.run()
            self.assertNotEqual(result, 0)
            self.assertEqual(campaign.receipt["runtime"]["status"], "FAILED")
            self.assertEqual(campaign.receipt["runtime"]["failure"]["code"], "CONTAINER_CREATE_UNCERTAIN")
            self.assertEqual(campaign.receipt["cleanup"]["status"], "FAILED_CLEANUP")
            self.assertFalse(campaign.receipt["cleanup"]["absenceVerified"])
            self.assertTrue(campaign.receipt["cleanup"]["networkRemoved"])
            self.assertEqual(
                campaign.receipt["cleanup"]["unresolvedResources"],
                [{"resource": "container", "name": names["container"], "id": None}],
            )
            self.assertEqual(len(daemon.networks), 0)
            self.assertFalse(any(call[1:3] == ["container", "rm"] for call in daemon.calls))

    def test_thrown_container_remove_failure_keeps_ids_and_never_claims_absence(self):
        names = fixture.names_for("worker", TASK_ID)
        labels = fixture.expected_labels("worker", TASK_ID)

        class ThrowingRemoveOps(FakeResourceOps):
            def remove_container(self, identifier):
                self.removed.append(("container", identifier))
                raise RuntimeError("simulated remove failure")

        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            task_dir = root / "task"
            task_dir.mkdir()
            campaign = fixture.Campaign(root, "worker", TASK_ID, task_dir)
            network = {
                "id": NETWORK_ID,
                "name": names["network"],
                "labels": labels,
                "containers": {CONTAINER_ID: {"Name": names["container"]}},
            }
            ops = ThrowingRemoveOps(container=valid_container(), network=network)
            campaign.ops = ops
            campaign.container_id = CONTAINER_ID
            campaign.network_id = NETWORK_ID
            campaign.receipt["runtime"].update(
                {"status": "FAILED", "containerId": CONTAINER_ID, "networkId": NETWORK_ID}
            )
            campaign._cleanup()
            cleanup = campaign.receipt["cleanup"]
            self.assertEqual(cleanup["status"], "FAILED_CLEANUP")
            self.assertFalse(cleanup["absenceVerified"])
            self.assertEqual(cleanup["containerStatus"], "FAILED")
            self.assertEqual(cleanup["failureCode"], "CLEANUP_UNCERTAIN")
            self.assertEqual(campaign.receipt["plan"]["intendedResources"]["container"], names["container"])
            self.assertEqual(campaign.receipt["runtime"]["containerId"], CONTAINER_ID)
            self.assertEqual(campaign.receipt["runtime"]["networkId"], NETWORK_ID)
            self.assertEqual(ops.removed, [("container", CONTAINER_ID)])


if __name__ == "__main__":
    unittest.main()
