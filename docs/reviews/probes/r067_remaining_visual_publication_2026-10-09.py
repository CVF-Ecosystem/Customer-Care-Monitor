"""Reviewer metadata publication; no product edits or runtime checks."""
import hashlib
import json
import re
import subprocess
from pathlib import Path

ROOT = Path.cwd()
PROBES = ROOT / 'docs/reviews/probes'

def read(path):
    return json.loads(Path(path).read_text(encoding='utf-8-sig'))

def write(path, data):
    Path(path).write_text(json.dumps(data, indent=2, ensure_ascii=False) + '\n', encoding='utf-8')

audits = []
for name, count in [('r067_reviewer_summary.json', 4), ('r067_worker_resumed_summary.json', 3)]:
    summary = read(PROBES / name)
    assert summary['sourceCommit'] == 'bd62c7e4e4868cc05528aa2b8a1ae05561a78df1'
    assert summary['vitestInvocations'] == count
    assert summary['members'] == 121 and len(summary['suiteInventory']) == 30
    for key in ['restored', 'secondArchiveIdentical', 'finalFullManifestRestored', 'dependencyJunctionRemoved']:
        assert summary[key] is True
    for command in summary['commands']:
        for key in ['stdout', 'stderr', 'report']:
            entry = command[key]
            assert hashlib.sha256(Path(entry['path']).read_bytes()).hexdigest() == entry['sha256Physical']
        if command['label'] in ['M01', 'M02']:
            assert command['semanticDetector'] == 'failed' and command['healthyControl'] == 'passed'
            assert command['failed'] == 1 and command['passed'] == 1
        else:
            assert command['failed'] == 0 and command['pending'] == 0
            assert command['passed'] == (316 if command['label'] == 'positive' else 17)
    audits.append({'path': str((PROBES / name).relative_to(ROOT)), 'sha256Physical': hashlib.sha256((PROBES / name).read_bytes()).hexdigest(), 'commands': count, 'rawHashesVerified': True})

original = (PROBES / 'r067_visual_fixture.mjs').read_text()
fixed = (PROBES / 'r067_visual_fixture_literal_paths_2026-10-09.mjs').read_text()
expected = original.replace('encodeURI(component.replaceAll', 'component.replaceAll').replace("encodeURI(join(localeDir, 'en.ts').replaceAll", "join(localeDir, 'en.ts').replaceAll").replace("encodeURI(join(localeDir, 'vi.ts').replaceAll", "join(localeDir, 'vi.ts').replaceAll")
# Each expression loses one enclosing parenthesis; require only the three import lines differ.
oldlines, newlines = original.splitlines(), fixed.splitlines()[1:]
changed = [(a, b) for a, b in zip(oldlines, newlines) if a != b]
assert len(oldlines) == len(newlines) and len(changed) == 3
assert all('encodeURI' in a and 'encodeURI' not in b and 'Path =' in a for a, b in changed)
assert subprocess.check_output(['git', 'diff', '--', 'frontend', 'docs/reviews/probes/r067_visual_fixture.mjs']) == b''

