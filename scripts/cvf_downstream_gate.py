#!/usr/bin/env python3
"""CCMAI-GOV-001: portable downstream CVF machine gates (fail closed).

A small downstream adapter derived from the CVF core control families (provenance, active
continuity, phase/role/work-order contract, return/review/claim boundary, workflow coverage,
secret hygiene, governed-artifact catalog). It does NOT run or copy the core-only checkers:
those depend on core state files, marker grammar and gate IDs this project does not have.
Spec: docs/specs/CVF_DOWNSTREAM_MACHINE_GATES_2026-09-30.md
Applicability: docs/decisions/CVF_DOWNSTREAM_GATE_APPLICABILITY_2026-09-30.md

Passing proves only the checked repository state. It does not prove that CVF controls
runtime AI behavior and it never verifies an external GitHub Actions run.

Usage:
  python scripts/cvf_downstream_gate.py preflight [--root DIR] [--base REF [--head REF]]
                                                  [--files F ...] [--only GATE,...] [--skip-catalog]
Gates: provenance, continuity, tranche, claims, secrets, workflows, catalog
Exit code 0 only when every selected gate passes; diagnostics never print credential values.
"""
import argparse
import fnmatch
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import types

PHASES = ["INTAKE", "DESIGN", "SPEC", "WORK_ORDER", "BUILD", "REVIEW", "FREEZE"]
STATUS_PHASE = {
    "DISPATCH_READY": "WORK_ORDER", "WORK_ORDER": "WORK_ORDER",
    "BUILD": "BUILD", "BUILD_BLOCKED": "BUILD",
    "REVIEW_PENDING": "REVIEW", "CHANGES_REQUIRED": "REVIEW", "REVIEW_PASS": "REVIEW", "PARKED": "REVIEW",
    "FROZEN": "FREEZE",
}
TRANSITIONS = {
    "DISPATCH_READY": {"BUILD", "REVIEW_PENDING", "PARKED"},
    "WORK_ORDER": {"DISPATCH_READY", "BUILD", "REVIEW_PENDING", "PARKED"},
    "BUILD": {"BUILD_BLOCKED", "REVIEW_PENDING"},
    "BUILD_BLOCKED": {"BUILD", "WORK_ORDER", "DISPATCH_READY"},
    "REVIEW_PENDING": {"CHANGES_REQUIRED", "REVIEW_PASS"},
    "CHANGES_REQUIRED": {"REVIEW_PENDING", "BUILD"},
    "REVIEW_PASS": {"FROZEN", "PARKED", "CHANGES_REQUIRED"},
    "PARKED": {"REVIEW_PENDING", "BUILD", "FROZEN", "CHANGES_REQUIRED"},
    "FROZEN": set(),
}
RISKS = ["R0", "R1", "R2", "R3"]
ALWAYS_ALLOWED = ["CVF_SESSION/**", "CVF_SESSION_MEMORY.md", "IMPLEMENTATION_STATUS.json",
                  "docs/reviews/**", "docs/INDEX.md", "docs/catalog/**"]
HEX40 = re.compile(r"^[0-9a-f]{40}$")
TRANCHE_ID = re.compile(r"CCMAI-[A-Z]+-\d+(?:-[A-Z0-9]+)?")
SHA = re.compile(r"\b[0-9a-f]{7,40}\b")
NEGATION = re.compile(r"(?i)\b(not|no|never|without|cannot|can't|doesn't|does not|nor|neither|unless|until|"
                      r"requires?|must|pending|unverified|boundary|limit|limits|claims?|prove|proves|proof)\b")
GOV_CLAIM = re.compile(r"(?i)\bCVF\b[^.\n]{0,40}\b(governs|governed|controls|controlled|enforces)\b[^.\n]{0,40}\b(AI|agent|runtime|provider)")
CI_CLAIM = re.compile(r"(?i)(GitHub Actions|Actions run|workflow run)[^.\n]{0,60}\b(verified|passed|succeeded|success(?:ful)?)\b")
RUN_URL = re.compile(r"^https://github\.com/[\w.-]+/[\w.-]+/actions/runs/\d+$")

SECRET_FILE_PATTERNS = [".env", ".env.*", "*.pem", "*.key", "*.p12", "*.pfx", "id_rsa", "id_rsa.*", "id_ed25519", "id_ed25519.*"]
SECRET_FILE_OK = {".env.example", ".env.sample", ".env.template"}
# Built from fragments so this file never contains a literal the gate itself would match.
_K = "PRIVATE" + " KEY"
SECRET_PATTERNS = [
    ("private-key-block", re.compile("-----BEGIN (?:[A-Z]+ )?" + _K + "-----")),
    ("aws-access-key-id", re.compile(r"\bAKIA[0-9A-Z]{16}\b")),
    ("github-token", re.compile(r"\b(?:gh[pousr]_[A-Za-z0-9]{36,}|github_pat_[A-Za-z0-9_]{40,})\b")),
    ("openai-style-key", re.compile(r"\bsk-[A-Za-z0-9_-]{24,}\b")),
    ("alibaba-access-key", re.compile(r"\bLTAI[A-Za-z0-9]{12,}\b")),
    ("quoted-credential-literal", re.compile(r"(?i)\b(?:api[_-]?key|secret|passw(?:or)?d|token)\b[\"']?\s*[:=]\s*[\"'][A-Za-z0-9/+_\-]{24,}[\"']")),
]
ALLOW_MARKER = "cvf-allow-secret-fixture"

# ----------------------------------------------------------------------------- profile compatibility (PI03)
# workOrderStatusGrammar@1.0.0 opt-in: a new record may carry
#   "workOrderStatusGrammar": "cvf.workOrderStatusGrammar@1.0.0"
#   "workOrderStatus": <mapped display value>
# Both must be present together; partial/unknown/unpinned/fenced/duplicate opt-in fails.
# Only active when a known compatible 1.1.0 or 1.2.0 lock is installed. The work-order
# Status declaration is classified by the lock-pinned Core classifier (classify_status), loaded only
# after its bytes match the lock, so fences (backtick and tilde), duplicate/contradictory/malformed
# declarations and backtick-wrapped values are judged by the same grammar as CVF-DG-APPL-01.
_WORK_ORDER_GRAMMAR_ID = "cvf.workOrderStatusGrammar@1.0.0"
_COMPATIBLE_PROFILE_LOCK_IDS = frozenset((
    "cvf.downstreamGateProfile@1.1.0", "cvf.downstreamGateProfile@1.2.0", "cvf.downstreamGateProfile@1.3.0"))
_MAINTENANCE_TRANCHE = "CCMAI-RUNTIME-139"
_MAINTENANCE_BASE = "cdf3554da9e4359488f9f8dd389be9d16d11e302"
_MAINTENANCE_ADMISSION = (
    "CVF_SESSION/authority/CCMAI-RUNTIME-139-COMMIT-MAINTENANCE-ADMISSION.json")
_MAINTENANCE_FIELDS = frozenset((
    "schemaVersion", "trancheId", "baseCommit", "seedSha256", "riskCeiling",
    "purpose", "ownerInstruction", "roles", "files", "bootstrapException"))
_MAINTENANCE_ROLES = frozenset(("worker", "reviewer", "commitSteward"))
_REPIN_BASE = "0812d972d7d719b3151506e321f651d51cc3ef1e"
_REPIN_CORE_FROM = "b8ead4f0d1bf9224338a206b2abcd893ec1801b1"
_REPIN_CORE_TO = "15e39450853184eb65dbde73398c69c3171a6241"
_REPIN_ADMISSION = "CVF_SESSION/authority/CCMAI-RUNTIME-139-CORE-REPIN-MAINTENANCE-ADMISSION.json"
_REPIN_FIELDS = frozenset(("schemaVersion", "trancheId", "purpose", "baseCommit", "predecessor", "seedSha256", "riskCeiling", "roles", "coreFrom", "coreTo", "files", "bootstrapException", "ownerInstruction"))
_REPIN_EXACT = frozenset((".cvf/manifest.json", ".cvf/gate-profile.lock.json", "AGENTS.md", ".github/workflows/cvf-downstream-gates.yml", "scripts/cvf_downstream_gate.py", "scripts/tests/test_cvf_downstream_gate_core_repin.py"))
_REPIN_INSTALLED = frozenset(("cvf_downstream_gate_profile.json", "cvf_downstream_gate_runner.py", "cvf_dg_common.py", "cvf_dg_install.py", "cvf_dg_continuity.py", "cvf_dg_applicability.py", "cvf_dg_routing.py", "cvf_dg_review.py", "cvf_dg_intake.py", "cvf_dg_coverage.py", "check_gate_to_role_closeability.py", "git-pre-commit.template", "project_learning_home.template.md", "project_learning_record.template.md", "cvf_milestone_coverage.py"))
_REPIN2_BASE = "38952df8bcf9e01b18a05a1da1095b1e7eda6fd8"
_REPIN2_CORE_TO = "c4c8767ac8861854f40d466ba338ffd11eb9558e"
_REPIN2_ADMISSION = "CVF_SESSION/authority/CCMAI-RUNTIME-139-CORE-REPIN-2-MAINTENANCE-ADMISSION.json"
_REPIN2_REPLACEMENTS = frozenset((".cvf/manifest.json", ".cvf/gate-profile.lock.json", "AGENTS.md", "scripts/cvf_downstream_gate.py", "scripts/tests/test_cvf_downstream_gate_core_repin.py"))
_LAW_BASE = "56f2171af6e72e5b5c8525334966a22342232a5e"
_LAW_ADMISSION = "CVF_SESSION/authority/CCMAI-RUNTIME-139-LAW-PROFILE-ADMISSION.json"
_LAW_REPLACE = frozenset((".cvf/gate-profile.lock.json", "AGENTS.md", ".github/workflows/cvf-downstream-gates.yml", "docs/milestone-coverage-authority.json", "docs/roadmaps/CCMA_DELIVERY_ACCEPTANCE_PLAN_2026-10-21.md", "scripts/cvf_downstream_gate.py", "scripts/tests/test_cvf_downstream_gate_core_repin.py", "scripts/tests/test_cvf_downstream_gate_law_profile.py", "scripts/tests/test_cvf_downstream_gate_profile_compatibility.py"))
_LAW_INSTALLED = _REPIN_INSTALLED | {"cvf_repair_convergence.py"}
_MAINTENANCE_PATH_EXACT = frozenset((
    ".cvf/gate-profile.lock.json", ".cvf/manifest.json",
    ".github/workflows/cvf-downstream-gates.yml", "AGENTS.md",
    "docs/milestone-coverage-authority.json", "scripts/cvf_downstream_gate.py",
    "scripts/lib/downstream_catalog/CvfDownstreamCatalogLib.ps1", ".gitignore",
    "scripts/tests/test_cvf_downstream_gate_profile_compatibility.py",
    "scripts/tests/test_cvf_downstream_gate_maintenance.py",
    "scripts/tests/test_cvf_downstream_catalog_parity.py",
    "docs/roadmaps/CCMA_DELIVERY_ACCEPTANCE_PLAN_2026-10-21.md",
    "docs/reviews/planning/CCMA_PRODUCT_DELIVERY_ROADMAP_2026-10-13.md"))
