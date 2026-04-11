$signtool = "C:\Program Files (x86)\Windows Kits\10\bin\10.0.22621.0\x64\signtool.exe"
$thumbprint = "8912A5582370B2BFE414223B2121C1C522143EA9"
$installer = "bin\Bore-Setup-v0.1.14.exe"

Write-Host "Signing $installer..."
& $signtool sign /sha1 $thumbprint /fd SHA256 /t http://timestamp.digicert.com $installer
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to sign installer"
    exit 1
}
Write-Host "Installer signed."
