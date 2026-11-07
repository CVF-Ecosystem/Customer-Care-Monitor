"""Declared, pinned retrospective admission; never effect permission.

Caller context is explicit. Bindings/review are declared repository artifacts, not
authenticated parent acceptance. Only the old audit/action join is substituted.
"""
import hashlib
import json
import os
import re
import subprocess
import stat
from pathlib import Path

LIMIT = 1024 * 1024
TOTAL = 16 * LIMIT
SHA40 = re.compile(r"[0-9a-f]{40}\Z")
SHA64 = re.compile(r"[0-9a-f]{64}\Z")
SCHEMA = "cvf.historicalRepairAdmissions@1.0.0"
FIELDS = ("milestone", "tranche", "packet", "admission", "build", "terminal",
          "registry", "audit", "seed", "terminalRecord", "carrier", "reviewEvidence")


def digest(raw):
    return hashlib.sha256(raw.replace(b"\r\n", b"\n")).hexdigest()


class Reader:
    def __init__(self, root, api):
        self.root, self.api, self.total = Path(root).resolve(), api, 0

    def safe(self, relative):
        self.api.need(type(relative) is str and 0 < len(relative) <= 1024 and "\\" not in relative
                      and ":" not in relative and not relative.startswith("/")
                      and all(p not in ("", ".", "..") for p in relative.split("/")), "HIST_PATH")
        target = self.root
        for part in relative.split("/"):
            target /= part
            try:
                info = target.lstat()
            except FileNotFoundError:
                continue
            self.api.need(not stat.S_ISLNK(info.st_mode) and not
                          (getattr(info, "st_file_attributes", 0) & 0x400), "HIST_SYMLINK")
        self.api.need(target.resolve().is_relative_to(self.root), "HIST_PATH")
        return target

    def charge(self, raw):
        self.total += len(raw)
        self.api.need(len(raw) <= LIMIT and self.total <= TOTAL, "HIST_READ_BUDGET")
        return raw

    def file(self, relative):
        target = self.safe(relative)
        if os.name == "nt" and not str(target).startswith("\\\\?\\"):
            target = Path("\\\\?\\" + str(target.absolute()))
        before = target.stat()
        self.api.need(target.is_file() and before.st_size <= LIMIT, "HIST_OVERSIZE")
        with target.open("rb") as stream:
            raw = stream.read(LIMIT + 1)
        after = target.stat()
        self.api.need((before.st_ino, before.st_dev, before.st_size, before.st_mtime_ns)
                      == (after.st_ino, after.st_dev, after.st_size, after.st_mtime_ns), "HIST_FILE_RACE")
        self.safe(relative)
        return self.charge(raw)

    def git(self, *args):
        try:
            result = subprocess.run(["git", "--no-replace-objects", "--literal-pathspecs", "-C", str(self.root), *args],
                                    capture_output=True, timeout=10)
        except (OSError, subprocess.TimeoutExpired):
            self.api.need(False, "HIST_GIT_UNAVAILABLE")
        self.api.need(result.returncode == 0, "HIST_GIT_FAILED")
        self.api.need(len(result.stdout) <= LIMIT, "HIST_GIT_OUTPUT")
        return result.stdout

    def commit(self, value):
        self.api.need(type(value) is str and bool(SHA40.fullmatch(value)), "HIST_COMMIT")
        self.api.need(self.git("rev-parse", "--verify", value + "^{commit}").decode().strip() == value,
                      "HIST_COMMIT")
        return value

    def ancestor(self, before, after):
        self.git("merge-base", "--is-ancestor", self.commit(before), self.commit(after))

    def blob(self, commit, relative):
        self.safe(relative)
        self.commit(commit)
        entry = self.git("ls-tree", "-z", commit, "--", relative)
        self.api.need(entry.count(b"\0") == 1, "HIST_BLOB_ENTRY")
        meta, name = entry[:-1].split(b"\t", 1)
        fields = meta.split()
        self.api.need(len(fields) == 3 and fields[0] in (b"100644", b"100755")
                      and fields[1] == b"blob" and name.decode() == relative, "HIST_BLOB_ENTRY")
        oid = fields[2].decode()
        size = int(self.git("cat-file", "-s", oid))
        self.api.need(0 <= size <= LIMIT and self.total + size <= TOTAL, "HIST_OVERSIZE")
        return self.charge(self.git("cat-file", "blob", oid))

    def reference(self, value, commit=None, retained=True):
        self.api.shape(value, ("path", "sha256"))
        self.api.need(type(value["sha256"]) is str and bool(SHA64.fullmatch(value["sha256"])), "HIST_HASH")
        raw = self.blob(commit, value["path"]) if commit else self.file(value["path"])
        self.api.need(digest(raw) == value["sha256"], "HIST_STALE")
        if commit and retained:
            self.api.need(digest(self.file(value["path"])) == value["sha256"], "HIST_RETAINED_STALE")
        return raw

    def immutable(self, value, commit):
        self.ancestor(commit, self.head)
        raw = self.reference(value, commit)
        changes = self.git("log", "--format=%H", "--max-count=2", self.head, "--", value["path"]).decode().splitlines()
        self.api.need(changes == [commit], "HIST_PROVENANCE_CHANGED")
        return raw

    def unchanged_since(self, relative, terminal):
        self.safe(relative)
        changed = self.git("log", "--ancestry-path", "--format=%H", "--max-count=1",
                           terminal + ".." + self.head, "--", relative)
        self.api.need(not changed.strip(), "HIST_POST_TERMINAL_MUTATION")


