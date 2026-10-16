# Gỡ saveme trên Windows: từ chối khi còn khóa, rồi gỡ service, rule chặn, chính sách DoH và file.
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')

Assert-Administrator
$daemon = Join-Path $ProgramDir 'saved.exe'
$service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if (-not ((Test-Path $ProgramDir) -or (Test-Path $DataDir) -or $service)) { throw 'No saveme installation found.' }
# Bản cài dở thiếu daemon hoặc config hợp lệ thì daemon không chạy được nên không có khóa nào đang thực thi; chỉ kiểm khóa khi đủ cả hai.
$complete = (Test-Path $daemon) -and (Test-Path $ConfigPath)
if ($complete) {
    & $daemon -check-config -config $ConfigPath | Out-Null
    $complete = $LASTEXITCODE -eq 0
}
# Từ chối gỡ khi còn domain có khóa hoặc lịch, giống điều kiện update.
if ($complete) { Invoke-Daemon $daemon @('-check-update', '-config', $ConfigPath) }
if ($service) {
    Stop-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($complete) { Invoke-Daemon $daemon @('-check-update', '-config', $ConfigPath) }
    & sc.exe delete $ServiceName | Out-Null
}
Remove-Installation
Write-Output 'Uninstalled. Restart browsers to drop the DNS-over-HTTPS policy.'
