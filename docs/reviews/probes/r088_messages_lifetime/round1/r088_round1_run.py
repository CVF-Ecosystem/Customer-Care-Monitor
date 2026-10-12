"""R088 round1 evidence runner (worker ccmai-r088-worker). Synthetic UI tests only: no provider, backend, DB, Docker, Go, SMTP or customer data.
Usage: python -B r088_round1_run.py <stage> [--tests path ...]
stages: oldsource | tests | typecheck | build | hashes
Every invocation, including failed or partial ones, is written to its own numbered JSON receipt next to this script."""
import hashlib, json, os, subprocess, sys, tempfile, time, datetime

HERE = os.path.dirname(os.path.abspath(__file__))
PROJECT = os.path.abspath(os.path.join(HERE, "..", "..", "..", "..", ".."))
FRONTEND = os.path.join(PROJECT, "frontend")
NEW_TESTS = [
    "src/__tests__/messages-lifetime-startup.spec.ts",
    "src/__tests__/messages-lifetime.spec.ts",
    "src/__tests__/messages-lifetime-resources.spec.ts",
    "src/__tests__/conversations-lifetime.spec.ts",
]
EXISTING = ["src/__tests__/messages-screen.spec.ts", "src/__tests__/messages-business-day.spec.ts"]
TRACKED = [
    "frontend/src/views/Messages.vue", "frontend/src/stores/conversations.ts",
    "frontend/src/__tests__/messages-lifetime-startup.spec.ts", "frontend/src/__tests__/messages-lifetime.spec.ts",
    "frontend/src/__tests__/messages-lifetime-resources.spec.ts", "frontend/src/__tests__/conversations-lifetime.spec.ts",
    "frontend/src/__tests__/fixtures/conversation-lifetime/harness.ts",
    "frontend/src/__tests__/messages-screen.spec.ts", "frontend/src/__tests__/messages-business-day.spec.ts",
    "frontend/package.json", "frontend/package-lock.json",
]


def lf_sha(rel):
    data = open(os.path.join(PROJECT, rel), "rb").read().replace(b"\r\n", b"\n")
    return hashlib.sha256(data).hexdigest()


def head():
    return subprocess.run(["git", "rev-parse", "HEAD"], cwd=PROJECT, capture_output=True, text=True).stdout.strip()


def next_receipt(stage):
    n = 1
    while os.path.exists(os.path.join(HERE, f"{stage}-run{n}.json")):
        n += 1
    return os.path.join(HERE, f"{stage}-run{n}.json")


def run(cmd, cwd, timeout):
    t0 = time.time()
    try:
        p = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=timeout, shell=(os.name == "nt"))
        return p.returncode, p.stdout, p.stderr, round(time.time() - t0, 2), False
    except subprocess.TimeoutExpired as e:
        return None, (e.stdout or ""), (e.stderr or ""), round(time.time() - t0, 2), True


def tail(s, n=40):
    return "\n".join((s or "").splitlines()[-n:])


def vitest(tests, timeout=600):
    out_json = os.path.join(tempfile.gettempdir(), "r088-round1-vitest.json")
    if os.path.exists(out_json):
        os.remove(out_json)
    cmd = ["npm", "--prefix", "frontend", "test", "--"] + tests + ["--reporter=json", f"--outputFile={out_json}"]
    code, so, se, secs, timed_out = run(cmd, PROJECT, timeout)
    summary = {"command": "npm --prefix frontend test -- " + " ".join(tests) + " --reporter=json --outputFile=<tmp>", "exitCode": code,
               "seconds": secs, "timedOut": timed_out, "jsonProduced": os.path.exists(out_json)}
    if os.path.exists(out_json):
        j = json.load(open(out_json, encoding="utf-8"))
        cases = []
        for f in j.get("testResults", []):
            for a in f.get("assertionResults", []):
                cases.append({"file": os.path.basename(f.get("name", "")), "title": a.get("fullName") or a.get("title"), "status": a.get("status"),
                              "firstFailure": ((a.get("failureMessages") or [""])[0].splitlines() or [""])[0][:300] if a.get("status") == "failed" else None})
        summary.update(total=j.get("numTotalTests"), passed=j.get("numPassedTests"), failed=j.get("numFailedTests"), skippedOrPending=j.get("numPendingTests"),
                       todo=j.get("numTodoTests"), cases=cases)
    summary["stdoutTail"] = tail(so)
    summary["stderrTail"] = tail(se)
    return summary


def main():
    stage = sys.argv[1]
    args = sys.argv[2:]
    rec = {"schemaVersion": "r088.round1Run@1", "stage": stage, "recordedAt": datetime.datetime.now().astimezone().isoformat(timespec="seconds"),
           "executionHead": head(), "providerCalls": 0, "backendRuns": 0, "networkUse": "none"}
    if stage == "oldsource":
        tests = args or [NEW_TESTS[0]]
        rec["sourceMessagesVueLf"] = lf_sha("frontend/src/views/Messages.vue")
        rec.update(vitest(tests))
    elif stage == "tests":
        rec.update(vitest(args or (NEW_TESTS + EXISTING)))
    elif stage == "typecheck":
        code, so, se, secs, to = run(["npx", "--no-install", "vue-tsc", "-b", "--force"], FRONTEND, 900)
        rec.update(command="npx --no-install vue-tsc -b --force (cwd frontend)", cwd="frontend", exitCode=code, seconds=secs, timedOut=to, stdoutTail=tail(so), stderrTail=tail(se))
    elif stage == "build":
        code, so, se, secs, to = run(["npm", "--prefix", "frontend", "run", "build"], PROJECT, 900)
        rec.update(command="npm --prefix frontend run build", exitCode=code, seconds=secs, timedOut=to, stdoutTail=tail(so), stderrTail=tail(se))
    elif stage == "hashes":
        rec["lfSha256"] = {p: lf_sha(p) for p in TRACKED}
    else:
        raise SystemExit("unknown stage")
    path = next_receipt(stage)
    json.dump(rec, open(path, "w", encoding="utf-8", newline="\n"), indent=2, ensure_ascii=False)
    print(os.path.basename(path), "exit", rec.get("exitCode"), {k: rec.get(k) for k in ("total", "passed", "failed", "skippedOrPending") if k in rec})


main()
