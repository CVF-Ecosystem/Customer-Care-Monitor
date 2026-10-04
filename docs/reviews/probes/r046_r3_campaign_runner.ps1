# R046-R3 Evidence Campaign Runner (byte-safe, live-copy)
# Role: Claude REPAIR_WORKER / repair BUILD COMMIT_STEWARD
# Source: Filesystem copy of live backend at R2 (91da0e88)
# Fix: Use ReadAllBytes/WriteAllBytes with no-BOM UTF8 to preserve exact file bytes

$ErrorActionPreference = 'Continue'
$SCRATCH = "C:\Users\tiennm\.gemini\antigravity-ide\brain\dc3013b0-c95d-4c95-97d9-8e786c64953b\scratch"
$SRCDIR  = "$SCRATCH\r046-r3-livecopy"   # byte-identical copy of live backend
$LOGS    = "$SCRATCH\r046-r3-logs"

$NETWORK_NAME = "r046r3net"
$DB_CONTAINER = "r046r3mysql"
$GO_IMAGE     = "golang:1.26-alpine"
$MYSQL_IMAGE  = "mysql:8.0"

$PROD_SHA = "4b7477aa12fd6c232e85818960f6ec0f2947548cd6ac24e714568ebff5a90f0b"
$TEST_SHA  = "06ec2c8d15d6199a679f9daaf0835db1baa7a8a63a35b79df5e9465d90f7c449"
$DB_PASS  = "r046r3pass"

# No-BOM UTF8 encoder for byte-preserving write-back
$NoBomEnc = New-Object System.Text.UTF8Encoding($false)

New-Item -ItemType Directory -Force -Path $LOGS | Out-Null

function Write-Log { param($msg) $ts = (Get-Date -Format 'HH:mm:ss'); Write-Host "[$ts] $msg" }
function Sha256 { param($f) (certutil -hashfile $f SHA256 | Select-String -Pattern "^[0-9a-f]{64}$").ToString().Trim() }

# Byte-safe mutation: read bytes, replace as UTF8 string, write with no-BOM UTF8
function Apply-Mutation {
  param($File, $OldStr, $NewStr)
  $bytes  = [IO.File]::ReadAllBytes($File)
  $text   = [Text.Encoding]::UTF8.GetString($bytes)
  $count  = ([regex]::Matches($text, [regex]::Escape($OldStr))).Count
  if ($count -ne 1) { return $count }
  $mutated = $text.Replace($OldStr, $NewStr)
  $mutBytes = $NoBomEnc.GetBytes($mutated)
  [IO.File]::WriteAllBytes($File, $mutBytes)
  return $count
}

function Restore-Mutation {
  param($File, $OriginalBytes)
  [IO.File]::WriteAllBytes($File, $OriginalBytes)
}

# ---- VERIFY SOURCE COPY ----
$TARGET = "$SRCDIR\engine\analyzer_incremental.go"
$srcHash = Sha256 $TARGET
$tstHash = Sha256 "$SRCDIR\engine\analyzer_finalizer_logging_test.go"
Write-Log "Source copy src=$srcHash match=$($srcHash -eq $PROD_SHA)"
Write-Log "Source copy tst=$tstHash match=$($tstHash -eq $TEST_SHA)"
if ($srcHash -ne $PROD_SHA) { Write-Log "STOP: source hash mismatch; re-run copy"; exit 1 }
if ($tstHash -ne $TEST_SHA) { Write-Log "STOP: test hash mismatch; re-run copy"; exit 1 }

# ---- MODULE CACHE ----
$modCache = (& go env GOMODCACHE).Trim()
Write-Log "GOMODCACHE: $modCache"
if (-not $modCache -or -not (Test-Path $modCache)) { Write-Log "STOP: module cache missing"; exit 1 }

# ---- NETWORK ----
docker network rm $NETWORK_NAME 2>$null | Out-Null
docker network create --internal $NETWORK_NAME | Out-Null
if ($LASTEXITCODE -ne 0) { Write-Log "STOP: network create failed"; exit 1 }
Write-Log "Network $NETWORK_NAME created (--internal)"

