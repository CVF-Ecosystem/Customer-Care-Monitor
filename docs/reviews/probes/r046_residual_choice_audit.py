"""Read-only audit of explicitly declared R2 reviewer logs and restored archives.

Run from project root. Generates decision evidence only; never executes runtime.
"""
import collections
import hashlib
import io
import json
import pathlib
import subprocess
import tarfile


def sha(data):
    return hashlib.sha256(data).hexdigest()


def main():
    summary = json.loads(pathlib.Path('docs/reviews/probes/r046_r2_independent_summary.json').read_text())
    ref = summary['sourceCommit']
    canonical = subprocess.check_output(['git', 'archive', ref, 'backend'])
    archive = tarfile.open(fileobj=io.BytesIO(canonical))
    expected = {m.name: archive.extractfile(m).read() for m in archive.getmembers() if m.isfile()}
    result = {'sourceCommit': ref, 'auditDate': '2026-10-05', 'runtimeExecuted': False,
              'governanceProof': False, 'campaigns': [], 'limits':
              'Recovered independent reviewer R2 evidence only; does not certify Claude historical execution, resource exits or empty build/vet process exits independently.'}
    for campaign in summary['campaigns']:
        task = pathlib.Path(campaign['taskPath'])
        runs = []
        for run in campaign['runs']:
            raw = (task / (run['name'] + '.jsonl')).read_bytes()
            encoding = 'utf-16' if raw.startswith((b'\xff\xfe', b'\xfe\xff')) else 'utf-8-sig'
            events = [json.loads(line) for line in raw.decode(encoding).splitlines() if line.strip()]
            done = [e for e in events if e.get('Test') and e.get('Action') in ('pass', 'fail', 'skip')]
            counts = collections.Counter(e['Action'] for e in done)
            actual = {a: counts[a] for a in ('pass', 'fail', 'skip')}
            passes = [e['Test'] for e in done if e['Action'] == 'pass']
            failures = [e['Test'] for e in done if e['Action'] == 'fail']
            row = {'name': run['name'], 'digestMatches': sha(raw) == run['logSha256'],
                   'rawLogSha256': sha(raw), 'counts': actual,
                   'countsMatch': actual == run['counts'],
                   'topLevelPass': sum('/' not in name for name in passes),
                   'passNamesMatch': sorted(passes) == sorted(run['passedTests']),
                   'failNamesMatch': sorted(failures) == sorted(run['failures']),
                   'emptyBuildVetLog': not raw and run['name'] in ('build', 'vet')}
            assert all(row[k] for k in ('digestMatches', 'countsMatch', 'passNamesMatch', 'failNamesMatch')), row['name']
            assert row['topLevelPass'] == run['topLevelPass'], row['name']
            runs.append(row)
        unequal = []
        source = task / 'source' / 'backend'
        for name, data in expected.items():
            candidate = source / name.removeprefix('backend/')
            if not candidate.exists() or candidate.read_bytes() != data:
                unequal.append(name)
        names = {p.relative_to(source).as_posix() for p in source.rglob('*') if p.is_file()}
        extra = sorted(names - {p.removeprefix('backend/') for p in expected})
        assert not unequal and not extra, (unequal, extra)
        result['campaigns'].append({'reviewRound': campaign['reviewRound'], 'runs': runs,
                                   'restoredArchiveFiles': len(expected), 'byteDifferences': unequal,
                                   'extraFiles': extra})
    committed = subprocess.check_output(['git', 'rev-parse', ref + ':backend']).decode().strip()
    current = subprocess.check_output(['git', 'rev-parse', 'HEAD:backend']).decode().strip()
    assert committed == current
    result['backendTreeOid'] = current
    result['backendTreeUnchanged'] = True
    result['result'] = 'PASS_RECOVERED_REVIEWER_EVIDENCE_AUDIT'
    pathlib.Path('docs/reviews/probes/r046_residual_choice_audit.json').write_text(json.dumps(result, indent=2) + '\n', encoding='utf-8')
    print(json.dumps({'result': result['result'], 'campaigns': len(result['campaigns']),
                      'logs': sum(len(c['runs']) for c in result['campaigns']),
                      'restoredFilesPerArchive': len(expected), 'backendTreeUnchanged': True}))


if __name__ == '__main__':
    main()
