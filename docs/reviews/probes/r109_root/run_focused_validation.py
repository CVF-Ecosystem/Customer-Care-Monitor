"""One append-only focused UI validation pair; no fixture or app server."""
from pathlib import Path
import datetime
import hashlib
import json
import os
import subprocess

ROOT = Path.cwd().resolve()
OUT = ROOT / 'docs/reviews/probes/r109_root/validation_1'
NODE = Path('C:/Program Files/nodejs/node.exe')
PIN = '3602f2bb1a10f2cbab4c36886218a33c1ab3db87290e73b033c46c77147d0237'
assert hashlib.sha256(NODE.read_bytes()).hexdigest() == PIN, 'Node pin drift'
OUT.mkdir(exist_ok=False)
ENV = {k: os.environ[k] for k in ('SystemRoot', 'WINDIR', 'TEMP', 'TMP') if k in os.environ}
sources = [
    'frontend/src/views/InterventionCases.vue',
    'frontend/src/utils/interventionPresentation.ts',
    'frontend/src/i18n/vi.ts', 'frontend/src/i18n/en.ts',
    'frontend/src/__tests__/intervention-ui.test.ts',
    'frontend/src/__tests__/intervention-presentation.test.ts',
    'frontend/src/__tests__/intervention-api.test.ts',
    'docs/reviews/probes/r109_root/focused-test.config.mjs',
]
pins = {p: hashlib.sha256((ROOT / p).read_bytes()).hexdigest() for p in sources}
(OUT / 'source_pins.json').write_text(json.dumps(pins, indent=2) + '\n', encoding='utf-8')
commands = [
    ('focused_tests', [str(NODE), str(ROOT / 'frontend/node_modules/vitest/vitest.mjs'),
       'run', '--config', str(ROOT / 'docs/reviews/probes/r109_root/focused-test.config.mjs'),
       'src/__tests__/intervention-api.test.ts', 'src/__tests__/intervention-ui.test.ts',
       'src/__tests__/intervention-presentation.test.ts']),
    ('forced_typecheck', [str(NODE), str(ROOT / 'frontend/node_modules/vue-tsc/bin/vue-tsc.js'),
       '-b', str(ROOT / 'frontend/tsconfig.json'), '--force']),
]
receipts = []
for label, command in commands:
    started = datetime.datetime.now(datetime.timezone.utc).isoformat()
    result = subprocess.run(command, cwd=ROOT, env=ENV, capture_output=True, timeout=150, check=False)
    (OUT / (label + '.stdout.txt')).write_bytes(result.stdout)
    (OUT / (label + '.stderr.txt')).write_bytes(result.stderr)
    receipt = {'command': command, 'exitCode': result.returncode, 'startedAtUTC': started,
               'nodeSha256': PIN, 'environmentKeys': sorted(ENV),
               'scope': 'focused cached source validation only; runtime NOT_RUN'}
    (OUT / (label + '.json')).write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
    receipts.append(receipt)
    print(label, 'exit', result.returncode, flush=True)
    print(result.stdout.decode('utf-8', errors='replace')[-5000:], flush=True)
    print(result.stderr.decode('utf-8', errors='replace')[-1500:], flush=True)
assert pins == {p: hashlib.sha256((ROOT / p).read_bytes()).hexdigest() for p in sources}, 'source changed during validation'
raise SystemExit(0 if all(r['exitCode'] == 0 for r in receipts) else 1)
