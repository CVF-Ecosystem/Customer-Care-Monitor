"""Meter one authorized candidate invocation and preserve its exact process output."""
import datetime
import json
from pathlib import Path
import subprocess
import sys
import time

here=Path(__file__).resolve().parent
clock=json.loads((here/'clock.json').read_text())
elapsed=time.monotonic()-clock['startMonotonic']
if elapsed>=clock['ceilingSeconds'] or clock['r156Invocations']>=2:
    raise SystemExit('R156 allowance exhausted')
clock['r156Invocations']+=1
(here/'clock.json').write_text(json.dumps(clock,indent=2)+'\n')
number=clock['r156Invocations']
command=[sys.executable,'-B',str(here/'test_join.py'),*sys.argv[1:]]
start=time.monotonic()
with (here/f'candidate_run_{number}.txt').open('wb') as output:
    result=subprocess.run(command,stdout=output,stderr=subprocess.STDOUT,
                          timeout=max(1,int(clock['ceilingSeconds']-elapsed)))
receipt={'label':'STATIC_CANDIDATE_ONLY','command':command,'exitCode':result.returncode,
         'invocation':number,'cumulativeWorkerInvocations':3+number,
         'durationSeconds':time.monotonic()-start,
         'prospectiveElapsedSeconds':time.monotonic()-clock['startMonotonic'],
         'endUTC':datetime.datetime.now(datetime.timezone.utc).isoformat()}
(here/f'candidate_run_{number}.json').write_text(json.dumps(receipt,indent=2)+'\n')
print(json.dumps(receipt))
print((here/f'candidate_run_{number}.txt').read_text())
raise SystemExit(result.returncode)
