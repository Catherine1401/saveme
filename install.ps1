# Tải gói saveme cho Windows đã kiểm SHA-256 rồi chạy bộ cài trong gói; cần PowerShell quyền Administrator.
param(
    [string]$User = '',
    [string]$Version = '',
    [switch]$Update,
    [switch]$Uninstall,
    [string]$Archive = '',
    [string]$Checksums = '',
    [switch]$Help,
    [switch]$Elevated
)
$ErrorActionPreference = 'Stop'
$Base = 'https://github.com/Catherine1401/saveme/releases'
$ScriptUrl = 'https://raw.githubusercontent.com/Catherine1401/saveme/main/install.ps1'
$TagPattern = '^v?[0-9]+\.[0-9]+\.[0-9]+$'
$ChecksumName = 'checksums.txt'
$Members = @('bin\saveme.exe', 'bin\saved.exe', 'deploy\windows\install.ps1', 'deploy\windows\update.ps1', 'deploy\windows\uninstall.ps1',
    'deploy\windows\common.ps1', 'deploy\windows\config.json', 'LICENSE', 'VERSION')
$Usage = @'
Usage: install.ps1 [-User ACCOUNT] [-Version vX.Y.Z] [-Update | -Uninstall]
       install.ps1 -Version vX.Y.Z -Archive FILE -Checksums FILE

Downloads a verified Windows bundle and runs its installer. Run from a normal
PowerShell; it asks for administrator rights through UAC. -User selects the
ordinary account that manages locks and defaults to the current account. -Update keeps
configuration, state and policies; -Uninstall removes saveme. Both require no
stored lock. Offline bundles
require -Archive, -Checksums and -Version.
'@

# Ánh xạ kiến trúc CPU của Windows sang tên gói.
function Get-Architecture {
    switch ($env:PROCESSOR_ARCHITECTURE) {
        'AMD64' { return 'amd64' }
        'ARM64' { return 'arm64' }
        default { throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
    }
}

# Lấy tag mới nhất từ địa chỉ chuyển hướng của trang release latest.
function Get-LatestVersion {
    $request = [Net.WebRequest]::Create("$Base/latest")
    try {
        $response = $request.GetResponse()
        $resolved = $response.ResponseUri.AbsoluteUri
        $response.Close()
    } catch {
        throw 'No published release found. Build from source instead.'
    }
    return $resolved.Substring($resolved.LastIndexOf('/') + 1)
}

# Kiểm file gói có đúng một dòng checksum và hash khớp.
function Assert-Checksum([string]$Path, [string]$Asset, [string]$Manifest) {
    $entries = @(Get-Content $Manifest | Where-Object { ($_ -split '\s+')[1] -eq $Asset })
    if ($entries.Count -ne 1) { throw 'Missing or duplicate archive checksum.' }
    $expected = ($entries[0] -split '\s+')[0]
    if ((Get-FileHash -Path $Path -Algorithm SHA256).Hash -ne $expected.ToUpperInvariant()) {
        throw 'Checksum mismatch; nothing installed.'
    }
}

# Giải nén và từ chối gói thiếu file hoặc chứa liên kết.
function Expand-Bundle([string]$Path, [string]$Destination) {
    Expand-Archive -Path $Path -DestinationPath $Destination
    foreach ($member in $Members) {
        $item = Get-Item -LiteralPath (Join-Path $Destination $member) -ErrorAction SilentlyContinue
        if (-not $item -or $item.PSIsContainer -or ($item.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
            throw "Invalid bundle file: $member"
        }
    }
}

# Mở lại script với quyền Administrator qua UAC, giữ tài khoản và tham số ban đầu; trả exit code của tiến trình đó.
function Invoke-Elevated([string]$Account) {
    $quoted = { param($value) "'" + $value.Replace("'", "''") + "'" }
    $call = "-User $(& $quoted $Account) -Elevated"
    if ($Version) { $call += " -Version $(& $quoted $Version)" }
    if ($Update) { $call += ' -Update' }
    if ($Uninstall) { $call += ' -Uninstall' }
    if ($Archive) { $call += " -Archive $(& $quoted (Resolve-Path $Archive).Path) -Checksums $(& $quoted (Resolve-Path $Checksums).Path)" }
    $source = "[scriptblock]::Create((Invoke-RestMethod -UseBasicParsing $(& $quoted $ScriptUrl)))"
    if ($PSCommandPath) { $source = "[scriptblock]::Create((Get-Content -Raw $(& $quoted $PSCommandPath)))" }
    $command = "[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12; & ($source) $call"
    $process = Start-Process -FilePath (Join-Path $PSHOME 'powershell.exe') -Verb RunAs -Wait -PassThru `
        -ArgumentList @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-Command', $command)
    return $process.ExitCode
}

if ($Help) { Write-Output $Usage; return }
if (-not $User) { $User = "$env:USERDOMAIN\$env:USERNAME" }
$principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    $code = Invoke-Elevated $User
    if ($code -ne 0) { throw "Installation failed with exit code $code." }
    return
}
# Cửa sổ quyền admin tự đóng khi xong; giữ lại để đọc lỗi nếu thất bại.
if ($Elevated) {
    trap {
        Write-Host $_ -ForegroundColor Red
        Read-Host 'Installation failed. Press Enter to close' | Out-Null
        exit 1
    }
}
$offline = $Archive -or $Checksums
if ($offline -and -not ($Archive -and $Checksums -and $Version -and (Test-Path $Archive) -and (Test-Path $Checksums))) {
    throw 'Offline install needs -Archive, -Checksums and -Version.'
}
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
$arch = Get-Architecture
if (-not $Version) { $Version = Get-LatestVersion }
if ($Version -notmatch $TagPattern) { throw 'Expected a stable version such as 0.1.0 or v0.1.0.' }
$asset = "saveme_$($Version.TrimStart('v'))_windows_$arch.zip"
$workspace = Join-Path $env:TEMP ('saveme-install-' + [guid]::NewGuid())
New-Item -ItemType Directory -Path $workspace | Out-Null
try {
    $archivePath = Join-Path $workspace $asset
    $manifest = Join-Path $workspace $ChecksumName
    if ($offline) {
        Copy-Item $Archive $archivePath
        Copy-Item $Checksums $manifest
    } else {
        Write-Output "Downloading saveme $Version for windows/$arch"
        Invoke-WebRequest -UseBasicParsing -Uri "$Base/download/$Version/$asset" -OutFile $archivePath
        Invoke-WebRequest -UseBasicParsing -Uri "$Base/download/$Version/$ChecksumName" -OutFile $manifest
    }
    Assert-Checksum $archivePath $asset $manifest
    $package = Join-Path $workspace 'payload'
    Expand-Bundle $archivePath $package
    if ((Get-Content -Raw (Join-Path $package 'VERSION')).Trim() -ne $Version) {
        throw 'Bundle version does not match the requested version.'
    }
    $shell = Join-Path $PSHOME 'powershell.exe'
    if (-not (Test-Path $shell)) { $shell = Join-Path $PSHOME 'pwsh.exe' }
    $installer = Join-Path $package 'deploy\windows\install.ps1'
    $arguments = @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $installer, '-User', $User)
    if ($Update -or $Uninstall) {
        $installer = Join-Path $package $(if ($Uninstall) { 'deploy\windows\uninstall.ps1' } else { 'deploy\windows\update.ps1' })
        $arguments = @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $installer)
    }
    & $shell @arguments
    if ($LASTEXITCODE -ne 0) { throw "Installer failed with exit code $LASTEXITCODE." }
} finally {
    Remove-Item -Recurse -Force $workspace -ErrorAction SilentlyContinue
}
