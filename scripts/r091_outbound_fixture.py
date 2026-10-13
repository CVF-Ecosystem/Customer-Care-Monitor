#!/usr/bin/env python3
"""Run R091 Go checks against one owned, disposable, loopback-only MySQL fixture."""

from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import re
import secrets
import socket
import subprocess
import sys
import time
from datetime import datetime, timezone
from uuid import uuid4


ROOT = Path(__file__).resolve().parents[1]
RUNTIME = ROOT / ".cvf" / "runtime"
STATE_PATH = RUNTIME / "r091_outbound_fixture_state.json"
PROBE_DIR = ROOT / "docs" / "reviews" / "probes" / "r091_outbound_delivery"
ALLOWED_PROBE_DIRS = {
    "docs/reviews/probes/r091_outbound_delivery",
    "docs/reviews/probes/r091_outbound_delivery_r1",
}
ALLOWED_STATE_FILES = {
    ".cvf/runtime/r091_outbound_fixture_state.json",
    ".cvf/runtime/r091_outbound_fixture_r1_state.json",
}
IMAGE = "mysql:8.0"
LABELS = {
    "org.cvf.tranche": "CCMAI-RUNTIME-091",
    "org.cvf.worker": "/root/outbound_intake",
}


def docker(*args: str, check: bool = True) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(
        ["docker", *args], text=True, stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT, check=False,
    )
    if check and result.returncode:
        raise RuntimeError(f"docker {args[0] if args else '<empty>'} exited {result.returncode}: {result.stdout.strip()}")
    return result


def utc_now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds")


def save_json(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2) + "\n", encoding="utf-8")


def labels_for(run_id: str) -> dict[str, str]:
    return {**LABELS, "org.cvf.run": run_id}


def label_args(labels: dict[str, str]) -> list[str]:
    result: list[str] = []
    for key, value in labels.items():
        result.extend(["--label", f"{key}={value}"])
    return result


def inspect_labels(kind: str, name: str) -> dict[str, str]:
    fmt = "{{json .Config.Labels}}" if kind == "container" else "{{json .Labels}}"
    result = docker("inspect", "--format", fmt, name, check=False)
    if result.returncode:
        return {}
    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError:
        return {}


def inspect_mounts(container: str) -> list[dict[str, object]]:
    result = docker("inspect", "--format", "{{json .Mounts}}", container, check=False)
    if result.returncode:
        return []
    try:
        mounts = json.loads(result.stdout)
    except json.JSONDecodeError:
        return []
    # Keep ownership-relevant fields only; never persist Docker host Source paths.
    safe_fields = ("Type", "Name", "Destination", "RW", "Propagation", "Driver", "Mode")
    return [{key: mount.get(key) for key in safe_fields if key in mount} for mount in mounts]


def assert_owned(state: dict[str, object]) -> None:
    expected = labels_for(str(state["run_id"]))
    container = str(state["container"])
    network = str(state["network"])
    actual_container = inspect_labels("container", container)
    actual_network = inspect_labels("network", network)
    if any(actual_container.get(key) != value for key, value in expected.items()):
        raise RuntimeError(f"refusing operation: container {container!r} is absent or labels do not match R091 ownership")
    if any(actual_network.get(key) != value for key, value in expected.items()):
        raise RuntimeError(f"refusing operation: network {network!r} is absent or labels do not match R091 ownership")


def write_manifest(state: dict[str, object]) -> None:
    safe = {key: value for key, value in state.items() if key != "password"}
    save_json(PROBE_DIR / "fixture_manifest.json", safe)


