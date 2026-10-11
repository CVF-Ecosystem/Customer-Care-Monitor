#!/usr/bin/env python3
"""Run the bounded R083 disposable-MySQL fixture prerequisite.

This standard-library tool only creates its own internal Docker network,
tmpfs-backed MySQL container, schema and probe table. It does not inspect
container environment variables or use an existing DSN. Actual fixture
execution is intentionally separate from the source/unit-test BUILD stage.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import subprocess
import sys
import time
import uuid
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Iterable, Mapping, Sequence
from urllib.parse import urlsplit


TRANCHE_ID = "CCMAI-RUNTIME-083"
BUILD_ORDER_SNAPSHOT_COMMIT = "c23d84e0048218e1d91d93e9b6557345e3ec2690"
WORK_ORDER_PATH = "docs/work_orders/CCMAI_RUNTIME_083.md"
SOURCE_PATHS = (
    "scripts/jobrun_fence_fixture.py",
    "scripts/tests/test_jobrun_fence_fixture.py",
    "docs/reviews/R083_FIXTURE_PREREQUISITE_READINESS_2026-10-11.md",
)
OUTPUT_ROOT = "docs/reviews/probes/r083_fixture"
ROLES = ("worker", "reviewer")
MAX_COMMAND_SECONDS = 30
READINESS_ATTEMPTS = 60
READINESS_INTERVAL_SECONDS = 2
MAX_CAPTURE_BYTES = 1_000_000
MAX_RECEIPT_FACT_BYTES = 32_000

CONTAINER_INSPECT_TEMPLATE = (
    '{{printf "{\\"id\\":%q,\\"name\\":%q,\\"imageId\\":%q,\\"imageReference\\":%q,'
    '\\"labels\\":%s,\\"networkMode\\":%q,\\"networks\\":%s,'
    '\\"ports\\":%s,\\"binds\\":%s,\\"tmpfs\\":%s,\\"mounts\\":%s}" '
    '.Id .Name .Image .Config.Image (json .Config.Labels) .HostConfig.NetworkMode '
    '(json .NetworkSettings.Networks) (json .HostConfig.PortBindings) '
    '(json .HostConfig.Binds) (json .HostConfig.Tmpfs) (json .Mounts)}}'
)
NETWORK_INSPECT_TEMPLATE = (
    '{{printf "{\\"id\\":%q,\\"name\\":%q,\\"internal\\":%t,'
    '\\"labels\\":%s,\\"containers\\":%s}" .Id .Name .Internal '
    '(json .Labels) (json .Containers)}}'
)


class FixtureError(Exception):
    """A safe, non-secret failure suitable for a local receipt."""

    def __init__(
        self,
        code: str,
        message: str,
        *,
        command_label: str | None = None,
        exit_code: int | None = None,
        output_sha256: str | None = None,
        timed_out: bool = False,
    ) -> None:
        super().__init__(message)
        self.code = code
        self.message = message
        self.command_label = command_label
        self.exit_code = exit_code
        self.output_sha256 = output_sha256
        self.timed_out = timed_out


@dataclass(frozen=True)
class CommandResult:
    exit_code: int | None
    stdout: str
    stderr: str
    elapsed_ms: int
    timed_out: bool = False
    failure_code: str | None = None


def validate_role(role: str) -> str:
    if role not in ROLES:
        raise FixtureError("UNSAFE_ROLE", "role must be worker or reviewer")
    return role


def validate_task_id(task_id: str) -> str:
    if not re.fullmatch(r"[0-9a-f]{32}", task_id):
        raise FixtureError("UNSAFE_TASK_ID", "task id must be 32 lowercase hex characters")
    return task_id


def names_for(role: str, task_id: str) -> dict[str, str]:
    validate_role(role)
    validate_task_id(task_id)
    role_code = "w" if role == "worker" else "r"
    return {
        "network": f"ccmai-r083-{role_code}-{task_id}",
        "container": f"ccmai-r083-{role_code}-{task_id}",
        # Alphanumeric only: MySQL treats '_' and '%' as wildcard characters
        # in database-level grant scopes on some supported versions.
        "schema": f"c83{role_code}{task_id}",
        "principal": f"u83{role_code}{task_id[:28]}",
        "denySchema": f"d83{role_code}{task_id[:28]}",
        "probeTable": f"probe{task_id[:12]}",
    }


def _inside(root: Path, candidate: Path) -> bool:
    try:
        candidate.relative_to(root)
        return True
    except ValueError:
        return False


def create_task_output_directory(root: Path, role: str, task_id: str) -> Path:
    """Create only the fixed role subtree; reject symlinks and escapes."""
    validate_role(role)
    validate_task_id(task_id)
    root = root.resolve(strict=True)
    relative = Path(OUTPUT_ROOT) / role
    cursor = root
    for component in relative.parts:
        cursor = cursor / component
        if cursor.is_symlink():
            raise FixtureError("OUTPUT_PATH_SYMLINK", "fixture output path contains a symlink")
        if cursor.exists():
            if not cursor.is_dir():
                raise FixtureError("OUTPUT_PATH_NOT_DIRECTORY", "fixture output parent is not a directory")
        else:
            cursor.mkdir()
        resolved = cursor.resolve(strict=True)
        if not _inside(root, resolved):
            raise FixtureError("OUTPUT_PATH_ESCAPE", "fixture output path escaped the project root")
    task_dir = cursor / task_id
    if task_dir.is_symlink() or task_dir.exists():
        raise FixtureError("OUTPUT_TASK_EXISTS", "random task output directory already exists")
    task_dir.mkdir()
    if not _inside(root, task_dir.resolve(strict=True)):
        raise FixtureError("OUTPUT_PATH_ESCAPE", "fixture task directory escaped the project root")
    return task_dir


def _atomic_json_write(path: Path, value: Mapping[str, Any]) -> None:
    if path.is_symlink():
        raise FixtureError("OUTPUT_FILE_SYMLINK", "fixture receipt path is a symlink")
    temp = path.with_name(f".{path.name}.{uuid.uuid4().hex}.tmp")
    payload = (json.dumps(value, indent=2, sort_keys=True, ensure_ascii=True) + "\n").encode("utf-8")
    try:
        with temp.open("xb") as stream:
            stream.write(payload)
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temp, path)
    except OSError as exc:
        try:
            temp.unlink(missing_ok=True)
        except OSError:
            pass
        raise FixtureError("RECEIPT_WRITE_FAILED", "could not write the task-owned fixture receipt") from exc


def redact_text(text: str, secrets_to_redact: Iterable[str] = ()) -> str:
    cleaned = text
    for secret in sorted((item for item in secrets_to_redact if item), key=len, reverse=True):
        cleaned = cleaned.replace(secret, "[REDACTED]")
    cleaned = re.sub(
        r"(?i)\b(password|passwd|pwd|dsn)\b\s*([:=])\s*[^\s,;]+",
        r"\1\2[REDACTED]",
        cleaned,
    )
    cleaned = re.sub(r"(?i)(mysql://)[^\s]+", r"\1[REDACTED]", cleaned)
    return cleaned


def output_digest(stdout: str, stderr: str, secrets_to_redact: Iterable[str] = ()) -> str:
    safe = redact_text(stdout + "\n" + stderr, secrets_to_redact)
    return hashlib.sha256(safe.encode("utf-8", "replace")).hexdigest()


def validate_local_endpoint(endpoint: str) -> str:
    """Return a safe endpoint kind; all TCP/SSH contexts fail closed."""
    if endpoint.startswith("npipe://"):
        # Only Docker Desktop's explicit local named pipe is supported.
        if re.fullmatch(r"(?i)npipe:////\./pipe/[A-Za-z0-9_.-]+", endpoint):
            return "npipe-local"
        raise FixtureError("REMOTE_DOCKER_CONTEXT", "named-pipe Docker endpoint is remote or ambiguous")
    if endpoint.startswith("unix://"):
        parsed = urlsplit(endpoint)
        if (
            parsed.scheme == "unix"
            and not parsed.netloc
            and parsed.path.startswith("/")
            and not parsed.path.startswith("//")
            and not parsed.query
            and not parsed.fragment
        ):
            return "unix-local"
        raise FixtureError("REMOTE_DOCKER_CONTEXT", "Unix Docker endpoint is not an absolute local socket")
    if endpoint.startswith(("tcp://", "ssh://", "http://", "https://")):
        raise FixtureError("REMOTE_DOCKER_CONTEXT", "TCP and SSH Docker contexts are not allowed")
    raise FixtureError("UNSUPPORTED_DOCKER_CONTEXT", "Docker endpoint is not a supported local socket")


def parse_image_metadata(output: str) -> dict[str, str]:
    parts = output.strip().split("|", 3)
    if len(parts) != 4:
        raise FixtureError("IMAGE_METADATA_INVALID", "cached MySQL image metadata was incomplete")
    image_id, digests_json, os_name, architecture = parts
    if not re.fullmatch(r"sha256:[0-9a-f]{64}", image_id):
        raise FixtureError("IMAGE_ID_INVALID", "cached MySQL image id was not a full sha256 id")
    try:
        digests = json.loads(digests_json)
    except (json.JSONDecodeError, TypeError) as exc:
        raise FixtureError("IMAGE_DIGEST_INVALID", "cached MySQL repository digest metadata was invalid") from exc
    if not isinstance(digests, list) or not digests:
        raise FixtureError("IMAGE_DIGEST_MISSING", "cached MySQL image has no repository digest")
    canonical: set[str] = set()
    for value in digests:
        if not isinstance(value, str):
            raise FixtureError("IMAGE_DIGEST_INVALID", "cached MySQL repository digest metadata was invalid")
        match = re.fullmatch(r"(?:docker\.io/library/mysql|mysql)@(sha256:[0-9a-f]{64})", value)
        if not match:
            raise FixtureError("IMAGE_DIGEST_INVALID", "cached MySQL repository digest was not canonical")
        canonical.add(match.group(1))
    if len(canonical) != 1:
        raise FixtureError("IMAGE_DIGEST_AMBIGUOUS", "cached MySQL image has ambiguous repository digests")
    if not re.fullmatch(r"[a-z0-9_.-]+", os_name) or not re.fullmatch(r"[a-z0-9_.-]+", architecture):
        raise FixtureError("IMAGE_PLATFORM_INVALID", "cached MySQL platform metadata was invalid")
    return {
        "imageId": image_id,
        "repositoryDigest": f"mysql@{next(iter(canonical))}",
        "os": os_name,
        "architecture": architecture,
    }


def _read_only_process_environment() -> dict[str, str]:
    """Pass only path/context metadata to Docker; never forward app credentials."""
    env: dict[str, str] = {}
    for key in (
        "PATH",
        "SYSTEMROOT",
        "WINDIR",
        "HOME",
        "USERPROFILE",
        "APPDATA",
        "LOCALAPPDATA",
        "DOCKER_CONFIG",
        "DOCKER_CONTEXT",
        "DOCKER_HOST",
    ):
        value = os.environ.get(key)
        if value:
            env[key] = value
    if env.get("DOCKER_HOST") and env.get("DOCKER_CONTEXT"):
        raise FixtureError("DOCKER_CONTEXT_AMBIGUOUS", "both Docker host and context overrides are set")
    if env.get("DOCKER_HOST"):
        validate_local_endpoint(env["DOCKER_HOST"])
    return env


class DockerCli:
    """Small fixed-command adapter; argv is never emitted into receipts."""

    def __init__(self, root: Path, env: Mapping[str, str] | None = None) -> None:
        self.root = root
        self.env = dict(env) if env is not None else _read_only_process_environment()
        self.commands: list[dict[str, Any]] = []
        self._redactions: list[str] = []

    def add_redaction(self, value: str) -> None:
        if value and value not in self._redactions:
            self._redactions.append(value)

    def call(
        self,
        label: str,
        args: Sequence[str],
        *,
        timeout: int = MAX_COMMAND_SECONDS,
        input_text: str | None = None,
        check: bool = True,
        redactions: Iterable[str] = (),
        extra_env: Mapping[str, str] | None = None,
        record_output: bool = False,
    ) -> CommandResult:
        for value in redactions:
            self.add_redaction(value)
        command_env = dict(self.env)
        if extra_env:
            command_env.update(extra_env)
        started = time.monotonic()
        try:
            completed = subprocess.run(
                ["docker", *args],
                cwd=str(self.root),
                env=command_env,
                input=input_text,
                text=True,
                capture_output=True,
                timeout=timeout,
                check=False,
                shell=False,
            )
            elapsed = int((time.monotonic() - started) * 1000)
            stdout = completed.stdout or ""
            stderr = completed.stderr or ""
            if len(stdout.encode("utf-8", "replace")) + len(stderr.encode("utf-8", "replace")) > MAX_CAPTURE_BYTES:
                result = CommandResult(
                    completed.returncode, "", "", elapsed, False, "COMMAND_OUTPUT_TOO_LARGE"
                )
                self.commands.append({
                    "label": label,
                    "timeoutSeconds": timeout,
                    "exitCode": completed.returncode,
                    "timedOut": False,
                    "elapsedMs": elapsed,
                    "safeOutputSha256": output_digest(stdout, stderr, self._redactions),
                    "outputStatus": "OMITTED_LIMIT_EXCEEDED",
                })
                if check:
                    raise FixtureError(
                        "COMMAND_OUTPUT_TOO_LARGE",
                        "Docker command output exceeded the bounded capture limit",
                        command_label=label,
                        exit_code=completed.returncode,
                        output_sha256=output_digest(stdout, stderr, self._redactions),
                    )
                return result
            command_record: dict[str, Any] = {
                "label": label,
                "timeoutSeconds": timeout,
                "exitCode": completed.returncode,
                "timedOut": False,
                "elapsedMs": elapsed,
                "safeOutputSha256": output_digest(stdout, stderr, self._redactions),
            }
            if record_output:
                safe_stdout = redact_text(stdout, self._redactions)
                safe_stderr = redact_text(stderr, self._redactions)
                safe_bytes = len(safe_stdout.encode("utf-8", "replace")) + len(safe_stderr.encode("utf-8", "replace"))
                if safe_bytes > MAX_RECEIPT_FACT_BYTES:
                    command_record["outputStatus"] = "OMITTED_FACT_LIMIT_EXCEEDED"
                    self.commands.append(command_record)
                    result = CommandResult(completed.returncode, "", "", elapsed, False, "FACT_OUTPUT_TOO_LARGE")
                    if check:
                        raise FixtureError(
                            "FACT_OUTPUT_TOO_LARGE",
                            "sanitized MySQL facts exceeded the receipt limit",
                            command_label=label,
                            exit_code=completed.returncode,
                            output_sha256=command_record["safeOutputSha256"],
                        )
                    return result
                command_record["safeStdout"] = safe_stdout
                command_record["safeStderr"] = safe_stderr
            self.commands.append(command_record)
            result = CommandResult(completed.returncode, stdout, stderr, elapsed)
            if check and completed.returncode != 0:
                raise FixtureError(
                    "DOCKER_COMMAND_FAILED",
                    f"{label} failed with exit code {completed.returncode}",
                    command_label=label,
                    exit_code=completed.returncode,
                    output_sha256=output_digest(stdout, stderr, self._redactions),
                )
            return result
        except subprocess.TimeoutExpired as exc:
            elapsed = int((time.monotonic() - started) * 1000)
            stdout = _as_text(exc.stdout)
            stderr = _as_text(exc.stderr)
            digest = output_digest(stdout, stderr, self._redactions)
            self.commands.append({
                "label": label,
                "timeoutSeconds": timeout,
                "exitCode": None,
                "timedOut": True,
                "elapsedMs": elapsed,
                "safeOutputSha256": digest,
                "outputStatus": "OMITTED_TIMEOUT_PARTIAL",
            })
            result = CommandResult(None, "", "", elapsed, True, "DOCKER_COMMAND_TIMEOUT")
            if check:
                raise FixtureError(
                    "DOCKER_COMMAND_TIMEOUT",
                    f"{label} exceeded its {timeout}s timeout",
                    command_label=label,
                    output_sha256=digest,
                    timed_out=True,
                ) from None
            return result
        except OSError:
            elapsed = int((time.monotonic() - started) * 1000)
            digest = output_digest("", "docker executable unavailable", self._redactions)
            self.commands.append({
                "label": label,
                "timeoutSeconds": timeout,
                "exitCode": None,
                "timedOut": False,
                "elapsedMs": elapsed,
                "safeOutputSha256": digest,
                "outputStatus": "OMITTED_EXECUTABLE_UNAVAILABLE",
            })
            result = CommandResult(None, "", "", elapsed, False, "DOCKER_UNAVAILABLE")
            if check:
                raise FixtureError("DOCKER_UNAVAILABLE", "Docker CLI is unavailable") from None
            return result


def _as_text(value: str | bytes | None) -> str:
    if value is None:
        return ""
    if isinstance(value, bytes):
        return value.decode("utf-8", "replace")
    return value


def _git(root: Path, *args: str, binary: bool = False) -> str | bytes:
    try:
        result = subprocess.run(
            ["git", *args],
            cwd=str(root),
            capture_output=True,
            check=False,
            shell=False,
            text=not binary,
            timeout=10,
        )
    except (OSError, subprocess.TimeoutExpired):
        raise FixtureError("GIT_IDENTITY_UNAVAILABLE", "Git identity could not be read") from None
    if result.returncode != 0:
        raise FixtureError("GIT_IDENTITY_UNAVAILABLE", "Git identity command failed")
    return result.stdout


def _blob_sha1(data: bytes) -> str:
    header = f"blob {len(data)}\0".encode("ascii")
    return hashlib.sha1(header + data).hexdigest()


def _filtered_worktree_blob_sha1(root: Path, relative: str, data: bytes) -> str:
    """Hash worktree bytes after applying the path's configured Git clean filters."""
    try:
        result = subprocess.run(
            ["git", "hash-object", f"--path={relative}", "--stdin"],
            cwd=str(root),
            input=data,
            capture_output=True,
            check=False,
            shell=False,
            timeout=10,
        )
    except (OSError, subprocess.TimeoutExpired):
        raise FixtureError("GIT_IDENTITY_UNAVAILABLE", "Git filtered source identity could not be read") from None
    try:
        value = result.stdout.decode("ascii", "strict").strip() if result.returncode == 0 else ""
    except UnicodeDecodeError:
        value = ""
    if not re.fullmatch(r"[0-9a-f]{40}", value):
        raise FixtureError("GIT_IDENTITY_UNAVAILABLE", "Git filtered source identity command failed")
    return value


