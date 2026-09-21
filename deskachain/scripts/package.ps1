param(
    [string]$Version = "v0.4.10-testnet-rc1",
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$Dist = Join-Path $Root "dist"
$ReleaseDir = Join-Path $Dist "releases"

& (Join-Path $PSScriptRoot "build.ps1") -Version $Version -SkipTests:$SkipTests

if (Test-Path $ReleaseDir) {
    Remove-Item -Recurse -Force $ReleaseDir
}
New-Item -ItemType Directory -Force -Path $ReleaseDir | Out-Null

$Forbidden = @("wallets.json", "chain.db", "mempool.json", "peers.json", "node.lock", "node_id", "faucet_state.json", "indoservice-state.json", "indoservice-states.json", "*.key", "*.pem")
$ForbiddenDirs = @("testdata", "data", "wallets", ".git", "dist")
$Targets = @("windows-amd64", "linux-amd64", "linux-arm64")

foreach ($Target in $Targets) {
    $SourceDir = Join-Path $Dist $Target
    if (-not (Test-Path $SourceDir)) {
        throw "missing build output: $SourceDir"
    }
    $Stage = Join-Path $ReleaseDir "stage-$Target"
    if (Test-Path $Stage) {
        Remove-Item -Recurse -Force $Stage
    }
    New-Item -ItemType Directory -Force -Path $Stage | Out-Null
    New-Item -ItemType Directory -Force -Path (Join-Path $Stage "docs") | Out-Null
    New-Item -ItemType Directory -Force -Path (Join-Path $Stage "config") | Out-Null
    New-Item -ItemType Directory -Force -Path (Join-Path $Stage "scripts") | Out-Null
    Copy-Item -Path (Join-Path $SourceDir "*") -Destination $Stage -Recurse
    Copy-Item -Path (Join-Path $Root "docs\Release.md") -Destination (Join-Path $Stage "QUICKSTART.md")
    Copy-Item -Path (Join-Path $Root "docs\Release.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\Announcement-v0.4.6-testnet-rc1.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\DeployTestnet.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\DifficultyObservation.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\Explorer.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\Faucet.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\FeedbackSummaryTemplate.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\GitHubLabels.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\GitHubReleaseChecklist.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\GitHubRelease-v0.4.6-testnet-rc1.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\IssueTriage.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\KnownIssues.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\LongRunTestnet.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\Mining.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\MiningStability.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\MultiHostTestnet.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\OperatorChecklist.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\PostReleaseMonitoring.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\Preflight.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\PublicTestnetQuickstart.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\RC2Planning.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\ReleaseNotes-v0.4.6-testnet-rc1.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\SeedOperatorPublish.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\SeedMonitoringChecklist.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\ServiceNode.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\Staking.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\Systemd.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\TesterOnboarding.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\TesterResponseSnippets.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\TestnetFeedbackChecklist.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\TestnetGenesis.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "docs\TestnetTopology.md") -Destination (Join-Path $Stage "docs")
    Copy-Item -Path (Join-Path $Root "scripts\testnet-health.ps1") -Destination (Join-Path $Stage "scripts")
    Copy-Item -Path (Join-Path $Root "scripts\testnet-health.sh") -Destination (Join-Path $Stage "scripts")
    Copy-Item -Path (Join-Path $Root "config\testnet-seeds.example.txt") -Destination (Join-Path $Stage "config")
    Copy-Item -Path (Join-Path $Root "README.md") -Destination $Stage
    if (Test-Path (Join-Path $Root "README-ID.md")) {
        Copy-Item -Path (Join-Path $Root "README-ID.md") -Destination $Stage
    }
    if (Test-Path (Join-Path $Root "LICENSE")) {
        Copy-Item -Path (Join-Path $Root "LICENSE") -Destination $Stage
    }
    New-Item -ItemType Directory -Force -Path (Join-Path $Stage "examples") | Out-Null
    Copy-Item -Path (Join-Path $Root "examples\systemd") -Destination (Join-Path $Stage "examples") -Recurse
    Copy-Item -Path (Join-Path $Root "examples\testnet") -Destination (Join-Path $Stage "examples") -Recurse

    foreach ($Name in $Forbidden) {
        if (Get-ChildItem -Path $Stage -Recurse -Force -Filter $Name -ErrorAction SilentlyContinue) {
            throw "refusing to package runtime/private file: $Name"
        }
    }
    foreach ($Name in $ForbiddenDirs) {
        if (Get-ChildItem -Path $Stage -Recurse -Force -Directory -Filter $Name -ErrorAction SilentlyContinue) {
            throw "refusing to package runtime/private directory: $Name"
        }
    }

    if ($Target -eq "windows-amd64") {
        $Archive = Join-Path $ReleaseDir "indochain-$Version-$Target.zip"
        Compress-Archive -Path (Join-Path $Stage "*") -DestinationPath $Archive -Force
    } else {
        $Archive = Join-Path $ReleaseDir "indochain-$Version-$Target.tar.gz"
        Push-Location $Stage
        tar -czf $Archive .
        Pop-Location
    }
    if ($Target -eq "windows-amd64") {
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        $Zip = [System.IO.Compression.ZipFile]::OpenRead($Archive)
        try {
            $ArchiveEntries = @($Zip.Entries | ForEach-Object { $_.FullName })
        } finally {
            $Zip.Dispose()
        }
    } else {
        $ArchiveEntries = @(tar -tzf $Archive)
    }
    foreach ($Entry in $ArchiveEntries) {
        $Normalized = $Entry -replace "\\", "/"
        if ($Normalized -match "(^|/)(testdata|data|wallets|\.git|dist)(/|$)" -or
            $Normalized -match "(^|/)(wallets\.json|chain\.db|mempool\.json|peers\.json|node\.lock|node_id|faucet_state\.json|indoservice-state\.json|indoservice-states\.json)$" -or
            ($Normalized -match "\.env$" -and $Normalized -notmatch "^(\./)?examples/(systemd|testnet)/") -or
            $Normalized -match "\.(key|pem)$") {
            throw "refusing archive with runtime/private contents: $Entry"
        }
    }
    Write-Host "contents $Archive"
    $ArchiveEntries | Select-Object -First 40 | ForEach-Object { Write-Host $_ }
    Remove-Item -Recurse -Force $Stage
    Write-Host "packaged $Archive"
}

$ChecksumFile = Join-Path $ReleaseDir "SHA256SUMS.txt"
if (Test-Path $ChecksumFile) {
    Remove-Item $ChecksumFile
}
Get-ChildItem -Path $ReleaseDir -File | Where-Object { $_.Name -ne "SHA256SUMS.txt" } | ForEach-Object {
    $Hash = Get-FileHash -Algorithm SHA256 $_.FullName
    "$($Hash.Hash.ToLower())  $($_.Name)" | Add-Content -Path $ChecksumFile
}
Write-Host "checksums $ChecksumFile"
