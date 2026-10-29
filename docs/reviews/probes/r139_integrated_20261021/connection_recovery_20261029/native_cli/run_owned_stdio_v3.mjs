// R139: source-reviewed, single-use native SDK adapter. No SDK is imported on module load.
import {createHash} from 'node:crypto';
import {execFileSync} from 'node:child_process';
import {readFileSync, writeFileSync, mkdirSync, openSync, closeSync, lstatSync, realpathSync, existsSync} from 'node:fs';
import {dirname, join, resolve, relative, isAbsolute, basename} from 'node:path';
import {fileURLToPath, pathToFileURL} from 'node:url';
import {tmpdir} from 'node:os';

const here = dirname(fileURLToPath(import.meta.url));
const root = resolve(here, '../../../../../..');
const source = 'docs/reviews/probes/r139_integrated_20261021/connection_recovery_20261029/native_cli/run_owned_stdio_v3.mjs';
const admission = 'CVF_SESSION/authority/CCMAI-RUNTIME-139-NATIVE-SDK-V3-ADMISSION.json';
const release = 'docs/reviews/probes/r139_integrated_20261021/connection_recovery_20261029/native_cli/runtime_release_v3.json';
const execution = join(here, 'execution_v3');
const preparation = '55f52e064a30e0e6314836b303ba8afffce36f35';
const runId = '3e2ccdd0-b292-4c0f-8f27-9b123e9fabed';
const digest = b => createHash('sha256').update(b).digest('hex');
const fail = s => { throw new Error(s); };
const git = (...a) => execFileSync('git', ['-C',root,...a], {timeout:10000});
const gitText = (...a) => git(...a).toString('utf8').trim();
const bytes = p => readFileSync(join(root,p));
const committed = p => { if (!git('show',`HEAD:${p}`).equals(bytes(p))) fail(`uncommitted ${p}`); };
const exactKeys = (o, names) => o && typeof o === 'object' && !Array.isArray(o) && Object.keys(o).sort().join('|') === [...names].sort().join('|');

function inspect() {
  if (resolve(root,source) !== fileURLToPath(import.meta.url)) fail('project root/source mismatch');
  const raw = bytes(admission), g = JSON.parse(raw);
  if (g.kind !== 'OWNER_DELEGATED_R139_NATIVE_SDK_FINITE_VERIFICATION' || g.runId !== runId ||
      g.sourceAuthorityPath !== admission || g.sourceAdapter !== source || g.effectRelease !== release ||
      g.riskCeiling !== 'R2' || g.status !== 'SOURCE_REPAIR_ONLY_RUNTIME_HELD') fail('admission identity');
  if (Object.keys(g.pinsRawSha256 || {}).length !== 20) fail('pin count');
  for (const [path, sha] of Object.entries(g.pinsRawSha256)) {
    if (!/^[0-9a-f]{64}$/.test(sha) || digest(readFileSync(path)) !== sha) fail(`pin mismatch ${path}`);
  }
  if (g.nodePath !== 'C:/Program Files/nodejs/node.exe' || g.chromePath !== 'C:/Program Files/Google/Chrome/Application/chrome.exe' ||
      g.serverPath !== 'C:/Users/tiennm/AppData/Local/Programs/Antigravity/resources/app.asar.unpacked/node_modules/chrome-devtools-mcp/build/src/bin/chrome-devtools-mcp.js' ||
      g.sdkBundlePath !== 'C:/Users/tiennm/AppData/Local/Programs/Antigravity/resources/app.asar.unpacked/node_modules/chrome-devtools-mcp/build/src/third_party/index.js' ||
      !g.pinsRawSha256[g.nodePath] || !g.pinsRawSha256[g.chromePath] || !g.pinsRawSha256[g.serverPath] || !g.pinsRawSha256[g.sdkBundlePath]) fail('executable identity');
  committed(admission);
  return {g, raw};
}
export function validateReleaseEnvelope(r, admissionBytes, sourceBytes) {
  if (!exactKeys(r,['schemaVersion','approved','admissionSha256','adapterSha256','preparationHead','runId','reviewer','scope']) ||
      r.schemaVersion !== '1.0' || r.approved !== true || r.reviewer !== 'Root' || r.scope !== 'ONE_OWNED_STDIO_CAPABILITY' ||
      r.runId !== runId || r.preparationHead !== preparation || r.admissionSha256 !== digest(admissionBytes) || r.adapterSha256 !== digest(sourceBytes)) fail('release schema or binding');
}
function authorize() {
  const {g,raw} = inspect();
  const r = JSON.parse(bytes(release));
  validateReleaseEnvelope(r,raw,bytes(source));
  const head = gitText('rev-parse','HEAD');
  if (gitText('merge-base', preparation, head) !== preparation || gitText('log','-1','--format=%H','--',release) !== head) fail('release ancestry');
  for (const p of [admission,source,release]) committed(p);
  return g;
}