def collect_source_identity(root: Path) -> dict[str, Any]:
    root = root.resolve(strict=True)
    current_commit = str(_git(root, "rev-parse", "HEAD")).strip()
    if not re.fullmatch(r"[0-9a-f]{40}", current_commit):
        raise FixtureError("SOURCE_COMMIT_INVALID", "current Git commit was not a full SHA-1 id")
    try:
        ancestry = subprocess.run(
            ["git", "merge-base", "--is-ancestor", BUILD_ORDER_SNAPSHOT_COMMIT, current_commit],
            cwd=str(root),
            capture_output=True,
            check=False,
            shell=False,
            timeout=10,
        )
    except (OSError, subprocess.TimeoutExpired):
        raise FixtureError("GIT_IDENTITY_UNAVAILABLE", "Git ancestry could not be verified") from None
    if ancestry.returncode != 0:
        raise FixtureError("SOURCE_ORDER_ANCESTRY_INVALID", "current source commit is not descended from the BUILD order snapshot")
    changed = str(
        _git(root, "status", "--porcelain", "--untracked-files=no", "--", *SOURCE_PATHS, WORK_ORDER_PATH)
    )
    if changed.strip():
        raise FixtureError("SOURCE_NOT_COMMITTED", "fixture source/order files must match the committed BUILD snapshot")

    files: dict[str, Any] = {}
    for relative in (*SOURCE_PATHS, WORK_ORDER_PATH):
        path = root / relative
        if path.is_symlink() or not path.is_file():
            raise FixtureError("SOURCE_FILE_INVALID", f"required source file is missing or unsafe: {relative}")
        data = path.read_bytes()
        commit_blob = str(_git(root, "rev-parse", f"{current_commit}:{relative}")).strip()
        filtered_blob = _filtered_worktree_blob_sha1(root, relative, data)
        if filtered_blob != commit_blob:
            raise FixtureError("SOURCE_BLOB_MISMATCH", f"working file does not match the committed Git blob: {relative}")
        files[relative] = {
            "gitBlob": commit_blob,
            "filteredWorktreeBlob": filtered_blob,
            "physicalSha256": hashlib.sha256(data).hexdigest(),
            "byteLength": len(data),
        }

    snapshot_blob = str(_git(root, "rev-parse", f"{BUILD_ORDER_SNAPSHOT_COMMIT}:{WORK_ORDER_PATH}")).strip()
    snapshot_data = _git(root, "show", f"{BUILD_ORDER_SNAPSHOT_COMMIT}:{WORK_ORDER_PATH}", binary=True)
    if not isinstance(snapshot_data, bytes) or _blob_sha1(snapshot_data) != snapshot_blob:
        raise FixtureError("BUILD_ORDER_SNAPSHOT_INVALID", "committed BUILD order snapshot could not be verified")
    snapshot_text = snapshot_data.decode("utf-8", "replace")
    if not re.search(r"(?m)^Status:\s*BUILD\s*$", snapshot_text):
        raise FixtureError("BUILD_ORDER_SNAPSHOT_INVALID", "pinned BUILD order snapshot is not a BUILD work order")

    return {
        "sourceCommit": current_commit,
        "buildOrderSnapshot": {
            "commit": BUILD_ORDER_SNAPSHOT_COMMIT,
            "path": WORK_ORDER_PATH,
            "gitBlob": snapshot_blob,
            "physicalSha256": hashlib.sha256(snapshot_data).hexdigest(),
            "byteLength": len(snapshot_data),
        },
        "currentFiles": files,
    }


