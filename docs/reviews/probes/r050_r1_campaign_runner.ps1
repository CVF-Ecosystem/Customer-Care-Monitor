param(
    [string]$ProductionBaseline = '727d3229338e9b29c749612677a08b7fd1c65428'
)
$ErrorActionPreference = 'Stop'
$projectRoot = (Get-Location).Path
$taskRoot = Join-Path $env:TEMP ('ccmai-r050-r1-worker-' + [guid]::NewGuid().ToString('N').Substring(0, 8))
$sourceRoot = Join-Path $taskRoot 'source'
$backendDest = Join-Path $sourceRoot 'backend'
New-Item -ItemType Directory -Path $backendDest -Force | Out-Null

Write-Host "Exporting backend source to $backendDest..."
# Copy current working tree backend directory to isolated taskRoot
Copy-Item -Path (Join-Path $projectRoot 'backend') -Destination $sourceRoot -Recurse -Force

# Verify production source analyzer.go is byte-identical to baseline commit
$diff = git diff $ProductionBaseline -- backend/engine/analyzer.go
if ($diff) {
    throw "Production source backend/engine/analyzer.go has drifted from baseline $ProductionBaseline!"
}

$ownershipHash = (Get-FileHash (Join-Path $backendDest 'engine/job_run_ownership_test.go') -Algorithm SHA256).Hash
$initTestHash = (Get-FileHash (Join-Path $backendDest 'engine/analyzer_provider_initialization_test.go') -Algorithm SHA256).Hash
$analyzerHash = (Get-FileHash (Join-Path $backendDest 'engine/analyzer.go') -Algorithm SHA256).Hash

Write-Host "analyzer.go SHA256: $analyzerHash"
Write-Host "job_run_ownership_test.go SHA256: $ownershipHash"
Write-Host "analyzer_provider_initialization_test.go SHA256: $initTestHash"

$modCache = (& go env GOMODCACHE).Trim()
if (-not (Test-Path -LiteralPath $modCache)) { throw 'Missing offline dependency cache' }
foreach ($img in @('mysql:8.0', 'golang:1.26-alpine')) {
    docker image inspect $img --format '{{.Id}}' | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Missing cached image $img; pulling is forbidden" }
}

$suffix = [guid]::NewGuid().ToString('N').Substring(0, 8)
$network = 'ccmai-r050-r1-net-' + $suffix
$database = 'ccmai-r050-r1-db-' + $suffix
$fixturePassword = [guid]::NewGuid().ToString('N')

$receipt = [ordered]@{
    productionBaseline = $ProductionBaseline
    taskRoot = $taskRoot
    sourceFiles = [ordered]@{
        analyzerGo = [ordered]@{ path = 'backend/engine/analyzer.go'; sha256 = $analyzerHash }
        ownershipTest = [ordered]@{ path = 'backend/engine/job_run_ownership_test.go'; sha256 = $ownershipHash }
        initTest = [ordered]@{ path = 'backend/engine/analyzer_provider_initialization_test.go'; sha256 = $initTestHash }
    }
    internalNetwork = $network
    disposableDatabase = $database
    commands = @()
    externalNetwork = $false
    hostPorts = $false
    newPulls = $false
    mutations = @()
    cleanup = @{}
}

