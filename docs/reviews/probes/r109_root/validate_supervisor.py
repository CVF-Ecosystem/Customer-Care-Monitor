"""Offline harness regressions and actual config import; no product runtime."""
from pathlib import Path
import ast
import hashlib
import importlib.util
import io
import json
import os
import queue
import subprocess
import time
import unittest
from unittest.mock import patch

ROOT = Path.cwd().resolve()
OUT = ROOT / 'docs/reviews/probes/r109_root/harness_validation'
OUT.mkdir(exist_ok=False)
SOURCE = ROOT / 'docs/reviews/probes/r109_luna/r109_workflow_runner.py'
TEMPLATE = ROOT / 'docs/reviews/probes/r109_luna/r109_vite_config.mjs.template'
pins = {str(p.relative_to(ROOT)): hashlib.sha256(p.read_bytes()).hexdigest() for p in (SOURCE, TEMPLATE)}
ast.parse(SOURCE.read_text(encoding='utf-8'))
spec = importlib.util.spec_from_file_location('reviewed_r109', SOURCE)
assert spec and spec.loader
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)
helper = m.load_r108_helpers(ROOT / m.R108_HELPER_PATH)

class Connection:
    def __init__(self, packet): self.packet = io.BytesIO(packet)
    def __enter__(self): return self
    def __exit__(self, *_args): pass
    def settimeout(self, _timeout): pass
    def recv(self, size): return self.packet.read(min(size, 3))

def packet(payload): return len(payload).to_bytes(3, 'little') + b'\x00' + payload

class HarnessRegression(unittest.TestCase):
    def campaign(self, name):
        c = m.Campaign.__new__(m.Campaign)
        c.directory = OUT / name
        c.directory.mkdir()
        c.effect_deadline = time.monotonic() + 8
        c.deadline = c.effect_deadline + 5
        return c

    def test_early_eof_is_pending_then_fragmented_complete_greeting_is_ready(self):
        greeting = packet(b'\x0a8.0.44\x00' + b'\x01\x00\x00\x00abcdefgh\x00\xff\xf7' + b'\x00' * 24)
        c = self.campaign('eof_then_ready')
        with patch.object(m.socket, 'create_connection', side_effect=[Connection(b''), Connection(greeting)]) as connect, patch.object(m.time, 'sleep'):
            self.assertEqual(c.wait_mysql_handshake(3306), 2)
            self.assertEqual(connect.call_count, 2)
        first = json.loads((c.directory / 'readiness/mysql_probe_0001.json').read_text())
        self.assertEqual(first['state'], 'NOT_READY')

    def test_complete_mysql_error_stops_once_and_records_only_safe_number(self):
        c = self.campaign('fatal_server_error')
        error = packet(b'\xff' + (1130).to_bytes(2, 'little') + b'#HY000private-host-detail')
        with patch.object(m.socket, 'create_connection', return_value=Connection(error)) as connect:
            with self.assertRaises(m.Hold): c.wait_mysql_handshake(3306)
            self.assertEqual(connect.call_count, 1)
        raw = (c.directory / 'readiness/mysql_probe_0001.json').read_text()
        self.assertEqual(json.loads(raw)['safeServerErrorNumber'], 1130)
        self.assertNotIn('private-host-detail', raw)

    def test_pipe_readiness_arrives_before_eof_and_split_password_is_redacted(self):
        read_fd, write_fd = os.pipe()
        stream = os.fdopen(read_fd, 'rb')
        events = queue.Queue()
        path = OUT / 'offline_pipe.bin'
        reader = m.PipeCapture(stream, path, helper, events)
        reader.start()
        try:
            os.write(write_fd, b'OFFLINE_HARNESS_READY\n')
            self.assertEqual(events.get(timeout=2), 'OFFLINE_HARNESS_READY')
            password = m.SYNTHETIC_BROWSER_PASSWORD.encode()
            os.write(write_fd, password[:7])
            os.write(write_fd, password[7:] + b'\n')
        finally:
            os.close(write_fd)
            reader.thread.join(timeout=2)
            stream.close()
        self.assertFalse(reader.thread.is_alive())
        self.assertIsNone(reader.error)
        self.assertNotIn(password, path.read_bytes())
        self.assertIn(b'<synthetic-browser-password-redacted>', path.read_bytes())

output = io.StringIO()
result = unittest.TextTestRunner(stream=output, verbosity=2).run(unittest.defaultTestLoader.loadTestsFromTestCase(HarnessRegression))
(OUT / 'regressions.txt').write_text(output.getvalue(), encoding='utf-8')
print(output.getvalue(), flush=True)
config = OUT / 'vite.config.mjs'
config.write_text(m.render_vite_config(TEMPLATE.read_text(encoding='utf-8'), ROOT / 'frontend', OUT,
                  'http://127.0.0.1:52962', 'a' * 32), encoding='utf-8', newline='\n')
node = Path(m.NODE_EXE)
assert hashlib.sha256(node.read_bytes()).hexdigest() == m.NODE_SHA256
command = [str(node), str(ROOT / 'docs/reviews/probes/r109_root/assert_config.mjs'), str(config), str(ROOT / 'frontend')]
executed = subprocess.run(command, cwd=ROOT, env=m.minimal_env(), capture_output=True, timeout=120, check=False)
(OUT / 'config_import.stdout.txt').write_bytes(executed.stdout)
(OUT / 'config_import.stderr.txt').write_bytes(executed.stderr)
receipt = {'regressionTests': result.testsRun, 'regressionsPassed': result.wasSuccessful(),
           'sourcePins': pins, 'configImportCommand': command, 'configImportExitCode': executed.returncode,
           'nodeSha256': m.NODE_SHA256, 'scope': 'OFFLINE_HARNESS_AND_MODULE_IMPORT_ONLY; product runtime NOT_RUN'}
(OUT / 'result.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
print(json.dumps(receipt), flush=True)
assert pins == {str(p.relative_to(ROOT)): hashlib.sha256(p.read_bytes()).hexdigest() for p in (SOURCE, TEMPLATE)}
raise SystemExit(0 if result.wasSuccessful() and executed.returncode == 0 else 1)
