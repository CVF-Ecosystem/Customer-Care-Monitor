// R150 finite Windows A1/A2 VERIFY owner. Do not invoke before committed
// audit/registry, pre-dispatch PASS, and the separate effect admission.
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import net from 'node:net';
import {OwnershipLedger, negativeProbe, waitOriginal, validateSourceManifest, createHandleCustody, replaceOwnedChrome, terminalChromeDecision} from './diagnostic_helpers.mjs';
import {spawn, spawnSync} from 'node:child_process';
import {randomUUID, createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';

const [pkgRoot, manifestPath, chromeExe, outputDir, projectRoot] = process.argv.slice(2);
if (![pkgRoot, manifestPath, chromeExe, outputDir, projectRoot].every(Boolean)) {
  throw new Error('five fixed absolute paths required');
}
const fixed = {
  project: 'D:/UNG DUNG AI/TOOL AI 2026/CVF-Workspace/Customer-Care-Monitor',
  package: 'C:/Users/DELL/AppData/Local/Programs/Antigravity/resources/app.asar.unpacked/node_modules/chrome-devtools-mcp',
  manifest: 'D:/UNG DUNG AI/TOOL AI 2026/CVF-Workspace/Customer-Care-Monitor/docs/reviews/probes/r144_a3_package_metadata_20261105/root_npm_full_tree_manifest.json',
  chrome: 'C:/Program Files/Google/Chrome/Application/chrome.exe',
  output: 'D:/UNG DUNG AI/TOOL AI 2026/CVF-Workspace/Customer-Care-Monitor/docs/reviews/probes/r151_evidence_capture_20261105',
  antigravity: 'C:/Users/DELL/AppData/Local/Programs/Antigravity/Antigravity.exe',
  temp: 'C:/Users/DELL/AppData/Local/Temp',
};
function real(p) { return fs.realpathSync.native(p).toLowerCase(); }
for (const name of ['project','package','manifest','chrome','output','temp']) {
  if (real(fixed[name]) !== path.resolve(fixed[name]).toLowerCase()) throw new Error(`reparse or redirected ${name} path`);
}
for (const [name, given] of Object.entries({project:projectRoot,package:pkgRoot,
    manifest:manifestPath,chrome:chromeExe,output:outputDir})) {
  if (real(given) !== real(fixed[name])) throw new Error(`unadmitted ${name} path`);
}
if (path.basename(chromeExe).toLowerCase() !== 'chrome.exe' ||
    real(process.env.TEMP || os.tmpdir()) !== real(fixed.temp)) throw new Error('binary/temp path guard');
if (!/^v(2[2-9]|[3-9]\d)\./.test(process.version) || typeof WebSocket !== 'function') {
  throw new Error('Node >=22 built-in WebSocket required');
}
function lfHash(p) { return createHash('sha256').update(fs.readFileSync(p).toString('utf8').replace(/\r\n/g,'\n')).digest('hex'); }
if (lfHash(manifestPath) !== '272c516aa8513252e3613748406fc49d77247650e2f8610ada6b49bd4a19b398') {
  throw new Error('pinned A3 manifest mismatch');
}
const projectFile = p => path.join(projectRoot,p);
const admission = JSON.parse(fs.readFileSync(projectFile('CVF_SESSION/authority/CCMAI-RUNTIME-151-A1A2-EFFECT-ADMISSION.json'),'utf8'));
if (admission.trancheId !== 'CCMAI-RUNTIME-151' || admission.decision !== 'ADMITTED_BOUNDED_VERIFY_ONLY') throw new Error('effect admission missing');
for (const field of ['workOrder','audit','independentAuditReview','independentSourceReview']) {
  if (lfHash(projectFile(admission[field].path)) !== admission[field].lfSha256) throw new Error(`${field} admission pin mismatch`);
}
const sourcePaths=['owner_runner.mjs','diagnostic_helpers.mjs','process_snapshot.ps1','prestart_full_tree.ps1','actor_call.mjs'].map(p=>'docs/reviews/probes/r151_evidence_capture_20261105/'+p);
validateSourceManifest({manifestBytes:fs.readFileSync(projectFile(admission.sourceManifest.path)),expectedManifestSha256:admission.sourceManifest.sha256,reviewText:fs.readFileSync(projectFile(admission.independentSourceReview.path),'utf8'),requiredPaths:sourcePaths,readFile:p=>fs.readFileSync(projectFile(p))});
const coverage = JSON.parse(fs.readFileSync(projectFile('docs/milestone-coverage-authority.json'),'utf8'));
if (coverage.repairControls?.M3?.audit?.path !== admission.audit.path ||
    coverage.repairControls.M3.audit.sha256 !== admission.audit.lfSha256) throw new Error('repair registry pointer mismatch');
const state = JSON.parse(fs.readFileSync(projectFile('CVF_SESSION/ACTIVE_SESSION_STATE.json'),'utf8'));
if (state.activeTranche !== admission.trancheId || !['WORK_ORDER','BUILD'].includes(state.currentMode)) throw new Error('active R151 continuity mismatch');
const gate = spawnSync('python',['-B','scripts/cvf_gates/cvf_downstream_gate_runner.py','run',
  '--phase','pre-dispatch','--project-root','.','--packet',admission.workOrder.path],
  {cwd:projectRoot,encoding:'utf8',timeout:120000,maxBuffer:4_000_000});
if (gate.status !== 0 || !/RESULT:\s*PASS/.test(gate.stdout || '')) throw new Error(`pre-dispatch gate failed: ${gate.stdout || gate.stderr}`);
const pinnedPaths = [
  'CVF_SESSION/authority/CCMAI-RUNTIME-151-A1A2-EFFECT-ADMISSION.json',
  'CVF_SESSION/ACTIVE_SESSION_STATE.json',
  'docs/milestone-coverage-authority.json',
  admission.workOrder.path, admission.audit.path, admission.independentAuditReview.path,
  'docs/reviews/probes/r151_evidence_capture_20261105/owner_runner.mjs',
  'docs/reviews/probes/r151_evidence_capture_20261105/prestart_full_tree.ps1',
  'docs/reviews/probes/r151_evidence_capture_20261105/process_snapshot.ps1',
  'docs/reviews/probes/r151_evidence_capture_20261105/actor_call.mjs',
  'docs/reviews/probes/r151_evidence_capture_20261105/diagnostic_helpers.mjs',
  admission.independentSourceReview.path, admission.sourceManifest.path,
];
const committed = spawnSync('git',['status','--porcelain','--untracked-files=all','--',...pinnedPaths],
  {cwd:projectRoot,encoding:'utf8',timeout:30000});
if (committed.status !== 0 || committed.stdout.trim()) throw new Error(`admission or probe file not committed: ${committed.stdout || committed.stderr}`);
const powershellExe = path.join(process.env.SystemRoot || 'C:/Windows',
  'System32/WindowsPowerShell/v1.0/powershell.exe');
const originalEnv = {...process.env};
const sessionId = randomUUID();
const tempRoot = path.join(originalEnv.TEMP || os.tmpdir(), `cvf-r151-${sessionId}`);
const profile1 = path.join(tempRoot, 'chrome1');
const profile2 = path.join(tempRoot, 'chrome2');
const evidenceRoot = path.join(tempRoot, 'mcp-evidence');
for (const dir of [tempRoot, profile1, profile2, evidenceRoot, outputDir]) {
  fs.mkdirSync(dir, {recursive: true});
}
const osKeys = ['SystemRoot','WINDIR','USERPROFILE','APPDATA','LOCALAPPDATA','USERNAME',
  'USERDOMAIN','COMSPEC','PATH','PATHEXT','HOMEDRIVE','HOMEPATH','ProgramData',
  'ProgramFiles','CommonProgramFiles','OS','PROCESSOR_ARCHITECTURE'];
const env = {};
for (const key of osKeys) if (originalEnv[key] !== undefined) env[key] = originalEnv[key];
env.TEMP = tempRoot;
env.TMP = tempRoot;
env.CHROME_DEVTOOLS_MCP_SESSION_ID = sessionId;
env.CHROME_DEVTOOLS_MCP_NO_UPDATE_CHECKS = '1';
env.CHROME_DEVTOOLS_MCP_NO_USAGE_STATISTICS = '1';
for (const key of Object.keys(process.env)) delete process.env[key];
Object.assign(process.env, env);
if (Object.keys(env).some(k => /^(XDG_|NODE_|npm_|PUPPETEER_|DEBUG$|CI$|DISPLAY$|CHROME_DEVTOOLS_MCP_CRASH_ON_UNCAUGHT$)/i.test(k))) {
  throw new Error('ambient variable survived allowlist');
}
const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const prestartScript = path.join(scriptDir, 'prestart_full_tree.ps1');
const processScript = path.join(scriptDir, 'process_snapshot.ps1');
const actorScript = path.join(scriptDir, 'actor_call.mjs');
const capturePath = path.join(outputDir, 'a3_prestart_full_tree.json');
const resultPath = path.join(outputDir, 'windows_verify_raw.json');
const pidFile = path.join(tempRoot, `chrome-devtools-mcp-${sessionId}`, 'daemon.pid');
const counters = {chromeLaunches: 0, daemonStarts: 0, nativeCalls: 0,
  rawWebSocketHandshakes: 0, endpointDiscoveryQueries: 0};
const events = [];
let chrome = null, daemon = null, mcpPid = null, endpoint1 = null, endpoint2 = null;
let outcome = 'INCONCLUSIVE', stopReason = null, startedAt = Date.now();
const ledgers = new Map(); const cleanupErrors=[];
const ceilingMs = 45 * 60 * 1000;
const caps = {chromeLaunches: 3, daemonStarts: 1, nativeCalls: 8,
  rawWebSocketHandshakes: 6, endpointDiscoveryQueries: 2};
const save = () => fs.writeFileSync(resultPath, JSON.stringify({
  schemaVersion: 'ccma.r151.windowsVerifyRaw@1.0.0', sessionId,
  generatedAt: new Date().toISOString(), outcome, stopReason, counters,
  activeElapsedMs: Date.now() - startedAt, packageRoot: pkgRoot, chromeExe,
  nodeVersion: process.version, osBuild: os.release(),
  environmentKeys: Object.keys(env).sort(),
  admittedEnvironmentValues: {
    CHROME_DEVTOOLS_MCP_SESSION_ID: sessionId,
    CHROME_DEVTOOLS_MCP_NO_UPDATE_CHECKS: '1',
    CHROME_DEVTOOLS_MCP_NO_USAGE_STATISTICS: '1',
  },
  tempRoot, evidenceRoot, capturePath, endpoints: {endpoint1, endpoint2}, events,
  admissionGate: {exitCode:gate.status, output:gate.stdout},
}, null, 2) + '\n');
function record(kind, data = {}) { events.push({at: new Date().toISOString(), kind, ...data}); save(); }
function cleanupRecord(kind,data={}) { try { record(kind,data); } catch(error) { cleanupErrors.push(String(error)); process.stderr.write(`cleanup evidence failure: ${String(error)}\n`); } }
const custody=createHandleCustody({wait:waitOriginal,record});
function debit(key) {
  if (Date.now() - startedAt > ceilingMs) throw new Error('45 active minute ceiling');
  if (counters[key] >= caps[key]) throw new Error(`${key} ceiling`);
  counters[key]++; save();
}
function ps(args, timeout = 30000) {
  const r = spawnSync(powershellExe, ['-NoProfile','-ExecutionPolicy','Bypass',...args],
    {cwd: projectRoot, env, encoding: 'utf8', timeout, maxBuffer: 4_000_000});
  if (r.error) throw r.error;
  return {status: r.status, stdout: r.stdout, stderr: r.stderr};
}
function snapshot(profiles = [profile1,profile2]) {
  const r = ps(['-File', processScript, '-OwnedProfilesJson', JSON.stringify(profiles)]);
  if (r.status !== 0) throw new Error(`process snapshot failed: ${r.stderr}`);
  const rows = JSON.parse(r.stdout || '[]');
  const all = Array.isArray(rows) ? rows : [rows];
  if (all.some(row => row.name.toLowerCase() === 'chrome.exe' && !row.commandLineKnown))
    throw new Error('Chrome command line unknown in process inventory');
  return all;
}
function descendant(pid, rootPid, byPid) {
  const seen = new Set();
  while (pid && !seen.has(pid)) {
    if (pid === rootPid) return true;
    seen.add(pid); pid = byPid.get(pid)?.parentPid;
  }
  return false;
}
function checkOwnedChrome(rootPid) {
  const rows=snapshot();
  const expectedProfile=chrome?.pid===rootPid ? chrome.r151Profile : null;
  const root=rows.find(r=>r.pid===rootPid);
  if (!expectedProfile || !root?.createdAt || !snapshot([expectedProfile]).some(r=>r.pid===rootPid && r.createdAt===root.createdAt && r.ownedProfileArg)) throw new Error('root expected profile missing');
  let ledger=ledgers.get(rootPid);
  if (!ledger) { ledger=new OwnershipLedger(rootPid,chromeExe); ledgers.set(rootPid,ledger); }
  const owned=ledger.observe(rows);
  record('process_snapshot',{rootPid,ownedChrome:owned});
  return rows;
}
function alive(child, name) {
  if (!child || child.exitCode !== null || child.signalCode !== null) {
    throw new Error(`${name} original handle exited`);
  }
}
async function freePort() {
  const server = net.createServer();
  await new Promise((resolve,reject) => server.once('error',reject).listen(0,'127.0.0.1',resolve));
  const port = server.address().port;
  await new Promise(resolve => server.close(resolve));
  return port;
}
function startProcess(exe, args, name) {
  const child = spawn(exe,args,{cwd:projectRoot,env,windowsHide:true,stdio:['ignore','pipe','pipe']});
  const lines = {stdout:'',stderr:''};
  for (const stream of ['stdout','stderr']) child[stream].on('data',chunk => {
    lines[stream] = (lines[stream] + chunk.toString()).slice(-10000);
  });
  child.on('error', error => record('process_error',{name,error:String(error)}));
  record('process_start',{name,pid:child.pid,exe,args,startedAt:new Date().toISOString()});
  child.r150Lines = lines;
  return child;
}
async function waitExit(child, name, timeoutMs = 12000) {
  if (!child) return null;
  return custody.wait(child,name,timeoutMs);
}
async function closeHandle(child,name) {
  if (!child) return;
  if (custody.hasTimedOut(child)) throw new Error(`${name} original wait already timed out`);
  if (child.exitCode === null && child.signalCode === null) {
    const sent = child.kill('SIGTERM');
    record('original_handle_close',{name,pid:child.pid,signal:'SIGTERM',sent});
  }
  const result=await waitExit(child,name);
  if (result?.timeout) throw new Error(`${name} original wait timeout`);
  return result;
}
async function discoverEndpoint(port, profile, label) {
  const activePort = path.join(profile,'DevToolsActivePort');
  let endpoint = null, method = null;
  for (let i=0;i<50;i++) {
    if (fs.existsSync(activePort)) {
      const lines = fs.readFileSync(activePort,'utf8').trim().split(/\r?\n/);
      if (Number(lines[0]) !== port) throw new Error(`${label} DevToolsActivePort port mismatch`);
      endpoint = `ws://127.0.0.1:${port}${lines[1]}`; method='DevToolsActivePort'; break;
    }
    await new Promise(resolve => setTimeout(resolve,100));
  }
  if (!endpoint) {
    debit('endpointDiscoveryQueries');
    const url = `http://127.0.0.1:${port}/json/version`;
    const response = await fetch(url,{signal:AbortSignal.timeout(5000)});
    const data = await response.json();
    endpoint = data.webSocketDebuggerUrl; method='/json/version';
    record('endpoint_query',{label,url,httpStatus:response.status,endpoint});
  }
  const parsed = new URL(endpoint);
  if (parsed.protocol !== 'ws:' || parsed.hostname !== '127.0.0.1' ||
      Number(parsed.port) !== port || !/^\/devtools\/browser\/[0-9a-fA-F-]+$/.test(parsed.pathname)) {
    throw new Error(`${label} endpoint shape mismatch`);
  }
  record('endpoint_discovery',{label,method,endpoint});
  return endpoint;
}
async function wsProbe(url, control, label) {
  debit('rawWebSocketHandshakes');
  const result = await new Promise(resolve => {
    let done=false, opened=false, socketError=null;
    const ws = new WebSocket(url);
    const timer=setTimeout(() => finish({opened,timeout:true,error:socketError}),10000);
    function finish(data) {
      if (done) return;
      done=true;clearTimeout(timer);
      try {ws.close();} catch {}
      resolve(data);
    }
    ws.addEventListener('open',() => {
      opened=true;
      if (control) ws.send(JSON.stringify({id:1,method:'Browser.getVersion'}));
      else finish({opened:true});
    });
    ws.addEventListener('message',e => {
      if (!control) return;
      let reply; try {reply=JSON.parse(String(e.data));} catch {reply={raw:String(e.data)};}
      if (reply.id === 1) finish({opened:true,browserVersionReply:reply});
    });
    ws.addEventListener('error',e => {socketError=e.message || String(e);});
    ws.addEventListener('close',e => finish({opened,closeCode:e.code,closeReason:e.reason,error:socketError}));
  });
  record('websocket_handshake',{label,url,...result});
  return result;
}
async function actor(action,pageId) {
  alive(chrome,'Chrome'); alive(daemon,'daemon'); checkOwnedChrome(chrome.pid);
  debit('nativeCalls');
  const args=[actorScript,pkgRoot,sessionId,action];
  if (pageId !== undefined) args.push(String(pageId));
  const child=startProcess(process.execPath,args,`actor:${action}`);
  const wait=await waitExit(child,`actor:${action}`,35000);
  if (wait.timeout) {child.kill('SIGTERM');throw new Error(`${action} actor timeout`);}
  let envelope;
  try {envelope=JSON.parse(child.r150Lines.stdout);} catch {throw new Error(`${action} invalid actor output`);}
  record('native_sendCommand',{action,actorPid:child.pid,envelope});
  return envelope;
}
function inner(envelope) {
  if (envelope.terminal?.kind !== 'RESOLVED' || !envelope.response?.success) {
    throw new Error(`native call failed: ${envelope.actor}`);
  }
  return JSON.parse(envelope.response.result);
}
function pages(result) {
  const list=result?.structuredContent?.pages;
  if (!Array.isArray(list)) throw new Error('native pages inventory missing');
  return list.map(p => ({id:p.id,url:p.url,title:p.title,selected:p.selected}));
}
function sameInventory(a,b) { return JSON.stringify(a) === JSON.stringify(b); }
try {
  save();
  const versionScript = `$c=(Get-Item -LiteralPath '${fixed.chrome}').VersionInfo.FileVersion; $a=(Get-Item -LiteralPath '${fixed.antigravity}').VersionInfo.FileVersion; [pscustomobject]@{chrome=$c;antigravity=$a} | ConvertTo-Json -Compress`;
  const versions = ps(['-Command',versionScript]);
  if (versions.status !== 0) throw new Error('version read failed');
  record('versions',{...JSON.parse(versions.stdout),node:process.version,osBuild:os.release()});
  const port=await freePort();
  record('preflight',{port,pidFile,profiles:[profile1,profile2],envKeys:Object.keys(env).sort()});
  if (fs.existsSync(pidFile) || snapshot().some(r => r.ownedProfileArg)) {
    throw new Error('session PID/profile collision before start');
  }
  // This is the immediate last check before the first process start.
  const capture=ps(['-File',prestartScript,'-PackageRoot',pkgRoot,
    '-ManifestPath',manifestPath,'-OutputPath',capturePath],120000);
  record('a3_prestart',{status:capture.status,stdout:capture.stdout,stderr:capture.stderr});
  if (capture.status !== 0) { outcome='NO_START';stopReason='A3 full-tree recapture failed';save(); }
  else {
    startedAt=Date.now();
    const chromeArgs=(profile) => [`--remote-debugging-port=${port}`,
      `--user-data-dir=${profile}`,'--headless=new','--no-first-run',
      '--no-default-browser-check','about:blank'];
    debit('chromeLaunches');chrome=startProcess(chromeExe,chromeArgs(profile1),'Chrome #1'); chrome.r151Profile=profile1;
    endpoint1=await discoverEndpoint(port,profile1,'Chrome #1');
    alive(chrome,'Chrome #1');checkOwnedChrome(chrome.pid);
    const control1=await wsProbe(endpoint1,true,'A1-a control');
    if (!control1.opened || !control1.browserVersionReply?.result) throw new Error('A1-a control failed');
    const wrong=new URL(endpoint1);
    const guid=wrong.pathname.split('/').pop();
    const first=guid[0].toLowerCase();
    wrong.pathname=wrong.pathname.slice(0,-guid.length)+(first==='a'?'b':'a')+guid.slice(1);
    alive(chrome,'Chrome #1 before A1-b');checkOwnedChrome(chrome.pid);
    const negative1=await negativeProbe(wrong.href,()=>debit('rawWebSocketHandshakes')); record('websocket_handshake',{label:'A1-b wrong GUID',...negative1});
    alive(chrome,'Chrome #1 after A1-b');checkOwnedChrome(chrome.pid);
    if (negative1.classification==='A1_FAIL') {outcome='A1_FAIL';stopReason='non-matching GUID connected';}
    else if (!['HTTP_4XX_REFUSAL','REMOTE_CLOSE_BEFORE_OPEN'].includes(negative1.classification)) {
      throw new Error('A1-b refusal was not observed');
    }
    if (outcome !== 'A1_FAIL') {
      const post1=await wsProbe(endpoint1,true,'A1-b post-control');
      if (!post1.opened || !post1.browserVersionReply?.result) throw new Error('A1-b post-control failed');
    }
    if (outcome !== 'A1_FAIL') {
      const firstPid=chrome.pid;
      chrome=await replaceOwnedChrome({handle:chrome,close:h=>closeHandle(h,'Chrome #1'),snapshot,ledger:ledgers.get(firstPid),record,launch:()=>{ debit('chromeLaunches'); const next=startProcess(chromeExe,chromeArgs(profile2),'Chrome #2'); next.r151Profile=profile2; return next; }});
      endpoint2=await discoverEndpoint(port,profile2,'Chrome #2');
      if (endpoint2===endpoint1) throw new Error('Chrome #2 GUID did not change');
      alive(chrome,'Chrome #2');checkOwnedChrome(chrome.pid);
      alive(chrome,'Chrome #2 before A1-c');checkOwnedChrome(chrome.pid);
      const negative2=await negativeProbe(endpoint1,()=>debit('rawWebSocketHandshakes')); record('websocket_handshake',{label:'A1-c old GUID',...negative2});
      alive(chrome,'Chrome #2 after A1-c');checkOwnedChrome(chrome.pid);
      if (negative2.classification==='A1_FAIL') {outcome='A1_FAIL';stopReason='old GUID connected after port reuse';}
      else if (!['HTTP_4XX_REFUSAL','REMOTE_CLOSE_BEFORE_OPEN'].includes(negative2.classification)) {
        throw new Error('A1-c refusal was not observed');
      }
      if (outcome !== 'A1_FAIL') {
        const control2=await wsProbe(endpoint2,true,'A1-c new control');
        if (!control2.opened || !control2.browserVersionReply?.result) throw new Error('A1-c control failed');
        const daemonArgs=['--viaCli',`--wsEndpoint=${endpoint2}`,'--no-usage-statistics',
          '--no-performance-crux',`--filesystemRoot=${evidenceRoot}`];
        debit('daemonStarts');
        daemon=startProcess(process.execPath,[path.join(pkgRoot,'build/src/daemon/daemon.js'),...daemonArgs],'daemon');
        for (let i=0;i<100 && !fs.existsSync(pidFile);i++) await new Promise(r=>setTimeout(r,100));
        if (!fs.existsSync(pidFile)) throw new Error('daemon PID file missing');
        const processRows=snapshot();
        mcpPid=processRows.find(r => r.name.toLowerCase()==='node.exe' && r.parentPid===daemon.pid)?.pid ?? null;
        record('daemon_process_identity',{pid:daemon.pid,mcpPid,pidFile,processRows:processRows.filter(r=>r.pid===daemon.pid||r.pid===mcpPid)});
        const status=await actor('status');
        const statusValue=JSON.parse(status.response.result);
        if (statusValue.pid!==daemon.pid || statusValue.version!=='1.9.0' ||
            JSON.stringify(statusValue.args)!==JSON.stringify(daemonArgs)) throw new Error('daemon status identity/argv mismatch');
        if (mcpPid===null) {
          mcpPid=snapshot().find(r=>r.name.toLowerCase()==='node.exe'&&r.parentPid===daemon.pid)?.pid??null;
          record('mcp_child_identity_after_status',{mcpPid});
        }
        if (mcpPid===null) throw new Error('MCP child identity unproven');
        const created=inner(await actor('new_page'));
        if (created.isError) throw new Error('new_page returned isError');
        const createdPages=pages(created);
        const admitted=createdPages.find(p=>p.selected && p.url==='about:blank');
        if (!admitted || !Number.isInteger(admitted.id)) throw new Error('new_page admitted page ID missing');
        const inventory=pages(inner(await actor('list_pages')));
        if (inventory.length<2 || !inventory.some(p=>p.id===admitted.id)) throw new Error('full page inventory missing initial/admitted tab');
        record('pinned_inventory',{inventory,admittedPageId:admitted.id});
        const missingEnvelope=await actor('snapshot_missing');
        const missingResponse=missingEnvelope.response;
        const after=pages(inner(await actor('list_pages')));
        if (missingEnvelope.terminal?.kind !== 'RESOLVED' || !missingResponse) throw new Error('missing pageId raw envelope unavailable');
        if (missingResponse.success || sameInventory(inventory,after)===false) {
          outcome='A2_FAIL';stopReason='missing pageId executed or inventory changed';
        } else if (!/^Input validation error: Invalid arguments for tool take_snapshot:/.test(missingResponse.error || '') || !/pageId/.test(missingResponse.error || '')) {
          throw new Error('missing pageId rejection class unclear');
        } else {
          const valid=inner(await actor('snapshot_valid',admitted.id));
          if (valid.isError) throw new Error('valid pageId control failed');
          outcome='A1_A2_PASS';stopReason=null;
        }
      }
    }
  }
} catch (error) {
  if (outcome!=='A1_FAIL' && outcome!=='A2_FAIL' && outcome!=='NO_START') outcome='INCONCLUSIVE';
  stopReason=stopReason || String(error);
  record('error',{message:String(error),stack:error?.stack});
} finally {
  if (daemon && daemon.exitCode===null && daemon.signalCode===null) {
    try {
      if (counters.nativeCalls<8 && chrome && chrome.exitCode===null && chrome.signalCode===null) {
        const stopped=await actor('stop'); record('daemon_stop_ack',{envelope:stopped});
      }
    } catch (error) {cleanupErrors.push(String(error));cleanupRecord('cleanup_error',{name:'daemon stop',message:String(error)});}
    const waited=await waitExit(daemon,'daemon',12000);
    if (waited?.timeout) { const sent=daemon.kill('SIGTERM'); cleanupErrors.push('daemon original wait timed out'); cleanupRecord('cleanup_error',{name:'daemon wait',message:'original wait timed out',terminationSent:sent}); }
  }
  if (chrome && chrome.exitCode===null && chrome.signalCode===null) { try { await closeHandle(chrome,'last Chrome'); } catch(error) { cleanupErrors.push(String(error));cleanupRecord('cleanup_error',{name:'Chrome wait',message:String(error)}); } }
  try {
    const rows=snapshot();
    const terminalLedgers=[...ledgers.values()].map(l=>l.terminal(rows));
    cleanupRecord('chrome_identity_terminal',{terminalLedgers,waitTimedOut:custody.anyTimeout});
    if (terminalChromeDecision({ledgerResults:terminalLedgers,waitTimedOut:custody.anyTimeout,cleanupErrors})!=='ABSENT') { if(outcome==='A1_A2_PASS') outcome='INCONCLUSIVE'; stopReason=stopReason || 'Chrome terminal ownership unknown'; }
    const leftovers=rows.filter(r=>r.name.toLowerCase()==='chrome.exe'&&r.ownedProfileArg);
    const childLeft=mcpPid!==null && rows.some(r=>r.pid===mcpPid);
    const daemonLeft=daemon && (daemon.exitCode===null && daemon.signalCode===null || rows.some(r=>r.pid===daemon.pid));
    record('cleanup_terminal',{leftoverOwnedChrome: leftovers, mcpChildStillAlive:childLeft,
      daemonStillAlive:!!daemonLeft,pidFileStillExists:fs.existsSync(pidFile),daemonPid:daemon?.pid,chromePid:chrome?.pid});
    if (leftovers.length || childLeft || daemonLeft || (daemon && mcpPid===null) || fs.existsSync(pidFile)) {
      if (outcome==='A1_A2_PASS') outcome='INCONCLUSIVE';
      stopReason=stopReason || 'cleanup incomplete';
    }
  } catch(error) {
    if (outcome==='A1_A2_PASS') outcome='INCONCLUSIVE';
    cleanupErrors.push(String(error)); cleanupRecord('cleanup_error',{name:'terminal snapshot',message:String(error)});
  }
  if (Date.now()-startedAt > ceilingMs && outcome==='A1_A2_PASS') {
    outcome='INCONCLUSIVE';stopReason='45 active minute ceiling exceeded';
  }
  save();
  process.stdout.write(JSON.stringify({outcome,stopReason,counters,resultPath,capturePath})+'\n');
}
