param([string]$BuildCommit = '145bd41109c1e3ac3fb1a85f261c61f668b2fe4d')
$ErrorActionPreference = 'Stop'
$projectRoot = (Get-Location).Path
$taskRoot = Join-Path $env:TEMP ('ccmai-r051-r2-independent-' + [guid]::NewGuid().ToString('N').Substring(0,8))
$sourceRoot = Join-Path $taskRoot 'source'
New-Item -ItemType Directory -Path $sourceRoot -Force | Out-Null
$archive = Join-Path $taskRoot 'source.tar'
git archive --format=tar --output=$archive $BuildCommit backend
if ($LASTEXITCODE -ne 0) { throw 'Cannot export exact BUILD' }
tar -xf $archive -C $sourceRoot
if ($LASTEXITCODE -ne 0) { throw 'Cannot extract exact BUILD' }
$backend = Join-Path $sourceRoot 'backend'
$modCache = (& go env GOMODCACHE).Trim()
if (-not (Test-Path -LiteralPath $modCache)) { throw 'Missing offline dependency cache' }
foreach ($img in @('mysql:8.0','golang:1.26-alpine')) {
    docker image inspect $img --format '{{.Id}}' | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Missing cached image; pulling is forbidden' }
}
$suffix = [guid]::NewGuid().ToString('N').Substring(0,8)
$network = 'ccmai-r051-r2-review-net-' + $suffix
$database = 'ccmai-r051-r2-review-db-' + $suffix
$fixturePassword = [guid]::NewGuid().ToString('N')
$receipt = [ordered]@{buildCommit=$BuildCommit; taskRoot=$taskRoot; archiveSha256=(Get-FileHash $archive).Hash; internalNetwork=$network; disposableDatabase=$database; commands=@(); externalNetwork=$false; hostPorts=$false; newPulls=$false; cleanup=@{}}
try {
    docker network create --internal $network | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Cannot create isolated network' }
    docker run --pull=never -d --name $database --network $network -e "MYSQL_ROOT_PASSWORD=$fixturePassword" -e MYSQL_DATABASE=CCMA -e MYSQL_USER=ccma -e "MYSQL_PASSWORD=$fixturePassword" mysql:8.0 --log-bin-trust-function-creators=1 --max-connections=1000 | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Cannot start disposable database' }
    $ready=$false
    for ($i=0;$i -lt 45;$i++) {
        $ErrorActionPreference='Continue'
        $value=docker exec -e "MYSQL_PWD=$fixturePassword" $database mysql -uccma -N -e 'SELECT @@log_bin_trust_function_creators' CCMA 2>$null
        $dbExit=$LASTEXITCODE
        $ErrorActionPreference='Stop'
        if ($dbExit -eq 0 -and "$value".Trim() -eq '1') { $ready=$true; break }
        Start-Sleep -Seconds 2
    }
    if (-not $ready) { throw 'Disposable database readiness failed' }
    $receipt.networkInternal = ((docker network inspect $network | ConvertFrom-Json)[0].Internal)
    $receipt.portBindings = ((docker inspect $database | ConvertFrom-Json)[0].HostConfig.PortBindings)
    $receipt.databaseVolumes = @((docker inspect $database | ConvertFrom-Json)[0].Mounts | Where-Object Type -eq 'volume' | Select-Object -ExpandProperty Name)
    $runs=@(
        @{name='exact-repair'; args=@('test','./engine','-json','-count=1','-p','1','-timeout','300s','-run','TestLP|TestEveryTerminalPath|TestEarlyFailureClasses|TestEarlyFailuresAreCheckedBoundedAndReleaseOwnership|TestFL|TestF05|TestF06|TestTerminalRunHoldsTheSlot')}
    )
    foreach ($run in $runs) {
        $logPath=Join-Path $taskRoot ($run.name+'.jsonl')
        $start=Get-Date
        $ErrorActionPreference='Continue'
        docker run --pull=never --rm --network $network --mount "type=bind,source=$backend,target=/src,readonly" --mount "type=bind,source=$modCache,target=/go/pkg/mod,readonly" -w /src -e GOFLAGS=-mod=readonly -e GOPROXY=off -e GOTOOLCHAIN=local -e CGO_ENABLED=0 -e "TEST_DB_DSN=ccma:${fixturePassword}@tcp(${database}:3306)/CCMA?charset=utf8mb4&parseTime=True&loc=UTC" golang:1.26-alpine go @($run.args) *> $logPath
        $result=$LASTEXITCODE
        $ErrorActionPreference='Stop'
        $receipt.commands += @{name=$run.name;goArgs=$run.args;exit=$result;seconds=((Get-Date)-$start).TotalSeconds;logPath=$logPath;logSha256=(Get-FileHash $logPath).Hash}
        Write-Host ($run.name+': exit '+$result)
    }
} finally {
    $ErrorActionPreference='Continue'
    docker rm -f -v $database 2>$null | Out-Null
    $receipt.cleanup.databaseRemoveExit=$LASTEXITCODE
    docker network rm $network 2>$null | Out-Null
    $receipt.cleanup.networkRemoveExit=$LASTEXITCODE
    $receipt.cleanup.databaseAbsent = -not [bool](docker ps -a --filter "name=^/$database$" --format '{{.Names}}')
    $receipt.cleanup.networkAbsent = -not [bool](docker network ls --filter "name=^$network$" --format '{{.Name}}')
    $volumesAfter = @(docker volume ls --format '{{.Name}}')
    $receipt.cleanup.remainingNamedVolumes = @($receipt.databaseVolumes | Where-Object { $_ -in $volumesAfter })
    $receipt.cleanup.anonymousVolumesAbsent = $receipt.cleanup.remainingNamedVolumes.Count -eq 0
    $receipt | ConvertTo-Json -Depth 12 | Set-Content -LiteralPath (Join-Path $taskRoot 'receipt.json') -Encoding utf8
    Write-Host ('RECEIPT: '+(Join-Path $taskRoot 'receipt.json'))
}
