# Export the code signing certificate as base64-encoded PFX for GitHub secrets.
# Usage: powershell -File scripts\export-cert.ps1

$cert = Get-ChildItem Cert:\CurrentUser\My -CodeSigningCert |
    Where-Object { $_.Subject -eq 'CN=Hyperplex, O=Hyperplex' }

if (-not $cert) {
    Write-Error "Code signing certificate not found"
    exit 1
}

$password = Read-Host -Prompt "Enter a password for the PFX file" -AsSecureString
$pfxPath = Join-Path $env:TEMP "bore-codesign.pfx"

Export-PfxCertificate -Cert $cert -FilePath $pfxPath -Password $password | Out-Null
$base64 = [Convert]::ToBase64String([IO.File]::ReadAllBytes($pfxPath))
Remove-Item $pfxPath

Write-Host ""
Write-Host "Thumbprint: $($cert.Thumbprint)"
Write-Host ""
Write-Host "Set these GitHub secrets (Settings > Secrets and variables > Actions):"
Write-Host ""
Write-Host "  SIGN_CERT_THUMBPRINT = $($cert.Thumbprint)"
Write-Host "  SIGN_CERT_PASSWORD   = <the password you just entered>"
Write-Host "  SIGN_CERT_PFX        = <base64 below>"
Write-Host ""
Write-Host "--- Base64 PFX (copy everything below this line) ---"
Write-Host $base64
