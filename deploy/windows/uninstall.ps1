# Gỡ saveme trên Windows: từ chối khi còn khóa, rồi gỡ service, rule chặn, chính sách DoH và file.
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')
$group = 'saveme'

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
Get-NetFirewallRule -Group $group -ErrorAction SilentlyContinue | Remove-NetFirewallRule
Get-DnsClientNrptRule | Where-Object Comment -eq $group | ForEach-Object { Remove-DnsClientNrptRule -Name $_.Name -Force }
Clear-DnsClientCache
foreach ($key in @('HKLM:\SOFTWARE\Policies\Google\Chrome', 'HKLM:\SOFTWARE\Policies\Microsoft\Edge', 'HKLM:\SOFTWARE\Policies\Chromium')) {
    Remove-ItemProperty -Path $key -Name 'DnsOverHttpsMode' -ErrorAction SilentlyContinue
}
Remove-Item -Path 'HKLM:\SOFTWARE\Policies\Mozilla\Firefox\DNSOverHTTPS' -Recurse -ErrorAction SilentlyContinue
foreach ($path in @($ProgramDir, $DataDir)) {
    if (Test-Path $path) {
        Get-ChildItem -Path $path -Recurse -File | ForEach-Object { $_.IsReadOnly = $false }
        Remove-Item -Path $path -Recurse -Force
    }
}
Write-Output 'Uninstalled. Restart browsers to drop the DNS-over-HTTPS policy.'
