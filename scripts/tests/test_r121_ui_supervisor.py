"""Offline software checks only; no Docker, Go, browser, or provider effect."""
import importlib.util
import json
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

SCRIPT = Path(__file__).resolve().parents[1] / 'r121_ui_supervisor.py'
spec = importlib.util.spec_from_file_location('r121_ui_supervisor', SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class FakeClock:
    def __init__(self):
        self.now = 0.0
        self.sleeps = 0

    def monotonic(self):
        return self.now

    def sleep(self, seconds):
        self.sleeps += 1
        if self.sleeps >= 3:
            self.now = module.MYSQL_READY_SECONDS
        else:
            self.now += seconds


class FakeSocket:
    def __init__(self, reads):
        self.reads = list(reads)
        self.timeouts = []

    def settimeout(self, timeout):
        self.timeouts.append(timeout)

    def recv(self, size):
        value = self.reads.pop(0)
        if isinstance(value, BaseException):
            raise value
        return value[:size]

    def __enter__(self):
        return self

    def __exit__(self, *_args):
        return False


class R121SupervisorTests(unittest.TestCase):
    def test_campaign_ready_stdout_is_ascii_safe_on_cp1252_and_readiness_follows_output(self):
        with tempfile.TemporaryDirectory() as directory:
            runtime = Path(directory) / 'runtime'
            code = '''
import importlib.util, json, sys
from pathlib import Path
spec = importlib.util.spec_from_file_location("r121_ui_supervisor", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
module.RUNTIME = Path(sys.argv[2])
module.RUNTIME.mkdir(parents=True)
sys.stdout.reconfigure(encoding="cp1252", errors="strict")
write_new = module.write_new
def tracked_write(path, value):
    if Path(path).name == "campaign_ready.json":
        print("READY_FILE_WRITE", flush=True)
    return write_new(path, value)
module.write_new = tracked_write
manifest = {"tranche":"CCMAI-RUNTIME-121", "synthetic":True, "note":"Cần kiểm tra hội thoại"}
module.publish_campaign_ready("http://127.0.0.1:4321", manifest)
print("SUPERVISOR_CONTINUES", flush=True)
'''
            completed = subprocess.run(
                [sys.executable,'-c',code,str(SCRIPT),str(runtime)],
                capture_output=True,
                timeout=10,
                env={**module.minimal_env(),'PYTHONIOENCODING':'cp1252:strict'},
            )
            self.assertEqual(completed.returncode,0,completed.stderr.decode('ascii','replace'))
            self.assertTrue(completed.stdout.isascii())
            lines = completed.stdout.decode('ascii').splitlines()
            self.assertEqual(lines[1:],["READY_FILE_WRITE",'SUPERVISOR_CONTINUES'])
            announced = json.loads(lines[0])
            ready = json.loads((runtime/'campaign_ready.json').read_text(encoding='utf-8'))
            self.assertEqual(announced['result'],'CAMPAIGN_READY')
            self.assertEqual(announced['manifest']['note'],'Cần kiểm tra hội thoại')
            self.assertEqual(ready['manifest'],announced['manifest'])

    def test_published_port_requires_one_complete_loopback_binding(self):
        self.assertEqual(module.published_port({'3306/tcp':[{'HostIp':'127.0.0.1','HostPort':'49152'}]}),49152)
        for ports in ({}, {'3306/tcp':None}, {'3306/tcp':[]},
                      {'3306/tcp':[{'HostIp':'0.0.0.0','HostPort':'49152'}]},
                      {'3306/tcp':[{'HostIp':'127.0.0.1','HostPort':'0'}]},
                      {'3306/tcp':[{'HostIp':'127.0.0.1','HostPort':'49152'}],'other':[]},
                      {'3306/tcp':[{'HostIp':'127.0.0.1','HostPort':'49152'}, {'HostIp':'127.0.0.1','HostPort':'49153'}]}):
            with self.subTest(ports=ports), self.assertRaises(module.Hold): module.published_port(ports)

    def test_mysql_greeting_accumulates_fragmented_header_with_bounded_reads(self):
        fake = FakeSocket([b'\x01\x00',b'\x00\x00\x0a'])
        with patch.object(module.time,'monotonic',side_effect=[4.6,4.7,4.8]):
            greeting = module.read_mysql_greeting(fake,5.0)
        self.assertEqual(greeting,b'\x01\x00\x00\x00\x0a')
        self.assertEqual(len(fake.timeouts),2)
        self.assertAlmostEqual(fake.timeouts[0],0.4)
        self.assertAlmostEqual(fake.timeouts[1],0.3)

    def test_mysql_greeting_deadline_preserves_partial_byte_count(self):
        fake = FakeSocket([b'\x01\x00'])
        with patch.object(module.time,'monotonic',side_effect=[0.0,1.0]):
            with self.assertRaises(module.MySQLGreetingTimeout) as caught:
                module.read_mysql_greeting(fake,1.0)
        self.assertEqual(caught.exception.header_byte_count,2)

    def test_mysql_readiness_retries_eof_then_accepts_complete_greeting(self):
        clock = FakeClock()
        sockets = [ConnectionRefusedError('not listening yet'),FakeSocket([b'']),FakeSocket([b'\x01\x00',b'\x00\x00\x0a'])]
        with tempfile.TemporaryDirectory() as directory, \
             patch.object(module,'RUNTIME',Path(directory)/'runtime'), \
             patch.object(module.socket,'create_connection',side_effect=sockets) as connect, \
             patch.object(module.time,'monotonic',side_effect=clock.monotonic), \
             patch.object(module.time,'sleep',side_effect=clock.sleep), \
             patch.object(module.time,'time',return_value=123.0):
            result = module.wait_mysql(49152)
            rows = [json.loads(line) for line in (Path(directory)/'runtime'/'mysql_readiness.jsonl').read_text(encoding='utf-8').splitlines()]
        self.assertEqual(connect.call_count,3)
        self.assertEqual([row['result'] for row in rows],['transient_connect_error','transient_eof','ready'])
        self.assertEqual([row['headerByteCount'] for row in rows],[0,0,5])
        self.assertEqual(result['status'],'READY')
        self.assertEqual(result['attemptCount'],3)
        self.assertNotIn('header',rows[1])

    def test_mysql_repeated_eof_and_timeout_stop_at_deadline(self):
        clock = FakeClock()
        sockets = [FakeSocket([b'']),FakeSocket([b'\x01\x00',socket.timeout('partial')]),FakeSocket([b''])]
        with tempfile.TemporaryDirectory() as directory, \
             patch.object(module,'RUNTIME',Path(directory)/'runtime'), \
             patch.object(module.socket,'create_connection',side_effect=sockets) as connect, \
             patch.object(module.time,'monotonic',side_effect=clock.monotonic), \
             patch.object(module.time,'sleep',side_effect=clock.sleep), \
             patch.object(module.time,'time',return_value=123.0):
            with self.assertRaisesRegex(module.Hold,'MYSQL_NOT_READY_DEADLINE'):
                module.wait_mysql(49152)
            rows = [json.loads(line) for line in (Path(directory)/'runtime'/'mysql_readiness.jsonl').read_text(encoding='utf-8').splitlines()]
            summaries = [json.loads(line) for line in (Path(directory)/'runtime'/'mysql_readiness_summary.jsonl').read_text(encoding='utf-8').splitlines()]
        self.assertEqual(connect.call_count,3)
        self.assertEqual([row['result'] for row in rows],['transient_eof','transient_timeout','transient_eof'])
        self.assertEqual([row['headerByteCount'] for row in rows],[0,2,0])
        self.assertEqual(summaries[0]['attemptCount'],3)
        self.assertEqual(summaries[0]['status'],'HOLD')

    def test_mysql_transient_greeting_error_keeps_partial_byte_count(self):
        clock = FakeClock()
        sockets = [FakeSocket([b'\x02\x00',ConnectionResetError('reset')]),FakeSocket([b'\x01\x00\x00\x00\x0a'])]
        with tempfile.TemporaryDirectory() as directory, \
             patch.object(module,'RUNTIME',Path(directory)/'runtime'), \
             patch.object(module.socket,'create_connection',side_effect=sockets) as connect, \
             patch.object(module.time,'monotonic',side_effect=clock.monotonic), \
             patch.object(module.time,'sleep',side_effect=clock.sleep), \
             patch.object(module.time,'time',return_value=123.0):
            result = module.wait_mysql(49152)
            rows = [json.loads(line) for line in (Path(directory)/'runtime'/'mysql_readiness.jsonl').read_text(encoding='utf-8').splitlines()]
        self.assertEqual(connect.call_count,2)
        self.assertEqual([row['result'] for row in rows],['transient_greeting_error','ready'])
        self.assertEqual([row['headerByteCount'] for row in rows],[2,5])
        self.assertEqual(result['status'],'READY')

    def test_mysql_complete_invalid_header_fails_without_retry(self):
        fake = FakeSocket([b'\x01\x00\x00\x00\x09'])
        with tempfile.TemporaryDirectory() as directory, \
             patch.object(module,'RUNTIME',Path(directory)/'runtime'), \
             patch.object(module.socket,'create_connection',return_value=fake) as connect, \
             patch.object(module.time,'monotonic',return_value=0.0), \
             patch.object(module.time,'time',return_value=123.0):
            with self.assertRaisesRegex(module.Hold,'MYSQL_PROTOCOL_INVALID'):
                module.wait_mysql(49152)
            rows = [json.loads(line) for line in (Path(directory)/'runtime'/'mysql_readiness.jsonl').read_text(encoding='utf-8').splitlines()]
        self.assertEqual(connect.call_count,1)
        self.assertEqual(rows[0]['result'],'invalid_complete_header')
        self.assertEqual(rows[0]['headerByteCount'],5)

    def _observe_listener(self, stdout=b'[]', stderr=b'', code=0, timeout=False, mutate_pin=False):
        directory = tempfile.TemporaryDirectory()
        root = Path(directory.name)
        powershell = root/'System32'/'WindowsPowerShell'/'v1.0'/'powershell.exe'
        powershell.parent.mkdir(parents=True)
        powershell.write_bytes(b'synthetic pinned executable')
        digest = module.sha(powershell)
        if mutate_pin:
            powershell.write_bytes(b'changed executable')
        source = {'powershellPath':str(powershell),'powershellSha256':digest}
        completed = subprocess.CompletedProcess(['powershell.exe'],code,stdout=stdout,stderr=stderr)
        run = patch.object(module.subprocess,'run',return_value=completed)
        if timeout:
            run = patch.object(module.subprocess,'run',side_effect=subprocess.TimeoutExpired(['powershell.exe'],module.POWERSHELL_QUERY_SECONDS,output=b'partial',stderr=b'query timeout'))
        with patch.object(module,'ROOT',root), patch.object(module,'RUNTIME',root/'runtime'), patch.object(module,'IS_WINDOWS',True), \
             patch.dict(module.os.environ,{'SystemRoot':str(root)},clear=False), run as mocked_run:
            result = module.host_listener_observation(source,'backend',56024)
        return directory,result,root/'runtime',mocked_run

    def test_listener_table_accepts_only_valid_empty_json_array_and_keeps_raw_evidence(self):
        directory,result,runtime,run = self._observe_listener(stdout=b'[]')
        try:
            observation = next((runtime/'listener_observations').iterdir())
            command = json.loads((observation/'command.json').read_text(encoding='utf-8'))
            terminal = json.loads((observation/'exit.json').read_text(encoding='utf-8'))
            self.assertTrue(result['listenerAbsent'])
            self.assertEqual(result['evidenceKind'],'HOST_LISTENER_ABSENT_MAPPING_REMOVED')
            self.assertEqual((observation/'stdout.bin').read_bytes(),b'[]')
            self.assertEqual((observation/'stderr.bin').read_bytes(),b'')
            self.assertEqual(terminal['exitCode'],0)
            self.assertEqual(terminal['powershellSha256'],command['powershellSha256'])
            self.assertIn('$p=[int]56024',command['argv'][-1])
            self.assertIn('Get-NetTCPConnection -State Listen',command['argv'][-1])
            self.assertEqual(run.call_count,1)
        finally:
            directory.cleanup()

    def test_listener_table_nonempty_query_error_malformed_and_timeout_fail_closed(self):
        cases = [
            (b'[{"LocalAddress":"127.0.0.1","LocalPort":56024,"OwningProcess":12}]',b'',0,False,'HOST_LISTENER_PRESENT'),
            (b'[]',b'query failed',0,False,'HOST_LISTENER_QUERY_ERROR'),
            (b'[]',b'',1,False,'HOST_LISTENER_QUERY_ERROR'),
            (b'not json',b'',0,False,'HOST_LISTENER_QUERY_MALFORMED'),
            (b'{}',b'',0,False,'HOST_LISTENER_QUERY_MALFORMED'),
            (b'partial',b'query timeout',0,True,'HOST_LISTENER_QUERY_TIMEOUT'),
        ]
        for stdout,stderr,code,timeout,evidence_kind in cases:
            with self.subTest(evidence_kind=evidence_kind):
                directory,result,runtime,run = self._observe_listener(stdout,stderr,code,timeout)
                try:
                    observation = next((runtime/'listener_observations').iterdir())
                    terminal = json.loads((observation/'exit.json').read_text(encoding='utf-8'))
                    self.assertFalse(result['listenerAbsent'])
                    self.assertEqual(result['evidenceKind'],evidence_kind)
                    self.assertEqual(run.call_count,1)
                    self.assertTrue((observation/'stdout.bin').is_file())
                    self.assertTrue((observation/'stderr.bin').is_file())
                    if timeout: self.assertTrue(terminal['timeout'])
                finally:
                    directory.cleanup()

    def test_listener_query_does_not_run_when_powershell_hash_mismatches(self):
        directory,result,runtime,run = self._observe_listener(mutate_pin=True)
        try:
            self.assertFalse(result['listenerAbsent'])
            self.assertEqual(result['evidenceKind'],'HOST_LISTENER_PIN_OR_COMMAND_ERROR')
            run.assert_not_called()
        finally:
            directory.cleanup()

    def test_stopped_container_requires_exited_pid_zero_and_empty_ports(self):
        valid = {'State':'exited','Pid':0,'Ports':{}}
        module.require_stopped_container(valid)
        for row in ({'State':'running','Pid':0,'Ports':{}},
                    {'State':'exited','Pid':1,'Ports':{}},
                    {'State':'exited','Pid':0,'Ports':{'3306/tcp':[]}},
                    {'State':'exited','Pid':0,'Ports':None}):
            with self.subTest(row=row), self.assertRaises(module.Hold):
                module.require_stopped_container(row)

    def test_owner_identity_projection_includes_docker_pid_for_terminal_proof(self):
        container_id = 'a'*64
        network_id = 'b'*64
        owner_nonce = 'nonce'
        labels = {'ccmai.tranche':'CCMAI-RUNTIME-121','ccmai.owner':owner_nonce}
        row = {
            'Id':container_id,'Image':module.IMAGE,'Name':'/'+module.NAME,
            'State':'exited','Pid':0,'Labels':labels,
            'Mounts':[{'Type':'volume','Name':module.NAME,'Destination':'/var/lib/mysql'}],
            'NetworkMode':module.NAME,'Networks':{},'Ports':{},
        }
        network = {
            'Id':network_id,'Name':module.NAME,'Driver':'bridge','Internal':False,
            'Labels':labels,'Options':{
                'com.docker.network.bridge.gateway_mode_ipv4':'nat',
                'com.docker.network.bridge.host_binding_ipv4':'127.0.0.1',
            },
        }
        volume = {
            'Name':module.NAME,'Driver':'local','Labels':labels,'Options':{},
            'Mountpoint':'/var/lib/docker/volumes/'+module.NAME+'/_data',
        }
        calls = []
        def fake_docker(_value,*args,**_kwargs):
            calls.append(args)
            if args[0] == 'inspect': return json.dumps(row)
            if args[:2] == ('network','inspect'): return json.dumps(network)
            if args[:2] == ('volume','inspect'): return json.dumps(volume)
            if args[:2] == ('ps','-aq'): return container_id+'\n'
            raise AssertionError(args)
        receipt = {'containerId':container_id,'networkId':network_id,'ownerNonce':owner_nonce}
        with patch.object(module,'docker',side_effect=fake_docker):
            observed = module.owner_identity({'dockerPath':'docker'},receipt,require_port=False,require_network=False)
        self.assertEqual(observed['Pid'],0)
        self.assertEqual(observed['State'],'exited')
        self.assertIn('"Pid":{{json .State.Pid}}',calls[0][2])

    def test_route_policy_allows_reads_and_only_two_auth_posts(self):
        used = 0
        routes = [('GET','/api/v1/tenants/t/intervention-cases/c',True),
                  ('POST','/api/v1/auth/login',True),
                  ('POST','/api/v1/tenants/t/intervention-cases/c/actions',False),
                  ('POST','/api/v1/auth/logout',True),
                  ('POST','/api/v1/auth/login',False),
                  ('POST','/api/v1/execute',False),
                  ('DELETE','/api/v1/tenants/t/intervention-cases/c',False)]
        for method,path,expected in routes:
            allowed,used = module.route_allowed(method,path,used)
            self.assertEqual(allowed,expected)
        self.assertEqual(used,2)

    def test_terminal_route_ledgers_match_forwarded_backend_statuses(self):
        detail_path = '/api/v1/tenants/t/intervention-cases/c'
        vite_rows = [
            {'phase':'attempt','sequence':1,'method':'GET','path':detail_path,'allowed':True,'authAttemptCount':0,'authAllowedCount':0},
            {'phase':'response','sequence':1,'method':'GET','path':detail_path,'status':200},
            {'phase':'attempt','sequence':2,'method':'POST','path':'/api/v1/auth/login','allowed':True,'authAttemptCount':1,'authAllowedCount':1},
            {'phase':'response','sequence':2,'method':'POST','path':'/api/v1/auth/login','status':200},
        ]
        capture = {'case':{'id':'c','active_proposal_id':'p2'},
                   'proposals':[{'id':'p2','revision':2},{'id':'p3','revision':3}],
                   'human_events':[{'action':action,'reason':reason} for action,reason in [
                       ('OBSERVE_DATA','Minh họa: đã kiểm tra dữ liệu.'),
                       ('REQUEST_RERUN','Minh họa: đã yêu cầu xem lại.'),
                       ('EXECUTE_RERUN','approved execution')]]}
        backend_log = '\n'.join([
            f'R121_ROUTE method=GET path={detail_path} allowed=true status=200',
            'R121_DETAIL_CAPTURE '+json.dumps(capture),
            'R121_AUTH_ATTEMPT method=POST path=/api/v1/auth/login attempt=1 allowedCount=1 allowed=true',
            'R121_ROUTE method=POST path=/api/v1/auth/login allowed=true status=200',
        ])
        manifest = {'tenant':'t','case':'c','activeProposalId':'p2','laterOffLineageProposalId':'p3',
                    'eventActions':['OBSERVE_DATA','REQUEST_RERUN','EXECUTE_RERUN']}
        with tempfile.TemporaryDirectory() as directory:
            ledger = Path(directory) / 'vite_routes.jsonl'
            ledger.write_text(''.join(json.dumps(row)+'\n' for row in vite_rows),encoding='utf-8')
            result = module.validate_route_ledgers(ledger,backend_log,manifest)
        self.assertEqual(result['authAttempts'],1)
        self.assertEqual(result['authAllowed'],1)
        self.assertEqual(result['detailCaptures'],1)

    def test_vite_and_go_ledgers_reject_a_third_auth_attempt_independently(self):
        vite_rows = []
        backend_lines = []
        for sequence,path in enumerate(['/api/v1/auth/login','/api/v1/auth/logout','/api/v1/auth/login'],1):
            allowed = sequence <= 2
            vite_rows.extend([
                {'phase':'attempt','sequence':sequence,'method':'POST','path':path,'allowed':allowed,
                 'authAttemptCount':sequence,'authAllowedCount':min(sequence,2)},
                {'phase':'response','sequence':sequence,'method':'POST','path':path,'status':200 if allowed else 405},
            ])
            backend_lines.append(f'R121_AUTH_ATTEMPT method=POST path={path} attempt={sequence} allowedCount={min(sequence,2)} allowed={str(allowed).lower()}')
            backend_lines.append(f'R121_ROUTE method=POST path={path} allowed={str(allowed).lower()} status={200 if allowed else 405}')
        with tempfile.TemporaryDirectory() as directory:
            ledger = Path(directory) / 'vite_routes.jsonl'
            ledger.write_text(''.join(json.dumps(row)+'\n' for row in vite_rows),encoding='utf-8')
            with self.assertRaises(module.Hold): module.validate_vite_routes(ledger)
        with self.assertRaises(module.Hold): module.validate_go_routes('\n'.join(backend_lines),{})

    def test_final_fingerprint_and_provider_related_counts_must_match(self):
        counts = {'attempts':0,'executions':0,'grants':0,'budgets':0,'receipts':0,'runs':1,'snapshots':1,'humanEvents':3}
        manifest = {'initialFingerprint':'a'*64,'initialEffectCounts':counts}
        log = '\n'.join([
            f"R121_BROWSER_FINAL initial={'a'*64} final={'a'*64} equal=true",
            'R121_EFFECT_COUNTS '+json.dumps({'initial':counts,'final':counts}),
        ])
        self.assertEqual(module.validate_fixture_accounting(log,manifest)['providerCalls'],0)
        changed = dict(counts,attempts=1)
        bad_log = log.replace(json.dumps({'initial':counts,'final':counts}),json.dumps({'initial':counts,'final':changed}))
        with self.assertRaises(module.Hold): module.validate_fixture_accounting(bad_log,manifest)

    def test_docker_command_keeps_raw_stdout_and_stderr_bytes(self):
        with tempfile.TemporaryDirectory() as directory:
            runtime = Path(directory) / 'r121_ui'
            config = runtime / 'docker-config'
            config.mkdir(parents=True)
            (config / 'config.json').write_text('{}',encoding='utf-8')
            completed = subprocess.CompletedProcess(['docker','version'],0,stdout=b'raw\x00stdout',stderr=b'raw\xffstderr')
            with patch.object(module,'RUNTIME',runtime), patch.object(module.subprocess,'run',return_value=completed):
                result = module.docker({'dockerPath':'docker'},'version')
            command = runtime / 'docker_commands' / '0001'
            self.assertEqual(result,'raw\x00stdout')
            self.assertEqual((command/'stdout.bin').read_bytes(),b'raw\x00stdout')
            self.assertEqual((command/'stderr.bin').read_bytes(),b'raw\xffstderr')

    def test_stop_uses_exact_owned_id_without_a_published_port(self):
        baseline = {'containers':[],'networks':[],'volumes':[]}
        receipt = {'containerId':'a'*64,'protectedBefore':baseline}
        calls = []
        rows = iter([{'State':'running','Pid':23,'Ports':{}},{'State':'exited','Pid':0,'Ports':{}}])
        def fake_docker(_value,*args,**kwargs):
            calls.append(args)
            return ''
        with tempfile.TemporaryDirectory() as directory, \
             patch.object(module,'owner_identity',side_effect=lambda *_args,**_kwargs:next(rows)) as owned, \
             patch.object(module,'docker',side_effect=fake_docker), \
             patch.object(module,'inventory',return_value=baseline), \
             patch.object(module,'host_listener_observation',return_value={'listenerAbsent':False,'evidenceKind':'HOST_LISTENER_INVALID_PORT'}), \
             patch.object(module,'RUNTIME',Path(directory)/'runtime'):
            result = module.stop_fixture({},receipt)
        self.assertTrue(result['dockerTerminalObserved'])
        self.assertFalse(result['listenerAbsent'])
        self.assertEqual(result['evidenceKind'],'HOST_LISTENER_INVALID_PORT')
        self.assertTrue(result['protectedEqual'])
        self.assertEqual(calls,[('stop','--time','10','a'*64)])
        self.assertEqual(owned.call_count,2)
        self.assertTrue(all(call.kwargs.get('require_port') is False for call in owned.call_args_list))

    def test_unknown_ownership_prevents_stop(self):
        with patch.object(module,'owner_identity',side_effect=module.Hold('OWNED_LABEL_CHANGED')), \
             patch.object(module,'docker') as docker:
            with self.assertRaises(module.Hold): module.stop_fixture({}, {'containerId':'a'*64})
            docker.assert_not_called()


if __name__ == '__main__': unittest.main()