# ---- MYSQL ----
docker rm -f $DB_CONTAINER 2>$null | Out-Null
docker run -d --pull=never --name $DB_CONTAINER `
  --network $NETWORK_NAME `
  -e MYSQL_ROOT_PASSWORD=r046r3root `
  -e MYSQL_DATABASE=CCMA `
  -e MYSQL_USER=ccma `
  -e MYSQL_PASSWORD=$DB_PASS `
  $MYSQL_IMAGE --log-bin-trust-function-creators=1 | Out-Null
if ($LASTEXITCODE -ne 0) { Write-Log "STOP: MySQL start failed"; docker network rm $NETWORK_NAME 2>$null; exit 1 }
Write-Log "MySQL $DB_CONTAINER started (--pull=never, no host ports)"

$ready = $false
for ($i = 0; $i -lt 30; $i++) {
  Start-Sleep 2
  $val = docker exec -e MYSQL_PWD=$DB_PASS $DB_CONTAINER mysql -uccma -N -e 'SELECT @@log_bin_trust_function_creators' CCMA 2>$null
  if ($LASTEXITCODE -eq 0 -and "$val".Trim() -eq '1') { $ready = $true; break }
}
if (-not $ready) { Write-Log "STOP: MySQL not ready"; docker rm -f -v $DB_CONTAINER 2>$null; docker network rm $NETWORK_NAME 2>$null; exit 1 }
Write-Log "MySQL ready (log_bin_trust_function_creators=1)"

$DSN = "ccma:${DB_PASS}@tcp(${DB_CONTAINER}:3306)/CCMA?charset=utf8mb4&parseTime=True&loc=UTC"

function Run-Test {
  param($Name, $Run)
  Write-Log "--- RUN: $Name ---"
  $logFile = "$LOGS\${Name}.jsonl"
  $t0 = Get-Date
  docker run --rm --pull=never `
    --network $NETWORK_NAME `
    -v "${SRCDIR}:/src" `
    -v "${modCache}:/go/pkg/mod:ro" `
    -w /src `
    -e GOFLAGS=-mod=readonly `
    -e GOPROXY=off `
    -e GOTOOLCHAIN=local `
    -e CGO_ENABLED=0 `
    -e "TEST_DB_DSN=${DSN}" `
    $GO_IMAGE `
    go test ./engine -count=1 -p 1 -json -timeout=180s -run "^($Run)" `
  2>&1 | Tee-Object -FilePath $logFile
  $ex = $LASTEXITCODE
  $elapsed = [math]::Round(((Get-Date) - $t0).TotalSeconds, 3)
  $logSha = Sha256 $logFile
  return [PSCustomObject]@{ Name=$Name; Run=$Run; Exit=$ex; Seconds=$elapsed; LogFile=$logFile; LogSha=$logSha }
}

function Parse-Events {
  param($LogFile)
  $topPass=@(); $topFail=@(); $topSkip=@(); $subPass=@(); $subFail=@()
  $failOut=@{}
  Get-Content $LogFile | ForEach-Object {
    try {
      $e = $_ | ConvertFrom-Json
      if ($null -eq $e.Test -or $e.Test -eq "") { return }
      $isSub = $e.Test -match '/'
      switch ($e.Action) {
        "pass" { if ($isSub) { $subPass+=$e.Test } else { $topPass+=$e.Test } }
        "fail" { if ($isSub) { $subFail+=$e.Test } else { $topFail+=$e.Test } }
        "skip" { $topSkip+=$e.Test }
        "output" {
          if ($e.Output -notmatch '^\s*$') {
            if (-not $failOut[$e.Test]) { $failOut[$e.Test] = @() }
            $failOut[$e.Test] += $e.Output.TrimEnd()
          }
        }
      }
    } catch {}
  }
  return [PSCustomObject]@{
    TopPass=$topPass; TopFail=$topFail; TopSkip=$topSkip
    SubPass=$subPass; SubFail=$subFail; FailOut=$failOut
  }
}