def _parse_template_json(output: str, label: str) -> dict[str, Any]:
    try:
        value = json.loads(output.strip())
    except (json.JSONDecodeError, TypeError) as exc:
        raise FixtureError("INSPECT_OUTPUT_INVALID", f"{label} returned invalid selected-field metadata") from exc
    if not isinstance(value, dict):
        raise FixtureError("INSPECT_OUTPUT_INVALID", f"{label} returned invalid selected-field metadata")
    return value


def expected_labels(role: str, task_id: str) -> dict[str, str]:
    validate_role(role)
    validate_task_id(task_id)
    return {
        "ccmai.cvf.tranche": TRANCHE_ID,
        "ccmai.cvf.purpose": "fixture-prerequisite",
        "ccmai.cvf.role": role,
        "ccmai.cvf.task-id": task_id,
    }


def labels_match(value: Mapping[str, Any], expected: Mapping[str, str]) -> bool:
    """Require every task-owner label while permitting inherited image labels."""
    observed = value.get("labels")
    return isinstance(observed, Mapping) and all(observed.get(key) == item for key, item in expected.items())


def selected_owner_labels(value: Mapping[str, Any], expected: Mapping[str, str]) -> dict[str, Any]:
    observed = value.get("labels")
    if not isinstance(observed, Mapping) or not labels_match(value, expected):
        raise FixtureError("RESOURCE_OWNERSHIP_MISMATCH", "required task-owner labels did not match")
    return {key: observed[key] for key in expected}


def validate_network_inspect(
    value: Mapping[str, Any],
    *,
    name: str,
    labels: Mapping[str, str],
    expected_id: str | None = None,
    expected_container_id: str | None = None,
    expected_container_name: str | None = None,
) -> str:
    network_id = value.get("id")
    if not isinstance(network_id, str) or not re.fullmatch(r"[0-9a-f]{64}", network_id):
        raise FixtureError("NETWORK_ID_INVALID", "owned network id was not a full id")
    if (
        value.get("name") != name
        or value.get("internal") is not True
        or not labels_match(value, labels)
        or (expected_id is not None and network_id != expected_id)
    ):
        raise FixtureError("NETWORK_OWNERSHIP_MISMATCH", "network ownership or internal status did not match")
    containers = value.get("containers")
    if containers is not None and not isinstance(containers, dict):
        raise FixtureError("NETWORK_MEMBERSHIP_INVALID", "network membership metadata was invalid")
    if expected_container_id is not None:
        if not isinstance(containers, dict) or set(containers) != {expected_container_id}:
            raise FixtureError("NETWORK_MEMBERSHIP_MISMATCH", "network did not contain only the exact fixture container")
        member = containers[expected_container_id]
        member_name = member.get("Name") if isinstance(member, Mapping) else None
        if not isinstance(member_name, str) or member_name.lstrip("/") != expected_container_name:
            raise FixtureError("NETWORK_MEMBERSHIP_INVALID", "network membership did not identify a selected container")
    return network_id


