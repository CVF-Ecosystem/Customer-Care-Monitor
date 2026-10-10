"""R071 shared capture infrastructure; independent role archives, no retries.

Product/tests belong to Luna. Root provides this harness; retain that assistance
in the quality ledger. No provider/database/application runtime governance proof.
"""
import argparse
import difflib
import hashlib
import io
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tarfile
import tempfile
import time

ROOT = Path(__file__).resolve().parents[3]
OUT = ROOT / 'docs/reviews/probes'

def digest(data):
    return hashlib.sha256(data).hexdigest()

def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT)

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--role', choices=['worker', 'root'], required=True)
    parser.add_argument('--mode', choices=['compile', 'tests'], required=True)
    parser.add_argument('--plan', required=True)
    parser.add_argument('--stage', choices=['first', 'repair'], default='first')
    args = parser.parse_args()
    plan_path = Path(args.plan).resolve()
    assert plan_path.is_relative_to(ROOT)
    plan_relative = plan_path.relative_to(ROOT).as_posix()
    assert git('show', 'HEAD:' + plan_relative).replace(b'\r\n', b'\n') == plan_path.read_bytes().replace(b'\r\n', b'\n')
    self_relative = Path(__file__).relative_to(ROOT).as_posix()
    assert git('show', 'HEAD:' + self_relative).replace(b'\r\n', b'\n') == Path(__file__).read_bytes().replace(b'\r\n', b'\n')
    plan = json.loads(plan_path.read_text(encoding='utf-8-sig'))
    source = plan['sourceCommit']
    assert re.fullmatch('[0-9a-f]{40}', source)
    rec = json.loads((ROOT / 'CVF_SESSION/tranches/CCMAI-RUNTIME-071.json').read_text())
    assert rec['status'] == 'BUILD'
    assert source in rec['rootStaticApprovedSources']
    assert plan_relative in rec['rootStaticApprovedPlans']
    assert git('show', 'HEAD:CVF_SESSION/tranches/CCMAI-RUNTIME-071.json').replace(b'\r\n', b'\n') == (ROOT / 'CVF_SESSION/tranches/CCMAI-RUNTIME-071.json').read_bytes().replace(b'\r\n', b'\n')
    archive = git('archive', '--format=tar', source, 'backend')
    assert digest(archive) == plan['backendArchiveSha256']
    with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
        members = {m.name: tar.extractfile(m).read() for m in tar if m.isfile()}
    assert len(members) == plan['backendArchiveMembers']
    assert all(p.startswith('backend/') and '..' not in Path(p).parts for p in members)
    legacy, new = plan['legacyPureTestNames'], plan['newTestNames']
    assert len(legacy) == 10 and new
    assert len(set(legacy + new)) == len(legacy + new)
    assert all(re.fullmatch(r'Test\w+', n) for n in legacy + new)
    mutations = plan['mutations']
    assert len(mutations) == 2
    assert [m['label'] for m in mutations] == ['m01', 'm02']
    assert [m['diagnosticContains'] for m in mutations] == ['APO:M01', 'APO:M02']
    for mutation in mutations:
        assert mutation['path'] == 'backend/engine/adapter_usage_presence_observation.go'
        assert members[mutation['path']].decode().count(mutation['old']) == 1
        assert mutation['old'] != mutation['new']
        assert mutation['detector'] in new and mutation['healthy'] in new
        assert mutation['diagnosticContains'] in ['APO:M01', 'APO:M02']
    prefix = f'r071_runtime_{args.role}_{args.mode}_{args.stage}'
    packet_dir = OUT / prefix
    packet_dir.mkdir()  # Exclusive reservation: never overwrite/retry a packet.
    scratch = Path(tempfile.mkdtemp(prefix=f'ccmai-r071-{args.role}-{args.mode}-'))
    for name, data in members.items():
        path = scratch / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data)
    (scratch / 'source-before.tar').write_bytes(archive)
    def snapshot():
        return {p.relative_to(scratch).as_posix(): dict(bytes=p.stat().st_size, sha256=digest(p.read_bytes()))
                for p in sorted((scratch / 'backend').rglob('*')) if p.is_file()}
    baseline = snapshot()
    def save(name, value):
        path = packet_dir / name
        with path.open('xb') as fh:
            fh.write((json.dumps(value, indent=2) + '\n').encode())
    fixed = dict(GOPROXY='off', GOSUMDB='off', GOTOOLCHAIN='local', CGO_ENABLED='0')
    env = os.environ.copy()
    env.update(fixed)
    env.pop('TEST_DB_DSN', None)
    exe = shutil.which('go')
    packet = dict(date='2026-10-10', sourceCommit=source, role=args.role, mode=args.mode, stage=args.stage,
                  approvalCommit=git('rev-parse', 'HEAD').decode().strip(), archiveSha256=digest(archive),
                  members=len(members), scratch=str(scratch), plan=plan_relative,
                  planSha256Physical=digest(plan_path.read_bytes()), fixedEnvironment=fixed,
                  commands=[], attempts=0, actualGoProcesses=0, automaticRetries=0, governanceClaim=False)
    def run(label, argv, mutation=None):
        reservations = [json.loads(p.read_text()) for p in OUT.glob(f'r071_runtime_{args.role}_*/reservation-*.json')]
        assert len(reservations) < 8
        assert sum(x['mode'] == args.mode for x in reservations) < (2 if args.mode == 'compile' else 6)
        assert packet['attempts'] < (1 if args.mode == 'compile' else 4)
        before = snapshot()
        save(label + '-manifest-before.json', before)
        save('reservation-' + label + '.json', dict(role=args.role, mode=args.mode, sourceCommit=source, command=argv, ordinal=len(reservations) + 1))
        row = dict(label=label, command=argv, processStarted=False, timedOut=False, exitCode=None)
        packet['commands'].append(row)
        packet['attempts'] += 1
        started = time.monotonic()
        with (packet_dir / (label + '-stdout.bin')).open('xb') as stdout, (packet_dir / (label + '-stderr.bin')).open('xb') as stderr:
            try:
                process = subprocess.Popen(argv, cwd=ROOT, env=env, stdout=stdout, stderr=stderr,
                                           creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
                row.update(processStarted=True, ownedPid=process.pid)
                packet['actualGoProcesses'] += 1
                try:
                    row['exitCode'] = process.wait(timeout=240)
                except subprocess.TimeoutExpired:
                    row['timedOut'] = True
                    cleanup = subprocess.run(['taskkill', '/PID', str(process.pid), '/T', '/F'], capture_output=True, timeout=15)
                    row['cleanup'] = dict(ownedRootPid=process.pid, exitCode=cleanup.returncode)
                    process.wait(timeout=15)
                    row['exitCode'] = process.returncode
            except Exception as error:
                row['errorType'] = type(error).__name__
        row['elapsedSeconds'] = time.monotonic() - started
        after = snapshot()
        save(label + '-manifest-after.json', after)
        assert before == after, 'Native command changed source inventory'
        assert row['processStarted'] and not row['timedOut'] and 'errorType' not in row
        if args.mode == 'compile':
            assert row['exitCode'] == 0, 'Whole backend compiler failed'
            return
        events = [json.loads(line) for line in (packet_dir / (label + '-stdout.bin')).read_text(encoding='utf-8').splitlines() if line.strip()]
        passed = [e['Test'] for e in events if e.get('Action') == 'pass' and e.get('Test')]
        failed = [e['Test'] for e in events if e.get('Action') == 'fail' and e.get('Test')]
        skipped = [e['Test'] for e in events if e.get('Action') == 'skip' and e.get('Test')]
        row.update(passed=passed, failed=failed, skipped=skipped)
        outputs = ''.join(e.get('Output', '') for e in events)
        assert not skipped and not re.search(r'(^|\n)(panic:|fatal error:)', outputs)
        if mutation:
            detector = mutation['detector']
            assert row['exitCode'] != 0 and detector in failed
            assert all(n == detector or n.startswith(detector + '/') for n in failed)
            assert mutation['healthy'] in passed
            assert any(e.get('Test', '').startswith(detector) and mutation['diagnosticContains'] in e.get('Output', '') for e in events)
        else:
            required = set(legacy + new if label == 'positive' else new)
            top = {n for n in passed if '/' not in n}
            assert row['exitCode'] == 0 and not failed and top == required
            assert any(e.get('Action') == 'pass' and not e.get('Test') and e.get('Package', '').endswith('/engine') for e in events)
    failure = None
    try:
        go_base = [exe, '-C', str(scratch / 'backend')]
        if args.mode == 'compile':
            run('compile', go_base + ['build', './...'])
        else:
            base = go_base + ['test', '-json', '-count=1', '-timeout=120s', './engine']
            run('positive', base + ['-run', '^(' + '|'.join(legacy + new) + ')$'])
            for mutation in mutations:
                path = scratch / mutation['path']
                original = members[mutation['path']]
                changed = original.decode().replace(mutation['old'], mutation['new']).encode()
                path.write_bytes(changed)
                actual = snapshot()
                assert set(actual) == set(baseline)
                assert [n for n in baseline if baseline[n] != actual[n]] == [mutation['path']]
                save(mutation['label'] + '-source-diff.json', dict(path=mutation['path'], before=digest(original), after=digest(changed), replacementMatches=1,
                     diff=''.join(difflib.unified_diff(original.decode().splitlines(True), changed.decode().splitlines(True)))))
                try:
                    run(mutation['label'], base + ['-run', '^(' + mutation['detector'] + '|' + mutation['healthy'] + ')$'], mutation)
                finally:
                    path.write_bytes(original)
                assert snapshot() == baseline
            run('restored', base + ['-run', '^(' + '|'.join(new) + ')$'])
        packet['status'] = 'PASS'
    except BaseException as error:
        failure = type(error).__name__ + ': ' + str(error)
        packet.update(status='FAIL_STOPPED_NO_RETRY', failure=failure)
    finally:
        restoration_errors = []
        for name, data in members.items():
            try:
                (scratch / name).write_bytes(data)
            except Exception as error:
                restoration_errors.append(dict(path=name, errorType=type(error).__name__))
        packet['restorationErrors'] = restoration_errors
        try:
            packet['sourceRestored'] = snapshot() == baseline and not restoration_errors
        except Exception as error:
            packet['sourceRestored'] = False
            restoration_errors.append(dict(operation='snapshot', errorType=type(error).__name__))
        second = None
        try:
            second = git('archive', '--format=tar', source, 'backend')
            (scratch / 'source-after.tar').write_bytes(second)
        except Exception as error:
            restoration_errors.append(dict(operation='second-archive', errorType=type(error).__name__))
        packet['secondArchiveIdentical'] = second == archive
        if not packet['sourceRestored'] or not packet['secondArchiveIdentical']:
            failure = failure or 'Source restoration/archive mismatch'
            packet.update(status='FAIL_STOPPED_NO_RETRY', failure=failure)
        packet['raw'] = {p.name: dict(bytes=p.stat().st_size, sha256=digest(p.read_bytes())) for p in packet_dir.iterdir() if p.is_file()}
        backup = Path(tempfile.mkdtemp(prefix=f'ccmai-r071-{args.role}-raw-backup-'))
        packet['rawBackup'] = str(backup)
        save('summary.json', packet)
        inventory = {}
        for path in packet_dir.iterdir():
            if path.is_file():
                shutil.copyfile(path, backup / path.name)
                assert (backup / path.name).read_bytes() == path.read_bytes()
                inventory[path.name] = digest(path.read_bytes())
        for name in ['source-before.tar', 'source-after.tar']:
            if (scratch / name).is_file():
                shutil.copyfile(scratch / name, backup / name)
                assert (backup / name).read_bytes() == (scratch / name).read_bytes()
                inventory[name] = digest((scratch / name).read_bytes())
        (backup / 'backup-verification.json').write_text(json.dumps(inventory, indent=2) + '\n')
        print(json.dumps(dict(summary=(packet_dir / 'summary.json').relative_to(ROOT).as_posix(), status=packet['status'], attempts=packet['attempts'], actualGoProcesses=packet['actualGoProcesses'], backup=str(backup))))
    return 1 if failure else 0

if __name__ == '__main__':
    raise SystemExit(main())
