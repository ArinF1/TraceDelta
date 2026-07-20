Set-StrictMode -Version Latest

if ($null -eq ('TraceDelta.WindowsJobNative' -as [type])) {
    Add-Type -TypeDefinition @'
using System;
using System.ComponentModel;
using System.Runtime.InteropServices;

namespace TraceDelta
{
    public static class WindowsJobNative
    {
        private const uint KillOnJobClose = 0x00002000;
        private const int BasicAccountingInformationClass = 1;
        private const int ExtendedLimitInformation = 9;

        [StructLayout(LayoutKind.Sequential)]
        private struct BasicAccountingInformation
        {
            public long TotalUserTime;
            public long TotalKernelTime;
            public long ThisPeriodTotalUserTime;
            public long ThisPeriodTotalKernelTime;
            public uint TotalPageFaultCount;
            public uint TotalProcesses;
            public uint ActiveProcesses;
            public uint TotalTerminatedProcesses;
        }

        [StructLayout(LayoutKind.Sequential)]
        private struct IoCounters
        {
            public ulong ReadOperationCount;
            public ulong WriteOperationCount;
            public ulong OtherOperationCount;
            public ulong ReadTransferCount;
            public ulong WriteTransferCount;
            public ulong OtherTransferCount;
        }

        [StructLayout(LayoutKind.Sequential)]
        private struct BasicLimits
        {
            public long PerProcessUserTimeLimit;
            public long PerJobUserTimeLimit;
            public uint LimitFlags;
            public UIntPtr MinimumWorkingSetSize;
            public UIntPtr MaximumWorkingSetSize;
            public uint ActiveProcessLimit;
            public UIntPtr Affinity;
            public uint PriorityClass;
            public uint SchedulingClass;
        }

        [StructLayout(LayoutKind.Sequential)]
        private struct ExtendedLimits
        {
            public BasicLimits BasicLimitInformation;
            public IoCounters IoInfo;
            public UIntPtr ProcessMemoryLimit;
            public UIntPtr JobMemoryLimit;
            public UIntPtr PeakProcessMemoryUsed;
            public UIntPtr PeakJobMemoryUsed;
        }

        [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
        private static extern IntPtr CreateJobObject(IntPtr attributes, string name);

        [DllImport("kernel32.dll", SetLastError = true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool SetInformationJobObject(
            IntPtr job,
            int informationClass,
            IntPtr information,
            uint informationLength);

        [DllImport("kernel32.dll", SetLastError = true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        public static extern bool AssignProcessToJobObject(IntPtr job, IntPtr process);

        [DllImport("kernel32.dll", SetLastError = true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        public static extern bool TerminateJobObject(IntPtr job, uint exitCode);

        [DllImport("kernel32.dll", SetLastError = true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool QueryInformationJobObject(
            IntPtr job,
            int informationClass,
            out BasicAccountingInformation information,
            uint informationLength,
            IntPtr returnLength);

        [DllImport("kernel32.dll", SetLastError = true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        public static extern bool CloseHandle(IntPtr handle);

        public static IntPtr CreateKillOnCloseJob()
        {
            IntPtr job = CreateJobObject(IntPtr.Zero, null);
            if (job == IntPtr.Zero)
            {
                throw new Win32Exception(Marshal.GetLastWin32Error());
            }

            ExtendedLimits limits = new ExtendedLimits();
            limits.BasicLimitInformation.LimitFlags = KillOnJobClose;
            int size = Marshal.SizeOf(typeof(ExtendedLimits));
            IntPtr memory = Marshal.AllocHGlobal(size);
            try
            {
                Marshal.StructureToPtr(limits, memory, false);
                if (!SetInformationJobObject(job, ExtendedLimitInformation, memory, (uint)size))
                {
                    throw new Win32Exception(Marshal.GetLastWin32Error());
                }
                return job;
            }
            catch
            {
                CloseHandle(job);
                throw;
            }
            finally
            {
                Marshal.FreeHGlobal(memory);
            }
        }

        public static uint GetActiveProcessCount(IntPtr job)
        {
            BasicAccountingInformation information;
            uint size = (uint)Marshal.SizeOf(typeof(BasicAccountingInformation));
            if (!QueryInformationJobObject(
                job,
                BasicAccountingInformationClass,
                out information,
                size,
                IntPtr.Zero))
            {
                throw new Win32Exception(Marshal.GetLastWin32Error());
            }
            return information.ActiveProcesses;
        }
    }
}
'@
}

$script:TraceDeltaWindowsChildPath = Join-Path $PSScriptRoot 'windows-check-child.ps1'

function ConvertTo-TraceDeltaProcessArgument {
    [OutputType([string])]
    param(
        [AllowEmptyString()]
        [string]$Argument
    )

    if ($null -eq $Argument -or $Argument.Length -eq 0) {
        return '""'
    }
    if ($Argument -notmatch '[\s"]') {
        return $Argument
    }

    $result = New-Object System.Text.StringBuilder
    [void]$result.Append('"')
    $backslashes = 0
    foreach ($character in $Argument.ToCharArray()) {
        if ($character -eq '\') {
            $backslashes++
            continue
        }
        if ($character -eq '"') {
            if ($backslashes -gt 0) {
                [void]$result.Append((('\' * ($backslashes * 2)) -join ''))
            }
            [void]$result.Append('\"')
            $backslashes = 0
            continue
        }
        if ($backslashes -gt 0) {
            [void]$result.Append((('\' * $backslashes) -join ''))
            $backslashes = 0
        }
        [void]$result.Append($character)
    }
    if ($backslashes -gt 0) {
        [void]$result.Append((('\' * ($backslashes * 2)) -join ''))
    }
    [void]$result.Append('"')
    return $result.ToString()
}

function Resolve-TraceDeltaApplicationPath {
    [OutputType([string])]
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $commands = @(Get-Command $Name -CommandType Application -ErrorAction Stop)
    if ($commands.Count -eq 0) {
        throw "could not resolve required application $Name."
    }
    return $commands[0].Source
}

function Stop-TraceDeltaProcessTree {
    [OutputType([bool])]
    param(
        [Parameter(Mandatory = $true)]
        [IntPtr]$JobHandle,

        [ValidateRange(1, 30)]
        [int]$TimeoutSeconds = 5
    )

    if (-not [TraceDelta.WindowsJobNative]::TerminateJobObject($JobHandle, 1460)) {
        $win32Error = [System.Runtime.InteropServices.Marshal]::GetLastWin32Error()
        throw "TerminateJobObject failed with Win32 error $win32Error."
    }

    $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
    do {
        if ([TraceDelta.WindowsJobNative]::GetActiveProcessCount($JobHandle) -eq 0) {
            return $true
        }
        Start-Sleep -Milliseconds 25
    }
    while ($stopwatch.Elapsed.TotalSeconds -lt $TimeoutSeconds)

    return $false
}

function Write-TraceDeltaCapturedOutput {
    param(
        [AllowEmptyString()]
        [string]$StandardOutput,

        [AllowEmptyString()]
        [string]$StandardError
    )

    if (-not [string]::IsNullOrEmpty($StandardOutput)) {
        [Console]::Out.Write($StandardOutput)
    }
    if (-not [string]::IsNullOrEmpty($StandardError)) {
        [Console]::Error.Write($StandardError)
    }
}

function Invoke-TraceDeltaBoundedProcess {
    [OutputType([pscustomobject])]
    param(
        [Parameter(Mandatory = $true)]
        [string]$StepName,

        [Parameter(Mandatory = $true)]
        [string]$FilePath,

        [string[]]$ArgumentList = @(),

        [Parameter(Mandatory = $true)]
        [string]$WorkingDirectory,

        [ValidateRange(1, 3600)]
        [int]$TimeoutSeconds = 300
    )

    if (-not (Test-Path -LiteralPath $script:TraceDeltaWindowsChildPath -PathType Leaf)) {
        throw "$StepName could not find its Windows child launcher at $script:TraceDeltaWindowsChildPath."
    }

    $argumentJson = ConvertTo-Json -InputObject ([object[]]$ArgumentList) -Compress
    $argumentsBase64 = [System.Convert]::ToBase64String(
        [System.Text.Encoding]::UTF8.GetBytes($argumentJson)
    )
    $startGateName = 'Local\TraceDelta-Child-' + [guid]::NewGuid().ToString('N')
    $windowsPowerShell = Join-Path $env:SystemRoot 'System32\WindowsPowerShell\v1.0\powershell.exe'
    $childArguments = @(
        '-NoLogo',
        '-NoProfile',
        '-NonInteractive',
        '-ExecutionPolicy', 'Bypass',
        '-File', $script:TraceDeltaWindowsChildPath,
        '-StartGateName', $startGateName,
        '-FilePath', $FilePath,
        '-WorkingDirectory', $WorkingDirectory,
        '-ArgumentsBase64', $argumentsBase64
    )
    $quotedArguments = @($childArguments | ForEach-Object {
        ConvertTo-TraceDeltaProcessArgument -Argument $_
    })
    $startInfo = New-Object System.Diagnostics.ProcessStartInfo
    $startInfo.FileName = $windowsPowerShell
    $startInfo.Arguments = $quotedArguments -join ' '
    $startInfo.WorkingDirectory = $WorkingDirectory
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true

    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $startInfo
    $jobHandle = [IntPtr]::Zero
    $startGate = $null
    try {
        try {
            $startGate = New-Object System.Threading.EventWaitHandle -ArgumentList @(
                $false,
                [System.Threading.EventResetMode]::ManualReset,
                $startGateName
            )
        }
        catch {
            throw "$StepName could not create its Windows process start gate: $($_.Exception.Message)"
        }

        try {
            $jobHandle = [TraceDelta.WindowsJobNative]::CreateKillOnCloseJob()
        }
        catch {
            throw "$StepName could not create its Windows process watchdog: $($_.Exception.Message)"
        }

        try {
            if (-not $process.Start()) {
                throw "process did not start"
            }
        }
        catch {
            throw "$StepName could not start ${FilePath}: $($_.Exception.Message)"
        }

        $processId = $process.Id
        if (-not [TraceDelta.WindowsJobNative]::AssignProcessToJobObject($jobHandle, $process.Handle)) {
            $win32Error = [System.Runtime.InteropServices.Marshal]::GetLastWin32Error()
            $rootStopped = $process.HasExited
            if (-not $rootStopped) {
                try {
                    $process.Kill()
                    $rootStopped = $process.WaitForExit(5000)
                }
                catch {
                    $rootStopped = $false
                }
            }
            $cleanup = if ($rootStopped) { 'succeeded' } else { 'failed' }
            throw "$StepName could not enter its Windows process watchdog (Win32 error $win32Error); root-process cleanup $cleanup."
        }

        $stdoutTask = $process.StandardOutput.ReadToEndAsync()
        $stderrTask = $process.StandardError.ReadToEndAsync()
        if (-not $startGate.Set()) {
            throw "$StepName could not release its Windows process start gate."
        }
        if (-not $process.WaitForExit($TimeoutSeconds * 1000)) {
            $treeStopError = $null
            try {
                $treeStopped = Stop-TraceDeltaProcessTree -JobHandle $jobHandle
            }
            catch {
                $treeStopped = $false
                $treeStopError = $_.Exception.Message
            }
            $processExited = $process.WaitForExit(5000)
            $streams = [System.Threading.Tasks.Task[]]@($stdoutTask, $stderrTask)
            $streamsDrained = [System.Threading.Tasks.Task]::WaitAll($streams, 5000)
            if ($streamsDrained) {
                Write-TraceDeltaCapturedOutput -StandardOutput $stdoutTask.Result -StandardError $stderrTask.Result
            }
            $cleanup = if ($treeStopped -and $processExited -and $streamsDrained) { 'succeeded' } else { 'failed' }
            $watchdogDetail = if ($null -ne $treeStopError) { " Watchdog error: $treeStopError" } else { '' }
            throw "$StepName timed out after $TimeoutSeconds second(s) (PID $processId); process-tree cleanup $cleanup.$watchdogDetail"
        }

        $process.WaitForExit()
        $outputStreams = [System.Threading.Tasks.Task[]]@($stdoutTask, $stderrTask)
        if (-not [System.Threading.Tasks.Task]::WaitAll($outputStreams, 5000)) {
            throw "$StepName exited but its output streams did not close within 5 seconds (PID $processId)."
        }

        $standardOutput = $stdoutTask.Result
        $standardError = $stderrTask.Result
        Write-TraceDeltaCapturedOutput -StandardOutput $standardOutput -StandardError $standardError
        if ($process.ExitCode -ne 0) {
            throw "$StepName failed with exit code $($process.ExitCode)."
        }

        return [pscustomobject]@{
            ExitCode       = $process.ExitCode
            ProcessId      = $processId
            StandardOutput = $standardOutput
            StandardError  = $standardError
        }
    }
    finally {
        if ($jobHandle -ne [IntPtr]::Zero) {
            [void][TraceDelta.WindowsJobNative]::CloseHandle($jobHandle)
        }
        if ($null -ne $startGate) {
            $startGate.Dispose()
        }
        $process.Dispose()
    }
}

function Get-TraceDeltaValidationMutexName {
    [OutputType([string])]
    param(
        [Parameter(Mandatory = $true)]
        [string]$RepositoryRoot
    )

    $normalizedRoot = [System.IO.Path]::GetFullPath($RepositoryRoot).TrimEnd('\').ToLowerInvariant()
    $sha256 = [System.Security.Cryptography.SHA256]::Create()
    try {
        $hash = $sha256.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($normalizedRoot))
    }
    finally {
        $sha256.Dispose()
    }
    $suffix = [System.BitConverter]::ToString($hash, 0, 12).Replace('-', '')
    return "Local\TraceDelta-Check-$suffix"
}

function Enter-TraceDeltaValidationMutex {
    [OutputType([System.Threading.Mutex])]
    param(
        [Parameter(Mandatory = $true)]
        [string]$RepositoryRoot,

        [ValidateRange(0, 300)]
        [int]$LockTimeoutSeconds = 5
    )

    $mutexName = Get-TraceDeltaValidationMutexName -RepositoryRoot $RepositoryRoot
    $mutex = New-Object System.Threading.Mutex -ArgumentList $false, $mutexName
    try {
        try {
            $acquired = $mutex.WaitOne($LockTimeoutSeconds * 1000)
        }
        catch [System.Threading.AbandonedMutexException] {
            $acquired = $true
        }
        if (-not $acquired) {
            throw "another TraceDelta Windows validation is already running; lock wait exceeded $LockTimeoutSeconds second(s)."
        }
        return ,$mutex
    }
    catch {
        $mutex.Dispose()
        throw
    }
}

function Remove-TraceDeltaValidationDirectory {
    param(
        [Parameter(Mandatory = $true)]
        [string]$RepositoryRoot,

        [Parameter(Mandatory = $true)]
        [string]$Directory
    )

    if (-not (Test-Path -LiteralPath $Directory)) {
        return
    }
    $temporaryRoot = [System.IO.Path]::GetFullPath((Join-Path $RepositoryRoot 'tmp'))
    $candidate = [System.IO.Path]::GetFullPath($Directory)
    $prefix = $temporaryRoot.TrimEnd('\') + '\'
    if (-not $candidate.StartsWith($prefix, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "refusing to remove validation directory outside $temporaryRoot"
    }
    Remove-Item -LiteralPath $candidate -Recurse -Force
}
