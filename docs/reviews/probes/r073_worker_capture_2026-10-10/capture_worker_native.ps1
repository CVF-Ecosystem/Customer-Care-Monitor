$ErrorActionPreference = 'Stop'

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..\..\..'))
$frontendRoot = Join-Path $repoRoot 'frontend'
$captureRoot = $PSScriptRoot
$sourceCommit = '3a05b5032f69f1d0bee8e7079c8008813129b928'
$workspaceHead = '96246a060b46318a9d8e0de319c26bfef870fcb3'
$productPaths = @(
  'frontend/src/views/Jobs/job-detail/adapter-usage-presence.ts',
  'frontend/src/__tests__/adapter-usage-presence.spec.ts',
  'frontend/src/components/ui/RunObservationPanel.vue',
  'frontend/src/i18n/en.ts',
  'frontend/src/i18n/vi.ts'
)

function Write-JsonFile([string]$Path, [object]$Value) {
  $json = $Value | ConvertTo-Json -Depth 20
  [System.IO.File]::WriteAllText($Path, $json, [System.Text.UTF8Encoding]::new($false))
}

function Write-TextFile([string]$Path, [string]$Value) {
  [System.IO.File]::WriteAllText($Path, $Value, [System.Text.UTF8Encoding]::new($false))
}

function Get-ProductSnapshot([string]$Label) {
  $files = foreach ($relativePath in $productPaths) {
    $fullPath = Join-Path $repoRoot $relativePath
    $sourceBlob = (& git -C $repoRoot rev-parse "${sourceCommit}:$relativePath").Trim()
    $currentBlob = (& git -C $repoRoot hash-object -- $relativePath).Trim()
    [pscustomobject]@{
      path = $relativePath
      bytes = (Get-Item -LiteralPath $fullPath).Length
      physicalSha256 = (Get-FileHash -LiteralPath $fullPath -Algorithm SHA256).Hash.ToLowerInvariant()
      gitBlobAtSourceCommit = $sourceBlob
      gitBlobCurrent = $currentBlob
    }
  }
  $snapshot = [pscustomobject]@{
    label = $Label
    capturedAtUtc = [DateTimeOffset]::UtcNow.ToString('o')
    sourceCommit = $sourceCommit
    workspaceHead = $workspaceHead
    files = @($files)
  }
  Write-JsonFile (Join-Path $captureRoot "source-$Label.json") $snapshot
  return $snapshot
}

function Compare-ProductSnapshots([object]$Left, [object]$Right) {
  foreach ($relativePath in $productPaths) {
    $leftFile = $Left.files | Where-Object path -eq $relativePath
    $rightFile = $Right.files | Where-Object path -eq $relativePath
    if ($null -eq $leftFile -or $null -eq $rightFile -or
        $leftFile.physicalSha256 -ne $rightFile.physicalSha256 -or
        $leftFile.gitBlobCurrent -ne $rightFile.gitBlobCurrent -or
        $leftFile.gitBlobAtSourceCommit -ne $rightFile.gitBlobAtSourceCommit) {
      return $false
    }
  }
  return $true
}

function Invoke-CapturedProcess([string]$Name, [string[]]$Arguments) {
  $nodeExe = (Get-Command node.exe -ErrorAction Stop).Source
  $startedAt = [DateTimeOffset]::UtcNow
  $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
  $stdoutPath = Join-Path $captureRoot "$Name.stdout.log"
  $stderrPath = Join-Path $captureRoot "$Name.stderr.log"
  $launchError = $null
  $processId = $null
  $exitCode = $null
  try {
    $process = Start-Process -FilePath $nodeExe -ArgumentList $Arguments -WorkingDirectory $frontendRoot -PassThru -Wait -WindowStyle Hidden -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath
    $processId = $process.Id
    $exitCode = $process.ExitCode
  } catch {
    $launchError = $_.ToString()
  }
  $stopwatch.Stop()
  $endedAt = [DateTimeOffset]::UtcNow
  $commandLine = '"' + $nodeExe + '" ' + ($Arguments -join ' ')
  $metadata = [pscustomobject]@{
    name = $Name
    executable = $nodeExe
    workingDirectory = $frontendRoot
    argv = @($Arguments)
    commandLine = $commandLine
    pid = $processId
    startedAtUtc = $startedAt.ToString('o')
    endedAtUtc = $endedAt.ToString('o')
    elapsedMilliseconds = [Math]::Round($stopwatch.Elapsed.TotalMilliseconds, 3)
    exitCode = $exitCode
    launchError = $launchError
    stdout = [System.IO.Path]::GetFileName($stdoutPath)
    stderr = [System.IO.Path]::GetFileName($stderrPath)
  }
  Write-JsonFile (Join-Path $captureRoot "$Name.process.json") $metadata
  return $metadata
}

