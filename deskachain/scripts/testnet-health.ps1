param(
    [string]$RpcUrl,
    [string]$ExpectedNetwork = "",
    [string]$ExpectedNetworkID = "",
    [UInt64]$ExpectedChainID = 0,
    [string]$BinDir = "",
    [string]$Datadir = "",
    [switch]$AllowWalletRPC,
    [switch]$AllowAdminRPC,
    [int]$MinPeers = -1,
    [UInt64]$ExpectedMinHeight = 0,
    [UInt64]$ExpectedMaxHeightLag = 0,
    [switch]$CheckPeerList,
    [string[]]$CheckSeed = @(),
    [switch]$FailOnZeroPeers,
    [switch]$CheckMining,
    [switch]$CheckDifficulty,
    [int]$WarnIfNoRecentBlockMinutes = 0,
    [int]$FailIfNoRecentBlockMinutes = 0,
    [switch]$Json,
    [int]$TimeoutSeconds = 10,
    [switch]$Help
)

if ($Help) {
    Write-Host "Usage: powershell -ExecutionPolicy Bypass -File .\scripts\testnet-health.ps1 -RpcUrl http://127.0.0.1:9311 [-ExpectedNetwork testnet] [-ExpectedNetworkID dkc-testnet-1] [-ExpectedChainID 777101] [-MinPeers 1] [-ExpectedMinHeight 100] [-ExpectedMaxHeightLag 10] [-CheckPeerList] [-CheckSeed http://host:10311] [-FailOnZeroPeers] [-CheckMining] [-CheckDifficulty] [-WarnIfNoRecentBlockMinutes 10] [-FailIfNoRecentBlockMinutes 30] [-AllowWalletRPC] [-AllowAdminRPC] [-Json] [-BinDir .\dist\windows-amd64] [-Datadir .\data\testnet] [-TimeoutSeconds 10]"
    exit 0
}

$ErrorActionPreference = "Stop"
if (-not $RpcUrl) {
    throw "RpcUrl is required. Use -Help for usage."
}
$warnings = New-Object System.Collections.Generic.List[string]
$errors = New-Object System.Collections.Generic.List[string]
$checks = [ordered]@{}

function Add-Warning([string]$Message) {
    $warnings.Add($Message) | Out-Null
}

function Add-Error([string]$Message) {
    $errors.Add($Message) | Out-Null
}

function Get-Endpoint([string]$Path) {
    $base = $RpcUrl.TrimEnd("/")
    Invoke-RestMethod -Uri "$base$Path" -TimeoutSec $TimeoutSeconds
}

function Get-WebEndpoint([string]$Path) {
    $base = $RpcUrl.TrimEnd("/")
    Invoke-WebRequest -Uri "$base$Path" -TimeoutSec $TimeoutSeconds -UseBasicParsing
}

$health = $null
$status = $null
$blocks = $null
$ui = $null
$peerHealth = $null
$peerList = $null
$miningStatus = $null

try {
    $health = Get-Endpoint "/health"
    $checks.health = $true
} catch {
    $checks.health = $false
    Add-Error "health unreachable: $($_.Exception.Message)"
}

try {
    $status = Get-Endpoint "/explorer/status"
    $checks.explorer_status = [bool]$status.ok
    if (-not $status.ok) {
        Add-Error "explorer status returned ok=false"
    }
} catch {
    $checks.explorer_status = $false
    Add-Error "explorer status unreachable: $($_.Exception.Message)"
}

try {
    $blocks = Get-Endpoint "/explorer/blocks?limit=1"
    $checks.explorer_blocks = $true
} catch {
    $checks.explorer_blocks = $false
    Add-Warning "explorer blocks check failed: $($_.Exception.Message)"
}

try {
    $ui = Get-WebEndpoint "/explorer-ui/"
    $checks.explorer_ui = ($ui.StatusCode -ge 200 -and $ui.StatusCode -lt 300 -and $ui.Content.Contains("DesKaChain Explorer"))
    if (-not $checks.explorer_ui) {
        Add-Warning "explorer UI did not return expected content"
    }
} catch {
    $checks.explorer_ui = $false
    Add-Warning "explorer UI check failed: $($_.Exception.Message)"
}

