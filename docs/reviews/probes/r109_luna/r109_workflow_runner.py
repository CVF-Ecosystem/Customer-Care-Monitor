"""One-shot R109 supervisor; inspect is read-only and start requires Root admission.

The campaign reuses the exact immutable R108 compiled binary and a pinned,
read-only subset of R108's safe Docker/process helpers. It does not compile Go,
read Docker Config.Env or .env files, create a fixture, or own a browser.
"""

from __future__ import annotations

import argparse
import hashlib
import http.client
import importlib.util
import json
import os
import queue
import re
import secrets
import shutil
import socket
import subprocess
import sys
import threading
import time
from pathlib import Path
from typing import Any


TRANCHE = "CCMAI-RUNTIME-109"
WORKER = "/root/r108_browser_luna"
SEED_PATH = "CVF_SESSION/authority/CCMAI-RUNTIME-109.json"
RECORD_PATH = "CVF_SESSION/tranches/CCMAI-RUNTIME-109.json"
STATE_PATH = "CVF_SESSION/ACTIVE_SESSION_STATE.json"
HANDOFF_PATH = "CVF_SESSION/handoffs/AGENT_HANDOFF_R109_HUMAN_UI_2026-10-16.md"
ORDER_PATH = "docs/work_orders/CCMAI_RUNTIME_109_HUMAN_UI_COMPLETION.md"
ADMISSION_PATH = "docs/reviews/probes/r109_root/runtime_admission.json"
RUNTIME_REVIEW_PATH = "docs/reviews/R109_INDEPENDENT_RUNTIME_ADMISSION_REVIEW_2026-10-16.md"
SOURCE_REVIEW_PATH = "docs/reviews/R109_INDEPENDENT_UI_SOURCE_REVIEW_2026-10-16.md"
VALIDATION_ROOT = "docs/reviews/probes/r109_root/validation_1"
SOURCE_PINS_PATH = f"{VALIDATION_ROOT}/source_pins.json"
BINARY_REUSE_PATH = "docs/reviews/probes/r109_root/binary_reuse_verification.json"
FOCUSED_TESTS_PATH = f"{VALIDATION_ROOT}/focused_tests.json"
FORCED_TYPECHECK_PATH = f"{VALIDATION_ROOT}/forced_typecheck.json"
RUNTIME_PATH = ".cvf/runtime/r109_workflow"
R108_HELPER_PATH = "docs/reviews/probes/r108_luna/r108_native_runner.py"
R108_HELPER_SHA256 = "899afa92934dede2d99d28d591bc4232d29a52f764d10f8420588e8e940fb138"
RUNNER_PATH = "docs/reviews/probes/r109_luna/r109_workflow_runner.py"
VITE_TEMPLATE_PATH = "docs/reviews/probes/r109_luna/r109_vite_config.mjs.template"

UI_SOURCE_PATHS = (
    "frontend/src/views/InterventionCases.vue",
    "frontend/src/utils/interventionPresentation.ts",
    "frontend/src/i18n/vi.ts",
    "frontend/src/i18n/en.ts",
    "frontend/src/__tests__/intervention-ui.test.ts",
    "frontend/src/__tests__/intervention-presentation.test.ts",
)
VALIDATION_INPUT_PATHS = (
    "frontend/src/__tests__/intervention-api.test.ts",
    "docs/reviews/probes/r109_root/focused-test.config.mjs",
)
PINNED_SOURCE_PATHS = (*UI_SOURCE_PATHS, *VALIDATION_INPUT_PATHS, RUNNER_PATH, VITE_TEMPLATE_PATH, R108_HELPER_PATH)
EVIDENCE_PATHS = (
    SOURCE_REVIEW_PATH,
    RUNTIME_REVIEW_PATH,
    SOURCE_PINS_PATH,
    BINARY_REUSE_PATH,
    FOCUSED_TESTS_PATH,
    f"{VALIDATION_ROOT}/focused_tests.stdout.txt",
    f"{VALIDATION_ROOT}/focused_tests.stderr.txt",
    FORCED_TYPECHECK_PATH,
    f"{VALIDATION_ROOT}/forced_typecheck.stdout.txt",
    f"{VALIDATION_ROOT}/forced_typecheck.stderr.txt",
)

GO_COMPILED_BINARY = ".cvf/runtime/r108_native/campaign/r108_handlers.test.exe"
NODE_EXE = "C:/Program Files/nodejs/node.exe"
NODE_SHA256 = "3602f2bb1a10f2cbab4c36886218a33c1ab3db87290e73b033c46c77147d0237"
DOCKER_EXE = "C:/Users/tiennm/AppData/Local/Programs/DockerDesktop/resources/bin/docker.exe"
DOCKER_SHA256 = "6c699720bb9523dedb9eb5d22a49951ea9210dbdb1314a3554029eee9be76896"
DOCKER_HOST = "npipe:////./pipe/dockerDesktopLinuxEngine"
COMPILED_BINARY_SHA256 = "cb73ee13e0369cca75ea276cc668f30c3ced42cead5f62c56d68f3c1eaa66f93"
COMPILED_BINARY_BYTES = 58031616

CAMPAIGN_SECONDS = 2100
CLEANUP_RESERVE_SECONDS = 75
MYSQL_READY_SECONDS = 90
BACKEND_READY_SECONDS = 60
VITE_READY_SECONDS = 45
VITE_PORT = 3011
HEALTH_PATH = "/__r109_health"
UI_POST_LIMIT = 14
UI_POST_LEDGER = "ui_post_submissions.jsonl"
GIN_POST_LEDGER = "gin_post_observations.jsonl"

HEX40 = re.compile(r"[0-9a-f]{40}\Z")
HEX64 = re.compile(r"[0-9a-f]{64}\Z")
UUID = r"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"
READY = re.compile(
    r"^R101_BROWSER_READY (http://127\.0\.0\.1:[1-9][0-9]{0,4}) "
    rf"tenant=({UUID}) case=({UUID}) reader=({UUID})$"
)
OWNER = re.compile(rf"^R101_BROWSER_OWNER ({UUID}@example\.invalid)$")
GIN_POST = re.compile(r'^\[GIN\].*\|\s*POST\s+"')

# cvf-allow-secret-fixture: exact synthetic R101 application password from the
# committed fixture source; used only to redact diagnostic output if echoed.
SYNTHETIC_BROWSER_PASSWORD = "R101SyntheticOnly!123"


class Hold(RuntimeError):
    """A prerequisite or bounded campaign check failed; no retry is allowed."""


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


def write_json_once(path: Path, value: Any) -> None:
    data = (json.dumps(value, ensure_ascii=False, sort_keys=True, indent=2) + "\n").encode("utf-8")
    with path.open("xb") as stream:
        stream.write(data)
        stream.flush()
        os.fsync(stream.fileno())


def append_json_line(path: Path, value: Any) -> None:
    data = (json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":")) + "\n").encode("utf-8")
    with path.open("ab") as stream:
        stream.write(data)
        stream.flush()
        os.fsync(stream.fileno())


def append_reservation(directory: Path, effect: str, campaign_id: str) -> None:
    write_json_once(directory / f"{effect}.reservation.json", {
        "campaignId": campaign_id,
        "effect": effect,
        "state": "RESERVED",
        "replayAllowed": False,
    })


def normalize_text(data: bytes) -> bytes:
    require(b"\r" not in data.replace(b"\r\n", b""), "bare CR in governed text")
    data.decode("utf-8", errors="strict")
    return data.replace(b"\r\n", b"\n")


def load_r108_helpers(path: Path) -> Any:
    spec = importlib.util.spec_from_file_location("r109_pinned_r108_helpers", path)
    require(spec is not None and spec.loader is not None, "pinned R108 helper module cannot be loaded")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def run_git(root: Path, helper: Any, *args: str, deadline: float | None = None) -> bytes:
    result = helper.git(root, *args, deadline=deadline)
    require(result.returncode == 0, "bounded Git source-history inspection failed")
    return result.stdout


