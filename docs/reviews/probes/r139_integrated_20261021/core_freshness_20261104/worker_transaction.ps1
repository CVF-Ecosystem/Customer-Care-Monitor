param([switch]$PreflightOnly, [switch]$Execute, [string]$ExpectedSourceSha256)

# SOURCE CANDIDATE ONLY. Root and independent review must precede invocation.
# Default invocation is inert. No project adoption, installer, doctor or rollback.
if ($Execute -and $PreflightOnly) { throw 'Choose one mode' }
if (-not $Execute -and -not $PreflightOnly) { Write-Output 'SOURCE_ONLY'; return }
$ErrorActionPreference = 'Stop'
if ($PSVersionTable.PSVersion -lt [version]'7.4') { throw 'pwsh 7.4+ child-environment support required' }
$workspace = 'D:\UNG DUNG AI\TOOL AI 2026\CVF-Workspace'
$project = Join-Path $workspace 'Customer-Care-Monitor-AI'
$core = Join-Path $workspace '.Controlled-Vibe-Framework-CVF'
$backupRoot = Join-Path $workspace '_cvf-core-backups'
$probe = Join-Path $project 'docs\reviews\probes\r139_integrated_20261021\core_freshness_20261104'
$mirror = Join-Path $backupRoot 'r139-c4-reviewed-mirror-38952df8.git'
$support = Join-Path $backupRoot 'r139-c4-reviewed-support-38952df8'
$source = '15e39450853184eb65dbde73398c69c3171a6241'
$target = 'c4c8767ac8861854f40d466ba338ffd11eb9558e'
$preparation = '38952df8bcf9e01b18a05a1da1095b1e7eda6fd8'
$origin = 'https://github.com/Blackbird081/Controlled-Vibe-Framework-CVF.git'
$preimagesPath = Join-Path $probe 'root_preimages.json'
$ExpectedPreimagesSha256 = '848d6c1ffee3362fd17b032e11f802172951663793ac7d3eb2dcf46e891b0c42'
$authorityPath = Join-Path $project 'CVF_SESSION\authority\CCMAI-RUNTIME-139-CORE-FRESHNESS-20261104-ADMISSION.json'
$authoritySha256 = 'd2a6cc327c93c0a3ee7f98a9ec7614d7c929148af650fbce6f764b2fb3413dac'
$authorityGitLfSha256 = 'c756b41ce4e9b953bde25c4f25e29b82afa4cf2220b60c60aa0667df1d0b5222'
$claim = Join-Path $probe 'root_transaction_attempt.claim'
$stdout = Join-Path $probe 'root_updater.stdout.txt'
$stderr = Join-Path $probe 'root_updater.stderr.txt'
$resultPath = Join-Path $probe 'root_transaction_result.json'
$updater = Join-Path $core 'scripts\update_cvf_workspace_public_core.ps1'
$template = Join-Path $support 'empty-template'
$hooks = Join-Path $support 'empty-hooks'

