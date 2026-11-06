"""Reproducible proposal patch construction; not a test invocation."""
from pathlib import Path
import difflib
import hashlib
import json

HERE = Path(__file__).resolve().parent
PROJECT = HERE.parents[3]
SOURCE = PROJECT / 'scripts/cvf_gates'
TARGET = HERE / 'candidate'

def edit(name, changes):
    text = (SOURCE / name).read_text(encoding='utf-8')
    for old, new in changes:
        assert old in text, (name, old)
        text = text.replace(old, new)
    (TARGET / name).write_text(text, encoding='utf-8', newline='\n')

edit('cvf_repair_convergence.py', [
    ('def validate(root, data, registry, api):', 'def validate(root, data, registry, api, historical=None):'),
    ('    if data["action"]["kind"] not in ("IMPLEMENT", "VERIFY") or not affected:',
     '    if historical is not None:\n'
     '        from cvf_historical_admissions import select\n'
     '        from types import SimpleNamespace\n'
     '        historical_audit = select(root, api.packet_path, historical, registry,\n'
     '            SimpleNamespace(controls=controls, parse=parse, inspect_control=inspect_control, REGISTRY=REGISTRY), api)\n'
     '        if historical_audit is not None:\n'
     '            api.need(historical_audit["milestoneId"] == milestone, "HIST_MILESTONE")\n'
     '            audit = historical_audit\n'
     '    if data["action"]["kind"] not in ("IMPLEMENT", "VERIFY") or not affected:')])

edit('cvf_milestone_coverage.py', [
    ('def _repair_api(packet_path=None, packet_text=None):',
     'def _repair_api(packet_path=None, packet_text=None, strict=False):'),
    ('        if key not in cache:\n            raw = read_file(root, path)',
     '        if key not in cache:\n'
     '            if strict:\n'
     '                from cvf_historical_admissions import Reader\n'
     '                Reader(root, SimpleNamespace(need=need)).safe(path)\n'
     '            raw = read_file(root, path)'),
    ('def validate(root: Path, data, packet_path=None, packet_text=None)',
     'def validate(root: Path, data, packet_path=None, packet_text=None, historical=None)'),
    ('        _validate(root, data, packet_path, packet_text)',
     '        if historical is not None:\n'
     '            _repair().transition(root, historical["base"], _repair_api(strict=True))\n'
     '        _validate(root, data, packet_path, packet_text, historical)'),
    ('def _validate(root: Path, data, packet_path=None, packet_text=None):',
     'def _validate(root: Path, data, packet_path=None, packet_text=None, historical=None):'),
    ('_repair().validate(root, data, registry, _repair_api(packet_path, packet_text))',
     '_repair().validate(root, data, registry, _repair_api(packet_path, packet_text, strict=historical is not None), historical)'),
    ('def validate_text(root: Path, relative: str, value: str, required=None)',
     'def validate_text(root: Path, relative: str, value: str, required=None, historical=None)'),
    ('issues = validate(root, data, relative, value)', 'issues = validate(root, data, relative, value, historical)'),
    ('def validate_file(root: Path, relative: str)',
     'def validate_historical_context(root, historical):\n'
     '    try:\n'
     '        from cvf_historical_admissions import select\n'
     '        api = _repair_api(strict=True)\n'
     '        _repair().transition(root, historical["base"], api)\n'
     '        registry = _repair().parse(read_file(root, AUTHORITY_REGISTRY), api)\n'
     '        select(root, None, historical, registry, _repair(), api)\n'
     '        return []\n'
     '    except (Invalid, OSError, UnicodeError, ValueError, TypeError, KeyError) as exc:\n'
     '        return [str(exc) if isinstance(exc, Invalid) else "HIST_INVALID_INPUT:" + type(exc).__name__]\n\n\n'
     'def validate_file(root: Path, relative: str)'),
    ('def validate_file(root: Path, relative: str)', 'def validate_file(root: Path, relative: str, historical=None)'),
    ('return validate_text(root, relative, read_file(root, relative).decode("utf-8-sig"))',
     'return validate_text(root, relative, read_file(root, relative).decode("utf-8-sig"), historical=historical)')])

edit('cvf_downstream_gate_runner.py', [
    ('def _milestone_result(M: dict, root: Path, packets: list[str], profile: dict, base=None):',
     'def _milestone_result(M: dict, root: Path, packets: list[str], profile: dict, base=None, historical=None):'),
    ('    findings, checked = [], []', '    findings, checked = [], []\n    orders = []'),
    ('        issues = coverage.validate_file(root, rel)',
     '        context = None if historical is None else dict(historical, active=orders)\n'
     '        issues = coverage.validate_file(root, rel, historical=context)'),
    ('    for rel in sorted(candidates):',
     '    if historical is not None:\n'
     '        historical = dict(historical, active=orders)\n'
     '        findings.extend(Finding(code, "historical-admissions", "retrospective binding refused")\n'
     '                        for code in coverage.validate_historical_context(root, historical))\n'
     '    for rel in sorted(candidates):'),
    ('profile, candidates.base))',
     'profile, candidates.base, historical=None if not args.historical_admissions else {\n'
     '                "phase": phase, "head": candidates.head, "base": candidates.base, "active": [],\n'
     '                "explicit": args.packet or [], "manifest": {"path": args.historical_admissions,\n'
     '                "sha256": args.historical_admissions_sha256, "commit": args.historical_admissions_commit}}))'),
    ('    run.add_argument("--head")',
     '    run.add_argument("--head")\n'
     '    run.add_argument("--historical-admissions", help="UNINSTALLED candidate proposed binding path")\n'
     '    run.add_argument("--historical-admissions-sha256")\n'
     '    run.add_argument("--historical-admissions-commit")')])

names = ('cvf_downstream_gate_runner.py', 'cvf_milestone_coverage.py', 'cvf_repair_convergence.py')
diff = ''.join(''.join(difflib.unified_diff((SOURCE/n).read_text().splitlines(True),
        (TARGET/n).read_text().splitlines(True), fromfile='a/'+n, tofile='b/'+n)) for n in names)
(HERE/'parent_candidate.patch').write_text(diff, encoding='utf-8')
def sha(path):
    return hashlib.sha256(path.read_bytes().replace(b'\r\n', b'\n')).hexdigest()
manifest = {p.name: {'source': sha(SOURCE/p.name) if (SOURCE/p.name).is_file() else None,
                     'candidate': sha(p), 'modified': p.name in names or not (SOURCE/p.name).exists()}
            for p in TARGET.iterdir() if p.is_file()}
(HERE/'source_manifest.json').write_text(json.dumps(manifest, indent=2)+'\n', encoding='utf-8')