def git_text(root: Path, helper: Any, *args: str, deadline: float | None = None) -> str:
    return run_git(root, helper, *args, deadline=deadline).decode("utf-8", errors="strict").strip()


def verify_committed_file(root: Path, helper: Any, relative: str, expected_hash: str, *, commit: str = "HEAD", deadline: float | None = None) -> None:
    require(bool(HEX64.fullmatch(expected_hash)), f"source pin malformed: {relative}")
    path = root / relative
    require(path.is_file() and not path.is_symlink(), f"source input missing or linked: {relative}")
    actual = sha256_file(path)
    require(actual == expected_hash, f"workspace source pin mismatch: {relative}")
    blob = run_git(root, helper, "show", f"{commit}:{relative}", deadline=deadline)
    require(blob == path.read_bytes(), f"committed bytes differ from workspace: {relative}")


def verify_admission(root: Path, *, deadline: float | None = None) -> tuple[dict[str, Any], Any, Path, Path, Path, str]:
    """Verify committed R109 source/runtime evidence before any side effect."""
    require((root / ".git").exists() and (root / ".cvf/manifest.json").is_file(), "project root is not the admitted checkout")
    admission_path = root / ADMISSION_PATH
    require(admission_path.is_file() and not admission_path.is_symlink(), "committed Root runtime admission is missing")
    helper_path = root / R108_HELPER_PATH
    require(helper_path.is_file() and not helper_path.is_symlink(), "pinned R108 helper is missing")
    require(sha256_file(helper_path) == R108_HELPER_SHA256, "R108 helper bytes differ from independent review")
    helper = load_r108_helpers(helper_path)

    admission = read_json(admission_path)
    admission_blob = run_git(root, helper, "show", f"HEAD:{ADMISSION_PATH}", deadline=deadline)
    require(admission_blob == admission_path.read_bytes(), "runtime admission must be committed")
    admission_status = run_git(root, helper, "status", "--porcelain", "--untracked-files=all", "--", ADMISSION_PATH, deadline=deadline)
    require(not admission_status.strip(), "runtime admission worktree is not clean")
    require(admission.get("schemaVersion") == "1.0", "runtime admission schema mismatch")
    require(admission.get("trancheId") == TRANCHE and admission.get("decision") == "RUNTIME_ADMITTED", "Root runtime admission is not active")
    require(admission.get("reviewer") == "Codex /root", "independent Root runtime reviewer mismatch")

    seed_path = root / SEED_PATH
    record_path = root / RECORD_PATH
    order_path = root / ORDER_PATH
    state_path = root / STATE_PATH
    handoff_path = root / HANDOFF_PATH
    for path in (seed_path, record_path, order_path, state_path, handoff_path):
        require(path.is_file() and not path.is_symlink(), f"required authority file missing: {path.name}")
    seed = read_json(seed_path)
    record = read_json(record_path)
    state = read_json(state_path)
    require(seed.get("trancheId") == TRANCHE and record.get("trancheId") == TRANCHE, "R109 tranche identity mismatch")
    require(seed.get("worker") == WORKER and record.get("worker") == WORKER, "R109 worker binding mismatch")
    require(seed.get("riskCeiling") == "R2" and record.get("riskCeiling") == "R2", "R109 risk ceiling mismatch")
    require(record.get("phase") == "BUILD" and record.get("status") == "BUILD", "R109 record is not in BUILD")
    require(record.get("workOrderStatus") == "ACTIVE" and record.get("workOrderStatusGrammar") == "cvf.workOrderStatusGrammar@1.0.0", "R109 order status binding mismatch")
    require(record.get("workOrder") == ORDER_PATH, "R109 order path binding mismatch")
    require(state.get("activeTranche") == TRANCHE and state.get("activePhase") == "BUILD" and state.get("currentMode") == "BUILD", "R109 continuity is not in BUILD")
    require(state.get("activeHandoff") == HANDOFF_PATH, "R109 active handoff mismatch")
    require(seed.get("browserOperator") == "Codex /root" and seed.get("browserTool") == "mcp__cua_repl" and seed.get("browserSurface") == "iab", "Root connected-browser ownership mismatch")

    budget = seed.get("executionBudget")
    require(isinstance(budget, dict), "R109 execution budget missing")
    expected_budget = {
        "goCompiles": 0,
        "frontendTestInvocations": 2,
        "frontendTypecheckInvocations": 2,
        "existingFixtureStarts": 1,
        "nativeInvocations": 1,
        "nativeActionPosts": 4,
        "backendListenerLaunches": 1,
        "viteLaunches": 1,
        "productTabs": 3,
        "uiPostSubmissionsIncludingAuth": 14,
        "campaignSeconds": CAMPAIGN_SECONDS,
        "runtimeRetries": 0,
        "newFixtures": 0,
        "providerCalls": 0,
        "mockCalls": 0,
    }
    require({key: budget.get(key) for key in expected_budget} == expected_budget, "R109 effect budget drift")

    base_commit = record.get("baseCommit")
    require(isinstance(base_commit, str) and HEX40.fullmatch(base_commit) is not None, "R109 base commit missing")
    head = git_text(root, helper, "rev-parse", "HEAD", deadline=deadline)
    require(run_git(root, helper, "merge-base", "--is-ancestor", base_commit, "HEAD", deadline=deadline) == b"", "R109 base commit is not an ancestor")
    seed_blob = run_git(root, helper, "show", f"{base_commit}:{SEED_PATH}", deadline=deadline)
    require(normalize_text(seed_blob) == normalize_text(seed_path.read_bytes()), "R109 dispatcher seed differs from its base commit")
    seed_adds = git_text(root, helper, "log", "--diff-filter=A", "--format=%H", "--reverse", "--", SEED_PATH, deadline=deadline).splitlines()
    require(bool(seed_adds) and seed_adds[0] == base_commit, "R109 seed base is not its first committed content")
    order_text = order_path.read_text(encoding="utf-8")
    require(re.search(r"(?m)^Status:\s*ACTIVE\s*$", order_text) is not None, "R109 formal order is not ACTIVE")

    admission_source_commit = admission.get("reviewedSourceCommit")
    require(isinstance(admission_source_commit, str) and HEX40.fullmatch(admission_source_commit) is not None, "reviewed source commit is missing")
    require(run_git(root, helper, "merge-base", "--is-ancestor", admission_source_commit, "HEAD", deadline=deadline) == b"", "reviewed source commit is not an ancestor")
    source_pins = admission.get("sourcePins")
    require(isinstance(source_pins, dict) and set(source_pins) == set(PINNED_SOURCE_PATHS), "runtime admission source path set mismatch")
    for relative, expected in source_pins.items():
        verify_committed_file(root, helper, relative, expected, commit="HEAD", deadline=deadline)
        source_blob = run_git(root, helper, "show", f"{admission_source_commit}:{relative}", deadline=deadline)
        require(source_blob == (root / relative).read_bytes(), f"source differs from reviewed commit: {relative}")
    source_status = run_git(root, helper, "status", "--porcelain", "--untracked-files=all", "--", *PINNED_SOURCE_PATHS, deadline=deadline)
    require(not source_status.strip(), "reviewed UI/supervisor sources are not clean")

    evidence_pins = admission.get("evidencePins")
    require(isinstance(evidence_pins, dict) and set(evidence_pins) == set(EVIDENCE_PATHS), "runtime admission evidence path set mismatch")
    for relative, expected in evidence_pins.items():
        verify_committed_file(root, helper, relative, expected, commit="HEAD", deadline=deadline)
    require(admission.get("seedSha256") == sha256_file(seed_path), "runtime admission seed hash mismatch")
    require(admission.get("workOrderSha256") == sha256_file(order_path), "runtime admission work-order hash mismatch")

    source_review = (root / SOURCE_REVIEW_PATH).read_text(encoding="utf-8")
    runtime_review = (root / RUNTIME_REVIEW_PATH).read_text(encoding="utf-8")
    require("Status: REVIEW_PASS" in source_review and "Codex /root" in source_review, "independent UI source review not accepted")
    require("RUNTIME_ADMITTED" in runtime_review and "Codex /root" in runtime_review, "independent runtime review not accepted")
    tests_receipt = read_json(root / FOCUSED_TESTS_PATH)
    typecheck_receipt = read_json(root / FORCED_TYPECHECK_PATH)
    require(tests_receipt.get("exitCode") == 0 and typecheck_receipt.get("exitCode") == 0, "focused UI validation did not pass")
    require(tests_receipt.get("nodeSha256") == NODE_SHA256 and typecheck_receipt.get("nodeSha256") == NODE_SHA256, "focused validation Node pin mismatch")
    source_pin_receipt = read_json(root / SOURCE_PINS_PATH)
    require(isinstance(source_pin_receipt, dict), "source pin validation receipt malformed")
    for relative in (*UI_SOURCE_PATHS, *VALIDATION_INPUT_PATHS):
        expected = source_pin_receipt.get(relative)
        require(isinstance(expected, str) and sha256_file(root / relative) == expected, f"source validation pin mismatch: {relative}")
    binary_receipt = read_json(root / BINARY_REUSE_PATH)
    require(binary_receipt.get("sha256") == COMPILED_BINARY_SHA256 and binary_receipt.get("bytes") == COMPILED_BINARY_BYTES, "binary reuse receipt mismatch")
    require(binary_receipt.get("backendDiff") == "" and binary_receipt.get("backendWorktree") == "", "binary receipt reports a backend delta")

    compiled = seed.get("compiledBackend")
    require(isinstance(compiled, dict), "immutable compiled backend binding missing")
    binary_relative = str(compiled.get("path", "")).replace("\\", "/")
    binary_path = root / binary_relative
    require(binary_relative == GO_COMPILED_BINARY and binary_path.is_file() and not binary_path.is_symlink(), "R108 compiled binary path mismatch")
    require(binary_path.stat().st_size == COMPILED_BINARY_BYTES and sha256_file(binary_path) == COMPILED_BINARY_SHA256 == compiled.get("sha256"), "R108 compiled binary bytes mismatch")
    require(compiled.get("bytes") == COMPILED_BINARY_BYTES and compiled.get("requireNoBackendDelta") is True, "compiled backend source binding mismatch")
    compiled_source = compiled.get("reviewedSourceCommit")
    require(isinstance(compiled_source, str) and HEX40.fullmatch(compiled_source) is not None, "compiled backend source commit missing")
    require(run_git(root, helper, "merge-base", "--is-ancestor", compiled_source, "HEAD", deadline=deadline) == b"", "compiled backend source is not an ancestor")
    backend_diff = helper.git(root, "diff", "--quiet", compiled_source, "HEAD", "--", "backend", deadline=deadline)
    backend_status = run_git(root, helper, "status", "--porcelain", "--untracked-files=all", "--", "backend", deadline=deadline)
    require(backend_diff.returncode == 0 and not backend_status.strip(), "backend differs from immutable binary source")
    frontend_diff = helper.git(root, "diff", "--quiet", admission_source_commit, "HEAD", "--", "frontend", deadline=deadline)
    frontend_status = run_git(root, helper, "status", "--porcelain", "--untracked-files=all", "--", "frontend", deadline=deadline)
    require(frontend_diff.returncode == 0 and not frontend_status.strip(), "frontend differs from the reviewed source commit")

    node = Path(str(seed.get("nodeExe", ""))).resolve()
    require(node.as_posix().lower() == NODE_EXE.lower() and node.is_file() and sha256_file(node) == NODE_SHA256 == seed.get("nodeSha256"), "pinned Node executable mismatch")
    docker = Path(str(seed.get("dockerExe", ""))).resolve()
    require(docker.as_posix().lower() == DOCKER_EXE.lower() and docker.is_file(), "pinned Docker executable path mismatch")
    require(sha256_file(docker) == DOCKER_SHA256 == seed.get("dockerSha256"), "pinned Docker executable bytes mismatch")
    require(seed.get("dockerHost") == DOCKER_HOST, "remote or non-pinned Docker host rejected")

    fixture = seed.get("fixture")
    require(isinstance(fixture, dict), "existing fixture binding missing")
    require(fixture.get("container") == helper.CONTAINER and fixture.get("image") == helper.IMAGE, "existing fixture identity mismatch")
    require(fixture.get("volume") == helper.VOLUME and fixture.get("network") == helper.NETWORK, "existing fixture storage/network mismatch")
    require(fixture.get("label") == helper.LABEL_VALUE and fixture.get("db") == helper.DATABASE, "existing fixture label/database mismatch")
    require(admission.get("helperSha256") == R108_HELPER_SHA256, "admission R108 helper hash mismatch")
    return admission, helper, docker, node, binary_path, head


