#!/usr/bin/env python3
"""GOV-002 round1 candidate-bound proof driver (R1-F1, R1-F2).

Static repository checks only: no provider, Docker, SQL, Go, application, customer data or secret.
It never commits in, resets, or writes outside the live worktree except for its own `--out` stage files.
All mutation, tamper, drift and range negatives run in disposable git clones under `--work`.
The only network use is the declared fresh-clone step (public project snapshot -> `git clone` of a LOCAL
candidate repo, and the initializer's clone/fetch of the public CVF Core).

Stages (run in this order; each writes `<out>/stage-<name>.json`):
  buildtime      live-tree runner phases with --packet while the work order is ACTIVE; live installer rerun
  discrimination new compatibility tests against the preserved returned adapter vs the repaired gate
  snapshot       disposable candidate repo = HEAD + live change set (minus declared evidence outputs)
  clone          separate-root clone, initializer/doctor, preflight, unit tests, phases, installer idempotence
  negatives      tamper / drift / role / range / status negatives in disposable clones
  prci           explicit-range pr-ci in the candidate repo with all three pins, plus range negatives
  final          live-tree final-state gates (run after the return document exists)

Usage: python -B gov002_round1_proof.py --project <root> --work <empty-or-reused-dir> --out <dir> --stage <name>
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

CORE_COMMIT = "e9ddcbcdcac2a9df520f5eb0761dbdb609b9deba"
WORK_ORDER = "docs/work_orders/CCMAI_GOV_002.md"
REVIEW = "docs/reviews/GOV002_PORTABLE_INHERITANCE_INDEPENDENT_REVIEW_2026-10-12.md"
RETURN_DOC = "docs/reviews/GOV002_PORTABLE_INHERITANCE_WORKER_RETURN_2026-10-12.md"
ROUND1_RECEIPT = "docs/reviews/probes/gov002_round1_worker_receipt_2026-10-12.json"
MANIFEST_OUT = "docs/reviews/probes/gov002_round1_candidate_manifest_2026-10-12.json"
GATES = "scripts/cvf_gates/cvf_downstream_gate_runner.py"
EXCLUDED_PREFIXES = ("docs/reviews/probes/gov002_review_round1/proof/",)
PROOF_DRIVER = "docs/reviews/probes/gov002_review_round1/gov002_round1_proof.py"
EXCLUDED_FILES = {RETURN_DOC, ROUND1_RECEIPT, MANIFEST_OUT, PROOF_DRIVER}
MAX_OUTPUT = 14000
# Paths another actor added to the shared worktree while this repair ran; recorded, never edited, staged or counted as candidate.
OTHER_ACTOR_PATHS = {
    "docs/reviews/planning/CCMA_RUNTIME_CVF_INHERITANCE_ROADMAP_2026-10-12.md":
        "Codex /root planning draft (Status DRAFT, HOLD) that appeared at 2026-10-12T19:43+07:00 after the candidate snapshot; "
        "not authored, edited or staged by the worker and not part of the GOV002 candidate",
}


def utc_now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat(timespec="seconds")


def rmtree(path):
    """Remove a directory tree including read-only git objects (Windows)."""
    def onerror(func, target, _exc):
        try:
            os.chmod(target, stat.S_IWRITE)
            func(target)
        except OSError:
            pass
    if os.path.exists(path):
        shutil.rmtree(path, onerror=onerror)


def lf_sha(data):
    return hashlib.sha256(data.replace(b"\r\n", b"\n")).hexdigest()


class Ctx:
    def __init__(self, project, work, out):
        self.project = os.path.abspath(project)
        self.work = os.path.abspath(work)
        self.out = os.path.abspath(out)
        self.records = []
        os.makedirs(self.work, exist_ok=True)
        os.makedirs(self.out, exist_ok=True)
        base = [self.work, self.project, os.path.dirname(self.project)]
        self._subs = []
        for token, path in zip(("<WORK>", "<PROJECT>", "<WORKSPACE>"), base):
            for variant in {path, path.replace("\\", "/"), path.replace("/", "\\")}:
                self._subs.append((variant, token))
        self._subs.sort(key=lambda item: -len(item[0]))

    def clean(self, text):
        for variant, token in self._subs:
            text = text.replace(variant, token)
        return text

    def run(self, rid, purpose, cmd, cwd, expect=None, expect_text=None, forbid_text=None, env=None, timeout=1200):
        """Run a command, record sanitized evidence. expect: None (any), int exit, or 'nonzero'."""
        full_env = dict(os.environ)
        full_env["PYTHONDONTWRITEBYTECODE"] = "1"
        if env:
            full_env.update(env)
        start = time.time()
        try:
            proc = subprocess.run(cmd, cwd=cwd, env=full_env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                  timeout=timeout)
            code, raw = proc.returncode, proc.stdout
        except subprocess.TimeoutExpired as exc:
            code, raw = -9, (exc.stdout or b"") + b"\nTIMEOUT"
        text = self.clean(raw.decode("utf-8", errors="replace").replace("\r\n", "\n"))
        ok = True
        if expect == "nonzero":
            ok = code != 0
        elif isinstance(expect, int):
            ok = code == expect
        for needle in ([expect_text] if isinstance(expect_text, str) else (expect_text or [])):
            ok = ok and needle in text
        for needle in ([forbid_text] if isinstance(forbid_text, str) else (forbid_text or [])):
            ok = ok and needle not in text
        shown = text if len(text) <= MAX_OUTPUT else text[: MAX_OUTPUT // 2] + "\n...[truncated]...\n" + text[-MAX_OUTPUT // 2:]
        rec = {
            "id": rid, "purpose": purpose,
            "command": self.clean(" ".join(cmd)), "cwd": self.clean(cwd),
            "exitCode": code, "expected": expect if expect is not None else "any",
            "expectedText": expect_text, "forbiddenText": forbid_text,
            "outcome": "AS_EXPECTED" if ok else "UNEXPECTED",
            "durationSeconds": round(time.time() - start, 2),
            "outputSha256": hashlib.sha256(text.encode("utf-8")).hexdigest(),
            "outputTruncated": len(text) > MAX_OUTPUT, "output": shown,
        }
        self.records.append(rec)
        print(f"[{rec['outcome']}] {rid} exit={code}")
        return rec, text

    def add(self, rid, purpose, ok, detail):
        rec = {"id": rid, "purpose": purpose, "outcome": "AS_EXPECTED" if ok else "UNEXPECTED", "detail": detail}
        self.records.append(rec)
        print(f"[{rec['outcome']}] {rid}")
        return ok

    def write_stage(self, name, extra=None):
        body = {
            "schemaVersion": "gov002.round1.proofStage@1", "stage": name, "recordedAtUtc": utc_now(),
            "pythonVersion": sys.version.split()[0], "gitVersion": self._tool(["git", "--version"]),
            "powershellVersion": self._tool(["pwsh", "-NoProfile", "-Command", "$PSVersionTable.PSVersion.ToString()"]),
            "pathPlaceholders": {"<PROJECT>": "live worktree", "<WORK>": "disposable proof work directory",
                                 "<WORKSPACE>": "container directory that holds the project and the sibling Core"},
            "claimBoundary": "static repository checks in the stated trees only; no provider, runtime, hosted CI or deployment evidence",
            "unexpectedCount": sum(1 for r in self.records if r["outcome"] != "AS_EXPECTED"),
            "recordCount": len(self.records), "records": self.records,
        }
        if extra:
            body.update(extra)
        path = os.path.join(self.out, f"stage-{name}.json")
        with open(path, "w", encoding="utf-8", newline="\n") as fh:
            json.dump(body, fh, indent=2, ensure_ascii=False)
            fh.write("\n")
        print(f"stage {name}: {len(self.records)} records, unexpected={body['unexpectedCount']} -> {path}")
        return body["unexpectedCount"]

    @staticmethod
    def _tool(cmd):
        try:
            return subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=30).stdout.decode().strip().splitlines()[0]
        except (OSError, subprocess.SubprocessError, IndexError):
            return None


def git(args, cwd, check=True):
    proc = subprocess.run(["git"] + args, cwd=cwd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    if check and proc.returncode:
        raise RuntimeError(f"git {' '.join(args)} failed: {proc.stderr.decode(errors='replace')}")
    return proc.stdout.decode("utf-8", errors="replace")


def excluded(path):
    return path in EXCLUDED_FILES or path.startswith(EXCLUDED_PREFIXES)


def change_set(project):
    """(present, deleted) paths = modified or untracked, not ignored, relative to HEAD."""
    listed = [p for p in git(["ls-files", "-m", "-o", "--exclude-standard", "-z"], project).split("\0") if p]
    deleted = [p for p in git(["ls-files", "-d", "-z"], project).split("\0") if p]
    present = sorted({p for p in listed if os.path.isfile(os.path.join(project, p))})
    return present, sorted(set(deleted))


def tree_hash(root):
    """Digest over every tracked or untracked-not-ignored file (LF-normalized), excluding the proof outputs."""
    files = sorted(set(p for p in git(["ls-files", "-c", "-o", "--exclude-standard", "-z"], root).split("\0") if p))
    h = hashlib.sha256()
    count = 0
    for rel in files:
        if excluded(rel):
            continue
        full = os.path.join(root, rel)
        if os.path.isfile(full):
            with open(full, "rb") as fh:
                h.update(f"{rel}:{lf_sha(fh.read())}\n".encode("utf-8"))
            count += 1
    return h.hexdigest(), count


def py(*args):
    return [sys.executable, "-B"] + list(args)


def phase_cmd(phase, packets=(), extra=()):
    cmd = py(GATES, "run", "--phase", phase, "--project-root", ".", "--no-receipt")
    for packet in packets:
        cmd += ["--packet", packet]
    return cmd + list(extra)


def pins_from_files(root):
    """Three pins as read from the generated workflow, cross-checked against the lock."""
    with open(os.path.join(root, ".github/workflows/cvf-downstream-gates.yml"), encoding="utf-8") as fh:
        wf = fh.read()
    with open(os.path.join(root, ".cvf/gate-profile.lock.json"), encoding="utf-8") as fh:
        lock = json.load(fh)
    found = {k: re.search(rf"--expect-{k}-sha256 ([0-9a-f]{{64}})", wf).group(1) for k in ("profile", "runner", "bundle")}
    agree = (found["profile"] == lock["profileSha256"] and found["runner"] == lock["runnerSha256"]
             and found["bundle"] == lock["bundleSha256"])
    return found, agree


def expect_flags(pins):
    return ["--expect-profile-sha256", pins["profile"], "--expect-runner-sha256", pins["runner"],
            "--expect-bundle-sha256", pins["bundle"]]


# ----------------------------------------------------------------------------------------- stages
def stage_buildtime(c):
    pkts = [WORK_ORDER, REVIEW, RETURN_DOC]
    status = open(os.path.join(c.project, WORK_ORDER), encoding="utf-8").read()
    m = re.search(r"(?m)^Status:\s*(\S+)", status)
    c.add("BT-0", "work order status when the BUILD-time phases run (must be ACTIVE for contract coverage)",
          bool(m) and m.group(1) == "ACTIVE", {"workOrderStatus": m.group(1) if m else None})
    before = tree_hash(c.project)
    for phase in ("pre-dispatch", "worker-return", "reviewer-fast", "pre-commit"):
        _, text = c.run(f"BT-{phase}", f"live {phase} with explicit --packet while Status is ACTIVE", phase_cmd(phase, pkts),
                        c.project, expect=0, expect_text=["RESULT: PASS"])
        c.add(f"BT-{phase}-coverage", "APPL-01 must be a CHECK that names the work order, not a no-candidate N/A",
              "[PASS] CVF-DG-APPL-01 PASS" in text and f"checked: {WORK_ORDER}" in text
              and "NOT_APPLICABLE_WITH_REASON - no candidate" not in text, {"phase": phase})
    core = os.path.join(os.path.dirname(c.project), ".Controlled-Vibe-Framework-CVF")
    c.run("BT-install-rerun", "explicit installer rerun in the live worktree (expected command)",
          py(os.path.join(core, "scripts/lib/downstream_governance/cvf_downstream_gate_runner.py"), "install",
             "--project-root", ".", "--core-root", core, "--core-commit", CORE_COMMIT),
          c.project, expect=0, expect_text=["ALREADY_INSTALLED"])
    after = tree_hash(c.project)
    c.add("BT-install-rerun-bytes", "installer rerun changed no non-ignored file bytes (live worktree)",
          before == after, {"beforeTreeSha256": before[0], "afterTreeSha256": after[0], "files": after[1]})
    return c.write_stage("buildtime")


def stage_discrimination(c):
    root = os.path.join(c.work, "discrimination")
    rmtree(root)
    returned = os.path.join(c.project, "docs/reviews/probes/gov002_review_round1/original-returned-cvf_downstream_gate.py.txt")
    inputs = ["scripts/cvf_downstream_gate.py", "scripts/tests/test_cvf_downstream_gate_profile_compatibility.py",
              ".cvf/gate-profile.lock.json", "docs/reviews/probes/gov002_review_round1/original-returned-cvf_downstream_gate.py.txt"]
    for label, swap in (("returned", True), ("repaired", False)):
        copy = os.path.join(root, label)
        shutil.copytree(os.path.join(c.project, "scripts/cvf_gates"), os.path.join(copy, "scripts/cvf_gates"))
        for rel in inputs:
            os.makedirs(os.path.dirname(os.path.join(copy, rel)), exist_ok=True)
            shutil.copyfile(os.path.join(c.project, rel), os.path.join(copy, rel))
        if swap:
            shutil.copyfile(returned, os.path.join(copy, "scripts/cvf_downstream_gate.py"))
        digest = hashlib.sha256(open(os.path.join(copy, "scripts/cvf_downstream_gate.py"), "rb").read()).hexdigest()
        rec, text = c.run(
            f"DISC-{label}", f"new compatibility tests against the {label} adapter in an isolated copy",
            py("-m", "unittest", "discover", "-s", "scripts/tests", "-p", "test_cvf_downstream_gate_profile_compatibility.py"),
            copy, expect="nonzero" if swap else 0, expect_text=["FAILED"] if swap else ["OK"])
        failing = sorted(set(re.findall(r"^(?:FAIL|ERROR): (\S+) \(([^)]*)\)", text, re.M)))
        rec["adapterRawSha256"] = digest
        rec["failingTests"] = [f"{n} ({k})" for n, k in failing]
        rec["failingTestCount"] = len(failing)
    return c.write_stage("discrimination")


def stage_snapshot(c):
    cand = os.path.join(c.work, "cand_src")
    rmtree(cand)
    git(["clone", "-c", "core.longpaths=true", "--quiet", "--no-hardlinks", c.project, cand], c.work)
    base = git(["rev-parse", "HEAD"], cand).strip()
    present, deleted = change_set(c.project)
    copied, skipped = [], []
    for rel in present:
        if excluded(rel):
            skipped.append(rel)
            continue
        dest = os.path.join(cand, rel)
        os.makedirs(os.path.dirname(dest), exist_ok=True)
        shutil.copyfile(os.path.join(c.project, rel), dest)
        copied.append(rel)
    for rel in deleted:
        if os.path.exists(os.path.join(cand, rel)):
            os.remove(os.path.join(cand, rel))
    # catalog: the snapshot cannot register paths that the declared evidence outputs add later
    reg_path = os.path.join(cand, "docs/catalog/ARTIFACT_REGISTRY.json")
    with open(reg_path, "rb") as fh:
        raw = fh.read()
    reg = json.loads(raw.decode("utf-8"))
    kept = [a for a in reg["artifacts"] if not excluded(a["path"]) and os.path.exists(os.path.join(cand, a["path"]))]
    dropped = [a["path"] for a in reg["artifacts"] if a not in kept]
    reg["artifacts"] = kept
    with open(reg_path, "wb") as fh:
        fh.write((json.dumps(reg, indent=2, ensure_ascii=False) + "\n").replace("\n", "\r\n").encode("utf-8"))
    shell = shutil.which("pwsh") or shutil.which("powershell")
    c.run("SNAP-catalog", "regenerate catalog views in the candidate repo from the filtered registry",
          [shell, "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "scripts/manage_cvf_downstream_catalog.ps1", "-Write"],
          cand, expect=0)
    git(["add", "-A"], cand)
    git(["-c", "user.name=gov002-proof", "-c", "user.email=gov002-proof@invalid", "commit", "--quiet", "-m",
         "disposable GOV002 round1 candidate snapshot (not a project commit)"], cand)
    commit = git(["rev-parse", "HEAD"], cand).strip()
    tree = git(["rev-parse", "HEAD^{tree}"], cand).strip()
    # faithfulness: every candidate path equals the live file (LF-normalized), nothing omitted
    mismatched, manifest = [], {}
    for rel in copied:
        a = lf_sha(open(os.path.join(c.project, rel), "rb").read())
        b = git(["show", f"HEAD:{rel}"], cand)
        manifest[rel] = a
        if hashlib.sha256(b.encode("utf-8")).hexdigest() != a and rel not in ("docs/INDEX.md", "docs/catalog/MODULE_CATALOG.md",
                                                                          "docs/catalog/ARTIFACT_REGISTRY.json"):
            mismatched.append(rel)
    snap_changed = [p for p in git(["diff", "--name-only", base, commit], cand).split("\n") if p]
    omitted = sorted(set(copied) - set(snap_changed))
    unexpected = sorted(set(snap_changed) - set(copied) - set(deleted)
                        - {"docs/INDEX.md", "docs/catalog/MODULE_CATALOG.md", "docs/catalog/ARTIFACT_REGISTRY.json"})
    c.add("SNAP-faithful", "candidate repo carries every live change except the declared evidence outputs",
          not mismatched and not omitted and not unexpected,
          {"baseCommit": base, "candidateCommit": commit, "candidateTree": tree, "copiedPaths": len(copied),
           "deletedPaths": deleted, "excludedEvidenceOutputs": skipped, "catalogRegistryEntriesDroppedInSnapshotOnly": dropped,
           "catalogViewsRegenerated": ["docs/INDEX.md", "docs/catalog/MODULE_CATALOG.md", "docs/catalog/ARTIFACT_REGISTRY.json"],
           "contentMismatches": mismatched, "omittedPaths": omitted, "unexpectedPaths": unexpected})
    ignored = git(["status", "--porcelain", "--ignored"], cand)
    c.add("SNAP-no-local-state", "binding, runtime receipts and bytecode are not part of the candidate",
          not any(x in git(["ls-files"], cand) for x in (".cvf/local-binding.json", ".cvf/runtime/", "__pycache__")),
          {"statusPorcelainIgnored": ignored.strip()[:300]})
    result = c.write_stage("snapshot", {"candidateCommit": commit, "candidateTree": tree, "baseCommit": base,
                                         "candidateManifestLfSha256": dict(sorted(manifest.items()))})
    with open(os.path.join(c.work, "snapshot.json"), "w", encoding="utf-8") as fh:
        json.dump({"cand": cand, "commit": commit, "tree": tree, "base": base, "copied": copied, "excluded": skipped}, fh)
    return result


def load_snapshot(c):
    with open(os.path.join(c.work, "snapshot.json"), encoding="utf-8") as fh:
        return json.load(fh)


def stage_clone(c):
    snap = load_snapshot(c)
    container = os.path.join(c.work, "container")
    rmtree(container)
    os.makedirs(container)
    clone = os.path.join(container, "Customer-Care-Monitor")
    c.run("CL-clone", "git clone of the candidate snapshot into a different absolute container directory",
          ["git", "clone", "-c", "core.longpaths=true", "--quiet", "--no-hardlinks", snap["cand"], clone], container, expect=0)
    head = git(["rev-parse", "HEAD"], clone).strip()
    tree = git(["rev-parse", "HEAD^{tree}"], clone).strip()
    c.add("CL-identity", "clone HEAD/tree equal the recorded candidate snapshot", head == snap["commit"] and tree == snap["tree"],
          {"cloneHead": head, "cloneTree": tree, "originalProjectPathAbsentFromCloneRoot": c.project != clone})
    shell = shutil.which("powershell") or shutil.which("pwsh")
    rec, _ = c.run("CL-init", "scripts/initialize_cvf_clone.ps1 in the clone (clones/pins public Core, writes ignored binding, runs doctor)",
                   [shell, "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "scripts/initialize_cvf_clone.ps1"],
                   clone, expect=0, expect_text=["FRESH_CLONE_CONTINUITY_PASS"], timeout=1800)
    core = os.path.join(container, ".Controlled-Vibe-Framework-CVF")
    core_head = git(["rev-parse", "HEAD"], core).strip() if os.path.isdir(core) else None
    c.add("CL-core-identity", "sibling Core in the container is the pinned public commit and clean",
          core_head == CORE_COMMIT and not git(["status", "--porcelain"], core).strip(),
          {"coreHead": core_head, "coreRemote": git(["config", "--get", "remote.origin.url"], core).strip()})
    st = git(["status", "--porcelain", "--ignored"], clone)
    c.add("CL-binding-ignored", "local binding exists but is ignored; the clone tree stays clean",
          os.path.isfile(os.path.join(clone, ".cvf/local-binding.json")) and "!! .cvf/local-binding.json" in st
          and not [l for l in st.splitlines() if not l.startswith("!!")], {"porcelainIgnored": st.strip()[:600]})
    binding = json.load(open(os.path.join(clone, ".cvf/local-binding.json"), encoding="utf-8-sig"))
    c.add("CL-binding-paths", "binding records the clone's own paths (not the original machine)",
          os.path.normcase(binding.get("projectPath", "")) == os.path.normcase(clone), {"projectPath": c.clean(binding.get("projectPath", ""))})
    carriers = ["AGENTS.md", "README.md", "docs/reference/CVF_PROJECT_PORTABLE_CONTINUATION.md", ".cvf/manifest.json",
                "scripts/initialize_cvf_clone.ps1", ".github/workflows/cvf-downstream-gates.yml", ".cvf/gate-profile.lock.json"]
    hits = {}
    for rel in carriers + [p for p in git(["ls-files", "scripts/cvf_gates"], clone).split("\n") if p]:
        text = open(os.path.join(clone, rel), encoding="utf-8-sig", errors="replace").read()
        found = [needle for needle in ("UNG DUNG AI", "TOOL AI 2026", "Users\\DELL", "Users/DELL", "CVF-Workspace\\") if needle in text]
        if found:
            hits[rel] = found
    c.add("CL-no-origin-paths", "tracked instruction carriers contain no original-machine absolute path", not hits, {"matches": hits})
    c.run("CL-preflight", "legacy seven-gate preflight in the clone", py("scripts/cvf_downstream_gate.py", "preflight"), clone,
          expect=0, expect_text=["PASS (7/7 executed gates passed)"])
    c.run("CL-unittest-all", "all gate self-tests once in the clone (46 legacy + new compatibility)",
          py("-m", "unittest", "discover", "-s", "scripts/tests", "-p", "test_cvf_downstream_gate*.py"), clone,
          expect=0, expect_text=["OK"], timeout=1800)
    for phase in ("pre-dispatch", "worker-return", "reviewer-fast", "pre-commit"):
        c.run(f"CL-{phase}", f"installed {phase} in the clone with explicit work-order packet (status is not ACTIVE: APPL N/A expected)",
              phase_cmd(phase, [WORK_ORDER]), clone, expect=0, expect_text=["RESULT: PASS"])
    before = tree_hash(clone)
    status_before = git(["status", "--porcelain"], clone)
    c.run("CL-install-rerun", "installer rerun from the pinned Core in the clone",
          py(os.path.join(core, "scripts/lib/downstream_governance/cvf_downstream_gate_runner.py"), "install", "--project-root", ".",
             "--core-root", core, "--core-commit", CORE_COMMIT), clone, expect=0, expect_text=["ALREADY_INSTALLED"])
    after = tree_hash(clone)
    c.add("CL-install-idempotent", "installer rerun changed no tracked/untracked bytes and left git clean",
          before == after and git(["status", "--porcelain"], clone) == status_before == "",
          {"beforeTreeSha256": before[0], "afterTreeSha256": after[0], "files": after[1]})
    c.run("CL-trusted", "Core runner in --trusted mode compares every installed byte to Core sources",
          py(os.path.join(core, "scripts/lib/downstream_governance/cvf_downstream_gate_runner.py"), "run", "--phase", "pre-dispatch",
             "--project-root", clone, "--trusted", "--no-receipt"), clone, expect=0, expect_text=["RESULT: PASS"])
    # refusals: dirty / unrelated Core must not be overwritten and the initializer must stop
    sentinel = os.path.join(core, "GOV002_DIRTY_SENTINEL.txt")
    with open(sentinel, "w", encoding="utf-8") as fh:
        fh.write("local work that must survive\n")
    c.run("CL-refuse-dirty-core", "initializer refuses a dirty Core and does not reset it", [shell, "-NoProfile", "-ExecutionPolicy", "Bypass",
          "-File", "scripts/initialize_cvf_clone.ps1"], clone, expect="nonzero", expect_text=["dirty"])
    c.add("CL-dirty-core-preserved", "dirty file and Core HEAD survived the refusal", os.path.isfile(sentinel)
          and git(["rev-parse", "HEAD"], core).strip() == CORE_COMMIT, {"sentinel": os.path.isfile(sentinel)})
    os.remove(sentinel)
    other = os.path.join(c.work, "container_unrelated")
    rmtree(other)
    os.makedirs(os.path.join(other, ".Controlled-Vibe-Framework-CVF"))
    unrelated = os.path.join(other, ".Controlled-Vibe-Framework-CVF")
    git(["init", "--quiet"], unrelated)
    with open(os.path.join(unrelated, "unrelated.txt"), "w", encoding="utf-8") as fh:
        fh.write("not CVF\n")
    git(["add", "-A"], unrelated)
    git(["-c", "user.name=x", "-c", "user.email=x@invalid", "commit", "--quiet", "-m", "unrelated"], unrelated)
    uhead = git(["rev-parse", "HEAD"], unrelated).strip()
    clone2 = os.path.join(other, "Customer-Care-Monitor")
    git(["clone", "-c", "core.longpaths=true", "--quiet", "--no-hardlinks", snap["cand"], clone2], other)
    c.run("CL-refuse-unrelated-core", "initializer refuses an unrelated Core (no origin/main history) and does not rewrite it",
          [shell, "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "scripts/initialize_cvf_clone.ps1"], clone2, expect="nonzero")
    c.add("CL-unrelated-core-preserved", "unrelated Core HEAD unchanged after the refusal",
          git(["rev-parse", "HEAD"], unrelated).strip() == uhead and os.path.isfile(os.path.join(unrelated, "unrelated.txt")), {"head": uhead})
    return c.write_stage("clone", {"cloneHead": head, "cloneTree": tree, "candidateCommit": snap["commit"]})


def fresh_copy(c, label, clone):
    dest = os.path.join(c.work, "neg", label)
    rmtree(dest)
    os.makedirs(os.path.dirname(dest), exist_ok=True)
    git(["clone", "-c", "core.longpaths=true", "--quiet", "--no-hardlinks", clone, dest], c.work)
    return dest


def edit(path, fn):
    """Apply a text mutation on LF-normalized content and write it back with CRLF (the clone's checkout style)."""
    with open(path, "rb") as fh:
        data = fh.read().decode("utf-8").replace("\r\n", "\n")
    data = fn(data)
    with open(path, "wb") as fh:
        fh.write(data.replace("\n", "\r\n").encode("utf-8"))


def stage_negatives(c):
    container = os.path.join(c.work, "container")
    clone = os.path.join(container, "Customer-Care-Monitor")
    core = os.path.join(container, ".Controlled-Vibe-Framework-CVF")
    pins, agree = pins_from_files(clone)
    c.add("NEG-pins-agree", "workflow pins equal the lock pins in the candidate", agree, pins)
    # N00 positive control: unmutated copy passes with all three pins
    n = fresh_copy(c, "n00_control", clone)
    c.run("N00", "control: unmutated copy passes pre-dispatch with the three pins",
          phase_cmd("pre-dispatch", [WORK_ORDER], expect_flags(pins)), n, expect=0, expect_text=["RESULT: PASS"])
    n = fresh_copy(c, "n01_module_tamper", clone)
    flag = os.path.join(n, "IMPORT_EXECUTED.flag")
    edit(os.path.join(n, "scripts/cvf_gates/cvf_dg_continuity.py"), lambda t: t + "\nopen('IMPORT_EXECUTED.flag', 'w').write('x')\n")
    c.run("N01", "copied inherited module tampered: the pinned (CI-style) run is refused at stage 0 before any module import",
          phase_cmd("pre-dispatch", [WORK_ORDER], expect_flags(pins)), n, expect="nonzero", expect_text=["stage-0", "REFUSED"])
    c.add("N01-not-executed", "tampered module code never ran in the pinned run", not os.path.exists(flag), {"flagPresent": os.path.exists(flag)})
    c.run("N01b", "same tamper without pins (plain local invocation): detected by INST-01 FILE_TAMPERED",
          phase_cmd("pre-dispatch", [WORK_ORDER]), n, expect="nonzero", expect_text=["FILE_TAMPERED"])
    c.add("N01b-unpinned-disclosure", "DISCLOSED LIMIT of the Core design: an unpinned local run imports project modules before verify_install, "
          "so tampered code can execute once before the failure is reported; the pinned CI run and the Core --trusted doctor refuse first",
          True, {"tamperedCodeExecutedInUnpinnedRun": os.path.exists(flag)})
    n = fresh_copy(c, "n02_runner_tamper", clone)
    edit(os.path.join(n, "scripts/cvf_gates/cvf_downstream_gate_runner.py"), lambda t: t + "\n# tamper\n")
    c.run("N02", "runner tampered: the CI pin on the runner digest refuses it", phase_cmd("pre-dispatch", [WORK_ORDER], expect_flags(pins)),
          n, expect="nonzero", expect_text=["REFUSED"])
    n = fresh_copy(c, "n03_lock_tamper", clone)
    def lock_edit(t):
        d = json.loads(t)
        d["files"]["cvf_dg_review.py"] = "0" * 64
        return json.dumps(d, indent=2)
    edit(os.path.join(n, ".cvf/gate-profile.lock.json"), lock_edit)
    c.run("N03", "lock hash for a module altered", phase_cmd("pre-dispatch", [WORK_ORDER]), n, expect="nonzero")
    n = fresh_copy(c, "n04_workflow_pin", clone)
    edit(os.path.join(n, ".github/workflows/cvf-downstream-gates.yml"), lambda t: t.replace(pins["bundle"], "0" * 64))
    _, t04 = c.run("N04", "workflow bundle pin altered so it disagrees with the lock",
                   phase_cmd("pre-dispatch", [WORK_ORDER]), n, expect="nonzero")
    n = fresh_copy(c, "n05_consistent_tamper", clone)
    sys.path.insert(0, os.path.join(n, "scripts/cvf_gates"))
    import importlib
    common = importlib.import_module("cvf_dg_common")
    edit(os.path.join(n, "scripts/cvf_gates/cvf_dg_review.py"), lambda t: t + "\n# consistent tamper\n")
    lock_path = os.path.join(n, ".cvf/gate-profile.lock.json")
    lock = json.load(open(lock_path, encoding="utf-8"))
    gates = os.path.join(n, "scripts/cvf_gates")
    lock["files"]["cvf_dg_review.py"] = common.file_content_sha256(__import__("pathlib").Path(gates) / "cvf_dg_review.py")
    lock["bundleSha256"] = common.bundle_digest(__import__("pathlib").Path(gates))[0]
    open(lock_path, "wb").write(common.canonical_json(lock).encode("utf-8"))
    sys.path.remove(os.path.join(n, "scripts/cvf_gates"))
    sys.modules.pop("cvf_dg_common", None)
    c.run("N05a", "module AND lock edited consistently, workflow pins untouched: CI-pinned run is refused",
          phase_cmd("pre-dispatch", [WORK_ORDER], expect_flags(pins)), n, expect="nonzero")
    c.run("N05b", "same consistent tamper: the Core --trusted run (independent anchor) refuses it",
          py(os.path.join(core, "scripts/lib/downstream_governance/cvf_downstream_gate_runner.py"), "run", "--phase", "pre-dispatch",
             "--project-root", n, "--trusted", "--no-receipt"), n, expect="nonzero")
    n = fresh_copy(c, "n06_wrong_pin_arg", clone)
    wrong = dict(pins, bundle="1" * 64)
    c.run("N06", "wrong --expect-bundle-sha256", phase_cmd("pre-dispatch", [WORK_ORDER], expect_flags(wrong)), n,
          expect="nonzero", expect_text=["REFUSED"])
    cont = {
        "n07_mode_drift": ("one surface drift: handoff Current mode differs from state", "SURFACE_MISMATCH"),
    }
    n = fresh_copy(c, "n07_mode_drift", clone)
    handoff = [p for p in os.listdir(os.path.join(n, "CVF_SESSION/handoffs")) if "PORTABLE_CVF_GOV002" in p][0]
    edit(os.path.join(n, "CVF_SESSION/handoffs", handoff), lambda t: re.sub(r"(?m)^- Current mode: \S+", "- Current mode: BUILD", t, count=1))
    c.run("N07", cont["n07_mode_drift"][0], phase_cmd("pre-dispatch"), n, expect="nonzero", expect_text=["SURFACE_MISMATCH"])
    n = fresh_copy(c, "n08_unknown_field", clone)
    def add_field(t):
        d = json.loads(t)
        d["unknownStandardField"] = 1
        return json.dumps(d, indent=2)
    edit(os.path.join(n, "CVF_SESSION/ACTIVE_SESSION_STATE.json"), add_field)
    c.run("N08", "unknown standard state field rejected (extensions need x- prefix)", phase_cmd("pre-dispatch"), n,
          expect="nonzero", expect_text=["UNKNOWN_FIELD"])
    n = fresh_copy(c, "n09_missing_active_wo", clone)
    def drop_field(t):
        d = json.loads(t)
        del d["activeWorkOrders"]
        return json.dumps(d, indent=2)
    edit(os.path.join(n, "IMPLEMENTATION_STATUS.json"), drop_field)
    c.run("N09", "implementation status without required activeWorkOrders", phase_cmd("pre-dispatch"), n,
          expect="nonzero", expect_text=["MISSING_FIELD"])
    n = fresh_copy(c, "n10_waiting_review_executable", clone)
    def list_wo(t):
        d = json.loads(t)
        d["activeWorkOrders"] = [WORK_ORDER]
        return json.dumps(d, indent=2)
    edit(os.path.join(n, "IMPLEMENTATION_STATUS.json"), list_wo)
    c.run("N10", "a waiting-review order listed as executable is rejected", phase_cmd("pre-dispatch"), n,
          expect="nonzero", expect_text=["ACTIVE_WORK_ORDER_NOT_ACTIVE"])
    n = fresh_copy(c, "n11_self_review", clone)
    review = os.path.join(n, "docs/reviews/GOV002_NEG_SELF_REVIEW.md")
    with open(review, "w", encoding="utf-8", newline="\n") as fh:
        fh.write("# Self review fixture\n\nRisk: R2\nWorker: same-agent\nReviewer: same-agent\n")
    c.run("N11", "R2 self-review rejected by ROLE-01", phase_cmd("worker-return", [WORK_ORDER, "docs/reviews/GOV002_NEG_SELF_REVIEW.md"]),
          n, expect="nonzero", expect_text=["SELF_REVIEW"])
    n = fresh_copy(c, "n12_active_without_contract", clone)
    def active_no_contract(t):
        t = re.sub(r"(?m)^Status:.*$", "Status: DISPATCH_READY", t, count=1)
        return t.replace("## Gate-To-Role Closeability Contract", "## Removed Contract Heading")
    edit(os.path.join(n, WORK_ORDER), active_no_contract)
    c.run("N12", "an ACTIVE-class order without its closeability contract fails APPL-01", phase_cmd("pre-dispatch", [WORK_ORDER]),
          n, expect="nonzero", expect_text=["contract_missing"])
    n = fresh_copy(c, "n13_active_with_contract", clone)
    edit(os.path.join(n, WORK_ORDER), lambda t: re.sub(r"(?m)^Status:.*$", "Status: DISPATCH_READY", t, count=1))
    _, t13 = c.run("N13", "positive control: DISPATCH_READY with the intact contract is a CHECK/PASS",
                   phase_cmd("pre-dispatch", [WORK_ORDER]), n, expect_text=["CVF-DG-APPL-01 PASS", "checked: " + WORK_ORDER])
    # legacy gate integration of F1 on the real work order and real lock
    n = fresh_copy(c, "n14_legacy_tranche", clone)
    c.run("N14-control", "control: legacy tranche gate on the unmutated copy", py("scripts/cvf_downstream_gate.py", "preflight", "--only", "tranche"),
          n, expect=0)
    for rid, label, mut in (
        ("n15_fenced_status", "work-order Status moved into a backtick fence",
         lambda t: re.sub(r"(?m)^Status: (\S+)$", lambda m: "```\nStatus: " + m.group(1) + "\n```", t, count=1)),
        ("n16_tilde_fenced_status", "work-order Status moved into a tilde fence",
         lambda t: re.sub(r"(?m)^Status: (\S+)$", lambda m: "~~~\nStatus: " + m.group(1) + "\n~~~", t, count=1)),
        ("n17_backtick_value", "work-order Status value wrapped in backticks",
         lambda t: re.sub(r"(?m)^Status: (\S+)$", lambda m: "Status: `" + m.group(1) + "`", t, count=1)),
        ("n18_malformed_second", "malformed second Status declaration appended",
         lambda t: t.replace("\nOwner:", "\nStatus: soon, pending\nOwner:", 1)),
        ("n19_duplicate_status", "duplicate identical Status declaration",
         lambda t: re.sub(r"(?m)^Status: (\S+)$", lambda m: "Status: " + m.group(1) + "\nStatus: " + m.group(1), t, count=1)),
    ):
        n = fresh_copy(c, rid, clone)
        edit(os.path.join(n, WORK_ORDER), mut)
        c.run(rid[:3].upper(), f"real gate + real lock: {label}", py("scripts/cvf_downstream_gate.py", "preflight", "--only", "tranche"),
              n, expect="nonzero", expect_text=["work order"])
    n = fresh_copy(c, "n20_classifier_tamper", clone)
    edit(os.path.join(n, "scripts/cvf_gates/check_gate_to_role_closeability.py"), lambda t: t + "\n# tamper\n")
    c.run("N20", "legacy tranche gate refuses a drifted pinned classifier before using it",
          py("scripts/cvf_downstream_gate.py", "preflight", "--only", "tranche"), n, expect="nonzero", expect_text=["lock pin"])
    return c.write_stage("negatives")


def stage_prci(c):
    snap = load_snapshot(c)
    cand = snap["cand"]
    pins, agree = pins_from_files(cand)
    c.add("PRCI-pins", "three pins read from the generated workflow equal the lock", agree, pins)
    base = snap["base"]
    c.run("PRCI-positive", "pr-ci with explicit legitimate base/head and all three pins in the candidate repo",
          py(GATES, "run", "--phase", "pr-ci", "--project-root", ".", "--base", base, "--head", "HEAD", "--no-receipt") + expect_flags(pins),
          cand, expect=0, expect_text=["RESULT: PASS", "mode=RANGE"])
    c.run("PRCI-json", "same range with --json so each mandatory control outcome is machine-readable",
          py(GATES, "run", "--phase", "pr-ci", "--project-root", ".", "--base", base, "--head", "HEAD", "--no-receipt", "--json") + expect_flags(pins),
          cand, expect=0, expect_text=["CVF-DG-APPL-01", "CVF-DG-ROUTE-01", "CVF-DG-ROLE-01", "CVF-DG-CLAIM-01", "CVF-DG-RANGE-01"])
    unknown = "0123456789abcdef0123456789abcdef01234567"
    c.run("PRCI-unknown-base", "unknown base commit is refused, not treated as an empty range",
          py(GATES, "run", "--phase", "pr-ci", "--project-root", ".", "--base", unknown, "--head", "HEAD", "--no-receipt") + expect_flags(pins),
          cand, expect="nonzero", expect_text=["RANGE_BASE_UNRESOLVED"])
    c.run("PRCI-no-base", "pr-ci without a base is refused", py(GATES, "run", "--phase", "pr-ci", "--project-root", ".", "--head", "HEAD",
          "--no-receipt") + expect_flags(pins), cand, expect="nonzero", expect_text=["RANGE_BASE_MISSING"])
    c.run("PRCI-env-unset", "--base-from-env with an unset variable is refused",
          py(GATES, "run", "--phase", "pr-ci", "--project-root", ".", "--base-from-env", "CVF_DG_BASE_UNSET_FOR_PROOF", "--head", "HEAD",
             "--no-receipt") + expect_flags(pins), cand, expect="nonzero", expect_text=["RANGE_BASE_MISSING"])
    c.run("PRCI-head-not-checked-out", "head that is not the checked-out commit is refused",
          py(GATES, "run", "--phase", "pr-ci", "--project-root", ".", "--base", base, "--head", base, "--no-receipt") + expect_flags(pins),
          cand, expect="nonzero", expect_text=["RANGE_HEAD_NOT_CHECKED_OUT"])
    wrong = dict(pins, profile="2" * 64)
    c.run("PRCI-wrong-pin", "pr-ci with a wrong profile pin is refused",
          py(GATES, "run", "--phase", "pr-ci", "--project-root", ".", "--base", base, "--head", "HEAD", "--no-receipt") + expect_flags(wrong),
          cand, expect="nonzero", expect_text=["REFUSED"])
    return c.write_stage("prci", {"candidateCommit": snap["commit"], "baseCommit": base})


def stage_final(c):
    pkts = [WORK_ORDER, REVIEW, RETURN_DOC]
    digests = {rel: lf_sha(open(os.path.join(c.project, rel), "rb").read()) for rel in pkts}
    c.add("FIN-doc-digests", "digests of the packets judged by the final-state gates", True, digests)
    status = re.search(r"(?m)^Status:\s*(\S+)", open(os.path.join(c.project, WORK_ORDER), encoding="utf-8").read())
    c.add("FIN-status", "work order is in its return display state", bool(status) and status.group(1) == "COMPLETE_PENDING_REVIEW",
          {"workOrderStatus": status.group(1) if status else None})
    for phase in ("pre-dispatch", "worker-return", "reviewer-fast", "pre-commit"):
        c.run(f"FIN-{phase}", f"live final-state {phase} with explicit packets", phase_cmd(phase, pkts), c.project, expect=0,
              expect_text=["RESULT: PASS"])
    c.run("FIN-preflight", "legacy seven-gate preflight on the final tree", py("scripts/cvf_downstream_gate.py", "preflight"), c.project,
          expect=0, expect_text=["PASS (7/7 executed gates passed)"])
    base = git(["rev-parse", "HEAD"], c.project).strip()
    c.run("FIN-closeability", "pinned gate-to-role closeability checker (order + return recheck) over the change set",
          py("scripts/cvf_gates/check_gate_to_role_closeability.py", "--repo-root", ".", "--base", base, "--head", base, "--enforce"),
          c.project, expect=0, expect_text=["Violations: 0", "COMPLIANT"])
    shell = shutil.which("powershell") or shutil.which("pwsh")
    core = os.path.join(os.path.dirname(c.project), ".Controlled-Vibe-Framework-CVF")
    c.run("FIN-doctor", "Core workspace enforcement doctor on the live worktree",
          [shell, "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", os.path.join(core, "scripts/check_cvf_workspace_agent_enforcement.ps1"),
           "-ProjectPath", "."], c.project, expect=0, expect_text=["RESULT: PASS"])
    c.run("FIN-diff-check", "git diff --check over tracked changes", ["git", "diff", "--check"], c.project, expect=0)
    broken = {}
    readme = open(os.path.join(c.project, "README.md"), encoding="utf-8-sig").read()
    if "(docs/reference/CVF_PROJECT_PORTABLE_CONTINUATION.md)" not in readme:
        broken["README.md"] = ["docs/reference/CVF_PROJECT_PORTABLE_CONTINUATION.md link missing"]
    for rel in ("docs/reference/CVF_PROJECT_PORTABLE_CONTINUATION.md", "AGENTS.md", RETURN_DOC):
        text = open(os.path.join(c.project, rel), encoding="utf-8-sig").read()
        text = re.sub(r"(?s)```.*?```", "", text)
        for target in re.findall(r"\[[^\]]*\]\(([^)\s]+)\)", text):
            if re.match(r"^(https?:|mailto:|#)", target):
                continue
            path = target.split("#", 1)[0]
            if path and not os.path.exists(os.path.normpath(os.path.join(c.project, os.path.dirname(rel), path))):
                broken.setdefault(rel, []).append(target)
    c.add("FIN-local-links", "local markdown links in the files this repair authored all resolve (no new dead link)", not broken, {"broken": broken})
    workflows = [p for p in git(["ls-files", ".github/workflows"], c.project).split("\n") if p]
    protected = [".cvf/policy.json", "scripts/tests/test_cvf_downstream_gate.py", "CVF_SESSION/authority", "backend", "frontend"] + workflows
    existing = [p for p in protected if os.path.exists(os.path.join(c.project, p))]
    diff = git(["status", "--porcelain", "--"] + existing, c.project)
    c.add("FIN-protected", "no product code, policy, seed, existing workflow or existing legacy test changed",
          diff.strip() == "", {"checkedPaths": existing, "porcelain": diff.strip()[:400]})
    return c.write_stage("final", {"workOrderStatus": status.group(1) if status else None})


def stage_manifest(c):
    """Candidate manifest (every non-ignored changed/untracked path) and the round1 receipt that indexes the stage files."""
    head = git(["rev-parse", "HEAD"], c.project).strip()
    present, deleted = change_set(c.project)
    present = [p for p in present if p not in (MANIFEST_OUT, ROUND1_RECEIPT)]
    audit = json.load(open(os.path.join(c.project, "docs/reviews/probes/gov002_review_round0/local-review-audit.json"), encoding="utf-8"))
    round0 = {item["path"]: item["sha256"] for item in audit["candidateChangedManifest"]}
    files, raw_by_path = {}, {}
    for rel in present:
        data = open(os.path.join(c.project, rel), "rb").read()
        raw_by_path[rel] = hashlib.sha256(data).hexdigest()
        files[rel] = {"rawSha256": raw_by_path[rel], "lfNormalizedSha256": lf_sha(data), "bytes": len(data)}
    other_actor = {rel: reason for rel, reason in OTHER_ACTOR_PATHS.items() if rel in files}
    for rel, reason in other_actor.items():
        files[rel]["notWorkerAuthored"] = reason
    changed = sorted(p for p in present if p in round0 and round0[p] != raw_by_path[p])
    added = sorted(p for p in present if p not in round0)
    removed = sorted(p for p in round0 if p not in raw_by_path)
    unchanged = sorted(p for p in present if p in round0 and round0[p] == raw_by_path[p])
    snap = load_snapshot(c)
    recorded = json.load(open(os.path.join(c.out, "stage-snapshot.json"), encoding="utf-8"))["candidateManifestLfSha256"]
    catalog_trio = {"docs/INDEX.md", "docs/catalog/MODULE_CATALOG.md", "docs/catalog/ARTIFACT_REGISTRY.json"}
    drift = sorted(rel for rel, digest in recorded.items()
                   if rel not in catalog_trio and (not os.path.isfile(os.path.join(c.project, rel))
                                                   or lf_sha(open(os.path.join(c.project, rel), "rb").read()) != digest))
    c.add("MAN-snapshot-still-faithful", "every snapshot-covered live file (except the declared catalog trio) is unchanged since the snapshot",
          not drift, {"driftedPaths": drift, "checkedPaths": len(recorded) - len(catalog_trio & set(recorded))})
    manifest = {
        "schemaVersion": "gov002.round1.candidateManifest@1", "generatedAtUtc": utc_now(), "executionHead": head,
        "authoritySeedCommit": "f7bf57d0f4fe0e1920efdcc3609b6a8ea84cb68d",
        "note": "executionHead is the actual checked-out dispatch commit; the authority seed commit is an earlier, different commit",
        "pathCount": len(files), "workerCandidatePathCount": len(files) - len(other_actor), "concurrentOtherActorPaths": other_actor,
        "deletedPaths": deleted,
        "round0Comparison": {"basis": "docs/reviews/probes/gov002_review_round0/local-review-audit.json#/candidateChangedManifest (raw sha256)",
                             "round0PathCount": len(round0), "changedSinceRound0": changed, "addedSinceRound0": added,
                             "removedSinceRound0": removed, "unchangedSinceRound0Count": len(unchanged)},
        "faithfulSnapshot": {"candidateCommit": snap["commit"], "candidateTree": snap["tree"], "baseCommit": snap["base"],
                             "snapshotCoveredPaths": len(snap["copied"]), "declaredNotInSnapshot": snap["excluded"],
                             "liveDriftSinceSnapshot": drift,
                             "snapshotCatalogDelta": "snapshot registry omits entries for declared evidence outputs; docs/INDEX.md, "
                                                     "docs/catalog/MODULE_CATALOG.md and ARTIFACT_REGISTRY.json differ from the live tree only by those entries"},
        "files": dict(sorted(files.items())),
    }
    with open(os.path.join(c.project, MANIFEST_OUT), "w", encoding="utf-8", newline="\n") as fh:
        json.dump(manifest, fh, indent=2, ensure_ascii=False)
        fh.write("\n")
    stages = {}
    unexpected_total = 0
    for name in ("buildtime", "discrimination", "snapshot", "clone", "negatives", "prci", "final"):
        path = os.path.join(c.out, f"stage-{name}.json")
        if os.path.isfile(path):
            body = json.load(open(path, encoding="utf-8"))
            stages[name] = {"path": os.path.relpath(path, c.project).replace("\\", "/"), "recordCount": body["recordCount"],
                            "unexpectedCount": body["unexpectedCount"], "recordedAtUtc": body["recordedAtUtc"],
                            "sha256": hashlib.sha256(open(path, "rb").read()).hexdigest()}
            unexpected_total += body["unexpectedCount"]
    receipt = {
        "schemaVersion": "gov002.round1.workerReceipt@1", "generatedAtUtc": utc_now(), "packet": "CCMAI-GOV-002-C1", "reviewRound": 1,
        "actor": "ccmai-gov002-worker", "role": "REPAIR_WORKER", "returnStatus": "COMPLETE_PENDING_REVIEW",
        "executionHead": head, "committed": False, "pushed": False,
        "workerModel": {"model": "claude-sonnet-5-5", "host": "Claude Code (VS Code extension)",
                        "reasoningEffort": "session parameter 10 (scale not documented to the worker)",
                        "tokenUsage": None, "bill": None,
                        "unavailableReason": "token and bill telemetry is not exposed to the worker; effort scale is not documented to it"},
        "priorFindingSetDigest": "4c062e7c74efefa2d2e9e6cd5811cbfb48b152ee7cd9c99905e6e8826f4c5bdb",
        "candidateManifest": MANIFEST_OUT, "stages": stages, "unexpectedTotal": unexpected_total,
        "externalInvocationCount": 0, "providerCalls": 0, "applicationRuns": 0,
        "networkUse": "git clone/fetch of the declared public project candidate (local) and the public CVF Core by scripts/initialize_cvf_clone.ps1 only",
        "claimBoundary": "static repository evidence in the stated trees; no hosted CI, runtime governance, provider, docs55, R086 native or deployment claim",
    }
    with open(os.path.join(c.project, ROUND1_RECEIPT), "w", encoding="utf-8", newline="\n") as fh:
        json.dump(receipt, fh, indent=2, ensure_ascii=False)
        fh.write("\n")
    c.add("MAN-written", "manifest and round1 receipt written", unexpected_total == 0,
          {"pathCount": len(files), "changedSinceRound0": changed, "addedSinceRound0Count": len(added), "unexpectedTotal": unexpected_total})
    return 0 if unexpected_total == 0 and not drift else 1


STAGES = {"buildtime": stage_buildtime, "discrimination": stage_discrimination, "snapshot": stage_snapshot, "clone": stage_clone,
          "negatives": stage_negatives, "prci": stage_prci, "final": stage_final, "manifest": stage_manifest}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--project", required=True)
    parser.add_argument("--work", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--stage", required=True, choices=sorted(STAGES))
    args = parser.parse_args()
    ctx = Ctx(args.project, args.work, args.out)
    unexpected = STAGES[args.stage](ctx)
    return 1 if unexpected else 0


if __name__ == "__main__":
    sys.exit(main())
