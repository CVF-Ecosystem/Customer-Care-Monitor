"""PI03 profile-compatibility gate tests.

Focused positive / negative test suite for the workOrderStatusGrammar@1.0.0 opt-in
added to cvf_downstream_gate.py.  Does NOT modify or depend on any of the 46 legacy
tests in test_cvf_downstream_gate.py.

Coverage requirements (from CCMAI_GOV_002.md PI03 and the round1 admission R1-F1):
- All canonical status -> workOrderStatus mappings pass (9 rows)
- Legacy records without opt-in remain unaffected by the new code path
- Partial opt-in (only one of the two required fields) fails
- Unknown / unsupported grammar string fails
- Missing, wrong-version, unpinned or tampered lock / pinned classifier rejects the opt-in
- workOrderStatus value doesn't match canonical mapping fails
- Work-order document has no matching Status line fails
- Real fenced blocks (backtick and tilde) never supply or hide a Status declaration
- Duplicate, contradictory, malformed and backtick-wrapped declarations fail
- Supported strict syntax (plain, bold, annotated, CRLF) passes
- Discrimination: the three captured false accepts of the returned adapter (preserved under
  docs/reviews/probes/gov002_review_round1) are accepted by that adapter and rejected here
"""
import hashlib
import json
import os
import shutil
import sys
import tempfile
import types
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.abspath(os.path.join(HERE, "..", ".."))
sys.path.insert(0, os.path.join(HERE, ".."))
import cvf_downstream_gate as gate

PROFILE_LOCK_ID = "cvf.downstreamGateProfile@1.1.0"
GRAMMAR_ID = "cvf.workOrderStatusGrammar@1.0.0"
CHECKER = "check_gate_to_role_closeability.py"
WO_PATH = "docs/work_orders/CCMAI_TEST_WO.md"
RETURNED_ADAPTER = os.path.join(
    REPO, "docs", "reviews", "probes", "gov002_review_round1", "original-returned-cvf_downstream_gate.py.txt")

CANONICAL = {
    "WORK_ORDER":       "DRAFT",
    "DISPATCH_READY":   "DISPATCH_READY",
    "BUILD":            "ACTIVE",
    "BUILD_BLOCKED":    "BLOCKED_WITH_REASON",
    "REVIEW_PENDING":   "COMPLETE_PENDING_REVIEW",
    "CHANGES_REQUIRED": "BLOCKED_WITH_REASON",
    "REVIEW_PASS":      "REVIEWER_ACCEPTED",
    "PARKED":           "PARKED",
    "FROZEN":           "CLOSED",
}


def _make_lock(tmpdir, profile_id=PROFILE_LOCK_ID, with_checker=True):
    """Install a lock plus the pinned classifier copied from the real project (hash-consistent)."""
    cvf_dir = os.path.join(tmpdir, ".cvf")
    os.makedirs(cvf_dir, exist_ok=True)
    with open(os.path.join(REPO, ".cvf", "gate-profile.lock.json"), encoding="utf-8") as fh:
        lock = json.load(fh)
    lock["profileId"] = profile_id
    with open(os.path.join(cvf_dir, "gate-profile.lock.json"), "w", encoding="utf-8") as fh:
        json.dump(lock, fh)
    if with_checker:
        gates = os.path.join(tmpdir, *lock["gatesDir"].split("/"))
        os.makedirs(gates, exist_ok=True)
        shutil.copyfile(os.path.join(REPO, "scripts", "cvf_gates", CHECKER), os.path.join(gates, CHECKER))
    return lock


def _make_wo(tmpdir, status_line, wo_name="test_work_order.md"):
    wo_dir = os.path.join(tmpdir, "docs", "work_orders")
    os.makedirs(wo_dir, exist_ok=True)
    rel = "docs/work_orders/" + wo_name
    with open(os.path.join(tmpdir, rel), "w") as f:
        f.write("# Test Work Order\n" + status_line + "\n\nBody text.\n")
    return rel


def _run_opt_in(tmpdir, tranche_status, wo_status, wo_text=None, module=gate):
    r = module.Result("test")
    rec = {
        "trancheId": "CCMAI-TEST-001",
        "status": tranche_status,
        "workOrder": WO_PATH,
        "workOrderStatusGrammar": GRAMMAR_ID,
        "workOrderStatus": wo_status,
    }
    if wo_text is None:
        wo_text = "# Test\nStatus: " + wo_status + "\n\nBody."
    module.check_work_order_status_opt_in(tmpdir, "CCMAI-TEST-001", rec, wo_text, r)
    return r