def start_fixture(args: argparse.Namespace) -> int:
    RUNTIME.mkdir(parents=True, exist_ok=True)
    if STATE_PATH.exists():
        raise RuntimeError(f"state file already exists; refusing to overwrite {STATE_PATH}")
    image_check = docker("image", "inspect", IMAGE, check=False)
    if image_check.returncode:
        raise RuntimeError(f"admitted image {IMAGE} is not cached; this helper does not initiate another image pull")
    repo_digests = json.loads(docker("image", "inspect", "--format", "{{json .RepoDigests}}", IMAGE).stdout.strip())
    admitted_digest = "mysql@sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b"
    if admitted_digest not in repo_digests:
        raise RuntimeError("cached mysql:8.0 image does not include the exact digest pinned by immutable R091 seed")

    run_id = uuid4().hex
    suffix = run_id[:12]
    network = f"ccma-r091-net-{suffix}"
    container = f"ccma-r091-db-{suffix}"
    db_name = "ccma_r091"
    db_user = "ccma_r091"
    password = secrets.token_urlsafe(24)
    labels = labels_for(run_id)
    image_id = docker("image", "inspect", "--format", "{{.Id}}", IMAGE).stdout.strip()
    repo_digests = docker("image", "inspect", "--format", "{{json .RepoDigests}}", IMAGE).stdout.strip()
    state: dict[str, object] = {
        "run_id": run_id,
        "container": container,
        "network": network,
        "db_name": db_name,
        "db_user": db_user,
        "password": password,
        "image": IMAGE,
        "image_id": image_id,
        "image_repo_digests": repo_digests,
        "host_port": None,
        "proof_directory": args.proof_directory,
        "state_file": args.state_file,
        "started_utc": utc_now(),
        "commands": [],
        "cleanup": None,
    }
    save_json(STATE_PATH, state)
    try:
        docker("network", "create", "--driver", "bridge", *label_args(labels), network)
        docker(
            "run", "--detach", "--name", container, "--network", network,
            *label_args(labels), "--publish", "127.0.0.1::3306",
            "--env", "MYSQL_ROOT_PASSWORD=" + password,
            "--env", "MYSQL_DATABASE=" + db_name,
            "--env", "MYSQL_USER=" + db_user,
            "--env", "MYSQL_PASSWORD=" + password,
            image_id, "--skip-log-bin", "--log-bin-trust-function-creators=1",
        )
        state["mounts"] = inspect_mounts(container)
        save_json(STATE_PATH, state)
        write_manifest(state)
        port_line = docker("port", container, "3306/tcp").stdout.strip().splitlines()[0]
        match = re.fullmatch(r"127\.0\.0\.1:(\d+)", port_line)
        if not match:
            raise RuntimeError(f"unexpected MySQL host bind {port_line!r}; expected an ephemeral loopback-only port")
        state["host_port"] = int(match.group(1))
        state["network_internal"] = docker("network", "inspect", "--format", "{{.Internal}}", network).stdout.strip().lower() == "true"
        requested_raw = docker("inspect", "--format", "{{json .HostConfig.PortBindings}}", container).stdout.strip()
        requested = json.loads(requested_raw).get("3306/tcp") or []
        actual_raw = docker("inspect", "--format", "{{json .NetworkSettings.Ports}}", container).stdout.strip()
        actual = (json.loads(actual_raw).get("3306/tcp") or [])
        if len(requested) != 1 or requested[0].get("HostIp") != "127.0.0.1":
            raise RuntimeError(f"container request is not one loopback-only 3306 mapping: {requested_raw}")
        if len(actual) != 1 or actual[0].get("HostIp") != "127.0.0.1" or actual[0].get("HostPort") != str(state["host_port"]):
            raise RuntimeError(f"actual container port publication is not the expected single 127.0.0.1 mapping: {actual_raw}")
        assert_owned(state)
        save_json(STATE_PATH, state)
        write_manifest(state)

        deadline = time.monotonic() + 120
        ready = False
        while time.monotonic() < deadline:
            try:
                with socket.create_connection(("127.0.0.1", int(state["host_port"])), timeout=3):
                    tcp_ready = True
            except OSError:
                tcp_ready = False
            if tcp_ready:
                sql_check = docker(
                    "exec", container, "mysql", "--protocol=tcp", "--host=127.0.0.1", "--port=3306",
                    "--user=" + db_user, "--password=" + password, "--database=" + db_name,
                    "--batch", "--skip-column-names", "--execute=SELECT 1", check=False,
                )
                if sql_check.returncode == 0 and sql_check.stdout.strip().splitlines()[-1:] == ["1"]:
                    ready = True
                    break
            time.sleep(1)
        if not ready:
            raise RuntimeError("synthetic MySQL did not pass loopback TCP and application-user SELECT 1 readiness within 120 seconds")
        state["ready_utc"] = utc_now()
        state["readiness_method"] = "host 127.0.0.1 TCP handshake plus in-container app-user SELECT 1 over TCP"
        save_json(STATE_PATH, state)
        write_manifest(state)
        print(json.dumps({
            "status": "READY",
            "run_id": run_id,
            "container": container,
            "network": network,
            "mysql_image": IMAGE,
            "image_id": image_id,
            "image_repo_digests": repo_digests,
            "bind": f"{port_line} -> internal 3306",
            "network_internal": state["network_internal"],
            "password": "redacted",
        }, indent=2))
        return 0
    except Exception as error:
        state["startup_error"] = f"{type(error).__name__}: {error}"
        save_json(STATE_PATH, state)
        write_manifest(state)
        try:
            stop_fixture(argparse.Namespace())
        except Exception as cleanup_error:  # Preserve the original fixture-start failure.
            print(f"fixture startup cleanup also failed: {cleanup_error}", file=sys.stderr)
        raise


