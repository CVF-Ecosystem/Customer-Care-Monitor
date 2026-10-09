param(
    [Parameter(Mandatory = $true)]
    [string]$EvidenceDirectory,
    [Parameter(Mandatory = $true)]
    [string]$SourceCommit,
    [Parameter(Mandatory = $true)]
    [string]$RootStaticApproval,
    [ValidateRange(60, 1800)]
    [int]$ProcessTimeoutSeconds = 600
)

$ErrorActionPreference = 'Stop'
$script:invocationCount = 0
$script:runReceipts = [System.Collections.Generic.List[object]]::new()
$script:failureReason = $null
$script:campaignCompleted = $false
$script:ownedTempPath = $null
$script:ownershipToken = $null
$script:backendRoot = $null
$script:usageSourcePath = $null
$script:originalUsageBytes = $null
$script:originalUsageSha256 = $null
$script:manifestBefore = $null
$script:manifestAfter = $null
$script:physicalArchiveBefore = $null
$script:physicalArchiveAfter = $null
$script:cleanupStatus = 'NOT_STARTED'
$script:finalizationErrors = [System.Collections.Generic.List[string]]::new()
$script:utf8 = New-Object System.Text.UTF8Encoding($false)
$script:previousEnvironment = $null
$script:variableNames = @('GOPROXY', 'GOSUMDB', 'GOTOOLCHAIN', 'CGO_ENABLED')
$script:markerPath = $null
$script:head = $null
$script:sourceArchivePath = $null
$script:sourceArchiveAfterPath = $null
$script:sourceArchiveSha256 = $null
$script:sourceArchiveBytes = $null
$script:sourceArchiveRun = $null
$script:sourceArchiveAfterRun = $null
$script:goExecutable = $null
$script:goResolutionCandidates = @()
$script:gitExecutable = $null
$script:tarExecutable = $null
$script:restoredPhysicalSource = $false
$script:finalSourceSha256 = $null
$script:committedArchiveAfter = $null
$script:rawBackupPath = $null
$script:rawBackupReceipt = $null
$script:rawBackupMarkerPath = $null
$script:rawBackupOwnershipToken = $null
$script:summaryPath = $null
$script:summaryObject = $null

function Write-JsonFile([string]$Path, [object]$Value) {
    [IO.File]::WriteAllText($Path, ($Value | ConvertTo-Json -Depth 20) + [Environment]::NewLine, $script:utf8)
}

