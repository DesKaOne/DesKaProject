param(
    [string]$BinDir = ".\dist\windows-amd64",
    [switch]$KeepData,
    [switch]$Help
)

if ($Help) {
    Write-Host "Usage: powershell -ExecutionPolicy Bypass -File .\scripts\smoke-test.ps1 [-BinDir .\dist\windows-amd64] [-KeepData]"
    Write-Host "Runs a local testnet smoke test using temporary datadirs. No secrets are required."
    exit 0
}

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$ResolvedBinDir = (Resolve-Path (Join-Path $Root $BinDir) -ErrorAction SilentlyContinue)
if (-not $ResolvedBinDir) {
    $ResolvedBinDir = Resolve-Path $BinDir
}
$BinDirPath = $ResolvedBinDir.Path
$IndoChain = Join-Path $BinDirPath "indochain.exe"
$Miner = Join-Path $BinDirPath "indominer.exe"
if (-not (Test-Path $IndoChain)) {
    $IndoChain = Join-Path $BinDirPath "indochain"
}
if (-not (Test-Path $Miner)) {
    $Miner = Join-Path $BinDirPath "indominer"
}
if (-not (Test-Path $IndoChain) -or -not (Test-Path $Miner)) {
    throw "indochain and indominer binaries are required in $BinDirPath"
}

$TempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("indochain-smoke-" + [guid]::NewGuid().ToString("N"))
$NodeDir = Join-Path $TempRoot "node"
$WalletDir = Join-Path $TempRoot "miner-wallet"
$RPC = "http://127.0.0.1:19311"
$NodeProcess = $null

try {
    New-Item -ItemType Directory -Force -Path $TempRoot | Out-Null
    & $IndoChain version
    & $IndoChain --datadir $NodeDir --network testnet init
    & $IndoChain --datadir $WalletDir --network testnet init
    $Address = (& $IndoChain --datadir $WalletDir wallet new | Select-Object -Last 1).Trim()
    if (-not $Address) {
        throw "wallet new did not return an address"
    }

    $NodeLog = Join-Path $TempRoot "node.log"
    $NodeErrLog = Join-Path $TempRoot "node.err.log"
    $NodeProcess = Start-Process -FilePath $IndoChain -ArgumentList @(
        "--datadir", $NodeDir,
        "--network", "testnet",
        "node", "start",
        "--rpc", "127.0.0.1:19311",
        "--p2p", "127.0.0.1:19312",
        "--advertise-p2p", "http://127.0.0.1:19312",
        "--public-rpc",
        "--enable-miner-rpc"
    ) -RedirectStandardOutput $NodeLog -RedirectStandardError $NodeErrLog -PassThru -WindowStyle Hidden

    $Ready = $false
    for ($i = 0; $i -lt 60; $i++) {
        try {
            $Health = Invoke-RestMethod -Uri "$RPC/health" -TimeoutSec 2
            if ($Health.ok) {
                $Ready = $true
                break
            }
        } catch {
            Start-Sleep -Seconds 1
        }
        if ($NodeProcess.HasExited) {
            Get-Content $NodeLog -ErrorAction SilentlyContinue
            Get-Content $NodeErrLog -ErrorAction SilentlyContinue
            throw "node exited before becoming healthy"
        }
    }
    if (-not $Ready) {
        Get-Content $NodeLog -ErrorAction SilentlyContinue
        Get-Content $NodeErrLog -ErrorAction SilentlyContinue
        throw "node did not become healthy"
    }

    & $IndoChain --rpc-url $RPC chain info
    & $IndoChain --rpc-url $RPC chain validate
    & $Miner --rpc-url $RPC --address $Address --threads 2 --once
    & $IndoChain --rpc-url $RPC chain info
    & $IndoChain --rpc-url $RPC chain validate

    $Status = Invoke-RestMethod -Uri "$RPC/explorer/status" -TimeoutSec 5
    $Blocks = Invoke-RestMethod -Uri "$RPC/explorer/blocks?limit=1" -TimeoutSec 5
    $UI = Invoke-WebRequest -Uri "$RPC/explorer-ui/" -TimeoutSec 5
    if ($Status.network -ne "testnet" -or $Blocks.count -lt 1 -or -not $UI.Content.Contains("IndoChain Explorer")) {
        throw "explorer smoke checks failed"
    }

    Write-Host "smoke test passed using $BinDirPath"
} finally {
    if ($NodeProcess -and -not $NodeProcess.HasExited) {
        Stop-Process -Id $NodeProcess.Id -Force
        try {
            Wait-Process -Id $NodeProcess.Id -Timeout 10 -ErrorAction SilentlyContinue
        } catch {}
    }
    if (-not $KeepData -and (Test-Path $TempRoot)) {
        for ($i = 0; $i -lt 5; $i++) {
            try {
                Remove-Item -Recurse -Force $TempRoot
                break
            } catch {
                if ($i -eq 4) {
                    throw
                }
                Start-Sleep -Milliseconds 300
            }
        }
    } elseif ($KeepData) {
        Write-Host "kept smoke data: $TempRoot"
    }
}
