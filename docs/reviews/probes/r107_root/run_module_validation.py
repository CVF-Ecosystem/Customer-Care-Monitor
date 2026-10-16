"""One actual config import, no Vite/backend/fixture/browser server launch."""
from pathlib import Path
import hashlib, importlib.util, json, os, subprocess, time
ROOT = Path.cwd().resolve()
HERE = ROOT / 'docs/reviews/probes/r107_root'
OUT = ROOT / '.cvf/runtime/r107_module_validation'
NODE = Path('C:/Program Files/nodejs/node.exe')
PIN = '3602f2bb1a10f2cbab4c36886218a33c1ab3db87290e73b033c46c77147d0237'
def sha(path): return hashlib.sha256(path.read_bytes()).hexdigest()
assert sha(NODE) == PIN, 'Node pin drift'
assert not OUT.exists(), 'Finite module validation already used; no replay'
OUT.mkdir(parents=True)
spec = importlib.util.spec_from_file_location('candidate', HERE / 'r107_ui_runner.py')
runner = importlib.util.module_from_spec(spec); spec.loader.exec_module(runner)
config = OUT / 'vite.config.mjs'
config.write_text(runner.render_config((HERE / 'r107_vite_config.mjs.template').read_text(), ROOT / 'frontend', 'http://127.0.0.1:52962', 'a' * 32), encoding='utf-8', newline='\n')
cmd = [str(NODE), str(HERE / 'validate_config.mjs'), str(config), str(ROOT / 'frontend')]
start = time.time()
try:
    result = subprocess.run(cmd, cwd=ROOT, env=runner.minimal_env(), capture_output=True, timeout=60)
    stdout, stderr, code = result.stdout, result.stderr, result.returncode
    failure = None
except subprocess.TimeoutExpired as exc:
    stdout, stderr, code = exc.stdout or b'', exc.stderr or b'', None
    failure = 'TimeoutExpired'
(HERE / 'module_validation.stdout.txt').write_bytes(stdout)
(HERE / 'module_validation.stderr.txt').write_bytes(stderr)
receipt = {'disposition': 'PASS' if code == 0 else 'FAIL', 'exitCode': code, 'failure': failure, 'command': cmd, 'cwd': str(ROOT), 'elapsedSeconds': time.time()-start, 'nodeSha256': sha(NODE), 'templateSha256': sha(HERE / 'r107_vite_config.mjs.template'), 'configSha256': sha(config), 'checkerSha256': sha(HERE / 'validate_config.mjs'), 'stdoutSha256': sha(HERE / 'module_validation.stdout.txt'), 'stderrSha256': sha(HERE / 'module_validation.stderr.txt'), 'runtimeCampaign': 'NOT_RUN', 'claim': 'Actual cached Node config import and assertions only; no server/UI/governance acceptance'}
(HERE / 'module_validation.json').write_text(json.dumps(receipt, indent=2)+'\n', encoding='utf-8', newline='\n')
print(json.dumps(receipt))
raise SystemExit(0 if code == 0 else 1)
