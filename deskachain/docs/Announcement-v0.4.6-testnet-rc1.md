# Announcement Draft - DesKaChain Public Testnet RC1

DesKaChain Public Testnet RC1 (`v0.4.6-testnet-rc1`) is ready for limited technical testing.

DesKaChain is a small experimental CPU-mined blockchain written in Go. This RC includes the node/CLI, standalone CPU miner, service-node simulation agent, public testnet profile, read-only Explorer API, embedded Explorer Web UI, faucet operator mode, staking collateral, and service-node simulation.

This is testnet only. Testnet DKC has no monetary value. Mainnet is not available. There is no mining income, staking APY, profit, or reward promise. Service points are simulation-only and are not spendable DKC.

## Downloads

Download the artifact for your platform from the official GitHub Release:

- Windows amd64
- Linux amd64
- Linux arm64

Verify checksums with `SHA256SUMS.txt` before running any binary.

## Join The Testnet

Use the public quickstart:

```text
docs/PublicTestnetQuickstart.md
```

Seed peer placeholder:

```text
http://<SEED_HOST>:10311
```

Operator-published seeds will be announced separately.

## Explorer

After starting a node, open:

```text
http://127.0.0.1:9311/explorer-ui/
```

## Feedback Needed

Please report sync issues, explorer UI issues, mining issues, faucet issues, service-node simulation issues, OS/architecture, binary version, commands used, and logs.

Do not paste private keys, wallet files, or secrets.
