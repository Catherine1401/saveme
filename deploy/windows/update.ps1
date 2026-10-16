# Cập nhật binary saveme; từ chối khi còn lock, khôi phục binary và state nếu service mới không chạy.
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')
$base = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)

Assert-Administrator
if (-not (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) -or -not (Test-Path $ConfigPath)) {
    throw 'Existing installation is incomplete; refusing update.'
}
$newDaemon = Join-Path $base 'bin\saved.exe'
Invoke-Daemon $newDaemon @('-check-update', '-config', $ConfigPath)
Enable-TimeSync
$backup = Join-Path $env:TEMP ('saveme-update-' + [guid]::NewGuid())
New-Item -ItemType Directory -Path (Join-Path $backup 'state') | Out-Null
$stopped = $false
$stateSaved = $false
$committed = $false
try {
    foreach ($name in $Binaries + @('VERSION')) {
        $path = Join-Path $ProgramDir $name
        if (Test-Path $path) { Copy-Item $path $backup }
    }
    $stopped = $true
    Stop-Service -Name $ServiceName
    Invoke-Daemon $newDaemon @('-check-update', '-config', $ConfigPath)
    foreach ($name in $StateFiles) {
        $path = Join-Path $StateDir $name
        if (Test-Path $path) { Copy-Item $path (Join-Path $backup 'state') }
    }
    $stateSaved = $true
    foreach ($binary in $Binaries) { Copy-Item (Join-Path $base "bin\$binary") $ProgramDir -Force }
    Start-Service -Name $ServiceName
    Wait-Ready
    if (Test-Path (Join-Path $base 'VERSION')) { Copy-Item (Join-Path $base 'VERSION') $ProgramDir -Force }
    $committed = $true
    Write-Output 'Updated. Configuration, state and browser policies preserved.'
}
finally {
    if ($stopped -and -not $committed) {
        Stop-Service -Name $ServiceName -ErrorAction SilentlyContinue
        foreach ($name in $Binaries + @('VERSION')) {
            $saved = Join-Path $backup $name
            if (Test-Path $saved) { Copy-Item $saved $ProgramDir -Force }
        }
        # Khôi phục state cũ vì service mới có thể đã ghi định dạng mà bản cũ không đọc được.
        foreach ($name in $(if ($stateSaved) { $StateFiles } else { @() })) {
            $path = Join-Path $StateDir $name
            $saved = Join-Path (Join-Path $backup 'state') $name
            if (Test-Path $path) { Set-ItemProperty -Path $path -Name IsReadOnly -Value $false }
            if (Test-Path $saved) {
                Copy-Item $saved $path -Force
                Set-ItemProperty -Path $path -Name IsReadOnly -Value $true
            } elseif (Test-Path $path) {
                Remove-Item $path -Force
            }
        }
        Start-Service -Name $ServiceName -ErrorAction SilentlyContinue
        Write-Error 'Update failed; restored previous binaries and state. Inspect saved.log.' -ErrorAction Continue
    }
    Remove-Item -Recurse -Force $backup -ErrorAction SilentlyContinue
}