$source = if ($status) { $status } else { $health }
if ($source) {
    if ($ExpectedNetwork -and "$($source.network)" -ne $ExpectedNetwork) {
        Add-Error "network mismatch: expected $ExpectedNetwork got $($source.network)"
    }
    if ($ExpectedNetworkID -and "$($source.network_id)" -ne $ExpectedNetworkID) {
        Add-Error "network_id mismatch: expected $ExpectedNetworkID got $($source.network_id)"
    }
    if ($ExpectedChainID -ne 0 -and [UInt64]$source.chain_id -ne $ExpectedChainID) {
        Add-Error "chain_id mismatch: expected $ExpectedChainID got $($source.chain_id)"
    }
    if (-not $AllowWalletRPC -and [bool]$source.wallet_rpc) {
        Add-Error "wallet_rpc is true on a public safety check"
    }
    if (-not $AllowAdminRPC -and [bool]$source.admin_rpc) {
        Add-Error "admin_rpc is true on a public safety check"
    }
    if ($source.public_rpc -ne $true) {
        Add-Warning "public_rpc is not true; this may be expected for private/local checks"
    }
    if ($ExpectedMinHeight -gt 0 -and [UInt64]$source.height -lt $ExpectedMinHeight) {
        Add-Error "height below expected minimum: expected >= $ExpectedMinHeight got $($source.height)"
    }
    if ($ExpectedMaxHeightLag -gt 0 -and $blocks -and ($blocks.PSObject.Properties.Name -contains "blocks") -and $blocks.blocks.Count -gt 0) {
        $observed = [UInt64]$source.height
        $latestListed = [UInt64]$blocks.blocks[0].height
        if ($latestListed -gt $observed + $ExpectedMaxHeightLag) {
            Add-Error "height lag exceeds maximum: expected lag <= $ExpectedMaxHeightLag got $($latestListed - $observed)"
        }
    }
}

if ($CheckPeerList -or $MinPeers -ge 0 -or $FailOnZeroPeers -or $CheckSeed.Count -gt 0) {
    try {
        $peerHealth = Get-Endpoint "/peer/health"
        $checks.peer_health = $true
    } catch {
        $checks.peer_health = $false
        Add-Error "peer health check failed: $($_.Exception.Message)"
    }
    try {
        $peerList = Get-Endpoint "/peer/list"
        $checks.peer_list = $true
    } catch {
        $checks.peer_list = $false
        if ($CheckPeerList) {
            Add-Error "peer list check failed: $($_.Exception.Message)"
        } else {
            Add-Warning "peer list check failed: $($_.Exception.Message)"
        }
    }
}

$knownPeerCount = $null
$activePeerCount = $null
if ($peerHealth) {
    $knownPeerCount = $peerHealth.known_peer_count
    $activePeerCount = $peerHealth.active_peer_count
} elseif ($source) {
    if ($source.PSObject.Properties.Name -contains "known_peer_count") {
        $knownPeerCount = $source.known_peer_count
    }
    $activePeerCount = if ($status) { $status.peer_count } elseif ($health) { $health.peers } else { $null }
}
if ($null -eq $knownPeerCount) {
    $knownPeerCount = 0
}
if ($null -eq $activePeerCount) {
    $activePeerCount = if ($status) { $status.peer_count } elseif ($health) { $health.peers } else { 0 }
}
if ($FailOnZeroPeers -and [int]$activePeerCount -eq 0) {
    Add-Error "active peer count is zero"
} elseif ([int]$activePeerCount -eq 0) {
    Add-Warning "active peer count is zero"
}
if ($MinPeers -ge 0 -and [int]$activePeerCount -lt $MinPeers) {
    Add-Error "active peer count below minimum: expected >= $MinPeers got $activePeerCount"
}

foreach ($seed in $CheckSeed) {
    $found = $false
    if ($peerList -and ($peerList.PSObject.Properties.Name -contains "peers")) {
        foreach ($peer in $peerList.peers) {
            if ("$($peer.url)".TrimEnd("/") -eq "$seed".TrimEnd("/")) {
                $found = $true
                break
            }
        }
    }
    if (-not $found) {
        Add-Warning "seed not found in peer list: $seed"
    }
}

if ($CheckMining -or $CheckDifficulty -or $WarnIfNoRecentBlockMinutes -gt 0 -or $FailIfNoRecentBlockMinutes -gt 0) {
    try {
        $miningStatus = Get-Endpoint "/mining/status"
        $checks.mining_status = [bool]$miningStatus.ok
        if (-not $checks.mining_status) {
            Add-Warning "mining status returned ok=false"
        }
    } catch {
        $checks.mining_status = $false
        Add-Error "mining status check failed: $($_.Exception.Message)"
    }
    if ($CheckDifficulty) {
        try {
            $null = Get-Endpoint "/mining/difficulty"
            $checks.mining_difficulty = $true
        } catch {
            $checks.mining_difficulty = $false
            Add-Error "mining difficulty check failed: $($_.Exception.Message)"
        }
    }
}

$lastBlockAgeSeconds = $null
if ($miningStatus) {
    $lastBlockAgeSeconds = [int64]$miningStatus.last_block_age_seconds
    if ($WarnIfNoRecentBlockMinutes -gt 0 -and $lastBlockAgeSeconds -gt ($WarnIfNoRecentBlockMinutes * 60)) {
        Add-Warning "no recent block within warning window: age ${lastBlockAgeSeconds}s"
    }
    if ($FailIfNoRecentBlockMinutes -gt 0 -and $lastBlockAgeSeconds -gt ($FailIfNoRecentBlockMinutes * 60)) {
        Add-Error "no recent block within failure window: age ${lastBlockAgeSeconds}s"
    }
}

