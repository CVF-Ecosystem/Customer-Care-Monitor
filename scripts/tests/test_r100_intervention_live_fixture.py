"""Pure admission and continuation checks; no provider, Docker or DB effect."""

import importlib.util
import json
import tempfile
import unittest
import uuid
from pathlib import Path
from unittest.mock import patch

try:
    from ._live_fixture_test_support import (
        fake_git, isolated_authority, isolated_parent_manifest, write_json,
    )
except ImportError:
    from _live_fixture_test_support import (
        fake_git, isolated_authority, isolated_parent_manifest, write_json,
    )

SOURCE = Path(__file__).resolve().parents[1] / "r100_intervention_live_fixture.py"
SPEC = importlib.util.spec_from_file_location("r100_fixture", SOURCE)
assert SPEC and SPEC.loader
r100 = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(r100)


class R100AdmissionTests(unittest.TestCase):
    def test_duplicate_json_and_nonfinite_number_rejected(self):
        for raw in (b'{"a":1,"a":2}', b'{"a":NaN}', b'[]'):
            with self.assertRaises((ValueError, json.JSONDecodeError)):
                r100.strict_json(raw)

    def test_secret_sanitized_in_nested_wire(self):
        key = "sk-R100-synthetic-secret-value"  # cvf-allow-secret-fixture
        raw = r100.safe_serialized({"wire": {"response": json.dumps({"echo": key})}}, key)
        self.assertNotIn(key.encode(), raw)
        self.assertIn(b"[REDACTED]", raw)

    def test_wrong_endpoint_rejected(self):
        with self.assertRaises(r100.AdmissionError):
            r100.endpoint_metadata("https://example.invalid/compatible-mode/v1")

    def test_synthetic_seed_budget_and_phase_guards(self):
        with tempfile.TemporaryDirectory() as directory:
            with isolated_authority(r100, Path(directory)) as fixture:
                r100.canonical_authority("worker")

                fixture["seed"]["executionBudget"]["freshGrantCap"] = 99
                write_json(fixture["seed_path"], fixture["seed"])
                with patch.object(r100, "git", side_effect=fake_git(
                        seed_bytes=fixture["seed_path"].read_bytes())):
                    with self.assertRaises(r100.AdmissionError) as caught:
                        r100.canonical_authority("worker")
                    self.assertEqual(caught.exception.code, "SEED_BUDGET_MISMATCH")

    def test_synthetic_inactive_phase_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            with isolated_authority(r100, Path(directory)) as fixture:
                fixture["state"].update(currentMode="FREEZE", activePhase="FREEZE")
                write_json(fixture["state_path"], fixture["state"])
                with self.assertRaises(r100.AdmissionError) as caught:
                    r100.canonical_authority("worker")
                self.assertEqual(caught.exception.code, "ACTIVE_CONTINUITY_MISMATCH")

    def test_historical_protection_uses_only_synthetic_files(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            protected_path = root / "history/protected.txt"
            marker_path = root / "history/old-marker.json"
            protected_path.parent.mkdir(parents=True)
            protected_path.write_bytes(b"synthetic protected content")
            marker_path.write_bytes(b"synthetic consumed marker")
            baseline = {
                "protectedFiles": {"history/protected.txt": r100.digest(protected_path.read_bytes())},
                "oldConsumedMarkers": {"history/old-marker.json": r100.digest(marker_path.read_bytes())},
            }
            frozen_identity = root / "docs/reviews/probes/r100_root/protected_baseline.json"
            write_json(frozen_identity, baseline)
            with patch.object(r100, "ROOT", root), patch.object(r100, "FROZEN_IDENTITY", frozen_identity):
                r100.historical_protection()
                protected_path.write_bytes(b"changed synthetic content")
                with self.assertRaises(r100.AdmissionError) as caught:
                    r100.historical_protection()
                self.assertEqual(caught.exception.code, "PROTECTED_BASELINE_CHANGED")

    def test_parent_manifest_and_history_are_isolated_and_guarded(self):
        with tempfile.TemporaryDirectory() as directory:
            with isolated_parent_manifest(r100, Path(directory)) as fixture:
                self.assertEqual(r100.parent_manifest(), r100.digest(fixture["ledger_path"].read_bytes()))

                history = [{"attempts": 0, "model": r100.MODEL},
                           {"attempts": 1, "model": r100.MODEL}]
                history_bytes = json.dumps(history, sort_keys=True, separators=(",", ":")).encode()
                fixture["history_path"].write_bytes(history_bytes)
                fixture["ledger"]["historicalReceiptSha256"] = r100.digest(history_bytes)
                write_json(fixture["ledger_path"], fixture["ledger"])
                with self.assertRaises(r100.AdmissionError) as caught:
                    r100.parent_manifest()
                self.assertEqual(caught.exception.code, "HISTORICAL_EFFECT_UNKNOWN")

    def test_claim_is_single_use_before_physical_effect(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with patch.object(r100, "RUNTIME", root), patch.object(r100, "CAMPAIGN", root / "campaign.json"), patch.object(r100, "PARENT", root / "owner_cap.json"):
                r100.claim("worker")
                self.assertTrue((root / "worker.claim").is_file())
                self.assertTrue((root / "campaign.json").is_file())
                with self.assertRaises(r100.AdmissionError):
                    r100.claim("worker")

    def test_continuation_cannot_accept_invalid_slot(self):
        for slot in (0, 4, -1):
            with self.assertRaises(r100.AdmissionError):
                r100.authorize_continuation(slot)

    def test_fixture_parser_accepts_four_distinct_uuid_cases_only(self):
        fields = {name: str(uuid.uuid4()) for name in (
            "tenant", "user", "job", "channel", "wait_conversation", "deny_job", "deny_conversation")}
        fields["conversations"] = [str(uuid.uuid4()) for _ in range(4)]
        self.assertEqual(r100.validate_fixture_fields(fields), fields)
        with self.assertRaises(r100.AdmissionError):
            r100.validate_fixture_fields(dict(fields, conversations=fields["conversations"][:3]))
        with self.assertRaises(r100.AdmissionError):
            r100.validate_fixture_fields(dict(fields, conversations=[fields["conversations"][0]] * 4))
        with self.assertRaises(r100.AdmissionError):
            r100.validate_fixture_fields(dict(fields, job="not-a-uuid"))


if __name__ == "__main__":
    unittest.main()
