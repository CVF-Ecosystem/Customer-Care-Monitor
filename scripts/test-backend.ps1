<#
.SYNOPSIS
  Run backend Go tests against a throwaway MySQL, entirely inside Docker.

.DESCRIPTION
  - Starts a disposable mysql:8.0 on its own Docker network (no host port, no
    host data) with --log-bin-trust-function-creators=1 set at startup, so the
    trigger-based failure tests never hit "Error 1419".
  - Waits until the application user can log in and the setting reads 1.
  - Runs `go test` in golang:1.26-alpine with the host module cache mounted
    read-only and GOPROXY=off (no downloads), so it also works where host
    security policy blocks freshly built Go test binaries.
  - Always removes the container and network afterwards.
  Never touches the persistent Compose database.

.EXAMPLE
  powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1
.EXAMPLE
  powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./api/handlers -Run 'TestSyncChannelNow' -VerboseTests
#>
param(
  [string]$Packages = './...',
  [string]$Run = '',
  [switch]$VerboseTests
)

# Native commands are checked via $LASTEXITCODE: with 'Stop', Windows PowerShell
# 5.1 turns expected stderr (e.g. "MySQL not ready yet") into a terminating error.
$ErrorActionPreference = 'Continue'
$backend = Join-Path (Split-Path $PSScriptRoot -Parent) 'backend'
$suffix = [guid]::NewGuid().ToString('N').Substring(0, 8)
$network = "ccma-test-net-$suffix"
$dbName = "ccma-test-db-$suffix"
$dbPassword = [guid]::NewGuid().ToString('N')
$modCache = (& go env GOMODCACHE).Trim()
if (-not $modCache -or -not (Test-Path $modCache)) { throw "Go module cache not found; run 'go mod download' in backend/ once." }

$exitCode = 1
try {
  docker network create $network | Out-Null
  if ($LASTEXITCODE -ne 0) { throw "Could not create Docker network $network." }
  docker run -d --name $dbName --network $network `
    -e MYSQL_ROOT_PASSWORD=$dbPassword -e MYSQL_DATABASE=CCMA -e MYSQL_USER=ccma -e MYSQL_PASSWORD=$dbPassword `
    mysql:8.0 --log-bin-trust-function-creators=1 | Out-Null
  if ($LASTEXITCODE -ne 0) { throw "Could not start disposable MySQL $dbName." }

  Write-Host "Waiting for disposable MySQL ($dbName)..."
  $ready = $false
  for ($i = 0; $i -lt 60; $i++) {
    $value = docker exec -e MYSQL_PWD=$dbPassword $dbName mysql -uccma -N -e 'SELECT @@log_bin_trust_function_creators' CCMA 2>$null
    if ($LASTEXITCODE -eq 0 -and "$value".Trim() -eq '1') { $ready = $true; break }
    Start-Sleep -Seconds 2
  }
  if (-not $ready) { throw 'Disposable MySQL did not become ready with log_bin_trust_function_creators=1.' }

  $testArgs = @('test', $Packages, '-count=1', '-p', '1')
  if ($Run) { $testArgs += @('-run', $Run) }
  if ($VerboseTests) { $testArgs += '-v' }

  docker run --rm --network $network `
    -v "${backend}:/src" -v "${modCache}:/go/pkg/mod:ro" -w /src `
    -e GOFLAGS=-mod=readonly -e GOPROXY=off -e GOTOOLCHAIN=local -e CGO_ENABLED=0 `
    -e "TEST_DB_DSN=ccma:${dbPassword}@tcp(${dbName}:3306)/CCMA?charset=utf8mb4&parseTime=True&loc=UTC" `
    golang:1.26-alpine go @testArgs
  $exitCode = $LASTEXITCODE
}
finally {
  docker rm -f $dbName 2>$null | Out-Null
  docker network rm $network 2>$null | Out-Null
  Write-Host "Removed disposable MySQL and network."
}
exit $exitCode