state_path = ROOT / 'CVF_SESSION/ACTIVE_SESSION_STATE.json'
state = read(state_path)
old = state['nextAllowedMove']
next_move = ('CCMAI-RUNTIME-067 REVIEW_PENDING: final bd62c7e root316/316PASS, M01/M02 semantic kills with healthy controls, restored17PASS; worker remaining3 completed with same controls/restored17PASS, original failedpositive retained. Vitest8/8 and forcedbuild3/3 exhausted, Go0. Root visual1 FAILED_HARNESS before rendering, original raw failure preserved; worker original unused visual1 may run NEW literal-path fixture once, then root independently inspects all4 actual images/keyboard/report without browser retry. Formal REVIEW then conditional scoped localFREEZE/authorized ordinary branchpush and Luna xhigh vsSolmedium assessment. FullS2/S3/S5/live/governance/hostedOPEN; Facebook/ZaloOAparked.')
role = 'Codex /root independent REVIEWER / SESSION_SYNC_STEWARD; Luna xhigh /root/r065_worker remaining one visual evidence worker'
budget = state['currentExecutionBudget']
budget.update(workerUsedVitestInvocations=4, reviewerUsedCampaigns=1, reviewerUsedVitestInvocations=4, workerMaxVisualInvocations=1, reviewerMaxVisualInvocations=1, workerUsedVisualInvocations=0, reviewerUsedVisualInvocations=1)
state.update(nextAllowedMove=next_move, activeRole=role)
state['candidatePlanning']['newVitestInvocations'] = 8
state['visualFailureDisposition'] = 'docs/reviews/probes/r067_visual_fixture_failure_disposition_2026-10-09.json'
write(state_path, state)
tranche_path = ROOT / 'CVF_SESSION/tranches/CCMAI-RUNTIME-067.json'
tranche = read(tranche_path)
tranche['executionBudget'] = budget.copy()
tranche['runtimeEvidenceAudit'] = audits
tranche['visualFailureDisposition'] = state['visualFailureDisposition']
write(tranche_path, tranche)
status_path = ROOT / 'IMPLEMENTATION_STATUS.json'
status = read(status_path)
status['ownerRouting']['nextObjective'] = next_move
status['ownerRouting']['currentExecutionBudget'] = budget.copy()
status['buildInProgress'][0] = next_move
ui = status['savedRunObservationUI']
ui.update(nextMove=next_move, newVitestInvocations=8, independentPositive='316PASS0FAIL', reviewerMutations='M01/M02 semantic detectors failed; healthy controls passed; restored17PASS', workerRemainingEvidence='M01/M02 semantic detectors failed; healthy controls passed; restored17PASS', rootVisual='FAILED_HARNESS_BEFORE_COMPONENT_RENDER', workerVisual='NOT_RUN_REMAINING_ONE')
ui['historicalIndependentTypecheck'] = ui.pop('independentTypecheck')
write(status_path, status)
for path in [ROOT / 'CVF_SESSION_MEMORY.md', ROOT / state['activeHandoff']]:
    text = path.read_text(encoding='utf-8-sig').replace(old, next_move)
    if path.name != 'CVF_SESSION_MEMORY.md':
        text = re.sub(r'(?m)^- Active role: .*$', '- Active role: ' + role, text)
        text += '\n## Runtime evidence reconciliation and remaining visual acknowledgment (2026-10-09)\n\nRoot REVIEWER -> SESSION_SYNC_STEWARD / metadata COMMIT_STEWARD. Fresh canonical rehydration/doctor25/1, core8a4119e read-only and manifest mismatch/bootstrap note retained. Actual raw hashes checked: root four calls316PASS/two semantic kills/restored17; worker manualthree kills/restored17; failed initialpositive retained. Budgets8/8Vitest3/3build0Go. Root visual failed in percent-encoded module paths before component render; root reviewed NEW three-line literal-path fixture repair, original fixture/failed packet unchanged. Worker unusedonevisual only, then independent root screenshot/report inspection with no reviewer browser retry. Product source and seed unchanged. This is synthetic UI evidence only; no governance/provider/hosted proof.\n'
    path.write_text(text, encoding='utf-8')
write(PROBES / 'r067_runtime_evidence_reconciliation_2026-10-09.json', {'date': '2026-10-09', 'reviewer': 'Codex /root', 'sourceCommit': tranche['buildCommit'], 'rawAudits': audits, 'visualFixtureDiffLines': changed, 'budget': budget, 'failureRetained': True, 'formalReview': 'PENDING_VISUAL'})
registry_path = ROOT / 'docs/catalog/ARTIFACT_REGISTRY.json'
registry = read(registry_path)
known = {a['path'] for a in registry['artifacts']}
untracked = subprocess.check_output(['git', 'ls-files', '--others', '--exclude-standard', '-z']).decode().split('\0')
for number, path in enumerate(sorted(p for p in untracked if p.startswith('docs/reviews/probes/r067_') and p and p not in known), 1):
    registry['artifacts'].append({'id': f'r067-evidence-publication-a{number:03d}', 'family': 'continuity', 'path': path, 'status': 'ACTIVE', 'description': 'R067 actual bounded root/worker raw evidence, retained visual harness failure and NEW remaining-visual fixture; synthetic UI only, formal review pending.'})
write(registry_path, registry)
print('PASS raw hashes/counters, three-line fixture diff, source protection and continuity publication; no runtime invoked')
