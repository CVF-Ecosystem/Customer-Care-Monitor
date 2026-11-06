// Offline-testable diagnostic primitives. No effect authority is conveyed here.
import net from 'node:net';
import {randomBytes, createHash} from 'node:crypto';
import path from 'node:path';

export function validateSourceManifest({manifestBytes,expectedManifestSha256,reviewText,requiredPaths,readFile}) {
  const sha=b=>createHash('sha256').update(b).digest('hex');
  if (!/^[a-f0-9]{64}$/.test(expectedManifestSha256) || sha(manifestBytes)!==expectedManifestSha256 || !reviewText.includes(expectedManifestSha256)) throw Error('reviewed source manifest mismatch');
  const m=JSON.parse(manifestBytes.toString('utf8'));
  if (!m.files || Object.keys(m.files).length!==requiredPaths.length || requiredPaths.some(p=>!Object.hasOwn(m.files,p))) throw Error('source manifest path set mismatch');
  for (const p of requiredPaths) if (!/^[a-f0-9]{64}$/.test(m.files[p]) || sha(readFile(p))!==m.files[p]) throw Error(`source manifest bytes mismatch: ${p}`);
}

export function createHandleCustody({wait,record}) {
  const timedOut=new Set(), waited=new Set(); let anyTimeout=false;
  return {
    get anyTimeout(){return anyTimeout;},
    async wait(child,name,timeoutMs) {
      if (waited.has(child)) throw Error(`${name} original handle already waited`);
      waited.add(child);
      const result={name,...await wait(child,timeoutMs)};
      if(result.timeout){timedOut.add(child);anyTimeout=true;}
      record('original_handle_wait',result);
      return result;
    },
    hasTimedOut(child){return timedOut.has(child);},
  };
}

export async function replaceOwnedChrome({handle,close,snapshot,ledger,launch,record}) {
  await close(handle);
  const terminal=ledger.terminal(await snapshot());
  record('first_chrome_terminal',terminal);
  if(terminal.verdict!=='ABSENT') throw Error('Chrome #1 cleanup not proven');
  return launch();
}

export function terminalChromeDecision({ledgerResults,waitTimedOut,cleanupErrors=[]}) {
  return !waitTimedOut && cleanupErrors.length===0 && ledgerResults.length>0 && ledgerResults.every(x=>x.verdict==='ABSENT') ? 'ABSENT' : 'UNKNOWN';
}

const chrome = r => String(r.name || '').toLowerCase() === 'chrome.exe';
const key = r => `${r.pid}:${r.createdAt}`;
export class OwnershipLedger {
  constructor(rootPid, exe) { this.rootPid=rootPid; this.exe=path.win32.normalize(exe).toLowerCase(); this.known=new Map(); this.root=null; }
  observe(rows) {
    if (!Array.isArray(rows)) throw Error('inventory unknown');
    const byPid=new Map();
    for (const r of rows) {
      if (!Number.isInteger(r.pid) || byPid.has(r.pid)) throw Error('ambiguous process inventory');
      byPid.set(r.pid,r);
      if (chrome(r) && (!r.commandLineKnown || !r.createdAt || !r.executablePath)) throw Error('Chrome inventory unknown');
    }
    const root=byPid.get(this.rootPid);
    if (!root || !chrome(root) || !root.ownedProfileArg || path.win32.normalize(root.executablePath).toLowerCase() !== this.exe) throw Error('positive root witness missing');
    if (this.root && key(root)!==this.root) throw Error('root PID reused');
    this.root=key(root); this.known.set(this.root,root);
    for (const r of rows) {
      if (!chrome(r)) continue;
      let p=r, found=false; const seen=new Set();
      while (p && !seen.has(p.pid)) {
        seen.add(p.pid);
        if (key(p)===this.root || this.known.has(key(p))) { found=true; break; }
        p=byPid.get(p.parentPid);
      }
      if (found) this.known.set(key(r),r);
      else if (r.ownedProfileArg) throw Error('foreign profile process');
    }
    return [...this.known.values()];
  }
  terminal(rows) {
    if (!Array.isArray(rows) || !this.root) return {verdict:'UNKNOWN',identities:[]};
    const byPid=new Map();
    for (const r of rows) { if (!Number.isInteger(r.pid) || byPid.has(r.pid)) return {verdict:'UNKNOWN',identities:[]}; byPid.set(r.pid,r); }
    const identities=[...this.known.values()].map(r=>{
      const current=byPid.get(r.pid);
      return {pid:r.pid,createdAt:r.createdAt,state:!current?'ABSENT':!current.createdAt?'UNKNOWN':current.createdAt===r.createdAt?'PRESENT':'PID_REUSED'};
    });
    return {verdict:identities.every(i=>i.state==='ABSENT')?'ABSENT':'UNKNOWN',identities};
  }
}

