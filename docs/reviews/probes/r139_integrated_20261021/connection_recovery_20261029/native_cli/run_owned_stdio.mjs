// R139 source candidate. Import/help is inert; --run requires Root's committed release.
import {createHash} from 'node:crypto';
import {execFileSync} from 'node:child_process';
import {readFileSync, writeFileSync, mkdirSync, openSync, closeSync} from 'node:fs';
import {dirname, join, resolve} from 'node:path';
import {fileURLToPath, pathToFileURL} from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const root = resolve(here, '../../../../../../..');
const rel = p => p.replaceAll('\\', '/');
const sourceRel = rel(resolve(here, 'run_owned_stdio.mjs').slice(root.length + 1));
const grantRel = 'CVF_SESSION/authority/CCMAI-RUNTIME-139-OFFICIAL-CLI-ADMISSION.json';
const releaseRel = rel(resolve(here, 'runtime_release.json').slice(root.length + 1));
const execution = join(here, 'execution');
const sha = b => createHash('sha256').update(b).digest('hex');
const fail = message => { throw Error(message); };
const git = (...args) => execFileSync('git', ['-C', root, ...args], {encoding: 'utf8', timeout: 10000}).trim();
const rawGit = (...args) => execFileSync('git', ['-C', root, ...args], {timeout: 10000});
const readJson = p => JSON.parse(readFileSync(p, 'utf8'));
const log = (name, value) => writeFileSync(join(execution, name), JSON.stringify(value, null, 2) + '\n', {flag: 'wx'});

function preflight() {
  const grantPath = join(root, grantRel), releasePath = join(root, releaseRel);
  const grantBytes = readFileSync(grantPath), releaseBytes = readFileSync(releasePath);
  const grant = JSON.parse(grantBytes), release = JSON.parse(releaseBytes);
  const expected = ['schemaVersion','approved','admissionSha256','adapterSha256','preparationHead','runId','reviewer','scope'];
  if (Object.keys(release).sort().join('|') !== expected.sort().join('|')) fail('release schema');
  if (release.schemaVersion !== '1.0' || release.approved !== true || release.reviewer !== 'Root' ||
      release.scope !== 'ONE_OWNED_STDIO_CAPABILITY' || release.runId !== grant.runId ||
      grant.kind !== 'OWNER_DELEGATED_OFFICIAL_MCP_OWNED_STDIO_CAPABILITY') fail('release identity');
  if (release.admissionSha256 !== sha(grantBytes) || release.adapterSha256 !== sha(readFileSync(fileURLToPath(import.meta.url)))) fail('release hash');
  const head = git('rev-parse', 'HEAD');
  if (release.preparationHead !== '5eccd326d84884121cfe95ff1ca18134d3b4518c' ||
      git('merge-base', release.preparationHead, head) !== release.preparationHead ||
      git('log','-1','--format=%H','--',releaseRel) !== head) fail('release commit ancestry');
  for (const p of [grantRel, releaseRel, sourceRel]) {
    if (!rawGit('show', `HEAD:${p}`).equals(readFileSync(join(root, p)))) fail(`uncommitted ${p}`);
  }
  for (const [p, digest] of Object.entries(grant.pinsRawSha256)) {
    if (sha(readFileSync(p)) !== digest) fail(`pin mismatch ${p}`);
  }
  if (grant.nodePath !== 'C:/Program Files/nodejs/node.exe' ||
      grant.chromePath !== 'C:/Program Files/Google/Chrome/Application/chrome.exe' ||
      grant.sdkBundlePath !== 'C:/Users/tiennm/AppData/Local/Programs/Antigravity/resources/app.asar.unpacked/node_modules/chrome-devtools-mcp/build/src/third_party/index.js' ||
      grant.limits.sdkConnectCalls !== 1 || grant.limits.browserToolCalls !== 8 ||
      grant.limits.perToolTimeoutMilliseconds !== 30000) fail('grant contract changed');
  return grant;
}

function processMetadata(pid, pass) {
  if (!Number.isSafeInteger(pid) || pid <= 0 || pass > 3) fail('invalid process metadata scope');
  const script = `$ErrorActionPreference='Stop'; $p=${pid}; ` +
    `Get-CimInstance Win32_Process -Filter "ProcessId=$p or ParentProcessId=$p" | ` +
    `Select-Object ProcessId,ParentProcessId,ExecutablePath,CommandLine,CreationDate | ConvertTo-Json -Depth 3 -Compress`;
  const out = execFileSync('powershell', ['-NoProfile','-NonInteractive','-Command',script], {encoding:'utf8',timeout:10000});
  log(`process_pass_${pass}.json`, {at:new Date().toISOString(), pid, raw:out});
  const rows = out.trim() ? JSON.parse(out) : [];
  return Array.isArray(rows) ? rows : [rows];
}

function ownedChrome(rows, grant, serverPid) {
  const chrome = rows.filter(p => p.ParentProcessId === serverPid &&
    String(p.ExecutablePath).toLowerCase() === grant.chromePath.replaceAll('/','\\').toLowerCase());
  if (chrome.length !== 1) fail('owned Chrome child not uniquely observed');
  const cmd = String(chrome[0].CommandLine);
  const m = cmd.match(/--user-data-dir=(?:"([^"]+)"|([^\s]+))/i);
  if (!m || !/--remote-debugging-pipe\b/i.test(cmd) || !/chrome-devtools-mcp/i.test(m[1] || m[2]) ||
      !/\b(?:temp|tmp)\b/i.test(m[1] || m[2])) fail('isolated temporary profile not evidenced');
  return {pid:chrome[0].ProcessId, profile:m[1] || m[2]};
}

