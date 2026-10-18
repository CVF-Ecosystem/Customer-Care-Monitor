#!/usr/bin/env python3
"""One owned R121 synthetic UI campaign. Import and inspect are read-only.

cvf-allow-secret-fixture: the MySQL password below belongs only to a fresh,
disposable synthetic database. No provider, customer, or installed DB is used.
"""
from __future__ import annotations

import argparse
from collections import Counter
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import socket
import subprocess
import sys
import time
import urllib.request
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parent.parent
RUNTIME = ROOT / '.cvf/runtime/r121_ui'
EVIDENCE = ROOT / 'docs/reviews/probes/r121_root'
SOURCE = EVIDENCE / 'source_binary_admission.json'
RESOURCE = EVIDENCE / 'resource_admission.json'
NAME = 'ccmai-r121-ui-20261018'
IMAGE = 'sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b'
PASSWORD = 'r121-synthetic-only'
DATABASE = 'r121_fixture'
CAMPAIGN_SECONDS = 1200
MYSQL_READY_SECONDS = 90
MYSQL_PER_READ_SECONDS = 2.0
MYSQL_RETRY_DELAY_SECONDS = 0.5
POWERSHELL_QUERY_SECONDS = 15
IS_WINDOWS = os.name == 'nt'
HEX64 = re.compile(r'[0-9a-f]{64}\Z')
HEX40 = re.compile(r'[0-9a-f]{40}\Z')
READY = re.compile(r'^R121_BROWSER_READY (http://127\.0\.0\.1:[1-9][0-9]{0,4}) (\{.*\})$')


class Hold(RuntimeError): pass


def require(condition, message):
    if not condition: raise Hold(message)


def sha(path): return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def read(path): return json.loads(Path(path).read_text(encoding='utf-8'))


def source_snapshot(admission):
    return {'admissionSha256':sha(SOURCE),'sourceCommit':admission['sourceCommit'],'sourceSha256':admission['sourceSha256'],'binarySha256':admission['binarySha256'],'nodeSha256':admission['nodeSha256'],'dockerSha256':admission['dockerSha256'],'powershellPath':admission['powershellPath'],'powershellSha256':admission['powershellSha256']}


def verify_source_snapshot(expected):
    current = source_admission()
    observed = source_snapshot(current)
    require(observed == expected, 'SOURCE_PINS_CHANGED_DURING_CAMPAIGN')
    return observed


