<#
.SYNOPSIS
  CCMAI-UX-000: capture UI screenshots at desktop/mobile in light/dark.

.DESCRIPTION
  -Mode preview : builds the frontend and serves dist with `vite preview`; captures the
                  static /wireframes/* routes (no backend, no API data).
  -Mode app     : starts a DISPOSABLE Compose project (scripts/ui-screenshots/compose.yml)
                  with throwaway secrets written to a temp file outside the repo and MySQL in
                  tmpfs, creates a throwaway admin, imports the built-in synthetic demo data,
                  binds two synthetic snapshots (one "source changed", one "cannot verify"),
                  captures the app routes, then removes the project. It never reads the repo
                  .env, never touches Compose project `ccma`, and never calls an AI provider.

.EXAMPLE
  powershell -ExecutionPolicy Bypass -File scripts/ui-screenshots.ps1 -Mode preview -OutDir out/shots
  powershell -ExecutionPolicy Bypass -File scripts/ui-screenshots.ps1 -Mode app -OutDir out/shots
#>
param(
  [ValidateSet('preview', 'app')] [string] $Mode = 'preview',
  [string] $OutDir = '',
  [string] $Themes = 'light,dark',
  [string] $Viewports = 'desktop,mobile',
  [int] $Port = 0,
  [switch] $KeepEnvironment
)

$ErrorActionPreference = 'Stop'
$repo = Split-Path -Parent $PSScriptRoot
$frontend = Join-Path $repo 'frontend'
$capture = Join-Path $PSScriptRoot 'ui-screenshots.mjs'
if (-not $OutDir) { $OutDir = Join-Path ([IO.Path]::GetTempPath()) ("ccma-ui-shots-" + (Get-Date -Format 'yyyyMMdd-HHmmss')) }
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

function New-Secret([int] $length) {
  $chars = 'abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789'.ToCharArray()
  $bytes = [byte[]]::new($length)
  # Create().GetBytes works on both Windows PowerShell 5.1 and PowerShell 7.
  $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
  try { $rng.GetBytes($bytes) } finally { $rng.Dispose() }
  -join ($bytes | ForEach-Object { $chars[$_ % $chars.Length] })
}

function Wait-Http([string] $url, [int] $seconds) {
  $deadline = (Get-Date).AddSeconds($seconds)
  while ((Get-Date) -lt $deadline) {
    try { Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 3 | Out-Null; return } catch { Start-Sleep -Milliseconds 700 }
  }
  throw "Timed out waiting for $url"
}

if ($Mode -eq 'preview') {
  if ($Port -eq 0) { $Port = 4173 }
  Push-Location $frontend
  try {
    npm run build | Out-Host
    if ($LASTEXITCODE -ne 0) { throw 'frontend build failed' }
    $server = Start-Process -FilePath 'npx.cmd' -ArgumentList @('vite', 'preview', '--port', "$Port", '--strictPort', '--host', '127.0.0.1') -PassThru -WindowStyle Hidden
  } finally { Pop-Location }
  try {
    Wait-Http "http://127.0.0.1:$Port/" 60
    $routes = 'design-system=/wireframes/design-system,design-system-dialog=/wireframes/design-system?dialog=review,design-system-confirm=/wireframes/design-system?dialog=confirm'
    node $capture --base "http://127.0.0.1:$Port" --out $OutDir --routes $routes --themes $Themes --viewports $Viewports
    $code = $LASTEXITCODE
  } finally {
    # npx starts a child node process; stop the whole tree.
    & taskkill /PID $server.Id /T /F 2>$null | Out-Null
  }
  Write-Host "Screenshots: $OutDir"
  exit $code
}

# ---- app mode: disposable Compose project ----
if ($Port -eq 0) { $Port = 18088 }
$project = 'ccma-uishot-' + (Get-Date -Format 'yyyyMMddHHmmss')
if ($project -eq 'ccma') { throw 'refusing to use the persistent project name' }
$compose = Join-Path $PSScriptRoot 'ui-screenshots/compose.yml'
$envFile = Join-Path ([IO.Path]::GetTempPath()) "$project.env"
$tokenFile = Join-Path ([IO.Path]::GetTempPath()) "$project.token"
@(
  "UISHOT_DB_ROOT_PASSWORD=$(New-Secret 24)",
  "UISHOT_DB_PASSWORD=$(New-Secret 24)",
  "UISHOT_JWT_SECRET=$(New-Secret 48)",
  "UISHOT_ENCRYPTION_KEY=$(New-Secret 32)",
  "UISHOT_PORT=$Port"
) | Set-Content -Path $envFile -Encoding ascii
$dc = @('compose', '-p', $project, '-f', $compose, '--env-file', $envFile)
$base = "http://127.0.0.1:$Port"
$code = 2

try {
  & docker @dc up -d --build | Out-Host
  if ($LASTEXITCODE -ne 0) { throw 'docker compose up failed' }
  Wait-Http "$base/api/v1/setup/status" 180

  # Throwaway admin; the password never leaves this process.
  $setup = @{ email = 'ui-shots@example.invalid'; password = (New-Secret 20) + 'aA1!'; name = 'UI Screenshots'; workspace_name = 'Cửa hàng mẫu' } | ConvertTo-Json
  $tok = Invoke-RestMethod -Method Post -Uri "$base/api/v1/setup" -ContentType 'application/json; charset=utf-8' -Body ([Text.Encoding]::UTF8.GetBytes($setup))
  $headers = @{ Authorization = "Bearer $($tok.access_token)" }
  Set-Content -Path $tokenFile -Value $tok.access_token -Encoding ascii -NoNewline
  $tenant = @((Invoke-RestMethod -Uri "$base/api/v1/tenants" -Headers $headers) | ForEach-Object { $_ })[0].id
  Invoke-RestMethod -Method Post -Uri "$base/api/v1/tenants/$tenant/demo/import" -Headers $headers | Out-Null

  # Two synthetic snapshots in the disposable DB so the source states render:
  # a valid manifest listing a message that does not exist -> changed_since_analysis;
  # a manifest whose digest does not match its bytes -> verification_unavailable.
  $seed = @'
SET NAMES utf8mb4;
SELECT tenant_id, job_run_id, conversation_id INTO @t, @run, @c FROM job_results
  WHERE result_type = 'conversation_evaluation' AND severity <> 'PASS' ORDER BY created_at, id LIMIT 1;
SET @m := CONCAT('{"schema_version":"ccma.snapshot.v1","tenant_id":"', @t, '","conversation_id":"', @c,
  '","coverage":"complete","coverage_reasons":[],"omitted_earlier_messages":0,"messages":[{"message_id":"00000000-0000-4000-8000-000000000001","external_message_id":"synthetic-removed","sender_type":"agent","sender_name":"Synthetic","content_type":"text","sent_at":"2020-01-01T00:00:00Z","content_sha256":"',
  SHA2('synthetic', 256), '","content_code_points":9,"attachment_coverage":"none","attachment_count":0}]}');
SET @s := UUID();
INSERT INTO analysis_snapshots (id, tenant_id, job_run_id, conversation_id, schema_version, digest, coverage, coverage_reasons, message_count, manifest, created_at)
  VALUES (@s, @t, @run, @c, 'ccma.snapshot.v1', SHA2(@m, 256), 'complete', '[]', 1, @m, NOW());
UPDATE job_results SET analysis_snapshot_id = @s WHERE job_run_id = @run AND conversation_id = @c;
SELECT tenant_id, job_run_id, conversation_id INTO @t2, @run2, @c2 FROM job_results
  WHERE result_type = 'conversation_evaluation' AND severity = 'PASS' AND analysis_snapshot_id IS NULL ORDER BY created_at, id LIMIT 1;
SET @m2 := REPLACE(REPLACE(@m, @c, @c2), @t, @t2);
SET @s2 := UUID();
INSERT INTO analysis_snapshots (id, tenant_id, job_run_id, conversation_id, schema_version, digest, coverage, coverage_reasons, message_count, manifest, created_at)
  VALUES (@s2, @t2, @run2, @c2, 'ccma.snapshot.v1', REPEAT('0', 64), 'complete', '[]', 1, @m2, NOW());
UPDATE job_results SET analysis_snapshot_id = @s2 WHERE job_run_id = @run2 AND conversation_id = @c2;
'@
  $seed | & docker @dc exec -T db sh -c 'MYSQL_PWD="$MYSQL_PASSWORD" mysql -u"$MYSQL_USER" CCMA' | Out-Host
  if ($LASTEXITCODE -ne 0) { throw 'synthetic snapshot seed failed' }

  # Windows PowerShell 5.1 returns a JSON array as one pipeline object; unroll it.
  $jobs = @((Invoke-RestMethod -Uri "$base/api/v1/tenants/$tenant/jobs" -Headers $headers) | ForEach-Object { $_ })
  $qc = $jobs | Where-Object { $_.job_type -eq 'qc_analysis' } | Select-Object -First 1
  $cls = $jobs | Where-Object { $_.job_type -eq 'classification' } | Select-Object -First 1
  $channels = @((Invoke-RestMethod -Uri "$base/api/v1/tenants/$tenant/channels" -Headers $headers) | ForEach-Object { $_ })

  $routes = @(
    "dashboard=/$tenant",
    "channels=/$tenant/channels",
    "messages=/$tenant/messages",
    "jobs=/$tenant/jobs",
    "results=/$tenant/results",
    "settings=/$tenant/settings"
  )
  if ($qc) { $routes += "jobdetail-qc=/$tenant/jobs/$($qc.id)" }
  if ($cls) { $routes += "jobdetail-classification=/$tenant/jobs/$($cls.id)" }
  if ($channels.Count -gt 0) { $routes += "channel-detail=/$tenant/channels/$($channels[0].id)" }

  node $capture --base $base --out $OutDir --routes ($routes -join ',') --themes $Themes --viewports $Viewports --token-file $tokenFile --wait 2500
  $code = $LASTEXITCODE
} finally {
  if (-not $KeepEnvironment) {
    # docker writes progress to stderr; do not let that surface as a PowerShell error.
    $ErrorActionPreference = 'Continue'
    & docker @dc down -v --remove-orphans --rmi local 2>&1 | ForEach-Object { "$_" } | Out-Host
    Remove-Item -Force -ErrorAction SilentlyContinue $envFile, $tokenFile
  } else {
    Write-Host "Kept project $project (env file $envFile). Remove with: docker compose -p $project -f $compose --env-file $envFile down -v"
  }
}
Write-Host "Screenshots: $OutDir"
exit $code