def minimal_env(extra: dict[str, str] | None = None, node: Path | None = None) -> dict[str, str]:
    """Construct a clean environment; never inherit provider or application secrets."""
    allowed = ("SystemRoot", "WINDIR", "TEMP", "TMP")
    env = {key: os.environ[key] for key in allowed if key in os.environ}
    system_root = env.get("SystemRoot") or env.get("WINDIR")
    path_parts = []
    if node is not None:
        path_parts.append(str(node.parent))
    if system_root:
        path_parts.append(str(Path(system_root) / "System32"))
    if path_parts:
        env["PATH"] = ";".join(path_parts)
    if extra:
        env.update(extra)
    return env


def redact(data: bytes, helper: Any) -> bytes:
    safe = helper.redact_bytes(data)
    return safe.replace(SYNTHETIC_BROWSER_PASSWORD.encode("utf-8"), b"<synthetic-browser-password-redacted>")


def render_vite_config(template: str, frontend: Path, campaign_directory: Path, api_origin: str, nonce: str) -> str:
    api_match = re.fullmatch(r"http://127\.0\.0\.1:([1-9][0-9]{0,4})", api_origin)
    require(api_match is not None and 1 <= int(api_match.group(1)) <= 65535, "API origin must be loopback HTTP")
    require(re.fullmatch(r"[0-9a-f]{32}", nonce) is not None, "health nonce malformed")
    markers = {
        "__R109_FRONTEND_ROOT__": json.dumps(str(frontend.resolve()).replace("\\", "/")),
        "__R109_CAMPAIGN_DIRECTORY__": json.dumps(str(campaign_directory.resolve()).replace("\\", "/")),
        "__R109_API_ORIGIN__": json.dumps(api_origin),
        "__R109_NONCE__": json.dumps(nonce),
    }
    rendered = template
    for marker, value in markers.items():
        require(rendered.count("'" + marker + "'") == 1, f"Vite config marker mismatch: {marker}")
        rendered = rendered.replace("'" + marker + "'", value)
    require("envDir: false" in rendered and "strictPort: true" in rendered and "127.0.0.1" in rendered and "ui_post_submissions.jsonl" in rendered, "Vite template safety settings missing")
    return rendered


def parse_backend_ready(lines: list[str]) -> dict[str, str]:
    ready = [READY.fullmatch(line.strip()) for line in lines]
    owners = [OWNER.fullmatch(line.strip()) for line in lines]
    ready = [match for match in ready if match is not None]
    owners = [match for match in owners if match is not None]
    require(len(ready) == 1 and len(owners) == 1, "backend readiness marker is missing or ambiguous")
    api_origin, tenant, case_id, reader = ready[0].groups()
    api_port = int(api_origin.rsplit(":", 1)[1])
    require(1 <= api_port <= 65535, "backend API port is malformed")
    return {"apiOrigin": api_origin, "tenant": tenant, "case": case_id, "reader": reader, "owner": owners[0].group(1)}


