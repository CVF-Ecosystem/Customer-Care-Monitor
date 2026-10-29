"""Windows PowerShell 5 and pwsh must render both catalog views identically."""
import json
import os
import shutil
import subprocess
import unittest


ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
LIB = os.path.join(ROOT, "scripts", "lib", "downstream_catalog", "CvfDownstreamCatalogLib.ps1")


@unittest.skipUnless(os.name == "nt", "Windows PowerShell 5 parity is Windows-only")
class CatalogParity(unittest.TestCase):
    def test_real_registry_views_match_in_both_shells(self):
        self.assertTrue(shutil.which("powershell"), "Windows PowerShell 5 is required")
        self.assertTrue(shutil.which("pwsh"), "PowerShell 7 is required")
        command = (
            "$ErrorActionPreference = 'Stop'; . '" + LIB.replace("'", "''") + "'; "
            "$a = Get-Content 'docs/catalog/ARTIFACT_REGISTRY.json' -Raw -Encoding utf8 | ConvertFrom-Json; "
            "$m = Get-Content 'docs/catalog/MODULE_REGISTRY.json' -Raw -Encoding utf8 | ConvertFrom-Json; "
            "$i = ConvertTo-CvfIndexMarkdown -Registry $a; "
            "$c = ConvertTo-CvfModuleCatalogMarkdown -Registry $m; "
            "$h = [System.Security.Cryptography.SHA256]::Create(); "
            "$u = [System.Text.Encoding]::UTF8; "
            "$ih = [BitConverter]::ToString($h.ComputeHash($u.GetBytes($i))).Replace('-', '').ToLowerInvariant(); "
            "$ch = [BitConverter]::ToString($h.ComputeHash($u.GetBytes($c))).Replace('-', '').ToLowerInvariant(); "
            "Write-Output ($ih + ' ' + $ch)")
        outputs = []
        for shell in ("powershell", "pwsh"):
            proc = subprocess.run([shell, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", command],
                                  cwd=ROOT, text=True, capture_output=True, timeout=90)
            self.assertEqual(proc.returncode, 0, proc.stderr)
            hashes = proc.stdout.strip().split()
            self.assertEqual(len(hashes), 2, proc.stdout)
            outputs.append(hashes)
        self.assertEqual(outputs[0], outputs[1], "index and module views must match across shells")

    def test_ordinal_id_collision_order(self):
        command = (
            "$ErrorActionPreference = 'Stop'; . '" + LIB.replace("'", "''") + "'; "
            "$r = @([pscustomobject]@{id='r139-pin'}, [pscustomobject]@{id='r139-pinned'}, "
            "[pscustomobject]@{id='ccma-product'}, [pscustomobject]@{id='ccmai-milestone'}); "
            "(Sort-CvfRecordsByOrdinalId -Records $r | ForEach-Object { $_.id }) -join ','")
        for shell in ("powershell", "pwsh"):
            proc = subprocess.run([shell, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", command],
                                  cwd=ROOT, text=True, capture_output=True, timeout=30)
            self.assertEqual(proc.returncode, 0, proc.stderr)
            self.assertEqual(proc.stdout.strip(), "ccma-product,ccmai-milestone,r139-pin,r139-pinned")


if __name__ == "__main__":
    unittest.main()
