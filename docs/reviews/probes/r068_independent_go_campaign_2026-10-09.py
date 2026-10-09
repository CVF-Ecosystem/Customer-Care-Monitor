"""R068 root independent exact-source offline compile or four-test campaign.

Source and semantic plan must be committed before execution. No retries, live
provider, database or runtime-governance assertion. Scratch/raw backup retained.
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
    parser.add_argument('--source', required=True)
    parser.add_argument('--plan', required=True)
    parser.add_argument('--mode', choices=['compile', 'tests', 'inspect'], required=True)
    args = parser.parse_args()
    assert re.fullmatch('[0-9a-f]{40}', args.source)
    plan_path = Path(args.plan).resolve()
    assert plan_path.is_relative_to(ROOT)
    plan = json.loads(plan_path.read_text(encoding='utf-8-sig'))
    assert plan['sourceCommit'] == args.source
    archive = git('archive', '--format=tar', args.source, 'backend')
    assert digest(archive) == plan['backendArchiveSha256']
    with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
        members = {m.name: tar.extractfile(m).read() for m in tar if m.isfile()}
    assert len(members) == plan['backendArchiveMembers']
    assert all(p.startswith('backend/') and '..' not in Path(p).parts for p in members)
    mutations = plan['mutations']
    assert len(mutations) == 2
    assert all(m['path'] == 'backend/ai/usage_presence.go' for m in mutations)
    newtests = plan['newTestNames']
    assert newtests and all(re.fullmatch(r'Test\w+', n) for n in newtests)
    for mutation in mutations:
        assert members[mutation['path']].decode().count(mutation['old']) == 1
        assert mutation['old'] != mutation['new']
        assert mutation['detector'] in newtests and mutation['healthy'] in newtests
    if args.mode == 'inspect':
        print(json.dumps(dict(source=args.source, members=len(members), archive=digest(archive), mutations=mutations)))
        return
    # Validate the committed plan bytes, not an editable working-only authority.
    assert git('show', 'HEAD:' + plan_path.relative_to(ROOT).as_posix()).replace(b'\r\n', b'\n') == plan_path.read_bytes().replace(b'\r\n', b'\n')
    prefix = 'r068_root_' + args.mode + '_actual'
    assert not any(OUT.glob(prefix + '*')), 'Existing raw packet: never retry/overwrite'
    scratch = Path(tempfile.mkdtemp(prefix='ccmai-r068-root-' + args.mode + '-'))
    for relative, data in members.items():
        path = scratch / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data)

    def snapshot():
        return {p.relative_to(scratch).as_posix(): dict(bytes=p.stat().st_size, sha256Physical=digest(p.read_bytes()))
                for p in sorted((scratch / 'backend').rglob('*')) if p.is_file()}

    baseline = snapshot()
    packet = dict(date='2026-10-09', sourceCommit=args.source, independentReviewer='Codex /root',
                  mode=args.mode, archiveSha256=digest(archive), members=len(members),
                  plan=args.plan, planSha256Physical=digest(plan_path.read_bytes()), scratch=str(scratch),
                  commands=[], raw=[], GoInvocations=0, automaticRetries=0, governanceClaim=False)
    env = os.environ.copy()
    env.update(GOPROXY='off', GOSUMDB='off', GOTOOLCHAIN='local', CGO_ENABLED='0')

    def write_raw(label, data):
        path = OUT / (prefix + '_' + label)
        with path.open('xb') as handle:
            handle.write(data)
        item = dict(path=path.relative_to(ROOT).as_posix(), bytes=len(data), sha256Physical=digest(data))
        packet['raw'].append(item)
        return item

    def write_json(label, data):
        return write_raw(label, (json.dumps(data, indent=2) + '\n').encode())

    def run(label, command, mutation=None):
        assert packet['GoInvocations'] < (1 if args.mode == 'compile' else 4)
        before = snapshot()
        write_json(label + '_manifest_before.json', before)
        stdout_path = OUT / (prefix + '_' + label + '_stdout.jsonl')
        stderr_path = OUT / (prefix + '_' + label + '_stderr.log')
        started = time.monotonic()
        packet['GoInvocations'] += 1
        timed_out, exit_code, launch_error = False, None, None
        with stdout_path.open('xb') as stdout, stderr_path.open('xb') as stderr:
            try:
                process = subprocess.Popen(command, cwd=ROOT, env=env, stdout=stdout, stderr=stderr,
                                           creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
                try:
                    exit_code = process.wait(timeout=240)
                except subprocess.TimeoutExpired:
                    timed_out = True
                    packet['timeoutCleanup'] = dict(ownedRootPid=process.pid,
                                                   scope='Owned Go root PID/tree only; no universal process absence claim')
                    try:
                        cleanup = subprocess.run(['taskkill', '/PID', str(process.pid), '/T', '/F'], capture_output=True, timeout=15)
                        packet['timeoutCleanup']['taskkillExit'] = cleanup.returncode
                        process.wait(timeout=15)
                    except BaseException as cleanup_error:
                        packet['timeoutCleanup']['error'] = type(cleanup_error).__name__
                    packet['timeoutCleanup']['ownedParentExited'] = process.poll() is not None
                    exit_code = 124
            except BaseException as error:
                launch_error = type(error).__name__
        raw_items = []
        for path in [stdout_path, stderr_path]:
            data = path.read_bytes()
            item = dict(path=path.relative_to(ROOT).as_posix(), bytes=len(data), sha256Physical=digest(data))
            packet['raw'].append(item)
            raw_items.append(item)
        row = dict(label=label, command=command, exitCode=exit_code, elapsedSeconds=time.monotonic()-started,
                   raw=raw_items, timedOut=timed_out, launchError=launch_error)
        packet['commands'].append(row)
        after = snapshot()
        write_json(label + '_manifest_after.json', after)
        assert before == after, 'Go changed source/scratch inventory'
        assert not timed_out and launch_error is None, 'Timeout/launch error is not a semantic kill'
        if args.mode == 'compile':
            assert exit_code == 0, 'Wholebackend compile failed'
            return
        events = [json.loads(line) for line in stdout_path.read_text(encoding='utf-8').splitlines() if line.strip()]
        passed = [x['Test'] for x in events if x.get('Action') == 'pass' and x.get('Test')]
        failed = [x['Test'] for x in events if x.get('Action') == 'fail' and x.get('Test')]
        skipped = [x['Test'] for x in events if x.get('Action') == 'skip' and x.get('Test')]
        row.update(passed=passed, failed=failed, skipped=skipped)
        assert not skipped, 'Required ai test skipped'
        if mutation:
            assert exit_code != 0 and failed == [mutation['detector']], 'Unrelated/build failure is not a semantic kill'
            assert mutation['healthy'] in passed, 'Healthy control failed/missing'
            row['semanticDetector'] = 'FAIL_EXPECTED'
            row['healthyControl'] = 'PASS'
        else:
            assert exit_code == 0 and passed and not failed
            assert set(newtests) <= set(passed), 'NEW inventory missing'
            if label == 'positive':
                assert any(x.get('Action') == 'pass' and not x.get('Test') and x.get('Package', '').endswith('/ai') for x in events)

    failure = None
    try:
        go_base = ['go', '-C', str(scratch / 'backend')]
        if args.mode == 'compile':
            run('compile', go_base + ['build', './...'])
        else:
            test_command = go_base + ['test', '-json', '-count=1', '-timeout=120s', './ai']
            run('positive', test_command)
            for mutation in mutations:
                original = members[mutation['path']]
                target = scratch / mutation['path']
                changed = original.decode().replace(mutation['old'], mutation['new']).encode()
                target.write_bytes(changed)
                actual = snapshot()
                assert [p for p in baseline if baseline[p] != actual[p]] == [mutation['path']]
                write_json(mutation['label'] + '_diff.json', dict(path=mutation['path'], before=digest(original),
                           after=digest(changed), replacementMatches=1,
                           diff=''.join(difflib.unified_diff(original.decode().splitlines(True), changed.decode().splitlines(True)))))
                try:
                    pattern = '^(' + re.escape(mutation['detector']) + '|' + re.escape(mutation['healthy']) + ')$'
                    run(mutation['label'], test_command + ['-run', pattern], mutation)
                finally:
                    target.write_bytes(original)
                assert snapshot() == baseline, 'Wholebackend restoration failed'
            run('restored', test_command + ['-run', '^(' + '|'.join(newtests) + ')$'])
        assert snapshot() == baseline
        assert git('archive', '--format=tar', args.source, 'backend') == archive
        packet['secondArchiveIdentical'] = True
        packet['status'] = 'PASS'
    except BaseException as error:
        failure = str(error)
        packet.update(status='FAIL_STOPPED_NO_RETRY', failure=failure)
    finally:
        restoration_errors = []
        for relative, original in members.items():
            try:
                (scratch / relative).write_bytes(original)
            except BaseException as restore_error:
                restoration_errors.append(dict(path=relative, error=type(restore_error).__name__))
        packet['restorationErrors'] = restoration_errors
        try:
            packet['finalFullSourceRestored'] = not restoration_errors and snapshot() == baseline
        except BaseException as restore_error:
            packet['finalFullSourceRestored'] = False
            packet['restorationError'] = type(restore_error).__name__
        if not packet['finalFullSourceRestored']:
            failure = failure or 'Scratch extras/changes retained; no restoration claim'
            packet.update(status='FAIL_STOPPED_NO_RETRY', failure=failure)
        backup = Path(tempfile.mkdtemp(prefix='r068-root-' + args.mode + '-raw-backup-'))
        packet['rawBackup'] = str(backup)
        summary_path = OUT / (prefix + '_summary.json')
        with summary_path.open('xb') as handle:
            handle.write((json.dumps(packet, indent=2) + '\n').encode())
        for item in packet['raw'] + [dict(path=summary_path.relative_to(ROOT).as_posix(), sha256Physical=digest(summary_path.read_bytes()))]:
            path = ROOT / item['path']
            shutil.copyfile(path, backup / path.name)
            assert digest((backup / path.name).read_bytes()) == item['sha256Physical']
        print(json.dumps(dict(summary=summary_path.relative_to(ROOT).as_posix(), status=packet['status'],
                              GoInvocations=packet['GoInvocations'], backup=str(backup))))
    if failure:
        raise SystemExit(1)


if __name__ == '__main__':
    main()
