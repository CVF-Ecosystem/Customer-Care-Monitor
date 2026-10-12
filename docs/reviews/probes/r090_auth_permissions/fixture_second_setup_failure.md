# Second synthetic fixture setup attempt

Risk: R2
Reviewer: Codex /root

Worker: `/root/auth_intake`; tranche: `CCMAI-RUNTIME-090`.

The second `python -B scripts/r089_auth_fixture.py start` attempt exited 1 at `docker port ccma-r090-db-7c4b599d31f7 3306/tcp` with Docker's exact output:

```text
no public port '3306/tcp' published for ccma-r090-db-7c4b599d31f7
```

The image inspection reported exposed ports `3306/tcp` and `33060/tcp`. The `docker run` call itself returned successfully with `--publish 127.0.0.1::3306`, `--network ccma-r090-net-7c4b599d31f7`, and image digest `sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b`. The helper immediately removed the uniquely labeled container and internal network and verified their labels; a subsequent `docker ps --all` was empty. This cleanup used `docker rm --force` without `--volumes` and did not snapshot mount metadata. Root observed two anonymous volumes and is correlating Docker events to the two failed container incarnations; worker does not claim these volumes were removed. No test ran and this is not source-negative detector credit.

The second setup manifest is `fixture_manifest_second_setup_failure.json`. It predates the state-error persistence correction and therefore lacks the startup error field; this note preserves Docker's exact diagnostic. At the time, I did not retry before Root diagnosed the internal-bridge publication behavior. The helper now uses the selected ordinary bridge, persists ready state before test dispatch, enforces a 900-second test deadline, and writes timed-out output as partial/NOT PASS.

Final cleanup correction: Root matched this attempt's anonymous volume `6539043069f00a5096e9148a34d452a2c55ef1c7bc85deb0e84159825136417c` to container `2f3f79e86f783ebb8b3bc32d4e06d58c678830fe86f9980eb8750d0a4a76f020` / `ccma-r090-db-7c4b599d31f7` using Docker create/mount/unmount/destroy events, removed it, and verified absence. See `local_anonymous_volume_cleanup.json`; no unrelated volume was removed.
