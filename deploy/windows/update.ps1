# Nâng cấp bundle có phiên bản bằng giao dịch giữ cam kết; bản source giữ luồng legacy.
$ErrorActionPreference = 'Stop'
New-Variable -Scope Local -Option ReadOnly -Name commonScript -Value ('common.ps1')
New-Variable -Scope Local -Option ReadOnly -Name autoScript -Value ('auto.ps1')
New-Variable -Scope Local -Option ReadOnly -Name versionName -Value ('VERSION')
New-Variable -Scope Local -Option ReadOnly -Name legacyScript -Value ('legacy.ps1')
New-Variable -Scope Local -Option ReadOnly -Name incomplete -Value ('Existing installation is incomplete; refusing update.')
New-Variable -Scope Local -Option ReadOnly -Name launcherName -Value ('saveme.exe')
New-Variable -Scope Local -Option ReadOnly -Name processName -Value ('saveme')
New-Variable -Scope Local -Option ReadOnly -Name runningMessage -Value ('Close all saveme windows before this one-time elevated upgrade.')
New-Variable -Scope Local -Option ReadOnly -Name helperBinary -Value ('bin\saveme-update.exe')
New-Variable -Scope Local -Option ReadOnly -Name runtimeName -Value ('runtime')
New-Variable -Scope Local -Option ReadOnly -Name currentName -Value ('current')
New-Variable -Scope Local -Option ReadOnly -Name helperName -Value ('saveme-updater')
New-Variable -Scope Local -Option ReadOnly -Name bootstrapPrefix -Value ('saveme-bootstrap-')
New-Variable -Scope Local -Option ReadOnly -Name directory -Value ('Directory')
New-Variable -Scope Local -Option ReadOnly -Name binName -Value ('bin')
New-Variable -Scope Local -Option ReadOnly -Name installFlag -Value ('-install-runtime')
New-Variable -Scope Local -Option ReadOnly -Name sourceFlag -Value ('-source')
New-Variable -Scope Local -Option ReadOnly -Name launcherBinary -Value ('bin\saveme.exe')
New-Variable -Scope Local -Option ReadOnly -Name successMessage -Value ('Updated and enabled automatic updates. Commitments and enforcement verified.')
. (Join-Path $PSScriptRoot $commonScript)
. (Join-Path $PSScriptRoot $autoScript)
$base = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
Assert-Administrator
New-Variable -Scope Local -Option ReadOnly -Name versionPath -Value (Join-Path $base $versionName)
if (-not (Test-Path $versionPath)) {
    . (Join-Path $PSScriptRoot $legacyScript)
    Install-AutoUpdater $base
    return
}
if (-not (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) -or -not (Test-Path $ConfigPath)) {
    throw $incomplete
}
New-Variable -Scope Local -Option ReadOnly -Name launcher -Value (Join-Path $ProgramDir $launcherName)
New-Variable -Scope Local -Option ReadOnly -Name running -Value (@(Get-Process -Name $processName -ErrorAction SilentlyContinue | Where-Object { $_.Path -eq $launcher }))
if ($running.Count) { throw $runningMessage }
New-Variable -Scope Local -Option ReadOnly -Name helper -Value (Join-Path $base $helperBinary)
New-Variable -Scope Local -Option ReadOnly -Name runtime -Value (Join-Path $ProgramDir $runtimeName)
New-Variable -Scope Local -Option ReadOnly -Name current -Value (Join-Path $runtime $currentName)
if (Get-Service -Name $helperName -ErrorAction SilentlyContinue) { Stop-Service -Name $helperName }
if (-not (Test-Path $current)) {
    New-Variable -Scope Local -Option ReadOnly -Name bootstrap -Value (Join-Path $env:TEMP ($bootstrapPrefix + [guid]::NewGuid()))
    try {
        New-Item -ItemType $directory -Path (Join-Path $bootstrap $binName) | Out-Null
        foreach ($binary in $Binaries) { Copy-Item (Join-Path $ProgramDir $binary) (Join-Path $bootstrap $binName) }
        Copy-Item $helper (Join-Path $bootstrap $helperBinary)
        Copy-Item (Join-Path $ProgramDir $versionName) (Join-Path $bootstrap $versionName)
        Invoke-Daemon $helper @($installFlag, $sourceFlag, $bootstrap)
    } finally { Remove-Item -Recurse -Force $bootstrap -ErrorAction SilentlyContinue }
}
Enable-TimeSync
Invoke-SystemUpgrade $helper $base
Copy-Item (Join-Path $base $launcherBinary) $launcher -Force
Install-AutoUpdater $base
Write-Output $successMessage
