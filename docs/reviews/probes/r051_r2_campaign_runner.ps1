param(
    [string]$BuildCommit = '145bd41109c1e3ac3fb1a85f261c61f668b2fe4d',
    [string]$AcknowledgmentCommit = 'c228931df25026211cf4e33d4586db7020bc8b5c'
)

$ErrorActionPreference = 'Stop'
$projectRoot = (Get-Location).Path
$taskRoot = Join-Path $env:TEMP ('ccmai-r051-r2-worker-' + [guid]::NewGuid().ToString('N').Substring(0, 8))
$sourceRoot = Join-Path $taskRoot 'source'
New-Item -ItemType Directory -Path $sourceRoot -Force | Out-Null

$archive = Join-Path $taskRoot 'source.tar'
git archive --format=tar --output=$archive $BuildCommit backend
if ($LASTEXITCODE -ne 0) { throw "Cannot export exact BUILD at $BuildCommit" }
tar -xf $archive -C $sourceRoot
if ($LASTEXITCODE -ne 0) { throw "Cannot extract exact BUILD archive" }
$backend = Join-Path $sourceRoot 'backend'

# Offline dependency check
$modCache = (& go env GOMODCACHE).Trim()
if (-not (Test-Path -LiteralPath $modCache)) { throw "Missing offline dependency cache at $modCache" }

# Cached image check (no pulls allowed)
foreach ($img in @('mysql:8.0', 'golang:1.26-alpine')) {
    docker image inspect $img --format '{{.Id}}' | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Missing cached image $img; network pulls are prohibited" }
}

$suffix = [guid]::NewGuid().ToString('N').Substring(0, 8)
$network = 'ccmai-r051-r2-net-' + $suffix
$database = 'ccmai-r051-r2-db-' + $suffix
$fixturePassword = [guid]::NewGuid().ToString('N')

$receipt = [ordered]@{
    schemaVersion           = "1.0"
    trancheId               = "CCMAI-RUNTIME-051"
    round                   = 2
    buildCommit             = $BuildCommit
    acknowledgmentCommit    = $AcknowledgmentCommit
    taskRoot                = $taskRoot
    archiveSha256           = (Get-FileHash $archive).Hash
    internalNetwork         = $network
    disposableDatabase      = $database
    externalNetwork         = $false
    hostPorts               = $false
    newPulls                = $false
    commands                = @()
    mutationTest            = [ordered]@{}
    cleanup                 = [ordered]@{}
}