class PipeCapture:
    """Append complete redacted process output and publish bounded readiness lines."""

    def __init__(self, stream: Any, path: Path, helper: Any, events: queue.Queue[str] | None = None, line_callback: Any | None = None):
        self.stream = stream
        self.path = path
        self.helper = helper
        self.events = events
        self.line_callback = line_callback
        self.error: str | None = None
        self.thread = threading.Thread(target=self._read, daemon=True)

    def start(self) -> None:
        self.thread.start()

    def _publish_line(self, line: bytes) -> None:
        safe_text = redact(line, self.helper).decode("utf-8", errors="replace").rstrip("\r\n")
        if self.events is not None:
            try:
                self.events.put_nowait(safe_text)
            except queue.Full:
                pass
        if self.line_callback is not None:
            self.line_callback(safe_text)

    def _read(self) -> None:
        digest = hashlib.sha256()
        raw_bytes = 0
        safe_bytes = 0
        line_pending = bytearray()
        problem: str | None = None
        try:
            with self.path.open("xb") as output:
                while True:
                    chunk = self.stream.read1(8192) if hasattr(self.stream, "read1") else self.stream.read(8192)
                    if not chunk:
                        break
                    raw_bytes += len(chunk)
                    line_pending.extend(chunk)
                    while b"\n" in line_pending:
                        boundary = line_pending.index(b"\n") + 1
                        raw_line = bytes(line_pending[:boundary])
                        del line_pending[:boundary]
                        safe = redact(raw_line, self.helper)
                        output.write(safe)
                        output.flush()
                        digest.update(safe)
                        safe_bytes += len(safe)
                        self._publish_line(raw_line)
                    if len(line_pending) > 1024 * 1024:
                        raise RuntimeError("output line exceeded the bounded redaction buffer")
                if line_pending:
                    raw_line = bytes(line_pending)
                    safe = redact(raw_line, self.helper)
                    output.write(safe)
                    output.flush()
                    digest.update(safe)
                    safe_bytes += len(safe)
                    self._publish_line(raw_line)
        except BaseException as exc:
            problem = type(exc).__name__
            self.error = problem
        try:
            write_json_once(self.path.with_suffix(self.path.suffix + ".meta.json"), {
                "rawBytesObserved": raw_bytes,
                "redactedBytesRetained": safe_bytes,
                "redactedSha256": digest.hexdigest(),
                "readerError": problem,
                "outputRedaction": "exact synthetic R101 fixture passwords, R108 DSN pattern and bearer-token pattern",
            })
        except BaseException as exc:
            self.error = type(exc).__name__


class Campaign:
    """R109 wrapper around the reviewed R108 read-only Docker/process helpers."""

    def __init__(self, root: Path, directory: Path, helper: Any, docker: Path, campaign_id: str):
        self.helper = helper
        self.base = helper.Campaign(root, directory, docker, Path(sys.executable), campaign_id)
        self.root = root
        self.directory = directory
        self.campaign_id = campaign_id
        self.readers: list[PipeCapture] = []
        self.long_processes: dict[int, dict[str, Any]] = {}
        self.deadline = 0.0
        self.effect_deadline = 0.0

    def __getattr__(self, name: str) -> Any:
        return getattr(self.base, name)

    def remaining(self, cleanup: bool = False) -> float:
        limit = self.deadline if cleanup else self.effect_deadline
        return max(0.0, limit - time.monotonic())

    def stop_requested(self) -> bool:
        return (self.directory / "stop.request").is_file()

    def stop_fixture(self) -> dict[str, Any]:
        current = self.base.target_container(cleanup=True)
        status = current.get("Status")
        if status == "running":
            network = self.base.target_network(cleanup=True)
            users = self.base.volume_users(cleanup=True)
            self.base.running_fixture_ready_shape(current, network, users)
            output = self.base.docker_command(
                "fixture_stop",
                ["stop", "--timeout", "10", self.helper.CONTAINER],
                timeout=min(15, max(1, self.remaining(cleanup=True))),
                cleanup=True,
            )
            write_json_once(self.directory / "fixture_stop_command.json", {
                "exitCode": 0,
                "stdoutBytes": len(output),
                "stdoutSha256": sha256_bytes(redact(output, self.helper)),
                "identityCheckedBy": "structured post-stop container/network/volume inspect",
            })
        elif status != "exited":
            raise Hold("owned fixture is neither running nor stopped during cleanup")

        final = self.base.target_container(cleanup=True)
        network_final = self.base.target_network(cleanup=True)
        users_final = self.base.volume_users(cleanup=True)
        self.base.validate_fixture(final, network_final, users_final, expected_status="exited")
        require(final.get("Id") == self.helper.CONTAINER and final.get("Name") == "/" + self.helper.NAME and final.get("Image") == self.helper.IMAGE, "structured final fixture identity mismatch")
        receipt = {
            "stopCommandNeeded": status == "running",
            "finalContainerId": final.get("Id"),
            "finalStatus": final.get("Status"),
            "networkId": network_final.get("Id"),
            "volumeUsers": users_final,
            "identityVerifiedFromStructuredInspection": True,
        }
        write_json_once(self.directory / "fixture_final_state.json", receipt)
        return receipt

    def wait_mysql_handshake(self, port: int) -> int:
        probes = 0
        readiness_dir = self.directory / "readiness"
        readiness_dir.mkdir(exist_ok=True)
        self.mysql_error_number: int | None = None
        self.mysql_readiness_probes = 0
        mysql_deadline = min(time.monotonic() + MYSQL_READY_SECONDS, self.effect_deadline)
        while time.monotonic() < mysql_deadline:
            if self.stop_requested():
                raise Hold("Root requested cooperative stop during fixture readiness")
            probes += 1
            started = time.monotonic()
            state = "NOT_READY"
            classification = "NO_HANDSHAKE"
            mysql_error_number: int | None = None
            try:
                with socket.create_connection(("127.0.0.1", port), timeout=min(1.5, self.remaining())) as connection:
                    connection.settimeout(min(1.5, self.remaining()))
                    header = self._read_exact(connection, 4)
                    if header is not None:
                        packet_size = int.from_bytes(header[:3], "little")
                        sequence = header[3]
                        if not 0 < packet_size <= 65535:
                            payload = None
                            state = "FAILED"
                            classification = "INVALID_MYSQL_PACKET_LENGTH"
                        else:
                            payload = self._read_exact(connection, packet_size)
                        if payload is not None and sequence == 0 and payload[:1] == b"\xff":
                            if len(payload) < 9 or payload[3:4] != b"#":
                                state = "FAILED"
                                classification = "MALFORMED_SERVER_ERROR_PACKET"
                            else:
                                state = "FAILED"
                                classification = "MYSQL_SERVER_ERROR"
                                mysql_error_number = int.from_bytes(payload[1:3], "little")
                        elif payload is not None and sequence == 0 and len(payload) >= 20 and payload[:1] == b"\x0a" and payload.find(b"\x00", 1, 128) >= 2:
                            state = "READY"
                            classification = "MYSQL_PROTOCOL_10_GREETING"
                        elif payload is None and state != "FAILED":
                            classification = "EARLY_EOF_OR_SHORT_READ"
                        elif payload is not None:
                            state = "FAILED"
                            classification = "UNEXPECTED_MYSQL_PACKET"
                    else:
                        classification = "EARLY_EOF_OR_SHORT_READ"
            except (OSError, TimeoutError, ConnectionError):
                classification = "CONNECT_OR_READ_PENDING"
            self.mysql_readiness_probes = probes
            if mysql_error_number is not None:
                self.mysql_error_number = mysql_error_number
            write_json_once(readiness_dir / f"mysql_probe_{probes:04d}.json", {
                "probe": probes,
                "state": state,
                "classification": classification,
                "safeServerErrorNumber": mysql_error_number,
                "elapsedSeconds": round(time.monotonic() - started, 3),
            })
            if state == "READY":
                write_json_once(readiness_dir / "mysql_ready.json", {"port": port, "probeCount": probes, "result": "READY"})
                return probes
            if state == "FAILED":
                raise Hold("fixture MySQL returned a complete server error or unexpected packet during startup")
            time.sleep(min(0.75, max(0.0, mysql_deadline - time.monotonic())))
        raise Hold("fixture MySQL remained NOT_READY until the 90-second startup deadline")

    @staticmethod
    def _read_exact(connection: socket.socket, size: int) -> bytes | None:
        data = bytearray()
        while len(data) < size:
            part = connection.recv(size - len(data))
            if not part:
                return None
            data.extend(part)
        return bytes(data)

    def launch_service(self, label: str, argv: list[str], env: dict[str, str], *, stdout_path: Path, stderr_path: Path, readiness_queue: queue.Queue[str] | None = None, stdout_line_callback: Any | None = None) -> subprocess.Popen[bytes]:
        require(self.remaining() > 1, "campaign effect deadline reached before process launch")
        intent_path = self.directory / f"{label}_launch_intent.json"
        write_json_once(intent_path, {
            "label": label,
            "argv": [redact(item.encode("utf-8"), self.helper).decode("utf-8", errors="replace") for item in argv],
            "environmentNames": sorted(env),
            "environmentValuesRecorded": False,
            "startedAtUnix": time.time(),
        })
        process: subprocess.Popen[bytes] | None = None
        creation_flags = getattr(subprocess, "CREATE_NEW_PROCESS_GROUP", 0) | getattr(subprocess, "CREATE_NO_WINDOW", 0)
        try:
            process = subprocess.Popen(
                argv,
                cwd=self.root,
                env=env,
                stdin=subprocess.DEVNULL,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                creationflags=creation_flags,
            )
        except OSError as exc:
            write_json_once(self.directory / f"{label}_launch_result.json", {
                "pid": None,
                "failureType": type(exc).__name__,
                "exitCode": None,
            })
            raise Hold(f"owned {label} process could not start") from exc

        # Register the exact Popen handle before any result receipt can fail.
        self.base.owned_processes.append(process)
        detail = {"label": label, "pid": process.pid, "exitCode": None, "timedOut": False, "failureType": None, "ownedChildTreeTermination": None}
        self.base.process_details.append(detail)
        self.long_processes[process.pid] = detail
        try:
            write_json_once(self.directory / f"{label}_launch_result.json", {
                "pid": process.pid,
                "failureType": None,
                "exitCode": None,
                "executableSha256": sha256_file(Path(argv[0])),
            })
            require(process.stdout is not None and process.stderr is not None, f"{label} output pipes unavailable")
            out_reader = PipeCapture(process.stdout, stdout_path, self.helper, readiness_queue, stdout_line_callback)
            err_reader = PipeCapture(process.stderr, stderr_path, self.helper)
            self.readers.extend((out_reader, err_reader))
            out_reader.start()
            err_reader.start()
        except BaseException as exc:
            termination = self.base.stop_owned(process)
            detail["ownedChildTreeTermination"] = termination
            detail["exitCode"] = process.poll()
            raise Hold(f"{label} launch receipt or output capture failed; owned process stopped") from exc
        return process

    def finish_processes(self, cleanup_errors: list[str]) -> None:
        for process in reversed(self.base.owned_processes):
            if process.poll() is None:
                try:
                    termination = self.base.stop_owned(process)
                    detail = self.long_processes.get(process.pid)
                    if detail is not None:
                        detail["ownedChildTreeTermination"] = termination
                except BaseException as exc:
                    cleanup_errors.append(f"owned process stop: {type(exc).__name__}")
            detail = self.long_processes.get(process.pid)
            if detail is not None:
                detail["exitCode"] = process.poll()
        for reader in self.readers:
            reader.thread.join(timeout=min(2.0, self.remaining(cleanup=True)))
            if reader.thread.is_alive():
                cleanup_errors.append("owned process output reader did not finish")
            elif reader.error:
                cleanup_errors.append("owned process output capture failed: " + reader.error)
        if any(process.poll() is None for process in self.base.owned_processes):
            cleanup_errors.append("owned child process still running after handle-scoped cleanup")


