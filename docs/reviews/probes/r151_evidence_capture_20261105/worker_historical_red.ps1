$profiles = @(ConvertFrom-Json -InputObject '["C:\\Temp\\profile one","C:\\Temp\\profile two"]')
$line = 'chrome.exe --user-data-dir="C:\Temp\profile one"'
$matches = @($profiles | Where-Object { $line.IndexOf($_, [StringComparison]::OrdinalIgnoreCase) -ge 0 })
Write-Output "wrapped_count=$($profiles.Count) match_count=$($matches.Count) expected_match_count=1"
if ($matches.Count -ne 1) { exit 1 }
