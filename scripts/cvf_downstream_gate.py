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
import json
import os
import re
import shutil
import subprocess
import sys

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
    for f in out_of_scope[:20]:
        r.fail(f"changed path outside the authorized allowedPaths: {f}")
    if len(out_of_scope) > 20:
        r.fail(f"... and {len(out_of_scope) - 20} more paths outside allowedPaths")
    return r


ORDER_STATUS = re.compile(r"^Status:\s*`?([A-Z][A-Z_]+)", re.M)


def check_order_status(root, rec, r):
    tid, wo = rec.get("trancheId", "?"), rec.get("workOrder")
    if not wo or not inside(root, wo) or not os.path.isfile(os.path.join(root, wo)):
        return  # validate_record already reports a missing work order
    head = "\n".join(read_text(os.path.join(root, wo)).splitlines()[:20])
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
