# Cài runtime được bảo vệ và service helper, chỉ dùng bundle đã xác minh khi nâng quyền.
function Install-AutoUpdater([string]$Bundle) {
    New-Variable -Scope Local -Option ReadOnly -Name helperName -Value ('saveme-updater')
    New-Variable -Scope Local -Option ReadOnly -Name runtimeName -Value ('runtime')
    New-Variable -Scope Local -Option ReadOnly -Name helperFile -Value ('helper.exe')
    New-Variable -Scope Local -Option ReadOnly -Name versionName -Value ('VERSION')
    New-Variable -Scope Local -Option ReadOnly -Name currentName -Value ('current')
    New-Variable -Scope Local -Option ReadOnly -Name manual -Value ('Manual')
    New-Variable -Scope Local -Option ReadOnly -Name automatic -Value ('Automatic')
    New-Variable -Scope Local -Option ReadOnly -Name systemSid -Value ('S-1-5-18')
    New-Variable -Scope Local -Option ReadOnly -Name adminSid -Value ('S-1-5-32-544')
    New-Variable -Scope Local -Option ReadOnly -Name directory -Value ('Directory')
    New-Variable -Scope Local -Option ReadOnly -Name fullControl -Value ('*{0}:(OI)(CI)F')
    New-Variable -Scope Local -Option ReadOnly -Name readExecute -Value ('*{0}:(OI)(CI)RX')
    New-Variable -Scope Local -Option ReadOnly -Name success -Value (0)
    New-Variable -Scope Local -Option ReadOnly -Name permissionFailure -Value ('Could not restrict updater permissions.')
    New-Variable -Scope Local -Option ReadOnly -Name recoveryFailure -Value ('Could not configure updater recovery.')
    New-Variable -Scope Local -Option ReadOnly -Name binaryName -Value ('bin\saveme-update.exe')
    New-Variable -Scope Local -Option ReadOnly -Name installFlag -Value ('-install-runtime')
    New-Variable -Scope Local -Option ReadOnly -Name sourceFlag -Value ('-source')
    New-Variable -Scope Local -Option ReadOnly -Name displayName -Value ('saveme automatic update helper')
    New-Variable -Scope Local -Option ReadOnly -Name binaryTemplate -Value ('"{0}" -config "{1}"')
    New-Variable -Scope Local -Option ReadOnly -Name reset -Value ('reset=')
    New-Variable -Scope Local -Option ReadOnly -Name actions -Value ('actions=')
    New-Variable -Scope Local -Option ReadOnly -Name recovery -Value ('restart/5000/restart/5000/restart/5000')
    New-Variable -Scope Local -Option ReadOnly -Name runtime -Value (Join-Path $ProgramDir $runtimeName)
    New-Variable -Scope Local -Option ReadOnly -Name helper -Value (Join-Path $runtime $helperFile)
    New-Variable -Scope Local -Option ReadOnly -Name version -Value (Join-Path $Bundle $versionName)
    if (-not (Test-Path $version)) {
        if (Get-Service -Name $helperName -ErrorAction SilentlyContinue) {
            Stop-Service -Name $helperName
            Set-Service -Name $helperName -StartupType $manual
        }
        Remove-Item (Join-Path $runtime $currentName) -Force -ErrorAction SilentlyContinue
        return
    }
    New-Variable -Scope Local -Option ReadOnly -Name existing -Value (Get-Service -Name $helperName -ErrorAction SilentlyContinue)
    if ($existing) { Stop-Service -Name $helperName }
    New-Variable -Scope Local -Option ReadOnly -Name ownerSid -Value ((Get-Content -Raw $ConfigPath | ConvertFrom-Json).allowed_sid)
    New-Item -ItemType $directory -Path $runtime -Force | Out-Null
    & icacls.exe $runtime /inheritance:r /grant:r ($fullControl -f $systemSid) ($fullControl -f $adminSid) ($readExecute -f $ownerSid) | Out-Null
    if ($LASTEXITCODE -ne $success) { throw $permissionFailure }
    Invoke-Daemon (Join-Path $Bundle $binaryName) @($installFlag, $sourceFlag, $Bundle)
    if (-not $existing) {
        New-Service -Name $helperName -DisplayName $displayName -StartupType $automatic -BinaryPathName ($binaryTemplate -f $helper, $ConfigPath) | Out-Null
    }
    & sc.exe failure $helperName $reset $success $actions $recovery | Out-Null
    if ($LASTEXITCODE -ne $success) { throw $recoveryFailure }
    Set-Service -Name $helperName -StartupType $automatic
    Start-Service -Name $helperName
}