def await_backend_ready(campaign: Campaign, process: subprocess.Popen[bytes], events: queue.Queue[str]) -> dict[str, str]:
    deadline = min(time.monotonic() + BACKEND_READY_SECONDS, campaign.effect_deadline)
    seen: list[str] = []
    probe = 0
    while time.monotonic() < deadline:
        if campaign.stop_requested():
            raise Hold("Root requested cooperative stop while backend was starting")
        if process.poll() is not None:
            raise Hold("backend exited before its readiness markers appeared")
        try:
            seen.append(events.get(timeout=min(0.25, max(0.01, deadline - time.monotonic()))))
        except queue.Empty:
            pass
        probe += 1
        ready_lines = [line for line in seen if READY.fullmatch(line.strip())]
        owner_lines = [line for line in seen if OWNER.fullmatch(line.strip())]
        state = "READY" if len(ready_lines) == 1 and len(owner_lines) == 1 else "NOT_READY"
        write_json_once(campaign.directory / f"backend_readiness_{probe:04d}.json", {
            "probe": probe,
            "state": state,
            "readyMarkerCount": len(ready_lines),
            "ownerMarkerCount": len(owner_lines),
        })
        if state == "READY":
            require(process.poll() is None, "backend exited immediately after readiness")
            return parse_backend_ready(seen)
    raise Hold("backend remained NOT_READY until its bounded startup deadline")


def port_available(port: int) -> bool:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        try:
            sock.bind(("127.0.0.1", port))
        except OSError:
            return False
    return True


def wait_vite_health(campaign: Campaign, process: subprocess.Popen[bytes], nonce: str) -> int:
    deadline = min(time.monotonic() + VITE_READY_SECONDS, campaign.effect_deadline)
    probe = 0
    while time.monotonic() < deadline:
        if campaign.stop_requested():
            raise Hold("Root requested cooperative stop while Vite was starting")
        if process.poll() is not None:
            raise Hold("Vite exited before its owned health response")
        probe += 1
        state = "NOT_READY"
        response_status: int | None = None
        response_bytes = 0
        try:
            connection = http.client.HTTPConnection("127.0.0.1", VITE_PORT, timeout=min(1.5, campaign.remaining()))
            connection.request("GET", HEALTH_PATH, headers={"Host": f"127.0.0.1:{VITE_PORT}", "Connection": "close"})
            response = connection.getresponse()
            body = response.read(4096)
            response_status = response.status
            response_bytes = len(body)
            if response.status == 200 and body == nonce.encode("ascii"):
                state = "READY"
            connection.close()
        except (OSError, TimeoutError, http.client.HTTPException):
            state = "NOT_READY"
        write_json_once(campaign.directory / f"vite_readiness_{probe:04d}.json", {
            "probe": probe,
            "state": state,
            "httpStatus": response_status,
            "bodyBytes": response_bytes,
        })
        if state == "READY":
            require(process.poll() is None, "Vite exited after its health response")
            return probe
        time.sleep(min(0.5, campaign.remaining()))
    raise Hold("Vite remained NOT_READY until its bounded startup deadline")


def verify_native(stdout: bytes, exit_code: int | None, helper: Any) -> tuple[str, dict[str, Any]]:
    classification, report = helper.verify_diagnostic(stdout, exit_code)
    require(classification in ("NOT_REPRODUCED", "REPRODUCED_503"), "native diagnostic returned an unsupported classification")
    return classification, report


def file_manifest(directory: Path) -> list[dict[str, Any]]:
    entries = []
    for path in sorted(directory.rglob("*")):
        if path.is_file() and path.name not in ("files_manifest.json", "campaign_disposition.json"):
            entries.append({
                "path": path.relative_to(directory).as_posix(),
                "bytes": path.stat().st_size,
                "sha256": sha256_file(path),
            })
    return entries


def jsonl_rows(path: Path, *, allow_partial: bool = False) -> list[dict[str, Any]]:
    if not path.is_file():
        return []
    data = path.read_bytes()
    if data and not data.endswith(b"\n"):
        if not allow_partial:
            raise Hold(f"append-only POST ledger has an incomplete final row: {path.name}")
        data = data[: data.rfind(b"\n") + 1] if b"\n" in data else b""
    rows = []
    for line in data.splitlines():
        value = json.loads(line.decode("utf-8", errors="strict"))
        require(isinstance(value, dict), f"append-only POST ledger row is invalid: {path.name}")
        rows.append(value)
    return rows


