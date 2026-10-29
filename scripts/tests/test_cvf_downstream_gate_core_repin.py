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
        self.seed = {"riskCeiling": "R2", "allowedPaths": []}
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
        paths = sorted(gate._MAINTENANCE_PATH_EXACT)[:12] + ["scripts/cvf_gates/" + n for n in sorted(gate._REPIN_INSTALLED)[:5]]
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
        self.paths = sorted(p for p in changed if not gate.path_matches(p, gate.ALWAYS_ALLOWED))
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


if __name__ == "__main__":
    unittest.main()
