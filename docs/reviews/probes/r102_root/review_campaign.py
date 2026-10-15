"""Root-owned R102 local software verification, inert until source admission."""
import hashlib, json, os, pathlib, re, subprocess, sys, time

R=pathlib.Path.cwd(); E=R/'docs/reviews/probes/r102_root'; T=R/'.cvf/runtime/r102_root'
NAME='ccmai-r102-review-20261015'; LABEL='ccmai.tranche=CCMAI-RUNTIME-102'
IMAGE='sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b'
DB='r102_fixture'; FIXTURE_PASS='r102LocalOnly42'

def write(name,obj):
    E.mkdir(parents=True,exist_ok=True)
    (E/name).write_text(json.dumps(obj,indent=2)+'\n',encoding='utf-8')

def read(name):return json.loads((E/name).read_text(encoding='utf-8-sig'))

def run(args,check=True,**kwargs):
    p=subprocess.run(args,cwd=R,capture_output=True,text=True,encoding='utf-8',errors='replace',**kwargs)
    if check and p.returncode:raise RuntimeError('probe failed: '+str(args[:3])+' '+p.stderr[:200])
    return p

def source():
    a=read('source_admission.json'); assert a['admitted'] is True
    for f in a['files']:assert hashlib.sha256((R/f['path']).read_bytes()).hexdigest()==f['sha256'],f['path']
    return a

def safe_inventory():
    ids=run(['docker','ps','-aq']).stdout.split(); result=[]
    fmt='{"id":{{json .Id}},"name":{{json .Name}},"image":{{json .Image}},"status":{{json .State.Status}},"mounts":{{json .Mounts}}}'
    for cid in ids:
        obj=json.loads(run(['docker','inspect',cid,'--format',fmt]).stdout)
        if obj['name']!='/'+NAME:result.append(obj)
    return sorted(result,key=lambda x:x['id'])

def owned():
    o=read('fixture_owner.json')
    fmt='{"id":{{json .Id}},"name":{{json .Name}},"image":{{json .Image}},"status":{{json .State.Status}},"labels":{{json .Config.Labels}},"mounts":{{json .Mounts}},"ports":{{json .NetworkSettings.Ports}},"networks":{{json .NetworkSettings.Networks}}}'
    c=json.loads(run(['docker','inspect',o['containerId'],'--format',fmt]).stdout)
    assert c['id']==o['containerId'] and c['name']=='/'+NAME and c['image']==IMAGE
    assert c['labels'].get('ccmai.tranche')=='CCMAI-RUNTIME-102'
    assert list(c['networks'])==[NAME] and c['networks'][NAME]['NetworkID']==o['networkId']
    assert any(m['Type']=='volume' and m['Name']==NAME and m['Destination']=='/var/lib/mysql' for m in c['mounts'])
    net=json.loads(run(['docker','network','inspect',o['networkId'],'--format','{"labels":{{json .Labels}},"containers":{{json .Containers}}}']).stdout)
    assert net['labels'].get('ccmai.tranche')=='CCMAI-RUNTIME-102'
    assert set(net['containers']).issubset({o['containerId']}),'unowned network attachment'
    vol=json.loads(run(['docker','volume','inspect',NAME,'--format','{{json .Labels}}']).stdout)
    assert vol.get('ccmai.tranche')=='CCMAI-RUNTIME-102'
    if c['status']=='running':assert c['ports'].get('3306/tcp')==[{'HostIp':'127.0.0.1','HostPort':o['port']}]
    return o,c

def mysql(o,sql):
    return run(['docker','exec','-e','MYSQL_PWD='+FIXTURE_PASS,o['containerId'],'mysql','--protocol=TCP','-h127.0.0.1','-uroot','--batch','--skip-column-names','-e',sql],False)