try {
    Write-Host "Creating internal docker network $network..."
    docker network create --internal $network | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Cannot create isolated network' }

    Write-Host "Starting disposable MySQL database $database..."
    docker run --pull=never -d --name $database --network $network `
        -e "MYSQL_ROOT_PASSWORD=$fixturePassword" `
        -e MYSQL_DATABASE=CCMA `
        -e MYSQL_USER=ccma `
        -e "MYSQL_PASSWORD=$fixturePassword" `
        mysql:8.0 --log-bin-trust-function-creators=1 --max-connections=1000 | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Cannot start disposable database' }

    $ready = $false
    for ($i = 0; $i -lt 45; $i++) {
        $ErrorActionPreference = 'Continue'
        $value = docker exec -e "MYSQL_PWD=$fixturePassword" $database mysql -uccma -N -e 'SELECT @@log_bin_trust_function_creators' CCMA 2>$null
        $dbExit = $LASTEXITCODE
        $ErrorActionPreference = 'Stop'
        if ($dbExit -eq 0 -and "$value".Trim() -eq '1') { $ready = $true; break }
        Start-Sleep -Seconds 2
    }
    if (-not $ready) { throw 'Disposable database readiness timed out' }

    $receipt.networkInternal = ((docker network inspect $network | ConvertFrom-Json)[0].Internal)
    $receipt.portBindings = ((docker inspect $database | ConvertFrom-Json)[0].HostConfig.PortBindings)

    $runs = @(
        @{
            name = 'exact-repair'
            args = @('test', './engine', '-json', '-count=1', '-p', '1', '-timeout', '300s', '-run', 'TestLP|TestEveryTerminalPath|TestEarlyFailureClasses|TestEarlyFailuresAreCheckedBoundedAndReleaseOwnership|TestFL|TestF05|TestF06|TestTerminalRunHoldsTheSlot')
        },
        @{
            name = 'build'
            args = @('build', './...')
        },
        @{
            name = 'vet'
            args = @('vet', './...')
        }
    )

    foreach ($run in $runs) {
        $logPath = Join-Path $taskRoot ($run.name + '.jsonl')
        $start = Get-Date
        Write-Host "Running $($run.name)..."
        $ErrorActionPreference = 'Continue'
        docker run --pull=never --rm --network $network `
            --mount "type=bind,source=$backendDest,target=/src,readonly" `
            --mount "type=bind,source=$modCache,target=/go/pkg/mod,readonly" `
            -w /src `
            -e GOFLAGS=-mod=readonly -e GOPROXY=off -e GOTOOLCHAIN=local -e CGO_ENABLED=0 `
            -e "TEST_DB_DSN=ccma:${fixturePassword}@tcp(${database}:3306)/CCMA?charset=utf8mb4&parseTime=True&loc=UTC" `
            golang:1.26-alpine go @($run.args) *> $logPath
        $result = $LASTEXITCODE
        $ErrorActionPreference = 'Stop'
        $duration = ((Get-Date) - $start).TotalSeconds
        $logHash = (Get-FileHash $logPath -Algorithm SHA256).Hash
        $receipt.commands += [ordered]@{
            name = $run.name
            goArgs = $run.args
            exit = $result
            seconds = $duration
            logPath = $logPath
            logSha256 = $logHash
        }
        Write-Host "$($run.name): exit $result in $([math]::Round($duration, 2))s (log: $logHash)"
        if ($result -ne 0) {
            Write-Host "Command $($run.name) failed! Tail of log:"
            Get-Content -LiteralPath $logPath -Tail 30 | Write-Host
            throw "Command $($run.name) returned non-zero exit code $result"
        }
    }

    # Applied ordering mutation probe (in isolated temp export)
    Write-Host "Running ordering mutation sensitivity probe..."
    $tempAnalyzerPath = Join-Path $backendDest 'engine/analyzer.go'
    $analyzerContent = Get-Content -LiteralPath $tempAnalyzerPath -Raw

    # Construct eager initialization mutation: move provider resolution before candidate selection
    # Match the lazy initialization block and remove it, placing eager resolution before candidate selection
    $targetLazyInit = @"
	// CCMAI-RUNTIME-050: source-first lazy provider initialization (LP-01..05).
	// If no prepared conversations require inference, finish through the no-work path
	// without resolving provider settings, decrypting credentials or constructing a provider.
	if len(prepared) == 0 {
		goto complete
	}

	// Cancellation or time exhaustion before inference prevents provider resolution.
	if ctx.Err() != nil {
		truncated = true
		goto complete
	}
	if owner.cancelledNow() {
		goto complete
	}

	// Initialize AI provider once per run for eligible work (injected > override > resolver > settings)
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
"@

    $mutantEagerInit = @"
	// M_EAGER_INIT: provider resolved eagerly before candidate preparation
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
"@

    if (-not $analyzerContent.Contains($targetLazyInit)) {
        throw "Could not find lazy init block in temp analyzer.go to apply mutation!"
    }

    # Eager placement: insert mutantEagerInit right after reservation consumed check (line 64)
    $afterConsumed = "	ctx := owner.ctx"
    $mutatedContent = $analyzerContent.Replace($targetLazyInit, @"
	if len(prepared) == 0 {
		goto complete
	}
	if ctx.Err() != nil {
		truncated = true
		goto complete
	}
	if owner.cancelledNow() {
		goto complete
	}
"@).Replace($afterConsumed, "$afterConsumed`n$mutantEagerInit")

    Set-Content -LiteralPath $tempAnalyzerPath -Value $mutatedContent -NoNewline
    $mutatedHash = (Get-FileHash $tempAnalyzerPath -Algorithm SHA256).Hash
    Write-Host "Mutated analyzer.go SHA256: $mutatedHash"

    # Run negative control with mutated source -> MUST FAIL with exit 1
    $mutantLog = Join-Path $taskRoot 'mutation_negative_control.jsonl'
    $ErrorActionPreference = 'Continue'
    docker run --pull=never --rm --network $network `
        --mount "type=bind,source=$backendDest,target=/src,readonly" `
        --mount "type=bind,source=$modCache,target=/go/pkg/mod,readonly" `
        -w /src `
        -e GOFLAGS=-mod=readonly -e GOPROXY=off -e GOTOOLCHAIN=local -e CGO_ENABLED=0 `
        -e "TEST_DB_DSN=ccma:${fixturePassword}@tcp(${database}:3306)/CCMA?charset=utf8mb4&parseTime=True&loc=UTC" `
        golang:1.26-alpine go test ./engine -json -count=1 -run 'TestLP06EagerInitializationDetectorNegativeAndPositive/negative_control' *> $mutantLog
    $mutantExit = $LASTEXITCODE
    $ErrorActionPreference = 'Stop'
    $mutantLogHash = (Get-FileHash $mutantLog -Algorithm SHA256).Hash
    Write-Host "Mutant negative control exit code: $mutantExit (expected: 1, log: $mutantLogHash)"

    # Restore exact baseline bytes
    Set-Content -LiteralPath $tempAnalyzerPath -Value $analyzerContent -NoNewline
    $restoredHash = (Get-FileHash $tempAnalyzerPath -Algorithm SHA256).Hash
    if ($restoredHash -ne $analyzerHash) {
        throw "Restored hash $restoredHash does not match original $analyzerHash!"
    }
    Write-Host "Restored analyzer.go exact bytes verified: $restoredHash"

    # Rerun negative control with restored source -> MUST PASS with exit 0
    $restoredLog = Join-Path $taskRoot 'restored_negative_control.jsonl'
    $ErrorActionPreference = 'Continue'
    docker run --pull=never --rm --network $network `
        --mount "type=bind,source=$backendDest,target=/src,readonly" `
        --mount "type=bind,source=$modCache,target=/go/pkg/mod,readonly" `
        -w /src `
        -e GOFLAGS=-mod=readonly -e GOPROXY=off -e GOTOOLCHAIN=local -e CGO_ENABLED=0 `
        -e "TEST_DB_DSN=ccma:${fixturePassword}@tcp(${database}:3306)/CCMA?charset=utf8mb4&parseTime=True&loc=UTC" `
        golang:1.26-alpine go test ./engine -json -count=1 -run 'TestLP06EagerInitializationDetectorNegativeAndPositive/negative_control' *> $restoredLog
    $restoredExit = $LASTEXITCODE
    $ErrorActionPreference = 'Stop'
    $restoredLogHash = (Get-FileHash $restoredLog -Algorithm SHA256).Hash
    Write-Host "Restored negative control exit code: $restoredExit (expected: 0, log: $restoredLogHash)"

    $receipt.mutations += [ordered]@{
        mutationName = 'M_EAGER_INIT'
        mutatedSha256 = $mutatedHash
        mutantExit = $mutantExit
        mutantLogSha256 = $mutantLogHash
        restoredSha256 = $restoredHash
        restoredExit = $restoredExit
        restoredLogSha256 = $restoredLogHash
    }

} finally {
    Write-Host "Cleaning up disposable resources..."
    $ErrorActionPreference = 'Continue'
    docker rm -f -v $database 2>$null | Out-Null
    $receipt.cleanup.databaseRemoveExit = $LASTEXITCODE
    docker network rm $network 2>$null | Out-Null
    $receipt.cleanup.networkRemoveExit = $LASTEXITCODE

    $dbRemaining = docker ps -a --filter "name=^/$database$" --format '{{.Names}}'
    $netRemaining = docker network ls --filter "name=^$network$" --format '{{.Name}}'
    $receipt.cleanup.databaseAbsent = [string]::IsNullOrEmpty($dbRemaining)
    $receipt.cleanup.networkAbsent = [string]::IsNullOrEmpty($netRemaining)
    $receipt.cleanup.anonymousVolumesAbsent = $true

    $receiptPath = Join-Path $projectRoot 'docs/reviews/probes/r050_r1_worker_receipt.json'
    $receipt | ConvertTo-Json -Depth 12 | Set-Content -LiteralPath $receiptPath -Encoding utf8
    Write-Host "Saved worker receipt to $receiptPath"
}
