#!/usr/bin/env python3
"""CCMAI-RUNTIME-019 / F08: fail backend CI when DB-backed tests did not really run.

Reads `go test -json` output and exits nonzero unless:
  * the log is non-empty, every line is a valid JSON record, and no test/package reported "fail";
  * every required DB sentinel test (package-qualified) finished with "pass"
    and its package reached a terminal "pass" event (so truncated logs fail);
  * no test anywhere skipped because the test database was missing or unreachable.
Other skips (optional S3 / live-fetch tests) are reported but do not fail the gate.

Only test names, packages and counts are printed. Raw test output, which could
carry connection details, is inspected for skip classification but never echoed.
"""
import json
import sys
from collections import defaultdict

MODULE = "github.com/CVF-Ecosystem/Customer-Care-Monitor/backend"

# One DB-dependent test per package that owns DB tests. Each calls a helper that
# t.Skip()s when TEST_DB_DSN is unset or the database is unreachable.
REQUIRED_SENTINELS = [
    (f"{MODULE}/db", "TestChannelSyncRunIDColumnOnFreshSchema"),
    (f"{MODULE}/api/handlers", "TestDeleteChannelRemovesResultsAndSnapshotsTogether"),
    (f"{MODULE}/engine", "TestSingleAndBatchShareSnapshotContract"),
    (f"{MODULE}/cli", "TestApplyPrunePlanCleansOrphanSnapshotsKeepsReferenced"),
    (f"{MODULE}/storagecfg", "TestTatS3VanDocDuocThongTinDaLuu"),
]

# Markers used by the existing test helpers for an absent/unreachable test DB.
DB_SKIP_MARKERS = ("TEST_DB_DSN", "khong ket noi duoc DB test", "DB not available")


def evaluate(lines, sentinels=REQUIRED_SENTINELS):
    """Return (ok, report_lines) for an iterable of `go test -json` lines."""
    events = 0
    bad_lines = 0
    terminal = {}                # (package, test) -> last pass/fail/skip action
    pkg_terminal = {}            # package -> last package-level pass/fail/skip
    skip_output = defaultdict(list)
    for raw in lines:
        raw = raw.strip()
        if not raw:
            continue
        try:
            ev = json.loads(raw)
        except ValueError:
            bad_lines += 1
            continue
        if not isinstance(ev, dict) or "Action" not in ev:
            bad_lines += 1
            continue
        events += 1
        action, pkg, test = ev["Action"], ev.get("Package", ""), ev.get("Test", "")
        if action == "output" and test:
            skip_output[(pkg, test)].append(ev.get("Output", ""))
        elif action in ("pass", "fail", "skip"):
            if test:
                terminal[(pkg, test)] = action
            elif pkg:
                pkg_terminal[pkg] = action

    problems = []
    if events == 0:
        problems.append("no go test events found (empty or truncated log)")
    if bad_lines:
        # go test -json emits only JSON records; anything else means a damaged or incomplete log.
        problems.append(f"invalid JSON record(s) in log: {bad_lines}")

    failed = sorted(f"{p}.{t}" for (p, t), a in terminal.items() if a == "fail")
    failed_pkgs = sorted(p for p, a in pkg_terminal.items() if a == "fail")
    for name in failed:
        problems.append(f"test failed: {name}")
    for p in failed_pkgs:
        problems.append(f"package failed: {p}")

    counts = {"pass": 0, "fail": 0, "skip": 0}
    for a in terminal.values():
        counts[a] += 1

    db_skips, other_skips = [], []
    for (p, t), a in sorted(terminal.items()):
        if a != "skip":
            continue
        text = "".join(skip_output.get((p, t), []))
        (db_skips if any(m in text for m in DB_SKIP_MARKERS) else other_skips).append(f"{p}.{t}")
    for name in db_skips:
        problems.append(f"DB-unavailable skip: {name}")

    sentinel_lines = []
    for p, t in sentinels:
        state = terminal.get((p, t))
        if state is None:
            problems.append(f"required test missing: {p}.{t}")
            sentinel_lines.append(f"  MISSING {p}.{t}")
            continue
        if state != "pass":
            problems.append(f"required test {state}: {p}.{t}")
        elif pkg_terminal.get(p) != "pass":
            problems.append(f"required test package not passed: {p}")
        sentinel_lines.append(f"  {state.upper():7} {p}.{t}")

    report = [
        f"go test events: {events} (invalid records: {bad_lines})",
        f"tests: pass={counts['pass']} fail={counts['fail']} skip={counts['skip']}",
        "required sentinels:",
        *sentinel_lines,
        f"skips: db-unavailable={len(db_skips)} other={len(other_skips)}",
        *(f"  other skip: {n}" for n in other_skips),
    ]
    if problems:
        report.append("GATE FAILED:")
        report.extend(f"  - {x}" for x in problems)
    else:
        report.append("GATE PASSED: all required DB sentinels ran and passed; no DB-unavailable skips.")
    return (not problems), report


def main(argv):
    if len(argv) != 2:
        print("usage: ci_db_test_gate.py <go-test-json-file>", file=sys.stderr)
        return 2
    try:
        with open(argv[1], encoding="utf-8", errors="replace") as fh:
            ok, report = evaluate(fh)
    except OSError as exc:
        print(f"GATE FAILED: cannot read test log: {exc.__class__.__name__}", file=sys.stderr)
        return 1
    print("\n".join(report))
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv))
