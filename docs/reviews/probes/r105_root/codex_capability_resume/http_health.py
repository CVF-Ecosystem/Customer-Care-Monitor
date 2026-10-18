"""Bounded loopback-only browser capability probe; no product/DB interaction."""
from http.server import BaseHTTPRequestHandler, HTTPServer
import time

PAGE = b'<!doctype html><title>R105 browser capability</title><button onclick="this.textContent=\'Control OK\'">Check control</button>'

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path not in ('/', '/favicon.ico'):
            self.send_error(404)
            return
        self.send_response(200)
        self.send_header('Content-Type', 'text/html; charset=utf-8')
        self.send_header('Content-Length', str(len(PAGE)))
        self.end_headers()
        self.wfile.write(PAGE)

if __name__ == '__main__':
    with HTTPServer(('127.0.0.1', 0), Handler) as server:
        server.timeout = 1
        print(f'HEALTH_ORIGIN=http://127.0.0.1:{server.server_port}', flush=True)
        deadline = time.monotonic() + 180
        while time.monotonic() < deadline:
            server.handle_request()
