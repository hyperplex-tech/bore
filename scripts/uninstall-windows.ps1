#
# uninstall-windows.ps1 — Fully uninstall bore from Windows (installer or manual).
#
# Usage:
#   powershell -ExecutionPolicy Bypass -File scripts\uninstall-windows.ps1 [-Purge]
#
# Without -Purge: removes binaries, shortcuts, and daemon task.
# With    -Purge: also removes config, data, and logs.
#

param(
    [switch]$Purge
)

$ErrorActionPreference = "Stop"
$TaskName = "Bore Daemon"
$InstallDir = "$env:LOCALAPPDATA\Bore"
$ConfigDir = "$env:APPDATA\Bore"

Write-Host "=== Uninstalling bore from Windows ==="
Write-Host ""

# --- Stop and remove scheduled task ---

Write-Host "Stopping daemon..."
schtasks /end /tn $TaskName 2>$null
schtasks /delete /tn $TaskName /f 2>$null

# --- Remove Start Menu shortcut ---

$ShortcutPath = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Bore.lnk"
if (Test-Path $ShortcutPath) {
    Remove-Item $ShortcutPath -Force
    Write-Host "  Removed Start Menu shortcut."
}

# --- Remove desktop shortcut ---

$DesktopShortcut = "$env:USERPROFILE\Desktop\Bore.lnk"
if (Test-Path $DesktopShortcut) {
    Remove-Item $DesktopShortcut -Force
    Write-Host "  Removed Desktop shortcut."
}

# --- Remove installed binaries ---

foreach ($bin in @("bored.exe", "bore.exe", "bore-tui.exe", "bore-desktop.exe")) {
    $p = "$InstallDir\$bin"
    if (Test-Path $p) {
        Remove-Item $p -Force
        Write-Host "  Removed $p"
    }
}

# --- Remove from PATH (if added by installer) ---

$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -and $UserPath.Contains($InstallDir)) {
    $NewPath = ($UserPath.Split(';') | Where-Object { $_ -ne $InstallDir }) -join ';'
    [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
    Write-Host "  Removed $InstallDir from user PATH."
}

# --- Uninstall registry entry (Inno Setup) ---

$UninstallKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\Bore_is1"
if (Test-Path $UninstallKey) {
    Remove-Item $UninstallKey -Force
    Write-Host "  Removed uninstall registry entry."
}

Write-Host ""
Write-Host "Removed: binaries, shortcuts, and daemon task."

# --- Purge config and data ---

if ($Purge) {
    Write-Host ""
    if (Test-Path $ConfigDir) {
        Remove-Item $ConfigDir -Recurse -Force
        Write-Host "  Removed $ConfigDir"
    }
    if (Test-Path $InstallDir) {
        Remove-Item $InstallDir -Recurse -Force
        Write-Host "  Removed $InstallDir"
    }
    Write-Host ""
    Write-Host "bore is fully removed from this system."
} else {
    Write-Host ""
    Write-Host "Config and data were NOT removed:"
    Write-Host "  Config: $ConfigDir"
    Write-Host "  Data:   $InstallDir"
    Write-Host ""
    Write-Host "To also remove config and data, re-run with -Purge:"
    Write-Host "  powershell -ExecutionPolicy Bypass -File uninstall-windows.ps1 -Purge"
}
