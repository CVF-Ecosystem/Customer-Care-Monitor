import pathlib,json,subprocess,hashlib,shutil,time,sys,os
root=pathlib.Path.cwd(); src=sys.argv[1]; role='root'
head=subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip()
assert subprocess.run(['git','merge-base','--is-ancestor',src,'HEAD']).returncode==0
assert not subprocess.check_output(['git','diff',src,'--','frontend/src'],text=True).strip()
baseline=json.loads((root/'docs/reviews/probes/r073_protected_baseline_2026-10-10.json').read_text(encoding='utf-8-sig'))
allowed=['frontend/src/components/ui/RunObservationPanel.vue','frontend/src/i18n/en.ts','frontend/src/i18n/vi.ts']
changed=[s for s,h in baseline['files'].items() if hashlib.sha256((root/s).read_bytes()).hexdigest()!=h]
assert set(changed)==set(allowed),changed
paths=subprocess.check_output(['git','ls-files','frontend/src'],text=True).splitlines()
def inventory():return {s:hashlib.sha256((root/s).read_bytes()).hexdigest() for s in paths}
before=inventory()
packet=root/'docs/reviews/probes/r073_root_runtime_2026-10-10';packet.mkdir(exist_ok=False)
node=shutil.which('node'); assert node
summary={'sourceCommit':src,'repositoryHead':head,'sourceDiffAgainstCommit':'EMPTY frontend/src','role':role,'captureAuthor':'Codex /root','nodeExecutable':node,'cwd':str(root/'frontend'),'baselineChanged':changed,'beforePhysicalSha256':before,'scope':'Saved adapter-presence UI projection and mocked happy-dom rendering; parsed safe integers, no governance/billing/live evidence','attempts':[]}
fixtures=['src/__tests__/run-observation.spec.ts','src/__tests__/run-observation-panel.spec.ts','src/__tests__/run-observation-job-detail.spec.ts','src/__tests__/run-observation-adapter-compat.spec.ts','src/__tests__/adapter-usage-presence.spec.ts']
summary['environment']={'platform':sys.platform,'nodeVersionObservedBeforeCapture':'v24.19.0','installedPackages':{n:json.loads((root/'frontend/node_modules'/n/'package.json').read_text(encoding='utf-8'))['version'] for n in ['vitest','happy-dom','vue-tsc','vue','vue-i18n']},'testEnvironment':'Existing Node projector specs; mocked happy-dom panel/job-detail/NEW presence spec; no real browser','inheritedProcessEnvironment':'Not fully serialized; no secrets/config copied'}
commands=[('vitest',[node,'node_modules/vitest/vitest.mjs','run',*fixtures,'--reporter=json','--outputFile='+str(packet/'vitest.report.json')]),('typecheck',[node,'node_modules/vue-tsc/bin/vue-tsc.js','-b','--force','--pretty','false'])]
for name,cmd in commands:
 started=time.perf_counter(); entry={'name':name,'argv':cmd,'startedUtc':time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime()),'exitCode':None,'timeout':False}
 try:
  child=subprocess.Popen(cmd,cwd=root/'frontend',stdout=subprocess.PIPE,stderr=subprocess.PIPE,env=dict(os.environ));entry['pid']=child.pid
  try:out,err=child.communicate(timeout=240)
  except subprocess.TimeoutExpired:
   entry['timeout']=True; subprocess.run(['taskkill','/PID',str(child.pid),'/T','/F'],capture_output=True);out,err=child.communicate()
  entry['exitCode']=child.returncode
 except Exception as ex:out=b'';err=(type(ex).__name__+': '+str(ex)).encode();entry['launchFailure']=True
 (packet/(name+'.stdout.bin')).write_bytes(out);(packet/(name+'.stderr.bin')).write_bytes(err)
 entry['elapsedNativeSeconds']=time.perf_counter()-started;summary['attempts'].append(entry)
 (packet/'capture.json').write_text(json.dumps(summary,indent=2)+'\n',encoding='utf-8')
 if entry['exitCode']!=0 or entry['timeout']:break
summary['afterPhysicalSha256']=inventory();summary['sourceUnchanged']=summary['afterPhysicalSha256']==before
report=packet/'vitest.report.json'
if report.exists():
 r=json.loads(report.read_text(encoding='utf-8-sig'));summary['vitestCounts']={k:r.get(k) for k in ['numTotalTests','numPassedTests','numFailedTests','numPendingTests','numTodoTests','success']};summary['testStatuses']={x['name']:[t['status'] for t in x['assertionResults']] for x in r['testResults']}
summary['actualTestFileCount']=len(summary.get('testStatuses',{}));summary['expectedTestFileCount']=len(fixtures)
summary['verdict']='PASS' if len(summary['attempts'])==2 and all(x['exitCode']==0 and not x['timeout'] for x in summary['attempts']) and summary['sourceUnchanged'] and summary.get('vitestCounts',{}).get('success') and not summary.get('vitestCounts',{}).get('numFailedTests') and not summary.get('vitestCounts',{}).get('numPendingTests') and not summary.get('vitestCounts',{}).get('numTodoTests') and summary['actualTestFileCount']==len(fixtures) else 'NOT_ACCEPTED'
(packet/'capture.json').write_text(json.dumps(summary,indent=2)+'\n',encoding='utf-8')
print(json.dumps({k:summary[k] for k in ['sourceCommit','baselineChanged','vitestCounts','verdict','sourceUnchanged']},indent=2));sys.exit(0 if summary['verdict']=='PASS' else 1)