$cliInfo = ""
$cliValidate = ""
if ($BinDir) {
    $resolved = Resolve-Path $BinDir -ErrorAction Stop
    $deskachain = Join-Path $resolved.Path "deskachain.exe"
    if (-not (Test-Path $deskachain)) {
        $deskachain = Join-Path $resolved.Path "deskachain"
    }
    if (-not (Test-Path $deskachain)) {
        Add-Warning "deskachain binary not found in $($resolved.Path)"
    } else {
        try {
            $cliInfo = (& $deskachain --rpc-url $RpcUrl chain info 2>&1 | Out-String).Trim()
            $checks.cli_chain_info = $LASTEXITCODE -eq 0
            if ($LASTEXITCODE -ne 0) {
                Add-Warning "chain info via CLI failed"
            }
        } catch {
            $checks.cli_chain_info = $false
            Add-Warning "chain info via CLI failed: $($_.Exception.Message)"
        }
        if ($Datadir) {
            try {
                $cliValidate = (& $deskachain --rpc-url $RpcUrl chain validate 2>&1 | Out-String).Trim()
                $checks.cli_chain_validate = $LASTEXITCODE -eq 0
                if ($LASTEXITCODE -ne 0) {
                    Add-Error "chain validate via CLI failed"
                }
            } catch {
                $checks.cli_chain_validate = $false
                Add-Error "chain validate via CLI failed: $($_.Exception.Message)"
            }
        }
    }
}

$result = [ordered]@{
    ok = ($errors.Count -eq 0)
    rpc_url = $RpcUrl
    network = if ($source) { $source.network } else { $null }
    network_id = if ($source) { $source.network_id } else { $null }
    chain_id = if ($source) { $source.chain_id } else { $null }
    height = if ($source) { $source.height } else { $null }
    tip_hash = if ($source) { $source.tip_hash } else { $null }
    peer_count = if ($status) { $status.peer_count } elseif ($health) { $health.peers } else { $null }
    known_peer_count = $knownPeerCount
    active_peer_count = $activePeerCount
    pending_tx_count = if ($status) { $status.pending_tx_count } elseif ($health) { $health.mempool } else { $null }
    public_rpc = if ($source) { $source.public_rpc } else { $null }
    wallet_rpc = if ($source) { $source.wallet_rpc } else { $null }
    admin_rpc = if ($source) { $source.admin_rpc } else { $null }
    explorer_ok = ($checks.explorer_status -eq $true -and $checks.explorer_blocks -eq $true -and $checks.explorer_ui -eq $true)
    mining = if ($miningStatus) { [ordered]@{
        current_difficulty = $miningStatus.current_difficulty
        next_difficulty = $miningStatus.next_difficulty
        target_block_time_seconds = $miningStatus.target_block_time_seconds
        blocks_until_retarget = $miningStatus.blocks_until_retarget
        last_block_age_seconds = $miningStatus.last_block_age_seconds
        average_interval_seconds = $miningStatus.average_interval_seconds
        projected_retarget_direction = $miningStatus.projected_retarget_direction
    }} else { $null }
    checks = $checks
    warnings = @($warnings)
    errors = @($errors)
}

if ($cliInfo) {
    $result.cli_chain_info = $cliInfo
}
if ($cliValidate) {
    $result.cli_chain_validate = $cliValidate
}

if ($Json) {
    $result | ConvertTo-Json -Depth 6
} else {
    if ($result.ok) {
        Write-Host "ok: testnet health check passed"
    } else {
        Write-Host "fail: testnet health check failed"
    }
    Write-Host "rpc_url: $($result.rpc_url)"
    Write-Host "network: $($result.network)"
    Write-Host "network_id: $($result.network_id)"
    Write-Host "chain_id: $($result.chain_id)"
    Write-Host "height: $($result.height)"
    Write-Host "tip_hash: $($result.tip_hash)"
    Write-Host "peer_count: $($result.peer_count)"
    Write-Host "known_peer_count: $($result.known_peer_count)"
    Write-Host "active_peer_count: $($result.active_peer_count)"
    Write-Host "pending_tx_count: $($result.pending_tx_count)"
    Write-Host "public_rpc: $($result.public_rpc)"
    Write-Host "wallet_rpc: $($result.wallet_rpc)"
    Write-Host "admin_rpc: $($result.admin_rpc)"
    Write-Host "explorer_ok: $($result.explorer_ok)"
    if ($result.mining) {
        Write-Host "mining_current_difficulty: $($result.mining.current_difficulty)"
        Write-Host "mining_next_difficulty: $($result.mining.next_difficulty)"
        Write-Host "mining_target_block_time: $($result.mining.target_block_time_seconds)s"
        Write-Host "mining_blocks_until_retarget: $($result.mining.blocks_until_retarget)"
        Write-Host "mining_last_block_age: $($result.mining.last_block_age_seconds)s"
        Write-Host "mining_average_interval: $($result.mining.average_interval_seconds)s"
        Write-Host "mining_retarget_direction: $($result.mining.projected_retarget_direction)"
    }
    foreach ($w in $warnings) {
        Write-Host "warning: $w"
    }
    foreach ($e in $errors) {
        Write-Host "error: $e"
    }
}

if ($errors.Count -gt 0) {
    exit 1
}
