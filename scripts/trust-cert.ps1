$cert = Get-ChildItem Cert:\CurrentUser\My -CodeSigningCert | Where-Object { $_.Subject -eq 'CN=Hyperplex, O=Hyperplex' }
if (-not $cert) { Write-Error "Certificate not found"; exit 1 }
$tempFile = Join-Path $env:TEMP 'bore-codesign.cer'
Export-Certificate -Cert $cert -FilePath $tempFile | Out-Null
Import-Certificate -FilePath $tempFile -CertStoreLocation Cert:\CurrentUser\Root | Out-Null
Remove-Item $tempFile
Write-Host "Certificate trusted locally (Thumbprint: $($cert.Thumbprint))"
