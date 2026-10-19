"""Offline admission/sanitation checks. No Docker or provider effect."""

import importlib.util
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

try:
    from ._live_fixture_test_support import isolated_authority, write_json
except ImportError:
    from _live_fixture_test_support import isolated_authority, write_json


SOURCE = Path(__file__).resolve().parents[1] / "r095_intervention_live_fixture.py"
SPEC = importlib.util.spec_from_file_location("r095_fixture", SOURCE)
assert SPEC and SPEC.loader
r095 = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(r095)


class R095FixtureOfflineTests(unittest.TestCase):
    # cvf-allow-secret-fixture: synthetic string; no real credential is present.
    def test_exact_and_nested_key_redacted(self):
        key = "sk-R095-synthetic-secret-value"  # cvf-allow-secret-fixture
        value = {"wire": {"response": json.dumps({"nested": key})}, "array": [key, "Bearer " + key]}
        raw = r095.safe_serialized(value, key)
        self.assertNotIn(key.encode(), raw)
        self.assertIn(b"[REDACTED]", raw)

    def test_strict_json_duplicate_rejected(self):
        with self.assertRaises(ValueError):
            r095.strict_json(b'{"a":1,"a":2}')

    def test_unicode_escaped_and_duplicate_nested_key_fail_closed(self):
        key = "sk-R095-synthetic-secret-value"  # cvf-allow-secret-fixture
        escaped = "".join("\\u%04x" % ord(char) for char in key)
        self.assertEqual(r095.sanitize({"response": escaped}, key)["response"], "[REDACTED]")
        nested = '{"safe":"yes","safe":"' + escaped + '"}'
        self.assertEqual(r095.sanitize({"response": nested}, key)["response"], "[REDACTED]")
        self.assertEqual(r095.sanitize({key: "value"}, key), {"[REDACTED]": "value"})

    def test_sanitize_depth_fails_closed(self):
        value = "safe"
        for _ in range(20):
            value = [value]
        with self.assertRaises(r095.AdmissionError):
            r095.safe_serialized({"deep": value}, "")

    def test_metadata_pin_rejects_other_endpoint(self):
        with self.assertRaises(r095.AdmissionError):
            r095.endpoint_metadata("https://example.invalid/compatible-mode/v1")

    def test_role_claim_is_exclusive_and_not_refunded(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with patch.object(r095, "RUNTIME", root), patch.object(r095, "CAMPAIGN", root / "campaign.json"):
                r095.claim("worker")
                self.assertTrue((root / "worker.claim").exists())
                with self.assertRaises(r095.AdmissionError):
                    r095.claim("worker")
                self.assertTrue((root / "worker.claim").exists())

    def test_binary_never_runs_without_approval(self):
        with patch.object(r095, "APPROVAL", Path("/missing-r095-approval")):
            with self.assertRaises(r095.AdmissionError):
                r095.admission()

    def test_synthetic_active_worker_passes_source_only_authority(self):
        with tempfile.TemporaryDirectory() as directory:
            with isolated_authority(r095, Path(directory)):
                r095.canonical_authority("worker")

    def test_inactive_synthetic_phase_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            with isolated_authority(r095, Path(directory)) as fixture:
                fixture["state"]["activePhase"] = "FREEZE"
                write_json(fixture["state_path"], fixture["state"])
                with self.assertRaises(r095.AdmissionError) as caught:
                    r095.canonical_authority("worker")
                self.assertEqual(caught.exception.code, "ACTIVE_CONTINUITY_MISMATCH")

    def test_reviewer_raw_usage_breach_blocks_second_claim(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with patch.object(r095, "RUNTIME", root), patch.object(r095, "CAMPAIGN", root / "campaign.json"):
                r095.claim("worker")
                (root / "worker.dispatch").write_text("{}", encoding="utf-8")
                (root / "worker_receipt.json").write_text(json.dumps({
                    "actual": {"attempt": None, "wire": {"response": json.dumps({
                        "usage": {"completion_tokens": 769}})}}}), encoding="utf-8")
                with self.assertRaises(r095.AdmissionError) as caught:
                    r095.claim("reviewer")
                self.assertEqual(caught.exception.code, "OBSERVED_OUTPUT_CEILING_BREACH")
                self.assertFalse((root / "reviewer.claim").exists())


if __name__ == "__main__":
    unittest.main()