def run_test(args: argparse.Namespace) -> int:
    if not STATE_PATH.exists():
        raise RuntimeError("no R091 fixture state; start the explicitly owned fixture first")
    state = json.loads(STATE_PATH.read_text(encoding="utf-8"))
    if state.get("proof_directory", PROBE_DIR.relative_to(ROOT).as_posix()) != PROBE_DIR.relative_to(ROOT).as_posix():
        raise RuntimeError("selected proof directory does not match the owned fixture state")
    if state.get("state_file", STATE_PATH.relative_to(ROOT).as_posix()) != STATE_PATH.relative_to(ROOT).as_posix():
        raise RuntimeError("selected state file does not match the owned fixture state")
    assert_owned(state)
    if not state.get("host_port") or not state.get("ready_utc"):
        raise RuntimeError("fixture state is incomplete; host port and MySQL readiness must be persisted before tests run")
    command = args.command
    if command and command[0] == "--":
        command = command[1:]
    if not command:
        raise RuntimeError("run requires a command after --")
    cwd = (ROOT / args.cwd).resolve()
    if not cwd.is_relative_to(ROOT):
        raise RuntimeError("test work directory must remain inside the target workspace")
    executable = Path(command[0])
    if not executable.is_absolute():
        child_relative = (cwd / executable).resolve()
        if child_relative.is_file() and child_relative.is_relative_to(ROOT):
            command[0] = str(child_relative)
    env = os.environ.copy()
    env["TEST_DB_DSN"] = (
        f"{state['db_user']}:{state['password']}@tcp(127.0.0.1:{state['host_port']})/"
        f"{state['db_name']}?charset=utf8mb4&parseTime=True&loc=Local"
    )
    started_utc = utc_now()
    try:
        result = subprocess.run(
            command, cwd=cwd, env=env, text=True, encoding="utf-8",
            errors="replace", stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
            timeout=900,
        )
        output = result.stdout
        exit_code: int | str = result.returncode
    except subprocess.TimeoutExpired as timeout:
        output = timeout.stdout or ""
        if isinstance(output, bytes):
            output = output.decode("utf-8", errors="replace")
        output += "\n[R091 fixture runner timeout after 900 seconds; partial result is NOT PASS]\n"
        exit_code = "TIMEOUT"
    except OSError as spawn_error:
        output = f"[R091 test process did not start; no test executed; NOT source detector: {type(spawn_error).__name__}: {spawn_error}]\n"
        exit_code = "SPAWN_ERROR"
    proof_path = PROBE_DIR / f"{args.proof_name}.log"
    proof_path.parent.mkdir(parents=True, exist_ok=True)
    proof_path.write_text(output, encoding="utf-8")
    state.setdefault("commands", []).append({
        "proof": proof_path.relative_to(ROOT).as_posix(),
        "command": command,
        "cwd": cwd.relative_to(ROOT).as_posix(),
        "started_utc": started_utc,
        "finished_utc": utc_now(),
        "exit_code": exit_code,
    })
    save_json(STATE_PATH, state)
    write_manifest(state)
    try:
        sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    except (AttributeError, ValueError):
        pass
    sys.stdout.write(output)
    print(f"R091 fixture test result: exit_code={exit_code}; log={proof_path.relative_to(ROOT).as_posix()}")
    if exit_code == "TIMEOUT":
        return 124
    if exit_code == "SPAWN_ERROR":
        return 127
    return int(exit_code)


