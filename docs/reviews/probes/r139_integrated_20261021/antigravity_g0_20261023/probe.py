import http.server
import errno
import json
import os
import pathlib
import secrets
import socket
import subprocess
import sys
import time
import urllib.request


def save(path, value):
    tmp = path.with_name(path.name + ".tmp")
    tmp.write_text(json.dumps(value, ensure_ascii=False, sort_keys=True), encoding="utf-8")
    os.replace(tmp, path)


def child(root, nonce):
    deadline = time.monotonic() + 85

    class Handler(http.server.BaseHTTPRequestHandler):
        def setup(self):
            self.request.settimeout(0.5)
            super().setup()

        def do_GET(self):
            if self.path != "/g0/" + nonce:
                self.send_error(404)
                return
            body = ("<!doctype html><html lang='vi'><meta charset='utf-8'>"
                    "<p id='nonce'>" + nonce + "</p>"
                    "<title>Kiểm tra thao tác</title><button id='b' "
                    "onclick=\"document.getElementById('s').textContent='Đã bấm'\">"
                    "Bấm thử</button><p id='s'>Chưa bấm</p></html>").encode("utf-8")
            self.send_response(200)
            self.send_header("Content-Type", "text/html; charset=utf-8")
            self.send_header("Cache-Control", "no-store")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, fmt, *args):
            print("HTTP " + fmt % args, file=sys.stderr, flush=True)

    class BoundedServer(http.server.HTTPServer):
        accepted = 0

        def get_request(self):
            conn, addr = super().get_request()
            self.accepted += 1
            conn.settimeout(0.5)
            print("ACCEPT %d" % self.accepted, file=sys.stderr, flush=True)
            return conn, addr

    server = BoundedServer(("127.0.0.1", 0), Handler)
    server.timeout = 0.25
    port = server.server_address[1]
    save(root / "ready.json", {"nonce": nonce, "host": "127.0.0.1",
                               "port": port, "child_pid": os.getpid()})
    try:
        while (time.monotonic() < deadline and server.accepted < 8
               and not (root / "child.stop").exists()):
            try:
                server.handle_request()
            except (TimeoutError, ConnectionError, OSError) as exc:
                print("CLIENT_ERROR " + type(exc).__name__, file=sys.stderr, flush=True)
    finally:
        server.server_close()
        save(root / "child.final.json", {"child_pid": os.getpid(),
                                         "accepted_connections": server.accepted})
        print("CHILD_EXIT port=%d accepted=%d" % (port, server.accepted), flush=True)