def ui_post_counts(campaign_dir: Path, *, allow_partial: bool = False) -> dict[str, int | bool]:
    submitted = jsonl_rows(campaign_dir / UI_POST_LEDGER, allow_partial=allow_partial)
    gin_observed = jsonl_rows(campaign_dir / GIN_POST_LEDGER, allow_partial=allow_partial)
    forwarded = sum(row.get("state") == "FORWARDED_TO_API" for row in submitted)
    blocked = sum(row.get("state") == "BLOCKED_POST_LIMIT" for row in submitted)
    used = max(len(submitted), len(gin_observed))
    return {
        "submitted": len(submitted),
        "forwarded": forwarded,
        "blockedByLocalHarness": blocked,
        "ginPostObserved": len(gin_observed),
        "budgetCounted": used,
        "remaining": max(0, UI_POST_LIMIT - used),
        "exceeded": used > UI_POST_LIMIT or blocked > 0,
    }


def run_campaign(root: Path, admission: dict[str, Any], helper: Any, docker: Path, node: Path, binary: Path, head: str) -> bool:
    require(Path.cwd().resolve() == root.resolve(), "start must run from the project root")
    require(port_available(VITE_PORT), "Vite loopback port 3011 is already occupied")
    runtime_parent = root / ".cvf/runtime"
    require(runtime_parent.is_dir() and not runtime_parent.is_symlink(), "runtime parent is missing or linked")
    runtime = root / RUNTIME_PATH
    require(not runtime.is_symlink(), "runtime root is a symlink")
    if runtime.exists():
        require(runtime.is_dir() and not any(runtime.iterdir()), "R109 append-only runtime ledger is already used")
    else:
        runtime.mkdir(parents=True, exist_ok=False)
    campaign_dir = runtime / "campaign"
    campaign_dir.mkdir(exist_ok=False)

    started = time.monotonic()
    deadline = started + CAMPAIGN_SECONDS
    effect_deadline = deadline - CLEANUP_RESERVE_SECONDS
    campaign_id = f"r109-{int(time.time())}-{os.getpid()}"
    campaign = Campaign(root, campaign_dir, helper, docker, campaign_id)
    campaign.deadline = deadline
    campaign.effect_deadline = effect_deadline
    campaign.base.deadline = deadline
    campaign.base.effect_deadline = effect_deadline

    status = "HOLD"
    reason = "campaign has no terminal disposition"
    native_classification: str | None = None
    native_report: dict[str, Any] | None = None
    native_exit: int | None = None
    browser_ready: dict[str, str] | None = None
    mysql_port = 0
    fixture_cleanup: dict[str, Any] = {"startAttempted": False, "verified": False}
    protected_before: list[dict[str, Any]] | None = None
    protected_unchanged: bool | None = None
    cleanup_errors: list[str] = []
    gin_post_observed = 0

    def observe_gin_post(line: str) -> None:
        nonlocal gin_post_observed
        if GIN_POST.search(line):
            gin_post_observed += 1
            append_json_line(campaign_dir / GIN_POST_LEDGER, {
                "ordinal": gin_post_observed,
                "source": "Gin access log",
                "method": "POST",
                "routeClass": "backend request",
            })
    source_admission_hash = ""
    binary_hash = ""
    try:
        source_admission_hash = sha256_file(root / ADMISSION_PATH)
        binary_hash = sha256_file(binary)
        campaign.base.docker_config.mkdir(exist_ok=False)
        write_json_once(campaign_dir / "campaign_intent.json", {
            "campaignId": campaign_id,
            "trancheId": TRANCHE,
            "headCommit": head,
            "reviewedSourceCommit": admission["reviewedSourceCommit"],
            "runtimeAdmissionSha256": source_admission_hash,
            "sourcePins": admission["sourcePins"],
            "evidencePins": admission["evidencePins"],
            "seedSha256": admission["seedSha256"],
            "workOrderSha256": admission["workOrderSha256"],
            "helperSha256": R108_HELPER_SHA256,
            "binaryPath": str(binary.relative_to(root)).replace("\\", "/"),
            "binaryBytes": binary.stat().st_size,
            "binarySha256": binary_hash,
            "nodeSha256": NODE_SHA256,
            "dockerSha256": DOCKER_SHA256,
            "startedAtUnix": time.time(),
            "campaignLimitSeconds": CAMPAIGN_SECONDS,
            "cleanupReserveSeconds": CLEANUP_RESERVE_SECONDS,
            "budget": {"goCompiles": 0, "fixtureStarts": 1, "nativeInvocations": 1, "nativeActionPosts": 4, "backendLaunches": 1, "viteLaunches": 1, "runtimeRetries": 0},
            "environment": "SystemRoot/WINDIR/TEMP/TMP plus explicitly declared synthetic fixture values only; no .env or provider settings",
        })
    except BaseException as exc:
        setup_reason = "R109 campaign setup failed before any external effect"
        try:
            write_json_once(campaign_dir / "setup_failure.json", {
                "status": "FAILED",
                "reason": setup_reason,
                "errorType": type(exc).__name__,
                "fixtureStartAttempted": False,
            })
        except BaseException:
            pass
        print(json.dumps({"trancheId": TRANCHE, "status": "FAILED", "reason": setup_reason, "evidenceDirectory": str(campaign_dir)}, ensure_ascii=False))
        return False

    try:
        # Before any effect, retain a safe protected-resource baseline and exact fixture identity.
        fixture_before = campaign.base.target_container()
        network_before = campaign.base.target_network()
        volume_users_before = campaign.base.volume_users()
        campaign.base.validate_fixture(fixture_before, network_before, volume_users_before, expected_status="exited")
        protected_before = campaign.base.protected_inventory()
        campaign.base.protected_before = protected_before
        write_json_once(campaign_dir / "fixture_before.json", fixture_before)
        write_json_once(campaign_dir / "network_before.json", network_before)
        write_json_once(campaign_dir / "fixture_volume_users_before.json", volume_users_before)
        write_json_once(campaign_dir / "protected_before.json", protected_before)

        # Recheck the exact stopped fixture and protected inventory immediately before starting it.
        fixture_prestart = campaign.base.target_container()
        network_prestart = campaign.base.target_network()
        volume_users_prestart = campaign.base.volume_users()
        campaign.base.validate_fixture(fixture_prestart, network_prestart, volume_users_prestart, expected_status="exited")
        require(campaign.base.protected_inventory() == protected_before, "protected user/R102 container inventory changed before fixture start")
        append_reservation(campaign_dir, "fixture_start", campaign_id)
        campaign.base.started_fixture_identity = {"Id": helper.CONTAINER, "Name": "/" + helper.NAME, "Image": helper.IMAGE}
        campaign.base.start_attempted = True  # set before Docker start; clean up ambiguous start outcomes
        start_output = campaign.base.docker_command("fixture_start", ["start", helper.CONTAINER], timeout=min(20, campaign.remaining()))
        start_id = start_output.decode("utf-8", errors="replace").strip()
        write_json_once(campaign_dir / "fixture_start_result.json", {
            "startAttempted": True,
            "returnedId": start_id,
            "expectedId": helper.CONTAINER,
            "identityMatched": start_id == helper.CONTAINER,
        })
        require(start_id == helper.CONTAINER, "exact fixture start returned a different identity")

        probe_count = 0
        while campaign.remaining() > 2:
            if campaign.stop_requested():
                raise Hold("Root requested cooperative stop during fixture startup")
            running = campaign.base.target_container()
            require(running.get("Id") == helper.CONTAINER and running.get("Status") == "running", "exact fixture did not remain running")
            running_network = campaign.base.target_network()
            running_users = campaign.base.volume_users()
            campaign.base.running_fixture_ready_shape(running, running_network, running_users)
            ports = running.get("Ports", {}).get("3306/tcp")
            if isinstance(ports, list) and len(ports) == 1 and isinstance(ports[0].get("HostPort"), str) and ports[0]["HostPort"].isdecimal():
                mysql_port = campaign.base.mysql_port(running)
                probe_count = campaign.wait_mysql_handshake(mysql_port)
                break
            time.sleep(min(0.75, campaign.remaining()))
        require(mysql_port > 0 and probe_count > 0, "fixture did not reach a ready loopback database")
        write_json_once(campaign_dir / "fixture_ready.json", {
            "containerId": helper.CONTAINER,
            "mysqlHost": "127.0.0.1",
            "mysqlPort": mysql_port,
            "readinessProbes": probe_count,
            "state": "READY",
        })

        append_reservation(campaign_dir, "native_invocation", campaign_id)
        dsn = f"root:{helper.FIXTURE_PASSWORD}@tcp(127.0.0.1:{mysql_port})/{helper.DATABASE}?parseTime=true&charset=utf8mb4"
        native_args = [str(binary), "-test.run", "^TestR108ReopenDiagnostic$", "-test.v", "-test.count=1", "-test.timeout=10m"]
        native_result = campaign.base.run_process("native_diagnostic", native_args, helper.minimal_env(dsn=dsn), min(600, campaign.remaining()))
        native_exit = native_result.get("exitCode")
        require(native_result.get("failureType") is None and native_result.get("timedOut") is False, "native diagnostic did not exit normally")
        native_classification, native_report = verify_native(native_result["stdout"], native_exit, helper)
        write_json_once(campaign_dir / "native_diagnostic.json", {
            "classification": native_classification,
            "exitCode": native_exit,
            "actions": native_report.get("actions"),
            "lifecycleAccepted": native_report.get("lifecycle_accepted"),
            "actionCount": len(native_report.get("actions", [])),
            "claim": "current-source synthetic human workflow diagnostic only",
        })
        if native_classification == "REPRODUCED_503":
            status = "REPRODUCED_503"
            reason = "REOPEN returned HTTP 503 and the retained diagnostic proved the persisted state comparison"
        else:
            require(not campaign.stop_requested(), "Root requested stop after native diagnostic")
            # One native backend listener reuses the same fixture and creates fresh synthetic browser data.
            require(port_available(VITE_PORT), "Vite loopback port 3011 is already occupied")
            append_reservation(campaign_dir, "backend_launch", campaign_id)
            backend_env = minimal_env({
                "R101_BROWSER_FIXTURE": "1",
                "R101_MYSQL_DSN": dsn,
            }, node=node)
            events: queue.Queue[str] = queue.Queue(maxsize=512)
            with (campaign_dir / GIN_POST_LEDGER).open("xb") as ledger:
                ledger.flush()
                os.fsync(ledger.fileno())
            backend = campaign.launch_service(
                "backend",
                [str(binary), "-test.run", "^TestR101BrowserFixture$", "-test.v", "-test.timeout=35m"],
                backend_env,
                stdout_path=campaign_dir / "backend.stdout.bin",
                stderr_path=campaign_dir / "backend.stderr.bin",
                readiness_queue=events,
                stdout_line_callback=observe_gin_post,
            )
            backend_ready_at = time.monotonic()
            browser_ready = await_backend_ready(campaign, backend, events)
            write_json_once(campaign_dir / "ready.json", {
                **browser_ready,
                "observedAtUnix": time.time(),
                "startupSeconds": round(time.monotonic() - backend_ready_at, 3),
                "syntheticCredentialsIncluded": False,
            })

            health_nonce = secrets.token_hex(16)
            template_path = root / VITE_TEMPLATE_PATH
            template = template_path.read_text(encoding="utf-8")
            config_text = render_vite_config(template, root / "frontend", campaign_dir, browser_ready["apiOrigin"], health_nonce)
            config_path = campaign_dir / "vite.config.mjs"
            with config_path.open("x", encoding="utf-8", newline="\n") as stream:
                stream.write(config_text)
                stream.flush()
                os.fsync(stream.fileno())
            write_json_once(campaign_dir / "vite_config_receipt.json", {
                "path": config_path.name,
                "sha256": sha256_file(config_path),
                "apiOrigin": browser_ready["apiOrigin"],
                "healthRoute": HEALTH_PATH,
                "envDir": False,
                "ambientEnvironmentDisabled": True,
            })

            require(port_available(VITE_PORT), "Vite loopback port 3011 became occupied before launch")
            vite_cli = root / "frontend/node_modules/vite/bin/vite.js"
            require(vite_cli.is_file() and not vite_cli.is_symlink(), "cached Vite CLI is missing or linked")
            append_reservation(campaign_dir, "vite_launch", campaign_id)
            vite_args = [str(node), str(vite_cli), "--config", str(config_path), "--host", "127.0.0.1", "--port", str(VITE_PORT), "--strictPort"]
            vite_env = minimal_env(node=node)
            with (campaign_dir / UI_POST_LEDGER).open("xb") as ledger:
                ledger.flush()
                os.fsync(ledger.fileno())
            vite = campaign.launch_service(
                "vite",
                vite_args,
                vite_env,
                stdout_path=campaign_dir / "vite.stdout.bin",
                stderr_path=campaign_dir / "vite.stderr.bin",
            )
            vite_probe_count = wait_vite_health(campaign, vite, health_nonce)
            write_json_once(campaign_dir / "running.json", {
                "backendPid": backend.pid,
                "vitePid": vite.pid,
                "browserUrl": f"http://127.0.0.1:{VITE_PORT}/",
                "apiOrigin": browser_ready["apiOrigin"],
                "viteReadinessProbes": vite_probe_count,
                "campaignDeadlineUnix": time.time() + campaign.remaining(),
                "rootBrowserOwner": "Codex /root",
                "workerBrowserRole": "NONE",
            })
            status = "UI_READY_WAITING_ROOT"
            reason = "shared native and supervised UI runtime are ready for Root's connected browser checks"
            while campaign.remaining() > 0:
                if campaign.stop_requested():
                    status = "STOPPED_BY_ROOT"
                    reason = "Root requested cooperative cleanup after its browser observation"
                    break
                require(backend.poll() is None and vite.poll() is None, "owned backend or Vite exited during browser campaign")
                time.sleep(min(0.5, campaign.remaining()))
            else:
                status = "DEADLINE_REACHED"
                reason = "shared 2100-second campaign reached its effect deadline"

    except BaseException as exc:
        status = "HOLD" if isinstance(exc, Hold) else "FAILED"
        reason = str(exc) if isinstance(exc, Hold) else f"bounded campaign exception: {type(exc).__name__}"
    finally:
        campaign.finish_processes(cleanup_errors)
        if campaign.base.start_attempted:
            try:
                fixture_cleanup = campaign.stop_fixture()
                fixture_cleanup["startAttempted"] = True
                fixture_cleanup["verified"] = True
            except BaseException as exc:
                cleanup_errors.append("fixture cleanup: " + (str(exc) if isinstance(exc, Hold) else type(exc).__name__))
                fixture_cleanup = {"startAttempted": True, "verified": False}

        try:
            protected_after = campaign.base.protected_inventory(cleanup=True)
            write_json_once(campaign_dir / "protected_after.json", {
                "rows": protected_after,
                "matchesBefore": protected_before is not None and protected_after == protected_before,
            })
            protected_unchanged = protected_before is not None and protected_after == protected_before
            if protected_unchanged is not True:
                cleanup_errors.append("protected R102/user container inventory changed or was unavailable")
        except BaseException as exc:
            protected_unchanged = None
            cleanup_errors.append("post-campaign protected inventory: " + (str(exc) if isinstance(exc, Hold) else type(exc).__name__))

        try:
            post_counts = ui_post_counts(campaign_dir)
            if post_counts["exceeded"]:
                cleanup_errors.append("UI POST budget was exceeded or a local harness denial occurred")
        except BaseException as exc:
            post_counts = {
                "submitted": 0,
                "forwarded": 0,
                "blockedByLocalHarness": 0,
                "ginPostObserved": gin_post_observed,
                "budgetCounted": gin_post_observed,
                "remaining": max(0, UI_POST_LIMIT - gin_post_observed),
                "exceeded": True,
            }
            cleanup_errors.append("UI POST ledgers could not be read: " + (str(exc) if isinstance(exc, Hold) else type(exc).__name__))

        try:
            current_admission, current_helper, _, _, current_binary, current_head = verify_admission(root, deadline=campaign.deadline)
            pins_stable = (
                current_head == head
                and sha256_file(root / ADMISSION_PATH) == source_admission_hash
                and current_admission.get("sourcePins") == admission.get("sourcePins")
                and sha256_file(root / R108_HELPER_PATH) == R108_HELPER_SHA256
                and sha256_file(current_binary) == binary_hash
            )
            write_json_once(campaign_dir / "source_admission_after.json", {
                "runtimeAdmissionSha256": sha256_file(root / ADMISSION_PATH),
                "headCommit": current_head,
                "sourcePinsStable": pins_stable,
                "binarySha256": sha256_file(current_binary),
            })
            if not pins_stable:
                cleanup_errors.append("source/admission/binary pins changed during campaign")
        except BaseException as exc:
            cleanup_errors.append("post-campaign source verification: " + (str(exc) if isinstance(exc, Hold) else type(exc).__name__))

        elapsed = round(time.monotonic() - started, 3)
        if elapsed > CAMPAIGN_SECONDS:
            cleanup_errors.append("campaign exceeded the 2100-second deadline including cleanup")
            status = "HOLD"
            reason = "campaign exceeded the total deadline"
        if not fixture_cleanup.get("verified"):
            cleanup_errors.append("owned fixture final state was not verified")
        if any(process.poll() is None for process in campaign.base.owned_processes):
            cleanup_errors.append("owned child process remains running")
        if cleanup_errors and status not in ("HOLD", "FAILED"):
            status = "HOLD"
            reason = "campaign held because cleanup or post-campaign verification was incomplete"

        finalization_error: str | None = None
        try:
            write_json_once(campaign_dir / "cleanup.json", {
                "ownedProcessHandleCleanupChecked": True,
                "ownedProcesses": campaign.base.process_details,
                "readerStates": [{"path": reader.path.name, "finished": not reader.thread.is_alive(), "error": reader.error} for reader in campaign.readers],
                "fixtureStartAttempted": campaign.base.start_attempted,
                "fixtureCleanup": fixture_cleanup,
                "protectedInventoryUnchanged": protected_unchanged,
                "uiPostCounts": post_counts,
                "errors": cleanup_errors,
                "elapsedSeconds": elapsed,
            })
            write_json_once(campaign_dir / "campaign_result.json", {
                "campaignId": campaign_id,
                "trancheId": TRANCHE,
                "status": "PENDING_FINALIZATION",
                "proposedStatus": status,
                "reason": reason,
                "nativeClassification": native_classification,
                "nativeExitCode": native_exit,
                "nativeActionCount": len(native_report.get("actions", [])) if native_report else None,
                "nativeLifecycleAccepted": native_report.get("lifecycle_accepted") if native_report else None,
                "mysqlReadinessProbes": getattr(campaign, "mysql_readiness_probes", 0),
                "mysqlServerErrorNumber": getattr(campaign, "mysql_error_number", None),
                "browserReadyObserved": browser_ready is not None,
                "fixtureCleanupVerified": fixture_cleanup.get("verified") is True,
                "protectedInventoryUnchanged": protected_unchanged,
                "uiPostCounts": post_counts,
                "binarySha256": binary_hash,
                "elapsedSeconds": elapsed,
                "cleanupErrors": cleanup_errors,
                "claimLimit": "synthetic current-source workflow and supervised local UI only; no full product/governance/release claim",
            })
            write_json_once(campaign_dir / "files_manifest.json", file_manifest(campaign_dir))
            write_json_once(campaign_dir / "campaign_disposition.json", {
                "campaignId": campaign_id,
                "status": status,
                "reason": reason,
                "uiPostCounts": post_counts,
                "evidenceManifest": "files_manifest.json",
                "evidenceManifestSha256": sha256_file(campaign_dir / "files_manifest.json"),
            })
        except BaseException as exc:
            status = "HOLD"
            reason = "campaign held because append-only evidence finalization failed"
            finalization_error = type(exc).__name__
            try:
                write_json_once(campaign_dir / "finalization_failure.json", {
                    "status": status,
                    "reason": reason,
                    "errorType": finalization_error,
                })
            except BaseException:
                pass

    result = {
        "trancheId": TRANCHE,
        "status": status,
        "reason": reason,
        "evidenceDirectory": str(campaign_dir),
        "elapsedSeconds": round(time.monotonic() - started, 3),
        "nativeClassification": native_classification,
    }
    print(json.dumps(result, ensure_ascii=False))
    return status in ("NOT_REPRODUCED", "REPRODUCED_503", "STOPPED_BY_ROOT")


