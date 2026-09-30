"""Negative/positive checks for scripts/ci_db_test_gate.py (synthetic go test -json logs only)."""
import json
import os
import subprocess
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.dirname(HERE))
import ci_db_test_gate as gate  # noqa: E402

SENT = gate.REQUIRED_SENTINELS


def ev(action, pkg, test=None, output=None):
    d = {"Action": action, "Package": pkg}
    if test:
        d["Test"] = test
    if output is not None:
        d["Output"] = output
    return json.dumps(d)


def good_log(skip=None, fail=None, drop=None, truncate_pkg=None):
    """A complete passing log; options mutate one aspect."""
    lines = []
    for pkg, test in SENT:
        if test == drop:
            continue
        lines.append(ev("run", pkg, test))
        if test == skip:
            lines.append(ev("output", pkg, test, "    x_test.go:9: bo qua: TEST_DB_DSN chua duoc thiet lap\n"))
            lines.append(ev("skip", pkg, test))
        elif test == fail:
            lines.append(ev("fail", pkg, test))
        else:
            lines.append(ev("pass", pkg, test))
    for pkg, _ in SENT:
        if pkg == truncate_pkg:
            continue
        lines.append(ev("fail" if any(t == fail and p == pkg for p, t in SENT) else "pass", pkg))
    return lines


class GateTests(unittest.TestCase):
    def test_positive_all_sentinels_pass(self):
        ok, report = gate.evaluate(good_log())
        self.assertTrue(ok, report)
        self.assertIn("GATE PASSED", report[-1])

    def test_required_skip_fails_even_with_package_pass(self):
        ok, report = gate.evaluate(good_log(skip=SENT[2][1]))
        self.assertFalse(ok)
        self.assertTrue(any("required test skip" in r for r in report))
        self.assertTrue(any("DB-unavailable skip" in r for r in report))

    def test_missing_required_result_fails(self):
        ok, report = gate.evaluate(good_log(drop=SENT[0][1]))
        self.assertFalse(ok)
        self.assertTrue(any("required test missing" in r for r in report))

    def test_failed_required_test_fails(self):
        ok, report = gate.evaluate(good_log(fail=SENT[3][1]))
        self.assertFalse(ok)
        self.assertTrue(any("test failed" in r for r in report))

    def test_truncated_package_result_fails(self):
        ok, report = gate.evaluate(good_log(truncate_pkg=SENT[4][0]))
        self.assertFalse(ok)
        self.assertTrue(any("package not passed" in r for r in report))

    def test_empty_log_fails(self):
        ok, _ = gate.evaluate([])
        self.assertFalse(ok)
        ok, _ = gate.evaluate(["not json at all", ""])
        self.assertFalse(ok)

    def test_unrelated_db_skip_fails_but_optional_skip_does_not(self):
        extra_db = [ev("run", "p/other", "TestX"), ev("output", "p/other", "TestX", "khong ket noi duoc DB test: dial tcp\n"), ev("skip", "p/other", "TestX")]
        ok, report = gate.evaluate(good_log() + extra_db)
        self.assertFalse(ok)
        self.assertTrue(any("p/other.TestX" in r for r in report))
        optional = [ev("run", "p/s3", "TestS3"), ev("output", "p/s3", "TestS3", "skipping: S3 not configured\n"), ev("skip", "p/s3", "TestS3")]
        ok, report = gate.evaluate(good_log() + optional)
        self.assertTrue(ok, report)
        self.assertTrue(any("other skip: p/s3.TestS3" in r for r in report))

    def test_report_never_echoes_raw_output(self):
        secret = "user:hunter2@tcp(db:3306)"
        log = good_log() + [ev("run", "p/x", "TestY"), ev("output", "p/x", "TestY", f"TEST_DB_DSN {secret}\n"), ev("skip", "p/x", "TestY")]
        _, report = gate.evaluate(log)
        self.assertNotIn("hunter2", "\n".join(report))

    def test_cli_exit_codes_and_unreadable_log(self):
        script = os.path.join(os.path.dirname(HERE), "ci_db_test_gate.py")
        with tempfile.TemporaryDirectory() as d:
            good, bad = os.path.join(d, "good.json"), os.path.join(d, "bad.json")
            open(good, "w").write("\n".join(good_log()))
            open(bad, "w").write("\n".join(good_log(skip=SENT[1][1])))
            run = lambda p: subprocess.run([sys.executable, script, p], capture_output=True, text=True).returncode
            self.assertEqual(run(good), 0)
            self.assertEqual(run(bad), 1)
            self.assertEqual(run(os.path.join(d, "missing.json")), 1)


if __name__ == "__main__":
    unittest.main()