def stop_fixture(_: argparse.Namespace) -> int:
    if not STATE_PATH.exists():
        print("no R091 fixture state; nothing removed")
        return 0
    state = json.loads(STATE_PATH.read_text(encoding="utf-8"))
    if state.get("proof_directory", PROBE_DIR.relative_to(ROOT).as_posix()) != PROBE_DIR.relative_to(ROOT).as_posix():
        raise RuntimeError("selected proof directory does not match the owned fixture state")
    if state.get("state_file", STATE_PATH.relative_to(ROOT).as_posix()) != STATE_PATH.relative_to(ROOT).as_posix():
        raise RuntimeError("selected state file does not match the owned fixture state")
    run_id = str(state["run_id"])
    expected = labels_for(run_id)
    container = str(state["container"])
    network = str(state["network"])
    actual_container = inspect_labels("container", container)
    actual_network = inspect_labels("network", network)
    if actual_container and any(actual_container.get(key) != value for key, value in expected.items()):
        raise RuntimeError(f"refusing to remove container {container!r}: R091 ownership labels mismatch")
    if actual_network and any(actual_network.get(key) != value for key, value in expected.items()):
        raise RuntimeError(f"refusing to remove network {network!r}: R091 ownership labels mismatch")
    removed_container = False
    removed_network = False
    mounts = inspect_mounts(container) if actual_container else state.get("mounts", [])
    state["mounts"] = mounts
    if actual_container:
        docker("rm", "--force", "--volumes", container)
        removed_container = True
    remaining_volumes: list[str] = []
    checked_volumes: list[str] = []
    for mount in mounts:
        if not isinstance(mount, dict) or mount.get("Type") != "volume" or not mount.get("Name"):
            continue
        name = str(mount["Name"])
        checked_volumes.append(name)
        if docker("volume", "inspect", "--format", "{{.Name}}", name, check=False).returncode == 0:
            remaining_volumes.append(name)
    if actual_network:
        docker("network", "rm", network)
        removed_network = True
    state["cleanup"] = {
        "at_utc": utc_now(),
        "container_removed": removed_container,
        "container_mounts_captured_before_removal": mounts,
        "volume_names_checked_after_removal": checked_volumes,
        "remaining_owned_container_volumes": remaining_volumes,
        "all_recorded_container_volumes_absent": not remaining_volumes,
        "network_removed": removed_network,
        "ownership_labels_verified": True,
    }
    write_manifest(state)
    STATE_PATH.unlink()
    print(json.dumps(state["cleanup"], indent=2))
    return 1 if remaining_volumes else 0


def main() -> int:
    global PROBE_DIR, STATE_PATH
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="action", required=True)
    start = sub.add_parser("start", help="create the one loopback-only R091 MySQL fixture")
    run = sub.add_parser("run", help="run a test command with TEST_DB_DSN; keep fixture alive")
    run.add_argument("--proof-name", required=True, help="safe stem for captured stdout/stderr")
    run.add_argument("--cwd", default="backend", help="target-workspace-relative command directory")
    run.add_argument("command", nargs=argparse.REMAINDER)
    stop = sub.add_parser("stop", help="remove only the exact R091-labeled container and network")
    for command_parser in (start, run, stop):
        command_parser.add_argument("--proof-dir", choices=sorted(ALLOWED_PROBE_DIRS), default="docs/reviews/probes/r091_outbound_delivery")
        command_parser.add_argument("--state-file", choices=sorted(ALLOWED_STATE_FILES), default=".cvf/runtime/r091_outbound_fixture_state.json")
    args = parser.parse_args()
    PROBE_DIR = (ROOT / args.proof_dir).resolve()
    STATE_PATH = (ROOT / args.state_file).resolve()
    if not PROBE_DIR.is_relative_to(ROOT) or not STATE_PATH.is_relative_to(ROOT / ".cvf" / "runtime"):
        raise RuntimeError("fixture paths must remain within the admitted R091 proof/runtime families")
    if args.action == "start":
        args.proof_directory = args.proof_dir
        args.state_file = args.state_file
        return start_fixture(args)
    if args.action == "run":
        return run_test(args)
    return stop_fixture(args)


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as error:
        print(f"r091_outbound_fixture: {error}", file=sys.stderr)
        raise SystemExit(1)
