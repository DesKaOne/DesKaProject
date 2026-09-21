param(
    [Parameter(Mandatory=$true)][string]$RpcUrl,
    [string]$ExpectedNetwork = "",
    [string]$ExpectedNetworkID = "",
    [UInt64]$ExpectedChainID = 0,
    [int]$MinPeers = 0,
    [switch]$RequireIndexerReady,
    [UInt64]$MaxIndexerLag = 0,
    [switch]$CheckMining,
    [switch]$Json,
    [int]$TimeoutSeconds = 10,
    [switch]$Help
)

if ($Help) {
    Write-Host "Usage: powershell -ExecutionPolicy Bypass -File .\scripts\testnet-smoke.ps1 -RpcUrl http://127.0.0.1:9311 [-ExpectedNetwork testnet] [-ExpectedNetworkID ind-testnet-1] [-ExpectedChainID 777101] [-MinPeers 1] [-RequireIndexerReady] [-MaxIndexerLag 2] [-CheckMining] [-Json]"
    exit 0
}

$ErrorActionPreference = "Stop"
$base = $RpcUrl.TrimEnd("/")
$errors = New-Object System.Collections.Generic.List[string]
$checks = [ordered]@{}

function Get-Endpoint([string]$Path) {
    Invoke-RestMethod -Uri "$base$Path" -TimeoutSec $TimeoutSeconds
}

function Get-WebEndpoint([string]$Path) {
    Invoke-WebRequest -Uri "$base$Path" -TimeoutSec $TimeoutSeconds -UseBasicParsing
}

function Add-Error([string]$Message) {
    $errors.Add($Message) | Out-Null
}

$health = $null
$metrics = $null
$peer = $null
$indexer = $null
$ui = $null
$mining = $null

try { $health = Get-Endpoint "/health"; $checks.health = $true } catch { $checks.health = $false; Add-Error "health endpoint failed: $($_.Exception.Message)" }
try { $metrics = Get-Endpoint "/node/metrics"; $checks.metrics = $true } catch { $checks.metrics = $false; Add-Error "node metrics endpoint failed: $($_.Exception.Message)" }
try { $peer = Get-Endpoint "/peer/health"; $checks.peer_health = $true } catch { $checks.peer_health = $false; Add-Error "peer health endpoint failed: $($_.Exception.Message)" }
try { $indexer = Get-Endpoint "/explorer/indexer/stats"; $checks.explorer_indexer = $true } catch { $checks.explorer_indexer = $false; Add-Error "explorer indexer stats endpoint failed: $($_.Exception.Message)" }
try { $ui = Get-WebEndpoint "/explorer-ui/"; $checks.explorer_ui = $true } catch { $checks.explorer_ui = $false; Add-Error "explorer UI endpoint failed: $($_.Exception.Message)" }
if ($CheckMining) {
    try { $mining = Get-Endpoint "/mining/status"; $checks.mining = $true } catch { $checks.mining = $false; Add-Error "mining status endpoint failed: $($_.Exception.Message)" }
}

if ($metrics) {
    if ($ExpectedNetwork -and "$($metrics.network)" -ne $ExpectedNetwork) { Add-Error "network mismatch: expected $ExpectedNetwork got $($metrics.network)" }
    if ($ExpectedNetworkID -and "$($metrics.network_id)" -ne $ExpectedNetworkID) { Add-Error "network_id mismatch: expected $ExpectedNetworkID got $($metrics.network_id)" }
    if ($ExpectedChainID -ne 0 -and [UInt64]$metrics.chain_id -ne $ExpectedChainID) { Add-Error "chain_id mismatch: expected $ExpectedChainID got $($metrics.chain_id)" }
    if ("$($metrics.schema_version)" -ne "v1") { Add-Error "node metrics schema is not v1: $($metrics.schema_version)" }
    if ([int]$metrics.peers.active -lt $MinPeers) { Add-Error "active peers below minimum: expected >= $MinPeers got $($metrics.peers.active)" }
    if ($RequireIndexerReady -and -not [bool]$metrics.indexer.stats.ready) { Add-Error "explorer indexer is not ready" }
    if ($MaxIndexerLag -gt 0 -and [UInt64]$metrics.indexer.stats.lag -gt $MaxIndexerLag) { Add-Error "explorer indexer lag exceeds maximum: expected <= $MaxIndexerLag got $($metrics.indexer.stats.lag)" }
}
if ($health -and -not [bool]$health.ok) { Add-Error "health returned ok=false" }
if ($ui -and -not $ui.Content.Contains("IndoChain Explorer")) { Add-Error "explorer UI content check failed" }
if ($CheckMining -and $mining -and -not [bool]$mining.ok) { Add-Error "mining status returned ok=false" }

try {
    $request = [System.Net.HttpWebRequest]::Create("$base/node/metrics")
    $request.Method = "POST"
    $request.Timeout = $TimeoutSeconds * 1000
    $response = $request.GetResponse()
    $postStatus = [int]$response.StatusCode
    $response.Close()
} catch {
    if ($_.Exception.Response) {
        $postStatus = [int]$_.Exception.Response.StatusCode
    } else {
        $postStatus = 0
    }
}
if ($postStatus -ne 405) { Add-Error "POST /node/metrics expected 405 got $postStatus" }

$result = [ordered]@{
    ok = ($errors.Count -eq 0)
    network = if ($metrics) { $metrics.network } else { $null }
    network_id = if ($metrics) { $metrics.network_id } else { $null }
    chain_id = if ($metrics) { $metrics.chain_id } else { $null }
    height = if ($metrics) { $metrics.chain.height } else { $null }
    active_peers = if ($metrics) { $metrics.peers.active } else { $null }
    indexer_ready = if ($metrics) { $metrics.indexer.stats.ready } else { $false }
    indexer_lag = if ($metrics) { $metrics.indexer.stats.lag } else { $null }
    metrics_schema = if ($metrics) { $metrics.schema_version } else { $null }
    checks = $checks
    errors = @($errors)
}

if ($Json) {
    $result | ConvertTo-Json -Depth 8
} else {
    if ($result.ok) { Write-Host "ok: testnet operator smoke passed" } else { Write-Host "fail: testnet operator smoke failed" }
    Write-Host "network: $($result.network)"
    Write-Host "network_id: $($result.network_id)"
    Write-Host "chain_id: $($result.chain_id)"
    Write-Host "height: $($result.height)"
    Write-Host "active_peers: $($result.active_peers)"
    Write-Host "indexer_ready: $($result.indexer_ready)"
    Write-Host "indexer_lag: $($result.indexer_lag)"
    Write-Host "metrics_schema: $($result.metrics_schema)"
    foreach ($e in $errors) { Write-Host "error: $e" }
}

if ($errors.Count -gt 0) { exit 1 }