def select(root, packet_path, context, registry, repair, api):
    """Validate EVERY binding, then select an exact old audit or leave default strict."""
    api.shape(context, ("phase", "head", "base", "active", "explicit", "manifest"))
    api.need(context["phase"] == "pr-ci", "HIST_EXECUTION_FORBIDDEN")
    api.need(type(context["active"]) is list and type(context["explicit"]) is list
             and all(type(p) is str for p in context["active"] + context["explicit"]), "HIST_CONTEXT")
    reader = Reader(root, api)
    # The runner's discovery normalizes aliases. Reject noncanonical caller paths
    # here so a raw backslash/./ argument cannot evade exact execution exclusions.
    for path in context["active"] + context["explicit"]:
        reader.safe(path)
    reader.head = reader.commit(context["head"])
    api.need(reader.git("rev-parse", "HEAD").decode().strip() == reader.head, "HIST_HEAD_MISMATCH")
    reader.ancestor(context["base"], reader.head)
    api.shape(context["manifest"], ("path", "sha256", "commit"))
    mref = {k: context["manifest"][k] for k in ("path", "sha256")}
    introduced = reader.commit(context["manifest"]["commit"])
    doc = repair.parse(reader.immutable(mref, introduced), api)
    api.shape(doc, ("schemaVersion", "bindings", "review"))
    api.need(doc["schemaVersion"] == SCHEMA and type(doc["bindings"]) is list
             and 0 < len(doc["bindings"]) <= 32, "HIST_SCHEMA")
    api.shape(doc["review"], ("path", "sha256", "commit"))
    review_ref = {k: doc["review"][k] for k in ("path", "sha256")}
    reader.ancestor(doc["review"]["commit"], introduced)
    review = repair.parse(reader.immutable(review_ref, doc["review"]["commit"]), api)
    api.shape(review, ("schemaVersion", "decision", "reviewer", "worker", "bindingsSha256"))
    canonical = json.dumps(doc["bindings"], ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode()
    api.need(review["schemaVersion"] == "cvf.historicalAdmissionReview@1.0.0"
             and review["decision"] == "ACCEPTED" and review["bindingsSha256"] == digest(canonical), "HIST_REVIEW")
    api.text(review["reviewer"]); api.text(review["worker"])
    api.need(review["reviewer"] != review["worker"], "HIST_INDEPENDENT_REVIEW")
    packets, tranches, chosen = set(), set(), None
    current = repair.controls(registry, api)
    for row in doc["bindings"]:
        api.shape(row, FIELDS)
        api.text(row["milestone"]); api.text(row["tranche"])
        api.shape(row["packet"], ("path", "sha256"))
        path = row["packet"]["path"]
        api.need(type(path) is str and path.startswith("docs/work_orders/") and "/archive/" not in path,
                 "HIST_PACKET_LOCATION")
        api.need(path not in packets and row["tranche"] not in tranches, "HIST_DUPLICATE")
        packets.add(path); tranches.add(row["tranche"])
        api.need(path not in context["active"] and path not in context["explicit"], "HIST_EXECUTION_FORBIDDEN")
        for key in ("admission", "build", "terminal"):
            reader.ancestor(row[key], reader.head)
        reader.ancestor(row["admission"], row["build"])
        reader.ancestor(row["build"], row["terminal"])
        api.need(row["admission"] != row["build"] and row["build"] != row["terminal"], "HIST_ORDER")
        reader.ancestor(row["terminal"], doc["review"]["commit"])
        for commit in (row["admission"], row["terminal"]):
            reader.reference(row["packet"], commit)
        api.need(row["registry"]["path"] == repair.REGISTRY, "HIST_REGISTRY_LOCATION")
        old = repair.parse(reader.reference(row["registry"], row["admission"], retained=False), api)
        milestone = row["milestone"]
        old_controls = repair.controls(old, api)
        api.need(milestone in old_controls and milestone in current
                 and old["milestones"].get(milestone) == registry["milestones"].get(milestone), "HIST_MILESTONE")
        before, after = old_controls[milestone], current[milestone]
        api.need(before["audit"] == row["audit"], "HIST_AUDIT_SELECTION")
        api.need(after["history"][:len(before["history"])] == before["history"]
                 and set(before["affectedCriteria"]) <= set(after["affectedCriteria"]), "HIST_HISTORY_RESET")
        reader.reference(row["audit"], row["admission"])
        # Current audit/history/cost checks precede any historical substitution.
        repair.inspect_control(root, milestone, after, api)
        # Existing pinned audit parser and all retained dependency/cost checks.
        _, old_audit = repair.inspect_control(root, milestone, before, api)
        api.need(old_audit is not None and old_audit["admittedAction"] is not None
                 and old_audit["admittedAction"]["packet"] == row["packet"], "HIST_AUDIT_PACKET")
        api.need(row["seed"]["path"] == "CVF_SESSION/authority/" + row["tranche"] + ".json"
                 and row["terminalRecord"]["path"] == "CVF_SESSION/tranches/" + row["tranche"] + ".json",
                 "HIST_TRANCHE_LOCATION")
        seed = repair.parse(reader.reference(row["seed"], row["admission"]), api)
        reader.reference(row["seed"], row["terminal"])
        terminal = repair.parse(reader.reference(row["terminalRecord"], row["terminal"]), api)
        api.need(seed.get("trancheId") == terminal.get("trancheId") == row["tranche"]
                 and seed.get("workOrder") == path and path in seed.get("allowedPaths", [])
                 and terminal.get("baseCommit") == row["admission"]
                 and terminal.get("buildCommit") == row["build"]
                 and terminal.get("status") == terminal.get("freeze") == "FROZEN"
                 and terminal.get("phase") == "FREEZE" and terminal.get("disposition") == "REVIEW_PASS"
                 and terminal.get("workOrderStatus") == "CLOSED", "HIST_TERMINAL")
        roles = terminal.get("roles", {})
        api.need(type(roles.get("reviewer")) is str and roles.get("reviewer")
                 != roles.get("implementationWorker") and type(roles.get("implementationWorker")) is str,
                 "HIST_TERMINAL_REVIEWER")
        carrier = reader.reference(row["carrier"], row["terminal"]).decode("utf-8-sig")
        api.need(row["carrier"]["path"] == terminal.get("workOrder")
                 and re.findall(r"^Status:\s*(\S+)", carrier, re.M) == ["CLOSED"]
                 and re.findall(r"^Tranche:\s*(\S+)", carrier, re.M) == [row["tranche"]], "HIST_CARRIER")
        api.need(type(row["reviewEvidence"]) is list and bool(row["reviewEvidence"])
                 and [r.get("path") for r in row["reviewEvidence"]] == terminal.get("reviewEvidence"), "HIST_REVIEW_EVIDENCE")
        for evidence in row["reviewEvidence"]:
            reader.reference(evidence, row["terminal"])
        for value in [row[k] for k in ("packet", "audit", "seed", "terminalRecord", "carrier")] + row["reviewEvidence"]:
            reader.unchanged_since(value["path"], row["terminal"])
        if path == packet_path:
            chosen = old_audit
    return chosen
