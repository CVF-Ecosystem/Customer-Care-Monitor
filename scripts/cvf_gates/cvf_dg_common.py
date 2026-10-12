"""Shared primitives for the portable downstream gate profile (stdlib only).

Core-owned source: scripts/lib/downstream_governance/. Project copies are
identity-pinned by .cvf/gate-profile.lock.json and verified by CVF-DG-INST-01.
"""

from __future__ import annotations

import hashlib
import json
import re
import subprocess
from dataclasses import dataclass, field
from pathlib import Path

PROFILE_ID = "cvf.downstreamGateProfile@1.1.0"
PROFILE_FILE = "cvf_downstream_gate_profile.json"
OUTCOMES_OK = ("PASS", "NOT_APPLICABLE_WITH_REASON")
FENCED_BLOCK = re.compile(r"(?ms)^[ \t]*(```|~~~).*?^[ \t]*\1[ \t]*$")


@dataclass(frozen=True)
class Finding:
    code: str
    locator: str
    message: str

    def as_dict(self) -> dict[str, str]:
        return {"code": self.code, "locator": self.locator, "message": self.message}


@dataclass
class ControlResult:
    control_id: str
    outcome: str
    findings: list[Finding] = field(default_factory=list)
    reason: str = ""
    checked_control_ids: list[str] = field(default_factory=list)
    notes: list[str] = field(default_factory=list)

    @property
    def ok(self) -> bool:
        if self.outcome == "NOT_APPLICABLE_WITH_REASON":
            return bool(self.reason.strip()) and bool(self.checked_control_ids)
        return self.outcome == "PASS" and not self.findings

    def as_dict(self) -> dict:
        ordered = sorted(self.findings, key=lambda f: (f.locator, f.code, f.message))
        return {
            "controlId": self.control_id,
            "outcome": self.outcome,
            "ok": self.ok,
            "reason": self.reason,
            "checkedControlIds": list(self.checked_control_ids),
            "findings": [f.as_dict() for f in ordered],
            "notes": list(self.notes),
        }


def failed(control_id: str, findings: list[Finding], outcome: str = "FAIL") -> ControlResult:
    return ControlResult(control_id, outcome, findings)


def content_sha256(data: bytes) -> str:
    """LF-normalised identity so a CRLF checkout of the same text keeps its pin."""
    return hashlib.sha256(data.replace(b"\r\n", b"\n")).hexdigest()


def raw_sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def file_content_sha256(path: Path) -> str:
    return content_sha256(path.read_bytes())


class DuplicateKeyError(ValueError):
    pass


def _no_duplicate_pairs(pairs: list[tuple[str, object]]) -> dict:
    seen: dict[str, object] = {}
    for key, value in pairs:
        if key in seen:
            raise DuplicateKeyError(f"duplicate JSON key {key!r}")
        seen[key] = value
    return seen


def load_json_strict(path: Path) -> object:
    """Parse JSON (BOM tolerant) rejecting duplicate keys; raises ValueError."""
    text = path.read_bytes().decode("utf-8-sig")
    return json.loads(text, object_pairs_hook=_no_duplicate_pairs)


def canonical_json(value: object) -> str:
    return json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True) + "\n"


def strip_fences(text: str) -> str:
    return FENCED_BLOCK.sub("", text)


def read_text(path: Path) -> str:
    """UTF-8 (BOM tolerant) text with LF line endings so CRLF checkouts parse identically."""
    return path.read_bytes().decode("utf-8-sig", errors="replace").replace("\r\n", "\n")


def normalize_value(value: str) -> str:
    value = value.strip()
    if len(value) >= 2 and value[0] == "`" and value[-1] == "`":
        value = value[1:-1].strip()
    return value


def labeled_lines(section: str) -> tuple[dict[str, list[str]], list[str]]:
    """Return label->values for `- Label: value` bullets and any malformed bullets."""
    found: dict[str, list[str]] = {}
    malformed: list[str] = []
    for line in section.splitlines():
        stripped = line.strip()
        if not stripped.startswith("- "):
            continue
        match = re.match(r"^- ([A-Za-z][A-Za-z0-9 /_-]*?):[ \t]*(.*)$", stripped)
        if not match:
            malformed.append(stripped)
            continue
        found.setdefault(match.group(1), []).append(normalize_value(match.group(2)))
    return found, malformed


