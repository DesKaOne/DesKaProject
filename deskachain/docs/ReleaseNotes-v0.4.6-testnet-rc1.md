# IndoChain v0.4.6-testnet-rc1 Release Notes

## Summary

`v0.4.6-testnet-rc1` is a public testnet release candidate for external testers and operators. Multi-host testnet flows have been validated, including restart recovery, peer catch-up, faucet-funded staking collateral, service-node simulation, and the read-only Explorer API/Web UI.

This is not mainnet. Testnet dIDR has no monetary value.

## Included Binaries

- `indochain`
- `indominer`
- `indoservice`

## Supported Platforms

- `windows-amd64`
- `linux-amd64`
- `linux-arm64`

## Features Included

- Localnet and testnet network profiles.
- Proof-of-work mining.
- Public RPC read-only mode.
- P2P seed peer sync with network/genesis checks.
- Faucet operator mode for controlled testnet funding.
- Staking collateral for service-node eligibility.
- Service-node simulation and `indoservice` agent.
- Explorer API under `/explorer/*`.
- Explorer Web UI under `/explorer-ui/`.
- Release archives and `SHA256SUMS.txt`.

## Safety Warnings

- Testnet dIDR has no monetary value.
- Mainnet is not available.
- Do not treat testnet dIDR as an investment.
- There are no mining income, staking APY, profit, or reward promises.
- Do not expose wallet/admin RPC publicly.
- Back up wallet files before deleting or moving datadirs.
- Use separate datadirs for node, miner wallet, faucet wallet, and service agent roles.
- Testnet may reset.

## Known Limitations

- Explorer/indexer uses simple bounded scans; no persistent explorer database yet.
- Explorer UI has no write actions.
- No mainnet.
- No Stratum or mining pool protocol.
- No mobile, desktop, or web wallet app yet.
- Service points are simulation-only and are not spendable dIDR.
- Staking is collateral-only, with no APY, no validators, and no staking rewards.
- Public seed lists are operator-supplied.
- Faucet must be enabled deliberately and funded manually.

## Upgrade Notes

- Existing testnet datadirs may be reused when they use the same testnet genesis and network identity.
- Operators should verify the genesis hash with `chain info` or `/explorer/status`.
- Verify `SHA256SUMS.txt` before running downloaded artifacts.
- If switching from older localnet/testnet experiments, reset or reinitialize intentionally. Do not point the RC binary at an unknown datadir.

## Smoke Validation

After extracting an archive:

```sh
./indochain version
./indochain --datadir ./data/testnet --network testnet init
./indochain --datadir ./data/testnet node start --rpc 127.0.0.1:9311 --p2p 127.0.0.1:10311 --advertise-p2p http://127.0.0.1:10311 --public-rpc --enable-miner-rpc
```

In another shell:

```sh
./indochain --rpc-url http://127.0.0.1:9311 chain info
./indochain --rpc-url http://127.0.0.1:9311 chain validate
curl http://127.0.0.1:9311/explorer/status
```

Open:

```text
http://127.0.0.1:9311/explorer-ui/
```

Or run the bundled repository smoke scripts before packaging:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-test.ps1 -BinDir .\dist\windows-amd64
```

```sh
bash ./scripts/smoke-test.sh ./dist/linux-amd64
```
