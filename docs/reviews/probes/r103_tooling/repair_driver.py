#!/usr/bin/env python3
"""Explicit, receipt-producing Playwright Go driver recovery; never runs on import."""
import argparse
import base64
import hashlib
import json
import os
import re
import shutil
import stat
import subprocess
import sys
import tarfile
import time
import urllib.parse
import urllib.request
import uuid
from datetime import datetime, timezone
from pathlib import Path, PurePosixPath

PROJECT = Path(__file__).resolve().parents[4]
LOCAL = Path(r"C:\Users\tiennm\AppData\Local")
DRIVER = LOCAL / "ms-playwright-go" / "1.57.0"
BROWSERS = LOCAL / "ms-playwright"
NODE = Path(r"C:\Program Files\nodejs\node.exe")
META_URL = "https://registry.npmjs.org/playwright-core/1.57.0"
NPM_HOSTS = {"registry.npmjs.org"}
PW_HOSTS = {"cdn.playwright.dev", "playwright.download.prss.microsoft.com"}
MAX_BYTES = 800_000_000
DEADLINE = 0
HOST_OVERRIDES = ("PLAYWRIGHT_DOWNLOAD_HOST", "PLAYWRIGHT_CHROMIUM_DOWNLOAD_HOST",
                  "PLAYWRIGHT_FIREFOX_DOWNLOAD_HOST", "PLAYWRIGHT_WEBKIT_DOWNLOAD_HOST",
                  "PLAYWRIGHT_FFMPEG_DOWNLOAD_HOST")


def write_json(path, value):
    with path.open("x", encoding="utf-8", newline="\n") as f:
        json.dump(value, f, indent=2, sort_keys=True)
        f.write("\n")


def digest(path):
    h = hashlib.sha256()
    with path.open("rb") as f:
        for block in iter(lambda: f.read(1024 * 1024), b""):
            h.update(block)
    return h.hexdigest()


def sha512(path):
    h = hashlib.sha512()
    with path.open("rb") as f:
        for block in iter(lambda: f.read(1024 * 1024), b""):
            h.update(block)
    return base64.b64encode(h.digest()).decode("ascii")


def append_jsonl(path, value):
    with path.open("a", encoding="utf-8", newline="\n") as f:
        f.write(json.dumps(value, sort_keys=True) + "\n")


def remaining():
    left = DEADLINE - time.monotonic()
    if left <= 0:
        raise TimeoutError("R103 campaign deadline exceeded")
    return left


def is_reparse(path):
    try:
        st = path.lstat()
    except FileNotFoundError:
        return False
    flag = getattr(stat, "FILE_ATTRIBUTE_REPARSE_POINT", 0x400)
    return stat.S_ISLNK(st.st_mode) or bool(getattr(st, "st_file_attributes", 0) & flag)


def check_path(path):
    resolved = path.resolve(strict=True)
    if os.path.normcase(str(resolved)) != os.path.normcase(str(path)):
        raise RuntimeError(f"path resolves outside its authorized literal: {path} -> {resolved}")
    current = path
    while current != current.parent:
        if is_reparse(current):
            raise RuntimeError(f"symlink/junction/reparse path rejected: {current}")
        current = current.parent


def tool_env():
    env = os.environ.copy()
    cleared = [name for name in HOST_OVERRIDES if env.pop(name, None) is not None]
    env.update({"PLAYWRIGHT_BROWSERS_PATH": str(BROWSERS), "PLAYWRIGHT_SKIP_BROWSER_GC": "1",
                "DEBUG": "pw:install,pw:download"})
    return env, {"PLAYWRIGHT_BROWSERS_PATH": str(BROWSERS), "PLAYWRIGHT_SKIP_BROWSER_GC": "1",
                 "DEBUG": env["DEBUG"], "clearedDownloadHostOverrides": cleared}


def dir_inventory(path):
    return {p.name: ("reparse" if is_reparse(p) else "dir" if p.is_dir() else "file")
            for p in path.iterdir()}


def package_manifest(package):
    rows = []
    for p in sorted(package.rglob("*")):
        if is_reparse(p):
            raise RuntimeError(f"installed package contains a link/junction: {p}")
        if p.is_file():
            rows.append({"path": p.relative_to(package).as_posix(), "bytes": p.stat().st_size,
                         "sha256": digest(p)})
    return rows


