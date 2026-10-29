"""Static, fail-closed R139 maintenance admission checks (no runtime effects)."""
import hashlib
import json
import os
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))
import cvf_downstream_gate as gate


def run(root, *args):
    return subprocess.run(["git", "-C", root, *args], check=True, capture_output=True).stdout


def write(root, rel, data):
    path = os.path.join(root, *rel.split("/"))
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "wb") as fh:
        fh.write(data)


class MaintenanceAdmission(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = self.tmp.name
        run(self.root, "init", "-q")
        run(self.root, "config", "user.email", "test@example.invalid")
        run(self.root, "config", "user.name", "Gate Test")
        self.seed_rel = "CVF_SESSION/authority/CCMAI-RUNTIME-139.json"
        self.seed_bytes = b'{"trancheId":"CCMAI-RUNTIME-139","riskCeiling":"R2"}\n'
        write(self.root, self.seed_rel, self.seed_bytes)
        write(self.root, "base.txt", b"base\n")
        run(self.root, "add", ".")
        run(self.root, "commit", "-qm", "base")
        self.base = run(self.root, "rev-parse", "HEAD").decode().strip()
        self.old_base = gate._MAINTENANCE_BASE
        gate._MAINTENANCE_BASE = self.base
        self.changed = "scripts/cvf_downstream_gate.py"
        self.changed_data = b"maintenance\n"
        write(self.root, self.changed, self.changed_data)
        self.admission = {
            "schemaVersion": "1.0", "trancheId": "CCMAI-RUNTIME-139",
            "baseCommit": self.base, "seedSha256": hashlib.sha256(self.seed_bytes).hexdigest(),
            "riskCeiling": "R2", "purpose": "STATIC_REPOSITORY_MAINTENANCE_ONLY",
            "ownerInstruction": "commit scoped maintenance",
            "roles": {"worker": "Sol", "reviewer": "Root", "commitSteward": "Root"},
            "files": {self.changed: hashlib.sha256(self.changed_data).hexdigest()},
            "bootstrapException": {"reason": "first same-commit admission", "ownerInstruction": "commit scoped maintenance"},
        }
        self.save()

    def tearDown(self):
        gate._MAINTENANCE_BASE = self.old_base
        self.tmp.cleanup()

    def save(self):
        write(self.root, gate._MAINTENANCE_ADMISSION,
              (json.dumps(self.admission, sort_keys=True) + "\n").encode())

    def check(self, paths=None):
        result = gate.Result("tranche")
        found = gate.check_maintenance_admission(
            self.root, "CCMAI-RUNTIME-139", {"riskCeiling": "R2"},
            paths if paths is not None else [self.changed], result)
        return found, result.problems

    def test_exact_first_commit_and_immutable_committed_blob(self):
        found, problems = self.check()
        self.assertEqual(problems, [])
        self.assertEqual(found, [self.changed])
        run(self.root, "add", ".")
        self.assertEqual(self.check()[1], [])
        run(self.root, "commit", "-qm", "admit maintenance")
        self.assertEqual(self.check()[1], [])
        self.admission["ownerInstruction"] = "changed later"
        self.admission["bootstrapException"]["ownerInstruction"] = "changed later"
        self.save()
        self.assertTrue(any("first committed" in p for p in self.check()[1]))

    def test_identity_scope_and_hash_failures(self):
        mutations = (
            ("baseCommit", "0" * 40), ("seedSha256", "0" * 64),
            ("riskCeiling", "R1"), ("purpose", "RUNTIME"),
        )
        for key, value in mutations:
            with self.subTest(key=key):
                original = self.admission[key]
                self.admission[key] = value
                self.save()
                self.assertTrue(self.check()[1])
                self.admission[key] = original
        self.admission["files"][self.changed] = "0" * 64
        self.save()
        self.assertTrue(any("hash differs" in p for p in self.check()[1]))

    def test_role_unsafe_path_and_unused_scope_fail(self):
        self.admission["roles"]["reviewer"] = "Sol"
        self.save()
        self.assertTrue(self.check()[1])
        self.admission["roles"]["reviewer"] = "Root"
        self.admission["files"] = {"backend/product.go": "0" * 64}
        self.save()
        self.assertTrue(self.check(["backend/product.go"])[1])
        self.admission["files"] = {self.changed: hashlib.sha256(self.changed_data).hexdigest()}
        self.save()
        self.assertTrue(any("unused" in p for p in self.check([])[1]))
        self.assertTrue(self.check([self.changed, "another.py"])[1])

    def test_duplicate_json_key_rejected(self):
        raw = json.dumps(self.admission)
        write(self.root, gate._MAINTENANCE_ADMISSION,
              raw.replace('"schemaVersion": "1.0"', '"schemaVersion": "1.0", "schemaVersion": "1.0"').encode())
        self.assertTrue(any("invalid" in p for p in self.check()[1]))

    def test_staged_content_must_match_hash(self):
        run(self.root, "add", ".")
        write(self.root, self.changed, b"changed after staging\n")
        self.assertTrue(any("hash differs" in p for p in self.check()[1]))

    def test_staged_only_payload_and_admission_tamper(self):
        run(self.root, "add", ".")
        write(self.root, self.changed, b"staged wrong\n")
        run(self.root, "add", self.changed)
        write(self.root, self.changed, self.changed_data)
        self.assertTrue(any("staged maintenance file hash differs" in p for p in self.check()[1]))
        run(self.root, "add", self.changed)
        self.admission["ownerInstruction"] = "worktree changed after staging"
        self.admission["bootstrapException"]["ownerInstruction"] = "worktree changed after staging"
        self.save()
        self.assertTrue(any("staged maintenance admission differs" in p for p in self.check()[1]))

    def test_uncommitted_bootstrap_cannot_follow_another_commit(self):
        write(self.root, "intervening.txt", b"another commit\n")
        run(self.root, "add", "intervening.txt")
        run(self.root, "commit", "-qm", "intervening")
        self.assertTrue(any("bootstrap requires absent history" in p for p in self.check()[1]))

    def test_missing_payload_and_unexpected_field_fail(self):
        os.unlink(os.path.join(self.root, *self.changed.split("/")))
        self.assertTrue(any("missing" in p for p in self.check()[1]))
        write(self.root, self.changed, self.changed_data)
        self.admission["extraAuthority"] = True
        self.save()
        self.assertTrue(any("field set" in p for p in self.check()[1]))

    def test_traversal_glob_and_ads_paths_fail(self):
        for bad in ("scripts/../scripts/cvf_downstream_gate.py",
                    "scripts/cvf_gates/*.py", "scripts/cvf_gates/module.py:stream"):
            with self.subTest(path=bad):
                self.admission["files"] = {bad: "0" * 64}
                self.save()
                self.assertTrue(any("path or digest" in p for p in self.check([bad])[1]))

    def test_inherited_gate_requires_matching_lock_pin(self):
        path = "scripts/cvf_gates/check_gate_to_role_closeability.py"
        data = b"pinned gate fixture\n"
        digest = hashlib.sha256(data).hexdigest()
        write(self.root, path, data)
        self.admission["files"] = {path: digest}
        self.save()
        write(self.root, ".cvf/gate-profile.lock.json",
              json.dumps({"gatesDir": "scripts/cvf_gates", "files": {os.path.basename(path): digest}}).encode())
        self.assertEqual(self.check([path])[1], [])
        self.admission["files"] = {"scripts/cvf_gates/unpinned.py": digest}
        self.save()
        write(self.root, "scripts/cvf_gates/unpinned.py", data)
        self.assertTrue(any("not pinned" in p for p in self.check(["scripts/cvf_gates/unpinned.py"])[1]))
        self.admission["files"] = {path: digest}
        self.save()
        write(self.root, ".cvf/gate-profile.lock.json",
              json.dumps({"gatesDir": "scripts/cvf_gates", "files": {os.path.basename(path): "0" * 64}}).encode())
        self.assertTrue(any("not pinned" in p for p in self.check([path])[1]))

    def test_admission_alias_rejected_before_read(self):
        admission_path = os.path.join(self.root, *gate._MAINTENANCE_ADMISSION.split("/"))
        original_realpath = gate.os.path.realpath
        with mock.patch.object(gate.os.path, "realpath",
                               side_effect=lambda p: os.path.join(self.root, "other.json") if p == admission_path else original_realpath(p)):
            self.assertTrue(any("admission path is an alias" in p for p in self.check()[1]))

    def test_symlink_alias_is_rejected_when_supported(self):
        target = os.path.join(self.root, "base.txt")
        link = os.path.join(self.root, *self.changed.split("/"))
        os.unlink(link)
        try:
            os.symlink(target, link)
        except (OSError, NotImplementedError):
            original_realpath = gate.os.path.realpath
            with mock.patch.object(gate.os.path, "realpath", side_effect=lambda p: target if p == link else original_realpath(p)):
                self.assertTrue(any("outside project" in p for p in self.check()[1]))
            return
        self.admission["files"][self.changed] = hashlib.sha256(b"base\n").hexdigest()
        self.save()
        self.assertTrue(any("outside project" in p for p in self.check()[1]))


if __name__ == "__main__":
    unittest.main()
