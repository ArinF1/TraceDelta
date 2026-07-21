[CmdletBinding()]
param(
    [string]$OutputDirectory = (Join-Path $PSScriptRoot '..\dist\demo'),
    [int]$TimeoutSeconds = 90
)

$ErrorActionPreference = 'Stop'

if ($TimeoutSeconds -lt 60 -or $TimeoutSeconds -gt 180) {
    throw 'TimeoutSeconds must be between 60 and 180.'
}

$repositoryRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$sourcePath = Join-Path $repositoryRoot 'docs\demo\recording.html'
$edgePath = 'C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe'
if (-not (Test-Path -LiteralPath $edgePath -PathType Leaf)) {
    $edgePath = 'C:\Program Files\Microsoft\Edge\Application\msedge.exe'
}
if (-not (Test-Path -LiteralPath $edgePath -PathType Leaf)) {
    throw 'Microsoft Edge is required to render the demonstration.'
}

$resolvedOutputDirectory = [System.IO.Path]::GetFullPath($OutputDirectory)
if (-not (Test-Path -LiteralPath $resolvedOutputDirectory)) {
    New-Item -ItemType Directory -Path $resolvedOutputDirectory | Out-Null
}
$outputPath = Join-Path $resolvedOutputDirectory 'tracedelta-v0.1-demo.webm'
if (Test-Path -LiteralPath $outputPath) {
    throw "Refusing to overwrite existing demonstration: $outputPath"
}
$fallbackDownloadPath = Join-Path ([Environment]::GetFolderPath('UserProfile')) 'Downloads\tracedelta-v0.1-demo.webm'
if (Test-Path -LiteralPath $fallbackDownloadPath) {
    throw "Refusing to overwrite Edge's existing fallback download: $fallbackDownloadPath"
}

$profileRoot = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath())
$profilePath = Join-Path $profileRoot ("tracedelta-demo-{0}-{1}" -f $PID, [guid]::NewGuid().ToString('N'))
$verificationOutputPath = Join-Path $profileRoot ("tracedelta-demo-verify-{0}-{1}.stdout" -f $PID, [guid]::NewGuid().ToString('N'))
$verificationErrorPath = Join-Path $profileRoot ("tracedelta-demo-verify-{0}-{1}.stderr" -f $PID, [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $profilePath | Out-Null

$sourceUri = ([uri]$sourcePath).AbsoluteUri
$edgeArguments = @(
    '--headless=new',
    '--disable-gpu',
    '--disable-background-timer-throttling',
    '--disable-renderer-backgrounding',
    '--mute-audio',
    '--no-first-run',
    '--no-default-browser-check',
    '--allow-file-access-from-files',
    "--user-data-dir=`"$profilePath`"",
    "--download-default-directory=`"$resolvedOutputDirectory`"",
    $sourceUri
)

$edgeProcess = $null
try {
    $edgeProcess = Start-Process -FilePath $edgePath -ArgumentList $edgeArguments -WindowStyle Hidden -PassThru
    $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    $lastLength = -1
    $stableChecks = 0
    while ([DateTime]::UtcNow -lt $deadline) {
        Start-Sleep -Milliseconds 500
        $completedPath = if (Test-Path -LiteralPath $outputPath -PathType Leaf) {
            $outputPath
        } elseif (Test-Path -LiteralPath $fallbackDownloadPath -PathType Leaf) {
            $fallbackDownloadPath
        } else {
            $null
        }
        if ($completedPath) {
            $length = (Get-Item -LiteralPath $completedPath).Length
            if ($length -gt 0 -and $length -eq $lastLength) {
                $stableChecks++
                if ($stableChecks -ge 4) { break }
            } else {
                $stableChecks = 0
                $lastLength = $length
            }
        }
        if ($edgeProcess.HasExited) {
            throw "Edge exited before the demonstration download completed (exit $($edgeProcess.ExitCode))."
        }
    }
    if (-not (Test-Path -LiteralPath $outputPath -PathType Leaf) -and
        (Test-Path -LiteralPath $fallbackDownloadPath -PathType Leaf)) {
        Move-Item -LiteralPath $fallbackDownloadPath -Destination $outputPath
    }
    if (-not (Test-Path -LiteralPath $outputPath -PathType Leaf)) {
        throw "Demonstration recording did not complete within $TimeoutSeconds seconds."
    }

    if ($edgeProcess -and -not $edgeProcess.HasExited) {
        Stop-Process -Id $edgeProcess.Id -Force -ErrorAction SilentlyContinue
        $edgeProcess.WaitForExit(5000) | Out-Null
    }

    $videoUri = ([uri]$outputPath).AbsoluteUri
    $verifyUri = "${sourceUri}?verify=$([uri]::EscapeDataString($videoUri))"
    $verificationArguments = @(
        '--headless=new',
        '--disable-gpu',
        '--allow-file-access-from-files',
        "--user-data-dir=`"$profilePath`"",
        '--virtual-time-budget=5000',
        '--dump-dom',
        $verifyUri
    )
    $verificationProcess = Start-Process -FilePath $edgePath -ArgumentList $verificationArguments -WindowStyle Hidden -RedirectStandardOutput $verificationOutputPath -RedirectStandardError $verificationErrorPath -PassThru
    if (-not $verificationProcess.WaitForExit(15000)) {
        Stop-Process -Id $verificationProcess.Id -Force -ErrorAction SilentlyContinue
        throw 'Timed out while reading the demonstration duration metadata.'
    }
    $verificationText = Get-Content -Raw -LiteralPath $verificationOutputPath
    if ($verificationText -notmatch 'id="duration" data-seconds="([0-9.]+)"') {
        throw 'Could not read the demonstration duration from the rendered WebM metadata.'
    }
    $duration = [double]::Parse($matches[1], [Globalization.CultureInfo]::InvariantCulture)
    if ($duration -lt 45 -or $duration -gt 60) {
        throw "Demonstration duration $duration seconds is outside the required 45-60 second window."
    }

    $hash = Get-FileHash -Algorithm SHA256 -LiteralPath $outputPath
    [pscustomobject]@{
        Path = $outputPath
        DurationSeconds = [math]::Round($duration, 3)
        Bytes = (Get-Item -LiteralPath $outputPath).Length
        SHA256 = $hash.Hash.ToLowerInvariant()
    }
} finally {
    if ($edgeProcess -and -not $edgeProcess.HasExited) {
        Stop-Process -Id $edgeProcess.Id -Force -ErrorAction SilentlyContinue
    }
    foreach ($verificationPath in @($verificationOutputPath, $verificationErrorPath)) {
        if (Test-Path -LiteralPath $verificationPath -PathType Leaf) {
            Remove-Item -LiteralPath $verificationPath -Force -ErrorAction SilentlyContinue
        }
    }
    $resolvedProfilePath = [System.IO.Path]::GetFullPath($profilePath)
    if ($resolvedProfilePath.StartsWith($profileRoot, [System.StringComparison]::OrdinalIgnoreCase) -and
        (Split-Path -Leaf $resolvedProfilePath).StartsWith('tracedelta-demo-', [System.StringComparison]::Ordinal)) {
        for ($cleanupAttempt = 0; $cleanupAttempt -lt 10 -and (Test-Path -LiteralPath $resolvedProfilePath); $cleanupAttempt++) {
            try {
                Remove-Item -LiteralPath $resolvedProfilePath -Recurse -Force -ErrorAction Stop
            } catch {
                Start-Sleep -Milliseconds 200
            }
        }
    }
}
