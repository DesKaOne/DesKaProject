# IndoChain Testnet Genesis Candidate

This document records the Phase 3.9 public-testnet genesis candidate. This is not mainnet. Testnet dIDR has no monetary value, and mainnet is not available.

## Candidate Parameters

- network: `testnet`
- network_id: `ind-testnet-1`
- chain_id: `777101`
- genesis hash: `db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4`
- block reward: `50 dIDR`
- target block time: `30s`
- retarget window: `30`
- min difficulty: `1`
- max difficulty: `12`
- coinbase maturity: `100`
- min stake amount: `100 dIDR`
- min service stake: `1000 dIDR`
- unbonding period: `100`

Seed peers are startup hints only. Peers still validate network ID, chain ID, genesis hash, and protocol compatibility.

## Verify Locally

Using a release binary:

```sh
./indochain --datadir ./data/testnet --network testnet init
./indochain --datadir ./data/testnet chain info
./indochain --datadir ./data/testnet chain validate
```

Using the source tree:

```sh
go run ./node/cmd/indochain --datadir ./data/testnet --network testnet init
go run ./node/cmd/indochain --datadir ./data/testnet chain info
go run ./node/cmd/indochain --datadir ./data/testnet chain validate
```

Expected:

- `network: testnet`
- `chain id: 777101`
- `tip hash: db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4`
- `chain valid`
- `mainnet` remains unavailable