_PINNED_CLASSIFIER_FILE = "check_gate_to_role_closeability.py"
_PINNED_CLASSIFIER_CACHE = {}
# Canonical mapping: tranche status -> allowed workOrderStatus value
_GRAMMAR_STATUS_MAP = {
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


def _profile_lock(root):
    """Return the parsed installed profile lock (dict) or None when absent/unreadable."""
    lock_path = os.path.join(root, ".cvf", "gate-profile.lock.json")
    if not os.path.isfile(lock_path):
        return None
    try:
        with open(lock_path, encoding="utf-8-sig") as fh:
            data = json.load(fh)
    except (OSError, ValueError):
        return None
    return data if isinstance(data, dict) else None


def _profile_lock_version(root):
    """Return installed profileId string if lock exists and is readable, else None."""
    lock = _profile_lock(root)
    pid = lock.get("profileId") if lock else None
    return pid if isinstance(pid, str) else None


def _load_pinned_classifier(root, lock):
    """Return (classify_status, None) or (None, reason).

    The checker bytes are hashed (LF-normalized sha256, as in the lock) and must equal the lock entry
    BEFORE they are executed; the very bytes that were hashed are the ones compiled, so there is no
    verify/use gap and no bytecode is written.
    """
    gates_dir = lock.get("gatesDir")
    files = lock.get("files")
    if not isinstance(gates_dir, str) or not isinstance(files, dict) or not isinstance(files.get(_PINNED_CLASSIFIER_FILE), str):
        return None, f"lock does not pin {_PINNED_CLASSIFIER_FILE}"
    if not inside(root, gates_dir):
        return None, "lock gatesDir is outside the project"
    path = os.path.join(root, gates_dir, _PINNED_CLASSIFIER_FILE)
    try:
        with open(path, "rb") as fh:
            data = fh.read()
    except OSError as exc:
        return None, f"pinned classifier unreadable: {exc.__class__.__name__}"
    digest = hashlib.sha256(data.replace(b"\r\n", b"\n")).hexdigest()
    if digest != files[_PINNED_CLASSIFIER_FILE]:
        return None, f"{_PINNED_CLASSIFIER_FILE} does not match the lock pin (drift or tamper)"
    cached = _PINNED_CLASSIFIER_CACHE.get(digest)
    if cached is not None:
        return cached, None
    name = "cvf_gate_pinned_status_classifier"
    module = types.ModuleType(name)
    module.__file__ = path
    sys.modules[name] = module  # dataclass annotation resolution needs the module registered while executing
    try:
        exec(compile(data, path, "exec"), module.__dict__)
    except Exception as exc:  # fail closed: a broken classifier never admits an opt-in
        return None, f"pinned classifier failed to load: {exc.__class__.__name__}"
    finally:
        sys.modules.pop(name, None)
    classify = getattr(module, "classify_status", None)
    if not callable(classify):
        return None, "pinned classifier lacks classify_status"
    _PINNED_CLASSIFIER_CACHE[digest] = classify
    return classify, None


def check_work_order_status_opt_in(root, tid, rec, wo_text, r):
    """Validate workOrderStatusGrammar opt-in for a new-grammar record.

    Both fields must be present together; if only one is present that is an error.
    If neither is present the record is a legacy record and this function does nothing.
    The check requires a compatible lock and the lock-pinned status classifier.
    """
    has_grammar = "workOrderStatusGrammar" in rec
    has_status = "workOrderStatus" in rec
    if not has_grammar and not has_status:
        return  # legacy record - unchanged semantics
    if has_grammar != has_status:
        r.fail(f"{tid}: workOrderStatusGrammar and workOrderStatus must both be present or both absent")
        return
    grammar = rec.get("workOrderStatusGrammar")
    wo_status = rec.get("workOrderStatus")
    if grammar != _WORK_ORDER_GRAMMAR_ID:
        r.fail(f"{tid}: unknown or unsupported workOrderStatusGrammar {grammar!r}; only {_WORK_ORDER_GRAMMAR_ID!r} is permitted")
        return
    # Require installed lock matching the profile id that defines this grammar
    lock = _profile_lock(root)
    lock_id = lock.get("profileId") if lock else None
    if lock_id not in _COMPATIBLE_PROFILE_LOCK_IDS:
        r.fail(f"{tid}: workOrderStatusGrammar opt-in requires a known compatible profile lock; got {lock_id!r}")
        return
    tranche_status = rec.get("status")
    expected_wo = _GRAMMAR_STATUS_MAP.get(tranche_status) if isinstance(tranche_status, str) else None
    if expected_wo is None:
        r.fail(f"{tid}: no workOrderStatus mapping for tranche status {tranche_status!r}")
        return
    if wo_status != expected_wo:
        r.fail(f"{tid}: workOrderStatus {wo_status!r} does not match canonical mapping {expected_wo!r} for status {tranche_status!r}")
        return
    classify, reason = _load_pinned_classifier(root, lock)
    if classify is None:
        r.fail(f"{tid}: cannot verify work-order Status declaration: {reason}")
        return
    # The work-order document must carry exactly one unfenced, well-formed Status declaration equal to the mapped value.
    decision = classify(rec.get("workOrder") or "work_order.md", wo_text.replace("\r\n", "\n"))
    if decision.outcome == "INVALID":
        r.fail(f"{tid}: work order Status declaration rejected by the pinned grammar ({decision.code}): {decision.reason}")
        return
    if decision.code == "not_work_order" or decision.token != wo_status:
        r.fail(f"{tid}: work order Status {decision.token!r} does not match record workOrderStatus {wo_status!r}")


class Result:
    def __init__(self, name):
        self.name, self.problems, self.notes = name, [], []
        self.skipped = None  # reason string when the control was not executed in this process

    def fail(self, msg):
        self.problems.append(msg)

    def note(self, msg):
        self.notes.append(msg)

    @property
    def ok(self):
        return not self.problems


def read_text(path):
    with open(path, encoding="utf-8-sig", errors="replace", newline=None) as fh:
        return fh.read()


def load_json(path, res, label):
    try:
        with open(path, encoding="utf-8-sig") as fh:
            return json.load(fh)
    except (OSError, ValueError) as exc:
        res.fail(f"{label} missing or not valid JSON ({exc.__class__.__name__}): {path}")
        return None


def inside(root, rel):
    root_abs = os.path.realpath(root)
    target = os.path.realpath(os.path.join(root_abs, rel))
    return target == root_abs or target.startswith(root_abs + os.sep)


def glob_to_regex(pattern):
    out, i = "", 0
    while i < len(pattern):
        if pattern.startswith("**/", i):
            out += "(?:.*/)?"
            i += 3
        elif pattern.startswith("**", i):
            out += ".*"
            i += 2
        elif pattern[i] == "*":
            out += "[^/]*"
            i += 1
        elif pattern[i] == "?":
            out += "[^/]"
            i += 1
        else:
            out += re.escape(pattern[i])
            i += 1
    return re.compile("^" + out + "$")


def path_matches(path, patterns):
    return any(glob_to_regex(p).match(path) for p in patterns)


def norm(s):
    return " ".join(str(s).split())


def norm_move(s):
    """Normalize an instruction: drop markdown code/emphasis marks, unify quotes and spacing."""
    t = str(s).replace("`", "").replace("**", "").replace("‘", "'").replace("’", "'").replace("“", '"').replace("”", '"')
    return " ".join(t.split())


# ----------------------------------------------------------------------------- git helpers
class GateError(Exception):
    """A required input (Git range, pin, repository) is unusable; callers fail closed."""


def _git(root, *args, check=True):
    try:
        out = subprocess.run(["git", "-C", root, *args], capture_output=True, timeout=120)
    except (OSError, subprocess.SubprocessError):
        raise GateError(f"git {args[0]} could not run")
    if check and out.returncode != 0:
        raise GateError(f"git {args[0]} failed")
    return out


def git_z(root, *args):
    """NUL-delimited Git path output (safe for spaces/newlines/quotes in names)."""
    raw = _git(root, *args).stdout
    return [p.decode("utf-8", "surrogateescape") for p in raw.split(b"\0") if p]


def have_git(root):
    try:
        return _git(root, "rev-parse", "--git-dir", check=False).returncode == 0
    except GateError:
        return False


def verify_commit(root, ref, label):
    if not ref or ref.startswith("-") or len(ref) > 200:
        raise GateError(f"{label} is empty or malformed")
    out = _git(root, "rev-parse", "--verify", "--quiet", ref + "^{commit}", check=False)
    if out.returncode != 0:
        raise GateError(f"{label} is unknown or unreachable: {ref[:80]}")
    return out.stdout.decode().strip()


def existing(root, names):
    return sorted(n.replace("\\", "/") for n in names if os.path.isfile(os.path.join(root, n)))


def pr_files(root, args):
    """All changed paths (including deletions) of an explicit set or the PR range base...head."""
    if args.files is not None:
        return sorted(set(n.replace("\\", "/") for n in args.files))
    if args.base:
        if not have_git(root):
            raise GateError("--base needs a git checkout")
        verify_commit(root, args.base, "PR base")
        verify_commit(root, args.head or "HEAD", "PR head")
        return sorted(set(git_z(root, "diff", "--name-only", "-z", f"{args.base}...{args.head or 'HEAD'}")))
    return []


def worktree_files(root):
    """Uncommitted tracked changes plus untracked (non-ignored) files, including deletions."""
    if not have_git(root):
        raise GateError("not a git checkout: worktree changes cannot be determined")
    names = set(git_z(root, "diff", "--name-only", "-z", "HEAD"))
    names |= set(git_z(root, "ls-files", "-o", "--exclude-standard", "-z"))
    return sorted(n.replace("\\", "/") for n in names)


def scope_files(root, args, base_commit):
    """Paths the active tranche touched (including deletions): explicit set, else worktree
    changes plus the validated pin range base_commit..HEAD. Unusable input raises GateError."""
    if args.files is not None:
        return sorted(set(n.replace("\\", "/") for n in args.files))
    if not have_git(root):
        raise GateError("not a git checkout and no --files: changed-file scope cannot be determined")
    if not HEX40.match(str(base_commit or "")):
        raise GateError("tranche baseCommit is missing or not a 40-hex commit")
    verify_commit(root, base_commit, "tranche baseCommit")
    if _git(root, "merge-base", "--is-ancestor", base_commit, "HEAD", check=False).returncode != 0:
        raise GateError("tranche baseCommit is not an ancestor of HEAD")
    names = set(git_z(root, "diff", "--name-only", "-z", "HEAD"))
    names |= set(git_z(root, "ls-files", "-o", "--exclude-standard", "-z"))
    names |= set(git_z(root, "diff", "--name-only", "-z", f"{base_commit}..HEAD"))
    return sorted(n.replace("\\", "/") for n in names)


def _no_duplicate_pairs(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("duplicate JSON key")
        result[key] = value
    return result


def _lf_sha(data):
    return hashlib.sha256(data.replace(b"\r\n", b"\n")).hexdigest()


def _safe_maintenance_path(path):
    if not isinstance(path, str) or not path or "\\" in path or "\x00" in path:
        return False
    if path.startswith("/") or ":" in path or any(part in ("", ".", "..") for part in path.split("/")):
        return False
    if any(c in path for c in "*?[]"):
        return False
    if any(part.lower().startswith(".env") or part.lower() in ("local_provider_secrets.json", "__pycache__")
           for part in path.split("/")) or path.endswith((".pyc", ".pyo")):
        return False
    return path in _MAINTENANCE_PATH_EXACT or path.startswith("scripts/cvf_gates/")


def check_core_repin(root, tid, seed, out_of_scope, r):
    """One fixed successor; verify historical authority before current effective scope."""
    def reject(message):
        raise GateError(message)

    def read_safe(rel):
        full = os.path.join(root, *rel.split("/"))
        expected = os.path.join(os.path.realpath(root), *rel.split("/"))
        if not inside(root, rel) or os.path.normcase(os.path.realpath(full)) != os.path.normcase(expected) or not os.path.isfile(full):
            reject("missing or aliased path: " + rel)
        with open(full, "rb") as fh:
            return fh.read()

    def obj(raw):
        return json.loads(raw.decode("utf-8-sig"), object_pairs_hook=_no_duplicate_pairs)

    def ancestor(a, b):
        if _git(root, "merge-base", "--is-ancestor", a, b, check=False).returncode != 0:
            reject("required ancestry missing")

    try:
        raw = read_safe(_REPIN_ADMISSION)
        admission = obj(raw)
        if not isinstance(admission, dict) or set(admission) != _REPIN_FIELDS:
            reject("successor field set invalid")
        if (tid != _MAINTENANCE_TRANCHE or admission["schemaVersion"] != "1.0" or
                admission["trancheId"] != tid or admission["purpose"] != "STATIC_PUBLIC_CORE_REPIN_ONLY" or
                admission["baseCommit"] != _REPIN_BASE or admission["coreFrom"] != _REPIN_CORE_FROM or
                admission["coreTo"] != _REPIN_CORE_TO or admission["riskCeiling"] != seed.get("riskCeiling") or
                not isinstance(admission["ownerInstruction"], str) or not admission["ownerInstruction"].strip()):
            reject("successor identity or fixed pins invalid")
        roles, exception, predecessor = admission["roles"], admission["bootstrapException"], admission["predecessor"]
        if (not isinstance(roles, dict) or set(roles) != _MAINTENANCE_ROLES or
                not all(isinstance(v, str) and v.strip() for v in roles.values()) or
                roles["worker"].strip().casefold() == roles["reviewer"].strip().casefold() or
                roles["commitSteward"].strip().casefold() != "root"):
            reject("successor roles invalid")
        if (not isinstance(exception, dict) or set(exception) != {"reason", "ownerInstruction"} or
                not isinstance(exception["reason"], str) or not exception["reason"].strip() or
                exception["ownerInstruction"] != admission["ownerInstruction"]):
            reject("successor bootstrap exception invalid")
        if not isinstance(predecessor, dict) or set(predecessor) != {"path", "sha256"} or predecessor["path"] != _MAINTENANCE_ADMISSION:
            reject("predecessor binding invalid")
        old_raw = read_safe(_MAINTENANCE_ADMISSION)
        old = obj(old_raw)
        if predecessor["sha256"] != _lf_sha(old_raw) or not isinstance(old, dict) or set(old) != _MAINTENANCE_FIELDS:
            reject("predecessor hash or schema invalid")
        seed_rel = "CVF_SESSION/authority/" + tid + ".json"
        seed_raw = read_safe(seed_rel)
        if admission["seedSha256"] != _lf_sha(seed_raw) or old["seedSha256"] != _lf_sha(seed_raw):
            reject("seed binding invalid")
        if (_lf_sha(_git(root, "show", f"{_MAINTENANCE_BASE}:{seed_rel}").stdout) != _lf_sha(seed_raw) or
                old["baseCommit"] != _MAINTENANCE_BASE or old["trancheId"] != tid or
                old["purpose"] != "STATIC_REPOSITORY_MAINTENANCE_ONLY" or old["riskCeiling"] != seed.get("riskCeiling")):
            reject("historical seed or predecessor identity invalid")
        head = verify_commit(root, "HEAD", "HEAD")
        verify_commit(root, _REPIN_BASE, "repin base")
        ancestor(_MAINTENANCE_BASE, _REPIN_BASE)
        ancestor(_REPIN_BASE, head)
        old_added = _git(root, "log", "--diff-filter=A", "--format=%H", "--reverse", "--", _MAINTENANCE_ADMISSION).stdout.decode().split()
        if not old_added:
            reject("predecessor first commit absent")
        first_old = old_added[0]
        ancestor(_MAINTENANCE_BASE, first_old)
        ancestor(first_old, _REPIN_BASE)
        if _lf_sha(_git(root, "show", f"{first_old}:{_MAINTENANCE_ADMISSION}").stdout) != _lf_sha(old_raw):
            reject("predecessor first blob changed")
        old_files = old.get("files")
        if not isinstance(old_files, dict) or len(old_files) != 17:
            reject("historical payload must contain 17 files")
        for p, digest in old_files.items():
            if not _safe_maintenance_path(p) or not isinstance(digest, str) or not re.fullmatch(r"[0-9a-f]{64}", digest):
                reject("historical payload entry invalid")
            if _lf_sha(_git(root, "show", f"{first_old}:{p}").stdout) != digest:
                reject("historical payload differs at first commit: " + p)
        added = _git(root, "log", "--diff-filter=A", "--format=%H", "--reverse", "--", _REPIN_ADMISSION).stdout.decode().split()
        if added:
            ancestor(_REPIN_BASE, added[0])
            ancestor(added[0], head)
            if _lf_sha(_git(root, "show", f"{added[0]}:{_REPIN_ADMISSION}").stdout) != _lf_sha(raw):
                reject("successor first blob changed")
        elif head != _REPIN_BASE or _git(root, "cat-file", "-e", f"HEAD:{_REPIN_ADMISSION}", check=False).returncode == 0:
            reject("successor bootstrap requires absent history and exact base HEAD")
        files = admission["files"]
        if not isinstance(files, dict) or not files or set(files) != set(out_of_scope):
            reject("successor effective file set differs")
        # Explicit --files cannot narrow the real active range or hide a worktree deletion.
        record = obj(read_safe("CVF_SESSION/tranches/" + tid + ".json"))
        base = record.get("baseCommit")
        if not isinstance(base, str) or not HEX40.fullmatch(base):
            reject("active range base invalid")
        verify_commit(root, base, "active tranche base")
        ancestor(base, head)
        actual = set(git_z(root, "diff", "--name-only", "-z", f"{base}..HEAD")) | set(worktree_files(root))
        actual = {p for p in actual if not path_matches(p, list(seed.get("allowedPaths") or []) + ALWAYS_ALLOWED)}
        if set(files) != actual:
            reject("successor differs from actual active range and worktree")
        lock = obj(read_safe(".cvf/gate-profile.lock.json"))
        manifest = obj(read_safe(".cvf/manifest.json"))
        if (lock.get("profileId") != "cvf.downstreamGateProfile@1.2.0" or lock.get("profileVersion") != "1.2.0" or
                lock.get("gatesDir") != "scripts/cvf_gates" or lock.get("coreCommit") != _REPIN_CORE_TO or
                manifest.get("cvfCoreCommit") != _REPIN_CORE_TO or not isinstance(lock.get("files"), dict) or
                set(lock["files"]) != _REPIN_INSTALLED):
            reject("unsupported profile or manifest/lock target mismatch")
        staged = set(git_z(root, "diff", "--cached", "--name-only", "-z"))
        if not added and staged and _REPIN_ADMISSION not in staged:
            reject("bootstrap successor must be staged with payload")
        for rel, content in ((_MAINTENANCE_ADMISSION, old_raw), (seed_rel, seed_raw)):
            if rel in staged and _lf_sha(_git(root, "show", f":{rel}").stdout) != _lf_sha(content):
                reject("staged historical authority differs: " + rel)
        if _REPIN_ADMISSION in staged and _lf_sha(_git(root, "show", f":{_REPIN_ADMISSION}").stdout) != _lf_sha(raw):
            reject("staged successor differs")
        for p, digest in files.items():
            installed = p.startswith("scripts/cvf_gates/") and p[len("scripts/cvf_gates/"):] in _REPIN_INSTALLED
            if not _safe_maintenance_path(p) and p not in _REPIN_EXACT:
                reject("unsafe or unauthorized payload path: " + str(p))
            if p.startswith("scripts/cvf_gates/") and not installed:
                reject("unsupported installed file: " + p)
            if not isinstance(digest, str) or not re.fullmatch(r"[0-9a-f]{64}", digest):
                reject("invalid payload digest")
            if p in old_files and p not in _REPIN_EXACT and not installed and old_files[p] != digest:
                reject("nonrepinnable historical digest changed: " + p)
            if p not in old_files and p not in _REPIN_EXACT and not installed:
                reject("new payload is outside repin scope: " + p)
            if _lf_sha(read_safe(p)) != digest or (p in staged and _lf_sha(_git(root, "show", f":{p}").stdout) != digest):
                reject("worktree or staged payload hash differs: " + p)
            if not added and _REPIN_ADMISSION in staged and p not in staged:
                previous = _git(root, "show", f"HEAD:{p}", check=False)
                if previous.returncode != 0 or _lf_sha(previous.stdout) != digest:
                    reject("changed bootstrap payload not staged: " + p)
            if added and _lf_sha(_git(root, "show", f"{added[0]}:{p}").stdout) != digest:
                reject("successor first commit payload differs: " + p)
            if installed and lock["files"].get(p[len("scripts/cvf_gates/"):]) != digest:
                reject("installed payload differs from lock: " + p)
        for p, digest in old_files.items():
            if p not in _REPIN_EXACT and not (p.startswith("scripts/cvf_gates/") and p[len("scripts/cvf_gates/"):] in _REPIN_INSTALLED):
                if _lf_sha(read_safe(p)) != digest or (p in files and files[p] != digest):
                    reject("nonrepinnable historical payload missing or changed: " + p)
                if p in staged and _lf_sha(_git(root, "show", f":{p}").stdout) != digest:
                    reject("staged nonrepinnable historical payload changed: " + p)
        for name in _REPIN_INSTALLED:
            if _lf_sha(read_safe("scripts/cvf_gates/" + name)) != lock["files"].get(name):
                reject("installed file differs from lock: " + name)
        return list(files)
    except (GateError, OSError, UnicodeError, ValueError, TypeError, KeyError, AttributeError) as exc:
        r.fail(f"{tid}: core repin admission invalid ({exc})")
        return []


def check_core_repin_2(root, tid, seed, out_of_scope, r, historical_only=False):
    """The fixed documentation-only c4 successor; historical proof comes from Git blobs."""
    def reject(message):
        raise GateError(message)

    def read_safe(rel):
        full = os.path.join(root, *rel.split("/"))
        expected = os.path.join(os.path.realpath(root), *rel.split("/"))
        if not inside(root, rel) or os.path.normcase(os.path.realpath(full)) != os.path.normcase(expected) or not os.path.isfile(full):
            reject("missing or aliased path: " + rel)
        with open(full, "rb") as fh:
            return fh.read()

    def obj(raw):
        return json.loads(raw.decode("utf-8-sig"), object_pairs_hook=_no_duplicate_pairs)

    def ancestor(a, b):
        if _git(root, "merge-base", "--is-ancestor", a, b, check=False).returncode != 0:
            reject("required ancestry missing")

    def blob(commit, rel):
        # A symlink blob must never substitute for a regular historical payload.
        entry = _git(root, "ls-tree", "-z", commit, "--", rel).stdout
        if not entry.startswith((b"100644 blob ", b"100755 blob ")) or entry.count(b"\x00") != 1:
            reject("historical path is not a regular file: " + rel)
        return _git(root, "show", f"{commit}:{rel}").stdout

    def first_add(rel):
        commits = _git(root, "log", "--diff-filter=A", "--format=%H", "--reverse", "--", rel).stdout.decode().split()
        return commits[0] if commits else None

    def index_blob(rel):
        entry = _git(root, "ls-files", "--stage", "-z", "--", rel).stdout
        if (not entry.startswith((b"100644 ", b"100755 ")) or entry.count(b"\x00") != 1 or
                b" 0\t" not in entry):
            reject("staged path is not a regular stage-zero file: " + rel)
        return _git(root, "show", f":{rel}").stdout

    def roles_and_exception(admission):
        roles, exception = admission.get("roles"), admission.get("bootstrapException")
        instruction = admission.get("ownerInstruction")
        if (not isinstance(instruction, str) or not instruction.strip() or
                not isinstance(roles, dict) or set(roles) != _MAINTENANCE_ROLES or
                not all(isinstance(v, str) and v.strip() for v in roles.values()) or
                roles["worker"].strip().casefold() == roles["reviewer"].strip().casefold() or
                roles["commitSteward"].strip().casefold() != "root" or
                not isinstance(exception, dict) or set(exception) != {"reason", "ownerInstruction"} or
                not isinstance(exception["reason"], str) or not exception["reason"].strip() or
                exception["ownerInstruction"] != instruction):
            reject("roles or bootstrap exception invalid")

    def payload(admission, commit, count):
        files = admission.get("files")
        if not isinstance(files, dict) or len(files) != count:
            reject("historical payload count invalid")
        for path, digest in files.items():
            installed = path.startswith("scripts/cvf_gates/") and path[len("scripts/cvf_gates/"):] in _REPIN_INSTALLED
            if (not _safe_maintenance_path(path) and path not in _REPIN_EXACT or
                    path.startswith("scripts/cvf_gates/") and not installed or
                    not isinstance(digest, str) or not re.fullmatch(r"[0-9a-f]{64}", digest)):
                reject("historical payload entry invalid")
            if _lf_sha(blob(commit, path)) != digest:
                reject("historical first-commit payload differs: " + path)
        return files

    try:
        head = verify_commit(root, "HEAD", "HEAD")
        for fixed in (_MAINTENANCE_BASE, _REPIN_BASE, _REPIN2_BASE):
            verify_commit(root, fixed, "fixed repin base")
        staged = set(git_z(root, "diff", "--cached", "--name-only", "-z"))
        seed_rel = "CVF_SESSION/authority/" + tid + ".json"
        seed_raw = read_safe(seed_rel)
        seed_first = first_add(seed_rel)
        if not seed_first or obj(seed_raw) != seed:
            reject("seed first commit absent or supplied seed differs")
        ancestor(seed_first, _MAINTENANCE_BASE)
        if any(_lf_sha(blob(c, seed_rel)) != _lf_sha(seed_raw) for c in (seed_first, _MAINTENANCE_BASE)):
            reject("original seed first blob changed")

        old_raw, prior_raw = read_safe(_MAINTENANCE_ADMISSION), read_safe(_REPIN_ADMISSION)
        old, prior = obj(old_raw), obj(prior_raw)
        old_first, prior_first = first_add(_MAINTENANCE_ADMISSION), first_add(_REPIN_ADMISSION)
        if not old_first or not prior_first:
            reject("historical admission first commit absent")
        for a, b in ((_MAINTENANCE_BASE, old_first), (old_first, _REPIN_BASE),
                     (_REPIN_BASE, prior_first), (prior_first, _REPIN2_BASE), (_REPIN2_BASE, head)):
            ancestor(a, b)
        for rel, raw, first in ((_MAINTENANCE_ADMISSION, old_raw, old_first), (_REPIN_ADMISSION, prior_raw, prior_first)):
            if _lf_sha(blob(first, rel)) != _lf_sha(raw):
                reject("historical admission first blob changed: " + rel)
        if (not isinstance(old, dict) or set(old) != _MAINTENANCE_FIELDS or
                old["schemaVersion"] != "1.0" or old["trancheId"] != tid or
                old["baseCommit"] != _MAINTENANCE_BASE or old["purpose"] != "STATIC_REPOSITORY_MAINTENANCE_ONLY" or
                old["riskCeiling"] != seed.get("riskCeiling") or old["seedSha256"] != _lf_sha(seed_raw)):
            reject("historical maintenance identity invalid")
        roles_and_exception(old)
        old_files = payload(old, old_first, 17)
        if (not isinstance(prior, dict) or set(prior) != _REPIN_FIELDS or
                prior["schemaVersion"] != "1.0" or prior["trancheId"] != tid or
                prior["purpose"] != "STATIC_PUBLIC_CORE_REPIN_ONLY" or prior["baseCommit"] != _REPIN_BASE or
                prior["coreFrom"] != _REPIN_CORE_FROM or prior["coreTo"] != _REPIN_CORE_TO or
                prior["riskCeiling"] != seed.get("riskCeiling") or prior["seedSha256"] != _lf_sha(seed_raw) or
                prior["predecessor"] != {"path": _MAINTENANCE_ADMISSION, "sha256": _lf_sha(old_raw)}):
            reject("historical first repin identity invalid")
        roles_and_exception(prior)
        prior_files = payload(prior, prior_first, 18)
        for path, digest in old_files.items():
            installed = path.startswith("scripts/cvf_gates/") and path[len("scripts/cvf_gates/"):] in _REPIN_INSTALLED
            if path not in _REPIN_EXACT and not installed and prior_files.get(path) != digest:
                reject("historical nonrepinnable payload changed")
        old_lock = obj(blob(prior_first, ".cvf/gate-profile.lock.json"))
        old_manifest = obj(blob(prior_first, ".cvf/manifest.json"))
        if (not isinstance(old_lock, dict) or old_lock.get("coreCommit") != _REPIN_CORE_TO or
                old_lock.get("profileId") != "cvf.downstreamGateProfile@1.2.0" or old_lock.get("profileVersion") != "1.2.0" or
                old_lock.get("gatesDir") != "scripts/cvf_gates" or not isinstance(old_lock.get("files"), dict) or
                set(old_lock["files"]) != _REPIN_INSTALLED or
                not isinstance(old_manifest, dict) or old_manifest.get("cvfCoreCommit") != _REPIN_CORE_TO):
            reject("historical profile or manifest invalid")
        if (any(not isinstance(old_lock.get(key), str) or not re.fullmatch(r"[0-9a-f]{64}", old_lock[key])
                for key in ("profileSha256", "runnerSha256", "bundleSha256")) or
                old_lock["profileSha256"] != old_lock["files"]["cvf_downstream_gate_profile.json"] or
                old_lock["runnerSha256"] != old_lock["files"]["cvf_downstream_gate_runner.py"]):
            reject("historical profile digest pins invalid")
        for name in _REPIN_INSTALLED:
            digest = old_lock["files"][name]
            if (not isinstance(digest, str) or not re.fullmatch(r"[0-9a-f]{64}", digest) or
                    _lf_sha(blob(prior_first, "scripts/cvf_gates/" + name)) != digest or
                    (not historical_only and _lf_sha(read_safe("scripts/cvf_gates/" + name)) != digest)):
                reject("installed source differs from historical lock: " + name)

        raw = read_safe(_REPIN2_ADMISSION)
        admission = obj(raw)
        if (tid != _MAINTENANCE_TRANCHE or not isinstance(admission, dict) or set(admission) != _REPIN_FIELDS or
                admission["schemaVersion"] != "1.0" or admission["trancheId"] != tid or
                admission["purpose"] != "STATIC_PUBLIC_CORE_REPIN_ONLY" or admission["baseCommit"] != _REPIN2_BASE or
                admission["coreFrom"] != _REPIN_CORE_TO or admission["coreTo"] != _REPIN2_CORE_TO or
                admission["riskCeiling"] != seed.get("riskCeiling") or admission["seedSha256"] != _lf_sha(seed_raw) or
                admission["predecessor"] != {"path": _REPIN_ADMISSION, "sha256": _lf_sha(prior_raw)}):
            reject("second repin identity or fixed pins invalid")
        roles_and_exception(admission)
        second_first = first_add(_REPIN2_ADMISSION)
        if second_first:
            ancestor(_REPIN2_BASE, second_first)
            ancestor(second_first, head)
            if _lf_sha(blob(second_first, _REPIN2_ADMISSION)) != _lf_sha(raw):
                reject("second repin first blob changed")
        elif head != _REPIN2_BASE or _git(root, "cat-file", "-e", f"HEAD:{_REPIN2_ADMISSION}", check=False).returncode == 0:
            reject("second bootstrap requires absent history and exact base HEAD")
        files = admission["files"]
        if historical_only:
            if not second_first or not isinstance(files, dict) or set(files) != set(prior_files):
                reject("historical second payload missing or differs")
            historical_lock = obj(blob(second_first, ".cvf/gate-profile.lock.json"))
            historical_manifest = obj(blob(second_first, ".cvf/manifest.json"))
            if historical_lock != dict(old_lock, coreCommit=_REPIN2_CORE_TO) or historical_manifest != dict(old_manifest, cvfCoreCommit=_REPIN2_CORE_TO):
                reject("historical second manifest/lock differs")
            for path, digest in files.items():
                if (not isinstance(digest, str) or not re.fullmatch(r"[0-9a-f]{64}", digest) or
                        path not in _REPIN2_REPLACEMENTS and digest != prior_files[path] or
                        _lf_sha(blob(second_first, path)) != digest):
                    reject("historical second payload differs: " + path)
            return files
        if not isinstance(files, dict) or set(files) != set(prior_files) or set(files) != set(out_of_scope):
            reject("second effective file set differs")
        record = obj(read_safe("CVF_SESSION/tranches/" + tid + ".json"))
        base = record.get("baseCommit")
        if not isinstance(base, str) or not HEX40.fullmatch(base):
            reject("active range base invalid")
        verify_commit(root, base, "active tranche base")
        ancestor(base, head)
        actual = set(git_z(root, "diff", "--name-only", "-z", f"{base}..HEAD")) | set(worktree_files(root))
        actual = {p for p in actual if not path_matches(p, list(seed.get("allowedPaths") or []) + ALWAYS_ALLOWED)}
        if set(files) != actual:
            reject("second differs from actual active range and worktree")
        current_lock = obj(read_safe(".cvf/gate-profile.lock.json"))
        expected_lock = dict(old_lock, coreCommit=_REPIN2_CORE_TO)
        current_manifest = obj(read_safe(".cvf/manifest.json"))
        if current_lock != expected_lock or current_manifest != dict(old_manifest, cvfCoreCommit=_REPIN2_CORE_TO):
            reject("second manifest/lock differs from fixed documentation-only repin")
        if not second_first and staged and _REPIN2_ADMISSION not in staged:
            reject("bootstrap second admission must be staged with payload")
        for rel, content in ((_MAINTENANCE_ADMISSION, old_raw), (_REPIN_ADMISSION, prior_raw),
                             (seed_rel, seed_raw), (_REPIN2_ADMISSION, raw)):
            if rel in staged and _lf_sha(index_blob(rel)) != _lf_sha(content):
                reject("staged authority differs: " + rel)
        for path, digest in files.items():
            if not isinstance(digest, str) or not re.fullmatch(r"[0-9a-f]{64}", digest):
                reject("second payload digest invalid")
            if path not in _REPIN2_REPLACEMENTS and digest != prior_files[path]:
                reject("documentation-only frozen payload changed: " + path)
            if _lf_sha(read_safe(path)) != digest or (path in staged and _lf_sha(index_blob(path)) != digest):
                reject("worktree or staged second payload differs: " + path)
            if second_first and _lf_sha(blob(second_first, path)) != digest:
                reject("second first-commit payload differs: " + path)
            if not second_first and _REPIN2_ADMISSION in staged and path not in staged:
                if _lf_sha(blob(head, path)) != digest:
                    reject("changed bootstrap second payload not staged: " + path)
        return list(files)
    except (GateError, OSError, UnicodeError, ValueError, TypeError, KeyError, AttributeError) as exc:
        r.fail(f"{tid}: second core repin admission invalid ({exc})")
        return []


def check_law_profile_admission(root, tid, seed, out_of_scope, r):
    """One explicit learning migration; validate previous proof at original Git blobs."""
    def reject(message):
        raise GateError(message)

    def safe(rel):
        full = os.path.join(root, *rel.split("/"))
        expected = os.path.join(os.path.realpath(root), *rel.split("/"))
        if not inside(root, rel) or os.path.normcase(os.path.realpath(full)) != os.path.normcase(expected) or not os.path.isfile(full):
            reject("missing or aliased path: " + rel)
        with open(full, "rb") as f:
            return f.read()

    def obj(raw):
        return json.loads(raw.decode("utf-8-sig"), object_pairs_hook=_no_duplicate_pairs)

    def blob(commit, path):
        entry = _git(root, "ls-tree", "-z", commit, "--", path).stdout
        if not entry.startswith((b"100644 blob ", b"100755 blob ")) or entry.count(b"\x00") != 1:
            reject("non-regular historical file: " + path)
        return _git(root, "show", f"{commit}:{path}").stdout

    try:
        previous = check_core_repin_2(root, tid, seed, out_of_scope, r, historical_only=True)
        if not previous:
            return []
        head = verify_commit(root, "HEAD", "HEAD")
        verify_commit(root, _LAW_BASE, "law base")
        if _git(root, "merge-base", "--is-ancestor", _LAW_BASE, head, check=False).returncode:
            reject("law base is not ancestor")
        raw = safe(_LAW_ADMISSION)
        admission = obj(raw)
        seed_raw = safe("CVF_SESSION/authority/" + tid + ".json")
        prior_raw = safe(_REPIN2_ADMISSION)
        if (not isinstance(admission, dict) or set(admission) != _REPIN_FIELDS or tid != _MAINTENANCE_TRANCHE or
            admission["schemaVersion"] != "1.0" or admission["trancheId"] != tid or
            admission["purpose"] != "STATIC_LAW_PROFILE_1_3_MIGRATION_ONLY" or admission["baseCommit"] != _LAW_BASE or
            admission["seedSha256"] != _lf_sha(seed_raw) or admission["riskCeiling"] != seed.get("riskCeiling") or
            admission["coreFrom"] != _REPIN2_CORE_TO or admission["coreTo"] != "UNKNOWN" or
            admission["predecessor"] != {"path": _REPIN2_ADMISSION, "sha256": _lf_sha(prior_raw)}):
            reject("law successor identity invalid")
        instruction, roles, exception = admission["ownerInstruction"], admission["roles"], admission["bootstrapException"]
        if (not isinstance(instruction, str) or "2026-11-05" not in instruction or not instruction.strip() or
            not isinstance(roles, dict) or set(roles) != _MAINTENANCE_ROLES or
            not all(isinstance(v, str) and v.strip() for v in roles.values()) or
            roles["worker"].strip().casefold() == roles["reviewer"].strip().casefold() or
            roles["commitSteward"].strip().casefold() != "root" or
            not isinstance(exception, dict) or set(exception) != {"reason", "ownerInstruction"} or
            not isinstance(exception["reason"], str) or not exception["reason"].strip() or exception["ownerInstruction"] != instruction):
            reject("law successor roles or authorization invalid")
        commits = _git(root, "log", "--diff-filter=A", "--format=%H", "--reverse", "--", _LAW_ADMISSION).stdout.decode().split()
        first = commits[0] if commits else None
        if first:
            if (_git(root, "merge-base", "--is-ancestor", _LAW_BASE, first, check=False).returncode or
                _git(root, "merge-base", "--is-ancestor", first, head, check=False).returncode or
                _lf_sha(blob(first, _LAW_ADMISSION)) != _lf_sha(raw)):
                reject("law admission first blob or ancestry invalid")
        elif head != _LAW_BASE or _git(root, "cat-file", "-e", "HEAD:" + _LAW_ADMISSION, check=False).returncode == 0:
            reject("law bootstrap needs exact base HEAD")
        files = admission["files"]
        allowed = _LAW_REPLACE | {"scripts/cvf_gates/" + n for n in _LAW_INSTALLED}
        if not isinstance(files, dict) or set(files) != set(out_of_scope) or not set(previous).issubset(files) or set(files) - set(previous) - allowed:
            reject("law payload scope differs")
        record = obj(safe("CVF_SESSION/tranches/" + tid + ".json"))
        base = record.get("baseCommit")
        verify_commit(root, base, "active tranche base")
        if _git(root, "merge-base", "--is-ancestor", base, head, check=False).returncode:
            reject("active tranche base not ancestor")
        actual = set(git_z(root, "diff", "--name-only", "-z", f"{base}..HEAD")) | set(worktree_files(root))
        actual = {p for p in actual if not path_matches(p, list(seed.get("allowedPaths") or []) + ALWAYS_ALLOWED)}
        if set(files) != actual:
            reject("law payload differs from actual range")
        lock, manifest = obj(safe(".cvf/gate-profile.lock.json")), obj(safe(".cvf/manifest.json"))
        if (manifest != obj(blob(_LAW_BASE, ".cvf/manifest.json")) or lock.get("coreCommit") != "UNKNOWN" or
            lock.get("profileId") != "cvf.downstreamGateProfile@1.3.0" or lock.get("profileVersion") != "1.3.0" or
            lock.get("gatesDir") != "scripts/cvf_gates" or not isinstance(lock.get("files"), dict) or set(lock["files"]) != _LAW_INSTALLED):
            reject("law profile or public manifest invalid")
        if any(_lf_sha(safe("scripts/cvf_gates/" + n)) != lock["files"][n] for n in _LAW_INSTALLED):
            reject("installed profile file differs from lock")
        gate_dir = os.path.join(root, "scripts", "cvf_gates")
        actual_names = set()
        for directory, dirs, names in os.walk(gate_dir, followlinks=False):
            if dirs:
                reject("unexpected installed profile directory")
            actual_names.update(os.path.relpath(os.path.join(directory, n), gate_dir).replace(os.sep, "/") for n in names)
        bundle = hashlib.sha256("".join(f"{n}:{lock['files'][n]}\n" for n in sorted(_LAW_INSTALLED)).encode("utf-8")).hexdigest()
        pins = (lock.get("profileSha256"), lock.get("runnerSha256"), lock.get("bundleSha256"))
        if (actual_names != _LAW_INSTALLED or pins != (lock["files"]["cvf_downstream_gate_profile.json"], lock["files"]["cvf_downstream_gate_runner.py"], bundle) or
            any(pin not in safe(".github/workflows/cvf-downstream-gates.yml").decode("utf-8") for pin in pins)):
            reject("installed bundle or CI pins differ")
        staged = set(git_z(root, "diff", "--cached", "--name-only", "-z"))
        if not first and staged and _LAW_ADMISSION not in staged:
            reject("bootstrap admission must be staged with payload")
        def index_blob(path):
            entry = _git(root, "ls-files", "--stage", "-z", "--", path).stdout
            if not entry.startswith((b"100644 ", b"100755 ")) or entry.count(b"\x00") != 1 or b" 0\t" not in entry:
                reject("staged path not regular: " + path)
            return _git(root, "show", ":" + path).stdout
        registry_path = "docs/milestone-coverage-authority.json"
        for path, digest in files.items():
            if (not isinstance(digest, str) or not re.fullmatch(r"[0-9a-f]{64}", digest) or
                path in previous and path not in allowed and digest != previous[path]):
                reject("law payload digest or frozen path differs: " + path)
            current_digest = _lf_sha(safe(path))
            if current_digest != digest and not (first and path == registry_path):
                reject("law payload digest or frozen path differs: " + path)
            if path in staged and _lf_sha(index_blob(path)) != current_digest:
                reject("staged law payload differs: " + path)
            if first and _lf_sha(blob(first, path)) != digest:
                reject("law first-commit payload differs: " + path)
            if not first and _LAW_ADMISSION in staged and path not in staged and _lf_sha(blob(head, path)) != digest:
                reject("changed bootstrap payload not staged: " + path)
        if first:
            before, after = obj(blob(first, registry_path)), obj(safe(registry_path))
            if (not isinstance(before, dict) or not isinstance(after, dict) or
                {k: v for k, v in before.items() if k != "repairControls"} !=
                {k: v for k, v in after.items() if k != "repairControls"}):
                reject("registry non-repair definitions changed")
            import importlib.util
            from pathlib import Path
            spec = importlib.util.spec_from_file_location("cvf_law_pinned_coverage", os.path.join(gate_dir, "cvf_milestone_coverage.py"))
            if spec is None or spec.loader is None:
                reject("pinned repair transition unavailable")
            module = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(module)
            failures = module.validate_repair_transition(Path(root), first)
            if failures:
                reject("invalid registry repair transition: " + ", ".join(failures))
        for path, data in ((_LAW_ADMISSION, raw), (_REPIN2_ADMISSION, prior_raw),
                           (_REPIN_ADMISSION, safe(_REPIN_ADMISSION)), (_MAINTENANCE_ADMISSION, safe(_MAINTENANCE_ADMISSION)),
                           ("CVF_SESSION/authority/" + tid + ".json", seed_raw)):
            if path in staged and _lf_sha(index_blob(path)) != _lf_sha(data):
                reject("staged authority differs: " + path)
        return list(files)
    except (GateError, OSError, UnicodeError, ValueError, TypeError, KeyError, AttributeError) as exc:
        r.fail(f"{tid}: law profile admission invalid ({exc})")
        return []


def check_maintenance_admission(root, tid, seed, out_of_scope, r):
    """Admit only exact, hash-bound R139 static maintenance; never alter the seed.

    The first same-commit admission is an explicit bootstrap exception. Once committed,
    its first blob is immutable. In both cases every admitted file is rehashed.
    """
    if tid != _MAINTENANCE_TRANCHE:
        return []
    law_exists = os.path.lexists(os.path.join(root, *_LAW_ADMISSION.split("/")))
    if not law_exists and have_git(root):
        try:
            law_exists = bool(_git(root, "log", "--diff-filter=A", "--format=%H", "--", _LAW_ADMISSION).stdout.strip())
        except GateError:
            r.fail(f"{tid}: law successor history could not be verified")
            return []
    if law_exists:
        return check_law_profile_admission(root, tid, seed, out_of_scope, r)
    second_exists = os.path.lexists(os.path.join(root, *_REPIN2_ADMISSION.split("/")))
    if not second_exists and have_git(root):
        try:
            second_exists = bool(_git(root, "log", "--diff-filter=A", "--format=%H", "--", _REPIN2_ADMISSION).stdout.strip())
        except GateError:
            r.fail(f"{tid}: second successor history could not be verified")
            return []
    if second_exists:
        return check_core_repin_2(root, tid, seed, out_of_scope, r)
    successor_exists = os.path.lexists(os.path.join(root, *_REPIN_ADMISSION.split("/")))
    if not successor_exists and have_git(root):
        try:
            successor_exists = bool(_git(root, "log", "--diff-filter=A", "--format=%H", "--", _REPIN_ADMISSION).stdout.strip())
        except GateError:
            r.fail(f"{tid}: successor history could not be verified")
            return []
    if successor_exists:
        return check_core_repin(root, tid, seed, out_of_scope, r)
    rel = _MAINTENANCE_ADMISSION
    path = os.path.join(root, *rel.split("/"))
    if not os.path.isfile(path):
        return []
    expected_real = os.path.join(os.path.realpath(root), *rel.split("/"))
    if os.path.normcase(os.path.realpath(path)) != os.path.normcase(expected_real):
        r.fail(f"{tid}: maintenance admission path is an alias")
        return []
    if not out_of_scope:
        r.fail(f"{tid}: unused maintenance admission has no changed out-of-seed files")
        return []
    try:
        with open(path, "rb") as fh:
            raw = fh.read()
        admission = json.loads(raw.decode("utf-8-sig"), object_pairs_hook=_no_duplicate_pairs)
    except (OSError, UnicodeError, ValueError) as exc:
        r.fail(f"{tid}: maintenance admission unreadable or invalid ({exc.__class__.__name__})")
        return []
    if not isinstance(admission, dict) or set(admission) != _MAINTENANCE_FIELDS:
        r.fail(f"{tid}: maintenance admission field set is invalid")
        return []
    if (admission["schemaVersion"] != "1.0" or admission["trancheId"] != tid or
            admission["purpose"] != "STATIC_REPOSITORY_MAINTENANCE_ONLY" or
            admission["riskCeiling"] != seed.get("riskCeiling") or
            not isinstance(admission["ownerInstruction"], str) or
            not admission["ownerInstruction"].strip()):
        r.fail(f"{tid}: maintenance admission identity, risk or purpose is invalid")
        return []
    roles = admission["roles"]
    if (not isinstance(roles, dict) or set(roles) != _MAINTENANCE_ROLES or
            not all(isinstance(v, str) and v.strip() for v in roles.values()) or
            roles["worker"].strip().casefold() == roles["reviewer"].strip().casefold() or
            roles["commitSteward"].strip().casefold() != "root"):
        r.fail(f"{tid}: maintenance admission roles are invalid")
        return []
    exception = admission["bootstrapException"]
    if (not isinstance(exception, dict) or set(exception) != {"reason", "ownerInstruction"} or
            not isinstance(exception["reason"], str) or not exception["reason"].strip() or
            exception["ownerInstruction"] != admission["ownerInstruction"]):
        r.fail(f"{tid}: maintenance bootstrap exception is invalid")
        return []
    base = admission["baseCommit"]
    if base != _MAINTENANCE_BASE:
        r.fail(f"{tid}: maintenance baseCommit is invalid")
        return []
    files = admission["files"]
    if not isinstance(files, dict) or not files or set(files) != set(out_of_scope):
        r.fail(f"{tid}: maintenance file set must equal the changed out-of-seed set")
        return []
    if not all(_safe_maintenance_path(p) and isinstance(h, str) and re.fullmatch(r"[0-9a-f]{64}", h)
               for p, h in files.items()):
        r.fail(f"{tid}: maintenance file path or digest is invalid")
        return []
    lock = _profile_lock(root)
    lock_files = lock.get("files") if isinstance(lock, dict) else None
    for p, digest in files.items():
        if p.startswith("scripts/cvf_gates/"):
            name = p[len("scripts/cvf_gates/"):]
            if (not isinstance(lock_files, dict) or lock.get("gatesDir") != "scripts/cvf_gates" or
                    name not in lock_files or lock_files[name] != digest):
                r.fail(f"{tid}: inherited gate file is not pinned by installed lock: {p}")
                return []
    seed_path = os.path.join(root, "CVF_SESSION", "authority", tid + ".json")
    try:
        with open(seed_path, "rb") as fh:
            seed_hash = _lf_sha(fh.read())
        if admission["seedSha256"] != seed_hash:
            r.fail(f"{tid}: maintenance seed hash differs from immutable seed")
            return []
        head = verify_commit(root, "HEAD", "maintenance HEAD")
        verify_commit(root, base, "maintenance baseCommit")
        if _git(root, "merge-base", "--is-ancestor", base, head, check=False).returncode != 0:
            r.fail(f"{tid}: maintenance baseCommit is not an ancestor of HEAD")
            return []
        added = _git(root, "log", "--diff-filter=A", "--format=%H", "--reverse", "--", rel).stdout.decode().split()
        if added:
            first = added[0]
            if _git(root, "merge-base", "--is-ancestor", base, first, check=False).returncode != 0:
                r.fail(f"{tid}: maintenance first commit predates its base")
                return []
            first_blob = _git(root, "show", f"{first}:{rel}").stdout
            if first_blob.replace(b"\r\n", b"\n") != raw.replace(b"\r\n", b"\n"):
                r.fail(f"{tid}: maintenance admission differs from its first committed content")
                return []
        elif head != base or _git(root, "cat-file", "-e", f"HEAD:{rel}", check=False).returncode == 0:
            r.fail(f"{tid}: maintenance bootstrap requires absent history and HEAD equal baseCommit")
            return []
        staged = set(git_z(root, "diff", "--cached", "--name-only", "-z"))
        for p, expected in files.items():
            full = os.path.join(root, *p.split("/"))
            expected_real = os.path.join(os.path.realpath(root), *p.split("/"))
            if (not inside(root, p) or os.path.normcase(os.path.realpath(full)) !=
                    os.path.normcase(expected_real) or not os.path.isfile(full)):
                r.fail(f"{tid}: maintenance file missing or outside project: {p}")
                continue
            with open(full, "rb") as fh:
                if _lf_sha(fh.read()) != expected:
                    r.fail(f"{tid}: maintenance file hash differs: {p}")
            if p in staged and _lf_sha(_git(root, "show", f":{p}").stdout) != expected:
                r.fail(f"{tid}: staged maintenance file hash differs: {p}")
        if rel in staged and _git(root, "show", f":{rel}").stdout.replace(b"\r\n", b"\n") != raw.replace(b"\r\n", b"\n"):
            r.fail(f"{tid}: staged maintenance admission differs")
    except (GateError, OSError) as exc:
        r.fail(f"{tid}: maintenance admission could not be verified ({exc.__class__.__name__})")
        return []
    return list(files)


# ----------------------------------------------------------------------------- gates
def gate_provenance(root, ctx):
    r = Result("provenance")
    manifest = load_json(os.path.join(root, ".cvf", "manifest.json"), r, "manifest")
    policy = load_json(os.path.join(root, ".cvf", "policy.json"), r, "policy")
    ctx["manifest"], ctx["policy"] = manifest, policy
    if manifest:
        if manifest.get("phaseModel") != PHASES:
            r.fail("manifest.phaseModel is not the canonical INTAKE..FREEZE model")
        if not HEX40.match(str(manifest.get("cvfCoreCommit", ""))):
            r.fail("manifest.cvfCoreCommit is not a 40-hex commit")
        for key in ("cvfCoreRepository", "cvfCoreRelativePath", "requiredDocs"):
            if not manifest.get(key):
                r.fail(f"manifest.{key} missing")
        if manifest.get("liveGovernanceEvidenceRequired") is not True or manifest.get("mockAllowedOnlyForUi") is not True:
            r.fail("manifest live-evidence/mock policy flags must both be true")
        for doc in manifest.get("requiredDocs") or []:
            if doc.startswith("..") or os.path.isabs(doc):
                r.note(f"required doc outside project not checked (portable mode): {doc}")
            elif not inside(root, doc):
                r.fail(f"required doc escapes project root: {doc}")
            elif not os.path.exists(os.path.join(root, doc)):
                r.fail(f"required doc missing: {doc}")
    if policy:
        for key in ("liveGovernanceEvidenceRequired", "mockAllowedOnlyForUi", "workspaceIsolationRequired", "phaseTransitionRequired"):
            if policy.get(key) is not True:
                r.fail(f"policy.{key} must be true")
        if policy.get("riskCeiling") not in RISKS:
            r.fail("policy.riskCeiling is not R0..R3")
    if not os.path.isfile(os.path.join(root, "AGENTS.md")):
        r.fail("AGENTS.md missing")
    r.note("sibling CVF core and ../WORKSPACE_RULES.md are local-doctor concerns; this gate assumes no sibling core")
    return r


MARKER_RE = re.compile(r"<!--\s*cvf-front-marker\s*(\{.*?\})\s*-->", re.S)


def header_fields(text):
    fields = {}
    for key in ("Current mode", "Active phase", "Active role", "Next allowed move", "Parked operator checkpoint"):
        m = re.search(r"^- " + re.escape(key) + r":[ \t]*(.*)$", text, re.M)
        if m:
            fields[key] = m.group(1).strip().strip("`")
    return fields


def gate_continuity(root, ctx):
    r = Result("continuity")
    state = load_json(os.path.join(root, "CVF_SESSION", "ACTIVE_SESSION_STATE.json"), r, "active state")
    ctx["state"] = state
    if not state:
        return r
    manifest = ctx.get("manifest") or {}
    phases = manifest.get("phaseModel") or PHASES
    mode, phase = state.get("currentMode"), state.get("activePhase")
    for label, val in (("currentMode", mode), ("activePhase", phase)):
        if val not in phases:
            r.fail(f"state.{label} {val!r} is not in the manifest phase model")
    for key in ("activeHandoff", "nextAllowedMove", "activeRole"):
        if not state.get(key):
            r.fail(f"state.{key} missing")
    if "parkedOperatorCheckpoint" not in state:
        r.fail("state.parkedOperatorCheckpoint must be declared (null when none)")
    handoff_rel = state.get("activeHandoff") or ""
    if not handoff_rel or not inside(root, handoff_rel):
        r.fail("state.activeHandoff is missing or escapes the project root")
        return r
    handoff_path = os.path.join(root, handoff_rel)
    if not os.path.isfile(handoff_path):
        r.fail(f"active handoff not found: {handoff_rel}")
        return r
    hf = header_fields(read_text(handoff_path))
    for key in ("Current mode", "Active phase", "Active role", "Next allowed move", "Parked operator checkpoint"):
        if key not in hf:
            r.fail(f"handoff header missing '{key}'")
    if hf:
        if hf.get("Current mode") != mode:
            r.fail(f"handoff Current mode {hf.get('Current mode')!r} != state.currentMode {mode!r}")
        if (hf.get("Active phase", "").split() or [""])[0] != phase:
            r.fail(f"handoff Active phase {hf.get('Active phase', '')[:20]!r} does not start with state.activePhase {phase!r}")
        if "Active role" in hf and norm(hf["Active role"]) != norm(state.get("activeRole", "")):
            r.fail("handoff Active role differs from state.activeRole")
        s_ids, h_ids = set(TRANCHE_ID.findall(state.get("nextAllowedMove", ""))), set(TRANCHE_ID.findall(hf.get("Next allowed move", "")))
        if s_ids != h_ids:
            r.fail(f"next-move tranche IDs differ: state {sorted(s_ids)} vs handoff {sorted(h_ids)}")
        # The actionable instruction itself is bound too: the handoff header must carry the
        # canonical state text (whitespace and markdown emphasis are normalized away).
        if norm_move(hf.get("Next allowed move", "")) != norm_move(state.get("nextAllowedMove", "")):
            r.fail("handoff Next allowed move differs from state.nextAllowedMove (normalized text)")
        parked = state.get("parkedOperatorCheckpoint")
        h_parked = hf.get("Parked operator checkpoint", "")
        if (parked in (None, "")) != (h_parked.lower() in ("none", "")):
            r.fail("parked checkpoint declared in only one of state and handoff")
        elif parked:
            if set(SHA.findall(parked)) != set(SHA.findall(h_parked)):
                r.fail("parked checkpoint commit references differ between state and handoff")

    mem_path = os.path.join(root, "CVF_SESSION_MEMORY.md")
    marker = None
    if not os.path.isfile(mem_path):
        r.fail("CVF_SESSION_MEMORY.md missing")
    else:
        m = MARKER_RE.search(read_text(mem_path))
        if not m:
            r.fail("CVF_SESSION_MEMORY.md has no machine-readable <!-- cvf-front-marker {...} --> block")
        else:
            try:
                marker = json.loads(m.group(1))
            except ValueError:
                r.fail("memory front marker is not valid JSON")
    ctx["marker"] = marker
    if marker:
        if marker.get("currentMode") != mode:
            r.fail(f"memory marker currentMode {marker.get('currentMode')!r} != state {mode!r}")
        if marker.get("activePhase") != phase:
            r.fail(f"memory marker activePhase {marker.get('activePhase')!r} != state {phase!r}")
        if marker.get("activeHandoff") != handoff_rel:
            r.fail("memory marker activeHandoff differs from state.activeHandoff")
        if bool(marker.get("parked")) != bool(state.get("parkedOperatorCheckpoint")):
            r.fail("memory marker parked flag differs from state.parkedOperatorCheckpoint")
        if not marker.get("activeTranche"):
            r.fail("memory marker activeTranche missing")
        elif marker["activeTranche"] not in TRANCHE_ID.findall(state.get("nextAllowedMove", "")):
            r.fail(f"state.nextAllowedMove does not mention the active tranche {marker['activeTranche']}")

    status = load_json(os.path.join(root, "IMPLEMENTATION_STATUS.json"), r, "implementation status")
    if status is not None and status.get("currentPhase") != phase:
        r.fail(f"IMPLEMENTATION_STATUS.currentPhase {status.get('currentPhase')!r} != state.activePhase {phase!r}")
    bootstrap = os.path.join(root, "CVF_SESSION", "ACTIVE_SESSION_BOOTSTRAP_READ_MODEL.json")
    if os.path.isfile(bootstrap):
        bm = load_json(bootstrap, r, "bootstrap read model")
        if isinstance(bm, dict):
            for key in ("currentMode", "activeHandoff"):
                if key in bm and bm[key] != state.get(key):
                    r.fail(f"bootstrap read model {key} differs from canonical state")
    else:
        r.note("BOOTSTRAP_MIGRATION_PENDING (compact bootstrap read model absent; nonblocking)")
    return r


def validate_record(root, rec, r, active_phase=None):
    tid = rec.get("trancheId", "?")
    status, phase, risk = rec.get("status"), rec.get("phase"), rec.get("riskCeiling")
    if status not in STATUS_PHASE:
        r.fail(f"{tid}: unknown status {status!r}")
        return
    if phase not in PHASES or STATUS_PHASE[status] != phase:
        r.fail(f"{tid}: phase {phase!r} is inconsistent with status {status}")
    if active_phase is not None and phase != active_phase:
        r.fail(f"{tid}: record phase {phase!r} != active session phase {active_phase!r}")
    if risk not in RISKS:
        r.fail(f"{tid}: riskCeiling {risk!r} is not R0..R3")
    roles = rec.get("roles") or {}
    for key in ("workOrderAuthor", "implementationWorker", "commitSteward", "reviewer"):
        if not roles.get(key):
            r.fail(f"{tid}: roles.{key} missing")
    if risk in ("R2", "R3"):
        worker = str(roles.get("implementationWorker", "")).strip().lower()
        repair = str(roles.get("repairWorker", "")).strip().lower()
        reviewer = str(roles.get("reviewer", "")).strip().lower()
        if reviewer and reviewer in (worker, repair):
            r.fail(f"{tid}: {risk} reviewer must be independent of the implementation/repair worker")
    if risk == "R3" and not rec.get("formalApproval"):
        r.fail(f"{tid}: R3 requires formalApproval")
    if not rec.get("allowedPaths") or not isinstance(rec.get("allowedPaths"), list):
        r.fail(f"{tid}: allowedPaths must be a non-empty list")
    if not isinstance(rec.get("prohibitedEffects"), list) or not rec.get("prohibitedEffects"):
        r.fail(f"{tid}: prohibitedEffects must be a non-empty list")
    wo = rec.get("workOrder")
    if not wo or not inside(root, wo) or not os.path.isfile(os.path.join(root, wo)):
        r.fail(f"{tid}: workOrder path missing or outside the project")
    freeze = rec.get("freeze")
    if freeze not in ("OPEN", "FROZEN"):
        r.fail(f"{tid}: freeze must be OPEN or FROZEN")
    if freeze == "FROZEN" and (status != "FROZEN" or rec.get("disposition") != "REVIEW_PASS" or not rec.get("closer")):
        r.fail(f"{tid}: FROZEN needs status FROZEN, disposition REVIEW_PASS and a recorded closer")
    if status == "FROZEN" and freeze != "FROZEN":
        r.fail(f"{tid}: status FROZEN contradicts freeze {freeze!r}")
    if status in ("BUILD", "BUILD_BLOCKED", "REVIEW_PENDING", "CHANGES_REQUIRED", "WORK_ORDER", "DISPATCH_READY") and freeze != "OPEN":
        r.fail(f"{tid}: a {status} record cannot claim FREEZE")
    if status in ("REVIEW_PASS", "FROZEN"):
        if rec.get("disposition") != "REVIEW_PASS":
            r.fail(f"{tid}: {status} requires disposition REVIEW_PASS")
        if not HEX40.match(str(rec.get("buildCommit", ""))):
            r.fail(f"{tid}: {status} requires a 40-hex buildCommit")
        ev = rec.get("reviewEvidence") or []
        if not ev:
            r.fail(f"{tid}: {status} requires reviewEvidence")
        for p in ev:
            if not inside(root, p) or not os.path.isfile(os.path.join(root, p)):
                r.fail(f"{tid}: review evidence missing or outside project: {p}")
    hist = rec.get("history")
    if hist is not None:
        if not isinstance(hist, list) or not hist or hist[-1] != status:
            r.fail(f"{tid}: history must be a list ending with the current status")
        else:
            for a, b in zip(hist, hist[1:]):
                if a not in TRANSITIONS or b not in TRANSITIONS[a]:
                    r.fail(f"{tid}: invalid status transition {a} -> {b}")


def gate_tranche(root, ctx, files, scope_err=None):
    r = Result("tranche")
    marker = ctx.get("marker")
    state = ctx.get("state") or {}
    tdir = os.path.join(root, "CVF_SESSION", "tranches")
    records = {}
    if os.path.isdir(tdir):
        for name in sorted(os.listdir(tdir)):
            if name.endswith(".json"):
                rec = load_json(os.path.join(tdir, name), r, f"tranche record {name}")
                if isinstance(rec, dict):
                    if rec.get("trancheId") != name[:-5]:
                        r.fail(f"{name}: trancheId must equal the file name")
                    records[rec.get("trancheId")] = rec
    ctx["records"] = records
    active_id = (marker or {}).get("activeTranche")
    if not active_id:
        r.fail("no active tranche declared (memory front marker)")
        return r
    active = records.get(active_id)
    if active is None:
        r.fail(f"active tranche {active_id} has no CVF_SESSION/tranches/{active_id}.json record")
        return r
    ctx["active_record"] = active
    validate_record(root, active, r, active_phase=state.get("activePhase"))
    for tid, rec in records.items():
        if tid != active_id:
            validate_record(root, rec, r)

    # changed work orders must be bound to a record
    bound = {rec.get("workOrder") for rec in records.values()}
    for f in (files or []):
        if f.startswith("docs/work_orders/") and f.endswith(".md") and os.path.basename(f) != "README.md" and f not in bound:
            r.fail(f"changed work order has no tranche record binding it: {f}")
    # the active order's current Status line must agree with the canonical record
    check_order_status(root, active, r)
    for f in (files or []):
        for rec in records.values():
            if rec is not active and rec.get("workOrder") == f:
                check_order_status(root, rec, r)
    # authority: role/risk/path/effect fields are verified against the dispatcher-owned seed
    seed = check_authority(root, active, r, ctx)
    if scope_err:
        r.fail(f"changed-file scope unavailable: {scope_err}")
        return r
    if files is None:
        r.fail("changed-file scope unavailable")
        return r
    if seed is None:
        return r  # already failed; scope cannot be judged without the authority
    allowed = list(seed.get("allowedPaths") or []) + ALWAYS_ALLOWED
    out_of_scope = [f for f in files if not path_matches(f, allowed)]
    admitted = set(check_maintenance_admission(root, active_id, seed, out_of_scope, r))
    remaining = [f for f in out_of_scope if f not in admitted]
    for f in remaining[:20]:
        r.fail(f"changed path outside the authorized allowedPaths: {f}")
    if len(remaining) > 20:
        r.fail(f"... and {len(remaining) - 20} more paths outside allowedPaths")
    return r


ORDER_STATUS = re.compile(r"^Status:\s*`?([A-Z][A-Z_]+)", re.M)


def check_order_status(root, rec, r):
    tid, wo = rec.get("trancheId", "?"), rec.get("workOrder")
    if not wo or not inside(root, wo) or not os.path.isfile(os.path.join(root, wo)):
        return  # validate_record already reports a missing work order
    wo_text = read_text(os.path.join(root, wo))
    # opt-in grammar check (PI03): must run before legacy check because opt-in uses a different Status vocabulary
    if "workOrderStatusGrammar" in rec or "workOrderStatus" in rec:
        check_work_order_status_opt_in(root, tid, rec, wo_text, r)
        return  # opt-in records are not subject to the legacy STATUS_PHASE comparison
    head = "\n".join(wo_text.splitlines()[:20])
    m = ORDER_STATUS.search(head)
    if not m:
        r.fail(f"{tid}: work order has no 'Status: <STATE>' line in its first 20 lines")
    elif m.group(1) not in STATUS_PHASE:
        r.fail(f"{tid}: work order Status {m.group(1)!r} is not a known state")
    elif m.group(1) != rec.get("status"):
        r.fail(f"{tid}: work order Status {m.group(1)} is stale or contradicts the tranche record status {rec.get('status')}")


def check_authority(root, rec, r, ctx):
    """Compare the worker-editable record with the dispatcher-owned authority seed and, in a git
    checkout, prove the seed is the content it had when first committed. The seed author cannot
    be proven (one git identity), so a reviewer must still confirm who seeded it."""
    tid = rec.get("trancheId", "?")
    rel = f"CVF_SESSION/authority/{tid}.json"
    path = os.path.join(root, *rel.split("/"))
    if not os.path.isfile(path):
        r.fail(f"{tid}: no dispatcher-owned authority seed {rel}")
        return None
    seed = load_json(path, r, "authority seed")
    if not isinstance(seed, dict):
        return None
    if seed.get("trancheId") != tid or not str(seed.get("authorityKind", "")).startswith("DISPATCHER_OWNED"):
        r.fail(f"{tid}: authority seed must declare this trancheId and a DISPATCHER_OWNED authorityKind")
        return None
    if rec.get("riskCeiling") != seed.get("riskCeiling"):
        r.fail(f"{tid}: record riskCeiling differs from the authority seed")
    for key, val in (seed.get("roles") or {}).items():
        if (rec.get("roles") or {}).get(key) != val:
            r.fail(f"{tid}: record roles.{key} differs from the authority seed")
    widened = sorted(set(rec.get("allowedPaths") or []) - set(seed.get("allowedPaths") or []))
    if widened:
        r.fail(f"{tid}: record allowedPaths widen the authority seed: {', '.join(widened)[:200]}")
    dropped = sorted(set(seed.get("prohibitedEffects") or []) - set(rec.get("prohibitedEffects") or []))
    if dropped:
        r.fail(f"{tid}: record drops prohibitedEffects required by the authority seed: {', '.join(dropped)}")
    if not have_git(root):
        r.note("authority seed history unverified (no git checkout)")
        return seed
    try:
        added = _git(root, "log", "--diff-filter=A", "--format=%H", "--reverse", "--", rel).stdout.decode().split()
        if not added:
            r.fail(f"{tid}: authority seed is not committed; it must be introduced in history before it is relied on")
            return seed
        then = _git(root, "show", f"{added[0]}:{rel}").stdout.replace(b"\r\n", b"\n")
        with open(path, "rb") as fh:
            now = fh.read().replace(b"\r\n", b"\n")
        if then != now:
            r.fail(f"{tid}: authority seed differs from its first committed content (rewritten or deleted and re-added)")
        if not seed.get("bootstrapException"):
            base = rec.get("baseCommit")
            if not HEX40.match(str(base or "")) or _git(root, "cat-file", "-e", f"{base}:{rel}", check=False).returncode != 0:
                r.fail(f"{tid}: authority seed was not recorded at the tranche baseCommit (before BUILD) and declares no bootstrapException")
        else:
            r.note(f"{tid}: authority seed uses a declared bootstrapException")
    except GateError as exc:
        r.fail(f"{tid}: authority seed history could not be verified ({exc})")
    return seed


def gate_claims(root, ctx, files, scope_err=None):
    r = Result("claims")
    if scope_err:
        r.fail(f"changed-file scope unavailable: {scope_err}")
        return r
    active = ctx.get("active_record") or {}
    files = existing(root, files or [])
    claim_files, ci_files = [], []
    for f in files:
        if not f.endswith(".md") or not (f.startswith("docs/") or f in ("AGENTS.md", "CVF_SESSION_MEMORY.md")):
            continue
        for line in read_text(os.path.join(root, f)).splitlines():
            if GOV_CLAIM.search(line) and not NEGATION.search(line):
                claim_files.append(f)
                break
        for line in read_text(os.path.join(root, f)).splitlines():
            if CI_CLAIM.search(line) and not NEGATION.search(line):
                ci_files.append(f)
                break
        if "BUILD" in os.path.basename(f) and f.startswith("docs/reviews/"):
            if re.search(r"\bFROZEN\b|\bFREEZE_PASS\b|\bFREEZE_CLOSED\b", read_text(os.path.join(root, f))):
                r.fail(f"BUILD evidence claims a frozen/closed state: {f}")
    if claim_files:
        rcpt = active.get("governanceReceipt")
        ok = False
        if rcpt and inside(root, rcpt) and os.path.isfile(os.path.join(root, rcpt)):
            data = load_json(os.path.join(root, rcpt), r, "governance receipt")
            if isinstance(data, dict):
                ok = (data.get("providerCall") is True and data.get("mock") is not True and data.get("synthetic") is not True
                      and all(data.get(k) for k in ("provider", "model", "request", "response")))
        if not ok:
            for f in sorted(set(claim_files)):
                r.fail(f"{f} claims CVF governs AI/agent behavior without a real-provider receipt "
                       "(tranche.governanceReceipt with provider, model, request, response, providerCall true, mock/synthetic not true)")
    if ci_files:
        run = active.get("ciRun") or {}
        valid = (RUN_URL.match(str(run.get("url", ""))) and HEX40.match(str(run.get("sha", ""))) and run.get("conclusion") == "success")
        if not valid:
            for f in sorted(set(ci_files)):
                r.fail(f"{f} claims a GitHub Actions result without tranche.ciRun (run URL, 40-hex sha, conclusion success)")
        else:
            r.note("ciRun fields are well-formed only; the run is NOT verified by this gate (reviewer checks GitHub)")
    return r


def scan_secrets(rel, text):
    hits = []
    for i, line in enumerate(text.splitlines(), 1):
        if ALLOW_MARKER in line:
            continue
        for name, rx in SECRET_PATTERNS:
            if rx.search(line):
                hits.append((i, name))
    return hits


def gate_secrets(root, ctx, files, err=None):
    r = Result("secrets")
    if err:
        r.fail(f"changed-file set unavailable: {err}")
        return r
    files = existing(root, files or [])
    for f in files or []:
        base = os.path.basename(f)
        if base not in SECRET_FILE_OK and any(fnmatch.fnmatch(base, p) for p in SECRET_FILE_PATTERNS):
            r.fail(f"forbidden credential-type file in the changed set: {f}")
            continue
        path = os.path.join(root, f)
        try:
            if os.path.getsize(path) > 2_000_000:
                continue
            with open(path, "rb") as fh:
                raw = fh.read()
        except OSError:
            continue
        if b"\0" in raw[:4096]:
            continue
        for line, name in scan_secrets(f, raw.decode("utf-8", errors="replace")):
            r.fail(f"{f}:{line}: credential-like literal ({name}); value not shown")
    r.note("bounded static scan of changed files; false negatives are possible and ignored/untracked local secrets are never read")
    return r


def workflow_triggers(root):
    try:
        import yaml  # noqa: F401
    except ImportError:
        return None, "PyYAML is required for the workflow coverage gate"
    import yaml
    out = {}
    wdir = os.path.join(root, ".github", "workflows")
    if not os.path.isdir(wdir):
        return {}, None
    for name in sorted(os.listdir(wdir)):
        if not name.endswith((".yml", ".yaml")):
            continue
        with open(os.path.join(wdir, name), encoding="utf-8") as fh:
            doc = yaml.safe_load(fh) or {}
        on = doc.get("on", doc.get(True, {}))
        if isinstance(on, str):
            on = {on: {}}
        elif isinstance(on, list):
            on = {k: {} for k in on}
        pr = (on or {}).get("pull_request")
        if pr is None and "pull_request" not in (on or {}):
            continue
        pr = pr or {}
        out[name] = pr.get("paths")  # None == every PR
    return out, None


# One representative path per class of change that must reach a relevant PR workflow.
COVERAGE_CLASSES = [
    ("backend", "backend/engine/sync.go", ("backend.yml",)),
    ("backend-ci-gate", "scripts/ci_db_test_gate.py", ("backend.yml", "governance.yml")),
    ("frontend", "frontend/src/views/Dashboard.vue", ("frontend.yml",)),
    ("docs", "docs/guide/index.md", ("docs.yml", "governance.yml")),
    ("cvf-authority", ".cvf/manifest.json", ("governance.yml",)),
    ("session", "CVF_SESSION/ACTIVE_SESSION_STATE.json", ("governance.yml",)),
    ("session-memory", "CVF_SESSION_MEMORY.md", ("governance.yml",)),
    ("status", "IMPLEMENTATION_STATUS.json", ("governance.yml",)),
    ("agent-rules", "AGENTS.md", ("governance.yml",)),
    ("gate-tool", "scripts/cvf_downstream_gate.py", ("governance.yml",)),
    ("workflows", ".github/workflows/governance.yml", ("governance.yml",)),
]


def gate_workflows(root, ctx):
    r = Result("workflows")
    triggers, err = workflow_triggers(root)
    if err:
        r.fail(err)
        return r
    for label, sample, required in COVERAGE_CLASSES:
        hit = []
        for wf in required:
            if wf not in triggers:
                continue
            paths = triggers[wf]
            if paths is None or path_matches(sample, paths):
                hit.append(wf)
        if not hit:
            r.fail(f"changed path class '{label}' ({sample}) reaches none of the required PR workflows {list(required)}")
    return r


def gate_catalog(root, ctx, skip):
    r = Result("catalog")
    if skip:
        r.skipped = "not executed in this process; the Windows CI job (preflight --only catalog) must run it"
        return r
    script = os.path.join(root, "scripts", "manage_cvf_downstream_catalog.ps1")
    if not os.path.isfile(script):
        r.fail("scripts/manage_cvf_downstream_catalog.ps1 missing")
        return r
    shell = shutil.which("pwsh") or shutil.which("powershell")
    if not shell:
        r.fail("no PowerShell (pwsh/powershell) available to run the catalog -Check")
        return r
    try:
        out = subprocess.run([shell, "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script, "-Check", "-ProjectPath", root],
                             capture_output=True, text=True, timeout=300)
    except (OSError, subprocess.SubprocessError) as exc:
        r.fail(f"catalog check could not run: {exc.__class__.__name__}")
        return r
    if out.returncode != 0:
        tail = [ln for ln in (out.stdout + out.stderr).splitlines() if ln.strip()][-3:]
        r.fail("catalog -Check failed: " + " | ".join(tail)[:300])
    return r


ALL_GATES = ["provenance", "continuity", "tranche", "claims", "secrets", "workflows", "catalog"]


def run(root, args):
    selected = args.only.split(",") if args.only else list(ALL_GATES)
    for g in selected:
        if g not in ALL_GATES:
            print(f"unknown gate: {g}", file=sys.stderr)
            return 2
    need = set(selected)
    if need & {"continuity", "tranche", "claims"}:
        need |= {"provenance", "continuity"}
    if "claims" in need:
        need.add("tranche")
    ctx, res = {}, {}
    if "provenance" in need:
        res["provenance"] = gate_provenance(root, ctx)
    if "continuity" in need:
        res["continuity"] = gate_continuity(root, ctx)
    scope = pr = None
    scope_err = pr_err = None
    if need & {"tranche", "claims", "secrets"}:
        base_commit = None
        tid = (ctx.get("marker") or {}).get("activeTranche")
        if tid and os.path.isfile(os.path.join(root, "CVF_SESSION", "tranches", tid + ".json")):
            try:
                with open(os.path.join(root, "CVF_SESSION", "tranches", tid + ".json"), encoding="utf-8-sig") as fh:
                    base_commit = json.load(fh).get("baseCommit")
            except (OSError, ValueError):
                pass
        try:
            scope = scope_files(root, args, base_commit)
        except GateError as exc:
            scope_err = str(exc)
        try:
            pr = pr_files(root, args)
        except GateError as exc:
            pr_err = str(exc)
    if "tranche" in need:
        res["tranche"] = gate_tranche(root, ctx, scope, scope_err or pr_err)
    if "claims" in need:
        # Claims are prospective: only the active tranche's own changes are checked, so accepted
        # historical records in the PR range are not re-judged against the current tranche record.
        res["claims"] = gate_claims(root, ctx, scope, scope_err or pr_err)
    if "secrets" in need:
        if args.files is not None:
            res["secrets"] = gate_secrets(root, ctx, pr, pr_err)
        elif args.base:
            # PR range plus any uncommitted work; the tranche pin is not needed to scan a range
            try:
                res["secrets"] = gate_secrets(root, ctx, sorted(set(pr or []) | set(worktree_files(root))), pr_err)
            except GateError as exc:
                res["secrets"] = gate_secrets(root, ctx, [], str(exc))
        else:
            res["secrets"] = gate_secrets(root, ctx, scope, scope_err)
    if "workflows" in need:
        res["workflows"] = gate_workflows(root, ctx)
    if "catalog" in need:
        res["catalog"] = gate_catalog(root, ctx, args.skip_catalog)
    results = [res[g] for g in ALL_GATES if g in res and g in selected]
    bad = skipped = 0
    for item in results:
        if item.skipped:
            skipped += 1
            print(f"[SKIP] {item.name}: {item.skipped}")
        elif item.ok:
            print(f"[PASS] {item.name}")
        else:
            bad += 1
            print(f"[FAIL] {item.name}")
            for p in item.problems:
                print(f"       - {p}")
        for n in item.notes:
            print(f"       note: {n}")
    executed = len(results) - skipped
    extra = f"; {skipped} skipped" if skipped else ""
    print(f"CVF downstream preflight: {'PASS' if not bad else 'FAIL'} ({executed - bad}/{executed} executed gates passed{extra})")
    return 0 if not bad else 1


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)
    p = sub.add_parser("preflight")
    p.add_argument("--root", default=os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
    p.add_argument("--base")
    p.add_argument("--head")
    p.add_argument("--files", nargs="*")
    p.add_argument("--only")
    p.add_argument("--skip-catalog", action="store_true")
    args = ap.parse_args(argv)
    root = os.path.abspath(args.root)
    if not os.path.isdir(root):
        print("root is not a directory", file=sys.stderr)
        return 2
    return run(root, args)


if __name__ == "__main__":
    sys.exit(main())
