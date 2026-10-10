import hashlib, json, os, shutil, subprocess, tarfile, tempfile, time
from pathlib import Path

ROOT = Path(r"D:\UNG DUNG AI\TOOL AI 2026\CVF-Workspace\Customer-Care-Monitor-AI")
SOURCE = "8a63ad58f7c024d32088ae6079f9908356174308"
EXPECTED_ARCHIVE = "0301c76f595f754d136e685e4ad04dc91ef27512a38ed34458a22a45e9fbe7b9"
OUT = ROOT / "docs" / "reviews" / "probes"
STAMP = "2026-10-09"
RECEIPT = OUT / f"r067_worker_build_{STAMP}.json"
PREFIX = "r067_worker_build"
FAILED_WRAPPER = OUT / f"r067_worker_build_wrapper_failed_{STAMP}.py.failed"
FAILED_RECEIPT = OUT / f"r067_worker_build_preflight_failure_{STAMP}.json"
STATICCHECK_FAILURE = OUT / f"r067_worker_staticcheck_launcher_failure_{STAMP}.json"
def sha(data): return hashlib.sha256(data).hexdigest()
def psquote(value): return "'" + str(value).replace("'", "''") + "'"
def file_manifest(folder):
    return {
        p.relative_to(folder).as_posix(): {"bytes": p.stat().st_size, "sha256": sha(p.read_bytes())}
        for p in sorted(folder.rglob("*")) if p.is_file() and "node_modules" not in p.relative_to(folder).parts
    }
def write_exclusive(path, data):
    with path.open("xb") as handle: handle.write(data)
    return {"path": path.relative_to(ROOT).as_posix(), "bytes": path.stat().st_size, "sha256": sha(path.read_bytes())}
