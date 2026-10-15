"""Bounded R101 independent human-software verification; no provider/config access."""
import hashlib,json,os,pathlib,re,subprocess,sys,time
R=pathlib.Path.cwd(); E=R/'docs/reviews/probes/r101_root'; T=R/'.cvf/runtime/r101_root'; T.mkdir(parents=True,exist_ok=True)
NAME='ccmai-r101-review-final-20261015'; LABEL='ccmai.tranche=CCMAI-RUNTIME-101'; PASS='r101LocalOnly42'; DB='r101_fixture'
BINARY=R/'.cvf/runtime/r101_worker/r101_handlers.test.exe'; SHA='4c7575e180dd198d995bfe09bdc45061740a987be8ec0765f02b203e1c251f31'
def write(name,data): (E/name).write_text(json.dumps(data,indent=2)+'\n',encoding='utf-8')
def run(args,check=True,**kw):
 p=subprocess.run(args,cwd=R,capture_output=True,text=True,encoding='utf-8',errors='replace',**kw)
 if check and p.returncode: raise RuntimeError('command failed: '+str(args[:3])+' '+p.stderr[:300])
 return p
def verify_source():
 m=json.loads((E/'formatted_source_manifest.json').read_text())
 assert all(hashlib.sha256((R/f['path']).read_bytes()).hexdigest()==f['sha256'] for f in m['files'])
 assert hashlib.sha256(BINARY.read_bytes()).hexdigest()==SHA
def inspect(kind,name,fmt):return json.loads(run(['docker',kind,'inspect',name,'--format',fmt]).stdout)
FMT='{"id":{{json .Id}},"name":{{json .Name}},"image":{{json .Image}},"status":{{json .State.Status}},"mounts":{{json .Mounts}}}'
def inventory():
 ids=run(['docker','ps','-aq']).stdout.split()
 return [json.loads(run(['docker','inspect',i,'--format',FMT]).stdout) for i in ids if run(['docker','inspect',i,'--format','{{.Name}}']).stdout.strip()!='/'+NAME]
def owned():
 o=json.loads((E/'fixture_owner.json').read_text())
 fmt='{"id":{{json .Id}},"name":{{json .Name}},"image":{{json .Image}},"labels":{{json .Config.Labels}},"mounts":{{json .Mounts}},"ports":{{json .NetworkSettings.Ports}},"networks":{{json .NetworkSettings.Networks}}}'
 c=json.loads(run(['docker','inspect',NAME,'--format',fmt]).stdout)
 assert c['id']==o['containerId'] and c['name']=='/'+NAME and c['image']==o['imageId']
 assert c['labels']['ccmai.tranche']=='CCMAI-RUNTIME-101'
 assert list(c['networks'])==[NAME]
 assert c['ports']['3306/tcp']==[{'HostIp':'127.0.0.1','HostPort':o['port']}]
 assert any(m['Type']=='volume' and m['Name']==NAME and m['Destination']=='/var/lib/mysql' for m in c['mounts'])
 return o
def schema(o):
 p=run(['docker','exec','-e','MYSQL_PWD='+PASS,o['containerId'],'mysql','--protocol=TCP','-h127.0.0.1','-uroot','--batch','--skip-column-names','-e',"SELECT SCHEMA_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME='r101_fixture'"])
 assert p.stdout.strip()==DB
def launch_counter(kind):
 p=E/'review_counters.json'; d=json.loads(p.read_text()) if p.exists() else {'workerGo':1,'reviewerGo':0,'aggregateGo':1,'fixtureReservations':2,'nodeChecks':3,'frontendBuilds':1,'providerCalls':0,'mockProviderCalls':0,'launches':[]}
 assert d['reviewerGo']<6 and d['aggregateGo']<16
 d['reviewerGo']+=1;d['aggregateGo']+=1;d['launches'].append(kind);write(p.name,d)
