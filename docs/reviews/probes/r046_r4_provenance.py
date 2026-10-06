"""Read-only recovery manifest; no runtime, mutations, extraction or copy repair.

Run from project root: python -B docs/reviews/probes/r046_r4_provenance.py
Only the scratch paths explicitly named by the R3 committed runner are read.
"""
import hashlib
import io
import json
import pathlib
import subprocess
import tarfile

REF = "91da0e88118b76a68031f432da50521fe6a341b7"
SCRATCH = pathlib.Path(r"C:\Users\tiennm\.gemini\antigravity-ide\brain\dc3013b0-c95d-4c95-97d9-8e786c64953b\scratch")
OUT = pathlib.Path("docs/reviews/probes/r046_r4_snapshot_manifest.json")


def sha(data):
    return hashlib.sha256(data).hexdigest()


def main():
    canonical = subprocess.check_output(["git", "archive", REF, "backend"])
    saved_bytes = (SCRATCH / "r046-r3-source.tar").read_bytes()
    live_root = SCRATCH / "r046-r3-livecopy"
    files = []
    with tarfile.open(fileobj=io.BytesIO(canonical)) as exported, tarfile.open(fileobj=io.BytesIO(saved_bytes)) as saved:
        for member in exported.getmembers():
            if not member.isfile():
                continue
            original = exported.extractfile(member).read()
            archived = saved.extractfile(member.name).read()
            copied = (live_root / member.name.removeprefix("backend/")).read_bytes()
            files.append({"path": member.name, "canonicalExportSha256": sha(original),
                          "savedArchiveSha256": sha(archived), "recoveredLivecopySha256": sha(copied),
                          "canonicalNormalizedSha256": sha(original.replace(b"\r\n", b"\n")),
                          "livecopyNormalizedSha256": sha(copied.replace(b"\r\n", b"\n")),
                          "archiveByteEqual": original == archived, "livecopyByteEqual": original == copied,
                          "livecopyCRLFNormalizedEqual": original.replace(b"\r\n", b"\n") == copied.replace(b"\r\n", b"\n")})
        comment = saved.pax_headers.get("comment")
    names = {row["path"] for row in files}
    recovered_names = {"backend/" + f.relative_to(live_root).as_posix() for f in live_root.rglob("*") if f.is_file()}
    receipt = json.loads(pathlib.Path("docs/reviews/probes/r046_r3_worker_receipts.json").read_text())
    logs = []
    for run in receipt["runs"]:
        anchor = run["secretFreeLogOrExtractPath"].rsplit("#", 1)[1]
        data = (SCRATCH / "r046-r3-logs" / (anchor + ".jsonl")).read_bytes()
        text = data.decode("utf-16" if data.startswith((b"\xff\xfe", b"\xfe\xff")) else "utf-8-sig")
        events = [json.loads(line) for line in text.splitlines() if line.strip()]
        completed = [e for e in events if e.get("Test") and e.get("Action") in ("pass", "fail", "skip")]
        logs.append({"name": run["name"], "rawLogSha256": sha(data), "digestMatchesR3Receipt": sha(data) == run["rawLogSha256"],
                     "topLevelPass": sum(e["Action"] == "pass" and "/" not in e["Test"] for e in completed),
                     "subtestPass": sum(e["Action"] == "pass" and "/" in e["Test"] for e in completed),
                     "completedEvents": len(completed), "failNames": [e["Test"] for e in completed if e["Action"] == "fail"],
                     "skipNames": [e["Test"] for e in completed if e["Action"] == "skip"]})
    manifest = {"date": "2026-10-05", "worker": "Codex subagent r046_r4_worker", "executionTranche": "CCMAI-RUNTIME-047",
                "productReferenceCommit": REF, "recoveryParentCommit": subprocess.check_output(["git", "rev-parse", "HEAD"], text=True).strip(),
                "comparisonTarget": "Fresh git archive export, not raw Git blobs; exports can apply CRLF conversion",
                "savedArchive": {"sha256": sha(saved_bytes), "commitComment": comment, "executedByCommittedRunner": False},
                "declaredRunnerExecutionPath": str(live_root), "backendFiles": len(files),
                "archiveEquality": all(f["archiveByteEqual"] for f in files), "executedCopyByteEquality": all(f["livecopyByteEqual"] for f in files),
                "recoveredCopyCRLFNormalizedEquality": all(f["livecopyCRLFNormalizedEqual"] for f in files),
                "livecopyByteDifferences": [f["path"] for f in files if not f["livecopyByteEqual"]],
                "unexpectedLivecopyFiles": sorted(recovered_names - names), "missingLivecopyFiles": sorted(names - recovered_names),
                "exactArchiveHistoricalExecution": "NOT MET / NOT VERIFIED; no acceptance waiver",
                "temporalLimit": "Present surviving copy and logs are read-only recovered observations. They cannot independently prove filesystem bytes, isolation inspections or removal exits at past execution time.",
                "files": files, "recoveredR3Logs": logs, "newRuntimeCampaigns": 0, "newMutations": 0}
    assert len(files) == 203 and manifest["archiveEquality"] and manifest["recoveredCopyCRLFNormalizedEquality"]
    assert not manifest["executedCopyByteEquality"] and len(manifest["livecopyByteDifferences"]) == 3
    assert not manifest["unexpectedLivecopyFiles"] and not manifest["missingLivecopyFiles"]
    assert all(row["digestMatchesR3Receipt"] for row in logs)
    OUT.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({k: v for k, v in manifest.items() if k not in ("files", "recoveredR3Logs")}, indent=2))


if __name__ == "__main__":
    main()
