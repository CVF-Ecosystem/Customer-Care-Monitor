"""STATIC_CANDIDATE_ONLY: synthetic history, no project repository commits."""
import copy
import json
import os
import shutil
import stat
import subprocess
import tempfile
import unittest
from pathlib import Path

import candidate as c

HERE = Path(__file__).resolve().parent

class CandidateTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.tmp = Path(tempfile.mkdtemp(prefix="r155_fixture_", dir=HERE))
        cls.root = cls.tmp / "repo"
        cls.root.mkdir()
        def git(*args):
            return subprocess.run(["git", "-C", str(cls.root), *args], check=True,
                                  stdout=subprocess.PIPE, stderr=subprocess.PIPE).stdout.decode().strip()
        cls.git = staticmethod(git)
        git("init", "-q")
        git("config", "user.name", "Synthetic Fixture")
        git("config", "user.email", "fixture@example.invalid")
        for name in ("packet", "registry", "audit", "seed", "record", "review"):
            (cls.root / name).write_text(name + "\n", encoding="utf-8")
        git("add", "."); git("commit", "-qm", "admission")
        cls.admission = git("rev-parse", "HEAD")
        (cls.root / "build-marker").write_text("build\n")
        git("add", "."); git("commit", "-qm", "build")
        cls.build = git("rev-parse", "HEAD")
        (cls.root / "terminal-marker").write_text("terminal\n")
        git("add", "."); git("commit", "-qm", "terminal")
        cls.terminal = git("rev-parse", "HEAD")
        cls.head = cls.terminal
        def reference(name):
            return {"path": name, "sha256": c.digest((cls.root / name).read_bytes())}
        cls.row = {"milestone": "M3", "tranche": "T150", "packet": reference("packet"),
                   "admission": cls.admission, "build": cls.build, "terminal": cls.terminal,
                   "registry": reference("registry"), "audit": reference("audit"),
                   "seed": reference("seed"), "terminalRecord": reference("record"),
                   "reviewEvidence": [reference("review")]}

    @classmethod
    def tearDownClass(cls):
        def writable_then_retry(function, name, _error):
            os.chmod(name, stat.S_IWRITE | stat.S_IREAD)
            function(name)
        shutil.rmtree(cls.tmp, onexc=writable_then_retry)

    def run_row(self, row=None, **kwargs):
        doc = {"schemaVersion": "cvf.historicalRepairAdmissions@candidate1",
               "bindings": [self.row if row is None else row]}
        return c.preflight(self.root, json.dumps(doc).encode(), phase=kwargs.get("phase", "pr-ci"),
                           checkout_head=kwargs.get("head", self.head), packet_path="packet",
                           active=kwargs.get("active", False), explicit=kwargs.get("explicit", False))

    def test_provenance_only_is_unsupported(self):
        self.assertEqual(self.run_row(), "UNSUPPORTED_HISTORICAL_COVERAGE_JOIN")

    def test_execution_paths_rejected(self):
        for opts in ({"phase": "pre-dispatch"}, {"phase": "worker-return"},
                     {"phase": "bootstrap"}, {"active": True}, {"explicit": True}):
            with self.subTest(opts=opts), self.assertRaisesRegex(c.Reject, "EXECUTION_FORBIDDEN"):
                self.run_row(**opts)

    def test_head_and_ancestry_rejected(self):
        with self.assertRaisesRegex(c.Reject, "HEAD_MISMATCH"):
            self.run_row(head=self.admission)
        for name, value, error in (("admission", "0" * 40, "NONANCESTOR"),
                                   ("build", self.admission, "ORDER"),
                                   ("terminal", "bad", "COMMIT_INVALID")):
            row = copy.deepcopy(self.row); row[name] = value
            with self.subTest(name=name), self.assertRaisesRegex(c.Reject, error):
                self.run_row(row)

    def test_mutated_bound_bytes_rejected(self):
        for name in ("packet", "registry", "audit", "seed", "terminalRecord"):
            row = copy.deepcopy(self.row); row[name]["sha256"] = "0" * 64
            with self.subTest(name=name), self.assertRaisesRegex(c.Reject, "STALE"):
                self.run_row(row)
        row = copy.deepcopy(self.row); row["reviewEvidence"][0]["sha256"] = "0" * 64
        with self.assertRaisesRegex(c.Reject, "STALE"):
            self.run_row(row)

    def test_binding_parse_and_path_fail_closed(self):
        for raw, code in ((b'{"a":1,"a":2}', "DUPLICATE_KEY"),
                          (b'{"x":NaN}', "NONFINITE"), (b" " * (c.LIMIT + 1), "OVERSIZE")):
            with self.subTest(code=code), self.assertRaisesRegex(c.Reject, code):
                c.parse(raw)
        row = copy.deepcopy(self.row); row["seed"]["path"] = "../seed"
        with self.assertRaisesRegex(c.Reject, "UNSAFE_PATH"):
            self.run_row(row)
        doc = {"schemaVersion": "cvf.historicalRepairAdmissions@candidate1", "bindings": [self.row, self.row]}
        with self.assertRaisesRegex(c.Reject, "DUPLICATE_BINDING"):
            c.preflight(self.root, json.dumps(doc).encode(), phase="pr-ci", checkout_head=self.head, packet_path="packet")

    def test_missing_binding_and_review(self):
        row = copy.deepcopy(self.row); row["reviewEvidence"] = []
        with self.assertRaisesRegex(c.Reject, "REVIEW_REQUIRED"):
            self.run_row(row)
        row = copy.deepcopy(self.row); row["packet"]["path"] = "other"
        with self.assertRaisesRegex(c.Reject, "BINDING_MISSING"):
            self.run_row(row)

if __name__ == "__main__":
    unittest.main(verbosity=2)
