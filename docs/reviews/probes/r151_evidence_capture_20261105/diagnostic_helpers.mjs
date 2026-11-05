// Offline-testable diagnostic primitives. No effect authority is conveyed here.
import net from 'node:net';
import {randomBytes, createHash} from 'node:crypto';
import path from 'node:path';

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

export function negativeProbe(url, debit, timeoutMs=10000) {
  const u=new URL(url);
  if(u.protocol!=='ws:' || u.hostname!=='127.0.0.1' || !u.port || u.username || u.password || u.search || u.hash || !/^\/devtools\/browser\/[0-9a-f-]+$/i.test(u.pathname)) throw Error('invalid endpoint');
  debit(); const key=randomBytes(16).toString('base64');
  return new Promise(resolve=>{
    const socket=net.createConnection({host:'127.0.0.1',port:Number(u.port)}); let timer;
    const cap=createUpgradeCapture({key,settle:resolve,teardown:()=>{clearTimeout(timer);socket.destroy();}});
    timer=setTimeout(()=>cap.timeout(),timeoutMs);
    socket.once('connect',()=>socket.write(`GET ${u.pathname} HTTP/1.1\r\nHost: 127.0.0.1:${u.port}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: ${key}\r\nSec-WebSocket-Version: 13\r\n\r\n`));
    socket.on('data',c=>cap.data(c)); socket.once('end',()=>cap.end()); socket.once('error',e=>cap.error(e));
    socket.once('close',()=>cap.error('socket closed without remote EOF'));
  });
}
