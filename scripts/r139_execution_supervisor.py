#!/usr/bin/env python3
"""Finite, opt-in R139 execution fixture supervisor.

Import, inspect, and --help are inert. `run` requires a separate, committed,
owner-approved runtime/effect admission. The R139 source-resume admission is
explicitly insufficient. This script never interprets a synthetic AI result
as real governance evidence.
"""
from __future__ import annotations

import argparse
from datetime import datetime, timedelta, timezone
import errno
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import socket
import subprocess
import sys
import time
import urllib.request
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parent.parent
IMAGE = "sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b"
HEX40 = re.compile(r"[0-9a-f]{40}\Z")
HEX64 = re.compile(r"[0-9a-f]{64}\Z")
READY = re.compile(r"R139_BROWSER_READY (http://127\.0\.0\.1:[0-9]{2,5})\Z")
EXPECTED_NATIVE = 8
EXPECTED_BROWSER = 1
EXPECTED_GRANTS = 3
MAX_SECONDS = 20 * 60
MIN_CLEANUP_SECONDS = 180
GRANT_MIN_SECONDS = 60
GRANT_MAX_SECONDS = 24 * 60 * 60
PROMPT_TEXT = "Analyze only fabricated Vietnamese complaint; return the required JSON."
PROMPT_SHA256 = hashlib.sha256(PROMPT_TEXT.encode("utf-8")).hexdigest()
ACTIVE_HARD_DEADLINE: float | None = None
ACTIVE_CLEANUP_DEADLINE: float | None = None
CLEANUP_ERRORS: list[str] = []
COMMAND_SEQUENCE = 0
MINIMAL_ENVIRONMENT_KEYS = {"SYSTEMROOT", "WINDIR", "TEMP", "TMP", "PATH", "USERPROFILE", "LOCALAPPDATA"}


class Hold(RuntimeError):
    pass


def require(ok: bool, code: str) -> None:
    if not ok:
        raise Hold(code)


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read_json(path: Path) -> dict:
    value = json.loads(path.read_text(encoding="utf-8"))
    require(isinstance(value, dict), "JSON_OBJECT_REQUIRED")
    return value


