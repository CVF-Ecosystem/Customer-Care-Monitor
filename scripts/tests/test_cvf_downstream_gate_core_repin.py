"""Fixed successor static contract, using disposable Git histories only."""
import hashlib
import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path
import unittest
from unittest import mock

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))
import cvf_downstream_gate as gate


def git(root, *args):
    return subprocess.run(["git", "-C", root, *args], capture_output=True, check=True).stdout


def write(root, path, data):
    full = os.path.join(root, *path.split("/"))
    os.makedirs(os.path.dirname(full), exist_ok=True)
    with open(full, "wb") as f:
        f.write(data)


def encode(value):
    return (json.dumps(value, sort_keys=True) + "\n").encode()


class CoreRepin(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = self.tmp.name
        git(self.root, "init", "-q")
        git(self.root, "config", "user.email", "fixture@example.invalid")
        git(self.root, "config", "user.name", "Fixture")
        self.seed_rel = "CVF_SESSION/authority/CCMAI-RUNTIME-139.json"
        self.seed = {"riskCeiling": "R2", "allowedPaths": list(getattr(self, "inherited_seed_paths", []))}
        seed_bytes = encode(self.seed)
        write(self.root, self.seed_rel, seed_bytes)
        git(self.root, "add", ".")
        git(self.root, "commit", "-qm", "seed")
        self.oldbase = gate._MAINTENANCE_BASE
        self.oldrepin = gate._REPIN_BASE
        gate._MAINTENANCE_BASE = git(self.root, "rev-parse", "HEAD").decode().strip()
        write(self.root, "CVF_SESSION/tranches/CCMAI-RUNTIME-139.json", encode({"baseCommit": gate._MAINTENANCE_BASE}))
        for name in gate._REPIN_INSTALLED:
            write(self.root, "scripts/cvf_gates/" + name, (name + "\n").encode())
        paths = list(getattr(self, "maintenance_fixture_paths", sorted(gate._MAINTENANCE_PATH_EXACT)[:12])) + ["scripts/cvf_gates/" + n for n in sorted(gate._REPIN_INSTALLED)[:5]]
        for p in paths:
            if not p.startswith("scripts/cvf_gates/"):
                write(self.root, p, b"# historical fixture\n")
        oldfiles = {p: gate._lf_sha(Path(self.root, *p.split("/")).read_bytes()) for p in paths}
        self.old = {"schemaVersion":"1.0", "trancheId":"CCMAI-RUNTIME-139", "baseCommit":gate._MAINTENANCE_BASE,
                    "seedSha256":gate._lf_sha(seed_bytes), "riskCeiling":"R2", "purpose":"STATIC_REPOSITORY_MAINTENANCE_ONLY",
                    "ownerInstruction":"old", "roles":{"worker":"Sol", "reviewer":"Root", "commitSteward":"Root"},
                    "files":oldfiles, "bootstrapException":{"reason":"same commit", "ownerInstruction":"old"}}
        self.oldbytes = encode(self.old)
        write(self.root, gate._MAINTENANCE_ADMISSION, self.oldbytes)
        git(self.root, "add", ".")
        git(self.root, "commit", "-qm", "old maintenance")
        gate._REPIN_BASE = git(self.root, "rev-parse", "HEAD").decode().strip()
        lockfiles = {n:gate._lf_sha(Path(self.root, "scripts/cvf_gates", n).read_bytes()) for n in gate._REPIN_INSTALLED}
        write(self.root, ".cvf/gate-profile.lock.json", encode({"profileId":"cvf.downstreamGateProfile@1.2.0", "profileVersion":"1.2.0", "coreCommit":gate._REPIN_CORE_TO, "gatesDir":"scripts/cvf_gates", "files":lockfiles}))
        write(self.root, ".cvf/manifest.json", encode({"cvfCoreCommit":gate._REPIN_CORE_TO}))
        write(self.root, "scripts/tests/test_cvf_downstream_gate_core_repin.py", b"new test\n")
        changed = set(gate.git_z(self.root,"diff","--name-only","-z",gate._MAINTENANCE_BASE+"..HEAD")) | set(gate.worktree_files(self.root))
        self.paths = sorted(p for p in changed if not gate.path_matches(p, self.seed["allowedPaths"] + gate.ALWAYS_ALLOWED))
        self.admission = {"schemaVersion":"1.0", "trancheId":"CCMAI-RUNTIME-139", "purpose":"STATIC_PUBLIC_CORE_REPIN_ONLY", "baseCommit":gate._REPIN_BASE,
                          "predecessor":{"path":gate._MAINTENANCE_ADMISSION, "sha256":gate._lf_sha(self.oldbytes)},
                          "seedSha256":gate._lf_sha(seed_bytes), "riskCeiling":"R2", "roles":{"worker":"Terra", "reviewer":"Root", "commitSteward":"Root"},
                          "coreFrom":gate._REPIN_CORE_FROM, "coreTo":gate._REPIN_CORE_TO,
                          "files":{p:gate._lf_sha(Path(self.root, *p.split("/")).read_bytes()) for p in self.paths},
                          "bootstrapException":{"reason":"first same-commit", "ownerInstruction":"approved"}, "ownerInstruction":"approved"}
        self.save()

    def tearDown(self):
        gate._MAINTENANCE_BASE = self.oldbase
        gate._REPIN_BASE = self.oldrepin
        self.tmp.cleanup()

    def save(self):
        write(self.root, gate._REPIN_ADMISSION, encode(self.admission))

    def check(self, paths=None):
        r=gate.Result("tranche")
        found=gate.check_maintenance_admission(self.root, "CCMAI-RUNTIME-139", self.seed, self.paths if paths is None else paths, r)
        return found, r.problems

    def test_bootstrap_and_committed_successor(self):
        self.assertEqual(self.check()[1], [])
        git(self.root,"add",".")
        self.assertEqual(self.check()[1], [])
        git(self.root,"commit","-qm","repin")
        self.assertEqual(self.check()[1], [])
        self.admission["ownerInstruction"]="later"
        self.admission["bootstrapException"]["ownerInstruction"]="later"
        self.save()
        self.assertTrue(self.check()[1])

    def test_identity_schema_roles_and_pins_rejected(self):
        for k,v in (("coreTo","0"*40),("baseCommit","0"*40),("riskCeiling","R1"),("purpose","runtime"),("extra",True)):
            with self.subTest(k=k):
                self.admission[k]=v; self.save(); self.assertTrue(self.check()[1]); self.admission.pop(k) if k=="extra" else self.admission.__setitem__(k, {"coreTo":gate._REPIN_CORE_TO,"baseCommit":gate._REPIN_BASE,"riskCeiling":"R2","purpose":"STATIC_PUBLIC_CORE_REPIN_ONLY"}[k])
        self.admission["roles"]["reviewer"]="Terra"; self.save(); self.assertTrue(self.check()[1])

    def test_history_payload_and_scope_tamper(self):
        self.admission["predecessor"]["sha256"]="0"*64; self.save(); self.assertTrue(self.check()[1])
        self.admission["predecessor"]["sha256"]=gate._lf_sha(self.oldbytes)
        original=self.admission["files"].pop(self.paths[0]); self.save(); self.assertTrue(self.check()[1]); self.admission["files"][self.paths[0]]=original
        nonrep=next(p for p in self.old["files"] if p not in gate._REPIN_EXACT and not p.startswith("scripts/cvf_gates/"))
        write(self.root,nonrep,b"changed\n"); self.admission["files"][nonrep]=gate._lf_sha(b"changed\n"); self.save(); self.assertTrue(self.check()[1])

    def test_staging_and_duplicate_rejected(self):
        git(self.root,"add",".")
        write(self.root,".cvf/manifest.json",b"bad\n"); git(self.root,"add",".cvf/manifest.json")
        write(self.root,".cvf/manifest.json",encode({"cvfCoreCommit":gate._REPIN_CORE_TO}))
        self.assertTrue(self.check()[1])
        raw=encode(self.admission).replace(b'"schemaVersion": "1.0"',b'"schemaVersion": "1.0", "schemaVersion": "1.0"')
        write(self.root,gate._REPIN_ADMISSION,raw); self.assertTrue(self.check()[1])

    def test_bad_profile_alias_and_range_rejected(self):
        lockpath=os.path.join(self.root,".cvf/gate-profile.lock.json")
        original=Path(lockpath).read_bytes(); lock=json.loads(original); lock["profileVersion"]="1.3.0"; write(self.root,".cvf/gate-profile.lock.json",encode(lock))
        self.assertTrue(self.check()[1]); write(self.root,".cvf/gate-profile.lock.json",original)
        real=gate.os.path.realpath; target=os.path.join(self.root,*gate._REPIN_ADMISSION.split("/"))
        with mock.patch.object(gate.os.path,"realpath",side_effect=lambda p: os.path.join(self.root,"alias") if p==target else real(p)):
            self.assertTrue(self.check()[1])
        write(self.root,"CVF_SESSION/tranches/CCMAI-RUNTIME-139.json",encode({"baseCommit":"0"*40})); self.assertTrue(self.check()[1])

    def test_predecessor_tamper_and_unapproved_file_rejected(self):
        modified=dict(self.old); modified["ownerInstruction"]="rewritten"
        altered=encode(modified); write(self.root,gate._MAINTENANCE_ADMISSION,altered)
        self.admission["predecessor"]["sha256"]=gate._lf_sha(altered); self.save()
        self.assertTrue(self.check()[1])
        write(self.root,gate._MAINTENANCE_ADMISSION,self.oldbytes)
        self.admission["predecessor"]["sha256"]=gate._lf_sha(self.oldbytes)
        path="scripts/cvf_gates/unapproved.py"; write(self.root,path,b"extra\n")
        self.admission["files"][path]=gate._lf_sha(b"extra\n"); self.save()
        self.assertTrue(self.check(self.paths+[path])[1])

    def test_payload_and_authority_staged_together(self):
        git(self.root,"add",gate._REPIN_ADMISSION)
        self.assertTrue(self.check()[1])

    def test_historical_first_commit_payload_tamper(self):
        path = next(p for p in self.old["files"] if p not in gate._REPIN_EXACT and not p.startswith("scripts/cvf_gates/"))
        original = Path(self.root, *path.split("/")).read_bytes()
        write(self.root, path, b"tampered committed history\n")
        git(self.root, "add", path)
        git(self.root, "commit", "--amend", "--no-edit", "-q")
        gate._REPIN_BASE = git(self.root, "rev-parse", "HEAD").decode().strip()
        self.admission["baseCommit"] = gate._REPIN_BASE
        write(self.root, path, original)
        self.save()
        self.assertTrue(any("historical payload differs at first commit" in p for p in self.check()[1]))

    def test_narrowed_claim_and_supplied_scope_rejected_by_actual_range(self):
        removed = self.paths[0]
        del self.admission["files"][removed]
        self.save()
        reduced = [p for p in self.paths if p != removed]
        self.assertTrue(any("actual active range" in p for p in self.check(reduced)[1]))

    def test_committed_successor_deletion_fails_closed(self):
        git(self.root, "add", ".")
        git(self.root, "commit", "-qm", "repin")
        os.unlink(os.path.join(self.root, *gate._REPIN_ADMISSION.split("/")))
        problems = self.check()[1]
        self.assertTrue(any("core repin admission invalid" in p for p in problems))


class CoreRepinTwo(unittest.TestCase):
    """Fixed second successor uses committed snapshots, never current-file stand-ins."""

    inherited_seed_paths = ["scripts/cvf_gates/" + n for n in sorted(gate._REPIN_INSTALLED)[5:]]
    maintenance_fixture_paths = sorted(p for p in gate._MAINTENANCE_PATH_EXACT
                                       if not gate.path_matches(p, gate.ALWAYS_ALLOWED))[:12]

    def save(self):
        write(self.root, gate._REPIN_ADMISSION, encode(self.admission))

    def setUp(self):
        CoreRepin.setUp(self)
        self.original_second_base = gate._REPIN2_BASE
        # Complete a real first successor before preparing the second one.
        self.oldbytes = Path(self.root, gate._MAINTENANCE_ADMISSION).read_bytes()
        self.first_lock = json.loads(Path(self.root, ".cvf/gate-profile.lock.json").read_bytes())
        self.first_lock.update({"profileSha256": self.first_lock["files"]["cvf_downstream_gate_profile.json"],
                                "runnerSha256": self.first_lock["files"]["cvf_downstream_gate_runner.py"],
                                "bundleSha256": "a" * 64})
        write(self.root, ".cvf/gate-profile.lock.json", encode(self.first_lock))
        for p in self.paths:
            self.admission["files"][p] = gate._lf_sha(Path(self.root, *p.split("/")).read_bytes())
        self.save()
        git(self.root, "add", ".")
        git(self.root, "commit", "-qm", "first repin")
        self.first_commit = git(self.root, "rev-parse", "HEAD").decode().strip()
        self.first_bytes = Path(self.root, gate._REPIN_ADMISSION).read_bytes()
        # A distinct committed preparation is required before second BUILD.
        write(self.root, "CVF_SESSION/preparation.txt", b"fixed second preparation\n")
        git(self.root, "add", ".")
        git(self.root, "commit", "-qm", "prepare second")
        gate._REPIN2_BASE = git(self.root, "rev-parse", "HEAD").decode().strip()
        write(self.root, ".cvf/manifest.json", encode({"cvfCoreCommit": gate._REPIN2_CORE_TO}))
        write(self.root, ".cvf/gate-profile.lock.json", encode(dict(self.first_lock, coreCommit=gate._REPIN2_CORE_TO)))
        write(self.root, "scripts/tests/test_cvf_downstream_gate_core_repin.py", b"second test source\n")
        self.second = dict(self.admission, baseCommit=gate._REPIN2_BASE,
                           coreFrom=gate._REPIN_CORE_TO, coreTo=gate._REPIN2_CORE_TO,
                           predecessor={"path": gate._REPIN_ADMISSION, "sha256": gate._lf_sha(self.first_bytes)},
                           files={p: gate._lf_sha(Path(self.root, *p.split("/")).read_bytes()) for p in self.paths})
        self.save_second()

    def tearDown(self):
        gate._REPIN2_BASE = self.original_second_base
        CoreRepin.tearDown(self)

    def save_second(self):
        write(self.root, gate._REPIN2_ADMISSION, encode(self.second))

    def check(self, paths=None):
        result = gate.Result("tranche")
        found = gate.check_maintenance_admission(self.root, "CCMAI-RUNTIME-139", self.seed,
                                                self.paths if paths is None else paths, result)
        return found, result.problems

    def commit_second(self):
        git(self.root, "add", ".")
        git(self.root, "commit", "-qm", "second repin")

    def test_valid_bootstrap_staged_committed_and_snapshot_reuse(self):
        self.assertEqual(len(self.second["files"]), 18)
        self.assertEqual(self.check()[1], [])
        git(self.root, "add", ".")
        self.assertEqual(self.check()[1], [])
        self.commit_second()
        self.assertEqual(self.check()[1], [])
        # Current manifest differs from first snapshot by design, not historical tampering.
        self.assertNotEqual(git(self.root, "show", self.first_commit + ":.cvf/manifest.json"),
                            Path(self.root, ".cvf/manifest.json").read_bytes())

    def test_fixed_identity_fields_and_duplicate_json(self):
        for key, value in (("baseCommit", "0" * 40), ("coreFrom", "1" * 40),
                           ("coreTo", "2" * 40), ("riskCeiling", "R1"), ("extra", True)):
            with self.subTest(key=key):
                original = dict(self.second)
                self.second[key] = value
                self.save_second()
                self.assertTrue(self.check()[1])
                self.second = original
        self.save_second()
        raw = encode(self.second).replace(b'"schemaVersion": "1.0"',
                                         b'"schemaVersion": "1.0", "schemaVersion": "1.0"')
        write(self.root, gate._REPIN2_ADMISSION, raw)
        self.assertTrue(self.check()[1])

    def test_predecessor_and_roles(self):
        self.second["predecessor"]["sha256"] = "0" * 64
        self.save_second()
        self.assertTrue(self.check()[1])
        self.second["predecessor"]["sha256"] = gate._lf_sha(self.first_bytes)
        self.second["roles"] = dict(self.second["roles"], reviewer=self.second["roles"]["worker"])
        self.save_second()
        self.assertTrue(self.check()[1])

    def test_original_seed_and_both_authorities_immutable(self):
        for rel in (self.seed_rel, gate._MAINTENANCE_ADMISSION, gate._REPIN_ADMISSION):
            with self.subTest(path=rel):
                original = Path(self.root, *rel.split("/")).read_bytes()
                value = json.loads(original)
                value["tampered"] = True
                write(self.root, rel, encode(value))
                self.assertTrue(self.check()[1])
                write(self.root, rel, original)

    def test_frozen_installed_workflow_and_lock_metadata(self):
        for path in ("scripts/cvf_gates/cvf_dg_common.py", ".github/workflows/cvf-downstream-gates.yml"):
            with self.subTest(path=path):
                original = Path(self.root, *path.split("/")).read_bytes()
                old_digest = self.second["files"].get(path)
                write(self.root, path, b"changed\n")
                if old_digest is not None:
                    self.second["files"][path] = gate._lf_sha(b"changed\n")
                self.save_second()
                self.assertTrue(self.check()[1])
                write(self.root, path, original)
                if old_digest is not None:
                    self.second["files"][path] = old_digest
        lock = dict(self.first_lock, coreCommit=gate._REPIN2_CORE_TO, bundleSha256="b" * 64)
        write(self.root, ".cvf/gate-profile.lock.json", encode(lock))
        self.second["files"][".cvf/gate-profile.lock.json"] = gate._lf_sha(encode(lock))
        self.save_second()
        self.assertTrue(self.check()[1])

    def test_narrowed_range_new_payload_and_missing_file(self):
        removed = self.paths[0]
        digest = self.second["files"].pop(removed)
        self.save_second()
        self.assertTrue(self.check([p for p in self.paths if p != removed])[1])
        self.second["files"][removed] = digest
        path = "scripts/cvf_gates/not-admitted.py"
        write(self.root, path, b"extra\n")
        self.save_second()
        self.assertTrue(any("actual active range" in p for p in self.check()[1]))
        os.unlink(os.path.join(self.root, *path.split("/")))
        os.unlink(os.path.join(self.root, *removed.split("/")))
        self.assertTrue(self.check()[1])

    def test_bootstrap_staging_atomicity_and_index_mismatch(self):
        git(self.root, "add", gate._REPIN2_ADMISSION)
        self.assertTrue(self.check()[1])
        git(self.root, "add", ".")
        original = Path(self.root, ".cvf/manifest.json").read_bytes()
        write(self.root, ".cvf/manifest.json", b"staged bad\n")
        git(self.root, "add", ".cvf/manifest.json")
        write(self.root, ".cvf/manifest.json", original)
        self.assertTrue(self.check()[1])

    def test_deleted_second_cannot_fall_back_and_later_mutation(self):
        self.commit_second()
        os.unlink(os.path.join(self.root, *gate._REPIN2_ADMISSION.split("/")))
        self.assertTrue(any("second core repin" in p for p in self.check()[1]))
        self.save_second()
        self.second["ownerInstruction"] = "changed"
        self.second["bootstrapException"] = {"reason": "changed", "ownerInstruction": "changed"}
        self.save_second()
        self.assertTrue(any("second repin first blob changed" in p for p in self.check()[1]))

    def test_staged_symlink_mode_even_with_matching_blob(self):
        git(self.root, "add", ".")
        path = ".cvf/manifest.json"
        digest = git(self.root, "rev-parse", ":" + path).decode().strip()
        git(self.root, "update-index", "--cacheinfo", "120000," + digest + "," + path)
        self.assertTrue(any("regular stage-zero" in p for p in self.check()[1]))

    def test_committed_payload_mutation_and_alias(self):
        self.commit_second()
        path = "scripts/tests/test_cvf_downstream_gate_core_repin.py"
        write(self.root, path, b"later mutation\n")
        self.assertTrue(self.check()[1])
        real = gate.os.path.realpath
        target = os.path.join(self.root, *gate._REPIN2_ADMISSION.split("/"))
        with mock.patch.object(gate.os.path, "realpath", side_effect=lambda p: os.path.join(self.root, "alias") if p == target else real(p)):
            self.assertTrue(self.check()[1])

    def test_historical_snapshot_tamper_even_when_present_payload_is_valid(self):
        # Rewrite the first-repin commit and prepare a new second base. A current
        # source cannot serve as replacement evidence for the altered snapshot.
        git(self.root, "reset", "--hard", self.first_commit)
        os.unlink(os.path.join(self.root, *gate._REPIN2_ADMISSION.split("/")))
        path = "scripts/tests/test_cvf_downstream_gate_core_repin.py"
        write(self.root, path, b"tampered first snapshot\n")
        git(self.root, "add", path)
        git(self.root, "commit", "--amend", "--no-edit", "-q")
        gate._REPIN2_BASE = git(self.root, "rev-parse", "HEAD").decode().strip()
        self.second["baseCommit"] = gate._REPIN2_BASE
        self.save_second()
        self.assertTrue(any("historical first-commit payload differs" in p for p in self.check()[1]))


if __name__ == "__main__":
    unittest.main()