def validate_container_inspect(
    value: Mapping[str, Any],
    *,
    name: str,
    image_id: str,
    network_name: str,
    labels: Mapping[str, str],
    expected_id: str | None = None,
    expected_network_id: str | None = None,
    require_network_id: bool = False,
) -> str:
    container_id = value.get("id")
    if not isinstance(container_id, str) or not re.fullmatch(r"[0-9a-f]{64}", container_id):
        raise FixtureError("CONTAINER_ID_INVALID", "owned container id was not a full id")
    observed_name = value.get("name")
    if (
        not isinstance(observed_name, str)
        or observed_name.lstrip("/") != name
        or value.get("imageId") != image_id
        or not labels_match(value, labels)
        or value.get("networkMode") != network_name
        or (expected_id is not None and container_id != expected_id)
    ):
        raise FixtureError("CONTAINER_OWNERSHIP_MISMATCH", "container identity or ownership did not match")
    ports = value.get("ports")
    if ports not in (None, {}):
        raise FixtureError("HOST_PORT_FORBIDDEN", "fixture container has a host port binding")
    binds = value.get("binds")
    if binds not in (None, []):
        raise FixtureError("HOST_BIND_FORBIDDEN", "fixture container has a host bind")
    networks = value.get("networks")
    if not isinstance(networks, dict) or set(networks) != {network_name}:
        raise FixtureError("CONTAINER_NETWORK_MISMATCH", "fixture container is attached to an unexpected network")
    network = networks[network_name]
    if not isinstance(network, Mapping):
        raise FixtureError("CONTAINER_NETWORK_MISMATCH", "fixture container network metadata was invalid")
    observed_network_id = network.get("NetworkID", "")
    if require_network_id:
        if not expected_network_id or observed_network_id != expected_network_id:
            raise FixtureError("CONTAINER_NETWORK_ID_MISMATCH", "started container network id did not match the owned network")
    elif observed_network_id not in (None, "", expected_network_id):
        raise FixtureError("CONTAINER_NETWORK_ID_MISMATCH", "pending container network id did not match the owned network")
    tmpfs = value.get("tmpfs")
    if not isinstance(tmpfs, dict) or set(tmpfs) != {"/var/lib/mysql"}:
        raise FixtureError("TMPFS_INVALID", "fixture container must use only the authorized MySQL tmpfs")
    tmpfs_options = tmpfs.get("/var/lib/mysql")
    if not isinstance(tmpfs_options, str):
        raise FixtureError("TMPFS_INVALID", "fixture MySQL tmpfs options were missing")
    option_set = {part.strip().lower() for part in tmpfs_options.split(",") if part.strip()}
    if option_set != {"rw", "noexec", "nosuid", "size=512m"}:
        raise FixtureError("TMPFS_INVALID", "fixture MySQL tmpfs options did not match the authorized limits")
    mounts = value.get("mounts")
    if not isinstance(mounts, list) or len(mounts) != 1:
        raise FixtureError("MOUNT_INVALID", "fixture container must have exactly one tmpfs mount")
    mount = mounts[0]
    if not isinstance(mount, dict) or mount.get("Type") != "tmpfs" or mount.get("Destination") != "/var/lib/mysql":
        raise FixtureError("MOUNT_INVALID", "fixture container has an unauthorized mount")
    if mount.get("RW") is not True:
        raise FixtureError("MOUNT_INVALID", "MySQL tmpfs is not writable")
    return container_id


def _quoted_account(value: str) -> str:
    if len(value) >= 2 and value[0] == value[-1] and value[0] in ("'", "`"):
        return value[1:-1]
    return value


def parse_grant_scope_lines(lines: Sequence[str], *, schema: str, principal: str) -> dict[str, Any]:
    if not re.fullmatch(r"[a-z0-9]{1,64}", schema) or not re.fullmatch(r"[a-z0-9]{1,32}", principal):
        raise FixtureError("GRANT_IDENTITY_UNSAFE", "schema or principal contains unsafe grant characters")
    expected_privileges = {"CREATE", "DELETE", "DROP", "INSERT", "SELECT", "UPDATE"}
    seen_usage = False
    seen_schema = False
    normalized: list[dict[str, Any]] = []
    for line in lines:
        text = line.strip().rstrip(";")
        match = re.fullmatch(r"GRANT\s+(.+?)\s+ON\s+(.+?)\s+TO\s+(.+?)", text, re.IGNORECASE)
        if not match:
            raise FixtureError("GRANT_SCOPE_UNSAFE", "fixture principal grants were not safely parseable")
        privilege_text, scope, account = match.groups()
        account_match = re.fullmatch(r"(.+?)@(.+)", account)
        if not account_match:
            raise FixtureError("GRANT_SCOPE_UNSAFE", "fixture principal grant identity was invalid")
        grant_user = _quoted_account(account_match.group(1))
        grant_host = _quoted_account(account_match.group(2))
        if grant_user != principal or grant_host != "127.0.0.1":
            raise FixtureError("GRANT_SCOPE_UNSAFE", "fixture principal grants reference an unexpected account")
        privileges = {item.strip().upper() for item in privilege_text.split(",")}
        if scope == "*.*" and privileges == {"USAGE"}:
            if seen_usage:
                raise FixtureError("GRANT_SCOPE_UNSAFE", "duplicate global usage grant")
            seen_usage = True
            normalized.append({"scope": "global", "privileges": ["USAGE"]})
            continue
        scope_match = re.fullmatch(r"[`']?([a-z0-9]+)[`']?\.\*", scope)
        if not scope_match or scope_match.group(1) != schema or privileges != expected_privileges:
            raise FixtureError("GRANT_SCOPE_UNSAFE", "fixture principal has an unexpected or wildcard grant")
        if seen_schema:
            raise FixtureError("GRANT_SCOPE_UNSAFE", "fixture principal has duplicate schema grants")
        seen_schema = True
        normalized.append({"scope": "task-schema", "privileges": sorted(privileges)})
    if not seen_usage or not seen_schema or len(normalized) != 2:
        raise FixtureError("GRANT_SCOPE_UNSAFE", "fixture principal grants do not match the exact task schema")
    return {"grants": normalized, "noGlobalOrUnownedSchemaPrivileges": True}


def expected_privilege_denial(stderr: str, *, principal: str, schema: str) -> bool:
    """Accept only MySQL's exact database-privilege denial for the task account/schema."""
    if not re.fullmatch(r"[a-z0-9]{1,32}", principal) or not re.fullmatch(r"[a-z0-9]{1,64}", schema):
        return False
    account = rf"Access denied for user '{re.escape(principal)}'@'127\.0\.0\.1' to database '{re.escape(schema)}'"
    patterns = (
        rf"ERROR 1044 \(42000\): {account}(?: at line 1)?",
        rf"ERROR 1044 \(42000\) at line 1: {account}",
    )
    return any(re.fullmatch(pattern, stderr.strip()) is not None for pattern in patterns)


def reconcile_unknown_creation(
    *,
    result: CommandResult,
    intended_name: str,
    expected_labels_value: Mapping[str, str],
    inspect_exact_name: Any,
) -> tuple[str | None, str, str | None]:
    """Reconcile only the exact random name after our own create call is uncertain."""
    if result.exit_code == 0 and not result.timed_out:
        value = result.stdout.strip()
        if re.fullmatch(r"[0-9a-f]{64}", value):
            return value, "CONFIRMED", value
    exact = inspect_exact_name(intended_name)
    if exact is not None and labels_match(exact, expected_labels_value):
        resource_id = exact.get("id")
        if isinstance(resource_id, str) and re.fullmatch(r"[0-9a-f]{64}", resource_id):
            return resource_id, "RECONCILED_OWN_CREATE_RESPONSE_UNKNOWN", resource_id
    if exact is None and result.exit_code not in (None, 0) and not result.timed_out:
        return None, "CREATE_FAILED_AND_ABSENT", None
    observed_id = exact.get("id") if isinstance(exact, Mapping) else None
    if not isinstance(observed_id, str) or not re.fullmatch(r"[0-9a-f]{64}", observed_id):
        observed_id = None
    return None, "UNKNOWN_OWNERSHIP_OR_CREATE_OUTCOME", observed_id


class FixtureResourceOps:
    """Narrow cleanup interface shared by the campaign and portable tests."""

    def inspect_container(self, identifier: str) -> Mapping[str, Any] | None:
        raise NotImplementedError

    def remove_container(self, identifier: str) -> None:
        raise NotImplementedError

    def inspect_network(self, identifier: str) -> Mapping[str, Any] | None:
        raise NotImplementedError

    def remove_network(self, identifier: str) -> None:
        raise NotImplementedError