function Write-JsonFileExclusive([string]$Path, [object]$Value) {
    $bytes = $script:utf8.GetBytes(($Value | ConvertTo-Json -Depth 20) + [Environment]::NewLine)
    $stream = [IO.File]::Open($Path, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
    try { $stream.Write($bytes, 0, $bytes.Length) }
    finally { $stream.Dispose() }
}

function Get-Sha256([string]$Path) {
    (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
}

function Get-ByteSha256([byte[]]$Bytes) {
    [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData($Bytes)).ToLowerInvariant()
}

function Invoke-NativeCapture(
    [string]$FilePath,
    [string[]]$Arguments,
    [string]$WorkingDirectory,
    [string]$StdoutPath,
    [string]$StderrPath,
    [int]$TimeoutSeconds = 120
) {
    [IO.File]::WriteAllBytes($StdoutPath, [byte[]]@())
    [IO.File]::WriteAllBytes($StderrPath, [byte[]]@())
    $startInfo = [Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = $FilePath
    $startInfo.WorkingDirectory = $WorkingDirectory
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    foreach ($argument in $Arguments) { $null = $startInfo.ArgumentList.Add($argument) }

    $stdoutStream = [IO.File]::Open($StdoutPath, [IO.FileMode]::Create, [IO.FileAccess]::Write, [IO.FileShare]::None)
    $stderrStream = [IO.File]::Open($StderrPath, [IO.FileMode]::Create, [IO.FileAccess]::Write, [IO.FileShare]::None)
    $process = [Diagnostics.Process]::new()
    $process.StartInfo = $startInfo
    $started = $false
    $timedOut = $false
    $exitCode = $null
    $failure = $null
    $stdoutCopy = $null
    $stderrCopy = $null
    try {
        $started = $process.Start()
        if (-not $started) { throw 'Process.Start returned false.' }
        $stdoutCopy = $process.StandardOutput.BaseStream.CopyToAsync($stdoutStream)
        $stderrCopy = $process.StandardError.BaseStream.CopyToAsync($stderrStream)
        if (-not $process.WaitForExit($TimeoutSeconds * 1000)) {
            $timedOut = $true
            $process.Kill($true)
            $process.WaitForExit()
        }
        [Threading.Tasks.Task]::WaitAll([Threading.Tasks.Task[]]@($stdoutCopy, $stderrCopy))
        $exitCode = [int]$process.ExitCode
    }
    catch {
        $failure = $_.Exception.Message
        if ($started -and -not $process.HasExited) {
            try { $process.Kill($true); $process.WaitForExit() }
            catch { $failure += '; owned-process cleanup failed: ' + $_.Exception.Message }
        }
        if ($null -ne $stdoutCopy -and $null -ne $stderrCopy) {
            try { [Threading.Tasks.Task]::WaitAll([Threading.Tasks.Task[]]@($stdoutCopy, $stderrCopy)) }
            catch { $failure += '; stream-drain failure: ' + $_.Exception.Message }
        }
    }
    finally {
        $stdoutStream.Dispose()
        $stderrStream.Dispose()
        $process.Dispose()
    }

    [pscustomobject]@{
        processStarted = $started
        timedOut = $timedOut
        exitCode = $exitCode
        failure = $failure
        stdoutPath = $StdoutPath
        stderrPath = $StderrPath
        stdoutBytes = (Get-Item -LiteralPath $StdoutPath).Length
        stderrBytes = (Get-Item -LiteralPath $StderrPath).Length
        stdoutSha256 = Get-Sha256 $StdoutPath
        stderrSha256 = Get-Sha256 $StderrPath
    }
}

function Get-PhysicalManifest([string]$Root, [string]$OutputPath) {
    $rootPath = [IO.Path]::GetFullPath($Root)
    $rootPrefix = $rootPath.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
    $relativePaths = [System.Collections.Generic.List[string]]::new()
    foreach ($file in Get-ChildItem -LiteralPath $rootPath -File -Recurse -Force) {
        if (($file.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) { throw "Reparse file is not allowed in the backend archive: $($file.FullName)" }
        $resolved = [IO.Path]::GetFullPath($file.FullName)
        if (-not $resolved.StartsWith($rootPrefix, [StringComparison]::OrdinalIgnoreCase)) { throw 'A backend member escaped the extracted root.' }
        $relativePaths.Add([IO.Path]::GetRelativePath($rootPath, $resolved).Replace('\', '/'))
    }
    $relativePaths.Sort([StringComparer]::Ordinal)
    $entries = [System.Collections.Generic.List[object]]::new()
    foreach ($relative in $relativePaths) {
        $physical = Join-Path $rootPath $relative.Replace('/', [IO.Path]::DirectorySeparatorChar)
        $entries.Add([ordered]@{ path = $relative; bytes = (Get-Item -LiteralPath $physical).Length; sha256 = Get-Sha256 $physical })
    }
    $manifest = [ordered]@{ schemaVersion = '1.0'; members = $entries.ToArray() }
    Write-JsonFile $OutputPath $manifest
    [pscustomobject]@{ path = $OutputPath; memberCount = $entries.Count; sha256 = Get-Sha256 $OutputPath; entries = $entries.ToArray() }
}

function Export-PhysicalZip([string]$Root, [string]$ArchivePath) {
    $rootPath = [IO.Path]::GetFullPath($Root)
    $rootPrefix = $rootPath.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
    $paths = [System.Collections.Generic.List[string]]::new()
    foreach ($file in Get-ChildItem -LiteralPath $rootPath -File -Recurse -Force) {
        $resolved = [IO.Path]::GetFullPath($file.FullName)
        if (-not $resolved.StartsWith($rootPrefix, [StringComparison]::OrdinalIgnoreCase)) { throw 'A physical archive member escaped the backend root.' }
        $paths.Add([IO.Path]::GetRelativePath($rootPath, $resolved).Replace('\', '/'))
    }
    $paths.Sort([StringComparer]::Ordinal)
    $stream = [IO.File]::Open($ArchivePath, [IO.FileMode]::CreateNew, [IO.FileAccess]::ReadWrite, [IO.FileShare]::None)
    try {
        $zip = [IO.Compression.ZipArchive]::new($stream, [IO.Compression.ZipArchiveMode]::Create, $false)
        try {
            foreach ($relative in $paths) {
                $entry = $zip.CreateEntry($relative, [IO.Compression.CompressionLevel]::NoCompression)
                $entry.LastWriteTime = [DateTimeOffset]::new(1980, 1, 1, 0, 0, 0, [TimeSpan]::Zero)
                $physical = Join-Path $rootPath $relative.Replace('/', [IO.Path]::DirectorySeparatorChar)
                $input = [IO.File]::OpenRead($physical)
                try {
                    $output = $entry.Open()
                    try { $input.CopyTo($output) } finally { $output.Dispose() }
                }
                finally { $input.Dispose() }
            }
        }
        finally { $zip.Dispose() }
    }
    finally { $stream.Dispose() }
    [pscustomobject]@{ path = $ArchivePath; bytes = (Get-Item -LiteralPath $ArchivePath).Length; sha256 = Get-Sha256 $ArchivePath; members = $paths.Count }
}

function Get-TopLevelTestInventory([string]$AIPath, [string]$OutputPath) {
    $files = @(Get-ChildItem -LiteralPath $AIPath -File -Filter '*_test.go' | Sort-Object Name)
    $items = [System.Collections.Generic.List[object]]::new()
    foreach ($file in $files) {
        $text = [IO.File]::ReadAllText($file.FullName)
        $matches = [regex]::Matches($text, '(?m)^func\s+(Test[A-Za-z0-9_]*)\s*\(')
        foreach ($match in $matches) {
            $items.Add([ordered]@{ name = $match.Groups[1].Value; file = 'backend/ai/' + $file.Name; fileSha256 = Get-Sha256 $file.FullName })
        }
    }
    $names = @($items | ForEach-Object { $_.name })
    if (($names | Sort-Object -Unique).Count -ne $names.Count) { throw 'Top-level test function names are not unique in the source inventory.' }
    $expectedNew = @(
        'TestUsagePresencePositive',
        'TestUsagePresenceM01AbsentNullDetector',
        'TestUsagePresenceM01ExplicitZeroHealthyControl',
        'TestUsagePresenceM02InvalidDetector',
        'TestUsagePresenceM02ValidIntegerHealthyControl',
        'TestUsagePresenceDuplicateRules',
        'TestUsagePresenceRawBoundsAndMalformedJSON',
        'TestUsagePresenceSDKAdapters'
    )
    foreach ($name in $expectedNew) {
        if (@($names | Where-Object { $_ -ceq $name }).Count -ne 1) { throw "Required new top-level test is missing or duplicated: $name" }
    }
    if ($names.Count -ne 18) { throw "Expected exactly 18 top-level ai tests (8 new plus 10 unchanged); source has $($names.Count)." }
    $inventory = [ordered]@{
        schemaVersion = '1.0'
        package = 'backend/ai'
        topLevelTestCount = $names.Count
        newTestNames = $expectedNew
        existingTestNames = @($items | Where-Object { $expectedNew -cnotcontains $_.name } | ForEach-Object { $_.name })
        noSkipExpected = $true
        tests = $items.ToArray()
    }
    Write-JsonFile $OutputPath $inventory
    [pscustomobject]@{ path = $OutputPath; sha256 = Get-Sha256 $OutputPath; total = $names.Count; new = $expectedNew.Count; existing = $names.Count - $expectedNew.Count; names = $names; newNames = $expectedNew }
}

function Read-GoJSONEvents([string]$StdoutPath) {
    $events = [System.Collections.Generic.List[object]]::new()
    foreach ($line in [IO.File]::ReadAllLines($StdoutPath, [Text.Encoding]::UTF8)) {
        if ([string]::IsNullOrWhiteSpace($line)) { continue }
        try { $events.Add(($line | ConvertFrom-Json -ErrorAction Stop)) }
        catch { throw "Non-JSON line in go test -json output: $($_.Exception.Message)" }
    }
    $events.ToArray()
}

function Invoke-GoTest([string]$Name, [string[]]$Arguments, [object]$ExpectedManifest = $null) {
    if ($null -eq $ExpectedManifest) { $ExpectedManifest = $script:manifestBefore }
    $manifestBeforePath = Join-Path $script:evidencePath "$Name.backend-manifest-before.json"
    $manifestAfterPath = Join-Path $script:evidencePath "$Name.backend-manifest-after.json"
    $manifestBefore = Get-PhysicalManifest $script:backendRoot $manifestBeforePath
    if (-not (Compare-ManifestEntries $ExpectedManifest.entries $manifestBefore.entries)) {
        throw "$Name backend does not match its expected pre-test physical manifest; Go was not invoked."
    }
    $script:invocationCount++
    $stdoutPath = Join-Path $script:evidencePath "$Name.stdout.jsonl.bin"
    $stderrPath = Join-Path $script:evidencePath "$Name.stderr.bin"
    $receiptPath = Join-Path $script:evidencePath "$Name.result.json"
    $startedAt = [DateTimeOffset]::UtcNow
    $timer = [Diagnostics.Stopwatch]::StartNew()
    $capture = Invoke-NativeCapture $script:goExecutable $Arguments $script:extractRoot $stdoutPath $stderrPath $ProcessTimeoutSeconds
    $timer.Stop()
    $manifestAfter = $null
    $manifestCaptureFailure = $null
    try { $manifestAfter = Get-PhysicalManifest $script:backendRoot $manifestAfterPath }
    catch { $manifestCaptureFailure = $_.Exception.Message }
    $manifestMatchesExpected = $null -ne $manifestAfter -and (Compare-ManifestEntries $ExpectedManifest.entries $manifestAfter.entries)
    $events = @()
    $parseFailure = $null
    try { $events = @(Read-GoJSONEvents $stdoutPath) }
    catch { $parseFailure = $_.Exception.Message }
    $receipt = [ordered]@{
        schemaVersion = '1.0'
        ordinal = $script:invocationCount
        sourceCommit = $SourceCommit
        sourceArchiveSha256 = $script:sourceArchiveSha256
        goExecutable = $script:goExecutable
        goResolutionCandidates = $script:goResolutionCandidates
        command = (@('go') + $Arguments) -join ' '
        environment = [ordered]@{ GOPROXY = 'off'; GOSUMDB = 'off'; GOTOOLCHAIN = 'local'; CGO_ENABLED = '0' }
        startUtc = $startedAt.ToString('o')
        durationMilliseconds = $timer.ElapsedMilliseconds
        processTimeoutSeconds = $ProcessTimeoutSeconds
        processStarted = $capture.processStarted
        processTimedOut = $capture.timedOut
        processFailure = $capture.failure
        exitCode = $capture.exitCode
        physicalManifestBefore = [ordered]@{ path = [IO.Path]::GetFileName($manifestBeforePath); sha256 = $manifestBefore.sha256; members = $manifestBefore.memberCount }
        physicalManifestAfter = if ($null -ne $manifestAfter) { [ordered]@{ path = [IO.Path]::GetFileName($manifestAfterPath); sha256 = $manifestAfter.sha256; members = $manifestAfter.memberCount } } else { $null }
        physicalManifestMatchesExpected = $manifestMatchesExpected
        physicalManifestCaptureFailure = $manifestCaptureFailure
        jsonEventCount = $events.Count
        jsonParseFailure = $parseFailure
        stdout = [ordered]@{ path = [IO.Path]::GetFileName($stdoutPath); bytes = $capture.stdoutBytes; sha256 = $capture.stdoutSha256 }
        stderr = [ordered]@{ path = [IO.Path]::GetFileName($stderrPath); bytes = $capture.stderrBytes; sha256 = $capture.stderrSha256 }
    }
    Write-JsonFile $receiptPath $receipt
    $script:runReceipts.Add([pscustomobject]@{ name = $Name; receiptPath = [IO.Path]::GetFileName($receiptPath); ordinal = $script:invocationCount; exitCode = $capture.exitCode; timedOut = $capture.timedOut; jsonEventCount = $events.Count; jsonParseFailure = $parseFailure })
    if ($null -ne $manifestCaptureFailure) { throw "$Name post-test backend manifest failed: $manifestCaptureFailure" }
    if (-not $manifestMatchesExpected) { throw "$Name changed backend physical bytes during Go test." }
    [pscustomobject]@{ name = $Name; receipt = $receipt; events = $events; stderr = [IO.File]::ReadAllText($stderrPath, [Text.Encoding]::UTF8); stdoutPath = $stdoutPath; stderrPath = $stderrPath }
}

function Assert-NoUnexpectedEvents([object[]]$Events, [string]$StderrText) {
    $skips = @($Events | Where-Object { $_.Action -ceq 'skip' })
    if ($skips.Count -gt 0) { throw 'A Go test emitted a skip event; this campaign permits no skips.' }
    $output = (($Events | Where-Object { $_.Action -ceq 'output' } | ForEach-Object { [string]$_.Output }) -join '') + $StderrText
    if ($output -match '(?i)panic:|test timed out|timed out after|build failed|failed to build|no required module|cannot find module') {
        throw 'Go output contains a panic, timeout or build-failure diagnostic; it is not a semantic detector kill.'
    }
}

function Assert-HealthyPositive([object]$Run, [string[]]$RequiredNames) {
    if (-not $Run.receipt.processStarted -or $Run.receipt.processTimedOut -or $Run.receipt.exitCode -ne 0 -or $Run.receipt.jsonParseFailure) {
        throw 'Full ai positive did not complete successfully; no retry is allowed.'
    }
    Assert-NoUnexpectedEvents $Run.events $Run.stderr
    foreach ($name in $RequiredNames) {
        if (-not @($Run.events | Where-Object { $_.Test -ceq $name -and $_.Action -ceq 'run' }).Count) { throw "Inventory test did not start: $name" }
        if (-not @($Run.events | Where-Object { $_.Test -ceq $name -and $_.Action -ceq 'pass' }).Count) { throw "Inventory test did not pass: $name" }
        if (@($Run.events | Where-Object { $_.Test -ceq $name -and $_.Action -in @('fail', 'skip') }).Count) { throw "Inventory test failed or skipped: $name" }
    }
}

function Assert-NewTestsPositive([object]$Run, [string[]]$RequiredNames) {
    Assert-HealthyPositive $Run $RequiredNames
}

function Assert-ExpectedMutationKill([object]$Run, [string]$Detector, [string]$Control) {
    if (-not $Run.receipt.processStarted -or $Run.receipt.processTimedOut -or $Run.receipt.exitCode -ne 1 -or $Run.receipt.jsonParseFailure) {
        throw "$Detector was not a completed test assertion failure with exit 1; build/panic/timeout/launch failures do not count."
    }
    Assert-NoUnexpectedEvents $Run.events $Run.stderr
    if (-not @($Run.events | Where-Object { $_.Test -ceq $Detector -and $_.Action -ceq 'run' }).Count) { throw "Detector did not start: $Detector" }
    if (-not @($Run.events | Where-Object { $_.Test -ceq $Control -and $_.Action -ceq 'run' }).Count) { throw "Healthy control did not start: $Control" }
    if (-not @($Run.events | Where-Object { $_.Test -ceq $Detector -and $_.Action -ceq 'fail' }).Count) { throw "Named top-level detector did not fail: $Detector" }
    if (-not @($Run.events | Where-Object { $_.Test -ceq $Control -and $_.Action -ceq 'pass' }).Count) { throw "Named healthy control did not pass: $Control" }
    $terminal = @($Run.events | Where-Object { $_.Test -and $_.Action -in @('pass', 'fail', 'skip') })
    foreach ($event in $terminal) {
        $name = [string]$event.Test
        if ($name -cne $Detector -and -not $name.StartsWith($Detector + '/', [StringComparison]::Ordinal) -and $name -cne $Control) {
            throw "Unexpected selected test outcome outside detector/control: $name ($($event.Action))."
        }
    }
    $detectorFailures = @($terminal | Where-Object { $_.Action -ceq 'fail' -and ($_.Test -ceq $Detector -or ([string]$_.Test).StartsWith($Detector + '/', [StringComparison]::Ordinal)) })
    if ($detectorFailures.Count -lt 2) { throw 'Detector did not fail with at least one named child-subtest assertion.' }
}

function Save-MutationDiff([string]$Name, [string]$BaselinePath, [string]$MutatedPath) {
    $stdoutPath = Join-Path $script:evidencePath "$Name.diff.stdout.bin"
    $stderrPath = Join-Path $script:evidencePath "$Name.diff.stderr.bin"
    $gitCapture = Invoke-NativeCapture $script:gitExecutable @('diff', '--no-index', '--binary', '--', $BaselinePath, $MutatedPath) $script:projectRoot $stdoutPath $stderrPath 120
    if (-not $gitCapture.processStarted -or $gitCapture.timedOut -or $gitCapture.exitCode -ne 1 -or $gitCapture.stdoutBytes -eq 0) {
        throw "$Name expected a one-file source diff (git diff --no-index exit 1)."
    }
    $patchPath = Join-Path $script:evidencePath "$Name.source-diff.patch"
    [IO.File]::Copy($stdoutPath, $patchPath, $true)
    [pscustomobject]@{ path = [IO.Path]::GetFileName($patchPath); bytes = (Get-Item -LiteralPath $patchPath).Length; sha256 = Get-Sha256 $patchPath; onlySourcePath = 'backend/ai/usage_presence.go' }
}

function Invoke-Mutation([string]$Name, [string]$Before, [string]$After, [string[]]$Arguments, [string]$Detector, [string]$Control, [string]$BaselinePath) {
    $beforeManifestPath = Join-Path $script:evidencePath "$Name.backend-manifest-before-mutation.json"
    $baselineNow = Get-PhysicalManifest $script:backendRoot $beforeManifestPath
    if (-not (Compare-ManifestEntries $script:manifestBefore.entries $baselineNow.entries)) {
        throw "$Name baseline backend changed before mutation."
    }
    $beforeText = $script:utf8.GetString($script:originalUsageBytes)
    if ($beforeText.IndexOf($Before, [StringComparison]::Ordinal) -lt 0 -or $beforeText.IndexOf($Before, [StringComparison]::Ordinal) -ne $beforeText.LastIndexOf($Before, [StringComparison]::Ordinal)) {
        throw "$Name mutation anchor must occur exactly once in the committed source."
    }
    if ($After -ceq $Before) { throw "$Name replacement does not change source bytes." }
    $mutatedText = $beforeText.Replace($Before, $After)
    $mutationExpected = $false
    $diff = $null
    $run = $null
    $mutatedHash = $null
    $mutatedManifest = $null
    $changedMembers = @()
    try {
        [IO.File]::WriteAllText($script:usageSourcePath, $mutatedText, $script:utf8)
        $mutatedHash = Get-Sha256 $script:usageSourcePath
        if ($mutatedHash -ceq $script:originalUsageSha256) { throw "$Name mutation did not change physical source bytes." }
        $mutatedManifest = Get-PhysicalManifest $script:backendRoot (Join-Path $script:evidencePath "$Name.backend-manifest-mutated.json")
        $changedMembers = @(Assert-OnlyUsageSourceChanged $script:manifestBefore $mutatedManifest $Name)
        $diff = Save-MutationDiff $Name $BaselinePath $script:usageSourcePath
        $run = Invoke-GoTest "$Name-test" $Arguments $mutatedManifest
        Assert-ExpectedMutationKill $run $Detector $Control
        $mutationExpected = $true
    }
    finally {
        [IO.File]::WriteAllBytes($script:usageSourcePath, $script:originalUsageBytes)
        $restoredHash = Get-Sha256 $script:usageSourcePath
        $restored = $restoredHash -ceq $script:originalUsageSha256
        $restoredManifest = Get-PhysicalManifest $script:backendRoot (Join-Path $script:evidencePath "$Name.backend-manifest-after-restoration.json")
        $backendRestored = Compare-ManifestEntries $script:manifestBefore.entries $restoredManifest.entries
        Write-JsonFile (Join-Path $script:evidencePath "$Name.restoration.json") ([ordered]@{
            schemaVersion = '1.0'
            sourcePath = 'backend/ai/usage_presence.go'
            baselineSha256 = $script:originalUsageSha256
            restoredSha256 = $restoredHash
            byteForByteRestored = $restored
            backendManifestAfterRestoration = [ordered]@{ path = [IO.Path]::GetFileName($restoredManifest.path); sha256 = $restoredManifest.sha256; members = $restoredManifest.memberCount }
            fullBackendBaselineRestored = $backendRestored
        })
        if (-not $restored -or -not $backendRestored) { throw "$Name source or full-backend restoration failed." }
        Write-JsonFile (Join-Path $script:evidencePath "$Name.mutation-result.json") ([ordered]@{
            schemaVersion = '1.0'
            mutation = $Name
            detector = $Detector
            healthyControl = $Control
            sourcePath = 'backend/ai/usage_presence.go'
            baselineSourceSha256 = $script:originalUsageSha256
            mutatedSourceSha256 = $mutatedHash
            changedBackendMembers = $changedMembers
            testFilesUnchanged = (@($changedMembers | Where-Object { $_ -match '(^|/)[^/]*_test\.go$' }).Count -eq 0)
            diff = $diff
            detectorExpectedKillAndControlPass = $mutationExpected
            detectorProcessExitCode = if ($null -ne $run) { $run.receipt.exitCode } else { $null }
            detectorRun = if ($null -ne $run) { $run.receiptPath } else { $null }
        })
    }
    if (-not $mutationExpected) { throw "$Name was not a named semantic detector kill with a passing healthy control." }
}

function Compare-ManifestEntries([object[]]$Before, [object[]]$After) {
    if ($Before.Count -ne $After.Count) { return $false }
    for ($i = 0; $i -lt $Before.Count; $i++) {
        if ($Before[$i].path -cne $After[$i].path -or $Before[$i].bytes -ne $After[$i].bytes -or $Before[$i].sha256 -cne $After[$i].sha256) { return $false }
    }
    $true
}

function Assert-OnlyUsageSourceChanged([object]$Baseline, [object]$Mutated, [string]$Name) {
    if ($Baseline.entries.Count -ne $Mutated.entries.Count) { throw "$Name changed the backend member count." }
    $changed = [System.Collections.Generic.List[string]]::new()
    for ($i = 0; $i -lt $Baseline.entries.Count; $i++) {
        $before = $Baseline.entries[$i]
        $after = $Mutated.entries[$i]
        if ($before.path -cne $after.path) { throw "$Name changed backend member ordering or names." }
        if ($before.bytes -ne $after.bytes -or $before.sha256 -cne $after.sha256) { $changed.Add([string]$before.path) }
    }
    if ($changed.Count -ne 1 -or $changed[0] -cne 'ai/usage_presence.go') {
        throw "$Name must change exactly backend/ai/usage_presence.go; changed members: $($changed -join ', ')."
    }
    if (@($changed | Where-Object { $_ -match '(^|/)[^/]*_test\.go$' }).Count -ne 0) { throw "$Name changed a test source file." }
    $changed.ToArray()
}

function Test-OwnedTempPath([string]$Path, [string]$MarkerPath, [string]$Token) {
    $tempBase = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
    $tempPrefix = $tempBase.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
    $resolved = [IO.Path]::GetFullPath($Path)
    if (-not $resolved.StartsWith($tempPrefix, [StringComparison]::OrdinalIgnoreCase)) { return $false }
    if (-not (Test-Path -LiteralPath $MarkerPath -PathType Leaf)) { return $false }
    $marker = Get-Content -Raw -LiteralPath $MarkerPath | ConvertFrom-Json
    return $marker.ownershipToken -ceq $Token -and [IO.Path]::GetFullPath($marker.ownedDirectory) -ceq $resolved
}

function Get-FileInventory([string]$RootPath) {
    $resolvedRoot = [IO.Path]::GetFullPath($RootPath)
    $rootPrefix = $resolvedRoot.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
    $relativePaths = [System.Collections.Generic.List[string]]::new()
    foreach ($file in Get-ChildItem -LiteralPath $resolvedRoot -File -Recurse -Force) {
        if (($file.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) { throw "Reparse file is not allowed in evidence backup: $($file.FullName)" }
        $resolvedFile = [IO.Path]::GetFullPath($file.FullName)
        if (-not $resolvedFile.StartsWith($rootPrefix, [StringComparison]::OrdinalIgnoreCase)) { throw 'Evidence backup member escaped its root.' }
        $relativePaths.Add([IO.Path]::GetRelativePath($resolvedRoot, $resolvedFile).Replace('\', '/'))
    }
    $relativePaths.Sort([StringComparer]::Ordinal)
    $files = [System.Collections.Generic.List[object]]::new()
    foreach ($relative in $relativePaths) {
        $physical = Join-Path $resolvedRoot $relative.Replace('/', [IO.Path]::DirectorySeparatorChar)
        $files.Add([ordered]@{
            path = $relative
            bytes = (Get-Item -LiteralPath $physical).Length
            sha256 = Get-Sha256 $physical
        })
    }
    $files.ToArray()
}

function Copy-VerifiedDirectory([string]$SourceRoot, [string]$DestinationRoot) {
    $sourceInventory = @(Get-FileInventory $SourceRoot)
    foreach ($entry in $sourceInventory) {
        $sourceFile = Join-Path $SourceRoot $entry.path.Replace('/', [IO.Path]::DirectorySeparatorChar)
        $destinationFile = Join-Path $DestinationRoot $entry.path.Replace('/', [IO.Path]::DirectorySeparatorChar)
        $null = New-Item -ItemType Directory -Path (Split-Path -Parent $destinationFile) -Force
        [IO.File]::Copy($sourceFile, $destinationFile, $false)
    }
    $destinationInventory = @(Get-FileInventory $DestinationRoot)
    if ($sourceInventory.Count -ne $destinationInventory.Count) { throw 'Evidence backup member count differs from its source.' }
    for ($i = 0; $i -lt $sourceInventory.Count; $i++) {
        if ($sourceInventory[$i].path -cne $destinationInventory[$i].path -or $sourceInventory[$i].bytes -ne $destinationInventory[$i].bytes -or $sourceInventory[$i].sha256 -cne $destinationInventory[$i].sha256) {
            throw "Evidence backup bytes differ at $($sourceInventory[$i].path)."
        }
    }
    $sourceJson = ConvertTo-Json -InputObject @($sourceInventory) -Depth 10 -Compress
    $destinationJson = ConvertTo-Json -InputObject @($destinationInventory) -Depth 10 -Compress
    [pscustomobject]@{ members = $sourceInventory.Count; sourceSha256 = Get-ByteSha256 ($script:utf8.GetBytes($sourceJson)); destinationSha256 = Get-ByteSha256 ($script:utf8.GetBytes($destinationJson)) }
}

function Copy-VerifiedFile([string]$SourcePath, [string]$DestinationPath, [switch]$Overwrite) {
    if (-not (Test-Path -LiteralPath $SourcePath -PathType Leaf)) { throw "Backup source file is missing: $SourcePath" }
    [IO.File]::Copy($SourcePath, $DestinationPath, [bool]$Overwrite)
    $source = Get-Item -LiteralPath $SourcePath
    $destination = Get-Item -LiteralPath $DestinationPath
    if ($source.Length -ne $destination.Length -or (Get-Sha256 $SourcePath) -cne (Get-Sha256 $DestinationPath)) { throw "Backup file failed byte/hash verification: $SourcePath" }
    [pscustomobject]@{ path = [IO.Path]::GetFileName($DestinationPath); bytes = $destination.Length; sha256 = Get-Sha256 $DestinationPath }
}

function New-RawEvidenceBackup {
    $tempBase = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
    $backupToken = 'R068-LUNA-BACKUP-' + [guid]::NewGuid().ToString('N')
    $backupPath = [IO.Path]::GetFullPath((Join-Path $tempBase ('ccmai-r068-luna-evidence-' + [guid]::NewGuid().ToString('N'))))
    $tempPrefix = $tempBase.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
    if (-not $backupPath.StartsWith($tempPrefix, [StringComparison]::OrdinalIgnoreCase) -or (Test-Path -LiteralPath $backupPath)) { throw 'Could not establish a unique TEMP evidence-backup directory.' }
    $null = New-Item -ItemType Directory -Path $backupPath
    $script:rawBackupPath = $backupPath
    $script:rawBackupOwnershipToken = $backupToken
    $markerPath = Join-Path $backupPath 'owner-marker.json'
    $script:rawBackupMarkerPath = $markerPath
    Write-JsonFileExclusive $markerPath ([ordered]@{ schemaVersion='1.0'; ownershipToken=$backupToken; owner='Codex /root/r068_worker'; sourceCommit=$script:sourceCommit; ownedDirectory=$backupPath; createdAt=[DateTimeOffset]::UtcNow.ToString('o') })
    $evidenceCopyPath = Join-Path $backupPath 'evidence'
    $null = New-Item -ItemType Directory -Path $evidenceCopyPath
    $evidenceCopy = Copy-VerifiedDirectory $script:evidencePath $evidenceCopyPath
    $archiveCopyRoot = Join-Path $backupPath 'archives'
    $null = New-Item -ItemType Directory -Path $archiveCopyRoot
    $archiveCopies = [System.Collections.Generic.List[object]]::new()
    $archiveSources = @(
        [pscustomobject]@{ name='backend-source-before.tar'; path=$script:sourceArchivePath },
        [pscustomobject]@{ name='backend-source-after.tar'; path=$script:sourceArchiveAfterPath },
        [pscustomobject]@{ name='backend-physical-before.zip'; path=$script:physicalArchiveBefore.path },
        [pscustomobject]@{ name='backend-physical-after-restoration.zip'; path=$script:physicalArchiveAfter.path }
    )
    $missingArchives = [System.Collections.Generic.List[string]]::new()
    foreach ($archive in $archiveSources) {
        if ($archive.path -and (Test-Path -LiteralPath $archive.path -PathType Leaf)) {
            $archiveCopies.Add((Copy-VerifiedFile $archive.path (Join-Path $archiveCopyRoot $archive.name)))
        }
        else { $missingArchives.Add($archive.name) }
    }
    if (-not (Test-OwnedTempPath $backupPath $markerPath $backupToken)) { throw 'TEMP evidence-backup ownership marker did not verify.' }
    [pscustomobject]@{
        path = $backupPath
        markerPath = $markerPath
        markerSha256 = Get-Sha256 $markerPath
        ownershipToken = $backupToken
        evidenceMembers = $evidenceCopy.members
        evidenceManifestSha256 = $evidenceCopy.sourceSha256
        archiveCopies = $archiveCopies.ToArray()
        missingArchives = $missingArchives.ToArray()
        allFourArchivesPresent = ($missingArchives.Count -eq 0 -and $archiveCopies.Count -eq 4)
    }
}

$script:projectRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path
$script:probeRoot = [IO.Path]::GetFullPath((Join-Path $script:projectRoot 'docs\reviews\probes'))
$probePrefix = $script:probeRoot.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
$script:evidencePath = [IO.Path]::GetFullPath((Join-Path $script:projectRoot $EvidenceDirectory))
$script:sourceCommit = $SourceCommit
if (-not $script:evidencePath.StartsWith($probePrefix, [StringComparison]::OrdinalIgnoreCase)) { throw 'Evidence directory must stay inside docs/reviews/probes.' }
if (Test-Path -LiteralPath $script:evidencePath) { throw 'This test campaign is single-use; the evidence directory already exists.' }
$null = New-Item -ItemType Directory -Path $script:evidencePath

try {
if ($SourceCommit -notmatch '^[0-9a-f]{40}$' -or $RootStaticApproval -cne "APPROVED $SourceCommit") { throw 'Supply root approval as APPROVED <exact 40-hex source commit>.' }

$script:sourcePaths = @(
    'backend/ai/provider.go', 'backend/ai/openai_compatible.go', 'backend/ai/claude.go',
    'backend/ai/gemini.go', 'backend/ai/usage_presence.go', 'backend/ai/usage_presence_test.go'
)
$script:head = (& git -C $script:projectRoot rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0 -or $script:head -notmatch '^[0-9a-f]{40}$') { throw 'Could not record current HEAD.' }
& git -C $script:projectRoot merge-base --is-ancestor $SourceCommit $script:head
if ($LASTEXITCODE -ne 0) { throw 'Approved source commit is not an ancestor of HEAD.' }
$committedBackendChanges = @(& git -C $script:projectRoot diff --name-only $SourceCommit $script:head -- backend)
if ($LASTEXITCODE -ne 0 -or $committedBackendChanges.Count -ne 0) { throw 'Committed backend source changed after the approved source commit.' }
foreach ($path in $script:sourcePaths) {
    $sourceBlob = (& git -C $script:projectRoot rev-parse "${SourceCommit}:$path").Trim()
    $headBlob = (& git -C $script:projectRoot rev-parse "HEAD:$path").Trim()
    if ($LASTEXITCODE -ne 0 -or $sourceBlob -cne $headBlob) { throw "HEAD source differs from approval source: $path" }
}
$workingBackendChanges = @(& git -C $script:projectRoot diff --name-only HEAD -- backend)
$stagedBackendChanges = @(& git -C $script:projectRoot diff --cached --name-only HEAD -- backend)
$untrackedBackend = @(& git -C $script:projectRoot ls-files --others --exclude-standard -- backend)
if ($LASTEXITCODE -ne 0 -or $workingBackendChanges.Count -or $stagedBackendChanges.Count -or $untrackedBackend.Count) { throw 'Backend worktree must exactly match the approved committed source before campaign setup.' }

$goCandidates = @(Get-Command go.exe -CommandType Application -ErrorAction Stop)
if ($goCandidates.Count -lt 1) { throw 'No Go executable was found.' }
$script:goResolutionCandidates = @($goCandidates | ForEach-Object { $_.Source })
$script:goExecutable = [IO.Path]::GetFullPath([string]$goCandidates[0].Source)
if (-not (Test-Path -LiteralPath $script:goExecutable -PathType Leaf)) { throw 'The first PATH-resolved Go executable is not a file.' }
$gitCandidates = @(Get-Command git.exe -CommandType Application -ErrorAction Stop)
if ($gitCandidates.Count -lt 1) { throw 'No Git executable was found.' }
$script:gitExecutable = [IO.Path]::GetFullPath([string]$gitCandidates[0].Source)
$tarCandidates = @(Get-Command tar.exe -CommandType Application -ErrorAction Stop)
if ($tarCandidates.Count -lt 1) { throw 'No tar executable was found.' }
$script:tarExecutable = [IO.Path]::GetFullPath([string]$tarCandidates[0].Source)

$tempBase = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
$script:ownershipToken = 'R068-LUNA-' + [guid]::NewGuid().ToString('N')
$script:ownedTempPath = [IO.Path]::GetFullPath((Join-Path $tempBase ('ccmai-r068-luna-tests-' + [guid]::NewGuid().ToString('N'))))
$tempPrefix = $tempBase.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
if (-not $script:ownedTempPath.StartsWith($tempPrefix, [StringComparison]::OrdinalIgnoreCase) -or (Test-Path -LiteralPath $script:ownedTempPath)) { throw 'Could not establish a unique owned TEMP directory.' }
$null = New-Item -ItemType Directory -Path $script:ownedTempPath
$script:markerPath = Join-Path $script:ownedTempPath 'owner-marker.json'
Write-JsonFile $script:markerPath ([ordered]@{ schemaVersion='1.0'; ownershipToken=$script:ownershipToken; owner='Codex /root/r068_worker'; sourceCommit=$SourceCommit; ownedDirectory=$script:ownedTempPath; createdAt=[DateTimeOffset]::UtcNow.ToString('o') })
$script:extractRoot = Join-Path $script:ownedTempPath 'extracted'
$null = New-Item -ItemType Directory -Path $script:extractRoot
$script:sourceArchivePath = Join-Path $script:ownedTempPath 'backend-source-before.tar'
$script:sourceArchiveAfterPath = Join-Path $script:ownedTempPath 'backend-source-after.tar'
$script:sourceArchiveSha256 = $null
$sourceArchiveStdout = Join-Path $script:ownedTempPath 'git-archive.stdout.bin'
$sourceArchiveStderr = Join-Path $script:ownedTempPath 'git-archive.stderr.bin'
$archiveRun = Invoke-NativeCapture $script:gitExecutable @('-C', $script:projectRoot, 'archive', '--format=tar', "--output=$($script:sourceArchivePath)", $SourceCommit, 'backend') $script:projectRoot $sourceArchiveStdout $sourceArchiveStderr 120
$script:sourceArchiveRun = $archiveRun
if (-not $archiveRun.processStarted -or $archiveRun.timedOut -or $archiveRun.exitCode -ne 0) { throw 'Could not create the exact committed backend-only archive.' }
$script:sourceArchiveSha256 = Get-Sha256 $script:sourceArchivePath
$script:sourceArchiveBytes = (Get-Item -LiteralPath $script:sourceArchivePath).Length
$extractOut = Join-Path $script:ownedTempPath 'tar-extract.stdout.bin'
$extractErr = Join-Path $script:ownedTempPath 'tar-extract.stderr.bin'
$extractRun = Invoke-NativeCapture $script:tarExecutable @('-xf', $script:sourceArchivePath, '-C', $script:extractRoot) $script:projectRoot $extractOut $extractErr 120
if (-not $extractRun.processStarted -or $extractRun.timedOut -or $extractRun.exitCode -ne 0) { throw 'Could not extract the exact committed backend archive.' }
$script:backendRoot = Join-Path $script:extractRoot 'backend'
if (-not (Test-Path -LiteralPath $script:backendRoot -PathType Container)) { throw 'Exact backend archive did not extract the backend directory.' }
$script:usageSourcePath = Join-Path $script:backendRoot 'ai\usage_presence.go'

$gitTreePath = Join-Path $script:evidencePath 'backend-committed-tree.txt'
$committedPaths = @(& git -C $script:projectRoot ls-tree -r --name-only $SourceCommit -- backend)
if ($LASTEXITCODE -ne 0) { throw 'Could not capture committed backend member list.' }
$committedRelative = @($committedPaths | ForEach-Object { $_.Substring('backend/'.Length) })
[Array]::Sort([string[]]$committedRelative, [StringComparer]::Ordinal)
[IO.File]::WriteAllText($gitTreePath, ($committedPaths -join [Environment]::NewLine) + [Environment]::NewLine, $script:utf8)
$script:manifestBefore = Get-PhysicalManifest $script:backendRoot (Join-Path $script:evidencePath 'backend-physical-manifest-before.json')
$physicalPaths = @($script:manifestBefore.entries | ForEach-Object { $_.path })
if (($physicalPaths -join "`n") -cne ($committedRelative -join "`n") -or $physicalPaths.Count -ne 218) { throw 'Extracted backend physical members do not exactly match the 218 committed backend members.' }
$script:physicalArchiveBefore = Export-PhysicalZip $script:backendRoot (Join-Path $script:ownedTempPath 'backend-physical-before.zip')

$inventoryPath = Join-Path $script:evidencePath 'ai-top-level-test-inventory.json'
$script:testInventory = Get-TopLevelTestInventory (Join-Path $script:backendRoot 'ai') $inventoryPath
$script:originalUsageBytes = [IO.File]::ReadAllBytes($script:usageSourcePath)
$script:originalUsageSha256 = Get-ByteSha256 $script:originalUsageBytes
$backupPath = Join-Path $script:evidencePath 'usage_presence.go.original.bin'
[IO.File]::WriteAllBytes($backupPath, $script:originalUsageBytes)
if ((Get-Sha256 $backupPath) -cne $script:originalUsageSha256) { throw 'Physical raw source backup did not verify before tests.' }
$tempBaselinePath = Join-Path $script:ownedTempPath 'usage_presence.go.original.bin'
[IO.File]::WriteAllBytes($tempBaselinePath, $script:originalUsageBytes)

$script:variableNames = @('GOPROXY', 'GOSUMDB', 'GOTOOLCHAIN', 'CGO_ENABLED')
$script:previousEnvironment = @{}
foreach ($name in $script:variableNames) { $script:previousEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process') }
$env:GOPROXY = 'off'
$env:GOSUMDB = 'off'
$env:GOTOOLCHAIN = 'local'
$env:CGO_ENABLED = '0'

    $allNames = @($script:testInventory.names)
    $newNames = @($script:testInventory.newNames)
    $positive = Invoke-GoTest 'positive-ai-package' @('-C', 'backend', 'test', '-json', './ai', '-count=1')
    Assert-HealthyPositive $positive $allNames

    Invoke-Mutation 'M01' `
        'return AIUsageCount{Status: status}' `
        'if status == AIUsagePresenceAbsent || status == AIUsagePresenceNull { return knownAIUsageCount(0) }; return AIUsageCount{Status: status}' `
        @('-C', 'backend', 'test', '-json', './ai', '-run', '^TestUsagePresenceM01(AbsentNullDetector|ExplicitZeroHealthyControl)$', '-count=1') `
        'TestUsagePresenceM01AbsentNullDetector' `
        'TestUsagePresenceM01ExplicitZeroHealthyControl' `
        $tempBaselinePath

    Invoke-Mutation 'M02' `
        'return usageCountState(AIUsagePresenceInvalid)' `
        'return knownAIUsageCount(0)' `
        @('-C', 'backend', 'test', '-json', './ai', '-run', '^TestUsagePresenceM02(InvalidDetector|ValidIntegerHealthyControl)$', '-count=1') `
        'TestUsagePresenceM02InvalidDetector' `
        'TestUsagePresenceM02ValidIntegerHealthyControl' `
        $tempBaselinePath

    $restored = Invoke-GoTest 'restored-new-tests' @('-C', 'backend', 'test', '-json', './ai', '-run', '^TestUsagePresence', '-count=1')
    Assert-NewTestsPositive $restored $newNames
    $script:campaignCompleted = $true
}
catch {
    $script:failureReason = $_.Exception.Message
}
finally {
    if ($null -ne $script:previousEnvironment) {
        foreach ($name in $script:variableNames) {
            try { [Environment]::SetEnvironmentVariable($name, $script:previousEnvironment[$name], 'Process') }
            catch { $script:finalizationErrors.Add("restore environment $name`: $($_.Exception.Message)") }
        }
    }
    if ($null -ne $script:usageSourcePath -and $null -ne $script:originalUsageBytes -and (Test-Path -LiteralPath $script:usageSourcePath -PathType Leaf)) {
        try {
            [IO.File]::WriteAllBytes($script:usageSourcePath, $script:originalUsageBytes)
            $finalSourceHash = Get-Sha256 $script:usageSourcePath
            $script:restoredPhysicalSource = $finalSourceHash -ceq $script:originalUsageSha256
            $script:finalSourceSha256 = $finalSourceHash
        }
        catch { $script:finalizationErrors.Add('source restore: ' + $_.Exception.Message) }
    }
    if ($null -ne $script:backendRoot -and (Test-Path -LiteralPath $script:backendRoot -PathType Container)) {
        try {
            $script:manifestAfter = Get-PhysicalManifest $script:backendRoot (Join-Path $script:evidencePath 'backend-physical-manifest-after-restoration.json')
            if ($null -eq $script:ownedTempPath) { throw 'Owned TEMP path is unavailable for final physical archive.' }
            $script:physicalArchiveAfter = Export-PhysicalZip $script:backendRoot (Join-Path $script:ownedTempPath 'backend-physical-after-restoration.zip')
        }
        catch { $script:finalizationErrors.Add('final physical snapshot: ' + $_.Exception.Message) }
    }
    $manifestsEqual = $false
    if ($null -ne $script:manifestBefore -and $null -ne $script:manifestAfter) { $manifestsEqual = Compare-ManifestEntries $script:manifestBefore.entries $script:manifestAfter.entries }
    $physicalArchivesEqual = $false
    if ($null -ne $script:physicalArchiveBefore -and $null -ne $script:physicalArchiveAfter) {
        $physicalArchivesEqual = $script:physicalArchiveBefore.bytes -eq $script:physicalArchiveAfter.bytes -and $script:physicalArchiveBefore.sha256 -ceq $script:physicalArchiveAfter.sha256
    }

    if ($null -ne $script:gitExecutable -and $null -ne $script:ownedTempPath) {
        try {
            $archiveAfterStdout = Join-Path $script:ownedTempPath 'git-archive-after.stdout.bin'
            $archiveAfterStderr = Join-Path $script:ownedTempPath 'git-archive-after.stderr.bin'
            $script:sourceArchiveAfterRun = Invoke-NativeCapture $script:gitExecutable @('-C', $script:projectRoot, 'archive', '--format=tar', "--output=$($script:sourceArchiveAfterPath)", $SourceCommit, 'backend') $script:projectRoot $archiveAfterStdout $archiveAfterStderr 120
            if (-not $script:sourceArchiveAfterRun.processStarted -or $script:sourceArchiveAfterRun.timedOut -or $script:sourceArchiveAfterRun.exitCode -ne 0) { throw 'Second committed backend tar command failed.' }
            $afterTarBytes = (Get-Item -LiteralPath $script:sourceArchiveAfterPath).Length
            $afterTarSha256 = Get-Sha256 $script:sourceArchiveAfterPath
            $script:committedArchiveAfter = [ordered]@{ path=$script:sourceArchiveAfterPath; bytes=$afterTarBytes; sha256=$afterTarSha256; processStarted=$script:sourceArchiveAfterRun.processStarted; timedOut=$script:sourceArchiveAfterRun.timedOut; exitCode=$script:sourceArchiveAfterRun.exitCode }
        }
        catch { $script:finalizationErrors.Add('second committed backend tar: ' + $_.Exception.Message) }
    }
    $committedTarsEqual = $false
    if ($null -ne $script:sourceArchiveBytes -and $null -ne $script:sourceArchiveSha256 -and $null -ne $script:committedArchiveAfter) {
        $committedTarsEqual = $script:sourceArchiveBytes -eq $script:committedArchiveAfter.bytes -and $script:sourceArchiveSha256 -ceq $script:committedArchiveAfter.sha256
    }
    if (-not $manifestsEqual -or -not $physicalArchivesEqual -or -not $committedTarsEqual -or -not $script:restoredPhysicalSource) {
        $script:campaignCompleted = $false
        if (-not $script:failureReason) { $script:failureReason = 'Final physical source restoration, backend archive, or second committed tar comparison failed.' }
    }

    $script:cleanupStatus = 'RETAINED_OWNED_TEMP_FOR_RAW_EVIDENCE'
    $cleanupMarkerHash = $null
    if ($null -ne $script:ownedTempPath -and (Test-Path -LiteralPath $script:ownedTempPath -PathType Container)) {
        try {
            if (Test-OwnedTempPath $script:ownedTempPath $script:markerPath $script:ownershipToken) { $cleanupMarkerHash = Get-Sha256 $script:markerPath }
            else { $script:finalizationErrors.Add('owned TEMP marker failed ownership verification') }
        }
        catch { $script:finalizationErrors.Add('owned TEMP ownership verification: ' + $_.Exception.Message) }
    }
    if (-not $script:campaignCompleted -and -not $script:failureReason) { $script:failureReason = 'Campaign did not reach all four successful/expected result states.' }

    $script:summaryPath = Join-Path $script:evidencePath 'campaign-result.json'
    $summaryFactory = {
        param([bool]$Completed, [string]$BackupStatus, [object]$BackupInfo)
        [ordered]@{
            schemaVersion = '1.0'
            sourceCommit = $SourceCommit
            currentHead = $script:head
            rootStaticApproval = $RootStaticApproval
            sourceArchive = [ordered]@{ kind='committed backend-only tar'; sha256=$script:sourceArchiveSha256; bytes=$script:sourceArchiveBytes; memberCount=218; ownedTempLocation=$script:sourceArchivePath }
            sourceArchiveAfter = $script:committedArchiveAfter
            sourceArchiveAfterRun = $script:sourceArchiveAfterRun
            committedBackendTarsEqualByBytesAndSha256 = $committedTarsEqual
            goExecutable = $script:goExecutable
            goResolutionCandidates = $script:goResolutionCandidates
            environment = [ordered]@{ GOPROXY='off'; GOSUMDB='off'; GOTOOLCHAIN='local'; CGO_ENABLED='0' }
            processTimeoutSeconds = $ProcessTimeoutSeconds
            goInvocationCount = $script:invocationCount
            expectedGoInvocationCount = 4
            automaticRetry = $false
            runReceipts = $script:runReceipts.ToArray()
            testInventory = if ($null -ne $script:testInventory) { [ordered]@{ path='ai-top-level-test-inventory.json'; sha256=$script:testInventory.sha256; total=$script:testInventory.total; new=$script:testInventory.new; existing=$script:testInventory.existing } } else { $null }
            sourceRawBackup = if (Test-Path -LiteralPath (Join-Path $script:evidencePath 'usage_presence.go.original.bin')) { [ordered]@{ path='usage_presence.go.original.bin'; bytes=(Get-Item -LiteralPath (Join-Path $script:evidencePath 'usage_presence.go.original.bin')).Length; sha256=Get-Sha256 (Join-Path $script:evidencePath 'usage_presence.go.original.bin'); restoredPhysicalSha256=$script:finalSourceSha256; byteForByteRestored=$script:restoredPhysicalSource } } else { $null }
            backendManifestBefore = if ($null -ne $script:manifestBefore) { [ordered]@{ path='backend-physical-manifest-before.json'; members=$script:manifestBefore.memberCount; sha256=$script:manifestBefore.sha256 } } else { $null }
            backendManifestAfterRestoration = if ($null -ne $script:manifestAfter) { [ordered]@{ path='backend-physical-manifest-after-restoration.json'; members=$script:manifestAfter.memberCount; sha256=$script:manifestAfter.sha256 } } else { $null }
            fullBackendPhysicalManifestsEqual = $manifestsEqual
            physicalArchiveBefore = if ($null -ne $script:physicalArchiveBefore) { [ordered]@{ sha256=$script:physicalArchiveBefore.sha256; bytes=$script:physicalArchiveBefore.bytes; ownedTempLocation=$script:physicalArchiveBefore.path } } else { $null }
            physicalArchiveAfterRestoration = if ($null -ne $script:physicalArchiveAfter) { [ordered]@{ sha256=$script:physicalArchiveAfter.sha256; bytes=$script:physicalArchiveAfter.bytes; ownedTempLocation=$script:physicalArchiveAfter.path } } else { $null }
            fullBackendPhysicalArchivesEqual = $physicalArchivesEqual
            ownedTempDirectory = $script:ownedTempPath
            ownershipMarkerSha256 = $cleanupMarkerHash
            cleanupStatus = $script:cleanupStatus
            rawBackupStatus = $BackupStatus
            rawBackup = $BackupInfo
            firstSummaryWriteWasExclusive = $true
            completed = $Completed
            failure = $script:failureReason
            finalizationErrors = $script:finalizationErrors.ToArray()
        }
    }

    try {
        Write-JsonFileExclusive $script:summaryPath (& $summaryFactory $false 'PENDING' $null)
    }
    catch {
        $script:campaignCompleted = $false
        $script:failureReason = 'Initial exclusive campaign summary write failed: ' + $_.Exception.Message
        $script:finalizationErrors.Add($script:failureReason)
    }

    $backup = $null
    try {
        $backup = New-RawEvidenceBackup
        $script:rawBackupPath = $backup.path
        if (-not $backup.allFourArchivesPresent) {
            $script:campaignCompleted = $false
            $script:finalizationErrors.Add('Raw backup is missing one or more of the four expected committed/physical archives.')
        }
        $backupVerified = $backup.allFourArchivesPresent -and (Test-OwnedTempPath $backup.path $backup.markerPath $backup.ownershipToken)
        $candidateCompleted = $script:campaignCompleted -and $backupVerified -and $script:finalizationErrors.Count -eq 0 -and $script:invocationCount -eq 4
        if (-not $candidateCompleted -and -not $script:failureReason) { $script:failureReason = 'Campaign or raw-evidence backup did not reach all required verified states.' }
        $backupInfo = [ordered]@{
            path=$backup.path
            evidenceMembersCopied=$backup.evidenceMembers
            evidenceManifestSha256=$backup.evidenceManifestSha256
            archiveCopies=$backup.archiveCopies
            missingArchives=$backup.missingArchives
            allFourArchivesPresent=$backup.allFourArchivesPresent
            ownershipMarkerSha256=$backup.markerSha256
            verified=$backupVerified
            receiptRelativePath='raw-backup-verification.json'
        }
        $backupSummaryPath = Join-Path (Join-Path $backup.path 'evidence') 'campaign-result.json'
        $script:campaignCompleted = $candidateCompleted
        $script:summaryObject = & $summaryFactory $candidateCompleted $(if ($backupVerified) { 'VERIFIED' } else { 'INCOMPLETE' }) $backupInfo
        Write-JsonFile $script:summaryPath $script:summaryObject
        $null = Copy-VerifiedFile $script:summaryPath $backupSummaryPath -Overwrite
        $backupEvidenceInventory = @(Get-FileInventory (Join-Path $backup.path 'evidence'))
        $backupEvidenceJson = ConvertTo-Json -InputObject @($backupEvidenceInventory) -Depth 10 -Compress
        $archiveInventory = @(Get-FileInventory (Join-Path $backup.path 'archives'))
        $archiveJson = ConvertTo-Json -InputObject @($archiveInventory) -Depth 10 -Compress
        $script:rawBackupReceipt = [ordered]@{
            schemaVersion='1.0'
            sourceCommit=$SourceCommit
            verified=$backupVerified
            backupPath=$backup.path
            ownershipMarkerSha256=$backup.markerSha256
            evidenceMembers=$backupEvidenceInventory.Count
            evidenceManifestSha256=Get-ByteSha256 ($script:utf8.GetBytes($backupEvidenceJson))
            archiveMembers=$archiveInventory.Count
            archiveManifestSha256=Get-ByteSha256 ($script:utf8.GetBytes($archiveJson))
            campaignSummarySha256=Get-Sha256 $backupSummaryPath
            archiveCopies=$backup.archiveCopies
            missingArchives=$backup.missingArchives
        }
        Write-JsonFileExclusive (Join-Path $backup.path 'raw-backup-verification.json') $script:rawBackupReceipt
        if (-not (Test-OwnedTempPath $backup.path $backup.markerPath $backup.ownershipToken)) { throw 'Final raw-backup ownership verification failed.' }
        if (-not $script:campaignCompleted -and -not $script:failureReason) { $script:failureReason = 'Campaign failed one or more approved test or finalization checks.' }
    }
    catch {
        $script:campaignCompleted = $false
        $script:failureReason = 'Raw evidence backup or finalization failed: ' + $_.Exception.Message
        $script:finalizationErrors.Add($script:failureReason)
        if ($null -ne $script:summaryPath) {
            try {
                $script:summaryObject = & $summaryFactory $false 'FAILED' $(if ($null -ne $backup) { [ordered]@{ path=$backup.path; verified=$false; receiptRelativePath='raw-backup-verification.json' } } else { $null })
                if (Test-Path -LiteralPath $script:summaryPath -PathType Leaf) { Write-JsonFile $script:summaryPath $script:summaryObject }
                else { Write-JsonFileExclusive $script:summaryPath $script:summaryObject }
                if ($null -ne $backup -and (Test-Path -LiteralPath (Join-Path $backup.path 'evidence') -PathType Container)) {
                    $null = Copy-VerifiedFile $script:summaryPath (Join-Path (Join-Path $backup.path 'evidence') 'campaign-result.json') -Overwrite
                }
            }
            catch { $script:finalizationErrors.Add('failure summary persistence: ' + $_.Exception.Message) }
        }
    }
    if ($script:finalizationErrors.Count -gt 0) {
        $script:campaignCompleted = $false
        if (-not $script:failureReason) { $script:failureReason = 'Finalization reported one or more errors.' }
        if ($null -ne $script:summaryPath -and (Test-Path -LiteralPath $script:summaryPath -PathType Leaf)) {
            try {
                $script:summaryObject = & $summaryFactory $false 'FAILED' $(if ($null -ne $backup) { [ordered]@{ path=$backup.path; verified=$false; receiptRelativePath='raw-backup-verification.json' } } else { $null })
                Write-JsonFile $script:summaryPath $script:summaryObject
                if ($null -ne $backup -and (Test-Path -LiteralPath (Join-Path $backup.path 'evidence') -PathType Container)) {
                    $null = Copy-VerifiedFile $script:summaryPath (Join-Path (Join-Path $backup.path 'evidence') 'campaign-result.json') -Overwrite
                }
            }
            catch { $script:finalizationErrors.Add('final failed-status persistence: ' + $_.Exception.Message) }
        }
    }
}

if (-not $script:campaignCompleted) { throw "R068 test campaign stopped without retry: $($script:failureReason). See $script:evidencePath" }
Write-Output "Completed four approved Go test invocations on $SourceCommit; backend restored and raw TEMP evidence backup verified at $script:rawBackupPath."
Write-Output "Evidence=$script:evidencePath"