# ==============================================================
# BASELINE
# ==============================================================
$baseRun    = Run-Test -Name "baseline" -Run "TestFL"
$baseParsed = Parse-Events -LogFile $baseRun.LogFile
Write-Log "BASELINE: topPass=$($baseParsed.TopPass.Count) subPass=$($baseParsed.SubPass.Count) topFail=$($baseParsed.TopFail.Count) skip=$($baseParsed.TopSkip.Count) exit=$($baseRun.Exit) sec=$($baseRun.Seconds)"
Write-Log "BASELINE names: $($baseParsed.TopPass | Sort-Object | Join-String -Separator ' | ')"
if ($baseRun.Exit -ne 0) {
  Write-Log "STOP: baseline failed; exit=$($baseRun.Exit); failures=$($baseParsed.TopFail)"
  docker rm -f -v $DB_CONTAINER 2>$null; docker network rm $NETWORK_NAME 2>$null
  exit 1
}

# ==============================================================
# MUTATIONS (byte-safe)
# ==============================================================
$M01_OLD = "`t`tswitch {`n`t`tcase txErr == nil:`n`t`t`tlastErr = nil`n`t`tcase errors.Is(txErr, errFinalizeMissing):`n`t`t`tlastErr = errFinalizeMissing`n`t`tdefault:`n`t`t`t// Unknown driver or boundary error: contains raw text that must not`n`t`t`t// reach logs or the returned error wrapper.`n`t`t`tlastErr = errFinalizeWrite`n`t`t}"
$M01_NEW = "`t`tlastErr = txErr"

$M02_OLD = "`t`tlog.Printf(`"[analyzer] fallback error mark failed for run %s: write error`", run.ID)"
$M02_NEW = "`t`tlog.Printf(`"[analyzer] fallback error mark failed for run %s: write error: %v`", run.ID, err)"

$M03_OLD = "`treturn db.DB.Session(&gorm.Session{Logger: db.DB.Logger.LogMode(logger.Silent)})"
$M03_NEW = "`treturn db.DB.Session(&gorm.Session{Logger: db.DB.Logger.LogMode(logger.Info)})"

$MUTS = @(
  @{ Name="M01"; Old=$M01_OLD; New=$M01_NEW; ExpMut="b212ec5594d86ca7958c499b40cd7269277ccd82e9f8dd152640bdd11274cbcd"; Sel="TestFL01TransactionBoundaryUnknownErrorIsContained|TestFL05TransactionBoundaryNormalizationDetector"; Kill="TestFL05TransactionBoundaryNormalizationDetector" },
  @{ Name="M02"; Old=$M02_OLD; New=$M02_NEW; ExpMut="a7306bb0ec6a0b551fa0407ccde94a10b30f59f99bc9df185a4b6a93a87375a6"; Sel="TestFL05FallbackLogDetector"; Kill="TestFL05FallbackLogDetector" },
  @{ Name="M03"; Old=$M03_OLD; New=$M03_NEW; ExpMut="426a427bd5dd7d378fcf489cc8128e098b52194cf34bc242206eb7b2fc7bf9d3"; Sel="TestFL05GormSinkDetector"; Kill="TestFL05GormSinkDetector" }
)

