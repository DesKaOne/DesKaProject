param(
    [string]$Version = "v0.4.10-testnet-rc1",
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$Dist = Join-Path $Root "dist"
$Commit = "unknown"
try {
    $Commit = (git -C $Root rev-parse --short HEAD).Trim()
} catch {}
$BuildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-dd'T'HH':'mm':'ss'Z'")
$LdFlags = "-s -w -X indochain/internal/version.Version=$Version -X indochain/internal/version.Commit=$Commit -X indochain/internal/version.BuildDate=$BuildDate"

if (-not $SkipTests) {
    Push-Location $Root
    go test ./node/...
    Pop-Location
}

if (Test-Path $Dist) {
    try {
        Remove-Item -Recurse -Force $Dist
    } catch {
        Write-Warning "could not remove dist root, will clean target output directories in place: $($_.Exception.Message)"
    }
}
New-Item -ItemType Directory -Force -Path $Dist | Out-Null

$Targets = @(
    @{ GOOS = "windows"; GOARCH = "amd64"; Dir = "windows-amd64"; Ext = ".exe" },
    @{ GOOS = "linux"; GOARCH = "amd64"; Dir = "linux-amd64"; Ext = "" },
    @{ GOOS = "linux"; GOARCH = "arm64"; Dir = "linux-arm64"; Ext = "" }
)

$Apps = @(
    @{ Name = "indochain"; Path = "./node/cmd/indochain" },
    @{ Name = "indominer"; Path = "./node/cmd/indominer" },
    @{ Name = "indoservice"; Path = "./node/cmd/indoservice" }
)

$OldGOOS = $env:GOOS
$OldGOARCH = $env:GOARCH
try {
    Push-Location $Root
    foreach ($Target in $Targets) {
        $env:GOOS = $Target.GOOS
        $env:GOARCH = $Target.GOARCH
        $OutDir = Join-Path $Dist $Target.Dir
        if (Test-Path $OutDir) {
            Get-ChildItem -LiteralPath $OutDir -Force | Remove-Item -Recurse -Force
        }
        New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
        foreach ($App in $Apps) {
            $Out = Join-Path $OutDir ($App.Name + $Target.Ext)
            go build -trimpath -ldflags $LdFlags -o $Out $App.Path
            Write-Host "built $Out"
        }
    }
} finally {
    Pop-Location
    $env:GOOS = $OldGOOS
    $env:GOARCH = $OldGOARCH
}

Write-Host "release build complete: $Dist"
