param(
    [Parameter(Mandatory=$true)][string]$PackageRoot,
    [Parameter(Mandatory=$true)][string]$ManifestPath,
    [Parameter(Mandatory=$true)][string]$OutputPath
)
$ErrorActionPreference = 'Stop'
$startedAt = (Get-Date).ToUniversalTime().ToString('o')
$root = [System.IO.Path]::GetFullPath($PackageRoot).TrimEnd('\','/')
$expectedRoot = 'C:\Users\DELL\AppData\Local\Programs\Antigravity\resources\app.asar.unpacked\node_modules\chrome-devtools-mcp'
$expectedManifest = 'D:\UNG DUNG AI\TOOL AI 2026\CVF-Workspace\Customer-Care-Monitor\docs\reviews\probes\r144_a3_package_metadata_20261105\root_npm_full_tree_manifest.json'
$expectedOutputRoot = 'D:\UNG DUNG AI\TOOL AI 2026\CVF-Workspace\Customer-Care-Monitor\docs\reviews\probes\r150_a1_a2_windows_20261105'
if (-not [string]::Equals($root, $expectedRoot, [System.StringComparison]::OrdinalIgnoreCase)) { throw 'unexpected installed package path' }
if (-not [string]::Equals([System.IO.Path]::GetFullPath($ManifestPath), $expectedManifest, [System.StringComparison]::OrdinalIgnoreCase)) { throw 'unexpected manifest path' }
if (-not [string]::Equals([System.IO.Path]::GetFullPath($OutputPath), (Join-Path $expectedOutputRoot 'a3_prestart_full_tree.json'), [System.StringComparison]::OrdinalIgnoreCase)) { throw 'unexpected receipt path' }
$manifestBytes = [System.IO.File]::ReadAllBytes($expectedManifest)
$manifestText = [System.Text.Encoding]::UTF8.GetString($manifestBytes).Replace("`r`n", "`n")
$sha = [System.Security.Cryptography.SHA256]::Create()
try { $manifestHash = ([BitConverter]::ToString($sha.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($manifestText)))).Replace('-','').ToLowerInvariant() }
finally { $sha.Dispose() }
if ($manifestHash -ne '272c516aa8513252e3613748406fc49d77247650e2f8610ada6b49bd4a19b398') { throw 'manifest content hash mismatch' }
$rootItem = Get-Item -LiteralPath $root
if (($rootItem.Attributes -band [System.IO.FileAttributes]::ReparsePoint) -ne 0) { throw 'package root is a reparse point' }
foreach ($dir in Get-ChildItem -LiteralPath $root -Directory -Recurse -Force) {
    if (($dir.Attributes -band [System.IO.FileAttributes]::ReparsePoint) -ne 0) { throw 'reparse directory in package tree' }
}
$manifest = Get-Content -LiteralPath $ManifestPath -Raw -Encoding utf8 | ConvertFrom-Json
if ($manifest.fileCount -ne 348 -or $manifest.files.Count -ne 348) { throw 'unexpected npm manifest count' }
$expected = @{}
foreach ($entry in $manifest.files) { $expected[[string]$entry.path] = $entry }
$actual = @{}
foreach ($file in Get-ChildItem -LiteralPath $root -File -Recurse -Force) {
    $full = [System.IO.Path]::GetFullPath($file.FullName)
    if (-not $full.StartsWith($root + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) { throw 'path escaped package root' }
    if (($file.Attributes -band [System.IO.FileAttributes]::ReparsePoint) -ne 0) { throw 'reparse point in package tree' }
    $rel = $full.Substring($root.Length + 1).Replace('\','/')
    $actual[$rel] = [ordered]@{path=$rel; bytes=[long]$file.Length; sha256=(Get-FileHash -LiteralPath $full -Algorithm SHA256).Hash.ToLowerInvariant()}
}
$missing = @($expected.Keys | Where-Object { -not $actual.ContainsKey($_) } | Sort-Object)
$extra = @($actual.Keys | Where-Object { -not $expected.ContainsKey($_) } | Sort-Object)
$mismatch = @()
$otherMatched = 0
foreach ($rel in ($expected.Keys | Sort-Object)) {
    if ($rel -eq 'package.json' -or $rel -eq 'README.md' -or -not $actual.ContainsKey($rel)) { continue }
    $exp = $expected[$rel]; $act = $actual[$rel]
    if ([long]$exp.bytes -eq [long]$act.bytes -and [string]$exp.sha256 -eq [string]$act.sha256) { $otherMatched++ }
    else { $mismatch += $rel }
}
$packageHash = if ($actual.ContainsKey('package.json')) { $actual['package.json'].sha256 } else { $null }
$pass = ($actual.Count -eq 347 -and $missing.Count -eq 1 -and $missing[0] -eq 'README.md' -and $extra.Count -eq 0 -and $mismatch.Count -eq 0 -and $otherMatched -eq 346 -and $packageHash -eq '592ef654a308edcba5fb3c2b302173650096a9487e4120b96e7adbe830429b9f')
$receipt = [ordered]@{
    schemaVersion='ccma.r150.prestartFullTree@1.0.0'; capturedAt=$startedAt;
    packageRoot=$root; manifestPath=$ManifestPath; manifestLfSha256=$manifestHash; expectedCount=348; actualCount=$actual.Count;
    expectedPackageJsonSha256='592ef654a308edcba5fb3c2b302173650096a9487e4120b96e7adbe830429b9f';
    installedPackageJsonSha256=$packageHash; missing=$missing; extra=$extra;
    otherMatched=$otherMatched; otherMismatch=$mismatch; verdict=$(if($pass){'PASS'}else{'NO_START'});
    files=@($actual.Values | Sort-Object path)
}
$parent = Split-Path -Parent $OutputPath
New-Item -ItemType Directory -Path $parent -Force | Out-Null
$receipt | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $OutputPath -Encoding utf8
Write-Output ("A3_PRESTART {0} actual={1} otherMatched={2} packageJson={3} missing={4} extra={5} mismatch={6}" -f $receipt.verdict,$actual.Count,$otherMatched,$packageHash,($missing -join ','),($extra -join ','),($mismatch -join ','))
if ($pass) { exit 0 } else { exit 2 }