def validate_url(url, hosts):
    p = urllib.parse.urlsplit(url)
    if p.scheme != "https" or p.hostname not in hosts or p.username or p.password or p.port not in (None, 443):
        raise RuntimeError(f"URL outside approved HTTPS hosts: {url}")


class Redirects(urllib.request.HTTPRedirectHandler):
    def __init__(self, hosts, audit):
        self.hosts, self.audit = hosts, audit

    def redirect_request(self, req, fp, code, msg, headers, newurl):
        validate_url(newurl, self.hosts)
        append_jsonl(self.audit, {"event": "redirect", "from": req.full_url,
                                  "to": newurl, "status": code})
        return super().redirect_request(req, fp, code, msg, headers, newurl)


def fetch(url, out, cap, opener, total, audit):
    validate_url(url, NPM_HOSTS)
    request_id = uuid.uuid4().hex
    append_jsonl(audit, {"event": "request-intent", "id": request_id, "url": url,
                         "capBytes": cap, "startedUtc": datetime.now(timezone.utc).isoformat()})
    req = urllib.request.Request(url, headers={"User-Agent": "CCMAI-R103/1.0"})
    h, size = hashlib.sha256(), 0
    final, status, failure = None, None, None
    try:
        with opener.open(req, timeout=min(300, remaining())) as response, out.open("xb") as f:
            final, status = response.geturl(), response.status
            validate_url(final, NPM_HOSTS)
            declared = response.headers.get("Content-Length")
            if declared and int(declared) > cap:
                raise RuntimeError("response exceeds byte limit")
            while True:
                remaining()
                block = response.read(1024 * 1024)
                if not block:
                    break
                size += len(block)
                total[0] += len(block)
                if size > cap or total[0] > MAX_BYTES:
                    raise RuntimeError("download byte limit exceeded")
                f.write(block)
                h.update(block)
    except Exception as exc:
        failure = f"{type(exc).__name__}: {exc}"
        raise
    finally:
        append_jsonl(audit, {"event": "request-result", "id": request_id, "url": url,
            "finalUrl": final, "status": status, "bytes": size, "sha256": h.hexdigest(),
            "failure": failure, "finishedUtc": datetime.now(timezone.utc).isoformat()})
    return {"requested": url, "final": final, "status": status, "bytes": size, "sha256": h.hexdigest()}


def extract_package(archive, dest):
    dest.mkdir()
    check_path(dest)
    seen, spellings, kinds, total = set(), {}, {}, 0
    reserved = {"CON", "PRN", "AUX", "NUL"} | {f"COM{i}" for i in range(1, 10)} | {f"LPT{i}" for i in range(1, 10)}

    def safe_component(part, raw):
        if part in ("", ".", "..") or part.endswith((".", " ")) or ":" in part:
            raise RuntimeError(f"unsafe Windows archive component: {raw!r}")
        if any(ord(ch) < 32 or ch in '<>"|?*' for ch in part):
            raise RuntimeError(f"unsafe Windows archive component: {raw!r}")
        stem = part.split(".", 1)[0].upper()
        if stem in reserved:
            raise RuntimeError(f"reserved Windows archive name: {raw!r}")

    with tarfile.open(archive, "r:gz") as tf:
        members = tf.getmembers()
        if len(members) > 20000:
            raise RuntimeError("archive member limit exceeded")
        for m in members:
            raw = m.name
            if "\\" in raw or raw.startswith("/") or "\x00" in raw:
                raise RuntimeError(f"unsafe archive path: {raw!r}")
            parts = raw.split("/")
            if parts and parts[-1] == "":
                parts = parts[:-1]
            if not parts or parts[0] != "package" or any(x in ("", ".", "..") for x in parts):
                raise RuntimeError(f"unsafe archive path: {raw!r}")
            for part in parts[1:]:
                safe_component(part, raw)
            if len(parts) == 1 and m.isdir():
                key = ""
                if key in seen:
                    raise RuntimeError(f"duplicate archive path: {raw!r}")
                seen.add(key)
                continue
            if not (m.isfile() or m.isdir()):
                raise RuntimeError(f"archive links/special files rejected: {raw!r}")
            rel = PurePosixPath(*parts[1:])
            key = rel.as_posix().casefold()
            if key in seen:
                raise RuntimeError(f"duplicate archive path: {raw!r}")
            seen.add(key)
            for i in range(1, len(parts)):
                original = "/".join(parts[1:i + 1])
                folded = original.casefold()
                if folded in spellings and spellings[folded] != original:
                    raise RuntimeError(f"case-colliding archive paths: {raw!r}")
                spellings[folded] = original
                if i < len(parts) - 1 and kinds.get(folded) == "file":
                    raise RuntimeError(f"archive path descends through file: {raw!r}")
            if key in kinds:
                raise RuntimeError(f"archive file/directory collision: {raw!r}")
            if m.isfile() and any(existing.startswith(key + "/") for existing in kinds):
                raise RuntimeError(f"archive file replaces an existing directory: {raw!r}")
            kinds[key] = "dir" if m.isdir() else "file"
            out = dest.joinpath(*rel.parts)
            resolved = out.resolve(strict=False)
            if not resolved.is_relative_to(dest.resolve(strict=True)):
                raise RuntimeError(f"archive path escapes staging: {raw!r}")
            if m.isdir():
                out.mkdir(parents=True, exist_ok=True)
                check_path(out)
                continue
            total += m.size
            if m.size < 0 or total > 300_000_000:
                raise RuntimeError("expanded package byte limit exceeded")
            out.parent.mkdir(parents=True, exist_ok=True)
            check_path(out.parent)
            src = tf.extractfile(m)
            if src is None:
                raise RuntimeError(f"missing archive payload: {raw!r}")
            with src, out.open("xb") as f:
                shutil.copyfileobj(src, f, 1024 * 1024)


