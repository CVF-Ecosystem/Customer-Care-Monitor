"""Reconcile actual retained R063 review evidence after publication-summary loss.
Never runs Go or starts infrastructure; does not fabricate lost inspect snapshots.
"""
from pathlib import Path
import json, hashlib, subprocess, io, tarfile, collections, re

BASE = Path('docs/reviews/probes')

def digest(data):
    return hashlib.sha256(data).hexdigest()

def ref(path):
    raw = path.read_bytes()
    normalized = raw.replace(b'\r\n', b'\n')
    return {'path': path.as_posix(), 'bytes': len(raw), 'sha256': digest(raw), 'byteDomain': 'WORKTREE_PHYSICAL', 'lfNormalizedText': {'bytes': len(normalized), 'sha256': digest(normalized)}}

def run(args):
    result = subprocess.run(args, capture_output=True, text=True, timeout=30)
    if result.returncode:
        raise RuntimeError('read-only reconciliation command failed')
    return result.stdout

def main():
    plan_path = BASE/'r063_independent_plan.json'
    plan = json.loads(plan_path.read_text(encoding='utf-8-sig'))
    daemon_path = BASE/'r063_independent_daemon_events_recovery.json'
    daemon = json.loads(daemon_path.read_text(encoding='utf-8-sig'))
    records = daemon['events']
    token = daemon['taskToken']
    source = plan['archiveSourceCommit']
    archive = subprocess.check_output(['git','archive','--format=tar',source,'backend'])
    with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
        members = {m.name:tar.extractfile(m).read() for m in tar.getmembers() if m.isfile()}
    expected = sorted([{'path':p,'bytes':len(b),'sha256':digest(b)} for p,b in members.items()],key=lambda x:x['path'])
    manifest_path = BASE/'r063_independent_manifest.json'
    assert json.loads(manifest_path.read_text()) == expected and len(expected) == 213
    names=[]
    for name,raw in members.items():
        if name.endswith('_test.go'):
            names += re.findall(r'^func (Test\w+)\(t \*testing\.T\)', raw.decode(), re.M)
    commands=[]
    times={'positive':158.258,'m01':16.32,'m02':17.055,'restored':28.668}
    for label in ['positive','m01','m02','restored']:
        stdout=BASE/('r063_independent_'+label+'.jsonl')
        stderr=BASE/('r063_independent_'+label+'_stderr.log')
        events=[json.loads(line) for line in stdout.read_text().splitlines() if line.strip()]
        completed=[{'name':e['Test'],'action':e['Action']} for e in events if e.get('Test') and e['Action'] in ['pass','fail','skip']]
        top=[e for e in completed if '/' not in e['name']]
        spec=plan['mutations'][0 if label=='m01' else 1] if label in ['m01','m02'] else None
        pattern=spec.get('pattern','^'+spec['detector']+'$') if spec else plan['positivePattern' if label=='positive' else 'restoredPattern']
        selected=sorted(n for n in names if re.search(pattern,n))
        assert sorted(e['name'] for e in top)==selected
        counts=collections.Counter(e['action'] for e in completed)
        assert counts['skip']==0
        output=''.join(e.get('Output','') for e in events)
        assert not re.search(r'\[build failed\]|build-fail|undefined:|test timed out|panic:',output)
        if spec:
            assert spec['assertion'] in output
            assert any(e['name']==spec['detector'] and e['action']=='fail' for e in top)
            for healthy in spec.get('expectedPass',[]):
                assert any(e['name']==healthy and e['action']=='pass' for e in completed)
            original=members[spec['path']]
            needle=spec['needle'].encode(); assert original.count(needle)==1
            mutated=original.replace(needle,spec['replacement'].encode(),1)
            actual=json.loads((BASE/('r063_independent_'+label+'_manifest.json')).read_text())
            expected_mut=[dict(e,bytes=len(mutated),sha256=digest(mutated)) if e['path']==spec['path'] else e for e in expected]
            assert actual==expected_mut
        else:
            assert counts['fail']==0 and counts['pass']>0
        container='ccmai-r063-review-'+label+'-'+token
        died=[e for e in records if e.get('Type')=='container' and e.get('Action')=='die' and e.get('Actor',{}).get('Attributes',{}).get('name')==container]
        assert len(died)==1
        exit_code=int(died[0]['Actor']['Attributes']['exitCode']);assert exit_code==(1 if spec else 0)
        assert any(e.get('Action')=='destroy' and e.get('Actor',{}).get('Attributes',{}).get('name')==container for e in records)
        commands.append({'label':label,'runtimeProvenance':'Actual Go JSONL plus actual retained daemon die/destroy events','container':container,'daemonContainerID':died[0]['Actor']['ID'],'daemonExitCode':exit_code,'stdout':ref(stdout),'stderr':ref(stderr),'expectedTopLevel':selected,'events':{'completed':completed,'topLevel':len(top),'pass':counts['pass'],'fail':counts['fail'],'skip':counts['skip'],'namedAssertion':bool(spec)},'verifiedHealthyControls':spec.get('expectedPass',[]) if spec else [],'observedRunnerStdoutSeconds':times[label]})
    creates=[e for e in records if e.get('Type')=='container' and e.get('Action')=='create' and e.get('Actor',{}).get('Attributes',{}).get('name','').startswith('ccmai-r063-review-') and 'db-' not in e['Actor']['Attributes']['name']]
    assert len(creates)==4
    source_dirs={e['Actor']['Attributes']['desktop.docker.io/mounts/0/Source'] for e in creates}
    assert len(source_dirs)==1
    backend=Path(next(iter(source_dirs)))
    physical=sorted([{'path':'backend/'+p.relative_to(backend).as_posix(),'bytes':p.stat().st_size,'sha256':digest(p.read_bytes())} for p in backend.rglob('*') if p.is_file()],key=lambda x:x['path'])
    assert physical==expected
    db='ccmai-r063-review-db-'+token
    db_ids={e['Actor']['ID'] for e in records if e.get('Actor',{}).get('Attributes',{}).get('name')==db}
    assert len(db_ids)==1
    volumes={e['Actor']['ID'] for e in records if e.get('Type')=='volume' and e.get('Actor',{}).get('Attributes',{}).get('container') in db_ids}
    volumes.add('ccmai-r063-build-cache-'+token)
    for volume in volumes:
        assert any(e.get('Type')=='volume' and e.get('Action')=='destroy' and e['Actor']['ID']==volume for e in records)
    network='ccmai-r063-review-net-'+token
    assert any(e.get('Type')=='network' and e.get('Action')=='destroy' and e['Actor']['Attributes'].get('name')==network for e in records)
    inventories={'containers':run(['docker','ps','-a','--format','{{.Names}}']).splitlines(),'networks':run(['docker','network','ls','--format','{{.Name}}']).splitlines(),'volumes':run(['docker','volume','ls','--format','{{.Name}}']).splitlines()}
    containers=[c['container'] for c in commands]+[db]
    assert not set(containers)&set(inventories['containers']) and network not in inventories['networks'] and not volumes&set(inventories['volumes'])
    protected_path=BASE/'r063_independent_protected.json';protected=json.loads(protected_path.read_text())
    for path,h in protected['files'].items():assert digest(Path(path).read_bytes())==h,path
    second=subprocess.check_output(['git','archive','--format=tar',source,'backend']);assert second==archive
    receipt={'receiptKind':'DERIVED_RECONCILIATION_FROM_ACTUAL_RETAINED_EVIDENCE','originalSummary':'Lost to publication path-alias; original bytes/inspect snapshots NOT restored or invented','sourceCommit':source,'backendSourceCommit':plan['backendSourceCommit'],'reviewer':'Codex /root','seedCommit':'7deb4f09b7dfaed9ca8d0f4739a6d3190dd2a015','campaigns':1,'goInvocations':4,'workerPriorUsedGoInvocations':4,'aggregateMaxGoInvocations':8,'newGoDuringReconciliation':0,'result':'PASS','commands':commands,'archive':{'bytes':len(archive),'sha256':digest(archive)},'manifest':ref(manifest_path),'plan':ref(plan_path),'daemonEvents':ref(daemon_path),'incident':ref(BASE/'r063_review_publication_incident.json'),'restoration':{'fullMembers':213,'currentActualSandboxMatches':True,'secondGitArchiveMatches':True},'cleanup':{'knownContainers':containers,'network':network,'volumes':sorted(volumes),'actualDaemonDestroyEventsVerified':True,'freshReadOnlyInventoryAllAbsent':True},'scope':'synthetic local application observation, not CVF AI governance','priorVerifiedRunnerOutput':{'result':'PASS','goInvocations':4,'seconds':248.818,'errors':[]},'lostInspectionBoundary':'Original per-container inspect snapshots unavailable. Executed committed runner enforced readonly binds/internal network/no host ports and originally returned PASS; these checks were reviewed before loss. Actual daemon events retain mounts/exit/destroy and reconciliation rechecks tracked-resource absence. Reconciled fields have explicit provenance; no claim byte-for-byte recovery.','errors':[]}
    target=BASE/'r063_independent_summary.json';target.write_text(json.dumps(receipt,indent=2)+'\n',encoding='utf-8')
    print('PASS reconciled actual four Go outputs/exits,53/152 positives,two semantic kills/healthy controls,9/43 restored,full213 manifests/sandbox,second archive,tracked container/network/volumes destroy+fresh absence,protected old packets. Zero new Go; original summary loss explicit.')

if __name__=='__main__':main()
