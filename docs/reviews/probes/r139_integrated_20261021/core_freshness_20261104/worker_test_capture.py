"""Bounded worker-only test capture; not a product or gate module."""
import base64
import contextlib
import datetime
import hashlib
import io
import json
from pathlib import Path
import subprocess
import sys
import time
import unittest

home = Path(__file__).resolve().parent
batch = sys.argv[1]
if batch not in ("focused", "full"):
    raise SystemExit("unknown batch")
started = time.monotonic()
started_utc = datetime.datetime.now(datetime.timezone.utc).isoformat()
source_paths = ("scripts/cvf_downstream_gate.py", "scripts/tests/test_cvf_downstream_gate_core_repin.py")
source_hashes = {p: hashlib.sha256(Path(p).read_bytes()).hexdigest() for p in source_paths}
original_run = subprocess.run
child_log = (home / ("worker_" + batch + "_children.jsonl")).open("w", encoding="utf-8")

def encoded(value):
    if value is None:
        return None
    return base64.b64encode(value.encode("utf-8") if isinstance(value, str) else value).decode("ascii")

def captured_run(*args, **kwargs):
    before = time.monotonic()
    utc = datetime.datetime.now(datetime.timezone.utc).isoformat()
    result = None
    error = None
    try:
        result = original_run(*args, **kwargs)
        return result
    except BaseException as exc:
        error = type(exc).__name__
        result = exc if hasattr(exc, "stdout") else None
        raise
    finally:
        child_log.write(json.dumps({"argv": args[0] if args else kwargs.get("args"),
                                   "startedAtUTC": utc, "durationSeconds": time.monotonic() - before,
                                   "exitCode": getattr(result, "returncode", None), "error": error,
                                   "stdoutBase64": encoded(getattr(result, "stdout", None)),
                                   "stderrBase64": encoded(getattr(result, "stderr", None))}, default=str) + "\n")
        child_log.flush()

subprocess.run = captured_run
stdout, stderr = io.StringIO(), io.StringIO()
result = None
error = None
try:
    with contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(stderr):
        suite = unittest.defaultTestLoader.discover("scripts/tests", pattern=("test_cvf_downstream_gate_core_repin.py" if batch == "focused" else "test_cvf_downstream_gate*.py"))
        result = unittest.TextTestRunner(stream=stderr, verbosity=2).run(suite)
except BaseException as exc:
    error = repr(exc)
finally:
    subprocess.run = original_run
    child_log.close()
    (home / ("worker_" + batch + ".stdout.txt")).write_text(stdout.getvalue(), encoding="utf-8")
    (home / ("worker_" + batch + ".stderr.txt")).write_text(stderr.getvalue(), encoding="utf-8")
    receipt = {"argv": sys.argv, "startedAtUTC": started_utc, "startedMonotonic": started,
               "durationSeconds": time.monotonic() - started, "sourceRawSha256": source_hashes,
               "exitCode": 0 if result and result.wasSuccessful() and not error else 1,
               "testsRun": result.testsRun if result else None,
               "failures": len(result.failures) if result else None,
               "errors": len(result.errors) if result else None, "error": error,
               "skipped": len(result.skipped) if result else None}
    (home / ("worker_" + batch + "_receipt.json")).write_text(json.dumps(receipt, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(receipt))
raise SystemExit(receipt["exitCode"])
