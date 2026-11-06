"""R155 candidate preflight, UNINSTALLED: never grants historical admission.

This implements the bounded provenance front end only.  A successful preflight
returns UNSUPPORTED until the parent implements the coverage/audit join described
in README.md.  It must not be used as a bypass or integrated as a gate.
"""
import hashlib
import json
import re
import subprocess
from pathlib import Path, PurePosixPath

LIMIT = 1024 * 1024
HEX40 = re.compile(r"[0-9a-f]{40}\Z")
HEX64 = re.compile(r"[0-9a-f]{64}\Z")
KEYS = {"schemaVersion", "bindings"}
BINDING = {"milestone", "tranche", "packet", "admission", "build", "terminal",
           "registry", "audit", "seed", "terminalRecord", "reviewEvidence"}

class Reject(ValueError):
    pass

def require(ok, code):
    if not ok:
        raise Reject(code)

def pairs(items):
    result = {}
    for key, value in items:
        require(key not in result, "DUPLICATE_KEY")
        result[key] = value
    return result

def parse(raw):
    require(len(raw) <= LIMIT, "OVERSIZE")
    def nonfinite(_):
        raise Reject("NONFINITE")
    try:
        return json.loads(raw.decode("utf-8-sig"), object_pairs_hook=pairs,
                          parse_constant=nonfinite)
    except (UnicodeError, json.JSONDecodeError) as exc:
        raise Reject("INVALID_JSON") from exc

def shape(value, keys):
    require(type(value) is dict and set(value) == keys, "SHAPE")

def path(root, relative):
    require(type(relative) is str and relative and "\\" not in relative
            and ":" not in relative and not relative.startswith("/")
            and all(part not in ("", ".", "..") for part in relative.split("/")),
            "UNSAFE_PATH")
    parts = PurePosixPath(relative).parts
    target = root
    for part in parts:
        target = target / part
        require(not target.is_symlink(), "SYMLINK")
    require(target.is_file() and target.stat().st_size <= LIMIT, "MISSING_OR_OVERSIZE")
    return target

def digest(raw):
    return hashlib.sha256(raw.replace(b"\r\n", b"\n")).hexdigest()

def ref(root, value):
    shape(value, {"path", "sha256"})
    require(type(value["sha256"]) is str and HEX64.fullmatch(value["sha256"]), "HASH")
    raw = path(root, value["path"]).read_bytes()
    require(len(raw) <= LIMIT and digest(raw) == value["sha256"], "STALE")
    return raw

def git(root, *args):
    try:
        result = subprocess.run(["git", "-C", str(root), *args], capture_output=True,
                                timeout=10, check=False)
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise Reject("GIT_UNAVAILABLE") from exc
    return result

def committed(root, commit, relative, expected):
    # ls-tree mode prevents following Git symlinks and rejects missing/ambiguous entries.
    tree = git(root, "ls-tree", "-z", commit, "--", relative)
    require(tree.returncode == 0 and tree.stdout.count(b"\0") == 1, "TREE_ENTRY")
    meta, name = tree.stdout[:-1].split(b"\t", 1)
    require(meta.startswith((b"100644 blob ", b"100755 blob "))
            and name.decode("utf-8") == relative, "TREE_ENTRY")
    blob = git(root, "show", f"{commit}:{relative}")
    require(blob.returncode == 0 and len(blob.stdout) <= LIMIT
            and digest(blob.stdout) == expected, "COMMITTED_MISMATCH")

def preflight(root, raw, *, phase, checkout_head, packet_path, active=False, explicit=False):
    root = Path(root).resolve()
    require(phase == "pr-ci" and not active and not explicit, "EXECUTION_FORBIDDEN")
    require(HEX40.fullmatch(checkout_head or "") is not None, "HEAD_INVALID")
    head = git(root, "rev-parse", "--verify", "HEAD^{commit}")
    require(head.returncode == 0 and head.stdout.decode().strip() == checkout_head, "HEAD_MISMATCH")
    doc = parse(raw)
    shape(doc, KEYS)
    require(doc["schemaVersion"] == "cvf.historicalRepairAdmissions@candidate1", "SCHEMA")
    rows = doc["bindings"]
    require(type(rows) is list and 0 < len(rows) <= 100, "BINDINGS")
    seen_packets, seen_tranches = set(), set()
    selected = None
    for row in rows:
        shape(row, BINDING)
        require(type(row["milestone"]) is str and bool(row["milestone"]), "MILESTONE")
        require(type(row["tranche"]) is str and bool(row["tranche"]), "TRANCHE")
        shape(row["packet"], {"path", "sha256"})
        packet = row["packet"]["path"]
        require(type(packet) is str and packet not in seen_packets
                and row["tranche"] not in seen_tranches, "DUPLICATE_BINDING")
        seen_packets.add(packet); seen_tranches.add(row["tranche"])
        if packet == packet_path:
            selected = row
    require(selected is not None, "BINDING_MISSING")
    row = selected
    for name in ("admission", "build", "terminal"):
        require(type(row[name]) is str and HEX40.fullmatch(row[name]), "COMMIT_INVALID")
        require(git(root, "merge-base", "--is-ancestor", row[name], checkout_head).returncode == 0,
                "NONANCESTOR")
    for earlier, later in (("admission", "build"), ("build", "terminal")):
        require(row[earlier] != row[later] and
                git(root, "merge-base", "--is-ancestor", row[earlier], row[later]).returncode == 0,
                "ORDER")
    for name in ("packet", "registry", "audit", "seed", "terminalRecord"):
        ref(root, row[name])
    require(type(row["reviewEvidence"]) is list and bool(row["reviewEvidence"])
            and len(row["reviewEvidence"]) <= 32, "REVIEW_REQUIRED")
    for evidence in row["reviewEvidence"]:
        ref(root, evidence)
    for commit in (row["admission"], row["terminal"]):
        committed(root, commit, packet_path, row["packet"]["sha256"])
    committed(root, row["admission"], row["registry"]["path"], row["registry"]["sha256"])
    committed(root, row["admission"], row["audit"]["path"], row["audit"]["sha256"])
    committed(root, row["admission"], row["seed"]["path"], row["seed"]["sha256"])
    committed(root, row["terminal"], row["terminalRecord"]["path"], row["terminalRecord"]["sha256"])
    # Provenance alone does not prove frozen status, audit-action join, transition,
    # retained dependency validity or independent review. Refuse rather than pass.
    return "UNSUPPORTED_HISTORICAL_COVERAGE_JOIN"
