"""One supplemental visual only after an explicit NEW owner grant; no tests/build."""
import argparse
import hashlib
import io
import json
import os
import re
import shutil
import subprocess
import tarfile
import tempfile
import time
from types import SimpleNamespace
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument('--grant', required=True)
args = parser.parse_args()
grant = json.loads(Path(args.grant).read_text(encoding='utf-8-sig'))
assert grant['status'] == 'OWNER_APPROVED_EXACTLY_ONE_VISUAL'
assert grant['request'] == 'docs/reviews/probes/r067_corrected_keyboard_visual_request_2026-10-09.json'
source = 'bd62c7e4e4868cc05528aa2b8a1ae05561a78df1'
archive = subprocess.check_output(['git', 'archive', source, 'frontend'])
assert hashlib.sha256(archive).hexdigest() == '60d07e3481a03d0d9ce7b6d32cc961a60eedd623e99039eddeeb3875f20a94c6'
with tarfile.open(fileobj=io.BytesIO(archive)) as members:
    files = {m.name: members.extractfile(m).read() for m in members if m.isfile()}
assert len(files) == 121
for path, data in files.items():
    assert Path(path).read_bytes().replace(b'\r\n', b'\n') == data.replace(b'\r\n', b'\n'), path
out = Path('docs/reviews/probes')
prefix = 'r067_native_keyboard_capture'
assert not any(out.glob(prefix + '*')), 'Supplemental evidence exists; never rerun or overwrite'
command = ['node', str(out / 'r067_visual_fixture_native_keyboard_PROPOSED_2026-10-09.mjs')]
started = time.monotonic()
process = subprocess.Popen(command, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
    creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
timeout_cleanup = None
try:
    stdout, stderr = process.communicate(timeout=180)
    result = SimpleNamespace(returncode=process.returncode, stdout=stdout, stderr=stderr)
except subprocess.TimeoutExpired as expired:
    # Only this owned process and its currently observed descendants are candidates.
    script = r'''$ErrorActionPreference='Stop'; $pendingIds = [System.Collections.Generic.Queue[int]]::new(); $pendingIds.Enqueue(ROOTPID); $rows = @(); while ($pendingIds.Count -gt 0) { $targetId = $pendingIds.Dequeue(); $item = Get-CimInstance Win32_Process -Filter "ProcessId=$targetId"; if ($null -ne $item) { $rows += [pscustomobject]@{pid=[int]$item.ProcessId; parent=[int]$item.ParentProcessId; created=$item.CreationDate.ToUniversalTime().ToString('o')}; foreach ($child in @(Get-CimInstance Win32_Process -Filter "ParentProcessId=$targetId")) { $pendingIds.Enqueue([int]$child.ProcessId) } } }; ConvertTo-Json -InputObject @($rows) -Compress'''.replace('ROOTPID', str(process.pid))
    stdout, stderr = expired.stdout or b'', expired.stderr or b''
    try:
        snapshot = subprocess.run(['powershell', '-NoProfile', '-NonInteractive', '-Command', script], capture_output=True, timeout=15)
        owned = json.loads(snapshot.stdout.decode('utf-8-sig')) if snapshot.returncode == 0 else None
        killed = None
        if process.poll() is None:
            killed = subprocess.run(['taskkill', '/PID', str(process.pid), '/T', '/F'], capture_output=True, timeout=15)
        try:
            stdout, stderr = process.communicate(timeout=10)
        except subprocess.TimeoutExpired as tail:
            stdout, stderr = tail.stdout or expired.stdout or b'', tail.stderr or expired.stderr or b''
        remaining = None
        if owned is not None:
            remaining = []
            for row in owned:
                check_script = '$ErrorActionPreference="Stop"; $item=Get-CimInstance Win32_Process -Filter "ProcessId=' + str(row['pid']) + '"; if ($null -ne $item) { ConvertTo-Json -InputObject @{pid=[int]$item.ProcessId; created=$item.CreationDate.ToUniversalTime().ToString("o")} -Compress }'
                check = subprocess.run(['powershell', '-NoProfile', '-NonInteractive', '-Command', check_script], capture_output=True, timeout=10)
                if check.returncode != 0:
                    remaining.append({'pid': row['pid'], 'state': 'UNKNOWN_READBACK_FAILED'})
                elif check.stdout.strip():
                    current = json.loads(check.stdout.decode('utf-8-sig'))
                    if current['created'] == row['created']:
                        remaining.append(current)
        timeout_cleanup = dict(ownedRootPid=process.pid, observedTreeBefore=owned,
            remainingSameIdentities=remaining, verifiedObservedTreeAbsent=bool(owned) and remaining == [] if remaining is not None else False,
            taskkillExit=killed.returncode if killed else None,
            scope='Only spawned fixture and its observed descendants; unknown readback is not absent')
    except Exception as cleanup_error:
        timeout_cleanup = dict(ownedRootPid=process.pid, verifiedObservedTreeAbsent=False,
            error=type(cleanup_error).__name__, scope='Cleanup/readback incomplete; never claim absent')
        try:
            if process.poll() is None:
                subprocess.run(['taskkill', '/PID', str(process.pid), '/T', '/F'], capture_output=True, timeout=15)
            stdout, stderr = process.communicate(timeout=10)
        except Exception:
            stdout, stderr = expired.stdout or b'', expired.stderr or b''
    result = SimpleNamespace(returncode=124, stdout=stdout, stderr=stderr)
elapsed = time.monotonic() - started
artifacts = []

def exclusive(path, data):
    with path.open('xb') as handle:
        handle.write(data)
    artifacts.append({'path': path.as_posix(), 'bytes': len(data), 'sha256Physical': hashlib.sha256(data).hexdigest()})

exclusive(out / (prefix + '_stdout.log'), result.stdout)
exclusive(out / (prefix + '_stderr.log'), result.stderr)
temp_root = None
status = 'FAILED'
marker = re.search(r'^R067_VISUAL_TEMP=([^\r\n]+)', result.stdout.decode(errors='replace'), re.MULTILINE)
if marker:
    temp_root = Path(marker.group(1))
if result.returncode == 0:
    packet = json.loads(result.stdout.decode().strip().splitlines()[-1])
    assert packet['status'] == 'PASS'
    temp_root = Path(packet['logs'])
    status = 'PASS_CAPTURE_PENDING_INDEPENDENT_IMAGE_REVIEW'
else:
    match = re.search(r'(?:raw output retained in|report retained in) ([^\r\n]+)', result.stderr.decode(errors='replace'))
    if match:
        temp_root = Path(match.group(1))
if temp_root is not None:
    for name in ['vite.log', 'capture.log', 'screenshots/report.json']:
        item = temp_root / name
        if item.is_file():
            exclusive(out / (prefix + '_' + name.replace('/', '_')), item.read_bytes())
    for item in sorted((temp_root / 'screenshots').glob('*.png')):
        exclusive(out / (prefix + '_' + item.name), item.read_bytes())
    junction_path = temp_root / 'fixture/node_modules'
    try:
        junction_path.lstat()
        junction_absent = False
    except FileNotFoundError:
        junction_absent = not os.path.lexists(junction_path)
    except OSError:
        junction_absent = None
else:
    junction_absent = None
if result.returncode == 0 and junction_absent is not True:
    status = 'FAILED_CLEANUP_NOT_VERIFIED'
source_unchanged = all(Path(path).read_bytes().replace(b'\r\n', b'\n') == data.replace(b'\r\n', b'\n') for path, data in files.items())
if not source_unchanged: status = 'FAILED_SOURCE_CHANGED'
receipt = dict(date='2026-10-09', status=status, grant=args.grant, sourceCommit=source,
    command=command, exitCode=result.returncode, elapsedSeconds=elapsed, visualInvocations=1,
    automaticRetries=0, newVitestInvocations=0, newBuildInvocations=0, newGoInvocations=0,
    tempRoot=str(temp_root) if temp_root else None, fixtureDependencyJunctionAbsent=junction_absent,
    sourceUnchanged=source_unchanged, artifacts=artifacts, timeoutCleanup=timeout_cleanup,
    scope='Synthetic local component UI only; no provider/AI governance claim')
exclusive(out / (prefix + '_receipt.json'), (json.dumps(receipt, indent=2) + '\n').encode())
backup = Path(tempfile.mkdtemp(prefix='r067-supplemental-visual-raw-backup-'))
for artifact in artifacts:
    path = Path(artifact['path'])
    shutil.copyfile(path, backup / path.name)
    assert hashlib.sha256((backup / path.name).read_bytes()).hexdigest() == artifact['sha256Physical']
print(json.dumps({'status': status, 'receipt': str(out / (prefix + '_receipt.json')), 'rawBackup': str(backup)}))
raise SystemExit(result.returncode or (0 if source_unchanged and junction_absent is True else 1))
