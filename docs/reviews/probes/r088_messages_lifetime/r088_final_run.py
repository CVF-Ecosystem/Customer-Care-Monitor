#!/usr/bin/env python3
"""R088 final live-tree verification (synthetic UI tests, forced type-check and build; no network,
backend, browser or provider). Records actual HEAD, source/test LF hashes, exact commands, exit codes,
test identities (including skips), sanitized output tails and protected-path status.

  python -B r088_final_run.py --project <root> --out <json> --stage tests|typecheck|build|protected|hashes
"""
import argparse
import datetime
import hashlib
import json
import os
import re
import subprocess
import sys
import tempfile
import time

TESTS = ["src/__tests__/conversations-lifetime.spec.ts", "src/__tests__/messages-lifetime.spec.ts",
         "src/__tests__/messages-lifetime-startup.spec.ts", "src/__tests__/messages-lifetime-resources.spec.ts",
         "src/__tests__/messages-screen.spec.ts", "src/__tests__/messages-business-day.spec.ts"]
PRODUCTION = ["frontend/src/views/Messages.vue", "frontend/src/stores/conversations.ts"]
PROTECTED = ["frontend/package.json", "frontend/package-lock.json", "frontend/vite.config.ts", "frontend/tsconfig.json",
             "frontend/tsconfig.app.json", "frontend/tsconfig.node.json", "backend", ".cvf", "scripts", ".github",
             "CVF_SESSION/authority"]
ALLOWED_FRONTEND = ("frontend/src/views/Messages.vue", "frontend/src/stores/conversations.ts", "frontend/src/utils/conversationRequestScope",
                    "frontend/src/__tests__/messages-lifetime", "frontend/src/__tests__/conversations-lifetime",
                    "frontend/src/__tests__/fixtures/conversation-lifetime/", "frontend/src/__tests__/messages-screen.spec.ts",
                    "frontend/src/__tests__/messages-business-day.spec.ts")


def lf_sha(data):
    return hashlib.sha256(data.replace(b"\r\n", b"\n")).hexdigest()


