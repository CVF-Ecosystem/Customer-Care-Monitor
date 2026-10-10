"""R067 manual remaining-three continuation under recorded cost disposition; no positive retry/new campaign. Root-owned probe."""
import argparse, hashlib, io, json, os, re, subprocess, tarfile, tempfile, time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
NEW_SUITES = ['src/__tests__/run-observation.spec.ts', 'src/__tests__/run-observation-panel.spec.ts', 'src/__tests__/run-observation-job-detail.spec.ts']
MUTATIONS = [
    dict(label='M01', path='src/components/ui/RunObservationPanel.vue', old="if (value === null) return t('run_obs_unknown')", new="if (value === null) return key === 'local_estimate_usd' ? '0 USD' : t('run_obs_unknown')", suite=NEW_SUITES[1], detector='renders an unknown local estimate without showing zero', healthy='shows a recorded local zero as zero'),
    dict(label='M02', path='src/views/Jobs/job-detail/run-observation.ts', old=' && value.run_id === context.runId', new='', suite=NEW_SUITES[0], detector='rejects a saved receipt bound to another run', healthy='accepts the exact tenant, job, run, and run.job_id context'),
]
def digest(data): return hashlib.sha256(data).hexdigest()
def exclusive_json(path, value):
    with path.open('x', encoding='utf-8', newline='\n') as handle: json.dump(value, handle, indent=2); handle.write('\n')
    assert json.loads(path.read_text()) == value
    return dict(path=str(path.relative_to(ROOT)), bytes=path.stat().st_size, sha256Physical=digest(path.read_bytes()))
