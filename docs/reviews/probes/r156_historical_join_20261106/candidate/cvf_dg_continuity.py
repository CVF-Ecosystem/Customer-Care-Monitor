"""CVF-DG-CONT-01: pinned cross-surface continuity consistency (stdlib only).

The downstream project schema (state 1.0 / IMPLEMENTATION_STATUS 1.0) differs
from the Core schema, so nothing here reads Core state paths or marker grammar.
Every comparison emits a field-specific locator; there is no heuristic alignment.
"""

from __future__ import annotations

import re
from pathlib import Path

from cvf_dg_applicability import load_checker
from cvf_dg_common import (
    ControlResult, Finding, declarations, labeled_lines, load_json_strict, markdown_section, read_text, safe_relative,
)

CONTROL_ID = "CVF-DG-CONT-01"


def _is_extension(key: str, prefixes: list[str]) -> bool:
    return any(key.startswith(prefix) for prefix in prefixes)


def _check_keys(data: dict, required: list[str], optional: list[str], prefixes: list[str],
                where: str, out: list[Finding]) -> None:
    for key in required:
        if key not in data:
            out.append(Finding("MISSING_FIELD", f"{where}#/{key}", f"required field {key} is missing"))
    for key in sorted(data):
        if key not in required and key not in optional and not _is_extension(key, prefixes):
            out.append(Finding("UNKNOWN_FIELD", f"{where}#/{key}", f"field {key} is not in the pinned contract (extensions use {'/'.join(prefixes)} prefixes)"))


def _text_field(data: dict, key: str, where: str, out: list[Finding]) -> str | None:
    value = data.get(key)
    if key not in data:
        return None
    if not isinstance(value, str) or not value.strip():
        out.append(Finding("INVALID_FIELD", f"{where}#/{key}", f"{key} must be a non-empty string"))
        return None
    return value.strip()


def _labels(text: str, section: str, required: list[str], optional: list[str], where: str,
            out: list[Finding]) -> dict[str, str]:
    bodies, count = markdown_section(text, section)
    if count == 0:
        out.append(Finding("MISSING_SECTION", f"{where}#{section}", f"section `## {section}` is missing"))
        return {}
    if count > 1:
        out.append(Finding("DUPLICATE_SECTION", f"{where}#{section}", f"section `## {section}` appears {count} times"))
        return {}
    found, malformed = labeled_lines(bodies[0])
    for line in malformed:
        out.append(Finding("MALFORMED_LINE", f"{where}#{section}", f"unparseable bullet: {line[:60]}"))
    result: dict[str, str] = {}
    for label, values in sorted(found.items()):
        if label not in required and label not in optional:
            out.append(Finding("UNKNOWN_FIELD", f"{where}#{section}/{label}", f"label {label} is not in the pinned contract"))
        elif len(values) > 1:
            out.append(Finding("DUPLICATE_FIELD", f"{where}#{section}/{label}", f"label {label} appears {len(values)} times"))
        else:
            result[label] = values[0]
    for label in required:
        if label not in found:
            out.append(Finding("MISSING_FIELD", f"{where}#{section}/{label}", f"required label {label} is missing"))
    return result


def _equal(label: str, pairs: list[tuple[str, str | None]], out: list[Finding]) -> None:
    present = [(loc, val) for loc, val in pairs if val is not None]
    if len({val for _, val in present}) > 1:
        detail = "; ".join(f"{loc}={val!r}" for loc, val in present)
        out.append(Finding("SURFACE_MISMATCH", present[0][0], f"{label} differs across surfaces: {detail}"))