def markdown_section(text: str, heading: str) -> tuple[list[str], int]:
    """Bodies of every `## heading` section and how many headings matched."""
    bodies: list[str] = []
    pattern = re.compile(rf"(?m)^##[ \t]+{re.escape(heading)}[ \t]*$")
    matches = list(pattern.finditer(text))
    for match in matches:
        rest = text[match.end():]
        nxt = re.search(r"(?m)^##[ \t]+\S", rest)
        bodies.append(rest[: nxt.start()] if nxt else rest)
    return bodies, len(matches)


def scalar_fields(text: str, names: list[str]) -> dict[str, list[str]]:
    """Collect `name: value` scalar lines (outside fences) for the given names."""
    body = strip_fences(text)
    out: dict[str, list[str]] = {name: [] for name in names}
    for name in names:
        for match in re.finditer(rf"(?m)^[ \t]*(?:[-*][ \t]+)?{re.escape(name)}[ \t]*:[ \t]*(.*?)[ \t\r]*$", body):
            out[name].append(normalize_value(match.group(1)))
    return out


EMPTY_TREE = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
ZERO_SHA = re.compile(r"^0{40}$")


@dataclass
class CandidateRange:
    """Resolved candidate scope: a committed range, the worktree, or an explicit refusal."""

    mode: str = "UNRESOLVED"
    base: str | None = None
    head: str | None = None
    paths: list[str] = field(default_factory=list)
    deleted: list[str] = field(default_factory=list)
    findings: list[Finding] = field(default_factory=list)

    @property
    def disposition(self) -> str:
        if self.findings:
            return "RANGE_REFUSED"
        if self.mode in ("RANGE", "RANGE_INITIAL") and not self.paths and not self.deleted:
            return "RANGE_EMPTY_CHECKED"
        return {"RANGE": "RANGE_RESOLVED", "RANGE_INITIAL": "RANGE_INITIAL_ALL_FILES"}.get(self.mode, self.mode)

    def summary(self) -> str:
        return (f"{self.disposition} mode={self.mode} base={self.base} head={self.head} "
                f"candidates={len(self.paths)} deleted={len(self.deleted)}")


