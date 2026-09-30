"""CCMAI-GOV-001: positive and negative fixtures for scripts/cvf_downstream_gate.py.

Every negative case first proves the same fixture passes (baseline) and then applies one
mutation, so a failure is attributable to that mutation and not to a broken fixture.
Synthetic credential-like strings are assembled at runtime so this file never contains one.
"""
import contextlib
import io
import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.dirname(HERE))
import cvf_downstream_gate as gate  # noqa: E402

try:
    import yaml  # noqa: F401
    HAVE_YAML = True
except ImportError:
    HAVE_YAML = False

HANDOFF = "CVF_SESSION/handoffs/AGENT_HANDOFF_V1_2026-09-26.md"
TID = "CCMAI-TEST-001"
SHA_A = "a" * 40
NEXT_MOVE = f"{TID} BUILD is REVIEW_PENDING; the reviewer checks the exact commit. No push or FREEZE."
ROLE = "COMMIT_STEWARD (Worker) -> REVIEWER (Reviewer) next"

WF_ANY = "name: {n}\non:\n  pull_request:\n  push:\n    branches: [main]\njobs:\n  j:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo ok\n"
WF_PATHS = "name: {n}\non:\n  pull_request:\n    paths: [{p}]\njobs:\n  j:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo ok\n"


def write(root, rel, text):
    path = os.path.join(root, *rel.split("/"))
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8", newline="\n") as fh:
        fh.write(text)


def write_json(root, rel, obj):
    write(root, rel, json.dumps(obj, indent=2) + "\n")


def record(**over):
    rec = {
        "schemaVersion": "1.0", "trancheId": TID, "workOrder": "docs/work_orders/CCMAI_TEST_001.md",
        "status": "REVIEW_PENDING", "phase": "REVIEW", "riskCeiling": "R2",
        "roles": {"workOrderAuthor": "Reviewer", "implementationWorker": "Worker", "commitSteward": "Worker", "reviewer": "Reviewer"},
        "baseCommit": None, "allowedPaths": ["tools/**", "docs/work_orders/CCMAI_TEST_001.md"],
        "prohibitedEffects": ["push", "freeze"], "buildCommit": None, "reviewEvidence": [], "disposition": None,
        "freeze": "OPEN", "history": ["DISPATCH_READY", "BUILD", "REVIEW_PENDING"],
    }
    rec.update(over)
    return rec


def make_project(root):
    """A minimal project that passes every gate except catalog (skipped in these tests)."""
    write_json(root, ".cvf/manifest.json", {
        "schemaVersion": "2.0", "cvfCoreRepository": "https://example.invalid/core.git", "cvfCoreCommit": SHA_A,
        "cvfCoreRelativePath": "../core", "phaseModel": gate.PHASES, "liveGovernanceEvidenceRequired": True,
        "mockAllowedOnlyForUi": True, "requiredDocs": [".cvf/manifest.json", "CVF_SESSION_MEMORY.md", "..\\WORKSPACE_RULES.md"],
    })
    write_json(root, ".cvf/policy.json", {"liveGovernanceEvidenceRequired": True, "mockAllowedOnlyForUi": True,
                                          "workspaceIsolationRequired": True, "phaseTransitionRequired": True, "riskCeiling": "R2"})
    write(root, "AGENTS.md", "# rules\n")
    write_json(root, "CVF_SESSION/ACTIVE_SESSION_STATE.json", {
        "currentMode": "REVIEW", "activePhase": "REVIEW", "activeHandoff": HANDOFF, "nextAllowedMove": NEXT_MOVE,
        "parkedOperatorCheckpoint": None, "activeRole": ROLE})
    write(root, HANDOFF, "# Handoff\n\n## Current State\n\n- Project: T\n- Current mode: REVIEW\n- Active phase: REVIEW (" + TID + ")\n"
          f"- Active role: {ROLE}\n- Next allowed move: {NEXT_MOVE}\n- Parked operator checkpoint: none\n")
    marker = {"currentMode": "REVIEW", "activePhase": "REVIEW", "activeHandoff": HANDOFF, "activeTranche": TID, "parked": False}
    write(root, "CVF_SESSION_MEMORY.md", "# Memory\n\n<!-- cvf-front-marker " + json.dumps(marker) + " -->\n")
    write_json(root, "IMPLEMENTATION_STATUS.json", {"currentPhase": "REVIEW"})
    write_json(root, f"CVF_SESSION/tranches/{TID}.json", record())
    write(root, "docs/work_orders/CCMAI_TEST_001.md", "# wo\n")
    write(root, ".github/workflows/governance.yml", WF_ANY.format(n="gov"))
    write(root, ".github/workflows/backend.yml", WF_PATHS.format(n="be", p="'backend/**', 'scripts/ci_db_test_gate.py'"))
    write(root, ".github/workflows/frontend.yml", WF_PATHS.format(n="fe", p="'frontend/**'"))
    write(root, ".github/workflows/docs.yml", WF_PATHS.format(n="docs", p="'docs/**'"))


