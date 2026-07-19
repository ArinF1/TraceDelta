[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$libraryPath = Join-Path $PSScriptRoot 'windows-check-lib.ps1'
. $libraryPath

$testRoot = Join-Path (Join-Path $repositoryRoot 'tmp') ('windows-check-tests-' + [guid]::NewGuid().ToString('N'))
$null = New-Item -ItemType Directory -Path $testRoot -Force
$heldMutex = $null
try {
    $commandInterpreter = Join-Path $env:SystemRoot 'System32\cmd.exe'
    $powershell = Join-Path $PSHOME 'powershell.exe'
    $probeDirectory = Join-Path $testRoot 'probe files'
    $null = New-Item -ItemType Directory -Path $probeDirectory -Force

    Write-Output 'Testing successful child process...'
    $success = Invoke-TraceDeltaBoundedProcess `
        -StepName 'success probe' `
        -FilePath $commandInterpreter `
        -ArgumentList @('/d', '/c', 'exit', '0') `
        -WorkingDirectory $repositoryRoot `
        -TimeoutSeconds 5
    if ($success.ExitCode -ne 0) {
        throw "success probe returned exit code $($success.ExitCode)."
    }

    Write-Output 'Testing failing child process...'
    $failureObserved = $false
    try {
        $null = Invoke-TraceDeltaBoundedProcess `
            -StepName 'failure probe' `
            -FilePath $commandInterpreter `
            -ArgumentList @('/d', '/c', 'exit', '7') `
            -WorkingDirectory $repositoryRoot `
            -TimeoutSeconds 5
    }
    catch {
        if ($_.Exception.Message -notmatch 'failure probe failed with exit code 7') {
            throw
        }
        $failureObserved = $true
    }
    if (-not $failureObserved) {
        throw 'failure probe unexpectedly succeeded.'
    }

    Write-Output 'Testing gated child launch...'
    $gateName = 'Local\TraceDelta-Test-Gate-' + [guid]::NewGuid().ToString('N')
    $readyGateName = 'Local\TraceDelta-Test-Ready-' + [guid]::NewGuid().ToString('N')
    $gate = New-Object System.Threading.EventWaitHandle -ArgumentList @(
        $false,
        [System.Threading.EventResetMode]::ManualReset,
        $gateName
    )
    $readyGate = New-Object System.Threading.EventWaitHandle -ArgumentList @(
        $false,
        [System.Threading.EventResetMode]::ManualReset,
        $readyGateName
    )
    $gateJob = [IntPtr]::Zero
    $gateRunner = New-Object System.Diagnostics.Process
    $gateRunnerStarted = $false
    $markerPath = Join-Path $probeDirectory 'gate-marker.txt'
    $markerCommand = 'echo started>"' + $markerPath + '"'
    $targetArgumentJson = ConvertTo-Json -InputObject ([object[]]@(
        '/d',
        '/c',
        $markerCommand
    )) -Compress
    $targetArgumentsBase64 = [System.Convert]::ToBase64String(
        [System.Text.Encoding]::UTF8.GetBytes($targetArgumentJson)
    )
    $gateRunnerArguments = @(
        '-NoLogo',
        '-NoProfile',
        '-NonInteractive',
        '-ExecutionPolicy', 'Bypass',
        '-File', (Join-Path $PSScriptRoot 'windows-check-child.ps1'),
        '-StartGateName', $gateName,
        '-ReadyGateName', $readyGateName,
        '-FilePath', $commandInterpreter,
        '-WorkingDirectory', $repositoryRoot,
        '-ArgumentsBase64', $targetArgumentsBase64
    )
    $gateRunnerStartInfo = New-Object System.Diagnostics.ProcessStartInfo
    $gateRunnerStartInfo.FileName = $powershell
    $gateRunnerStartInfo.Arguments = (@($gateRunnerArguments | ForEach-Object {
        ConvertTo-TraceDeltaProcessArgument -Argument $_
    })) -join ' '
    $gateRunnerStartInfo.WorkingDirectory = $repositoryRoot
    $gateRunnerStartInfo.UseShellExecute = $false
    $gateRunnerStartInfo.CreateNoWindow = $true
    $gateRunnerStartInfo.RedirectStandardOutput = $true
    $gateRunnerStartInfo.RedirectStandardError = $true
    $gateRunner.StartInfo = $gateRunnerStartInfo
    try {
        $gateJob = [TraceDelta.WindowsJobNative]::CreateKillOnCloseJob()
        if (-not $gateRunner.Start()) {
            throw 'gated child runner did not start.'
        }
        $gateRunnerStarted = $true
        $gateOutput = $gateRunner.StandardOutput.ReadToEndAsync()
        $gateError = $gateRunner.StandardError.ReadToEndAsync()
        if (-not [TraceDelta.WindowsJobNative]::AssignProcessToJobObject($gateJob, $gateRunner.Handle)) {
            $win32Error = [System.Runtime.InteropServices.Marshal]::GetLastWin32Error()
            throw "gated child runner assignment failed with Win32 error $win32Error."
        }
        if (-not $readyGate.WaitOne(5000)) {
            throw 'gated child runner did not signal readiness before its start gate.'
        }
        Start-Sleep -Milliseconds 250
        if (Test-Path -LiteralPath $markerPath) {
            throw 'gated child executed its target before the start gate was released.'
        }
        if (-not $gate.Set()) {
            throw 'could not release the gated child test.'
        }
        if (-not $gateRunner.WaitForExit(5000)) {
            throw 'gated child runner did not exit after its start gate was released.'
        }
        $gateRunner.WaitForExit()
        $gateStreams = [System.Threading.Tasks.Task[]]@($gateOutput, $gateError)
        if (-not [System.Threading.Tasks.Task]::WaitAll($gateStreams, 5000)) {
            throw 'gated child runner output did not close.'
        }
        if ($gateRunner.ExitCode -ne 0) {
            throw "gated child runner failed with exit code $($gateRunner.ExitCode): $($gateError.Result)"
        }
        if (-not (Test-Path -LiteralPath $markerPath)) {
            throw 'gated child did not execute its target after the start gate was released.'
        }
        $jobEmptyWait = [System.Diagnostics.Stopwatch]::StartNew()
        while (
            [TraceDelta.WindowsJobNative]::GetActiveProcessCount($gateJob) -ne 0 -and
            $jobEmptyWait.Elapsed.TotalSeconds -lt 2
        ) {
            Start-Sleep -Milliseconds 25
        }
        if ([TraceDelta.WindowsJobNative]::GetActiveProcessCount($gateJob) -ne 0) {
            throw 'gated child job still contained an active process after completion.'
        }
    }
    finally {
        if ($gateRunnerStarted -and -not $gateRunner.HasExited) {
            $gateRunner.Kill()
            [void]$gateRunner.WaitForExit(5000)
        }
        if ($gateJob -ne [IntPtr]::Zero) {
            [void][TraceDelta.WindowsJobNative]::CloseHandle($gateJob)
        }
        $gate.Dispose()
        $readyGate.Dispose()
        $gateRunner.Dispose()
    }

    Write-Output 'Testing timed-out child process tree...'
    $timeoutProbe = Join-Path $probeDirectory 'timeout-probe.ps1'
    $childPidFile = Join-Path $probeDirectory 'child-pid.txt'
    @'
param([Parameter(Mandatory = $true)][string]$ChildPidFile)
$ping = [System.Diagnostics.Process]::Start(
    (Join-Path $env:SystemRoot 'System32\PING.EXE'),
    '-n 30 127.0.0.1'
)
[System.IO.File]::WriteAllText($ChildPidFile, [string]$ping.Id)
Start-Sleep -Seconds 30
'@ | Set-Content -LiteralPath $timeoutProbe -Encoding UTF8

    $timeoutObserved = $false
    try {
        $null = Invoke-TraceDeltaBoundedProcess `
            -StepName 'timeout probe' `
            -FilePath $powershell `
            -ArgumentList @(
                '-NoLogo',
                '-NoProfile',
                '-NonInteractive',
                '-ExecutionPolicy', 'Bypass',
                '-File', $timeoutProbe,
                '-ChildPidFile', $childPidFile
            ) `
            -WorkingDirectory $repositoryRoot `
            -TimeoutSeconds 10
    }
    catch {
        if ($_.Exception.Message -notmatch 'timeout probe timed out after 10 second') {
            throw
        }
        if ($_.Exception.Message -notmatch 'process-tree cleanup succeeded') {
            throw
        }
        $timeoutObserved = $true
    }
    if (-not $timeoutObserved) {
        throw 'timeout probe unexpectedly succeeded.'
    }
    if (-not (Test-Path -LiteralPath $childPidFile)) {
        throw 'timeout probe did not record its child process ID.'
    }
    $childProcessId = [int]([System.IO.File]::ReadAllText($childPidFile))
    if ($null -ne (Get-Process -Id $childProcessId -ErrorAction SilentlyContinue)) {
        throw "timeout probe left child process $childProcessId running."
    }

    Write-Output 'Testing concurrent validation lock...'
    $mutexScope = Join-Path $testRoot 'mutex scope'
    $heldMutex = Enter-TraceDeltaValidationMutex -RepositoryRoot $mutexScope -LockTimeoutSeconds 1
    $mutexProbe = Join-Path $probeDirectory 'mutex-probe.ps1'
    @'
param(
    [Parameter(Mandatory = $true)][string]$LibraryPath,
    [Parameter(Mandatory = $true)][string]$RepositoryRoot
)
. $LibraryPath
$mutex = $null
try {
    $mutex = Enter-TraceDeltaValidationMutex -RepositoryRoot $RepositoryRoot -LockTimeoutSeconds 1
}
catch {
    [Console]::Error.WriteLine($_.Exception.Message)
    exit 23
}
finally {
    if ($null -ne $mutex) {
        $mutex.ReleaseMutex()
        $mutex.Dispose()
    }
}
'@ | Set-Content -LiteralPath $mutexProbe -Encoding UTF8

    $lockFailureObserved = $false
    try {
        $null = Invoke-TraceDeltaBoundedProcess `
            -StepName 'lock probe' `
            -FilePath $powershell `
            -ArgumentList @(
                '-NoLogo',
                '-NoProfile',
                '-NonInteractive',
                '-ExecutionPolicy', 'Bypass',
                '-File', $mutexProbe,
                '-LibraryPath', $libraryPath,
                '-RepositoryRoot', $mutexScope
            ) `
            -WorkingDirectory $repositoryRoot `
            -TimeoutSeconds 5
    }
    catch {
        if ($_.Exception.Message -notmatch 'lock probe failed with exit code 23') {
            throw
        }
        $lockFailureObserved = $true
    }
    if (-not $lockFailureObserved) {
        throw 'lock probe unexpectedly acquired a held validation mutex.'
    }

    Write-Output 'Windows validation watchdog tests passed.'
}
finally {
    if ($null -ne $heldMutex) {
        try {
            $heldMutex.ReleaseMutex()
        }
        finally {
            $heldMutex.Dispose()
        }
    }
    Remove-TraceDeltaValidationDirectory -RepositoryRoot $repositoryRoot -Directory $testRoot
}
