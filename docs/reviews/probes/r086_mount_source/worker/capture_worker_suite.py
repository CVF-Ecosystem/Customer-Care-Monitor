"""One-shot raw capture for the authorized R086 worker unittest suite."""

from __future__ import annotations

import hashlib
import json
import re
import subprocess
import sys
import time
from pathlib import Path


ROOT = Path(__file__).resolve().parents[5]
PROBE_DIR = ROOT / "docs/reviews/probes/r086_mount_source/worker"
SOURCE = ROOT / "scripts/jobrun_fence_fixture.py"
TEST = ROOT / "scripts/tests/test_jobrun_fence_fixture.py"
COMMAND = [
    sys.executable,
    "-m",
    "unittest",
    "discover",
    "-s",
    "scripts/tests",
    "-p",
    "test_jobrun_fence_fixture.py",
    "-v",
]


def file_sha(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def write_json(path: Path, payload: dict[str, object]) -> None:
    path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")


def main() -> int:
    PROBE_DIR.mkdir(parents=True, exist_ok=True)
    source_before = file_sha(SOURCE)
    test_before = file_sha(TEST)
    stdout_path = PROBE_DIR / "worker_harness_stdout.raw.bin"
    stderr_path = PROBE_DIR / "worker_harness_stderr.raw.bin"
    launcher_errors: list[dict[str, str]] = []
    process = None
    stdout = b""
    stderr = b""
    return_code: int | None = None
    timed_out = False
    launch_attempts = 1
    start = time.perf_counter()
    try:
        process = subprocess.Popen(
            COMMAND,
            cwd=ROOT,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        try:
            stdout, stderr = process.communicate(timeout=180)
        except subprocess.TimeoutExpired:
            timed_out = True
            process.kill()
            stdout, stderr = process.communicate()
        return_code = process.returncode
    except BaseException as exc:  # Persist actual launch/capture failures separately.
        launcher_errors.append({"type": type(exc).__name__, "message": str(exc)})
        if process is not None and process.poll() is None:
            process.kill()
            stdout, stderr = process.communicate()
            return_code = process.returncode
    elapsed = time.perf_counter() - start
    stdout_path.write_bytes(stdout)
    stderr_path.write_bytes(stderr)
    write_json(PROBE_DIR / "worker_launcher_errors.json", {
        "launchAttempts": launch_attempts,
        "errors": launcher_errors,
    })

    summary_text = (stdout + b"\n" + stderr).decode("utf-8", errors="replace")
    ran_match = re.search(r"^Ran (\d+) tests? in ([0-9.]+)s$", summary_text, flags=re.MULTILINE)
    result_line = next((line.strip() for line in reversed(summary_text.splitlines()) if line.strip() in ("OK", "FAILED", "OK (skipped=0)")), None)
    receipt = {
        "trancheId": "CCMAI-RUNTIME-086",
        "phase": "BUILD_WORKER_HARNESS_RESULT",
        "command": COMMAND,
        "launchAttempts": launch_attempts,
        "processCount": 1 if process is not None else 0,
        "returnCode": return_code,
        "timedOut": timed_out,
        "elapsedSeconds": round(elapsed, 6),
        "testsRunFromRawSummary": int(ran_match.group(1)) if ran_match else None,
        "unittestElapsedFromRawSummary": float(ran_match.group(2)) if ran_match else None,
        "resultLineFromRawOutput": result_line,
        "stdout": {
            "path": str(stdout_path.relative_to(ROOT)).replace("\\", "/"),
            "bytes": len(stdout),
            "sha256": hashlib.sha256(stdout).hexdigest(),
        },
        "stderr": {
            "path": str(stderr_path.relative_to(ROOT)).replace("\\", "/"),
            "bytes": len(stderr),
            "sha256": hashlib.sha256(stderr).hexdigest(),
        },
        "sourcePhysicalSha256Before": source_before,
        "sourcePhysicalSha256After": file_sha(SOURCE),
        "testPhysicalSha256Before": test_before,
        "testPhysicalSha256After": file_sha(TEST),
        "nativeResourceEffects": 0,
        "fixtureCampaigns": 0,
        "sqlCalls": "mocked in-process readiness boundary only",
    }
    write_json(PROBE_DIR / "worker_harness_receipt.json", receipt)
    print(json.dumps(receipt, indent=2))
    return 0 if return_code == 0 and not timed_out and not launcher_errors else 1


if __name__ == "__main__":
    raise SystemExit(main())