function pages(result) {
  const list = result?.structuredContent?.pages;
  if (!Array.isArray(list)) fail('structured page IDs missing');
  return list;
}
function text(result) { return (result?.content || []).filter(x => x.type === 'text').map(x => x.text).join('\n'); }

export async function runOwnedStdio() {
  const started = Date.now();
  const grant = preflight(); // No SDK import, transport or process before all pins/release checks.
  mkdirSync(execution, {recursive:true});
  const claim = join(execution, `claim_${grant.runId}.json`);
  const fd = openSync(claim, 'wx');
  writeFileSync(fd, JSON.stringify({runId:grant.runId, at:new Date().toISOString(), adapter:sha(readFileSync(fileURLToPath(import.meta.url)))}));
  closeSync(fd); // A second --run cannot claim the same runId.
  const events = [], mark = (kind, data) => events.push({at:new Date().toISOString(), kind, data});
  let client, transport, serverPid, chromePid, calls = 0, targetId, connected = false, pass = 0;
  const remaining = () => 300000 - (Date.now() - started);
  try {
    process.env.CHROME_DEVTOOLS_MCP_NO_UPDATE_CHECKS = '1';
    process.env.CHROME_DEVTOOLS_MCP_NO_USAGE_STATISTICS = '1';
    const {Client, StdioClientTransport} = await import(pathToFileURL(grant.sdkBundlePath).href);
    const childEnv = {PATH:process.env.PATH, SystemRoot:process.env.SystemRoot,
      TEMP:process.env.TEMP, TMP:process.env.TMP,
      CHROME_DEVTOOLS_MCP_NO_UPDATE_CHECKS:'1', CHROME_DEVTOOLS_MCP_NO_USAGE_STATISTICS:'1'};
    transport = new StdioClientTransport({
      command:grant.nodePath,
      args:[grant.serverPath,'--isolated','--headless',`--executablePath=${grant.chromePath}`,
        '--no-usage-statistics','--no-performance-crux',`--filesystemRoot=${here}`],
      env:childEnv, cwd:root, stderr:'pipe'
    });
    transport.stderr?.on('data', b => mark('server_stderr', b.toString('utf8')));
    client = new Client({name:'r139-owned-stdio',version:'1.0.0'},{capabilities:{}});
    mark('connect_request', {runId:grant.runId});
    await client.connect(transport, {timeout:30000}); // Exactly one connect; no restart path.
    connected = true; serverPid = transport.pid;
    if (!Number.isSafeInteger(serverPid) || serverPid <= 0) fail('transport PID unavailable');
    mark('connect_result', {serverPid});
    const call = async (name, args={}) => {
      if (!connected || !transport.pid || calls >= 8 || remaining() <= 60000) fail('method/deadline bound');
      calls += 1; const n = calls;
      mark('tool_request', {n,name,args});
      try {
        const result = await client.callTool({name,arguments:args}, undefined, {timeout:Math.min(30000,remaining()-60000)});
        mark('tool_result', {n,name,result});
        if (result?.isError) fail(`tool ${name} returned isError`);
        return result;
      } catch (error) { mark('tool_failure', {n,name,error:String(error)}); throw error; }
    };
    const initial = pages(await call('list_pages'));
    if (initial.length !== 1 || initial[0].url !== 'about:blank') fail('owned startup blank not observed');
    const observed = processMetadata(serverPid, ++pass);
    const owner = ownedChrome(observed, grant, serverPid); chromePid = owner.pid;
    mark('ownership', owner);
    const created = pages(await call('new_page',{url:'https://example.com/',timeout:30000}));
    const targets = created.filter(p => p.id !== initial[0].id && new URL(p.url).origin === 'https://example.com');
    if (targets.length !== 1) fail('unique returned Example Domain target missing');
    targetId = targets[0].id; mark('target', {targetId,url:targets[0].url});
    const snap = await call('take_snapshot',{pageId:targetId});
    if (!/Example Domain/i.test(text(snap)) || !/example\.com/i.test(targets[0].url)) fail('Example Domain snapshot missing');
    const shot = join(execution,'example.png');
    await call('take_screenshot',{pageId:targetId,format:'png',filePath:shot});
    const bytes = readFileSync(shot);
    if (bytes.length < 8 || !bytes.subarray(0,8).equals(Buffer.from([137,80,78,71,13,10,26,10]))) fail('PNG original absent');
    mark('capture', {path:shot,bytes:bytes.length,sha256:sha(bytes)});
    await call('close_page',{pageId:targetId});
    const finalPages = pages(await call('list_pages'));
    if (finalPages.some(p => p.id === targetId) || !finalPages.some(p => p.id === initial[0].id)) fail('target close not observed');
    mark('target_closed', {targetId});
  } catch (error) { mark('stop', {error:String(error)}); }
  finally {
    serverPid ||= transport?.pid;
    try { await client?.close(); mark('client_close', 'returned'); } catch (e) { mark('client_close_error', String(e)); }
    try { await transport?.close(); mark('transport_close', 'returned'); } catch (e) { mark('transport_close_error', String(e)); }
    if (serverPid && pass < 3) {
      try { mark('post_close_processes', processMetadata(serverPid, ++pass)); }
      catch (e) { mark('post_close_unknown', String(e)); }
    }
    log('events.json', {runId:grant.runId, serverPid:serverPid??null, chromePid:chromePid??null,
      browserCalls:calls, elapsedMs:Date.now()-started, events});
  }
}

if (process.argv.includes('--run')) {
  if (process.argv.length !== 3) fail('only --run accepted');
  await runOwnedStdio();
} else if (process.argv.length > 2 && !process.argv.includes('--help')) fail('unsupported argument');
else if (process.argv.includes('--help')) console.log('R139 source adapter; --run requires committed Root runtime release.');
