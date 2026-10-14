"""Offline admission/sanitation checks. No Docker or provider effect."""

import importlib.util
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch


SOURCE = Path(__file__).resolve().parents[1] / "r096_intervention_live_fixture.py"
SPEC = importlib.util.spec_from_file_location("r096_fixture", SOURCE)
assert SPEC and SPEC.loader
r096 = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(r096)


class R096FixtureOfflineTests(unittest.TestCase):
    # cvf-allow-secret-fixture: synthetic string; no real credential is present.
    def test_exact_and_nested_key_redacted(self):
        key = "sk-R096-synthetic-secret-value"  # cvf-allow-secret-fixture
        value = {"wire": {"response": json.dumps({"nested": key})}, "array": [key, "Bearer " + key]}
        raw = r096.safe_serialized(value, key)
        self.assertNotIn(key.encode(), raw)
        self.assertIn(b"[REDACTED]", raw)

    def test_strict_json_duplicate_rejected(self):
        with self.assertRaises(ValueError):
            r096.strict_json(b'{"a":1,"a":2}')

    def test_unicode_escaped_and_duplicate_nested_key_fail_closed(self):
        key = "sk-R096-synthetic-secret-value"  # cvf-allow-secret-fixture
        escaped = "".join("\\u%04x" % ord(char) for char in key)
        self.assertEqual(r096.sanitize({"response": escaped}, key)["response"], "[REDACTED]")
        nested = '{"safe":"yes","safe":"' + escaped + '"}'
        self.assertEqual(r096.sanitize({"response": nested}, key)["response"], "[REDACTED]")
        self.assertEqual(r096.sanitize({key: "value"}, key), {"[REDACTED]": "value"})

    def test_sanitize_depth_fails_closed(self):
        value = "safe"
        for _ in range(20):
            value = [value]
        with self.assertRaises(r096.AdmissionError):
            r096.safe_serialized({"deep": value}, "")

    def test_metadata_pin_rejects_other_endpoint(self):
        with self.assertRaises(r096.AdmissionError):
            r096.endpoint_metadata("https://example.invalid/compatible-mode/v1")

    def test_current_provider_shape_uses_owner_model_not_presets(self):
        endpoint = "https://tenant.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1"
        key = "sk-R096-synthetic-secret-value"  # cvf-allow-secret-fixture
        with patch.object(r096, "ENDPOINT_SHA", r096.digest(endpoint.encode())):
            parsed_key, parsed_endpoint, preset_mentions = r096.parse_provider_config({
                "providers": {"alibaba_maas": {"api_key": key, "base_url": endpoint,
                                               "recommended_model": "other-preset", "models": ["other-preset"],
                                               "curl_command": "IGNORED_UNTRUSTED_TEXT"}}})
        self.assertEqual(parsed_key, key)
        self.assertEqual(parsed_endpoint, endpoint)
        self.assertFalse(preset_mentions)

    def test_preset_object_id_is_observation_only(self):
        endpoint = "https://tenant.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1"
        key = "sk-R096-synthetic-secret-value"  # cvf-allow-secret-fixture
        with patch.object(r096, "ENDPOINT_SHA", r096.digest(endpoint.encode())):
            _, _, mentioned = r096.parse_provider_config({"providers": {"alibaba_maas": {
                "api_key": key, "base_url": endpoint,
                "models": [{"id": r096.MODEL}], "recommended_model": "other"}}})
        self.assertTrue(mentioned)

    def test_old_config_shape_rejected(self):
        with self.assertRaises(r096.AdmissionError):
            r096.parse_provider_config({"auth": {}, "endpoints": {}, "recommendedDefaults": {}})

    def test_role_claim_is_exclusive_and_not_refunded(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with patch.object(r096, "RUNTIME", root), patch.object(r096, "CAMPAIGN", root / "campaign.json"), patch.object(r096, "PARENT", root / "owner_cap.json"):
                r096.claim("worker")
                self.assertTrue((root / "worker.claim").exists())
                with self.assertRaises(r096.AdmissionError):
                    r096.claim("worker")
                self.assertTrue((root / "worker.claim").exists())

    def test_binary_never_runs_without_approval(self):
        with patch.object(r096, "APPROVAL", Path("/missing-r096-approval")):
            with self.assertRaises(r096.AdmissionError):
                r096.admission()

    def test_actual_current_state_passes_source_only_authority(self):
        # This check reads no approval, credential, fixture or provider path.
        state = r096.read_object(r096.ROOT / "CVF_SESSION/ACTIVE_SESSION_STATE.json", "STATE", 256 << 10)
        role = "reviewer" if state.get("currentMode") == "REVIEW" else "worker"
        if state.get("currentMode") in ("BUILD", "REVIEW"):
            r096.canonical_authority(role)
        else:
            with self.assertRaises(r096.AdmissionError):
                r096.canonical_authority(role)

    def test_inactive_phase_rejected(self):
        original = r096.read_object
        def inactive(path, code, max_bytes=65536):
            value = original(path, code, max_bytes)
            if str(path).endswith("ACTIVE_SESSION_STATE.json"):
                value = dict(value, currentMode="FREEZE", activePhase="FREEZE")
            return value
        with patch.object(r096, "read_object", side_effect=inactive):
            with self.assertRaises(r096.AdmissionError):
                r096.canonical_authority("worker")

    def test_actual_parent_manifest_proves_known_zero(self):
        self.assertEqual(len(r096.parent_manifest()), 64)

    def test_reviewer_raw_usage_breach_blocks_second_claim(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with patch.object(r096, "RUNTIME", root), patch.object(r096, "CAMPAIGN", root / "campaign.json"), patch.object(r096, "PARENT", root / "owner_cap.json"):
                r096.claim("worker")
                (root / "slot-1.effect").write_text('{"role":"worker"}', encoding="utf-8")
                (root / "worker.dispatch").write_text("{}", encoding="utf-8")
                (root / "worker_receipt.json").write_text(json.dumps({
                    "actual": {"attempt": None, "wire": {"response": json.dumps({
                        "usage": {"completion_tokens": 769}})}}}), encoding="utf-8")
                with self.assertRaises(r096.AdmissionError) as caught:
                    r096.claim("reviewer")
                self.assertEqual(caught.exception.code, "OBSERVED_OUTPUT_CEILING_BREACH")
                self.assertFalse((root / "reviewer.claim").exists())


if __name__ == "__main__":
    unittest.main()
