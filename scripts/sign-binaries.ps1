$signtool = "C:\Program Files (x86)\Windows Kits\10\bin\10.0.22621.0\x64\signtool.exe"
$thumbprint = "8912A5582370B2BFE414223B2121C1C522143EA9"

$binaries = @(
    "bin\bore-desktop.exe",
    "bin\bored.exe",
    "bin\bore.exe",
    "bin\bore-tui.exe"
)

foreach ($bin in $binaries) {
    if (Test-Path $bin) {
        Write-Host "Signing $bin..."
        & $signtool sign /sha1 $thumbprint /fd SHA256 /t http://timestamp.digicert.com $bin
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to sign $bin"
            exit 1
        }
    } else {
        Write-Host "Skipping $bin (not found)"
    }
}

Write-Host ""
Write-Host "All binaries signed."