// One scoped observation per invocation, with explicit native modules and no user profile/autoload.
function metadata(pids, parent, timeout = 10000) {
  if (!pids.every(p => Number.isSafeInteger(p) && p > 0) || !Number.isSafeInteger(parent) || parent <= 0) fail('bad PID scope');
  const ids = [...new Set(pids)].join(',');
  const script = `$ErrorActionPreference='Stop'; Import-Module (Join-Path $PSHOME 'Modules/Microsoft.PowerShell.Utility/Microsoft.PowerShell.Utility.psd1'); Import-Module (Join-Path $PSHOME 'Modules/CimCmdlets/CimCmdlets.psd1'); $ids=@(${ids}); $parent=${parent}; $f=($ids | ForEach-Object {"ProcessId=$_"}) -join ' OR '; $f += " OR ParentProcessId=$parent"; $rows=@(Get-CimInstance Win32_Process -Filter $f | Select-Object ProcessId,ParentProcessId,ExecutablePath,CommandLine,@{Name='CreationDateUtc';Expression={$_.CreationDate.ToUniversalTime().ToString('o')}}); ConvertTo-Json -InputObject $rows -Depth 4 -Compress`;
  const shell = join(process.env.SystemRoot || 'C:/Windows','System32/WindowsPowerShell/v1.0/powershell.exe');
  const raw = execFileSync(shell,['-NoProfile','-NonInteractive','-Command',script],{encoding:'utf8',timeout,windowsHide:true});
  const result = JSON.parse(raw); return {raw, rows:Array.isArray(result) ? result : [result]};
}
function owner(rows, pid, chromePath, nodePath, serverPath, connectStarted) {
  const server = rows.find(x => x.ProcessId === pid);
  const matches = rows.filter(x => x.ParentProcessId === pid && String(x.ExecutablePath).toLowerCase() === chromePath.replaceAll('/','\\').toLowerCase());
  if (!server?.CreationDateUtc || server.ParentProcessId !== process.pid || String(server.ExecutablePath).toLowerCase() !== nodePath.replaceAll('/','\\').toLowerCase() ||
      !String(server.CommandLine).toLowerCase().replaceAll('\\','/').includes(serverPath.toLowerCase()) || matches.length !== 1 || !matches[0].CreationDateUtc) fail('server/Chrome identity unknown');
  const chrome = matches[0], command = String(chrome.CommandLine || '');
  const serverTime=Date.parse(server.CreationDateUtc), chromeTime=Date.parse(chrome.CreationDateUtc), now=Date.now();
  if (!Number.isFinite(serverTime) || !Number.isFinite(chromeTime) || serverTime < connectStarted-2000 || serverTime > now || chromeTime < serverTime || chromeTime > now) fail('creation chronology unknown');
  const m = command.match(/--user-data-dir=(?:"([^"]+)"|([^\s]+))/i);
  if (!m || !/--remote-debugging-pipe(?:\s|$)/i.test(command)) fail('profile/pipe missing');
  const profile = resolve(m[1] || m[2]), temp = realpathSync(tmpdir()), real = realpathSync(profile);
  const rel = relative(temp,real);
  if (!rel || rel.startsWith('..') || isAbsolute(rel) || !basename(real).startsWith('puppeteer_dev_chrome_profile-') || resolve(profile).toLowerCase() !== real.toLowerCase()) fail('profile isolation');
  let path = real;
  while (path.toLowerCase() !== temp.toLowerCase()) { if (lstatSync(path).isSymbolicLink()) fail('reparse profile'); path = dirname(path); }
  return {serverPid:pid,serverCreationUtc:new Date(serverTime).toISOString(),chromePid:chrome.ProcessId,chromeCreationUtc:new Date(chromeTime).toISOString(),profile:real};
}
export function parseNativePages(r) {
  if (!r || typeof r !== 'object' || r.isError || r.structuredContent !== undefined || !Array.isArray(r.content) || r.content.length !== 1 ||
      r.content[0]?.type !== 'text' || typeof r.content[0].text !== 'string') fail('page response invalid');
  const lines=r.content[0].text.split('\n');
  if (lines[0] !== '## Pages' || lines.length < 2 || lines.length > 3) fail('page section invalid');
  const ids=new Set(); let selected=0;
  const result=lines.slice(1).map(line=>{
    const m=line.match(/^([1-9][0-9]*): (about:blank|https:\/\/example\.com\/|[^\r\n()]+ \((?:about:blank|https:\/\/example\.com\/)\))( \[selected\])?$/);
    if (!m) fail('page row invalid');
    const id=Number(m[1]); if (!Number.isSafeInteger(id) || ids.has(id)) fail('page ID invalid'); ids.add(id);
    const label=m[2], url=label.endsWith(')') ? label.slice(label.lastIndexOf(' (')+2,-1) : label;
    if (m[3]) selected++;
    return {id,url,selected:!!m[3]};
  });
  if (selected !== 1 || new Set(result.map(x=>x.url)).size !== result.length) fail('page selection or URL invalid');
  return result;
}

