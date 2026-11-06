param([Parameter(Mandatory=$true)][string]$OwnedProfilesJson, [string]$FixtureRowsJson)
$ErrorActionPreference = 'Stop'
if (-not $OwnedProfilesJson.TrimStart().StartsWith('[')) { throw 'profiles must be an array' }
$profiles = ConvertFrom-Json -InputObject $OwnedProfilesJson
if ($null -eq $profiles) { $profiles = @() }
if ($profiles -isnot [array]) { throw 'profiles must be an array' }
$seen = New-Object 'System.Collections.Generic.HashSet[string]' ([StringComparer]::OrdinalIgnoreCase)
foreach ($p in $profiles) {
  if ($p -isnot [string] -or [string]::IsNullOrWhiteSpace($p)) { throw 'invalid profile' }
  $normalized = [IO.Path]::GetFullPath($p).TrimEnd('\')
  if (-not $seen.Add($normalized)) { throw 'duplicate profile' }
}
Add-Type @'
using System;
using System.Runtime.InteropServices;
public static class R151Argv {
 [DllImport("shell32.dll", SetLastError=true, CharSet=CharSet.Unicode)] public static extern IntPtr CommandLineToArgvW(string line, out int argc);
 [DllImport("kernel32.dll")] public static extern IntPtr LocalFree(IntPtr p);
 public static string[] Parse(string line) {
  int count; IntPtr p=CommandLineToArgvW(line, out count); if (p==IntPtr.Zero) throw new System.ComponentModel.Win32Exception();
  try { var a=new string[count]; for(int i=0;i<count;i++) a[i]=Marshal.PtrToStringUni(Marshal.ReadIntPtr(p,i*IntPtr.Size)); return a; }
  finally { LocalFree(p); }
 }
}
'@
function Get-ProfileMatch([string]$line) {
  if ([string]::IsNullOrWhiteSpace($line)) { return $false }
  $argv=[R151Argv]::Parse($line)
  $values=@()
  for ($i=1; $i -lt $argv.Length; $i++) {
    if ($argv[$i] -match '^--user-data-dir=(.*)$') { $values += $Matches[1] }
    elseif ($argv[$i] -ieq '--user-data-dir') { if (++$i -ge $argv.Length) { throw 'missing profile argument' }; $values += $argv[$i] }
  }
  if ($values.Count -gt 1) { throw 'ambiguous profile arguments' }
  if ($values.Count -eq 0 -or [string]::IsNullOrWhiteSpace($values[0])) { return $false }
  return $seen.Contains([IO.Path]::GetFullPath($values[0]).TrimEnd('\'))
}
$rows = if ($FixtureRowsJson) { ConvertFrom-Json -InputObject $FixtureRowsJson } else { @(Get-CimInstance Win32_Process) }
$result = @($rows | ForEach-Object {
  $line=[string]$_.CommandLine
  [pscustomobject]@{ pid=[int]$_.ProcessId; parentPid=[int]$_.ParentProcessId; name=[string]$_.Name;
    createdAt=$(if ($_.CreationDate) { ([datetime]$_.CreationDate).ToUniversalTime().ToString('o') } else { $null });
    executablePath=[string]$_.ExecutablePath; ownedProfileArg=$(if ($_.Name -ieq 'chrome.exe') { Get-ProfileMatch $line } else { $false }); commandLineKnown=(-not [string]::IsNullOrWhiteSpace($line)) }
})
ConvertTo-Json -InputObject $result -Depth 4 -Compress
