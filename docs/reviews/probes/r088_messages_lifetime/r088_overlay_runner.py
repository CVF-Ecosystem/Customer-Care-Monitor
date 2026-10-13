#!/usr/bin/env python3
"""R088 isolated-overlay test runner (stdlib only; synthetic UI tests; no network, backend or provider).

Builds a disposable copy of frontend/ OUTSIDE the live worktree (node_modules is a junction to the
locked, ignored install), optionally swaps in the BASELINE production sources from git (`--source head`)
or applies a declared mutation, runs the named vitest files with the JSON reporter, and records a
sanitized per-test result file. The live worktree is never written except for the evidence file.

  python -B r088_overlay_runner.py --project <root> --work <dir> --label baseline --source head --out <json> [--files ...]
  python -B r088_overlay_runner.py ... --label m1 --source live --mutate '{"path": "...", "old": "...", "new": "...", "expected": 1}'
"""
import argparse
import datetime
import hashlib
import json
import os
import re
import shutil
import stat
import subprocess
import sys
import time

PRODUCTION = ["frontend/src/views/Messages.vue", "frontend/src/stores/conversations.ts"]
IGNORE = shutil.ignore_patterns("node_modules", "dist", ".tmp", ".vite", "*.tsbuildinfo")


def lf_sha(data):
    return hashlib.sha256(data.replace(b"\r\n", b"\n")).hexdigest()


def rmtree(path):
    def onerror(func, target, _exc):
        try:
            os.chmod(target, stat.S_IWRITE)
            func(target)
        except OSError:
            pass
    if os.path.lexists(path):
        if os.path.isdir(path) and not os.path.islink(path):
            # a junction must be removed as a link, never traversed
            junction = os.path.join(path, "frontend", "node_modules")
            if os.path.lexists(junction):
                os.rmdir(junction)  # removes only the link, never the locked install it points to
            shutil.rmtree(path, onerror=onerror)


def git_show(project, rel):
    return subprocess.run(["git", "show", f"HEAD:{rel}"], cwd=project, stdout=subprocess.PIPE, check=True).stdout


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--project", required=True)
    ap.add_argument("--work", required=True)
    ap.add_argument("--label", required=True)
    ap.add_argument("--source", choices=["live", "head"], required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--files", nargs="*", default=None)
    ap.add_argument("--mutate", action="append", default=[], help="JSON object: path, old, new, expected")
    ap.add_argument("--timeout", type=int, default=900)
    args = ap.parse_args()

    project = os.path.abspath(args.project)
    overlay = os.path.join(os.path.abspath(args.work), args.label)
    rmtree(overlay)
    os.makedirs(overlay)
    dest = os.path.join(overlay, "frontend")
    shutil.copytree(os.path.join(project, "frontend"), dest, ignore=IGNORE)
    subprocess.run(["cmd", "/c", "mklink", "/J", os.path.join(dest, "node_modules"), os.path.join(project, "frontend", "node_modules")],
                   check=True, stdout=subprocess.DEVNULL)

    swapped, mutations = [], []
    if args.source == "head":
        for rel in PRODUCTION:
            data = git_show(project, rel)
            with open(os.path.join(overlay, *rel.split("/")), "wb") as fh:
                fh.write(data.replace(b"\r\n", b"\n").replace(b"\n", b"\r\n"))
            swapped.append({"path": rel, "lfSha256": lf_sha(data)})
    for spec in args.mutate:
        parsed = json.loads(spec)
        rel, old, new, expected = parsed["path"], parsed["old"], parsed["new"], int(parsed["expected"])
        full = os.path.join(overlay, *rel.split("/"))
        text = open(full, "rb").read().decode("utf-8").replace("\r\n", "\n")
        count = text.count(old)
        record = {"path": rel, "expectedMatches": expected, "actualMatches": count, "applied": False}
        if count == expected:
            patched = text.replace(old, new)
            open(full, "wb").write(patched.replace("\n", "\r\n").encode("utf-8"))
            record.update({"applied": True, "beforeLfSha256": lf_sha(text.encode()), "afterLfSha256": lf_sha(patched.encode())})
        mutations.append(record)
    if any(not m["applied"] for m in mutations):
        result = {"label": args.label, "disposition": "NOT_APPLIED", "mutations": mutations}
        json.dump(result, open(args.out, "w", encoding="utf-8"), indent=2, ensure_ascii=False)
        print("NOT_APPLIED", mutations)
        return 3

    files = args.files if args.files else None
    json_out = os.path.join(overlay, "vitest-result.json")
    cmd = ["npm", "--prefix", dest, "test", "--"] + (files or []) + ["--reporter=json", f"--outputFile={json_out}"]
    start = time.time()
    try:
        proc = subprocess.run(cmd, cwd=overlay, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=args.timeout, shell=(os.name == "nt"))
        code, raw = proc.returncode, proc.stdout.decode("utf-8", errors="replace")
    except subprocess.TimeoutExpired as exc:
        code, raw = -9, ((exc.stdout or b"").decode("utf-8", errors="replace")) + "\nTIMEOUT"
    elapsed = round(time.time() - start, 1)

    def clean(text):
        for variant in {overlay, overlay.replace("\\", "/"), project, project.replace("\\", "/")}:
            text = text.replace(variant, "<ROOT>")
        return re.sub(r"\x1b\[[0-9;]*m", "", text)

    tests = []
    suites_failed = []
    if os.path.isfile(json_out):
        data = json.load(open(json_out, encoding="utf-8"))
        for suite in data.get("testResults", []):
            name = clean(suite.get("name", "")).split("<ROOT>/")[-1]
            if suite.get("status") != "passed" and not suite.get("assertionResults"):
                suites_failed.append({"file": name, "status": suite.get("status"), "message": clean(suite.get("message", ""))[:1500]})
            for t in suite.get("assertionResults", []):
                tests.append({
                    "file": name.split("src/__tests__/")[-1], "title": t.get("fullName"), "status": t.get("status"),
                    "failure": clean("\n".join(t.get("failureMessages") or []))[:900] if t.get("status") == "failed" else None,
                })
    counts = {}
    for t in tests:
        counts[t["status"]] = counts.get(t["status"], 0) + 1
    result = {
        "schemaVersion": "r088.overlayRun@1", "label": args.label, "recordedAtUtc": datetime.datetime.now(datetime.timezone.utc).isoformat(timespec="seconds"),
        "source": args.source, "swappedProduction": swapped, "mutations": mutations,
        "command": clean(" ".join(cmd)), "exitCode": code, "elapsedSeconds": elapsed, "counts": counts,
        "suiteLevelFailures": suites_failed, "tests": tests, "stdoutTail": clean(raw)[-1500:],
        "liveWorktreeMutated": False,
    }
    json.dump(result, open(args.out, "w", encoding="utf-8"), indent=2, ensure_ascii=False)
    print(f"{args.label}: exit={code} counts={counts} suiteFailures={len(suites_failed)} elapsed={elapsed}s")
    return 0


if __name__ == "__main__":
    sys.exit(main())
