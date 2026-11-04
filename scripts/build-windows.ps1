$ErrorActionPreference = "Stop"

Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    go build -ldflags "-H=windowsgui" -o xiangying.exe .
} finally {
    Pop-Location
}
