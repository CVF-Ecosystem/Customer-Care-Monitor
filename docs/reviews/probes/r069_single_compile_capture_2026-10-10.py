"""Single-use R069 cached offline compile; no tests, retries or source edits."""
import hashlib
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import time

ROOT = Path(__file__).resolve().parents[3]
PLAN = 'docs/reviews/probes/r069_exact_source_plan_2026-10-10.json'
OUT = ROOT / 'docs/reviews/probes/r069_compile_actual'

def sha(data):
    return hashlib.sha256(data).hexdigest()

def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT)

def main():
    plan_path = ROOT / PLAN
    assert git('show', 'HEAD:' + PLAN).replace(b'\r\n', b'\n') == plan_path.read_bytes().replace(b'\r\n', b'\n')
    script = Path(__file__).relative_to(ROOT).as_posix()
    assert git('show', 'HEAD:' + script).replace(b'\r\n', b'\n') == Path(__file__).read_bytes().replace(b'\r\n', b'\n')
    plan = json.loads(plan_path.read_text(encoding='utf-8-sig'))
    record = json.loads((ROOT / 'CVF_SESSION/tranches/CCMAI-RUNTIME-069.json').read_text())
    assert record['status'] == 'BUILD' and record['executionBudget']['usedGoInvocations'] == 0
    assert git('show', 'HEAD:CVF_SESSION/tranches/CCMAI-RUNTIME-069.json').replace(b'\r\n', b'\n') == (ROOT / 'CVF_SESSION/tranches/CCMAI-RUNTIME-069.json').read_bytes().replace(b'\r\n', b'\n')
    archive = git('archive', '--format=tar', plan['sourceCommit'], 'backend')
    assert sha(archive) == plan['backendArchiveSha256'] and len(archive) == plan['backendArchiveBytes']
    with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
        members = {m.name: tar.extractfile(m).read() for m in tar if m.isfile()}
    assert len(members) == plan['backendArchiveMembers']
    assert all(p.startswith('backend/') and '..' not in Path(p).parts for p in members)
    OUT.mkdir()  # Exclusive single-use reservation, before any Go launch.
    scratch = Path(tempfile.mkdtemp(prefix='ccmai-r069-compile-'))
    for relative, data in members.items():
        path = scratch / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data)
    def snapshot():
        return {p.relative_to(scratch).as_posix(): {'bytes': p.stat().st_size, 'sha256': sha(p.read_bytes())}
                for p in sorted((scratch / 'backend').rglob('*')) if p.is_file()}
    def save(name, value):
        (OUT / name).write_text(json.dumps(value, indent=2) + '\n', encoding='utf-8')
    before = snapshot()
    save('manifest-before.json', before)
    (scratch / 'source-before.tar').write_bytes(archive)
    exe = shutil.which('go')
    assert exe and Path(exe).is_file()
    argv = [exe, '-C', str(scratch / 'backend'), 'build', './...']
    fixed_env = dict(GOPROXY='off', GOSUMDB='off', GOTOOLCHAIN='local', CGO_ENABLED='0')
    env = os.environ.copy()
    env.update(fixed_env)
    result = dict(date='2026-10-10', sourceCommit=plan['sourceCommit'], validationTranche='CCMAI-RUNTIME-069',
                  approvalCommit=git('rev-parse', 'HEAD').decode().strip(), executor='Codex /root',
                  command=argv, fixedEnvironment=fixed_env, scratch=str(scratch),
                  archiveSha256=sha(archive), members=len(members), attempts=1, actualGoProcesses=0,
                  automaticRetries=0, tests=0, processStarted=False, timedOut=False, exitCode=None,
                  governanceClaim=False, projectSourceEdited=False)
    save('launch-reservation.json', result)
    started = time.monotonic()
    with (OUT / 'stdout.bin').open('xb') as stdout, (OUT / 'stderr.bin').open('xb') as stderr:
        try:
            process = subprocess.Popen(argv, cwd=ROOT, env=env, stdout=stdout, stderr=stderr,
                                       creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
            result.update(processStarted=True, actualGoProcesses=1, ownedPid=process.pid)
            try:
                result['exitCode'] = process.wait(timeout=240)
            except subprocess.TimeoutExpired:
                result['timedOut'] = True
                cleanup = subprocess.run(['taskkill', '/PID', str(process.pid), '/T', '/F'], capture_output=True, timeout=15)
                result['cleanup'] = dict(ownedRootPid=process.pid, exitCode=cleanup.returncode)
                process.wait(timeout=15)
                result['exitCode'] = process.returncode
        except Exception as error:
            result['errorType'] = type(error).__name__
    result['elapsedSeconds'] = time.monotonic() - started
    after = snapshot()
    save('manifest-after.json', after)
    second = git('archive', '--format=tar', plan['sourceCommit'], 'backend')
    (scratch / 'source-after.tar').write_bytes(second)
    result['sourceRestored'] = before == after and before == {p: dict(bytes=len(b), sha256=sha(b)) for p, b in members.items()}
    result['secondArchiveMatches'] = second == archive
    result['raw'] = {p.name: dict(bytes=p.stat().st_size, sha256=sha(p.read_bytes())) for p in OUT.iterdir() if p.is_file()}
    backup = Path(tempfile.mkdtemp(prefix='ccmai-r069-raw-backup-'))
    result['backup'] = str(backup)
    result['status'] = 'PASS' if result['processStarted'] and result['exitCode'] == 0 and not result['timedOut'] and result['sourceRestored'] and result['secondArchiveMatches'] and 'errorType' not in result else 'FAIL'
    save('summary.json', result)
    inventory = {}
    for path in OUT.iterdir():
        if path.is_file():
            shutil.copyfile(path, backup / path.name)
            assert path.read_bytes() == (backup / path.name).read_bytes()
            inventory[path.name] = sha(path.read_bytes())
    for name in ['source-before.tar', 'source-after.tar']:
        shutil.copyfile(scratch / name, backup / name)
        assert (backup / name).read_bytes() == archive
        inventory[name] = sha(archive)
    (backup / 'backup-verification.json').write_text(json.dumps(inventory, indent=2) + '\n')
    print(json.dumps(result))
    return 0 if result['status'] == 'PASS' else 1

if __name__ == '__main__':
    raise SystemExit(main())