mode=sys.argv[1]
if mode=='setup':
    source(); assert not (E/'fixture_reservation.json').exists(),'one fixture slot already reserved; no recreation'
    for kind in ['container','network','volume']:
        assert run(['docker',kind,'inspect',NAME],False).returncode!=0,'name already exists'
    assert run(['docker','image','inspect',IMAGE,'--format','{{.Id}}']).stdout.strip()==IMAGE
    before=safe_inventory();write('protected_baseline.json',before)
    o={'name':NAME,'imageId':IMAGE,'database':DB,'label':LABEL,'fixtureReservations':1}
    write('fixture_reservation.json',o)
    o['networkId']=run(['docker','network','create','--driver','bridge','--opt','com.docker.network.bridge.host_binding_ipv4=127.0.0.1','--label',LABEL,NAME]).stdout.strip();write('fixture_reservation.json',o)
    o['volumeName']=run(['docker','volume','create','--label',LABEL,NAME]).stdout.strip();write('fixture_reservation.json',o)
    o['containerId']=run(['docker','run','-d','--pull','never','--name',NAME,'--label',LABEL,'--network',NAME,'--mount','type=volume,source='+NAME+',target=/var/lib/mysql','-p','127.0.0.1::3306','-e','MYSQL_ROOT_PASSWORD='+FIXTURE_PASS,'-e','MYSQL_DATABASE='+DB,IMAGE,'--skip-name-resolve']).stdout.strip();write('fixture_reservation.json',o)
    p=json.loads(run(['docker','inspect',o['containerId'],'--format','{{json .NetworkSettings.Ports}}']).stdout)
    assert p.get('3306/tcp') and p['3306/tcp'][0]['HostIp']=='127.0.0.1','loopback mapping missing'
    o['port']=p['3306/tcp'][0]['HostPort'];write('fixture_owner.json',o);owned()
    for _ in range(100):
        p=mysql(o,"SELECT SCHEMA_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME='r102_fixture'")
        if p.returncode==0 and p.stdout.strip()==DB:break
        time.sleep(1)
    else:raise RuntimeError('final TCP schema not ready; no recreation')
    assert safe_inventory()==before,'protected snapshot changed'
    write('fixture_ready.json',dict(o,ready=True,probe='TCP schema',protectedSafeFieldsEqual=True))
    print('Same single R102 fixture ready on 127.0.0.1:'+o['port'])
elif mode=='test':
    a=source();o,c=owned();assert c['status']=='running'
    role=sys.argv[2];assert role in ('worker','reviewer')
    counters=read('counters.json');limit=4 if role=='worker' else 3
    assert counters[role+'Go']<limit and counters['aggregateGo']<8
    binary=R/sys.argv[3]; pattern=sys.argv[4]
    expected=json.loads((R/'.cvf/runtime/r102_binary_manifest.json').read_text())[str(binary.relative_to(R)).replace('\\','/')]
    assert hashlib.sha256(binary.read_bytes()).hexdigest()==expected
    counters[role+'Go']+=1;counters['aggregateGo']+=1
    counters['launches'].append({'role':role,'kind':'native_binary','pattern':pattern});write('counters.json',counters)
    env=dict(os.environ,R102_MYSQL_DSN=f'root:{FIXTURE_PASS}@tcp(127.0.0.1:{o["port"]})/{DB}?parseTime=true&charset=utf8mb4')
    cmd=[str(binary),'-test.run',pattern,'-test.count=1','-test.v','-test.failfast','-test.timeout=240s']
    p=run(cmd,False,env=env,timeout=255)
    tag=role+'_launch_'+str(counters[role+'Go'])
    (E/(tag+'_stdout.log')).write_text(p.stdout,encoding='utf-8');(E/(tag+'_stderr.log')).write_text(p.stderr,encoding='utf-8')
    test_source='backend/api/handlers/intervention_waitdata_test.go' if 'handlers' in binary.name else 'backend/engine/intervention_waitdata_test.go'
    required=[n for n in re.findall(r'^func (TestR102\w+)\(', (R/test_source).read_text(encoding='utf-8-sig'),re.M) if re.search(pattern,n)]
    missing=[n for n in required if not re.search(r'^--- PASS: '+re.escape(n)+r'\b',p.stdout,re.M)]
    ok=p.returncode==0 and bool(required) and not missing and '--- SKIP:' not in p.stdout and '--- FAIL:' not in p.stdout
    write(tag+'_result.json',{'command':cmd,'exit':p.returncode,'accepted':ok,'requiredTopLevel':required,'missingPasses':missing,'noSkip':'--- SKIP:' not in p.stdout,'binarySha256':expected,'source':a,'providerCalls':0,'mockProviderCalls':0})
    print(json.dumps({'exit':p.returncode,'accepted':ok,'topPass':len(re.findall(r'^--- PASS:',p.stdout,re.M))}));print(p.stdout[-1800:]);assert ok
elif mode=='stop':
    o,c=owned()
    if c['status']=='running':run(['docker','stop','-t','15',o['containerId']])
    o,c=owned();assert c['status']=='exited'
    equal=safe_inventory()==read('protected_baseline.json')
    write('cleanup_checkpoint.json',{'containerId':o['containerId'],'state':c['status'],'networkId':o['networkId'],'volumeName':o['volumeName'],'preserved':True,'protectedSafeFieldsEqual':equal})
    print(json.dumps({'ownedFixtureStopped':True,'preserved':True,'protectedSafeFieldsEqual':equal}))
else:raise ValueError('unknown finite campaign action')