def cleanup_owned_resources(
    ops: FixtureResourceOps,
    *,
    container_id: str | None,
    network_id: str | None,
    container_expected: Mapping[str, Any],
    network_expected: Mapping[str, Any],
) -> dict[str, Any]:
    """Clean each known resource independently; never let one error hide another cleanup."""
    result: dict[str, Any] = {
        "status": "VERIFIED_CLEAN",
        "containerRemoved": False,
        "networkRemoved": False,
        "containerStatus": "NOT_REQUIRED",
        "networkStatus": "NOT_REQUIRED",
    }
    failures: list[dict[str, str]] = []
    if container_id:
        try:
            observed = ops.inspect_container(container_id)
            if observed is not None:
                if not labels_match(observed, container_expected["labels"]):
                    raise FixtureError("CLEANUP_OWNERSHIP_MISMATCH", "container ownership could not be verified")
                observed_name = observed.get("name")
                if (
                    observed.get("id") != container_id
                    or not isinstance(observed_name, str)
                    or observed_name.lstrip("/") != container_expected["name"]
                    or (
                        container_expected.get("imageId") is not None
                        and observed.get("imageId") != container_expected["imageId"]
                    )
                ):
                    raise FixtureError("CLEANUP_ID_MISMATCH", "container id or name changed before cleanup")
                networks = observed.get("networks")
                if networks is not None and (
                    not isinstance(networks, Mapping)
                    or set(networks) - {network_expected["name"]}
                ):
                    raise FixtureError("CLEANUP_NETWORK_MISMATCH", "container was attached to an unexpected network")
                ops.remove_container(container_id)
                if ops.inspect_container(container_id) is not None:
                    raise FixtureError("CLEANUP_CONTAINER_REMAINS", "owned container still exists after removal")
                result["containerRemoved"] = True
            result["containerStatus"] = "ABSENT_VERIFIED"
        except FixtureError as exc:
            failures.append({"resource": "container", "code": exc.code, "safeMessage": exc.message})
            result["containerStatus"] = "FAILED"
        except Exception:
            failures.append({"resource": "container", "code": "CLEANUP_UNCERTAIN", "safeMessage": "container cleanup outcome is uncertain"})
            result["containerStatus"] = "FAILED"
    if network_id:
        try:
            observed_network = ops.inspect_network(network_id)
            if observed_network is not None:
                if not labels_match(observed_network, network_expected["labels"]):
                    raise FixtureError("CLEANUP_OWNERSHIP_MISMATCH", "network ownership could not be verified")
                if observed_network.get("id") != network_id or observed_network.get("name") != network_expected["name"]:
                    raise FixtureError("CLEANUP_ID_MISMATCH", "network id or name changed before cleanup")
                members = observed_network.get("containers") or {}
                if members:
                    raise FixtureError("CLEANUP_NETWORK_IN_USE", "owned network still has attached containers")
                ops.remove_network(network_id)
                if ops.inspect_network(network_id) is not None:
                    raise FixtureError("CLEANUP_NETWORK_REMAINS", "owned network still exists after removal")
                result["networkRemoved"] = True
            result["networkStatus"] = "ABSENT_VERIFIED"
        except FixtureError as exc:
            failures.append({"resource": "network", "code": exc.code, "safeMessage": exc.message})
            result["networkStatus"] = "FAILED"
        except Exception:
            failures.append({"resource": "network", "code": "CLEANUP_UNCERTAIN", "safeMessage": "network cleanup outcome is uncertain"})
            result["networkStatus"] = "FAILED"
    if failures:
        result.update({"status": "FAILED_CLEANUP", "failures": failures})
        result["failureCode"] = failures[0]["code"]
        result["safeMessage"] = failures[0]["safeMessage"]
    return result


class DockerResourceOps(FixtureResourceOps):
    def __init__(
        self,
        docker: DockerCli,
        *,
        expected_network_name: str,
        network_labels: Mapping[str, str],
        container_name: str,
        image_id: str,
        container_labels: Mapping[str, str],
    ) -> None:
        self.docker = docker
        self.expected_network_name = expected_network_name
        self.network_labels = dict(network_labels)
        self.container_name = container_name
        self.image_id = image_id
        self.container_labels = dict(container_labels)

    def inspect_container(self, identifier: str) -> Mapping[str, Any] | None:
        result = self.docker.call(
            "inspect exact fixture container",
            ["container", "inspect", "--format", CONTAINER_INSPECT_TEMPLATE, identifier],
            check=False,
        )
        if result.failure_code:
            raise FixtureError("CONTAINER_INSPECT_UNCERTAIN", "exact container inspection was uncertain")
        if result.exit_code != 0:
            if _known_not_found(result.stderr, resource="container", identifier=identifier):
                return None
            raise FixtureError("CONTAINER_INSPECT_UNCERTAIN", "exact container inspection was uncertain")
        value = _parse_template_json(result.stdout, "container inspect")
        # Source host paths are discarded immediately and never enter the receipt.
        return value

    def remove_container(self, identifier: str) -> None:
        self.docker.call("remove exact fixture container", ["container", "rm", "--force", identifier])

    def inspect_network(self, identifier: str) -> Mapping[str, Any] | None:
        result = self.docker.call(
            "inspect exact fixture network",
            ["network", "inspect", "--format", NETWORK_INSPECT_TEMPLATE, identifier],
            check=False,
        )
        if result.failure_code:
            raise FixtureError("NETWORK_INSPECT_UNCERTAIN", "exact network inspection was uncertain")
        if result.exit_code != 0:
            if _known_not_found(result.stderr, resource="network", identifier=identifier):
                return None
            raise FixtureError("NETWORK_INSPECT_UNCERTAIN", "exact network inspection was uncertain")
        return _parse_template_json(result.stdout, "network inspect")

    def remove_network(self, identifier: str) -> None:
        self.docker.call("remove exact fixture network", ["network", "rm", identifier])


def _known_not_found(stderr: str, *, resource: str, identifier: str) -> bool:
    """Accept only an exact resource-specific daemon absence response."""
    escaped = re.escape(identifier)
    if resource == "network":
        patterns = (
            rf"Error response from daemon: network {escaped} not found\.?",
            rf"Error response from daemon: No such network: {escaped}\.?",
            rf"Error: No such network: {escaped}\.?",
        )
    elif resource == "container":
        patterns = (
            rf"Error response from daemon: container {escaped} not found\.?",
            rf"Error response from daemon: No such container: {escaped}\.?",
            rf"Error: No such container: {escaped}\.?",
        )
    else:
        return False
    text = stderr.strip()
    return any(re.fullmatch(pattern, text, re.IGNORECASE) for pattern in patterns)