if sys.argv[1]=='setup':
 verify_source(); before=inventory();write('protected_final_baseline.json',before)
 for kind in ['container','network','volume']:
  p=run(['docker',kind,'inspect',NAME],False); assert p.returncode!=0
 image=run(['docker','image','inspect','mysql:8.0','--format','{{.Id}}']).stdout.strip()
 write('fixture_final_reservation.json',{'fixtureReservations':2,'name':NAME,'imageId':image,'source':'200df3e','providerCalls':0,'created':False})
 net=run(['docker','network','create','--driver','bridge','--opt','com.docker.network.bridge.host_binding_ipv4=127.0.0.1','--label',LABEL,NAME]).stdout.strip()
 volume=run(['docker','volume','create','--label',LABEL,NAME]).stdout.strip()
 cid=run(['docker','run','-d','--pull','never','--name',NAME,'--label',LABEL,'--network',NAME,'--mount','type=volume,source='+NAME+',target=/var/lib/mysql','-p','127.0.0.1::3306','-e','MYSQL_ROOT_PASSWORD='+PASS,'-e','MYSQL_DATABASE='+DB,image,'--skip-name-resolve']).stdout.strip()
 ports={}
 for _ in range(30):
  ports=json.loads(run(['docker','inspect',cid,'--format','{{json .NetworkSettings.Ports}}']).stdout)
  if ports.get('3306/tcp'):break
  time.sleep(1)
 assert ports.get('3306/tcp'), 'No published port; final fixture cannot be recreated'
 port=ports['3306/tcp'][0]['HostPort']
 o={'containerId':cid,'networkId':net,'volumeName':volume,'name':NAME,'port':port,'database':DB,'imageId':image,'label':LABEL,'loopbackOnly':True}
 write('fixture_owner.json',o); owned()
 ready=False
 for _ in range(90):
  p=run(['docker','exec','-e','MYSQL_PWD='+PASS,cid,'mysql','--protocol=TCP','-h127.0.0.1','-uroot','--batch','--skip-column-names','-e','SELECT 1'],False)
  if p.returncode==0 and p.stdout.strip()=='1':ready=True;break
  time.sleep(1)
 assert ready;schema(o);assert inventory()==before
 write('fixture_ready.json',dict(o,ready=True,protectedIdentityUnchanged=True));print('Owned loopback fixture ready on '+port)
elif sys.argv[1]=='ready_existing':
 verify_source();o=owned();schema(o);assert inventory()==json.loads((E/'protected_final_baseline.json').read_text())
 write('fixture_ready.json',dict(o,ready=True,probe='TCP schema on final server',protectedIdentityUnchanged=True));print('Same owned fixture ready; no recreation')
elif sys.argv[1]=='native':
 verify_source();o=owned();schema(o)
 env=dict(os.environ,R101_MYSQL_DSN=f'root:{PASS}@tcp(127.0.0.1:{o["port"]})/{DB}?parseTime=true&charset=utf8mb4')
 names=['HumanWorkflowNative','WaitDataAndAtomicity','CompetingReviewers','SourceAndPermissionConflicts','SourceLockSerialization','PermissionLockSerialization']
 cmd=[str(BINARY),'-test.run','^TestR101('+'|'.join(names)+')$','-test.count=1','-test.v','-test.timeout=180s']
 launch_counter('six_native_human_tests')
 p=run(cmd,False,env=env,timeout=195)
 (E/'native_stdout.log').write_text(p.stdout,encoding='utf-8');(E/'native_stderr.log').write_text(p.stderr,encoding='utf-8')
 passed=all(re.search(r'^--- PASS: TestR101'+n+r'\b',p.stdout,re.M) for n in names)
 ok=p.returncode==0 and passed and '--- SKIP:' not in p.stdout and '--- FAIL:' not in p.stdout
 write('native_result.json',{'command':cmd,'cwd':str(R),'exit':p.returncode,'requiredTopLevel':names,'requiredAllPassed':passed,'noSkip': '--- SKIP:' not in p.stdout,'accepted':ok,'binarySha256':SHA,'sourceCommit':'200df3e4a7bc2a326425922a5230508a4fd9b0fe','providerCalls':0,'mockProviderCalls':0})
 print(json.dumps({'exit':p.returncode,'accepted':ok,'requiredAllPassed':passed}));print(p.stdout[-2500:]);assert ok
elif sys.argv[1]=='serve':
 verify_source();o=owned();schema(o);launch_counter('real_router_browser_fixture')
 env=dict(os.environ,R101_MYSQL_DSN=f'root:{PASS}@tcp(127.0.0.1:{o["port"]})/{DB}?parseTime=true&charset=utf8mb4',R101_BROWSER_FIXTURE='1')
 cmd=[str(BINARY),'-test.run','^TestR101BrowserFixture$','-test.v','-test.timeout=35m']
 out=(T/'server_stdout.log').open('w',encoding='utf-8');err=(T/'server_stderr.log').open('w',encoding='utf-8')
 p=subprocess.Popen(cmd,cwd=R,env=env,stdout=out,stderr=err,creationflags=subprocess.CREATE_NO_WINDOW)
 write('browser_process.json',{'pid':p.pid,'command':cmd,'binarySha256':SHA,'fixtureContainerId':o['containerId'],'sourceCommit':'200df3e','providerCalls':0})
 for _ in range(60):
  text=(T/'server_stdout.log').read_text(encoding='utf-8')
  match=re.search(r'R101_BROWSER_READY (http://127\.0\.0\.1:\d+) tenant=([^ ]+) case=([^ ]+) reader=([^\s]+)',text)
  owner=re.search(r'R101_BROWSER_OWNER ([^\s]+)',text)
  if match and owner:
   d={'apiUrl':match[1],'tenant':match[2],'case':match[3],'reader':match[4],'ownerEmail':owner[1],'pid':p.pid,'syntheticFixture':True};write('browser_ready.json',d);print(json.dumps(d));break
  assert p.poll() is None;time.sleep(1)
 else:raise RuntimeError('browser fixture did not become ready')
