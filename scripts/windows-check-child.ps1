[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$StartGateName,

    [string]$ReadyGateName = '',

    [Parameter(Mandatory = $true)]
    [string]$FilePath,

    [Parameter(Mandatory = $true)]
    [string]$WorkingDirectory,

    [Parameter(Mandatory = $true)]
    [string]$ArgumentsBase64
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$startGate = $null
try {
    $startGate = [System.Threading.EventWaitHandle]::OpenExisting($StartGateName)
    if (-not [string]::IsNullOrEmpty($ReadyGateName)) {
        $readyGate = [System.Threading.EventWaitHandle]::OpenExisting($ReadyGateName)
        try {
            if (-not $readyGate.Set()) {
                throw 'could not signal readiness.'
            }
        }
        finally {
            $readyGate.Dispose()
        }
    }
    if (-not $startGate.WaitOne(30000)) {
        [Console]::Error.WriteLine('validation child start gate timed out after 30 seconds.')
        exit 124
    }
}
catch {
    [Console]::Error.WriteLine("validation child could not open its start gate: $($_.Exception.Message)")
    exit 125
}
finally {
    if ($null -ne $startGate) {
        $startGate.Dispose()
    }
}

try {
    $argumentJson = [System.Text.Encoding]::UTF8.GetString(
        [System.Convert]::FromBase64String($ArgumentsBase64)
    )
    $arguments = @($argumentJson | ConvertFrom-Json)
    Push-Location -LiteralPath $WorkingDirectory
    try {
        & $FilePath @arguments
        $exitCode = $LASTEXITCODE
    }
    finally {
        Pop-Location
    }
    if ($null -eq $exitCode) {
        $exitCode = 0
    }
    exit $exitCode
}
catch {
    [Console]::Error.WriteLine("validation child failed to invoke ${FilePath}: $($_.Exception.Message)")
    exit 126
}
