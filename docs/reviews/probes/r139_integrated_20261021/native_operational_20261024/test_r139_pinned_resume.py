"""Pure offline tests for the R139 code-as-data resume contract."""

import importlib.machinery
import importlib.util
import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

HERE = Path(__file__).resolve().parent


def load(name):
    path = HERE / (name + ".py.txt")
    loader = importlib.machinery.SourceFileLoader(name, str(path))
    spec = importlib.util.spec_from_loader(name, loader)
    module = importlib.util.module_from_spec(spec)
    loader.exec_module(module)
    return module


resume = load("r139_durable_resume")
preflight = load("r139_pinned_preflight")
NOW = "2026-10-26T00:00:00+00:00"
LATER = "2026-10-27T00:00:00+00:00"


class ResumeTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.source = self.root / "frozen-source.txt"
        self.source.write_bytes(b"frozen")
        self.path = self.root / "ledger.json"
        self.files = {self.source: resume.file_digest(self.source)}
        self.binding = {"order": "R139", "source": "source-commit", "contract": "contract-v3",
                        "pins": {"core": preflight.PIN, "profile": preflight.PROFILE},
                        "inputs": {str(self.source): self.files[self.source]},
                        "checkpointHashes": {}, "ledgerPath": str(self.path.resolve()),
                        "campaignId": "R139-clock-resume-repair"}
        self.limits = {"offlineChecks": 2, "providerCalls": 1}
        self.input = self.files[self.source]

    def ledger(self, deadline=LATER):
        return resume.ResumeLedger(self.path, binding=self.binding,
                                   source_files=self.files, limits=self.limits,
                                   deadline_utc=deadline)

    def receipt(self, stage_id, *, external=False, intent=None, result=b"original"):
        result_path = self.root / (stage_id + "-result.bin")
        result_path.write_bytes(result)
        return resume.canonical({"schema": resume.RECEIPT_SCHEMA,
                "stageId": stage_id,
                "bindingSha256": resume.digest(resume.canonical(self.binding)),
                "inputSha256": self.input, "kind": "external" if external else "offline",
                "intentSha256": intent, "originalReturnObserved": True,
                "resultRef": {"path": str(result_path), "sha256": resume.digest(result)}})

    def test_complete_reused_without_effect_or_budget_reset(self):
        with self.ledger() as ledger:
            ledger.plan("check", kind="offline", cost={"offlineChecks": 1}, input_sha256=self.input)
            ledger.reserve("check", now_utc=NOW)
            ledger.complete("check", receipt_path=self.root / "receipt.bin", receipt_bytes=self.receipt("check"))
            self.assertEqual(ledger.data["usage"]["offlineChecks"], 1)
        with self.ledger() as resumed:
            self.assertEqual(resumed.reuse("check", input_sha256=self.input), self.receipt("check"))
            self.assertEqual(resumed.data["usage"]["offlineChecks"], 1)
            with self.assertRaises(ValueError):
                resumed.reuse("check", input_sha256="b" * 64)

    def test_inflight_interruption_is_unknown_with_no_replay(self):
        with self.ledger() as ledger:
            ledger.plan("provider", kind="external", cost={"providerCalls": 1}, input_sha256=self.input)
            ledger.reserve("provider", now_utc=NOW)
            ledger.begin_external("provider", now_utc=NOW, intent_sha256="b" * 64)
        with self.ledger() as resumed:
            self.assertEqual(resumed.data["stages"]["provider"]["state"], "unknown")
            self.assertEqual(resumed.data["usage"]["providerCalls"], 1)
            with self.assertRaises(ValueError):
                resumed.begin_external("provider", now_utc=NOW, intent_sha256="b" * 64)
            with self.assertRaises(ValueError):
                resumed.reserve("provider", now_utc=NOW)

    def test_original_return_can_be_reconciled_without_new_effect(self):
        with self.ledger() as ledger:
            ledger.plan("provider", kind="external", cost={"providerCalls": 1}, input_sha256=self.input)
            ledger.reserve("provider", now_utc=NOW)
            ledger.begin_external("provider", now_utc=NOW, intent_sha256="b" * 64)
        with self.ledger() as ledger:
            ledger.reconcile_original("provider", receipt_path=self.root / "original-return.bin",
                                      receipt_bytes=self.receipt("provider", external=True,
                                                                 intent="b" * 64, result=b"returned"))
        with self.ledger() as ledger:
            self.assertEqual(ledger.reuse("provider", input_sha256=self.input),
                             self.receipt("provider", external=True, intent="b" * 64, result=b"returned"))
            self.assertEqual(ledger.data["usage"]["providerCalls"], 1)

    def test_concurrent_writer_fails_closed(self):
        with self.ledger():
            with self.assertRaisesRegex(ValueError, "another resume writer"):
                with self.ledger():
                    pass

    def test_tampered_ledger_or_receipt_rejected(self):
        with self.ledger() as ledger:
            ledger.plan("check", kind="offline", cost={"offlineChecks": 1}, input_sha256=self.input)
            ledger.reserve("check", now_utc=NOW)
            ledger.complete("check", receipt_path=self.root / "receipt.bin", receipt_bytes=self.receipt("check"))
        (self.root / "receipt.bin").write_bytes(b"tampered")
        with self.assertRaisesRegex(ValueError, "receipt missing or changed"):
            with self.ledger():
                pass
        (self.root / "receipt.bin").write_bytes(self.receipt("check"))
        data = json.loads(self.path.read_text())
        data["usage"]["offlineChecks"] = 0
        self.path.write_text(json.dumps(data))
        with self.assertRaisesRegex(ValueError, "self digest"):
            with self.ledger():
                pass

    def test_frozen_source_and_binding_change_rejected(self):
        with self.ledger():
            pass
        self.source.write_bytes(b"changed")
        with self.assertRaisesRegex(ValueError, "frozen source"):
            with self.ledger():
                pass
        self.source.write_bytes(b"frozen")
        self.binding["contract"] = "new-contract"
        with self.assertRaisesRegex(ValueError, "anchor differs"):
            with self.ledger():
                pass

    def test_expiry_preserves_offline_receipt_but_blocks_new_effect(self):
        with self.ledger(NOW) as ledger:
            ledger.plan("check", kind="offline", cost={"offlineChecks": 1}, input_sha256=self.input)
            ledger.reserve("check", now_utc=LATER)
            ledger.complete("check", receipt_path=self.root / "receipt.bin", receipt_bytes=self.receipt("check"))
            ledger.plan("provider", kind="external", cost={"providerCalls": 1}, input_sha256=self.input)
            with self.assertRaisesRegex(ValueError, "expired"):
                ledger.reserve("provider", now_utc=LATER)
            self.assertEqual(ledger.reuse("check", input_sha256=self.input), self.receipt("check"))

    def test_failed_intent_write_never_returns_effect_permission(self):
        with self.ledger() as ledger:
            ledger.plan("provider", kind="external", cost={"providerCalls": 1}, input_sha256=self.input)
            ledger.reserve("provider", now_utc=NOW)
            original = resume._atomic
            try:
                def fail(*_):
                    raise OSError("disk full")
                resume._atomic = fail
                with self.assertRaises(OSError):
                    ledger.begin_external("provider", now_utc=NOW, intent_sha256="b" * 64)
            finally:
                resume._atomic = original
            with self.assertRaises(ValueError):
                ledger.begin_external("provider", now_utc=NOW, intent_sha256="b" * 64)
        with self.ledger() as resumed:
            self.assertEqual(resumed.data["stages"]["provider"]["state"], "reserved")

    def test_wrong_original_receipt_and_intent_rejected(self):
        with self.ledger() as ledger:
            ledger.plan("provider", kind="external", cost={"providerCalls": 1}, input_sha256=self.input)
            ledger.reserve("provider", now_utc=NOW)
            ledger.begin_external("provider", now_utc=NOW, intent_sha256="b" * 64)
            wrong = self.receipt("different", external=True, intent="b" * 64)
            with self.assertRaisesRegex(ValueError, "does not bind"):
                ledger.complete("provider", receipt_path=self.root / "wrong.bin", receipt_bytes=wrong)
        with self.ledger() as ledger:
            wrong_intent = self.receipt("provider", external=True, intent="c" * 64)
            with self.assertRaisesRegex(ValueError, "does not bind"):
                ledger.reconcile_original("provider", receipt_path=self.root / "wrong.bin",
                                          receipt_bytes=wrong_intent)

    def test_missing_or_replaced_original_result_rejected_on_complete_and_reopen(self):
        with self.ledger() as ledger:
            ledger.plan("check", kind="offline", cost={"offlineChecks": 1}, input_sha256=self.input)
            ledger.reserve("check", now_utc=NOW)
            envelope = self.receipt("check")
            (self.root / "check-result.bin").unlink()
            with self.assertRaisesRegex(ValueError, "result file missing"):
                ledger.complete("check", receipt_path=self.root / "receipt.bin", receipt_bytes=envelope)
            envelope = self.receipt("check")
            ledger.complete("check", receipt_path=self.root / "receipt.bin", receipt_bytes=envelope)
        (self.root / "check-result.bin").write_bytes(b"replacement")
        with self.assertRaisesRegex(ValueError, "result file missing or changed"):
            with self.ledger():
                pass

    def test_missing_ledger_or_changed_authority_cannot_reset_usage(self):
        with self.ledger() as ledger:
            ledger.plan("check", kind="offline", cost={"offlineChecks": 1}, input_sha256=self.input)
            ledger.reserve("check", now_utc=NOW)
        self.path.unlink()
        with self.assertRaisesRegex(ValueError, "no automatic reset"):
            with self.ledger():
                pass
        changed = dict(self.binding)
        changed["ledgerPath"] = str(self.root / "new-ledger.json")
        with self.assertRaisesRegex(ValueError, "ledger path"):
            resume.ResumeLedger(self.path, binding=changed, source_files=self.files,
                                limits=self.limits, deadline_utc=LATER)


class MockedPinnedGitTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        workspace = Path(self.tmp.name)
        self.project = workspace / "Customer-Care-Monitor-AI"
        self.core = workspace / ".Controlled-Vibe-Framework-CVF"
        self.project.mkdir()
        self.core.mkdir()
        (workspace / "WORKSPACE_RULES.md").write_text("rules")
        (self.project / ".cvf").mkdir()
        (self.project / "scripts/cvf_gates").mkdir(parents=True)
        self.profile = {"profileId": preflight.PROFILE,
                        "installedFiles": ["cvf_downstream_gate_profile.json",
                                           "cvf_downstream_gate_runner.py"],
                        "projectPaths": {"gatesDir": "scripts/cvf_gates"}, "coreSources": {}}
        self.source_bytes = {
            "cvf_downstream_gate_profile.json": json.dumps(self.profile).encode(),
            "cvf_downstream_gate_runner.py": b"trusted runner"}
        for name, raw in self.source_bytes.items():
            (self.project / "scripts/cvf_gates" / name).write_bytes(raw)
        files = {name: preflight._sha(raw) for name, raw in self.source_bytes.items()}
        listing = "".join(f"{name}:{files[name]}\n" for name in sorted(files))
        lock = {"coreCommit": preflight.PIN, "profileId": preflight.PROFILE,
                "gatesDir": "scripts/cvf_gates", "files": files,
                "bundleSha256": hashlib.sha256(listing.encode()).hexdigest(),
                "profileSha256": files["cvf_downstream_gate_profile.json"],
                "runnerSha256": files["cvf_downstream_gate_runner.py"]}
        (self.project / ".cvf/manifest.json").write_text(json.dumps({
            "cvfCoreCommit": preflight.PIN, "cvfCoreRepository": preflight.PUBLIC,
            "cvfCoreRelativePath": "../.Controlled-Vibe-Framework-CVF",
            "liveGovernanceEvidenceRequired": True, "mockAllowedOnlyForUi": True}))
        (self.project / ".cvf/policy.json").write_text(json.dumps({
            "liveGovernanceEvidenceRequired": True, "mockAllowedOnlyForUi": True,
            "workspaceIsolationRequired": True,
            "cvfCoreRef": "../.Controlled-Vibe-Framework-CVF"}))
        (self.project / ".cvf/gate-profile.lock.json").write_text(json.dumps(lock))
        self.origin = preflight.PUBLIC
        self.pin_available = True
        self.cached = "b" * 40
        self.head = "c" * 40  # clean newer checkout: stable pin remains valid

    def fake_git(self, _core, *args):
        if args == ("remote", "get-url", "origin"):
            return (self.origin + "\n").encode()
        if args == ("rev-parse", "HEAD"):
            return (self.head + "\n").encode()
        if args == ("rev-parse", "refs/remotes/origin/main"):
            return (self.cached + "\n").encode()
        if args == ("status", "--porcelain", "--untracked-files=all"):
            return b""
        if args == ("rev-parse", f"{preflight.PIN}^{{commit}}"):
            if not self.pin_available:
                raise ValueError("pin absent")
            return (preflight.PIN + "\n").encode()
        if args == ("merge-base", "--is-ancestor", preflight.PIN, self.cached):
            return b""
        if args[0] == "show":
            source = args[1].split(":", 1)[1]
            if source in preflight.KIT:
                return b"public-safe kit"
            if source.startswith("scripts/lib/downstream_governance/"):
                return self.source_bytes[source.rsplit("/", 1)[1]]
        raise AssertionError(args)

    def test_clean_newer_checkout_exports_exact_pin_without_fetch(self):
        with patch.object(preflight, "_git", self.fake_git):
            facts = preflight.inspect(self.project, self.core)
            self.assertFalse(facts["checkoutMatchesCachedOriginMain"])
            self.assertFalse(facts["remoteFreshnessChecked"])
            runner = preflight.export(facts)
            self.assertEqual(runner.read_bytes(), self.source_bytes["cvf_downstream_gate_runner.py"])
            self.assertEqual(preflight.export(facts), runner)

    def test_wrong_origin_missing_pin_and_installed_tamper_rejected(self):
        with patch.object(preflight, "_git", self.fake_git):
            self.origin = "https://wrong.invalid/core.git"
            with self.assertRaisesRegex(ValueError, "origin"):
                preflight.inspect(self.project, self.core)
            self.origin = preflight.PUBLIC
            self.pin_available = False
            with self.assertRaisesRegex(ValueError, "pin absent"):
                preflight.inspect(self.project, self.core)
            self.pin_available = True
            (self.project / "scripts/cvf_gates/cvf_downstream_gate_runner.py").write_bytes(b"tampered")
            with self.assertRaisesRegex(ValueError, "identity"):
                preflight.inspect(self.project, self.core)

    def test_unsafe_lock_path_and_reparse_rejected(self):
        lockpath = self.project / ".cvf/gate-profile.lock.json"
        lock = json.loads(lockpath.read_text())
        lock["gatesDir"] = "../outside"
        lockpath.write_text(json.dumps(lock))
        with patch.object(preflight, "_git", self.fake_git):
            with self.assertRaisesRegex(ValueError, "directory mismatch"):
                preflight.inspect(self.project, self.core)
        with patch.object(preflight, "_no_reparse", side_effect=ValueError("reparse")):
            with self.assertRaisesRegex(ValueError, "reparse"):
                preflight.inspect(self.project, self.core)

    def test_both_existing_check_commands_are_invoked_and_fail_closed(self):
        calls = []
        class Result:
            returncode = 0
            stdout = "[PASS] CVF-DG-INST-01\nRESULT: PASS"
            stderr = ""
        def fake_run(cmd, **_):
            calls.append(cmd)
            return Result()
        with patch.object(preflight, "_git", self.fake_git), patch.object(preflight.subprocess, "run", fake_run):
            result = preflight.run(self.project, self.core, "pre-commit", [])
        self.assertTrue(result["passed"])
        self.assertEqual(len(calls), 2)
        self.assertIn("preflight", calls[0])
        self.assertIn("--trusted", calls[1])
        self.assertIn("--expect-bundle-sha256", calls[1])


class PinHelpersTests(unittest.TestCase):
    def test_pin_is_stable_and_lf_hash_matches_lock_convention(self):
        self.assertEqual(preflight.PIN, "e9ddcbcdcac2a9df520f5eb0761dbdb609b9deba")
        self.assertEqual(preflight._sha(b"a\r\nb"), preflight._sha(b"a\nb"))


if __name__ == "__main__":
    unittest.main()
