"""Project-to-parent finding intake: deterministic, explicit, no daemon or remote submission.

`build_intake` turns one explicit finding input into a bounded record at the
review boundary. Generation is NOT acceptance: parent dedup, admission and
accept/decline links are written only by the CVF parent owner (F2G standard).
"""

from __future__ import annotations

import re
from pathlib import Path

from cvf_dg_common import Finding, canonical_json, content_sha256, load_json_strict

INPUT_FIELDS = ["findingId", "title", "sourceProject", "sourceSha", "observed", "expected", "negativeEvidence",
                "chainJoins", "defectClass", "learningLane", "proposedDisposition", "candidateControl",
                "claimLimits", "dedupCandidates"]
PARENT_LINK_KEYS = ["admissionRecord", "workOrder", "checker", "goldenProof"]
SECRET_PATTERNS = (re.compile(r"sk-[A-Za-z0-9]{16,}"), re.compile(r"AKIA[0-9A-Z]{16}"),
                   re.compile(r"(?i)(?:api[_-]?key|secret|token)\s*[=:]\s*\S{8,}"))
ID_PATTERN = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]{2,63}")


def validate_input(data: object, profile: dict) -> list[Finding]:
    spec = profile["intake"]
    out: list[Finding] = []
    if not isinstance(data, dict):
        return [Finding("INTAKE_INVALID", "$", "finding input must be a JSON object")]
    for key in sorted(set(data) - set(INPUT_FIELDS)):
        out.append(Finding("UNKNOWN_FIELD", f"$/{key}", "field is not part of the intake contract"))
    for key in INPUT_FIELDS:
        if key not in data:
            out.append(Finding("MISSING_FIELD", f"$/{key}", f"mandatory field {key} is missing"))
    if out:
        return out
    for key in ("title", "sourceProject", "observed", "expected", "candidateControl", "claimLimits"):
        if not isinstance(data[key], str) or len(data[key].strip()) < 8:
            out.append(Finding("INVALID_FIELD", f"$/{key}", f"{key} needs at least 8 characters of concrete text"))
    if not isinstance(data["findingId"], str) or not ID_PATTERN.fullmatch(data["findingId"]):
        out.append(Finding("INVALID_FIELD", "$/findingId", "findingId must be 3-64 chars of [A-Za-z0-9._-]"))
    if not isinstance(data["sourceSha"], str) or not re.fullmatch(r"[0-9a-f]{40}", data["sourceSha"]):
        out.append(Finding("INVALID_FIELD", "$/sourceSha", "sourceSha must be a lowercase 40-hex commit SHA"))
    for key, allowed in (("defectClass", spec["defectClasses"]), ("learningLane", spec["learningLanes"]),
                         ("proposedDisposition", spec["proposedDispositions"])):
        if data[key] not in allowed:
            out.append(Finding("INVALID_FIELD", f"$/{key}", f"{key} must be one of {allowed}"))
    evidence = data["negativeEvidence"]
    if not isinstance(evidence, list) or not evidence or not all(isinstance(e, dict) and set(e) == {"locator", "description"}
                                                                 and all(isinstance(v, str) and v.strip() for v in e.values()) for e in evidence):
        out.append(Finding("INVALID_FIELD", "$/negativeEvidence", "needs >=1 item of exactly {locator, description}"))
    joins = data["chainJoins"]
    if not isinstance(joins, list) or not joins or not all(isinstance(j, dict) and set(j) == {"join", "gap", "earliestControlPoint"}
                                                           and all(isinstance(v, str) and v.strip() for v in j.values()) for j in joins):
        out.append(Finding("INVALID_FIELD", "$/chainJoins", "needs >=1 item of exactly {join, gap, earliestControlPoint}"))
    dedup = data["dedupCandidates"]
    if not isinstance(dedup, list) or not all(isinstance(d, str) and d.strip() for d in dedup):
        out.append(Finding("INVALID_FIELD", "$/dedupCandidates", "must be a list of owner ids/paths (empty list = none known)"))
    blob = canonical_json(data)
    for pattern in SECRET_PATTERNS:
        if pattern.search(blob):
            out.append(Finding("SECRET_LIKE_CONTENT", "$", "input resembles a credential; intake never carries secrets or customer data"))
            break
    return out


