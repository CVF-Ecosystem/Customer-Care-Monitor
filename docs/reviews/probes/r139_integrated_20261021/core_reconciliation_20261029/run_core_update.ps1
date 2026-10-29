param([switch]$Execute, [switch]$PreflightOnly, [string]$ExpectedSourceSha256)

# Source candidate only. Root must review and explicitly release one execution.
# No updater, claim, receipt or filesystem write without -Execute.
if ($Execute -and $PreflightOnly) { throw 'Choose only one mode' }
if (-not $Execute -and -not $PreflightOnly) { Write-Output 'SOURCE_ONLY: updater not invoked'; return }
$ErrorActionPreference = 'Stop'
$workspace = 'D:\UNG DUNG AI\TOOL AI 2026\CVF-Workspace'
$project = Join-Path $workspace 'Customer-Care-Monitor-AI'
$core = Join-Path $workspace '.Controlled-Vibe-Framework-CVF'
$backupRoot = Join-Path $workspace '_cvf-core-backups'
$backup = Join-Path $backupRoot 'r139-preimage-20261029T110155487556Z'
$probe = Join-Path $project 'docs\reviews\probes\r139_integrated_20261021\core_reconciliation_20261029'
$receiptPath = Join-Path $probe 'preimages.json'
$aclPath = Join-Path $probe 'preimage_acls.json'
$target = '15e39450853184eb65dbde73398c69c3171a6241'
$previous = 'b8ead4f0d1bf9224338a206b2abcd893ec1801b1'
$origin = 'https://github.com/Blackbird081/Controlled-Vibe-Framework-CVF.git'
$preimageHash = 'c4c0ddcdd3ddf8a89ae5718b612968f0cea7885354383328d1856073bb4cee51'
$aclHash = 'ea944b2377330218dc6774f6088ae75bed039d6cb7c75d399fb37caf5db915a7'
$grantHash = '6a357d91d2071a4237294c4e875c71f06b1516dd4dfbba8810db94ec5f141e5e'
$fetchHash = '3f74f861e718a148b63623a037ad15f812e727aa9ddb3d0a28ebf128acbab721'
$fetchVerificationHash = '91893304ee7bf19d18017349faaa7a2dd2c97374573c2e7cdc47e795e03974d5'

