#
# install-service-windows.ps1 — Install bored as a Windows scheduled task.
#
# Usage:
#   powershell -ExecutionPolicy Bypass -File scripts\install-service-windows.ps1 [path-to-bored.exe]
#
# The daemon auto-starts on login via HKCU\...\Run registry key.
# To remove: Remove-ItemProperty -Path "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run" -Name "Bore Daemon"
#

param(
    [string]$BinaryPath = ""
)

$ErrorActionPreference = "Stop"

$InstallDir = "$env:LOCALAPPDATA\Bore"
$TaskName = "Bore Daemon"

# --- Resolve binary ---

if (-not $BinaryPath) {
    $candidates = @(
        ".\bin\bored.exe",
        "$InstallDir\bored.exe",
        (Get-Command bored -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source)
    )
    foreach ($c in $candidates) {
        if ($c -and (Test-Path $c)) {
            $BinaryPath = $c
            break
        }
    }
}

if (-not $BinaryPath -or -not (Test-Path $BinaryPath)) {
    Write-Error "bored.exe not found. Build it first with: make build-daemon"
    exit 1
}

$BinaryPath = (Resolve-Path $BinaryPath).Path
Write-Host "Using binary: $BinaryPath"

# --- Install binary ---

if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$DestPath = "$InstallDir\bored.exe"
if ($BinaryPath -ne $DestPath) {
    Copy-Item $BinaryPath $DestPath -Force
    Write-Host "Installed binary to $DestPath"
}

# --- Ensure data directories ---

$AppData = "$env:APPDATA\Bore"
if (-not (Test-Path $AppData)) {
    New-Item -ItemType Directory -Path $AppData -Force | Out-Null
}

# --- Auto-start daemon on login via registry (no elevation required) ---

$RunKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run"
Set-ItemProperty -Path $RunKey -Name $TaskName -Value "`"$DestPath`""
Write-Host "Registered daemon auto-start in registry"

# Remove legacy scheduled task if present.
schtasks /query /tn $TaskName 2>$null
if ($LASTEXITCODE -eq 0) {
    Write-Host "Removing legacy scheduled task..."
    schtasks /delete /tn $TaskName /f | Out-Null
}

# --- Install desktop app + Start Menu shortcut ---

$DesktopBin = ".\bin\bore-desktop.exe"
if (Test-Path $DesktopBin) {
    $DesktopDest = "$InstallDir\bore-desktop.exe"
    Copy-Item $DesktopBin $DesktopDest -Force
    Write-Host "Installed desktop app to $DesktopDest"

    # Create Start Menu shortcut
    $StartMenuDir = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs"
    $ShortcutPath = "$StartMenuDir\Bore.lnk"
    $WshShell = New-Object -ComObject WScript.Shell
    $Shortcut = $WshShell.CreateShortcut($ShortcutPath)
    $Shortcut.TargetPath = $DesktopDest
    $Shortcut.WorkingDirectory = $InstallDir
    $Shortcut.Description = "Bore SSH Tunnel Manager"
    $Shortcut.Save()
    Write-Host "Created Start Menu shortcut — search 'Bore' in the Start Menu."
} else {
    Write-Host "Desktop binary not found at $DesktopBin — skipping desktop app install."
    Write-Host "Build it with: make build-desktop"
}

# --- Install CLI tools ---

$CliBin = ".\bin\bore.exe"
$TuiBin = ".\bin\bore-tui.exe"
if (Test-Path $CliBin) {
    Copy-Item $CliBin "$InstallDir\bore.exe" -Force
    Write-Host "Installed bore CLI to $InstallDir\bore.exe"
}
if (Test-Path $TuiBin) {
    Copy-Item $TuiBin "$InstallDir\bore-tui.exe" -Force
    Write-Host "Installed bore-tui to $InstallDir\bore-tui.exe"
}

Write-Host ""
Write-Host "============================================"
Write-Host "  bore installed!"
Write-Host "============================================"
Write-Host ""
Write-Host "  Start daemon:  & `"$DestPath`""
Write-Host "  Stop daemon:   taskkill /im bored.exe"
Write-Host "  Auto-start:    registered in HKCU\...\Run (starts on login)"
Write-Host ""
Write-Host "  Config:   $AppData\tunnels.yaml"
Write-Host "  Data:     $InstallDir\"
Write-Host ""
Write-Host "  The daemon will auto-start on login."

# Check if install dir is in PATH
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    Write-Host ""
    Write-Host "  To use bore from any terminal, add to your PATH:"
    Write-Host "    [Environment]::SetEnvironmentVariable('Path', `"$InstallDir;`$env:Path`", 'User')"
}

Write-Host ""
