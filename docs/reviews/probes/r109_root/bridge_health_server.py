"""Loopback HTTP button page for the integrated browser bridge, no product API."""
from http.server import HTTPServer, BaseHTTPRequestHandler
from pathlib import Path
import json
import time

OUT = Path('docs/reviews/probes/r109_root/bridge_renewed')
OUT.mkdir(exist_ok=False)

class Health(BaseHTTPRequestHandler):
    def do_GET(self):
        body = ('<!doctype html><html lang="vi"><meta charset="utf-8">'
                '<title>Kiểm tra kết nối</title><h1>Kiểm tra kết nối</h1>'
                '<button onclick="document.querySelector(\'output\').textContent=\'Đã kết nối\'">'
                'Kiểm tra</button><output>Chưa kiểm tra</output></html>').encode('utf-8')
        self.send_response(200)
        self.send_header('Content-Type', 'text/html; charset=utf-8')
        self.send_header('Content-Length', str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *_args):
        pass

server = HTTPServer(('127.0.0.1', 0), Health)
server.timeout = 0.5
ready = {'origin': f'http://127.0.0.1:{server.server_port}', 'ownedHealthOnly': True}
(OUT / 'ready.json').write_text(json.dumps(ready), encoding='utf-8')
print(json.dumps(ready), flush=True)
limit = time.monotonic() + 120
try:
    while time.monotonic() < limit and not (OUT / 'stop.request').exists():
        server.handle_request()
finally:
    server.server_close()
    print('HEALTH CLOSED', flush=True)
