$ErrorActionPreference = 'Stop'
$env:GOPROXY = 'off'
$env:GOSUMDB = 'off'
$env:GOTOOLCHAIN = 'local'
& 'C:/Program Files/Go/bin/go.exe' -C backend test ./engine ./api/handlers -run '^$' 1> docs/reviews/probes/r116_worker/go6_stdout.txt 2> docs/reviews/probes/r116_worker/go6_stderr.txt
$code = $LASTEXITCODE
Set-Content -LiteralPath docs/reviews/probes/r116_worker/go6_exit.txt -Value $code
exit $code
