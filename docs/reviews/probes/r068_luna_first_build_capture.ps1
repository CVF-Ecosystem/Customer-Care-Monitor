param(
    [Parameter(Mandatory = $true)]
    [string]$EvidenceDirectory
)

$ErrorActionPreference = 'Stop'

$projectRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path
$probeRoot = [IO.Path]::GetFullPath((Join-Path $projectRoot 'docs\reviews\probes'))
$probeRootPrefix = $probeRoot.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
$evidencePath = [IO.Path]::GetFullPath((Join-Path $projectRoot $EvidenceDirectory))
if (-not $evidencePath.StartsWith($probeRootPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'Evidence directory must stay inside docs/reviews/probes.'
}
if (Test-Path -LiteralPath $evidencePath) {
    throw 'Evidence directory already exists; this first-build runner is single-use.'
}

$head = (& git -C $projectRoot rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0 -or $head -notmatch '^[0-9a-f]{40}$') {
    throw 'Could not record the committed source SHA before the build.'
}
$worktree = @(& git -C $projectRoot status --porcelain)
if ($LASTEXITCODE -ne 0 -or $worktree.Count -ne 0) {
    throw 'First-build capture requires a clean worktree at the committed source.'
}

New-Item -ItemType Directory -Path $evidencePath | Out-Null
$sourceManifestPath = Join-Path $evidencePath 'source-tree-manifest.txt'
$archiveBeforePath = Join-Path $evidencePath 'source-before.tar'
$archiveAfterPath = Join-Path $evidencePath 'source-after.tar'
$stdoutPath = Join-Path $evidencePath 'build.stdout.bin'
$stderrPath = Join-Path $evidencePath 'build.stderr.bin'
$summaryPath = Join-Path $evidencePath 'build-result.json'
$null = [IO.File]::WriteAllBytes($stdoutPath, [byte[]]@())
$null = [IO.File]::WriteAllBytes($stderrPath, [byte[]]@())
$manifestLines = @(& git -C $projectRoot ls-tree -r --full-tree $head)
if ($LASTEXITCODE -ne 0) {
    throw 'Could not enumerate the committed source tree.'
}
$utf8 = New-Object System.Text.UTF8Encoding($false)
[IO.File]::WriteAllText($sourceManifestPath, ($manifestLines -join [Environment]::NewLine) + [Environment]::NewLine, $utf8)

& git -C $projectRoot archive --format=tar "--output=$archiveBeforePath" $head
if ($LASTEXITCODE -ne 0) {
    throw 'Could not create the pre-build committed-source archive.'
}

$variableNames = @('GOPROXY', 'GOSUMDB', 'GOTOOLCHAIN', 'CGO_ENABLED')
$previousEnvironment = @{}
foreach ($name in $variableNames) {
    $previousEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}
$env:GOPROXY = 'off'
$env:GOSUMDB = 'off'
$env:GOTOOLCHAIN = 'local'
$env:CGO_ENABLED = '0'
$startedAt = [DateTimeOffset]::UtcNow
$stopwatch = [Diagnostics.Stopwatch]::StartNew()
$processStarted = $false
$exitCode = $null
$failure = $null
$stdoutStream = $null
$stderrStream = $null
$process = $null
try {
    $goCommand = Get-Command go.exe -CommandType Application -ErrorAction Stop
    $startInfo = [Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = $goCommand.Source
    $startInfo.WorkingDirectory = $projectRoot
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    foreach ($argument in @('-C', 'backend', 'build', './...')) {
        $null = $startInfo.ArgumentList.Add($argument)
    }
    $stdoutStream = [IO.File]::Open($stdoutPath, [IO.FileMode]::Create, [IO.FileAccess]::Write, [IO.FileShare]::None)
    $stderrStream = [IO.File]::Open($stderrPath, [IO.FileMode]::Create, [IO.FileAccess]::Write, [IO.FileShare]::None)
    $process = [Diagnostics.Process]::new()
    $process.StartInfo = $startInfo
    $processStarted = $process.Start()
    if (-not $processStarted) {
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
    if ($null -ne $stdoutStream) { $stdoutStream.Dispose() }
    if ($null -ne $stderrStream) { $stderrStream.Dispose() }
    if ($null -ne $process) { $process.Dispose() }
    foreach ($name in $variableNames) {
        [Environment]::SetEnvironmentVariable($name, $previousEnvironment[$name], 'Process')
    }
    $stopwatch.Stop()
}

& git -C $projectRoot archive --format=tar "--output=$archiveAfterPath" $head
if ($LASTEXITCODE -ne 0) {
    throw 'Could not create the post-build committed-source archive.'
}
$stdoutHash = (Get-FileHash -LiteralPath $stdoutPath -Algorithm SHA256).Hash.ToLowerInvariant()
$stderrHash = (Get-FileHash -LiteralPath $stderrPath -Algorithm SHA256).Hash.ToLowerInvariant()
$manifestHash = (Get-FileHash -LiteralPath $sourceManifestPath -Algorithm SHA256).Hash.ToLowerInvariant()
$archiveBeforeHash = (Get-FileHash -LiteralPath $archiveBeforePath -Algorithm SHA256).Hash.ToLowerInvariant()
$archiveAfterHash = (Get-FileHash -LiteralPath $archiveAfterPath -Algorithm SHA256).Hash.ToLowerInvariant()
$receipt = [ordered]@{
    schemaVersion = '1.0'
    commit = $head
    command = 'go -C backend build ./...'
    environment = [ordered]@{ GOPROXY = 'off'; GOSUMDB = 'off'; GOTOOLCHAIN = 'local'; CGO_ENABLED = '0' }
    startUtc = $startedAt.ToString('o')
    durationMilliseconds = $stopwatch.ElapsedMilliseconds
    processStarted = $processStarted
    processStartFailure = $failure
    exitCode = $exitCode
    invocationCount = 1
    stdout = [ordered]@{ path = 'build.stdout.bin'; sha256 = $stdoutHash; bytes = (Get-Item -LiteralPath $stdoutPath).Length }
    stderr = [ordered]@{ path = 'build.stderr.bin'; sha256 = $stderrHash; bytes = (Get-Item -LiteralPath $stderrPath).Length }
    sourceManifest = [ordered]@{ path = 'source-tree-manifest.txt'; sha256 = $manifestHash; entries = $manifestLines.Count }
    archiveBefore = [ordered]@{ path = 'source-before.tar'; sha256 = $archiveBeforeHash; bytes = (Get-Item -LiteralPath $archiveBeforePath).Length }
    archiveAfter = [ordered]@{ path = 'source-after.tar'; sha256 = $archiveAfterHash; bytes = (Get-Item -LiteralPath $archiveAfterPath).Length }
    archiveRestoredEqual = ($archiveBeforeHash -eq $archiveAfterHash)
}
[IO.File]::WriteAllText($summaryPath, ($receipt | ConvertTo-Json -Depth 8) + [Environment]::NewLine, $utf8)
Write-Output ('Build exit=' + $exitCode + '; commit=' + $head + '; archiveEqual=' + $receipt.archiveRestoredEqual)
Write-Output ('Evidence=' + $evidencePath)
if (-not $receipt.archiveRestoredEqual) {
    throw 'Committed source archive changed across the build.'
}
if (-not $receipt.processStarted) {
    throw 'The single build process did not start; inspect the captured receipt and empty raw streams.'
}
exit $exitCode