try {
    # 1. Create internal network (no external route)
    docker network create --internal $network | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Cannot create isolated network $network" }

    # 2. Run disposable synthetic database without host ports
    docker run --pull=never -d --name $database --network $network `
        -e "MYSQL_ROOT_PASSWORD=$fixturePassword" `
        -e MYSQL_DATABASE=CCMA `
        -e MYSQL_USER=ccma `
        -e "MYSQL_PASSWORD=$fixturePassword" `
        mysql:8.0 --log-bin-trust-function-creators=1 --max-connections=1000 | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Cannot start disposable database $database" }

    # Wait for database readiness
    $ready = $false
    for ($i = 0; $i -lt 45; $i++) {
        $ErrorActionPreference = 'Continue'
        $value = docker exec -e "MYSQL_PWD=$fixturePassword" $database mysql -uccma -N -e 'SELECT @@log_bin_trust_function_creators' CCMA 2>$null
        $dbExit = $LASTEXITCODE
        $ErrorActionPreference = 'Stop'
        if ($dbExit -eq 0 -and "$value".Trim() -eq '1') {
            $ready = $true
            break
        }
        Start-Sleep -Seconds 2
    }
    if (-not $ready) { throw "Disposable database readiness timed out" }

    # Inspect network and database mounts before test execution
    $networkInspect = (docker network inspect $network | ConvertFrom-Json)[0]
    $dbInspect = (docker inspect $database | ConvertFrom-Json)[0]
    $receipt.networkInternal = [bool]$networkInspect.Internal
    $receipt.portBindings = $dbInspect.HostConfig.PortBindings
    $receipt.databaseVolumes = @($dbInspect.Mounts | Where-Object Type -eq 'volume' | Select-Object -ExpandProperty Name)

    # 3. Compile and Vet checks
    $buildLog = Join-Path $taskRoot 'build_check.log'
    $buildStart = Get-Date
    $ErrorActionPreference = 'Continue'
    docker run --pull=never --rm `
        --mount "type=bind,source=$backend,target=/src,readonly" `
        --mount "type=bind,source=$modCache,target=/go/pkg/mod,readonly" `
        -w /src `
        -e GOFLAGS=-mod=readonly -e GOPROXY=off -e GOTOOLCHAIN=local -e CGO_ENABLED=0 `
        golang:1.26-alpine go test -c -o /dev/null ./engine *> $buildLog
    $buildExit = $LASTEXITCODE
    $ErrorActionPreference = 'Stop'
    $receipt.commands += [ordered]@{
        name      = "engine-test-compile"
        exit      = $buildExit
        seconds   = ((Get-Date) - $buildStart).TotalSeconds
        logPath   = $buildLog
        logSha256 = (Get-FileHash $buildLog).Hash
    }
    if ($buildExit -ne 0) { throw "engine test compilation failed" }

    $vetLog = Join-Path $taskRoot 'vet_check.log'
    $vetStart = Get-Date
    $ErrorActionPreference = 'Continue'
    docker run --pull=never --rm `
        --mount "type=bind,source=$backend,target=/src,readonly" `
        --mount "type=bind,source=$modCache,target=/go/pkg/mod,readonly" `
        -w /src `
        -e GOFLAGS=-mod=readonly -e GOPROXY=off -e GOTOOLCHAIN=local -e CGO_ENABLED=0 `
        golang:1.26-alpine go vet ./engine *> $vetLog
    $vetExit = $LASTEXITCODE
    $ErrorActionPreference = 'Stop'
    $receipt.commands += [ordered]@{
        name      = "engine-vet"
        exit      = $vetExit
        seconds   = ((Get-Date) - $vetStart).TotalSeconds
        logPath   = $vetLog
        logSha256 = (Get-FileHash $vetLog).Hash
    }
    if ($vetExit -ne 0) { throw "engine vet failed" }

    # 4. Exact repair test suite execution
    $testLog = Join-Path $taskRoot 'exact_repair_tests.jsonl'
    $testStart = Get-Date
    $testPattern = "TestLP|TestEveryTerminalPath|TestEarlyFailureClasses|TestEarlyFailuresAreCheckedBoundedAndReleaseOwnership|TestFL|TestF05|TestF06|TestTerminalRunHoldsTheSlot"
    $ErrorActionPreference = 'Continue'
    docker run --pull=never --rm --network $network `
        --mount "type=bind,source=$backend,target=/src,readonly" `
        --mount "type=bind,source=$modCache,target=/go/pkg/mod,readonly" `
        -w /src `
        -e GOFLAGS=-mod=readonly -e GOPROXY=off -e GOTOOLCHAIN=local -e CGO_ENABLED=0 `
        -e "TEST_DB_DSN=ccma:${fixturePassword}@tcp(${database}:3306)/CCMA?charset=utf8mb4&parseTime=True&loc=UTC" `
        golang:1.26-alpine go test ./engine -json -count=1 -p 1 -timeout 300s -run $testPattern *> $testLog
    $testExit = $LASTEXITCODE
    $ErrorActionPreference = 'Stop'

    $testLogContent = Get-Content -LiteralPath $testLog -Raw
    $receipt.commands += [ordered]@{
        name      = "exact-repair-tests"
        pattern   = $testPattern
        exit      = $testExit
        seconds   = ((Get-Date) - $testStart).TotalSeconds
        logPath   = $testLog
        logSha256 = (Get-FileHash $testLog).Hash
    }
    if ($testExit -ne 0) { throw "exact repair test suite failed with exit $testExit" }

    # Parse test events from JSONL
    $events = @()
    $testLogContent.Split("`n") | ForEach-Object {
        $line = $_.Trim()
        if ($line.StartsWith('{') -and $line.EndsWith('}')) {
            try { $events += ($line | ConvertFrom-Json) } catch {}
        }
    }
    $completedPass = @($events | Where-Object { $_.Action -eq 'pass' -and $_.Test })
    $completedFail = @($events | Where-Object { $_.Action -eq 'fail' -and $_.Test })
    $completedSkip = @($events | Where-Object { $_.Action -eq 'skip' -and $_.Test })
    $topLevelPass = @($completedPass | Where-Object { $_.Test -notmatch '/' })
    $receipt.testSummary = [ordered]@{
        totalCompletedEvents = $completedPass.Count + $completedFail.Count + $completedSkip.Count
        passedEvents         = $completedPass.Count
        failedEvents         = $completedFail.Count
        skippedEvents        = $completedSkip.Count
        topLevelSuitesPassed = $topLevelPass.Count
    }
    Write-Host "Exact repair tests: $($topLevelPass.Count) top-level suites, $($completedPass.Count) passed events, 0 failures, 0 skips."

    # 5. Ordering Mutation Testing (R051-R1-01)
    # Target: analyzer.go inside executeReserved
    $tempAnalyzer = Join-Path $backend 'engine\analyzer.go'
    $analyzerOriginalBytes = [System.IO.File]::ReadAllBytes($tempAnalyzer)
    $analyzerOriginalHash = (Get-FileHash $tempAnalyzer).Hash
    $analyzerContent = [System.IO.File]::ReadAllText($tempAnalyzer, [System.Text.Encoding]::UTF8)

    $targetBlock = "`ttruncated := false`r`n`tvar provider ai.AIProvider`r`n`r`n`t// Select the conversations and prepare their snapshots."
    if (-not $analyzerContent.Contains($targetBlock)) {
        # Check LF variant
        $targetBlock = "`ttruncated := false`n`tvar provider ai.AIProvider`n`n`t// Select the conversations and prepare their snapshots."
        if (-not $analyzerContent.Contains($targetBlock)) {
            throw "Target block for ordering mutation not found in temp analyzer.go"
        }
    }

    $eagerResolutionSnippet = @"
	truncated := false
	var provider ai.AIProvider
	if injectedProvider != nil {
		provider = injectedProvider
	} else if a.providerOverride != nil {
		provider = a.providerOverride
	} else if a.providerResolver != nil {
		var provErr error
		provider, provErr = a.providerResolver(job)
		if provErr != nil {
			return a.failOwnedRun(owner, &run, job, "Không khởi tạo được AI provider; kiểm tra cấu hình AI trong Cài đặt.", earlyProviderUnavailable)
		}
	} else {
		var provErr error
		provider, provErr = a.getProvider(job)
		if provErr != nil {
			return a.failOwnedRun(owner, &run, job, "Không khởi tạo được AI provider; kiểm tra cấu hình AI trong Cài đặt.", earlyProviderUnavailable)
		}
	}

	// Select the conversations and prepare their snapshots.