def run_gate(root, *argv, files=None):
    args = ["preflight", "--root", root, "--skip-catalog", *argv]
    if files is not None:
        args += ["--files", *files]
    out = io.StringIO()
    with contextlib.redirect_stdout(out), contextlib.redirect_stderr(io.StringIO()):
        code = gate.main(args)
    return code, out.getvalue()


class Base(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.mkdtemp(prefix="cvfgate-")
        self.root = os.path.join(self.tmp, "p")
        os.makedirs(self.root)
        make_project(self.root)

    def tearDown(self):
        shutil.rmtree(self.tmp, ignore_errors=True)

    def edit_json(self, rel, fn):
        path = os.path.join(self.root, *rel.split("/"))
        with open(path, encoding="utf-8") as fh:
            data = json.load(fh)
        fn(data)
        write_json(self.root, rel, data)

    def edit_text(self, rel, old, new):
        path = os.path.join(self.root, *rel.split("/"))
        with open(path, encoding="utf-8") as fh:
            text = fh.read()
        self.assertIn(old, text)
        write(self.root, rel, text.replace(old, new, 1))

    def gate_ok(self, only, files=()):
        code, out = run_gate(self.root, "--only", only, files=list(files))
        self.assertEqual(code, 0, f"baseline for {only} should pass:\n{out}")

    def assert_fails(self, only, needle, files=()):
        code, out = run_gate(self.root, "--only", only, files=list(files))
        self.assertEqual(code, 1, out)
        self.assertIn(needle, out)
        return out


class ProvenanceAndContinuity(Base):
    def test_baseline_passes_every_non_catalog_gate(self):
        if not HAVE_YAML:
            self.skipTest("PyYAML not installed")
        code, out = run_gate(self.root, files=["tools/x.py"])
        write(self.root, "tools/x.py", "print(1)\n")
        code, out = run_gate(self.root, files=["tools/x.py"])
        self.assertEqual(code, 0, out)
        self.assertIn("BOOTSTRAP_MIGRATION_PENDING", out)

    def test_observed_drift_stale_memory_front_pointer_fails(self):
        self.gate_ok("continuity")
        self.edit_text("CVF_SESSION_MEMORY.md", '"currentMode": "REVIEW"', '"currentMode": "WORK_ORDER"')
        self.assert_fails("continuity", "memory marker currentMode 'WORK_ORDER' != state 'REVIEW'")

    def test_observed_drift_stale_status_phase_fails(self):
        self.gate_ok("continuity")
        self.edit_json("IMPLEMENTATION_STATUS.json", lambda d: d.update(currentPhase="WORK_ORDER"))
        self.assert_fails("continuity", "IMPLEMENTATION_STATUS.currentPhase 'WORK_ORDER' != state.activePhase 'REVIEW'")

    def test_missing_marker_and_malformed_marker_fail(self):
        self.gate_ok("continuity")
        self.edit_text("CVF_SESSION_MEMORY.md", "<!-- cvf-front-marker", "<!-- cvf-front-markerX")
        self.assert_fails("continuity", "no machine-readable")
        write(self.root, "CVF_SESSION_MEMORY.md", "# Memory\n\n<!-- cvf-front-marker {not json} -->\n")
        self.assert_fails("continuity", "marker is not valid JSON")

    def test_handoff_path_missing_or_escaping_fails(self):
        self.gate_ok("continuity")
        self.edit_json("CVF_SESSION/ACTIVE_SESSION_STATE.json", lambda d: d.update(activeHandoff="CVF_SESSION/handoffs/none.md"))
        self.assert_fails("continuity", "active handoff not found")
        self.edit_json("CVF_SESSION/ACTIVE_SESSION_STATE.json", lambda d: d.update(activeHandoff="../../outside.md"))
        self.assert_fails("continuity", "escapes the project root")

    def test_handoff_header_drift_fails(self):
        self.gate_ok("continuity")
        self.edit_text(HANDOFF, "- Current mode: REVIEW", "- Current mode: BUILD")
        self.assert_fails("continuity", "handoff Current mode 'BUILD' != state.currentMode 'REVIEW'")
        self.edit_text(HANDOFF, "- Current mode: BUILD", "- Current mode: REVIEW")
        self.edit_text(HANDOFF, TID + " BUILD is REVIEW_PENDING", "CCMAI-OTHER-009 BUILD is REVIEW_PENDING")
        self.assert_fails("continuity", "next-move tranche IDs differ")

    def test_parked_checkpoint_must_agree(self):
        self.gate_ok("continuity")
        self.edit_json("CVF_SESSION/ACTIVE_SESSION_STATE.json", lambda d: d.update(parkedOperatorCheckpoint="R9 parked at BUILD `abc1234`"))
        self.assert_fails("continuity", "parked checkpoint declared in only one")
        self.edit_text(HANDOFF, "checkpoint: none", "checkpoint: R9 parked at BUILD `abc1234`")
        self.assert_fails("continuity", "memory marker parked flag differs")  # marker still says parked false

    def test_present_bootstrap_model_must_match(self):
        self.gate_ok("continuity")
        write_json(self.root, "CVF_SESSION/ACTIVE_SESSION_BOOTSTRAP_READ_MODEL.json", {"currentMode": "REVIEW", "activeHandoff": HANDOFF})
        self.gate_ok("continuity")
        write_json(self.root, "CVF_SESSION/ACTIVE_SESSION_BOOTSTRAP_READ_MODEL.json", {"currentMode": "BUILD", "activeHandoff": HANDOFF})
        self.assert_fails("continuity", "bootstrap read model currentMode differs")

    def test_provenance_failures(self):
        self.gate_ok("provenance")
        self.edit_json(".cvf/manifest.json", lambda d: d.update(phaseModel=["INTAKE", "BUILD"]))
        self.assert_fails("provenance", "canonical INTAKE..FREEZE")
        make_project(self.root)
        self.edit_json(".cvf/policy.json", lambda d: d.update(mockAllowedOnlyForUi=False))
        self.assert_fails("provenance", "policy.mockAllowedOnlyForUi must be true")
        make_project(self.root)
        self.edit_json(".cvf/manifest.json", lambda d: d["requiredDocs"].append("docs/missing.md"))
        self.assert_fails("provenance", "required doc missing: docs/missing.md")
        self.edit_json(".cvf/manifest.json", lambda d: d["requiredDocs"].__setitem__(-1, "docs/../../escape.md"))
        self.assert_fails("provenance", "escapes project root")
        make_project(self.root)
        self.edit_json(".cvf/manifest.json", lambda d: d.update(cvfCoreCommit="main"))
        self.assert_fails("provenance", "40-hex")

    def test_malformed_or_missing_inputs_fail_closed(self):
        write(self.root, "CVF_SESSION/ACTIVE_SESSION_STATE.json", "{ not json")
        self.assert_fails("continuity", "not valid JSON")
        shutil.rmtree(os.path.join(self.root, ".cvf"))
        self.assert_fails("provenance", "manifest missing or not valid JSON")


class TrancheContract(Base):
    def test_path_scope(self):
        write(self.root, "tools/ok.py", "x = 1\n")
        write(self.root, "backend/engine/x.go", "package engine\n")
        self.gate_ok("tranche", ["tools/ok.py", "CVF_SESSION_MEMORY.md", "docs/reviews/A_BUILD_1.md"])
        out = self.assert_fails("tranche", "outside the active tranche's allowedPaths: backend/engine/x.go", ["tools/ok.py", "backend/engine/x.go"])
        self.assertNotIn("tools/ok.py", out)

    def test_r2_worker_self_review_fails(self):
        self.gate_ok("tranche")
        self.edit_json(f"CVF_SESSION/tranches/{TID}.json", lambda d: d["roles"].update(reviewer="Worker"))
        self.assert_fails("tranche", "reviewer must be independent")
        self.edit_json(f"CVF_SESSION/tranches/{TID}.json", lambda d: d["roles"].update(reviewer="Reviewer", repairWorker="Reviewer"))
        self.assert_fails("tranche", "reviewer must be independent")
        self.edit_json(f"CVF_SESSION/tranches/{TID}.json", lambda d: d.update(riskCeiling="R1"))
        self.edit_json(f"CVF_SESSION/tranches/{TID}.json", lambda d: d["roles"].update(reviewer="Worker", repairWorker="Worker"))
        self.gate_ok("tranche")  # the independence rule is R2+

    def test_review_pass_needs_evidence_and_build_commit(self):
        self.gate_ok("tranche")
        self.edit_json(f"CVF_SESSION/tranches/{TID}.json", lambda d: d.update(
            status="REVIEW_PASS", disposition="REVIEW_PASS", history=["DISPATCH_READY", "BUILD", "REVIEW_PENDING", "REVIEW_PASS"]))
        out = self.assert_fails("tranche", "requires reviewEvidence")
        self.assertIn("40-hex buildCommit", out)
        write(self.root, "docs/reviews/R.md", "review\n")
        self.edit_json(f"CVF_SESSION/tranches/{TID}.json", lambda d: d.update(buildCommit=SHA_A, reviewEvidence=["docs/reviews/R.md"]))
        self.gate_ok("tranche")
        self.edit_json(f"CVF_SESSION/tranches/{TID}.json", lambda d: d.update(reviewEvidence=["docs/reviews/gone.md"]))
        self.assert_fails("tranche", "review evidence missing")

    def test_freeze_claims_are_rejected(self):
        self.gate_ok("tranche")
        self.edit_json(f"CVF_SESSION/tranches/{TID}.json", lambda d: d.update(freeze="FROZEN"))
        self.assert_fails("tranche", "FROZEN needs status FROZEN")
        self.edit_json(f"CVF_SESSION/tranches/{TID}.json", lambda d: d.update(freeze="OPEN", status="FROZEN", phase="FREEZE", disposition="REVIEW_PASS", closer="Reviewer"))
        self.assert_fails("tranche", "contradicts freeze")

    def test_invalid_transition_and_phase_inconsistency(self):
        self.gate_ok("tranche")
        self.edit_json(f"CVF_SESSION/tranches/{TID}.json", lambda d: d.update(history=["DISPATCH_READY", "FROZEN", "REVIEW_PENDING"]))
        self.assert_fails("tranche", "invalid status transition DISPATCH_READY -> FROZEN")
        self.edit_json(f"CVF_SESSION/tranches/{TID}.json", lambda d: d.update(history=["BUILD", "REVIEW_PENDING"], phase="BUILD"))
        self.assert_fails("tranche", "inconsistent with status REVIEW_PENDING")
        self.edit_json(f"CVF_SESSION/tranches/{TID}.json", lambda d: d.update(phase="REVIEW"))
        self.edit_json("CVF_SESSION/ACTIVE_SESSION_STATE.json", lambda d: d.update(activePhase="BUILD", currentMode="BUILD"))
        self.assert_fails("tranche", "!= active session phase 'BUILD'")

    def test_unbound_work_order_and_missing_record(self):
        write(self.root, "docs/work_orders/CCMAI_NEW_002.md", "# new\n")
        self.assert_fails("tranche", "changed work order has no tranche record binding it: docs/work_orders/CCMAI_NEW_002.md",
                          ["docs/work_orders/CCMAI_NEW_002.md"])
        os.remove(os.path.join(self.root, "CVF_SESSION", "tranches", TID + ".json"))
        self.assert_fails("tranche", "has no CVF_SESSION/tranches")

    def test_r3_requires_formal_approval_and_required_fields(self):
        self.gate_ok("tranche")
        self.edit_json(f"CVF_SESSION/tranches/{TID}.json", lambda d: d.update(riskCeiling="R3"))
        self.assert_fails("tranche", "R3 requires formalApproval")
        self.edit_json(f"CVF_SESSION/tranches/{TID}.json", lambda d: d.update(riskCeiling="R2", allowedPaths=[], prohibitedEffects=[]))
        out = self.assert_fails("tranche", "allowedPaths must be a non-empty list")
        self.assertIn("prohibitedEffects must be a non-empty list", out)


class ClaimBoundary(Base):
    def test_build_evidence_cannot_claim_freeze(self):
        write(self.root, "docs/reviews/X_BUILD_2026.md", "Status: REVIEW_PENDING\n")
        self.gate_ok("claims", ["docs/reviews/X_BUILD_2026.md"])
        write(self.root, "docs/reviews/X_BUILD_2026.md", "Status: FROZEN after build\n")
        self.assert_fails("claims", "claims a frozen/closed state", ["docs/reviews/X_BUILD_2026.md"])

    def test_governance_claim_needs_real_provider_receipt(self):
        doc = "docs/reviews/Y_BUILD_2026.md"
        write(self.root, doc, "Result: CVF governs the agent runtime for this flow.\n")
        self.assert_fails("claims", "without a real-provider receipt", [doc])
        write(self.root, doc, "No CVF governance claim is made; CVF does not control the agent runtime here.\n")
        self.gate_ok("claims", [doc])  # negated wording is not a claim
        write(self.root, doc, "Result: CVF governs the agent runtime for this flow.\n")
        receipt = {"providerCall": True, "provider": "p", "model": "m", "request": "req", "response": "resp"}
        write_json(self.root, "docs/reviews/receipt.json", receipt)
        self.edit_json(f"CVF_SESSION/tranches/{TID}.json", lambda d: d.update(governanceReceipt="docs/reviews/receipt.json"))
        self.gate_ok("claims", [doc])
        for bad in ({"synthetic": True}, {"mock": True}, {"providerCall": False}, {"response": ""}):
            write_json(self.root, "docs/reviews/receipt.json", {**receipt, **bad})
            self.assert_fails("claims", "without a real-provider receipt", [doc])

    def test_actions_result_claim_needs_well_formed_run_record(self):
        doc = "docs/reviews/Z_2026.md"
        write(self.root, doc, "The GitHub Actions run passed for the backend gate.\n")
        self.assert_fails("claims", "claims a GitHub Actions result without tranche.ciRun", [doc])
        write(self.root, doc, "Local gate REVIEW_PASS; the GitHub Actions result is pending and unverified.\n")
        self.gate_ok("claims", [doc])
        write(self.root, doc, "The GitHub Actions run passed for the backend gate.\n")
        good = {"url": "https://github.com/o/r/actions/runs/123", "sha": SHA_A, "conclusion": "success"}
        self.edit_json(f"CVF_SESSION/tranches/{TID}.json", lambda d: d.update(ciRun=good))
        code, out = run_gate(self.root, "--only", "claims", files=[doc])
        self.assertEqual(code, 0, out)
        self.assertIn("NOT verified by this gate", out)
        for bad in ({"url": "http://example.com/x"}, {"sha": "abc"}, {"conclusion": "failure"}):
            self.edit_json(f"CVF_SESSION/tranches/{TID}.json", lambda d, b=bad: d.update(ciRun={**good, **b}))
            self.assert_fails("claims", "without tranche.ciRun", [doc])


class SecretHygiene(Base):
    def test_credential_files_and_literals_fail_without_printing_values(self):
        aws = "AK" + "IA" + "0123456789ABCDEF"
        pem = "-----BEGIN " + "PRIVATE" + " KEY-----"
        literal = "api_key = \"" + "Ab1" * 10 + "\""
        write(self.root, "src/ok.py", "value = 'nothing secret'\n")
        self.gate_ok("secrets", ["src/ok.py"])
        write(self.root, "src/aws.py", f"K = '{aws}'\n")
        write(self.root, "src/k.txt", pem + "\nMIIB\n")
        write(self.root, "src/lit.py", literal + "\n")
        for name, needle in (("src/aws.py", "aws-access-key-id"), ("src/k.txt", "private-key-block"), ("src/lit.py", "quoted-credential-literal")):
            out = self.assert_fails("secrets", needle, [name])
            self.assertNotIn(aws, out)
            self.assertNotIn("Ab1Ab1", out)
        write(self.root, ".env", "X=1\n")
        write(self.root, "deploy/prod.env.key", "x\n")
        write(self.root, ".env.example", "X=\n")
        self.assert_fails("secrets", "forbidden credential-type file in the changed set: .env", [".env"])
        self.assert_fails("secrets", "forbidden credential-type file", ["deploy/prod.env.key"])
        self.gate_ok("secrets", [".env.example"])

    def test_allow_marker_binary_and_unread_untracked(self):
        aws = "AK" + "IA" + "0123456789ABCDEF"
        write(self.root, "t/fixture.py", f"K = '{aws}'  # cvf-allow-secret-fixture\n")
        self.gate_ok("secrets", ["t/fixture.py"])
        with open(os.path.join(self.root, "bin.dat"), "wb") as fh:
            fh.write(b"\0\0" + aws.encode())
        self.gate_ok("secrets", ["bin.dat"])
        write(self.root, "CVF_SESSION/LOCAL_PROVIDER_SECRETS.json", '{"k": "' + aws + '"}\n')
        self.gate_ok("secrets", ["tools/a.py"])  # not in the changed set, so never read


@unittest.skipUnless(HAVE_YAML, "PyYAML not installed")
class WorkflowCoverage(Base):
    def test_baseline_and_bypass_mutations(self):
        self.gate_ok("workflows")
        # a governance workflow that ignores authority paths is a bypass
        write(self.root, ".github/workflows/governance.yml", WF_PATHS.format(n="gov", p="'backend/**'"))
        out = self.assert_fails("workflows", "changed path class 'cvf-authority'")
        self.assertIn("'session-memory'", out)
        make_project(self.root)
        os.remove(os.path.join(self.root, ".github", "workflows", "frontend.yml"))
        self.assert_fails("workflows", "changed path class 'frontend'")
        make_project(self.root)
        write(self.root, ".github/workflows/backend.yml", WF_PATHS.format(n="be", p="'backend/**'"))
        self.gate_ok("workflows")  # governance covers the CI-gate helper on every PR

    def test_push_only_workflow_does_not_count(self):
        write(self.root, ".github/workflows/frontend.yml", "name: fe\non:\n  push:\n    branches: [main]\njobs:\n  j:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo\n")
        self.assert_fails("workflows", "changed path class 'frontend'")

    def test_real_repository_workflows_cover_every_class(self):
        repo = os.path.dirname(os.path.dirname(HERE))
        if not os.path.isdir(os.path.join(repo, ".github", "workflows")):
            self.skipTest("not a full checkout")
        res = gate.gate_workflows(repo, {})
        self.assertTrue(res.ok, res.problems)


class Cli(Base):
    def test_unknown_gate_and_bad_root(self):
        code, _ = run_gate(self.root, "--only", "nope", files=[])
        self.assertEqual(code, 2)
        out = io.StringIO()
        with contextlib.redirect_stdout(out), contextlib.redirect_stderr(io.StringIO()):
            self.assertEqual(gate.main(["preflight", "--root", os.path.join(self.tmp, "absent")]), 2)

    def test_subprocess_exit_codes(self):
        script = os.path.join(os.path.dirname(HERE), "cvf_downstream_gate.py")
        ok = subprocess.run([sys.executable, script, "preflight", "--root", self.root, "--skip-catalog", "--only", "provenance,continuity", "--files"],
                            capture_output=True, text=True)
        self.assertEqual(ok.returncode, 0, ok.stdout)
        self.edit_json("IMPLEMENTATION_STATUS.json", lambda d: d.update(currentPhase="WORK_ORDER"))
        bad = subprocess.run([sys.executable, script, "preflight", "--root", self.root, "--skip-catalog", "--only", "continuity", "--files"],
                             capture_output=True, text=True)
        self.assertEqual(bad.returncode, 1)


HAVE_GIT = shutil.which("git") is not None


@unittest.skipUnless(HAVE_GIT, "git not installed")
class GitRange(Base):
    def git(self, *a):
        subprocess.run(["git", "-C", self.root, "-c", "user.name=t", "-c", "user.email=t@example.invalid", *a],
                       check=True, capture_output=True)

    def test_scope_uses_base_commit_and_pr_range_finds_secret_files(self):
        self.git("init", "-q")
        self.git("add", "-A")
        self.git("commit", "-q", "-m", "base")
        base = subprocess.run(["git", "-C", self.root, "rev-parse", "HEAD"], capture_output=True, text=True).stdout.strip()
        self.edit_json(f"CVF_SESSION/tranches/{TID}.json", lambda d: d.update(baseCommit=base))
        self.git("add", "-A")
        self.git("commit", "-q", "-m", "record")
        write(self.root, "tools/in_scope.py", "x = 1\n")
        self.git("add", "-A")
        self.git("commit", "-q", "-m", "in scope")
        code, out = run_gate(self.root, "--only", "tranche")
        self.assertEqual(code, 0, out)
        write(self.root, "backend/out_of_scope.go", "package x\n")  # uncommitted: the worktree counts
        code, out = run_gate(self.root, "--only", "tranche")
        self.assertEqual(code, 1, out)
        self.assertIn("backend/out_of_scope.go", out)
        self.git("add", "-A")
        self.git("commit", "-q", "-m", "out of scope")
        code, out = run_gate(self.root, "--only", "tranche")
        self.assertEqual(code, 1, out)  # committed out-of-scope work is still caught via base..HEAD
        write(self.root, ".env", "A=1\n")
        self.git("add", "-f", ".env")
        self.git("commit", "-q", "-m", "env")
        code, out = run_gate(self.root, "--only", "secrets", "--base", base, "--head", "HEAD")
        self.assertEqual(code, 1, out)
        self.assertIn("forbidden credential-type file in the changed set: .env", out)


if __name__ == "__main__":
    unittest.main()