function Assert-PathSafe([string]$Path) {
    $full = [IO.Path]::GetFullPath($Path)
    $w = [IO.Path]::GetFullPath($workspace).TrimEnd('\')
    if ($full -ine $w -and -not $full.StartsWith($w + '\', [StringComparison]::OrdinalIgnoreCase)) { throw "Outside workspace: $full" }
    $item = $full
    while ($item) {
        if (Test-Path -LiteralPath $item) {
            if (((Get-Item -LiteralPath $item -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) { throw "Reparse path: $item" }
        }
        if ($item -ieq [IO.Path]::GetPathRoot($item)) { break }
        $item = [IO.Directory]::GetParent($item).FullName
    }
    return $full
}
function Hash([string]$Path) { return (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant() }
function Git([string]$GitRoot, [string[]]$GitArgs) {
    $gitExe = (Get-Command git.exe -CommandType Application -ErrorAction Stop).Source
    $ErrorActionPreference = 'Continue' # Native Git writes normal progress to stderr in Windows PowerShell 5.1.
    $value = & $gitExe -C $GitRoot @GitArgs 2>&1
    if ($LASTEXITCODE -ne 0) { throw "Git verification failed: $($GitArgs[0])" }
    return ($value | Out-String).Trim()
}
function Descendants([int]$RootPid) {
    $queue = [Collections.Generic.Queue[int]]::new(); $queue.Enqueue($RootPid)
    $found = @(); $seen = @{}
    while ($queue.Count -gt 0) {
        $parent = $queue.Dequeue()
        foreach ($p in @(Get-CimInstance Win32_Process -Filter "ParentProcessId=$parent" -ErrorAction Stop)) {
            $id = [int]$p.ProcessId
            if (-not $seen.ContainsKey($id)) {
                $seen[$id] = $true
                $found += [pscustomobject]@{ pid=$id; parent=[int]$p.ParentProcessId; created=$p.CreationDate; executable=$p.ExecutablePath }
                if ($found.Count -gt 64) { throw 'Descendant observation cap exceeded' }
                $queue.Enqueue($id)
            }
        }
    }
    return $found
}

if ((Get-Location).ProviderPath -ine $project) { throw 'Run from project root' }
if (@(Get-ChildItem Env: | Where-Object { $_.Name -like 'GIT_*' }).Count -ne 0) { throw 'Inherited GIT environment must be cleared before dispatch' }
if ($ExpectedSourceSha256 -notmatch '^[0-9a-f]{64}$' -or (Hash $PSCommandPath) -ne $ExpectedSourceSha256) { throw 'Reviewed source pin required' }
foreach ($p in @($workspace,$project,$core,$backupRoot,$backup,$probe,$receiptPath,$aclPath)) { $null = Assert-PathSafe $p }
if ((Hash $receiptPath) -ne $preimageHash -or (Hash $aclPath) -ne $aclHash) { throw 'Preimage or ACL receipt changed' }
$fetchReceipt = Assert-PathSafe (Join-Path $probe 'fetch.json')
$fetchVerification = Assert-PathSafe (Join-Path $probe 'fetch_verification.json')
if ((Hash $fetchReceipt) -ne $fetchHash -or (Hash $fetchVerification) -ne $fetchVerificationHash) { throw 'Public fetch provenance changed' }
$receipt = Get-Content -LiteralPath $receiptPath -Raw | ConvertFrom-Json
$acls = Get-Content -LiteralPath $aclPath -Raw | ConvertFrom-Json
$grantPath = Assert-PathSafe (Join-Path $project 'CVF_SESSION\authority\CCMAI-RUNTIME-139-CORE-RECONCILIATION-SCOPE-ADMISSION.json')
if ((Hash $grantPath) -ne $grantHash) { throw 'Grant bytes changed' }
$committedGrantHash = & python -B -c "import sys,subprocess,hashlib;print(hashlib.sha256(subprocess.check_output(['git','-C',sys.argv[1],'show','0812d972d7d719b3151506e321f651d51cc3ef1e:CVF_SESSION/authority/CCMAI-RUNTIME-139-CORE-RECONCILIATION-SCOPE-ADMISSION.json'])).hexdigest())" $project
if ($LASTEXITCODE -ne 0 -or $committedGrantHash.Trim() -ne $grantHash) { throw 'Committed grant differs' }
$grant = Get-Content -LiteralPath $grantPath -Raw | ConvertFrom-Json
if ($receipt.backupRoot -ine $backup -or @($receipt.preimages).Count -ne 44 -or $acls.Count -ne 40) { throw 'Receipt shape mismatch' }
$sharedExpected = @($grant.sharedExactFiles) + @($grant.sharedObservedPreserve[0])
$recordShared = @($receipt.preimages | Where-Object { $_.kind -eq 'shared' })
if ($recordShared.Count -ne $sharedExpected.Count) { throw 'Shared preimage count mismatch' }
foreach ($rel in $sharedExpected) {
    $expectedPath = Assert-PathSafe (Join-Path $workspace $rel)
    if (@($recordShared | Where-Object { $_.path -ieq $expectedPath }).Count -ne 1) { throw "Shared target mismatch: $rel" }
}
foreach ($entry in $receipt.preimages) {
    $path = Assert-PathSafe $entry.path
    $isMutableSource = $entry.kind -eq 'project' -and ($path -ieq (Join-Path $project 'scripts\cvf_downstream_gate.py') -or $path -ieq (Join-Path $project 'scripts\tests\test_cvf_downstream_gate_core_repin.py'))
    if ($entry.state -eq 'ABSENT') {
        if (-not $isMutableSource -and (Test-Path -LiteralPath $path)) { throw "Unexpected present target: $path" }
        continue
    }
    if ($entry.state -ne 'PRESENT') { throw 'Unknown preimage state' }
    $copy = Assert-PathSafe $entry.backup
    if (-not $copy.StartsWith($backup + '\', [StringComparison]::OrdinalIgnoreCase) -or -not (Test-Path -LiteralPath $copy -PathType Leaf) -or (Hash $copy) -ne $entry.sha256) { throw "Backup mismatch: $path" }
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { throw "Preexisting source absent: $path" }
    $match = @($acls | Where-Object { $_.path -ieq $path })
    if ($match.Count -ne 1 -or (Get-Acl -LiteralPath $path).Sddl -cne $match[0].sddl) { throw "ACL mismatch: $path" }
    if (-not $isMutableSource) {
        if ((Hash $path) -ne $entry.sha256) { throw "Original mismatch: $path" }
    }
}
foreach ($s in $receipt.siblingManifestHashes) {
    $p = Assert-PathSafe $s.path
    if (-not (Test-Path -LiteralPath $p -PathType Leaf) -or (Hash $p) -ne $s.sha256) { throw "Sibling manifest mismatch: $p" }
}
if (@($receipt.siblingManifestHashes).Count -ne 15) { throw 'Sibling manifest inventory mismatch' }
$currentManifests = @(Get-ChildItem -LiteralPath $workspace -Directory -Force | ForEach-Object {
    $m = Join-Path $_.FullName '.cvf\manifest.json'
    if (Test-Path -LiteralPath $m -PathType Leaf) { Assert-PathSafe $m }
} | Sort-Object)
if ((@($receipt.siblingManifestHashes | ForEach-Object { $_.path } | Sort-Object) -join "`n") -cne ($currentManifests -join "`n")) { throw 'Sibling manifest set changed' }
$active = Get-Content -LiteralPath (Join-Path $workspace 'CVF_RULE_PACKS\ACTIVE_RULE_PACK.json') -Raw | ConvertFrom-Json
if ($active.activeProfile -ne 'operator-local') { throw 'Active profile changed' }
if ((Git $core @('rev-parse','HEAD')) -ne $previous -or (Git $core @('rev-parse','origin/main')) -ne $target -or
    (Git $core @('rev-parse','FETCH_HEAD')) -ne $target -or (Git $core @('remote','get-url','origin')) -ne $origin -or
    (Git $core @('status','--porcelain')) -ne '') { throw 'Core preconditions differ' }
$updater = Assert-PathSafe (Join-Path $core 'scripts\update_cvf_workspace_public_core.ps1')
$mirror = Assert-PathSafe (Join-Path $backupRoot 'r139-reviewed-public-15e39450.git')
$support = Assert-PathSafe (Join-Path $backupRoot 'r139-reviewed-public-15e39450-support')
if ((Test-Path -LiteralPath $mirror) -or (Test-Path -LiteralPath $support)) { throw 'Mirror or support collision' }
foreach ($rel in @('.git\shallow','.git\objects\info\alternates','.git\info\grafts')) {
    if (Test-Path -LiteralPath (Assert-PathSafe (Join-Path $core $rel))) { throw "Unsupported source object mechanism: $rel" }
}
if ((Git $core @('rev-parse','--is-shallow-repository')) -ne 'false' -or (Git $core @('for-each-ref','--format=%(refname)','refs/replace')) -ne '') { throw 'Shallow or replacement refs forbidden' }
$sourceTree = Git $core @('rev-parse',($target + '^{tree}'))
if ($sourceTree -ne '27c7d337155774917dc9fa026edef25e0294b4cb') { throw 'Public target tree mismatch' }
$sourceListing = Git $core @('ls-tree','-r','--full-tree',$target)
$null = Git $core @('fsck','--full','--strict')
$existing = @(Get-ChildItem -LiteralPath $backupRoot -Directory -Force | ForEach-Object { $_.FullName })
$collisionScanTime = [DateTime]::Now
$validatedBackupRoot = Assert-PathSafe $backupRoot
foreach ($second in 0..610) {
    $candidate = Join-Path $backupRoot ('.Controlled-Vibe-Framework-CVF-' + $collisionScanTime.AddSeconds($second).ToString('yyyyMMdd-HHmmss'))
    $candidate = [IO.Path]::GetFullPath($candidate)
    if (-not $candidate.StartsWith($validatedBackupRoot.TrimEnd('\') + '\', [StringComparison]::OrdinalIgnoreCase)) { throw 'Backup candidate outside root' }
    if (Test-Path -LiteralPath $candidate) { throw 'Possible timestamp backup collision' }
}

$claim = Assert-PathSafe (Join-Path $probe 'update_attempt.claim')
$stdout = Join-Path $probe 'updater_stdout.txt'; $stderr = Join-Path $probe 'updater_stderr.txt'; $resultFile = Join-Path $probe 'updater_result.json'
foreach ($p in @($claim,$stdout,$stderr,$resultFile)) { $null = Assert-PathSafe $p; if (Test-Path -LiteralPath $p) { throw "Evidence path collision: $p" } }
if ($PreflightOnly) { Write-Output 'PREFLIGHT_PASS: complete read-only checks; no claim, mirror or updater'; return }
# CreateNew is a permanent attempt latch: a timeout/failure is never retried.
$fs = [IO.File]::Open($claim, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
try { $bytes = [Text.Encoding]::UTF8.GetBytes("one updater attempt`n"); $fs.Write($bytes,0,$bytes.Length) } finally { $fs.Dispose() }
$result = [ordered]@{ startUtc=[DateTime]::UtcNow.ToString('o'); updaterPid=$null; exitCode=$null; endUtc=$null; status='STARTING'; observedChildren=@(); childObservation='NOT_STARTED'; transport='LOCAL_MIRROR_FROM_PREVIOUS_PUBLIC_FETCH'; sourceTree=$sourceTree; note='Sampled child inventory is not exhaustive; timeout or observer failure remains UNKNOWN. Mirror file handles limit concurrent writes; same-user races are not excluded.' }
$known = @{}
$launchAttempted = $false; $closureConfirmed = $false
$locks = @(); $configuredGitEnvironment = $false
try {
    $null = New-Item -ItemType Directory -Path $support -ErrorAction Stop
    $template = Join-Path $support 'empty-template'; $hooks = Join-Path $support 'empty-hooks'
    $null = New-Item -ItemType Directory -Path $template,$hooks -ErrorAction Stop
    # Wrapper process environment only; caller environment is never changed.
    $env:GIT_CONFIG_NOSYSTEM='1'; $env:GIT_CONFIG_GLOBAL='NUL'; $env:GIT_TERMINAL_PROMPT='0'
    $env:GIT_NO_REPLACE_OBJECTS='1'; $env:GIT_NO_LAZY_FETCH='1'; $env:GIT_ATTR_NOSYSTEM='1'
    $env:GIT_TEMPLATE_DIR=$template; $env:GIT_CONFIG_COUNT='4'
    $env:GIT_CONFIG_KEY_0='url.'+([Uri]$mirror).AbsoluteUri+'.insteadOf'; $env:GIT_CONFIG_VALUE_0=$origin
    $env:GIT_CONFIG_KEY_1='protocol.allow'; $env:GIT_CONFIG_VALUE_1='never'
    $env:GIT_CONFIG_KEY_2='protocol.file.allow'; $env:GIT_CONFIG_VALUE_2='always'
    $env:GIT_CONFIG_KEY_3='core.hooksPath'; $env:GIT_CONFIG_VALUE_3=$hooks
    $configuredGitEnvironment = $true
    $null = Git $project @('clone','--mirror','--no-local',$core,$mirror)
    $mirrorRefs = @((Git $mirror @('for-each-ref','--format=%(refname)')) -split "`r?`n" | Where-Object { $_ })
    foreach ($ref in $mirrorRefs) { if ($ref -ne 'refs/heads/main') { $null = Git $mirror @('update-ref','-d',$ref) } }
    $null = Git $mirror @('update-ref','refs/heads/main',$target)
    $null = Git $mirror @('symbolic-ref','HEAD','refs/heads/main')
    foreach ($rel in @('shallow','objects\info\alternates','info\grafts')) { if (Test-Path -LiteralPath (Join-Path $mirror $rel)) { throw "Unsupported mirror object mechanism: $rel" } }
    if ((Git $mirror @('rev-parse','HEAD')) -ne $target -or (Git $mirror @('rev-parse','refs/heads/main')) -ne $target -or
        (Git $mirror @('rev-parse',($target + '^{tree}'))) -ne $sourceTree -or (Git $mirror @('ls-tree','-r','--full-tree',$target)) -cne $sourceListing -or
        (Git $mirror @('for-each-ref','--format=%(refname)','refs/replace')) -ne '') { throw 'Mirror target or object listing mismatch' }
    $null = Git $mirror @('fsck','--full','--strict')
    $mirrorFiles = @(Get-ChildItem -LiteralPath $mirror -Recurse -File -Force)
    if ($mirrorFiles.Count -eq 0) { throw 'Empty mirror' }
    $mirrorHashes = @{}
    foreach ($f in $mirrorFiles) {
        $path = Assert-PathSafe $f.FullName
        $handle = [IO.File]::Open($path,[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::Read)
        $locks += $handle; $mirrorHashes[$path] = Hash $path
    }
    $result.mirrorPath = $mirror; $result.mirrorFiles = $mirrorHashes
    if ((Git $mirror @('rev-parse','HEAD')) -ne $target -or (Git $mirror @('ls-tree','-r','--full-tree',$target)) -cne $sourceListing) { throw 'Locked mirror differs from reviewed source' }
    $null = Git $mirror @('fsck','--full','--strict')
    $collisionScanTime = [DateTime]::Now
    foreach ($second in 0..610) {
        $candidate = [IO.Path]::GetFullPath((Join-Path $backupRoot ('.Controlled-Vibe-Framework-CVF-' + $collisionScanTime.AddSeconds($second).ToString('yyyyMMdd-HHmmss'))))
        if (-not $candidate.StartsWith($validatedBackupRoot.TrimEnd('\') + '\', [StringComparison]::OrdinalIgnoreCase) -or (Test-Path -LiteralPath $candidate)) { throw 'Possible backup collision' }
    }
    if (([DateTime]::Now - $collisionScanTime).TotalSeconds -gt 20) { throw 'Collision scan stale; do not start updater' }
    $arg = @('-NoProfile','-ExecutionPolicy','Bypass','-File',('"' + $updater + '"'),'-WorkspaceRoot',('"' + $workspace + '"'))
    $launchAttempted = $true
    $proc = Start-Process -FilePath 'powershell.exe' -ArgumentList $arg -WorkingDirectory $project -RedirectStandardOutput $stdout -RedirectStandardError $stderr -PassThru -WindowStyle Hidden
    $null = $proc.Handle # Retain the original process handle before it can exit.
    $result.updaterPid = $proc.Id
    $deadline = [DateTime]::UtcNow.AddSeconds(600)
    $observerError = $null
    while (-not $proc.HasExited -and [DateTime]::UtcNow -lt $deadline) {
        try { foreach ($child in @(Descendants $proc.Id)) { $known["$($child.pid)|$($child.created)"] = $child } }
        catch { $observerError = $_.Exception.Message; break }
        Start-Sleep -Milliseconds 500
        $proc.Refresh()
    }
    $result.observedChildren = @($known.Values)
    if ($observerError -or -not $proc.HasExited) {
        $result.status = 'UNKNOWN_WRITER_MAY_BE_ACTIVE'; $result.childObservation = $observerError
        throw 'Updater exit/child observation unavailable; no kill, retry or restore'
    }
    $proc.WaitForExit(); $result.exitCode = $proc.ExitCode; $result.endUtc = [DateTime]::UtcNow.ToString('o')
    foreach ($child in $known.Values) {
        try { $live = @(Get-CimInstance Win32_Process -Filter "ProcessId=$($child.pid)" -ErrorAction Stop) }
        catch { $result.status = 'UNKNOWN_WRITER_MAY_BE_ACTIVE'; throw 'Child absence observation failed; no restore' }
        if ($live.Count -gt 0 -and "$($live[0].CreationDate)" -eq "$($child.created)") { $result.status = 'UNKNOWN_WRITER_MAY_BE_ACTIVE'; throw 'Known updater descendant remains alive; no restore' }
    }
    $result.childObservation = 'KNOWN_IDENTITIES_ABSENT_SAMPLED_NOT_EXHAUSTIVE'
    $closureConfirmed = $true
    foreach ($f in $mirrorHashes.Keys) { if ((Hash $f) -ne $mirrorHashes[$f]) { throw 'Mirror bytes changed' } }
    $afterFiles = @(Get-ChildItem -LiteralPath $mirror -Recurse -File -Force | ForEach-Object { Assert-PathSafe $_.FullName })
    if ($afterFiles.Count -ne $mirrorHashes.Count -or @($afterFiles | Where-Object { -not $mirrorHashes.ContainsKey($_) }).Count -ne 0) { throw 'Mirror file set changed' }
    if ((Git $mirror @('rev-parse','HEAD')) -ne $target -or (Git $mirror @('ls-tree','-r','--full-tree',$target)) -cne $sourceListing) { throw 'Mirror postcheck mismatch' }
    $null = Git $mirror @('fsck','--full','--strict')
    # Remove all temporary Git settings before querying the actual stored origin.
    Get-ChildItem Env: | Where-Object { $_.Name -like 'GIT_*' } | ForEach-Object { Remove-Item -LiteralPath ('Env:' + $_.Name) }
    $configuredGitEnvironment = $false
    if ($proc.ExitCode -ne 0) { throw 'Updater failed; no automatic restore or retry' }
    $newDirs = @(Get-ChildItem -LiteralPath $backupRoot -Directory -Force | Where-Object { $_.FullName -notin $existing -and $_.Name -match '^\.Controlled-Vibe-Framework-CVF-\d{8}-\d{6}$' })
    if ($newDirs.Count -ne 1) { throw 'Original Core backup identity ambiguous' }
    $oldCore = Assert-PathSafe $newDirs[0].FullName
    $result.originalCoreBackup = $oldCore
    if ((Git $oldCore @('rev-parse','HEAD')) -ne $previous -or (Git $oldCore @('status','--porcelain')) -ne '') { throw 'Original Core backup mismatch' }
    if ((Git $core @('rev-parse','HEAD')) -ne $target -or (Git $core @('rev-parse','origin/main')) -ne $target -or (Git $core @('config','--local','--get','remote.origin.url')) -ne $origin -or (Git $core @('status','--porcelain')) -ne '') { throw 'Clone target, stored origin or clean state mismatch' }
    foreach ($rel in @('AGENTS.md','AGENT_HANDOFF.md','docs\reference\CVF_WORKSPACE_RULES.md','governance\toolkit\05_OPERATION\CVF_DOWNSTREAM_AGENTS_TEMPLATE.md','scripts\check_cvf_workspace_agent_enforcement.ps1','scripts\install_cvf_workspace_root_wrappers.ps1','scripts\sync_cvf_workspace_public_profile.ps1','scripts\get_cvf_workspace_status.ps1','scripts\repair_cvf_workspace.ps1','scripts\manage_cvf_workspace.ps1','scripts\ingest_cvf_downstream_knowledge.ps1','scripts\new-cvf-workspace.ps1','scripts\update_cvf_workspace_public_core.ps1','scripts\write_cvf_workspace_web_evidence_bridge.ps1','scripts\lib\downstream_catalog\CvfDownstreamCatalogLib.ps1','scripts\lib\downstream_catalog\CvfDownstreamBootstrapContent.ps1','scripts\lib\downstream_catalog\CvfWorkspaceDoctorLiveReadiness.ps1','scripts\lib\downstream_catalog\manage_cvf_downstream_catalog.ps1','scripts\lib\downstream_catalog\schemas\ARTIFACT_REGISTRY.schema.json','scripts\lib\downstream_catalog\schemas\MODULE_REGISTRY.schema.json','governance\toolkit\05_OPERATION\downstream_catalog\CVF_DOWNSTREAM_CATALOG_GUARD.md','scripts\lib\downstream_governance\cvf_downstream_gate_runner.py')) {
        if (-not (Test-Path -LiteralPath (Assert-PathSafe (Join-Path $core $rel)) -PathType Leaf)) { throw "Core kit missing: $rel" }
    }
    foreach ($s in $receipt.siblingManifestHashes) { if ((Hash (Assert-PathSafe $s.path)) -ne $s.sha256) { throw "Sibling changed: $($s.path)" } }
    $postManifests = @(Get-ChildItem -LiteralPath $workspace -Directory -Force | ForEach-Object { $m=Join-Path $_.FullName '.cvf\manifest.json'; if (Test-Path -LiteralPath $m -PathType Leaf) { Assert-PathSafe $m } } | Sort-Object)
    if (($postManifests -join "`n") -cne ($currentManifests -join "`n")) { throw 'Sibling manifest set changed' }
    $activeEntry = @($recordShared | Where-Object { $_.path -ieq (Join-Path $workspace 'CVF_RULE_PACKS\ACTIVE_RULE_PACK.json') })
    if ($activeEntry.Count -ne 1 -or (Hash $activeEntry[0].path) -ne $activeEntry[0].sha256) { throw 'Active profile bytes changed' }
    if ((Get-Content -LiteralPath (Join-Path $workspace 'CVF_RULE_PACKS\ACTIVE_RULE_PACK.json') -Raw | ConvertFrom-Json).activeProfile -ne 'operator-local') { throw 'Active profile changed after updater' }
    foreach ($entry in $recordShared) {
        $path = Assert-PathSafe $entry.path
        if ($entry.path -match '(Get-CVF-Workspace-OverlayProfiles\.ps1|Update-CVF-Workspace-Overlay\.ps1|CVF_WORKSPACE_OVERLAY_STATUS\.json)$') {
            if (Test-Path -LiteralPath $path) { throw "Unexpected overlay output: $path" }
        } elseif (-not (Test-Path -LiteralPath $path -PathType Leaf)) { throw "Shared output absent: $path" }
    }
    $result.status = 'UPDATER_RETURNED_POSTCHECKS_PASS_REVIEW_REQUIRED'
} catch {
    if ($launchAttempted -and -not $closureConfirmed) { $result.status = 'UNKNOWN_WRITER_MAY_BE_ACTIVE' } else { $result.status = 'STOP_REVIEW_REQUIRED' }
    $result.error = $_.Exception.Message
} finally {
    foreach ($h in $locks) { $h.Dispose() }
    if ($configuredGitEnvironment) { Get-ChildItem Env: | Where-Object { $_.Name -like 'GIT_*' } | ForEach-Object { Remove-Item -LiteralPath ('Env:' + $_.Name) } }
    $result | ConvertTo-Json -Depth 7 | Set-Content -LiteralPath $resultFile -Encoding utf8
}
if ($result.status -ne 'UPDATER_RETURNED_POSTCHECKS_PASS_REVIEW_REQUIRED') { exit 1 }
