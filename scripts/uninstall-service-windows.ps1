#
# uninstall-service-windows.ps1 — Remove bore (daemon task, binaries, Start Menu shortcut).
#

$ErrorActionPreference = "Stop"
$TaskName = "Bore Daemon"
$InstallDir = "$env:LOCALAPPDATA\Bore"

Write-Host "Stopping and removing bored task..."
schtasks /end /tn $TaskName 2>$null
schtasks /delete /tn $TaskName /f 2>$null

# Remove Start Menu shortcut
$ShortcutPath = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Bore.lnk"
if (Test-Path $ShortcutPath) {
    Remove-Item $ShortcutPath -Force
    Write-Host "Removed Start Menu shortcut."
}

# Remove installed binaries
foreach ($bin in @("bored.exe", "bore.exe", "bore-tui.exe", "bore-desktop.exe")) {
    $p = "$InstallDir\$bin"
    if (Test-Path $p) {
        Remove-Item $p -Force
    }
}
Write-Host "Removed installed binaries from $InstallDir."

Write-Host ""
Write-Host "Uninstalled bore. Config and data were NOT removed."
Write-Host "  Config: $env:APPDATA\Bore\"
Write-Host "  Data:   $InstallDir\"
Write-Host ""
Write-Host "To fully remove, delete those directories manually."