def write_new(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open('x', encoding='utf-8') as stream:
        json.dump(value, stream, ensure_ascii=False, indent=2)
        stream.write('\n')
        stream.flush(); os.fsync(stream.fileno())


def write_bytes_new(path, data):
    with Path(path).open('xb') as stream:
        stream.write(data)
        stream.flush(); os.fsync(stream.fileno())


def append(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open('a', encoding='utf-8') as stream:
        stream.write(json.dumps(value, ensure_ascii=False, sort_keys=True) + '\n')
        stream.flush(); os.fsync(stream.fileno())


def minimal_env():
    keys = {'SYSTEMROOT','WINDIR','TEMP','TMP','PATH','USERPROFILE','LOCALAPPDATA'}
    return {key:value for key,value in os.environ.items() if key.upper() in keys}


def checked_file(relative):
    require(isinstance(relative,str) and '/' in relative and '\\' not in relative and '..' not in Path(relative).parts, 'INVALID_PIN_PATH')
    path = (ROOT / relative).resolve()
    require(path.is_relative_to(ROOT.resolve()) and path.is_file(), 'PIN_PATH_OUTSIDE_OR_MISSING')
    return path


def committed(path):
    relative = path.relative_to(ROOT).as_posix()
    result = subprocess.run(['git','show','HEAD:'+relative], cwd=ROOT, capture_output=True, timeout=20)
    require(result.returncode == 0 and result.stdout.replace(b'\r\n',b'\n') == path.read_bytes().replace(b'\r\n',b'\n'), 'ADMISSION_NOT_COMMITTED')
    return read(path)


def source_admission():
    value = committed(SOURCE)
    require(value.get('trancheId') == 'CCMAI-RUNTIME-121' and value.get('image') == IMAGE, 'SOURCE_ADMISSION_IDENTITY')
    commit = value.get('sourceCommit','')
    require(isinstance(commit,str) and HEX40.fullmatch(commit), 'SOURCE_COMMIT_INVALID')
    ancestor = subprocess.run(['git','merge-base','--is-ancestor',commit,'HEAD'],cwd=ROOT,capture_output=True,timeout=15)
    require(ancestor.returncode == 0, 'SOURCE_COMMIT_NOT_ANCESTOR')
    pins = value.get('sourceSha256')
    require(isinstance(pins,dict), 'SOURCE_PINS_MISSING')
    required = {'backend/api/handlers/intervention_ui_r121_test.go','backend/go.mod','backend/go.sum','scripts/r121_ui_supervisor.py','scripts/r121_vite_config.mjs.template','scripts/tests/test_r121_ui_supervisor.py','frontend/package.json','frontend/package-lock.json','frontend/vite.config.ts','frontend/index.html','frontend/tsconfig.json','frontend/tsconfig.app.json','frontend/tsconfig.node.json'}
    # The source set is exact at admission time, including every compiled Go source.
    actual_go = {p.relative_to(ROOT).as_posix() for p in (ROOT/'backend').rglob('*.go')}
    frontend_roots = [ROOT/'frontend/src', ROOT/'frontend/public']
    actual_frontend = {p.relative_to(ROOT).as_posix() for source_root in frontend_roots if source_root.exists() for p in source_root.rglob('*') if p.is_file()}
    require(required.issubset(pins) and actual_go.issubset(pins) and actual_frontend.issubset(pins), 'SOURCE_PINS_INCOMPLETE')
    for relative,digest in pins.items():
        require(isinstance(digest,str) and HEX64.fullmatch(digest) and sha(checked_file(relative)) == digest, 'SOURCE_PIN_CHANGED')
        blob = subprocess.run(['git','show','HEAD:'+relative],cwd=ROOT,capture_output=True,timeout=15)
        require(blob.returncode == 0 and blob.stdout.replace(b'\r\n',b'\n') == checked_file(relative).read_bytes().replace(b'\r\n',b'\n'), 'SOURCE_NOT_COMMITTED')
    for key in ('binaryPath','nodePath','dockerPath'):
        require(isinstance(value.get(key),str), 'EXECUTABLE_PIN_MISSING')
    binary = checked_file(value['binaryPath'])
    require(binary.is_relative_to((ROOT/'.cvf/runtime/r121_build').resolve()), 'BINARY_PATH_INVALID')
    require(HEX64.fullmatch(value.get('binarySha256','')) and sha(binary) == value['binarySha256'], 'BINARY_PIN_CHANGED')
    for name in ('node','docker'):
        path = Path(value[name+'Path'])
        require(path.is_file() and HEX64.fullmatch(value.get(name+'Sha256','')) and sha(path) == value[name+'Sha256'], name.upper()+'_PIN_CHANGED')
    require(IS_WINDOWS, 'WINDOWS_POWERSHELL_REQUIRED')
    powershell_path, powershell_hash = validate_powershell_pin(value)
    require(powershell_path.is_file() and sha(powershell_path) == powershell_hash, 'POWERSHELL_PIN_CHANGED')
    return value


def docker(value, *args, mutation=False, timeout=20, checked=True):
    if mutation: append(RUNTIME/'effect_reservations.jsonl', {'kind':'docker','args':list(args),'at':time.time()})
    config = RUNTIME/'docker-config'
    # A clean explicit Docker config prevents ambient credential/plugin reads.
    if not config.exists():
        require(mutation, 'INSPECTION_REQUIRES_EXISTING_CONFIG')
        config.mkdir(parents=True,exist_ok=False)
        write_new(config/'config.json',{})
    argv = [value['dockerPath'],'--host','npipe:////./pipe/dockerDesktopLinuxEngine','--config',str(config),*args]
    commands = RUNTIME/'docker_commands'
    commands.mkdir(parents=True,exist_ok=True)
    command_dir = commands/f'{len(list(commands.glob("*")))+1:04d}'
    command_dir.mkdir(parents=True,exist_ok=False)
    write_new(command_dir/'command.json',{'args':list(args),'mutation':mutation,'timeoutSeconds':timeout})
    try: result = subprocess.run(argv,cwd=ROOT,env=minimal_env(),capture_output=True,timeout=timeout)
    except subprocess.TimeoutExpired as error:
        write_bytes_new(command_dir/'stdout.bin',error.stdout or b'')
        write_bytes_new(command_dir/'stderr.bin',error.stderr or b'')
        write_new(command_dir/'exit.json',{'code':None,'timeout':True})
        raise Hold('DOCKER_TIMEOUT_NO_RETRY') from None
    write_bytes_new(command_dir/'stdout.bin',result.stdout)
    write_bytes_new(command_dir/'stderr.bin',result.stderr)
    write_new(command_dir/'exit.json',{'code':result.returncode,'stdoutSha256':hashlib.sha256(result.stdout).hexdigest(),'stderrSha256':hashlib.sha256(result.stderr).hexdigest()})
    if checked: require(result.returncode == 0, 'DOCKER_COMMAND_FAILED')
    return result.stdout.decode('utf-8','replace').strip()


def inventory(value, excluded=None):
    excluded = excluded or {}
    containers = [json.loads(x) for x in docker(value,'ps','-a','--no-trunc','--format','{{json .}}').splitlines() if x]
    networks = [json.loads(x) for x in docker(value,'network','ls','--no-trunc','--format','{{json .}}').splitlines() if x]
    volumes = [json.loads(x) for x in docker(value,'volume','ls','--format','{{json .}}').splitlines() if x]
    container_keys=('ID','Image','Names','State','Networks','Mounts','Ports')
    network_keys=('ID','Name','Driver','Scope')
    volume_keys=('Name','Driver','Scope')
    return {
        'containers':sorted([{k:x.get(k) for k in container_keys} for x in containers if x.get('ID') != excluded.get('containerId')],key=lambda x:x.get('ID','')),
        'networks':sorted([{k:x.get(k) for k in network_keys} for x in networks if x.get('ID') != excluded.get('networkId')],key=lambda x:x.get('ID','')),
        'volumes':sorted([{k:x.get(k) for k in volume_keys} for x in volumes if x.get('Name') != excluded.get('volumeName')],key=lambda x:x.get('Name','')),
    }


def owner_identity(value, receipt, require_port=False, require_network=True):
    # Restrict inspect projection; Config.Env is never read or recorded.
    fields = '{"Id":{{json .Id}},"Image":{{json .Image}},"Name":{{json .Name}},"State":{{json .State.Status}},"Pid":{{json .State.Pid}},"Labels":{{json .Config.Labels}},"Mounts":{{json .Mounts}},"NetworkMode":{{json .HostConfig.NetworkMode}},"Networks":{{json .NetworkSettings.Networks}},"Ports":{{json .NetworkSettings.Ports}}}'
    row = json.loads(docker(value,'inspect','--format',fields,receipt['containerId']))
    require(row['Id'] == receipt['containerId'] and row['Image'] == IMAGE and row['Name'] == '/'+NAME, 'OWNED_CONTAINER_CHANGED')
    labels = {'ccmai.tranche':'CCMAI-RUNTIME-121','ccmai.owner':receipt['ownerNonce']}
    require(all(row['Labels'].get(k) == v for k,v in labels.items()), 'OWNED_LABEL_CHANGED')
    mounts = row['Mounts']
    require(len(mounts) == 1 and mounts[0].get('Type') == 'volume' and mounts[0].get('Name') == NAME and mounts[0].get('Destination') == '/var/lib/mysql', 'OWNED_MOUNT_CHANGED')
    require(row.get('NetworkMode') == NAME, 'OWNED_NETWORK_MODE_CHANGED')
    nets = row['Networks'] or {}
    require(set(nets).issubset({NAME}), 'OWNED_NETWORK_CHANGED')
    if require_network:
        require(set(nets) == {NAME} and nets[NAME].get('NetworkID') == receipt['networkId'], 'OWNED_NETWORK_ID_CHANGED')
    network = json.loads(docker(value,'network','inspect','--format','{{json .}}',receipt['networkId']))
    volume = json.loads(docker(value,'volume','inspect','--format','{{json .}}',NAME))
    require(network.get('Id') == receipt['networkId'] and network.get('Name') == NAME and network.get('Driver') == 'bridge' and network.get('Internal') is False and network.get('Labels') == labels, 'OWNED_NETWORK_IDENTITY_CHANGED')
    options = network.get('Options') or {}
    require(options.get('com.docker.network.bridge.gateway_mode_ipv4') == 'nat' and options.get('com.docker.network.bridge.host_binding_ipv4') == '127.0.0.1', 'OWNED_NETWORK_OPTIONS_CHANGED')
    require(volume.get('Name') == NAME and volume.get('Driver') == 'local' and volume.get('Labels') == labels and (volume.get('Options') or {}) == {} and isinstance(volume.get('Mountpoint'),str) and volume['Mountpoint'], 'OWNED_VOLUME_CHANGED')
    network_identity = {k:network.get(k) for k in ('Id','Name','Created','Driver','Internal','Labels','Options')}
    volume_identity = {k:volume.get(k) for k in ('Name','CreatedAt','Mountpoint','Driver','Labels','Options')}
    if 'networkIdentity' in receipt: require(receipt['networkIdentity'] == network_identity, 'OWNED_NETWORK_REPLACED')
    if 'volumeIdentity' in receipt: require(receipt['volumeIdentity'] == volume_identity, 'OWNED_VOLUME_REPLACED')
    users = docker(value,'ps','-aq','--no-trunc','--filter','volume='+NAME).splitlines()
    net_users = docker(value,'ps','-aq','--no-trunc','--filter','network='+NAME).splitlines()
    require(users == [receipt['containerId']] and net_users == [receipt['containerId']], 'OWNED_RESOURCE_NOT_EXCLUSIVE')
    if require_port:
        require(row['State'] == 'running' and type(row.get('Pid')) is int and row['Pid'] > 0 and published_port(row.get('Ports')) == receipt['port'], 'OWNED_PORT_CHANGED')
    row['networkIdentity'] = network_identity
    row['volumeIdentity'] = volume_identity
    return row


def published_port(ports):
    require(isinstance(ports,dict) and set(ports) == {'3306/tcp'}, 'PORTS_INCOMPLETE')
    binding = ports['3306/tcp']
    require(isinstance(binding,list) and len(binding) == 1 and isinstance(binding[0],dict), 'PORTS_INCOMPLETE')
    item = binding[0]
    require(set(item) == {'HostIp','HostPort'} and item['HostIp'] == '127.0.0.1' and re.fullmatch(r'[0-9]{4,5}',str(item['HostPort'])), 'PORT_NOT_LOOPBACK')
    port = int(item['HostPort'])
    require(1024 <= port <= 65535, 'PORT_OUT_OF_RANGE')
    return port


def route_allowed(method,path,used):
    """Audit the Vite/backend route ledgers against the bounded contract."""
    if method in {'GET','HEAD'}: return True,used
    if method == 'POST' and path in {'/api/v1/auth/login','/api/v1/auth/logout'} and used < 2:
        return True,used+1
    return False,used


def jsonl(path):
    require(Path(path).is_file(), 'LEDGER_MISSING_'+Path(path).name)
    rows = []
    for line in Path(path).read_text(encoding='utf-8').splitlines():
        if line.strip(): rows.append(json.loads(line))
    return rows


def validate_vite_routes(path):
    rows = jsonl(path)
    attempts, responses = {}, {}
    auth_attempts = auth_allowed = 0
    for row in rows:
        require(isinstance(row,dict) and row.get('phase') in {'attempt','response'}, 'VITE_LEDGER_ROW_INVALID')
        sequence = row.get('sequence')
        require(type(sequence) is int and sequence > 0, 'VITE_LEDGER_SEQUENCE_INVALID')
        target = attempts if row['phase'] == 'attempt' else responses
        require(sequence not in target, 'VITE_LEDGER_DUPLICATE_SEQUENCE')
        target[sequence] = row
    require(attempts and set(attempts) == set(responses), 'VITE_LEDGER_RESPONSE_MISSING')
    require(sorted(attempts) == list(range(1,len(attempts)+1)), 'VITE_LEDGER_SEQUENCE_GAP')
    forwarded = Counter()
    observed = []
    for sequence in sorted(attempts):
        attempt, response = attempts[sequence], responses[sequence]
        method, path_value = attempt.get('method'), attempt.get('path')
        require(isinstance(method,str) and isinstance(path_value,str) and (path_value == '/api' or path_value.startswith('/api/')), 'VITE_ROUTE_INVALID')
        expected_allowed, next_auth_allowed = route_allowed(method,path_value,auth_allowed)
        require(attempt.get('allowed') is expected_allowed, 'VITE_ROUTE_ALLOWANCE_MISMATCH')
        if method == 'POST' and path_value in {'/api/v1/auth/login','/api/v1/auth/logout'}:
            auth_attempts += 1
            if expected_allowed: auth_allowed = next_auth_allowed
        require(attempt.get('authAttemptCount') == auth_attempts and attempt.get('authAllowedCount') == auth_allowed, 'VITE_AUTH_LEDGER_MISMATCH')
        require(response.get('method') == method and response.get('path') == path_value, 'VITE_RESPONSE_ROUTE_MISMATCH')
        status = response.get('status')
        require(type(status) is int and 100 <= status <= 599, 'VITE_RESPONSE_STATUS_INVALID')
        if expected_allowed: forwarded[(method,path_value,status)] += 1
        observed.append({'sequence':sequence,'method':method,'path':path_value,'allowed':expected_allowed,'status':status})
    require(auth_attempts <= 2 and auth_allowed <= 2, 'VITE_AUTH_CAP_EXCEEDED')
    return {'attempts':observed,'authAttempts':auth_attempts,'authAllowed':auth_allowed,'forwarded':forwarded}


def validate_go_routes(log_text, manifest):
    route_re = re.compile(r'^R121_ROUTE method=(\S+) path=(\S+) allowed=(true|false) status=([0-9]{3})$')
    auth_re = re.compile(r'^R121_AUTH_ATTEMPT method=(\S+) path=(\S+) attempt=([0-9]+) allowedCount=([0-9]+) allowed=(true|false)$')
    routes, auth_rows, captures = [], [], []
    for line in log_text.splitlines():
        if line.startswith('R121_ROUTE '):
            match = route_re.fullmatch(line)
            require(match is not None, 'GO_ROUTE_LEDGER_INVALID')
            method,path_value,allowed,status = match.groups()
            routes.append({'method':method,'path':path_value,'allowed':allowed == 'true','status':int(status)})
        elif line.startswith('R121_AUTH_ATTEMPT '):
            match = auth_re.fullmatch(line)
            require(match is not None, 'GO_AUTH_LEDGER_INVALID')
            method,path_value,attempt,allowed_count,allowed = match.groups()
            auth_rows.append({'method':method,'path':path_value,'attempt':int(attempt),'allowedCount':int(allowed_count),'allowed':allowed == 'true'})
        elif line.startswith('R121_DETAIL_CAPTURE '):
            captures.append(json.loads(line[len('R121_DETAIL_CAPTURE '):]))
        elif line.startswith('R121_DETAIL_CAPTURE_INVALID'):
            raise Hold('DETAIL_CAPTURE_INVALID')
    require(routes, 'GO_ROUTE_LEDGER_EMPTY')
    auth_attempts = auth_allowed = 0
    for route in routes:
        expected, next_allowed = route_allowed(route['method'],route['path'],auth_allowed)
        require(route['allowed'] is expected, 'GO_ROUTE_ALLOWANCE_MISMATCH')
        require(100 <= route['status'] <= 599, 'GO_ROUTE_STATUS_INVALID')
        if route['method'] == 'POST' and route['path'] in {'/api/v1/auth/login','/api/v1/auth/logout'}:
            auth_attempts += 1
            if expected: auth_allowed = next_allowed
    require(auth_attempts == len(auth_rows) and auth_attempts <= 2 and auth_allowed <= 2, 'GO_AUTH_CAP_EXCEEDED')
    require([row['attempt'] for row in auth_rows] == list(range(1,auth_attempts+1)), 'GO_AUTH_ATTEMPT_SEQUENCE_INVALID')
    require([row['allowedCount'] for row in auth_rows] == [min(index,2) for index in range(1,auth_attempts+1)], 'GO_AUTH_ALLOWED_SEQUENCE_INVALID')
    require(all(row['allowed'] == (row['attempt'] <= 2) for row in auth_rows), 'GO_AUTH_ALLOWANCE_INVALID')
    route_auth = Counter((row['method'],row['path'],row['allowed']) for row in routes if row['method'] == 'POST' and row['path'] in {'/api/v1/auth/login','/api/v1/auth/logout'})
    log_auth = Counter((row['method'],row['path'],row['allowed']) for row in auth_rows)
    require(route_auth == log_auth, 'GO_AUTH_ATTEMPT_ROUTE_MISMATCH')
    detail_path = f"/api/v1/tenants/{manifest.get('tenant')}/intervention-cases/{manifest.get('case')}"
    require(any(row['method'] == 'GET' and row['path'] == detail_path and row['status'] == 200 for row in routes), 'DETAIL_GET_NOT_OBSERVED')
    require(captures, 'DETAIL_CAPTURE_MISSING')
    for capture in captures:
        case_row = capture.get('case') or {}
        require(case_row.get('id') == manifest.get('case'), 'DETAIL_CAPTURE_CASE_MISMATCH')
        require(case_row.get('active_proposal_id') == manifest.get('activeProposalId'), 'DETAIL_CAPTURE_ACTIVE_POINTER_MISMATCH')
        proposals = capture.get('proposals') or []
        require(any(p.get('id') == manifest.get('activeProposalId') and p.get('revision') == 2 for p in proposals), 'DETAIL_CAPTURE_CURRENT_PROPOSAL_MISSING')
        require(any(p.get('id') == manifest.get('laterOffLineageProposalId') and p.get('revision') == 3 for p in proposals), 'DETAIL_CAPTURE_HISTORY_PROPOSAL_MISSING')
        events = capture.get('human_events') or []
        require([event.get('action') for event in events[-3:]] == manifest.get('eventActions'), 'DETAIL_CAPTURE_HISTORY_MISSING')
        require([event.get('reason') for event in events[-3:]][-1] == 'approved execution', 'DETAIL_CAPTURE_SYSTEM_REASON_MISSING')
    return {'routes':routes,'authAttempts':auth_attempts,'authAllowed':auth_allowed,'detailCaptures':len(captures)}


def validate_route_ledgers(vite_path, backend_log, manifest):
    vite = validate_vite_routes(vite_path)
    go = validate_go_routes(backend_log,manifest)
    forwarded = vite['forwarded']
    backend = Counter((row['method'],row['path'],row['status']) for row in go['routes'])
    require(forwarded == backend, 'VITE_BACKEND_ROUTE_LEDGER_MISMATCH')
    require(vite['authAttempts'] == go['authAttempts'] and vite['authAllowed'] == go['authAllowed'], 'AUTH_LEDGER_MISMATCH')
    return {'viteAttempts':len(vite['attempts']),'backendRoutes':len(go['routes']),'authAttempts':go['authAttempts'],'authAllowed':go['authAllowed'],'detailCaptures':go['detailCaptures']}


def validate_fixture_accounting(backend_log, manifest):
    final_re = re.compile(r'^R121_BROWSER_FINAL initial=([0-9a-f]{64}) final=([0-9a-f]{64}) equal=(true|false)$')
    final_rows = [final_re.fullmatch(line) for line in backend_log.splitlines() if line.startswith('R121_BROWSER_FINAL ')]
    effect_rows = [line[len('R121_EFFECT_COUNTS '):] for line in backend_log.splitlines() if line.startswith('R121_EFFECT_COUNTS ')]
    require(len(final_rows) == 1 and final_rows[0] is not None and len(effect_rows) == 1, 'FIXTURE_FINAL_ACCOUNTING_MISSING')
    initial_fingerprint, final_fingerprint, equal = final_rows[0].groups()
    require(equal == 'true' and initial_fingerprint == final_fingerprint == manifest.get('initialFingerprint'), 'FIXTURE_FINGERPRINT_CHANGED')
    counts = json.loads(effect_rows[0])
    initial, final = counts.get('initial'), counts.get('final')
    required = {'attempts','executions','grants','budgets','receipts','runs','snapshots','humanEvents'}
    require(isinstance(initial,dict) and isinstance(final,dict) and required.issubset(initial) and required.issubset(final), 'FIXTURE_EFFECT_COUNTS_INCOMPLETE')
    require(initial == final and initial == manifest.get('initialEffectCounts'), 'FIXTURE_EFFECT_COUNTS_CHANGED')
    return {'initialFingerprint':initial_fingerprint,'finalFingerprint':final_fingerprint,'effectCounts':initial,'providerCalls':0}


class IncompleteMySQLGreeting(Exception):
    def __init__(self, header_byte_count):
        super().__init__('MYSQL_GREETING_EOF')
        self.header_byte_count = header_byte_count


class InvalidMySQLGreeting(Exception):
    def __init__(self, header_byte_count):
        super().__init__('MYSQL_PROTOCOL_INVALID')
        self.header_byte_count = header_byte_count


class MySQLGreetingTimeout(TimeoutError):
    def __init__(self, header_byte_count):
        super().__init__('MYSQL_GREETING_TIMEOUT')
        self.header_byte_count = header_byte_count


class MySQLGreetingReadError(Exception):
    def __init__(self, header_byte_count, error_type):
        super().__init__('MYSQL_GREETING_READ_ERROR')
        self.header_byte_count = header_byte_count
        self.error_type = error_type


def read_mysql_greeting(sock, deadline):
    """Read the complete five-byte protocol header without retaining its payload."""
    header = bytearray()
    while len(header) < 5:
        remaining = deadline-time.monotonic()
        if remaining <= 0: raise MySQLGreetingTimeout(len(header))
        try:
            sock.settimeout(min(MYSQL_PER_READ_SECONDS,remaining))
            chunk = sock.recv(5-len(header))
        except (socket.timeout,TimeoutError):
            raise MySQLGreetingTimeout(len(header)) from None
        except OSError as error:
            raise MySQLGreetingReadError(len(header),type(error).__name__) from None
        if not chunk: raise IncompleteMySQLGreeting(len(header))
        header.extend(chunk)
    if header[4] != 10: raise InvalidMySQLGreeting(len(header))
    return bytes(header)


def wait_mysql(port):
    require(type(port) is int and 1024 <= port <= 65535, 'MYSQL_PORT_INVALID')
    started_campaign = time.monotonic()
    deadline = started_campaign+MYSQL_READY_SECONDS
    attempts = []
    while True:
        remaining = deadline-time.monotonic()
        if remaining <= 0:
            summary = {'status':'HOLD','attemptCount':len(attempts),'elapsedSeconds':round(time.monotonic()-started_campaign,3)}
            append(RUNTIME/'mysql_readiness_summary.jsonl',summary)
            raise Hold('MYSQL_NOT_READY_DEADLINE')
        attempt_started = time.monotonic()
        attempt_epoch = time.time()
        header_byte_count = 0
        phase = 'connect'
        try:
            with socket.create_connection(('127.0.0.1',port),timeout=min(MYSQL_PER_READ_SECONDS,remaining)) as sock:
                phase = 'greeting'
                header = read_mysql_greeting(sock,deadline)
                header_byte_count = len(header)
            outcome = 'ready'
        except InvalidMySQLGreeting as error:
            header_byte_count = error.header_byte_count
            outcome = 'invalid_complete_header'
        except IncompleteMySQLGreeting as error:
            header_byte_count = error.header_byte_count
            outcome = 'transient_eof'
        except MySQLGreetingTimeout as error:
            header_byte_count = error.header_byte_count
            outcome = 'transient_timeout'
        except MySQLGreetingReadError as error:
            header_byte_count = error.header_byte_count
            outcome = 'transient_greeting_error'
        except (socket.timeout,TimeoutError):
            outcome = 'transient_timeout'
        except OSError:
            outcome = 'transient_'+phase+'_error'
        event = {
            'attempt':len(attempts)+1,
            'atEpochSeconds':attempt_epoch,
            'elapsedSeconds':round(time.monotonic()-attempt_started,3),
            'result':outcome,
            'headerByteCount':header_byte_count,
        }
        append(RUNTIME/'mysql_readiness.jsonl',event)
        attempts.append(event)
        if outcome == 'ready':
            return {'status':'READY','attemptCount':len(attempts),'elapsedSeconds':round(time.monotonic()-started_campaign,3),'attempts':attempts}
        if outcome == 'invalid_complete_header':
            append(RUNTIME/'mysql_readiness_summary.jsonl',{'status':'FATAL','attemptCount':len(attempts),'result':outcome,'elapsedSeconds':round(time.monotonic()-started_campaign,3)})
            raise Hold('MYSQL_PROTOCOL_INVALID')
        remaining = deadline-time.monotonic()
        if remaining <= 0:
            append(RUNTIME/'mysql_readiness_summary.jsonl',{'status':'HOLD','attemptCount':len(attempts),'result':'deadline_exhausted','elapsedSeconds':round(time.monotonic()-started_campaign,3)})
            raise Hold('MYSQL_NOT_READY_DEADLINE')
        time.sleep(min(MYSQL_RETRY_DELAY_SECONDS,remaining))


def validate_powershell_pin(value):
    path_value = value.get('powershellPath')
    digest = value.get('powershellSha256')
    require(isinstance(path_value,str) and path_value and isinstance(digest,str) and HEX64.fullmatch(digest), 'POWERSHELL_PIN_MISSING')
    path = Path(path_value)
    canonical_suffix = '/windowspowershell/v1.0/powershell.exe'
    require(path.is_absolute() and path.name.lower() == 'powershell.exe' and path.as_posix().lower().endswith(canonical_suffix), 'CANONICAL_WINDOWS_POWERSHELL_REQUIRED')
    system_root = os.environ.get('SystemRoot') or os.environ.get('WINDIR')
    require(isinstance(system_root,str) and system_root, 'WINDOWS_SYSTEM_ROOT_MISSING')
    canonical_path = Path(system_root)/'System32'/'WindowsPowerShell'/'v1.0'/'powershell.exe'
    require(os.path.normcase(str(path.resolve())) == os.path.normcase(str(canonical_path.resolve())), 'CANONICAL_WINDOWS_POWERSHELL_REQUIRED')
    require(path.is_file() and sha(path) == digest, 'POWERSHELL_PIN_CHANGED')
    return path,digest


def _listener_observation_dir(name,port):
    safe_name = re.sub(r'[^a-z0-9_-]','_',str(name).lower())
    base = RUNTIME/'listener_observations'
    base.mkdir(parents=True,exist_ok=True)
    suffix = str(port) if type(port) is int else 'unknown'
    index = 1
    while True:
        path = base/f'{index:04d}-{safe_name}-{suffix}'
        try:
            path.mkdir(exist_ok=False)
            return path
        except FileExistsError:
            index += 1


def host_listener_observation(value,name,port):
    """Capture one pinned Windows listener-table query; uncertainty stays HOLD."""
    directory = _listener_observation_dir(name,port)
    base = {
        'observer':name,
        'port':port if type(port) is int else None,
        'listenerAbsent':False,
        'evidenceKind':'HOST_LISTENER_UNKNOWN',
        'argv':None,
        'powershellSha256':None,
    }
    stdout = b''
    stderr = b''
    exit_code = None
    timeout = False
    invoked = False
    error = None
    command = None
    command_hash = None
    path = None
    digest = None
    if not IS_WINDOWS:
        error = 'WINDOWS_POWERSHELL_REQUIRED'
        base['evidenceKind'] = 'HOST_LISTENER_UNSUPPORTED_PLATFORM'
    elif type(port) is not int or not 1 <= port <= 65535:
        error = 'LISTENER_PORT_INVALID'
        base['evidenceKind'] = 'HOST_LISTENER_INVALID_PORT'
    else:
        try:
            path,digest = validate_powershell_pin(value)
            command = (
                "$ErrorActionPreference='Stop'; "
                f"$p=[int]{port}; "
                "$rows=@(Get-NetTCPConnection -State Listen -ErrorAction Stop | "
                "Where-Object { [int]$_.LocalPort -eq $p } | "
                "Select-Object -Property LocalAddress,LocalPort,OwningProcess); "
                "ConvertTo-Json -InputObject $rows -Compress"
            )
            argv = [str(path),'-NoLogo','-NoProfile','-NonInteractive','-ExecutionPolicy','Bypass','-Command',command]
            command_hash = hashlib.sha256(command.encode('utf-8')).hexdigest()
            base.update({'argv':argv,'powershellPath':str(path),'powershellSha256':digest,'commandSha256':command_hash})
            write_new(directory/'command.json',{'argv':argv,'observer':name,'port':port,'powershellPath':str(path),'powershellSha256':digest,'commandSha256':command_hash,'timeoutSeconds':POWERSHELL_QUERY_SECONDS})
            try:
                invoked = True
                result = subprocess.run(argv,cwd=ROOT,env=minimal_env(),capture_output=True,timeout=POWERSHELL_QUERY_SECONDS)
                stdout,stderr,exit_code = result.stdout,result.stderr,result.returncode
            except subprocess.TimeoutExpired as exception:
                stdout = exception.stdout or b''
                stderr = exception.stderr or b''
                timeout = True
                error = 'LISTENER_QUERY_TIMEOUT'
                base['evidenceKind'] = 'HOST_LISTENER_QUERY_TIMEOUT'
            except OSError as exception:
                error = type(exception).__name__
                base['evidenceKind'] = 'HOST_LISTENER_QUERY_ERROR'
        except Exception as exception:
            error = type(exception).__name__+':'+str(exception)
            base['evidenceKind'] = 'HOST_LISTENER_PIN_OR_COMMAND_ERROR'
    write_bytes_new(directory/'stdout.bin',stdout)
    write_bytes_new(directory/'stderr.bin',stderr)
    command_record = {
        'argv':base.get('argv'),
        'observer':name,
        'port':base['port'],
        'powershellPath':base.get('powershellPath'),
        'powershellSha256':base.get('powershellSha256'),
        'commandSha256':command_hash,
        'invoked':invoked,
        'timeout':timeout,
        'exitCode':exit_code,
        'stdoutSha256':hashlib.sha256(stdout).hexdigest(),
        'stderrSha256':hashlib.sha256(stderr).hexdigest(),
    }
    if error is not None: command_record['error'] = error
    write_new(directory/'exit.json',command_record)
    if error is None:
        if timeout:
            base['evidenceKind'] = 'HOST_LISTENER_QUERY_TIMEOUT'
        elif exit_code != 0 or stderr:
            error = 'LISTENER_QUERY_FAILED'
            base['evidenceKind'] = 'HOST_LISTENER_QUERY_ERROR'
        else:
            try:
                rows = json.loads(stdout.decode('utf-8'))
                require(isinstance(rows,list),'LISTENER_RESULT_NOT_ARRAY')
                require(all(isinstance(row,dict) and type(row.get('LocalPort')) is int and row['LocalPort'] == port for row in rows),'LISTENER_RESULT_ROW_INVALID')
                base['rows'] = rows
                if rows:
                    base['evidenceKind'] = 'HOST_LISTENER_PRESENT'
                else:
                    base['listenerAbsent'] = True
                    base['evidenceKind'] = 'HOST_LISTENER_ABSENT_MAPPING_REMOVED'
            except Exception as exception:
                error = type(exception).__name__+':'+str(exception)
                base['evidenceKind'] = 'HOST_LISTENER_QUERY_MALFORMED'
    if error is not None: base['error'] = error
    base['capturedAtEpochSeconds'] = time.time()
    base['raw'] = {'directory':directory.relative_to(ROOT).as_posix(),'stdout':'stdout.bin','stderr':'stderr.bin','exit':'exit.json','command':'command.json' if (directory/'command.json').exists() else None}
    write_new(directory/'result.json',base)
    return base


def require_stopped_container(row):
    require(row.get('State') == 'exited' and type(row.get('Pid')) is int and row['Pid'] == 0 and row.get('Ports') == {}, 'OWNED_FIXTURE_TERMINAL_STATE_UNPROVEN')


def process_creation_identity(process):
    pid = process if isinstance(process,int) else process.pid
    if os.name == 'nt':
        import ctypes
        from ctypes import wintypes
        creation = wintypes.FILETIME()
        exit_time = wintypes.FILETIME()
        kernel = ctypes.WinDLL('kernel32',use_last_error=True)
        owned_handle = None
        handle = getattr(process,'_handle',None)
        if handle is None:
            open_process = kernel.OpenProcess
            open_process.argtypes = [wintypes.DWORD,wintypes.BOOL,wintypes.DWORD]
            open_process.restype = wintypes.HANDLE
            owned_handle = open_process(0x1000,False,pid)
            require(bool(owned_handle), 'PROCESS_CREATION_ID_UNAVAILABLE')
            handle = owned_handle
        get_times = kernel.GetProcessTimes
        get_times.argtypes = [wintypes.HANDLE,ctypes.POINTER(wintypes.FILETIME),ctypes.POINTER(wintypes.FILETIME),ctypes.POINTER(wintypes.FILETIME),ctypes.POINTER(wintypes.FILETIME)]
        get_times.restype = wintypes.BOOL
        try:
            require(get_times(wintypes.HANDLE(handle),ctypes.byref(creation),ctypes.byref(exit_time),ctypes.byref(wintypes.FILETIME()),ctypes.byref(wintypes.FILETIME())), 'PROCESS_CREATION_ID_UNAVAILABLE')
        finally:
            if owned_handle is not None: kernel.CloseHandle(owned_handle)
        ticks = (creation.dwHighDateTime << 32) | creation.dwLowDateTime
        return f'windows-filetime-{ticks:016x}'
    stat = Path(f'/proc/{pid}/stat')
    require(stat.is_file(), 'PROCESS_CREATION_ID_UNAVAILABLE')
    fields = stat.read_text(encoding='ascii').rsplit(')',1)[-1].split()
    require(len(fields) > 19, 'PROCESS_CREATION_ID_UNAVAILABLE')
    return f'proc-start-ticks-{fields[19]}'


def setup():
    value = source_admission()
    admitted_source = source_snapshot(value)
    require(not RUNTIME.exists(), 'R121_RUNTIME_ALREADY_EXISTS')
    RUNTIME.mkdir(parents=True,exist_ok=False)
    config = RUNTIME/'docker-config'
    config.mkdir(exist_ok=False)
    write_new(config/'config.json',{})
    before = inventory(value)
    require(not any(x.get('Names') == NAME for x in before['containers']), 'R121_CONTAINER_NAME_EXISTS')
    require(not any(x.get('Name') == NAME for x in before['networks']+before['volumes']), 'R121_RESOURCE_NAME_EXISTS')
    nonce = secrets.token_hex(16)
    labels = ['--label','ccmai.tranche=CCMAI-RUNTIME-121','--label','ccmai.owner='+nonce]
    partial = {'ownerNonce':nonce,'protectedBefore':before,'volumeName':NAME}
    write_new(RUNTIME/'setup_intent.json',partial)
    try:
        network = docker(value,'network','create','--driver','bridge','--opt','com.docker.network.bridge.gateway_mode_ipv4=nat','--opt','com.docker.network.bridge.host_binding_ipv4=127.0.0.1',*labels,NAME,mutation=True)
        require(HEX64.fullmatch(network), 'NETWORK_ID_INVALID')
        partial['networkId'] = network
        write_new(RUNTIME/'network_created.json',partial)
        require(docker(value,'volume','create',*labels,NAME,mutation=True) == NAME, 'VOLUME_ID_INVALID')
        write_new(RUNTIME/'volume_created.json',partial)
        container = docker(value,'create','--pull','never','--name',NAME,*labels,'--network',NAME,'--mount','type=volume,src='+NAME+',dst=/var/lib/mysql','-p','127.0.0.1::3306','-e','MYSQL_ROOT_PASSWORD='+PASSWORD,'-e','MYSQL_DATABASE='+DATABASE,IMAGE,mutation=True)
        require(HEX64.fullmatch(container), 'CONTAINER_ID_INVALID')
        partial['containerId'] = container
        write_new(RUNTIME/'container_created.json',partial)
        identity = owner_identity(value,partial,require_network=False)
        partial['networkIdentity'] = identity['networkIdentity']
        partial['volumeIdentity'] = identity['volumeIdentity']
        docker(value,'start',container,mutation=True)
        row = owner_identity(value,partial)
        port = published_port(row.get('Ports'))
        partial['port'] = port
        partial['mysqlReadiness'] = wait_mysql(port)
        require(source_snapshot(source_admission()) == admitted_source, 'SOURCE_PINS_CHANGED_DURING_SETUP')
        write_new(RUNTIME/'setup_receipt.json',partial)
        print(json.dumps({'result':'SETUP_READY','containerId':container,'networkId':network,'volumeName':NAME,'port':port,'ownerNonce':nonce},sort_keys=True),flush=True)
    except Exception as exc:
        append(RUNTIME/'setup_errors.jsonl',{'error':type(exc).__name__+':'+str(exc)})
        if 'containerId' in partial:
            try: write_new(RUNTIME/'setup_cleanup.json',stop_fixture(value,partial))
            except Exception as cleanup: append(RUNTIME/'setup_errors.jsonl',{'cleanupError':type(cleanup).__name__+':'+str(cleanup)})
        raise


def stop_fixture(value, receipt):
    row = owner_identity(value,receipt,require_port=False)
    if row['State'] == 'running':
        docker(value,'stop','--time','10',receipt['containerId'],mutation=True,timeout=30)
    final = owner_identity(value,receipt,require_port=False)
    try:
        require_stopped_container(final)
        docker_terminal = True
    except Hold:
        docker_terminal = False
    listener = host_listener_observation(value,'fixture',receipt.get('port'))
    protected_equal = inventory(value,receipt) == receipt['protectedBefore']
    return {
        'containerState':final.get('State'),
        'containerPid':final.get('Pid'),
        'containerPorts':final.get('Ports'),
        'dockerTerminalObserved':docker_terminal,
        'listenerAbsent':listener['listenerAbsent'],
        'evidenceKind':listener['evidenceKind'],
        'listenerObservation':listener,
        'protectedEqual':protected_equal,
    }


def resource_admission(setup_receipt):
    value = committed(RESOURCE)
    for key,expected in {'trancheId':'CCMAI-RUNTIME-121','database':DATABASE,'host':'127.0.0.1','port':setup_receipt['port'],'containerId':setup_receipt['containerId'],'networkId':setup_receipt['networkId'],'volumeName':NAME,'ownerNonce':setup_receipt['ownerNonce'],'networkIdentity':setup_receipt['networkIdentity'],'volumeIdentity':setup_receipt['volumeIdentity']}.items():
        require(value.get(key) == expected, 'RESOURCE_ADMISSION_MISMATCH_'+key)
    require(value.get('protectedBefore') == setup_receipt['protectedBefore'], 'PROTECTED_BASELINE_MISMATCH')
    return value


def wait_ready(process, output, deadline):
    while time.monotonic() < deadline:
        require(process.poll() is None, 'BACKEND_EXITED_BEFORE_READY')
        if output.exists():
            for line in output.read_text(encoding='utf-8',errors='replace').splitlines():
                match = READY.fullmatch(line)
                if match:
                    manifest = json.loads(match.group(2))
                    require(manifest.get('tranche') == 'CCMAI-RUNTIME-121' and manifest.get('synthetic') is True, 'BROWSER_MANIFEST_INVALID')
                    return match.group(1),manifest
        time.sleep(.2)
    raise Hold('BACKEND_READY_TIMEOUT')


def launch(argv, env, output, name):
    append(RUNTIME/'effect_reservations.jsonl',{'kind':'process','name':name,'argv':argv,'at':time.time()})
    write_new(RUNTIME/'process_intents'/f'{name}.json',{'name':name,'argv':argv,'cwd':str(ROOT),'intentAt':time.time(),'supervisorPid':os.getpid()})
    stream = output.open('xb')
    try:
        process = subprocess.Popen(argv,cwd=ROOT,env=env,stdout=stream,stderr=subprocess.STDOUT,creationflags=getattr(subprocess,'CREATE_NO_WINDOW',0))
    finally:
        stream.close()
    try:
        creation_id = process_creation_identity(process)
        process._r121_creation_id = creation_id
        append(RUNTIME/'processes.jsonl',{'name':name,'pid':process.pid,'creationIdentity':creation_id,'at':time.time()})
    except Exception:
        process.terminate()
        try: process.wait(timeout=8)
        except subprocess.TimeoutExpired: process.kill(); process.wait(timeout=8)
        raise
    return process


def end_process(value,process,name,port):
    if process is None:
        listener = host_listener_observation(value,name,port)
        return {'name':name,'launched':False,'waitObserved':False,'listenerPort':port,'listenerAbsent':listener['listenerAbsent'],'evidenceKind':listener['evidenceKind'],'listenerObservation':listener}
    if process.poll() is None:
        append(RUNTIME/'effect_reservations.jsonl',{'kind':'process-terminate','name':name,'pid':process.pid,'creationIdentity':getattr(process,'_r121_creation_id',None),'at':time.time()})
        process.terminate()
        try: process.wait(timeout=12)
        except subprocess.TimeoutExpired:
            append(RUNTIME/'effect_reservations.jsonl',{'kind':'process-kill','name':name,'pid':process.pid,'creationIdentity':getattr(process,'_r121_creation_id',None),'at':time.time()})
            process.kill(); process.wait(timeout=8)
    else:
        process.wait(timeout=1)
    require(process.returncode is not None, 'CHILD_EXIT_NOT_OBSERVED')
    listener = host_listener_observation(value,name,port)
    return {'name':name,'pid':process.pid,'creationIdentity':getattr(process,'_r121_creation_id',None),'exit':process.returncode,'waitObserved':True,'listenerPort':port,'listenerAbsent':listener['listenerAbsent'],'evidenceKind':listener['evidenceKind'],'listenerObservation':listener}


def request_stop_file(reason):
    path = RUNTIME/'stop_requested'
    append(RUNTIME/'effect_reservations.jsonl',{'kind':'stop-file','reason':reason,'path':str(path),'at':time.time()})
    if path.exists(): return
    path.parent.mkdir(parents=True,exist_ok=True)
    with path.open('x',encoding='utf-8') as stream:
        stream.write(reason+'\n')
        stream.flush(); os.fsync(stream.fileno())


def logged_backend_port(path):
    if not Path(path).is_file(): return None
    for line in Path(path).read_text(encoding='utf-8',errors='replace').splitlines():
        match = READY.fullmatch(line)
        if match:
            parsed = urlsplit(match.group(1))
            if parsed.hostname == '127.0.0.1' and parsed.port: return parsed.port
    return None


def serve():
    source = source_admission()
    admitted_source = source_snapshot(source)
    receipt = read(RUNTIME/'setup_receipt.json')
    admitted_resource = resource_admission(receipt)
    admitted_resource_sha = sha(RESOURCE)
    require(not (RUNTIME/'serve_intent.json').exists(), 'SERVE_ALREADY_USED')
    require(not (RUNTIME/'stop_requested').exists(), 'STOP_REQUEST_ALREADY_EXISTS')
    owner_identity(source,receipt,require_port=True)
    require(inventory(source,receipt) == receipt['protectedBefore'], 'PROTECTED_INVENTORY_CHANGED')
    supervisor_creation = process_creation_identity(os.getpid())
    write_new(RUNTIME/'serve_intent.json',{'pid':os.getpid(),'creationIdentity':supervisor_creation,'startedAt':time.time(),'deadlineSeconds':CAMPAIGN_SECONDS,'source':admitted_source,'resourceAdmissionSha256':sha(RESOURCE)})
    backend = vite = None
    backend_port = None
    manifest = None
    result = {'result':'HOLD','terminal':False}
    try:
        deadline = time.monotonic()+CAMPAIGN_SECONDS-180
        env = minimal_env()
        env.update({'R121_BROWSER_FIXTURE':'1','R121_MYSQL_DSN':f'root:{PASSWORD}@tcp(127.0.0.1:{receipt["port"]})/{DATABASE}?parseTime=true','R121_RESOURCE_ADMISSION':RESOURCE.relative_to(ROOT).as_posix(),'R121_STOP_FILE':str(RUNTIME/'stop_requested')})
        backend = launch([str(checked_file(source['binaryPath'])),'-test.run','^TestR121BrowserFixture$','-test.v'],env,RUNTIME/'backend.log','backend')
        origin,manifest = wait_ready(backend,RUNTIME/'backend.log',min(deadline,time.monotonic()+60))
        parsed_origin = urlsplit(origin)
        require(parsed_origin.hostname == '127.0.0.1' and parsed_origin.port is not None, 'BACKEND_ORIGIN_INVALID')
        backend_port = parsed_origin.port
        write_new(RUNTIME/'browser_manifest.json',manifest)
        template = (ROOT/'scripts/r121_vite_config.mjs.template').read_text(encoding='utf-8')
        for key,replacement in {'__R121_FRONTEND_ROOT__':str(ROOT/'frontend'),'__R121_RUNTIME__':str(RUNTIME),'__R121_API_ORIGIN__':origin,'__R121_NONCE__':secrets.token_hex(16)}.items():
            template = template.replace(key,json.dumps(replacement))
        require('__R121_' not in template and 'envDir: false' in template and 'strictPort:true' in template, 'VITE_CONFIG_INVALID')
        config = RUNTIME/'vite_config.mjs'
        config.write_text(template,encoding='utf-8')
        vite_cli = ROOT/'frontend/node_modules/vite/bin/vite.js'
        require(vite_cli.is_file(), 'VITE_CLI_MISSING')
        vite = launch([source['nodePath'],str(vite_cli),'--config',str(config),'--host','127.0.0.1','--port','3019','--strictPort'],minimal_env(),RUNTIME/'vite.log','vite')
        health = re.search(r'const nonce = ("[0-9a-f]{32}")',template)
        require(health is not None,'HEALTH_NONCE_MISSING')
        expected = json.loads(health.group(1))
        health_deadline = min(deadline,time.monotonic()+45)
        while time.monotonic() < health_deadline:
            require(vite.poll() is None,'VITE_EXITED_BEFORE_READY')
            try:
                with urllib.request.urlopen('http://127.0.0.1:3019/__r121_health',timeout=2) as response:
                    if response.read().decode() == expected: break
            except OSError: time.sleep(.2)
        else: raise Hold('VITE_HEALTH_TIMEOUT')
        write_new(RUNTIME/'campaign_ready.json',{'uiOrigin':'http://127.0.0.1:3019','apiOrigin':origin,'manifest':manifest,'startedAt':time.time()})
        print(json.dumps({'result':'CAMPAIGN_READY','uiOrigin':'http://127.0.0.1:3019','manifest':manifest},ensure_ascii=False),flush=True)
        while time.monotonic() < deadline:
            require(backend.poll() is None and vite.poll() is None,'CHILD_EXITED_DURING_CAMPAIGN')
            if (RUNTIME/'stop_requested').is_file(): break
            time.sleep(.25)
        else: raise Hold('CAMPAIGN_DEADLINE')
        result['result'] = 'CAMPAIGN_STOP_REQUESTED'
    except Exception as exc:
        result['error'] = type(exc).__name__+':'+str(exc)
    finally:
        terminal_errors = []
        # Ask the Go fixture to emit its final durable fingerprint before closing its listener.
        try: request_stop_file('supervisor-finally')
        except Exception as exc: terminal_errors.append('STOP_REQUEST_FAILED:'+type(exc).__name__)
        if backend is not None:
            try: backend.wait(timeout=25)
            except subprocess.TimeoutExpired: terminal_errors.append('BACKEND_FINAL_TIMEOUT')
        if backend_port is None: backend_port = logged_backend_port(RUNTIME/'backend.log')
        result['children'] = []
        for child,name,port in ((vite,'vite',3019),(backend,'backend',backend_port)):
            try:
                child_result = end_process(source,child,name,port)
                result['children'].append(child_result)
                if not child_result.get('waitObserved'):
                    terminal_errors.append(name.upper()+'_WAIT_UNPROVEN')
                if not child_result.get('listenerAbsent'):
                    terminal_errors.append(name.upper()+'_LISTENER_CLOSE_UNPROVEN')
                if name == 'backend' and child_result.get('exit') != 0:
                    terminal_errors.append('BACKEND_EXIT_NOT_ZERO')
            except Exception as exc:
                result['children'].append({'name':name,'cleanupError':type(exc).__name__+':'+str(exc)})
                terminal_errors.append(name.upper()+'_PROCESS_CLEANUP_FAILED')
        backend_log = RUNTIME/'backend.log'
        if manifest is not None and backend_log.is_file() and (RUNTIME/'vite_routes.jsonl').is_file():
            try: result['routeAccounting'] = validate_route_ledgers(RUNTIME/'vite_routes.jsonl',backend_log.read_text(encoding='utf-8',errors='replace'),manifest)
            except Exception as exc: terminal_errors.append('ROUTE_ACCOUNTING_FAILED:'+type(exc).__name__+':'+str(exc))
            try: result['fixtureAccounting'] = validate_fixture_accounting(backend_log.read_text(encoding='utf-8',errors='replace'),manifest)
            except Exception as exc: terminal_errors.append('FIXTURE_ACCOUNTING_FAILED:'+type(exc).__name__+':'+str(exc))
        else:
            terminal_errors.append('FINAL_EVIDENCE_MISSING')
        try: result['sourceAfter'] = verify_source_snapshot(admitted_source)
        except Exception as exc: terminal_errors.append('SOURCE_RECHECK_FAILED:'+type(exc).__name__+':'+str(exc))
        try:
            result['resourceAdmissionAfter'] = resource_admission(receipt)
            require(sha(RESOURCE) == admitted_resource_sha and result['resourceAdmissionAfter'] == admitted_resource, 'RESOURCE_ADMISSION_CHANGED_DURING_CAMPAIGN')
        except Exception as exc: terminal_errors.append('RESOURCE_ADMISSION_RECHECK_FAILED:'+type(exc).__name__+':'+str(exc))
        try:
            result['fixture'] = stop_fixture(source,receipt)
            if not result['fixture'].get('dockerTerminalObserved'):
                terminal_errors.append('FIXTURE_DOCKER_TERMINAL_STATE_UNPROVEN')
            if not result['fixture'].get('listenerAbsent'):
                terminal_errors.append('FIXTURE_LISTENER_CLOSE_UNPROVEN')
            if not result['fixture'].get('protectedEqual'):
                terminal_errors.append('PROTECTED_INVENTORY_CHANGED')
        except Exception as exc: terminal_errors.append('FIXTURE_CLEANUP_FAILED:'+type(exc).__name__+':'+str(exc))
        try: result['sourceAfterCleanup'] = verify_source_snapshot(admitted_source)
        except Exception as exc: terminal_errors.append('SOURCE_FINAL_RECHECK_FAILED:'+type(exc).__name__+':'+str(exc))
        result['terminalErrors'] = terminal_errors
        result['terminal'] = True
        result['finishedAt'] = time.time()
        if terminal_errors: result['result'] = 'HOLD'
        write_new(RUNTIME/'terminal.json',result)
        print(json.dumps(result,sort_keys=True),flush=True)
    require(result.get('result') == 'CAMPAIGN_STOP_REQUESTED' and not terminal_errors and result.get('fixture',{}).get('dockerTerminalObserved') and result.get('fixture',{}).get('listenerAbsent') and result.get('fixture',{}).get('protectedEqual') and result['children'][1].get('exit') == 0 and all(child.get('waitObserved') and child.get('listenerAbsent') for child in result['children']), 'CAMPAIGN_TERMINAL_HOLD')


def stop():
    require((RUNTIME/'serve_intent.json').is_file() and not (RUNTIME/'terminal.json').exists(), 'NO_ACTIVE_CAMPAIGN')
    write_new(RUNTIME/'stop_intent.json',{'pid':os.getpid(),'creationIdentity':process_creation_identity(os.getpid()),'requestedAt':time.time(),'path':str(RUNTIME/'stop_requested')})
    request_stop_file('explicit-stop-command')
    print('R121_STOP_REQUESTED',flush=True)


def inspect():
    print(json.dumps({'sourceAdmissionPresent':SOURCE.is_file(),'resourceAdmissionPresent':RESOURCE.is_file(),'runtimePresent':RUNTIME.exists(),'setupReceiptPresent':(RUNTIME/'setup_receipt.json').is_file(),'serveIntentPresent':(RUNTIME/'serve_intent.json').is_file(),'terminalPresent':(RUNTIME/'terminal.json').is_file()},sort_keys=True))


def main():
    parser=argparse.ArgumentParser()
    parser.add_argument('command',choices=['inspect','setup','serve','stop'])
    args=parser.parse_args()
    {'inspect':inspect,'setup':setup,'serve':serve,'stop':stop}[args.command]()


if __name__ == '__main__':
    try: main()
    except Exception as error:
        print(type(error).__name__+':'+str(error),file=sys.stderr)
        raise SystemExit(1)
