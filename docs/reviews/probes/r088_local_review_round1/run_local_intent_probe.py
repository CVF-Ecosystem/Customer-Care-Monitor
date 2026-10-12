"""Local R088 review: three synthetic deep-link intent probes in a disposable overlay only."""
import pathlib,shutil,subprocess,json,hashlib,datetime,time
root=pathlib.Path.cwd().resolve()
evidence=root/'docs/reviews/probes/r088_local_review_round1'
work=pathlib.Path('D:/r088-local-review-20261013-run2').resolve()
if work.exists(): raise RuntimeError('Refuse to reuse an existing overlay')
work.mkdir()
dest=work/'frontend'
shutil.copytree(root/'frontend',dest,ignore=shutil.ignore_patterns('node_modules','dist','.tmp','.vite','*.tsbuildinfo'))
quote=lambda value: "'"+str(value).replace("'","''")+"'"
subprocess.run(['powershell','-NoProfile','-Command','New-Item -ItemType Junction -Path '+quote(dest/'node_modules')+' -Target '+quote(root/'frontend/node_modules')+' | Out-Null'],check=True)
shutil.copy2(evidence/'deep-link-user-intent.spec.ts',dest/'src/__tests__/messages-lifetime-local-intent.spec.ts')
result=work/'result.json'; start=time.time()
command=['cmd','/d','/c','npm','test','--','src/__tests__/messages-lifetime-local-intent.spec.ts','--reporter=json','--outputFile='+str(result)]
proc=subprocess.run(command,cwd=dest,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,timeout=90)
data=json.loads(result.read_text(encoding='utf-8-sig')) if result.exists() else {}
tests=[]
for suite in data.get('testResults',[]):
 for t in suite.get('assertionResults',[]):
  tests.append({'title':t['fullName'],'status':t['status'],'failure':'\n'.join(t.get('failureMessages',[])).replace(str(work),'<OVERLAY>')[:1700]})
output={'recordedAtUtc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'purpose':'Resolve UI01/UI03/UI05/UI07 explicit selection/deselect versus pending initialization contradiction; three tests only, consume worker 76-test evidence without rerun','command':command,'cwd':'<OVERLAY>/frontend','exitCode':proc.returncode,'elapsedSeconds':round(time.time()-start,1),'tests':tests,'sourceLfSha256':{str(p.relative_to(root)).replace('\\','/'):hashlib.sha256(p.read_bytes().replace(b'\r\n',b'\n')).hexdigest() for p in [root/'frontend/src/views/Messages.vue',root/'frontend/src/stores/conversations.ts']},'liveFrontendWritten':False,'providerCalls':0,'backendRuns':0,'stdoutTail':proc.stdout.decode('utf-8',errors='replace').replace(str(work),'<OVERLAY>')[-1700:]}
(evidence/'deep-link-user-intent-result.json').write_text(json.dumps(output,indent=2,ensure_ascii=False)+'\n',encoding='utf-8')
print(json.dumps({'exitCode':proc.returncode,'tests':tests},ensure_ascii=True))
