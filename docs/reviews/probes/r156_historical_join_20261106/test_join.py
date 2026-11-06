"""STATIC_CANDIDATE_ONLY; actual retained history + synthetic local graph mutations.

No real repository commits, no network/provider/application runtime. Fixtures are
retained under .runtime (ignored) to avoid destructive Windows Git cleanup.
"""
import copy
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import time
import unittest

HERE = Path(__file__).resolve().parent
PROJECT = HERE.parents[3]
sys.path.insert(0, str(HERE/'candidate'))
import cvf_milestone_coverage as C
import cvf_historical_admissions as H
import cvf_downstream_gate_runner as R
import cvf_dg_common as COMMON

ADMISSION = 'c61b01fa89c40135093103a0852de70f82bd39ec'
WRONG = 'ec6eed18a5f1951d9dc54a5b49d14f804574ab67'
BUILD = '7052df5c6da31d6867172d26a298a7bb4b72bb9c'
TERMINAL = 'ed9d55b25290fd8b640f16e0635fbe895e9b9b5f'
BASE = 'f7405c470207dc0d3be64e6eca79b7050d3f79d8'
PACKET = 'docs/work_orders/CCMAI_RUNTIME_150_A1_A2_WINDOWS_VERIFY.md'
CURRENT_PACKET = 'docs/work_orders/CCMAI_RUNTIME_151_EVIDENCE_CAPTURE_REPAIR.md'
REGISTRY = 'docs/milestone-coverage-authority.json'
SEED = 'CVF_SESSION/authority/CCMAI-RUNTIME-150.json'
RECORD = 'CVF_SESSION/tranches/CCMAI-RUNTIME-150.json'
AUDIT = 'docs/reviews/probes/r139_integrated_20261021/causal_assessment_20261105/repair_chain_assessment_a1_a2_windows_verify.json'

def git(root, *args):
    r = subprocess.run(['git', '--no-replace-objects', '-C', str(root), *args],
                       capture_output=True, timeout=15)
    if r.returncode:
        raise RuntimeError((args, r.stderr.decode(errors='replace')))
    return r.stdout

def write(root, path, raw):
    p = fs(root/path); p.parent.mkdir(parents=True, exist_ok=True)
    p.write_bytes(raw)

def fs(path):
    return Path('\\\\?\\'+str(path.absolute())) if os.name=='nt' and not str(path).startswith('\\\\?\\') else path

def encoded(value):
    return (json.dumps(value, ensure_ascii=False, indent=2)+'\n').encode()

def block(text):
    return json.loads(re.search(r'```cvf-milestone-coverage-json\s*\n(.*?)\n```', text, re.S)[1])

def reference(root, path):
    return {'path':path, 'sha256':C.digest(fs(root/path).read_bytes())}

def commit(root, paths, message):
    git(root, 'add', '--', *paths)
    git(root, '-c', 'core.hooksPath=NUL', 'commit', '-qm', message)
    return git(root, 'rev-parse', 'HEAD').decode().strip()

class HistoricalJoinTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        runtime = PROJECT/'.cvf/runtime/r156'; runtime.mkdir(parents=True,exist_ok=True)
        cls.root = Path(tempfile.mkdtemp(prefix='retained_', dir=runtime))
        git(cls.root, 'init', '-q')
        git(cls.root, 'config', 'user.name', 'STATIC_CANDIDATE_ONLY')
        git(cls.root, 'config', 'user.email', 'fixture@example.invalid')
        git(cls.root, 'config', 'commit.gpgsign', 'false')
        (cls.root/'.git/no-hooks').mkdir()
        git(cls.root, 'config', 'core.hooksPath', str(cls.root/'.git/no-hooks'))
        git(cls.root, 'config', 'core.autocrlf', 'false')
        git(cls.root, 'config', 'core.longpaths', 'true')
        objects = git(PROJECT, 'rev-parse', '--path-format=absolute', '--git-path', 'objects').decode().strip()
        write(cls.root, '.git/objects/info/alternates', (objects+'\n').encode())
        cls.original_head = git(PROJECT, 'rev-parse', 'HEAD').decode().strip()
        git(cls.root, 'update-ref', 'HEAD', cls.original_head)
        git(cls.root, 'read-tree', cls.original_head)  # index metadata only; no full-tree checkout
        # Materialize a finite pinned evidence closure only, never a full Git tree.
        seen, parsed = set(), set()
        def materialize(path, follow=False):
            if path not in seen:
                seen.add(path)
                write(cls.root, path, (PROJECT/path).read_bytes().replace(b'\r\n',b'\n'))
            if not follow or path in parsed: return
            parsed.add(path)
            raw = fs(cls.root/path).read_bytes()
            try: data = json.loads(raw)
            except (ValueError, UnicodeError):
                text = raw.decode('utf-8-sig')
                match = re.search(r'```cvf-milestone-coverage-json\s*\n(.*?)\n```', text, re.S)
                data = json.loads(match[1]) if match else None
            def walk(value):
                if isinstance(value, dict):
                    if isinstance(value.get('path'), str) and (PROJECT/value['path']).is_file():
                        materialize(value['path'])
                    for v in value.values(): walk(v)
                elif isinstance(value, list):
                    for v in value: walk(v)
            walk(data)
        for path in (REGISTRY, PACKET, CURRENT_PACKET, AUDIT, SEED, RECORD): materialize(path,True)
        live_registry=json.loads((cls.root/REGISTRY).read_bytes())
        for control in live_registry['repairControls'].values():
            for ref in control['history']:materialize(ref['path'],True)
            if control['audit']:materialize(control['audit']['path'],True)
        record = json.loads((cls.root/RECORD).read_bytes())
        materialize(record['workOrder'])
        for p in record['reviewEvidence']: materialize(p)
        cls.materialized = sorted(seen)
        historical = json.loads(git(cls.root, 'show', ADMISSION+':'+REGISTRY))
        cls.row = {'milestone':'M3', 'tranche':'CCMAI-RUNTIME-150', 'packet':reference(cls.root, PACKET),
                   'admission':ADMISSION, 'build':BUILD, 'terminal':TERMINAL,
                   'registry':{'path':REGISTRY,'sha256':C.digest(git(cls.root, 'show', ADMISSION+':'+REGISTRY))},
                   'audit':reference(cls.root,AUDIT),'seed':reference(cls.root,SEED),
                   'terminalRecord':reference(cls.root,RECORD),'carrier':reference(cls.root,record['workOrder']),
                   'reviewEvidence':[reference(cls.root,p) for p in record['reviewEvidence']]}
        cls.serial = 0
        cls.default_context = cls.publish([cls.row])
        cls.text = (cls.root/PACKET).read_text(encoding='utf-8-sig')
        cls.data = block(cls.text)
        report=Path(os.environ.get('R156_REVIEW_REPORT_PATH',str(HERE/'fixture_sources.json')))
        report.parent.mkdir(parents=True,exist_ok=True)
        report.write_bytes(encoded({'head':cls.original_head,
            'materializedEvidenceFiles':cls.materialized,'actualAdmission':ADMISSION,
            'actualWrongAnchor':WRONG,'fixtureRoot':str(cls.root)}))

    @classmethod
    def publish(cls, rows, raw_override=None, review_change=None):
        cls.serial += 1
        prefix = f'docs/proposed/r156_{cls.serial}'
        review = {'schemaVersion':'cvf.historicalAdmissionReview@candidate1','decision':'ACCEPTED',
                  'reviewer':'Independent synthetic binding reviewer','worker':'Synthetic binding author',
                  'bindingsSha256':H.digest(json.dumps(rows,ensure_ascii=False,sort_keys=True,separators=(',',':')).encode())}
        if review_change: review.update(review_change)
        rp = prefix+'_review.json'; write(cls.root,rp,encoded(review))
        review_commit = commit(cls.root,[rp],'STATIC_CANDIDATE_ONLY review declaration')
        doc = {'schemaVersion':H.SCHEMA,'bindings':rows,
               'review':dict(reference(cls.root,rp),commit=review_commit)}
        mp=prefix+'_bindings.json'; write(cls.root,mp,raw_override if raw_override is not None else encoded(doc))
        manifest_commit = commit(cls.root,[mp],'STATIC_CANDIDATE_ONLY proposed bindings')
        return {'phase':'pr-ci','head':manifest_commit,'base':BASE,'active':[],'explicit':[],
                'manifest':dict(reference(cls.root,mp),commit=manifest_commit)}

    def context(self):
        ctx=copy.deepcopy(self.default_context)
        ctx['head']=git(self.root,'rev-parse','HEAD').decode().strip()
        return ctx

    def validate(self, context=None, data=None, text=None):
        return C.validate(self.root, self.data if data is None else data, PACKET,
                          self.text if text is None else text,
                          self.context() if context is None else context)

    def assert_rejected(self, issues, expected=None):
        self.assertTrue(issues, 'unexpected admission PASS')
        if expected: self.assertTrue(any(expected in i for i in issues), issues)

    def test_actual_retained_positive_and_default_strict(self):
        self.assertEqual(self.validate(), [])
        self.assertEqual(C.validate_file(self.root,PACKET), ['COV_REPAIR_ACTION_MISMATCH'])
        self.assertNotEqual(json.loads((self.root/REGISTRY).read_bytes())['repairControls']['M3']['audit'],self.row['audit'])
        self.assertEqual(C.validate_file(self.root,CURRENT_PACKET), [])
        self.assertEqual(C.validate_file(self.root,CURRENT_PACKET,historical=self.context()), [])

    def test_real_copied_runner_integration(self):
        M={'cvf_dg_common':COMMON,'cvf_milestone_coverage':C}
        profile={'continuityContract':{'implementationPath':'IMPLEMENTATION_STATUS.json'}}
        result=R._milestone_result(M,self.root,[PACKET,CURRENT_PACKET],profile,BASE,self.context())
        self.assertTrue(result.ok, result.as_dict())
        strict=R._milestone_result(M,self.root,[PACKET],profile,BASE)
        self.assertFalse(strict.ok)
        self.assertIn('COV_REPAIR_ACTION_MISMATCH',[f.code for f in strict.findings])
        try:
            write(self.root,'IMPLEMENTATION_STATUS.json',encoded({'activeWorkOrders':[PACKET]}))
            active=R._milestone_result(M,self.root,[PACKET],profile,BASE,self.context())
            self.assertFalse(active.ok)
            self.assertIn('HIST_EXECUTION_FORBIDDEN',[f.code for f in active.findings])
        finally:(self.root/'IMPLEMENTATION_STATUS.json').unlink(missing_ok=True)

    def test_actual_wrong_anchor(self):
        row=copy.deepcopy(self.row);row['admission']=WRONG
        wrong_registry=git(self.root,'show',WRONG+':'+REGISTRY)
        row['registry']['sha256']=C.digest(wrong_registry)
        self.assertNotEqual(json.loads(wrong_registry)['repairControls']['M3']['audit'],row['audit'])
        # The actual earlier seed anchor lacks the admitted packet and fails at
        # that earlier provenance check, before the audit-selection join.
        self.assert_rejected(self.validate(self.publish([row])),'HIST_BLOB_ENTRY')

    def test_duplicate_bindings_are_rejected(self):
        self.assert_rejected(self.validate(self.publish([self.row,self.row])),'HIST_DUPLICATE')

    def test_unselected_bindings_are_fully_checked(self):
        row=copy.deepcopy(self.row);row['packet']['path']='docs/work_orders/unselected.md'
        row['tranche']='CCMAI-RUNTIME-OTHER'
        self.assert_rejected(self.validate(self.publish([self.row,row])),'HIST_BLOB_ENTRY')

    def test_execution_head_commits_and_review_denials(self):
        for key,value in [('phase','pre-dispatch'),('phase','worker-return'),('phase','bootstrap'),
                          ('active',[PACKET]),('explicit',[PACKET]),
                          ('active',[PACKET.replace('/','\\')]),('explicit',[PACKET.replace('/','\\')]),
                          ('active',['./'+PACKET]),('explicit',['./'+PACKET]),
                          ('head',ADMISSION),('head','0'*40),('base','0'*40)]:
            with self.subTest(key=key,value=value):
                ctx=self.context();ctx[key]=value
                self.assert_rejected(self.validate(ctx))
        for key,value in [('admission','0'*40),('terminal','bad'),('build',ADMISSION),('tranche','UNKNOWN')]:
            row=copy.deepcopy(self.row);row[key]=value
            with self.subTest(key=key):self.assert_rejected(self.validate(self.publish([row])))
        self.assert_rejected(self.validate(self.publish([self.row],review_change={'reviewer':'Synthetic binding author'})),
                             'HIST_INDEPENDENT_REVIEW')

    def test_mutable_files_and_action_semantics(self):
        for name in ('packet','audit','seed','terminalRecord','carrier'):
            ref=self.row[name]; target=fs(self.root/ref['path']); saved=target.read_bytes()
            try:
                target.write_bytes(saved+b'\nchanged')
                with self.subTest(name=name):self.assert_rejected(self.validate())
            finally:target.write_bytes(saved)
        rp=self.row['reviewEvidence'][0]['path']; saved=(self.root/rp).read_bytes()
        try:
            (self.root/rp).write_bytes(saved+b'changed');self.assert_rejected(self.validate())
        finally:(self.root/rp).write_bytes(saved)
        for key,value in [('owner','wrong'),('output','wrong'),('selected',['M3-05']),
                          ('question','wrong'),('budgetMinutes',46),('kind','IMPLEMENT')]:
            data=copy.deepcopy(self.data);data['action'][key]=value
            with self.subTest(key=key):self.assert_rejected(self.validate(data=data))

    def test_parse_paths_bounds_and_nonancestor(self):
        for raw in (b'{"a":1,"a":2}',b'{"x":NaN}',b' '*(H.LIMIT+1)):
            self.assert_rejected(self.validate(self.publish([self.row],raw_override=raw)))
        for name in ('seed','audit'):
            row=copy.deepcopy(self.row);row[name]['path']='../escape'
            self.assert_rejected(self.validate(self.publish([row])))
        reader=H.Reader(self.root,C._repair_api())
        large='docs/proposed/oversize';write(self.root,large,b'x'*(H.LIMIT+1))
        large_commit=commit(self.root,[large],'STATIC_CANDIDATE_ONLY oversized blob')
        with self.assertRaisesRegex(C.Invalid,'HIST_OVERSIZE'):reader.blob(large_commit,large)
        # Synthetic symlink in Git tree; works without Windows symlink privilege.
        path='docs/proposed/link'
        blob=subprocess.run(['git','-C',str(self.root),'hash-object','-w','--stdin'],input=b'../escape',capture_output=True,check=True).stdout.decode().strip()
        git(self.root,'update-index','--add','--cacheinfo','120000,'+blob+','+path)
        git(self.root,'-c','core.hooksPath=NUL','commit','-qm','STATIC_CANDIDATE_ONLY symlink tree')
        symlink_commit=git(self.root,'rev-parse','HEAD').decode().strip()
        with self.assertRaisesRegex(C.Invalid,'HIST_BLOB_ENTRY'):reader.blob(symlink_commit,path)
        # An orphan commit exists but is not an ancestor of the checked head.
        tree=git(self.root,'rev-parse','HEAD^{tree}').decode().strip()
        orphan=subprocess.run(['git','-C',str(self.root),'commit-tree',tree],input=b'orphan fixture\n',capture_output=True,check=True).stdout.decode().strip()
        row=copy.deepcopy(self.row);row['build']=orphan
        self.assert_rejected(self.validate(self.publish([row])))

    def test_current_history_audit_cost_product_guards(self):
        registry_path=self.root/REGISTRY; original=registry_path.read_bytes(); registry=json.loads(original)
        current_audit_relative=registry['repairControls']['M3']['audit']['path']
        current_audit_path=fs(self.root/current_audit_relative); audit_raw=current_audit_path.read_bytes()
        try:
            for mode in ('erase','reorder','malformed_audit','cost_reset'):
                changed=copy.deepcopy(registry)
                if mode=='erase':changed['repairControls']={}
                elif mode=='reorder':changed['repairControls']['M3']['history'].reverse()
                else:
                    audit=json.loads(audit_raw)
                    if mode=='malformed_audit':audit['unexpected']=True
                    else:audit['cumulativeMinutes']=0
                    current_audit_path.write_bytes(encoded(audit))
                    changed['repairControls']['M3']['audit']=reference(self.root,current_audit_relative)
                registry_path.write_bytes(encoded(changed))
                with self.subTest(mode=mode):self.assert_rejected(self.validate())
                current_audit_path.write_bytes(audit_raw)
            registry_path.write_bytes(original)
            data=copy.deepcopy(self.data);data['phase']='REVIEW';data['action']['kind']='REVIEW'
            data['conclusions']['product']='COMPLETE'
            self.assert_rejected(self.validate(data=data))
            with self.assertRaisesRegex(C.Invalid,'COV_REPAIR_UNRESOLVED_PRODUCT'):
                C._repair().validate(self.root,data,registry,C._repair_api(PACKET,self.text,strict=True),self.context())
        finally:registry_path.write_bytes(original);current_audit_path.write_bytes(audit_raw)

    def test_forged_terminal_and_binding_provenance(self):
        # Rehashing a working-tree forgery cannot change the bound terminal blob.
        target=self.root/RECORD;saved=target.read_bytes()
        try:
            data=json.loads(saved);data['trancheId']='UNKNOWN';target.write_bytes(encoded(data))
            row=copy.deepcopy(self.row);row['terminalRecord']=reference(self.root,RECORD)
            self.assert_rejected(self.validate(self.publish([row])))
        finally:target.write_bytes(saved)
        ctx=self.context();mp=self.root/ctx['manifest']['path'];saved=mp.read_bytes()
        try:
            mp.write_bytes(saved+b'\n')
            self.assert_rejected(self.validate(ctx),'HIST_RETAINED_STALE')
        finally:mp.write_bytes(saved)

    def test_post_terminal_commit_then_revert_is_not_immutable(self):
        original_head=git(self.root,'rev-parse','HEAD').decode().strip()
        target=self.root/PACKET;saved=target.read_bytes()
        try:
            target.write_bytes(saved+b'\nmutated after freeze\n')
            commit(self.root,[PACKET],'STATIC_CANDIDATE_ONLY forbidden post-freeze mutation')
            target.write_bytes(saved)
            commit(self.root,[PACKET],'STATIC_CANDIDATE_ONLY revert still leaves mutation history')
            self.assert_rejected(self.validate(),'HIST_POST_TERMINAL_MUTATION')
        finally:
            git(self.root,'checkout','--detach',original_head)
            target.write_bytes(saved)

    def test_both_parent_merge_collision_preserves_current_authority(self):
        # Two branches from current fixture head with genuinely conflicting authority.
        initial=git(self.root,'rev-parse','HEAD').decode().strip()
        path='docs/proposed/collision.json';write(self.root,path,b'{"owner":"historical"}\n')
        left=commit(self.root,[path],'STATIC_CANDIDATE_ONLY local authority')
        git(self.root,'checkout','-qb','collision_other',initial)
        write(self.root,path,b'{"owner":"remote"}\n');right=commit(self.root,[path],'STATIC_CANDIDATE_ONLY remote authority')
        git(self.root,'checkout','--detach',left)
        merged=subprocess.run(['git','-C',str(self.root),'merge','--no-commit','--no-ff',right],capture_output=True)
        self.assertNotEqual(merged.returncode,0)
        write(self.root,path,b'{"owner":"current","historicalOrigin":"retained"}\n')
        commit(self.root,[path],'STATIC_CANDIDATE_ONLY preserve both origins')
        parents=git(self.root,'show','-s','--format=%P','HEAD').decode().split()
        self.assertEqual(set(parents),{left,right})
        self.assertEqual(self.validate(),[])
        self.assertEqual(C.validate_file(self.root,CURRENT_PACKET),[])

if __name__=='__main__':
    unittest.main(verbosity=2)