class Campaign:
    def __init__(self, root: Path, role: str, task_id: str, task_dir: Path) -> None:
        self.root = root.resolve(strict=True)
        self.role = validate_role(role)
        self.task_id = validate_task_id(task_id)
        self.names = names_for(self.role, self.task_id)
        self.labels = expected_labels(self.role, self.task_id)
        self.task_dir = task_dir
        self.manifest_path = task_dir / "manifest.json"
        self.docker: DockerCli | None = None
        self.ops: DockerResourceOps | None = None
        self.network_id: str | None = None
        self.container_id: str | None = None
        self.network_create_attempted = False
        self.container_create_attempted = False
        self.uncertain_create_outcome = False
        self.unresolved_create_resources: list[dict[str, Any]] = []
        self.root_password: str | None = None
        self.password: str | None = None
        self.receipt: dict[str, Any] = {
            "schemaVersion": "1.0",
            "trancheId": TRANCHE_ID,
            "disposition": "PENDING_INDEPENDENT_REVIEW",
            "role": self.role,
            "taskId": self.task_id,
            "invocation": {"commandLabel": f"python scripts/jobrun_fence_fixture.py --role {self.role}"},
            "plan": {
                "status": "PENDING_PREFLIGHT",
                "intendedResources": self.names,
                "labels": self.labels,
                "imageLookup": "cached mysql:8.0 only; no pull",
            },
            "runtime": {"status": "NOT_RUN", "readinessObservations": 0, "facts": {}},
            "cleanup": {"status": "NOT_REQUIRED", "containerRemoved": False, "networkRemoved": False},
            "commands": [],
        }

    def persist(self) -> None:
        if self.docker:
            self.receipt["commands"] = list(self.docker.commands)
        _atomic_json_write(self.manifest_path, self.receipt)

    def prepare(self) -> tuple[dict[str, Any], str]:
        identity = collect_source_identity(self.root)
        env = _read_only_process_environment()
        docker = DockerCli(self.root, env)
        self.docker = docker
        context = self._validate_local_docker(docker)
        engine = docker.call("check local Docker availability", ["version", "--format", "{{.Server.Version}}"], timeout=15)
        engine_version = engine.stdout.strip()
        if not engine_version or not re.fullmatch(r"[0-9A-Za-z.+_-]+", engine_version):
            raise FixtureError("DOCKER_VERSION_INVALID", "local Docker availability response was invalid")
        image_out = docker.call(
            "inspect cached mysql:8.0 metadata",
            ["image", "inspect", "mysql:8.0", "--format", "{{.Id}}|{{json .RepoDigests}}|{{.Os}}|{{.Architecture}}"],
            timeout=20,
        )
        image = parse_image_metadata(image_out.stdout)
        network_existing = docker.call(
            "check exact fixture network name is unused",
            ["network", "inspect", "--format", "{{.Id}}", self.names["network"]],
            check=False,
        )
        if network_existing.failure_code:
            raise FixtureError("NETWORK_NAME_CHECK_UNCERTAIN", "could not prove random task network name is unused")
        if network_existing.exit_code == 0:
            raise FixtureError("NETWORK_NAME_EXISTS", "random task network name already exists; refusing adoption")
        if not _known_not_found(network_existing.stderr, resource="network", identifier=self.names["network"]):
            raise FixtureError("NETWORK_NAME_CHECK_UNCERTAIN", "could not prove random task network name is unused")
        container_existing = docker.call(
            "check exact fixture container name is unused",
            ["container", "inspect", "--format", "{{.Id}}", self.names["container"]],
            check=False,
        )
        if container_existing.failure_code:
            raise FixtureError("CONTAINER_NAME_CHECK_UNCERTAIN", "could not prove random task container name is unused")
        if container_existing.exit_code == 0:
            raise FixtureError("CONTAINER_NAME_EXISTS", "random task container name already exists; refusing adoption")
        if not _known_not_found(container_existing.stderr, resource="container", identifier=self.names["container"]):
            raise FixtureError("CONTAINER_NAME_CHECK_UNCERTAIN", "could not prove random task container name is unused")
        self.receipt["sourceIdentity"] = identity
        self.receipt["plan"].update(
            {
                "status": "READY_FOR_ROOT_EFFECT_APPROVAL",
                "dockerContext": context,
                "dockerEngineVersion": engine_version,
                "image": image,
                "intendedResources": self.names,
                "labels": self.labels,
                "externalEffects": "none; local Docker context only",
            }
        )
        self.receipt["runtime"]["status"] = "PENDING_APPROVED_FIXTURE_CAMPAIGN"
        self.persist()
        return image, engine_version

    def _validate_local_docker(self, docker: DockerCli) -> dict[str, str]:
        if docker.env.get("DOCKER_HOST"):
            endpoint = docker.env["DOCKER_HOST"]
            endpoint_kind = validate_local_endpoint(endpoint)
            context_name = "DOCKER_HOST override"
        else:
            context_result = docker.call("read current Docker context name", ["context", "show"], timeout=10)
            context_name = context_result.stdout.strip()
            if not re.fullmatch(r"[A-Za-z0-9_.-]{1,128}", context_name):
                raise FixtureError("DOCKER_CONTEXT_INVALID", "current Docker context name was invalid")
            endpoint_result = docker.call(
                "read safe current Docker endpoint metadata",
                ["context", "inspect", "--format", "{{.Endpoints.docker.Host}}", context_name],
                timeout=10,
            )
            endpoint = endpoint_result.stdout.strip()
            endpoint_kind = validate_local_endpoint(endpoint)
        return {"name": context_name, "endpointKind": endpoint_kind, "localOnly": "true"}

    def _inspect_network_name(self, name: str) -> Mapping[str, Any] | None:
        assert self.docker is not None
        result = self.docker.call(
            "reconcile exact fixture network name",
            ["network", "inspect", "--format", NETWORK_INSPECT_TEMPLATE, name],
            check=False,
        )
        if result.failure_code:
            raise FixtureError("NETWORK_RECONCILE_UNCERTAIN", "exact network create response could not be reconciled")
        if result.exit_code != 0:
            if _known_not_found(result.stderr, resource="network", identifier=name):
                return None
            raise FixtureError("NETWORK_RECONCILE_UNCERTAIN", "exact network create response could not be reconciled")
        return _parse_template_json(result.stdout, "network reconcile")

    def _inspect_container_name(self, name: str) -> Mapping[str, Any] | None:
        assert self.docker is not None
        result = self.docker.call(
            "reconcile exact fixture container name",
            ["container", "inspect", "--format", CONTAINER_INSPECT_TEMPLATE, name],
            check=False,
        )
        if result.failure_code:
            raise FixtureError("CONTAINER_RECONCILE_UNCERTAIN", "exact container create response could not be reconciled")
        if result.exit_code != 0:
            if _known_not_found(result.stderr, resource="container", identifier=name):
                return None
            raise FixtureError("CONTAINER_RECONCILE_UNCERTAIN", "exact container create response could not be reconciled")
        return _parse_template_json(result.stdout, "container reconcile")

    def _create_network(self) -> None:
        assert self.docker is not None
        args = ["network", "create", "--driver", "bridge", "--internal"]
        for key, value in self.labels.items():
            args.extend(["--label", f"{key}={value}"])
        args.append(self.names["network"])
        self.receipt["runtime"]["networkCreate"] = "PENDING"
        self.unresolved_create_resources.append({"resource": "network", "name": self.names["network"], "id": None})
        self.uncertain_create_outcome = True
        self.persist()
        self.network_create_attempted = True
        result = self.docker.call("create owned internal network", args, timeout=20, check=False)
        try:
            resource_id, disposition, observed_id = reconcile_unknown_creation(
                result=result,
                intended_name=self.names["network"],
                expected_labels_value=self.labels,
                inspect_exact_name=self._inspect_network_name,
            )
        except FixtureError:
            disposition, resource_id, observed_id = "UNKNOWN_OWNERSHIP_OR_CREATE_OUTCOME", None, None
        if resource_id is None:
            self.receipt["runtime"]["networkCreate"] = disposition
            if disposition == "UNKNOWN_OWNERSHIP_OR_CREATE_OUTCOME":
                self.unresolved_create_resources[-1]["id"] = observed_id
            else:
                self.unresolved_create_resources.pop()
            self.uncertain_create_outcome = bool(self.unresolved_create_resources)
            self.receipt["runtime"]["unresolvedCreateResources"] = list(self.unresolved_create_resources)
            self.persist()
            if disposition == "CREATE_FAILED_AND_ABSENT":
                raise FixtureError("NETWORK_CREATE_FAILED", "owned internal network creation failed and exact-name absence was verified")
            raise FixtureError("NETWORK_CREATE_UNCERTAIN", "owned internal network creation outcome is uncertain")
        self.network_id = resource_id
        self.unresolved_create_resources.pop()
        self.uncertain_create_outcome = bool(self.unresolved_create_resources)
        self.receipt["runtime"]["networkCreate"] = disposition
        self.receipt["runtime"]["networkId"] = resource_id
        self.persist()
        observed = self._inspect_network_name(self.names["network"])
        if observed is None:
            raise FixtureError("NETWORK_DISAPPEARED", "owned internal network disappeared after creation")
        validate_network_inspect(
            observed,
            name=self.names["network"],
            labels=self.labels,
            expected_id=self.network_id,
        )

    def _create_container(self, image: Mapping[str, str]) -> None:
        assert self.docker is not None
        self.root_password = uuid.uuid4().hex + uuid.uuid4().hex
        self.docker.add_redaction(self.root_password)
        args = [
            "create",
            "--pull=never",
            "--name",
            self.names["container"],
            "--network",
            self.names["network"],
            "--tmpfs",
            "/var/lib/mysql:rw,noexec,nosuid,size=512m",
            "--env",
            "MYSQL_ROOT_PASSWORD",
        ]
        for key, value in self.labels.items():
            args.extend(["--label", f"{key}={value}"])
        args.append(image["imageId"])
        self.receipt["runtime"]["containerCreate"] = "PENDING"
        self.unresolved_create_resources.append({"resource": "container", "name": self.names["container"], "id": None})
        self.uncertain_create_outcome = True
        self.persist()
        self.container_create_attempted = True
        result = self.docker.call(
            "create owned tmpfs MySQL container",
            args,
            timeout=30,
            check=False,
            redactions=(self.root_password,),
            extra_env={"MYSQL_ROOT_PASSWORD": self.root_password},
        )
        try:
            resource_id, disposition, observed_id = reconcile_unknown_creation(
                result=result,
                intended_name=self.names["container"],
                expected_labels_value=self.labels,
                inspect_exact_name=self._inspect_container_name,
            )
        except FixtureError:
            disposition, resource_id, observed_id = "UNKNOWN_OWNERSHIP_OR_CREATE_OUTCOME", None, None
        if resource_id is None:
            self.receipt["runtime"]["containerCreate"] = disposition
            if disposition == "UNKNOWN_OWNERSHIP_OR_CREATE_OUTCOME":
                self.unresolved_create_resources[-1]["id"] = observed_id
            else:
                self.unresolved_create_resources.pop()
            self.uncertain_create_outcome = bool(self.unresolved_create_resources)
            self.receipt["runtime"]["unresolvedCreateResources"] = list(self.unresolved_create_resources)
            self.persist()
            if disposition == "CREATE_FAILED_AND_ABSENT":
                raise FixtureError("CONTAINER_CREATE_FAILED", "owned MySQL container creation failed and exact-name absence was verified")
            raise FixtureError("CONTAINER_CREATE_UNCERTAIN", "owned MySQL container creation outcome is uncertain")
        self.container_id = resource_id
        self.unresolved_create_resources.pop()
        self.uncertain_create_outcome = bool(self.unresolved_create_resources)
        self.receipt["runtime"]["containerCreate"] = disposition
        self.receipt["runtime"]["containerId"] = resource_id
        self.persist()
        self.ops = DockerResourceOps(
            self.docker,
            expected_network_name=self.names["network"],
            network_labels=self.labels,
            container_name=self.names["container"],
            image_id=image["imageId"],
            container_labels=self.labels,
        )
        observed = self.ops.inspect_container(resource_id)
        if observed is None:
            raise FixtureError("CONTAINER_DISAPPEARED", "owned MySQL container disappeared after creation")
        validate_container_inspect(
            observed,
            name=self.names["container"],
            image_id=image["imageId"],
            network_name=self.names["network"],
            labels=self.labels,
            expected_id=self.container_id,
            expected_network_id=self.network_id,
        )

    def _start_and_wait(self) -> None:
        assert self.docker is not None and self.container_id is not None
        self.docker.call("start exact fixture container", ["container", "start", self.container_id], timeout=30)
        ready = False
        for attempt in range(READINESS_ATTEMPTS):
            self.receipt["runtime"]["readinessObservations"] = attempt + 1
            self.persist()
            result = self._mysql(
                "authenticated final-daemon readiness observation",
                "SELECT 1;\n",
                check=False,
                timeout=5,
            )
            if result.exit_code == 0 and result.stdout.strip() == "1":
                ready = True
                break
            if attempt + 1 < READINESS_ATTEMPTS:
                time.sleep(READINESS_INTERVAL_SECONDS)
        if not ready:
            raise FixtureError("MYSQL_NOT_READY", "MySQL did not become ready within the bounded observations")
        assert self.ops is not None
        observed = self.ops.inspect_container(self.container_id)
        if observed is None:
            raise FixtureError("CONTAINER_DISAPPEARED", "MySQL container disappeared after readiness")
        validate_container_inspect(
            observed,
            name=self.names["container"],
            image_id=self.receipt["plan"]["image"]["imageId"],
            network_name=self.names["network"],
            labels=self.labels,
            expected_id=self.container_id,
            expected_network_id=self.network_id,
            require_network_id=True,
        )
        network = self.ops.inspect_network(self.network_id or "")
        if network is None:
            raise FixtureError("NETWORK_DISAPPEARED", "internal network disappeared before SQL")
        validate_network_inspect(
            network,
            name=self.names["network"],
            labels=self.labels,
            expected_id=self.network_id,
            expected_container_id=self.container_id,
            expected_container_name=self.names["container"],
        )
        self.receipt["runtime"]["facts"].update(
            {
                "actualImageId": observed.get("imageId"),
                "containerId": self.container_id,
                "containerOwnerLabels": selected_owner_labels(observed, self.labels),
                "networkId": self.network_id,
                "networkOwnerLabels": selected_owner_labels(network, self.labels),
                "networkInternal": network.get("internal"),
                "finalDaemonAuthenticated": True,
            }
        )
        self.persist()

    def _mysql(
        self,
        label: str,
        sql: str,
        *,
        principal: bool = False,
        check: bool = True,
        timeout: int = 20,
    ) -> CommandResult:
        assert self.docker is not None and self.container_id is not None
        password = self.password if principal else self.root_password
        if not password:
            raise FixtureError("FIXTURE_CREDENTIAL_MISSING", "task fixture credential is unavailable")
        args = ["exec", "--env", "MYSQL_PWD"]
        args.extend(["-i", self.container_id, "mysql", "--batch", "--skip-column-names"])
        if principal:
            args.extend(["--protocol=tcp", "--host=127.0.0.1", f"--user={self.names['principal']}", f"--database={self.names['schema']}"])
        else:
            args.extend(["--protocol=tcp", "--host=127.0.0.1", "--user=root"])
        result = self.docker.call(
            label,
            args,
            timeout=timeout,
            input_text=sql,
            check=check,
            redactions=(self.root_password or "", self.password or ""),
            extra_env={"MYSQL_PWD": password},
            record_output=True,
        )
        return result

    def _run_schema_probe(self) -> None:
        schema = self.names["schema"]
        principal = self.names["principal"]
        table = self.names["probeTable"]
        password = uuid.uuid4().hex + uuid.uuid4().hex
        self.password = password
        assert self.docker is not None
        self.docker.add_redaction(password)
        self.receipt["runtime"]["status"] = "RUNNING"
        self.receipt["runtime"]["credentialScope"] = "task-generated memory-only; never serialized"
        self.receipt["runtime"]["facts"]["schemaName"] = schema
        self.receipt["runtime"]["facts"]["principalName"] = principal
        self.persist()
        create_sql = (
            f"CREATE DATABASE `{schema}` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;\n"
            f"CREATE USER '{principal}'@'127.0.0.1' IDENTIFIED BY '{password}';\n"
            f"GRANT CREATE, INSERT, SELECT, UPDATE, DELETE, DROP ON `{schema}`.* TO '{principal}'@'127.0.0.1';\n"
            "SELECT VERSION();\n"
            "SELECT @@transaction_isolation;\n"
            "SELECT @@sql_mode;\n"
            "SELECT @@lower_case_table_names;\n"
            "SELECT @@log_bin_trust_function_creators;\n"
        )
        root_facts = self._mysql("create task schema and inspect MySQL settings", create_sql)
        fact_lines = root_facts.stdout.strip().splitlines()
        if len(fact_lines) < 5:
            raise FixtureError("MYSQL_FACTS_INVALID", "MySQL version/settings output was incomplete")
        settings = {
            "serverVersion": fact_lines[-5],
            "transactionIsolation": fact_lines[-4],
            "sqlMode": fact_lines[-3],
            "lowerCaseTableNames": fact_lines[-2],
            "logBinTrustFunctionCreators": fact_lines[-1],
        }
        self.receipt["runtime"]["facts"]["serverSettings"] = settings

        grant_output = self._mysql(
            "inspect exact fixture principal grants",
            f"SHOW GRANTS FOR '{principal}'@'127.0.0.1';\n",
        )
        grant_check = parse_grant_scope_lines(grant_output.stdout.strip().splitlines(), schema=schema, principal=principal)
        self.receipt["runtime"]["facts"]["grants"] = grant_check

        deny_schema = self.names["denySchema"]
        denied = self._mysql(
            "verify task principal cannot create an unowned schema",
            f"CREATE DATABASE `{deny_schema}`;\n",
            principal=True,
            check=False,
        )
        if denied.exit_code == 0:
            self._mysql("remove unexpectedly created denial-probe schema", f"DROP DATABASE `{deny_schema}`;\n")
            raise FixtureError("UNEXPECTED_RESTRICTED_PRIVILEGE", "task principal created a schema outside its grant")
        if denied.failure_code or denied.exit_code is None or not expected_privilege_denial(
            denied.stderr, principal=principal, schema=deny_schema
        ):
            raise FixtureError("RESTRICTED_DENIAL_UNVERIFIED", "task principal denial probe did not return the exact expected privilege error")
        self.receipt["runtime"]["facts"]["restrictedCreateDenial"] = {
            "outcome": "ACCESS_DENIED",
            "mysqlErrorCode": 1044,
            "targetSchema": deny_schema,
        }

        current_identity = self._mysql(
            "verify task principal identity",
            "SELECT CURRENT_USER(), DATABASE();\n",
            principal=True,
        )
        expected_identity = f"{principal}@127.0.0.1\t{schema}"
        if current_identity.stdout.strip().strip("'`") != expected_identity:
            raise FixtureError("PRINCIPAL_IDENTITY_MISMATCH", "fixture principal connected with an unexpected identity/schema")
        own_grants = self._mysql("verify task principal effective grants", "SHOW GRANTS;\n", principal=True)
        parse_grant_scope_lines(own_grants.stdout.strip().splitlines(), schema=schema, principal=principal)

        ddl = f"CREATE TABLE `{table}` (id INT NOT NULL PRIMARY KEY, payload VARCHAR(32) NOT NULL) ENGINE=InnoDB;\n"
        self._mysql("create fixture probe table", ddl, principal=True)
        ddl_out = self._mysql("capture fixture probe DDL", f"SHOW CREATE TABLE `{table}`;\n", principal=True)
        ddl_parts = ddl_out.stdout.strip().split("\t", 1)
        if len(ddl_parts) != 2 or ddl_parts[0] != table:
            raise FixtureError("PROBE_DDL_INVALID", "probe table DDL response was invalid")
        ddl_text = ddl_parts[1]
        ddl_sha = hashlib.sha256(ddl_text.encode("utf-8")).hexdigest()
        insert_out = self._mysql(
            "insert one fixture probe row",
            f"INSERT INTO `{table}` (id, payload) VALUES (1, 'before'); SELECT ROW_COUNT();\n",
            principal=True,
        )
        update_out = self._mysql(
            "update one fixture probe row",
            f"UPDATE `{table}` SET payload='after' WHERE id=1; SELECT ROW_COUNT();\n",
            principal=True,
        )
        select_out = self._mysql("select fixture probe row", f"SELECT id, payload FROM `{table}`;\n", principal=True)
        delete_out = self._mysql(
            "delete one fixture probe row",
            f"DELETE FROM `{table}` WHERE id=1; SELECT ROW_COUNT();\n",
            principal=True,
        )
        if insert_out.stdout.strip().splitlines()[-1:] != ["1"] or update_out.stdout.strip().splitlines()[-1:] != ["1"] or delete_out.stdout.strip().splitlines()[-1:] != ["1"]:
            raise FixtureError("PROBE_ROW_COUNT_INVALID", "fixture probe DML did not affect exactly one row")
        if select_out.stdout.strip() != "1\tafter":
            raise FixtureError("PROBE_ROW_INVALID", "fixture probe row readback did not match")
        self._mysql("drop fixture probe table", f"DROP TABLE `{table}`;\n", principal=True)
        self._mysql("drop task fixture schema", f"DROP DATABASE `{schema}`;\n")
        schema_count = self._mysql(
            "verify task schema absence",
            f"SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME='{schema}';\n",
        )
        if schema_count.stdout.strip() != "0":
            raise FixtureError("SCHEMA_DROP_UNVERIFIED", "task schema remained after drop")
        self.receipt["runtime"]["facts"].update(
            {
                "probeTable": table,
                "probeDdlSha256": ddl_sha,
                "probeDdl": ddl_text,
                "affectedRows": {"insert": 1, "update": 1, "delete": 1},
                "selectedRow": "1\tafter",
                "probeTableDropped": True,
                "taskSchemaAbsent": True,
            }
        )
        self.receipt["runtime"]["status"] = "SUCCEEDED"
        self.persist()

    def _cleanup(self) -> None:
        unresolved = list(self.unresolved_create_resources)
        if not self.network_id and not self.container_id and not unresolved:
            self.receipt["cleanup"] = {
                "status": "NOT_REQUIRED",
                "containerRemoved": False,
                "networkRemoved": False,
                "absenceVerified": True,
            }
            return
        if (self.network_id or self.container_id) and self.docker is None and self.ops is None:
            self.receipt["cleanup"] = {
                "status": "FAILED_CLEANUP",
                "containerRemoved": False,
                "networkRemoved": False,
                "absenceVerified": False,
                "failureCode": "DOCKER_CLIENT_MISSING",
                "unresolvedResources": [
                    *unresolved,
                    *([{"resource": "container", "name": self.names["container"], "id": self.container_id}] if self.container_id else []),
                    *([{"resource": "network", "name": self.names["network"], "id": self.network_id}] if self.network_id else []),
                ],
            }
            return
        if self.ops is None and self.docker is not None:
            # A network may exist while container creation never succeeded.
            self.ops = DockerResourceOps(
                self.docker,
                expected_network_name=self.names["network"],
                network_labels=self.labels,
                container_name=self.names["container"],
                image_id=self.receipt.get("plan", {}).get("image", {}).get("imageId", ""),
                container_labels=self.labels,
            )
        if self.ops is None:
            cleanup = {
                "status": "VERIFIED_CLEAN",
                "containerRemoved": False,
                "networkRemoved": False,
                "containerStatus": "NOT_REQUIRED",
                "networkStatus": "NOT_REQUIRED",
            }
        else:
            cleanup = cleanup_owned_resources(
                self.ops,
                container_id=self.container_id,
                network_id=self.network_id,
                container_expected={
                    "name": self.names["container"],
                    "labels": self.labels,
                    "imageId": self.receipt.get("plan", {}).get("image", {}).get("imageId"),
                },
                network_expected={"name": self.names["network"], "labels": self.labels},
            )
        if unresolved:
            cleanup.update(
                {
                    "status": "FAILED_CLEANUP",
                    "failureCode": "RESOURCE_CREATE_OUTCOME_UNCERTAIN",
                    "safeMessage": "one or more create outcomes could not be reconciled to a verified owned resource",
                    "unresolvedResources": unresolved,
                }
            )
        cleanup["absenceVerified"] = cleanup["status"] == "VERIFIED_CLEAN"
        self.receipt["cleanup"] = cleanup

    def run(self) -> int:
        try:
            image, _engine = self.prepare()
            self._create_network()
            self._create_container(image)
            self._start_and_wait()
            self._run_schema_probe()
        except FixtureError as exc:
            self.receipt["runtime"]["status"] = "NOT_READY" if exc.code.startswith(("GIT_", "SOURCE_", "DOCKER_", "IMAGE_", "BUILD_ORDER", "OUTPUT_", "NETWORK_NAME_", "CONTAINER_NAME_")) else "FAILED"
            self.receipt["runtime"]["failure"] = {
                "code": exc.code,
                "safeMessage": exc.message,
                "commandLabel": exc.command_label,
                "exitCode": exc.exit_code,
                "safeOutputSha256": exc.output_sha256,
                "timedOut": exc.timed_out,
            }
        except Exception:
            self.receipt["runtime"]["status"] = "FAILED"
            self.receipt["runtime"]["failure"] = {
                "code": "UNEXPECTED_FAILURE",
                "safeMessage": "fixture campaign failed with an unexpected error",
            }
        finally:
            self._cleanup()
            if self.docker:
                self.receipt["commands"] = list(self.docker.commands)
            cleanup_status = self.receipt.get("cleanup", {}).get("status")
            runtime_status = self.receipt.get("runtime", {}).get("status")
            self.receipt["overallStatus"] = (
                "READY_FOR_INDEPENDENT_REVIEW"
                if runtime_status == "SUCCEEDED" and cleanup_status == "VERIFIED_CLEAN"
                else "FAILED_CLEANUP"
                if cleanup_status == "FAILED_CLEANUP"
                else "NOT_READY"
            )
            try:
                self.persist()
            except FixtureError:
                self.receipt["overallStatus"] = "RECEIPT_WRITE_FAILED"
                return 3
        return 0 if self.receipt.get("overallStatus") == "READY_FOR_INDEPENDENT_REVIEW" else 2


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="R083 disposable MySQL fixture prerequisite")
    parser.add_argument("--role", required=True, choices=ROLES, help="separate worker or reviewer campaign")
    return parser


def find_project_root(script_path: Path) -> Path:
    root = script_path.resolve(strict=True).parents[1]
    if not (root / "AGENTS.md").is_file() or not (root / ".git").exists():
        raise FixtureError("PROJECT_ROOT_INVALID", "fixture tool is not inside the project root")
    return root


def main(argv: Sequence[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    role = validate_role(args.role)
    try:
        root = find_project_root(Path(__file__))
        if Path.cwd().resolve(strict=True) != root:
            raise FixtureError("WRONG_WORKING_DIRECTORY", "run this command from the project root")
        task_id = uuid.uuid4().hex
        task_dir = create_task_output_directory(root, role, task_id)
        campaign = Campaign(root, role, task_id, task_dir)
        return campaign.run()
    except FixtureError as exc:
        print(json.dumps({"status": "NOT_READY", "code": exc.code, "message": exc.message}, sort_keys=True), file=sys.stderr)
        return 2
    except OSError:
        print(json.dumps({"status": "NOT_READY", "code": "LOCAL_IO_FAILURE", "message": "local fixture setup failed"}, sort_keys=True), file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
