param(
    [Parameter(Mandatory = $true)]
    [string]$EvidenceDirectory,

    [Parameter(Mandatory = $true)]
    [string]$SourceCommit,

    [Parameter(Mandatory = $true)]
    [string]$RootStaticApproval
)

$ErrorActionPreference = 'Stop'

if ($SourceCommit -notmatch '^[0-9a-f]{40}$' -or $RootStaticApproval -cne "APPROVED $SourceCommit") {
    throw 'Supply the exact full source commit and root approval token: APPROVED <source-commit>.'
}

$projectRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path
$probeRoot = [IO.Path]::GetFullPath((Join-Path $projectRoot 'docs\reviews\probes'))
$probeRootPrefix = $probeRoot.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
$evidencePath = [IO.Path]::GetFullPath((Join-Path $projectRoot $EvidenceDirectory))
if (-not $evidencePath.StartsWith($probeRootPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'Evidence directory must stay inside docs/reviews/probes.'
}
if (Test-Path -LiteralPath $evidencePath) {
    throw 'Test campaign capture is single-use; evidence directory already exists.'
}

$sourcePaths = @(
    'backend/ai/provider.go',
    'backend/ai/openai_compatible.go',
    'backend/ai/claude.go',
    'backend/ai/gemini.go',
    'backend/ai/usage_presence.go',
    'backend/ai/usage_presence_test.go'
)
$head = (& git -C $projectRoot rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0 -or $head -notmatch '^[0-9a-f]{40}$') {
    throw 'Could not record current HEAD.'
}
& git -C $projectRoot merge-base --is-ancestor $SourceCommit $head
if ($LASTEXITCODE -ne 0) {
    throw 'Approved source commit is not an ancestor of current HEAD.'
}
$committedBackendAIChanges = @(& git -C $projectRoot diff --name-only $SourceCommit $head -- backend/ai)
if ($LASTEXITCODE -ne 0 -or $committedBackendAIChanges.Count -ne 0) {
    throw 'Committed backend/ai source changed after the approved source commit.'
}
foreach ($path in $sourcePaths) {
    $approvedBlob = (& git -C $projectRoot rev-parse "$SourceCommit`:$path").Trim()
    if ($LASTEXITCODE -ne 0) {
        throw "Could not resolve approved source blob for $path."
    }
    $headBlob = (& git -C $projectRoot rev-parse "HEAD`:$path").Trim()
    if ($LASTEXITCODE -ne 0 -or $approvedBlob -cne $headBlob) {
        throw "Current HEAD source differs from the approved commit: $path."
    }
}
& git -C $projectRoot diff --quiet HEAD -- $sourcePaths
if ($LASTEXITCODE -ne 0) {
    throw 'Authorized source paths have unstaged changes before the test campaign.'
}
& git -C $projectRoot diff --cached --quiet HEAD -- $sourcePaths
if ($LASTEXITCODE -ne 0) {
    throw 'Authorized source paths have staged changes before the test campaign.'
}
$backendAIChanges = @(& git -C $projectRoot diff --name-only HEAD -- backend/ai)
if ($LASTEXITCODE -ne 0 -or $backendAIChanges.Count -ne 0) {
    throw 'Any backend/ai working-tree change is forbidden before the test campaign.'
}
$stagedBackendAIChanges = @(& git -C $projectRoot diff --cached --name-only HEAD -- backend/ai)
if ($LASTEXITCODE -ne 0 -or $stagedBackendAIChanges.Count -ne 0) {
    throw 'Any staged backend/ai change is forbidden before the test campaign.'
}
$untrackedAI = @(& git -C $projectRoot ls-files --others --exclude-standard -- backend/ai)
if ($LASTEXITCODE -ne 0 -or $untrackedAI.Count -ne 0) {
    throw 'Unexpected untracked backend/ai paths exist before the test campaign.'
}

$null = New-Item -ItemType Directory -Path $evidencePath
$utf8 = New-Object System.Text.UTF8Encoding($false)
$sourceFile = Join-Path $projectRoot 'backend\ai\usage_presence.go'
$originalSource = [IO.File]::ReadAllBytes($sourceFile)
$originalSourceHash = [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData($originalSource)).ToLowerInvariant()
$originalSourceText = $utf8.GetString($originalSource)
$invocation = 0
$variableNames = @('GOPROXY', 'GOSUMDB', 'GOTOOLCHAIN', 'CGO_ENABLED')
$previousEnvironment = @{}
foreach ($name in $variableNames) {
    $previousEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}
$env:GOPROXY = 'off'
$env:GOSUMDB = 'off'
$env:GOTOOLCHAIN = 'local'
$env:CGO_ENABLED = '0'

function Write-JsonFile([string]$Path, [object]$Value) {
    [IO.File]::WriteAllText($Path, ($Value | ConvertTo-Json -Depth 16) + [Environment]::NewLine, $utf8)
}

function Get-Sha256([string]$Path) {
    (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
}

function Invoke-NativeCapture([string]$FilePath, [string[]]$Arguments, [string]$WorkingDirectory, [string]$StdoutPath, [string]$StderrPath) {
    $startInfo = [Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = $FilePath
    $startInfo.WorkingDirectory = $WorkingDirectory
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    foreach ($argument in $Arguments) {
        $null = $startInfo.ArgumentList.Add($argument)
    }

    $stdoutStream = [IO.File]::Open($StdoutPath, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
    $stderrStream = [IO.File]::Open($StderrPath, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
    $process = [Diagnostics.Process]::new()
    $process.StartInfo = $startInfo
    $started = $false
    $exitCode = $null
    $failure = $null
    try {
        $started = $process.Start()
        if (-not $started) {
            throw 'Process.Start returned false.'
        }
        $stdoutCopy = $process.StandardOutput.BaseStream.CopyToAsync($stdoutStream)
        $stderrCopy = $process.StandardError.BaseStream.CopyToAsync($stderrStream)
        $process.WaitForExit()
        [Threading.Tasks.Task]::WaitAll([Threading.Tasks.Task[]]@($stdoutCopy, $stderrCopy))
        $exitCode = [int]$process.ExitCode
    }
    catch {
        $failure = $_.Exception.Message
    }
    finally {
        $stdoutStream.Dispose()
        $stderrStream.Dispose()
        $process.Dispose()
    }

    [pscustomobject]@{
        processStarted = $started
        exitCode = $exitCode
        processStartFailure = $failure
        stdoutPath = $StdoutPath
        stderrPath = $StderrPath
        stdoutBytes = (Get-Item -LiteralPath $StdoutPath).Length
        stderrBytes = (Get-Item -LiteralPath $StderrPath).Length
        stdoutSha256 = Get-Sha256 $StdoutPath
        stderrSha256 = Get-Sha256 $StderrPath
    }
}

function Invoke-GoTest([string]$Name, [string[]]$Arguments) {
    $script:invocation++
    $stdoutPath = Join-Path $evidencePath "$Name.stdout.bin"
    $stderrPath = Join-Path $evidencePath "$Name.stderr.bin"
    $receiptPath = Join-Path $evidencePath "$Name.result.json"
    $startedAt = [DateTimeOffset]::UtcNow
    $timer = [Diagnostics.Stopwatch]::StartNew()
    $go = Get-Command go.exe -CommandType Application -ErrorAction Stop
    $capture = Invoke-NativeCapture $go.Source $Arguments $projectRoot $stdoutPath $stderrPath
    $timer.Stop()
    $receipt = [ordered]@{
        schemaVersion = '1.0'
        ordinal = $script:invocation
        sourceCommit = $SourceCommit
        currentHead = $head
        rootStaticApproval = $RootStaticApproval
        command = (@('go') + $Arguments) -join ' '
        environment = [ordered]@{ GOPROXY = 'off'; GOSUMDB = 'off'; GOTOOLCHAIN = 'local'; CGO_ENABLED = '0' }
        startUtc = $startedAt.ToString('o')
        durationMilliseconds = $timer.ElapsedMilliseconds
        processStarted = $capture.processStarted
        processStartFailure = $capture.processStartFailure
        exitCode = $capture.exitCode
        stdout = [ordered]@{ path = [IO.Path]::GetFileName($stdoutPath); bytes = $capture.stdoutBytes; sha256 = $capture.stdoutSha256 }
        stderr = [ordered]@{ path = [IO.Path]::GetFileName($stderrPath); bytes = $capture.stderrBytes; sha256 = $capture.stderrSha256 }
    }
    Write-JsonFile $receiptPath $receipt
    [pscustomobject]@{ Name = $Name; Receipt = $receipt; ReceiptPath = $receiptPath; Stdout = [IO.File]::ReadAllText($stdoutPath); Stderr = [IO.File]::ReadAllText($stderrPath) }
}

function Save-MutationDiff([string]$Name) {
    $diffPath = Join-Path $evidencePath "$Name.source-diff.patch"
    $stdoutPath = Join-Path $evidencePath "$Name.diff.stdout.bin"
    $stderrPath = Join-Path $evidencePath "$Name.diff.stderr.bin"
    $git = Get-Command git.exe -CommandType Application -ErrorAction Stop
    $capture = Invoke-NativeCapture $git.Source @('-C', $projectRoot, 'diff', '--no-ext-diff', '--binary', '--', 'backend/ai/usage_presence.go') $projectRoot $stdoutPath $stderrPath
    if (-not $capture.processStarted -or $capture.exitCode -ne 0) {
        throw "Could not capture exact $Name mutation source diff."
    }
    [IO.File]::Copy($stdoutPath, $diffPath, $true)
    $changed = @(& git -C $projectRoot diff --name-only -- backend/ai)
    if ($LASTEXITCODE -ne 0 -or $changed.Count -ne 1 -or $changed[0] -cne 'backend/ai/usage_presence.go') {
        throw "$Name mutation changed a path outside backend/ai/usage_presence.go."
    }
    [pscustomobject]@{ path = [IO.Path]::GetFileName($diffPath); sha256 = Get-Sha256 $diffPath; changedPaths = $changed }
}

function Invoke-Mutation([string]$Name, [string]$Before, [string]$After, [string[]]$Arguments, [string]$DetectorName, [string]$ControlName) {
    if ($originalSourceText.IndexOf($Before, [StringComparison]::Ordinal) -lt 0 -or $originalSourceText.IndexOf($Before, [StringComparison]::Ordinal) -ne $originalSourceText.LastIndexOf($Before, [StringComparison]::Ordinal)) {
        throw "$Name mutation anchor must occur exactly once in the original source."
    }
    if ($After -ceq $Before) {
        throw "$Name mutation replacement must change the source."
    }
    $mutatedText = $originalSourceText.Replace($Before, $After)
    try {
        [IO.File]::WriteAllText($sourceFile, $mutatedText, $utf8)
        $diff = Save-MutationDiff $Name
        $result = Invoke-GoTest "$Name-test" $Arguments
        $detectorFailed = ($result.Stdout + $result.Stderr).Contains("--- FAIL: $DetectorName")
        $controlPassed = ($result.Stdout + $result.Stderr).Contains("--- PASS: $ControlName")
        $expectedDetection = $result.Receipt.processStarted -and $result.Receipt.exitCode -ne 0 -and $detectorFailed -and $controlPassed
        $mutationReceipt = [ordered]@{
            schemaVersion = '1.0'
            mutation = $Name
            detector = $DetectorName
            healthyControl = $ControlName
            sourcePath = 'backend/ai/usage_presence.go'
            baselineSourceSha256 = $originalSourceHash
            mutatedSourceSha256 = Get-Sha256 $sourceFile
            diff = $diff
            detectorFailed = $detectorFailed
            healthyControlPassed = $controlPassed
            expectedDetection = $expectedDetection
            testInvocationOrdinal = $result.Receipt.ordinal
            testReceipt = [IO.Path]::GetFileName($result.ReceiptPath)
        }
        Write-JsonFile (Join-Path $evidencePath "$Name.mutation-result.json") $mutationReceipt
    }
    finally {
        [IO.File]::WriteAllBytes($sourceFile, $originalSource)
        $restoredHash = Get-Sha256 $sourceFile
        $restored = $restoredHash -ceq $originalSourceHash
        Write-JsonFile (Join-Path $evidencePath "$Name.restoration.json") ([ordered]@{
            schemaVersion = '1.0'
            sourcePath = 'backend/ai/usage_presence.go'
            baselineSha256 = $originalSourceHash
            restoredSha256 = $restoredHash
            byteForByteRestored = $restored
        })
        if (-not $restored) {
            throw "$Name mutation source restoration failed byte-for-byte."
        }
    }
    if (-not $expectedDetection) {
        throw "$Name mutation did not fail only its detector while its healthy control passed."
    }
}

function Save-CommittedSourceSnapshot([string]$Name) {
    $manifestPath = Join-Path $evidencePath "$Name.tree-manifest.txt"
    $archivePath = Join-Path $evidencePath "$Name.source.tar"
    $manifest = @(& git -C $projectRoot ls-tree -r --full-tree $SourceCommit)
    if ($LASTEXITCODE -ne 0) {
        throw "Could not capture $Name committed tree manifest."
    }
    [IO.File]::WriteAllText($manifestPath, ($manifest -join [Environment]::NewLine) + [Environment]::NewLine, $utf8)
    & git -C $projectRoot archive --format=tar "--output=$archivePath" $SourceCommit
    if ($LASTEXITCODE -ne 0) {
        throw "Could not capture $Name full source archive."
    }
    [pscustomobject]@{
        manifestPath = [IO.Path]::GetFileName($manifestPath)
        manifestEntries = $manifest.Count
        manifestSha256 = Get-Sha256 $manifestPath
        archivePath = [IO.Path]::GetFileName($archivePath)
        archiveBytes = (Get-Item -LiteralPath $archivePath).Length
        archiveSha256 = Get-Sha256 $archivePath
    }
}

try {
    [IO.File]::WriteAllBytes((Join-Path $evidencePath 'usage_presence.go.original.bin'), $originalSource)
    $archiveBefore = Save-CommittedSourceSnapshot 'before-campaign'
    $positive = Invoke-GoTest 'positive-ai-package' @('-C', 'backend', 'test', '-v', './ai', '-count=1')
    if (-not $positive.Receipt.processStarted -or $positive.Receipt.exitCode -ne 0) {
        throw 'Full AI package positive test did not pass; campaign stopped without retry.'
    }

    Invoke-Mutation 'M01' `
        'return AIUsageCount{Status: status}' `
        'if status == AIUsagePresenceAbsent || status == AIUsagePresenceNull { return knownAIUsageCount(0) }; return AIUsageCount{Status: status}' `
        @('-C', 'backend', 'test', '-v', './ai', '-run', '^TestUsagePresenceM01(AbsentNullDetector|ExplicitZeroHealthyControl)$', '-count=1') `
        'TestUsagePresenceM01AbsentNullDetector' `
        'TestUsagePresenceM01ExplicitZeroHealthyControl'

    Invoke-Mutation 'M02' `
        'return usageCountState(AIUsagePresenceInvalid)' `
        'return knownAIUsageCount(0)' `
        @('-C', 'backend', 'test', '-v', './ai', '-run', '^TestUsagePresenceM02(InvalidDetector|ValidIntegerHealthyControl)$', '-count=1') `
        'TestUsagePresenceM02InvalidDetector' `
        'TestUsagePresenceM02ValidIntegerHealthyControl'

    $restoredHash = Get-Sha256 $sourceFile
    if ($restoredHash -cne $originalSourceHash) {
        throw 'Source does not match the original bytes before restored tests.'
    }
    $archiveAfterRestore = Save-CommittedSourceSnapshot 'after-restoration'
    if ($archiveBefore.archiveSha256 -cne $archiveAfterRestore.archiveSha256 -or $archiveBefore.manifestSha256 -cne $archiveAfterRestore.manifestSha256) {
        throw 'Full committed source archive or manifest changed across mutation/restoration.'
    }

    $restored = Invoke-GoTest 'restored-new-tests' @('-C', 'backend', 'test', '-v', './ai', '-run', '^TestUsagePresence', '-count=1')
    if (-not $restored.Receipt.processStarted -or $restored.Receipt.exitCode -ne 0) {
        throw 'Restored usage-presence tests did not pass; campaign stopped without retry.'
    }
    $campaign = [ordered]@{
        schemaVersion = '1.0'
        sourceCommit = $SourceCommit
        currentHead = $head
        rootStaticApproval = $RootStaticApproval
        goInvocationCount = $script:invocation
        expectedGoInvocationCount = 4
        automaticRetry = $false
        sourcePath = 'backend/ai/usage_presence.go'
        originalSourceSha256 = $originalSourceHash
        restoredSourceSha256 = Get-Sha256 $sourceFile
        byteForByteRestored = ((Get-Sha256 $sourceFile) -ceq $originalSourceHash)
        beforeArchive = $archiveBefore
        afterRestoreArchive = $archiveAfterRestore
        completed = $true
    }
    Write-JsonFile (Join-Path $evidencePath 'campaign-result.json') $campaign
}
finally {
    foreach ($name in $variableNames) {
        [Environment]::SetEnvironmentVariable($name, $previousEnvironment[$name], 'Process')
    }
}

Write-Output "Completed four Go test invocations on $SourceCommit; full source restored byte-for-byte."
Write-Output "Evidence=$evidencePath"