def run(cmd, cwd, timeout):
    start = time.time()
    try:
        proc = subprocess.run(cmd, cwd=cwd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=timeout, shell=(os.name == "nt"))
        return proc.returncode, proc.stdout.decode("utf-8", errors="replace"), round(time.time() - start, 1)
    except subprocess.TimeoutExpired as exc:
        return -9, (exc.stdout or b"").decode("utf-8", errors="replace") + "\nTIMEOUT", round(time.time() - start, 1)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--project", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--stage", required=True, choices=["tests", "typecheck", "build", "protected", "hashes", "manifest", "gates"])
    args = ap.parse_args()
    project = os.path.abspath(args.project)

    def clean(text):
        for variant in {project, project.replace("\\", "/")}:
            text = text.replace(variant, "<ROOT>")
        return re.sub(r"\x1b\[[0-9;]*m", "", text)

    head = subprocess.run(["git", "rev-parse", "HEAD"], cwd=project, stdout=subprocess.PIPE, text=True).stdout.strip()
    result = {"schemaVersion": "r088.finalRun@1", "stage": args.stage, "recordedAtUtc": datetime.datetime.now(datetime.timezone.utc).isoformat(timespec="seconds"),
              "executionHead": head, "providerCalls": 0, "backendRuns": 0, "networkUse": "none"}

    if args.stage == "hashes":
        listed = subprocess.run(["git", "ls-files", "-m", "-o", "--exclude-standard", "frontend/src"], cwd=project, stdout=subprocess.PIPE, text=True).stdout.split("\n")
        files = sorted(p for p in listed if p)
        result["frontendChangedOrNew"] = {p: lf_sha(open(os.path.join(project, p), "rb").read()) for p in files}
        result["baselineProductionLfSha256"] = {p: json.load(open(os.path.join(project, "docs/reviews/probes/r088_dispatch_source_audit_2026-10-12.json")))["sources"][p]["lfNormalizedSha256"] for p in PRODUCTION}

    elif args.stage == "tests":
        out_json = os.path.join(tempfile.gettempdir(), "r088-final-vitest.json")
        if os.path.exists(out_json):
            os.remove(out_json)
        cmd = ["npm", "--prefix", "frontend", "test", "--"] + TESTS + ["--reporter=json", f"--outputFile={out_json}"]
        code, text, secs = run(cmd, project, 900)
        tests, counts = [], {}
        if os.path.isfile(out_json):
            data = json.load(open(out_json, encoding="utf-8"))
            for suite in data.get("testResults", []):
                for t in suite.get("assertionResults", []):
                    tests.append({"file": clean(suite["name"]).split("src/__tests__/")[-1], "title": t.get("fullName"), "status": t.get("status")})
                    counts[t.get("status")] = counts.get(t.get("status"), 0) + 1
        result.update({"command": "npm --prefix frontend test -- " + " ".join(TESTS) + " --reporter=json --outputFile=<tmp>", "exitCode": code,
                       "elapsedSeconds": secs, "counts": counts, "skippedOrTodo": [t["title"] for t in tests if t["status"] not in ("passed", "failed")],
                       "perFileCounts": {}, "tests": tests, "stdoutTail": clean(text)[-1200:]})
        for t in tests:
            result["perFileCounts"][t["file"]] = result["perFileCounts"].get(t["file"], 0) + 1

    elif args.stage in ("typecheck", "build"):
        if args.stage == "typecheck":
            cmd, cwd, label = ["npx", "--no-install", "vue-tsc", "-b", "--force"], os.path.join(project, "frontend"), "Push-Location frontend; npx --no-install vue-tsc -b --force"
        else:
            cmd, cwd, label = ["npm", "--prefix", "frontend", "run", "build"], project, "npm --prefix frontend run build"
        code, text, secs = run(cmd, cwd, 900)
        result.update({"command": label, "cwd": clean(cwd), "exitCode": code, "elapsedSeconds": secs, "stdoutTail": clean(text)[-1500:]})

    elif args.stage == "protected":
        status = subprocess.run(["git", "status", "--porcelain", "--"] + PROTECTED, cwd=project, stdout=subprocess.PIPE, text=True).stdout.strip()
        listed = subprocess.run(["git", "status", "--porcelain", "--", "frontend"], cwd=project, stdout=subprocess.PIPE, text=True).stdout.splitlines()
        outside = []
        for line in listed:
            path = line[3:].strip().strip('"').replace("\\", "/")
            if path.endswith("/"):
                continue
            if not path.startswith(ALLOWED_FRONTEND):
                outside.append(path)
        untracked_dirs = [l[3:] for l in listed if l.startswith("??") and l.rstrip().endswith("/")]
        result.update({"protectedPathStatus": status, "protectedPathsClean": status == "", "frontendPathsOutsideSeed": outside,
                       "untrackedFrontendDirectories": untracked_dirs, "seedUnchanged": subprocess.run(["git", "diff", "--quiet", "HEAD", "--", "CVF_SESSION/authority/CCMAI-RUNTIME-088.json"], cwd=project).returncode == 0})

    elif args.stage == "manifest":
        listed = subprocess.run(["git", "ls-files", "-m", "-o", "--exclude-standard", "-z"], cwd=project, stdout=subprocess.PIPE).stdout.decode("utf-8").split("\0")
        self_rel = os.path.relpath(os.path.abspath(args.out), project).replace("\\", "/")
        files = {}
        for rel in sorted(set(p for p in listed if p)):
            full = os.path.join(project, rel)
            if rel == self_rel or not os.path.isfile(full):
                continue
            data = open(full, "rb").read()
            files[rel] = {"rawSha256": hashlib.sha256(data).hexdigest(), "lfNormalizedSha256": lf_sha(data), "bytes": len(data)}
        result.update({"pathCount": len(files), "note": "every non-ignored changed or untracked path at generation time, except this manifest",
                       "authoritySeedCommit": "45ac7a91a60f9f64239492e0db838c809ca51fca", "dispatchMaterialCommit": "5d11fd4ffd5a912dad633213defd8c1e0d3333b5",
                       "files": files})

    elif args.stage == "gates":
        wo = "docs/work_orders/CCMAI_RUNTIME_088_MESSAGES_LIFETIME.md"
        ret = "docs/reviews/R088_MESSAGES_LIFETIME_WORKER_RETURN_2026-10-12.md"
        core = os.path.join(os.path.dirname(project), ".Controlled-Vibe-Framework-CVF")
        runner = "scripts/cvf_gates/cvf_downstream_gate_runner.py"
        trusted = os.path.join(core, "scripts/lib/downstream_governance/cvf_downstream_gate_runner.py")
        py = [sys.executable, "-B"]
        shell = "pwsh"
        entries = [
            ("preflight", py + ["scripts/cvf_downstream_gate.py", "preflight"], "PASS (7/7 executed gates passed)"),
            ("worker-return-trusted", py + [trusted, "run", "--phase", "worker-return", "--project-root", ".", "--trusted", "--packet", wo, "--packet", ret, "--no-receipt"], "RESULT: PASS"),
            ("pre-dispatch", py + [runner, "run", "--phase", "pre-dispatch", "--project-root", ".", "--packet", wo, "--no-receipt"], "RESULT: PASS"),
            ("worker-return", py + [runner, "run", "--phase", "worker-return", "--project-root", ".", "--packet", wo, "--packet", ret, "--no-receipt"], "RESULT: PASS"),
            ("reviewer-fast", py + [runner, "run", "--phase", "reviewer-fast", "--project-root", ".", "--packet", wo, "--packet", ret, "--no-receipt"], "RESULT: PASS"),
            ("pre-commit", py + [runner, "run", "--phase", "pre-commit", "--project-root", ".", "--packet", wo, "--packet", ret, "--no-receipt"], "RESULT: PASS"),
            ("closeability", py + ["scripts/cvf_gates/check_gate_to_role_closeability.py", "--repo-root", ".", "--base", head, "--head", head, "--enforce"], "Violations: 0"),
            ("catalog-pwsh", ["pwsh", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "scripts/manage_cvf_downstream_catalog.ps1", "-Check"], "[PASS]"),
            ("catalog-windows-powershell", ["powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "scripts/manage_cvf_downstream_catalog.ps1", "-Check"], "[PASS]"),
            ("doctor", ["powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", os.path.join(core, "scripts/check_cvf_workspace_agent_enforcement.ps1"), "-ProjectPath", "."], "RESULT: PASS"),
            ("diff-check", ["git", "diff", "--check"], ""),
        ]
        runs = []
        for name, cmd, expect in entries:
            proc = subprocess.run(cmd, cwd=project, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
            text = clean(proc.stdout.decode("utf-8", errors="replace"))
            applicability = [l.strip() for l in text.splitlines() if "CVF-DG-APPL-01" in l or "CVF-DG-ROUTE-01" in l]
            runs.append({"name": name, "command": clean(" ".join(cmd)), "exitCode": proc.returncode, "expected": expect or "exit 0",
                         "outcome": "AS_EXPECTED" if proc.returncode == 0 and (expect in text) else "UNEXPECTED",
                         "applicabilityAndRouting": applicability, "outputTail": text[-900:]})
            print(name, proc.returncode)
        result.update({"runs": runs, "unexpected": [r["name"] for r in runs if r["outcome"] != "AS_EXPECTED"],
                       "note": "NOT_APPLICABLE_WITH_REASON rows mean that contract was not checked for that packet; the order is COMPLETE_PENDING_REVIEW, not an active dispatch state"})

    json.dump(result, open(args.out, "w", encoding="utf-8"), indent=2, ensure_ascii=False)
    print(args.stage, {k: result.get(k) for k in ("exitCode", "counts", "protectedPathsClean", "frontendPathsOutsideSeed")})
    return 0


if __name__ == "__main__":
    sys.exit(main())