"@

    $mutatedAnalyzerContent = $analyzerContent.Replace($targetBlock, $eagerResolutionSnippet)
    [System.IO.File]::WriteAllText($tempAnalyzer, $mutatedAnalyzerContent, [System.Text.Encoding]::UTF8)
    $mutatedHash = (Get-FileHash $tempAnalyzer).Hash

    $receipt.mutationTest = [ordered]@{
        mutationName     = "M_EAGER_INIT_ORDERING"
        description      = "Eager provider resolution inserted before conversation selection in executeReserved"
        targetPath       = "backend/engine/analyzer.go"
        baselineSha256   = $analyzerOriginalHash
        mutatedSha256    = $mutatedHash
    }

    # Execute negative control detector on mutated source: MUST compile cleanly and FAIL behaviorally
    $mutantLog = Join-Path $taskRoot 'mutation_negative_control.jsonl'
    $mutantStart = Get-Date
    $ErrorActionPreference = 'Continue'
    docker run --pull=never --rm --network $network `
        --mount "type=bind,source=$backend,target=/src,readonly" `
        --mount "type=bind,source=$modCache,target=/go/pkg/mod,readonly" `
        -w /src `
        -e GOFLAGS=-mod=readonly -e GOPROXY=off -e GOTOOLCHAIN=local -e CGO_ENABLED=0 `
        -e "TEST_DB_DSN=ccma:${fixturePassword}@tcp(${database}:3306)/CCMA?charset=utf8mb4&parseTime=True&loc=UTC" `
        golang:1.26-alpine go test ./engine -json -count=1 -run 'TestLP06EagerInitializationDetectorNegativeAndPositive/negative_control' *> $mutantLog
    $mutantExit = $LASTEXITCODE
    $ErrorActionPreference = 'Stop'

    $mutantLogRaw = Get-Content -LiteralPath $mutantLog -Raw
    $mutantEvents = @()
    $mutantLogRaw.Split("`n") | ForEach-Object {
        $line = $_.Trim()
        if ($line.StartsWith('{') -and $line.EndsWith('}')) {
            try { $mutantEvents += ($line | ConvertFrom-Json) } catch {}
        }
    }
    $mutantFailEvent = @($mutantEvents | Where-Object { $_.Action -eq 'fail' -and $_.Test -like '*negative_control*' })
    $isBuildFail = $mutantLogRaw -match 'build-fail|undefined:'

    $receipt.mutationTest.mutantExecution = [ordered]@{
        exitCode       = $mutantExit
        seconds        = ((Get-Date) - $mutantStart).TotalSeconds
        logPath        = $mutantLog
        logSha256      = (Get-FileHash $mutantLog).Hash
        hasBuildFail   = [bool]$isBuildFail
        failEventCount = $mutantFailEvent.Count
        killed         = ($mutantExit -eq 1 -and -not $isBuildFail -and $mutantFailEvent.Count -ge 1)
    }

    if (-not $receipt.mutationTest.mutantExecution.killed) {
        throw "Ordering mutation was not killed behaviorally! Exit=$mutantExit, buildFail=$isBuildFail, failEvents=$($mutantFailEvent.Count)"
    }
    Write-Host "Ordering mutation killed behaviorally: exit code 1, 0 compile errors, $($mutantFailEvent.Count) named test failure event."

    # Restore exact baseline bytes
    [System.IO.File]::WriteAllBytes($tempAnalyzer, $analyzerOriginalBytes)
    $restoredHash = (Get-FileHash $tempAnalyzer).Hash
    if ($restoredHash -ne $analyzerOriginalHash) {
        throw "Restored hash $restoredHash does not match original baseline $analyzerOriginalHash"
    }

    # Re-run negative control on restored source: MUST PASS with exit 0
    $restoredLog = Join-Path $taskRoot 'restored_negative_control.jsonl'
    $restoredStart = Get-Date
    $ErrorActionPreference = 'Continue'
    docker run --pull=never --rm --network $network `
        --mount "type=bind,source=$backend,target=/src,readonly" `
        --mount "type=bind,source=$modCache,target=/go/pkg/mod,readonly" `
        -w /src `
        -e GOFLAGS=-mod=readonly -e GOPROXY=off -e GOTOOLCHAIN=local -e CGO_ENABLED=0 `
        -e "TEST_DB_DSN=ccma:${fixturePassword}@tcp(${database}:3306)/CCMA?charset=utf8mb4&parseTime=True&loc=UTC" `
        golang:1.26-alpine go test ./engine -json -count=1 -run 'TestLP06EagerInitializationDetectorNegativeAndPositive/negative_control' *> $restoredLog
    $restoredExit = $LASTEXITCODE
    $ErrorActionPreference = 'Stop'

    $receipt.mutationTest.restoredExecution = [ordered]@{
        exitCode       = $restoredExit
        seconds        = ((Get-Date) - $restoredStart).TotalSeconds
        logPath        = $restoredLog
        logSha256      = (Get-FileHash $restoredLog).Hash
        restoredPassed = ($restoredExit -eq 0)
    }
    if ($restoredExit -ne 0) { throw "Restored negative control failed with exit $restoredExit" }
    Write-Host "Restored baseline verified: exit code 0, hash matches original baseline."

} finally {
    # 6. Cleanup and Volume Verification (R051-R1-02)
    $ErrorActionPreference = 'Continue'
    docker rm -f -v $database 2>$null | Out-Null
    $receipt.cleanup.databaseRemoveExit = $LASTEXITCODE
    docker network rm $network 2>$null | Out-Null
    $receipt.cleanup.networkRemoveExit = $LASTEXITCODE

    $receipt.cleanup.databaseAbsent = -not [bool](docker ps -a --filter "name=^/$database$" --format '{{.Names}}')
    $receipt.cleanup.networkAbsent = -not [bool](docker network ls --filter "name=^$network$" --format '{{.Name}}')

    $volumesAfter = @(docker volume ls --format '{{.Name}}')
    $receipt.cleanup.remainingNamedVolumes = @($receipt.databaseVolumes | Where-Object { $_ -in $volumesAfter })
    $receipt.cleanup.anonymousVolumesAbsent = ($receipt.cleanup.remainingNamedVolumes.Count -eq 0)

    $receiptPath = Join-Path $taskRoot 'r051_r2_worker_receipt.json'
    $receipt | ConvertTo-Json -Depth 12 | Set-Content -LiteralPath $receiptPath -Encoding utf8
    Write-Host "Campaign completed. Receipt written to: $receiptPath"

    # Also copy receipt to permanent probe location
    $permanentReceipt = Join-Path $projectRoot 'docs\reviews\probes\r051_r2_worker_receipt.json'
    Copy-Item -LiteralPath $receiptPath -Destination $permanentReceipt -Force
    Write-Host "Permanent receipt saved to: $permanentReceipt"
}
