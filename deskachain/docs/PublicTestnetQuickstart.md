# DesKaChain Public Testnet Quickstart

This guide is for external public-testnet testers and operators. Testnet DKC has no monetary value. Mainnet is not available. Do not expose wallet/admin RPC publicly.

## 1. Download

Download the archive for your platform from the official repository workflow or release artifacts.

Supported RC platforms:

- `windows-amd64`
- `linux-amd64`
- `linux-arm64`

## 2. Verify SHA256

Linux:

```sh
cd dist/releases
sha256sum -c SHA256SUMS.txt
```

Windows PowerShell:

```powershell
Get-FileHash .\deskachain-v0.4.6-testnet-rc1-windows-amd64.zip -Algorithm SHA256
Get-Content .\SHA256SUMS.txt
```

Checksums verify download integrity. They do not replace source review. Do not run random binaries from unknown sources.

## 3. Run Version

Linux:

```sh
mkdir deskachain-rc
cd deskachain-rc
tar -xzf deskachain-v0.4.6-testnet-rc1-linux-amd64.tar.gz
./deskachain version
```

Windows PowerShell:

```powershell
.\deskachain.exe version
```

Expected output includes version, commit, build time, Go version, OS/ARCH, networks `localnet,testnet`, and `mainnet: not available`.

## 4. Init Testnet

Linux:

```sh
./deskachain --datadir ./data/testnet --network testnet init
```

Windows PowerShell:

```powershell
.\deskachain.exe --datadir .\data\testnet --network testnet init
```

## 5. Start A Peer Node

Replace seed and advertised P2P placeholders with operator-published values.

Linux:

```sh
./deskachain --datadir ./data/testnet node start \
  --rpc 127.0.0.1:9311 \
  --p2p 0.0.0.0:10311 \
  --advertise-p2p http://<YOUR_REACHABLE_IP>:10311 \
  --public-rpc \
  --seed-peer http://100.86.152.39:10311 \
  --seed-peer http://100.101.251.7:10311
```

Windows PowerShell:

```powershell
.\deskachain.exe --datadir .\data\testnet node start `
  --rpc 127.0.0.1:9311 `
  --p2p 0.0.0.0:10311 `
  --advertise-p2p http://<YOUR_REACHABLE_IP>:10311 `
  --public-rpc `
  --seed-peer http://100.86.152.39:10311 `
  --seed-peer http://100.101.251.7:10311
```

Seeds are not trusted authorities. They only help a fresh node find peers; every node still validates network ID, chain ID, genesis hash, headers, and blocks. Multiple seeds improve availability when one operator is offline.

Mining nodes should also set an upstream peer so accepted blocks are pushed/backfilled to the public head/explorer node:

```sh
./deskachain --datadir ./data/testnet --network testnet node start \
  --public-rpc \
  --enable-miner-rpc \
  --seed-peer http://100.86.152.39:10311 \
  --upstream-peer http://100.86.152.39:10311 \
  --min-mining-peers 1
```

`--seed-peer` pulls and discovers. `--upstream-peer` pushes/backfills. Testnet isolated mining and faucet writes are blocked by default unless explicitly overridden for private testing.

## 6. Open Explorer UI

```text
http://127.0.0.1:9311/explorer-ui/
```

API examples:

```sh
curl http://127.0.0.1:9311/health
curl http://127.0.0.1:9311/explorer/status
curl "http://127.0.0.1:9311/explorer/blocks?limit=5"
```

Peer diagnostics:

```sh
./deskachain --rpc-url http://127.0.0.1:9311 peer list
./deskachain --rpc-url http://127.0.0.1:9311 peer health
./deskachain --rpc-url http://127.0.0.1:9311 peer discover
./deskachain --rpc-url http://127.0.0.1:9311 mining status
./deskachain --rpc-url http://127.0.0.1:9311 mining difficulty
```

## 7. After Installing RC1

Run the health check:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\testnet-health.ps1 -RpcUrl http://127.0.0.1:9311 -ExpectedNetwork testnet -ExpectedNetworkID dkc-testnet-1 -ExpectedChainID 777101 -CheckMining
```

Linux:

```sh
bash ./scripts/testnet-health.sh http://127.0.0.1:9311 --expected-network testnet --expected-network-id dkc-testnet-1 --expected-chain-id 777101 --check-peer-list --check-mining
```

Then verify the explorer, report issues using `.github/ISSUE_TEMPLATE/testnet-bug-report.md`, do not expose wallet/admin RPC, and remember testnet DKC has no monetary value.

Monitoring and feedback docs:

- `docs/PostReleaseMonitoring.md`
- `docs/SeedMonitoringChecklist.md`
- `docs/KnownIssues.md`
- `docs/IssueTriage.md`
- `docs/FeedbackSummaryTemplate.md`
- `docs/RC2Planning.md`
- `docs/TestnetTopology.md`
- `docs/Mining.md`
- `docs/MiningStability.md`
- `docs/DifficultyObservation.md`

## 8. Optional Mine Once

Use a separate miner wallet datadir.

Linux:

```sh
./deskachain --datadir ./data/miner-wallet --network testnet init
ADDR="$(./deskachain --datadir ./data/miner-wallet wallet new)"
./dkcminer --rpc-url http://127.0.0.1:9311 --address "$ADDR" --threads 2 --once
```

Windows PowerShell:

```powershell
.\deskachain.exe --datadir .\data\miner-wallet --network testnet init
$addr = .\deskachain.exe --datadir .\data\miner-wallet wallet new
.\dkcminer.exe --rpc-url http://127.0.0.1:9311 --address $addr --threads 2 --once
```

## 9. Optional Faucet

Use faucet only if an operator provides a faucet RPC URL. Faucet funds are testnet-only and require mining for confirmation.

## 10. Safety

- Testnet only.
- Testnet DKC has no monetary value.
- Mainnet is unavailable.
- No mining income, staking APY, profit, or reward promise.
- Staking is collateral-only.
- Service points are simulation-only and are not spendable.
- Do not expose wallet/admin RPC publicly.
- Back up wallets.
- Testnet may reset.