$mutResults = @()
foreach ($m in $MUTS) {
  Write-Log "=== $($m.Name) ==="
  
  # Pre-check baseline
  $preSha = Sha256 $TARGET
  if ($preSha -ne $PROD_SHA) { Write-Log "STOP: $($m.Name) pre-mutation hash wrong: $preSha"; docker rm -f -v $DB_CONTAINER 2>$null; docker network rm $NETWORK_NAME 2>$null; exit 1 }
  
  # Save original bytes for exact restore
  $origBytes = [IO.File]::ReadAllBytes($TARGET)
  
  # Apply mutation (byte-safe)
  $cnt = Apply-Mutation -File $TARGET -OldStr $m.Old -NewStr $m.New
  $mutSha = Sha256 $TARGET
  Write-Log "$($m.Name): matches=$cnt mutSha=$mutSha expectedSha=$($m.ExpMut) shaMatch=$($mutSha -eq $m.ExpMut)"
  if ($cnt -ne 1) { Write-Log "STOP: $($m.Name) NOT_APPLIED (count=$cnt)"; Restore-Mutation -File $TARGET -OriginalBytes $origBytes; docker rm -f -v $DB_CONTAINER 2>$null; docker network rm $NETWORK_NAME 2>$null; exit 1 }
  
  # Run mutated
  $mutRun = Run-Test -Name "$($m.Name)-mutated" -Run $m.Sel
  $mutP   = Parse-Events -LogFile $mutRun.LogFile
  
  # Restore exactly (byte-for-byte)
  Restore-Mutation -File $TARGET -OriginalBytes $origBytes
  $restSha = Sha256 $TARGET
  Write-Log "$($m.Name): restored=$restSha eq=$($restSha -eq $PROD_SHA)"
  if ($restSha -ne $PROD_SHA) { Write-Log "STOP: restoration failed for $($m.Name)"; docker rm -f -v $DB_CONTAINER 2>$null; docker network rm $NETWORK_NAME 2>$null; exit 1 }
  
  # Run restored
  $restRun = Run-Test -Name "$($m.Name)-restored" -Run $m.Sel
  $restP   = Parse-Events -LogFile $restRun.LogFile
  
  # Outcome
  $outcome = "INCONCLUSIVE"
  if ($mutRun.Exit -ne 0 -and $mutP.TopFail -contains $m.Kill) { $outcome = "KILLED_NAMED_BEHAVIORAL_ASSERTION" }
  elseif ($mutRun.Exit -eq 0) { $outcome = "SURVIVED" }
  
  # Grab named failure output
  $namedFailOut = ""
  if ($mutP.FailOut[$m.Kill]) { $namedFailOut = ($mutP.FailOut[$m.Kill] | Where-Object { $_ -match 'FAIL|DETECTOR|BLOCKED' } | Select-Object -First 5) -join " | " }
  
  Write-Log "$($m.Name): outcome=$outcome mutFail='$($mutP.TopFail)' namedOut='$namedFailOut' restPass='$($restP.TopPass)' restExit=$($restRun.Exit)"
  
  $mutResults += [PSCustomObject]@{
    Name=$m.Name; Matches=$cnt; MutSha=$mutSha; ExpMutSha=$m.ExpMut; RestSha=$restSha
    Outcome=$outcome; MutExit=$mutRun.Exit; RestExit=$restRun.Exit
    MutFail=$mutP.TopFail; MutPass=$mutP.TopPass; RestPass=$restP.TopPass
    MutLogSha=$mutRun.LogSha; RestLogSha=$restRun.LogSha; NamedFailOut=$namedFailOut
    Kill=$m.Kill; Old=$m.Old; New=$m.New; Sel=$m.Sel; MutLogFile=$mutRun.LogFile; RestLogFile=$restRun.LogFile
  }
}

# ==============================================================
# COMBINED SELECTION
# ==============================================================
$combSel = "TestFL|TestOrdinary|TestEveryTerminal|TestTerminalRunHolds|TestAcceptedCancel|TestCancelled|TestReservation|TestOwnership"
$combRun    = Run-Test -Name "combined" -Run $combSel
$combParsed = Parse-Events -LogFile $combRun.LogFile
Write-Log "COMBINED: topPass=$($combParsed.TopPass.Count) subPass=$($combParsed.SubPass.Count) topFail=$($combParsed.TopFail.Count) skip=$($combParsed.TopSkip.Count) exit=$($combRun.Exit) sec=$($combRun.Seconds)"
if ($combParsed.TopFail.Count -gt 0) { Write-Log "COMBINED FAIL: $($combParsed.TopFail)" }

