"""Root foreground command recorder; no effect on import. Do not run without admission."""
import ctypes,json,os,subprocess,sys,time
from pathlib import Path
ROOT=Path(__file__).resolve().parents[4]
def write(path,value):
 with path.open('x',encoding='utf-8') as f:
  json.dump(value,f,indent=2);f.write('\n');f.flush();os.fsync(f.fileno())
def creation(handle):
 from ctypes import wintypes
 k=ctypes.WinDLL('kernel32',use_last_error=True);v=[wintypes.FILETIME() for _ in range(4)]
 f=k.GetProcessTimes;f.argtypes=[wintypes.HANDLE]+[ctypes.POINTER(wintypes.FILETIME)]*4;f.restype=wintypes.BOOL
 if not f(wintypes.HANDLE(handle),*[ctypes.byref(a) for a in v]):raise RuntimeError('CREATION_ID_UNAVAILABLE')
 return 'windows-filetime-%016x'%((v[0].dwHighDateTime<<32)|v[0].dwLowDateTime)
def main():
 mode=sys.argv[1]
 if mode not in {'setup','serve','stop'}:raise SystemExit('INVALID_MODE')
 slot=sys.argv[2] if len(sys.argv)==3 else '001'
 if slot not in {'001','002'}:raise SystemExit('INVALID_RECORDING_SLOT')
 out=ROOT/'docs/reviews/probes/r121_root'/(mode+'_'+slot);out.mkdir(exist_ok=False)
 argv=[sys.executable,'-B',str(ROOT/'scripts/r121_ui_supervisor.py'),mode]
 write(out/'reservation.json',{'mode':mode,'argv':argv,'wrapperPid':os.getpid(),'startedAt':time.time(),'invocation':int(slot)})
 env={k:v for k,v in os.environ.items() if k.upper() in {'SYSTEMROOT','WINDIR','TEMP','TMP','PATH','USERPROFILE','LOCALAPPDATA'}}
 with (out/'stdout.bin').open('xb') as stdout,(out/'stderr.bin').open('xb') as stderr:
  child=subprocess.Popen(argv,cwd=ROOT,env=env,stdout=stdout,stderr=stderr,creationflags=subprocess.CREATE_NO_WINDOW)
  metadata_error=None
  try:
   write(out/'child_started.json',{'pid':child.pid,'creationIdentity':creation(child._handle),'at':time.time()})
   print('FOREGROUND_HELD '+mode+' pid='+str(child.pid),flush=True)
  except BaseException as exc:
   metadata_error=type(exc).__name__
  finally:
   result=child.wait()
 write(out/'receipt.json',{'exit':result,'waitObserved':True,'childPid':child.pid,'metadataError':metadata_error,'finishedAt':time.time()})
 print('FOREGROUND_EXIT '+mode+' '+str(result),flush=True)
 return result if metadata_error is None else 1
if __name__=='__main__':sys.exit(main())