def _git(root: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(["git", *args], cwd=root, text=True, encoding="utf-8", errors="replace",
                          stdout=subprocess.PIPE, stderr=subprocess.PIPE)


def _parse_name_status(output: str, into: CandidateRange) -> None:
    for line in output.splitlines():
        parts = line.split("\t")
        if len(parts) < 2 or not parts[0]:
            continue
        status = parts[0][0]
        target = parts[-1].replace("\\", "/")
        if status == "D":
            into.deleted.append(target)
        elif status in "AMTRC":
            into.paths.append(target)
            if status == "R":
                into.deleted.append(parts[1].replace("\\", "/"))


def resolve_candidates(root: Path, base: str | None, head: str | None, require_range: bool) -> CandidateRange:
    """Resolve the candidate change set. A required range that cannot be resolved is refused, never emptied."""
    out = CandidateRange()
    if not (root / ".git").exists():
        out.mode = "NO_GIT_NO_CANDIDATES"
        if require_range or base:
            out.findings.append(Finding("RANGE_UNRESOLVED", "git", "a candidate range needs a git worktree"))
        return out
    if require_range and not base:
        out.findings.append(Finding("RANGE_BASE_MISSING", "--base", "this phase needs the PR/push base commit; no range means no candidate discovery"))
        return out
    head_arg = head or "HEAD"
    head_sha = _git(root, "rev-parse", "--verify", "--quiet", f"{head_arg}^{{commit}}").stdout.strip()
    current = _git(root, "rev-parse", "--verify", "--quiet", "HEAD^{commit}").stdout.strip()
    if base:
        if not head_sha:
            out.findings.append(Finding("RANGE_HEAD_UNRESOLVED", head_arg, "head revision does not resolve to a commit"))
            return out
        if head_sha != current:
            out.findings.append(Finding("RANGE_HEAD_NOT_CHECKED_OUT", head_arg, "candidate files are read from the checkout, so head must be the checked-out HEAD"))
            return out
        out.head = head_sha
        if ZERO_SHA.match(base):
            out.mode, out.base = "RANGE_INITIAL", EMPTY_TREE
        else:
            base_sha = _git(root, "rev-parse", "--verify", "--quiet", f"{base}^{{commit}}").stdout.strip()
            if not base_sha:
                out.findings.append(Finding("RANGE_BASE_UNRESOLVED", base, "base commit is not available in this checkout (shallow clone or wrong revision)"))
                return out
            merge_base = _git(root, "merge-base", base_sha, head_sha).stdout.strip()
            if not merge_base:
                out.findings.append(Finding("RANGE_MERGE_BASE_UNRESOLVED", base, "no merge base between base and head (unrelated or truncated history)"))
                return out
            out.mode, out.base = "RANGE", merge_base
        diff = _git(root, "diff", "--name-status", "--find-renames", out.base, head_sha)
        if diff.returncode:
            out.findings.append(Finding("RANGE_DIFF_FAILED", out.base, (diff.stderr or "git diff failed").strip()[:200]))
            return out
        _parse_name_status(diff.stdout, out)
    else:
        out.mode = "WORKTREE_SCOPE"
        out.head = current or None
    for args in (("diff", "--name-status", "--find-renames", "HEAD"), ("diff", "--cached", "--name-status", "--find-renames", "HEAD")):
        extra = _git(root, *args)
        if extra.returncode == 0:
            _parse_name_status(extra.stdout, out)
    untracked = _git(root, "ls-files", "--others", "--exclude-standard")
    if untracked.returncode == 0:
        out.paths.extend(u for u in untracked.stdout.splitlines() if u.strip())
    out.paths = sorted({p.replace("\\", "/") for p in out.paths if p.strip()} - set(out.deleted))
    out.deleted = sorted({p.replace("\\", "/") for p in out.deleted if p.strip()} - set(out.paths))
    return out


def discover_packets(root: Path, explicit: list[str], candidates: CandidateRange, prefix: str) -> list[str]:
    """Markdown packets under `prefix` from the resolved candidate set plus explicit paths."""
    found = {p for p in candidates.paths if p.startswith(prefix) and p.endswith(".md") and (root / p).is_file()}
    found.update(p.replace("\\", "/") for p in explicit if p.replace("\\", "/").startswith(prefix))
    return sorted(found)


def declarations(text: str, label: str) -> list[tuple[str, str]]:
    """Every unfenced `label:` line, matched case-insensitively, as (spelled label, raw value)."""
    spaced = re.escape(label).replace(r"\ ", r"[ \t]+")
    pattern = re.compile(rf"(?im)^[ \t]*(?:[-*][ \t]+)?({spaced})[ \t]*:[ \t]*(.*?)[ \t\r]*$")
    return [(m.group(1), m.group(2)) for m in pattern.finditer(strip_fences(text))]


def bundle_digest(directory: Path) -> tuple[str, dict[str, str]]:
    """Digest over EVERY file under `directory` (name + LF-normalised SHA-256); parity-tested with the runner stage-0 copy."""
    files = {p.relative_to(directory).as_posix(): content_sha256(p.read_bytes()) for p in directory.rglob("*") if p.is_file()}
    files = dict(sorted(files.items()))  # ordinal order of the posix name: identical on Windows and Linux
    listing = "".join(f"{k}:{v}\n" for k, v in files.items())
    return hashlib.sha256(listing.encode("utf-8")).hexdigest(), files


def safe_relative(root: Path, relative: str) -> Path | None:
    """Resolve a forward-slash project-relative path; None if absolute/escaping/backslashed."""
    if not relative or "\\" in relative or relative.startswith("/") or re.match(r"^[A-Za-z]:", relative):
        return None
    parts = relative.split("/")
    if any(part in ("..", "") for part in parts):
        return None
    candidate = (root / relative).resolve()
    try:
        candidate.relative_to(root.resolve())
    except ValueError:
        return None
    return candidate