const timeout = (promise, ms, name) => new Promise((yes,no) => { const timer=setTimeout(() => no(new Error(`${name} timeout`)),ms); promise.then(v=>{clearTimeout(timer);yes(v);},e=>{clearTimeout(timer);no(e);}); });
// Cleanup must continue even when the evidence destination fails.
export function safeEvidenceWrite(writer, state, kind, data) {
  try { writer(kind,data); return true; }
  catch (e) {
    state.failed=true;
    try { process.stderr.write(`evidence write failed (${kind}): ${String(e)}\n`); } catch { /* best effort only */ }
    return false;
  }
}

export async function runOwnedStdio() {
  const start = Date.now(), g = authorize();
  mkdirSync(execution,{recursive:true});
  const claim = openSync(join(execution,`claim_${runId}.json`),'wx');
  writeFileSync(claim,JSON.stringify({runId,at:new Date().toISOString(),adapterSha256:digest(bytes(source))})); closeSync(claim);
  let seq=0, client, transport, pid, chromePid, ownerIdentity, connectStarted, calls=0, passes=0, connectCount=0, alive=true, success=false;
  const record=(kind,data) => writeFileSync(join(execution,`${String(++seq).padStart(3,'0')}_${kind}.json`),JSON.stringify({at:new Date().toISOString(),data},null,2)+'\n',{flag:'wx'});
  const evidence={failed:false};
  const safeRecord=(kind,data)=>safeEvidenceWrite(record,evidence,kind,data);
  const remaining=()=>300000-(Date.now()-start);
  const watch=(kind,value)=>{if (!safeRecord(kind,value)) alive=false;};
  const warning=w=>watch('warning',String(w)); process.on('warning',warning);
  try {
    process.env.CHROME_DEVTOOLS_MCP_NO_UPDATE_CHECKS='1';
    process.env.CHROME_DEVTOOLS_MCP_NO_USAGE_STATISTICS='1';
    const {Client,StdioClientTransport} = await import(pathToFileURL(g.sdkBundlePath).href);
    const env={PATH:process.env.PATH,SystemRoot:process.env.SystemRoot,TEMP:process.env.TEMP,TMP:process.env.TMP,CHROME_DEVTOOLS_MCP_NO_UPDATE_CHECKS:'1',CHROME_DEVTOOLS_MCP_NO_USAGE_STATISTICS:'1'};
    transport=new StdioClientTransport({command:g.nodePath,args:[g.serverPath,'--isolated','--headless',`--executablePath=${g.chromePath}`,'--no-usage-statistics','--no-performance-crux',`--filesystemRoot=${here}`],env,cwd:root,stderr:'pipe'});
    transport.stderr?.on('data', b=>watch('stderr',b.toString('utf8')));
    client=new Client({name:'r139-owned-stdio-v3',version:'1.0.0'},{capabilities:{}});
    client.onerror=e=>{alive=false;watch('client_error',String(e));};
    client.onclose=()=>{alive=false;watch('client_closed',{});};
    connectStarted=Date.now(); record('connect_request',{count:++connectCount,connectStarted});
    const connect=client.connect(transport,{timeout:30000});
    const provisionalPid=transport.pid; pid=provisionalPid; record('server_pid_provisional',{pid:pid ?? null});
    await timeout(connect,Math.min(30000,Math.max(1,remaining()-60000)),'connect');
    const connectedPid=transport.pid;
    if (provisionalPid && connectedPid!==provisionalPid) fail('server PID changed');
    pid=connectedPid;
    if (!alive || !Number.isSafeInteger(pid) || pid<=0) fail('connect ownership lost');
    record('connect_result',{pid});
    const call=async(name,args={})=>{
      if (!alive || transport.pid!==pid || calls>=8 || remaining()<=60000) fail('transport or budget');
      const n=++calls; record('tool_request',{n,name,args});
      try { const result=await timeout(client.callTool({name,arguments:args},undefined,{timeout:30000}),Math.min(30000,remaining()-60000),name); record('tool_result',{n,name,result}); if (!alive || result?.isError) fail('tool failed or transport lost'); return result; }
      catch(e) { alive=false;record('tool_failure',{n,name,error:String(e)});throw e; }
    };
    const firstResult=await call('list_pages');
    ++passes; const obs=metadata([pid],pid); record('process_observation',{pass:passes,...obs});
    ownerIdentity=owner(obs.rows,pid,g.chromePath,g.nodePath,g.serverPath,connectStarted);chromePid=ownerIdentity.chromePid;record('ownership',ownerIdentity);
    const first=parseNativePages(firstResult);
    if (first.length!==1 || first[0].url!=='about:blank') fail('startup page mismatch');
    const created=parseNativePages(await call('new_page',{url:'https://example.com/',timeout:30000}));
    const targets=created.filter(p=>p.id!==first[0].id && p.url==='https://example.com/');
    if (created.length!==2 || targets.length!==1 || !created.some(p=>p.id===first[0].id && p.url==='about:blank')) fail('target identity');
    const target=targets[0].id;
    const snapshot=await call('take_snapshot',{pageId:target});
    if (!/Example Domain/i.test(JSON.stringify(snapshot))) fail('snapshot content');
    const image=join(execution,'example.png');
    if (existsSync(image)) fail('capture path already exists');
    await call('take_screenshot',{pageId:target,format:'png',filePath:image});
    const png=readFileSync(image); if (!png.subarray(0,8).equals(Buffer.from([137,80,78,71,13,10,26,10]))) fail('PNG absent');
    record('capture',{path:image,bytes:png.length,sha256:digest(png)});
    await call('close_page',{pageId:target});
    const last=parseNativePages(await call('list_pages'));
    if (last.length!==1 || last[0].id!==first[0].id || last[0].url!=='about:blank') fail('target closure unknown');
    success=true;
  } catch(e) { alive=false;safeRecord('stop',{error:String(e)}); }
  finally {
    alive=false; pid ||= transport?.pid;
    if (!ownerIdentity && pid && connectStarted && passes<2 && remaining()>1000) {
      try { ++passes; const early=metadata([pid],pid,Math.min(10000,remaining()-1)); safeRecord('process_observation',{pass:passes,...early});
        ownerIdentity=owner(early.rows,pid,g.chromePath,g.nodePath,g.serverPath,connectStarted); chromePid=ownerIdentity.chromePid; safeRecord('ownership',ownerIdentity);
      } catch(e) { safeRecord('early_ownership',{status:'UNKNOWN',error:String(e)}); }
    }
    try { if (client) { await timeout(client.close(),Math.max(1,Math.min(20000,remaining())),'client close'); safeRecord('client_close',{result:'returned'}); } }
    catch(e) {success=false;safeRecord('client_close',{error:String(e)});}
    if (transport) {
      try { await timeout(transport.close(),Math.max(1,Math.min(10000,remaining())),'transport close'); safeRecord('transport_close',{result:'returned'}); }
      catch(e) { success=false;safeRecord('transport_close',{error:String(e)}); }
    }
    if (pid && passes<3 && remaining()>1000) {
      try { ++passes; const observation=metadata(chromePid?[pid,chromePid]:[pid],pid,Math.min(10000,remaining()-1)); safeRecord('process_observation',{pass:passes,...observation});
        const present=observation.rows.filter(x=>x.ProcessId===pid || x.ProcessId===chromePid);
        safeRecord('cleanup',{status:chromePid && present.length===0?'KNOWN_PIDS_ABSENT':'UNKNOWN',knownPidRows:present,identity:ownerIdentity ?? null});
        if (!chromePid || present.length) success=false;
      } catch(e) { success=false;safeRecord('cleanup',{status:'UNKNOWN',error:String(e)}); }
    } else {success=false;safeRecord('cleanup',{status:'UNKNOWN'});}
    success = success && !evidence.failed;
    safeRecord('terminal',{success,connectCount,calls,passes,elapsedMs:Date.now()-start,serverPid:pid ?? null,chromePid:chromePid ?? null,evidenceFailure:evidence.failed});
    success = success && !evidence.failed;
    process.removeListener('warning',warning);
  }
  if (!success) throw new Error('bounded lifecycle blocked or cleanup unknown');
}

async function main() {
  const arg=process.argv.slice(2);
  if (arg.length===0 || (arg.length===1 && arg[0]==='--help')) { console.log('R139 SDK v3: --preflight or --run'); return; }
  if (arg.length!==1) fail('unsupported arguments');
  if (arg[0]==='--preflight') { try { authorize(); console.log('READY_FOR_RELEASED_RUN'); } catch(e) { console.log(`HELD: ${e.message}`); process.exitCode=1; } return; }
  if (arg[0]==='--run') { await runOwnedStdio(); return; }
  fail('unsupported argument');
}
if (process.argv[1] && resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
  main().catch(e=>{ console.error(String(e)); process.exitCode=1; });
}