# Giao dịch thủ công chạy dưới SYSTEM để đọc named pipe daemon, không cần mật khẩu owner.
function Invoke-SystemUpgrade([string]$Helper, [string]$Bundle) {
    New-Variable -Scope Local -Option ReadOnly -Name prefix -Value ('saveme-upgrade-')
    New-Variable -Scope Local -Option ReadOnly -Name notRun -Value (267011)
    New-Variable -Scope Local -Option ReadOnly -Name pendingStates -Value (@('Running', 'Queued'))
    New-Variable -Scope Local -Option ReadOnly -Name logName -Value ('manual-update.log')
    New-Variable -Scope Local -Option ReadOnly -Name commandTemplate -Value ('& {0} -apply-bundle {1} -config {2} *> {3}; exit $LASTEXITCODE')
    New-Variable -Scope Local -Option ReadOnly -Name powershell -Value ('powershell.exe')
    New-Variable -Scope Local -Option ReadOnly -Name argumentTemplate -Value ('-NoProfile -NonInteractive -ExecutionPolicy Bypass -EncodedCommand {0}')
    New-Variable -Scope Local -Option ReadOnly -Name system -Value ('SYSTEM')
    New-Variable -Scope Local -Option ReadOnly -Name logonType -Value ('ServiceAccount')
    New-Variable -Scope Local -Option ReadOnly -Name runLevel -Value ('Highest')
    New-Variable -Scope Local -Option ReadOnly -Name zero -Value (0)
    New-Variable -Scope Local -Option ReadOnly -Name attempts -Value (300)
    New-Variable -Scope Local -Option ReadOnly -Name interval -Value (1)
    New-Variable -Scope Local -Option ReadOnly -Name failureTemplate -Value ('Update failed; inspect {0} (result {1}).')
    New-Variable -Scope Local -Option ReadOnly -Name pendingTemplate -Value ('Update still running in SYSTEM task {0}; inspect {1}. Do not interrupt its recovery.')
    New-Variable -Scope Local -Option ReadOnly -Name name -Value ($prefix + [guid]::NewGuid())
    New-Variable -Scope Local -Option ReadOnly -Name log -Value (Join-Path $DataDir $logName)
    New-Variable -Scope Local -Option ReadOnly -Name quote -Value ({ param($text) New-Variable -Scope Local -Option ReadOnly -Name single -Value ("'"); New-Variable -Scope Local -Option ReadOnly -Name escaped -Value ("''"); $single + $text.Replace($single, $escaped) + $single })
    New-Variable -Scope Local -Option ReadOnly -Name command -Value ($commandTemplate -f (& $quote $Helper), (& $quote $Bundle), (& $quote $ConfigPath), (& $quote $log))
    New-Variable -Scope Local -Option ReadOnly -Name encoded -Value ([Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($command)))
    New-Variable -Scope Local -Option ReadOnly -Name action -Value (New-ScheduledTaskAction -Execute (Join-Path $PSHOME $powershell) -Argument ($argumentTemplate -f $encoded))
    New-Variable -Scope Local -Option ReadOnly -Name principal -Value (New-ScheduledTaskPrincipal -UserId $system -LogonType $logonType -RunLevel $runLevel)
    Register-ScheduledTask -TaskName $name -Action $action -Principal $principal | Out-Null
    $finished = $false
    try {
        Start-ScheduledTask -TaskName $name
        for ($attempt = $zero; $attempt -lt $attempts; $attempt++) {
            $info = Get-ScheduledTaskInfo -TaskName $name
            $task = Get-ScheduledTask -TaskName $name
            if ($info.LastTaskResult -ne $notRun -and $task.State -notin $pendingStates) {
                $finished = $true
                if (Test-Path $log) { Get-Content $log | Write-Host }
                if ($info.LastTaskResult -ne $zero) { throw ($failureTemplate -f $log, $info.LastTaskResult) }
                return
            }
            Start-Sleep -Seconds $interval
        }
        throw ($pendingTemplate -f $name, $log)
    } finally {
        if ($finished) { Unregister-ScheduledTask -TaskName $name -Confirm:$false }
    }
}
