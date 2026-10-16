"""One-shot, fail-closed supervisor for the admitted R108 native diagnostic.

Importing and the default `inspect` command are read-only. `start` requires the
committed independent Root admission and burns a single append-only campaign.
It never reads Docker Config.Env or any .env file.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import shutil
import socket
import subprocess
import time
from pathlib import Path
from typing import Any


TRANCHE = "CCMAI-RUNTIME-108"
WORKER = "/root/r108_browser_luna"
SEED_PATH = "CVF_SESSION/authority/CCMAI-RUNTIME-108.json"
RECORD_PATH = "CVF_SESSION/tranches/CCMAI-RUNTIME-108.json"
STATE_PATH = "CVF_SESSION/ACTIVE_SESSION_STATE.json"
ORDER_PATH = "docs/work_orders/CCMAI_RUNTIME_108_REOPEN_DIAGNOSTIC.md"
TEST_PATH = "backend/api/handlers/intervention_reopen_r108_test.go"
RUNNER_PATH = "docs/reviews/probes/r108_luna/r108_native_runner.py"
ADMISSION_PATH = "docs/reviews/probes/r108_root/source_admission.json"
RUNTIME_PATH = ".cvf/runtime/r108_native"

CONTAINER = "e1555e03d69360e1cd7729afd9e52856443f5172d76454653beec981a7d6477a"
IMAGE = "sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b"
VOLUME = "ccmai-r101-review-final-20261015"
NETWORK = "35c3959ac41517050defad05cbad7a08717b24c8879683e25baf3e2702625e98"
NAME = "ccmai-r101-review-final-20261015"
LABEL_KEY = "ccmai.tranche"
LABEL_VALUE = "CCMAI-RUNTIME-101"
DATABASE = "r101_fixture"
DOCKER_EXE = "C:/Users/tiennm/AppData/Local/Programs/DockerDesktop/resources/bin/docker.exe"
DOCKER_SHA256 = "6c699720bb9523dedb9eb5d22a49951ea9210dbdb1314a3554029eee9be76896"
DOCKER_HOST = "npipe:////./pipe/dockerDesktopLinuxEngine"
GO_EXE = "C:/Program Files/Go/bin/go.exe"
HEX40 = re.compile(r"[0-9a-f]{40}\Z")
HEX64 = re.compile(r"[0-9a-f]{64}\Z")
ACTION_SEQUENCE = ["CORRECT", "ASSIGN", "RESOLVE", "REOPEN"]
CAMPAIGN_SECONDS = 600
CLEANUP_RESERVE_SECONDS = 75

# cvf-allow-secret-fixture: synthetic disposable R101 fixture credential only.
# Never print, serialize, put in argv, or inherit this value from the environment.
FIXTURE_PASSWORD = "r101LocalOnly42"

CONTAINER_FORMAT = (
    '{"Id":{{json .Id}},"Name":{{json .Name}},"Image":{{json .Image}},'
    '"Status":{{json .State.Status}},'
    '"FixtureLabel":{{json (index .Config.Labels "ccmai.tranche")}},'
    '"Mounts":[{{range $i,$m := .Mounts}}{{if $i}},{{end}}'
    '{"Type":{{json $m.Type}},"Name":{{json $m.Name}},'
    '"Destination":{{json $m.Destination}},"RW":{{json $m.RW}}}{{end}}],'
    '"PortBindings":{{json .HostConfig.PortBindings}},'
    '"Networks":[{{ $first := true }}{{range $name,$net := .NetworkSettings.Networks}}'
    '{{if not $first}},{{end}}{{ $first = false }}'
    '{"Name":{{json $name}},"NetworkID":{{json $net.NetworkID}}}{{end}}],'
    '"Ports":{{json .NetworkSettings.Ports}}}'
)
PROTECTED_FORMAT = (
    '{"Id":{{json .Id}},"Name":{{json .Name}},"Image":{{json .Image}},'
    '"Status":{{json .State.Status}},"Mounts":[{{range $i,$m := .Mounts}}{{if $i}},{{end}}'
    '{"Type":{{json $m.Type}},"Name":{{json $m.Name}},'
    '"Destination":{{json $m.Destination}},"RW":{{json $m.RW}}}{{end}}],'
    '"PortBindings":{{json .HostConfig.PortBindings}},'
    '"Networks":[{{ $first := true }}{{range $name,$net := .NetworkSettings.Networks}}'
    '{{if not $first}},{{end}}{{ $first = false }}'
    '{"Name":{{json $name}},"NetworkID":{{json $net.NetworkID}}}{{end}}]}'
)
NETWORK_FORMAT = (
    '{"Id":{{json .Id}},"Name":{{json .Name}},'
    '"Containers":[{{ $first := true }}{{range $id,$c := .Containers}}'
    '{{if not $first}},{{end}}{{ $first = false }}'
    '{"Id":{{json $id}},"Name":{{json $c.Name}}}{{end}}]}'
)


class Hold(RuntimeError):
    """A prerequisite failed; no additional attempt is permitted."""


def require(condition: bool, reason: str) -> None:
    if not condition:
        raise Hold(reason)


def sha256_bytes(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def sha256_file(path: Path) -> str:
    return sha256_bytes(path.read_bytes())


def read_json(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    require(isinstance(value, dict), f"JSON object required: {path.name}")
    return value


def git(root: Path, *args: str, timeout: int = 20, deadline: float | None = None) -> subprocess.CompletedProcess[bytes]:
    try:
        require(deadline is None or deadline > time.monotonic(), "R108 campaign deadline reached during source verification")
        executable = shutil.which("git")
        require(executable is not None, "Git executable is unavailable")
        env = minimal_env()
        env["GIT_TERMINAL_PROMPT"] = "0"
        env["GIT_CONFIG_NOSYSTEM"] = "1"
        if deadline is not None:
            timeout = min(timeout, max(0.1, deadline - time.monotonic()))
        return subprocess.run(
            [executable, *args], cwd=root, env=env, stdout=subprocess.PIPE,
            stderr=subprocess.PIPE, timeout=timeout, check=False,
        )
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise Hold("bounded Git inspection failed") from exc


def git_text(root: Path, *args: str) -> str:
    result = git(root, *args)
    require(result.returncode == 0, "Git source-history inspection failed")
    return result.stdout.decode("utf-8", errors="strict").strip()


def verify_admission(root: Path, deadline: float | None = None) -> tuple[dict[str, Any], Path, Path, str]:
    admission_path = root / ADMISSION_PATH
    require(admission_path.is_file(), "Root committed source admission is missing")
    admission = read_json(admission_path)
    committed_admission = git(root, "show", f"HEAD:{ADMISSION_PATH}", deadline=deadline)
    require(committed_admission.returncode == 0 and committed_admission.stdout == admission_path.read_bytes(), "Root source admission must be committed")
    admission_status = git(root, "status", "--porcelain", "--untracked-files=all", "--", ADMISSION_PATH, deadline=deadline)
    require(admission_status.returncode == 0 and not admission_status.stdout.strip(), "Root source admission worktree is not clean")
    require(admission.get("trancheId") == TRANCHE, "source admission tranche mismatch")
    require(admission.get("decision") == "RUNTIME_ADMITTED", "Root runtime admission is not active")
    require(admission.get("reviewer") == "Codex /root", "independent Root reviewer mismatch")
    reviewed_commit = admission.get("reviewedSourceCommit")
    require(isinstance(reviewed_commit, str) and HEX40.fullmatch(reviewed_commit) is not None, "reviewed source commit missing")
    pins = admission.get("sourcePins")
    require(isinstance(pins, dict) and set(pins) == {TEST_PATH, RUNNER_PATH}, "source admission must pin exactly the test and runner")
    for relative, expected in pins.items():
        candidate = root / relative
        require(candidate.is_file() and isinstance(expected, str) and HEX64.fullmatch(expected) is not None, "admitted source pin malformed")
        require(sha256_file(candidate) == expected, f"source pin mismatch: {relative}")
        committed = git(root, "show", f"HEAD:{relative}", deadline=deadline)
        require(committed.returncode == 0 and committed.stdout == candidate.read_bytes(), f"committed source differs: {relative}")
    require(admission.get("seedSha256") == sha256_file(root / SEED_PATH), "dispatcher seed pin mismatch")
    require(admission.get("workOrderSha256") == sha256_file(root / ORDER_PATH), "work-order pin mismatch")
    require(git(root, "merge-base", "--is-ancestor", reviewed_commit, "HEAD", deadline=deadline).returncode == 0, "reviewed source commit is not an ancestor of HEAD")
    source_delta = git(root, "diff", "--quiet", reviewed_commit, "HEAD", "--", "backend", RUNNER_PATH, deadline=deadline)
    require(source_delta.returncode == 0, "R108 source changed after independent review")
    backend_status = git(root, "status", "--porcelain", "--untracked-files=all", "--", "backend", deadline=deadline)
    runner_status = git(root, "status", "--porcelain", "--untracked-files=all", "--", RUNNER_PATH, deadline=deadline)
    require(backend_status.returncode == 0 and not backend_status.stdout.strip(), "backend worktree must be committed and clean")
    require(runner_status.returncode == 0 and not runner_status.stdout.strip(), "runner worktree must be committed and clean")

    head_result = git(root, "rev-parse", "HEAD", deadline=deadline)
    require(head_result.returncode == 0, "Git HEAD inspection failed")
    head = head_result.stdout.decode("utf-8", errors="strict").strip()
    seed = read_json(root / SEED_PATH)
    record = read_json(root / RECORD_PATH)
    state = read_json(root / STATE_PATH)
    require(seed.get("trancheId") == TRANCHE and record.get("trancheId") == TRANCHE, "current tranche identity mismatch")
    require(record.get("status") == "BUILD" and record.get("phase") == "BUILD", "R108 record is not in BUILD")
    require(state.get("activeTranche") == TRANCHE and state.get("activePhase") == "BUILD" and state.get("currentMode") == "BUILD", "R108 continuity is not in BUILD")
    base_commit = record.get("baseCommit")
    require(isinstance(base_commit, str) and HEX40.fullmatch(base_commit) is not None, "tranche base commit missing")
    require(git(root, "merge-base", "--is-ancestor", base_commit, "HEAD", deadline=deadline).returncode == 0, "dispatcher base is not an ancestor of HEAD")
    committed_seed = git(root, "show", f"{base_commit}:{SEED_PATH}", deadline=deadline)
    require(committed_seed.returncode == 0 and committed_seed.stdout.replace(b"\r\n", b"\n") == (root / SEED_PATH).read_bytes().replace(b"\r\n", b"\n"), "dispatcher seed changed from its base commit")
    seed_additions = git(root, "log", "--diff-filter=A", "--format=%H", "--", SEED_PATH, deadline=deadline)
    require(seed_additions.returncode == 0, "dispatcher seed first-add history is unreadable")
    first_adds = [line for line in seed_additions.stdout.decode("ascii", errors="strict").splitlines() if HEX40.fullmatch(line)]
    require(bool(first_adds) and first_adds[-1] == base_commit, "dispatcher seed base must be its first committed content")
    order_text = (root / ORDER_PATH).read_text(encoding="utf-8")
    require(re.search(r"(?m)^Status:\s*ACTIVE\s*$", order_text) is not None, "R108 work order is not ACTIVE")
    require(seed.get("worker") == WORKER, "dispatcher worker identity mismatch")

    docker = Path(str(seed.get("dockerExe", ""))).resolve()
    require(docker.as_posix().lower() == DOCKER_EXE.lower(), "Docker executable path differs from the seed")
    require(docker.is_file() and sha256_file(docker) == seed.get("dockerSha256") == DOCKER_SHA256, "Docker executable pin mismatch")
    require(seed.get("dockerHost") == DOCKER_HOST, "only the pinned local Docker pipe is allowed")
    go_path = admission.get("goExe", GO_EXE)
    go_digest = admission.get("goSha256")
    go_exe = Path(str(go_path)).resolve()
    require(go_exe.as_posix().lower() == GO_EXE.lower(), "Go executable path differs from the pinned local installation")
    require(go_exe.is_file() and isinstance(go_digest, str) and HEX64.fullmatch(go_digest) is not None and sha256_file(go_exe) == go_digest, "Root Go executable pin mismatch")
    return admission, docker, go_exe, head


def minimal_env(go_exe: Path | None = None, dsn: str | None = None) -> dict[str, str]:
    allowed = ("SystemRoot", "WINDIR", "TEMP", "TMP", "USERPROFILE", "LOCALAPPDATA")
    env = {key: os.environ[key] for key in allowed if key in os.environ}
    system_root = env.get("SystemRoot") or env.get("WINDIR")
    path_parts = [str(go_exe.parent)] if go_exe is not None else []
    if system_root:
        path_parts.append(str(Path(system_root) / "System32"))
    if path_parts:
        env["PATH"] = ";".join(path_parts)
    if go_exe is not None:
        env.update({"GOPROXY": "off", "GOSUMDB": "off", "GOTOOLCHAIN": "local", "GOVCS": "*:off"})
    if dsn is not None:
        env["R108_DIAGNOSTIC"] = "1"
        env["R101_MYSQL_DSN"] = dsn
    return env


def redact_bytes(data: bytes) -> bytes:
    # Preserve complete output while masking only explicitly sensitive patterns.
    safe = data.replace(FIXTURE_PASSWORD.encode("utf-8"), b"<synthetic-fixture-password-redacted>")
    safe = re.sub(
        rb"root:[^@\s]+@tcp\(127\.0\.0\.1:\d+\)/r101_fixture\?parseTime=true&charset=utf8mb4",
        b"root:<redacted>@tcp(127.0.0.1:<port>)/r101_fixture?parseTime=true&charset=utf8mb4",
        safe,
    )
    safe = re.sub(rb"(?i)(authorization\s*:\s*bearer\s+)[A-Za-z0-9._~+/-]+=*", rb"\1<redacted>", safe)
    return safe


def write_bytes_once(path: Path, data: bytes) -> None:
    with path.open("xb") as stream:
        stream.write(data)
        stream.flush()
        os.fsync(stream.fileno())


def write_json_once(path: Path, value: Any) -> None:
    data = (json.dumps(value, ensure_ascii=False, sort_keys=True, indent=2) + "\n").encode("utf-8")
    write_bytes_once(path, data)


def append_reservation(directory: Path, name: str, campaign_id: str) -> None:
    write_json_once(directory / f"{name}.reservation.json", {
        "campaignId": campaign_id,
        "effect": name,
        "state": "RESERVED",
        "replayAllowed": False,
    })


class Campaign:
    def __init__(self, root: Path, directory: Path, docker: Path, go_exe: Path, campaign_id: str):
        self.root = root
        self.directory = directory
        self.docker = docker
        self.go_exe = go_exe
        self.campaign_id = campaign_id
        self.docker_config = directory / "docker-cli-config"
        self.docker_prefix = [str(docker), "--host", DOCKER_HOST, "--config", str(self.docker_config)]
        self.command_number = 0
        self.owned_processes: list[subprocess.Popen[bytes]] = []
        self.process_details: list[dict[str, Any]] = []
        self.start_attempted = False
        self.protected_before: list[dict[str, Any]] | None = None
        self.started_fixture_identity: dict[str, Any] | None = None
        self.deadline = 0.0
        self.effect_deadline = 0.0

    def remaining(self, cleanup: bool = False) -> float:
        limit = self.deadline if cleanup else self.effect_deadline
        return max(0.0, limit - time.monotonic())

    def _record_command(self, label: str, argv: list[str], result: dict[str, Any], out: bytes, err: bytes) -> None:
        safe_out, safe_err = redact_bytes(out), redact_bytes(err)
        self.command_number += 1
        stem = f"command_{self.command_number:03d}_{label}"
        write_bytes_once(self.directory / f"{stem}.stdout.bin", safe_out)
        write_bytes_once(self.directory / f"{stem}.stderr.bin", safe_err)
        write_json_once(self.directory / f"{stem}.json", {
            "argv": [redact_bytes(arg.encode("utf-8")).decode("utf-8", errors="replace") for arg in argv],
            "exitCode": result.get("exitCode"),
            "failureType": result.get("failureType"),
            "timedOut": result.get("timedOut", False),
            "pid": result.get("pid"),
            "stdoutSha256": sha256_bytes(safe_out),
            "stdoutBytes": len(safe_out),
            "stderrSha256": sha256_bytes(safe_err),
            "stderrBytes": len(safe_err),
            "outputRedaction": "exact synthetic fixture credential and bearer-token patterns only",
        })

    def run_process(self, label: str, argv: list[str], env: dict[str, str], timeout: float, *, cleanup: bool = False) -> dict[str, Any]:
        time_left = self.remaining(cleanup=cleanup)
        require(time_left > 0, "R108 campaign deadline reached")
        allowed_timeout = min(timeout, time_left)
        flags = getattr(subprocess, "CREATE_NEW_PROCESS_GROUP", 0) | getattr(subprocess, "CREATE_NO_WINDOW", 0)
        process: subprocess.Popen[bytes] | None = None
        out = b""
        err = b""
        timed_out = False
        failure_type = None
        termination: dict[str, Any] | None = None
        try:
            process = subprocess.Popen(
                argv, cwd=self.root, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                creationflags=flags,
            )
            self.owned_processes.append(process)
            try:
                out, err = process.communicate(timeout=allowed_timeout)
            except subprocess.TimeoutExpired:
                timed_out = True
                termination = self.stop_owned(process)
                out, err = process.communicate(timeout=5)
        except (OSError, subprocess.TimeoutExpired) as exc:
            failure_type = type(exc).__name__
            if process is not None:
                termination = self.stop_owned(process)
                try:
                    out, err = process.communicate(timeout=3)
                except subprocess.TimeoutExpired:
                    err += b"\nowned process output collection timed out after stop\n"
        result = {
            "exitCode": process.returncode if process is not None else None,
            "failureType": failure_type,
            "timedOut": timed_out,
            "pid": process.pid if process is not None else None,
        }
        self.process_details.append({
            "label": label,
            "pid": result["pid"],
            "exitCode": result["exitCode"],
            "timedOut": timed_out,
            "failureType": failure_type,
            "ownedChildTreeTermination": termination,
        })
        self._record_command(label, argv, result, out, err)
        return {**result, "stdout": redact_bytes(out), "stderr": redact_bytes(err)}

    def stop_owned(self, process: subprocess.Popen[bytes]) -> dict[str, Any]:
        if process.poll() is not None:
            return {"attempted": False, "exitCode": process.returncode, "stillRunning": False}
        # Windows taskkill is scoped to this directly owned PID and its child tree.
        system_root = os.environ.get("SystemRoot") or os.environ.get("WINDIR")
        taskkill = Path(system_root) / "System32" / "taskkill.exe" if system_root else None
        tree_attempted = False
        tree_exit_code = None
        if taskkill is not None and taskkill.is_file():
            try:
                tree_attempted = True
                tree_result = subprocess.run(
                    [str(taskkill), "/PID", str(process.pid), "/T", "/F"],
                    cwd=self.root, env=minimal_env(), stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE, timeout=10, check=False,
                )
                tree_exit_code = tree_result.returncode
            except (OSError, subprocess.TimeoutExpired):
                tree_exit_code = None
        direct_attempted = False
        try:
            direct_attempted = True
            process.terminate()
            process.wait(timeout=5)
        except (OSError, subprocess.TimeoutExpired):
            try:
                process.kill()
                process.wait(timeout=5)
            except (OSError, subprocess.TimeoutExpired):
                pass
        return {
            "attempted": True,
            "pid": process.pid,
            "ownedChildTreeTerminationAttempted": tree_attempted,
            "ownedChildTreeTerminationExitCode": tree_exit_code,
            "directHandleTerminationAttempted": direct_attempted,
            "exitCode": process.poll(),
            "stillRunning": process.poll() is None,
        }

    def docker_command(self, label: str, args: list[str], timeout: float = 15, *, cleanup: bool = False) -> bytes:
        result = self.run_process(label, [*self.docker_prefix, *args], minimal_env(), timeout, cleanup=cleanup)
        require(result["failureType"] is None and not result["timedOut"], f"Docker {label} command timed out or could not start")
        require(result["exitCode"] == 0, f"Docker {label} command failed")
        return result["stdout"]

    def target_container(self, *, cleanup: bool = False) -> dict[str, Any]:
        out = self.docker_command(
            "target_inspect",
            ["container", "inspect", "--format", CONTAINER_FORMAT, CONTAINER],
            timeout=15, cleanup=cleanup,
        )
        rows = [json.loads(line) for line in out.decode("utf-8", errors="strict").splitlines() if line.strip()]
        require(len(rows) == 1, "exact fixture inspect count mismatch")
        return rows[0]

    def target_network(self, *, cleanup: bool = False) -> dict[str, Any]:
        out = self.docker_command(
            "target_network_inspect",
            ["network", "inspect", "--format", NETWORK_FORMAT, NETWORK],
            timeout=15, cleanup=cleanup,
        )
        rows = [json.loads(line) for line in out.decode("utf-8", errors="strict").splitlines() if line.strip()]
        require(len(rows) == 1, "exact fixture network inspect count mismatch")
        return rows[0]

    def volume_users(self, *, cleanup: bool = False) -> list[str]:
        out = self.docker_command(
            "fixture_volume_users",
            ["ps", "-aq", "--no-trunc", "--filter", f"volume={VOLUME}"], cleanup=cleanup,
        )
        return sorted(line.strip() for line in out.decode("utf-8", errors="strict").splitlines() if line.strip())

    def protected_inventory(self, *, cleanup: bool = False) -> list[dict[str, Any]]:
        ids_raw = self.docker_command("all_container_ids", ["ps", "-aq", "--no-trunc"], cleanup=cleanup)
        ids = sorted({line.strip() for line in ids_raw.decode("utf-8", errors="strict").splitlines() if line.strip()} - {CONTAINER})
        if not ids:
            return []
        out = self.docker_command("protected_container_inspect", ["container", "inspect", "--format", PROTECTED_FORMAT, *ids], timeout=30, cleanup=cleanup)
        rows = [json.loads(line) for line in out.decode("utf-8", errors="strict").splitlines() if line.strip()]
        require(len(rows) == len(ids), "protected container inventory count mismatch")
        for row in rows:
            row["Mounts"] = sorted(row.get("Mounts", []), key=lambda item: (item.get("Destination", ""), item.get("Name", "")))
            row["Networks"] = sorted(row.get("Networks", []), key=lambda item: (item.get("Name", ""), item.get("NetworkID", "")))
        return sorted(rows, key=lambda row: row["Id"])

    def validate_fixture(self, container: dict[str, Any], network: dict[str, Any], users: list[str], *, expected_status: str) -> None:
        require(container.get("Id") == CONTAINER, "fixture ID mismatch")
        require(container.get("Name") == "/" + NAME, "fixture name mismatch")
        require(container.get("Image") == IMAGE, "fixture image mismatch")
        require(container.get("FixtureLabel") == LABEL_VALUE, "fixture ownership label mismatch")
        require(container.get("Status") == expected_status, f"fixture must be {expected_status}")
        mounts = container.get("Mounts")
        require(isinstance(mounts, list) and len(mounts) == 1, "fixture mount count mismatch")
        mount = mounts[0]
        require(mount == {"Type": "volume", "Name": VOLUME, "Destination": "/var/lib/mysql", "RW": True}, "fixture volume pin mismatch")
        require(container.get("PortBindings") == {"3306/tcp": [{"HostIp": "127.0.0.1", "HostPort": ""}]}, "fixture port must be dynamic loopback only")
        networks = container.get("Networks")
        require(isinstance(networks, list) and networks == [{"Name": NAME, "NetworkID": NETWORK}], "fixture network pin mismatch")
        require(network.get("Id") == NETWORK and network.get("Name") == NAME, "fixture network identity mismatch")
        require(network.get("Containers") == [], "fixture network is not exclusive")
        require(users == [CONTAINER], "fixture volume is shared or attached elsewhere")

    def mysql_port(self, container: dict[str, Any]) -> int:
        ports = container.get("Ports", {}).get("3306/tcp")
        require(isinstance(ports, list) and len(ports) == 1, "fixture MySQL host port count mismatch")
        item = ports[0]
        require(item.get("HostIp") == "127.0.0.1", "fixture MySQL is not loopback-bound")
        raw = item.get("HostPort")
        require(isinstance(raw, str) and raw.isdecimal(), "fixture MySQL port is not assigned")
        port = int(raw)
        require(1 <= port <= 65535, "fixture MySQL port invalid")
        return port

    def running_fixture_ready_shape(self, container: dict[str, Any], network: dict[str, Any], users: list[str]) -> None:
        require(container.get("Id") == CONTAINER and container.get("Name") == "/" + NAME and container.get("Image") == IMAGE, "running fixture identity mismatch")
        require(container.get("FixtureLabel") == LABEL_VALUE and container.get("Status") == "running", "running fixture label/state mismatch")
        require(container.get("Mounts") == [{"Type": "volume", "Name": VOLUME, "Destination": "/var/lib/mysql", "RW": True}], "running fixture volume mismatch")
        require(container.get("PortBindings") == {"3306/tcp": [{"HostIp": "127.0.0.1", "HostPort": ""}]}, "running fixture port binding mismatch")
        require(container.get("Networks") == [{"Name": NAME, "NetworkID": NETWORK}], "running fixture network membership mismatch")
        attached = network.get("Containers")
        require(network.get("Id") == NETWORK and network.get("Name") == NAME, "running fixture network identity mismatch")
        require(isinstance(attached, list) and len(attached) == 1 and attached[0].get("Id") == CONTAINER, "running fixture network is not exclusive")
        require(users == [CONTAINER], "running fixture volume is shared")

    def wait_mysql_handshake(self, port: int) -> int:
        probes = 0
        while self.remaining() > 2:
            probes += 1
            try:
                with socket.create_connection(("127.0.0.1", port), timeout=min(2.0, self.remaining())) as connection:
                    connection.settimeout(min(2.0, self.remaining()))
                    header = self._recv_exact(connection, 4)
                    payload_size = int.from_bytes(header[:3], "little")
                    require(0 < payload_size <= 65535, "MySQL handshake packet length invalid")
                    first_payload = self._recv_exact(connection, min(payload_size, 1))
                    require(first_payload != b"\xff", "MySQL reported an initialization error")
                    return probes
            except (OSError, TimeoutError):
                time.sleep(min(1.0, self.remaining()))
        raise Hold("fixture MySQL did not become ready before the campaign deadline")

    @staticmethod
    def _recv_exact(connection: socket.socket, size: int) -> bytes:
        data = bytearray()
        while len(data) < size:
            part = connection.recv(size - len(data))
            require(bool(part), "MySQL handshake ended early")
            data.extend(part)
        return bytes(data)

    def stop_fixture(self) -> dict[str, Any]:
        container = self.target_container(cleanup=True)
        require(container.get("Id") == CONTAINER and container.get("Name") == "/" + NAME and container.get("Image") == IMAGE, "fixture identity changed before cleanup")
        if container.get("Status") == "running":
            output = self.docker_command("fixture_stop", ["stop", "--time", "10", CONTAINER], timeout=min(15, max(1, self.remaining(cleanup=True))), cleanup=True)
            require(output.decode("utf-8", errors="replace").strip() == CONTAINER, "fixture stop identity mismatch")
        final = self.target_container(cleanup=True)
        final_network = self.target_network(cleanup=True)
        final_users = self.volume_users(cleanup=True)
        self.validate_fixture(final, final_network, final_users, expected_status="exited")
        return {
            "stopAttempted": True,
            "finalStatus": final.get("Status"),
            "container": final.get("Id"),
            "network": final_network.get("Id"),
            "volumeUsers": final_users,
        }


def verify_diagnostic(stdout: bytes, exit_code: int | None) -> tuple[str, dict[str, Any]]:
    lines = [line for line in stdout.decode("utf-8", errors="replace").splitlines() if line.startswith("R108_DIAGNOSTIC_JSON=")]
    require(len(lines) == 1, "native test must emit exactly one diagnostic JSON marker")
    try:
        report = json.loads(lines[0].split("=", 1)[1])
    except json.JSONDecodeError as exc:
        raise Hold("native diagnostic JSON is malformed") from exc
    require("NOT_RUN:" not in stdout.decode("utf-8", errors="replace") and "--- SKIP: TestR108ReopenDiagnostic" not in stdout.decode("utf-8", errors="replace"), "native diagnostic was skipped")
    actions = report.get("actions")
    require(isinstance(actions, list) and len(actions) == 4, "native diagnostic action count is not exactly four")
    require([action.get("action") for action in actions] == ACTION_SEQUENCE, "native diagnostic action sequence mismatch")
    require(all(isinstance(action.get("http_status"), int) for action in actions), "native diagnostic action status missing")
    require(all(action["http_status"] == 200 for action in actions[:3]), "a pre-REOPEN action failed; one-shot campaign stops")
    if actions[3]["http_status"] == 200:
        require(exit_code == 0 and report.get("lifecycle_accepted") is True, "native success requires a passing accepted lifecycle")
        return "NOT_REPRODUCED", report
    if actions[3]["http_status"] == 503:
        require(exit_code != 0, "expected 503 diagnostic did not fail the native assertion")
        require(report.get("failed_action_state_comparison_available") is True and report.get("failed_action_state_unchanged") is True, "REOPEN 503 did not prove unchanged persisted case and events")
        require(report.get("proposal_comparison_available") is True and report.get("proposal_unchanged") is True, "REOPEN 503 did not prove proposal retention")
        return "REPRODUCED_503", report
    raise Hold("REOPEN produced an unexpected HTTP result; no retry is allowed")


def file_manifest(directory: Path) -> list[dict[str, Any]]:
    result = []
    for path in sorted(directory.rglob("*")):
        if path.is_file() and path.name != "files_manifest.json":
            result.append({
                "path": path.relative_to(directory).as_posix(),
                "bytes": path.stat().st_size,
                "sha256": sha256_file(path),
            })
    return result


def start_campaign(root: Path) -> int:
    admission, docker, go_exe, head = verify_admission(root)
    runtime = root / RUNTIME_PATH
    campaign_dir = runtime / "campaign"
    require(not campaign_dir.exists(), "R108 one-shot campaign slot already exists; replay is forbidden")
    require(not runtime.exists() or not any(runtime.iterdir()), "R108 append-only runtime ledger is not empty; replay is forbidden")
    runtime.mkdir(parents=True, exist_ok=True)
    campaign_dir.mkdir(exist_ok=False)
    started_at = time.monotonic()
    campaign_id = f"r108-{int(time.time())}-{os.getpid()}"
    campaign = Campaign(root, campaign_dir, docker, go_exe, campaign_id)
    campaign.deadline = started_at + CAMPAIGN_SECONDS
    campaign.effect_deadline = campaign.deadline - CLEANUP_RESERVE_SECONDS
    intent = {
        "campaignId": campaign_id,
        "trancheId": TRANCHE,
        "headCommit": head,
        "reviewedSourceCommit": admission["reviewedSourceCommit"],
        "sourceAdmissionSha256": sha256_file(root / ADMISSION_PATH),
        "sourcePins": admission["sourcePins"],
        "seedSha256": admission["seedSha256"],
        "workOrderSha256": admission["workOrderSha256"],
        "goExe": str(go_exe),
        "goSha256": admission["goSha256"],
        "startedAtUnix": time.time(),
        "campaignLimitSeconds": CAMPAIGN_SECONDS,
        "cleanupReserveSeconds": CLEANUP_RESERVE_SECONDS,
        "budget": {"compile": 1, "fixtureStart": 1, "nativeInvocation": 1, "actionPosts": 4, "retries": 0},
        "environment": "minimal allowlist; synthetic test DSN is not serialized",
    }
    write_json_once(campaign_dir / "campaign_intent.json", intent)
    status = "HOLD"
    reason = "campaign did not reach a terminal result"
    native_classification: str | None = None
    diagnostic: dict[str, Any] | None = None
    binary_path = campaign_dir / "r108_handlers.test.exe"
    fixture_cleanup: dict[str, Any] = {"stopAttempted": False}
    protection_unchanged: bool | None = None
    cleanup_errors: list[str] = []
    go_process_exit: int | None = None
    native_process_exit: int | None = None
    source_admission_digest = sha256_file(root / ADMISSION_PATH)

    try:
        campaign.docker_config.mkdir(exist_ok=False)
        write_json_once(campaign_dir / "source_admission_before.json", {
            "admissionPath": ADMISSION_PATH,
            "admissionSha256": source_admission_digest,
            "reviewedSourceCommit": admission["reviewedSourceCommit"],
            "headCommit": head,
            "sourcePins": admission["sourcePins"],
            "seedSha256": admission["seedSha256"],
            "workOrderSha256": admission["workOrderSha256"],
        })
        before_fixture = campaign.target_container()
        before_network = campaign.target_network()
        before_users = campaign.volume_users()
        campaign.validate_fixture(before_fixture, before_network, before_users, expected_status="exited")
        campaign.protected_before = campaign.protected_inventory()
        write_json_once(campaign_dir / "fixture_before.json", before_fixture)
        write_json_once(campaign_dir / "network_before.json", before_network)
        write_json_once(campaign_dir / "protected_before.json", campaign.protected_before)
        write_json_once(campaign_dir / "fixture_volume_users_before.json", before_users)

        append_reservation(campaign_dir, "compile", campaign_id)
        binary_absolute = str(binary_path.resolve())
        compile_argv = [str(go_exe), "-C", "backend", "test", "-c", "-o", binary_absolute, "./api/handlers"]
        compile_result = campaign.run_process("compile", compile_argv, minimal_env(go_exe), campaign.remaining())
        go_process_exit = compile_result["exitCode"]
        require(compile_result["failureType"] is None and not compile_result["timedOut"] and go_process_exit == 0, "single offline Go compile failed; campaign stops")
        require(binary_path.is_file(), "compile did not create the new R108 binary")
        binary_digest = sha256_file(binary_path)
        write_json_once(campaign_dir / "binary_manifest.json", {"path": str(binary_path.resolve()), "bytes": binary_path.stat().st_size, "sha256": binary_digest, "preserveAfterCleanup": True})

        # Recheck the exact stopped fixture and protection baseline immediately before start.
        pre_start = campaign.target_container()
        network = campaign.target_network()
        users = campaign.volume_users()
        campaign.validate_fixture(pre_start, network, users, expected_status="exited")
        require(campaign.protected_inventory() == campaign.protected_before, "protected R102/user container inventory changed before fixture start")
        append_reservation(campaign_dir, "fixture_start", campaign_id)
        campaign.started_fixture_identity = {"Id": CONTAINER, "Name": "/" + NAME, "Image": IMAGE}
        campaign.start_attempted = True
        start_output = campaign.docker_command("fixture_start", ["start", CONTAINER], timeout=min(20, campaign.remaining()))
        start_id = start_output.decode("utf-8", errors="replace").strip()
        write_json_once(campaign_dir / "fixture_start.result.json", {"attempted": True, "returnedContainer": start_id, "expectedContainer": CONTAINER, "identityMatched": start_id == CONTAINER})
        require(start_id == CONTAINER, "fixture start identity mismatch")

        port = 0
        probes = 0
        while campaign.remaining() > 2:
            running = campaign.target_container()
            require(running.get("Id") == CONTAINER and running.get("Status") == "running", "exact fixture did not remain running")
            network_running = campaign.target_network()
            users_running = campaign.volume_users()
            campaign.running_fixture_ready_shape(running, network_running, users_running)
            ports = running.get("Ports", {}).get("3306/tcp")
            if isinstance(ports, list) and len(ports) == 1 and isinstance(ports[0].get("HostPort"), str) and ports[0]["HostPort"].isdecimal():
                port = campaign.mysql_port(running)
                probes = campaign.wait_mysql_handshake(port)
                break
            else:
                time.sleep(min(1, campaign.remaining()))
        require(port > 0 and probes > 0, "fixture readiness did not complete")
        write_json_once(campaign_dir / "fixture_ready.json", {"container": CONTAINER, "mysqlHost": "127.0.0.1", "mysqlPort": port, "handshakeProbes": probes})

        append_reservation(campaign_dir, "native_invocation", campaign_id)
        dsn = f"root:{FIXTURE_PASSWORD}@tcp(127.0.0.1:{port})/{DATABASE}?parseTime=true&charset=utf8mb4"
        native_argv = [str(binary_path.resolve()), "-test.run", "^TestR108ReopenDiagnostic$", "-test.v", "-test.count=1"]
        native_result = campaign.run_process("native", native_argv, minimal_env(dsn=dsn), campaign.remaining())
        native_process_exit = native_result["exitCode"]
        require(native_result["failureType"] is None and not native_result["timedOut"], "single native invocation did not exit normally")
        native_classification, diagnostic = verify_diagnostic(native_result["stdout"], native_process_exit)
        write_json_once(campaign_dir / "native_diagnostic.json", diagnostic)
        status = native_classification
        reason = "exact four-action synthetic diagnostic completed"
        if native_classification == "REPRODUCED_503":
            reason = "REOPEN returned HTTP 503 and the persisted case/events and proposal remained unchanged"
    except BaseException as exc:  # cleanup is mandatory after every admitted attempt
        status = "HOLD" if isinstance(exc, Hold) else "FAILED"
        reason = str(exc) if isinstance(exc, Hold) else f"bounded campaign exception: {type(exc).__name__}"
    finally:
        for process in reversed(campaign.owned_processes):
            if process.poll() is None:
                try:
                    termination = campaign.stop_owned(process)
                    for detail in campaign.process_details:
                        if detail.get("pid") == process.pid:
                            detail["ownedChildTreeTermination"] = termination
                except BaseException as exc:
                    cleanup_errors.append(f"owned process cleanup: {type(exc).__name__}")

        if campaign.start_attempted:
            try:
                fixture_cleanup = campaign.stop_fixture()
            except BaseException as exc:
                cleanup_errors.append(f"fixture cleanup: {str(exc) if isinstance(exc, Hold) else type(exc).__name__}")

        try:
            protected_after = campaign.protected_inventory(cleanup=True)
            write_json_once(campaign_dir / "protected_after.json", protected_after)
            protection_unchanged = campaign.protected_before is not None and protected_after == campaign.protected_before
            if protection_unchanged is not True:
                cleanup_errors.append("protected R102/user container inventory changed")
        except BaseException as exc:
            protected_after = None
            protection_unchanged = None
            cleanup_errors.append(f"post-campaign protection inventory: {str(exc) if isinstance(exc, Hold) else type(exc).__name__}")

        try:
            post_admission, _, _, post_head = verify_admission(root, deadline=campaign.deadline)
            post_digest = sha256_file(root / ADMISSION_PATH)
            pins_stable = post_head == head and post_digest == source_admission_digest and post_admission.get("sourcePins") == admission.get("sourcePins")
            write_json_once(campaign_dir / "source_admission_after.json", {
                "admissionSha256": post_digest,
                "reviewedSourceCommit": post_admission["reviewedSourceCommit"],
                "headCommit": post_head,
                "sourcePins": post_admission["sourcePins"],
                "pinsStable": pins_stable,
            })
            if not pins_stable:
                cleanup_errors.append("source or admission pins changed during campaign")
        except BaseException as exc:
            cleanup_errors.append(f"post-campaign source pin verification: {str(exc) if isinstance(exc, Hold) else type(exc).__name__}")

        elapsed = round(time.monotonic() - started_at, 3)
        if elapsed > CAMPAIGN_SECONDS:
            cleanup_errors.append("campaign exceeded the 600-second limit")
            status = "HOLD"
            reason = "campaign exceeded the 600-second limit"
        if cleanup_errors and status in ("NOT_REPRODUCED", "REPRODUCED_503"):
            status = "HOLD"
            reason = "campaign result held because cleanup or protection verification failed"
        try:
            write_json_once(campaign_dir / "cleanup.json", {
                "processHandleCleanupChecked": True,
                "ownedProcessStopAttempted": any(
                (detail.get("ownedChildTreeTermination") or {}).get("attempted") is True
                    for detail in campaign.process_details
                ),
                "ownedProcessStates": campaign.process_details,
                "fixtureStartAttempted": campaign.start_attempted,
                "fixtureCleanup": fixture_cleanup,
                "protectedInventoryUnchanged": protection_unchanged,
                "errors": cleanup_errors,
                "elapsedSeconds": elapsed,
            })
            write_json_once(campaign_dir / "campaign_result.json", {
                "campaignId": campaign_id,
                "trancheId": TRANCHE,
                "status": "PENDING_FINALIZATION",
                "proposedStatus": status,
                "reason": reason,
                "goCompileExitCode": go_process_exit,
                "nativeInvocationExitCode": native_process_exit,
                "nativeClassification": native_classification,
                "actionPosts": len(diagnostic.get("actions", [])) if diagnostic else None,
                "binarySha256": sha256_file(binary_path) if binary_path.is_file() else None,
                "protectedInventoryUnchanged": protection_unchanged,
                "elapsedSeconds": elapsed,
                "cleanupErrors": cleanup_errors,
                "claimLimit": "synthetic current-source diagnostic only; no product/governance/release claim",
            })
            write_json_once(campaign_dir / "files_manifest.json", file_manifest(campaign_dir))
            write_json_once(campaign_dir / "campaign_disposition.json", {
                "campaignId": campaign_id,
                "status": status,
                "reason": reason,
                "evidenceManifest": "files_manifest.json",
            })
        except BaseException as exc:
            cleanup_errors.append(f"append-only evidence finalization: {type(exc).__name__}")
            status = "HOLD"
            reason = "campaign result held because append-only evidence finalization failed"
            try:
                write_json_once(campaign_dir / "finalization_failure.json", {
                    "errorType": type(exc).__name__,
                    "status": status,
                    "reason": reason,
                })
            except BaseException:
                pass

    print(json.dumps({
        "trancheId": TRANCHE,
        "status": status,
        "reason": reason,
        "evidenceDirectory": str(campaign_dir),
        "elapsedSeconds": round(time.monotonic() - started_at, 3),
    }, ensure_ascii=False))
    return 0 if status in ("NOT_REPRODUCED", "REPRODUCED_503") else 1


def main() -> int:
    parser = argparse.ArgumentParser(description="One-shot admitted R108 native diagnosis; default is read-only inspection.")
    parser.add_argument("command", nargs="?", choices=("inspect", "start"), default="inspect")
    parser.add_argument("--project-root", default=".")
    args = parser.parse_args()
    root = Path(args.project_root).resolve()
    require((root / ".git").exists() and (root / SEED_PATH).is_file(), "project root is not the admitted Customer-Care-Monitor-AI checkout")
    if args.command == "inspect":
        admission = root / ADMISSION_PATH
        campaign = root / RUNTIME_PATH / "campaign"
        print(json.dumps({
            "trancheId": TRANCHE,
            "mode": "INSPECT_ONLY",
            "rootAdmissionPresent": admission.is_file(),
            "oneShotCampaignSlotUsed": campaign.exists(),
            "runtimeStarted": False,
        }, ensure_ascii=False))
        return 0
    return start_campaign(root)


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Hold as exc:
        print(json.dumps({"trancheId": TRANCHE, "status": "HOLD", "reason": str(exc)}, ensure_ascii=False))
        raise SystemExit(1)