def write_new(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("x", encoding="utf-8") as stream:
        json.dump(value, stream, sort_keys=True, ensure_ascii=True, indent=2)
        stream.write("\n")
        stream.flush()
        os.fsync(stream.fileno())


def append(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("a", encoding="utf-8") as stream:
        stream.write(json.dumps(value, sort_keys=True, ensure_ascii=True) + "\n")
        stream.flush()
        os.fsync(stream.fileno())


def minimal_environment() -> dict[str, str]:
    return {key: value for key, value in os.environ.items() if key.upper() in MINIMAL_ENVIRONMENT_KEYS}


def parse_grant_deadline_utc(value: object, now: datetime | None = None) -> datetime:
    require(isinstance(value, str) and value.endswith("Z"), "GRANT_DEADLINE_UTC_INVALID")
    try:
        deadline = datetime.strptime(value, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)
    except ValueError as exc:
        raise Hold("GRANT_DEADLINE_UTC_INVALID") from exc
    require(deadline.strftime("%Y-%m-%dT%H:%M:%SZ") == value, "GRANT_DEADLINE_UTC_INVALID")
    current = now or datetime.now(timezone.utc)
    require(current.tzinfo is not None and current.utcoffset() == timedelta(0), "GRANT_DEADLINE_REFERENCE_NOT_UTC")
    require(current + timedelta(seconds=GRANT_MIN_SECONDS) <= deadline <= current + timedelta(seconds=GRANT_MAX_SECONDS), "GRANT_DEADLINE_OUT_OF_RANGE")
    return deadline


def project_path(relative: str) -> Path:
    require(isinstance(relative, str) and relative and not Path(relative).is_absolute(), "PATH_INVALID")
    require("\\" not in relative and ".." not in Path(relative).parts, "PATH_INVALID")
    path = (ROOT / relative).resolve()
    require(path.is_relative_to(ROOT.resolve()) and path.is_file(), "PATH_MISSING_OR_OUTSIDE_PROJECT")
    return path


def committed(relative: str) -> Path:
    path = project_path(relative)
    result = subprocess.run(["git", "show", "HEAD:" + relative], cwd=ROOT, capture_output=True, timeout=15)
    require(result.returncode == 0 and result.stdout.replace(b"\r\n", b"\n") == path.read_bytes().replace(b"\r\n", b"\n"), "ADMISSION_NOT_COMMITTED")
    return path


def verify_browser_receipt(value: dict) -> None:
    require(value.get("browserTool") == "supported_CUA_IAB" and value.get("browserSupported") is True, "BROWSER_CAPABILITY_UNPROVEN")
    browser = project_path(value.get("browserReceiptPath", ""))
    expected = value.get("browserReceiptSha256", "")
    require(isinstance(expected, str) and HEX64.fullmatch(expected) and digest(browser) == expected, "BROWSER_RECEIPT_MISMATCH")
    proof = read_json(browser)
    require(proof.get("status") == "PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE" and proof.get("browserId") == "iab" and proof.get("controlObserved") is True and proof.get("captureObserved") is True and proof.get("closeObserved") is True and proof.get("ownedTabsFinal") == 0 and proof.get("protocol") in ("http", "https"), "BROWSER_PROOF_CONTENT_INCOMPLETE")


SOURCE_PIN_REQUIRED = {"backend/api/handlers/intervention_execution_browser_test.go", "scripts/r139_execution_supervisor.py", "scripts/r139_vite_config.mjs.template", "scripts/tests/test_r139_execution_supervisor.py", "backend/go.mod", "backend/go.sum", "frontend/package.json", "frontend/package-lock.json", "frontend/vite.config.ts"}


def verify_source_pins(pins: object) -> None:
    require(isinstance(pins, dict) and SOURCE_PIN_REQUIRED.issubset(pins), "SOURCE_PIN_SET_INCOMPLETE")
    for relative_path, expected in pins.items():
        require(isinstance(expected, str) and HEX64.fullmatch(expected) and digest(project_path(relative_path)) == expected, "SOURCE_PIN_MISMATCH")


def fresh_runtime_admission(relative: str, *, expect_fresh_external: bool = True, validate_live_deadline: bool = True) -> tuple[dict, Path]:
    # Only a new immutable R139 effect admission can enable `run`.
    require(bool(re.fullmatch(r"CVF_SESSION/authority/CCMAI-RUNTIME-139-[A-Za-z0-9_-]+-ADMISSION\.json", relative)), "RUNTIME_ADMISSION_PATH_INVALID")
    path = committed(relative)
    value = read_json(path)
    require(value.get("trancheId") == "CCMAI-RUNTIME-139" and value.get("kind") == "OWNER_APPROVED_RUNTIME_EFFECT_ADMISSION", "RUNTIME_ADMISSION_KIND_INVALID")
    require(value.get("ownerApproved") is True and value.get("sourceOnly") is not True, "OWNER_EFFECT_APPROVAL_MISSING")
    require(value.get("parentAuthorityCommit") == "642f5bd026dfe31e6dda6441dd25704eb3ececa3", "PARENT_SEED_MISMATCH")
    require(value.get("sourceResumeAdmissionCommit") == "6fc951a84c54cd31f20f2404d6dc8957d97bc339", "SOURCE_RESUME_BINDING_MISMATCH")
    source = value.get("sourceCommit")
    require(isinstance(source, str) and HEX40.fullmatch(source), "SOURCE_COMMIT_INVALID")
    ancestor = subprocess.run(["git", "merge-base", "--is-ancestor", source, "HEAD"], cwd=ROOT, capture_output=True, timeout=10)
    require(ancestor.returncode == 0, "SOURCE_COMMIT_NOT_ANCESTOR")
    dirty = subprocess.run(["git", "status", "--porcelain", "--untracked-files=all"], cwd=ROOT, capture_output=True, timeout=10)
    require(dirty.returncode == 0 and not dirty.stdout.strip(), "SOURCE_WORKTREE_DIRTY")
    require(value.get("nativeExecutePOST") == EXPECTED_NATIVE and value.get("browserExecutePOST") == EXPECTED_BROWSER and value.get("grantCap") == EXPECTED_GRANTS and value.get("providerCap") == 1 and value.get("automaticRetry") == 0, "EFFECT_CAPS_INVALID")
    verify_browser_receipt(value)
    require(value.get("image") == IMAGE, "IMAGE_MISMATCH")
    require(value.get("fixtureName", "").startswith("ccmai-r139-"), "FIXTURE_NAME_INVALID")
    require(value.get("database", "").startswith("ccma_r139_"), "DATABASE_NAME_INVALID")
    require(value.get("dockerHost") == "npipe:////./pipe/dockerDesktopLinuxEngine", "DOCKER_HOST_INVALID")
    require(value.get("deadlineSeconds") == MAX_SECONDS, "LIFECYCLE_LIMIT_INVALID")
    verify_source_pins(value.get("sourceSha256"))
    binary = project_path(value.get("goTestBinaryPath", ""))
    require(digest(binary) == value.get("goTestBinarySha256"), "GO_BINARY_PIN_MISMATCH")
    for field in ("dockerBinary", "nodeBinary", "powershellBinary"):
        binary_path = Path(value.get(field, ""))
        require(binary_path.is_absolute() and binary_path.is_file() and digest(binary_path) == value.get(field + "Sha256"), field.upper() + "_PIN_MISMATCH")
    require(Path(value.get("runtimeRoot", "")).resolve().is_relative_to((ROOT / ".cvf/runtime").resolve()), "RUNTIME_ROOT_OUTSIDE_IGNORED_AREA")
    external = ROOT.parent / ".ccmai_local_evidence" / "r139_execution_proof_20261021" / "fixture"
    require(value.get("externalEvidenceDir") == str(external), "EXTERNAL_EVIDENCE_PATH_INVALID")
    if expect_fresh_external:
        require(not external.exists(), "EXTERNAL_EVIDENCE_NOT_FRESH")
    else:
        require(external.is_dir() and external.resolve() == external, "EXTERNAL_EVIDENCE_CHANGED")
    require(value.get("keyCustodian") and value.get("budgetAuthorized") is True and value.get("syntheticIdentityApproved") is True, "PROVIDER_PREREQUISITE_MISSING")
    require(value.get("provider") == "alibaba_maas" and value.get("model") == "qwen3.7-flash" and value.get("completionTokens") == 768 and value.get("maxRequestBytes") == 6000 and value.get("thinking") is False, "PROVIDER_LIMITS_INVALID")
    require(isinstance(value.get("grantDeadlineUTC"), str), "GRANT_DEADLINE_UTC_MISSING")
    if validate_live_deadline:
        parse_grant_deadline_utc(value.get("grantDeadlineUTC"))
    budget = value.get("approvedBudget")
    require(isinstance(budget, dict) and isinstance(budget.get("amount"), (int, float)) and budget["amount"] > 0 and isinstance(budget.get("currency"), str) and len(budget["currency"]) == 3 and value.get("ownerApprovalReference"), "OWNER_BUDGET_APPROVAL_MISSING")
    namespace = value.get("syntheticNamespace")
    require(isinstance(namespace, str) and re.fullmatch(r"ccma_r139_[a-z0-9_-]{1,64}", namespace) is not None and value.get("issuerRole") == "owner" and value.get("actorRole") == "writer" and value.get("configRef") == "r139-owned-config" and value.get("promptVersion") == "r139-synthetic-v1" and value.get("promptSha256") == PROMPT_SHA256 and value.get("grantDeadlineUTC"), "SYNTHETIC_ISSUER_BINDING_MISSING")
    endpoint = value.get("endpoint", "")
    require(isinstance(endpoint, str) and hashlib.sha256(endpoint.encode("utf-8")).hexdigest() == "b22a15b6e8aa4113c2816acae6935a84d052a2d7b81db734ccdc4f5be81b4f8f", "PROVIDER_ENDPOINT_MISMATCH")
    require(isinstance(value.get("vitePort"), int) and 1024 <= value["vitePort"] <= 65535, "VITE_PORT_INVALID")
    vite_cli = project_path("frontend/node_modules/vite/bin/vite.js")
    require(digest(vite_cli) == value.get("viteCliSha256"), "VITE_CLI_PIN_MISMATCH")
    key_path = Path(value.get("keyPath", ""))
    require(key_path.is_absolute() and key_path.is_file() and not key_path.resolve().is_relative_to(ROOT.resolve()) and key_path.stat().st_size <= 8192, "KEY_CUSTODY_PATH_INVALID")
    for tree in ("backend", "frontend"):
        original = subprocess.run(["git", "rev-parse", source + ":" + tree], cwd=ROOT, capture_output=True, text=True, timeout=10)
        current = subprocess.run(["git", "rev-parse", "HEAD:" + tree], cwd=ROOT, capture_output=True, text=True, timeout=10)
        require(original.returncode == 0 and current.returncode == 0 and original.stdout == current.stdout and original.stdout.strip() == value.get(tree + "Tree"), "PRODUCT_TREE_CHANGED")
    return value, path


def docker(admission: dict, runtime: Path, *args: str, timeout: int = 20) -> str:
    global COMMAND_SEQUENCE
    cleanup = ACTIVE_CLEANUP_DEADLINE is not None
    deadline = ACTIVE_CLEANUP_DEADLINE if cleanup else ACTIVE_HARD_DEADLINE
    if deadline is not None:
        remaining = deadline - time.monotonic()
        require(remaining > 0.05, "CLEANUP_DEADLINE_EXPIRED" if cleanup else "GLOBAL_LIFECYCLE_DEADLINE")
        timeout = min(timeout, max(0.05, remaining))
    binary = Path(admission["dockerBinary"])
    require(digest(binary) == admission["dockerBinarySha256"], "DOCKER_BINARY_CHANGED")
    config = runtime / "docker_config"
    if not config.exists():
        try:
            config.mkdir(parents=True, exist_ok=True)
            if not (config / "config.json").exists():
                write_new(config / "config.json", {})
        except Exception as exc:
            if not cleanup:
                raise
            CLEANUP_ERRORS.append("docker_config:" + type(exc).__name__)
    argv = [str(binary), "--host", admission["dockerHost"], "--config", str(config), *args]
    COMMAND_SEQUENCE += 1
    number = COMMAND_SEQUENCE
    command_dir = runtime / ("cleanup_commands" if cleanup else "commands")
    try:
        write_new(command_dir / f"{number:04d}.json", {"argv": argv, "at": time.time(), "timeout": timeout, "cleanup": cleanup})
    except Exception as exc:
        if not cleanup:
            raise
        CLEANUP_ERRORS.append("journal_intent:" + type(exc).__name__)
    try:
        result = subprocess.run(argv, cwd=ROOT, env=minimal_environment(), capture_output=True, timeout=timeout)
    except subprocess.TimeoutExpired:
        if cleanup:
            CLEANUP_ERRORS.append("docker_timeout")
            raise Hold("DOCKER_CLEANUP_TIMEOUT") from None
        append(runtime / "errors.jsonl", {"command": number, "code": "DOCKER_TIMEOUT_NO_RETRY"})
        raise Hold("DOCKER_TIMEOUT_NO_RETRY") from None
    try:
        (command_dir / f"{number:04d}.stdout").write_bytes(result.stdout)
        (command_dir / f"{number:04d}.stderr").write_bytes(result.stderr)
        write_new(command_dir / f"{number:04d}.exit.json", {"code": result.returncode})
    except Exception as exc:
        if not cleanup:
            raise
        CLEANUP_ERRORS.append("journal_result:" + type(exc).__name__)
    if cleanup and result.returncode != 0:
        CLEANUP_ERRORS.append("docker_exit:" + str(result.returncode))
    require(result.returncode == 0, "DOCKER_COMMAND_FAILED")
    return result.stdout.decode("utf-8", errors="strict").strip()


def inventory(admission: dict, runtime: Path) -> dict:
    def rows(*args: str) -> list[dict]:
        raw = docker(admission, runtime, *args)
        return sorted((json.loads(line) for line in raw.splitlines() if line), key=lambda row: json.dumps(row, sort_keys=True))
    return {"containers": rows("ps", "-a", "--no-trunc", "--format", "{{json .}}"), "networks": rows("network", "ls", "--no-trunc", "--format", "{{json .}}"), "volumes": rows("volume", "ls", "--format", "{{json .}}")}


def excluded_owned(observed: dict, owner: dict) -> dict:
    return {"containers": [row for row in observed["containers"] if row.get("ID") != owner.get("containerId")], "networks": [row for row in observed["networks"] if row.get("ID") != owner.get("networkId")], "volumes": [row for row in observed["volumes"] if row.get("Name") != owner.get("volumeName")]}


def verify_owned_resources(admission: dict, runtime: Path, owner: dict) -> dict:
    name, nonce = owner["fixtureName"], owner["nonce"]
    found = {}
    if owner.get("networkId"):
        rows = json.loads(docker(admission, runtime, "network", "inspect", owner["networkId"]))
        require(len(rows) == 1 and rows[0].get("Id") == owner["networkId"] and rows[0].get("Name") == name and rows[0].get("Labels", {}).get("ccmai.owner") == nonce and rows[0].get("Labels", {}).get("ccmai.tranche") == "CCMAI-RUNTIME-139", "NETWORK_OWNERSHIP_UNPROVEN")
        found["networkId"] = owner["networkId"]
    if owner.get("volumeName"):
        rows = json.loads(docker(admission, runtime, "volume", "inspect", owner["volumeName"]))
        require(len(rows) == 1 and rows[0].get("Name") == name and rows[0].get("Labels", {}).get("ccmai.owner") == nonce and rows[0].get("Labels", {}).get("ccmai.tranche") == "CCMAI-RUNTIME-139", "VOLUME_OWNERSHIP_UNPROVEN")
        found["volumeName"] = owner["volumeName"]
    return found


def owned_container(admission: dict, runtime: Path, owner: dict) -> dict:
    raw = docker(admission, runtime, "inspect", owner["containerId"])
    rows = json.loads(raw)
    require(isinstance(rows, list) and len(rows) == 1, "OWNED_CONTAINER_INSPECT_INVALID")
    row = rows[0]
    labels = row.get("Config", {}).get("Labels", {})
    require(row.get("Id") == owner["containerId"] and row.get("Name") == "/" + admission["fixtureName"] and labels.get("ccmai.tranche") == "CCMAI-RUNTIME-139" and labels.get("ccmai.owner") == owner["nonce"], "CONTAINER_OWNERSHIP_UNPROVEN")
    require(row.get("Image") == IMAGE, "CONTAINER_IMAGE_CHANGED")
    mounts = row.get("Mounts", [])
    require(any(m.get("Type") == "volume" and m.get("Name") == owner.get("volumeName") and m.get("Destination") == "/var/lib/mysql" for m in mounts), "CONTAINER_MOUNT_MISMATCH")
    require(row.get("HostConfig", {}).get("NetworkMode") == owner.get("fixtureName"), "CONTAINER_NETWORK_MISMATCH")
    return row


def recover_partial_owner(admission: dict, runtime: Path, owner: dict) -> dict:
    # The named resources are inspected, never inferred solely from names.
    if not owner.get("nonce"):
        return owner
    name = owner["fixtureName"]
    for kind, key, id_field in (("network", "networkId", "Id"), ("volume", "volumeName", "Name"), ("container", "containerId", "Id")):
        if owner.get(key):
            continue
        command = ("inspect", name) if kind == "container" else (kind, "inspect", name)
        try:
            rows = json.loads(docker(admission, runtime, *command))
        except Exception as exc:
            if ACTIVE_CLEANUP_DEADLINE is not None:
                CLEANUP_ERRORS.append("owner_recovery:" + type(exc).__name__)
            continue
        if not isinstance(rows, list) or len(rows) != 1:
            continue
        row = rows[0]
        labels = row.get("Config", {}).get("Labels", {}) if kind == "container" else row.get("Labels", {})
        if labels.get("ccmai.owner") == owner["nonce"] and labels.get("ccmai.tranche") == "CCMAI-RUNTIME-139" and (row.get("Name") in (name, "/" + name)):
            owner[key] = row.get(id_field)
    return owner


def start_fixture(admission: dict, runtime: Path, external: Path, owner: dict) -> dict:
    before = inventory(admission, runtime)
    name = admission["fixtureName"]
    require(not any(row.get("Names") == name for row in before["containers"]), "CONTAINER_NAME_EXISTS")
    require(not any(row.get("Name") == name for row in before["networks"] + before["volumes"]), "RESOURCE_NAME_EXISTS")
    nonce = secrets.token_hex(16)
    owner.update({"nonce": nonce, "protectedBefore": before, "fixtureName": name, "database": admission["database"], "fixturePassword": "r139-synthetic-" + nonce})
    write_new(runtime / "fixture_intent.json", owner)
    labels = ["--label", "ccmai.tranche=CCMAI-RUNTIME-139", "--label", "ccmai.owner=" + nonce]
    owner["networkId"] = docker(admission, runtime, "network", "create", "--driver", "bridge", *labels, name)
    require(bool(HEX64.fullmatch(owner["networkId"])), "NETWORK_ID_INVALID")
    write_new(runtime / "network_created.json", owner)
    require(docker(admission, runtime, "volume", "create", *labels, name) == name, "VOLUME_NAME_MISMATCH")
    owner["volumeName"] = name
    write_new(runtime / "volume_created.json", owner)
    owner["containerId"] = docker(admission, runtime, "create", "--pull", "never", "--name", name, *labels, "--network", name, "--mount", "type=volume,src=" + name + ",dst=/var/lib/mysql", "-p", "127.0.0.1::3306", "-e", "MYSQL_ROOT_PASSWORD=" + owner["fixturePassword"], "-e", "MYSQL_DATABASE=" + admission["database"], IMAGE)
    require(bool(HEX64.fullmatch(owner["containerId"])), "CONTAINER_ID_INVALID")
    write_new(runtime / "container_created.json", owner)
    verify_owned_resources(admission, runtime, owner)
    owned_container(admission, runtime, owner)
    docker(admission, runtime, "start", owner["containerId"], timeout=30)
    row = owned_container(admission, runtime, owner)
    ports = row.get("NetworkSettings", {}).get("Ports", {}).get("3306/tcp") or []
    require(len(ports) == 1 and ports[0].get("HostIp") == "127.0.0.1" and str(ports[0].get("HostPort", "")).isdigit(), "OWNED_LOOPBACK_PORT_INVALID")
    owner["port"] = int(ports[0]["HostPort"])
    require(1024 <= owner["port"] <= 65535, "OWNED_PORT_INVALID")
    # A fresh server can publish its port before MySQL accepts statements.
    # Bound readiness to six owned container probes; no ambient SQL endpoint.
    for attempt in range(6):
        try:
            result = docker(admission, runtime, "exec", "-e", "MYSQL_PWD=" + owner["fixturePassword"], owner["containerId"], "mysql", "-uroot", "-D", admission["database"], "-N", "-B", "-e", "SELECT 1", timeout=8)
            if result == "1":
                break
        except Hold:
            if attempt == 5:
                raise Hold("MYSQL_READY_NOT_PROVEN") from None
            time.sleep(2)
    else:
        raise Hold("MYSQL_READY_NOT_PROVEN")
    sql = "CREATE TABLE ccma_r139_fixture_owner (nonce CHAR(32) NOT NULL); INSERT INTO ccma_r139_fixture_owner (nonce) VALUES ('" + nonce + "');"
    docker(admission, runtime, "exec", "-e", "MYSQL_PWD=" + owner["fixturePassword"], owner["containerId"], "mysql", "-uroot", "-D", admission["database"], "-e", sql, timeout=8)
    write_new(external / "fixture_identity.json", {"trancheId": "CCMAI-RUNTIME-139", "host": "127.0.0.1", "port": owner["port"], "database": admission["database"], "name": name, "nonce": nonce})
    write_new(runtime / "fixture_started.json", owner)
    return owner


def launch(admission: dict, runtime: Path, name: str, argv: list[str], env: dict, children: dict) -> subprocess.Popen:
    cwd = ROOT / "backend/api/handlers" if name == "go" else ROOT
    write_new(runtime / f"{name}_intent.json", {"argv": argv, "cwd": str(cwd), "at": time.time()})
    stream = (runtime / f"{name}.log").open("xb")
    try:
        process = subprocess.Popen(argv, cwd=cwd, env=env, stdout=stream, stderr=subprocess.STDOUT, creationflags=getattr(subprocess, "CREATE_NO_WINDOW", 0))
        children[name] = process  # publish ownership before fallible stream close/receipts
    finally:
        stream.close()
    try:
        identity = process_creation_identity(process)
        write_new(runtime / f"{name}_process.json", {"pid": process.pid, "creationIdentity": identity, "at": time.time()})
    except Exception:
        try:
            stop_child(runtime, name, process)
        finally:
            raise
    return process


def process_creation_identity(process: subprocess.Popen) -> str:
    if os.name == "nt":
        import ctypes
        from ctypes import wintypes
        kernel = ctypes.WinDLL("kernel32", use_last_error=True)
        creation = wintypes.FILETIME()
        exit_time = wintypes.FILETIME()
        get_times = kernel.GetProcessTimes
        get_times.argtypes = [wintypes.HANDLE, ctypes.POINTER(wintypes.FILETIME), ctypes.POINTER(wintypes.FILETIME), ctypes.POINTER(wintypes.FILETIME), ctypes.POINTER(wintypes.FILETIME)]
        get_times.restype = wintypes.BOOL
        require(bool(get_times(wintypes.HANDLE(process._handle), ctypes.byref(creation), ctypes.byref(exit_time), ctypes.byref(wintypes.FILETIME()), ctypes.byref(wintypes.FILETIME()))), "PROCESS_CREATION_ID_UNAVAILABLE")
        return f"windows-filetime-{(creation.dwHighDateTime << 32) | creation.dwLowDateTime:016x}"
    stat = Path(f"/proc/{process.pid}/stat")
    require(stat.is_file(), "PROCESS_CREATION_ID_UNAVAILABLE")
    fields = stat.read_text(encoding="ascii").rsplit(")", 1)[-1].split()
    require(len(fields) > 19, "PROCESS_CREATION_ID_UNAVAILABLE")
    return f"proc-start-ticks-{fields[19]}"


def listener_absent(port: int | None) -> dict:
    require(isinstance(port, int) and 1024 <= port <= 65535, "LISTENER_PORT_UNKNOWN")
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as client:
        client.settimeout(2)
        code = client.connect_ex(("127.0.0.1", port))
    require(code in (errno.ECONNREFUSED, 10061), "LISTENER_ABSENCE_UNPROVEN")
    return {"host": "127.0.0.1", "port": port, "connectEx": code, "listenerAbsent": True}


def stop_child(runtime: Path, name: str, process: subprocess.Popen | None) -> dict:
    if process is None:
        return {"name": name, "started": False}
    if process.poll() is None:
        try:
            append(runtime / "effects.jsonl", {"kind": "terminate", "name": name, "pid": process.pid})
        except OSError:
            pass  # process termination must proceed even when the disk fails
        process.terminate()
        try:
            remaining = None if ACTIVE_CLEANUP_DEADLINE is None else max(0.0, ACTIVE_CLEANUP_DEADLINE - time.monotonic())
            if remaining is not None and remaining <= 0:
                raise subprocess.TimeoutExpired(process.args or name, 0)
            process.wait(timeout=12 if remaining is None else min(12, remaining))
        except subprocess.TimeoutExpired:
            try:
                append(runtime / "effects.jsonl", {"kind": "kill", "name": name, "pid": process.pid})
            except OSError:
                pass
            process.kill()
            remaining = None if ACTIVE_CLEANUP_DEADLINE is None else max(0.0, ACTIVE_CLEANUP_DEADLINE - time.monotonic())
            if remaining is not None and remaining <= 0:
                raise subprocess.TimeoutExpired(process.args or name, 0)
            process.wait(timeout=8 if remaining is None else min(8, remaining))
    else:
        remaining = None if ACTIVE_CLEANUP_DEADLINE is None else max(0.0, ACTIVE_CLEANUP_DEADLINE - time.monotonic())
        if remaining is not None and remaining <= 0:
            raise Hold("CHILD_WAIT_CLEANUP_DEADLINE")
        process.wait(timeout=1 if remaining is None else min(1, remaining))
    return {"name": name, "pid": process.pid, "exitCode": process.returncode, "waitObserved": True}


def stop_fixture(admission: dict, runtime: Path, owner: dict) -> dict:
    ownership_error: Exception | None = None
    try:
        verify_owned_resources(admission, runtime, owner)
    except Exception as exc:
        ownership_error = exc
    # Stop only the exact container whose label, image, mount and network are
    # independently inspected. A volume/network inspection failure cannot
    # leave a verified live container running.
    before = owned_container(admission, runtime, owner)
    if before.get("State", {}).get("Running") is True:
        docker(admission, runtime, "stop", "--time", "10", owner["containerId"], timeout=30)
    after = owned_container(admission, runtime, owner)
    state = after.get("State", {})
    require(state.get("Running") is False and state.get("Pid") == 0, "CONTAINER_NOT_TERMINAL")
    require(after.get("NetworkSettings", {}).get("Ports", {}).get("3306/tcp") in (None, []), "CONTAINER_PORT_STILL_PUBLISHED")
    protected = excluded_owned(inventory(admission, runtime), owner)
    require(protected == owner["protectedBefore"], "PROTECTED_INVENTORY_CHANGED")
    if ownership_error is not None:
        raise Hold("RELATED_RESOURCE_OWNERSHIP_UNPROVEN") from ownership_error
    return {"containerId": owner["containerId"], "running": False, "pid": 0, "protectedEqual": True}


def wait_go_ready(process: subprocess.Popen, log: Path, deadline: float) -> str:
    while time.monotonic() < deadline:
        require(process.poll() is None, "GO_CHILD_EXITED_BEFORE_READY")
        if log.exists():
            for line in log.read_text(encoding="utf-8", errors="replace").splitlines():
                match = READY.fullmatch(line)
                if match:
                    return match.group(1)
        time.sleep(0.2)
    raise Hold("GO_CHILD_READY_TIMEOUT")


def _validate_read_rows(rows: object, expected_names: set[str], *, ready_name: str | None = None) -> list[dict]:
    require(isinstance(rows, list) and all(isinstance(row, dict) for row in rows), "NATIVE_READ_ROWS_INVALID")
    names = [row.get("name") for row in rows]
    require(len(names) == len(set(names)) and set(names) == expected_names, "NATIVE_READ_MATRIX_INCOMPLETE")
    for row in rows:
        require(isinstance(row.get("status"), int) and isinstance(row.get("bodySha256"), str) and HEX64.fullmatch(row["bodySha256"]), "NATIVE_READ_RECEIPT_INVALID")
        before, after = row.get("durableBefore"), row.get("durableAfter")
        require(isinstance(before, str) and HEX64.fullmatch(before) and before == after, "NATIVE_READ_MUTATED_DURABLE_STATE")
        is_ready = row.get("name") == ready_name
        require(row.get("noStore") is True and row.get("deniedHandleAbsent") is (not is_ready), "NATIVE_READ_CACHE_OR_HANDLE_GUARD_FAILED")
        require(row.get("grantHandlePresent") is is_ready, "NATIVE_READ_GRANT_HANDLE_MISMATCH")
    return rows


def validate_native_evidence(runtime: Path, *, require_published: bool = False) -> dict:
    negative = read_json(runtime / "native_rejections.json")
    race = read_json(runtime / "native_precall_denied.json")
    cases = negative.get("cases")
    require(isinstance(cases, list) and all(isinstance(row, dict) for row in cases), "NATIVE_SCENARIO_SET_INCOMPLETE")
    names = [row.get("name") for row in cases]
    require(names == ["malformed-json", "unknown-field", "missing-version", "stale-version", "foreign-actor-grant", "expired-grant", "foreign-case"], "NATIVE_SCENARIO_SET_INCOMPLETE")
    require(negative.get("slotsUsed") == 7 and [row.get("slot") for row in cases] == list(range(1, 8)), "NATIVE_SLOT_SEQUENCE_INVALID")
    require([row.get("status") for row in cases] == [400, 400, 400, 409, 409, 409, 404], "NATIVE_REJECTION_STATUS_INVALID")
    expected_errors = ["invalid_request", "invalid_request", "invalid_request", "case_conflict", "case_conflict", "case_conflict", "not_found"]
    require([row.get("error") for row in cases] == expected_errors, "NATIVE_REJECTION_REASON_INVALID")
    for row in cases:
        require(isinstance(row.get("bodySha256"), str) and HEX64.fullmatch(row["bodySha256"]) and isinstance(row.get("durableBefore"), str) and HEX64.fullmatch(row["durableBefore"]) and row.get("durableBefore") == row.get("durableAfter"), "NATIVE_REJECTION_NO_WRITE_UNPROVEN")
    require(negative.get("kind") == "SOFTWARE_ONLY_NO_AI_OUTPUT" and negative.get("trapCalls") == 0 and negative.get("nativeExecutePOST") == 7 and negative.get("grantIssuances") == 1 and race.get("kind") == "SOFTWARE_ONLY_NO_AI_OUTPUT", "NATIVE_AI_CLAIM_INVALID")
    require(race.get("slot") == 8 and race.get("status") == 202 and race.get("acceptedState") == "CLAIMED" and race.get("terminal") == "DENIED" and race.get("trapCalls") == 0 and race.get("attempts") == 0 and race.get("budgetConsumed") == 0 and race.get("nativeExecutePOSTBeforeBrowser") == 8 and race.get("grantIssuancesBeforeBrowser") == 2 and isinstance(race.get("executionId"), str) and race.get("executionId") and race.get("acceptedExecutionId") == race.get("executionId") and isinstance(race.get("bodySha256"), str) and HEX64.fullmatch(race["bodySha256"]) and isinstance(race.get("durableBefore"), str) and HEX64.fullmatch(race["durableBefore"]) and isinstance(race.get("durableAfter"), str) and HEX64.fullmatch(race["durableAfter"]) and race["durableBefore"] != race["durableAfter"], "NATIVE_PRECALL_DENIAL_INCOMPLETE")
    negative_reads = read_json(runtime / "native_reads_negative.json")
    pre_positive_reads = read_json(runtime / "native_reads_positive_pre_browser.json")
    positive_reads = read_json(runtime / "native_reads_positive.json") if require_published else pre_positive_reads
    require(negative_reads.get("kind") == "SOFTWARE_ONLY_NO_AI_OUTPUT" and pre_positive_reads.get("kind") == "SOFTWARE_ONLY_NO_AI_OUTPUT" and positive_reads.get("kind") == "SOFTWARE_ONLY_NO_AI_OUTPUT", "NATIVE_READ_EVIDENCE_KIND_INVALID")
    expected_negative = {"grant-absent", "consumed-grant", "config-mismatch", "budget-exhausted", "case-version-changed", "writer-permission-revoked", "source-changed", "foreign-actor-grant", "expired-grant"}
    neg_rows = _validate_read_rows(negative_reads.get("reads"), expected_negative)
    states = {"grant-absent": "AWAITING_OPERATOR_AUTHORITY", "consumed-grant": "AWAITING_OPERATOR_AUTHORITY", "config-mismatch": "AWAITING_OPERATOR_AUTHORITY", "budget-exhausted": "STALE_OR_INELIGIBLE", "case-version-changed": "STALE_OR_INELIGIBLE", "source-changed": "STALE_OR_INELIGIBLE", "foreign-actor-grant": "AWAITING_OPERATOR_AUTHORITY", "expired-grant": "AWAITING_OPERATOR_AUTHORITY"}
    for row in neg_rows:
        name = row["name"]
        if name == "writer-permission-revoked":
            require(row.get("status") == 403 and row.get("error") == "permission_denied", "NATIVE_PERMISSION_READ_DENIAL_INVALID")
        else:
            require(row.get("status") == 200 and row.get("state") == states[name], "NATIVE_READINESS_STATE_INVALID")
    pre_positive_source_rows = pre_positive_reads.get("reads")
    require(isinstance(pre_positive_source_rows, list) and all(isinstance(row, dict) for row in pre_positive_source_rows), "NATIVE_READ_ROWS_INVALID")
    positive_source_rows = positive_reads.get("reads")
    require(isinstance(positive_source_rows, list) and all(isinstance(row, dict) for row in positive_source_rows), "NATIVE_READ_ROWS_INVALID")
    denied_names = {"readiness-denied-reader", "readiness-denied-nojobs"}
    base_positive = {"readiness-foreign-case", "writer-READY", "reader-status-no-row", "process-disabled-status", "process-disabled-readiness", "status-unreadable-503", "readiness-unreadable-503", *denied_names}
    published_positive = {"reader-published-status", "writer-published-status", "reader-history", "reader-open-detail"}
    expected_positive = base_positive | (published_positive if require_published else set())
    require(sum(row.get("name") in denied_names for row in positive_source_rows) == 2, "NATIVE_PERMISSION_READ_MATRIX_INCOMPLETE")
    pos_rows = _validate_read_rows(positive_reads.get("reads"), expected_positive, ready_name="writer-READY")
    if require_published:
        pre_names = [row.get("name") for row in pre_positive_source_rows]
        require(set(pre_names) == base_positive and len(pre_names) == len(set(pre_names)), "NATIVE_PRE_BROWSER_READ_MATRIX_INCOMPLETE")
        full_by_name = {row["name"]: row for row in pos_rows}
        pre_by_name = {row["name"]: row for row in pre_positive_source_rows}
        require(all(full_by_name[name] == row for name, row in pre_by_name.items()), "NATIVE_PRE_BROWSER_READ_RECEIPT_CHANGED")
    by_name = {row["name"]: row for row in pos_rows}
    require(all(by_name[name].get("status") == 403 and by_name[name].get("error") == "permission_denied" for name in denied_names), "NATIVE_PERMISSION_READ_DENIAL_INVALID")
    require(by_name["readiness-foreign-case"].get("status") == 404 and by_name["readiness-foreign-case"].get("error") == "not_found", "NATIVE_FOREIGN_READ_INVALID")
    require(by_name["writer-READY"].get("status") == 200 and by_name["writer-READY"].get("state") == "READY", "NATIVE_READY_READ_INVALID")
    for name in ("reader-status-no-row", "process-disabled-status"):
        require(by_name[name].get("status") == 200 and by_name[name].get("executionPresent") is False and by_name[name].get("phase") == "NOT_STARTED", "NATIVE_EMPTY_STATUS_READ_INVALID")
    require(by_name["process-disabled-readiness"].get("status") == 200 and by_name["process-disabled-readiness"].get("state") == "PROCESS_DISABLED", "NATIVE_DISABLED_READ_INVALID")
    for name, error in (("status-unreadable-503", "status_unavailable"), ("readiness-unreadable-503", "readiness_unavailable")):
        require(by_name[name].get("status") == 503 and by_name[name].get("error") == error, "NATIVE_UNREADABLE_READ_INVALID")
    if require_published:
        require(by_name["reader-published-status"].get("status") == 200 and by_name["reader-published-status"].get("phase") == "PUBLISHED" and by_name["reader-published-status"].get("proposalAvailable") is True and isinstance(by_name["reader-published-status"].get("publishedCaseVersion"), int), "NATIVE_PUBLISHED_READ_INVALID")
        require(by_name["writer-published-status"].get("status") == 200 and by_name["writer-published-status"].get("phase") == "PUBLISHED", "NATIVE_WRITER_PUBLISHED_READ_INVALID")
        require(by_name["reader-history"].get("status") == 200 and by_name["reader-open-detail"].get("status") == 200, "NATIVE_HISTORY_DETAIL_READ_INVALID")
    return {"nativeExecutePOST": race["nativeExecutePOSTBeforeBrowser"], "rejectionSlots": len(cases), "grantIssuancesBeforeBrowser": race["grantIssuancesBeforeBrowser"], "negativeReadCount": len(neg_rows), "positiveReadCount": len(pos_rows), "transportCalls": 0, "kind": "SOFTWARE_ONLY_NO_AI_OUTPUT"}


def validate_vite_ledger(runtime: Path, terminal: dict) -> dict:
    path = runtime / "vite_routes.jsonl"
    require(path.is_file(), "VITE_ROUTE_LEDGER_MISSING")
    rows = [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines() if line]
    attempts = [row for row in rows if row.get("phase") == "attempt"]
    responses = [row for row in rows if row.get("phase") == "response"]
    attempt_ids = [row.get("id") for row in attempts]
    response_ids = [row.get("id") for row in responses]
    require(len(attempts) == len(responses) and len(set(attempt_ids)) == len(attempt_ids) and len(set(response_ids)) == len(response_ids) and set(attempt_ids) == set(response_ids), "VITE_ROUTE_RESPONSE_GAP")
    attempt_by_id = {row["id"]: row for row in attempts}
    require(all(row.get("method") == attempt_by_id[row["id"]].get("method") and row.get("path") == attempt_by_id[row["id"]].get("path") for row in responses), "VITE_ROUTE_RESPONSE_IDENTITY_MISMATCH")
    execute = [row for row in attempts if row.get("method") == "POST" and str(row.get("path", "")).endswith("/execute")]
    auth = [row for row in attempts if row.get("method") == "POST" and "/auth/" in str(row.get("path", ""))]
    require(len(execute) == terminal.get("executePOST") == 1 and execute[0].get("allowed") is True, "BROWSER_EXECUTE_LEDGER_MISMATCH")
    require(len(auth) == terminal.get("authPOST") and len(auth) <= 4, "BROWSER_AUTH_LEDGER_MISMATCH")
    require(all(row.get("allowed") is True for row in attempts), "BLOCKED_ROUTE_ATTEMPTED")
    return {"attempts": len(attempts), "responses": len(responses), "executePOST": len(execute), "authPOST": len(auth)}


def validate_go_ledger(external: Path, terminal: dict) -> dict:
    path = external / "go_routes.jsonl"
    require(path.is_file(), "GO_ROUTE_LEDGER_MISSING")
    rows = [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines() if line]
    attempts = {row.get("id"): row for row in rows if row.get("phase") == "attempt"}
    responses = {row.get("id"): row for row in rows if row.get("phase") == "response"}
    require(len(attempts) + len(responses) == len(rows) and len(attempts) == len(responses) and None not in attempts and attempts.keys() == responses.keys(), "GO_ROUTE_RESPONSE_GAP")
    require(all(responses[key].get("method") == row.get("method") and responses[key].get("path") == row.get("path") and isinstance(responses[key].get("status"), int) for key, row in attempts.items()), "GO_ROUTE_RESPONSE_IDENTITY_MISMATCH")
    require(all(row.get("allowed") is True for row in attempts.values()), "GO_BLOCKED_ROUTE_ATTEMPTED")
    execute = [row for row in attempts.values() if row.get("method") == "POST" and str(row.get("path", "")).endswith("/execute")]
    auth = [row for row in attempts.values() if row.get("method") == "POST" and "/auth/" in str(row.get("path", ""))]
    require(len(execute) == terminal.get("executePOST") == 1 and len(auth) == terminal.get("authPOST") and len(auth) <= 4, "GO_EFFECT_ROUTE_MISMATCH")
    return {"attempts": len(attempts), "responses": len(responses), "executePOST": len(execute), "authPOST": len(auth)}


def validate_go_terminal(external: Path, terminal: dict, admission: dict | None = None, native_validation: dict | None = None) -> dict:
    require(terminal.get("nativeExecutePOST") == 8 and terminal.get("grantIssuances") == 3 and terminal.get("executePOST") == 1 and terminal.get("providerPhysicalCalls") == 1, "GO_EFFECT_ACCOUNTING_INCOMPLETE")
    require(terminal.get("providerHTTPStatus") == 200 and terminal.get("budgetConsumed") == 1 and terminal.get("executionState") == "PUBLISHED" and terminal.get("caseWorkState") == "OPEN", "GO_PUBLICATION_STATE_INCOMPLETE")
    require(all(terminal.get(field) is True for field in ("originalRequestUnchanged", "originalSnapshotUnchanged", "lineageVerified", "readerWriterPublishedStatus", "readerHistoryAndOpenDetail", "executionPresent")), "GO_LINEAGE_OR_READ_INCOMPLETE")
    require(terminal.get("fingerprintError") is False and isinstance(terminal.get("durableFingerprint"), str) and HEX64.fullmatch(terminal["durableFingerprint"]), "GO_DURABLE_FINGERPRINT_INVALID")
    lineage_pairs = (("storedRequestId", "requestId"), ("executionRequestId", "requestId"), ("executionCaseId", "caseId"), ("executionGrantId", "grantId"), ("executionRunId", "jobRunId"), ("attemptRunId", "jobRunId"), ("receiptRunId", "jobRunId"), ("proposalRunId", "jobRunId"), ("executionSnapshotId", "snapshotId"), ("proposalSnapshotId", "snapshotId"), ("eventRequestId", "requestId"), ("eventProposalId", "proposalId"), ("attemptExecutionId", "executionId"))
    for field, target in lineage_pairs:
        require(isinstance(terminal.get(field), str) and terminal.get(field) and terminal[field] == terminal.get(target), "GO_LINEAGE_ID_MISMATCH")
    for field in ("caseId", "requestId", "grantId", "executionId", "jobRunId", "snapshotId", "proposalId", "eventId", "attemptId", "receiptId"):
        require(isinstance(terminal.get(field), str) and terminal[field], "GO_LINEAGE_ID_MISSING")
    require(terminal.get("storedRequestCaseId") == terminal.get("caseId") and terminal.get("proposalRevision") == terminal.get("eventCaseVersion"), "GO_LINEAGE_REVISION_MISMATCH")
    require(isinstance(terminal.get("snapshotDigest"), str) and HEX64.fullmatch(terminal["snapshotDigest"]), "GO_SNAPSHOT_DIGEST_INVALID")
    if admission is not None:
        admitted = {"provider": admission.get("provider"), "model": admission.get("model"), "configRef": admission.get("configRef"), "promptVersion": admission.get("promptVersion"), "promptSha256": admission.get("promptSha256"), "syntheticNamespace": admission.get("syntheticNamespace"), "issuerRole": admission.get("issuerRole"), "actorRole": admission.get("actorRole"), "grantDeadlineUTC": admission.get("grantDeadlineUTC"), "completionTokens": admission.get("completionTokens"), "maxRequestBytes": admission.get("maxRequestBytes"), "thinkingDisabled": admission.get("thinking") is False}
        require(all(terminal.get(field) == expected for field, expected in admitted.items()), "GO_TERMINAL_ADMISSION_MISMATCH")
        require(terminal.get("responseModel") == admission.get("model"), "GO_RESPONSE_MODEL_MISMATCH")
    if native_validation is not None:
        require(terminal.get("nativeExecutePOST") == native_validation.get("nativeExecutePOST") == 8 and terminal.get("nativeNegativeReadCount") == native_validation.get("negativeReadCount") and terminal.get("nativePositiveReadCount") == native_validation.get("positiveReadCount") and terminal.get("grantIssuances") == native_validation.get("grantIssuancesBeforeBrowser") + 1 == 3 and native_validation.get("transportCalls") == 0 and native_validation.get("rejectionSlots") == 7, "GO_NATIVE_MATRIX_COUNTER_MISMATCH")
    wire = read_json(external / "provider_wire_private.json")
    require(wire.get("provider") == "alibaba_maas" and wire.get("model") == "qwen3.7-flash" and wire.get("physicalCalls") == 1 and wire.get("httpStatus") == 200, "GO_PROVIDER_WIRE_INCOMPLETE")
    for field in ("request", "response"):
        raw = wire.get(field)
        require(isinstance(raw, str) and raw and hashlib.sha256(raw.encode("utf-8")).hexdigest() == terminal.get(field + "Sha256"), "GO_PROVIDER_WIRE_HASH_MISMATCH")
    return validate_go_ledger(external, terminal)


def run(relative: str) -> None:
    global ACTIVE_HARD_DEADLINE, ACTIVE_CLEANUP_DEADLINE
    admission, path = fresh_runtime_admission(relative)
    started = time.monotonic()
    hard_deadline = started + MAX_SECONDS
    ACTIVE_HARD_DEADLINE = hard_deadline
    ACTIVE_CLEANUP_DEADLINE = None
    CLEANUP_ERRORS.clear()
    runtime = Path(admission["runtimeRoot"]).resolve()
    external = Path(admission["externalEvidenceDir"])
    require(not runtime.exists(), "R139_RUNTIME_ALREADY_EXISTS")
    require(not external.exists(), "R139_EXTERNAL_EVIDENCE_ALREADY_EXISTS")
    runtime.mkdir(parents=True, exist_ok=False)
    external.mkdir(parents=True, exist_ok=False)
    write_new(runtime / "run_intent.json", {"admissionSha256": digest(path), "sourceCommit": admission["sourceCommit"], "startedAt": time.time(), "deadlineSeconds": MAX_SECONDS})
    owner: dict = {}
    children: dict[str, subprocess.Popen] = {}
    go = vite = None
    api_port: int | None = None
    outcome: dict = {"result": "HOLD", "terminal": False}
    native_validation: dict | None = None
    try:
        image_rows = json.loads(docker(admission, runtime, "image", "inspect", IMAGE))
        require(isinstance(image_rows, list) and len(image_rows) == 1 and image_rows[0].get("Id") == IMAGE, "PINNED_IMAGE_NOT_CACHED")
        key_file = Path(admission["keyPath"])
        key = key_file.read_text(encoding="utf-8").strip()
        require(len(key) >= 16 and "\n" not in key, "KEY_INVALID")
        require(time.monotonic() < hard_deadline - 180, "PREFLIGHT_DEADLINE")
        start_fixture(admission, runtime, external, owner)
        minimal = minimal_environment()
        env = dict(minimal)
        env.update({"R139_RUN": "1", "R139_RUNTIME_ADMISSION": str(path), "R139_RUNTIME_ADMISSION_SHA256": digest(path), "R139_SOURCE_COMMIT": admission["sourceCommit"], "R139_FIXTURE_IDENTITY": str(external / "fixture_identity.json"), "R139_MYSQL_DSN": f'root:{owner["fixturePassword"]}@tcp(127.0.0.1:{owner["port"]})/{admission["database"]}?parseTime=true', "R139_RUNTIME_ROOT": str(runtime), "R139_EVIDENCE_DIR": str(external), "R139_EFFECT_DIR": str(external / "effect_slots"), "R139_ALIBABA_KEY": key, "R139_ALIBABA_ENDPOINT": admission["endpoint"], "R139_GRANT_DEADLINE_UTC": admission["grantDeadlineUTC"], "R139_STOP_FILE": str(runtime / "stop_requested")})
        go = launch(admission, runtime, "go", [str(project_path(admission["goTestBinaryPath"])), "-test.run", "^TestR139IntegratedExecutionProof$", "-test.v"], env, children)
        origin = wait_go_ready(go, runtime / "go.log", min(time.monotonic() + 90, hard_deadline - 150))
        api_port = urlsplit(origin).port
        native_validation = validate_native_evidence(external)
        write_new(runtime / "native_evidence_validation.json", native_validation)
        template = project_path("scripts/r139_vite_config.mjs.template").read_text(encoding="utf-8")
        for token, replacement in {"__R139_FRONTEND_ROOT__": str(ROOT / "frontend"), "__R139_RUNTIME__": str(runtime), "__R139_API_ORIGIN__": origin, "__R139_NONCE__": secrets.token_hex(16)}.items():
            template = template.replace(token, json.dumps(replacement))
        require("__R139_" not in template and "strictPort: true" in template, "VITE_TEMPLATE_INVALID")
        config = runtime / "vite_config.mjs"
        config.write_text(template, encoding="utf-8")
        node = Path(admission["nodeBinary"])
        vite_cli = project_path("frontend/node_modules/vite/bin/vite.js")
        vite = launch(admission, runtime, "vite", [str(node), str(vite_cli), "--config", str(config), "--host", "127.0.0.1", "--port", str(admission["vitePort"]), "--strictPort"], minimal, children)
        health_deadline = min(time.monotonic() + 45, hard_deadline - 130)
        opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
        while time.monotonic() < health_deadline:
            require(vite.poll() is None, "VITE_EXITED_BEFORE_READY")
            try:
                with opener.open(f'http://127.0.0.1:{admission["vitePort"]}/__r139_health', timeout=2) as response:
                    if response.read().decode("utf-8") == json.loads(template.split("const nonce = ", 1)[1].split("\n", 1)[0]):
                        break
            except (OSError, ValueError):
                time.sleep(0.2)
        else:
            raise Hold("VITE_HEALTH_TIMEOUT")
        write_new(runtime / "browser_ready.json", {"apiOrigin": origin, "uiOrigin": f'http://127.0.0.1:{admission["vitePort"]}', "sourceCommit": admission["sourceCommit"], "admissionSha256": digest(path)})
        print(json.dumps({"result": "BROWSER_READY", "apiOrigin": origin, "uiOrigin": f'http://127.0.0.1:{admission["vitePort"]}'}, sort_keys=True), flush=True)
        deadline = hard_deadline - min(120, MIN_CLEANUP_SECONDS)
        while time.monotonic() < deadline:
            require(go.poll() is None and vite.poll() is None, "CHILD_EXIT_DURING_CAMPAIGN")
            if (runtime / "stop_requested").is_file():
                break
            time.sleep(0.25)
        else:
            raise Hold("CAMPAIGN_DEADLINE")
        outcome["result"] = "STOP_REQUESTED"
    except Exception as exc:
        outcome["error"] = type(exc).__name__ + ":" + str(exc)
    finally:
        cleanup_started = time.monotonic()
        cleanup_wall_started = time.time()
        ACTIVE_CLEANUP_DEADLINE = cleanup_started + MIN_CLEANUP_SECONDS
        errors: list[str] = []
        outcome["cleanup"] = {"startedAt": cleanup_wall_started, "deadlineSeconds": MIN_CLEANUP_SECONDS, "campaignDeadlineExpiredAtStart": cleanup_started >= hard_deadline - min(120, MIN_CLEANUP_SECONDS)}
        try:
            write_new(runtime / "stop_requested", {"reason": "supervisor-finally", "at": time.time()})
        except FileExistsError:
            pass
        except Exception as exc:
            errors.append("STOP_SIGNAL_FAILED:" + type(exc).__name__)
        if children.get("go") is not None:
            try:
                remaining = max(0.0, ACTIVE_CLEANUP_DEADLINE - time.monotonic())
                require(remaining > 0, "CLEANUP_DEADLINE_EXPIRED")
                children["go"].wait(timeout=min(30, remaining))
            except subprocess.TimeoutExpired:
                errors.append("GO_FINAL_WAIT_TIMEOUT")
            except Exception as exc:
                errors.append("GO_FINAL_WAIT_FAILED:" + type(exc).__name__)
        for name in ("vite", "go"):
            try:
                outcome[name] = stop_child(runtime, name, children.get(name))
            except Exception as exc:
                errors.append(name.upper() + "_STOP_FAILED:" + type(exc).__name__)
        if outcome.get("go", {}).get("exitCode") != 0 or not outcome.get("go", {}).get("waitObserved"):
            errors.append("ORIGINAL_GO_EXIT_NOT_ZERO_OR_UNOBSERVED")
        if children.get("vite") is not None and not outcome.get("vite", {}).get("waitObserved"):
            errors.append("VITE_EXIT_UNOBSERVED")
        try:
            outcome["apiListener"] = listener_absent(api_port)
        except Exception as exc:
            errors.append("API_LISTENER_CLOSE_UNPROVEN:" + type(exc).__name__)
        if children.get("vite") is not None:
            try:
                outcome["viteListener"] = listener_absent(admission["vitePort"])
            except Exception as exc:
                errors.append("VITE_LISTENER_CLOSE_UNPROVEN:" + type(exc).__name__)
        if children.get("go") is None or not (external / "go_terminal.json").is_file():
            errors.append("GO_FINAL_DURABLE_EVIDENCE_MISSING")
        else:
            try:
                outcome["goTerminal"] = read_json(external / "go_terminal.json")
                require(native_validation is not None or (runtime / "native_evidence_validation.json").is_file(), "GO_NATIVE_VALIDATION_MISSING")
                final_native_validation = validate_native_evidence(external, require_published=True)
                write_new(runtime / "native_evidence_terminal_validation.json", final_native_validation)
                outcome["nativeEvidenceFinal"] = final_native_validation
                outcome["goRouteAccounting"] = validate_go_terminal(external, outcome["goTerminal"], admission, final_native_validation)
                outcome["viteRouteAccounting"] = validate_vite_ledger(runtime, outcome["goTerminal"])
            except Exception as exc:
                errors.append("TERMINAL_OR_ROUTE_VALIDATION_FAILED:" + type(exc).__name__)
        try:
            recover_partial_owner(admission, runtime, owner)
        except Exception as exc:
            errors.append("PARTIAL_OWNER_RECOVERY_FAILED:" + type(exc).__name__)
        if owner.get("containerId"):
            try:
                outcome["fixture"] = stop_fixture(admission, runtime, owner)
            except Exception as exc:
                errors.append("FIXTURE_TERMINAL_FAILED:" + type(exc).__name__)
        elif owner.get("networkId") or owner.get("volumeName"):
            try:
                outcome["partialOwnership"] = verify_owned_resources(admission, runtime, owner)
            except Exception as exc:
                errors.append("PARTIAL_RESOURCE_OWNERSHIP_UNPROVEN:" + type(exc).__name__)
            errors.append("PARTIAL_RESOURCE_PRESERVED_HOLD")
        if owner.get("port"):
            try:
                outcome["dbListener"] = listener_absent(owner["port"])
            except Exception as exc:
                errors.append("DB_LISTENER_CLOSE_UNPROVEN:" + type(exc).__name__)
        try:
            fresh_runtime_admission(relative, expect_fresh_external=False, validate_live_deadline=False)
        except Exception as exc:
            errors.append("SOURCE_RECHECK_FAILED:" + type(exc).__name__)
        errors.extend("CLEANUP_BEST_EFFORT_ERROR:" + issue for issue in CLEANUP_ERRORS)
        cleanup_finished = time.monotonic()
        outcome["cleanup"].update({"finishedAt": time.time(), "elapsedSeconds": round(cleanup_finished - cleanup_started, 3), "completedBeforeDeadline": cleanup_finished <= ACTIVE_CLEANUP_DEADLINE})
        if cleanup_finished > ACTIVE_CLEANUP_DEADLINE:
            errors.append("CLEANUP_DEADLINE_OVERRUN")
        if cleanup_finished > hard_deadline:
            errors.append("LIFECYCLE_DEADLINE_OVERRUN")
        outcome["terminalErrors"] = errors
        outcome["terminal"] = True
        if errors:
            outcome["result"] = "HOLD"
        try:
            write_new(runtime / "terminal.json", outcome)
        except Exception as exc:
            errors.append("TERMINAL_RECEIPT_WRITE_FAILED:" + type(exc).__name__)
            outcome["result"] = "HOLD"
        print(json.dumps(outcome, sort_keys=True), flush=True)
    require(outcome["result"] == "STOP_REQUESTED" and not outcome["terminalErrors"], "R139_TERMINAL_HOLD")


def inspect() -> None:
    print(json.dumps({"tranche": "CCMAI-RUNTIME-139", "sourceOnly": True, "runtimeEffectsAdmittedByThisSource": 0, "browserDiagnosticRemaining": 0}, sort_keys=True))


def main() -> None:
    parser = argparse.ArgumentParser(description="R139 admitted finite runtime supervisor")
    parser.add_argument("command", choices=["inspect", "run"])
    parser.add_argument("--admission", help="Committed fresh R139 runtime/effect admission path")
    args = parser.parse_args()
    if args.command == "inspect":
        inspect()
    else:
        require(bool(args.admission), "RUNTIME_ADMISSION_REQUIRED")
        run(args.admission)


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        print(type(error).__name__ + ":" + str(error), file=sys.stderr)
        raise SystemExit(1)
