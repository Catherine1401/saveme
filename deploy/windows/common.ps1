# Hằng số và hàm dùng chung cho install.ps1 và update.ps1.
$ServiceName = 'saved'
$ProgramDir = Join-Path $env:ProgramFiles 'saveme'
$DataDir = 'C:\ProgramData\saveme'
$ConfigPath = Join-Path $DataDir 'config.json'
$StateDir = Join-Path $DataDir 'state'
$StateFiles = @('state.json', 'state.bak.json')
$Binaries = @('saved.exe', 'saveme.exe')

# Dừng script nếu không chạy bằng Administrator.
function Assert-Administrator {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($identity)
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        throw 'Run from an elevated PowerShell (Run as administrator).'
    }
}

# Chạy saved.exe và ném lỗi khi exit code khác 0.
function Invoke-Daemon([string]$Daemon, [string[]]$Arguments) {
    & $Daemon @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$Daemon $($Arguments -join ' ') failed with exit code $LASTEXITCODE." }
}

# Chờ service ở trạng thái Running ổn định trong khoảng thời gian cho trước.
function Wait-Ready([int]$Attempts = 10) {
    for ($attempt = 0; $attempt -lt $Attempts; $attempt++) {
        Start-Sleep -Seconds 1
        if ((Get-Service -Name $ServiceName).Status -eq 'Running') {
            Start-Sleep -Seconds 1
            if ((Get-Service -Name $ServiceName).Status -eq 'Running') { return }
        }
    }
    throw "Service $ServiceName did not become ready. Inspect $DataDir\saved.log."
}
