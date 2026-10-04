"""Read-only R3 receipt audit. Run from project root; never runs worker campaign.

Usage: python -B docs/reviews/probes/r046_r3_independent_audit.py SCRATCH OUTPUT
SCRATCH is the explicit scratch directory declared in the committed worker runner.
Only declared logs, archive and livecopy are read. Output contains no raw SQL.
"""
import collections
import hashlib
import io
import json
import pathlib
import subprocess
import sys
import tarfile

REF = "91da0e88118b76a68031f432da50521fe6a341b7"
HANDBACK = "c09c8f62585945d3624e5a0f3ddc305bb35c4552"


def sha(data):
    return hashlib.sha256(data).hexdigest()


def main():
    scratch, output = map(pathlib.Path, sys.argv[1:])
    receipt = json.loads(pathlib.Path("docs/reviews/probes/r046_r3_worker_receipts.json").read_text())
    extracts = json.loads(pathlib.Path("docs/reviews/probes/r046_r3_log_extracts.json").read_text())
    result = {"sourceCommit": REF, "evidenceCommitAtHandback": HANDBACK,
              "role": "Codex independent REVIEWER", "runs": [], "mutations": [],
              "claimScope": "Recovered synthetic application evidence; no live governance proof."}
    for run in receipt["runs"]:
        log_name = run["secretFreeLogOrExtractPath"].rsplit("#", 1)[1]
        raw = (scratch / "r046-r3-logs" / (log_name + ".jsonl")).read_bytes()
        text = raw.decode("utf-16" if raw.startswith((b"\xff\xfe", b"\xfe\xff")) else "utf-8-sig")
        events, bad = [], 0
        for line in text.splitlines():
            if not line.strip():
                continue
            try:
                events.append(json.loads(line))
            except json.JSONDecodeError:
                bad += 1
        done = [e for e in events if e.get("Test") and e.get("Action") in ("pass", "fail", "skip")]
        count = collections.Counter((e["Action"], "/" in e["Test"]) for e in done)
        derived = dict(topLevelPass=count["pass", False], subtestPass=count["pass", True],
                       topLevelFail=count["fail", False], subtestFail=count["fail", True],
                       skipCount=count["skip", False] + count["skip", True])
        names = {action: [e["Test"] for e in done if e["Action"] == action] for action in ("pass", "fail", "skip")}
        ext = extracts[log_name]
        sample_matches = []
        for sample in ext["sampleOutput"]:
            try:
                sample_matches.append(json.loads(sample) in events)
            except (json.JSONDecodeError, TypeError):
                sample_matches.append(any(sample in e.get("Output", "") for e in events))
        package = [e for e in events if not e.get("Test") and e.get("Action") in ("pass", "fail", "skip")]
        fragments = run["namedFailureOutput"].replace("\\x1b", "\x1b").split(" | ")
        joined_fragments_match = all(any(fragment in e.get("Output", "") for e in events)
                                     for fragment in fragments if fragment)
        result["runs"].append({"name": run["name"], "rawLogSha256": sha(raw),
            "digestMatches": sha(raw) == run["rawLogSha256"] == ext["rawLogSha256"],
            "parseableJsonEvents": len(events), "completedTestEvents": len(done),
            "parseFailures": bad, **derived, "passNames": names["pass"], "failNames": names["fail"],
            "countsMatchReceiptAndExtract": all(run[k] == ext[k] == v for k, v in derived.items()),
            "namesMatchReceiptAndExtract": all(sorted(names[a]) == sorted(run[a + "Names"]) == sorted(ext[a + "Names"]) for a in ("pass", "fail")),
            "extractSamplesMatchRaw": all(sample_matches),
            "packageTerminalAction": [e["Action"] for e in package],
            "exitConsistentWithPackage": len(package) == 1 and (run["exit"] == 0) == (package[0]["Action"] == "pass"),
            "namedFailureJoinedFragmentsPresent": joined_fragments_match})
    canonical = subprocess.check_output(["git", "archive", REF, "backend"])
    expected = tarfile.open(fileobj=io.BytesIO(canonical))
    worker_raw = (scratch / "r046-r3-source.tar").read_bytes()
    worker = tarfile.open(fileobj=io.BytesIO(worker_raw))
    archive_diff, copy_diff, semantic_diff = [], [], []
    baseline = None
    members = [m for m in expected.getmembers() if m.isfile()]
    for member in members:
        original = expected.extractfile(member).read()
        archived = worker.extractfile(member.name).read()
        copied = (scratch / "r046-r3-livecopy" / member.name.removeprefix("backend/")).read_bytes()
        if archived != original:
            archive_diff.append(member.name)
        if copied != original:
            copy_diff.append(member.name)
        if copied.replace(b"\r\n", b"\n") != original.replace(b"\r\n", b"\n"):
            semantic_diff.append(member.name)
        if member.name == "backend/engine/analyzer_incremental.go":
            baseline = original
    expected_names = {m.name for m in members}
    copy_names = {"backend/" + p.relative_to(scratch / "r046-r3-livecopy").as_posix()
                  for p in (scratch / "r046-r3-livecopy").rglob("*") if p.is_file()}
    result["snapshot"] = {"backendFiles": len(members), "archiveSha256": sha(worker_raw),
        "archiveHashMatchesReceipt": sha(worker_raw) == receipt["taskArchiveSha256"],
        "archiveCommitComment": worker.pax_headers.get("comment"),
        "archiveVsCanonicalExportByteDifferences": archive_diff,
        "livecopyVsCanonicalExportByteDifferences": copy_diff,
        "livecopySemanticDifferencesAfterCRLFNormalization": semantic_diff,
        "livecopyUnexpectedFiles": sorted(copy_names - expected_names),
        "runnerSha256": sha(pathlib.Path("docs/reviews/probes/r046_r3_campaign_runner.ps1").read_bytes())}
    for mutant in receipt["mutants"]:
        old, new = (mutant["exactReplacementOrDiff"][k].encode() for k in ("old", "new"))
        modified = baseline.replace(old, new)
        result["mutations"].append({"id": mutant["id"], "matches": baseline.count(old),
            "mutatedSha256": sha(modified), "hashMatchesReceipt": sha(modified) == mutant["mutatedSha256"],
            "killingTest": mutant["killingTest"], "restoredFileCurrentlyEqualsBaseline": sha(baseline) == mutant["restoredSha256"]})
    result["workerReceiptLimitations"] = {"evidenceCommitAtHandback": receipt["evidenceCommitAtHandback"],
        "docsBuildAfterFinalMarkdown": receipt["prerequisites"]["docsBuildAfterFinalMarkdown"],
        "note": "Catalog generation is not a VitePress build. Reviewer publication and current absence checks do not prove historical worker execution or isolation inspections."}
    output.write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"runs": len(result["runs"]), "snapshot": result["snapshot"],
        "runChecks": [{k:v for k,v in run.items() if isinstance(v,bool)} for run in result["runs"]]}, indent=2))


if __name__ == "__main__":
    main()