def capture(run, label, argv, env=None):
    try:
        cp = subprocess.run(argv, cwd=PROJECT, env=env, stdin=subprocess.DEVNULL,
                            capture_output=True, timeout=min(300, remaining()), check=False)
        code, stdout, stderr = cp.returncode, cp.stdout, cp.stderr
        timed_out = False
    except subprocess.TimeoutExpired as exc:
        code, stdout, stderr, timed_out = None, exc.stdout or b"", exc.stderr or b"", True
    out, err = run / f"{label}.stdout.txt", run / f"{label}.stderr.txt"
    out.write_bytes(stdout)
    err.write_bytes(stderr)
    write_json(run / f"{label}.command.json", {
        "argv": [str(x) for x in argv], "exitCode": code, "timedOut": timed_out,
        "stdout": out.name, "stdoutSha256": digest(out), "stderr": err.name, "stderrSha256": digest(err),
        "environmentOverrides": {k: v for k, v in (env or {}).items() if k in {
            "PLAYWRIGHT_BROWSERS_PATH", "PLAYWRIGHT_SKIP_BROWSER_GC", "DEBUG"}},
    })
    if timed_out or code != 0:
        raise RuntimeError(f"{label} failed or timed out; raw output preserved in {run}")
    return stdout.decode("utf-8", "replace") + "\n" + stderr.decode("utf-8", "replace")