$actualHead = (& git -C $repoRoot rev-parse HEAD).Trim()
$sourceLockDiff = & git -C $repoRoot diff --quiet $sourceCommit -- $productPaths
$sourceLockExit = $LASTEXITCODE
if ($actualHead -ne $workspaceHead -or $sourceLockExit -ne 0) {
  Write-JsonFile (Join-Path $captureRoot 'validation-summary.json') ([pscustomobject]@{
    status = 'PRECONDITION_FAILED'
    workspaceHeadExpected = $workspaceHead
    workspaceHeadActual = $actualHead
    productDiffExit = $sourceLockExit
    nativeInvocations = 0
  })
  exit 1
}

$nodeExe = (Get-Command node.exe -ErrorAction Stop).Source
$packageVersions = [ordered]@{}
foreach ($packageName in @('vitest', 'vue-tsc', 'vue', 'vue-i18n', 'happy-dom', 'typescript')) {
  $packageJson = Join-Path $frontendRoot "node_modules/$packageName/package.json"
  $package = Get-Content -LiteralPath $packageJson -Raw | ConvertFrom-Json
  $packageVersions[$packageName] = $package.version
}
$manifest = [pscustomobject]@{
  schemaVersion = 1
  tranche = 'CCMAI-RUNTIME-073'
  role = 'IMPLEMENTATION_WORKER_VALIDATION_ONLY'
  approval = 'docs/reviews/R073_REPAIRED_SOURCE_STATIC_APPROVAL_2026-10-10.md'
  sourceCommit = $sourceCommit
  workspaceHead = $workspaceHead
  firstSourceNativeDisposition = 'NOT_RUN'
  currentSourceLocked = $true
  liveGovernanceEvidenceRequired = $true
  nodeExecutable = $nodeExe
  nodeVersionObservedBeforeCampaign = 'v24.19.0'
  packageVersions = $packageVersions
  expectedReporterInventory = [pscustomobject]@{ files = 5; tests = 49; newFile = 28; inherited = 21; actual = 'reporter determines' }
  permittedInvocations = [pscustomobject]@{ vitest = 1; forcedVueTsc = 1; go = 0; browser = 0; fullFrontendBuild = 0 }
  productPaths = $productPaths
  vitestArgv = @(
    'node_modules/vitest/vitest.mjs', 'run', '--reporter=default', '--reporter=json',
    '--outputFile.json=../docs/reviews/probes/r073_worker_capture_2026-10-10/vitest-report.json',
    'src/__tests__/run-observation.spec.ts',
    'src/__tests__/run-observation-panel.spec.ts',
    'src/__tests__/run-observation-job-detail.spec.ts',
    'src/__tests__/run-observation-adapter-compat.spec.ts',
    'src/__tests__/adapter-usage-presence.spec.ts'
  )
  vueTscArgv = @('node_modules/vue-tsc/bin/vue-tsc.js', '-b', '--force')
}
Write-JsonFile (Join-Path $captureRoot 'capture-manifest.json') $manifest
$gitStatusBefore = (& git -C $repoRoot status --short) -join "`n"
Write-TextFile (Join-Path $captureRoot 'git-status-before.txt') $gitStatusBefore
$beforeSnapshot = Get-ProductSnapshot 'before-native'

