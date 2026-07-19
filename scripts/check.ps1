[CmdletBinding()]
param(
    [ValidateRange(1, 3600)]
    [int]$CommandTimeoutSeconds = 300,

    [ValidateRange(0, 300)]
    [int]$LockTimeoutSeconds = 5
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
. (Join-Path $PSScriptRoot 'windows-check-lib.ps1')

$validationMutex = $null
$validationDirectory = $null
try {
    $validationMutex = Enter-TraceDeltaValidationMutex `
        -RepositoryRoot $repositoryRoot `
        -LockTimeoutSeconds $LockTimeoutSeconds

    $runName = 'windows-check-' + [guid]::NewGuid().ToString('N')
    $validationDirectory = Join-Path (Join-Path $repositoryRoot 'tmp') $runName
    $null = New-Item -ItemType Directory -Path $validationDirectory -Force

    $go = (Get-Command go -CommandType Application -ErrorAction Stop).Source
    $gofmt = (Get-Command gofmt -CommandType Application -ErrorAction Stop).Source
    $git = (Get-Command git -CommandType Application -ErrorAction Stop).Source

    Write-Output 'Checking gofmt...'
    $goFiles = @(& $git -c "safe.directory=$repositoryRoot" -C $repositoryRoot ls-files --cached --others --exclude-standard -- '*.go')
    if ($LASTEXITCODE -ne 0) {
        throw "list Go files failed with exit code $LASTEXITCODE."
    }
    if ($goFiles.Count -gt 0) {
        $formatResult = Invoke-TraceDeltaBoundedProcess `
            -StepName 'gofmt verification' `
            -FilePath $gofmt `
            -ArgumentList (@('-l') + $goFiles) `
            -WorkingDirectory $repositoryRoot `
            -TimeoutSeconds $CommandTimeoutSeconds
        if (-not [string]::IsNullOrWhiteSpace($formatResult.StandardOutput)) {
            throw 'gofmt verification found files that require formatting.'
        }
    }

    Write-Output 'Running tests...'
    $null = Invoke-TraceDeltaBoundedProcess `
        -StepName 'go test' `
        -FilePath $go `
        -ArgumentList @('test', './...') `
        -WorkingDirectory $repositoryRoot `
        -TimeoutSeconds $CommandTimeoutSeconds

    Write-Output 'Running go vet...'
    $null = Invoke-TraceDeltaBoundedProcess `
        -StepName 'go vet' `
        -FilePath $go `
        -ArgumentList @('vet', './...') `
        -WorkingDirectory $repositoryRoot `
        -TimeoutSeconds $CommandTimeoutSeconds

    Write-Output 'Building CLI...'
    $binaryPath = "tmp/$runName/tracedelta.exe"
    $null = Invoke-TraceDeltaBoundedProcess `
        -StepName 'go build' `
        -FilePath $go `
        -ArgumentList @('build', '-trimpath', '-o', $binaryPath, './cmd/tracedelta') `
        -WorkingDirectory $repositoryRoot `
        -TimeoutSeconds $CommandTimeoutSeconds

    Write-Output 'All essential Windows checks passed.'
}
finally {
    try {
        if ($null -ne $validationDirectory) {
            Remove-TraceDeltaValidationDirectory `
                -RepositoryRoot $repositoryRoot `
                -Directory $validationDirectory
        }
    }
    finally {
        if ($null -ne $validationMutex) {
            try {
                $validationMutex.ReleaseMutex()
            }
            finally {
                $validationMutex.Dispose()
            }
        }
    }
}