def inspect(root: Path) -> dict[str, Any]:
    admission_path = root / ADMISSION_PATH
    runtime = root / RUNTIME_PATH
    binary = root / GO_COMPILED_BINARY
    return {
        "trancheId": TRANCHE,
        "mode": "INSPECT_ONLY",
        "runtimeAdmissionPresent": admission_path.is_file(),
        "runtimeAdmissionCommitted": False,
        "binaryPresent": binary.is_file(),
        "binarySha256": sha256_file(binary) if binary.is_file() else None,
        "campaignSlotUsed": (runtime / "campaign").exists(),
        "runtimeStarted": False,
        "browserOwner": "Codex /root",
        "workerBrowserRole": "NONE",
    }


def request_stop(root: Path) -> dict[str, Any]:
    campaign_dir = root / RUNTIME_PATH / "campaign"
    require(campaign_dir.is_dir() and not campaign_dir.is_symlink(), "R109 campaign is not active")
    require((campaign_dir / "campaign_intent.json").is_file(), "R109 campaign intent is missing")
    require(not (campaign_dir / "cleanup.json").exists(), "R109 campaign has already entered cleanup")
    stop_path = campaign_dir / "stop.request"
    if stop_path.exists():
        return {"trancheId": TRANCHE, "status": "STOP_ALREADY_REQUESTED", "runtimeStarted": True}
    write_json_once(stop_path, {"requestedAtUnix": time.time(), "requester": "Root cooperative stop"})
    return {"trancheId": TRANCHE, "status": "STOP_REQUESTED", "runtimeStarted": True}


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", nargs="?", choices=("inspect", "start", "status", "stop"), default="inspect")
    parser.add_argument("--project-root", type=Path, required=True)
    args = parser.parse_args(argv)
    root = args.project_root.resolve()
    try:
        require((root / ".git").exists() and (root / ".cvf/manifest.json").is_file(), "project root is not Customer-Care-Monitor-AI")
        if args.command == "inspect":
            print(json.dumps(inspect(root), ensure_ascii=False))
            return 0
        if args.command == "stop":
            print(json.dumps(request_stop(root), ensure_ascii=False))
            return 0
        if args.command == "status":
            campaign_dir = root / RUNTIME_PATH / "campaign"
            require(campaign_dir.is_dir(), "R109 campaign has not started")
            disposition_path = campaign_dir / "campaign_disposition.json"
            result_path = campaign_dir / "campaign_result.json"
            if disposition_path.is_file():
                state = read_json(disposition_path)
            elif result_path.is_file():
                state = read_json(result_path)
            else:
                state = {"status": "RUNNING_OR_FINALIZING"}
            print(json.dumps({"trancheId": TRANCHE, **state, "uiPostCounts": ui_post_counts(campaign_dir, allow_partial=not (campaign_dir / "cleanup.json").exists())}, ensure_ascii=False))
            return 0
        admission, helper, docker, node, binary, head = verify_admission(root)
        status_ok = run_campaign(root, admission, helper, docker, node, binary, head)
        return 0 if status_ok else 1
    except Hold as exc:
        print(json.dumps({"trancheId": TRANCHE, "status": "HOLD", "reason": str(exc)}, ensure_ascii=False))
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
