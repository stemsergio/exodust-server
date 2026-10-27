# Loads ..\Secrets\server.env into environment variables and runs the server.
$envFile = Join-Path $PSScriptRoot "..\Secrets\server.env"
if (-not (Test-Path $envFile)) {
    Write-Error "Missing $envFile"
    exit 1
}

Get-Content $envFile | Where-Object { $_ -match '=' -and $_ -notmatch '^\s*#' } | ForEach-Object {
    $k, $v = $_ -split '=', 2
    Set-Item "env:$($k.Trim())" $v.Trim()
}

Set-Location $PSScriptRoot
go run ./cmd/server
