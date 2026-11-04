"""Exact learning-profile successor tests on disposable committed Git histories."""
import hashlib
import json
import os
from pathlib import Path
import unittest

import test_cvf_downstream_gate_core_repin as fixture

gate = fixture.gate


class LawProfile(unittest.TestCase):
    def setUp(self):
        self.f = fixture.CoreRepinTwo(methodName="test_valid_bootstrap_staged_committed_and_snapshot_reuse")
        self.f.setUp()
        self.f.commit_second()
        self.old_base = gate._LAW_BASE
        gate._LAW_BASE = fixture.git(self.f.root, "rev-parse", "HEAD").decode().strip()
        self.root = self.f.root
        self.second_bytes = Path(self.root, gate._REPIN2_ADMISSION).read_bytes()
        self.seed_bytes = Path(self.root, self.f.seed_rel).read_bytes()
        for name in ("cvf_repair_convergence.py", "cvf_milestone_coverage.py"):
            fixture.write(self.root, "scripts/cvf_gates/" + name, Path("scripts/cvf_gates", name).read_bytes())
        self.registry = {"schemaVersion":"cvf.milestoneAuthority@1.1.0", "milestones":{"M3":{}}, "repairControls":{}}
        fixture.write(self.root, "docs/milestone-coverage-authority.json", fixture.encode(self.registry))
        hashes = {name: gate._lf_sha(Path(self.root, "scripts/cvf_gates", name).read_bytes()) for name in gate._LAW_INSTALLED}
        bundle = hashlib.sha256("".join(f"{n}:{hashes[n]}\n" for n in sorted(hashes)).encode()).hexdigest()
        lock = {"profileId":"cvf.downstreamGateProfile@1.3.0", "profileVersion":"1.3.0", "coreCommit":"UNKNOWN", "gatesDir":"scripts/cvf_gates", "files":hashes,
                "profileSha256":hashes["cvf_downstream_gate_profile.json"], "runnerSha256":hashes["cvf_downstream_gate_runner.py"], "bundleSha256":bundle}
        fixture.write(self.root, ".cvf/gate-profile.lock.json", fixture.encode(lock))
        fixture.write(self.root, ".github/workflows/cvf-downstream-gates.yml", (" ".join((lock["profileSha256"], lock["runnerSha256"], bundle))+"\n").encode())
        fixture.write(self.root, "scripts/tests/test_cvf_downstream_gate_law_profile.py", b"new tests\n")
        changed = set(gate.git_z(self.root, "diff", "--name-only", "-z", gate._MAINTENANCE_BASE + "..HEAD")) | set(gate.worktree_files(self.root))
        self.paths = sorted(p for p in changed if not gate.path_matches(p, self.f.seed["allowedPaths"] + gate.ALWAYS_ALLOWED))
        instruction = "Owner 2026-11-05 directs profile migration"
        self.admission = {"schemaVersion":"1.0", "trancheId":"CCMAI-RUNTIME-139", "purpose":"STATIC_LAW_PROFILE_1_3_MIGRATION_ONLY", "baseCommit":gate._LAW_BASE,
                          "predecessor":{"path":gate._REPIN2_ADMISSION, "sha256":gate._lf_sha(self.second_bytes)}, "seedSha256":gate._lf_sha(self.seed_bytes), "riskCeiling":"R2",
                          "roles":{"worker":"Worker", "reviewer":"Reviewer", "commitSteward":"Root"}, "coreFrom":gate._REPIN2_CORE_TO, "coreTo":"UNKNOWN",
                          "files":{p:gate._lf_sha(Path(self.root, *p.split("/")).read_bytes()) for p in self.paths},
                          "ownerInstruction":instruction, "bootstrapException":{"reason":"same commit", "ownerInstruction":instruction}}
        self.save()

    def tearDown(self):
        gate._LAW_BASE = self.old_base
        self.f.tearDown()

    def save(self):
        fixture.write(self.root, gate._LAW_ADMISSION, fixture.encode(self.admission))

    def check(self):
        result = gate.Result("tranche")
        gate.check_maintenance_admission(self.root, "CCMAI-RUNTIME-139", self.f.seed, self.paths, result)
        return result.problems

    def test_bootstrap_committed_and_deleted_admission(self):
        self.assertEqual(self.check(), [])
        fixture.git(self.root, "add", ".")
        self.assertEqual(self.check(), [])
        fixture.git(self.root, "commit", "-qm", "law profile")
        self.assertEqual(self.check(), [])
        os.remove(Path(self.root, gate._LAW_ADMISSION))
        self.assertTrue(self.check())

    def test_payload_and_historical_authority_tampering(self):
        p = next(p for p in self.f.second["files"] if p not in gate._LAW_REPLACE and not p.startswith("scripts/cvf_gates/"))
        fixture.write(self.root, p, b"unauthorized\n")
        self.admission["files"][p] = gate._lf_sha(b"unauthorized\n")
        self.save()
        self.assertTrue(self.check())
        fixture.write(self.root, p, fixture.git(self.root, "show", "HEAD:" + p))
        self.admission["files"][p] = self.f.second["files"][p]
        self.save()
        fixture.write(self.root, gate._REPIN2_ADMISSION, self.second_bytes + b" ")
        self.admission["predecessor"]["sha256"] = gate._lf_sha(self.second_bytes + b" ")
        self.save()
        self.assertTrue(self.check())

    def test_scope_lock_and_ci_tampering(self):
        self.admission["files"][self.paths[0]] = "0" * 64
        self.save()
        self.assertTrue(self.check())
        self.admission["files"][self.paths[0]] = gate._lf_sha(Path(self.root, *self.paths[0].split("/")).read_bytes())
        self.save()
        fixture.write(self.root, "scripts/cvf_gates/cvf_repair_convergence.py", b"changed\n")
        self.assertTrue(self.check())

    def test_staged_old_authority_tamper(self):
        fixture.git(self.root, "add", ".")
        path = gate._REPIN_ADMISSION
        original = Path(self.root, path).read_bytes()
        fixture.write(self.root, path, original + b" ")
        fixture.git(self.root, "add", path)
        fixture.write(self.root, path, original)
        self.assertTrue(self.check())

    def test_native_repair_registry_evolution_and_reset(self):
        fixture.git(self.root, "add", ".")
        fixture.git(self.root, "commit", "-qm", "law profile")
        evidence_path = "docs/reviews/probes/r139_integrated_20261021/attempt-evidence.txt"
        attempt_path = "docs/reviews/probes/r139_integrated_20261021/attempt.json"
        evidence = b"observed failure\n"
        fixture.write(self.root, evidence_path, evidence)
        attempt = {"schemaVersion":"cvf.repairAttempt@1.0.0", "milestoneId":"M3", "criterionIds":["M3-05"], "trancheId":"R139", "activity":"REPAIR", "result":"UNMET", "proofState":"UNPROVED", "elapsedMinutes":4,
                   "evidence":[{"path":evidence_path,"sha256":gate._lf_sha(evidence)}]}
        attempt_bytes = fixture.encode(attempt)
        fixture.write(self.root, attempt_path, attempt_bytes)
        row = {"history":[{"path":attempt_path,"sha256":gate._lf_sha(attempt_bytes)}], "affectedCriteria":["M3-05"], "audit":None}
        self.registry["repairControls"]["M3"] = row
        registry_path = "docs/milestone-coverage-authority.json"
        fixture.write(self.root, registry_path, fixture.encode(self.registry))
        self.assertEqual(self.check(), [])
        fixture.git(self.root, "add", ".")
        fixture.git(self.root, "commit", "-qm", "enroll failure")
        self.assertEqual(self.check(), [])
        self.registry["repairControls"] = {}
        fixture.write(self.root, registry_path, fixture.encode(self.registry))
        self.assertTrue(self.check())
        self.registry["repairControls"]["M3"] = row
        fixture.write(self.root, registry_path, fixture.encode(self.registry))
        fixture.git(self.root, "add", registry_path)
        fixture.write(self.root, registry_path, fixture.encode(dict(self.registry, repairControls={})))
        self.assertTrue(self.check())

    def test_roles_base_and_duplicate_json_rejected(self):
        self.admission["roles"]["reviewer"] = "Worker"
        self.save()
        self.assertTrue(self.check())
        self.admission["roles"]["reviewer"] = "Reviewer"
        self.admission["baseCommit"] = "0" * 40
        self.save()
        self.assertTrue(self.check())


if __name__ == "__main__":
    unittest.main()

