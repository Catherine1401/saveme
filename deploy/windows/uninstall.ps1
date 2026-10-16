# Gỡ saveme trên Windows: từ chối khi còn khóa, rồi gỡ service, rule chặn, chính sách DoH và file.
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')

Assert-Administrator
$daemon = Join-Path $ProgramDir 'saved.exe'
if (-not (Test-Path $daemon) -or -not (Test-Path $ConfigPath)) { throw 'No saveme installation found.' }
# Từ chối gỡ khi còn domain có khóa hoặc lịch, giống điều kiện update.
Invoke-Daemon $daemon @('-check-update', '-config', $ConfigPath)
if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
    Stop-Service -Name $ServiceName -ErrorAction SilentlyContinue
    Invoke-Daemon $daemon @('-check-update', '-config', $ConfigPath)
    & sc.exe delete $ServiceName | Out-Null
}
Remove-Installation
Write-Output 'Uninstalled. Restart browsers to drop the DNS-over-HTTPS policy.'