def check_continuity(project_root: Path, profile: dict, checker_dir: Path) -> ControlResult:
    contract = profile["continuityContract"]
    prefixes = contract["extensionPrefixes"]
    out: list[Finding] = []
    state_rel, memory_rel, impl_rel = contract["statePath"], contract["memoryPath"], contract["implementationPath"]
    surfaces = {state_rel: project_root / state_rel, memory_rel: project_root / memory_rel, impl_rel: project_root / impl_rel}
    for rel, path in surfaces.items():
        if not path.is_file():
            out.append(Finding("MISSING_SURFACE", rel, "continuity surface is missing"))
    if out:
        return ControlResult(CONTROL_ID, "FAIL", out)
    try:
        state = load_json_strict(surfaces[state_rel])
        impl = load_json_strict(surfaces[impl_rel])
    except ValueError as exc:
        return ControlResult(CONTROL_ID, "FAIL", [Finding("INVALID_JSON", state_rel, f"{exc}")])
    if not isinstance(state, dict) or not isinstance(impl, dict):
        return ControlResult(CONTROL_ID, "FAIL", [Finding("INVALID_JSON", state_rel, "state and implementation status must be JSON objects")])
    if "continuityContract" not in state:
        return ControlResult(
            CONTROL_ID, "BLOCKED_MIGRATION_REQUIRED",
            [Finding("MIGRATION_REQUIRED", f"{state_rel}#/continuityContract",
                     f"state predates {contract['id']}; the project owner must adopt the pinned fields explicitly (no automatic rewrite)")],
        )
    if state.get("continuityContract") != contract["id"]:
        out.append(Finding("CONTRACT_PIN_MISMATCH", f"{state_rel}#/continuityContract", f"expected {contract['id']}"))
    _check_keys(state, contract["stateRequired"], contract["stateOptional"], prefixes, state_rel, out)
    _check_keys(impl, contract["implementationRequired"], contract["implementationOptional"], prefixes, impl_rel, out)
    if state.get("schemaVersion") not in contract["stateSchemaVersions"]:
        out.append(Finding("SCHEMA_VERSION_UNSUPPORTED", f"{state_rel}#/schemaVersion", f"supported: {contract['stateSchemaVersions']}"))
    if impl.get("schemaVersion") not in contract["implementationSchemaVersions"]:
        out.append(Finding("SCHEMA_VERSION_UNSUPPORTED", f"{impl_rel}#/schemaVersion", f"supported: {contract['implementationSchemaVersions']}"))
    mode = _text_field(state, "currentMode", state_rel, out)
    phase = _text_field(state, "activePhase", state_rel, out)
    role = _text_field(state, "activeRole", state_rel, out)
    next_move = _text_field(state, "nextAllowedMove", state_rel, out)
    handoff_rel = _text_field(state, "activeHandoff", state_rel, out)
    impl_phase = _text_field(impl, "currentPhase", impl_rel, out)
    if state.get("phaseModel") != contract["phaseModel"]:
        out.append(Finding("PHASE_MODEL_MISMATCH", f"{state_rel}#/phaseModel", "phaseModel differs from the pinned seven-step model"))
    if mode is not None and mode not in contract["modePhaseMap"]:
        out.append(Finding("UNKNOWN_MODE", f"{state_rel}#/currentMode", f"mode {mode!r} is not a pinned mode"))
    if phase is not None and phase not in contract["phaseModel"]:
        out.append(Finding("UNKNOWN_PHASE", f"{state_rel}#/activePhase", f"phase {phase!r} is not in the phase model"))
    if mode in contract["modePhaseMap"] and phase in contract["phaseModel"] and contract["modePhaseMap"][mode] != phase:
        out.append(Finding("MODE_PHASE_MISMATCH", f"{state_rel}#/currentMode", f"mode {mode} maps to phase {contract['modePhaseMap'][mode]} but activePhase is {phase}"))

    memory = _labels(read_text(surfaces[memory_rel]), contract["memorySection"], contract["memoryLabels"], [], memory_rel, out)
    if memory.get("Contract") not in (None, contract["id"]):
        out.append(Finding("CONTRACT_PIN_MISMATCH", f"{memory_rel}#{contract['memorySection']}/Contract", f"expected {contract['id']}"))
    handoff: dict[str, str] = {}
    handoff_path = safe_relative(project_root, handoff_rel) if handoff_rel else None
    handoff_loc = handoff_rel or state_rel + "#/activeHandoff"
    if handoff_rel and handoff_path is None:
        out.append(Finding("INVALID_HANDOFF_PATH", f"{state_rel}#/activeHandoff", "handoff path must be project-relative with forward slashes"))
    elif handoff_path is not None and not handoff_path.is_file():
        out.append(Finding("HANDOFF_MISSING", handoff_loc, "active handoff does not exist"))
    elif handoff_path is not None:
        htext = read_text(handoff_path)
        handoff = _labels(htext, contract["handoffSection"], contract["handoffRequiredLabels"], contract["handoffOptionalLabels"], handoff_loc, out)
        statuses = declarations(htext, "Status")
        if len(statuses) != 1:
            out.append(Finding("HANDOFF_STATUS_INVALID", handoff_loc, f"active handoff must declare exactly one unfenced Status line; found {len(statuses)} ({', '.join(v for _, v in statuses) or 'none'})"))
        elif statuses[0] != ("Status", "ACTIVE"):
            out.append(Finding("HANDOFF_NOT_ACTIVE", handoff_loc, f"active handoff status must be exactly `Status: ACTIVE`, found {statuses[0][0]}: {statuses[0][1]!r}"))
        handoff_dir = project_root / contract["handoffDir"]
        for other in sorted(handoff_dir.glob("*.md")) if handoff_dir.is_dir() else []:
            if other.resolve() != handoff_path.resolve() and any(v.strip().upper().startswith("ACTIVE") for _, v in declarations(read_text(other), "Status")):
                out.append(Finding("CONFLICTING_ACTIVE_HANDOFF", f"{contract['handoffDir']}/{other.name}", "a second handoff also declares an ACTIVE status"))

    _equal("currentMode", [(f"{state_rel}#/currentMode", mode), (f"{handoff_loc}#Current mode", handoff.get("Current mode")),
                           (f"{memory_rel}#Current mode", memory.get("Current mode"))], out)
    _equal("activePhase", [(f"{state_rel}#/activePhase", phase), (f"{handoff_loc}#Active phase", handoff.get("Active phase")),
                           (f"{memory_rel}#Active phase", memory.get("Active phase")), (f"{impl_rel}#/currentPhase", impl_phase)], out)
    _equal("activeRole", [(f"{state_rel}#/activeRole", role), (f"{handoff_loc}#Active role", handoff.get("Active role"))], out)
    _equal("nextAllowedMove", [(f"{state_rel}#/nextAllowedMove", next_move), (f"{handoff_loc}#Next allowed move", handoff.get("Next allowed move"))], out)
    _equal("activeHandoff", [(f"{state_rel}#/activeHandoff", handoff_rel), (f"{memory_rel}#Active handoff", memory.get("Active handoff"))], out)
    _equal("activeTranche", [(f"{state_rel}#/activeTranche", state.get("activeTranche")), (f"{impl_rel}#/activeTranche", impl.get("activeTranche")),
                             (f"{handoff_loc}#Active tranche", handoff.get("Active tranche"))], out)
    tranche_sources = [("activeTranche" in state), ("activeTranche" in impl), ("Active tranche" in handoff)]
    if any(tranche_sources) and not all(tranche_sources):
        out.append(Finding("TRANCHE_RELATION_INCOMPLETE", f"{state_rel}#/activeTranche", "activeTranche must be present in state, implementation status and handoff together"))

    work_orders = impl.get("activeWorkOrders")
    if isinstance(work_orders, list):
        checker = None
        seen: set[str] = set()
        prefix = profile["activeWorkOrderPrefix"]
        for index, item in enumerate(work_orders):
            target = safe_relative(project_root, item) if isinstance(item, str) else None
            where = f"{impl_rel}#/activeWorkOrders/{index}"
            if target is None or not target.is_file():
                out.append(Finding("ACTIVE_WORK_ORDER_MISSING", where, "active work order must be an existing project-relative path"))
                continue
            if item in seen:
                out.append(Finding("ACTIVE_WORK_ORDER_DUPLICATE", where, f"{item} is listed more than once"))
            seen.add(item)
            if not item.startswith(prefix) or not item.endswith(".md"):
                out.append(Finding("ACTIVE_WORK_ORDER_PATH_INVALID", where, f"an active work order must be a {prefix}*.md file"))
                continue
            try:
                checker = checker or load_checker(checker_dir)
            except Exception as exc:  # no status grammar available: an active order cannot be admitted
                out.append(Finding("CHECKER_UNAVAILABLE", where, f"{type(exc).__name__}: {exc}"))
                continue
            decision = checker.classify_status(item, read_text(target))
            if decision.outcome != "CHECK":
                out.append(Finding("ACTIVE_WORK_ORDER_NOT_ACTIVE", where, f"{item} is not an admitted active work order ({decision.outcome} {decision.code}: {decision.reason})"))
        if work_orders and phase in ("INTAKE", "DESIGN", "SPEC"):
            out.append(Finding("WORK_ORDER_BEFORE_PHASE", f"{impl_rel}#/activeWorkOrders", f"active work orders are not allowed while the phase is {phase}"))
    elif "activeWorkOrders" in impl:
        out.append(Finding("INVALID_FIELD", f"{impl_rel}#/activeWorkOrders", "must be a list"))
    return ControlResult(CONTROL_ID, "FAIL" if out else "PASS", out)