def parent(root):
    nonce = secrets.token_hex(12)
    started = time.monotonic()
    proc = None
    out = None
    err = None
    result = {"status": "HOLD", "nonce": nonce, "child_pid": None,
              "child_exit": None, "original_wait_observed": False,
              "accepted_connections": None,
              "listener_connect_code": None, "listener_refused": None,
              "stop_marker_consumed": False, "errors": []}
    try:
        if any(p.name != "probe.py" for p in root.iterdir()):
            raise RuntimeError("owned directory must start empty except probe.py")
        out = (root / "child.stdout.log").open("xb")
        err = (root / "child.stderr.log").open("xb")
        child_env = {k: os.environ[k] for k in ("SystemRoot", "WINDIR", "TEMP", "TMP")
                     if k in os.environ}
        proc = subprocess.Popen([sys.executable, "-I", "-u", str(root / "probe.py"),
                                 "--child", str(root), nonce], cwd=str(root),
                                env=child_env, stdin=subprocess.DEVNULL,
                                stdout=out, stderr=err)
        result["child_pid"] = proc.pid
        ready = root / "ready.json"
        while not ready.exists() and proc.poll() is None and time.monotonic() - started < 10:
            time.sleep(0.1)
        if not ready.exists():
            raise RuntimeError("child readiness missing")
        data = json.loads(ready.read_text(encoding="utf-8"))
        if (data.get("nonce") != nonce or data.get("host") != "127.0.0.1"
                or data.get("child_pid") != proc.pid):
            raise RuntimeError("child readiness mismatch")
        port = data["port"]
        if type(port) is not int or not 1 <= port <= 65535:
            raise RuntimeError("child port invalid")
        url = "http://127.0.0.1:%d/g0/%s" % (port, nonce)
        class NoRedirect(urllib.request.HTTPRedirectHandler):
            def redirect_request(self, request, fp, code, msg, headers, newurl):
                return None
        opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
        with opener.open(url, timeout=2) as response:
            body = response.read(2048)
            if (response.status != 200 or b"Ch\xc6\xb0a b\xe1\xba\xa5m" not in body
                    or nonce.encode("ascii") not in body or response.geturl() != url):
                raise RuntimeError("inert readiness response mismatch")
        print(json.dumps({"READY_URL": url, "stop_marker": str(root / "stop.marker"),
                          "nonce": nonce}), flush=True)
        marker = root / "stop.marker"
        while time.monotonic() - started < 90 and proc.poll() is None:
            if marker.exists():
                if marker.read_text(encoding="utf-8").strip() != nonce:
                    raise RuntimeError("stop marker nonce mismatch")
                os.replace(marker, root / "stop.consumed")
                result["stop_marker_consumed"] = True
                (root / "child.stop").write_text(nonce, encoding="utf-8")
                break
            time.sleep(0.1)
        if not result["stop_marker_consumed"]:
            raise RuntimeError("stop marker missing or child exited early")
        try:
            result["child_exit"] = proc.wait(timeout=3)
            result["original_wait_observed"] = True
        except subprocess.TimeoutExpired:
            raise RuntimeError("child did not honor stop marker")
        if result["child_exit"] != 0:
            raise RuntimeError("child exited nonzero")
        final_child = json.loads((root / "child.final.json").read_text(encoding="utf-8"))
        result["accepted_connections"] = final_child.get("accepted_connections")
        if (final_child.get("child_pid") != proc.pid
                or type(result["accepted_connections"]) is not int
                or not 1 <= result["accepted_connections"] <= 8):
            raise RuntimeError("child connection count invalid")
        with socket.socket() as sock:
            sock.settimeout(1)
            code = sock.connect_ex(("127.0.0.1", port))
            result["listener_connect_code"] = code
            result["listener_refused"] = code in (errno.ECONNREFUSED, 10061)
        if not result["listener_refused"]:
            raise RuntimeError("owned listener still accepts connections")
        result["status"] = "PARENT_CLEANUP_PASS_BROWSER_REVIEW_PENDING"
    except Exception as exc:
        result["errors"].append(type(exc).__name__ + ": " + str(exc))
    finally:
        if proc is not None:
            try:
                (root / "child.stop").write_text(nonce, encoding="utf-8")
            except Exception as exc:
                result["errors"].append("stop_write: " + repr(exc))
            for action in ("wait", "terminate", "kill"):
                if result["original_wait_observed"]:
                    break
                try:
                    if action == "terminate" and proc.poll() is None:
                        proc.terminate()
                    if action == "kill" and proc.poll() is None:
                        proc.kill()
                    result["child_exit"] = proc.wait(timeout=2)
                    result["original_wait_observed"] = True
                except Exception as exc:
                    result["errors"].append(action + ": " + repr(exc))
            result["child_exit"] = proc.returncode
        for stream in (out, err):
            if stream is not None:
                try:
                    stream.close()
                except Exception as exc:
                    result["errors"].append("log_close: " + repr(exc))
        if result["errors"] or not result["original_wait_observed"]:
            result["status"] = "HOLD"
        result["elapsed_seconds"] = round(time.monotonic() - started, 3)
        try:
            save(root / "parent.result.json", result)
        except Exception as exc:
            result["status"] = "HOLD"
            result["errors"].append("result_write: " + repr(exc))
        print(json.dumps(result, ensure_ascii=False, sort_keys=True), flush=True)
    if result["status"] != "PARENT_CLEANUP_PASS_BROWSER_REVIEW_PENDING":
        raise SystemExit(1)


if __name__ == "__main__":
    if not ((len(sys.argv) == 3 and sys.argv[1] == "--run")
            or (len(sys.argv) == 4 and sys.argv[1] == "--child")):
        raise SystemExit("usage: probe.py --run DIR | --child DIR NONCE")
    raw_work = pathlib.Path(sys.argv[2])
    if raw_work.is_symlink() or not raw_work.is_dir():
        raise SystemExit("owned directory missing or linked")
    work = raw_work.resolve(strict=True)
    if (work / "probe.py").is_symlink() or not (work / "probe.py").is_file():
        raise SystemExit("owned harness missing or linked")
    if sys.argv[1] == "--child":
        child(work, sys.argv[3])
    else:
        parent(work)