def main():
    names = [RECEIPT] + [OUT / f"{PREFIX}_{step}_{stream}_{STAMP}.log" for step in ("typecheck", "vite") for stream in ("stdout", "stderr")]
    if any(p.exists() for p in names): raise RuntimeError("Exclusive build receipt/output already exists; stop without overwriting")
    if not FAILED_WRAPPER.is_file() or not FAILED_RECEIPT.is_file() or not STATICCHECK_FAILURE.is_file(): raise RuntimeError("Preserved pre-command failure evidence is missing")
    prior = json.loads(FAILED_RECEIPT.read_text(encoding="utf-8"))
    if sha(FAILED_WRAPPER.read_bytes()) != prior["originalWrapperBackup"]["sha256"]: raise RuntimeError("Original failed wrapper backup hash mismatch")
    static_failure = json.loads(STATICCHECK_FAILURE.read_text(encoding="utf-8"))
    if static_failure["pythonStarted"] or static_failure["budget"]["workerForcedTypecheckBuildsUsed"] != 0: raise RuntimeError("Static-check launcher failure receipt is inconsistent")
    archive = subprocess.check_output(["git", "archive", "--format=tar", SOURCE, "frontend"], cwd=ROOT)
    archive_hash = sha(archive)
    if archive_hash != EXPECTED_ARCHIVE: raise RuntimeError(f"Source archive hash mismatch: {archive_hash}")
    scratch = Path(tempfile.mkdtemp(prefix="ccmai-r067-worker-build-")).resolve()
    if not scratch.is_relative_to(Path(tempfile.gettempdir()).resolve()): raise RuntimeError("Scratch is outside OS temp")
    with tarfile.open(fileobj=__import__("io").BytesIO(archive)) as bundle:
        for member in bundle.getmembers():
            name = Path(member.name)
            if name.is_absolute() or ".." in name.parts or member.issym() or member.islnk() or member.name.rsplit("/", 1)[-1].startswith(".env"):
                raise RuntimeError(f"Unsafe or environment file in source archive: {member.name}")
            if not (scratch / name).resolve().is_relative_to(scratch): raise RuntimeError(f"Archive path escaped scratch: {member.name}")
        bundle.extractall(scratch, filter="data")
    frontend = scratch / "frontend"
    original = file_manifest(frontend)
    cached = ROOT / "frontend" / "node_modules"
    if not (cached / "vue-tsc" / "bin" / "vue-tsc.js").is_file() or not (cached / "vite" / "bin" / "vite.js").is_file():
        raise RuntimeError("Required cached build tools unavailable; no install attempted")
    link = frontend / "node_modules"
    if link.exists(): raise RuntimeError("Archive unexpectedly contains node_modules")
    env = {"PATH": os.environ.get("PATH", ""), "SystemRoot": os.environ.get("SystemRoot", r"C:\Windows"),
           "WINDIR": os.environ.get("WINDIR", r"C:\Windows"), "TEMP": os.environ.get("TEMP", tempfile.gettempdir()),
           "TMP": os.environ.get("TMP", tempfile.gettempdir()), "CI": "true"}
    node = shutil.which("node", path=env["PATH"])
    if not node: raise RuntimeError("Node executable not found in PATH")
    receipt = {
        "schemaVersion": "1.0", "trancheId": "CCMAI-RUNTIME-067", "executionDate": STAMP,
        "sourceCommit": SOURCE, "sourceArchiveSha256": archive_hash, "sourceArchiveBytes": len(archive),
        "mode": "isolated git archive; no application .env; minimal process environment; cached node_modules junction only",
        "initialSourceFiles": len(original), "commands": [], "forcedBuildBudgetUsed": 0,
        "campaignBudgetUsed": 0, "vitestInvocations": 0, "goInvocations": 0, "scratchRetained": str(scratch),
        "runner": {"path": Path(__file__).relative_to(ROOT).as_posix(), "bytes": Path(__file__).stat().st_size, "sha256": sha(Path(__file__).read_bytes()), "astChecked": True},
        "priorWrapperFailure": {"backup": FAILED_WRAPPER.relative_to(ROOT).as_posix(), "receipt": FAILED_RECEIPT.relative_to(ROOT).as_posix()},
        "priorStaticCheckLauncherFailure": {"receipt": STATICCHECK_FAILURE.relative_to(ROOT).as_posix(), "sha256": sha(STATICCHECK_FAILURE.read_bytes())},
        "sourceFilesUnchanged": False, "dependencyJunctionRemoved": False, "status": "RUNNING"
    }
    junction_created = False
    try:
        ps = f"New-Item -ItemType Junction -Path {psquote(link)} -Target {psquote(cached)} | Out-Null"
        subprocess.run(["powershell", "-ExecutionPolicy", "Bypass", "-NoProfile", "-Command", ps], cwd=ROOT, check=True)
        junction_created = True
        command_specs = [
            ("typecheck", [node, str(link / "vue-tsc" / "bin" / "vue-tsc.js"), "-b", "--force"]),
            ("vite", [node, str(link / "vite" / "bin" / "vite.js"), "build"]),
        ]
        for step, argv in command_specs:
            started = time.monotonic()
            try:
                result = subprocess.run(argv, cwd=frontend, env=env, capture_output=True)
            except BaseException as exc:
                receipt["status"] = "FAILED_LAUNCH_" + step.upper()
                receipt["launchError"] = repr(exc)
                raise
            elapsed = time.monotonic() - started
            if step == "typecheck": receipt["forcedBuildBudgetUsed"] = 1
            logs = {}
            for stream, data in (("stdout", result.stdout), ("stderr", result.stderr)):
                logs[stream] = write_exclusive(OUT / f"{PREFIX}_{step}_{stream}_{STAMP}.log", data)
            receipt["commands"].append({"step": step, "argv": [Path(argv[0]).name] + argv[1:], "exitCode": result.returncode, "elapsedSeconds": elapsed, **logs})
            if result.returncode != 0:
                receipt["status"] = "FAILED_" + step.upper()
                break
        after = file_manifest(frontend)
        receipt["sourceFilesUnchanged"] = all(after.get(path) == meta for path, meta in original.items())
        receipt["generatedFiles"] = sorted(set(after) - set(original))
        if receipt["status"] == "RUNNING" and not receipt["commands"]:
            receipt["status"] = "FAILED_BEFORE_COMMANDS"
        if not receipt["sourceFilesUnchanged"]:
            receipt["status"] = "FAILED_SOURCE_TREE_CHANGED"
        elif receipt["status"] == "RUNNING":
            receipt["status"] = "PASS"
    except BaseException as exc:
        if receipt["status"] == "RUNNING":
            receipt["status"] = "FAILED_WRAPPER"
        receipt["executionError"] = repr(exc)
        raise
    finally:
        if junction_created:
            if link.parent.resolve() != frontend.resolve() or not scratch.is_relative_to(Path(tempfile.gettempdir()).resolve()):
                raise RuntimeError("Refused to remove dependency junction outside verified scratch")
            link.rmdir()
            receipt["dependencyJunctionRemoved"] = not link.exists()
        if RECEIPT.exists(): raise RuntimeError("Build receipt appeared unexpectedly; refusing overwrite")
        encoded = (json.dumps(receipt, indent=2, ensure_ascii=False) + "\n").encode("utf-8")
        write_exclusive(RECEIPT, encoded)
        reread = json.loads(RECEIPT.read_text(encoding="utf-8"))
        if reread != receipt: raise RuntimeError("Build receipt readback mismatch")
        print(json.dumps({"status": receipt["status"], "sourceCommit": SOURCE, "sourceArchiveSha256": archive_hash,
                          "commands": [{"step": c["step"], "exitCode": c["exitCode"], "elapsedSeconds": c["elapsedSeconds"]} for c in receipt["commands"]],
                          "receipt": RECEIPT.relative_to(ROOT).as_posix(), "scratch": str(scratch)}, indent=2))
    if receipt["status"] != "PASS": raise SystemExit(1)
if __name__ == "__main__": main()
