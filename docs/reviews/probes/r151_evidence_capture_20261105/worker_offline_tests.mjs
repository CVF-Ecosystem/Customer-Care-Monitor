import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {EventEmitter} from 'node:events';
import {createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';
import fs from 'node:fs';
import {OwnershipLedger, waitOriginal, createUpgradeCapture, negativeProbe, createHandleCustody, replaceOwnedChrome, terminalChromeDecision, validateSourceManifest} from './diagnostic_helpers.mjs';

let count=0; function check(name,fn){fn(); count++; console.log(`PASS ${name}`);}
const ps=process.env.SystemRoot+'\\System32\\WindowsPowerShell\\v1.0\\powershell.exe';
const script=fileURLToPath(new URL('./process_snapshot.ps1',import.meta.url));
const profile='C:\\Temp\\profile one';
const row=(line)=>[{ProcessId:1,ParentProcessId:0,Name:'chrome.exe',CreationDate:'2026-01-01T00:00:00Z',ExecutablePath:'C:\\Chrome\\chrome.exe',CommandLine:line}];
function snapshot(profiles,line='chrome.exe --user-data-dir="C:\\Temp\\profile one"') {
  return spawnSync(ps,['-NoProfile','-ExecutionPolicy','Bypass','-File',script,'-OwnedProfilesJson',profiles,'-FixtureRowsJson',JSON.stringify(row(line))],{encoding:'utf8'});
}
for (const [name,raw,expected] of [['empty','[]',false],['one',JSON.stringify([profile]),true],['two',JSON.stringify([profile,'C:\\Temp\\other']),true]]) check('PS '+name,()=>{const r=snapshot(raw); assert.equal(r.status,0,r.stderr);assert.equal(JSON.parse(r.stdout)[0].ownedProfileArg,expected);});
for (const [name,raw] of [['malformed','['],['null','null'],['empty value','[""]'],['non string','[3]'],['duplicates',JSON.stringify([profile,profile.toUpperCase()])]]) check('PS invalid '+name,()=>assert.notEqual(snapshot(raw).status,0));
for (const [name,line,expected] of [['prefix','chrome.exe --user-data-dir="C:\\Temp\\profile one-more"',false],['unrelated','chrome.exe --other="C:\\Temp\\profile one"',false],['quoted uppercase','chrome.exe --user-data-dir="c:\\TEMP\\PROFILE ONE"',true],['separate','chrome.exe --user-data-dir "C:\\Temp\\profile one"',true],['unknown','',false]]) check('PS argv '+name,()=>{const r=snapshot(JSON.stringify([profile]),line);assert.equal(r.status,0,r.stderr);assert.equal(JSON.parse(r.stdout)[0].ownedProfileArg,expected);});
const root={pid:1,parentPid:0,name:'chrome.exe',createdAt:'t1',executablePath:'C:\\Chrome\\chrome.exe',commandLineKnown:true,ownedProfileArg:true};
const child={...root,pid:2,parentPid:1,createdAt:'t2',ownedProfileArg:false};
function ledger(){return new OwnershipLedger(1,'C:/Chrome/chrome.exe');}
check('missing root',()=>assert.throws(()=>ledger().observe([])));
check('false root',()=>assert.throws(()=>ledger().observe([{...root,ownedProfileArg:false}])));
check('unknown inventory',()=>assert.throws(()=>ledger().observe(null)));
check('foreign profile',()=>assert.throws(()=>ledger().observe([root,{...child,parentPid:99,ownedProfileArg:true}])));
check('unknown command',()=>assert.throws(()=>ledger().observe([{...root,commandLineKnown:false}])));
check('descendant and orphan retained',()=>{const l=ledger();l.observe([root,child]);l.observe([root,{...child,parentPid:99}]);assert.equal(l.known.size,2);assert.equal(l.terminal([child]).identities.find(x=>x.pid===2).state,'PRESENT');});
check('terminal absent and pid reuse',()=>{const l=ledger();l.observe([root,child]);assert.equal(l.terminal([]).verdict,'ABSENT');assert.equal(l.terminal([{...child,createdAt:'new'}]).identities.find(x=>x.pid===2).state,'PID_REUSED');});
check('terminal inventory unknown',()=>{const l=ledger();l.observe([root]);assert.equal(l.terminal(null).verdict,'UNKNOWN');});
check('unknown terminal creation time',()=>{const l=ledger();l.observe([root]);assert.equal(l.terminal([{...root,createdAt:null}]).verdict,'UNKNOWN');});
check('descendant via intermediate process',()=>{const l=ledger();const intermediate={pid:7,parentPid:1,name:'utility.exe',createdAt:'t7'};l.observe([root,intermediate,{...child,parentPid:7}]);assert.equal(l.known.size,2);});
check('root pid reuse blocks',()=>{const l=ledger();l.observe([root]);assert.throws(()=>l.observe([{...root,createdAt:'replacement'}]));});
const childHandle=()=>Object.assign(new EventEmitter(),{pid:3,exitCode:null,signalCode:null});
{ const c=childHandle();const p=waitOriginal(c,1000); c.emit('exit',0,null); assert.equal((await p).exitCode,0);count++;console.log('PASS original exit'); }
{ const c=childHandle();const p=waitOriginal(c,1);assert.equal((await p).timeout,true);count++;console.log('PASS original timeout'); }
const key='dGhlIHNhbXBsZSBub25jZQ=='; const accept=createHash('sha1').update(key+'258EAFA5-E914-47DA-95CA-C5AB0DC85B11').digest('base64');
function capture(chunks, event='data') { let result,teardowns=0;const c=createUpgradeCapture({key,maxHeader:256,settle:r=>result=r,teardown:()=>teardowns++});for(const chunk of chunks)c.data(chunk);if(event!=='data')c[event]('reset');c.timeout();return {result,teardowns}; }
for(const status of [403,404]) check('fragmented '+status,()=>{const x=capture(['HTTP/1.1 '+status+' Denied\r\nCon','tent-Length: 0\r\n\r\n']);assert.equal(x.result.classification,'HTTP_4XX_REFUSAL');assert.equal(x.teardowns,1);});
check('valid 101',()=>assert.equal(capture([`HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: ${accept}\r\n\r\n`]).result.classification,'A1_FAIL'));
check('invalid 101',()=>assert.equal(capture(['HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\n\r\n']).result.classification,'INCONCLUSIVE'));
check('remote eof',()=>assert.equal(capture([],'end').result.classification,'REMOTE_CLOSE_BEFORE_OPEN'));
check('partial eof',()=>assert.equal(capture(['HTTP/1.1'],'end').result.classification,'INCONCLUSIVE'));
check('reset',()=>assert.equal(capture([],'error').result.classification,'INCONCLUSIVE'));
check('timeout',()=>assert.equal(capture([],'timeout').result.classification,'INCONCLUSIVE'));
check('malformed',()=>assert.equal(capture(['bogus\r\n\r\n']).result.classification,'INCONCLUSIVE'));
check('oversized',()=>assert.equal(capture(['x'.repeat(257)]).result.classification,'INCONCLUSIVE'));
check('duplicate and late',()=>{const x=capture(['HTTP/1.1 403 Denied\r\nA: 1\r\nA: 2\r\n\r\n','HTTP/1.1 403 Denied\r\n\r\n']);assert.equal(x.result.classification,'INCONCLUSIVE');assert.equal(x.teardowns,1);});
check('positive and A2 controls retained',()=>{const source=fs.readFileSync(new URL('./owner_runner.mjs',import.meta.url),'utf8');for(const text of ["Browser.getVersion","A1-c new control","snapshot_missing","snapshot_valid","sameInventory(inventory,after)","a3_prestart"]) assert.ok(source.includes(text),text);});
check('actor source is historical preimage',()=>assert.deepEqual(fs.readFileSync(new URL('./actor_call.mjs',import.meta.url)),fs.readFileSync(new URL('../r150_a1_a2_windows_20261105/actor_call.mjs',import.meta.url))));
{
 let waits=0, launches=0; const handle={}; const events=[];
 const custody=createHandleCustody({wait:async()=>{waits++;return {timeout:true};},record:(kind,data)=>events.push({kind,data})});
 const close=async h=>{const r=await custody.wait(h,'original',1);if(r.timeout)throw Error('timeout');};
 await assert.rejects(replaceOwnedChrome({handle,close,snapshot:()=>[],ledger:{terminal:()=>({verdict:'ABSENT'})},launch:()=>{launches++;},record:()=>{}}));
 await assert.rejects(custody.wait(handle,'original',1));
 assert.equal(waits,1);assert.equal(launches,0);assert.equal(custody.anyTimeout,true);assert.equal(events[0].data.timeout,true);
 assert.equal(terminalChromeDecision({ledgerResults:[{verdict:'ABSENT'}],waitTimedOut:custody.anyTimeout}),'UNKNOWN');count++;console.log('PASS actual replacement seam timeout');
}
{
 let launches=0; const h={};const c=createHandleCustody({wait:async()=>({exitCode:0}),record:()=>{}});
 await replaceOwnedChrome({handle:h,close:async x=>c.wait(x,'original',1),snapshot:()=>[],ledger:{terminal:()=>({verdict:'ABSENT'})},launch:()=>{launches++;return {};},record:()=>{}});
 await assert.rejects(c.wait(h,'original',1));assert.equal(launches,1);assert.equal(terminalChromeDecision({ledgerResults:[{verdict:'UNKNOWN'}],waitTimedOut:false}),'UNKNOWN');count++;console.log('PASS replacement exit and unknown terminal');
}
{
 const h=childHandle();let fire;const p=waitOriginal(h,1,fn=>(fire=fn,1),()=>{});fire();const first=await p;h.emit('exit',0,null);assert.equal(first.timeout,true);count++;console.log('PASS late exit stays timed out');
}
{
 const h={};let waits=0;const c=createHandleCustody({wait:async()=>{waits++;return {timeout:true};},record:()=>{throw Error('record failed');}});
 await assert.rejects(c.wait(h,'original',1));await assert.rejects(c.wait(h,'original',1));assert.equal(waits,1);assert.equal(c.anyTimeout,true);
 assert.equal(terminalChromeDecision({ledgerResults:[{verdict:'ABSENT'}],waitTimedOut:false,cleanupErrors:['record failed']}),'UNKNOWN');count++;console.log('PASS cleanup record failure retained');
}
class FakeSocket extends EventEmitter { write(s){this.request=s;} destroy(){this.destroyed=true;this.emit('close',false);} }
async function transport(event) {
 const socket=new FakeSocket();let debit=0, connection;
 const p=negativeProbe('ws://127.0.0.1:9222/devtools/browser/abc-def',()=>debit++,100,{connect:o=>(connection=o,socket),randomBytes:n=>Buffer.alloc(n,1),setTimer:()=>1,clearTimer:()=>{}});
 socket.emit('connect');
 if(event==='response') {socket.emit('data',Buffer.from('HTTP/1.1 403 Denied\r\nContent-Length: 0\r\n\r\n'));}
 else if(event==='end')socket.emit('end'); else if(event==='reset')socket.emit('error',Error('reset')); else socket.emit('close',false);
 return {r:await p,socket,debit,connection};
}
for(const [event,expected] of [['response','HTTP_4XX_REFUSAL'],['end','REMOTE_CLOSE_BEFORE_OPEN'],['reset','INCONCLUSIVE'],['local','INCONCLUSIVE']]) {
 const {r,socket,debit,connection}=await transport(event);assert.equal(r.classification,expected);assert.equal(debit,1);assert.deepEqual(connection,{host:'127.0.0.1',port:9222});assert.equal(r.url,'ws://127.0.0.1:9222/devtools/browser/abc-def');assert.ok(socket.request.includes(`Sec-WebSocket-Key: ${r.requestKey}\r\n`));assert.ok(socket.request.includes('GET /devtools/browser/abc-def HTTP/1.1\r\n'));assert.ok(socket.request.includes('Upgrade: websocket\r\nConnection: Upgrade\r\n'));assert.equal(socket.destroyed,true);count++;console.log('PASS injected transport '+event);
}
check('reviewed manifest binding',()=>{
 const bytes=Buffer.from('source');const hash=b=>createHash('sha256').update(b).digest('hex');const manifest=Buffer.from(JSON.stringify({files:{'a':hash(bytes)}}));const pin=hash(manifest);const args={manifestBytes:manifest,expectedManifestSha256:pin,reviewText:`review ${pin}`,requiredPaths:['a'],readFile:()=>bytes};validateSourceManifest(args);assert.throws(()=>validateSourceManifest({...args,readFile:()=>Buffer.from('altered')}));assert.throws(()=>validateSourceManifest({...args,reviewText:'unreviewed'}));assert.throws(()=>validateSourceManifest({...args,requiredPaths:['a','b']}));
});
check('exact candidate manifest',()=>{const m=JSON.parse(fs.readFileSync(new URL('./worker_exact_source_manifest.json',import.meta.url),'utf8'));for(const [name,hash] of Object.entries(m.files)){const bytes=fs.readFileSync(new URL('./'+name,import.meta.url));assert.equal(createHash('sha256').update(bytes).digest('hex'),hash,name);}});
console.log(`TOTAL ${count} PASS; SKIPS 0`);
