"""Read-only R078 exact-source/raw/cleanup audit. No Go or Docker execution."""
import hashlib
import json
import os
from pathlib import Path
import runpy
import subprocess


def digest(data):
    return hashlib.sha256(data).hexdigest()


def audit(role):
    prefix = 'docs/reviews/probes/r078_' + role + '_'
    path = Path(prefix + 'summary.json')
    raw = path.read_bytes()
    packet = json.loads(raw)
    harness = runpy.run_path(prefix + 'campaign.py')
    harness['validate_published_summary'](path, {'bytes': len(raw), 'sha256': digest(raw)})
    assert packet['result'] == 'PASS' and not packet['errors']
    assert packet['goInvocations'] == {'build': 1, 'tests': 3}
    assert packet['automaticRetries'] == 0 and not packet['notRun']
    source = 'b7c826aa9e412b9e463716d6449b99a78819bfa9'
    assert packet['sourceCommit'] == source
    archive = subprocess.check_output(['git', 'archive', '--format=tar', source, 'backend'])
    assert digest(archive) == packet['archive']['sha256']
    assert len(archive) == packet['archive']['bytes']
    assert packet['sourceBinding']['backendWorktreeClean']
    assert packet['sourceBinding']['sourceBackendTree'] == packet['sourceBinding']['headBackendTree']
    assert all(check['equal'] for check in packet['sourceChecks'])
    assert packet['M01Mutation']['restoredExactBytes']
    assert packet['M01Mutation']['originalSha256'] == packet['M01Mutation']['finalRestoredSha256']
    counts = {}
    for command in packet['commands']:
        assert command['imageId'] == 'sha256:8ac98ca534ac3f51e1f420a1dd2c15e74c75cfa0f23f3ad27eb5d7236c349a0c'
        state = command['state']
        assert not state['Running'] and not state['OOMKilled'] and not state['Error']
        assert state['ExitCode'] == command['attachExit']
        if command['label'] == 'build':
            assert state['ExitCode'] == 0 and command['network'] == 'none'
            continue
        events = harness['event_summary'](Path(command['stdout']['path']).read_bytes(), command['expectedTopLevelTests'])
        assert events == command['events']
        assert events['skip'] == 0 and not events['buildOrTimeout']
        if command['label'] == 'M01':
            assert state['ExitCode'] == 1 and events['namedAssertion']
            assert events['completed'] == [
                {'name': harness['M01_DETECTOR'], 'action': 'fail'},
                {'name': harness['M01_HEALTHY_CONTROL'], 'action': 'pass'},
            ]
        else:
            assert state['ExitCode'] == 0 and events['topLevel'] == 12 and events['fail'] == 0
        counts[command['label']] = {k: events[k] for k in ('topLevel', 'pass', 'fail', 'skip')}
    cleanup = packet['cleanup']
    assert cleanup['ownedResourcesOnly'] and cleanup['verifiedByExactName']
    assert cleanup['sourceSnapshot']['removed']
    for item in cleanup['containers'] + cleanup['volumes'] + [cleanup['network']]:
        assert item['absent'] and item['removeExit'] == 0
    assert not Path(packet['sourceDirectory']).exists()
    refs = harness['collect_raw_references'](packet)
    backup = Path(packet['rawBackupDirectory'])
    assert all((backup / Path(ref['path']).name).read_bytes() == Path(ref['path']).read_bytes() for ref in refs)
    return {'role': role, 'sourceCommit': source, 'summarySha256': digest(raw),
            'rawReferencesVerified': len(refs), 'counts': counts,
            'sourceRestoration': 'PASS', 'ownedCleanup': 'PASS',
            'archiveSha256': digest(archive), 'result': 'PASS'}


if __name__ == '__main__':
    import sys
    results = [audit(role) for role in sys.argv[1:]]
    baseline = json.loads((Path(os.environ['TEMP']) / 'ccmai_r078_baseline.json').read_text())
    for group in ('protectedSource', 'protectedHistory'):
        for path, expected in baseline[group].items():
            assert digest(Path(path).read_bytes()) == expected, path
    print(json.dumps({'campaigns': results, 'protectedSource': len(baseline['protectedSource']),
                      'protectedHistory': len(baseline['protectedHistory']), 'result': 'PASS'}, indent=2))