export function waitOriginal(child, timeoutMs, setTimer=setTimeout, clearTimer=clearTimeout) {
  return new Promise(resolve=>{
    let timer, done=false;
    const finish=(result)=>{ if(done)return; done=true; if(timer) clearTimer(timer); child.removeListener('exit',onExit); resolve(result); };
    const onExit=(code,signal)=>finish({pid:child.pid,exitCode:code,signalCode:signal});
    if (child.exitCode!==null || child.signalCode!==null) return onExit(child.exitCode,child.signalCode);
    child.once('exit',onExit); timer=setTimer(()=>finish({pid:child.pid,timeout:true}),timeoutMs);
  });
}

export function createUpgradeCapture({key, maxHeader=16384, settle, teardown}) {
  let bytes=Buffer.alloc(0), done=false;
  const finish=(result)=>{ if(done)return; done=true; teardown(); settle(result); };
  const accept=createHash('sha1').update(key+'258EAFA5-E914-47DA-95CA-C5AB0DC85B11').digest('base64');
  return {
    data(chunk) {
      if(done)return; bytes=Buffer.concat([bytes,Buffer.from(chunk)]);
      const end=bytes.indexOf('\r\n\r\n');
      if(end<0) { if(bytes.length>maxHeader)finish({classification:'INCONCLUSIVE',reason:'oversized headers'}); return; }
      if(end+4>maxHeader) return finish({classification:'INCONCLUSIVE',reason:'oversized headers'});
      const lines=bytes.subarray(0,end).toString('latin1').split('\r\n');
      const status=/^HTTP\/(1\.[01]) ([0-9]{3})(?: [\x20-\x7e]*)?$/.exec(lines.shift());
      if(!status) return finish({classification:'INCONCLUSIVE',reason:'malformed status'});
      const headers={};
      for(const line of lines) { const m=/^([!#$%&'*+.^_`|~0-9A-Za-z-]+):[ \t]*([^\r\n]*)$/.exec(line); if(!m)return finish({classification:'INCONCLUSIVE',reason:'malformed header'}); const k=m[1].toLowerCase(); if(headers[k]!==undefined)return finish({classification:'INCONCLUSIVE',reason:'duplicate header'}); headers[k]=m[2].trim(); }
      const code=Number(status[2]); const evidence={httpVersion:status[1],status:code,headers};
      if(code>=400 && code<=499) return finish({classification:'HTTP_4XX_REFUSAL',...evidence});
      if(code===101 && headers.upgrade?.toLowerCase()==='websocket' && /(?:^|,)\s*upgrade\s*(?:,|$)/i.test(headers.connection||'') && headers['sec-websocket-accept']===accept) return finish({classification:'A1_FAIL',...evidence});
      finish({classification:'INCONCLUSIVE',reason:'unaccepted status or invalid upgrade',...evidence});
    },
    end() { finish(bytes.length?{classification:'INCONCLUSIVE',reason:'incomplete response'}:{classification:'REMOTE_CLOSE_BEFORE_OPEN'}); },
    error(error) { finish({classification:'INCONCLUSIVE',reason:'transport error',error:String(error)}); },
    timeout() { finish({classification:'INCONCLUSIVE',reason:'timeout'}); },
  };
}

export function negativeProbe(url, debit, timeoutMs=10000, deps={}) {
  const u=new URL(url);
  if(u.protocol!=='ws:' || u.hostname!=='127.0.0.1' || !u.port || u.username || u.password || u.search || u.hash || !/^\/devtools\/browser\/[0-9a-f-]+$/i.test(u.pathname)) throw Error('invalid endpoint');
  debit(); const key=(deps.randomBytes || randomBytes)(16).toString('base64');
  return new Promise(resolve=>{
    const socket=(deps.connect || net.createConnection)({host:'127.0.0.1',port:Number(u.port)}); let timer;
    const request=`GET ${u.pathname} HTTP/1.1\r\nHost: 127.0.0.1:${u.port}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: ${key}\r\nSec-WebSocket-Version: 13\r\n\r\n`;
    const cap=createUpgradeCapture({key,settle:r=>resolve({...r,url:u.href,requestKey:key,requestHeaders:{host:`127.0.0.1:${u.port}`,upgrade:'websocket',connection:'Upgrade',version:'13'}}),teardown:()=>{(deps.clearTimer || clearTimeout)(timer);socket.destroy();}});
    timer=(deps.setTimer || setTimeout)(()=>cap.timeout(),timeoutMs);
    socket.once('connect',()=>socket.write(request));
    socket.on('data',c=>cap.data(c)); socket.once('end',()=>cap.end()); socket.once('error',e=>cap.error(e));
    socket.once('close',()=>cap.error('socket closed without remote EOF'));
  });
}