def new_run(action):
    rel = ".cvf/runtime/r103_tooling"
    check = subprocess.run(["git", "check-ignore", "--quiet", rel], cwd=PROJECT, check=False)
    if check.returncode != 0:
        raise RuntimeError("receipt path is not git-ignored")
    root = PROJECT / rel
    root.mkdir(parents=True, exist_ok=True)
    run = root / (datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") + "_" + action + "_" + uuid.uuid4().hex[:8])
    run.mkdir()
    return run


def check_local_paths(require_empty_driver):
    local = Path(os.environ.get("LOCALAPPDATA", ""))
    if os.name != "nt" or os.path.normcase(str(local.resolve(strict=True))) != os.path.normcase(str(LOCAL.resolve(strict=True))):
        raise RuntimeError("unexpected Windows/LOCALAPPDATA; refusing cache writes")
    for path in (LOCAL, DRIVER.parent, DRIVER, BROWSERS, NODE):
        check_path(path)
    if not DRIVER.is_dir() or (require_empty_driver and any(DRIVER.iterdir())):
        raise RuntimeError("driver 1.57.0 target must exist and be empty for install-driver")
    if not BROWSERS.is_dir() or not NODE.is_file():
        raise RuntimeError("expected browser cache or existing node.exe is missing")


def browser_plan(run, node, package):
    env, env_record = tool_env()
    raw = capture(run, "chromium-dry-run", [node, package / "cli.js", "install", "--dry-run", "chromium"], env)
    urls = sorted(set(re.findall(r"https://[^\s\"'<>]+", raw)))
    unexpected = []
    for url in urls:
        try:
            validate_url(url.rstrip(".,)"), PW_HOSTS)
        except RuntimeError:
            unexpected.append(url)
    data = json.loads((package / "browsers.json").read_text(encoding="utf-8"))
    rows = [b for b in data.get("browsers", []) if b.get("name", "").startswith("chromium") or b.get("name") == "ffmpeg"]
    write_json(run / "chromium-plan.json", {"browserEntries": rows, "urls": urls,
        "unexpectedHosts": unexpected, "downloadHostEnforcement": "URLs emitted by dry-run only; actual browser CLI redirects are not intercepted",
        "environment": env_record})
    if not urls:
        raise RuntimeError("dry-run reported no HTTPS download URLs")
    if unexpected:
        raise RuntimeError("dry-run emitted URL outside approved official host set")


def install_driver(run):
    check_local_paths(require_empty_driver=True)
    write_json(run / "browser-cache-before.json", dir_inventory(BROWSERS))
    node_hash = digest(NODE)
    node_ver = capture(run, "existing-node-version", [NODE, "--version"]).strip()
    if node_ver != "v24.19.0":
        raise RuntimeError(f"existing Node version changed: {node_ver}")
    audit = run / "npm-requests.jsonl"
    audit.touch(exist_ok=False)
    redirects = Redirects(NPM_HOSTS, audit)
    opener, total = urllib.request.build_opener(redirects), [0]
    meta_path = run / "npm-metadata.json"
    meta_receipt = fetch(META_URL, meta_path, 2_000_000, opener, total, audit)
    meta = json.loads(meta_path.read_text(encoding="utf-8"))
    if (meta.get("name"), meta.get("version")) != ("playwright-core", "1.57.0"):
        raise RuntimeError("npm metadata name/version mismatch")
    dist = meta.get("dist", {})
    tar_url = dist.get("tarball", "")
    validate_url(tar_url, NPM_HOSTS)
    if urllib.parse.urlsplit(tar_url).path != "/playwright-core/-/playwright-core-1.57.0.tgz":
        raise RuntimeError("unexpected npm tarball path")
    tarball = run / "playwright-core-1.57.0.tgz"
    tar_receipt = fetch(tar_url, tarball, MAX_BYTES, opener, total, audit)
    sri = [v[7:] for v in dist.get("integrity", "").split() if v.startswith("sha512-")]
    actual = sha512(tarball)
    if not sri or actual not in sri:
        raise RuntimeError("npm dist.integrity SHA512 mismatch")
    package = run / "package"
    extract_package(tarball, package)
    info = json.loads((package / "package.json").read_text(encoding="utf-8"))
    if (info.get("name"), info.get("version")) != ("playwright-core", "1.57.0"):
        raise RuntimeError("extracted package name/version mismatch")
    if not all((package / name).is_file() for name in ("cli.js", "browsers.json")):
        raise RuntimeError("required package files missing")
    staged_node = run / "node.exe"
    shutil.copy2(NODE, staged_node)
    if digest(staged_node) != node_hash:
        raise RuntimeError("staged node.exe hash mismatch")
    if any(DRIVER.iterdir()):
        raise RuntimeError("driver target changed during staging; refusing publish")
    shutil.copytree(package, DRIVER / "package")
    with staged_node.open("rb") as src, (DRIVER / "node.exe").open("xb") as dst:
        shutil.copyfileobj(src, dst, 1024 * 1024)
    installed_node_hash = digest(DRIVER / "node.exe")
    if installed_node_hash != node_hash:
        raise RuntimeError("published node.exe hash mismatch; target may be partial")
    installed_package_files = package_manifest(DRIVER / "package")
    write_json(run / "network.json", {"bytes": total[0], "metadata": meta_receipt,
        "tarball": tar_receipt, "tarballSRI": "sha512-" + actual,
        "browserTransferLimits": "not applicable; browser download is a separate action"})
    write_json(run / "published-files.json", {"target": str(DRIVER), "nodeVersion": node_ver,
        "nodeSha256": node_hash, "installedNodeSha256": installed_node_hash,
        "package": "playwright-core@1.57.0", "packageFiles": installed_package_files,
        "browserCacheTopLevel": dir_inventory(BROWSERS)})
    env, _ = tool_env()
    version = capture(run, "driver-version", [DRIVER / "node.exe", DRIVER / "package" / "cli.js", "--version"], env)
    if version.strip() != "Version 1.57.0":
        raise RuntimeError(f"installed driver version mismatch: {version.strip()}")
    browser_plan(run, DRIVER / "node.exe", DRIVER / "package")
    write_json(run / "browser-cache-after-driver", dir_inventory(BROWSERS))


def verify_installed_driver(run):
    info = json.loads((DRIVER / "package" / "package.json").read_text(encoding="utf-8"))
    if (info.get("name"), info.get("version")) != ("playwright-core", "1.57.0"):
        raise RuntimeError("installed package name/version mismatch")
    if not (DRIVER / "package" / "cli.js").is_file() or not (DRIVER / "package" / "browsers.json").is_file():
        raise RuntimeError("installed CLI/browser metadata missing")
    candidates = []
    root = PROJECT / ".cvf/runtime/r103_tooling"
    if root.is_dir():
        for receipt in root.iterdir():
            published, result = receipt / "published-files.json", receipt / "result.json"
            if published.is_file() and result.is_file():
                try:
                    if json.loads(result.read_text(encoding="utf-8")).get("result") == "PASS":
                        candidates.append((receipt.stat().st_mtime_ns, published))
                except (OSError, ValueError):
                    continue
    if not candidates:
        raise RuntimeError("no successful R103 install-driver manifest receipt")
    published = json.loads(max(candidates)[1].read_text(encoding="utf-8"))
    if os.path.normcase(published.get("target", "")) != os.path.normcase(str(DRIVER)):
        raise RuntimeError("published driver receipt target mismatch")
    if published.get("package") != "playwright-core@1.57.0":
        raise RuntimeError("published driver receipt package mismatch")
    actual_files = package_manifest(DRIVER / "package")
    if actual_files != published.get("packageFiles"):
        raise RuntimeError("installed package differs from successful published manifest")
    actual_node = digest(DRIVER / "node.exe")
    if actual_node != published.get("nodeSha256") or actual_node != published.get("installedNodeSha256"):
        raise RuntimeError("installed node.exe differs from successful published manifest")
    node_version = capture(run, "installed-node-version", [DRIVER / "node.exe", "--version"])
    if node_version.strip() != "v24.19.0" or published.get("nodeVersion") != node_version.strip():
        raise RuntimeError("installed Node version does not match successful published manifest")
    env, _ = tool_env()
    version = capture(run, "installed-driver-version", [DRIVER / "node.exe", DRIVER / "package" / "cli.js", "--version"], env)
    if version.strip() != "Version 1.57.0":
        raise RuntimeError(f"installed driver CLI version mismatch: {version.strip()}")
    return published


def install_browser(run):
    check_local_paths(require_empty_driver=False)
    published = verify_installed_driver(run)
    before = dir_inventory(BROWSERS)
    write_json(run / "browser-cache-before.json", before)
    browser_plan(run, DRIVER / "node.exe", DRIVER / "package")
    env, env_record = tool_env()
    log = capture(run, "chromium-install", [DRIVER / "node.exe", DRIVER / "package" / "cli.js", "install", "chromium"], env)
    urls = sorted(set(re.findall(r"https://[^\s\"'<>]+", log)))
    unexpected = []
    for url in urls:
        try:
            validate_url(url.rstrip(".,)"), PW_HOSTS)
        except RuntimeError:
            unexpected.append(url)
    after = dir_inventory(BROWSERS)
    write_json(run / "browser-cache-after.json", after)
    if any(after.get(name) != kind for name, kind in before.items()):
        raise RuntimeError("preexisting browser cache entry removed or changed type")
    if unexpected:
        raise RuntimeError("install output reported URL outside approved host set; CLI redirects remain unobserved")
    write_json(run / "chromium-install-result.json", {"urls": urls,
        "driverManifestReceipt": published, "environment": env_record,
        "downloadLimitations": "CLI-emitted URLs are recorded; redirect hosts and aggregate download bytes are not intercepted or enforced"})


def main():
    global DEADLINE
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("action", choices=("install-driver", "install-browser"))
    args = ap.parse_args()
    run = None
    started = time.monotonic()
    DEADLINE = started + 1800
    try:
        run = new_run(args.action)
        if args.action == "install-driver":
            install_driver(run)
        else:
            install_browser(run)
        write_json(run / "result.json", {"result": "PASS", "seconds": round(time.monotonic() - started, 2)})
        print(f"PASS; receipts: {run}")
        return 0
    except Exception as exc:
        if run:
            partial = dir_inventory(DRIVER) if args.action == "install-driver" and DRIVER.is_dir() else {}
            write_json(run / "result.json", {"result": "STOP", "error": str(exc),
                "partialDriverTarget": partial, "seconds": round(time.monotonic() - started, 2)})
        print(f"STOP: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
