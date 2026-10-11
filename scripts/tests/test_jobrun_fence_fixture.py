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


if __name__ == "__main__":
    unittest.main()
