; bore-installer.iss — Inno Setup script for Bore on Windows.
;
; Prerequisites:
;   - Inno Setup 6+ (https://jrsoftware.org/isinfo.php)
;   - Built binaries in bin\ (make build on Windows, or cross-compiled)
;
; Build the installer:
;   iscc scripts\bore-installer.iss
;
; Output:
;   bin\Bore-Setup-<version>.exe

#ifndef VERSION
  #define VERSION "dev"
#endif

[Setup]
AppId={{B0RE-SSH-TUNNEL-MANAGER}
AppName=Bore
AppVersion={#VERSION}
AppVerName=Bore
AppPublisher=Hyperplex
AppPublisherURL=https://github.com/hyperplex-tech/bore
DefaultDirName={localappdata}\Bore
DefaultGroupName=Bore
DisableProgramGroupPage=yes
OutputDir=..\bin
OutputBaseFilename=Bore-Setup-{#VERSION}
Compression=lzma2
SolidCompression=yes
PrivilegesRequired=lowest
UninstallDisplayIcon={app}\bore-desktop.exe
SetupIconFile=..\assets\icon.ico
WizardStyle=modern
ArchitecturesInstallIn64BitMode=x64compatible

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Additional shortcuts:"; Flags: checkedonce
Name: "addtopath"; Description: "Add bore CLI tools to PATH"; GroupDescription: "CLI tools:"; Flags: checkedonce

[Files]
; Desktop app
Source: "..\bin\bore-desktop.exe"; DestDir: "{app}"; Flags: ignoreversion
; Daemon
Source: "..\bin\bored.exe"; DestDir: "{app}"; Flags: ignoreversion
; CLI
Source: "..\bin\bore.exe"; DestDir: "{app}"; Flags: ignoreversion
; TUI
Source: "..\bin\bore-tui.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
; Start Menu
Name: "{group}\Bore"; Filename: "{app}\bore-desktop.exe"; Comment: "Bore SSH Tunnel Manager"
Name: "{group}\Uninstall Bore"; Filename: "{uninstallexe}"
; Desktop shortcut (optional)
Name: "{userdesktop}\Bore"; Filename: "{app}\bore-desktop.exe"; Tasks: desktopicon; Comment: "Bore SSH Tunnel Manager"

[Registry]
; Add to user PATH (optional)
Root: HKCU; Subkey: "Environment"; ValueType: expandsz; ValueName: "Path"; ValueData: "{olddata};{app}"; Tasks: addtopath; Check: NeedsAddPath(ExpandConstant('{app}'))
; Auto-start daemon on login (no elevation required, unlike schtasks)
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "Bore Daemon"; ValueData: """{app}\bored.exe"""; Flags: uninsdeletevalue

[Run]
; Launch the app (which will auto-start the daemon)
Filename: "{app}\bore-desktop.exe"; Flags: nowait postinstall skipifsilent; Description: "Launch Bore"

[UninstallRun]
; Kill the daemon process before removing files
Filename: "taskkill.exe"; Parameters: "/f /im bored.exe"; Flags: runhidden

[UninstallDelete]
; Clean up any runtime files in the install directory
Type: filesandordirs; Name: "{app}\bore.db"
Type: filesandordirs; Name: "{app}\bored.lock"

[Code]
// Check if the install directory is already in PATH to avoid duplicates.
function NeedsAddPath(Param: string): Boolean;
var
  OrigPath: string;
begin
  if not RegQueryStringValue(HKEY_CURRENT_USER,
    'Environment', 'Path', OrigPath) then
  begin
    Result := True;
    exit;
  end;
  Result := Pos(';' + Uppercase(Param) + ';',
    ';' + Uppercase(OrigPath) + ';') = 0;
end;
