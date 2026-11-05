param([string]$OwnedProfilesJson = '[]')
$ErrorActionPreference = 'Stop'
$OwnedProfiles = @(ConvertFrom-Json -InputObject $OwnedProfilesJson)
$rows = @(
  Get-CimInstance Win32_Process |
    Where-Object { $_.Name -in @('chrome.exe','node.exe') } |
    ForEach-Object {
      $line = [string]$_.CommandLine
      $matches = @($OwnedProfiles | Where-Object { $line.IndexOf($_, [System.StringComparison]::OrdinalIgnoreCase) -ge 0 })
      [pscustomobject]@{
        pid = [int]$_.ProcessId
        parentPid = [int]$_.ParentProcessId
        name = [string]$_.Name
        createdAt = $(if ($_.CreationDate) { ([datetime]$_.CreationDate).ToUniversalTime().ToString('o') } else { $null })
        executablePath = [string]$_.ExecutablePath
        ownedProfileArg = ($matches.Count -gt 0)
        commandLineKnown = -not [string]::IsNullOrEmpty($line)
      }
    }
)
ConvertTo-Json -InputObject $rows -Depth 3 -Compress