def build_record(data: dict, profile: dict) -> dict:
    spec = profile["intake"]
    body = {
        "schemaVersion": spec["schemaVersion"],
        "profileId": profile["profileId"],
        **{key: data[key] for key in INPUT_FIELDS},
        "parentOwner": spec["parentOwner"],
        "acceptance": "NOT_ACCEPTED_GENERATION_ONLY",
        "parentDisposition": "PENDING_PARENT_ADMISSION",
        "parentLinks": {key: None for key in PARENT_LINK_KEYS},
        "sourceClaimStatus": "UNVERIFIED_UNTIL_PROJECT_EVIDENCE_IS_SEPARATELY_ADMITTED",
    }
    return {**body, "contentSha256": content_sha256(canonical_json(body).encode("utf-8"))}


def build_intake(finding_path: Path, project_root: Path, profile: dict) -> tuple[Path | None, list[Finding]]:
    try:
        data = load_json_strict(finding_path)
    except (OSError, ValueError) as exc:
        return None, [Finding("INTAKE_INVALID", str(finding_path), f"unreadable finding input: {exc}")]
    problems = validate_input(data, profile)
    if problems:
        return None, problems
    record = build_record(data, profile)  # type: ignore[arg-type]
    out_dir = project_root / profile["projectPaths"]["intakeDir"]
    out_dir.mkdir(parents=True, exist_ok=True)
    target = out_dir / f"{record['findingId']}.intake.json"
    text = canonical_json(record)
    if target.exists() and target.read_bytes().replace(b"\r\n", b"\n") != text.encode("utf-8"):
        return None, [Finding("INTAKE_EXISTS_DIFFERENT", str(target), "an intake with this findingId already exists with different content; refusing to overwrite")]
    target.write_bytes(text.encode("utf-8"))
    return target, []


def validate_record(record: object, profile: dict) -> list[Finding]:
    """Parent-side validation of an emitted record: exact key set, mandatory metadata values, link/state coherence and identity.

    A self-consistent digest never proves schema conformance, so every mandatory field is checked independently of it.
    """
    spec = profile["intake"]
    if not isinstance(record, dict):
        return [Finding("INTAKE_INVALID", "$", "record must be a JSON object")]
    generated = ["schemaVersion", "profileId", "parentOwner", "acceptance", "parentDisposition", "parentLinks", "sourceClaimStatus", "contentSha256"]
    expected = set(INPUT_FIELDS) | set(generated)
    out = [Finding("MISSING_FIELD", f"$/{k}", f"mandatory record field {k} is missing") for k in sorted(expected - set(record))]
    out += [Finding("UNKNOWN_FIELD", f"$/{k}", "field is not part of the record contract") for k in sorted(set(record) - expected)]
    if out:
        return out
    fixed = (("schemaVersion", spec["schemaVersion"]), ("profileId", profile["profileId"]), ("parentOwner", spec["parentOwner"]),
             ("sourceClaimStatus", spec["sourceClaimStatus"]))
    out += [Finding("INVALID_FIELD", f"$/{k}", f"{k} must be exactly {v!r}") for k, v in fixed if record[k] != v]
    out += validate_input({k: record[k] for k in INPUT_FIELDS}, profile)
    links = record["parentLinks"]
    if not isinstance(links, dict) or set(links) != set(spec["parentLinkKeys"]) or not all(v is None or (isinstance(v, str) and v.strip()) for v in links.values()):
        out.append(Finding("INVALID_FIELD", "$/parentLinks", f"must be an object with exactly {spec['parentLinkKeys']}, each a non-empty string or null"))
        links = {}
    acceptance, disposition = record["acceptance"], record["parentDisposition"]
    if acceptance not in spec["acceptanceStates"]:
        out.append(Finding("INVALID_FIELD", "$/acceptance", f"must be one of {spec['acceptanceStates']}"))
    if disposition not in spec["parentDispositions"]:
        out.append(Finding("INVALID_FIELD", "$/parentDisposition", f"must be one of {spec['parentDispositions']}"))
    pending = acceptance == "NOT_ACCEPTED_GENERATION_ONLY" and disposition == "PENDING_PARENT_ADMISSION"
    if pending and any(links.get(k) for k in links):
        out.append(Finding("STATE_LINK_INCOHERENT", "$/parentLinks", "a generated, pending record carries no parent link; links are written only by the parent owner"))
    if not pending and acceptance in spec["acceptanceStates"] and disposition in spec["parentDispositions"]:
        if acceptance != disposition or not links.get("admissionRecord"):
            out.append(Finding("STATE_LINK_INCOHERENT", "$/acceptance", "a parent decision needs acceptance equal to parentDisposition and an explicit admissionRecord link"))
    body = {k: v for k, v in record.items() if k != "contentSha256"}
    if record["contentSha256"] != content_sha256(canonical_json(body).encode("utf-8")):
        out.append(Finding("CONTENT_IDENTITY_MISMATCH", "$/contentSha256", "record content differs from its recorded identity"))
    return out
