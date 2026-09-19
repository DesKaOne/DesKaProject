# DesKaChain Public Testnet RC1 - v0.4.6-testnet-rc1

## Status

Public testnet release candidate for limited testing.

## Important Warnings

- Testnet DKC has no monetary value.
- Mainnet is not available.
- Do not treat testnet DKC as investment.
- There is no mining income or profit promise.
- Staking is collateral-only.
- Service points are simulation-only and not spendable DKC.
- Testnet may reset.
- Back up wallet files.
- Do not expose wallet/admin RPC publicly.

## What Is Included

- `deskachain` node and CLI.
- `dkcminer` CPU miner.
- `dkcservice` service-node simulation agent.
- Public testnet profile.
- P2P seed peer sync.
- Public read-only RPC mode.
- Explorer API.
- Explorer Web UI.
- Faucet operator mode.
- Staking collateral.
- Service-node simulation.
- Windows/Linux release artifacts.
- SHA256 checksums.

## Downloads

- `deskachain-v0.4.6-testnet-rc1-windows-amd64.zip`
- `deskachain-v0.4.6-testnet-rc1-linux-amd64.tar.gz`
- `deskachain-v0.4.6-testnet-rc1-linux-arm64.tar.gz`
- `SHA256SUMS.txt`

## Verify Checksums

Windows PowerShell:

```powershell
Get-FileHash .\deskachain-v0.4.6-testnet-rc1-windows-amd64.zip -Algorithm SHA256
Get-Content .\SHA256SUMS.txt
```

Linux:

```sh
sha256sum -c SHA256SUMS.txt
```

Checksums verify download integrity. They do not replace source review. Download only from the official repository release or workflow artifacts.

## Quick Start

See:

- `docs/PublicTestnetQuickstart.md`
- `docs/OperatorChecklist.md`
- `docs/Explorer.md`

## Seed Peers

Placeholder:

```text
http://<SEED_HOST>:10311
```

Operator-published seeds will be announced separately. Seed peers are not trusted authorities. Nodes validate network ID, chain ID, genesis hash, and protocol compatibility.

## Known Limitations

- No mainnet.
- No persistent explorer database yet.
- Explorer currently uses simple scan mode.
- No wallet web UI.
- No Stratum or pool mining.
- No mobile wallet app yet.
- Faucet must be manually operated and funded.
- Service-node scoring is simulation-only.
- Staking is collateral-only.

## Feedback Requested

Please report:

- OS and architecture.
- Binary version.
- Node mode: LAN, Tailscale, VPS, or local only.
- Whether the node synced.
- Explorer UI issues.
- Mining issues.
- Faucet issues.
- Service-node issues.
- Logs and commands used.

Do not paste private keys, wallet files, or secrets.
