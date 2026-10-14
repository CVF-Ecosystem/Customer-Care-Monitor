"""Pure admission and continuation checks; no provider, Docker or DB effect."""

import importlib.util
import json
import tempfile
import unittest
import uuid
from pathlib import Path
from unittest.mock import patch

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

    def test_current_seed_budget_and_phase(self):
        r100.canonical_authority("worker")

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
