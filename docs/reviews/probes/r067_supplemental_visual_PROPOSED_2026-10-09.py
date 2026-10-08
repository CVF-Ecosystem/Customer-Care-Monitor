"""One supplemental visual only after an explicit NEW owner grant; no tests/build."""
import argparse
import hashlib
import io
import json
import re
import shutil
import subprocess
import tarfile
import tempfile
import time
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument('--grant', required=True)
args = parser.parse_args()
grant = json.loads(Path(args.grant).read_text(encoding='utf-8-sig'))
assert grant['status'] == 'OWNER_APPROVED_EXACTLY_ONE_VISUAL'
assert grant['request'] == 'docs/reviews/probes/r067_additional_visual_request_2026-10-09.json'
source = 'bd62c7e4e4868cc05528aa2b8a1ae05561a78df1'
archive = subprocess.check_output(['git', 'archive', source, 'frontend'])
assert hashlib.sha256(archive).hexdigest() == '60d07e3481a03d0d9ce7b6d32cc961a60eedd623e99039eddeeb3875f20a94c6'
with tarfile.open(fileobj=io.BytesIO(archive)) as members:
    files = {m.name: members.extractfile(m).read() for m in members if m.isfile()}
assert len(files) == 121
for path, data in files.items():
    assert Path(path).read_bytes().replace(b'\r\n', b'\n') == data.replace(b'\r\n', b'\n'), path
out = Path('docs/reviews/probes')
prefix = 'r067_supplemental_reviewer_visual'
assert not any(out.glob(prefix + '*')), 'Supplemental evidence exists; never rerun or overwrite'
command = ['node', str(out / 'r067_visual_fixture_ready_diagnostics_PROPOSED_2026-10-09.mjs')]
started = time.monotonic()
result = subprocess.run(command, capture_output=True, timeout=180)
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
if result.returncode == 0:
    packet = json.loads(result.stdout.decode().strip().splitlines()[-1])
    assert packet['status'] == 'PASS'
    temp_root = Path(packet['logs'])
    status = 'PASS_CAPTURE_PENDING_INDEPENDENT_IMAGE_REVIEW'
else:
    match = re.search(r'raw output retained in ([^\r\n]+)', result.stderr.decode(errors='replace'))
    if match:
        temp_root = Path(match.group(1))
if temp_root is not None:
    for name in ['vite.log', 'capture.log', 'screenshots/report.json']:
        item = temp_root / name
        if item.is_file():
            exclusive(out / (prefix + '_' + name.replace('/', '_')), item.read_bytes())
    for item in sorted((temp_root / 'screenshots').glob('*.png')):
        exclusive(out / (prefix + '_' + item.name), item.read_bytes())
    junction_absent = not (temp_root / 'fixture/node_modules').exists()
else:
    junction_absent = None
for path, data in files.items():
    assert Path(path).read_bytes().replace(b'\r\n', b'\n') == data.replace(b'\r\n', b'\n'), path
receipt = dict(date='2026-10-09', status=status, grant=args.grant, sourceCommit=source,
    command=command, exitCode=result.returncode, elapsedSeconds=elapsed, visualInvocations=1,
    automaticRetries=0, newVitestInvocations=0, newBuildInvocations=0, newGoInvocations=0,
    tempRoot=str(temp_root) if temp_root else None, fixtureDependencyJunctionAbsent=junction_absent,
    sourceUnchanged=True, artifacts=artifacts, scope='Synthetic local component UI only; no provider/AI governance claim')
exclusive(out / (prefix + '_receipt.json'), (json.dumps(receipt, indent=2) + '\n').encode())
backup = Path(tempfile.mkdtemp(prefix='r067-supplemental-visual-raw-backup-'))
for artifact in artifacts:
    path = Path(artifact['path'])
    shutil.copyfile(path, backup / path.name)
    assert hashlib.sha256((backup / path.name).read_bytes()).hexdigest() == artifact['sha256Physical']
print(json.dumps({'status': status, 'receipt': str(out / (prefix + '_receipt.json')), 'rawBackup': str(backup)}))
raise SystemExit(result.returncode)