# ==============================================================
# TEARDOWN
# ==============================================================
Write-Log "=== Teardown ==="
docker rm -f -v $DB_CONTAINER 2>&1 | Tee-Object -Append "$LOGS\teardown.log"
$dbRmExit = $LASTEXITCODE
docker network rm $NETWORK_NAME 2>&1 | Tee-Object -Append "$LOGS\teardown.log"
$netRmExit = $LASTEXITCODE

$dbCheck  = docker ps -aq --filter "name=^${DB_CONTAINER}$" 2>$null
$netCheck = docker network ls -q --filter "name=^${NETWORK_NAME}$" 2>$null
Write-Log "Teardown: dbRmExit=$dbRmExit netRmExit=$netRmExit dbAbsent=$([string]::IsNullOrWhiteSpace($dbCheck)) netAbsent=$([string]::IsNullOrWhiteSpace($netCheck))"

# Final source integrity
$finalSrcSha = Sha256 $TARGET
Write-Log "Final source SHA: $finalSrcSha eq=$($finalSrcSha -eq $PROD_SHA)"

# ==============================================================
# MACHINE RECEIPT SUMMARY
# ==============================================================
Write-Log "=========================================="
Write-Log "RECEIPT SUMMARY"
Write-Log "=========================================="
Write-Log "ProductionSha: $PROD_SHA"
Write-Log "TestSha: $TEST_SHA"
Write-Log ""
Write-Log "BASELINE: topPass=$($baseParsed.TopPass.Count) subPass=$($baseParsed.SubPass.Count) exit=$($baseRun.Exit) sec=$($baseRun.Seconds) logSha=$($baseRun.LogSha)"
Write-Log "Top-level names:"
$baseParsed.TopPass | Sort-Object | ForEach-Object { Write-Log "  PASS: $_" }
Write-Log "Subtest names:"
$baseParsed.SubPass | Sort-Object | ForEach-Object { Write-Log "  PASS: $_" }
Write-Log ""
foreach ($mr in $mutResults) {
  Write-Log "$($mr.Name): matches=$($mr.Matches) mutSha=$($mr.MutSha) shaMatch=$($mr.MutSha -eq $mr.ExpMutSha) restSha=$($mr.RestSha) restMatch=$($mr.RestSha -eq $PROD_SHA) outcome=$($mr.Outcome)"
  Write-Log "  mutExit=$($mr.MutExit) mutFail='$($mr.MutFail)' namedOut='$($mr.NamedFailOut)'"
  Write-Log "  restExit=$($mr.RestExit) restPass='$($mr.RestPass)'"
  Write-Log "  mutLogSha=$($mr.MutLogSha) restLogSha=$($mr.RestLogSha)"
}
Write-Log ""
Write-Log "COMBINED: topPass=$($combParsed.TopPass.Count) subPass=$($combParsed.SubPass.Count) topFail=$($combParsed.TopFail.Count) skip=$($combParsed.TopSkip.Count) exit=$($combRun.Exit) sec=$($combRun.Seconds) logSha=$($combRun.LogSha)"
Write-Log "Combined top-level names:"
$combParsed.TopPass | Sort-Object | ForEach-Object { Write-Log "  PASS: $_" }
if ($combParsed.TopFail.Count -gt 0) { $combParsed.TopFail | ForEach-Object { Write-Log "  FAIL: $_" } }
Write-Log "Combined subtest names:"
$combParsed.SubPass | Sort-Object | ForEach-Object { Write-Log "  PASS: $_" }
Write-Log ""
Write-Log "TEARDOWN: dbRmExit=$dbRmExit netRmExit=$netRmExit dbAbsent=$([string]::IsNullOrWhiteSpace($dbCheck)) netAbsent=$([string]::IsNullOrWhiteSpace($netCheck))"
Write-Log "Final source SHA: $finalSrcSha eq=$($finalSrcSha -eq $PROD_SHA)"
Write-Log "=========================================="
Write-Log "CAMPAIGN COMPLETE"
Write-Log "=========================================="