def _load_returned_adapter():
    """Load the preserved returned adapter (round0 worker version) as an isolated module object."""
    with open(RETURNED_ADAPTER, "rb") as fh:
        source = fh.read()
    module = types.ModuleType("gov002_returned_adapter")
    module.__file__ = RETURNED_ADAPTER
    exec(compile(source, RETURNED_ADAPTER, "exec"), module.__dict__)
    return module


class CanonicalMappings(unittest.TestCase):
    def test_all_canonical_mappings(self):
        for ts, ws in CANONICAL.items():
            with self.subTest(tranche_status=ts, wo_status=ws):
                with tempfile.TemporaryDirectory() as tmp:
                    _make_lock(tmp)
                    wo_text = "# WO\nStatus: " + ws + "\n\nBody.\n"
                    r = _run_opt_in(tmp, ts, ws, wo_text)
                    self.assertFalse(r.problems,
                        "Expected no problems for " + ts + "->" + ws + ", got: " + str(r.problems))

    def test_mapping_table_is_exactly_the_nine_canonical_rows(self):
        self.assertEqual(gate._GRAMMAR_STATUS_MAP, CANONICAL)


class LegacyRecordsUnaffected(unittest.TestCase):
    def test_legacy_record_no_fields_passes_through(self):
        r = gate.Result("test")
        rec = {"trancheId": "CCMAI-TEST-001", "status": "WORK_ORDER"}
        gate.check_work_order_status_opt_in(".", "CCMAI-TEST-001", rec, "irrelevant", r)
        self.assertFalse(r.problems)

    def test_check_order_status_legacy_uses_old_path(self):
        with tempfile.TemporaryDirectory() as tmp:
            rel = _make_wo(tmp, "Status: BUILD")
            r = gate.Result("test")
            rec = {"trancheId": "CCMAI-TEST-001", "status": "BUILD", "workOrder": rel}
            gate.check_order_status(tmp, rec, r)
            self.assertFalse(r.problems)

    def test_legacy_record_still_compares_original_status_and_ignores_new_grammar(self):
        with tempfile.TemporaryDirectory() as tmp:
            rel = _make_wo(tmp, "Status: ACTIVE")  # new-grammar token is NOT a legacy status
            r = gate.Result("test")
            rec = {"trancheId": "CCMAI-TEST-001", "status": "BUILD", "workOrder": rel}
            gate.check_order_status(tmp, rec, r)
            self.assertTrue(r.problems, "a legacy record must keep comparing its original status")

    def test_legacy_record_needs_no_lock_or_classifier(self):
        with tempfile.TemporaryDirectory() as tmp:
            rel = _make_wo(tmp, "Status: REVIEW_PENDING")
            r = gate.Result("test")
            rec = {"trancheId": "CCMAI-TEST-001", "status": "REVIEW_PENDING", "workOrder": rel}
            gate.check_order_status(tmp, rec, r)
            self.assertFalse(r.problems)