$vitestArgs = @($manifest.vitestArgv)
$vitestArgs[0] = 'node_modules/vitest/vitest.mjs'
$vitest = Invoke-CapturedProcess 'vitest' $vitestArgs
$afterVitestSnapshot = Get-ProductSnapshot 'after-vitest'
if ($vitest.launchError -or $vitest.exitCode -ne 0) {
  Write-JsonFile (Join-Path $captureRoot 'validation-summary.json') ([pscustomobject]@{
    status = 'VITEST_FAILED_STOPPED'
    vitestExitCode = $vitest.exitCode
    vueTsc = 'NOT_RUN_AFTER_FAILURE'
    nativeInvocations = 1
    sourceCommit = $sourceCommit
    workspaceHead = $workspaceHead
  })
  exit 1
}
if (-not (Compare-ProductSnapshots $beforeSnapshot $afterVitestSnapshot)) {
  Write-JsonFile (Join-Path $captureRoot 'validation-summary.json') ([pscustomobject]@{
    status = 'VITEST_SOURCE_DRIFT_STOPPED'
    vitestExitCode = $vitest.exitCode
    vueTsc = 'NOT_RUN_AFTER_SOURCE_DRIFT'
    nativeInvocations = 1
    sourceCommit = $sourceCommit
    workspaceHead = $workspaceHead
  })
  exit 1
}

$reportPath = Join-Path $captureRoot 'vitest-report.json'
if (-not (Test-Path -LiteralPath $reportPath)) {
  Write-JsonFile (Join-Path $captureRoot 'validation-summary.json') ([pscustomobject]@{
    status = 'VITEST_REPORT_MISSING_STOPPED'
    vitestExitCode = $vitest.exitCode
    vueTsc = 'NOT_RUN_AFTER_CAPTURE_FAILURE'
    nativeInvocations = 1
    sourceCommit = $sourceCommit
    workspaceHead = $workspaceHead
  })
  exit 1
}
try {
  $report = Get-Content -LiteralPath $reportPath -Raw | ConvertFrom-Json
} catch {
  Write-JsonFile (Join-Path $captureRoot 'validation-summary.json') ([pscustomobject]@{
    status = 'VITEST_REPORT_PARSE_FAILED_STOPPED'
    vitestExitCode = $vitest.exitCode
    reportParseError = $_.ToString()
    vueTsc = 'NOT_RUN_AFTER_CAPTURE_FAILURE'
    nativeInvocations = 1
    sourceCommit = $sourceCommit
    workspaceHead = $workspaceHead
  })
  exit 1
}

$vueTscArgs = @($manifest.vueTscArgv)
$vueTsc = Invoke-CapturedProcess 'vue-tsc' $vueTscArgs
$afterVueTscSnapshot = Get-ProductSnapshot 'after-vue-tsc'
$gitStatusAfter = (& git -C $repoRoot status --short) -join "`n"
Write-TextFile (Join-Path $captureRoot 'git-status-after.txt') $gitStatusAfter
$sourceUnchangedAfterTypecheck = Compare-ProductSnapshots $beforeSnapshot $afterVueTscSnapshot

$summary = [pscustomobject]@{
  status = if ($vueTsc.launchError -or $vueTsc.exitCode -ne 0) { 'VUE_TSC_FAILED' } elseif (-not $sourceUnchangedAfterTypecheck) { 'VUE_TSC_SOURCE_DRIFT' } else { 'BOTH_COMMANDS_EXITED_ZERO' }
  sourceCommit = $sourceCommit
  workspaceHead = $workspaceHead
  vitestExitCode = $vitest.exitCode
  vueTscExitCode = $vueTsc.exitCode
  vitestReport = [pscustomobject]@{
    file = 'vitest-report.json'
    numTotalTestSuites = $report.numTotalTestSuites
    numPassedTestSuites = $report.numPassedTestSuites
    numFailedTestSuites = $report.numFailedTestSuites
    numPendingTestSuites = $report.numPendingTestSuites
    numTotalTests = $report.numTotalTests
    numPassedTests = $report.numPassedTests
    numFailedTests = $report.numFailedTests
    numPendingTests = $report.numPendingTests
    numTodoTests = $report.numTodoTests
    success = $report.success
  }
  sourceUnchangedBeforeToAfterVitest = $true
  sourceUnchangedBeforeToAfterTypecheck = $sourceUnchangedAfterTypecheck
  nativeInvocations = 2
  sourceSnapshots = @('source-before-native.json', 'source-after-vitest.json', 'source-after-vue-tsc.json')
  rawLogs = @('vitest.stdout.log', 'vitest.stderr.log', 'vue-tsc.stdout.log', 'vue-tsc.stderr.log')
}
Write-JsonFile (Join-Path $captureRoot 'validation-summary.json') $summary

if ($vueTsc.launchError -or $null -eq $vueTsc.exitCode -or $vueTsc.exitCode -ne 0 -or -not $sourceUnchangedAfterTypecheck) {
  exit 1
}
exit 0
