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

# Bật Windows Time tự khởi động và đồng bộ ngay; máy không vào domain thì dịch vụ này không tự chạy.
function Enable-TimeSync {
    $timeService = 'w32time'
    $warning = 'Windows Time has not synchronized yet; locks stay on and only adding domains works until it does.'
    Set-Service -Name $timeService -StartupType Automatic -ErrorAction SilentlyContinue
    Start-Service -Name $timeService -ErrorAction SilentlyContinue
    & w32tm.exe /resync /force | Out-Null
    if ($LASTEXITCODE -ne 0) { Write-Warning $warning }
}

# Thêm ($Add) hoặc gỡ thư mục cài khỏi Path của máy, giữ nguyên các biến %...% chưa mở rộng.
function Set-ProgramPath([bool]$Add) {
    $key = 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Environment'
    $name = 'Path'
    $separator = ';'
    $raw = (Get-Item $key).GetValue($name, '', 'DoNotExpandEnvironmentNames')
    $entries = @($raw -split $separator | Where-Object { $_ -and $_.TrimEnd('\') -ne $ProgramDir })
    if ($Add) { $entries += $ProgramDir }
    Set-ItemProperty -Path $key -Name $name -Value ($entries -join $separator) -Type ExpandString
    Send-EnvironmentChange
}

# Báo các cửa sổ đang mở rằng biến môi trường đã đổi để terminal mới nhận Path mới.
function Send-EnvironmentChange {
    $broadcast = [IntPtr]0xffff
    $settingChange = 0x1A
    $abortIfHung = 2
    $timeoutMs = 5000
    if (-not ('SaveMe.Native' -as [type])) {
        Add-Type -Namespace SaveMe -Name Native -MemberDefinition '[DllImport("user32.dll", CharSet = CharSet.Unicode)] public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint msg, UIntPtr wParam, string lParam, uint flags, uint timeout, out UIntPtr result);'
    }
    $result = [UIntPtr]::Zero
    [SaveMe.Native]::SendMessageTimeout($broadcast, $settingChange, [UIntPtr]::Zero, 'Environment', $abortIfHung, $timeoutMs, [ref]$result) | Out-Null
}

# Gỡ rule chặn, NRPT, chính sách DoH và file của saveme; dùng cho uninstall và rollback khi cài hỏng.
function Remove-Installation {
    $group = 'saveme'
    Get-NetFirewallRule -Group $group -ErrorAction SilentlyContinue | Remove-NetFirewallRule
    Get-DnsClientNrptRule | Where-Object Comment -eq $group | ForEach-Object { Remove-DnsClientNrptRule -Name $_.Name -Force }
    Clear-DnsClientCache
    foreach ($key in @('HKLM:\SOFTWARE\Policies\Google\Chrome', 'HKLM:\SOFTWARE\Policies\Microsoft\Edge', 'HKLM:\SOFTWARE\Policies\Chromium')) {
        Remove-ItemProperty -Path $key -Name 'DnsOverHttpsMode' -ErrorAction SilentlyContinue
    }
    Remove-Item -Path 'HKLM:\SOFTWARE\Policies\Mozilla\Firefox\DNSOverHTTPS' -Recurse -ErrorAction SilentlyContinue
    Set-ProgramPath $false
    foreach ($path in @($ProgramDir, $DataDir)) {
        if (Test-Path $path) {
            Get-ChildItem -Path $path -Recurse -File | ForEach-Object { $_.IsReadOnly = $false }
            Remove-Item -Path $path -Recurse -Force
        }
    }
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