function Safe([string]$Path) {
    $full = [IO.Path]::GetFullPath($Path)
    $root = [IO.Path]::GetFullPath($workspace).TrimEnd('\')
    if ($full -ine $root -and -not $full.StartsWith($root + '\', [StringComparison]::OrdinalIgnoreCase)) { throw "Outside workspace: $full" }
    $ancestor = $full
    while ($ancestor) {
        if ((Test-Path -LiteralPath $ancestor) -and ((Get-Item -LiteralPath $ancestor -Force).Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw "Reparse path: $ancestor" }
        $parent = [IO.Directory]::GetParent($ancestor)
        if ($null -eq $parent) { break }
        $ancestor = $parent.FullName
    }
    return $full
}
function Hash([string]$Path) { return (Get-FileHash -LiteralPath (Safe $Path) -Algorithm SHA256).Hash.ToLowerInvariant() }
function Pinned([string]$Path, [string]$Digest) {
    if ($Digest -cnotmatch '^[0-9a-f]{64}$' -or (Hash $Path) -cne $Digest) { throw "Pin mismatch: $Path" }
}
function Json([string]$Path) {
    # System.Text.Json enumerates duplicate properties rather than silently merging.
    $raw = [IO.File]::ReadAllText((Safe $Path))
    if ($raw.Length -gt 1048576) { throw 'JSON size cap' }
    $doc = [Text.Json.JsonDocument]::Parse($raw)
    function Unique($Element) {
        if ($Element.ValueKind -eq [Text.Json.JsonValueKind]::Object) {
            $seen = [Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
            foreach ($property in $Element.EnumerateObject()) {
                if (-not $seen.Add($property.Name)) { throw 'Duplicate JSON key' }
                Unique $property.Value
            }
        } elseif ($Element.ValueKind -eq [Text.Json.JsonValueKind]::Array) {
            foreach ($element in $Element.EnumerateArray()) { Unique $element }
        }
    }
    try { Unique $doc.RootElement } finally { $doc.Dispose() }
    return $raw | ConvertFrom-Json
}
$gitExe = (Get-Command git.exe -CommandType Application -ErrorAction Stop | Select-Object -First 1).Source
$pwshExe = (Get-Command pwsh.exe -CommandType Application -ErrorAction Stop | Select-Object -First 1).Source
$childGit = @{
    GIT_CONFIG_NOSYSTEM='1'; GIT_CONFIG_GLOBAL='NUL'; GIT_TERMINAL_PROMPT='0';
    GIT_NO_REPLACE_OBJECTS='1'; GIT_NO_LAZY_FETCH='1'; GIT_ATTR_NOSYSTEM='1';
    GIT_TEMPLATE_DIR=$template; GIT_CONFIG_COUNT='4';
    GIT_CONFIG_KEY_0=('url.' + ([Uri]$mirror).AbsoluteUri + '.insteadOf'); GIT_CONFIG_VALUE_0=$origin;
    GIT_CONFIG_KEY_1='protocol.allow'; GIT_CONFIG_VALUE_1='never';
    GIT_CONFIG_KEY_2='protocol.file.allow'; GIT_CONFIG_VALUE_2='always';
    GIT_CONFIG_KEY_3='core.hooksPath'; GIT_CONFIG_VALUE_3=$hooks
}
$gitReceipts = [Collections.Generic.List[object]]::new()
function Git([string]$GitRoot, [string[]]$Arguments) {
    $info = [Diagnostics.ProcessStartInfo]::new($gitExe)
    $info.WorkingDirectory = $project; $info.UseShellExecute = $false
    $info.CreateNoWindow = $true; $info.RedirectStandardOutput = $true; $info.RedirectStandardError = $true
    $info.ArgumentList.Add('--no-optional-locks'); $info.ArgumentList.Add('-C'); $info.ArgumentList.Add((Safe $GitRoot))
    foreach ($argument in $Arguments) { $info.ArgumentList.Add($argument) }
    foreach ($key in $childGit.Keys) { $info.Environment[$key] = $childGit[$key] }
    $clock = [Diagnostics.Stopwatch]::StartNew(); $at = [DateTime]::UtcNow.ToString('o')
    $process = [Diagnostics.Process]::Start($info)
    $outTask = $process.StandardOutput.ReadToEndAsync(); $errTask = $process.StandardError.ReadToEndAsync()
    $process.WaitForExit(); $output = $outTask.GetAwaiter().GetResult(); $errorText = $errTask.GetAwaiter().GetResult()
    $code = $process.ExitCode; $process.Dispose(); $clock.Stop()
    $gitReceipts.Add([ordered]@{argv=@('-C',$GitRoot)+$Arguments;startUtc=$at;durationMs=$clock.ElapsedMilliseconds;exitCode=$code;stdout=$output;stderr=$errorText})
    if ($code -ne 0) { throw "Git failed: $($Arguments[0])" }
    return $output.TrimEnd("`r", "`n")
}
function ManifestSet {
    return @(Get-ChildItem -LiteralPath $workspace -Directory -Force | ForEach-Object {
        $file = Join-Path $_.FullName '.cvf\manifest.json'
        if (Test-Path -LiteralPath $file -PathType Leaf) { Safe $file }
    } | Sort-Object)
}
function CheckPreimages {
    Pinned $preimagesPath $ExpectedPreimagesSha256
    if ($receipt.workspace -ine $workspace -or $receipt.core -ine $core -or $receipt.expectedCoreFrom -cne $source -or
        $receipt.expectedCoreTo -cne $target -or @($receipt.entries).Count -ne 38 -or @($receipt.siblingManifests).Count -ne 15) { throw 'Preimage identity/count mismatch' }
    $preimageRoot = Safe $receipt.preimageRoot
    if (-not $preimageRoot.StartsWith($backupRoot + '\', [StringComparison]::OrdinalIgnoreCase)) { throw 'Preimages outside backup root' }
    if (@($receipt.entries | Where-Object state -eq 'PRESENT').Count -ne 35 -or @($receipt.entries | Where-Object state -eq 'ABSENT').Count -ne 3) { throw 'Preimage state counts mismatch' }
    $seen = [Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
    foreach ($entry in $receipt.entries) {
        $path = Safe $entry.path
        if (-not $seen.Add($path) -or $entry.kind -notin @('shared','project')) { throw 'Duplicate/invalid preimage target' }
        if ($entry.kind -eq 'project' -and -not $path.StartsWith($project + '\',[StringComparison]::OrdinalIgnoreCase)) { throw 'Project preimage outside project' }
        if ($entry.state -eq 'ABSENT') {
            if (Test-Path -LiteralPath $path) { throw "Expected absent: $path" }
        } elseif ($entry.state -eq 'PRESENT') {
            if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { throw "Expected original file: $path" }
            Pinned $path $entry.sha256
            $copy = Safe $entry.backup
            if (-not $copy.StartsWith($preimageRoot + '\',[StringComparison]::OrdinalIgnoreCase)) { throw 'Backup path outside preimage root' }
            Pinned $copy $entry.sha256
            if ((Get-Acl -LiteralPath $path).Sddl -cne $entry.sddl) { throw "Original ACL differs: $path" }
        } else { throw 'Unknown preimage state' }
    }
    $expectedManifests = @($receipt.siblingManifests.path | Sort-Object)
    if ((ManifestSet) -join "`n" -cne ($expectedManifests -join "`n")) { throw 'Sibling manifest inventory differs' }
    foreach ($entry in $receipt.siblingManifests) { Pinned $entry.path $entry.sha256 }
    $active = Join-Path $workspace 'CVF_RULE_PACKS\ACTIVE_RULE_PACK.json'
    if (@($receipt.entries | Where-Object path -IEQ $active).Count -ne 1 -or (Json $active).activeProfile -cne 'operator-local') { throw 'Operator-local profile required' }
    if ((Git $core @('rev-parse','HEAD')) -cne $source -or (Git $core @('rev-parse','origin/main')) -cne $target -or
        (Git $core @('config','--local','--get','remote.origin.url')) -cne $origin -or (Git $core @('status','--porcelain')) -cne '') { throw 'Source Core identity/cleanliness mismatch' }
}
function NoSpecialObjects([string]$Root, [bool]$Bare) {
    $gitDir = if ($Bare) { $Root } else { Join-Path $Root '.git' }
    foreach ($rel in @('shallow','objects\info\alternates','info\grafts')) {
        if (Test-Path -LiteralPath (Safe (Join-Path $gitDir $rel))) { throw "Unsupported object mechanism: $rel" }
    }
    if ((Git $Root @('rev-parse','--is-shallow-repository')) -cne 'false' -or
        (Git $Root @('for-each-ref','--format=%(refname)','refs/replace')) -cne '') { throw 'Shallow/replacement objects forbidden' }
}
function CollisionGuard {
    $at = [DateTime]::Now
    $validatedBackupRoot = Safe $backupRoot
    foreach ($second in 0..610) {
        foreach ($prefix in @('.Controlled-Vibe-Framework-CVF-','.Controlled-Vibe-Framework-CVF-failed-')) {
            $candidate = [IO.Path]::GetFullPath((Join-Path $validatedBackupRoot ($prefix + $at.AddSeconds($second).ToString('yyyyMMdd-HHmmss'))))
            if (-not $candidate.StartsWith($validatedBackupRoot.TrimEnd('\') + '\',[StringComparison]::OrdinalIgnoreCase)) { throw 'Backup candidate outside validated root' }
            if (Test-Path -LiteralPath $candidate) { throw "Backup collision: $candidate" }
        }
    }
    if (([DateTime]::Now - $at).TotalSeconds -gt 10) { throw 'Collision scan stale' }
}
function MirrorCheck {
    NoSpecialObjects $mirror $true
    if ((Git $mirror @('rev-parse','HEAD')) -cne $target -or (Git $mirror @('symbolic-ref','HEAD')) -cne 'refs/heads/main' -or
        (Git $mirror @('for-each-ref','--format=%(refname)')) -cne 'refs/heads/main' -or
        (Git $mirror @('rev-parse',($target+'^{tree}'))) -cne $sourceTree -or
        (Git $mirror @('ls-tree','-r','--full-tree',$target)) -cne $sourceListing) { throw 'Mirror target/refs/listing mismatch' }
    $null = Git $mirror @('fsck','--full','--strict')
}
function Descendants([int]$RootPid) {
    $queue = [Collections.Generic.Queue[int]]::new(); $queue.Enqueue($RootPid)
    $seen = @{}; $found = @()
    while ($queue.Count) {
        $parent = $queue.Dequeue()
        foreach ($child in @(Get-CimInstance Win32_Process -Filter "ParentProcessId=$parent" -ErrorAction Stop)) {
            $identity = "$($child.ProcessId)|$($child.CreationDate)"
            if (-not $seen.ContainsKey($identity)) {
                $seen[$identity] = $true
                $found += [pscustomobject]@{pid=[int]$child.ProcessId;parent=[int]$child.ParentProcessId;created=[string]$child.CreationDate;executable=[string]$child.ExecutablePath}
                if ($found.Count -gt 64) { throw 'Scoped descendant cap exceeded' }
                $queue.Enqueue([int]$child.ProcessId)
            }
        }
    }
    return $found
}
function ObserveChildren {
    foreach ($child in @(Descendants $process.Id)) {
        $known["$($child.pid)|$($child.created)"] = $child
        if ($known.Count -gt 64) { throw 'Cumulative scoped descendant cap exceeded' }
    }
}
function CustodyQuiescent {
    if ($null -eq $process) { return $false }
    $process.Refresh()
    if (-not $process.HasExited) { ObserveChildren; return $false }
    $process.WaitForExit()
    ObserveChildren
    foreach ($child in $known.Values) {
        $live = @(Get-CimInstance Win32_Process -Filter "ProcessId=$($child.pid)" -ErrorAction Stop)
        if ($live.Count -eq 0) { continue }
        if ($live.Count -ne 1) { return $false }
        $observedCreation = [string]$live[0].CreationDate
        if ([string]::IsNullOrWhiteSpace($child.created) -or [string]::IsNullOrWhiteSpace($observedCreation)) { return $false }
        if ($observedCreation -ceq $child.created) { return $false }
        # Only a different, known nonempty creation identity establishes reuse.
        # ExecutablePath is diagnostic: null/change must never imply absence.
    }
    return $true
}
function CheckMirrorInventory {
    foreach ($path in $inventory.Keys) { Pinned $path $inventory[$path] }
    $files = @(Get-ChildItem -LiteralPath $mirror -File -Recurse -Force | ForEach-Object { Safe $_.FullName })
    if ($files.Count -ne $inventory.Count -or @($files | Where-Object { -not $inventory.ContainsKey($_) }).Count) { throw 'Mirror inventory changed' }
}
function WriteResult {
    $result.observedChildren = @($known.Values)
    $result.gitReceipts = @($gitReceipts.ToArray())
    $result | ConvertTo-Json -Depth 12 | Set-Content -LiteralPath $resultPath -Encoding utf8
}

if ((Get-Location).ProviderPath -ine $project) { throw 'Run from project root' }
if (@(Get-ChildItem Env: | Where-Object Name -Like 'GIT_*').Count) { throw 'Inherited GIT_* forbidden' }
foreach ($path in @($workspace,$project,$core,$backupRoot,$probe,$mirror,$support,$updater,$preimagesPath,$authorityPath,$claim,$stdout,$stderr,$resultPath)) { $null = Safe $path }
Pinned $PSCommandPath $ExpectedSourceSha256
Pinned $preimagesPath $ExpectedPreimagesSha256
Pinned $authorityPath $authoritySha256
$committedAuthority = Git $project @('show',($preparation + ':CVF_SESSION/authority/CCMAI-RUNTIME-139-CORE-FRESHNESS-20261104-ADMISSION.json'))
$committedBytes = [Text.Encoding]::UTF8.GetBytes($committedAuthority.Replace("`r`n","`n") + "`n")
$committedHash = [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData($committedBytes)).ToLowerInvariant()
if ($committedHash -cne $authorityGitLfSha256) { throw 'Committed preparation authority mismatch' }
$receipt = Json $preimagesPath
CheckPreimages
NoSpecialObjects $core $false
$sourceTree = Git $core @('rev-parse',($target+'^{tree}'))
$sourceListing = Git $core @('ls-tree','-r','--full-tree',$target)
$null = Git $core @('fsck','--full','--strict')
foreach ($path in @($mirror,$support,$claim,$stdout,$stderr,$resultPath)) {
    if (Test-Path -LiteralPath $path) { throw "Permanent collision/attempt: $path" }
}
CollisionGuard
if ($PreflightOnly) { Write-Output 'PREFLIGHT_ONLY: read checks passed; no latch/mirror/updater'; return }

# Root's reviewed exact -Execute -ExpectedSourceSha256 command is the explicit dispatch.
Git $project @('merge-base','--is-ancestor',$preparation,'HEAD') | Out-Null
$existing = @(Get-ChildItem -LiteralPath $backupRoot -Directory -Force | ForEach-Object FullName)
$locks = [Collections.Generic.List[IO.FileStream]]::new()
$result = [ordered]@{status='STARTING';startUtc=[DateTime]::UtcNow.ToString('o');sourceSha256=$ExpectedSourceSha256;preimagesSha256=$ExpectedPreimagesSha256;authoritySha256=$authoritySha256;updaterPid=$null;exitCode=$null;originalCoreBackup=$null;failedCloneBackup=$null;childObservation='NOT_OBSERVED_UNKNOWN';note='Mirror file handles limit writes; new-file/same-user races remain possible. No global cleanup or rollback claim.'}
$launched = $false; $exited = $false; $custodySettled = $false
$process = $null; $known = @{}
# Permanent CreateNew attempt latch precedes every first filesystem effect.
$latch = [IO.File]::Open($claim,[IO.FileMode]::CreateNew,[IO.FileAccess]::Write,[IO.FileShare]::None)
try { $bytes=[Text.Encoding]::UTF8.GetBytes("one fixed c4 updater attempt`n"); $latch.Write($bytes,0,$bytes.Length) } finally { $latch.Dispose() }
try {
    $null = New-Item -ItemType Directory -Path $support -ErrorAction Stop
    $null = New-Item -ItemType Directory -Path $template,$hooks -ErrorAction Stop
    $null = Git $project @('clone','--mirror','--no-local',$core,$mirror)
    foreach ($ref in @((Git $mirror @('for-each-ref','--format=%(refname)')) -split "`r?`n" | Where-Object { $_ })) {
        if ($ref -cne 'refs/heads/main') { $null = Git $mirror @('update-ref','-d',$ref) }
    }
    $null = Git $mirror @('update-ref','refs/heads/main',$target)
    $null = Git $mirror @('symbolic-ref','HEAD','refs/heads/main')
    MirrorCheck
    $inventory = @{}
    foreach ($file in @(Get-ChildItem -LiteralPath $mirror -File -Recurse -Force)) {
        $path = Safe $file.FullName
        $handle = [IO.File]::Open($path,[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::Read)
        $locks.Add($handle); $inventory[$path] = Hash $path
    }
    if ($inventory.Count -eq 0) { throw 'Empty mirror inventory' }
    $result.mirrorInventory = $inventory
    MirrorCheck
    CheckMirrorInventory
    CheckPreimages
    Pinned $authorityPath $authoritySha256
    Pinned $PSCommandPath $ExpectedSourceSha256
    CollisionGuard
    $arguments = @('-NoProfile','-ExecutionPolicy','Bypass','-File',('"'+$updater+'"'),'-WorkspaceRoot',('"'+$workspace+'"'))
    $launched = $true
    $process = Start-Process -FilePath $pwshExe -ArgumentList $arguments -Environment $childGit -WorkingDirectory $project -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr -PassThru
    $null = $process.Handle
    $result.updaterPid = $process.Id
    $result.argv = @($pwshExe)+$arguments
    $deadline = [DateTime]::UtcNow.AddSeconds(600)
    while (-not $process.HasExited -and [DateTime]::UtcNow -lt $deadline) {
        ObserveChildren
        Start-Sleep -Milliseconds 500
        $process.Refresh()
    }
    if (-not $process.HasExited) { throw 'Acceptance-window timeout: custody held; no kill/retry/restore' }
    $process.WaitForExit(); $result.exitCode = $process.ExitCode; $exited = $true
    if (-not (CustodyQuiescent)) { throw 'Known descendant still active: custody held' }
    $custodySettled = $true
    $result.childObservation = 'KNOWN_IDENTITIES_ABSENT_SAMPLED_NOT_GLOBAL'
    $result.endUtc = [DateTime]::UtcNow.ToString('o')
    CheckMirrorInventory
    MirrorCheck
    $new = @(Get-ChildItem -LiteralPath $backupRoot -Directory -Force | Where-Object { $_.FullName -notin $existing })
    $old = @($new | Where-Object Name -Match '^\.Controlled-Vibe-Framework-CVF-\d{8}-\d{6}$')
    $failed = @($new | Where-Object Name -Match '^\.Controlled-Vibe-Framework-CVF-failed-\d{8}-\d{6}$')
    $result.originalCoreBackup = @($old | ForEach-Object { Safe $_.FullName })
    $result.failedCloneBackup = @($failed | ForEach-Object { Safe $_.FullName })
    if ($result.exitCode -ne 0) { throw 'Official updater failed; built-in recovery only; preserve all paths for review' }
    if ($old.Count -ne 1 -or $failed.Count -ne 0) { throw 'Original backup identity ambiguous' }
    if ((Git $old[0].FullName @('rev-parse','HEAD')) -cne $source -or (Git $old[0].FullName @('status','--porcelain')) -cne '') { throw 'Original backup differs' }
    if ((Git $core @('rev-parse','HEAD')) -cne $target -or (Git $core @('rev-parse','origin/main')) -cne $target -or
        (Git $core @('config','--local','--get','remote.origin.url')) -cne $origin -or (Git $core @('status','--porcelain')) -cne '' -or
        (Git $core @('ls-tree','-r','--full-tree','HEAD')) -cne $sourceListing) { throw 'Postclone exact public tree/identity mismatch' }
    NoSpecialObjects $core $false
    foreach ($entry in $receipt.siblingManifests) { Pinned $entry.path $entry.sha256 }
    if (((ManifestSet) -join "`n") -cne (@($receipt.siblingManifests.path | Sort-Object) -join "`n")) { throw 'Sibling set changed' }
    $active = Join-Path $workspace 'CVF_RULE_PACKS\ACTIVE_RULE_PACK.json'
    $activeEntry = @($receipt.entries | Where-Object path -IEQ $active)
    Pinned $active $activeEntry[0].sha256
    if ((Json $active).activeProfile -cne 'operator-local') { throw 'Profile changed' }
    foreach ($entry in @($receipt.entries | Where-Object kind -EQ 'shared')) {
        if ($entry.state -eq 'ABSENT') { if (Test-Path -LiteralPath (Safe $entry.path)) { throw 'Unexpected shared output' } }
        elseif (-not (Test-Path -LiteralPath (Safe $entry.path) -PathType Leaf)) { throw 'Shared output missing' }
    }
    $result.status = 'UPDATER_RETURNED_EXACT_TREE_POSTCHECKS_PASS_REVIEW_REQUIRED'
} catch {
    $result.status = if ($launched -and -not $custodySettled) { 'UNKNOWN_WRITER_CUSTODY_HELD_NO_ADOPTION' } else { 'HOLD_REVIEW_REQUIRED' }
    if ($launched -and $null -eq $process) { $result.launchObservation = 'START_PROCESS_ASSIGNMENT_AMBIGUOUS_UNKNOWN_CUSTODY_HELD' }
    $result.error = $_.Exception.Message
} finally {
    if ($launched -and -not $custodySettled) {
        # The 600s limit ends acceptance, never the input-lock custody obligation.
        # Keep THIS process and its original handles alive; no new guardian/kill.
        $result.status = 'UNKNOWN_WRITER_CUSTODY_HELD_NO_ADOPTION'
        try { WriteResult } catch { Write-Warning 'Pending receipt write failed; input custody is still retained' }
        while (-not $custodySettled) {
            try { $custodySettled = CustodyQuiescent }
            catch { $result.custodyObservationError = $_.Exception.Message }
            if (-not $custodySettled) { Start-Sleep -Seconds 1 }
        }
        $result.childObservation = 'LATER_KNOWN_IDENTITIES_ABSENT_SAMPLED_ACCEPTANCE_REMAINS_UNKNOWN'
        $result.exitCode = $process.ExitCode
        $result.custodyReleasedUtc = [DateTime]::UtcNow.ToString('o')
        # Never promote a late timeout/observation recovery to acceptance PASS.
        $result.status = 'UNKNOWN_ACCEPTANCE_CUSTODY_LATER_QUIESCENT_REVIEW_REQUIRED'
    }
    if ($launched) {
        $new = @(Get-ChildItem -LiteralPath $backupRoot -Directory -Force | Where-Object { $_.FullName -notin $existing })
        $result.originalCoreBackup = @($new | Where-Object Name -Match '^\.Controlled-Vibe-Framework-CVF-\d{8}-\d{6}$' | ForEach-Object { Safe $_.FullName })
        $result.failedCloneBackup = @($new | Where-Object Name -Match '^\.Controlled-Vibe-Framework-CVF-failed-\d{8}-\d{6}$' | ForEach-Object { Safe $_.FullName })
    }
    foreach ($handle in $locks) { $handle.Dispose() }
    WriteResult
}
if ($result.status -cne 'UPDATER_RETURNED_EXACT_TREE_POSTCHECKS_PASS_REVIEW_REQUIRED') { exit 1 }
