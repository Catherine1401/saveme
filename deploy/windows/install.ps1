# Cài saveme trên Windows: copy binary, tạo config theo SID, khóa ACL, đăng ký service và chính sách DoH.
param([Parameter(Mandatory = $true)][string]$User)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')
$base = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$systemSid = 'S-1-5-18'
$adminSid = 'S-1-5-32-544'

Assert-Administrator
$sid = (New-Object Security.Principal.NTAccount($User)).Translate([Security.Principal.SecurityIdentifier]).Value
if ($sid -eq $systemSid) { throw 'Select an ordinary account, not SYSTEM.' }
foreach ($binary in $Binaries) {
    if (-not (Test-Path (Join-Path $base "bin\$binary"))) { throw "Missing bin\$binary in the bundle." }
}
if ((Test-Path $ProgramDir) -or (Test-Path $DataDir) -or (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue)) {
    throw 'An existing installation was found; use update.ps1 instead.'
}

$committed = $false
try {
    New-Item -ItemType Directory -Path $ProgramDir, $StateDir | Out-Null
    foreach ($binary in $Binaries) { Copy-Item (Join-Path $base "bin\$binary") $ProgramDir }
    if (Test-Path (Join-Path $base 'VERSION')) { Copy-Item (Join-Path $base 'VERSION') $ProgramDir }
    Set-ProgramPath $true
    $config = Get-Content -Raw (Join-Path $PSScriptRoot 'config.json') | ConvertFrom-Json
    $config.allowed_sid = $sid
    $config.state_dir = $StateDir
    # Ghi UTF-8 không BOM để daemon Go đọc được.
    [IO.File]::WriteAllText($ConfigPath, ($config | ConvertTo-Json -Depth 4))
    & icacls.exe $DataDir /inheritance:r /grant:r "*${systemSid}:(OI)(CI)F" "*${adminSid}:(OI)(CI)F" "*${sid}:(OI)(CI)RX" | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Could not restrict permissions on the data directory.' }

    $daemon = Join-Path $ProgramDir 'saved.exe'
    Invoke-Daemon $daemon @('-check-config', '-config', $ConfigPath)
    New-Service -Name $ServiceName -DisplayName 'saveme daemon' -StartupType Automatic -BinaryPathName "`"$daemon`" -config `"$ConfigPath`"" | Out-Null
    & sc.exe failure $ServiceName reset= 0 actions= restart/5000/restart/5000/restart/5000 | Out-Null

    # Tắt DNS-over-HTTPS của trình duyệt để mọi truy vấn đi qua resolver cục bộ.
    $chromium = @('HKLM:\SOFTWARE\Policies\Google\Chrome', 'HKLM:\SOFTWARE\Policies\Microsoft\Edge', 'HKLM:\SOFTWARE\Policies\Chromium')
    foreach ($key in $chromium) {
        New-Item -Path $key -Force | Out-Null
        New-ItemProperty -Path $key -Name 'DnsOverHttpsMode' -Value 'off' -PropertyType String -Force | Out-Null
    }
    $firefox = 'HKLM:\SOFTWARE\Policies\Mozilla\Firefox\DNSOverHTTPS'
    New-Item -Path $firefox -Force | Out-Null
    New-ItemProperty -Path $firefox -Name 'Enabled' -Value 0 -PropertyType DWord -Force | Out-Null
    New-ItemProperty -Path $firefox -Name 'Locked' -Value 1 -PropertyType DWord -Force | Out-Null

    Start-Service -Name $ServiceName
    Wait-Ready
    $committed = $true
}
finally {
    # Cài hỏng giữa chừng: gỡ service và mọi thứ đã tạo để có thể chạy lại install.ps1.
    if (-not $committed) {
        if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
            Stop-Service -Name $ServiceName -ErrorAction SilentlyContinue
            & sc.exe delete $ServiceName | Out-Null
        }
        Remove-Installation
        Write-Error 'Installation failed; removed the partial installation.' -ErrorAction Continue
    }
}
Write-Output "Installed. Run `"$ProgramDir\saveme.exe`" as $User and restart browsers before locking."