def git(*args): return subprocess.check_output(['git', *args], cwd=ROOT)
def main():
    parser=argparse.ArgumentParser(); parser.add_argument('--source', required=True); parser.add_argument('--role', choices=['worker'], required=True); parser.add_argument('--execute', action='store_true'); args=parser.parse_args()
    assert re.fullmatch('[0-9a-f]{40}', args.source)
    prior=json.loads((ROOT/'docs/reviews/probes/r067_worker_summary.json').read_text())
    assert prior['vitestInvocations']==1 and len(prior['commands'])==1 and prior['commands'][0]['failed']==1
    assert prior['finalFullManifestRestored'] and prior['dependencyJunctionRemoved']
    changed=git('diff','--name-only',prior['sourceCommit'],args.source,'--','frontend').decode().splitlines()
    assert changed==['frontend/src/__tests__/run-observation.spec.ts'], 'Resume requires only accepted NEW assertion fixture repair'
    if args.execute:
        accepted=json.loads((ROOT/'docs/reviews/probes/r067_reviewer_summary.json').read_text())
        assert accepted['sourceCommit']==args.source and accepted['vitestInvocations']==4 and accepted['restored'] and accepted['finalFullManifestRestored'] and 'failure' not in accepted
        assert accepted['commands'][0]['exitCode']==0 and accepted['commands'][0]['failed']==0
    archive=git('archive','--format=tar',args.source,'frontend')
    scratch=Path(tempfile.mkdtemp(prefix='ccmai-r067-'+args.role+'-')); frontend=scratch/'frontend'
    with tarfile.open(fileobj=io.BytesIO(archive)) as bundle:
        for member in bundle.getmembers():
            assert not member.issym() and not member.islnk()
            resolved=(scratch/member.name).resolve(); assert resolved.is_relative_to(scratch.resolve())
            assert not Path(member.name).name.startswith('.env')
        bundle.extractall(scratch, filter='data')
    source_files=sorted(p for p in frontend.rglob('*') if p.is_file())
    original={p.relative_to(frontend).as_posix():p.read_bytes() for p in source_files}
    baseline={name:dict(bytes=len(data),sha256Physical=digest(data)) for name,data in original.items()}
    suites=sorted(name for name in original if re.search(r'\.(spec|test)\.[cm]?[jt]sx?$', name))
    assert set(NEW_SUITES).issubset(suites)
    for mutation in MUTATIONS:
        assert original[mutation['path']].decode().count(mutation['old'])==1
        test=original[mutation['suite']].decode(); assert mutation['detector'] in test and mutation['healthy'] in test
    summary=dict(sourceCommit=args.source,role=args.role,runnerAuthor='Codex /root independent REVIEWER',archiveSha256=digest(archive),archiveBytes=len(archive),members=len(original),suiteInventory=suites,baseline=baseline,commands=[],vitestInvocations=0,automaticRetries=0,engineGoInvocations=0,scratch=str(scratch),restored=False)
    if not args.execute:
        print(json.dumps(dict(source=args.source,archiveSha256=summary['archiveSha256'],members=len(original),suites=len(suites),mutations=MUTATIONS),indent=2)); return
    prefix='r067_worker_resumed'
    out=ROOT/'docs/reviews/probes'; summary_path=out/(prefix+'_summary.json')
    assert not summary_path.exists(), 'Existing raw receipt must never be overwritten'
    deps=ROOT/'frontend/node_modules'; assert (deps/'vitest/vitest.mjs').is_file()
    link=frontend/'node_modules'; assert not link.exists()
    def psquote(value): return "'"+str(value).replace("'","''")+"'"
    subprocess.run(['powershell','-NoProfile','-Command','New-Item -ItemType Junction -Path '+psquote(link)+' -Target '+psquote(deps)+' | Out-Null'],check=True,cwd=ROOT)
    env=os.environ.copy(); env['CI']='true'
    def snapshot():
        files={}
        for directory, dirs, names in os.walk(frontend, followlinks=False):
            if Path(directory)==frontend: dirs[:]=[name for name in dirs if name!='node_modules']
            for name in names:
                path=Path(directory)/name
                assert not path.is_symlink(), 'Unexpected source-tree symlink'
                relative=path.relative_to(frontend).as_posix()
                files[relative]=dict(bytes=path.stat().st_size,sha256Physical=digest(path.read_bytes()))
        return dict(sorted(files.items()))
    def run(label, suite=None, mutation=None):
        report=out/(prefix+'_'+label+'_vitest.json'); stdout=out/(prefix+'_'+label+'_stdout.log'); stderr=out/(prefix+'_'+label+'_stderr.log')
        assert not report.exists()
        before=snapshot(); exclusive_json(out/(prefix+'_'+label+'_manifest_before.json'),before)
        command=['node',str(deps/'vitest/vitest.mjs'),'run','--reporter=json','--outputFile='+str(report)]
        if suite: command+=suite
        if mutation: command+=['--testNamePattern',re.escape(mutation['detector'])+'|'+re.escape(mutation['healthy'])]
        summary['vitestInvocations']+=1; assert summary['vitestInvocations']<=3
        started=time.monotonic()
        with stdout.open('xb') as so, stderr.open('xb') as se: result=subprocess.run(command,cwd=frontend,env=env,stdout=so,stderr=se)
        elapsed=time.monotonic()-started
        entry=dict(label=label,command=command,exitCode=result.returncode,elapsedSeconds=elapsed,stdout=dict(path=str(stdout.relative_to(ROOT)),sha256Physical=digest(stdout.read_bytes())),stderr=dict(path=str(stderr.relative_to(ROOT)),sha256Physical=digest(stderr.read_bytes())))
        summary['commands'].append(entry)
        assert report.exists(), 'Vitest did not emit actual JSON'
        data=json.loads(report.read_text(encoding='utf-8')); assert isinstance(data.get('testResults'),list)
        entry.update(report=dict(path=str(report.relative_to(ROOT)),bytes=report.stat().st_size,sha256Physical=digest(report.read_bytes())),tests=data['numTotalTests'],passed=data['numPassedTests'],failed=data['numFailedTests'],pending=data['numPendingTests'])
        assertions=[a for test in data['testResults'] for a in test.get('assertionResults',[])]
        if mutation:
            detector=[a for a in assertions if a.get('title')==mutation['detector']]; healthy=[a for a in assertions if a.get('title')==mutation['healthy']]
            assert len(detector)==len(healthy)==1 and detector[0]['status']=='failed' and healthy[0]['status']=='passed'
            assert result.returncode!=0 and data['numFailedTests']==1 and data['numPassedTests']==1, 'Not a discriminating semantic kill'
            entry['semanticDetector']='failed';entry['healthyControl']='passed'
        else:
            assert result.returncode==0 and data['numFailedTests']==0 and data['numPassedTests']>0
            if label=='positive':
                actual={Path(item['name']).relative_to(frontend).as_posix() for item in data['testResults']}
                assert actual==set(suites),(actual,set(suites))
        after=snapshot(); exclusive_json(out/(prefix+'_'+label+'_manifest_after.json'),after)
        assert before==after,'Campaign command changed source'
    error=None
    try:
        summary['manualContinuation']=True; summary['historicalWorkerVitestUsed']=1
        for mutation in MUTATIONS:
            target=frontend/mutation['path']; original_text=original[mutation['path']].decode(); changed=original_text.replace(mutation['old'],mutation['new'])
            target.write_bytes(changed.encode())
            diff=dict(label=mutation['label'],sourcePath=mutation['path'],old=mutation['old'],new=mutation['new'],before=digest(original[mutation['path']]),after=digest(target.read_bytes()))
            exclusive_json(out/(prefix+'_'+mutation['label']+'_mutation.json'),diff)
            changed_snapshot=snapshot()
            assert [name for name in baseline if changed_snapshot[name]!=baseline[name]]==[mutation['path']]
            try: run(mutation['label'],[mutation['suite']],mutation)
            finally: target.write_bytes(original[mutation['path']])
            assert snapshot()==baseline,'Full member restoration failed'
        run('restored',NEW_SUITES)
        assert summary['vitestInvocations']==3 and snapshot()==baseline
        assert git('archive','--format=tar',args.source,'frontend')==archive
        summary['restored']=True;summary['secondArchiveIdentical']=True
    except BaseException as exc:
        error=repr(exc);summary['failure']=error
    finally:
        for name,data in original.items(): (frontend/name).write_bytes(data)
        summary['aggregateWorkerVitestUsed']=1+summary['vitestInvocations']
        assert summary['aggregateWorkerVitestUsed']<=4
        summary['finalFullManifestRestored']=snapshot()==baseline
        # Remove only the verified task-local junction, never recurse into cached dependencies.
        assert link.parent.resolve()==frontend.resolve() and scratch.resolve().is_relative_to(Path(tempfile.gettempdir()).resolve())
        link.rmdir();summary['dependencyJunctionRemoved']=not link.exists()
        summary['scratchRetainedForAudit']=str(scratch)
        exclusive_json(summary_path,summary)
        backup=Path(tempfile.mkdtemp(prefix=prefix+'-raw-backup-'))
        for proof in out.glob(prefix+'_*'):
            if proof.is_file(): (backup/proof.name).write_bytes(proof.read_bytes())
        print(json.dumps(dict(summary=str(summary_path.relative_to(ROOT)),rawBackup=str(backup),vitest=summary['vitestInvocations'],failure=error),indent=2))
    if error: raise SystemExit(1)
if __name__=='__main__': main()
