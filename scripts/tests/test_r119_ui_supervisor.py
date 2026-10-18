"""Offline software checks only; no Docker, Go, browser, or provider effect."""
import importlib.util
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

SCRIPT = Path(__file__).resolve().parents[1] / 'r119_ui_supervisor.py'
spec = importlib.util.spec_from_file_location('r119_ui_supervisor', SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class R119SupervisorTests(unittest.TestCase):
    def test_published_port_requires_one_complete_loopback_binding(self):
        self.assertEqual(module.published_port({'3306/tcp':[{'HostIp':'127.0.0.1','HostPort':'49152'}]}),49152)
        for ports in ({}, {'3306/tcp':None}, {'3306/tcp':[]},
                      {'3306/tcp':[{'HostIp':'0.0.0.0','HostPort':'49152'}]},
                      {'3306/tcp':[{'HostIp':'127.0.0.1','HostPort':'0'}]},
                      {'3306/tcp':[{'HostIp':'127.0.0.1','HostPort':'49152'}],'other':[]},
                      {'3306/tcp':[{'HostIp':'127.0.0.1','HostPort':'49152'}, {'HostIp':'127.0.0.1','HostPort':'49153'}]}):
            with self.subTest(ports=ports), self.assertRaises(module.Hold): module.published_port(ports)

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
            f'R119_ROUTE method=GET path={detail_path} allowed=true status=200',
            'R119_DETAIL_CAPTURE '+json.dumps(capture),
            'R119_AUTH_ATTEMPT method=POST path=/api/v1/auth/login attempt=1 allowedCount=1 allowed=true',
            'R119_ROUTE method=POST path=/api/v1/auth/login allowed=true status=200',
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
            backend_lines.append(f'R119_AUTH_ATTEMPT method=POST path={path} attempt={sequence} allowedCount={min(sequence,2)} allowed={str(allowed).lower()}')
            backend_lines.append(f'R119_ROUTE method=POST path={path} allowed={str(allowed).lower()} status={200 if allowed else 405}')
        with tempfile.TemporaryDirectory() as directory:
            ledger = Path(directory) / 'vite_routes.jsonl'
            ledger.write_text(''.join(json.dumps(row)+'\n' for row in vite_rows),encoding='utf-8')
            with self.assertRaises(module.Hold): module.validate_vite_routes(ledger)
        with self.assertRaises(module.Hold): module.validate_go_routes('\n'.join(backend_lines),{})

    def test_final_fingerprint_and_provider_related_counts_must_match(self):
        counts = {'attempts':0,'executions':0,'grants':0,'budgets':0,'receipts':0,'runs':1,'snapshots':1,'humanEvents':3}
        manifest = {'initialFingerprint':'a'*64,'initialEffectCounts':counts}
        log = '\n'.join([
            f"R119_BROWSER_FINAL initial={'a'*64} final={'a'*64} equal=true",
            'R119_EFFECT_COUNTS '+json.dumps({'initial':counts,'final':counts}),
        ])
        self.assertEqual(module.validate_fixture_accounting(log,manifest)['providerCalls'],0)
        changed = dict(counts,attempts=1)
        bad_log = log.replace(json.dumps({'initial':counts,'final':counts}),json.dumps({'initial':counts,'final':changed}))
        with self.assertRaises(module.Hold): module.validate_fixture_accounting(bad_log,manifest)

    def test_docker_command_keeps_raw_stdout_and_stderr_bytes(self):
        with tempfile.TemporaryDirectory() as directory:
            runtime = Path(directory) / 'r119_ui'
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
        rows = iter([{'State':'running','Ports':{}},{'State':'exited','Ports':{}}])
        def fake_docker(_value,*args,**kwargs):
            calls.append(args)
            return ''
        with patch.object(module,'owner_identity',side_effect=lambda *_args,**_kwargs:next(rows)) as owned, \
             patch.object(module,'docker',side_effect=fake_docker), \
             patch.object(module,'inventory',return_value=baseline):
            result = module.stop_fixture({},receipt)
        self.assertEqual(result,{'containerState':'exited','portClosed':None,'protectedEqual':True})
        self.assertEqual(calls,[('stop','--time','10','a'*64)])
        self.assertEqual(owned.call_count,2)
        self.assertTrue(all(call.kwargs.get('require_port') is False for call in owned.call_args_list))

    def test_unknown_ownership_prevents_stop(self):
        with patch.object(module,'owner_identity',side_effect=module.Hold('OWNED_LABEL_CHANGED')), \
             patch.object(module,'docker') as docker:
            with self.assertRaises(module.Hold): module.stop_fixture({}, {'containerId':'a'*64})
            docker.assert_not_called()


if __name__ == '__main__': unittest.main()