class PartialOptIn(unittest.TestCase):
    def test_only_grammar_present(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            r = gate.Result("test")
            rec = {"trancheId": "CCMAI-TEST-001", "status": "BUILD",
                   "workOrderStatusGrammar": GRAMMAR_ID}
            gate.check_work_order_status_opt_in(tmp, "CCMAI-TEST-001", rec, "irrelevant", r)
            self.assertTrue(r.problems, "Partial opt-in must fail")

    def test_only_status_present(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            r = gate.Result("test")
            rec = {"trancheId": "CCMAI-TEST-001", "status": "BUILD",
                   "workOrderStatus": "ACTIVE"}
            gate.check_work_order_status_opt_in(tmp, "CCMAI-TEST-001", rec, "irrelevant", r)
            self.assertTrue(r.problems, "Partial opt-in must fail")


class UnknownGrammar(unittest.TestCase):
    def test_unknown_grammar_string(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            r = gate.Result("test")
            rec = {"trancheId": "CCMAI-TEST-001", "status": "BUILD",
                   "workOrderStatusGrammar": "cvf.workOrderStatusGrammar@9.9.9",
                   "workOrderStatus": "ACTIVE"}
            gate.check_work_order_status_opt_in(tmp, "CCMAI-TEST-001", rec, "irrelevant", r)
            self.assertTrue(any("unknown or unsupported" in p for p in r.problems))

    def test_empty_grammar_string(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            r = gate.Result("test")
            rec = {"trancheId": "CCMAI-TEST-001", "status": "BUILD",
                   "workOrderStatusGrammar": "",
                   "workOrderStatus": "ACTIVE"}
            gate.check_work_order_status_opt_in(tmp, "CCMAI-TEST-001", rec, "irrelevant", r)
            self.assertTrue(r.problems)

    def test_non_string_grammar_and_status_values_fail_closed(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            for grammar, status, tranche in ((["x"], "ACTIVE", "BUILD"), (GRAMMAR_ID, ["ACTIVE"], "BUILD"),
                                             (GRAMMAR_ID, "ACTIVE", ["BUILD"]), (GRAMMAR_ID, None, "BUILD")):
                with self.subTest(grammar=grammar, status=status, tranche=tranche):
                    r = gate.Result("test")
                    rec = {"trancheId": "CCMAI-TEST-001", "status": tranche, "workOrder": WO_PATH,
                           "workOrderStatusGrammar": grammar, "workOrderStatus": status}
                    gate.check_work_order_status_opt_in(tmp, "CCMAI-TEST-001", rec, "# WO\nStatus: ACTIVE\n", r)
                    self.assertTrue(r.problems)


class MissingOrWrongLock(unittest.TestCase):
    def test_no_lock_file_rejects_opt_in(self):
        with tempfile.TemporaryDirectory() as tmp:
            r = _run_opt_in(tmp, "BUILD", "ACTIVE")
            self.assertTrue(any("lock" in p.lower() for p in r.problems))

    def test_wrong_lock_version_rejects_opt_in(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp, profile_id="cvf.downstreamGateProfile@1.0.0")
            r = _run_opt_in(tmp, "BUILD", "ACTIVE")
            self.assertTrue(r.problems)

    def test_malformed_lock_rejects_opt_in(self):
        with tempfile.TemporaryDirectory() as tmp:
            cvf_dir = os.path.join(tmp, ".cvf")
            os.makedirs(cvf_dir)
            with open(os.path.join(cvf_dir, "gate-profile.lock.json"), "w") as f:
                f.write("NOT JSON {{{")
            r = _run_opt_in(tmp, "BUILD", "ACTIVE")
            self.assertTrue(r.problems)


class PinnedClassifierIntegrity(unittest.TestCase):
    """The opt-in must not rely on unpinned or drifted classifier code."""

    def _gates_checker(self, tmp, lock):
        return os.path.join(tmp, *lock["gatesDir"].split("/"), CHECKER)

    def test_tampered_classifier_copy_is_refused_before_import(self):
        with tempfile.TemporaryDirectory() as tmp:
            lock = _make_lock(tmp)
            marker = os.path.join(tmp, "executed.flag")
            with open(self._gates_checker(tmp, lock), "a", encoding="utf-8") as fh:
                fh.write("\nopen(%r, 'w').write('x')\n" % marker)
            r = _run_opt_in(tmp, "BUILD", "ACTIVE")
            self.assertTrue(any("lock pin" in p for p in r.problems), r.problems)
            self.assertFalse(os.path.exists(marker), "tampered code must not execute")

    def test_missing_classifier_file_is_refused(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp, with_checker=False)
            r = _run_opt_in(tmp, "BUILD", "ACTIVE")
            self.assertTrue(any("classifier" in p for p in r.problems), r.problems)

    def test_lock_without_classifier_pin_is_refused(self):
        with tempfile.TemporaryDirectory() as tmp:
            lock = _make_lock(tmp)
            del lock["files"][CHECKER]
            with open(os.path.join(tmp, ".cvf", "gate-profile.lock.json"), "w", encoding="utf-8") as fh:
                json.dump(lock, fh)
            r = _run_opt_in(tmp, "BUILD", "ACTIVE")
            self.assertTrue(any("does not pin" in p for p in r.problems), r.problems)

    def test_lock_pin_changed_to_wrong_hash_is_refused(self):
        with tempfile.TemporaryDirectory() as tmp:
            lock = _make_lock(tmp)
            lock["files"][CHECKER] = "0" * 64
            with open(os.path.join(tmp, ".cvf", "gate-profile.lock.json"), "w", encoding="utf-8") as fh:
                json.dump(lock, fh)
            r = _run_opt_in(tmp, "BUILD", "ACTIVE")
            self.assertTrue(any("lock pin" in p for p in r.problems), r.problems)

    def test_gates_dir_escaping_the_project_is_refused(self):
        with tempfile.TemporaryDirectory() as tmp:
            lock = _make_lock(tmp)
            lock["gatesDir"] = "../outside"
            with open(os.path.join(tmp, ".cvf", "gate-profile.lock.json"), "w", encoding="utf-8") as fh:
                json.dump(lock, fh)
            r = _run_opt_in(tmp, "BUILD", "ACTIVE")
            self.assertTrue(any("outside the project" in p for p in r.problems), r.problems)

    def test_crlf_checker_checkout_matches_the_lf_normalized_pin(self):
        with tempfile.TemporaryDirectory() as tmp:
            lock = _make_lock(tmp)
            path = self._gates_checker(tmp, lock)
            with open(path, "rb") as fh:
                data = fh.read().replace(b"\r\n", b"\n")
            with open(path, "wb") as fh:
                fh.write(data.replace(b"\n", b"\r\n"))
            r = _run_opt_in(tmp, "BUILD", "ACTIVE")
            self.assertFalse(r.problems, r.problems)

    def test_real_project_checker_matches_real_lock(self):
        with open(os.path.join(REPO, ".cvf", "gate-profile.lock.json"), encoding="utf-8") as fh:
            lock = json.load(fh)
        with open(os.path.join(REPO, "scripts", "cvf_gates", CHECKER), "rb") as fh:
            digest = hashlib.sha256(fh.read().replace(b"\r\n", b"\n")).hexdigest()
        self.assertEqual(digest, lock["files"][CHECKER])


class MismatchedStatus(unittest.TestCase):
    def test_wrong_wo_status_for_tranche_status(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            r = _run_opt_in(tmp, "BUILD", "DRAFT",
                            wo_text="# WO\nStatus: DRAFT\n\nBody.")
            self.assertTrue(any("canonical mapping" in p for p in r.problems))

    def test_review_pending_must_map_to_complete_pending_review(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            r = _run_opt_in(tmp, "REVIEW_PENDING", "REVIEWER_ACCEPTED",
                            wo_text="# WO\nStatus: REVIEWER_ACCEPTED\n\nBody.")
            self.assertTrue(r.problems)

    def test_unknown_tranche_status_has_no_mapping(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            r = _run_opt_in(tmp, "NOT_A_STATUS", "ACTIVE")
            self.assertTrue(any("no workOrderStatus mapping" in p for p in r.problems))


class WorkOrderDocumentChecks(unittest.TestCase):
    def test_no_status_line_in_document_fails(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            r = _run_opt_in(tmp, "BUILD", "ACTIVE",
                            wo_text="# WO\ndocType: work_order\n\nNo status line here.")
            self.assertTrue(any("status_missing" in p for p in r.problems), r.problems)

    def test_duplicate_status_lines_fail(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            r = _run_opt_in(tmp, "BUILD", "ACTIVE",
                            wo_text="# WO\nStatus: ACTIVE\nStatus: ACTIVE\n\nBody.")
            self.assertTrue(any("status_duplicate" in p for p in r.problems), r.problems)

    def test_document_status_mismatch_fails(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            r = _run_opt_in(tmp, "BUILD", "ACTIVE",
                            wo_text="# WO\nStatus: DRAFT\n\nBody.")
            self.assertTrue(any("workOrderStatus" in p for p in r.problems))

    def test_unknown_status_token_in_document_fails(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            r = _run_opt_in(tmp, "BUILD", "ACTIVE", wo_text="# WO\nStatus: ACTIVELY_WORKING\n")
            self.assertTrue(any("status_unknown" in p for p in r.problems), r.problems)

    def test_inline_code_mention_is_not_a_declaration(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            wo_text = "# WO\ndocType: work_order\n\nSome text `Status: ACTIVE` in prose.\n\nBody."
            r = _run_opt_in(tmp, "BUILD", "ACTIVE", wo_text=wo_text)
            self.assertTrue(r.problems)

    def test_valid_opt_in_via_check_order_status_dispatch(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            rel = _make_wo(tmp, "Status: ACTIVE")
            r = gate.Result("test")
            rec = {
                "trancheId": "CCMAI-TEST-001",
                "status": "BUILD",
                "workOrder": rel,
                "workOrderStatusGrammar": GRAMMAR_ID,
                "workOrderStatus": "ACTIVE",
            }
            gate.check_order_status(tmp, rec, r)
            self.assertFalse(r.problems)

    def test_invalid_opt_in_via_check_order_status_fails(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            rel = _make_wo(tmp, "Status: DRAFT")
            r = gate.Result("test")
            rec = {
                "trancheId": "CCMAI-TEST-001",
                "status": "BUILD",
                "workOrder": rel,
                "workOrderStatusGrammar": GRAMMAR_ID,
                "workOrderStatus": "ACTIVE",
            }
            gate.check_order_status(tmp, rec, r)
            self.assertTrue(r.problems)


# Documents whose Status declaration the pinned grammar rejects (or never sees) although a
# line-oriented regex over the raw text would find "Status: ACTIVE" or ignore the bad line.
FENCE_AND_SYNTAX_COUNTEREXAMPLES = {
    "fenced_only_backtick": "# WO\ndocType: work_order\n\n```\nStatus: ACTIVE\n```\n",
    "fenced_only_tilde": "# WO\ndocType: work_order\n\n~~~\nStatus: ACTIVE\n~~~\n",
    "fenced_only_info_string": "# WO\ndocType: work_order\n\n```text\nStatus: ACTIVE\n```\n",
    "malformed_second_declaration": "# WO\nStatus: ACTIVE\nStatus: ACTIVE soon, pending\n",
    "contradictory_second_declaration": "# WO\nStatus: ACTIVE\nStatus: DRAFT\n",
    "contradictory_bold_second_declaration": "# WO\nStatus: ACTIVE\n**Status:** DRAFT\n",
    "lowercase_duplicate_declaration": "# WO\nStatus: ACTIVE\nstatus: ACTIVE\n",
    "backtick_wrapped_value": "# WO\nStatus: `ACTIVE`\n",
    "blank_value": "# WO\ndocType: work_order\nStatus:\n",
    "trailing_prose": "# WO\nStatus: ACTIVE and ready\n",
}


class FencesAndStrictSyntax(unittest.TestCase):
    def test_counterexamples_fail_for_every_active_mapping(self):
        for name, text in FENCE_AND_SYNTAX_COUNTEREXAMPLES.items():
            with self.subTest(case=name):
                with tempfile.TemporaryDirectory() as tmp:
                    _make_lock(tmp)
                    r = _run_opt_in(tmp, "BUILD", "ACTIVE", wo_text=text)
                    self.assertTrue(r.problems, f"{name} must fail but passed")

    def test_fenced_block_with_other_status_does_not_count_when_real_line_present(self):
        for fence in ("```", "~~~"):
            with self.subTest(fence=fence):
                with tempfile.TemporaryDirectory() as tmp:
                    _make_lock(tmp)
                    text = "# WO\nStatus: ACTIVE\n\n" + fence + "\nStatus: DRAFT\n" + fence + "\n"
                    r = _run_opt_in(tmp, "BUILD", "ACTIVE", wo_text=text)
                    self.assertFalse(r.problems, r.problems)

    def test_unclosed_fence_is_not_treated_as_a_fence(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            text = "# WO\n```\nStatus: ACTIVE\n"  # no closing fence: the declaration stays visible to the grammar
            r = _run_opt_in(tmp, "BUILD", "ACTIVE", wo_text=text)
            self.assertFalse(r.problems, r.problems)

    def test_supported_strict_syntax_passes(self):
        cases = {
            "plain": "# WO\nStatus: ACTIVE\n",
            "bold_inside": "# WO\n**Status:** ACTIVE\n",
            "bold_label": "# WO\n**Status**: ACTIVE\n",
            "annotated": "# WO\nStatus: ACTIVE (round1 repair)\n",
            "crlf": "# WO\r\nStatus: ACTIVE\r\n\r\nBody.\r\n",
            "leading_blank_and_prose": "\n\nSome prose first.\n\nStatus: ACTIVE\n\nMore prose.\n",
        }
        for name, text in cases.items():
            with self.subTest(case=name):
                with tempfile.TemporaryDirectory() as tmp:
                    _make_lock(tmp)
                    r = _run_opt_in(tmp, "BUILD", "ACTIVE", wo_text=text)
                    self.assertFalse(r.problems, r.problems)

    def test_end_to_end_check_order_status_rejects_fenced_only_files(self):
        for fence in ("```", "~~~"):
            with self.subTest(fence=fence):
                with tempfile.TemporaryDirectory() as tmp:
                    _make_lock(tmp)
                    rel = _make_wo(tmp, fence + "\nStatus: ACTIVE\n" + fence)
                    r = gate.Result("test")
                    rec = {"trancheId": "CCMAI-TEST-001", "status": "BUILD", "workOrder": rel,
                           "workOrderStatusGrammar": GRAMMAR_ID, "workOrderStatus": "ACTIVE"}
                    gate.check_order_status(tmp, rec, r)
                    self.assertTrue(r.problems)

    def test_end_to_end_check_order_status_rejects_malformed_duplicate(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            rel = _make_wo(tmp, "Status: ACTIVE\nStatus: ACTIVE soon, pending")
            r = gate.Result("test")
            rec = {"trancheId": "CCMAI-TEST-001", "status": "BUILD", "workOrder": rel,
                   "workOrderStatusGrammar": GRAMMAR_ID, "workOrderStatus": "ACTIVE"}
            gate.check_order_status(tmp, rec, r)
            self.assertTrue(r.problems)


@unittest.skipUnless(os.path.isfile(RETURNED_ADAPTER), "preserved returned adapter evidence is absent")
class ReturnedAdapterDiscrimination(unittest.TestCase):
    """The new regressions must tell the returned (round0) adapter apart from the repaired one.

    The three captured false accepts are asserted against the preserved returned adapter loaded as
    an isolated module (never the live gate), then against the repaired gate.
    """

    CAPTURED = {
        "fenced_only_status": "# WO\ndocType: work_order\n\n```\nStatus: ACTIVE\n```\n",
        "malformed_second_status": "# WO\nStatus: ACTIVE\nStatus: ACTIVE soon, pending\n",
        "backtick_wrapped_value": "# WO\nStatus: `ACTIVE`\n",
    }

    def test_returned_adapter_falsely_accepts_the_captured_cases(self):
        returned = _load_returned_adapter()
        for name, text in self.CAPTURED.items():
            with self.subTest(case=name):
                with tempfile.TemporaryDirectory() as tmp:
                    _make_lock(tmp)
                    r = _run_opt_in(tmp, "BUILD", "ACTIVE", wo_text=text, module=returned)
                    self.assertFalse(r.problems, f"returned adapter was expected to falsely accept {name}")

    def test_repaired_gate_rejects_the_captured_cases(self):
        for name, text in self.CAPTURED.items():
            with self.subTest(case=name):
                with tempfile.TemporaryDirectory() as tmp:
                    _make_lock(tmp)
                    r = _run_opt_in(tmp, "BUILD", "ACTIVE", wo_text=text)
                    self.assertTrue(r.problems, f"repaired gate must reject {name}")

    def test_returned_and_repaired_agree_on_healthy_and_control_cases(self):
        returned = _load_returned_adapter()
        for ts, ws in CANONICAL.items():
            with self.subTest(tranche_status=ts):
                with tempfile.TemporaryDirectory() as tmp:
                    _make_lock(tmp)
                    text = "# WO\nStatus: " + ws + "\n"
                    old = _run_opt_in(tmp, ts, ws, wo_text=text, module=returned)
                    new = _run_opt_in(tmp, ts, ws, wo_text=text)
                    self.assertFalse(old.problems)
                    self.assertFalse(new.problems)
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            for text in ("# WO\nStatus: DRAFT\n", "# WO\nno status\n", "# WO\nStatus: ACTIVE\nStatus: ACTIVE\n"):
                self.assertTrue(_run_opt_in(tmp, "BUILD", "ACTIVE", wo_text=text, module=returned).problems)
                self.assertTrue(_run_opt_in(tmp, "BUILD", "ACTIVE", wo_text=text).problems)


class UnpinnedOptIn(unittest.TestCase):
    def test_no_lock_directory_at_all(self):
        with tempfile.TemporaryDirectory() as tmp:
            r = _run_opt_in(tmp, "DISPATCH_READY", "DISPATCH_READY")
            self.assertTrue(r.problems)

    def test_correct_lock_clears_the_check(self):
        with tempfile.TemporaryDirectory() as tmp:
            _make_lock(tmp)
            r = _run_opt_in(tmp, "DISPATCH_READY", "DISPATCH_READY",
                            wo_text="# WO\nStatus: DISPATCH_READY\n\nBody.")
            self.assertFalse(r.problems)

    def test_profile_lock_version_helper_reports_none_for_garbage(self):
        with tempfile.TemporaryDirectory() as tmp:
            self.assertIsNone(gate._profile_lock_version(tmp))
            os.makedirs(os.path.join(tmp, ".cvf"))
            with open(os.path.join(tmp, ".cvf", "gate-profile.lock.json"), "w") as f:
                f.write("[]")
            self.assertIsNone(gate._profile_lock_version(tmp))


if __name__ == "__main__":
    unittest.main()
